package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/gorm"
)

// PostRepository 文章仓库接口
type PostRepository interface {
	Create(post *model.Post) error
	GetByID(id int) (*model.Post, error)
	GetBySlug(slug string) (*model.Post, error)
	Update(post *model.Post) error
	Delete(id int) error
	List(page, size int, filters *PostFilters) ([]*model.Post, int64, error)
	ListByAuthor(authorID int, page, size int, filters *PostFilters) ([]*model.Post, int64, error)
	ListPublished(page, size int, filters *PostFilters) ([]*model.Post, int64, error)
	Search(keyword string, page, size int) ([]*model.Post, int64, error)
	GetByIDWithRelations(id int) (*model.Post, error)
	UpdateViewCount(id int) error
	UpdateLikeCount(id int, increment bool) error
	UpdateCommentCount(id int, increment bool) error
	Publish(id int) error
	Unpublish(id int) error
	SetTop(id int, isTop bool) error
	GetRecentPosts(limit int) ([]*model.Post, error)
	GetRelatedPosts(postID int, limit int) ([]*model.Post, error)
}

// PostFilters 文章查询过滤器
type PostFilters struct {
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	CategoryID *int      `json:"category_id"`
	TagIDs     []int     `json:"tag_ids"`
	AuthorID   *int      `json:"author_id"`
	StartDate  *time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date"`
	Keyword    string    `json:"keyword"`
}

// postRepository 文章仓库实现
type postRepository struct {
	db *gorm.DB
}

// NewPostRepository 创建文章仓库实例
func NewPostRepository() PostRepository {
	return &postRepository{
		db: database.GetDB(),
	}
}

// Create 创建文章
func (r *postRepository) Create(post *model.Post) error {
	if err := r.db.Create(post).Error; err != nil {
		logger.Errorf("创建文章失败: %v", err)
		return fmt.Errorf("创建文章失败: %w", err)
	}

	logger.Infof("文章创建成功: %s (ID: %d)", post.Title, post.ID)
	return nil
}

// GetByID 根据ID获取文章
func (r *postRepository) GetByID(id int) (*model.Post, error) {
	var post model.Post
	err := r.db.First(&post, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文章不存在")
		}
		logger.Errorf("获取文章失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取文章失败: %w", err)
	}

	return &post, nil
}

// GetBySlug 根据slug获取文章
func (r *postRepository) GetBySlug(slug string) (*model.Post, error) {
	var post model.Post
	err := r.db.Where("slug = ?", slug).First(&post).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文章不存在")
		}
		logger.Errorf("获取文章失败(slug: %s): %v", slug, err)
		return nil, fmt.Errorf("获取文章失败: %w", err)
	}

	return &post, nil
}

// Update 更新文章
func (r *postRepository) Update(post *model.Post) error {
	if err := r.db.Save(post).Error; err != nil {
		logger.Errorf("更新文章失败(ID: %d): %v", post.ID, err)
		return fmt.Errorf("更新文章失败: %w", err)
	}

	logger.Infof("文章更新成功: %s (ID: %d)", post.Title, post.ID)
	return nil
}

// Delete 删除文章
func (r *postRepository) Delete(id int) error {
	if err := r.db.Delete(&model.Post{}, id).Error; err != nil {
		logger.Errorf("删除文章失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除文章失败: %w", err)
	}

	logger.Infof("文章删除成功 (ID: %d)", id)
	return nil
}

// List 获取文章列表
func (r *postRepository) List(page, size int, filters *PostFilters) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var total int64

	query := r.buildQuery(filters)

	// 获取总数
	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		logger.Errorf("获取文章总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取文章总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").
		Offset(offset).Limit(size).
		Order("created_at DESC").
		Find(&posts).Error; err != nil {
		logger.Errorf("获取文章列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取文章列表失败: %w", err)
	}

	return posts, total, nil
}

// ListByAuthor 根据作者获取文章列表
func (r *postRepository) ListByAuthor(authorID int, page, size int, filters *PostFilters) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var total int64

	query := r.buildQuery(filters)
	query = query.Where("author_id = ?", authorID)

	// 获取总数
	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		logger.Errorf("获取作者文章总数失败(作者ID: %d): %v", authorID, err)
		return nil, 0, fmt.Errorf("获取作者文章总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").
		Offset(offset).Limit(size).
		Order("created_at DESC").
		Find(&posts).Error; err != nil {
		logger.Errorf("获取作者文章列表失败(作者ID: %d): %v", authorID, err)
		return nil, 0, fmt.Errorf("获取作者文章列表失败: %w", err)
	}

	return posts, total, nil
}

// ListPublished 获取已发布文章列表
func (r *postRepository) ListPublished(page, size int, filters *PostFilters) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var total int64

	query := r.buildQuery(filters)
	query = query.Where("status = ?", "published")

	// 获取总数
	if err := query.Model(&model.Post{}).Count(&total).Error; err != nil {
		logger.Errorf("获取已发布文章总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取已发布文章总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").Preload("Tags").
		Offset(offset).Limit(size).
		Order("is_top DESC, published_at DESC").
		Find(&posts).Error; err != nil {
		logger.Errorf("获取已发布文章列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取已发布文章列表失败: %w", err)
	}

	return posts, total, nil
}

// Search 搜索文章
func (r *postRepository) Search(keyword string, page, size int) ([]*model.Post, int64, error) {
	var posts []*model.Post
	var total int64

	if strings.TrimSpace(keyword) == "" {
		return []*model.Post{}, 0, nil
	}

	searchPattern := "%" + keyword + "%"

	// 构建搜索查询
	query := r.db.Model(&model.Post{}).
		Where("status = ? AND (title LIKE ? OR content LIKE ?)", "published", searchPattern, searchPattern)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("搜索文章总数失败: %v", err)
		return nil, 0, fmt.Errorf("搜索文章总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Category").Preload("Tags").
		Offset(offset).Limit(size).
		Order("published_at DESC").
		Find(&posts).Error; err != nil {
		logger.Errorf("搜索文章失败: %v", err)
		return nil, 0, fmt.Errorf("搜索文章失败: %w", err)
	}

	return posts, total, nil
}

// GetByIDWithRelations 根据ID获取文章（包含关联数据）
func (r *postRepository) GetByIDWithRelations(id int) (*model.Post, error) {
	var post model.Post
	err := r.db.Preload("Author").Preload("Category").Preload("Tags").
		First(&post, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文章不存在")
		}
		logger.Errorf("获取文章详情失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取文章详情失败: %w", err)
	}

	return &post, nil
}

// UpdateViewCount 更新文章浏览量
func (r *postRepository) UpdateViewCount(id int) error {
	if err := r.db.Model(&model.Post{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error; err != nil {
		logger.Errorf("更新文章浏览量失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新文章浏览量失败: %w", err)
	}

	return nil
}

// UpdateLikeCount 更新文章点赞数
func (r *postRepository) UpdateLikeCount(id int, increment bool) error {
	var expr string
	if increment {
		expr = "like_count + ?"
	} else {
		expr = "CASE WHEN like_count > 0 THEN like_count - ? ELSE 0 END"
	}

	if err := r.db.Model(&model.Post{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr(expr, 1)).Error; err != nil {
		logger.Errorf("更新文章点赞数失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新文章点赞数失败: %w", err)
	}

	return nil
}

// UpdateCommentCount 更新文章评论数
func (r *postRepository) UpdateCommentCount(id int, increment bool) error {
	var expr string
	if increment {
		expr = "comment_count + ?"
	} else {
		expr = "CASE WHEN comment_count > 0 THEN comment_count - ? ELSE 0 END"
	}

	if err := r.db.Model(&model.Post{}).Where("id = ?", id).
		UpdateColumn("comment_count", gorm.Expr(expr, 1)).Error; err != nil {
		logger.Errorf("更新文章评论数失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新文章评论数失败: %w", err)
	}

	return nil
}

// Publish 发布文章
func (r *postRepository) Publish(id int) error {
	now := time.Now()
	if err := r.db.Model(&model.Post{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": &now,
		}).Error; err != nil {
		logger.Errorf("发布文章失败(ID: %d): %v", id, err)
		return fmt.Errorf("发布文章失败: %w", err)
	}

	logger.Infof("文章发布成功 (ID: %d)", id)
	return nil
}

// Unpublish 取消发布文章
func (r *postRepository) Unpublish(id int) error {
	if err := r.db.Model(&model.Post{}).Where("id = ?", id).
		Update("status", "draft").Error; err != nil {
		logger.Errorf("取消发布文章失败(ID: %d): %v", id, err)
		return fmt.Errorf("取消发布文章失败: %w", err)
	}

	logger.Infof("文章取消发布成功 (ID: %d)", id)
	return nil
}

// SetTop 设置文章置顶
func (r *postRepository) SetTop(id int, isTop bool) error {
	if err := r.db.Model(&model.Post{}).Where("id = ?", id).
		Update("is_top", isTop).Error; err != nil {
		logger.Errorf("设置文章置顶失败(ID: %d): %v", id, err)
		return fmt.Errorf("设置文章置顶失败: %w", err)
	}

	logger.Infof("文章置顶设置成功 (ID: %d, 置顶: %v)", id, isTop)
	return nil
}

// GetRecentPosts 获取最新文章
func (r *postRepository) GetRecentPosts(limit int) ([]*model.Post, error) {
	var posts []*model.Post
	if err := r.db.Where("status = ?", "published").
		Preload("Author").
		Order("published_at DESC").
		Limit(limit).
		Find(&posts).Error; err != nil {
		logger.Errorf("获取最新文章失败: %v", err)
		return nil, fmt.Errorf("获取最新文章失败: %w", err)
	}

	return posts, nil
}

// GetRelatedPosts 获取相关文章
func (r *postRepository) GetRelatedPosts(postID int, limit int) ([]*model.Post, error) {
	var post model.Post
	if err := r.db.Preload("Tags").First(&post, postID).Error; err != nil {
		return nil, fmt.Errorf("获取原文章失败: %w", err)
	}

	var posts []*model.Post

	// 如果有标签，根据标签查找相关文章
	if len(post.Tags) > 0 {
		tagIDs := make([]int, len(post.Tags))
		for i, tag := range post.Tags {
			tagIDs[i] = tag.ID
		}

		if err := r.db.Table("posts").
			Select("posts.*").
			Joins("INNER JOIN post_tags ON posts.id = post_tags.post_id").
			Where("post_tags.tag_id IN ? AND posts.id != ? AND posts.status = ?", tagIDs, postID, "published").
			Preload("Author").
			Group("posts.id").
			Order("posts.published_at DESC").
			Limit(limit).
			Find(&posts).Error; err != nil {
			logger.Errorf("根据标签获取相关文章失败: %v", err)
			return nil, fmt.Errorf("获取相关文章失败: %w", err)
		}
	}

	// 如果根据标签找到的文章不够，根据分类补充
	if len(posts) < limit && post.CategoryID != nil {
		remaining := limit - len(posts)
		var categoryPosts []*model.Post

		if err := r.db.Where("category_id = ? AND id != ? AND status = ?", post.CategoryID, postID, "published").
			Preload("Author").
			Order("published_at DESC").
			Limit(remaining).
			Find(&categoryPosts).Error; err != nil {
			logger.Errorf("根据分类获取相关文章失败: %v", err)
		} else {
			posts = append(posts, categoryPosts...)
		}
	}

	return posts, nil
}

// buildQuery 构建查询条件
func (r *postRepository) buildQuery(filters *PostFilters) *gorm.DB {
	query := r.db.Model(&model.Post{})

	if filters == nil {
		return query
	}

	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
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

// checkSlugExists 检查slug是否存在
func (r *postRepository) checkSlugExists(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Post{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}