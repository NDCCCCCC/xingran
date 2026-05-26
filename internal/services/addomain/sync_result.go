package addomain

import "time"

// DeptSyncResult 部门同步结果
type DeptSyncResult struct {
	StartTime    time.Time        `json:"startTime"`
	EndTime      time.Time        `json:"endTime"`
	Duration     time.Duration    `json:"duration"`
	TotalDepts   int              `json:"totalDepts"`
	SuccessDepts int              `json:"successDepts"`
	FailedDepts  int              `json:"failedDepts"`
	SkippedDepts int              `json:"skippedDepts"`
	Errors       []DeptSyncError  `json:"errors,omitempty"`
	ADConfigID   string           `json:"adConfigId"`
	Status       string           `json:"status"` // running/completed/failed
}

// DeptSyncError 部门同步错误
type DeptSyncError struct {
	DeptID   string `json:"deptId"`
	DeptName string `json:"deptName"`
	Error    string `json:"error"`
}

// DeptSyncStats 部门同步统计
type DeptSyncStats struct {
	LastSyncTime   *time.Time `json:"lastSyncTime,omitempty"`
	LastSyncStatus string     `json:"lastSyncStatus"` // pending/synced/failed
	TotalDepts     int        `json:"totalDepts"`
	SyncedDepts    int        `json:"syncedDepts"`
	FailedDepts    int        `json:"failedDepts"`
	NextSyncTime   *time.Time `json:"nextSyncTime,omitempty"`
}
