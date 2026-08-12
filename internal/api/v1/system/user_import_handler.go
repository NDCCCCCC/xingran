package system

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// ImportUser 批量导入用户
//
// 通过 Excel 文件批量导入用户，以 username 为唯一标识：
//   - 新用户：创建（性别默认 2=保密，状态默认 0=启用）
//   - 已存在用户：更新部门/email/手机号/工号/昵称等信息
//
// 导入成功后异步触发 AD 域控同步（降级处理，失败不影响导入结果）。
//
// @Summary 批量导入用户
// @Description 通过 Excel 批量导入用户，以用户名为唯一标识，重复用户更新相关信息，导入后触发 AD 同步
// @Tags 用户管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/users/import [post]
func (h *UserHandler) ImportUser(c *gin.Context) {
	// 鉴权：获取当前操作者 ID（同时用于导入数据审计字段）
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, response.ErrUnauthorized)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.ErrBadRequest, "未找到上传文件")
		return
	}

	// 文件安全校验：扩展名 + 大小上限 + 内容魔数三重校验（与 operations 模块一致）
	if !isValidExcelFile(file.Filename) {
		response.Error(c, response.ErrBadRequest, "只支持 .xlsx 格式的Excel文件。如需导入 .xls 文件，请先用 Excel 或 WPS 另存为 .xlsx 格式")
		return
	}
	if file.Size > userExcelUploadSizeLimit {
		response.Error(c, response.ErrBadRequest, fmt.Sprintf("文件过大(%d 字节,上限 %d 字节)", file.Size, userExcelUploadSizeLimit))
		return
	}
	if err := verifyExcelMagicBytes(file); err != nil {
		response.Error(c, response.ErrBadRequest, "文件内容非有效的 .xlsx 格式")
		return
	}

	// 构造 ExcelService 并执行导入（entityType="user" 对应 excel_config.go 的 user 配置）
	excelService := h.buildExcelService()
	result, err := excelService.ImportData(c.Request.Context(), "user", file, userID.(string))
	if err != nil {
		// 记录详细错误：err.Error() 含 upsert/GORM 具体原因（如类型错误、约束冲突），
		// 便于诊断。避免 HTTP 访问日志里 request_body 的二进制内容淹没真正根因。
		applogger.Errorf("[USER-IMPORT] 用户导入失败 (file=%s, size=%d): %v", file.Filename, file.Size, err)
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	// 导入成功后异步触发 AD 域控同步（降级处理，不阻断导入响应）
	// AffectedKeys 为本次真正写入记录的 username 列表（ImportData 内部收集）
	if h.userADSyncService != nil && len(result.AffectedKeys) > 0 {
		h.triggerADSyncAfterImport(result.AffectedKeys)
	}

	// 操作日志：只记录 filename + 行数统计，绝不记录原始 Excel 行数据（信息泄露缓解）
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeImport,
		operlog.WithOperParam(fmt.Sprintf(`{"filename":%q,"size":%d,"inserted":%d,"updated":%d,"failed":%d}`,
			file.Filename, file.Size, result.Inserted, result.Updated, result.Failed)))

	response.Success(c, gin.H{
		"inserted": result.Inserted,
		"updated":  result.Updated,
		"failed":   result.Failed,
		"errors":   result.Errors,
	})
}

// buildExcelService 构造用于用户导入的 ExcelService。
// 复用 operations 模块的 ExcelService（含引用解析、批量 upsert、缓存失效），
// geocoding 传 nil（用户导入不需要地理编码）。
func (h *UserHandler) buildExcelService() *opsServices.ExcelService {
	cacheProvider := systemServices.NewCacheProvider(h.core.DataCacheService)
	return opsServices.NewExcelService(h.core.DB.GetDB(), h.core.PwdManager, cacheProvider, nil)
}

// triggerADSyncAfterImport 导入成功后异步触发 AD 域控批量同步。
//
// 性能：调 BatchSyncUsersToAD 复用单个已绑定 LDAP 连接 + errgroup 并发
// （MaxConcurrentADSync=3），2274 用户从 ~10 分钟降到 ~10 秒。取代原先的
// 「逐用户串行 + 每用户新建连接 + 重试3次」模式。
//
// 行为：
//   - 查有 ad_dn 的用户 ID（导入新增的本地用户无 ad_dn 跳过）
//   - BatchSyncUsersToAD 复用单连接批量同步（单失败不中断，收集到 Errors）
//   - 降级：任何同步失败仅记录日志，不影响导入结果
func (h *UserHandler) triggerADSyncAfterImport(usernames []string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				applogger.Errorf("[AD-SYNC] 导入后批量同步 panic 已恢复: %v", r)
			}
		}()

		ctx := context.Background()
		// 查有 ad_dn 的用户 ID（导入新增的本地用户无 ad_dn 跳过；BatchSyncUsersToAD
		// 内部也会过滤，这里提前过滤减少传入量）
		var userIDs []string
		if err := h.core.GetDB().WithContext(ctx).
			Model(&models.User{}).
			Where("username IN ?", usernames).
			Where("ad_dn IS NOT NULL AND ad_dn <> ''").
			Pluck("id", &userIDs).Error; err != nil {
			applogger.Errorf("[AD-SYNC] 查询导入用户失败: %v", err)
			return
		}
		if len(userIDs) == 0 {
			applogger.Infof("[AD-SYNC] 导入后无待同步用户（ad_dn 非空），跳过")
			return
		}

		applogger.Infof("[AD-SYNC] 导入后批量同步开始，待同步用户 %d 个", len(userIDs))
		result, err := h.userADSyncService.BatchSyncUsersToAD(ctx, userIDs)
		if err != nil {
			applogger.Errorf("[AD-SYNC] 批量同步失败: %v", err)
			return
		}
		applogger.Infof("[AD-SYNC] 导入后批量同步完成: total=%d synced=%d failed=%d",
			result.Total, result.Synced, result.Failed)
	}()
}

// userExcelUploadSizeLimit 用户导入 Excel 上传大小上限(50 MB)，防止超大文件 OOM。
// 与 operations 模块保持一致。
const userExcelUploadSizeLimit int64 = 50 * 1024 * 1024

// isValidExcelFile 验证文件扩展名为 .xlsx。
func isValidExcelFile(filename string) bool {
	return len(filename) >= 5 && filename[len(filename)-5:] == ".xlsx"
}

// DownloadImportTemplate 下载用户导入 Excel 模板。
// 复用 ExcelService.GenerateTemplate("user")，模板字段与 excel_config.go 的
// user 配置一致（用户名/昵称/工号/邮箱/手机号/所属部门/性别/状态/备注）。
//
// @Summary 下载用户导入模板
// @Description 生成并下载用户批量导入用的 Excel 模板
// @Tags 用户管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Router /system/users/import/template [get]
func (h *UserHandler) DownloadImportTemplate(c *gin.Context) {
	excelService := h.buildExcelService()
	file, err := excelService.GenerateTemplate("user")
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := file.WriteTo(&buf); err != nil {
		response.Error(c, response.ErrServerError, "生成模板失败")
		return
	}

	filename := fmt.Sprintf("user_template_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeDownload)
}

// verifyExcelMagicBytes 读取文件前 4 字节，验证 OOXML/ZIP 魔数 (PK\x03\x04)。
// 防止仅扩展名校验被改后缀绕过。调用后重置文件读位置，后续 excelize 仍能读取。
// 与 operations 模块的实现一致（本包内联以避免 api 包间依赖）。
func verifyExcelMagicBytes(fileHeader *multipart.FileHeader) error {
	f, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	head := make([]byte, 4)
	n, err := f.Read(head)
	if err != nil || n != 4 {
		return fmt.Errorf("读取文件头失败: %w", err)
	}
	// OOXML / ZIP magic bytes: PK\x03\x04
	if head[0] != 0x50 || head[1] != 0x4B || head[2] != 0x03 || head[3] != 0x04 {
		return fmt.Errorf("文件内容魔数错误: 期望 PK\\x03\\x04, 实际 % X", head)
	}
	return nil
}

// SyncManagers 手动触发"从部门负责人同步 AD manager 属性"全量任务。
//
// 行为：调用 UserADSyncService.SyncManagersToAD 全量同步所有有 ad_dn 的用户，
// 将其部门（含祖先递归）leader 的 ad_dn 写入 AD manager 属性。返回同步统计
// （总数/成功/跳过/失败）。单个用户失败不中断批量（降级记录到 errors）。
//
// 业务语义：AD 中 manager 是 DN 格式，故 leader 用户必须有 ad_dn；无 leader、
// leader 自指、leader 无 ad_dn 的用户均跳过。
//
// @Summary 同步用户经理到AD
// @Description 根据 sys_dept.leader 全量同步用户 AD manager 属性（部门无 leader 时递归父部门）
// @Tags 用户管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/users/sync-managers [post]
func (h *UserHandler) SyncManagers(c *gin.Context) {
	if h.userADSyncService == nil {
		response.Error(c, response.ErrServerError, "AD 同步服务未启用")
		return
	}

	result, err := h.userADSyncService.SyncManagersToAD(c.Request.Context(), nil)
	if err != nil {
		applogger.Errorf("[USER-MANAGER-SYNC] 同步失败: %v", err)
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	// 操作日志：仅记录聚合统计，不记录 errors 明细（可能含 username，避免信息泄露）
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeSync,
		operlog.WithOperParam(fmt.Sprintf(`{"total":%d,"synced":%d,"skipped":%d,"failed":%d}`,
			result.Total, result.Synced, result.Skipped, result.Failed)))

	response.Success(c, gin.H{
		"total":   result.Total,
		"synced":  result.Synced,
		"skipped": result.Skipped,
		"failed":  result.Failed,
		"errors":  result.Errors,
	})
}
