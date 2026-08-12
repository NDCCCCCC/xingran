package models

import "time"

// OperLog 操作日志模型
type OperLog struct {
	BaseTimeLine
	Title         string    `gorm:"size:50;column:title" json:"title,omitempty"`
	BusinessType  int       `gorm:"column:business_type" json:"businessType"`
	Method        string    `gorm:"size:100;column:method" json:"method,omitempty"`
	RequestMethod string    `gorm:"size:10;column:request_method" json:"requestMethod,omitempty"`
	OperatorType  int       `gorm:"column:operator_type" json:"operatorType"`
	OperatorName  *string   `gorm:"size:50;column:oper_name" json:"operName,omitempty"`
	Nickname      *string   `gorm:"size:50;column:nickname" json:"nickname,omitempty"`
	DeptName      *string   `gorm:"size:50;column:dept_name" json:"deptName,omitempty"`
	OperUrl       *string   `gorm:"size:255;column:oper_url" json:"operUrl,omitempty"`
	OperIP        *string   `gorm:"size:128;column:oper_ip" json:"operIp,omitempty"`
	OperLocation  *string   `gorm:"size:255;column:oper_location" json:"operLocation,omitempty"`
	OperParam     *string   `gorm:"type:text;column:oper_param" json:"operParam,omitempty"`
	JsonResult    *string   `gorm:"type:text;column:json_result" json:"jsonResult,omitempty"`
	Status        int       `gorm:"column:status" json:"status"`
	ErrorMsg      *string   `gorm:"size:2000;column:error_msg" json:"errorMessage,omitempty"`
	OperTime      time.Time `gorm:"column:oper_time" json:"operTime"`
	CostTime      int64     `gorm:"column:cost_time" json:"costTime"`
}

// LoginLog 登录日志模型
type LoginLog struct {
	BaseTimeLine
	Username      string    `gorm:"size:50;column:user_name" json:"userName,omitempty"`
	Nickname      *string   `gorm:"size:50;column:nickname" json:"nickname,omitempty"`
	IPAddr        string    `gorm:"size:128;column:ipaddr" json:"ipAddr,omitempty"`
	LoginLocation *string   `gorm:"size:255;column:login_location" json:"loginLocation,omitempty"`
	Browser       *string   `gorm:"size:50;column:browser" json:"browser,omitempty"`
	OS            *string   `gorm:"size:50;column:os" json:"os,omitempty"`
	Status        int       `gorm:"column:status" json:"status"`
	Msg           *string   `gorm:"size:255;column:msg" json:"message,omitempty"`
	LoginTime     time.Time `gorm:"column:login_time" json:"loginTime"`
}

// MisfirePolicy 错误策略枚举
type MisfirePolicy int

const (
	MisfirePolicyDefault     MisfirePolicy = 0 // 默认策略
	MisfirePolicyImmediately MisfirePolicy = 1 // 立即执行
	MisfirePolicyExecuteOnce MisfirePolicy = 2 // 执行一次
	MisfirePolicyDiscard     MisfirePolicy = 3 // 放弃执行
)

// JobStatus 任务状态枚举（遵循状态值规范：0=正常,1=暂停）
type JobStatus int

const (
	JobStatusNormal JobStatus = 0 // 0: 正常
	JobStatusPause  JobStatus = 1 // 1: 暂停
)

// Job 定时任务模型
type Job struct {
	BaseModel
	JobName        string        `gorm:"size:64;not null" json:"jobName"`
	JobGroup       string        `gorm:"size:64;not null" json:"jobGroup"`
	InvokeTarget   string        `gorm:"size:500;not null" json:"invokeTarget"`
	CronExpression string        `gorm:"size:255" json:"cronExpression"`
	MisfirePolicy  MisfirePolicy `gorm:"default:0" json:"misfirePolicy"`
	Concurrent     bool          `gorm:"default:false" json:"concurrent"`
	Status         JobStatus     `gorm:"default:0" json:"status"`
	NextRunTime    *time.Time    `gorm:"column:next_run_time" json:"nextRunTime,omitempty"`
	PrevRunTime    *time.Time    `gorm:"column:prev_run_time" json:"prevRunTime,omitempty"`
	Remark         *string       `gorm:"size:500" json:"remark,omitempty"`
}

// JobLog 任务日志模型
type JobLog struct {
	BaseTimeLine
	JobName       string     `gorm:"size:64;not null;index" json:"jobName"`
	JobGroup      string     `gorm:"size:64;not null" json:"jobGroup"`
	InvokeTarget  string     `gorm:"size:500;not null" json:"invokeTarget"`
	JobMessage    string     `gorm:"size:500" json:"jobMessage,omitempty"`
	Status        int        `json:"status"`
	ExceptionInfo *string    `gorm:"type:text" json:"exceptionInfo,omitempty"`
	StartTime     *time.Time `gorm:"column:start_time" json:"startTime,omitempty"`
	EndTime       *time.Time `gorm:"column:end_time" json:"endTime,omitempty"`
	Duration      int64      `gorm:"default:0" json:"duration"` // 执行时长(毫秒)
}

// Notice 通知公告模型
type Notice struct {
	BaseModel
	NoticeTitle   string `gorm:"size:100;not null" json:"noticeTitle"`
	NoticeType    string `gorm:"default:'1';size:1" json:"noticeType"`    // 1=公告 2=警告
	NoticeContent string `gorm:"type:text;not null" json:"noticeContent"` // 通知内容
	Status        int    `gorm:"default:0" json:"status"`                 // 0=正常 1=关闭

	// 新增字段
	Priority      NoticePriority `gorm:"default:0" json:"priority"`                // 优先级: 0=普通 1=重要 2=紧急
	PublishTime   *time.Time     `json:"publishTime,omitempty"`                    // 定时发布时间
	PublishStatus PublishStatus  `gorm:"default:0" json:"publishStatus"`           // 发布状态: 0=草稿 1=已发布 2=定时发布中 3=已撤回
	TargetType    TargetType     `gorm:"default:0" json:"targetType"`              // 目标类型: 0=全部 1=部门 2=角色 3=用户
	CreatedByName string         `gorm:"size:64" json:"createdByName"`             // 创建人姓名
	IsMarkdown    bool           `gorm:"default:false" json:"isMarkdown"`          // 是否为Markdown格式
	EndDate       *time.Time     `gorm:"column:end_date" json:"endDate,omitempty"` // 周期性通知结束时间

	// 关联
	Targets     []NoticeTarget        `gorm:"foreignKey:NoticeID" json:"targets,omitempty"`
	Reads       []NoticeRead          `gorm:"foreignKey:NoticeID" json:"reads,omitempty"`
	Attachments []NoticeAttachment    `gorm:"foreignKey:NoticeID" json:"attachments,omitempty"`
	Channels    []NotificationChannel `gorm:"foreignKey:NoticeID" json:"channels,omitempty"`
}

// GenTable 代码生成表信息模型
type GenTable struct {
	BaseModel
	TableInfo      string      `gorm:"size:200;not null" json:"tableName"` // 修改字段名
	TableComment   *string     `gorm:"size:500" json:"tableComment,omitempty"`
	ClassName      string      `gorm:"size:100;not null" json:"className"`
	TplCategory    string      `gorm:"size:200;default:'crud'" json:"tplCategory"`
	PackageName    string      `gorm:"size:100" json:"packageName"`
	ModuleName     string      `gorm:"size:30" json:"moduleName"`
	BusinessName   string      `gorm:"size:50" json:"businessName"`
	FunctionName   string      `gorm:"size:50" json:"functionName"`
	FunctionAuthor string      `gorm:"size:50" json:"functionAuthor"`
	GenType        string      `gorm:"size:1;default:'0'" json:"genType"`
	GenPath        *string     `gorm:"size:200" json:"genPath,omitempty"`
	Options        *string     `gorm:"size:1000" json:"options,omitempty"`
	Columns        []GenColumn `gorm:"foreignKey:TableID;constraint:OnDelete:CASCADE" json:"columns,omitempty"`
}

// GenColumn 代码生成字段信息模型
type GenColumn struct {
	BaseModel
	TableID       string  `gorm:"size:64;not null;index" json:"tableId"`
	ColumnName    string  `gorm:"size:200;not null" json:"columnName"`
	ColumnComment *string `gorm:"size:500" json:"columnComment,omitempty"`
	ColumnType    string  `gorm:"size:100" json:"columnType"`
	JavaType      *string `gorm:"size:500" json:"javaType,omitempty"`
	JavaField     *string `gorm:"size:100" json:"javaField,omitempty"`
	IsPk          bool    `gorm:"default:false" json:"isPk"`
	IsIncrement   bool    `gorm:"default:false" json:"isIncrement"`
	IsRequired    bool    `gorm:"default:false" json:"isRequired"`
	IsInsert      bool    `gorm:"default:true" json:"isInsert"`
	IsEdit        bool    `gorm:"default:true" json:"isEdit"`
	IsList        bool    `gorm:"default:true" json:"isList"`
	IsQuery       bool    `gorm:"default:false" json:"isQuery"`
	QueryType     string  `gorm:"size:200;default:'EQ'" json:"queryType"`
	HtmlType      string  `gorm:"size:200" json:"htmlType"`
	DictType      *string `gorm:"size:200" json:"dictType,omitempty"`
	Sort          int     `gorm:"default:0" json:"sort"`
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
