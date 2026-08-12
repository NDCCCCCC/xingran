package portcollection

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestInterfaceNameSortExpr 锁定接口名排序 ORDER BY 表达式按 dialect 分支正确生成。
//
// 核心守护:
//   - postgres: 必须含 速率前缀(regexp_replace) + 数字段 int 数组(regexp_matches/array_agg),
//     即数值排序而非字符串排序;ASC/DESC 方向正确
//   - 其他 dialect(sqlite 等): 降级为 interface_name 字符串排序,方向正确
//   - 防止误回归成普通 "ORDER BY interface_name"(那样 GE0/10 < GE0/2 字典序错误)
//
// SQL 数值排序语义的正确性由 TestInterfaceNameSortSemantics 用纯 Go 模拟验证,
// 不依赖真实 PostgreSQL。
func TestInterfaceNameSortExpr(t *testing.T) {
	tests := []struct {
		name           string
		dialect        string
		direction      string
		mustContain    []string
		mustNotContain []string
	}{
		{
			name:           "postgres ASC 数值排序(前缀+数字段数组)",
			dialect:        "postgres",
			direction:      "ASC",
			mustContain:    []string{"regexp_replace", "regexp_matches", "array_agg", "::int", "ASC"},
			mustNotContain: []string{"DESC"},
		},
		{
			name:           "postgres DESC 数值排序",
			dialect:        "postgres",
			direction:      "DESC",
			mustContain:    []string{"regexp_replace", "regexp_matches", "array_agg", "::int", "DESC"},
			mustNotContain: []string{"ASC"},
		},
		{
			name:           "sqlite ASC 降级字符串排序",
			dialect:        "sqlite",
			direction:      "ASC",
			mustContain:    []string{"interface_name", "ASC"},
			mustNotContain: []string{"regexp_matches", "array_agg"},
		},
		{
			name:           "sqlite DESC 降级字符串排序",
			dialect:        "sqlite",
			direction:      "DESC",
			mustContain:    []string{"interface_name", "DESC"},
			mustNotContain: []string{"regexp_matches", "array_agg"},
		},
		{
			name:           "未知 dialect(mysql) 降级字符串排序",
			dialect:        "mysql",
			direction:      "ASC",
			mustContain:    []string{"interface_name", "ASC"},
			mustNotContain: []string{"regexp_matches"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := interfaceNameSortExpr(tt.dialect, tt.direction)
			for _, s := range tt.mustContain {
				if !strings.Contains(got, s) {
					t.Errorf("dialect=%s dir=%s: 表达式\n  %s\n必须包含 %q,实际缺失", tt.dialect, tt.direction, got, s)
				}
			}
			for _, s := range tt.mustNotContain {
				if strings.Contains(got, s) {
					t.Errorf("dialect=%s dir=%s: 表达式\n  %s\n不得包含 %q,实际存在", tt.dialect, tt.direction, got, s)
				}
			}
		})
	}
}

// digitRe 与 prefixRe 镜像 interfaceNameSortExpr 中 PG 正则的语义,供纯 Go 模拟排序用。
var (
	digitRe  = regexp.MustCompile(`[0-9]+`)
	prefixRe = regexp.MustCompile(`^[^0-9]+`)
)

// extractDigitGroups 模拟 PG regexp_matches(name,'([0-9]+)','g')::int[]:
// 从接口名提取所有连续数字段为 int 切片。GE0/1 → {0,1},GE0/0/1 → {0,0,1}。
func extractDigitGroups(name string) []int {
	var nums []int
	for _, m := range digitRe.FindAllString(name, -1) {
		n, _ := strconv.Atoi(m)
		nums = append(nums, n)
	}
	return nums
}

// interfacePrefix 模拟 PG regexp_replace(name,'[0-9].*$',''):
// 取接口名首个数字之前的字母前缀。GE0/1 → GE,XGE1/0/24 → XGE。
func interfacePrefix(name string) string {
	return prefixRe.FindString(name)
}

// TestInterfaceNameSortSemantics 用纯 Go 模拟 interfaceNameSortExpr 的 PG 排序语义
// (速率前缀 → 数字段数组 → 接口名),证明排序规则对用户需求示例正确。
//
// 这不连真实 PG,而是验证"按这三键排序"这一规则本身与业务期望一致。
// 生产 SQL 与本测试用同一正则语义(digitRe/prefixRe 镜像 PG regexp)。
func TestInterfaceNameSortSemantics(t *testing.T) {
	// 用户需求示例:乱序输入,期望按 板卡号→接口号 数值升序
	input := []string{
		"GE0/10", "GE1/1", "GE0/2", "GE1/10", "GE0/1", "GE1/2", "GE0/11", "GE0/0/1",
	}
	expectedAsc := []string{
		"GE0/0/1", "GE0/1", "GE0/2", "GE0/10", "GE0/11", "GE1/1", "GE1/2", "GE1/10",
	}

	// ASC: 板卡号 → 接口号 数值升序
	asc := make([]string, len(input))
	copy(asc, input)
	sort.Sort(byInterfaceAsc(asc))
	if !equalStrings(asc, expectedAsc) {
		t.Errorf("ASC 排序错误:\n  got:      %v\n  expected: %v", asc, expectedAsc)
	}

	// DESC: ASC 的严格逆序
	descExpected := reverseStrings(expectedAsc)
	desc := make([]string, len(input))
	copy(desc, input)
	sort.Sort(byInterfaceDesc(desc))
	if !equalStrings(desc, descExpected) {
		t.Errorf("DESC 排序错误:\n  got:      %v\n  expected: %v", desc, descExpected)
	}
}

// TestInterfaceNameSortSemantics_MixedRate 验证不同速率前缀分组正确:
// FE/GE/XGE 各自成组,组内按板卡/接口排序,不会交替混排。
func TestInterfaceNameSortSemantics_MixedRate(t *testing.T) {
	input := []string{"GE0/2", "FE0/2", "XGE0/1", "GE0/1", "FE0/1", "XGE0/10"}
	// 期望: FE 组 → GE 组 → XGE 组(字母字典序),组内按板卡/接口
	expected := []string{"FE0/1", "FE0/2", "GE0/1", "GE0/2", "XGE0/1", "XGE0/10"}

	sort.Sort(byInterfaceAsc(input))
	if !equalStrings(input, expected) {
		t.Errorf("混合速率排序错误:\n  got:      %v\n  expected: %v", input, expected)
	}
}

// TestInterfaceNameSortSemantics_NoDigits 验证无数字接口名(Vlan/NULL)不崩溃,
// 排到末尾(空数组/空前缀的 COALESCE 兜底语义)。
func TestInterfaceNameSortSemantics_NoDigits(t *testing.T) {
	input := []string{"GE0/2", "Vlan100", "NULL", "GE0/1", "Vlan10"}
	// 有数字的正常按前缀+数字排;Vlan10/Vlan100 按 Vlan 前缀 + {10}/{100} 排;
	// NULL 无数字,数字组为 nil,排末尾
	expected := []string{"GE0/1", "GE0/2", "NULL", "Vlan10", "Vlan100"}

	sort.Sort(byInterfaceAsc(input))
	if !equalStrings(input, expected) {
		t.Errorf("含无数字接口名排序错误:\n  got:      %v\n  expected: %v", input, expected)
	}
}

// --- 排序辅助类型(镜像生产 SQL 的三键语义) ---

// interfaceLess 镜像 PG: 前缀 → 数字段数组 → 接口名(全 ASC)。
func interfaceLess(a, b string) bool {
	pa, pb := interfacePrefix(a), interfacePrefix(b)
	if pa != pb {
		return pa < pb
	}
	na, nb := extractDigitGroups(a), extractDigitGroups(b)
	if cmp := compareIntSlices(na, nb); cmp != 0 {
		return cmp < 0
	}
	return a < b
}

type byInterfaceAsc []string

func (s byInterfaceAsc) Len() int           { return len(s) }
func (s byInterfaceAsc) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byInterfaceAsc) Less(i, j int) bool { return interfaceLess(s[i], s[j]) }

type byInterfaceDesc []string

func (s byInterfaceDesc) Len() int           { return len(s) }
func (s byInterfaceDesc) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byInterfaceDesc) Less(i, j int) bool { return interfaceLess(s[j], s[i]) }

// compareIntSlices 逐元素数值比较(镜像 PG int[] 数组比较)。nil 视为空切片。
func compareIntSlices(a, b []int) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[i] - b[i]
		}
	}
	return len(a) - len(b)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func reverseStrings(s []string) []string {
	out := make([]string, len(s))
	for i := range s {
		out[len(s)-1-i] = s[i]
	}
	return out
}
