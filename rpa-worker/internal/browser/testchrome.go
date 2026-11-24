package browser

import "os"

// TestChromePath 探测本机可用 Chromium/Chrome（供集成测试与运维探针复用）。
// 可用 RPA_TEST_CHROME_PATH 显式覆盖。找不到返回 ""。
func TestChromePath() string {
	if p := os.Getenv("RPA_TEST_CHROME_PATH"); p != "" {
		return p
	}
	candidates := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}
