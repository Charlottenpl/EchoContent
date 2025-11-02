package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/charlottepl/blog-system/internal/core/config"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var GlobalLogger *logrus.Logger

// Init 初始化日志系统
func Init() {
	GlobalLogger = logrus.New()

	// 设置日志格式
	setFormatter()

	// 设置输出
	setOutput()

	// 设置日志级别
	setLevel()
}

// setFormatter 设置日志格式
func setFormatter() {
	if cfg := config.Get(); cfg != nil && cfg.Log.Format == "json" {
		GlobalLogger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		GlobalLogger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}
}

// setOutput 设置日志输出
func setOutput() {
	cfg := config.Get()
	if cfg == nil {
		GlobalLogger.SetOutput(os.Stdout)
		return
	}

	switch cfg.Log.Output {
	case "file":
		if cfg.Log.FilePath != "" {
			// 确保目录存在
			dir := filepath.Dir(cfg.Log.FilePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				GlobalLogger.Warnf("Failed to create log directory: %v", err)
				GlobalLogger.SetOutput(os.Stdout)
				return
			}

			// 使用lumberjack进行日志轮转
			fileWriter := &lumberjack.Logger{
				Filename:   cfg.Log.FilePath,
				MaxSize:    cfg.Log.MaxSize,
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge,
				Compress:   true,
			}
			GlobalLogger.SetOutput(fileWriter)
		} else {
			GlobalLogger.SetOutput(os.Stdout)
		}
	case "both":
		if cfg.Log.FilePath != "" {
			// 确保目录存在
			dir := filepath.Dir(cfg.Log.FilePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				GlobalLogger.Warnf("Failed to create log directory: %v", err)
				GlobalLogger.SetOutput(os.Stdout)
				return
			}

			fileWriter := &lumberjack.Logger{
				Filename:   cfg.Log.FilePath,
				MaxSize:    cfg.Log.MaxSize,
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge,
				Compress:   true,
			}
			GlobalLogger.SetOutput(io.MultiWriter(os.Stdout, fileWriter))
		} else {
			GlobalLogger.SetOutput(os.Stdout)
		}
	default:
		GlobalLogger.SetOutput(os.Stdout)
	}
}

// setLevel 设置日志级别
func setLevel() {
	cfg := config.Get()
	if cfg == nil {
		GlobalLogger.SetLevel(logrus.InfoLevel)
		return
	}

	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		GlobalLogger.Warnf("Invalid log level: %s, using info level", cfg.Log.Level)
		GlobalLogger.SetLevel(logrus.InfoLevel)
	} else {
		GlobalLogger.SetLevel(level)
	}
}

// GetLogger 获取全局日志实例
func GetLogger() *logrus.Logger {
	if GlobalLogger == nil {
		Init()
	}
	return GlobalLogger
}

// WithFields 创建带字段的日志实例
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// WithField 创建带单个字段的日志实例
func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

// Debug 记录调试日志
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Debugf 记录格式化调试日志
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info 记录信息日志
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Infof 记录格式化信息日志
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn 记录警告日志
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Warnf 记录格式化警告日志
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error 记录错误日志
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Errorf 记录格式化错误日志
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal 记录致命错误日志并退出程序
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// Fatalf 记录格式化致命错误日志并退出程序
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// Panic 记录恐慌日志并触发panic
func Panic(args ...interface{}) {
	GetLogger().Panic(args...)
}

// Panicf 记录格式化恐慌日志并触发panic
func Panicf(format string, args ...interface{}) {
	GetLogger().Panicf(format, args...)
}