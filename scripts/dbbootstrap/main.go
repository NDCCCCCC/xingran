// DB 引导工具:绕过 GORM AutoMigrate hang,直接用 pgx 连 Supabase 跑 DDL。
//   go run ./scripts/dbbootstrap
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	password := readEnvValue(".env", "DB_PASSWORD")
	host := readEnvValue(".env", "DB_HOST")
	port := readEnvValue(".env", "DB_PORT")
	user := readEnvValue(".env", "DB_USER")
	dbname := readEnvValue(".env", "DB_NAME")

	if host == "" || password == "" {
		fmt.Fprintln(os.Stderr, "DB_HOST / DB_PASSWORD 缺失")
		os.Exit(1)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)

	fmt.Printf("连接到 %s:%s/%s ...\n", host, port, dbname)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ connect failed: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)
	fmt.Println("✅ connected")

	ddl := []string{
		`CREATE TABLE IF NOT EXISTS public.sys_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version BIGINT,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    salt VARCHAR(32) NOT NULL,
    key_prefix VARCHAR(12) NOT NULL,
    user_id UUID,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    ip_whitelist JSONB NOT NULL DEFAULT '[]'::jsonb,
    description VARCHAR(500),
    inherit_perms BOOLEAN NOT NULL DEFAULT FALSE
)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON public.sys_api_keys(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON public.sys_api_keys(key_prefix)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON public.sys_api_keys(deleted_at)`,
		`CREATE TABLE IF NOT EXISTS public.sys_api_key_usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL,
    user_id UUID NOT NULL,
    method VARCHAR(10),
    path VARCHAR(500),
    status_code INTEGER,
    client_ip VARCHAR(50),
    user_agent TEXT,
    duration INTEGER,
    success BOOLEAN,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_api_key_id ON public.sys_api_key_usage_logs(api_key_id)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_created_at ON public.sys_api_key_usage_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_api_key_logs_user_id ON public.sys_api_key_usage_logs(user_id)`,
		// sys_config: model missing from AutoMigrate list (database.go line 303-386)
		// init_data.go createNetworkDeviceSystemParams 写入此表,缺则 InitData 中途失败
		`CREATE TABLE IF NOT EXISTS public.sys_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version BIGINT,
    config_name VARCHAR(100) NOT NULL,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value VARCHAR(500),
    config_type CHAR(1) NOT NULL DEFAULT 'Y',
    is_system SMALLINT NOT NULL DEFAULT 0,
    remark VARCHAR(500)
)`,
	}

	for i, stmt := range ddl {
		fmt.Printf("[%d/%d] exec ... ", i+1, len(ddl))
		if _, err := conn.Exec(ctx, stmt); err != nil {
			fmt.Fprintf(os.Stderr, "❌ DDL[%d]: %v\n", i+1, err)
			os.Exit(1)
		}
		fmt.Println("ok")
	}
	fmt.Println("✅ done")
}

func readEnvValue(path, key string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	return ""
}