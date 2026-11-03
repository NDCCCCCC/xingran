package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// 前端嵌入资源的静态服务基础件（本文件不带 build tag）。
//
// 为什么放在这里，而不是 embed_frontend_prod.go：
//
//	embed_frontend_prod.go 带 `//go:build embed` 标签，而 CI 的
//	`go test ./...` 不带该标签 ⇒ 该文件不参与编译，也就不进覆盖率数据。
//	于是落在这两个文件里的改动都会被 diff coverage 门禁（GOV-03，≥80%）
//	判为「未覆盖」，且它们的测试在 CI 里根本不执行 —— 前端子路径 base
//	探测的缺陷（比较串漏前导斜杠 → detectedFrontendBase 恒为 ""）正是
//	这样被放过的。
//
//	拆分原则：**可测的判定逻辑放本文件**（无 tag，CI 覆盖），
//	embed 构建只负责把 embed.FS 注入进来。

// frontendFS 嵌入的前端资源集合；由 embed 构建通过 setEmbeddedFrontend 注入。
//
// 用 fs.FS 接口类型（而非 embed.FS 具体类型）：测试可注入 fstest.MapFS，
// 于是「读取 index.html」这条路径在 CI 的非 embed 构建下也能被覆盖 ——
// 否则它只存在于 embed 专属代码里，diff coverage 永远判为未覆盖。
var frontendFS fs.FS

// frontendEmbedded 标记本次构建是否内嵌了前端产物。
var frontendEmbedded bool

// frontendIndexHTML 读取嵌入的 index.html；仅供测试注入。
// 为 nil 时回落到 defaultFrontendIndexHTML()（embed 构建读真实产物）。
var frontendIndexHTML func() ([]byte, error)

var (
	detectedBaseOnce sync.Once
	detectedBase     string
)

// setEmbeddedFrontend 由 embed 构建在 init 中调用，注入资源集合。
func setEmbeddedFrontend(fsys embed.FS) {
	frontendFS = fsys
	frontendEmbedded = true
}

// defaultFrontendIndexHTML 返回读取嵌入 index.html 的函数。
// 非 embed 构建没有内嵌前端 → 返回 nil（探测按「无 basename」处理）。
func defaultFrontendIndexHTML() func() ([]byte, error) {
	if !frontendEmbedded {
		return nil
	}
	return readEmbeddedIndexHTML
}

// readEmbeddedIndexHTML 读取嵌入产物中的 index.html。
func readEmbeddedIndexHTML() ([]byte, error) {
	return fs.ReadFile(frontendFS, "xingran-react-frontend/dist/index.html")
}

// detectedFrontendBase 从嵌入 dist 的 index.html 自动探测前端构建 base。
//
// Vite 以 VITE_BASE 构建时,资源引用形如 src="/xingran/assets/xxx.js";
// 根路径构建时为 src="/assets/xxx.js"。取首个 "/...assets/" 引用的前缀:
//   - "/xingran/assets/" → 返回 "/xingran"(子路径构建)
//   - "/assets/"         → 返回 ""(根路径构建)
//
// 两处重定向行为(根路径 302、陈旧无前缀 SPA URL 补前缀)都以它为开关:
// 根路径构建的 dist 没有 basename,任何重定向到 /xingran 的行为都会把
// 浏览器送进不存在的地址 —— 必须全部停用。
func detectedFrontendBase() string {
	detectedBaseOnce.Do(func() {
		provider := frontendIndexHTML
		if provider == nil {
			provider = defaultFrontendIndexHTML()
		}
		if provider == nil {
			return
		}
		content, err := provider()
		if err != nil {
			return
		}
		html := string(content)
		for _, marker := range []string{"\"/" + strings.TrimPrefix(frontendSubPath, "/") + "/assets/", "\"/assets/"} {
			idx := strings.Index(html, marker)
			if idx < 0 {
				continue
			}
			rest := html[idx+1:]
			// rest 只去掉了 marker 的首字符(引号),前导斜杠仍在:
			//   marker = `"/xingran/assets/` → rest = `/xingran/assets/xxx.js`
			// 比较串必须同样带前导斜杠,否则永远不匹配 —— 这正是
			// detectedFrontendBase() 恒返回 "" 的根因:
			// 直连 :9000 时后端不做 302 引导,浏览器停在 "/",
			// 与前端 basename "/xingran/" 不匹配 → 白屏。
			if strings.HasPrefix(rest, frontendSubPath+"/assets/") {
				detectedBase = frontendSubPath
			} else {
				detectedBase = ""
			}
			return
		}
	})
	return detectedBase
}

// redirectSubPathTrailingSlash 处理「缺尾斜杠」的子路径地址。
//
// 子路径部署下 basename="/xingran/" 要求 URL 以 "/xingran/" 开头;
// 请求 "/xingran" 时 React Router 会因 basename 不匹配而整个不渲染 ——
// 白屏,控制台报 "<Router basename=/xingran/> is not able to match the URL /xingran"。
// nginx 侧有 location = /xingran { return 301 /xingran/; } 兜底,
// 但直连 :9000 的部署没有这层 → 后端补一个尾斜杠重定向。
//
// 返回 true 表示已重定向,调用方应立即返回。
func redirectSubPathTrailingSlash(c *gin.Context, requestPath string) bool {
	if detectedFrontendBase() == "" || requestPath != frontendSubPath {
		return false
	}
	c.Redirect(http.StatusMovedPermanently, frontendSubPath+"/")
	return true
}
