# Roadmap

## Project: 信息点导入设备/端口关联

**Created**: 2026-04-16
**Total Phases**: 1
**Total Requirements**: 5 (5 v1.1, 7 v1.0 completed)

---

## Phase 3: 信息点导入设备端口配置

**Goal**: 在 Excel 导入配置中添加设备和端口列，实现名称→ID 自动匹配

**Requirements**: IMPORT-05, IMPORT-06, IMPORT-07, VAL-03, VAL-04
**Plans:** 1 plan
**UI hint**: no

Plans:
- [x] 03-01-PLAN.md — Add deviceName and portName ExcelColumn entries with Reference resolution + template example values

**Changes**:
- `internal/services/operations/excel_config.go` — infoPoint 配置添加设备和端口列
- `internal/services/operations/excel_service.go` — 模板示例值添加 deviceName/portName cases

**Success Criteria**:
1. 导入含设备名和端口名的 Excel 文件后，`ops_info_points` 表的 `device_id` 和 `port_id` 字段正确存储 UUID
2. 导入不含设备/端口信息的 Excel 文件后，`device_id` 和 `port_id` 为空，不报错
3. 设备名或端口名在数据库中不存在时，对应字段留空，导入不中断
4. 信息点列表正确显示设备名称和端口名称（已有 JOIN，验证不退化）

---

## Requirement Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| IMPORT-01 | Phase 1 | completed |
| IMPORT-02 | Phase 1 | completed |
| IMPORT-03 | Phase 1 | completed |
| IMPORT-04 | Phase 2 | completed |
| TMPL-01 | Phase 2 | completed |
| VAL-01 | Phase 2 | completed |
| VAL-02 | Phase 2 | completed |
| IMPORT-05 | Phase 3 | completed |
| IMPORT-06 | Phase 3 | completed |
| IMPORT-07 | Phase 3 | completed |
| VAL-03 | Phase 3 | completed |
| VAL-04 | Phase 3 | completed |

---

## Coverage Check

- [x] All v1.1 requirements mapped to phases
- [x] No orphan requirements
- [x] Success criteria are observable behaviors
