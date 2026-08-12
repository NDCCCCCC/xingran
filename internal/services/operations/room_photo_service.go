package operations

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/xingran-next/xingran-go-backend/internal/models/operations"
	"github.com/xingran-next/xingran-go-backend/internal/models/system"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"gorm.io/gorm"
)

// RoomPhotoService 机房照片服务接口
type RoomPhotoService interface {
	UploadPhotos(ctx context.Context, roomID string, files []*multipart.FileHeader, primaryIndex int, userID string) ([]*operations.OpsRoomPhoto, error)
	ListByRoom(ctx context.Context, roomID string) ([]*operations.OpsRoomPhoto, error)
	GetPhotoWithFile(ctx context.Context, photoID string) (*operations.OpsRoomPhoto, *system.SysFile, error)
	SetPrimary(ctx context.Context, photoID string) error
	UpdateSort(ctx context.Context, photoIDs []string) error
	UpdateDescription(ctx context.Context, photoID string, description string) error
	DeletePhoto(ctx context.Context, photoID string) error
	BatchDeletePhotos(ctx context.Context, photoIDs []string) error
	GetPrimaryPhoto(ctx context.Context, roomID string) (*operations.OpsRoomPhoto, error)
	CountPhotos(ctx context.Context, roomID string) (int64, error)
}

type roomPhotoService struct {
	db          *gorm.DB
	fileService *systemServices.FileService
}

// NewRoomPhotoService 创建机房照片服务实例
func NewRoomPhotoService(db *gorm.DB) RoomPhotoService {
	return &roomPhotoService{
		db:          db,
		fileService: systemServices.NewFileService(db),
	}
}

// UploadPhotos 批量上传机房照片
func (s *roomPhotoService) UploadPhotos(ctx context.Context, roomID string, files []*multipart.FileHeader, primaryIndex int, userID string) ([]*operations.OpsRoomPhoto, error) {
	if len(files) == 0 {
		return nil, apperrors.ParamMissing("照片")
	}

	// 验证机房是否存在
	var room operations.OpsServerRoom
	if err := s.db.WithContext(ctx).Where("id = ?", roomID).First(&room).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ServerRoomNotFound()
		}
		return nil, fmt.Errorf("验证机房失败: %w", err)
	}

	// 上传文件并创建照片记录
	photos := make([]*operations.OpsRoomPhoto, 0, len(files))
	for i, file := range files {
		// 上传文件
		sysFile, err := s.fileService.UploadFile(ctx, file, "room-photo", userID, systemServices.ImageValidation)
		if err != nil {
			return nil, fmt.Errorf("上传文件失败: %w", err)
		}

		// 获取当前最大排序号
		var maxSort int
		s.db.WithContext(ctx).Model(&operations.OpsRoomPhoto{}).
			Where("room_id = ?", roomID).
			Select("COALESCE(MAX(sort_order), -1) + 1").
			Scan(&maxSort)

		// 如果是第一个上传或指定为主图，设置为主图
		isPrimary := (i == primaryIndex) || (i == 0 && len(photos) == 0)

		// 创建照片记录
		photo := &operations.OpsRoomPhoto{
			RoomID:    roomID,
			FileID:    sysFile.ID,
			SortOrder: maxSort,
			IsPrimary: isPrimary,
		}
		// 设置文件URL
		photo.FileURL = &sysFile.StoragePath

		if err := s.db.WithContext(ctx).Create(photo).Error; err != nil {
			return nil, fmt.Errorf("保存照片记录失败: %w", err)
		}

		photos = append(photos, photo)

		// 如果设置为主图，取消其他主图
		if isPrimary {
			s.db.WithContext(ctx).Model(&operations.OpsRoomPhoto{}).
				Where("room_id = ? AND id != ?", roomID, photo.ID).
				Update("is_primary", false)
		}
	}

	return photos, nil
}

// ListByRoom 获取机房照片列表
func (s *roomPhotoService) ListByRoom(ctx context.Context, roomID string) ([]*operations.OpsRoomPhoto, error) {
	var photos []*operations.OpsRoomPhoto

	if err := s.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("is_primary DESC, sort_order ASC, created_at ASC").
		Find(&photos).Error; err != nil {
		return nil, fmt.Errorf("查询照片列表失败: %w", err)
	}

	return photos, nil
}

// SetPrimary 设置主图
func (s *roomPhotoService) SetPrimary(ctx context.Context, photoID string) error {
	// 获取照片信息
	var photo operations.OpsRoomPhoto
	if err := s.db.WithContext(ctx).Where("id = ?", photoID).First(&photo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.RoomPhotoNotFound()
		}
		return fmt.Errorf("查询照片失败: %w", err)
	}

	// 使用事务更新
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 取消该机房的其他主图
		if err := tx.Model(&operations.OpsRoomPhoto{}).
			Where("room_id = ? AND id != ?", photo.RoomID, photoID).
			Update("is_primary", false).Error; err != nil {
			return fmt.Errorf("取消旧主图失败: %w", err)
		}

		// 设置新的主图
		if err := tx.Model(&operations.OpsRoomPhoto{}).
			Where("id = ?", photoID).
			Update("is_primary", true).Error; err != nil {
			return fmt.Errorf("设置主图失败: %w", err)
		}

		return nil
	})
}

// UpdateSort 更新照片排序
func (s *roomPhotoService) UpdateSort(ctx context.Context, photoIDs []string) error {
	if len(photoIDs) == 0 {
		return apperrors.ParamMissing("照片ID列表")
	}

	// 使用事务批量更新排序
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, photoID := range photoIDs {
			if err := tx.Model(&operations.OpsRoomPhoto{}).
				Where("id = ?", photoID).
				Update("sort_order", i).Error; err != nil {
				return fmt.Errorf("更新照片排序失败: %w", err)
			}
		}
		return nil
	})
}

// UpdateDescription 更新照片描述
func (s *roomPhotoService) UpdateDescription(ctx context.Context, photoID string, description string) error {
	result := s.db.WithContext(ctx).
		Model(&operations.OpsRoomPhoto{}).
		Where("id = ?", photoID).
		Update("description", &description)

	if result.Error != nil {
		return fmt.Errorf("更新照片描述失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return apperrors.RoomPhotoNotFound()
	}

	return nil
}

// DeletePhoto 删除照片
func (s *roomPhotoService) DeletePhoto(ctx context.Context, photoID string) error {
	// 获取照片信息
	var photo operations.OpsRoomPhoto
	if err := s.db.WithContext(ctx).Where("id = ?", photoID).First(&photo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.RoomPhotoNotFound()
		}
		return fmt.Errorf("查询照片失败: %w", err)
	}

	// 使用事务删除
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除照片记录
		if err := tx.Delete(&photo).Error; err != nil {
			return fmt.Errorf("删除照片记录失败: %w", err)
		}

		// 删除关联的文件
		if err := s.fileService.DeleteFile(ctx, photo.FileID); err != nil {
			// 文件删除失败不影响照片删除
			return nil
		}

		// 如果删除的是主图，设置第一张照片为主图
		if photo.IsPrimary {
			var firstPhoto operations.OpsRoomPhoto
			if err := tx.Where("room_id = ?", photo.RoomID).
				Order("sort_order ASC").
				First(&firstPhoto).Error; err == nil {
				tx.Model(&firstPhoto).Update("is_primary", true)
			}
		}

		return nil
	})
}

// BatchDeletePhotos 批量删除照片
func (s *roomPhotoService) BatchDeletePhotos(ctx context.Context, photoIDs []string) error {
	if len(photoIDs) == 0 {
		return apperrors.ParamMissing("照片ID列表")
	}

	for _, photoID := range photoIDs {
		if err := s.DeletePhoto(ctx, photoID); err != nil {
			return fmt.Errorf("删除照片 %s 失败: %w", photoID, err)
		}
	}

	return nil
}

// GetPrimaryPhoto 获取机房主图
func (s *roomPhotoService) GetPrimaryPhoto(ctx context.Context, roomID string) (*operations.OpsRoomPhoto, error) {
	var photo operations.OpsRoomPhoto

	if err := s.db.WithContext(ctx).
		Where("room_id = ? AND is_primary = ?", roomID, true).
		First(&photo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果没有主图，返回第一张照片
			if err2 := s.db.WithContext(ctx).
				Where("room_id = ?", roomID).
				Order("sort_order ASC").
				First(&photo).Error; err2 != nil {
				return nil, fmt.Errorf("查询主图失败: %w", err2)
			}
			return &photo, nil
		}
		return nil, fmt.Errorf("查询主图失败: %w", err)
	}

	return &photo, nil
}

// GetPhotoWithFile 获取照片及文件信息
func (s *roomPhotoService) GetPhotoWithFile(ctx context.Context, photoID string) (*operations.OpsRoomPhoto, *system.SysFile, error) {
	var photo operations.OpsRoomPhoto
	if err := s.db.WithContext(ctx).Where("id = ?", photoID).First(&photo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, apperrors.RoomPhotoNotFound()
		}
		return nil, nil, fmt.Errorf("查询照片失败: %w", err)
	}

	var file system.SysFile
	if err := s.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", photo.FileID, false).First(&file).Error; err != nil {
		return nil, nil, fmt.Errorf("查询文件失败: %w", err)
	}

	return &photo, &file, nil
}

// CountPhotos 统计机房照片数量
func (s *roomPhotoService) CountPhotos(ctx context.Context, roomID string) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&operations.OpsRoomPhoto{}).
		Where("room_id = ?", roomID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计照片数量失败: %w", err)
	}
	return count, nil
}
