package rpa

// Phase 157 / FLEET-01 验收单测：
//   1) provider 选择三档（container / vdivm / 未知值报错）
//   2) mock VDI provider 下的扩缩容全流程（Provision 成功 / 超时回收 / Reclaim / List）

import (
	"context"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services/vdi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeVDIClient VDI 客户端 fake（不发起 HTTP）。
type fakeVDIClient struct {
	createdVMs []string
	started    []string
	stopped    []string
	startErr   map[string]error
	vms        []vdi.VMInfo
}

func (f *fakeVDIClient) CreateVM(req *vdi.CreateVMRequest) (string, error) {
	id := "vm-" + req.VMName
	f.createdVMs = append(f.createdVMs, id)
	f.vms = append(f.vms, vdi.VMInfo{VMID: id, VMName: req.VMName, IP: "10.0.0.1", Status: "running"})
	return id, nil
}

func (f *fakeVDIClient) StartVM(vmID string) error {
	if err, ok := f.startErr[vmID]; ok {
		return err
	}
	f.started = append(f.started, vmID)
	return nil
}

func (f *fakeVDIClient) StopVM(vmID string) error {
	f.stopped = append(f.stopped, vmID)
	return nil
}

func (f *fakeVDIClient) ListVMs() ([]vdi.VMInfo, error) { return f.vms, nil }

func (f *fakeVDIClient) GetVMInfo(vmID string) (*vdi.VMInfo, error) {
	for i := range f.vms {
		if f.vms[i].VMID == vmID {
			return &f.vms[i], nil
		}
	}
	return nil, nil
}

// fakePresence 可控的 worker 在线探针 fake。
type fakePresence struct {
	count int
	err   error
}

func (f *fakePresence) OnlineCount(ctx context.Context) (int, error) {
	return f.count, f.err
}

func vdiTestCfg() *ScalingConfig {
	cfg := DefaultScalingConfig()
	cfg.Provider = ProviderVDIVM
	d := DefaultVDIProviderConfig()
	d.ServerID = "srv-1"
	cfg.VDI = d
	return cfg
}

// 1) provider 选择三档
func TestNewComputeProvider_Selection(t *testing.T) {
	ctx := context.Background()

	// container（显式）
	p, err := NewComputeProvider(&ScalingConfig{Provider: ProviderContainer}, NewMockDockerClient(), nil, nil)
	require.NoError(t, err)
	assert.True(t, p.Healthy(ctx), "mock Docker 默认健康")

	// container（空值默认）
	p, err = NewComputeProvider(&ScalingConfig{}, NewMockDockerClient(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, p)

	// vdivm
	p, err = NewComputeProvider(vdiTestCfg(), nil, func() (VDIVMClient, error) {
		return &fakeVDIClient{}, nil
	}, &fakePresence{})
	require.NoError(t, err)
	assert.True(t, p.Healthy(ctx))

	// 未知值 → 启动即报错
	_, err = NewComputeProvider(&ScalingConfig{Provider: "kubernetes"}, nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未知的 rpa.scaling.provider")

	// vdivm 但缺 server_id → 报错
	cfg := vdiTestCfg()
	cfg.VDI.ServerID = ""
	_, err = NewComputeProvider(cfg, nil, func() (VDIVMClient, error) {
		return &fakeVDIClient{}, nil
	}, &fakePresence{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server_id")
}

// 2) VDIVMProvider 扩容全流程（worker 逐个上线）
func TestVDIVMProvider_Provision_FullFlow(t *testing.T) {
	ctx := context.Background()
	fake := &fakeVDIClient{}
	presence := &fakePresence{count: 1} // 基线 1 台在线
	cfg := DefaultVDIProviderConfig()
	cfg.ServerID = "srv-1"
	cfg.PollInterval = 10 * time.Millisecond

	provider := NewVDIVMProvider(fake, cfg, presence)

	// 模拟 worker 逐台上线：用 goroutine 在短延迟后递增在线数
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(30 * time.Millisecond)
		presence.count = 3 // 1 基线 + 2 新增
	}()

	refs, err := provider.Provision(ctx, 2)
	<-done
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, ProviderVDIVM, refs[0].Kind)
	assert.Len(t, fake.createdVMs, 2)
	assert.Len(t, fake.started, 2)
	assert.Empty(t, fake.stopped, "成功路径不回收")
}

// 3) VDIVMProvider 超时 → 回收全部本次实例
func TestVDIVMProvider_Provision_Timeout_Reclaim(t *testing.T) {
	ctx := context.Background()
	fake := &fakeVDIClient{}
	presence := &fakePresence{count: 0}
	cfg := DefaultVDIProviderConfig()
	cfg.ServerID = "srv-1"
	cfg.ProvisionTimeout = 50 * time.Millisecond
	cfg.PollInterval = 10 * time.Millisecond

	provider := NewVDIVMProvider(fake, cfg, presence)

	_, err := provider.Provision(ctx, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "超时")
	assert.Len(t, fake.createdVMs, 2, "创建了 2 台")
	assert.Len(t, fake.stopped, 2, "超时后全部回收")
}

// 4) StartVM 失败 → 回滚已创建实例
func TestVDIVMProvider_StartVM_Failure_Rollback(t *testing.T) {
	ctx := context.Background()
	fake := &fakeVDIClient{}
	alwaysFail := &failStartClient{inner: fake}
	cfg := DefaultVDIProviderConfig()
	cfg.ServerID = "srv-1"
	cfg.ProvisionTimeout = 200 * time.Millisecond

	provider := NewVDIVMProvider(alwaysFail, cfg, &fakePresence{count: 0})
	_, err := provider.Provision(ctx, 2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "启动 VM")
	assert.Len(t, fake.stopped, 1, "第一台 Start 失败后该台被回滚，第二台不再创建")
	assert.Len(t, fake.createdVMs, 1, "Start 失败立即中止，不再创建后续实例")
}

// failStartClient StartVM 永远失败。
type failStartClient struct{ inner *fakeVDIClient }

func (f *failStartClient) CreateVM(req *vdi.CreateVMRequest) (string, error) {
	return f.inner.CreateVM(req)
}
func (f *failStartClient) StartVM(vmID string) error { return assert.AnError }
func (f *failStartClient) StopVM(vmID string) error  { return f.inner.StopVM(vmID) }
func (f *failStartClient) ListVMs() ([]vdi.VMInfo, error) {
	return f.inner.ListVMs()
}
func (f *failStartClient) GetVMInfo(vmID string) (*vdi.VMInfo, error) {
	return f.inner.GetVMInfo(vmID)
}

// 5) Reclaim 幂等 + List 映射 + Reclaim 拒绝非 VDI 实例
func TestVDIVMProvider_Reclaim_List(t *testing.T) {
	ctx := context.Background()
	fake := &fakeVDIClient{vms: []vdi.VMInfo{
		{VMID: "vm-a", VMName: "rpa-worker-a", IP: "10.0.0.2", Status: "running"},
		{VMID: "vm-b", VMName: "rpa-worker-b", Status: "stopped"},
	}}
	cfg := DefaultVDIProviderConfig()
	cfg.ServerID = "srv-1"
	provider := NewVDIVMProvider(fake, cfg, &fakePresence{})

	refs, err := provider.List(ctx)
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, "10.0.0.2", refs[0].IP)

	// 重复回收同一实例（幂等：fake 不报错）
	require.NoError(t, provider.Reclaim(ctx, refs))
	require.NoError(t, provider.Reclaim(ctx, refs))
	assert.Len(t, fake.stopped, 4)

	// 非 VDI 实例拒绝
	err = provider.Reclaim(ctx, []InstanceRef{{ID: "c1", Kind: ProviderContainer}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非 VDI 实例")
}

// 6) DBWorkerPresence SQL 烟测（真实 sqlite 表）
func TestDBWorkerPresence_OnlineCount(t *testing.T) {
	db := newMetricsTestDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_rpa_workers (id, status, last_heartbeat) VALUES
		('w1','online',strftime('%s','now')-10),
		('w2','online',strftime('%s','now')-3600),
		('w3','offline',strftime('%s','now')-10)`).Error)

	p := NewDBWorkerPresence(db, 90*time.Second)
	n, err := p.OnlineCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n, "仅心跳窗口内的 online worker 计入")
}
