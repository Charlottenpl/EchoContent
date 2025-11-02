package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/blog/repository"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/core/validator"
)

// CommentService 评论服务
type CommentService struct {
	commentRepo repository.CommentRepository
	postRepo   repository.PostRepository
}

// NewCommentService 创建评论服务实例
func NewCommentService() *CommentService {
	return &CommentService{
		commentRepo: repository.NewCommentRepository(),
		postRepo:    repository.NewPostRepository(),
	}
}

// CreateCommentRequest 创建评论请求
type CreateCommentRequest struct {
	PostID      int    `json:"post_id" validate:"required,min=1"`
	ParentID    *int   `json:"parent_id"`
	Content     string `json:"content" validate:"required,min=1,max=1000"`
	AuthorName  string `json:"author_name" validate:"required,min=1,max=50"`
	AuthorEmail string `json:"author_email" validate:"required,email,max=100"`
	AuthorURL   string `json:"author_url" validate:"omitempty,url,max=200"`
	AuthorIP    string `json:"author_ip"`
	UserAgent   string `json:"user_agent"`
}

// UpdateCommentRequest 更新评论请求
type UpdateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// CommentListRequest 评论列表请求
type CommentListRequest struct {
	Page   int    `form:"page,default=1" validate:"min=1"`
	Size   int    `form:"size,default=20" validate:"min=1,max=100"`
	Status string `form:"status"`
}

// CreateComment 创建评论
func (s *CommentService) CreateComment(ctx context.Context, req *CreateCommentRequest, userID *int) (*model.Comment, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 检查文章是否存在
	post, err := s.postRepo.GetByID(req.PostID)
	if err != nil {
		return nil, fmt.Errorf("文章不存在: %w", err)
	}

	// 检查文章是否允许评论
	if !post.CanComment() {
		return nil, fmt.Errorf("该文章不允许评论")
	}

	// 如果有父评论，检查父评论是否存在
	if req.ParentID != nil && *req.ParentID > 0 {
		parentComment, err := s.commentRepo.GetByID(*req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("父评论不存在: %w", err)
		}

		// 检查父评论是否属于同一篇文章
		if parentComment.PostID != req.PostID {
			return nil, fmt.Errorf("父评论不属于该文章")
		}

		// 防止嵌套层级过深（最多3级）
		if parentComment.ParentID != nil && *parentComment.ParentID > 0 {
			return nil, fmt.Errorf("评论嵌套层级过深")
		}
	}

	// 内容过滤和清理
	content := strings.TrimSpace(req.Content)
	content = s.filterContent(content)

	// 创建评论
	comment := &model.Comment{
		PostID:      req.PostID,
		ParentID:    req.ParentID,
		Content:     content,
		AuthorID:    userID,
		AuthorName:  req.AuthorName,
		AuthorEmail: req.AuthorEmail,
		AuthorURL:   req.AuthorURL,
		AuthorIP:    req.AuthorIP,
		UserAgent:   req.UserAgent,
		Status:      s.determineCommentStatus(userID, content),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.commentRepo.Create(comment); err != nil {
		logger.Errorf("创建评论失败: %v", err)
		return nil, fmt.Errorf("创建评论失败: %w", err)
	}

	// 获取完整的评论信息
	fullComment, err := s.commentRepo.GetByIDWithRelations(comment.ID)
	if err != nil {
		logger.Errorf("获取评论详情失败: %v", err)
		return comment, nil // 返回基本评论信息
	}

	logger.Infof("评论创建成功 (ID: %d, 文章ID: %d, 用户ID: %v)", fullComment.ID, fullComment.PostID, userID)
	return fullComment, nil
}

// UpdateComment 更新评论
func (s *CommentService) UpdateComment(ctx context.Context, id int, req *UpdateCommentRequest, userID *int) (*model.Comment, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 获取评论
	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("评论不存在: %w", err)
	}

	// 权限检查
	if !s.canEditComment(comment, userID) {
		return nil, fmt.Errorf("无权限编辑该评论")
	}

	// 内容过滤和清理
	content := strings.TrimSpace(req.Content)
	content = s.filterContent(content)

	// 更新评论
	comment.Content = content
	comment.UpdatedAt = time.Now()

	if err := s.commentRepo.Update(comment); err != nil {
		logger.Errorf("更新评论失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("更新评论失败: %w", err)
	}

	// 获取完整的评论信息
	fullComment, err := s.commentRepo.GetByIDWithRelations(id)
	if err != nil {
		logger.Errorf("获取评论详情失败: %v", err)
		return comment, nil // 返回基本评论信息
	}

	logger.Infof("评论更新成功 (ID: %d, 用户ID: %v)", id, userID)
	return fullComment, nil
}

// DeleteComment 删除评论
func (s *CommentService) DeleteComment(ctx context.Context, id int, userID *int) error {
	// 获取评论
	comment, err := s.commentRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("评论不存在: %w", err)
	}

	// 权限检查
	if !s.canDeleteComment(comment, userID) {
		return fmt.Errorf("无权限删除该评论")
	}

	// 检查是否有子评论
	replies, err := s.commentRepo.GetReplies(id)
	if err != nil {
		logger.Errorf("获取子评论失败: %v", err)
	} else if len(replies) > 0 {
		// 如果有子评论，不删除评论，而是标记为已删除
		comment.Status = "deleted"
		comment.Content = "[该评论已被删除]"
		comment.UpdatedAt = time.Now()

		if err := s.commentRepo.Update(comment); err != nil {
			logger.Errorf("标记评论删除失败(ID: %d): %v", id, err)
			return fmt.Errorf("标记评论删除失败: %w", err)
		}

		logger.Infof("评论标记删除成功 (ID: %d, 用户ID: %v)", id, userID)
		return nil
	}

	// 删除评论
	if err := s.commentRepo.Delete(id); err != nil {
		logger.Errorf("删除评论失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除评论失败: %w", err)
	}

	logger.Infof("评论删除成功 (ID: %d, 用户ID: %v)", id, userID)
	return nil
}

// GetComment 根据ID获取评论
func (s *CommentService) GetComment(ctx context.Context, id int) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByIDWithRelations(id)
	if err != nil {
		return nil, fmt.Errorf("获取评论失败: %w", err)
	}

	return comment, nil
}

// GetCommentsByPost 获取文章评论列表
func (s *CommentService) GetCommentsByPost(ctx context.Context, postID int, req *CommentListRequest) ([]*model.Comment, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 检查文章是否存在
	_, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, 0, fmt.Errorf("文章不存在: %w", err)
	}

	// 获取评论列表
	comments, total, err := s.commentRepo.GetByPostWithRelations(postID, req.Page, req.Size)
	if err != nil {
		return nil, 0, fmt.Errorf("获取评论列表失败: %w", err)
	}

	return comments, total, nil
}

// GetCommentsByAuthor 获取作者评论列表
func (s *CommentService) GetCommentsByAuthor(ctx context.Context, authorID int, req *CommentListRequest) ([]*model.Comment, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 获取评论列表
	comments, total, err := s.commentRepo.ListByAuthor(authorID, req.Page, req.Size)
	if err != nil {
		return nil, 0, fmt.Errorf("获取作者评论列表失败: %w", err)
	}

	return comments, total, nil
}

// SearchComments 搜索评论
func (s *CommentService) SearchComments(ctx context.Context, keyword string, req *CommentListRequest) ([]*model.Comment, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	// 搜索评论
	comments, total, err := s.commentRepo.Search(keyword, req.Page, req.Size)
	if err != nil {
		return nil, 0, fmt.Errorf("搜索评论失败: %w", err)
	}

	return comments, total, nil
}

// ApproveComment 审核通过评论
func (s *CommentService) ApproveComment(ctx context.Context, id int) error {
	if err := s.commentRepo.ApproveComment(id); err != nil {
		logger.Errorf("审核通过评论失败(ID: %d): %v", id, err)
		return fmt.Errorf("审核通过评论失败: %w", err)
	}

	logger.Infof("评论审核通过 (ID: %d)", id)
	return nil
}

// RejectComment 拒绝评论
func (s *CommentService) RejectComment(ctx context.Context, id int) error {
	if err := s.commentRepo.RejectComment(id); err != nil {
		logger.Errorf("拒绝评论失败(ID: %d): %v", id, err)
		return fmt.Errorf("拒绝评论失败: %w", err)
	}

	logger.Infof("评论被拒绝 (ID: %d)", id)
	return nil
}

// GetPendingComments 获取待审核评论列表
func (s *CommentService) GetPendingComments(ctx context.Context, req *CommentListRequest) ([]*model.Comment, int64, error) {
	// 验证请求数据
	if err := validator.Validate(req); err != nil {
		return nil, 0, fmt.Errorf("请求参数验证失败: %w", err)
	}

	comments, total, err := s.commentRepo.ListPending(req.Page, req.Size)
	if err != nil {
		return nil, 0, fmt.Errorf("获取待审核评论列表失败: %w", err)
	}

	return comments, total, nil
}

// GetCommentReplies 获取评论回复
func (s *CommentService) GetCommentReplies(ctx context.Context, commentID int) ([]*model.Comment, error) {
	replies, err := s.commentRepo.GetReplies(commentID)
	if err != nil {
		return nil, fmt.Errorf("获取评论回复失败: %w", err)
	}

	return replies, nil
}

// GetCommentStats 获取评论统计
func (s *CommentService) GetCommentStats(ctx context.Context, postID int) (int64, error) {
	count, err := s.commentRepo.GetCommentCount(postID, "approved")
	if err != nil {
		return 0, fmt.Errorf("获取评论统计失败: %w", err)
	}

	return count, nil
}

// determineCommentStatus 确定评论状态
func (s *CommentService) determineCommentStatus(userID *int, content string) string {
	// 如果是登录用户的评论，直接通过
	if userID != nil {
		return "approved"
	}

	// 检查内容是否包含敏感词
	if s.containsSensitiveWords(content) {
		return "pending"
	}

	// 其他情况直接通过
	return "approved"
}

// filterContent 过滤评论内容
func (s *CommentService) filterContent(content string) string {
	// 移除多余的空白字符
	content = strings.Join(strings.Fields(content), " ")

	// 这里可以添加更多的内容过滤逻辑
	// 比如敏感词过滤、HTML标签过滤等

	return content
}

// containsSensitiveWords 检查是否包含敏感词
func (s *CommentService) containsSensitiveWords(content string) bool {
	// 这里可以实现敏感词检测逻辑
	// 为了简化，暂时返回false
	return false
}

// canEditComment 检查是否可以编辑评论
func (s *CommentService) canEditComment(comment *model.Comment, userID *int) bool {
	// 管理员可以编辑所有评论
	if userID != nil {
		// 这里应该检查用户是否为管理员
		// 暂时简化处理
	}

	// 评论作者可以编辑自己的评论
	if userID != nil && comment.AuthorID != nil && *comment.AuthorID == *userID {
		// 检查评论创建时间，超过24小时不允许编辑
		if time.Since(comment.CreatedAt) > 24*time.Hour {
			return false
		}
		return true
	}

	return false
}

// canDeleteComment 检查是否可以删除评论
func (s *CommentService) canDeleteComment(comment *model.Comment, userID *int) bool {
	// 管理员可以删除所有评论
	if userID != nil {
		// 这里应该检查用户是否为管理员
		// 暂时简化处理
		return true
	}

	// 评论作者可以删除自己的评论
	if userID != nil && comment.AuthorID != nil && *comment.AuthorID == *userID {
		return true
	}

	return false
}

// LikeComment 点赞评论
func (s *CommentService) LikeComment(ctx context.Context, commentID int, userID int) error {
	// 检查评论是否存在
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return fmt.Errorf("评论不存在: %w", err)
	}

	// 更新评论点赞数
	if err := s.commentRepo.UpdateLikeCount(commentID); err != nil {
		logger.Errorf("更新评论点赞数失败(ID: %d): %v", commentID, err)
		return fmt.Errorf("更新评论点赞数失败: %w", err)
	}

	logger.Infof("评论点赞成功 (ID: %d, 用户ID: %d)", commentID, userID)
	return nil
}

// GetCommentWithRelations 获取评论及其关联数据
func (s *CommentService) GetCommentWithRelations(ctx context.Context, id int) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByIDWithRelations(id)
	if err != nil {
		return nil, fmt.Errorf("获取评论详情失败: %w", err)
	}

	return comment, nil
}