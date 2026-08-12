package templates

import (
	"embed"
	"io/fs"
)

// 嵌入所有 TextFSM 模板（含 .textfsm 和 lldp/ 子目录）。
//
// 历史背景：templates/ 原本在项目根目录，运行时用 os.ReadFile + findProjectRoot(go.mod)
// 加载。生产部署到 /app/szh/ 后没有 templates/，所有依赖模板的 cron 任务失败
// （参见 /app/szh/logs 中 "no such file or directory"）。
//
// 改用 go:embed 后 binary 自包含，部署无需再同步 templates/ 目录。
//
// 排除：templates/samples/（测试用 fixtures，含真实设备 IP/hostname/serial，
// 不嵌入生产 binary）。
//
//go:embed all:embedded/templates
var embeddedTemplates embed.FS

// embeddedTemplatesRoot 返回嵌入的 templates FS，
// 路径前缀为 "embedded/templates/"。
func embeddedTemplatesRoot() fs.FS {
	return embeddedTemplates
}
