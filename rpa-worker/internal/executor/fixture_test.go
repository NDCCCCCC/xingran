package executor

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
)

// fixtureHTML ENGINE-05 的本地 fixture 页面：集中覆盖 ENGINE-02 的 14 个动作
// 所需的最小交互面（form / button / select / file input / drag 区域 / iframe /
// 下载链接 / 新页签入口）。每个交互点都把结果写回 DOM 或 #log，
// 断言只看「页面自己承认的副作用」，不用 mock 冒充。
const fixtureHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="utf-8"><title>ENGINE-02 fixture</title>
<style>
 #dropzone { width: 120px; height: 60px; border: 1px solid #999; }
 #draggable { width: 60px; height: 30px; background: #ddd; }
 #hoverbox { width: 80px; height: 30px; background: #eee; }
</style></head>
<body>
 <div id="log">init</div>

 <form id="form">
   <input id="name" name="name" type="text" value="">
   <select id="sel" name="sel">
     <option value="a">选项A</option>
     <option value="b">选项B</option>
   </select>
   <input id="file" name="file" type="file">
   <button id="btn-submit" type="button">提交</button>
 </form>

 <button id="btn-dbl" type="button">双击我</button>
 <div id="dbl-count">0</div>

 <div id="ctx-area" style="width:120px;height:40px;background:#fdd">右键区域</div>
 <div id="ctx-count">0</div>

 <div id="hoverbox">悬停目标</div>
 <div id="hover-count">0</div>

 <div id="draggable" draggable="true">拖我</div>
 <div id="dropzone">放下</div>
 <div id="drop-count">0</div>

 <input id="keyin" type="text" value="">
 <div id="key-log"></div>

 <a id="download-link" href="/download">下载文件</a>

 <button id="open-tab" type="button" onclick="window.open('/tab2','_blank')">开新页签</button>

 <iframe id="frame1" src="/iframe" width="300" height="120"></iframe>

 <div id="editable" contenteditable="true">可编辑</div>

<script>
 document.getElementById('btn-submit').addEventListener('click', function(){
   document.getElementById('log').textContent = 'submit-button';
 });
 document.getElementById('btn-dbl').addEventListener('dblclick', function(){
   const c = document.getElementById('dbl-count');
   c.textContent = String(parseInt(c.textContent,10)+1);
 });
 document.getElementById('ctx-area').addEventListener('contextmenu', function(e){
   e.preventDefault();
   const c = document.getElementById('ctx-count');
   c.textContent = String(parseInt(c.textContent,10)+1);
 });
 document.getElementById('hoverbox').addEventListener('mouseenter', function(){
   document.getElementById('hover-count').textContent = '1';
 });
 (function(){
   const src = document.getElementById('draggable');
   const dst = document.getElementById('dropzone');
   let dragging = false;
   src.addEventListener('dragstart', function(e){
     dragging = true;
     if (e.dataTransfer) { e.dataTransfer.setData('text/plain','d'); }
   });
   dst.addEventListener('dragover', function(e){ e.preventDefault(); });
   dst.addEventListener('drop', function(e){
     e.preventDefault();
     if (dragging) {
       document.getElementById('drop-count').textContent = '1';
       document.getElementById('log').textContent = 'dropped';
     }
   });
   src.addEventListener('dragend', function(){ dragging = false; });
 })();
 document.getElementById('keyin').addEventListener('keydown', function(e){
   const mods = [];
   if (e.ctrlKey) mods.push('ctrl');
   if (e.shiftKey) mods.push('shift');
   if (e.altKey) mods.push('alt');
   if (e.metaKey) mods.push('meta');
   mods.push(e.key);
   document.getElementById('key-log').textContent = mods.join('+');
 });
</script>
</body></html>`

// iframeHTML iframe 内部页面：用于进入/退出 frame 的断言。
const iframeHTML = `<!DOCTYPE html><html lang="zh-CN"><body>
 <input id="frame-input" type="text" value="">
 <button id="frame-btn" type="button"
   onclick="document.getElementById('frame-input').value='from-frame'">帧内按钮</button>
 <div id="frame-state">idle</div>
</body></html>`

// fixtureDownloadPayload 下载动作的产物内容。
const fixtureDownloadPayload = "ENGINE-02 download payload\n"

// startFixture 启动本地 fixture 服务，返回 server 与下载目录。
func startFixture(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, fixtureHTML)
	})
	mux.HandleFunc("/iframe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, iframeHTML)
	})
	mux.HandleFunc("/tab2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<!DOCTYPE html><html lang="zh-CN"><body>
 <div id="tab2-marker">tab2</div></body></html>`)
	})
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="engine02.txt"`)
		_, _ = fmt.Fprint(w, fixtureDownloadPayload)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	return srv, dir
}

// newIntegrationEngine 起真 Chromium 的 Engine（无 Chromium 或 -short 时 skip）。
func newIntegrationEngine(t *testing.T, downloadsDir string) (*Engine, *browser.Manager) {
	t.Helper()
	if testing.Short() {
		t.Skip("跳过真 Chromium 集成用例（-short）")
	}
	chrome := browser.TestChromePath()
	if chrome == "" {
		t.Skip("本机无 Chromium，跳过真浏览器用例")
	}

	cfg := &config.Config{
		Browser: config.BrowserConfig{
			Headless: true, Timeout: 10 * time.Second, NavigationTimeout: 15 * time.Second,
			MaxConcurrency: 2, ViewportWidth: 1280, ViewportHeight: 800,
			ChromePath: chrome,
		},
		Worker: config.WorkerConfig{
			MaxConcurrency: 2,
			DownloadsDir:   downloadsDir,
			ActionTimeout:  10 * time.Second,
			PauseTimeout:   1 * time.Second,
		},
		Executor: config.ExecutorConfig{
			InitialDelay: 100 * time.Millisecond, UndefinedVariablePolicy: config.UndefinedVariableEmpty,
		},
	}
	cfg.Browser.MaxConcurrency = cfg.Worker.MaxConcurrency

	mgr, err := browser.NewManager(&cfg.Browser, testLogger{t})
	if err != nil {
		t.Fatalf("创建 Manager 失败: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	eng := NewEngine(&cfg.Executor, &cfg.Worker, mgr, nil, testLogger{t}, nil)
	return eng, mgr
}

// testLogger 把 WARN/ERROR 打到测试输出，其余静默（避免噪声）。
type testLogger struct{ t *testing.T }

func (l testLogger) Debug(string, ...logger.Field) {}
func (l testLogger) Info(string, ...logger.Field)  {}
func (l testLogger) Warn(msg string, f ...logger.Field) {
	l.t.Logf("WARN %s %v", msg, f)
}
func (l testLogger) Error(msg string, f ...logger.Field) {
	l.t.Logf("ERROR %s %v", msg, f)
}
func (l testLogger) With(...logger.Field) logger.Logger { return l }
func (l testLogger) Sync() error                        { return nil }
