package repository

import (
	"fmt"
	"strings"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/gorm"
)

// CategoryRepository 分类仓库接口
type CategoryRepository interface {
	Create(category *model.Category) error
	GetByID(id int) (*model.Category, error)
	GetBySlug(slug string) (*model.Category, error)
	Update(category *model.Category) error
	Delete(id int) error
	List() ([]*model.Category, error)
	GetWithPostCount() ([]*CategoryWithCount, error)
	CheckExistsByName(name string) (bool, error)
	CheckExistsBySlug(slug string) (bool, error)
}

// CategoryWithCount 带文章数量的分类
type CategoryWithCount struct {
	model.Category
	PostCount int `json:"post_count"`
}

// categoryRepository 分类仓库实现
type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository 创建分类仓库实例
func NewCategoryRepository() CategoryRepository {
	return &categoryRepository{
		db: database.GetDB(),
	}
}

// Create 创建分类
func (r *categoryRepository) Create(category *model.Category) error {
	// 生成slug
	if category.Slug == "" {
		category.Slug = r.generateSlug(category.Name)
	}

	if err := r.db.Create(category).Error; err != nil {
		logger.Errorf("创建分类失败: %v", err)
		return fmt.Errorf("创建分类失败: %w", err)
	}

	logger.Infof("分类创建成功: %s (ID: %d)", category.Name, category.ID)
	return nil
}

// GetByID 根据ID获取分类
func (r *categoryRepository) GetByID(id int) (*model.Category, error) {
	var category model.Category
	err := r.db.First(&category, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分类不存在")
		}
		logger.Errorf("获取分类失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取分类失败: %w", err)
	}

	return &category, nil
}

// GetBySlug 根据slug获取分类
func (r *categoryRepository) GetBySlug(slug string) (*model.Category, error) {
	var category model.Category
	err := r.db.Where("slug = ?", slug).First(&category).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分类不存在")
		}
		logger.Errorf("获取分类失败(slug: %s): %v", slug, err)
		return nil, fmt.Errorf("获取分类失败: %w", err)
	}

	return &category, nil
}

// Update 更新分类
func (r *categoryRepository) Update(category *model.Category) error {
	// 生成slug（如果为空）
	if category.Slug == "" {
		category.Slug = r.generateSlug(category.Name)
	}

	if err := r.db.Save(category).Error; err != nil {
		logger.Errorf("更新分类失败(ID: %d): %v", category.ID, err)
		return fmt.Errorf("更新分类失败: %w", err)
	}

	logger.Infof("分类更新成功: %s (ID: %d)", category.Name, category.ID)
	return nil
}

// Delete 删除分类
func (r *categoryRepository) Delete(id int) error {
	// 检查是否有文章使用该分类
	var count int64
	if err := r.db.Model(&model.Post{}).Where("category_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查分类使用情况失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("该分类下还有文章，无法删除")
	}

	if err := r.db.Delete(&model.Category{}, id).Error; err != nil {
		logger.Errorf("删除分类失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除分类失败: %w", err)
	}

	logger.Infof("分类删除成功 (ID: %d)", id)
	return nil
}

// List 获取所有分类
func (r *categoryRepository) List() ([]*model.Category, error) {
	var categories []*model.Category
	if err := r.db.Order("name ASC").Find(&categories).Error; err != nil {
		logger.Errorf("获取分类列表失败: %v", err)
		return nil, fmt.Errorf("获取分类列表失败: %w", err)
	}

	return categories, nil
}

// GetWithPostCount 获取带文章数量的分类
func (r *categoryRepository) GetWithPostCount() ([]*CategoryWithCount, error) {
	var results []*CategoryWithCount

	query := `
		SELECT c.*, COUNT(p.id) as post_count
		FROM categories c
		LEFT JOIN posts p ON c.id = p.category_id AND p.status = 'published'
		GROUP BY c.id
		ORDER BY c.name ASC
	`

	if err := r.db.Raw(query).Scan(&results).Error; err != nil {
		logger.Errorf("获取带文章数量的分类失败: %v", err)
		return nil, fmt.Errorf("获取带文章数量的分类失败: %w", err)
	}

	return results, nil
}

// CheckExistsByName 检查分类名称是否存在
func (r *categoryRepository) CheckExistsByName(name string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Category{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

// CheckExistsBySlug 检查分类slug是否存在
func (r *categoryRepository) CheckExistsBySlug(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Category{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

// generateSlug 生成URL友好的slug
func (r *categoryRepository) generateSlug(name string) string {
	if name == "" {
		return "uncategorized"
	}

	// 转换为小写
	slug := strings.ToLower(name)

	// 替换空格为连字符
	slug = strings.ReplaceAll(slug, " ", "-")

	// 移除特殊字符
	replacer := strings.NewReplacer(
		"?", "", "！", "", "。", "", "，", "", "、", "",
		"：", "", "；", "", "（", "", "）", "", "【", "",
		"】", "", "\"", "", "'", "", "《", "", "》", "",
		"·", "", "—", "", "–", "", "…", "",
	)
	slug = replacer.Replace(slug)

	// 移除多余的连字符
	slug = strings.ReplaceAll(slug, "--", "-")
	slug = strings.Trim(slug, "-")

	// 确保不为空
	if slug == "" {
		slug = "category"
	}

	return slug
}

// EnsureDefaultCategory 确保默认分类存在
func (r *categoryRepository) EnsureDefaultCategory() (*model.Category, error) {
	// 查找"未分类"分类
	category, err := r.GetBySlug("uncategorized")
	if err == nil {
		return category, nil
	}

	// 创建默认分类
	defaultCategory := &model.Category{
		Name:        "未分类",
		Slug:        "uncategorized",
		Description: "未分类的文章",
	}

	if err := r.Create(defaultCategory); err != nil {
		return nil, fmt.Errorf("创建默认分类失败: %w", err)
	}

	return defaultCategory, nil
}