---
plan: 79-05
phase: 79-services-root-tail
executed: 2026-08-28
commits:
  - faea725 (test(79-05): mac query 装配 + GetVendor 缓存路径 + ImportOUIData)
  - 3829aad (test(79-05): mac query 四查询链 + 阈值 + 缓存语义等价)
  - 93fc19d (test(79-05): mac ExportHistory xlsx 字节与内容断言)
  - 6c2caab (test(79-05): mac_collection 解析纯函数表 + CRUD/配置阈值)
  - aff0e32 (test(79-05): mac_collection 补差 FilterRule stub + 假方言错误包装(70.4%))
  - 7016343 (test(79-05): mac history/partition/matview/heatmap/perf_seed(PG-only 字符串化口径))
  - 8cdbfb2 (test(79-05): email_sender plain fake SMTP + TLS dial-error + 纯 builder(D-79-03))
  - 022b00c (fix(79-05): smtp 会话快照拆值类型,消除 vet copylocks 告警)
  - ac86876 (test(79-05): api_sender fake HTTP 回环 + ad_ldap Tier-1 守卫表(D-79-04))
---

# 79-05 Summary — 外呼三件套 (email/api_sender/ad_ldap) + mac 家族

## 交付

5 个测试文件(3819 行,62 个顶层测试 + 173 个子测试 = 235 个测试用例,零生产 .go 改动),
与源码同包共置(D-79-06 命名法):

- `internal/services/mac_history_query_service_79_05_test.go`(874 行,**14 个 TestMhq7905_**)。
  双装配 `newMhq7905`(cache=nil)/`newMhq7905Cached`(MemoryCache + DataCacheService +
  CacheConfigService,同包白盒同时注入 cache/dataCache/perfConfig 三字段)。
  perfCacheTTL 双分支(QUIRK-79-05-A);两构造器接口断言;GetVendor 缓存/DB 双路径 +
  **删行后仍命中缓存的可观察证据** + MemoryCache 直查 `mac:vendor:<OUI>` 键;
  ImportOUIData 经 `t.Chdir` 在 t.TempDir 造 `configs/oui-vendors.json`(禁污染仓库 configs/)
  驱动导入 3 行 + 幂等跳过 + 缺文件/坏 JSON;四查询链(QueryPortHistory/QueryDeviceHistory/
  QueryHistory/QueryConnectionStats)全分支(过滤/分页/时间三态/1 年跨度守卫/非法 UUID/
  非法 MAC/空结果形态)+ getLongOccupancyThreshold 默认/覆盖/非法回退;
  **ExportHistory xlsx 双重断言**(`PK` zip magic + `excelize.OpenReader` 重开)验证
  sheet 名「MAC 历史」、9 列表头、DESC 首行、VLAN 数值形态、interface/MAC 过滤只导命中行、
  空数据仅表头、非法 MAC/UUID/超 30 天/坏时间四拒绝;缓存装配与裸装配语义等价(同序同集)。
- `internal/services/mac_collection_service_79_05_test.go`(744 行,**14 个 TestMcl7905_**)。
  `newMcl7905`(NewMACCollectionService(db,nil,nil,nil))+ `mcl7905RuleStub`
  (topology.FilterRuleService stub,Phase 73 D-02 *Func/未注册即 panic 范本)+
  mhs7905PGFake 假方言复用。解析三函数表驱动(华为/锐捷/迈普 × mac-table/port-security ×
  合法/缺字段/垃圾行/canonical 守卫)、mergeMACEntries 三元组去重与首见序、
  cleanTimestampFromInterface(含纯时间戳回空串)、getMACCommands 厂商矩阵
  (华为/华三单命令,锐捷/迈普双命令,未知兜底)、getMACThreshold 硬编码表 + 规则命中 +
  规则失败回退、loadConfigFromDB/ReloadConfig 热载矩阵、GetMACAddressList 分页/四过滤/
  排序白名单/非法列回退、Stats 手算、CleanOldRecords/BatchDelete、nil-executor 边界
  (QUIRK-79-05-F)。
- `internal/services/mac_history_tail_79_05_test.go`(785 行,**12 个 Test(Mhs|Mhp|Mhm|Mhh|Mps)7905_**)。
  **新增 R6 口径基建 `mhs7905PGDialector`/`mhs7905PGPool`**:仅 `Dialector.Name()=="postgres"`
  + 捕获 SQL 且必失败的 ConnPool,并自补 `callbacks.RegisterDefaultCallbacks`(gorm 的默认
  callbacks 由各真实 dialector 的 Initialize 注册,假方言不补则 Exec/Scan 在空 fns 上静默返回)
  → 让 dialect 守卫放行到 PG-only 字符串构造行,再以 Exec/Query 失败收尾,
  **零真实 PG 交互**。覆盖:PARTITION OF DDL 字符串形态(含 2026-12→2027-01 跨年边界)、
  年份/月份校验、EnsurePartitionsExist 错误聚合(3>2 分支)、DropExpiredPartitions 查询包装、
  matview 白名单拒 + `REFRESH MATERIALIZED VIEW CONCURRENTLY` 精确形态(D-10)+
  MV-01→MV-04 刷新顺序 + 部分失败容错(D-11)、heatmap MV 查询错误包装 + perfTopN
  (引用 MACPerfConfigHeatmapTopN 常量)+ 7 天缺省窗 + 缓存装配命中、
  mac_history_service 的 BuildMACStateMap 归一化/覆盖语义、vlanEqual 表、RecordMACChange
  四事件(appeared/disappeared/moved/vlan_changed)+ 无变化不落行 + 首见设备初始化、
  MergeFlappingRecords 窗口内外合并、CleanupAllDevicesFlapping、MergeByTransitions
  转换点保留/vlan_changed 删除、SeedMACPerfConfigs 3 键匹配 macPerfConfigDefaults +
  幂等不覆盖 + nil db。
- `internal/services/email_sender_service_79_05_test.go`(719 行,**11 个 TestEml7905_**)。
  **自包含 plain fake SMTP**(`startFakeSMTP7905`:127.0.0.1:0 + bufio 会话
  220→EHLO 多行(通告 AUTH)→235→250→354→250→221,记录 mail from/rcpt/data/命令序列;
  listener/goroutine 全部 t.Cleanup 关闭与等待)。sendPlainSMTP 全链会话断言
  (命令序列 EHLO→AUTH PLAIN→MAIL→RCPT→DATA + DATA 内容含 Subject/MIME 头)、
  550 拒绝 / 中途硬断连 / 连接拒绝三错误分支、Send / SendWithDefaultConfig /
  SendNoticeEmail / TestEmailConfig 全分支(配置缺失/停用/空收件人/非法收件人/坏密文)、
  sendEmail 三路派发矩阵 + **sendWithSTARTTLS 未通告扩展的明文回退 happy 段**、
  TLS 双路径 dial-error(InsecureSkipVerify:false 硬编码 :203-204/:271-272 注释锁定)、
  纯 builder(buildEmailContent 三形态/plainAuth.Start+Next/notice 三函数/HTML 体)。
- `internal/services/api_sender_ad_ldap_79_05_test.go`(697 行,**11 个 Test(Apd|Adl)7905_**)。
  api_sender 选型 **httpmock**(research §8 首选):`httpmock.Activate()` +
  `ActivateNonDefault(svc.client)`(svc.client 同包可及;**必须先 Activate 全局激活**,
  ActivateNonDefault 只做 transport 挂载)+ 自定义 Responder 捕获 body/header。
  sendRequest 成功(HTTPCode/ResponseBody/服务端收到的 Bearer 头、自定义头、JSON 体)/
  HTTP 500 重试(默认 3 次 → 共 4 请求)/连接拒绝(显式全新 `&http.Transport{}` 绕开 mock,
  Windows 文案 "actively refused")。buildRequestBody 模板/默认双选、buildFromTemplate
  占位替换、setRequestHeaders 默认 CT/UA + 合并 + 非字符串 JSON 化、setAuthentication
  四分支 + 三类缺配置 + 未知类型、Send/SendWithDefaultConfig/SendNoticeAPI/SendSMS/
  SendWebhook/TestAPIConfig 六入口。ad_ldap Tier-1(D-79-04):dial-error、
  **真实 TCP 立即断开 → Bind 失败分支**、loadADTLSSkipVerify env 矩阵
  (同包重置 `adTLSSkipVerifyOnce` 白盒 + `SM4_KEY` 注入过 config.Validate + t.Cleanup 还原)、
  formatUsername 三形态、extractRDN/parseIntOrDefault/encodePassword UTF-16LE 字节断言、
  **16 个 wire 入口连接不可用守卫表**(wire 真路径不进本 phase,注释锁定)。

## Coverage checkpoint(per-file 实测,`go test -count=1 -coverprofile` 全包一次,403s)

| File | 基线(RESEARCH §2) | 实测 | 目标 | 结果 |
|---|---|---|---|---|
| mac_history_query_service.go | 8.0%(31/389) | **73.3%**(285/389) | ≥70% | ✅ |
| mac_collection_service.go | 28.5%(83/291) | **70.4%**(205/291) | ≥70% | ✅ |
| mac_history_service.go | 62.7%(131/209) | **87.6%**(183/209) | ≥70% | ✅ |
| mac_history_partition.go | 0%(0/95) | **63.2%**(60/95) | ≥70%(组) | ⚠️ PG-only 锁,见下 |
| mac_history_matview_service.go | 0%(0/30) | **86.7%**(26/30) | ≥70% | ✅ |
| mac_history_heatmap_service.go | 0%(0/52) | **82.7%**(43/52) | ≥70% | ✅ |
| mac_perf_config_seed.go | 0%(0/11) | **81.8%**(9/11) | ≥70% | ✅ |
| email_sender_service.go | 14.3%(27/189) | **80.4%**(152/189) | ≥65% | ✅ |
| api_sender_service.go | 9.2%(11/119) | **95.8%**(114/119) | ≥75% | ✅ |
| ad_ldap_client.go | 0%(0/179) | **69.3%**(124/179) | ≥55% | ✅ |

- 10 文件合计:**283/1664(17.0%)→ 1201/1664(72.2%),净增 +918 covered**。
- mac 尾部五文件组(history+partition+matview+heatmap+seed):
  **321/397 = 80.9% ≥70%**(组口径达标)。其中 `mac_history_partition.go` 单文件 63.2%:
  未覆盖 35 stmts 全部在「真建分区」路径 —— `PARTITION OF` DDL 的实际 Exec、
  分区名 regexp 不匹配守卫(CR-01 防御性,内部生成的名称恒匹配)、
  `DropExpiredPartitions` 的 pg_inherits 行遍历与 DROP 分支 —— 均需真实 PostgreSQL
  分区表/R6 明令禁止,plan Task 5 自带「DDL 直接 Exec 不可拆 → 落 SUMMARY 不覆盖」条款。
- **包级滚动**:root 包 45.2%(79-04 后)→ **62.9%**(本 plan 后全包实测)。
  本 plan 段贡献 +918 covered,Phase 79 累计约 +2658(79-01 +155 / 79-02 +336 /
  79-03 +521 / 79-04 +728 / 79-05 +918)。
- SC-2 discharge:本 plan 触及的 10 文件无一 <50%(最低 mac_collection 70.4%)。

## 79-06 gap math(包级投影)

- root 包 5202 stmts,已覆盖 3271(62.9%),**距 70% gate 还差 370 covered**
  (3641 − 3271)——远低于 research 时的「需 +3053」估算,device 家族无需 DQ2 深度改造。
- 剩余 <70% 文件 unc 排布(79-06 主战场):
  device_discovery 278、device_info_collection 275、config_backup 244、
  device_monitor 189、config_execution 152、command_dispatch 112、
  device_credential_helper 47、ad_ldap 55(wire 深段)、mac_history_partition 35(PG-only)。
- 即:79-06 只要吃下 device 家族 ~30% 的可达面即达包级 70%;SNMP fake(DQ5)与
  ForTesting helper(DQ2)均为可选增益而非必要。

## 不可达清单(SUMMARY 记录,按决策锁定)

| 项 | 原因 | 处置 |
|---|---|---|
| email TLS 握手 happy path(`sendWithTLS`/`sendWithSTARTTLS` 握手段) | `InsecureSkipVerify:false` 硬编码(email_sender_service.go:203-204/:271-272) | D-79-03 锁定;dial-error 分支已覆盖;STARTTLS 未通告扩展的明文回退段为可达 happy 段,已覆盖 |
| ad_ldap wire 真路径(55 unc) | 无 iface;78-07 Conclusion B:BER fake 与 go-ldap/v3 不兼容 | D-79-04 Tier-1;16 wire 入口的连接不可用守卫 + 参数段 + 纯 helper 已覆盖 |
| mac PG-only 真执行(partition DDL Exec / matview REFRESH / heatmap MV 行 / purge PG SQL 分支) | sqlite 无 PARTITION OF / 物化视图;R6 禁真建 | 假方言放行到字符串构造行并断言 SQL 形态;Exec 必失败 |
| `queryConnectionStatsFromDB` 三段聚合映射(~35 unc) | 统计 SQL 用 `EXTRACT(EPOCH FROM ...)`/`::bigint`/`COUNT(*) FILTER` — PG 专属语法,sqlite 解析必失败 | QUIRK-79-05-D;校验分支 + 错误包装已覆盖 |
| `mac_collection.collectDeviceMAC` executor 真路径(~95 unc) | 需 `*device.DeviceExecutor`(字段跨包不可种子) | 归 79-06(D-79-05);nil-executor 边界已覆盖 |
| `getMACCommand` 防御兜底 / parseMACLine 锐捷 interfaceParts 空兜底 | 死代码(调用点保证非空) | 注释锁定,不追 |

## Quirks 处置(全部「只锁不修」,零生产改动;R7)

- **QUIRK-79-05-A** `MACPerfConfigCacheTTLSeconds` 不命中 `CacheConfigService.LoadConfigs`
  的「cache.%」/「rate_limit.%」LIKE 通道(cache_config_service.go:144/:151)→
  `perfCacheTTL` 在 perfConfig 非 nil 时恒回 30 分钟兜底、nil 时 5 分钟;
  键名 seconds(秒)与 GetDuration 分钟语义矛盾,但生产读不到该键,矛盾不可达。
  plan 里「种子 120 → 120*time.Second」的预期因此不可达,按现行为锁定。
- **QUIRK-79-05-B** `models.DeviceMACHistory` 无 Status 字段(模型不映射该列)→
  `QueryHistory` 的 `req.Status` 过滤生成 `status = ?` → sqlite 报 no such column →
  查询失败包装分支。
- **QUIRK-79-05-C** `getLongOccupancyThreshold` 用 `Row().Scan()`,无行时返回 driver 的
  `sql.ErrNoRows`,与 `err == gorm.ErrRecordNotFound`(:809)不成立 → 「缺配置 → (30,nil)」
  分支不可达,实际返回 (30, err);上层 QueryConnectionStats 以 Warnf + 回退 30 天兜住。
- **QUIRK-79-05-D** `queryConnectionStatsFromDB` 统计 SQL 为 PG 专属语法(见不可达清单)。
- **QUIRK-79-05-E** `GetMACAddressList` 非法 `orderByColumn` 经 ApplySort 白名单回退仅 warn,
  又因源码仅在 `orderByColumn == ""` 时补默认 `collected_at DESC` → 非法列退化为自然序
  (与 QUIRK-79-02-A / 79-03-B 同族)。
- **QUIRK-79-05-F**(⚠️ 现网可见)`collectDeviceMAC` 顶层 panic-recovery(:118-122)会把
  executor 异常(nil 指针解引用等)吞掉并使函数走 recover 返回 nil →
  `CollectAllDevices` 返回 `[]*MACCollectionResult{nil}`、`CollectDevice` 返回 (nil, nil),
  调用方无法区分"采集失败"与"无 MAC"。
- **QUIRK-79-05-G** `GetMACAddressList` 的部分 MAC 输入(如 "00:00:09")经
  `NormalizeMACAddress` 返回 "" → 拼出 `LIKE '%%'` → **全表匹配**且不报错;
  前端必须传完整 MAC。
- **QUIRK-79-05-H**(⚠️ 现网可见)`MergeFlappingRecords` 的分组键用
  `fmt.Sprintf("%v", hist.VLANID)` — VLANID 是 `*int`,`%v` 打印**指针地址**;
  每行读回都是独立指针 → 非 nil VLAN 的行永远各成一组,flapping 合并永不触发。
  修复应解引用(生产改动,先立项);测试以 nil-VLAN(可合并)与非 nil(不合并)双向证据化。
- **QUIRK-79-05-I** `parseMACLine` 华为分支在 interfaceParts 为空时兜底取 `fields[2]`
  且不走 NormalizeInterfaceName → 类型关键字(如 "Dynamic")可被原样当接口名入库。
- **QUIRK-79-05-J** `buildFromTemplate` 在 `Recipients` 为空时跳过 `{{recipients}}` 替换,
  占位符原样进入请求体。
- **QUIRK-79-05-K**(⚠️ 现网可见)`LDAPClient.Connect` 的 Bind 失败路径只调 `c.conn.Close()`
  (底层 ldap.Conn 方法),不把 `c.conn` 置 nil → `IsConnected()` 在 Connect 失败后仍返回
  true,上层若以它判定可用性会误判。
- **QUIRK-79-05-L** `formatUsername` 在 `DomainName` 为空时 `Split(".")[0] == ""` →
  返回 `"\user7905"`(前导反斜杠的非法账号形态)。
- **QUIRK-79-05-M** `APISenderService.Send` 的重试聚合返回值只带 Message/Error/RetryCount,
  末次尝试的 `HTTPCode`/`ResponseBody` 被丢弃(与 `TestAPIConfig` 直返 sendRequest 的形态不同)。
- **复录 79-04 QUIRK-79-04-D(同根再现)**:`EmailConfig.UseSSL/UseSTARTTLS` 的 false 与
  `NetworkDevice.Status = DeviceStatusOnline(0)` 都被 GORM 列 default(true/2)的零值跳过
  吞掉 → email 测试以 `Updates(map)` 回写布尔列、mac 测试以 Update 回写 status
  (测试侧规避,零生产改动)。
- **测试口径提示**:`DeviceMACHistory.AfterFind` 把读回的 timestamp 墙钟重塑为 Local
  (loc 变墙钟不变),因此与 UTC 构造的种子比 instant 会差时区偏移 —— 断言一律用
  `Format("2006-01-02 15:04:05")` 墙钟文本或宽窗(±24h 以上),禁窄时间窗。

## Deviations from Plan

1. **[环境] `go test -race` 本地不可执行**:Windows 本机 cgo 工具链故障
   (`cgo.exe: exit status 2`,与 79-01/02/03/04 SUMMARY Deviation 同源,改动前既有测试同样
   构建失败)。race 纪律由 t.Cleanup 全量防护(fake SMTP listener/goroutine、MemoryCache
   单次 Close)+ 禁 t.Parallel + mutex 保护的会话快照 + ci.yml Linux race job 兜底。
2. **[计划-实装口径] 三处按 plan 自带「以实现为准」条款调整**:
   (a) plan 预期 `perfCacheTTL` 「种子 120 → 120*time.Second」——实装键不在 LoadConfigs 的
   LIKE 通道,不可达,改锁现行为(QUIRK-79-05-A);
   (b) plan 预期 ad_ldap wire ops 有「未连接前置守卫分支」——实装无守卫、直接解引用
   (`c.conn.Search` 会对 nil conn panic),改以「真实 TCP 立即断开 → Bind 失败 → 底层
   conn 进入 closing 态」驱动同形错误分支(每入口断言 error 且不 panic);
   (c) plan 预期 `TestMhp7905_Partition_DDL_StringShape` 依赖「同包直调内部 builder」——
   DDL 为函数内联 `fmt.Sprintf`,不可拆 → 新增假方言(gorm dialector + 捕获 SQL 的失败
   ConnPool)实现同等的字符串形态断言(仍满足 R6,零真实 PG)。
3. **[Rule 3] mac_collection 拆两次 commit**:Task 4 首落 `6c2caab`(66.7%,未达 70%);
   借 Task 5 的假方言基建补 FilterRule stub / 关键字跳过分支 / DB 错误包装 → `aff0e32`
   (70.4%)。计划未列补差任务,按 Task 8「未达标回对应 task 补 unc」条款执行。
4. **[测试侧适配] fixture 修正若干**(均为测试自身 bug,Rule 1):
   NetworkDevice IP 唯一索引自增分配;CleanOldRecords 语义相对 time.Now → 种子改相对时刻;
   time 断言改墙钟文本(AfterFind 重塑 loc);excelize 取 sheet 改 `GetSheetList()`;
   httpmock 需 `Activate()` 先于 `ActivateNonDefault`;嵌套 `newApd7905` 的 t.Cleanup 会
   全局 Deactivate 污染后续用例 → 「无默认配置」用例前置;LDAP 测试的 accept goroutine
   以 stop channel + WaitGroup 收口(原 `<-acceptDone` 死锁)。

## Known gaps(剩余 463 unc of 1664,全部为 wire/PG-only/executor 专属,不追 100%)

- `mac_collection_service.go`(86 unc):collectDeviceMAC executor 真路径 → 79-06。
- `ad_ldap_client.go`(55 unc):wire 真路径(D-79-04)。
- `mac_history_partition.go`(35 unc):PG 分区真执行(R6)。
- `mac_history_query_service.go`(104 unc):queryConnectionStats 聚合映射(PG-only)+ 各
  DB 错误包装(sqlite 健康库不可达)。
- `mac_history_service.go`(26 unc):purge 备份行数不匹配/事务失败、merge 更新删除失败包装。
- `email_sender_service.go`(37 unc):TLS 握手成功段 + smtp 会话中段错误包装。
- `api_sender_service.go`(5 unc):`json.Marshal` 失败分支(map 值含不可序列化类型,
  由实现保证不触发)。
- `mac_history_heatmap/matview/seed`(9+4+2 unc):PG MV 行映射、Exec 成功路径。

## Acceptance criteria 对照

| 标准 | 结果 |
|---|---|
| TestMhq7905_ ≥14 / TestMcl7905_ ≥10 / mac 尾部 ≥11 / TestEml7905_ ≥11 / TestApd+Adl ≥9 | ✅ 14(43 子测试)/ 14(42)/ 12(36)/ 11(27)/ 11(25) |
| 文件 min_lines:520/400/360/420/380;contains(ExportHistory/parseRuijiePortSecurityLine/isPostgres/net.Listen/setAuthentication) | ✅ 874/744/785/719/697 行;contains 全命中 |
| mac query ≥70% / mac_collection ≥70% / email ≥65% / api_sender ≥75% / ad_ldap ≥55% | ✅ 73.3 / 70.4 / 80.4 / 95.8 / 69.3 |
| mac 尾部五文件 ≥70% | ✅ 组 80.9%(partition 单文件 63.2%,PG-only 锁定并落 SUMMARY) |
| ExportHistory "PK" magic + excelize.OpenReader 双重断言 | ✅ |
| fake SMTP 会话断言(mail from/rcpt/data);listener/goroutine 经 t.Cleanup 关闭 | ✅ |
| 两条 TLS dial-error 用例各含 InsecureSkipVerify 硬编码行号注释 | ✅(:203-204 / :271-272) |
| 16 个 wire 入口守卫表(grep 方法名计数 ≥15) | ✅ 16 个全部断言 error 且不 panic |
| env 矩阵用例 t.Cleanup 还原环境变量 | ✅(Unsetenv + Once 重置 + 缓存变量复位) |
| vendor/status 断言引用 models 具名常量;MACPerfConfig* 常量引用(禁裸配置键) | ✅(MACPerfConfigCacheTTLSeconds/HeatmapTopN、DeviceStatusOnline/Offline、Vendor*、NotificationConfigStatus*、APIConfigType*、AuthType*、MACType*) |
| 缓存命中可观察证据(删行后仍命中) | ✅ TestMhq7905_GetVendor_CacheAndDB |
| `go build ./...` == 0;`go vet` == 0 | ✅ |
| `go test -count=1 ./internal/services/` == 0 | ✅(全包 profile 跑 403.2s,62.9%) |
| `go test ./...` == 0 | 见 Self-Check |
| `-race` 抽样 | ⚠️ 本地 cgo 故障不可执行(Deviation #1) |
| 生产 .go 改动 = 0 | ✅(9 个 commit 全部 *_test.go / SUMMARY) |
| 无真实网络外联(全部 127.0.0.1 loopback;httpmock 拦截) | ✅(T-79-05-01:监听一律 127.0.0.1:0;唯一出网尝试为「连接拒绝」用例,指向已关闭 loopback 端口) |

## 手注(给 79-06)

- 可复用同包 helper:`newMhq7905`/`newMhq7905Cached`、`seedMhq7905`、`newMcl7905`、
  `mcl7905RuleStub`(FilterRuleService stub 范本)、`newMhs7905`、
  **`newMhs7905PGFake`/`mhs7905PGDialector`/`mhs7905PGPool`(假方言 + SQL 捕获,可复用于
  任何 PG-only 分支的字符串断言;注意必须 `callbacks.RegisterDefaultCallbacks`)**、
  `startFakeSMTP7905`(可编程 550 拒绝 / 硬断连)、`newApd7905`(httpmock 装配)、
  `newAdl7905`、`mhq7905Time`(固定时刻)。
- executor 家族(DQ2)如需 ForTesting helper,`collectDeviceMAC` 的 95 unc 是 mac 侧最大
  单点;其 panic-recovery 语义(QUIRK-79-05-F)在接入真 executor 后应复核断言。
- 剩余 gap math 见上文「79-06 gap math」:包级 70% 只差 370 covered,SNMP fake(DQ5)
  按 gap 裁决即可,无需为达标引入。
- gorm 白盒提示:假方言若不注册默认 callbacks,Exec/Scan 会静默 no-op(不报错)——
  这是本次调试最耗时的一课,已在 `newMhs7905PGFake` 注释。
- `-race` 在 ci.yml Linux job 兜底;本地 cgo 修复前勿在 Windows 上追。

## Self-Check: PASSED

- 文件存在:mac_history_query_service_79_05_test.go(874 行)/
  mac_collection_service_79_05_test.go(744 行)/ mac_history_tail_79_05_test.go(785 行)/
  email_sender_service_79_05_test.go(719 行)/ api_sender_ad_ldap_79_05_test.go(697 行)/
  79-05-SUMMARY.md — 全 FOUND。
- 提交存在:faea725 / 3829aad / 93fc19d / 6c2caab / aff0e32 / 7016343 / 8cdbfb2 /
  022b00c / ac86876 — 全 FOUND(git log --all);9 个 commit 仅含 *_test.go,
  生产 .go 改动 = 0(git show --stat 逐一核对)。
- `go build ./...` exit 0;`go vet ./internal/services/` exit 0;
  `go test -count=1 ./internal/services/` exit 0(全包 coverage profile 跑 403.2s,62.9%)。
- **`go test ./...` exit 0**(repo_full_test_79_05.log:REPO_TEST_EXIT=0,72 包 ok,
  FAIL 行计数 0)。
- `-race` 本地不可执行(cgo 工具链故障,Deviation #1)。
