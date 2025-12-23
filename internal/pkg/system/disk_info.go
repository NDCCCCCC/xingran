package system

// 没有使用的导入，已移除

// DiskInfo 磁盘信息结构
type DiskInfo struct {
	MountPoint string `json:"mountPoint"` // 挂载点
	Total      uint64 `json:"total"`      // 总容量(字节)
	Available  uint64 `json:"available"`  // 可用容量(字节)
	Used       uint64 `json:"used"`       // 已用容量(字节)
}

// GetAllDiskInfo 获取所有磁盘信息
func GetAllDiskInfo() ([]DiskInfo, error) {
	return getAllDiskInfoByPlatform()
}

// GetDiskInfoDetailed 获取指定路径的磁盘信息（导出函数）
func GetDiskInfoDetailed(path string) (uint64, uint64, error) {
	total, available, err := getDiskInfoByPlatform(path)
	if err != nil {
		return 0, 0, err
	}
	return total, available, nil
}

// getEstimatedDiskInfo 获取估算的磁盘信息
func getEstimatedDiskInfo() (uint64, uint64, error) {
	// 使用估算值：假设磁盘容量为500GB，75%使用
	totalBytes := uint64(500) * 1024 * 1024 * 1024 // 500GB
	availableBytes := totalBytes / 4                 // 25%可用 = 125GB

	return totalBytes, availableBytes, nil
}

// getDiskInfoByPlatform 获取磁盘总容量和可用容量（内部函数）
func getDiskInfoByPlatform(path string) (uint64, uint64, error) {
	return GetDiskInfoDetailed(path)
}