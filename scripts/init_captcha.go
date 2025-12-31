package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接信息
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		"10.62.10.34",
		5432,
		"postgres",
		"Cpic1234",
		"ruoyi",
	)

	// 连接数据库
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}

	fmt.Println("数据库连接成功")

	// SQL插入语句
	sqls := []string{
		`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark)
		 VALUES (gen_random_uuid(), '验证码开关', 'sys.account.captchaEnabled', 'disabled', 'Y', 1, '验证码开关: disabled=停用 normal=数字验证码 slider=滑动验证码')
		 ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;`,

		`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark)
		 VALUES (gen_random_uuid(), '验证码类型', 'sys.account.captchaType', '4', 'Y', 1, '数字字母验证码长度(4-6位)')
		 ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;`,

		`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark)
		 VALUES (gen_random_uuid(), '验证码有效期', 'sys.account.captchaExpireTime', '5', 'Y', 1, '验证码有效期(分钟)')
		 ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;`,

		`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark)
		 VALUES (gen_random_uuid(), '验证码最大尝试次数', 'sys.account.captchaMaxAttempts', '3', 'Y', 1, '验证码验证失败最大次数')
		 ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;`,

		`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark)
		 VALUES (gen_random_uuid(), 'IP限流阈值', 'sys.account.ipRateLimit', '10', 'Y', 1, '每分钟获取验证码最大次数')
		 ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;`,

		`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark)
		 VALUES (gen_random_uuid(), '登录最大重试次数', 'sys.account.loginMaxRetry', '5', 'Y', 1, '登录失败最大次数')
		 ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;`,

		`INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_system, remark)
		 VALUES (gen_random_uuid(), '账号锁定时间', 'sys.account.loginLockTime', '30', 'Y', 1, '账号锁定时间(分钟)')
		 ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;`,
	}

	// 执行SQL
	for i, sql := range sqls {
		_, err := db.Exec(sql)
		if err != nil {
			log.Printf("执行第 %d 条SQL失败: %v\nSQL: %s", i+1, err, sql)
			continue
		}
		fmt.Printf("✓ 第 %d 条SQL执行成功\n", i+1)
	}

	fmt.Println("\n验证码配置参数初始化完成！")
	fmt.Println("\n配置说明:")
	fmt.Println("1. sys.account.captchaEnabled: 验证码开关")
	fmt.Println("   - disabled: 完全停用验证码")
	fmt.Println("   - normal: 使用数字字母验证码")
	fmt.Println("   - slider: 使用滑动拼图验证码")
	fmt.Println("\n2. sys.account.captchaType: 数字验证码长度 (4-6)")
	fmt.Println("3. sys.account.captchaExpireTime: 验证码有效期 (分钟)")
	fmt.Println("4. sys.account.captchaMaxAttempts: 单个验证码最大验证次数")
	fmt.Println("5. sys.account.ipRateLimit: IP限流 (次/分钟)")
	fmt.Println("6. sys.account.loginMaxRetry: 登录失败最大次数")
	fmt.Println("7. sys.account.loginLockTime: 账号锁定时间 (分钟)")
}
