---
last_updated: 2026-10-28
update_trigger: v1.24 milestone SHIPPED & ARCHIVED (视觉回归修复与遗留债务清偿, Phases 76-83, 18/18 REQ satisfied)
previous_update: 2026-10-28 v1.24 roadmap CREATED
---

# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped 2026-04-16)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- ✅ **v1.3 技术债清理** — Phases 8-10 (shipped 2026-04-27)
- ✅ **v1.4 MAC地址采集优化** — Phase 11 (shipped 2026-05-09)
- ✅ **v1.5 MAC地址历史数据管理** — Phases 12-15 (shipped 2026-06-15; Phase 13 gap closure retroactive 2026-06-26)
- ✅ **v1.6 API密钥管理系统** — Phase 16 (shipped 2026-05-19)
- ✅ **v1.7 前后端加密配置同步** — Phase 17 (shipped 2026-05-20)
- ✅ **v1.8 登录端点加密增强** — Phase 18 (shipped 2026-05-21)
- ✅ **v1.9 AD域控集成扩展** — Phases 19-20 (shipped 2026-05-24)
- ✅ **v1.10 网络设备权限修复** — Phase 21 (shipped 2026-05-24)
- ✅ **v1.11 AD组自动同步系统** — Phase 23 (shipped 2026-05-26)
- ✅ **v1.12 深信服桌面云集成 (22A+22B)** — Phases 22A/22B (shipped 2026-06-02)
- ✅ **v1.13 资产管理模块** — Phase 26 (shipped 2026-06-08)
- ✅ **v1.14 全局列自定义** — Phase 27 (shipped 2026-06-09)
- ✅ **v1.15 工位设备关联** — Phase 28 (shipped 2026-06-10) ｜ **v1.15 部门物理位置映射** — Phase 39 (shipped 2026-06-25)
- ✅ **v1.16 技术债清理** — Phases 40-41 (shipped 2026-06-26)
- ✅ **v1.17 资产对账** — Phases 42-47 (shipped 2026-07-03)
- ✅ **v1.18 网络设备硬件清单** — Phases 48-49 (shipped 2026-07-04)
- ✅ **v1.19 网络设备写命令** — Phases 50-55 (shipped 2026-07-08)
- ✅ **v1.20 网络设备 VLAN + 端口绑定** — Phase 56 (shipped 2026-07-10)
- ✅ **v1.21 AD 双向同步 bug 修复** — Phases 57-62 (shipped 2026-10-22)
- ✅ **v1.22 前端公用组件整治** — Phases 63-68.1 (shipped 2026-10-27)
- ✅ **v1.23 前端契约全量收尾与遗留清偿** — Phases 69-75 (shipped 2026-10-28)
- ✅ **v1.24 视觉回归修复与遗留债务清偿** — Phases 76-83 (shipped 2026-10-28, 18/18 REQ satisfied)

## Phases

<details>
<summary>✅ v1.24 视觉回归修复与遗留债务清偿 (Phases 76-83) — SHIPPED 2026-10-28</summary>

> 需求基线：`.planning/REQUIREMENTS.md`（18 REQ **全部 Satisfied**）｜研究：`.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS}.md`
> 执行波次（FEATURES 建议）：**W1 零依赖**（76/80/81/82）→ **W2 需 PG**（78/79）→ **W3 拆分**（83）；77（视觉验证）在 76 后收口

- [x] **Phase 76: 双框 P0 修复 + CSS 审计登记册** — 复位块 `.ant-input-affix-wrapper > input.ant-input`（6 项，无 box-shadow）+ status 还原；**双框元素数 3 → 0**；登记册 36 条（P0 已修 8 / P1 待修 7 / P2 登记 19 / 保留 1 / 样板 1）；契约检查脚本 ALL PASS（VIS-01/02/03/04）
- [x] **Phase 77: 双框视觉回归验证** — 21 形态 × 明暗矩阵（内层 4 值 + carrier 边框 + 聚焦环 + 错误态色）→ **双框 = 0**；15 页巡检无双框/无破相/无新增 console error；17 张截图证据入库；**并查出 VIS-03 的 P0 死规则**（antd 6 status class 名误用 → 站内绝大多数输入框错误态不变红），已修（VIS-05）
- [x] **Phase 78: #16 前后端双端闭环** — `RequireStrictPermissions`（只做 Step1 精确匹配，跳过 F 型继承与模块降级）+ `/system/users` 读组/写组**兄弟分离**（逐动作单一权限点）+ ActionButton 扩 `permission` 字段 + `useActionPermission` 三态门控 + 权限码单一真相源与双向校验脚本（PERM-01..04/06）
- [x] **Phase 79: #16 PG 实机验收** — 受限角色（0 个 F 权限）+ 测试用户落库 → **HTTP 实测 8/8 写接口 403**、读接口 200、跨模块 403；**决定性对照**（临时追加 xtgl → delete 200 且真删除 → 排除「接口坏了」）；DB 层前后对照（修复前 8/8 放行 → 修复后 8/8 拒绝）；**受影响生产角色清单为空**（PERM-05）
- [x] **Phase 80: 3f130 空因诊断前端接入** — `getPhysicalWithDiagnosis` 包装（`getPhysical` 对外仍返回数组）+ Table+Alert 双区渲染（`no_user_bound` → info 常态 / 其余 → warning / warnings 逐条 / diagnosis 计数折叠）+ 静默失败补 `message.warning` + 后端自相矛盾注释修正（DIAG-01）
- [x] **Phase 81: F5 刷新落地页修复** — catch-all 区分「菜单未就绪」(InitializingFallback) 与「真无路由」(跳 dashboard) + `saveLastPath` 加守卫防自毁反写 + `lastPathStorage.ts` 抽出（顺带清 3 处 react-refresh）；**新增 3 条回归测试并做红绿验证**（回退实现 2 条失败 / 修复后 3 条全绿）（NAV-01）
- [x] **Phase 82: lint 清零 + 豁免理由表修复** — 51（17 errors + 34 warnings）→ **eslint 0 problems**；预算双写归零（`package.json --max-warnings 0` + `assert-lint-budget` 0/0）+ react-refresh 33 三类分流 + 豁免理由表 159 → 38 且 `check-exemption-reasons.mjs` exit 0 并接入 `lint:ci`（HYG-01/02/03/05）
- [x] **Phase 83: devices 页拆分** — **714 → 306 行（-57%）**：D1 列定义工厂(130) + D2 搜索区构造函数(63) + D3 handlers+行操作+权限门控(413→278) + D4 页面 JSX 外壳(194) + D5 行操作再拆(192)；`reloadRef` 外部注入链路 4 段逐字保留；`columns` useMemo 故意不完整依赖未被"修正"；v1.19 port-write 62 测试零回归且 diff 为空（HYG-04）

**Wave 划分**：W1 = 76→80→81→82（零外部依赖，可并行）；W2 = 78→79（PG）；W3 = 83（依赖 78 的权限 gate 模式）。收口 = 77。

详情: [milestones/v1.24-ROADMAP.md](milestones/v1.24-ROADMAP.md) ｜ [requirements](milestones/v1.24-REQUIREMENTS.md)（18/18 REQ satisfied）

</details>

<details>
<summary>✅ v1.23 前端契约全量收尾与遗留清偿 (Phases 69-75) — SHIPPED 2026-10-28</summary>

- [x] Phase 69: 后端 debug 重验 spike + 信息收集下发 — 7 项三分类定级 + 外部信息清单（DBG-01）
- [x] Phase 70: TextArea 契约基建 (W0) — AppTextArea + SearchForm textarea + eslint 双 selector + 权威基线（TEXT-01/03/04 + HYG-05）
- [x] Phase 71: TextArea 存量迁移 + COMP-05 分域迁移 (W1-W6) — 121 文件、P65 豁免 127→0、裸组件归零（TEXT-02 + MIG-01/02/04）
- [x] Phase 72: 后端 debug 修复 wave — #3 MAC 写防线 → #2 Cisco 采集（DBG-02/03）
- [x] Phase 73: 后端 #5 双侧归一化 + #7 空因可诊断（DBG-06/07）
- [x] Phase 74: COMP-05 收口 + lint 清偿 — P66 校验模式 + errors 81→17 / warnings 1366→34 / any 958→0 + 门禁（MIG-03 + HYG-02/03/04）
- [x] Phase 75: 浏览器实机 UAT + 收档 — 68.1 §8 全表 18/19 + FAIL-1 修复复验 + §8 回写关债（UAT-01/02）

详情: [milestones/v1.23-ROADMAP.md](milestones/v1.23-ROADMAP.md) ｜ [requirements](milestones/v1.23-REQUIREMENTS.md)（UAT 详情随 phase 归档至 milestones/v1.23-phases/）

</details>

<details>
<summary>✅ v1.22 前端公用组件整治 (Phases 63-68.1) — SHIPPED 2026-10-27</summary>

- [x] Phase 63: P0 止血 — 数据可达与安全（STOP-01..07）
- [x] Phase 64: P1 修复 — 反馈与可访问性（FIX-01..05）
- [x] Phase 65: 基础契约组件 — AppInput/AppSelect/ConfirmDelete（COMP-01/02）
- [x] Phase 66: 高级契约组件 — PageTable/FormModal（COMP-03/04）
- [x] Phase 67: 存量迁移 — 按 ROI 落地（COMP-05）
- [x] Phase 68: P2 一致性债务 — 颜色/日期/类型收敛（DEBT-01..03）
- [x] Phase 68.1: 收尾清偿 — 危险操作收口/契约 minor/豁免收缩（DEBT-04）

详情: [milestones/v1.22-ROADMAP.md](milestones/v1.22-ROADMAP.md) ｜ [audit](milestones/v1.22-MILESTONE-AUDIT.md)

</details>

<details>
<summary>✅ v1.21 AD 双向同步 bug 修复 (Phases 57-62) — SHIPPED 2026-10-22</summary>

- [x] Phases 57-62 — outbox-lite 架构落地 + 11 项 bug 修复 + 对账视图 + 死代码清理（22 plans / 19/19 REQ verified）

详情: [milestones/v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md) ｜ [audit](milestones/v1.21-MILESTONE-AUDIT.md)

</details>

<details>
<summary>✅ Pre-v1.21 (Phases 1-56) — v1.0 ~ v1.20, SHIPPED 2026-04-16 ~ 2026-07-10</summary>

- ✓ Phases 1-11 — v1.0-v1.4 基础功能
- ✓ Phases 12-16 — v1.5 MAC 历史 / v1.6 API 密钥
- ✓ Phases 17-21 — v1.7-v1.10 加密与 AD 集成
- ✓ Phases 22A-28 — v1.11-v1.15 VDI/资产/列自定义/工位设备
- ✓ Phases 30-39 — 性能优化 / 操作日志全模块 / 部门选择收敛 / AD 账号池 / 部门物理位置
- ✓ Phases 40-49 — v1.16 技术债 / v1.17 资产对账 / v1.18 设备组件
- ✓ Phases 50-56 — v1.19 写命令 / v1.20 VLAN + 端口绑定

各里程碑详情见 [milestones/](milestones/) 目录对应归档。

</details>

---

*Last updated: 2026-10-28 — v1.23 shipped & archived（Phases 69-75，22/22 REQ satisfied）。下一里程碑待定义（候选债：#16 PG 复测定责、3f130 绑定 user_id、F5 跳转 /dashboard 等，见 milestones/v1.23-REQUIREMENTS.md Known Gaps）。*
