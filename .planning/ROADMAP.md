# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- 🚧 **v1.51 端口写生产级操作体验与平台质量收口** — Phases 121-129 (in progress, 21 REQ) — 见下方 Phases
- ✅ **v1.25 全站权限治理与样式收敛** — Phases 84-93 (shipped 2026-10-30) — [v1.25-ROADMAP.md](milestones/v1.25-ROADMAP.md)
- ✅ **v1.24 视觉回归修复与遗留债务清偿** — Phases 76-83 (shipped 2026-10-28) — [v1.24-ROADMAP.md](milestones/v1.24-ROADMAP.md)
- ✅ **v1.23 前端契约全量收尾与遗留清偿** — Phases 69-75 (shipped 2026-10-28) — [v1.23-ROADMAP.md](milestones/v1.23-ROADMAP.md)
- ✅ **v1.22 前端组件契约与设计系统对齐** — Phases 63-68.1 (shipped 2026-10-27) — [v1.22-ROADMAP.md](milestones/v1.22-ROADMAP.md)
- ✅ **v1.21 后端架构统一与接口解耦** — Phases 57-62 (shipped 2026-10-22) — [v1.21-ROADMAP.md](milestones/v1.21-ROADMAP.md)
- ✅ **v1.20 工位管理与空间可视化演进** — Phases 48-56 (shipped 2026-07-10) — [v1.20-ROADMAP.md](milestones/v1.20-ROADMAP.md)

---

## Phases

<details open>
<summary>🚧 v1.51 端口写生产级操作体验与平台质量收口 (Phases 121-129) — 代码工作完成 2026-11-01（真机 UAT 待设备）</summary>

> 需求基线：`.planning/REQUIREMENTS.md`（21 REQ）｜范围裁定：A 端口写深化 + B 平台技术债 + D 收尾杂项（C 组 RPA 编辑器 defer）
> Phase 编号：延续 GitHub v1.33 线从 **121** 续编
> **交付状态：** 21/21 REQ 交付 · PR #38 merged（e602e811）· tag v1.51 · 集成验证窗口 #1 全项 PASS

- [x] Phase 121: 端口写 dry-run 预览 + 命令预览 UI (PORTW-01/02) — completed 2026-10-31
- [x] Phase 122: 批量写实时进度 SSE/WS — BATCH-05 清偿 (PORTW-03) — completed 2026-11-01
- [x] Phase 123: 安全三件套 — 快照回滚 + 并发写互斥 + 不可达预检 (PORTW-04/05/06) — completed 2026-11-01
- [x] Phase 124: VLAN 扩展 — trunk native/allowed + hybrid 华为/H3C (PWX-01) — completed 2026-11-01
- [x] Phase 125: 端口安全 — port-security/sticky/静态 MAC 绑定 (PWX-02) — completed 2026-11-01
- [x] Phase 126: Cisco IOS-XE 第 4 厂商 + 跨固件差异矩阵 (PWX-03) — completed 2026-11-01
- [x] Phase 127: 审计 UI + 审计治理 + 3 厂商 e2e + handler e2e (PWX-04/05/06 + DEBT-02) — completed 2026-11-01
- [x] Phase 128: 平台质量收口 — addomain 测试 / RPA Token / operlog 白名单 / permission 时序 / 排序菜单注册 (DEBT-01/03/04/05 + TIDY-01/03) — completed 2026-11-01
- [ ] Phase 129: 目录改名 + 双线 planning 合流 + UAT 回补组织（需窗口与真机）(DEBT-06 + TIDY-02) — **阻塞：待设备窗口**（华为 S8700 / 锐捷 RS8607E / H3C 必需，Cisco 可选）

**执行波次记录：** W1 零依赖（121 → 122/128 并行）→ W2 需设备（123 → 124/125）→ W3（126 → 127）→ W4 收尾（129，需真机）

</details>

<details>
<summary>✅ v1.25 全站权限治理与样式收敛 (Phases 84-93) — SHIPPED 2026-10-30</summary>

- [x] Phase 84: 权限模型三缺口 (4/4 plans) — completed 2026-10-28
- [x] Phase 85: 后端动作级严格化 (4/4 plans) — completed 2026-10-29
- [x] Phase 86: 前端操作入口三态门控铺开 (6/6 plans) — completed 2026-10-29
- [x] Phase 87: 动作级权限授予闭环 (5/5 plans) — completed 2026-10-30
- [x] Phase 88: 验证底座与零风险清理 (4/4 plans) — completed 2026-10-29
- [x] Phase 89: 变量层与明暗色板重构 (4/4 plans) — completed 2026-10-29
- [x] Phase 90: Input 域交还 variant/token + 圆角统一 (4/4 plans) — completed 2026-10-30
- [x] Phase 91: Select / InputNumber / Form / Table / Modal 域交还 (3/3 plans) — completed 2026-10-30
- [x] Phase 92: 暗色 mega-cluster 合并 (5/5 plans) — completed 2026-10-30
- [x] Phase 93: room-devices 拆分 + 视觉回归验收 (2/2 plans) — completed 2026-10-30

</details>

<details>
<summary>✅ v1.24 视觉回归修复与遗留债务清偿 (Phases 76-83) — SHIPPED 2026-10-28</summary>

- [x] Phase 76: 双框 P0 修复 + CSS 审计登记册 — completed 2026-10-28
- [x] Phase 77: 双框视觉回归验证 — completed 2026-10-28
- [x] Phase 78: #16 前后端双端闭环 — completed 2026-10-28
- [x] Phase 79: #16 PG 实机验收 — completed 2026-10-28
- [x] Phase 80: 3f130 空因诊断前端接入 — completed 2026-10-28
- [x] Phase 81: F5 刷新落地页修复 — completed 2026-10-28
- [x] Phase 82: lint 清零 + 豁免理由表修复 — completed 2026-10-28
- [x] Phase 83: devices 页拆分 — completed 2026-10-28

</details>

---
*Roadmap updated: 2026-10-31 — v1.51 milestone initiated (Phases 121-129, 21 REQ)*
