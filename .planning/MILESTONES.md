# Milestones

## v1.17 资产对账 (Asset Reconciliation) — ✅ SHIPPED 2026-07-03

**Phases**: 5 (Phases 42-46) | **Plans**: 14 (16 planned, 42-03 skeleton superseded by R3) | **Tasks**: ~70

**Delivered**: 建立"实物层 vs 声明层"对账引擎（R1-R5 完整）+ 4 个根因修复（Phase 47 独立）。Observe-only 策略 + 告警驱动人工修复 + 工单闭环 + IP 段例外 + 半自动修复建议（高置信度 6 状态机）。

**Key Accomplishments**:
1. ✅ **R1 观测底座** — 物化视图 + 主表 + Dashboard + 6 个 Statistics 端点（防 MaxPageSize 钳制）+ operlog 全覆盖
2. ✅ **R2 告警 + 工单闭环** — critical/high 自动转工单 + WebSocket 推送 + SysNotice + 7d 静默期 + 24h 节流
3. ✅ **R3 IP 段例外** — CIDR 格式 + 5 种 actions 组合 + expires_at + 命中测试工具 + Excel 导入导出 + VERIFICATION 10/10
4. ✅ **R4 工位详情整合** — 健康度卡片（5 KPI + 趋势 mini chart）+ 对账健康列 + ReconciliationDrawer 3 Tab
5. ✅ **R5 半自动修复** — 6 状态机 + 7d 回滚 + 误修复率告警三通道（SysNotice+WS+operlog）+ UAT 9/10 PASS
6. ✅ **Phase 47 根因修复** — infoPoint drift / Layer3 UPSERT / port_status 漂移 / MAC parser 校验（独立完成）

**Known in-session fixes during UAT**:
- Apply 400: `resolution_method` → `resolution_note` 列不存在 bug (commit 87d5fc82)
- Rollback 400: 删除 `pre_fix_user_id == nil` 过度防御检查
- Cron 5 fields 缺年份: UPDATE sys_job 加 `0` 前缀

**Known deferred items at close**: 0 (Test 10 权限非 admin 端到端 skipped due to 缺测试账号，不阻塞)

---

## v1.3 技术债清理 — ✅ SHIPPED 2026-04-27

**Phases**: 3 (Phases 8-10) | **Plans**: 9 | **Tasks**: ~50

**Delivered**: 修复 SNMP panic 崩溃问题，清理后端死代码并修复安全漏洞，实现网络设备批量导出功能

**Key Accomplishments**:
1. ✅ SNMP panic 完全修复 — panic 恢复包装器 + RWMutex 并发安全 + WaitForReady 连接验证
2. ✅ 后端代码质量提升 — 删除 Core 死字段，文档化 12 个外部依赖字段
3. ✅ 安全修复完成 — WebSocket CheckOrigin 增强，并发安全，错误日志改进，15 个新测试全部通过
4. ✅ 批量导出功能 — 支持一次性选择多个实体类型并打包为 ZIP 下载，集成到 9 个网络管理页面

**Known deferred items at close**: 0 (all resolved)

---

## v1.2 可配置仪表盘生产级改造（仿 Zabbix）— ✅ SHIPPED 2026-04-21

**Phases**: 4 (Phases 4-7) | **Plans**: 11 | **Tasks**: ~60

**Delivered**: 实现可配置仪表盘系统的核心功能，包括 Widget 数据获取机制、前端交互完善、实时数据刷新和用户体验优化

**Key Accomplishments**:
1. ✅ Widget 数据获取机制 — `GetWidgetData` 和 `GetBatchWidgetData` 支持 API/WebSocket/Static 数据源
2. ✅ 前端交互完善 — Widget 选择器、仪表盘设置面板、模板预览功能
3. ✅ 实时数据刷新 — WebSocket 连接管理、基于刷新间隔的轮询、缓存策略
4. ✅ 用户体验优化 — Widget 拖拽布局优化、加载状态和错误处理、响应式布局适配

---

## v1.1 信息点导入设备端口关联 — ✅ SHIPPED 2026-04-16

**Phases**: 1 (Phase 3) | **Plans**: 1 | **Tasks**: ~5

**Delivered**: 信息点 Excel 导入时支持通过设备名/端口名自动匹配并关联设备 ID 和端口 ID

**Key Accomplishments**:
1. ✅ 信息点导入配置添加"所属设备"列 (`deviceName` → `device_id`)
2. ✅ 信息点导入配置添加"所属端口"列 (`portName` → `port_id`)
3. ✅ 两个新列均为可选，空值不阻断导入
4. ✅ 模板下载包含设备和端口示例值

---

## v1.0 工位导入部门/用户关联 — ✅ SHIPPED 2026-04-16

**Phases**: 2 (Phases 1-2) | **Plans**: 7 | **Tasks**: ~30

**Delivered**: 工位 Excel 导入时支持通过部门名称/用户名自动匹配并关联部门 ID 和用户 ID

**Key Accomplishments**:
1. ✅ Excel 导入模板添加"所属部门"和"所属用户"两个可选列
2. ✅ 导入时通过部门名称/用户名匹配，写入对应 ID
3. ✅ 匹配失败留空不阻断导入
4. ✅ 前端工位列表/详情页面展示关联的部门和用户信息
