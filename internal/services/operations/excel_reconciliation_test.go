package operations

// Phase 44 R3 / Plan 44-02 Task 4 — reconciliationExceptionRule Excel 配置 + handler/router 静态扫描
//
// 测试覆盖:
//  1. ExcelConfig 条目存在 + 9 列顺序严格 + name UpsertKey=true+DBField="name"
//  2. 临时字段 scope_name 无 DBField (后处理 UPDATE scope_id)
//  3. ImportRules/ExportRules/DownloadTemplate handler 静态源码扫描
//  4. ImportFromExcel service 方法存在 + scope_id 解析逻辑存在
//
// 注:ResolveReconScopeID / ParseCSVToTextArray 是 asset 包的辅助函数,其测试在
// internal/services/asset/reconciliation_exception_test.go 的同 plan 任务中。

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ExcelConfig 静态断言 (3 项目记忆强约束)
// ============================================================================

// TestReconciliationExceptionExcelConfigExists config 条目存在 + 9 列
func TestReconciliationExceptionExcelConfigExists(t *testing.T) {
	cfg, ok := GetExcelConfig("reconciliationExceptionRule")
	require.True(t, ok, "ExcelConfigs 必须含 'reconciliationExceptionRule' 条目")
	assert.Equal(t, "对账例外规则", cfg.SheetName)
	assert.Equal(t, "sys_reconciliation_exception", cfg.TableName)
	assert.Equal(t, 9, len(cfg.Columns), "必须 9 列(name/ip_range/conflict_types/exception_actions/severity_override/scope_type/scope_name/expires_at/reason)")
}

// TestReconciliationExceptionExcelColumnOrder 列顺序严格 = name/ip_range/conflict_types/exception_actions/severity_override/scope_type/scope_name/expires_at/reason
//
// 项目记忆 xingran-excel-import-column-position-matching: validateAndParseRow 按 row[i] 位置匹配,
// Columns 顺序必须与 Excel 模板列序严格一致。
func TestReconciliationExceptionExcelColumnOrder(t *testing.T) {
	cfg, ok := GetExcelConfig("reconciliationExceptionRule")
	require.True(t, ok)
	expected := []string{
		"name", "ipRange", "conflictTypes", "exceptionActions",
		"severityOverride", "scopeType", "scopeName", "expiresAt", "reason",
	}
	actual := make([]string, len(cfg.Columns))
	for i := range cfg.Columns {
		actual[i] = cfg.Columns[i].Field
	}
	assert.Equal(t, expected, actual,
		"Columns 顺序必须严格 = name/ip_range/conflict_types/exception_actions/severity_override/scope_type/scope_name/expires_at/reason")
}

// TestReconciliationExceptionExcelUpsertKeyHasDBField name 列 UpsertKey=true + DBField="name"
//
// 项目记忆 xingran-excel-import-upsertkey-needs-dbfield: UpsertKey 列漏 DBField → 冲突键失效。
func TestReconciliationExceptionExcelUpsertKeyHasDBField(t *testing.T) {
	cfg, ok := GetExcelConfig("reconciliationExceptionRule")
	require.True(t, ok)
	var nameCol *ExcelColumn
	for i := range cfg.Columns {
		if cfg.Columns[i].Field == "name" {
			nameCol = &cfg.Columns[i]
			break
		}
	}
	require.NotNil(t, nameCol, "必须含 name 列")
	assert.True(t, nameCol.UpsertKey, "name 列必须 UpsertKey=true (按规则名判重)")
	assert.Equal(t, "name", nameCol.DBField, "name 列必须配 DBField='name' (UpsertKey 需 DBField)")
}

// TestReconciliationExceptionExcelScopeNameNoDBField scopeName 是临时字段无 DBField
//
// 方案 B (WARN-7 锁定): scope_name 在 ImportFromExcel 后处理阶段按 scope_type 解析为 scope_id。
func TestReconciliationExceptionExcelScopeNameNoDBField(t *testing.T) {
	cfg, ok := GetExcelConfig("reconciliationExceptionRule")
	require.True(t, ok)
	var scopeNameCol *ExcelColumn
	for i := range cfg.Columns {
		if cfg.Columns[i].Field == "scopeName" {
			scopeNameCol = &cfg.Columns[i]
			break
		}
	}
	require.NotNil(t, scopeNameCol, "必须含 scopeName 列")
	assert.Equal(t, "", scopeNameCol.DBField, "scopeName 无 DBField (临时字段, 后处理 UPDATE scope_id)")
}

// ============================================================================
// ImportRules / ExportRules / DownloadTemplate handler 静态源码扫描
//
// 注:这些 handler/router 在 internal/api/v1/asset/ 包,不在 operations 包。
// 用相对路径读取(从 operations 包目录计算 ../../api/v1/asset/...)。
// ============================================================================

// TestImportRulesHandlerExists 静态断言:ImportRules handler 存在
func TestImportRulesHandlerExists(t *testing.T) {
	src := mustReadAssetSrc(t, "../../api/v1/asset/reconciliation_exception_handler.go")
	assert.Contains(t, src, "func (h *ReconciliationExceptionHandler) ImportRules(",
		"ImportRules handler 必须存在")
}

// TestImportRulesHandlerOperlog 静态断言:ImportRules 调 operlog.Record(OperTypeImport)
func TestImportRulesHandlerOperlog(t *testing.T) {
	src := mustReadAssetSrc(t, "../../api/v1/asset/reconciliation_exception_handler.go")
	assert.Contains(t, src,
		"operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModuleReconciliationExceptionRule, operlog.OperTypeImport)",
		"ImportRules 必须调 operlog.Record(OperTypeImport)")
}

// TestExportRulesAndDownloadTemplateHandlersExist 静态断言:ExportRules + DownloadTemplate handler 存在
func TestExportRulesAndDownloadTemplateHandlersExist(t *testing.T) {
	src := mustReadAssetSrc(t, "../../api/v1/asset/reconciliation_exception_handler.go")
	assert.Contains(t, src, "func (h *ReconciliationExceptionHandler) ExportRules(",
		"ExportRules handler 必须存在")
	assert.Contains(t, src, "func (h *ReconciliationExceptionHandler) DownloadTemplate(",
		"DownloadTemplate handler 必须存在")
}

// TestImportExportRoutesExist 静态断言:router 含 3 个 Excel 路由
//
// 项目记忆 xingran-excel-import-route-conflict: 不在 router.go 预注册 /asset/reconciliation/import,
// 由 SetupReconciliationExceptionRouter 自管专用路由。
func TestImportExportRoutesExist(t *testing.T) {
	routerSrc := mustReadAssetSrc(t, "../../api/v1/asset/reconciliation_exception_router.go")
	assert.Contains(t, routerSrc, `r.POST("/exception-rule/import"`,
		"router 必须含 /exception-rule/import 路由")
	assert.Contains(t, routerSrc, `r.POST("/exception-rule/export"`,
		"router 必须含 /exception-rule/export 路由")
	assert.Contains(t, routerSrc, `r.POST("/exception-rule/template"`,
		"router 必须含 /exception-rule/template 路由")
}

// TestImportFromExcelServiceMethodExists 静态断言:service 含 ImportFromExcel 方法
//
// 方案 B (WARN-7 锁定): ImportFromExcel 调 excel_service.ImportData 后处理
// scope_name→scope_id + TEXT[] 转换。
func TestImportFromExcelServiceMethodExists(t *testing.T) {
	src := mustReadAssetSrc(t, "../asset/reconciliation_exception.go")
	assert.Contains(t, src, "func (s *reconciliationExceptionServiceImpl) ImportFromExcel(",
		"reconciliationExceptionServiceImpl 必须含 ImportFromExcel 方法 (方案 B)")
	// 后处理逻辑必须含 scope_id 解析代码
	assert.True(t,
		strings.Contains(src, "ResolveReconScopeID") ||
			strings.Contains(src, "UPDATE sys_reconciliation_exception SET scope_id"),
		"ImportFromExcel 实现必须含 scope_name→scope_id 解析 (grep ResolveReconScopeID 或 UPDATE scope_id)")
}

// helpers

func mustReadAssetSrc(t *testing.T, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(relPath)
	require.NoError(t, err, "must read %s", relPath)
	return string(data)
}
