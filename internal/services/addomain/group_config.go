package addomain

const (
	// ConfigGroupSyncEnabled 组同步开关
	ConfigGroupSyncEnabled = "sys.ad.group.sync.enabled"
	// ConfigGroupSyncCron 组同步Cron表达式
	ConfigGroupSyncCron = "sys.ad.group.sync.cron"
	// ConfigGroupMemberOU MemberOU路径
	ConfigGroupMemberOU = "sys.ad.group.member_ou"
	// ConfigGroupAutoCreate 自动创建组
	ConfigGroupAutoCreate = "sys.ad.group.auto_create"
	// ConfigGroupMaxConcurrent 最大并发数
	ConfigGroupMaxConcurrent = "sys.ad.group.max_concurrent"
	// ConfigGroupSyncBatchSize 批量同步大小
	ConfigGroupSyncBatchSize = "sys.ad.group.sync.batch_size"
)

// GroupSyncConfig 组同步配置
type GroupSyncConfig struct {
	Enabled          bool   `json:"enabled"`
	Cron             string `json:"cron"`
	MemberOU         string `json:"member_ou"`
	AutoCreateGroups bool   `json:"auto_create_groups"`
	MaxConcurrent    int    `json:"max_concurrent"`
	SyncBatchSize    int    `json:"sync_batch_size"`
}

// GetDefaultGroupSyncConfig 获取默认配置
func GetDefaultGroupSyncConfig() *GroupSyncConfig {
	return &GroupSyncConfig{
		Enabled:          false,
		Cron:             "0 */15 * * * *",
		MemberOU:         "",
		AutoCreateGroups: true,
		MaxConcurrent:    5,
		SyncBatchSize:    100,
	}
}