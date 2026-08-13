package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// validYAML 用于测试的最小合法 YAML。
const validYAML = `
version: "1.0"
metadata:
  - module: system
    category: 系统管理
    icon: icon-system
    endpoints:
      - route: /system/users/list
        method: POST
        displayName: 用户列表
        description: 分页查询用户
        dataType: paginated
        dataPath: data.list
        supportedWidgets:
          - table
          - card
        permissions:
          - system:user:list
        exampleParams:
          current: "1"
          pageSize: "10"
      - route: /system/users/create
        method: POST
        displayName: 新增用户
        description: 创建新用户
        dataType: single
        dataPath: data
  - module: operations
    category: 运维管理
    icon: icon-ops
    endpoints:
      - route: /ops/building/list
        method: POST
        displayName: 楼宇列表
        dataType: paginated
        dataPath: data.list
`

// TestLoadAPIMetadata_Success 验证正常 YAML 解析。
func TestLoadAPIMetadata_Success(t *testing.T) {
	path := writeYAML(t, validYAML)

	cfg, err := LoadAPIMetadata(context.Background(), path)
	if err != nil {
		t.Fatalf("期望无错,实际: %v", err)
	}
	if cfg.Version != "1.0" {
		t.Errorf("Version 应为 1.0,实际 %q", cfg.Version)
	}
	if len(cfg.Metadata) != 2 {
		t.Fatalf("模块数应为 2,实际 %d", len(cfg.Metadata))
	}
	if cfg.Metadata[0].Module != "system" {
		t.Errorf("第一个模块应为 system,实际 %q", cfg.Metadata[0].Module)
	}
	if len(cfg.Metadata[0].Endpoints) != 2 {
		t.Errorf("system 端点数应为 2,实际 %d", len(cfg.Metadata[0].Endpoints))
	}
}

// TestLoadAPIMetadata_FileNotFound 验证文件不存在时返回包装错误。
func TestLoadAPIMetadata_FileNotFound(t *testing.T) {
	_, err := LoadAPIMetadata(context.Background(), filepath.Join(t.TempDir(), "does_not_exist.yaml"))
	if err == nil {
		t.Fatal("期望错误(文件不存在),实际无错")
	}
	if !strings.Contains(err.Error(), "读取API元数据配置失败") {
		t.Errorf("错误信息应包含读取失败提示,实际: %v", err)
	}
}

// TestLoadAPIMetadata_InvalidYAML 验证格式错误的 YAML 返回包装错误。
func TestLoadAPIMetadata_InvalidYAML(t *testing.T) {
	// 缩进错误会让 yaml.v3 报错。
	path := writeYAML(t, "version: 1.0\nmetadata:\n  - module: x\n  bad_indent: oops\n")
	_, err := LoadAPIMetadata(context.Background(), path)
	if err == nil {
		t.Fatal("期望解析错误,实际无错")
	}
	if !strings.Contains(err.Error(), "解析API元数据配置失败") {
		t.Errorf("错误信息应包含解析失败提示,实际: %v", err)
	}
}

// TestLoadAPIMetadata_ContextCanceled 验证 ctx 取消时 ReadFile 路径返回包装错误。
//
// 复现方案: 提供一个会在 read goroutine 内长期阻塞的"特殊文件"较麻烦,
// 这里改用立即取消的 ctx 来命中 select 的 ctx.Done 分支。
func TestLoadAPIMetadata_ContextCanceled(t *testing.T) {
	path := writeYAML(t, validYAML)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := LoadAPIMetadata(ctx, path)
	if err == nil {
		t.Fatal("期望 ctx 取消错误,实际无错")
	}
	if !strings.Contains(err.Error(), "读取API元数据配置已取消") {
		t.Errorf("错误信息应包含 ctx 取消提示,实际: %v", err)
	}
}

// TestLoadAPIMetadata_Normalize 验证 WR-3:Method / Route 在数据源不一致时
//
//	也能命中同一端点(规范化)。
func TestLoadAPIMetadata_Normalize(t *testing.T) {
	raw := `
version: "1.0"
metadata:
  - module: system
    category: 系统管理
    endpoints:
      - route: "system/users/list"     # 无前导 /
        method: "post"                # 小写
      - route: "/system/users/list/"  # 带尾部 /
        method: "POST"
      - route: "  /system/users/list  " # 前后空格
        method: "  POST  "
`
	cfg, err := LoadAPIMetadata(context.Background(), writeYAML(t, raw))
	if err != nil {
		t.Fatalf("准备失败: %v", err)
	}

	tests := []struct {
		name   string
		route  string
		method string
		wantOK bool
	}{
		// 查询侧 method 规范化(ToUpper + TrimSpace)
		{"查询小写 method → 命中", "/system/users/list", "post", true},
		{"查询带空白 method → 命中", "/system/users/list", "  POST  ", true},
		// 查询侧 route 规范化(WR-1:与存储侧 normalize 对称,handler 直传用户 query 也能命中)
		{"查询带尾斜杠 route → 命中", "/system/users/list/", "POST", true},
		{"查询无前导斜杠 route → 命中", "system/users/list", "POST", true},
		{"查询带前后空白 route → 命中", "  /system/users/list  ", "POST", true},
		// 未命中
		{"查询不存在的 route → miss", "/system/users/notfound", "POST", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := cfg.GetEndpointByRoute(tt.route, tt.method)
			if tt.wantOK && ep == nil {
				t.Fatalf("期望命中,实际 nil")
			}
			if !tt.wantOK && ep != nil {
				t.Fatalf("期望未命中,实际返回 %+v", ep)
			}
		})
	}

	// 同步验证:yaml 原文里三个端点的规范化后形态应该相同。
	for _, ep := range cfg.Metadata[0].Endpoints {
		if ep.Method != "POST" {
			t.Errorf("Method 应被规范化为 POST,实际 %q", ep.Method)
		}
		if ep.Route != "/system/users/list" {
			t.Errorf("Route 应被规范化为 /system/users/list,实际 %q", ep.Route)
		}
	}
}

// TestGetEndpointByRoute 表驱动测试命中/未命中场景。
func TestGetEndpointByRoute(t *testing.T) {
	cfg, err := LoadAPIMetadata(context.Background(), writeYAML(t, validYAML))
	if err != nil {
		t.Fatalf("测试准备失败: %v", err)
	}

	tests := []struct {
		name    string
		route   string
		method  string
		wantHit bool
	}{
		{"命中-第一个端点", "/system/users/list", "POST", true},
		{"命中-第二个端点", "/system/users/create", "POST", true},
		{"命中-不同模块", "/ops/building/list", "POST", true},
		{"未命中-路由不存在", "/system/users/notfound", "POST", false},
		{"未命中-method 不匹配", "/system/users/list", "GET", false},
		{"未命中-完全虚构", "/api/v1/fake", "DELETE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := cfg.GetEndpointByRoute(tt.route, tt.method)
			if tt.wantHit && ep == nil {
				t.Fatalf("期望命中,实际 nil")
			}
			if !tt.wantHit && ep != nil {
				t.Fatalf("期望未命中,实际返回 %+v", ep)
			}
		})
	}
}

// TestGetEndpointByRoute_Concurrent 验证 sync.Once 懒加载索引的并发安全性。
//
// 100 goroutine 同时调用 GetEndpointByRoute,触发竞争:
//   - indexOnce.Do 必须只执行一次
//   - index map 写入/读取必须无 data race
//
// 配合 go test -race 才能真正检测到竞争,这里直接跑会偶现 panic 或错误结果。
func TestGetEndpointByRoute_Concurrent(t *testing.T) {
	cfg, err := LoadAPIMetadata(context.Background(), writeYAML(t, validYAML))
	if err != nil {
		t.Fatalf("测试准备失败: %v", err)
	}

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// 用 channel 收集结果,确保全部 goroutine 都看到了命中。
	results := make(chan *EndpointMeta, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			results <- cfg.GetEndpointByRoute("/system/users/list", "POST")
		}()
	}
	wg.Wait()
	close(results)

	count := 0
	for ep := range results {
		if ep == nil {
			t.Fatal("并发查询返回 nil,索引可能未正确构建")
		}
		count++
	}
	if count != goroutines {
		t.Errorf("应有 %d 个查询结果,实际 %d", goroutines, count)
	}
}

// TestGetEndpointByRoute_ReturnsCopy 验证 WR-3:GetEndpointByRoute 返回值拷贝,
// 调用方修改返回值不影响内部 Metadata(保持"加载后只读"不变量)。
func TestGetEndpointByRoute_ReturnsCopy(t *testing.T) {
	cfg, err := LoadAPIMetadata(context.Background(), writeYAML(t, validYAML))
	if err != nil {
		t.Fatalf("测试准备失败: %v", err)
	}

	ep := cfg.GetEndpointByRoute("/system/users/list", "POST")
	if ep == nil {
		t.Fatal("期望命中")
	}
	originalRoute := ep.Route

	// 改写返回值,不应污染内部状态。
	ep.Route = "/tampered"
	ep.Method = "DELETE"

	ep2 := cfg.GetEndpointByRoute("/system/users/list", "POST")
	if ep2 == nil {
		t.Fatal("改写返回值后再次查询仍应命中;若 nil 说明索引被污染")
	}
	if ep2.Route != originalRoute {
		t.Errorf("内部 Route 应仍为 %q,实际 %q(返回值非独立拷贝)", originalRoute, ep2.Route)
	}
	if ep2.Method != "POST" {
		t.Errorf("内部 Method 应仍为 POST,实际 %q", ep2.Method)
	}
}

// TestGetAllEndpoints 验证返回深拷贝切片(WR-4 修复:顶层 + 嵌套 Endpoints 均拷贝)。
//
// 语义说明:GetAllEndpoints 返回独立快照,调用方修改任意层级值字段都不影响 cfg.Metadata。
// 注意:EndpointMeta 内的切片/映射字段(SupportedWidgets 等)仍共享底层数据,不应修改。
func TestGetAllEndpoints(t *testing.T) {
	cfg, err := LoadAPIMetadata(context.Background(), writeYAML(t, validYAML))
	if err != nil {
		t.Fatalf("测试准备失败: %v", err)
	}

	all := cfg.GetAllEndpoints()
	if len(all) != 2 {
		t.Fatalf("模块数应为 2,实际 %d", len(all))
	}
	if all[0].Module != "system" {
		t.Errorf("第一个模块应为 system,实际 %q", all[0].Module)
	}

	// 修改返回值顶层字段不应该影响 cfg.Metadata。
	all[0].Module = "tampered"
	if cfg.Metadata[0].Module == "tampered" {
		t.Error("GetAllEndpoints 返回的是内部引用,违反深拷贝语义")
	}

	// 嵌套 Endpoints 切片也应独立(WR-4:原浅拷贝共享底层数组,会污染内部)。
	all[0].Endpoints[0].Route = "/tampered"
	if cfg.Metadata[0].Endpoints[0].Route == "/tampered" {
		t.Error("GetAllEndpoints 嵌套 Endpoints 仍共享内部底层数组,违反深拷贝语义")
	}

	// nil Metadata 应返回空切片(便于调用方 len() == 0 判断)。
	var nilCfgAPIMeta APIMetadataConfig // Metadata 字段为 nil
	if got := nilCfgAPIMeta.GetAllEndpoints(); len(got) != 0 {
		t.Errorf("nil Metadata 应返回空切片,实际 len=%d", len(got))
	}
}

// BenchmarkGetEndpointByRoute 量化索引收益(IN-4)。
//
// 对照组 BenchmarkGetEndpointByRoute_Linear 直接线性扫描 c.Metadata,
// 用于验证 O(1) 索引 vs O(N*M) 扫描的延迟差异。
func BenchmarkGetEndpointByRoute(b *testing.B) {
	path := writeYAML(b, largeYAML())
	cfg, err := LoadAPIMetadata(context.Background(), path)
	if err != nil {
		b.Fatalf("准备失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.GetEndpointByRoute("/bench/m0/endpoint0", "POST")
	}
}

// BenchmarkGetEndpointByRoute_Linear 对照组:线性扫描(无索引)。
func BenchmarkGetEndpointByRoute_Linear(b *testing.B) {
	path := writeYAML(b, largeYAML())
	cfg, err := LoadAPIMetadata(context.Background(), path)
	if err != nil {
		b.Fatalf("准备失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for m := range cfg.Metadata {
			for j := range cfg.Metadata[m].Endpoints {
				if cfg.Metadata[m].Endpoints[j].Route == "/bench/m0/endpoint0" &&
					cfg.Metadata[m].Endpoints[j].Method == "POST" {
					_ = cfg.Metadata[m].Endpoints[j]
					break
				}
			}
		}
	}
}

// largeYAML 生成 N=10 模块 × M=20 端点 = 200 端点的 YAML,用于 benchmark。
func largeYAML() string {
	var sb strings.Builder
	sb.WriteString("version: \"1.0\"\nmetadata:\n")
	for m := 0; m < 10; m++ {
		sb.WriteString("  - module: bench-module\n")
		sb.WriteString("    category: bench\n")
		sb.WriteString("    icon: icon\n")
		sb.WriteString("    endpoints:\n")
		for e := 0; e < 20; e++ {
			fmt.Fprintf(&sb, "      - route: /bench/m%d/endpoint%d\n", m, e)
			sb.WriteString("        method: POST\n")
			sb.WriteString("        displayName: bench\n")
		}
	}
	return sb.String()
}

// writeYAML 是测试辅助函数:在临时目录写入内容并返回路径。
//
// 接受 testing.TB 接口,同时支持 *testing.T 和 *testing.B。
// t.TempDir() / b.TempDir() 自动清理,无需 defer os.Remove。
func writeYAML(tb testing.TB, content string) string {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "api_metadata.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		tb.Fatalf("写入测试 YAML 失败: %v", err)
	}
	return path
}