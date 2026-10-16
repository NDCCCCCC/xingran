package v1

// =====================================================================
// Phase 80-03 Task 7 part A: job_cron_util 全纯一次收满(67 stmts 覆盖)。
//
// 全纯直调表驱动,无 fixture;CronExpressionBuilder 链式 + EveryXxx 常量。
// =====================================================================

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// validateCronExpression
// =====================================================================

func TestJcu8003_ValidateCronExpression(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"0 * * * * ?", true},
		{"0 0 * * * ?", true},
		{"0 0 0 1 * ?", true},
		{"0 0 0 * * MON", true},
		{"30 15 10 ? * MON-FRI", true},
		{"", false},
		{"not a cron", false},
		{"60 0 * * * ?", false}, // 60 秒越界
		{"* * * * *", false},    // 5 字段,需 6/7
		{"a b c d e f", false},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			assert.Equal(t, tt.want, validateCronExpression(tt.expr))
		})
	}
}

// =====================================================================
// calculateNextRunTime
// =====================================================================

func TestJcu8003_CalculateNextRunTime(t *testing.T) {
	t.Run("合法表达式_返回未来时间", func(t *testing.T) {
		got := calculateNextRunTime("0 * * * * ?")
		assert.False(t, got.IsZero())
		assert.True(t, got.After(time.Now().Add(-time.Second)),
			"未来执行时间应不早于 now-1s(防 edge 时序 flake)")
	})

	t.Run("非法表达式_返零值", func(t *testing.T) {
		got := calculateNextRunTime("garbage")
		assert.True(t, got.IsZero())
	})

	t.Run("空格表达式_返零值", func(t *testing.T) {
		got := calculateNextRunTime("")
		assert.True(t, got.IsZero())
	})
}

// =====================================================================
// GetCronDescription
// =====================================================================

func TestJcu8003_GetCronDescription(t *testing.T) {
	tests := []struct {
		expr       string
		wantSubstr string
	}{
		{"0 * * * * ?", "每分钟执行"},
		{"0 0 * * * ?", "每小时执行"},
		{"0 0 0 * * ?", "每天0点执行"},
		{"0 0 0 * * MON ?", "每周一0点执行"},
		{"0 0 0 1 * ?", "每月1号0点执行"},
		{"garbage", "无效的Cron表达式"},
		{"0 */5 * * * ?", "下次执行:"}, // 走解析分支
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			desc := GetCronDescription(tt.expr)
			assert.Contains(t, desc, tt.wantSubstr)
		})
	}
}

// =====================================================================
// ParseCronExpression + ValidateCronExpression(聚合三返回值)
// =====================================================================

func TestJcu8003_ParseCronExpression(t *testing.T) {
	s, m, h, dom, mon, dow, err := ParseCronExpression("0 15 10 ? * MON")
	require.NoError(t, err)
	assert.Equal(t, "0", s)
	assert.Equal(t, "15", m)
	assert.Equal(t, "10", h)
	assert.Equal(t, "?", dom)
	assert.Equal(t, "*", mon)
	assert.Equal(t, "MON", dow)

	t.Run("空字符串_报错", func(t *testing.T) {
		_, _, _, _, _, _, err := ParseCronExpression("")
		assert.Error(t, err)
	})

	t.Run("少字段_尾部补空_返 nil", func(t *testing.T) {
		// QUIRK-80-03-G(就地锁定):ParseCronExpression 始终返 6 字段,缺位补空串且不报错。
		// 表达"格式错误"的责任交给 cron parser(ValidateCronExpression/robfig);
		// 本函数只做字段切分,不参与 cron 语义校验。
		s, m, h2, _, _, _, err := ParseCronExpression("0 15 10")
		require.NoError(t, err)
		assert.Equal(t, "0", s)
		assert.Equal(t, "15", m)
		assert.Equal(t, "10", h2)
	})

	t.Run("多字段超7_被截断为6", func(t *testing.T) {
		s, _, _, _, _, _, err := ParseCronExpression("0 15 10 ? * MON EXTRA1 EXTRA2")
		require.NoError(t, err)
		assert.Equal(t, "0", s)
	})
}

func TestJcu8003_ValidateCronExpression_Aggregate(t *testing.T) {
	t.Run("合法_三返回值全填", func(t *testing.T) {
		valid, next, desc, err := ValidateCronExpression("0 * * * * ?")
		require.NoError(t, err)
		assert.True(t, valid)
		assert.NotEmpty(t, next, "nextRunTime 应格式化为 YYYY-MM-DD HH:MM:SS")
		assert.NotEmpty(t, desc)
	})

	t.Run("非法_返错+valid=false", func(t *testing.T) {
		valid, _, _, err := ValidateCronExpression("garbage")
		assert.Error(t, err)
		assert.False(t, valid)
	})
}

// =====================================================================
// generateCronExpression(常被 Builder/Custom 调用,这里直测断言格式)
// =====================================================================

func TestJcu8003_GenerateCronExpression(t *testing.T) {
	got := generateCronExpression("0", "30", "10", "*", "*", "?")
	assert.Equal(t, "0 30 10 * * ?", got, "六字段空格拼接")
}

// =====================================================================
// CronExpressionBuilder 链式 + EveryXxx / Custom 常量
// =====================================================================

func TestJcu8003_CronExpressionBuilder_Chained(t *testing.T) {
	b := NewCronExpressionBuilder()
	built := b.SetSeconds("15").SetMinutes("30").SetHours("10").SetDay("5").SetMonth("*").SetDayOfWeek("?").Build()
	assert.Equal(t, "15 30 10 5 * ?", built)
	// 链式返回同一实例(供连续调用)
	b2 := NewCronExpressionBuilder().SetSeconds("5")
	assert.NotNil(t, b2)
	built2 := b2.SetMinutes("30").Build()
	assert.Equal(t, "5 30 * * * ?", built2, "未设字段用默认值(hours/day/month/dayOfWeek)")
}

func TestJcu8003_EveryXxx_Constants(t *testing.T) {
	assert.Equal(t, "0 * * * * ?", EveryMinute())
	assert.Equal(t, "0 0 * * * ?", EveryHour())
	assert.Equal(t, "0 0 0 * * ?", EveryDay())
	// EveryWeek 不再追加 "?" 字段 —— dayOfWeek 已是确定值,不再需要冲突占位
	assert.Equal(t, "0 0 0 * * MON", EveryWeek())
	assert.Equal(t, "0 0 0 1 * ?", EveryMonth())
}

func TestJcu8003_Custom_Format(t *testing.T) {
	got := Custom("5", "30", "10", "*", "*", "?")
	assert.Equal(t, "5 30 10 * * ?", got, "Custom 拼接结果与 generateCronExpression 一致")
}

// =====================================================================
// GetCommonCronExpressions + ParseNextRunTimes
// =====================================================================

func TestJcu8003_GetCommonCronExpressions(t *testing.T) {
	list := GetCommonCronExpressions()
	assert.NotEmpty(t, list)
	// 每条 entry 含 value + label 字段
	for _, e := range list {
		assert.Contains(t, e, "value")
		assert.Contains(t, e, "label")
	}
}

func TestJcu8003_ParseNextRunTimes(t *testing.T) {
	times, err := ParseNextRunTimes("0 * * * * ?", 5)
	require.NoError(t, err)
	assert.Len(t, times, 5)
	// 升序排列
	for i := 1; i < len(times); i++ {
		prev, _ := time.Parse("2006-01-02 15:04:05", times[i-1])
		curr, _ := time.Parse("2006-01-02 15:04:05", times[i])
		assert.True(t, curr.After(prev) || curr.Equal(prev), "应升序")
	}
}

// 烟雾测试:确认字符串组合函数不会 panic(对真实生产表达式做基本正确性验证)。
func TestJcu8003_CronComposition_Smoke(t *testing.T) {
	expr := Custom("0", "0", "9-17", "*", "*", "?")
	assert.True(t, validateCronExpression(expr), "Custom 输出经得起 validate 验证")
	assert.Contains(t, GetCronDescription(expr), "下次执行:",
		"不在字典表内的合法表达式走解析分支")
}

// 触发 strings/time 引用不被未使用。
var _ = strings.Builder{}
var _ = time.Second
