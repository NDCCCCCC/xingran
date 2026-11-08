package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"github.com/xuri/excelize/v2"
)

// ExportUser 导出用户 Excel。
//
// FIX-03（v1.53 Phase 136）：把 service 层既有 ExportData("user") 能力接线到
// POST /system/users/export。导出字段与导入模板同源——二者都取
// excel_config.go 的 "user" 配置（GetExcelConfig），不另立一套列定义。
//
// 权限：由 router.go 的 usersExport 兄弟写组挂 RequireStrictPermissions(
// system:user:export)，严格单码、不做降级；本 handler 不重复鉴权。
//
// operlog：OperType=Export(5)，只记录请求参数摘要（不记录导出的用户行数据，
// 避免信息泄露），在写文件流（成功响应）之前调用。
//
// @Summary 导出用户
// @Description 按筛选条件导出用户列表 Excel，字段与用户导入模板一致
// @Tags 用户管理
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body map[string]any false "导出筛选条件（可选）"
// @Success 200 {file} file "xlsx 文件"
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/users/export [post]
func (h *UserHandler) ExportUser(c *gin.Context) {
	// 与 operations.exportData 一致：动态参数、允许空请求体（导出全量）。
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	// 复用用户导入同一个 ExcelService 构造器（cacheProvider / geocoding=nil）。
	// ExcelService.ExportData 对 "user" 走 legacy 分支：GetExcelConfig("user")
	// 与 GenerateTemplate("user") 同源，导出列 = 导入模板列。
	file, err := h.buildExcelService().ExportData(c.Request.Context(), "user", req)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	defer file.Close()

	// 操作日志：导出属于业务写操作（审计"谁在何时导出过哪份数据"）。
	// 只记 module + 参数摘要，绝不记录具体用户行；放在成功响应写出之前。
	if paramBytes, marshalErr := json.Marshal(req); marshalErr == nil {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeExport,
			operlog.WithOperParam(operlog.FilterSensitiveParams(string(paramBytes))))
	} else {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeExport)
	}

	if err := writeUserExportResponse(c, file); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
	}
}

// writeUserExportResponse 把 excelize 文件写成 xlsx 下载响应。
// 与 user_import_handler.go 的 DownloadImportTemplate / operations.writeExcelResponse
// 同款（c.Data 二进制流；不是 response.Success 包装，因为 body 是文件而非 JSON）。
func writeUserExportResponse(c *gin.Context, file *excelize.File) error {
	var buf bytes.Buffer
	if _, err := file.WriteTo(&buf); err != nil {
		return fmt.Errorf("生成用户导出文件失败: %w", err)
	}

	filename := fmt.Sprintf("user_export_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
	return nil
}
