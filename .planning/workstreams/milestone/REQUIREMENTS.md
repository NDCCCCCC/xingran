---
milestone: v1.27
milestone_name: 后端测试覆盖率优秀 II
defined: 2026-08-23
sources:

  - .planning/research/v1.27-stack.md
  - .planning/research/v1.27-features.md
  - .planning/research/v1.27-architecture.md
  - .planning/research/v1.27-pitfalls.md

---

# Milestone v1.27 Requirements

**Goal:** 加权平均覆盖率 55.60% → **≥70%**(收掉 v1.26 SC-a 缺口 6287 stmts,含数学修正后的 TAIL 长尾),5 结构阻塞包 + 长尾包逐一 ≥70%,15 项 QUIRK 全部修复。

**数学校验基线**(2026-08-23 gate 实测):24269/43652 = 55.60%;70% 需 30556;缺口 6287 = BLOCK ~2402 + TAIL ~3885。不含 TAIL 目标必失守(v1.26 SC-a 覆辙预防)。

## 基建 (INFRA)

- [x] **INFRA-01**: 引入 miniredis/v2 (v2.38+) 与 httpmock (v1.4.x) 两个 test-only 依赖(MIT;redismock 硬淘汰——锁死 go-redis v8;testcontainers 不引入——Windows 无 Docker 断裂本地测试)。miniredis 三坑防护:TTL 用 FastForward / INFO 断言降级 / go-redis v9.5+ CLIENT SETINFO 兼容
- [x] **INFRA-02**: ScrapliWrapper 新增可注入 Driver 工厂入口(小重构,生产路径不变;pitfalls 实证 StandardTransport 即 x/crypto/ssh v0.46 零新增模块,fake server 需输出 prompt)
- [x] **INFRA-03**: addomain 走 LDAPClientIface 扩展 stub 主推线(零新依赖);嵌入式 vjeantet/ldapserver 停更风险不担
- [x] **INFRA-04**: agent 子进程 stub 统一 os/exec TestHelperProcess re-exec 模式(替换 exec.Command("echo") Windows/CI 分歧根源)
- [x] **INFRA-05**: 测试隔离治理:沿用 e2e_helpers A1(ForTesting 后缀+无 build tag) + AST 守护测试(仿 status_constants_test.go)

## 阻塞包攻破 (BLOCK)

- [ ] **BLOCK-01**: `internal/services/operations` ≥70%(缺 ~330;全 (c) 类纯补测试:workstation_device 445 + excel_service 399,sqlite+excelize,零基建依赖可先行)
- [x] **BLOCK-02**: `internal/agent/server` ≥70%(缺 ~295;platformStrategy 接口 + backendURL 参数 + httptest 先例)
- [x] **BLOCK-03**: `internal/core` ≥70%(基线 43.7% → 78-01 收 54.2%(captcha/metrics 链) → 78-02 收 82.5%(Init/Close 装配链 + 各阶段产物);Init 链 302 stmts + Close 60 stmts 由 78-02 全覆盖,核心链 8 initXxx 阶段 + Close 顺序/幂等/半装配 + reaper/RPA 全锁;实测 24 TestInit78_ 用例全绿,QUIRK-78-02-P1(二次 Close panic)+ P2(DeviceConnectionPool 1 goroutine 泄漏)记入 Phase 79/80 长尾)
- [x] **BLOCK-04**: `internal/device` ≥70%(基线 69.2% → 78-03 收 scrapli ~88%/executor ~75%/pool 89.9% → 78-04 收 snmp ~50%(lightweight)/scheduler 94.6%;FileTransport D-78-05 pre-seed 路径解锁;Windows loopback snmp 跨 socket 响应丢弃降级 error-path;P2_RATCHET_device 豁免行可删)
- [ ] **BLOCK-05**: `internal/services/addomain` ≥70%(基线 23.1% → 78-05 收 sync.go 83.9% → 78-06 收 computer 96.1%/ou_group_mapping 88.6%/group_config 86.7%/config 83.0%/account_pool 82.0% → 78-07 收 failover 88.9%/user 63%/group 67%;ldap_client ~36% → **实测 58.0%**;LDAP responder BER 不兼容 go-ldap/v3 导致 Conclusion B,ldap_client ~180 stmts 不可达;12pp gap 留 Phase 79 接手)

## 长尾补齐 (TAIL)

- [ ] **TAIL-01**: `internal/services`(root,5202 stmts @11.3%)补 ~3052 — 遗留 cache services 群(dept/role/dict/menu/user/post)+ token blacklist 等,sqlite 可测非结构阻塞(v1.26 从未进 P0/P1/P2 名单的最大隐藏缺口)
- [ ] **TAIL-02**: `internal/scheduler`(1103 @3.3%)补 ~736 — 内部 cron 引擎,注册表/执行器可测
- [ ] **TAIL-03**: 碎包合计 ~831:api/v1(366)+ models(310)+ internal/api(291;装配层仅可测纯函数段)+ pkg/errors(183)+ pkg/cache(161,Redis 路径经 miniredis)+ 其余 <50 的小尾巴(permission/websocket/base/lldp/gormutil/middleware/logger/query)

## QUIRK 修复 (QUIRK)

- [x] **QUIRK-01**: MemoryCache.IncrementBy 最先修(nil-deref panic + 非法字符串静默 0)——core_74_08_test.go 三处 captcha workaround 连锁解锁,DB 语义 INCR 缺键=1
- [x] **QUIRK-02**: 其余 14 项全修,每项**同 commit** 翻转 v1.26 锁定断言 + 回归测试 + 原子 commit:ModelExtractor 锚定(Q-3,发现落库 model 值会变,有 ExtractModelFromSysDescr 回退 caller)/ sm2.Decrypt 长度预检 / validateFile 无扩展名 / retry.containsIgnoreCase(retry 包零生产调用方,影响面=0)/ GetRandomEnabled PG-only fallback / MetricsCacheService.Stop 幂等 / nextIP 全零形态(须与 ScanIPRange 循环条件同 commit)等
- [x] **QUIRK-03**: Q-11 normalizeParentID 双实现分歧修复 + 存量数据迁移(Update 路径落库字面 "0" 的行归一为 NULL)

## 收口防线 (GATE)

- [ ] **GATE-01**: 加权平均 ≥70%(43652 stmts 口径;SC-a 收口)
- [ ] **GATE-02**: ratcheted floor 解除——core/device/agent-server 达标后删除 check-coverage.sh 对应 P2_RATCHET 行,回落 70% 全量 floor(UP-only 语义闭环)
- [ ] **GATE-03**: 4 层 gate + PR diff coverage 全程绿;QUIRK 业务变更经 PR diff coverage ≥80% 把关(v1.26 防线不倒退)

---

## Traceability

| REQ | Phase | Plans | Status |
|-----|-------|-------|--------|
| QUIRK-01 | Phase 75 | 75-01 | Complete |
| QUIRK-02 | Phase 75 | 75-02, 75-03, 75-04, 75-05 | Complete |
| QUIRK-03 | Phase 75 | 75-06 | Complete |
| INFRA-01 | Phase 76 | 76-01 | Pending |
| INFRA-02 | Phase 76 | 76-02 | Pending |
| INFRA-03 | Phase 76 | 76-03 | Pending |
| INFRA-04 | Phase 76 | 76-04 | Pending |
| INFRA-05 | Phase 76 | 76-05 | Pending |
| BLOCK-01 | Phase 77 | 77-01, 77-02, 77-03 | Pending |
| BLOCK-02 | Phase 77 | 77-04, 77-05 | Pending |
| BLOCK-03 | Phase 78 | 0fd3694 | Done (78-02) |
| BLOCK-04 | Phase 78 | TBD | Pending |
| BLOCK-05 | Phase 78 | TBD | Pending |
| TAIL-01 | Phase 79 | TBD | Pending |
| TAIL-02 | Phase 80 | TBD | Pending |
| TAIL-03 | Phase 80 | TBD | Pending |
| GATE-01 | Phase 81 | TBD | Pending |
| GATE-02 | Phase 81 | TBD | Pending |
| GATE-03 | Phase 81 | TBD | Pending |

Unmapped: 0 ✓

---
*Requirements defined: 2026-08-23 (4-scope decisions confirmed: addomain Iface-stub / device factory-inject refactor / Q-11 fix+migration / TAIL included per gap-math correction)*
