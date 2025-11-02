package model

import (
	"time"

	"gorm.io/gorm"
)

// Comment 评论模型
type Comment struct {
	ID         int       `json:"id" gorm:"primaryKey"`
	PostID     int       `json:"post_id" gorm:"not null;index"`
	AuthorID   *int      `json:"author_id" gorm:"index"`
	ParentID   *int      `json:"parent_id" gorm:"index"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	AuthorName string    `json:"author_name" gorm:"size:50"`
	AuthorEmail string   `json:"author_email" gorm:"size:100"`
	AuthorIP   string    `json:"-" gorm:"size:45"`
	Status     string    `json:"status" gorm:"size:20;not null;index;default:pending"` // approved, pending, spam, trash
	LikeCount  int       `json:"like_count" gorm:"default:0"`
	CreatedAt  time.Time `json:"created_at" gorm:"index"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Post     Post     `json:"post" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	Author   *User    `json:"author" gorm:"foreignKey:AuthorID;constraint:OnDelete:SET NULL"`
	Parent   *Comment `json:"parent" gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	Children []Comment `json:"children" gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	Likes    []Like    `json:"-" gorm:"foreignKey:TargetID;constraint:OnDelete:CASCADE"`
}

// Like 点赞模型
type Like struct {
	ID         int       `json:"id" gorm:"primaryKey"`
	UserID     int       `json:"user_id" gorm:"not null;index"`
	TargetType string    `json:"target_type" gorm:"size:20;not null;index"` // post, comment
	TargetID   int       `json:"target_id" gorm:"not null;index"`
	CreatedAt  time.Time `json:"created_at"`

	// 关联关系
	User     User      `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Post     *Post     `json:"-" gorm:"foreignKey:TargetID;constraint:OnDelete:CASCADE"`
	Comment  *Comment  `json:"-" gorm:"foreignKey:TargetID;constraint:OnDelete:CASCADE"`
}

// Favorite 收藏模型
type Favorite struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	UserID    int       `json:"user_id" gorm:"not null;index"`
	PostID    int       `json:"post_id" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at"`

	// 关联关系
	User User `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Post Post `json:"post" gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}

// MediaFile 媒体文件模型
type MediaFile struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	Filename     string    `json:"filename" gorm:"size:255;not null"`
	OriginalName string    `json:"original_name" gorm:"size:255;not null"`
	MimeType     string    `json:"mime_type" gorm:"size:100;not null"`
	Size         int64     `json:"size" gorm:"not null"`
	Path         string    `json:"path" gorm:"size:500;not null"`
	URL          string    `json:"url" gorm:"size:500"`
	UploaderID   *int      `json:"uploader_id" gorm:"index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Uploader *User `json:"uploader" gorm:"foreignKey:UploaderID;constraint:OnDelete:SET NULL"`
}

// SystemConfig 系统配置模型
type SystemConfig struct {
	Key         string    `json:"key" gorm:"primaryKey;size:100"`
	Value       string    `json:"value" gorm:"type:text"`
	Type        string    `json:"type" gorm:"size:20;default:string"` // string, number, boolean, json
	Description string    `json:"description" gorm:"type:text"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SystemLog 系统日志模型
type SystemLog struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Level     string    `json:"level" gorm:"size:20;not null;index"`
	Message   string    `json:"message" gorm:"type:text;not null"`
	Context   string    `json:"context" gorm:"type:json"` // 使用字符串存储JSON
	UserID    *int      `json:"user_id" gorm:"index"`
	IP        string    `json:"ip" gorm:"size:45"`
	UserAgent string    `json:"user_agent" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`

	// 关联关系
	User *User `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
}

// TableName 指定表名
func (Comment) TableName() string {
	return "comments"
}

// TableName 指定表名
func (Like) TableName() string {
	return "likes"
}

// TableName 指定表名
func (Favorite) TableName() string {
	return "favorites"
}

// TableName 指定表名
func (MediaFile) TableName() string {
	return "media_files"
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// TableName 指定表名
func (SystemLog) TableName() string {
	return "system_logs"
}

// BeforeCreate GORM钩子：创建前
func (c *Comment) BeforeCreate(tx *gorm.DB) error {
	if c.Status == "" {
		c.Status = "pending"
	}
	return nil
}

// IsApproved 检查评论是否已审核通过
func (c *Comment) IsApproved() bool {
	return c.Status == "approved"
}

// IsPending 检查评论是否待审核
func (c *Comment) IsPending() bool {
	return c.Status == "pending"
}

// IsSpam 检查评论是否为垃圾评论
func (c *Comment) IsSpam() bool {
	return c.Status == "spam"
}

// IsTrash 检查评论是否在回收站
func (c *Comment) IsTrash() bool {
	return c.Status == "trash"
}

// CanView 检查用户是否可以查看评论
func (c *Comment) CanView(user *User) bool {
	// 管理员可以查看所有评论
	if user != nil && user.IsAdmin() {
		return true
	}

	// 只有已审核通过的评论才能查看
	return c.IsApproved()
}

// CanEdit 检查用户是否可以编辑评论
func (c *Comment) CanEdit(user *User) bool {
	if user == nil {
		return false
	}

	// 管理员可以编辑所有评论
	if user.IsAdmin() {
		return true
	}

	// 评论作者可以编辑自己的评论
	if c.AuthorID != nil && *c.AuthorID == user.ID {
		return true
	}

	return false
}

// CanDelete 检查用户是否可以删除评论
func (c *Comment) CanDelete(user *User) bool {
	return c.CanEdit(user)
}

// IsTopLevel 检查是否为顶级评论
func (c *Comment) IsTopLevel() bool {
	return c.ParentID == nil
}

// GetAuthorName 获取作者名称
func (c *Comment) GetAuthorName() string {
	if c.Author != nil {
		return c.Author.GetDisplayName()
	}
	return c.AuthorName
}

// GetAuthorEmail 获取作者邮箱
func (c *Comment) GetAuthorEmail() string {
	if c.Author != nil {
		return c.Author.Email
	}
	return c.AuthorEmail
}

// CanModerate 检查用户是否可以审核评论
func (c *Comment) CanModerate(user *User) bool {
	return user != nil && user.IsAdmin()
}

// IsImage 检查是否为图片文件
func (m *MediaFile) IsImage() bool {
	imageTypes := []string{
		"image/jpeg", "image/jpg", "image/png", "image/gif",
		"image/webp", "image/bmp", "image/svg+xml",
	}
	for _, t := range imageTypes {
		if m.MimeType == t {
			return true
		}
	}
	return false
}

// IsVideo 检查是否为视频文件
func (m *MediaFile) IsVideo() bool {
	videoTypes := []string{
		"video/mp4", "video/avi", "video/mov", "video/wmv",
		"video/flv", "video/webm", "video/mkv",
	}
	for _, t := range videoTypes {
		if m.MimeType == t {
			return true
		}
	}
	return false
}

// GetFileExtension 获取文件扩展名
func (m *MediaFile) GetFileExtension() string {
	// 从原始文件名中提取扩展名
	if idx := len(m.OriginalName) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if m.OriginalName[i] == '.' {
				return m.OriginalName[i+1:]
			}
		}
	}
	return ""
}

// GetHumanSize 获取人类可读的文件大小
func (m *MediaFile) GetHumanSize() string {
	const unit = 1024
	if m.Size < unit {
		return "< 1 KB"
	}

	div, exp := int64(unit), 0
	for n := m.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	if exp < len(units) {
		return string(rune(m.Size/div)) + " " + units[exp]
	}
	return "Large"
}