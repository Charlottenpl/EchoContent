package service

import (
	"context"
	"fmt"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/blog/repository"
	"github.com/charlottepl/blog-system/internal/core/cache"
	"github.com/charlottepl/blog-system/internal/core/logger"
)

// CategoryService 分类服务
type CategoryService struct {
	repo repository.CategoryRepository
}

// NewCategoryService 创建分类服务实例
func NewCategoryService() *CategoryService {
	return &CategoryService{
		repo: repository.NewCategoryRepository(),
	}
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,max=50"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"omitempty,max=50"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// CategoryWithPostCount 带文章数量的分类响应
type CategoryWithPostCount struct {
	*repository.CategoryWithCount
}

// CreateCategory 创建分类
func (s *CategoryService) CreateCategory(ctx context.Context, req *CreateCategoryRequest) (*model.Category, error) {
	// 检查分类名称是否已存在
	if exists, err := s.repo.CheckExistsByName(req.Name); err != nil {
		return nil, fmt.Errorf("检查分类名称失败: %w", err)
	} else if exists {
		return nil, fmt.Errorf("分类名称已存在")
	}

	// 检查slug是否已存在（如果提供了slug）
	if req.Slug != "" {
		if exists, err := s.repo.CheckExistsBySlug(req.Slug); err != nil {
			return nil, fmt.Errorf("检查分类slug失败: %w", err)
		} else if exists {
			return nil, fmt.Errorf("分类slug已存在")
		}
	}

	// 创建分类
	category := &model.Category{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if err := s.repo.Create(category); err != nil {
		return nil, fmt.Errorf("创建分类失败: %w", err)
	}

	// 清除缓存
	s.clearCache()

	logger.Infof("分类创建成功: %s (ID: %d)", category.Name, category.ID)

	return category, nil
}

// UpdateCategory 更新分类
func (s *CategoryService) UpdateCategory(ctx context.Context, id int, req *UpdateCategoryRequest) (*model.Category, error) {
	// 获取分类
	category, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查名称是否已存在（如果要修改名称）
	if req.Name != "" && req.Name != category.Name {
		if exists, err := s.repo.CheckExistsByName(req.Name); err != nil {
			return nil, fmt.Errorf("检查分类名称失败: %w", err)
		} else if exists {
			return nil, fmt.Errorf("分类名称已存在")
		}
		category.Name = req.Name
	}

	// 检查slug是否已存在（如果要修改slug）
	if req.Slug != "" && req.Slug != category.Slug {
		if exists, err := s.repo.CheckExistsBySlug(req.Slug); err != nil {
			return nil, fmt.Errorf("检查分类slug失败: %w", err)
		} else if exists {
			return nil, fmt.Errorf("分类slug已存在")
		}
		category.Slug = req.Slug
	}

	if req.Description != "" {
		category.Description = req.Description
	}

	// 更新分类
	if err := s.repo.Update(category); err != nil {
		return nil, fmt.Errorf("更新分类失败: %w", err)
	}

	// 清除缓存
	s.clearCache()

	logger.Infof("分类更新成功: %s (ID: %d)", category.Name, category.ID)

	return category, nil
}

// DeleteCategory 删除分类
func (s *CategoryService) DeleteCategory(ctx context.Context, id int) error {
	// 检查分类是否存在
	_, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	// 删除分类
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("删除分类失败: %w", err)
	}

	// 清除缓存
	s.clearCache()

	logger.Infof("分类删除成功 (ID: %d)", id)

	return nil
}

// GetCategory 获取分类详情
func (s *CategoryService) GetCategory(ctx context.Context, id int) (*model.Category, error) {
	return s.repo.GetByID(id)
}

// GetCategoryBySlug 根据slug获取分类
func (s *CategoryService) GetCategoryBySlug(ctx context.Context, slug string) (*model.Category, error) {
	return s.repo.GetBySlug(slug)
}

// ListCategories 获取所有分类
func (s *CategoryService) ListCategories(ctx context.Context) ([]*model.Category, error) {
	// 先从缓存获取
	cacheKey := "categories:all"
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		// 这里简化处理，实际项目中需要反序列化
	}

	// 从数据库获取
	categories, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	// 缓存结果（10分钟）
	cache.Set(ctx, cacheKey, categories, 10*time.Minute)

	return categories, nil
}

// ListCategoriesWithPostCount 获取带文章数量的分类
func (s *CategoryService) ListCategoriesWithPostCount(ctx context.Context) ([]*CategoryWithPostCount, error) {
	// 先从缓存获取
	cacheKey := "categories:with_count"
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		// 这里简化处理，实际项目中需要反序列化
	}

	// 从数据库获取
	results, err := s.repo.GetWithPostCount()
	if err != nil {
		return nil, err
	}

	// 转换格式
	categories := make([]*CategoryWithPostCount, len(results))
	for i, result := range results {
		categories[i] = &CategoryWithPostCount{CategoryWithCount: result}
	}

	// 缓存结果（5分钟）
	cache.Set(ctx, cacheKey, categories, 5*time.Minute)

	return categories, nil
}

// EnsureDefaultCategory 确保默认分类存在
func (s *CategoryService) EnsureDefaultCategory(ctx context.Context) (*model.Category, error) {
	return s.repo.(*repository.categoryRepository).EnsureDefaultCategory()
}

// clearCache 清除缓存
func (s *CategoryService) clearCache() {
	ctx := context.Background()
	cache.Delete(ctx, "categories:*")
}