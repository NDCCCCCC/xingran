package core

// =====================================================================
// Phase 78-01 Task 3: CaptchaBackgroundService Upload 全分支 +
// GetRandomEnabled(t.TempDir 落盘隔离)
//
// 关键纪律:
//   - svc.config.StoragePath 一律 t.TempDir(),防止污染仓库 uploads/ 目录(T-78-01-01)。
//   - 状态断言一律引用 internal/models 具名常量(CLAUDE.md 0=normal, 1=disabled 反)。
//   - 真 PNG 字节用 stdlib image/png 现造,零二进制 fixture。
//   - 不构造带 L2Writer 的 MultiLevelCache(R-7 由 78-02 单独承担)。
// =====================================================================

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	coredb "github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// newBg78 装配 CaptchaBackgroundService + sqlite + MemoryCache + t.TempDir 落盘隔离。
// 同包白盒:直写 svc.config.StoragePath = t.TempDir() 强制覆盖默认值
// ./uploads/captcha/backgrounds,防污染仓库工作树(T-78-01-01 守护)。
// 表结构走 GORM Migrator.CreateTable(&models.CaptchaBackground{}) —— 自动覆盖
// CreatedBy/UpdatedBy/Version/remark 等所有 gorm tag 列,避免手写 DDL 漏列。
func newBg78(t *testing.T) (*CaptchaBackgroundService, *gorm.DB, *cache.MemoryCache) {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bg78.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := gormDB.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, gormDB.Migrator().CreateTable(&models.CaptchaBackground{}))

	mem := cache.NewMemoryCache(1000, 5*time.Minute)
	t.Cleanup(func() { _ = mem.Close() })
	dbWrapper := &coredb.Database{DB: gormDB, Type: "sqlite"}
	svc := NewCaptchaBackgroundService(dbWrapper, mem)
	svc.config.StoragePath = filepath.Join(t.TempDir(), "uploads", "captcha")
	return svc, gormDB, mem
}

// makeBG78PNG 用 stdlib image/png 现造一张 w×h 真 PNG(返回字节与 t.TempDir 落盘路径)。
func makeBG78PNG(t *testing.T, w, h int) ([]byte, string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 128, A: 255})
		}
	}
	dir := filepath.Join(t.TempDir(), "pngsrc")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	dst := filepath.Join(dir, fmt.Sprintf("%d_%dx%d.png", time.Now().UnixNano(), w, h))
	f, err := os.Create(dst)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	return data, dst
}

// =====================================================================
// Task 3: Upload 全分支 + GetRandomEnabled + validateFile/getImageDimensions/calculateMD5
// =====================================================================

// TestBg78_Upload_Success 真 PNG 字节 → 断言字段完整 + 落盘文件存在 + 状态用具名常量。
func TestBg78_Upload_Success(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newBg78(t)

	pngBytes, _ := makeBG78PNG(t, 300, 150)
	bg, err := svc.Upload(ctx, &UploadRequest{
		FileName:        "realbg.png",
		FileData:        pngBytes,
		FileSize:        int64(len(pngBytes)),
		PieceShape:      models.PieceShapeCircle,
		DifficultyLevel: 1,
		AllowedShapes:   []string{"circle"},
	})
	require.NoError(t, err)
	require.NotNil(t, bg)
	assert.Equal(t, 300, bg.FileWidth)
	assert.Equal(t, 150, bg.FileHeight)
	// 路径用正斜杠(filepath.ToSlash)兼容跨平台
	assert.NotContains(t, bg.FilePath, `\`, "FilePath 应使用正斜杠")
	assert.NotEmpty(t, bg.FileMD5, "FileMD5 必填")
	// MD5 验证
	sum := md5Sum(pngBytes)
	assert.Equal(t, sum, bg.FileMD5)
	// 状态用具名常量(CLAUDE.md Status 0=enabled/1=disabled — 但 CaptchaBg 约定相反,
	// 1=enabled/0=disabled;具名常量替代裸字面量,符合 78-VERIFICATION D10 约定)
	assert.Equal(t, models.CaptchaBgEnabled, bg.Status, "上传默认 status 应为 enabled")
	// 落盘文件存在
	_, err = os.Stat(bg.FilePath)
	assert.NoError(t, err, "落盘文件应真实存在")

	// DB 行存在
	var count int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_captcha_background WHERE id = ?`, bg.ID).Scan(&count).Error)
	assert.Equal(t, int64(1), count)

	// uploads/ 目录在 t.TempDir() 下,仓库工作树无污染
	// (由 svc.config.StoragePath = t.TempDir() 保证)
	_ = ctx
}

// TestBg78_Upload_ValidateFail 三种验证失败场景 + 断言目录内无新文件。
func TestBg78_Upload_ValidateFail(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newBg78(t)

	// 记录验证前的目录状态
	entriesBefore, _ := os.ReadDir(svc.config.StoragePath)

	cases := []struct {
		name     string
		req      *UploadRequest
		errMatch string
	}{
		{
			"非法扩展名.exe",
			&UploadRequest{FileName: "evil.exe", FileData: []byte("x"), FileSize: 1, PieceShape: models.PieceShapeCircle, DifficultyLevel: 1},
			"文件验证失败",
		},
		{
			"无扩展名",
			&UploadRequest{FileName: "noext", FileData: []byte("x"), FileSize: 1, PieceShape: models.PieceShapeCircle, DifficultyLevel: 1},
			"文件验证失败",
		},
		{
			"超 MaxFileSize",
			&UploadRequest{FileName: "big.png", FileData: []byte("x"), FileSize: 3 * 1024 * 1024, PieceShape: models.PieceShapeCircle, DifficultyLevel: 1},
			"文件验证失败",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Upload(ctx, tc.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMatch)
		})
	}

	// 验证失败路径绝不应写入任何文件
	entriesAfter, _ := os.ReadDir(svc.config.StoragePath)
	assert.Equal(t, len(entriesBefore), len(entriesAfter), "验证失败路径不应写入新文件")
}

// TestBg78_Upload_BadImageBytes 文件名 .png 但字节是 "not-an-image" →
// getImageDimensions 失败 → 落盘临时文件已被 os.Remove(:97)。
func TestBg78_Upload_BadImageBytes(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newBg78(t)

	entriesBefore, _ := os.ReadDir(svc.config.StoragePath)

	_, err := svc.Upload(ctx, &UploadRequest{
		FileName:        "fake.png",
		FileData:        []byte("not-an-image"),
		FileSize:        12,
		PieceShape:      models.PieceShapeCircle,
		DifficultyLevel: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "获取图片尺寸失败")

	// :97 os.Remove 清理验证
	entriesAfter, _ := os.ReadDir(svc.config.StoragePath)
	assert.Equal(t, len(entriesBefore), len(entriesAfter), "尺寸失败后落盘文件应被 os.Remove")
}

// TestBg78_Upload_DBCreateFail DROP TABLE sys_captcha_background 后上传真 PNG →
// 断言错误含"创建数据库记录失败" 且落盘文件已被 os.Remove(:124)。
func TestBg78_Upload_DBCreateFail(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newBg78(t)

	// DROP 表让 Create 必失败
	require.NoError(t, db.Exec(`DROP TABLE sys_captcha_background`).Error)

	entriesBefore, _ := os.ReadDir(svc.config.StoragePath)

	pngBytes, _ := makeBG78PNG(t, 100, 100)
	_, err := svc.Upload(ctx, &UploadRequest{
		FileName:        "ok.png",
		FileData:        pngBytes,
		FileSize:        int64(len(pngBytes)),
		PieceShape:      models.PieceShapeCircle,
		DifficultyLevel: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "创建数据库记录失败")

	// :124 os.Remove 清理验证
	entriesAfter, _ := os.ReadDir(svc.config.StoragePath)
	assert.Equal(t, len(entriesBefore), len(entriesAfter), "DB 失败后落盘文件应被 os.Remove")
}

// TestBg78_Upload_MkdirFail StoragePath 指向已存在**普通文件**路径 → MkdirAll 失败分支。
// Windows 平台 MkdirAll 对"路径是普通文件"的语义与 Unix 一致:返回 "not a directory" 类错误,
// 该分支可复现;若某平台不可复现,跳过即可(D-78-10 fallback)。
func TestBg78_Upload_MkdirFail(t *testing.T) {
	if testing.Short() {
		t.Skip("MkdirFail 路径对平台语义敏感,short 模式跳过")
	}
	ctx := context.Background()
	svc, _, _ := newBg78(t)

	// 把 StoragePath 指向一个普通文件 → MkdirAll 必失败
	conflictFile := filepath.Join(t.TempDir(), "conflict.txt")
	require.NoError(t, os.WriteFile(conflictFile, []byte("x"), 0o644))
	svc.config.StoragePath = conflictFile // MkdirAll(conflictFile) → ENOTDIR

	pngBytes, _ := makeBG78PNG(t, 50, 50)
	_, err := svc.Upload(ctx, &UploadRequest{
		FileName:        "x.png",
		FileData:        pngBytes,
		FileSize:        int64(len(pngBytes)),
		PieceShape:      models.PieceShapeCircle,
		DifficultyLevel: 1,
	})
	require.Error(t, err, "StoragePath 指向普通文件时 MkdirAll 应失败")
	assert.Contains(t, err.Error(), "创建存储目录失败")
}

// TestBg78_GetRandomEnabled_ExactMatch 2 行匹配 + 1 行 shape 不匹配 → 断言返回值 ∈ 匹配集合。
// 并断言结果被写入缓存(第二次调用命中缓存分支 :148-152,可观察证据:DROP TABLE 后第二次调用仍成功)。
func TestBg78_GetRandomEnabled_ExactMatch(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := newBg78(t)

	// 3 行:2 个 circle/1 (匹配) + 1 个 circle/2(不匹配 difficulty) + 1 个 square/1(shape 不匹配)
	require.NoError(t, db.Exec(`INSERT INTO sys_captcha_background
		(id, file_name, file_path, piece_shape, difficulty_level, status, use_count)
		VALUES
		('bg-c1-1', 'c1a.png', '/tmp/c1a.png', 'circle', 1, ?, 0),
		('bg-c1-2', 'c1b.png', '/tmp/c1b.png', 'circle', 1, ?, 0),
		('bg-c1-3', 'c1c.png', '/tmp/c1c.png', 'square', 1, ?, 0)`,
		models.CaptchaBgEnabled, models.CaptchaBgEnabled, models.CaptchaBgEnabled).Error)

	// 第 1 次:DB 查询 + 缓存写入
	bg1, err := svc.GetRandomEnabled(ctx, models.PieceShapeCircle, 1)
	require.NoError(t, err)
	require.NotNil(t, bg1)
	assert.Contains(t, []string{"bg-c1-1", "bg-c1-2"}, bg1.ID,
		"返回值应在 2 行匹配集合中")

	// 第 2 次:命中缓存分支
	bg2, err := svc.GetRandomEnabled(ctx, models.PieceShapeCircle, 1)
	require.NoError(t, err)
	require.NotNil(t, bg2)

	// DROP TABLE 后第 3 次:缓存命中,跳过 DB 查询
	require.NoError(t, db.Exec(`DROP TABLE sys_captcha_background`).Error)
	bg3, err := svc.GetRandomEnabled(ctx, models.PieceShapeCircle, 1)
	require.NoError(t, err, "缓存命中应跳过 DB 查询,DROP TABLE 后仍成功")
	require.NotNil(t, bg3)
}

// TestBg78_GetRandomEnabled_Empty 无匹配行 → 断言错误含"没有找到可用的背景图"。
// sqlite 下 s.db.Type != "postgres" 直接跳过 PG-only 的 allowed_shapes @> 分支。
// 该 PG-only 分支接受不覆盖(SUMMARY 记录 D-78-01a)。
func TestBg78_GetRandomEnabled_Empty(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newBg78(t)

	_, err := svc.GetRandomEnabled(ctx, models.PieceShapeCircle, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "没有找到可用的背景图")
	assert.NotContains(t, err.Error(), "unrecognized token")
	assert.NotContains(t, err.Error(), "syntax error")
}

// TestBg78_ValidateFile_Table 表驱动补齐边角:合法 png/jpg/jpeg / 超大小 / 非法格式 / 无扩展名。
func TestBg78_ValidateFile_Table(t *testing.T) {
	svc, _, _ := newBg78(t)

	cases := []struct {
		name     string
		fileName string
		size     int64
		wantErr  string
	}{
		{"合法 png", "bg.png", 1024, ""},
		{"合法 jpg", "bg.jpg", 1024, ""},
		{"合法 jpeg", "bg.jpeg", 1024, ""},
		{"超大小", "bg.png", 3 * 1024 * 1024, "超过限制"},
		{"非法 gif", "bg.gif", 100, "不支持的文件格式"},
		{"无扩展名", "noext", 100, "不支持的文件格式"},
		{"空文件名", "", 100, "不支持的文件格式"},
		{"单点扩展名", "bg.", 100, "不支持的文件格式"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.validateFile(tc.fileName, tc.size)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

// TestBg78_CalculateMD5 已知值"hello"的 MD5 验证。
func TestBg78_CalculateMD5(t *testing.T) {
	svc, _, _ := newBg78(t)
	assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", svc.calculateMD5([]byte("hello")))
	assert.Len(t, svc.calculateMD5([]byte("anything")), 32)
}

// TestBg78_GetImageDimensions 测两个分支:不存在文件 → 错误 / 非图片字节 → DecodeConfig 错误。
func TestBg78_GetImageDimensions(t *testing.T) {
	svc, _, _ := newBg78(t)

	// 不存在文件
	_, _, err := svc.getImageDimensions(filepath.Join(t.TempDir(), "no-such.png"))
	assert.Error(t, err)

	// 非图片字节 → DecodeConfig 错误
	bad := filepath.Join(t.TempDir(), "bad.png")
	require.NoError(t, os.WriteFile(bad, []byte("not an image"), 0o644))
	_, _, err = svc.getImageDimensions(bad)
	assert.Error(t, err)

	// 真 PNG → 正确尺寸
	pngBytes, dst := makeBG78PNG(t, 200, 100)
	w, h, err := svc.getImageDimensions(dst)
	require.NoError(t, err)
	assert.Equal(t, 200, w)
	assert.Equal(t, 100, h)
	_ = pngBytes
}

// md5Sum helper — 复用 captcha_78_01_test.go 的语义,但不导包避免循环依赖。
func md5Sum(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}
