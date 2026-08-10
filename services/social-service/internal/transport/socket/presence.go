package chatsocket

import (
	"context"
	"time"

	"github.com/zishang520/socket.io/servers/socket/v3"
	"go.uber.org/zap"
)

// Presence-события. Онлайн-статус — производная от наличия socket-соединения,
// поэтому он целиком живёт в транспортном слое: useCase про него не знает и в БД
// он не пишется.
const (
	// EventPresenceOnline — пользователь появился в сети (первое соединение).
	// Payload: PresencePayload.
	EventPresenceOnline = "presence.online"
	// EventPresenceOffline — пользователь ушёл из сети (закрылось последнее
	// соединение). Payload: PresencePayload.
	EventPresenceOffline = "presence.offline"
	// EventPresenceSnapshot — единоразовый список онлайн-собеседников, который
	// сервер шлёт только что подключившемуся сокету. Payload: SnapshotPayload.
	//
	// Зачем: presence.online/offline описывают только будущие переходы. Клиент,
	// открывший страницу, без снапшота считал бы офлайн всех, кто вошёл раньше,
	// пока те не переподключатся.
	EventPresenceSnapshot = "presence.snapshot"
)

// presenceRefreshInterval — как часто продлевать TTL ключа соединений в Redis.
// Заметно меньше самого TTL, чтобы пропуск одного тика не выкинул живого
// пользователя из онлайна.
const presenceRefreshInterval = 2 * time.Minute

// PresencePayload — тело событий presence.online/presence.offline.
type PresencePayload struct {
	UserID uint   `json:"userId"`
	Online bool   `json:"online"`
	At     string `json:"at"`
}

// SnapshotPayload — тело события presence.snapshot: id собеседников, которые
// сейчас онлайн. Всех остальных клиент считает офлайн.
type SnapshotPayload struct {
	Online []uint `json:"online"`
}

// PresenceStore — порт хранилища онлайн-статусов. Реализуется поверх Redis
// (infrastructure/realtime/presence); socket-слой не знает про его устройство.
//
// Connect/Disconnect возвращают факт перехода состояния, а не просто «записал»:
// у пользователя может быть несколько вкладок и устройств, а presence-событие
// должно уйти только на первое подключение и последнее отключение.
type PresenceStore interface {
	Connect(ctx context.Context, userID uint, socketID string) (becameOnline bool, err error)
	Disconnect(ctx context.Context, userID uint, socketID string) (becameOffline bool, err error)
	Refresh(ctx context.Context, userID uint) error
	OnlineAmong(ctx context.Context, userIDs []uint) ([]uint, error)
}

// handlePresenceConnect регистрирует соединение и, если пользователь только что
// появился в сети, рассылает presence.online его собеседникам. Затем отдаёт
// подключившемуся снапшот текущих онлайн-собеседников.
//
// Best-effort целиком: presence — украшение UI. Любая ошибка Redis логируется, но
// соединение остаётся рабочим — чат должен работать и без индикатора онлайна.
func (s *SocketServer) handlePresenceConnect(sock *socket.Socket, userID uint, chatIDs []uint) {
	if s.presence == nil {
		return
	}
	ctx := context.Background()

	becameOnline, err := s.presence.Connect(ctx, userID, string(sock.Id()))
	if err != nil {
		s.logger.Warn("presence connect failed",
			zap.Error(err), zap.Uint("user_id", userID))
		return
	}

	// Второе устройство/вкладка того же пользователя: он и так уже числится
	// онлайн, повторное событие только заставило бы клиентов перерисовываться.
	if becameOnline {
		s.broadcastPresence(userID, chatIDs, true)
	}

	s.sendPresenceSnapshot(ctx, sock, userID)
}

// handlePresenceDisconnect снимает регистрацию соединения и рассылает
// presence.offline, если закрылось последнее соединение пользователя.
//
// chatIDs — список, посчитанный при подключении. Комнаты сокета к моменту
// disconnect уже очищены библиотекой, поэтому адресовать рассылку по
// sock.Rooms() здесь нельзя — используем сохранённый срез.
func (s *SocketServer) handlePresenceDisconnect(sock *socket.Socket, userID uint, chatIDs []uint) {
	if s.presence == nil {
		return
	}

	becameOffline, err := s.presence.Disconnect(context.Background(), userID, string(sock.Id()))
	if err != nil {
		s.logger.Warn("presence disconnect failed",
			zap.Error(err), zap.Uint("user_id", userID))
		return
	}
	if !becameOffline {
		return
	}

	s.broadcastPresence(userID, chatIDs, false)
}

// AnnounceInChat рассылает в комнату нового чата актуальный онлайн-статус
// перечисленных участников.
//
// Нужно потому, что presence.online рассылается по чатам, существовавшим на
// момент подключения. Когда чат создаётся позже, его участники уже сидят онлайн,
// и без этого доброса они увидели бы статусы друг друга только после реконнекта.
//
// Вызывается из publisher после JoinUsers. Best-effort: ошибка только логируется.
func (s *SocketServer) AnnounceInChat(ctx context.Context, chatID uint, userIDs []uint) {
	if s.presence == nil || chatID == 0 || len(userIDs) == 0 {
		return
	}

	online, err := s.presence.OnlineAmong(ctx, userIDs)
	if err != nil {
		s.logger.Warn("failed to resolve online users for new chat",
			zap.Error(err), zap.Uint("chat_id", chatID))
		return
	}

	// Офлайн-участников не анонсируем: клиент по умолчанию считает всех офлайн,
	// лишнее событие ничего не изменит.
	now := time.Now().UTC().Format(time.RFC3339)
	for _, uid := range online {
		payload := PresencePayload{UserID: uid, Online: true, At: now}
		if err := s.ns.To(ChatRoom(chatID)).Emit(EventPresenceOnline, payload); err != nil {
			s.logger.Warn("failed to announce presence in new chat",
				zap.Error(err), zap.Uint("chat_id", chatID), zap.Uint("user_id", uid))
		}
	}
}

// broadcastPresence рассылает presence-событие в комнаты всех чатов пользователя:
// статус видят ровно те, кому он нужен для UI (список чатов и открытый диалог).
//
// Шлём через namespace, а не через sock.To(...): при disconnect сокет уже покинул
// комнаты, а сам пользователь как получатель здесь не нужен — свой статус клиент
// и так знает.
func (s *SocketServer) broadcastPresence(userID uint, chatIDs []uint, online bool) {
	if len(chatIDs) == 0 {
		return
	}

	payload := PresencePayload{
		UserID: userID,
		Online: online,
		At:     time.Now().UTC().Format(time.RFC3339),
	}
	event := EventPresenceOffline
	if online {
		event = EventPresenceOnline
	}

	rooms := make([]socket.Room, 0, len(chatIDs))
	for _, id := range chatIDs {
		rooms = append(rooms, ChatRoom(id))
	}

	// Один Emit на все комнаты: socket.io дедуплицирует получателей, состоящих
	// сразу в нескольких из них (несколько общих чатов с одним собеседником).
	if err := s.ns.To(rooms...).Emit(event, payload); err != nil {
		s.logger.Warn("failed to emit presence event",
			zap.Error(err), zap.String("event", event), zap.Uint("user_id", userID))
	}
}

// sendPresenceSnapshot отдаёт подключившемуся сокету список его собеседников,
// которые сейчас онлайн. Адресуется лично сокету (sock.Emit), а не пользователю:
// снапшот нужен именно новому соединению, остальные вкладки своё состояние уже
// накопили.
func (s *SocketServer) sendPresenceSnapshot(ctx context.Context, sock *socket.Socket, userID uint) {
	if s.chatLister == nil {
		return
	}

	peerIDs, err := s.chatLister.PeerIDsByUser(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to load peers for presence snapshot",
			zap.Error(err), zap.Uint("user_id", userID))
		return
	}

	online, err := s.presence.OnlineAmong(ctx, peerIDs)
	if err != nil {
		s.logger.Warn("failed to resolve online peers",
			zap.Error(err), zap.Uint("user_id", userID))
		return
	}

	// Пустой срез, а не nil: в JSON уйдёт [], а не null — клиенту не нужен
	// отдельный случай для «нет онлайн-собеседников».
	if online == nil {
		online = []uint{}
	}

	if err := sock.Emit(EventPresenceSnapshot, SnapshotPayload{Online: online}); err != nil {
		s.logger.Warn("failed to emit presence snapshot",
			zap.Error(err), zap.Uint("user_id", userID))
	}
}

// startPresenceRefresh продлевает TTL presence-ключа, пока сокет жив. Без этого
// долгое соединение без реконнекта протухло бы по TTL и пользователь пропал бы из
// онлайна, фактически оставаясь подключённым.
//
// Горутина завершается по закрытию done — его закрывает обработчик disconnect.
func (s *SocketServer) startPresenceRefresh(userID uint, done <-chan struct{}) {
	if s.presence == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(presenceRefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := s.presence.Refresh(context.Background(), userID); err != nil {
					s.logger.Warn("failed to refresh presence ttl",
						zap.Error(err), zap.Uint("user_id", userID))
				}
			}
		}
	}()
}
