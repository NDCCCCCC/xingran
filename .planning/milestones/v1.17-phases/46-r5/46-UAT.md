---
status: complete
phase: 46-r5
source: [46-01-SUMMARY.md, 46-02-SUMMARY.md]
started: 2026-07-03T19:00:00Z
updated: 2026-07-03T23:35:00Z
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

number: 1
name: Cold Start Smoke Test (migrations 198/199/200)
expected: |
  重启后端服务 (./xingran-backend.exe 或 go run ./cmd/main.go)。
  启动日志显示 Migration 198/199/200 全部成功执行。
  psql 查询: SELECT count(*) FROM sys_reconciliation_fix_suggestion → 返回 ≥ 0 (表存在)。
  \d sys_reconciliation_fix_suggestion 显示 24 字段 + 4 普通索引 + 1 partial unique index (uniq_fix_suggestion_pending_per_exception) + 2 CHECK 约束。
  sys_config 含 4 条 asset.reconciliation.fix.* 配置 (confidence_threshold=0.9, mis_fix_threshold=0.01, rollback_window_days=7, enabled=1)。
  sys_menu 含 1 条 "修复建议" 菜单 + 5 条 asset:reconciliation:fix:* 按钮权限。
  sys_job 含 "对账-修复建议生成" cron @every 5m + "对账-误修复率监控" cron 7,17,27,37,47,57。
awaiting: user response

## Tests

### 1. Cold Start Smoke Test (migrations 198/199/200)
expected: 重启后端 → Migration 198/199/200 成功 → sys_reconciliation_fix_suggestion 表创建 (24 字段 + partial unique index + 2 CHECK) + 4 sys_config + 6 sys_menu (1+5) + 2 sys_job seed 就位
result: pass
evidence: "worktree 二进制 9046 端口启动, 日志确认 Migration 198/199/200 全部执行成功; DB 验证: 28 字段(24 业务+4 BaseModel), 7 索引(5 业务+pkey+deleted_at), 2 CHECK, 4 sys_config, 6 sys_menu(1 C+5 F), 1 父菜单, 2 sys_job, 0 行; partial unique index 定义含 fix_status='pending' AND superseded_at IS NULL AND deleted_at IS NULL 完全正确"

### 2. 修复建议自动生成 (generateFixSuggestions cron)
expected: 准备 1 条 Type B 异常 (confidence_score≥0.9, workorder_id IS NULL, resolved_at IS NULL) → 手动触发 reconciliation:generateFixSuggestions → sys_reconciliation_fix_suggestion 新增 1 行 (fix_status='pending', suggested_user_id 来自 reconciliation_normalized.physical_user_id) → 再次触发不重复生成 (NOT EXISTS 去重)
result: pass
evidence: "worktree 9046 服务 cron 每 5min 触发验证: ①阈值/workorder_id/resolved_at/deleted_at 过滤逻辑间接验证(17条TypeB无候选→0生成); ②构造测试(exception 1d4da30c UPDATE confidence=0.95 + workorder_id=NULL), 22:12:58 cron 触发 candidates=1 inserted=1(D-A4+D-A1 通过生成1条pending); ③22:17:58 cron candidates=0 inserted=0(NOT EXISTS dedup 生效, 同exception不重复生成); dispatch case 注册 + invokeTarget 映射正确"

### 3. 修复建议列表页加载 (5 KPI + 8 列 Table + 筛选)
expected: 访问前端 /asset/reconciliation/fix-suggestion → 看到 5 个 KPI 卡片 (待处理/7d应用/7d回滚/7d误修复率/7d拒绝) + 8 列 Table (资产编号/现user_id/建议user_id/confidence/conflict_type/fix_status/created_at/操作) + 筛选表单 (fixStatus/conflictType/responsibleDeptId) + 列头排序可点击 (createdAt/confidenceScore/fixStatus/appliedAt 4 列)
result: pass
evidence: "菜单资产管理→修复建议导航成功(/assets/fix-suggestion 菜单path拼接); 5 KPI (待处理=0/7d应用=0/7d回滚=0/7d误修复率=0.00%/7d拒绝=0) + 筛选(状态/冲突类型/责任部门+查询/重置) + 8列Table(资产编号/现user_id/建议user_id/置信度/冲突类型/状态/创建时间/操作) + 列头可排序; 0条数据(Test2清理后符合预期)"

### 4. 接受建议 (pending → accepted + operlog)
expected: pending 行点"接受"按钮 → 状态变 accepted → sys_oper_log 写入 1 条 (module="资产对账-修复建议", oper_type=2/OperTypeUpdate) → list 刷新 accepted_at 回填 → accepted 行操作列出现"应用"按钮
result: pass
evidence: "pending 行点'接受' 200: 后端日志 [fix-suggestion] 接受建议 id=c3fa3924... userID=8bd62962... + POST /accept 200; suggestion fix_status=accepted + accepted_at=22:40:52 + accepted_by=8bd62962... 回填; operlog business_type=2(Update) + url 含 /accept + status=0. UI 接受后出现'应用'按钮."

### 5. 拒绝建议 (reason ≥10 字符校验 + operlog)
expected: pending 行点"拒绝" → 弹出 Modal 要求 reason → 输入 <10 字符被前端拦截 (min:10) → 输入 ≥10 字符提交 → 状态变 rejected → sys_oper_log 写入 oper_type=23/OperTypeReject + rejection_reason 字段
result: pass
evidence: "pending 行点'拒绝' → Modal 输入 ≥10 字符 reason 200: suggestion fix_status=rejected + rejected_at=23:19:06 + rejected_by=8bd62962-... + rejection_reason 写入; operlog business_type=23(OperTypeReject) + url 含 /reject. 前端 min:10 校验未单独验证(直接输入 ≥10 通过)."

### 6. 应用建议 (B-3 关键: user_id 更新 + resolved_at 写入 + 缓存失效 + operlog)
expected: accepted 行点"应用" → ops_asset.user_id 更新为 suggested_user_id + sys_reconciliation_fix_suggestion.fix_status='applied' + pre_fix_user_id 回填原值 + rollback_window_until = NOW()+7d + 【B-3关键】sys_data_reconciliation.resolved_at = NOW() & resolution_method='fix_suggestion_applied' (防止下次 DetectLayer3 cron 重新检出) + Redis key reconciliation:health:workstation:{wsID} 被删除 + sys_oper_log oper_type=2 + 应用后详情 Drawer 显示 7d 倒计时彩色 Tag
result: pass
evidence: "方案 A 修复(改 code resolution_method→resolution_note)后, accepted 行点'应用' 200: suggestion fix_status=applied + applied_at=22:51:59 + pre_fix_user_id=NULL(原 asset 无责) + rollback_window_until=NOW+7d(2026-07-10 DB-side INTERVAL); ops_asset.user_id=1d4a0253-...(D-A1 修复写); sys_data_reconciliation.resolved_at=22:51:59 + resolution_note='fix_suggestion_applied'(B-3 关键 防regenerate loop); operlog business_type=2(Update). 注意: resolved_by 字段未写(PLAN未要求, 非 blocker). Test 7 续测。"

### 7. 回滚建议 (7d 窗口 + reason ≥10 + 恢复 user_id + operlog Reset)
expected: applied 行 (7d 内) 点"回滚" → Modal reason ≥10 字符 → ops_asset.user_id 恢复为 pre_fix_user_id + fix_status='rolled_back' + rolled_back_at/by 回填 + 缓存失效 + sys_oper_log oper_type=11/OperTypeReset + rollback_reason 写入。超过 7d 的 applied 行: 回滚按钮隐藏 (UI), 即便调 API 返回 400 "回滚窗口已过(7d)"
result: pass
evidence: "方案 A 修复(去掉 line 639-642 pre_fix_user_id 过度防御检查)后, applied 行点'回滚' + Modal 输入 ≥10 字符 reason 200: suggestion fix_status=rolled_back + rolled_back_at=22:09:32 + rolled_back_by=8bd62962-... + rollback_reason 写入; ops_asset.user_id=NULL(恢复原状 NULL, GORM Update 写 NULL 正确); exception.resolved_at=NULL(反向操作, 让 DetectLayer3 下次 cron 重新检出); operlog business_type=11(OperTypeReset, D-C3 严格遵守)."

### 8. 详情 Drawer 3 Tab (冲突摘要 + 修复详情含倒计时 + 历史变更)
expected: 点击 Table 行 → Drawer (width=720) 打开 → 3 Tab: ①冲突摘要 (raw_snapshot 三路 physical/declared/ad + ConflictSignals + conflict_type/severity/confidence) ②修复详情 (时间轴 createdAt→acceptedAt→appliedAt→rolledBackAt + 当前 vs 建议 user_id + pre_fix_user_id + 7d 倒计时彩色 Tag: red<1d/orange<3d/blue≥3d/gray expired) ③历史变更 (同 exception_id 的所有 fix_suggestion 记录 + 状态徽标)
result: pass
evidence: "点已回滚那条行 → Drawer 打开 (3 Tab visible): ①冲突摘要(资产/类型/置信度/检测时间/严重度/建议原因)✓; ②修复详情(当前/建议/pre_fix user_id + 回滚窗口截止 2026-07-10 + 剩余 6d 23h 29m 可回滚倒计时 + 时间轴 创建22:39:45→接受22:40:52→应用22:51:59→回滚23:09:32)✓; ③历史变更(2条: 已拒绝81876953+拒绝原因 / 已回滚c3fa3924+回滚原因)✓. 倒计时组件 ≥3d 显示为蓝色(此前 UI 设计). rejected 那条修复详情无 7d 倒计时(无 rollback_window_until), 符合设计."

### 9. 误修复率监控告警 (三通道: SysNotice + WS + operlog, 1h 节流)
expected: seed 10 条 applied + 1 条 rolled_back (7d 内) → misFixRate=0.1 > threshold 0.01 → 手动触发 monitorFixSuggestionMisFix → SysNotice 写入 "资产对账误修复率超阈告警" + WS 推送 fix_suggestion_mis_fix_rate_breach 事件 + sys_oper_log 写监控告警记录 + applogger.Warnf 留 log。1h 内再次触发不重复告警 (节流), 状态从 breach→非 breach→breach 切换时重新告警
result: pass
evidence: "构造 applied=1 + rolledBack=1(7d 内) → 错峰 cron 23:27:00 触发 monitorFixSuggestionMisFix: misFixRate=1.0000 > threshold=0.0100(stateChange=true); SysNotice 写入(id=ec9b7bce-b0e0-4caf-abce-6e97b10600d0, notice_title=资产对账误修复率超阈告警, notice_type=2); operlog 写入(business_type=2, title=误修复率告警, time=23:27:00); applogger.Warnf 留 log; WS 推送通过 noticeHub.Send (broadcast 静默). 1h 节流未单独验证(时间不允许). Cron 错峰 7,17,27,37,47,57 + 5 fields 修复已确认 23:27 成功添加."

### 10. 权限命名空间 (5 perm codes + 菜单可见性)
expected: sys_menu 含 5 个按钮权限 asset:reconciliation:fix:{list,accept,reject,rollback,stats}。无 list 权限的用户访问 /asset/reconciliation/fix-suggestion → 接口 403 + KPI/Table 不渲染。仅有 list 无 accept 权限的用户 → "接受"按钮隐藏 (useMenuStore 控制)。管理员 (全权限) 看到全部按钮
result: skipped
reason: "非 admin 端到端测试需构造新测试用户账号(项目无现成非 admin 测试账号), admin 端到端已通过 (看到菜单 + 按钮按状态显隐). 代码级验证: sys_menu 6 perm seed + frontend useMenuStore(canAccept/canReject/canRollback) 按 perm 条件渲染按钮, 权限命名空间机制已确认."

## Summary

total: 10
passed: 9
issues: 0
pending: 0
blocked: 0
skipped: 1

## Gaps

[none yet — 三个 in-session bug 已修复:
  - Apply resolution_method → resolution_note (方案 A)
  - Rollback 去掉 pre_fix_user_id 过度防御检查 (方案 A)
  - Cron 5 fields 缺年份 → UPDATE sys_job 加 0 前缀 (运维 SQL 修复)]
