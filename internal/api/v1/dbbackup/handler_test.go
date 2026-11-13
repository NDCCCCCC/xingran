package dbbackup

// handler_test.go —— GetPlan 出参脱敏（maskPlanSecrets）与 SavePlan 入参
// 占位符回填（mergePlanMaskedSecrets）的函数级单测：不启动 gin、不连 DB，
// 用嵌入接口的 fake service 只覆写被测路径实际调用的方法。

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/services/dbbackup"
)

// newTestGinContext 构造带合法 Request 的最小 gin context
// （mergePlanMaskedSecrets 内部读取 c.Request.Context()，不能传 nil）。
func newTestGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/plan", nil)
	return c
}

// fakeSvc 仅覆写 GetPlan/SavePlan；其余方法在测试路径中不会被调用。
type fakeSvc struct {
	dbbackup.Service
	plan    *dbbackup.Plan
	planErr error
	getCalls int
}

func (f *fakeSvc) GetPlan(context.Context) (*dbbackup.Plan, error) {
	f.getCalls++
	return f.plan, f.planErr
}

func TestMaskPlanSecrets_MasksAllNonEmpty(t *testing.T) {
	p := &dbbackup.Plan{}
	p.OSS.AccessKey = "AK"
	p.OSS.SecretKey = "SK"
	p.FTP.Password = "PWD"
	maskPlanSecrets(p)
	if p.OSS.AccessKey != dbbackup.PlanMaskPlaceholder ||
		p.OSS.SecretKey != dbbackup.PlanMaskPlaceholder ||
		p.FTP.Password != dbbackup.PlanMaskPlaceholder {
		t.Fatalf("非空凭据应全部脱敏: %+v", p)
	}
}

func TestMaskPlanSecrets_KeepsEmptyAndNil(t *testing.T) {
	p := &dbbackup.Plan{}
	maskPlanSecrets(p)
	if p.OSS.AccessKey != "" || p.OSS.SecretKey != "" || p.FTP.Password != "" {
		t.Fatalf("空凭据应保持空（未配置语义）: %+v", p)
	}
	maskPlanSecrets(nil) // 不应 panic
}

func TestMergePlanMaskedSecrets_FillsPlaceholderAndEmpty(t *testing.T) {
	current := &dbbackup.Plan{}
	current.OSS.AccessKey = "REAL-AK"
	current.OSS.SecretKey = "REAL-SK"
	current.FTP.Password = "REAL-PWD"
	svc := &fakeSvc{plan: current}

	req := &dbbackup.Plan{}
	req.OSS.AccessKey = dbbackup.PlanMaskPlaceholder // 用户未改
	req.OSS.SecretKey = ""                           // 前端约定空串=未改
	req.FTP.Password = dbbackup.PlanMaskPlaceholder

	h := &DBBackupHandler{service: svc}
	if err := h.mergePlanMaskedSecrets(newTestGinContext(), req); err != nil {
		t.Fatalf("回填失败: %v", err)
	}
	if req.OSS.AccessKey != "REAL-AK" || req.OSS.SecretKey != "REAL-SK" || req.FTP.Password != "REAL-PWD" {
		t.Fatalf("占位符/空串应回填原值: %+v", req)
	}
}

func TestMergePlanMaskedSecrets_ClearMarkerPassthrough(t *testing.T) {
	current := &dbbackup.Plan{}
	current.OSS.AccessKey = "REAL-AK"
	svc := &fakeSvc{plan: current}

	req := &dbbackup.Plan{}
	req.OSS.AccessKey = dbbackup.CredentialClearMarker // 显式清空：不回填，透传服务层
	req.OSS.SecretKey = "NEW-SK"                       // 真实值：needMerge 由其它字段触发

	h := &DBBackupHandler{service: svc}
	if err := h.mergePlanMaskedSecrets(newTestGinContext(), req); err != nil {
		t.Fatalf("透传失败: %v", err)
	}
	if req.OSS.AccessKey != dbbackup.CredentialClearMarker {
		t.Fatalf("__CLEAR__ 应原样透传给服务层，实得 %q", req.OSS.AccessKey)
	}
}

func TestMergePlanMaskedSecrets_AllRealValuesSkipsLookup(t *testing.T) {
	svc := &fakeSvc{}
	req := &dbbackup.Plan{}
	req.OSS.AccessKey = "AK"
	req.OSS.SecretKey = "SK"
	req.FTP.Password = "PWD"

	h := &DBBackupHandler{service: svc}
	if err := h.mergePlanMaskedSecrets(newTestGinContext(), req); err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if svc.getCalls != 0 {
		t.Fatalf("全真实值不应读取当前计划，实得 %d 次调用", svc.getCalls)
	}
}

func TestMergePlanMaskedSecrets_GetPlanErrorPropagates(t *testing.T) {
	svc := &fakeSvc{planErr: errors.New("db down")}
	req := &dbbackup.Plan{}
	req.OSS.AccessKey = dbbackup.PlanMaskPlaceholder

	h := &DBBackupHandler{service: svc}
	err := h.mergePlanMaskedSecrets(newTestGinContext(), req)
	if err == nil || !strings.Contains(err.Error(), "db down") {
		t.Fatalf("GetPlan 失败应向上传播: %v", err)
	}
}
