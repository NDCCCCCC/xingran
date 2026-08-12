package system

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
	"gorm.io/gorm"
)

// UserHandler 用户处理器
type UserHandler struct {
	service           systemServices.UserService
	userADSyncService *addomain.UserADSyncService
	core              *core.Core
	// adSyncInFlight P0 #10: 按用户 ID 去重 AD 同步，避免短时间内
	// 多次更新同一用户触发并发 LDAP 连接。sync.Map 零值可用，无需初始化。
	adSyncInFlight sync.Map
}

// NewUserHandler 创建用户处理器实例
// 注：core 参数为 Phase 34 操作日志全模块覆盖新增，用于 operlog.Record 访问
// core.OperLogService 与 core.GetDB()。为保持向后兼容，core 以可变参形式追加在
// 末尾（现有 NewUserHandler(service) / NewUserHandler(service, adSync) 调用点无需修改）。
func NewUserHandler(service systemServices.UserService, userADSyncService ...*addomain.UserADSyncService) *UserHandler {
	handler := &UserHandler{service: service}
	if len(userADSyncService) > 0 && userADSyncService[0] != nil {
		handler.userADSyncService = userADSyncService[0]
	}
	return handler
}

// WithCore 注入 core 依赖（操作日志记录所需）。返回 receiver 自身以支持链式调用。
// 之所以单独提供此方法而非改写 NewUserHandler 签名，是为了不破坏既有可变参构造器
// 的所有调用点（含 NewUserHandler(service) 单参形态）。
func (h *UserHandler) WithCore(core *core.Core) *UserHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// Create 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body requests.UserCreateRequest true "用户信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /system/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req requests.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.Create(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeCreate)
	response.Success(c, gin.H{"message": "创建成功"})
}

// List 查询用户列表（类型安全版本）
// @Summary 查询用户列表
// @Description 分页查询用户列表，支持多条件过滤
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body requests.UserListParams true "查询条件"
// @Success 200 {object} response.Response{data=response.PageResponse}
// @Router /system/users/list [post]
func (h *UserHandler) List(c *gin.Context) {
	// 直接绑定到类型安全的请求结构体
	var params requests.UserListParams
	if err := c.ShouldBindJSON(&params); err != nil {
		// 如果是空请求体或解析失败，使用默认值
		params = requests.DefaultUserListParams()
	}

	// 应用分页限制
	current, pageSize := params.GetPagination()
	params.Current = current
	params.PageSize = pageSize

	result, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Page(c, result.List, result.Total, result.Current, result.PageSize)
}

// Statistics 用户统计(总数/正常/停用)
// @Summary 用户统计
// @Description 返回用户总数及启停状态计数,供统计卡片使用;用 COUNT 聚合而非加载全量行
// @Tags 用户管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /system/users/statistics [post]
func (h *UserHandler) Statistics(c *gin.Context) {
	result, err := h.service.Statistics(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// GetByID 获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/users/:id [post]
func (h *UserHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, user)
}

// Update 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body requests.UserUpdateRequest true "用户信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /system/users/:id/update [post]
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req requests.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	req.ID = id
	if err := h.service.Update(c.Request.Context(), &req); err != nil {
		response.Error(c, err)
		return
	}

	// 系统更新成功后，异步同步到AD域控（降级处理：不阻断系统更新）
	// P0 #10: 在原有 panic recover 基础上增加 (1) dedupe 防止短时间内对同一
	// 用户的多次更新触发并发 LDAP 连接；(2) 有限重试 + 指数退避，吸收 AD
	// 连接/网络的瞬时抖动。SyncUserUpdateToAD 每次调用都新建 LDAP 连接，
	// 故重试在 handler 层做最稳妥。
	if h.userADSyncService != nil {
		adSyncReq := h.buildADSyncMap(&req)
		// dedupe：若该用户的同步仍在飞行中，跳过本次重复触发
		if _, inFlight := h.adSyncInFlight.LoadOrStore(id, struct{}{}); inFlight {
			applogger.Infof("[AD-SYNC] 用户 %s 的 AD 同步仍在进行中，跳过本次重复触发", id)
		} else {
			go func() {
				defer func() {
					h.adSyncInFlight.Delete(id) // 释放去重标记（无论成功/失败/panic）
					if r := recover(); r != nil {
						applogger.Errorf("[AD-SYNC] SyncUserUpdateToAD panic 已恢复 [ID=%s]: panic=%v", id, r)
					}
				}()
				ctx := context.Background()
				const maxRetries = 3
				var lastErr error
				for attempt := 0; attempt < maxRetries; attempt++ {
					if attempt > 0 {
						// 指数退避：1s, 2s（仅阻塞本 goroutine,不影响请求）
						time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
					}
					err := h.userADSyncService.SyncUserUpdateToAD(ctx, id, adSyncReq)
					if err == nil {
						lastErr = nil
						break
					}
					lastErr = err
					applogger.Warnf("[AD-SYNC] 同步失败 [ID=%s] 第%d/%d次: %v", id, attempt+1, maxRetries, err)
					// Fix 2 (debug session ad-update-attr-no-such-object):
					// 目标 DN 在 AD 端不存在（LDAP code 32）属于"对象不存在"语义,
					// 重试无意义且会让 FailoverClient.MarkFailure 累加触发
					// 应用层 breaker 熔断 30 分钟 → 用户看到"管理员账号被锁"。
					// 短路 break,避免放大。
					if errors.Is(err, addomain.ErrADTargetNotFound) {
						applogger.Infof("[AD-SYNC] 目标 DN 在 AD 端不存在,短路重试 [ID=%s]", id)
						break
					}
					continue
				}
				if lastErr != nil && !errors.Is(lastErr, addomain.ErrADTargetNotFound) {
					applogger.Errorf("[AD-SYNC] 同步用户到AD最终失败（已重试%d次）[ID=%s]: %v", maxRetries, id, lastErr)
				}
			}()
		}
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeUpdate)
	response.Success(c, gin.H{"message": "更新成功"})
}

// buildADSyncMap 将UserUpdateRequest转换为AD同步用的map
func (h *UserHandler) buildADSyncMap(req *requests.UserUpdateRequest) map[string]interface{} {
	m := make(map[string]interface{})
	if req.Nickname != nil {
		m["nickname"] = *req.Nickname
	}
	if req.Email != nil {
		m["email"] = *req.Email
	}
	if req.Phone != nil {
		m["phone"] = *req.Phone
	}
	if req.DeptID != nil {
		m["deptId"] = *req.DeptID
	}
	return m
}

// Delete 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /system/users/:id/delete [post]
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeDelete)
	response.Success(c, nil)
}

// BatchDelete 批量删除用户
// @Summary 批量删除用户
// @Description 批量删除多个用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body object{ids=[]string} true "用户ID列表"
// @Success 200 {object} response.Response
// @Router /system/users/batch [post]
func (h *UserHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeBatch)
	response.Success(c, nil)
}

// UpdateStatus 更新用户状态
// @Summary 更新用户状态
// @Description 更新用户的启用/禁用状态
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Param request body object{status=int} true "状态"
// @Success 200 {object} response.Response
// @Router /system/users/:id/status [post]
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	var req struct {
		Status int `json:"status" binding:"min=0,max=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Error(c, err)
		return
	}

	operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeStatus)
	response.Success(c, gin.H{"message": "状态更新成功"})
}

// ResetPassword 重置用户密码
// @Summary 重置用户密码
// @Description 管理员重置指定用户的密码。优先读取 sys_config 中
// @Description sys.user.default_reset_password 的配置值；未配置或为空时
// @Description 自动生成 12 位强密码（含大小写字母与数字），仅写入审计日志。
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Router /system/users/:id/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("用户ID"))
		return
	}

	ctx := c.Request.Context()
	defaultPassword, generated, err := h.resolveResetPassword(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.service.ResetPassword(ctx, id, defaultPassword); err != nil {
		response.Error(c, err)
		return
	}

	// 审计日志：密码为自动生成时记录原值（管理员需告知用户），固定配置值
	// 时不记录原值（配置值本身可审计，避免日志膨胀）。F-BE-50 已确保响应
	// 不返回密码字段，原始明文仅留此一条审计行。
	if generated {
		applogger.Warnf("[AUDIT] 用户密码重置 (随机生成) [userID=%s, password=%s, operator=%s]",
			id, defaultPassword, clientIP(c))
	} else {
		applogger.Infof("[AUDIT] 用户密码重置 (使用 sys_config 默认值) [userID=%s, operator=%s]",
			id, clientIP(c))
	}

	// 敏感端点（密码重置）使用 RecordWithBody：读取+还原 c.Request.Body，
	// 并对 password 字段做遮蔽（FilterSensitiveParams）。即使当前 handler 不从
	// body 读密码（使用硬编码默认值），也按敏感路径记录以应对未来改造，且
	// 兼容 SM2+SM4 加密中间件（中间件先解密还原 body，本调用再读取+还原一次）。
	operlog.RecordWithBody(c, h.core.OperLogService, h.core.GetDB(), "用户管理", operlog.OperTypeReset)
	response.Success(c, gin.H{"message": "密码重置成功"})
}

// defaultResetPasswordConfigKey 是 sys_config 表中用于读取「默认重置密码」的配置项键名。
// 管理员可在「参数管理」中维护该键：值为空时回退到自动生成策略，避免继续使用弱默认值。
const defaultResetPasswordConfigKey = "sys.user.default_reset_password"

// generatedPasswordLength 缺省时自动生成的强密码长度。
const generatedPasswordLength = 12

// resolveResetPassword 决定本次重置要使用的明文密码。
// 返回值：
//   - password: 最终写入数据库的明文密码
//   - generated: true 表示由本方法自动生成（无 sys_config 配置或配置为空）；
//     false 表示使用 sys_config 中的固定值
//   - err: 查询/生成过程中的错误
//
// 实现策略：
//  1. 优先从 sys_config 读 sys.user.default_reset_password；
//  2. 配置缺失或为空字符串时，使用 crypto/rand 生成 12 位强密码；
//  3. 生成失败同样回退错误（不应静默使用弱密码）。
func (h *UserHandler) resolveResetPassword(ctx context.Context) (string, bool, error) {
	// 防御性处理：测试场景下 core 可能为 nil，直接走生成路径。
	if h.core == nil {
		pwd, err := generateStrongPassword(generatedPasswordLength)
		return pwd, true, err
	}

	var cfg models.Config
	db := h.core.GetDB()
	err := db.WithContext(ctx).Where("config_key = ?", defaultResetPasswordConfigKey).First(&cfg).Error
	if err == nil {
		value := strings.TrimSpace(cfg.ConfigValue)
		if value != "" {
			return value, false, nil
		}
		// 配置存在但值为空：视为未配置，继续走生成路径。
	} else if err != gorm.ErrRecordNotFound {
		// 真正的查询错误（非"记录不存在"）——为避免静默使用弱默认密码，
		// 将错误冒泡到 handler，让用户感知失败。
		return "", false, apperrors.Wrap(err, apperrors.CodeServerError, "读取默认密码配置失败")
	}

	pwd, genErr := generateStrongPassword(generatedPasswordLength)
	if genErr != nil {
		return "", false, apperrors.Wrap(genErr, apperrors.CodeServerError, "生成默认密码失败")
	}
	return pwd, true, nil
}

// generateStrongPassword 生成长度为 n 的密码，包含大小写字母与数字。
// 使用 crypto/rand 保证密码学强度，避免 math/rand 的可预测序列。
// 不使用易混淆字符（0/O、1/l/I）以提升人工抄录的可靠性。
func generateStrongPassword(n int) (string, error) {
	if n < 8 {
		n = 8 // 强密码下限保护
	}
	const (
		lower = "abcdefghijkmnpqrstuvwxyz" // 移除 l、o
		upper = "ABCDEFGHJKLMNPQRSTUVWXYZ" // 移除 I、O
		digit = "23456789"                 // 移除 0、1
		all   = lower + upper + digit
	)

	// 1) 从每个字符集各取 1 个，保证至少含一个小写、大写、数字
	must := []byte{
		pickRandom(lower),
		pickRandom(upper),
		pickRandom(digit),
	}

	// 2) 余下 n-3 位从全集随机填充
	rest := make([]byte, n-len(must))
	for i := range rest {
		rest[i] = pickRandom(all)
	}

	// 3) Fisher–Yates 打乱 must+rest，避免固定前缀
	buf := append(must, rest...)
	for i := len(buf) - 1; i > 0; i-- {
		j, err := randIntIntn(i + 1)
		if err != nil {
			return "", err
		}
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf), nil
}

// pickRandom 从字符集中取一个随机字节。
func pickRandom(charset string) byte {
	n, err := randIntIntn(len(charset))
	if err != nil {
		// crypto/rand 几乎不会失败；失败时回退到首位字符以保证可用性
		return charset[0]
	}
	return charset[n]
}

// randIntIntn 等价于 rand.Int(rand.Reader, big.NewInt(int64(n)))，但避免每次分配。
func randIntIntn(n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}
	max := big.NewInt(int64(n))
	r, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return int(r.Int64()), nil
}

// clientIP 提取客户端 IP，仅用于审计日志字段填充，不参与鉴权。
// 复用 Gin's c.ClientIP()，已处理 X-Forwarded-For 解析。
func clientIP(c *gin.Context) string {
	if c == nil {
		return "unknown"
	}
	return c.ClientIP()
}
