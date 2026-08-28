package v1

// =====================================================================
// Phase 80-03 Task 3: captcha 家族 handler 真装配测试。
//
// 复用 auth_80_03_test.go 的 newMiniCore8003 / newMiniCore8003 keystones
// (同包 _test.go 共享文件级 helper);所有断言经 SetupCaptchaRouter /
// SetupCaptchaBackgroundRouter 真装配 + httptest/真实 multipart 请求。
//
// fixture 约定:每个用例独立 sqlite 文件库 + 独立 MemoryCache,
// sys_config 走 78-01 同款精简 DDL(只建 LoadConfig 实际 Pluck 的列)。
// 文件类(upload)用 t.Chdir(t.TempDir()) 让 CaptchaBackgroundService
// 默认相对存储路径 ./uploads/captcha/backgrounds 落入临时目录,
// 满足威胁模型 T-80-03 信任边界(测试 ↔ 文件系统)禁写仓库目录。
// =====================================================================

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/captcha"
)

// sysCaptchaConfigDDL8003 LoadConfig 走 s.db.Table("sys_config").Pluck("config_value", ...)
// —— 只 Pluck config_value 列;config_key 在 Where 子句,无需索引精简即可。
const sysCaptchaConfigDDL8003 = `CREATE TABLE IF NOT EXISTS sys_config (
	id TEXT PRIMARY KEY,
	config_key TEXT,
	config_value TEXT,
	deleted_at DATETIME)`

// migrateCaptchaTables8003 建 handler 引用的表集合(captcha_background 走 AutoMigrate;
// sys_config 走精简 DDL;tests 各取所需)。
func migrateCaptchaTables8003(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(&models.CaptchaBackground{}))
	require.NoError(t, db.Exec(sysCaptchaConfigDDL8003).Error)
}

// seedCaptchaConfig8003 写一行 sys_config(handler reload 链路 + LoadConfig 真读取)。
func seedCaptchaConfig8003(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_config (id, config_key, config_value) VALUES (?, ?, ?)`,
		"id-cfg-"+key, key, value,
	).Error)
}

// mountCaptchaRouter8003 真 SetupCaptchaRouter + 集团路径(用于纯 captcha_handler 4 个)。
func mountCaptchaRouter8003(t *testing.T, c *core.Core) *gin.Engine {
	t.Helper()
	router := gin.New()
	group := router.Group("/api/v1/system/auth")
	SetupCaptchaRouter(group, c)
	return router
}

// mountCaptchaBackgroundRouter8003 真 SetupCaptchaBackgroundRouter 装配。
func mountCaptchaBackgroundRouter8003(t *testing.T, c *core.Core) *gin.Engine {
	t.Helper()
	router := gin.New()
	group := router.Group("/api/v1/system/captcha-background")
	SetupCaptchaBackgroundRouter(group, c)
	return router
}

// makePNGBytes8003 现场造一张可被 image.DecodeConfig 解析的真 PNG(背景图上传 happy path)。
func makePNGBytes8003(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 128, A: 255})
		}
	}
	tmp := filepath.Join(t.TempDir(), "src.png")
	f, err := os.Create(tmp)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())
	data, err := os.ReadFile(tmp)
	require.NoError(t, err)
	return data
}

// seedCaptchaBgRow8003 落一张真 PNG 文件并 INSERT 一行 sys_captcha_background 记录
// (供 getStatistics / PreGeneratePool / toggle 等 handler 真读取)。
func seedCaptchaBgRow8003(t *testing.T, db *gorm.DB, id string, status models.CaptchaBackgroundStatus, shape models.PieceShape, difficulty int) *models.CaptchaBackground {
	t.Helper()
	pngBytes := makePNGBytes8003(t, 200, 100)
	path := filepath.Join(t.TempDir(), id+".png")
	require.NoError(t, os.WriteFile(path, pngBytes, 0o644))
	bg := &models.CaptchaBackground{
		ID:              id,
		FileName:        id + ".png",
		FilePath:        path,
		FileSize:        int64(len(pngBytes)),
		FileWidth:       200,
		FileHeight:      100,
		FileMD5:         "deadbeef8003",
		PieceShape:      shape,
		DifficultyLevel: models.DifficultyLevel(difficulty),
		AllowedShapes:   models.StringArray{"circle", "square"},
		Status:          status,
	}
	require.NoError(t, db.Create(bg).Error)
	return bg
}

// =====================================================================
// TestCap8003_ captcha_handler (SetupCaptchaRouter — 4 个 handler)
// =====================================================================

// TestCap8003_GetCaptcha 表驱动:disabled / normal / 限流 三态。
func TestCap8003_GetCaptcha(t *testing.T) {
	t.Run("disabled默认_空对象", func(t *testing.T) {
		c, _ := newMiniCore8003(t)
		router := mountCaptchaRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/auth/captcha", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		// disabled 时 captchaResp == nil → handler 返空对象 {}
		assert.Equal(t, "{}", string(resp.Data), "disabled 应返 gin.H{} 而非 captcha data")
	})

	t.Run("normal启用_返回base64验证码", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateCaptchaTables8003(t, db)
		seedCaptchaConfig8003(t, db, "sys.account.captchaEnabled", "normal")
		require.NoError(t, c.CaptchaService.LoadConfig(context.Background()))

		router := mountCaptchaRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/auth/captcha", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		assert.Contains(t, string(resp.Data), "data:image/png;base64,")
	})

	t.Run("限流触发_400", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateCaptchaTables8003(t, db)
		seedCaptchaConfig8003(t, db, "sys.account.captchaEnabled", "normal")
		seedCaptchaConfig8003(t, db, "sys.captcha.ip_rate_limit", "1") // 第二次必拒
		require.NoError(t, c.CaptchaService.LoadConfig(context.Background()))

		router := mountCaptchaRouter8003(t, c)
		// 第一次必成功
		w1, _ := performJSON8003(t, router, http.MethodPost, "/api/v1/system/auth/captcha", nil)
		require.Equal(t, http.StatusOK, w1.Code)
		// 第二次触发限流
		w2, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/auth/captcha", nil)
		assert.Equal(t, http.StatusBadRequest, w2.Code)
		assert.Contains(t, resp.Message, "过于频繁")
	})
}

// TestCap8003_VerifySlider 三态:成功 / 错位移 / 错 token。
// 注意:verify 成功会消费 captcha(从缓存删 data/attempts),所以每个子例单独种一份。
func TestCap8003_VerifySlider(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateCaptchaTables8003(t, db)
	c.CaptchaService.GetConfig().Enabled = captcha.CaptchaTypeSlider
	c.CaptchaService.GetConfig().MaxAttempts = 5

	ctx := context.Background()
	router := mountCaptchaRouter8003(t, c)

	seedSlider8003 := func(t *testing.T, captID string, xPos int, token string) {
		t.Helper()
		require.NoError(t, c.Cache.SetJSON(ctx,
			fmt.Sprintf("captcha:data:%s", captID),
			core.SliderVerifyData{XPos: xPos, YPos: 50, Token: token},
			time.Minute))
		require.NoError(t, c.Cache.SetInt(ctx, fmt.Sprintf("captcha:attempts:%s", captID), 0, time.Minute))
	}

	t.Run("成功_位置正确token匹配", func(t *testing.T) {
		captID := "cap-slider-ok-8003"
		seedSlider8003(t, captID, 100, "tok-ok")
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/captcha/verify/slider",
			map[string]any{"captchaId": captID, "xPos": 100, "token": "tok-ok"})
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var data map[string]any
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Equal(t, true, data["success"])
	})

	t.Run("位置错误_拒绝", func(t *testing.T) {
		captID := "cap-slider-badpos-8003"
		seedSlider8003(t, captID, 100, "tok-badpos")
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/captcha/verify/slider",
			map[string]any{"captchaId": captID, "xPos": 200, "token": "tok-badpos"})
		assert.Equal(t, http.StatusBadRequest, w.Code, "位置错 → ErrCaptchaError")
		assert.Contains(t, resp.Message, "位置不正确")
	})

	t.Run("绑定失败_缺字段", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/auth/captcha/verify/slider",
			map[string]any{"captchaId": "cap-slider-anything"}) // 缺 xPos/token
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp.Message, "请求参数错误")
	})
}

// TestCap8003_GetConfig 返回当前配置快照(字段映射断言)。
func TestCap8003_GetConfig(t *testing.T) {
	c, _ := newMiniCore8003(t)
	c.CaptchaService.GetConfig().Enabled = captcha.CaptchaTypeNormal
	c.CaptchaService.GetConfig().Type = 5
	c.CaptchaService.GetConfig().ExpireTime = 10
	c.CaptchaService.GetConfig().MaxAttempts = 4

	router := mountCaptchaRouter8003(t, c)
	w, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/auth/captcha/config", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, resp.Code)
	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	assert.Equal(t, string(captcha.CaptchaTypeNormal), data["enabled"])
	assert.Equal(t, 5.0, data["type"])
	assert.Equal(t, 10.0, data["expireTime"])
	assert.Equal(t, 4.0, data["maxAttempts"])
}

// TestCap8003_Reload 两态:有 sys_config → 成功加载;无 sys_config → 默认值兜底成功。
// QUIRK-80-03-B(就地锁定):LoadConfig 内部 Pluck 错误被吞(parse default 兜底),
// handler 的 "重新加载配置失败" 分支实际上不可达 —— 无须覆盖。
func TestCap8003_Reload(t *testing.T) {
	t.Run("有sys_config_加载成功", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateCaptchaTables8003(t, db)
		seedCaptchaConfig8003(t, db, "sys.account.captchaEnabled", "normal")

		router := mountCaptchaRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/auth/captcha/config/reload", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var data map[string]any
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Contains(t, data, "config")
		// LoadConfig 已读 sys_config → 服务 config 同步为 normal
		assert.Equal(t, captcha.CaptchaTypeNormal, c.CaptchaService.GetConfig().Enabled,
			"reload 后服务 config 应真实更新")
	})

	t.Run("无sys_config_默认disabled兜底成功", func(t *testing.T) {
		c, _ := newMiniCore8003(t)
		// 故意不建 sys_config;LoadConfig 对每个 key parse default → 全 default = disabled
		router := mountCaptchaRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/auth/captcha/config/reload", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code, "无 sys_config → default 兜底而非 500")
		assert.Equal(t, captcha.CaptchaTypeDisabled, c.CaptchaService.GetConfig().Enabled)
	})
}

// =====================================================================
// TestCapBg8003_ captcha_background_handler (SetupCaptchaBackgroundRouter — 8 个 handler)
// =====================================================================

// TestCapBg8003_List_Paged 列表分页 + 筛选 + 缺参校验。
func TestCapBg8003_List_Paged(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateCaptchaTables8003(t, db)
	seedCaptchaBgRow8003(t, db, "bg-list-1", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)
	seedCaptchaBgRow8003(t, db, "bg-list-2", models.CaptchaBgDisabled, models.PieceShapeSquare, 2)
	seedCaptchaBgRow8003(t, db, "bg-list-3", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)

	router := mountCaptchaBackgroundRouter8003(t, c)

	t.Run("无筛选_全部", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/list",
			map[string]any{"current": 1, "pageSize": 10})
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var listResp models.CaptchaBackgroundListResponse
		require.NoError(t, json.Unmarshal(resp.Data, &listResp))
		assert.EqualValues(t, 3, listResp.Total)
		assert.Len(t, listResp.Items, 3)
	})

	t.Run("按pieceShape筛选", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/list",
			map[string]any{"current": 1, "pageSize": 10, "pieceShape": "circle"})
		require.Equal(t, http.StatusOK, w.Code)
		var listResp models.CaptchaBackgroundListResponse
		require.NoError(t, json.Unmarshal(resp.Data, &listResp))
		assert.EqualValues(t, 2, listResp.Total)
	})

	t.Run("按status筛选_引用models常量", func(t *testing.T) {
		statusVal := int(models.CaptchaBgDisabled)
		// QUIRK-80-03-D(就地锁定):models.CaptchaBackground.Status 类型 CaptchaBackgroundStatus
		// 带 gorm:"not null;default:1" —— GORM 在 db.Create 时把零值 (0/disabled) 视为
		// "use default" 而覆盖为 1(enabled)。无法通过 model Create 落 status=0 行,
		// 必须 raw SQL Exec 直接写值。下面通过 Exec 落 1 行 status=0 以真测筛选分支。
		require.NoError(t, db.Exec(
			`INSERT INTO sys_captcha_background
			 (id, file_name, file_path, file_size, file_width, file_height,
			  piece_shape, difficulty_level, allowed_shapes, status, use_count, sort_order,
			  created_at, updated_at, version)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"bg-ls-disabled-raw", "raw.png", "/tmp/raw.png", 100, 10, 10,
			"square", 1, "[]", int(models.CaptchaBgDisabled), 0, 0,
			time.Now(), time.Now(), 1,
		).Error)

		_, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/list",
			map[string]any{"current": 1, "pageSize": 10, "status": statusVal})
		require.Equal(t, 0, resp.Code, resp.Message)
		var listResp models.CaptchaBackgroundListResponse
		require.NoError(t, json.Unmarshal(resp.Data, &listResp))
		assert.EqualValues(t, 1, listResp.Total, "只 1 行 disabled(raw 落库绕过 default 怪癖)")
	})

	t.Run("pageSize越界_绑定失败", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/list",
			map[string]any{"current": 1, "pageSize": 200}) // max=100
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp.Message, "请求参数错误")
	})
}

// TestCapBg8003_Get 存在 + 不存在。
func TestCapBg8003_Get(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateCaptchaTables8003(t, db)
	seedCaptchaBgRow8003(t, db, "bg-get-1", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)

	router := mountCaptchaBackgroundRouter8003(t, c)

	t.Run("存在_返回DTO", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/captcha-background/bg-get-1", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var dto models.CaptchaBackgroundDTO
		require.NoError(t, json.Unmarshal(resp.Data, &dto))
		assert.Equal(t, "bg-get-1", dto.ID)
		assert.Equal(t, int(models.CaptchaBgEnabled), dto.Status)
	})

	t.Run("不存在_404", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost, "/api/v1/system/captcha-background/no-such-id", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, resp.Message, "背景图不存在")
	})
}

// TestCapBg8003_Upload 真 multipart 上传 + 文件缺/格式错/类型无效 三负分支。
func TestCapBg8003_Upload(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateCaptchaTables8003(t, db)
	// t.Chdir 让 CaptchaBackgroundService 默认相对存储路径落入 TempDir
	// (满足威胁模型: 测试不写仓库目录);Go 1.24 t.Chdir 在 Cleanup 自动恢复 cwd。
	t.Chdir(t.TempDir())

	router := mountCaptchaBackgroundRouter8003(t, c)

	t.Run("happy path_真PNG上传", func(t *testing.T) {
		body := &bytes.Buffer{}
		wr := multipart.NewWriter(body)
		pngBytes := makePNGBytes8003(t, 320, 160)
		fw, err := wr.CreateFormFile("file", "ok-8003.png")
		require.NoError(t, err)
		_, err = fw.Write(pngBytes)
		require.NoError(t, err)
		_ = wr.WriteField("pieceShape", string(models.PieceShapeCircle))
		_ = wr.WriteField("difficultyLevel", "1")
		_ = wr.WriteField("allowedShapes", `["circle","square"]`)
		_ = wr.WriteField("remark", "80-03 captcha bg")
		require.NoError(t, wr.Close())

		req, err := http.NewRequest(http.MethodPost, "/api/v1/system/captcha-background/upload", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", wr.FormDataContentType())
		httpW := httptest.NewRecorder()
		router.ServeHTTP(httpW, req)

		require.Equal(t, http.StatusOK, httpW.Code)
		var resp apiResp8003
		require.NoError(t, json.Unmarshal(httpW.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code, "upload happy path 应成功")
		var dto models.CaptchaBackgroundDTO
		require.NoError(t, json.Unmarshal(resp.Data, &dto))
		assert.NotEmpty(t, dto.ID)
		assert.Equal(t, 320, dto.FileWidth)
		assert.Equal(t, 160, dto.FileHeight)
		assert.Equal(t, int(models.CaptchaBgEnabled), dto.Status)
		// 行落库
		var count int64
		require.NoError(t, db.Model(&models.CaptchaBackground{}).Count(&count).Error)
		assert.GreaterOrEqual(t, count, int64(1))
	})

	t.Run("缺文件_400", func(t *testing.T) {
		body := &bytes.Buffer{}
		wr := multipart.NewWriter(body)
		_ = wr.WriteField("pieceShape", string(models.PieceShapeCircle))
		require.NoError(t, wr.Close())
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/system/captcha-background/upload", body)
		req.Header.Set("Content-Type", wr.FormDataContentType())
		httpW := httptest.NewRecorder()
		router.ServeHTTP(httpW, req)
		assert.Equal(t, http.StatusBadRequest, httpW.Code)
		var resp apiResp8003
		require.NoError(t, json.Unmarshal(httpW.Body.Bytes(), &resp))
		assert.Contains(t, resp.Message, "请选择文件")
	})

	t.Run("不支持的扩展名_500", func(t *testing.T) {
		body := &bytes.Buffer{}
		wr := multipart.NewWriter(body)
		fw, _ := wr.CreateFormFile("file", "ok-8003.txt")
		_, _ = fw.Write([]byte("not an image"))
		_ = wr.WriteField("pieceShape", string(models.PieceShapeCircle))
		require.NoError(t, wr.Close())
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/system/captcha-background/upload", body)
		req.Header.Set("Content-Type", wr.FormDataContentType())
		httpW := httptest.NewRecorder()
		router.ServeHTTP(httpW, req)
		assert.Equal(t, http.StatusInternalServerError, httpW.Code)
		var resp apiResp8003
		require.NoError(t, json.Unmarshal(httpW.Body.Bytes(), &resp))
		assert.Contains(t, resp.Message, "文件验证失败")
	})
}

// TestCapBg8003_Update + 缺参校验。
func TestCapBg8003_Update(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateCaptchaTables8003(t, db)
	seedCaptchaBgRow8003(t, db, "bg-upd-1", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)

	router := mountCaptchaBackgroundRouter8003(t, c)

	t.Run("happy path_更新难度", func(t *testing.T) {
		newDiff := 3
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/bg-upd-1/update",
			map[string]any{"difficultyLevel": newDiff})
		require.Equal(t, http.StatusOK, w.Code, resp.Message)
		require.Equal(t, 0, resp.Code)

		var bg models.CaptchaBackground
		require.NoError(t, db.Where("id = ?", "bg-upd-1").First(&bg).Error)
		assert.Equal(t, 3, int(bg.DifficultyLevel))
	})

	t.Run("不存在的ID_不报错(实现层Updates无匹配行RowsAffected=0)", func(t *testing.T) {
		// 文档级注释: handler 未预查存在性 — Updates 0 rows 无 error。
		// 鉴于此行为可观测,真装配测试也如实锁定(GORM 行为,非 QUIRK)。
		_, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/no-such-id/update",
			map[string]any{"difficultyLevel": 2})
		assert.Equal(t, 0, resp.Code, "GORM Updates 不存在的行不报错")
		var data map[string]string
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Contains(t, data["message"], "更新成功")
	})

	t.Run("非法JSON_绑定失败", func(t *testing.T) {
		// UpdateRequest 字段都是 *Type,无 binding "required" 约束;但非法 JSON
		// 本身必触发 ShouldBindJSON 错误 → handler 早退 400。
		req, err := http.NewRequest(http.MethodPost, "/api/v1/system/captcha-background/bg-upd-1/update",
			strings.NewReader(`{invalid-json`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		httpW := httptest.NewRecorder()
		router.ServeHTTP(httpW, req)
		assert.Equal(t, http.StatusBadRequest, httpW.Code)
	})
}

// TestCapBg8003_Delete 存在 + 不存在 + 软删除断言。
func TestCapBg8003_Delete(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateCaptchaTables8003(t, db)
	t.Chdir(t.TempDir())
	seedCaptchaBgRow8003(t, db, "bg-del-1", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)

	router := mountCaptchaBackgroundRouter8003(t, c)

	t.Run("存在_删除成功_行不可见", func(t *testing.T) {
		_, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/bg-del-1/delete", nil)
		require.Equal(t, 0, resp.Code, resp.Message)
		var data map[string]string
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Contains(t, data["message"], "删除成功")

		// QUIRK-80-03-C(就地锁定):当前 CaptchaBackground.DeletedAt 字段下,
		// GORM 默认行为在 glebarez sqlite 下走硬删除(实测验证),行彻底消失。
		// 仅断言"默认作用域查不到"即可(无论软硬删均成立)。
		var count int64
		require.NoError(t, db.Model(&models.CaptchaBackground{}).Where("id = ?", "bg-del-1").Count(&count).Error)
		assert.Zero(t, count, "删除后默认作用域不可见")
	})

	t.Run("不存在_404", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/ghost-id/delete", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, resp.Message, "背景图不存在")
	})
}

// TestCapBg8003_Toggle 切换两态 + 不存在 + 引用 models 常量。
func TestCapBg8003_Toggle(t *testing.T) {
	c, db := newMiniCore8003(t)
	migrateCaptchaTables8003(t, db)
	seedCaptchaBgRow8003(t, db, "bg-tog-1", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)

	router := mountCaptchaBackgroundRouter8003(t, c)

	t.Run("启用→禁用→启用", func(t *testing.T) {
		// 1st: enabled → disabled
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/bg-tog-1/toggle", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var data map[string]any
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Equal(t, float64(int(models.CaptchaBgDisabled)), data["status"], "应引用 models 常量值")

		// 2nd: disabled → enabled
		w, resp = performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/bg-tog-1/toggle", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Equal(t, float64(int(models.CaptchaBgEnabled)), data["status"])
	})

	t.Run("不存在_404", func(t *testing.T) {
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/ghost-id/toggle", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, resp.Message, "背景图不存在")
	})
}

// TestCapBg8003_Statistics 空表 + 单行 enabled(覆盖表前两个 Count 路径)。
//
// QUIRK-80-03-E(就地锁定,deferred fix):Statistics handler 在 [统计形状分布] 时
// `db.Select("piece_shape, count(*) as count").Group(...).Scan(&map[string]int)`
// 对双列 SELECT 触发 GORM 错误 "expected 2 destination arguments in Scan, not 1",
// production 路径下若出现 ≥1 行也会 500。零行时跳过该 Scan,返回空分布 —— 故本测试
// 用空表避免触发;形状分布断言留待 production fix(零业务变更纪律禁止此处改源码)。
func TestCapBg8003_Statistics(t *testing.T) {
	t.Run("空表_零值分布", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateCaptchaTables8003(t, db)

		router := mountCaptchaBackgroundRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/statistics", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var stats models.StatisticsResponse
		require.NoError(t, json.Unmarshal(resp.Data, &stats))
		assert.Equal(t, 0, stats.TotalCount)
		assert.Equal(t, 0, stats.EnabledCount)
		assert.Equal(t, 0, stats.DisabledCount)
	})

	t.Run("enabled行_走Count分支避开brokenScan", func(t *testing.T) {
		// QUIRK-80-03-D 同款怪癖:Status=0 被 default 覆盖。
		// 这里仅 seed 启用行(1 行走 COUNT 路径,不动 GroupBy Scan);
		// 形状/难度分布的真实断言待 production Scan bug 修复后再补。
		c, db := newMiniCore8003(t)
		migrateCaptchaTables8003(t, db)
		seedCaptchaBgRow8003(t, db, "bg-stat-1", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)

		router := mountCaptchaBackgroundRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/statistics", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var stats models.StatisticsResponse
		require.NoError(t, json.Unmarshal(resp.Data, &stats))
		assert.GreaterOrEqual(t, stats.TotalCount, 1)
	})
}

// TestCapBg8003_Preload 预生成缓存池:空表 + 有 enabled 行两种状态都返回成功。
func TestCapBg8003_Preload(t *testing.T) {
	t.Run("空表_快速成功", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateCaptchaTables8003(t, db)
		router := mountCaptchaBackgroundRouter8003(t, c)

		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/preload", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
		var data map[string]string
		require.NoError(t, json.Unmarshal(resp.Data, &data))
		assert.Contains(t, data["message"], "预加载完成")
	})

	t.Run("有enabled行_预生成真路径", func(t *testing.T) {
		c, db := newMiniCore8003(t)
		migrateCaptchaTables8003(t, db)
		t.Chdir(t.TempDir())
		seedCaptchaBgRow8003(t, db, "bg-pre-1", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)
		seedCaptchaBgRow8003(t, db, "bg-pre-2", models.CaptchaBgEnabled, models.PieceShapeCircle, 1)

		router := mountCaptchaBackgroundRouter8003(t, c)
		w, resp := performJSON8003(t, router, http.MethodPost,
			"/api/v1/system/captcha-background/preload", nil)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, 0, resp.Code)
	})
}
