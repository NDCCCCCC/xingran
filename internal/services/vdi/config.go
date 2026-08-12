package vdi

import (
	"context"
	"fmt"
	"sync"

	"github.com/xingran-next/xingran-go-backend/internal/config"
)

// tlsSkipVerify 缓存 VDI TLS 证书校验开关,从 config.VDI.TLSSkipVerify 读取。
// 默认 true(跳过校验),以兼容 VDI 服务器自签名证书的部署。
// 生产环境应在 configs/config.yaml 中显式将 vdi.tls_skip_verify 设为 false。
var (
	tlsSkipVerify     bool
	tlsSkipVerifyOnce sync.Once
)

// loadTLSSkipVerify 懒加载 VDI TLS 跳过校验开关
//
// 启动期配置错误是致命问题:Load() 失败时必须终止进程,不能让 VDI 模块
// 静默使用零值(默认 TLSSkipVerify=false → 严格校验,可能在某些自签证书
// 环境下导致 VDI 操作失败,掩盖真实配置错误)。
func loadTLSSkipVerify() bool {
	tlsSkipVerifyOnce.Do(func() {
		cfg, err := config.Load(context.Background())
		if err != nil {
			panic(fmt.Sprintf("[vdi] 配置加载失败: %v", err))
		}
		tlsSkipVerify = cfg.VDI.TLSSkipVerify
	})
	return tlsSkipVerify
}
