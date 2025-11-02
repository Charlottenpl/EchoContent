package repository

import (
	"fmt"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/gorm"
)

// CommentRepository 评论仓库接口
type CommentRepository interface {
	Create(comment *model.Comment) error
	GetByID(id int) (*model.Comment, error)
	Update(comment *model.Comment) error
	Delete(id int) error
	List(postID int, page, size int, status string) ([]*model.Comment, int64, error)
	ListByAuthor(authorID int, page, size int) ([]*model.Comment, int64, error)
	ListPending(page, size int) ([]*model.Comment, int64, error)
	Search(keyword string, page, size int) ([]*model.Comment, int64, error)
	GetByIDWithRelations(id int) (*model.Comment, error)
	GetByPostWithRelations(postID int, page, size int) ([]*model.Comment, int64, error)
	UpdateLikeCount(id int) error
	UpdateStatus(id int, status string) error
	ApproveComment(id int) error
	RejectComment(id int) error
	GetReplies(commentID int) ([]*model.Comment, error)
	GetCommentCount(postID int, status string) (int64, error)
}

// commentRepository 评论仓库实现
type commentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 创建评论仓库实例
func NewCommentRepository() CommentRepository {
	return &commentRepository{
		db: database.GetDB(),
	}
}

// Create 创建评论
func (r *commentRepository) Create(comment *model.Comment) error {
	if err := r.db.Create(comment).Error; err != nil {
		logger.Errorf("创建评论失败: %v", err)
		return fmt.Errorf("创建评论失败: %w", err)
	}

	logger.Infof("评论创建成功 (ID: %d, 文章ID: %d)", comment.ID, comment.PostID)
	return nil
}

// GetByID 根据ID获取评论
func (r *commentRepository) GetByID(id int) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.First(&comment, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("评论不存在")
		}
		logger.Errorf("获取评论失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取评论失败: %w", err)
	}

	return &comment, nil
}

// Update 更新评论
func (r *commentRepository) Update(comment *model.Comment) error {
	if err := r.db.Save(comment).Error; err != nil {
		logger.Errorf("更新评论失败(ID: %d): %v", comment.ID, err)
		return fmt.Errorf("更新评论失败: %w", err)
	}

	logger.Infof("评论更新成功 (ID: %d)", comment.ID)
	return nil
}

// Delete 删除评论
func (r *commentRepository) Delete(id int) error {
	if err := r.db.Delete(&model.Comment{}, id).Error; err != nil {
		logger.Errorf("删除评论失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除评论失败: %w", err)
	}

	logger.Infof("评论删除成功 (ID: %d)", id)
	return nil
}

// List 获取评论列表
func (r *commentRepository) List(postID int, page, size int, status string) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{}).Where("post_id = ?", postID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取评论总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取评论总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").
		Preload("Parent").
		Order("created_at ASC").
		Offset(offset).Limit(size).
		Find(&comments).Error; err != nil {
		logger.Errorf("获取评论列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取评论列表失败: %w", err)
	}

	return comments, total, nil
}

// ListByAuthor 根据作者获取评论列表
func (r *commentRepository) ListByAuthor(authorID int, page, size int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{}).Where("author_id = ?", authorID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取作者评论总数失败(作者ID: %d): %v", authorID, err)
		return nil, 0, fmt.Errorf("获取作者评论总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Parent").
		Preload("Post").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&comments).Error; err != nil {
		logger.Errorf("获取作者评论列表失败(作者ID: %d): %v", authorID, err)
		return nil, 0, fmt.Errorf("获取作者评论列表失败: %w", err)
	}

	return comments, total, nil
}

// ListPending 获取待审核评论列表
func (r *commentRepository) ListPending(page, size int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{}).Where("status = ?", "pending")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取待审核评论总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取待审核评论总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Parent").
		Preload("Post").
		Order("created_at ASC").
		Offset(offset).Limit(size).
		Find(&comments).Error; err != nil {
		logger.Errorf("获取待审核评论列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取待审核评论列表失败: %w", err)
	}

	return comments, total, nil
}

// Search 搜索评论
func (r *commentRepository) Search(keyword string, page, size int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	if keyword == "" {
		return []*model.Comment{}, 0, nil
	}

	searchPattern := "%" + keyword + "%"

	// 构建搜索查询
	query := r.db.Model(&model.Comment{}).
		Where("(content LIKE ? OR author_name LIKE ? OR author_email LIKE ?)", searchPattern, searchPattern, searchPattern)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("搜索评论总数失败: %v", err)
		return nil, 0, fmt.Errorf("搜索评论总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Parent").
		Preload("Post").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&comments).Error; err != nil {
		logger.Errorf("搜索评论失败: %v", err)
		return nil, 0, fmt.Errorf("搜索评论失败: %w", err)
	}

	return comments, total, nil
}

// GetByIDWithRelations 根据ID获取评论（包含关联数据）
func (r *commentRepository) GetByIDWithRelations(id int) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.Preload("Author").Preload("Parent").Preload("Children").Preload("Post").
		First(&comment, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("评论不存在")
		}
		logger.Errorf("获取评论详情失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取评论详情失败: %w", err)
	}

	return &comment, nil
}

// GetByPostWithRelations 根据文章ID获取评论（包含关联数据）
func (r *commentRepository) GetByPostWithRelations(postID int, page, size int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{}).Where("post_id = ? AND status = ?", postID, "approved")

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取文章评论总数失败(文章ID: %d): %v", postID, err)
		return nil, 0, fmt.Errorf("获取文章评论总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Author").Preload("Parent").
		Order("created_at ASC").
		Offset(offset).Limit(size).
		Find(&comments).Error; err != nil {
		logger.Errorf("获取文章评论列表失败(文章ID: %d): %v", postID, err)
		return nil, 0, fmt.Errorf("获取文章评论列表失败: %w", err)
	}

	return comments, total, nil
}

// UpdateLikeCount 更新评论点赞数
func (r *commentRepository) UpdateLikeCount(id int) error {
	if err := r.db.Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
		logger.Errorf("更新评论点赞数失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新评论点赞数失败: %w", err)
	}

	return nil
}

// UpdateStatus 更新评论状态
func (r *commentRepository) UpdateStatus(id int, status string) error {
	if err := r.db.Model(&model.Comment{}).Where("id = ?", id).
		Update("status", status).Error; err != nil {
		logger.Errorf("更新评论状态失败(ID: %d): %v", id, err)
		return fmt.Errorf("更新评论状态失败: %w", err)
	}

	return nil
}

// ApproveComment 审核通过评论
func (r *commentRepository) ApproveComment(id int) error {
	if err := r.UpdateStatus(id, "approved"); err != nil {
		return fmt.Errorf("审核通过评论失败: %w", err)
	}

	logger.Infof("评论审核通过 (ID: %d)", id)
	return nil
}

// RejectComment 拒绝评论
func (r *commentRepository) RejectComment(id int) error {
	if err := r.UpdateStatus(id, "rejected"); err != nil {
		return fmt.Errorf("拒绝评论失败: %w", err)
	}

	logger.Infof("评论被拒绝 (ID: %d)", id)
	return nil
}

// GetReplies 获取评论回复
func (r *commentRepository) GetReplies(commentID int) ([]*model.Comment, error) {
	var replies []*model.Comment

	if err := r.db.Where("parent_id = ? AND status = ?", commentID, "approved").
		Preload("Author").
		Order("created_at ASC").
		Find(&replies).Error; err != nil {
		logger.Errorf("获取评论回复失败(评论ID: %d): %v", commentID, err)
		return nil, fmt.Errorf("获取评论回复失败: %w", err)
	}

	return replies, nil
}

// GetCommentCount 获取文章评论数
func (r *commentRepository) GetCommentCount(postID int, status string) (int64, error) {
	var count int64

	query := r.db.Model(&model.Comment{}).Where("post_id = ?", postID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取评论数失败: %w", err)
	}

	return count, nil
}