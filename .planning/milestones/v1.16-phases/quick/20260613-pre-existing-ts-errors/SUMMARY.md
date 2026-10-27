---
status: complete
---

# 20260613-pre-existing-ts-errors — Outcome: DONE（收档）

原计划清理 21 个预先存在的 TypeScript 错误（13 文件）。该批错误已被后续工作陆续消化，
里程碑收档分诊（2026-10-26）复测 `npx tsc -b --noEmit` 退出码 0、零错误输出，
即 type-check 基线已清零，原目标达成。

注：lint 侧仍有 83 errors 存量豁免（与本任务的 tsc 口径不同），归 v1.23 存量清偿批次跟踪。
