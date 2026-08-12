package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

// 缓存键迁移验证工具
// 用于验证缓存键命名规范迁移是否成功

var (
	redisHost     = flag.String("host", "localhost", "Redis host")
	redisPort     = flag.Int("port", 6379, "Redis port")
	redisPassword = flag.String("password", "", "Redis password")
	redisDB       = flag.Int("db", 0, "Redis database number")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	// 连接 Redis
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", *redisHost, *redisPort),
		Password: *redisPassword,
		DB:       *redisDB,
	})

	// 测试连接
	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("连接 Redis 失败: %v", err)
	}
	fmt.Println("✓ 成功连接到 Redis")

	// 验证迁移结果
	fmt.Println("\n=== 缓存键迁移验证 ===")

	checkOldKeys(ctx, client)
	checkNewKeys(ctx, client)
	checkRedisStats(ctx, client)

	fmt.Println("\n=== 验证完成 ===")
}

// checkOldKeys 检查旧规范键
func checkOldKeys(ctx context.Context, client *redis.Client) {
	fmt.Println("1. 检查旧规范键（应已全部清理）")

	oldPatterns := []string{
		"user:*",
		"role:*",
		"dept:*",
		"menu:*",
		"dict:*",
		"post:*",
	}

	hasOldKeys := false
	for _, pattern := range oldPatterns {
		var count int
		iter := client.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			// 跳过新规范键
			if isOldKeyFormat(key) {
				count++
				if count <= 3 {
					fmt.Printf("  ✗ 发现旧键: %s\n", key)
				}
			}
		}
		if count > 0 {
			hasOldKeys = true
			fmt.Printf("  模式 '%s': %d 个旧键\n", pattern, count)
		}
	}

	if !hasOldKeys {
		fmt.Println("  ✓ 未发现旧规范键")
	} else {
		fmt.Println("  ✗ 仍有旧规范键未清理")
	}
}

// checkNewKeys 检查新规范键
func checkNewKeys(ctx context.Context, client *redis.Client) {
	fmt.Println("\n2. 检查新规范键（应正常使用）")

	newPatterns := []string{
		"cache:user:*",
		"cache:role:*",
		"cache:dept:*",
		"cache:menu:*",
		"cache:dict:*",
		"cache:post:*",
	}

	totalNewKeys := 0
	for _, pattern := range newPatterns {
		var count int
		iter := client.Scan(ctx, 0, "xingran:"+pattern, 100).Iterator()
		for iter.Next(ctx) {
			count++
		}
		totalNewKeys += count

		if count > 0 {
			fmt.Printf("  ✓ 模式 '%s': %d 个键\n", pattern, count)
		}
	}

	if totalNewKeys > 0 {
		fmt.Printf("  ✓ 共发现 %d 个新规范键\n", totalNewKeys)
	} else {
		fmt.Println("  ✗ 未发现新规范键（可能缓存未使用）")
	}
}

// checkRedisStats 检查 Redis 统计信息
func checkRedisStats(ctx context.Context, client *redis.Client) {
	fmt.Println("\n3. Redis 统计信息")

	// 获取总键数
	totalKeys, err := client.DBSize(ctx).Result()
	if err != nil {
		log.Printf("  ✗ 获取键总数失败: %v", err)
	} else {
		fmt.Printf("  总键数: %d\n", totalKeys)
	}

	// 获取内存使用
	info, err := client.Info(ctx, "memory").Result()
	if err != nil {
		log.Printf("  ✗ 获取内存信息失败: %v", err)
	} else {
		// 解析 used_memory
		lines := parseInfo(info)
		if usedMemory, ok := lines["used_memory"]; ok {
			fmt.Printf("  已用内存: %s bytes\n", usedMemory)
		}
	}
}

// isOldKeyFormat 检查是否为旧格式键
func isOldKeyFormat(key string) bool {
	// 移除 xingran: 前缀
	cleanKey := key
	if len(key) > 6 && key[:6] == "xingran:" {
		cleanKey = key[6:]
	}

	// 检查是否以旧规范模式开头
	oldPrefixes := []string{
		"user:", "role:", "dept:", "menu:", "dict:", "post:",
	}

	for _, prefix := range oldPrefixes {
		if len(cleanKey) > len(prefix) && cleanKey[:len(prefix)] == prefix {
			// 确保不是新规范键（以 cache: 开头）
			if !strings.HasPrefix(cleanKey, "cache:") {
				return true
			}
		}
	}

	return false
}

// parseInfo 解析 Redis INFO 命令输出
func parseInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, ")") || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}
