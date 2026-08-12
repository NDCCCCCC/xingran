package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"gorm.io/gorm"
)

// ConditionType 条件类型
type ConditionType string

const (
	ConditionTypeEquals         ConditionType = "equals"         // 等于
	ConditionTypeNotEquals      ConditionType = "notEquals"      // 不等于
	ConditionTypeContains       ConditionType = "contains"       // 包含
	ConditionTypeNotContains    ConditionType = "notContains"    // 不包含
	ConditionTypeGreaterThan    ConditionType = "greaterThan"    // 大于
	ConditionTypeLessThan       ConditionType = "lessThan"       // 小于
	ConditionTypeGreaterOrEqual ConditionType = "greaterOrEqual" // 大于等于
	ConditionTypeLessOrEqual    ConditionType = "lessOrEqual"    // 小于等于
	ConditionTypeMatches        ConditionType = "matches"        // 正则匹配
	ConditionTypeExists         ConditionType = "exists"         // 存在
	ConditionTypeEmpty          ConditionType = "empty"          // 为空
)

// LoopType 循环类型
type LoopType string

const (
	LoopTypeCount   LoopType = "count"   // 计数循环
	LoopTypeWhile   LoopType = "while"   // 条件循环
	LoopTypeForEach LoopType = "forEach" // 遍历循环
	LoopTypeUntil   LoopType = "until"   // 直到循环
)

// varPlaceholderRegex 变量占位符正则表达式
// 匹配 ${varName} 格式，varName 不含 '}'
// 提取为 package-level var 避免每次 EvaluateString 调用重复编译
var varPlaceholderRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// ExpressionEvaluator 表达式求值器
type ExpressionEvaluator interface {
	// EvaluateBool 计算布尔表达式
	EvaluateBool(ctx context.Context, expr string, variables map[string]interface{}) (bool, error)

	// EvaluateString 计算字符串表达式
	EvaluateString(ctx context.Context, expr string, variables map[string]interface{}) (string, error)

	// EvaluateNumber 计算数值表达式
	EvaluateNumber(ctx context.Context, expr string, variables map[string]interface{}) (float64, error)
}

// expressionEvaluatorImpl 表达式求值器实现
type expressionEvaluatorImpl struct {
	db *gorm.DB
}

// NewExpressionEvaluator 创建表达式求值器
func NewExpressionEvaluator(db *gorm.DB) ExpressionEvaluator {
	return &expressionEvaluatorImpl{db: db}
}

// EvaluateBool 计算布尔表达式
func (e *expressionEvaluatorImpl) EvaluateBool(ctx context.Context, expr string, variables map[string]interface{}) (bool, error) {
	// 解析条件表达式
	// 格式: "${var} equals value" 或 "${var} contains value"

	parts := strings.Fields(expr)
	if len(parts) < 2 {
		return false, fmt.Errorf("无效的条件表达式: %s", expr)
	}

	// 提取变量名和值
	varValue, err := e.extractVariableValue(parts[0], variables)
	if err != nil {
		return false, err
	}

	operator := parts[1]
	var compareValue interface{}
	if len(parts) > 2 {
		compareValue = strings.Join(parts[2:], " ")
	}

	return e.compareValues(varValue, ConditionType(operator), compareValue)
}

// EvaluateString 计算字符串表达式
func (e *expressionEvaluatorImpl) EvaluateString(ctx context.Context, expr string, variables map[string]interface{}) (string, error) {
	// 替换变量占位符 ${varName}
	result := expr

	matches := varPlaceholderRegex.FindAllStringSubmatch(expr, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			value, err := e.extractVariableValue("${"+varName+"}", variables)
			if err != nil {
				return "", err
			}
			result = strings.ReplaceAll(result, match[0], fmt.Sprintf("%v", value))
		}
	}

	return result, nil
}

// EvaluateNumber 计算数值表达式
func (e *expressionEvaluatorImpl) EvaluateNumber(ctx context.Context, expr string, variables map[string]interface{}) (float64, error) {
	evaluated, err := e.EvaluateString(ctx, expr, variables)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(evaluated, 64)
}

// extractVariableValue 提取变量值
func (e *expressionEvaluatorImpl) extractVariableValue(varExpr string, variables map[string]interface{}) (interface{}, error) {
	if !strings.HasPrefix(varExpr, "${") || !strings.HasSuffix(varExpr, "}") {
		return varExpr, nil
	}

	varName := varExpr[2 : len(varExpr)-1]

	// 支持嵌套访问，如 data.user.name
	parts := strings.Split(varName, ".")
	value, ok := variables[parts[0]]
	if !ok {
		return nil, fmt.Errorf("变量不存在: %s", parts[0])
	}

	// 处理嵌套访问
	for i := 1; i < len(parts); i++ {
		switch v := value.(type) {
		case map[string]interface{}:
			val, ok := v[parts[i]]
			if !ok {
				return nil, fmt.Errorf("字段不存在: %s", parts[i])
			}
			value = val
		default:
			return nil, fmt.Errorf("不支持嵌套访问类型: %T", value)
		}
	}

	return value, nil
}

// compareValues 比较两个值
func (e *expressionEvaluatorImpl) compareValues(left interface{}, operator ConditionType, right interface{}) (bool, error) {
	switch operator {
	case ConditionTypeEquals:
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right), nil

	case ConditionTypeNotEquals:
		return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right), nil

	case ConditionTypeContains:
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		return strings.Contains(leftStr, rightStr), nil

	case ConditionTypeNotContains:
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		return !strings.Contains(leftStr, rightStr), nil

	case ConditionTypeGreaterThan:
		return e.compareNumbers(left, ">", right)

	case ConditionTypeLessThan:
		return e.compareNumbers(left, "<", right)

	case ConditionTypeGreaterOrEqual:
		return e.compareNumbers(left, ">=", right)

	case ConditionTypeLessOrEqual:
		return e.compareNumbers(left, "<=", right)

	case ConditionTypeMatches:
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		matched, err := regexp.MatchString(rightStr, leftStr)
		return matched, err

	case ConditionTypeExists:
		return left != nil && left != "", nil

	case ConditionTypeEmpty:
		return left == nil || left == "", nil

	default:
		return false, fmt.Errorf("不支持的操作符: %s", operator)
	}
}

// compareNumbers 比较数值
func (e *expressionEvaluatorImpl) compareNumbers(left interface{}, op string, right interface{}) (bool, error) {
	leftNum, err := toFloat64Helper(left)
	if err != nil {
		return false, fmt.Errorf("左侧值不是数字: %w", err)
	}

	rightNum, err := toFloat64Helper(right)
	if err != nil {
		return false, fmt.Errorf("右侧值不是数字: %w", err)
	}

	switch op {
	case ">":
		return leftNum > rightNum, nil
	case "<":
		return leftNum < rightNum, nil
	case ">=":
		return leftNum >= rightNum, nil
	case "<=":
		return leftNum <= rightNum, nil
	default:
		return false, fmt.Errorf("不支持的操作符: %s", op)
	}
}

// toFloat64Helper 转换为 float64（辅助函数）
func toFloat64Helper(v interface{}) (float64, error) {
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

// FlowControlService 流程控制服务
type FlowControlService interface {
	// ExecuteCondition 执行条件分支
	ExecuteCondition(ctx context.Context, condition *ConditionAction, variables map[string]interface{}) (bool, error)

	// ExecuteLoop 执行循环
	ExecuteLoop(ctx context.Context, loop *LoopAction, variables map[string]interface{}) ([]map[string]interface{}, error)

	// EvaluateCondition 评估条件
	EvaluateCondition(ctx context.Context, expr string, variables map[string]interface{}) (bool, error)
}

// flowControlServiceImpl 流程控制服务实现
type flowControlServiceImpl struct {
	db               *gorm.DB
	evaluator        ExpressionEvaluator
	executionService ExecutionService
}

// NewFlowControlService 创建流程控制服务
func NewFlowControlService(db *gorm.DB, execService ExecutionService) FlowControlService {
	return &flowControlServiceImpl{
		db:               db,
		evaluator:        NewExpressionEvaluator(db),
		executionService: execService,
	}
}

// ConditionAction 条件分支动作
type ConditionAction struct {
	Type         ActionConditionType `json:"type"`
	Expression   string              `json:"expression"`
	TrueActions  []json.RawMessage   `json:"trueActions"`
	FalseActions []json.RawMessage   `json:"falseActions"`
}

// ActionConditionType 条件动作类型
type ActionConditionType string

const (
	ActionTypeCondition ActionConditionType = "condition"
	ActionTypeLoop      ActionConditionType = "loop"
)

// ExecuteCondition 执行条件分支
func (s *flowControlServiceImpl) ExecuteCondition(ctx context.Context, condition *ConditionAction, variables map[string]interface{}) (bool, error) {
	result, err := s.evaluator.EvaluateBool(ctx, condition.Expression, variables)
	if err != nil {
		return false, fmt.Errorf("条件评估失败: %w", err)
	}

	return result, nil
}

// LoopAction 循环动作
type LoopAction struct {
	Type       LoopType          `json:"type"`
	Count      int               `json:"count,omitempty"`      // 计数循环次数
	Expression string            `json:"expression,omitempty"` // while/until 条件
	Variable   string            `json:"variable,omitempty"`   // 遍历变量名
	Items      []interface{}     `json:"items,omitempty"`      // 遍历列表
	Actions    []json.RawMessage `json:"actions"`              // 循环体动作
	MaxIter    int               `json:"maxIter,omitempty"`    // 最大迭代次数（防止无限循环）
}

// ExecuteLoop 执行循环
func (s *flowControlServiceImpl) ExecuteLoop(ctx context.Context, loop *LoopAction, variables map[string]interface{}) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0)
	var iterCount int
	maxIter := loop.MaxIter
	if maxIter <= 0 {
		maxIter = 1000 // 默认最大迭代次数
	}

	switch loop.Type {
	case LoopTypeCount:
		for i := 0; i < loop.Count; i++ {
			if iterCount >= maxIter {
				return nil, fmt.Errorf("循环超过最大迭代次数: %d", maxIter)
			}
			variables["loopIndex"] = i
			variables["loopValue"] = i
			result := map[string]interface{}{"loopIndex": i, "loopValue": i}
			results = append(results, result)
			iterCount++
		}

	case LoopTypeWhile:
		for iterCount < maxIter {
			shouldContinue, err := s.evaluator.EvaluateBool(ctx, loop.Expression, variables)
			if err != nil {
				return nil, fmt.Errorf("while 条件评估失败: %w", err)
			}
			if !shouldContinue {
				break
			}
			results = append(results, map[string]interface{}{"loopIndex": iterCount})
			variables["loopIndex"] = iterCount
			iterCount++
		}

	case LoopTypeUntil:
		for iterCount < maxIter {
			shouldStop, err := s.evaluator.EvaluateBool(ctx, loop.Expression, variables)
			if err != nil {
				return nil, fmt.Errorf("until 条件评估失败: %w", err)
			}
			if shouldStop {
				break
			}
			results = append(results, map[string]interface{}{"loopIndex": iterCount})
			variables["loopIndex"] = iterCount
			iterCount++
		}

	case LoopTypeForEach:
		if loop.Items == nil {
			// 尝试从变量中获取列表
			val, ok := variables[loop.Variable]
			if !ok {
				return nil, fmt.Errorf("遍历变量不存在: %s", loop.Variable)
			}
			items, err := toSlice(val)
			if err != nil {
				return nil, fmt.Errorf("无法转换为列表: %w", err)
			}
			loop.Items = items
		}

		for i, item := range loop.Items {
			if iterCount >= maxIter {
				return nil, fmt.Errorf("循环超过最大迭代次数: %d", maxIter)
			}
			variables["loopIndex"] = i
			variables["loopItem"] = item
			results = append(results, map[string]interface{}{"loopIndex": i, "loopItem": item})
			iterCount++
		}

	default:
		return nil, fmt.Errorf("不支持的循环类型: %s", loop.Type)
	}

	return results, nil
}

// EvaluateCondition 评估条件
func (s *flowControlServiceImpl) EvaluateCondition(ctx context.Context, expr string, variables map[string]interface{}) (bool, error) {
	return s.evaluator.EvaluateBool(ctx, expr, variables)
}

// toSlice 转换为切片
func toSlice(v interface{}) ([]interface{}, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		result := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, nil
	}
	return nil, fmt.Errorf("不是切片类型")
}

// ConditionEvaluator 条件求值接口（用于向后兼容）
type ConditionEvaluator interface {
	Evaluate(ctx context.Context, execution *rpamodels.Execution) (bool, error)
}

// SimpleConditionEvaluator 简单条件求值器
type SimpleConditionEvaluator struct {
	expr string
}

// NewSimpleConditionEvaluator 创建简单条件求值器
func NewSimpleConditionEvaluator(expr string) ConditionEvaluator {
	return &SimpleConditionEvaluator{expr: expr}
}

// Evaluate 评估条件
func (e *SimpleConditionEvaluator) Evaluate(ctx context.Context, execution *rpamodels.Execution) (bool, error) {
	// 简单实现：基于执行状态
	if e.expr == "status == 'success'" {
		return execution.Status == string(rpamodels.RPAExecutionStatusSuccess), nil
	}
	if e.expr == "status == 'failed'" {
		return execution.Status == string(rpamodels.RPAExecutionStatusFailed), nil
	}
	return false, fmt.Errorf("不支持的条件表达式: %s", e.expr)
}
