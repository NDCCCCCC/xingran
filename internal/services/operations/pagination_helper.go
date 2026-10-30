package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
)

// PageResult 分页结果 — base.PageResult 的 type alias。
//
// 全项目唯一 struct 定义在 base 包（D-03 单一定义）；alias 使 opsServices.PageResult
// 与 base.PageResult 成为同一类型，包内既有引用（含测试）零改动编译。
type PageResult = base.PageResult

// extractSortRequest 从 map 参数中提取排序请求,构造 base.BaseListRequest。
// operations 模块 handler 直接 bind map[string]interface{},前端传的
// orderByColumn/isAsc 会随 map 透传到这里。
func extractSortRequest(params map[string]interface{}) base.BaseListRequest {
	req := base.BaseListRequest{
		Current:       extractIntParam(params, "current", constants.DefaultCurrent),
		PageSize:      extractIntParam(params, "pageSize", constants.DefaultPageSize),
		OrderByColumn: extractStringParam(params, "orderByColumn"),
	}
	if isAsc, ok := params["isAsc"].(bool); ok {
		req.IsAsc = &isAsc
	}
	return req
}

// extractIntParam 提取整数参数
func extractIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	if value, ok := params[key].(float64); ok {
		return int(value)
	}
	return defaultValue
}

// extractStringParam 提取字符串参数
func extractStringParam(params map[string]interface{}, key string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return ""
}

// extractPagination / clampPageSize / PaginationParams 已于 Phase 99-04
// (V130R-09 D-03-3) 删除：三个生产调用方(location_alias/building/asset List)
// 全部迁移到 pkg/query.NormalizePaginationWithMax 单一权威出口，零调用方
// 死代码按用户标准直接删除（不留 @deprecated 存根）。分页归一化唯一权威：
// pkg/query。偏移量计算函数已于 Phase 91-04 修剪（offset 计算由
// base.GORMRepository.List 内部承接；typed 路径统一走
// requests.PaginationParams.GetOffset）。
