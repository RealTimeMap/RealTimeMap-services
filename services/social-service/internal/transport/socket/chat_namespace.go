package chatsocket

import (
	"strconv"

	"github.com/zishang520/socket.io/servers/socket/v3"
	"go.uber.org/zap"
)

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

		// Сокет джойнит комнату своего пользователя — сюда publisher шлёт события.
		sock.Join(UserRoom(data.userID))
		s.logger.Info("socket connected",
			zap.Uint("user_id", data.userID),
			zap.String("socket_id", string(sock.Id())))

		sock.On("disconnect", func(reason ...any) {
			s.logger.Info("socket disconnected",
				zap.Uint("user_id", data.userID),
				zap.String("socket_id", string(sock.Id())))
		})
	})
}
