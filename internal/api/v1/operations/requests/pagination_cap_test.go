package requests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
)

// v129-recheck C-4 — workstation List 的平面图/3D 消费方以 pageSize:1000
// 请求楼层全集，迁移时误用 MaxListPageSize=100 上限导致 >100 工位的楼层
// 静默截断。锁定：全集口径走 MaxOptionsPageSize=10000，普通列表口径
// MaxListPageSize=100 不变，且 current<1 守卫两条路径都在。
func TestGetPaginationWithMaxRestoresFullSetCap(t *testing.T) {
	tests := []struct {
		name              string
		current, pageSize int
		wantCur, wantSize int
	}{
		{"平面图全集请求 1000 应放行", 1, 1000, 1, 1000},
		{"3D 视图大页 5000 仍在 10000 内", 1, 5000, 1, 5000},
		{"超过 10000 钳到上限", 1, 20000, 1, constants.MaxOptionsPageSize},
		{"current<1 回退默认（守卫保留）", 0, 1000, constants.DefaultCurrent, 1000},
		{"pageSize<10 回退默认", 1, 0, 1, constants.DefaultPageSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &PaginationParams{}
			req.Current = tt.current
			req.PageSize = tt.pageSize

			gotCur, gotSize := req.GetPaginationWithMax(constants.MaxOptionsPageSize)
			assert.Equal(t, tt.wantCur, gotCur)
			assert.Equal(t, tt.wantSize, gotSize, "GetPaginationWithMax(MaxOptionsPageSize)")
			assert.GreaterOrEqual(t, gotCur, constants.DefaultCurrent)
		})
	}

	// GetPagination 默认口径（普通列表）必须仍钳 MaxListPageSize=100
	req := &PaginationParams{BaseListRequest: base.BaseListRequest{Current: 1, PageSize: 1000}}
	defCur, defSize := req.GetPagination()
	assert.Equal(t, 1, defCur)
	assert.Equal(t, constants.MaxListPageSize, defSize, "GetPagination 默认口径必须仍钳 100")
}
