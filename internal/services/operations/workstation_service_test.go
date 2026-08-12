package operations

import (
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestApplyWorkstationOccupancyLink_WithUserID
// 场景: 工位指定 user_id,联动规则应把 status 设为占用(1)
func TestApplyWorkstationOccupancyLink_WithUserID(t *testing.T) {
	userID := "user-uuid-123"
	w := &models.Workstation{UserID: &userID}
	applyWorkstationOccupancyLink(w, models.WorkstationStatusAvailable)
	if w.Status != models.WorkstationStatusOccupied {
		t.Errorf("expected status=Occupied(1), got %d", w.Status)
	}
}

// TestApplyWorkstationOccupancyLink_EmptyUserID
// 场景: user_id 显式置空字符串,联动规则应把 status 设为空闲(0)
func TestApplyWorkstationOccupancyLink_EmptyUserID(t *testing.T) {
	empty := ""
	w := &models.Workstation{UserID: &empty}
	applyWorkstationOccupancyLink(w, models.WorkstationStatusOccupied)
	if w.Status != models.WorkstationStatusAvailable {
		t.Errorf("expected status=Available(0), got %d", w.Status)
	}
}

// TestApplyWorkstationOccupancyLink_NilUserID
// 场景: user_id 字段为 nil,联动规则应把 status 设为空闲(0)
func TestApplyWorkstationOccupancyLink_NilUserID(t *testing.T) {
	w := &models.Workstation{UserID: nil}
	applyWorkstationOccupancyLink(w, models.WorkstationStatusOccupied)
	if w.Status != models.WorkstationStatusAvailable {
		t.Errorf("expected status=Available(0), got %d", w.Status)
	}
}

// TestApplyWorkstationOccupancyLink_PreserveMaintain
// 场景: 当前 status 已经是 Maintain(2),即使 user_id 有值也不应改变 w.Status。
// 这是关键约束:n71 要求 Maintain 语义独立于人员分配。
//
// helper 实现: currentStatus==Maintain 时早 return,不触碰 w.Status。
// 此处用 sentinel 值 (occupier=99, 非法枚举值) 检测 helper 是否"漏改":
// 若 helper 在 Maintain 下错误地覆盖了 w.Status,会得到 0/1;若正确早 return,
// w.Status 维持 99。
func TestApplyWorkstationOccupancyLink_PreserveMaintain(t *testing.T) {
	userID := "user-uuid-123"
	const sentinel = models.WorkstationStatus(99)
	w := &models.Workstation{UserID: &userID, Status: sentinel}
	applyWorkstationOccupancyLink(w, models.WorkstationStatusMaintain)
	if w.Status != sentinel {
		t.Errorf("expected w.Status unchanged (%d) when currentStatus=Maintain, got %d", sentinel, w.Status)
	}
}

// TestApplyWorkstationOccupancyLink_PreserveMaintainOnClear
// 场景: 当前 status 是 Maintain,user_id 被清空, helper 仍应早 return 不改 w.Status
func TestApplyWorkstationOccupancyLink_PreserveMaintainOnClear(t *testing.T) {
	empty := ""
	const sentinel = models.WorkstationStatus(99)
	w := &models.Workstation{UserID: &empty, Status: sentinel}
	applyWorkstationOccupancyLink(w, models.WorkstationStatusMaintain)
	if w.Status != sentinel {
		t.Errorf("expected w.Status unchanged (%d) on user clear when currentStatus=Maintain, got %d", sentinel, w.Status)
	}
}