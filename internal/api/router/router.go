package router

import (
	"github.com/gin-gonic/gin"
	"github.com/charlottepl/blog-system/internal/api/handler"
	"github.com/charlottepl/blog-system/internal/api/middleware"
)

// Router 路由器
type Router struct {
	engine     *gin.Engine
	authHandler *handler.AuthHandler
	blogHandler *handler.BlogHandler
	middleware  *middleware.AuthMiddleware
}

// NewRouter 创建路由器实例
func NewRouter() *Router {
	return &Router{
		engine:      gin.New(),
		authHandler: handler.NewAuthHandler(),
		blogHandler: handler.NewBlogHandler(),
		middleware:  middleware.NewAuthMiddleware(),
	}
}

// SetupRoutes 设置路由
func (r *Router) SetupRoutes() *gin.Engine {
	// 全局中间件
	r.engine.Use(middleware.LoggerMiddleware())
	r.engine.Use(middleware.RecoveryMiddleware())
	r.engine.Use(middleware.CORSMiddleware())
	r.engine.Use(middleware.RequestIDMiddleware())

	// 速率限制
	limiter := middleware.NewRateLimiter(100, 60) // 每分钟100次请求
	r.engine.Use(limiter.Limit())

	// API版本组
	v1 := r.engine.Group("/api/v1")
	{
		// 认证路由
		r.setupAuthRoutes(v1)

		// 博客路由
		r.setupBlogRoutes(v1)

		// 随念路由
		r.setupMomentRoutes(v1)

		// 评论路由
		r.setupCommentRoutes(v1)

		// 互动路由
		r.setupInteractionRoutes(v1)

		// 媒体路由
		r.setupMediaRoutes(v1)

		// 用户路由
		r.setupUserRoutes(v1)
	}

	// 健康检查路由
	r.engine.GET("/health", r.healthCheck)
	r.engine.GET("/ping", r.ping)

	return r.engine
}

// setupAuthRoutes 设置认证路由
func (r *Router) setupAuthRoutes(group *gin.RouterGroup) {
	auth := group.Group("/auth")
	{
		// 公开路由
		auth.POST("/register", r.authHandler.Register)
		auth.POST("/login", r.authHandler.Login)
		auth.POST("/login/email", r.authHandler.EmailLogin)
		auth.POST("/refresh", r.authHandler.RefreshToken)
		auth.GET("/verify/email", r.authHandler.VerifyEmail)
		auth.POST("/reset/password", r.authHandler.ResetPassword)

		// 需要认证的路由
		auth.Use(r.middleware.RequireAuth())
		{
			auth.POST("/logout", r.authHandler.Logout)
			auth.GET("/profile", r.authHandler.GetProfile)
			auth.PUT("/profile", r.authHandler.UpdateProfile)
			auth.PUT("/password", r.authHandler.ChangePassword)
		}
	}
}

// setupBlogRoutes 设置博客路由
func (r *Router) setupBlogRoutes(group *gin.RouterGroup) {
	blog := group.Group("/blog")
	{
		// 公开路由
		blog.GET("/posts", r.blogHandler.GetPosts)
		blog.GET("/posts/search", r.blogHandler.SearchPosts)
		blog.GET("/posts/:id", r.blogHandler.GetPost)
		blog.GET("/categories", r.blogHandler.GetCategories)
		blog.GET("/tags", r.blogHandler.GetTags)
		blog.GET("/tags/popular", r.blogHandler.GetPopularTags)
		blog.GET("/stats", r.blogHandler.GetPostStats)

		// 需要认证的路由
		blog.Use(r.middleware.RequireAuth())
		{
			// 文章管理
			blog.POST("/posts", r.blogHandler.CreatePost)
			blog.PUT("/posts/:id", r.blogHandler.UpdatePost)
			blog.DELETE("/posts/:id", r.blogHandler.DeletePost)
			blog.POST("/posts/:id/publish", r.blogHandler.PublishPost)

			// 分类管理（需要管理员权限）
			admin := blog.Group("/admin")
			admin.Use(r.middleware.RequireAdmin())
			{
				admin.POST("/categories", r.blogHandler.CreateCategory)
				admin.PUT("/categories/:id", r.updateCategory)
				admin.DELETE("/categories/:id", r.deleteCategory)
				admin.POST("/tags", r.blogHandler.CreateTag)
				admin.PUT("/tags/:id", r.updateTag)
				admin.DELETE("/tags/:id", r.deleteTag)
			}
		}
	}
}

// setupMomentRoutes 设置随念路由
func (r *Router) setupMomentRoutes(group *gin.RouterGroup) {
	moment := group.Group("/moments")
	{
		// 公开路由
		moment.GET("", r.getMomentList)
		moment.GET("/trending", r.getTrendingMoments)
		moment.GET("/search", r.searchMoments)
		moment.GET("/:id", r.getMomentByID)

		// 需要认证的路由
		moment.Use(r.middleware.RequireAuth())
		{
			moment.POST("", r.createMoment)
			moment.PUT("/:id", r.updateMoment)
			moment.DELETE("/:id", r.deleteMoment)
			moment.POST("/:id/publish", r.publishMoment)
			moment.POST("/:id/unpublish", r.unpublishMoment)

			// 用户的随念
			moment.GET("/my", r.getMyMoments)
			moment.GET("/my/drafts", r.getMyDrafts)
		}
	}
}

// setupCommentRoutes 设置评论路由
func (r *Router) setupCommentRoutes(group *gin.RouterGroup) {
	comment := group.Group("/comments")
	{
		// 公开路由
		comment.GET("/post/:postId", r.getPostComments)
		comment.GET("/:id", r.getCommentByID)
		comment.GET("/:id/replies", r.getCommentReplies)

		// 需要认证的路由
		comment.Use(r.middleware.RequireAuth())
		{
			comment.POST("", r.createComment)
			comment.PUT("/:id", r.updateComment)
			comment.DELETE("/:id", r.deleteComment)
			comment.POST("/:id/like", r.likeComment)

			// 管理员路由
			admin := comment.Group("/admin")
			admin.Use(r.middleware.RequireAdmin())
			{
				admin.GET("/pending", r.getPendingComments)
				admin.POST("/:id/approve", r.approveComment)
				admin.POST("/:id/reject", r.rejectComment)
			}
		}
	}
}

// setupInteractionRoutes 设置互动路由
func (r *Router) setupInteractionRoutes(group *gin.RouterGroup) {
	interaction := group.Group("/interactions")
	{
		// 需要认证的路由
		interaction.Use(r.middleware.RequireAuth())
		{
			// 点赞
			interaction.POST("/like", r.likeContent)
			interaction.DELETE("/like", r.unlikeContent)
			interaction.GET("/like/status", r.getLikeStatus)

			// 收藏
			interaction.POST("/favorite", r.favoriteContent)
			interaction.DELETE("/favorite", r.unfavoriteContent)
			interaction.GET("/favorite/status", r.getFavoriteStatus)

			// 用户互动记录
			interaction.GET("/likes", r.getUserLikes)
			interaction.GET("/favorites", r.getUserFavorites)
			interaction.GET("/stats", r.getUserInteractionStats)
		}
	}
}

// setupMediaRoutes 设置媒体路由
func (r *Router) setupMediaRoutes(group *gin.RouterGroup) {
	media := group.Group("/media")
	{
		// 公开路由
		media.GET("/public", r.getPublicMedia)
		media.GET("/:id", r.getMediaByID)

		// 需要认证的路由
		media.Use(r.middleware.RequireAuth())
		{
			// 文件上传
			media.POST("/upload", r.uploadFile)
			media.POST("/upload/batch", r.uploadMultipleFiles)

			// 媒体管理
			media.GET("", r.getMediaList)
			media.PUT("/:id", r.updateMedia)
			media.DELETE("/:id", r.deleteMedia)
			media.GET("/search", r.searchMedia)
			media.GET("/stats", r.getMediaStats)

			// 图片处理
			media.POST("/:id/process", r.processImage)
			media.POST("/process/batch", r.batchProcessImages)
			media.GET("/:id/info", r.getImageInfo)
			media.POST("/:id/optimize", r.optimizeImage)
		}
	}
}

// setupUserRoutes 设置用户路由
func (r *Router) setupUserRoutes(group *gin.RouterGroup) {
	user := group.Group("/users")
	{
		// 公开路由
		user.GET("/profile/:username", r.getUserProfile)
		user.GET("/:id/posts", r.getUserPosts)
		user.GET("/:id/moments", r.getUserMoments)

		// 需要认证的路由
		user.Use(r.middleware.RequireAuth())
		{
			user.GET("/me", r.getCurrentUser)
			user.PUT("/me", r.updateCurrentUser)
			user.GET("/me/activities", r.getUserActivities)
			user.GET("/me/stats", r.getCurrentUserStats)
		}
	}
}

// 健康检查路由处理函数
func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"service": "blog-system",
	})
}

func (r *Router) ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

// 占位符处理函数（在后续实现中补充）
func (r *Router) updateCategory(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) deleteCategory(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) updateTag(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) deleteTag(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getMomentList(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getTrendingMoments(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) searchMoments(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getMomentByID(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) createMoment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) updateMoment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) deleteMoment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) publishMoment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) unpublishMoment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getMyMoments(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getMyDrafts(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getPostComments(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getCommentByID(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getCommentReplies(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) createComment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) updateComment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) deleteComment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) likeComment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getPendingComments(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) approveComment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) rejectComment(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) likeContent(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) unlikeContent(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getLikeStatus(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) favoriteContent(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) unfavoriteContent(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getFavoriteStatus(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getUserLikes(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getUserFavorites(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getUserInteractionStats(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getPublicMedia(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getMediaByID(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) uploadFile(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) uploadMultipleFiles(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getMediaList(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) updateMedia(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) deleteMedia(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) searchMedia(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getMediaStats(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) processImage(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) batchProcessImages(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getImageInfo(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) optimizeImage(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getUserProfile(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getUserPosts(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getUserMoments(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getCurrentUser(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) updateCurrentUser(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getUserActivities(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}

func (r *Router) getCurrentUserStats(c *gin.Context) {
	c.JSON(501, gin.H{"message": "功能未实现"})
}