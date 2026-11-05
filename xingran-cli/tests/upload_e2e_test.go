//go:build integration

package tests

// M8 真实后端 E2E（task-14）：工位模板下载 → 校验 magic → 上传该模板。
// 后端不在线自动 skip；凭据从环境变量读，缺失 skip；绝不硬编码凭据。

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/client"
	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/crypto"
)

// excelBaseURL Excel E2E 后端地址（默认企业环境，可用环境变量覆盖）。
func excelBaseURL() string {
	if v := os.Getenv("XINGRAN_BASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("XINGRAN_EXCEL_BASE_URL"); v != "" {
		return v
	}
	return "http://10.62.10.39:9000"
}

// probeExcelBackend 探测后端在线（public-key 公开端点），不在线 skip。
func probeExcelBackend(t *testing.T) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodGet, excelBaseURL()+"/api/v1/system/auth/public-key", nil)
	if err != nil {
		t.Skipf("构造探测请求失败: %v", err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Skipf("后端不可达(%s): %v —— 跳过 Excel E2E", excelBaseURL(), err)
	}
	defer resp.Body.Close()
}

// excelClientFor 构造经真实登录 token 的上传/下载客户端。
// 凭据与 token 文件全部用临时路径隔离，不污染真实 ~/.xingran-cli。
// 装配方式与 cmd/root.go 一致（crypto.Encryptor + auth.HTTPRequester + FileTokenStore）。
func excelClientFor(t *testing.T) UploadDownloader {
	t.Helper()
	user, pass, ok := testCreds()
	if !ok {
		t.Skip("未设置 XINGRAN_TEST_USER / XINGRAN_TEST_PASS")
	}

	// 验证码模式探测：非 disabled 需要 OCR/滑块注入，为避免锁号直接 skip
	if mode := captchaMode(t); mode != "disabled" {
		t.Skipf("后端验证码为 %s 模式，完整登录需注入验证码答案（见 TestLoginE2E），Excel E2E skip", mode)
	}

	base := excelBaseURL()
	enc := crypto.NewEncryptor()
	httpClient := &http.Client{Timeout: 60 * time.Second}
	authReq := &auth.HTTPRequester{BaseURL: base, Encryptor: enc, HTTPClient: httpClient}
	store := &auth.FileTokenStore{Path: tempTokenPath(t)}
	authn := auth.NewAuthenticator(authReq, store, enc, base)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)

	pair, err := authn.Login(ctx, user, pass, auth.CaptchaInput{Type: auth.CaptchaTypeNone})
	if err != nil {
		t.Skipf("登录失败（服务端原文透传）: %v", err)
	}
	t.Logf("登录成功 expiresIn=%ds", pair.ExpiresIn)
	t.Cleanup(func() { _ = authn.Logout(context.Background()) })

	c := client.New(client.Options{
		BaseURL:       base,
		HTTPClient:    httpClient,
		Authenticator: authn,
	})
	return mustUploadDownloader(c)
}

// TestExcelWorkstationTemplateAndImport 工位模板下载 → 校验 magic → 上传模板。
func TestExcelWorkstationTemplateAndImport(t *testing.T) {
	probeExcelBackend(t)
	c := excelClientFor(t)
	ctx := context.Background()

	// 1. 下载工位模板
	tpl := filepath.Join(t.TempDir(), "工位模板.xlsx")
	saved, err := c.DoDownload(ctx, "/ops/workstation/template", "", tpl, "GET")
	if err != nil {
		t.Fatalf("模板下载失败: %v", err)
	}
	data, err := os.ReadFile(saved)
	if err != nil {
		t.Fatalf("模板文件不可读: %v", err)
	}
	if len(data) < 4 || !bytes.HasPrefix(data, xlsxMagic) {
		t.Fatalf("模板非 xlsx（magic 不符）: %d bytes", len(data))
	}
	t.Logf("模板下载成功: %s (%d bytes)", saved, len(data))

	// 2. 上传该模板（模板自带示例行，允许 failed>0，只要 code=0）
	resp, err := c.DoUpload(ctx, "/ops/workstation/import", "file", saved, nil)
	if err != nil {
		t.Fatalf("模板上传失败: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("上传业务码非 0: %+v", resp)
	}
	t.Logf("上传返回: %s", string(resp.Data))
}

// TestExcelDeptMappingTemplate 工位部门映射模板下载。
func TestExcelDeptMappingTemplate(t *testing.T) {
	probeExcelBackend(t)
	c := excelClientFor(t)

	out := filepath.Join(t.TempDir(), "dept-mapping-template.xlsx")
	if _, err := c.DoDownload(context.Background(),
		"/ops/workstation/dept-mapping-template", "", out, "GET"); err != nil {
		t.Fatalf("部门映射模板下载失败: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !bytes.HasPrefix(data, xlsxMagic) {
		t.Fatalf("部门映射模板非 xlsx")
	}
	t.Logf("部门映射模板下载成功: %d bytes", len(data))
}

// TestExcelExportAllEntities 8 类实体 export 全部可下载且为 xlsx（只读操作，安全）。
func TestExcelExportAllEntities(t *testing.T) {
	probeExcelBackend(t)
	c := excelClientFor(t)
	ctx := context.Background()

	entities := []string{
		"building", "floor", "workstation", "asset",
		"infoPoint", "serverRoom", "dedicatedLine", "roomDevice",
	}
	for _, e := range entities {
		e := e
		t.Run(e, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), e+"-export.xlsx")
			saved, err := c.DoDownload(ctx, "/ops/"+e+"/export", "{}", out, "POST")
			if err != nil {
				t.Fatalf("%s export 失败: %v", e, err)
			}
			data, err := os.ReadFile(saved)
			if err != nil {
				t.Fatalf("%s 导出文件不可读: %v", e, err)
			}
			if !bytes.HasPrefix(data, xlsxMagic) {
				t.Errorf("%s 导出非 xlsx", e)
			}
			t.Logf("%s export: %d bytes", e, len(data))
		})
	}
}
