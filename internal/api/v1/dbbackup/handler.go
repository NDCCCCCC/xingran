package dbbackup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services/dbbackup"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// 全库备份模块 operlog 的中文模块名（契约第 8 节统一「数据备份」）。
const dbBackupModuleName = "数据备份"

const (
	// restoreUploadMaxSize 恢复上传 zip 大小上限（2GB），防止超大文件打满磁盘。
	restoreUploadMaxSize int64 = 2 << 30
	// planOperParamMaxBytes 计划保存 operlog 摘要上限（与 operlog.FilterSensitiveParams 的 8192 对齐）。
	// 敏感字段脱敏占位符复用服务层导出常量 dbbackup.PlanMaskPlaceholder（单一权威，避免双份漂移）。
	planOperParamMaxBytes = 8192
	// restoreUploadFilePattern os.CreateTemp 的模板，落在系统临时目录（os.TempDir），
	// 生成随机文件名 + .zip 后缀，不落仓库 data/ 目录。
	restoreUploadFilePattern = "dbbackup-restore-*.zip"
)

// DBBackupHandler 全库备份处理器。
// 持有 dbbackup.Service（业务逻辑）与 *core.Core（operlog 所需的 OperLogService/DB）。
type DBBackupHandler struct {
	service dbbackup.Service
	core    *core.Core
}

// record 统一封装 operlog.Record，集中模块名与 core 取值。
// operlog.Record 自身 panic-safe 且对 nil 依赖静默返回。
func (h *DBBackupHandler) record(c *gin.Context, operType int, opts ...operlog.RecordOption) {
	if h.core == nil {
		operlog.Record(c, nil, nil, dbBackupModuleName, operType, opts...)
		return
	}
	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), dbBackupModuleName, operType, opts...)
}

// List 备份记录分页（POST /list）。
// 解析 current/pageSize（默认 1/10），调 Service.List，返回 response.PageResponse。
func (h *DBBackupHandler) List(c *gin.Context) {
	current, pageSize, err := parsePagination(c)
	if err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	result, err := h.service.List(c.Request.Context(), current, pageSize)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	if result == nil {
		// 防御：服务层异常返回 nil,nil 时退化为空白页，避免空指针。
		result = &dbbackup.ListResult{List: []dbbackup.BackupRecord{}}
	}

	list := result.List
	if list == nil {
		// 保证前端拿到的 list 是 [] 而不是 null。
		list = []dbbackup.BackupRecord{}
	}
	response.Success(c, response.PageResponse{
		List:     list,
		Total:    result.Total,
		Current:  current,
		PageSize: pageSize,
	})
}

// Capabilities 能力预检（GET /capabilities）：返回当前部署环境 pg_dump /
// pg_restore 是否可用，供页面在触发导出前给出安装/配置提示。
func (h *DBBackupHandler) Capabilities(c *gin.Context) {
	cap, err := h.service.Capabilities(c.Request.Context())
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	response.Success(c, cap)
}

// Statistics 统计卡（POST /statistics）。
func (h *DBBackupHandler) Statistics(c *gin.Context) {
	stats, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	response.Success(c, stats)
}

// GetTask 异步任务状态轮询（GET /tasks/:id）。
func (h *DBBackupHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, response.ErrBadRequest, "任务ID不能为空")
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	response.Success(c, task)
}

// Download 下载备份 zip（GET /:id/download）。
// 用 c.File 输出二进制流，不套 response 包装；operlog 在 c.File 之前记录。
//
// ⚠️ 安全告警：备份包含全库敏感数据（用户口令散列、个人信息、业务数据等），
// 下载即等同于导出整库明文。此处显式留日志 + 前端下载按钮二次提示（见
// BackupListTab.handleDownload），下载端点 operlog 记录操作人与文件 ID 供追溯。
func (h *DBBackupHandler) Download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, response.ErrBadRequest, "备份ID不能为空")
		return
	}

	path, err := h.service.DownloadPath(c.Request.Context(), id)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	// 显式告警日志：备份包含全库敏感数据，落审计供事后追溯（不打印文件内容/路径以外信息）。
	applogger.Warnf("[全库备份] ⚠️ 备份包含全库敏感数据：下载备份 id=%s（操作人已由 operlog 记录）", id)
	// 契约第 8 节：下载端点 operlog.Record(OperTypeDownload) 必须在 c.File 之前。
	h.record(c, operlog.OperTypeDownload)
	c.File(path)
}

// Export 触发导出（POST /export），返回 {"taskId":"..."}。
// source 固定 "manual"（手动触发），operator 取登录用户 ID。
func (h *DBBackupHandler) Export(c *gin.Context) {
	operator := utils.GetUserID(c)

	// TriggerExport 立即返回 taskID，服务内部用独立 ctx 异步执行导出任务；
	// 这里传 c.Request.Context() 仅用于创建任务记录，后台任务不随 HTTP 请求结束而取消。
	taskID, err := h.service.TriggerExport(c.Request.Context(), dbbackup.SourceManual, operator)
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	// 业务写操作 success path：operlog 在 response.Success 之前。
	h.record(c, operlog.OperTypeExport)
	response.Success(c, gin.H{"taskId": taskID})
}

// Restore 上传 zip 并触发恢复（POST /restore，multipart），返回 {"taskId":"..."}。
//
// 临时文件生命周期（关键：避免异步竞态）：
//   - 校验扩展名 .zip 与 2GB 大小上限后，用 os.CreateTemp("", ...) 把上传流
//     同步写入**系统临时目录**（os.TempDir）。不落仓库 data/ 目录、不在仓库创建目录。
//   - TriggerRestore 是异步的：建 pending 任务后立刻返回，后台 goroutine 之后还要
//     读取该 zip。因此 handler **绝不能** defer os.Remove(tmpPath)，否则会出现
//     「文件已删除、异步任务读不到」的竞态。
//   - 采用契约「选一种」中的「服务自己负责清理」方案：上传文件路径的所有权在
//     TriggerRestore 成功返回后移交给服务层，由服务层在 restore 任务完成
//     （成功/失败）后删除该临时文件。handler 只在「同步写盘失败」或
//     「TriggerRestore 同步返回错误（尚未建立后台任务）」这两个分支删除半成品。
//   - 需 lead 与 Service 实现方（B 代理）对齐：TriggerRestore 的异步任务结束后
//     必须 os.Remove(uploadPath)。
func (h *DBBackupHandler) Restore(c *gin.Context) {
	operator := utils.GetUserID(c)

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.ErrBadRequest, "未找到上传文件")
		return
	}
	// 扩展名校验（大小写不敏感）。
	if !strings.EqualFold(filepath.Ext(file.Filename), ".zip") {
		response.Error(c, response.ErrBadRequest, "只支持 .zip 格式的备份文件")
		return
	}
	// 大小上限校验，防止超大上传打满磁盘。
	if file.Size > restoreUploadMaxSize {
		response.Error(c, response.ErrBadRequest,
			fmt.Sprintf("文件过大(%d 字节,上限 %d 字节)", file.Size, restoreUploadMaxSize))
		return
	}
	if file.Size <= 0 {
		response.Error(c, response.ErrBadRequest, "上传文件为空")
		return
	}

	// 落盘到系统临时目录（不写仓库 data/；该目录由 OS 管理、被 .gitignore 之外的运行期使用）。
	tmpFile, err := os.CreateTemp("", restoreUploadFilePattern)
	if err != nil {
		response.Error(c, response.ErrServerError, "创建临时文件失败: "+err.Error())
		return
	}
	tmpPath := tmpFile.Name()

	// 同步完成「打开 multipart → 流式拷贝 → Sync/关闭」，之后才交给异步服务读取。
	copyErr := func() error {
		defer tmpFile.Close()
		src, openErr := file.Open()
		if openErr != nil {
			return openErr
		}
		defer src.Close()
		if _, cpErr := io.Copy(tmpFile, src); cpErr != nil {
			return cpErr
		}
		return tmpFile.Sync()
	}()
	if copyErr != nil {
		// 同步阶段失败：异步任务尚未建立，可安全删除半成品。
		_ = os.Remove(tmpPath)
		response.Error(c, response.ErrServerError, "保存上传文件失败: "+copyErr.Error())
		return
	}

	// TriggerRestore 立即返回 taskID，后台 goroutine 会在之后读取 tmpPath。
	// 故此处不 defer 删除；仅当服务同步返回错误（未建立任务）时才清理。
	taskID, err := h.service.TriggerRestore(c.Request.Context(), tmpPath, operator)
	if err != nil {
		_ = os.Remove(tmpPath)
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	// 手工构造 oper_param（multipart body 不能进 operlog，参考 file_handler.go 上传端点）：
	// 只记录文件名与大小，绝不记录原始文件内容或任何凭据。
	operParam := fmt.Sprintf(`{"filename":%q,"size":%d}`, file.Filename, file.Size)
	h.record(c, operlog.OperTypeImport, operlog.WithOperParam(operParam))
	response.Success(c, gin.H{"taskId": taskID})
}

// Delete 删除备份记录与文件（POST /:id/delete）。
func (h *DBBackupHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, response.ErrBadRequest, "备份ID不能为空")
		return
	}

	operator := utils.GetUserID(c)
	if err := h.service.Delete(c.Request.Context(), id, operator); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	h.record(c, operlog.OperTypeDelete)
	response.Success(c, nil)
}

// GetPlan 读取备份计划（GET /plan）。
// 敏感字段脱敏：OSS AccessKey/SecretKey、FTP Password 一律返回 "******" 占位，
// 避免凭据经接口明文泄露；前端回传占位符时 SavePlan 会保留原值。
func (h *DBBackupHandler) GetPlan(c *gin.Context) {
	plan, err := h.service.GetPlan(c.Request.Context())
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	maskPlanSecrets(plan)
	response.Success(c, plan)
}

// SavePlan 保存备份计划（POST /plan）。
//
// 敏感字段若收到 "******" 占位（GetPlan 返回的脱敏值），说明用户未改动该凭据，
// 则读取当前计划中的原值回填，避免用占位符覆盖真实凭据。
// operlog：计划含凭据 → 用 RecordWithBody（对原始 body 自动脱敏）；
// 请求体超过 8192 字节时改用 WithOperParam 传已过滤的小 JSON（契约第 8 节）。
func (h *DBBackupHandler) SavePlan(c *gin.Context) {
	// 先读取原始请求体，供 JSON 绑定与 operlog 共用。
	// 本端点已在 security.request_encryption.exclude_paths 中豁免（契约第 9 节），
	// 故 body 为明文 JSON，可安全读/恢复。
	raw, err := c.GetRawData()
	if err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))

	var req dbbackup.Plan
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.ErrBadRequest, "请求参数错误")
		return
	}

	// 占位符回填：保留未改动的真实凭据。
	if err := h.mergePlanMaskedSecrets(c, &req); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	operator := utils.GetUserID(c)
	if err := h.service.SavePlan(c.Request.Context(), &req, operator); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}

	// 计划保存含凭据：默认 RecordWithBody（FilterSensitiveParams 会二次脱敏
	// accessKey/secretKey/password）；超大 body 用 WithOperParam 传脱敏后小 JSON。
	if len(raw) > planOperParamMaxBytes {
		h.record(c, operlog.OperTypeUpdate, operlog.WithOperParam(buildPlanOperParam(&req)))
	} else {
		// 恢复 body 供 RecordWithBody 的 GetRawData 读取。
		c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
		if h.core == nil {
			h.record(c, operlog.OperTypeUpdate)
		} else {
			operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(),
				dbBackupModuleName, operlog.OperTypeUpdate)
		}
	}
	response.Success(c, nil)
}

// parsePagination 解析分页参数 current/pageSize，默认 1/10；绑定失败返回 error。
// 兼容 body 为空（使用默认分页）与 JSON 数字/字符串两种形式。
func parsePagination(c *gin.Context) (int, int, error) {
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		// 空 body 视为使用默认分页；非空但非法 JSON 才是参数错误。
		if !errors.Is(err, io.EOF) {
			return 0, 0, err
		}
		raw = map[string]interface{}{}
	}

	current := parseIntField(raw, "current", 1)
	pageSize := parseIntField(raw, "pageSize", 10)
	if current < 1 {
		current = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	return current, pageSize, nil
}

// parseIntField 从 map 中取 int 字段，兼容 float64（JSON 数字）与字符串，缺省用 def。
func parseIntField(raw map[string]interface{}, key string, def int) int {
	v, ok := raw[key]
	if !ok || v == nil {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, e := n.Int64()
		if e == nil {
			return int(i)
		}
	case string:
		i, e := strconv.Atoi(strings.TrimSpace(n))
		if e == nil {
			return i
		}
	}
	return def
}

// maskPlanSecrets 把计划中的敏感凭据替换为占位符（GetPlan 出参脱敏、operlog 摘要共用）。
// 仅对非空值替换：空凭据保持空，前端才能区分「未配置」与「已配置（脱敏）」。
func maskPlanSecrets(p *dbbackup.Plan) {
	if p == nil {
		return
	}
	if p.OSS.AccessKey != "" {
		p.OSS.AccessKey = dbbackup.PlanMaskPlaceholder
	}
	if p.OSS.SecretKey != "" {
		p.OSS.SecretKey = dbbackup.PlanMaskPlaceholder
	}
	if p.FTP.Password != "" {
		p.FTP.Password = dbbackup.PlanMaskPlaceholder
	}
}

// mergePlanMaskedSecrets 处理 SavePlan 入参中的脱敏占位符与空串：
// 读取当前计划，用原凭据回填，避免占位符/空串覆盖真实值。
//
// 前后端约定（quick 261110-tbq 集成对齐）：GetPlan 出参把已存凭据脱敏为 "******"；
// 前端提交时把未改动的凭据统一改写为 ""（空串）。因此对敏感字段，
// "******" 与 "" 都表示「用户未改动，保留原值」——空串凭据本身无业务意义，
// 若当作「清空」反而会导致管理员一保存计划就丢失 OSS/FTP 凭据。
func (h *DBBackupHandler) mergePlanMaskedSecrets(c *gin.Context, p *dbbackup.Plan) error {
	// keep =「未改动」：占位符或空串（前端 261110-tbq 约定）。显式清空标记
	// dbbackup.CredentialClearMarker 不属于 keep，原样透传给服务层 savePlan
	// 置空落库，实现「管理员可显式删除已配置凭据」。
	keep := func(v string) bool { return v == dbbackup.PlanMaskPlaceholder || strings.TrimSpace(v) == "" }
	needMerge := keep(p.OSS.AccessKey) || keep(p.OSS.SecretKey) || keep(p.FTP.Password)
	if !needMerge {
		return nil
	}

	current, err := h.service.GetPlan(c.Request.Context())
	if err != nil {
		return fmt.Errorf("读取当前备份计划以回填脱敏凭据失败: %w", err)
	}
	if current == nil {
		return nil
	}
	if keep(p.OSS.AccessKey) {
		p.OSS.AccessKey = current.OSS.AccessKey
	}
	if keep(p.OSS.SecretKey) {
		p.OSS.SecretKey = current.OSS.SecretKey
	}
	if keep(p.FTP.Password) {
		p.FTP.Password = current.FTP.Password
	}
	return nil
}

// buildPlanOperParam 构造计划保存的 oper_param：脱敏凭据后的小 JSON。
// 绝不把明文凭据写进审计日志（契约第 8/9 节）。
func buildPlanOperParam(p *dbbackup.Plan) string {
	if p == nil {
		return "{}"
	}
	masked := *p // 值拷贝，避免修改即将落库的真实对象
	maskPlanSecrets(&masked)
	b, err := json.Marshal(&masked)
	if err != nil {
		return fmt.Sprintf(`{"storage":%q}`, string(p.Storage))
	}
	s := string(b)
	if len(s) > planOperParamMaxBytes {
		// 极端情况下再兜底：只保留非敏感元信息摘要。
		s = fmt.Sprintf(`{"enabled":%t,"cron":%q,"retention":%d,"storage":%q,"truncated":true}`,
			p.Enabled, p.Cron, p.Retention, string(p.Storage))
	}
	return s
}
