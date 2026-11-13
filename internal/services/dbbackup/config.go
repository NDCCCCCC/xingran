package dbbackup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// 本文件负责备份计划的持久化（sys_config）与定时门控。
//
// 热重载方案（PLAN.md「固定 tick + 每 tick 读配置门控」）：调度层以固定频率
// 调用 RunScheduled，本函数每 tick 重新读 sys_config 的 enabled/cron，用
// 「上次实际执行时间 + robfig/cron 解析出的下一次触发时间」判断是否到点；
// 未到点直接返回 nil。改配置无需重启。
//
// 注意：默认 cron 是 6 字段含秒（CONTRACT.md 第 2 节 `0 0 2 * * *`），
// 与 internal/scheduler 的 cron.WithSeconds() 口径一致；兼容 5 字段与
// @every/@daily 等 descriptor（用 cron.ParseStandard + Descriptor 选项）。

// sys_config 键（CONTRACT.md 第 2 节，13 个键）
const (
	cfgKeyEnabled      = "sys.database.backup.enabled"
	cfgKeyCron         = "sys.database.backup.cron"
	cfgKeyRetention    = "sys.database.backup.retention"
	cfgKeyStorage      = "sys.database.backup.storage"
	cfgKeyOSSEndpoint  = "sys.database.backup.oss.endpoint"
	cfgKeyOSSBucket    = "sys.database.backup.oss.bucket"
	cfgKeyOSSAccessKey = "sys.database.backup.oss.accessKey"
	cfgKeyOSSSecretKey = "sys.database.backup.oss.secretKey"
	cfgKeyOSSUseSSL    = "sys.database.backup.oss.useSSL"
	cfgKeyFTPHost      = "sys.database.backup.ftp.host"
	cfgKeyFTPPort      = "sys.database.backup.ftp.port"
	cfgKeyFTPUser      = "sys.database.backup.ftp.user"
	cfgKeyFTPPassword  = "sys.database.backup.ftp.password"
	cfgKeyFTPRemoteDir = "sys.database.backup.ftp.remoteDir"
	cfgKeySMBPath      = "sys.database.backup.smb.path"
	// pg_dump / pg_restore 可执行文件绝对路径（部署机未把 PostgreSQL 客户端
	// 加入 PATH 时，由管理员在参数管理页显式配置；为空则回退 PATH 查找）。
	cfgKeyPgDumpPath    = "sys.database.backup.pg_dump_path"
	cfgKeyPgRestorePath = "sys.database.backup.pg_restore_path"
	// cfgKeyLastRunAt 记录上次定时导出实际触发时间（跨重启持久化，防重启后
	// 每分钟 tick 都把「从未执行」当成到点而补跑）。
	cfgKeyLastRunAt = "sys.database.backup.last_run_at"
)

// 默认值（与 CONTRACT.md 第 2 节一致）
const (
	defaultCronExpr  = "0 0 2 * * *"
	defaultRetention = 7
	defaultStorage   = StorageLocal
	defaultFTPPort   = 21
	defaultFTPDir    = "/"
	// lastRunAtLayout 持久化格式（UTC，秒精度足够 tick 粒度）
	lastRunAtLayout = time.RFC3339
)

// readConfigValue 读取单个 sys_config 值；键不存在时返回 ("", nil)，
// 由调用方套用默认值（迁移 240 seed 未跑/新库也安全）。
func readConfigValue(ctx context.Context, db *gorm.DB, key string) (string, error) {
	var c models.Config
	err := db.WithContext(ctx).Where("config_key = ?", key).First(&c).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", fmt.Errorf("读取 sys_config[%s] 失败: %w", key, err)
	}
	return c.ConfigValue, nil
}

// writeConfigValue 以 Upsert 写入单个 sys_config 键。
// 只更新 config_value：不覆盖管理员在参数管理页维护的 ConfigType/IsSystem/Remark，
// 且新写入的键补齐 ConfigTypeNo + IsSystemNo（CONTRACT 要求可在参数管理页编辑）。
func writeConfigValue(ctx context.Context, db *gorm.DB, key, value string) error {
	c := models.Config{
		ConfigKey:   key,
		ConfigValue: value,
		ConfigType:  models.ConfigTypeNo,
		IsSystem:    models.ConfigIsSystemNo,
		Remark:      "数据库全库备份计划（由服务层维护）",
	}
	if err := db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "config_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_value", "updated_at"}),
	}).Create(&c).Error; err != nil {
		return fmt.Errorf("写入 sys_config[%s] 失败: %w", key, err)
	}
	return nil
}

// parseBool 解析 sys_config 的布尔值（容错 "1"/"true"/"on"）
func parseBool(s string, def bool) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return def
	}
	switch s {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

// parseIntDefault 解析整数，失败或 <=0 时回退默认值
func parseIntDefault(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// normalizeStorageType 归一存储类型，非法值回退 local
func normalizeStorageType(s string) StorageType {
	switch StorageType(strings.ToLower(strings.TrimSpace(s))) {
	case StorageOSS:
		return StorageOSS
	case StorageFTP:
		return StorageFTP
	case StorageSMB:
		return StorageSMB
	default:
		return defaultStorage
	}
}

// loadPlan 从 sys_config 读取完整备份计划（每 tick / 每次 GetPlan 都重读，
// 保证改配置无需重启）。
//
// 凭据字段（OSS accessKey/secretKey、FTP password）在 sys_config 中以 SM4 密文
// 形式存放，这里逐字段解密后返回真实值供存储适配器使用；返回值只存在于内存中的
// Plan，绝不写日志（日志/审计脱敏由 API 层与 operlog 承担）。
// 解密失败不静默吞掉：明确报错，避免把密文串当凭据交给远端导致难排查的失败。
// 配置页展示场景请用 loadPlanTolerant（降级不整体失败）。
func loadPlan(ctx context.Context, db *gorm.DB) (*Plan, error) {
	return loadPlanOpts(ctx, db, true)
}

// loadPlanTolerant GetPlan（备份配置页展示）专用降级读取：凭据解密失败（如
// SM4_KEY 被更换、装配未完成）时记告警、相应字段置空串并返回计划主体，页面
// 仍可查看/编辑非敏感配置；空串在出参上表现为「未配置」，绝不回吐密文。
// 定时任务与 savePlan 回填必须用严格 loadPlan，不得换用本函数。
func loadPlanTolerant(ctx context.Context, db *gorm.DB) (*Plan, error) {
	return loadPlanOpts(ctx, db, false)
}

func loadPlanOpts(ctx context.Context, db *gorm.DB, strict bool) (*Plan, error) {
	keys := []string{
		cfgKeyEnabled, cfgKeyCron, cfgKeyRetention, cfgKeyStorage,
		cfgKeyOSSEndpoint, cfgKeyOSSBucket, cfgKeyOSSAccessKey, cfgKeyOSSSecretKey, cfgKeyOSSUseSSL,
		cfgKeyFTPHost, cfgKeyFTPPort, cfgKeyFTPUser, cfgKeyFTPPassword, cfgKeyFTPRemoteDir,
		cfgKeySMBPath,
	}
	vals := make(map[string]string, len(keys))
	for _, k := range keys {
		v, err := readConfigValue(ctx, db, k)
		if err != nil {
			return nil, err
		}
		vals[k] = v
	}

	// 三个凭据字段：逐字段解密。字段集合与 savePlan 的加密字段必须一致，
	// 否则会出现「写进去是密文、读出来还是密文」的错配。
	// 解密失败时字段保持空串（绝不把密文串当真值交给调用方），错误收集后由
	// 严格/降级两种模式分别决定整体失败还是置空返回。
	var credErrs []error
	accessKey, err := decryptCredential(vals[cfgKeyOSSAccessKey])
	if err != nil {
		credErrs = append(credErrs, fmt.Errorf("读取备份计划 OSS accessKey: %w", err))
	}
	secretKey, err := decryptCredential(vals[cfgKeyOSSSecretKey])
	if err != nil {
		credErrs = append(credErrs, fmt.Errorf("读取备份计划 OSS secretKey: %w", err))
	}
	ftpPassword, err := decryptCredential(vals[cfgKeyFTPPassword])
	if err != nil {
		credErrs = append(credErrs, fmt.Errorf("读取备份计划 FTP password: %w", err))
	}

	p := &Plan{
		Enabled:   parseBool(vals[cfgKeyEnabled], false),
		Cron:      strings.TrimSpace(vals[cfgKeyCron]),
		Retention: parseIntDefault(vals[cfgKeyRetention], defaultRetention),
		Storage:   normalizeStorageType(vals[cfgKeyStorage]),
		OSS: OSSConfig{
			Endpoint:  vals[cfgKeyOSSEndpoint],
			Bucket:    vals[cfgKeyOSSBucket],
			AccessKey: accessKey,
			SecretKey: secretKey,
			UseSSL:    parseBool(vals[cfgKeyOSSUseSSL], true),
		},
		FTP: FTPConfig{
			Host:      vals[cfgKeyFTPHost],
			Port:      parseIntDefault(vals[cfgKeyFTPPort], defaultFTPPort),
			User:      vals[cfgKeyFTPUser],
			Password:  ftpPassword,
			RemoteDir: vals[cfgKeyFTPRemoteDir],
		},
		SMB: SMBConfig{Path: vals[cfgKeySMBPath]},
	}
	if p.Cron == "" {
		p.Cron = defaultCronExpr
	}
	if strings.TrimSpace(p.FTP.RemoteDir) == "" {
		p.FTP.RemoteDir = defaultFTPDir
	}

	// 严格模式：任一凭据解密失败则整体失败（定时任务 / savePlan 回填路径，
	// 避免拿半份计划执行远端上传或用空值覆盖真实凭据）。
	if strict && len(credErrs) > 0 {
		return nil, errors.Join(credErrs...)
	}
	// 降级模式：记告警后返回计划主体（解密失败的字段已为空串）。
	for _, credErr := range credErrs {
		applogger.Warnf("[全库备份] %v（展示降级：凭据字段置空，配置页仍可用）", credErr)
	}
	return p, nil
}

// savePlan 把完整计划写回 sys_config。operator 仅用于审计留痕（本层不写 operlog）。
//
// 敏感凭据（OSS accessKey/secretKey、FTP password）改为 **SM4 字段加密后落库**：
// 绝不把明文或脱敏占位符写进 sys_config；不在 manifest / operlog 出现，
// 由 handler 出参脱敏为 ******。
//
// 占位符兼容（双保险，API 层 handler.mergePlanMaskedSecrets 已做一次）：
// 若调用方直接传了 "******" 或空串（用户未改凭据），这里再次读取当前计划中的真实
// 凭据回填，确保「用户未改凭据时不能把占位符写成真值」，也绝不误清空已有凭据。
func savePlan(ctx context.Context, db *gorm.DB, p *Plan, operator string) error {
	if p == nil {
		return fmt.Errorf("备份计划不能为 nil")
	}
	normalized := *p
	if normalized.Cron == "" {
		normalized.Cron = defaultCronExpr
	}
	if _, err := parseCronExpr(normalized.Cron); err != nil {
		return fmt.Errorf("备份计划 cron 表达式非法: %w", err)
	}
	if normalized.Retention <= 0 {
		normalized.Retention = defaultRetention
	}
	normalized.Storage = normalizeStorageType(string(normalized.Storage))
	if normalized.FTP.Port <= 0 {
		normalized.FTP.Port = defaultFTPPort
	}
	if strings.TrimSpace(normalized.FTP.RemoteDir) == "" {
		normalized.FTP.RemoteDir = defaultFTPDir
	}

	// 显式清空（__CLEAR__ 标记 → 空串落库 = 删除凭据）。清空后的空串在形态上
	// 与「未改动的空串」无法区分，故用 cleared 集合记录清空意图，回填时跳过
	// 这些字段，避免旧凭据被回填逻辑复活。
	cleared := map[string]bool{}
	if isCredentialClear(normalized.OSS.AccessKey) {
		normalized.OSS.AccessKey = ""
		cleared["ossAK"] = true
	}
	if isCredentialClear(normalized.OSS.SecretKey) {
		normalized.OSS.SecretKey = ""
		cleared["ossSK"] = true
	}
	if isCredentialClear(normalized.FTP.Password) {
		normalized.FTP.Password = ""
		cleared["ftpPwd"] = true
	}

	// 占位符/空串回填：只要有一个敏感字段表示「未改动」，就读取当前计划取真值。
	// 注意读的是「解密后的真实凭据」，回填后再统一加密写回，不会出现二次加密。
	if isCredentialUnchanged(normalized.OSS.AccessKey) ||
		isCredentialUnchanged(normalized.OSS.SecretKey) ||
		isCredentialUnchanged(normalized.FTP.Password) {
		current, err := loadPlan(ctx, db)
		if err != nil {
			return fmt.Errorf("读取当前备份计划以回填未改动的凭据失败: %w", err)
		}
		if isCredentialUnchanged(normalized.OSS.AccessKey) && !cleared["ossAK"] {
			normalized.OSS.AccessKey = current.OSS.AccessKey
		}
		if isCredentialUnchanged(normalized.OSS.SecretKey) && !cleared["ossSK"] {
			normalized.OSS.SecretKey = current.OSS.SecretKey
		}
		if isCredentialUnchanged(normalized.FTP.Password) && !cleared["ftpPwd"] {
			normalized.FTP.Password = current.FTP.Password
		}
	}

	// 逐字段加密：写路径 fail-closed，cipher 未注册且有真实凭据时拒绝落库，
	// 不把明文回退写进 sys_config。空串保持空串（未配置语义）。
	encAccessKey, err := encryptCredential(normalized.OSS.AccessKey)
	if err != nil {
		return err
	}
	encSecretKey, err := encryptCredential(normalized.OSS.SecretKey)
	if err != nil {
		return err
	}
	encFTPPassword, err := encryptCredential(normalized.FTP.Password)
	if err != nil {
		return err
	}

	pairs := map[string]string{
		cfgKeyEnabled:      strconv.FormatBool(normalized.Enabled),
		cfgKeyCron:         normalized.Cron,
		cfgKeyRetention:    strconv.Itoa(normalized.Retention),
		cfgKeyStorage:      string(normalized.Storage),
		cfgKeyOSSEndpoint:  normalized.OSS.Endpoint,
		cfgKeyOSSBucket:    normalized.OSS.Bucket,
		cfgKeyOSSAccessKey: encAccessKey,
		cfgKeyOSSSecretKey: encSecretKey,
		cfgKeyOSSUseSSL:    strconv.FormatBool(normalized.OSS.UseSSL),
		cfgKeyFTPHost:      normalized.FTP.Host,
		cfgKeyFTPPort:      strconv.Itoa(normalized.FTP.Port),
		cfgKeyFTPUser:      normalized.FTP.User,
		cfgKeyFTPPassword:  encFTPPassword,
		cfgKeyFTPRemoteDir: normalized.FTP.RemoteDir,
		cfgKeySMBPath:      normalized.SMB.Path,
	}
	// 按固定顺序写，避免 map 遍历顺序差异影响审计
	order := []string{
		cfgKeyEnabled, cfgKeyCron, cfgKeyRetention, cfgKeyStorage,
		cfgKeyOSSEndpoint, cfgKeyOSSBucket, cfgKeyOSSAccessKey, cfgKeyOSSSecretKey, cfgKeyOSSUseSSL,
		cfgKeyFTPHost, cfgKeyFTPPort, cfgKeyFTPUser, cfgKeyFTPPassword, cfgKeyFTPRemoteDir,
		cfgKeySMBPath,
	}
	for _, k := range order {
		if err := writeConfigValue(ctx, db, k, pairs[k]); err != nil {
			return err
		}
	}
	_ = operator // 审计由 handler 层 operlog.RecordWithBody 承担
	return nil
}

// parseCronExpr 解析 cron 表达式。优先 6 字段含秒（调度层口径），
// 兼容 5 字段标准 cron 与 @every/@daily descriptor。
func parseCronExpr(expr string) (cron.Schedule, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, fmt.Errorf("cron 表达式为空")
	}
	// 6 字段含秒：带 Descriptor 选项同时兼容 @every / @daily 写法
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if sched, err := parser.Parse(expr); err == nil {
		return sched, nil
	}
	// 退化为 5 字段标准 cron（无秒）
	standard := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	sched, err := standard.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("解析 cron %q 失败: %w", expr, err)
	}
	return sched, nil
}

// shouldRunScheduled 门控判定：返回本 tick 是否应真正执行一次导出。
//
// 算法（lead 澄清 + Phase 58 方案 A）：以「上次实际执行时间 lastRun」为基准，
// 用 cron 表达式算出下一次触发时间 next；仅当 now >= next 时才到点。
// lastRun 来自 sys_config[sys.database.backup.last_run_at]（跨重启持久化），
// 查不到时回退查 sys_db_backup 最近一条 source=cron 的 created_at；两者都没有
// 才把基准设为 now-1s（刚启用时立刻跑一次，但绝不补跑历史错过的触发）。
//
// 周期精度有 ≤1 分钟抖动：调度层固定 tick `0 * * * * *`（每分钟一次），
// tick 粒度决定了实际触发相对 cron 标称时间的最大偏差。这是 Phase 58
// 方案 A「固定最小 tick + 每 tick 读配置门控」的既有取舍，用户已接受。
//
// 同一进程内由 service.taskMu 串行化调用，故本函数不再自带 mutex；
// 跨实例互斥由「备份任务表活跃状态行 + sys_config last_run_at」共同约束。
func shouldRunScheduled(ctx context.Context, db *gorm.DB, now time.Time, expr string) (bool, error) {
	sched, err := parseCronExpr(expr)
	if err != nil {
		return false, err
	}
	lastRun, err := loadLastRunAt(ctx, db)
	if err != nil {
		return false, err
	}
	base := lastRun
	if base.IsZero() {
		base = now.Add(-time.Second)
	}
	next := sched.Next(base)
	if now.Before(next) {
		return false, nil
	}
	return true, nil
}

// loadLastRunAt 读取上次定时导出的实际触发时间。
// 优先 sys_config 专用键；缺失时回退查 sys_db_backup 最近一条 source=cron 记录。
func loadLastRunAt(ctx context.Context, db *gorm.DB) (time.Time, error) {
	raw, err := readConfigValue(ctx, db, cfgKeyLastRunAt)
	if err != nil {
		return time.Time{}, err
	}
	if strings.TrimSpace(raw) != "" {
		if t, err := time.Parse(lastRunAtLayout, strings.TrimSpace(raw)); err == nil {
			return t, nil
		}
		// 值损坏（被人工改坏）不阻断门控，继续走备份记录回退
	}
	var last models.DBBackup
	err = db.WithContext(ctx).
		Where("source = ?", SourceCron).
		Order("created_at DESC").
		First(&last).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("查询最近一次定时备份记录失败: %w", err)
	}
	return last.CreatedAt, nil
}

// persistLastRunAt 持久化本次定时导出触发时间，供下一轮门控与重启后复用。
// 写入失败不阻断已触发的导出（仅记日志由调用方处理），因为真正的事实源
// 还有 sys_db_backup 的 source=cron 记录兜底。
func persistLastRunAt(ctx context.Context, db *gorm.DB, t time.Time) error {
	return writeConfigValue(ctx, db, cfgKeyLastRunAt, t.UTC().Format(lastRunAtLayout))
}

// resolvePGToolPath 解析 PostgreSQL 客户端工具的可执行路径。
//
// 优先级（高→低）：
//  1. sys_config 配置键（参数管理页可改，运行时热生效，无需重启）；
//  2. DatabaseConfig.PgDumpPath / PgRestorePath（来自 config.yaml，部署期固定）；
//  3. 默认命令名，由 exec.LookPath 在 PATH 中查找。
//
// 返回 (解析后的路径, 是否可用, 供页面展示的提示)。缺工具时「是否可用=false」，
// 但错误本身留给 dumper/restorer 包装成可执行的诊断信息，不在此处吞掉。
func resolvePGToolPath(ctx context.Context, db *gorm.DB, cfg DatabaseConfig, which string) (string, bool, string) {
	var configKey, cfgPath, defaultName string
	switch which {
	case "pg_dump":
		configKey, cfgPath, defaultName = cfgKeyPgDumpPath, cfg.PgDumpPath, "pg_dump"
	case "pg_restore":
		configKey, cfgPath, defaultName = cfgKeyPgRestorePath, cfg.PgRestorePath, "pg_restore"
	default:
		return "", false, "未知工具类型"
	}

	// 1) sys_config（DB 不可用时静默跳过，回落 config.yaml / PATH）
	if db != nil {
		if v, err := readConfigValue(ctx, db, configKey); err == nil && strings.TrimSpace(v) != "" {
			p := strings.TrimSpace(v)
			if _, statErr := os.Stat(p); statErr == nil {
				return p, true, ""
			}
			return p, false, fmt.Sprintf("参数管理页配置的 %s 路径不存在或不可访问: %s", configKey, p)
		}
	}
	// 2) config.yaml 注入的绝对路径
	if strings.TrimSpace(cfgPath) != "" {
		p := strings.TrimSpace(cfgPath)
		if _, statErr := os.Stat(p); statErr == nil {
			return p, true, ""
		}
		return p, false, fmt.Sprintf("config.yaml 配置的 %s 路径不存在或不可访问: %s", which, p)
	}
	// 3) PATH 查找
	if p, err := exec.LookPath(defaultName); err == nil {
		return p, true, ""
	}
	return defaultName, false, fmt.Sprintf(
		"部署机未找到 %s：请安装与服务端版本匹配的 PostgreSQL 客户端（如 apt install postgresql-client / yum install postgresql），"+
			"或把 %s 加入 PATH，或在参数管理页配置 %s 指向可执行文件绝对路径（如 /usr/lib/postgresql/16/bin/%s）。修改后无需重启。",
		which, which, configKey, which)
}
