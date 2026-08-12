package models

// Config 系统配置模型
type Config struct {
	BaseModel
	ConfigName  string         `gorm:"size:100;not null" json:"configName"`
	ConfigKey   string         `gorm:"uniqueIndex;size:100;not null" json:"configKey"`
	ConfigValue string         `gorm:"size:500" json:"configValue"`
	ConfigType  ConfigType     `gorm:"default:'Y';size:1" json:"configType"`
	IsSystem    ConfigIsSystem `gorm:"default:0" json:"isSystem"`
	Remark      string         `gorm:"size:500" json:"remark,omitempty"`
}

// TableName 设置表名
func (Config) TableName() string {
	return "sys_config"
}
