package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

// 缓存键迁移清理工具
// 用于清理 Redis 中的旧规范缓存键（无 cache: 前缀的键）

var (
	redisHost     = flag.String("host", "localhost", "Redis host")
	redisPort     = flag.Int("port", 6379, "Redis port")
	redisPassword = flag.String("password", "", "Redis password")
	redisDB       = flag.Int("db", 0, "Redis database number")
	dryRun        = flag.Bool("dry-run", true, "只显示将要删除的键，不实际删除")
	verbose       = flag.Bool("verbose", false, "显示详细输出")
)

// 旧规范键模式（需要删除的键）
var oldPatterns = []string{
	"user:*",
	"role:*",
	"dept:*",
	"menu:*",
	"dict:*",
	"post:*",
	// 注意：不要删除 config:* 开头的键，因为可能有其他用途
}

// 新规范键前缀（需要保留的键）
var newPrefixes = []string{
	"cache:user:",
	"cache:role:",
	"cache:dept:",
	"cache:menu:",
	"cache:dict:",
	"cache:post:",
	"cache:config:",
}

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
	fmt.Println("成功连接到 Redis")

	if *dryRun {
		fmt.Println("=== DRY RUN 模式：只显示将要删除的键，不实际删除 ===")
	}

	// 统计和清理旧键
	totalOldKeys := 0
	totalDeleted := 0

	for _, pattern := range oldPatterns {
		fmt.Printf("\n处理模式: %s\n", pattern)

		// 扫描匹配的键
		var keys []string
		iter := client.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			// 跳过新规范的键（避免误删）
			if isNewKeyFormat(key) {
				if *verbose {
					fmt.Printf("  跳过新规范键: %s\n", key)
				}
				continue
			}
			keys = append(keys, key)
		}
		if err := iter.Err(); err != nil {
			log.Printf("扫描键失败: %v", err)
			continue
		}

		if len(keys) == 0 {
			fmt.Printf("  未找到匹配的旧键\n")
			continue
		}

		totalOldKeys += len(keys)
		fmt.Printf("  找到 %d 个旧键\n", len(keys))

		// 显示示例键
		showSampleKeys(keys)

		// 删除键
		if !*dryRun {
			// 批量删除
			pipeline := client.Pipeline()
			for _, key := range keys {
				pipeline.Del(ctx, key)
			}
			cmds, err := pipeline.Exec(ctx)
			if err != nil {
				log.Printf("批量删除失败: %v", err)
				continue
			}

			// 统计删除数量
			deleted := 0
			for _, cmd := range cmds {
				if cmd != nil {
					deleted++
				}
			}
			totalDeleted += deleted
			fmt.Printf("  已删除 %d 个键\n", deleted)
		}
	}

	// 输出总结
	fmt.Println("\n=== 总结 ===")
	if *dryRun {
		fmt.Printf("DRY RUN 模式：找到 %d 个旧键需要删除\n", totalOldKeys)
		fmt.Println("要实际删除，请运行: ./cleanup_tool -dry-run=false")
	} else {
		fmt.Printf("总共删除了 %d 个旧键\n", totalDeleted)
		if totalDeleted < totalOldKeys {
			fmt.Printf("警告：部分键删除失败（计划删除 %d 个）\n", totalOldKeys)
		}
	}

	// 验证清理结果
	verifyCleanup(ctx, client)
}

// isNewKeyFormat 检查是否为新规范键
func isNewKeyFormat(key string) bool {
	for _, prefix := range newPrefixes {
		if strings.HasPrefix(key, "xingran:"+prefix) || strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// showSampleKeys 显示示例键
func showSampleKeys(keys []string) {
	sampleSize := 5
	if len(keys) < sampleSize {
		sampleSize = len(keys)
	}

	fmt.Print("  示例键: ")
	for i := 0; i < sampleSize; i++ {
		fmt.Printf("%s ", keys[i])
	}
	if len(keys) > sampleSize {
		fmt.Printf("...(共 %d 个)", len(keys))
	}
	fmt.Println()
}

// verifyCleanup 验证清理结果
func verifyCleanup(ctx context.Context, client *redis.Client) {
	fmt.Println("\n=== 验证清理结果 ===")

	hasOldKeys := false
	for _, pattern := range oldPatterns {
		iter := client.Scan(ctx, 0, pattern, 100).Iterator()
		oldKeyCount := 0
		for iter.Next(ctx) {
			key := iter.Val()
			if !isNewKeyFormat(key) {
				oldKeyCount++
				if oldKeyCount <= 3 {
					fmt.Printf("  发现旧键: %s (模式: %s)\n", key, pattern)
				}
			}
		}

		if oldKeyCount > 0 {
			hasOldKeys = true
			fmt.Printf("警告：模式 %s 仍有 %d 个旧键\n", pattern, oldKeyCount)
		}
	}

	if !hasOldKeys {
		fmt.Println("✓ 所有旧键已清理完成")
	} else {
		fmt.Println("✗ 仍有旧键未清理，请重新运行清理工具")
	}
}
