package middleware

import (
	"fitmind/backend/internal/service"
	"fitmind/backend/pkg/header/common"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ContextUserIDKey = "userID"

func Auth(authService *service.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := extractBearerToken(ctx.GetHeader("Authorization"))
		if token == "" {
			abortUnauthorized(ctx, "缺少访问令牌")
			return
		}
		userID, err := authService.ParseAccessToken(token)
		if err != nil {
			abortUnauthorized(ctx, service.MessageOf(err))
			return
		}

		ctx.Set(ContextUserIDKey, userID)
		ctx.Next()
	}
}

func GetUserID(ctx *gin.Context) (string, bool) {
	value, exists := ctx.Get(ContextUserIDKey)
	if !exists {
		return "", false
	}

	userID, ok := value.(string)
	return userID, ok && userID != ""
}

func extractBearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func abortUnauthorized(ctx *gin.Context, message string) {
	response := common.NewResponse()
	response.SetError(http.StatusUnauthorized, message)
	response.Response(ctx)
	ctx.Abort()
}
