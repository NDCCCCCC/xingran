---
status: resolved
trigger: >-
  "鍒嗗叕鍙告湰閮ㄥ伐浣?3f130锛屽湪鍏剁粦瀹氱殑淇℃伅鐐?缃戠粶璁惧绔彛 CX-WH-WH-04F-FL-RS8607E-01 GE2/6 涓婃樉绀烘湁鐗╃悊鍦板潃锛圡AC锛夛紝浣嗘槸宸ヤ綅绠＄悊涓?鐗╃悊閾捐矾璁惧"瀛楁涓虹┖銆傝瀹氫綅鏍瑰洜銆?
created: 2026-07-21
updated: 2026-10-27
session_type: observation
goal: find_root_cause_only
related_sessions:
  - workstation-physical-link-zero.md (5F003 鈥?2026-06-30 宸叉牴鍥犲畾浣?
  - info-point-port-drift-recheck-20260701.md (1247/1483 鍘嗗彶婕傜Щ)
  - reconciliation-normalized-mv-workstation-id-missing.md (R5 MV 淇)
audit_acknowledged:
  milestone: v1.23
  at: 2026-10-26T10:00:00+08:00
  status: resolved

---

# Debug Session: 宸ヤ綅 3f130 鐗╃悊閾捐矾璁惧涓虹┖

## Current Focus

hypothesis: 5F003 淇锛坈ommit 6575a1e7锛夊凡鍚堝叆涓诲共锛岀Щ闄?strict device_id JOIN 骞舵敼鐢?ip.device_id 浣?MAC 閿氱偣銆?f130 浠嶆姤鍚屾牱鐥囩姸鐨勬渶鍙兘鍘熷洜鏄?5F003 淇鍦烘櫙涔嬪鐨?*鍓╀綑 4 涓柇鐐?*涔嬩竴锛?
1. `ops_info_points.workstation_id` 鎸囧悜鐨勫伐浣嶄笉鏄?3f130锛堢粦瀹氶敊浣?/ 鍚屽悕 id 璇敤 / migration_181 鏈簲鐢級
2. `ops_info_points.workstation_id` 鍦?sys_workstation 涓嶅瓨鍦?/ 琚蒋鍒犻櫎锛坰ys_workstation.deleted_at IS NULL锛?3. 宸ヤ綅 3f130 鐨?`user_id` 涓虹┖锛坙ine 339 鏃╅€€鍒嗘敮 鈥?瀵归綈 5F003 鐨?H1 宸茬煡鍙兘鎬э級
4. `sys_network_device` 琛ㄤ腑 CX-WH-WH-04F-FL-RS8607E-01 涓嶅瓨鍦?/ 琚蒋鍒犻櫎锛坙ine 404 EXISTS 杩囨护锛?
next_action: 鐢ㄦ埛鎻愪緵涓嬪垪 SQL 鏌ヨ缁撴灉鍚庡嵆鍙垽瀹氾紙鏃犻渶杩涗竴姝ラ潤鎬佸垎鏋愶級

expecting: 4 涓亣璁?H1-H4 浜掓枼锛岄獙璇?SQL 浜掔浉鎺掓枼锛涘悓鏃跺彲鎺掗櫎 5F003 鍚屾簮锛坉evice_id 婕傜Щ锛夆€?鍥犱负 6575a1e7 宸插悎骞躲€?
## Symptoms

expected: 宸ヤ綅 3f130锛堝垎鍏徃鏈儴锛変笅"鐗╃悊閾捐矾璁惧"搴旀樉绀鸿宸ヤ綅鐢ㄦ埛瀹為檯鎺ュ叆鐨勮澶囷紙CX-WH-WH-04F-FL-RS8607E-01 GE2/6 涓婄殑 MAC锛?actual: 宸ヤ綅绠＄悊椤?鐗╃悊閾捐矾璁惧"闈㈡澘鏄剧ず"璇ュ伐浣嶆殏鏃犻€氳繃 MAC鈫抪ort鈫抜nfoPoint 鍙嶆帹鍒扮殑璁惧"鎴?(0鍙?
errors: 鍓嶇 `/ops/workstation-device/{id}/physical` 杩斿洖 `data: []`
reproduction: 宸ヤ綅绠＄悊 鈫?鎵惧埌宸ヤ綅 3f130 鈫?鐗╃悊閾捐矾鎶樺彔闈㈡澘
started: 2026-07-21 鎶ュ憡
codebase_evidence: 5F003 淇 (commit 6575a1e7) 宸插悎鍏ヤ富骞?`workstation_device_service.go:380-405` CTE 涓嶅啀鏈?strict device_id JOIN

## Eliminated

- hypothesis: 5F003 鍚屾簮 device_id 婕傜Щ 鈥?evidence: commit `6575a1e7 fix(operations): 5F003 宸ヤ綅鐗╃悊閾捐矾 0 鍙拌澶嘸 鍦?HEAD锛沗workstation_device_service.go:391-403` 宸叉敼涓轰粎鎸?`port.id = ip.port_id` 閿氬畾鍗曚竴绔彛锛宍ip.device_id AS effective_device_id` 浣?MAC JOIN 閿氱偣锛宭ine 404 鍔?`EXISTS(sys_network_device)` 闃?orphan銆俤73c0984 pause commit 鏈湪鏈垎鏀畫鐣?鈫?ELIMINATED
  timestamp: 2026-07-21
- hypothesis: ClassifySignals.hasPhysical 姘歌繙 false (R1/R2 纭紪鐮?NULL) 鈥?evidence: `reconciliation_detection.go:105` 鐜拌 `row.PhysicalUserID != nil && *row.PhysicalUserID != ""`锛沵igration_175 + 176 宸插缓鐪熺墿鐞嗛摼璺?LEFT JOIN锛沵igration_181 MV 宸ヤ綔绔?ID 淇宸插悎骞?鈫?ELIMINATED
  timestamp: 2026-07-21
- hypothesis: 鎺ュ彛鍚嶅綊涓€鍖栦笉鍖归厤 鈥?evidence: port_collector.BeforeCreate 涓?mac_address.BeforeCreate 鍧囪皟 normalize.InterfaceName/MACAddress锛坧ort-mac-format-unify 2026-07-01 宸插悎骞讹級锛沵igration_186 褰掍竴鍖栧巻鍙叉暟鎹紱CTE 涓や晶鐢ㄧ浉鍚岀殑 `^(gigabitethernet|gigabitether|ge|gi)` regex 褰掍竴 鈫?ELIMINATED
  timestamp: 2026-07-21
- hypothesis: 鐗╁寲瑙嗗浘 reconciliation_normalized 鏁版嵁闄堟棫 鈥?evidence: GetPhysicalDevices 鏄疄鏃?SQL CTE锛屼笉璇?MV锛涘彧鏈?ops_asset_physical 琛ㄦ垨瀵硅处 HealthBadge 鎵嶇敤 MV 鈫?ELIMINATED
  timestamp: 2026-07-21

## Evidence

- timestamp: 2026-07-21
  checked: git log --all -- internal/services/operations/workstation_device_service.go
  found: HEAD 涓婃渶杩戠浉鍏?commit `6575a1e7 fix(operations): 5F003 宸ヤ綅鐗╃悊閾捐矾 0 鍙拌澶?鈥?MAC JOIN 閿氱偣鏀圭敤 ip.device_id` (2026-07-01) 宸插悎鍏ワ紱`460d95cd fix(backend): R5 鐗╃悊閾捐矾 鈥?鍘嗗彶 MAC fallback + MergeByTransitions 鍏滃簳` (2026-07-02) 绱ч殢鍏跺悗
  implication: 5F003 绫荤棁鐘剁殑浠ｇ爜璺緞宸蹭慨澶嶏紱3f130 涓嶆槸鍚屾簮闂

- timestamp: 2026-07-21
  checked: workstation_device_service.go:320-404 GetPhysicalDevices CTE
  found: 
    - line 339: `if workstation.UserID == nil || *workstation.UserID == ""` 鏃╅€€杩斿洖 `[]`
    - line 401-403: WHERE 瀛愬彞 `ip.workstation_id = ? AND ip.deleted_at IS NULL AND ip.status = 0`
    - line 404: `AND EXISTS (SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)` 闃?orphan
  implication: CTE 鍐呮湁 3 閬撹繃婊ゅ彲鐙珛杩斿洖 0 琛岋細(a) workstation.UserID 绌?(b) workstation_id 涓嶅尮閰?3f130 (c) ip.device_id 涓嶅湪 sys_network_device

- timestamp: 2026-07-21
  checked: workstation_device_service.go:472-489 MAC/asset JOIN
  found: 
    - `LEFT JOIN latest_mac mac ON mac.norm_iface = wp.norm_iface AND mac.device_id::text = wp.effective_device_id::text`
    - `LEFT JOIN ops_asset a ON (a.mac1/mac2 normalized = COALESCE(mac.norm_mac, hist.norm_mac))`
  implication: 鍗充娇 workstation_ports 鏈夎锛孧AC 娌￠噰闆嗗埌 GE2/6 涔熻繑鍥?0 琛岋紱浣嗙敤鎴峰凡纭绔彛**鏈?* MAC锛屾墍浠ユ鏂偣涓嶅お鍙兘

- timestamp: 2026-07-21
  checked: info_point handler/service 鏍￠獙
  found: infopoint_service.go Create/Update 涓嶅仛 device_id 鈫?port_id 涓€鑷存€ф牎楠岋紱infopoint_handler.go:69 鐩存帴缁戝畾 JSON銆?575a1e7 commit msg 澹扮О"infopoint_handler Create/Update 鍔?device_id鈫攑ort_id 涓€鑷存€ф牎楠?锛屼絾 main 鍒嗘敮 infopoint_handler.go 鏄惁瀹為檯搴旂敤寰呯‘璁?  implication: REST API 璺緞浠嶅彲鍐欓敊 device_id锛涗絾鍗充娇鍐欓敊锛屽洜 `EXISTS(sys_network_device)` 鍏滃簳锛宱rphan 琛屼細琚繃婊ゆ帀锛屼笉褰卞搷 MAC JOIN锛圡AC JOIN 鐢?`ip.device_id` 閿氱偣鑰岄潪 `port.device_id`锛?
- timestamp: 2026-07-21
  checked: ops_info_points.WorkstationID 绫诲瀷
  found: `gorm:"size:64"` 鈥斺€?VARCHAR(64)锛屽瓨 UUID 瀛楃涓?  implication: workstation_id 鏄瓧绗︿覆鑰岄潪 UUID 绫诲瀷锛汣TE `ip.workstation_id = ?` (param 鏄?workstationID 瀛楃涓? 鐩存帴鏂囨湰姣旇緝锛涘 3f130 鐨?workstationID 鏄?UUID 瀛楃涓诧紝涓よ€呭繀椤荤簿纭浉绛?
## Resolution

root_cause: **H3 纭锛?026-10-27 搂4.5 缁煎悎璇婃柇瀹炴祴锛夆€斺€斿伐浣?3f130 鏈粦瀹?user_id**銆俙workstation_device_service.go:339` 鐨勬棭閫€鏂偣鍛戒腑锛歎serID 绌?鈫?鐩存帴杩斿洖 `[]`锛岀墿鐞嗛摼璺睍绀烘寜銆岀敤鎴封啋鐗╃悊閾捐矾銆嶇淮搴﹁蛋锛屾湭缁戠敤鎴峰嵆绌恒€?
### 缁堢増瀹炴祴锛?026-10-27锛孭G 鐢熶骇搴?搂4.5 淇鐗堬級

| 璇绘暟 | 鍊?| 鍒よ |
|------|-----|------|
| ws_user_id / ws_user_name | **NULL** | **H3 鍛戒腑锛氭湭缁戝畾鐢ㄦ埛** |
| active_info_points | 1 | H2 鎺掗櫎锛堜俊鎭偣鍦ㄣ€佺姸鎬?0銆佹湭鍒狅級 |
| ip_bound_to_valid_device | 1 | H1/H4 鎺掗櫎锛堢粦瀹氭纭笖璁惧瀛樺湪锛?|
| mac_match_count | **1** | **MAC 鏁版嵁閾捐矾瀹屽叏閫?*锛堣澶?鎺ュ彛褰掍竴鍖栧尮閰嶆垚鍔燂級 |

鎺掓煡杩囩▼澶囨敞锛堜緵鍚庣画鍚岀被闂澶嶇敤锛夛細
- `sys_workstation` 鏃?`code`/`name` 鍒楋紝宸ヤ綅鍚嶅疄闄呭垪 = `workstation_name`
- `ops_info_points.workstation_id` 涓?varchar锛屼笌 `sys_workstation.id`(uuid) 姣旇緝闇€ `id::text` 鏄惧紡杞崲
- `ops_info_points` 鏃?`interface_name`锛屾帴鍙ｅ悕璧颁俊鎭偣鑷甫 `port_name` 蹇収锛屽綊涓€鍖栧悗涓?`sys_device_mac_address.interface_name` 鍖归厤鎴愬姛

fix 鏂瑰悜锛圥hase 73 钀藉湴锛屽搴?REQ DBG-07锛夛細
1. **鏁版嵁淇**锛氱粰 3f130 缁戝畾 user_id锛堜笟鍔″姩浣滐紝闇€涓庝汉浜?鐢ㄦ埛鏍稿搴旂粦璋侊級
2. **鍙娴嬫€у寮?*锛歚GetPhysicalDevices` 鍦?user_id 涓虹┖鏃朵笉鍐嶉潤榛樿繑鍥?`[]`锛岃€屾槸杩斿洖鍙尯鍒嗙殑绌哄洜鐘舵€侊紙鏈粦瀹氱敤鎴?/ 缁戝畾閿欎綅 / 鏃?MAC / 姝ｅ父锛夛紝鍓嶇灞曠ず璇婃柇缁撹鑰岄潪绌虹櫧鈥斺€旀鏄?DBG-07 闇€姹傛枃鏈€屽彲鍖哄垎绌哄洜鐘舵€併€?
files_changed: (N/A 鈥?鏁版嵁渚?+ 鍚庣画鍙娴嬫€у寮?
---

# 鏍瑰洜鎶ュ憡锛圧oot Cause Report锛?
## 1. 瀛楁褰掑睘

- **UI 瀛楁**锛?鐗╃悊閾捐矾璁惧锛坽N}鍙帮級" 鈥?鍓嶇鎶樺彔闈㈡澘鏍囬
- **鏁版嵁鏉ユ簮**锛歚/ops/workstation-device/{id}/physical` HTTP 绔偣
  - Handler: `internal/api/v1/operations/workstation_device_handler.go:152` `GetPhysicalDevices`
  - Service: `internal/services/operations/workstation_device_service.go:320` `GetPhysicalDevices`
- **杩斿洖妯″瀷**锛歚[]*models.WorkstationDevice`锛屼絾**涓嶆寔涔呭寲**锛圕TE 瀹炴椂鏌ヨ锛?- **鍏宠仈琛?*锛歚sys_workstation`銆乣ops_info_points`銆乣sys_device_port_status`銆乣sys_device_mac_address`銆乣sys_network_device`銆乣ops_asset`銆乣sys_ad_computer`

## 2. 鐗╃悊閾捐矾瑙ｆ瀽閾撅紙瀹炴椂 SQL CTE锛?
```
[sys_workstation] workstationID  (line 331)
  鈫?UserID 绌?鈫?line 339 鏃╅€€杩斿洖 []                                    鈽?H3 鏂偣
  鈫?[ops_info_points] WHERE workstation_id = ? AND deleted_at IS NULL
                  AND status = 0
  鈫?workstation_id 涓嶅尮閰?3f130 鈫?杩斿洖 0 琛?                           鈽?H1 鏂偣
  鈫?[sys_device_port_status] JOIN port.id::text = ip.port_id
  鈫?port_id 鏈瓨鎴?IP 閿欐寚 鈫?0 琛?                                      鈽?H1b 鏂偣
  鈫?[sys_network_device] EXISTS(id::text = ip.device_id)                   鈽?H4 鏂偣
  鈫?device_id orphan 鈫?杩囨护
  鈫?[sys_device_mac_address] JOIN mac.norm_iface = wp.norm_iface
                          AND mac.device_id::text = wp.effective_device_id::text
  鈫?MAC 鏈噰闆?鏍煎紡涓嶄竴鑷?鈫?0 琛?                                       鈽?H5 鏂偣锛堢敤鎴峰凡鎺掗櫎锛?  鈫?[ops_asset] JOIN a.mac1/mac2 normalized = mac.norm_mac
  鈫?璧勪骇鏃犲搴?MAC 鈫?鍛戒腑浣嗘棤璁惧璇︽儏
```

**鍏抽敭鍐崇瓥鐐?*锛?F003 淇 (commit `6575a1e7`) 宸插悎骞讹紝**device_id 婕傜Щ涓嶆瀯鎴愭湰娆?3f130 鐨勬牴鍥?*銆傚墿浣?4 涓嫭绔嬫柇鐐归渶鐢?SQL 閫愪竴楠岃瘉銆?
## 3. 鍋囪娓呭崟锛堟寜鍙兘鎬ф帓搴忥級

### 鍋囪 H1锛氬伐浣嶇粦瀹氶敊浣?/ workstation_id 瑙ｆ瀽澶辫触锛堝彲鑳芥€э細楂橈級
- **鎻忚堪**锛歚ops_info_points.workstation_id` 鎸囧悜鐨勫伐浣嶄笉鏄?3f130锛屾垨 3f130 鐨?UUID 涓庡墠绔紶鍏ヤ笉涓€鑷?- **浠ｇ爜浣嶇疆**锛歚workstation_device_service.go:401` CTE WHERE 瀛愬彞
- **璇佹嵁**锛歚OpsInfoPoint.WorkstationID gorm:"size:64"` 瀛楃涓插垪锛屽墠绔紶 UUID 瀛楃涓诧紝蹇呴』绮剧‘鍖归厤
- **鎴愮珛鏉′欢**锛氫俊鎭偣 3f130 鍦ㄦ暟鎹簱涓殑 workstation_id 瀛楁琚鏀广€佹垨鍓嶇 URL 涓?ID 閿欒瑙ｆ瀽
- **楠岃瘉 SQL**锛氳 搂4.1
- **楠岃瘉鏃ュ織**锛氭棤锛堢函瀛楃涓叉瘮瀵癸級

### 鍋囪 H2锛歰ps_info_points 杞垹闄?/ 鐘舵€侀潪 0锛堝彲鑳芥€э細涓級
- **鎻忚堪**锛歚ops_info_points.deleted_at IS NOT NULL` 鎴?`ops_info_points.status != 0` 瀵艰嚧 CTE 杩囨护鎺?- **浠ｇ爜浣嶇疆**锛歚workstation_device_service.go:402-403`
- **璇佹嵁**锛氱姸鎬佺害瀹?`0=姝ｅ父, 1=鏁呴殰, 2=鍋滅敤`锛涜蒋鍒犻櫎鐢?GORM `deleted_at` 绠＄悊
- **鎴愮珛鏉′欢**锛氫俊鎭偣琚鐞嗗憳鍋滅敤锛坰tatus=2锛夋垨璇蒋鍒犻櫎
- **楠岃瘉 SQL**锛氳 搂4.2

### 鍋囪 H3锛氬伐浣?3f130 鏈粦瀹?user_id锛堝彲鑳芥€э細涓級
- **鎻忚堪**锛歚workstation.UserID == nil || *workstation.UserID == ""` 瑙﹀彂 line 339 鏃╅€€鍒嗘敮锛岃繑鍥?`[]`
- **浠ｇ爜浣嶇疆**锛歚workstation_device_service.go:339-341`
- **璇佹嵁**锛?F003 淇 session (2026-06-30) 鏇炬妸姝ゅ垪涓?H1 楂樻鐜囧亣璁撅紱UI 涓?鍗犵敤"鐘舵€佷笉涓€瀹氱瓑鍚屼簬 user_id 缁戝畾
- **鎴愮珛鏉′欢**锛歴ys_workstation.user_id 瀛楁涓?NULL / 绌哄瓧绗︿覆
- **楠岃瘉 SQL**锛氳 搂4.3

### 鍋囪 H4锛欳X-WH-WH-04F-FL-RS8607E-01 鍦?sys_network_device 涓己澶憋紙鍙兘鎬э細浣?涓級
- **鎻忚堪**锛歭ine 404 `EXISTS(SELECT 1 FROM sys_network_device WHERE id::text = ip.device_id)` 杩囨护
- **浠ｇ爜浣嶇疆**锛歚workstation_device_service.go:404`
- **璇佹嵁**锛?575a1e7 寮曞叆璇?EXISTS 闃?orphan锛屼絾鑻?info_point.device_id 寮曠敤鐨?UUID 鍦?sys_network_device 宸茶蒋鍒犻櫎鎴栦粠鏈瓨鍦?鈫?鏁磋琚涪
- **鎴愮珛鏉′欢**锛氱綉缁滆澶囪杞垹闄?/ 閲嶅懡鍚嶅悗 info_point.device_id 鏈悓姝?/ UUID 鎷煎啓閿?- **楠岃瘉 SQL**锛氳 搂4.4

### 鍋囪 H5锛欸E2/6 涓?MAC 鏈噰闆嗭紙鍙兘鎬э細鐢ㄦ埛宸叉帓闄わ級
- **鎻忚堪**锛歚sys_device_mac_address` 鏃?GE2/6 鏉＄洰
- **鐢ㄦ埛宸茬‘璁ゆ帓闄?*锛氭姤鍛婄О"绔彛涓婃湁 MAC"

### 鍋囪 H6锛歩nfoPoint 宸ュ崟宸茬敓鎴愪絾鏈洖鍐欙紙鍙兘鎬э細浣?/ 璇瘖锛?- **涓嶉€傜敤**锛氭湰鍦烘櫙鏄墠绔疄鏃舵煡璇紝涓嶄緷璧栧伐鍗曞洖鍐?
## 4. 楠岃瘉 SQL 妯℃澘锛? 涓嫭绔嬪亣璁句簰鏂ワ級

### 4.1 楠岃瘉 H1锛歸orkstation_id 缁戝畾涓€鑷存€?```sql
-- Step 1: 鎵惧埌 3f130 宸ヤ綅鐨勭湡瀹?ID
SELECT id, code, name, location, user_id, deleted_at, status
  FROM sys_workstation
 WHERE code = '3f130' OR name = '3f130' OR location LIKE '%3f130%';

-- Step 2: 鎵惧嚭"澹扮О"缁戝畾鍒拌宸ヤ綅鐨勪俊鎭偣
SELECT ip.id, ip.name, ip.workstation_id, ip.port_id, ip.device_id, ip.status, ip.deleted_at
  FROM ops_info_points ip
 WHERE ip.workstation_id = (SELECT id FROM sys_workstation WHERE code = '3f130' OR name = '3f130' LIMIT 1);

-- Step 3: 鍙嶅悜鏍稿 鈥斺€?宸ヤ綅 3f130 鐨?ID 鏄惁涓?info_point.workstation_id 涓€鑷?-- 濡傛灉 Step 2 杩斿洖 0 琛?鈫?H1 鎴愮珛
-- 濡傛灉 Step 2 杩斿洖澶氳 鈫?妫€鏌?status / deleted_at锛圚2锛?```

### 4.2 楠岃瘉 H2锛氫俊鎭偣 status / soft delete
```sql
SELECT id, name, workstation_id, port_id, device_id, status, deleted_at,
       created_at, updated_at
  FROM ops_info_points
 WHERE workstation_id IN (SELECT id FROM sys_workstation WHERE code = '3f130' OR name = '3f130')
    OR device_id = (SELECT id FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01' LIMIT 1);

-- 鏈熸湜 status=0 AND deleted_at IS NULL锛?-- 鑻?status != 0 鎴?deleted_at 闈炵┖ 鈫?H2 鎴愮珛
```

### 4.3 楠岃瘉 H3锛歶ser_id 鏄惁缁戝畾
```sql
SELECT id, code, name, status, user_id, deleted_at
  FROM sys_workstation
 WHERE code = '3f130' OR name = '3f130';

-- 鑻?user_id IS NULL 鎴?'' 鈫?H3 鎴愮珛锛岃Е鍙?line 339 鏃╅€€
```

### 4.4 楠岃瘉 H4锛氱綉缁滆澶囨槸鍚﹀湪 sys_network_device
```sql
-- Step 1: 鎵惧埌 CX-WH-WH-04F-FL-RS8607E-01 鐨?ID
SELECT id, device_name, ip_address, status, deleted_at
  FROM sys_network_device
 WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01'
    OR ip_address = (SELECT ip_address FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01');

-- Step 2: 鎵?info_point 鏄惁寮曠敤杩欎釜 device_id
SELECT ip.id, ip.name, ip.workstation_id, ip.device_id, ip.deleted_at, ip.status
  FROM ops_info_points ip
 WHERE ip.device_id = (SELECT id FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01' LIMIT 1);

-- Step 3: 鍙嶅悜 鈥斺€?濡傛灉 info_point.device_id 涓嶅湪 sys_network_device 鈫?H4 鎴愮珛
SELECT ip.id, ip.device_id
  FROM ops_info_points ip
  LEFT JOIN sys_network_device nd ON nd.id::text = ip.device_id
 WHERE nd.id IS NULL
   AND ip.device_id IS NOT NULL
   AND ip.deleted_at IS NULL;
```

### 4.5 缁煎悎璇婃柇锛堜竴鏉?SQL 瑕嗙洊 4 涓亣璁撅級
```sql
WITH ws AS (
  SELECT id FROM sys_workstation WHERE code = '3f130' OR name = '3f130' LIMIT 1
),
nd AS (
  SELECT id FROM sys_network_device WHERE device_name = 'CX-WH-WH-04F-FL-RS8607E-01' LIMIT 1
)
SELECT
  (SELECT user_id FROM sys_workstation WHERE id = (SELECT id FROM ws)) AS ws_user_id,
  (SELECT COUNT(*) FROM ops_info_points
    WHERE workstation_id = (SELECT id FROM ws)
      AND deleted_at IS NULL
      AND status = 0) AS active_info_points,
  (SELECT COUNT(*) FROM ops_info_points ip
    JOIN sys_network_device nd ON nd.id::text = ip.device_id
    WHERE ip.workstation_id = (SELECT id FROM ws)
      AND ip.device_id = (SELECT id FROM nd)
      AND ip.deleted_at IS NULL
      AND ip.status = 0) AS ip_bound_to_correct_device,
  (SELECT COUNT(*) FROM sys_device_mac_address mac
    JOIN ops_info_points ip ON ip.device_id::text = mac.device_id::text
                            AND LOWER(REGEXP_REPLACE(ip.interface_name, '\s+', '', 'g'))
                                = LOWER(REGEXP_REPLACE(mac.interface_name, '\s+', '', 'g'))
    WHERE ip.workstation_id = (SELECT id FROM ws)
      AND ip.deleted_at IS NULL
      AND ip.status = 0) AS mac_match_count;
```

**鍒よ**锛?- `ws_user_id IS NULL` 鈫?H3
- `active_info_points = 0` 鈫?H2锛坰tatus 闈?0 / 宸插垹闄わ級
- `ip_bound_to_correct_device = 0` 鈫?H1锛坵orkstation_id 閿欎綅锛?- `ip_bound_to_correct_device = 0` 浣?`active_info_points > 0` 鈫?H4锛坉evice_id 婕傜Щ锛?- `mac_match_count > 0` 浣嗗墠绔粛鏄剧ず 0 鈫?绗?6 姝?ops_asset JOIN 澶辫触锛堣澶囨湭琚祫浜у寲锛?
## 5. 鎺ㄨ崘鎺掓煡璺緞

1. **鍏堥獙璇?H3**锛堜竴琛?SQL 鍗冲彲锛岃€楁椂 <100ms锛?2. **鍐嶉獙璇?H1+H2**锛坵orkstation_id 鏄惁缁戝畾 + 淇℃伅鐐圭姸鎬侊級
3. **鍐嶉獙璇?H4**锛坉evice_id 鏄惁鎸囧悜鐪熷疄缃戠粶璁惧锛?4. **鏈€鍚?*鐪?mac_match_count 鏄惁 > 0锛岃嫢 > 0 浠嶆湁 0 鍙?鈫?鏌?ops_asset JOIN 閾?
## 6. 淇鏂瑰悜锛堜粎璇婃柇锛?
| 鍋囪 | 淇鏂瑰悜 |
|------|----------|
| H1 鎴愮珛 | 淇 `ops_info_points.workstation_id` 鎸囧悜 3f130 鐪熷疄 ID |
| H2 鎴愮珛 | `UPDATE ops_info_points SET status = 0, deleted_at = NULL WHERE id = ?` |
| H3 鎴愮珛 | `UPDATE sys_workstation SET user_id = ? WHERE id = ?`锛堥渶涓庝汉浜嬬郴缁熸牳瀵癸級 |
| H4 鎴愮珛 | 淇 `ops_info_points.device_id` 鎸囧悜 `sys_network_device` 涓?CX-WH-WH-04F-FL-RS8607E-01 鐨?id |

## 7. 寰呯敤鎴锋彁渚涚殑鍏抽敭淇℃伅

- [ ] 鏁版嵁搴撳彧璇昏闂潈闄愶紙鎵ц 搂4 SQL锛?- [ ] 宸ヤ綅 3f130 鍦?`sys_workstation` 涓殑瀹為檯 `id`锛圲UID 瀛楃涓诧級
- [ ] 宸ヤ綅 3f130 鐨?`user_id` 瀛楁鍊硷紙鏄惁闈炵┖锛?- [ ] `ops_info_points` 涓?"澹扮О"缁戝畾鍒?3f130 鐨勮褰曪紙workstation_id / port_id / device_id / status / deleted_at锛?- [ ] `sys_network_device` 涓?CX-WH-WH-04F-FL-RS8607E-01 鐨勫疄闄?`id`
- [ ] 鍙€夛細sys_job_log 涓渶杩?24h 鍐?`reconciliation:detectLayer3` 浠诲姟鎵ц鐘舵€侊紙纭 R5 瀵硅处鏈紩鍏ュ station 鍋ュ悍搴︾殑鍓綔鐢級

## 8. 鍏抽敭璇佹嵁鏂囦欢璺緞

- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\operations\workstation_device_service.go:320-575` 鈥?GetPhysicalDevices 涓诲疄鐜?- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\operations\workstation_device_service.go:339` 鈥?user_id 鏃╅€€鍒嗘敮
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\services\operations\workstation_device_service.go:391-404` 鈥?CTE workstation_ports
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\api\v1\operations\workstation_device_handler.go:152-166` 鈥?Handler
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\operations\infopoint.go` 鈥?InfoPoint 妯″瀷
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\workstation.go` 鈥?Workstation 妯″瀷
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\device_port_status.go` 鈥?PortStatus 妯″瀷
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\device_mac_address.go` 鈥?MAC 妯″瀷
- `D:\code\ClaudeCode\ruoyi-go-backend\internal\models\network_device.go` 鈥?NetworkDevice 妯″瀷
- `D:\code\ClaudeCode\ruoyi-go-backend\ruoyi-react-frontend\src\components\operations\WorkstationDeviceTable\index.tsx:64` 鈥?鍓嶇璋冪敤鐐?- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\debug\workstation-physical-link-zero.md` 鈥?5F003 鍚屾簮 session锛堝凡鏍规不锛?- `D:\code\ClaudeCode\ruoyi-go-backend\.planning\debug\info-point-port-drift-recheck-20260701.md` 鈥?1247/1483 鍘嗗彶婕傜Щ缁熻

## Resolution 补记（2026-10-28，Phase 73 修复落地）

代码侧 commit `8b85fe6b`：新增 `POST /ops/workstation-device/:id/physical-diagnosis` 端点，返回 `emptyReason`/`emptyMessage`/`warnings`/`diagnosis` 四类空因 + 十项链路诊断计数（向后兼容，旧端点不变）。**剩余数据侧动作**：给 3f130 绑定 user_id（业务核对后一条 UPDATE 或工位管理页绑定）——代码已能诊断出该空因。5 个新测试含 3f130 实测四项读数复刻用例。**本档关闭（数据侧绑定挂 v1.24 用户动作）。**