# Roadmap: 工位导入部门/用户关联功能

## Milestones

- ✅ **v1.0 工位导入部门/用户关联** — Phases 1-2 (shipped)
- ✅ **v1.1 信息点导入设备端口关联** — Phase 3 (shipped 2026-04-16)
- ✅ **v1.2 可配置仪表盘生产级改造** — Phases 4-7 (shipped 2026-04-21)
- 🚧 **v1.3 技术债清理** — Phases 8-10 (in progress)

## Phases

<details>
<summary>✅ v1.0 工位导入部门/用户关联 (Phases 1-2) — SHIPPED</summary>

- [x] Phase 1: 后端导入配置 (3/3 plans)
- [x] Phase 2: 验证与导出 (4/4 plans)

</details>

<details>
<summary>✅ v1.1 信息点导入设备端口关联 (Phase 3) — SHIPPED 2026-04-16</summary>

- [x] Phase 3: 信息点导入设备端口配置 (1/1 plan)

</details>

<details>
<summary>✅ v1.2 可配置仪表盘生产级改造 (Phases 4-7) — SHIPPED 2026-04-21</summary>

- [x] Phase 4: Widget 数据获取机制 (3/3 plans)
- [x] Phase 5: 前端交互完善 (3/3 plans)
- [x] Phase 6: 实时数据刷新 (3/3 plans)
- [x] Phase 7: 用户体验优化 (2/2 plans)

</details>

---

## v1.3 技术债清理

### Phase 8: SNMP Panic 修复

**Goal:** 修复 scrapligo 传输层 panic 问题，确保 SNMP 操作稳定

**Depends on:** —

**Requirements:**
- SNMP-01a: 实现 panic 恢复包装器
- SNMP-01b: 实现并发安全机制
- SNMP-01c: 实现连接就绪性验证

**Success Criteria:**
1. SNMP Ping 操作不再导致应用崩溃
2. 高并发场景下连接池稳定（20+ 并发）
3. Panic 事件被正确记录和恢复

**Plans:**
- [x] 08-01: 添加 panic 恢复包装器和日志记录
- [x] 08-02: 实现 RWMutex 并发安全机制
- [x] 08-03: 添加连接就绪性验证和超时处理

---

### Phase 9: 后端代码优化

**Goal:** 清理死代码、修复安全问题、简化 Core 结构

**Depends on:** Phase 8

**Requirements:**
- CODE-02a: 删除死代码文件
- CODE-02b: 清理 Core 结构
- CODE-02c: 修复安全问题

**Success Criteria:**
1. 所有死代码已删除，构建通过
2. Core 结构清晰，依赖明确
3. 安全漏洞已修复

**Plans:**
- [ ] 09-01: 删除死代码并验证构建
- [ ] 09-02: 清理 Core 结构死字段
- [ ] 09-03: 修复安全问题和添加错误日志

---

### Phase 10: 网络设备导出集成

**Goal:** 集成现有导出功能到路由和前端

**Depends on:** Phase 9

**Requirements:**
- EXPORT-03a: 注册导出路由
- EXPORT-03b: 前端集成
- EXPORT-03c: 厂商特定格式

**Success Criteria:**
1. 用户可以从 UI 导出设备配置
2. 支持主流厂商格式
3. 大数据集导出稳定

**Plans:**
- [ ] 10-01: 注册导出路由和测试端点
- [ ] 10-02: 前端导出按钮和进度显示
- [ ] 10-03: 实现厂商特定格式支持

---

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. 后端导入配置 | v1.0 | 3/3 | Complete | — |
| 2. 验证与导出 | v1.0 | 4/4 | Complete | — |
| 3. 信息点导入设备端口配置 | v1.1 | 1/1 | Complete | 2026-04-16 |
| 4. Widget 数据获取机制 | v1.2 | 3/3 | Complete | 2026-04-20 |
| 5. 前端交互完善 | v1.2 | 3/3 | Complete | 2026-04-21 |
| 6. 实时数据刷新 | v1.2 | 3/3 | Complete | 2026-04-21 |
| 7. 用户体验优化 | v1.2 | 2/2 | Complete | 2026-04-21 |
| 8. SNMP Panic 修复 | v1.3 | 0/3 | Not started | — |
| 9. 后端代码优化 | v1.3 | 0/3 | Not started | — |
| 10. 网络设备导出集成 | v1.3 | 0/3 | Not started | — |
