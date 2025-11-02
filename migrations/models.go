package migrations

import (
	"time"

	"github.com/charlottepl/blog-system/internal/blog/model"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/user/model"
	"gorm.io/gorm"
)

// AllModels 包含所有需要迁移的模型
var AllModels = []interface{}{
	// 用户相关模型
	&model.User{},
	&model.UserAuthProvider{},

	// 内容相关模型
	&model.Category{},
	&model.Tag{},
	&model.Post{},
	&model.PostTag{},

	// 互动相关模型
	&model.Comment{},
	&model.Like{},
	&model.Favorite{},

	// 系统相关模型
	&model.MediaFile{},
	&model.SystemConfig{},
	&model.SystemLog{},
}

// AutoMigrate 自动迁移所有模型
func AutoMigrate(db *gorm.DB) error {
	logger.Info("开始数据库迁移...")

	// 记录开始时间
	startTime := time.Now()

	// 执行迁移
	if err := db.AutoMigrate(AllModels...); err != nil {
		logger.Errorf("数据库迁移失败: %v", err)
		return err
	}

	// 记录耗时
	duration := time.Since(startTime)
	logger.Infof("数据库迁移完成，耗时: %v", duration)

	// 创建索引
	if err := createIndexes(db); err != nil {
		logger.Errorf("创建索引失败: %v", err)
		return err
	}

	// 初始化基础数据
	if err := seedInitialData(db); err != nil {
		logger.Errorf("初始化基础数据失败: %v", err)
		return err
	}

	logger.Info("数据库迁移成功完成")
	return nil
}

// createIndexes 创建额外的索引
func createIndexes(db *gorm.DB) error {
	logger.Info("创建数据库索引...")

	indexes := []string{
		// 文章相关索引
		"CREATE INDEX IF NOT EXISTS idx_posts_type_status ON posts(type, status)",
		"CREATE INDEX IF NOT EXISTS idx_posts_author_status ON posts(author_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_posts_published_at ON posts(published_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_posts_category_status ON posts(category_id, status) WHERE category_id IS NOT NULL",

		// 评论相关索引
		"CREATE INDEX IF NOT EXISTS idx_comments_post_status ON comments(post_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id) WHERE parent_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at DESC)",

		// 点赞相关索引
		"CREATE INDEX IF NOT EXISTS idx_likes_target ON likes(target_type, target_id)",
		"CREATE INDEX IF NOT EXISTS idx_likes_user_target ON likes(user_id, target_type, target_id)",

		// 用户相关索引
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)",
		"CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)",

		// 认证提供者索引
		"CREATE INDEX IF NOT EXISTS idx_user_auth_providers_provider ON user_auth_providers(provider, provider_id) WHERE provider_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_user_auth_providers_primary ON user_auth_providers(user_id, is_primary)",

		// 媒体文件索引
		"CREATE INDEX IF NOT EXISTS idx_media_files_uploader ON media_files(uploader_id) WHERE uploader_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_media_files_mime_type ON media_files(mime_type)",
		"CREATE INDEX IF NOT EXISTS idx_media_files_created_at ON media_files(created_at DESC)",

		// 系统日志索引
		"CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level)",
		"CREATE INDEX IF NOT EXISTS idx_system_logs_created_at ON system_logs(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_system_logs_user ON system_logs(user_id) WHERE user_id IS NOT NULL",
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			logger.Warnf("创建索引失败: %s, 错误: %v", indexSQL, err)
			// 索引创建失败不中断迁移，只记录警告
		}
	}

	logger.Info("数据库索引创建完成")
	return nil
}

// seedInitialData 初始化基础数据
func seedInitialData(db *gorm.DB) error {
	logger.Info("初始化基础数据...")

	// 初始化系统配置
	if err := seedSystemConfigs(db); err != nil {
		return err
	}

	// 初始化默认分类
	if err := seedDefaultCategories(db); err != nil {
		return err
	}

	// 初始化默认标签
	if err := seedDefaultTags(db); err != nil {
		return err
	}

	logger.Info("基础数据初始化完成")
	return nil
}

// seedSystemConfigs 初始化系统配置
func seedSystemConfigs(db *gorm.DB) error {
	configs := []model.SystemConfig{
		{
			Key:         "site_name",
			Value:       "博客系统",
			Type:        "string",
			Description: "网站名称",
		},
		{
			Key:         "site_description",
			Value:       "基于Go语言开发的轻量级博客系统",
			Type:        "string",
			Description: "网站描述",
		},
		{
			Key:         "site_keywords",
			Value:       "博客,Go,技术分享",
			Type:        "string",
			Description: "网站关键词",
		},
		{
			Key:         "posts_per_page",
			Value:       "10",
			Type:        "number",
			Description: "每页文章数量",
		},
		{
			Key:         "comment_moderation",
			Value:       "true",
			Type:        "boolean",
			Description: "是否需要评论审核",
		},
		{
			Key:         "allow_registration",
			Value:       "true",
			Type:        "boolean",
			Description: "是否允许用户注册",
		},
		{
			Key:         "email_verification",
			Value:       "true",
			Type:        "boolean",
			Description: "是否需要邮箱验证",
		},
	}

	for _, config := range configs {
		var existing model.SystemConfig
		result := db.Where("key = ?", config.Key).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&config).Error; err != nil {
				logger.Warnf("创建系统配置失败: %s, 错误: %v", config.Key, err)
			}
		}
	}

	return nil
}

// seedDefaultCategories 初始化默认分类
func seedDefaultCategories(db *gorm.DB) error {
	categories := []model.Category{
		{
			Name:        "技术",
			Slug:        "tech",
			Description: "技术相关的文章",
		},
		{
			Name:        "生活",
			Slug:        "life",
			Description: "生活相关的随记",
		},
		{
			Name:        "随笔",
			Slug:        "essay",
			Description: "随笔感想",
		},
		{
			Name:        "未分类",
			Slug:        "uncategorized",
			Description: "未分类的文章",
		},
	}

	for _, category := range categories {
		var existing model.Category
		result := db.Where("slug = ?", category.Slug).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&category).Error; err != nil {
				logger.Warnf("创建默认分类失败: %s, 错误: %v", category.Name, err)
			}
		}
	}

	return nil
}

// seedDefaultTags 初始化默认标签
func seedDefaultTags(db *gorm.DB) error {
	tags := []model.Tag{
		{Name: "Go", Slug: "go"},
		{Name: "编程", Slug: "programming"},
		{Name: "数据库", Slug: "database"},
		{Name: "前端", Slug: "frontend"},
		{Name: "后端", Slug: "backend"},
		{Name: "架构", Slug: "architecture"},
		{Name: "运维", Slug: "devops"},
		{Name: "算法", Slug: "algorithm"},
		{Name: "设计", Slug: "design"},
		{Name: "随笔", Slug: "notes"},
	}

	for _, tag := range tags {
		var existing model.Tag
		result := db.Where("slug = ?", tag.Slug).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&tag).Error; err != nil {
				logger.Warnf("创建默认标签失败: %s, 错误: %v", tag.Name, err)
			}
		}
	}

	return nil
}

// Rollback 回滚迁移（SQLite有限支持）
func Rollback(db *gorm.DB) error {
	logger.Warn("SQLite不支持完整的回滚操作，请手动清理数据")

	// SQLite不支持DROP COLUMN，所以只能删除表重新创建
	// 这里只是记录日志，不执行实际的删除操作
	tables := []string{
		"system_logs", "media_files", "favorites", "likes", "comments",
		"post_tags", "posts", "tags", "categories", "user_auth_providers", "users",
	}

	for _, table := range tables {
		logger.Warnf("如需完全回滚，请手动删除表: %s", table)
	}

	return nil
}

// GetMigrationVersion 获取迁移版本
func GetMigrationVersion(db *gorm.DB) (string, error) {
	// SQLite没有内置的版本控制表，这里使用系统配置来存储版本
	var config model.SystemConfig
	err := db.Where("key = ?", "migration_version").First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "0.0.0", nil
		}
		return "", err
	}
	return config.Value, nil
}

// SetMigrationVersion 设置迁移版本
func SetMigrationVersion(db *gorm.DB, version string) error {
	config := model.SystemConfig{
		Key:         "migration_version",
		Value:       version,
		Type:        "string",
		Description: "数据库迁移版本",
	}

	var existing model.SystemConfig
	err := db.Where("key = ?", "migration_version").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return db.Create(&config).Error
	}

	return db.Model(&existing).Update("value", version).Error
}