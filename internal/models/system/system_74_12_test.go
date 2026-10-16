package system

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =====================================================================
// 74-11 escalation gap-closure: internal/models/system — SysFile
// TableName + getter 访问器锁定。
// =====================================================================

func TestSysFileTableNames(t *testing.T) {
	assert.Equal(t, "sys_files", SysFile{}.TableName())
	assert.Equal(t, "sys_file_access_logs", SysFileAccessLog{}.TableName())
}

func TestSysFileGetters(t *testing.T) {
	w, h := 800, 600
	meta := `{"k":"v"}`
	f := SysFile{
		FileName:  "plan.png",
		FileSize:  1024,
		FileType:  "image/png",
		Extension: ".png",
		Width:     &w,
		Height:    &h,
		Metadata:  &meta,
	}
	f.ID = "f-1" // 嵌入 BaseModel 字段逐个赋值

	assert.Equal(t, "f-1", f.GetID())
	assert.Equal(t, "plan.png", f.GetFileName())
	assert.Equal(t, int64(1024), f.GetFileSize())
	assert.Equal(t, "image/png", f.GetFileType())
	assert.Equal(t, ".png", f.GetExtension())
	assert.Equal(t, &w, f.GetWidth())
	assert.Equal(t, &h, f.GetHeight())
	assert.Equal(t, &meta, f.GetMetadata())
}
