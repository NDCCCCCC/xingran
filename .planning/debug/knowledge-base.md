# GSD Debug Knowledge Base

Resolved debug sessions. Used by `gsd-debugger` to surface known-pattern hypotheses at the start of new investigations.

---

## workstation-ad-asset-404-recurre — 工位设备 /ad /asset /set-primary-and-save 路由 404 反复出现
- **Date:** 2026-06-15
- **Error patterns:** workstation-device, 404, route missing, handler not registered, awaiting_human_verify
- **Root cause:** internal/api/router.go 工位设备路由组中缺失 POST /:id/ad、POST /:id/asset、POST /:id/set-primary-and-save 三个路由注册；handler 方法已实现但从未挂载到 gin.Engine
- **Fix:** 在 router.go:595 后追加 3 行 POST 注册（GetADDevices/GetAssetDevices/SetPrimaryAndSave）
- **Files changed:** internal/api/router.go
- **Recurrence pattern:** 6/13 session 状态停留在 awaiting_human_verify，文档声称已注册但实际未落地；session 关闭后修复永远丢失
- **Prevention:** 1) awaiting_human_verify→resolved 需强约束后端冒烟测试通过 2) 集成测试扫描 handler 方法 vs router 注册一致性 3) 清理散乱 worktree
---
