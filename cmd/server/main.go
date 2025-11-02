package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/charlottepl/blog-system/internal/api/router"
	"github.com/charlottepl/blog-system/internal/core/config"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/migrations"
)

var (
	configFile = flag.String("config", "config/config.yaml", "配置文件路径")
	version    = flag.Bool("version", false, "显示版本信息")
	help       = flag.Bool("help", false, "显示帮助信息")
)

// 版本信息（构建时注入）
const (
	AppName    = "blog-system"
	AppVersion = "1.0.0"
	BuildTime  = "2024-01-01T00:00:00Z"
	GitCommit  = "unknown"
)

func main() {
	flag.Parse()

	// 显示版本信息
	if *version {
		printVersion()
		return
	}

	// 显示帮助信息
	if *help {
		printHelp()
		return
	}

	// 初始化应用
	app, err := NewApplication(*configFile)
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 启动应用
	if err := app.Run(); err != nil {
		log.Fatalf("启动应用失败: %v", err)
	}
}

// Application 应用结构
type Application struct {
	config    *config.Config
	logger    *logger.Logger
	server    *http.Server
	router    *router.Router
	dbManager *database.Manager
}

// NewApplication 创建应用实例
func NewApplication(configFile string) (*Application, error) {
	// 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 初始化日志
	log := logger.NewLogger(cfg.Log)

	// 初始化数据库
	dbManager, err := database.NewManager(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	// 运行数据库迁移
	if err := migrations.RunMigrations(dbManager.GetDB()); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 创建路由器
	r := router.NewRouter()

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r.SetupRoutes(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	return &Application{
		config:    cfg,
		logger:    log,
		server:    server,
		router:    r,
		dbManager: dbManager,
	}, nil
}

// Run 运行应用
func (app *Application) Run() error {
	// 启动信息
	app.logger.Infof("🚀 启动 %s v%s", AppName, AppVersion)
	app.logger.Infof("📝 配置文件: %s", app.config.Path)
	app.logger.Infof("🌐 服务地址: http://localhost:%d", app.config.Server.Port)
	app.logger.Infof("🏃 运行模式: %s", app.config.Server.Mode)
	app.logger.Infof("💾 数据库: %s", app.config.Database.Type)
	app.logger.Infof("📊 日志级别: %s", app.config.Log.Level)

	// 启动服务器
	go func() {
		app.logger.Infof("🎯 HTTP服务器启动在端口 %d", app.config.Server.Port)
		if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.logger.Errorf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	app.waitForShutdown()

	return nil
}

// waitForShutdown 等待关闭信号
func (app *Application) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.logger.Info("🛑 正在关闭服务器...")

	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := app.server.Shutdown(ctx); err != nil {
		app.logger.Errorf("服务器关闭失败: %v", err)
	}

	// 关闭数据库连接
	if err := app.dbManager.Close(); err != nil {
		app.logger.Errorf("数据库关闭失败: %v", err)
	}

	app.logger.Info("✅ 服务器已关闭")
}

// Close 关闭应用
func (app *Application) Close() error {
	// 关闭数据库连接
	if err := app.dbManager.Close(); err != nil {
		return fmt.Errorf("关闭数据库失败: %w", err)
	}

	return nil
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf(`%s
版本: %s
构建时间: %s
Git提交: %s
`, AppName, AppVersion, BuildTime, GitCommit)
}

// printHelp 打印帮助信息
func printHelp() {
	fmt.Printf(`%s - 博客管理系统

用法:
  %s [选项]

选项:
  -config string
        配置文件路径 (默认: "config/config.yaml")
  -version
        显示版本信息
  -help
        显示帮助信息

示例:
  %s                    # 使用默认配置启动
  %s -config prod.yaml   # 使用指定配置文件启动
  %s -version           # 显示版本信息

环境变量:
  BLOG_CONFIG          配置文件路径
  BLOG_LOG_LEVEL       日志级别 (debug, info, warn, error)
  BLOG_SERVER_PORT     服务器端口
  BLOG_DB_TYPE         数据库类型 (sqlite, mysql, postgres)
  BLOG_DB_DSN          数据库连接字符串

更多信息请访问: https://github.com/charlottepl/blog-system
`, AppName, AppName, AppName, AppName, AppName)
}