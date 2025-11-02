package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/blog/repository"
	"github.com/charlottepl/blog-system/internal/core/cache"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/user/model"
	"gorm.io/gorm"
)

// PostService 文章服务
type PostService struct {
	postRepo    repository.PostRepository
	categoryRepo repository.CategoryRepository
	tagRepo     repository.TagRepository
}

// NewPostService 创建文章服务实例
func NewPostService() *PostService {
	return &PostService{
		postRepo:    repository.NewPostRepository(),
		categoryRepo: repository.NewCategoryRepository(),
		tagRepo:     repository.NewTagRepository(),
	}
}

// CreatePostRequest 创建文章请求
type CreatePostRequest struct {
	Title         string   `json:"title" binding:"required,max=200"`
	Content       string   `json:"content" binding:"required"`
	Excerpt       string   `json:"excerpt"`
	Type          string   `json:"type" binding:"required,oneof=blog moment"`
	Status        string   `json:"status" binding:"omitempty,oneof=draft published private"`
	CategoryID    *int     `json:"category_id"`
	TagNames      []string `json:"tag_names"`
	FeaturedImage string   `json:"featured_image"`
}

// UpdatePostRequest 更新文章请求
type UpdatePostRequest struct {
	Title         string   `json:"title" binding:"omitempty,max=200"`
	Content       string   `json:"content" binding:"omitempty"`
	Excerpt       string   `json:"excerpt"`
	Status        string   `json:"status" binding:"omitempty,oneof=draft published private"`
	CategoryID    *int     `json:"category_id"`
	TagNames      []string `json:"tag_names"`
	FeaturedImage string   `json:"featured_image"`
}

// PostListResponse 文章列表响应
type PostListResponse struct {
	Posts      []*model.Post           `json:"posts"`
	Pagination repository.Pagination   `json:"pagination"`
}

// CreatePost 创建文章
func (s *PostService) CreatePost(ctx context.Context, req *CreatePostRequest, author *model.User) (*model.Post, error) {
	// 权限检查
	if !author.CanCreatePost() {
		return nil, fmt.Errorf("权限不足，只有管理员可以创建文章")
	}

	// 内容验证
	if err := s.validatePostContent(req.Title, req.Content); err != nil {
		return nil, err
	}

	// 验证分类
	if req.CategoryID != nil {
		if _, err := s.categoryRepo.GetByID(*req.CategoryID); err != nil {
			return nil, fmt.Errorf("分类不存在")
		}
	}

	// 处理标签
	tags, err := s.tagRepo.GetOrCreateTags(req.TagNames)
	if err != nil {
		return nil, fmt.Errorf("处理标签失败: %w", err)
	}

	// 生成slug
	slug := s.generateSlug(req.Title)
	if exists, _ := s.postRepo.(*repository.postRepository).checkSlugExists(slug); exists {
		slug = s.generateUniqueSlug(req.Title)
	}

	// 创建文章
	post := &model.Post{
		Title:         req.Title,
		Slug:          slug,
		Content:       req.Content,
		Excerpt:       req.Excerpt,
		Type:          req.Type,
		Status:        "draft", // 默认为草稿
		AuthorID:      author.ID,
		CategoryID:    req.CategoryID,
		FeaturedImage: req.FeaturedImage,
	}

	if req.Status != "" {
		post.Status = req.Status
	}

	// 保存文章
	if err := s.postRepo.Create(post); err != nil {
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}

	// 关联标签
	if len(tags) > 0 {
		if err := s.associatePostTags(post.ID, tags); err != nil {
			// 回滚文章创建
			s.postRepo.Delete(post.ID)
			return nil, fmt.Errorf("关联标签失败: %w", err)
		}
	}

	// 如果状态是发布，设置发布时间
	if post.Status == "published" && post.PublishedAt == nil {
		if err := s.postRepo.Publish(post.ID); err != nil {
			logger.Warnf("发布文章失败(ID: %d): %v", post.ID, err)
		}
	}

	// 清除相关缓存
	s.clearPostCache()

	logger.Infof("文章创建成功: %s (ID: %d, 作者: %s)", post.Title, post.ID, author.Username)

	return post, nil
}

// UpdatePost 更新文章
func (s *PostService) UpdatePost(ctx context.Context, id int, req *UpdatePostRequest, user *model.User) (*model.Post, error) {
	// 获取文章
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 权限检查
	if !post.CanEdit(user) {
		return nil, fmt.Errorf("权限不足")
	}

	// 验证分类
	if req.CategoryID != nil {
		if _, err := s.categoryRepo.GetByID(*req.CategoryID); err != nil {
			return nil, fmt.Errorf("分类不存在")
		}
	}

	// 更新字段
	if req.Title != "" {
		post.Title = req.Title
		// 重新生成slug
		newSlug := s.generateSlug(req.Title)
		if newSlug != post.Slug {
			if exists, _ := s.postRepo.(*repository.postRepository).checkSlugExists(newSlug); exists {
				newSlug = s.generateUniqueSlug(req.Title)
			}
			post.Slug = newSlug
		}
	}

	if req.Content != "" {
		post.Content = req.Content
	}

	if req.Excerpt != "" {
		post.Excerpt = req.Excerpt
	}

	if req.Status != "" {
		post.Status = req.Status
	}

	if req.CategoryID != nil {
		post.CategoryID = req.CategoryID
	}

	if req.FeaturedImage != "" {
		post.FeaturedImage = req.FeaturedImage
	}

	// 处理标签
	if req.TagNames != nil {
		tags, err := s.tagRepo.GetOrCreateTags(req.TagNames)
		if err != nil {
			return nil, fmt.Errorf("处理标签失败: %w", err)
		}

		// 更新标签关联
		if err := s.updatePostTags(post.ID, tags); err != nil {
			return nil, fmt.Errorf("更新标签关联失败: %w", err)
		}
	}

	// 保存文章
	if err := s.postRepo.Update(post); err != nil {
		return nil, fmt.Errorf("更新文章失败: %w", err)
	}

	// 如果状态改为发布，设置发布时间
	if post.Status == "published" && post.PublishedAt == nil {
		if err := s.postRepo.Publish(post.ID); err != nil {
			logger.Warnf("发布文章失败(ID: %d): %v", post.ID, err)
		}
	}

	// 清除相关缓存
	s.clearPostCache()

	logger.Infof("文章更新成功: %s (ID: %d, 操作者: %s)", post.Title, post.ID, user.Username)

	return post, nil
}

// GetPost 获取文章详情
func (s *PostService) GetPost(ctx context.Context, id int, user *model.User) (*model.Post, error) {
	// 先从缓存获取
	cacheKey := fmt.Sprintf("post:%d", id)
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		// 这里简化处理，实际项目中需要反序列化
	}

	// 从数据库获取
	post, err := s.postRepo.GetByIDWithRelations(id)
	if err != nil {
		return nil, err
	}

	// 权限检查
	if !post.CanView(user) {
		return nil, fmt.Errorf("权限不足")
	}

	// 更新浏览量（异步）
	go func() {
		if err := s.postRepo.UpdateViewCount(id); err != nil {
			logger.Errorf("更新文章浏览量失败(ID: %d): %v", id, err)
		}
	}()

	// 缓存文章（5分钟）
	cache.Set(ctx, cacheKey, post.ToSafeJSON(), 5*time.Minute)

	return post, nil
}

// DeletePost 删除文章
func (s *PostService) DeletePost(ctx context.Context, id int, user *model.User) error {
	// 获取文章
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 权限检查
	if !post.CanDelete(user) {
		return nil, fmt.Errorf("权限不足")
	}

	// 删除文章
	if err := s.postRepo.Delete(id); err != nil {
		return fmt.Errorf("删除文章失败: %w", err)
	}

	// 清除相关缓存
	s.clearPostCache()

	logger.Infof("文章删除成功: %s (ID: %d, 操作者: %s)", post.Title, post.ID, user.Username)

	return nil
}

// ListPosts 获取文章列表
func (s *PostService) ListPosts(ctx context.Context, page, size int, filters *repository.PostFilters) (*PostListResponse, error) {
	posts, total, err := s.postRepo.List(page, size, filters)
	if err != nil {
		return nil, err
	}

	pagination := repository.NewPagination(page, size, total)

	return &PostListResponse{
		Posts:      posts,
		Pagination: pagination,
	}, nil
}

// ListPublishedPosts 获取已发布文章列表
func (s *PostService) ListPublishedPosts(ctx context.Context, page, size int, filters *repository.PostFilters) (*PostListResponse, error) {
	posts, total, err := s.postRepo.ListPublished(page, size, filters)
	if err != nil {
		return nil, err
	}

	pagination := repository.NewPagination(page, size, total)

	return &PostListResponse{
		Posts:      posts,
		Pagination: pagination,
	}, nil
}

// SearchPosts 搜索文章
func (s *PostService) SearchPosts(ctx context.Context, keyword string, page, size int) (*PostListResponse, error) {
	posts, total, err := s.postRepo.Search(keyword, page, size)
	if err != nil {
		return nil, err
	}

	pagination := repository.NewPagination(page, size, total)

	return &PostListResponse{
		Posts:      posts,
		Pagination: pagination,
	}, nil
}

// PublishPost 发布文章
func (s *PostService) PublishPost(ctx context.Context, id int, user *model.User) error {
	// 获取文章
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 权限检查
	if !post.CanEdit(user) {
		return fmt.Errorf("权限不足")
	}

	// 发布文章
	if err := s.postRepo.Publish(id); err != nil {
		return fmt.Errorf("发布文章失败: %w", err)
	}

	// 清除相关缓存
	s.clearPostCache()

	logger.Infof("文章发布成功: %s (ID: %d, 操作者: %s)", post.Title, post.ID, user.Username)

	return nil
}

// UnpublishPost 取消发布文章
func (s *PostService) UnpublishPost(ctx context.Context, id int, user *model.User) error {
	// 获取文章
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 权限检查
	if !post.CanEdit(user) {
		return fmt.Errorf("权限不足")
	}

	// 取消发布
	if err := s.postRepo.Unpublish(id); err != nil {
		return fmt.Errorf("取消发布失败: %w", err)
	}

	// 清除相关缓存
	s.clearPostCache()

	logger.Infof("文章取消发布成功: %s (ID: %d, 操作者: %s)", post.Title, post.ID, user.Username)

	return nil
}

// GetRelatedPosts 获取相关文章
func (s *PostService) GetRelatedPosts(ctx context.Context, postID int, limit int) ([]*model.Post, error) {
	return s.postRepo.GetRelatedPosts(postID, limit)
}

// GetRecentPosts 获取最新文章
func (s *PostService) GetRecentPosts(ctx context.Context, limit int) ([]*model.Post, error) {
	return s.postRepo.GetRecentPosts(limit)
}

// validatePostContent 验证文章内容
func (s *PostService) validatePostContent(title, content string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("标题不能为空")
	}

	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("内容不能为空")
	}

	// 随念内容长度限制
	if len([]rune(content)) > 300 {
		return fmt.Errorf("随念内容不能超过300字")
	}

	return nil
}

// generateSlug 生成URL友好的slug
func (s *PostService) generateSlug(title string) string {
	if title == "" {
		return "untitled"
	}

	// URL编码
	slug := url.QueryEscape(title)

	// 替换特殊字符
	replacer := strings.NewReplacer(
		"+", "-", "%20", "-", ".", "", ",", "", ":", "",
		";", "", "!", "", "(", "", ")", "", "[", "", "]", "",
		"{", "", "}", "", "@", "", "#", "", "$", "", "%", "",
		"^", "", "&", "", "*", "", "_", "-", "+", "-",
		"=", "", "~", "", "`", "", "|", "", "\\", "",
	)

	slug = replacer.Replace(slug)

	// 移除连续的连字符
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// 移除首尾连字符
	slug = strings.Trim(slug, "-")

	// 确保不为空
	if slug == "" {
		slug = "untitled"
	}

	return slug
}

// generateUniqueSlug 生成唯一的slug
func (s *PostService) generateUniqueSlug(title string) string {
	baseSlug := s.generateSlug(title)
	slug := baseSlug
	counter := 1

	for {
		// 检查slug是否存在
		if _, err := s.postRepo.GetBySlug(slug); err != nil {
			// 不存在，可以使用
			break
		}

		// 存在，添加数字后缀
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	return slug
}

// associatePostTags 关联文章标签
func (s *PostService) associatePostTags(postID int, tags []*model.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	// 批量创建关联记录
	postTags := make([]model.PostTag, len(tags))
	for i, tag := range tags {
		postTags[i] = model.PostTag{
			PostID: postID,
			TagID:  tag.ID,
		}
	}

	return s.postRepo.(*repository.postRepository).db.Create(&postTags).Error
}

// updatePostTags 更新文章标签关联
func (s *PostService) updatePostTags(postID int, tags []*model.Tag) error {
	// 删除现有关联
	if err := s.postRepo.(*repository.postRepository).db.Where("post_id = ?", postID).Delete(&model.PostTag{}).Error; err != nil {
		return fmt.Errorf("删除现有标签关联失败: %w", err)
	}

	// 创建新关联
	return s.associatePostTags(postID, tags)
}

// clearPostCache 清除文章相关缓存
func (s *PostService) clearPostCache() {
	ctx := context.Background()
	// 清除文章列表缓存
	cache.Delete(ctx, "posts:published:*")
	// 清除标签缓存
	cache.Delete(ctx, "tags:*")
	// 清除分类缓存
	cache.Delete(ctx, "categories:*")
}