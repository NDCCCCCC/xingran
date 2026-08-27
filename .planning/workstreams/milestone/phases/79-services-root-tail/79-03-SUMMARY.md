---
plan: 79-03
phase: 79-services-root-tail
executed: 2026-08-28
commits:
  - b307ce9 (test(79-03): notice CRUD + 发布撤回 + 状态统计(E 簇常量化断言))
  - bc0e68a (test(79-03): notice 已读/忽略链 + 四类目标解析 + 可见性矩阵)
  - 3f2d48e (test(79-03): template_service CRUD + 校验 + 统计)
  - aaa1ea3 (test(79-03): template Preview/Render/Clone/Export-Import round-trip)
  - 7f1255a (test(79-03): oper_log/api_endpoint/swagger/cron-util 四小文件 0→70%)
---

# 79-03 Summary — notice/template/operlog/api-endpoint 簇 9 文件 0%/3.3% → 全部 ≥85%

## 交付

3 个测试文件(2163 行,31 个新测试函数,零生产 .go 改动),与源码同包共置(D-79-06 命名法):

- `internal/services/notice_service_79_03_test.go`(973 行,**15 个测试**:
  7 TestNtc7903_ + 3 TestNrd7903_ + 3 TestNtg7903_ + 2 TestNqv7903_)。
  helper `newNtc7903`(sqlite t.TempDir 文件库 + AutoMigrate notice 链)/
  `ntc7903User`(sys_user + sys_user_role 种子)/ `ntc7903Publish` / `ntc7903VisibleIDs`。
  四类 target_type 建目标形态(dept 型经 buildTargets 递归展开父/子/孙三层)+
  定时/过去 PublishTime 双态 + creatorName 落库(creatorID 实装不落库已锁定);
  分页/模糊/精确/白名单排序/非法列回退;发布撤回全链(重复发布拒/撤回回草稿/
  Withdrawn 态拒发布/定时保留原 publish_time)+ 状态统计条件聚合口径(软删排除);
  已读链(ip 落库/重复读幂等/未读递减/全量归零)+ 忽略链 + status 过滤(read/unread/
  未知值不过滤)+ 分页;四类目标用户解析(sys_user_role DISTINCT / unique 首现序)+
  getChildDeptIDs 递归含孙 + 阅读统计防除零;**buildUserVisibleQuery 的 4-OR
  可见性矩阵逐支断言**(全体/部门/角色/指定用户 × 命中未命中 + 草稿排除 + 停用排除 +
  ignore 排除),断言走 ctx.Query 实际执行 Count/Find。
- `internal/services/template_service_79_03_test.go`(634 行,**10 个 TestTsv7903_**)。
  helper `newTsv7903`(sqlite t.TempDir + AutoMigrate ConfigTemplate +
  `sys_config_execution` 裸表)/ `baseReq7903` / `withOrder7903` / `*Ptr7903`。
  Create/GetByID/GetByCode 双通道 round-trip(变量 JSON 往返一致)+ 重复 code +
  text/template 语法拦截;List 分页 + 五类过滤 + 白名单排序 + 非法列回退;
  Update 全字段读回 + 系统模板改/删保护;GetStatistics 条件聚合;
  GetVariables 序一致;ValidateVariables 必填缺失/默认值回填/多余变量接受;
  **Preview/Render `{{.name}}` 占位替换**(可选缺省回填默认值)+ 双通道未命中透传;
  **Export→Import round-trip**(同库须硬删源行,见 QUIRK-79-03-K)+ code 冲突/
  坏 JSON/空字节分支;Clone 内容一致 + newCode 重复;GetTemplatesByVendor
  vendor/deviceType 二级过滤(具名常量)。
- `internal/services/operlog_endpoint_tail_79_03_test.go`(556 行,**8 个测试**:
  4 TestOpl7903_ + 2 TestAep7903_ + 1 TestSwe7903_ + 1 TestNcu7903_)。
  helper `newOpl7903`(sqlite + AutoMigrate operlog/menu 链 + MemoryCache 装配
  APIEndpointService)/ `aep7903Metadata`(两模块测试元数据)/ `svcImpl7903`
  (实现型断言,FilterSensitiveParams 定义在实现而非接口)。
  RecordOperLog 直写落库;RecordAsync/RecordFromGinContext 用
  `require.Eventually` 轮询(2s/10ms,禁裸 sleep)等待 goroutine 落库;
  CreateTestContext 构造 GET/POST 请求(URL 含 query/RemoteAddr/claims/
  response_body/start_time/c.Errors)驱动成功失败两态;FilterSensitiveParams
  表驱动覆盖 CLAUDE.md 11 强制关键词 + 大小写不敏感 + 多次出现 + 非敏感保留 +
  非 JSON 不 panic + 与 operlog 包委托一致性;APIEndpointService 权限过滤
  (sys_menu/sys_role_menu/sys_user_role 三表 join,停用与空 perms 排除)+
  hasPermission 表驱动 + 缓存命中证据(删授权链后仍返回)+ InvalidateUserCache
  失效链;swagger 路由抽取(排除 /swagger /metrics 前缀)+ RouteExists +
  shouldExcludeRoute 表驱动;cron 四函数全分支(六段形态/周月参数校验/
  自定义透传/非法 executeTime/常用表达式六段断言)。

## Coverage checkpoint(per-file 实测,`go test -count=1 -coverprofile` 全包一次,388.5s)

| File | 基线(79-RESEARCH §2) | 实测 | 目标 | 结果 |
|---|---|---|---|---|
| notice_service.go | 3.3%(116 unc) | **85.8%**(103/120) | ≥70% | ✅ |
| notice_read_service.go | 0%(70 unc) | **87.1%**(61/70) | ≥70% | ✅ |
| notice_target_service.go | 0%(71 unc) | **93.0%**(66/71) | ≥70% | ✅ |
| notice_query_service.go | 0%(10 unc) | **100.0%**(10/10) | ≥70% | ✅ |
| notice_cron_util.go | 0%(22 unc) | **100.0%**(22/22) | ≥70% | ✅ |
| template_service.go | 0%(166 unc) | **90.4%**(150/166) | ≥70% | ✅ |
| oper_log_service.go | 0%(56 unc) | **89.3%**(50/56) | ≥70% | ✅ |
| api_endpoint_service.go | 0%(46 unc) | **97.8%**(45/46) | ≥70% | ✅ |
| swagger_extractor.go | 0%(18 unc) | **100.0%**(18/18) | ≥70% | ✅ |

- 簇合计:**525/579 covered = 90.7%**;基线 ~575 unc → 54 unc。
- **滚动累计**:root 包总口径 11.3%(79-RESEARCH §2)→ 20.4%(79-02 后)→
  **30.4%**(本 plan 后全包 profile 实测)。本 plan 段贡献 +521 covered,
  Phase 79 累计约 +1012(79-01 +155 / 79-02 +336 / 79-03 +521)。
- SC-2 discharge:9 文件全部脱离 <50% 区(最低 85.8%),SC-2 该簇清欠完成。

## D-79-07 记录(notice_query_service 归属)

notice_query_service.go(10 stmts,buildUserVisibleQuery)按 D-79-07 从 79-01 移入
本 plan:该函数是 `*NoticeService` 方法且依赖 sys_notice/sys_notice_target/
sys_notice_ignore/sys_user 表,全部由本 plan 的 `newNtc7903` fixture 建 —— fixture
内聚。净效果零遗漏:79-01 SUMMARY 已声明移出,本 SUMMARY 为承接记录,实测 100%。

## Quirks 处置(全部「只锁不修」,零生产改动;R7 / Phase 73-04 Q5 同款)

- **QUIRK-79-03-A** `getChildDeptIDs`(notice_target_service.go:116)原生 SQL 读
  `sys_department` 表,而 `models.Department.TableName() = "sys_dept"` —— 两套部门
  表名并存(sys_department 仅存于 archive 的 init_data.sql,GORM 模型层不映射)。
  fixture 显式建 sys_department 裸表驱动递归分支;生产侧递归实际空转(表不存在时
  错误被吞、返回空集),修复属 escape hatch。
- **QUIRK-79-03-B** `GetNoticeList` 非法 `orderByColumn`:ApplySort 白名单回退仅
  warn,又因源码仅在 `orderByColumn == ""` 时补默认 `priority DESC, created_at DESC`
  → 非法列退化为 sqlite 自然序(与 QUIRK-79-02-A 同款)。断言无错误 + 总数正确。
- **QUIRK-79-03-C** `WithdrawNotice` 撤回后写 `PublishStatusDraft`(0)而非
  `PublishStatusWithdrawn`(3);PublishStatusWithdrawn 在 notice_service 无写入点,
  仅可经 UpdateNotice 显式覆盖,且该态下 PublishNotice 被拒。
- **QUIRK-79-03-D** `MarkAllNoticesRead` 可见集口径 = 全部「已发布+正常」通知
  (notice_read_service.go:39-41 不带 target/ignore 过滤),与
  buildUserVisibleQuery 的 4-OR 口径不同。
- **QUIRK-79-03-E**(⚠️ 现网可见)`models.NoticeIgnore` 的唯一索引
  `idx_notice_ignore_user_notice` 标签只含 UserID 一个成员 → 唯一索引实为
  user_id 单列,一个用户至多忽略一条通知,忽略第二条撞 UNIQUE 约束(2067)。
  修复需动 model 标签(生产 schema 变更)。
- **QUIRK-79-03-F** `MarkAllNoticesRead` 不校验既有已读行 → 重复已读记录累积;
  GetUnreadCount 以「可见总数 − 已读行数」计,负值夹 0 恰好掩盖重复行。
- **QUIRK-79-03-G** `template_service.List` 非法 orderByColumn 同 QUIRK-79-03-B
  (白名单回退 + 默认排序不补)。
- **QUIRK-79-03-H** `template_service.Delete` 先统计 sys_config_execution 使用
  计数,但结果未参与任何判断(死代码)—— 有执行记录引用也照删。
- **QUIRK-79-03-I** 系统模板可 Export/Import,导入后 IsSystem 强制重置 false ——
  系统模板经「导出→导入」通道即变为可改可删的自定义模板(系统保护可绕过)。
- **QUIRK-79-03-J** `Clone` 无系统模板保护(与 Update/Delete 不对称),克隆产物
  IsSystem=false。
- **QUIRK-79-03-K**(⚠️ 现网可见,与 QUIRK-79-02-K 同根)软删不释放
  `template_code` 硬唯一索引:Delete 后行仍在,Import 的存在性计数(带
  deleted_at IS NULL)放行但 INSERT 撞 UNIQUE 约束;同库 round-trip 须 Unscoped 硬删。
- **QUIRK-79-03-L** `FilterSensitiveParams` 按 `"<key>":"` 子串定位(委托
  operlog 包):非 JSON 形态(query 形态 password=abc)不脱敏、原样返回、不 panic;
  坏 JSON 但含 `"key":"` 形态仍按子串规则脱敏。
- **QUIRK-79-03-M** `shouldExcludeRoute` 前缀匹配无边界:`/swaggeriff`、`/metricsx`
  因 HasPrefix 命中被排除。

## Deviations from Plan

1. **[环境] `go test -race` 本地不可执行**:Windows 本机 cgo 工具链故障
   (`cgo.exe exit status 2`),与 79-01 Deviation #2 / 79-02 Deviation #3 同源
   (改动前既有测试同样构建失败)。race 纪律由 t.Cleanup 单次 Close +
   禁 t.Parallel + RecordAsync goroutine 的 require.Eventually 轮询防护,
   ci.yml Linux race job 兜底。
2. **[计划-实装口径] research §2 称 template Preview/Render 为「纯字符串替换、
   无 text/template」与实装有出入**:`utils.NewTemplateEngine` 走 text/template,
   占位符为 `{{.name}}` 形态。按 plan 自带「变量形态以实现为准」条款采用
   text/template 口径,并在文件头注释锁定(research §2 的描述偏差如实记录)。
3. **[Rule 1] 测试侧fixture 修正三处**(均为测试自身 bug,非生产问题):
   List 过滤用例漏设 noticeType;Export→Import 首轮未删源行即断言成功(实装
   code 冲突即拒绝);APIEndpointService 失效链断言误以为免权限端点会随缓存
   失效消失(免权限端点对任何用户恒可达)。已按实装语义修正断言。

## Known gaps(剩余 54 unc,全部为 DB 层报错 / 并发不可达分支,不追 100%)

- `notice_service.go`(17 unc):Create 事务失败/建目标失败包装(:83/:91)、
  统计查询失败(:127)、列表 Count/Find 失败(:160/:176)、ByID 非 NotFound 分支、
  Update/Delete/Publish/Withdraw 的 Updates 失败与 RowsAffected==0 夹缝分支
  —— sqlite 单机健康库不可达。
- `notice_read_service.go`(9 unc):GetIgnoredNotices Count/Find 失败、
  GetUnreadCount Pluck 失败、GetUserNotices Count/Find 失败。
- `notice_target_service.go`(5 unc):GetTargetUsers 四类查询失败包装、
  GetNoticeStatistics 的通知查询失败包装。
- `template_service.go`(16 unc):GetStatistics 查询失败、List Count/Find 失败、
  Create/Update/Delete 的 DB 失败包装。
- `oper_log_service.go`(6 unc):RecordOperLog Create 失败、RecordAsync 落库
  失败静默分支(需底层故障注入)、getResponseResult 的 4xx 分支与截断分支
  (>2000 截断)—— Recorder 注入与超长响应可在后续 plan 以故障注入 double 补。
- `api_endpoint_service.go`(1 unc):GetUserAccessibleEndpoints 的
  getUserPermissions 报错包装。
- `notice_query_service.go` / `notice_cron_util.go` / `swagger_extractor.go`:
  0 unc。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestNtc7903_ ≥7 / Nrd+Ntg+Nqv 合计 ≥8 / TestTsv7903_ ≥10 / Opl+Aep+Swe+Ncu 合计 ≥8 | ✅ 7 / 8 / 10 / 8(合计 33 个测试函数) |
| 文件 min_lines:500 / 380 / 320 | ✅ 973 / 634 / 556;contains(CreateNoticeWithTargets/ValidateVariables/FilterSensitiveParams)全命中 |
| 9 文件 per-file ≥70%(数字落 SUMMARY) | ✅ 最低 85.8%,簇均 90.7% |
| E 簇 publish_status 反转语义以具名常量锁定(禁裸 0/1) | ✅ 22 处 models.PublishStatus*/NoticeStatus* 引用 |
| target_type 四类建目标 + 4-OR 可见性矩阵 + getChildDeptIDs 递归断言 | ✅ |
| GetByCode 与 GetByID 双通道 / Export→Import round-trip / Preview+Render 替换断言 | ✅ |
| FilterSensitiveParams ≥6 敏感关键词 + 与 operlog 包一致 | ✅ 11 强制关键词全取样 + 委托一致性断言 |
| RecordFromGinContext 用 CreateTestContext 构造 | ✅(grep 命中 4 处) |
| 既有 notice_status_statistics_test.go 不回归 | ✅(全包跑绿) |
| `go build ./...` == 0 | ✅ |
| `go test ./internal/services/` == 0 | ✅(全包 profile 跑 388.5s exit 0) |
| `go test ./...` == 0 | 见 Self-Check |
| 生产 .go 改动 = 0 | ✅(5 个 commit 全部 *_test.go) |

## 手注(给 79-04..79-06)

- 可复用同包 helper:`newNtc7903`(notice 链 sqlite 装配,含 sys_department /
  sys_notification_channel 裸表 DDL,PG 专属 default 的 sanitize 形态)/
  `ntc7903User` / `ntc7903Publish` / `newTsv7903`(ConfigTemplate 装配)/
  `baseReq7903` / `withOrder7903` / `newOpl7903`(OperLog + APIEndpointService
  + MemoryCache)/ `aep7903Metadata`(APIMetadataConfig 手工构造形态,
  buildIndex 双侧 normalize,无需调 normalize())/ `svcImpl7903`(实现型断言)。
- **goroutine 异步写库纪律**:RecordAsync / RecordFromGinContext 内部起
  goroutine,断言一律 `require.Eventually(cond, 2s, 10ms)`;勿直接断言行数。
- **models 标签陷阱再添两例**(与 79-02 QUIRK-K 同族):唯一索引标签只写一个
  成员(NoticeIgnore)→ 实为单列唯一;软删不释放硬唯一索引(template_code)。
  后续 plan 涉及「删后再建」形态的断言先查 model 标签。
- 若要修 QUIRK-79-03-A(部门表名双轨)/ E(忽略唯一索引)/ I(系统模板导出
  绕过保护)/ K(软删占唯一索引),均属生产/schema 改动,先立项再动手;
  测试已把现行为证据化,修复时改断言即可。
- oper_log 剩余 6 unc 需要故障注入 Recorder/超长响应 double,建议随 79-06
  收口时按投入产出比裁决。

## Self-Check: PASSED

- 文件存在:notice_service_79_03_test.go(973 行)/ template_service_79_03_test.go
  (634 行)/ operlog_endpoint_tail_79_03_test.go(556 行)/ 79-03-SUMMARY.md — 全 FOUND。
- 提交存在:b307ce9 / bc0e68a / 3f2d48e / aaa1ea3 / 7f1255a — 全 FOUND(git log --all)。
- `go build ./...` exit 0(BUILD OK)。
- `go test ./internal/services/` exit 0(全包 coverage profile 跑 388.5s,
  coverage 30.4%,上表 per-file 数字即出自该 profile)。
- `go test ./...` exit 0(全仓后台跑,输出 0 个 FAIL 行)。
- `-race` 抽样:本地 cgo 故障不可执行(见 Deviation #1),非本 plan 引入。
- 生产 .go 改动 = 0(5 个 commit 全部 *_test.go + 本 SUMMARY docs commit)。
