package v1

import (
	"fitmind/backend/internal/middleware"
	"fitmind/backend/internal/service"
	"fitmind/backend/pkg/conf"
	"fitmind/backend/pkg/database"
	"fitmind/backend/pkg/header/common"
	"fitmind/backend/pkg/header/dto"
	"fitmind/backend/pkg/manager"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *service.AuthService
}

func NewAuthController(authService *service.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

func (ctrl *AuthController) Add(enging *gin.Engine) {
	authService := ctrl.service()

	group := enging.Group("/api/v1")
	group.POST("/auth/register", ctrl.Register)
	group.POST("/auth/login", ctrl.Login)

	protected := group.Group("")
	protected.Use(middleware.Auth(authService))
	protected.GET("/me", ctrl.Me)
}

func (ctrl *AuthController) Register(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	var request dto.RegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.SetError(http.StatusBadRequest, "请求体格式错误")
		return
	}

	data, err := ctrl.service().Register(ctx.Request.Context(), request)
	if err != nil {
		writeServiceError(response, err)
		return
	}

	response.Data = data
}

func (ctrl *AuthController) Login(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	var request dto.LoginRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.SetError(http.StatusBadRequest, "请求体格式错误")
		return
	}

	data, err := ctrl.service().Login(ctx.Request.Context(), request)
	if err != nil {
		writeServiceError(response, err)
		return
	}

	response.Data = data
}

func (ctrl *AuthController) Me(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	userID, exists := middleware.GetUserID(ctx)
	if !exists {
		response.SetError(http.StatusUnauthorized, "用户未登录")
		return
	}

	user, err := ctrl.service().CurrentUser(ctx.Request.Context(), userID)
	if err != nil {
		writeServiceError(response, err)
		return
	}

	response.Data = user
}

func (ctrl *AuthController) service() *service.AuthService {
	if ctrl.authService != nil {
		return ctrl.authService
	}

	userManager := manager.NewUserManager(database.GetDB())
	ctrl.authService = service.NewAuthService(userManager, conf.ServerConfig.Auth)
	return ctrl.authService
}

func writeServiceError(response *common.BaseResponse, err error) {
	switch service.KindOf(err) {
	case service.KindInvalidInput:
		response.SetError(http.StatusBadRequest, service.MessageOf(err))
	case service.KindConflict:
		response.SetError(http.StatusConflict, service.MessageOf(err))
	case service.KindUnauthorized:
		response.SetError(http.StatusUnauthorized, service.MessageOf(err))
	case service.KindNotFound:
		response.SetError(http.StatusNotFound, service.MessageOf(err))
	case service.KindForbidden:
		response.SetError(http.StatusForbidden, service.MessageOf(err))
	default:
		response.SetError(http.StatusInternalServerError, "服务器内部错误")
	}
}
