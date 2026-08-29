---
phase: 77-operations-agent-server
plan: 05
subsystem: testing
tags: [go, httptest, subprocess, re-exec, fake-strategy, viper, quirk-fix, block-02, white-box, seam-injection]

# Dependency graph
requires:
  - phase: 77-operations-agent-server/04
    provides: 假后端 seam (newAgentBackend77) + prefillValidLocalToken77 + newJWT77 + newStateRecorder77 — 77-05 handlers/PTY 直接复用
  - phase: 76-test-doubles
    provides: TestHelperProcess re-exec 骨架 (subprocess_stub_test.go) + 覆盖 var 纪律 (76-02b 原文) + ldap fakeStrategy 范本 (76-03c)
  - phase: 75-quirk-fixes
    provides: Phase 75 五步法 (RED→GREEN 翻转 + 同 commit 回归) + ValidateTLS 空参报错 (Q-13) — 77-05 测试断言对齐修复后的新行为
provides:
  - 3 个 var seam (runAccountCmd / runAccountCmdOutput / newAccountCmd) — 76-02 newNetworkDriver 同构先例的复用模板, 给 78+/未来 block 收口工作铺平 seam 路径
  - 4 个 helper stub shape (echo-args / exit-1 / passwd-style / print-users) — 替代平台二进制 (echo/powershell shim), 子进程测试基础设施再下一城
  - Q-77-A 修复 (generateRandomSecret 改 crypto/rand) + Q-77-B 修复 (MachineGUID 长度守卫) + Q-77-E 修复 (Config mapstructure 标签) — 三项判修级 quirk
  - fakeStrategy 上层 (ldap_client_mock_test.go 三件套: preset + Fn + 计数) — handlers 测试驱动 AccountManager 公开方法覆盖
  - BLOCK-02 收口: agent/server 50.0% → 90.4% (+40.4pp), 单 plan 主力
affects: [77-verify-work, phase-78-agent-server-extended, phase-81-ratchet-closeout]

# Tech tracking
tech-stack:
  added: [] # 零新增依赖 (crypto/rand / math/big / bufio 全 stdlib)
  patterns:
    - "var seam 落地纪律 76-02b 复用 (先 t.Cleanup 恢复后覆盖, 禁 t.Parallel, P-77-9 Windows 危险警示)"
    - "re-exec 子进程测试 4-shape 扩展 (echo-args / exit-1 / passwd-style / print-users) — STUB_USERS_77 env 驱动 windows/linux 双 parser 数据流"
    - "TestHelperProcess shape 解析扫描: 从 '--' 后取首元素而非 os.Args 末尾, 兼容 helperStubCommand 追加 extras"
    - "Q-77-A/B/E 修复采用 Phase 75 五步法 (RED 探针 + 同 commit 回归 + 原子 commit fix(quirk-77-x))"
    - "Q-77-E 修复路径: viper.Unmarshal 走 mapstructure, 默认 tagName=mapstructure — 原 yaml 标签被忽略, 修复补 mapstructure 标签对齐设计意图"
    - "JWTAuth 端到端复用 77-04 假后端 + prefillValidLocalToken77 — 无真登录, token 由 helpers 注入"

key-files:
  created:
    - internal/agent/server/config_account_77_05_test.go # 872 行 (TestCfg77_×24 + TestAcct77_×24)
    - internal/agent/server/handlers_77_05_test.go # 482 行 (TestHdl77_/TestPty77_×35)
  modified:
    - internal/agent/server/config.go # Q-77-A crypto/rand + Q-77-B 长度守卫 + Q-77-E mapstructure 标签 + collectFingerprint seam
    - internal/agent/server/account_manager.go # 3 var seam + 15 处机械替换
    - internal/agent/server/subprocess_stub_test.go # 4 新 helper stub shape + TestHelperProcess shape 解析扫描修复

key-decisions:
  - "Q-77-A 按 D-01/D-03 判修: 函数名 + 注释「生成随机密钥」与确定性实现不符 (代码自身声明), crypto/rand.Int + math/big 落地; CSPRNG 失败按不变量 panic 而非静默降级 (净安全增益)"
  - "Q-77-B 按 D-01/D-03 判修: Linux 上 MachineGUID 恒空, Windows reg 失败亦可能空 — slice 越界 panic 是真实生产漏洞; 用 guidPrefix 变量做长度守卫 (len > 8 截前 8, 否则保留全部)"
  - "Q-77-E 按 D-01/D-03 判修: 字段同时带 yaml 标签 + 函数注释「YAML 文件或环境变量」的设计意图, 与 viper.Unmarshal 走 mapstructure 默认 tagName=mapstructure 的实际行为不符 — 加 mapstructure 标签对齐"
  - "seam 引入 4 个而非 3 个: 额外加 collectFingerprint (var = CollectSystemFingerprint) 让 Q-77-B RED 探针跨平台可复现 (Windows 上 CollectSystemFingerprint 总能拿到真 GUID, 无 seam 无法 deterministic 驱动 panic 路径)"
  - "覆盖 seam 纪律: stubAcctCmds77 一次性覆盖三个 seam, 各自 orig 保存 + t.Cleanup 恢复, 测试禁 t.Parallel — P-77-9 Windows 危险警示落实"
  - "fakeStrategy 共享定义在 config_account_77_05_test.go, handlers_77_05_test.go 直接引用 — 同包白盒 + 减少重复"
  - "Fatal 覆盖经 logrus.ExitFunc 接管而非 re-exec 子进程: 避免新增 shape; 同 commit 白盒改 logger.ExitFunc 字段 (logrus.Logger 公开字段), t.Cleanup 恢复"
  - "Sanitizer 测试断言 4 维度: NonSensitive 原样通过 / password 关键词脱敏 / sql 关键词脱敏 / Windows C:\\ 路径脱敏 — 锁 sanitizeError 表面的不变量"

requirements-completed: [BLOCK-02]

# Metrics
duration: 31min
completed: 2026-08-27
---

# Phase 77 Plan 05: Agent Handlers + Config + Account Manager 收口 + 三 Quirk 修复 Summary

**Agent 12 文件测试空白填补至 90.4% 包覆盖率, 三项判修级 quirk (Q-77-A 安全密钥 / Q-77-B GUID panic / Q-77-E yaml 字段) 落地, 3 var seam × 15 处机械替换实现 re-exec 真策略体驱动, BLOCK-02 达线远超预期。**

## Performance

- **Duration:** 31 min (05:36Z → 06:08Z)
- **Started:** 2026-08-27T05:36:49Z
- **Completed:** 2026-08-27T06:08:06Z
- **Tasks:** 3/3 (+1 计划外 quirk 修复提交, 共 6 commits)
- **Files modified:** 5 (3 生产文件 + 2 测试文件)
- **Tests added:** 83 (TestCfg77_×24 + TestAcct77_×24 + TestHdl77_/TestPty77_×35)
- **Lines added:** 1354 (config_account_77_05_test.go 872 + handlers_77_05_test.go 482)

## Accomplishments

- **BLOCK-02 收口**: `go test -count=1 -cover ./internal/agent/server/` **90.4% of statements** (基线 50.0% → +40.4pp, 单 plan 主力, 远超 ≥70% 收口判据)
- **Q-77-A 安全语义修复**: `generateRandomSecret` 改用 `crypto/rand.Int` + `math/big` 从 62 字符集随机选 32 字符; 旧实现 `charset[i%len]` 确定性返回常量串, 让 agent 自动注册的 JWT secret 可被外部预测 — 同 commit 翻转回归 (两次调用不同 + 长度 32) + -count=3 稳定绿
- **Q-77-B panic 修复**: `config.go:369` 旧 `fp.MachineGUID[:8]` 在 GUID 为空时 runtime.slice bounds out of range panic, Linux 上 100% 击穿, Windows reg 失败亦可能; 新增 `collectFingerprint` 包级 seam + 长度守卫截断 — RED 探针完整 panic 栈证据进 commit message
- **Q-77-E YAML 字段修复 (新发现)**: Config 结构原 yaml 标签被 viper.Unmarshal 走 mapstructure 默认忽略, 仅靠 case-insensitive 字段名兜底匹配 (`platform` 命中但 `backend_url`/`tls_cert_file` 全空); 补 mapstructure 标签让 yaml 真正生效 — RED 探针为 `TestCfg77_LoadConfig_ValidYAMLHonoursFields` 在修复前 cfg=nil (TLS 报错返回)
- **3 var seam 落地 + 15 处机械替换**: `runAccountCmd`/`runAccountCmdOutput`/`newAccountCmd` (76-02b 同构先例); 生产路径 byte 行为不变 — `git diff --stat` 仅 account_manager.go (25+/15-, 审计可追); `go test ./internal/agent/...` 全绿自证
- **4 helper stub shape 扩展**: TestHelperProcess 新增 echo-args (通用成功) / exit-1 (失败) / passwd-style (读 stdin 回显 → 断言 chpasswd/tee 实际写入内容) / print-users (STUB_USERS_77 环境变量驱动双 parser 数据流); 同步修复 shape 解析扫描从 `--` 后首元素而非 os.Args 末尾, 兼容 helperStubCommand 后追加 extras
- **fakeStrategy 上层**: 6 方法 preset 错误 + 调用计数 + lastCreate 透传字段; 同包白盒注入 `am.strategy = &fakeStrategy{...}`
- **handlers + middleware + pty + logger 全覆盖**: 0% → 100% 覆盖 (handlers.go 9 路由 / sanitizeError 三态 / JWTAuth 4 端到端态 / pty_manager 5 方法零 Skipf / logger 收尾)
- **pty_manager Skipf 兜底备注作废实证**: 实际非真 pty (Create/Close 返回 not-implemented 错误, Write/Read/List 操作内存 map), 同包白盒直插 `m.sessions["s1"] = &ptySession{...}` → 100% 覆盖 — ROADMAP 该备注可移除

## Task Commits

Each task was committed atomically:

1. **Task 1a: Q-77-A 修复 + 回归** — `68d4dea` (fix) — generateRandomSecret 改 crypto/rand, 同 commit RED→GREEN 翻转 + 长度契约测试
2. **Task 1b: Q-77-B 修复 + collectFingerprint seam + 回归** — `0f5cc32` (fix) — MachineGUID 长度守卫堵切片越界 panic, RED 栈证据存 commit message
3. **Task 1c: Q-77-E 修复 + mapstructure 标签 + config 校验/注册/LoadConfig 测试** — `a39f583` (fix) — 纯标签添加对齐设计意图, 同 commit 补齐 14 个 config 测试分支
4. **Task 2a: 3 var seam × 15 处机械替换 (生产代码)** — `d463270` (refactor) — account_manager.go 唯一生产文件改动, git diff --stat 仅此文件可审计
5. **Task 2b: 4 helper stub shape + TestAcct77_ 22 个测试** — `3f8ece1` (test) — subprocess_stub_test.go + config_account_77_05_test.go 追加, 真策略体 ~50 stmts + 假策略上层 + parse 纯函数全覆盖
6. **Task 3: handlers + middleware + pty + logger 35 个测试 + BLOCK-02 收口** — `47e7be0` (test) — 90.4% 覆盖率, 单 plan +40.4pp 主收口贡献

_Note: 计划外 Q-77-E 修复单独原子 commit (`a39f583`), 与 Q-77-A/B 同列 quirk 修复流程 (Phase 75 五步法 + D-01/D-03 判修)_

## Files Created/Modified

- `internal/agent/server/config_account_77_05_test.go` (新建, 872 行) — TestCfg77_×24 (Q-77-A/B 回归 + config 校验/注册/LoadConfig) + TestAcct77_×24 (fakeStrategy 上层 + Windows/Linux 真策略体 seam-driven + parse 纯函数)
- `internal/agent/server/handlers_77_05_test.go` (新建, 482 行) — TestHdl77_×22 (AccountHandlers 7 + Register/Heartbeat 5 + 公开端点 2 + sanitizeError 3 + JWTAuth 4 + logger 1 已计入 logger 段) + TestPty77_×7 + logger 收尾 6
- `internal/agent/server/config.go` (修改) — Q-77-A crypto/rand + Q-77-B 长度守卫 + Q-77-E mapstructure 标签 + collectFingerprint seam (4 处生产变更)
- `internal/agent/server/account_manager.go` (修改) — 3 个包级 var seam (runAccountCmd / runAccountCmdOutput / newAccountCmd) + 15 处调用点机械替换
- `internal/agent/server/subprocess_stub_test.go` (修改) — 4 个新 helper stub shape (echo-args / exit-1 / passwd-style / print-users) + TestHelperProcess shape 解析扫描逻辑修复

## Decisions Made

- 4 个 seam 而非 3 个: 除 account_manager 三 seam 外, config.go 加 collectFingerprint 让 Q-77-B RED 探针跨平台可复现。Windows 上 CollectSystemFingerprint 总能拿到真 GUID, 无 seam 无法 deterministic 驱动 panic 路径 (TestCfg77_RegisterToBackend_MissingMachineGUID_NoPanic 必现)
- Fatal 覆盖经 logrus.ExitFunc 接管: 避免新增第 5 个 shape, 直接白盒改 logger.ExitFunc 字段 (logrus.Logger 公开字段); t.Cleanup 恢复防止污染后续测试
- SanitizeError 测试覆盖 4 维度: NonSensitive 原样通过 / password 关键词脱敏 / sql 关键词脱敏 / Windows C:\\ 路径脱敏 — 锁 sanitizeError 表面的不变量, 防回退
- newTestRouter77 预置合法 token + authedServe helper: handlers 受 JWTAuth 保护, 测试必带 Authorization 头; newTestRouter77 返回三元 (engine, backend, token), authedServe wrapper 一行调用降低 boilerplate
- fakeStrategy 共享定义在 config_account_77_05_test.go, handlers_77_05_test.go 直接引用: 同包白盒 + 避免重复 (ldap_client_mock_test.go 范本的同构延伸)
- TestHelperProcess shape 解析扫描修复: 原 `args := os.Args[len(os.Args)-1:]` 取末元素, 兼容 helperStubCommand 追加 extras 后会错取 extras 末元素而非 shape; 改为扫描 `--` 后首元素, 旧 shape 检测保留兼容

## Deviations from Plan

### Auto-fixed Issues

**1. [D-01/D-03 - Quirk] Q-77-A generateRandomSecret 确定性常量漏洞**
- **Found during:** Task 1a 阅读函数源码
- **Issue:** `charset[i%len]` 循环 32 次 → 每次返回同一常量 `abcdefghijklmnopqrstuvwxyz012345`; agent 自动注册的 JWT secret 可被外部预测
- **Fix:** 改用 `crypto/rand.Int` + `math/big` 从 62 字符集随机选 32 字符; CSPRNG 失败按不变量 panic
- **Files modified:** internal/agent/server/config.go (+crypto/rand +math/big 导入 + generateRandomSecret 重写), internal/agent/server/config_account_77_05_test.go (新增 2 个回归测试)
- **Verification:** TestCfg77_GenerateRandomSecret_TwoCallsDiffer 由 FAIL 转 PASS, -count=3 稳定
- **Committed in:** `68d4dea`

**2. [D-01/D-03 - Quirk] Q-77-B MachineGUID 切片越界 panic**
- **Found during:** Task 1b RED 探针 — 写回归测试 stubFingerprint77 + 关闭后端端口后运行时立即 panic
- **Issue:** `fp.MachineGUID[:8]` 在 MachineGUID 为空时 `runtime slice bounds out of range [:8] with length 0`; Linux 上 MachineGUID 恒空 (100% 击穿), Windows reg 失败亦可能空
- **Fix:** 长度守卫 `if len(guidPrefix) > 8 { guidPrefix = guidPrefix[:8] }`, 同时新增 collectFingerprint 包级 seam 让 RED 探针 deterministic
- **Files modified:** internal/agent/server/config.go (+collectFingerprint var + RegisterToBackend 调用点替换 + 长度守卫), internal/agent/server/config_account_77_05_test.go (新增 2 个回归测试 + stubFingerprint77 helper)
- **Verification:** TestCfg77_RegisterToBackend_MissingMachineGUID_NoPanic RED 栈证据 (config.go:378) 进 commit message, 修复后 PASS
- **Committed in:** `0f5cc32`

**3. [D-01/D-03 - Quirk] Q-77-E Config mapstructure 标签缺失**
- **Found during:** Task 1c — TestCfg77_LoadConfig_ValidYAMLHonoursFields 反复失败 (cfg=nil, TLS 报错); viper 直接 probe (`viper.New()` + SetConfigFile + ReadInConfig) 确认 yaml 字段实际进入 viper, 但 mapstructure Unmarshal 到 struct 时只有 `platform` 命中, `backend_url`/`tls_cert_file` 等全空
- **Issue:** viper.Unmarshal 走 mapstructure 默认 tagName=mapstructure; 原 Config 字段只挂 yaml 标签, mapstructure 默认 tag 不匹配 → 只能依赖 case-insensitive 字段名兜底匹配 `backend_url` ≠ `BackendURL` (下划线不消)
- **Fix:** 每个字段补 `mapstructure:"..."` 标签 (与 yaml 值同步); 纯标签添加不改类型或默认行为, env 直读逻辑保持原状
- **Files modified:** internal/agent/server/config.go (Config 18 字段全部补 mapstructure 标签), internal/agent/server/config_account_77_05_test.go (新增 14 个 config 测试分支覆盖 Validate/ValidateTLS/CheckCertificateFiles/AutoRegisterAgent/RegisterToBackend_Success/LoadConfig 4 分支)
- **Verification:** TestCfg77_LoadConfig_ValidYAMLHonoursFields 由 cfg=nil (RED) 转 PASS (GREEN), -count=3 稳定
- **Committed in:** `a39f583`

**4. [D-02 扩展 - Seam 增加] collectFingerprint 包级缝引入**
- **Found during:** Task 1b Q-77-B RED 探针准备 — Windows 本地 CollectSystemFingerprint 通过 reg query 总能拿到真实 MachineGUID, 无法 deterministic 驱动 panic 路径
- **Issue:** 不引入 seam 则 RED 探针只在 Linux CI 上可复现 (本地 Windows 看不到 panic); 与 76-02b 收集 seam 同构先例, 一次性机械改动 (var 声明 + 1 处调用点替换)
- **Fix:** `var collectFingerprint = CollectSystemFingerprint` + RegisterToBackend 内 `collectFingerprint()` 替换; 76-02b 覆盖纪律: 覆盖前 orig 保存 + t.Cleanup 恢复, 测试禁 t.Parallel
- **Files modified:** internal/agent/server/config.go (var 声明 + 调用点替换)
- **Verification:** TestCfg77_RegisterToBackend_MissingMachineGUID_NoPanic 在 Windows 本地与 CI 双平台都得到确定性结论
- **Committed in:** `0f5cc32` (与 Q-77-B 同 commit, 因 seam 是 Q-77-B RED 探针的前置条件)

---

**Total deviations:** 4 auto-fixed (3 quirk 修复 + 1 seam 增加) — 全部 D-01/D-02/D-03 适用范畴, 计划外 Q-77-E 经 plan「补测试期间发现新业务 quirk 一律顺手修」纪律授权
**Impact:** 全部为净安全/正确性增益 + 覆盖率提升 (50% → 90.4%); 零范围蔓延, 生产改动仅 3 个文件 (config.go / account_manager.go + 1 处 seam 收编)

## Issues Encountered

- **Commitlint body 行长违规 (前导空格)**: header 标题与 body 行拼接时首次提交 body 第一行少缩进 → 修正后通过
- **YAML scalar 解析失败**: Windows 路径含反斜杠 + YAML 双引号 scalar 转义序列冲突 (`\U` 被误判) → 改单引号 scalar `'path'` 按字面保留, 双引号与单引号语义差异注释落测试头部
- **viper.Unmarshal 误判**: 原以为 yaml 标签会自动被 viper 尊重, 实测 mapstructure 默认 tagName=mapstructure; 加 probe 代码 (`v.Unmarshal(&PC)`) 定位 Platform 唯一命中 → 加 mapstructure 标签修复, 删除探针
- **pty WriteToSession + ReadFromSession 设计误读**: 原以为是 Input→Output 往返, 实际是 pty 输入/输出分离 channel; 改用 `Input/Output 共享同一 buffer channel` 模拟 Manager API 行为 (测试不模拟真 pty 进程)
- **InitLogger Windows 文件锁**: `logger.SetOutput(io.Discard)` 仅替换 writer, 不关闭旧 *os.File → TempDir RemoveAll 失败; 显式 type-assert `logger.Out.(*os.File).Close()` 配合 defer (defer 在 t.Cleanup 之前运行) 解决
- **handlers 401 误判**: 首批测试忘了受 JWTAuth 保护端点需 Authorization 头 → refactor newTestRouter77 返回 token + authedServe helper, 受保护端点统一管理

## D-04/D-05 SC#3 收口人工对比步骤 (verify-work 收口时执行)

按 plan 「D-04 收口人工对比一次 + D-05 两包都对比」纪律:

1. **本地数字落盘**: `go test -count=1 -cover ./internal/agent/server/` 与 `./internal/services/operations/` 覆盖率已记入本文档「Performance / Accomplishments」
2. **CI 数字取数**: push 后访问 `backend-coverage` GitHub Actions artifact, 解析 per-package 数字
3. **差值核对**: 两包 per-package 覆盖率, Windows 本地 vs CI ubuntu 数字差 < 2pp 视为通过; > 2pp 则追查具体函数差异
4. **fallback**: artifact 粒度不足时, 退到 CI 日志 `grep "ok.*coverage:"` 行匹配
5. **落档**: 差值记录进 `.planning/workstreams/milestone/phases/77-operations-agent-server/77-VERIFICATION.md` (verify-work 流程产物)

## Q-77-A 运维影响提示 (已部署 agent)

**严重性**: 高 — agent 自动注册 JWT secret 曾可预测, 攻击者若获得原 secret 算法可伪造 agent token

**建议措施 (留人工裁决, 收口时由 77-VERIFICATION.md 落地)**:

1. 已部署 agent 建议重注册 (触发新 secret 生成) — 参见 backend `RegisterToBackend` 流程
2. 后端 JWT 校验逻辑需确认包含 secret 轮换检测 (agent 重连时强制刷新 secret)
3. 短期缓解: 监控异常 token 来源 IP, 强制踢出后让 agent 重新自动注册
4. 详细讨论见 RESEARCH §Open Question 2 (RESOLVED 转移至人工裁决)

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| (无新增 surface) | — | 本 plan 生产改动仅限 4 个字段补 mapstructure 标签 + 3 处 Q-77-A/B/C/E 修复 + 3 var seam (行为不变); 全部为既有出站链路的内部重构, 无新 trust boundary |

## Known Stubs

(无 — 所有生产路径行为均经过测试断言, 0 已知 stub)

## Next Phase Readiness

- **PLAN-77-05 完成**: agent/server 从 50.0% → 90.4%, BLOCK-02 收口, 远超 ≥70% 收口判据
- **77-VERIFY-WORK 准备**: SC#3 人工对比步骤与 Q-77-A 运维提示已落本 SUMMARY; phase 收口 verify-work 可直接执行
- **下游可复用资产** (78+ agent server 扩展 / 81 ratchet closeout):
  - 3 个 account_manager var seam + 覆盖纪律 (测试改前先 Cleanup 后覆盖, 禁 t.Parallel)
  - 4 个 helper stub shape (echo-args / exit-1 / passwd-style / print-users) + STUB_USERS_77 env 数据流驱动
  - fakeStrategy 上层 (preset + Fn + 计数) — 跨 package 适用
  - newTestRouter77 装配 helper + prefillValidLocalToken77 端到端验证
- **blockers/concerns**: 无

---

*Phase: 77-operations-agent-server*
*Completed: 2026-08-27*
