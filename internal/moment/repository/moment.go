package repository

import (
	"fmt"
	"time"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/gorm"
)

// MomentRepository 随念仓库接口
type MomentRepository interface {
	Create(moment *model.Post) error
	GetByID(id int) (*model.Post, error)
	Update(moment *model.Post) error
	Delete(id int) error
	List(page, size int, filters *MomentFilters) ([]*model.Post, int64, error)
	ListByAuthor(authorID int, page, size int) ([]*model.Post, int64, error)
	ListPublished(page, size int) ([]*model.Post, int64, error)
	Search(keyword string, page, size int) ([]*model.Post, int64, error)
	GetByIDWithRelations(id int) (*model.Post, error)
	UpdateViewCount(id int) error
	UpdateLikeCount(id int, increment bool) error
	UpdateCommentCount(id int, increment bool) error
	Publish(id int) error
	Unpublish(id int) error
	GetRecentMoments(limit int) ([]*model.Post, error)
	GetTrendingMoments(limit int) ([]*model.Post, error)
}

// MomentFilters 随念查询过滤器
type MomentFilters struct {
	Status     string    `json:"status"`
	CategoryID *int      `json:"category_id"`
	TagIDs     []int     `json:"tag_ids"`
	AuthorID   *int      `json:"author_id"`
	StartDate  *time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date"`
	Keyword    string    `json:"keyword"`
}

// momentRepository 随念仓库实现
type momentRepository struct {
	db *gorm.DB
}

// NewMomentRepository 创建随念仓库实例
func NewMomentRepository() MomentRepository {
	return &momentRepository{
		db: database.GetDB(),
	}
}

// Create 创建随念
func (r *momentRepository) Create(moment *model.Post) error {
	// 确保类型为moment
	moment.Type = "moment"

	if err := r.db.Create(moment).Error; err != nil {
		logger.Errorf("创建随念失败: %v", err)
		return fmt.Errorf("创建随念失败: %w", err)
	}

	logger.Infof("随念创建成功: %s (ID: %d)", moment.Title, moment.ID)
	return nil
}

// GetByID 根据ID获取随念
func (r *momentRepository) GetByID(id int) (*model.Post, error) {
	var moment model.Post
	err := r.db.Where("type = ?", "moment").First(&moment, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("随念不存在")
		}
		logger.Errorf("获取随念失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取随念失败: %w", err)
	}

	return &moment, nil
}

// Update 更新随念
func (r *momentRepository) Update(moment *model.Post) error {
	// 确保类型为moment
	moment.Type = "moment"

	if err := r.db.Save(moment).Error; err != nil {
		logger.Errorf("更新随念失败(ID: %d): %v", moment.ID, err)
		return fmt.Errorf("更新随念失败: %w", err)
	}

	logger.Infof("随念更新成功: %s (ID: %d)", moment.Title, moment.ID)
	return nil
}

// Delete 删除随念
func (r *momentRepository) Delete(id int) error {
	if err := r.db.Where("type = ?", "moment").Delete(&model.Post{}, id).Error; err != nil {
		logger.Errorf("删除随念失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除随念失败: %w", err)
	}

	logger.Infof("随念删除成功 (ID: %d)", id)
	return nil
}

// List 获取随念列表
func (r *momentRepository) List(page, size int, filters *MomentFilters) ([]*model.Post, int64, error) {
	var moments []*model.Post
	var total int64

	query := r.buildMomentQuery(filters)

	// 获取总数
	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		logger.Errorf("获取随念总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取随念总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").
		Offset(offset).Limit(size).
		Order("created_at DESC").
		Find(&moments).Error; err != nil {
		logger.Errorf("获取随念列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取随念列表失败: %w", err)
	}

	return moments, total, nil
}

// ListByAuthor 根据作者获取随念列表
func (r *momentRepository) ListByAuthor(authorID int, page, size int) ([]*model.Post, int64, error) {
	var moments []*model.Post
	var total int64

	query := r.db.Where("type = ? AND author_id = ?", "moment", authorID)

	// 获取总数
	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		logger.Errorf("获取作者随念总数失败(作者ID: %d): %v", authorID, err)
		return nil, 0, fmt.Errorf("获取作者随念总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").
		Offset(offset).Limit(size).
		Order("created_at DESC").
		Find(&moments).Error; err != nil {
		logger.Errorf("获取作者随念列表失败(作者ID: %d): %v", authorID, err)
		return nil, 0, fmt.Errorf("获取作者随念列表失败: %w", err)
	}

	return moments, total, nil
}

// ListPublished 获取已发布随念列表
func (r *momentRepository) ListPublished(page, size int) ([]*model.Post, int64, error) {
	var moments []*model.Post
	var total int64

	query := r.db.Where("type = ? AND status = ?", "moment", "published")

	// 获取总数
	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		logger.Errorf("获取已发布随念总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取已发布随念总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").Preload("Tags").
		Offset(offset).Limit(size).
		Order("published_at DESC").
		Find(&moments).Error; err != nil {
		logger.Errorf("获取已发布随念列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取已发布随念列表失败: %w", err)
	}

	return moments, total, nil
}

// Search 搜索随念
func (r *momentRepository) Search(keyword string, page, size int) ([]*model.Post, int64, error) {
	var moments []*model.Post
	var total int64

	if keyword == "" {
		return []*model.Post{}, 0, nil
	}

	searchPattern := "%" + keyword + "%"

	// 构建搜索查询
	query := r.db.Model(&model.Post{}).
		Where("type = ? AND status = ? AND (title LIKE ? OR content LIKE ?)", "moment", "published", searchPattern, searchPattern)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("搜索随念总数失败: %v", err)
		return nil, 0, fmt.Errorf("搜索随念总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").Preload("Tags").
		Offset(offset).Limit(size).
		Order("published_at DESC").
		Find(&moments).Error; err != nil {
		logger.Errorf("搜索随念失败: %v", err)
		return nil, 0, fmt.Errorf("搜索随念失败: %w", err)
	}

	return moments, total, nil
}

// GetByIDWithRelations 根据ID获取随念（包含关联数据）
func (r *momentRepository) GetByIDWithRelations(id int) (*model.Post, error) {
	var moment model.Post
	err := r.db.Where("type = ?", "moment").Preload("Author").Preload("Category").Preload("Tags").
		First(&moment, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("随念不存在")
		}
		logger.Errorf("获取随念详情失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取随念详情失败: %w", err)
	}

	return &moment, nil
}

// UpdateViewCount 更新随念浏览量
func (r *momentRepository) UpdateViewCount(id int) error {
	if err := r.db.Model(&model.Post{}).Where("id = ? AND type = ?", id, "moment").
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error; err != nil {
		logger.Errorf("更新随念浏览量失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新随念浏览量失败: %w", err)
	}

	return nil
}

// UpdateLikeCount 更新随念点赞数
func (r *momentRepository) UpdateLikeCount(id int, increment bool) error {
	var expr string
	if increment {
		expr = "like_count + ?"
	} else {
		expr = "CASE WHEN like_count > 0 THEN like_count - ? ELSE 0 END"
	}

	if err := r.db.Model(&model.Post{}).Where("id = ? AND type = ?", id, "moment").
		UpdateColumn("like_count", gorm.Expr(expr, 1)).Error; err != nil {
		logger.Errorf("更新随念点赞数失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新随念点赞数失败: %w", err)
	}

	return nil
}

// UpdateCommentCount 更新随念评论数
func (r *momentRepository) UpdateCommentCount(id int, increment bool) error {
	var expr string
	if increment {
		expr = "comment_count + ?"
	} else {
		expr = "CASE WHEN comment_count > 0 THEN comment_count - ? ELSE 0 END"
	}

	if err := r.db.Model(&model.Post{}).Where("id = ? AND type = ?", id, "moment").
		UpdateColumn("comment_count", gorm.Expr(expr, 1)).Error; err != nil {
		logger.Errorf("更新随念评论数失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新随念评论数失败: %w", err)
	}

	return nil
}

// Publish 发布随念
func (r *momentRepository) Publish(id int) error {
	now := time.Now()
	if err := r.db.Model(&model.Post{}).Where("id = ? AND type = ?", id, "moment").
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": &now,
		}).Error; err != nil {
		logger.Errorf("发布随念失败(ID: %d): %v", id, err)
		return fmt.Errorf("发布随念失败: %w", err)
	}

	logger.Infof("随念发布成功 (ID: %d)", id)
	return nil
}

// Unpublish 取消发布随念
func (r *momentRepository) Unpublish(id int) error {
	if err := r.db.Model(&model.Post{}).Where("id = ? AND type = ?", id, "moment").
		Update("status", "draft").Error; err != nil {
		logger.Errorf("取消发布随念失败(ID: %d): %v", id, err)
		return fmt.Errorf("取消发布随念失败: %w", err)
	}

	logger.Infof("随念取消发布成功 (ID: %d)", id)
	return nil
}

// GetRecentMoments 获取最新随念
func (r *momentRepository) GetRecentMoments(limit int) ([]*model.Post, error) {
	var moments []*model.Post
	if err := r.db.Where("type = ? AND status = ?", "moment", "published").
		Preload("Author").
		Order("published_at DESC").
		Limit(limit).
		Find(&moments).Error; err != nil {
		logger.Errorf("获取最新随念失败: %v", err)
		return nil, fmt.Errorf("获取最新随念失败: %w", err)
	}

	return moments, nil
}

// GetTrendingMoments 获取热门随念
func (r *momentRepository) GetTrendingMoments(limit int) ([]*model.Post, error) {
	var moments []*model.Post

	// 查询最近7天内点赞数和评论数最多的随念
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	if err := r.db.Where("type = ? AND status = ? AND published_at >= ?", "moment", "published", sevenDaysAgo).
		Preload("Author").
		Order("(like_count * 3 + comment_count * 2 + view_count) DESC, published_at DESC").
		Limit(limit).
		Find(&moments).Error; err != nil {
		logger.Errorf("获取热门随念失败: %v", err)
		return nil, fmt.Errorf("获取热门随念失败: %w", err)
	}

	return moments, nil
}

// buildMomentQuery 构建随念查询条件
func (r *momentRepository) buildMomentQuery(filters *MomentFilters) *gorm.DB {
	query := r.db.Model(&model.Post{}).Where("type = ?", "moment")

	if filters == nil {
		return query
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.CategoryID != nil {
		query = query.Where("category_id = ?", *filters.CategoryID)
	}

	if filters.AuthorID != nil {
		query = query.Where("author_id = ?", *filters.AuthorID)
	}

	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", *filters.StartDate)
	}

	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", *filters.EndDate)
	}

	if filters.Keyword != "" {
		searchPattern := "%" + filters.Keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", searchPattern, searchPattern)
	}

	// 标签过滤
	if len(filters.TagIDs) > 0 {
		query = query.Joins("INNER JOIN post_tags ON posts.id = post_tags.post_id").
			Where("post_tags.tag_id IN ?", filters.TagIDs).
			Group("posts.id")
	}

	return query
}

// GetBySlug 根据slug获取随念
func (r *momentRepository) GetBySlug(slug string) (*model.Post, error) {
	var moment model.Post
	err := r.db.Where("type = ? AND slug = ?", "moment", slug).First(&moment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("随念不存在")
		}
		logger.Errorf("获取随念失败(slug: %s): %v", slug, err)
		return nil, fmt.Errorf("获取随念失败: %w", err)
	}

	return &moment, nil
}

// checkSlugExists 检查slug是否存在
func (r *momentRepository) checkSlugExists(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Post{}).Where("type = ? AND slug = ?", "moment", slug).Count(&count).Error
	return count > 0, err
}