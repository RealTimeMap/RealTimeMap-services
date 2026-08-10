package app

import (
	pkgprogress "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/progress"
	pkgmark "github.com/RealTimeMap/RealTimeMap-backend/pkg/clients/stats/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/database/txmanager"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/mediavalidator"
	pkgredis "github.com/RealTimeMap/RealTimeMap-backend/pkg/redis"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/storage"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app/use_cases/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/config"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/chat/services"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/repository"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/blockeduser"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/friendship"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	progressadapter "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/grpc/progress"
	markstatadapter "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/grpc/stats"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/persistence/postgres"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/realtime/presence"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/infrastructure/realtime/socketpub"
	profilegrpc "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/grpc/profile"
	chatsocket "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/socket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	ProfileRepo        repository.ProfileRepository
	ProfileService     *profile.Service
	ProfileStatService *profile.StatService
	ProfileGRPCHandler *profilegrpc.Handler

	BlockedUserRepo    repository.BlockedUserRepository
	BlockedUserService *blockeduser.Service

	FriendshipRepo    repository.FriendShipRepository
	FriendshipService *friendship.Service

	Storage storage.Storage

	ChatCases  *chat.Application
	ChatSocket *chatsocket.SocketServer

	ProgressClient *pkgprogress.Client
	MarkStatClient *pkgmark.Client

	Redis *redis.Client

	Logger *zap.Logger
	DB     *gorm.DB
}

func (c *Container) Close() error {
	// Закрываем Socket.IO: рвём соединения и подписки Redis adapter.
	if c.ChatSocket != nil {
		c.ChatSocket.Close()
	}

	if c.ProgressClient != nil {
		return c.ProgressClient.Close()
	}

	return nil
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {
	store, err := storage.NewLocalStorage(cfg.Storage.BasePath, cfg.Storage.BaseURL, logger)
	if err != nil {
		panic(err)
	}
	photoValidator := mediavalidator.NewPhotoValidator()

	var (
		progressClient *pkgprogress.Client
		progressPort   profile.ProgressGetter
		markStatClient *pkgmark.Client
		markStatPort   profile.MarkStatGetter
	)
	if cfg.Gamification.Address != "" {
		c, err := pkgprogress.NewClient(&cfg.Gamification)
		if err != nil {
			logger.Warn("gamification client init failed, continuing without progress",
				zap.Error(err))
		} else {
			progressClient = c
			progressPort = progressadapter.NewAdapter(c)
		}
	}
	if cfg.MarkStat.Address != "" {
		c, err := pkgmark.NewClient(cfg.MarkStat)
		if err != nil {
			logger.Warn("mark_action client init failed, continuing without progress", zap.Error(err))
		} else {
			markStatClient = c
			markStatPort = markstatadapter.NewAdapter(markStatClient)
		}
	}

	profileRepo := postgres.NewPgProfileRepository(db, logger)
	profileService := profile.NewProfileService(profileRepo, store, photoValidator, progressPort, logger)
	friendRepo := postgres.NewPgFriendshipRepository(db, logger)
	profileStatService := profile.NewStatService(markStatPort, friendRepo, logger)
	profileHandler := profilegrpc.NewHandler(profileService, logger)

	redisCli := getRedisCli(cfg.Redis)

	blockedUserRepo := postgres.NewPgBlockedUserRepository(db, logger)
	blockedUserService := blockeduser.NewService(blockedUserRepo, profileRepo, logger)

	friendshipService := friendship.NewService(friendRepo, profileRepo, blockedUserRepo, logger)

	// V2

	txm := txmanager.NewTxManager(db)

	chatRepoV2 := postgres.NewChatRepository(db, logger)
	chatParticipantRepoV2 := postgres.NewChatParticipantRepository(db, logger)
	messageRepoV2 := postgres.NewMessageRepository(db, logger)
	chatServiceV2 := services.NewChatService(chatRepoV2, chatParticipantRepoV2, blockedUserRepo, &txm, logger)
	messageServiceV2 := services.NewMessageService(messageRepoV2, chatRepoV2, chatParticipantRepoV2, blockedUserRepo, &txm, logger)

	// Socket.IO для realtime-событий чата. Redis adapter внутри обеспечивает
	// доставку между инстансами. Publisher — реализация порта chat.EventPublisher.
	// Presence в том же Redis: онлайн-статус общий для всех инстансов, иначе
	// пользователь, подключённый к соседнему инстансу, числился бы офлайн.
	chatSocket := chatsocket.New(chatsocket.Deps{
		Redis:          redisCli,
		AllowedOrigins: cfg.Http.AllowOrigins,
		ChatLister:     chatParticipantRepoV2,
		Presence:       presence.NewStore(redisCli),
		Logger:         logger,
	})
	chatEventPublisher := socketpub.NewPublisher(chatSocket.Namespace(), chatSocket, logger)

	chatCases := &chat.Application{
		Direct:      chat.NewDirectChatHandler(chatServiceV2, profileService, chatEventPublisher, logger),
		Group:       chat.NewGroupChatHandler(chatServiceV2, chatEventPublisher, logger),
		SendMessage: chat.NewMessageSenderHandler(messageServiceV2, profileService, chatEventPublisher, logger),
		History:     chat.NewChatHistoryHandler(messageServiceV2, profileService, logger),
		ListChats:   chat.NewListUserChatsHandler(chatServiceV2, profileService, logger),
		MarkRead:    chat.NewMarkReadHandler(chatServiceV2, chatEventPublisher, logger),
		Leave:       chat.NewLeaveHandler(chatServiceV2, chatEventPublisher, logger),
	}

	return &Container{
		ProfileRepo:        profileRepo,
		ProfileService:     profileService,
		ProfileStatService: profileStatService,
		ProfileGRPCHandler: profileHandler,

		BlockedUserRepo:    blockedUserRepo,
		BlockedUserService: blockedUserService,

		FriendshipRepo:    friendRepo,
		FriendshipService: friendshipService,

		Storage: store,

		ProgressClient: progressClient,

		ChatCases:  chatCases,
		ChatSocket: chatSocket,

		Redis:  redisCli,
		Logger: logger,
		DB:     db,
	}
}

func getRedisCli(cfg pkgredis.Config) *redis.Client {
	return pkgredis.NewRedisCli(cfg)
}
