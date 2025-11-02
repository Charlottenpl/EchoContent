package model

import (
	"fmt"
	"time"
)

// MediaFile 媒体文件模型
type MediaFile struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Filename    string    `json:"filename" gorm:"not null;size:255"`       // 原始文件名
	StoragePath string    `json:"storage_path" gorm:"not null;size:500"`   // 存储路径
	FileSize    int64     `json:"file_size" gorm:"not null"`               // 文件大小（字节）
	MimeType    string    `json:"mime_type" gorm:"not null;size:100"`      // MIME类型
	FileType    string    `json:"file_type" gorm:"not null;size:50"`       // 文件类型：image, video, document, other
	Width       *int      `json:"width"`                                  // 图片宽度
	Height      *int      `json:"height"`                                 // 图片高度
	Duration    *int      `json:"duration"`                               // 视频/音频时长（秒）
	Hash        string    `json:"hash" gorm:"not null;size:64;uniqueIndex"` // 文件哈希值，用于去重
	Alt         string    `json:"alt" gorm:"size:200"`                    // 图片替代文本
	Caption     string    `json:"caption" gorm:"size:500"`                // 图片说明
	UploaderID  *int      `json:"uploader_id"`                            // 上传者ID
	Status      string    `json:"status" gorm:"default:'active';size:20"` // 状态：active, deleted
	IsPublic    bool      `json:"is_public" gorm:"default:true"`          // 是否公开
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 关联关系
	Uploader *User `json:"uploader,omitempty" gorm:"foreignKey:UploaderID"`
}

// MediaUsage 媒体文件使用记录
type MediaUsage struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	MediaID   int       `json:"media_id" gorm:"not null;index"`    // 媒体文件ID
	UsageType string    `json:"usage_type" gorm:"not null;size:50"` // 使用类型：post_avatar, post_cover, post_content, moment_content, user_avatar
	TargetID  *int      `json:"target_id"`                         // 目标ID（文章ID、用户ID等）
	UploaderID int      `json:"uploader_id" gorm:"not null"`      // 使用者ID
	CreatedAt time.Time `json:"created_at"`

	// 关联关系
	Media   MediaFile `json:"media" gorm:"foreignKey:MediaID"`
	Uploader User      `json:"uploader" gorm:"foreignKey:UploaderID"`
}

// TableName 指定表名
func (MediaFile) TableName() string {
	return "media_files"
}

// TableName 指定表名
func (MediaUsage) TableName() string {
	return "media_usages"
}

// IsImage 检查是否为图片
func (m *MediaFile) IsImage() bool {
	return m.FileType == "image"
}

// IsVideo 检查是否为视频
func (m *MediaFile) IsVideo() bool {
	return m.FileType == "video"
}

// IsDocument 检查是否为文档
func (m *MediaFile) IsDocument() bool {
	return m.FileType == "document"
}

// GetFileExtension 获取文件扩展名
func (m *MediaFile) GetFileExtension() string {
	for i := len(m.Filename) - 1; i >= 0; i-- {
		if m.Filename[i] == '.' {
			return m.Filename[i+1:]
		}
	}
	return ""
}

// GetFormattedSize 获取格式化的文件大小
func (m *MediaFile) GetFormattedSize() string {
	const unit = 1024
	if m.FileSize < unit {
		return fmt.Sprintf("%d B", m.FileSize)
	}

	div, exp := int64(unit), 0
	for n := m.FileSize / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.1f %s", float64(m.FileSize)/float64(div), units[exp])
}

// GetAspectRatio 获取图片宽高比
func (m *MediaFile) GetAspectRatio() *float64 {
	if m.Width == nil || m.Height == nil || *m.Height == 0 {
		return nil
	}
	ratio := float64(*m.Width) / float64(*m.Height)
	return &ratio
}

// CanEdit 检查用户是否可以编辑媒体文件
func (m *MediaFile) CanEdit(user *User) bool {
	if user == nil {
		return false
	}

	// 管理员可以编辑所有媒体文件
	if user.IsAdmin() {
		return true
	}

	// 上传者可以编辑自己的媒体文件
	if m.UploaderID != nil && *m.UploaderID == user.ID {
		return true
	}

	return false
}

// CanDelete 检查用户是否可以删除媒体文件
func (m *MediaFile) CanDelete(user *User) bool {
	return m.CanEdit(user)
}

// ToSafeJSON 转换为安全的JSON格式（不包含敏感信息）
func (m *MediaFile) ToSafeJSON() string {
	return fmt.Sprintf("media_%d_%s", m.ID, m.Hash[:8])
}

// NewMediaFile 创建媒体文件
func NewMediaFile(filename, storagePath, mimeType, fileType, hash string, fileSize int64, uploaderID *int) *MediaFile {
	return &MediaFile{
		Filename:    filename,
		StoragePath: storagePath,
		MimeType:    mimeType,
		FileType:    fileType,
		Hash:        hash,
		FileSize:    fileSize,
		UploaderID:  uploaderID,
		Status:      "active",
		IsPublic:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// NewMediaUsage 创建媒体使用记录
func NewMediaUsage(mediaID int, usageType string, targetID, uploaderID *int) *MediaUsage {
	return &MediaUsage{
		MediaID:    mediaID,
		UsageType:  usageType,
		TargetID:   targetID,
		UploaderID: *uploaderID,
		CreatedAt:  time.Now(),
	}
}

// SetDimensions 设置图片尺寸
func (m *MediaFile) SetDimensions(width, height int) {
	m.Width = &width
	m.Height = &height
}

// SetDuration 设置视频/音频时长
func (m *MediaFile) SetDuration(duration int) {
	m.Duration = &duration
}

// SoftDelete 软删除
func (m *MediaFile) SoftDelete() {
	m.Status = "deleted"
	m.UpdatedAt = time.Now()
}

// Restore 恢复删除
func (m *MediaFile) Restore() {
	m.Status = "active"
	m.UpdatedAt = time.Now()
}

// IsActive 检查是否处于活跃状态
func (m *MediaFile) IsActive() bool {
	return m.Status == "active"
}

// SetPublic 设置公开状态
func (m *MediaFile) SetPublic(isPublic bool) {
	m.IsPublic = isPublic
	m.UpdatedAt = time.Now()
}