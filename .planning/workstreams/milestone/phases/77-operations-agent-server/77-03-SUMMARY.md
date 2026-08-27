---
plan: 77-03
phase: 77-operations-agent-server
executed: 2026-08-27
commits:
  - 4539464 (fix(quirk-77-c): normalizeHeaderTrim 注释称转小写但实现仅 TrimSpace)
  - fd15f4a (test(77-03): ImportData 剩余分支 + ReadRawRowsByName + Q-77-C 回归 Task 1)
  - 36cb5d3 (test(77-03): reference_resolver + workstation/floor/code_generator Task 2/3)
---

# 77-03 Summary — Excel 导入剩余 + reference_resolver + 卫星测试 (BLOCK-01 收口)

## 交付

### 4 个新测试文件 (44 个 TestImp77_ 函数,远超验收 ≥12)

| 文件 | 行数 | 测试数 | 覆盖目标 |
|------|------|--------|----------|
| `excel_import_rest_77_03_test.go` | ~370 | 16 | ImportData 剩余分支 + populateNewUserPasswords + 二阶段依赖引用端到端 |
| `excel_raw_rows_77_03_test.go` | ~200 | 11 | ReadRawRowsByName 全分支 + Q-77-C normalizeHeaderTrim 行为锁定 |
| `reference_resolver_77_03_test.go` | ~290 | 8 | ResolveSingleWithCondition + ResolveBatchWithDependencies 尾部 + 私有 helper 白盒 |
| `workstation_floor_code_77_03_test.go` | ~440 | 9 | workstation CRUD + 6 表 JOIN List + BatchUpdatePositions + floor 全 CRUD + code_generator + Q-77-D |

### 1 个生产文件 doc-only 修改 (Q-77-C)

- `internal/services/operations/excel_raw_rows.go` 注释修正:
  normalizeHeaderTrim 注释「转小写用于匹配」与实现（仅 TrimSpace）不符 → 只修注释对齐
  现行为,零代码行变更;新增"行为锁定 (Q-77-C, D-03)"段落说明不改 ToLower 的根因。

## Coverage 收口 (BLOCK-01 达成判据)

- **基线 73.2%(77-02 后)→ 实测 83.7%(本 plan),+10.5pp**,远超 BLOCK-01 的 ≥70.0% 线。
- `go test -count=1 -cover ./internal/services/operations/` 反复跑两次确认数字稳定。
- `go build ./...` exit 0;`go test ./internal/services/operations/` 全绿(含 77-01/77-02 无回归)。
- 新测试 `-count=3` flake 筛查通过(两轮,Task 1 commit 前 + 全 plan 完成后)。

## Quirks 处置 (D-01/D-03)

### Q-77-C (修复,D-01/D-02 plan 内就地修)

- **位置:** `excel_raw_rows.go:114-120` normalizeHeaderTrim 注释
- **症状:** 注释「转小写用于匹配」与实现(仅 `strings.TrimSpace`)不符
- **修复:** doc-only 注释修正 + 「行为锁定 (Q-77-C, D-03)」段落说明不加 ToLower 的根因
  - 加 ToLower 会改变 "Name"/"name" 表头匹配行为,与既有 ImportFromExcel 调用方契约不符
  - 无 models 常量/字段注释/开发规范明文证据支持修代码(D-03)
- **回归用例:** `TestImp77_NormalizeHeaderTrim_BehaviorLockdown` 9 case 断言**不调用 ToLower**;
  `TestImp77_ReadRawRowsByName_HeaderTrailingWhitespace` 锁定 TrimSpace 行为。
- **生产 blast radius:** 零,代码行 0 改动,只改注释(commit 4539464 仅 +5/-1 行)。

### Q-77-D (记录不修,D-03 无据不臆断)

- **位置:** `code_generator.go:50-54` 与 `:87-91` 的 `if err == gorm.ErrRecordNotFound` 分支
- **症状:** `gorm.DB.Raw(...).Scan(&maxCode)` 不返回 `ErrRecordNotFound`(Scan 永远不报 not-found),
  该分支为死代码;`GenerateCodeWithCustomPrefix` 同款问题。
- **决策:** 按现行为断言,不删代码(SUMMARY 记录待人工裁决)。models 常量 / 字段注释 /
  开发规范均无「错误码必须删除死分支」约定;贸然删除会影响未来若改成 `.First` + `.Scan`
  的兼容性,保守按现行为锁。
- **回归用例:** `TestImp77_GenerateCode` 覆盖空表 → -001 / 递增 / Sscanf 非数字 → 回 1 /
  `GenerateCodeWithCustomPrefix` 5 形态,固化现行为。

## Deviations from Plan

### 1. ResolveBatchWithDependencies 调用契约澄清

- **Found during:** Task 2
- **Issue:** PLAN 假定的 `resolvedIDs` map 格式为 `{"ref:value": "id"}`,但实际 `ResolveSingleWithCondition`
  接收的 `conditions` map 是 `{column: value}` 格式。`resolveDependentReferencesBatch` 调用方传的就是
  `{"building_id": "b1"}`(column→value),而 PLAN 假设的 `resolvedIDs` 在函数体内**直接传给
  `ResolveSingleWithCondition`**——不是先 `makeKey` 反查。
- **Fix:** 测试用正确的 column→value 格式,新增注释说明:`ResolveBatchWithDependencies` 内部把
  resolvedIDs 作为 conditions 传,ResolveSingleWithCondition 把它转成 WHERE clause。
- **Verification:** TestImp77_ResolveBatchWithDependencies_Tail PASS;无生产代码改动。

### 2. WorkstationStatus 联动现行为锁

- **Found during:** Task 2 (WorkstationCRUD)
- **Issue:** `WorkstationService.Create` 调用 `applyWorkstationOccupancyLink` 联动 user_id → status,
  UserID=nil 时强制 status=Available(0);初始 ws.Status=Occupied 会被覆盖 → List by status=1 命中 0 条。
- **Fix:** 测试用 UserID 非 nil 触发联动,语义与生产一致。
- **Verification:** WorkstationCRUD 全 7 子测试 PASS;无生产代码改动。

### 3. 6 表 fixture 扩展

- **Found during:** Task 2
- **Issue:** workstation List 6 表 JOIN 含 `LEFT JOIN sys_user` 与 floor `LEFT JOIN sys_files`,
  原 fixture 缺这两张表 → "no such table" 错误。
- **Fix:** `setupWSFloorCode77DB` 新增 `sys_user` + `sys_files` 手动 DDL,workstation List 全 6 表可查。
- **Verification:** WorkstationCRUD 全子测试 + WorkstationOptions + WorkstationStatistics PASS。

### 4. FloorService 软删恢复路径 sqlite 不可达 (P-77-2 锁定)

- **Found during:** Task 3
- **Issue:** `floor_service.go:101-107` 软删恢复 UPDATE 含 `NOW()`,sqlite 无 NOW() 函数 →
  "no such function: NOW" 错误必报。
- **Fix:** 测试断言错误信息含「恢复楼层失败」即止;不改 NOW()(PG 行为正确,D-03 无据)。
- **Verification:** `TestImp77_FloorCreate` 软删分支断言 PASS;floor_service.go `git diff` 为 0。

**Total deviations:** 4 项(0 个生产代码越界)+ 1 项 quirk 修复(Q-77-C,D-01 授权)+ 1 项记录
不修 quirk(Q-77-D,D-03 判定)。
**Impact on plan:** 4 项 deviations 均为平台/调用契约/测试基建驱动,目标函数全部照常覆盖,
零 scope creep。BLOCK-01 收口按 ≥70% 判据已超 +13.7pp 余量。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestImp77_ 函数 4 文件合计 ≥ 12 | ✅ 44(16+11+8+9) |
| workstation_floor_code 含 List 过滤分支用例(≥name/status/floorId) | ✅ 全 6 过滤 |
| reference_resolver 含 ResolveSingleWithCondition | ✅ +5 子测试 |
| Q-77-C git diff 仅注释行 | ✅ commit 4539464 (+5/-1, 注释) |
| `go test -count=1 -run "TestImp77_"` exit 0 | ✅ |
| `go test -count=1 ./internal/services/operations/` exit 0 | ✅ |
| `go test -count=1 -cover` ≥ 70.0% | ✅ 83.7% |
| `go build ./...` exit 0 | ✅ |
| git status 无新增 testdata/*.xlsx | ✅(全部 `[]byte("not a zip")` / `buildTestXLSX` 内存) |
| 生产 .go 改动 = 仅 excel_raw_rows.go (Q-77-C) | ✅ git diff --stat 零其他 |

## 文件隔离核对(双里程碑协议)

- 本 plan 全部改动:`internal/services/operations/excel_raw_rows.go`(注释 1 处)+ 4 个新测试文件。
- 未触碰:`xingran-react-frontend/`、`xingran-frontend/`、`.planning/workstreams/frontend-coverage/**`、
  `internal/agent/**`(兄弟 executor 持有)、`.planning/workstreams/milestone/STATE.md`(并发持有)。
- 3 个 commit 全部 `git add` 显式路径,无 `git add .` 模式;commitlint 通过(中文动词开头,
  body ≤100 chars)。

## Next Phase Readiness

- **BLOCK-01 已收口**:operations per-package coverage 从 Phase 77 起点 61.1% → 77-01 后 69.8%
  → 77-02 后 73.2% → **77-03 后 83.7%**,远超 ≥70% 线 +13.7pp 余量。
- 77-04 / 77-05 agent 包 BLOCK-02 收口独立进行,本 plan 不影响。
- 本 plan 新增的两个 helper(`setupImportRest77DB` + `setupWSFloorCode77DB`)可作为后续 operations
  包测试标准起手式;`applyWorkstationOccupancyLink` 现行为已锁回归(77-02 也覆盖)。
- 注意事项:ResolveBatchWithDependencies 的 conditions 格式是 column→value,docstring 模糊
  (已记录在测试注释);若未来要按 ref:value→id 自动反查,需改实现 + 加 helper。

---
*Phase: 77-operations-agent-server*
*Completed: 2026-08-27*

## Self-Check: PASSED

- 4 个新测试文件全部存在 ✅(行数 370+200+290+440 ≈ 1300 行)
- 3 个 commit (4539464、fd15f4a、36cb5d3) 均在 main 历史 ✅
- 包级测试 `-count=1` 全绿、`-count=3` flake 筛查通过 ✅
- coverage 83.7% ≥ 70% ✅
- Q-77-C 仅注释变更无代码变更 ✅(`git show 4539464 --stat` 仅 +5/-1)
- Q-77-D 死分支按现行为断言并在 SUMMARY 记录 ✅
