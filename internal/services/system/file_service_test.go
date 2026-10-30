package system

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models/system"
)

// =====================================================================
// Phase 74-07: file_service.go 全量测试。上传走 t.TempDir() 真实落盘,
// PNG 由 image/png 现场编码,multipart.FileHeader 与 handler 入口同构。
// =====================================================================

func newFileTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:filesvc_"+t.Name()+"?mode=memory&cache=shared&_enable_boolean=true&_busy_timeout=5000"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&system.SysFile{}, &system.SysFileAccessLog{}))
	return db
}

func newFileSvc(t *testing.T) *FileService {
	t.Helper()
	svc := NewFileService(newFileTestDB(t))
	svc.uploadBaseDir = t.TempDir() // 同包改私有字段,避免污染仓库目录
	return svc
}

// tinyPNG 现场编码 w×h PNG 字节流。
func tinyPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))))
	return buf.Bytes()
}

// fileHeaderFor 构造 *multipart.FileHeader(Content-Type 可指定)。
func fileHeaderFor(t *testing.T, data []byte, filename, contentType string) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = fw.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	form, err := multipart.NewReader(&buf, w.Boundary()).ReadForm(32 << 20)
	require.NoError(t, err)
	require.Len(t, form.File["file"], 1)
	fh := form.File["file"][0]
	if contentType != "" {
		fh.Header.Set("Content-Type", contentType)
	}
	return fh
}

// memFile 把 bytes.Reader 适配成 multipart.File(Read/ReadAt/Seek 由 Reader 提供)。
type memFile struct{ *bytes.Reader }

func (memFile) Close() error { return nil }

// failReader 注入读错误,覆盖 calculateFileHash 失败分支。
type failReader struct{}

func (failReader) Read([]byte) (int, error)       { return 0, assert.AnError }
func (failReader) ReadAt([]byte, int64) (int, error) { return 0, assert.AnError }
func (failReader) Seek(int64, int) (int64, error) { return 0, nil }
func (failReader) Close() error                   { return nil }

func TestFileService_ConfigsAndPureHelpers(t *testing.T) {
	// 分类配置:命中 / 未知回退 image 默认
	cfg := GetCategoryConfig("avatar")
	require.NotNil(t, cfg)
	assert.Equal(t, int64(2*megabyte), cfg.MaxSize)
	assert.True(t, cfg.ExtractDimensions)
	fallback := GetCategoryConfig("nope")
	assert.Equal(t, CategoryConfigs["image"].MaxSize, fallback.MaxSize)

	// 校验转换:已配置分类 → ext map;未配置分类 → ImageValidation
	v := GetValidationByCategory("import")
	assert.True(t, v.AllowedExts[".xlsx"])
	assert.False(t, v.AllowedExts[".png"])
	fallbackV := GetValidationByCategory("nope")
	assert.Equal(t, ImageValidation.MaxSize, fallbackV.MaxSize)
	assert.True(t, fallbackV.AllowedExts[".webp"])
	doc := CategoryConfigs["document"]
	dv := doc.toFileValidation()
	assert.True(t, dv.AllowedExts[".pdf"])

	assert.True(t, isImageFile(".png"))
	assert.False(t, isImageFile(".pdf"))

	meta := buildImageMetadata(640, 480)
	require.NotNil(t, meta)
	assert.Contains(t, *meta, `"width":640`)

	// 哈希:正常 + 读错误
	h1, err := calculateFileHash(memFile{bytes.NewReader([]byte("hello"))})
	require.NoError(t, err)
	h2, _ := calculateFileHash(memFile{bytes.NewReader([]byte("hello"))})
	h3, _ := calculateFileHash(memFile{bytes.NewReader([]byte("world"))})
	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
	assert.Len(t, h1, 64)
	_, err = calculateFileHash(failReader{})
	require.ErrorContains(t, err, "计算文件哈希失败")

	// 尺寸提取:真 PNG / 打不开 / 非图片
	dir := t.TempDir()
	p := filepath.Join(dir, "a.png")
	require.NoError(t, writeFileForTest(p, tinyPNG(t, 3, 5)))
	w, h, err := extractImageDimensions(p)
	require.NoError(t, err)
	assert.Equal(t, 3, w)
	assert.Equal(t, 5, h)
	_, _, err = extractImageDimensions(filepath.Join(dir, "missing.png"))
	require.Error(t, err)
	np := filepath.Join(dir, "not.png")
	require.NoError(t, writeFileForTest(np, []byte("plain text")))
	_, _, err = extractImageDimensions(np)
	require.Error(t, err)
}

func TestFileService_UploadFile(t *testing.T) {
	svc := newFileSvc(t)
	ctx := context.Background()

	pngBytes := tinyPNG(t, 3, 5)

	// 超大小限制
	big := GetValidationByCategory("avatar")
	big.MaxSize = 2
	_, err := svc.UploadFile(ctx, fileHeaderFor(t, pngBytes, "a.png", "image/png"), "avatar", "u1", big)
	require.ErrorContains(t, err, "文件大小超过限制")

	// 非法扩展名
	_, err = svc.UploadFile(ctx, fileHeaderFor(t, []byte("x"), "a.exe", ""), "avatar", "u1", GetValidationByCategory("avatar"))
	require.ErrorContains(t, err, "不支持的文件类型: .exe")

	// 图片成功上传:尺寸提取 + metadata
	f1, err := svc.UploadFile(ctx, fileHeaderFor(t, pngBytes, "pic.PNG", "image/png"), "avatar", "u1", nil)
	require.NoError(t, err)
	require.NotNil(t, f1.Width)
	require.NotNil(t, f1.Height)
	assert.Equal(t, 3, *f1.Width)
	assert.Equal(t, 5, *f1.Height)
	require.NotNil(t, f1.Metadata)
	assert.Contains(t, *f1.Metadata, `"height":5`)
	assert.Equal(t, ".png", f1.Extension, "扩展名应转小写")
	assert.Equal(t, "image/png", f1.FileType)

	// 相同内容二次上传 → 秒传返回已有记录
	var before int64
	require.NoError(t, svc.db.Model(&system.SysFile{}).Count(&before).Error)
	f2, err := svc.UploadFile(ctx, fileHeaderFor(t, pngBytes, "dup.png", "image/png"), "avatar", "u2", nil)
	require.NoError(t, err)
	assert.Equal(t, f1.ID, f2.ID)
	var after int64
	require.NoError(t, svc.db.Model(&system.SysFile{}).Count(&after).Error)
	assert.Equal(t, before, after, "秒传不应新增记录")

	// document 类:不做尺寸提取
	txtHdr := fileHeaderFor(t, []byte("note"), "a.txt", "text/plain")
	f3, err := svc.UploadFile(ctx, txtHdr, "document", "u1", GetValidationByCategory("document"))
	require.NoError(t, err)
	assert.Nil(t, f3.Width)
	assert.Nil(t, f3.Metadata)
	assert.True(t, strings.HasPrefix(f3.StoragePath, "document"), "存储路径应含分类子目录")
}

func TestFileService_GetDeleteList(t *testing.T) {
	svc := newFileSvc(t)
	ctx := context.Background()

	// 直接造记录(不经上传)
	mk := func(name, biz, uploader string) *system.SysFile {
		f := &system.SysFile{FileName: name, FileSize: 1, StoragePath: biz + "/x", FileHash: name, UploaderID: uploader, BusinessType: biz}
		require.NoError(t, svc.db.Create(f).Error)
		return f
	}
	a := mk("a.png", "avatar", "u1")
	mk("b.png", "avatar", "u2")
	mk("c.txt", "document", "u1")

	// GetFile:命中 / 不存在
	got, err := svc.GetFile(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "a.png", got.FileName)
	_, err = svc.GetFile(ctx, "ghost")
	require.ErrorContains(t, err, "文件不存在")

	// DeleteFile:软删 + 访问日志
	require.NoError(t, svc.DeleteFile(ctx, a.ID))
	_, err = svc.GetFile(ctx, a.ID)
	require.Error(t, err, "软删后不可见")
	var logs []*system.SysFileAccessLog
	require.NoError(t, svc.db.Where("file_id = ?", a.ID).Find(&logs).Error)
	assert.Len(t, logs, 1, "DeleteFile 应记录 delete 日志")

	// BatchDeleteFiles:空列表 / 命中
	require.ErrorContains(t, svc.BatchDeleteFiles(ctx, nil), "不能为空")
	require.NoError(t, svc.BatchDeleteFiles(ctx, []string{"ghost-id"}))

	// ListFiles:全量 / businessType / userID / 分页
	files, total, err := svc.ListFiles(ctx, "", "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, files, 2)
	_, total, err = svc.ListFiles(ctx, "document", "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	_, total, err = svc.ListFiles(ctx, "", "u1", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	files, _, err = svc.ListFiles(ctx, "", "", 1, 1)
	require.NoError(t, err)
	assert.Len(t, files, 1, "offset=1 limit=1")
}

func TestFileService_PathsLogsAndCleanup(t *testing.T) {
	svc := newFileSvc(t)
	ctx := context.Background()

	// 路径 helpers
	f := &system.SysFile{StoragePath: "avatar/x.png"}
	assert.Equal(t, filepath.Join(svc.uploadBaseDir, "avatar/x.png"), svc.GetFilePath(f))
	assert.Equal(t, "/uploads/avatar/x.png", svc.GetFileURL(f))

	// LogAccess / GetAccessLogs
	require.NoError(t, svc.LogAccess(ctx, "f1", "view", "u1", "alice", "127.0.0.1"))
	require.NoError(t, svc.LogAccess(ctx, "f1", "download", "u1", "alice", "127.0.0.1"))
	require.NoError(t, svc.LogAccess(ctx, "f2", "view", "u2", "bob", "::1"))
	logs, total, err := svc.GetAccessLogs(ctx, "f1", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)
	_, total, err = svc.GetAccessLogs(ctx, "f2", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// CleanupDeletedFiles:两条软删记录,磁盘上只放一个真文件 → count=1
	old := time.Now().AddDate(0, 0, -10)
	livePath := filepath.Join(svc.uploadBaseDir, "avatar")
	require.NoError(t, writeFileForTest(filepath.Join(livePath, "live.png"), []byte("x")))
	svc.db.Create(&system.SysFile{
		FileName: "live.png", StoragePath: "avatar/live.png", FileHash: "h1",
		IsDeleted: true, DeleteTime: &old,
	})
	svc.db.Create(&system.SysFile{
		FileName: "gone.png", StoragePath: "avatar/gone.png", FileHash: "h2",
		IsDeleted: true, DeleteTime: &old,
	})
	// 未删除的记录不应被清理
	svc.db.Create(&system.SysFile{FileName: "keep.png", StoragePath: "avatar/keep.png", FileHash: "h3"})

	count, err := svc.CleanupDeletedFiles(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "仅磁盘上真实存在的文件计入清理")

	var remain int64
	require.NoError(t, svc.db.Model(&system.SysFile{}).Where("is_deleted = ?", true).Count(&remain).Error)
	assert.Zero(t, remain, "清理后软删记录应被移除")
	var kept int64
	require.NoError(t, svc.db.Model(&system.SysFile{}).Where("file_name = ?", "keep.png").Count(&kept).Error)
	assert.Equal(t, int64(1), kept)
}

// writeFileForTest 写测试文件并确保父目录存在。
func writeFileForTest(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
