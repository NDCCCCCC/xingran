package vdi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// Phase 74-06: vm_service_impl.go + vm_data_scope_filter.go 测试。
// =====================================================================

// fakeVMVDIClient VDIClientExtended 假实现：按方法注入结果/错误。
type fakeVMVDIClient struct {
	authErr        error
	authToken      string
	getVM          *VDIVMDetail
	getVMErr       error
	listRes        []VDIResource
	listResErr     error
	groups         []VDIResourceGroup
	groupsErr      error
	servers        []VDIVMResource
	serversTotal   int
	serversErr     error
	operateErr     error
	deleteErr      error
	bindDetail     *VDIVMDetail
	bindErr        error
	createResp     *CreateServerResponse
	createErr      error
	runPositions   []RunPosition
	storages       []Storage
	networks       []Network
	operateCalls   [][]string
	deleteCalls    [][]string
	listVMsSummary []VDIVMSummary
	listVMsErr     error
}

func (f *fakeVMVDIClient) Authenticate(ctx context.Context) (string, error) {
	if f.authErr != nil {
		return "", f.authErr
	}
	if f.authToken != "" {
		return f.authToken, nil
	}
	return "fake-token", nil
}
func (f *fakeVMVDIClient) GetVM(ctx context.Context, vmID string) (*VDIVMDetail, error) {
	return f.getVM, f.getVMErr
}
func (f *fakeVMVDIClient) ListVMs(ctx context.Context, resourceID string) ([]VDIVMSummary, error) {
	return f.listVMsSummary, f.listVMsErr
}
func (f *fakeVMVDIClient) GetUserVMs(ctx context.Context, userID string) ([]VDIVMSummary, error) {
	return f.listVMsSummary, f.listVMsErr
}
func (f *fakeVMVDIClient) OperateVM(ctx context.Context, vmIDs []string, action string) error {
	f.operateCalls = append(f.operateCalls, vmIDs)
	return f.operateErr
}
func (f *fakeVMVDIClient) DeleteVM(ctx context.Context, vmIDs []string) error {
	f.deleteCalls = append(f.deleteCalls, vmIDs)
	return f.deleteErr
}
func (f *fakeVMVDIClient) BindUser(ctx context.Context, vmID, userID string) (*VDIVMDetail, error) {
	return f.bindDetail, f.bindErr
}
func (f *fakeVMVDIClient) GetAvailableUsers(ctx context.Context, vmID string) ([]VDIUser, error) {
	return []VDIUser{}, nil
}
func (f *fakeVMVDIClient) ListResources(ctx context.Context, groupID string) ([]VDIResource, error) {
	return f.listRes, f.listResErr
}
func (f *fakeVMVDIClient) ListResourceGroups(ctx context.Context) ([]VDIResourceGroup, error) {
	return f.groups, f.groupsErr
}
func (f *fakeVMVDIClient) ListResourceServers(ctx context.Context, resourceID string, page, pageSize int) ([]VDIVMResource, int, error) {
	return f.servers, f.serversTotal, f.serversErr
}
func (f *fakeVMVDIClient) GetVTPPlatforms(ctx context.Context) ([]VDIPlatform, error) {
	return nil, nil
}
func (f *fakeVMVDIClient) GetRunPositions(ctx context.Context, vtpID int) ([]RunPosition, error) {
	return f.runPositions, nil
}
func (f *fakeVMVDIClient) GetStorages(ctx context.Context, vtpID int) ([]Storage, error) {
	return f.storages, nil
}
func (f *fakeVMVDIClient) GetNetworks(ctx context.Context, vtpID int) ([]Network, error) {
	return f.networks, nil
}
func (f *fakeVMVDIClient) CreateServer(ctx context.Context, req CreateServerRequest) (*CreateServerResponse, error) {
	return f.createResp, f.createErr
}

// insertVM 插入一条本地虚拟机记录。
func insertVM(t *testing.T, db *gorm.DB, id, vmID, name, serverID string) {
	t.Helper()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(
		`INSERT INTO sys_vdi_vm (id, vm_id, name, vdi_server_id, power_state, ip_address, resource_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'stopped', '1.2.3.4', '1', ?, ?)`,
		id, vmID, name, serverID, now, now,
	).Error)
}

func TestVMService_ListResourceGroups_LocalDB(t *testing.T) {
	db := newVDITestDB(t)
	svc := NewVMServiceWithDynamicClient(db)
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_resource_group
		(id, resource_group_id, name, vdi_server_id, type, status, created_at, updated_at) VALUES
		('r1','g1','grp-1','srv1','独享',0,?,?),
		('r2','g2','grp-2','srv1','池',1,?,?),
		('r3','g3','grp-3','srv2','独享',0,?,?)`, now, now, now, now, now, now).Error)

	ctx := context.Background()

	// status 过滤 + server 过滤
	dtos, err := svc.ListResourceGroups(ctx, "srv1")
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	assert.Equal(t, "g1", dtos[0].ResourceGroupID)
	assert.Equal(t, "独享", dtos[0].Type)

	// 不传 server → 全部启用的
	dtos, err = svc.ListResourceGroups(ctx, "")
	require.NoError(t, err)
	assert.Len(t, dtos, 2)
}

func TestVMService_ListResources(t *testing.T) {
	db := newVDITestDB(t)
	ctx := context.Background()

	svc := NewVMService(db, &fakeVMVDIClient{})
	_, err := svc.ListResources(ctx, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "group_id is required")

	fail := NewVMService(db, &fakeVMVDIClient{listResErr: errors.New("boom")})
	_, err = fail.ListResources(ctx, "", "g1")
	require.Error(t, err)

	ok := NewVMService(db, &fakeVMVDIClient{listRes: []VDIResource{{ID: 1, Name: "res-1", Note: "n", GrpID: 5}}})
	dtos, err := ok.ListResources(ctx, "", "g1")
	require.NoError(t, err)
	require.Len(t, dtos, 1)
	assert.Equal(t, 5, dtos[0].GrpID)
}

func TestVMService_CreateVM(t *testing.T) {
	db := newVDITestDB(t)
	ctx := context.Background()

	// 服务器不存在/停用（不触发 HTTP）
	svc := NewVMService(db, &fakeVMVDIClient{})
	_, err := svc.CreateVM(ctx, &CreateVMServiceRequest{VdiServerID: "missing", ResourceID: "1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VDI server not found or disabled")
	seedVDIServer(t, db, "srv1", "s", "http://x", 1)
	_, err = svc.CreateVM(ctx, &CreateVMServiceRequest{VdiServerID: "srv1", ResourceID: "1"})
	require.Error(t, err)

	// 注：CreateVM 内部用 NewVDIClientFromDB 自建真实 client（不消费注入的 fake）
	// 成功：本地落库
	okFake := newFakeVDIExtServer(t, false)
	seedVDIServer(t, db, "srv-ok", "s2", okFake.URL, 0)
	dto, err := svc.CreateVM(ctx, &CreateVMServiceRequest{
		VdiServerID: "srv-ok", ResourceID: "7", Name: "ignored", CPUNumber: 2, CPUCore: 4, Memory: 4096, Disk: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, "9001", dto.VMID)
	assert.Equal(t, "", dto.Name, "名称不从用户输入取")
	assert.Equal(t, "stopped", dto.PowerState)
	var count int
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_vdi_vm WHERE vm_id = '9001'`).Scan(&count).Error)
	assert.Equal(t, 1, count)

	// VDI API 返回空 server_id
	emptyFake := newFakeVDIExtServer(t, false)
	emptyFake.mu.Lock()
	emptyFake.emptyCreateIDs = true
	emptyFake.mu.Unlock()
	seedVDIServer(t, db, "srv-empty", "s3", emptyFake.URL, 0)
	_, err = svc.CreateVM(ctx, &CreateVMServiceRequest{VdiServerID: "srv-empty", ResourceID: "7"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no VM IDs")

	// 认证失败（500 → 无重试直接报错）
	failFake := newFakeVDIExtServer(t, false)
	failFake.mu.Lock()
	failFake.authFail = true
	failFake.mu.Unlock()
	seedVDIServer(t, db, "srv-fail", "s4", failFake.URL, 0)
	_, err = svc.CreateVM(ctx, &CreateVMServiceRequest{VdiServerID: "srv-fail", ResourceID: "7"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create VM via VDI API")
}

func TestVMService_GetAndUpdateVM(t *testing.T) {
	db := newVDITestDB(t)
	svc := NewVMService(db, &fakeVMVDIClient{})
	ctx := context.Background()

	insertVM(t, db, "vm1", "v1", "alpha", "srv1")

	dto, err := svc.GetVM(ctx, "vm1")
	require.NoError(t, err)
	assert.Equal(t, "v1", dto.VMID)
	assert.Equal(t, "alpha", dto.Name)

	_, err = svc.GetVM(ctx, "missing")
	require.Error(t, err)

	ip := "9.9.9.9"
	mac := "AA:AA"
	require.NoError(t, svc.UpdateVM(ctx, "vm1", &UpdateVMRequest{IPAddress: &ip, MACAddress: &mac}))
	dto, err = svc.GetVM(ctx, "vm1")
	require.NoError(t, err)
	assert.Equal(t, ip, dto.IPAddress)
	assert.Equal(t, mac, dto.MACAddress)

	require.Error(t, svc.UpdateVM(ctx, "missing", &UpdateVMRequest{IPAddress: &ip}))
}

func TestVMService_ListVMs(t *testing.T) {
	db := newVDITestDB(t)
	ctx := context.Background()

	// 无启用的 VDI 服务器 → 空结果
	svc := NewVMService(db, &fakeVMVDIClient{})
	page, err := svc.ListVMs(ctx, &ListVMRequest{}, "", 0)
	require.NoError(t, err)
	assert.Zero(t, page.Total)
	assert.Empty(t, page.List)

	seedVDIServer(t, db, "srv1", "s", "http://x", 0)
	insertVM(t, db, "vm1", "v1", "alpha", "srv1")
	insertVM(t, db, "vm2", "v2", "beta", "srv1")
	insertVM(t, db, "vm3", "v3", "gamma-other", "srv2")

	// 过滤：name / server / powerState；分页默认
	page, err = svc.ListVMs(ctx, &ListVMRequest{Name: "alp"}, "u", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Equal(t, "alpha", page.List[0].Name)

	page, err = svc.ListVMs(ctx, &ListVMRequest{VdiServerID: "srv1"}, "u", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)

	page, err = svc.ListVMs(ctx, &ListVMRequest{PowerState: "stopped"}, "u", 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total)

	// 排序白名单
	asc := true
	page, err = svc.ListVMs(ctx, &ListVMRequest{OrderByColumn: "name", IsAsc: &asc}, "", 0)
	require.NoError(t, err)
	assert.Equal(t, "alpha", page.List[0].Name)

	// DataScopeAll 不过滤
	page, err = svc.ListVMs(ctx, &ListVMRequest{}, "11111111-1111-1111-1111-111111111111", models.DataScopeAll)
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total)

	// 非法 userID → 1=0
	page, err = svc.ListVMs(ctx, &ListVMRequest{},"not-a-uuid", models.DataScopeSelf)
	require.NoError(t, err)
	assert.Zero(t, page.Total)

	// DataScopeSelf: 只看绑定自己的
	require.NoError(t, db.Exec(`UPDATE sys_vdi_vm SET bound_user_id = '11111111-1111-1111-1111-111111111111' WHERE id = 'vm1'`).Error)
	page, err = svc.ListVMs(ctx, &ListVMRequest{}, "11111111-1111-1111-1111-111111111111", models.DataScopeSelf)
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Equal(t, "vm1", page.List[0].ID)

	// 非 DataScopeAll → 隐藏未绑定 VM
	page, err = svc.ListVMs(ctx, &ListVMRequest{}, "11111111-1111-1111-1111-111111111111", models.DataScopeSelf)
	require.NoError(t, err)
	for _, d := range page.List {
		require.NotNil(t, d.BoundUserID)
	}
}

func TestVMService_DeleteVM(t *testing.T) {
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s", "http://x", 0)
	insertVM(t, db, "vm1", "v1", "alpha", "srv1")
	insertVM(t, db, "vm2", "v2", "beta", "srv1")
	ctx := context.Background()

	// 本地查无记录
	ok := NewVMService(db, &fakeVMVDIClient{})
	err := ok.DeleteVM(ctx, []string{"missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no VMs found")

	// VDI API 3001（VDI 上已不存在）→ 仍清理本地
	e3001 := NewVMService(db, &fakeVMVDIClient{deleteErr: &VDIError{Code: 3001, Message: "vm not found"}})
	require.NoError(t, e3001.DeleteVM(ctx, []string{"vm1"}))
	var cnt int
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_vdi_vm WHERE id = 'vm1' AND deleted_at IS NULL`).Scan(&cnt).Error)
	assert.Zero(t, cnt)

	// 其他 VDI 错误 → 保留本地
	other := NewVMService(db, &fakeVMVDIClient{deleteErr: &VDIError{Code: 500, Message: "server error"}})
	err = other.DeleteVM(ctx, []string{"vm2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete VMs from VDI")
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_vdi_vm WHERE id = 'vm2' AND deleted_at IS NULL`).Scan(&cnt).Error)
	assert.Equal(t, 1, cnt)

	// 成功删除
	success := NewVMService(db, &fakeVMVDIClient{})
	require.NoError(t, success.DeleteVM(ctx, []string{"vm2"}))
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_vdi_vm WHERE deleted_at IS NULL`).Scan(&cnt).Error)
	assert.Zero(t, cnt)
}

func TestVMService_OperateVM(t *testing.T) {
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s", "http://x", 0)
	insertVM(t, db, "vm1", "v1", "alpha", "srv1")
	ctx := context.Background()

	fake := &fakeVMVDIClient{}
	svc := NewVMService(db, fake)

	// 本地查无记录
	err := svc.OperateVM(ctx, &VMOperateRequest{VMIDs: []string{"missing"}, Action: VMPowerOn})
	require.Error(t, err)

	// 开机 → 本地状态 running
	require.NoError(t, svc.OperateVM(ctx, &VMOperateRequest{VMIDs: []string{"vm1"}, Action: VMPowerOn}))
	require.Len(t, fake.operateCalls, 1)
	assert.Equal(t, []string{"v1"}, fake.operateCalls[0])
	var state string
	require.NoError(t, db.Raw(`SELECT power_state FROM sys_vdi_vm WHERE id = 'vm1'`).Scan(&state).Error)
	assert.Equal(t, "running", state)

	// 关机 → stopped
	require.NoError(t, svc.OperateVM(ctx, &VMOperateRequest{VMIDs: []string{"vm1"}, Action: VMPowerOff}))
	require.NoError(t, db.Raw(`SELECT power_state FROM sys_vdi_vm WHERE id = 'vm1'`).Scan(&state).Error)
	assert.Equal(t, "stopped", state)

	// 挂起 → suspended
	require.NoError(t, svc.OperateVM(ctx, &VMOperateRequest{VMIDs: []string{"vm1"}, Action: VMPowerSuspend}))
	require.NoError(t, db.Raw(`SELECT power_state FROM sys_vdi_vm WHERE id = 'vm1'`).Scan(&state).Error)
	assert.Equal(t, "suspended", state)

	// 未知 action → API 层不报错（fake），本地状态不更新
	require.NoError(t, svc.OperateVM(ctx, &VMOperateRequest{VMIDs: []string{"vm1"}, Action: VMPowerAction("bogus")}))
	require.NoError(t, db.Raw(`SELECT power_state FROM sys_vdi_vm WHERE id = 'vm1'`).Scan(&state).Error)
	assert.Equal(t, "suspended", state)

	// API 错误
	fail := NewVMService(db, &fakeVMVDIClient{operateErr: errors.New("boom")})
	err = fail.OperateVM(ctx, &VMOperateRequest{VMIDs: []string{"vm1"}, Action: VMPowerRestart})
	require.Error(t, err)
}

func TestVMService_BindUnbindUser(t *testing.T) {
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s", "http://x", 0)
	insertVM(t, db, "vm1", "v1", "alpha", "srv1")
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, nickname, dept_id, created_at, updated_at)
		VALUES ('u1','alice','爱丽丝','d1',?,?)`, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, created_at, updated_at)
		VALUES ('u2','bob',?,?)`, now, now).Error)
	ctx := context.Background()

	svc := NewVMService(db, &fakeVMVDIClient{})

	// VM 不存在
	err := svc.BindUser(ctx, "missing", &BindUserServiceRequest{Username: "alice"})
	require.Error(t, err)

	// 系统用户不存在
	err = svc.BindUser(ctx, "vm1", &BindUserServiceRequest{Username: "ghost"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "system user not found")

	// 有昵称 → "nickname (username)"
	require.NoError(t, svc.BindUser(ctx, "vm1", &BindUserServiceRequest{Username: "alice"}))
	var boundID, boundName string
	require.NoError(t, db.Raw(`SELECT bound_user_id, bound_user_name FROM sys_vdi_vm WHERE id = 'vm1'`).Row().Scan(&boundID, &boundName))
	assert.Equal(t, "u1", boundID)
	assert.Equal(t, "爱丽丝 (alice)", boundName)

	// 无昵称 → 纯用户名
	require.NoError(t, svc.BindUser(ctx, "vm1", &BindUserServiceRequest{Username: "bob"}))
	require.NoError(t, db.Raw(`SELECT bound_user_name FROM sys_vdi_vm WHERE id = 'vm1'`).Scan(&boundName).Error)
	assert.Equal(t, "bob", boundName)

	// 解绑 → NULL
	require.NoError(t, svc.UnbindUser(ctx, "vm1"))
	var boundIDPtr *string
	require.NoError(t, db.Raw(`SELECT bound_user_id FROM sys_vdi_vm WHERE id = 'vm1'`).Scan(&boundIDPtr).Error)
	assert.Nil(t, boundIDPtr)

	// 解绑不存在的 VM
	require.Error(t, svc.UnbindUser(ctx, "missing"))
}

func TestVMService_SyncVMFromVDI(t *testing.T) {
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s", "http://x", 0)
	insertVM(t, db, "vm1", "v101", "alpha", "srv1")
	ctx := context.Background()

	// VM 不存在
	svc := NewVMService(db, &fakeVMVDIClient{})
	err := svc.SyncVMFromVDI(ctx, "missing")
	require.Error(t, err)

	// 资源服务器列表获取失败
	fail := NewVMService(db, &fakeVMVDIClient{serversErr: errors.New("boom")})
	err = fail.SyncVMFromVDI(ctx, "vm1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch VMs from VDI")

	// 目标 VM 不在列表中
	absent := NewVMService(db, &fakeVMVDIClient{servers: []VDIVMResource{{ID: "999", VMName: "other"}}, serversTotal: 1})
	err = absent.SyncVMFromVDI(ctx, "vm1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in VDI server")

	// 命中：更新内存字段（注意：该方法不落库 —— 见 SUMMARY quirk）
	hit := NewVMService(db, &fakeVMVDIClient{servers: []VDIVMResource{{
		ID: "v101", VMName: "renamed", Status: "15", IP: "10.0.0.9", AssignIP: "192.168.9.9",
		MAC: "AA:AA:AA:AA:AA:AA", CPUNumber: "2", CPUCore: "4", CPUPer: "50",
		MemAll: "8192", MemPer: "60", DiscAll: "100", DiscPer: "70", OSType: "win10",
	}}, serversTotal: 1})
	require.NoError(t, hit.SyncVMFromVDI(ctx, "vm1"))
}

func TestVMService_SyncAllAndByServer(t *testing.T) {
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s", "http://x", 0)
	insertVM(t, db, "vm1", "v1", "alpha", "srv1")
	ctx := context.Background()

	svc := NewVMService(db, &fakeVMVDIClient{})

	// 服务器不存在
	err := svc.SyncAllVMs(ctx, "missing")
	require.Error(t, err)

	// 逐台同步（每台都会失败，但整体不报错）
	require.NoError(t, svc.SyncAllVMs(ctx, "srv1"))

	// 按服务器对象同步（内部创建真实 client，fake httptest 会失败 → 报错）
	server := &models.VDIServer{}
	server.ID = "srv1"
	server.Name = "s"
	require.NoError(t, db.Exec(`UPDATE sys_vdi_server SET endpoint = '' WHERE id = 'srv1'`).Error)
	err = svc.SyncVMsFromVDIByServer(ctx, server)
	require.Error(t, err)
}

func TestVMService_Mappers(t *testing.T) {
	s := &vmServiceImpl{}

	assert.Equal(t, "pending", s.mapPowerState("10"))
	assert.Equal(t, "stopped", s.mapPowerState("11"))
	assert.Equal(t, "suspended", s.mapPowerState("12"))
	assert.Equal(t, "unknown13", s.mapPowerState("13"))
	assert.Equal(t, "unknown14", s.mapPowerState("14"))
	assert.Equal(t, "in_use", s.mapPowerState("15"))
	assert.Equal(t, "unknown", s.mapPowerState("99"))

	assert.Equal(t, "DHCP", s.mapIPType("0"))
	assert.Equal(t, "STATIC", s.mapIPType("1"))
	assert.Equal(t, "DHCP", s.mapIPType("x"))

	assert.Equal(t, 42, s.parseIntSafe("42"))
	assert.Equal(t, 0, s.parseIntSafe("notnum"))
	assert.Equal(t, 7, s.parseIntSafe("7abc"))

	assert.Equal(t, "1.1.1.1", getBestIPAddress(VDIVMResource{AssignIP: "1.1.1.1", IP: "2.2.2.2"}))
	assert.Equal(t, "2.2.2.2", getBestIPAddress(VDIVMResource{AssignIP: "-", IP: "2.2.2.2"}))
	assert.Equal(t, "2.2.2.2", getBestIPAddress(VDIVMResource{IP: "2.2.2.2"}))
}

func TestApplyVMDataScopeFilter(t *testing.T) {
	db := newVDITestDB(t)
	now := time.Now().Format("2006-01-02 15:04:05")
	u1 := "11111111-1111-1111-1111-111111111111" // dept d1
	u2 := "22222222-2222-2222-2222-222222222222" // dept d11 (d1 的子部门)
	u3 := "33333333-3333-3333-3333-333333333333" // dept d2
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, dept_id, created_at) VALUES
		(?, 'u1', 'd1', ?), (?, 'u2', 'd11', ?), (?, 'u3', 'd2', ?)`, u1, now, u2, now, u3, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_dept (id, parent_id, status) VALUES
		('d1', '', 0), ('d11', 'd1', 0), ('d2', '', 0)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, 'r1')`, u3).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_role_dept (role_id, dept_id) VALUES ('r1', 'd2')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_vm (id, vm_id, name, vdi_server_id, bound_user_id, created_at) VALUES
		('vm1', 'v1', 'a', 's', ?, ?),
		('vm2', 'v2', 'b', 's', ?, ?),
		('vm3', 'v3', 'c', 's', ?, ?)`, u1, now, u2, now, u3, now).Error)

	count := func(scope models.DataScope, userID string) int64 {
		q := ApplyVMDataScopeFilter(db.Model(&models.VDIVirtualMachine{}), userID, scope, db)
		var n int64
		require.NoError(t, q.Count(&n).Error)
		return n
	}

	assert.Equal(t, int64(3), count(models.DataScopeAll, u1), "All 不过滤")
	assert.Equal(t, int64(1), count(models.DataScopeSelf, u1), "Self 只看自己")
	assert.Equal(t, int64(1), count(models.DataScopeDept, u1), "Dept 本部门")
	assert.Equal(t, int64(2), count(models.DataScopeDeptChild, u1), "DeptChild 含子部门")
	assert.Equal(t, int64(1), count(models.DataScopeCustom, u3), "Custom 走角色-部门映射")
	assert.Equal(t, int64(0), count(models.DataScope(99), u1), "未知范围 1=0")
	assert.Equal(t, int64(0), count(models.DataScopeAll, "not-uuid"), "非法 userID 1=0")
	assert.Equal(t, int64(0), count(models.DataScopeAll, ""), "空 userID 1=0")

	// 无角色映射的 Custom
	require.NoError(t, db.Exec(`DELETE FROM sys_role_dept`).Error)
	assert.Equal(t, int64(0), count(models.DataScopeCustom, u3))

	// 用户不存在 → Dept 1=0
	assert.Equal(t, int64(0), count(models.DataScopeDept, "99999999-9999-9999-9999-999999999999"))

	// BoundUserFilter：非 All 时隐藏未绑定
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_vm (id, vm_id, name, vdi_server_id, created_at) VALUES ('vm4', 'v4', 'd', 's', ?)`, now).Error)
	var n int64
	require.NoError(t, ApplyBoundUserFilter(db.Model(&models.VDIVirtualMachine{}), models.DataScopeSelf).Count(&n).Error)
	assert.Equal(t, int64(3), n, "Self 过滤掉未绑定 vm4")
	require.NoError(t, ApplyBoundUserFilter(db.Model(&models.VDIVirtualMachine{}), models.DataScopeAll).Count(&n).Error)
	assert.Equal(t, int64(4), n, "All 不隐藏未绑定")
}
