# Phase 22 拆分完成指南

**拆分完成时间**: 2026-05-25
**原阶段**: Phase 22-sangfor-vdi-integration (10 Waves)
**拆分后**: 4 个子阶段 (22A, 22B, 22C, 22D)

---

## 📂 目录结构

```
.planning/phases/
├── 22-sangfor-vdi-integration/        # 原始目录（保留，用于参考）
│   ├── RESEARCH.md
│   ├── PATTERNS.md
│   ├── 22-CONTEXT.md
│   ├── PHASE_SPLIT.md                 # 拆分方案说明
│   ├── PLAN-ASSESSMENT.md             # 评估报告
│   └── ... (其他参考文档)
│
├── 22a-vdi-basic-integration/         # Phase 22A: VDI 基础集成
│   ├── PHASE.md                       # 阶段配置
│   └── plans/
│       ├── 22-01-PLAN.md
│       ├── 22-02-PLAN.md
│       ├── 22-03-PLAN.md
│       ├── 22-04-PLAN.md
│       └── 22-05-PLAN.md
│
├── 22b-vm-agent-service/              # Phase 22B: VM Agent 服务
│   ├── PHASE.md
│   └── plans/
│       └── 22-06-PLAN.md
│
├── 22c-account-management/            # Phase 22C: 账号管理与密码
│   ├── PHASE.md
│   └── plans/
│       ├── 22-07-PLAN.md
│       └── 22-08-PLAN.md
│
└── 22d-webconsole-monitoring/         # Phase 22D: Web Console 与监控
    ├── PHASE.md
    └── plans/
        ├── 22-09-PLAN.md
        └── 22-10-PLAN.md
```

---

## 🚀 执行命令

### 顺序执行（推荐）

```bash
# 1. 执行 Phase 22A: VDI 基础集成 (12-15h)
/gsd-execute-phase 22a-vdi-basic-integration

# 2. 执行 Phase 22B: VM Agent 服务 (6-8h)
/gsd-execute-phase 22b-vm-agent-service

# 3. 执行 Phase 22C: 账号管理与密码 (7-9h)
/gsd-execute-phase 22c-account-management

# 4. 执行 Phase 22D: Web Console 与监控 (6-8h)
/gsd-execute-phase 22d-webconsole-monitoring
```

### 按 Wave 执行（可选）

每个子阶段也可以按 Wave 顺序执行：

```bash
# Phase 22A 示例
/gsd-execute-phase 22a-vdi-basic-integration --wave 1
/gsd-execute-phase 22a-vdi-basic-integration --wave 2
/gsd-execute-phase 22a-vdi-basic-integration --wave 3
/gsd-execute-phase 22a-vdi-basic-integration --wave 4
/gsd-execute-phase 22a-vdi-basic-integration --wave 5
```

---

## 📊 子阶段概览

| 子阶段 | Waves | 工作量 | 状态 | 说明 |
|--------|-------|--------|------|------|
| **22A** | 1-5 | 12-15h | 📋 Planned | VDI 基础集成，可独立使用 |
| **22B** | 6 | 6-8h | 📋 Planned | VM Agent 服务 |
| **22C** | 7-8 | 7-9h | 📋 Planned | 账号管理与密码轮换 |
| **22D** | 9-10 | 6-8h | 📋 Planned | Web Console 与监控 |
| **总计** | 10 | 31-40h | - | 原 Phase 22 内容 |

---

## 🔄 依赖关系

```
22A (VDI 基础)
    │
    ├─→ 可独立交付使用
    │
    └─→ 22B (VM Agent)
            │
            └─→ 22C (账号与密码)
                    │
                    └─→ 22D (Console 与监控)
```

**关键点**:
- 22A 完成后可独立使用（基础的 VDI 管理功能）
- 22B、22C、22D 需要按顺序执行
- 22B 可与 22A 的尾部并行开发

---

## ✅ 验证拆分结果

### 检查文件结构

```bash
# 检查所有子阶段目录
ls -la .planning/phases/22*

# 检查 Phase 文件
cat .planning/phases/22a-vdi-basic-integration/PHASE.md
cat .planning/phases/22b-vm-agent-service/PHASE.md
cat .planning/phases/22c-account-management/PHASE.md
cat .planning/phases/22d-webconsole-monitoring/PHASE.md

# 检查 Wave 文件
ls -la .planning/phases/22a-vdi-basic-integration/plans/
ls -la .planning/phases/22b-vm-agent-service/plans/
ls -la .planning/phases/22c-account-management/plans/
ls -la .planning/phases/22d-webconsole-monitoring/plans/
```

### 验证 phase 字段

```bash
# 确认所有 Wave 文件的 phase 字段已更新
grep "^phase:" .planning/phases/22*/plans/*.md

# 预期输出：
# 22a-vdi-basic-integration/plans/22-01-PLAN.md:phase: 22a-vdi-basic-integration
# 22b-vm-agent-service/plans/22-06-PLAN.md:phase: 22b-vm-agent-service
# 22c-account-management/plans/22-07-PLAN.md:phase: 22c-account-management
# 22d-webconsole-monitoring/plans/22-09-PLAN.md:phase: 22d-webconsole-monitoring
```

---

## 📝 ROADMAP 更新

ROADMAP.md 已更新为：

```markdown
- 📋 v1.11 深信服桌面云集成 (Phases 22A-22D) — PLANNED
  - [ ] Phase 22A: VDI 基础集成 (5 plans) — 12-15h
  - [ ] Phase 22B: VM Agent 服务 (1 plan) — 6-8h
  - [ ] Phase 22C: 账号管理与密码 (2 plans) — 7-9h
  - [ ] Phase 22D: Web Console 与监控 (2 plans) — 6-8h

依赖关系: 22A → 22B → 22C → 22D

说明: 原 Phase 22 拆分为 4 个子阶段，每个阶段可独立执行，上下文更充足
```

---

## 🎯 执行建议

### 1. 前置条件

在执行任何子阶段之前，请确认：

```bash
- [ ] Go 1.24+ 环境
- [ ] Node.js 20+ 环境
- [ ] PostgreSQL 18+ 数据库
- [ ] Redis 7.4+ 缓存
- [ ] 深信服 VDI 服务器访问权限
- [ ] VDI API 文档（V1.2）
```

### 2. 执行顺序

**推荐顺序**: 22A → 22B → 22C → 22D

**原因**:
- 22A 提供基础数据模型和 API
- 22B 需要 22A 的数据模型
- 22C 需要 22A 的数据模型和 22B 的 Agent
- 22D 需要前面的所有功能

### 3. 每个子阶段执行后

每个子阶段完成后，建议：

1. **运行测试**: `go test ./...` 和 `npm test`
2. **验证功能**: 手动测试关键功能
3. **提交代码**: 提交已完成的工作
4. **更新文档**: 记录任何变更或问题

---

## 🔧 故障排查

### 问题：找不到 Wave 文件

**症状**: `/gsd-execute-phase` 报错找不到 Wave 计划文件

**解决**:
```bash
# 检查 plans 目录是否存在
ls -la .planning/phases/22a-vdi-basic-integration/plans/

# 检查文件权限
chmod +x .planning/phases/22*/plans/*.md
```

### 问题：phase 字段未更新

**症状**: Wave 文件中的 phase 仍指向旧的 `22-sangfor-vdi-integration`

**解决**:
```bash
# 批量更新 phase 字段
find .planning/phases/22*/plans/ -name "*.md" -exec sed -i 's/phase: 22-sangfor-vdi-integration/phase: [对应phase]/' {} \;
```

### 问题：依赖关系错误

**症状**: Wave 执行顺序不正确

**解决**:
```bash
# 检查每个 Wave 的 depends_on 字段
grep -A2 "wave:" .planning/phases/22*/plans/*.md
grep -A2 "depends_on:" .planning/phases/22*/plans/*.md
```

---

## 📚 参考文档

拆分相关的所有文档都保存在原 Phase 22 目录中：

- `PHASE_SPLIT.md` - 拆分方案详细说明
- `PLAN-ASSESSMENT.md` - 完整性评估报告
- `22-CONTEXT.md` - 架构决策和上下文
- `RESEARCH.md` - 研究文档
- `PATTERNS.md` - 代码模式参考

这些文档对**所有**子阶段都适用。

---

## ✅ 拆分完成检查清单

- [x] 创建 4 个子阶段目录
- [x] 创建每个子阶段的 PHASE.md 文件
- [x] 复制 Wave 计划文件到对应目录
- [x] 更新 Wave 文件的 phase 字段
- [x] 更新 Phase 22A Wave 依赖关系
- [x] 创建 PHASE_SPLIT.md 说明文档
- [x] 更新 ROADMAP.md
- [x] 创建本指南 (SPLIT_GUIDE.md)

**状态**: ✅ 拆分完成，可以开始执行

---

**拆分完成时间**: 2026-05-25
**下一步**: 执行 `/gsd-execute-phase 22a-vdi-basic-integration`
