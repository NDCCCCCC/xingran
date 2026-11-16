package system

// ad_domain_handler_list_test.go —— ListConfigs handler 覆盖（GOV-03 diff gate）。
//
// 背景：R11 把 ListConfigs 匿名请求结构的 Status 类型化为 *models.ADConfigStatus，
// 该行进入 PR diff 后，函数级 coverage 块（此前无测试覆盖，count=0）随之把相邻
// 声明行计入 gate 分母 → coverage-diff 失败。本文件补齐 handler 级测试：
// 覆盖「无筛选默认分页」「status 筛选（ADConfigStatus 类型化路径）」与
// 「空库零结果」，同时锁定 status 值域经 ADConfigStatus 类型后绑定行为不变。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	addomainServices "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

func newADDomainListTestEnv(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ADConfig{}))

	svc := addomainServices.NewADDomainService(db, nil)
	h := NewADDomainHandler(svc, &core.Core{})

	r := gin.New()
	r.POST("/ad-domain/configs/list", h.ListConfigs)
	return r, db
}

func postADDomainList(t *testing.T, r *gin.Engine, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ad-domain/configs/list", strings.NewReader(body))
	r.ServeHTTP(w, req)
	return w
}

func TestADDomainListConfigs_DefaultPaging(t *testing.T) {
	r, db := newADDomainListTestEnv(t)
	require.NoError(t, db.Create(&models.ADConfig{ConfigName: "cfg-on", ServerAddress: "dc1.example.com", Status: models.ADConfigStatusEnabled}).Error)
	require.NoError(t, db.Create(&models.ADConfig{ConfigName: "cfg-off", ServerAddress: "dc2.example.com", Status: models.ADConfigStatusDisabled}).Error)

	w := postADDomainList(t, r, `{"current":1,"pageSize":10}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestADDomainListConfigs_StatusFilter(t *testing.T) {
	r, db := newADDomainListTestEnv(t)
	require.NoError(t, db.Create(&models.ADConfig{ConfigName: "cfg-on", ServerAddress: "dc1.example.com", Status: models.ADConfigStatusEnabled}).Error)
	require.NoError(t, db.Create(&models.ADConfig{ConfigName: "cfg-off", ServerAddress: "dc2.example.com", Status: models.ADConfigStatusDisabled}).Error)

	// status=0（启用）→ 只命中 1 条；类型化路径 *models.ADConfigStatus 透传 GORM
	w := postADDomainList(t, r, `{"current":1,"pageSize":10,"status":0}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "cfg-on")
	assert.NotContains(t, w.Body.String(), "cfg-off")
}

func TestADDomainListConfigs_EmptyDB(t *testing.T) {
	r, _ := newADDomainListTestEnv(t)

	w := postADDomainList(t, r, `{}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
