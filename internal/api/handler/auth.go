package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/charlottepl/blog-system/internal/auth/service"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/core/validator"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	*BaseHandler
	authService *service.AuthService
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		BaseHandler: NewBaseHandler(),
		authService: service.NewAuthService(),
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=20,alpha_num"`
	Email    string `json:"email" validate:"required,email,max=100"`
	Password string `json:"password" validate:"required,min=6,max=50"`
	Nickname string `json:"nickname" validate:"max=50"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember"`
}

// EmailLoginRequest 邮箱登录请求
type EmailLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Remember bool   `json:"remember"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=50"`
}

// UpdateProfileRequest 更新用户资料请求
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" validate:"max=50"`
	Bio      string `json:"bio" validate:"max=500"`
	Avatar   string `json:"avatar" validate:"url,max=200"`
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层注册
	user, err := h.authService.Register(&service.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
		IP:       h.GetClientIP(c),
		UserAgent: h.GetUserAgent(c),
	})
	if err != nil {
		if strings.Contains(err.Error(), "已存在") {
			h.Error(c, http.StatusConflict, err.Error())
		} else {
			h.InternalError(c, "注册失败: "+err.Error())
		}
		return
	}

	logger.Infof("用户注册成功: %s (%s)", user.Username, user.Email)

	// 返回用户信息（不包含敏感信息）
	h.SuccessWithMessage(c, "注册成功", user.ToSafeResponse())
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层登录
	result, err := h.authService.Login(&service.LoginRequest{
		Username:  req.Username,
		Password:  req.Password,
		IP:        h.GetClientIP(c),
		UserAgent: h.GetUserAgent(c),
		Remember:  req.Remember,
	})
	if err != nil {
		if strings.Contains(err.Error(), "不存在") || strings.Contains(err.Error(), "密码错误") {
			h.UnauthorizedError(c, "用户名或密码错误")
		} else {
			h.InternalError(c, "登录失败: "+err.Error())
		}
		return
	}

	logger.Infof("用户登录成功: %s", result.User.Username)

	// 设置cookie
	maxAge := 3600 // 1小时
	if req.Remember {
		maxAge = 3600 * 24 * 7 // 7天
	}

	c.SetCookie("access_token", result.AccessToken, maxAge, "/", "", false, true)
	c.SetCookie("refresh_token", result.RefreshToken, maxAge*24*7, "/", "", false, true) // refresh token有效期更长

	h.SuccessWithMessage(c, "登录成功", result)
}

// EmailLogin 邮箱登录
func (h *AuthHandler) EmailLogin(c *gin.Context) {
	var req EmailLoginRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层邮箱登录
	result, err := h.authService.EmailLogin(&service.EmailLoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		IP:        h.GetClientIP(c),
		UserAgent: h.GetUserAgent(c),
		Remember:  req.Remember,
	})
	if err != nil {
		if strings.Contains(err.Error(), "不存在") || strings.Contains(err.Error(), "密码错误") {
			h.UnauthorizedError(c, "邮箱或密码错误")
		} else {
			h.InternalError(c, "登录失败: "+err.Error())
		}
		return
	}

	logger.Infof("用户邮箱登录成功: %s", result.User.Email)

	// 设置cookie
	maxAge := 3600 // 1小时
	if req.Remember {
		maxAge = 3600 * 24 * 7 // 7天
	}

	c.SetCookie("access_token", result.AccessToken, maxAge, "/", "", false, true)
	c.SetCookie("refresh_token", result.RefreshToken, maxAge*24*7, "/", "", false, true)

	h.SuccessWithMessage(c, "登录成功", result)
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 获取用户ID
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	// 获取token
	token := h.extractTokenFromRequest(c)
	if token != "" {
		// 调用服务层登出（将token加入黑名单）
		if err := h.authService.Logout(*userID, token); err != nil {
			logger.Errorf("用户登出失败: %v", err)
		}
	}

	// 清除cookie
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	logger.Infof("用户登出成功: %d", *userID)
	h.SuccessWithMessage(c, "登出成功", nil)
}

// RefreshToken 刷新token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 从cookie或请求头中获取refresh token
	refreshToken := h.extractRefreshTokenFromRequest(c)
	if refreshToken == "" {
		h.UnauthorizedError(c, "缺少refresh token")
		return
	}

	// 调用服务层刷新token
	result, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		h.UnauthorizedError(c, "token刷新失败: "+err.Error())
		return
	}

	logger.Infof("Token刷新成功: %d", result.User.ID)

	// 设置新的cookie
	c.SetCookie("access_token", result.AccessToken, 3600, "/", "", false, true)

	h.SuccessWithMessage(c, "Token刷新成功", result)
}

// GetProfile 获取用户资料
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	// 调用服务层获取用户资料
	user, err := h.authService.GetProfile(*userID)
	if err != nil {
		h.InternalError(c, "获取用户资料失败: "+err.Error())
		return
	}

	h.SuccessWithMessage(c, "获取用户资料成功", user.ToSafeResponse())
}

// UpdateProfile 更新用户资料
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	var req UpdateProfileRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层更新用户资料
	user, err := h.authService.UpdateProfile(*userID, &service.UpdateProfileRequest{
		Nickname: req.Nickname,
		Bio:      req.Bio,
		Avatar:   req.Avatar,
	})
	if err != nil {
		h.InternalError(c, "更新用户资料失败: "+err.Error())
		return
	}

	logger.Infof("用户资料更新成功: %d", *userID)
	h.SuccessWithMessage(c, "用户资料更新成功", user.ToSafeResponse())
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	var req ChangePasswordRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层修改密码
	err := h.authService.ChangePassword(*userID, &service.ChangePasswordRequest{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		if strings.Contains(err.Error(), "原密码错误") {
			h.UnauthorizedError(c, "原密码错误")
		} else {
			h.InternalError(c, "修改密码失败: "+err.Error())
		}
		return
	}

	logger.Infof("用户密码修改成功: %d", *userID)
	h.SuccessWithMessage(c, "密码修改成功", nil)
}

// ResetPassword 重置密码
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层重置密码
	err := h.authService.ResetPassword(&service.ResetPasswordRequest{
		Email: req.Email,
	})
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			h.NotFoundError(c, "邮箱未注册")
		} else {
			h.InternalError(c, "重置密码失败: "+err.Error())
		}
		return
	}

	logger.Infof("密码重置邮件发送成功: %s", req.Email)
	h.SuccessWithMessage(c, "密码重置邮件已发送，请查收邮件", nil)
}

// VerifyEmail 验证邮箱
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		h.ValidationError(c, "缺少验证token")
		return
	}

	// 调用服务层验证邮箱
	err := h.authService.VerifyEmail(token)
	if err != nil {
		if strings.Contains(err.Error(), "无效") || strings.Contains(err.Error(), "过期") {
			h.Error(c, http.StatusBadRequest, err.Error())
		} else {
			h.InternalError(c, "邮箱验证失败: "+err.Error())
		}
		return
	}

	logger.Infof("邮箱验证成功: %s", token)
	h.SuccessWithMessage(c, "邮箱验证成功", nil)
}

// extractTokenFromRequest 从请求中提取access token
func (h *AuthHandler) extractTokenFromRequest(c *gin.Context) string {
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

	return ""
}

// extractRefreshTokenFromRequest 从请求中提取refresh token
func (h *AuthHandler) extractRefreshTokenFromRequest(c *gin.Context) string {
	// 从Cookie中提取
	token, err := c.Cookie("refresh_token")
	if err == nil && token != "" {
		return token
	}

	// 从请求头中提取
	token = c.GetHeader("X-Refresh-Token")
	if token != "" {
		return token
	}

	return ""
}