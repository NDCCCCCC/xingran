package transform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestJSONataEvaluator_Evaluate 测试表达式求值
func TestJSONataEvaluator_Evaluate(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	// 测试简单表达式
	input := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"value": 100.0},
			map[string]interface{}{"value": 200.0},
		},
	}

	// 表达式：获取第一个元素的 value
	result, err := evaluator.Evaluate("items[0].value", input)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, result)

	// 表达式：获取所有 value
	result, err = evaluator.Evaluate("items.value", input)
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{100.0, 200.0}, result)
}

// TestJSONataEvaluator_EmptyExpression 测试空表达式
func TestJSONataEvaluator_EmptyExpression(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	input := map[string]interface{}{"value": 100.0}

	// 空表达式返回原始数据
	result, err := evaluator.Evaluate("", input)
	assert.NoError(t, err)
	assert.Equal(t, input, result)
}

// TestJSONataEvaluator_ComplexExpression 测试复杂表达式
func TestJSONataEvaluator_ComplexExpression(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	input := map[string]interface{}{
		"data": map[string]interface{}{
			"nested": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"name": "Alice", "score": 95.0},
					map[string]interface{}{"name": "Bob", "score": 87.0},
				},
			},
		},
	}

	// 嵌套路径访问
	result, err := evaluator.Evaluate("data.nested.items[0].name", input)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", result)

	// 数组过滤 - JSONata 返回单个值或数组取决于结果
	result, err = evaluator.Evaluate("data.nested.items[score > 90].name", input)
	assert.NoError(t, err)
	// JSONata 可能返回单个值或数组
	if arr, ok := result.([]interface{}); ok {
		assert.Equal(t, []interface{}{"Alice"}, arr)
	} else {
		assert.Equal(t, "Alice", result)
	}
}

// TestJSONataEvaluator_Aggregate 测试聚合函数
func TestJSONataEvaluator_Aggregate(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	arr := []interface{}{1.0, 2.0, 3.0, 4.0, 5.0}

	// sum
	result, err := evaluator.ApplyAggregate(arr, "sum")
	assert.NoError(t, err)
	assert.Equal(t, 15.0, result)

	// avg
	result, err = evaluator.ApplyAggregate(arr, "avg")
	assert.NoError(t, err)
	assert.Equal(t, 3.0, result)

	// count
	result, err = evaluator.ApplyAggregate(arr, "count")
	assert.NoError(t, err)
	assert.Equal(t, 5, result)

	// min
	result, err = evaluator.ApplyAggregate(arr, "min")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, result)

	// max
	result, err = evaluator.ApplyAggregate(arr, "max")
	assert.NoError(t, err)
	assert.Equal(t, 5.0, result)
}

// TestJSONataEvaluator_Aggregate_EmptyArray 测试空数组聚合
func TestJSONataEvaluator_Aggregate_EmptyArray(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	arr := []interface{}{}

	// sum of empty array
	result, err := evaluator.ApplyAggregate(arr, "sum")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, result)

	// avg of empty array
	result, err = evaluator.ApplyAggregate(arr, "avg")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, result)

	// count of empty array
	result, err = evaluator.ApplyAggregate(arr, "count")
	assert.NoError(t, err)
	assert.Equal(t, 0, result)

	// min of empty array
	result, err = evaluator.ApplyAggregate(arr, "min")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// max of empty array
	result, err = evaluator.ApplyAggregate(arr, "max")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// TestJSONataEvaluator_Aggregate_InvalidType 测试非数组聚合
func TestJSONataEvaluator_Aggregate_InvalidType(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	// 非数组输入
	result, err := evaluator.ApplyAggregate("not an array", "sum")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestJSONataEvaluator_Aggregate_Unsupported 测试不支持的聚合函数
func TestJSONataEvaluator_Aggregate_Unsupported(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	arr := []interface{}{1.0, 2.0, 3.0}

	result, err := evaluator.ApplyAggregate(arr, "unsupported")
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestJSONataEvaluator_OrderBy 测试排序
func TestJSONataEvaluator_OrderBy(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	arr := []interface{}{
		map[string]interface{}{"name": "Charlie", "age": 30.0},
		map[string]interface{}{"name": "Alice", "age": 25.0},
		map[string]interface{}{"name": "Bob", "age": 35.0},
	}

	// 升序排序
	result, err := evaluator.ApplyOrderBy(arr, &models.OrderByConfig{Field: "age", Direction: "asc"})
	assert.NoError(t, err)
	resultArr := result.([]interface{})
	assert.Equal(t, "Alice", resultArr[0].(map[string]interface{})["name"])
	assert.Equal(t, "Charlie", resultArr[1].(map[string]interface{})["name"])
	assert.Equal(t, "Bob", resultArr[2].(map[string]interface{})["name"])

	// 降序排序
	result, err = evaluator.ApplyOrderBy(arr, &models.OrderByConfig{Field: "age", Direction: "desc"})
	assert.NoError(t, err)
	resultArr = result.([]interface{})
	assert.Equal(t, "Bob", resultArr[0].(map[string]interface{})["name"])
	assert.Equal(t, "Charlie", resultArr[1].(map[string]interface{})["name"])
	assert.Equal(t, "Alice", resultArr[2].(map[string]interface{})["name"])
}

// TestJSONataEvaluator_OrderBy_NilConfig 测试空排序配置
func TestJSONataEvaluator_OrderBy_NilConfig(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	arr := []interface{}{1.0, 2.0, 3.0}

	result, err := evaluator.ApplyOrderBy(arr, nil)
	assert.NoError(t, err)
	assert.Equal(t, arr, result)
}

// TestJSONataEvaluator_OrderBy_NonArray 测试非数组排序
func TestJSONataEvaluator_OrderBy_NonArray(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	result, err := evaluator.ApplyOrderBy("not an array", &models.OrderByConfig{Field: "age", Direction: "asc"})
	assert.NoError(t, err)
	assert.Equal(t, "not an array", result)
}

// TestJSONataEvaluator_Limit 测试限制
func TestJSONataEvaluator_Limit(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	arr := []interface{}{1.0, 2.0, 3.0, 4.0, 5.0}

	// 限制 3 个
	result := evaluator.ApplyLimit(arr, 3)
	assert.Equal(t, []interface{}{1.0, 2.0, 3.0}, result)

	// 限制超过数组长度
	result = evaluator.ApplyLimit(arr, 10)
	assert.Equal(t, 5, len(result.([]interface{})))

	// 限制 0 或负数
	result = evaluator.ApplyLimit(arr, 0)
	assert.Equal(t, arr, result)

	result = evaluator.ApplyLimit(arr, -1)
	assert.Equal(t, arr, result)
}

// TestJSONataEvaluator_Limit_NonArray 测试非数组限制
func TestJSONataEvaluator_Limit_NonArray(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	result := evaluator.ApplyLimit("not an array", 3)
	assert.Equal(t, "not an array", result)
}

// TestJSONataEvaluator_Transform 测试完整转换管道
func TestJSONataEvaluator_Transform(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	input := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"value": 100.0},
			map[string]interface{}{"value": 200.0},
			map[string]interface{}{"value": 300.0},
		},
	}

	aggregate := "sum"
	config := &models.DataTransformConfig{
		Expression: "items.value", // 提取所有 value
		Aggregate:  &aggregate,    // 求和
	}

	result, err := evaluator.Transform(input, config)
	assert.NoError(t, err)
	assert.Equal(t, 600.0, result)
}

// TestJSONataEvaluator_Transform_NilConfig 测试空配置转换
func TestJSONataEvaluator_Transform_NilConfig(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	input := map[string]interface{}{"value": 100.0}

	result, err := evaluator.Transform(input, nil)
	assert.NoError(t, err)
	assert.Equal(t, input, result)
}

// TestJSONataEvaluator_Transform_OnlyExpression 测试仅表达式转换
func TestJSONataEvaluator_Transform_OnlyExpression(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	input := map[string]interface{}{
		"data": map[string]interface{}{"value": 100.0},
	}

	config := &models.DataTransformConfig{
		Expression: "data.value",
	}

	result, err := evaluator.Transform(input, config)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, result)
}

// TestJSONataEvaluator_Transform_FullPipeline 测试完整管道
func TestJSONataEvaluator_Transform_FullPipeline(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	input := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "Alice", "score": 95.0},
			map[string]interface{}{"name": "Bob", "score": 87.0},
			map[string]interface{}{"name": "Charlie", "score": 92.0},
			map[string]interface{}{"name": "David", "score": 78.0},
		},
	}

	// 测试排序 + 限制
	config := &models.DataTransformConfig{
		Expression: "items", // 获取 items 数组
		OrderBy:    &models.OrderByConfig{Field: "score", Direction: "desc"},
		Limit:      3,
	}

	result, err := evaluator.Transform(input, config)
	assert.NoError(t, err)
	resultArr := result.([]interface{})
	assert.Equal(t, 3, len(resultArr))
	// 验证排序正确（降序）
	assert.Equal(t, "Alice", resultArr[0].(map[string]interface{})["name"])
	assert.Equal(t, "Charlie", resultArr[1].(map[string]interface{})["name"])
	assert.Equal(t, "Bob", resultArr[2].(map[string]interface{})["name"])
}

// TestJSONataEvaluator_InvalidExpression 测试无效表达式
func TestJSONataEvaluator_InvalidExpression(t *testing.T) {
	evaluator := NewJSONataEvaluator()

	input := map[string]interface{}{"value": 100.0}

	// 无效表达式语法
	result, err := evaluator.Evaluate("items[", input)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func strPtr(s string) *string {
	return &s
}
