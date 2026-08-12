# Phase 22A 执行方案更新

**日期**: 2026-05-25  
**状态**: 已审查现有基础设施，制定补充方案

---

## 方案 1 修订：基于现有基础设施的补充执行

### 原方案 1 问题

原审计报告建议**从零创建**测试基础设施、审计日志等，这会与项目现有实现重复。

### 修订方案

**核心原则**: 复用现有实现，只补充 VDI 专用部分

---

## 现有基础设施（可复用）

| 组件 | 现有实现 | 复用方式 |
|------|----------|----------|
| **错误处理** | `pkg/errors/errors.go` | ✅ 扩展添加 VDI 错误码 |
| **操作日志** | `pkg/middleware/oper_log.go` | ✅ 添加 `/vdi/*` 路径到配置 |
| **审计日志模型** | `internal/models/rpa/audit_log.go` | ✅ 创建 VDI 审计服务使用该模型 |
| **测试框架** | httptest + testify | ✅ 参考 `auth_integration_test.go` 模式 |
| **密码加密** | `internal/services/addomain/utils.go` | ✅ Phase 22A 已计划复用 |

---

## 补充内容（仅需新增）

### P0 必须新增

1. **VDI Mock Server** - `internal/services/vdi/mock_server.go`
   - 模拟深信服 VDI API 的 11 个端点
   - 支持测试失败场景
   - 统计 API 调用次数

2. **VDI 专用错误码** - 扩展 `pkg/errors/errors.go`
   - 错误码 54001-54100
   - 便捷错误函数（`VMNotFound()`, `VDIApiFailed()` 等）

### P1 强烈建议

3. **VDI 审计日志服务** - `internal/services/vdi/audit_service.go`
   - 复用 `rpa.AuditLog` 模型
   - 记录所有 VM 操作

4. **VDI Client 管理器** - `internal/services/vdi/client_manager.go`
   - 单例模式
   - 使用 `sync.Map` 缓存客户端

---

## 执行计划

### 阶段 1: 基础设施补充（1-2 天）

```
任务 1.1: 创建 VDI Mock Server
任务 1.2: 扩展 VDI 错误码
任务 1.3: 创建审计日志服务
任务 1.4: 创建 ClientManager
任务 1.5: 编写单元测试模板
```

### 阶段 2: 执行 Phase 22A（2-3 天）

```
/gsd-execute-phase 22a-vdi-basic-integration
```

---

## 详细文档

完整的补充计划见：**`INFRASTRUCTURE_SUPPLEMENT.md`**

包含：
- Mock Server 完整实现代码
- 错误码定义列表
- 审计日志服务接口
- ClientManager 实现细节
- 集成到 Phase 22A 各 Wave 的具体位置

---

## 时间估算

| 阶段 | 原估算 | 修订估算 | 差异 |
|------|--------|----------|------|
| 基础设施补充 | - | 1-2 天 | 新增 |
| Wave 1-5 执行 | 12-15h | 12-15h | 不变 |
| **总计** | - | **14-17h** | +1-2天 |

---

## 验证标准更新

**新增成功标准**：
11. [ ] VDI Mock Server 覆盖所有 11 个 API 端点
12. [ ] VDI 错误码正确定义和映射
13. [ ] 所有 VM 操作有审计日志记录
14. [ ] ClientManager 单例模式工作正常
15. [ ] 单元测试覆盖率 >80%

**保持原成功标准**：
1-10 项（原计划已定义）

---

## 风险降低

| 原风险 | 修订后风险 | 缓解措施 |
|--------|-----------|----------|
| 测试基础设施缺失 | 低 | 复用项目测试模式 |
| 审计日志未知 | 低 | 复用 RPA 审计模型 |
| 重复开发 | 无 | 明确复用原则 |

---

## 下一步行动

### Option A: 立即开始基础设施补充
```bash
# 开始执行补充任务
/gsd-quick "创建 VDI Mock Server 和错误码定义"
```

### Option B: 审查补充计划后再开始
```bash
# 审查 INFRASTRUCTURE_SUPPLEMENT.md
/gsd-note "审查基础设施补充计划"
```

### Option C: 直接执行 Phase 22A（不推荐）
```bash
# 缺少基础设施，风险较高
/gsd-execute-phase 22a-vdi-basic-integration
```

---

**建议**: 选择 **Option A**，先完成基础设施补充，确保生产级质量。

---

**更新完成**: 2026-05-25  
**相关文档**: 
- `PLAN_QUALITY_AUDIT.md` - 原审计报告
- `INFRASTRUCTURE_SUPPLEMENT.md` - 补充实现细节
- `22-01-PLAN.md` ~ `22-05-PLAN.md` - 原 Phase 22A 计划
