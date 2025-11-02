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
	"github.com/charlottepl/blog-system/internal/moment/model"
	"github.com/charlottepl/blog-system/internal/user/model"
	"gorm.io/gorm"
)

// MomentService 随念服务
type MomentService struct {
	momentRepo  repository.MomentRepository
	categoryRepo repository.CategoryRepository
	tagRepo     repository.TagRepository
}

// NewMomentService 创建随念服务实例
func NewMomentService() *MomentService {
	return &MomentService{
		momentRepo:  repository.NewMomentRepository(),
		categoryRepo: repository.NewCategoryRepository(),
		tagRepo:     repository.NewTagRepository(),
	}
}

// CreateMomentRequest 创建随念请求
type CreateMomentRequest struct {
	Content       string   `json:"content" binding:"required,max=300"`
	Title         string   `json:"title"`
	Excerpt       string   `json:"excerpt"`
	Status        string   `json:"status" binding:"omitempty,oneof=draft published private"`
	CategoryID    *int     `json:"category_id"`
	TagNames      []string `json:"tag_names"`
	FeaturedImage string   `json:"featured_image"`
}

// UpdateMomentRequest 更新随念请求
type UpdateMomentRequest struct {
	Content       string   `json:"content" binding:"omitempty,max=300"`
	Title         string   `json:"title"`
	Excerpt       string   `json:"excerpt"`
	Status        string   `json:"status" binding:"omitempty,oneof=draft published private"`
	CategoryID    *int     `json:"category_id"`
	TagNames      []string `json:"tag_names"`
	FeaturedImage string   `json:"featured_image"`
}

// MomentListResponse 随念列表响应
type MomentListResponse struct {
	Moments    []*model.Post           `json:"moments"`
	Pagination repository.Pagination   `json:"pagination"`
}

// CreateMoment 创建随念
func (s *MomentService) CreateMoment(ctx context.Context, req *CreateMomentRequest, author *model.User) (*model.Post, error) {
	// 权限检查（普通用户可以创建随念）
	if !author.CanCreatePost() {
		return nil, fmt.Errorf("权限不足，只有管理员可以创建随念")
	}

	// 内容验证
	if err := s.validateMomentContent(req.Content); err != nil {
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

	// 生成标题（如果为空）
	title := req.Title
	if title == "" {
		title = s.generateTitleFromContent(req.Content)
	}

	// 生成slug
	slug := s.generateSlug(title)
	if exists, _ := s.momentRepo.(*repository.momentRepository).checkSlugExists(slug); exists {
		slug = s.generateUniqueSlug(title)
	}

	// 创建随念
	moment := &model.Post{
		Title:         title,
		Slug:          slug,
		Content:       req.Content,
		Excerpt:       req.Excerpt,
		Type:          "moment",
		Status:        "draft", // 默认为草稿
		AuthorID:      author.ID,
		CategoryID:    req.CategoryID,
		FeaturedImage: req.FeaturedImage,
	}

	if req.Status != "" {
		moment.Status = req.Status
	}

	// 保存随念
	if err := s.momentRepo.Create(moment); err != nil {
		return nil, fmt.Errorf("创建随念失败: %w", err)
	}

	// 关联标签
	if len(tags) > 0 {
		if err := s.associateMomentTags(moment.ID, tags); err != nil {
			// 回滚随念创建
			s.momentRepo.Delete(moment.ID)
			return nil, fmt.Errorf("关联标签失败: %w", err)
		}
	}

	// 如果状态是发布，设置发布时间
	if moment.Status == "published" && moment.PublishedAt == nil {
		if err := s.momentRepo.Publish(moment.ID); err != nil {
			logger.Warnf("发布随念失败(ID: %d): %v", moment.ID, err)
		}
	}

	// 清除相关缓存
	s.clearMomentCache()

	logger.Infof("随念创建成功: %s (ID: %d, 作者: %s)", moment.Title, moment.ID, author.Username)

	return moment, nil
}

// UpdateMoment 更新随念
func (s *MomentService) UpdateMoment(ctx context.Context, id int, req *UpdateMomentRequest, user *model.User) (*model.Post, error) {
	// 获取随念
	moment, err := s.momentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 权限检查
	if !moment.CanEdit(user) {
		return nil, fmt.Errorf("权限不足")
	}

	// 内容验证
	if req.Content != "" {
		if err := s.validateMomentContent(req.Content); err != nil {
			return nil, err
		}
	}

	// 验证分类
	if req.CategoryID != nil {
		if _, err := s.categoryRepo.GetByID(*req.CategoryID); err != nil {
			return nil, fmt.Errorf("分类不存在")
		}
	}

	// 更新字段
	if req.Title != "" {
		moment.Title = req.Title
		// 重新生成slug
		newSlug := s.generateSlug(req.Title)
		if newSlug != moment.Slug {
			if exists, _ := s.momentRepo.(*repository.momentRepository).checkSlugExists(newSlug); exists {
				newSlug = s.generateUniqueSlug(req.Title)
			}
			moment.Slug = newSlug
		}
	}

	if req.Content != "" {
		moment.Content = req.Content
	}

	if req.Excerpt != "" {
		moment.Excerpt = req.Excerpt
	}

	if req.Status != "" {
		moment.Status = req.Status
	}

	if req.CategoryID != nil {
		moment.CategoryID = req.CategoryID
	}

	if req.FeaturedImage != "" {
		moment.FeaturedImage = req.FeaturedImage
	}

	// 处理标签
	if req.TagNames != nil {
		tags, err := s.tagRepo.GetOrCreateTags(req.TagNames)
		if err != nil {
			return nil, fmt.Errorf("处理标签失败: %w", err)
		}

		// 更新标签关联
		if err := s.updateMomentTags(moment.ID, tags); err != nil {
			return nil, fmt.Errorf("更新标签关联失败: %w", err)
		}
	}

	// 保存随念
	if err := s.momentRepo.Update(moment); err != nil {
		return nil, fmt.Errorf("更新随念失败: %w", err)
	}

	// 如果状态改为发布，设置发布时间
	if moment.Status == "published" && moment.PublishedAt == nil {
		if err := s.momentRepo.Publish(moment.ID); err != nil {
			logger.Warnf("发布随念失败(ID: %d): %v", moment.ID, err)
		}
	}

	// 清除相关缓存
	s.clearMomentCache()

	logger.Infof("随念更新成功: %s (ID: %d, 操作者: %s)", moment.Title, moment.ID, user.Username)

	return moment, nil
}

// GetMoment 获取随念详情
func (s *MomentService) GetMoment(ctx context.Context, id int, user *model.User) (*model.Post, error) {
	// 先从缓存获取
	cacheKey := fmt.Sprintf("moment:%d", id)
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		// 这里简化处理，实际项目中需要反序列化
	}

	// 从数据库获取
	moment, err := s.momentRepo.GetByIDWithRelations(id)
	if err != nil {
		return nil, err
	}

	// 权限检查
	if !moment.CanView(user) {
		return nil, fmt.Errorf("权限不足")
	}

	// 更新浏览量（异步）
	go func() {
		if err := s.momentRepo.UpdateViewCount(id); err != nil {
			logger.Errorf("更新随念浏览量失败(ID: %d): %v", id, err)
		}
	}()

	// 缓存随念（5分钟）
	cache.Set(ctx, cacheKey, moment.ToSafeJSON(), 5*time.Minute)

	return moment, nil
}

// DeleteMoment 删除随念
func (s *MomentService) DeleteMoment(ctx context.Context, id int, user *model.User) error {
	// 获取随念
	moment, err := s.momentRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 权限检查
	if !moment.CanDelete(user) {
		return nil, fmt.Errorf("权限不足")
	}

	// 删除随念
	if err := s.momentRepo.Delete(id); err != nil {
		return fmt.Errorf("删除随念失败: %w", err)
	}

	// 清除相关缓存
	s.clearMomentCache()

	logger.Infof("随念删除成功: %s (ID: %d, 操作者: %s)", moment.Title, moment.ID, user.Username)

	return nil
}

// ListMoments 获取随念列表
func (s *MomentService) ListMoments(ctx context.Context, page, size int, filters *repository.MomentFilters) (*MomentListResponse, error) {
	moments, total, err := s.momentRepo.List(page, size, filters)
	if err != nil {
		return nil, err
	}

	pagination := repository.NewPagination(page, size, total)

	return &MomentListResponse{
		Moments:    moments,
		Pagination: pagination,
	}, nil
}

// ListPublishedMoments 获取已发布随念列表
func (s *MomentService) ListPublishedMoments(ctx context.Context, page, size int) (*MomentListResponse, error) {
	moments, total, err := s.momentRepo.ListPublished(page, size)
	if err != nil {
		return nil, err
	}

	pagination := repository.NewPagination(page, size, total)

	return &MomentListResponse{
		Moments:    moments,
		Pagination: pagination,
	}, nil
}

// SearchMoments 搜索随念
func (s *MomentService) SearchMoments(ctx context.Context, keyword string, page, size int) (*MomentListResponse, error) {
	moments, total, err := s.momentRepo.Search(keyword, page, size)
	if err != nil {
		return nil, err
	}

	pagination := repository.NewPagination(page, size, total)

	return &MomentListResponse{
		Moments:    moments,
		Pagination: pagination,
	}, nil
}

// PublishMoment 发布随念
func (s *MomentService) PublishMoment(ctx context.Context, id int, user *model.User) error {
	// 获取随念
	moment, err := s.momentRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 权限检查
	if !moment.CanEdit(user) {
		return fmt.Errorf("权限不足")
	}

	// 发布随念
	if err := s.momentRepo.Publish(id); err != nil {
		return fmt.Errorf("发布随念失败: %w", err)
	}

	// 清除相关缓存
	s.clearMomentCache()

	logger.Infof("随念发布成功: %s (ID: %d, 操作者: %s)", moment.Title, moment.ID, user.Username)

	return nil
}

// UnpublishMoment 取消发布随念
func (s *MomentService) UnpublishMoment(ctx context.Context, id int, user *model.User) error {
	// 获取随念
	moment, err := s.momentRepo.GetByID(id)
	if err != nil {
		return err
	}

	// 权限检查
	if !moment.CanEdit(user) {
		return fmt.Errorf("权限不足")
	}

	// 取消发布
	if err := s.momentRepo.Unpublish(id); err != nil {
		return fmt.Errorf("取消发布失败: %w", err)
	}

	// 清除相关缓存
	s.clearMomentCache()

	logger.Infof("随念取消发布成功: %s (ID: %d, 操作者: %s)", moment.Title, moment.ID, user.Username)

	return nil
}

// GetRecentMoments 获取最新随念
func (s *MomentService) GetRecentMoments(ctx context.Context, limit int) ([]*model.Post, error) {
	return s.momentRepo.GetRecentMoments(limit)
}

// GetTrendingMoments 获取热门随念
func (s *MomentService) GetTrendingMoments(ctx context.Context, limit int) ([]*model.Post, error) {
	return s.momentRepo.GetTrendingMoments(limit)
}

// GetMomentsByAuthor 根据作者获取随念
func (s *MomentService) GetMomentsByAuthor(ctx context.Context, authorID int, page, size int) (*MomentListResponse, error) {
	moments, total, err := s.momentRepo.ListByAuthor(authorID, page, size)
	if err != nil {
		return nil, err
	}

	pagination := repository.NewPagination(page, size, total)

	return &MomentListResponse{
		Moments:    moments,
		Pagination: pagination,
	}, nil
}

// validateMomentContent 验证随念内容
func (s *MomentService) validateMomentContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("内容不能为空")
	}

	// 随念内容长度限制
	if len([]rune(content)) > 300 {
		return fmt.Errorf("随念内容不能超过300字")
	}

	return nil
}

// generateTitleFromContent 从内容生成标题
func (s *MomentService) generateTitleFromContent(content string) string {
	if content == "" {
		return "无标题"
	}

	// 移除多余的空格和换行
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")

	// 压缩连续空格
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	// 截取前50个字符作为标题
	runes := []rune(content)
	if len(runes) > 50 {
		content = string(runes[:50]) + "..."
	}

	return content
}

// generateSlug 生成URL友好的slug
func (s *MomentService) generateSlug(title string) string {
	if title == "" {
		return "moment"
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
		slug = "moment"
	}

	return slug
}

// generateUniqueSlug 生成唯一的slug
func (s *MomentService) generateUniqueSlug(title string) string {
	baseSlug := s.generateSlug(title)
	slug := baseSlug
	counter := 1

	for {
		// 检查slug是否存在
		if _, err := s.momentRepo.GetBySlug(slug); err != nil {
			// 不存在，可以使用
			break
		}

		// 存在，添加数字后缀
		slug = fmt.Sprintf("%s-%d", baseSlug, counter)
		counter++
	}

	return slug
}

// associateMomentTags 关联随念标签
func (s *MomentService) associateMomentTags(momentID int, tags []*model.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	// 批量创建关联记录
	postTags := make([]model.PostTag, len(tags))
	for i, tag := range tags {
		postTags[i] = model.PostTag{
			PostID: momentID,
			TagID:  tag.ID,
		}
	}

	return s.momentRepo.(*repository.momentRepository).db.Create(&postTags).Error
}

// updateMomentTags 更新随念标签关联
func (s *MomentService) updateMomentTags(momentID int, tags []*model.Tag) error {
	// 删除现有关联
	if err := s.momentRepo.(*repository.momentRepository).db.Where("post_id = ?", momentID).Delete(&model.PostTag{}).Error; err != nil {
		return fmt.Errorf("删除现有标签关联失败: %w", err)
	}

	// 创建新关联
	return s.associateMomentTags(momentID, tags)
}

// clearMomentCache 清除随念相关缓存
func (s *MomentService) clearMomentCache() {
	ctx := context.Background()
	// 清除随念列表缓存
	cache.Delete(ctx, "moments:*")
	// 清除标签缓存
	cache.Delete(ctx, "tags:*")
}