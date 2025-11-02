package repository

import (
	"fmt"

	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/media/model"
	"gorm.io/gorm"
)

// mediaUsageRepository 媒体使用记录仓库实现
type mediaUsageRepository struct {
	db *gorm.DB
}

// NewMediaUsageRepository 创建媒体使用记录仓库实例
func NewMediaUsageRepository() MediaUsageRepository {
	return &mediaUsageRepository{
		db: database.GetDB(),
	}
}

// Create 创建媒体使用记录
func (r *mediaUsageRepository) Create(usage *model.MediaUsage) error {
	if err := r.db.Create(usage).Error; err != nil {
		logger.Errorf("创建媒体使用记录失败: %v", err)
		return fmt.Errorf("创建媒体使用记录失败: %w", err)
	}

	logger.Infof("媒体使用记录创建成功 (ID: %d, 媒体ID: %d, 类型: %s)",
		usage.ID, usage.MediaID, usage.UsageType)
	return nil
}

// GetByID 根据ID获取媒体使用记录
func (r *mediaUsageRepository) GetByID(id int) (*model.MediaUsage, error) {
	var usage model.MediaUsage
	err := r.db.Preload("Media").Preload("Media.Uploader").
		Preload("Uploader").First(&usage, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("媒体使用记录不存在")
		}
		logger.Errorf("获取媒体使用记录失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取媒体使用记录失败: %w", err)
	}

	return &usage, nil
}

// GetByMediaID 根据媒体ID获取使用记录
func (r *mediaUsageRepository) GetByMediaID(mediaID int) ([]*model.MediaUsage, error) {
	var usages []*model.MediaUsage

	if err := r.db.Where("media_id = ?", mediaID).
		Preload("Media").Preload("Media.Uploader").
		Preload("Uploader").
		Order("created_at DESC").
		Find(&usages).Error; err != nil {
		logger.Errorf("根据媒体ID获取使用记录失败(媒体ID: %d): %v", mediaID, err)
		return nil, fmt.Errorf("根据媒体ID获取使用记录失败: %w", err)
	}

	return usages, nil
}

// GetByTarget 根据目标获取使用记录
func (r *mediaUsageRepository) GetByTarget(usageType string, targetID int) ([]*model.MediaUsage, error) {
	var usages []*model.MediaUsage

	if err := r.db.Where("usage_type = ? AND target_id = ?", usageType, targetID).
		Preload("Media").Preload("Media.Uploader").
		Preload("Uploader").
		Order("created_at DESC").
		Find(&usages).Error; err != nil {
		logger.Errorf("根据目标获取使用记录失败(类型: %s, 目标ID: %d): %v", usageType, targetID, err)
		return nil, fmt.Errorf("根据目标获取使用记录失败: %w", err)
	}

	return usages, nil
}

// Delete 删除媒体使用记录
func (r *mediaUsageRepository) Delete(id int) error {
	if err := r.db.Delete(&model.MediaUsage{}, id).Error; err != nil {
		logger.Errorf("删除媒体使用记录失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除媒体使用记录失败: %w", err)
	}

	logger.Infof("媒体使用记录删除成功 (ID: %d)", id)
	return nil
}

// DeleteByMediaID 根据媒体ID删除使用记录
func (r *mediaUsageRepository) DeleteByMediaID(mediaID int) error {
	if err := r.db.Where("media_id = ?", mediaID).Delete(&model.MediaUsage{}).Error; err != nil {
		logger.Errorf("根据媒体ID删除使用记录失败(媒体ID: %d): %v", mediaID, err)
		return fmt.Errorf("根据媒体ID删除使用记录失败: %w", err)
	}

	logger.Infof("根据媒体ID删除使用记录成功 (媒体ID: %d)", mediaID)
	return nil
}

// List 获取媒体使用记录列表
func (r *mediaUsageRepository) List(page, size int) ([]*model.MediaUsage, int64, error) {
	var usages []*model.MediaUsage
	var total int64

	query := r.db.Model(&model.MediaUsage{})

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取媒体使用记录总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取媒体使用记录总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Media").Preload("Media.Uploader").
		Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&usages).Error; err != nil {
		logger.Errorf("获取媒体使用记录列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取媒体使用记录列表失败: %w", err)
	}

	return usages, total, nil
}

// GetUsageStats 获取使用统计
func (r *mediaUsageRepository) GetUsageStats() (*UsageStats, error) {
	stats := &UsageStats{
		UsageByType: make(map[string]int64),
	}

	// 总使用数
	if err := r.db.Model(&model.MediaUsage{}).
		Count(&stats.TotalUsages).Error; err != nil {
		logger.Errorf("获取总使用数失败: %v", err)
		return nil, fmt.Errorf("获取总使用数失败: %w", err)
	}

	// 按类型统计
	typeStats := []struct {
		UsageType string `json:"usage_type"`
		Count     int64  `json:"count"`
	}{}

	if err := r.db.Model(&model.MediaUsage{}).
		Select("usage_type, COUNT(*) as count").
		Group("usage_type").
		Scan(&typeStats).Error; err != nil {
		logger.Errorf("获取使用类型统计失败: %v", err)
		return nil, fmt.Errorf("获取使用类型统计失败: %w", err)
	}

	for _, ts := range typeStats {
		stats.UsageByType[ts.UsageType] = ts.Count
	}

	// 获取热门媒体（使用次数最多的媒体）
	topMediaQuery := `
		SELECT m.*, COUNT(mu.id) as usage_count
		FROM media_files m
		LEFT JOIN media_usages mu ON m.id = mu.media_id
		WHERE m.status = 'active'
		GROUP BY m.id
		ORDER BY usage_count DESC, m.created_at DESC
		LIMIT 10
	`

	if err := r.db.Raw(topMediaQuery).Scan(&stats.TopMedia).Error; err != nil {
		logger.Errorf("获取热门媒体失败: %v", err)
		// 不返回错误，只是热门媒体为空
		stats.TopMedia = []model.MediaFile{}
	}

	// 获取最近的使用记录
	if err := r.db.Preload("Media").Preload("Media.Uploader").
		Preload("Uploader").
		Order("created_at DESC").
		Limit(10).
		Find(&stats.RecentUsages).Error; err != nil {
		logger.Errorf("获取最近使用记录失败: %v", err)
		// 不返回错误，只是最近使用记录为空
		stats.RecentUsages = []*model.MediaUsage{}
	}

	return stats, nil
}

// GetUsageCountByMedia 获取媒体文件的使用次数
func (r *mediaUsageRepository) GetUsageCountByMedia(mediaID int) (int64, error) {
	var count int64
	if err := r.db.Model(&model.MediaUsage{}).
		Where("media_id = ?", mediaID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取媒体使用次数失败: %w", err)
	}

	return count, nil
}

// GetUsagesByUploader 获取上传者的使用记录
func (r *mediaUsageRepository) GetUsagesByUploader(uploaderID int, page, size int) ([]*model.MediaUsage, int64, error) {
	var usages []*model.MediaUsage
	var total int64

	query := r.db.Model(&model.MediaUsage{}).Where("uploader_id = ?", uploaderID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取上传者使用记录总数失败(上传者ID: %d): %v", uploaderID, err)
		return nil, 0, fmt.Errorf("获取上传者使用记录总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Media").Preload("Media.Uploader").
		Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&usages).Error; err != nil {
		logger.Errorf("获取上传者使用记录列表失败(上传者ID: %d): %v", uploaderID, err)
		return nil, 0, fmt.Errorf("获取上传者使用记录列表失败: %w", err)
	}

	return usages, total, nil
}

// GetUsagesByType 根据使用类型获取使用记录
func (r *mediaUsageRepository) GetUsagesByType(usageType string, page, size int) ([]*model.MediaUsage, int64, error) {
	var usages []*model.MediaUsage
	var total int64

	query := r.db.Model(&model.MediaUsage{}).Where("usage_type = ?", usageType)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取类型使用记录总数失败(类型: %s): %v", usageType, err)
		return nil, 0, fmt.Errorf("获取类型使用记录总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Media").Preload("Media.Uploader").
		Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&usages).Error; err != nil {
		logger.Errorf("获取类型使用记录列表失败(类型: %s): %v", usageType, err)
		return nil, 0, fmt.Errorf("获取类型使用记录列表失败: %w", err)
	}

	return usages, total, nil
}

// BatchCreate 批量创建使用记录
func (r *mediaUsageRepository) BatchCreate(usages []*model.MediaUsage) error {
	if len(usages) == 0 {
		return nil
	}

	if err := r.db.CreateInBatches(usages, 100).Error; err != nil {
		logger.Errorf("批量创建媒体使用记录失败: %v", err)
		return fmt.Errorf("批量创建媒体使用记录失败: %w", err)
	}

	logger.Infof("批量创建媒体使用记录成功 (数量: %d)", len(usages))
	return nil
}

// DeleteByTarget 根据目标删除使用记录
func (r *mediaUsageRepository) DeleteByTarget(usageType string, targetID int) error {
	if err := r.db.Where("usage_type = ? AND target_id = ?", usageType, targetID).
		Delete(&model.MediaUsage{}).Error; err != nil {
		logger.Errorf("根据目标删除使用记录失败(类型: %s, 目标ID: %d): %v", usageType, targetID, err)
		return fmt.Errorf("根据目标删除使用记录失败: %w", err)
	}

	logger.Infof("根据目标删除使用记录成功 (类型: %s, 目标ID: %d)", usageType, targetID)
	return nil
}

// GetMediaUsageHistory 获取媒体文件使用历史
func (r *mediaUsageRepository) GetMediaUsageHistory(mediaID int, page, size int) ([]*model.MediaUsage, int64, error) {
	var usages []*model.MediaUsage
	var total int64

	query := r.db.Model(&model.MediaUsage{}).Where("media_id = ?", mediaID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取媒体使用历史总数失败(媒体ID: %d): %v", mediaID, err)
		return nil, 0, fmt.Errorf("获取媒体使用历史总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&usages).Error; err != nil {
		logger.Errorf("获取媒体使用历史列表失败(媒体ID: %d): %v", mediaID, err)
		return nil, 0, fmt.Errorf("获取媒体使用历史列表失败: %w", err)
	}

	return usages, total, nil
}