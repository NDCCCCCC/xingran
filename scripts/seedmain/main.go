// 补 seed: 直接调 internal/core/db.initData() 跑 admin/menu/role seed。
// 绕过 AutoMigrate hang,跑完后 backend 即可 :9000 LISTENING。
//   go run ./scripts/seedmain
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

func main() {
	applogger.Init(applogger.DefaultConfig())
	loadEnvFile(".env")
	if v := os.Getenv("SM4_KEY"); v == "" {
		if data, err := os.ReadFile(".env"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "SM4_KEY=") {
					os.Setenv("SM4_KEY", strings.TrimPrefix(line, "SM4_KEY="))
					break
				}
			}
		}
	}
	cfg, err := config.Load(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Load:", err)
		os.Exit(1)
	}
	if !cfg.Server.SkipSetup {
		cfg.Server.SkipSetup = true
	}

	d, err := db.NewDatabase(&cfg.Database)
	if err != nil {
		fmt.Fprintln(os.Stderr, "NewDatabase:", err)
		os.Exit(1)
	}

	if err := d.InitData(); err != nil {
		fmt.Fprintln(os.Stderr, "InitData:", err)
		os.Exit(1)
	}
	fmt.Println("✅ InitData OK")
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}