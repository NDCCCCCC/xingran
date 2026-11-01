// 临时工具：真机连通性探测（跑完即删）
// 从库中读设备与凭据 → SM4 解密 → SSH 登录 → 执行只读命令 → 输出报告
// 安全约束：绝不输出密码，只输出连通性、登录结果与设备版本输出摘要。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	"golang.org/x/crypto/ssh"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Device struct {
	DeviceName   string
	Vendor       string
	Model        string
	IPAddress    string
	Port         int
	CredentialID *string
	Location     string
	Status       int
}

type Cred struct {
	ID             string
	CredentialName string
	ProtocolType   string
	Username       string
	Password       string
	EnablePassword string
}

func main() {
	var cfgPath, filter, ipSel, sm4EnvFile string
	var timeoutSec int
	flag.StringVar(&cfgPath, "config", "/app/szh/configs/config.yaml", "配置文件路径")
	flag.StringVar(&filter, "filter", "", "设备名/型号子串过滤")
	flag.StringVar(&ipSel, "ip", "", "精确 IP")
	flag.StringVar(&sm4EnvFile, "sm4-env-file", "", "从该文件的 NUL 分隔环境变量中取 SM4_KEY（如 /proc/<pid>/environ）")
	flag.IntVar(&timeoutSec, "timeout", 12, "命令读取等待秒数")
	flag.Parse()

	host, port, user, pass, db, sm4Key := loadConfig(cfgPath)
	if sm4EnvFile != "" {
		if k, ok := readEnvFromFile(sm4EnvFile, "SM4_KEY"); ok {
			sm4Key = k
		} else {
			fmt.Fprintf(os.Stderr, "未在 %s 找到 SM4_KEY\n", sm4EnvFile)
			os.Exit(3)
		}
	}
	if sm4Key == "" {
		sm4Key = "dGVzdC1zZWNyZXQxNiEhIQ==" // 仓库默认（与代码一致）
	}
	fmt.Fprintf(os.Stderr, "DB=%s:%s/%s user=%s pwd_len=%d sm4key_len=%d\n",
		host, port, db, user, len(pass), len(sm4Key))

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, db)
	dbh, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB 连接失败: %v\n", err)
		os.Exit(3)
	}

	cipher, err := crypto.NewSM4Cipher(sm4Key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SM4 cipher 失败: %v\n", err)
		os.Exit(3)
	}

	var devices []Device
	q := dbh.Table("sys_network_device").Select("device_name, vendor, model, ip_address, port, credential_id, location, status").
		Where("deleted_at IS NULL")
	if filter != "" {
		q = q.Where("device_name LIKE ? OR model LIKE ?", "%"+filter+"%", "%"+filter+"%")
	}
	if ipSel != "" {
		q = q.Where("ip_address = ?", ipSel)
	}
	if err := q.Order("vendor, device_name").Find(&devices).Error; err != nil {
		fmt.Fprintf(os.Stderr, "设备查询失败: %v\n", err)
		os.Exit(4)
	}

	// 凭据缓存
	credCache := map[string]*Cred{}
	for _, d := range devices {
		if d.CredentialID == nil || *d.CredentialID == "" {
			continue
		}
		if _, ok := credCache[*d.CredentialID]; ok {
			continue
		}
		var c Cred
		if err := dbh.Table("sys_auth_credential").Where("id = ?", *d.CredentialID).First(&c).Error; err != nil {
			fmt.Fprintf(os.Stderr, "凭据读取失败 %s: %v\n", *d.CredentialID, err)
			continue
		}
		credCache[*d.CredentialID] = &c
	}

	fmt.Printf("=== 探测 %d 台设备（只读命令）===\n", len(devices))
	ok, fail := 0, 0
	for _, d := range devices {
		var c *Cred
		if d.CredentialID != nil {
			c = credCache[*d.CredentialID]
		}
		if c == nil {
			fmt.Printf("[SKIP] %s (%s) 无关联凭据\n", d.DeviceName, d.IPAddress)
			continue
		}
		plain, derr := cipher.Decrypt(c.Password)
		if derr != nil || plain == "" {
			fmt.Printf("[FAIL] %s (%s) 凭据解密失败: %v\n", d.DeviceName, d.IPAddress, derr)
			fail++
			continue
		}
		cmd := "display version"
		if strings.EqualFold(d.Vendor, "ruijie") {
			cmd = "show version"
		}
		out, serr := sshProbe(d.IPAddress, portOf(d.Port), c.Username, plain, cmd, timeoutSec)
		if serr != nil {
			fmt.Printf("[FAIL] %s (%s:%d) user=%s 登录/执行失败: %v\n",
				d.DeviceName, d.IPAddress, portOf(d.Port), c.Username, serr)
			fail++
			continue
		}
		ok++
		fmt.Printf("[OK]   %s | %s %s | %s:%d | user=%s | %s\n",
			d.DeviceName, d.Vendor, d.Model, d.IPAddress, portOf(d.Port), c.Username, summarize(out))
	}
	fmt.Printf("\n汇总: 成功 %d / 失败 %d\n", ok, fail)
}

func portOf(p int) int {
	if p == 0 {
		return 22
	}
	return p
}

// readEnvFromFile 从 NUL 分隔的环境变量文件（如 /proc/<pid>/environ）读取指定键
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

func sshProbe(host string, port int, user, password, cmd string, timeoutSec int) (string, error) {
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return "", fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("session: %w", err)
	}
	defer session.Close()
	var buf bytes.Buffer
	session.Stdout = &buf
	session.Stderr = &buf
	if err := session.Start(cmd); err != nil {
		return "", fmt.Errorf("start: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err = <-done:
		if err != nil {
			// 设备命令退出码常非 0（telnet/ssh 差异），只要有输出即视为已执行
			if buf.Len() == 0 {
				return "", fmt.Errorf("exec: %w", err)
			}
		}
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		_ = session.Close()
	}
	return buf.String(), nil
}

func summarize(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var keep []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "%") {
			continue
		}
		if strings.Contains(strings.ToLower(l), "password") || strings.Contains(strings.ToLower(l), "secret") {
			continue
		}
		keep = append(keep, l)
		if len(keep) >= 3 {
			break
		}
	}
	s := strings.Join(keep, " / ")
	if len(s) > 220 {
		s = s[:220] + "…"
	}
	return s
}

// loadConfig 解析 yaml：database 段 + security.sm4_key
func loadConfig(path string) (host, port, user, pass, db, sm4Key string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读配置失败: %v\n", err)
		os.Exit(2)
	}
	lines := strings.Split(string(raw), "\n")
	section := ""
	for _, ln := range lines {
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
	if v := os.Getenv("SM4_KEY"); v != "" {
		sm4Key = v
	}
	return
}
