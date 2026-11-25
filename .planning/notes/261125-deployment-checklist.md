---
created: 2026-11-25
topic: v1.56 + W6 生产部署检查单（10.62.10.39 后端 / 10.62.10.34 worker）
status: deployed（后端 v1.56.2 / worker W6 已上线）
---

# 部署检查单（2026-11-25 实际执行记录）

## 1. 拓扑（实测确认）

| 角色 | 主机 | 形态 | 备注 |
|---|---|---|---|
| 后端 | **10.62.10.39** | systemd `ruoyi-backend.service`，嵌入式单二进制 | 路径 `/app/szh/ruoyi-backend-embedded-linux`，EnvironmentFile `/etc/ruoyi/secrets.env` |
| Worker | **10.62.10.34** | Docker 容器 `rpa-worker` | 端口 9001→8080（健康）、9222（CDP） |
| PG | 10.62.10.34 | 容器 `postgresql`，库 `ruoyi` | 后端与 worker 共用 |
| Redis | 10.62.10.34:6379 | 外部，需密码 | 密码经容器 env 注入，不落盘仓库 |

> ⚠️ 旧配置漂移已修正：原 `backend_url` 指向 `10.62.10.33:9000`（不可达），现为 `10.62.10.39:9000`。

## 2. 本次部署内容

### 2.1 后端（39）— 去掉 /xingran 前缀
- **根因**：39 上旧二进制内嵌的是 `baseURL:"/xingran/api/v1"` 的旧前端构建，而仓库 `.env.production` 早已是 `/api/v1`。
- **修复**：用当前 HEAD 重新构建 embedded Linux 后端（`-tags=embed`），前端产物根路径构建，二进制内 `baseURL:"/api/v1"`。
- **关键坑**：`go:embed all:xingran-react-frontend/dist` 是**相对包目录**解析的 → 必须把 dist 放到 `internal/server/xingran-react-frontend/dist`，不是仓库根。`build-linux.bat` 本来就是这么做的（已核实）。
- **上线版本**：v1.56.1（去前缀）→ v1.56.2（含 W6 调度前端页）。
- **验证**：`/health` 返回 `v1.56.2`；首页资产路径 `/assets/index-*.js`（无前缀）；`POST /api/v1/system/auth/login` 返回业务校验（验证码）而非 404；`/xingran/*` 兼容层仍 200 平滑过渡。
- **回滚点**：`/app/szh/ruoyi-backend-embedded-linux.bak-v1.55-11250533`。

### 2.2 Worker（34）— v1.56 + W6 录制能力
- **镜像**：`rpa-worker:v1.56`（本地构建，修复了 alpine 源 404 与 gdk-pixbuf trigger 问题）。
- **W6 二进制**：本地交叉编译 `linux/amd64`，以只读卷挂载到容器 `/app/rpa-worker` 覆盖镜像内旧二进制（绕开 34 出网慢，秒级生效且可秒回滚）。
- **配置**：`max_concurrency: 8`（157 基准）、`backend_url: http://10.62.10.39:9000/api/v1`、`max_pages` 已废弃并告警。
- **资源**：`shm 1gb / memory 6G / cpus 4`（157 承载模型基准）。
- **验证**：`/health` healthy；日志 `1 Chrome + N page 模型`、`liveview hub`、`recording hub (max_sessions 2)`、订阅 `recording:session:*` / `execution:liveview:*` / `execution:cancel:*` / `worker:scale:*`；PG 内 `worker-930e920b` online、`max_concurrency=8`。
- **回滚点**：`rpa-worker-v156-prev`（停用保留）、`rpa-worker-old-v1.55`（已删）。

## 3. 部署命令（可复用）

### 3.1 后端（39）
```bash
# 本地构建（务必先 npm run build 出根路径 dist）
scripts\build-linux.bat          # 默认 base=/（2026-11-25 起）
# 上传
scp out/xingran-backend-embedded-linux root@10.62.10.39:/tmp/
# 远端替换 + 重启（systemd 系统级服务）
install -m 755 /tmp/xingran-backend-embedded-linux /app/szh/ruoyi-backend-embedded-linux
chown ninedrunk:ninedrunk /app/szh/ruoyi-backend-embedded-linux
systemctl restart ruoyi-backend   # 需 root
```
> 若无 root：`kill $(pgrep -f ruoyi-backend-embedded-linux$)`，systemd `Restart=always` 会自动拉起新二进制。

### 3.2 Worker（34）
```bash
cd rpa-worker && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o ../out/rpa-worker-linux ./cmd/main.go
scp ../out/rpa-worker-linux root@10.62.10.34:/opt/rpa-worker-new/
# 重启容器（挂载新二进制）
docker stop rpa-worker && docker rename rpa-worker rpa-worker-prev
docker run -d --name rpa-worker --restart unless-stopped \
  -p 9001:8080 -p 9222:9222 \
  -v /opt/rpa-worker-new/configs/config.yaml:/app/configs/config.yaml:ro \
  -v /opt/rpa-worker-new/rpa-worker-linux:/app/rpa-worker:ro \
  -v rpa-chrome-cache:/home/chrome/.cache \
  -v rpa-chrome-downloads:/home/chrome/Downloads \
  -e REDIS_ADDR=10.62.10.34:6379 -e REDIS_PASSWORD=*** -e REDIS_DB=0 \
  -e HEADLESS=true --shm-size=1gb --memory=6g --cpus=4 rpa-worker:v1.56
```

## 4. 未清偿项（UAT，不伪造）

| # | 项 | 阻塞 |
|---|---|---|
| 1 | 批量报告真值在生产 PG 验证（migration 257） | 需确认 39 后端启动时 migration 已执行 |
| 2 | 扩缩容配置重启后仍生效 | 同上 |
| 3 | 调度在真实 Worker 集群到点触发 | 可测：本机单 worker 已就绪 |
| 4 | 调度触发任务端到端终态 | 需真实任务 |
| 5 | ~~`ops:rpa:schedule:*` 角色授权实测~~ **已通过（2026-11-25）**：给「普通用户」角色绑 7 条菜单链，ninedrunk 重登实测菜单可见/列表真读/新建落库/编辑落库/删除软删，四项 CRUD 全过。遗留：created_by 未回填 JWT 用户 ID（下轮修） |
| 6 | ~~多实例调度重复触发收敛~~ **不适用（用户确认后端单实例）** |
| 7 | vdi 审计测试 | 已闭环（d4147207） |
| 8 | e2e exit 127（bash 路径） | 环境问题 |
| 9 | 调度前端页 | **W6 已交付，本次已部署** |
| 10 | FLEET-01~03 真机 VDI 容量矩阵 | 用户暂缓 VDI |
| 11 | LIVEVIEW L1 端到端真机链路 | 可测：worker 已订阅 liveview 通道 |

## 4.5 LIVEVIEW L1 端到端真机验证（2026-11-25，部分通过）

**通过项**（真机证据）：
- 实时画面页对非超管可见，HUD/叠加层/接管按钮/copilot 下拉/未接管遮罩全部渲染
- 可观看执行列表真实数据（含新触发执行）
- 前端-后端-worker 的 watch 指令全链路（worker 日志实证收到）
- 僵尸执行（3 月堆积）正确拒绝「执行不在本 Worker 运行」
- copilot gate 状态机工作（gate waiting 日志）
- **调度连续 50 次真实执行 success**（平均 476ms/最长 1411ms，baidu 导航）

**新遗留 L6（重要）**：页面导航期间 Page.startScreencast 启动超时（context deadline exceeded）——任务执行中页面频繁跳转时 CDP target 不稳定，screencast 难以抓到稳定入口。155 本机静态 fixture 测试 fps 正常，此为真实动态页面场景的边缘 case。修复方向：startScreencast 前等待 page load 稳定 + 启动重试。

**新观察 L7**：11 步任务「页面导航与滚动测试」多轮卡在第 1 步（1 步任务全部 success），需排查 11 步脚本首动作兼容性。


## 5. 关键约束备忘

- **39 是 systemd 系统级服务**，只允许 `systemctl restart ruoyi-backend`；不要创建用户级服务。
- **单实例后端** → 调度多实例互斥问题（§6 #6）当前不适用。
- **worker 二进制挂载方案**：镜像只提供 Chromium 运行时，业务二进制走卷挂载，升级只需换文件 + 重启容器。
- **`go:embed` 路径语义**：相对包目录，不是仓库根。
- **`build-linux.bat` 默认 base=/**（2026-11-25 起），`/xingran/` 为兼容参数。
