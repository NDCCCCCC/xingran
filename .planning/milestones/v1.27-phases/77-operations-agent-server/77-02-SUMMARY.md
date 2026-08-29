---
plan: 77-02
phase: 77-operations-agent-server
executed: 2026-08-27
commits:
  - 27bd979 (fix(quirk-77-2): 模板说明合并锚点修复 + Task 1 legacy 导出链结构断言)
  - c37b30f (test(77-02): Task 2 工位追加链三 sheet 断言 + 导出查询参数矩阵)
---

# 77-02 Summary — excel_service 导出链结构断言

## 交付

- `internal/services/operations/excel_export_chain_77_02_test.go`(531 行):**10 个 TestExp77_* 测试函数**,交付 excel_service 导出链(D-08 切分的第二大贡献块):
  - **legacy ExportData 主路径**:user / asset / reconciliationExceptionRule 三类未注册进 GetExportConfig 的类型,断言 `config.SheetName` 存在、表头行逐字等于 `config.Columns[].Header` 序列、数据行数 = 在册种子数、UpsertKey 锚点列(deviceSN 等)单元格抽查 + 未知类型兜底分支。
  - **queryData 过滤链**(100%):name LIKE / code LIKE / status 等值 / 空参跳过 / LIKE 不命中空集;sys_dept 字段名特判分支(name→dept_name,code→dept_code)经白盒直调覆盖(department 被新导出链截流,ExportData 入口不可达)。
  - **formatCellValue 四分支**(100%):Options 反查命中(DBField 作键)/ createdAt+updatedAt 时间格式化 / nil 早退 / Sprintf 兜底;并锁定「int64 扫描值与 Options int 键不符 → 反查落空退化为数字文本」现行为(D-03 无据判修)。
  - **writeInstructions**(80%):非空说明逐行合并写入、空串行跳过合并、全空说明零合并、nil 整体早退;含 QUIRK-77-2 回归断言。GREP 实证当前无任何 ExcelConfig 配置 Instructions → 非空分支仅白盒可达。
  - **getExampleValue**(67.3%):单选项 Options 确定性取值、多选项 Contains 防遍历序 flake、常见 Field 映射抽查、Required 回落 "示例"、非必填回落 ""、Reference 不参与示例器分支。
  - **appendWorkstationDeviceSheets 三 sheet 追加链**(73.3%):复用 77-01 七表 fixture(setupWSD77DB)+ ALTER 补 last_logon 列,断言 AD设备/资产设备/物理链路设备三 sheet 存在、表头行逐字相等、AD sheet 行数 = 种子命中数,序列号列锚点(mergeBySerial 合并主键)、OS/LastLogon(batchGetADEnrichment 81%)、IP(machine_ip,batchGetAssetEnrichment 78%)单元格抽查;守卫分支(deviceService 未注入 / 无命中工位静默跳过)齐备。
  - **queryWorkstationIDsForExport 参数矩阵**(95.7%):string LIKE(FilterMapping)/int/bool/[]interface{} IN/nil+空串跳过/未声明字段回退 paramKey 报错六态。

## Coverage checkpoint(计划要求落 SUMMARY)

- **基线 69.8%(77-01 后)→ 实测 73.2%(+3.4pp)**,超过本 plan ≥68% 目标,且为 77-03 收口 BLOCK-01 的 70% 线留足余量(73.2% 已越过 70%,77-03 只需维持不倒退即可锁定)。
- 目标函数覆盖率:`go tool cover -func`:queryData 100% / formatCellValue 100% / writeDataRows 100% / ExportData 78.8% / appendWorkstationDeviceSheets 73.3% / queryWorkstationIDsForExport 95.7% / batchGetADEnrichment 81.0% / batchGetAssetEnrichment 77.8% / batchGetWorkstationNames 80.0% / writeDeviceSheet 90.5% / getExampleValue 67.3% / writeInstructions 80.0%。剩余盲区集中在 physErr 非降级分支与 PG-only SQL(计划内豁免,P-77-1)。
- `go build ./...` exit 0;`go test -count=1 ./internal/services/operations/` 全绿(含 77-01 无回归);新测试 `-count=3` flake 筛查通过两次(v2 完整版与提交后各一次)。按双里程碑隔离协议未跑根级 coverage gate/CI 脚本。
- 生产 .go 改动仅 quirk-77-2 一处(27bd979,D-01/D-02 登记,见下)。

## Quirks 处置(D-01/D-03)

- **quirk-77-2(修复,D-01 发现即修 + D-02 plan 内就地修)**:`writeInstructions` 的合并 end 锚点固定为第 1 行(`CoordinatesToCellName(len(columns), 1)`),多行说明时第二次 `MergeCell` 变成 `A{row}:M1` 跨行巨型合并 —— excelize 清空区间内非锚点单元格,**首个说明文本被吞只剩末行**,与同函数「逐行写样式/逐行 SetRowHeight」的设计意图相悖(D-03 据源 = 自身代码意图注释)。修复:改为逐行计算 endCol。回归断言在同一 commit(Phase 75 五步法):首个说明必须留在 A1、两处合并且 end 锚点列恒为 M。**生产 blast radius 为零**:GREP 实证没有任何 ExcelConfig 配置 Instructions,修复的是休眠路径,字节级行为变化不触达任何现有调用方。
- **quirk-77-3(记录不修,D-03 无据)**:`formatCellValue` 的 Options 反查键用 `interface{}` 保存 `int`,DB 扫描返回 `int64` → 反查必落空退化为数字文本("0" 而非 "否")。models 常量 / 字段注释 / 开发规范均未定义导出层枚举文本化契约,保守按现行为断言留待人工裁决;该行为在生产同样存在(PG 扫描亦返回数值类型),不在本 plan 修改面内。

## Deviations from Plan

### 前提修正(D-03 有据,plan 内自适应)

1. **legacy 类型清单修正**
   - **Found during:** Task 1
   - **Issue:** PLAN must_haves 称 legacy 路径含 "department/serverRoom/roomDevice/dedicatedLine/infoPoint",实际这些类型均已注册进 `GetExportConfig`(excel_export_config.go map 键清单,本次逐一 Read 复核),走 `excelExportServiceImpl` 新导出链而非 excel_service.go 的 legacy 分支。
   - **Fix:** 真正的 legacy 类型改用 user / asset / reconciliationExceptionRule 三类(仍满足「至少 3 个 legacy 类型断言 config.SheetName GetRows」验收);sys_dept 特判分支以 queryData 白盒直调补覆盖。
   - **Verification:** 全绿 + coverage 数字见上。
2. **workstation 追加链入口改白盒直调(P-77-1 平台限制文档化)**
   - **Found during:** Task 2
   - **Issue:** PLAN 设想 `svc.ExportData(ctx,"workstation",...)` 直达追加链;实测 workstation 主表导出走 WorkstationQueryBuilder,其 `::uuid`/`::text` 强转在 sqlite 解析必败(`unrecognized token ":"`),ExportData 在追加链之前即报错返回。
   - **Fix:** 追加链改白盒直调 `svc.appendWorkstationDeviceSheets(...)`;另增 `TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` 把该现行为显式锁进测试(错误含「查询数据失败」),防止后续误判为可绕过。物理 sheet 的 A1 假设成立:physErr != nil 降级后 0 数据行但 12 列表头保留(P-77-10),由 77-01 FrontSegment 测试实证背书。
   - **Verification:** TestExp77_ExportData_WorkstationAppendSheets 全绿,三 sheet 断言齐备。

**Total deviations:** 2 项前提修正(0 个生产代码越界)+ 1 项登记 quirk 修复(quirk-77-2,D-01 授权)+ 1 项记录不修 quirk(quirk-77-3,D-03 判定)。
**Impact on plan:** 两项修正均为平台/代码事实驱动的入口等价替换,目标函数全部照常覆盖,零 scope creep。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestExp77_ 函数 ≥5(Task 1)/ ≥7(Task 2) | ✅ 5(Task 1 提交点)/ ✅ 10(最终) |
| ≥3 legacy 类型断言 config.SheetName GetRows | ✅ user/asset/reconciliationExceptionRule |
| 物理链路设备≥1 且 AD设备/资产设备 出现 | ✅ 3/2/3 次 |
| `go test -count=1 -run TestExp77_` exit 0 | ✅ |
| `go test -count=1 ./internal/services/operations/` 全绿 | ✅(含 77-01 无回归)|
| `go test -count=1 -cover` ≥68% 落 SUMMARY | ✅ 73.2% |
| `go build ./...` exit 0 | ✅ |
| git status 无新增 testdata/ 或 *.xlsx 二进制 | ✅(grep 空)|
| 生产 .go 改动 = 0 文件(除 D-01 登记修复) | ✅ 仅 excel_service.go 一处 |

## 文件隔离核对(双里程碑协议)

- 本 plan 全部改动:`internal/services/operations/excel_export_chain_77_02_test.go`(新)+ `internal/services/operations/excel_service.go`(quirk-77-2 单点修复)。
- 未触碰:xingran-react-frontend/、xingran-frontend/(另一会话遗留 untracked)、.planning/workstreams/frontend-coverage/**、milestone STATE.md/config.json(并发会话持有,全部绕过未 stage);两次 commit 均 `git add` 显式路径。

## Next Phase Readiness

- 77-03(excel 导入剩余 + reference_resolver + 卫星文件)可直接收口 BLOCK-01:包覆盖率已在 73.2%,只需维护性增量即可锁定 ≥70%;legacy/queryData/写行格式化函数已被本 plan 打满或接近打满,77-03 应聚焦 ImportData 解析链、ON CONFLICT upsert 与 resolver。
- fixture 复用接口:`setupWSD77DB` + `ALTER TABLE sys_ad_computer ADD COLUMN last_logon DATETIME` 已验证可作为追加链类测试的标准起手式;settlement 于 `setupExp77LegacyDB` 的四表 schema(user/asset/recon/dept)供导入侧沿用。
- 注意事项:WorkstationQueryBuilder 的 PG-only 强转仍是 sqlite 不可达区(P-77-1 已有两个守护测试锁定现状);若后续做跨方言改造,`TestExp77_ExportData_WorkstationMainSheet_PGOnlySQL` 会先行报警提示更新断言。

---
*Phase: 77-operations-agent-server*
*Completed: 2026-08-27*

## Self-Check: PASSED

- `internal/services/operations/excel_export_chain_77_02_test.go` 存在(531 行 / 10 个 TestExp77_ 函数)✅
- commit 27bd979、c37b30f 均在 main 历史 ✅(git log/show 复核;27bd979 含 excel_service.go +11/-4)
- 包级测试 `-count=1` 全绿、`-count=3` flake 筛查通过、coverage 73.2% ✅
