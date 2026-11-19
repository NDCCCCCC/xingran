package portwrite

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ====================== Phase 127 前置（PWX-04）审计查询测试 ======================

// TestListAudit_FilterAndPaginate 分页 + 各过滤维度。
func TestListAudit_FilterAndPaginate(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	rows := []*models.PortWriteAudit{
		{DeviceID: "dev-1", PortID: "p1", Action: "shutdown", Status: "succeeded", Operator: "alice", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{DeviceID: "dev-1", PortID: "p2", Action: "set_access_vlan", Status: "failed", Operator: "bob", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{DeviceID: "dev-2", PortID: "p3", Action: "shutdown", Status: "succeeded", Operator: "alice", CreatedAt: time.Now()},
	}
	for _, r := range rows {
		require.NoError(t, db.Create(r).Error)
	}
	ctx := context.Background()

	// 全量分页：created_at 倒序
	res, err := svc.ListAudit(ctx, AuditListRequest{Current: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.Total)
	assert.Len(t, res.List, 2)
	assert.Equal(t, "dev-2", res.List[0].DeviceID, "created_at 倒序")

	// status 过滤
	res, err = svc.ListAudit(ctx, AuditListRequest{Status: "failed"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "bob", res.List[0].Operator)

	// action + deviceId 组合
	res, err = svc.ListAudit(ctx, AuditListRequest{DeviceID: "dev-1", Action: "shutdown"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	// operator 过滤
	res, err = svc.ListAudit(ctx, AuditListRequest{Operator: "alice"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)

	// 时间范围：只取最近 90 分钟
	start := time.Now().Add(-90 * time.Minute)
	res, err = svc.ListAudit(ctx, AuditListRequest{StartTime: &start})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)

	// portId 过滤
	res, err = svc.ListAudit(ctx, AuditListRequest{PortID: "p3"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	// 默认分页兜底（零值 → current=1, pageSize=20）
	res, err = svc.ListAudit(ctx, AuditListRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Current)
	assert.Equal(t, 20, res.PageSize)
}

// TestListAudit_BatchIdFilter Phase 122 收尾：batchId 精确对账过滤。
func TestListAudit_BatchIdFilter(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	rows := []*models.PortWriteAudit{
		{DeviceID: "dev-1", PortID: "p1", Action: "shutdown", Status: "succeeded", Operator: "a", BatchID: "batch-A"},
		{DeviceID: "dev-1", PortID: "p2", Action: "shutdown", Status: "succeeded", Operator: "a", BatchID: "batch-A"},
		{DeviceID: "dev-1", PortID: "p3", Action: "shutdown", Status: "succeeded", Operator: "a"}, // 历史行 NULL
	}
	for _, r := range rows {
		require.NoError(t, db.Create(r).Error)
	}
	ctx := context.Background()

	res, err := svc.ListAudit(ctx, AuditListRequest{BatchID: "batch-A"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "batchId 等值过滤")

	res, err = svc.ListAudit(ctx, AuditListRequest{BatchID: "batch-NULL-not-exist"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Total)

	// 无 batchId → 全量（含 NULL 历史行）
	res, err = svc.ListAudit(ctx, AuditListRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.Total)
}

// ====================== v1.57 名称可读性（快照 + 实时兜底） ======================

// TestListAudit_EnrichFromSnapshot 快照非空时优先用快照，不再回查主数据。
func TestListAudit_EnrichFromSnapshot(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	// 主数据名称与快照故意不同：审计必须显示"操作当时"的快照名
	require.NoError(t, db.Create(&models.NetworkDevice{
		BaseModel:  models.BaseModel{ID: "dev-snap"},
		DeviceName: "改名后的设备",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.9.9.9",
	}).Error)
	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-snap", DeviceID: "dev-snap", InterfaceName: "GE9/9/9",
		Description: "改名后的描述", CollectedAt: time.Now(),
	}).Error)

	require.NoError(t, db.Create(&models.PortWriteAudit{
		DeviceID: "dev-snap", PortID: "port-snap", Action: "shutdown", Status: "succeeded",
		Operator: "alice", CreatedAt: time.Now(),
		DeviceNameSnapshot: "核心交换机-01", DeviceIPSnapshot: "10.1.1.1",
		InterfaceNameSnapshot: "GE0/0/1", PortDescriptionSnapshot: "上行到核心",
	}).Error)

	res, err := svc.ListAudit(context.Background(), AuditListRequest{})
	require.NoError(t, err)
	require.Len(t, res.List, 1)
	row := res.List[0]
	assert.Equal(t, "核心交换机-01", row.DeviceName, "快照优先于实时主数据")
	assert.Equal(t, "10.1.1.1", row.DeviceIP)
	assert.Equal(t, "GE0/0/1", row.InterfaceName)
	assert.Equal(t, "上行到核心", row.PortDescription)
	assert.True(t, row.NameFromSnapshot)
	// 原 uuid 字段仍保留（断线对账等既有消费方零改动）
	assert.Equal(t, "dev-snap", row.DeviceID)
	assert.Equal(t, "port-snap", row.PortID)
}

// TestListAudit_EnrichFallbackToLiveTables 快照为空（migration 254 前的历史行）
// 时回退实时表补名，设备软删除也必须能补出名称。
func TestListAudit_EnrichFallbackToLiveTables(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	dev := &models.NetworkDevice{
		BaseModel:  models.BaseModel{ID: "dev-live"},
		DeviceName: "接入交换机-02",
		DeviceType: models.DeviceTypeSwitch,
		Vendor:     models.VendorHuawei,
		IPAddress:  "10.2.2.2",
	}
	require.NoError(t, db.Create(dev).Error)
	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-live", DeviceID: "dev-live", InterfaceName: "GE0/0/8",
		Description: "会议室", CollectedAt: time.Now(),
	}).Error)
	// 设备软删除：审计行仍需显示名称（Unscoped 兜底）
	require.NoError(t, db.Delete(dev).Error)

	require.NoError(t, db.Create(&models.PortWriteAudit{
		DeviceID: "dev-live", PortID: "port-live", Action: "shutdown",
		Status: "succeeded", Operator: "bob", CreatedAt: time.Now(),
	}).Error)

	res, err := svc.ListAudit(context.Background(), AuditListRequest{})
	require.NoError(t, err)
	require.Len(t, res.List, 1)
	row := res.List[0]
	assert.Equal(t, "接入交换机-02", row.DeviceName, "软删设备仍需补名")
	assert.Equal(t, "10.2.2.2", row.DeviceIP)
	assert.Equal(t, "GE0/0/8", row.InterfaceName)
	assert.Equal(t, "会议室", row.PortDescription)
	assert.False(t, row.NameFromSnapshot, "实时补名不算快照")
}

// TestListAudit_EnrichMissingMasterData 主数据都查不到时名称留空（不 panic、不塞 UUID）。
func TestListAudit_EnrichMissingMasterData(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	require.NoError(t, db.Create(&models.PortWriteAudit{
		DeviceID: "dev-gone", PortID: "port-gone", Action: "shutdown",
		Status: "failed", Operator: "c", CreatedAt: time.Now(),
	}).Error)

	res, err := svc.ListAudit(context.Background(), AuditListRequest{})
	require.NoError(t, err)
	require.Len(t, res.List, 1)
	row := res.List[0]
	assert.Empty(t, row.DeviceName)
	assert.Empty(t, row.InterfaceName)
	assert.Equal(t, "dev-gone", row.DeviceID, "UUID 仍原样返回供前端兜底展示")
	assert.Equal(t, "port-gone", row.PortID)
}

// TestListAudit_FilterByName 名称模糊过滤（设备名 / 接口名）+ LIKE 通配符转义。
func TestListAudit_FilterByName(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	require.NoError(t, db.Create(&models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "dev-a"}, DeviceName: "核心交换机-01",
		DeviceType: models.DeviceTypeSwitch, Vendor: models.VendorHuawei, IPAddress: "10.3.3.1",
	}).Error)
	require.NoError(t, db.Create(&models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "dev-b"}, DeviceName: "接入交换机-01",
		DeviceType: models.DeviceTypeSwitch, Vendor: models.VendorHuawei, IPAddress: "10.3.3.2",
	}).Error)
	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-a1", DeviceID: "dev-a", InterfaceName: "GE0/0/1", CollectedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-b1", DeviceID: "dev-b", InterfaceName: "GE0/0/1", CollectedAt: time.Now(),
	}).Error)

	base := time.Now()
	rows := []*models.PortWriteAudit{
		{DeviceID: "dev-a", PortID: "port-a1", Action: "shutdown", Status: "succeeded", CreatedAt: base},
		{DeviceID: "dev-b", PortID: "port-b1", Action: "shutdown", Status: "succeeded", CreatedAt: base.Add(-time.Minute)},
	}
	for _, r := range rows {
		require.NoError(t, db.Create(r).Error)
	}
	ctx := context.Background()

	res, err := svc.ListAudit(ctx, AuditListRequest{DeviceName: "核心"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total, "设备名模糊过滤")
	assert.Equal(t, "dev-a", res.List[0].DeviceID)

	// 接口名跨设备重复：模糊过滤会命中两台设备的同名列
	res, err = svc.ListAudit(ctx, AuditListRequest{InterfaceName: "GE0/0/1"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)

	// 名称过滤 + deviceId 等值 → 交集
	res, err = svc.ListAudit(ctx, AuditListRequest{InterfaceName: "GE0", DeviceID: "dev-b"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "dev-b", res.List[0].DeviceID)

	// 通配符转义：字面量 % 不应匹配全部
	res, err = svc.ListAudit(ctx, AuditListRequest{DeviceName: "%"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Total, "LIKE 通配符必须转义为字面量")
}

// TestEscapeLike 单元：% _ 反斜杠 全部转义。
func TestEscapeLike(t *testing.T) {
	assert.Equal(t, "100"+`\%`, escapeLike("100%"))
	assert.Equal(t, "a"+`\_`+"b", escapeLike("a_b"))
	assert.Equal(t, "c"+`\`+"d", escapeLike(`c\d`))
	assert.Equal(t, "normal", escapeLike("normal"))
}

// ====================== v1.57 审计页选择器端点 ======================

func TestListDeviceOptions_KeywordAndSoftDeleted(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	devA := &models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "dev-opt-1"}, DeviceName: "核心交换机",
		DeviceType: models.DeviceTypeSwitch, Vendor: models.VendorHuawei, IPAddress: "10.4.4.1",
	}
	devB := &models.NetworkDevice{
		BaseModel: models.BaseModel{ID: "dev-opt-2"}, DeviceName: "接入交换机",
		DeviceType: models.DeviceTypeSwitch, Vendor: models.VendorHuawei, IPAddress: "10.4.4.2",
	}
	require.NoError(t, db.Create(devA).Error)
	require.NoError(t, db.Create(devB).Error)
	// 软删设备仍要出现在选择器（历史审计行指向它）
	require.NoError(t, db.Delete(devB).Error)

	ctx := context.Background()
	res, err := svc.ListDeviceOptions(ctx, AuditOptionRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "软删设备必须仍可选")
	assert.Len(t, res.List, 2)

	res, err = svc.ListDeviceOptions(ctx, AuditOptionRequest{Keyword: "接入"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "dev-opt-2", res.List[0].Value)
	assert.Equal(t, "接入交换机", res.List[0].Label)
	assert.Equal(t, "10.4.4.2", res.List[0].IPAddress)

	// IP 也能搜（重名设备靠 IP 定位）
	res, err = svc.ListDeviceOptions(ctx, AuditOptionRequest{Keyword: "10.4.4.1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "dev-opt-1", res.List[0].Value)
}

func TestListPortOptions_RequiresDeviceAndFilters(t *testing.T) {
	db := newTestDB(t)
	svc := newTestService(&countingExecutor{}, nil, db)

	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-opt-1", DeviceID: "dev-p1", InterfaceName: "GE0/0/1",
		Description: "上行到核心", CollectedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-opt-2", DeviceID: "dev-p1", InterfaceName: "GE0/0/2",
		Description: "会议室", CollectedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&models.DevicePortStatus{
		ID: "port-opt-3", DeviceID: "dev-p2", InterfaceName: "GE0/0/1",
		Description: "其他设备同名列", CollectedAt: time.Now(),
	}).Error)

	ctx := context.Background()
	// deviceId 必填 —— 接口名跨设备重名，禁止全局裸选
	_, err := svc.ListPortOptions(ctx, AuditOptionRequest{})
	assert.ErrorIs(t, err, ErrAuditPortOptionDeviceRequired)

	res, err := svc.ListPortOptions(ctx, AuditOptionRequest{DeviceID: "dev-p1"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total, "只返回该设备的端口")
	assert.Len(t, res.List, 2)

	res, err = svc.ListPortOptions(ctx, AuditOptionRequest{DeviceID: "dev-p1", Keyword: "会议室"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "port-opt-2", res.List[0].Value)
	assert.Equal(t, "GE0/0/2", res.List[0].Label)
	assert.Equal(t, "GE0/0/2", res.List[0].InterfaceName)
	assert.Equal(t, "会议室", res.List[0].Description)
}
