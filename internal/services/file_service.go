package services

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
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system"
	"gorm.io/gorm"
)

// FileService 文件服务
type FileService struct {
	db            *gorm.DB
	uploadBaseDir string
}

// NewFileService 创建文件服务
func NewFileService(db *gorm.DB) *FileService {
	return &FileService{
		db:            db,
		uploadBaseDir: "./uploads",
	}
}

// FileValidation 文件验证规则
type FileValidation struct {
	MaxSize      int64           // 最大文件大小（字节）
	AllowedExts  map[string]bool // 允许的扩展名
	AllowedMimes map[string]bool // 允许的MIME类型
}

// FileCategoryConfig 文件分类配置
type FileCategoryConfig struct {
	MaxSize           int64    // 最大文件大小（字节）
	AllowedExts       []string // 允许的扩展名列表
	ExtractDimensions bool     // 是否提取图片尺寸
	MetadataOnly      bool     // 仅返回元数据不保存
}

// GetFileValidation 返回FileValidation格式
func (c *FileCategoryConfig) GetFileValidation() *FileValidation {
	extMap := make(map[string]bool)
	for _, ext := range c.AllowedExts {
		extMap[ext] = true
	}
	return &FileValidation{
		MaxSize:     c.MaxSize,
		AllowedExts: extMap,
	}
}

// CategoryConfigs 文件分类配置
var CategoryConfigs = map[string]FileCategoryConfig{
	"avatar": {
		MaxSize:           2 * 1024 * 1024, // 2MB
		AllowedExts:       []string{".jpg", ".jpeg", ".png"},
		ExtractDimensions: true,
	},
	"room-photo": {
		MaxSize:           5 * 1024 * 1024, // 5MB
		AllowedExts:       []string{".jpg", ".jpeg", ".png"},
		ExtractDimensions: true,
	},
	"image": {
		MaxSize:           5 * 1024 * 1024, // 5MB
		AllowedExts:       []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
		ExtractDimensions: true,
	},
	"document": {
		MaxSize:           10 * 1024 * 1024, // 10MB
		AllowedExts:       []string{".pdf", ".doc", ".docx", ".xlsx", ".txt"},
		ExtractDimensions: false,
	},
	"import": {
		MaxSize:           10 * 1024 * 1024, // 10MB
		AllowedExts:       []string{".xlsx", ".xls"},
		ExtractDimensions: false,
	},
	"export": {
		MaxSize:           10 * 1024 * 1024, // 10MB
		AllowedExts:       []string{".xlsx", ".pdf"},
		ExtractDimensions: false,
	},
}

// GetCategoryConfig 获取分类配置，不存在则返回默认图片配置
func GetCategoryConfig(category string) *FileCategoryConfig {
	if config, ok := CategoryConfigs[category]; ok {
		// 返回配置的副本地址
		result := config
		return &result
	}
	// 默认返回图片配置副本
	result := CategoryConfigs["image"]
	return &result
}

// 默认验证规则（保留向后兼容）
var (
	// ImageValidation 图片文件验证（5MB）
	ImageValidation = &FileValidation{
		MaxSize: 5 * 1024 * 1024,
		AllowedExts: map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		},
	}

	// DocumentValidation 文档文件验证（10MB）
	DocumentValidation = &FileValidation{
		MaxSize: 10 * 1024 * 1024,
		AllowedExts: map[string]bool{
			".pdf":  true,
			".doc":  true,
			".docx": true,
			".xls":  true,
			".xlsx": true,
			".txt":  true,
		},
	}
)

// UploadFile 上传文件
func (s *FileService) UploadFile(ctx context.Context, file *multipart.FileHeader, category string, userID string, validation *FileValidation) (*system.SysFile, error) {
	// 1. 验证文件大小
	if validation != nil && file.Size > validation.MaxSize {
		return nil, fmt.Errorf("文件大小超过限制，最大允许 %d MB", validation.MaxSize/(1024*1024))
	}

	// 2. 验证文件扩展名
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if validation != nil && !validation.AllowedExts[ext] {
		return nil, fmt.Errorf("不支持的文件类型: %s", ext)
	}

	// 3. 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	// 4. 计算文件哈希（用于去重）
	hash := sha256.New()
	if _, err := io.Copy(hash, src); err != nil {
		return nil, fmt.Errorf("计算文件哈希失败: %w", err)
	}
	fileHash := hex.EncodeToString(hash.Sum(nil))

	// 5. 检查是否已存在相同文件（去重）
	var existingFile system.SysFile
	err = s.db.WithContext(ctx).Where("file_hash = ? AND is_deleted = ?", fileHash, false).First(&existingFile).Error
	if err == nil {
		// 文件已存在，返回现有记录
		return &existingFile, nil
	}

	// 6. 重置文件指针，准备保存
	if _, err := src.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("重置文件指针失败: %w", err)
	}

	// 7. 生成唯一文件名
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String()[:8], time.Now().Unix(), ext)

	// 8. 确保上传目录存在
	uploadDir := filepath.Join(s.uploadBaseDir, category)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	// 9. 保存文件
	dstPath := filepath.Join(uploadDir, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	// 10. 提取图片尺寸（如果是图片且配置允许）
	var width, height *int
	var metadata *string
	config := GetCategoryConfig(category)
	if config.ExtractDimensions && isImageFile(ext) {
		w, h, err := extractImageDimensions(dstPath)
		if err == nil {
			width, height = &w, &h
		}
		// 构建元数据
		metaMap := map[string]interface{}{
			"width":  w,
			"height": h,
		}
		metaBytes, _ := json.Marshal(metaMap)
		metaStr := string(metaBytes)
		metadata = &metaStr
	}

	// 11. 创建数据库记录
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
		// 数据库创建失败，删除已保存的文件
		os.Remove(dstPath)
		return nil, fmt.Errorf("保存文件记录失败: %w", err)
	}

	return sysFile, nil
}

// isImageFile 判断是否为图片文件
func isImageFile(ext string) bool {
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true,
		".gif": true, ".webp": true, ".bmp": true,
	}
	return imageExts[ext]
}

// extractImageDimensions 提取图片尺寸
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

// GetFile 获取文件信息
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

// DeleteFile 删除文件（软删除）
func (s *FileService) DeleteFile(ctx context.Context, fileID string) error {
	// 1. 获取文件信息
	_, err := s.GetFile(ctx, fileID)
	if err != nil {
		return err
	}

	// 2. 软删除数据库记录
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&system.SysFile{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"is_deleted":  true,
			"delete_time": now,
		}).Error; err != nil {
		return fmt.Errorf("删除文件记录失败: %w", err)
	}

	// 3. 记录访问日志
	_ = s.LogAccess(ctx, fileID, "delete", "", "", "")

	return nil
}

// BatchDeleteFiles 批量删除文件
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

// ListFiles 列出文件
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

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计文件数量失败: %w", err)
	}

	// 分页查询
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&files).Error; err != nil {
		return nil, 0, fmt.Errorf("查询文件列表失败: %w", err)
	}

	return files, total, nil
}

// GetFilePath 获取文件的物理路径
func (s *FileService) GetFilePath(file *system.SysFile) string {
	return filepath.Join(s.uploadBaseDir, file.StoragePath)
}

// GetFileURL 获取文件的访问URL
func (s *FileService) GetFileURL(file *system.SysFile) string {
	return "/uploads/" + file.StoragePath
}

// LogAccess 记录文件访问日志
func (s *FileService) LogAccess(ctx context.Context, fileID, actionType, userID, userName, ipAddress string) error {
	log := &system.SysFileAccessLog{
		FileID:     fileID,
		ActionType: actionType,
		UserID:     userID,
		UserName:   userName,
		IPAddress:  ipAddress,
	}

	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		// 日志记录失败不影响主流程
		return err
	}
	return nil
}

// GetAccessLogs 获取文件访问日志
func (s *FileService) GetAccessLogs(ctx context.Context, fileID string, offset, limit int) ([]*system.SysFileAccessLog, int64, error) {
	var logs []*system.SysFileAccessLog
	var total int64

	query := s.db.WithContext(ctx).Model(&system.SysFileAccessLog{}).Where("file_id = ?", fileID)

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计日志数量失败: %w", err)
	}

	// 分页查询
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询访问日志失败: %w", err)
	}

	return logs, total, nil
}

// CleanupDeletedFiles 清理已删除的物理文件（定时任务）
func (s *FileService) CleanupDeletedFiles(ctx context.Context, days int) (int, error) {
	// 查找指定天数前删除的文件
	cutoffDate := time.Now().AddDate(0, 0, -days)
	var files []*system.SysFile

	if err := s.db.WithContext(ctx).Where("is_deleted = ? AND delete_time < ?", true, cutoffDate).Find(&files).Error; err != nil {
		return 0, fmt.Errorf("查询待清理文件失败: %w", err)
	}

	count := 0
	for _, file := range files {
		// 删除物理文件
		filePath := s.GetFilePath(file)
		if err := os.Remove(filePath); err == nil {
			count++
		}

		// 删除数据库记录（硬删除）
		s.db.WithContext(ctx).Delete(&file)
	}

	return count, nil
}
