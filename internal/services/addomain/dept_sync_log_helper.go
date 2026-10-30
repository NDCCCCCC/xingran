package addomain

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// WriteMemberSyncLog 写组成员同步日志并更新映射 last_sync_at。
// 域控审查 P0-1：sys_ou_group_mapping_sync_log 此前从未接线（生产 0 条）；
// P2-6：last_sync_at 统一走 UpdateSyncStatus 单入口，不再裸 Update。
// 放 addomain 包（依赖 OUGroupMappingService），scheduler 的组员同步任务调用。
// 日志写入失败仅 WARN 不阻断同步（可观测性失败不能放大为业务失败）。
func WriteMemberSyncLog(ctx context.Context, svc *OUGroupMappingService, mapping *models.OUGroupMapping, startTime time.Time, added, removed, total int, status, errMsgText string) {
	now := time.Now()
	logRow := &models.OUGroupMappingSyncLog{
		MappingID:      mapping.ID,
		OUdn:           mapping.OUDN,
		ADGroupID:      mapping.ADGroupID,
		SyncType:       "member_sync",
		MembersAdded:   added,
		MembersRemoved: removed,
		TotalMembers:   total,
		Status:         status,
		StartedAt:      startTime,
		CompletedAt:    &now,
		DurationMs:     int(now.Sub(startTime).Milliseconds()),
	}
	if errMsgText != "" {
		logRow.ErrorMsg = &errMsgText
	}
	if err := svc.CreateSyncLog(ctx, logRow); err != nil {
		applogger.Warnf("写组成员同步日志失败(mapping=%s): %v", mapping.ID, err)
	}
	if err := svc.UpdateSyncStatus(ctx, mapping.ID, now); err != nil {
		applogger.Errorf("更新映射同步时间失败(mapping=%s): %v", mapping.ID, err)
	}
}
