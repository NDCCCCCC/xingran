# Phase 83 Deferred Items (out-of-scope discoveries)

执行中发现但按 scope boundary 不在本 plan 修复的问题登记。

## 83-03 (2026-08-24)

- **adDomainApi.deleteMapping URL 笔误**: `src/lib/adDomainApi.ts:501` 请求路径为
  `` `/ad-domain/mappings/${id}/delete}` `` (多了一个 `}`)。且后端
  `internal/api/` 中不存在任何 `/ad-domain/mappings` 路由组(部门组映射 legacy
  surface,仅有 ou-group-mappings)。测试按 actual 锁定并注释标记
  (`src/lib/adDomainApi.test.ts` "按 actual 锁定" 用例)。处置建议:确认
  dept-group mapping 前端是否仍有消费方,若无则删除 legacy 函数;若有则修
  URL 笔误并补后端路由。归属:业务代码缺陷,超出本 coverage plan 范围。
