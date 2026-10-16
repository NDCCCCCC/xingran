package permission

// resource_action_map.go 由 Phase 61 AUTH-04 引入。
//
// 本文件维护一个「编译期常量」的 (resource, action) → PermissionCode 静态映射,
// 供 internal/middleware.RequireAPIKeyResourcePermission 在请求级别做资源级
// 权限校验时查询使用。新增 resource / action 时必须显式补 map entry,否则
// LookupResourceAction 走 fail-closed 路径返回 (code, false),由调用方
// 返回 403 拒绝请求 — 这是 D-03 fail-closed 语义的强制约束。
//
// 覆盖范围(Phase 61 / D-02):仅 system:* 模块(user / role / menu / dept / post /
// workstation / dict / config / captchaBackground / notice / apikey 共 11 个资源),
// 共 59 个 (resource, action) 组合。
//
// **未纳入**:monitor:* / network:* / tool:* / operations:* 等模块。这些模块的
// 资源权限矩阵留待后续 phase 单独评估(避免 Phase 61 爆炸半径外溢),本 phase
// 仅做「resource 参数真实生效」的接入工作。
//
// 值显式包 PermissionCode(...) 是为了让"PermissionCode"字面量在源码中出现
// 59 次以上 — grep 验证条目数;同时让每个 entry 的类型意图明确。

// resourceActionMap 资源 × 操作 → PermissionCode 静态映射。
//
// 嵌套 map 结构:外层 key = resource(如 "system:user"),内层 key = action(如
// "list"),值 = PermissionCode 常量(对齐 pkg/permission/config.go 既有常量,严禁
// 字符串拼接 — 防止拼写错误漏检)。
//
// action 词汇(D-04 对齐 config.go 既有 PermissionCode 末段):
//   list / view / add / edit / remove / export / import / resetPwd
//
// 注:`remove` 与 `delete` 同义 — `remove` 是 system:* 模块约定,`delete` 是
// network:* 模块约定,本 phase map 仅含 `remove`(D-04 末段)。
var resourceActionMap = map[string]map[string]PermissionCode{
	// 用户管理:8 个 action(包含 export/import/resetPwd 三个独有 action)
	"system:user": {
		"list":     PermissionCode(UserList),
		"view":     PermissionCode(UserView),
		"add":      PermissionCode(UserAdd),
		"edit":     PermissionCode(UserEdit),
		"remove":   PermissionCode(UserRemove),
		"export":   PermissionCode(UserExport),
		"import":   PermissionCode(UserImport),
		"resetPwd": PermissionCode(UserResetPwd),
	},

	// 角色管理:6 个 action(包含 export)
	"system:role": {
		"list":   PermissionCode(RoleList),
		"view":   PermissionCode(RoleView),
		"add":    PermissionCode(RoleAdd),
		"edit":   PermissionCode(RoleEdit),
		"remove": PermissionCode(RoleRemove),
		"export": PermissionCode(RoleExport),
	},

	// 菜单管理:5 个 action
	"system:menu": {
		"list":   PermissionCode(MenuList),
		"view":   PermissionCode(MenuView),
		"add":    PermissionCode(MenuAdd),
		"edit":   PermissionCode(MenuEdit),
		"remove": PermissionCode(MenuRemove),
	},

	// 部门管理:5 个 action
	"system:dept": {
		"list":   PermissionCode(DeptList),
		"view":   PermissionCode(DeptView),
		"add":    PermissionCode(DeptAdd),
		"edit":   PermissionCode(DeptEdit),
		"remove": PermissionCode(DeptRemove),
	},

	// 岗位管理:5 个 action
	"system:post": {
		"list":   PermissionCode(PostList),
		"view":   PermissionCode(PostView),
		"add":    PermissionCode(PostAdd),
		"edit":   PermissionCode(PostEdit),
		"remove": PermissionCode(PostRemove),
	},

	// 工位管理:5 个 action
	"system:workstation": {
		"list":   PermissionCode(WorkstationList),
		"view":   PermissionCode(WorkstationView),
		"add":    PermissionCode(WorkstationAdd),
		"edit":   PermissionCode(WorkstationEdit),
		"remove": PermissionCode(WorkstationRemove),
	},

	// 字典管理:5 个 action
	"system:dict": {
		"list":   PermissionCode(DictTypeList),
		"view":   PermissionCode(DictTypeView),
		"add":    PermissionCode(DictTypeAdd),
		"edit":   PermissionCode(DictTypeEdit),
		"remove": PermissionCode(DictTypeRemove),
	},

	// 参数配置:5 个 action
	"system:config": {
		"list":   PermissionCode(ConfigList),
		"view":   PermissionCode(ConfigView),
		"add":    PermissionCode(ConfigAdd),
		"edit":   PermissionCode(ConfigEdit),
		"remove": PermissionCode(ConfigRemove),
	},

	// 验证码背景图:5 个 action
	"system:captchaBackground": {
		"list":   PermissionCode(CaptchaBackgroundList),
		"view":   PermissionCode(CaptchaBackgroundView),
		"add":    PermissionCode(CaptchaBackgroundAdd),
		"edit":   PermissionCode(CaptchaBackgroundEdit),
		"remove": PermissionCode(CaptchaBackgroundRemove),
	},

	// 通知公告:5 个 action
	"system:notice": {
		"list":   PermissionCode(NoticeList),
		"view":   PermissionCode(NoticeView),
		"add":    PermissionCode(NoticeAdd),
		"edit":   PermissionCode(NoticeEdit),
		"remove": PermissionCode(NoticeRemove),
	},

	// API Key 管理:5 个 action
	"system:apikey": {
		"list":   PermissionCode(APIKeyList),
		"view":   PermissionCode(APIKeyView),
		"add":    PermissionCode(APIKeyAdd),
		"edit":   PermissionCode(APIKeyEdit),
		"remove": PermissionCode(APIKeyRemove),
	},
}

// LookupResourceAction 查 map 命中 → 返回 (PermissionCode, true),否则返回 ("", false)。
//
// 调用方必须把 `ok=false` 视为「资源权限未定义」并 fail-closed 返回 403
// (见 internal/middleware.RequireAPIKeyResourcePermission)。函数本身不返回
// 默认值、不 panic — 严格保持纯查询语义,失败处理权交给调用方。
func LookupResourceAction(resource, action string) (PermissionCode, bool) {
	actions, ok := resourceActionMap[resource]
	if !ok {
		return "", false
	}
	code, ok := actions[action]
	if !ok {
		return "", false
	}
	return code, true
}

// MapKeys 返回所有已注册 resource / action key 列表(供测试断言完整性)。
//
// 返回值顺序不固定(Go map 迭代随机),测试断言须用 ElementsMatch/Contains。
func MapKeys() (resources []string, actions []string) {
	actionSet := make(map[string]struct{})
	for resource, actionMap := range resourceActionMap {
		resources = append(resources, resource)
		for action := range actionMap {
			actionSet[action] = struct{}{}
		}
	}
	for action := range actionSet {
		actions = append(actions, action)
	}
	return resources, actions
}
