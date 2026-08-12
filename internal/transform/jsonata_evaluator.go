package transform

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jsonata-go/jsonata"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// JSONataEvaluator JSONata 表达式求值器
type JSONataEvaluator struct {
	maxTime  int // 最大执行时间（毫秒），防止无限循环
	maxDepth int // 最大递归深度，防止栈溢出
}

// NewJSONataEvaluator 创建 JSONata 求值器
func NewJSONataEvaluator() *JSONataEvaluator {
	return &JSONataEvaluator{
		maxTime:  1000, // 1 秒超时，per D-02 安全限制
		maxDepth: 100,  // 最大递归深度 100
	}
}

// Evaluate 执行表达式求值
func (e *JSONataEvaluator) Evaluate(expression string, input interface{}) (interface{}, error) {
	if expression == "" {
		return input, nil // 空表达式返回原始数据
	}

	// 1. 打开 JSONata 实例
	instance, err := jsonata.OpenLatest()
	if err != nil {
		return nil, fmt.Errorf("failed to open jsonata: %w", err)
	}

	// 2. 编译表达式（recoveryMode=false 表示严格模式）
	expr, err := instance.Compile(expression, false)
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression '%s': %w", expression, err)
	}

	// 3. 设置安全限制（防止表达式注入攻击）
	expr.SetMaxTime(e.maxTime)
	expr.SetMaxDepth(e.maxDepth)

	// 4. 序列化输入
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// 5. 执行求值
	resultJSON, err := expr.Evaluate(inputJSON, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", expression, err)
	}

	// 6. 反序列化结果
	var result interface{}
	if len(resultJSON) > 0 {
		if err := json.Unmarshal(resultJSON, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return result, nil
}

// EvaluateWithTimeout 执行表达式求值（带超时）
func (e *JSONataEvaluator) EvaluateWithTimeout(expression string, input interface{}, timeout time.Duration) (interface{}, error) {
	// 创建带超时的求值器
	evaluator := &JSONataEvaluator{
		maxTime:  int(timeout.Milliseconds()),
		maxDepth: e.maxDepth,
	}
	return evaluator.Evaluate(expression, input)
}

// ApplyAggregate 应用聚合函数
func (e *JSONataEvaluator) ApplyAggregate(data interface{}, aggregateFunc string) (interface{}, error) {
	// 将数据转换为数组（如果不是数组）
	arr, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("aggregate function requires array input, got %T", data)
	}

	switch aggregateFunc {
	case "sum":
		return e.aggregateSum(arr), nil
	case "avg":
		return e.aggregateAvg(arr), nil
	case "count":
		return len(arr), nil
	case "min":
		return e.aggregateMin(arr), nil
	case "max":
		return e.aggregateMax(arr), nil
	default:
		return nil, fmt.Errorf("unsupported aggregate function: %s", aggregateFunc)
	}
}

func (e *JSONataEvaluator) aggregateSum(arr []interface{}) float64 {
	var sum float64
	for _, item := range arr {
		if num, ok := item.(float64); ok {
			sum += num
		}
	}
	return sum
}

func (e *JSONataEvaluator) aggregateAvg(arr []interface{}) float64 {
	if len(arr) == 0 {
		return 0
	}
	return e.aggregateSum(arr) / float64(len(arr))
}

func (e *JSONataEvaluator) aggregateMin(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	min := arr[0]
	for _, item := range arr[1:] {
		if less(item, min) {
			min = item
		}
	}
	return min
}

func (e *JSONataEvaluator) aggregateMax(arr []interface{}) interface{} {
	if len(arr) == 0 {
		return nil
	}
	max := arr[0]
	for _, item := range arr[1:] {
		if less(max, item) {
			max = item
		}
	}
	return max
}

// less 比较两个值的大小
func less(a, b interface{}) bool {
	switch a.(type) {
	case float64:
		if bFloat, ok := b.(float64); ok {
			return a.(float64) < bFloat
		}
	case int:
		if bInt, ok := b.(int); ok {
			return a.(int) < bInt
		}
	case string:
		if bString, ok := b.(string); ok {
			return a.(string) < bString
		}
	}
	return false
}

// ApplyOrderBy 应用排序
func (e *JSONataEvaluator) ApplyOrderBy(data interface{}, orderBy *models.OrderByConfig) (interface{}, error) {
	if orderBy == nil {
		return data, nil
	}

	// 将数据转换为数组
	arr, ok := data.([]interface{})
	if !ok {
		return data, nil // 非数组不排序
	}

	// 创建可排序的包装
	sortable := &sortableArray{
		data:    arr,
		field:   orderBy.Field,
		reverse: orderBy.Direction == "desc",
	}
	sort.Sort(sortable)

	return sortable.data, nil
}

// ApplyLimit 应用限制
func (e *JSONataEvaluator) ApplyLimit(data interface{}, limit int) interface{} {
	if limit <= 0 {
		return data
	}

	arr, ok := data.([]interface{})
	if !ok {
		return data
	}

	if limit > len(arr) {
		limit = len(arr)
	}
	return arr[:limit]
}

// sortableArray 可排序数组包装
type sortableArray struct {
	data    []interface{}
	field   string
	reverse bool
}

func (s *sortableArray) Len() int {
	return len(s.data)
}

func (s *sortableArray) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}

func (s *sortableArray) Less(i, j int) bool {
	valI := getFieldValue(s.data[i], s.field)
	valJ := getFieldValue(s.data[j], s.field)

	result := less(valI, valJ)
	if s.reverse {
		return !result
	}
	return result
}

// getFieldValue 从对象中获取字段值
func getFieldValue(obj interface{}, field string) interface{} {
	m, ok := obj.(map[string]interface{})
	if !ok {
		return nil
	}
	return m[field]
}

// Transform 执行完整的数据转换管道
func (e *JSONataEvaluator) Transform(data interface{}, config *models.DataTransformConfig) (interface{}, error) {
	if config == nil {
		return data, nil
	}

	var err error
	result := data

	// 1. 应用表达式求值
	if config.Expression != "" {
		result, err = e.Evaluate(config.Expression, result)
		if err != nil {
			return nil, fmt.Errorf("expression evaluation failed: %w", err)
		}
	}

	// 2. 应用聚合函数
	if config.Aggregate != nil && *config.Aggregate != "" {
		result, err = e.ApplyAggregate(result, *config.Aggregate)
		if err != nil {
			return nil, fmt.Errorf("aggregate failed: %w", err)
		}
	}

	// 3. 应用排序
	if config.OrderBy != nil {
		result, err = e.ApplyOrderBy(result, config.OrderBy)
		if err != nil {
			return nil, fmt.Errorf("orderBy failed: %w", err)
		}
	}

	// 4. 应用限制
	if config.Limit > 0 {
		result = e.ApplyLimit(result, config.Limit)
	}

	return result, nil
}
