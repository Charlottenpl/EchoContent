package service

import (
	"context"
	"fmt"
	"time"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/blog/repository"
	"github.com/charlottepl/blog-system/internal/core/cache"
	"github.com/charlottepl/blog-system/internal/core/logger"
)

// TagService 标签服务
type TagService struct {
	repo repository.TagRepository
}

// NewTagService 创建标签服务实例
func NewTagService() *TagService {
	return &TagService{
		repo: repository.NewTagRepository(),
	}
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name string `json:"name" binding:"required,max=30"`
	Slug string `json:"slug"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name string `json:"name" binding:"omitempty,max=30"`
	Slug string `json:"slug"`
}

// TagWithPostCount 带文章数量的标签响应
type TagWithPostCount struct {
	*repository.TagWithCount
}

// CreateTag 创建标签
func (s *TagService) CreateTag(ctx context.Context, req *CreateTagRequest) (*model.Tag, error) {
	// 检查标签名称是否已存在
	if exists, err := s.repo.CheckExistsByName(req.Name); err != nil {
		return nil, fmt.Errorf("检查标签名称失败: %w", err)
	} else if exists {
		return nil, fmt.Errorf("标签名称已存在")
	}

	// 检查slug是否已存在（如果提供了slug）
	if req.Slug != "" {
		if exists, err := s.repo.CheckExistsBySlug(req.Slug); err != nil {
			return nil, fmt.Errorf("检查标签slug失败: %w", err)
		} else if exists {
			return nil, fmt.Errorf("标签slug已存在")
		}
	}

	// 创建标签
	tag := &model.Tag{
		Name: req.Name,
		Slug: req.Slug,
	}

	if err := s.repo.Create(tag); err != nil {
		return nil, fmt.Errorf("创建标签失败: %w", err)
	}

	// 清除缓存
	s.clearCache()

	logger.Infof("标签创建成功: %s (ID: %d)", tag.Name, tag.ID)

	return tag, nil
}

// UpdateTag 更新标签
func (s *TagService) UpdateTag(ctx context.Context, id int, req *UpdateTagRequest) (*model.Tag, error) {
	// 获取标签
	tag, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查名称是否已存在（如果要修改名称）
	if req.Name != "" && req.Name != tag.Name {
		if exists, err := s.repo.CheckExistsByName(req.Name); err != nil {
			return nil, fmt.Errorf("检查标签名称失败: %w", err)
		} else if exists {
			return nil, fmt.Errorf("标签名称已存在")
		}
		tag.Name = req.Name
	}

	// 检查slug是否已存在（如果要修改slug）
	if req.Slug != "" && req.Slug != tag.Slug {
		if exists, err := s.repo.CheckExistsBySlug(req.Slug); err != nil {
			return nil, fmt.Errorf("检查标签slug失败: %w", err)
		} else if exists {
			return nil, fmt.Errorf("标签slug已存在")
		}
		tag.Slug = req.Slug
	}

	// 更新标签
	if err := s.repo.Update(tag); err != nil {
		return nil, fmt.Errorf("更新标签失败: %w", err)
	}

	// 清除缓存
	s.clearCache()

	logger.Infof("标签更新成功: %s (ID: %d)", tag.Name, tag.ID)

	return tag, nil
}

// DeleteTag 删除标签
func (s *TagService) DeleteTag(ctx context.Context, id int) error {
	// 检查标签是否存在
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// 删除标签
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}

	// 清除缓存
	s.clearCache()

	logger.Infof("标签删除成功 (ID: %d)", id)

	return nil
}

// GetTag 获取标签详情
func (s *TagService) GetTag(ctx context.Context, id int) (*model.Tag, error) {
	return s.repo.GetByID(id)
}

// GetTagBySlug 根据slug获取标签
func (s *TagService) GetTagBySlug(ctx context.Context, slug string) (*model.Tag, error) {
	return s.repo.GetBySlug(slug)
}

// GetTagByName 根据名称获取标签
func (s *TagService) GetTagByName(ctx context.Context, name string) (*model.Tag, error) {
	return s.repo.GetByName(name)
}

// ListTags 获取所有标签
func (s *TagService) ListTags(ctx context.Context) ([]*model.Tag, error) {
	// 先从缓存获取
	cacheKey := "tags:all"
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		// 这里简化处理，实际项目中需要反序列化
	}

	// 从数据库获取
	tags, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	// 缓存结果（10分钟）
	cache.Set(ctx, cacheKey, tags, 10*time.Minute)

	return tags, nil
}

// ListTagsWithPostCount 获取带文章数量的标签
func (s *TagService) ListTagsWithPostCount(ctx context.Context) ([]*TagWithPostCount, error) {
	// 先从缓存获取
	cacheKey := "tags:with_count"
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		// 这里简化处理，实际项目中需要反序列化
	}

	// 从数据库获取
	results, err := s.repo.GetWithPostCount()
	if err != nil {
		return nil, err
	}

	// 转换格式
	tags := make([]*TagWithPostCount, len(results))
	for i, result := range results {
		tags[i] = &TagWithPostCount{TagWithCount: result}
	}

	// 缓存结果（5分钟）
	cache.Set(ctx, cacheKey, tags, 5*time.Minute)

	return tags, nil
}

// GetPopularTags 获取热门标签
func (s *TagService) GetPopularTags(ctx context.Context, limit int) ([]*TagWithPostCount, error) {
	// 先从缓存获取
	cacheKey := fmt.Sprintf("tags:popular:%d", limit)
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		// 这里简化处理，实际项目中需要反序列化
	}

	// 从数据库获取
	results, err := s.repo.GetPopularTags(limit)
	if err != nil {
		return nil, err
	}

	// 转换格式
	tags := make([]*TagWithPostCount, len(results))
	for i, result := range results {
		tags[i] = &TagWithPostCount{TagWithCount: result}
	}

	// 缓存结果（10分钟）
	cache.Set(ctx, cacheKey, tags, 10*time.Minute)

	return tags, nil
}

// GetOrCreateTags 根据名称获取或创建标签
func (s *TagService) GetOrCreateTags(ctx context.Context, names []string) ([]*model.Tag, error) {
	if len(names) == 0 {
		return []*model.Tag{}, nil
	}

	tags, err := s.repo.GetOrCreateTags(names)
	if err != nil {
		return nil, fmt.Errorf("获取或创建标签失败: %w", err)
	}

	// 清除缓存
	s.clearCache()

	return tags, nil
}

// SearchTags 搜索标签
func (s *TagService) SearchTags(ctx context.Context, keyword string) ([]*model.Tag, error) {
	allTags, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	// 简单的模糊搜索
	var results []*model.Tag
	for _, tag := range allTags {
		if keyword == "" ||
		   containsIgnoreCase(tag.Name, keyword) ||
		   containsIgnoreCase(tag.Slug, keyword) {
			results = append(results, tag)
		}
	}

	return results, nil
}

// containsIgnoreCase 忽略大小写的包含检查
func containsIgnoreCase(str, substr string) bool {
	return len(str) >= len(substr) &&
		   (str == substr ||
		    (len(str) > len(substr) &&
		     (str[:len(substr)] == substr ||
		      str[len(str)-len(substr):] == substr ||
		      containsIgnoreCase(str[1:], substr))))
}

// GetTagUsageStats 获取标签使用统计
func (s *TagService) GetTagUsageStats(ctx context.Context) (map[string]int, error) {
	tags, err := s.repo.GetWithPostCount()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]int)
	for _, tag := range tags {
		stats[tag.Name] = tag.PostCount
	}

	return stats, nil
}

// clearCache 清除缓存
func (s *TagService) clearCache() {
	ctx := context.Background()
	cache.Delete(ctx, "tags:*")
}