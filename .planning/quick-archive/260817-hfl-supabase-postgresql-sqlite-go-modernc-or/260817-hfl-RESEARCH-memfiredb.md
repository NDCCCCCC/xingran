# Quick Task 260817-hfl: MemFire Cloud 调研（Supabase 国产替代）

**调研日期:** 2026-08-17
**调研目的:** 判断 MemFire Cloud 能否替代当前 Supabase（远程 PostgreSQL）作为开发/生产数据库，评估迁移难度与使用限制。

## 1. 产品定位

- **MemFire Cloud**（memfiredb.com）是敏博科技（NimbleX）推出的国产 BaaS，官方明确"采用开源的 Supabase"构建，兼容国内开发生态。
- 数据库内核为自研 **MemFireDB**：基于 Raft 的分布式关系数据库，**复用 PostgreSQL 原生查询层，兼容 PostgreSQL 11.2 语法/协议**。
- 功能对标 Supabase：Postgres 数据库 + 自动生成 REST API、认证授权（含微信/短信）、实时数据库（WebSocket 订阅）、对象存储、云函数、静态托管、RLS 权限。
- 支持标准 PG 客户端直连（psql / lib/pq / psycopg2 / JDBC 等），即可以只把它当作一个 PG 云数据库用。

**佐证:** [官网首页](https://memfiredb.com/) | [官方 FAQ](https://docs.memfiredb.com/docs/frequently-asked-questions/) | [掘金介绍](https://juejin.cn/post/7340076087249125410)

## 2. 使用限制（免费额度与计费）

| 项目 | 免费额度 | 佐证 |
|------|---------|------|
| 存储空间 | 512 MB | [令川博客汇总](https://lccl.cc/post/2024/%E5%88%86%E4%BA%AB%E4%B8%80%E4%BA%9B%E5%85%8D%E8%B4%B9%E7%9A%84-postgresql-%E4%BA%91%E6%95%B0%E6%8D%AE%E5%BA%93) / [CSDN](https://blog.csdn.net/qq_34640315/article/details/149962000) |
| 流量 | 1 GB/月 | 同上 |
| 计费模式 | 2024-01 起正式计费：基础套餐 + 按量付费；套餐分 免费版/入门版/基础版/专业版/定制版 | [官方公告](https://docs.memfiredb.com/docs/announcement/bulletin/bulletins/) / [计费FAQ](https://docs.memfiredb.com/docs/app/purchase/qa/) / [掘金套餐文](https://juejin.cn/post/7322518704495329306) |
| 不适用场景 | 官方明示不适合 OLAP/即席分析 | 官方 FAQ 第 10 条 |
| 账号 | 需微信扫码注册；静态托管自定义域名需实名认证 | 官网 |

**注意:** 免费额度的最新细则（连接数、并发、暂停策略）官网价格页抓取失败未能核实，需注册后在控制台确认。

## 3. 迁移难度评估（针对本项目）

本项目只用 Supabase 的 **Postgres 直连**（Go + GORM + lib/pq 协议），不使用 Supabase 的 Auth/Storage/Realtime 等 BaaS 功能。因此：

**有利面（难度低的部分）:**
- 切换本质上 = 换连接串（host/port/user/password），代码无需改动
- MemFire 提供 Go (lib/pq) 连接示例，协议级兼容
- 支持触发器、存储过程、外键、部分索引、分布式事务（官方 FAQ）
- 扩展商店提供 uuid-ossp、pgvector、pg_cron、http 等常用扩展

**风险面（需实测验证）:**

| 风险点 | 本项目依赖 | MemFire 情况 | 严重度 |
|--------|-----------|-------------|--------|
| PG 版本差距 | 项目目标 PG 18 | 仅兼容 **PG 11.2** | 🔴 高 |
| `gen_random_uuid()` | BaseModel 主键约定（CLAUDE.md 声明 PK 用 `gen_random_uuid()` 默认值） | PG13 才内置；MemFire 需启用 uuid-ossp 改用 `uuid_generate_v4()`，迁移 SQL 要改 | 🟡 中（注意：实际代码已用应用层 `uuid.New()` BeforeCreate，数据库层默认值依赖程度需确认） |
| 物化视图 | migrations 175/176 含物化视图 | 分布式内核对 MV 支持未在文档中确认 | 🟡 中 |
| advisory lock | database.go 有 advisory lock（已按 postgres 分支守卫） | PG 咨询锁在分布式内核上语义可能不同 | 🟡 中 |
| 迁移脚本 PG 方言 | 206 个 Go MigrateNNN 迁移 | 需逐个在 MemFire 上跑一遍验证 | 🔴 高（工作量） |
| 运维惯性 | 现有数据在 Supabase | 需要 pg_dump/导入迁移 + 兼容性清洗 | 🟡 中 |

**迁移工作量预估:** 连接切换 0.5 天；但 206 个迁移脚本 + 全量功能回归在 PG 11.2 兼容层上验证，预估 2-5 天，且存在个别功能无法兼容需要绕行的可能。

## 4. 与 SQLite 方案对比

| 维度 | MemFire Cloud | 本地 SQLite (glebarez/sqlite, 纯Go) |
|------|--------------|-----------------------------------|
| 解决的问题 | 网络慢（国内节点，理论上快） | 彻底无网络（本地文件，最快） |
| 成本 | 免费 512MB/1GB 流量，超额付费 | 零成本 |
| 代码改动 | 几乎零（换连接串） | 中（恢复 sqlite 分支、驱动切换、迁移守卫） |
| 兼容性风险 | PG 11.2 方言 vs 项目 PG18 特性 | SQLite 方言 vs PG 特性（同样有差异，但 PG-only 代码已有 `if d.Type == "postgres"` 守卫） |
| 数据共享/协作 | 云端，多人可连 | 单机文件 |
| 生产可用性 | 可作为生产候选（有 SLA 套餐） | 仅适合开发/演示 |

## 5. 结论与建议

- **如果痛点只是"Supabase 访问慢"且场景是本地开发** → SQLite 是最快最省的路径（依赖已在 go.mod 中）。
- **如果需要云端共享数据库、且能接受付费** → MemFire Cloud 值得试：先注册免费版，用连接串跑一次完整迁移冒烟（重点验 gen_random_uuid、物化视图、advisory lock、迁移 175/176/202-206），通过再考虑正式迁移。
- **不建议**在未做兼容性冒烟前直接把 MemFire 作为默认开发库。
