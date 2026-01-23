package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils"
)

const (
	OperTypeOther   = 0 // 其他
	OperTypeCreate  = 1 // 新增
	OperTypeUpdate  = 2 // 修改
	OperTypeDelete  = 3 // 删除
	OperTypeGrant   = 4 // 授权
	OperTypeExport  = 5 // 导出
	OperTypeImport  = 6 // 导入
	OperTypeForce   = 7 // 强退
	OperTypeGenCode = 8 // 生成代码
	OperTypeClean   = 9 // 清空数据
)

// recordOperLog 记录操作日志
func recordOperLog(c *gin.Context, core *core.Core, module string, operType int) {
	clientIP := utils.GetClientIP(c)
	operURL := c.Request.URL.String()
	core.OperLogService.RecordAsync(
		core.GetDB(),
		module,
		operType,
		"",
		c.Request.Method,
		operURL,
		utils.GetUsernamePtr(c),
		utils.GetDeptNameFromDB(c, core.GetDB()),
		&clientIP,
		nil,
		nil,
		nil,
		0,
		0,
	)
}
