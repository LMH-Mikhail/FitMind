package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"fitmind/backend/pkg/header/common"
)

type HealthController struct{}

func (ctrl *HealthController) Add(engine *gin.Engine) {
	group := engine.Group("/api/v1")
	group.GET("/health", ctrl.Check)
}

func (ctrl *HealthController) Check(ctx *gin.Context) {
	response := common.NewResponse()
	defer response.Response(ctx)

	response.Code = http.StatusOK
	response.Data = gin.H{
		"status":    "ok",
		"service":   "fitmind-backend",
		"timestamp": time.Now().Format(time.RFC3339),
	}
}
