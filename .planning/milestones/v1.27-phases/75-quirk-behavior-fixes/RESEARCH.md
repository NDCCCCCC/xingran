---
phase: 75
phase_name: QUIRK 行为修正
compiled: 2026-08-23
mode: lightweight-assembly (从 milestone 级研究抽取 QUIRK 专段,未重新 spawn researcher)
sources:
  - .planning/research/v1.27-pitfalls.md (§5 blast radius 逐项)
  - .planning/research/v1.27-architecture.md (五步法 + 受影响测试坐标)
---

# Phase 75 RESEARCH: 15 项 QUIRK 修复(汇编)

## 正典清单校正

15 项 QUIRK 正典**不**只在测试注释里:`internal/templates/templates_74_08_test.go`
内没有 QUIRK 注释(grep 证实)。正典分布在 74-08-SUMMARY(9 项)+ 74-09/74-12
SUMMARY 与各测试注释中。retry QUIRK 的业务影响面 = 0(零生产调用方),但 v1.26
测试锁定了 QUIRK 行为——**每个修复必须同 commit 翻转锁定断言**。

## 五步法(每个 QUIRK 一个原子 commit)

**五步法(每个 QUIRK 一个原子 commit):**
1. `grep -rn "QUIRK" --glob *_test.go` 定位锁定该行为的测试(当前 43 处注释命中);
2. 翻转断言:删 QUIRK 注释、旧期望改新期望(如 `"" → "S5735-L48T4X-A"`);
3. 补回归用例(同文件,用例名标注重语义,如"行首锚定命中");
4. 跑本包 + 受联动包(`go test ./pkg/cache/... ./internal/core/...`);
5. commit 格式 `fix(quirk-N): <语义> + 回归测试`,SUMMARY 记录生产影响面。

**已知受影响测试清单(实证坐标):**

| QUIRK | 锁定测试位置 | 联动效应 |
|-------|-------------|----------|
| MemoryCache.IncrementBy nil-deref + 静默 0 | `pkg/cache/cache_74_08_test.go:131,157`;`internal/core/core_74_08_test.go:127,155,210` | **core 三处 workaround(手工种 captcha 数据)可改为直测 GenerateCaptcha 真实链路 → core 覆盖率直接受益。必须最先修** |
| retry.containsIgnoreCase 恒 true | `internal/agent/pkg/retry/retry_74_12_test.go:42-47,105,144-148` | `TestIsNetworkError`/`TestDoWithRetry_NonRetryableFailsFast`/`TestContainsHelpers` 三处翻转;"长串恒 true"用例改断言 false |
| ModelExtractor 锚定 | `internal/device/device_74_08_test.go:39-45`(行首 QUIRK 空用例) | 同文件 USG 尾字母截断(QUIRK #8,~:80)一并翻 |
| parseTemplateContent $$ 约定 | `internal/templates/templates_74_08_test.go:138-139` + `textfsm_dot1x_test.go` | 行尾 `$$` 锚点语义行为锁定,翻转前先确认期望新语义 |
| utils(operlog background / config diff) | `internal/utils/config_diff_test.go:34`;`operlog/background_74_12_test.go` | diff 为轻度 QUIRK(空串语义),可留最后 |
| gmsm sm2.Decrypt panic | `pkg/crypto/crypto_74_08_test.go:324` | 修复=防御性 recover/长度预检,不翻既有断言、只加新用例 |
| validateFile 无扩展名 panic | `internal/core/core_74_08_test.go:413`(子用例曾被**移除**) | 恢复被移除的子用例 |

**排序原则:** 被其他包测试 workaround 依赖的 QUIRK(MemoryCache)最优先;纯语义翻转(ModelExtractor/retry)其次;防御性加固(gmsm)最后。

## Q5 go.mod 布局

**单一主 module,测试依赖直接进主 go.mod。** 理由:
- Go 没有原生 "test-only dependency" 隔离;分 module(如 `internal/agent/go.mod`)会制造 import 边界地狱(rpa-worker 先例是**可部署物**才分,module)。
- v1.27 D-02 已解锁"测试专用 devDependencies"——在 require 行加注释 `// test-only (v1.27 D-02)` 标记即可,grep 可审计。
- **tools directive 不需要**:不引入 CLI 工具链(golangci-lint 走 action 版本锁定)。

新增依赖预估:`github.com/alicebob/miniredis/v2`(v2.3x,go-redis v9 官方兼容,MEDIUM 版本号置信)+ `github.com/vjeantet/ldapserver`(近期仍有 pseudo-version 提交,libraries.io 显示 v1.0.2-0.2026x;备选 cloudogu 维护的 fork)。两者均纯 Go、无 CGO、跨平台。

## Q6 CI 策略

**约定 6:全部替身进程内化,CI 零改动。** 各包注入点(实证):

| 阻塞包 | 替身 | 注入点(已验证) | 新文件 |
|--------|------|----------------|--------|
| core(754)+ pkg/cache Redis 路径 | miniredis | Cache 接口实现换 Addr | `internal/core/redis_itest_<plan>_test.go` |
| addomain(2415) | 嵌入式 LDAP server(127.0.0.1:0) | `ADConfig` host/port 指向随机端口,LDAPClient.dialConnection 真实 wire | `internal/services/addomain/ldapserver_helper_test.go` |
| operations(3714) | httptest + RoundTripper | **geocoding_service.go:88 已有可注入 httpClient 构造器**——注入自定义 RoundTripper 拦截 `api.map.baidu.com` 请求返回罐头 JSON,**零业务改动** | `internal/services/operations/geocoding_itest_<plan>_test.go` |
| device(1249) | scrapligo FileTransport(已验证于 portwrite e2e,10+11 个 TestE2E_*) | 扩展 ForTesting 工厂家族(`NewPooledConnectionForTesting` 先例) | 复用 `internal/device/e2e_helpers.go` 模式 |
| agent-server(616) | TestHelperProcess 再执行模式(`os.Args[0] -test.run=TestHelperProcess`) | subprocess.go 的 cmd 构造点 | `internal/agent/server/subprocess_stub_test.go` |

**子进程 stub 跨平台警告(实证):** 现有 `subprocess_pgroup_test.go` 直接 `exec.Command("echo")`——Windows 上 echo 是 shell 内建无独立 exe,这正是 P2_RATCHET 注释记录的 agent-server "env-branch divergence"(本地 22.08% vs CI 19.48%)来源之一。新 stub 一律用 TestHelperProcess 再执行模式(可移植,Go 官方惯用法),不依赖平台二进制。

**若未来引入 Docker 级测试:** 单独 CI job(仿 coverage-diff 的 PR-only 模式)+ env guard `XINGRAN_ITEST=1` + 测试开头 `if os.Getenv("XINGRAN_ITEST") == "" { t.Skip }`,主 backend job 永不接触 Docker。`-short` flag 方案不推荐(语义被"快速单测"占用,易误触发)。

**覆盖率联动:** itest 自动计入 coverage.out → 4 层 gate(check-coverage.sh 的加权 + P1/P2 floor)与 diff-coverage ≥80% 无需改脚本;5 包过 70% 后按既有"删除 P2_RATCHET 条目"流程收口 floor。

## 置信度与开放问题

| 项 | 置信度 | 依据 |
|----|--------|------|
| 同包共置 / per-test fixture / go.mod 单 module | HIGH | 仓库内模式实证(newMem/newVDITestDB/A1/CI 注释) |
| miniredis go-redis v9 兼容 | HIGH(版本号 MEDIUM) | 官方 README |
| vjeantet/ldapserver 可用性 | MEDIUM | 社区检索,未跑 PoC;备选 cloudogu fork,隔离在单个 helper 文件内可低成本换 |
| geocoding RoundTripper 零改动注入 | HIGH | 构造器源码实证 |
| Windows 本地 5 包全可跑 | MEDIUM | miniredis/LDAP/httptest 理论跨平台,agent pty 路径可能仍需 skip |

**开放问题 (RESOLVED,2026-08-23 plan-checker 收口):**
1. vjeantet/ldapserver 对 StartTLS/LDAPS 路径的支持范围 — RESOLVED:Phase 75 不涉及 LDAP;addomain 主推线为 LDAPClientIface stub(零新依赖,vjeantet 停更风险不担),本问题整体 defer 到 Phase 76 INFRA-03,届时按 Iface-stub 路线无需回答 ldapserver 支持面。
2. core.Core 完整 Init 图能否用 miniredis + sqlite 拼出 — RESOLVED:defer 到 Phase 78 BLOCK-03(core Init 链 302 stmts 属该 plan 范围);Phase 75 的 Q-7 Stop 幂等修复只需 manager 级单测,不依赖完整 Init 图。
3. device createConnection 是否补生产级 transport 注入 seam — RESOLVED:Phase 75 不加(D-03 范围围栏,ForTesting 工厂已够);生产级 seam 属 Phase 76 INFRA-02 工厂注入重构,届时一并设计。

**来源:** [miniredis](https://github.com/alicebob/miniredis) / [vjeantet/ldapserver](https://github.com/vjeantet/ldapserver) / [go-ldap issue #146](https://github.com/go-ldap/ldap/issues/146) / [cloudogu/go-ldap](https://github.com/cloudogu/go-ldap) + 仓库内 ci.yml、e2e_helpers.go、check-coverage.sh、各 QUIRK 测试实证。


## QUIRK blast radius 逐项(caller 为 grep 实测)

## 5. QUIRK 修复 blast radius(逐项,caller 为 grep 实测)

> 通用过程陷阱:**v1.26 的测试把 QUIRK 行为断言死了**(如 retry_74_12_test.go:46-47
> `assert.True(IsNetworkError("some business error"))`)。每项修复必须同 commit 翻转这些
> "锁定断言"为回归断言,否则 CI 红;这是 15 个修复的共同前置动作。

| # | QUIRK | 修复后行为变化 | 实测 caller(影响面) | 风险 |
|---|-------|----------------|----------------------|------|
| Q-1 | MemoryCache.IncrementBy 缺 key 时 `item.Expiration` nil-deref panic(memory.go:215) | 缺 key 视为 0 起算并建新项(对齐 Redis INCR) | 生产走 Redis(redis.go:166-181)不受影响;MemoryCache 仅 dev/测试。间接解锁 internal/core/captcha.go:264/379/439/445/504 的 dev 路径 | 低 |
| Q-2 | IncrementBy 非数字串静默按 0 累加 | 返回错误(对齐 Redis "not an integer") | 同上;captcha.go 有 `_, _ =` 忽略错误的调用点(:379/:439/:445),不炸;新错误值需进 errors.go 谓词 | 低 |
| Q-3 | ModelExtractor 锚定:仅串首命中,行首/空格分隔 sysDescr 返回空 | 真实形状能提取型号 | internal/services/device_discovery_service.go:607,**:611 有旧回退 ExtractModelFromSysDescr(internal/device/snmp_client.go:400)**。修复后部分设备从"回退结果"变"新提取器结果",model 字段落库值会变 | 中:发现结果 diff,回归测试要含双方言/双路径样本 |
| Q-4 | gmsm v1.4.1 sm2.Decrypt 对合法 base64 非密文 panic(pkg/crypto/sm2_jwt.go:383 DecryptWithSM2 无长度预检) | 前置长度校验返回 error | caller×4:pkg/crypto/request_encryption.go:184/:249(加密中间件)、internal/core/security/jwt.go:302、internal/api/v1/auth.go:559。全部已判 err。行为变化:垃圾密文从 gin Recovery 500 panic 变正常 4xx error | 低,但登录加密 e2e 若有"500"期望需同步改 |
| Q-5 | validateFile 无扩展名 `ext[1:]` 越界 panic | ext=="" → "不支持的文件格式" | 仅 internal/core/captcha_background.go Upload 链 | 低 |
| Q-6 | GetRandomEnabled fallback 查询 PG-only `@>`/jsonb_array_length,sqlite 报 unrecognized token | 按方言分支或空结果短路 | core/captcha_background.go;sqlite dev 从 SQL error 变正常"无可用背景图"分支 | 低 |
| Q-7 | MetricsCacheService.Stop 双调 close 已关 channel panic | sync.Once 幂等 | core/metrics_cache.go;core 关停链 + 测试 defer | 低 |
| Q-8 | USG6000E 正则不含尾字母 → 提取 USG6000 | 正则加 `[A-Z]{0,2}` 尾缀 | device/model_extractor.go;影响发现期 model 值(同 Q-3 链) | 低 |
| Q-9 | nextIP(255.255.255.255) 返回全零 IP 非 nil | To4() 修正后返回 nil,循环终止条件改为 nil 检查 | 仅 internal/device/snmp_client.go:710/726(ScanIPRange);现靠 ipToUint32==0 兜底不发散,修复要同步改循环条件,**否则 nil deref** | 中:两处必须同一 commit 改 |
| Q-10 | retry.containsIgnoreCase 恒 true → IsNetworkError 对长串恒可重试 | strings.Contains+ToLower 精确匹配 | **零生产调用方**(全仓 grep 仅 retry 包自身测试)。风险全在测试翻转:retry_74_12_test.go:42-47/:105-117/:143-149 三处锁定断言 | 低 |
| Q-11 | normalizeParentID 双实现分歧:requests/menu_requests.go:62 塌缩 nil/""/"0";menu_service.go:389 只塌缩 ""(不处理 "0") | 统一为 requests 语义 | menu_service.go:50(Create)/:80(Update)、menu_requests.go:32(ToModel)。**Update 带 ParentID="0" 会落库字面 "0"** → 修复前先查库里 parent_id='0' 行(菜单树孤儿风险),必要时配一条数据迁移 | 中:含数据修正,不只改代码 |
| Q-12 | agent InitLogger 非法 level 降级 info 不返回 err | 返回 error 或至少记录 | internal/agent/server/logger.go;调用方为 agent server 启动链(cmd/agent) | 低 |
| Q-13 | agent NewTLSConfigFromConfig 全空参数不报错仅建空 TLS config | 空参校验 | internal/agent/server(agent client TLS);测试 agent_smoke_test.go:180 锁定断言要翻转 | 低 |
| Q-14 | GetUnifiedDiff 相同配置返回 ""(无 headers) | 返回带 headers 的空 diff 或维持+文档化 | internal/utils/config_diff*(config 对比 UI) | 低 |
| Q-15 | GetDiskInfoDetailed → getDiskInfoByPlatform 递归 → 栈溢出 | 去递归 | internal/pkg/system/disk_info.go;monitor 详细磁盘端点 | 中:修复时避免行为面扩大(见 M-3) |

## 6. Windows-dev + ubuntu-CI(容器/子进程)

| # | 陷阱 | 严重度 | 缓解 |
|---|------|--------|------|
| W-1 | testcontainers 类方案 Windows dev 无 Docker 直接不可用;CI 有 Docker 但 +时长 +flake | 高 | 主路径全部选纯 Go 嵌入件(miniredis / 自写 LDAP / x-crypto ssh fake);容器只做可选 CI 增强 |
| W-2 | agent/server 子进程 stub:`setProcessGroup` 已做平台分支(Linux Setpgid / Windows CREATE_NEW_PROCESS_GROUP,subprocess.go:34-40),但测试用 `os.Args[0]` re-exec 模式时 Windows 下 helper 需 `GO_WANT_HELPER_PROCESS` env + 忽略标志位 | 中 | 标准 Go exec 子进程测试模式(TestHelperProcess);CI linux + dev windows 双跑一次定基线 |
| W-3 | burnCPU 空转在 CI 共享 runner 上可能被 cgroup 限流 → 偶发仍零差 | 低 | burn 200ms×2 已有先例;失败信息里带环境提示,不无限重试 |
| W-4 | 路径/换行:fixture 与 golden 文件 Windows CRLF diff | 中 | .gitattributes 强制 LF;测试读文件用 strings.TrimSpace 比对 |
| W-5 | -race 只在 CI linux 跑(dev windows 默认无 race);Q-1/Q-7 修复引入的并发路径本地验证不到 | 中 | 修复 commit 后本地至少跑一次 `go test -race ./pkg/cache/... ./internal/core/...`(Windows 支持本机 race) |

## 7. 置信度与未决项

- HIGH:miniredis 命令支持矩阵(官方 README)、go-ldap v3.4.12 无 server 子包(module cache
  实证)、scrapligo StandardTransport=x/crypto/ssh + 双 auth 挂载(pinned 源码)、全部
  QUIRK caller 清单(本地 grep)。
- MEDIUM:go-redis CLIENT SETINFO 兼容细节(issue #2911,未在本地复现)、vjeantet/ldapserver
  维护状态(社区信息)、glauth 分页行为。
- LOW/未验证:miniredis XREAD BLOCK 差异(本仓 Go 侧只用 XADD,不受影响)。
- **需 plan-phase 前置决策**:S-4(ScrapliWrapper Driver 工厂注入是 device 70% 的硬前置)、
  Q-11(是否配 parent_id='0' 数据迁移)、Q-3(型号提取行为变化是否需要发现结果对照说明)。


## 已知受影响测试坐标(实证)

**已知受影响测试清单(实证坐标):**

| QUIRK | 锁定测试位置 | 联动效应 |
|-------|-------------|----------|
| MemoryCache.IncrementBy nil-deref + 静默 0 | `pkg/cache/cache_74_08_test.go:131,157`;`internal/core/core_74_08_test.go:127,155,210` | **core 三处 workaround(手工种 captcha 数据)可改为直测 GenerateCaptcha 真实链路 → core 覆盖率直接受益。必须最先修** |
| retry.containsIgnoreCase 恒 true | `internal/agent/pkg/retry/retry_74_12_test.go:42-47,105,144-148` | `TestIsNetworkError`/`TestDoWithRetry_NonRetryableFailsFast`/`TestContainsHelpers` 三处翻转;"长串恒 true"用例改断言 false |
| ModelExtractor 锚定 | `internal/device/device_74_08_test.go:39-45`(行首 QUIRK 空用例) | 同文件 USG 尾字母截断(QUIRK #8,~:80)一并翻 |
| parseTemplateContent $$ 约定 | `internal/templates/templates_74_08_test.go:138-139` + `textfsm_dot1x_test.go` | 行尾 `$$` 锚点语义行为锁定,翻转前先确认期望新语义 |
| utils(operlog background / config diff) | `internal/utils/config_diff_test.go:34`;`operlog/background_74_12_test.go` | diff 为轻度 QUIRK(空串语义),可留最后 |
| gmsm sm2.Decrypt panic | `pkg/crypto/crypto_74_08_test.go:324` | 修复=防御性 recover/长度预检,不翻既有断言、只加新用例 |
| validateFile 无扩展名 panic | `internal/core/core_74_08_test.go:413`(子用例曾被**移除**) | 恢复被移除的子用例 |

**排序原则:** 被其他包测试 workaround 依赖的 QUIRK(MemoryCache)最优先;纯语义翻转(ModelExtractor/retry)其次;防御性加固(gmsm)最后。



## 修复顺序硬约束(roadmap SC + 研究结论)

1. **QUIRK-01 MemoryCache.IncrementBy = 75-01 = 全 milestone 第一个 plan**:
   缺 key 按 0 起算返回 1(Redis INCR 语义)+ 非法串 error;连锁解锁 core 三处
   captcha workaround(core_74_08_test.go:127,155,210)
2. Q-7(MetricsCacheService.Stop 幂等)+ Q-3/Q-8(device 提取器新行为)修复
   **先于 Phase 78**(core Close 收尾/设备包新测试直接按正确行为断言)
3. Q-9(nextIP nil)必须与 ScanIPRange 循环条件**同 commit**(单独改任一侧发散)
4. Q-11 修复前**先实测存量 parent_id='0' 行数**,迁移与代码统一同一交付面
5. Q-15 修复行为面最小化(M-3);M-2 cpu_linux 顺手加 mutex 去递归(行为不变)
6. Q-1/Q-7 修复后本地跑 `go test -race ./pkg/cache/... ./internal/core/...`

## 验收信号(Phase 75 SC 摘要)

- captcha 三处 workaround 删除,GenerateCaptcha 真实链路直测
- 15 个原子 commit `fix(quirk-N): ...`,每个 commit 点 4 层 gate + diff ≥80% 绿
- Q-11 迁移后菜单树查询无孤儿节点
