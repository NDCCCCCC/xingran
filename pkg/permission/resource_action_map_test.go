package permission

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLookupResourceAction_Hit 遍历 D-02 全部 (resource, action) 组合,断言
// 返回 PermissionCode 非空 + bool=true + 值与 config.go 常量一致。
//
// 这是 Phase 61 / D-02 的核心契约:11 个 resource × 每个 resource 的所有
// action 必须全部命中,且值必须 = 对应 PermissionCode 常量(严禁字符串拼接)。
func TestLookupResourceAction_Hit(t *testing.T) {
	cases := []struct {
		resource string
		action   string
		want     PermissionCode
	}{
		// system:user 8 项
		{"system:user", "list", UserList},
		{"system:user", "view", UserView},
		{"system:user", "add", UserAdd},
		{"system:user", "edit", UserEdit},
		{"system:user", "remove", UserRemove},
		{"system:user", "export", UserExport},
		{"system:user", "import", UserImport},
		{"system:user", "resetPwd", UserResetPwd},
		// system:role 6 项
		{"system:role", "list", RoleList},
		{"system:role", "view", RoleView},
		{"system:role", "add", RoleAdd},
		{"system:role", "edit", RoleEdit},
		{"system:role", "remove", RoleRemove},
		{"system:role", "export", RoleExport},
		// system:menu 5 项
		{"system:menu", "list", MenuList},
		{"system:menu", "view", MenuView},
		{"system:menu", "add", MenuAdd},
		{"system:menu", "edit", MenuEdit},
		{"system:menu", "remove", MenuRemove},
		// system:dept 5 项
		{"system:dept", "list", DeptList},
		{"system:dept", "view", DeptView},
		{"system:dept", "add", DeptAdd},
		{"system:dept", "edit", DeptEdit},
		{"system:dept", "remove", DeptRemove},
		// system:post 5 项
		{"system:post", "list", PostList},
		{"system:post", "view", PostView},
		{"system:post", "add", PostAdd},
		{"system:post", "edit", PostEdit},
		{"system:post", "remove", PostRemove},
		// system:workstation 5 项
		{"system:workstation", "list", WorkstationList},
		{"system:workstation", "view", WorkstationView},
		{"system:workstation", "add", WorkstationAdd},
		{"system:workstation", "edit", WorkstationEdit},
		{"system:workstation", "remove", WorkstationRemove},
		// system:dict 5 项
		{"system:dict", "list", DictTypeList},
		{"system:dict", "view", DictTypeView},
		{"system:dict", "add", DictTypeAdd},
		{"system:dict", "edit", DictTypeEdit},
		{"system:dict", "remove", DictTypeRemove},
		// system:config 5 项
		{"system:config", "list", ConfigList},
		{"system:config", "view", ConfigView},
		{"system:config", "add", ConfigAdd},
		{"system:config", "edit", ConfigEdit},
		{"system:config", "remove", ConfigRemove},
		// system:captchaBackground 5 项
		{"system:captchaBackground", "list", CaptchaBackgroundList},
		{"system:captchaBackground", "view", CaptchaBackgroundView},
		{"system:captchaBackground", "add", CaptchaBackgroundAdd},
		{"system:captchaBackground", "edit", CaptchaBackgroundEdit},
		{"system:captchaBackground", "remove", CaptchaBackgroundRemove},
		// system:notice 5 项
		{"system:notice", "list", NoticeList},
		{"system:notice", "view", NoticeView},
		{"system:notice", "add", NoticeAdd},
		{"system:notice", "edit", NoticeEdit},
		{"system:notice", "remove", NoticeRemove},
		// system:apikey 5 项
		{"system:apikey", "list", APIKeyList},
		{"system:apikey", "view", APIKeyView},
		{"system:apikey", "add", APIKeyAdd},
		{"system:apikey", "edit", APIKeyEdit},
		{"system:apikey", "remove", APIKeyRemove},
	}

	require.Len(t, cases, 59, "D-02 应覆盖 11 个 resource 的全部 action 组合 = 8+6+5*9 = 59 项")

	for _, c := range cases {
		t.Run(c.resource+":"+c.action, func(t *testing.T) {
			code, ok := LookupResourceAction(c.resource, c.action)
			assert.True(t, ok, "命中 (resource=%s, action=%s) 必须返回 ok=true", c.resource, c.action)
			assert.NotEmpty(t, code, "命中必须返回非空 PermissionCode")
			assert.Equal(t, c.want, code, "返回值必须与 config.go 既有常量一致,严禁字符串拼接")
		})
	}
}

// TestLookupResourceAction_UnmappedResource monitor:* 不在本 phase map 中
// (D-02 范围限定),Lookup 必须返回 ("", false) — fail-closed 语义。
func TestLookupResourceAction_UnmappedResource(t *testing.T) {
	code, ok := LookupResourceAction("monitor:operlog", "list")
	assert.False(t, ok, "monitor:operlog 未注册, 必须返回 ok=false (D-03 fail-closed)")
	assert.Empty(t, code, "未命中资源必须返回空 PermissionCode,不返回默认值")
}

// TestLookupResourceAction_UnmappedAction system:user 已注册但 flyToMoon
// 不在 action 词汇集内,必须返回 ("", false)。
func TestLookupResourceAction_UnmappedAction(t *testing.T) {
	code, ok := LookupResourceAction("system:user", "flyToMoon")
	assert.False(t, ok, "system:user.flyToMoon 未注册, 必须返回 ok=false")
	assert.Empty(t, code)
}

// TestMapKeys_AllSystemModules 断言返回的 resources 包含全部 D-02 资源名,
// 共 11 项(含 system:apikey,这是 Phase 61 新增的 resource)。
func TestMapKeys_AllSystemModules(t *testing.T) {
	resources, _ := MapKeys()
	sort.Strings(resources)
	expected := []string{
		"system:apikey",
		"system:captchaBackground",
		"system:config",
		"system:dept",
		"system:dict",
		"system:menu",
		"system:notice",
		"system:post",
		"system:role",
		"system:user",
		"system:workstation",
	}
	assert.Equal(t, expected, resources, "MapKeys 必须返回 D-02 全部 11 个 resource")
}

// TestMapKeys_ActionVocabularyAligned 断言 actions 包含 D-04 词汇子集。
// 注意:MapKeys 返回的 actions 是所有 resource 词汇的并集,所以 resetPwd
// (system:user 独有) 与 export / import 等都应存在。
func TestMapKeys_ActionVocabularyAligned(t *testing.T) {
	_, actions := MapKeys()
	actionSet := make(map[string]bool, len(actions))
	for _, a := range actions {
		actionSet[a] = true
	}

	expectedActions := []string{
		"list", "view", "add", "edit", "remove",
		"export", "import", "resetPwd",
	}
	for _, want := range expectedActions {
		assert.True(t, actionSet[want], "action 词汇 %q 必须出现在 MapKeys 返回值中", want)
	}
}

// TestMapKeys_NoMonitorNetworkToolOperations 防御性断言:本 phase 不纳入的
// 模块前缀绝不能出现在 resources 列表(D-02 范围限定)。
func TestMapKeys_NoMonitorNetworkToolOperations(t *testing.T) {
	resources, _ := MapKeys()
	forbiddenPrefixes := []string{"monitor:", "network:", "tool:", "operations:"}
	for _, r := range resources {
		for _, prefix := range forbiddenPrefixes {
			assert.False(t,
				len(r) >= len(prefix) && r[:len(prefix)] == prefix,
				"resource %q 属于 %q 模块, 不在本 phase D-02 范围内", r, prefix,
			)
		}
	}
}
