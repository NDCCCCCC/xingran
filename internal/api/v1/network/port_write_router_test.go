package network

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Phase 52 Wave 1 router 测试 — 全部走源码 grep 断言。
//
// 原因：SetupPortWriteRouter 内部第一行调 core.GetDB()，空 Core 会 nil deref；
// 完整构造 Core 需要 sqlite + DeviceExecutor 等大量依赖，超出 router 验证目的。
// VALIDATION.md §4.5 Wave 0 接受 grep 形式的源断言。本文件作为编译期校验存在，
// 同时把 grep 期望写在 _test.go 中便于未来 CI 拦截。
//
// grep 用 file 读源文件 → strings.Contains 断言，避免引入 go:embed。

// readFile 读相对本包的源文件路径。
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestSetupPortWriteRouter_RequirePermissions2Arg router 源码含完整 2-arg
// RequirePermissions([]string{string(permission.NetworkPortWrite)}, core)
//
// 这是 5 个 load-bearing corrections 之一（critical_constraints #1）。
// 必须含 core 第二参；禁止 1-arg 错误形式。
func TestSetupPortWriteRouter_RequirePermissions2Arg(t *testing.T) {
	src := readFile(t, "port_write_router.go")
	// 注释/blank line 去掉避免干扰（行内注释会带 // 但 RequirePermissions 调用本身是代码行）
	// 简单做：源文件含此完整 token 串即可
	assert.Contains(t, src,
		"middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core)",
		"RequirePermissions 必须用 2-arg 形式含 core 第二参")
	// 防御：不含 1-arg 错误形式（无尾随 , core）
	assert.NotContains(t, src,
		"middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)})",
		"禁止 1-arg RequirePermissions 形式（缺 core 第二参）")
}

// TestSetupPortWriteRouter_UsesNetworkPortWriteConstant router 引用 permission.NetworkPortWrite
// 而非硬编码字符串 "network:port:write"
func TestSetupPortWriteRouter_UsesNetworkPortWriteConstant(t *testing.T) {
	src := readFile(t, "port_write_router.go")
	assert.Contains(t, src, "permission.NetworkPortWrite",
		"router 必须引用常量 permission.NetworkPortWrite，禁止硬编码字符串")
	// 注释里可以提到字符串（文档），但代码行不应出现 "network:port:write" 硬编码
	// 此处只校验常量引用存在，进一步硬编码检测留给 goimports/vet
}

// TestSetupPortWriteRouter_Registers6KebabEndpoints router 注册 6 个 kebab 端点
func TestSetupPortWriteRouter_Registers6KebabEndpoints(t *testing.T) {
	src := readFile(t, "port_write_router.go")
	expected := []string{
		`write.POST("/shutdown"`,
		`write.POST("/undo-shutdown"`,
		`write.POST("/description"`,
		`write.POST("/dot1x-enable"`,
		`write.POST("/dot1x-disable"`,
		`write.POST("/batch"`,
	}
	for _, w := range expected {
		assert.Contains(t, src, w, "router 缺少端点注册：%s", w)
	}
}

// TestSetupPortWriteRouter_RegistersV1201KebabEndpoints router 注册 v1.20.1 (Phase 56 W3)
// 新增的 2 个 kebab 端点：set-access-vlan + port-binding。
//
// 这 2 个端点必须复用 v1.19 的组级 RequirePermissions([network:port:write], core)
// 中间件，不引入新 perm constant（design.md §5）。
func TestSetupPortWriteRouter_RegistersV1201KebabEndpoints(t *testing.T) {
	src := readFile(t, "port_write_router.go")
	expected := []string{
		`write.POST("/set-access-vlan"`,
		`write.POST("/port-binding"`,
	}
	for _, w := range expected {
		assert.Contains(t, src, w, "router 缺少 v1.20.1 端点注册：%s", w)
	}
}

// TestNetworkRouter_RegistersSetupPortWriteRouter network_router.go 含
// SetupPortWriteRouter(ports, core) 调用（在 SetupPortRouter 之后）
func TestNetworkRouter_RegistersSetupPortWriteRouter(t *testing.T) {
	src := readFile(t, "network_router.go")
	assert.Contains(t, src, "SetupPortWriteRouter(ports, core)",
		"network_router.go 必须在 ports 组内调 SetupPortWriteRouter(ports, core)")

	// 验证 SetupPortWriteRouter 在 SetupPortRouter 之后（同 ports 组内）
	idx1 := strings.Index(src, "SetupPortRouter(ports, core, exportHandler)")
	idx2 := strings.Index(src, "SetupPortWriteRouter(ports, core)")
	assert.GreaterOrEqual(t, idx1, 0, "SetupPortRouter must exist in network_router.go")
	assert.Greater(t, idx2, idx1, "SetupPortWriteRouter must come AFTER SetupPortRouter")
}

// TestSetupPortWriteRouter_DefinesSetupFunction port_write_router.go 含函数定义
func TestSetupPortWriteRouter_DefinesSetupFunction(t *testing.T) {
	src := readFile(t, "port_write_router.go")
	assert.Contains(t, src, "func SetupPortWriteRouter(r *gin.RouterGroup, core *core.Core)",
		"router 文件必须定义 func SetupPortWriteRouter(r *gin.RouterGroup, core *core.Core)")
}
