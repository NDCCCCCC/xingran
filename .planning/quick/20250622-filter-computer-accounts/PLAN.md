# 修复 AD 域用户同步的两个问题

## 问题1：计算机账号过滤

**症状：** AD 域中以 `$` 结尾的账号（如 `CXHUBTY-E4BD9CE$`）是计算机账号，不应该出现在用户列表和同步中。

**示例日志：**
```
INFO[2026-06-22 21:48:38] [SyncADUser] 开始同步 AD 用户: username=CXHUBTY-E4BD9CE$, userDN=CN=CXHUBTY-E4BD9CE,OU=恩施市三岔镇营销服务部,OU=恩施市支公司,OU=恩施中心支公司,OU=Computer,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
ERRO[2026-06-22 21:48:38] [SyncADUser] 同步失败: 同步AD用户失败: context canceled
```

**根因：** `internal/services/addomain/user.go` 的 `GetList` 和 `GetUserIds` 方法只过滤了 `$DUPLICATE-` 前缀，没有过滤以 `$` 结尾的计算机账号。

**修复方案：** 在两个方法的查询中添加 `username NOT LIKE '%$'` 过滤条件。

## 问题2：context canceled 错误

**症状：** 批量同步时出现大量 `context canceled` 错误。

**需要检查：**
1. 批量同步的超时设置
2. 是否有并发限制导致超时
3. AD连接池的熔断机制

## 实施步骤

1. 修改 `GetList` 方法，添加计算机账号过滤
2. 修改 `GetUserIds` 方法，添加计算机账号过滤
3. 检查批量同步的超时配置
4. 提交修复
