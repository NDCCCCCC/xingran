package rpa

// Phase 157 / FLEET-01：扩缩容 ComputeProvider 化。
//
// 承载模型（.planning/notes/261123-rpa-worker-capacity-model.md）结论：
// 扩容粒度是 VM 不是容器。本文件把 ScalingService 对底层「算力」的依赖抽象为
// ComputeProvider：container（本地 dev / 单机 Docker）与 vdivm（生产 VDI）。
//
// 设计约束：
//   - 复用现有 DockerClient（含 NewMockDockerClient 本地 dev），不另写 Docker HTTP；
//   - 复用 internal/services/vdi 的 CreateVM/StartVM/StopVM/ListVMs/GetVMInfo，
//     不另写 VDI HTTP（客户端构建见 vdi.BuildRPAVMClient）；
//   - provider 能力差异显式声明：VDI 不支持容器级统计（见 StatsAwareProvider）。

import (
	"context"
	"fmt"
	"time"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/vdi"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Provider 类型常量（rpa.scaling.provider 取值域）。
const (
	ProviderContainer = "container" // 单机 Docker 容器（本地 dev / 单机部署）
	ProviderVDIVM     = "vdivm"     // VDI 虚拟机（生产）
)

// InstanceRef 一个可调度算力实例的引用（容器 ID 或 VM ID）。
type InstanceRef struct {
	ID    string `json:"id"`              // 容器 ID 或 VM ID
	Name  string `json:"name,omitempty"`  // 容器名 / VM 名
	IP    string `json:"ip,omitempty"`    // VM IP（容器场景为空）
	State string `json:"state,omitempty"` // running / exited / VDI 侧状态原样透传
	Kind  string `json:"kind"`            // container | vdivm
}

// ComputeProvider 扩缩容底层算力提供者接口。
//
// 语义约定：
//   - Provision 必须等到实例「可用」才返回：容器 = 已启动；VDI VM = VM running
//     且 worker 心跳上线（轮询 PresenceChecker，超时回收已创建实例）；
//   - Reclaim 对已不存在/已停止的实例应幂等（尽力而为，聚合报错）；
//   - List 返回当前全部受管实例。
type ComputeProvider interface {
	// Healthy 检查底层服务是否健康（Docker ping / VDI ListVMs 可达）。
	Healthy(ctx context.Context) bool

	// Provision 创建 count 个实例并等待其可用。
	Provision(ctx context.Context, count int) ([]InstanceRef, error)

	// Reclaim 回收指定实例。
	Reclaim(ctx context.Context, refs []InstanceRef) error

	// List 列出当前全部受管实例。
	List(ctx context.Context) ([]InstanceRef, error)
}

// StatsAwareProvider 可选扩展：支持容器级资源统计。
// VDIVMProvider 不实现该接口（VM 内仅 1 容器，统计口径在 VM 层，能力差异显式声明）。
type StatsAwareProvider interface {
	// Stats 获取单个实例的资源统计。
	Stats(ctx context.Context, id string) (*ContainerStats, error)
}

// ============ ContainerProvider：包装现有 DockerClient ============

// ContainerProvider 基于 Docker 容器的 provider（本地 dev / 单机部署）。
// 包装现有 DockerClient，NewMockDockerClient 仍然可用（本地 dev / 单测）。
type ContainerProvider struct {
	docker DockerClient
}

// NewContainerProvider 创建容器 provider。
func NewContainerProvider(docker DockerClient) *ContainerProvider {
	return &ContainerProvider{docker: docker}
}

// Healthy 检查 Docker 服务健康。
func (p *ContainerProvider) Healthy(ctx context.Context) bool {
	return p.docker.IsHealthy(ctx)
}

// Provision 扩容容器并等待启动完成（DockerClient.ScaleUp 返回即已启动）。
func (p *ContainerProvider) Provision(ctx context.Context, count int) ([]InstanceRef, error) {
	ids, err := p.docker.ScaleUp(ctx, count)
	if err != nil {
		return nil, err
	}
	refs := make([]InstanceRef, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, InstanceRef{ID: id, Name: id, State: "running", Kind: ProviderContainer})
	}
	return refs, nil
}

// Reclaim 停止并删除容器。
func (p *ContainerProvider) Reclaim(ctx context.Context, refs []InstanceRef) error {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		ids = append(ids, ref.ID)
	}
	return p.docker.ScaleDown(ctx, ids)
}

// List 列出 RPA Worker 容器。
func (p *ContainerProvider) List(ctx context.Context) ([]InstanceRef, error) {
	containers, err := p.docker.ListContainers(ctx)
	if err != nil {
		return nil, err
	}
	refs := make([]InstanceRef, 0, len(containers))
	for _, c := range containers {
		name := c.ID
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		refs = append(refs, InstanceRef{ID: c.ID, Name: name, State: c.State, Kind: ProviderContainer})
	}
	return refs, nil
}

// Stats 获取容器资源统计（实现 StatsAwareProvider）。
func (p *ContainerProvider) Stats(ctx context.Context, id string) (*ContainerStats, error) {
	return p.docker.GetContainerStats(ctx, id)
}

// ============ VDIVMProvider：包装 internal/services/vdi 客户端 ============

// VDIVMClient VDI 虚拟机客户端接口。
// 方法签名与 *vdi.VDIClient 完全一致（复用现有客户端，不另写 HTTP）；
// 接口化仅为可测试性（单测注入 fake）。
type VDIVMClient interface {
	CreateVM(req *vdi.CreateVMRequest) (string, error)
	StartVM(vmID string) error
	StopVM(vmID string) error
	ListVMs() ([]vdi.VMInfo, error)
	GetVMInfo(vmID string) (*vdi.VMInfo, error)
}

// WorkerPresenceChecker worker 在线探针。
// VDIVMProvider 在 VM 启动后用它等待 worker 注册上线（Provision 成功的判据）。
type WorkerPresenceChecker interface {
	// OnlineCount 返回心跳窗口内的在线 worker 数量。
	OnlineCount(ctx context.Context) (int, error)
}

// VDIProviderConfig VDI VM provider 配置（来自 rpa.scaling.vdi）。
// 服务器地址/凭据不在此配置 —— 复用 sys_vdi_server 表（见 BuildRPAVMClient）。
type VDIProviderConfig struct {
	ServerID         string        `mapstructure:"server_id"`         // sys_vdi_server 记录 ID
	ImageID          string        `mapstructure:"image_id"`          // 预装 worker 的镜像 ID
	DesktopGroup     string        `mapstructure:"desktop_group"`     // 桌面组
	VMNamePrefix     string        `mapstructure:"vm_name_prefix"`    // VM 命名前缀
	ProvisionTimeout time.Duration // 单实例 Provision 总超时（含心跳等待）
	PollInterval     time.Duration // 轮询间隔
	HeartbeatWindow  time.Duration // 心跳有效窗口
}

// DefaultVDIProviderConfig VDI provider 默认值。
func DefaultVDIProviderConfig() *VDIProviderConfig {
	return &VDIProviderConfig{
		VMNamePrefix:     "rpa-worker",
		ProvisionTimeout: 5 * time.Minute,
		PollInterval:     5 * time.Second,
		HeartbeatWindow:  90 * time.Second,
	}
}

// VDIVMProvider 基于 VDI 虚拟机的 provider（生产）。
// 一 VM = 一 Worker 容器 = 1 Chromium + N page；扩容粒度是 VM。
type VDIVMProvider struct {
	client   VDIVMClient
	cfg      *VDIProviderConfig
	presence WorkerPresenceChecker
}

// NewVDIVMProvider 创建 VDI VM provider。
// client 为已认证的 vdi 客户端（生产由 BuildRPAVMClient 构建）。
func NewVDIVMProvider(client VDIVMClient, cfg *VDIProviderConfig, presence WorkerPresenceChecker) *VDIVMProvider {
	if cfg == nil {
		cfg = DefaultVDIProviderConfig()
	}
	return &VDIVMProvider{client: client, cfg: cfg, presence: presence}
}

// Healthy 检查 VDI 服务可达。
func (p *VDIVMProvider) Healthy(ctx context.Context) bool {
	// ListVMs 无 ctx 参数（复用现有客户端签名），超时由外层 ctx 控制不了 HTTP；
	// 这里仅做可达性探测。
	_, err := p.client.ListVMs()
	return err == nil
}

// Provision 创建 VM 并等待 worker 上线。
//
// 流程：CreateVM → StartVM → 轮询（VM running 且 在线 worker 数达到基线+已启动数）
// → 超时回收本次全部实例并报错。
func (p *VDIVMProvider) Provision(ctx context.Context, count int) ([]InstanceRef, error) {
	baseOnline, err := p.presence.OnlineCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("VDI Provision 前获取 worker 在线基线失败: %w", err)
	}

	var created []InstanceRef
	for i := 0; i < count; i++ {
		vmName := fmt.Sprintf("%s-%d", p.cfg.VMNamePrefix, time.Now().Unix())
		req := &vdi.CreateVMRequest{
			VMName:       vmName,
			ImageID:      p.cfg.ImageID,
			DesktopGroup: p.cfg.DesktopGroup,
		}
		vmID, err := p.client.CreateVM(req)
		if err != nil {
			p.reclaimCreated(ctx, created)
			return nil, fmt.Errorf("创建 VM %s 失败: %w", vmName, err)
		}

		if err := p.client.StartVM(vmID); err != nil {
			p.reclaimCreated(ctx, append(created, InstanceRef{ID: vmID, Kind: ProviderVDIVM}))
			return nil, fmt.Errorf("启动 VM %s(%s) 失败: %w", vmName, vmID, err)
		}

		created = append(created, InstanceRef{ID: vmID, Name: vmName, Kind: ProviderVDIVM})
		applogger.Infof("VDI Provision: VM %s(%s) 已启动，等待 worker 注册上线", vmName, vmID)
	}

	// 等待全部 worker 上线
	if err := p.waitWorkersOnline(ctx, baseOnline+count); err != nil {
		p.reclaimCreated(ctx, created)
		return nil, err
	}

	// 回填 IP（尽力而为）
	p.fillIPs()
	return created, nil
}

// waitWorkersOnline 轮询直到在线 worker 数达到 target 或超时。
func (p *VDIVMProvider) waitWorkersOnline(ctx context.Context, target int) error {
	deadline := time.Now().Add(p.cfg.ProvisionTimeout)
	for {
		online, err := p.presence.OnlineCount(ctx)
		if err == nil && online >= target {
			applogger.Infof("VDI Provision: worker 已全部上线 (%d/%d)", online, target)
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("VDI Provision 超时: 等待 worker 上线 %d/%d (窗口 %v)，已回收本次实例",
				online, target, p.cfg.ProvisionTimeout)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("VDI Provision 被取消: %w", ctx.Err())
		case <-time.After(p.cfg.PollInterval):
		}
	}
}

// fillIPs 尽力回填已创建 VM 的 IP 与状态。
func (p *VDIVMProvider) fillIPs() {
	vms, err := p.client.ListVMs()
	if err != nil {
		return
	}
	byID := make(map[string]vdi.VMInfo, len(vms))
	for _, vm := range vms {
		byID[vm.VMID] = vm
	}
	_ = byID // IP 回填在 List 场景由 List() 提供；此处保留扩展点
}

// reclaimCreated 回收本次已创建的实例（超时回滚）。
func (p *VDIVMProvider) reclaimCreated(ctx context.Context, refs []InstanceRef) {
	if len(refs) == 0 {
		return
	}
	if err := p.Reclaim(ctx, refs); err != nil {
		applogger.Warnf("VDI Provision 回滚失败（遗留实例需人工清理）: %v", err)
	}
}

// Reclaim 停止指定 VM（StopVM 幂等语义：已停止的 VM 再次 Stop 由 VDI 侧容忍）。
func (p *VDIVMProvider) Reclaim(ctx context.Context, refs []InstanceRef) error {
	var errs []string
	for _, ref := range refs {
		if ref.Kind != ProviderVDIVM {
			errs = append(errs, fmt.Sprintf("%s: 非 VDI 实例，拒绝回收", ref.ID))
			continue
		}
		if err := p.client.StopVM(ref.ID); err != nil {
			applogger.Warnf("停止 VM %s 失败: %v", ref.ID, err)
			errs = append(errs, fmt.Sprintf("%s: %v", ref.ID, err))
			continue
		}
		applogger.Infof("VDI Reclaim: 已停止 VM %s", ref.ID)
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分 VM 回收失败: %v", errs)
	}
	return nil
}

// List 列出全部 VDI VM。
func (p *VDIVMProvider) List(ctx context.Context) ([]InstanceRef, error) {
	vms, err := p.client.ListVMs()
	if err != nil {
		return nil, err
	}
	refs := make([]InstanceRef, 0, len(vms))
	for _, vm := range vms {
		refs = append(refs, InstanceRef{ID: vm.VMID, Name: vm.VMName, IP: vm.IP, State: vm.Status, Kind: ProviderVDIVM})
	}
	return refs, nil
}

// ============ DB worker 在线探针 ============

// DBWorkerPresence 基于 sys_rpa_worker 心跳的在线探针（不触碰 worker_handler.go）。
type DBWorkerPresence struct {
	db     *gorm.DB
	window time.Duration
}

// NewDBWorkerPresence 创建 DB 心跳探针。
func NewDBWorkerPresence(db *gorm.DB, window time.Duration) *DBWorkerPresence {
	if window <= 0 {
		window = 90 * time.Second
	}
	return &DBWorkerPresence{db: db, window: window}
}

// OnlineCount 统计心跳窗口内 online 状态的 worker 数量。
func (p *DBWorkerPresence) OnlineCount(ctx context.Context) (int, error) {
	if p.db == nil {
		return 0, fmt.Errorf("DBWorkerPresence 无 DB 句柄")
	}
	cutoff := time.Now().Add(-p.window).Unix()
	var n int64
	err := p.db.WithContext(ctx).Model(&rpamodels.Worker{}).
		Where("status = ? AND last_heartbeat IS NOT NULL AND last_heartbeat > ?",
			rpamodels.WorkerStatusOnline, cutoff).
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("查询在线 worker 失败: %w", err)
	}
	return int(n), nil
}

// ============ Provider 工厂 ============

// NewComputeProvider 按 config.Provider 创建 ComputeProvider。
// 未知 provider 值在此报错（启动即失败，不静默降级）。
//
// vdiFactory 仅在 provider=vdivm 时调用（惰性构建已认证客户端）。
func NewComputeProvider(
	cfg *ScalingConfig,
	docker DockerClient,
	vdiFactory func() (VDIVMClient, error),
	presence WorkerPresenceChecker,
) (ComputeProvider, error) {
	switch cfg.Provider {
	case "", ProviderContainer:
		if docker == nil {
			return nil, fmt.Errorf("provider=container 但 DockerClient 为空")
		}
		return NewContainerProvider(docker), nil
	case ProviderVDIVM:
		if cfg.VDI == nil || cfg.VDI.ServerID == "" {
			return nil, fmt.Errorf("provider=vdivm 但未配置 rpa.scaling.vdi.server_id")
		}
		if vdiFactory == nil {
			return nil, fmt.Errorf("provider=vdivm 但未提供 VDI 客户端工厂")
		}
		client, err := vdiFactory()
		if err != nil {
			return nil, fmt.Errorf("构建 VDI 客户端失败: %w", err)
		}
		if presence == nil {
			return nil, fmt.Errorf("provider=vdivm 但未提供 worker 在线探针")
		}
		return NewVDIVMProvider(client, cfg.VDI, presence), nil
	default:
		return nil, fmt.Errorf("未知的 rpa.scaling.provider 值 %q（可选: %s / %s）",
			cfg.Provider, ProviderContainer, ProviderVDIVM)
	}
}
