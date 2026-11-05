package client

// multipart 上传与二进制下载能力（M8-CLI：工位 Excel 导入/导出/模板）。
//
// 复用既有装配：Bearer token（Authenticator）、401/1007 自动 refresh 重试一次。
// 上传/下载 body 不做整体加密（后端 multipart/文件流端点按明文解析）；
// 发送前做 auto 加密探测（probePublicKey，失败静默，跟随服务端动态开关）。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DoUpload multipart/form-data 文件上传。
//
// fileField 为表单字段名（后端固定 `file`，可覆盖）；fields 为附加表单字段。
// 文件流式写入（io.Pipe，避免大文件整体进内存）。
// 返回解析后的统一响应包装（import 端点 data 为 {inserted,updated,failed,errors}）。
// 401/1007 自动 refresh 后重试一次（重放时重新打开文件构建 multipart）。
func (c *clientImpl) DoUpload(ctx context.Context, path, fileField, filePath string, fields map[string]string) (*Response, error) {
	if fileField == "" {
		fileField = "file"
	}

	// buildBody 构建一次完整的 multipart 请求流（每次尝试重建，保证可重放）
	buildBody := func() (io.ReadCloser, string, error) {
		fh, err := os.Open(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("打开上传文件失败: %w", err)
		}
		fi, err := fh.Stat()
		if err != nil {
			_ = fh.Close()
			return nil, "", fmt.Errorf("读取文件信息失败: %w", err)
		}
		// 扩展名校验（.xlsx 约定；明显不支持的扩展提前拒绝）
		switch strings.ToLower(filepath.Ext(filePath)) {
		case ".xls", ".csv", ".doc", ".docx":
			_ = fh.Close()
			return nil, "", fmt.Errorf("仅支持 .xlsx（OOXML）文件，收到 %s", filepath.Ext(filePath))
		}

		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)
		go func() {
			defer func() { _ = pw.Close() }()
			defer func() { _ = fh.Close() }()
			// 附加字段（字母序，保证 multipart 输出稳定）
			keys := make([]string, 0, len(fields))
			for k := range fields {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if err := mw.WriteField(k, fields[k]); err != nil {
					_ = pw.CloseWithError(err)
					return
				}
			}
			// 文件字段（流式拷贝，不整体读入内存）
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(fileField), escapeQuotes(filepath.Base(filePath))))
			h.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			part, err := mw.CreatePart(h)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if _, err := io.CopyN(part, fh, fi.Size()); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if err := mw.Close(); err != nil {
				_ = pw.CloseWithError(err)
			}
		}()
		return pr, mw.FormDataContentType(), nil
	}

	attemptUpload := func() (*attempt, error) {
		bodyReader, ct, err := buildBody()
		if err != nil {
			return nil, &TransportError{Method: "POST", URL: path, Err: err}
		}
		token, _ := c.token(ctx)
		req, err := c.newRequest(ctx, "POST", path, nil, true, token)
		if err != nil {
			return nil, &TransportError{Method: "POST", URL: path, Err: err}
		}
		req.Header.Set("Content-Type", ct)
		req.Body = bodyReader
		req.ContentLength = -1

		httpResp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, &TransportError{Method: "POST", URL: req.URL.String(), Err: err}
		}
		raw, err := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		if err != nil {
			return nil, &TransportError{Method: "POST", URL: req.URL.String(), Err: err}
		}
		at := &attempt{rawBody: raw, httpStatus: httpResp.StatusCode}
		var wrapped Response
		if jerr := json.Unmarshal(raw, &wrapped); jerr == nil && hasCodeField(raw) {
			at.wrapped = true
			at.resp = wrapped
		}
		return at, nil
	}

	// auto 加密探测：跟随服务端动态开关；探测失败不阻断（明文发送 multipart）
	c.probePublicKey(ctx)

	at, err := attemptUpload()
	if err != nil {
		return nil, err
	}
	// 401/1007 自动 refresh 后重试一次（重建 multipart 流）
	if c.shouldRetryAuth(at) && c.authenticator != nil {
		if _, rerr := c.authenticator.Refresh(ctx); rerr == nil {
			at2, err2 := attemptUpload()
			if err2 == nil {
				at = at2
			}
		}
	}
	return c.resolve(at)
}

// DoDownload 二进制流下载（export=POST+JSON / template=GET）。
//
// 响应为文件流时写入 destPath；若响应是 JSON 错误包装（code!=0）则不落盘、
// 返回 APIError。destPath 为空时从 Content-Disposition 解析默认文件名。
// 返回实际保存路径。
func (c *clientImpl) DoDownload(ctx context.Context, path, jsonBody, destPath, method string) (string, error) {
	if method == "" {
		method = "GET"
	}
	// auto 加密探测：跟随服务端动态开关；探测失败不阻断
	c.probePublicKey(ctx)
	download := func() (*attempt, http.Header, error) {
		var payload []byte
		hasBody := false
		if strings.EqualFold(method, "POST") && jsonBody != "" {
			payload = []byte(jsonBody)
			hasBody = true
		}
		token, _ := c.token(ctx)
		req, err := c.newRequest(ctx, method, path, payload, hasBody, token)
		if err != nil {
			return nil, nil, &TransportError{Method: method, URL: path, Err: err}
		}
		httpResp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, nil, &TransportError{Method: method, URL: req.URL.String(), Err: err}
		}
		raw, err := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		if err != nil {
			return nil, nil, &TransportError{Method: method, URL: req.URL.String(), Err: err}
		}
		at := &attempt{rawBody: raw, httpStatus: httpResp.StatusCode}
		var wrapped Response
		if jerr := json.Unmarshal(raw, &wrapped); jerr == nil && hasCodeField(raw) {
			at.wrapped = true
			at.resp = wrapped
		}
		return at, httpResp.Header, nil
	}

	at, header, err := download()
	if err != nil {
		return "", err
	}
	// 401/1007 自动 refresh 后重试一次
	if c.shouldRetryAuth(at) && c.authenticator != nil {
		if _, rerr := c.authenticator.Refresh(ctx); rerr == nil {
			at2, h2, err2 := download()
			if err2 == nil {
				at, header = at2, h2
			}
		}
	}

	// JSON 响应：绝不落盘成 xlsx（code!=0 → APIError；code=0 JSON → 明确报错）
	ct := header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		if at.wrapped && at.resp.Code != 0 {
			return "", &APIError{Resp: at.resp, HTTPStatus: at.httpStatus}
		}
		return "", fmt.Errorf("响应为 JSON 而非文件流（Content-Type=%s），响应体: %s", ct, truncateStr(string(at.rawBody), 200))
	}

	filename := destPath
	if filename == "" {
		filename = filenameFromDisposition(header.Get("Content-Disposition"), path)
	}
	if filename == "" {
		filename = "download.bin"
	}
	if err := os.WriteFile(filename, at.rawBody, 0o644); err != nil {
		return "", &TransportError{Method: method, URL: path, Err: fmt.Errorf("写入文件失败: %w", err)}
	}
	return filename, nil
}

// filenameFromDisposition 从 Content-Disposition 解析文件名；
// 解析失败时用路径末段/实体名派生。
func filenameFromDisposition(cd, path string) string {
	if cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fn := params["filename"]; fn != "" {
				return fn
			}
		}
		// 宽松解析（filename*=UTF-8''xxx 等 Go mime 不认的变体）
		if i := strings.Index(cd, "filename="); i >= 0 {
			fn := strings.Trim(strings.TrimSpace(cd[i+len("filename="):]), `"`)
			if fn != "" {
				return fn
			}
		}
	}
	segs := strings.Split(strings.Trim(path, "/"), "/")
	base := segs[len(segs)-1]
	if base == "" || base == "template" || base == "export" {
		if len(segs) >= 2 {
			return segs[len(segs)-2] + ".xlsx"
		}
	}
	return base + ".xlsx"
}

// probePublicKey auto 加密探测：尝试拉取后端 SM2 公钥（一次），失败静默
// （服务端加密关闭时公钥端点不可用，按明文继续——跟随服务端动态开关）。
func (c *clientImpl) probePublicKey(ctx context.Context) {
	if c.encryptor == nil || c.pubFetched {
		return
	}
	c.pubFetched = true
	_ = c.encryptor.FetchPublicKey(ctx, c.baseURL)
}

// escapeQuotes multipart Content-Disposition 中的引号转义。
func escapeQuotes(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	return r.Replace(s)
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
