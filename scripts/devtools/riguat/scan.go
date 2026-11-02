package main

// 空闲候选测试口扫描（只读）。
//
// 用途（task-30 项 1 的前置调研）：锐捷 hybrid / trunk 的真机实跑需要一个
// **空闲且无 description** 的端口 —— 在用工位口（如 GigabitEthernet 1/1，
// access vlan 318 / description 3D001）下发 hybrid 会改变转发模式、影响业务。
//
// 本模式**只查库**，不连设备、不解密凭据、不下发任何命令：
//   - 设备表 sys_network_device：按 model 关键字筛（默认 RS8607）
//   - 端口表 sys_device_port_status：采集快照（description / admin / oper / vlan）
//
// 候选判据（保守，宁可少报）：
//   - description 为空（无业务描述 ⇒ 不像在用工位口）
//   - admin_status = up（未 shutdown）
//
// 输出 oper_status（link 状态）供人工判断：link down 的口更"空闲"。
// 采集快照的时间以表内 updated_at 为准（新鲜度需人工确认，必要时先刷新采集）。

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// scanDevice 设备行（sys_network_device）。
type scanDevice struct {
	ID         string
	DeviceName string
	IPAddress  string
	Model      string
	Vendor     string
}

// scanPort 端口采集行（sys_device_port_status）。
//
// description / admin_status / oper_status 用 COALESCE 在 SQL 侧兜成非 NULL，
// 避免 NULL 扫进 string 字段。
type scanPort struct {
	DeviceID       string
	InterfaceName  string
	AdminStatus    string
	OperStatus     string
	Description    string
	VLAN           *int
	Dot1xUserLimit *int
	UpdatedAt      *time.Time
}

// runDBCheck 诊断模式：打印 config.yaml 的 database 段解析结果（**不打印密码**），
// 用于排查「解析出空 dbname / 密码带引号」这类配置问题。
func runDBCheck(cfg dbConfig, rep *report) {
	rep.section("config.yaml database 段解析结果（密码不打印）")
	rep.line("Host   : %q", cfg.Host)
	rep.line("Port   : %q", cfg.Port)
	rep.line("User   : %q", cfg.User)
	rep.line("DBName : %q", cfg.Name)
	rep.line("Pass   : <已读取，长度 %d>", len(cfg.Pass))
}

// runScanIdle 只读扫描候选测试口并输出清单（不连设备、不改任何配置）。
func runScanIdle(db *gorm.DB, rep *report, modelKeyword string) {
	rep.section("空闲候选口扫描（只读：仅查库，不连设备、不下发命令）")

	var devs []scanDevice
	if err := db.Table("sys_network_device").
		Select("id, device_name, ip_address, model, vendor").
		Where("model ILIKE ? AND deleted_at IS NULL", "%"+modelKeyword+"%").
		Order("ip_address").
		Scan(&devs).Error; err != nil {
		rep.line("设备查询失败: %v", err)
		return
	}
	rep.line("匹配设备 %d 台（model LIKE %%%s%%）", len(devs), modelKeyword)
	if len(devs) == 0 {
		rep.line("提示：库里没有匹配该 model 关键字的设备，可改用 --scan-model 关键字重试")
		return
	}

	totalPorts, candidates := 0, 0
	for _, d := range devs {
		ports, err := queryScanPorts(db, d.ID)
		if err != nil {
			rep.line("  %s 端口查询失败: %v", d.DeviceName, err)
			continue
		}
		totalPorts += len(ports)

		rep.section(fmt.Sprintf("设备 %s（%s %s，%s）", d.DeviceName, d.Model, d.IPAddress, d.Vendor))
		rep.line("采集端口数: %d", len(ports))
		for _, p := range ports {
			if p.Description != "" {
				continue // 有 description → 业务口，排除
			}
			if !strings.EqualFold(p.AdminStatus, "up") {
				continue // 非 up（shutdown / 未知）→ 不是可用空闲口
			}
			candidates++
			rep.line("  [候选] %-22s admin=%-4s oper=%-8s desc=%q vlan=%v dot1x_limit=%v 采集=%s",
				p.InterfaceName, p.AdminStatus, p.OperStatus, p.Description,
				intPtrStr(p.VLAN), intPtrStr(p.Dot1xUserLimit), timePtrStr(p.UpdatedAt))
		}
	}

	rep.section("汇总")
	rep.line("扫描端口总数: %d", totalPorts)
	rep.line("候选（无 description + admin up）: %d", candidates)
	rep.line("说明：候选仅代表「采集快照里无描述且未 shutdown」；link 状态与真实占用需现场确认。")
}

// queryScanPorts 查某设备的端口采集行。
//
// 容错：updated_at 列在部分环境的表结构里可能不存在 → 先带它查，失败则回退到
// 不含该列的查询（采集时间缺失不影响候选判定）。
func queryScanPorts(db *gorm.DB, deviceID string) ([]scanPort, error) {
	baseCols := "device_id, interface_name, " +
		"COALESCE(admin_status,'') AS admin_status, " +
		"COALESCE(oper_status,'') AS oper_status, " +
		"COALESCE(description,'') AS description, " +
		"vlan, dot1x_user_limit"
	var ports []scanPort
	err := db.Table("sys_device_port_status").
		Select(baseCols+", updated_at").
		Where("device_id = ?", deviceID).
		Order("interface_name").
		Scan(&ports).Error
	if err == nil {
		return ports, nil
	}
	ports = nil
	if err2 := db.Table("sys_device_port_status").
		Select(baseCols).
		Where("device_id = ?", deviceID).
		Order("interface_name").
		Scan(&ports).Error; err2 != nil {
		return nil, err2
	}
	return ports, nil
}

// intPtrStr 可空整数的展示（NULL 显示为 "-"）。
func intPtrStr(v *int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}

// timePtrStr 可空时间的展示（NULL 显示为 "-"）。
func timePtrStr(v *time.Time) string {
	if v == nil {
		return "-"
	}
	return v.Format("2006-01-02 15:04:05")
}
