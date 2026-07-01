# MAC 接口名 `GE1/0/4SECURITY` / `GigabitEthernet 0/2` 回归诊断

**状态**: 根因已定位,修复中
**日期**: 2026-07-02
**触发**: /gsd-debug

## 症状

用户报告 MAC 地址采集后接口名出现两类不规范数据(均 2026-07-02 00:01 采集):

1. `GE1/0/4SECURITY` —— 短名已折叠,但尾部粘连 `SECURITY`(大写)
2. `GigabitEthernet 0/2` + 小写 MAC(`0074.9c2e.ac71` 等)—— 全称带空格 + 小写

## 根因(两类独立)

### 症状① `GE1/0/4SECURITY` —— normalize.InterfaceName 真实代码 BUG

`pkg/normalize/iface.go:64` 的 `standardShortPrefix` 守卫(`^(GE|XGE|TWE|HGE|FOE|FE|ET)[0-9]`)
命中 `GE1/...` 后直接 `return strings.ToUpper(name)`,**不剥离尾部字母垃圾**。

华为 VRP `display mac-address` 输出 security 类型 MAC 时,接口名与 `security` 标记粘连
(无空格),`strings.Fields` 切不开 → `GE1/0/4security` → 守卫放行 → `GE1/0/4SECURITY`。

**实测验证**(临时测试,已删):
```
IN  "GE1/0/4SECURITY"   OUT "GE1/0/4SECURITY"   ❌ 尾部未剥离
IN  "GE1/0/4security"   OUT "GE1/0/4SECURITY"   ❌
IN  "GigabitEthernet 0/2" OUT "GE0/2"            ✓ 折叠正确(反证症状②)
IN  "0074.9c2e.ac71"    OUT "00:74:9C:2E:AC:71"  ✓ MAC 归一化正确(反证症状②)
```

这是 [[normalize-iface-reverse-expand-trap]] 对称化时引入的副作用:守卫为防"反向展开"
做得太激进,放行了尾部脏字符。**即使最新二进制仍会复现。必修。**

### 症状② `GigabitEthernet 0/2` + 小写 MAC —— 生产二进制旧(部署问题)

代码侧 MAC 写入路径已 100% 归一化,反证症状② 在当前代码下不可能:
- service 层: `mac_collection_service.go:281` 入库前调 `NormalizeInterfaceName`
- model hook: `device_mac_address.go:47` BeforeCreate 调 `normalize.InterfaceName/MACAddress`
- mac_history / port_status hook 同样归一化(device_mac_history.go:51, device_port_status.go:70)
- 转发链: `portcollection.NormalizeInterfaceName → normalize.InterfaceName`(utils.go:21)完整
- GORM 配置无 SkipHooks(database.go:97)

带**空格**全称 + 小写 MAC = normalize 第一行 `ReplaceAll(" ","")` 都没执行 = hook 不在生产
二进制里。与 [[normalize-iface-reverse-expand-trap]] 警告一致:"必须重启服务部署新二进制"。

用户称"已部署 07-01 后最新代码",但代码逻辑反证 → 部署/构建/重启环节有问题。

## 修复方案(用户已授权:代码根治 + 历史清理)

1. **normalize.InterfaceName 尾部剥离**(治症状①): 守卫命中后用正则
   `^(GE|XGE|...)[0-9][0-9/.:]*` 提取合法主体,丢弃尾部字母垃圾
2. **parseMACLine 华为分支 security 识别加固**(防御层): Type 跳过列表加 `security`
3. **migration_194 历史脏数据清理**: 清 mac_address / mac_history / port_status 的
   `*SECURITY` 后缀 + 全称/小写 MAC 残留
4. **iface_test 加尾部剥离契约**: 锁定 `GE1/0/4SECURITY → GE1/0/4`
5. **运维(用户)**: 生产重新编译部署 17459ec9+ 二进制,治症状②

## 关键证据文件

- `pkg/normalize/iface.go:57-66` (守卫放行尾部)
- `internal/services/mac_collection_service.go:444-461` (parseMACLine 华为分支,Type 跳过列表无 security)
- `internal/models/device_mac_address.go:42-49` (BeforeCreate hook)
- `internal/core/db/database.go:97-102` (GORM 配置无 SkipHooks)
