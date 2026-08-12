package config

import (
	"context"
	"strings"
	"testing"
)

// TestValidate 用表驱动测试覆盖所有 Validate 校验分支。
func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError string // 子串匹配;空表示期望无错
	}{
		{
			name: "合法 release 模式配置",
			cfg: Config{
				Server:   ServerConfig{Mode: "release"},
				Security: SecurityConfig{SM4Key: "kS1dZK+9Z2Wc0vRPa2YtRw=="},
			},
			wantError: "",
		},
		{
			name: "合法 debug 模式配置",
			cfg: Config{
				Server:   ServerConfig{Mode: "debug"},
				Security: SecurityConfig{SM4Key: "kS1dZK+9Z2Wc0vRPa2YtRw=="},
			},
			wantError: "",
		},
		{
			name: "合法 test 模式配置",
			cfg: Config{
				Server:   ServerConfig{Mode: "test"},
				Security: SecurityConfig{SM4Key: "kS1dZK+9Z2Wc0vRPa2YtRw=="},
			},
			wantError: "",
		},
		{
			name: "SM4 密钥为空 → 必须报错",
			cfg: Config{
				Server:   ServerConfig{Mode: "release"},
				Security: SecurityConfig{SM4Key: ""},
			},
			wantError: "security.sm4_key 必须配置",
		},
		{
			name: "Server.Mode 非法值 → 必须报错",
			cfg: Config{
				Server:   ServerConfig{Mode: "production"}, // 注意:不是 release
				Security: SecurityConfig{SM4Key: "valid"},
			},
			wantError: `server.mode 必须是 debug/release/test`,
		},
		{
			name: "连接池配置反了 → 必须报错",
			cfg: Config{
				Server:   ServerConfig{Mode: "release"},
				Security: SecurityConfig{SM4Key: "valid"},
				Database: DatabaseConfig{
					MaxOpenConns: 5,
					MaxIdleConns: 10,
				},
			},
			wantError: "max_open_conns(5) 不能小于 max_idle_conns(10)",
		},
		{
			name: "连接池都设为 0 → 不触发校验(允许由外部默认值接管)",
			cfg: Config{
				Server:   ServerConfig{Mode: "release"},
				Security: SecurityConfig{SM4Key: "valid"},
				Database: DatabaseConfig{
					MaxOpenConns: 0,
					MaxIdleConns: 0,
				},
			},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("期望无错,实际得到: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望错误包含 %q,实际无错", tt.wantError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("错误信息不匹配\n期望包含: %q\n实际: %q", tt.wantError, err.Error())
			}
		})
	}
}

// TestGetDSN_URLEscape 验证密码含特殊字符时 URL 编码正确(IN-3 改名)。
//
// 历史 bug:旧实现 fmt.Sprintf("password=%s", ...) 在密码含 @ 时会让
// lib/pq 把 @ 后面的内容误解析为 host 部分,导致连接失败。
// 新实现用 net/url.UserPassword 自动 escape,这里验证这一点。
func TestGetDSN_URLEscape(t *testing.T) {
	tests := []struct {
		name     string
		cfg      DatabaseConfig
		wantSubs []string // 期望输出必须包含的子串(顺序无关)
	}{
		{
			name: "普通密码",
			cfg: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "postgres",
				Password: "secret",
				DBName:   "xingran",
				SSLMode:  "disable",
			},
			wantSubs: []string{
				"postgres://",
				"postgres:secret@localhost:5432/xingran",
				"sslmode=disable",
				"timezone=Asia%2FShanghai", // URL 编码后 / → %2F,是 net/url 标准行为
			},
		},
		{
			name: "密码含 @ 符号 → 必须 URL 转义为 %40",
			cfg: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5432,
				User:     "admin",
				Password: "p@ss:word/123",
				DBName:   "prod",
				SSLMode:  "require",
			},
			wantSubs: []string{
				"admin:p%40ss%3Aword%2F123@db.example.com:5432/prod",
				"sslmode=require",
			},
		},
		{
			name: "用户名为空(信任外部认证) → UserInfo 为空",
			cfg: DatabaseConfig{
				Host:    "localhost",
				Port:    5432,
				User:    "",
				DBName:  "test",
				SSLMode: "disable",
			},
			wantSubs: []string{
				"localhost:5432/test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetDSN()
			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("DSN 输出 %q 缺少子串 %q", got, sub)
				}
			}
		})
	}
}

// TestLoad_NoConfigFile 验证当 config.yaml 不存在时 Load 仍能成功(走默认值 + env)。
//
// 这是 dev 工作流的关键场景 —— 刚 clone 仓库、还没复制 config.example.yaml 的开发
// 不应该被启动错误挡在门外。
//
// WR-1 修复后,viper 状态在 Load() 入口已 Reset,这里无需再依赖"package 级
// 并行默认关闭"的隐含约定。
func TestLoad_NoConfigFile(t *testing.T) {
	// 在临时目录隔离 viper 工作目录,确保找不到 config.yaml。
	t.Setenv("HOME", t.TempDir()) // viper 在部分平台会回退 $HOME
	t.Setenv("USERPROFILE", t.TempDir())

	// 必须从 env 提供 SM4 密钥,否则 Validate 失败。
	t.Setenv("SM4_KEY", "testSm4KeyBase64Encoded16==")

	cfg, err := Load(context.Background())
	if err != nil {
		t.Fatalf("期望无错(走默认值),实际: %v", err)
	}
	if cfg.Security.SM4Key == "" {
		t.Fatal("SM4Key 应被 SM4_KEY 环境变量注入,但实际为空")
	}
	if cfg.Server.Mode == "" {
		t.Fatal("Server.Mode 应有默认值 debug,但实际为空")
	}
}

// TestLoad_EnvOverride 验证 bindEnvVars 显式绑定的 env 变量优先级。
//
// 步骤:
//  1. 通过 env 注入 SM4_KEY 和 SERVER_MODE
//  2. 调用 Load()
//  3. 验证 env 值覆盖了默认值
func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("SM4_KEY", "envInjectedSm4==")
	t.Setenv("SERVER_MODE", "test")
	t.Setenv("SERVER_SKIP_SETUP", "true")

	cfg, err := Load(context.Background())
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}

	if cfg.Security.SM4Key != "envInjectedSm4==" {
		t.Errorf("SM4Key 应为 %q,实际 %q", "envInjectedSm4==", cfg.Security.SM4Key)
	}
	if cfg.Server.Mode != "test" {
		t.Errorf("Server.Mode 应为 test,实际 %q", cfg.Server.Mode)
	}
	if !cfg.Server.SkipSetup {
		t.Error("Server.SkipSetup 应为 true(SERVER_SKIP_SETUP=true)")
	}
}

// TestLoad_ResetState 验证 WR-1 修复:多次调用 Load() 不会污染 viper 状态。
//
// 复现方案: 第一次 Load() 设 SERVER_PORT=12345,第二次不设 env,期望第二次
// 拿到的 Port 是默认值 8080(说明 Reset() 清空了第一次的 BindEnv)。
func TestLoad_ResetState(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("SM4_KEY", "anyValid==")

	// 第一次:env 注入 PORT=12345。
	t.Setenv("SERVER_PORT", "12345")
	cfg1, err := Load(context.Background())
	if err != nil {
		t.Fatalf("第一次 Load 失败: %v", err)
	}
	if cfg1.Server.Port != 12345 {
		t.Fatalf("第一次 Port 应为 12345,实际 %d", cfg1.Server.Port)
	}

	// 第二次:不再设 SERVER_PORT,期望 Reset() 已清空,走默认 8080。
	t.Setenv("SERVER_PORT", "")
	cfg2, err := Load(context.Background())
	if err != nil {
		t.Fatalf("第二次 Load 失败: %v", err)
	}
	if cfg2.Server.Port == 12345 {
		t.Errorf("第二次 Load Port 仍为 12345,viper.Reset() 未生效或 SERVER_PORT 空串被当作 12345")
	}
}