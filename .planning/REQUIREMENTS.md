# Requirements: XingRan-Next（RuoYi-Next 运维管理系统）

**Defined:** 2026-11-08
**Core Value:** 以 `.planning/OPEN-ITEMS.md`（2026-11-08「真实仍存在的问题唯一权威清单」）的 **21 条真实开放项**为唯一输入，按「代码可修 / 待人工验证 / 已裁定接受」三类分别闭环，使 v1.53 收档时每条开放项都有可核对的结论与证据，消除「历史 TODO/FUTURE/待办字样驱动修复」的文档债。

## v1.53 Requirements

**范围裁定（用户 2026-11-08）:** 21 条全覆盖（A 节 8 + B 节 13）；C 节 13 类「明确不属于开放项」防误读清单不纳入；Phase 从 **134** 续编（v1.52 结束于 133）；**跳过领域研究**（21 条均带本仓 `文件:行号` 实证、无新领域能力需调研，沿用 v1.25「研究范围仅定向」先例）；**纳入 seed** `seeds/SEED-ad-push-enable-direction.md` 作为 FIX-01 输入上下文。

**效力约束（继承 OPEN-ITEMS.md）:**
- 凡未列入 OPEN-ITEMS.md 的历史 `TODO` / `FUTURE` / `待办` / `deferred` / `Known Gaps` 字样，一律视为文档债或历史归档，不驱动修复、不据此判断当前待办。
- 状态权威 = `.planning/STATE.md` frontmatter / `.planning/ROADMAP.md` / `.planning/MILESTONE-v1.52-SUMMARY.md`；开放项权威 = `.planning/OPEN-ITEMS.md`；二者冲突时**以证据为准**。
- 路径换算：OPEN-ITEMS.md 内 `phases/...` 证据路径按 `.planning/ARCHIVE-INDEX.md` 换算到 `.planning/milestones/v1.52-phases/`（条目本身与状态不变）。
- 行号口径：OPEN-ITEMS.md A 节对 `PROJECT.md` / `MILESTONES.md` 的行号引用为 v1.52 收档基线行号，定位以「段标题 + 条目序号」为准。

### FIX — 代码可修开放项（14 条）

- [ ] **FIX-01**: sys 侧恢复（软删恢复 / 状态 1→0 启用）用户时，AD 端账号自动重新启用——`enable` 动作入队 → 排水器执行 `EnableUser`（UAC 544）→ 该账号可 AD 登录；身份不一致的恢复不入队 enable 且记安全 WARN。
  - **来源**: OPEN-ITEMS.md #A1（功能缺口）｜**seed**: `seeds/SEED-ad-push-enable-direction.md`（已纳入，含常量/入队/排水/SAFE-01 交互/F-01 压制模式建议方案）
  - **证据**: `internal/models/ad_push_queue.go:27-29`、`internal/services/addomain/ad_push_queue_service.go:228-229`、`internal/services/system/user_service.go:778-779`；全库无 `ADPushActionEnable` 入队/排水路径
  - **验收**: 新增 `ADPushActionEnable` + 恢复路径同事务入队 + `drainRow` enable 分支 + 与 SAFE-01/SAFE-02 守卫集成 + disable/enable 最新意图胜出（F-01 压制）；sqlite 自动化用例 + 一次与 2026-10-21 d_chenchao 场景对称的实机 UAT
- [ ] **FIX-02**: Swagger 产物链路修复——`cmd/main.go` blank import `_ ".../internal/docs"` 使 `swag.Register` 生效；两个生成脚本产物目录由 `docs` 改为 `internal/docs`；修复 `generate_swagger.bat` 的 `cd /d "%~dp0.."` 层级 bug；清理 `docs/` 下 5 份 Swagger 死文件与 `scripts/README.md:256-257` 自相矛盾表述。
  - **来源**: OPEN-ITEMS.md #B1 / #B2 / #B3（真实缺陷）+ #B13③（docs 死文件，仅登记项中的可清理部分）
  - **证据**: `cmd/main.go:3-29,244`、`internal/docs/docs.go:21513-21527`、`scripts/build/generate_swagger.bat:19,47`、`scripts/generate_swagger.sh:46`、`scripts/README.md:256-257`、`docs/{docs.go,swagger.json,swagger.yaml,swagger_templates.go,models.go}`
  - **验收**: `/swagger/index.html` 能取到规范；两脚本生成的产物落在 `internal/docs/`；bat 从 `scripts/build/` 执行可正常找到 `cmd/main.go`；`docs/` 下 5 份死文件清零；`go build ./...` 全绿
- [ ] **FIX-03**: 用户导出端点接线——新增 `POST /system/users/export` 路由与前端导出按钮，复用 service 层已有 `ExportData("user")` 能力（导入已落地，导出缺失）。
  - **来源**: OPEN-ITEMS.md #B4（功能缺口）
  - **证据**: `internal/api/router.go:194,208-209`（注释明示「导出接口不存在」）、`internal/services/operations/excel_service.go:1609-1632`
  - **验收**: 端点返回可用 Excel 文件、导出字段与导入模板对齐；operlog 记录 `OperTypeExport`；前端按钮可用；不破坏已落地导入路径
- [ ] **FIX-04**: 端口写补齐「物理参数：速率/双工/MTU 调整」——`internal/services/portwrite` 实现对应能力（现状 grep `speed`/`duplex`/`MTU` 命中 0）。
  - **来源**: OPEN-ITEMS.md #B5（功能缺口；v1.51 FUTURE 项中唯一真实未落地项）
  - **证据**: `docs/管理员使用手册.md:812`；`internal/services/portwrite/` 全包 grep → 0
  - **验收**: 3 厂商命令模板 + 参数校验 + 审计（沿用 v1.19/v1.20.1 端口写基建）；单测/e2e 零回归；手册 §22「后续方向」表同步更新
- [ ] **FIX-05**: scheduler 层周期工单 Rotation 分支补齐——cron 生成的 rotation 模板工单按轮询分配处理人，与 service 层 `periodic.go` 已实现的 Rotation 语义一致（现状直接 `return nil, nil` 不分配）。
  - **来源**: OPEN-ITEMS.md #B8（真实缺陷 / 功能缺口）
  - **证据**: `internal/scheduler/workorder_tasks.go:87,231-234`、`internal/scheduler/workorder_tasks_80_02_test.go:306-314`、`internal/services/workorder/periodic.go:401-421`
  - **验收**: 双路径一致性测试（同输入产生同处理人）；`go test ./internal/scheduler/` 全套零回归
- [ ] **FIX-06**: 工单评价端点接线——`SetupWorkOrderRatingsRouter` 空函数体接入已实现的 `RatingService`，补齐前端评价页。
  - **来源**: OPEN-ITEMS.md #B9（功能缺口）
  - **证据**: `internal/api/v1/workorder/workorder_router.go:62-67`、`internal/api/router.go:580`、`internal/services/workorder/rating.go:12-134`
  - **验收**: 评价 CRUD HTTP 端点可用、权限码挂载、operlog 覆盖；前端评价页可见可提交
- [ ] **FIX-07**: `GetDiscoveryResults` stub 实现——发现结果查询返回真实结果而非恒空列表。
  - **来源**: OPEN-ITEMS.md #B10（功能缺口 / 真实缺陷）
  - **证据**: `internal/services/device_discovery_service.go:655-663`、`internal/api/v1/network/discovery_handler.go:186`、`internal/api/v1/network/discovery_handler_test.go:153,159`（测试如实反映 stub 行为，非过时测试）
  - **验收**: 查询返回已持久化的发现结果；handler 与测试同步更新（删除「恒空」断言）；发现流程零回归
- [ ] **FIX-08**: rpa-worker browser stub 与双副本漂移治理——前端副本 `createPage` 改用真实 Chrome（`NewChromePage`）；两份 `page_manager.go` 的 `WaitForURL`/`WaitForText` 去除首次 tick 即 `return nil` 的假校验；收敛/登记两份 `go.mod` 子模块漂移。
  - **来源**: OPEN-ITEMS.md #B11（真实缺陷 / 低优先级技术债）
  - **证据**: `xingran-react-frontend/rpa-worker/internal/browser/pool.go:148-156`、两份 `page_manager.go`（根 `:118,137` / 前端 `:84,103`）、根 `rpa-worker/internal/browser/pool.go:160-168`
  - **验收**: 两副本行为一致、`go build` 各自通过；超时/失败路径有断言；漂移结论（合并或显式登记差异）落 `.planning/notes/`
- [ ] **FIX-09**: 5 个登录加密 benchmark 空桩补实测——`encrypted_login` / `plaintext_login` / `nonce_check_and_store` / `sm2_encrypt_sm4_key` / `sm4_cbc_encrypt_request` 去除 TODO 空循环，给出 `P99<350ms` 目标的真实测量。
  - **来源**: OPEN-ITEMS.md #B12（低优先级技术债 / 测试覆盖债）
  - **证据**: `tests/benchmark/login_encryption_bench_test.go:26,34,44,55,65`
  - **验收**: 5 个 benchmark 可运行且产出数值；结论（达标/未达标 + 证据）落盘；不引入新依赖
- [ ] **FIX-10**: 全量同步节流产品化——3h 全量 cron（`sync_interval=10800`）+ 失败后每 5min tick 重试的重试风暴，落地节流/退避策略。
  - **来源**: OPEN-ITEMS.md #A6（低优先级技术债）
  - **证据**: `internal/scheduler/ad_sync_tasks.go:303-316`、`internal/models/ad_domain.go:37`
  - **验收**: 配置化退避 + 单测覆盖「连续失败不成风暴」；`go test ./internal/scheduler/` 零回归
- [ ] **FIX-11**: 同步漂移告警产品化——把一次性 UAT 采样脚本 `scripts/uat61-t1-poller-v5.ps1` 升级为常态化同步漂移告警通道。
  - **来源**: OPEN-ITEMS.md #A7（功能缺口 / 低优先级技术债）
  - **证据**: `RETROSPECTIVE.md:374`（3025 次采样实锤为一次性重放窗口验证，非告警通道）
  - **验收**: 告警通道可配置、可观测、有触发记录；与 A8①「未来应加告警通道而非改回整批 403」裁定方向一致
- [ ] **FIX-12**: 对账 MV 启动刷新 30s 超时配置化——去除 `context.WithTimeout(..., 30*time.Second)` 硬编码（O(N²) 根因已由 migration_180 根治，此项为非阻断低优先级）。
  - **来源**: OPEN-ITEMS.md #B7①（低优先级技术债）
  - **证据**: `internal/core/core.go:241`、`notes/260630-mv-refresh-30s-timeout.md:4,85`
  - **验收**: 超时值可配置 + 默认值不变 + 启动路径零回归
- [ ] **FIX-13**: OQ-7「5 个权限码读路由恒 403」独立后续项闭环——定位并修复 5 个读路由的权限判定，使持有对应权限码的角色可正常读取。
  - **来源**: OPEN-ITEMS.md #A3 附带的 OQ-7（功能缺口，v1.25 Phase 87 明确「本 phase 不闭合、独立登记」）
  - **证据**: `phases/129-real-device-uat/129-UAT-BACKFILL.md:34-45,83,91-92`（OQ-7 段）
  - **验收**: 5 个读路由 403→200 的对照证据；不破坏 v1.25 严格权限中间件（RBAC 零回归）
- [ ] **FIX-14**: 3 处排序收尾——「我的值班统计 / 节假日排序 / 菜单·部门树形排序」推进产品确认并落地（与已完成的 4 页面排序菜单注册是两回事）。
  - **来源**: OPEN-ITEMS.md #B6（功能缺口 / 待产品决策）
  - **证据**: `docs/管理员使用手册.md:829`
  - **验收**: 产品结论落 `.planning/notes/`；按结论实现或显式登记「不推进」；服务端排序契约零回归

### VAL — 待人工验证开放项（7 条）

> **共同约定（不伪造闭环）**：本类 REQ 的交付物是**可执行清单 + 统一跟踪表 + 回写口径**，实际执行受外部条件约束（生产 PG 只读访问 / 写操作授权 / 现场运维同事真机 / DBA 回传）。AI 不擅自写生产数据、不擅自切换生产认证模式（沿用 v1.25 Phase 90 先例）。执行完成后的 UAT 勾选与 OPEN-ITEMS.md 回写由 GOV-01 统一收口。

- [ ] **VAL-01**: Phase 84 的 2 项 PG 实机验证——`data_scope=3` 复测 total、`sys_role_menu`/`sys_user_role` 生产库实际列集，按已登记判定方法组织为可执行清单。
  - **来源**: OPEN-ITEMS.md #A2（待人工验证，需生产 PG 只读）
  - **证据**: `phases/129-real-device-uat/129-UAT-BACKFILL.md:26-27,91`（84-1 / 84-2 ⏳）、`notes/2026-10-29-pg-data-scope-verification.md`（A~F 六步 + 三个已知坑）
  - **验收**: 清单含步骤/预期/证据位；状态标「待执行（依赖 PG 只读授权）」，不标已完成
- [ ] **VAL-02**: Phase 87 的 7 项生产侧验证——受影响生产角色清单 / 既有行零改动核对 / 迁移 PG 实执行 / 唯一键复核 / HTTP 403→200 / 前端 UI 实机交互 / 生产部署顺序；其中 87-6 额外需写操作授权。
  - **来源**: OPEN-ITEMS.md #A3（待人工验证，需生产窗口 + PG 只读 + 87-6 写授权）
  - **证据**: `phases/129-real-device-uat/129-UAT-BACKFILL.md:34-45,83,91-92`（87-1~87-7 表、选项 A 中 87-6 仍未做）
  - **验收**: 7 项逐项列出前置授权与证据位；87-6 单列写操作授权门槛
- [ ] **VAL-03**: v1.21 57-HUMAN-UAT 三项——重建磁盘缺失的原始 `57-HUMAN-UAT.md`，2 项（PG 合并路径 / AD UAC 533 闭环）组织待闭合，1 项（kill -9 重启耐久）按 `debug/ad-push-queue-drain-interval-not-hot-reloaded.md:172-174` 独立证据核销回写。
  - **来源**: OPEN-ITEMS.md #A4（待人工验证，1 项已可核销 / 2 项待闭合）
  - **证据**: `STATE.md:248`、`PROJECT.md:180`、`debug/ad-push-queue-drain-interval-not-hot-reloaded.md:172-174`
  - **验收**: 重建清单落盘；核销项标注清偿日期 + 证据；2 项标待闭合
- [ ] **VAL-04**: 现场 / 真机 UAT 债——重建磁盘缺失的 `48-HUMAN-UAT.md` / `54-HUMAN-UAT.md` / `56-HUMAN-UAT.md` 三份清单，并把 v1.18（3 项 SNMP/CLI）、v1.19（7 项 SSH，WR-02 已 RESOLVED）、v1.20（12 项 VLAN/绑定 site-visit）、v1.51 Phase 129（`129-RESULTS.md §五` 5 条复测 + `129-HUMAN-UAT.md:63-66` H3C deferred 2 条）汇入统一跟踪表。
  - **来源**: OPEN-ITEMS.md #A5（待人工验证，现场运维同事执行后回写 UAT）
  - **证据**: `MILESTONES.md:138,141,172,189,198`、`phases/129-real-device-uat/129-RESULTS.md:310-316`、`phases/129-real-device-uat/129-HUMAN-UAT.md:63-66`、`notes/261102-planning-sync-strategy.md`（归档策略）
  - **验收**: 3 份清单重建 + 统一跟踪表（含设备型号/动作/owner/证据位）；不标已完成
- [ ] **VAL-05**: v1.52 `133-UAT.md` 人工 UAT 清单（A1-A11 / B1-B4 / C / D / E / F / G，约 30 项）全部 `- [ ]` 未勾选，纳入统一跟踪表，需真实 PG + 跨角色浏览器执行后方能勾选。
  - **来源**: OPEN-ITEMS.md #A8 附带（v1.52 已交付 UAT 清单，性质为待人工验证、不驱动代码修复）
  - **证据**: `phases/133-scope-final/133-UAT.md:13-127`、`MILESTONE-v1.52-SUMMARY.md:78-84`
  - **验收**: 约 30 项逐项登记前置条件与证据位；状态保持未勾选
- [ ] **VAL-06**: DBA 生产验证回传——对账 MV 超时根治（migration_180）的 DBA 生产验证结论回传并归档。
  - **来源**: OPEN-ITEMS.md #B7②（待人工验证 / DBA 回传）
  - **证据**: `notes/260630-mv-refresh-dba-request.md:159-190`
  - **验收**: 回传记录位 + 结论落盘；未回传前不标已完成
- [ ] **VAL-07**: 人工验证执行框架——产出「5 份磁盘缺失 UAT 清单重建 + 统一人工验证跟踪表（覆盖 VAL-01~06 全部条目）+ OPEN-ITEMS.md 回写口径」三件套，作为 Phase 140 的可交付机制。
  - **来源**: OPEN-ITEMS.md 维护约定 §2（清偿须标注清偿日期 + commit/证据并移入 C 节）+ A4/A5 的「原始清单磁盘不存在、须重建」实测
  - **验收**: 跟踪表可按「条目 → 前置授权 → 执行人 → 证据 → 回写状态」筛选；5 份重建清单落盘；口径文档明确「未执行不得勾选」

### ACK — 已裁定接受 / 仅登记残余（2 条）

> **共同约定（只落结论，不写修复代码）**：本类 REQ 对应 OPEN-ITEMS.md 明示 LOCKED / 「仅登记，不驱动修复」的条目。交付物是「接受/已知约束」结论落盘 + 证据链接，**不得**顺手改代码。

- [ ] **ACK-01**: v1.52 收档已裁定接受 / 独立登记的 4 项残余结论落盘——① P1-5 聚合计数侧信道（裁定「接受，未来应加告警通道而非改回整批 403」）；② 5 项显式豁免面（walls / doors / floor-plan-texts / rooms-photos / location-alias，ds05 裁定接受风险）；③ `excel_export_service.go:148` `RightCast:"text"` 动态拼接与 `workstation_device_service.go` PG 专属 CTE 的 SQL 债；④ `scripts/operlog_e2e_verify.sh:57-66` 凭据门禁先于 `SKIP_LIVE:162` 的既有语义缺陷。
  - **来源**: OPEN-ITEMS.md #A8（已裁定接受 / LOCKED，不驱动修复）
  - **证据**: `MILESTONE-v1.52-SUMMARY.md:78-84`
  - **验收**: 4 项结论逐条落 `.planning/notes/`，每条带「裁定 + 残余风险 + 不改代码理由」；与 v1.52 终态基线无冲突
- [ ] **ACK-02**: B13 仅登记 / 已知约束结论落盘——① `invalidateCache` 在无 Redis 场景为 no-op 的 R3+ TODO（例外规则写后缓存未失效）；② `pbkdf2SM3` 未实现 `keyLen>32` 多块循环（全部调用点 `keyLen=32`，安全不受影响）。
  - **来源**: OPEN-ITEMS.md #B13①②（低优先级技术债 / 已知约束，仅登记不驱动修复）
  - **证据**: `internal/services/asset/reconciliation_exception.go:579`、`internal/core/security/password.go:67`
  - **验收**: 2 项结论落盘；明确「不补实现」的理由与触发复核条件（如未来出现 `keyLen>32` 调用点）

### GOV — 清单治理与清偿回写（1 条）

- [ ] **GOV-01**: 按 OPEN-ITEMS.md 维护约定完成清偿回写与权威分层一致性复核——① 每条已清偿开放项在 OPEN-ITEMS.md 标注**清偿日期 + commit/证据**并从 A/B 节**移入 C 节**（不直接删除）；② 未清偿项保留 A/B 节并更新状态；③ 复核 `.planning/STATE.md` frontmatter / `.planning/ROADMAP.md` / `.planning/MILESTONE-v1.52-SUMMARY.md`（状态权威）与 OPEN-ITEMS.md（开放项权威）无冲突，冲突时以证据为准并修正失准方；④ 确认其它 `.planning/` 文件只引用 OPEN-ITEMS.md 作为真实问题唯一入口，不另立并行清单。
  - **来源**: OPEN-ITEMS.md 维护约定 §1~§4 + 头部「效力/权威状态来源/计数」段
  - **验收**: 21/21 条开放项在 OPEN-ITEMS.md 均有终态结论（已清偿并移入 C 节 / 保留挂账并注明前置 / LOCKED 结论登记）；A/B 节计数与 C 节新增行一致；权威分层复核记录落盘

## Future Requirements（deferred，不进 v1.53 roadmap）

- **FUTURE-01**: A8① 聚合计数侧信道的「告警通道」产品化（裁定方向：加告警通道而非改回整批 403）——与 FIX-11 同步漂移告警通道合并评估，本里程碑仅登记结论（ACK-01）。
- **FUTURE-02**: C 节 13 类防误读清单中若未来出现反证（证明某「已清偿」条目实为真实问题），按 OPEN-ITEMS.md §1 重新登记后再立项，不在此里程碑推进。

## Out of Scope

| Feature | Reason |
|---------|--------|
| OPEN-ITEMS.md C 节 13 类「明确不属于开放项」条目 | 已标注为历史归档或已交付（RPA Token / operlog nickname / exclude_paths / 端口写审计索引 / sqlite join 表列 / v1.23 seeds / v1.17 对账 / 27 条前端 P0 / 历史 code-review / 用户导入 / 4 页面排序菜单 / v1.51 FUTURE 除 B5 外 / v1.24 Known Gaps 已立项 / 同组管理员 OU 写权限已裁决关闭 / v1.23 三项延后已落地），按清单效力约定「一律勿再推进、勿据此新建修复任务」 |
| A8 四项 LOCKED 残余的代码修改 | v1.52 收档明确裁定「接受 / 独立登记」，性质 LOCKED、仅登记不改代码（ACK-01 只落结论） |
| 生产数据写入 / 生产认证模式切换 | VAL-* 只做只读验证与跟踪表；生产写入需显式授权，沿用 v1.25 Phase 90「不擅自执行」先例 |
| 新功能开发 / 架构重构 | 本里程碑定位为「开放项清偿收口」，不引入清单外新特性 |
| `phases/` 归档路径换算之外的历史目录改名 | 保留 v1.51/v1.52 归档可追溯性，不改名（沿用 STATE.md「历史目录编号消歧」决策） |

## Traceability

| Requirement | Phase | Status | 开放项来源 |
|-------------|-------|--------|-----------|
| FIX-01 | Phase 134 | Pending | A1 |
| FIX-10 | Phase 134 | Pending | A6 |
| FIX-11 | Phase 134 | Pending | A7 |
| FIX-02 | Phase 135 | Pending | B1 / B2 / B3 / B13③ |
| FIX-03 | Phase 136 | Pending | B4 |
| FIX-06 | Phase 136 | Pending | B9 |
| FIX-13 | Phase 136 | Pending | A3-OQ7 |
| FIX-07 | Phase 137 | Pending | B10 |
| FIX-08 | Phase 137 | Pending | B11 |
| FIX-04 | Phase 138 | Pending | B5 |
| FIX-05 | Phase 138 | Pending | B8 |
| FIX-14 | Phase 138 | Pending | B6 |
| FIX-09 | Phase 139 | Pending | B12 |
| FIX-12 | Phase 139 | Pending | B7① |
| VAL-01 | Phase 140 | Pending | A2 |
| VAL-02 | Phase 140 | Pending | A3（7 项） |
| VAL-03 | Phase 140 | Pending | A4 |
| VAL-04 | Phase 140 | Pending | A5 |
| VAL-05 | Phase 140 | Pending | A8（133-UAT） |
| VAL-06 | Phase 140 | Pending | B7② |
| VAL-07 | Phase 140 | Pending | 维护约定 §2 + A4/A5 清单重建 |
| ACK-01 | Phase 141 | Pending | A8（4 项 LOCKED） |
| ACK-02 | Phase 141 | Pending | B13①② |
| GOV-01 | Phase 141 | Pending | 维护约定 §1~§4 |

**Coverage:**
- v1.53 requirements: **24 total**（FIX 14 / VAL 7 / ACK 2 / GOV 1）
- Mapped to phases: **24**（Phases 134-141，8 phases）
- Unmapped: **0** ✓
- 开放项覆盖: **21/21** ✓（A1~A8 + B1~B13，逐条映射见上表「开放项来源」列；A3 / A8 / B7 / B13 为一条开放项拆到两个 REQ）

---
*Requirements defined: 2026-11-08*
*Last updated: 2026-11-08 after v1.53 milestone INITIATED*
