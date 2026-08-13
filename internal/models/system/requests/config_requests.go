package requests

import (
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// ConfigListParams 参数配置列表查询参数
type ConfigListParams struct {
	base.BaseListRequest
	ConfigName *string `json:"configName"`
	ConfigKey  *string `json:"configKey"`
	ConfigType *string `json:"configType"`
	BeginTime  *string `json:"beginTime"`
	EndTime    *string `json:"endTime"`
}

// DefaultConfigListParams 默认列表参数
func DefaultConfigListParams() ConfigListParams {
	return ConfigListParams{
		BaseListRequest: base.BaseListRequest{
			Current:  constants.DefaultCurrent,
			PageSize: constants.DefaultPageSize,
		},
	}
}

// GetPagination 获取分页参数
func (p *ConfigListParams) GetPagination() (current, pageSize int) {
	current = p.Current
	if current < 1 {
		current = constants.DefaultCurrent
	}
	pageSize = p.PageSize
	if pageSize < constants.MinPageSize {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxListPageSize {
		pageSize = constants.MaxListPageSize
	}
	return current, pageSize
}

// GetOffset 计算偏移量
func (p *ConfigListParams) GetOffset() int {
	current, pageSize := p.GetPagination()
	return (current - 1) * pageSize
}

// ConfigCreateRequest 创建参数配置请求
type ConfigCreateRequest struct {
	ConfigName  string            `json:"configName" binding:"required"`
	ConfigKey   string            `json:"configKey" binding:"required"`
	ConfigValue string            `json:"configValue" binding:"required"`
	ConfigType  models.ConfigType `json:"configType"`
	IsSystem    int               `json:"isSystem"`
	Remark      *string           `json:"remark"`
}

// ConfigUpdateRequest 更新参数配置请求
type ConfigUpdateRequest struct {
	ID          string            `json:"id"` // ID 从 URL 参数获取，不在请求体验证
	ConfigName  string            `json:"configName" binding:"required"`
	// F-17: 可选,客户端可显式声明 ConfigKey 触发服务端校验
	// (系统内置参数的 ConfigKey 不可修改;非内置参数本字段被忽略)
	ConfigKey   string            `json:"configKey,omitempty"`
	ConfigValue string            `json:"configValue" binding:"required"`
	ConfigType  models.ConfigType `json:"configType"`
	Remark      *string           `json:"remark"`
}
