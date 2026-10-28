---
last_updated: 2026-10-28
update_trigger: v1.24 roadmap CREATED (视觉回归修复与遗留债务清偿, Phases 76-83, 18 REQ coverage)
previous_update: 2026-10-28 v1.23 SHIPPED & ARCHIVED
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
- 🔄 **v1.24 视觉回归修复与遗留债务清偿** — Phases 76-83 (planning 2026-10-28)

## Phases

### 🔄 v1.24 视觉回归修复与遗留债务清偿 (Phases 76-83) — IN PROGRESS

> 需求基线：`.planning/REQUIREMENTS.md`（18 REQ）｜研究：`.planning/research/{STACK,FEATURES,ARCHITECTURE,PITFALLS}.md`
> 执行波次（FEATURES 建议）：**W1 零依赖**（76/80/81/82）→ **W2 需 PG**（78/79）→ **W3 拆分**（83）；77（视觉验证）在 76 后收口

- [ ] **Phase 76: 双框 P0 修复 + CSS 审计登记册** — wrapper 上下文复位规则（同步 4 个 CSS 块：:871-905 浅色主块 / :2252-2284 暗色含全局元素选择器 / :3717-3757 暗色二块）+ computed-style 断言脚本（修前取「双框>0」基线）+ CSS 一致性审计登记册（26 输入类 + 8 非输入类 + 12 条 Select 死规则）（VIS-01/02/03/04）
- [ ] **Phase 77: 双框视觉回归验证** — 26 格截图矩阵（11 类型 × 明暗 × 默认/聚焦）+ 14 回归页面 + 「allowClear 开/关视觉同形」机械判据 + 与 v1.23 分页持久化联动验收（VIS-05）
- [ ] **Phase 78: #16 前后端双端闭环** — ActionButton 契约扩 `permission` 字段（加法）+ 三态门控 helper（useActionPermission）+ 系统用户页接入 + 后端严格权限中间件（不做模块降级）+ `/system/users` 写操作拆独立子组（绑定 UserRemove 等既有常量）+ 权限码单一真相源与校验脚本（PERM-01..04/06）
- [ ] **Phase 79: #16 PG 实机验收** — UI 正常路径建受限角色（有页面无按钮权限）→ 按钮不可见 + curl 直调写接口 403 + admin/xtgl 零变化（PERM-05；外部依赖：PG 环境已具备）
- [ ] **Phase 80: 3f130 空因诊断前端接入** — `getPhysicalWithDiagnosis` 包装（调用方零改动）+ Table+Alert 双区渲染 + 后端矛盾注释修正 + 「有数据+空因」不变量测试 + 静默失败补提示（DIAG-01）
- [ ] **Phase 81: F5 刷新落地页修复** — `menusReady` 三态 catch-all + `saveLastPath` 跳过兜底跳转 + 与 FAIL-1 分页持久化联动验收（NAV-01）
- [ ] **Phase 82: lint 清零 + 豁免理由表修复** — 17 errors + 34 warnings → 0（预算双写同步 + AppSelect/local 规则冲突决策）+ react-refresh 33 三类分流（P1 拆 18 / P2 改代码 13 / P3 登记 2）+ `check-exemption-reasons.mjs` 123 项失配修复并接入 lint:ci（HYG-01/02/03/05）
- [ ] **Phase 83: devices 页拆分** — buildRowActions+权限 gate → 列定义工厂 → 采集端口 → 快速创建 → searchFields；reloadRef 回填链路保持；收敛 ~250 行；v1.19 port-write 62 测试零回归（HYG-04）

**Wave 划分**：W1 = 76→80→81→82（零外部依赖，可并行）；W2 = 78→79（PG）；W3 = 83（依赖 78 的权限 gate 模式）。收口 = 77。

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
