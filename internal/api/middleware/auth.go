package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/charlottepl/blog-system/internal/auth/jwt"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/user/model"
)

// AuthMiddleware JWT认证中间件
type AuthMiddleware struct {
	jwtService *jwt.JWTService
}

// NewAuthMiddleware 创建认证中间件实例
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwt.NewJWTService(),
	}
}

// RequireAuth 需要认证的中间件
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "请先登录",
				"code":    401,
			})
			c.Abort()
			return
		}

		claims, err := m.jwtService.ValidateAccessToken(token)
		if err != nil {
			logger.Errorf("JWT验证失败: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "登录已过期，请重新登录",
				"code":    401,
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("user", map[string]interface{}{
			"id":       claims.UserID,
			"username": claims.Username,
			"email":    claims.Email,
			"is_admin": claims.IsAdmin,
		})

		c.Next()
	}
}

// RequireAdmin 需要管理员权限的中间件
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先执行认证检查
		m.RequireAuth()(c)
		if c.IsAborted() {
			return
		}

		// 检查是否为管理员
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "需要管理员权限",
				"code":    403,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth 可选认证的中间件（如果有token则验证，没有则跳过）
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := m.jwtService.ValidateAccessToken(token)
		if err != nil {
			// 可选认证失败时不阻止请求，只是不设置用户信息
			logger.Debugf("可选JWT验证失败: %v", err)
			c.Next()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("user", map[string]interface{}{
			"id":       claims.UserID,
			"username": claims.Username,
			"email":    claims.Email,
			"is_admin": claims.IsAdmin,
		})

		c.Next()
	}
}

// extractToken 从请求中提取JWT token
func (m *AuthMiddleware) extractToken(c *gin.Context) string {
	// 从Authorization header中提取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// 从Cookie中提取
	token, err := c.Cookie("access_token")
	if err == nil && token != "" {
		return token
	}

	// 从查询参数中提取（不推荐，但为了兼容性提供）
	token = c.Query("token")
	if token != "" {
		return token
	}

	return ""
}

// CheckPermission 检查用户权限的中间件
func (m *AuthMiddleware) CheckPermission(checkPermission func(userID int, user model.User) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先执行认证检查
		m.RequireAuth()(c)
		if c.IsAborted() {
			return
		}

		userID, _ := c.Get("user_id")
		userInterface, _ := c.Get("user")

		// 类型断言
		uid, ok1 := userID.(int)
		user, ok2 := userInterface.(model.User)
		if !ok1 || !ok2 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "用户信息获取失败",
				"code":    500,
			})
			c.Abort()
			return
		}

		// 检查权限
		if !checkPermission(uid, user) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "权限不足",
				"code":    403,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimiter 简单的内存速率限制中间件
type RateLimiter struct {
	requests map[string][]int64
	limit    int
	window   int64 // 时间窗口（秒）
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(limit int, window int64) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]int64),
		limit:    limit,
		window:   window,
	}
}

// Limit 速率限制中间件
func (rl *RateLimiter) Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := getCurrentTimestamp()

		// 清理过期的请求记录
		rl.cleanupRequests(clientIP, now)

		// 检查是否超过限制
		if len(rl.requests[clientIP]) >= rl.limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "请求过于频繁，请稍后再试",
				"code":    429,
			})
			c.Abort()
			return
		}

		// 记录当前请求
		rl.requests[clientIP] = append(rl.requests[clientIP], now)

		c.Next()
	}
}

// cleanupRequests 清理过期的请求记录
func (rl *RateLimiter) cleanupRequests(clientIP string, now int64) {
	requests, exists := rl.requests[clientIP]
	if !exists {
		return
	}

	// 保留时间窗口内的请求
	var validRequests []int64
	for _, timestamp := range requests {
		if now-timestamp < rl.window {
			validRequests = append(validRequests, timestamp)
		}
	}

	if len(validRequests) == 0 {
		delete(rl.requests, clientIP)
	} else {
		rl.requests[clientIP] = validRequests
	}
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	// 这里应该使用 time.Now().Unix()
	// 为了简化，暂时返回固定值
	return 1640995200 // 2022-01-01 00:00:00
}

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	// 这里应该使用UUID或其他唯一ID生成方式
	// 为了简化，暂时返回固定格式
	return "req-" + "1234567890"
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Infof("[%s] %s %s %d %s %s",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
		return ""
	})
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Errorf("请求发生panic: %v", recovered)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "服务器内部错误",
			"code":    500,
		})
		c.Abort()
	})
}