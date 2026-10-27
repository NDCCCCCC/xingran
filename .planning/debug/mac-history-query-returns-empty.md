---
status: resolved
trigger: "MAC 轨迹查询无法查到任何 mac，在数据库里查是存在的"
created: 2026-06-30T00:00:00Z
updated: 2026-10-27T16:40:00+08:00
goal: find_root_cause_only
audit_acknowledged:
  milestone: v1.23
  at: 2026-10-28
  status: resolved

---

## Current Focus

hypothesis: 杞ㄨ抗鏌ヨ (POST /network/history/trajectory) 璧?`QueryMACTrajectory`
瀹為檯鍛戒腑鐨勬槸 PG 鍒嗗尯鏍硅〃 `sys_device_mac_history`锛屾甯稿簲琚?PG 璺敱鍒板瓙鍒嗗尯
`sys_device_mac_history_2026_06`锛涗絾 model 鏂囦欢缂?`deleted_at` + Raw SQL 涔熸病鏈変换浣?`deleted_at IS NULL` 杩囨护锛屾墍浠ヨ蒋鍒犻櫎鍦烘櫙涓嬭繕鑳藉懡涓€傞渶瑕佸湪 DB 鐪熷疄鐘舵€佷腑纭
锛堟槸鍚﹀瓨鍦?reconciliation/cleanup 浠诲姟鎶婂巻鍙叉爣杞垹锛夛紝鍚屾椂鎺掓煡 UPDATE/CREATE
鐨勫疄闄呭啓鍏ヨ矾寰勩€?
test: 鍦?PG 绔拡瀵?MAC `B0227A2E4A4F` 鍒嗗埆璺戯細
1. `SELECT count(*) FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F';`
2. `SELECT count(*) FROM sys_device_mac_history_2026_06 WHERE mac_address = 'B0227A2E4A4F';`
3. `SELECT column_name FROM information_schema.columns WHERE table_name = 'sys_device_mac_history' AND column_name LIKE '%delet%';`
4. 鐪?application 鏃ュ織涓?`[MAC杞ㄨ抗鏌ヨ] 鏌ヨ澶辫触` 鏄惁鍖呭惈 SQL 閿欒
5. 鐪?routine query 涓〃璺敱鏄惁璧?partition pruning

expecting: 鑻ュ瓙鍒嗗尯鏈?N 琛岃€屾牴琛ㄨ繑鍥?0 琛岋紝鍒欎腑闂村眰(RLS/permission/瑙嗗浘)鍚炴帀浜嗭紱
鑻ユ牴琛ㄨ繑鍥?0 鑰屽瓙鍒嗗尯鏈?N锛屽垯 GORM 鎴?Raw 瑙﹀彂 PG 鎶ラ敊锛涘惁鍒欎竴瀹氬瓨鍦ㄥ叾浠?杩囨护(濡?first_seen 鑼冨洿銆乵ac_address 澶у皬鍐欍€乨eleted_at)瀵艰嚧绌洪泦
next_action: 1) 妫€鏌?`internal/services/mac_history_service.go` 涓?scheduler
涓殑娓呯悊閫昏緫鏄惁瀵?`deleted_at` 璧嬪€硷紱2) 鐪?application 鏃ュ織鏄惁鏈?PG 鎶ラ敊
(`column ... does not exist` / `relation ... does not exist`)锛?) 鎺掓煡
Phase 39+ reconciliation / matview 閲嶅缓鏄惁鏀瑰彉鏁版嵁鍩虹嚎

## Symptoms

expected: 鍦ㄥ墠绔?杞ㄨ抗鏌ヨ"椤佃緭鍏?MAC `B0:22:7A:2E:4A:4F` 鑳芥煡鍒拌 MAC 鍦?`sys_device_mac_history_2026_06` 鐨勮嚦灏?1 琛?璁惧 `a8c30b8c-...` 鎺ュ彛
`GigabitEthernet 2/25` VLAN 308)

actual: API 杩斿洖绌哄垪琛?娌℃湁浠讳綍 mac)

errors: 鏃犳槑纭姤閿?鍙湁鍙兘 query 娌″尮閰嶅埌 row 璧扮┖鏁扮粍杩斿洖)

reproduction: 璋冪敤 `POST /network/history/trajectory`锛宐ody = `{"macAddress": "B0:22:7A:2E:4A:4F"}`锛?鎴栧湪 pg 绔洿鏌?`SELECT * FROM sys_device_mac_history WHERE mac_address='B0227A2E4A4F';`

started: 寰呯敤鎴风‘璁?鍩轰簬鐜扮姸鎶ュ憡锛屾湭缁欓娆″彂鐢熸椂闂?

## Eliminated

- hypothesis: A. 鍒嗗尯鏍硅〃 vs 瀛愬垎鍖鸿矾鐢遍棶棰?  evidence: PG 鍘熺敓鍒嗗尯琛?`FROM sys_device_mac_history` 浼氳嚜鍔?partition pruning
  鍒?`sys_device_mac_history_2026_06`(鍓嶆彁 first_seen 鍦ㄨ鏈堣寖鍥?銆俙QueryMACTrajectory`
  浣跨敤 `s.db.WithContext(ctx).Raw(sql, normalizedMAC, startTime, endTime)` 璧?  `first_seen >= ? AND first_seen <= ?`锛宲artition pruning 搴旇嚜鍔ㄥ惎鐢ㄣ€?  timestamp: 2026-06-30
  conclusion: 杩欐潯璺緞闇€瑕?DB 绔疄娴嬮獙璇侊紝浠ｇ爜灞傞潰娌￠棶棰?
- hypothesis: B. 琛ㄥ悕鎷兼帴閿欒(鏈堜唤鏍煎紡 `2026_06` vs `2026-06` vs `202606`)
  evidence: `mac_history_partition.go:60`銆乣migration_mac_history.go:65, 144, 176`
  涓夊浣跨敤 `fmt.Sprintf("sys_device_mac_history_%d_%02d", year, monthNum)` 涓€鑷村湴
  鐢熸垚 `sys_device_mac_history_2026_06`锛屼笌鐢ㄦ埛鎶ュ憡鐨勫疄闄呰〃鍚嶅尮閰嶃€係QL `Raw(sql, ...)`
  涔熺洿鎺ョ‖缂栫爜 `FROM sys_device_mac_history`锛屼笉璧拌繍琛屾椂鎷兼帴銆?  timestamp: 2026-06-30
  conclusion: 琛ㄥ悕闂鍙帓闄?
- hypothesis: C. MAC 鍦板潃澶у皬鍐?鏍煎紡杩囨护
  evidence: `QueryMACTrajectory` 鏄惧紡 normalize锛氱Щ闄?`.`銆乣:`銆乣-`锛岃浆 `ToUpper`
  (lines 849-852)銆傜敤鎴风粰鐨?MAC `B0:22:7A:2E:4A:4F` 鈫?鏍囧噯鍖栦负 `B0227A2E4A4F`銆?  鏁版嵁搴撴牱鏈€间篃鏄?`B0:22:7A:2E:4A:4F` 褰㈠紡锛屾爣鍑嗗寲鍚庝細鍖归厤銆侱B schema
  `mac_address VARCHAR(30)` 娌℃湁 COLLATE 闄愬埗锛屼絾浠ｇ爜宸茬粺涓€澶у啓銆?  timestamp: 2026-06-30
  conclusion: MAC 鏍煎紡鍖归厤鍙帓闄?
- hypothesis: D. 鏃堕棿鑼冨洿澶弗
  evidence: 鏍锋湰鏁版嵁 `first_seen = 2026-06-29 12:46`銆俙QueryMACTrajectory` 榛樿
  `startTime = time.Now().Add(-30 * 24 * time.Hour)` 鍗?2026-05-31锛沗endTime = time.Now()`
  鍗?2026-06-30(褰撳墠)銆傛牱鏈椂闂?2026-06-29 鍦ㄨ寖鍥村唴锛屽簲琚懡涓€?  timestamp: 2026-06-30
  conclusion: 鏃堕棿绐楀彛鍙帓闄?

## Evidence

- timestamp: 2026-06-30
  source: internal/services/mac_history_query_service.go:842-915
  finding: |
    `QueryMACTrajectory` 璧?`Raw(sql, normalizedMAC, startTime, endTime)` 鐩存帴
    瀵?`sys_device_mac_history` 鍙?SQL锛學HERE 鏉′欢 = `mac_address = ? AND first_seen >= ? AND first_seen <= ?`銆?    娌℃湁 `deleted_at IS NULL`銆佹病鏈?ORDER BY 瀛愬彞涔嬪鐨勮繃婊わ紝娌℃湁 LIMIT/OFFSET銆?    鏁翠釜鏂囦欢涓嶅嚭鐜?`deleted_at` / `DeletedAt` 鍏抽敭瀛椼€?
- timestamp: 2026-06-30
  source: internal/models/device_mac_history.go:21-46
  finding: |
    `DeviceMACHistory` struct 瀛楁锛歚ID`, `DeviceID`, `DeviceNameSnapshot`,
    `MACAddress`, `InterfaceName`, `VLANID`, `EventType`, `FirstSeen`,
    `LastSeen`, `CollectedAt`, `CreatedAt` 鈥斺€?**娌℃湁 `DeletedAt` / `deleted_at` 瀛楁**銆?    `TableName()` 杩斿洖甯搁噺 `"sys_device_mac_history"`銆?
- timestamp: 2026-06-30
  source: internal/core/db/migrations/migration_mac_history.go:33-47
  finding: |
    `CREATE TABLE IF NOT EXISTS sys_device_mac_history` DDL 鍒楁竻鍗曪細
    `id`, `device_id`, `device_name_snapshot`, `mac_address`, `interface_name`,
    `vlan_id`, `event_type`, `first_seen`, `last_seen`, `collected_at`,
    `created_at DEFAULT CURRENT_TIMESTAMP` 鈥斺€?**娌℃湁 `deleted_at` 鍒?*銆?    琛ㄨ澹版槑涓?`PARTITION BY RANGE (first_seen)`銆?
- timestamp: 2026-06-30
  source: internal/core/db/migrations/migration_mac_history.go:62-86
  finding: |
    杩佺Щ鏃朵负 now / now+1m / now+2m 鍚勫垱寤轰竴涓湀搴﹀垎鍖猴細
    `CREATE TABLE IF NOT EXISTS sys_device_mac_history_%d_%02d PARTITION OF ...`
    鐢ㄦ埛鎶ュ憡鐨?`sys_device_mac_history_2026_06` 涓庢鍛藉悕涓€鑷达紱2026-06-30 鐨?now
    鍒涘缓鐨勫氨鏄?`sys_device_mac_history_2026_06`銆傛ā寮忓尮閰嶇‘璁ゃ€?
- timestamp: 2026-06-30
  source: internal/services/mac_history_partition.go:50-93
  finding: |
    `CreateMonthlyPartition` 涓庤縼绉婚€昏緫绛変环锛氱敓鎴?`sys_device_mac_history_YYYY_MM`锛?    骞剁敤 `FOR VALUES FROM ('YYYY-MM-01') TO ('YYYY-MM-01')`銆?    楠岃瘉鍒嗗尯鍚嶆牸寮忔鍒欙細
    `^[a-z]+(?:_[a-z]+)*_[0-9]{4}_[0-9]{2}$` 鈥斺€?`sys_device_mac_history_2026_06` 閫氳繃銆?
- timestamp: 2026-06-30
  source: internal/core/db/migrations/migration_175_reconciliation_physical_link.go:69
  finding: |
    璇ヨ縼绉绘枃妗ｆ敞閲婂０绉帮細
    `sys_device_mac_history` 鎵嶆槸甯?`deleted_at` 鐨勮蒋鍒犻櫎琛?    (瑙?internal/models/device_mac_history.go)銆?    浣嗗疄闄?model 鏂囦欢(migration_mac_history.go 鐨?DDL 涓?device_mac_history.go
    鐨?struct)閮芥病鏈?deleted_at 瀛楁鈥斺€?*娉ㄩ噴涓庡疄闄?schema 涓嶄竴鑷?*銆傞渶鍒?DB 绔?    楠岃瘉 `information_schema.columns` 鐪嬫槸鍚︾湡鏈?`deleted_at` 鍒椼€?
- timestamp: 2026-06-30
  source: internal/api/v1/network/mac_history_router.go:11-32
  finding: |
    璺敱琛細`POST /history/trajectory` 鈫?`historyHandler.QueryTrajectory` 鈫?    `h.historyQueryService.QueryMACTrajectory`銆?    鏈嶅姟鏋勯€?`NewMACHistoryQueryService(core.GetDB())` 涓嶅甫 cache(NoCache 璺緞)銆?

## Resolution

root_cause: **TBD 鈥?闇€瑕?DB 绔疄娴?*銆備唬鐮佸眰闈㈡病鏈夊彂鐜颁細瀵艰嚧绌洪泦鐨勯€昏緫缂洪櫡銆?鏈€鍙兘浠嶉渶鍦?PG 绔疄娴嬩互涓嬪洓绉嶅満鏅細

(a) `deleted_at` 鍦?DB 瀹為檯 schema 涓瓨鍦?鐢卞悗缁?reconciliation 杩佺Щ娣诲姞)锛?浣嗘墍鏈夊巻鍙?row 宸茶鏍囪 `deleted_at IS NOT NULL`锛屼絾 `QueryMACTrajectory` 娌?鏈夎繃婊よ瀛楁 鈫?鍛戒腑鍗宠繑鍥炵┖銆傝繖鏄枃妗ｆ敞閲?vs 瀹為檯 schema 涓嶈嚜娲界殑鍙枒鐐?(`migration_175` 娉ㄩ噴鑷浉鐭涚浘)銆?
(b) 鐢ㄦ埛缁欑殑鏍锋湰 MAC `B0:22:7A:2E:4A:4F` 鍦?DB 涓疄闄呮槸 `b0:22:7a:2e:4a:4f`
鎴栧寘鍚┖鏍?涓嶅彲瑙佸瓧绗︼紱闇€ `SELECT mac_address, length(mac_address) FROM sys_device_mac_history WHERE ...`銆?
(c) PG 绔繍琛岃 Raw 鏌ヨ鏃跺疄闄呮姤 `column ... does not exist`(column name
涓嶅湪 whitelist/permission schema 鍐?锛宎pplication 鍚?error 杩斿洖绌恒€傚缓璁細
鐪?backend application 鏃ュ織涓?`[MAC杞ㄨ抗鏌ヨ] 鏌ヨ澶辫触: ...` 鏄惁鏈夊叿浣?PG 閿欒銆?
(d) 鏁版嵁琚?partition prune 閿欓厤锛氱敤鎴风殑鏍锋湰鍒嗗竷鍦?`sys_device_mac_history_2026_06`
浣?`first_seen` 瀹為檯鏄?UTC 涔嬪鏃跺尯鎴栧瓧绗︿覆锛孭G 璺敱涓嶅埌銆傞渶瑕?`SELECT pg_partition_tree('sys_device_mac_history');` 涓?`EXPLAIN SELECT ... FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F' AND first_seen >= ...`

fix: **涓嶅湪鏈疆浠诲姟鑼冨洿**(鍙瘖鏂?銆傚彲鑳戒慨澶嶆柟鍚戯細
- 濡傛灉 (a)锛氬湪 `QueryMACTrajectory` SQL 鍔?`AND deleted_at IS NULL`
  (鍓嶆彁 DB 鐪熸湁璇ュ垪锛屼笖鍒犻櫎绛栫暐纭疄鏍囪蒋鍒?
- 濡傛灉 (b)锛氭牎瀵规暟鎹湰韬?鍙兘 collector 鍐欏叆浜嗘煇绉嶉潪棰勬湡鏍煎紡)
- 濡傛灉 (c)锛氶噸鍐?Raw 涓?GORM chainable 澶勭悊閿欒浼犳挱锛屽苟鍦?handler 涓?  杩斿洖 500 鑰岄潪鍚?error
- 濡傛灉 (d)锛氬湪 GORM Raw 鍓?wrap `SET TIME ZONE 'UTC'` 鎴?cast 鏃堕棿

verification: 缂哄け鐨?DB 绔疄娴嬶細
1. `SELECT column_name FROM information_schema.columns WHERE table_name = 'sys_device_mac_history' AND column_name = 'deleted_at';`
2. `SELECT count(*) FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F';`
   vs 瀛愬垎鍖哄悓鏌ヨ
3. `EXPLAIN (ANALYZE, BUFFERS) SELECT ... FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F' AND first_seen >= '2026-05-31T00:00:00Z' AND first_seen <= '2026-06-30T23:59:59Z';`
4. backend application 鏃ュ織涓?`[MAC杞ㄨ抗鏌ヨ] 鏌ヨ澶辫触:` 鎴?   `[MAC鍘嗗彶鏌ヨ] 鏌ヨ璁板綍澶辫触:` 鏄惁鏈?PG 閿欒鐮?

files_changed: []

---

## 鐢ㄦ埛瀹炴祴鍥炲～锛?026-10-27锛孭G 鐢熶骇搴擄級

鐢ㄦ埛鎵ц浜嗘竻鍗?B 鐨?4 鏉?SQL锛坧g 瀹㈡埛绔埅鍥撅級锛?
- **鍒嗗尯鏍戯紙pg_partition_tree锛?*锛氱埗琛?`sys_device_mac_history` + **2026_06 ~ 2026_12 鍏?7 涓湀搴﹀垎鍖?*锛堟棤鏇存棭鍒嗗尯锛夈€?- **EXPLAIN ANALYZE**锛坢ac_address='B0227A2E4A4F' AND first_seen 2026-05-31~2026-06-30锛夛細鍒嗗尯鍓灊姝ｇ‘锛堝彧鎵?2026_06 鍒嗗尯鐨?`device_id_mac_address_first_idx1` 绱㈠紩锛夛紝Index Actual Rows: 6锛孭lanning 29.8ms / Execution 0.105ms銆?- **鏍锋湰 count**锛歚B0227A2E4A4F` 鍦ㄥ叏琛ㄤ笌 2026_06 鍒嗗尯鐨?count(*) 鎴浘鍧囨樉绀?**count=0**銆?- **deleted_at 鍒楀瓨鍦ㄦ€?*锛坕nformation_schema 鏌ヨ锛夛細鎴浘 tab 鏈兘鍒嗚鲸璇ユ煡璇㈢粨鏋溾€斺€?*寰呯‘璁?*銆?

### 闃舵鎬х粨璁猴紙濡傚疄璁板綍锛屼笉杩囧害鎺ㄦ柇锛?

- 鍒嗗尯鍓灊涓庣储寮曟湰韬伐浣滄甯革紙EXPLAIN 璇佹嵁锛夈€?- 鏍锋湰 MAC `B0227A2E4A4F` 褰撳墠鍏ㄨ〃 count=0锛屼袱绉嶅彲鑳斤細
  1. 6 鏈堝垎鍖烘竻鐞嗙瓥鐣ュ凡鎶婅鏍锋湰娓呴櫎锛堢敤鎴锋鍓嶉璀﹁繃姝ゅ彲鑳斤級锛?  2. 瀛樺偍鏍煎紡涓庢煡璇㈠瓧闈㈤噺涓嶄竴鑷达紙鏈。鍘熷鐤戠偣锛夈€?- **鍥?EXPLAIN 鐨?Index Actual Rows=6 涓?count=0 骞跺瓨锛屼笉鑳芥帓闄ゆ煡璇㈠瓧闈㈤噺鏍煎紡涓庡瓨鍌ㄦ牸寮忎笉涓€鑷?*锛堣繖姝ｆ槸鏈。鍘熷鐤戠偣锛夈€?

### Open follow-up

1. 鎹竴涓綋鍓嶇‘瀹炲瓨鍦ㄧ殑鏍锋湰 MAC锛堢敤鎴烽渶鎻愪緵锛夛紝澶嶆煡銆孌B 鐩存煡鏈夎 / 搴旂敤 API 杩斿洖绌恒€嶆槸鍚︿粛澶嶇幇銆?2. 纭 `deleted_at` 鍒楁槸鍚﹀瓨鍦紙information_schema 鏌ヨ B1 閲嶈窇涓€娆★級銆?

鍦ㄦ嬁鍒版柊鏍锋湰鍓嶏紝鏈。淇濇寔 **investigating**銆?
---

## 鏍瑰洜纭锛?026-10-27 绗簩鎵瑰疄娴嬶紝PG 鐢熶骇搴擄級

鐢ㄦ埛鎵ц绗簩鎵?SQL锛堢幇瀛樻牱鏈娊鏍?+ 鏍煎紡瀵圭収锛夛紝缁撴灉锛?
| 楠岃瘉椤?| 缁撴灉 |
|--------|------|
| `sys_device_mac_history` 鏈€鏂?5 琛?| **瀛樺偍鏍煎紡 = 澶у啓鍐掑彿鍒嗛殧**锛堝 `00:E0:FC:09:BC:F9`銆乣A4:0C:66:2C:E9:44`锛宖irst_seen 2026-10-27 16:02锛?|
| `sys_device_mac_address` 鏈€鏂?5 琛?| 鍚屼负澶у啓鍐掑彿鏍煎紡 |
| `B0227A2E4A4F`锛堟棤鍒嗛殧锛?| **0 琛?* |
| `B0:22:7A:2E:4A:4F`锛堝啋鍙凤級 | **53 琛?* |
| `deleted_at` 鍒?| **涓嶅瓨鍦?*锛坕nformation_schema count=0锛宮igration_175 娉ㄩ噴鐭涚浘鍧愬疄锛?|

### ROOT CAUSE锛堢‘璁わ級

**鏌ヨ绔妸 MAC normalize 鎴愭棤鍒嗛殧澶у啓锛坄QueryMACTrajectory` lines 849-852锛氱Щ闄?`.`銆乣:`銆乣-` + ToUpper锛夛紝鑰?DB 瀛樺偍鏄ぇ鍐欏啋鍙峰垎闅旀牸寮忊€斺€旂瓑鍊煎尮閰?`mac_address = 'B0227A2E4A4F'` 姘歌繙绌洪泦銆?*

- 鍦烘櫙 (b) 鍛戒腑锛氭暟鎹病闂锛屾槸**鏌ヨ瀛楅潰閲忎笌瀛樺偍鏍煎紡涓嶄竴鑷?*
- 鍦烘櫙 (a) 璇佷吉锛歚deleted_at` 鍒楁牴鏈笉瀛樺湪锛屾棤杞垹杩囨护鍙█
- 涓婃柟 Eliminated C 鐨勭粨璁?*閿欒**锛氬叾鎺ㄧ悊銆屾暟鎹簱鏍锋湰鍊间篃鏄?`B0:22:7A:2E:4A:4F` 褰㈠紡锛屾爣鍑嗗寲鍚庝細鍖归厤銆嶈嚜鐩哥煕鐩锯€斺€攏ormalize 鎴?`B0227A2E4A4F` 鍚庝笌鍐掑彿瀛樺偍**姘镐笉鐩哥瓑**銆傚綋骞翠粎鍑唬鐮佹帹鐞嗘帓闄わ紝鏈粡 DB 瀹炴祴锛屾湰杞疄娴嬫帹缈?- 闄勫甫鍙戠幇锛欵XPLAIN銆孖ndex Actual Rows=6銆嶄笌 count=0 骞跺瓨鐨勭枒鍥篃瑙ｅ紑鈥斺€斾袱鏉?SQL 瀛楅潰閲忎笉鍚岋紙EXPLAIN 鐗堟湰鐢ㄧ殑鏉′欢涓?count 鐗堟湰瀛樺湪宸紓锛夛紝浠ユ湰杞笁鏍煎紡瀵圭収涓哄噯

### fix 鏂瑰悜锛圥hase 72/73 钀藉湴锛?

鍙屼晶褰掍竴鍖栫粺涓€锛堟帹鑽愨憼锛夛細
1. **鏌ヨ渚?*锛歋QL 绛夊€兼潯浠舵敼涓哄瀛樺偍鍊煎仛鍚屾牱 normalize锛屽 `WHERE REPLACE(REPLACE(mac_address, ':', ''), '-', '') = ?`锛堟垨 PG 渚?`regexp_replace(upper(mac_address), '[^0-9A-F]', '', 'g') = ?`锛夛紝浣夸换鎰忚緭鍏ユ牸寮忓潎鍙懡涓?2. 鍐欎晶 normalize锛圖BG-03 鐨?Go 渚ч槻绾匡級浜у嚭鏍煎紡涓庢煡璇晶涓€鑷达紝鏉滅粷鏂版牸寮忔紓绉?3. 椤哄甫淇锛歊aw SQL 鈫?GORM chainable + 閿欒浼犳挱锛堢┖ 鈫?500锛屼笉鍐嶅悶 error锛夆€斺€擱EQ DBG-06 鐨勬棦瀹氭柟妗?4. 鏂囨。淇锛歮igration_175 娉ㄩ噴澹扮О銆屽甫 deleted_at 鐨勮蒋鍒犻櫎琛ㄣ€嶄笌瀹為檯 schema 涓嶇锛岄殢淇涓€骞舵洿姝ｆ敞閲?

### 鐜板瓨鏍锋湰锛堜緵淇鍚庨獙璇侊級

- `A4:0C:66:2C:E9:44`锛坉evice `96aab2e4-65c5-464f-9ee2-c4645d8ae868`锛宨face GE0/40锛宖irst_seen 2026-10-27 16:02锛?- `00:E0:FC:09:BC:F9`锛坉evice `48d75b51-c495-4cd2-a3f8-2babbe396c84`锛宨face GE0/14锛?

status 鍗囩骇涓?**root_cause_identified**锛屽緟 Phase 73 淇 + 鐢ㄤ笂鏍锋湰鍥炲綊楠岃瘉鍚庡叧闂€?

## Resolution 补记（2026-10-28，Phase 73 修复落地）

修复 commit `da984927`：双侧归一化双分支 SQL（`mac_address = ? OR REPLACE(REPLACE(REPLACE(UPPER(mac_address),':',''),'-',''),'.','') = ?`）+ 5 处等值匹配改造 + Raw SQL 错误传播 + migration_175 注释更正。回归测试 5 个（含脏格式存储反向用例与失败传播用例，红绿验证通过）。**本档关闭。**
