package chatsocket

import (
	"context"
	"strconv"

	"github.com/zishang520/socket.io/servers/socket/v3"
	"go.uber.org/zap"
)

// ChatLister — узкий порт, дающий namespace список чатов пользователя для
// eager-join в комнаты chat:<id> при подключении и список собеседников для
// presence-снапшота. Реализуется поверх репозитория участников; namespace не
// знает про БД/GORM.
type ChatLister interface {
	ChatIDsByUser(ctx context.Context, userID uint) ([]uint, error)
	// PeerIDsByUser возвращает id пользователей, с которыми userID состоит хотя
	// бы в одном активном чате — кандидатов на presence.snapshot.
	PeerIDsByUser(ctx context.Context, userID uint) ([]uint, error)
}

// Заголовки идентичности, которые Traefik forwardAuth вклеивает в handshake-запрос
// после валидации Bearer-токена (см. traefik/dynamic.yml → middleware auth-check).
const (
	headerUserID   = "X-User-Id"
	headerUserName = "X-User-Name"
)

// socketData — данные, привязанные к сокету после аутентификации.
type socketData struct {
	userID   uint
	userName string
}

// UserRoom возвращает имя комнаты пользователя. Каждый сокет джойнит комнату
// своего userID, а доставка события пользователю = Namespace().To(UserRoom(id)).Emit(...).
// Единая точка формирования имени: используется и здесь (Join), и в publisher (Emit).
func UserRoom(userID uint) socket.Room {
	return socket.Room("user:" + strconv.FormatUint(uint64(userID), 10))
}

// ChatRoom возвращает имя комнаты чата. Сокет джойнит комнату каждого своего чата
// при подключении — членство в комнате chat:<id> = участие в чате. На этом
// строится адресация событий чату (typing/read/edited/deleted/reactions) без
// проверки участия в БД. Единая точка формирования имени: Join (namespace),
// SocketsJoin/Leave (room-sync) и Emit (publisher).
func ChatRoom(chatID uint) socket.Room {
	return socket.Room("chat:" + strconv.FormatUint(uint64(chatID), 10))
}

// InitChatNamespace вешает на namespace /chats аутентификацию и обработчик
// подключения.
//
// Модель доверия: тот же forwardAuth, что и для HTTP. Traefik валидирует
// Authorization: Bearer через auth-service и вклеивает X-User-Id/X-User-Name в
// handshake-запрос. zishang520 пробрасывает эти заголовки в Handshake().Headers,
// откуда мы и берём identity. Клиент шлёт только токен:
// io("<host>", { path: "/socket.io/", extraHeaders: { Authorization: "Bearer <t>" } }).
// userId клиентом не передаётся и не может быть подделан.
func InitChatNamespace(s *SocketServer) {
	ns := s.ns
	s.logger.Info("init chat socket namespace", zap.String("namespace", namespaceName))

	// Use выполняется на этапе хендшейка: без валидного identity соединение
	// отклоняется и обработчик connection не вызывается.
	ns.Use(func(sock *socket.Socket, next func(*socket.ExtendedError)) {
		headers := sock.Handshake().Headers.Header()

		userID, err := strconv.ParseUint(headers.Get(headerUserID), 10, 64)
		if err != nil || userID == 0 {
			s.logger.Warn("socket auth failed: missing/invalid X-User-Id",
				zap.String("raw", headers.Get(headerUserID)))
			next(socket.NewExtendedError("unauthorized", nil))
			return
		}

		sock.SetData(socketData{
			userID:   uint(userID),
			userName: headers.Get(headerUserName),
		})
		next(nil)
	})

	ns.On("connection", func(clients ...any) {
		sock, ok := clients[0].(*socket.Socket)
		if !ok {
			return
		}

		data, ok := sock.Data().(socketData)
		if !ok {
			s.logger.Warn("socket connected without auth data, disconnecting",
				zap.String("socket_id", string(sock.Id())))
			sock.Disconnect(true)
			return
		}

		// Сокет джойнит комнату своего пользователя — сюда publisher шлёт события
		// адресно (message.new, presence-эхо).
		sock.Join(UserRoom(data.userID))

		// Eager-join во все чаты пользователя: членство в комнате chat:<id> =
		// участие в чате. Best-effort — если список не загрузился, соединение
		// остаётся, но событий чата сокет не получит до реконнекта; сообщения
		// всё равно доберутся через историю (pull-gap).
		chatIDs := s.joinUserChats(sock, data.userID)

		// Приём typing.start/typing.stop от клиента. Вешаем до presence, чтобы
		// сокет был готов принимать события сразу после подключения.
		typing := s.bindTypingHandlers(sock, data)

		// Регистрируем присутствие и рассылаем presence.online, если это первое
		// соединение пользователя. chatIDs передаём готовым — второй раз ходить
		// в БД за тем же списком незачем.
		s.handlePresenceConnect(sock, data.userID, chatIDs)

		// Держим presence-ключ живым, пока держится соединение.
		refreshDone := make(chan struct{})
		s.startPresenceRefresh(data.userID, refreshDone)

		s.logger.Info("socket connected",
			zap.Uint("user_id", data.userID),
			zap.String("socket_id", string(sock.Id())))

		sock.On("disconnect", func(reason ...any) {
			close(refreshDone)

			// Гасим индикаторы набора, оставшиеся от этого соединения: у
			// собеседников не должно висеть «Печатает...» от того, кто отвалился.
			s.stopTypingOnDisconnect(sock, data, typing)

			s.handlePresenceDisconnect(sock, data.userID, chatIDs)

			s.logger.Info("socket disconnected",
				zap.Uint("user_id", data.userID),
				zap.String("socket_id", string(sock.Id())))
		})
	})
}

// joinUserChats джойнит сокет во все комнаты chat:<id> пользователя и возвращает
// список этих чатов — он же используется как адресация presence-событий.
// Вызывается при подключении. Ошибка загрузки списка не рвёт соединение — сокет
// остаётся в user:<id>, пропущенное добирается через историю (pull-gap).
func (s *SocketServer) joinUserChats(sock *socket.Socket, userID uint) []uint {
	if s.chatLister == nil {
		return nil
	}
	chatIDs, err := s.chatLister.ChatIDsByUser(context.Background(), userID)
	if err != nil {
		s.logger.Warn("failed to load user chats for eager-join",
			zap.Error(err), zap.Uint("user_id", userID))
		return nil
	}
	for _, id := range chatIDs {
		sock.Join(ChatRoom(id))
	}
	return chatIDs
}

// stopTypingOnDisconnect снимает таймеры набора отвалившегося сокета и гасит
// индикаторы в тех чатах, где они успели зажечься.
//
// Рассылаем до того, как обработчик disconnect отработает до конца: пока сокет
// ещё числится в комнатах, sock.To(room) доставит событие остальным участникам.
func (s *SocketServer) stopTypingOnDisconnect(sock *socket.Socket, data socketData, tracker *typingTracker) {
	for _, chatID := range tracker.stopAll() {
		s.emitTyping(sock, data, chatID, false)
	}
}
