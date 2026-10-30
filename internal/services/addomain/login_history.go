package addomain

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ADLoginHistoryWriter AD 设备登录历史写入器(quick 261016-ad-login-history)
//
// 职责:AD 同步过程中收集 lastLogon 变化事件,同步写库(batchCreate/batchUpdate)
// 成功后一次性批量写入 sys_ad_login_history。
// 约束(PLAN 风险 4/5/8):
//   - 批量收集 + 一次性批量 INSERT,禁止逐条 INSERT;
//   - flush 失败仅记日志,不阻断主同步;
//   - 非 PostgreSQL(如 sqlite 内存单测/dev)整体跳过。
type ADLoginHistoryWriter struct {
	db *gorm.DB
}

// NewADLoginHistoryWriter 创建登录历史写入器
func NewADLoginHistoryWriter(db *gorm.DB) *ADLoginHistoryWriter {
	return &ADLoginHistoryWriter{db: db}
}

// isPostgres 方言守卫:分区表/sys_ad_login_history 仅存在于 PG;
// sqlite 下写入整体跳过,保证内存单测不因缺表报错。
func (w *ADLoginHistoryWriter) isPostgres() bool {
	return w.db != nil && w.db.Dialector.Name() == "postgres"
}

// detectLoginEvent 纯函数:对比新旧 lastLogon 判定是否产生登录事件。
//
// 三分支决策(PLAN 锁定 D1 / 风险 3):
//   - 双 nil                    → 不记录(设备从未登录)
//   - 旧 nil,新非 nil           → initial(设备首次发现且 lastLogon 非空)
//   - 旧非 nil,新 > 旧          → login(检测到新登录)
//   - 相等(无变化)或新 < 旧      → 不记录(AD 时钟回拨 / 多 DC lastLogon 回退)
func detectLoginEvent(oldLL, newLL *time.Time) (models.ADLoginEventType, bool) {
	switch {
	case oldLL == nil && newLL == nil:
		return "", false
	case oldLL == nil && newLL != nil:
		return models.ADLoginEventInitial, true
	case oldLL != nil && newLL != nil && newLL.After(*oldLL):
		return models.ADLoginEventLogin, true
	default:
		return "", false
	}
}

// pendingLoginEvent 单条待写入的登录事件(同步过程内存收集,批量 flush)
type pendingLoginEvent struct {
	computerKey  string                 // toCreate 路径用: configID + "/" + computerName(batchCreate 回填 ID 后解析)
	computerID   string                 // toUpdate 路径直接带 ID
	configID     string                 // 所属AD配置(快照)
	computerName string                 // 计算机名快照
	eventType    models.ADLoginEventType // initial / login
	loginTime    time.Time              // 登录时间(AD lastLogon 值)
	loginUser    string                 // 登录用户(description parts[1])
	ipAddress    string                 // IP 地址(description parts[2])
	detectedAt   time.Time              // 检测时间(同步时刻)
}

// Flush 将收集到的登录事件批量写入 sys_ad_login_history。
//
// computerIDByKey:调用方在 batchCreate 成功后从最终切片(此时 ID 已被
// BeforeCreate hook 回填)构建的 computerKey→ID 映射;解析不到的 initial
// 事件跳过并 Warn(PLAN 风险 6)。
// 任何错误仅记日志,不向上返回(PLAN 风险 5:历史插入失败不阻断主同步)。
func (w *ADLoginHistoryWriter) Flush(ctx context.Context, events []pendingLoginEvent, computerIDByKey map[string]string) {
	if len(events) == 0 || !w.isPostgres() {
		return
	}
	flushLoginEvents(ctx, w.db, events, computerIDByKey)
}

// flushLoginEvents 批量写入实现(CreateInBatches 500/批,错误仅 log)
func flushLoginEvents(ctx context.Context, db *gorm.DB, events []pendingLoginEvent, computerIDByKey map[string]string) {
	rows := make([]models.ADLoginHistory, 0, len(events))
	skipped := 0
	for _, ev := range events {
		id := ev.computerID
		if id == "" && ev.computerKey != "" {
			id = computerIDByKey[ev.computerKey]
			if id == "" {
				applogger.Warnf("[AD登录历史] 解析不到 computer_id,跳过事件: key=%s, computer=%s",
					ev.computerKey, ev.computerName)
				skipped++
				continue
			}
		}
		rows = append(rows, models.ADLoginHistory{
			ComputerID:           id,
			ComputerNameSnapshot: ev.computerName,
			ADConfigID:           ev.configID,
			LoginTime:            ev.loginTime,
			DetectedAt:           ev.detectedAt,
			LoginUser:            ev.loginUser,
			IPAddress:            ev.ipAddress,
			EventType:            ev.eventType,
		})
	}

	if len(rows) == 0 {
		return
	}

	if err := db.WithContext(ctx).CreateInBatches(rows, 500).Error; err != nil {
		applogger.Errorf("[AD登录历史] 批量写入失败(不阻断同步): %v", err)
		return
	}
	applogger.Infof("[AD登录历史] 写入 %d 条登录事件(跳过 %d 条未解析ID)", len(rows), skipped)
}
