package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Upload   UploadConfig   `mapstructure:"upload"`
	Email    EmailConfig    `mapstructure:"email"`
	GitHub   GitHubConfig   `mapstructure:"github"`
	Security SecurityConfig `mapstructure:"security"`
	Cache    CacheConfig    `mapstructure:"cache"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Port    int    `mapstructure:"port"`
	Mode    string `mapstructure:"mode"`
	Host    string `mapstructure:"host"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string        `mapstructure:"type"`
	DSN             string        `mapstructure:"dsn"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret            string        `mapstructure:"secret"`
	ExpireHours       int           `mapstructure:"expire_hours"`
	RefreshExpireDays int           `mapstructure:"refresh_expire_days"`
	Issuer            string        `mapstructure:"issuer"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// UploadConfig 文件上传配置
type UploadConfig struct {
	Path            string   `mapstructure:"path"`
	MaxSize         int64    `mapstructure:"max_size"`
	AllowedTypes    []string `mapstructure:"allowed_types"`
	AllowedMIMETypes []string `mapstructure:"allowed_mime_types"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromName     string `mapstructure:"from_name"`
	FromEmail    string `mapstructure:"from_email"`
}

// GitHubConfig GitHub同步配置
type GitHubConfig struct {
	SyncEnabled  bool          `mapstructure:"sync_enabled"`
	Token        string        `mapstructure:"token"`
	Repo         string        `mapstructure:"repo"`
	Branch       string        `mapstructure:"branch"`
	SyncInterval time.Duration `mapstructure:"sync_interval"`
	AutoSync     bool          `mapstructure:"auto_sync"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	CORSOrigins        []string `mapstructure:"cors_origins"`
	CORSMethods        []string `mapstructure:"cors_methods"`
	CORSHeaders        []string `mapstructure:"cors_headers"`
	RateLimitEnabled   bool     `mapstructure:"rate_limit_enabled"`
	RateLimitRequests  int      `mapstructure:"rate_limit_requests"`
	RateLimitWindow    string   `mapstructure:"rate_limit_window"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type         string `mapstructure:"type"`
	RedisAddr    string `mapstructure:"redis_addr"`
	RedisPassword string `mapstructure:"redis_password"`
	RedisDB      int    `mapstructure:"redis_db"`
}

var GlobalConfig *Config

// Load 加载配置文件
func Load(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置环境变量前缀
	viper.SetEnvPrefix("BLOG")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置到结构体
	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "Blog System")
	viper.SetDefault("app.version", "1.0.0")
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("app.mode", "debug")
	viper.SetDefault("app.host", "0.0.0.0")

	// Database defaults
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.dsn", "./data/blog.db")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.conn_max_lifetime", "1h")

	// JWT defaults
	viper.SetDefault("jwt.expire_hours", 24)
	viper.SetDefault("jwt.refresh_expire_days", 7)
	viper.SetDefault("jwt.issuer", "blog-system")

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)

	// Upload defaults
	viper.SetDefault("upload.path", "./uploads")
	viper.SetDefault("upload.max_size", 10485760) // 10MB

	// Security defaults
	viper.SetDefault("security.cors_origins", []string{"*"})
	viper.SetDefault("security.cors_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("security.cors_headers", []string{"*"})
	viper.SetDefault("security.rate_limit_enabled", true)
	viper.SetDefault("security.rate_limit_requests", 100)
	viper.SetDefault("security.rate_limit_window", "1m")

	// Cache defaults
	viper.SetDefault("cache.type", "memory")
}

// Get 获取全局配置
func Get() *Config {
	return GlobalConfig
}

// IsDebugMode 是否为调试模式
func IsDebugMode() bool {
	return GlobalConfig != nil && GlobalConfig.App.Mode == "debug"
}

// IsProductionMode 是否为生产模式
func IsProductionMode() bool {
	return GlobalConfig != nil && GlobalConfig.App.Mode == "release"
}