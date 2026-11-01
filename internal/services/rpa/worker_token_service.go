package rpa

// DEBT-03（v1.51）：RPA Worker API Token 认证 —— Token 服务层。
//
// 方案（docs/RPA-Worker-API认证方案-待办.md，Phase 35 搁置、v1.51 DEBT-03 落地）：
//   - Provisioning Token：预共享白名单（configs yaml + env 注入），仅注册接口使用，
//     命中后换发 Worker Token；
//   - Worker Token：`wk_` 前缀 + 32 字节 hex 随机串，落库 sys_rpa_workers.token，
//     带 TokenExpireDays（默认 90 天）过期时间；心跳/进度等运行期接口携带
//     X-Worker-Token，由 pkg/middleware.WorkerAuth 校验。
//
// 向后兼容宽限期：core.Config.RPA.Worker.Auth.AuthRequired=false（默认）时，
// 中间件对无 Token 请求放行（行为与 DEBT-03 之前一致），仅记录 WARN。

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
)

// DefaultWorkerTokenExpireDays Worker Token 默认有效期（天）。
const DefaultWorkerTokenExpireDays = 90

// workerTokenPrefix Worker Token 前缀（便于日志/审计中与其它凭证形态区分）。
const workerTokenPrefix = "wk_"

// WorkerTokenPair 换发给 Worker 的凭证（register 响应体 data 的一部分）。
type WorkerTokenPair struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// WorkerClaims 校验通过后的 Worker 身份（中间件据此设置 gin 上下文）。
type WorkerClaims struct {
	WorkerID   string
	WorkerName string
	ExpiresAt  time.Time
}

// WorkerTokenService Worker Token 换发与校验。
type WorkerTokenService interface {
	// ValidateProvisioningToken 校验预共享注册令牌（constant-time 比较）。
	ValidateProvisioningToken(provToken string) bool
	// IssueWorkerToken 为已注册 worker 生成（或轮换）Worker Token。
	// 宽限期模式下 worker 行不存在则返回 (nil, nil)——调用方决定是否降级放行。
	IssueWorkerToken(ctx context.Context, workerID string) (*WorkerTokenPair, error)
	// ValidateWorkerToken 校验 Worker Token：存在、匹配、未过期。
	// token 无效返回 error；宽限期语义（空 token）由中间件在调用前处理。
	ValidateWorkerToken(ctx context.Context, token string) (*WorkerClaims, error)
	// RevokeWorkerToken 撤销指定 Worker 的 Token（置空 + 过期时间清零）。
	RevokeWorkerToken(ctx context.Context, workerID string) error
}

type workerTokenServiceImpl struct {
	db        *gorm.DB
	provToken string // config 里白名单（部署态通常 1-2 个；constant-time 逐个比较）
	expireDays int
}

// NewWorkerTokenService 创建 Worker Token 服务。provTokens 为空 = 未配置认证，
// 所有校验返回 false（配合宽限期中间件 = 保持旧行为）。
func NewWorkerTokenService(db *gorm.DB, auth config.RPAWorkerAuthConfig) WorkerTokenService {
	expireDays := auth.TokenExpireDays
	if expireDays <= 0 {
		expireDays = DefaultWorkerTokenExpireDays
	}
	return &workerTokenServiceImpl{
		db:         db,
		provToken:  strings.Join(auth.ProvisioningTokens, "\x00"),
		expireDays: expireDays,
	}
}

// ValidateProvisioningToken constant-time 白名单比较。
// 白名单用 \x00 join 成单串后整体比较每个候选，避免 map 查找的时序泄露。
func (s *workerTokenServiceImpl) ValidateProvisioningToken(provToken string) bool {
	if provToken == "" || s.provToken == "" {
		return false
	}
	for _, candidate := range strings.Split(s.provToken, "\x00") {
		if candidate == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(provToken)) == 1 {
			return true
		}
	}
	return false
}

// generateWorkerToken 生成 wk_ 前缀随机 Token（32 字节 hex，碰撞概率可忽略，
// 因此模型层不用唯一索引——宽限期下多行空 token 会撞唯一约束）。
func generateWorkerToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成 Worker Token 随机数失败: %w", err)
	}
	return workerTokenPrefix + hex.EncodeToString(buf), nil
}

// IssueWorkerToken 为 worker 生成/轮换 Token（upsert 语义：worker 行不存在则先建）。
func (s *workerTokenServiceImpl) IssueWorkerToken(ctx context.Context, workerID string) (*WorkerTokenPair, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库不可用")
	}
	token, err := generateWorkerToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().AddDate(0, 0, s.expireDays)
	now := time.Now()

	var worker rpamodels.Worker
	err = s.db.WithContext(ctx).Where("worker_id = ?", workerID).First(&worker).Error
	switch {
	case err == nil:
		if err := s.db.WithContext(ctx).Model(&worker).Updates(map[string]interface{}{
			"token":               token,
			"token_expires_at":    expiresAt,
			"last_token_rotation": now,
		}).Error; err != nil {
			return nil, fmt.Errorf("写入 Worker Token 失败: %w", err)
		}
	default:
		// 宽限期模式：worker 行不存在（WorkerService.Register 未先建行）→ 先建最小行
		worker = rpamodels.Worker{
			WorkerID:          workerID,
			WorkerName:        workerID,
			Status:            rpamodels.WorkerStatusOnline,
			Token:             token,
			TokenExpiresAt:    &expiresAt,
			LastTokenRotation: &now,
		}
		if err := s.db.WithContext(ctx).Create(&worker).Error; err != nil {
			return nil, fmt.Errorf("创建 Worker 行并写入 Token 失败: %w", err)
		}
	}
	applogger.Infof("[RPA-WORKER-AUTH] Worker %s 的 API Token 已换发，有效期至 %s", workerID, expiresAt.Format(time.RFC3339))
	return &WorkerTokenPair{Token: token, ExpiresAt: expiresAt}, nil
}

// ValidateWorkerToken 校验运行期 Token。
func (s *workerTokenServiceImpl) ValidateWorkerToken(ctx context.Context, token string) (*WorkerClaims, error) {
	if token == "" || !strings.HasPrefix(token, workerTokenPrefix) {
		return nil, fmt.Errorf("无效的 Worker Token")
	}
	var worker rpamodels.Worker
	if err := s.db.WithContext(ctx).
		Where("token = ? AND deleted_at IS NULL", token).
		First(&worker).Error; err != nil {
		return nil, fmt.Errorf("Worker Token 不存在或不匹配")
	}
	if worker.TokenExpiresAt != nil && worker.TokenExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("Worker Token 已过期（%s），请重新注册轮换", worker.TokenExpiresAt.Format(time.RFC3339))
	}
	return &WorkerClaims{
		WorkerID:   worker.WorkerID,
		WorkerName: worker.WorkerName,
		ExpiresAt:  func() time.Time { if worker.TokenExpiresAt != nil { return *worker.TokenExpiresAt }; return time.Time{} }(),
	}, nil
}

// RevokeWorkerToken 撤销 Token（置空 + 清过期时间；下次注册重新换发）。
func (s *workerTokenServiceImpl) RevokeWorkerToken(ctx context.Context, workerID string) error {
	if s.db == nil {
		return fmt.Errorf("数据库不可用")
	}
	return s.db.WithContext(ctx).Model(&rpamodels.Worker{}).
		Where("worker_id = ?", workerID).
		Updates(map[string]interface{}{"token": "", "token_expires_at": nil}).Error
}
