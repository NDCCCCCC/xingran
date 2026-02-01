package db

import (
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Postgres PostgreSQL数据库连接
type Postgres struct {
	DB *gorm.DB
}

// NewPostgres 创建PostgreSQL连接
// 注意：此函数未被当前系统使用，保留用于可能的未来需求
// 实际使用的数据库初始化函数在 database.go 中的 NewDatabase()
func NewPostgres(cfg *config.DatabaseConfig) (*Postgres, error) {
	dsn := cfg.GetDSN()

	// 配置GORM - 使用可配置的日志过滤器
	filterConfig := LogFilterConfig{
		MinLevel: logger.Silent,
		FilterTypes: map[LogType]bool{
			LogTypeSQL:       true, // 过滤普通 SQL
			LogTypeSlowQuery: true, // 过滤慢查询
			LogTypeError:     false, // 保留错误
			LogTypeNilRecord: true, // 过滤未找到记录
		},
		SlowThreshold: 1000,
	}

	customLogger := NewFilterLogger(filterConfig)

	gormConfig := &gorm.Config{
		Logger: customLogger,
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层的sql.DB对象进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	applogger.Infof("PostgreSQL连接成功")

	return &Postgres{DB: db}, nil
}

// Close 关闭数据库连接
func (p *Postgres) Close() error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB 获取数据库实例
func (p *Postgres) GetDB() *gorm.DB {
	return p.DB
}
