package v1

import (
	"fitmind/backend/internal/middleware"
	"fitmind/backend/internal/service"
	"fitmind/backend/pkg/conf"
	"fitmind/backend/pkg/database"
	"fitmind/backend/pkg/header/common"
	"fitmind/backend/pkg/manager"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UploadController struct {
	uploadService *service.UploadService
	authService   *service.AuthService
}

func (ctrl *UploadController) Add(engine *gin.Engine) {
	group := engine.Group("/api/v1")

	protected := group.Group("")
	protected.Use(middleware.Auth(ctrl.auth()))

	protected.POST("/uploads/clothing-image", ctrl.UploadClothingImage)
}

func (ctrl *UploadController) UploadClothingImage(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.SetError(http.StatusUnauthorized, "未登录")
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		response.SetError(http.StatusBadRequest, "请选择要上传的衣物图片")
		return
	}

	data, err := ctrl.service().UploadClothingImage(ctx.Request.Context(), userID, fileHeader)
	if err != nil {
		writeUploadError(response, err)
		return
	}

	response.Data = data
}

func (ctrl *UploadController) service() *service.UploadService {
	if ctrl.uploadService != nil {
		return ctrl.uploadService
	}

	ctrl.uploadService = service.NewUploadService(manager.NewUploadManager(conf.ServerConfig.Storage))
	return ctrl.uploadService
}

func (ctrl *UploadController) auth() *service.AuthService {
	if ctrl.authService != nil {
		return ctrl.authService
	}

	ctrl.authService = service.NewAuthService(
		manager.NewUserManager(database.GetDB()),
		conf.ServerConfig.Auth,
	)
	return ctrl.authService
}

func writeUploadError(response *common.BaseResponse, err error) {
	switch service.UploadKindOf(err) {
	case service.UploadKindInvalidInput:
		response.SetError(http.StatusBadRequest, service.UploadMessageOf(err))
	case service.UploadKindUnauthorized:
		response.SetError(http.StatusUnauthorized, service.UploadMessageOf(err))
	case service.UploadKindUnsupportedType:
		response.SetError(http.StatusUnsupportedMediaType, service.UploadMessageOf(err))
	case service.UploadKindTooLarge:
		response.SetError(http.StatusRequestEntityTooLarge, service.UploadMessageOf(err))
	default:
		response.SetError(http.StatusInternalServerError, "服务器内部错误")
	}
}
