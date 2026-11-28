package migrations

import (
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate259BrandConfigSeed 站点品牌信息 seed 到 sys_config（brand.* 命名空间）
//
// 背景:
//   - 登录页/侧边栏的站名、宣传语、版权等品牌文案曾硬编码在前端源码里，
//     改动曾因 skip-worktree 标志无法提交、又被环境重置"恢复"旧文案。
//   - 本迁移把品牌文案收敛为数据库可配置项（系统设置页"站点品牌"tab 可改），
//     前端只保留同名 fallback 常量（后端不可达时兜底），代码层不再承载品牌事实。
//
// 键位约定（config_key 前缀 brand.）:
//
//	brand.site_name      站点名称（侧边栏字标 / 登录页 wordmark / document.title）
//	brand.site_mark      Logo 方块字（未上传 logo 图时显示的单字）
//	brand.logo_url       Logo 图片 URL（上传后由 branding handler 写入，空=未上传）
//	brand.login_tagline  登录页宣传语
//	brand.copyright      登录页版权行
//	brand.site_subtitle  登录表单副标题
//
// IsSystem=1 防止在参数管理页被误删；ConfigType=N 前端可编辑。
// 幂等：已存在的 config_key 跳过，不覆盖管理员已改的值。
func Migrate259BrandConfigSeed(db *gorm.DB) error {
	log.Println("Running migration 259: brand config seeds")

	configs := []struct {
		configName  string
		configKey   string
		configValue string
		remark      string
	}{
		{
			configName:  "品牌-站点名称",
			configKey:   "brand.site_name",
			configValue: "数字化平台",
			remark:      "显示在侧边栏字标、登录页、浏览器标签页标题",
		},
		{
			configName:  "品牌-Logo方块字",
			configKey:   "brand.site_mark",
			configValue: "数",
			remark:      "Logo 位置显示的单个汉字；上传 Logo 图片后不再显示",
		},
		{
			configName:  "品牌-Logo图片",
			configKey:   "brand.logo_url",
			configValue: "",
			remark:      "Logo 图片访问 URL；通过系统设置-站点品牌上传后自动写入",
		},
		{
			configName:  "品牌-登录页宣传语",
			configKey:   "brand.login_tagline",
			configValue: "以国密算法为基座,融合自动化与 AI 调度,让每一次资源调度都可观测、可审计、可信赖。",
			remark:      "登录页左侧品牌面板的宣传语",
		},
		{
			configName:  "品牌-版权信息",
			configKey:   "brand.copyright",
			configValue: "© 2026 数字化平台",
			remark:      "登录页页脚版权行",
		},
		{
			configName:  "品牌-登录副标题",
			configKey:   "brand.site_subtitle",
			configValue: "请使用您的账号登录以访问运维控制台",
			remark:      "登录表单标题下方的副标题",
		},
	}

	for _, c := range configs {
		var existingCount int64
		if err := db.Model(&models.Config{}).Where("config_key = ?", c.configKey).Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			continue
		}
		cfg := &models.Config{
			ConfigName:  c.configName,
			ConfigKey:   c.configKey,
			ConfigValue: c.configValue,
			ConfigType:  models.ConfigTypeNo,      // N = 前端"站点品牌"页可编辑
			IsSystem:    models.ConfigIsSystemYes, // 防误删
			Remark:      c.remark,
		}
		if err := db.Create(cfg).Error; err != nil {
			log.Printf("Migration 259: create config %s failed: %v", c.configKey, err)
			continue
		}
		applogger.Infof("[迁移] sys_config brand seed: %s = %s", c.configKey, c.configValue)
	}

	log.Println("Migration 259 completed: brand configs seeded")
	return nil
}
