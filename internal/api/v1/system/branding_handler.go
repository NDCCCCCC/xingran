package system

import (
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// logoUploadDir logo 图片落盘目录（engine.Static("/uploads", "./uploads") 已托管）。
// 声明为变量而非常量：契约测试把它重定向到 t.TempDir()，避免测试把文件写进包目录。
var logoUploadDir = "./uploads/brand"

// logoPublicPath logo 的 URL 前缀，必须与 logoUploadDir 的静态托管映射一致
// （./uploads/brand ↔ /uploads/brand）。
var logoPublicPath = "/uploads/brand"

// logoMaxBytes logo 大小上限 2MB
const logoMaxBytes = 2 << 20

// logoAllowedExts 允许的图片扩展名（白名单，防任意文件上传）
var logoAllowedExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".svg": true, ".webp": true, ".ico": true,
}

// BrandingHandler 站点品牌配置处理器
type BrandingHandler struct {
	service systemServices.BrandingService
	core    *core.Core
}

// NewBrandingHandler 创建站点品牌配置处理器
func NewBrandingHandler(service systemServices.BrandingService) *BrandingHandler {
	return &BrandingHandler{service: service}
}

// WithCore 注入 core（操作日志所需）
func (h *BrandingHandler) WithCore(core *core.Core) *BrandingHandler {
	if h != nil {
		h.core = core
	}
	return h
}

// GetBranding 获取品牌配置（管理端，需认证）
//
// @Summary      获取站点品牌配置
// @Description  读取全部 brand.* 配置聚合视图
// @Tags         系统设置
// @Produce      json
// @Success      200 {object} response.Response{data=systemServices.BrandingConfig}
// @Router       /system/branding [get]
func (h *BrandingHandler) GetBranding(c *gin.Context) {
	cfg, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	response.Success(c, cfg)
}

// UpdateBranding 更新品牌配置（管理端；body: {configKey: value} 映射）
//
// @Summary      更新站点品牌配置
// @Description  批量更新 brand.* 配置；只接受 brand.* 键；成功后失效公开缓存
// @Tags         系统设置
// @Accept       json
// @Produce      json
// @Param        request body map[string]string true "config_key → value 映射"
// @Success      200 {object} response.Response
// @Router       /system/branding/update [post]
func (h *BrandingHandler) UpdateBranding(c *gin.Context) {
	var values map[string]string
	if err := c.ShouldBindJSON(&values); err != nil {
		response.Error(c, apperrors.Wrap(err, apperrors.CodeParamError, "请求参数错误"))
		return
	}
	if len(values) == 0 {
		response.Error(c, response.ErrBadRequest, "请提供要更新的配置项")
		return
	}
	if err := h.service.Update(c.Request.Context(), values); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	operlogRecord(c, h.core, "站点品牌")
	response.Success(c, gin.H{"message": "品牌配置已更新"})
}

// UploadLogo 上传 Logo 图片（管理端）
//
// @Summary      上传站点 Logo
// @Description  保存图片到 uploads/brand/ 并把访问 URL 写入 brand.logo_url
// @Tags         系统设置
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Logo 图片(≤2MB, png/jpg/svg/webp/ico)"
// @Success      200 {object} response.Response{data=object{logoUrl=string}}
// @Router       /system/branding/logo [post]
func (h *BrandingHandler) UploadLogo(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.ErrBadRequest, "请选择文件")
		return
	}
	if file.Size > logoMaxBytes {
		response.Error(c, response.ErrBadRequest, "Logo 图片不能超过 2MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !logoAllowedExts[ext] {
		response.Error(c, response.ErrBadRequest, "仅支持 png/jpg/jpeg/svg/webp/ico 格式")
		return
	}

	if err := os.MkdirAll(logoUploadDir, 0o755); err != nil {
		response.Error(c, response.ErrServerError, "创建存储目录失败")
		return
	}

	// 固定文件名 logo.<ext>：同一时刻只保留一份当前 logo，换格式自动淘汰旧文件
	dst := filepath.Join(logoUploadDir, "logo"+ext)
	src, err := file.Open()
	if err != nil {
		response.Error(c, response.ErrServerError, "读取文件失败")
		return
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		response.Error(c, response.ErrServerError, "保存文件失败")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		response.Error(c, response.ErrServerError, "写入文件失败")
		return
	}

	// URL 带 ?v=<unix> 破坏浏览器缓存；清理其它扩展名的旧 logo
	url := fmt.Sprintf("%s/logo%s?v=%d", logoPublicPath, ext, time.Now().Unix())
	for other := range logoAllowedExts {
		if other != ext {
			_ = os.Remove(filepath.Join(logoUploadDir, "logo"+other))
		}
	}

	if err := h.service.SetLogoURL(c.Request.Context(), url); err != nil {
		response.Error(c, response.ErrServerError, err.Error())
		return
	}
	operlogRecord(c, h.core, "站点品牌")
	response.Success(c, gin.H{"logoUrl": url})
}

// operlogRecord 统一操作日志包装（core 未注入时静默跳过，方便单测）
func operlogRecord(c *gin.Context, core *core.Core, module string) {
	if core == nil {
		return
	}
	operlog.Record(c, core.OperLogService, core.GetDB(), module, operlog.OperTypeUpdate)
}

// newBrandingServiceFromCore 从 Core 组装 BrandingService（db + CacheProvider）
//
// core.Cache 为 nil 时回落 NoOpCacheProvider：CacheAdapter.GetOrSet 内部直接调
// a.cache.Get，nil cache 会 panic。生产启动链路强制要求 Cache 非空，但
// handler 契约测试会构造最小 core（只给 DB），这里必须自保。
func newBrandingServiceFromCore(core *core.Core) systemServices.BrandingService {
	if core == nil {
		return systemServices.NewBrandingService(nil, &systemServices.NoOpCacheProvider{})
	}
	var provider systemServices.CacheProvider
	if core.Cache != nil {
		provider = systemServices.NewCacheAdapter(core.Cache)
	} else {
		provider = &systemServices.NoOpCacheProvider{}
	}
	return systemServices.NewBrandingService(core.GetDB(), provider)
}

// GetPublicBranding 获取公开品牌信息（免认证；登录页渲染用）
//
// @Summary      获取公开品牌信息
// @Description  免认证读取品牌聚合配置；CacheProvider 缓存 5 分钟；后端故障时返回 seed 默认值保证登录页可渲染
// @Tags         系统设置
// @Produce      json
// @Success      200 {object} response.Response{data=systemServices.BrandingConfig}
// @Router       /system/branding/public [get]
func (h *BrandingHandler) GetPublicBranding(c *gin.Context) {
	cfg, err := h.service.GetPublic(c.Request.Context())
	if err != nil || cfg == nil {
		// GetPublic 内部已多重降级；到这里的失败再兜一次底（返回空对象让前端走 fallback 常量）
		response.Success(c, &systemServices.BrandingConfig{})
		return
	}
	response.Success(c, cfg)
}

// ServeLogo 返回当前 Logo 文件（免认证）
//
// @Summary      获取站点 Logo 文件
// @Description  免认证返回 uploads/brand/logo.*；未上传时 404（前端回退方块字 mark）
// @Tags         系统设置
// @Produce      octet-stream
// @Success      200 {file} file
// @Failure      404 {object} response.Response
// @Router       /system/branding/logo [get]
func (h *BrandingHandler) ServeLogo(c *gin.Context) {
	cfg, err := h.service.GetPublic(c.Request.Context())
	if err == nil && cfg != nil && cfg.LogoURL != "" {
		// LogoURL 形如 /uploads/brand/logo.png?v=...；按 logoPublicPath ↔ logoUploadDir
		// 的映射还原磁盘路径（不写死 "./uploads"，否则测试重定向与将来换存储目录都会失效）。
		// rel 来自 DB（管理员可写），必须先拒绝 .. 防目录穿越，再用 FromSlash 转平台分隔符。
		pathOnly := strings.SplitN(cfg.LogoURL, "?", 2)[0]
		if rel, ok := strings.CutPrefix(pathOnly, logoPublicPath); ok && !strings.Contains(rel, "..") {
			local := filepath.Join(logoUploadDir, filepath.FromSlash(rel))
			if fi, statErr := os.Stat(local); statErr == nil && !fi.IsDir() {
				c.File(local)
				return
			}
		}
	}
	// 未上传或文件缺失：按 brand.site_mark 输出 SVG 方块字，避免 404 噪音。
	// mark 来自 DB（管理员可写），写入 SVG 前必须转义，防止注入标签。
	mark := "数"
	if err == nil && cfg != nil && cfg.SiteMark != "" {
		mark = cfg.SiteMark
	}
	mark = html.EscapeString(mark)
	c.Header("Cache-Control", "no-store")
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 48 48"><rect width="48" height="48" rx="8" fill="#156031"/><text x="24" y="32" font-size="24" text-anchor="middle" fill="#fff" font-family="system-ui">%s</text></svg>`, mark)
	c.Data(200, "image/svg+xml; charset=utf-8", []byte(svg))
}
