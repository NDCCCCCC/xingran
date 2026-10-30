package rpa

import (
	"encoding/json"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// WorkerStatus Worker状态
type WorkerStatus string

const (
	WorkerStatusOnline  WorkerStatus = "online"  // 在线
	WorkerStatusOffline WorkerStatus = "offline" // 离线
	WorkerStatusBusy    WorkerStatus = "busy"    // 忙碌
)

// WorkerCapability Worker能力配置
type WorkerCapability struct {
	BrowserTypes   []string `json:"browserTypes"`   // 支持的浏览器类型
	Headless       bool     `json:"headless"`       // 是否支持无头模式
	ScreenShots    bool     `json:"screenShots"`    // 是否支持截图
	FileDownload   bool     `json:"fileDownload"`   // 是否支持文件下载
	FileUpload     bool     `json:"fileUpload"`     // 是否支持文件上传
	MaxTimeout     int      `json:"maxTimeout"`     // 最大超时时间（秒）
	SupportsAI     bool     `json:"supportsAI"`     // 是否支持AI
	AITypes        []string `json:"aiTypes"`        // 支持的AI类型
	OS             string   `json:"os"`             // 操作系统
	BrowserVersion string   `json:"browserVersion"` // 浏览器版本
}

// Worker RPA Worker节点
type Worker struct {
	models.BaseModel
	WorkerName        string          `gorm:"size:100;not null" json:"workerName"`
	WorkerID          string          `gorm:"size:100;uniqueIndex;not null" json:"workerId"`
	IPAddress         string          `gorm:"size:50" json:"ipAddress"`
	Port              int             `json:"port"`
	Status            WorkerStatus    `gorm:"size:20;default:'offline'" json:"status"`
	Capabilities      json.RawMessage `gorm:"type:jsonb" json:"capabilities"`
	MaxConcurrency    int             `gorm:"default:3" json:"maxConcurrency"`
	CurrentTasks      int             `gorm:"default:0" json:"currentTasks"`
	LastHeartbeat     *int64          `json:"lastHeartbeat"`
	DockerContainerID string          `gorm:"size:100" json:"dockerContainerId"`
}

// TableName 指定表名
func (Worker) TableName() string {
	return "sys_rpa_workers"
}

// BeforeCreate GORM钩子
func (w *Worker) BeforeCreate(tx *gorm.DB) error {
	_ = w.BaseModel.BeforeCreate(tx)
	return nil
}

// GetCapabilities 获取能力配置
func (w *Worker) GetCapabilities() (*WorkerCapability, error) {
	if w.Capabilities == nil {
		return &WorkerCapability{}, nil
	}
	var caps WorkerCapability
	if err := json.Unmarshal(w.Capabilities, &caps); err != nil {
		return nil, err
	}
	return &caps, nil
}

// SetCapabilities 设置能力配置
func (w *Worker) SetCapabilities(caps WorkerCapability) error {
	data, err := json.Marshal(caps)
	if err != nil {
		return err
	}
	w.Capabilities = data
	return nil
}

// IsOnline 是否在线
func (w *Worker) IsOnline() bool {
	return w.Status == WorkerStatusOnline || w.Status == WorkerStatusBusy
}

// IsAvailable 是否可用（未达到最大并发数）
func (w *Worker) IsAvailable() bool {
	return w.IsOnline() && w.CurrentTasks < w.MaxConcurrency
}
