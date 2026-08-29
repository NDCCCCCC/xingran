---
plan: 79-04
phase: 79-services-root-tail
executed: 2026-08-28
commits:
  - cfa81b9 (test(79-04): knowledge 文章族 CRUD/搜索/统计/计数器)
  - bec638f (test(79-04): knowledge 分类树递归 + 标签族(GetOrCreateTag 双路径))
  - ccbd0c4 (test(79-04): network_device CRUD + nil 依赖注入 + 统计)
  - ae0c7f7 (test(79-04): notification_config CRUD + AES 密码对 round-trip)
  - 7b260df (test(79-04): notification_sender 派发链 + nil-sender 分支 + 收件人解析)
  - 11f1362 (test(79-04): auth_credential PasswordCipher stub 全链)
---

# 79-04 Summary — knowledge+network+notification+auth 簇 5 文件 1.4%/0% → 全部 ≥76.7%

## 交付

4 个测试文件(2920 行,45 个新测试函数,零生产 .go 改动),与源码同包共置(D-79-06 命名法):

- `internal/services/knowledge_service_79_04_test.go`(877 行,**13 个 TestKsv7904_**)。
  helper `newKsv7904`(sqlite t.TempDir 文件库 + AutoMigrate 知识库三族 + WorkOrder)/
  `ksv7904SeedTag`(Q2 规避:标签 pre-seed)/ `ksv7904Category` / `ksv7904WorkOrder` /
  `ksv7904TagCounts` / `ksv7904AssocTagIDs` / `baseListReq7904`。
  文章创建(名称/UUID 双分支 + 重复名去重 + use_count)/List 分页与状态/分类/标题/创建人
  过滤/Get/Update/Delete 往返 + 不存在分支;Update 标签同步差量(QUIRK-79-04-A 锁定);
  Search 标题/内容/摘要三分支 + 已发布过滤 + 分类过滤 + 分页钳制(0→100、>500→500、
  负 pageNum 归零)+ TagID 过滤(QUIRK-79-04-B 锁定报错);Increment 计数器(含不存在 ID
  的 Update 语义);工单转文章(已完成/已关闭放行 + 待处理/不存在/已转换三拒绝);
  GetArticleStatistics 条件聚合对照既有口径 + 空表 COALESCE;ParseTagsFromContent 表驱动;
  三层分类递归树 + ParentID/Status 过滤 + 分类 CRUD 删除拒绝(子类/文章)+ 标签 7 方法
  全链(排序 use_count DESC/未命中 (nil,nil)/重名唯一索引拒)+ GetOrCreateTag 双路径 +
  AutoCreateTagsFromContent 幂等。
- `internal/services/network_device_service_79_04_test.go`(596 行,**9 个 TestNdv7904_**)。
  helper `newNdv7904`(`NewNetworkDeviceService(db, nil, nil)` 双 nil 依赖)/
  `ndv7904SeedDevice`(status 强制回写,Q4)/ `ndv7904SeedCredential`(pq.StringArray)/
  `ndv7904SeedDept`。Create/GetByID(Q3 关联名丢失锁定 + List 路径 enrichment 对照)/
  List 六类过滤 + 分页 + 排序白名单(合法 ASC/DESC + 非法列回退)/Update 全字段读回 +
  四拒绝分支/Delete/BatchDelete(非原子锁定)/UpdateStatus 单批 + 越界值无校验锁定/
  QuickCreate 五个不出网分支(IP 已存在/凭证缺失/部门缺失/探测失败阻止创建/空 IP)/
  getLastIPOctet 表驱动/GetDeviceStatistics 三维分组 + ByDept LEFT JOIN 空键/
  GetDevicesByDept/ByCredential 命中与空集/nil 依赖安全路径。
- `internal/services/notification_chain_79_04_test.go`(806 行,**17 个测试**:
  5 TestNcf7904_ + 12 TestNsn7904_)。
  helper `newNcf7904` / `newNsn7904`(notice 链 + `sys_notification_channel` 裸表,
  PG gen_random_uuid sanitize 形态)/ `nsn7904NilSender` / `nsn7904User` / `nsn7904Notice`
  / `nsn7904Channel` / `ncf7904EmailConfig`。EncryptPassword/DecryptPassword AES-GCM
  往返(随机 nonce/空明文/默认 key/短 key 补齐/长 key 截断)+ 错 key GCM 认证失败 +
  坏 base64/too short/篡改密文/空密文四形态;EmailConfig 单条规则(自动默认 + 第二条拒 +
  软删后可重建)/List 状态过滤/Update 默认互斥 + 零值字段保留密码/Delete 软删 del_flag/
  GetDefault 命中与"未设置";APINotificationConfig 六方法 + 类型过滤 + 同类默认互斥;
  SendNotification 无渠道兜底 + 不存在/草稿/撤回三错误分支 + isValidPublishStatus 表驱动;
  nil-sender email/api 安全分支(配置缺失/空收件人早退,不 panic)+ 真实 sender 出网前
  终止路径(停用配置/坏密文/配置不存在);web+sms 渠道分支;buildRecipientList 合并去重;
  getUserInfo/getUserEmails/getUserPhones 空值剔除;getEmailConfigID 三分支;
  PublishAndSendNotice(发布态断言 + 非幂等锁定);Set/Get 渠道覆盖写与空集合语义;
  uniqueStrings 表驱动。
- `internal/services/auth_credential_service_79_04_test.go`(543 行,**10 个 TestAcv7904_**)。
  stub `stubCipher7904`(addomain.PasswordCipher,per-interface *Func 字段 + 未注册即
  panic + encryptCalls 观测面,Phase 73 D-02 范本)/ helper `newAcv7904` /
  `acv7904SeedCredential` / `acv7904ReversibleCipher`。Create 双密码经 cipher 落库 +
  返回值脱敏 + 重名拒 + validateCredentialConfig 五分支表驱动 + ValidateCredential 兼容
  入口;List 分页/名称模糊/协议具名常量过滤/白名单排序/密码隐藏;GetByIDWithPassword 与
  GetDecryptedCredential 往返 + 密码/特权密码解密失败严格模式 + GetByID 对照;
  Update 轮换与留空保留 + 名称冲突/不存在拒;Delete 设备占用拒 + 不存在 ID 静默 no-op
  锁定;默认凭据唯一性(Set/Update 双路径 + 无默认错误);GetStatistics 条件聚合;
  GetDevicesByCredential 命中与空集;nil cipher 防御分支;stub 未注册 panic 契约。

## Coverage checkpoint(per-file 实测,`go test -count=1 -coverprofile` 全包一次,393.3s)

| File | 基线(79-RESEARCH §2) | 实测 | 目标 | 结果 |
|---|---|---|---|---|
| knowledge_service.go | 1.4%(285 unc) | **82.0%**(237/289) | ≥70% | ✅ |
| network_device_service.go | 0%(202 unc) | **76.7%**(155/202) | ≥70% | ✅ |
| notification_config_service.go | 0%(127 unc) | **83.5%**(106/127) | ≥70% | ✅ |
| notification_sender_service.go | 0%(120 unc) | **90.8%**(109/120) | ≥70% | ✅ |
| auth_credential_service.go | 0%(142 unc) | **85.2%**(121/142) | ≥70% | ✅ |

- 簇合计:**728/880 covered = 82.7%**;基线 ~876 unc → 152 unc。
  (stmts 计 880 vs research 876:跨 plan 的行级漂移,口径不变。)
- **滚动累计**:root 包总口径 11.3%(79-RESEARCH §2)→ 20.4%(79-02 后)→
  30.4%(79-03 后)→ **45.2%**(本 plan 后全包 profile 实测)。本 plan 段贡献
  +728 covered,Phase 79 累计约 +1740(79-01 +155 / 79-02 +336 / 79-03 +521 / 79-04 +728)。
- SC-2 discharge:5 文件全部脱离 <50% 区(最低 76.7%)。

## Stub 范式说明(Phase 73 D-02 沿用)

`stubCipher7904` 实现 `addomain.PasswordCipher`(编译期 `var _` 断言):每方法一个
`*func(...)` 字段,**未注册即 panic**(防止测试静默走通非预期路径),另带 `encryptCalls`
观测面供 service 层断言"明文确实交给了 cipher"。可逆行为(`enc:` 前缀加解)由
`acv7904ReversibleCipher` 注册;错误注入与部分失败(密码成功/特权密码失败)用独立
stub 实例 + 定制 Func 表达。race 纪律:*Func 字段每用例独立赋值不共享 + 全文件禁
t.Parallel。零 testify/mock 依赖,与 STATE.md 锁定范式一致。

## Quirks 处置(全部「只锁不修」,零生产改动;R7 / Phase 73-03 决策沿用)

**Phase 73-03 已记录、本 plan 复测锁定(plan notes 点名三项):**

- **Q1 knowledge UUID inline-Delete**:`DeleteKnowledgeCategory`(:683)/`DeleteTag`
  (:775)用 `db.Delete(&Model{}, id)` 内联条件,GORM 把非数字字符串实参按原生 SQL
  片段处理 → sqlite 报 `unrecognized token: "9dba"`。测试以数字串主键("9901"/"8801")
  走 happy path,并用真实 UUID 断言报错 + 行不被删(knowledge 测试两处)。
- **Q2 knowledge GetOrCreateTag 跨 tx 连接**:`CreateKnowledgeArticle`/`Update`/
  `AutoCreateTagsFromContent` 在事务内经外层 `s.db` 读建标签 → 第二条连接 INSERT
  撞写锁,glebarez 5s busy 超时后 `SQLITE_BUSY`。按 73-03 同款处置:**标签一律
  pre-seed**(`ksv7904SeedTag`)让外层连接走纯读分支;GetOrCreateTag 的创建路径用
  无事务直调覆盖(TestKsv7904_GetOrCreateTag_BothPaths)。
- **Q3 network GetByID 关联名丢失**:`GetByID`(:375)把 loadAssociations 结果写进
  一次性切片 `&[]models.NetworkDevice{device}` → DeptName/CredentialName 永不回填。
  断言两者为 nil,并与 List 路径(真实切片,回填正常)形成对照断言。

**本 plan 新发现(锁定不修,SUMMARY 复记;修复属 escape hatch):**

- **QUIRK-79-04-A**(⚠️ 现网可见)`UpdateKnowledgeArticle` 的旧标签关联清理用
  `tx.Delete(&oldTag)`(knowledge_service.go:333),而 `models.KnowledgeArticleTag`
  三列均非主键 → GORM 报 `WHERE conditions required` 且返回值被忽略 → **旧关联行
  从不删除**。净效果:标签同步只增不减,且 use_count 在删除失败后仍无条件 -1(计数
  与关联行数脱钩)。同样形态存在于 `DeleteKnowledgeArticle`(:402,文章删除后留下
  孤儿关联行)。测试锁定:替换 {甲,乙}→{乙,丙} 后关联集累积为 {甲,乙,丙}。
- **QUIRK-79-04-B**(⚠️ 现网可见)`SearchKnowledgeArticles` 带 TagID 过滤时
  INNER JOIN `sys_kb_article_tags`(:478)后仍追加 `Order("created_at DESC")`
  (:504),两表同名列 → sqlite/PG 均报 `ambiguous column name: created_at`,
  **按标签搜索在现网必然报错**。测试锁定报错形态。
- **QUIRK-79-04-C** `UpdateEmailConfig`/`UpdateAPINotificationConfig` 把调用方传入的
  struct 直接交给 `Updates(...)`:载荷携带非零主键时 GORM 会把 `id` 写进 SET →
  **改行主键**。测试以空 ID 载荷规避并注释锁定。
- **QUIRK-79-04-D** GORM 零值跳过 ×2(knowledge/network 两侧同根):
  `CreateKnowledgeCategory` 传 `KnowledgeArticleStatusDraft`(0)被列 default:1
  覆盖;`NetworkDeviceService.Create` 传 `DeviceStatusOnline`(0)被列 default:2
  覆盖 → **"在线"建机实际落库为"未知"**。测试以建后回写(种子)与现行为断言(服务)
  双向锁定。
- **QUIRK-79-04-E** `AuthCredentialService.Delete` 不校验行存在性(只查设备占用)→
  删除不存在的凭据是静默 no-op,`BatchDelete` 混入不存在 ID 也不报错(与
  `network_device_service.Delete` 的先查后删语义不一致)。
- **QUIRK-79-04-F** `NetworkDeviceService.UpdateStatus/UpdateStatusBatch` 无状态
  白名单校验,越界枚举值(`models.DeviceStatus(99)`)同样落库。
- **QUIRK-79-04-G** `CreateEmailConfig`/`CreateAPINotificationConfig` 的默认互斥:
  API 版 Create 分支(:192)不排除自身,新建第二条默认后第一条被清、第二条为默认
  (结果唯一但写序依赖实现);Email 版则是"系统只允许一条"硬规则(软删后可重建)。
- **QUIRK-79-04-H**(network)`BatchDelete` 逐条 `Delete` 非原子:前面行已软删后才在
  缺失 ID 上失败,无事务回滚。

## Deviations from Plan

1. **[环境] `go test -race` 本地不可执行**:Windows 本机 cgo 工具链故障
   (`cgo.exe exit status 2`),与 79-01 Deviation #2 / 79-02 #3 / 79-03 #1 同源
   (改动前既有测试同样构建失败)。race 纪律由 t.Cleanup 单次 Close + 全文件禁
   t.Parallel + stub *Func 每用例独立赋值防护,ci.yml Linux race job 兜底。
2. **[计划-实装口径] Task 1 两条断言按 plan 预期写不上**:plan 写「更新时替换 tag
   集合 → 关联行差量正确(新增/移除)」与「TagID 命中 → 返回命中集」,实装分别因
   QUIRK-79-04-A / 79-04-B 不可达。按 plan 自带的「quirk 锁定不修 + 按现行为断言」
   条款改锁现行为,并在 Quirks 段登记(R7 纪律,未做任何生产改动)。
3. **[测试侧适配] SQLITE_BUSY 首跑失败**:knowledge 事务内经外层连接 INSERT 标签
   必撞写锁(Q2),按 73-03 commit 记录的处置改为 pre-seed,并在 fixture 注释中
   说明规避前提(文件库而非 :memory:)。属测试装配适配,非生产/计划语义变更。
4. **[Task 3 范围] QuickCreateDevice 成功路径不可达**:探测成功后的建行/恢复软删/
   Enqueue nil-guard(:346)需要 SNMP wire fake(78-04 的 fake 是 in-package
   test-only,不可导入;DQ5 归 79-06 裁决)。本 plan 覆盖探测前 3 个校验短路 +
   探测失败阻止创建 + 空 IP 参数短路,该函数实测 34.7%,文件整体 76.7% 达标。

## Known gaps(剩余 152 unc,主体为 DB 报错包装与 wire 专属分支,不追 100%)

- `knowledge_service.go`(52 unc):各方法的 `fmt.Errorf("...: %w", err)` 包装分支
  (统计/列表 Count+Find/详情非 NotFound/创建与更新与删除的事务失败/计数器/
  分类与标签各查询失败)—— sqlite 单机健康库不可达;另 QUIRK-79-04-A 的"正确差量"
  分支在现行为下不存在。
- `network_device_service.go`(47 unc):**QuickCreateDevice 成功路径**(探测成功后
  的命名/类型回退、软删恢复、新建、Enqueue nil-guard,:284-353)需 SNMP fake(79-06
  DQ5);其余为 Create/Update/Delete/UpdateStatus/GetDevicesBy* 的错误包装。
- `notification_config_service.go`(21 unc):EncryptPassword 的 `aes.NewCipher`/
  `NewGCM`/`rand.Reader` 失败分支(键长已由实现钳到 16 字节,不可达)、各 CRUD 的
  Count/Create/Updates 失败包装。
- `notification_sender_service.go`(11 unc):SendNotification 的 getUserInfo 失败
  包装、渠道/用户查询失败包装、SetNotificationChannels 事务失败分支。
- `auth_credential_service.go`(21 unc):GetStatistics/List/Create/Update/Delete/
  SetDefaultCredential 的查询与写入失败包装(需底层故障注入 double,建议随 79-06
  收口按投入产出比裁决)。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestKsv7904_ ≥13 / TestNdv7904_ ≥9 / TestNcf7904_ ≥5 / TestNsn7904_ ≥10 / TestAcv7904_ ≥8 | ✅ 13 / 9 / 5 / 12 / 10(合计 49 个测试函数,含子测试) |
| 文件 min_lines:520 / 380 / 380 / 300;contains(GetOrCreateTag / QuickCreateDevice / EncryptPassword / GetDecryptedCredential) | ✅ 877 / 596 / 806 / 543 行;contains 全命中 |
| 5 文件 per-file ≥70%(数字落 SUMMARY) | ✅ 最低 76.7%,簇均 82.7% |
| status 断言引用 models.DeviceStatus*(禁裸 0/1) | ✅ network 测试 39 处 models.DeviceStatus* 引用 |
| getLastIPOctet 表驱动存在 | ✅ 6 用例(标准 IP/末段多位/无点/空串/结尾点/单字符) |
| AES round-trip + 错 key + 坏密文三断言齐备;默认配置唯一性分支断言存在 | ✅(round-trip 7 形态 + 错 key 2 形态 + 坏密文 4 形态;Email/API 默认互斥各有断言) |
| nil-sender 两个错误分支(email/api)各有用例且不 panic | ✅(email 配置缺失/空收件人;api APIConfigID=nil 早退) |
| PasswordCipher stub 未注册方法被调时 panic(stub 契约) | ✅ TestAcv7904_StubContract_UnregisteredPanics(PanicsWithValue) |
| key_notes 三 quirk(knowledge×2 / network×1)按现行为断言+注释 | ✅(文件头 + 用例内注释;SUMMARY 复记) |
| `go build ./...` == 0 | ✅ |
| `go test -count=1 ./internal/services/` == 0 | ✅(全包 profile 跑 393.3s exit 0) |
| `go test ./...` == 0 | 见 Self-Check |
| 生产 .go 改动 = 0 | ✅(6 个 commit 全部 *_test.go,git show --stat 逐一核对) |
| 既有 knowledge_statistics_test.go / discovery_statistics_test.go 不回归 | ✅(全包跑绿) |

## 手注(给 79-05 / 79-06)

- 可复用同包 helper:`newNsn7904`(notice 链 + `sys_notification_channel` 裸表,
  PG default sanitize)/ `nsn7904User` / `nsn7904Notice` / `nsn7904Channel` /
  `ncf7904EmailConfig` / `newNcf7904` / `newKsv7904`(知识库三族)/ `newNdv7904`
  (NetworkDevice+AuthCredential+Department,pq.StringArray 可用)/ `newAcv7904` /
  `stubCipher7904`(PasswordCipher *Func stub)/ `baseListReq7904` /
  `ndv7904StrPtr`。
- **发送链测试不出网纪律**(79-05 wire 级接管):真实 sender 的断言一律停在
  「配置缺失 / 配置停用(models.NotificationConfigStatusStopped)/ 密码不可解密 /
  收件人无效」前置校验;nil-sender 下不得让带收件人的 email/api 渠道触达 Send
  (nil 指针解引用必 panic)。
- **标签同步语义**:在 QUIRK-79-04-A/B 修复前,knowledge 测试的标签差量断言与
  TagID 搜索断言都是「现行为证据」;若立项修复,直接改这两处断言即可。
- network QuickCreateDevice 剩余 47 unc 里的探测成功路径,建议 79-06 的 SNMP fake
  (DQ5)一并覆盖;届时 `ndv7904SeedCredential` 已可产出带 communities 的凭证。
- auth_credential / notification 剩余 unc 全是 DB 故障包装,如需故障注入 double
  请沿用 `stubCipher7904` 的 *Func 范式(勿引入 testify/mock)。

## Self-Check: PASSED

- 文件存在:knowledge_service_79_04_test.go / network_device_service_79_04_test.go /
  notification_chain_79_04_test.go / auth_credential_service_79_04_test.go /
  79-04-SUMMARY.md — 全 FOUND。
- 提交存在:cfa81b9 / bec638f / ccbd0c4 / ae0c7f7 / 7b260df / 11f1362 —
  全 FOUND(git log --all);`git show --stat` 逐一核对 6 个 commit 仅含 *_test.go,
  生产 .go 改动 = 0。
- `go build ./...` exit 0;`go test ./internal/services/` exit 0
  (profile 跑 393.3s;全包复跑 405.9s);
  **`go test ./...` exit 0(repo_full_test_79_04.log EXIT=0,0 FAIL,日志已清理)**。
- `-race` 抽样本地不可执行(Deviation #1,cgo 工具链故障,非本 plan 引入)。
