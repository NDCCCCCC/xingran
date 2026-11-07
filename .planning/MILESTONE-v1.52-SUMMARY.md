# v1.52 运维域数据权限统一收口（终态）— 里程碑总结

- **状态**：✅ SHIPPED 2026-11-07
- **PR**：#75 merged → `b56f1286`（全量代码）；本分支 `feat/v1.52-data-scope-unification` 为收档 PR
- **需求**：7 REQ（DS-01~07）全部交付
- **Phase**：130 / 131 / 132 / 133 全部完成（4/4）
- **提交**：24 commits（含 PR #75 前置 7 提交 + v1.52 四 phase 14 提交 + 审查/红灯修复 3 提交）

## 核心成果

把数据权限从「用户 / VDI 两个模块的局部实现」升级为运维域全模块的声明式统一收口：读写两端按登录人数据范围自动过滤与校验，使「中支管理员只能管本机构的工位/楼层/资产」从口头约定变为架构保证。

### Phase 130 — ScopeProvider 基建
- 新增 `pkg/datascope`：可见部门集单一真实源（All / 0 / unset 特判短路零查询、Custom / Dept / DeptChild 展开、Self 资源语义钩子、异常枚举 fail-closed、多角色 MAX = 最严档、UserID 透传）
- `DataScopePermission` 注入完整 ScopeResult 到 request context；ops 全域 13 读组 + 13 写组 = 26 处挂载（既有 5 处不动，总 31），`datascope_mount_test.go` 守卫
- `user_service.applyUserDataScope` 迁移到单一真实源

### Phase 131 — 读端点收口
- 四域 20 个读函数（含 `workstations-all` / `tree` / `dept-options` / `counts-by-floor` / `GetByID`）统一「有效机构 ∈ 可见集」
- Excel 导出两条独立路径 + 审查 P1-1 补收口的 `dept-mapping-template` 独立注入可见集
- 缓存 key 增 scope 维度（`floor:tree:s{scope}` / `floor:building:s{scope}:%s` / `BuildScopedDropdownCacheKey`），交付 `131-CACHE-INVENTORY.md`
- 403 越权矩阵 + 跨用户缓存对照实验 + PG contract 扩展

### Phase 132 — 写端点收口
- 四域 Create / Update / Delete / Batch / positions 单条越界显式 403（`NewWithHTTPStatus`，规避 `CodePermissionDenied` 默认落 401 的框架坑）、批量按可见集裁剪
- backfill 候选 / COUNT / CASE UPDATE 三层注入可见集；非 All 档 `Samples` 不逐条回显；operlog 补范围档位 / 四类计数
- Excel 导入逐行可见集校验（越界行不写入、不阻断整批、计入 `ImportResult`）
- 前端确认框「本次范围」四类文案 + FailClosed disabled + vitest 断言

### Phase 133 — 扩展实体与收尾
- asset / serverRoom / dedicatedLine / roomDevice 读写收口（dedicatedLine source/dest 双链 OR 命中、链断 fail-closed）
- workstation-device 按工位可见集收口
- vdi 调用方迁移到 ScopeProvider（签名不变、旧实现零回归、异常枚举不再放行全库）
- 删除 `pkg/middleware` 三份死实现 + 死测试；前端 DeptSidebar / DeptTree 可见集联动
- 显式豁免清单登记 + `133-UAT.md` 人工 UAT 清单
- 四实体 403 矩阵 **115 个语义断言**（读 28 / 写 21 / 批量 13 / 断链 11 / 档位边界 42）

## 三项开工输入裁定（LOCKED）
`.planning/notes/261107-v152-{self-scope,batch-import,ds05-scope}-decision.md`：Self 档 ops 域语义 / Dept-Self 批量与 Excel 导入豁免 / DS-05 三项归属链，按工程最优 + fail-closed 原则自主定案，全部在对应 phase 开工前落盘。

## 独立代码审查
`agent-8` 只读审查产出 3 P0 + 11 P1 清单（`.planning/notes/261107-v152-independent-review.md`）：

| P0 | 处置 |
|---|---|
| P0-1 迁移 235 被误嵌在 234 失败分支内 → 正常启动永不 seed | ✅ 已修（`cb04d0f1`） |
| P0-2 框架把 `CodePermissionDenied` 默认映射 401 / 裸 error 映射 500 | ✅ 132 用显式 403 规避 |
| P0-3 已合入 backfill 全库扫描 + 逐条明细回显 | ✅ 132 三层裁剪 + 非 All 档 Samples 置空 |

另修 **P1-4** vdi `dataScope > 0` 让异常枚举放行全库。

## PR #75 三项 CI 红灯修复
| 红灯 | 修复 |
|---|---|
| `prettier --check` 4 文件不合规 | 格式化 4 文件 |
| `lint:ci` antd 6 弃用 prop 3 项（`size` / `maskClosable` / `Alert.message`） | 逐一迁移 |
| `coverage-diff` 76.63% < 80% | 新增 644 行 / 27 个 `TestDiffCov_*` 厚测试，门禁 **83.37% PASS** |

推送后又修两处 CI：`DeptTree` 的 `DataNode.key` 类型不兼容（`type-check`）、`datascope_test.go` 的 staticcheck SA1012（`golangci-lint`）。最终 PR #75 全 checks pass、`mergeStateStatus: CLEAN` 后合并。

## 回归闸门（全绿）
- `go build ./...` EXIT=0
- `go test -count=1 ./internal/... ./pkg/... ./cmd/...` EXIT=0（全量）
- `go vet ./...` + `golangci-lint run ./...` **0 issues**
- 前端 `format:check` / `type-check` / `lint:ci` **ALL PASS**（route-registry 702/311/138）
- 前端 vitest **592 文件 / 4128 用例全绿**
- `npm run build` 生产构建通过（vendor-react gzip 640.80 kB ≤ 776 kB 预算）
- diff coverage 门禁 PASS
- GitHub CI：backend / frontend / build / coverage-diff / 4×frontend-coverage / coverage-merge / coverage-diff 全 pass

## 已知残余（登记，非阻断）
- `WorkstationQueryBuilder` 既有 `::uuid/::text` PG 专有 JOIN 在 SQLite 必炸（P1-8，既有技术债，新增过滤条件已用双方言 CAST）
- 聚合计数（越界行数 / N / 行号）仍为枚举侧信道（P1-5，已知接受，未来应加告警通道而非改回整批 403）
- `scripts/e2e/TestE2EAllEndpointsLogged` 在 Windows 下 bash 路径拼接报 127（环境既有）
- 显式豁免面：walls / doors / floor-plan-texts / rooms-photos / location-alias（详见 `133-UAT.md`）
- 双门控 `TEST_PG_DSN` / `XINGRAN_TEST_PG_DSN` 在 CI 默认 Skip（P1-9，skip 可观测）

## 交付物索引
- Phase 产物：`.planning/phases/{130-scope-provider,131-read-scope,132-write-scope,133-scope-final}/`
- 裁定：`.planning/notes/261107-v152-*-decision.md`
- 审查：`.planning/notes/261107-v152-independent-review.md`
- UAT：`.planning/phases/133-scope-final/133-UAT.md`
- 缓存清点：`.planning/phases/131-read-scope/131-CACHE-INVENTORY.md`
