package repository

import (
	"fmt"
	"time"

	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/media/model"
	"gorm.io/gorm"
)

// MediaRepository 媒体文件仓库接口
type MediaRepository interface {
	Create(media *model.MediaFile) error
	GetByID(id int) (*model.MediaFile, error)
	GetByHash(hash string) (*model.MediaFile, error)
	Update(media *model.MediaFile) error
	Delete(id int) error
	SoftDelete(id int) error
	Restore(id int) error
	List(page, size int, filters *MediaFilters) ([]*model.MediaFile, int64, error)
	ListByUploader(uploaderID int, page, size int) ([]*model.MediaFile, int64, error)
	ListPublic(page, size int) ([]*model.MediaFile, int64, error)
	Search(keyword string, page, size int) ([]*model.MediaFile, int64, error)
	GetStats() (*MediaStats, error)
	GetUploaderStats(uploaderID int) (*MediaStats, error)
	GetByFileType(fileType string, page, size int) ([]*model.MediaFile, int64, error)
	GetRecent(limit int) ([]*model.MediaFile, error)
}

// MediaUsageRepository 媒体使用记录仓库接口
type MediaUsageRepository interface {
	Create(usage *model.MediaUsage) error
	GetByID(id int) (*model.MediaUsage, error)
	GetByMediaID(mediaID int) ([]*model.MediaUsage, error)
	GetByTarget(usageType string, targetID int) ([]*model.MediaUsage, error)
	Delete(id int) error
	DeleteByMediaID(mediaID int) error
	List(page, size int) ([]*model.MediaUsage, int64, error)
	GetUsageStats() (*UsageStats, error)
}

// MediaFilters 媒体文件查询过滤器
type MediaFilters struct {
	FileType   string    `json:"file_type"`
	UploaderID *int      `json:"uploader_id"`
	Status     string    `json:"status"`
	IsPublic   *bool     `json:"is_public"`
	MinSize    *int64    `json:"min_size"`
	MaxSize    *int64    `json:"max_size"`
	StartDate  *time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date"`
	Keyword    string    `json:"keyword"`
}

// MediaStats 媒体文件统计
type MediaStats struct {
	TotalCount   int64 `json:"total_count"`
	TotalSize    int64 `json:"total_size"`
	ImageCount   int64 `json:"image_count"`
	VideoCount   int64 `json:"video_count"`
	DocumentCount int64 `json:"document_count"`
	OtherCount   int64 `json:"other_count"`
}

// UsageStats 使用统计
type UsageStats struct {
	TotalUsages int64                    `json:"total_usages"`
	UsageByType map[string]int64        `json:"usage_by_type"`
	TopMedia    []model.MediaFile       `json:"top_media"`
	RecentUsages []*model.MediaUsage    `json:"recent_usages"`
}

// mediaRepository 媒体文件仓库实现
type mediaRepository struct {
	db *gorm.DB
}

// NewMediaRepository 创建媒体文件仓库实例
func NewMediaRepository() MediaRepository {
	return &mediaRepository{
		db: database.GetDB(),
	}
}

// Create 创建媒体文件
func (r *mediaRepository) Create(media *model.MediaFile) error {
	if err := r.db.Create(media).Error; err != nil {
		logger.Errorf("创建媒体文件失败: %v", err)
		return fmt.Errorf("创建媒体文件失败: %w", err)
	}

	logger.Infof("媒体文件创建成功 (ID: %d, 文件名: %s)", media.ID, media.Filename)
	return nil
}

// GetByID 根据ID获取媒体文件
func (r *mediaRepository) GetByID(id int) (*model.MediaFile, error) {
	var media model.MediaFile
	err := r.db.Preload("Uploader").First(&media, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("媒体文件不存在")
		}
		logger.Errorf("获取媒体文件失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取媒体文件失败: %w", err)
	}

	return &media, nil
}

// GetByHash 根据哈希值获取媒体文件
func (r *mediaRepository) GetByHash(hash string) (*model.MediaFile, error) {
	var media model.MediaFile
	err := r.db.Where("hash = ?", hash).First(&media).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 返回nil表示未找到，这是正常情况
		}
		logger.Errorf("根据哈希获取媒体文件失败(Hash: %s): %v", hash, err)
		return nil, fmt.Errorf("根据哈希获取媒体文件失败: %w", err)
	}

	return &media, nil
}

// Update 更新媒体文件
func (r *mediaRepository) Update(media *model.MediaFile) error {
	if err := r.db.Save(media).Error; err != nil {
		logger.Errorf("更新媒体文件失败(ID: %d): %v", media.ID, err)
		return fmt.Errorf("更新媒体文件失败: %w", err)
	}

	logger.Infof("媒体文件更新成功 (ID: %d)", media.ID)
	return nil
}

// Delete 删除媒体文件
func (r *mediaRepository) Delete(id int) error {
	if err := r.db.Delete(&model.MediaFile{}, id).Error; err != nil {
		logger.Errorf("删除媒体文件失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除媒体文件失败: %w", err)
	}

	logger.Infof("媒体文件删除成功 (ID: %d)", id)
	return nil
}

// SoftDelete 软删除媒体文件
func (r *mediaRepository) SoftDelete(id int) error {
	if err := r.db.Model(&model.MediaFile{}).Where("id = ?", id).
		Update("status", "deleted").Error; err != nil {
		logger.Errorf("软删除媒体文件失败(ID: %d): %v", id, err)
		return fmt.Errorf("软删除媒体文件失败: %w", err)
	}

	logger.Infof("媒体文件软删除成功 (ID: %d)", id)
	return nil
}

// Restore 恢复删除的媒体文件
func (r *mediaRepository) Restore(id int) error {
	if err := r.db.Model(&model.MediaFile{}).Where("id = ?", id).
		Update("status", "active").Error; err != nil {
		logger.Errorf("恢复媒体文件失败(ID: %d): %v", id, err)
		return fmt.Errorf("恢复媒体文件失败: %w", err)
	}

	logger.Infof("媒体文件恢复成功 (ID: %d)", id)
	return nil
}

// List 获取媒体文件列表
func (r *mediaRepository) List(page, size int, filters *MediaFilters) ([]*model.MediaFile, int64, error) {
	var medias []*model.MediaFile
	var total int64

	query := r.buildMediaQuery(filters)

	// 获取总数
	if err := query.Model(&model.MediaFile{}).Count(&total).Error; err != nil {
		logger.Errorf("获取媒体文件总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取媒体文件总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&medias).Error; err != nil {
		logger.Errorf("获取媒体文件列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取媒体文件列表失败: %w", err)
	}

	return medias, total, nil
}

// ListByUploader 根据上传者获取媒体文件列表
func (r *mediaRepository) ListByUploader(uploaderID int, page, size int) ([]*model.MediaFile, int64, error) {
	var medias []*model.MediaFile
	var total int64

	query := r.db.Model(&model.MediaFile{}).Where("uploader_id = ?", uploaderID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取上传者媒体文件总数失败(上传者ID: %d): %v", uploaderID, err)
		return nil, 0, fmt.Errorf("获取上传者媒体文件总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Where("status = ?", "active").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&medias).Error; err != nil {
		logger.Errorf("获取上传者媒体文件列表失败(上传者ID: %d): %v", uploaderID, err)
		return nil, 0, fmt.Errorf("获取上传者媒体文件列表失败: %w", err)
	}

	return medias, total, nil
}

// ListPublic 获取公开媒体文件列表
func (r *mediaRepository) ListPublic(page, size int) ([]*model.MediaFile, int64, error) {
	var medias []*model.MediaFile
	var total int64

	query := r.db.Model(&model.MediaFile{}).Where("is_public = ? AND status = ?", true, "active")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取公开媒体文件总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取公开媒体文件总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&medias).Error; err != nil {
		logger.Errorf("获取公开媒体文件列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取公开媒体文件列表失败: %w", err)
	}

	return medias, total, nil
}

// Search 搜索媒体文件
func (r *mediaRepository) Search(keyword string, page, size int) ([]*model.MediaFile, int64, error) {
	var medias []*model.MediaFile
	var total int64

	if keyword == "" {
		return []*model.MediaFile{}, 0, nil
	}

	searchPattern := "%" + keyword + "%"

	// 构建搜索查询
	query := r.db.Model(&model.MediaFile{}).
		Where("(filename LIKE ? OR alt LIKE ? OR caption LIKE ?) AND status = ?",
			searchPattern, searchPattern, searchPattern, "active")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("搜索媒体文件总数失败: %v", err)
		return nil, 0, fmt.Errorf("搜索媒体文件总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&medias).Error; err != nil {
		logger.Errorf("搜索媒体文件失败: %v", err)
		return nil, 0, fmt.Errorf("搜索媒体文件失败: %w", err)
	}

	return medias, total, nil
}

// GetStats 获取媒体文件统计
func (r *mediaRepository) GetStats() (*MediaStats, error) {
	stats := &MediaStats{}

	// 总数和总大小
	if err := r.db.Model(&model.MediaFile{}).
		Where("status = ?", "active").
		Select("COUNT(*) as total_count, COALESCE(SUM(file_size), 0) as total_size").
		Scan(stats).Error; err != nil {
		logger.Errorf("获取媒体文件基本统计失败: %v", err)
		return nil, fmt.Errorf("获取媒体文件基本统计失败: %w", err)
	}

	// 按类型统计
	typeStats := []struct {
		FileType string `json:"file_type"`
		Count    int64  `json:"count"`
	}{}

	if err := r.db.Model(&model.MediaFile{}).
		Where("status = ?", "active").
		Select("file_type, COUNT(*) as count").
		Group("file_type").
		Scan(&typeStats).Error; err != nil {
		logger.Errorf("获取媒体文件类型统计失败: %v", err)
		return nil, fmt.Errorf("获取媒体文件类型统计失败: %w", err)
	}

	for _, ts := range typeStats {
		switch ts.FileType {
		case "image":
			stats.ImageCount = ts.Count
		case "video":
			stats.VideoCount = ts.Count
		case "document":
			stats.DocumentCount = ts.Count
		default:
			stats.OtherCount += ts.Count
		}
	}

	return stats, nil
}

// GetUploaderStats 获取上传者统计
func (r *mediaRepository) GetUploaderStats(uploaderID int) (*MediaStats, error) {
	stats := &MediaStats{}

	// 总数和总大小
	if err := r.db.Model(&model.MediaFile{}).
		Where("uploader_id = ? AND status = ?", uploaderID, "active").
		Select("COUNT(*) as total_count, COALESCE(SUM(file_size), 0) as total_size").
		Scan(stats).Error; err != nil {
		logger.Errorf("获取上传者媒体文件统计失败(上传者ID: %d): %v", uploaderID, err)
		return nil, fmt.Errorf("获取上传者媒体文件统计失败: %w", err)
	}

	// 按类型统计
	typeStats := []struct {
		FileType string `json:"file_type"`
		Count    int64  `json:"count"`
	}{}

	if err := r.db.Model(&model.MediaFile{}).
		Where("uploader_id = ? AND status = ?", uploaderID, "active").
		Select("file_type, COUNT(*) as count").
		Group("file_type").
		Scan(&typeStats).Error; err != nil {
		logger.Errorf("获取上传者媒体文件类型统计失败(上传者ID: %d): %v", uploaderID, err)
		return nil, fmt.Errorf("获取上传者媒体文件类型统计失败: %w", err)
	}

	for _, ts := range typeStats {
		switch ts.FileType {
		case "image":
			stats.ImageCount = ts.Count
		case "video":
			stats.VideoCount = ts.Count
		case "document":
			stats.DocumentCount = ts.Count
		default:
			stats.OtherCount += ts.Count
		}
	}

	return stats, nil
}

// GetByFileType 根据文件类型获取媒体文件
func (r *mediaRepository) GetByFileType(fileType string, page, size int) ([]*model.MediaFile, int64, error) {
	var medias []*model.MediaFile
	var total int64

	query := r.db.Model(&model.MediaFile{}).
		Where("file_type = ? AND status = ?", fileType, "active")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取类型媒体文件总数失败(类型: %s): %v", fileType, err)
		return nil, 0, fmt.Errorf("获取类型媒体文件总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Uploader").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&medias).Error; err != nil {
		logger.Errorf("获取类型媒体文件列表失败(类型: %s): %v", fileType, err)
		return nil, 0, fmt.Errorf("获取类型媒体文件列表失败: %w", err)
	}

	return medias, total, nil
}

// GetRecent 获取最新的媒体文件
func (r *mediaRepository) GetRecent(limit int) ([]*model.MediaFile, error) {
	var medias []*model.MediaFile

	if err := r.db.Where("status = ? AND is_public = ?", "active", true).
		Preload("Uploader").
		Order("created_at DESC").
		Limit(limit).
		Find(&medias).Error; err != nil {
		logger.Errorf("获取最新媒体文件失败: %v", err)
		return nil, fmt.Errorf("获取最新媒体文件失败: %w", err)
	}

	return medias, nil
}

// buildMediaQuery 构建媒体文件查询条件
func (r *mediaRepository) buildMediaQuery(filters *MediaFilters) *gorm.DB {
	query := r.db.Model(&model.MediaFile{})

	if filters == nil {
		return query
	}

	if filters.FileType != "" {
		query = query.Where("file_type = ?", filters.FileType)
	}

	if filters.UploaderID != nil {
		query = query.Where("uploader_id = ?", *filters.UploaderID)
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.IsPublic != nil {
		query = query.Where("is_public = ?", *filters.IsPublic)
	}

	if filters.MinSize != nil {
		query = query.Where("file_size >= ?", *filters.MinSize)
	}

	if filters.MaxSize != nil {
		query = query.Where("file_size <= ?", *filters.MaxSize)
	}

	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", *filters.StartDate)
	}

	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", *filters.EndDate)
	}

	if filters.Keyword != "" {
		searchPattern := "%" + filters.Keyword + "%"
		query = query.Where("filename LIKE ? OR alt LIKE ? OR caption LIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	return query
}