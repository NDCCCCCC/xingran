package tests

// M8 单测（无 build tag，httptest mock，不依赖真后端）。
// 测试对象：internal/client/upload.go 的 DoUpload/DoDownload（经 UploadDownloader
// 契约接口断言取用，mock Authenticator 注入鉴权）。
// 验证：multipart 形状、Bearer、二进制落盘 magic、JSON 错误不落盘、401 自动 refresh 重试。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUploadMultipartContract 上传契约：Content-Type、字段名 file、文件内容完整、Bearer 头。
func TestUploadMultipartContract(t *testing.T) {
	var gotContentType, gotAuth string
	var gotFileField, gotFilename string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			http.Error(w, "not multipart", 400)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", 400)
			return
		}
		gotFileField = "file"
		gotFilename = header.Filename
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		gotBody = buf[:n]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"inserted":1,"updated":0,"failed":0,"errors":[]},"request_id":"r1"}`))
	}))
	defer srv.Close()

	xl := writeTempXlsx(t, []byte("workbook-bytes"))
	ud := uploadClientFor(srv.URL, srv.Client(), &mockAuthenticator{token: "tok-upload-1"})
	resp, err := ud.DoUpload(context.Background(), "/ops/workstation/import", "file", xl, nil)
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("业务码非 0: %+v", resp)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Fatalf("Content-Type 应为 multipart/form-data: %q", gotContentType)
	}
	if gotFileField != "file" {
		t.Fatal("字段名应为 file")
	}
	if gotFilename != "data.xlsx" {
		t.Fatalf("filename 应为原文件名: %q", gotFilename)
	}
	if gotAuth != "Bearer tok-upload-1" {
		t.Fatalf("Authorization 头不正确: %q", gotAuth)
	}
	if !bytes.HasPrefix(gotBody, xlsxMagic) || string(gotBody[4:]) != "workbook-bytes" {
		t.Fatalf("文件内容不完整: %q", gotBody)
	}
}

// TestUpload401RefreshRetry 上传遇 401：自动 refresh 后重放成功。
func TestUpload401RefreshRetry(t *testing.T) {
	au := &mockAuthenticator{token: "stale-token", nextToken: "fresh-token"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer stale-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"message":"unauthorized"}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{}}`))
	}))
	defer srv.Close()

	xl := writeTempXlsx(t, nil)
	ud := uploadClientFor(srv.URL, srv.Client(), au)
	resp, err := ud.DoUpload(context.Background(), "/ops/workstation/import", "file", xl, nil)
	if err != nil {
		t.Fatalf("刷新重试后上传应成功: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("业务码非 0: %+v", resp)
	}
	if au.refreshed != 1 {
		t.Fatalf("应刷新一次: refreshed=%d", au.refreshed)
	}
}

// TestUploadImportErrorsResponse 解析逐行 errors 的 import 响应结构。
func TestUploadImportErrorsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"inserted":0,"updated":0,"failed":2,"errors":[{"row":2,"message":"引用的 所属楼宇 不存在"},{"row":3,"message":"部门不存在"}]},"request_id":"r2"}`))
	}))
	defer srv.Close()

	xl := writeTempXlsx(t, nil)
	ud := uploadClientFor(srv.URL, srv.Client(), &mockAuthenticator{token: "t"})
	resp, err := ud.DoUpload(context.Background(), "/ops/workstation/import", "file", xl, nil)
	if err != nil {
		t.Fatalf("上传失败: %v", err)
	}
	if resp.Code != 0 {
		t.Fatal("逐行错误不应导致 code!=0")
	}
	var data struct {
		Failed int `json:"failed"`
		Errors []struct {
			Row     int    `json:"row"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("解析 import data 失败: %v", err)
	}
	if data.Failed != 2 || len(data.Errors) != 2 || data.Errors[0].Message == "" {
		t.Fatalf("逐行 errors 结构不正确: %+v", data)
	}
}

// TestDownloadBinaryToDisk 下载：destPath 落盘、magic 正确。
func TestDownloadBinaryToDisk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="workstation-template.xlsx"`)
		_, _ = w.Write(append([]byte{}, append(xlsxMagic, []byte("sheet-data")...)...))
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "template.xlsx")
	ud := uploadClientFor(srv.URL, srv.Client(), &mockAuthenticator{token: "t"})
	saved, err := ud.DoDownload(context.Background(), "/ops/workstation/template", "", out, "GET")
	if err != nil {
		t.Fatalf("下载失败: %v", err)
	}
	if saved != out {
		t.Fatalf("savedPath 应为 destPath: %q", saved)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("落盘文件不可读: %v", err)
	}
	if !bytes.HasPrefix(data, xlsxMagic) || string(data[4:]) != "sheet-data" {
		t.Fatalf("落盘内容不正确: %q", data)
	}
}

// TestDownloadErrorNotWritten code!=0 的 JSON 错误响应不得落盘成 xlsx。
func TestDownloadErrorNotWritten(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":500,"message":"内部错误","data":null}`))
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "should-not-exist.xlsx")
	ud := uploadClientFor(srv.URL, srv.Client(), &mockAuthenticator{token: "t"})
	_, err := ud.DoDownload(context.Background(), "/ops/workstation/template", "", out, "GET")
	if err == nil {
		t.Fatal("错误响应应返回 error")
	}
	if _, serr := os.Stat(out); !os.IsNotExist(serr) {
		t.Fatalf("JSON 错误响应不得落盘: %s", out)
	}
}

// TestDownload401RefreshRetry 下载遇 401：刷新后重试成功。
func TestDownload401RefreshRetry(t *testing.T) {
	au := &mockAuthenticator{token: "stale", nextToken: "fresh"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer stale" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401,"message":"unauthorized"}`))
			return
		}
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "t.xlsx")
	ud := uploadClientFor(srv.URL, srv.Client(), au)
	if _, err := ud.DoDownload(context.Background(), "/ops/workstation/template", "", out, "GET"); err != nil {
		t.Fatalf("刷新重试后下载应成功: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !bytes.HasPrefix(data, xlsxMagic) {
		t.Fatalf("落盘内容不正确: %q", data)
	}
}

// TestDownloadExportPost export 是 POST+JSON body → xlsx。
func TestDownloadExportPost(t *testing.T) {
	var gotMethod, gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotCT = r.Header.Get("Content-Type")
		buf := make([]byte, 256)
		n, _ := r.Body.Read(buf)
		gotBody = string(buf[:n])
		_, _ = w.Write(xlsxMagic)
	}))
	defer srv.Close()

	out := filepath.Join(t.TempDir(), "export.xlsx")
	ud := uploadClientFor(srv.URL, srv.Client(), &mockAuthenticator{token: "t"})
	if _, err := ud.DoDownload(context.Background(), "/ops/workstation/export", "{}", out, "POST"); err != nil {
		t.Fatalf("export 下载失败: %v", err)
	}
	if gotMethod != "POST" || gotCT != "application/json" || gotBody != "{}" {
		t.Fatalf("export 请求形状不正确: %s %s %q", gotMethod, gotCT, gotBody)
	}
}
