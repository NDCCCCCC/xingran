# Phase 93: config_backup 三处 TODO 闭环 (P2) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-05
**Phase:** 93-config_backup 三处 TODO 闭环
**Areas discussed:** Restore 实现策略（3 批）、恢复安全护栏（3 批，含异步化扩展）、压缩/解压细节、失败场景与测试、延伸区域（权限/文档/顺手修复/任务表）

---

## Restore 实现策略

| Option | Description | Selected |
|--------|-------------|----------|
| 按代码现实校准 | 恢复 = 备份配置下发到设备；93-02 措辞按 Phase 92 D-06 先例修订 | ✓ |
| 按 ROADMAP 字面 | 数据库语义恢复（校验 schema/批量 upsert/失效缓存）——代码中无此对象，无法自洽 | |

| Option | Description | Selected |
|--------|-------------|----------|
| DeviceExecutor.RestoreConfig | device 包新增厂商感知方法，与 GetConfig 对称，内部 SendConfigs，复用连接池/重试 | ✓ |
| 复用 portwrite wrapper | v1.19 已验证但端口写语义绑定，跨包复用需抽象 | |
| ExecuteMultipleOnDevice | exec 模式逐条，无配置模式语义，厂商差异需自行拼接 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 基础清洗后下发 | 滤空行/注释行（!/#），规则放 device 包内部 | ✓ |
| 原样逐行下发 | 真实设备上 ！ 和空行可能报错/噪声 | |
| Claude discretion | 清洗策略不锁定 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 仅限同设备 | backup.DeviceID == req.DeviceID 校验，防跨设备误推 | ✓ |
| 允许跨设备 | 支持设备更换迁移场景，风险操作者自担 | |
| 同设备 + force 逃生门 | 折中但增加护栏绕过风险 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 独立超时常量 | pkg/constants/timeouts.go 新增（Phase 90 惯例） | ✓ |
| 复用 CommandExecTimeout | 300s 单命令语义，批量下发不够 | |
| Claude discretion | 值与位置不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 硬编码 vendor map | 进入/退出配置模式命令硬编码（v1.19 W1 先例） | ✓ |
| 纯 scrapligo 零模板 | 仅靠 priv 提升，厂商 CLI 差异靠失败兼容 | |
| Claude discretion | 不锁实现形态 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 结构化 RestoreResult | 下发行数/耗时/恢复后 hash，handler 响应扩展 | ✓ |
| 维持 error-only | handler 零改动但高危操作无细节反馈 | |
| Claude discretion | 返回形态不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 同设备互斥 | 同时只允许一个 restore，交叉下发比慢更有害 | ✓ |
| 不互斥靠排队 | 命令流可能交错 | |
| Claude discretion | 锁形态/粒度不锁是否需要 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 插入恢复记录 | 版本链可追溯"何时恢复过、恢复到哪个版本" | ✓ |
| 仅 operlog | 版本链上下文断裂 | |
| Claude discretion | 记录形态不锁 | |

**User's choice:** 全部按推荐项锁定（D-01..D-09）
**Notes:** 用户在第 4 问后主动选择"More questions"三轮，追加超时/vendor map/返回形态/互斥/版本链五问。

---

## 恢复安全护栏

| Option | Description | Selected |
|--------|-------------|----------|
| 恢复前自动备份 | 复用现有备份链路，高危操作变可回退 | ✓ |
| 不自动备份 | 靠操作规范，误操作无退路 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 警告不失败 | hashMatched=false + 警告日志，restore 整体成功（真实设备回读一致几乎不可能） | ✓ |
| 严格失败 | 大量误报，运维会习惯性忽略 | |
| 不回读 | 端到端验收落不了地 | |

| Option | Description | Selected |
|--------|-------------|----------|
| fail-fast 即停 | 失败行即停，记进度，可从恢复前备份回退（v1.19 先例） | ✓ |
| 尽力下发汇总报错 | 关键行失败后继续可能连锁错误 | |
| Claude discretion | 结合 scrapligo 行为定 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 复用 OperTypeUpdate | 25 常量锁值防线零改动 | ✓ |
| 新增 OperTypeRestore=25 | 需同步改锁值测试 + CLAUDE.md 表 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 中止恢复 | 无法备份 = 无回退退路，安全优先 | ✓ |
| 警告后继续 | 可用性优先但裸奔 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| **异步任务**（用户选择，超出推荐的同步方案） | 任务表 + 状态机 + 轮询，彻底解决超时；确认为 HOW 范畴非 scope creep | ✓ |
| 同步阻塞 | 接受大配置超时为已知限制 | |
| 同步 + 预留接口 | YAGNI 风险 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 新建任务表 | 任务生命周期与备份版本链职责分离 | ✓ |
| 复用备份表加字段 | 少一张表但语义混杂 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 前端轮询 | 现有页面同款模式，实现最简 | ✓ |
| WebSocket 推送 | 实时但两端复杂度 + 断线重连 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 返回 taskId | 校验+建任务+立即返回；新增任务查询端点 | ✓ |
| 同步等 + 超时转后台 | 语义混乱 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| handler 发起时记 operlog | 符合 CLAUDE.md 约定零特殊处理，结果只在任务表 | ✓ |
| 发起+完成双条目 | 审计链完整但统计口径需说明 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 后端 + 最小前端 | 备份页按钮异步交互 + Modal 进度/结果，不做独立任务页 | ✓ |
| 仅后端前端延后 | API 无消费，验收只能靠测试 | |
| 完整任务管理页 | 超出 scope | |

| Option | Description | Selected |
|--------|-------------|----------|
| 不做清理 | 与备份记录同等保留，低频量级小 | ✓ |
| 定时清理 cron | 超出最小 scope | |

| Option | Description | Selected |
|--------|-------------|----------|
| 调度器排队 | 同设备互斥 + DeviceExecutor 队列自然限流 | ✓ |
| 全局上限 | 防挤占连接池但过度设限 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 重新发起代替重试 | 每次重新走完整流程含恢复前备份，幂等安全 | ✓ |
| 任务级重试 | 与重新发起收益重叠 | |

**User's choice:** D-10..D-22 + D-34；**异步任务化是用户在推荐同步方案之外明确选择的实现形态**
**Notes:** 异步化确认属 HOW 范畴（非新能力 scope creep），随之展开任务表/轮询/taskId/operlog/前端范围/并发/重试 7 个子决策。

---

## 压缩/解压细节

| Option | Description | Selected |
|--------|-------------|----------|
| .gz 后缀 + 双检查 | .conf.gz 命名，Compressed 标志 + 后缀双触发，文件系统可直接识别 | ✓ |
| 仅标志位 | 文件名不变，运维排查困惑 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 原始大小 | BackupSize = len(config)，口径统一 | ✓ |
| 压缩后大小 | 与 DB 路径口径分裂 | |
| 双字段 | 改表结构收益低 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 统一压缩 | createNewAutoBackup 复用同一 helper，消除状态混杂 | ✓ |
| 严格 scope 不动 | D-05 最小变更但混杂长期存在 | |
| 本期不动 + 登记 | 折中 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 默认级别 | gzip DefaultCompression，配置文本重复度高已有可观压缩比 | ✓ |
| BestSpeed | 批量并发 5 路 CPU 更低但压缩率降 | |
| Claude discretion | 不锁 | |

**User's choice:** D-23..D-26 全按推荐
**Notes:** 存量非压缩备份天然兼容（双检查都不过→原样返回），无需迁移。

---

## 失败场景与测试

| Option | Description | Selected |
|--------|-------------|----------|
| 93_NN 新文件 | REQUIREMENTS _78_NN 是笔误；不追加 79_06（职责分离） | ✓ |
| 追加 79_06 文件 | 集中但混杂 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 锁场景清单，手法不锁 | ①写失败 ②损坏 gzip ③下发中断留痕 ④DB 回滚；手法对照 79_06 | ✓ |
| 连手法一起锁 | 约束过死，CI 分歧风险 | |
| Claude discretion | 场景可能被弱化 | |

| Option | Description | Selected |
|--------|-------------|----------|
| FileTransport e2e | 零 SSH 全链路（79_06 + v1.19 TestE2E 先例），可进 CI | ✓ |
| 仅 mock 单测 | SendConfigs/priv 真实行为未验证 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 自动化测试落地 | 备份→恢复→hash 一致断言链（Phase 92 D-09 先例） | ✓ |
| 自动化 + 真机 UAT | 真机高危写风险大需现场配合 | |
| Claude discretion | 不锁 | |

**User's choice:** D-27..D-30 全按推荐
**Notes:** 真机 UAT 登记 deferred（可仿 v1.18/19 site-visit 模式后续启动）。

---

## 延伸区域

| Option | Description | Selected |
|--------|-------------|----------|
| 复用现有组权限 | 任务端点挂 /backups 组，继承 4 权限点零迁移 | ✓ |
| 新建专用权限点 | 粒度细但成本高 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 加 Convention 段 | 锁 gzip helper 单一路径 / RestoreConfig 唯一入口 / 异步模式强制（Phase 90/92 惯例） | ✓ |
| 不加 | 后续代码无权威路径可循 | |
| Claude discretion | 不锁 | |

| Option | Description | Selected |
|--------|-------------|----------|
| 两处都修 | REQUIREMENTS 笔误 + status 排序列删除，各自独立原子 commit（Phase 91 附带修复先例） | ✓ |
| 修笔误 + 登记 status | 保守但留 500 触发点 | |
| 都只登记 | 严守最小 scope | |

| Option | Description | Selected |
|--------|-------------|----------|
| 不支持取消 | 四态状态机；scrapli 下发无法安全中断 | ✓ |
| pending 可取消 | 多状态转移 + 端点 + 按钮 | |

**User's choice:** D-31..D-34 全按推荐
**Notes:** 任务表设计惯例（string 枚举 / UUID BeforeCreate hook / migration 双注册）直接锁入 CONTEXT，不另行提问。

---

## Claude's Discretion

- RestoreConfig 超时常量具体值；vendor map 具体命令；清洗规则具体过滤集
- 任务表名/字段/migration 编号/Task service-handler 文件布局；任务查询端点路由形态
- RestoreResult 与任务表 result JSON 字段集；恢复记录 BackupType 表达形态
- 失败场景具体模拟手法；93_NN 用例分组；同设备互斥实现形态；handler 响应 shape 细节

## Deferred Ideas

- 真机恢复 site-visit UAT（仿 v1.18/19 模式，owner = 现场运维同事）
- 任务管理独立页 / WS 推送 / 取消 / 任务级重试 / 清理 cron / 归档策略（量级增长再评估）
- RestoreConfig dry-run 预览模式（v1.20 FUTURE-06 同源；异步架构留了 pending 态自然扩展点）
- 恢复操作 Redis 缓存失效联动（ConfigBackup 不在业务缓存体系，无需联动）
