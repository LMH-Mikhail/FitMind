package conf

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr           = ":8080"
	defaultReadTimeoutSecond  = 30
	defaultWriteTimeoutSecond = 30
	defaultIdleTimeoutSecond  = 60
	defaultUploadDir          = "./uploads"
	defaultPublicUploadURL    = "/uploads"

	defaultJWTIssuer         = "fitmind"
	defaultJWTSecret         = "fitmind-development-secret-change-me"
	defaultAccessTokenSecond = 24 * 60 * 60
)

// ServerConfig保存应用启动后使用的全局配置
// 说明：
// 1. main.go启动时调用InitConfig初始化
// 2. 其他包只读取配置，不在运行中修改配置
// 3. 当前配置全部来自环境变量，先不引入yaml/viper，保持项目简单
var ServerConfig Config

// Config是FitMind后端的总配置结构
// App：HTTP服务相关配置
// Database：PostgreSQL数据库配置
// Storage：本地上传文件配置
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Storage  StorageConfig
	Auth     AuthConfig
}

// AppConfig保存HTTP服务配置
// HTTPAddr：服务监听地址，例如 “:8080”或“127.0.0.1:8080”
// ReadTimeout：读取请求的超时时间
// WriteTimeout：写入响应的超时时间
// IdleTimeout：keep-alive空闲连接超时时间
type AppConfig struct {
	HTTPAddr     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig保存数据库配置
// URL：PostgreSQL连接串，例如：
// postgres://fitmind:password@localhost:5433/fitmind_db?sslmode=disable
type DatabaseConfig struct {
	URL string
}

// StorageConfig保存文件存储配置
// UploadDir：后端实际保存上传文件的本地目录
// PublicUploadURL：对外访问上传文件时使用的URL前缀
type StorageConfig struct {
	UploadDir       string
	PublicUploadURL string
}

type AuthConfig struct {
	JWTSecret      string
	JWTIssuer      string
	AccessTokenTTL time.Duration
}

// InitConfig从环境变量初始化全局配置
// 输入：进程环境变量
// 输出：初始化后的Config
// 注意：这个函数不连接数据库、不创建目录，只负责读取配置
func InitConfig() Config {
	ServerConfig = Config{
		App: AppConfig{
			HTTPAddr:     getEnv("FITMIND_HTTP_ADDR", defaultHTTPAddr),
			ReadTimeout:  getEnvDurationSecond("FITMIND_READ_TIMEOUT_SECONDS", defaultReadTimeoutSecond),
			WriteTimeout: getEnvDurationSecond("FITMIND_WRITE_TIMEOUT_SECONDS", defaultWriteTimeoutSecond),
			IdleTimeout:  getEnvDurationSecond("FITMIND_IDLE_TIMEOUT_SECONDS", defaultIdleTimeoutSecond),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Storage: StorageConfig{
			UploadDir:       getEnv("FITMIND_UPLOAD_DIR", defaultUploadDir),
			PublicUploadURL: getEnv("FITMIND_PUBLIC_UPLOAD_URL", defaultPublicUploadURL),
		},
		Auth: AuthConfig{
			JWTSecret:      getEnv("FITMIND_JWT_SECRET", defaultJWTSecret),
			JWTIssuer:      getEnv("FITMIND_JWT_ISSUER", defaultJWTIssuer),
			AccessTokenTTL: getEnvDurationSecond("FITMIND_ACCESS_TOKEN_TTL_SECONDS", int(defaultAccessTokenSecond)),
		},
	}

	return ServerConfig
}

// getEnv读取字符串环境变量
// 输入：
// - key：环境变量名
// - fallback：环境变量为空时使用的默认值
// 输出：去掉首尾空格后的配置值
func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// getEnvDurationSecond读取秒级超时环境变量
// 输入：
// - key：环境变量名
// - fallbackSecond：环境变量为空或非法时使用的默认秒数
// 输出：time.Duration，单位为秒
func getEnvDurationSecond(key string, fallbackSecond int) time.Duration {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return time.Duration(fallbackSecond) * time.Second
	}

	second, err := strconv.Atoi(rawValue)
	if err != nil || second <= 0 {
		return time.Duration(fallbackSecond) * time.Second
	}
	return time.Duration(second) * time.Second
}
