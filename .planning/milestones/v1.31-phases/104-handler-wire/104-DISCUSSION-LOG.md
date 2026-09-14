# Phase 104: handler层架构收敛（wire契约+样板去重）- Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-08
**Phase:** 104-handler-wire
**Areas discussed:** wire契约方向 · operations样板收敛机制 · monitor双日志去重形态 · 契约回归守护

---

## wire契约方向

| Option | Description | Selected |
|--------|-------------|----------|
| A: HTTP-as-code（pkg现状） | code=HTTP status（400/500），BusinessError的bizCode塞进Data.bizCode。简单但语义错误：成功code=0≠HTTP status，失败code=HTTP status是双重hack | |
| B: pkg改apperrors业务码 | pkg helper内部改走apperrors（code=业务码，HTTPStatus派生），保留BusinessError平行类型与toAppError特判 | |
| C: 重写统一权威（最佳实践） | 单一错误体系：apperrors.AppError唯一载体；BusinessError迁后删除；code=业务码/HTTPStatus语义派生；6包全切 | ✓ |

**User's choice:** C — 重写统一权威（最佳实践）
**Notes:** 用户要求先分析最佳实践（不考虑工程量）。分析结论：C方案有硬技术依据——apperrors已原生覆盖BusinessError全部能力（NewWithHTTPStatus/WrapWithHTTPStatus）；BusinessError是与errors生态割裂的孤儿类型；A方案压扁业务语义堵死前端按码分支能力；B方案双错误体系并存。确认方向后，选推荐方案。

---

## wire子项：binding错误message策略

| Option | Description | Selected |
|--------|-------------|----------|
| 透传err.Error()到message | binding错误透传gin validator输出（含字段名，如「user_name: 不能为空」） | ✓ |
| 泛泛「参数错误」 | 不透传具体字段信息 | |

**User's choice:** 透传err.Error()
**Notes:** UX好；gin validator输出无服务器内部信息；现状泛泛「参数错误」四字让用户无法定位字段。

---

## wire子项：BusinessError类型删除时机

| Option | Description | Selected |
|--------|-------------|----------|
| 同commit删除 | 统一helper落地+3构造点迁apperrors+BusinessError同commit删除 | ✓ |
| 分两步删 | 统一helper先落地+迁后留commit窗口再删 | |

**User's choice:** 同commit删除
**Notes:** D-01不留兼容壳原则。

---

## wire子项：响应data字段语义

| Option | Description | Selected |
|--------|-------------|----------|
| Data始终是业务数据（无bizCode征用） | Response.data统一是业务数据；成功/失败schema一致 | ✓ |

**User's choice:** Data始终是业务数据
**Notes:** 已知即可。

---

## operations样板收敛机制

| Option | Description | Selected |
|--------|-------------|----------|
| 泛型handler helper（T实例化路由） | 单泛型struct+路由工厂；最大压缩但Go方法泛型限制（无方法级类型参数）+service接口差异大（List参数各异/个性化方法多） | |
| 四步管线小函数族 | bind→service→err→operlog抽出小函数族；Go高阶函数是「同构流程+局部差异」的标准做法；泛型不适合本场景 | ✓ |
| Base struct嵌入 | embedding无法override个别方法（预研究已否决） | |

**User's choice:** 四步管线小函数族（Go最佳实践）
**Notes:** 用户先要求分析泛型是否符合Go最佳实践。分析结论：14个service接口不同构（List参数各异）；Go方法泛型有局限（方法返回泛型需T作为receiver类型参数）；泛型适用于路由注册工厂/中间件组合（完全同构场景），不适用于handler业务方法。确认四步管线小函数族为正确工具。

---

## operations泛型文件组织

| Option | Description | Selected |
|--------|-------------|----------|
| 单泛型文件+router层swagger | 单文件internal/api/v1/operations/handler.go；swagger注释随路由注册点在router文件 | ✓ |
| 14个泛型文件各保留 | 14个文件各embedding generic OpsHandler[T]；swagger各文件维护；改动均匀但文件数不变 | |

**User's choice:** 单泛型文件+router层swagger
**Notes:** 推荐方案。

---

## operations文件结构

| Option | Description | Selected |
|--------|-------------|----------|
| 文件不变，只改内部调用 | 14个handler文件保留；内部改调小函数；swagger和个性化方法不变；测试不动 | ✓ |

**User's choice:** 文件不变，只改内部调用
**Notes:** 推荐方案。

---

## monitor双日志去重形态

| Option | Description | Selected |
|--------|-------------|----------|
| 泛型MonitorLogHandler | MonitorLogHandler[T] struct+5同构方法；因service接口高度同构（5方法签名完全一致）泛型真正适用 | ✓ |
| 共享函数族 | 5方法抽出共享函数（ListByLogType/GetByID...） | |

**User's choice:** 泛型MonitorLogHandler
**Notes:** 推荐方案。侦察发现monitor service层高度同构（OperLogService/LoginLogService 5方法签名完全一致），泛型在monitor域真正适用。

---

## monitor去重方案细化：Clean差异

| Option | Description | Selected |
|--------|-------------|----------|
| 泛型5方法+各文件保两Clean | 泛型含5同构方法；OperLog.Clean（含同步审计+post-verify事务）和LoginLog.Clean（简单async operlog.Record）差异大各保留 | ✓ |

**User's choice:** 泛型5方法+各文件保两Clean
**Notes:** 推荐方案。OperLog.Clean有200行注释解释的同步审计+事后验证chicken-and-egg防御，与LoginLog.Clean完全不同。

---

## 契约回归守护

| Option | Description | Selected |
|--------|-------------|----------|
| httptest契约测试（全量升级） | httptest断言(HTTPStatus,code,message)三元组；4类路径各1用例；既有200+handler测试升级code断言值 | ✓ |
| 独立wire_contract_test.go | response层独立测试；既有测试继续跑只检查HTTPStatus | |

**User's choice:** httptest契约测试（全量升级）
**Notes:** 推荐方案。

---

## Claude's Discretion

以下由Claude自行决定（用户未明确指定）：

- 小函数命名（`BindAndCall`/`CreateBind`/`ListBind`等）
- 泛型`MonitorLogHandler[T]`构造方命名
- `config_restore_task_service.go`三个BusinessError构造点对应的apperrors.ErrorCode值选择（需匹配业务语义）
- `operations/handler.go`新文件位置
- `config_restore_task_service_97_03_test.go`升级后的断言写法

## Deferred Ideas

- **UnlockUser实现**（login_log_handler.go:188）：Phase 107 TODO-03占位，本相去重不动
- **前端按业务码分支**：当前api.ts只依赖`code===0`；契约已就绪（code=业务码），前端逻辑需单独迭代
- **泛型handler扩展**：若未来更多日志类handler加入，`MonitorLogHandler[T]`可复用（副产品）

---

*Discussion completed: 2026-09-08*
