package operations

import "testing"

// TestNormalizeHeader 验证表头归一化: 去括号英文标识 + "代码"→"编码" + trim/lower。
// 这是按表头匹配能跨格式工作的基础。
func TestNormalizeHeader(t *testing.T) {
	cases := []struct{ in, want string }{
		{"科室编码(SECTION_OFFICE_CODE)", "科室编码"},
		{"科室代码", "科室编码"}, // 业务侧"代码" ↔ config"编码"
		{"部门组名称(DEPARTMENT_GROUP_NAME)", "部门组名称"},
		{"部门组代码", "部门组编码"},
		{"  部门名称  ", "部门名称"},
		{"科室名称(SECTION_OFFICE_NAME)", "科室名称"},
	}
	for _, c := range cases {
		if got := normalizeHeader(c.in); got != c.want {
			t.Errorf("normalizeHeader(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestResolveColumnsByHeader_UserColumnOrder 验证用户的业务 Excel 列序
// (部门组名称→部门组代码→部门名称→部门代码→科室名称→科室代码, 与模板完全不同)
// 能被正确匹配到对应的 config 字段。这是本次 bug 修复的核心场景。
func TestResolveColumnsByHeader_UserColumnOrder(t *testing.T) {
	config, ok := GetExcelConfig("department")
	if !ok {
		t.Fatal("department config not found")
	}

	userHeader := []string{"部门组名称", "部门组代码", "部门名称", "部门代码", "科室名称", "科室代码"}

	effective, matchedRequired := resolveColumnsByHeader(userHeader, config)

	requiredTotal := 0
	for _, col := range config.Columns {
		if col.Required {
			requiredTotal++
		}
	}
	if matchedRequired != requiredTotal {
		t.Fatalf("matchedRequired = %d, want %d (all required matched)", matchedRequired, requiredTotal)
	}

	// 按用户列序, 第 i 列应映射到对应字段(而非 config 的固定顺序)
	want := []string{
		"departmentGroupName", // 部门组名称
		"departmentGroupCode", // 部门组代码
		"departmentName",      // 部门名称
		"departmentCode",      // 部门代码
		"deptName",            // 科室名称
		"deptCode",            // 科室代码
	}
	for i, w := range want {
		if effective[i].Field != w {
			t.Errorf("effective[%d].Field = %q, want %q", i, effective[i].Field, w)
		}
	}
}

// TestResolveColumnsByHeader_TemplateOrder 验证用下载模板导入(列序=config.Header 顺序)时,
// effectiveColumns 与 config.Columns 顺序完全一致 — 保证 building/workstation 等已验证模块行为不变。
func TestResolveColumnsByHeader_TemplateOrder(t *testing.T) {
	config, ok := GetExcelConfig("department")
	if !ok {
		t.Fatal("department config not found")
	}

	templateHeader := make([]string, len(config.Columns))
	for i, col := range config.Columns {
		templateHeader[i] = col.Header
	}

	effective, matchedRequired := resolveColumnsByHeader(templateHeader, config)

	requiredTotal := 0
	for _, col := range config.Columns {
		if col.Required {
			requiredTotal++
		}
	}
	if matchedRequired != requiredTotal {
		t.Fatalf("matchedRequired = %d, want %d", matchedRequired, requiredTotal)
	}

	for i, col := range config.Columns {
		if effective[i].Field != col.Field {
			t.Errorf("effective[%d].Field = %q, want %q (template order must be preserved)", i, effective[i].Field, col.Field)
		}
	}
}
