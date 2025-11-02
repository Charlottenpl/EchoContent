package model

import (
	"time"

	"github.com/charlottepl/blog-system/internal/blog/model"
)

// Moment 随念模型
// 随念本质上是特殊的Post，为了代码组织，创建这个结构体
type Moment struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Content     string    `json:"content"`
	Excerpt     string    `json:"excerpt"`
	Type        string    `json:"type"` // 固定为 "moment"
	Status      string    `json:"status"`
	AuthorID    int       `json:"author_id"`
	CategoryID  *int      `json:"category_id"`
	ViewCount   int       `json:"view_count"`
	LikeCount   int       `json:"like_count"`
	CommentCount int      `json:"comment_count"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联关系
	Author       model.User                 `json:"author"`
	Category     *model.Category            `json:"category"`
	Tags         []model.Tag                `json:"tags"`
	Comments     []model.Comment            `json:"comments"`
	MediaFiles   []model.MediaFile          `json:"media_files"`
}

// MomentMedia 随念媒体文件关联模型
type MomentMedia struct {
	ID       int             `json:"id" gorm:"primaryKey"`
	MomentID int             `json:"moment_id" gorm:"not null;index"`
	MediaID  int             `json:"media_id" gorm:"not null;index"`
	Sort     int             `json:"sort" gorm:"default:0"`
	Media    model.MediaFile `json:"media" gorm:"foreignKey:MediaID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (MomentMedia) TableName() string {
	return "moment_medias"
}

// NewMoment 从Post创建Moment
func NewMoment(post *model.Post) *Moment {
	if post == nil || !post.IsMoment() {
		return nil
	}

	return &Moment{
		ID:          post.ID,
		Title:       post.Title,
		Slug:        post.Slug,
		Content:     post.Content,
		Excerpt:     post.Excerpt,
		Type:        post.Type,
		Status:      post.Status,
		AuthorID:    post.AuthorID,
		CategoryID:  post.CategoryID,
		ViewCount:   post.ViewCount,
		LikeCount:   post.LikeCount,
		CommentCount: post.CommentCount,
		PublishedAt: post.PublishedAt,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
		Author:      post.Author,
		Category:    post.Category,
		Tags:        post.Tags,
		Comments:    post.Comments,
	}
}

// ToPost 转换为Post
func (m *Moment) ToPost() *model.Post {
	if m == nil {
		return nil
	}

	return &model.Post{
		ID:          m.ID,
		Title:       m.Title,
		Slug:        m.Slug,
		Content:     m.Content,
		Excerpt:     m.Excerpt,
		Type:        "moment",
		Status:      m.Status,
		AuthorID:    m.AuthorID,
		CategoryID:  m.CategoryID,
		ViewCount:   m.ViewCount,
		LikeCount:   m.LikeCount,
		CommentCount: m.CommentCount,
		PublishedAt: m.PublishedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		Author:      m.Author,
		Category:    m.Category,
		Tags:        m.Tags,
		Comments:    m.Comments,
	}
}

// IsPublished 检查是否已发布
func (m *Moment) IsPublished() bool {
	return m.Status == "published"
}

// IsDraft 检查是否为草稿
func (m *Moment) IsDraft() bool {
	return m.Status == "draft"
}

// CanView 检查用户是否可以查看
func (m *Moment) CanView(user *model.User) bool {
	// 管理员可以查看所有内容
	if user != nil && user.IsAdmin() {
		return true
	}

	// 已发布的内容可以查看
	if m.IsPublished() {
		return true
	}

	// 作者可以查看自己的草稿和私密内容
	if user != nil && m.AuthorID == user.ID {
		return true
	}

	return false
}

// CanEdit 检查用户是否可以编辑
func (m *Moment) CanEdit(user *model.User) bool {
	if user == nil {
		return false
	}

	// 管理员可以编辑所有内容
	if user.IsAdmin() {
		return true
	}

	// 作者可以编辑自己的内容
	return m.AuthorID == user.ID
}

// CanDelete 检查用户是否可以删除
func (m *Moment) CanDelete(user *model.User) bool {
	return m.CanEdit(user)
}

// GetTagNames 获取标签名称列表
func (m *Moment) GetTagNames() []string {
	if len(m.Tags) == 0 {
		return []string{}
	}

	names := make([]string, len(m.Tags))
	for i, tag := range m.Tags {
		names[i] = tag.Name
	}
	return names
}

// GetTagSlugs 获取标签slug列表
func (m *Moment) GetTagSlugs() []string {
	if len(m.Tags) == 0 {
		return []string{}
	}

	slugs := make([]string, len(m.Tags))
	for i, tag := range m.Tags {
		slugs[i] = tag.Slug
	}
	return slugs
}