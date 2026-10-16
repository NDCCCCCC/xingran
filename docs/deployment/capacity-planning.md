# 容量规划与硬件选型

> 适用场景：在采购硬件 / 调优数据库连接池 / 决定是否拆库拆服务前，给出量化决策依据。
> 输入：用户数、工单/月、操作日志量、启用的模块（AD/RPA/3D 等）。
> 输出：CPU / 内存 / 磁盘 / 网络 规格建议，以及单 vs 多机分界点。

---

## 1. 三档规模定义

| 档次 | 用户数 | 工单/月 | 典型场景 | 推荐架构 |
|---|---|---|---|---|
| **小型（试点）** | < 100 | < 500 | PoC、单机演示、单部门 | **单机全栈**（[single-machine-deployment.md](single-machine-deployment.md)） |
| **中型（企业部门）** | 100 - 1000 | 500 - 5000 | 单业务线、跨部门 | **单机 + 调优** 或 **前后端分离 + 独立 PG/Redis** |
| **大型（集团）** | 1000+ | 5000+ | 多业务线、多子公司 | **前后端分离 + 读写分离 + 独立缓存集群**（**不再适合单机**） |

> ⚠️ "大型"档已超出本系统单机舒适区。建议**先拆 PG（主从/读写分离）+ 拆 Redis（哨兵/集群）+ 后端多实例 + nginx 负载均衡**。

---

## 2. 模块开关影响

每个模块对资源占用有不同影响。规划前先确认实际启用清单：

| 模块 | 路径 | 对 CPU 影响 | 对内存影响 | 对磁盘影响 | 备注 |
|---|---|---|---|---|---|
| **基础 RBAC** | `internal/services/system/` | 低 | 低 | 低 | 默认必启 |
| **3D 楼宇可视化** | `internal/services/operations/` + 前端 `three.js` | **前端**高 | 前端 200-500 MB | 低 | 渲染在浏览器，服务端轻；楼层多时前端内存膨胀 |
| **AD 域同步** | `internal/services/addomain/` | 中（LDAP 查询 + SM4 加解密） | 中（账号池缓存） | 中（`sys_ad_service_accounts` 表增长） | 启用后建议至少中型规格 |
| **网络设备纳管** | `internal/device/` + `internal/services/portcollection/` | 中（Scrapli + TextFSM 解析） | 中（模板缓存） | 中（`sys_port_*` 历史表） | 设备数 > 500 时建议独立 Worker |
| **RPA 自动化** | `rpa-worker/` + Docker | **极高** | **极高**（每 Worker 1-2 GB） | 中（截图/下载文件） | 真实模式必须按 `max_workers` 加内存 |
| **VDI 桌面云** | `internal/services/vdi/` | 低 | 低 | 低 | 仅 API 转发 |
| **定时任务** | `internal/scheduler/` | 中（高频 cron） | 低 | 中（`sys_job_log`） | 高频任务多时 DB 写入放大 |
| **监控大盘 / 操作日志** | `internal/api/v1/monitor/` + `sys_oper_log` | 低 | 低 | **高**（每天 GB 级） | 必须定期归档/清理 |
| **百度地图地理编码** | `internal/services/operations/geocoding_service.go` | 低（30 min 缓存） | 低 | 低 | 仅影响导入性能 |

---

## 3. 硬件规格推荐表

### 3.1 综合选型

| 规模 | CPU | 内存 | 系统盘 | 数据盘 | 网络 | 备注 |
|---|---|---|---|---|---|---|
| **小型** | 4 核 | 16 GB | 60 GB SSD | 100 GB | 千兆 | 单机全栈 |
| **中型（无 RPA）** | 8 核 | 32 GB | 100 GB SSD | 500 GB | 千兆 | 单机 + 调优 |
| **中型（启用 RPA）** | 8 核 | 64 GB | 100 GB SSD | 500 GB | 千兆 | RPA Worker 内存为主 |
| **大型** | 16 核+ × 多实例 | 64 GB / 实例 | 200 GB SSD | 1 TB+ | 千兆 + 负载均衡 | 拆 PG/Redis |

### 3.2 数据库独立部署（中型 → 大型过渡）

当单机 CPU 持续 > 70%、内存使用 > 80%、磁盘 IO > 50 ms 时，建议拆库：

| 服务 | 独立服务器最低 | 推荐 |
|---|---|---|
| PostgreSQL | 8 核 / 32 GB / 500 GB SSD | 16 核 / 64 GB / 1 TB NVMe |
| Redis | 4 核 / 16 GB / 100 GB SSD | 8 核 / 32 GB / 200 GB SSD |
| 后端 | 4 核 / 8 GB（每实例） | 8 核 / 16 GB（多实例） |

---

## 4. PostgreSQL 容量估算

### 4.1 表空间增长估算（按"中型"档）

| 表 | 单条记录 | 增速（中型场景） | 1 年预估 |
|---|---|---|---|
| `sys_oper_log` | ~1 KB（含请求体截断） | 5000-20000 条/天 | 2-8 GB |
| `sys_job_log` | ~500 B | 1000-5000 条/天 | 200 MB - 1 GB |
| `sys_port_*`（网络设备历史） | ~300 B | 10000+ 条/天 | 1-3 GB |
| `sys_workorder` | ~2 KB | 500-2000 条/月 | < 100 MB |
| `sys_rpa_task` + `sys_rpa_task_log` | ~5 KB（含截图路径） | 100-1000 条/天 | 200 MB - 2 GB |
| `sys_upload_file` + 二进制 | ~500 KB（平均） | 与上传量挂钩 | 用户上传 + 工单附件：10-100 GB/年 |

> **关键发现**：操作日志 + 上传文件 是磁盘增长大头。生产必须配置定期归档/清理。

### 4.2 连接池调优

`configs/config.yaml` 中 `database.max_open_conns` 和 `cache.pool_size` 必须按并发量调整：

| 场景 | 并发用户 | 推荐 max_open_conns | 推荐 cache.pool_size |
|---|---|---|---|
| 小型（< 100 用户） | < 50 同时在线 | **25**（Phase 62-DBG-01 默认值） | 10 |
| 中型（100-1000） | 50-300 | **50** | 50 |
| 大型（> 1000） | 300+ | **100+** + PG 主从 | 100+ |

⚠️ **PG `max_connections` 必须 ≥ 所有应用实例 max_open_conns 之和 + 10**。默认 100，集群场景需调大到 500+。

```sql
-- 查看当前连接数
SELECT count(*) FROM pg_stat_activity;

-- 调大 max_connections（需重启 PG）
-- postgresql.conf: max_connections = 500
```

### 4.3 PG 内存配置（中型场景推荐）

`postgresql.conf`：

```ini
shared_buffers = 8GB               # 物理内存 1/4（32GB 主机）
effective_cache_size = 24GB       # 物理内存 3/4
work_mem = 64MB                   # 排序/哈希
maintenance_work_mem = 1GB        # vacuum / index build
wal_buffers = 16MB
max_connections = 200
random_page_cost = 1.1            # SSD 必调（默认 4 是 HDD）
effective_io_concurrency = 200    # SSD
```

---

## 5. Redis 容量估算

### 5.1 键空间分析

XingRan-Next 所有 key 加 `xingran:` 前缀（`internal/core/core.go:342`）。常见 key：

| Key 模式 | 内容 | 单 key 大小 | 数量级 |
|---|---|---|---|
| `xingran:user:all` | 全量用户列表（JSON） | 50-500 KB | 1 |
| `xingran:user:{id}` | 单个用户 | 1-5 KB | 用户数 |
| `xingran:role:all` | 角色 + 权限 | 100 KB - 1 MB | 1 |
| `xingran:dict:data:{type}` | 字典数据 | 1-50 KB | 字典类型数 |
| `xingran:dept:tree` | 部门树 | 50-500 KB | 1 |
| `xingran:menu:user:{userId}` | 用户菜单 | 5-50 KB | 在线用户数 |
| `xingran:menu:role:{roleId}` | 角色菜单 | 5-50 KB | 角色数 |
| `xingran:post:all` | 岗位列表 | 10-100 KB | 1 |
| `xingran:ad:account:list` | AD 账号池 | 10-50 KB | 1 |
| 业务缓存（杂） | | | 取决于模块 |

### 5.2 内存估算

**粗略公式**：基础缓存（user/role/dept/menu/dict/post）≈ **用户数 × 5 KB** + **角色数 × 10 KB** + **部门数 × 2 KB**

| 用户数 | 角色数 | 部门数 | 基础缓存 | 业务杂项 | **Redis 最小** | **推荐 maxmemory** |
|---|---|---|---|---|---|---|
| 100 | 10 | 20 | ~600 KB | 100 MB | 200 MB | 512 MB |
| 500 | 30 | 50 | ~3 MB | 200 MB | 500 MB | 1 GB |
| 1000 | 50 | 100 | ~6 MB | 500 MB | 1 GB | 2 GB |
| 5000 | 100 | 200 | ~30 MB | 2 GB | 3 GB | 4 GB |
| 10000 | 200 | 500 | ~60 MB | 5 GB | 7 GB | 8 GB |

### 5.3 Redis 配置建议

`redis.conf`（中型）：

```ini
maxmemory 4gb
maxmemory-policy allkeys-lru        # 业务缓存可容忍 LRU 淘汰
appendonly yes                       # AOF 持久化
appendfsync everysec
tcp-keepalive 60
timeout 300
```

---

## 6. 后端进程调优

### 6.1 Go runtime（GODEBUG）

`/etc/xingran/secrets.env` 或 systemd `Environment=`：

```bash
GODEBUG=gctrace=0
GOMAXPROCS=0                  # 0 = 自动 = CPU 核数
GOGC=100                      # GC 触发阈值，默认 100%
GOMEMLIMIT=24GiB              # Go 1.21+ 软内存上限（推荐设置为物理内存 75%）
```

### 6.2 后端连接池

`configs/config.yaml`：

```yaml
database:
  max_open_conns: 50
  max_idle_conns: 10           # ≈ max_open_conns 的 20%
  max_lifetime: 1800           # 30 分钟（防止 PG 端 stale）

cache:
  pool_size: 50
  max_size: 10000
  l2_writer:
    worker_count: 5            # 异步写 Redis worker
    queue_size: 5000
```

### 6.3 Gin / Swagger / Swagger UI

- 生产关闭 `server.mode: debug`（避免 gin debug 路由泄漏堆栈）
- Swagger UI 仅在内网开放或加 basic auth

---

## 7. 前端构建与缓存

### 7.1 构建产物大小

`npm run build` 产物（生产环境）：

| 资源 | 大小（gzip） |
|---|---|
| 主 JS chunk | 1-3 MB（含 React + Ant Design + Three.js） |
| CSS | 100-300 KB |
| Three.js 单独 chunk | 500 KB - 1 MB |
| ECharts chunk | 300 KB - 1 MB |
| 总 dist | **5-15 MB（gzip 前）** |

### 7.2 nginx 缓存策略

```nginx
# 静态资源：长缓存 + immutable
location ~* \.(js|css|woff2?|png|jpg|svg)$ {
    expires 365d;
    add_header Cache-Control "public, immutable";
}

# HTML：必须 no-cache（避免新版本加载不到）
location / {
    add_header Cache-Control "no-cache, no-store, must-revalidate";
    try_files $uri $uri/ /index.html;
}
```

---

## 8. 网络与带宽

### 8.1 出方向流量（中型场景）

| 出方向 | 用途 | 估算 |
|---|---|---|
| 百度地图 API | 工位/楼宇地理编码 | 1000 次/天 × 1 KB ≈ 1 MB/天 |
| AD/LDAP | 用户/组同步 | 取决于目录大小 |
| Scrapli SSH | 网络设备纳管 | 取决于命令量 |
| RPA AI 调用 | OpenAI/Claude API | 取决于任务量；可 GB/天 |

### 8.2 入方向流量

| 入方向 | 估算 |
|---|---|
| 用户浏览器 | 静态资源为主，1-10 MB/会话 |
| 上传文件 | 取决于业务；中等 10-50 GB/月 |
| Swagger UI 调用 | 几乎可忽略 |

### 8.3 公网带宽建议

- **小型**：无需求，nginx 直出 + HTTPS 即可（5 Mbps 足够）
- **中型**：10-50 Mbps（应对集中导入/导出时段）
- **大型**：100 Mbps+（含 RPA 大文件、AI 调用响应）

---

## 9. 监控指标基线（中型场景）

部署后建议接入 Prometheus + Grafana（或厂商 APM），基线指标：

| 指标 | 健康范围 | 告警阈值 |
|---|---|---|
| CPU 使用率 | < 60% | > 80% 持续 5 min |
| 内存使用率 | < 75% | > 90% |
| PG 活跃连接数 | < 70% max_connections | > 85% |
| PG 慢查询（> 1 s） | < 10/min | > 50/min |
| Redis 内存 | < 70% maxmemory | > 85% |
| Redis 命中率 | > 95% | < 90% |
| 后端 goroutine 数 | < 10000 | > 20000 |
| 接口 P99 延迟 | < 500 ms | > 1 s |
| `sys_oper_log` 每日新增 | 视业务定 | 异常突增 |

---

## 10. 单机 vs 多机决策树

```
当前是单机？
├── 是 → 流量是否已超中型上限？
│       ├── 否 → 继续单机（优化连接池/缓存/日志清理）
│       └── 是 → 拆分 PG → 拆分 Redis → 多后端实例
└── 否（已多机） → 是否需要读写分离？
        ├── 否 → 维持现状
        └── 是 → PG 主从 + 读流量拆分
```

**触发拆机的硬指标**：

- CPU 持续 > 70%（即使 GOMAXPROCS 已用满）
- 内存压力导致 OOM 或 GOMEMLIMIT 触发 GC 抖动
- PG `pg_stat_activity` 排队 > 5
- Redis maxmemory 频繁触顶（`evicted_keys` 持续增长）
- 后端 `goroutine` 数 > 10000（说明请求堆积）

---

## 11. 容量调整 Checklist

扩容/调优前对照检查：

- [ ] PG `max_connections` ≥ 所有应用实例 max_open_conns 之和 + 10
- [ ] Redis maxmemory ≥ §5.2 估算值 × 1.5（含缓冲）
- [ ] `cache.pool_size` ≥ 后端实例数 × 50
- [ ] `l2_writer.queue_size` ≥ 后端实例数 × 5000
- [ ] nginx `worker_processes auto; worker_connections 1024;`
- [ ] 后端 `GOMEMLIMIT` 设为物理内存 75%
- [ ] 日志保留天数合理（`log.max_age: 90` 默认；磁盘紧张降至 30）
- [ ] 操作日志定期归档（建议保留 90 天在线，更老转冷存储）
- [ ] 上传文件分目录存储（按月/年），便于清理
- [ ] 启用 Gzip/Brotli 压缩 nginx 响应

---

## 12. 容量估算速查表（Copy-Paste）

| 输入 | 数值 | 来源 |
|---|---|---|
| 用户总数 | _ | 业务方 |
| 同时在线峰值 | _ | 历史 / 估计（一般 10-20% 总用户） |
| 工单/月 | _ | 业务方 |
| 操作日志/天 | _ | 估算：用户 × 50-200 |
| 网络设备数 | _ | 业务方 |
| RPA 任务/天 | _ | 业务方 |
| 上传文件 GB/月 | _ | 业务方 |
| 启用模块清单 | _ | RBAC ✓ / 3D / AD / 网络设备 / RPA / VDI |

把上述填入下方公式（每用户估算）：

```
PG 数据/年 ≈ 用户数 × 10 MB（基础）+ 工单/月 × 12 × 5 KB + 日志/天 × 365 × 1 KB
Redis 内存 ≈ 用户数 × 5 KB + 角色数 × 10 KB + 业务杂项 100-500 MB
CPU 核数 ≈ max(4, 同时在线 / 50) + (启用 RPA ? 4 : 0)
内存 GB ≈ max(16, 用户数 × 0.02) + (启用 RPA 真实模式 ? 16 : 0)
```

---

## 13. 相关文档

- [single-machine-deployment.md](single-machine-deployment.md) — 小/中型单机部署
- [deployment.md](deployment.md) — 生产 systemd 部署
- [docker-compose.md](docker-compose.md) — Docker Compose 一键编排（含资源限制示例）
- [secret-management.md](secret-management.md) — 密钥管理
- [架构/数据库设计](../architecture/数据库设计.md) — 表结构
- [guides/cache_usage.md](../guides/cache_usage.md) — 缓存键约定与失效策略

---

## 14. 变更日志

| 日期 | 变更 | 作者 |
|---|---|---|
| 2026-08-15 | 初版：硬件规格、PG/Redis 容量估算、连接池调优、单 vs 多机决策 | Claude |