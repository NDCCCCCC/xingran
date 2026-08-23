---
phase: 76
slug: test-doubles
status: verified
threats_open: 0
asvs_level: 1
created: 2026-08-23
---

# Phase 76 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Phase 性质：** 测试基建阶段（test doubles + 注入缝）。引入 miniredis/httpmock test-only 依赖，落地 Driver 工厂 var、LDAPClientIface 扩容（16→20）、clientFactory 注入字段、TestHelperProcess re-exec、ForTesting AST 守护。无用户可见生产行为变更。

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| 供应链（go.mod 新增外部模块） | miniredis/httpmock 及 indirect 依赖进入构建图（仅测试侧） | 包名 + 精确版本 pin（go.sum hash 锁定） |
| 测试进程内（miniredis TCP / httpmock Transport） | 无外部输入；fixture 全 dummy 值 | dummy 字面量（"ak-test"/测试地址/示例坐标） |
| 生产代码 ↔ 测试注入 | 包级 var `newNetworkDriver` / `clientFactory` 字段是唯一注入点；生产默认实现与原行为一致 | 函数值替换（仅测试进程内） |
| 测试父进程 ↔ re-exec 子进程 | 子进程 = 测试二进制自身，经 `GO_WANT_HELPER_PROCESS` env + `-test.run` 区分形态 | 环境变量（无安全语义，t.Setenv 自动恢复） |
| 生产代码 ↔ 测试基建符号 | `ForTesting` 后缀符号（跳过簿记/锁/可达性检查）禁止被生产代码引用 | 符号引用（AST 守护扫描） |
| 仓库 ↔ worktrees 副本 | `.claude/worktrees` 下陈旧完整仓库拷贝必须从守护扫描中隔离 | 文件系统路径 |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-76-01-SC | Tampering（供应链投毒） | go.mod 依赖引入 | mitigate | 精确 pin `miniredis/v2 v2.38.0`、`httpmock v1.4.2`（`go.mod:8,17`，`// test-only (v1.27 D-02)` 注释）+ go.sum hash + 合法性审计（76-01-SUMMARY） | closed |
| T-76-01-02 | Information Disclosure | 冒烟/PoC 测试文件 | accept | 全 dummy 值（"ak-test"/测试地址/示例坐标 116.404/39.915）；密钥模式扫描零命中 | closed |
| T-76-01-03 | Tampering（生产依赖漂移） | go.mod 生产 require 块 | mitigate | 独立 diff `e3e9b04^..HEAD -- go.mod`：仅 +miniredis/+httpmock（test-only 标注）+ gopher-lua indirect + 两个版本不变的分类移动；零生产依赖增删改 | closed |
| T-76-02-01 | Tampering（生产行为漂移） | scrapli_wrapper.go 重构 | mitigate | `scrapli_wrapper.go:115` 包级 var；错误文案 byte 不变（:122 创建平台实例失败 / :127 获取网络驱动失败）；两构造器经工厂（:172,:226）；`platform.NewPlatform` 直调 1 处（:116）；device 包测试复跑 ok | closed |
| T-76-02-02 | Elevation（注入缝被生产滥用） | newNetworkDriver var | accept | 小写包私有，跨包不可触达；76-05 AST 守护兜底 | closed |
| T-76-02-03 | Information Disclosure | testdata fixture | mitigate | `huawei_vrp_open.fixture` 全 dummy（dummy-host 提示符）；测试设备 dummy-host/u/p（`driver_factory_76_02_test.go:83-88`）；无真实 IP/凭据 | closed |
| T-76-02-04 | DoS（测试挂起） | FileTransport Close | mitigate | no-Close 纪律成文（`driver_factory_76_02_test.go:15-21` 引 port_write 先例 + :115 Intentionally no Close）；全测试无 Close 调用 | closed |
| T-76-03-01 | Tampering（24 处替换漂移） | *LDAPClient → LDAPClientIface | mitigate | `ldap_iface.go:44-51` 计划方法 + SearchWithRequest（编译门逃生条款）+ :56 编译期断言；双 grep 零残留复跑均零命中；接口使用确认于 scheduler/dept_sync_tasks.go:183 等 | closed |
| T-76-03-02 | Elevation（PickFirstConnect 误接口化） | ad 认证链 | mitigate | `failover_client.go:99` 仍返回具体类型 `(*LDAPClient, ...)`；:117 直调 NewLDAPClient（未走 f.newClient） | closed |
| T-76-03-03 | Repudiation（failover 簿记绕过） | AccountPool MarkFailure/MarkSuccess | mitigate | `failover_client_76_03_test.go:61-66` 断言 failure_count=1 落库；:68-74 断言 MarkSuccess 路径（failure_count=0 AND last_success_at NOT NULL） | closed |
| T-76-03-04 | Information Disclosure | mock preset 数据 | accept | dummy example.com DN / u1-u3 / username-N / 通用错误文案；密钥模式扫描零命中 | closed |
| T-76-04-01 | DoS（go test 挂起） | TestHelperProcess 守卫缺失 | mitigate | `subprocess_stub_test.go:26-28` 守卫首行 + :51 os.Exit(0) 全分支必达 + :34 linux 守卫；agent/server 包测试复跑 ok 无挂起 | closed |
| T-76-04-02 | Tampering（生产代码波及） | subprocess.go | mitigate | `git diff e3e9b04^..HEAD` 对 subprocess.go/sysproc_*.go 为空（最后触碰 = repo-init）；文件零测试钩子/env 处理 | closed |
| T-76-04-03 | Elevation（env 串扰） | t.Setenv 作用域 | mitigate | `subprocess_pgroup_test.go:45,57,73` t.Setenv 自动 Cleanup；internal/agent/server 全包 t.Parallel 零匹配 | closed |
| T-76-04-04 | Information Disclosure | 子进程输出 | accept | 输出仅字面量 "hello"/"line"/"still-alive"/"sigterm-armed"（:40,:42,:45,:49） | closed |
| T-76-05-01 | Elevation（测试替身泄漏进生产） | ForTesting 符号误用 | mitigate | `for_testing_guard_test.go` 白名单（:85-88 filepath.Rel 归一化）+ CallExpr（:98-103）/SelectorExpr（:104-109）双分支；auditor 实跑 712 文件 0 违规；注毒自证记录于 76-05-SUMMARY | closed |
| T-76-05-02 | Tampering（守护假绿） | 扫描器缺陷 | mitigate | :118-120 scannedFiles==0 → t.Fatal；:91-93 parse 失败报错 → :115-117 t.Fatalf（fail-not-skip）；双毒株注毒自证 | closed |
| T-76-05-03 | DoS（守护随机红） | worktrees 副本误报 | mitigate | :69 点前缀目录 SkipDir（.claude/worktrees/.git）+ :70-74 vendor/node_modules/frontend/testdata/tests + :65-67 根目录豁免 | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-76-01 | T-76-01-02 | miniredis 冒烟 + geocoding httpmock PoC 测试数据全 dummy（"ak-test"、测试地址、示例坐标）；密钥模式扫描（api_key/password/secret/token/IP 字面量）零命中，2026-08-23 复核 | PLAN 76-01 (plan-time) / gsd-security-auditor (复核) | 2026-08-23 |
| AR-76-02 | T-76-02-02 | `newNetworkDriver` 注入缝为包私有（小写 var），跨包生产滥用不可能；76-05 AST 守护作为纵深防御已激活 | PLAN 76-02 (plan-time) / gsd-security-auditor (复核) | 2026-08-23 |
| AR-76-03 | T-76-03-04 | mockLDAPClient preset 全为 dummy ldap.Entry/DN（example.com DN、u1-u3），无真实 AD 身份信息 | PLAN 76-03 (plan-time) / gsd-security-auditor (复核) | 2026-08-23 |
| AR-76-04 | T-76-04-04 | re-exec 子进程输出仅惰性字面量（"hello"/"line"/"still-alive"/"sigterm-armed"），无敏感数据 | PLAN 76-04 (plan-time) / gsd-security-auditor (复核) | 2026-08-23 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-23 | 18 | 18 | 0 | gsd-security-auditor (standard) |

**Auditor 独立验证（超出 SUMMARY 声明的复核）：**
- go.mod pre-phase diff（e3e9b04^..HEAD）：生产 require 块零漂移
- 双 grep 零残留验收复跑：均零命中
- ForTesting AST 守护实跑：712 files / 0 violations
- `internal/device`、`internal/agent/server` 包测试复跑：均 ok
- subprocess.go 阶段 diff 为空；t.Parallel 全包缺失确认；密钥模式扫描零命中
- 5 个 SUMMARY 均无 `## Threat Flags` 未注册攻击面声明

---

## Sign-Off

- [x] All threats have a disposition (mitigate 14 / accept 4)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter
