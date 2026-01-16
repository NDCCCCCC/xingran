package system

import (
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// SysFile 通用文件模型
type SysFile struct {
	models.BaseModel
	FileName     string     // 原始文件名
	FileSize     int64      // 文件大小(字节)
	FileType     string     // 文件类型(MIME)
	Extension    string     // 文件扩展名
	StoragePath  string     // 存储路径
	FileHash     string     // 文件哈希(SHA256)用于去重
	UploaderID   string     // 上传者ID
	BusinessType string     // 业务类型(room_photo/import/export等)
	IsDeleted    bool       // 是否已删除(软删除)
	DeleteTime   *time.Time // 删除时间
}

// TableName 指定表名
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
