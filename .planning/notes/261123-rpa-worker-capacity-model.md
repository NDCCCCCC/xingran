---
created: 2026-11-23
topic: RPA Worker 部署承载模型（一 VM 多容器 vs 一容器多浏览器）
status: decision-input（结论需 CAP-01 实测确认）
related:
  - rpa-worker/internal/browser/pool.go
  - rpa-worker/internal/browser/chrome_page.go
  - rpa-worker/deployments/docker-compose.yml
  - internal/services/rpa/scaling_service.go
  - internal/services/vdi/client.go
---

# RPA Worker 承载模型分析（VM / 容器 / 浏览器实例）

> 本文是 v1.56 Phase「CAPACITY」的决策输入。结构性结论可先定，**具体容量数字必须由 CAP-01 实测回填**，不得把推算当实测。

## 1. 现状代码事实（`文件:行号` 实证）

| 事实 | 位置 | 影响 |
|---|---|---|
| 每个 page 都启动一个独立 Chrome 进程 | `pool.go:createPage` → `chrome_page.go:NewChromePage` 里 `launcher.MustLaunch()` + `rod.New().ControlURL().MustConnect()` + `browser.MustPage()` | `browser.max_pages` 被当成「页面数」用，但实际含义是「Chrome 进程数」。`max_pages: 10` = 单 worker 最多 10 个 Chromium 进程 |
| 页面 Close 连带关浏览器 | `chrome_page.go:Close()` → `p.browser.Close()` | 池化回收实际是销毁 Chrome 再重建，冷启动成本无法摊销 |
| 容器内存上限 2G + shm 2g | `deployments/docker-compose.yml` `deploy.resources.limits.memory: 2G`、`shm_size: 2gb` | 若一 VM 跑 N 个容器，shm 与内存上限按 N 倍叠加，是硬性否决因素 |
| 扩缩容按容器（Docker Remote API） | `scaling_service.go` + `docker_client.go`（含 mock 客户端） | 与「VM 由 VDI API 管理」的现实不符：容器是 VM 内部的事，VM 才是调度单位 |
| VDI 已具备 VM 生命周期 API | `internal/services/vdi/client.go`：`CreateVM/StartVM/StopVM/RestartVM/BindUser/ListVMs/GetVMInfo` | 扩缩容天然应落在 VM 层，容器层只需被镜像预置 |

**结论一（结构性）**：现实现的「每 page 一个 Chrome」必须改。否则讨论容器数量没有意义——单容器就会吃掉与并发数线性相关的 Chrome 进程内存。

## 2. 三种承载形态对比

| 形态 | 说明 | 优点 | 缺点 |
|---|---|---|---|
| **A. 1 VM + 1 容器 + 1 Chrome + N page** | 容器内单 Chromium 进程，按并发开 N 个 page | 内存最优；冷启动一次；shm 只一份；运维面最小 | 单进程故障会带走整组 page（需引擎级隔离与重试） |
| **B. 1 VM + M 容器 + 每容器 1 Chrome + n page** | 现 compose 的横向复制 | 故障隔离好；可滚动升级 | 内存 ×M（含 shm ×M）；Chrome 冷启动 ×M；端口/健康检查/日志 ×M |
| **C. 1 VM + 1 容器 + N Chrome + 每 Chrome 1 page** | 现状语义 | 页面间彻底隔离 | 内存与启动成本最高；与「一般员工会开 5-20 个会话」的现实直接冲突 |

## 3. 结构性推算（假设标注，待 CAP-01 实测）

假设（Chromium 124+ headless=new，Linux x86_64，无 GPU）：
- 单个 Chromium 进程基线 RSS ≈ 100–180 MB（空页 idle）；
- 同进程内每额外 page 增量 ≈ 15–40 MB（取决于页面资源）；
- 容器本身（Go worker + 日志）≈ 30–60 MB；
- 每 page 的系统开销还包含 tmpfs/shm 与 page cache，峰值由页面内容决定。

则 8 GB / 4 vCPU 虚拟机：
- 形态 A（1 Chrome + 8 page）：≈ 180 + 8×35 + 50 ≈ **0.5 GB**，余量充足，可把所有 VM 内存留给 page cache 与突发；
- 形态 B（4 容器 × 2 page）：≈ 4×(150+2×35+50) + shm 4×2GB（但 shm 实际按需占用）≈ **1.0–1.2 GB** 常驻 + 4 份启动抖动；
- 形态 C（8 Chrome × 1 page）：≈ 8×150 + 50 ≈ **1.25 GB** 且启动时会有 8 次 ~200-400ms 冷启动叠加。

**结论二**：在 8–16 GB 的 VDI 规格下，**形态 A 明显占优**；只有当单 VM ≥ 32 GB 且需要强故障隔离/多租户时才值得上形态 B，且应把 shm 降为 512MB–1GB 并显式配置 `--disable-dev-shm-usage`（现已在 `chrome_page.go` launcher flags 里）。

## 4. 推荐决策（写入 v1.56 CAPACITY 交付）

1. **默认形态 = A：一 VM 一 Worker 容器，容器内 1 个 Chromium + N 个 page。**
   - N = `worker.max_concurrency`（默认 5，8 GB VM 可到 8–10），与 `browser.max_pages` 语义统一为「页面数」而非「进程数」。
2. **扩容粒度 = VM，不是容器。** `ScalingService` 的 Docker provider 抽象改为 provider 化：`ContainerProvider`（本地 dev）/ `VDIVMProvider`（生产），按 `rpa.scaling.provider` 选择；VM 创建后由 cloud-init/预置镜像拉起容器（镜像已预装 worker）。
3. **故障隔离靠引擎重试而非进程复制**：单 Chrome 崩溃时引擎捕获 `rod` 错误 → 销毁重建 Chrome → 该执行标记 failed 并按 `retry_count` 重跑；不因此扩大进程数。
4. **一 VM 多容器的唯一保留场景**：需要「不同租户/不同浏览器版本/滚动升级不停机」。此时每容器 shm ≤ 1GB 且 VM ≥ 32 GB，并在 CAP-02 决策矩阵中单列。
5. **每 VM 上限先按「1 容器 8 page」压测**（CAP-01），以 P95 执行时延、OOM 率、Chrome 崩溃率为验收口径；超过再增 VM。

## 5. 对 v1.56 的影响

- `CAP-03`：Worker 池化改造 = 引入 `browser.Manager`（1 进程多 page），`Pool` 改为「page 池」而不是「Chrome 池」；`Page.Close()` 语义改为「关闭 page 而非关浏览器」。
- `CAP-04`：`ScalingService` provider 化 + VDI VM 编排；容器内并发由 `worker.max_concurrency` 控制，VM 内容器数固定为 1（可配）。
- `VIEW-02` 中转带宽：screencast 单路 ~2–8 Mbps（视分辨率/帧率），一 VM 多容器会线性叠加出向带宽，是一 VM 一容器之外的另一条支撑论据。
