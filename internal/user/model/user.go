package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           int       `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email        string    `json:"email" gorm:"uniqueIndex;size:100"`
	Phone        string    `json:"phone" gorm:"uniqueIndex;size:20"`
	Nickname     string    `json:"nickname" gorm:"size:50"`
	Avatar       string    `json:"avatar" gorm:"size:255"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	Role         string    `json:"role" gorm:"default:user;size:20;not null"`
	Status       string    `json:"status" gorm:"default:active;size:20;not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	AuthProviders []UserAuthProvider `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Posts         []Post             `json:"-" gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`
	Comments      []Comment          `json:"-" gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`
	Likes         []Like             `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Favorites     []Favorite         `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	MediaFiles    []MediaFile        `json:"-" gorm:"foreignKey:UploaderID;constraint:OnDelete:SET NULL"`
}

// UserAuthProvider 用户认证提供者
type UserAuthProvider struct {
	ID           int                    `json:"id" gorm:"primaryKey"`
	UserID       int                    `json:"user_id" gorm:"not null;index"`
	Provider     string                 `json:"provider" gorm:"size:50;not null"`
	ProviderID   string                 `json:"provider_id" gorm:"size:100"`
	ProviderData string                 `json:"provider_data" gorm:"type:text"`
	IsPrimary    bool                   `json:"is_primary" gorm:"default:false"`
	CreatedAt    time.Time              `json:"created_at"`

	// 关联关系
	User         User                   `json:"user" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// TableName 指定表名
func (UserAuthProvider) TableName() string {
	return "user_auth_providers"
}

// BeforeCreate GORM钩子：创建前
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = "user"
	}
	if u.Status == "" {
		u.Status = "active"
	}
	return nil
}

// IsAdmin 检查是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

// IsActive 检查用户是否激活
func (u *User) IsActive() bool {
	return u.Status == "active"
}

// CanCreatePost 检查是否可以创建文章
func (u *User) CanCreatePost() bool {
	return u.IsAdmin() // 只有管理员可以创建文章
}

// CanComment 检查是否可以评论
func (u *User) CanComment() bool {
	return u.IsActive()
}

// GetDisplayName 获取显示名称
func (u *User) GetDisplayName() string {
	if u.Nickname != "" {
		return u.Nickname
	}
	return u.Username
}

// ToSafeJSON 转换为安全的JSON格式（不包含敏感信息）
func (u *User) ToSafeJSON() map[string]interface{} {
	return map[string]interface{}{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
		"nickname": u.Nickname,
		"avatar":   u.Avatar,
		"role":     u.Role,
		"status":   u.Status,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}