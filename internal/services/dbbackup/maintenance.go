package dbbackup

import "sync/atomic"

// ---------------------------------------------------------------------------
// 恢复期维护模式（供 A8 在 internal/api/router.go 挂全局中间件调用）
// ---------------------------------------------------------------------------
//
// 【接口契约 · 严格锁定，不得改名/改签名】
//
//	func IsRestoring() bool
//
//   - 包名 dbbackup，函数无参数、返回 bool，进程级包级函数（非方法、非接口成员）。
//   - A8 会在 internal/api/router.go 的全局中间件链里调用 dbbackup.IsRestoring()：
//     返回 true 时把非备份模块的写请求挡在门外（维护模式），避免恢复过程中
//     并发写入被恢复动作覆盖、产生「写后即丢失」的数据不一致。
//   - 语义：仅当「整库恢复任务」正在推进时为 true；空闲/导出任务/恢复已完成
//     （无论 success 还是 failed）一律为 false。
//   - 生命周期由 runRestore 独占管理：进入 runRestore 时置 true，所有退出路径
//     （成功、失败、panic）通过 defer 复位为 false。导出任务与其它模块不得改动。
//   - 进程内状态：用 atomic.Bool 实现，多 goroutine 读（HTTP 中间件）与写
//     （恢复任务）无数据竞争；不依赖任何外部存储，多实例部署时各实例各自判断。
//     由于恢复本身由 taskMu + 任务表活跃行做全局互斥，同一时刻只有一个实例
//     在跑恢复，故不存在「另一实例仍在写、本实例已复位」的窗口。
//   - 该标志只是「尽力而为的在线拦截」，不替代 SQLite 场景要求的停服离线恢复：
//     SQLite 整库替换必须停服后由离线 CLI 完成（见 restorer.go）。
var restoring atomic.Bool

// IsRestoring 报告当前进程是否处于「整库恢复进行中」维护模式。
//
// 这是 A8 依赖的**稳定接口契约**：全局中间件只应读取本函数，不应直接访问
// restoring 变量，也不应在别处置位/复位。返回 true 的区间严格等于一次
// runRestore 调用的存活区间（含其所有 return / panic 展开路径）。
func IsRestoring() bool {
	return restoring.Load()
}
