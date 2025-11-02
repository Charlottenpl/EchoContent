package repository

import (
	"fmt"
	"strings"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/gorm"
)

// TagRepository 标签仓库接口
type TagRepository interface {
	Create(tag *model.Tag) error
	GetByID(id int) (*model.Tag, error)
	GetBySlug(slug string) (*model.Tag, error)
	GetByName(name string) (*model.Tag, error)
	Update(tag *model.Tag) error
	Delete(id int) error
	List() ([]*model.Tag, error)
	GetByNames(names []string) ([]*model.Tag, error)
	GetOrCreateTags(names []string) ([]*model.Tag, error)
	GetWithPostCount() ([]*TagWithCount, error)
	GetPopularTags(limit int) ([]*TagWithCount, error)
	CheckExistsByName(name string) (bool, error)
	CheckExistsBySlug(slug string) (bool, error)
}

// TagWithCount 带文章数量的标签
type TagWithCount struct {
	model.Tag
	PostCount int `json:"post_count"`
}

// tagRepository 标签仓库实现
type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository 创建标签仓库实例
func NewTagRepository() TagRepository {
	return &tagRepository{
		db: database.GetDB(),
	}
}

// Create 创建标签
func (r *tagRepository) Create(tag *model.Tag) error {
	// 生成slug
	if tag.Slug == "" {
		tag.Slug = r.generateSlug(tag.Name)
	}

	if err := r.db.Create(tag).Error; err != nil {
		logger.Errorf("创建标签失败: %v", err)
		return fmt.Errorf("创建标签失败: %w", err)
	}

	logger.Infof("标签创建成功: %s (ID: %d)", tag.Name, tag.ID)
	return nil
}

// GetByID 根据ID获取标签
func (r *tagRepository) GetByID(id int) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.First(&tag, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("标签不存在")
		}
		logger.Errorf("获取标签失败(ID: %d): %v", id, err)
		return nil, fmt.Errorf("获取标签失败: %w", err)
	}

	return &tag, nil
}

// GetBySlug 根据slug获取标签
func (r *tagRepository) GetBySlug(slug string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("slug = ?", slug).First(&tag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("标签不存在")
		}
		logger.Errorf("获取标签失败(slug: %s): %v", slug, err)
		return nil, fmt.Errorf("获取标签失败: %w", err)
	}

	return &tag, nil
}

// GetByName 根据名称获取标签
func (r *tagRepository) GetByName(name string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.Where("name = ?", name).First(&tag).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("标签不存在")
		}
		logger.Errorf("获取标签失败(name: %s): %v", name, err)
		return nil, fmt.Errorf("获取标签失败: %w", err)
	}

	return &tag, nil
}

// Update 更新标签
func (r *tagRepository) Update(tag *model.Tag) error {
	// 生成slug（如果为空）
	if tag.Slug == "" {
		tag.Slug = r.generateSlug(tag.Name)
	}

	if err := r.db.Save(tag).Error; err != nil {
		logger.Errorf("更新标签失败(ID: %d): %v", tag.ID, err)
		return fmt.Errorf("更新标签失败: %w", err)
	}

	logger.Infof("标签更新成功: %s (ID: %d)", tag.Name, tag.ID)
	return nil
}

// Delete 删除标签
func (r *tagRepository) Delete(id int) error {
	// 检查是否有文章使用该标签
	var count int64
	if err := r.db.Table("post_tags").Where("tag_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查标签使用情况失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("该标签还被文章使用，无法删除")
	}

	if err := r.db.Delete(&model.Tag{}, id).Error; err != nil {
		logger.Errorf("删除标签失败(ID: %d): %v", id, err)
		return fmt.Errorf("删除标签失败: %w", err)
	}

	logger.Infof("标签删除成功 (ID: %d)", id)
	return nil
}

// List 获取所有标签
func (r *tagRepository) List() ([]*model.Tag, error) {
	var tags []*model.Tag
	if err := r.db.Order("name ASC").Find(&tags).Error; err != nil {
		logger.Errorf("获取标签列表失败: %v", err)
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}

	return tags, nil
}

// GetByNames 根据名称列表获取标签
func (r *tagRepository) GetByNames(names []string) ([]*model.Tag, error) {
	if len(names) == 0 {
		return []*model.Tag{}, nil
	}

	var tags []*model.Tag
	if err := r.db.Where("name IN ?", names).Find(&tags).Error; err != nil {
		logger.Errorf("根据名称获取标签失败: %v", err)
		return nil, fmt.Errorf("根据名称获取标签失败: %w", err)
	}

	return tags, nil
}

// GetOrCreateTags 根据名称获取或创建标签
func (r *tagRepository) GetOrCreateTags(names []string) ([]*model.Tag, error) {
	if len(names) == 0 {
		return []*model.Tag{}, nil
	}

	// 去重
	uniqueNames := make(map[string]bool)
	for _, name := range names {
		if strings.TrimSpace(name) != "" {
			uniqueNames[strings.TrimSpace(name)] = true
		}
	}

	if len(uniqueNames) == 0 {
		return []*model.Tag{}, nil
	}

	// 转换为切片
	nameList := make([]string, 0, len(uniqueNames))
	for name := range uniqueNames {
		nameList = append(nameList, name)
	}

	// 查找已存在的标签
	existingTags, err := r.GetByNames(nameList)
	if err != nil {
		return nil, err
	}

	// 创建已存在标签的映射
	existingMap := make(map[string]*model.Tag)
	for _, tag := range existingTags {
		existingMap[tag.Name] = tag
	}

	// 找出需要创建的标签
	var tags []*model.Tag
	for _, name := range nameList {
		if tag, exists := existingMap[name]; exists {
			tags = append(tags, tag)
		} else {
			// 创建新标签
			newTag := &model.Tag{
				Name: name,
				Slug: r.generateSlug(name),
			}

			if err := r.Create(newTag); err != nil {
				logger.Errorf("创建标签失败(%s): %v", name, err)
				continue
			}

			tags = append(tags, newTag)
		}
	}

	return tags, nil
}

// GetWithPostCount 获取带文章数量的标签
func (r *tagRepository) GetWithPostCount() ([]*TagWithCount, error) {
	var results []*TagWithCount

	query := `
		SELECT t.*, COUNT(pt.post_id) as post_count
		FROM tags t
		LEFT JOIN post_tags pt ON t.id = pt.tag_id
		LEFT JOIN posts p ON pt.post_id = p.id AND p.status = 'published'
		GROUP BY t.id
		ORDER BY t.name ASC
	`

	if err := r.db.Raw(query).Scan(&results).Error; err != nil {
		logger.Errorf("获取带文章数量的标签失败: %v", err)
		return nil, fmt.Errorf("获取带文章数量的标签失败: %w", err)
	}

	return results, nil
}

// GetPopularTags 获取热门标签
func (r *tagRepository) GetPopularTags(limit int) ([]*TagWithCount, error) {
	var results []*TagWithCount

	query := `
		SELECT t.*, COUNT(pt.post_id) as post_count
		FROM tags t
		LEFT JOIN post_tags pt ON t.id = pt.tag_id
		LEFT JOIN posts p ON pt.post_id = p.id AND p.status = 'published'
		GROUP BY t.id
		HAVING post_count > 0
		ORDER BY post_count DESC, t.name ASC
		LIMIT ?
	`

	if err := r.db.Raw(query, limit).Scan(&results).Error; err != nil {
		logger.Errorf("获取热门标签失败: %v", err)
		return nil, fmt.Errorf("获取热门标签失败: %w", err)
	}

	return results, nil
}

// CheckExistsByName 检查标签名称是否存在
func (r *tagRepository) CheckExistsByName(name string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Tag{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

// CheckExistsBySlug 检查标签slug是否存在
func (r *tagRepository) CheckExistsBySlug(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Tag{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

// generateSlug 生成URL友好的slug
func (r *tagRepository) generateSlug(name string) string {
	if name == "" {
		return "tag"
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
		slug = "tag"
	}

	return slug
}