// 一次性运维工具：清除 AD 服务账号池的熔断状态（本地状态机，不影响域控账号）。
// 只重置 status / failure_count / circuit_breaker_until，保留 last_failure_reason 供事后排查。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

func main() {
	_ = godotenv.Load(".env", "configs/.env")

	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Printf("!! 配置加载失败: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("!! 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	var accts []models.ADServiceAccount
	if err := db.Order("created_at").Find(&accts).Error; err != nil {
		fmt.Printf("!! 读取账号池失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 重置前 ===")
	for _, a := range accts {
		breaker := "-"
		if a.CircuitBreakerUntil != nil {
			breaker = a.CircuitBreakerUntil.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("  %-16s status=%d failureCount=%d breakerUntil=%s available=%v\n",
			a.Username, a.Status, a.FailureCount, breaker, a.IsAvailable())
	}

	now := time.Now()
	res := db.Model(&models.ADServiceAccount{}).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"status":                 0,    // 0 = 可用
			"failure_count":          0,    // 清零，避免刚恢复就被旧计数再次熔断
			"circuit_breaker_until":  nil,  // 清除熔断到期时间
			"last_failure_at":        nil,  // 清空最近失败时间
			"manual_unlocked_at":     &now, // 记录为「人工解锁」
			"manual_unlocked_by":     "ops-reset",
			"manual_unlock_reason":   "域控侧账号正常，清理本地假熔断状态",
		})
	if res.Error != nil {
		fmt.Printf("!! 更新失败: %v\n", res.Error)
		os.Exit(1)
	}
	fmt.Printf("\n已重置 %d 个账号的熔断状态\n", res.RowsAffected)

	var after []models.ADServiceAccount
	db.Order("created_at").Find(&after)
	fmt.Println("\n=== 重置后 ===")
	for _, a := range after {
		breaker := "-"
		if a.CircuitBreakerUntil != nil {
			breaker = a.CircuitBreakerUntil.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("  %-16s status=%d failureCount=%d breakerUntil=%s available=%v\n",
			a.Username, a.Status, a.FailureCount, breaker, a.IsAvailable())
	}
}
