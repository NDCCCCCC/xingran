---
phase: quick-260814-ehg
plan: 01
subsystem: database-seed
tags: [menu, dedupe, data-migration, dev-db, admin-role]
requires:
  - xingran_menus_clean.sql (生产库菜单导出，386 sys_menu + 309 sys_role_menu + 5 sys_role)
provides:
  - xingran_menus_dedup.sql (239 条去重幂等 INSERT)
  - dedupe-report.md (R0-R5 映射报告)
  - dev 库全量菜单 + admin 全量授权
affects:
  - dev DB sys_menu (36 → 239 存活)
  - dev DB sys_role_menu (admin 239 条)
tech-stack:
  added: []
  patterns: [字符级 SQL 状态机解析, idMap 传递闭包重定向, fixpoint 同级去重, 软删归并]
key-files:
  created:
    - xingran_menus_dedup.sql
    - .planning/quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/dedupe-report.md
  modified: []
  deleted:
    - scripts/tmp_menuimport/ (临时工具，用完即删；parse.go/dedupe.go/main.go 在 git 历史 ef1ba87/cb81443 中)
decisions:
  - "R1 规范 id 优先级: 存活 > 存活子树大 > path 非空 > created_at 早 > id 升序"
  - "dev 既有同名顶级目录归并用软删（可恢复），不硬删"
  - "perms 字符串原样保留（xxx:list vs xxx:query 版本差异仅记录不处理）"
metrics:
  duration: ~75min (含 dev pooler 慢连接的 3 次 import 运行)
  completed: 2026-08-14
---

# Quick 260814-ehg: 菜单数据去重并导入 dev 库 Summary

**One-liner:** 字符级 SQL 解析器读出生产菜单 386 行 → R0-R5 去重为 239 行保留集（顶级目录同名唯一）→ 幂等导入 dev 并把 dev 原 36 条同名「运维管理」子树软删归并，admin 角色获全量 239 菜单授权，五项复核全 PASS。

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | 字符级 SQL 解析器（386+309+5 计数断言） | ef1ba87 | scripts/tmp_menuimport/parse.go, main.go |
| 2 | 去重引擎 R0-R5 + 干净 SQL + 映射报告 | cb81443 | +dedupe.go, xingran_menus_dedup.sql, dedupe-report.md |
| 3 | 导入 dev + admin 授权 + 复核 + 清理 | e02d837 | -scripts/tmp_menuimport/（含 import.go，见偏差 3） |

## 去重结果概要（R0-R5）

- **输入**: sys_menu 386 条（含 147 条软删）、sys_role_menu 309 条、sys_role 5 条
- **保留集: 239 行**；折叠组: R1 ×2 组（折叠 7 id）；R3 ×0 组（R0 过滤后存活行无同级重复）
- **R0 软删过滤**: 147 条不进保留集；含「网络设备管理test」`5cd243d3`（软删空壳，单列一组随 R0 消除）；复活 0 条（无存活行的祖先被软删的情况）
- **R1 顶级目录合并**:
  - 「系统监控」×6 → 1：规范 `3f5f2844`（存活、子树 5、path=monitor、created 最早），折叠 `84dd9afb`/`6c3b308d`/`4787c446`/`db33c67c`/`53bfe9e9`（全软删）
  - 「网络设备」×3 → 1：规范 `0013f129`（存活、子树 15、path=network），折叠 `95f849d7`/`f5c087d6`（软删空壳）
- **R2**: idMap 传递闭包重定向 parent_id，闭包无环（不变量④代码断言）
- **R3**: fixpoint 同级 (parent,name,type,perms) 去重——0 折叠（「值班池管理」×4 中 c50b5b01 下 4 条全部软删由 R0 消除，存活副本 `c4a5ff39` 挂 `e7d2962c` 下天然唯一）
- **R4**: sys_role_menu 309 条映射后 0 条引用折叠 id，(role_id,menu_id) 去重后仍 309——仅报告，不导入
- **R5**: sys_role 5 条不导入（dev 复用现有 admin/user）

详见 `dedupe-report.md`（同目录）。

## 导入验证结果（dev 库）

- **sys_menu 存活总数**: 36（导入前）→ 275（导入 239 后）→ **239**（归并软删 dev 原 36 条后）；保留集 239/239 全部在库
- **顶级目录数**: 10 个顶级入口（9 M + 1 C 仪表盘），同名唯一（复核② PASS）
- **admin 菜单数**: **239**（保留集全覆盖，缺失 0；admin role_id=`ebbd7c15-da04-475e-9b42-dc505308759b`）
- **无悬空 parent_id**: dangling=0（复核③ PASS）
- **幂等性**: 第 2/3 次运行 `新增 0 / 归并跳过 / 补 0`，全部 VERIFY PASS

### 五项复核（最终运行 transcript）

```
复核[PASS] ①菜单总数: sys_menu 存活=239，保留集 239 条在库 239 条
复核[PASS] ②顶级目录唯一: 无重复
复核[PASS] ③无悬空parent_id: dangling=0
复核[PASS] ④admin全量授权: admin 菜单总数=239，保留集缺失=0
复核[PASS] ⑤最终目录树（见下）
VERIFY PASS
```

### 最终目录树（顶级 + 子节点计数）

```
仪表盘(C) [0]
网络设备(M) [9]   — 设备发现/设备管理/授权凭证/命令分发/配置执行/配置备份/配置模板/MAC地址/端口状态
空间管理(M) [9]   — 楼宇空间/楼宇空间3D/楼宇/楼层/工位/信息点/机房/专线/机房设备
运维管理(M) [11]  — 工单系统/值班管理/知识库文章/知识库查看 + 楼宇/楼层/工位/信息点/机房/专线/机房设备
AD域控(M) [6]     — 域控用户/域用户组/电脑设备/组织单元/同步日志/域控配置
资产管理(M) [5]   — 资产列表/对账看板/异常列表/例外规则管理/修复建议
虚拟机(M) [4]     — 虚拟机列表/虚拟机详情/VDI服务器配置/RPA 管理
系统监控(M) [5]   — 监控仪表盘/服务监控/日志管理/定时任务/缓存管理
系统管理(M) [10]  — 系统用户/角色/菜单/部门/岗位/字典/参数配置/通知公告/系统设置/密钥列表
用户中心(M) [3]   — 个人中心/用户设置/我的通知
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] pg name[] vs text[] 类型不匹配**
- **Found during:** Task 3 约束探测
- **Issue:** `array_agg(attname) = ARRAY[...]` 报 `operator does not exist: name[] = text[]`
- **Fix:** `array_agg(...)::text[]` 显式转换
- **Commit:** e02d837（工具已删，逻辑见 git 历史与本文件）

**2. [Rule 1 - Bug] dev sys_role_menu 无 id 列**
- **Found during:** Task 3 admin 授权
- **Issue:** 计划假设 dev 表结构与生产一致（id/role_id/menu_id/created_at 4 列），实际 dev 只有 (role_id, menu_id) 两列且无任何唯一约束，`ON CONFLICT (role_id,menu_id)` 不可用
- **Fix:** information_schema 探测列与约束；无 id 列则两列 INSERT，无唯一约束则「先 SELECT 跳过已存在对」保证幂等（计划预见的降级路径）

**3. [Rule 3 - Blocking] dev 既有 36 条菜单与保留集同名不同 id**
- **Found during:** Task 3 复核② FAIL（运维管理×2）
- **Issue:** 计划预期 dev 36 条与保留集「同 id ON CONFLICT 跳过」，实际 dev 顶级「运维管理」`73e43260`（2026-08-13 迁移创建）与生产 `c50b5b01` 不同 id，导入后顶级重名
- **Fix:** 新增步骤 2.5 归并：规范 id=保留集成员（子树为严格超集——已核实 dev 7C+28F perms 与 c50b5b01 对应节点完全一致）；被折叠目录子节点按 (name,type,perms) 匹配则整支软删、不匹配则重挂规范 id；最后软删被折叠目录本身并清理其 sys_role_menu 行。软删可恢复，非硬删
- **Result:** 软删 36 行，role_menu 同步清理；归并幂等（重跑无重名即跳过）

**4. [Rule 1 - Bug] 复核① 存活计数在归并前采样**
- **Found during:** Task 3 输出核对（显示 275 而非实际 239）
- **Fix:** 归并后重新 SELECT count 刷新 postAlive

### 其他偏差

- **import.go 未单独入 git 历史**: Task 3 的 import.go 在 cb81443 之后创建、e02d837 之前删除，未形成独立 commit；其完整逻辑（连接/预检/单事务导入/归并/授权/五项复核）记录于本 SUMMARY 与 commit message。parse.go/dedupe.go/main.go 在 ef1ba87/cb81443 历史中可查
- **TDD 标记**: Task 1/2 标 tdd=true，但为一次性迁移脚本，采用「断言内置于运行模式」的验证方式（-mode=parse 计数断言、-mode=gen 不变量断言失败即 exit 1），未建独立测试文件；脚本目录已删
- **两次瞬时网络失败**: dev pooler i/o timeout（连接阶段），重跑即恢复；与代码无关

## Known Stubs

None.

## 遗留事项（仅记录不处理）

1. **perms 版本差异**: 保留集含 `xxx:list`（旧版）与 `xxx:query`（新版）两种风格 perms（如 `system:user:list` vs `ops:workstation:query`），本次原样保留未对齐。若后端权限中间件或前端按钮按某一风格校验，可能出现按钮可见性差异——后续需要对齐时另开任务
2. **空间管理 vs 运维管理内容部分重复**: 生产数据本身在两个顶级 M 下各有一套楼宇/楼层/工位等 C 目录（不同 parent，不违反不变量），忠实导入未合并；是否需要合并属业务决策
3. **dev sys_role_menu 无约束**: (role_id,menu_id) 无唯一约束，幂等靠应用层 SELECT 跳过；建议后续迁移补唯一约束

## Self-Check

- [x] xingran_menus_dedup.sql 存在（239 INSERT + ON CONFLICT DO NOTHING）
- [x] dedupe-report.md 存在（含 R0/R1/R3/R4 各组映射）
- [x] commit ef1ba87 / cb81443 / e02d837 均在 git log
- [x] scripts/tmp_menuimport/ 已删除，`go build ./...` exit 0
- [x] dev 库五项复核 VERIFY PASS（最终运行 exit 0）

## Self-Check: PASSED
