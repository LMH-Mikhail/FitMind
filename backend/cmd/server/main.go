package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	appinternal "fitmind/backend/internal"
	"fitmind/backend/pkg/conf"
	"fitmind/backend/pkg/database"
)

const (
	// maxHeaderBytes 限制单个 HTTP 请求头的最大字节数。
	// 作用：避免异常大请求头长期占用服务端内存。
	maxHeaderBytes = 1 << 20
)

func main() {
	// 1. 初始化应用配置。
	// 输入：进程环境变量，例如 FITMIND_HTTP_ADDR、DATABASE_URL、FITMIND_UPLOAD_DIR。
	// 输出：config 保存 HTTP、数据库和文件存储等启动配置。
	config := conf.InitConfig()

	// 2. 初始化数据库连接池。
	// 输入：config.Database.URL，即 DATABASE_URL 环境变量。
	// 输出：全局 database 连接池，后续 manager/model 通过 database.GetDB() 使用。
	// 失败处理：数据库是后端启动前置依赖，初始化失败时直接退出进程。
	if _, err := database.InitDB(&config.Database); err != nil {
		log.Printf("init database failed: %v", err)
		os.Exit(1)
	}

	// 3. 注册进程退出前的数据库连接池清理逻辑。
	// 输入：无。
	// 输出：关闭 database.InitDB 创建的全局连接池。
	// 说明：当前服务还没有 graceful shutdown，defer 至少能覆盖启动失败后的正常返回路径。
	defer func() {
		if err := database.CloseDB(); err != nil {
			log.Printf("close database failed: %v", err)
		}
	}()

	// 4. 创建 Gin 引擎。
	// gin.New 不带默认中间件，因此这里显式挂载访问日志和 panic recovery。
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())
	engine.Static(conf.ServerConfig.Storage.PublicUploadURL, conf.ServerConfig.Storage.UploadDir)

	// 5. 注册项目 HTTP 路由。
	// internal.Register 只负责把各个 controller 挂到 Gin engine 上。
	// main.go 不直接写业务路由，保持启动入口只做装配。
	appinternal.Register(engine)

	// 6. 创建标准库 HTTP Server。
	// 输入：Gin engine 作为 Handler，config.App 提供监听地址和超时配置。
	// 输出：可被 ListenAndServe 启动的 HTTP 服务实例。
	// 说明：这里集中配置超时，避免慢请求长期占用连接资源。
	server := &http.Server{
		Addr:           config.App.HTTPAddr,
		Handler:        engine,
		ReadTimeout:    config.App.ReadTimeout,
		WriteTimeout:   config.App.WriteTimeout,
		IdleTimeout:    config.App.IdleTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}

	// 7. 启动 HTTP 服务。
	// ListenAndServe 正常情况下会一直阻塞。
	// 只有启动失败、监听端口被占用或服务被关闭时才返回。
	log.Printf("fitmind backend listening on %s", config.App.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("fitmind backend stopped with error: %v", err)
		os.Exit(1)
	}
}
