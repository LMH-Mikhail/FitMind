package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"fitmind/backend/pkg/conf"
)

const (
	postgresDriverName = "pgx"

	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 5 * time.Minute
	defaultPingTimeout     = 5 * time.Second
)

// SourceDB保存应用数据库连接
// DB：标准库database/sql连接池
// 说明：当前项目使用PostgreSQL，底层驱动为pgx stdlib
type SourceDB struct {
	DB *sql.DB
}

var dbConn *SourceDB

// InitDB初始化PostgreSQL数据库连接池
// 输入：config.Database.URL，格式示例：
// postgres://fitmind:password@localhost:5433/fitmind_db?sslmode=disable
// 输出：初始化后的SourceDB
// 行为：
// 1. 校验DATABASE_URL不为空
// 2. 创建sql.DB连接池
// 3. 配置连接池参数
// 4. Ping数据库，确保连接可用
// 5. 写入全局dbConn，供manager/model使用
func InitDB(config *conf.DatabaseConfig) (*SourceDB, error) {
	if config == nil {
		return nil, fmt.Errorf("database config is nil")
	}

	databaseURL := strings.TrimSpace(config.URL)
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is empty")
	}

	sqlDB, err := sql.Open(postgresDriverName, databaseURL)
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(defaultMaxOpenConns)
	sqlDB.SetMaxIdleConns(defaultMaxIdleConns)
	sqlDB.SetConnMaxLifetime(defaultConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()

	if err = sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	dbConn = &SourceDB{DB: sqlDB}
	return dbConn, nil
}

// GetDB返回全局数据库连接池
// 输入：无
// 输出：*sql.DB
// 注意：必须先在main.go中调用InitDB，否则这里会panic
// 这样可以在开发阶段尽早暴露“忘记初始化数据库”的错误
func GetDB() *sql.DB {
	if dbConn == nil || dbConn.DB == nil {
		panic("database is nil, call database.InitDB first")
	}
	return dbConn.DB
}

// GetSourceDB返回完整数据库包装结构
// 输入：无
// 输出：*SourceDB
// 使用场景：后续如果SourceDB增加事务、健康状态或配置字段，可以通过这个函数取得
func GetSourceDB() *SourceDB {
	if dbConn == nil || dbConn.DB == nil {
		panic("database is nil, call database.InitDB first")
	}
	return dbConn
}

// Ping检查当前数据库连接是否可用
// 输入：ctx控制超时和取消
// 输出：error，nil表示数据库当前可访问
// 使用场景：health/db接口或启动后的数据库连通性检查
func Ping(ctx context.Context) error {
	return GetDB().PingContext(ctx)
}

// CloseDB关闭全局数据库连接池
// 输入：无
// 输出：关闭连接池时产生的error
// 使用场景：后续做graceful shutdown时调用
func CloseDB() error {
	if dbConn == nil || dbConn.DB == nil {
		return nil
	}

	err := dbConn.DB.Close()
	dbConn = nil
	return err
}
