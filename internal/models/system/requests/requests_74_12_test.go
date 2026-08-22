package requests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// 74-11 escalation gap-closure: internal/models/system/requests —
// 各模块 ListParams 默认值/分页 + ToModel 转换。
// =====================================================================

func intPtr(v int) *int { return &v }

func TestListParamsDefaults(t *testing.T) {
	// 方法为 pointer receiver → 用闭包统一
	cases := []struct {
		name string
		got  func() (int, int)
		off  func() int
	}{
		{"apikeys", func() (int, int) { p := DefaultListAPIKeysParams(); return p.GetPagination() }, func() int { p := DefaultListAPIKeysParams(); return p.GetOffset() }},
		{"config", func() (int, int) { p := DefaultConfigListParams(); return p.GetPagination() }, func() int { p := DefaultConfigListParams(); return p.GetOffset() }},
		{"dicttype", func() (int, int) { p := DefaultDictTypeListParams(); return p.GetPagination() }, func() int { p := DefaultDictTypeListParams(); return p.GetOffset() }},
		{"dictdata", func() (int, int) { p := DefaultDictDataListParams(); return p.GetPagination() }, func() int { p := DefaultDictDataListParams(); return p.GetOffset() }},
		{"notice", func() (int, int) { p := DefaultNoticeListParams(); return p.GetPagination() }, func() int { p := DefaultNoticeListParams(); return p.GetOffset() }},
		{"post", func() (int, int) { p := DefaultPostListParams(); return p.GetPagination() }, func() int { p := DefaultPostListParams(); return p.GetOffset() }},
		{"role", func() (int, int) { p := DefaultRoleListParams(); return p.GetPagination() }, func() int { p := DefaultRoleListParams(); return p.GetOffset() }},
		{"user", func() (int, int) { p := DefaultUserListParams(); return p.GetPagination() }, func() int { p := DefaultUserListParams(); return p.GetOffset() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			current, pageSize := tc.got()
			assert.Equal(t, 1, current)
			assert.Equal(t, 10, pageSize)
			assert.Equal(t, 0, tc.off())
		})
	}
}

func TestUserCreateRequest_ToModel(t *testing.T) {
	remark := "ops"
	r := UserCreateRequest{
		Username:   "alice",
		Nickname:   strptr("Alice"),
		EmployeeNo: strptr("E001"),
		Email:      strptr("a@x.com"),
		Phone:      strptr("13800000000"),
		Gender:     models.GenderFemale,
		Status:     models.UserStatusEnabled,
		DeptID:     strptr("d1"),
		Remark:     &remark,
	}
	u := (&r).ToModel("hashed-pwd")
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "hashed-pwd", u.Password)
	assert.Equal(t, "Alice", *u.Nickname)
	assert.Equal(t, models.GenderFemale, u.Gender)
	assert.Equal(t, models.UserStatusEnabled, u.Status)
	assert.Equal(t, "ops", u.Remark)

	// nil 指针 → 空值
	r2 := UserCreateRequest{Username: "bob"}
	u2 := (&r2).ToModel("p")
	assert.Empty(t, u2.Remark)
	assert.Nil(t, u2.DeptID)
}

func TestUserUpdateRequest_ToModel(t *testing.T) {
	nick := "NewNick"
	dept := "d2"
	r := UserUpdateRequest{
		ID:       "u-1",
		Nickname: &nick,
		DeptID:   &dept,
		Status:   models.UserStatusDisabled,
	}
	u := (&r).ToModel()
	require.NotNil(t, u.Nickname)
	assert.Equal(t, "NewNick", *u.Nickname)
	assert.Equal(t, "d2", *u.DeptID)
	assert.Equal(t, models.UserStatusDisabled, u.Status)
}

func TestDepartmentCreateRequest_ToModel(t *testing.T) {
	r := DepartmentCreateRequest{
		DeptName: "研发部",
		ParentID: strptr("p1"),
		OrderNum: 2,
	}
	d := (&r).ToModel("/0/p1/")
	assert.Equal(t, "研发部", d.DeptName)
	assert.Equal(t, "/0/p1/", d.Ancestors)
}

func TestMenuCreateRequest_ToModel(t *testing.T) {
	parentID := "m-0"
	path := "/system"
	component := "system/user/index"
	r := MenuCreateRequest{
		MenuName:  "用户管理",
		ParentID:  &parentID,
		Path:      &path,
		Component: &component,
		MenuType:  models.MenuTypeDir,
	}
	m := (&r).ToModel()
	assert.Equal(t, "用户管理", m.MenuName)
	assert.Equal(t, "m-0", *m.ParentID)
	assert.Equal(t, "/system", *m.Path)
	assert.Equal(t, models.MenuTypeDir, m.MenuType)

	// normalizeParentID: nil/""/"0" 全部归一为 nil
	assert.Nil(t, (&MenuCreateRequest{MenuName: "根"}).ToModel().ParentID)
	empty := ""
	assert.Nil(t, (&MenuCreateRequest{MenuName: "x", ParentID: &empty}).ToModel().ParentID)
	zero := "0"
	assert.Nil(t, (&MenuCreateRequest{MenuName: "x", ParentID: &zero}).ToModel().ParentID)

	// stringPtrValue 辅助
	assert.Equal(t, "", stringPtrValue(nil))
	pv := "v"
	assert.Equal(t, "v", stringPtrValue(&pv))
}

func strptr(s string) *string { return &s }

var _ = intPtr // 占位(未来扩展)
