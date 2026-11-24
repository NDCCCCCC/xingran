# Phase 157 FLEET 承载与 VDI 编排 — SUMMARY（v1.56 W2）

- **日期**: 2026-11-24
- **worktree**: `D:/CODE/ClaudeCode/wt-157`（branch `gsd/phase-157-fleet`，自 main `2a0843b4` 切出）
- **前置确认**: Phase 154 已合入 main（`2a096187` merge commit 在 `git log -8` 可见），并发 schema 冻结确认。
- **并行归属**: 与 Phase 153 并行执行；本波次零文件交集（153 不碰 rpa 服务三文件与根 config）。

## 逐条 REQ

### FLEET-01 ScalingService provider 化 — PASS

**provider 接口定义**（`internal/services/rpa/compute_provider.go`）：

```go
type ComputeProvider interface {
    Healthy(ctx context.Context) bool
    Provision(ctx context.Context, count int) ([]InstanceRef, error) // 等实例可用才返回
    Reclaim(ctx context.Context, refs []InstanceRef) error           // 幂等
    List(ctx context.Context) ([]InstanceRef, error)
}
type InstanceRef struct{ ID, Name, IP, State, Kind string } // Kind: container | vdivm
```

- `ContainerProvider` 包装现有 `DockerClient`；`NewMockDockerClient` 路径原样保留（本地 dev / 单测）。
  可选扩展接口 `StatsAwareProvider`（仅容器实现），能力差异显式声明。
- `VDIVMProvider` 包装 `internal/services/vdi` 的 `CreateVM/StartVM/StopVM/ListVMs/GetVMInfo`
  （接口化 `VDIVMClient`，签名与 `*vdi.VDIClient` 一致，不另写 HTTP）。Provision 流程：
  CreateVM → StartVM → 轮询（worker 心跳上线数 ≥ 基线+count）→ 成功返回；超时/启动失败回收本次实例。
  心跳探针 `DBWorkerPresence` 查 `sys_rpa_worker.last_heartbeat`（不触碰 `worker_handler.go`）。
- 凭据复用：新增 `internal/services/vdi/rpa_vm_client.go` 的 `BuildRPAVMClient(db, serverID)` ——
  从 `sys_vdi_server` 加载 endpoint + 解密密码 + 认证（解密函数未导出所以工厂必须放 vdi 包内）。
- `NewScalingService` 签名改 `(db, config, dockerConfig) (ScalingService, error)`；按
  `config.Provider` 选择，**未知值启动即报错**（`TestNewComputeProvider_Selection` 三档断言）。
- 根 `internal/config/config.go` `RPAScalingConfig` **只加不删**：新增 `Provider string` +
  `RPAScalingVDIConfig{ServerID/ImageID/DesktopGroup/VMNamePrefix}`（不另造 VDI 客户端配置，
  复用 `sys_vdi_server`）。原 `Docker DockerConfig` 字段保留。
- **装配说明（越界申报）**：`internal/core/core.go` 不在三文件独占清单内，但 provider 键传递
  必须改 `initRPAScalingService` 装配段（约 30 行，函数级改动）。153/158 均不碰该函数，无冲突。
- **测试**：`compute_provider_test.go` —— 三档选择 + VDI 全流程（成功/超时回收/启动失败回滚/
  Reclaim 幂等/List 映射）+ DBWorkerPresence SQL 烟测；`metrics_scaling_docker_test.go` 适配新签名。

### FLEET-02 一 VM 一容器落地 — PASS

- 两份 compose（`docker-compose.yml` / `docker-compose.local.yml`）+ `configs/config.yaml`
  按 8GB VM 基准统一：`shm_size: 1gb`（配合已有 `--disable-dev-shm-usage`）、`memory: 6G`、
  `cpus: '4'`、`MAX_CONCURRENCY:-8` == `max_concurrency: 8`。
- `rpa-worker/configs/config.yaml` 只改 `max_concurrency` **取值**（5→8）；154 的键语义与
  旧 `max_pages` 注释保留原样，未动 schema。
- **配置漂移校验**：`rpa-worker/deployments/drift_test.go`（4 个测试全过）：
  ① compose MAX_CONCURRENCY == config max_concurrency == README 锚点 `<!-- capacity-n: 8 -->`；
  ② 资源参数（shm/memory/cpus/disable-dev-shm-usage）断言；
  ③ 旧键 `max_pages` 非注释存在 → `t.Logf WARN` 容忍（协议第 1 条中间态），不存在亦通过；
  ④ `docker compose config` 解析 —— 本机无 docker **SKIP 登记**（UAT-FLEET-02）。
- `rpa-worker/deployments/README.md` 新增「容量模型」章节：一 VM 一容器一 Chromium 多 page、
  为何不是一 VM 多容器（shm/带宽/运维三乘法表）、扩缩容粒度是 VM 不是容器、多容器保留场景。

### FLEET-03 容量实测口径与执行 — PASS（宿主机部分实测；VDI 部分登记 UAT）

- 实测工具 `rpa-worker/cmd/capacitybench/main.go`（rod + tasklist，零新增依赖；PID 基线过滤）。
- **本机实测（Windows 宿主机，1 Chromium + N page，非容器口径，已注明限制）**：

| N | maxRSS(MB) | avgRSS(MB) | chrome 进程(max) | pageLoadFail | browserCrash |
|---|---|---|---|---|---|
| 1 | 409 | 403 | 7 | 0 | 0 |
| 2 | 485 | 468 | 9 | 0 | 0 |
| 4 | 607 | 593 | 11 | 0 | 0 |
| 8 | 859 | 843 | 15 | 0 | 0 |

  每 page 边际 ≈ 64MB（优于笔记 §3 推算的 15–40MB+进程开销假设的总量级），0 崩溃。
- **UAT 待办**（`notes/261124-fleet03-uat-capacity.md`，三条，含 owner/窗口/证据格式）：
  UAT-FLEET-01 两档 VM×N 矩阵（P95/OOM/screencast 带宽/容器 RSS，需真机 VDI + 生产 PG 授权）；
  UAT-FLEET-02 docker compose 解析（无 docker 环境）；UAT-FLEET-03 vdivm 真机全流程。
  **本环境无真机 VDI、无生产 PG 写授权，全部未执行、未勾选、零推算冒充。**
- **决策结论**：「一 VM 一容器」被宿主机实测**支持**（RSS 亚线性 + 零崩溃 + 6G 限额余量 >5×）；
  数字级默认 N=8 为待 UAT 确认的部署基线。已回填 capacity-model 笔记 §3.1/§5。

## 交付验证

| 项 | 结果 |
|---|---|
| `go build ./...`（主仓库） | PASS |
| `go test ./internal/services/rpa/ ./internal/config/` | PASS |
| `cd rpa-worker && go build ./...` | PASS |
| `cd rpa-worker && go test ./deployments/` | PASS（3 PASS + 1 SKIP 登记） |
| `go test ./internal/services/vdi/` | PASS（回归） |

## 遗留项 / 越界申报

1. `internal/core/core.go`：装配段改动（不在三文件独占清单，函数级隔离，见 FLEET-01 说明）。
2. `rpa-worker/cmd/capacitybench/`：新增常驻压测工具（rpa-worker 子模块自治，供 UAT 复用）。
3. 旧键 `max_pages` 若部署环境配置中仍存在：运行时 WARN 迁移，漂移校验容忍，后续清理。
4. `worker_handler.go` / `docker_client.go` / `rpa-worker/internal/config/config.go` 未碰（归属合规）。
5. 本地 dev 的 `container` provider 真机 Docker 路径未实测（无 docker，同 UAT-FLEET-02 窗口）。
