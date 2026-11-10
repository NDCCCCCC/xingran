---
doc_type: v1.56 里程碑草案（待产品/架构拍板后转正式立项）
created: 2026-11-10
created_by: quick 261110-161（用户裁定 B-2 / B-3 按推荐立里程碑）
status: draft / 未立项 / 不含任何代码改动
scope: B 类中型改动 4 项（B-2 路线 B / B-3 OID / B-1 前端补齐 / B-4 批量自动回滚〔可选〕）
---

# v1.56 里程碑草案 · B 类中型改动

> 本草案是 **quick 261110-161 的交付物之一**，用于用户在「交互式决策页」拍板后
> 承接 4 项需 migration / 多文件 / 前端表单 / 真机取证的中型改动。
> **尚未立项**：未改 REQUIREMENTS / ROADMAP / STATE 的里程碑字段，未跑 `/gsd-new-milestone`。
> 逐项决策点来自 `.planning/quick/261110-0ua-b-c-gaps-7-b-c/261110-0ua-DECISION-PACK.md`
> 的 2 个只读 subagent 全仓代码盘点（`文件:行号` 证据见该文档）。

## 0. 与已交付项的关系（quick 261110-161）

| 项 | 本轮已做 | 留给 v1.56 |
|---|---|---|
| B-1 标记已读 | 后端最小闭环：emit JsonResult 补 noticeId、logs 带 IsRead、`/logs/read` 端点复用 `sys_notice_read`、死表与路由注册表同步 | **前端契约对齐**（`ADDriftTriggerLog` 字段与后端 `DriftAlertTriggerLog` 完全错位）+ 已读列/按钮/未读筛选 |
| B-2 路线 B | 无 | 全部（增列 + provider 化 + HMAC 加签 + migration + 前端表单） |
| B-3 OID | 无 | 全部（落库 + 匹配键改造 + 签名扩散） |
| B-4 回滚 | 诚实性短板（before_value 边界声明 + 警示文案去误导） | 自动回滚本体（若产品坚持；需真机逐厂商逆命令取证） |
| B-5/B-6/B-7 | 已收口（改注 / 扩面断言 / 403 透传） | 无 |

## 1. 候选 Phase 与 REQ 骨架（编号自 152 续编，v1.55 止于 151）

| Phase | 主题 | REQ 草案 | 关键决策点（立项前必须拍板） |
|---|---|---|---|
| 152 | B-1 前端契约对齐 + 告警已读 UI | BALRT-01~03 | ① 页面语义确认为「按用户已读」（后端已按此实现）② 前端 `ADDriftTriggerLog` 改为后端 `DriftAlertTriggerLog` 真实字段（id/title/method/operTime/jsonResult/status/noticeId/isRead）③ 删除前端未实现的 startTime/endTime/status 筛选与 `webhookTemplate` 配置项，或补后端 ④ 已读列/标记按钮/未读筛选的交互与权限门控 |
| 153 | B-2 企微/钉钉路线 B | BWEBHOOK-01~05 | ① 平台范围（企微/钉钉/飞书/可扩展 provider）② 配置落点（推荐 `sys_api_notification_config` 增 Platform/SignSecret/SignMode/MsgType 列，Platform 空 = 路线 A 零回归）③ 消息体渲染责任（整段 JSON 模板需结构化替换+转义 vs 只填 markdown text 由代码包 msgtype 外壳）④ 加签每次重发重算、timestamp 秒/毫秒 ⑤ `SendWithDefaultConfig` 同类型多平台区分 ⑥ sign_secret 列入 mask/RecordWithBody 关键词 |
| 154 | B-3 `sys_object_id` 落库 + 匹配键改造 | BOID-01~06 | ① 落哪张表（`sys_network_device` / 同时 `sys_device_discovery_result`）、存量回填策略 ② 采集链补齐（修 `ScanDevice` 透传 vs `snmpProbe` 增一次 Get）③ **核心**：匹配键 =「SysObjectID 精确/前缀优先 + Model 子串兜底」（推荐）/ 只落库不改匹配 / 只用 OID ④ 规则仍留 Go 常量（不落规则库）⑤ `Evidence` 空串=未取证口径不受 OID 命中影响 ⑥ 存量无 OID 行 Model 兜底零回归 |
| 155 | B-4 批量物理参数自动回滚（**可选**） | BROLLBACK-01~04 | ① 范围（只物理批量 / 所有批量写纳入 PORTW-04）② 逆命令可得性（三家 `undo speed`/`speed auto`、MTU 默认值、`speedMode=auto` 无单一逆命令）③ 原值来源（扩 before_state/audit vs 读 running-config 解析）④ 失败判定（fail-fast 后回滚整批已成功 / 只回滚失败端口）⑤ 回滚自身失败状态与 operlog/审批口径 ⑥ **必须先有真机逐厂商逆命令取证 + 失败注入测试**，否则「假回滚成功」比显式警示更危险 |

## 2. 全局硬约束（继承 CLAUDE.md / AGENTS.md）

- Handler-Service、`response.Success()/Error()`、写操作 success path 在 `response.Success` 前 `operlog.Record`；敏感端点 `RecordWithBody`。
- SQLite + PG 双方言；migration 双方言幂等、`database.go` 两处注册（PG + sqlite 分支，沿用 238 先例）、`registry.go` 234+ 不进 ActiveMigrations。
- 权限三方逐字一致（`pkg/permission/config.go` + `src/constants/permissionCodes.ts` + migration seed）、`check-permission-codes.mjs` 双向零漂移、新增端点重生成 `scripts/route-registry.json` 并跑 `check-route-registry.mjs`。
- 零新依赖、构建产物只落 `out/`、Excel 模板动态生成、测试就近 `*_test.go` / 分层 `tests/`。
- 回归基线：`go build ./...` + `go test ./...` + `tsc` + `lint:ci` + `vitest` 全绿；含 forbidigo 与 B-6 扩面静态断言（quick 261110-161 新增）。
- **C 类 79 条人工验证跟踪表保持 0 勾选**；不写生产数据、不切生产认证模式、不伪造 UAT 闭环。

## 3. 立项前必须补齐的输入

| # | 输入 | 来源 |
|---|---|---|
| 1 | B-2 平台范围与配置落点最终裁定 | 产品 |
| 2 | B-3 匹配键是否从 Model 切到 SysObjectID、规则是否仍留 Go 常量 | 架构 |
| 3 | B-4 是否真的要做自动回滚（否则维持本轮「诚实性短板」收口即可） | 产品 + 现场真机取证可行性 |
| 4 | B-1 前端筛选/配置项是「删前端」还是「补后端」 | 产品 |
| 5 | 新 migration 编号（当前最大 238）与窗口 | DBA / 发布负责人 |

> 上述任一未定 → 对应 Phase 不进 `/gsd-plan-phase`；草案保持 draft，不伪造立项完成。
