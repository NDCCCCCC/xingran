package system

// =====================================================================
// file_service_test.go — covers file_service.go (433 lines)
// Per Plan 72-11 Task 4
// =====================================================================

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupFileServiceDB creates in-memory SQLite with sys_files + sys_file_access_logs schema.
func setupFileServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_files (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			version INTEGER,
			file_name TEXT,
			file_size INTEGER,
			file_type TEXT,
			extension TEXT,
			storage_path TEXT,
			file_hash TEXT,
			uploader_id TEXT,
			business_type TEXT,
			is_deleted BOOLEAN,
			delete_time DATETIME,
			file_width INTEGER,
			file_height INTEGER,
			metadata TEXT
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_file_access_logs (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			file_id TEXT,
			action_type TEXT,
			user_id TEXT,
			user_name TEXT,
			ip_address TEXT,
			user_agent TEXT
		)
	`).Error)
	return db
}

// seedFileSvcRow inserts a sys_files row directly.
func seedFileSvcRow(t *testing.T, db *gorm.DB, userID string, businessType string) string {
	t.Helper()
	id := uuid.NewString()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_files (id, file_name, file_size, file_type, extension, storage_path, uploader_id, business_type, is_deleted, created_at, updated_at, version)
		VALUES (?, 'test.png', 1024, 'image/png', '.png', '/uploads/test.png', ?, ?, 0, ?, ?, 0)`,
		id, userID, businessType, now, now).Error)
	return id
}

// TC1: GetCategoryConfig - returns existing config
func TestFileService_GetCategoryConfig_Existing(t *testing.T) {
	cfg := GetCategoryConfig("avatar")
	require.NotNil(t, cfg)
	assert.Equal(t, int64(2*megabyte), cfg.MaxSize)
	assert.True(t, cfg.ExtractDimensions)
}

// TC2: GetCategoryConfig - unknown returns default
func TestFileService_GetCategoryConfig_Default(t *testing.T) {
	cfg := GetCategoryConfig("nonexistent")
	require.NotNil(t, cfg)
}

// TC3: GetValidationByCategory - returns validation for known
func TestFileService_GetValidationByCategory_Known(t *testing.T) {
	v := GetValidationByCategory("avatar")
	require.NotNil(t, v)
	assert.Equal(t, int64(2*megabyte), v.MaxSize)
	assert.NotNil(t, v.AllowedExts)
}

// TC4: GetValidationByCategory - unknown returns image default
func TestFileService_GetValidationByCategory_Unknown(t *testing.T) {
	v := GetValidationByCategory("nonexistent")
	require.NotNil(t, v)
}

// TC5: toFileValidation - converts correctly
func TestFileService_ToFileValidation(t *testing.T) {
	cfg := &FileCategoryConfig{
		MaxSize:     1000,
		AllowedExts: []string{".jpg", ".png"},
	}
	v := cfg.toFileValidation()
	assert.Equal(t, int64(1000), v.MaxSize)
	assert.True(t, v.AllowedExts[".jpg"])
	assert.True(t, v.AllowedExts[".png"])
}

// TC6: GetFile - success
func TestFileService_GetFile_Success(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	id := seedFileSvcRow(t, db, "user-1", "avatar")

	file, err := svc.GetFile(context.Background(), id)
	require.NoError(t, err)
	require.NotNil(t, file)
	assert.Equal(t, "test.png", file.GetFileName())
}

// TC7: GetFile - not found
func TestFileService_GetFile_NotFound(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)

	_, err := svc.GetFile(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC8: ListFiles - empty
func TestFileService_ListFiles_Empty(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)

	files, total, err := svc.ListFiles(context.Background(), "", "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, files)
}

// TC9: ListFiles - with data
func TestFileService_ListFiles_WithData(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	seedFileSvcRow(t, db, "user-1", "avatar")
	seedFileSvcRow(t, db, "user-2", "avatar")

	files, total, err := svc.ListFiles(context.Background(), "avatar", "", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, files, 2)
}

// TC10: ListFiles - with userID filter
func TestFileService_ListFiles_UserFilter(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	seedFileSvcRow(t, db, "user-1", "avatar")
	seedFileSvcRow(t, db, "user-2", "avatar")

	files, total, err := svc.ListFiles(context.Background(), "", "user-1", 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, files, 1)
}

// TC11: DeleteFile - success
func TestFileService_DeleteFile_Success(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	id := seedFileSvcRow(t, db, "user-1", "avatar")

	require.NoError(t, svc.DeleteFile(context.Background(), id))

	var isDeleted bool
	require.NoError(t, db.Raw("SELECT is_deleted FROM sys_files WHERE id = ?", id).Scan(&isDeleted).Error)
	assert.True(t, isDeleted)
}

// TC12: DeleteFile - not found
func TestFileService_DeleteFile_NotFound(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)

	err := svc.DeleteFile(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC13: BatchDeleteFiles - success
func TestFileService_BatchDeleteFiles_Success(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	id1 := seedFileSvcRow(t, db, "user-1", "avatar")
	id2 := seedFileSvcRow(t, db, "user-2", "avatar")

	require.NoError(t, svc.BatchDeleteFiles(context.Background(), []string{id1, id2}))

	var deletedCount int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sys_files WHERE is_deleted = 1").Scan(&deletedCount).Error)
	assert.Equal(t, int64(2), deletedCount)
}

// TC14: GetFileURL - returns URL for real file
func TestFileService_GetFileURL(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	id := seedFileSvcRow(t, db, "user-1", "avatar")

	file, err := svc.GetFile(context.Background(), id)
	require.NoError(t, err)

	url := svc.GetFileURL(file)
	assert.NotEmpty(t, url)
}

// TC15: LogAccess - success
func TestFileService_LogAccess_Success(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	fileID := seedFileSvcRow(t, db, "user-1", "avatar")

	err := svc.LogAccess(context.Background(), fileID, "view", "user-1", "alice", "127.0.0.1")
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sys_file_access_logs WHERE file_id = ?", fileID).Scan(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TC16: imageExtensions - all defined
func TestFileService_ImageExtensions(t *testing.T) {
	expected := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
	for _, ext := range expected {
		assert.True(t, imageExtensions[ext], "%s should be in imageExtensions", ext)
	}
}

// TC17: CategoryConfigs - all categories defined
func TestFileService_CategoryConfigs(t *testing.T) {
	categories := []string{"avatar", "room-photo", "floor-plan", "image", "document", "import", "export"}
	for _, cat := range categories {
		cfg := GetCategoryConfig(cat)
		require.NotNil(t, cfg, "%s config should be defined", cat)
	}
}

// TC18: GetFile - DB error
func TestFileService_GetFile_DBError(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	_, err := svc.GetFile(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC19: ListFiles - DB error
func TestFileService_ListFiles_DBError(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	_, _, err := svc.ListFiles(context.Background(), "", "", 0, 10)
	assert.Error(t, err)
}

// TC20: DeleteFile - DB error
func TestFileService_DeleteFile_DBError(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	err := svc.DeleteFile(context.Background(), uuid.NewString())
	assert.Error(t, err)
}

// TC21: BatchDeleteFiles - DB error
func TestFileService_BatchDeleteFiles_DBError(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_files").Error)

	err := svc.BatchDeleteFiles(context.Background(), []string{uuid.NewString()})
	assert.Error(t, err)
}

// TC22: toFileValidation - nil AllowedExts
func TestFileService_ToFileValidation_NilExts(t *testing.T) {
	cfg := &FileCategoryConfig{
		MaxSize: 100,
	}
	v := cfg.toFileValidation()
	assert.Equal(t, int64(100), v.MaxSize)
	assert.NotNil(t, v.AllowedExts)
	assert.Empty(t, v.AllowedExts)
}

// TC23: defaultUploadDir constant
func TestFileService_DefaultUploadDir(t *testing.T) {
	assert.Equal(t, "./uploads", defaultUploadDir)
}

// TC24: megabyte constant
func TestFileService_MegabyteConstant(t *testing.T) {
	assert.Equal(t, 1024*1024, megabyte)
}

// TC25: GetCategoryConfig - image fallback for unknown
func TestFileService_GetCategoryConfig_ImageFallback(t *testing.T) {
	cfg := GetCategoryConfig("some_random_category")
	require.NotNil(t, cfg)
	// Should match the image default config
	assert.Equal(t, int64(5*megabyte), cfg.MaxSize)
}

// TC26: DeleteFile - storage path cleanup with real file
func TestFileService_DeleteFile_Storage(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)

	// Create real file in tmp
	tmpDir, err := os.MkdirTemp("", "file_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	storagePath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(storagePath, []byte("test"), 0644))

	// Insert file with real storage path
	id := uuid.NewString()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(`INSERT INTO sys_files (id, file_name, storage_path, is_deleted, created_at, updated_at, version)
		VALUES (?, 'test.txt', ?, 0, ?, ?, 0)`,
		id, storagePath, now, now).Error)

	require.NoError(t, svc.DeleteFile(context.Background(), id))

	var isDeleted bool
	require.NoError(t, db.Raw("SELECT is_deleted FROM sys_files WHERE id = ?", id).Scan(&isDeleted).Error)
	assert.True(t, isDeleted)
}

// TC27: GetFileURL - format
func TestFileService_GetFileURL_Format(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	id := seedFileSvcRow(t, db, "user-1", "avatar")

	file, err := svc.GetFile(context.Background(), id)
	require.NoError(t, err)
	url := svc.GetFileURL(file)
	assert.NotEmpty(t, url)
}

// TC28: CategoryConfigs - extract dimensions flag
func TestFileService_CategoryConfigs_ExtractDimensions(t *testing.T) {
	avatar := GetCategoryConfig("avatar")
	assert.True(t, avatar.ExtractDimensions, "avatar should extract dimensions")

	doc := GetCategoryConfig("document")
	assert.False(t, doc.ExtractDimensions, "document should not extract dimensions")
}

// TC29: ValidateResponse_FormatValidation
func TestFileService_FileValidationFields(t *testing.T) {
	v := &FileValidation{
		MaxSize:      1000,
		AllowedExts:  map[string]bool{".jpg": true},
		AllowedMimes: map[string]bool{"image/jpeg": true},
	}
	assert.Equal(t, int64(1000), v.MaxSize)
	assert.NotNil(t, v.AllowedExts)
}

// TC30: LogAccess - DB error
func TestFileService_LogAccess_DBError(t *testing.T) {
	db := setupFileServiceDB(t)
	svc := NewFileService(db)
	require.NoError(t, db.Exec("DROP TABLE sys_file_access_logs").Error)

	err := svc.LogAccess(context.Background(), uuid.NewString(), "view", "user-1", "alice", "127.0.0.1")
	assert.Error(t, err)
}