---
last_updated: 2026-10-28
update_trigger: v1.23 SHIPPED & ARCHIVED — ROADMAP 重构为紧凑分组
previous_update: 2026-10-27 v1.23 roadmap CREATED
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

## Phases

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
