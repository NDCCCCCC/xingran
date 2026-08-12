---
title: sys_workstation 表误删 - 从 工作簿2.xlsx 恢复
status: resolved
slug: sys-workstation-recovery-from-xlsx
created: 2026-07-01
resolved: 2026-07-01
severity: critical
data_loss: full_table
---

# sys_workstation 数据恢复 (2026-07-01)

## 症状

用户误删 `sys_workstation` 表全部数据,但有备份文件 `C:\Users\CPIC\Downloads\工作簿2.xlsx`(227 KB)。

## 备份文件分析

- **格式**: PostgreSQL `COPY` 风格 dump (不是设计过的 Excel 模板)
- **维度**: Sheet1,1447 行 × 27 列
- **NULL 标记**: 字面量 `\N` (部分列用空字符串)
- **列结构**: 与 `sys_workstation` 表 1:1 对应,含系统列(id/created_at/updated_at/deleted_at/created_by/updated_by/version)
- **数据特点**:
  - 33 个唯一 dept_id
  - 9 个唯一 floor_id
  - 3 个唯一 building_id
  - 234 个唯一 user_id
  - 所有 1447 行 width=160,depth=70 (常量)
- **编码问题**: 5 行 `workstation_name` 有 GBK→UTF-8 mojibake(`1FVIP��`、`6Fʳ�úͻ��ǽ��` 等),保留原样

## 决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 恢复方法 | 直接 SQL INSERT | 1:1 还原 id/timestamp/version |
| 编码异常 | 原样保留 | 数据 1:1 还原,后续单独修复 |
| DB 连接 | 项目自带 PG (10.62.10.34/xingran) | 生产库直接恢复 |
| 验证 | 前后 COUNT + 抽样 5 行 | 双向校验确保完整 |

## 不可用现成 Excel 导入服务的原因

1. ExcelConfig 期望中文列头("工位名称"),文件是英文 `workstation_name`
2. `prepareRecordsForUpsert` 强制按 `user_id` 覆盖 `status` 字段
3. 不处理 `id/created_at/updated_at/version` 等系统列
4. `position_x/y, width, depth, desk_type, rotation` 不在 ExcelConfig 里

## 实施结果

| 步骤 | 状态 | 结果 |
|---|---|---|
| 1. 收集症状和方案分析 | ✅ | Excel 是 PG COPY 风格 dump,1447 行 × 27 列 |
| 2. 写一次性 Go 脚本 | ✅ | `cmd/restore_workstation/main.go` 支持 convert/count/exec/dry 模式 |
| 3. go build 验证编译 | ✅ | 通过 |
| 4. 生成 `restore_workstation.sql` (1447 行 INSERT) | ✅ | 506 KB,BEGIN/COMMIT 包裹 |
| 5. 导入前 COUNT | ✅ | 0 行 (符合预期) |
| 6. 事务中执行 SQL | ✅ | 1447 行写入成功 |
| 7. 导入后 COUNT=1447 + 抽样 5 行字段对比 | ✅ | 全部字段与 Excel 一致 |
| 8. 清理临时文件 | ✅ | tmp_restore.exe / restore_workstation.sql / cmd/restore_workstation / cmd/sample_check_tmp 全部删除,`go build ./...` 通过 |

## 验证详情

```
总行数 = 1447 (期望 1447) ✅
去重 dept=32 floor=9 building=2 user=233 (Excel: dept=33 floor=9 building=3 user=234)
  → 微小差异是 Excel 把 \N 算入 distinct 的统计口径差异,不影响数据

抽样 5 行字段全部与 Excel 一致:
  [1] id=8cb1110a-…aeb1 1F001  dept=859f770a-…8153 user=52c255db-…e05d floor=5027dcc7-…af22
  [2-5] 6F食堂打菜台系列 name/created_at/floor_id/building_id 全部正确

version IS NULL = 1182 行 (Excel 原值 \N),DB 列 nullable=YES 允许 NULL ✅
```

## 关键经验

- **Excel 是 PG COPY 风格 dump,不是设计过的导入模板**:`\N` 标记 NULL,英文列头,27 列含 GORM 系统列
- **不可用现成 Excel 导入服务**:中文列头/状态覆盖/系统列处理全部不适配
- **SQL 含 BEGIN/COMMIT 但不用 `session_replication_role = replica`**:让 PG 正常检查 FK 约束
- **pq driver 的 `db.Exec` 多语句 RowsAffected=0 是已知行为**:用 `SELECT COUNT(*)` 验证
- **删除临时 `main.go` 时注意**:`cmd/<subcommand>/main.go` 是独立子命令不会被 `main redeclared`,但任务结束必须按 CLAUDE.md "Temporary Files Cleanup" 规则清理

## 备注

- 连接信息在 `.env`:`DB_HOST=10.62.10.34` / `DB_NAME=xingran` / `DB_PASSWORD=Cpic1234`
