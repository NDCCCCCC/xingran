package system

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// SettingsService 系统设置服务
type SettingsService interface {
	// GetUserPreferences 获取用户个人设置
	GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error)

	// UpdateUserPreferences 更新用户个人设置
	UpdateUserPreferences(ctx context.Context, userID string, req *UserPreferences) error
}

// settingsService 系统设置服务实现
//
// v1.22 收尾：移除对 sys.theme.default (默认主题) 的依赖（删除默认主题页面+API+表行后），
// 用户无偏好记录时直接返回硬编码默认值，不再注入 ConfigService。
type settingsService struct {
	db *gorm.DB
}

// NewSettingsService 创建系统设置服务实例
func NewSettingsService(db *gorm.DB) SettingsService {
	return &settingsService{db: db}
}

// UserPreference 用户个人设置模型（扩展版）
//
// 2026-08-17 (quick-260817-hfl): 模型本体迁移至 internal/models/user_preference.go
// (schema 单一事实源,sqlite 分支 AutoMigrate 注册需要;此处 alias 保持全部调用点兼容)。
type UserPreference = models.UserPreference

// ============================================================================
// 用户偏好允许集合与规范默认值 —— 与数据库 CHECK 约束逐字对齐的唯一事实源
//
// 数据库侧约束（PG 侧由归档 SQL 建立；SQLite 侧 GORM AutoMigrate 不生成 CHECK 约束）：
//
//	chk_theme_style    IN ('minimal','glassmorphism','neumorphism','flat2.0','luxury-quiet')
//	chk_layout_type    IN ('classic','hybrid','innovative')
//	chk_layout_density IN ('compact','comfortable','spacious')
//	chk_page_size      IN (5,10,20,50,100)
//
// 三方必须逐项一致：绑定标签（DTO tag） ↔ 服务层允许集合（本组定义） ↔ 数据库 CHECK 约束。
// 其中「绑定标签 ↔ 服务层允许集合」由
// TestSettingsService_BindingAllowListMatchesAllowedSets 用反射锁定，任一方漂移即测试失败。
//
// 背景：本组定义存在的理由是「应用层曾写出越界值」。前端自 Phase 65 移除多主题后不再发送
// themeStyle，后端 DTO 用普通 string 接收、omitempty 让省略字段留下空字符串，
// Update 分支的 map 更新把它当成「显式置空」写库，撞上 PG 的 chk_theme_style
// （SQLSTATE 23514）。约束本身是正确的，错误在应用层 —— 所以这里收紧应用层，
// 不放宽数据库约束。
// ============================================================================

// allowedThemeStyles 主题风格允许集（对应 chk_theme_style）。
var allowedThemeStyles = []string{"minimal", "glassmorphism", "neumorphism", "flat2.0", "luxury-quiet"}

// allowedLayoutTypes 布局类型允许集（对应 chk_layout_type）。
var allowedLayoutTypes = []string{"classic", "hybrid", "innovative"}

// allowedLayoutDensities 布局密度允许集（对应 chk_layout_density）。
var allowedLayoutDensities = []string{"compact", "comfortable", "spacious"}

// allowedPageSizes 每页条数允许集（对应 chk_page_size）。
var allowedPageSizes = []int{5, 10, 20, 50, 100}

// allowedThemes 主题明暗允许集。
var allowedThemes = []string{"light", "dark"}

// allowedLanguages 语言允许集。
var allowedLanguages = []string{"zh-CN", "en-US"}

// 规范默认值：既是无历史行时 GET 的返回值来源，也是「省略即不修改」语义下
// Create 分支（无原值可保留）的兜底值来源。buildDefaultPreferences 复用本组定义，
// 避免两处默认值各自硬编码造成漂移。
const (
	defaultTheme                 = "light"
	defaultThemeStyle            = "minimal"
	defaultLayoutType            = "classic"
	defaultLayoutDensity         = "comfortable"
	defaultSidebarWidth          = 280
	defaultSidebarCollapsedWidth = 64
	defaultPageSize              = 10
	defaultLanguage              = "zh-CN"
)

// UserPreferences 用户个人设置请求/响应DTO
//
// 绑定标签与数据库 CHECK 约束逐字对齐：越界值在 HTTP 层就被拒（400），
// 不再进入写库路径变成 500。
type UserPreferences struct {
	// 主题
	Theme      string `json:"theme" binding:"required,oneof=light dark"`
	ThemeStyle string `json:"themeStyle" binding:"omitempty,oneof=minimal glassmorphism neumorphism flat2.0 luxury-quiet"`

	// 布局
	LayoutType            string `json:"layoutType" binding:"omitempty,oneof=classic hybrid innovative"`
	LayoutDensity         string `json:"layoutDensity" binding:"omitempty,oneof=compact comfortable spacious"`
	SidebarWidth          int    `json:"sidebarWidth" binding:"omitempty,min=100,max=400"`
	SidebarCollapsedWidth int    `json:"sidebarCollapsedWidth" binding:"omitempty,min=40,max=100"`
	SidebarCollapsed      bool   `json:"sidebarCollapsed"`

	// 数据
	PageSize int `json:"pageSize" binding:"required,oneof=5 10 20 50 100"`

	// 自定义颜色（可选）
	CustomPrimaryColor string `json:"customPrimaryColor,omitempty" binding:"omitempty,eq=|len=7|len=4"`
	CustomSidebarColor string `json:"customSidebarColor,omitempty" binding:"omitempty,eq=|len=7|len=4"`

	// 语言
	Language string `json:"language" binding:"required,oneof=zh-CN en-US"`
}

// enumAllowed 判断 v 是否落在允许集合内。
func enumAllowed(v string, allowed []string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// sanitizeEnum 越界值 → 规范默认值（允许集合内的原样返回）。
func sanitizeEnum(v string, allowed []string, def string) string {
	if enumAllowed(v, allowed) {
		return v
	}
	return def
}

// intAllowed 判断 v 是否落在允许集合内。
func intAllowed(v int, allowed []int) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// firstNonEmpty 取第一个非空字符串（空串视为「本次未携带」）。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstNonZeroInt 取第一个非零整数（0 视为「本次未携带」）。
func firstNonZeroInt(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}

// sanitizePreferences 把落在允许集合之外的值替换为规范默认值。
//
// 对外（GET 响应）与对内（落库前）共用同一套自检，保证：
//   - API 响应永不返回越界值（历史行里残留的空串会被换成规范默认值）
//   - 服务层在任何输入下都不向带 CHECK 约束的列写出越界值
func sanitizePreferences(p *UserPreferences) *UserPreferences {
	p.Theme = sanitizeEnum(p.Theme, allowedThemes, defaultTheme)
	p.ThemeStyle = sanitizeEnum(p.ThemeStyle, allowedThemeStyles, defaultThemeStyle)
	p.LayoutType = sanitizeEnum(p.LayoutType, allowedLayoutTypes, defaultLayoutType)
	p.LayoutDensity = sanitizeEnum(p.LayoutDensity, allowedLayoutDensities, defaultLayoutDensity)
	p.Language = sanitizeEnum(p.Language, allowedLanguages, defaultLanguage)
	if !intAllowed(p.PageSize, allowedPageSizes) {
		p.PageSize = defaultPageSize
	}
	if p.SidebarWidth == 0 {
		p.SidebarWidth = defaultSidebarWidth
	}
	if p.SidebarCollapsedWidth == 0 {
		p.SidebarCollapsedWidth = defaultSidebarCollapsedWidth
	}
	return p
}

// mergePreferences 合并「请求值」与「兜底值」，并做允许集合自检。
//
// 空值语义：请求字段为零值（字符串零值 / 数值零值）表示「本次未携带」，
// 而不是「显式置空」—— 显式置空在该数据模型里根本不是合法状态
// （PG 的 CHECK 约束证明这些列的值域不含空串），把「未携带」映射成「置空」必然越界。
//
// 兜底来源：
//   - Update 分支（已有行）：取该行当前值 → 写回同值是幂等 no-op，
//     既不丢用户已有偏好（只切一次暗色模式不该连带重置布局密度），也不触碰 CHECK 约束。
//   - Create 分支（无历史行）：取规范默认值，使「先 GET 再 PUT」与「直接 PUT 建行」
//     两条路径落库结果一致。
//
// 例外（刻意为之）：SidebarCollapsed 始终按请求值写入 —— bool 无法区分
// 「未携带」与 false，前端始终显式发送它，数据库列也没有 CHECK 约束，不构成 500 风险。
func mergePreferences(req *UserPreferences, fallback *UserPreferences) *UserPreferences {
	merged := &UserPreferences{
		Theme:                 firstNonEmpty(req.Theme, fallback.Theme),
		ThemeStyle:            firstNonEmpty(req.ThemeStyle, fallback.ThemeStyle),
		LayoutType:            firstNonEmpty(req.LayoutType, fallback.LayoutType),
		LayoutDensity:         firstNonEmpty(req.LayoutDensity, fallback.LayoutDensity),
		SidebarWidth:          firstNonZeroInt(req.SidebarWidth, fallback.SidebarWidth),
		SidebarCollapsedWidth: firstNonZeroInt(req.SidebarCollapsedWidth, fallback.SidebarCollapsedWidth),
		SidebarCollapsed:      req.SidebarCollapsed,
		PageSize:              firstNonZeroInt(req.PageSize, fallback.PageSize),
		CustomPrimaryColor:    firstNonEmpty(req.CustomPrimaryColor, fallback.CustomPrimaryColor),
		CustomSidebarColor:    firstNonEmpty(req.CustomSidebarColor, fallback.CustomSidebarColor),
		Language:              firstNonEmpty(req.Language, fallback.Language),
	}
	return sanitizePreferences(merged)
}

// GetUserPreferences 获取用户个人设置
func (s *settingsService) GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	var pref UserPreference
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error

	// 如果不存在，返回硬编码默认值（v1.22 收尾：不再合并 sys.theme.default）
	if err == gorm.ErrRecordNotFound {
		return s.buildDefaultPreferences(), nil
	}

	if err != nil {
		return nil, fmt.Errorf("查询用户设置失败: %w", err)
	}

	// 历史行可能残留越界值（例如本 bug 曾写入的空串，SQLite 分支无 CHECK 约束时能落库），
	// 返回给前端前统一替换为规范默认值 —— API 响应永不返回越界值。
	return sanitizePreferences(&UserPreferences{
		Theme:                 pref.Theme,
		ThemeStyle:            pref.ThemeStyle,
		LayoutType:            pref.LayoutType,
		LayoutDensity:         pref.LayoutDensity,
		SidebarWidth:          pref.SidebarWidth,
		SidebarCollapsedWidth: pref.SidebarCollapsedWidth,
		SidebarCollapsed:      pref.SidebarCollapsed,
		PageSize:              pref.PageSize,
		CustomPrimaryColor:    pref.CustomPrimaryColor,
		CustomSidebarColor:    pref.CustomSidebarColor,
		Language:              pref.Language,
	}), nil
}

// buildDefaultPreferences 构建用户无偏好记录时的硬编码默认值
//
// v1.22 收尾：不再合并 sys.theme.default（已删除默认主题页面）。
// 行为变化：之前若管理员在 sys.theme.default 配置了 mode=dark，
//          新用户的默认 Theme 会被设为 dark；现在统一回到 light。
// 后续如需"管理员配置全局默认值"能力，应以独立的 sys.user_preferences_default 表
// 或 sys_config 通用键实现，不再耦合到 sys.theme.default 这一被删除的主题语义。
//
// 默认值统一取自本文件顶部的规范默认常量组，避免两处各自硬编码造成漂移。
func (s *settingsService) buildDefaultPreferences() *UserPreferences {
	return &UserPreferences{
		Theme:                 defaultTheme,
		ThemeStyle:            defaultThemeStyle,
		LayoutType:            defaultLayoutType,
		LayoutDensity:         defaultLayoutDensity,
		SidebarWidth:          defaultSidebarWidth,
		SidebarCollapsedWidth: defaultSidebarCollapsedWidth,
		SidebarCollapsed:      false,
		PageSize:              defaultPageSize,
		Language:              defaultLanguage,
	}
}

// UpdateUserPreferences 更新用户个人设置
//
// 落库前统一走「合并 + 自检」：请求省略的字段保留原值（Update 分支）或取规范默认值
// （Create 分支），越界值一律被替换为规范默认值 —— 服务层永远不写出越界值。
func (s *settingsService) UpdateUserPreferences(ctx context.Context, userID string, req *UserPreferences) error {
	var pref UserPreference

	// 查找现有记录
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		// 创建新记录：无历史行 → 兜底取规范默认值。
		// 显式给 pref 赋上合并后的值，不依赖 GORM 因 default tag 省略零值字段、
		// 从而落到数据库 DEFAULT 的隐式行为 —— 那样两种方言行为一致但无法被测试直接断言，
		// 且把正确性寄托在 GORM 的零值省略启发式上。
		merged := mergePreferences(req, s.buildDefaultPreferences())
		pref = UserPreference{
			UserID:                userID,
			Theme:                 merged.Theme,
			ThemeStyle:            merged.ThemeStyle,
			LayoutType:            merged.LayoutType,
			LayoutDensity:         merged.LayoutDensity,
			SidebarWidth:          merged.SidebarWidth,
			SidebarCollapsedWidth: merged.SidebarCollapsedWidth,
			SidebarCollapsed:      merged.SidebarCollapsed,
			PageSize:              merged.PageSize,
			CustomPrimaryColor:    merged.CustomPrimaryColor,
			CustomSidebarColor:    merged.CustomSidebarColor,
			Language:              merged.Language,
		}
		if createErr := s.db.WithContext(ctx).Create(&pref).Error; createErr != nil {
			return fmt.Errorf("创建用户设置失败: %w", createErr)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("查询用户设置失败: %w", err)
	}

	// 更新现有记录 - 使用 Updates 更新所有字段。
	// 兜底值就是该行原值，写回同值是幂等 no-op（既不丢用户已有偏好，也不触碰 CHECK 约束）。
	current := &UserPreferences{
		Theme:                 pref.Theme,
		ThemeStyle:            pref.ThemeStyle,
		LayoutType:            pref.LayoutType,
		LayoutDensity:         pref.LayoutDensity,
		SidebarWidth:          pref.SidebarWidth,
		SidebarCollapsedWidth: pref.SidebarCollapsedWidth,
		PageSize:              pref.PageSize,
		CustomPrimaryColor:    pref.CustomPrimaryColor,
		CustomSidebarColor:    pref.CustomSidebarColor,
		Language:              pref.Language,
	}
	merged := mergePreferences(req, current)
	if updateErr := s.db.WithContext(ctx).Model(&pref).Updates(map[string]interface{}{
		"theme":                   merged.Theme,
		"theme_style":             merged.ThemeStyle,
		"layout_type":             merged.LayoutType,
		"layout_density":          merged.LayoutDensity,
		"sidebar_width":           merged.SidebarWidth,
		"sidebar_collapsed_width": merged.SidebarCollapsedWidth,
		"sidebar_collapsed":       merged.SidebarCollapsed,
		"page_size":               merged.PageSize,
		"custom_primary_color":    merged.CustomPrimaryColor,
		"custom_sidebar_color":    merged.CustomSidebarColor,
		"language":                merged.Language,
	}).Error; updateErr != nil {
		return fmt.Errorf("更新用户设置失败: %w", updateErr)
	}

	return nil
}
