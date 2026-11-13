package api

// restore_write_guard.go —— 整库恢复期写入阻断全局中间件（A8 / #87 配套）。
//
// dbbackup 包在 runRestore 存活区间内把包级 atomic.Bool 置 true，并暴露
// 只读函数 IsRestoring()（契约见 internal/services/dbbackup/maintenance.go）。
// 本中间件挂在 internal/api/router.go 的 SetupRouter 入口处，对所有进入
// /api/v1 的请求生效：维护模式为 true 时，拦截所有写请求，避免恢复过程中
// 并发写入被恢复动作覆盖、产生「写后即丢失」的数据不一致。
//
// 判定采用「默认阻断（fail-safe）」：
//   - 安全方法 GET / HEAD / OPTIONS 一律放行（含 dbbackup 的任务轮询
//     GET /system/db-backup/tasks/:id、能力预检、计划读取、备份下载）；
//   - 其余方法（POST/PUT/PATCH/DELETE）中，只有路由末段命中
//     restoreReadOnlySegments（本项目读端点也大量使用 POST /list、
//     POST /statistics、POST /:id 等）或 dbbackup 任务轮询前缀才放行；
//   - 其它一律 503 阻断。新增写端点无需登记即自动被挡，不会漏放。
//
// 放行面覆盖任务书要求：GET / 只读端点 + dbbackup 自身的任务轮询端点。
// 标志复位（runRestore defer restoring.Store(false)）后中间件自动放行，
// 无需重启或重新注册路由。

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xingran-next/xingran-go-backend/internal/services/dbbackup"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// restoreGuardMessage 是维护模式拒绝写请求时返回的统一提示文案。
// 单测与前端据此识别「恢复中」状态，修改文案需同步测试。
const restoreGuardMessage = "系统正在执行全库恢复，请稍后"

// restoreSafeMethods 恢复期无条件放行的方法（只读 / 预检）。
var restoreSafeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

// restoreReadOnlySegments 恢复期放行的「读语义」路由末段。
// 本项目 API 约定：写端点末段为 update/delete/create/import/export/enable/
// disable/generate/assign/status/... ；读端点末段为 list/statistics/:id/
// tree/all/enabled/options/detail/logs/history/... 。取注册路径
// c.FullPath() 的末段判定（含 :id 占位符），可稳定区分「POST /:id 查询」
// 与「POST /:id/update 写入」。认证类端点单列，保证恢复期会话可维持。
var restoreReadOnlySegments = map[string]bool{
	// 列表 / 统计 / 树 / 选项 / 详情 / 日志 / 历史等典型读语义
	"list": true, "statistics": true, "stats": true, "tree": true,
	"all": true, "enabled": true, "options": true, "detail": true,
	"logs": true, "history": true, "query": true, "search": true,
	"page": true, "get": true, "info": true, "count": true,
	"preview": true, "check": true, "capabilities": true,
	"current": true,
	// 注："permissions"/"menus" 不在末段白名单——角色授权（POST /role/:id/permissions）
	// 等写端点常以此命名，末段放行会静默豁免写操作。用户自视图读端点
	// （POST /system/my-menus/permissions）由下方精确前缀 restoreUserScopedPrefixes 放行。
	// 认证 / 会话维持
	"login": true, "logout": true, "captcha": true,
	"refresh-token": true, "refresh": true, "validate": true,
}

// restoreTaskPollPrefix dbbackup 自身任务轮询端点（含任意任务 ID）。
const restoreTaskPollPrefix = "/system/db-backup/tasks/"

// restoreUserScopedPrefixes 恢复期按「路径前缀」精确放行的用户自视图读端点。
// 这些端点必须放行（恢复期用户/前端需要读取自己的菜单与权限维持会话视图），
// 但其末段 "permissions" 又是常见写端点命名，不能进末段白名单，故按前缀收窄。
var restoreUserScopedPrefixes = []string{
	"/system/my-menus/", // SetupUserMenuRouter：当前用户菜单/权限读取（含 POST /permissions）
}

// guardRequestPath 返回用于判定的请求路径（去掉 /api/v1 前缀）。
// gin 路由匹配完成后 c.FullPath() 为注册路径（含 :id 占位符）；未匹配
// 路由时回退 URL.Path。做 /api/v1 后缀剥离以兼容子路径部署形态。
func guardRequestPath(c *gin.Context) string {
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	if idx := strings.LastIndex(path, "/api/v1"); idx >= 0 {
		path = path[idx+len("/api/v1"):]
	}
	return path
}

// isRestoreReadOnly 判断恢复期是否放行该请求（不改变「写」的默认阻断）。
func isRestoreReadOnly(c *gin.Context) bool {
	if restoreSafeMethods[c.Request.Method] {
		return true
	}
	path := guardRequestPath(c)
	// dbbackup 任务轮询（GET 已由方法放行，此处兼容其它轮询形态）。
	if strings.HasPrefix(path, restoreTaskPollPrefix) {
		return true
	}
	// 用户自视图读端点（精确前缀，末段白名单无法安全覆盖）。
	for _, prefix := range restoreUserScopedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	seg := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		seg = path[i+1:]
	}
	// 路径参数占位符末段（:id / :dictType / :key ...）表示「按标识读」。
	if strings.HasPrefix(seg, ":") {
		return true
	}
	return restoreReadOnlySegments[seg]
}

// RestoreWriteGuard 返回恢复期写入阻断中间件（生产入口）。
//
// 固定读取 dbbackup.IsRestoring()（atomic.Bool）：每次请求动态读取，因此恢复
// 结束、标志复位后无需任何额外操作即自动放行；不缓存状态、不改动 dbbackup 包。
// 阻断时返回 HTTP 503 + 统一响应包装（code=503），并 Abort 掉后续中间件链
// （JWT / 权限 / 加解密 / handler 均不再执行），避免恢复期对将被替换的
// 鉴权表做无意义查询。错误响应不经过响应加密中间件，前端可直接识别。
func RestoreWriteGuard() gin.HandlerFunc {
	return restoreWriteGuardWith(dbbackup.IsRestoring)
}

// restoreWriteGuardWith 是中间件的可注入内核：isRestoring 参数让单测能稳定
// 覆盖「维护中 / 空闲」两态，无需驱动真实 runRestore（那是 dbbackup 包
// maintenance_test.go 的职责），也不触碰 dbbackup 的私有 atomic.Bool。
// 生产路径由 RestoreWriteGuard 传入 dbbackup.IsRestoring，行为完全一致。
func restoreWriteGuardWith(isRestoring func() bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isRestoring() || isRestoreReadOnly(c) {
			c.Next()
			return
		}
		response.Error(c, &response.AppError{
			Code:       http.StatusServiceUnavailable,
			Message:    restoreGuardMessage,
			HTTPStatus: http.StatusServiceUnavailable,
		})
		c.Abort()
	}
}
