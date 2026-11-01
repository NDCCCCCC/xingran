package addomain

// DEBT-01（v1.51）：manager_rule_service.go 行为测试补齐。
//
// 覆盖目标（断言行为，非凑覆盖率）：
//  1. ManagerResolver 纯逻辑：OrgOf 祖先回溯、matchSubject/matchScope 全分支、
//     resolveManagerID 四种来源、resolveTargetOrg 三种取值、pickPostHolder 过滤与
//     多任职人 WARN、resolveDeptLeader 深度回溯、子树判断
//  2. 规则引擎端到端：first-match 优先级、自指顺延、机构约束
//  3. ResolveManagerDN 双路径：无规则回退 leader 递归祖先 / 规则引擎 + ad_dn 查找
//  4. 规则/机构 CRUD 与列表翻译、参数校验、HasEnabledRules 兼容语义
//  5. PreviewManagers 预览组装
//
// 数据库层使用 SQLite 内存库 + AutoMigrate（与既有 *_test.go 建表 fixture 模式一致，
// 不 mock gorm 层）。注意 LoadResolver 有进程内 TTL 缓存，每个用例 setup 时必须
// invalidateResolverCache() 防止用例间串扰。

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// setupManagerRuleTestDB 创建 SQLite 内存库并迁移规则引擎所需表。
func setupManagerRuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 SQLite 内存库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&models.ManagerRule{},
		&models.ManagerOrg{},
		&models.Department{},
		&models.User{},
		&models.UserPost{},
		&models.Post{},
	); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	invalidateResolverCache()
	t.Cleanup(invalidateResolverCache)
	return db
}

// boolPtr 测试辅助（strPtr 已存在于 manager_sync_test.go，零副本复用）。
func boolPtr(b bool) *bool { return &b }

// mgrFixture 端到端场景数据（对应 manager_rule_service.go 头部概念模型示例）：
//
//	部门树: D_root(ancestors="0") → D_a(ancestors="0,D_root") → D_a1(ancestors="0,D_root,D_a")
//	机构:   O0=总公司(dept=D_root, 无上级)  O1=武汉中支(dept=D_a, parent=O0)
//	岗位:   P_gm=总经理（一把手岗）  P_vp=分管副总
//	用户:   gm(P_gm 任职, dept D_a, O1) / leader_a(D_a 的 leader) / emp(普通员工, dept D_a1)
type mgrFixture struct {
	db        *gorm.DB
	org0, org1 models.ManagerOrg
	gm, leader, emp models.User
	postGM, postVP string
}

func setupMgrFixture(t *testing.T) *mgrFixture {
	t.Helper()
	db := setupManagerRuleTestDB(t)
	ctx := context.Background()

	depts := []models.Department{
		{BaseModel: models.BaseModel{ID: "d-root"}, DeptName: "总公司", DeptCode: "ROOT", Ancestors: "0"},
		{BaseModel: models.BaseModel{ID: "d-a"}, DeptName: "武汉中支", DeptCode: "WH", Ancestors: "0,d-root", Leader: strPtr("u-leader")},
		{BaseModel: models.BaseModel{ID: "d-a1"}, DeptName: "综合部", DeptCode: "WH-ZH", Ancestors: "0,d-root,d-a"},
	}
	if err := db.Create(&depts).Error; err != nil {
		t.Fatalf("建部门失败: %v", err)
	}

	orgs := []models.ManagerOrg{
		{BaseModel: models.BaseModel{ID: "org-0"}, OrgName: "总公司机构", DeptID: "d-root", OrgType: 0, Enabled: true},
		{BaseModel: models.BaseModel{ID: "org-1"}, OrgName: "武汉中支机构", DeptID: "d-a", ParentOrgID: strPtr("org-0"), OrgType: 1, Enabled: true},
	}
	if err := db.Create(&orgs).Error; err != nil {
		t.Fatalf("建机构失败: %v", err)
	}

	posts := []models.Post{
		{BaseModel: models.BaseModel{ID: "p-gm"}, PostName: "总经理", PostCode: "GM"},
		{BaseModel: models.BaseModel{ID: "p-vp"}, PostName: "分管副总", PostCode: "VP"},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatalf("建岗位失败: %v", err)
	}

	users := []models.User{
		{BaseModel: models.BaseModel{ID: "u-gm"}, Username: "gm", DeptID: strPtr("d-a"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=gm,DC=test")},
		{BaseModel: models.BaseModel{ID: "u-leader"}, Username: "leader", Nickname: strPtr("王 leader"), DeptID: strPtr("d-a"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=leader,DC=test")},
		{BaseModel: models.BaseModel{ID: "u-emp"}, Username: "emp", DeptID: strPtr("d-a1"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=emp,DC=test")},
		{BaseModel: models.BaseModel{ID: "u-boss"}, Username: "boss", DeptID: strPtr("d-root"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=boss,DC=test")},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("建用户失败: %v", err)
	}

	// sys_user_post 表由 AutoMigrate 建（含 created_at），直接手写 INSERT 规避列差异
	if err := db.Exec(`INSERT INTO sys_user_post (user_id, post_id) VALUES ('u-gm','p-gm'),('u-leader','p-vp'),('u-boss','p-gm')`).Error; err != nil {
		t.Fatalf("建任职关系失败: %v", err)
	}

	_ = ctx
	return &mgrFixture{
		db: db, org0: orgs[0], org1: orgs[1],
		gm: users[0], leader: users[1], emp: users[2],
		postGM: "p-gm", postVP: "p-vp",
	}
}

// ============================ 纯逻辑：Resolver 单元 ============================

// buildTestResolver 手工构建 ManagerResolver（不经过 DB），精确控制每个索引。
func buildTestResolver() *ManagerResolver {
	r := &ManagerResolver{
		rules: []models.ManagerRule{},
		orgs: []models.ManagerOrg{
			{BaseModel: models.BaseModel{ID: "org-0"}, OrgName: "总公司机构", DeptID: "d-root"},
			{BaseModel: models.BaseModel{ID: "org-1"}, OrgName: "中支机构", DeptID: "d-a", ParentOrgID: strPtr("org-0")},
		},
		orgByID:   map[string]models.ManagerOrg{},
		orgByDept: map[string]models.ManagerOrg{},
		depts: map[string]models.Department{
			"d-root": {BaseModel: models.BaseModel{ID: "d-root"}, DeptName: "总公司", Ancestors: "0", Leader: strPtr("u-root-leader")},
			"d-a":    {BaseModel: models.BaseModel{ID: "d-a"}, DeptName: "中支", Ancestors: "0,d-root", Leader: strPtr("u-leader")},
			"d-a1":   {BaseModel: models.BaseModel{ID: "d-a1"}, DeptName: "综合部", Ancestors: "0,d-root,d-a"},
		},
		users: map[string]models.User{
			"u-a": {BaseModel: models.BaseModel{ID: "u-a"}, Username: "alice", DeptID: strPtr("d-a1"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=a")},
			"u-b": {BaseModel: models.BaseModel{ID: "u-b"}, Username: "bob", DeptID: strPtr("d-a"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=b")},
			"u-c": {BaseModel: models.BaseModel{ID: "u-c"}, Username: "carol", DeptID: strPtr("d-root"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=c")},
			// 禁用 / 无 dn / 机构外任职人：pickPostHolder 过滤路径用
			"u-off":  {BaseModel: models.BaseModel{ID: "u-off"}, Username: "off", DeptID: strPtr("d-a"), Status: models.UserStatusDisabled, AdDn: strPtr("CN=off")},
			"u-nodn": {BaseModel: models.BaseModel{ID: "u-nodn"}, Username: "nodn", DeptID: strPtr("d-a"), Status: models.UserStatusEnabled},
			"u-ext":  {BaseModel: models.BaseModel{ID: "u-ext"}, Username: "ext", DeptID: strPtr("d-root"), Status: models.UserStatusEnabled, AdDn: strPtr("CN=ext")},
		},
		postNames:         map[string]string{"p-vp": "分管副总"},
		deptNames:         map[string]string{"d-a": "中支"},
		orgNames:          map[string]string{"org-1": "中支机构"},
		deptLeaderUserIDs: map[string]bool{"u-leader": true},
		postHolders:       map[string][]string{},
		userPosts:         map[string][]string{},
	}
	for _, o := range r.orgs {
		r.orgByID[o.ID] = o
		r.orgByDept[o.DeptID] = o
	}
	// p-vp 任职人：u-b（有效）、u-off（禁用）、u-nodn（无 dn）、u-ext（机构外）
	for _, uid := range []string{"u-ext", "u-nodn", "u-off", "u-b"} {
		r.addPostHolder("p-vp", uid)
	}
	r.sortPostHolders()
	return r
}

func TestManagerResolver_OrgOf(t *testing.T) {
	r := buildTestResolver()

	// 用户无部门 → false
	if _, ok := r.OrgOf(&models.User{BaseModel: models.BaseModel{ID: "x"}}); ok {
		t.Fatal("无部门用户不应命中机构")
	}
	// 部门直接是机构节点 → 命中 org-1
	org, ok := r.OrgOf(&models.User{BaseModel: models.BaseModel{ID: "x"}, DeptID: strPtr("d-a")})
	if !ok || org.ID != "org-1" {
		t.Fatalf("直属机构节点应命中 org-1, got %v ok=%v", org, ok)
	}
	// 部门在链上回溯 → d-a1 无机构 → 沿祖先命中 org-1（最近优先）
	org, ok = r.OrgOf(&models.User{BaseModel: models.BaseModel{ID: "x"}, DeptID: strPtr("d-a1")})
	if !ok || org.ID != "org-1" {
		t.Fatalf("祖先回溯应命中最近的 org-1, got %v ok=%v", org, ok)
	}
	// 未知部门 → false
	if _, ok := r.OrgOf(&models.User{BaseModel: models.BaseModel{ID: "x"}, DeptID: strPtr("nope")}); ok {
		t.Fatal("未知部门不应命中机构")
	}
	// OrgOf(nil) → false
	if _, ok := r.OrgOf(nil); ok {
		t.Fatal("nil 用户不应命中机构")
	}
}

func TestManagerResolver_MatchSubjectAndScope(t *testing.T) {
	r := buildTestResolver()
	emp := &models.User{BaseModel: models.BaseModel{ID: "u-a"}, DeptID: strPtr("d-a1")}
	org, _ := r.OrgOf(emp)

	// employee：u-a 不是 leader 也不是 org_head → true
	base := models.ManagerRule{SubjectType: models.ManagerSubjectEmployee}
	if !r.matchSubject(base, emp, org) {
		t.Fatal("普通员工应匹配 employee 主体")
	}
	// employee：u-leader 是部门负责人 → false
	leader := &models.User{BaseModel: models.BaseModel{ID: "u-leader"}, DeptID: strPtr("d-a")}
	if r.matchSubject(base, leader, org) {
		t.Fatal("部门负责人不应匹配 employee 主体")
	}
	// dept_leader 主体
	if !r.matchSubject(models.ManagerRule{SubjectType: models.ManagerSubjectDeptLeader}, leader, org) {
		t.Fatal("负责人应匹配 dept_leader 主体")
	}
	// post_holder 主体：无 SubjectPostID → false；有且任职 → true
	if r.matchSubject(models.ManagerRule{SubjectType: models.ManagerSubjectPostHolder}, leader, org) {
		t.Fatal("post_holder 未指定岗位应返回 false")
	}
	holderB := r.users["u-b"]
	// org_head 主体：SubjectPostID 优先于 ManagerPostID
	if !r.matchSubject(models.ManagerRule{SubjectType: models.ManagerSubjectPostHolder, SubjectPostID: strPtr("p-vp")}, &holderB, org) {
		t.Fatal("任职人应匹配 post_holder 主体")
	}
	if !r.matchSubject(models.ManagerRule{SubjectType: models.ManagerSubjectOrgHead, SubjectPostID: strPtr("p-vp")}, &holderB, org) {
		t.Fatal("任职人应匹配 org_head 主体（SubjectPostID）")
	}
	if !r.matchSubject(models.ManagerRule{SubjectType: models.ManagerSubjectOrgHead, ManagerPostID: strPtr("p-vp")}, &holderB, org) {
		t.Fatal("任职人应匹配 org_head 主体（回退 ManagerPostID）")
	}
	// 未知 subject_type → false
	if r.matchSubject(models.ManagerRule{SubjectType: "bogus"}, emp, org) {
		t.Fatal("非法主体类型应返回 false")
	}
	// SubjectOrgID 约束：主体机构不匹配 → false
	constrained := models.ManagerRule{SubjectType: models.ManagerSubjectEmployee, SubjectOrgID: strPtr("org-0")}
	if r.matchSubject(constrained, emp, org) {
		t.Fatal("主体机构不匹配应返回 false")
	}

	// matchScope：All 恒真；DeptTree 含自身/子树/范围未配置；OrgTree 子树
	if !r.matchScope(models.ManagerRule{ScopeType: models.ManagerScopeAll}, emp, org) {
		t.Fatal("ScopeAll 应恒真")
	}
	if r.matchScope(models.ManagerRule{ScopeType: models.ManagerScopeDeptTree}, emp, org) {
		t.Fatal("DeptTree 未配置 ScopeID 应返回 false")
	}
	if !r.matchScope(models.ManagerRule{ScopeType: models.ManagerScopeDeptTree, ScopeID: strPtr("d-root")}, emp, org) {
		t.Fatal("d-a1 应在 d-root 部门子树内")
	}
	if r.matchScope(models.ManagerRule{ScopeType: models.ManagerScopeDeptTree, ScopeID: strPtr("d-a1")}, leader, org) {
		t.Fatal("d-a 不应在 d-a1 子树内")
	}
	if !r.matchScope(models.ManagerRule{ScopeType: models.ManagerScopeOrgTree, ScopeID: strPtr("org-0")}, emp, org) {
		t.Fatal("org-1 应在 org-0 机构子树内")
	}
	holderC := r.users["u-c"]
	if r.matchScope(models.ManagerRule{ScopeType: models.ManagerScopeOrgTree, ScopeID: strPtr("org-1")}, &holderC, r.orgByID["org-0"]) {
		t.Fatal("org-0 不应在 org-1 子树内")
	}
	if r.matchScope(models.ManagerRule{ScopeType: 99}, emp, org) {
		t.Fatal("非法范围类型应返回 false")
	}
}

func TestManagerResolver_ResolveManagerID(t *testing.T) {
	r := buildTestResolver()
	emp := &models.User{BaseModel: models.BaseModel{ID: "u-a"}, DeptID: strPtr("d-a1")}
	org, _ := r.OrgOf(emp)

	// dept_leader：d-a1 无 leader → 回溯 d-a → u-leader
	id, _ := r.resolveManagerID(models.ManagerRule{ManagerType: models.ManagerSourceDeptLeader}, emp, org)
	if id != "u-leader" {
		t.Fatalf("dept_leader 应回溯到 u-leader, got %q", id)
	}
	// dept_leader：无部门 → 空
	id, _ = r.resolveManagerID(models.ManagerRule{ManagerType: models.ManagerSourceDeptLeader}, &models.User{BaseModel: models.BaseModel{ID: "x"}}, org)
	if id != "" {
		t.Fatalf("无部门应返回空, got %q", id)
	}
	// org_leader：org-1 节点 d-a 的 leader
	id, _ = r.resolveManagerID(models.ManagerRule{ManagerType: models.ManagerSourceOrgLeader}, emp, org)
	if id != "u-leader" {
		t.Fatalf("org_leader 应取机构节点 leader, got %q", id)
	}
	// org_leader + parent：org-1 的上级 org-0 → d-root leader
	id, _ = r.resolveManagerID(models.ManagerRule{ManagerType: models.ManagerSourceOrgLeader, ManagerOrgID: strPtr("parent")}, emp, org)
	if id != "u-root-leader" {
		t.Fatalf("org_leader(parent) 应取上级机构节点 leader, got %q", id)
	}
	// org_leader + parent 但无上级 → WARN + 空
	holderC := r.users["u-c"]
	var warns []string
	id, warns = r.resolveManagerID(models.ManagerRule{ManagerType: models.ManagerSourceOrgLeader, ManagerOrgID: strPtr("parent")}, &holderC, r.orgByID["org-0"])
	if id != "" || len(warns) == 0 {
		t.Fatalf("无上级机构应返回空且带 WARN, got id=%q warns=%v", id, warns)
	}
	// org_head：未配置岗位 → WARN
	id, warns = r.resolveManagerID(models.ManagerRule{ManagerType: models.ManagerSourceOrgHead}, emp, org)
	if id != "" || len(warns) == 0 {
		t.Fatalf("org_head 未配置岗位应返回空且带 WARN, got id=%q warns=%v", id, warns)
	}
	// post_holder：p-vp 在 org-1 内唯一有效任职人 u-b（过滤禁用/无dn/机构外）
	id, warns = r.resolveManagerID(models.ManagerRule{ManagerType: models.ManagerSourcePostHolder, ManagerPostID: strPtr("p-vp")}, emp, org)
	if id != "u-b" {
		t.Fatalf("post_holder 应选中 u-b, got %q (warns=%v)", id, warns)
	}
	if len(warns) != 0 {
		t.Fatalf("唯一任职人不应有 WARN, got %v", warns)
	}
	// 未知 manager_type → 空
	if id, _ = r.resolveManagerID(models.ManagerRule{ManagerType: "bogus"}, emp, org); id != "" {
		t.Fatalf("非法经理来源应返回空, got %q", id)
	}
}

func TestManagerResolver_PickPostHolderWarn(t *testing.T) {
	r := buildTestResolver()
	// 追加第二个有效任职人 u-ext 挪进 org-1 的部门（改 dept 为 d-a）→ 触发多人 WARN
	uExt := r.users["u-ext"]
	uExt.DeptID = strPtr("d-a")
	r.users["u-ext"] = uExt

	var warns []string
	id := r.pickPostHolder("p-vp", "org-1", &warns)
	if id != "u-b" {
		t.Fatalf("多人时应取 id 升序第一个 u-b, got %q", id)
	}
	if len(warns) != 1 || warns[0] == "" {
		t.Fatalf("多人任职应产生一条 WARN, got %v", warns)
	}
	// 无人可选 → 空且无 WARN
	warns = nil
	if id = r.pickPostHolder("p-none", "org-1", &warns); id != "" || len(warns) != 0 {
		t.Fatalf("无任职人应返回空, got %q warns=%v", id, warns)
	}
}

func TestManagerResolver_ResolveDeptLeaderDepth(t *testing.T) {
	r := buildTestResolver()
	// d-a1 无 leader → 回溯 d-a → u-leader
	if id := r.resolveDeptLeader(nil); id != "" {
		t.Fatalf("nil 部门应返回空, got %q", id)
	}
	dA1 := r.depts["d-a1"]
	if id := r.resolveDeptLeader(&dA1); id != "u-leader" {
		t.Fatalf("应回溯祖先 leader, got %q", id)
	}
	// 无 leader 链耗尽 → 空（ancestors 去掉自身后无祖先，链上无 leader 可回溯）
	deep := models.Department{BaseModel: models.BaseModel{ID: "d-x"}, Ancestors: "0,d-x"}
	if id := r.resolveDeptLeader(&deep); id != "" {
		t.Fatalf("祖先链均无 leader 应返回空, got %q", id)
	}
}

func TestManagerResolver_SubtreeAndHelpers(t *testing.T) {
	r := buildTestResolver()
	if !r.DeptInSubtree("d-root", strPtr("d-root")) {
		t.Fatal("子树判断应含自身")
	}
	if r.DeptInSubtree("d-root", nil) {
		t.Fatal("nil 部门不在子树")
	}
	if !r.OrgInSubtree("org-0", "org-1") {
		t.Fatal("org-1 应在 org-0 子树内")
	}
	if !r.OrgInSubtree("org-1", "org-1") {
		t.Fatal("子树判断应含自身")
	}
	if r.OrgInSubtree("org-x", "org-1") {
		t.Fatal("无关机构不在子树")
	}
	// 名称翻译
	if r.PostName("") != "" || r.PostName("p-vp") != "分管副总" {
		t.Fatal("PostName 翻译错误")
	}
	if r.DeptName("") != "" || r.DeptName("d-a") != "中支" {
		t.Fatal("DeptName 翻译错误")
	}
	if r.OrgName("") != "" || r.OrgName("org-1") != "中支机构" {
		t.Fatal("OrgName 翻译错误")
	}
	if ptrValue(nil) != "" || ptrValue(strPtr("v")) != "v" {
		t.Fatal("ptrValue 语义错误")
	}
	// addPostHolder 去重
	r.addPostHolder("p-x", "u-a")
	r.addPostHolder("p-x", "u-a")
	if len(r.postHolders["p-x"]) != 1 || len(r.userPosts["u-a"]) != 1 {
		t.Fatal("addPostHolder 应去重")
	}
	if !r.HasPost("u-a", "p-x") || r.HasPost("u-a", "p-y") {
		t.Fatal("HasPost 判断错误")
	}
	if !r.IsDeptLeader("u-leader") || r.IsDeptLeader("u-a") {
		t.Fatal("IsDeptLeader 判断错误")
	}
}

func TestIsUUIDLike(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"123E4567-E89B-12D3-A456-426614174000", true},
		{"123e4567e89b12d3a456426614174000", false},  // 缺连字符
		{"123e4567-e89b-12d3-a456-42661417400", false}, // 长度 35
		{"123e4567_e89b-12d3-a456-426614174000", false},
		{"123e4567-g89b-12d3-a456-426614174000", false}, // 非法字符 g
		{"", false},
		{"parent", false},
	}
	for _, c := range cases {
		if got := isUUIDLike(c.in); got != c.want {
			t.Errorf("isUUIDLike(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestManagerRuleValidate(t *testing.T) {
	valid := func(m func(*ManagerRuleCreateRequest)) *ManagerRuleCreateRequest {
		req := &ManagerRuleCreateRequest{
			RuleName: "规则", SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader,
		}
		m(req)
		return req
	}
	if err := valid(func(r *ManagerRuleCreateRequest) {}).validate(); err != nil {
		t.Fatalf("合法基线请求不应报错: %v", err)
	}
	errCases := []struct {
		name string
		req  *ManagerRuleCreateRequest
	}{
		{"空名称", valid(func(r *ManagerRuleCreateRequest) { r.RuleName = "  " })},
		{"非法主体", valid(func(r *ManagerRuleCreateRequest) { r.SubjectType = "bogus" })},
		{"非法来源", valid(func(r *ManagerRuleCreateRequest) { r.ManagerType = "bogus" })},
		{"岗位来源缺岗位", valid(func(r *ManagerRuleCreateRequest) { r.ManagerType = models.ManagerSourcePostHolder })},
		{"范围缺节点", valid(func(r *ManagerRuleCreateRequest) { r.ScopeType = models.ManagerScopeDeptTree })},
		{"非法范围类型", valid(func(r *ManagerRuleCreateRequest) { r.ScopeType = 99 })},
		{"目标机构脏值", valid(func(r *ManagerRuleCreateRequest) { r.ManagerOrgID = strPtr("not-a-uuid") })},
		{"主体机构脏值", valid(func(r *ManagerRuleCreateRequest) { r.SubjectOrgID = strPtr("dept-name") })},
	}
	for _, c := range errCases {
		if err := c.req.validate(); err == nil {
			t.Errorf("%s: 应校验失败", c.name)
		}
	}
	// 合法变体：parent 目标机构 + OrgTree 范围带节点
	okReq := valid(func(r *ManagerRuleCreateRequest) {
		r.ManagerOrgID = strPtr(models.ManagerOrgSpecialParent)
		r.ScopeType = models.ManagerScopeOrgTree
		r.ScopeID = strPtr("123e4567-e89b-12d3-a456-426614174000")
	})
	if err := okReq.validate(); err != nil {
		t.Fatalf("parent 目标机构应合法: %v", err)
	}
}

func TestManagerOrgValidate(t *testing.T) {
	base := ManagerOrgCreateRequest{OrgName: "机构", DeptID: "d-1", OrgType: 1}
	if err := base.validate(); err != nil {
		t.Fatalf("合法机构请求不应报错: %v", err)
	}
	for name, mut := range map[string]func(*ManagerOrgCreateRequest){
		"空名称":  func(r *ManagerOrgCreateRequest) { r.OrgName = "" },
		"空部门":  func(r *ManagerOrgCreateRequest) { r.DeptID = " " },
		"非法类型": func(r *ManagerOrgCreateRequest) { r.OrgType = 3 },
		"负数类型": func(r *ManagerOrgCreateRequest) { r.OrgType = -1 },
	} {
		req := base
		mut(&req)
		if err := req.validate(); err == nil {
			t.Errorf("%s: 应校验失败", name)
		}
	}
}

// ============================ 端到端：SQLite 集成 ============================

func TestHasEnabledRules(t *testing.T) {
	f := setupMgrFixture(t)
	ctx := context.Background()
	svc := NewManagerRuleService(f.db)

	// 空表 → false
	has, err := svc.HasEnabledRules(ctx)
	if err != nil || has {
		t.Fatalf("空表应返回 (false,nil), got %v %v", has, err)
	}
	// 只有停用规则 → false
	// ⚠ Enabled 列有 gorm default:true，结构体 Create 的零值 false 会被 DB 默认值
	// 覆盖为 true（GORM 零值语义），必须用 map 显式写 false。
	if err := svc.db.Model(&models.ManagerRule{}).Create(map[string]interface{}{
		"id": "rule-off", "rule_name": "停用", "enabled": false,
		"subject_type": models.ManagerSubjectEmployee, "manager_type": models.ManagerSourceDeptLeader,
	}).Error; err != nil {
		t.Fatalf("建停用规则失败: %v", err)
	}
	if has, _ = svc.HasEnabledRules(ctx); has {
		t.Fatal("仅有停用规则应返回 false")
	}
	// 启用规则 → true
	if err := svc.db.Create(&models.ManagerRule{RuleName: "启用", Enabled: true, SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader}).Error; err != nil {
		t.Fatalf("建启用规则失败: %v", err)
	}
	if has, err = svc.HasEnabledRules(ctx); err != nil || !has {
		t.Fatalf("有启用规则应返回 true, got %v %v", has, err)
	}
	// nil db → (false, nil) 兼容语义
	if has, err := NewManagerRuleService(nil).HasEnabledRules(ctx); has || err != nil {
		t.Fatalf("nil db 应返回 (false,nil), got %v %v", has, err)
	}
}

func TestResolveManagers_EndToEnd(t *testing.T) {
	f := setupMgrFixture(t)
	ctx := context.Background()
	svc := NewManagerRuleService(f.db)

	rules := []models.ManagerRule{
		// R0：一把手 → 上级机构一把手。gm 的 org=org-1，parent=org-0，
		// p-gm 在 org-0 的任职人就是 gm 本人 → 自指 → 顺延 R1（不短路）。
		{RuleName: "一把手→上级一把手", Priority: 0, Enabled: true, SubjectType: models.ManagerSubjectOrgHead, SubjectPostID: strPtr("p-gm"), ManagerType: models.ManagerSourceOrgHead, ManagerPostID: strPtr("p-gm"), ManagerOrgID: strPtr("parent")},
		{RuleName: "员工→部门负责人", Priority: 1, Enabled: true, SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader, ScopeType: models.ManagerScopeOrgTree, ScopeID: strPtr("org-1")},
		{RuleName: "负责人→分管副总", Priority: 2, Enabled: true, SubjectType: models.ManagerSubjectDeptLeader, ManagerType: models.ManagerSourcePostHolder, ManagerPostID: strPtr("p-vp")},
	}
	if err := f.db.Create(&rules).Error; err != nil {
		t.Fatalf("建规则失败: %v", err)
	}
	invalidateResolverCache()

	res, err := svc.ResolveManagers(ctx, []models.User{f.emp, f.leader, f.gm})
	if err != nil {
		t.Fatalf("ResolveManagers 失败: %v", err)
	}
	byID := map[string]ManagerResolution{}
	for _, r := range res {
		byID[r.UserID] = r
	}

	// 普通员工：R0 主体 org_head 不匹配 → R1 命中 → d-a1 无 leader → 回溯 d-a → u-leader
	if got := byID["u-emp"]; got.Skipped || got.ManagerID != "u-leader" || got.RuleID != rules[1].ID {
		t.Fatalf("员工应解析到 u-leader（R1）, got %+v", got)
	}
	// 部门负责人：R2 命中 → p-vp 任职人就是自己 → 自指顺延 R3（主体 org_head 不匹配）
	// → 最终 Skipped 且保留 R2 的自指原因（first-match 不短路于自指）。
	if got := byID["u-leader"]; !got.Skipped || !contains(got.Reason, "自指") {
		t.Fatalf("负责人自指场景应 Skipped 且原因含「自指」, got %+v", got)
	}
	// 一把手：R0 主体匹配（gm 是 p-gm 任职人 + 有机构），manager_org=parent
	// → org-0 内 p-gm 任职人 u-boss → 经理 u-boss（跨机构取人路径）。
	if got := byID["u-gm"]; got.Skipped || got.ManagerID != "u-boss" || got.RuleID != rules[0].ID {
		t.Fatalf("一把手应经 R0(parent) 解析到 u-boss, got %+v", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestResolveOne_NoOrgSkips(t *testing.T) {
	f := setupMgrFixture(t)
	ctx := context.Background()
	svc := NewManagerRuleService(f.db)
	if err := f.db.Create(&models.ManagerRule{RuleName: "兜底", Enabled: true, SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader}).Error; err != nil {
		t.Fatalf("建规则失败: %v", err)
	}
	invalidateResolverCache()

	// 无部门用户 → 无机构 → Skipped
	res, err := svc.ResolveOne(ctx, &models.User{BaseModel: models.BaseModel{ID: "u-x"}, Username: "orphan"})
	if err != nil {
		t.Fatalf("ResolveOne 失败: %v", err)
	}
	if !res.Skipped || res.Reason == "" {
		t.Fatalf("无机构用户应 Skipped, got %+v", res)
	}
}

func TestResolveManagerDN_BothPaths(t *testing.T) {
	f := setupMgrFixture(t)
	ctx := context.Background()
	svc := NewManagerRuleService(f.db)

	// ---- 路径 1：无启用规则 → 回退 leader 递归祖先 ----
	dn, reason, err := svc.ResolveManagerDN(ctx, &f.emp)
	if err != nil {
		t.Fatalf("回退路径失败: %v", err)
	}
	if dn != "CN=leader,DC=test" || reason != "" {
		t.Fatalf("回退路径应返回 leader 的 dn, got dn=%q reason=%q", dn, reason)
	}
	// 无部门 → 跳过原因
	dn, reason, err = svc.ResolveManagerDN(ctx, &models.User{BaseModel: models.BaseModel{ID: "u-x"}})
	if err != nil || dn != "" || reason == "" {
		t.Fatalf("无部门应返回跳过原因, got dn=%q reason=%q err=%v", dn, reason, err)
	}
	// 经理为本人（d-root 无 leader；把 emp 设为 d-a leader 之外的场景：单部门自 leader）
	if err := f.db.Model(&models.Department{}).Where("id = ?", "d-a").Update("leader", "u-emp").Error; err != nil {
		t.Fatalf("改 leader 失败: %v", err)
	}
	dn, reason, err = svc.ResolveManagerDN(ctx, &f.emp)
	if err != nil || dn != "" || !contains(reason, "自指") {
		t.Fatalf("负责人为本人应跳过, got dn=%q reason=%q err=%v", dn, reason, err)
	}

	// ---- 路径 2：有启用规则 → 规则引擎 + ad_dn 查找 ----
	if err := f.db.Model(&models.Department{}).Where("id = ?", "d-a").Update("leader", "u-leader").Error; err != nil {
		t.Fatalf("恢复 leader 失败: %v", err)
	}
	if err := f.db.Create(&models.ManagerRule{RuleName: "员工→负责人", Enabled: true, SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader, ScopeType: models.ManagerScopeOrgTree, ScopeID: strPtr("org-1")}).Error; err != nil {
		t.Fatalf("建规则失败: %v", err)
	}
	invalidateResolverCache()
	dn, reason, err = svc.ResolveManagerDN(ctx, &f.emp)
	if err != nil {
		t.Fatalf("规则路径失败: %v", err)
	}
	if dn != "CN=leader,DC=test" || reason != "" {
		t.Fatalf("规则路径应返回 leader dn, got dn=%q reason=%q", dn, reason)
	}

	// 经理无 ad_dn → 跳过原因
	if err := f.db.Model(&models.User{}).Where("id = ?", "u-leader").Update("ad_dn", "").Error; err != nil {
		t.Fatalf("清 ad_dn 失败: %v", err)
	}
	invalidateResolverCache()
	dn, reason, err = svc.ResolveManagerDN(ctx, &f.emp)
	if err != nil || dn != "" || !contains(reason, "ad_dn") {
		t.Fatalf("经理无 ad_dn 应跳过, got dn=%q reason=%q err=%v", dn, reason, err)
	}
}

func TestPreviewManagers(t *testing.T) {
	f := setupMgrFixture(t)
	ctx := context.Background()
	svc := NewManagerRuleService(f.db)

	// 空入参 → 空列表
	empty, err := svc.PreviewManagers(ctx, nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("空入参应返回空列表, got %v %v", empty, err)
	}

	if err := svc.db.Create(&models.ManagerRule{RuleName: "员工→负责人", Enabled: true, SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader, ScopeType: models.ManagerScopeOrgTree, ScopeID: strPtr("org-1")}).Error; err != nil {
		t.Fatalf("建规则失败: %v", err)
	}
	invalidateResolverCache()

	items, err := svc.PreviewManagers(ctx, []string{"u-emp", "u-gone"})
	if err != nil {
		t.Fatalf("PreviewManagers 失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("未知用户应被跳过, got %d 项", len(items))
	}
	item := items[0]
	if item.UserID != "u-emp" || item.ManagerID != "u-leader" {
		t.Fatalf("预览应解析出经理, got %+v", item)
	}
	if item.ManagerName != "王 leader" || item.ManagerUsername != "leader" {
		t.Fatalf("预览应带经理姓名/账号, got %+v", item)
	}
}

func TestManagerRuleCRUD(t *testing.T) {
	f := setupMgrFixture(t)
	ctx := context.Background()
	svc := NewManagerRuleService(f.db)

	// Create：合法请求
	created, err := svc.CreateRule(ctx, &ManagerRuleCreateRequest{
		RuleName: "  规则A  ", Priority: 5, SubjectType: models.ManagerSubjectEmployee,
		ManagerType: models.ManagerSourceDeptLeader, ScopeType: models.ManagerScopeOrgTree, ScopeID: strPtr("org-1"),
		ManagerOrgID: strPtr(models.ManagerOrgSpecialParent),
	})
	if err != nil {
		t.Fatalf("CreateRule 失败: %v", err)
	}
	if created.RuleName != "规则A" || !created.Enabled {
		t.Fatalf("创建结果不符（应 trim 名称、默认启用）: %+v", created)
	}
	// Create：nil 请求 / 校验失败
	if _, err := svc.CreateRule(ctx, nil); err == nil {
		t.Fatal("nil 请求应报错")
	}
	if _, err := svc.CreateRule(ctx, &ManagerRuleCreateRequest{SubjectType: "x", ManagerType: "y"}); err == nil {
		t.Fatal("非法请求应报错")
	}

	// Get：存在 / 不存在
	got, err := svc.GetRule(ctx, created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("GetRule 失败: %v", err)
	}
	if _, err := svc.GetRule(ctx, "no-such-id"); err == nil {
		t.Fatal("不存在的规则应报错")
	}

	// Update：合法 + Enabled=false
	if err := svc.UpdateRule(ctx, created.ID, &ManagerRuleCreateRequest{
		RuleName: "规则A改", Priority: 1, Enabled: boolPtr(false),
		SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader,
	}); err != nil {
		t.Fatalf("UpdateRule 失败: %v", err)
	}
	got, _ = svc.GetRule(ctx, created.ID)
	if got.RuleName != "规则A改" || got.Enabled || got.Priority != 1 {
		t.Fatalf("更新结果不符: %+v", got)
	}
	if err := svc.UpdateRule(ctx, "no-such-id", &ManagerRuleCreateRequest{RuleName: "x", SubjectType: models.ManagerSubjectEmployee, ManagerType: models.ManagerSourceDeptLeader}); err == nil {
		t.Fatal("更新不存在的规则应报错")
	}
	if err := svc.UpdateRule(ctx, created.ID, nil); err == nil {
		t.Fatal("nil 请求应报错")
	}

	// List：keyword / enabled 过滤 + 名称翻译
	disabled := models.ManagerOrg{}
	_ = disabled
	if err := svc.db.Create(&models.ManagerRule{RuleName: "另一条", Enabled: true, SubjectType: models.ManagerSubjectDeptLeader, ManagerType: models.ManagerSourceOrgHead, ManagerPostID: strPtr("p-gm"), ManagerOrgID: strPtr("org-1")}).Error; err != nil {
		t.Fatalf("建第二条规则失败: %v", err)
	}
	page, err := svc.ListRules(ctx, 1, 10, "规则A", nil)
	if err != nil {
		t.Fatalf("ListRules 失败: %v", err)
	}
	// 「规则A」前缀命中更新后的「规则A改」（LIKE 前缀匹配）
	if page.Total != 1 || len(page.List) != 1 || page.List[0].RuleName != "规则A改" {
		t.Fatalf("keyword 过滤结果不符: total=%d list=%+v", page.Total, page.List)
	}
	enabled := false
	page, err = svc.ListRules(ctx, 1, 10, "", &enabled)
	if err != nil {
		t.Fatalf("ListRules(enabled=false) 失败: %v", err)
	}
	if page.Total != 1 || page.List[0].RuleName != "规则A改" {
		t.Fatalf("enabled 过滤应只含停用规则, got total=%d", page.Total)
	}
	// 翻译：第二条规则带机构/岗位/范围名称
	page, err = svc.ListRules(ctx, 1, 10, "另一条", nil)
	if err != nil {
		t.Fatalf("ListRules 失败: %v", err)
	}
	if page.List[0].ManagerPostName != "总经理" || page.List[0].ManagerOrgName != "武汉中支机构" {
		t.Fatalf("名称翻译不符: %+v", page.List[0])
	}
	// 分页边界归一
	page, err = svc.ListRules(ctx, 0, 0, "", nil)
	if err != nil || page.Current != 1 || page.PageSize != 10 {
		t.Fatalf("非法分页参数应归一, got current=%d pageSize=%d err=%v", page.Current, page.PageSize, err)
	}

	// Delete
	if err := svc.DeleteRule(ctx, created.ID); err != nil {
		t.Fatalf("DeleteRule 失败: %v", err)
	}
	if _, err := svc.GetRule(ctx, created.ID); err == nil {
		t.Fatal("删除后查询应报错")
	}
}

func TestManagerOrgCRUD(t *testing.T) {
	f := setupMgrFixture(t)
	ctx := context.Background()
	svc := NewManagerOrgService(f.db)

	// Create：d-a1 未被占用；d-root 已被 fixture 的 org-0 占用 → 唯一性校验拒绝
	org, err := svc.CreateOrg(ctx, &ManagerOrgCreateRequest{OrgName: "新机构", DeptID: "d-a1", OrgType: 2})
	if err != nil {
		t.Fatalf("CreateOrg 失败: %v", err)
	}
	if !org.Enabled || org.OrgName != "新机构" {
		t.Fatalf("创建结果不符: %+v", org)
	}
	if _, err := svc.CreateOrg(ctx, &ManagerOrgCreateRequest{OrgName: "重复", DeptID: "d-root", OrgType: 0}); err == nil {
		t.Fatal("占用已被绑定的部门应被校验拒绝")
	}
	if _, err := svc.CreateOrg(ctx, nil); err == nil {
		t.Fatal("nil 请求应报错")
	}
	if _, err := svc.CreateOrg(ctx, &ManagerOrgCreateRequest{OrgName: "x", DeptID: "", OrgType: 0}); err == nil {
		t.Fatal("空部门应报错")
	}

	// Get / List
	got, err := svc.GetOrg(ctx, org.ID)
	if err != nil || got.ID != org.ID {
		t.Fatalf("GetOrg 失败: %v", err)
	}
	if _, err := svc.GetOrg(ctx, "nope"); err == nil {
		t.Fatal("不存在的机构应报错")
	}
	page, err := svc.ListOrgs(ctx, 1, 10, "", nil)
	if err != nil {
		t.Fatalf("ListOrgs 失败: %v", err)
	}
	if page.Total != 3 { // fixture 的 org-0/org-1 + 新建
		t.Fatalf("机构总数应为 3, got %d", page.Total)
	}
	// 翻译：parent 名 + dept 名
	page, err = svc.ListOrgs(ctx, 1, 10, "武汉", nil)
	if err != nil || page.Total != 1 {
		t.Fatalf("keyword 过滤失败: %v", err)
	}
	item := page.List[0]
	if item.DeptName != "武汉中支" || item.ParentOrgName != "总公司机构" {
		t.Fatalf("机构列表翻译不符: %+v", item)
	}
	// Update / Delete（保持绑定自己已占用的部门，excludeOrgID 应放行）
	if err := svc.UpdateOrg(ctx, org.ID, &ManagerOrgCreateRequest{OrgName: "新机构改", DeptID: "d-a1", OrgType: 2, Enabled: boolPtr(false)}); err != nil {
		t.Fatalf("UpdateOrg 失败: %v", err)
	}
	got, _ = svc.GetOrg(ctx, org.ID)
	if got.OrgName != "新机构改" || got.Enabled || got.OrgType != 2 {
		t.Fatalf("更新结果不符: %+v", got)
	}
	if err := svc.DeleteOrg(ctx, org.ID); err != nil {
		t.Fatalf("DeleteOrg 失败: %v", err)
	}
	if _, err := svc.GetOrg(ctx, org.ID); err == nil {
		t.Fatal("删除后查询应报错")
	}
}
