// 临时工具：v1.51 端口写真机 UAT 执行器（跑完即删）
//
// 模式：
//   render   渲染 16 动作 × 2 厂商命令（不连设备）
//   baseline SSH 采集基线（只读命令）+ 存 JSON
//   probe    SSH 关键字探测（进配置视图用 ? 问帮助，零配置变更）
//   exec     SSH 下发动作命令（写操作）+ 采集回显
//
// 安全约束：
//   - 命令一律来自 portcollection.RenderCommand（产品真相源），不手写命令
//   - 密码只在内存中解密使用，绝不输出
//   - exec 需显式 --apply，否则只打印将要下发的命令
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	"golang.org/x/crypto/ssh"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	testIP      = "10.62.25.5"
	testMAC     = "AA-BB-CC-DD-EE-FF"
	testVLAN    = 100
	testDesc    = "UAT-TEST-129"
	testAllowed = "100,200"
)

type Device struct {
	DeviceName   string
	Vendor       string
	Model        string
	IPAddress    string
	Port         int
	CredentialID *string
	Location     string
}

type Cred struct {
	ID             string
	CredentialName string
	Username       string
	Password       string
}

type safeBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}
func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func main() {
	var cfgPath, sm4EnvFile, mode, deviceName, iface, action, outFile, rawCmds, rawView string
	var apply bool
	flag.StringVar(&cfgPath, "config", "/app/szh/configs/config.yaml", "配置文件路径")
	flag.StringVar(&sm4EnvFile, "sm4-env-file", "", "从该文件读 SM4_KEY（如 /proc/<pid>/environ）")
	flag.StringVar(&mode, "mode", "render", "render|baseline|probe|exec")
	flag.StringVar(&deviceName, "device", "", "设备名")
	flag.StringVar(&iface, "port", "", "接口名（如 10GE5/0/0）")
	flag.StringVar(&action, "action", "", "动作（exec 模式）")
	flag.StringVar(&outFile, "out", "", "输出文件")
	flag.StringVar(&rawCmds, "commands", "", "raw 模式：分号分隔的原始命令序列")
	flag.StringVar(&rawView, "view", "interface", "raw 模式视图：user|config|interface")
	flag.BoolVar(&apply, "apply", false, "exec 模式：真实下发（默认只打印）")
	flag.Parse()

	if mode == "render" {
		renderAll()
		return
	}

	host, port, user, pass, db, sm4Key := loadConfig(cfgPath)
	if sm4EnvFile != "" {
		if k, ok := readEnvFromFile(sm4EnvFile, "SM4_KEY"); ok {
			sm4Key = k
		}
	}
	if sm4Key == "" {
		sm4Key = "dGVzdC1zZWNyZXQxNiEhIQ=="
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, db)
	dbh, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB 连接失败: %v\n", err)
		os.Exit(3)
	}
	cipher, err := crypto.NewSM4Cipher(sm4Key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SM4 失败: %v\n", err)
		os.Exit(3)
	}

	var dev Device
	if err := dbh.Table("sys_network_device").Select("device_name, vendor, model, ip_address, port, credential_id, location").
		Where("device_name = ? AND deleted_at IS NULL", deviceName).First(&dev).Error; err != nil {
		fmt.Fprintf(os.Stderr, "设备不存在: %s (%v)\n", deviceName, err)
		os.Exit(4)
	}
	var cred Cred
	if err := dbh.Table("sys_auth_credential").Where("id = ?", *dev.CredentialID).First(&cred).Error; err != nil {
		fmt.Fprintf(os.Stderr, "凭据读取失败: %v\n", err)
		os.Exit(4)
	}
	plain, err := cipher.Decrypt(cred.Password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "凭据解密失败: %v\n", err)
		os.Exit(4)
	}
	fmt.Printf("# 设备 %s | %s %s | %s | user=%s | 接口=%s\n",
		dev.DeviceName, dev.Vendor, dev.Model, dev.IPAddress, cred.Username, iface)

	switch mode {
	case "baseline":
		runBaseline(dev, cred.Username, plain, iface, outFile)
	case "probe":
		runProbe(dev, cred.Username, plain, iface, outFile)
	case "raw":
		runRaw(dev, cred.Username, plain, iface, rawCmds, rawView, outFile)
	case "exec":
		if action == "" {
			fmt.Fprintln(os.Stderr, "exec 模式需要 --action")
			os.Exit(2)
		}
		runExec(dev, cred.Username, plain, iface, action, apply, outFile)
	}
}

// ---------- 渲染 ----------

func actionSet() []portcollection.PortAction {
	return []portcollection.PortAction{
		portcollection.ActionShutdown,
		portcollection.ActionUndoShutdown,
		portcollection.ActionDescription,
		portcollection.ActionDot1xEnable,
		portcollection.ActionDot1xDisable,
		portcollection.ActionSetAccessVLAN,
		portcollection.ActionPortBinding,
		portcollection.TrunkSetNative,
		portcollection.TrunkSetAllowed,
		portcollection.TrunkSetHybrid,
		portcollection.PortSecEnable,
		portcollection.PortSecDisable,
		portcollection.PortSecMaxMac,
		portcollection.PortSecSticky,
		portcollection.PortSecStaticBind,
	}
}

func paramsFor(action portcollection.PortAction, iface string) portcollection.PortTemplateParams {
	limit := 1
	p := portcollection.PortTemplateParams{InterfaceName: iface}
	switch action {
	case portcollection.ActionDescription:
		p.Description = testDesc
	case portcollection.ActionDot1xEnable:
		p.Dot1xUserLimit = &limit
	case portcollection.ActionSetAccessVLAN:
		p.VLANID = testVLAN
	case portcollection.ActionPortBinding:
		p.BindOp = "add"
		p.IPAddress = testIP
		p.MACAddress = testMAC
	case portcollection.TrunkSetNative:
		p.NativeVLAN = testVLAN
	case portcollection.TrunkSetAllowed:
		p.AllowedVLANs = []int{100, 200}
		p.AllowedMode = "add"
	case portcollection.TrunkSetHybrid:
		p.HybridTagged = []int{30}
		p.HybridUntagged = []int{10, 20}
	case portcollection.PortSecMaxMac:
		p.MaxMACCount = 5
	case portcollection.PortSecSticky:
		p.MACAddress = testMAC
	case portcollection.PortSecStaticBind:
		p.MACAddress = testMAC
		p.VLANID = testVLAN
	}
	return p
}

func renderAll() {
	vendors := []models.DeviceVendor{models.VendorHuawei, models.VendorRuijie}
	ifaces := map[models.DeviceVendor]string{
		models.VendorHuawei: "10GE5/0/0",
		models.VendorRuijie: "GigabitEthernet 0/1",
	}
	for _, v := range vendors {
		fmt.Printf("\n########## %s ##########\n", v)
		for _, a := range actionSet() {
			cmds, err := portcollection.RenderCommand(v, a, paramsFor(a, ifaces[v]))
			if err != nil {
				fmt.Printf("%-22s ERROR: %v\n", a, err)
				continue
			}
			fmt.Printf("%-22s | %s\n", a, strings.Join(cmds, " ; "))
		}
	}
}

// ---------- SSH 会话 ----------

type session struct {
	client *ssh.Client
	sess   *ssh.Session
	buf    *safeBuf
	stdin  io.WriteCloser
}

func dial(dev Device, user, pass string) (*session, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	c, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", dev.IPAddress, portOf(dev.Port)), cfg)
	if err != nil {
		return nil, err
	}
	s, err := c.NewSession()
	if err != nil {
		c.Close()
		return nil, err
	}
	buf := &safeBuf{}
	s.Stdout = buf
	s.Stderr = buf
	stdin, err := s.StdinPipe()
	if err != nil {
		c.Close()
		return nil, err
	}
	// 网络设备 CLI 需要 PTY 才能正常回显（锐捷尤甚）
	_ = s.RequestPty("vt100", 200, 60, ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 38400})
	if err := s.Shell(); err != nil {
		c.Close()
		return nil, err
	}
	go func() { _ = s.Wait() }()
	time.Sleep(1500 * time.Millisecond) // 等登录横幅
	// 敲一个回车让 CLI 输出 prompt（华为/锐捷都需要）
	_, _ = stdin.Write([]byte("\r\n"))
	time.Sleep(1200 * time.Millisecond)
	return &session{client: c, sess: s, stdin: stdin, buf: buf}, nil
}

func (s *session) close() {
	_ = s.stdin.Close()
	_ = s.sess.Close()
	_ = s.client.Close()
}

// send 发一条命令并等待输出稳定（不依赖 prompt 特征，兼容不同厂商 CLI）
// sendExpect 发送命令并等待输出中出现期望的视图标记（未出现则重发一次），
// 用于确保进入配置视图/接口视图后再执行下一条命令。
func (s *session) sendExpect(cmd string, markers []string, wait time.Duration) string {
	out := s.send(cmd, wait)
	for _, m := range markers {
		if strings.Contains(out, m) {
			return out
		}
	}
	// 未命中标记 → 重发一次并再等
	fmt.Printf("[retry] 命令 %q 未见标记 %v，重发一次\n", cmd, markers)
	out2 := s.send(cmd, wait)
	return out2
}

func (s *session) send(cmd string, wait time.Duration) string {
	before := len(s.buf.String())
	_, _ = s.stdin.Write([]byte(cmd + "\r\n"))
	start := time.Now()
	deadline := start.Add(wait)
	lastLen := before
	stable := 0
	echoed := false
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		out := s.buf.String()
		cur := len(out)
		if !echoed && strings.Contains(out[before:], cmd) {
			// 设备已完整回显该命令（说明接收完整）→ 进入稳定等待
			echoed = true
			stable = 0
			lastLen = cur
			continue
		}
		if cur == lastLen {
			stable++
			// 回显命中且连续 600ms 无新输出 → 命令执行完；
			// 未命中回显则最多等 2.5s 也放行（部分设备回显含控制序列无法精确匹配）
			if stable >= 3 && (echoed || time.Since(start) > 2500*time.Millisecond) {
				break
			}
		} else {
			stable = 0
		}
		lastLen = cur
	}
	return s.buf.String()[before:]
}

func (s *session) all() string { return s.buf.String() }

func portOf(p int) int {
	if p == 0 {
		return 22
	}
	return p
}

// ---------- 基线采集 ----------

func baselineCmds(vendor string, iface string) []string {
	if vendor == "ruijie" {
		return []string{
			"show running-config interface " + iface,
			"show interfaces status " + iface,
			"show port-security interface " + iface,
			"show port-security binding",
			"show vlan",
		}
	}
	return []string{
		"display current-configuration interface " + iface,
		"display interface " + iface,
		"display port vlan " + iface,
		"display port-security " + iface,
		"display user-bind static all",
	}
}

func runBaseline(dev Device, user, pass, iface, outFile string) {
	s, err := dial(dev, user, pass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SSH 失败: %v\n", err)
		os.Exit(5)
	}
	defer s.close()
	var sb strings.Builder
	for _, c := range baselineCmds(dev.Vendor, iface) {
		out := s.send(c, 6*time.Second)
		sb.WriteString("\n########## " + c + " ##########\n" + out + "\n")
		fmt.Printf("\n########## %s ##########\n%s\n", c, out)
	}
	if outFile != "" {
		_ = os.WriteFile(outFile, []byte(sb.String()), 0o644)
		fmt.Printf("\n[基线已保存] %s\n", outFile)
	}
}

// ---------- 关键字探测（只读，零配置变更）----------

func probeCmds(vendor, iface string) []string {
	if vendor == "ruijie" {
		return []string{
			"configure terminal",
			"interface " + iface,
			"switchport ?",
			"dot1x ?",
			"end",
			"exit",
		}
	}
	return []string{
		"system-view",
		"interface " + iface,
		"port-security ?",
		"port ?",
		"authentication-profile ?",
		"user-bind ?",
		"undo ?",
		"quit",
		"return",
	}
}

func runProbe(dev Device, user, pass, iface, outFile string) {
	s, err := dial(dev, user, pass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SSH 失败: %v\n", err)
		os.Exit(5)
	}
	defer s.close()
	var sb strings.Builder
	for _, c := range probeCmds(dev.Vendor, iface) {
		out := s.send(c, 5*time.Second)
		sb.WriteString("\n########## " + c + " ##########\n" + out + "\n")
		fmt.Printf("\n########## %s ##########\n%s\n", c, out)
	}
	if outFile != "" {
		_ = os.WriteFile(outFile, []byte(sb.String()), 0o644)
		fmt.Printf("\n[探测结果已保存] %s\n", outFile)
	}
}

// ---------- 执行动作（含自动还原与基线 diff 核对）----------

// expandInterfaceName 是 portwrite.expandInterfaceName 的测试副本
// （该函数为包内私有，无法跨包调用）。语义必须与产品一致。
func expandInterfaceName(name string, vendor models.DeviceVendor) string {
	if name == "" {
		return name
	}
	type rep struct{ short, full string }
	reps := []rep{
		{"XGE", "TenGigabitEthernet"},
		{"HGE", "HundredGigE"},
		{"TWE", "TwentyFiveGigE"},
		{"FOE", "FortyGigE"},
		{"GE", "GigabitEthernet"},
		{"FE", "FastEthernet"},
	}
	if vendor == models.VendorHuawei {
		return name
	}
	upper := strings.ToUpper(name)
	sep := ""
	if vendor == models.VendorRuijie {
		sep = " "
	}
	for _, r := range reps {
		if strings.HasPrefix(upper, r.short) {
			rest := name[len(r.short):]
			return r.full + sep + rest
		}
	}
	return name
}

// parseConfigLines 从接口配置输出中提取配置行（去掉回显、提示符与噪声）
func parseConfigLines(out string) []string {
	var lines []string
	for _, raw := range strings.Split(out, "\n") {
		l := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if l == "" || l == "#" {
			continue
		}
		if strings.HasPrefix(l, "interface ") {
			continue
		}
		if strings.HasPrefix(l, "display ") || strings.HasPrefix(l, "show ") {
			continue
		}
		if strings.Contains(l, "Invalid input") || strings.Contains(l, "Unrecognized command") ||
			strings.Contains(l, "Unknown command") || strings.Contains(l, "Error:") ||
			strings.Contains(l, "Incomplete command") || strings.Contains(l, "Too many parameters") ||
			strings.Contains(l, "Building configuration") || strings.Contains(l, "Current configuration") ||
			strings.Contains(l, "Enter configuration commands") {
			continue
		}
		if strings.HasPrefix(l, "%") || strings.HasPrefix(l, "Info:") || strings.HasPrefix(l, "Warning:") {
			continue
		}
		// 过滤视图退出命令与设备提示符（还原时不可重放这些行）
		if l == "return" || l == "quit" || l == "end" {
			continue
		}
		if strings.HasPrefix(l, "<") && strings.HasSuffix(l, ">") {
			continue
		}
		if strings.Contains(l, "(config") {
			continue
		}
		// 行首提示符残留（如 <CX-HUB># 或 hostname(config-if)#）
		if i := strings.Index(l, "#"); i >= 0 && i < 60 {
			head := l[:i+1]
			if strings.Contains(head, ">") || strings.Contains(head, "(") {
				l = strings.TrimSpace(l[i+1:])
				if l == "" {
					continue
				}
			}
		}
		lines = append(lines, l)
	}
	return lines
}

// undoCommandsFor 返回该动作对应的撤销命令（按厂商）
func undoCommandsFor(action portcollection.PortAction, vendor models.DeviceVendor, params portcollection.PortTemplateParams) []string {
	isRJ := vendor == models.VendorRuijie
	pre := "undo "
	if isRJ {
		pre = "no "
	}
	switch action {
	case portcollection.ActionShutdown:
		return []string{pre + "shutdown"}
	case portcollection.ActionUndoShutdown:
		return []string{}
	case portcollection.ActionDescription:
		return []string{pre + "description"}
	case portcollection.ActionDot1xEnable:
		if isRJ {
			return []string{"no dot1x port-control auto"}
		}
		return []string{"undo authentication-profile dot1x"}
	case portcollection.ActionDot1xDisable:
		if isRJ {
			return []string{"dot1x port-control auto"}
		}
		return []string{"authentication-profile dot1x"}
	case portcollection.ActionSetAccessVLAN:
		if isRJ {
			return []string{"no switchport access vlan"}
		}
		return []string{"undo port link-type", "undo port default vlan"}
	case portcollection.ActionPortBinding:
		if params.BindOp == "remove" {
			return []string{}
		}
		if isRJ {
			return []string{"no switchport port-security binding " + params.IPAddress}
		}
		cmd := "undo user-bind static ip-address " + params.IPAddress
		if params.MACAddress != "" {
			cmd += " mac-address " + params.MACAddress
		}
		return []string{cmd}
	case portcollection.TrunkSetNative:
		if isRJ {
			return []string{"no switchport trunk native vlan"}
		}
		return []string{"undo port trunk pvid vlan"}
	case portcollection.TrunkSetAllowed:
		vs := make([]string, 0, len(params.AllowedVLANs))
		for _, v := range params.AllowedVLANs {
			vs = append(vs, fmt.Sprint(v))
		}
		if isRJ {
			if params.AllowedMode == "add" {
				return []string{"no switchport trunk allowed vlan " + strings.Join(vs, ",")}
			}
			return []string{}
		}
		if params.AllowedMode == "add" {
			return []string{"undo port trunk allow-pass vlan " + strings.Join(vs, " ")}
		}
		return []string{}
	case portcollection.TrunkSetHybrid:
		if isRJ {
			return []string{"no switchport hybrid"}
		}
		return []string{"undo port hybrid tagged vlan", "undo port hybrid untagged vlan", "undo port link-type"}
	case portcollection.PortSecEnable:
		return []string{pre + "port-security enable"}
	case portcollection.PortSecDisable:
		if isRJ {
			return []string{"switchport port-security"}
		}
		return []string{"port-security enable"}
	case portcollection.PortSecMaxMac:
		if isRJ {
			return []string{"no switchport port-security maximum"}
		}
		return []string{"undo port-security max-mac-num"}
	case portcollection.PortSecSticky:
		if isRJ {
			return []string{"no switchport port-security mac-address sticky"}
		}
		return []string{"undo port-security mac-address sticky"}
	case portcollection.PortSecStaticBind:
		if isRJ {
			return []string{"no switchport port-security mac-address " + strings.ToLower(strings.ReplaceAll(params.MACAddress, "-", ""))}
		}
		return []string{"undo port-security mac-address sticky " + params.MACAddress}
	}
	return []string{}
}

// verifyCmdsFor 该动作的验证命令
func verifyCmdsFor(vendor models.DeviceVendor, cliIface string) []string {
	if vendor == models.VendorRuijie {
		return []string{
			"show running-config interface " + cliIface,
			"show port-security interface " + cliIface,
		}
	}
	return []string{
		"display current-configuration interface " + cliIface,
		"display interface " + cliIface,
	}
}

// collectConfig 采集接口配置（华为 display current-configuration / 锐捷 show running-config）
func collectConfig(s *session, vendor models.DeviceVendor, cliIface string) string {
	cmd := "display current-configuration interface " + cliIface
	if vendor == models.VendorRuijie {
		cmd = "show running-config interface " + cliIface
	}
	return s.send(cmd, 7*time.Second)
}

// enterConfig 进入配置视图与接口视图
func enterConfig(s *session, vendor models.DeviceVendor, cliIface string, cmds []string) {
	cfgMarker := []string{"(config)"}
	ifaceMarker := []string{"(config-if"}
	if vendor == models.VendorHuawei {
		// 华为：system-view 后 prompt 变为 <hostname>，接口视图 prompt 含 (config-if-... 或 GE1/0/0]
		cfgMarker = []string{"[", "system-view"}
		ifaceMarker = []string{"-", "]"}
	}
	out := s.sendExpect("configure terminal", cfgMarker, 3*time.Second)
	if vendor == models.VendorHuawei {
		out = s.sendExpect("system-view", cfgMarker, 3*time.Second)
	}
	fmt.Printf("[enterConfig] 进入配置视图: %s\n", strings.TrimSpace(out))
	if len(cmds) > 0 && strings.HasPrefix(strings.TrimSpace(cmds[0]), "interface ") {
		o := s.sendExpect(cmds[0], ifaceMarker, 3*time.Second)
		fmt.Printf("[enterConfig] 进入接口视图: %s\n", strings.TrimSpace(o))
		return
	}
	o := s.sendExpect("interface "+cliIface, ifaceMarker, 3*time.Second)
	fmt.Printf("[enterConfig] 进入接口视图: %s\n", strings.TrimSpace(o))
}

func leaveConfig(s *session, vendor models.DeviceVendor) {
	if vendor == models.VendorRuijie {
		s.send("end", 2*time.Second)
	} else {
		s.send("return", 2*time.Second)
	}
}

func diffLines(base, after []string) []string {
	set := map[string]bool{}
	for _, l := range base {
		set[l] = true
	}
	setA := map[string]bool{}
	for _, l := range after {
		setA[l] = true
	}
	var diff []string
	for _, l := range base {
		if !setA[l] {
			diff = append(diff, "缺失: "+l)
		}
	}
	for _, l := range after {
		if !set[l] {
			diff = append(diff, "多出: "+l)
		}
	}
	return diff
}

func runExec(dev Device, user, pass, iface, actionName string, apply bool, outFile string) {
	action := portcollection.PortAction(actionName)
	vendor := models.DeviceVendor(dev.Vendor)
	// 库里的归一化短名 → 厂商 CLI 全名（与产品 expandInterfaceName 语义一致）
	cliIface := expandInterfaceName(iface, vendor)
	params := paramsFor(action, cliIface) // 与产品一致：先 expand 再渲染
	cmds, err := portcollection.RenderCommand(vendor, action, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "渲染失败: %v\n", err)
		os.Exit(6)
	}
	fmt.Printf("\n[渲染命令] %s → %s\n", action, strings.Join(cmds, " ; "))
	if !apply {
		fmt.Println("[DRY-RUN] 未加 --apply，不下发。请人工核对后再执行。")
		return
	}
	s, err := dial(dev, user, pass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SSH 失败: %v\n", err)
		os.Exit(5)
	}
	defer s.close()

	var log strings.Builder
	record := func(title, body string) {
		log.WriteString("\n########## " + title + " ##########\n" + body + "\n")
		fmt.Printf("\n########## %s ##########\n%s\n", title, body)
	}

	// ① 动作前基线
	baseLines := parseConfigLines(collectConfig(s, vendor, cliIface))
	record("动作前基线", strings.Join(baseLines, "\n"))

	// ② 下发动作
	record("下发动作", strings.Join(cmds, " ; "))
	enterConfig(s, vendor, cliIface, cmds)
	for _, c := range cmds {
		if strings.HasPrefix(strings.TrimSpace(c), "interface ") {
			continue
		}
		record("下发: "+c, s.send(c, 5*time.Second))
	}
	leaveConfig(s, vendor)

	// ③ 动作后验证
	for _, v := range verifyCmdsFor(vendor, cliIface) {
		record("验证: "+v, s.send(v, 6*time.Second))
	}

	// ④ 还原：undo 动作影响项 → 重放基线行
	record("还原开始", "")
	undos := undoCommandsFor(action, vendor, params)
	enterConfig(s, vendor, cliIface, nil)
	for _, u := range undos {
		record("undo: "+u, s.send(u, 5*time.Second))
	}
	for _, l := range baseLines {
		record("restore: "+l, s.send(l, 4*time.Second))
	}
	leaveConfig(s, vendor)

	// ⑤ 还原核对（若多出 shutdown → 自动补 no shutdown 后复核）
	afterLines := parseConfigLines(collectConfig(s, vendor, cliIface))
	diff := diffLines(baseLines, afterLines)
	if containsShutdownExtra(diff) {
		fmt.Println("[restore-fix] 检测到还原后多出 shutdown，自动补 no shutdown")
		noShutdown := "no shutdown"
		if vendor == models.VendorHuawei {
			noShutdown = "undo shutdown"
		}
		enterConfig(s, vendor, cliIface, nil)
		record("restore-fix: "+noShutdown, s.send(noShutdown, 5*time.Second))
		leaveConfig(s, vendor)
		afterLines = parseConfigLines(collectConfig(s, vendor, cliIface))
		diff = diffLines(baseLines, afterLines)
	}
	record("还原后配置", strings.Join(afterLines, "\n"))
	if len(diff) == 0 {
		record("还原核对", "✅ 与基线完全一致")
	} else {
		record("还原核对", fmt.Sprintf("⚠️ 与基线存在 %d 处差异:\n%s", len(diff), strings.Join(diff, "\n")))
	}
	if outFile != "" {
		_ = os.WriteFile(outFile, []byte(log.String()), 0o644)
		fmt.Printf("\n[执行记录已保存] %s\n", outFile)
	}
}

// ---------- 配置解析 ----------

func loadConfig(path string) (host, port, user, pass, db, sm4Key string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读配置失败: %v\n", err)
		os.Exit(2)
	}
	section := ""
	for _, ln := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		m := regexp.MustCompile(`^(\s+)(\w+):\s*"?([^"\n]*)"?`).FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		key, val := m[2], strings.TrimSpace(m[3])
		switch {
		case section == "database":
			switch key {
			case "host":
				host = val
			case "port":
				port = val
			case "user":
				user = val
			case "password":
				pass = val
			case "dbname":
				db = val
			}
		case section == "security":
			if key == "sm4_key" {
				sm4Key = val
			}
		}
	}
	return
}

func readEnvFromFile(path, key string) (string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, kv := range strings.Split(string(raw), "\x00") {
		if strings.HasPrefix(kv, key+"=") {
			return strings.TrimPrefix(kv, key+"="), true
		}
	}
	return "", false
}

var _ = json.Marshal

func runRaw(dev Device, user, pass, iface, cmdsRaw, view, outFile string) {
	vendor := models.DeviceVendor(dev.Vendor)
	cliIface := expandInterfaceName(iface, vendor)
	var cmds []string
	for _, c := range strings.Split(cmdsRaw, ";") {
		c = strings.TrimSpace(c)
		if c != "" {
			cmds = append(cmds, c)
		}
	}
	s, err := dial(dev, user, pass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SSH 失败: %v\n", err)
		os.Exit(5)
	}
	defer s.close()
	switch view {
	case "user":
		// 已在用户视图，直接发
	case "config":
		if vendor == models.VendorRuijie {
			s.sendExpect("configure terminal", []string{"(config)"}, 3*time.Second)
		} else {
			s.sendExpect("system-view", []string{"[", "system-view"}, 3*time.Second)
		}
	default:
		enterConfig(s, vendor, cliIface, nil)
	}
	for _, c := range cmds {
		out := s.send(c, 5*time.Second)
		fmt.Printf("\n--- %s ---\n%s\n", c, out)
	}
	switch view {
	case "config":
		if vendor == models.VendorRuijie {
			s.send("end", 2*time.Second)
		} else {
			s.send("return", 2*time.Second)
		}
	case "interface":
		leaveConfig(s, vendor)
	}
	out := collectConfig(s, vendor, cliIface)
	fmt.Printf("\n===== 当前配置 =====\n%s\n", out)
	if outFile != "" {
		_ = os.WriteFile(outFile, []byte(out), 0o644)
	}
}

// containsShutdownExtra 判断差异中是否含“多出: shutdown”
func containsShutdownExtra(diff []string) bool {
	for _, d := range diff {
		if strings.TrimSpace(d) == "多出: shutdown" {
			return true
		}
	}
	return false
}
