---
slug: workstation-update-createdat-zeroed
status: resolved
trigger: "工位编辑保存后 created_at 变成 0001-01-01 08:06:26"
created: 2026-06-18T00:00:00+08:00
updated: 2026-06-18T16:36:00+08:00
---

## Current Focus

hypothesis: workstationService.Update 使用 GORM Save() 全量更新,handler 把请求体反序列化到空 models.Workstation(零值 CreatedAt),Save 用零值覆盖 created_at。
test: 1) 读 handler/service/model 源码确认证据链;2) 写 sqlite 单测重现 Save 清零行为;3) 修复后断言 created_at 未被清零。
expecting: 修复前测试失败(created_at=0001-01-01),修复后通过。
next_action: 实施 workstation_service.go 的 Update 修复 + 新增 updatedAt 排序列 + 写回归测试。

reasoning_checkpoint:
  hypothesis: "工位 Update 把请求体反序列化到空 models.Workstation,CreatedAt 是非指针 time.Time 零值,GORM Save 全量写库覆盖 created_at。修复方法:Save 前回填 CreatedAt/CreatedBy。"
  confirming_evidence:
    - "internal/api/v1/operations/workstation_handler.go:117-123 Update 用 `var workstation models.Workstation` 接收请求体,只设置 workstation.ID,未触碰 CreatedAt"
    - "internal/services/operations/workstation_service.go:87-89 Update 仅一行 `return s.db.WithContext(ctx).Save(workstation).Error`"
    - "internal/models/base.go:13 BaseModel.CreatedAt 是非指针 time.Time,零值即 0001-01-01"
    - "internal/models/workstation.go:37 Workstation 嵌入 BaseModel"
    - "前端 index.tsx:245-261 handleWorkstationModalOk 只传表单字段(submitValues),不含 createdAt/updatedAt"
  falsification_test: "若假设成立,sqlite 测试调用 Update(零 CreatedAt) 后库中 created_at 应=0001-01-01;若不成立,created_at 保留原值(说明 Save 不全量写或零值不生效)"
  fix_rationale: "Save 前先 First 查出现有记录,把 existing.CreatedAt/CreatedBy 回填到入参,这样 Save 写入的 created_at 仍是原值。这是 user_service.go:136-216 已有的成熟模式。"
  blind_spots: "未测试 PostgreSQL(只用 sqlite 模拟);未测试并发更新;Version 字段未单独处理(Workstation 的 Version 也是非指针 int,Save 会覆盖,但本次 scope 只针对 created_at)"

## Symptoms

expected: 工位编辑保存后,created_at 保持原值不变,updated_at 刷新为当前时间。
actual: created_at 变成 0001-01-01 08:06:26(=Go time.Time{} 零值 0001-01-01 00:00:00 UTC 在 Asia/Shanghai LMT 偏移 +8:06 的显示)。
errors: 无后端报错,GORM Save 成功但用零值覆盖了 created_at。
reproduction: 任何工位的"编辑 → 保存"操作。
started: 一直存在(GORM Save + 零值反序列化的固有问题)。

## Evidence

- timestamp: 2026-06-18T00:00:00+08:00
  checked: internal/api/v1/operations/workstation_handler.go:116-130
  found: Update handler `var workstation models.Workstation` + `workstation.ID = id` + 调用 service.Update(&workstation)。请求体不含 createdAt,所以 workstation.BaseModel.CreatedAt 是零值。
  implication: 入参的 CreatedAt=0001-01-01,会传给 Save。

- timestamp: 2026-06-18T00:00:00+08:00
  checked: internal/services/operations/workstation_service.go:87-89
  found: Update 仅 `return s.db.WithContext(ctx).Save(workstation).Error`。无 First 查 existing,无字段保护。
  implication: GORM Save 对所有字段(含零值)生成 UPDATE,GORM 自动 UpdatedAt 钩子会刷新 updated_at,但 created_at 会被零值覆盖。

- timestamp: 2026-06-18T00:00:00+08:00
  checked: internal/models/base.go:11-19
  found: BaseModel.CreatedAt time.Time(非指针),零值=0001-01-01。
  implication: 零值 time.Time 经 Save 写库即清零 created_at,与症状精确吻合。

- timestamp: 2026-06-18T00:00:00+08:00
  checked: internal/models/workstation.go:36-37
  found: Workstation struct 嵌入 BaseModel(匿名),继承 CreatedAt 字段。
  implication: 完成证据链:handler→service→model 字段类型=零值覆盖。

- timestamp: 2026-06-18T00:00:00+08:00
  checked: internal/services/system/user_service.go:136-216(参考实现)
  found: user Update 先 `First(&user, "id = ?", req.ID)` 查出 existing,修改字段后 Save。user 模型字段直接用 existing 的值(不会变零),所以无此 bug。
  implication: 参考模式成立,workstation Update 需复制此保护(至少回填 CreatedAt/CreatedBy)。

## Resolution

root_cause: workstationService.Update 直接对零值对象调用 GORM Save,全量写库覆盖 created_at 为 0001-01-01。
fix: |
  1. (后端) workstation_service.go Update:Save 前 First 查 existing,回填 CreatedAt/CreatedBy,再 Save。
  2. (前端) 新增"修改时间"列(columns.tsx 加 updatedAt 列 + index.tsx 加 sorterMeta)。
  3. (后端) workstationAllowedSortFields 追加 "updatedAt" → "sys_workstation.updated_at"。
verification: |
  - RED/GREEN 双向验证(关键):
    * 未修复时 stash workstation_service.go → 测试 FAIL "created_at was zeroed by Update() — bug returned"
    * 恢复修复 → 测试 PASS "created_at preserved=2024-06-01 10:30:00, updated_at=2026-06-18 16:34:52"
  - go build ./... 全部 OK
  - go test ./internal/services/operations/ -run TestWorkstationUpdate 两个测试全 PASS
  - npx tsc --noEmit -p tsconfig.app.json 0 errors(未引入新错误)
  - 边界:TestWorkstationUpdate_RecordNotFound 确认记录不存在时返回 gorm.ErrRecordNotFound(不再悄悄 Save 新记录)
files_changed:
  - internal/services/operations/workstation_service.go(Update 修复 + updatedAt 排序字段)
  - internal/services/operations/workstation_update_createdat_test.go(新增回归测试)
  - xingran-react-frontend/src/pages/operations/workstations/columns.tsx(新增 updatedAt 列)
  - xingran-react-frontend/src/pages/operations/workstations/index.tsx(新增 updatedAt sorterMeta)

## Scope Note

按 orchestrator 指令,本次仅修工位。floor/server_room 等 ops 模块若用同样的 handler→零值对象→Save 模式,也会有相同 bug(后续若报告可同样套用此 fix 模式)。operlog.Record(Phase 34)未动。
