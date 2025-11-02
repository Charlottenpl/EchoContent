package service

import (
	"context"
	"fmt"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/blog/repository"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/core/validator"
)

// InteractionService 互动服务
type InteractionService struct {
	likeRepo     repository.LikeRepository
	favoriteRepo repository.FavoriteRepository
	postRepo     repository.PostRepository
	commentRepo  repository.CommentRepository
}

// NewInteractionService 创建互动服务实例
func NewInteractionService() *InteractionService {
	return &InteractionService{
		likeRepo:     repository.NewLikeRepository(),
		favoriteRepo: repository.NewFavoriteRepository(),
		postRepo:     repository.NewPostRepository(),
		commentRepo:  repository.NewCommentRepository(),
	}
}

// LikeRequest 点赞请求
type LikeRequest struct {
	TargetID   int    `json:"target_id" validate:"required,min=1"`
	TargetType string `json:"target_type" validate:"required,oneof=post comment"`
}

// FavoriteRequest 收藏请求
type FavoriteRequest struct {
	PostID int `json:"post_id" validate:"required,min=1"`
}

// InteractionListRequest 互动列表请求
type InteractionListRequest struct {
	Page int `form:"page,default=1" validate:"min=1"`
	Size int `form:"size,default=20" validate:"min=1,max=100"`
}

// Like 点赞
func (s *InteractionService) Like(ctx context.Context, userID int, req *LikeRequest) error {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 验证目标是否存在
	if err := s.validateTarget(req.TargetType, req.TargetID); err != nil {
		return err
	}

	// 检查是否已经点赞
	isLiked, err := s.likeRepo.IsLiked(userID, req.TargetType, req.TargetID)
	if err != nil {
		logger.Errorf("检查点赞状态失败: %v", err)
		return fmt.Errorf("检查点赞状态失败: %w", err)
	}

	if isLiked {
		return fmt.Errorf("已经点赞过了")
	}

	// 创建点赞记录
	like := &model.Like{
		UserID:     userID,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
	}

	if err := s.likeRepo.Create(like); err != nil {
		logger.Errorf("创建点赞失败: %v", err)
		return fmt.Errorf("创建点赞失败: %w", err)
	}

	// 更新目标点赞数
	if err := s.updateTargetLikeCount(req.TargetType, req.TargetID, true); err != nil {
		logger.Errorf("更新目标点赞数失败: %v", err)
		// 不返回错误，因为点赞记录已经创建成功
	}

	logger.Infof("用户点赞成功 (用户ID: %d, 类型: %s, 目标ID: %d)", userID, req.TargetType, req.TargetID)
	return nil
}

// Unlike 取消点赞
func (s *InteractionService) Unlike(ctx context.Context, userID int, req *LikeRequest) error {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 检查是否已经点赞
	isLiked, err := s.likeRepo.IsLiked(userID, req.TargetType, req.TargetID)
	if err != nil {
		logger.Errorf("检查点赞状态失败: %v", err)
		return fmt.Errorf("检查点赞状态失败: %w", err)
	}

	if !isLiked {
		return fmt.Errorf("还未点赞")
	}

	// 删除点赞记录
	if err := s.likeRepo.Delete(userID, req.TargetType, req.TargetID); err != nil {
		logger.Errorf("删除点赞失败: %v", err)
		return fmt.Errorf("删除点赞失败: %w", err)
	}

	// 更新目标点赞数
	if err := s.updateTargetLikeCount(req.TargetType, req.TargetID, false); err != nil {
		logger.Errorf("更新目标点赞数失败: %v", err)
		// 不返回错误，因为点赞记录已经删除成功
	}

	logger.Infof("用户取消点赞成功 (用户ID: %d, 类型: %s, 目标ID: %d)", userID, req.TargetType, req.TargetID)
	return nil
}

// Favorite 收藏
func (s *InteractionService) Favorite(ctx context.Context, userID int, req *FavoriteRequest) error {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 验证文章是否存在
	post, err := s.postRepo.GetByID(req.PostID)
	if err != nil {
		return fmt.Errorf("文章不存在: %w", err)
	}

	// 检查是否已经收藏
	isFavorited, err := s.favoriteRepo.IsFavorited(userID, req.PostID)
	if err != nil {
		logger.Errorf("检查收藏状态失败: %v", err)
		return fmt.Errorf("检查收藏状态失败: %w", err)
	}

	if isFavorited {
		return fmt.Errorf("已经收藏过了")
	}

	// 创建收藏记录
	favorite := &model.Favorite{
		UserID: userID,
		PostID: req.PostID,
	}

	if err := s.favoriteRepo.Create(favorite); err != nil {
		logger.Errorf("创建收藏失败: %v", err)
		return fmt.Errorf("创建收藏失败: %w", err)
	}

	// 更新文章收藏数
	if err := s.postRepo.UpdateFavoriteCount(req.PostID, true); err != nil {
		logger.Errorf("更新文章收藏数失败: %v", err)
		// 不返回错误，因为收藏记录已经创建成功
	}

	logger.Infof("用户收藏成功 (用户ID: %d, 文章ID: %d)", userID, req.PostID)
	return nil
}

// Unfavorite 取消收藏
func (s *InteractionService) Unfavorite(ctx context.Context, userID int, req *FavoriteRequest) error {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 检查是否已经收藏
	isFavorited, err := s.favoriteRepo.IsFavorited(userID, req.PostID)
	if err != nil {
		logger.Errorf("检查收藏状态失败: %v", err)
		return fmt.Errorf("检查收藏状态失败: %w", err)
	}

	if !isFavorited {
		return fmt.Errorf("还未收藏")
	}

	// 删除收藏记录
	if err := s.favoriteRepo.Delete(userID, req.PostID); err != nil {
		logger.Errorf("删除收藏失败: %v", err)
		return fmt.Errorf("删除收藏失败: %w", err)
	}

	// 更新文章收藏数
	if err := s.postRepo.UpdateFavoriteCount(req.PostID, false); err != nil {
		logger.Errorf("更新文章收藏数失败: %v", err)
		// 不返回错误，因为收藏记录已经删除成功
	}

	logger.Infof("用户取消收藏成功 (用户ID: %d, 文章ID: %d)", userID, req.PostID)
	return nil
}

// GetLikeStatus 获取点赞状态
func (s *InteractionService) GetLikeStatus(ctx context.Context, userID int, targetType string, targetID int) (bool, error) {
	isLiked, err := s.likeRepo.IsLiked(userID, targetType, targetID)
	if err != nil {
		return false, fmt.Errorf("获取点赞状态失败: %w", err)
	}

	return isLiked, nil
}

// GetFavoriteStatus 获取收藏状态
func (s *InteractionService) GetFavoriteStatus(ctx context.Context, userID int, postID int) (bool, error) {
	isFavorited, err := s.favoriteRepo.IsFavorited(userID, postID)
	if err != nil {
		return false, fmt.Errorf("获取收藏状态失败: %w", err)
	}

	return isFavorited, nil
}

// GetUserLikes 获取用户点赞列表
func (s *InteractionService) GetUserLikes(ctx context.Context, userID int, targetType string, req *InteractionListRequest) ([]*model.Like, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 构建查询条件
	var likes []*model.Like
	var total int64

	// 由于仓库层没有直接提供根据用户获取点赞列表的方法，
	// 这里需要通过现有方法组合实现
	// 暂时返回空列表，实际项目中应该在仓库层添加相应方法

	return likes, total, nil
}

// GetUserFavorites 获取用户收藏列表
func (s *InteractionService) GetUserFavorites(ctx context.Context, userID int, req *InteractionListRequest) ([]*model.Favorite, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	favorites, total, err := s.favoriteRepo.GetFavoritesByUser(userID, req.Page, req.Size)
	if err != nil {
		return nil, 0, fmt.Errorf("获取用户收藏列表失败: %w", err)
	}

	return favorites, total, nil
}

// GetPostLikes 获取文章点赞列表
func (s *InteractionService) GetPostLikes(ctx context.Context, postID int, req *InteractionListRequest) ([]*model.Like, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 检查文章是否存在
	_, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, 0, fmt.Errorf("文章不存在: %w", err)
	}

	likes, err := s.likeRepo.GetLikesByTarget("post", postID)
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章点赞列表失败: %w", err)
	}

	// 手动分页
	total := int64(len(likes))
	start := (req.Page - 1) * req.Size
	end := start + req.Size

	if start >= len(likes) {
		return []*model.Like{}, total, nil
	}

	if end > len(likes) {
		end = len(likes)
	}

	paginatedLikes := likes[start:end]

	return paginatedLikes, total, nil
}

// GetPostFavorites 获取文章收藏列表
func (s *InteractionService) GetPostFavorites(ctx context.Context, postID int, req *InteractionListRequest) ([]*model.Favorite, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 检查文章是否存在
	_, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, 0, fmt.Errorf("文章不存在: %w", err)
	}

	favorites, total, err := s.favoriteRepo.GetFavoritesByPost(postID, req.Page, req.Size)
	if err != nil {
		return nil, 0, fmt.Errorf("获取文章收藏列表失败: %w", err)
	}

	return favorites, total, nil
}

// GetLikeStats 获取点赞统计
func (s *InteractionService) GetLikeStats(ctx context.Context, targetType string, targetID int) (int64, error) {
	count, err := s.likeRepo.GetLikeCount(targetType, targetID)
	if err != nil {
		return 0, fmt.Errorf("获取点赞统计失败: %w", err)
	}

	return count, nil
}

// GetFavoriteStats 获取收藏统计
func (s *InteractionService) GetFavoriteStats(ctx context.Context, postID int) (int64, error) {
	count, err := s.favoriteRepo.GetFavoriteCount(postID)
	if err != nil {
		return 0, fmt.Errorf("获取收藏统计失败: %w", err)
	}

	return count, nil
}

// GetUserInteractionStats 获取用户互动统计
func (s *InteractionService) GetUserInteractionStats(ctx context.Context, userID int) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 获取用户点赞数
	likeCount, err := s.getUserLikeCount(userID)
	if err != nil {
		logger.Errorf("获取用户点赞数失败: %v", err)
		likeCount = 0
	}
	stats["likes"] = likeCount

	// 获取用户收藏数
	favoriteCount, err := s.favoriteRepo.GetUserFavoriteStats(userID)
	if err != nil {
		logger.Errorf("获取用户收藏数失败: %v", err)
		favoriteCount = 0
	}
	stats["favorites"] = favoriteCount

	return stats, nil
}

// validateTarget 验证目标是否存在
func (s *InteractionService) validateTarget(targetType string, targetID int) error {
	switch targetType {
	case "post":
		_, err := s.postRepo.GetByID(targetID)
		if err != nil {
			return fmt.Errorf("文章不存在: %w", err)
		}
	case "comment":
		_, err := s.commentRepo.GetByID(targetID)
		if err != nil {
			return fmt.Errorf("评论不存在: %w", err)
		}
	default:
		return fmt.Errorf("不支持的目标类型: %s", targetType)
	}

	return nil
}

// updateTargetLikeCount 更新目标点赞数
func (s *InteractionService) updateTargetLikeCount(targetType string, targetID int, increment bool) error {
	switch targetType {
	case "post":
		return s.postRepo.UpdateLikeCount(targetID, increment)
	case "comment":
		// 评论点赞数更新需要在comment仓库中实现
		// 暂时跳过
		return nil
	default:
		return fmt.Errorf("不支持的目标类型: %s", targetType)
	}
}

// getUserLikeCount 获取用户点赞数（简化实现）
func (s *InteractionService) getUserLikeCount(userID int) (int64, error) {
	// 这里应该在like仓库中添加根据用户统计点赞数的方法
	// 暂时返回0
	return 0, nil
}

// GetMostLikedPosts 获取点赞最多的文章
func (s *InteractionService) GetMostLikedPosts(ctx context.Context, limit int) ([]*model.Post, error) {
	// 这个功能需要在post仓库中实现
	// 暂时返回空列表
	return []*model.Post{}, nil
}

// GetMostFavoritedPosts 获取收藏最多的文章
func (s *InteractionService) GetMostFavoritedPosts(ctx context.Context, limit int) ([]*model.Favorite, error) {
	favorites, err := s.favoriteRepo.GetMostFavoritedPosts(limit)
	if err != nil {
		return nil, fmt.Errorf("获取收藏最多的文章失败: %w", err)
	}

	return favorites, nil
}

// BatchLike 批量点赞
func (s *InteractionService) BatchLike(ctx context.Context, userID int, requests []LikeRequest) error {
	if len(requests) == 0 {
		return nil
	}

	for _, req := range requests {
		if err := s.Like(ctx, userID, &req); err != nil {
			logger.Errorf("批量点赞失败 (用户ID: %d, 类型: %s, 目标ID: %d): %v",
				userID, req.TargetType, req.TargetID, err)
			// 继续处理其他点赞请求，不中断整个批量操作
		}
	}

	logger.Infof("批量点赞完成 (用户ID: %d, 请求数量: %d)", userID, len(requests))
	return nil
}

// BatchUnfavorite 批量取消收藏
func (s *InteractionService) BatchUnfavorite(ctx context.Context, userID int, postIDs []int) error {
	if len(postIDs) == 0 {
		return nil
	}

	for _, postID := range postIDs {
		req := &FavoriteRequest{PostID: postID}
		if err := s.Unfavorite(ctx, userID, req); err != nil {
			logger.Errorf("批量取消收藏失败 (用户ID: %d, 文章ID: %d): %v",
				userID, postID, err)
			// 继续处理其他取消收藏请求，不中断整个批量操作
		}
	}

	logger.Infof("批量取消收藏完成 (用户ID: %d, 文章数量: %d)", userID, len(postIDs))
	return nil
}