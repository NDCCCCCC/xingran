package rpa

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// executionForEval SimpleConditionEvaluator 测试辅助（避免测试间循环依赖）。
func executionForEval(status string) rpamodels.Execution {
	return rpamodels.Execution{Status: status}
}

// =====================================================================
// Phase 74-05: data_mapper.go 测试（纯逻辑 + 规则驱动的映射）。
// =====================================================================

func TestDataMapper_TransformValue_All(t *testing.T) {
	svc := NewDataMapperService(nil)
	ctx := context.Background()

	out, err := svc.TransformValue(ctx, "abc", TransformToUpper, nil)
	require.NoError(t, err)
	assert.Equal(t, "ABC", out)

	out, err = svc.TransformValue(ctx, "ABC", TransformToLower, nil)
	require.NoError(t, err)
	assert.Equal(t, "abc", out)

	out, err = svc.TransformValue(ctx, "hello world", TransformToTitle, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello World", out)

	out, err = svc.TransformValue(ctx, "  pad  ", TransformTrim, nil)
	require.NoError(t, err)
	assert.Equal(t, "pad", out)

	out, err = svc.TransformValue(ctx, "a-b-c", TransformReplace, map[string]interface{}{"old": "-", "new": "+"})
	require.NoError(t, err)
	assert.Equal(t, "a+b+c", out)

	out, err = svc.TransformValue(ctx, "a,b,c", TransformSplit, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, out)

	out, err = svc.TransformValue(ctx, "a;b", TransformSplit, map[string]interface{}{"separator": ";"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, out)

	out, err = svc.TransformValue(ctx, []interface{}{"x", "y"}, TransformJoin, nil)
	require.NoError(t, err)
	assert.Equal(t, "x,y", out)

	// join 非数组 → 错误
	_, err = svc.TransformValue(ctx, "notarray", TransformJoin, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不是数组类型")

	out, err = svc.TransformValue(ctx, 123, TransformConcat, map[string]interface{}{"prefix": "<", "suffix": ">"})
	require.NoError(t, err)
	assert.Equal(t, "<123>", out)

	// substring 有效/无效
	out, err = svc.TransformValue(ctx, "abcdef", TransformSubstring, map[string]interface{}{"start": 1, "end": 4})
	require.NoError(t, err)
	assert.Equal(t, "bcd", out)
	_, err = svc.TransformValue(ctx, "abc", TransformSubstring, map[string]interface{}{"start": 5, "end": 9})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "子字符串参数无效")

	out, err = svc.TransformValue(ctx, `{"k":1}`, TransformParseJSON, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"k": float64(1)}, out)

	_, err = svc.TransformValue(ctx, "{bad", TransformParseJSON, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析 JSON 失败")

	out, err = svc.TransformValue(ctx, map[string]interface{}{"a": 1}, TransformStringify, nil)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, out.(string))

	// defaultValue: nil/空串替换
	out, err = svc.TransformValue(ctx, "", TransformDefaultValue, map[string]interface{}{"default": "fallback"})
	require.NoError(t, err)
	assert.Equal(t, "fallback", out)
	out, err = svc.TransformValue(ctx, "keep", TransformDefaultValue, map[string]interface{}{"default": "fallback"})
	require.NoError(t, err)
	assert.Equal(t, "keep", out)

	// 日期/数字格式化
	out, err = svc.TransformValue(ctx, "2026-08-21", TransformDateFormat, map[string]interface{}{"outputFormat": "x"})
	require.NoError(t, err)
	assert.Equal(t, "2026-08-21", out)

	out, err = svc.TransformValue(ctx, 1.23456, TransformNumberFormat, map[string]interface{}{"decimalPlaces": 3})
	require.NoError(t, err)
	assert.Equal(t, "1.235", out)

	// ParseFloat("NaN") 在 Go 中是合法输入返回 NaN, 用非数字串触发错误
	_, err = svc.TransformValue(ctx, "notnum", TransformNumberFormat, nil)
	require.Error(t, err)

	// nil 值短路
	out, err = svc.TransformValue(ctx, nil, TransformToUpper, nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	// 未知 transform 原样返回
	out, err = svc.TransformValue(ctx, "same", TransformFunction("nope"), nil)
	require.NoError(t, err)
	assert.Equal(t, "same", out)
}

func TestDataMapper_ExtractJSONPath(t *testing.T) {
	svc := NewDataMapperService(nil)
	ctx := context.Background()
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "ann",
			"tags": []interface{}{"a", "b", "c"},
		},
	}

	out, err := svc.ExtractJSONPath(ctx, data, "user.name")
	require.NoError(t, err)
	assert.Equal(t, "ann", out)

	out, err = svc.ExtractJSONPath(ctx, data, "user.tags[1]")
	require.NoError(t, err)
	assert.Equal(t, "b", out)

	// 路径不存在
	_, err = svc.ExtractJSONPath(ctx, data, "user.missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "路径不存在")

	// 非数组做索引
	_, err = svc.ExtractJSONPath(ctx, data, "user.name[0]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不是数组类型")

	// 索引越界
	_, err = svc.ExtractJSONPath(ctx, data, "user.tags[9]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "数组索引越界")

	// 无效索引字符串
	_, err = svc.ExtractJSONPath(ctx, data, "user.tags[x]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无效的数组索引")
}

func TestDataMapper_AggregateData(t *testing.T) {
	svc := NewDataMapperService(nil)
	ctx := context.Background()
	nums := []interface{}{1.0, 2.0, 3.0, 3.0}

	out, err := svc.AggregateData(ctx, nums, "sum")
	require.NoError(t, err)
	assert.Equal(t, 9.0, out)

	out, err = svc.AggregateData(ctx, nums, "avg")
	require.NoError(t, err)
	assert.Equal(t, 2.25, out)

	out, err = svc.AggregateData(ctx, nums, "min")
	require.NoError(t, err)
	assert.Equal(t, 1.0, out)

	out, err = svc.AggregateData(ctx, nums, "max")
	require.NoError(t, err)
	assert.Equal(t, 3.0, out)

	out, err = svc.AggregateData(ctx, nums, "count")
	require.NoError(t, err)
	assert.Equal(t, 4, out)

	out, err = svc.AggregateData(ctx, nums, "join")
	require.NoError(t, err)
	assert.Equal(t, "1,2,3,3", out)

	out, err = svc.AggregateData(ctx, nums, "first")
	require.NoError(t, err)
	assert.Equal(t, 1.0, out)

	out, err = svc.AggregateData(ctx, nums, "last")
	require.NoError(t, err)
	assert.Equal(t, 3.0, out)

	out, err = svc.AggregateData(ctx, nums, "unique")
	require.NoError(t, err)
	assert.Len(t, out, 3)

	// 空数据 / 非数字 / 未知类型
	_, err = svc.AggregateData(ctx, nil, "sum")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "数据为空")

	_, err = svc.AggregateData(ctx, []interface{}{"x"}, "sum")
	require.Error(t, err)

	_, err = svc.AggregateData(ctx, nums, "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的聚合类型")
}

func TestDataMapper_MapData(t *testing.T) {
	svc := NewDataMapperService(nil)
	ctx := context.Background()
	source := map[string]interface{}{"name": "ann", "age": 30}

	// direct + 常量 + 模板 + transform 后置 + 嵌套 target
	cfg := &DataMappingConfig{
		Mode: MappingModeLenient,
		Variables: map[string]interface{}{
			"env": "prod",
		},
		Rules: []DataMappingRule{
			{Target: "user.name", Type: MappingTypeDirect, Source: "name"},
			{Target: "flag", Type: MappingTypeConstant, Params: map[string]interface{}{"value": "on"}},
			{Target: "desc", Type: MappingTypeTemplate, Source: "name", Params: map[string]interface{}{"template": "${name}@${env}"}},
			{Target: "upper", Type: MappingTypeDirect, Source: "name", Transform: TransformToUpper},
			// lenient + 缺字段 + 无默认值 → 跳过
			{Target: "skipped", Type: MappingTypeDirect, Source: "missing"},
			// lenient + 缺字段 + 默认值 → 填默认
			{Target: "withdefault", Type: MappingTypeDirect, Source: "missing2", DefaultValue: "dv"},
		},
	}
	result, err := svc.MapData(ctx, cfg, source)
	require.NoError(t, err)
	assert.Equal(t, "ann", result["user"].(map[string]interface{})["name"])
	assert.Equal(t, "on", result["flag"])
	assert.Equal(t, "ann@prod", result["desc"])
	assert.Equal(t, "ANN", result["upper"])
	_, ok := result["skipped"]
	assert.False(t, ok)
	assert.Equal(t, "dv", result["withdefault"])

	// strict 模式缺字段 → 报错
	strict := &DataMappingConfig{
		Mode:  MappingModeStrict,
		Rules: []DataMappingRule{{Target: "x", Type: MappingTypeDirect, Source: "missing"}},
	}
	_, err = svc.MapData(ctx, strict, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "映射字段")

	// toMap 失败（不支持的类型）
	_, err = svc.MapData(ctx, &DataMappingConfig{Mode: MappingModeLenient}, 42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "转换源数据失败")

	// JSON 字符串源
	result, err = svc.MapData(ctx, &DataMappingConfig{
		Mode:  MappingModeLenient,
		Rules: []DataMappingRule{{Target: "n", Type: MappingTypeDirect, Source: "k"}},
	}, `{"k":"v"}`)
	require.NoError(t, err)
	assert.Equal(t, "v", result["n"])

	// YAML 字符串源
	result, err = svc.MapData(ctx, &DataMappingConfig{
		Mode:  MappingModeLenient,
		Rules: []DataMappingRule{{Target: "n", Type: MappingTypeDirect, Source: "k"}},
	}, "k: v\n")
	require.NoError(t, err)
	assert.Equal(t, "v", result["n"])

	// setNestedValue 路径冲突（target 中段是标量）
	conflict := &DataMappingConfig{
		Mode: MappingModeLenient,
		Rules: []DataMappingRule{
			{Target: "a", Type: MappingTypeConstant, Params: map[string]interface{}{"value": 1}},
			{Target: "a.b", Type: MappingTypeConstant, Params: map[string]interface{}{"value": 2}},
		},
	}
	_, err = svc.MapData(ctx, conflict, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "路径冲突")
}

func TestDataMapper_MapFields_Types(t *testing.T) {
	svc := NewDataMapperService(nil)
	ctx := context.Background()
	source := map[string]interface{}{
		"num":  7,
		"vals": []interface{}{1.0, 2.0},
		"obj":  map[string]interface{}{"inner": "iv"},
	}

	// transform 映射类型
	out, err := svc.MapFields(ctx, &DataMappingRule{
		Type: MappingTypeTransform, Source: "num", Transform: TransformNumberFormat,
	}, source)
	require.NoError(t, err)
	assert.Equal(t, "7.00", out)

	// jsonpath 映射类型
	out, err = svc.MapFields(ctx, &DataMappingRule{
		Type: MappingTypeJSONPath, Source: "obj.inner",
	}, source)
	require.NoError(t, err)
	assert.Equal(t, "iv", out)

	// lookup 命中 / 未命中走默认 / 未配置表
	out, err = svc.MapFields(ctx, &DataMappingRule{
		Type:  MappingTypeLookup, Source: "num",
		Params:      map[string]interface{}{"table": map[string]interface{}{"7": "seven"}},
		DefaultValue: "unknown",
	}, source)
	require.NoError(t, err)
	assert.Equal(t, "seven", out)

	out, err = svc.MapFields(ctx, &DataMappingRule{
		Type:  MappingTypeLookup, Source: "num",
		Params:      map[string]interface{}{"table": map[string]interface{}{"x": "y"}},
		DefaultValue: "dv",
	}, source)
	require.NoError(t, err)
	assert.Equal(t, "dv", out)

	_, err = svc.MapFields(ctx, &DataMappingRule{Type: MappingTypeLookup, Source: "num"}, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查找表未配置")

	_, err = svc.MapFields(ctx, &DataMappingRule{
		Type: MappingTypeLookup, Source: "num",
		Params: map[string]interface{}{"table": map[string]interface{}{}},
	}, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查找键不存在")

	// aggregate 映射类型
	out, err = svc.MapFields(ctx, &DataMappingRule{
		Type: MappingTypeAggregate,
		Params: map[string]interface{}{
			"type":   "sum",
			"fields": []interface{}{"num", "num"},
		},
	}, source)
	require.NoError(t, err)
	assert.Equal(t, 14.0, out)

	// 模板参数缺失
	_, err = svc.MapFields(ctx, &DataMappingRule{Type: MappingTypeTemplate}, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "模板参数缺失")

	// 未知映射类型
	_, err = svc.MapFields(ctx, &DataMappingRule{Type: MappingType("weird")}, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的映射类型")

	// direct 源字段缺失
	_, err = svc.MapFields(ctx, &DataMappingRule{Type: MappingTypeDirect, Source: "missing"}, source)
	require.Error(t, err)
}

func TestDataMapper_toMap_Variants(t *testing.T) {
	// struct 反射分支
	type sample struct {
		Name string `json:"nm"`
		Age  int
	}
	m, err := toMap(sample{Name: "x", Age: 3})
	require.NoError(t, err)
	assert.Equal(t, "x", m["nm"])
	assert.Equal(t, 3, m["Age"])

	// 无法解析的字符串
	_, err = toMap("::not json or yaml::\n\t-")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无法解析为 JSON 或 YAML")

	// 不支持类型
	_, err = toMap(3.14)
	require.Error(t, err)

	// toMapString 失败回退
	fallback := toMapString(42)
	assert.Equal(t, map[string]interface{}{"value": 42}, fallback)

	// extractValue 空路径
	v, err := extractValue(map[string]interface{}{"a": 1}, "")
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"a": 1}, v)

	// extractValue "." 路径
	v, err = extractValue(sourceMap(), ".")
	require.NoError(t, err)
	assert.Equal(t, sourceMap(), v)

	// getField struct 分支
	type withField struct{ Foo string }
	rv := getField(withField{Foo: "bar"}, "Foo")
	assert.Equal(t, "bar", rv)
	assert.Nil(t, getField(withField{}, "Nope"))
	assert.Nil(t, getField(42, "anything"))
}

func sourceMap() map[string]interface{} {
	return map[string]interface{}{"a": 1}
}
