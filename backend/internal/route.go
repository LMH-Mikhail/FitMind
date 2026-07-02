package internal

import (
	"github.com/gin-gonic/gin"

	v1 "fitmind/backend/internal/handler/v1"
)

// Route定义所有HTTP controller必须实现的路由注册接口
// 输入：*gin.Engine，由mmain.go创建并传入
// 输出：controller在Add方法内部把自己的HTTP路由注册到Gin
type Route interface {
	Add(router *gin.Engine)
}

// Register统一注册项目内全部HTTP路由
// 输入：*gin.Engine
// 输出：无返回值；函数内部按controller逐个调用Add
func Register(engine *gin.Engine) {
	routes := []Route{
		&v1.HealthController{},
	}

	for _, route := range routes {
		route.Add(engine)
	}
}
