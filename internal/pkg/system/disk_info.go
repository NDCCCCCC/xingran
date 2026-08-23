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
