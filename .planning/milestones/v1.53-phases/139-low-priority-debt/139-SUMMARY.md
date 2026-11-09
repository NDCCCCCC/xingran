---
one-liner: 对账 MV 30s 超时配置化（默认不变）、5 个登录加密 benchmark 空桩补国密实测并标注测量边界
---

# Phase 139 — 低优先级技术债（v1.53 / FIX-09 + FIX-12）

**一行摘要**：把对账 MV 启动刷新 30s 超时配置化（默认不变），并把 5 个登录加密 benchmark 空桩补成真实国密实测且如实标注测量边界。

## 交付清单

| REQ | 文件 | 内容 |
|---|---|---|
| FIX-12 | `internal/config/config.go:28-41,560-564` | 新增 `ReconciliationConfig.MVRefreshTimeoutSec`，viper `setDefaults` 默认 30s + `AutomaticEnv` 兜底 |
| FIX-12 | `internal/core/core.go:45-48,242-255` | 硬编码 `30*time.Second` 改读配置（>0 生效），加 `defaultMVRefreshTimeout` 兜底与 migration_180 历史注释 |
| FIX-12 | `configs/config.example.yaml:128-134`（受版本控制）+ 本地 `configs/config.yaml` | 配置段同步（后者被 .gitignore 忽略，不进交付） |
| FIX-09 | `tests/benchmark/login_encryption_bench_test.go`（整文件重写） | 5 个 benchmark 去空桩，复用 `pkg/crypto` 的 SM2 加密 SM4 key / SM4-CBC / 分片 nonce 设施，零新依赖 |
| FIX-09 | `.planning/notes/261108-login-encryption-benchmark.md`（新建） | 完整实测报告 |
| 清理 | `login_encryption_bench_test.go:8` | 删除 `.planning/OPEN-ITEMS.md B12` 悬空引用，改为如实描述 + 指向 Phase 产物 |

## benchmark 实测（`go test -bench=. -benchmem ./tests/benchmark/`）
| benchmark | ns/op | 换算 |
|---|---|---|
| encrypted_login | 1,799,556 | ≈1.80 ms |
| plaintext_login | 5,199 | ≈5.2 µs |
| nonce_check_and_store | 578.9 | ≈0.58 µs |
| sm2_encrypt_sm4_key | 2,082,304 | ≈2.08 ms |
| sm4_cbc_encrypt_request | 2,682 | ≈2.7 µs |

稳健性复测 `-benchtime=2s -count=3` PASS（3 轮数值稳定、内存分配恒定）。overhead≈1.79ms（28× 余量）、单次最慢≈2.08ms（168× 余量），**计算路径层面**达标。

## 验证
- `go build ./...` exit 0；`go test -run '^$' ./tests/benchmark/` 编译通过；两轮 bench 全 PASS
- `go.mod` / `go.sum` 未改

## 边界（未伪造数据）
- `encrypted_login` / `plaintext_login` 是「同等加密/序列化计算路径」近似基线，不含 Gin 路由 / 中间件 / DB / SM3 哈希 / 网络往返；`ns/op` 为均值非 P99 分位数。端到端 HTTP P99 仍需独立压测，本次未做也未伪造。
