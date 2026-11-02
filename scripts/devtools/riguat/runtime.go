package main

// 运行环境装配：配置 / 数据库 / 凭据 / SM4 密钥 / 产品执行通道。
//
// 安全约束：
//   - 设备密码全程只在内存中（由产品连接池内部解密），绝不打印、绝不落盘；
//   - SM4 密钥从"运行中后端进程的环境变量"读取（远程机 /proc/<pid>/environ），
//     打印时只报"来源进程 pid"，不打印密钥本身。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// dbConfig 数据库连接参数（来自后端 config.yaml 的 database 段）。
type dbConfig struct {
	Host string
	Port string
	User string
	Pass string
	Name string
}

// loadDBConfig 解析后端 config.yaml 的 database 段。
func loadDBConfig(path string) (dbConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return dbConfig{}, fmt.Errorf("读配置失败: %w", err)
	}
	section := ""
	re := regexp.MustCompile(`^(\s+)(\w+):\s*"?([^"\n]*)"?`)
	lines := strings.Split(string(raw), "\n")
	var cfg dbConfig
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		m := re.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		if section != "database" {
			continue
		}
		key, val := m[2], strings.TrimSpace(m[3])
		switch key {
		case "host":
			cfg.Host = val
		case "port":
			cfg.Port = val
		case "user":
			cfg.User = val
		case "password":
			cfg.Pass = val
		case "dbname":
			cfg.Name = val
		}
	}
	if cfg.Host == "" || cfg.Name == "" {
		return cfg, fmt.Errorf("配置 %s 中未解析出 database.host / database.dbname", path)
	}
	return cfg, nil
}

// openDB 连接数据库（只读用途，禁用日志）。
func openDB(cfg dbConfig) (*gorm.DB, error) {
	port := cfg.Port
	if port == "" {
		port = "5432"
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, port, cfg.User, cfg.Pass, cfg.Name)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, fmt.Errorf("DB 连接失败: %w", err)
	}
	return db, nil
}

// readEnvValueFromFile 从 NUL 分隔的环境变量文件里取指定 key（/proc/<pid>/environ 格式）。
func readEnvValueFromFile(path, key string) (string, bool) {
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

// discoverSM4Key 在 /proc 下自动寻找带 SM4_KEY 的进程环境。
//
// 返回密钥与来源 pid；找不到返回错误。密钥不打印。
func discoverSM4Key(procRoot string) (string, int, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return "", 0, fmt.Errorf("扫描 %s 失败: %w", procRoot, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		envPath := filepath.Join(procRoot, e.Name(), "environ")
		if v, ok := readEnvValueFromFile(envPath, "SM4_KEY"); ok && v != "" {
			return v, pid, nil
		}
	}
	return "", 0, fmt.Errorf("未在 %s/*/environ 中找到 SM4_KEY（可用 --backend-pid 或 --sm4-env-file 指定）", procRoot)
}

// resolveSM4Key 按优先级取 SM4 密钥：显式文件 > 指定 pid > 自动发现。
func resolveSM4Key(envFile string, pid int, procRoot string) (string, string, error) {
	if envFile != "" {
		v, ok := readEnvValueFromFile(envFile, "SM4_KEY")
		if !ok || v == "" {
			return "", "", fmt.Errorf("文件 %s 中未找到 SM4_KEY", envFile)
		}
		return v, "文件 " + envFile, nil
	}
	if pid > 0 {
		path := filepath.Join(procRoot, strconv.Itoa(pid), "environ")
		v, ok := readEnvValueFromFile(path, "SM4_KEY")
		if !ok || v == "" {
			return "", "", fmt.Errorf("%s 中未找到 SM4_KEY", path)
		}
		return v, fmt.Sprintf("进程 %d 的环境变量", pid), nil
	}
	v, foundPID, err := discoverSM4Key(procRoot)
	if err != nil {
		return "", "", err
	}
	return v, fmt.Sprintf("自动发现：进程 %d 的环境变量", foundPID), nil
}

// deviceInfo 设备基本信息（来自 sys_network_device）。
type deviceInfo struct {
	ID           string
	DeviceName   string
	Vendor       models.DeviceVendor
	Model        string
	IPAddress    string
	Port         int
	CredentialID *string
}

// loadDevice 按设备名查设备。
func loadDevice(db *gorm.DB, name string) (*deviceInfo, error) {
	var d deviceInfo
	err := db.Table("sys_network_device").
		Select("id, device_name, vendor, model, ip_address, port, credential_id").
		Where("device_name = ? AND deleted_at IS NULL", name).
		First(&d).Error
	if err != nil {
		return nil, fmt.Errorf("设备不存在: %s (%v)", name, err)
	}
	return &d, nil
}

// portRow 端口采集行（sys_device_port_status）。
type portRow struct {
	InterfaceName  string
	AdminStatus    string
	OperStatus     string
	Description    string
	VLAN           *int
	Dot1xUserLimit *int
}

// loadPort 查端口采集行（接口名为归一化短名，如 GE1/1）。
func loadPort(db *gorm.DB, deviceID, ifaceShort string) (*portRow, error) {
	var p portRow
	err := db.Table("sys_device_port_status").
		Select("interface_name, admin_status, oper_status, description, vlan, dot1x_user_limit").
		Where("device_id = ? AND interface_name = ?", deviceID, ifaceShort).
		First(&p).Error
	if err != nil {
		return nil, fmt.Errorf("端口采集行不存在: %s (%v)", ifaceShort, err)
	}
	return &p, nil
}

// expandInterfaceName 把归一化短名还原成设备 CLI 全名。
//
// 与 internal/services/portwrite/port_write_service.go 的 expandInterfaceName 语义一致
// （该函数为包内私有，无法跨包调用，故此处复制；改动须同步两边）。
//   - 锐捷 RGOS：接口类型与编号间带空格（interface GigabitEthernet 1/1）
//   - 华为 VRP：用归一化短名（GE1/0/0）
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
			return r.full + sep + name[len(r.short):]
		}
	}
	return name
}

// execChannel 产品 scrapli 执行通道。
//
// 组成与生产完全一致：
//
//	DeviceConnectionPool（连接池：查设备/凭据 → 解密密码 → 建 scrapligo 连接）
//	  → DeviceTaskScheduler（同设备串行）
//	    → DeviceExecutor.ExecuteCustom（产品写路径同款入口）
//	      → ScrapliWrapper.SendConfigs / SendCommand（锐捷 patched platform）
type execChannel struct {
	pool     *device.DeviceConnectionPool
	sched    *device.DeviceTaskScheduler
	executor *device.DeviceExecutor
}

// newExecChannel 用产品执行器组装执行通道。
func newExecChannel(db *gorm.DB, cipher *crypto.SM4Cipher) *execChannel {
	pool := device.NewDeviceConnectionPool(db, cipher, &device.PoolConfig{
		MaxIdle:        3 * time.Minute,
		MaxConnections: 4,
	})
	sched := device.NewDeviceTaskScheduler(pool, &device.SchedulerConfig{
		TaskTimeout: 5 * time.Minute,
		QueueSize:   10,
	})
	executor := device.NewDeviceExecutor(sched, &device.ExecutionConfig{
		MaxRetries:          1,
		Timeout:             3 * time.Minute,
		EnablePanicRecovery: true,
	})
	return &execChannel{pool: pool, sched: sched, executor: executor}
}

// Close 释放连接池与调度器。
func (c *execChannel) Close() {
	if c.sched != nil {
		c.sched.Stop()
	}
	if c.pool != nil {
		_ = c.pool.Close()
	}
}

// run 在设备上执行一段自定义逻辑（持设备锁 + 连接复用，与产品写路径一致）。
func (c *execChannel) run(ctx context.Context, deviceID string, timeout time.Duration, fn func(ctx context.Context, w *device.ScrapliWrapper) error) error {
	return c.executor.ExecuteCustom(ctx, deviceID, func(execCtx context.Context, pc *device.PooledConnection) error {
		return fn(execCtx, pc.GetWrapper())
	}, timeout)
}
