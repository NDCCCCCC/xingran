package commands

import "testing"

// TestFieldDocCommonFields 验证必含常见字段全部命中
func TestFieldDocCommonFields(t *testing.T) {
	required := []string{
		"username", "password", "nickname", "phone", "email", "status", "remark",
		"deptId", "roleIds", "postIds", "employeeNo", "gender", "current", "pageSize",
		"beginTime", "endTime", "orderByColumn", "isAsc", "id", "ids", "name", "title",
		"type", "sort", "parentId", "path", "component", "perms", "icon", "visible",
		"redirect", "createBy", "createTime", "updateBy", "updateTime", "params",
		"dataScope", "roleKey", "roleName", "postCode", "postName", "dictType",
		"dictLabel", "dictValue", "configKey", "configValue", "noticeTitle",
		"noticeType", "noticeContent", "avatar", "oldPassword", "newPassword",
	}
	for _, f := range required {
		desc, ok := FieldDoc(f)
		if !ok {
			t.Errorf("必含字段 %q 未收录", f)
			continue
		}
		if desc == "" {
			t.Errorf("字段 %q 说明为空", f)
		}
	}
}

// TestFieldDocCoverage 验证词典总量达到任务要求（扩容后 250-400 个）
func TestFieldDocCoverage(t *testing.T) {
	n := FieldDocCount()
	if n < 250 {
		t.Errorf("词典字段数 %d < 250，覆盖不足", n)
	}
	if n > 400 {
		t.Errorf("词典字段数 %d > 400，超出预期规模", n)
	}
}

// TestFieldDocModuleFields 验证 M6 阶段二新模块字段命中
func TestFieldDocModuleFields(t *testing.T) {
	cases := map[string]string{
		// operations
		"longitude":    "经度（通过地址自动解析，decimal(11,8)）",
		"orgId":        "所属机构 ID（UUID，必须为有效 UUID 而非部门名）",
		"floorNo":      "楼层编号（如 3F）",
		"isp":          "运营商（电信/移动/联通等）",
		"monthlyFee":   "月租费用（元）",
		// network
		"deviceName":   "设备名称",
		"ipAddress":    "IP 地址（设备管理地址，唯一）",
		"macAddress":   "MAC 地址（入库自动归一化为大写+冒号）",
		"vlanId":       "VLAN ID",
		"lastSeenAt":   "最后在线时间",
		// monitor
		"cpuUsage":     "CPU 使用率（%）",
		"memoryUsage":  "内存使用率（%）",
		"diskUsage":    "磁盘使用率（%）",
		"hitRate":      "缓存命中率",
		// workorder / scheduler / duty / knowledge / vdi / rpa
		"assigneeId":   "处理人（被分配人）ID（UUID）",
		"priority":     "优先级",
		"cronExpression": "Cron 表达式（5 段）",
		"misfirePolicy": "错失触发策略",
		"swapReason":   "调换原因",
		"sourceWorkOrderId": "来源工单 ID（工单转知识库）",
		"cpu_per":      "CPU 使用率（%）",
		"vm_id":        "虚拟机 ID（VDI 平台唯一标识）",
		"workerId":     "RPA Worker 标识",
	}
	for f, want := range cases {
		got, ok := FieldDoc(f)
		if !ok {
			t.Errorf("模块字段 %q 未收录", f)
			continue
		}
		if got != want {
			t.Errorf("FieldDoc(%q) = %q, want %q", f, got, want)
		}
	}
}

// TestEnumHintModuleFields 新模块枚举按仓库常量核实
func TestEnumHintModuleFields(t *testing.T) {
	cases := map[string]string{
		"deviceType":   "server=服务器 storage=存储 ups=UPS pdu=PDU firewall=防火墙 kvm=KVM cabinet=机柜 other=其他",
		"macType":      "dynamic=动态学习 static=静态配置 secure=安全MAC",
		"vendor":       "huawei=华为 h3c=华三 ruijie=锐捷 maipu=迈普 cisco=思科",
		"priority":     "0=低 1=中 2=高 3=紧急",
		"misfirePolicy": "0=默认 1=立即执行 2=执行一次 3=放弃执行",
		"deskType":     "0=一字型 1=L型",
		"ip_type":      "STATIC=静态 DHCP=动态",
		"level":        "1=城市级汇总 2=具体楼宇",
	}
	for f, want := range cases {
		if got := EnumHint(f); got != want {
			t.Errorf("EnumHint(%q) = %q, want %q", f, got, want)
		}
	}
}

// TestEnumHint 枚举语义与仓库 AGENTS.md 约定一致
func TestEnumHint(t *testing.T) {
	cases := []struct {
		field string
		want  string
	}{
		{"status", "0=正常/启用 1=停用/禁用"},
		{"gender", "0=男 1=女 2=保密"},
		{"visible", "1=显示 0=隐藏"}, // 菜单 visible 例外：1=可见 0=隐藏
		{"noticeType", "1=公告 2=警告"},
		{"dataScope", "1=全部数据 2=自定义数据 3=本部门数据 4=本部门及子部门数据 5=仅本人数据"},
		{"menuType", "M=目录 C=菜单 F=按钮"},
		{"configType", "Y=是 N=否"},
		{"isAsc", "asc=升序 desc=降序"},
		{"authSource", "local=本地 AD=AD 域 hybrid=混合"},
	}
	for _, c := range cases {
		if got := EnumHint(c.field); got != c.want {
			t.Errorf("EnumHint(%q) = %q, want %q", c.field, got, c.want)
		}
	}
	// 无枚举字段返回空串
	if got := EnumHint("username"); got != "" {
		t.Errorf("EnumHint(username) 应为空, got %q", got)
	}
	// 未知字段返回空串
	if got := EnumHint("totallyUnknownField"); got != "" {
		t.Errorf("未知字段 EnumHint 应为空, got %q", got)
	}
}

// TestFieldDocUnknown 未知字段返回 ok=false
func TestFieldDocUnknown(t *testing.T) {
	for _, f := range []string{"", "   ", "nonexistentField", "foo_bar_baz_99"} {
		if _, ok := FieldDoc(f); ok {
			t.Errorf("未知字段 %q 不应命中", f)
		}
	}
}

// TestFieldDocSnakeCase snake_case 变体可归一化命中
func TestFieldDocSnakeCase(t *testing.T) {
	cases := map[string]string{
		"user_id":     "用户 ID（UUID）",
		"role_key":    "角色权限字符（唯一，如 admin）",
		"dict_label":  "字典标签（显示文本）",
		"old_password": "原密码（修改密码时校验）",
		"page_size":   "每页条数",
	}
	for in, want := range cases {
		got, ok := FieldDoc(in)
		if !ok || got != want {
			t.Errorf("FieldDoc(%q) = (%q, %v), want (%q, true)", in, got, ok, want)
		}
	}
}

// TestFieldDocWithEnum 拼接渲染
func TestFieldDocWithEnum(t *testing.T) {
	if got := FieldDocWithEnum("status"); got != "状态（0=正常/启用 1=停用/禁用）" {
		t.Errorf("FieldDocWithEnum(status) = %q", got)
	}
	if got := FieldDocWithEnum("username"); got != "用户名（登录账号）" {
		t.Errorf("FieldDocWithEnum(username) = %q", got)
	}
	if got := FieldDocWithEnum("nope"); got != "" {
		t.Errorf("FieldDocWithEnum(未知字段) 应为空, got %q", got)
	}
}
