package system

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	opsServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/operations"
	systemServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/response"
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

// triggerADSyncAfterImport 导入成功后异步触发 AD 域控同步。
//
// 行为说明：
//   - 查询本次受影响用户名对应的用户记录
//   - 对每个用户调用 SyncUserUpdateToAD，带指数退避重试（吸收 LDAP/网络瞬时抖动）
//   - dedupe：跳过正在同步中的用户（与 Update handler 共享 adSyncInFlight）
//   - 降级：任何同步失败仅记录日志，不影响导入结果
//
// 业务语义：只有已有 AD DN 的用户（从 AD 同步过来的）才会真正推送；
// 导入新增的本地用户无 AD DN，SyncUserUpdateToAD 内部自动跳过。
func (h *UserHandler) triggerADSyncAfterImport(usernames []string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				applogger.Errorf("[AD-SYNC] 导入后同步 panic 已恢复: %v", r)
			}
		}()

		ctx := context.Background()
		var users []models.User
		if err := h.core.GetDB().WithContext(ctx).
			Where("username IN ?", usernames).
			Find(&users).Error; err != nil {
			applogger.Errorf("[AD-SYNC] 查询导入用户失败: %v", err)
			return
		}

		applogger.Infof("[AD-SYNC] 导入后同步开始，待同步用户 %d 个", len(users))
		synced, skipped := 0, 0
		for _, u := range users {
			// dedupe：该用户的同步仍在飞行中则跳过
			if _, inFlight := h.adSyncInFlight.LoadOrStore(u.ID, struct{}{}); inFlight {
				skipped++
				continue
			}
			if err := h.syncImportedUserWithRetry(ctx, u); err != nil {
				applogger.Warnf("[AD-SYNC] 用户 %s 同步失败: %v", u.Username, err)
			} else {
				synced++
			}
			h.adSyncInFlight.Delete(u.ID)
		}
		applogger.Infof("[AD-SYNC] 导入后同步完成：成功 %d，跳过 %d，失败 %d",
			synced, skipped, len(users)-synced-skipped)
	}()
}

// syncImportedUserWithRetry 同步单个导入用户到 AD，带指数退避重试。
// 构造 updateReq 时携带本次导入可能变更的字段（部门/email/手机号/昵称），
// SyncUserUpdateToAD 内部按需映射到 AD 属性并在部门变更时移动 OU。
func (h *UserHandler) syncImportedUserWithRetry(ctx context.Context, user models.User) error {
	adSyncReq := make(map[string]interface{})
	if user.DeptID != nil {
		adSyncReq["deptId"] = *user.DeptID
	}
	if user.Email != nil {
		adSyncReq["email"] = *user.Email
	}
	if user.Phone != nil {
		adSyncReq["phone"] = *user.Phone
	}
	if user.Nickname != nil {
		adSyncReq["nickname"] = *user.Nickname
	}

	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：1s, 2s（仅阻塞本 goroutine）
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}
		if err := h.userADSyncService.SyncUserUpdateToAD(ctx, user.ID, adSyncReq); err != nil {
			lastErr = err
			applogger.Warnf("[AD-SYNC] 同步失败 [user=%s] 第%d/%d次: %v", user.Username, attempt+1, maxRetries, err)
			continue
		}
		return nil
	}
	return lastErr
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
