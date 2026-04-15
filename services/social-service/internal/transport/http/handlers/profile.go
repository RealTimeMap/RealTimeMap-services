package handlers

import (
	"net/http"

	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	profileservice "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProfileDeps struct {
	Service *profileservice.Service

	Logger *zap.Logger
}
type ProfileHandler struct {
	service *profileservice.Service

	logger *zap.Logger
}

func RegisterProfileHandler(g *gin.RouterGroup, deps ProfileDeps) {
	handler := &ProfileHandler{
		service: deps.Service,
		logger:  deps.Logger,
	}

	profileGroup := g.Group("/profile")
	{
		profileGroup.GET("/me", auth.AuthRequired(), handler.GetMyProfile)
		profileGroup.GET("/search", handler.SearchProfile)
	}
}

func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	uProfile, err := h.service.GetProfile(c.Request.Context(), uint(userInfo.UserID))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewPersonalProfileResponse(uProfile))
}

func (h *ProfileHandler) SearchProfile(c *gin.Context) {
	var req dto.SearchProfileRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	input := &profileservice.SearchProfilesInput{
		Username: req.Query,
		Pagination: pagination.Params{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	}

	profiles, total, err := h.service.SearchProfiles(c.Request.Context(), input)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	items := dto.NewSearchProfileItems(profiles)
	c.JSON(http.StatusOK, pagination.NewResponse(items, input.Pagination, total))
}
