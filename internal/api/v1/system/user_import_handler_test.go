package system

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// Phase 72 W2 计划 72-05: User 导入 handler 测试。
//
// user_import_handler.go 的核心方法（ImportUser / DownloadImportTemplate / SyncManagers）依赖
// h.core.DB / h.core.PwdManager / h.core.DataCacheService / h.userADSyncService 等。
// 由于 CaptchaService / DB / Cache 等都需真实环境,Phase 72 主要覆盖：
//   - verifyExcelMagicBytes（纯函数,需要文件 magic 字节验证）
//   - userExcelUploadSizeLimit 常量
//   - isValidExcelFile 已在 user_handler_test.go 覆盖
//   - ImportUser 错误路径（unauthorized / 缺文件 / 错扩展名 / 大小超限 / magic 错）
//
// ImportUser happy path 需要完整 core + excelize + DB,跳过避免 scope creep。

// TestVerifyExcelMagicBytes_Valid 验证 OOXML magic bytes (PK\x03\x04) 接受。
func TestVerifyExcelMagicBytes_Valid(t *testing.T) {
	// 构造一个 multipart 文件 header,内容是合法的 xlsx magic + 一些字节
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	mimeType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="test.xlsx"`)
	hdr.Set("Content-Type", mimeType)
	part, err := writer.CreatePart(hdr)
	assert.NoError(t, err)
	// PK\x03\x04 + 一些字节
	xlsxMagic := []byte{0x50, 0x4B, 0x03, 0x04, 0x14, 0x00, 0x06, 0x00}
	_, _ = part.Write(xlsxMagic)
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, fileHeader, err := req.FormFile("file")
	assert.NoError(t, err)

	err = verifyExcelMagicBytes(fileHeader)
	assert.NoError(t, err, "valid xlsx magic should pass")
}

// TestVerifyExcelMagicBytes_Invalid 验证非法 magic bytes 返回错误。
func TestVerifyExcelMagicBytes_Invalid(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="test.xlsx"`)
	hdr.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	part, err := writer.CreatePart(hdr)
	assert.NoError(t, err)
	// 错误 magic:不是 PK\x03\x04
	_, _ = part.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) // PNG magic
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, fileHeader, err := req.FormFile("file")
	assert.NoError(t, err)

	err = verifyExcelMagicBytes(fileHeader)
	assert.Error(t, err, "invalid magic should fail")
	assert.Contains(t, err.Error(), "魔数错误")
}

// TestVerifyExcelMagicBytes_TooShort 验证文件过短无法读取 4 字节。
func TestVerifyExcelMagicBytes_TooShort(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="test.xlsx"`)
	hdr.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	part, err := writer.CreatePart(hdr)
	assert.NoError(t, err)
	_, _ = part.Write([]byte{0x50, 0x4B}) // 只有 2 字节
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, fileHeader, err := req.FormFile("file")
	assert.NoError(t, err)

	err = verifyExcelMagicBytes(fileHeader)
	assert.Error(t, err, "too short should fail")
}

// TestUserExcelUploadSizeLimit 验证常量值合理(50MB)。
func TestUserExcelUploadSizeLimit(t *testing.T) {
	assert.Equal(t, int64(50*1024*1024), userExcelUploadSizeLimit,
		"user import size limit must be 50MB")
	assert.Greater(t, userExcelUploadSizeLimit, int64(1024*1024),
		"limit must be > 1MB")
}

// TestImportUser_Unauthorized 验证无 user_id 上下文 → 401。
func TestImportUser_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// userADSyncService=nil,core=有但DataCacheService=nil → 不走 ExcelService 路径
	h := newImportTestHandler(t)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		// 不设置 user_id → 模拟未登录
		h.ImportUser(c)
	})

	req := httptest.NewRequest("POST", "/import", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "missing user_id → unauthorized")
	assert.Equal(t, http.StatusUnauthorized, w.Code, "code=401")
}

// TestImportUser_NoFile 验证未提供文件 → 400。
func TestImportUser_NoFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImportTestHandler(t)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		h.ImportUser(c)
	})

	// multipart 但没 file 字段
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("notfile", "x")
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "no file → bad request")
}

// TestImportUser_BadExtension 验证非 .xlsx 扩展名 → 400。
func TestImportUser_BadExtension(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImportTestHandler(t)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		h.ImportUser(c)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="data.xls"`)
	hdr.Set("Content-Type", "application/octet-stream")
	part, _ := writer.CreatePart(hdr)
	_, _ = part.Write([]byte{0x50, 0x4B, 0x03, 0x04}) // 任意字节
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, ".xls should be rejected")
	assert.Contains(t, w.Body.String(), "xlsx", "error msg mentions xlsx")
}

// TestImportUser_BadMagic 验证 .xlsx 扩展名但 magic 错误 → 400。
func TestImportUser_BadMagic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImportTestHandler(t)

	router := gin.New()
	router.POST("/import", func(c *gin.Context) {
		c.Set("user_id", "test-user-id")
		h.ImportUser(c)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="data.xlsx"`)
	hdr.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	part, _ := writer.CreatePart(hdr)
	_, _ = part.Write([]byte("not a real xlsx")) // 错误 magic
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "bad magic → bad request")
}

// TestSyncManagers_NoADService 验证 userADSyncService=nil → 服务端错误。
func TestSyncManagers_NoADService(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := newImportTestHandler(t)
	// 不设置 userADSyncService

	router := gin.New()
	router.POST("/sync-managers", h.SyncManagers)

	req := httptest.NewRequest("POST", "/sync-managers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "no AD service → error")
}

// newImportTestHandler 构造 UserHandler 用于导入路径测试。
// 用 nil DB 创建 UserService (handler 在 ExcelService.ImportData 之前 fail,
// 不会触发 service 调用)。
func newImportTestHandler(t *testing.T) *UserHandler {
	mockCore := &core.Core{
		CoreInfra:    &core.CoreInfra{},
		CoreServices: &core.CoreServices{},
	}
	svc := systemServices.NewUserService(nil, nil)
	return NewUserHandler(svc).WithCore(mockCore)
}