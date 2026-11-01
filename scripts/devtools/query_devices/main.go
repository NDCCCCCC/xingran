// 临时工具：查询 PG 中的网络设备资产（Linux 版，跑完即删）
// 用法: ./query_devices [config.yaml 路径]
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Device struct {
	DeviceName      string
	Vendor          string
	Model           string
	IPAddress       string
	Port            int
	CredentialID    string
	Description     string
	SerialNumber    string
	SoftwareVersion string
	Status          int
	Location        string
	DeptName        string
}

func main() {
	host, port, user, pass, db := "10.62.10.34", "5432", "postgres", "", "ruoyi"
	if len(os.Args) > 1 {
		raw, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "读配置失败: %v\n", err)
			os.Exit(2)
		}
		host, port, user, pass, db = parseDB(string(raw))
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		pass = v
	}
	fmt.Fprintf(os.Stderr, "目标: %s:%s db=%s user=%s pwd_len=%d\n", host, port, db, user, len(pass))

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, db)
	dbh, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接失败: %v\n", err)
		os.Exit(3)
	}

	var devices []Device
	if err := dbh.Table("sys_network_device").Select(
		"device_name, vendor, model, ip_address, port, credential_id, description, serial_number, software_version, status, location, dept_name",
	).Where("deleted_at IS NULL").Order("vendor, device_name").Find(&devices).Error; err != nil {
		fmt.Fprintf(os.Stderr, "设备查询失败: %v\n", err)
		os.Exit(4)
	}
	fmt.Printf("=== 设备资产 %d 条 ===\n", len(devices))
	for _, d := range devices {
		fmt.Printf("- %s | vendor=%s | model=%s | %s:%d | cred=%s | fw=%s | status=%d | loc=%s | sn=%s | desc=%s\n",
			d.DeviceName, d.Vendor, d.Model, d.IPAddress, d.Port, dash(d.CredentialID),
			dasht(d.SoftwareVersion), d.Status, dasht(d.Location), dasht(d.SerialNumber), dasht(d.Description))
	}

	// 凭据元信息（不取密码密文字段）
	type Cred struct {
		ID       string
		Name     string
		Username string
		Vendor   string
		Protocol string
		Port     int
	}
	var creds []Cred
	if err := dbh.Table("sys_device_credential").Select("id, name, username, vendor, protocol, port").Find(&creds).Error; err == nil {
		fmt.Printf("\n=== 设备凭据 %d 条（仅元信息）===\n", len(creds))
		for _, c := range creds {
			fmt.Printf("- %s | user=%s | vendor=%s | proto=%s | port=%d | id=%s\n",
				c.Name, dasht(c.Username), dasht(c.Vendor), dasht(c.Protocol), c.Port, c.ID)
		}
	} else {
		fmt.Fprintf(os.Stderr, "凭据查询失败: %v\n", err)
	}

	// 端口状态（抽样，验证可用端口）
	type PortStat struct {
		DeviceID      string
		InterfaceName string
		Status        string
		VlanID        int
	}
	var ps []PortStat
	if err := dbh.Table("sys_device_port_status").Select("device_id, interface_name, status, vlan_id").Where("deleted_at IS NULL").Limit(30).Find(&ps).Error; err == nil {
		fmt.Printf("\n=== 端口状态抽样 %d 条 ===\n", len(ps))
		for _, p := range ps {
			fmt.Printf("- dev=%s | %s | status=%s | vlan=%d\n", p.DeviceID, p.InterfaceName, p.Status, p.VlanID)
		}
	}
}

func dash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
func dasht(s string) string { return dash(s) }

// parseDB 从 yaml 的 database: 段提取连接参数
func parseDB(raw string) (host, port, user, pass, db string) {
	lines := strings.Split(raw, "\n")
	inDB := false
	for _, ln := range lines {
		if strings.HasPrefix(ln, "database:") {
			inDB = true
			continue
		}
		if inDB {
			if ln != "" && !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
				break
			}
			m := regexp.MustCompile(`^\s+(\w+):\s*"?([^"\n]*)"?`).FindStringSubmatch(ln)
			if m == nil {
				continue
			}
			switch m[1] {
			case "host":
				host = strings.TrimSpace(m[2])
			case "port":
				port = strings.TrimSpace(m[2])
			case "user":
				user = strings.TrimSpace(m[2])
			case "password":
				pass = strings.TrimSpace(m[2])
			case "dbname":
				db = strings.TrimSpace(m[2])
			}
		}
	}
	return
}
