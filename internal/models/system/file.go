package system

import (
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// SysFile 通用文件模型
type SysFile struct {
	models.BaseModel
	FileName     string
	FileSize     int64
	FileType     string
	Extension    string
	StoragePath  string
	FileHash     string
	UploaderID   string
	BusinessType string
	IsDeleted    bool
	DeleteTime   *time.Time
	Width        *int    `gorm:"column:file_width"`
	Height       *int    `gorm:"column:file_height"`
	Metadata     *string `gorm:"type:jsonb;column:metadata"`
}

func (f *SysFile) GetID() string        { return f.ID }
func (f *SysFile) GetFileName() string  { return f.FileName }
func (f *SysFile) GetFileSize() int64   { return f.FileSize }
func (f *SysFile) GetFileType() string  { return f.FileType }
func (f *SysFile) GetExtension() string { return f.Extension }
func (f *SysFile) GetWidth() *int       { return f.Width }
func (f *SysFile) GetHeight() *int      { return f.Height }
func (f *SysFile) GetMetadata() *string { return f.Metadata }
func (f *SysFile) GetCreatedAt() time.Time {
	return f.CreatedAt
}

func (SysFile) TableName() string {
	return "sys_files"
}

// SysFileAccessLog 文件访问日志模型
type SysFileAccessLog struct {
	models.BaseTimeLine
	FileID     string // 文件ID
	ActionType string // 操作类型(upload/download/delete/view)
	UserID     string // 操作者ID
	UserName   string // 操作者姓名
	IPAddress  string // IP地址
	UserAgent  string // 浏览器信息
}

// TableName 指定表名
func (SysFileAccessLog) TableName() string {
	return "sys_file_access_logs"
}
