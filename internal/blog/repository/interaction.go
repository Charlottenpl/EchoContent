package repository

import (
	"fmt"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/gorm"
)

// LikeRepository 点赞仓库接口
type LikeRepository interface {
	Create(like *model.Like) error
	Delete(userID int, targetType string, targetID int) error
	GetByUserAndTarget(userID int, targetType string, targetID int) (*model.Like, error)
	GetLikesByTarget(targetType string, targetID int) ([]*model.Like, error)
	GetLikeCount(targetType string, targetID int) int64
	IsLiked(userID int, targetType string, targetID int) (bool, error)
	GetLikeStats(targetType string, targetID int) (int64, error)
}

// FavoriteRepository 收藏仓库接口
type FavoriteRepository interface {
	Create(favorite *model.Favorite) error
	Delete(userID int, postID int) error
	GetByUserAndPost(userID int, postID int) (*model.Favorite, error)
	GetFavoritesByUser(userID int, page, size int) ([]*model.Favorite, int64, error)
	GetFavoritesByPost(postID int, page, size int) ([]*model.Favorite, int64, error)
	GetFavoriteCount(postID int) int64
	IsFavorited(userID int, postID int) (bool, error)
	GetFavoriteStats(postID int) (int64, error)
}

// likeRepository 点赞仓库实现
type likeRepository struct {
	db *gorm.DB
}

// NewLikeRepository 创建点赞仓库实例
func NewLikeRepository() LikeRepository {
	return &likeRepository{
		db: database.GetDB(),
	}
}

// Create 创建点赞
func (r *likeRepository) Create(like *model.Like) error {
	if err := r.db.Create(like).Error; err != nil {
		logger.Errorf("创建点赞失败: %v", err)
		return fmt.Errorf("创建点赞失败: %w", err)
	}

	logger.Infof("点赞创建成功 (用户ID: %d, 类型: %s, 目标ID: %d)", like.UserID, like.TargetType, like.TargetID)
	return nil
}

// Delete 删除点赞
func (r *likeRepository) Delete(userID int, targetType string, targetID int) error {
	if err := r.db.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&model.Like{}).Error; err != nil {
		logger.Errorf("删除点赞失败: %v", err)
		return fmt.Errorf("删除点赞失败: %w", err)
	}

	logger.Infof("点赞删除成功 (用户ID: %d, 类型: %s, 目标ID: %d)", userID, targetType, targetID)
	return nil
}

// GetByUserAndTarget 根据用户和目标获取点赞
func (r *likeRepository) GetByUserAndTarget(userID int, targetType string, targetID int) (*model.Like, error) {
	var like model.Like
	err := r.db.Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		First(&like).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("点赞不存在")
		}
		logger.Errorf("获取点赞失败: %v", err)
		return nil, fmt.Errorf("获取点赞失败: %w", err)
	}

	return &like, nil
}

// GetLikesByTarget 获取目标的点赞列表
func (r *likeRepository) GetLikesByTarget(targetType string, targetID int) ([]*model.Like, error) {
	var likes []*model.Like
	if err := r.db.Where("target_type = ? AND target_id = ?", targetType, targetID).
		Preload("User").
		Order("created_at DESC").
		Find(&likes).Error; err != nil {
		logger.Errorf("获取点赞列表失败: %v", err)
		return nil, fmt.Errorf("获取点赞列表失败: %w", err)
	}

	return likes, nil
}

// GetLikeCount 获取点赞数
func (r *likeRepository) GetLikeCount(targetType string, targetID int) int64 {
	var count int64
	if err := r.db.Model(&model.Like{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Count(&count).Error; err != nil {
		logger.Errorf("获取点赞数失败: %v", err)
		return 0
	}

	return count
}

// IsLiked 检查用户是否已点赞
func (r *likeRepository) IsLiked(userID int, targetType string, targetID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.Like{}).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetLikeStats 获取点赞统计
func (r *likeRepository) GetLikeStats(targetType string, targetID int) (int64, error) {
	return r.GetLikeCount(targetType, targetID)
}

// favoriteRepository 收藏仓库实现
type favoriteRepository struct {
	db *gorm.DB
}

// NewFavoriteRepository 创建收藏仓库实例
func NewFavoriteRepository() FavoriteRepository {
	return &favoriteRepository{
		db: database.GetDB(),
	}
}

// Create 创建收藏
func (r *favoriteRepository) Create(favorite *model.Favorite) error {
	if err := r.db.Create(favorite).Error; err != nil {
		logger.Errorf("创建收藏失败: %v", err)
		return fmt.Errorf("创建收藏失败: %w", err)
	}

	logger.Infof("收藏创建成功 (用户ID: %d, 文章ID: %d)", favorite.UserID, favorite.PostID)
	return nil
}

// Delete 删除收藏
func (r *favoriteRepository) Delete(userID int, postID int) error {
	if err := r.db.Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&model.Favorite{}).Error; err != nil {
		logger.Errorf("删除收藏失败: %v", err)
		return fmt.Errorf("删除收藏失败: %w", err)
	}

	logger.Infof("收藏删除成功 (用户ID: %d, 文章ID: %d)", userID, postID)
	return nil
}

// GetByUserAndPost 根据用户和文章获取收藏
func (r *favoriteRepository) GetByUserAndPost(userID int, postID int) (*model.Favorite, error) {
	var favorite model.Favorite
	err := r.db.Where("user_id = ? AND post_id = ?", userID, postID).First(&favorite).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("收藏不存在")
		}
		logger.Errorf("获取收藏失败: %v", err)
		return nil, fmt.Errorf("获取收藏失败: %w", err)
	}

	return &favorite, nil
}

// GetFavoritesByUser 获取用户的收藏列表
func (r *favoriteRepository) GetFavoritesByUser(userID int, page, size int) ([]*model.Favorite, int64, error) {
	var favorites []*model.Favorite
	var total int64

	query := r.db.Model(&model.Favorite{}).Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取用户收藏总数失败(用户ID: %d): %v", userID, err)
		return nil, 0, fmt.Errorf("获取用户收藏总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("Post").Preload("Post.Author").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&favorites).Error; err != nil {
		logger.Errorf("获取用户收藏列表失败(用户ID: %d): %v", userID, err)
		return nil, 0, fmt.Errorf("获取用户收藏列表失败: %w", err)
	}

	return favorites, total, nil
}

// GetFavoritesByPost 获取文章的收藏列表
func (r *favoriteRepository) GetFavoritesByPost(postID int, page, size int) ([]*model.Favorite, int64, error) {
	var favorites []*model.Favorite
	var total int64

	query := r.db.Model(&model.Favorite{}).Where("post_id = ?", postID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取文章收藏总数失败(文章ID: %d): %v", postID, err)
		return nil, 0, fmt.Errorf("获取文章收藏总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).Limit(size).
		Find(&favorites).Error; err != nil {
		logger.Errorf("获取文章收藏列表失败(文章ID: %d): %v", postID, err)
		return nil, 0, fmt.Errorf("获取文章收藏列表失败: %w", err)
	}

	return favorites, total, nil
}

// GetFavoriteCount 获取收藏数
func (r *favoriteRepository) GetFavoriteCount(postID int) int64 {
	var count int64
	if err := r.db.Model(&model.Favorite{}).Where("post_id = ?", postID).Count(&count).Error; err != nil {
		logger.Errorf("获取收藏数失败: %v", err)
		return 0
	}

	return count
}

// IsFavorited 检查用户是否已收藏
func (r *favoriteRepository) IsFavorited(userID int, postID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.Favorite{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetFavoriteStats 获取收藏统计
func (r *favoriteRepository) GetFavoriteStats(postID int) (int64, error) {
	return r.GetFavoriteCount(postID)
}

// GetMostFavoritedPosts 获取收藏最多的文章
func (r *favoriteRepository) GetMostFavoritedPosts(limit int) ([]*model.Favorite, error) {
	var favorites []*model.Favorite

	if err := r.db.Preload("Post").Preload("Post.Author").
		Group("post_id").
		Select("post_id, MAX(created_at) as created_at, COUNT(*) as count").
		Order("count DESC, created_at DESC").
		Limit(limit).
		Find(&favorites).Error; err != nil {
		logger.Errorf("获取最多收藏文章失败: %v", err)
		return nil, fmt.Errorf("获取最多收藏文章失败: %w", err)
	}

	// 重新查询以获取完整的数据
	postIDs := make([]int, len(favorites))
	for i, fav := range favorites {
		postIDs[i] = fav.PostID
	}

	if len(postIDs) == 0 {
		return favorites, nil
	}

	var finalFavorites []*model.Favorite
	if err := r.db.Preload("Post").Preload("Post.Author").
		Where("post_id IN ? AND id IN (SELECT MAX(id) FROM favorites GROUP BY post_id ORDER BY COUNT(*) DESC LIMIT ?)", postIDs, limit).
		Order("COUNT(*) DESC, MAX(created_at) DESC").
		Find(&finalFavorites).Error; err != nil {
		logger.Errorf("获取最终收藏数据失败: %v", err)
		return nil, fmt.Errorf("获取最终收藏数据失败: %w", err)
	}

	return finalFavorites, nil
}

// GetUserFavoriteStats 获取用户收藏统计
func (r *favoriteRepository) GetUserFavoriteStats(userID int) (int64, error) {
	var count int64
	if err := r.db.Model(&model.Favorite{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("获取用户收藏统计失败: %w", err)
	}

	return count, nil
}