package service

import (
	"context"
	"fmt"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/gorm"
)

// MomentMediaService 随念媒体服务
type MomentMediaService struct {
	db *gorm.DB
}

// NewMomentMediaService 创建随念媒体服务实例
func NewMomentMediaService() *MomentMediaService {
	return &MomentMediaService{
		db: database.GetDB(),
	}
}

// AddMediaToMoment 添加媒体文件到随念
func (s *MomentMediaService) AddMediaToMoment(ctx context.Context, momentID int, mediaID int, sort int) error {
	// 检查随念是否存在
	var moment model.Post
	if err := s.db.Where("id = ? AND type = ?", momentID, "moment").First(&moment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("随念不存在")
		}
		return fmt.Errorf("获取随念失败: %w", err)
	}

	// 检查媒体文件是否存在
	var media model.MediaFile
	if err := s.db.First(&media, mediaID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("媒体文件不存在")
		}
		return fmt.Errorf("获取媒体文件失败: %w", err)
	}

	// 创建关联记录
	momentMedia := &model.MomentMedia{
		MomentID: momentID,
		MediaID:  mediaID,
		Sort:     sort,
	}

	if err := s.db.Create(momentMedia).Error; err != nil {
		logger.Errorf("添加媒体到随念失败(随念ID: %d, 媒体ID: %d): %v", momentID, mediaID, err)
		return fmt.Errorf("添加媒体到随念失败: %w", err)
	}

	logger.Infof("媒体添加到随念成功 (随念ID: %d, 媒体ID: %d)", momentID, mediaID)

	return nil
}

// RemoveMediaFromMoment 从随念移除媒体文件
func (s *MomentMediaService) RemoveMediaFromMoment(ctx context.Context, momentID int, mediaID int) error {
	if err := s.db.Where("moment_id = ? AND media_id = ?", momentID, mediaID).Delete(&model.MomentMedia{}).Error; err != nil {
		logger.Errorf("从随念移除媒体失败(随念ID: %d, 媒体ID: %d): %v", momentID, mediaID, err)
		return fmt.Errorf("从随念移除媒体失败: %w", err)
	}

	logger.Infof("媒体从随念移除成功 (随念ID: %d, 媒体ID: %d)", momentID, mediaID)

	return nil
}

// UpdateMediaSort 更新媒体排序
func (s *MomentMediaService) UpdateMediaSort(ctx context.Context, momentID int, mediaID int, sort int) error {
	if err := s.db.Model(&model.MomentMedia{}).
		Where("moment_id = ? AND media_id = ?", momentID, mediaID).
		Update("sort", sort).Error; err != nil {
		logger.Errorf("更新媒体排序失败(随念ID: %d, 媒体ID: %d): %v", momentID, mediaID, err)
		return fmt.Errorf("更新媒体排序失败: %w", err)
	}

	return nil
}

// GetMomentMedia 获取随念的媒体文件
func (s *MomentMediaService) GetMomentMedia(ctx context.Context, momentID int) ([]*model.MomentMedia, error) {
	var momentMedia []*model.MomentMedia

	if err := s.db.Preload("Media").Where("moment_id = ?", momentID).
		Order("sort ASC, id ASC").
		Find(&momentMedia).Error; err != nil {
		logger.Errorf("获取随念媒体失败(随念ID: %d): %v", momentID, err)
		return nil, fmt.Errorf("获取随念媒体失败: %w", err)
	}

	return momentMedia, nil
}

// BatchAddMediaToMoment 批量添加媒体到随念
func (s *MomentMediaService) BatchAddMediaToMoment(ctx context.Context, momentID int, mediaIDs []int) error {
	if len(mediaIDs) == 0 {
		return nil
	}

	// 检查随念是否存在
	var moment model.Post
	if err := s.db.Where("id = ? AND type = ?", momentID, "moment").First(&moment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("随念不存在")
		}
		return fmt.Errorf("获取随念失败: %w", err)
	}

	// 创建关联记录
	momentMedias := make([]*model.MomentMedia, len(mediaIDs))
	for i, mediaID := range mediaIDs {
		momentMedias[i] = &model.MomentMedia{
			MomentID: momentID,
			MediaID:  mediaID,
			Sort:     i, // 默认按添加顺序排序
		}
	}

	if err := s.db.CreateInBatches(momentMedias, 100).Error; err != nil {
		logger.Errorf("批量添加媒体到随念失败(随念ID: %d): %v", momentID, err)
		return fmt.Errorf("批量添加媒体到随念失败: %w", err)
	}

	logger.Infof("批量添加媒体到随念成功 (随念ID: %d, 媒体数量: %d)", momentID, len(mediaIDs))

	return nil
}

// BatchRemoveMediaFromMoment 批量从随念移除媒体
func (s *MomentMediaService) BatchRemoveMediaFromMoment(ctx context.Context, momentID int, mediaIDs []int) error {
	if len(mediaIDs) == 0 {
		return nil
	}

	if err := s.db.Where("moment_id = ? AND media_id IN ?", momentID, mediaIDs).Delete(&model.MomentMedia{}).Error; err != nil {
		logger.Errorf("批量从随念移除媒体失败(随念ID: %d): %v", momentID, err)
		return fmt.Errorf("批量从随念移除媒体失败: %w", err)
	}

	logger.Infof("批量从随念移除媒体成功 (随念ID: %d, 媒体数量: %d)", momentID, len(mediaIDs))

	return nil
}

// ClearMomentMedia 清除随念的所有媒体
func (s *MomentMediaService) ClearMomentMedia(ctx context.Context, momentID int) error {
	if err := s.db.Where("moment_id = ?", momentID).Delete(&model.MomentMedia{}).Error; err != nil {
		logger.Errorf("清除随念媒体失败(随念ID: %d): %v", momentID, err)
		return fmt.Errorf("清除随念媒体失败: %w", err)
	}

	logger.Infof("清除随念媒体成功 (随念ID: %d)", momentID)

	return nil
}

// GetMediaUsageStats 获取媒体使用统计
func (s *MomentMediaService) GetMediaUsageStats(ctx context.Context, mediaID int) (int, error) {
	var count int64
	if err := s.db.Model(&model.MomentMedia{}).Where("media_id = ?", mediaID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取媒体使用统计失败: %w", err)
	}

	return int(count), nil
}

// GetMomentsByMedia 根据媒体文件获取使用它的随念
func (s *MomentMediaService) GetMomentsByMedia(ctx context.Context, mediaID int) ([]*model.Post, error) {
	var moments []*model.Post

	if err := s.db.Table("moments").
		Select("posts.*").
		Joins("INNER JOIN moment_medias ON moments.id = moment_medias.moment_id").
		Where("moment_medias.media_id = ?", mediaID).
		Preload("Author").
		Order("moments.created_at DESC").
		Find(&moments).Error; err != nil {
		logger.Errorf("获取使用媒体的随念失败(媒体ID: %d): %v", mediaID, err)
		return nil, fmt.Errorf("获取使用媒体的随念失败: %w", err)
	}

	return moments, nil
}

// ValidateMediaOwnership 验证媒体文件的所有权
func (s *MomentMediaService) ValidateMediaOwnership(ctx context.Context, momentID int, mediaID int, userID int) error {
	// 检查随念的所有权
	var moment model.Post
	if err := s.db.Where("id = ? AND author_id = ?", momentID, userID).First(&moment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("随念不存在或权限不足")
		}
		return fmt.Errorf("获取随念失败: %w", err)
	}

	// 检查媒体文件是否存在
	var media model.MediaFile
	if err := s.db.First(&media, mediaID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("媒体文件不存在")
		}
		return fmt.Errorf("获取媒体文件失败: %w", err)
	}

	return nil
}