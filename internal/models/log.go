package models

import "time"

// OperLog 操作日志模型
type OperLog struct {
	BaseTimeLine
	Title         string     `gorm:"size:50" json:"title,omitempty"`
	BusinessType  int        `json:"businessType"`
	Method        string     `gorm:"size:100" json:"method,omitempty"`
	RequestMethod string     `gorm:"size:10" json:"requestMethod,omitempty"`
	OperatorType  int        `json:"operatorType"`
	OperatorName  *string    `gorm:"size:50" json:"operatorName,omitempty"`
	DeptName      *string    `gorm:"size:50" json:"deptName,omitempty"`
	OperUrl       *string    `gorm:"size:255" json:"operUrl,omitempty"`
	OperIP        *string    `gorm:"size:128" json:"operIp,omitempty"`
	OperLocation  *string    `gorm:"size:255" json:"operLocation,omitempty"`
	OperParam     *string    `gorm:"type:text" json:"operParam,omitempty"`
	JsonResult    *string    `gorm:"type:text" json:"jsonResult,omitempty"`
	Status        int        `json:"status"`
	ErrorMsg      *string    `gorm:"size:2000" json:"errorMsg,omitempty"`
	OperTime      time.Time  `json:"operTime"`
	CostTime      int64      `json:"costTime"`
}

// LoginLog 登录日志模型
type LoginLog struct {
	BaseTimeLine
	Username      string     `gorm:"size:50" json:"username,omitempty"`
	IPAddr        string     `gorm:"size:128" json:"ipaddr,omitempty"`
	LoginLocation *string    `gorm:"size:255" json:"loginLocation,omitempty"`
	Browser       *string    `gorm:"size:50" json:"browser,omitempty"`
	OS            *string    `gorm:"size:50" json:"os,omitempty"`
	Status        int        `json:"status"`
	Msg           *string    `gorm:"size:255" json:"msg,omitempty"`
	LoginTime     time.Time  `json:"loginTime"`
}

// Job 定时任务模型
type Job struct {
	BaseModel
	JobName          string `gorm:"size:64;not null" json:"jobName"`
	JobGroup         string `gorm:"size:64;not null" json:"jobGroup"`
	InvokeTarget     string `gorm:"size:500;not null" json:"invokeTarget"`
	CronExpression   *string `gorm:"size:255" json:"cronExpression,omitempty"`
	MisfirePolicy    int    `gorm:"default:0" json:"misfirePolicy"`
	Concurrent       bool   `gorm:"default:false" json:"concurrent"`
	Status           int    `gorm:"default:0" json:"status"`
}

// JobLog 任务日志模型
type JobLog struct {
	BaseTimeLine
	JobName      string `gorm:"size:64;not null;index" json:"jobName"`
	JobGroup     string `gorm:"size:64;not null" json:"jobGroup"`
	InvokeTarget string `gorm:"size:500;not null" json:"invokeTarget"`
	JobMessage   string `gorm:"size:500" json:"jobMessage,omitempty"`
	Status       int    `json:"status"`
	ExceptionInfo *string `gorm:"type:text" json:"exceptionInfo,omitempty"`
}

// Notice 通知公告模型
type Notice struct {
	BaseModel
	NoticeTitle   string `gorm:"size:50;not null" json:"noticeTitle"`
	NoticeType    string `gorm:"default:'1';size:1" json:"noticeType"`
	NoticeContent *string `gorm:"type:text" json:"noticeContent,omitempty"`
	Status        int    `gorm:"default:0" json:"status"`
}

// GenTable 代码生成表信息模型
type GenTable struct {
	BaseModel
	TableInfo        string     `gorm:"size:200;not null" json:"tableName"`  // 修改字段名
	TableComment      *string    `gorm:"size:500" json:"tableComment,omitempty"`
	ClassName         string     `gorm:"size:100;not null" json:"className"`
	TplCategory       string     `gorm:"size:200;default:'crud'" json:"tplCategory"`
	PackageName       string     `gorm:"size:100" json:"packageName"`
	ModuleName        string     `gorm:"size:30" json:"moduleName"`
	BusinessName      string     `gorm:"size:50" json:"businessName"`
	FunctionName      string     `gorm:"size:50" json:"functionName"`
	FunctionAuthor    string     `gorm:"size:50" json:"functionAuthor"`
	GenType           string     `gorm:"size:1;default:'0'" json:"genType"`
	GenPath           *string    `gorm:"size:200" json:"genPath,omitempty"`
	Options           *string    `gorm:"size:1000" json:"options,omitempty"`
	Columns           []GenColumn `gorm:"foreignKey:TableID;constraint:OnDelete:CASCADE" json:"columns,omitempty"`
}

// GenColumn 代码生成字段信息模型
type GenColumn struct {
	BaseModel
	TableID        string  `gorm:"size:64;not null;index" json:"tableId"`
	ColumnName     string  `gorm:"size:200;not null" json:"columnName"`
	ColumnComment  *string `gorm:"size:500" json:"columnComment,omitempty"`
	ColumnType     string  `gorm:"size:100" json:"columnType"`
	JavaType       *string `gorm:"size:500" json:"javaType,omitempty"`
	JavaField      *string `gorm:"size:100" json:"javaField,omitempty"`
	IsPk           bool    `gorm:"default:false" json:"isPk"`
	IsIncrement    bool    `gorm:"default:false" json:"isIncrement"`
	IsRequired     bool    `gorm:"default:false" json:"isRequired"`
	IsInsert       bool    `gorm:"default:true" json:"isInsert"`
	IsEdit         bool    `gorm:"default:true" json:"isEdit"`
	IsList         bool    `gorm:"default:true" json:"isList"`
	IsQuery        bool    `gorm:"default:false" json:"isQuery"`
	QueryType      string  `gorm:"size:200;default:'EQ'" json:"queryType"`
	HtmlType       string  `gorm:"size:200" json:"htmlType"`
	DictType       *string `gorm:"size:200" json:"dictType,omitempty"`
	Sort           int     `gorm:"default:0" json:"sort"`
}

// TableName 设置表名
func (OperLog) TableName() string {
	return "sys_oper_log"
}

// TableName 设置表名
func (LoginLog) TableName() string {
	return "sys_logininfor"
}

// TableName 设置表名
func (Job) TableName() string {
	return "sys_job"
}

// TableName 设置表名
func (JobLog) TableName() string {
	return "sys_job_log"
}

// TableName 设置表名
func (Notice) TableName() string {
	return "sys_notice"
}

// TableName 设置表名
func (GenTable) TableName() string {
	return "gen_table"
}

// TableName 设置表名
func (GenColumn) TableName() string {
	return "gen_table_column"
}