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
	"strconv"

	"github.com/gin-gonic/gin"
)

type ClothingItemController struct {
	clothingService *service.ClothingItemService
	authService     *service.AuthService
}

func (ctrl *ClothingItemController) Add(engine *gin.Engine) {
	group := engine.Group("/api/v1")
	group.Use(middleware.Auth(ctrl.auth()))

	group.POST("/clothing-items", ctrl.Create)
	group.GET("/clothing-items", ctrl.List)
	group.GET("/clothing-items/:id", ctrl.Detail)
	group.PUT("/clothing-items/:id", ctrl.Update)
	group.DELETE("/clothing-items/:id", ctrl.Delete)
}

func (ctrl *ClothingItemController) Create(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.SetError(http.StatusUnauthorized, "未登录")
		return
	}

	var request dto.CreateClothingItemRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.SetError(http.StatusBadRequest, "请求体格式错误")
		return
	}

	item, err := ctrl.service().Create(ctx.Request.Context(), userID, request)
	if err != nil {
		writeClothingError(response, err)
		return
	}

	response.Data = item
}

func (ctrl *ClothingItemController) List(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.SetError(http.StatusUnauthorized, "未登录")
		return
	}

	pageNum := parsePositiveInt(ctx.Query("pageNum"), 1)
	pageSize := parsePositiveInt(ctx.Query("pageSize"), 20)

	page, err := ctrl.service().List(
		ctx.Request.Context(),
		userID,
		pageNum,
		pageSize,
		ctx.Query("category"),
		ctx.Query("status"),
	)
	if err != nil {
		writeClothingError(response, err)
		return
	}

	response.Data = page
}

func (ctrl *ClothingItemController) Detail(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.SetError(http.StatusUnauthorized, "未登录")
		return
	}

	item, err := ctrl.service().GetByID(ctx.Request.Context(), userID, ctx.Param("id"))
	if err != nil {
		writeClothingError(response, err)
		return
	}

	response.Data = item
}

func (ctrl *ClothingItemController) Update(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.SetError(http.StatusUnauthorized, "未登录")
		return
	}

	var request dto.UpdateClothingItemRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.SetError(http.StatusBadRequest, "请求体格式错误")
		return
	}

	item, err := ctrl.service().Update(ctx.Request.Context(), userID, ctx.Param("id"), request)
	if err != nil {
		writeClothingError(response, err)
		return
	}

	response.Data = item
}

func (ctrl *ClothingItemController) Delete(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		response.SetError(http.StatusUnauthorized, "未登录")
		return
	}

	id := ctx.Param("id")
	if err := ctrl.service().Delete(ctx.Request.Context(), userID, id); err != nil {
		writeClothingError(response, err)
		return
	}

	response.Data = gin.H{
		"id":      id,
		"deleted": true,
	}
}

func (ctrl *ClothingItemController) service() *service.ClothingItemService {
	if ctrl.clothingService != nil {
		return ctrl.clothingService
	}

	ctrl.clothingService = service.NewClothingItemService(manager.NewClothingItemManager(database.GetDB()))
	return ctrl.clothingService
}

func (ctrl *ClothingItemController) auth() *service.AuthService {
	if ctrl.authService != nil {
		return ctrl.authService
	}

	ctrl.authService = service.NewAuthService(
		manager.NewUserManager(database.GetDB()),
		conf.ServerConfig.Auth,
	)
	return ctrl.authService
}

func writeClothingError(response *common.BaseResponse, err error) {
	switch service.ClothingKindOf(err) {
	case service.ClothingKindInvalidInput:
		response.SetError(http.StatusBadRequest, service.ClothingMessageOf(err))
	case service.ClothingKindNotFound:
		response.SetError(http.StatusNotFound, service.ClothingMessageOf(err))
	default:
		response.SetError(http.StatusInternalServerError, "服务器内部错误")
	}
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
