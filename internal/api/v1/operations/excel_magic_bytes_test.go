// Package operations 提供 P1-S7 (Excel magic bytes) 的回归测试
//
// 背景: 之前仅按扩展名校验 .xlsx,攻击者可改后缀绕过(例如上传 .exe 改名为 .xlsx)。
// P1 fix (commit 2c74c06): importData 中增加 verifyExcelMagicBytes 三重校验之一,
// 读取文件前 4 字节验证 OOXML/ZIP 魔数 (PK\x03\x04)。
//
// 验证:
//   - 非 PK\x03\x04 魔数应被拒绝
//   - 合法 PK\x03\x04 魔数应通过
//   - 文件长度不足 4 字节应被拒绝
package operations

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// makeMultipartFileHeader 构造一个 *multipart.FileHeader,内容来自 body 字节。
// 使用 multipart.Writer 生成符合 RFC 7578 的 part header,这样 fileHeader.Open() 能正确读取。
func makeMultipartFileHeader(t *testing.T, filename string, body []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// 设置 filename 头 (MUST 用 textproto 编码引号内的 filename*)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		`form-data; name="file"; filename=`+strconvQuote(filename))
	h.Set("Content-Type", "application/octet-stream")
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("multipart.CreatePart failed: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("multipart part write failed: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("multipart.Close failed: %v", err)
	}

	// 解析 multipart,从 body 中取出 file part
	req, err := http.NewRequest("POST", "/", &buf)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatalf("ParseMultipartForm failed: %v", err)
	}
	_, hdr, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}
	return hdr
}

// strconvQuote 是 strconv.Quote 的简化版,避免引入 strconv 包的 import 警告
func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// TestVerifyExcelMagicBytes_RejectsNonPK 验证非 PK 魔数被拒绝
func TestVerifyExcelMagicBytes_RejectsNonPK(t *testing.T) {
	tests := []struct {
		name        string
		body        []byte
		wantError   bool
		description string
	}{
		{
			name:        "MZ_DOS_header",
			body:        []byte{'M', 'Z', 0x00, 0x00}, // DOS/PE 可执行文件头
			wantError:   true,
			description: "Windows .exe file (MZ header) should be rejected",
		},
		{
			name:        "PDF_header",
			body:        []byte{'%', 'P', 'D', 'F'}, // PDF 文件头
			wantError:   true,
			description: "PDF file should be rejected",
		},
		{
			name:        "PNG_header",
			body:        []byte{0x89, 'P', 'N', 'G'},
			wantError:   true,
			description: "PNG file should be rejected",
		},
		{
			name:        "ZIP_but_not_PK",
			body:        []byte{'P', 'K', 0x05, 0x06}, // ZIP end-of-central-dir (not local file header)
			wantError:   true,
			description: "ZIP end-of-central-dir should be rejected (only PK\\x03\\x04 is OOXML)",
		},
		{
			name:        "random_bytes",
			body:        []byte{0xFF, 0xFE, 0xFD, 0xFC},
			wantError:   true,
			description: "Random bytes should be rejected",
		},
		{
			name:        "empty_file",
			body:        []byte{},
			wantError:   true,
			description: "Empty file should be rejected (cannot read 4 bytes)",
		},
		{
			name:        "short_file_3bytes",
			body:        []byte{0x50, 0x4B, 0x03},
			wantError:   true,
			description: "File shorter than 4 bytes should be rejected",
		},
		{
			name: "valid_PK_magic",
			body: []byte{0x50, 0x4B, 0x03, 0x04, // PK\x03\x04
				0x14, 0x00, 0x00, 0x00, 0x08, 0x00}, // ZIP local file header continuation
			wantError:   false,
			description: "Valid PK\\x03\\x04 OOXML/ZIP header should pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := makeMultipartFileHeader(t, "test.xlsx", tt.body)
			err := verifyExcelMagicBytes(hdr)
			if tt.wantError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestVerifyExcelMagicBytes_AcceptsValidXLSX 验证合法 .xlsx 通过校验
//
// 构造一个最小的 .xlsx 内容 (PK 头 + 一些字节),确认 verifyExcelMagicBytes 通过。
func TestVerifyExcelMagicBytes_AcceptsValidXLSX(t *testing.T) {
	// 模拟 .xlsx 开头 — 真实 .xlsx 在 PK\x03\x04 后是 zip 局部文件头
	body := []byte{0x50, 0x4B, 0x03, 0x04}
	body = append(body, make([]byte, 100)...) // 填充 100 字节

	hdr := makeMultipartFileHeader(t, "valid.xlsx", body)
	err := verifyExcelMagicBytes(hdr)
	assert.NoError(t, err, "valid .xlsx header should pass magic byte check")
}

// TestVerifyExcelMagicBytes_RejectsEXEWithXLSXExtension 验证扩展名绕过被阻止
//
// 即使文件名是 .xlsx,文件内容是 .exe,魔数校验应拒绝。
func TestVerifyExcelMagicBytes_RejectsEXEWithXLSXExtension(t *testing.T) {
	exeBody := []byte{'M', 'Z', 0x90, 0x00, 0x03, 0x00} // DOS MZ header
	hdr := makeMultipartFileHeader(t, "evil.xlsx", exeBody)
	err := verifyExcelMagicBytes(hdr)
	assert.Error(t, err, "EXE content with .xlsx extension should be rejected by magic check")
}

// TestVerifyExcelMagicBytes_ReaderIntegration 验证 read+reset 集成
//
// 验证 verifyExcelMagicBytes 读取前 4 字节后,文件指针位置合理(不会破坏后续读取)。
// 这是 P1-S7 fix 中的隐性合约:调用 excelize.OpenReader 仍能读取全部内容。
func TestVerifyExcelMagicBytes_ReaderIntegration(t *testing.T) {
	body := []byte{0x50, 0x4B, 0x03, 0x04}
	body = append(body, []byte("rest of file content for follow-up reading")...)

	hdr := makeMultipartFileHeader(t, "test.xlsx", body)
	err := verifyExcelMagicBytes(hdr)
	assert.NoError(t, err)

	// 重新打开验证剩余内容可读
	f, err := hdr.Open()
	if err != nil {
		t.Fatalf("failed to re-open file: %v", err)
	}
	defer f.Close()
	rest, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read all failed: %v", err)
	}
	assert.GreaterOrEqual(t, len(rest), 4, "should be able to read all bytes after magic check")
	assert.Equal(t, body, rest, "file contents should be unchanged after magic check")
}
