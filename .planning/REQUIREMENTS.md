# Requirements: XingRan-Next（RuoYi-Next 运维管理系统）

**Defined:** 2026-10-31
**Core Value:** 网络设备端口写能力深化到生产级操作体验（dry-run、实时进度、回滚、并发安全），同步清偿平台技术债与收尾杂项，使端口写从「能写」进化为「敢写、可审计、可观测」。

## v1.51 Requirements

范围裁定（用户 2026-10-31）：**执行 A（端口写深化）+ B（平台技术债）+ D（收尾杂项）**；C 组（RPA 拖拽编辑器/插件录制）不入本里程碑。

### 端口写操作体验（PORTW）

- [ ] **PORTW-01**: 端口写命令 **dry-run 预览**——不下发生成将执行的命令序列供人工确认
- [ ] **PORTW-02**: 命令预览 UI（与 dry-run 配套的前端展示）
- [ ] **PORTW-03**: 批量写**实时进度**（SSE/WebSocket，BATCH-05 v1.20 遗留正式清偿）
- [ ] **PORTW-04**: critical 操作**快照 + 自动回滚**（执行前抓取端口状态，失败时 reverse 恢复）
- [ ] **PORTW-05**: 多用户并发写互斥（同端口同一时刻仅允许一个写会话）
- [ ] **PORTW-06**: 设备不可达预检（下发前 1s 级连通性探测）

### 端口写能力扩展（PWX）

- [ ] **PWX-01**: VLAN 扩展——trunk VLAN 修改（native/allowed list）+ hybrid 模式（华为/H3C）
- [ ] **PWX-02**: 端口安全——port-security enable/disable、max-mac-num、sticky MAC、静态 MAC 1:1 绑定
- [ ] **PWX-03**: Cisco IOS-XE/XR 第 4 厂商支持 + 跨固件版本命令差异矩阵
- [ ] **PWX-04**: 端口写审计明细 UI（`sys_port_write_audit` 可视化查询）
- [ ] **PWX-05**: 3 厂商（华为/锐捷/H3C）e2e fixture 全覆盖
- [ ] **PWX-06**: HTTP handler 层 e2e 测试（gin test engine）

### 平台技术债（DEBT）

- [ ] **DEBT-01**: 补 `internal/services/addomain` 测试（现 51.4%），后端覆盖率阈值从 76.5 **恢复到 77.5**
- [ ] **DEBT-02**: 端口写审计治理——批量写审计行移入事务 + `operator` 字段补索引（千万行级表）
- [ ] **DEBT-03**: RPA Worker API Token 认证加固（方案已设计于 docs，落地实施）
- [ ] **DEBT-04**: `operlog.exclude_paths` 配置驱动白名单完善
- [ ] **DEBT-05**: `permission.grant.test.tsx` 时序根治（替换 15s 重试补丁，修复 antd Form connect 时机依赖）
- [ ] **DEBT-06**: 本地目录改名 `xingran-react-frontend` → `xingran-react-frontend` + 本地/GitHub 双线 `.planning` 合流收尾

### 收尾杂项（TIDY）

- [ ] **TIDY-01**: 服务端排序 4 页面菜单注册（值班池/排班/知识文章/网络模板——代码已完成，路由不可达）
- [ ] **TIDY-02**: v1.25 三个 phase 的 UAT human_needed 回补组织（84/87/90；PORTW 真机 site-visit 19 项 deferred UAT 一并纳入，**需真机在场**）
- [ ] **TIDY-03**: RPA selector learner 两项 TODO（使用统计提取、更新通知机制）

## Deferred（本里程碑明确不做）

- RPA 可视化拖拽编辑器 / 浏览器插件录制（→ 候选 v1.52+）
- 对账 MV 刷新 30s 超时根治（继续等待 DBA 诊断回传）
- 物理参数写（速率/双工/MTU）
- 前端 TypeScript 全量严格化

## Traceability

| REQ 组 | Phase | 备注 |
|---|---|---|
| PORTW-01/02 | 121 | dry-run 是后续一切安全机制的基座，最先落地 |
| PORTW-03 | 122 | BATCH-05 独立成 phase（SSE/WS 基建） |
| PORTW-04/05/06 | 123 | 安全三件套（回滚/互斥/预检） |
| PWX-01 | 124 | VLAN 扩展 |
| PWX-02 | 125 | 端口安全 |
| PWX-03 | 126 | Cisco 第 4 厂商 |
| PWX-04/05/06 + DEBT-02 | 127 | 审计 UI + 审计治理 + e2e 补强 |
| DEBT-01/03/04/05 + TIDY-01/03 | 128 | 平台质量收口 |
| DEBT-06 + TIDY-02 | 129 | 目录改名/合流/UAT 回补（需窗口与真机） |
