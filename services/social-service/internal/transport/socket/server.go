// Package chatsocket поднимает Socket.IO-сервер для realtime-событий чата.
// Транспорт вынесен из useCase: доставку событий useCase инициирует через порт
// EventPublisher, а этот пакет отвечает за соединения, комнаты и доставку.
package chatsocket

import (
	"context"
	"net/http"
	"strings"

	rds "github.com/redis/go-redis/v9"
	redisclient "github.com/zishang520/socket.io/adapters/redis/v3"
	redisadapter "github.com/zishang520/socket.io/adapters/redis/v3/adapter"
	"github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
	"go.uber.org/zap"
)

// redisKeyPrefix — префикс ключей Redis adapter, чтобы не пересекаться с другими
// сервисами, использующими тот же Redis.
const redisKeyPrefix = "social:chat"

// namespaceName — Socket.IO namespace для чата. Клиент подключается на него:
// io("<host>/chats", { auth: { userId, userName } }).
const namespaceName = "/chats"

type SocketServer struct {
	io         *socket.Server
	ns         socket.Namespace
	chatLister ChatLister
	logger     *zap.Logger
}

type Deps struct {
	// Redis — уже созданный go-redis клиент (container.Redis). Оборачивается в
	// RedisClient adapter-а для межинстансной синхронизации комнат и emit.
	Redis *rds.Client
	// AllowedOrigins — разрешённые Origin для WebSocket-хендшейка (из
	// http_server.allow_origins). Пустой список = разрешить любой origin
	// (удобно для локальной разработки и тест-клиента). Без этой настройки
	// engine.io режет cross-origin апгрейд с 403.
	AllowedOrigins []string
	// ChatLister даёт namespace список чатов пользователя для eager-join в
	// комнаты chat:<id> при подключении.
	ChatLister ChatLister
	Logger     *zap.Logger
}

// New создаёт Socket.IO-сервер с Redis adapter и инициализирует namespace /chats.
// Redis adapter обеспечивает доставку событий между несколькими инстансами сервиса:
// io.To(room).Emit на одном инстансе долетает до сокетов на всех остальных.
func New(deps Deps) *SocketServer {
	rc := redisclient.NewRedisClient(context.Background(), deps.Redis)
	rc.On("error", func(errs ...any) {
		deps.Logger.Warn("socket.io redis adapter error", zap.Any("error", errs))
	})

	adapterOpts := &redisadapter.RedisAdapterOptions{}
	adapterOpts.SetKey(redisKeyPrefix)

	opts := socket.DefaultServerOptions()
	opts.SetAdapter(&redisadapter.RedisAdapterBuilder{
		Redis: rc,
		Opts:  adapterOpts,
	})
	// CORS для engine.io: без этого WebSocket-апгрейд с чужого Origin отклоняется
	// с 403 (дефолтный gorilla CheckOrigin режет cross-origin).
	opts.SetCors(&types.Cors{
		Origin:      originChecker(deps.AllowedOrigins),
		Credentials: true,
	})

	io := socket.NewServer(nil, opts)

	server := &SocketServer{
		io:         io,
		ns:         io.Of(namespaceName, nil),
		chatLister: deps.ChatLister,
		logger:     deps.Logger,
	}

	InitChatNamespace(server)

	return server
}

// originChecker возвращает предикат разрешённого Origin для CORS. Пустой список
// пропускает любой origin (dev/тест-клиент). Иначе — сверка без учёта регистра.
func originChecker(allowed []string) func(string) bool {
	return func(origin string) bool {
		if len(allowed) == 0 {
			return true
		}
		for _, a := range allowed {
			if strings.EqualFold(origin, a) {
				return true
			}
		}
		return false
	}
}

// HttpHandler отдаёт http.Handler Socket.IO для монтирования в существующий
// HTTP-роутер (через gin.WrapH на /socket.io/*any).
func (s *SocketServer) HttpHandler() http.Handler {
	return s.io.ServeHandler(nil)
}

// Close завершает работу Socket.IO-сервера: закрывает клиентские соединения и
// освобождает ресурсы Redis adapter. Вызывается при остановке приложения.
func (s *SocketServer) Close() {
	s.io.Close(func(err error) {
		if err != nil {
			s.logger.Warn("error closing socket.io server", zap.Error(err))
		}
	})
}

// Namespace возвращает namespace /chats для реализации порта EventPublisher
// (socketpub), которому нужен ns.To(room).Emit. Emit шлётся именно в этот
// namespace — тот же, к которому подключаются клиенты.
func (s *SocketServer) Namespace() socket.Namespace {
	return s.ns
}
