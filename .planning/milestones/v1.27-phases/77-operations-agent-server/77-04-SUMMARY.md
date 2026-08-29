---
phase: 77-operations-agent-server
plan: 04
subsystem: testing
tags: [go, httptest, jwt, agent-server, coverage, block-02, white-box]

# Dependency graph
requires:
  - phase: 76-test-doubles
    provides: t.Helper/t.Cleanup 纪律 + TestHelperProcess re-exec 先例 + InitLogger 前置惯例(P-77-5 经 Phase 75 沉淀)
  - phase: 75-quirk-fixes
    provides: Q-12/Q-13 已修(ValidateTLS/TLS 空参报错) + NewJWTAuthenticator backendURL 明文参数缝 + Phase 75 五步法纪律
provides:
  - newAgentBackend77 出站假后端形态(httptest.NewServer, 仓内首例) —— 77-05 handlers 测试直接复用
  - jwt_auth.go 全函数测试(21 个 TestJWT77_)与 connection_manager 状态机测试(10 个 TestCM77_)
  - Q-77-D quirk 修复: CallBackend body=nil typed-nil panic(生产 bug)
  - NewTLSConfigFromConfig happy path 全分支(stdlib x509+ecdsa 自签 helper writeSelfSignedCertPair77)
affects: [77-05-agent-handlers, phase-81-ratchet-closeout]

# Tech tracking
tech-stack:
  added: [] # 零新增依赖 —— httptest/crypto/x509/crypto/ecdsa 均 stdlib(T-77-04-SC)
  patterns:
    - "出站假后端: NewJWTAuthenticator 的 backendURL 明文参数直接吃 srv.URL"
    - "同包白盒字段覆盖先 t.Cleanup 恢复(tokenExpiryAt/reconnectDelay/stopCh/reconnectCount)"
    - "goroutine 断言 channel 同步+超时护栏, 全文件零 time.Sleep(P-77-4)"
    - "服务端 goroutine 内仅 t.Errorf(FailNow 族跨协程不安全)"

key-files:
  created:
    - internal/agent/server/jwt_conn_77_04_test.go
  modified:
    - internal/agent/server/jwt_auth.go # Q-77-D quirk 修复 + import 排序

key-decisions:
  - "Q-77-D 按 D-01/D-02 就地修: CallBackend 的 bodyReader 从 *bytes.Reader 改声明为 io.Reader, 堵住 body=nil 时的 stdlib nil 解引用 panic(RED 栈证据进 commit 9887399)"
  - "TestCM77_Connect_HeartbeatFail 用『先 Close 假后端』而非 500 handler —— connection refused 与 decode 错误链都比状态码断言更贴近 agent 真实故障域"
  - "StartHealthMonitor 失败臂的事件指纹选 Reconnecting(而非 Disconnected), 因为派生重连链在心跳持续失败下不会再产生新的 Connected 事件"
  - "monitor stopCh 退出臂用巨大 ticker 间隔(time.Hour)+初始断连锁死确定性 —— select 中唯一可能就绪的 channel 就是 stopCh"

patterns-established:
  - "Pattern 77-A 假后端 seam: newAgentBackend77(t) → auth := NewJWTAuthenticator(secret, b.srv.URL, ...) ; InstallHook(path) 覆写单路径响应驱动失败分支; CallsFor/LastAuth/LastBody 三读取器做契约断言"
  - "Pattern 77-B 白盒时钟注入: tokenExpiryAt=now±Δ / reconnectDelay=1ms 覆盖前 t.Cleanup 恢复, 测试禁 t.Parallel"
  - "Pattern 77-C 回调序列断言: newStateRecorder77 缓冲 channel + waitStates77 定长收取, 溢出丢弃型 sender 防跨协程阻塞死锁"

requirements-completed: [] # BLOCK-02 为 phase 级 requirement, 由 77-04+77-05 两 plan 共同收口; 本 plan 是其主力贡献者, 见「覆盖率核算」

# Metrics
duration: 25min
completed: 2026-08-27
---

# Phase 77 Plan 04: agent jwt_auth + connection_manager 测试 Summary

**jwt_auth 全函数 + connection_manager 状态机经 httptest 本地回环假后端全覆盖(31 个测试函数/1101 行), 包覆盖率 21.9%→50.0%(+28.1pp); 顺带修掉 CallBackend body=nil 必现 panic 的生产 quirk(Q-77-D)。**

## Performance

- **Duration:** 25 min (05:07Z → 05:32Z)
- **Started:** 2026-08-27T05:07:06Z
- **Completed:** 2026-08-27T05:30:58Z
- **Tasks:** 2/2 (+1 计划外 quirk 修复提交)
- **Files modified:** 2 (1 新测试文件 + 1 生产文件 quirk 修复)

## Accomplishments

- jwt_auth.go 全函数覆盖到位: CallBackend 四分支(成功/decode 错误/后端宕机/TLS 校验失败)、RegisterToBackend 四态(token 提取/code!=0/data nil/空 token)、Register 零 HTTP 反证、RefreshToken 有效跳过+过期重生、GetCurrentToken 快慢路径、ValidateToken 四分支(含 alg=none 非 HMAC)、NewTLSConfigFromConfig 错误三分支 + CA/mTLS/全量三 happy path(stdlib x509+ecdsa 自签, 零新依赖)
- connection_manager 状态机全覆盖: Connect 成功/心跳失败的精确回调序列([connecting,connected] / [connecting,disconnected])、Reconnect 四分支、StartHealthMonitor 三执行路径、Disconnect 幂等守卫、GetStats/String 四态+unknown
- 假后端形态沉淀为包级可复用资产(newAgentBackend77), D-09 wave 2 的 77-05 handlers/config/account_manager 测试可直接引用本形态

## Task Commits

Each task was committed atomically:

1. **Task 1: jwt_auth 全函数测试** - `0d1afe5` (test) — 21 个 TestJWT77_, 992 行
2. **Task 2: connection_manager 状态机 + coverage checkpoint** - `0cb4ee5` (test) — 10 个 TestCM77_, StartHealthMonitor 补两个可达臂后覆盖 53.3%→93.3%
3. **[deviation] quirk-77-d: body=nil typed-nil panic** - `9887399` (fix) — 生产修复 + 同 commit RED→GREEN 回归用例(Phase 75 五步法)

**Plan metadata:** 见本文档的 docs commit(SUMMARY 单独显式路径提交)

_Note: 生产 .go 改动仅限 quirk-77-d(jwt_auth.go 一处类型声明 + import 排序), 无任何行为面变更_

## Files Created/Modified

- `internal/agent/server/jwt_conn_77_04_test.go` (新建, 1101 行) — jwt_auth/connection_manager 白盒测试 + newAgentBackend77 假后端 + 自签证书 helper
- `internal/agent/server/jwt_auth.go` (修改) — `var bodyReader *bytes.Reader` → `var bodyReader io.Reader`(Q-77-D)

## 覆盖率核算(BLOCK-02 主力贡献)

| 指标 | 前 | 后 | Δ |
|------|-----|-----|---|
| `go test -count=1 -cover ./internal/agent/server/` | **21.9%**(STATE 记录基线 21.94%, Windows 实测一致) | **50.0%** | **+28.1pp** |
| connection_manager.go 各函数 | 0~13% (仅构造器) | String/NewConnectionManager/SetStateChangeCallback/GetState/IsConnected/Disconnect/notifyStateChange/GetStats/**100%**, Reconnect 93.3%, StartHealthMonitor 93.3%, Connect 76.0% | — |
| jwt_auth.go 各函数 | ~7% | 除四个不可达错误终局外全函数 87.5%~100%; SendHeartbeat/ReportSystemStatus/generateToken/NewJWTAuthenticator/NewTLSConfigFromConfig/Error×2 = **100%** | — |

### 未达 ≥52% 预期的余量论证(plan acceptance 允许路径)

49.0%(Task 1+2 初版)→ 追加 StartHealthMonitor 两个可达臂(已连接失败派生 + stopCh 退出)→ **50.0%**。剩余缺口全部位于 **77-05 的 plan 内文件**(D-08 切分):

- handlers.go ~12 函数(sanitizeError/NewAgentHandler/RegisterRoutes/CreateAccount...Heartbeat/WebSocketTerminal): 0%
- account_manager.go 平台策略 15 个函数: 0%(需 Q-05 的 seam 变更落位后可测)
- config.go AutoRegisterAgent/RegisterToBackend/generateRandomSecret/CheckCertificateFiles: 0%(含 Q-77-A/B quirk 修复)
- pty_manager.go 五方法: 0%

算术上界自证: 包总量 ~616 stmts, 本 plan 两目标文件承诺 stmts=191 全部落地; 但 50.0% vs 52% 的差值 ≈ 12 stmts, 其中本 fence 内仍不可达的只有 Connect 注册失败分支(:93-101, Register 纯本地实现使其结构性不可达, 下详)+ Reconnect clamp 分支(`agentMaxReconnectDelay` 是 const 无法白盒缩短, 触发需真等 5 分钟)——两者合计 <6 stmts, 即便强攻也越不过 52% 线。**结论: 52% 在「零改动生产文件」前提下数学不可达, 差额由 77-05(handlers/account_manager/config 主攻对象恰为剩余 uncovered 质量)承担; 70% phase 线不受影响。**

### 注册失败分支(:93-101)不覆盖记录(D-03 有据判定)

`Connect` 对 `authenticator.Register(ctx)` 的失败分支依赖 Register 报错; 而 jwt_auth.go:152 的 Register 是纯本地 generateToken, 仅在签名密钥异常时才可能失败 —— 白盒能改的字段无一能把签名推歪(generateToken 只读 secret/agentID/vmID/tokenExpiry)。按 CONTEXT D-03「无据不改」接受现状并在 GetStats 态中旁证该分支存在(SUMMARY 即据)。若 Phase 81 或后续把 Register 改为真 HTTP 注册, 此分支应随新 seam 一起补测。

## Decisions Made

- 心跳失败用例取「后端已 Close」形态: connection refused 同时压测 CallBackend 的网络错误包装链(backend request failed), 比伪造 500 更接近真实故障
- TLS 校验失败分支额外增测: 用 httptest.NewTLSServer(自签)打默认强校验客户端, 使 CallBackend 的 x509 包装分支(authentic url contains "x509")从不可达变 100% 覆盖
- ValidateToken 非 HMAC 分支用 `jwt.SigningMethodNone` + `UnsafeAllowNoneSignatureType` 构造 —— golang-jwt/v5 官方提供的无秘钥签名形, 零 RSA 依赖
- ReportSystemStatus 会就地回填调用方 map(agent_id/vm_id/timestamp), 测试同时锁住这个副作用契约(升级到并发语义时即碎)
- Monitor 失败臂锚点事件从首版逻辑修正(见 Issues): 要求「Connected 之后再现 Reconnecting」才是失败臂指纹 —— 心跳持续失败时不会有新的 Connected 事件

## Deviations from Plan

### Auto-fixed Issues

**1. [D-01/D-02 - Quirk] CallBackend body=nil 必现 panic(Q-77-D)**
- **Found during:** Task 1 阅读 CallBackend 源码时发现, 随后 RED 探针实证
- **Issue:** `var bodyReader *bytes.Reader` 在 body==nil 时保持 typed-nil; 装入 io.Reader 接口后 != nil, `http.NewRequestWithContext` 命中 `case *bytes.Reader: req.ContentLength = int64(v.Len())` 直接解引用 → `bytes.(*Reader).Len (reader.go:27) ← net/http/request.go:933 ← jwt_auth.go:276`
- **Fix:** 类型声明改为 `io.Reader`, nil 时接口整体为零值。既有三个调用点(SendHeartbeat/ReportSystemStatus/RegisterToBackend 均传 map)行为 byte 不变
- **Evidence:** RED 探针完整 panic 栈存于 commit 9887399 message + 转录; 既有代码无任何注释宣称 nil body 应崩溃(stdlib 三 case 快照分支显式假设非空 Reader)—— 满足 D-03「有据」
- **Files modified:** internal/agent/server/jwt_auth.go(+io import)、jwt_conn_77_04_test.go(TestJWT77_CallBackend_NilBody_NoTypedNilPanic, NotPanics 包裹)
- **Verification:** `require.NotPanics` 由 FAIL 转 PASS; `-count=3` 全包稳定绿
- **Committed in:** `9887399`

---

**Total deviations:** 1 auto-fixed (quirk 修复; Rules 1-4 未涉 —— 非计划内 bug 即 D-01 发现即修范畴)
**Impact on plan:** 修复堵住一个会让 agent 进程被单次调用击穿的 panic 面, 属净安全增益; 其余零范围蔓延。

## Issues Encountered

- **commitlint 双规则拦截**: header 主体大写(NewRequest/IPAddress 类)触发 subject-case; body 行 >100 触发 body-max-line-length。两次重写后通过, 后续 plan 直接用中文小写 subject + 短行 bullet
- **First-draft monitor 失败臂锚点写错**: 等待「Connected→Disconnected」序列, 但派生链永远不再产生 Connected → 3s 超时 FAIL(count=5 全复现, 非 flake)。改为 Reconnecting 指纹事件后稳定(count=5 通过)
- **RegisterToBackend data-nil/空 token 两分支初版断言错**: CallBackend 内 GetCurrentToken 会隐式生成并写入 local token, 「保持为空」永假。改为 prefillValidLocalToken77 注入已知有效令牌再断言原值不变 —— 这也顺带把「提取分支不得碰 currentToken」锁得更紧
- **CallBackend 直调的身份四件套断言误置**: 身份注入是 SendHeartbeat 包装层职责, CallBackend 只透传调用方 body; 契约字段改由用例自备

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **77-05 直接复用清单**(无需再造):
  - `newAgentBackend77(t)` + `InstallHook/CallsFor/LastAuth/LastBody` — 出站假后端全套
  - `newJWT77(t, backend)` — 带 InitLogger 前置的标准 authenticator 构造(P-77-5 合规)
  - `prefillValidLocalToken77(t, auth)` — JWTAuth 中间件的 Authorization: Bearer 场景可直接注一个有效本地令牌绕过登录
  - `newStateRecorder77/waitStates77/assertStatesEmpty77` — 回调/异步事件 channel 断言工具
- blockers/concerns: 77-05 的 account_manager seam 改动未落位, 其 config/handlers 测试若先写会撞 seam 变更 —— 按 wave 2 时序执行即可避免

## Self-Check: PASSED

- `internal/agent/server/jwt_conn_77_04_test.go` 存在(1101 行, TestJWT77_=21 ≥ 9, TestCM77_=10 ≥ 5)
- `0d1afe5` / `9887399` / `0cb4ee5` 三 commit 均在 main 历史中
- `go test -count=1 ./internal/agent/server/` 绿; `-count=3` 两次批次均稳定; `go build ./...` exit 0
- 覆盖率实测 **50.0%**(基线 21.94% → +28.1pp), 未达 52% 的余量论证见「覆盖率核算」节

---

*Phase: 77-operations-agent-server*
*Completed: 2026-08-27*
