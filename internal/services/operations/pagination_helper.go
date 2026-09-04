package operations

import (
	"math"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// PaginationParams 分页参数
type PaginationParams struct {
	Current  int
	PageSize int
}

// PageResult 分页结果 — base.PageResult 的 type alias。
//
// 全项目唯一 struct 定义在 base 包（D-03 单一定义）；alias 使 opsServices.PageResult
// 与 base.PageResult 成为同一类型，包内既有引用（含测试）零改动编译。
type PageResult = base.PageResult

// extractPagination 从参数中提取分页信息
func extractPagination(params map[string]interface{}) PaginationParams {
	current := extractIntParam(params, "current", constants.DefaultCurrent)
	pageSize := extractIntParam(params, "pageSize", constants.DefaultPageSize)
	pageSize = clampPageSize(pageSize)

	return PaginationParams{
		Current:  current,
		PageSize: pageSize,
	}
}

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

// clampPageSize 限制 pageSize 在有效范围内。
//
// operations 模块同时服务表格 list 与下拉全集,故使用 MaxOptionsPageSize
// (10000)作为上限,保持现有运行时行为;若未来拆分 options 端点,可改用
// MaxListPageSize。
func clampPageSize(pageSize int) int {
	return int(math.Max(float64(constants.MinPageSize), math.Min(float64(constants.MaxOptionsPageSize), float64(pageSize))))
}

// 偏移量计算函数已于 Phase 91-04 修剪：最后一个生产消费者（building/asset List）
// 已于 91-03 repo 化、workstation 于 91-02 repo 化，offset 计算由
// base.GORMRepository.List 内部承接；typed 路径统一走
// requests.PaginationParams.GetOffset。零消费者死代码。
