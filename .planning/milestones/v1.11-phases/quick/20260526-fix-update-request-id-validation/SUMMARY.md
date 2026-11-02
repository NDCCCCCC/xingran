---
status: complete
---

# 20260526-fix-update-request-id-validation — Outcome: DONE（收档）

原问题：`UpdateRequest.ID` 误标 `binding:"required"`，而 ID 实际取自 URL 路径参数，
前端 body 不带 id 时被 binding 拦截返回 400。

收档分诊（2026-10-26）复核发现仅剩最后一处未清理——`internal/models/system/requests/apikey_request.go:19`
的 `UpdateAPIKeyRequest.ID` 仍带 `binding:"required"`。本次已修正：移除该 binding 约束
（handler `apikey_handler.go:159/172` 从 URL 取 id 并回填，body 字段不参与校验），
`go build ./...` 通过，`go test ./internal/api/v1/system/ -run APIKey` 全绿。
该 quick task 至此完全关闭。
