---
phase: 146
plan: 01
wave: 1
depends_on: []
files_modified:
  - internal/core/security/password.go
  - internal/core/security/password_test.go
  - internal/core/security/password_owasp_test.go
  - internal/core/security/password_kdf_contract_test.go
  - .planning/phases/146-crypto-kdf-crypto-01-02-03-04/146-SUMMARY.md
autonomous: true
commit: false
---

# Phase 146 · CRYPTO KDF 契约 — 执行计划（PLAN 01）

> 本计划由 `gsd-planner` 产出，`autonomous: true`，供 W1 并行 executor **直接执行**。
> `commit: false`：本 phase 与 `145 ∥ 146 ∥ 147` 并行，**禁止任何 git 写操作**，由 Lead 统一提交。
> 输出语言：简体中文；技术标识符 / 路径 / 命令保留英文。

---

## 1. 必读顺序（executor 开工前按序读完，禁止跳读）

| # | 文件 | 读什么 / 为什么 |
|---|------|----------------|
| 1 | `.planning/STATE.md` | v1.55 active（line 3-9、`## Current Position` line 99-104）；确认本期是「LOCKED 残余根治」，不是新功能 |
| 2 | `.planning/ROADMAP.md` § `#### Phase 146`（line 41-51） | 4 个 REQ 映射、**5 条成功标准**、实现要点、文件面 |
| 2b | `.planning/ROADMAP.md` line 113-116、150-161、163-181 | W1 `145∥146∥147` 并行冲突面（写集合互斥）、Constraints 第 6/8/9 条、5 项裁定表、禁区（line 181「⑥ 只补多块循环不加守卫 = 能力扩展」） |
| 3 | `.planning/REQUIREMENTS.md` § `### CRYPTO`（line 61-66）+ Constraints（line 98-106） | CRYPTO-01/02/03/04 全文；DECISION-存量密码（line 14）「免盘点，直接上守卫」；向后兼容 / 零新依赖 |
| 4 | `.planning/notes/261109-v154-locked-root-cause-analysis.md` § ⑥（line 136-149）+ line 179-184、188-196、214 | 根因定性、根治定义（F+G）、方案对比、实现要点与向量、向后兼容论证、未决问题第 7 条、兼容基线行号 |
| 5 | `CLAUDE.md` / `AGENTS.md` | 项目硬约定：`service 返回 error` / **禁 panic** / 改 Go 后必须 `go build ./...` / Go 单测就近 `*_test.go` / `out/` 产物 / operlog 25 常量锁 / 提交前需人工确认 |
| 6 | `internal/core/security/password.go`（全文 199 行，重点 line 69-97、102-120、124-153） | 待改实现（逐行号见 §3） |
| 7 | `internal/core/security/password_test.go`、`password_owasp_test.go` | 既有测试基线；**`password_owasp_test.go:72` 单返回值调用必须随签名适配** |

---

## 2. 目标（Objective）

把 `pbkdf2SM3` 的**未定义行为变成定义行为**：非法 `keyLen` / `iterations` 显式返回 `error`（不再静默产出「后半段全零」的低熵弱密钥），`VerifyPassword` 显式拒绝非 32 字节存储哈希；用**独立于实现的黄金向量**锁死 `keyLen=32` 输出逐字节不变，从而保证存量 600k / 100k `$sm3$` 哈希零迁移、免生产盘点。

**核心命题（写进 SUMMARY）**：根治 = 立契约 + 守卫（CRYPTO-01/02/03），**不是**补多块循环。CRYPTO-04 是可选能力扩展，当前无需求；只补循环不加守卫只算缓解（ROADMAP line 181 禁区）。

---

## 3. 已确认事实（Lead 已调研，executor **不要重新调研**）

### 3.1 缺陷机理（`internal/core/security/password.go`）

- `pbkdf2SM3`（line 69-97）：`block := make([]byte,32)`（:71）、`result := make([]byte,keyLen)`（:72）、`blockData := append(password, salt...)` + 硬编码 `INT32_BE(1)`（:75-76，**只有 T_1**）、`copy(result, block)` 只拷 32（:82）、XOR 循环 `j < keyLen && j < len(block)` 恒 ≤32（:91）→ **`keyLen>32` 时不 panic / 不报错 / 不截断，而是 `result[32:]` 保持全 0**（静默弱密钥，最危险失败模式）。
- 附带 `:75` `append(password, salt...)` **slice 别名隐患**：当 `cap(password) >= len(password)+len(salt)` 时 append 会原地改写调用方底层数组。
- **裁定事实有误（已推翻）**：`HashPassword:110` 写死 32，但 **`VerifyPassword:149` 用 `len(hashBytes)`**（base64 解码 `parts[4]`）作 keyLen → 非 32 字节存储哈希即触发缺陷；`:152` `subtle.ConstantTimeCompare` 长度不等直返 0。`$sm3$` 格式（`:118`）**未编码 keyLen**。
- 全库 `pbkdf2SM3` 仅 4 处引用：`password.go:110`、`password.go:149`、`password_owasp_test.go:72`、`internal/services/operations/excel_service.go:1168`（注释）。**无 SM4 / 会话 / token 派生用途**；`GenerateRandomToken`（`password.go:185-199`）走 `crypto/rand`；SM4 密钥来自 `SM4Cipher`（`core.go:174-189`）非 KDF。

### 3.2 用户 2026-11-09 裁定「免盘点，直接上守卫」

- 全库 `sys_user.password` 写入均经 `HashPassword`：`user_service.go:841`、`user_sync_service.go:511/646/1143`（批量 hashMap 用 `"123456"`）、`excel_service.go:1172` → 恒 32 字节 `$sm3$`，**无绕过写入路径** → 无需生产盘点、无需迁移、无需数据修复脚本。
- 仅在上线说明 / SUMMARY 的 Known Gaps 提示「**人工直接改库的历史脏数据**」风险（代码层不产生，加守卫后这类账号会登录失败，属输入校验收紧的预期后果）。

### 3.3 向后兼容硬约束（最关键）

- `keyLen=32` 时 `l=1, i=1`，当前实现**本就是单块正确实现**：`U1 = SM3(password||salt||INT32_BE(1))`，后续 `U_{j+1}=SM3(U_j)` 逐次 XOR 累加，共 `iterations` 次。**不要重写迭代逻辑**。
- 因此入口加守卫 + `make+append` 修别名后，只要 **输入字节序列不变**（仍是 `password||salt||0x00000001`），`keyLen=32` 输出**逐字节完全一致**。
- 存量 `$sm3$<iter>$<salt>$<32字节hash>` 不变；`DefaultPasswordConfig.Iterations=600000`（`password.go:33-36`）与 100k 旧哈希（`password_owasp_test.go:61-97`）无需迁移；`$sm3$` 格式、`DefaultPasswordConfig`、**25 个 operlog 常量值**均不变。

### 3.4 已采集的黄金向量基线（planner 用当前实现实跑，executor 直接写死进测试）

> 固定输入：`password = []byte("GSD-Phase146-GoldenVector")`，`salt = 0x000102...0f`（16 字节），`keyLen = 32`。
> 采集命令：`go test -count=1 -run '^TestZZPhase146GoldenBaseline$' -v ./internal/core/security/`（临时采集文件已删除，不在交付文件面）。

| iterations | 期望 hex（`keyLen=32`） |
|---|---|
| 1 | `a044b33929ad0b9895724d0e0539ea6a1dd738034595316c4aab98037d0cbb8e` |
| 2 | `9c96ab821f4556776dc8736a466a672b930f98155f3f0f98048f289d67f966b9` |
| 1000 | `790998c08c743cf14f3a016b2d8e755c985548f581d01d28b70a916b582be71d` |
| 600000 | `aa35b8efd5fcb94f0297ce15d31f9b89f52ef17cfbebe739aec9f4c0eb9a81aa` |

> ⚠️ 这 4 个值来自**改动前**的实现，是独立基线。若 Task 1 改完后任一不一致 → 说明 `keyLen=32` 计算路径被意外改变，**立即回滚该函数改动**，不得「顺手更新期望值」（更新期望值等于自毁契约）。

### 3.5 依赖与调用方实证

- `go.mod:118` 已有 `golang.org/x/crypto v0.46.0 // indirect` → CRYPTO-04 的 oracle **零新依赖**（符合硬约束；若某环境缺该间接依赖则直接走 §Task 4 的自构造向量路径，**不 `go get`**）。
- `VerifyPassword` 签名 `(bool, error)` 不变；3 个生产调用方均为「err 即拒绝」式，**Task 2 选 error 零调用方改动**：
  - `internal/api/v1/auth.go:301` → `if ok, err := pwdManager.VerifyPassword(password, user.Password); err != nil || !ok { ... }`
  - `internal/core/security/local_authenticator.go:63` → `if ok, err := ...; err != nil { ... }`
  - `internal/services/system/profile_service.go:183` → `if ok, err := ...; err != nil || !ok { ... }`
- 既有测试 `password_test.go`（含 `:75-76` 格式错误用例）、`password_owasp_test.go`（100k 旧哈希兼容）保持全绿。

---

## 4. Lead 裁决表（不可推翻，executor 照办）

| # | 议题 | 裁决 | 落点 |
|---|------|------|------|
| D1 | 存量密码盘点 | **免盘点，直接上守卫**（DECISION-存量密码）。全库写入均经 `HashPassword`，恒 32 字节，无绕过路径；无需迁移 / 无需数据修复脚本 / 不写生产数据 | Task 2 + Task 6 Known Gaps 仅「上线说明提示人工改库脏数据」 |
| D2 | 向后兼容 | **`keyLen=32` 输出逐字节不变**为硬约束；不重写迭代逻辑；不改 `$sm3$` 格式 / `DefaultPasswordConfig` / 25 个 operlog 常量 | Task 1 + Task 3 黄金向量 |
| D3 | 主防线 | **守卫（CRYPTO-01/02）为主防线**；补多块循环（CRYPTO-04）是能力扩展，**不叠加守卫只算缓解**，禁区 | Task 1/2 必做；Task 4 optional |
| D4 | CRYPTO-04 | **optional 但纳入 PLAN 任务**；当前无 >32 派生需求；不做时在 SUMMARY 登记「守卫已收口、未补算法」并说明**这不是遗留缺陷** | Task 4 + Task 6 |
| D5 | `VerifyPassword` 非 32 字节处理 | **返回 `error`**（非 false），理由见 Task 2；调用方零改动 | Task 2 |
| D6 | 提交 | **`commit: false`**；并行 executor 禁一切 git 写操作，Lead 统一提交 | Task 6 + §7 禁止 |

---

## 5. 任务（XML，按序执行；Task 3 必须先绿再动 Task 4）

<tasks>

<task id="1" name="pbkdf2SM3 立契约：签名带 error + 三类守卫 + 修 slice 别名（CRYPTO-01）" wave="1">
  <files>internal/core/security/password.go</files>
  <read_first>password.go:53-97</read_first>
  <action>
1. 改签名：`func (pm *PasswordManager) pbkdf2SM3(password, salt []byte, iterations, keyLen int) ([]byte, error)`。
2. **函数入口最先**加三类守卫（项目禁 panic，**只用 `fmt.Errorf` 返回 error**，不得 panic / 不得静默返回）：
   - `if keyLen <= 0 { return nil, fmt.Errorf("pbkdf2SM3: 非法 keyLen=%d，须为正数且不超过 SM3 摘要长度 32", keyLen) }`
   - `if keyLen > 32 { return nil, fmt.Errorf("pbkdf2SM3: keyLen=%d 超过 32，当前实现仅支持单块（INT32_BE(1)），拒绝静默弱密钥", keyLen) }`
   - `if iterations <= 0 { return nil, fmt.Errorf("pbkdf2SM3: 非法 iterations=%d，须为正数", iterations) }`
   - 错误消息中文、含具体入参值即可；**不要**改守卫之外的任何计算语义。
3. 修 `password.go:75` slice 别名：把 `blockData := append(password, salt...)` 改为 `make + append`（先 `make([]byte, 0, len(password)+len(salt)+4)`，再依次 append `password...` / `salt...` / `0x00,0x00,0x00,0x01`）。**输入字节序列必须仍是 `password || salt || 0x00000001`**（与改前一致，否则 SM3 输出漂移）。
4. **保持 `keyLen=32` 计算路径逐字节不变**：`block := make([]byte,32)` / `result := make([]byte,keyLen)` / `h := sm3.New()` / `h.Write(blockData)` / `copy(block,h.Sum(nil))` / `copy(result,block)` / `for i:=1; i<iterations; i++` 的 `h.Reset()`+`h.Write(block)`+`block=h.Sum(nil)`+XOR 循环 **一字不改**；当前实现本就是 `i=1` 单块正确实现，禁止「顺手重写成标准 PBKDF2 / HMAC 语义」。
5. 正常返回改为 `return result, nil`。
6. 更新函数注释（`password.go:53-68`）：明确「仅支持 `1<=keyLen<=32` 单块；`keyLen>32` / `keyLen<=0` / `iterations<=0` 返回 error；`keyLen=32` 与历史实现逐字节一致」；保留「未实现多块循环」的事实陈述（CRYPTO-04 未做时不撒谎）。
  </action>
  <verify>
`go build ./internal/core/security/` 此刻会因 `HashPassword`/`VerifyPassword` 未接 error 而报编译错（预期，Task 2 修）；单独静态复核：守卫在入口、`keyLen=32` 计算块字节级未变、`make+append` 输入序列未变。
  </verify>
  <done>签名带 `([]byte, error)`；三类守卫均为 `fmt.Errorf` 无 panic；`:75` 改 `make+append`；`keyLen=32` 计算路径未重写。</done>
</task>

<task id="2" name="HashPassword / VerifyPassword 接 error + len(hashBytes)==32 守卫（CRYPTO-01/02）" wave="1">
  <files>internal/core/security/password.go</files>
  <read_first>password.go:99-153；auth.go:301；local_authenticator.go:63；profile_service.go:183</read_first>
  <action>
1. `HashPassword:110`：`hash, err := pm.pbkdf2SM3([]byte(password), salt, pm.config.Iterations, 32)`；`if err != nil { return "", fmt.Errorf("哈希密码失败: %w", err) }`。`32` 保持硬编码不变，`$sm3$` 格式串（`:118`）与 `DefaultPasswordConfig` 一字不动。
2. `VerifyPassword:143-152`：在 `DecodeString(parts[4])` 成功之后、`pbkdf2SM3` 之前，**显式校验长度**：
   `if len(hashBytes) != 32 { return false, fmt.Errorf("密码哈希长度非法: got %d bytes, want 32（$sm3$ 格式恒 32 字节）", len(hashBytes)) }`
3. **选型理由（写进 SUMMARY）**：选 **返回 `error`** 而非 `false` —— ① 非 32 字节哈希属「数据损坏 / 伪造」，与「密码输错」是不同安全类别，error 可观测 / 可审计 / 可告警，`false` 会把两者混同；② 3 个生产调用方均为 `err != nil || !ok` 拒绝式（见 §3.5），零调用方改动；③ 符合项目「service 返回 error / 禁 panic」约定；④ ROADMAP 成功标准 2 允许「显式报错 / 返回 false」二选一，error 是更严一档。
4. `comparisonHash, err := pm.pbkdf2SM3([]byte(password), salt, iterations, 32)` —— **注意 keyLen 由 `len(hashBytes)` 改为字面量 `32`**（因上一步已保证 `len==32`）；`if err != nil { return false, fmt.Errorf("重算密码哈希失败: %w", err) }`（兜住 `iterations<=0` 等非法存储值）。
5. **`subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1` 保持不变**（`:152`）；加 `len==32` 守卫后两侧长度恒等，顺带消除「长度不等直返 0」的隐性分支。
6. 不要改 `strings.Split` 解析（`:126-129`）、`strconv.Atoi`（`:132`）、base64 解码错误处理（`:138-146`）——既有 `password_test.go:71-84` 的 5 条格式错误用例仍须返回 error。
  </action>
  <verify>
`go build ./internal/core/security/` 通过；`go test -count=1 ./internal/core/security/` 此刻 `password_owasp_test.go:72` 会因未适配而编译失败（预期，Task 3 修）。
  </verify>
  <done>`HashPassword` 接 error；`VerifyPassword` 有 `len==32` 守卫且返回 error；KDF 调用 keyLen 写字面量 32；`ConstantTimeCompare` 保留；`$sm3$` / `DefaultPasswordConfig` 未动。</done>
</task>

<task id="3" name="黄金向量测试先行 + 既有测试适配 + 非法输入矩阵（CRYPTO-03）" wave="1">
  <files>
    internal/core/security/password_kdf_contract_test.go（新增，主战场）
    internal/core/security/password_owasp_test.go（适配签名，必须）
    internal/core/security/password_test.go（补 1 条格式用例）
  </files>
  <read_first>password_owasp_test.go:61-97；password_test.go:66-85；§3.4 基线表</read_first>
  <action>
**A. 新增 `password_kdf_contract_test.go`，`package security`（同包，可访问未导出 `pbkdf2SM3`），`import ("bytes"; "encoding/hex"; "fmt"; "strings"; "testing"; "encoding/base64")` 按需。**

A1. `TestPbkdf2SM3_GoldenVector_KeyLen32`（**重构前后逐字节锁**）：
- 常量化 `goldenPassword = []byte("GSD-Phase146-GoldenVector")`、`goldenSalt = [16]byte{0x00,0x01,...,0x0f}`。
- 表驱动 4 行，直接写死 §3.4 的 4 个 hex（1 / 2 / 1000 / 600000），**不得在测试里调用「当前实现现算」当期望值**（那就退化成自洽，无法发现漂移）。
- 每行：`dk, err := pm.pbkdf2SM3(goldenPassword, goldenSalt[:], iter, 32)`；断言 `err == nil`；`hex.EncodeToString(dk) == wantHex`，不一致 `t.Fatalf("iter=%d: keyLen=32 输出漂移 got=%s want=%s —— 禁止更新期望值，须回滚实现", ...)`。
- 采样含 `iterations=600000`（默认基线）与 `1000`，确保长循环路径也被锁。

A2. `TestPbkdf2SM3_RejectsIllegalInput`（**非法输入矩阵，断言必返 error、绝不返回全零弱结果**）：
- 表驱动：`keyLen=0, -1, 33, 64, 65`（配合合法 iterations=1/2）；`iterations=0, -1`（配合 keyLen=32）；可再加 `keyLen=32 + iterations=1` 作为「合法对照组」。
- 每个非法用例：`dk, err := pm.pbkdf2SM3(...)`；断言 `err != nil`；断言 `dk == nil`（或 `len(dk)==0`）——**关键**：不得再出现 `result[32:]` 全零的非 nil 切片。
- 追加显式回归断言（对齐 ROADMAP 成功标准 1）：若绕过守卫直接构造 `keyLen=64`，旧实现会产出 `dk[32:]` 全 0；本测试用「`err!=nil && dk==nil`」证明该失败模式已消灭。可在注释里写明「旧实现此处静默返回 32 真值 + 32 个 0 字节」。

A3. `TestPasswordManager_HashVerifyRoundTrip`（往返，**但不作为唯一防线**）：
- 默认 manager（`600000`，慢但 <1s）与自定义小迭代（`1000`）各跑一轮 `HashPassword` → `VerifyPassword` 正确密码 true / 错误密码 false / err 均 nil。
- 解析 `$sm3$` 5 段，断言 `parts[1]=="sm3"`、`parts[2]` 为对应迭代数、base64 解出的 salt=16 字节 / hash=**32 字节**（锁 `$sm3$` 恒 32 字节格式契约）。
- 注释明确：「Hash↔Verify 自洽无法发现实现漂移，真正的漂移防护是 A1 黄金向量」。

A4. `TestPasswordManager_Legacy100k_BackwardCompat`（100k 旧哈希零迁移，补强 `password_owasp_test.go` 既有同名测试）：
- 用 `pm.pbkdf2SM3(..., 100000, 32)` 构造 `$sm3$100000$<salt>$<hash>` 字符串（**注意适配后的双返回值，err 必 nil**），`VerifyPassword` 正确明文 true / 错误明文 false / err nil。
- 断言 `len(hashBytes)==32` 且 `strings.Split` 5 段。

A5. `TestVerifyPassword_RejectsNon32ByteHash`（CRYPTO-02 验收）：
- 构造 `$sm3$1000$<b64(16字节salt)>$<b64(N字节hash)>`，N 取 `0, 16, 31, 33, 64`（含人工构造 64 字节伪造哈希，对齐 ROADMAP 成功标准 2 的「人工构造 64 字节哈希的登录用例被拒绝」）。
- 每次 `ok, err := pm.VerifyPassword("...", forged)`；断言 **`err != nil` 且 `ok == false`**；断言错误消息含「32」或「长度」关键词。
- 再补一条 `iterations` 非法的存储值（`$sm3$0$...` / `$sm3$-1$...`）断言 `err != nil`（验证 Task 2 第 4 步的 KDF error 兜底链路）。
- 断言「非 32 字节哈希绝不再被当作 keyLen 喂给 KDF」。

A6.（可选但推荐）`TestPbkdf2SM3_SliceAliasFixed`：构造 `password` 底层数组 `cap` 富余（如 `make([]byte, 8, 64)` 填值后切片），调用 `pbkdf2SM3(password, salt, 1, 32)`，断言调用后 `password` 底层数组未被 salt 覆写（旧 `:75` `append` 会污染）。证明 `make+append` 修复生效且**不改变输出**（与 A1 `iter=1` 基线一致）。

**B. 适配 `password_owasp_test.go:72`**：`hashBytes := pm.pbkdf2SM3(...)` → `hashBytes, err := pm.pbkdf2SM3([]byte(plaintext), salt, legacyIterations, 32)` + `if err != nil { t.Fatalf(...) }`。其余 100k 兼容断言、600k 默认断言（`:23-52`）一字不改。

**C. `password_test.go:66-85` `TestPasswordManager_VerifyInvalidFormat`**：在 `invalidHashes` 追加 1 条非 32 字节哈希（如 `"$sm3$1000$AAECAwQFBgcICQoLDA0ODw$AAAA"`（hash 仅 3 字节）与一条 64 字节伪造哈希），保持「遍历断言 `err == nil` 即 `t.Errorf`」的既有风格；既有 5 条格式用例保持不动。

**D. 反向变异验证（必须现场做一次，见 Task 5）**：临时把 A1 任一 `wantHex` 改 1 个字符 → `go test` 必须红；验证后**改回**。
  </action>
  <verify>
`go test -count=1 -v ./internal/core/security/` 全绿；A1 四个 hex 与 §3.4 完全一致；A2/A5 非法用例全部 `err != nil`；A3/A4 往返与 100k 兼容全绿；变异验证红。
  </verify>
  <done>新增 contract 测试含「独立黄金向量 + 非法矩阵 + 往返 + 100k 兼容 + 非 32 拒绝」；`password_owasp_test.go:72` 双返回值适配；`password_test.go` 格式用例扩充；未出现「仅靠 Hash↔Verify 自洽」。</done>
</task>

<task id="4" name="CRYPTO-04 完整多块循环（OPTIONAL，可不做；先 Task 3 绿再决定）" wave="1" optional="true">
  <files>internal/core/security/password.go；internal/core/security/password_kdf_contract_test.go</files>
  <read_first>ROOT CAUSE NOTE §⑥ line 143-147；REQUIREMENTS.md:66；ROADMAP.md:50、181</read_first>
  <action>
> **进入条件（硬 gate）**：Task 3 全部绿、§3.4 四个黄金向量逐字节一致后才可动本任务。
> **退出条件（合规，二选一）**：① 产品确认未来需 >32 派生 → 实现下列要点；② 无需求（**当前默认**）→ **不写任何算法代码**，直接在 Task 6 SUMMARY 登记「守卫已收口、未补算法；CRYPTO-04 optional 未实现，按 D4 这不是遗留缺陷」，并在 SUMMARY 写明「⑥ 隐患已由 CRYPTO-01/02 守卫根治」。**不得为凑任务而改 `keyLen=32` 路径**。

若走 ①，实现要点（**守卫仍为主防线，不得删除 Task 1 的三类守卫**；仅把 `keyLen>32` 的「拒绝」放宽为「按多块计算」，但 `keyLen<=0` / `iterations<=0` 守卫必须保留）：

1. **oracle 前置核验（先做，防语义坑）**：`golang.org/x/crypto/pbkdf2.Key` 内部用 `hmac.New(h, password)`（PRF=HMAC-SM3），而本实现 U1=`SM3(password||salt||INT32_BE(1))`（裸哈希拼接，非 HMAC）→ **`pbkdf2.Key(sm3.New,...)` 的输出与本实现在 `keyLen=32` 上很可能不对齐**。动手前先写一个临时核验：`pbkdf2.Key(goldenPassword, goldenSalt[:], iter, 32, sm3.New)` vs §3.4 基线。
   - 若**对齐** → 按用户要求用 `pbkdf2.Key(sm3.New,...)` 比对 `keyLen=33/64/65`。
   - 若**不对齐**（预期）→ **不要强行改 `keyLen=32` 语义去迁就 oracle**（会破坏向后兼容硬约束 D2！）；改走「自构造向量 + 结构性断言」路径（见 3/4），并在 SUMMARY 记录该语义差异。零新依赖仍成立（`go.mod:118` 已有 `golang.org/x/crypto`）。
2. 多块循环实现要点：`for i:=1; i<=l; i++`（`l = ceil(keyLen/32)`），每块计数器用 **`INT32_BE(i)` 大端 4 字节**（替换硬编码 `0x00000001`）；每块**独立 `sm3.New()` / `Reset()`**，块内 U1=`SM3(password||salt||INT32_BE(i))`、U_{j+1}=`SM3(U_j)` 迭代 `iterations` 次并 XOR 累加；`dk = append(dk, block...)` 后 **`dk[:keyLen]` 截断**；沿用 Task 1 的 `make+append` 避免别名。
3. **结构性向量（不依赖外部 oracle 的主断言）**：`keyLen=1/32/33/64/65` 全部产出 `len(dk)==keyLen`；**前缀不变性** `DK(64)[0:32] == DK(32)`（§3.4 基线，多块 PBKDF2 核心性质）、`DK(65)[0:32] == DK(32)`、`DK(33)[0:32] == DK(32)`；`DK(33)[32]` 非零（证明第 2 块真参与，非补零）。
4. **`keyLen=32` 黄金向量四行必须仍全绿**（回归 D2）；A2 非法矩阵中 `keyLen=0/-1` 与 `iterations=0/-1` 仍须 `err!=nil`；`keyLen=33/64` 从「error」改为「成功产出正确长度」——**同步更新 A2 期望，并在注释写明「守卫已放宽为能力扩展，但非法入参守卫仍在」**。
5. 不引任何新依赖；不启动长驻进程；不改 `$sm3$` 格式（存储哈希仍恒 32 字节，`HashPassword` 仍写死 32）。
  </action>
  <verify>
若实现：`go test -count=1 ./internal/core/security/` 全绿 + 四个黄金向量仍一致 + 前缀不变性断言全绿。若不做：确认 `password.go` 的 `keyLen>32` 守卫仍在、`password_kdf_contract_test.go` 的 A2 仍断言 `keyLen=33/64` 返 error。
  </verify>
  <done>要么「实现 + oracle 核验记录 + 结构性向量全绿」，要么「合规跳过 + SUMMARY 明确登记非遗留缺陷」；**禁止**只补循环却删守卫。</done>
</task>

<task id="5" name="自验证（只跑相关包，不跑全仓）+ 反向变异验证" wave="1">
  <files>无（只读执行）</files>
  <action>
1. **包级测试**（AGENTS.md / ROADMAP 成功标准 5）：`go test -count=1 ./internal/core/security/`（`-count=1` 禁缓存；可加 `-v` 观察 A1/A2/A5）。
2. **改 Go 后 build 强制项**（AGENTS.md「Compilation & Build Verification」）：`go build ./...`（全仓编译，**不产出仓库根 exe**，产物如需落盘一律 `out/`；本任务不指定 `-o`，`go build ./...` 只做编译检查不落盘）。
   - 说明：`go test ./...` 全仓由 Lead 在 W1 集成后统一跑；本 executor **只跑相关包**，避免与 145/147 并行 executor 争用 / 误判。
3. **反向变异验证（必须）**：临时把 `password_kdf_contract_test.go` A1 表里的 `wantHex` 改 1 个字符（如末位 `...8e` → `...8f`）→ 跑 `go test -count=1 -run TestPbkdf2SM3_GoldenVector_KeyLen32 ./internal/core/security/` **必须 FAIL**；确认红之后**改回原值**并复跑全绿。在 SUMMARY 登记「变异验证：红 → 改回 → 绿」。
4. 复核无临时残留文件（`zz_*` / `temp_*` / `test_*.go` 等，AGENTS.md「Temporary Files Cleanup」）；`go build ./...` 若因根目录临时 `main` 冲突报错，先清理再复跑。
5. 核对文件面：仅 §frontmatter `files_modified` 五个文件被改动；`git status --short` **只读查看**（禁止 `git add/commit/stash/checkout`）。
  </action>
  <verify>
命令输出全绿；变异验证确为红；`git status --short` 里无 `password.go` 之外越界改动；无临时文件残留。
  </verify>
  <done>`go test -count=1 ./internal/core/security/` PASS + `go build ./...` PASS + 变异红绿证据 + 文件面核对记录。</done>
</task>

<task id="6" name="写 146-SUMMARY.md（无 git 写操作）" wave="1">
  <files>.planning/phases/146-crypto-kdf-crypto-01-02-03-04/146-SUMMARY.md（新增）</files>
  <action>
新增 SUMMARY，至少含以下小节（简体中文）：
1. **改动清单**：逐文件（`password.go` / `password_test.go` / `password_owasp_test.go` / 新增 `password_kdf_contract_test.go`）+ 关键行号 + 一句话改动；注明 `password_owasp_test.go:72` 因签名变更适配。
2. **4 个 REQ 满足情况**（逐条 `✅ 满足 / optional 未实现` + 证据）：
   - `CRYPTO-01` ✅ 三类守卫 `fmt.Errorf` 无 panic / `HashPassword`+`VerifyPassword` 接 error / 变异测试证明 `keyLen=33/64` 不再全零弱密钥；
   - `CRYPTO-02` ✅ `len(hashBytes)==32` 守卫返回 error（附 D5 选型理由 + 3 个调用方零改动实证）/ 人工构造 64 字节哈希被拒绝；
   - `CRYPTO-03` ✅ 四个黄金向量（1/2/1000/600000）逐字节一致 / 600k 默认 + 100k 旧哈希往返全绿 / `$sm3$` 格式 · `DefaultPasswordConfig` · 25 operlog 常量未变 / 零迁移；
   - `CRYPTO-04` ⬜ optional：**是否实现 + 理由**（实现则附 oracle 前置核验结论与结构性向量结果；未实现则写「守卫已收口、未补算法；当前无 >32 派生需求；按 D4/ROADMAP line 181 这不是遗留缺陷」）。
3. **验证结果**：`go test -count=1 ./internal/core/security/` / `go build ./...` 输出摘要 + 反向变异验证「改 1 字节 → 红 → 改回 → 绿」+ 文件面核对。
4. **向后兼容论证**：`keyLen=32` 时 `l=1,i=1` 与改前逐字节一致 → 免盘点（D1）、零迁移；仅 Known Gaps 提示「人工直接改库的历史脏数据」上线风险（代码层不产生）。
5. **Known Gaps**：仅登记「人工改库脏数据提示」一条；明确无生产盘点 / 无迁移 / 无数据修复脚本。
6. **提交状态**：明确写「**未执行 `git add` / `git commit` / `git stash` / `git checkout`**；`commit: false`，由 Lead 在 W1 集成后统一提交」。
7. **并行声明**：W1 `145∥146∥147` 写集合互斥，本 phase 未触碰其它 phase 文件面（见 §7）。
  </action>
  <verify>SUMMARY 六个小节齐全；4 个 REQ 逐条有结论；明确「未 git commit」；CRYPTO-04 结论与 Task 4 实际选择一致。</verify>
  <done>`146-SUMMARY.md` 落盘且内容自洽，与代码实际状态一致（不伪造闭环）。</done>
</task>

</tasks>

---

## 6. 验证标准（must_haves — 对应 ROADMAP Phase 146 的 5 条成功标准）

> executor 收工前逐条打勾；Lead 集成验收以此为准。

- [ ] **MH-1（成功标准 1 + 3｜`keyLen=32` 黄金向量逐字节一致 / 零迁移）**：`TestPbkdf2SM3_GoldenVector_KeyLen32` 四个 hex（`1/2/1000/600000`）与 §3.4 完全一致；`HashPassword` 默认 600k 与 `password_owasp_test.go` 的 100k 旧哈希 `Hash↔Verify` 全绿；`$sm3$<iter>$<salt>$<32字节hash>` 存量格式不变，**无需迁移**。反向变异验证：期望值改 1 字节 → 测试必红。
- [ ] **MH-2（成功标准 1｜非法输入显式 error，不静默弱结果）**：`keyLen<=0` / `keyLen>32`（`33/64/65` 采样）/ `iterations<=0`（`0/-1`）全部 `err != nil` 且 `dk == nil`，**绝不**返回 `result[32:]` 全零的非 nil 切片；实现用 `fmt.Errorf` 无 `panic`。
- [ ] **MH-3（成功标准 2｜`VerifyPassword` 非 32 字节哈希拒绝）**：`len(hashBytes)==32` 守卫存在且返回 `error`（D5）；人工构造 `64/33/31/16/0` 字节哈希的登录用例全部 `err!=nil && ok==false`；`VerifyPassword:149` 不再把 `len(hashBytes)` 作 keyLen 喂给 KDF；`subtle.ConstantTimeCompare` 保留。
- [ ] **MH-4（成功标准 3 + 5｜免盘点 / 格式与常量不变）**：**免生产盘点、无迁移脚本、不写生产数据**（D1：全库写入均经 `HashPassword` 恒 32 字节）；`$sm3$` 格式 / `DefaultPasswordConfig`（600000）/ 25 个 operlog 常量值一字未改；仅在上线说明 / SUMMARY Known Gaps 提示「人工直接改库的历史脏数据」风险。
- [ ] **MH-5（成功标准 4 + 5｜零新依赖 / 包级与 build 全绿 / CRYPTO-04 合规出口）**：不新增任何外部依赖（`go.mod` 未改，CRYPTO-04 若用 oracle 复用已有 `golang.org/x/crypto v0.46.0 // indirect`）；`go test -count=1 ./internal/core/security/` 全绿 + `go build ./...` 全绿；CRYPTO-04 要么「实现 + oracle 核验 + 前缀不变性 `DK(64)[0:32]==DK(32)` + 守卫保留」，要么「合规跳过 + SUMMARY 登记守卫已收口、非遗留缺陷」——**只补循环不加守卫判为不通过**（ROADMAP line 181 禁区）。

---

## 7. 禁止（硬边界，违者视为执行失败）

1. **禁止一切 git 写操作**：`git add` / `git commit` / `git stash` / `git checkout` / `git restore` / `git reset` 全部禁止；`git status --short` 只读允许。`commit: false`，Lead 统一提交。
2. **禁止改其它 phase 文件面**（W1 并行冲突面，ROADMAP line 115 写集合须互斥）：`internal/utils/operlog/**`、`scripts/**`、`.github/workflows/ci.yml`（归 145）、`internal/services/asset/**`（归 147）、`internal/services/operations/**`（归 148/149/150）、`xingran-react-frontend/**`（归 149/150）、`internal/models/base/**` / `internal/api/router.go` / `.golangci.yml`（归 148/149/151）。本 phase 写集合**仅** `internal/core/security/*` + 本 phase 的 `146-SUMMARY.md`。
3. **禁止改 25 个 operlog 常量**（`internal/utils/operlog/regression_test.go` 硬锁，值固定不可重排）。
4. **禁止改 `$sm3$` 格式 / `DefaultPasswordConfig`（含 `Iterations=600000`）**；禁止把 `HashPassword:110` 的 `32` 改成其它值；禁止让存量哈希需要迁移。
5. **禁止引入新外部依赖** / `go get` / 改 `go.mod`、go.sum（零新依赖硬约束；CRYPTO-04 只能用已有 `golang.org/x/crypto` 或自构造向量）。
6. **禁止 `panic`**（项目 service 返回 error 约定）；禁止对非法入参「静默降级 / 返回全零 / 仅打日志」。
7. **禁止重写 `keyLen=32` 计算路径**（禁止改成 HMAC / 标准 PBKDF2 语义 / 多块通用循环影响单块输出）；禁止在黄金向量不一致时「更新期望值」蒙混过关。
8. **禁止启动长驻进程 / dev server / watcher**；一次性 `go test` / `go build` 直接前台跑（符合 DSH 进程托管约定）。
9. **禁止写生产数据 / 生产盘点 / 迁移脚本 / 数据修复脚本**（D1 免盘点）；禁止跑全仓 `go test ./...`（由 Lead 集成后统一跑，本 executor 只跑相关包）。
10. **禁止改其它 phase 的 `.planning/phases/*` 文件**；禁止改 `.planning/STATE.md` / `ROADMAP.md` / `REQUIREMENTS.md` / 根因笔记（均为 Lead/roadmapper 权威件）。
11. **禁止「只补多块循环却删/弱化守卫」**（ROADMAP line 181）；CRYPTO-04 未实现时禁止在 SUMMARY 谎称已实现。
12. **禁止在仓库根落 `.exe` / 构建产物**（AGENTS.md：产物只落 `out/`）；临时采集 / 变异文件用完即删。

---

## 8. 完成条件（executor 返回给 Lead）

- 6 个任务全部完成（Task 4 可为「合规跳过」但必须在 SUMMARY 登记）；
- §6 五条 must_haves 全部打勾，附 `go test` / `go build` / 变异验证输出；
- `146-SUMMARY.md` 落盘、明确「未 git commit」；
- 返回 `## PLANNING COMPLETE (Phase 146)` + 任务数 / 验证命令 / 文件面一句话；
- 若出现冲突（如 W1 写集合实测交集非空、`go.mod` 缺 `x/crypto` 且不允许自构造向量、黄金向量与 §3.4 不一致且无法定位）→ 返回 `## PLANNING BLOCKED` + 具体原因，不擅自改约束。
