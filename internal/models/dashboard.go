package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DashboardStatus int

const (
	DashboardStatusNormal  DashboardStatus = 0
	DashboardStatusStopped DashboardStatus = 1
)

type WidgetPosition struct {
	X    int `json:"x"`
	Y    int `json:"y"`
	W    int `json:"w"`
	H    int `json:"h"`
	MinW int `json:"minW,omitempty"`
	MinH int `json:"minH,omitempty"`
	MaxW int `json:"maxW,omitempty"`
	MaxH int `json:"maxH,omitempty"`
}

type DataSourceType string

const (
	DataSourceTypeAPI       DataSourceType = "api"
	DataSourceTypeWebSocket DataSourceType = "websocket"
	DataSourceTypeStatic    DataSourceType = "static"
)

// DataTransformConfig 数据转换配置
type DataTransformConfig struct {
	Expression string         `json:"expression,omitempty"`
	Aggregate  *string        `json:"aggregate,omitempty"`
	GroupBy    string         `json:"groupBy,omitempty"`
	OrderBy    *OrderByConfig `json:"orderBy,omitempty"`
	Limit      int            `json:"limit,omitempty"`
}

// OrderByConfig 排序配置
type OrderByConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// ApiDataSourceConfig API 数据源配置
type ApiDataSourceConfig struct {
	Type      DataSourceType         `json:"type"`
	Endpoint  string                 `json:"endpoint"`
	Method    string                 `json:"method"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Body      map[string]interface{} `json:"body,omitempty"`
	Transform *DataTransformConfig   `json:"transform,omitempty"`
}

// WebSocketDataSourceConfig WebSocket 数据源配置
type WebSocketDataSourceConfig struct {
	Type      DataSourceType       `json:"type"`
	Channel   string               `json:"channel"`
	Transform *DataTransformConfig `json:"transform,omitempty"`
}

// StaticDataSourceConfig 静态数据源配置
type StaticDataSourceConfig struct {
	Type DataSourceType `json:"type"`
	Data interface{}    `json:"data"`
}

// DataSourceConfig 数据源配置（存储为 JSON）
type DataSourceConfig struct {
	Api       *ApiDataSourceConfig       `json:"api,omitempty"`
	WebSocket *WebSocketDataSourceConfig `json:"websocket,omitempty"`
	Static    *StaticDataSourceConfig    `json:"static,omitempty"`
}

// Scan 实现 sql.Scanner 接口
func (d *DataSourceConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal DataSourceConfig value")
	}
	return json.Unmarshal(bytes, d)
}

// Value 实现 driver.Valuer 接口
func (d DataSourceConfig) Value() (driver.Value, error) {
	return json.Marshal(d)
}

type BaseDisplayConfig struct {
	Color           string            `json:"color,omitempty"`
	BackgroundColor string            `json:"backgroundColor,omitempty"`
	ShowBorder      bool              `json:"showBorder,omitempty"`
	Transparent     bool              `json:"transparent,omitempty"`
	ClassName       string            `json:"className,omitempty"`
	Style           map[string]string `json:"style,omitempty"`
}

// StatCardDisplayConfig 统计卡片显示配置
type StatCardDisplayConfig struct {
	BaseDisplayConfig `json:",inline"`
	Type              string `json:"type"`
	Prefix            string `json:"prefix,omitempty"`
	Suffix            string `json:"suffix,omitempty"`
	Decimals          int    `json:"decimals,omitempty"`
	Percentage        bool   `json:"percentage,omitempty"`
	ShowTrend         bool   `json:"showTrend,omitempty"`
	Icon              string `json:"icon,omitempty"`
	IconColor         string `json:"iconColor,omitempty"`
}

// ChartDisplayConfig 图表显示配置
type ChartDisplayConfig struct {
	BaseDisplayConfig `json:",inline"`
	Type              string   `json:"type"`
	ChartType         string   `json:"chartType"`
	XField            string   `json:"xField,omitempty"`
	YField            string   `json:"yField,omitempty"`
	SeriesField       string   `json:"seriesField,omitempty"`
	Colors            []string `json:"colors,omitempty"`
	ShowLegend        bool     `json:"showLegend,omitempty"`
	ShowLabels        bool     `json:"showLabels,omitempty"`
	Smooth            bool     `json:"smooth,omitempty"`
	ShowArea          bool     `json:"showArea,omitempty"`
	Title             string   `json:"title,omitempty"`
}

// TableColumnConfig 表格列配置
type TableColumnConfig struct {
	DataIndex string `json:"dataIndex"`
	Title     string `json:"title"`
	Width     int    `json:"width,omitempty"`
	Align     string `json:"align,omitempty"`
	Render    string `json:"render,omitempty"`
	Sortable  bool   `json:"sortable,omitempty"`
}

// TableDisplayConfig 表格显示配置
type TableDisplayConfig struct {
	BaseDisplayConfig `json:",inline"`
	Type              string              `json:"type"`
	Columns           []TableColumnConfig `json:"columns"`
	Bordered          bool                `json:"bordered,omitempty"`
	Size              string              `json:"size,omitempty"`
	Pagination        *PaginationConfig   `json:"pagination,omitempty"`
	RowHeight         int                 `json:"rowHeight,omitempty"`
}

// PaginationConfig 分页配置
type PaginationConfig struct {
	Enabled  bool `json:"enabled"`
	PageSize int  `json:"pageSize"`
}

// ListDisplayConfig 列表显示配置
type ListDisplayConfig struct {
	BaseDisplayConfig `json:",inline"`
	Type              string `json:"type"`
	TitleField        string `json:"titleField"`
	DescriptionField  string `json:"descriptionField,omitempty"`
	TimeField         string `json:"timeField,omitempty"`
	IconField         string `json:"iconField,omitempty"`
	MaxItems          int    `json:"maxItems,omitempty"`
	ShowIndex         bool   `json:"showIndex,omitempty"`
}

// ProgressDisplayConfig 进度条显示配置
type ProgressDisplayConfig struct {
	BaseDisplayConfig `json:",inline"`
	Type              string           `json:"type"`
	ProgressType      string           `json:"progressType"`
	Target            int              `json:"target,omitempty"`
	ColorThresholds   []ColorThreshold `json:"colorThresholds,omitempty"`
}

// ColorThreshold 颜色阈值
type ColorThreshold struct {
	Value int    `json:"value"`
	Color string `json:"color"`
}

// DisplayConfig 显示配置（存储为 JSON）
type DisplayConfig struct {
	StatCard *StatCardDisplayConfig `json:"statCard,omitempty"`
	Chart    *ChartDisplayConfig    `json:"chart,omitempty"`
	Table    *TableDisplayConfig    `json:"table,omitempty"`
	List     *ListDisplayConfig     `json:"list,omitempty"`
	Progress *ProgressDisplayConfig `json:"progress,omitempty"`
}

// Scan 实现 sql.Scanner 接口
func (d *DisplayConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal DisplayConfig value")
	}
	return json.Unmarshal(bytes, d)
}

// Value 实现 driver.Valuer 接口
func (d DisplayConfig) Value() (driver.Value, error) {
	return json.Marshal(d)
}

type WidgetConfig struct {
	ID         string           `json:"id" gorm:"primaryKey;size:64"`
	Type       string           `json:"type" gorm:"size:20;not null"`
	Title      string           `json:"title" gorm:"size:200;not null"`
	Position   WidgetPosition   `json:"position" gorm:"type:jsonb;not null"`
	DataSource DataSourceConfig `json:"dataSource" gorm:"type:jsonb;not null"`
	Display    DisplayConfig    `json:"display" gorm:"type:jsonb;not null"`

	// 数据权限过滤
	DataFilter *WidgetDataFilter `json:"dataFilter,omitempty" gorm:"type:jsonb"`

	RefreshInterval int   `json:"refreshInterval,omitempty" gorm:"default:0"`
	Enabled         bool  `json:"enabled,omitempty" gorm:"default:true"`
	CreatedAt       int64 `json:"createdAt,omitempty" gorm:"autoCreateTime:milli"`
	UpdatedAt       int64 `json:"updatedAt,omitempty" gorm:"autoUpdateTime:milli"`
}

type Columns struct {
	Desktop int `json:"desktop"`
	Tablet  int `json:"tablet"`
	Mobile  int `json:"mobile"`
}

// LayoutConfig 布局配置
type LayoutConfig struct {
	Widgets   []WidgetConfig `json:"widgets" gorm:"type:jsonb;not null"`
	Columns   Columns        `json:"columns" gorm:"type:jsonb;not null"`
	RowHeight int            `json:"rowHeight" gorm:"not null;default:60"`
	Margin    []int          `json:"margin" gorm:"type:jsonb;not null"` // [horizontal, vertical]
	Draggable bool           `json:"draggable" gorm:"default:true"`
	Resizable bool           `json:"resizable" gorm:"default:true"`
}

// Scan 实现 sql.Scanner 接口
func (l *LayoutConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal LayoutConfig value")
	}
	return json.Unmarshal(bytes, l)
}

// Value 实现 driver.Valuer 接口
func (l LayoutConfig) Value() (driver.Value, error) {
	return json.Marshal(l)
}

type DashboardScope string

const (
	DashboardScopePrivate DashboardScope = "private" // 私有：仅创建者可见
	DashboardScopeDept    DashboardScope = "dept"    // 部门：本部门可见
	DashboardScopeGlobal  DashboardScope = "global"  // 全局：全员可见
)

type WidgetDataFilter struct {
	Enabled      bool              `json:"enabled"`
	FilterType   string            `json:"filterType"`
	FilterConfig map[string]string `json:"filterConfig"`
}

type TemplateScope string

const (
	TemplateScopeGlobal   TemplateScope = "global"
	TemplateScopeDept     TemplateScope = "dept"
	TemplateScopePersonal TemplateScope = "personal"
)

// Dashboard 仪表盘模型
type Dashboard struct {
	BaseModel
	Name        string `json:"name" gorm:"size:100;not null;index"`
	Description string `json:"description" gorm:"size:500"`
	OwnerID     string `json:"ownerId" gorm:"size:64;index"`
	OwnerDeptID string `json:"ownerDeptId" gorm:"size:64;index"` // 创建者部门
	IsDefault   bool   `json:"isDefault" gorm:"default:false;index"`
	IsTemplate  bool   `json:"isTemplate" gorm:"default:false;index"`
	IsSystem    bool   `json:"isSystem" gorm:"default:false;index"` // 系统仪表盘标识

	// 权限相关
	Scope  DashboardScope `json:"scope" gorm:"size:20;default:'private'"` // 可见范围
	DeptID *string        `json:"deptId" gorm:"size:64;index"`            // 关联部门（scope=dept时使用）

	// 模板相关
	TemplateScope TemplateScope `json:"templateScope" gorm:"size:20"`

	// 布局相关
	Layout          LayoutConfig    `json:"layout" gorm:"type:jsonb;not null"`
	RefreshInterval int             `json:"refreshInterval" gorm:"default:60"`
	Status          DashboardStatus `json:"status" gorm:"default:0;index"`
}

func (Dashboard) TableName() string {
	return "sys_dashboards"
}

func (d *Dashboard) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

type DashboardVersion struct {
	ID          string       `json:"id" gorm:"primaryKey;type:uuid"`
	DashboardID string       `json:"dashboardId" gorm:"type:uuid;not null;index"`
	Layout      LayoutConfig `json:"layout" gorm:"type:jsonb;not null"`
	Comment     string       `json:"comment" gorm:"size:500"`
	CreatedAt   time.Time    `json:"createdAt" gorm:"autoCreateTime"`
	CreatedBy   string       `json:"createdBy" gorm:"size:64"`
}

func (DashboardVersion) TableName() string {
	return "sys_dashboard_versions"
}

func (dv *DashboardVersion) BeforeCreate(tx *gorm.DB) error {
	if dv.ID == "" {
		dv.ID = uuid.New().String()
	}
	return nil
}
