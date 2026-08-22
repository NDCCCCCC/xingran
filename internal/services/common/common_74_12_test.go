package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =====================================================================
// 74-11 escalation gap-closure: internal/services/common(0% → 覆盖
// DefaultListParams + ListParams 嵌入字段)。
// =====================================================================

func TestDefaultListParams(t *testing.T) {
	p := DefaultListParams()
	assert.Equal(t, 1, p.Current)
	assert.Equal(t, 10, p.PageSize)
	assert.Equal(t, "", p.OrderByColumn)
	assert.Nil(t, p.IsAsc)

	// 直接构造覆盖
	custom := ListParams{}
	custom.Current = 2
	custom.PageSize = 50
	custom.OrderByColumn = "createdAt"
	asc := true
	custom.IsAsc = &asc
	assert.Equal(t, 2, custom.Current)
	assert.Equal(t, "createdAt", custom.OrderByColumn)
	assert.True(t, *custom.IsAsc)
}

func TestPageResultShape(t *testing.T) {
	r := PageResult{List: []int{1}, Total: 1, Current: 1, PageSize: 10}
	assert.Equal(t, []int{1}, r.List)
	assert.Equal(t, int64(1), r.Total)
}
