// dbprovision: 一次性补建远端 dev DB (Supabase pooler) 缺失表的工具。
//
// 背景 (debug session: backend-hang-on-automigrate, 2026-08-13):
//   - 远端 Supabase dev DB 从未完整 provision:133 张期望表缺 55 张
//     (sys_rpa_*、sys_dict_*、sys_oper_log、sys_logininfor、sys_workstation 等)。
//   - 正常启动路径 GORM AutoMigrate 在高延迟链路上对所有存在表做逐列内省,
//     数百次串行 round trip + 随机 TCP stall = 启动"挂死"。
//   - AutoMigrate 注册列表本就不含 RPA/dict/operlog 等 model,跑完也建不出来。
//
// 策略: 对每个 model 先 HasTable(1 round trip),仅对缺失表执行 AutoMigrate
// (CREATE-only 快路径,不对存在表做列内省/ALTER,安全且幂等)。
// 每条操作带 60s context 超时 + 重试,吸收链路随机 stall。
//
// 用法:
//
//	go run ./scripts/dbprovision
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models/system"
	svcrpa "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	svcsystem "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

func readEnv(key string) string {
	data, _ := os.ReadFile(".env")
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.Trim(strings.TrimPrefix(line, key+"="), `"'`)
		}
	}
	return os.Getenv(key)
}

// extraModels 返回不在 db.MigrateModelList() 中、但运行时代码依赖的模型。
// (AutoMigrate 注册列表的历史缺漏 — RPA/dict/operlog 等由 SQL 迁移建表,
// 新部署环境从未执行过那些迁移。)
func extraModels() []interface{} {
	return []interface{}{
		// 系统核心
		&models.DictType{},
		&models.DictData{},
		&models.OperLog{},
		&models.LoginLog{},
		&models.Workstation{},
		// WorkstationDevice 用扁平副本(见文件底部):原 model 带 GORM 关联
		// (Workstation/Asset/ADComputer),AutoMigrate 会级联内省 3 张关联表,
		// 在高延迟链路上 8×30s 重试仍跑不完。
		&workstationDeviceFlat{},
		&models.ADServiceAccount{},
		&models.DeptOUMapping{},
		&models.SysDeptLocationAlias{},
		&models.UserColumnConfig{},
		&models.NoticeIgnore{},
		&models.NotificationChannel{},
		&models.KnowledgeTag{},
		// 监控
		&models.ServerInfo{},
		&models.SystemMetrics{},
		&models.CacheInfo{},
		&models.CacheStats{},
		&models.MACFilterRule{},
		&models.MACOUIVendor{},
		// 文件
		&sysmodels.SysFile{},
		&sysmodels.SysFileAccessLog{},
		// 对账
		&models.SysDataReconciliation{},
		// 运维平面编辑器
		&operations.Door{},
		&operations.Wall{},
		&operations.FloorPlanText{},
		// RPA (新包 internal/models/rpa,服务实际使用)
		&rpamodels.Task{},
		&rpamodels.Worker{},
		&rpamodels.Execution{},
		&rpamodels.Schedule{},
		&rpamodels.Variable{},
		&rpamodels.Template{},
		&rpamodels.AuditLog{},
		&rpamodels.RPACredential{},
		&rpamodels.RPASession{},
		&rpamodels.Notification{},
		// RPA 服务层内嵌模型
		&svcrpa.HumanInterventionEvent{},
		&svcrpa.ScalingEvent{},
		// 用户偏好 (服务层内嵌模型)
		&svcsystem.UserPreference{},
	}
}

func main() {
	host := readEnv("DB_HOST")
	user := readEnv("DB_USER")
	port := readEnv("DB_PORT")
	name := readEnv("DB_NAME")
	pw := readEnv("DB_PASSWORD")
	if host == "" || pw == "" {
		fmt.Fprintln(os.Stderr, "DB_HOST / DB_PASSWORD 缺失 (.env)")
		os.Exit(1)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require&connect_timeout=20",
		user, url.QueryEscape(pw), host, port, name)

	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              false, // pooler 兼容,同 database.go
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect failed:", err)
		os.Exit(1)
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(1)
	fmt.Printf("connected %s:%s/%s\n", host, port, name)

	all := append(db.MigrateModelList(), extraModels()...)
	var created, existed, failed []string

	for _, m := range all {
		// HasTable 解析 model 的 TableName
		exists, err := withRetry(gdb, func(tx *gorm.DB) (bool, error) {
			return tx.Migrator().HasTable(m), nil
		})
		if err != nil {
			failed = append(failed, fmt.Sprintf("%T (HasTable: %v)", m, err))
			continue
		}
		if exists {
			existed = append(existed, fmt.Sprintf("%T", m))
			continue
		}
		_, err = withRetry(gdb, func(tx *gorm.DB) (bool, error) {
			return true, tx.Migrator().AutoMigrate(m)
		})
		if err != nil {
			failed = append(failed, fmt.Sprintf("%T (CREATE: %v)", m, err))
			continue
		}
		created = append(created, fmt.Sprintf("%T", m))
		fmt.Printf("CREATE  %T\n", m)
	}

	fmt.Printf("\n=== summary: %d created, %d existed, %d failed ===\n",
		len(created), len(existed), len(failed))
	for _, f := range failed {
		fmt.Println("FAILED:", f)
	}
	if len(failed) > 0 {
		os.Exit(2)
	}
}

// workstationDeviceFlat 是 models.WorkstationDevice 的无关联扁平副本,
// 列定义保持 1:1(关联对象字段去掉,避免 AutoMigrate 级联内省关联表)。
// 若 models.WorkstationDevice 改列,这里需同步。
type workstationDeviceFlat struct {
	ID        string         `gorm:"type:uuid;primary_key"`
	CreatedAt time.Time      ``
	UpdatedAt time.Time      ``
	DeletedAt gorm.DeletedAt `gorm:"index"`
	CreatedBy string         `gorm:"size:64"`
	UpdatedBy string         `gorm:"size:64"`
	Version   int            ``

	WorkstationID string  `gorm:"type:uuid;not null;index:idx_workstation_device_workstation,priority:1"`
	AssetID       *string `gorm:"type:uuid;index:idx_workstation_device_asset"`
	ADComputerID  *string `gorm:"type:uuid;index:idx_workstation_device_ad_computer"`

	DeviceSource string `gorm:"size:20;not null;index:idx_workstation_device_source_status,priority:1"`

	DeviceSerial *string `gorm:"size:200;index:idx_workstation_device_serial"`
	DeviceName   *string `gorm:"size:255"`
	DeviceModel  *string `gorm:"size:200"`
	DeviceType   *string `gorm:"size:100"`
	MACAddress   *string `gorm:"size:100"`
	IPAddress    *string `gorm:"size:64;index:idx_workstation_device_ip"`

	ResponsibleUser   *string `gorm:"size:100"`
	ResponsibleUserID *string `gorm:"size:64"`

	Status    int  `gorm:"default:0;index:idx_workstation_device_source_status,priority:2"`
	IsPrimary bool `gorm:"default:false"`
	Priority  int  `gorm:"default:0"`

	Description *string `gorm:"type:text"`
}

func (workstationDeviceFlat) TableName() string { return "ops_workstation_device" }

// withRetry: 每条操作 30s 超时,最多 8 次尝试。
// 链路 stall 是随机黑洞(企业网→新加坡 pooler,server 端确认无锁无 active query),
// context 取消后 pgx 会关闭被污染的连接,重试自动走新连接。
func withRetry(gdb *gorm.DB, fn func(tx *gorm.DB) (bool, error)) (bool, error) {
	var lastErr error
	for attempt := 1; attempt <= 8; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		v, err := fn(gdb.WithContext(ctx))
		cancel()
		if err == nil {
			return v, nil
		}
		lastErr = err
		fmt.Printf("  retry %d/8 after error: %v\n", attempt, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return false, lastErr
}
