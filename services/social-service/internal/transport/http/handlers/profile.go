package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"
	helper "github.com/RealTimeMap/RealTimeMap-backend/pkg/helpers/context"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/auth"
	errorhandler "github.com/RealTimeMap/RealTimeMap-backend/pkg/middleware/error"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/pagination"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/http/middleware"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/validation"
	profileservice "github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/profile"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/domain/service/subscription"
	"github.com/RealTimeMap/RealTimeMap-backend/services/social-service/internal/transport/http/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProfileDeps struct {
	Service *profileservice.Service

	SubscriptionService *subscription.Service

	Logger *zap.Logger
}
type ProfileHandler struct {
	service *profileservice.Service

	subscriptionService *subscription.Service

	logger *zap.Logger
}

func RegisterProfileHandler(g *gin.RouterGroup, deps ProfileDeps) {
	handler := &ProfileHandler{
		service:             deps.Service,
		subscriptionService: deps.SubscriptionService,
		logger:              deps.Logger,
	}

	profileGroup := g.Group("")
	{
		profileGroup.GET("/me", auth.AuthRequired(), handler.GetMyProfile)
		profileGroup.PATCH("/me", auth.AuthRequired(), handler.UpdateMyProfile)
		profileGroup.GET("/search", handler.SearchProfile)
		profileGroup.GET("/settings", auth.AuthRequired(), handler.GetSettings)
		profileGroup.PATCH("/settings", auth.AuthRequired(), handler.UpdateSettings)
		profileGroup.GET("/:profileID", auth.AuthOptional(), middleware.Exist(handler.service.Exist, handler.logger, "profileID"), handler.GetDetailProfile)
	}
}

func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	userInfo, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	uProfile, progress, err := h.service.GetProfileWithProgress(c.Request.Context(), uint(userInfo.UserID))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewPersonalProfileResponseWithGamification(uProfile, progress))
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

func (h *ProfileHandler) GetDetailProfile(c *gin.Context) {
	pID, err := strconv.Atoi(c.Param("profileID"))
	if err != nil {
		err = apperror.NewFieldValidationError("id", "id must be a number", "value_error", c.Param("id"))
		errorhandler.HandleError(c, err, h.logger)
	}

	profile, progress, err := h.service.GetProfileWithProgress(c.Request.Context(), uint(pID))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	response := dto.NewPersonalProfileResponseWithGamification(profile, progress)

	// Признак подписки имеет смысл только для чужого профиля, открытого
	// авторизованным пользователем: анониму подписываться нечем, а на себя
	// подписка запрещена. В остальных случаях поле не отдаётся вовсе.
	if viewerID, err := helper.GetUserID(c); err == nil && uint(viewerID) != uint(pID) {
		subscribed, err := h.subscriptionService.IsSubscribed(c.Request.Context(), uint(viewerID), uint(pID))
		if err != nil {
			// Профиль важнее флага: при сбое отдаём профиль без isSubscribed.
			h.logger.Warn("failed to resolve subscription state",
				zap.Int("viewer_id", viewerID), zap.Int("profile_id", pID), zap.Error(err))
		} else {
			response = response.WithSubscribed(subscribed)
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var req dto.ProfileUpdateRequest
	if err := c.ShouldBind(&req); err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	var avatar *profileservice.AvatarUpload
	fileHeader, err := c.FormFile("avatar")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	if fileHeader != nil {
		f, err := fileHeader.Open()
		if err != nil {
			errorhandler.HandleError(c, err, h.logger)
			return
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			errorhandler.HandleError(c, err, h.logger)
			return
		}
		avatar = &profileservice.AvatarUpload{
			Data:     data,
			FileName: fileHeader.Filename,
		}
	}

	updated, err := h.service.UpdateProfile(c.Request.Context(), profileservice.UpdateProfileInput{
		UserID:   uint(userData.UserID),
		Username: req.Username,
		Tag:      req.Tag,
		Avatar:   avatar,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}

	c.JSON(http.StatusOK, dto.NewPersonalProfileResponse(updated))
}

func (h *ProfileHandler) GetSettings(c *gin.Context) {
	userData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	prof, err := h.service.GetProfileSettings(c.Request.Context(), uint(userData.UserID))
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewProfileSettingsResponse(prof))
}

func (h *ProfileHandler) UpdateSettings(c *gin.Context) {
	userData, err := helper.GetUserInfo(c)
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	var req dto.UpdateProfileSettingsRequest
	if err := c.ShouldBind(&req); err != nil {
		validation.AbortWithBindingError(c, err)
		return
	}

	prof, err := h.service.UpdateSettings(c.Request.Context(), uint(userData.UserID), profileservice.UpdateSettingsParams{
		ShowInSearch:   req.ShowInSearch,
		PrivateProfile: req.PrivateProfile,
	})
	if err != nil {
		errorhandler.HandleError(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.NewProfileSettingsResponse(prof))

}
