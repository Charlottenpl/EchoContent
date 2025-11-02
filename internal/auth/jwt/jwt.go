package jwt

import (
	"fmt"
	"time"

	"github.com/charlottepl/blog-system/internal/core/config"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT claims结构
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair JWT令牌对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// JWTService JWT服务
type JWTService struct {
	secretKey          []byte
	expireHours        int
	refreshExpireDays  int
	issuer             string
}

// NewJWTService 创建JWT服务实例
func NewJWTService() *JWTService {
	cfg := config.Get()
	if cfg == nil {
		logger.Fatal("配置未加载，无法创建JWT服务")
	}

	return &JWTService{
		secretKey:         []byte(cfg.JWT.Secret),
		expireHours:       cfg.JWT.ExpireHours,
		refreshExpireDays: cfg.JWT.RefreshExpireDays,
		issuer:            cfg.JWT.Issuer,
	}
}

// GenerateTokenPair 生成访问令牌和刷新令牌
func (j *JWTService) GenerateTokenPair(userID int, username, role string) (*TokenPair, error) {
	now := time.Now()

	// 生成访问令牌
	accessToken, err := j.generateToken(userID, username, role, now.Add(time.Duration(j.expireHours)*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	// 生成刷新令牌
	refreshToken, err := j.generateToken(userID, username, role, now.Add(time.Duration(j.refreshExpireDays)*24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    now.Add(time.Duration(j.expireHours) * time.Hour),
	}, nil
}

// generateToken 生成JWT令牌
func (j *JWTService) generateToken(userID int, username, role string, expiresAt time.Time) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        fmt.Sprintf("%d_%d", userID, time.Now().Unix()),
			Issuer:    j.issuer,
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  []string{"blog-system"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ParseToken 解析JWT令牌
func (j *JWTService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析令牌失败: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("无效的令牌")
}

// ValidateToken 验证JWT令牌
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 检查令牌是否过期
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("令牌已过期")
	}

	// 检查发行者
	if claims.Issuer != j.issuer {
		return nil, fmt.Errorf("无效的发行者")
	}

	return claims, nil
}

// RefreshToken 刷新访问令牌
func (j *JWTService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	// 解析刷新令牌
	claims, err := j.ParseToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("解析刷新令牌失败: %w", err)
	}

	// 验证刷新令牌是否过期
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("刷新令牌已过期")
	}

	// 生成新的令牌对
	return j.GenerateTokenPair(claims.UserID, claims.Username, claims.Role)
}

// ExtractTokenFromHeader 从HTTP头部提取令牌
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("授权头部为空")
	}

	// 检查Bearer前缀
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("无效的授权头部格式")
	}

	return authHeader[len(bearerPrefix):], nil
}

// IsAdmin 检查用户是否为管理员
func IsAdmin(claims *Claims) bool {
	return claims.Role == "admin"
}

// IsActive 检查用户是否激活
func IsActive(claims *Claims) bool {
	return true // 这里可以扩展用户状态检查
}

// GetUserID 从令牌中获取用户ID
func GetUserID(tokenString string) (int, error) {
	service := NewJWTService()
	claims, err := service.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// GetUserInfo 从令牌中获取用户信息
func GetUserInfo(tokenString string) (*Claims, error) {
	service := NewJWTService()
	return service.ValidateToken(tokenString)
}

// 全局JWT服务实例
var GlobalJWTService *JWTService

// Init 初始化JWT服务
func Init() {
	GlobalJWTService = NewJWTService()
	logger.Info("JWT服务初始化完成")
}

// GetJWTService 获取全局JWT服务实例
func GetJWTService() *JWTService {
	if GlobalJWTService == nil {
		Init()
	}
	return GlobalJWTService
}