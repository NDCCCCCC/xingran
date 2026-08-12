package services

import (
	"context"
	"fmt"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MACHistoryMatViewService MAC 历史物化视图刷新服务
//
// Phase 15 PERF-02 (D-10/D-11 锁定):
//   - REFRESH MATERIALIZED VIEW CONCURRENTLY (需 UNIQUE 索引, 已由 15-01 保证)
//   - 部分失败容错: 单个 MV 刷新失败不中断后续 (D-11)
//   - 视图顺序硬编码 MV-01 → MV-02 → MV-03 → MV-04
type MACHistoryMatViewService interface {
	// RefreshAllMaterializedViews 刷新全部 4 个物化视图
	// 单个失败不阻断后续, 整体返回第一个错误供上层日志
	RefreshAllMaterializedViews(ctx context.Context) error
	// RefreshSingleMatView 刷新单个物化视图
	// name 必须在白名单内
	RefreshSingleMatView(ctx context.Context, name string) error
}

// matViewWhiteList 物化视图白名单 (CONCURRENTLY 强依赖 UNIQUE 索引)
var matViewWhiteList = map[string]struct{}{
	"mv_mac_port_latest":        {},
	"mv_mac_device_summary":     {},
	"mv_mac_long_occupancy_top": {},
	"mv_mac_port_daily_count":   {},
}

// matViewRefreshOrder 刷新顺序 (D-11 锁定)
var matViewRefreshOrder = []string{
	"mv_mac_port_latest",
	"mv_mac_device_summary",
	"mv_mac_long_occupancy_top",
	"mv_mac_port_daily_count",
}

type macHistoryMatViewServiceImpl struct {
	db *gorm.DB
}

// NewMACHistoryMatViewService 构造函数
func NewMACHistoryMatViewService(db *gorm.DB) MACHistoryMatViewService {
	return &macHistoryMatViewServiceImpl{db: db}
}

func (s *macHistoryMatViewServiceImpl) isPostgreSQL() bool {
	return s.db.Config.Dialector.Name() == "postgres"
}

func (s *macHistoryMatViewServiceImpl) RefreshSingleMatView(ctx context.Context, name string) error {
	if !s.isPostgreSQL() {
		applogger.Infof("[MatView] 跳过刷新 %s（非PostgreSQL数据库）", name)
		return nil
	}
	if _, ok := matViewWhiteList[name]; !ok {
		return fmt.Errorf("物化视图 %s 不在白名单中", name)
	}

	applogger.Infof("[MatView] 开始刷新 %s", name)
	sql := fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s", name)
	if err := s.db.Exec(sql).Error; err != nil {
		applogger.Errorf("[MatView] 刷新 %s 失败: %v", name, err)
		return fmt.Errorf("刷新物化视图 %s 失败: %w", name, err)
	}
	applogger.Infof("[MatView] 刷新 %s 成功", name)
	return nil
}

func (s *macHistoryMatViewServiceImpl) RefreshAllMaterializedViews(ctx context.Context) error {
	if !s.isPostgreSQL() {
		applogger.Infof("[MatView] 跳过批量刷新（非PostgreSQL数据库）")
		return nil
	}

	applogger.Infof("[MatView] 开始批量刷新 4 个物化视图")

	var (
		firstErr error
		success  int
		failed   int
	)

	for _, name := range matViewRefreshOrder {
		if err := s.RefreshSingleMatView(ctx, name); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			failed++
			continue
		}
		success++
	}

	applogger.Infof("[MatView] 批量刷新完成: 成功 %d 失败 %d (共 %d)", success, failed, len(matViewRefreshOrder))

	if firstErr != nil {
		return fmt.Errorf("物化视图刷新部分失败: %w", firstErr)
	}
	return nil
}
