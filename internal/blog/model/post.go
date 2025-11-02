package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Post 文章/随念模型
type Post struct {
	ID            int       `json:"id" gorm:"primaryKey"`
	Title         string    `json:"title" gorm:"size:200"`
	Slug          string    `json:"slug" gorm:"uniqueIndex;size:200"`
	Content       string    `json:"content" gorm:"type:text;not null"`
	Excerpt       string    `json:"excerpt" gorm:"type:text"`
	Type          string    `json:"type" gorm:"size:20;not null;index;default:blog"` // blog, moment
	Status        string    `json:"status" gorm:"size:20;not null;index;default:draft"` // published, draft, private
	AuthorID      int       `json:"author_id" gorm:"not null;index"`
	CategoryID    *int      `json:"category_id" gorm:"index"`
	FeaturedImage string    `json:"featured_image" gorm:"size:255"`
	ViewCount     int       `json:"view_count" gorm:"default:0"`
	LikeCount     int       `json:"like_count" gorm:"default:0"`
	CommentCount  int       `json:"comment_count" gorm:"default:0"`
	IsTop         bool      `json:"is_top" gorm:"default:false"`
	PublishedAt   *time.Time `json:"published_at" gorm:"index"`
	CreatedAt     time.Time `json:"created_at" gorm:"index"`
	UpdatedAt     time.Time `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Author    User            `json:"author" gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`
	Category  *Category       `json:"category" gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL"`
	Tags      []Tag           `json:"tags" gorm:"many2many:post_tags;constraint:OnDelete:CASCADE"`
	Comments  []Comment       `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Likes     []Like          `json:"-" gorm:"foreignKey:TargetID;constraint:OnDelete:CASCADE"`
	Favorites []Favorite      `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

// Category 分类模型
type Category struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:50;not null;uniqueIndex"`
	Slug        string    `json:"slug" gorm:"size:50;not null;uniqueIndex"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Posts []Post `json:"-" gorm:"foreignKey:CategoryID;constraint:OnDelete:SET NULL"`
}

// Tag 标签模型
type Tag struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:30;not null;uniqueIndex"`
	Slug      string    `json:"slug" gorm:"size:30;not null;uniqueIndex"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Posts []Post `json:"-" gorm:"many2many:post_tags;constraint:OnDelete:CASCADE"`
}

// PostTag 文章标签关联模型
type PostTag struct {
	PostID int `json:"post_id" gorm:"primaryKey;autoIncrement:false"`
	TagID  int `json:"tag_id" gorm:"primaryKey;autoIncrement:false"`

	// 关联关系
	Post Post `json:"-" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Tag  Tag  `json:"-" gorm:"foreignKey:TagID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (Post) TableName() string {
	return "posts"
}

// TableName 指定表名
func (Category) TableName() string {
	return "categories"
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}

// TableName 指定表名
func (PostTag) TableName() string {
	return "post_tags"
}

// BeforeCreate GORM钩子：创建前
func (p *Post) BeforeCreate(tx *gorm.DB) error {
	if p.Type == "" {
		p.Type = "blog"
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	return nil
}

// BeforeSave GORM钩子：保存前
func (p *Post) BeforeSave(tx *gorm.DB) error {
	// 生成摘要（如果为空）
	if p.Excerpt == "" && p.Content != "" {
		p.Excerpt = p.generateExcerpt()
	}

	// 生成slug（如果为空）
	if p.Slug == "" && p.Title != "" {
		p.Slug = p.generateSlug()
	}

	// 如果状态是发布且发布时间为空，设置发布时间
	if p.Status == "published" && p.PublishedAt == nil {
		now := time.Now()
		p.PublishedAt = &now
	}

	return nil
}

// IsPublished 检查是否已发布
func (p *Post) IsPublished() bool {
	return p.Status == "published"
}

// IsDraft 检查是否为草稿
func (p *Post) IsDraft() bool {
	return p.Status == "draft"
}

// IsPrivate 检查是否为私密
func (p *Post) IsPrivate() bool {
	return p.Status == "private"
}

// IsBlog 检查是否为博客文章
func (p *Post) IsBlog() bool {
	return p.Type == "blog"
}

// IsMoment 检查是否为随念
func (p *Post) IsMoment() bool {
	return p.Type == "moment"
}

// CanView 检查用户是否可以查看
func (p *Post) CanView(user *User) bool {
	// 管理员可以查看所有内容
	if user != nil && user.IsAdmin() {
		return true
	}

	// 已发布的内容可以查看
	if p.IsPublished() {
		return true
	}

	// 作者可以查看自己的草稿和私密内容
	if user != nil && p.AuthorID == user.ID {
		return true
	}

	return false
}

// CanEdit 检查用户是否可以编辑
func (p *Post) CanEdit(user *User) bool {
	if user == nil {
		return false
	}

	// 管理员可以编辑所有内容
	if user.IsAdmin() {
		return true
	}

	// 作者可以编辑自己的内容
	return p.AuthorID == user.ID
}

// CanDelete 检查用户是否可以删除
func (p *Post) CanDelete(user *User) bool {
	return p.CanEdit(user)
}

// GetTagNames 获取标签名称列表
func (p *Post) GetTagNames() []string {
	if len(p.Tags) == 0 {
		return []string{}
	}

	names := make([]string, len(p.Tags))
	for i, tag := range p.Tags {
		names[i] = tag.Name
	}
	return names
}

// GetTagSlugs 获取标签slug列表
func (p *Post) GetTagSlugs() []string {
	if len(p.Tags) == 0 {
		return []string{}
	}

	slugs := make([]string, len(p.Tags))
	for i, tag := range p.Tags {
		slugs[i] = tag.Slug
	}
	return slugs
}

// generateExcerpt 生成文章摘要
func (p *Post) generateExcerpt() string {
	if p.Content == "" {
		return ""
	}

	content := strings.TrimSpace(p.Content)
	if len(content) <= 200 {
		return content
	}

	// 尝试在句子边界截断
	sentences := strings.Split(content, "。")
	if len(sentences) > 1 {
		excerpt := sentences[0] + "。"
		if len([]rune(excerpt)) <= 200 {
			return excerpt
		}
	}

	// 按字符截断
	runes := []rune(content)
	if len(runes) > 200 {
		return string(runes[:200]) + "..."
	}
	return content
}

// generateSlug 生成URL友好的slug
func (p *Post) generateSlug() string {
	if p.Title == "" {
		return ""
	}

	// 简单的slug生成，实际项目中可以使用更复杂的算法
	slug := strings.ToLower(p.Title)

	// 替换空格为连字符
	slug = strings.ReplaceAll(slug, " ", "-")

	// 移除特殊字符
	replacer := strings.NewReplacer(
		"?", "", "！", "", "。", "", "，", "", "、", "",
		"：", "", "；", "", "（", "", "）", "", "【", "",
		"】", "", "\"", "", "'", "", "《", "", "》", "",
	)
	slug = replacer.Replace(slug)

	// 确保不为空
	if slug == "" {
		slug = "untitled"
	}

	return slug
}