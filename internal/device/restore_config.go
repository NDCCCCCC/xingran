package device

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/constants"
)

// restore_config.go — DeviceExecutor.RestoreConfig（Phase 93，配置备份恢复的唯一
// 设备下发入口，D-02）。与 GetConfig 对称：pool.GetDevice 取 vendor → ExecuteCustom
// 执行；内部经 wrapper.SendConfigs 配置模式批量下发（scrapligo AcquirePriv 自动进
// 入配置模式），下发前基础清洗（D-03），末尾按 vendor 追加"退一级"命令（D-06，V7
// 教训禁用 end），逐行 fail-fast（D-12）。

// RestoreResult 结构化下发结果（D-07 device 层字段；hash 校验由 service 层负责）。
type RestoreResult struct {
	TotalLines int           // 清洗后待下发行数（含末尾退出命令）
	SentLines  int           // 成功下发行数（fail-fast 时为部分计数）
	FailedLine string        // 首个失败行的命令内容（空表示全部成功）
	Duration   time.Duration // 下发总耗时
}

// cleanConfigLines 基础清洗（D-03）：过滤空行与前缀 #/! 注释行（TrimSpace 后判
// 定），保留中段含 #/! 的行（华为 # 是段分隔符、description 内可含 !——整行过滤
// 会误删配置）。
func cleanConfigLines(config string) []string {
	lines := strings.Split(config, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

// exitConfigCommand 返回厂商对应的"退一级"配置模式命令（D-06）。
// 与 portcollection/vendor_port_template.go VendorExitViewCmd 实证对齐：
// 华为/H3C → quit；锐捷/迈普及思科系 → exit。禁用 end（V7 教训：end 直退
// privileged EXEC 破坏 scrapli priv 跟踪）。
func exitConfigCommand(vendor string) string {
	switch vendor {
	case "huawei", "h3c":
		return "quit"
	default: // ruijie / maipu / cisco 风格
		return "exit"
	}
}

// RestoreConfig 把备份的 running-config 内容下发到目标设备（D-02 唯一下发入口）。
//
// 流程：GetDevice 取 vendor → cleanConfigLines 基础清洗 → 追加厂商退出命令 →
// ExecuteCustom 内 wrapper.SendConfigs 配置模式批量下发（scrapligo 按 platform
// 自动 AcquirePriv 进入配置模式）→ 逐行 fail-fast。
//
// 失败语义（D-12）：任一行报错/Failed 即停止后续行，返回的 RestoreResult 携带
// SentLines 部分计数与 FailedLine 定位，供任务进度留痕与"恢复前备份"回退。
// 超时：constants.RestoreConfigExecTimeout（D-05，全量配置千行级远超单命令语义）。
func (e *DeviceExecutor) RestoreConfig(ctx context.Context, deviceID, config string) (*RestoreResult, error) {
	pool := e.scheduler.GetConnectionPool()
	dev, err := pool.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}

	lines := cleanConfigLines(config)
	if len(lines) == 0 {
		return nil, fmt.Errorf("备份内容无有效配置行")
	}
	fullCmds := append(lines, exitConfigCommand(string(dev.Vendor)))

	result := &RestoreResult{TotalLines: len(fullCmds)}
	startTime := time.Now()

	execErr := e.ExecuteCustom(ctx, deviceID, func(_ context.Context, pc *PooledConnection) error {
		wrapper := pc.GetWrapper()
		if wrapper == nil {
			return fmt.Errorf("连接 wrapper 不可用")
		}
		// 一次 AcquirePriv 批量下发 + scrapligo 原生 StopOnFailed（D-12 fail-fast）。
		// 中断时 responses 含失败行（Failed=true）且 error 为 nil（见方法注释）。
		responses, sendErr := wrapper.SendConfigsStopOnFailed(fullCmds)
		result.SentLines = len(responses)
		if sendErr != nil {
			if result.SentLines < len(fullCmds) {
				result.FailedLine = fullCmds[result.SentLines]
			}
			return sendErr
		}
		for i, r := range responses {
			if r != nil && r.Failed {
				result.SentLines = i
				result.FailedLine = fullCmds[i]
				return fmt.Errorf("配置行下发失败 (第 %d/%d 行): %s", i+1, len(fullCmds), fullCmds[i])
			}
		}
		return nil
	}, constants.RestoreConfigExecTimeout)

	result.Duration = time.Since(startTime)
	if execErr != nil {
		return result, execErr
	}
	result.SentLines = result.TotalLines
	return result, nil
}
