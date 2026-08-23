// Phase 76-02 (INFRA-02) — newNetworkDriver 工厂注入演示测试。
//
// 生产构造链 NewScrapliWrapper → Open → SendCommand 全程真实代码，仅
// newNetworkDriver 包级 var 被临时替换为 FileTransport 版本（选项组合照
// internal/services/portwrite/port_write_e2e_test.go:71-90 的已验证先例），
// fixture 回放 huawei_vrp 设备的 Open 场景。Phase 78 BLOCK-04 的
// device ≥70% FileTransport/fake SSH 注入可直接照搬本模式。
//
// 注入纪律（Pitfall #6，与先例一致）：
//   - 禁止并行执行：包级 var 全局可变，并行测试互踩。
//   - 覆盖必须先保存 orig 并注册 t.Cleanup 恢复，再覆盖 var。
//   - 覆盖动作之后不得用 t.Fatal（用 t.Errorf + early return，防
//     Cleanup 前退出导致同包后续测试拿到 FileTransport 工厂）。
//
// We deliberately do NOT call d.Close()/w.Close(). The platform's
// `network-on-close` operations (acquire-priv + channel.write 'quit' +
// channel.return) need to read more bytes from the FileTransport after
// the fixture has been fully consumed, which would block on
// FileTransport.Read `select{}` (port_write_e2e_test.go:42-48 先例注释)。
// Skipping Close is safe because the file-transport driver has no real
// socket to release — Go GC will reap the wrapper when the test returns.
package device

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/scrapli/scrapligo/driver/network"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	"github.com/scrapli/scrapligo/transport"
	"github.com/scrapli/scrapligo/util"
)

// factoryFixturePath resolves the absolute path of a fixture under
// internal/device/testdata/ using runtime.Caller — robust to
// `go test ./...` invocations that change cwd
// （照 port_write_e2e_test.go:106-113 的 e2eFixturePath 模式，指向本包 testdata）。
func factoryFixturePath(t *testing.T, name string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", name)
}

// TestDriverFactoryFileTransportInjection — INFRA-02 注入能力实证：
// 覆盖 newNetworkDriver 后，经公开构造器 NewScrapliWrapper（非
// e2e_helpers 的 ForTesting 捷径）驱动 Open → SendCommand 全链 fixture 回放。
//
// fixture 字节流推演（huawei_vrp 平台，assets/platforms/huawei_vrp.yaml）：
//   - Open 读首个 prompt `<dummy-host>`（匹配 exec pattern ^<[\w.\-@/:]{1,63}>$）
//   - on-open acquire-priv：已在 exec（default desired），无读写
//   - on-open send-command 'screen-length 0 temporary'：读命令回显 + prompt
//   - 测试 SendCommand("display version")：读命令回显 + 版本输出 + prompt
func TestDriverFactoryFileTransportInjection(t *testing.T) {
	// 注入纪律：先保存 orig、注册 Cleanup 恢复，再覆盖 var（顺序写法）。
	orig := newNetworkDriver
	t.Cleanup(func() { newNetworkDriver = orig })

	fixturePath := factoryFixturePath(t, "huawei_vrp_open.fixture")

	newNetworkDriver = func(_ interface{}, _ string, _ ...util.Option) (*network.Driver, error) {
		p, err := platform.NewPlatform(
			"huawei_vrp",
			"dummy-host",
			options.WithTransportType(transport.FileTransport),
			options.WithFileTransportFile(fixturePath),
			options.WithTransportReadSize(1),
			options.WithReadDelay(0),
		)
		if err != nil {
			return nil, err
		}
		return p.GetNetworkDriver()
	}

	// 真实生产构造链：公开构造器（全 dummy 值，禁真实设备 IP/凭据）。
	dev := &models.NetworkDevice{
		DeviceName: "dummy-fixture-device",
		IPAddress:  "dummy-host",
		Vendor:     models.VendorHuawei,
	}
	w, err := NewScrapliWrapper(dev, "u", "p", models.ProtocolTypeSSH)
	if err != nil {
		t.Errorf("NewScrapliWrapper returned err: %v", err)
		return
	}

	if err := w.Open(); err != nil {
		t.Errorf("w.Open returned err: %v", err)
		return
	}

	resp, err := w.SendCommand("display version", false)
	if err != nil {
		t.Errorf("w.SendCommand returned err: %v", err)
		return
	}
	if resp == nil {
		t.Error("resp is nil")
		return
	}
	if trimmed := strings.TrimSpace(resp.Result); trimmed == "" {
		t.Errorf("SendCommand Result empty after TrimSpace; got %q", resp.Result)
	}
	if !strings.Contains(resp.Result, "Huawei") {
		t.Errorf("SendCommand Result missing 'Huawei'; got %q", resp.Result)
	}

	// Intentionally no Close — see file header comment.
}
