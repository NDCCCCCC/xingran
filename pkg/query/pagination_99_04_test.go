package query

// =====================================================================
// Phase 99-04 Task 1 (V130R-09 D-03-2): NormalizePaginationWithMax
// 3 参数归一化权威核心 — 表驱动回归测试。
//
// 口径要点(D-03-2 锁定实现):
//   - current <= 0 → DefaultCurrent(1),负值修正消除负 offset 隐患
//   - pageSize <= 0 → DefaultPageSize(10)
//   - pageSize > maxPageSize → maxPageSize(钳制,不静默重置默认)
//   - 不做 MinPageSize 下限放大:pageSize 1-9 原样透传
// =====================================================================

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
)

// TestNormalizePaginationWithMax:表驱动覆盖全部归一化分支 + 1-9 透传口径。
func TestNormalizePaginationWithMax(t *testing.T) {
	cases := []struct {
		name        string
		current     int
		pageSize    int
		maxPageSize int
		wantCurrent int
		wantSize    int
	}{
		{
			name:        "合法值原样透传",
			current:     3,
			pageSize:    50,
			maxPageSize: 200,
			wantCurrent: 3,
			wantSize:    50,
		},
		{
			name:        "current为0修正为默认1",
			current:     0,
			pageSize:    20,
			maxPageSize: 200,
			wantCurrent: 1,
			wantSize:    20,
		},
		{
			name:        "current为负修正为默认1_消除负offset隐患",
			current:     -5,
			pageSize:    20,
			maxPageSize: 200,
			wantCurrent: 1,
			wantSize:    20,
		},
		{
			name:        "pageSize为0修正为默认10",
			current:     1,
			pageSize:    0,
			maxPageSize: 200,
			wantCurrent: 1,
			wantSize:    10,
		},
		{
			name:        "pageSize为负修正为默认10",
			current:     1,
			pageSize:    -3,
			maxPageSize: 200,
			wantCurrent: 1,
			wantSize:    10,
		},
		{
			name:        "pageSize超上限钳到max_不静默重置默认",
			current:     1,
			pageSize:    500,
			maxPageSize: 100,
			wantCurrent: 1,
			wantSize:    100,
		},
		{
			name:        "pageSize恰好等于max_不钳制",
			current:     1,
			pageSize:    100,
			maxPageSize: 100,
			wantCurrent: 1,
			wantSize:    100,
		},
		{
			// 口径差异行(V130R-09):旧 clampPageSize 以 MinPageSize=10 放大下限
			// (5→10);新权威出口无下限放大,1-9 原样透传。
			name:        "pageSize1到9不放大下限_原样透传",
			current:     1,
			pageSize:    5,
			maxPageSize: 200,
			wantCurrent: 1,
			wantSize:    5,
		},
		{
			name:        "pageSize1透传",
			current:     2,
			pageSize:    1,
			maxPageSize: 200,
			wantCurrent: 2,
			wantSize:    1,
		},
		{
			name:        "ops全集口径_maxOptionsPageSize10000不钳正常值",
			current:     1,
			pageSize:    10000,
			maxPageSize: 10000,
			wantCurrent: 1,
			wantSize:    10000,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotCurrent, gotSize := NormalizePaginationWithMax(tc.current, tc.pageSize, tc.maxPageSize)
			assert.Equal(t, tc.wantCurrent, gotCurrent)
			assert.Equal(t, tc.wantSize, gotSize)
		})
	}
}

// TestNormalizePagination_Delegates:锁定 NormalizePagination 委托关系
// (cap = constants.MaxPageSize = 200,行为逐字节不变)。
func TestNormalizePagination_Delegates(t *testing.T) {
	// 超上限 → 钳到 MaxPageSize=200。
	gotCurrent, gotSize := NormalizePagination(1, 500)
	assert.Equal(t, 1, gotCurrent)
	assert.Equal(t, constants.MaxPageSize, gotSize)
	assert.Equal(t, 200, gotSize)

	// 全默认:0/0 → (1, 10)。
	gotCurrent, gotSize = NormalizePagination(0, 0)
	assert.Equal(t, constants.DefaultCurrent, gotCurrent)
	assert.Equal(t, constants.DefaultPageSize, gotSize)
}
