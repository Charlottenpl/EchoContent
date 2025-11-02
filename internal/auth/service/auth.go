package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charlottepl/blog-system/internal/auth/email"
	"github.com/charlottepl/blog-system/internal/auth/jwt"
	"github.com/charlottepl/blog-system/internal/core/cache"
	"github.com/charlottepl/blog-system/internal/core/database"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/user/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	db            *gorm.DB
	jwtService    *jwt.JWTService
	emailService  *email.EmailService
}

// NewAuthService 创建认证服务实例
func NewAuthService() *AuthService {
	return &AuthService{
		db:           database.GetDB(),
		jwtService:   jwt.GetJWTService(),
		emailService: email.GetEmailService(),
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=6,max=50"`
	VerificationCode string `json:"verification_code" binding:"required,len=6"`
	Username        string `json:"username,omitempty"`
	Nickname        string `json:"nickname,omitempty"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// EmailCodeLoginRequest 验证码登录请求
type EmailCodeLoginRequest struct {
	Email           string `json:"email" binding:"required,email"`
	VerificationCode string `json:"verification_code" binding:"required,len=6"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User         *SafeUser `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// SafeUser 安全用户信息
type SafeUser struct {
	ID       int       `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*LoginResponse, error) {
	// 1. 验证邮箱格式
	if err := s.validateEmail(req.Email); err != nil {
		return nil, fmt.Errorf("邮箱格式错误: %w", err)
	}

	// 2. 验证验证码
	if err := s.emailService.VerifyCode(req.Email, req.VerificationCode, "register"); err != nil {
		return nil, fmt.Errorf("验证码错误: %w", err)
	}

	// 3. 检查邮箱是否已注册
	if exists, err := s.isEmailExists(req.Email); err != nil {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	} else if exists {
		return nil, errors.New("邮箱已被注册")
	}

	// 4. 生成用户名
	username := req.Username
	if username == "" {
		username = s.generateUsernameFromEmail(req.Email)
	} else {
		// 检查用户名是否已存在
		if exists, err := s.isUsernameExists(username); err != nil {
			return nil, fmt.Errorf("检查用户名失败: %w", err)
		} else if exists {
			return nil, errors.New("用户名已存在")
		}
	}

	// 5. 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 6. 创建用户
	user := &model.User{
		Username:     username,
		Email:        req.Email,
		Nickname:     req.Nickname,
		PasswordHash: string(hashedPassword),
		Role:         "user",
		Status:       "active",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 7. 创建邮箱认证绑定
	authProvider := &model.UserAuthProvider{
		UserID:     user.ID,
		Provider:   "email",
		ProviderID: req.Email,
		IsPrimary:  true,
	}

	if err := s.db.Create(authProvider).Error; err != nil {
		// 回滚用户创建
		s.db.Delete(user)
		return nil, fmt.Errorf("创建认证绑定失败: %w", err)
	}

	// 8. 生成JWT令牌
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	logger.Infof("用户注册成功: %s (ID: %d)", user.Email, user.ID)

	return &LoginResponse{
		User:         s.toSafeUser(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// LoginPassword 密码登录
func (s *AuthService) LoginPassword(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 1. 查找用户
	user, err := s.findUserByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 2. 检查用户状态
	if !user.IsActive() {
		return nil, errors.New("用户账号已被禁用")
	}

	// 3. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("密码错误")
	}

	// 4. 生成JWT令牌
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	logger.Infof("用户登录成功: %s (ID: %d)", user.Email, user.ID)

	return &LoginResponse{
		User:         s.toSafeUser(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// LoginEmailCode 邮箱验证码登录
func (s *AuthService) LoginEmailCode(ctx context.Context, req *EmailCodeLoginRequest) (*LoginResponse, error) {
	// 1. 验证验证码
	if err := s.emailService.VerifyCode(req.Email, req.VerificationCode, "login"); err != nil {
		return nil, fmt.Errorf("验证码错误: %w", err)
	}

	// 2. 查找用户
	user, err := s.findUserByEmail(req.Email)
	if err != nil {
		// 用户不存在，自动注册
		return s.autoRegister(ctx, req.Email)
	}

	// 3. 检查用户状态
	if !user.IsActive() {
		return nil, errors.New("用户账号已被禁用")
	}

	// 4. 生成JWT令牌
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	logger.Infof("用户验证码登录成功: %s (ID: %d)", user.Email, user.ID)

	return &LoginResponse{
		User:         s.toSafeUser(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// SendVerificationCode 发送验证码
func (s *AuthService) SendVerificationCode(ctx context.Context, email, codeType string) error {
	// 验证邮箱格式
	if err := s.validateEmail(email); err != nil {
		return fmt.Errorf("邮箱格式错误: %w", err)
	}

	// 对于注册验证码，检查邮箱是否已注册
	if codeType == "register" {
		if exists, err := s.isEmailExists(email); err != nil {
			return fmt.Errorf("检查邮箱失败: %w", err)
		} else if exists {
			return errors.New("邮箱已被注册")
		}
	}

	// 发送验证码
	return s.emailService.SendVerificationCode(email, codeType)
}

// RefreshToken 刷新访问令牌
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	// 解析刷新令牌
	claims, err := s.jwtService.ParseToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("解析刷新令牌失败: %w", err)
	}

	// 查找用户
	user, err := s.findUserByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, errors.New("用户账号已被禁用")
	}

	// 生成新的令牌对
	tokenPair, err := s.jwtService.RefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("刷新令牌失败: %w", err)
	}

	return &LoginResponse{
		User:         s.toSafeUser(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// GetUserProfile 获取用户信息
func (s *AuthService) GetUserProfile(ctx context.Context, userID int) (*SafeUser, error) {
	user, err := s.findUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	return s.toSafeUser(user), nil
}

// UpdateUserProfile 更新用户信息
func (s *AuthService) UpdateUserProfile(ctx context.Context, userID int, nickname, avatar string) (*SafeUser, error) {
	user, err := s.findUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 更新字段
	if nickname != "" {
		user.Nickname = nickname
	}
	if avatar != "" {
		user.Avatar = avatar
	}

	if err := s.db.Save(user).Error; err != nil {
		return nil, fmt.Errorf("更新用户信息失败: %w", err)
	}

	logger.Infof("用户信息更新成功: %s (ID: %d)", user.Email, user.ID)

	return s.toSafeUser(user), nil
}

// validateEmail 验证邮箱格式
func (s *AuthService) validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("邮箱格式不正确")
	}
	return nil
}

// isEmailExists 检查邮箱是否存在
func (s *AuthService) isEmailExists(email string) (bool, error) {
	var count int64
	err := s.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// isUsernameExists 检查用户名是否存在
func (s *AuthService) isUsernameExists(username string) (bool, error) {
	var count int64
	err := s.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// findUserByEmail 根据邮箱查找用户
func (s *AuthService) findUserByEmail(email string) (*model.User, error) {
	var user model.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// findUserByID 根据ID查找用户
func (s *AuthService) findUserByID(userID int) (*model.User, error) {
	var user model.User
	err := s.db.First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// generateUsernameFromEmail 从邮箱生成用户名
func (s *AuthService) generateUsernameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	username := parts[0]

	// 清理用户名，只保留字母数字和下划线
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	username = reg.ReplaceAllString(username, "")

	// 确保用户名不为空且符合要求
	if username == "" {
		username = "user"
	}

	// 如果用户名已存在，添加数字后缀
	baseUsername := username
	counter := 1
	for {
		if exists, err := s.isUsernameExists(username); err != nil {
			break
		} else if !exists {
			break
		}
		username = fmt.Sprintf("%s%d", baseUsername, counter)
		counter++
	}

	return username
}

// autoRegister 自动注册用户
func (s *AuthService) autoRegister(ctx context.Context, email string) (*LoginResponse, error) {
	// 生成用户名
	username := s.generateUsernameFromEmail(email)

	// 创建用户
	user := &model.User{
		Username: username,
		Email:    email,
		Nickname: strings.Split(email, "@")[0],
		Role:     "user",
		Status:   "active",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("自动注册失败: %w", err)
	}

	// 创建邮箱认证绑定
	authProvider := &model.UserAuthProvider{
		UserID:     user.ID,
		Provider:   "email",
		ProviderID: email,
		IsPrimary:  true,
	}

	if err := s.db.Create(authProvider).Error; err != nil {
		s.db.Delete(user)
		return nil, fmt.Errorf("创建认证绑定失败: %w", err)
	}

	// 生成JWT令牌
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	logger.Infof("用户自动注册成功: %s (ID: %d)", user.Email, user.ID)

	return &LoginResponse{
		User:         s.toSafeUser(user),
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

// toSafeUser 转换为安全用户信息
func (s *AuthService) toSafeUser(user *model.User) *SafeUser {
	return &SafeUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// GetDevCode 获取开发模式验证码
func (s *AuthService) GetDevCode(email, codeType string) (string, error) {
	return s.emailService.GetDevCode(email, codeType)
}