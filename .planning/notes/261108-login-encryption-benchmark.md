# FIX-09：登录加密 5 个 benchmark 实测报告

**Phase**: v1.53 Phase 139
**日期**: 2026-11-08
**REQ**: FIX-09（`REQUIREMENTS.md:51-54`）/ ROADMAP Phase 139 行（`ROADMAP.md:34`）
**结论**: 5 个 benchmark 空桩已全部补真实测量；零新依赖；`go build ./...` 通过。

---

## 1. 背景与边界

`tests/benchmark/login_encryption_bench_test.go` 原有 5 个 benchmark
（`encrypted_login` / `plaintext_login` / `nonce_check_and_store` /
`sm2_encrypt_sm4_key` / `sm4_cbc_encrypt_request`）全部是 TODO + 空循环，
`P99<350ms` / `overhead<50ms` 目标无任何实测数据支撑。本 Phase 去除空桩，
改为复用项目既有国密与防重放设施（`pkg/crypto`）做真实测量。

### ⚠️ 测量边界（如实声明，不伪造端到端数据）

- 全部 5 个基准测量的是**库函数调用 / 计算路径耗时**（`ns/op`），复用的是项目
  生产代码同款算法实现：
  - SM2：`crypto.EncryptWithSM2`（`pkg/crypto/sm2_jwt.go`）+ `crypto.GenerateKeyPair`
  - SM4-CBC：`crypto.RequestEncryptor.EncryptResponseWithKey`（`pkg/crypto/request_encryption.go`
    内部 `sm4.NewCipher` + `cipher.NewCBCEncrypter` + PKCS#7），与服务端
    `DecryptRequest` 是同一套 `tjfoc/gmsm/sm4` 实现
  - nonce：`crypto.NewDefaultNonceStorage()` = `NewShardedNonceStorage`
    （`pkg/crypto/nonce_storage.go`，256 分片分段锁），与 `RequestDecryption`
    中间件在线使用的防重放设施一致
- `encrypted_login` / `plaintext_login` 为**「同等加密 / 序列化计算路径」近似基线**：
  前者组合「SM2 加密 SM4 key + SM4-CBC 加密请求体 + nonce CheckAndStore +
  组装 `EncryptedRequest` 并 JSON 序列化」，后者仅做 JSON 反序列化 / 序列化。
  二者**不含**完整 HTTP 链路（Gin 路由 / 中间件 / 数据库校验 / SM3 密码哈希 /
  网络往返 / 双 token 签发）。因此这两个数值只能用于衡量「加解密相对明文增加的
  计算开销」，**不可当作真实 HTTP P99 上报**。
- `go test -bench` 输出的是**每操作平均耗时（ns/op）**，不是 P99 分位数。本报告用
  3 轮 `-count=3` 观测抖动，并结合 `-benchmem` 的分配量评估稳定性；P99 达标结论
  按「单次计算路径耗时相对 350ms / 50ms 阈值有 2~3 个数量级余量」给出，
  并明确它是**计算路径层面**的结论，端到端 P99 仍需真实 HTTP 压测另行验证。

---

## 2. 实测环境

```
goos: windows
goarch: amd64
pkg: github.com/xingran-next/xingran-go-backend/tests/benchmark
cpu: Intel(R) Core(TM) i9-9900 CPU @ 3.10GHz
go 1.24.5 (module go 1.24.0)
```

命令（硬性约束指定）：

```
go test -bench=. -benchmem ./tests/benchmark/
```

复现用加强命令（本报告数值来源）：

```
go test -bench=. -benchmem -benchtime=2s -count=3 ./tests/benchmark/
```

---

## 3. 实测数值表

### 3.1 主命令输出（硬性约束确切命令 `go test -bench=. -benchmem ./tests/benchmark/`）

| benchmark | ns/op | B/op | allocs/op | 换算 ms/op |
|-----------|------:|-----:|----------:|-----------:|
| `encrypted_login` | 1,799,556 | 77,070 | 1,556 | ≈ 1.80 |
| `plaintext_login` | 5,199 | 1,153 | 32 | ≈ 0.0052 |
| `nonce_check_and_store` | 578.9 | 99 | 2 | ≈ 0.00058 |
| `sm2_encrypt_sm4_key` | 2,082,304 | 74,951 | 1,537 | ≈ 2.08 |
| `sm4_cbc_encrypt_request` | 2,682 | 1,137 | 15 | ≈ 0.0027 |

### 3.2 稳健性复测（`-benchtime=2s -count=3`，3 轮 ns/op）

| benchmark | 第 1 轮 | 第 2 轮 | 第 3 轮 | 均值 | 最大值 |
|-----------|-------:|-------:|-------:|-----:|-------:|
| `encrypted_login` | 1,900,519 | 1,588,634 | 1,583,377 | ≈ 1.69 ms | 1.90 ms |
| `plaintext_login` | 4,003 | 7,063 | 5,733 | ≈ 5.6 µs | 7.1 µs |
| `nonce_check_and_store` | 765.8 | 671.1 | 649.3 | ≈ 0.70 µs | 0.77 µs |
| `sm2_encrypt_sm4_key` | 1,779,752 | 1,869,122 | 1,865,081 | ≈ 1.84 ms | 1.87 ms |
| `sm4_cbc_encrypt_request` | 2,888 | 2,559 | 2,307 | ≈ 2.6 µs | 2.9 µs |

内存分配在 3 轮间稳定（如 `sm2_encrypt_sm4_key` 恒 ≈75 KB / 1540 allocs），
说明测量可复现、无异常 GC 抖动。`encrypted_login` 的耗时几乎全部由 SM2 加密
（≈1.8 ms，占 >99%）主导，SM4-CBC 与 nonce 均为 µs 级，符合国密混合加密的
预期特征。

### 3.3 与目标的对比结论

| 目标 | 观测值（计算路径，取多轮最大值口径） | 余量 | 判定 |
|------|--------------------------------------|------|------|
| `P99 < 350ms` | 单操作最慢 ≈ 2.08 ms（`sm2_encrypt_sm4_key` 主命令轮）/ 1.90 ms（encrypted_login 稳健复测最大轮） | 约 168×（按 2.08ms） | ✅ 计算路径层面达标 |
| `overhead < 50ms` | encrypted − plaintext ≈ 1.80 ms − 0.0052 ms ≈ **1.79 ms**（主命令）/ ≈ 1.68 ms（稳健复测均值） | 约 28× | ✅ 计算路径层面达标 |

**达标结论（附边界）**：在「同等加密 / 序列化计算路径」这一近似基线下，
登录加密的计算开销 ≈ 1.7~1.8 ms，相对 `overhead<50ms` 有约 28~30 倍余量；单次
全链路加密计算 ≈ 1.8~2.1 ms，相对 `P99<350ms` 有约 165~184 倍余量。加解密
**不是**登录延迟的瓶颈，瓶颈（若存在）应在 DB / 密码哈希 / 网络等本次未测的
HTTP 全链路环节。
本结论**不等价于**真实端到端 HTTP P99 已达标——端到端指标需要独立的 HTTP
压测（含 Gin 中间件、DB、SM3 密码校验、token 签发）才能给出，本次未做、
也未伪造此类数据。

---

## 4. 改动文件清单（路径:行号）

| 文件 | 改动 |
|------|------|
| `tests/benchmark/login_encryption_bench_test.go:17-27` | 头部注释改为如实描述（删除 `.planning/OPEN-ITEMS.md B12` 悬空引用），标明本 Phase 产物位置与测量边界 |
| `tests/benchmark/login_encryption_bench_test.go:29-60` | 新增 `loginReqBody` / `sm4KeyHex` / `newBenchEncryptor` 测试夹具（复用 `pkg/crypto`） |
| `tests/benchmark/login_encryption_bench_test.go:63-119` | `encrypted_login` / `plaintext_login` 去除空循环，做组合加密路径 / 序列化路径真实测量 |
| `tests/benchmark/login_encryption_bench_test.go:122-138` | `nonce_check_and_store` 改用 `crypto.NewDefaultNonceStorage()` 真实分段锁设施 |
| `tests/benchmark/login_encryption_bench_test.go:141-156` | `sm2_encrypt_sm4_key` 改用 `crypto.EncryptWithSM2` 真实加密 |
| `tests/benchmark/login_encryption_bench_test.go:159-172` | `sm4_cbc_encrypt_request` 改用 `EncryptResponseWithKey` 真实 SM4-CBC 加密 |
| `.planning/notes/261108-login-encryption-benchmark.md` | 本报告（新建，本 Phase 合法产物） |

**零新依赖**：benchmark 仅 import 标准库 + 项目既有 `github.com/tjfoc/gmsm/sm2`
（go.mod 已有）+ 项目内 `pkg/crypto`，`go.mod` / `go.sum` 未改动。

---

## 5. 清理掉的 OPEN-ITEMS.md 悬空引用

| 位置（改前） | 原内容 | 处理 |
|--------------|--------|------|
| `tests/benchmark/login_encryption_bench_test.go:8` | `已登记 .planning/OPEN-ITEMS.md B12，勿误认作已覆盖的性能回归守护` | 已删除。新注释（`:17-27`）如实写明「5 个 benchmark 已补真实测量」+ 指向本 Phase 产物 `.planning/notes/261108-login-encryption-benchmark.md` + 声明测量边界，不指向任何临时清单 |

> 背景：用户已裁定 `OPEN-ITEMS.md` 为临时文件、里程碑收档后归档并清理所有
> 对它的「权威 / 唯一入口」引用。本 Phase 负责文件内仅此 1 处悬空引用，已清理。
> 全仓其余对 `OPEN-ITEMS.md` 的引用属 Phase 141（GOV-01）收口范围，不在本
> Phase 文件面内，未越界修改。

---

## 6. 验证记录

| 命令 | 结果 |
|------|------|
| `go test -run '^$' ./tests/benchmark/` | `ok ... [no tests to run]`（编译通过） |
| `go test -bench=. -benchmem -benchtime=1s ./tests/benchmark/` | `PASS ok ... 8.354s`，5 个 benchmark 全部产出数值 |
| `go test -bench=. -benchmem -benchtime=2s -count=3 ./tests/benchmark/` | `PASS ok ... 53.386s`，3 轮数值稳定 |
| `go build ./...` | `BUILD_ALL_OK`（全量构建通过） |
| `go build ./internal/config/ ./internal/core/` | `BUILD_OK`（FIX-12 相关包单独验证） |

---

## 7. 遗留问题

1. **端到端 P99 未测**：本次为计算路径近似基线，未做含 Gin 中间件 / DB / SM3
   密码哈希 / token 签发的真实 HTTP 登录压测。`P99<350ms` 的端到端达标结论仍需
   独立压测（建议后续用 `httptest` + 真实 router 或生产预发环境压测补齐）。
2. **`ns/op` 非 P99 分位数**：`go test -bench` 输出为均值，本报告以「均值 + 3 轮
   最大值 + 数量级余量」论证，未采集真实分位数。若需严格 P99 数值，需引入
   `benchstat` / 延迟直方图采集（本次受「零新依赖」约束未引入）。
3. **`configs/config.yaml` 为本地忽略文件**：见 FIX-12 报告说明，配置化能力的
   交付载体为 `internal/config/config.go` + `configs/config.example.yaml`。
