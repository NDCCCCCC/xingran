package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

// 前端 base 探测与尾斜杠重定向的直达测试。
//
// 本文件**不带 build tag** —— embed_frontend_prod.go 带 `//go:build embed`，
// CI 的 `go test ./...` 不带该标签 ⇒ 该文件不参与编译、不进覆盖率数据。
// 可测逻辑因此必须放在无 tag 的 frontend_base.go 里，否则：
//   1. diff coverage 门禁（GOV-03，≥80%）把这里的改动判为「未覆盖」
//   2. 对应的测试在 CI 里根本不执行 —— 前端子路径 base 探测的缺陷
//      （比较串漏前导斜杠 → detectedFrontendBase 恒为 ""）正是这样被放过的。

// resetDetectedFrontendBase 重置包级探测缓存，供各用例独立注入。
// frontendIndexHTML 一并清空 —— 回到「默认读取器」，embed 构建下即读真实产物。
func resetDetectedFrontendBase() {
	detectedBaseOnce = sync.Once{}
	detectedBase = ""
	frontendIndexHTML = nil
}

// stubFrontendIndexHTML 注入一段假的 index.html 内容。
func stubFrontendIndexHTML(html string) {
	frontendIndexHTML = func() ([]byte, error) {
		return []byte(html), nil
	}
}

func TestDetectedFrontendBase_SubPathBuild(t *testing.T) {
	resetDetectedFrontendBase()
	defer resetDetectedFrontendBase()
	stubFrontendIndexHTML(`<script type="module" src="/xingran/assets/index-abc.js"></script>`)

	if got := detectedFrontendBase(); got != frontendSubPath {
		t.Fatalf("detectedFrontendBase() = %q, want %q（子路径构建）", got, frontendSubPath)
	}
}

func TestDetectedFrontendBase_RootBuild(t *testing.T) {
	resetDetectedFrontendBase()
	defer resetDetectedFrontendBase()
	stubFrontendIndexHTML(`<script type="module" src="/assets/index-abc.js"></script>`)

	if got := detectedFrontendBase(); got != "" {
		t.Fatalf("detectedFrontendBase() = %q, want %q（根路径构建没有 basename）", got, "")
	}
}

func TestDetectedFrontendBase_NoEmbeddedFrontend(t *testing.T) {
	resetDetectedFrontendBase()
	defer resetDetectedFrontendBase()
	// 模拟「本次构建没有内嵌前端」：默认读取器据此返回 nil。
	// 自行保存/恢复，避免污染后续用例（embed 构建下它本为 true）。
	savedEmbedded := frontendEmbedded
	frontendEmbedded = false
	defer func() { frontendEmbedded = savedEmbedded }()
	frontendIndexHTML = nil

	if got := detectedFrontendBase(); got != "" {
		t.Fatalf("detectedFrontendBase() = %q, want %q（未内嵌前端）", got, "")
	}
}

func TestDetectedFrontendBase_ReadError(t *testing.T) {
	resetDetectedFrontendBase()
	defer resetDetectedFrontendBase()
	frontendIndexHTML = func() ([]byte, error) {
		return nil, fmt.Errorf("boom")
	}

	if got := detectedFrontendBase(); got != "" {
		t.Fatalf("detectedFrontendBase() = %q, want %q（读取失败按无 basename 处理）", got, "")
	}
}

// TestDetectedFrontendBase_NoAssetMarker 覆盖「index.html 里没有任何资源引用」
// 的情况：两个 marker 都不命中 → 探测结果为空。
func TestDetectedFrontendBase_NoAssetMarker(t *testing.T) {
	resetDetectedFrontendBase()
	defer resetDetectedFrontendBase()
	stubFrontendIndexHTML(`<html><body>no assets here</body></html>`)

	if got := detectedFrontendBase(); got != "" {
		t.Fatalf("detectedFrontendBase() = %q, want %q（无资源引用）", got, "")
	}
}

func TestRedirectSubPathTrailingSlash(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("缺尾斜杠 → 301 到 /xingran/", func(t *testing.T) {
		resetDetectedFrontendBase()
		defer resetDetectedFrontendBase()
		stubFrontendIndexHTML(`<script src="/xingran/assets/index-abc.js"></script>`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/xingran", nil)

		if !redirectSubPathTrailingSlash(c, "/xingran") {
			t.Fatal("redirectSubPathTrailingSlash() = false, want true")
		}
		if w.Code != http.StatusMovedPermanently {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusMovedPermanently)
		}
		if got := w.Header().Get("Location"); got != frontendSubPath+"/" {
			t.Fatalf("Location = %q, want %q", got, frontendSubPath+"/")
		}
	})

	t.Run("已带尾斜杠 → 不重定向", func(t *testing.T) {
		resetDetectedFrontendBase()
		defer resetDetectedFrontendBase()
		stubFrontendIndexHTML(`<script src="/xingran/assets/index-abc.js"></script>`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/xingran/", nil)

		if redirectSubPathTrailingSlash(c, "/xingran/") {
			t.Fatal("redirectSubPathTrailingSlash() = true, want false")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d（未写响应时保持 200）", w.Code, http.StatusOK)
		}
	})

	t.Run("根路径构建 → 不重定向", func(t *testing.T) {
		resetDetectedFrontendBase()
		defer resetDetectedFrontendBase()
		stubFrontendIndexHTML(`<script src="/assets/index-abc.js"></script>`)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/xingran", nil)

		if redirectSubPathTrailingSlash(c, "/xingran") {
			t.Fatal("redirectSubPathTrailingSlash() = true, want false（根路径构建无 basename）")
		}
	})
}
