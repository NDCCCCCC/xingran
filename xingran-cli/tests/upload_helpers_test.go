package tests

// M8（task-14）Excel 上传/下载测试公共设施。
//
// 契约见 CONTRACT-M8.md，实现为 internal/client/upload.go（client-dev task-13 交付）：
//
//	DoUpload(ctx, path, fileField, filePath string, fields map[string]string) (*client.Response, error)
//	DoDownload(ctx, path, jsonBody, destPath string, method string) (savedPath string, err error)
//
// 两个方法在 clientImpl（client.New 返回值的具体类型）上而非 Client 接口上，
// 测试经 UploadDownloader 接口断言取用（uploadClientFor）。

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/client"
)

// UploadDownloader M8 上传/下载能力契约（CONTRACT-M8.md 签名）。
type UploadDownloader interface {
	DoUpload(ctx context.Context, path, fileField, filePath string, fields map[string]string) (*client.Response, error)
	DoDownload(ctx context.Context, path, jsonBody, destPath string, method string) (savedPath string, err error)
}

// xlsxMagic xlsx 文件 magic 字节（ZIP 容器，OOXML PK 0x50 0x4B 0x03 0x04）。
var xlsxMagic = []byte{0x50, 0x4B, 0x03, 0x04}

// mustUploadDownloader 编译/运行期契约断言。
func mustUploadDownloader(c client.Client) UploadDownloader {
	ud, ok := c.(UploadDownloader)
	if !ok {
		panic("client.Client 未实现 DoUpload/DoDownload 契约（CONTRACT-M8.md）")
	}
	return ud
}

// mockAuthenticator 测试鉴权：固定 token；Refresh 时切换到 nextToken（可编程）。
type mockAuthenticator struct {
	token     string
	nextToken string // Refresh 后生效；空则 Refresh 失败
	refreshed int
}

func (m *mockAuthenticator) Login(context.Context, string, string, auth.CaptchaInput) (*auth.TokenPair, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockAuthenticator) Refresh(context.Context) (*auth.TokenPair, error) {
	if m.nextToken == "" {
		return nil, errors.New("refresh 失败（未配置 nextToken）")
	}
	m.token = m.nextToken
	m.refreshed++
	return &auth.TokenPair{AccessToken: m.token, RefreshToken: "rt", ExpiresIn: 300}, nil
}

func (m *mockAuthenticator) GetAccessToken(context.Context) (string, error) {
	return m.token, nil
}

func (m *mockAuthenticator) Logout(context.Context) error { return nil }

// writeTempXlsx 写一个带 xlsx magic 的临时文件。
func writeTempXlsx(t *testing.T, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "data.xlsx")
	data := append([]byte{}, xlsxMagic...)
	data = append(data, content...)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// uploadClientFor 以给定 HTTP client 与鉴权构造上传/下载客户端。
func uploadClientFor(baseURL string, hc *http.Client, au client.Authenticator) UploadDownloader {
	c := client.New(client.Options{
		BaseURL:       baseURL,
		HTTPClient:    hc,
		Authenticator: au,
	})
	return mustUploadDownloader(c)
}
