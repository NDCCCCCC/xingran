// Package commands 提供 CLI 命令树的共享数据源。
//
// fielddocs.go 是「字段中文说明词典」：为 help 输出提供参数中文说明与枚举语义。
// 数据来源：internal/models/*.go（GORM 模型注释）、各 handler 请求结构体，
// 以及仓库 AGENTS.md 的状态值约定（0=正常/启用 1=停用/禁用；菜单 visible 例外：1=可见 0=隐藏）。
//
// 用法（供 help 渲染与 `schema` 命令补充说明）：
//
//	desc, ok := fielddocs.FieldDoc("status")   // "状态"
//	hint := fielddocs.EnumHint("status")       // "0=正常 1=停用"
package commands

import "strings"

// fieldEntry 单个字段词条
type fieldEntry struct {
	desc string // 中文说明
	enum string // 枚举语义（无枚举为空）
}

// fieldDocs 字段词典（键为后端 JSON 字段名，全小写匹配）
var fieldDocs = map[string]fieldEntry{
	// ---- 认证/登录 ----
	"username":          {desc: "用户名（登录账号）"},
	"password":          {desc: "密码（SM2 加密后传输）"},
	"oldPassword":       {desc: "原密码（修改密码时校验）"},
	"newPassword":       {desc: "新密码（SM2 加密后传输）"},
	"resetPassword":     {desc: "重置后的密码（SM2 加密后传输）"},
	"captchaId":         {desc: "验证码 ID（GET /system/auth/captcha 返回）"},
	"captcha":           {desc: "验证码答案（normal 模式）或滑动验证 token（slider 模式）"},
	"captchaCode":       {desc: "图形验证码"},
	"sliderX":           {desc: "滑动验证 X 坐标（滑块拼图位置，像素）"},
	"code":              {desc: "验证码/授权码"},
	"uuid":              {desc: "验证码会话标识"},
	"encryptedPassword": {desc: "密码是否已加密传输（true=已 SM2 加密）"},
	"grantType":         {desc: "授权类型（password=密码登录 refresh_token=刷新）"},
	"refreshToken":      {desc: "刷新令牌（JWT，长期有效）"},
	"accessToken":       {desc: "访问令牌（JWT，短期有效）"},

	// ---- 用户 ----
	"nickname":     {desc: "用户昵称"},
	"employeeNo":   {desc: "工号（企业微信代码）"},
	"email":        {desc: "电子邮箱"},
	"phone":        {desc: "联系电话"},
	"mobile":       {desc: "手机号（来自 AD mobile 同步）"},
	"avatar":       {desc: "头像地址（URL）"},
	"gender":       {desc: "性别", enum: "0=男 1=女 2=保密"},
	"deptId":       {desc: "部门 ID（UUID）"},
	"deptName":     {desc: "部门名称"},
	"deptFullName": {desc: "完整部门路径（所有层级）"},
	"roleIds":      {desc: "角色 ID 数组（用户/角色分配）"},
	"postIds":      {desc: "岗位 ID 数组（用户岗位分配）"},
	"roles":        {desc: "角色标识数组"},
	"posts":        {desc: "岗位对象数组"},
	"loginIp":      {desc: "最后登录 IP"},
	"loginTime":    {desc: "最后登录时间"},
	"pwdExpireDays": {desc: "密码有效期（天，默认 90）"},
	"initFlag":      {desc: "是否初始账号（true=未修改过初始密码）"},
	"authSource":    {desc: "认证来源", enum: "local=本地 AD=AD 域 hybrid=混合"},
	"adUsername":    {desc: "AD 域账号名"},
	"adConfigId":    {desc: "来源 AD 配置 ID（UUID，多域隔离）"},
	"title":         {desc: "职位名称"},
	"userId":        {desc: "用户 ID（UUID）"},
	"userIdList":    {desc: "用户 ID 数组（批量操作）"},
	"salt":          {desc: "密码盐值（服务端内部使用）"},

	// ---- 角色 ----
	"roleId":            {desc: "角色 ID（UUID）"},
	"roleName":          {desc: "角色名称"},
	"roleKey":           {desc: "角色权限字符（唯一，如 admin）"},
	"roleSort":          {desc: "角色显示顺序（越小越靠前）"},
	"dataScope":         {desc: "数据范围", enum: "1=全部数据 2=自定义数据 3=本部门数据 4=本部门及子部门数据 5=仅本人数据"},
	"menuCheckStrictly": {desc: "菜单树选择项是否关联显示（父子联动）"},
	"deptCheckStrictly": {desc: "部门树选择项是否关联显示（父子联动）"},
	"menuIds":           {desc: "菜单 ID 数组（角色授权菜单）"},
	"deptIds":           {desc: "部门 ID 数组（角色授权数据权限）"},

	// ---- 岗位 ----
	"postId":   {desc: "岗位 ID（UUID）"},
	"postCode": {desc: "岗位编码"},
	"postName": {desc: "岗位名称"},
	"postSort": {desc: "岗位显示顺序（越小越靠前）"},

	// ---- 部门 ----
	"parentId": {desc: "父级 ID（UUID，顶级为空或 0）"},
	"orderNum": {desc: "显示顺序（越小越靠前）"},
	"leader":   {desc: "负责人"},
	"ancestors": {desc: "祖级部门路径（逗号分隔 ID）"},

	// ---- 菜单 ----
	"menuId":    {desc: "菜单 ID（UUID）"},
	"menuName":  {desc: "菜单名称"},
	"menuType":  {desc: "菜单类型", enum: "M=目录 C=菜单 F=按钮"},
	"path":      {desc: "路由地址（前端路由 path）"},
	"component": {desc: "组件路径（前端页面组件）"},
	"perms":     {desc: "权限标识（如 system:user:list）"},
	"icon":      {desc: "菜单图标"},
	"visible":   {desc: "菜单显示状态", enum: "1=显示 0=隐藏"},
	"redirect":  {desc: "重定向地址（路由 redirect）"},
	"query":     {desc: "路由参数（如 {\"k\":\"v\"}）"},
	"meta":      {desc: "路由元数据对象（title/icon/keepAlive 等）"},
	"keepAlive": {desc: "页面是否缓存（组件保活）"},
	"hidden":    {desc: "是否隐藏菜单（路由仍可访问）"},
	"affix":     {desc: "是否固定标签页（不可关闭）"},
	"noCache":   {desc: "是否禁用页面缓存"},
	"link":      {desc: "外链跳转地址"},
	"i18nKey":   {desc: "国际化 key（预留）"},

	// ---- 字典 ----
	"dictId":    {desc: "字典类型 ID（UUID）"},
	"dictCode":  {desc: "字典数据 ID（UUID）"},
	"dictName":  {desc: "字典名称"},
	"dictType":  {desc: "字典类型（唯一键）"},
	"dictLabel": {desc: "字典标签（显示文本）"},
	"dictValue": {desc: "字典键值"},
	"dictSort":  {desc: "字典显示顺序（越小越靠前）"},
	"cssClass":  {desc: "样式属性（前端样式类名）"},
	"listClass": {desc: "表格回显样式（如 default/primary/success）"},
	"isDefault": {desc: "是否默认项（Y/N 或 true/false）"},

	// ---- 参数配置 ----
	"configId":    {desc: "参数配置 ID（UUID）"},
	"configName":  {desc: "参数名称"},
	"configKey":   {desc: "参数键名（如 sys.auth.ad.enabled）"},
	"configValue": {desc: "参数键值"},
	"configType":  {desc: "系统内置配置标记", enum: "Y=是 N=否"},
	"isSystem":    {desc: "是否系统内置（0=否 1=是，内置不可删除）"},

	// ---- 通知公告 ----
	"noticeId":      {desc: "公告 ID（UUID）"},
	"noticeTitle":   {desc: "公告标题"},
	"noticeType":    {desc: "公告类型", enum: "1=公告 2=警告"},
	"noticeContent": {desc: "公告内容（支持富文本）"},

	// ---- 通用查询/分页 ----
	"id":       {desc: "资源 ID（UUID）"},
	"ids":      {desc: "资源 ID 数组（批量操作）"},
	"name":     {desc: "名称（模糊查询/资源名）"},
	"keyword":  {desc: "关键字（模糊匹配）"},
	"searchKey": {desc: "搜索关键字"},
	"type":     {desc: "类型标识"},
	"sort":     {desc: "排序值（越小越靠前）"},
	"current":  {desc: "页码（从 1 开始）"},
	"pageSize": {desc: "每页条数"},
	"total":    {desc: "总记录数（响应字段）"},
	"list":     {desc: "当前页数据列表（响应字段）"},
	"orderByColumn": {desc: "排序字段名"},
	"isAsc":         {desc: "排序方向", enum: "asc=升序 desc=降序"},
	"beginTime":     {desc: "查询开始时间（YYYY-MM-DD 或完整时间）"},
	"endTime":       {desc: "查询结束时间（YYYY-MM-DD 或完整时间）"},
	"params":        {desc: "附加查询参数对象（如时间范围）"},
	"status":        {desc: "状态", enum: "0=正常/启用 1=停用/禁用"},
	"remark":        {desc: "备注"},
	"data":          {desc: "业务数据（响应包装字段）"},
	"message":       {desc: "响应消息"},
	"requestId":     {desc: "请求追踪 ID（响应字段）"},

	// ---- 审计字段 ----
	"createBy":  {desc: "创建者"},
	"createTime": {desc: "创建时间"},
	"updateBy":  {desc: "更新者"},
	"updateTime": {desc: "更新时间"},
	"deletedAt": {desc: "删除时间（软删除标记，非空=已删除）"},
	"createdAt": {desc: "创建时间（记录级）"},
	"updatedAt": {desc: "更新时间（记录级）"},

	// ---- 操作日志 / 登录日志 ----
	"operId":        {desc: "操作日志 ID（UUID）"},
	"operName":      {desc: "操作人员（用户名）"},
	"operIp":        {desc: "操作 IP 地址"},
	"operUrl":       {desc: "请求 URL"},
	"operParam":     {desc: "请求参数（敏感值已脱敏）"},
	"jsonResult":    {desc: "响应结果 JSON"},
	"businessType":  {desc: "业务操作类型", enum: "0=其他 1=新增 2=修改 3=删除 4=授权 5=导出 6=导入 7=强退 8=生成代码 9=清空数据"},
	"method":        {desc: "请求方法（HTTP method + handler 名）"},
	"module":        {desc: "业务模块中文名（如 用户管理）"},
	"costTime":      {desc: "耗时（毫秒）"},
	"errorMsg":      {desc: "错误消息"},
	"infoId":        {desc: "登录日志 ID（UUID）"},
	"ipaddr":        {desc: "登录 IP 地址"},
	"browser":       {desc: "浏览器类型"},
	"os":            {desc: "操作系统"},
	"loginLocation": {desc: "登录地点（IP 归属地）"},
	"msg":           {desc: "日志消息（登录结果描述）"},
	"loginStatus":   {desc: "登录结果", enum: "0=成功 1=失败"},
	"logId":         {desc: "日志 ID（UUID）"},
	"logType":       {desc: "日志类型（oper=操作日志 login=登录日志）"},

	// ---- 通知增强 / 消息推送 ----
	"targetType":  {desc: "通知目标类型", enum: "all=全员 user=指定用户 dept=指定部门 role=指定角色"},
	"targetIds":   {desc: "通知目标 ID 数组（按 targetType 解释）"},
	"readStatus":  {desc: "已读状态", enum: "0=未读 1=已读"},
	"ignoreStatus": {desc: "忽略状态", enum: "0=未忽略 1=已忽略"},
	"pushChannel": {desc: "推送渠道（如 inapp/email/webhook）"},
	"attachment":  {desc: "附件信息（URL 或对象）"},
	"publisher":   {desc: "发布人"},

	// ---- 文件/导入导出 ----
	"fileName":   {desc: "文件名"},
	"filePath":   {desc: "文件路径（服务器相对路径）"},
	"file":       {desc: "上传文件（multipart 表单字段）"},
	"templateType": {desc: "模板类型（excel 导入模板，如 building/floor/workstation）"},
	"updateSupport": {desc: "导入时已存在记录是否更新", enum: "true=更新 false=跳过"},
	"exportAll":  {desc: "是否导出全部（忽略分页）"},

	// ---- 其他 system 通用 ----
	"captchaEnabled": {desc: "验证码是否启用（配置项布尔值）"},
	"tenantId":       {desc: "租户 ID（预留多租户）"},
	"isFrame":        {desc: "是否外链菜单", enum: "0=是 1=否"},
	"cacheName":      {desc: "缓存名称（缓存监控操作）"},
	"cacheKey":       {desc: "缓存键（不要带 xingran: 前缀）"},
	"onlineId":       {desc: "在线用户会话 ID"},
	"forceLogout":    {desc: "是否强制下线"},
	"passwordStrength": {desc: "密码强度等级要求"},
	"deptCategory":   {desc: "部门类别标识"},

	// ==================== operations 运维管理 ====================
	// 楼宇（models/operations/building.go）
	"address":       {desc: "详细地址"},
	"longitude":     {desc: "经度（通过地址自动解析，decimal(11,8)）"},
	"latitude":      {desc: "纬度（通过地址自动解析，decimal(11,8)）"},
	"level":         {desc: "层级", enum: "1=城市级汇总 2=具体楼宇"},
	"orgId":         {desc: "所属机构 ID（UUID，必须为有效 UUID 而非部门名）"},
	"orgName":       {desc: "所属机构名称"},
	"totalFloors":   {desc: "楼层数（根据创建的楼层自动计算）"},
	"workstationCount": {desc: "工位数（子查询动态计算，只读）"},
	"floorCount":    {desc: "楼层数实时统计（按页内聚合注入，只读）"},
	"buildingId":    {desc: "楼宇 ID（UUID）"},
	"buildingName":  {desc: "楼宇名称"},
	// 楼层/机房/机房设备（models/operations.go）
	"floorId":       {desc: "楼层 ID（UUID）"},
	"floorNo":       {desc: "楼层编号（如 3F）"},
	"floorName":     {desc: "楼层名称"},
	"area":          {desc: "面积（平方米，decimal(10,2)）"},
	"planImageId":   {desc: "平面图图片 ID"},
	"planImageUrl":  {desc: "平面图图片 URL"},
	"roomId":        {desc: "机房 ID（UUID）"},
	"roomName":      {desc: "机房名称"},
	"deviceType":    {desc: "设备类型", enum: "server=服务器 storage=存储 ups=UPS pdu=PDU firewall=防火墙 kvm=KVM cabinet=机柜 other=其他"},
	"serialNumber":  {desc: "设备序列号"},
	"positionU":     {desc: "机柜 U 位（设备在机柜中的位置）"},
	"positionDesc":  {desc: "位置描述"},
	"assetNumber":   {desc: "资产编号"},
	"purchaseDate":  {desc: "采购日期"},
	"warrantyDate":  {desc: "保修到期日期"},
	"responsibleId": {desc: "责任人 ID（UUID）"},
	"responsibleName": {desc: "责任人姓名"},
	"powerConsumption": {desc: "功耗（瓦）"},
	"model":         {desc: "设备型号"},
	"vendor":        {desc: "厂商", enum: "huawei=华为 h3c=华三 ruijie=锐捷 maipu=迈普 cisco=思科"},
	// 工位（models/workstation.go）
	"userName":      {desc: "所属人员名称"},
	"location":      {desc: "位置描述"},
	"positionX":     {desc: "平面图 X 坐标"},
	"positionY":     {desc: "平面图 Y 坐标"},
	"rotation":      {desc: "旋转角度（度，0-360）"},
	"width":         {desc: "工位宽度（毫米，默认 1600）"},
	"depth":         {desc: "工位深度（毫米，默认 700）"},
	"deskType":      {desc: "桌型", enum: "0=一字型 1=L型"},
	"primaryDeviceSerial": {desc: "主设备序列号（来自子查询，只读）"},
	// 专线（models/operations/dedicated_line.go）
	"lineType":      {desc: "专线类型（数据字典）"},
	"bandwidth":     {desc: "带宽（如 100M、1G）"},
	"isp":           {desc: "运营商（电信/移动/联通等）"},
	"sourceDeviceId":   {desc: "源设备 ID"},
	"sourceDeviceName": {desc: "源设备名称"},
	"sourcePort":    {desc: "源端口"},
	"sourceRoomId":  {desc: "源机房 ID"},
	"sourceRoomName": {desc: "源机房名称"},
	"destDeviceId":  {desc: "目的设备 ID"},
	"destDeviceName": {desc: "目的设备名称"},
	"destPort":      {desc: "目的端口"},
	"destRoomId":    {desc: "目的机房 ID"},
	"destRoomName":  {desc: "目的机房名称"},
	"sourceIpAddress": {desc: "源 IP 地址"},
	"destIpAddress": {desc: "目的 IP 地址"},
	"sourceSubnetMask": {desc: "源子网掩码"},
	"destSubnetMask": {desc: "目的子网掩码"},
	"vlan":          {desc: "VLAN 标识"},
	"carrierContactId":   {desc: "运营商联系人 ID"},
	"carrierContactName": {desc: "运营商联系人姓名"},
	"carrierContactPhone": {desc: "运营商联系人电话"},
	"monthlyFee":    {desc: "月租费用（元）"},

	// ==================== network 网络管理 ====================
	// 网络设备（models/network_device.go）
	"deviceName":    {desc: "设备名称"},
	"ipAddress":     {desc: "IP 地址（设备管理地址，唯一）"},
	"snmpPort":      {desc: "SNMP 端口（默认 161）"},
	"credentialId":  {desc: "设备凭据 ID（UUID）"},
	"credentialName": {desc: "设备凭据名称"},
	"lastSeenAt":    {desc: "最后在线时间"},
	"lastConfigAt":  {desc: "最后配置备份时间"},
	"description":   {desc: "描述信息"},
	"softwareVersion": {desc: "软件版本（自动采集）"},
	"uptime":        {desc: "设备运行时长（自动采集）"},
	"deviceId":      {desc: "设备 ID（UUID）"},
	// MAC 地址（models/device_mac_address.go）
	"macAddress":    {desc: "MAC 地址（入库自动归一化为大写+冒号）"},
	"interfaceName": {desc: "接口名（自动归一化为大写短名）"},
	"vlanId":        {desc: "VLAN ID"},
	"macType":       {desc: "MAC 地址类型", enum: "dynamic=动态学习 static=静态配置 secure=安全MAC"},
	"collectedAt":   {desc: "采集时间"},
	"lastSeen":      {desc: "最后看到时间"},

	// ==================== monitor 监控 ====================
	"hostName":       {desc: "主机名"},
	"arch":           {desc: "系统架构（如 amd64）"},
	"cpuCount":       {desc: "CPU 核心数"},
	"totalMemory":    {desc: "总内存（字节）"},
	"availableMemory": {desc: "可用内存（字节）"},
	"diskTotal":      {desc: "磁盘总容量（字节）"},
	"diskAvailable":  {desc: "磁盘可用容量（字节）"},
	"lastActiveAt":   {desc: "最后活跃时间"},
	"serverId":       {desc: "服务器 ID"},
	"cpuUsage":       {desc: "CPU 使用率（%）"},
	"memoryUsage":    {desc: "内存使用率（%）"},
	"diskUsage":      {desc: "磁盘使用率（%）"},
	"networkRx":      {desc: "网络接收字节数"},
	"networkTx":      {desc: "网络发送字节数"},
	"processCount":   {desc: "进程数量"},
	"loadAverage":    {desc: "系统负载"},
	"timestamp":      {desc: "采集时间（监控指标）"},
	"value":          {desc: "缓存值（缓存监控）"},
	"ttl":            {desc: "过期时间（秒）"},
	"cacheType":      {desc: "缓存类型"},
	"hitCount":       {desc: "缓存命中次数"},
	"missCount":      {desc: "缓存未命中次数"},
	"hitRate":        {desc: "缓存命中率"},
	"usedMemory":     {desc: "已用内存"},
	"keyCount":       {desc: "键值对数量"},
	"expiredCount":   {desc: "过期键数量"},
	"collectTime":    {desc: "统计采集时间"},

	// ==================== workorder 工单 ====================
	"workOrderId":   {desc: "工单 ID（UUID）"},
	"categoryId":    {desc: "分类 ID（UUID）"},
	"priority":      {desc: "优先级", enum: "0=低 1=中 2=高 3=紧急"},
	"solution":      {desc: "解决方案"},
	"submitterId":   {desc: "提交人 ID（UUID）"},
	"assigneeId":    {desc: "处理人（被分配人）ID（UUID）"},
	"expectedResolveAt": {desc: "期望解决时间"},
	"resolvedAt":    {desc: "解决时间"},
	"closedAt":      {desc: "关闭时间"},
	"attachmentIds": {desc: "附件 ID 列表（逗号分隔）"},
	"dutyPoolId":    {desc: "关联值班池 ID（UUID）"},
	"dutyType":      {desc: "值班类型"},
	"assignStrategy": {desc: "分配策略（轮询/指定等）"},
	"assignType":    {desc: "分配方式", enum: "auto=自动 manual=手动指定"},
	"assignTargetId": {desc: "分配目标 ID（assignType=manual 时使用）"},
	"nextRunAt":     {desc: "下次执行时间（周期工单）"},
	"templateId":    {desc: "模板 ID（周期工单模板）"},
	"field":         {desc: "变更字段（工单历史）"},
	"oldValue":      {desc: "变更前值"},
	"newValue":      {desc: "变更后值"},
	"score":         {desc: "评分（工单评价）"},
	"comment":       {desc: "评论内容"},
	"result":        {desc: "执行结果"},
	"jobMessage":    {desc: "任务执行消息（调度日志）"},

	// ==================== scheduler 定时任务 ====================
	"jobName":        {desc: "任务名称"},
	"jobGroup":       {desc: "任务分组"},
	"invokeTarget":   {desc: "调用目标（函数或处理器标识）"},
	"cronExpression": {desc: "Cron 表达式（5 段）"},
	"misfirePolicy":  {desc: "错失触发策略", enum: "0=默认 1=立即执行 2=执行一次 3=放弃执行"},
	"concurrent":     {desc: "是否允许并发执行"},
	"nextRunTime":    {desc: "下次运行时间"},
	"prevRunTime":    {desc: "上次运行时间"},
	"jobId":          {desc: "定时任务 ID"},

	// ==================== duty 值班 ====================
	"poolId":            {desc: "值班池 ID（UUID）"},
	"dutyDate":          {desc: "值班日期"},
	"startDate":         {desc: "值班开始日期"},
	"endDate":           {desc: "值班结束日期"},
	"scheduleId":        {desc: "值班计划 ID（UUID）"},
	"swapFromDate":      {desc: "调换起始日期"},
	"swapReason":        {desc: "调换原因"},
	"originalUserId":    {desc: "原值班人 ID（UUID）"},
	"newUserId":         {desc: "新值班人 ID（UUID）"},
	"beforeReminderMinutes": {desc: "提前提醒分钟数（0=当天提醒）"},

	// ==================== knowledge 知识库 ====================
	"articleId":     {desc: "文章 ID（UUID）"},
	"summary":       {desc: "摘要"},
	"content":       {desc: "正文内容（Markdown）"},
	"sourceWorkOrderId": {desc: "来源工单 ID（工单转知识库）"},
	"tagId":         {desc: "标签 ID（UUID）"},
	"viewCount":     {desc: "浏览次数"},

	// ==================== vdi 虚拟机（json 为 snake_case） ====================
	"vm_id":          {desc: "虚拟机 ID（VDI 平台唯一标识）"},
	"vmId":           {desc: "虚拟机 ID"},
	"resource_id":    {desc: "VDI 平台资源 ID"},
	"power_state":    {desc: "电源状态（如 powered_on/powered_off）"},
	"os_type":        {desc: "操作系统类型"},
	"cpu_number":     {desc: "CPU 颗数"},
	"cpu_core":       {desc: "每颗 CPU 核数"},
	"cpu_per":        {desc: "CPU 使用率（%）"},
	"memory_per":     {desc: "内存使用率（%）"},
	"disk_per":       {desc: "磁盘使用率（%）"},
	"bound_user_id":  {desc: "绑定用户 ID"},
	"bound_user_name": {desc: "绑定用户名称"},
	"policy_group_id": {desc: "策略组 ID"},
	"ip_type":        {desc: "IP 类型", enum: "STATIC=静态 DHCP=动态"},
	"subnet_mask":    {desc: "子网掩码"},
	"default_gateway": {desc: "默认网关"},
	"name_server":    {desc: "DNS 服务器"},
	"assign_ip":      {desc: "分配的 IP 地址"},
	"vdi_server_id":  {desc: "VDI 服务器 ID"},
	"last_sync_at":   {desc: "最后同步时间"},
	"ip_address":     {desc: "IP 地址（VDI 虚拟机）"},
	"mac_address":    {desc: "MAC 地址（VDI 虚拟机）"},

	// ==================== rpa 自动化 ====================
	"targetUrl":     {desc: "目标页面 URL（RPA 任务）"},
	"script":        {desc: "RPA 脚本（动作数组 JSON）"},
	"selector":      {desc: "CSS 选择器（RPA 动作定位）"},
	"timeout":       {desc: "超时时间（RPA 毫秒/任务秒，按接口）"},
	"retry":         {desc: "重试次数"},
	"maxRetry":      {desc: "最大重试次数"},
	"workerId":      {desc: "RPA Worker 标识"},
	"lastExecutionId": {desc: "最近一次执行 ID（UUID）"},
	"version":       {desc: "版本号（乐观锁）"},
	"tags":          {desc: "标签（逗号分隔）"},
}

// aliases 字段名归一化映射：常见的 camelCase/snake_case 变体统一到词典键
var aliases = map[string]string{
	"user_id": "userId", "role_id": "roleId", "post_id": "postId",
	"dept_ids": "deptIds", "menu_id": "menuId", "config_id": "configId",
	"dict_id": "dictId", "dict_code": "dictCode", "notice_id": "noticeId",
	"order_by": "orderByColumn", "page_size": "pageSize",
	"begin_time": "beginTime", "end_time": "endTime",
	"old_password": "oldPassword", "new_password": "newPassword",
	"role_key": "roleKey", "role_name": "roleName", "post_code": "postCode",
	"post_name": "postName", "dict_type": "dictType", "dict_label": "dictLabel",
	"dict_value": "dictValue", "config_key": "configKey", "config_value": "configValue",
	"notice_title": "noticeTitle", "notice_type": "noticeType",
	"notice_content": "noticeContent", "data_scope": "dataScope",
	"parent_id": "parentId", "order_num": "orderNum", "create_by": "createBy",
	"create_time": "createTime", "update_by": "updateBy", "update_time": "updateTime",
	"employee_no": "employeeNo", "request_id": "requestId", "user_ids": "userIdList",
	"auth_source": "authSource", "keep_alive": "keepAlive", "is_default": "isDefault",
}

// normalize 归一化字段名：去空格、统一 snake_case 到 camelCase 词典键
func normalize(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	// 直接命中或小写命中
	if _, ok := fieldDocs[name]; ok {
		return name
	}
	if lower := strings.ToLower(name); lower != name {
		if _, ok := fieldDocs[lower]; ok {
			return lower
		}
	}
	// snake_case → camelCase（先查别名表）
	if alt, ok := aliases[name]; ok {
		return alt
	}
	// 通用 snake → camel 转换
	if strings.Contains(name, "_") {
		parts := strings.Split(name, "_")
		var b strings.Builder
		b.WriteString(strings.ToLower(parts[0]))
		for _, p := range parts[1:] {
			if p != "" {
				b.WriteString(strings.ToUpper(p[:1]) + strings.ToLower(p[1:]))
			}
		}
		return b.String()
	}
	return name
}

// lookup 归一化后查词典
func lookup(name string) (fieldEntry, bool) {
	key := normalize(name)
	e, ok := fieldDocs[key]
	return e, ok
}

// FieldDoc 返回字段的中文说明。
// name 支持后端 JSON 字段名（camelCase）与 snake_case 变体；未收录字段返回 ok=false。
func FieldDoc(name string) (desc string, ok bool) {
	e, found := lookup(name)
	return e.desc, found
}

// EnumHint 返回字段的枚举语义提示（如 "0=正常 1=停用"）。
// 无枚举语义的字段返回空字符串。
func EnumHint(name string) string {
	e, _ := lookup(name)
	return e.enum
}

// FieldDocWithEnum 返回拼接了枚举提示的说明，供 help 一行渲染：
// "状态（0=正常/启用 1=停用/禁用）"；未收录字段返回空串。
func FieldDocWithEnum(name string) string {
	e, ok := lookup(name)
	if !ok {
		return ""
	}
	if e.enum == "" {
		return e.desc
	}
	return e.desc + "（" + e.enum + "）"
}

// FieldDocCount 返回词典收录字段总数（测试与 help 统计用）
func FieldDocCount() int {
	return len(fieldDocs)
}
