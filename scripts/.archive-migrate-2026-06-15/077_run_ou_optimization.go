//go:build ignore

package main

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	_ "github.com/xingran-next/xingran-go-backend/internal/models"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化日志
	logConfig := &applogger.Config{
		Level:         cfg.Log.Level,
		LogDir:        cfg.Log.LogDir,
		MaxSize:       cfg.Log.MaxSize,
		MaxBackups:    cfg.Log.MaxBackups,
		MaxAge:        cfg.Log.MaxAge,
		Compress:      cfg.Log.Compress,
		ConsoleOutput: true,
	}
	if err := applogger.Init(logConfig); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}

	// 初始化核心组件
	appCore, err := core.New(cfg)
	if err != nil {
		log.Fatalf("创建核心模块失败: %v", err)
	}
	if err := appCore.Init(); err != nil {
		log.Fatalf("初始化核心模块失败: %v", err)
	}
	defer appCore.Close()

	db := appCore.GetDB()

	// 执行优化SQL
	sql := `
	-- 添加复合索引以优化OU查询
	CREATE INDEX IF NOT EXISTS idx_ad_ou_dn
	ON sys_ad_ou (ad_config_id, ou_dn);
	`

	if err := db.Exec(sql).Error; err != nil {
		log.Fatalf("执行迁移失败: %v", err)
	}

	// 验证索引是否创建成功
	var count int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE tablename = 'sys_ad_ou'
		AND indexname = 'idx_ad_ou_dn'
	`).Scan(&count).Error; err != nil {
		log.Fatalf("验证索引失败: %v", err)
	}

	if count > 0 {
		fmt.Println("✓ 迁移 077 执行成功: 索引 idx_ad_ou_dn 已创建")
	} else {
		log.Fatalf("✗ 迁移 077 验证失败: 索引未创建")
	}
}
