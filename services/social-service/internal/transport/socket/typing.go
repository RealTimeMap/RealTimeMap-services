package chatsocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/zishang520/socket.io/servers/socket/v3"
	"go.uber.org/zap"
)

// Typing-события. В отличие от message.new, состояние набора текста нигде не
// сохраняется: это эфемерный сигнал, живущий только внутри соединения.
const (
	// ClientEventTypingStart — клиент→сервер: пользователь начал набирать текст.
	// Payload: {"chatId": <uint>}. Клиенту следует слать его с throttle (~2с),
	// а не на каждое нажатие клавиши.
	ClientEventTypingStart = "typing.start"
	// ClientEventTypingStop — клиент→сервер: пользователь перестал набирать
	// (отправил сообщение, очистил поле, ушёл из чата). Payload: {"chatId": <uint>}.
	ClientEventTypingStop = "typing.stop"

	// EventChatTyping — сервер→клиент: кто-то в чате начал или перестал набирать.
	// Payload: TypingPayload. Одно событие с флагом isTyping вместо двух разных —
	// клиенту проще: один обработчик, который включает и выключает индикатор.
	EventChatTyping = "chat.typing"
)

// typingTTL — через сколько сервер сам гасит индикатор, если клиент не прислал
// typing.stop. Закрывает случай «пользователь начал печатать и закрыл ноутбук»:
// без автосброса «Печатает...» висел бы у собеседника вечно.
//
// Значение чуть больше рекомендованного клиентского throttle (~2с), чтобы
// непрерывный набор не мигал индикатором между повторными typing.start.
const typingTTL = 6 * time.Second

// TypingPayload — тело события chat.typing.
type TypingPayload struct {
	ChatID   uint   `json:"chatId"`
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	// IsTyping — включить (true) или погасить (false) индикатор. Значение false
	// приходит и по typing.stop от клиента, и по серверному автосбросу.
	IsTyping bool `json:"isTyping"`
}

// typingTracker гасит индикаторы набора для одного сокета по таймауту.
//
// На сокет — свой трекер: соединение уходит, вместе с ним снимаются все его
// таймеры. Ключ — chatID, потому что пользователь может печатать в нескольких
// чатах одновременно и у каждого свой независимый таймаут.
type typingTracker struct {
	mu     sync.Mutex
	timers map[uint]*time.Timer
	// stopped защищает от «выстрела после disconnect»: сокет уже закрыт, а
	// сработавший таймер попытался бы отправить в него автосброс.
	stopped bool
}

func newTypingTracker() *typingTracker {
	return &typingTracker{timers: make(map[uint]*time.Timer)}
}

// arm ставит (или сдвигает) таймер автосброса для чата. onExpire вызывается, если
// в течение typingTTL не пришёл ни новый typing.start, ни typing.stop.
func (t *typingTracker) arm(chatID uint, onExpire func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}

	// Повторный typing.start при непрерывном наборе: сдвигаем существующий
	// таймер, а не плодим новый.
	if timer, ok := t.timers[chatID]; ok {
		timer.Reset(typingTTL)
		return
	}

	t.timers[chatID] = time.AfterFunc(typingTTL, func() {
		// Снимаем таймер из карты до вызова колбэка: к этому моменту он уже
		// отработал и хранить его незачем. Если сокет успел закрыться — молча
		// выходим, слать автосброс некуда.
		t.mu.Lock()
		if t.stopped {
			t.mu.Unlock()
			return
		}
		delete(t.timers, chatID)
		t.mu.Unlock()

		onExpire()
	})
}

// disarm снимает таймер автосброса — набор завершился явным typing.stop, гасить
// повторно нечего. Возвращает false, если таймера не было: значит индикатор уже
// погашен и ретранслировать stop не нужно (защита от спама stop-ами от клиента).
func (t *typingTracker) disarm(chatID uint) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	timer, ok := t.timers[chatID]
	if !ok {
		return false
	}
	timer.Stop()
	delete(t.timers, chatID)
	return true
}

// stopAll снимает все таймеры сокета. Вызывается на disconnect.
func (t *typingTracker) stopAll() []uint {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.stopped = true
	chatIDs := make([]uint, 0, len(t.timers))
	for chatID, timer := range t.timers {
		timer.Stop()
		chatIDs = append(chatIDs, chatID)
	}
	t.timers = make(map[uint]*time.Timer)
	return chatIDs
}

// bindTypingHandlers вешает на сокет приём typing.start/typing.stop от клиента.
//
// Возвращает трекер, чтобы обработчик disconnect мог снять таймеры и погасить
// зависшие индикаторы.
func (s *SocketServer) bindTypingHandlers(sock *socket.Socket, data socketData) *typingTracker {
	tracker := newTypingTracker()

	sock.On(ClientEventTypingStart, func(args ...any) {
		chatID, ok := s.typingChatID(sock, data, ClientEventTypingStart, args)
		if !ok {
			return
		}

		tracker.arm(chatID, func() {
			// Клиент замолчал, не прислав stop, — гасим индикатор сами, иначе он
			// останется висеть у собеседников.
			s.emitTyping(sock, data, chatID, false)
		})
		s.emitTyping(sock, data, chatID, true)
	})

	sock.On(ClientEventTypingStop, func(args ...any) {
		chatID, ok := s.typingChatID(sock, data, ClientEventTypingStop, args)
		if !ok {
			return
		}

		// Индикатор уже погашен (автосбросом или предыдущим stop) — второй раз
		// рассылать нечего.
		if !tracker.disarm(chatID) {
			return
		}
		s.emitTyping(sock, data, chatID, false)
	})

	return tracker
}

// typingChatID достаёт chatId из payload клиента и проверяет право слать typing в
// этот чат.
//
// Авторизация — по членству сокета в комнате chat:<id>. Комнаты заполняются на
// подключении из активных участий в БД (eager-join) и синхронизируются при
// создании/выходе, поэтому членство в комнате = участие в чате. Это защищает от
// клиента, который подставит чужой chatId, и не стоит запроса в БД на каждое
// нажатие клавиши.
func (s *SocketServer) typingChatID(sock *socket.Socket, data socketData, event string, args []any) (uint, bool) {
	chatID, ok := parseChatID(args)
	if !ok {
		s.logger.Debug("typing event with invalid payload",
			zap.String("event", event), zap.Uint("user_id", data.userID))
		return 0, false
	}

	if !sock.Rooms().Has(ChatRoom(chatID)) {
		s.logger.Warn("typing event for chat the socket is not in",
			zap.String("event", event),
			zap.Uint("user_id", data.userID),
			zap.Uint("chat_id", chatID))
		return 0, false
	}

	return chatID, true
}

// emitTyping рассылает chat.typing остальным участникам чата.
//
// sock.To(room) исключает сам сокет-источник: печатающему его собственный
// индикатор не нужен. Другие устройства того же пользователя событие получат —
// это безвредно, клиент фильтрует по своему userId.
func (s *SocketServer) emitTyping(sock *socket.Socket, data socketData, chatID uint, isTyping bool) {
	payload := TypingPayload{
		ChatID:   chatID,
		UserID:   data.userID,
		Username: data.userName,
		IsTyping: isTyping,
	}

	if err := sock.To(ChatRoom(chatID)).Emit(EventChatTyping, payload); err != nil {
		s.logger.Warn("failed to emit chat.typing",
			zap.Error(err),
			zap.Uint("chat_id", chatID),
			zap.Uint("user_id", data.userID),
			zap.Bool("is_typing", isTyping))
	}
}

// parseChatID достаёт chatId из первого аргумента socket-события.
//
// Socket.IO отдаёт JSON-payload как map[string]any с числами в float64, но клиент
// может прислать и голое число, и строку — принимаем все три формы, чтобы
// контракт не ломался о мелочи сериализации на фронте.
func parseChatID(args []any) (uint, bool) {
	if len(args) == 0 {
		return 0, false
	}

	switch v := args[0].(type) {
	case map[string]any:
		raw, ok := v["chatId"]
		if !ok {
			return 0, false
		}
		return toChatID(raw)
	default:
		return toChatID(args[0])
	}
}

// toChatID приводит значение к валидному id чата. Ноль и отрицательные значения
// отбраковываются: chat:0 — несуществующая комната.
func toChatID(raw any) (uint, bool) {
	switch v := raw.(type) {
	case float64:
		if v <= 0 {
			return 0, false
		}
		return uint(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil || n <= 0 {
			return 0, false
		}
		return uint(n), true
	case string:
		var n json.Number = json.Number(v)
		parsed, err := n.Int64()
		if err != nil || parsed <= 0 {
			return 0, false
		}
		return uint(parsed), true
	}
	return 0, false
}
