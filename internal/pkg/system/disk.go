package system

// GetDiskUsage 获取磁盘使用率
func GetDiskUsage(path string) (float64, error) {
	// 使用跨平台的实现
	return getDiskUsageByPlatform(path)
}
