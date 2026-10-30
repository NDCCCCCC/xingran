package system

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	image "image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/internal/models/system"
	"gorm.io/gorm"
)

const (
	defaultUploadDir = "./uploads"
	megabyte         = 1024 * 1024
)

var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true,
	".gif": true, ".webp": true, ".bmp": true,
}

type FileService struct {
	db            *gorm.DB
	uploadBaseDir string
}

func NewFileService(db *gorm.DB) *FileService {
	return &FileService{
		db:            db,
		uploadBaseDir: defaultUploadDir,
	}
}

type FileValidation struct {
	MaxSize      int64
	AllowedExts  map[string]bool
	AllowedMimes map[string]bool
}

type UploadFileRequest struct {
	Category string `form:"category" binding:"required"`
}

type ListFilesRequest struct {
	BusinessType string `form:"businessType"`
	UserID       string `form:"userId"`
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

type BatchDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

type SysFileInfo interface {
	GetID() string
	GetFileName() string
	GetFileSize() int64
	GetFileType() string
	GetExtension() string
	GetWidth() *int
	GetHeight() *int
	GetMetadata() *string
	GetCreatedAt() time.Time
}

type FileCategoryConfig struct {
	MaxSize           int64    // 最大文件大小（字节）
	AllowedExts       []string // 允许的扩展名列表
	ExtractDimensions bool
}

func (c *FileCategoryConfig) toFileValidation() *FileValidation {
	extMap := make(map[string]bool, len(c.AllowedExts))
	for _, ext := range c.AllowedExts {
		extMap[ext] = true
	}
	return &FileValidation{
		MaxSize:     c.MaxSize,
		AllowedExts: extMap,
	}
}

func GetValidationByCategory(category string) *FileValidation {
	config := GetCategoryConfig(category)
	if config == nil {
		return ImageValidation
	}
	return config.toFileValidation()
}

var CategoryConfigs = map[string]FileCategoryConfig{
	"avatar": {
		MaxSize:           2 * megabyte,
		AllowedExts:       []string{".jpg", ".jpeg", ".png"},
		ExtractDimensions: true,
	},
	"room-photo": {
		MaxSize:           5 * megabyte,
		AllowedExts:       []string{".jpg", ".jpeg", ".png"},
		ExtractDimensions: true,
	},
	"floor-plan": {
		MaxSize:           5 * megabyte,
		AllowedExts:       []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
		ExtractDimensions: true,
	},
	"image": {
		MaxSize:           5 * megabyte,
		AllowedExts:       []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
		ExtractDimensions: true,
	},
	"document": {
		MaxSize:           10 * megabyte,
		AllowedExts:       []string{".pdf", ".doc", ".docx", ".xlsx", ".txt"},
		ExtractDimensions: false,
	},
	"import": {
		MaxSize:           10 * megabyte,
		AllowedExts:       []string{".xlsx", ".xls"},
		ExtractDimensions: false,
	},
	"export": {
		MaxSize:           10 * megabyte,
		AllowedExts:       []string{".xlsx", ".pdf"},
		ExtractDimensions: false,
	},
}

func GetCategoryConfig(category string) *FileCategoryConfig {
	if config, ok := CategoryConfigs[category]; ok {
		return &config
	}
	defaultConfig := CategoryConfigs["image"]
	return &defaultConfig
}

var (
	ImageValidation = &FileValidation{
		MaxSize: 5 * megabyte,
		AllowedExts: map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true,
			".gif": true, ".webp": true,
		},
	}

	DocumentValidation = &FileValidation{
		MaxSize: 10 * megabyte,
		AllowedExts: map[string]bool{
			".pdf": true, ".doc": true, ".docx": true,
			".xls": true, ".xlsx": true, ".txt": true,
		},
	}
)

func calculateFileHash(file multipart.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("计算文件哈希失败: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *FileService) checkExistingFile(ctx context.Context, fileHash string) (*system.SysFile, error) {
	var existingFile system.SysFile
	err := s.db.WithContext(ctx).Where("file_hash = ? AND is_deleted = ?", fileHash, false).First(&existingFile).Error
	if err == nil {
		return &existingFile, nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, err
}

func buildImageMetadata(width, height int) *string {
	metaMap := map[string]interface{}{
		"width":  width,
		"height": height,
	}
	metaBytes, _ := json.Marshal(metaMap)
	metaStr := string(metaBytes)
	return &metaStr
}

func isImageFile(ext string) bool {
	return imageExtensions[ext]
}

func extractImageDimensions(filePath string) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return img.Width, img.Height, nil
}

func (s *FileService) UploadFile(ctx context.Context, file *multipart.FileHeader, category string, userID string, validation *FileValidation) (*system.SysFile, error) {
	if validation != nil && file.Size > validation.MaxSize {
		maxSizeMB := validation.MaxSize / megabyte
		return nil, fmt.Errorf("文件大小超过限制，最大允许 %d MB", maxSizeMB)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if validation != nil && !validation.AllowedExts[ext] {
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	fileHash, err := calculateFileHash(src)
	if err != nil {
		return nil, err
	}

	existingFile, err := s.checkExistingFile(ctx, fileHash)
	if err != nil {
		return nil, err
	}
	if existingFile != nil {
		return existingFile, nil
	}

	if _, seekErr := src.Seek(0, 0); seekErr != nil {
		return nil, fmt.Errorf("重置文件指针失败: %w", seekErr)
	}

	filename := fmt.Sprintf("%s_%d%s", uuid.New().String()[:8], time.Now().Unix(), ext)

	uploadDir := filepath.Join(s.uploadBaseDir, category)
	if mkdirErr := os.MkdirAll(uploadDir, 0755); mkdirErr != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", mkdirErr)
	}

	dstPath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	config := GetCategoryConfig(category)
	var width, height *int
	var metadata *string
	if config.ExtractDimensions && isImageFile(ext) {
		w, h, err := extractImageDimensions(dstPath)
		if err == nil {
			width, height = &w, &h
			metadata = buildImageMetadata(w, h)
		}
	}

	sysFile := &system.SysFile{
		FileName:     file.Filename,
		FileSize:     file.Size,
		FileType:     file.Header.Get("Content-Type"),
		Extension:    ext,
		StoragePath:  filepath.Join(category, filename),
		FileHash:     fileHash,
		UploaderID:   userID,
		BusinessType: category,
		IsDeleted:    false,
		Width:        width,
		Height:       height,
		Metadata:     metadata,
	}

	if err := s.db.WithContext(ctx).Create(sysFile).Error; err != nil {
		os.Remove(dstPath)
		return nil, fmt.Errorf("保存文件记录失败: %w", err)
	}

	return sysFile, nil
}

func (s *FileService) GetFile(ctx context.Context, fileID string) (*system.SysFile, error) {
	var file system.SysFile
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", fileID, false).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文件不存在")
		}
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}
	return &file, nil
}

func (s *FileService) DeleteFile(ctx context.Context, fileID string) error {
	_, err := s.GetFile(ctx, fileID)
	if err != nil {
		return err
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&system.SysFile{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"is_deleted":  true,
			"delete_time": now,
		}).Error; err != nil {
		return fmt.Errorf("删除文件记录失败: %w", err)
	}

	_ = s.LogAccess(ctx, fileID, "delete", "", "", "")

	return nil
}

func (s *FileService) BatchDeleteFiles(ctx context.Context, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return fmt.Errorf("文件ID列表不能为空")
	}

	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&system.SysFile{}).
		Where("id IN ?", fileIDs).
		Updates(map[string]interface{}{
			"is_deleted":  true,
			"delete_time": now,
		}).Error; err != nil {
		return fmt.Errorf("批量删除文件记录失败: %w", err)
	}

	return nil
}

func (s *FileService) ListFiles(ctx context.Context, businessType string, userID string, offset, limit int) ([]*system.SysFile, int64, error) {
	var files []*system.SysFile
	var total int64

	query := s.db.WithContext(ctx).Model(&system.SysFile{}).Where("is_deleted = ?", false)

	if businessType != "" {
		query = query.Where("business_type = ?", businessType)
	}
	if userID != "" {
		query = query.Where("uploader_id = ?", userID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计文件数量失败: %w", err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&files).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文件列表失败: %w", err)
	}

	return files, total, nil
}

func (s *FileService) GetFilePath(file *system.SysFile) string {
	return filepath.Join(s.uploadBaseDir, file.StoragePath)
}

func (s *FileService) GetFileURL(file *system.SysFile) string {
	return "/uploads/" + file.StoragePath
}

func (s *FileService) LogAccess(ctx context.Context, fileID, actionType, userID, userName, ipAddress string) error {
	log := &system.SysFileAccessLog{
		FileID:     fileID,
		ActionType: actionType,
		UserID:     userID,
		UserName:   userName,
		IPAddress:  ipAddress,
	}

	return s.db.WithContext(ctx).Create(log).Error
}

func (s *FileService) GetAccessLogs(ctx context.Context, fileID string, offset, limit int) ([]*system.SysFileAccessLog, int64, error) {
	var logs []*system.SysFileAccessLog
	var total int64

	query := s.db.WithContext(ctx).Model(&system.SysFileAccessLog{}).Where("file_id = ?", fileID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计日志数量失败: %w", err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询访问日志失败: %w", err)
	}

	return logs, total, nil
}

func (s *FileService) CleanupDeletedFiles(ctx context.Context, days int) (int, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	var files []*system.SysFile

	if err := s.db.WithContext(ctx).Where("is_deleted = ? AND delete_time < ?", true, cutoffDate).Find(&files).Error; err != nil {
		return 0, fmt.Errorf("查询待清理文件失败: %w", err)
	}

	count := 0
	for _, file := range files {
		filePath := s.GetFilePath(file)
		if err := os.Remove(filePath); err == nil {
			count++
		}

		s.db.WithContext(ctx).Delete(&file)
	}

	return count, nil
}
