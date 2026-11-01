package portwrite

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// Phase 122 (v1.51 PORTW-07)：批量写 SSE/WS 进度推送 —— service 层。
//
// 选型：复用既有 WebSocket NoticeHub 通道（非新开 SSE），理由：
//   1. 前端已持有 NoticeHub WS 长连接（notice + rpa_progress 两类消息先例），
//      复用 = 零新增连接成本 + 复用既有断线重连逻辑；新开 SSE 意味着第二条
//      长连接 + 第二套重连 + 代理层 buffering 配置，纯增运维面。
//   2. NoticeHub 已支持按 userID 定向广播（BroadcastRPAProgressToUsers 同款），
//      批量写进度天然按操作者定向，无需匿名广播。
//   3. SSE 单向推送优势在此场景无意义（写进度本来就是纯服务端推送）。
//
// 推送粒度：batch_orchestrator 逐端口完成点发事件（每端口 PortResult 就绪即推，
// 非整批结束才推）——用户实时看到 1/50、2/50 推进。
//
// 断线重连语义（契约）：
//   - WS 断线期间的事件不补发（NoticeHub 无离线队列；RPA 进度同款语义）。
//   - 前端重连后以 batchId 调 POST /network/ports/write/audit/list 拉取已落库
//     审计行对账（audit 行是最终真相源，WS 事件只是实时性增强）。
//   - 终态事件 port_write_completed 丢弃时，前端兜底：batch 提交响应本身
//     （同步返回 BatchResult）已含全量结果，进度条仅是 UX 增强。

// 批量写进度事件类型。
const (
	ProgressTypePortWriteProgress  = "port_write_progress"  // 单端口完成（成功/失败/跳过）
	ProgressTypePortWriteCompleted = "port_write_completed" // 批次终态
)

// batchProgressNotifier 批量进度推送抽象（注入 websocket.NoticeHub；测试用 fake）。
type batchProgressNotifier interface {
	BroadcastPortWriteProgressToUsers(userIDs []string, message websocket.PortWriteProgressMessage)
}

// PortWriteNotifier 公开别名：router 层注入 core.NoticeHub 用
// （*websocket.NoticeHub 天然满足本接口）。
type PortWriteNotifier = batchProgressNotifier

// PortWriteProgressMessage = websocket.PortWriteProgressMessage 别名（契约文档就近可读）。
type PortWriteProgressMessage = websocket.PortWriteProgressMessage

// progressPublisher 批量进度发布器（BatchWritePorts 循环内调用）。
type progressPublisher struct {
	notifier    batchProgressNotifier
	operatorIDs []string // 操作者 WS userID（claims.UserID）列表，空 = 不推送
	batchID     string
	deviceID    string
	action      Action
	total       int
}

// newProgressPublisher 构造发布器；username → WS userID 解析失败时降级为不推送
// （进度条是 UX 增强，解析失败不阻塞批量写）。
func (s *portWriteServiceImpl) newProgressPublisher(ctx context.Context, operator string, deviceID string, action Action, total int) *progressPublisher {
	p := &progressPublisher{batchID: uuid.NewString(), deviceID: deviceID, action: action, total: total}
	if s.progressNotifier == nil || operator == "" {
		return p
	}
	var u models.User
	if err := s.db.WithContext(ctx).Select("id").Where("username = ?", operator).First(&u).Error; err != nil {
		applogger.Warnf("[portwrite-batch] 进度推送降级（operator→userID 解析失败）operator=%s err=%v", operator, err)
		return p
	}
	p.notifier = s.progressNotifier
	p.operatorIDs = []string{u.ID}
	return p
}

// emit 单端口事件（逐端口完成点调用）。
func (p *progressPublisher) emit(portID, iface string, index int, status, msg string) {
	if p.notifier == nil || len(p.operatorIDs) == 0 {
		return
	}
	p.notifier.BroadcastPortWriteProgressToUsers(p.operatorIDs, PortWriteProgressMessage{
		Type:          ProgressTypePortWriteProgress,
		BatchID:       p.batchID,
		DeviceID:      p.deviceID,
		Action:        string(p.action),
		PortID:        portID,
		InterfaceName: iface,
		Index:         index,
		Total:         p.total,
		Status:        status,
		Message:       msg,
		Timestamp:     time.Now().Unix(),
	})
}

// emitCompleted 批次终态事件。
func (p *progressPublisher) emitCompleted(succeeded, failed, skipped int, status, msg string) {
	if p.notifier == nil || len(p.operatorIDs) == 0 {
		return
	}
	p.notifier.BroadcastPortWriteProgressToUsers(p.operatorIDs, PortWriteProgressMessage{
		Type:      ProgressTypePortWriteCompleted,
		BatchID:   p.batchID,
		DeviceID:  p.deviceID,
		Action:    string(p.action),
		Index:     p.total,
		Total:     p.total,
		Status:    status,
		Message:   msg,
		Summary:   fmt.Sprintf(`{"succeeded":%d,"failed":%d,"skipped":%d}`, succeeded, failed, skipped),
		Timestamp: time.Now().Unix(),
	})
}
