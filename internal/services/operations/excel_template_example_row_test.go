package operations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExcelTemplate_ExampleRowMarkedAndStable 守护模板示例行的两个质量点：
//
//  1. **可发现性**：示例行必须带浅灰斜体样式 + 首列批注（提示"导入前请删除本行"）。
//     历史问题：用户未删示例行直接上传 → 报「引用的 所属楼宇 不存在: 示例楼宇A」，
//     报错与真实原因（示例行未删）不匹配，排查成本高。
//
//  2. **稳定性**：示例行的枚举列取值必须每次生成都一致。
//     历史问题：Options 是 map，遍历顺序随机 → 每次下载模板的"状态"示例值
//     在"空闲/占用/维护"之间跳变，用户对照模板排查时困惑。
func TestExcelTemplate_ExampleRowMarkedAndStable(t *testing.T) {
	svc := NewExcelService(nil, nil, nil, nil)

	// 1) 示例值稳定性：连续生成 5 次模板，枚举列取值必须完全一致
	var firstRow []string
	for i := 0; i < 5; i++ {
		f, err := svc.GenerateTemplate("workstation")
		require.NoError(t, err)
		rows, err := f.GetRows("工位列表")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(rows), 2, "模板应含表头 + 示例行")

		if i == 0 {
			firstRow = rows[1]
			continue
		}
		assert.Equal(t, firstRow, rows[1], "第 %d 次生成的示例行应与首次完全一致", i+1)
	}

	// 2) 枚举列必须取到最小 key 对应的 label（0=空闲），而不是随机项
	f, err := svc.GenerateTemplate("workstation")
	require.NoError(t, err)
	rows, err := f.GetRows("工位列表")
	require.NoError(t, err)

	// 工位配置中 status 列的 Options 是 {0:空闲, 1:占用, 2:维护}
	statusIdx := -1
	for ci, h := range rows[0] {
		if h == "状态" || h == "工位状态" {
			statusIdx = ci
		}
	}
	require.NotEqual(t, -1, statusIdx, "模板应含工位状态列")
	assert.Equal(t, "空闲", rows[1][statusIdx], "状态示例值应固定为最小 key(0)对应的 label")

	// 3) 示例行必须带批注提示删除
	comments, err := f.GetComments("工位列表")
	require.NoError(t, err)
	require.NotEmpty(t, comments, "示例行首列应带批注提示删除")
	found := false
	for _, c := range comments {
		if c.Cell == "A2" {
			found = true
			assert.Contains(t, c.Text, "删除", "批注文案应提示删除示例行")
		}
	}
	assert.True(t, found, "示例行首列(A2)应带批注")
}
