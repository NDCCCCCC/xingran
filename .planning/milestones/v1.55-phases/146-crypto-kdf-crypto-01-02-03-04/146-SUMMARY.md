---
phase: 146
plan: 01
status: complete
commit: false
executed_by: gsd-executor (exec-146)
date: 2026-11-09
---

# Phase 146 · CRYPTO KDF 契约 — 执行总结（SUMMARY）

> W1 `145 ∥ 146 ∥ 147` 并行 executor 之一；`commit: false`，**由 Lead 统一集成提交**。
> 核心命题：**根治 = 立契约 + 守卫（CRYPTO-01/02/03），不是补多块循环**。CRYPTO-04 是可选能力扩展，当前无需求，按 D4 合规跳过（**不是遗留缺陷**）。
> 已读必读件：`146-PLAN.md` 全文（296 行）、`ROADMAP.md` Phase 146 段 + 并行冲突面/Constraints/裁定/禁区、`REQUIREMENTS.md` CRYPTO-01~04 + Constraints、根因报告 §⑥ + 未决问题第 7 条、`CLAUDE.md`/`AGENTS.md`。

---

## 1. 改动清单

| 文件 | 关键位置 | 改动 |
|---|---|---|
| `internal/core/security/password.go` | `:53-97` `pbkdf2SM3` | 签名改为 `([]byte, error)`；入口加三类 `fmt.Errorf` 守卫（`keyLen<=0` / `keyLen>32` / `iterations<=0`，**无 panic**）；`:75` `append(password, salt...)` 改 `make+append`（输入序列仍为 `password‖salt‖0x00000001`）；`keyLen=32` 计算路径（`block`/`result`/`h`/`copy`/`for i:=1;i<iterations;i++` XOR 循环）**一字未改**；正常返回 `return result, nil`；更新函数注释为「单块契约」 |
| `internal/core/security/password.go` | `:128` `HashPassword` | 接 KDF error：`hash, err := pm.pbkdf2SM3(..., 32)`；`if err != nil { return "", fmt.Errorf("哈希密码失败: %w", err) }`；`32` 硬编码、`$sm3$` 格式串、DefaultPasswordConfig 均未动 |
| `internal/core/security/password.go` | `:166-180` `VerifyPassword` | base64 解码 `parts[4]` 之后加 `len(hashBytes)!=32` 守卫（返回 **error**）；KDF 调用 keyLen 由 `len(hashBytes)` 改为**字面量 `32`**；接 KDF error 兜底 `iterations<=0` 等非法存储值；`subtle.ConstantTimeCompare` 保留；`strings.Split` / `strconv.Atoi` / 解码错误处理未动 |
| `internal/core/security/password_kdf_contract_test.go` | 新增（主战场） | A1 四组独立黄金向量 hex；A2 非法输入矩阵（含 `keyLen=33/64/65`、`iterations=0/-1`）+ 合法对照；A3 往返 + `$sm3$` 恒 32 字节格式锁；A4 100k 旧哈希零迁移；A5 非 32 字节哈希 `0/16/31/33/64` 全拒绝 + 非法 iterations 兜底；A6 slice 别名修复证明 |
| `internal/core/security/password_owasp_test.go` | `:72` | **因签名变更适配**：`hashBytes := pm.pbkdf2SM3(...)` → `hashBytes, err := pm.pbkdf2SM3(...)` + `if err != nil { t.Fatalf(...) }`。其余 100k 兼容断言、600k 默认断言（`:23-52`）一字未改 |
| `internal/core/security/password_test.go` | `:3-6` import、`:71-79` `TestPasswordManager_VerifyInvalidFormat` | 加 `encoding/base64`；`invalidHashes` 追加 1 条 3 字节哈希（`"AAAA"`）+ 1 条 `base64.RawStdEncoding.EncodeToString(make([]byte,64))` 人工构造 64 字节伪造哈希；既有 5 条格式用例不动 |

**未改动**：`DefaultPasswordConfig`（`Iterations=600000`）、`$sm3$` 格式串、25 个 operlog 常量、`go.mod`/`go.sum`、任何生产调用方、任何其它 phase 文件、ROOT 无临时文件残留。

---

## 2. CRYPTO-01~04 满足情况

### ✅ CRYPTO-01 — 守卫把未定义行为变定义行为
- `pbkdf2SM3` 三类非法入参全部 `fmt.Errorf` 返回 error，无 `panic`、无「静默降级 / 返回全零 / 仅打日志」。
- `HashPassword`（`:128`）与 `VerifyPassword`（`:176`）均已接 error。
- 变异/回归证据：`TestPbkdf2SM3_RejectsIllegalInput`（A2）对 `keyLen=0/-1/33/64/65`、`iterations=0/-1` 断言 `err!=nil && dk==nil/空`；注释写明「旧实现在 `keyLen=64` 静默返回 32 真值 + 32 个 0 字节」。**`result[32:]` 全零弱密钥失败模式已消灭**。
- 全库 `pbkdf2SM3` 引用复核（grep）：`password.go:128/176`、contract 测试各点、`password_owasp_test.go:72` 全部双返回值；`excel_service.go:1168` 仅注释。**无遗漏单返回值调用点**（最易编译红处已消除）。

### ✅ CRYPTO-02 — `VerifyPassword` 非 32 字节哈希拒绝
- `len(hashBytes)==32` 守卫存在且**返回 `error`**（`password_kdf_contract_test.go` A5 + `password_test.go` 两条格式用例）。
- **D5 选型理由（返回 error 而非 false）**：① 非 32 字节哈希属「数据损坏 / 伪造」，与「密码输错」是不同安全类别，error 可观测 / 可审计 / 可告警，`false` 会把两者混同；② 3 个生产调用方均为「err 即拒绝」式，**零调用方改动**（实证：`internal/api/v1/auth.go:301` `err != nil || !ok`、`internal/core/security/local_authenticator.go:63` `if err != nil`、`internal/services/system/profile_service.go:183` `err != nil || !ok`）；③ 符合项目「service 返回 error / 禁 panic」约定；④ ROADMAP 成功标准 2 允许「显式报错 / 返回 false」二选一，error 是更严一档。
- `VerifyPassword:149` 不再把 `len(hashBytes)` 作 keyLen 喂给 KDF（改为字面量 32）；人工构造 `64/33/31/16/0` 字节哈希全部 `err!=nil && ok==false`，错误消息含「32」/「长度」；`subtle.ConstantTimeCompare` 保留（加 `len==32` 后两侧长度恒等，顺带消除「长度不等直返 0」隐性分支）。
- **免盘点**（DECISION-存量密码，D1）：全库 `sys_user.password` 写入均经 `HashPassword`，恒 32 字节 `$sm3$`，无绕过写入路径；无需迁移 / 无需数据修复脚本 / 不写生产数据。

### ✅ CRYPTO-03 — 向后兼容硬约束（逐字节不变 / 零迁移）
- `TestPbkdf2SM3_GoldenVector_KeyLen32`（A1）写死 §3.4 四组 hex（独立基线，**不在测试里现算期望值**）：
  - `1` → `a044b33929ad0b9895724d0e0539ea6a1dd738034595316c4aab98037d0cbb8e`
  - `2` → `9c96ab821f4556776dc8736a466a672b930f98155f3f0f98048f289d67f966b9`
  - `1000` → `790998c08c743cf14f3a016b2d8e755c985548f581d01d28b70a916b582be71d`
  - `600000` → `aa35b8efd5fcb94f0297ce15d31f9b89f52ef17cfbebe739aec9f4c0eb9a81aa`
  - 全绿 = 四组**逐字节一致**；含默认 600k 与 1000 长循环路径。
- 600k 默认（`DefaultPasswordConfig.Iterations=600000`）+ 100k 旧哈希（`password_owasp_test.go` 既有 + A4 补强）`Hash↔Verify` 全绿、零迁移。
- `$sm3$<iter>$<salt>$<32字节hash>` 格式、`DefaultPasswordConfig`、**25 个 operlog 常量值**、`HashPassword:110` 的 `32` 一字未改。
- A3 注释明确：「Hash↔Verify 自洽无法发现实现漂移，真正的漂移防护是 A1 黄金向量」——未把往返当唯一防线。

### ⬜ CRYPTO-04 — optional，**合规跳过（未实现）**
- **硬 gate 已过**：Task 3 全部绿、四组黄金向量逐字节一致后才判定。
- **判定依据**：① 当前**无 >32 派生需求**（`REQUIREMENTS.md:66`「当前无 SM4/会话/token 派生用途」；根因报告 §⑥ 实证 `pbkdf2SM3` 仅 2 处生产调用 + 1 测试 + 1 注释；`GenerateRandomToken` 走 `crypto/rand`、SM4 密钥来自 `SM4Cipher` 非 KDF）；② `golang.org/x/crypto/pbkdf2.Key` 用 HMAC-SM3（`hmac.New(h, password)`），与本项目 `U1=SM3(password‖salt‖INT32_BE(1))` 裸 SM3 拼接语义**很可能不对齐**（PLAN §3.5 已提示），**不对齐时禁止改 `keyLen=32` 路径迁就 oracle**（会破坏 D2 向后兼容硬约束）；③ ROADMAP line 181 禁区「只补多块循环不加守卫 = 能力扩展，不算隐患根治」。
- **结论**：按 D4 / PLAN Task 4 退出条件 ②「无需求 → 不写任何算法代码」合规跳过。`password.go` 的 `keyLen>32` 守卫仍在、`password_kdf_contract_test.go` A2 仍断言 `keyLen=33/64` 返 error。**守卫已收口、未补算法；⑥ 隐患已由 CRYPTO-01/02 守卫根治，这不是遗留缺陷**。
- 若未来产品确认需 >32 派生：先做 oracle 前置核验（对齐才用 `pbkdf2.Key(sm3.New,...)`，不对齐走自构造向量 + 前缀不变性 `DK(64)[0:32]==DK(32)`），且**守卫仍为主防线不得删除**；零新依赖（`go.mod:118` 已有 `golang.org/x/crypto v0.46.0 // indirect`）。本期未改 go.mod、未 `go get`。

---

## 3. 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt -w` 4 个 Go 文件 | 完成 |
| `go test -count=1 ./internal/core/security/` | `ok ... 48.523s`（改回后复跑 `ok ... 41.035s`，`-count=1` 禁缓存） |
| `go test -count=1 -v -run 'GoldenVector\|RejectsIllegal\|RejectsNon32\|SliceAliasFixed' ./internal/core/security/` | 4 个关键契约测试全部 `--- PASS` |
| `go build ./...` | `BUILD_OK`（全仓编译通过；不指定 `-o`，只做编译检查、无产物落盘；改回后复核仍 OK） |
| **反向变异验证** | 把 A1 `iter=1` 期望 hex 末位 `8e`→`8f`（改 1 字节）→ `go test -count=1 -run '^TestPbkdf2SM3_GoldenVector_KeyLen32$'` **FAIL**（`got=...8e want=...8f`，退出码 1，提示「禁止更新期望值，须回滚实现」）→ **改回 `8e`** → 全包复跑 **绿**（`ok ... 41.035s`，`$LASTEXITCODE=0`）。证据链「红 → 改回 → 绿」完整，同时证明期望值独立写死、实现输出仍是基线 |
| `git status --short`（只读） | 本 phase 改动 = `password.go` / `password_owasp_test.go` / `password_test.go` + 新增 `password_kdf_contract_test.go`；其余 M/?? 属 145（operlog）/147（asset）并行 executor 写集合，与本 phase 交集 = ∅ |
| 根目录临时文件残留 | `NO_ROOT_TEMP_FILES`（无 `zz_*.go` / `temp_*.go` / `test_*.go`） |

> 说明：`go test ./...` 全仓由 Lead 在 W1 集成后统一跑，本 executor 按 PLAN Task 5 只跑相关包。期间 `ops_bash` 出现瞬时 `EXECUTOR_UNAVAILABLE`（私有 runtime 副本 link count mismatch，命令未执行），改走 DSH 托管持久终端完成「改回后复跑」，随后 `ops_bash` 恢复并完成 build / status / 残留检查；未切换 shell 绕过沙箱、未启动长驻进程。

### must_haves 逐条打勾
- [x] **MH-1**：四组黄金向量逐字节一致；600k 默认 + 100k 旧哈希往返全绿；`$sm3$` 格式不变零迁移；反向变异期望值改 1 字节 → 测试必红。
- [x] **MH-2**：`keyLen<=0`/`>32`（`33/64/65`）/`iterations<=0`（`0/-1`）全部 `err!=nil && dk==nil`，无 `panic`，用 `fmt.Errorf`。
- [x] **MH-3**：`len(hashBytes)==32` 守卫返回 error（D5）；`64/33/31/16/0` 字节哈希全被拒；KDF keyLen 写字面量 32；`ConstantTimeCompare` 保留。
- [x] **MH-4**：免生产盘点 / 无迁移脚本 / 不写生产数据；`$sm3$` 格式 / `DefaultPasswordConfig` / 25 operlog 常量一字未改；仅 Known Gaps 提示人工改库脏数据。
- [x] **MH-5**：零新依赖（go.mod 未改）；`go test -count=1 ./internal/core/security/` + `go build ./...` 全绿；CRYPTO-04 走「合规跳过 + SUMMARY 登记守卫已收口、非遗留缺陷」出口。

---

## 4. 向后兼容论证

`keyLen=32` 时 `l=1, i=1`，当前实现本就是**单块正确实现**（`U1=SM3(password‖salt‖INT32_BE(1))`，后续 `U_{j+1}=SM3(U_j)` 逐次 XOR 累加，共 `iterations` 次）。本次仅在函数**入口加守卫** + `make+append` 修 slice 别名（输入字节序列仍严格是 `password‖salt‖0x00000001`），**未重写任何迭代逻辑** → `keyLen=32` 输出逐字节完全一致（A1 四组 hex 全绿实证）。因此：
- 存量 `$sm3$<iter>$<salt>$<32字节hash>` 不变；`DefaultPasswordConfig.Iterations=600000` 与 100k 旧哈希**零迁移**；
- **免盘点**（D1）：全库写入均经 `HashPassword`，恒 32 字节，无绕过路径；加 `len==32` 守卫是「输入校验收紧」，代码层不产生非 32 字节数据；
- 3 个生产调用方均为 `err!=nil||!ok` 拒绝式，`VerifyPassword` 选 error **零调用方改动**。

---

## 5. Known Gaps

- **仅一条**：「人工直接改库的历史脏数据」上线风险提示。若历史上有人绕过应用、直接改 `sys_user.password` 写入了非 32 字节哈希，加 `len==32` 守卫后这些账号会**登录失败**（输入校验收紧的预期后果，非输出漂移）。代码层不产生此类数据，按 D1 **免盘点**：本期无生产盘点、无迁移、无数据修复脚本，仅在上线说明登记该风险。
- 无其它 Known Gaps；CRYPTO-04 跳过按 D4 明确**不是遗留缺陷**（见 §2）。

---

## 6. 提交状态

**未执行 `git add` / `git commit` / `git stash` / `git checkout` / `git reset` / `git restore`**（仅执行 `git status --short` 只读查看）。`commit: false`，**由 Lead 在 W1 集成后统一提交**。

---

## 7. 并行声明

W1 `145 ∥ 146 ∥ 147` 写集合两两交集 = ∅（ROADMAP line 115）：145 = `internal/utils/operlog/*_test.go` + `scripts/e2e` + `tests/e2e` + CI；146 = `internal/core/security/password.go` + 就近 `*_test.go`；147 = `internal/services/asset/**`。本 phase **仅改** `internal/core/security/` 下 4 个文件 + 本 phase 的 `146-SUMMARY.md`，**未触碰** `internal/utils/operlog/**`（145）、`internal/services/asset/**`（147）、`internal/services/operations/**`、`xingran-react-frontend/**`、`internal/api/router.go`、`.golangci.yml`、`.github/workflows/ci.yml`、`.planning/STATE.md`/`ROADMAP.md`/`REQUIREMENTS.md`/根因笔记、任何其它 phase 的 PLAN/SUMMARY、ROOT 临时测试文件。`git status --short` 中出现的其它改动均为并行 executor 成果，与本 phase 无文件交集。

---

## 8. 完成条件

6 个 Task 全部完成（Task 4 为「合规跳过」且已登记）；§3 五条 must_haves 全部打勾，附 `go test` / `go build` / 反向变异验证输出；`146-SUMMARY.md` 落盘、明确「未 git commit」；代码实际状态与本文自洽（不伪造闭环）。
