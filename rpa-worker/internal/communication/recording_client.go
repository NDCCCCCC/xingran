package communication

// ============================================================================
// Phase W6 / REC-01：录制会话的上行出口（Worker → 后端 HTTP）。
//
// 与 155 liveview 同款处置：Worker 现为纯 HTTP client（APIClient）+ Redis
// Pub/Sub，故上行走 HTTP POST（复用 post 封装，携带 X-Worker-Token），
// **不新建 Worker→后端 WS 长连**（那会引入第二套重连/鉴权/背压生命周期）。
//
// 上行端点（后端侧挂 middleware.WorkerAuth —— Worker 不是用户，无 RBAC 主体）：
//   - POST /rpa/workers/recording/status  会话状态变化
//   - POST /rpa/workers/recording/result  停止后的最终产出（权威结果）
//   - POST /rpa/workers/recording/pick    拾取结果（7 级阶梯）
//
// **画面帧不走这里**：帧复用 155 的 liveview 帧通道
// （POST /rpa/workers/liveview/frame + rpa_liveview_frame WS 消息），
// 坐标系与可丢广播语义共用一套（见 W6-PLAN.md §2.2）。
// ============================================================================

import (
	"context"
	"fmt"

	"github.com/xingran-next/rpa-worker/internal/types"
)

// ReportStatus 上报录制会话状态。
func (c *APIClient) ReportStatus(ctx context.Context, st *types.RecordingUpstreamStatus) error {
	if st == nil || st.SessionID == "" {
		return fmt.Errorf("录制状态上报失败: sessionId 为空")
	}
	if !types.IsValidRecordingStatus(st.Status) {
		return fmt.Errorf("录制状态上报失败: 状态非法 (%q)", st.Status)
	}
	return c.post(ctx, "/rpa/workers/recording/status", st, nil)
}

// ReportResult 上报录制最终产出（后端据此写任务脚本）。
func (c *APIClient) ReportResult(ctx context.Context, res *types.RecordingUpstreamResult) error {
	if res == nil || res.SessionID == "" {
		return fmt.Errorf("录制结果上报失败: sessionId 为空")
	}
	return c.post(ctx, "/rpa/workers/recording/result", res, nil)
}

// ReportPick 上报拾取结果（7 级阶梯）。
func (c *APIClient) ReportPick(ctx context.Context, pick *types.RecordingUpstreamPick) error {
	if pick == nil || pick.SessionID == "" {
		return fmt.Errorf("拾取结果上报失败: sessionId 为空")
	}
	return c.post(ctx, "/rpa/workers/recording/pick", pick, nil)
}
