package handlers

import (
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/app/use_cases/chat"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	useCase *chat.Application

	logger *zap.Logger
}

type ChatHandlerDeps struct {
	UseCase *chat.Application

	Logger *zap.Logger
}

func InitChatHandler(g *gin.RouterGroup, deps ChatHandlerDeps) {
	h := &Handler{
		useCase: deps.UseCase,
		logger:  deps.Logger,
	}

	chatGroup := g.Group("/chats")
	{
		chatGroup.GET("", auth.AuthRequired(), h.GetUserChats)
		chatGroup.POST("/direct", auth.AuthRequired(), h.GetDirectChat)
		chatGroup.POST("/group", auth.AuthRequired(), h.CreateGroupChat)
		chatGroup.POST("/:chatID/messages", auth.AuthRequired(), h.SendMessage)
		chatGroup.GET("/:chatID/history", auth.AuthRequired(), h.GetChatHistory)
		chatGroup.POST("/:chatID/read", auth.AuthRequired(), h.MarkChatRead)
		chatGroup.POST("/:chatID/leave", auth.AuthRequired(), h.LeaveChat)
	}
}

type directChatRequest struct {
	PeerID uint `json:"peerId"`
}

func (h *Handler) GetDirectChat(c *gin.Context) {
	var req directChatRequest
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	if err = c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	result, err := h.useCase.Direct.Handle(c.Request.Context(), chat.CreateDirectCommand{
		UserID: uint(userID),
		PeerID: req.PeerID,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewDirectChatResponse(result))

}

type groupChatRequest struct {
	PeersIds []uint  `json:"peersIds"`
	Title    *string `json:"title"`
}

func (h *Handler) CreateGroupChat(c *gin.Context) {
	var req groupChatRequest
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	if err = c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}
	result, err := h.useCase.Group.Handle(c.Request.Context(), chat.GroupChatCreateCommand{
		InitiatorID: uint(userID),
		PeersIds:    req.PeersIds,
		Title:       req.Title,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewGroupChatResponse(result))
}

type MessageRequest struct {
	Content string `json:"content"`
	// ClientMessageID — UUID, сгенерированный клиентом для идемпотентной отправки.
	// Повтор с тем же значением не создаёт дубль, а возвращает исходное сообщение.
	ClientMessageID string `json:"clientMessageId"`
}

func (h *Handler) SendMessage(c *gin.Context) {
	var req MessageRequest
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	chatID, err := middleware.ParsePathParams(c, "chatID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	if err = c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.useCase.SendMessage.Handle(c.Request.Context(), chat.MessageCreateCommand{
		Content:         req.Content,
		ChatID:          chatID,
		SenderID:        uint(userID),
		ClientMessageID: req.ClientMessageID,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewMessageResponse(res))

}

type GetMessageRequest struct {
	LastMessageID *uint `form:"lastMessageId"`
}

func (h *Handler) GetChatHistory(c *gin.Context) {
	var req GetMessageRequest
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	chatID, err := middleware.ParsePathParams(c, "chatID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	if err = c.ShouldBindQuery(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	res, err := h.useCase.History.Handle(c.Request.Context(), chat.GetMessageCommand{
		ChatID:        chatID,
		UserID:        uint(userID),
		LastMessageID: req.LastMessageID,
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewMessageHistoryResponse(res))
}

func (h *Handler) GetUserChats(c *gin.Context) {
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	res, err := h.useCase.ListChats.Handle(c.Request.Context(), chat.ListUserChatsCommand{
		UserID: uint(userID),
	})
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewChatListResponse(res))
}

// MarkChatRead отмечает чат прочитанным до последнего сообщения для текущего
// пользователя. Тело не требуется — сервер двигает курсор на последнее сообщение чата.
func (h *Handler) MarkChatRead(c *gin.Context) {
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	chatID, err := middleware.ParsePathParams(c, "chatID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	if err = h.useCase.MarkRead.Handle(c.Request.Context(), chat.MarkReadCommand{
		ChatID: chatID,
		UserID: uint(userID),
	}); err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.Status(http.StatusNoContent)
}

// LeaveChat выводит текущего пользователя из группового чата. Из direct-чата выйти
// нельзя (409). После выхода сокеты пользователя покидают комнату чата.
func (h *Handler) LeaveChat(c *gin.Context) {
	userID, err := helper.GetUserID(c)
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}
	chatID, err := middleware.ParsePathParams(c, "chatID")
	if err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	if err = h.useCase.Leave.Handle(c.Request.Context(), chat.LeaveCommand{
		ChatID: chatID,
		UserID: uint(userID),
	}); err != nil {
		middleware.HandleError(c, err, h.logger)
		return
	}

	c.Status(http.StatusNoContent)
}
