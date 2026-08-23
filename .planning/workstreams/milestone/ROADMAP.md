---
last_updated: 2026-08-23
update_trigger: v1.27 roadmap created — 7 phases (75-81) / 19 reqs 100% mapped / QUIRK-first sequencing (IncrementBy 最先解锁 core captcha)
---

# Roadmap: XingRan-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联 + 部门物理位置映射** — Phases 28 + 39 (shipped 2026-06-10 / 06-25)
- ✅ **v1.16 技术债清理 (Tech-Debt Cleanup)** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账 (Asset Reconciliation)** — Phases 42-46 + Phase 47 root-cause (shipped 2026-07-03)
- ✅ **v1.18 网络设备硬件清单 (Device Component Serials)** — Phase 48 + Phase 49 gap closure (shipped 2026-07-04)
- ✅ **v1.19 网络设备写命令 (Network Device Port Write Operations)** — Phases 50-55 (shipped 2026-07-08)
- ✅ **v1.20 网络设备 VLAN + 端口绑定 (Network Device VLAN + Port Binding)** — Phase 56 (shipped 2026-07-10)
- ✅ **v1.21 API Key 认证链修复 + 能力补全 (API Key Auth Chain Repair)** — Phases 57-62 (shipped 2026-08-18)
- ✅ **v1.22 前端品牌化改造 (Frontend Brand Design-System)** — Phases 64-67 (shipped 2026-08-18)
- ✅ **v1.23 部署稳健性 & 文档一致性 (Deploy Robustness & Docs Consistency)** — Phase 68 (shipped 2026-08-19)
- ✅ **v1.24 字典与状态值治理 (Dict & Status Governance)** — Phase 69 (shipped 2026-08-19)
- ✅ **v1.25 系统设置页面布局重构 (Settings Page Layout Redesign)** — Phase 70 (shipped 2026-08-19)
- ✅ **Phase 63 前端工具链自动化 (Frontend Toolchain Automation)** — Phase 63 (shipped 2026-08-20)
- ✅ **v1.26 后端测试覆盖率优秀 (Backend Test Coverage Excellence)** — Phases 71-74 — **SHIPPED 2026-08-22** (weighted 12.8→55.5, 0%-pkg 33→5, 4-layer gate + diff coverage live; SC-a shortfall honestly documented; phase 详情已归档 `.planning/milestones/v1.26-phases/`)
- **v1.27 后端测试覆盖率优秀 II (Backend Test Coverage Excellence II)** — Phases 75-81 — **IN PROGRESS 2026-08-23**(weighted 55.60 → ≥70, 5 结构阻塞包 + TAIL 长尾逐一 ≥70, 15 项 QUIRK 全修, miniredis/httpmock 零 Docker 基建)

---

## Current Milestone: v1.27 后端测试覆盖率优秀 II (Backend Test Coverage Excellence II) — IN PROGRESS

**Goal:** 后端 Go 加权平均测试覆盖率 **55.60% → ≥70%**(收掉 v1.26 SC-a 缺口 6287 stmts,含数学修正后的 TAIL 长尾),5 结构阻塞包 + 长尾包逐一 ≥70%,15 项 QUIRK 全部修复。

**数学校验基线**(2026-08-23 gate 实测):24269/43652 = 55.60%;70% 需 30556 covered;缺口 6287 = BLOCK ~2402 + TAIL ~3885。不含 TAIL 目标必失守(v1.26 SC-a 覆辙预防)。

**Source planning data:**
- `.planning/research/v1.27-features.md`(5 阻塞包未覆盖语句地形图 + 投入产出排序: operations > agent/server > core > device > addomain;零基建可先行 stmts ≈1870)
- `.planning/research/v1.27-architecture.md`(基建接入架构: QUIRK-01 必须最先修、零 Docker 进程内替身、A1 隔离契约 + AST 守护)
- `.planning/research/v1.27-stack.md`(仅 2 个新 test-only 依赖: miniredis/v2 v2.38+ 与 httpmock v1.4.x)
- `.planning/research/v1.27-pitfalls.md`(miniredis 三坑 / fake SSH 三坑 / 15 项 QUIRK blast radius / Q-11 需数据迁移)

**Milestone success criteria:**

- SC-a: 加权平均 ≥70%(43652 stmts 口径;数学 30556/43652)
- SC-b: 5 结构阻塞包(operations / agent-server / core / device / addomain)逐一 ≥70%
- SC-c: 15 项 QUIRK 全部修复(每项原子 commit + 同 commit 翻转断言 + 回归用例)
- SC-d: TAIL 长尾清零(services root / scheduler / 碎包 ≥70%,堵住 v1.26 数学修正揭示的隐藏缺口)
- SC-e: ratchet 闭环(`.coverage-threshold` 55.5 → ≥70 实测值;P2_RATCHET 豁免行删除回落全量 floor;4 层 gate + diff coverage 全程绿)

**Phase 编号:** 从 v1.26 末尾 Phase 74 续编(75+)。

**排序原则(研究结论锁定):** QUIRK 修复先于覆盖补齐——被 v1.26 测试 workaround 依赖的 QUIRK-01 最优先(连锁解锁 core captcha 真实链路),Q-7 Stop 幂等是 core Init Close 收尾前置,Q-3/Q-8 提取器修正好让 device 新测试从第一天断言正确行为;基建(miniredis/工厂/iface/re-exec)第二批落地;随后零基建阻塞包与基建解锁阻塞包分两批攻破;TAIL 两批清欠;最后镜像 74-11 收口。

### Phase Dependency Graph

```
Phase 75 (QUIRK 全修 — IncrementBy 全场最先)
  └→ Phase 76 (测试基建: miniredis/httpmock + 4 类注入缝)
        ├→ Phase 77 (阻塞包·零基建: operations + agent/server)   [depends 76]
        ├→ Phase 78 (阻塞包·基建解锁: core + device + addomain)   [depends 75 + 76]
        ├→ Phase 79 (长尾 I: internal/services root)              [depends 76; 与 77/78 可并行穿插]
        └→ Phase 80 (长尾 II: scheduler + 碎包)                    [depends 76; 与 77-79 可并行穿插]
Phase 77-80 全部完成
  └→ Phase 81 (全仓收口: ratchet ≥70 + P2_RATCHET 删除 + audit)   [depends 75-80 全部]
```

每个 phase 是自然交付边界,phase boundary 处 `go test ./...` 全绿 + gate 不倒退。

### Phase 75: QUIRK 行为修正 (15 项业务怪癖关闭)

**Goal:** 按"修复 + 同 commit 翻转断言 + 回归用例 + 原子 commit"五步法关闭全部 15 项 QUIRK(MemoryCache.IncrementBy 全场最先,Q-11 连带存量数据迁移),让业务行为先归正、后续所有新测试从第一天起断言正确行为。

**Depends on**: Nothing (first phase;plan 75-01 必须是整个 milestone 第一个执行的 plan)

**Requirements**: QUIRK-01, QUIRK-02, QUIRK-03

**Success Criteria** (what must be TRUE):

1. `MemoryCache.IncrementBy` 缺 key 时按 0 起算并新建缓存项(返回 1,对齐 Redis INCR 语义),非法数字串返回 error 而非静默按 0 累加;`pkg/cache/cache_74_08_test.go:131,157` 与 `internal/core/core_74_08_test.go:127,155,210` 的锁定断言同 commit 翻转
2. core 的 3 处 captcha workaround(手工种 captcha 数据)删除,`GenerateCaptcha` 真实链路在测试中可直测(QUIRK-01 连锁解锁的验收信号)
3. 其余 QUIRK 逐项关闭且行为可观察:型号提取命中行首/空格分隔 sysDescr(Q-3)与 USG6000E 尾字母(Q-8);sm2.Decrypt 对垃圾密文返回 error→HTTP 4xx 而非 panic 500(Q-4);validateFile 无扩展名返回"不支持的文件格式"(Q-5);GetRandomEnabled 在 sqlite 下走空结果分支而非 SQL error(Q-6);MetricsCacheService.Stop 双调幂等(Q-7);nextIP(255.255.255.255) 返回 nil 且 ScanIPRange 循环条件同 commit 改 nil 检查(Q-9);retry.containsIgnoreCase 精确匹配(Q-10);InitLogger 非法 level 报错(Q-12);TLS 全空参数报错(Q-13);GetDiskInfoDetailed 不再递归栈溢出(Q-15)
4. Q-11:normalizeParentID 双实现统一为 requests 语义(nil/""/"0" 全塌缩),存量 `parent_id='0'` 行经数据迁移归一为 NULL,迁移后菜单树查询无孤儿节点
5. 每项 QUIRK 一个原子 commit(`fix(quirk-N): <语义> + 回归测试`),每个 commit 点 CI 全绿(4 层 gate + diff coverage ≥80% 把关业务变更)

**Plans**: TBD(建议 5)
- 75-01 QUIRK-01 MemoryCache.IncrementBy(nil-deref + 非法串 error)+ 翻转断言 + 删 3 处 captcha workaround — 全 milestone 第一个 plan
- 75-02 device 家族:Q-3 锚定 + Q-8 尾字母 + Q-9 nextIP/ScanIPRange 同 commit
- 75-03 core 防御家族:Q-4 sm2 长度预检 + Q-5 validateFile + Q-6 PG-only fallback + Q-7 Stop 幂等
- 75-04 agent/utils 家族:Q-10 retry + Q-12 InitLogger + Q-13 TLS 空参 + Q-14 GetUnifiedDiff + Q-15 磁盘递归
- 75-05 Q-11 normalizeParentID 统一 + parent_id='0' 存量数据迁移

**Notes**:

- 五步法与锁定断言坐标清单见 architecture 研究 Q4 表(43 处 QUIRK 注释命中)
- Q-1/Q-7 修复后本地至少跑一次 `go test -race ./pkg/cache/... ./internal/core/...`(W-5:Windows 支持本机 race)
- Q-11 修复前先实测存量 `parent_id='0'` 行数,迁移与代码统一同一交付面
- Q-3 回归测试须含双路径样本(新提取器 + 既有 ExtractModelFromSysDescr 回退 caller,落库 model 值会变);Q-15 修复行为面最小化(M-3);Q-10 retry 零生产调用方,风险全在测试翻转
- M-2:cpu_linux 包级全局无锁——顺手加 mutex 去递归(行为不变)

---

### Phase 76: 测试基建落地 (test doubles + 注入缝)

**Goal:** 引入 miniredis/httpmock 两个 test-only 依赖并落地全部注入缝(Driver 工厂 / LDAPClientIface / re-exec stub / AST 守护),零 Docker、Windows 本地与 ubuntu CI 同构,使 5 个结构阻塞包的覆盖工作不再被"没有真实依赖基建"卡死。

**Depends on**: Phase 75(顺序保证 QUIRK-01 全场最先;内容上无硬依赖)

**Requirements**: INFRA-01, INFRA-02, INFRA-03, INFRA-04, INFRA-05

**Success Criteria** (what must be TRUE):

1. go.mod 仅新增 2 个 test-only 依赖(miniredis/v2 v2.38+ 与 httpmock v1.4.x,MIT,require 行带 `// test-only (v1.27 D-02)` 注释),生产依赖零变更,`go test ./...` Windows 本地 + ubuntu CI 双绿且全程无 Docker
2. miniredis 三坑防护落地:TTL 断言用 FastForward(R-1)/ INFO 断言降级为 err==nil + key 存在性(R-2)/ CLIENT SETINFO 兼容(R-3);`pkg/cache` Redis 路径(INCR/EXPIRE/SCAN/HSET/INFO/EVAL)有冒烟测试实证命令面
3. ScrapliWrapper 具备可注入 Driver 工厂入口(生产路径行为不变,现有测试全绿),测试可注入 FileTransport/自定义 transport 进入 Open/SendCommand 链
4. LDAPClientIface mock 补全(walk/分页语义),FailoverClient 顺序遍历/maxHops 可经接口驱动;agent 子进程 stub 统一为 TestHelperProcess re-exec 模式,`exec.Command("echo")` 类 Windows/CI 平台分歧根源清除
5. AST 守护测试上线:扫描生产 .go 文件禁止引用 `*ForTesting` 符号(仿 status_constants_test.go),测试隔离契约由编译器(_test.go)+ 命名(ForTesting 后缀)+ AST 三层保证

**Plans**: TBD(建议 5)
- 76-01 miniredis + httpmock go.mod 落地 + pkg/cache Redis 冒烟(含三坑防护用例)
- 76-02 ScrapliWrapper Driver 工厂注入重构(生产路径不变)
- 76-03 LDAPClientIface mock 补全 + FailoverClient 接口 seam
- 76-04 TestHelperProcess re-exec helper + 替换 exec.Command("echo") 先例
- 76-05 AST 守护测试(ForTesting 隔离契约)

**Notes**:

- redismock(锁死 go-redis v8)/ gock(休眠)/ testcontainers(Docker)明令禁止引入
- `_test.go` import `golang.org/x/crypto/ssh` 会移除 go.mod 的 `// indirect` 注释——属测试可见性变化非生产依赖新增,SUMMARY 显式说明以免误判违反 D-02
- fake SSH server 的实现留给 Phase 78 device plan,本 phase 只交付工厂入口;httpmock 使用纪律(DeactivateAndRestore / 独立 MockTransport 二选一)在本 phase 成文

---

### Phase 77: 阻塞包攻破·零基建先行 (operations + agent/server)

**Goal:** 用既有 sqlite/httptest/假策略先攻投入产出排名前二的 operations 与 agent/server,双双越过 70% 线。

**Depends on**: Phase 76(INFRA-04 re-exec 供 agent;operations 各 plan 零基建硬依赖,必要时可提前执行)

**Requirements**: BLOCK-01, BLOCK-02

**Success Criteria** (what must be TRUE):

1. `internal/services/operations` ≥70%(61.1% → ≥70%,补 ~330 stmts;workstation_device_service 445 unc + excel_service 399 unc 为主力,sqlite+excelize 零基建)
2. `internal/agent/server` ≥70%(22.1% → ≥70%,补 ~295 stmts;jwt_auth/connection_manager 走 httptest 假后端,account_manager 走假策略 + re-exec stub)
3. Windows 本地与 ubuntu CI 的 agent 包覆盖率差 <2pp(env-branch divergence 消除,P2_RATCHET 注释记录的 22.08 vs 19.48 问题收口)
4. phase 边界 `go test ./...` 全绿,gate(weighted-avg / P1 floor / P2 floor / diff coverage)不倒退

**Plans**: TBD(建议 5)
- 77-01 workstation_device_service(GetADDevices/SyncFromAD/SyncFromAsset/mergeBySerial/SetPrimaryDevice*)
- 77-02 excel_service 导出链(ExportData/queryData/writeDataRows/writeInstructions/appendWorkstationDeviceSheets)
- 77-03 excel_service 导入剩余 + reference_resolver + workstation/floor/code_generator/excel_raw_rows
- 77-04 agent jwt_auth + connection_manager(httptest 假后端,backendURL 明文参数)
- 77-05 agent handlers(gin + Recorder)+ config 校验/注册 + account_manager(假策略上层 + re-exec 真策略体)

**Notes**:

- geocoding 已有 fakeGeocodeTransport(RoundTripper)先例,httpmock 在本包仅边际价值
- Q-12/Q-13 已在 Phase 75 修复,agent config 测试直接断言新行为(非法 level 返回 error / TLS 空参报错)
- pty_manager 真 pty 路径(18 stmts)允许平台 Skipf 兜底,不影响 70% 达线

---

### Phase 78: 阻塞包攻破·基建解锁 (core + device + addomain)

**Goal:** 用 Phase 75/76 的修复与基建把 core、device、addomain 全部推过 70%,5 结构阻塞包清零。

**Depends on**: Phase 75(Q-7 Stop 幂等是 core Init Close 收尾前置;Q-3/Q-8 修正好让 device 提取器测试断言新行为), Phase 76(INFRA-01 miniredis / INFRA-02 Driver 工厂 / INFRA-03 LDAPClientIface)

**Requirements**: BLOCK-03, BLOCK-04, BLOCK-05

**Success Criteria** (what must be TRUE):

1. `internal/core` 根包 ≥70%(40.2% → ≥70%;Init 链 302 stmts 经 miniredis+sqlite config 分支跑通并以 Close 收尾;captcha 98 stmts 纯补 + QUIRK-01 解锁的真实链路直测)
2. `internal/device` ≥70%(39.1% → ≥70%;FileTransport 照搬 portwrite 先例解锁 scrapli_wrapper/connection_pool/executor 346 stmts,x/crypto/ssh fake 补 Open/transport 路径,snmp UDP 对端 + task_scheduler 并发/取消分支)
3. `internal/services/addomain` ≥70%(21.8% → ≥70%,补 ~1165 stmts;两段式:sqlite+`[]*ldap.Entry` 段 ~535 先行,Iface stub 段 ldap_client 159 + failover 入口收尾)
4. check-coverage.sh 三条 P2_RATCHET 豁免行(core 38.33 / device 39.07 / agent-server 22.08)对应包全部实测 ≥70%(豁免行的删除动作统一留 Phase 81 收口)
5. 每个 plan 完成点 `go test ./...` 全绿,gate 不倒退

**Plans**: TBD(建议 7)
- 78-01 core captcha 真实链路(QUIRK-01 解锁)+ captcha_background(文件+DB)+ metrics_cache 边缘
- 78-02 core Init 链(miniredis+sqlite+reaper re-exec+Close 收尾;plan 首任务为 Init 可跑深度探针实验)
- 78-03 device scrapli_wrapper + connection_pool + executor(FileTransport + fake SSH,fake server 输出 prompt)
- 78-04 device snmp_client UDP 对端 + task_scheduler 剩余分支
- 78-05 addomain sqlite 段 A:sync.go 全链(`[]*ldap.Entry` 驱动,syncOUs/syncGroups/syncUsers/syncGroupMembers)
- 78-06 addomain sqlite 段 B:computer.go + ou_group_mapping/group_config/config 纯 CRUD + account_pool 剩余(MarkSuccess/RecoverExpiredBreakers)
- 78-07 addomain Iface stub 段:ldap_client 参数/错误分支 + FailoverClient 遍历 + user.go/group.go failover 入口

**Notes**:

- core Init 探针实验(research 数据缺口):scheduler goroutine / device pool 初始化副作用未知,78-02 首任务验证
- fake SSH 三坑:session channel 必须在 pty-req+shell 后输出 `<host>#` 提示符否则 hang(S-2)/ 拒绝时返回 (nil,false) 而非 error(S-3)/ ExtraCiphers 静默忽略不影响默认协商
- MultiLevelCache 构造即启 L2WriteWorker goroutine(R-7)——测试用 NewMultiLevelCacheSimple 或 t.Cleanup Close,防 -race 泄漏
- addomain StartTLS/LDAPS 分支若 stub 不可达:明文 ldap:// 分支覆盖 + TLS 分支降级断言(architecture 开放问题 1);嵌入式 vjeantet/ldapserver wire 线仅在 stub 段不足 70% 时按需立项(零新依赖主推线不动摇)

---

### Phase 79: 长尾清欠·internal/services root

**Goal:** 补齐 v1.26 从未进 P0/P1/P2 名单的最大隐藏缺口 internal/services root(5202 stmts @11.3%,补 ~3052 stmts)到 ≥70%。

**Depends on**: Phase 76(miniredis 供 legacy cache services 的 Redis 路径;与 Phase 77/78 无硬依赖,可并行穿插)

**Requirements**: TAIL-01

**Success Criteria** (what must be TRUE):

1. `internal/services`(root)包覆盖率 ≥70%,legacy cache services 群(dept/role/dict/menu/user/post)逐文件 ≥70%
2. token blacklist 及其余 root service 文件按覆盖率 profile 倒序补齐,无单文件留在 <50%
3. phase 收尾以 `go tool cover -func` 实测数字回填 `.planning/coverage-baseline.md`(ratchet 数据链连续)
4. gate 全程绿(本 phase 不动 `.coverage-threshold`,统一 Phase 81 收口)

**Plans**: TBD(建议 6)
- 79-01 legacy cache services 群 A(dept/role/dict)
- 79-02 legacy cache services 群 B(menu/user/post)
- 79-03 token blacklist + DataCacheService/CacheConfigService 深路径
- 79-04 root services 扫尾 A(profile 倒序)
- 79-05 root services 扫尾 B(profile 倒序)
- 79-06 phase 复测 + baseline 回填

**Notes**:

- 5202 stmts 是全 milestone 最大单包工作面;plan 边界按 profile 实测倒序,planner 可再细调拆分
- 新测试遇到 v1.26 锁定的非正典 quirk(如 73-04 系列的 monitor cache_service 锁定项)沿用"锁定+注释"惯例,不擅自扩修(15 项正典之外范围纪律)

---

### Phase 80: 长尾清欠·scheduler + 碎包

**Goal:** internal/scheduler 引擎与全部碎包(api/v1 / models / internal/api / pkg/errors / pkg/cache / 小尾巴)逐一 ≥70%,TAIL 长尾清零、70% 数学缺口关门。

**Depends on**: Phase 76(miniredis 供 pkg/cache Redis 路径;与 Phase 77/78/79 无硬依赖,可并行穿插)

**Requirements**: TAIL-02, TAIL-03

**Success Criteria** (what must be TRUE):

1. `internal/scheduler` ≥70%(3.3% → ≥70%,补 ~736 stmts;注册表/执行器/引擎并发与取消分支)
2. 碎包逐包 ≥70%:api/v1(366)+ models(310)+ internal/api(291,装配层纯函数段)+ pkg/errors(183)+ pkg/cache(161,Redis 路径经 miniredis)
3. <50 stmts 小尾巴(permission/websocket/base/lldp/gormutil/middleware/logger/query)合计 ≥70%,确不可测的按既有豁免规则文档化
4. gate 全程绿

**Plans**: TBD(建议 5)
- 80-01 scheduler 注册表 + job 执行器
- 80-02 scheduler 引擎分支(并发/取消/恢复)
- 80-03 碎包 A:api/v1 装配纯函数段 + models
- 80-04 碎包 B:internal/api + pkg/errors
- 80-05 碎包 C:pkg/cache(miniredis)+ 小尾巴清尾

**Notes**:

- scheduler goroutine 需 t.Cleanup 停机,防 -race 泄漏告警;internal/scheduler(引擎)≠ api/v1/scheduler(job CRUD,已达标)
- websocket 真 WS 握手 httptest 可测;lldp 优先报文解析纯函数段

---

### Phase 81: 全仓收口·ratchet 闭环与 milestone audit

**Goal:** 全量重测把 `.coverage-threshold` 从 55.5 ratchet 到 ≥70 实测值、删除 P2_RATCHET 豁免行回落全量 70% floor、milestone audit 定案,gate 防线全程不倒退。

**Depends on**: Phase 75, Phase 76, Phase 77, Phase 78, Phase 79, Phase 80(全部完成后收口)

**Requirements**: GATE-01, GATE-02, GATE-03

**Success Criteria** (what must be TRUE):

1. 全量 `go test -coverprofile` 重测:加权平均 ≥70%(43652 stmts 口径,SC-a 数学 30556/43652 达成),`.coverage-threshold` 55.5 → 新实测值(UP-only 语义)
2. check-coverage.sh 删除 core/device/agent-server 三条 P2_RATCHET 豁免行,P2 floor 回落全量 70%,删除后 gate 本地 + CI 实跑绿(UP-only 闭环:豁免只减不增)
3. 4 层 gate(weighted-avg / P1 floor exit 4 / P2 floor exit 5 / PR diff coverage ≥80%)在收口 commit 上 CI 全绿,且 milestone 全程无 gate 倒退记录
4. milestone audit 报告落盘:19/19 需求核验、15 项 QUIRK 关闭清单、SC-a..e 证据链、v1.26 SC-a 缺口(6287 stmts)收口数学

**Plans**: TBD(建议 3)
- 81-01 全量重测 + threshold ratchet 55.5 → 实测 ≥70 + coverage-baseline.md 回填
- 81-02 P2_RATCHET 豁免行删除 + floor 回落 70 全量 + CI 验证
- 81-03 milestone audit(镜像 74-11 收口流程 + v1.26 74-MILESTONE-AUDIT 模式)

**Notes**:

- 镜像 v1.26 的 74-11 收口:atomic ratchet commit 模式(D-07 六文件先例:threshold + baseline + check-coverage.sh + STATE + ROADMAP + SUMMARY)
- 若个别包最终 <70%(如平台绑定的 pty 路径),豁免必须在 audit 显式文档化并保留对应 P2_RATCHET 行——不允许静默豁免

---

## Progress

| Phase | Status | Plans | Requirements | Started | Completed |
|-------|--------|-------|--------------|---------|-----------|
| Phase 75 QUIRK 行为修正 | Not started | 0/5 (建议) | QUIRK-01..03 | - | - |
| Phase 76 测试基建落地 | Not started | 0/5 (建议) | INFRA-01..05 | - | - |
| Phase 77 阻塞包·零基建 | Not started | 0/5 (建议) | BLOCK-01, BLOCK-02 | - | - |
| Phase 78 阻塞包·基建解锁 | Not started | 0/7 (建议) | BLOCK-03..05 | - | - |
| Phase 79 长尾·services root | Not started | 0/6 (建议) | TAIL-01 | - | - |
| Phase 80 长尾·scheduler+碎包 | Not started | 0/5 (建议) | TAIL-02, TAIL-03 | - | - |
| Phase 81 收口·ratchet+gate | Not started | 0/3 (建议) | GATE-01..03 | - | - |

**Total:** 7 phases / 19 requirements (0/19 done — 100% mapped,无孤儿)

---

## Future Milestones (TBD)

候选方向(基于 STATE.md §Deferred Items + Future Requirements):

- **v1.28+**: Phase 63 启动块遗留 advisory 收尾(network/ports 3 timeout + total bundle +90kB + knip ignore 收紧)
- **v1.28+**: PROTO-01..04 逐屏原型对齐 + VIS-01..03 视觉深化(v1.22 Future Requirements)
- **v1.28+**: VDI 22C/22D(账号管理 + Web Console,依赖 v1.11 生产稳定性数据)
- **v1.28+**: 分支覆盖率 / mutation testing / PR 评论机器人(FUT-02..04)
- **v1.28+**: 真机 site-visit UAT 闭环(v1.18-v1.20 deferred items)
- **v1.28+**: CI-only smoke 层(真 Redis 连接池行为 / 真 OpenLDAP 报文;`//go:build smoke` + 本地自动 skip,不进默认测试面)

---

## Operational Architecture

```
Backend (Go)
├── internal/api/v1/{system,operations,workorder,scheduler,duty,network,
│                       knowledge,monitor,rpa,vdi,asset}/  ← v1.26 P0-P2 已补齐
├── internal/services/{system,operations,workorder,scheduler,duty,network,
│                       knowledge,monitor,rpa,vdi,asset,portwrite,
│                       component_collector,topology,lldp,addomain}/
│     └── (root) 遗留 cache services 群 ← v1.27 Phase 79 主战场
├── internal/middleware/  (86.2% 已高,维持)
├── internal/utils/operlog/  (82.2% 范本)
├── pkg/{cache,normalize,transform,response,query,logger,...}/
└── internal/{core,config,device,agent,scheduler,collector}/  ← v1.27 结构阻塞包所在

CI Gate (.github/workflows/ci.yml)
├── backend job:
│   ├── go test ./internal/... ./pkg/... ./cmd/... -coverprofile=coverage.out -covermode=atomic
│   ├── go tool cover -func → awk 加权平均 vs .coverage-threshold gate
│   ├── P1 floor (8 包 ≥70%, exit 4) + P2 floor (70% x 7 + UP-ONLY ratcheted x 3, exit 5)
│   └── PR diff coverage ≥80% 门槛 (74-10 落地)
└── frontend job:  Phase 63 vitest coverage thresholds 25/15/22/25(不动)

Test Infrastructure
├── glebarez sqlite in-memory (已有,不引新 mock)
├── testing + table-driven tests (已有模式)
├── portwrite 85.3% 范本 (Phase 50-55 落地)
├── operlog 82.2% 范本 (regression_test.go 锁公共 API)
├── middleware 86.2% 范本 (apikey 全链路测试)
└── v1.27 test doubles (Phase 76 落地,零 Docker,Windows/CI 同构)
    ├── miniredis/v2 v2.38+ (pkg/cache + core Init 链)     ← INFRA-01
    ├── httpmock v1.4.x (const-URL 出站 HTTP)              ← INFRA-01
    ├── scrapligo FileTransport + x/crypto/ssh fake (device) ← INFRA-02
    ├── LDAPClientIface stub (addomain)                    ← INFRA-03
    ├── TestHelperProcess re-exec (agent 子进程/reaper)     ← INFRA-04
    └── AST 守护测试 (ForTesting 隔离契约)                  ← INFRA-05
```

---

*Last updated: 2026-08-23 — v1.27 backend-test-coverage-excellence-II milestone roadmap drafted: 7 phases (75-81) / 19 requirements / 100% mapped。QUIRK-first 排序(IncrementBy 最先解锁 core captcha),零 Docker 双环境同构,gap-math 6287 stmts(BLOCK ~2402 + TAIL ~3885)。v1.26 已 SHIPPED 2026-08-22(详情归档 `.planning/milestones/v1.26-phases/`)。*
