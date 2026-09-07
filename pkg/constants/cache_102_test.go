package constants

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCaptchaCacheKeyEquivalence 等价快照测试：D-102-5①
// 期望列 = 原字面量快照，被测列 = 常量本身
// 任意常量值漂移（大小写/分隔符/动词）→ 测试失败
func TestCaptchaCacheKeyEquivalence(t *testing.T) {
	cases := []struct {
		name string // 常量名
		want string // 原字面量快照
		got string // 被测常量值
	}{
		{"CaptchaRateLimitKeyFormat", "captcha:rate:%s", CaptchaRateLimitKeyFormat},
		{"CaptchaDataKeyFormat", "captcha:data:%s", CaptchaDataKeyFormat},
		{"CaptchaAttemptsKeyFormat", "captcha:attempts:%s", CaptchaAttemptsKeyFormat},
		{"LoginFailKeyFormat", "login:fail:%s", LoginFailKeyFormat},
		{"CaptchaBgListKeyFormat", "captcha:bg:list:%s:%d", CaptchaBgListKeyFormat},
		{"CaptchaCachePoolPrefixFormat", "captcha:cache:pool:%s:%d", CaptchaCachePoolPrefixFormat},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.got,
				"常量 %s 值漂移：期望 %q，实际 %q", tc.name, tc.want, tc.got)
		})
	}
}
