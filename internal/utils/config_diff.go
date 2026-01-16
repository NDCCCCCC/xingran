package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

// DiffConfig 计算配置差异
func DiffConfig(oldConfig, newConfig string) *ConfigDiff {
	if oldConfig == newConfig {
		return &ConfigDiff{
			HasChanges: false,
		}
	}

	return &ConfigDiff{
		HasChanges:   true,
		OldConfig:    oldConfig,
		NewConfig:    newConfig,
		OldHash:      CalculateHash(oldConfig),
		NewHash:      CalculateHash(newConfig),
		LinesAdded:   calculateLinesAdded(oldConfig, newConfig),
		LinesRemoved: calculateLinesRemoved(oldConfig, newConfig),
	}
}

// ConfigDiff 配置差异结果
type ConfigDiff struct {
	HasChanges   bool
	OldConfig    string
	NewConfig    string
	OldHash      string
	NewHash      string
	LinesAdded   int
	LinesRemoved int
}

// CalculateHash 计算配置文件哈希值
func CalculateHash(config string) string {
	if config == "" {
		return ""
	}

	hash := md5.New()
	hash.Write([]byte(config))
	return hex.EncodeToString(hash.Sum(nil))
}

// calculateLinesAdded 计算新增的行数
func calculateLinesAdded(oldConfig, newConfig string) int {
	oldLines := splitLines(oldConfig)
	newLines := splitLines(newConfig)

	added := 0
	newLineSet := make(map[string]bool)

	for _, line := range newLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !isCommentLine(trimmed) {
			newLineSet[trimmed] = true
		}
	}

	for _, line := range oldLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !isCommentLine(trimmed) {
			delete(newLineSet, trimmed)
		}
	}

	added = len(newLineSet)
	return added
}

// calculateLinesRemoved 计算删除的行数
func calculateLinesRemoved(oldConfig, newConfig string) int {
	oldLines := splitLines(oldConfig)
	newLines := splitLines(newConfig)

	removed := 0
	oldLineSet := make(map[string]bool)

	for _, line := range oldLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !isCommentLine(trimmed) {
			oldLineSet[trimmed] = true
		}
	}

	for _, line := range newLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !isCommentLine(trimmed) {
			delete(oldLineSet, trimmed)
		}
	}

	removed = len(oldLineSet)
	return removed
}

// GetUnifiedDiff 获取统一格式的差异
func GetUnifiedDiff(oldConfig, newConfig string, contextLines int) string {
	oldLines := splitLines(oldConfig)
	newLines := splitLines(newConfig)

	diff := calculateUnifiedDiff(oldLines, newLines, contextLines)
	return formatUnifiedDiff(diff)
}

// DiffLine 差异行
type DiffLine struct {
	Type    DiffLineType
	Content string
	OldNum  int
	NewNum  int
}

// DiffLineType 差异行类型
type DiffLineType int

const (
	DiffLineContext DiffLineType = iota // 上下文行
	DiffLineAdded                       // 新增行
	DiffLineRemoved                     // 删除行
)

// UnifiedDiff 统一差异
type UnifiedDiff struct {
	OldLines []string
	NewLines []string
	Hunks    []*DiffHunk
}

// DiffHunk 差异块
type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []*DiffLine
}

// calculateUnifiedDiff 计算统一差异
func calculateUnifiedDiff(oldLines, newLines []string, contextLines int) *UnifiedDiff {
	lcs := longestCommonSubsequence(oldLines, newLines)

	diff := &UnifiedDiff{
		OldLines: oldLines,
		NewLines: newLines,
	}

	// 构建差异块
	var currentHunk *DiffHunk
	oldIdx, newIdx := 0, 0

	for i := 0; i < len(lcs); {
		// 跳过相同的行
		for oldIdx < len(oldLines) && newIdx < len(newLines) &&
			oldIdx < len(lcs) && newIdx < len(lcs) &&
			i < len(lcs) &&
			lcs[i][0] == oldIdx && lcs[i][1] == newIdx &&
			oldLines[oldIdx] == newLines[newIdx] {
			oldIdx++
			newIdx++
			i++
		}

		// 收集差异
		hunkOldStart := oldIdx
		_ = hunkOldStart // 暂未使用，保留用于后续扩展
		_ = newIdx       // 暂未使用，保留用于后续扩展
		var hunkLines []*DiffLine

		// 处理删除的行
		for oldIdx < len(oldLines) && (i >= len(lcs) || oldIdx < lcs[i][0]) {
			hunkLines = append(hunkLines, &DiffLine{
				Type:    DiffLineRemoved,
				Content: oldLines[oldIdx],
				OldNum:  oldIdx + 1,
			})
			oldIdx++
		}

		// 处理新增的行
		for newIdx < len(newLines) && (i >= len(lcs) || newIdx < lcs[i][1]) {
			hunkLines = append(hunkLines, &DiffLine{
				Type:    DiffLineAdded,
				Content: newLines[newIdx],
				NewNum:  newIdx + 1,
			})
			newIdx++
		}

		// 添加上下文行
		if len(hunkLines) > 0 {
			// 添加前导上下文
			contextStart := hunkOldStart - contextLines
			if contextStart < 0 {
				contextStart = 0
			}
			for i := contextStart; i < hunkOldStart; i++ {
				hunkLines = append([]*DiffLine{{
					Type:    DiffLineContext,
					Content: oldLines[i],
					OldNum:  i + 1,
					NewNum:  i + 1,
				}}, hunkLines...)
			}

			// 添加尾随上下文
			contextEnd := oldIdx + contextLines
			if contextEnd > len(oldLines) {
				contextEnd = len(oldLines)
			}
			for i := oldIdx; i < contextEnd; i++ {
				hunkLines = append(hunkLines, &DiffLine{
					Type:    DiffLineContext,
					Content: oldLines[i],
					OldNum:  i + 1,
					NewNum:  i + 1,
				})
			}

			currentHunk = &DiffHunk{
				OldStart: contextStart + 1,
				OldCount: oldIdx - contextStart,
				NewStart: contextStart + 1,
				NewCount: newIdx - contextStart,
				Lines:    hunkLines,
			}
			diff.Hunks = append(diff.Hunks, currentHunk)
		}

		// 继续处理相同的行
		for oldIdx < len(oldLines) && newIdx < len(newLines) &&
			oldLines[oldIdx] == newLines[newIdx] {
			oldIdx++
			newIdx++
		}
	}

	return diff
}

// formatUnifiedDiff 格式化统一差异
func formatUnifiedDiff(diff *UnifiedDiff) string {
	if len(diff.Hunks) == 0 {
		return ""
	}

	var builder strings.Builder

	for _, hunk := range diff.Hunks {
		// 写入差异块头
		builder.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount))

		// 写入差异行
		for _, line := range hunk.Lines {
			prefix := " "
			switch line.Type {
			case DiffLineAdded:
				prefix = "+"
			case DiffLineRemoved:
				prefix = "-"
			}
			builder.WriteString(fmt.Sprintf("%s%s\n", prefix, line.Content))
		}
	}

	return builder.String()
}

// longestCommonSubsequence 计算最长公共子序列
func longestCommonSubsequence(oldLines, newLines []string) [][]int {
	m, n := len(oldLines), len(newLines)

	// 构建LCS表
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// 回溯构建LCS
	var lcs [][]int
	i, j := m, n
	for i > 0 && j > 0 {
		if oldLines[i-1] == newLines[j-1] {
			lcs = append([][]int{{i - 1, j - 1}}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// GetSideBySideDiff 获取并排格式的差异
func GetSideBySideDiff(oldConfig, newConfig string) string {
	oldLines := splitLines(oldConfig)
	newLines := splitLines(newConfig)

	maxWidth := 80
	for _, line := range oldLines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	for _, line := range newLines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	var builder strings.Builder
	builder.WriteString(strings.Repeat("=", (maxWidth*2)+7) + "\n")
	builder.WriteString(fmt.Sprintf("%-*s | %-*s\n", maxWidth, "OLD", maxWidth, "NEW"))
	builder.WriteString(strings.Repeat("-", (maxWidth*2)+7) + "\n")

	maxLen := max(len(oldLines), len(newLines))
	for i := 0; i < maxLen; i++ {
		oldLine := ""
		newLine := ""

		prefix := "  "

		if i < len(oldLines) && i < len(newLines) {
			oldLine = oldLines[i]
			newLine = newLines[i]
			if oldLine != newLine {
				prefix = "||"
			}
		} else if i < len(oldLines) {
			oldLine = oldLines[i]
			prefix = "< "
		} else if i < len(newLines) {
			newLine = newLines[i]
			prefix = "> "
		}

		builder.WriteString(fmt.Sprintf("%s%-*s | %-*s\n",
			prefix, maxWidth, oldLine, maxWidth, newLine))
	}

	builder.WriteString(strings.Repeat("=", (maxWidth*2)+7) + "\n")
	return builder.String()
}

// splitLines 分割配置为行
func splitLines(config string) []string {
	if config == "" {
		return []string{}
	}
	return strings.Split(config, "\n")
}

// isCommentLine 判断是否为注释行
func isCommentLine(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return false
	}

	// 检查常见注释前缀
	commentPrefixes := []string{"!", "#", "//", "/*", "*", ";"}
	for _, prefix := range commentPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

// normalizeConfigLine 规范化配置行
func normalizeConfigLine(line string) string {
	// 去除前后空白
	line = strings.TrimSpace(line)

	// 如果是空行或注释，直接返回
	if line == "" || isCommentLine(line) {
		return line
	}

	// 统一空格为单个空格
	var normalized strings.Builder
	prevSpace := true
	for _, r := range line {
		if unicode.IsSpace(r) {
			if !prevSpace {
				normalized.WriteRune(' ')
				prevSpace = true
			}
		} else {
			normalized.WriteRune(r)
			prevSpace = false
		}
	}

	return normalized.String()
}

// NormalizeConfig 规范化配置内容
func NormalizeConfig(config string) string {
	lines := splitLines(config)
	normalizedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		normalizedLine := normalizeConfigLine(line)
		normalizedLines = append(normalizedLines, normalizedLine)
	}

	return strings.Join(normalizedLines, "\n")
}

// max 返回两个整数的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
