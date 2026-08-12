package vdi

import (
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
func loadTLSSkipVerify() bool {
	tlsSkipVerifyOnce.Do(func() {
		cfg := config.Load()
		tlsSkipVerify = cfg.VDI.TLSSkipVerify
	})
	return tlsSkipVerify
}
