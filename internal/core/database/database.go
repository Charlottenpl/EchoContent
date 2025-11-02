package database

import (
	"fmt"
	"time"

	"github.com/charlottepl/blog-system/internal/core/config"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/loggergorm"
)

var GlobalDB *gorm.DB

// Init 初始化数据库连接
func Init() error {
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// 配置GORM日志
	gormLogger := loggergorm.Default
	if cfg.Log.Level == "debug" {
		gormLogger = loggergorm.Default.LogMode(loggergorm.Info)
	} else {
		gormLogger = loggergorm.Default.LogMode(loggergorm.Error)
	}

	// 打开数据库连接
	var err error
	switch cfg.Database.Type {
	case "sqlite":
		GlobalDB, err = openSQLite(cfg.Database.DSN, gormLogger)
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	if err := configureConnectionPool(GlobalDB, cfg.Database); err != nil {
		return fmt.Errorf("failed to configure connection pool: %w", err)
	}

	logger.Info("Database connected successfully")
	return nil
}

// openSQLite 打开SQLite数据库连接
func openSQLite(dsn string, gormLogger loggergorm.Interface) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
}

// configureConnectionPool 配置数据库连接池
func configureConnectionPool(db *gorm.DB, cfg config.DatabaseConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置最大空闲连接数
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	// 设置最大打开连接数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// 设置连接最大生命周期
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return nil
}

// GetDB 获取全局数据库实例
func GetDB() *gorm.DB {
	return GlobalDB
}

// Close 关闭数据库连接
func Close() error {
	if GlobalDB != nil {
		sqlDB, err := GlobalDB.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(models ...interface{}) error {
	if GlobalDB == nil {
		return fmt.Errorf("database not initialized")
	}

	return GlobalDB.AutoMigrate(models...)
}

// Health 检查数据库连接健康状态
func Health() error {
	if GlobalDB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := GlobalDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// GetStats 获取数据库连接统计信息
func GetStats() map[string]interface{} {
	if GlobalDB == nil {
		return map[string]interface{}{
			"error": "database not initialized",
		}
	}

	sqlDB, err := GlobalDB.DB()
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to get underlying sql.DB: %v", err),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections":   stats.MaxOpenConnections,
		"open_connections":       stats.OpenConnections,
		"in_use":                stats.InUse,
		"idle":                  stats.Idle,
		"wait_count":            stats.WaitCount,
		"wait_duration":         stats.WaitDuration.String(),
		"max_idle_closed":       stats.MaxIdleClosed,
		"max_idle_time_closed":  stats.MaxIdleTimeClosed,
		"max_lifetime_closed":   stats.MaxLifetimeClosed,
	}
}