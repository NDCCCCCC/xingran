package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// MappingType 映射类型
type MappingType string

const (
	MappingTypeDirect    MappingType = "direct"    // 直接映射
	MappingTypeTransform MappingType = "transform" // 转换映射
	MappingTypeConstant  MappingType = "constant"  // 常量映射
	MappingTypeTemplate  MappingType = "template"  // 模板映射
	MappingTypeLookup    MappingType = "lookup"    // 查找映射
	MappingTypeJSONPath  MappingType = "jsonpath"  // JSON 路径提取
	MappingTypeAggregate MappingType = "aggregate" // 聚合映射
)

// TransformFunction 转换函数类型
type TransformFunction string

const (
	TransformToUpper      TransformFunction = "toUpper"
	TransformToLower      TransformFunction = "toLower"
	TransformToTitle      TransformFunction = "toTitle"
	TransformTrim         TransformFunction = "trim"
	TransformReplace      TransformFunction = "replace"
	TransformSplit        TransformFunction = "split"
	TransformJoin         TransformFunction = "join"
	TransformDateFormat   TransformFunction = "dateFormat"
	TransformNumberFormat TransformFunction = "numberFormat"
	TransformConcat       TransformFunction = "concat"
	TransformSubstring    TransformFunction = "substring"
	TransformParseJSON    TransformFunction = "parseJSON"
	TransformStringify    TransformFunction = "stringify"
	TransformDefaultValue TransformFunction = "defaultValue"
)

// DataMappingRule 数据映射规则
type DataMappingRule struct {
	Source       string                 `json:"source"`       // 源字段路径
	Target       string                 `json:"target"`       // 目标字段路径
	Type         MappingType            `json:"type"`         // 映射类型
	Transform    TransformFunction      `json:"transform"`    // 转换函数
	Params       map[string]interface{} `json:"params"`       // 转换参数
	Required     bool                   `json:"required"`     // 是否必填
	DefaultValue interface{}            `json:"defaultValue"` // 默认值
}

// DataMappingConfig 数据映射配置
type DataMappingConfig struct {
	Rules     []DataMappingRule      `json:"rules"`
	Variables map[string]interface{} `json:"variables"`
	Mode      MappingMode            `json:"mode"` // 映射模式
}

// MappingMode 映射模式
type MappingMode string

const (
	MappingModeStrict  MappingMode = "strict"  // 严格模式：缺少字段时报错
	MappingModeLenient MappingMode = "lenient" // 宽松模式：缺少字段时使用默认值
)

// DataMapperService 数据映射服务
type DataMapperService interface {
	// MapData 执行数据映射
	MapData(ctx context.Context, config *DataMappingConfig, source interface{}) (map[string]interface{}, error)

	// MapFields 映射单个字段
	MapFields(ctx context.Context, rule *DataMappingRule, source interface{}) (interface{}, error)

	// TransformValue 转换值
	TransformValue(ctx context.Context, value interface{}, transform TransformFunction, params map[string]interface{}) (interface{}, error)

	// ExtractJSONPath 从 JSON 提取路径值
	ExtractJSONPath(ctx context.Context, data interface{}, path string) (interface{}, error)

	// AggregateData 聚合数据
	AggregateData(ctx context.Context, data []interface{}, aggregateType string) (interface{}, error)
}

// dataMapperServiceImpl 数据映射服务实现
type dataMapperServiceImpl struct {
	db        *gorm.DB
	evaluator ExpressionEvaluator
}

// NewDataMapperService 创建数据映射服务
func NewDataMapperService(db *gorm.DB) DataMapperService {
	return &dataMapperServiceImpl{
		db:        db,
		evaluator: NewExpressionEvaluator(db),
	}
}

// MapData 执行数据映射
func (s *dataMapperServiceImpl) MapData(ctx context.Context, config *DataMappingConfig, source interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 将 source 转换为 map
	sourceMap, err := toMap(source)
	if err != nil {
		return nil, fmt.Errorf("转换源数据失败: %w", err)
	}

	// 合并变量
	if config.Variables != nil {
		for k, v := range config.Variables {
			sourceMap[k] = v
		}
	}

	// 执行映射规则
	for _, rule := range config.Rules {
		value, err := s.MapFields(ctx, &rule, sourceMap)
		if err != nil {
			if config.Mode == MappingModeStrict || rule.Required {
				return nil, fmt.Errorf("映射字段 %s 失败: %w", rule.Target, err)
			}
			// 宽松模式下使用默认值
			if rule.DefaultValue != nil {
				value = rule.DefaultValue
			} else {
				continue
			}
		}

		// 设置目标值
		if err := setNestedValue(result, rule.Target, value); err != nil {
			return nil, fmt.Errorf("设置目标字段 %s 失败: %w", rule.Target, err)
		}
	}

	return result, nil
}

// MapFields 映射单个字段
func (s *dataMapperServiceImpl) MapFields(ctx context.Context, rule *DataMappingRule, source interface{}) (interface{}, error) {
	var value interface{}

	switch rule.Type {
	case MappingTypeDirect:
		val, err := extractValue(source, rule.Source)
		if err != nil {
			return nil, err
		}
		value = val

	case MappingTypeConstant:
		value = rule.Params["value"]

	case MappingTypeTemplate:
		template, ok := rule.Params["template"].(string)
		if !ok {
			return nil, fmt.Errorf("模板参数缺失")
		}
		evaluated, err := s.evaluator.EvaluateString(ctx, template, toMapString(source))
		if err != nil {
			return nil, err
		}
		value = evaluated

	case MappingTypeJSONPath:
		val, err := s.ExtractJSONPath(ctx, source, rule.Source)
		if err != nil {
			return nil, err
		}
		value = val

	case MappingTypeLookup:
		// 查找映射（从数据库或配置中）
		lookupKey := fmt.Sprintf("%v", extractValueSafe(source, rule.Source))
		var lookupErr error
		value, lookupErr = s.performLookup(ctx, rule, lookupKey)
		if lookupErr != nil {
			return nil, lookupErr
		}

	case MappingTypeTransform:
		val, err := extractValue(source, rule.Source)
		if err != nil {
			return nil, err
		}
		transformed, err := s.TransformValue(ctx, val, rule.Transform, rule.Params)
		if err != nil {
			return nil, err
		}
		value = transformed

	case MappingTypeAggregate:
		// 聚合多个字段
		aggregateType, _ := rule.Params["type"].(string)
		fields := rule.Params["fields"].([]interface{})
		values := make([]interface{}, len(fields))
		for i, field := range fields {
			val, err := extractValue(source, field.(string))
			if err != nil {
				return nil, err
			}
			values[i] = val
		}
		aggregated, err := s.AggregateData(ctx, values, aggregateType)
		if err != nil {
			return nil, err
		}
		value = aggregated

	default:
		return nil, fmt.Errorf("不支持的映射类型: %s", rule.Type)
	}

	// 应用后置转换
	if rule.Transform != "" && rule.Type != MappingTypeTransform {
		transformed, err := s.TransformValue(ctx, value, rule.Transform, rule.Params)
		if err != nil {
			return nil, err
		}
		value = transformed
	}

	return value, nil
}

// TransformValue 转换值
func (s *dataMapperServiceImpl) TransformValue(ctx context.Context, value interface{}, transform TransformFunction, params map[string]interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch transform {
	case TransformToUpper:
		str := fmt.Sprintf("%v", value)
		return strings.ToUpper(str), nil

	case TransformToLower:
		str := fmt.Sprintf("%v", value)
		return strings.ToLower(str), nil

	case TransformToTitle:
		str := fmt.Sprintf("%v", value)
		return cases.Title(language.English).String(str), nil

	case TransformTrim:
		str := fmt.Sprintf("%v", value)
		return strings.TrimSpace(str), nil

	case TransformReplace:
		str := fmt.Sprintf("%v", value)
		oldValue, _ := params["old"].(string)
		newValue, _ := params["new"].(string)
		return strings.ReplaceAll(str, oldValue, newValue), nil

	case TransformSplit:
		str := fmt.Sprintf("%v", value)
		separator, _ := params["separator"].(string)
		if separator == "" {
			separator = ","
		}
		return strings.Split(str, separator), nil

	case TransformJoin:
		arr, ok := value.([]interface{})
		if !ok {
			return nil, fmt.Errorf("值不是数组类型")
		}
		separator, _ := params["separator"].(string)
		if separator == "" {
			separator = ","
		}
		strs := make([]string, len(arr))
		for i, v := range arr {
			strs[i] = fmt.Sprintf("%v", v)
		}
		return strings.Join(strs, separator), nil

	case TransformDateFormat:
		// 日期格式转换
		return s.formatDate(value, params)

	case TransformNumberFormat:
		// 数字格式化
		return s.formatNumber(value, params)

	case TransformConcat:
		// 字符串连接
		prefix, _ := params["prefix"].(string)
		suffix, _ := params["suffix"].(string)
		str := fmt.Sprintf("%v", value)
		return prefix + str + suffix, nil

	case TransformSubstring:
		str := fmt.Sprintf("%v", value)
		start, _ := params["start"].(int)
		end := len(str)
		if endParam, ok := params["end"].(int); ok {
			end = endParam
		}
		if start < 0 || end > len(str) || start > end {
			return nil, fmt.Errorf("子字符串参数无效")
		}
		return str[start:end], nil

	case TransformParseJSON:
		str := fmt.Sprintf("%v", value)
		var result interface{}
		if err := json.Unmarshal([]byte(str), &result); err != nil {
			return nil, fmt.Errorf("解析 JSON 失败: %w", err)
		}
		return result, nil

	case TransformStringify:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return string(data), nil

	case TransformDefaultValue:
		if value == nil || value == "" {
			return params["default"], nil
		}
		return value, nil

	default:
		return value, nil
	}
}

// ExtractJSONPath 从 JSON 提取路径值
func (s *dataMapperServiceImpl) ExtractJSONPath(ctx context.Context, data interface{}, path string) (interface{}, error) {
	// 简单的 JSONPath 实现
	// 支持: field, field.nested, array[0], field.array[0].nested

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		// 处理数组索引
		if strings.Contains(part, "[") {
			openBracket := strings.Index(part, "[")
			closeBracket := strings.Index(part, "]")

			fieldName := part[:openBracket]
			indexStr := part[openBracket+1 : closeBracket]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("无效的数组索引: %s", indexStr)
			}

			// 获取字段
			if fieldName != "" {
				current = getField(current, fieldName)
			}

			// 获取数组元素
			arr, ok := current.([]interface{})
			if !ok {
				return nil, fmt.Errorf("值不是数组类型")
			}
			if index < 0 || index >= len(arr) {
				return nil, fmt.Errorf("数组索引越界: %d", index)
			}
			current = arr[index]
		} else {
			current = getField(current, part)
		}

		if current == nil {
			return nil, fmt.Errorf("路径不存在: %s", path)
		}
	}

	return current, nil
}

// AggregateData 聚合数据
func (s *dataMapperServiceImpl) AggregateData(ctx context.Context, data []interface{}, aggregateType string) (interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("数据为空")
	}

	switch aggregateType {
	case "sum":
		sum := 0.0
		for _, v := range data {
			num, err := toFloat64(v)
			if err != nil {
				return nil, err
			}
			sum += num
		}
		return sum, nil

	case "avg":
		sum := 0.0
		for _, v := range data {
			num, err := toFloat64(v)
			if err != nil {
				return nil, err
			}
			sum += num
		}
		return sum / float64(len(data)), nil

	case "min":
		min, _ := toFloat64(data[0])
		for _, v := range data[1:] {
			num, err := toFloat64(v)
			if err != nil {
				return nil, err
			}
			if num < min {
				min = num
			}
		}
		return min, nil

	case "max":
		max, _ := toFloat64(data[0])
		for _, v := range data[1:] {
			num, err := toFloat64(v)
			if err != nil {
				return nil, err
			}
			if num > max {
				max = num
			}
		}
		return max, nil

	case "count":
		return len(data), nil

	case "join":
		separator := ","
		strs := make([]string, len(data))
		for i, v := range data {
			strs[i] = fmt.Sprintf("%v", v)
		}
		return strings.Join(strs, separator), nil

	case "first":
		return data[0], nil

	case "last":
		return data[len(data)-1], nil

	case "unique":
		unique := make(map[string]interface{})
		for _, v := range data {
			key := fmt.Sprintf("%v", v)
			unique[key] = v
		}
		result := make([]interface{}, 0, len(unique))
		for _, v := range unique {
			result = append(result, v)
		}
		return result, nil

	default:
		return nil, fmt.Errorf("不支持的聚合类型: %s", aggregateType)
	}
}

// performLookup 执行查找
func (s *dataMapperServiceImpl) performLookup(ctx context.Context, rule *DataMappingRule, key string) (interface{}, error) {
	// 从参数中获取查找表
	lookupTable, ok := rule.Params["table"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("查找表未配置")
	}

	value, ok := lookupTable[key]
	if !ok {
		if rule.DefaultValue != nil {
			return rule.DefaultValue, nil
		}
		return nil, fmt.Errorf("查找键不存在: %s", key)
	}

	return value, nil
}

// formatDate 格式化日期
func (s *dataMapperServiceImpl) formatDate(value interface{}, params map[string]interface{}) (string, error) {
	// 简单实现：支持输入/输出格式
	inputFormat := "2006-01-02T15:04:05Z"
	outputFormat := "2006-01-02 15:04:05"

	if v, ok := params["inputFormat"].(string); ok {
		inputFormat = v
	}
	if v, ok := params["outputFormat"].(string); ok {
		outputFormat = v
	}

	// 尝试解析并重新格式化日期
	_ = inputFormat  // 预留用于完整实现
	_ = outputFormat // 预留用于完整实现

	// 这里需要实际的时间解析逻辑
	// 简化实现：直接返回字符串
	return fmt.Sprintf("%v", value), nil
}

// formatNumber 格式化数字
func (s *dataMapperServiceImpl) formatNumber(value interface{}, params map[string]interface{}) (string, error) {
	num, err := toFloat64(value)
	if err != nil {
		return "", err
	}

	decimalPlaces := 2
	if v, ok := params["decimalPlaces"].(int); ok {
		decimalPlaces = v
	}

	format := fmt.Sprintf("%%.%df", decimalPlaces)
	return fmt.Sprintf(format, num), nil
}

// toMap 转换为 map
func toMap(v interface{}) (map[string]interface{}, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		return val, nil
	case string:
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			// 尝试 YAML
			if err := yaml.Unmarshal([]byte(val), &result); err != nil {
				return nil, fmt.Errorf("无法解析为 JSON 或 YAML")
			}
		}
		return result, nil
	default:
		// 使用反射
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Struct {
			result := make(map[string]interface{})
			for i := 0; i < rv.NumField(); i++ {
				field := rv.Type().Field(i)
				tag := field.Tag.Get("json")
				if tag == "" {
					tag = field.Name
				}
				result[tag] = rv.Field(i).Interface()
			}
			return result, nil
		}
		return nil, fmt.Errorf("不支持的类型: %T", v)
	}
}

// toMapString 转换为字符串 map（用于表达式求值）
func toMapString(v interface{}) map[string]interface{} {
	result, err := toMap(v)
	if err != nil {
		return map[string]interface{}{"value": v}
	}
	return result
}

// extractValue 提取值
func extractValue(data interface{}, path string) (interface{}, error) {
	if path == "" || path == "." {
		return data, nil
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		current = getField(current, part)
		if current == nil {
			return nil, fmt.Errorf("字段不存在: %s", part)
		}
	}

	return current, nil
}

// extractValueSafe 安全提取值（不返回错误）
func extractValueSafe(data interface{}, path string) interface{} {
	val, _ := extractValue(data, path)
	return val
}

// getField 获取字段值
func getField(data interface{}, field string) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		return v[field]
	case map[interface{}]interface{}:
		return v[field]
	default:
		rv := reflect.ValueOf(data)
		if rv.Kind() == reflect.Struct {
			fieldVal := rv.FieldByName(field)
			if fieldVal.IsValid() {
				return fieldVal.Interface()
			}
		}
		return nil
	}
}

// setNestedValue 设置嵌套值
func setNestedValue(data map[string]interface{}, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	current := data

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}

		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}

		next, ok := current[part].(map[string]interface{})
		if !ok {
			return fmt.Errorf("路径冲突: %s 不是对象", part)
		}
		current = next
	}

	return nil
}

// toFloat64 转换为 float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("无法转换为数字: %T", v)
	}
}
