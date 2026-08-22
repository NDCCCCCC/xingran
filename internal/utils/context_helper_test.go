package utils

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func newGinCtx() *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestGetUserID(t *testing.T) {
	c := newGinCtx()
	assert.Equal(t, "", GetUserID(c))
	c.Set("user_id", "u1")
	assert.Equal(t, "u1", GetUserID(c))

	c2 := newGinCtx()
	c2.Set("user_id", 123)
	assert.Equal(t, "", GetUserID(c2), "非 string 类型应返空")
}

func TestGetUsername(t *testing.T) {
	c := newGinCtx()
	assert.Equal(t, "", GetUsername(c))
	c.Set("username", "alice")
	assert.Equal(t, "alice", GetUsername(c))
}

func TestGetUsernamePtr(t *testing.T) {
	c := newGinCtx()
	// 无 username → "unknown"
	p := GetUsernamePtr(c)
	require.NotNil(t, p)
	assert.Equal(t, "unknown", *p)

	// 空 string → "unknown"
	c.Set("username", "")
	p = GetUsernamePtr(c)
	require.NotNil(t, p)
	assert.Equal(t, "unknown", *p)

	// 有值
	c.Set("username", "alice")
	p = GetUsernamePtr(c)
	assert.Equal(t, "alice", *p)
}

func TestGetNicknamePtr(t *testing.T) {
	c := newGinCtx()
	assert.Nil(t, GetNicknamePtr(c))
	c.Set("nickname", "")
	assert.Nil(t, GetNicknamePtr(c))
	c.Set("nickname", "Nick")
	p := GetNicknamePtr(c)
	require.NotNil(t, p)
	assert.Equal(t, "Nick", *p)
}

func newCtxDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE sys_user (id TEXT, nickname TEXT, dept_name TEXT)`).Error)
	return db
}

func TestGetNicknamePtrWithDB(t *testing.T) {
	db := newCtxDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u1', '张三', '研发部')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, nickname, dept_name) VALUES ('u2', NULL, NULL)`).Error)

	// context 已有 nickname → 直接返回,不走 DB
	c := newGinCtx()
	c.Set("nickname", "CtxNick")
	p := GetNicknamePtrWithDB(c, db)
	require.NotNil(t, p)
	assert.Equal(t, "CtxNick", *p)

	// context 无 nickname + nil db → nil
	c2 := newGinCtx()
	c2.Set("user_id", "u1")
	assert.Nil(t, GetNicknamePtrWithDB(c2, nil))

	// 无 user_id → nil
	c3 := newGinCtx()
	assert.Nil(t, GetNicknamePtrWithDB(c3, db))

	// DB 命中
	c4 := newGinCtx()
	c4.Set("user_id", "u1")
	p = GetNicknamePtrWithDB(c4, db)
	require.NotNil(t, p)
	assert.Equal(t, "张三", *p)

	// NULL nickname → nil
	c5 := newGinCtx()
	c5.Set("user_id", "u2")
	assert.Nil(t, GetNicknamePtrWithDB(c5, db))
}

func TestGetDeptName(t *testing.T) {
	c := newGinCtx()
	assert.Nil(t, GetDeptName(c))
	c.Set("dept_name", "")
	assert.Nil(t, GetDeptName(c))
	c.Set("dept_name", "研发部")
	p := GetDeptName(c)
	require.NotNil(t, p)
	assert.Equal(t, "研发部", *p)
}

func TestGetDeptNameFromDB(t *testing.T) {
	db := newCtxDB(t)
	require.NoError(t, db.Exec(`INSERT INTO sys_user VALUES ('u1', 'nick', '研发部')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, nickname, dept_name) VALUES ('u2', 'x', NULL)`).Error)

	// context 已有
	c := newGinCtx()
	c.Set("dept_name", "CtxDept")
	p := GetDeptNameFromDB(c, db)
	require.NotNil(t, p)
	assert.Equal(t, "CtxDept", *p)

	// 无 user_id → nil
	c2 := newGinCtx()
	assert.Nil(t, GetDeptNameFromDB(c2, db))

	// DB 命中
	c3 := newGinCtx()
	c3.Set("user_id", "u1")
	p = GetDeptNameFromDB(c3, db)
	require.NotNil(t, p)
	assert.Equal(t, "研发部", *p)

	// NULL dept_name → nil (F-03 兼容)
	c4 := newGinCtx()
	c4.Set("user_id", "u2")
	assert.Nil(t, GetDeptNameFromDB(c4, db))
}

func TestGetClientIP(t *testing.T) {
	c := newGinCtx()
	c.Request = httptest.NewRequest("GET", "/", nil)
	_ = GetClientIP(c) // 不 panic 即可
}

func TestGetRequiredParam(t *testing.T) {
	c := newGinCtx()
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	v, ok := GetRequiredParam(c, "id", "ID")
	require.True(t, ok)
	assert.Equal(t, "abc", v)

	// 空 → false
	c2 := newGinCtx()
	c2.Params = gin.Params{{Key: "id", Value: ""}}
	_, ok = GetRequiredParam(c2, "id", "ID")
	assert.False(t, ok)
}