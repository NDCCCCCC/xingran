# Roadmap: RuoYi-Next 运维管理系统

## Milestones

- ✅ **v1.53 开放项全量清偿（Open Items Clearance）** — Phases 134-141 (shipped 2026-11-08, 24 REQ / 21 条开放项全覆盖) — [v1.53-ROADMAP.md](milestones/v1.53-ROADMAP.md) · [v1.53-REQUIREMENTS.md](milestones/v1.53-REQUIREMENTS.md)
- ✅ **v1.52 运维域数据权限统一收口（终态）** — Phases 130-133 (shipped 2026-11-07, 7 REQ；archived 2026-11-08) — [v1.52-ROADMAP.md](milestones/v1.52-ROADMAP.md) · [v1.52-REQUIREMENTS.md](milestones/v1.52-REQUIREMENTS.md)
- ✅ **v1.51 端口写生产级操作体验与平台质量收口** — Phases 121-129 (shipped 2026-11-02, 21 REQ) — [v1.51-REQUIREMENTS.md](milestones/v1.51-REQUIREMENTS.md)
- ✅ **v1.25 全站权限治理与样式收敛** — Phases 84-93 (shipped 2026-10-30)
- ✅ **v1.24 视觉回归修复与遗留债务清偿** — Phases 76-83 (shipped 2026-10-28)
- ✅ **v1.23 前端契约全量收尾与遗留清偿** — Phases 69-75 (shipped 2026-10-28)
- ✅ **v1.22 前端组件契约与设计系统对齐** — Phases 63-68.1 (shipped 2026-10-27)
- ✅ **v1.21 后端架构统一与接口解耦** — Phases 57-62 (shipped 2026-10-22)
- ✅ **v1.20 工位管理与空间可视化演进** — Phases 48-56 (shipped 2026-07-10)

---

## Phases

<details>
<summary>✅ v1.53 开放项全量清偿 (Phases 134-141) — SHIPPED 2026-11-08</summary>

- [x] Phase 134: AD 同步域 —— enable 方向 + 全量同步节流 + 漂移告警产品化 (FIX-01/10/11) — completed 2026-11-08
- [x] Phase 135: Swagger / 文档产物链路修复与死文件清理 (FIX-02) — completed 2026-11-08
- [x] Phase 136: 接线型端点缺口 —— 用户导出 + 工单评价 + OQ-7 五个读权限码 (FIX-03/06/13) — completed 2026-11-08
- [x] Phase 137: Stub 与双副本漂移治理 —— GetDiscoveryResults + rpa-worker (FIX-07/08) — completed 2026-11-08
- [x] Phase 138: 功能分支补齐 —— scheduler Rotation + 端口写物理参数 + 3 处排序 (FIX-04/05/14) — completed 2026-11-08
- [x] Phase 139: 低优先级技术债 —— 对账 MV 超时配置化 + 登录加密 benchmark 实测 (FIX-09/12) — completed 2026-11-08
- [x] Phase 140: 人工验证债组织 —— 5 份缺失 UAT 清单重建 + 79 条跟踪表 + 回写口径 (VAL-01~07) — completed 2026-11-08
- [x] Phase 141: 登记接受残余 + 临时清单归档 + 权威交回 GSD (ACK-01/02 + GOV-01) — completed 2026-11-08

**交付口径**：18 REQ 交付 + 6 项人工验证跟踪（VAL-01~06 交付物已落盘、实际验证待执行，不伪造闭环）；21/21 条开放项全覆盖。全量回归闸门：go build / go test 全仓 / tsc / lint:ci / vitest 592 文件 4128 用例全绿。完整 phase 详情见 [v1.53-ROADMAP.md](milestones/v1.53-ROADMAP.md) 与 `.planning/milestones/v1.53-phases/`。

**Known Gaps（登记，非阻断）**：人工验证跟踪表 `.planning/notes/261108-human-verification-tracker.md`（79 条全部待执行）；LOCKED 残余结论 `.planning/notes/261108-v153-locked-residuals.md`。

</details>

<details>
<summary>✅ v1.52 运维域数据权限统一收口（终态）(Phases 130-133) — SHIPPED 2026-11-07</summary>

- [x] Phase 130: ScopeProvider 基建 (DS-01/02) — completed 2026-11-07
- [x] Phase 131: 四域读端点 + Excel 导出独立收口 (DS-03/06) — completed 2026-11-07
- [x] Phase 132: 四域写端点收口 (DS-04) — completed 2026-11-07
- [x] Phase 133: 扩展实体 + vdi 合并迁移 + UAT (DS-05/06/07) — completed 2026-11-07

</details>

<details>
<summary>✅ v1.51 端口写生产级操作体验与平台质量收口 (Phases 121-129) — SHIPPED 2026-11-02</summary>

- [x] Phase 121: 端口写 dry-run 预览 + 命令预览 UI — completed 2026-10-31
- [x] Phase 122: 批量写实时进度 SSE/WS — completed 2026-11-01
- [x] Phase 123: 安全三件套 — 快照回滚 + 并发写互斥 + 不可达预检 — completed 2026-11-01
- [x] Phase 124: VLAN 扩展 — trunk native/allowed + hybrid — completed 2026-11-01
- [x] Phase 125: 端口安全 — port-security/sticky/静态 MAC 绑定 — completed 2026-11-01
- [x] Phase 126: Cisco IOS-XE 第 4 厂商 + 跨固件差异矩阵 — completed 2026-11-01
- [x] Phase 127: 审计 UI + 审计治理 + 3 厂商 e2e — completed 2026-11-01
- [x] Phase 128: 平台质量收口 — completed 2026-11-01
- [x] Phase 129: 目录改名 + 双线 planning 合流 + UAT 回补组织 — completed 2026-11-01

</details>

<details>
<summary>✅ v1.25 全站权限治理与样式收敛 (Phases 84-93) — SHIPPED 2026-10-30</summary>

- [x] Phase 84: 权限模型三缺口 — completed 2026-10-28
- [x] Phase 85: 后端动作级严格化 — completed 2026-10-29
- [x] Phase 86: 前端操作入口三态门控铺开 — completed 2026-10-29
- [x] Phase 87: 动作级权限授予闭环 — completed 2026-10-30
- [x] Phase 88: 验证底座与零风险清理 — completed 2026-10-29
- [x] Phase 89: 变量层与明暗色板重构 — completed 2026-10-29
- [x] Phase 90: Input 域交还 variant/token + 圆角统一 — completed 2026-10-30
- [x] Phase 91: Select / InputNumber / Form / Table / Modal 域交还 — completed 2026-10-30
- [x] Phase 92: 暗色 mega-cluster 合并 — completed 2026-10-30
- [x] Phase 93: room-devices 拆分 + 视觉回归验收 — completed 2026-10-30

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
*Roadmap reorganized: 2026-11-08 after v1.53 SHIPPED & GSD 归档（phase 详情归档至 `milestones/v1.53-ROADMAP.md`；权威由 GSD 自动生成体系承载）*
