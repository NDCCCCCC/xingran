package vdi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// =====================================================================
// Phase 74-06: audit_service.go + client.go（旧 VDIClient）+ mock_server.go 测试。
// =====================================================================

// seedAuditLog 直接插审计行（绕过 RecordOperation —— sqlite 无法绑定 map 参数，见 SUMMARY quirk）。
func seedAuditLog(t *testing.T, db *gorm.DB, id, vmID, action, result string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_rpa_audit_logs (id, resource_type, resource_id, action, result, operator_id, ip_address, created_at)
		 VALUES (?, 'vdi', ?, ?, ?, 'u1', '1.2.3.4', ?)`,
		id, vmID, action, result, time.Now().Format("2006-01-02 15:04:05"),
	).Error)
}

func TestAuditService_RecordOperation(t *testing.T) {
	db := newVDITestDB(t)
	svc := NewAuditService(db)

	// QUIRK: AuditLog.OldValue/NewValue 是 map[string]interface{}，sqlite 驱动无法绑定
	// （PG 下由 pgx 原生编码为 jsonb）→ RecordOperation 在 sqlite 恒报错。
	err := svc.RecordOperation(context.Background(), &AuditRequest{
		VMID: "vm-1", Operation: OpVMStart, Status: OpStatusSuccess,
		OperatorID: "u1", OperatorName: "alice", OperatorIP: "1.2.3.4",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "数据库操作失败")
}

func TestAuditService_QueryLogs(t *testing.T) {
	db := newVDITestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	seedAuditLog(t, db, "a1", "vm-1", "start", "success")
	seedAuditLog(t, db, "a2", "vm-1", "stop", "failed")
	seedAuditLog(t, db, "a3", "vm-2", "create", "success")

	// 全量
	logs, total, err := svc.QueryLogs(ctx, &AuditQueryRequest{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)

	// VMID 过滤
	logs, total, err = svc.QueryLogs(ctx, &AuditQueryRequest{VMID: "vm-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)

	// 操作类型过滤
	logs, total, err = svc.QueryLogs(ctx, &AuditQueryRequest{VMID: "vm-1", Operation: OpVMStart, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "start", string(logs[0].Action))

	// 时间范围
	early := time.Now().Add(-time.Hour)
	late := time.Now().Add(time.Hour)
	_, total, err = svc.QueryLogs(ctx, &AuditQueryRequest{StartTime: early, EndTime: late, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	// 排序白名单
	asc := true
	logs, _, err = svc.QueryLogs(ctx, &AuditQueryRequest{Page: 1, PageSize: 10, OrderByColumn: "action", IsAsc: &asc})
	require.NoError(t, err)
	assert.Equal(t, "create", string(logs[0].Action))

	// 分页
	logs, total, err = svc.QueryLogs(ctx, &AuditQueryRequest{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 1)
}

func TestAuditService_GetVMOperationsSummary(t *testing.T) {
	db := newVDITestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	seedAuditLog(t, db, "a1", "vm-1", "restart", "success")
	seedAuditLog(t, db, "a2", "vm-1", "restart", "success")
	seedAuditLog(t, db, "a3", "vm-1", "restart", "success")
	seedAuditLog(t, db, "a4", "vm-1", "stop", "failed")
	seedAuditLog(t, db, "a5", "vm-2", "start", "success")

	sum, err := svc.GetVMOperationsSummary(ctx, "vm-1", 7)
	require.NoError(t, err)
	assert.Equal(t, 4, sum.TotalOperations)
	assert.Equal(t, 3, sum.SuccessCount)
	assert.Equal(t, 1, sum.FailureCount)
	require.Len(t, sum.RecentLogs, 4)
	assert.Equal(t, "7 days", sum.Period)
}

// =====================================================================
// 旧版 VDIClient（/api/v1/* 端点）× VDIAPIMock
// =====================================================================

func TestVDIClient_LegacyAgainstMock(t *testing.T) {
	mock := NewVDIAPIMock()
	defer mock.Close()
	client := NewVDIClient(mock.GetServerURL(), 5000)

	// 未认证 → makeRequest 报 token 无效
	_, err := client.ListVMs()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VDI Token 无效或已过期")

	// 认证
	require.NoError(t, client.Authenticate("admin", "any"))
	assert.GreaterOrEqual(t, mock.GetCallCount("auth"), 1)

	// 创建
	vmID, err := client.CreateVM(&CreateVMRequest{VMName: "test-vm", IP: "1.1.1.1", MAC: "AA:BB"})
	require.NoError(t, err)
	assert.NotEmpty(t, vmID)
	assert.GreaterOrEqual(t, mock.GetCallCount("create_vm"), 1)

	// 列表
	vms, err := client.ListVMs()
	require.NoError(t, err)
	require.Len(t, vms, 1)
	assert.Equal(t, "test-vm", vms[0].VMName)

	// 详情
	info, err := client.GetVMInfo(vmID)
	require.NoError(t, err)
	assert.Equal(t, vmID, info.VMID)

	// 详情不存在
	_, err = client.GetVMInfo("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取虚拟机详情失败")

	// 电源操作
	require.NoError(t, client.StartVM(vmID))
	require.NoError(t, client.StopVM(vmID))
	require.NoError(t, client.RestartVM(vmID))
	info, err = client.GetVMInfo(vmID)
	require.NoError(t, err)
	assert.Equal(t, "running", info.Status, "restart 后 running")

	// 重命名 / 绑定 / 解绑
	require.NoError(t, client.RenameVM(vmID, "renamed"))
	require.NoError(t, client.BindUser(vmID, "alice", "pw"))
	require.NoError(t, client.UnbindUser(vmID))
	assert.GreaterOrEqual(t, mock.GetCallCount("rename"), 1)
	assert.GreaterOrEqual(t, mock.GetCallCount("bind_user"), 1)
	assert.GreaterOrEqual(t, mock.GetCallCount("unbind_user"), 1)

	// 操作不存在的 VM
	err = client.StartVM("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "启动虚拟机失败")
	err = client.StopVM("missing")
	require.Error(t, err)
	err = client.RestartVM("missing")
	require.Error(t, err)
	err = client.DeleteVM("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "删除虚拟机失败")
	err = client.RenameVM("missing", "x")
	require.Error(t, err)
	err = client.BindUser("missing", "u", "p")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "绑定用户失败")
	err = client.UnbindUser("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解绑用户失败")

	// 删除存在的 VM
	require.NoError(t, client.DeleteVM(vmID))
	vms, err = client.ListVMs()
	require.NoError(t, err)
	assert.Empty(t, vms)

	// config_ip 端点（旧 client 无对应方法，直接打 HTTP 覆盖 mock handler）
	cpResp, err := http.Post(mock.GetServerURL()+"/api/v1/vm/config_ip", "application/json",
		strings.NewReader(`{"vM_ID":"none"}`))
	require.NoError(t, err)
	defer cpResp.Body.Close()
	assert.Equal(t, http.StatusOK, cpResp.StatusCode)
}

func TestVDIClient_AuthFailure(t *testing.T) {
	mock := NewVDIAPIMock()
	defer mock.Close()
	client := NewVDIClient(mock.GetServerURL(), 5000)

	// mock 接受任意凭证，这里用错误 URL 测认证请求失败
	bad := NewVDIClient("http://127.0.0.1:1", 100)
	err := bad.Authenticate("admin", "pw")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VDI 认证请求失败")

	// 认证响应非 JSON → 解析失败（复用 mock 的 auth 端点但打到不存在的 path 不行，
	// 直接用 error_code 非零无法触发；此处覆盖 ensureAuth 过期分支）
	client.authExpiry = time.Now().Add(-time.Hour)
	_, err = client.ListVMs()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VDI Token 无效或已过期")
}
