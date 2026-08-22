package requests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =====================================================================
// 74-11 escalation gap-closure: internal/api/v1/operations/requests —
// 分页默认值/上限 + 状态筛选。
// =====================================================================

func TestPaginationParams_GetPagination(t *testing.T) {
	// 零值 → 默认 1/10
	p := PaginationParams{}
	c, ps := p.GetPagination()
	assert.Equal(t, 1, c)
	assert.Equal(t, 10, ps)
	assert.Equal(t, 0, p.GetOffset())

	// 正常值
	p2 := PaginationParams{}
	p2.Current, p2.PageSize = 3, 20
	c, ps = p2.GetPagination()
	assert.Equal(t, 3, c)
	assert.Equal(t, 20, ps)
	assert.Equal(t, 40, p2.GetOffset())

	// 上限截断
	p3 := PaginationParams{}
	p3.Current, p3.PageSize = 1, 99999
	_, ps = p3.GetPagination()
	assert.Equal(t, 100, ps, "MaxListPageSize 上限")
}

func TestStatusRequest(t *testing.T) {
	s := StatusRequest{}
	assert.False(t, s.HasStatus())
	assert.Equal(t, 0, s.GetStatus(0))

	stopped := 1
	s2 := StatusRequest{Status: &stopped}
	assert.True(t, s2.HasStatus())
	assert.Equal(t, 1, s2.GetStatus(0))
	assert.Equal(t, 1, s2.GetStatus(9), "已设置时忽略默认值")
}
