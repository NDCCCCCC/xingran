package dbbackup

import (
	"context"
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ---------------------------------------------------------------------------
// 恢复任务收尾一致性：进程内菜单/权限缓存清理 + 已登录会话失效
// ---------------------------------------------------------------------------
//
// 整库恢复会用备份里的数据替换现库的 sys_user / sys_menu / sys_role /
// sys_role_menu / sys_user_role 等鉴权相关表。若不收口，进程内缓存（菜单树、
// 角色权限、用户详情）与「已经签发的 JWT」仍指向旧库状态，出现两类不一致：
//   - 缓存里还是旧菜单/旧权限，恢复后用户看到的仍是恢复前的界面与鉴权结果；
//   - 旧 JWT 里的用户 ID / 角色在恢复后的库里可能已不存在或被改权限，却仍被放行。
//
// 本项目的既有能力与边界（已逐项核对后采用对应方案）：
//   - 缓存：core.Cache（pkg/cache.Cache）提供 Keys/MDelete/FlushDB，
//     由 API 装配层注册到 RegisterRestoreCacheFlusher，恢复收尾时按模式删除
//     menu/role/user/permission 等前缀的键 → **真实清理进程内菜单/权限缓存**。
//   - 会话：TokenBlacklistService 只能按**单个已知 token** 拉黑
//     （AddToBlacklist(ctx, token, expiry)），JWT 为无状态签发、服务层拿不到全部
//     已登录 token，且 middleware 无 tokenVersion / 全局 epoch 校验 →
//     **不存在「让全部已登录会话失效」的统一入口**（core.Cache.FlushDB 清的是
//     token:blacklist 黑名单键，方向相反，只会更放行，不可用）。
//   - 因此对会话采用需求允许的最小实现：在**任务结果**里写明确「需重新登录」
//     （成功写 stage、失败写 error_message），前端在恢复任务终态常驻提示，
//     明确要求用户重新登录以让前端丢弃旧 token 并重建会话。禁止留空。

const (
	// restoreReloginNotice 恢复任务结果里的统一「需重新登录」提示文案。
	// 只陈述确定成立的结论，不在这里断言「缓存已清理」——缓存后端可能未注册/抖动
	// （收尾日志会单独说明缓存动作），避免结果里出现自相矛盾的措辞。
	restoreReloginNotice = "恢复已结束，所有已登录会话需重新登录"
	// stageRestoreDoneWithRelogin 恢复成功任务的 stage（TaskView.stage 直接透出给前端）。
	stageRestoreDoneWithRelogin = "完成（需重新登录）"
	// stageRestoreDone 普通任务成功 stage。
	stageRestoreDone = "完成"
)

// finalizeRestoreConsistency 恢复收尾的一致性动作，覆盖 success / failed / panic
// 三条路径（由 runTask 在正常收尾与 panic recover 分支各调用一次）。
//
// 不做任何会改变任务终态写法的操作（终态仍由 failTask/succeedTask 统一写），
// 只负责：① 清进程内菜单/权限缓存；② 记录会话失效结论日志。
// 缓存清理失败只告警、不阻断任务终态——恢复本身已经成功/失败，不能因为
// 缓存后端抖动把任务卡成「未知状态」；失败场景下前端仍会按「需重新登录」提示。
func (s *service) finalizeRestoreConsistency(ctx context.Context, taskID string) {
	flusher := currentRestoreCacheFlusher()
	if flusher == nil {
		applogger.Warnf(
			"[全库备份] 恢复收尾 (taskID=%s)：未注册缓存清理钩子，进程内菜单/权限缓存未清理；会话处置=%s",
			taskID, restoreReloginNotice)
		return
	}
	if err := flusher(ctx); err != nil {
		applogger.Warnf("[全库备份] 恢复收尾 (taskID=%s)：清理进程内菜单/权限缓存失败: %v；会话处置=%s",
			taskID, err, restoreReloginNotice)
		return
	}
	applogger.Infof("[全库备份] 恢复收尾 (taskID=%s)：已清理进程内菜单/权限缓存；会话处置=%s",
		taskID, restoreReloginNotice)
}

// withReloginNotice 给恢复任务的失败信息追加「需重新登录」提示。
// 失败信息本身（含 SQLite 离线命令等）保持完整，提示只作为追加段，便于前端
// 在 errorMessage 里同时展示失败原因与后续动作。
func withReloginNotice(message string) string {
	return fmt.Sprintf("%s；%s", message, restoreReloginNotice)
}
