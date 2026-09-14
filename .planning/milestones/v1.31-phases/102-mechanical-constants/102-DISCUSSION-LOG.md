# Phase 102: 机械常量化（缓存键 / 状态 / 分页） - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-07
**Phase:** 102-mechanical-constants
**Areas discussed:** 键注册位置, 回归守护强度, status 补全范围, 分页收敛形态

---

## 键注册位置

### Q1: 业务模块 ~35 处内联 cache key 注册到哪里？

| Option | Description | Selected |
|--------|-------------|----------|
| 全集中 cache_keys.go | 跟随 CLAUDE.md 既有声明（唯一真相源），10 模块键与 user/role/menu/dept/post 键族同文件；代价：文件膨胀到 ~600+ 行 | ✓ |
| 各模块包内注册表 | 各模块包内建 key 常量文件，键与使用处同文件族；代价：单一真相源声明被稀释 | |
| 核心集中+边缘包内 | 高频核心模块进 cache_keys.go，边缘一次性键留包内；代价：需划定判断标准 | |

**User's choice:** 全集中 cache_keys.go（D-102-1）
**Notes:** 无附加说明。

### Q2: captcha 12 处键注册到哪里？

| Option | Description | Selected |
|--------|-------------|----------|
| pkg/constants/cache.go | 跟随 CaptchaVerifiedKeyFormat 既有先例（captcha.go:385 在用），同族聚齐；core 不反向 import services | ✓ |
| internal/core 包内 | 键与使用处同包，但与既有先例分居两地 | |
| 也进 cache_keys.go | 严格字面执行唯一真相源；代价：core → services/system 反向 import | |

**User's choice:** pkg/constants/cache.go（D-102-2）
**Notes:** 无附加说明。

### Q3: 注册进 cache_keys.go 的键用什么形态？

| Option | Description | Selected |
|--------|-------------|----------|
| 前缀常量+helper | CacheKeyXxx + GetXxxKey() helper（同 GetDictDataByTypeKey 模式，CLAUDE.md 明文约定）；调用处 fmt.Sprintf 改 helper | ✓ |
| 格式常量直替 | Sprintf 格式常量只换字面量，diff 最小但同文件两种风格并存 | |
| Claude 决定 | CONTEXT 只锁位置不锁形态 | |

**User's choice:** 前缀常量+helper（D-102-3）
**Notes:** 无附加说明。

### Q4: 失效 pattern 怎么具名化？

| Option | Description | Selected |
|--------|-------------|----------|
| 前缀派生 | BuildPattern / 前缀+":*"，不新增独立 pattern 常量，键与 pattern 单一来源防漂移 | ✓ |
| 独立 pattern 常量 | 每个 pattern 独立常量与前缀并列，直观但同前缀存两份有漂移风险 | |

**User's choice:** 前缀派生（D-102-4）
**Notes:** 无附加说明。

---

## 回归守护强度

### Q1: 缓存键常量化的回归守护用什么强度？

| Option | Description | Selected |
|--------|-------------|----------|
| 等价测试+硬失败扫描 | ①键值等价测试（含 TTL 不变）②invariants 风格内联键字面量扫描硬失败+白名单 | ✓ |
| 只等价测试 | 无扫描守护，新增内联键可能回潮 | |
| 等价测试+警告档扫描 | 扫描先警告档 Phase 103 升硬；本相收敛面已清零，警告档意义不大 | |

**User's choice:** 等价测试+硬失败扫描（D-102-5）
**Notes:** 无附加说明。

### Q2: STATUS-01 的「grep 无残留」落成什么形态？

| Option | Description | Selected |
|--------|-------------|----------|
| AST 使用点扫描 | status_constants_test.go 同文件扩展：业务代码 status 数字字面量硬失败 + geocoding 白名单表 | ✓ |
| 字符串 grep 测试 | 实现最简单但误报高、白名单膨胀 | |
| 仅验证时人工 grep | 无常驻守护，与 D-02 回归纪律不符 | |

**User's choice:** AST 使用点扫描（D-102-6）
**Notes:** 无附加说明。STATUS 扫描口径按 SC-2 自然口径取全后端（audit 时点仅剩 12 处 + geocoding 白名单），由 Claude 说明后无需单独设问。

### Q3: 缓存键硬失败扫描的覆盖面怎么划？

| Option | Description | Selected |
|--------|-------------|----------|
| 窄扫收敛面 | 本相收敛的 captcha 2 文件 + CACHE-02 10 模块；Phase 103 由 CONV-04 自行扩口 | ✓ |
| 宽扫全后端 | 本相暴露所有残余（含 Phase 103 待迁位点），需临时豁免表 → 103 再收 | |

**User's choice:** 窄扫收敛面（D-102-7）
**Notes:** 无附加说明。

---

## status 补全范围

### Q1: 若逐处核对发现某 model 缺常量，新增范围怎么定？

| Option | Description | Selected |
|--------|-------------|----------|
| 只加用到的值 | 机械相最小 diff，与行为等价目标最一致；后续用到再补 | ✓ |
| 补全完整状态族 | 一次到位，但需逐值核实语义，超出机械相最小范围 | |

**User's choice:** 只加用到的值（D-102-8）
**Notes:** 讨论中澄清：12 处大多可直接引用既有常量（JobLogStatus*/JobStatusNormal/WorkOrderStatus*/DutyStatusNormal 均已存在），真正需新增的 model 预计 0~2 个。

---

## 分页收敛形态

### Q1: 唯一调用方迁移后 cap 保持 100 还是接受 pkg 默认 200？

| Option | Description | Selected |
|--------|-------------|----------|
| 严格等价 cap=100 | NormalizePaginationWithMax + MaxListPageSize，符合 phase goal「全部行为等价」，差异点注释自证 | ✓ |
| 默认 cap=200 | 更简洁但 pageSize 101~200 请求从 clamp 变放行，违反行为等价 | |

**User's choice:** 严格等价 cap=100（D-102-9）
**Notes:** 无附加说明。utils/pagination.go 整文件删除由 Phase 99 D-03-3 长程标准继承锁定，未单独设问。

---

## Claude's Discretion

- captcha_background 池键族的常量切分方式（整格式 vs 前缀+后缀拼接）
- 等价测试文件组织与命名、两个扫描守护的测试文件命名与放置包
- 12 处 status 位点 → 既有常量的逐处映射
- captcha 内联 TTL 字面量不在本相范围（键不是 TTL；TTL 值保持逐处等价并锁定）

## Deferred Ideas

- captcha 内联 TTL 字面量顺带常量化（failKey 1*time.Hour 等）——Timeout 约定范畴但非台账项，另立时机
- utils.BuildListResponse 的 page/pageSize 响应键名与前端 PageData current 约定的历史分叉——契约治理另立 phase
