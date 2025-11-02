package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Provider OAuth提供者接口
type Provider interface {
	Name() string
	GetAuthURL(state string, scopes []string) string
	ExchangeCodeForToken(ctx context.Context, code string) (*Token, error)
	GetUserInfo(ctx context.Context, token *Token) (*UserInfo, error)
	RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
}

// Token OAuth令牌
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// UserInfo OAuth用户信息
type UserInfo struct {
	ID       string            `json:"id"`
	Username string            `json:"username"`
	Nickname string            `json:"nickname"`
	Email    string            `json:"email"`
	Avatar   string            `json:"avatar"`
	Raw      map[string]interface{} `json:"raw"`
}

// ProviderConfig OAuth提供者配置
type ProviderConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	UserInfoURL  string   `json:"user_info_url"`
}

// OAuthService OAuth服务
type OAuthService struct {
	providers map[string]Provider
}

// NewOAuthService 创建OAuth服务实例
func NewOAuthService() *OAuthService {
	return &OAuthService{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider 注册OAuth提供者
func (s *OAuthService) RegisterProvider(provider Provider) {
	s.providers[provider.Name()] = provider
}

// GetProvider 获取OAuth提供者
func (s *OAuthService) GetProvider(name string) (Provider, error) {
	provider, exists := s.providers[name]
	if !exists {
		return nil, fmt.Errorf("不支持的OAuth提供者: %s", name)
	}
	return provider, nil
}

// ListProviders 列出所有已注册的提供者
func (s *OAuthService) ListProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// OAuthFlow OAuth流程
type OAuthFlow struct {
	service       *OAuthService
	stateManager StateManager
}

// StateManager 状态管理器接口
type StateManager interface {
	GenerateState() string
	ValidateState(state string) (bool, error)
	StoreState(state string, data interface{}) error
	GetState(state string) (interface{}, error)
	DeleteState(state string) error
}

// NewOAuthFlow 创建OAuth流程实例
func NewOAuthFlow(service *OAuthService, stateManager StateManager) *OAuthFlow {
	return &OAuthFlow{
		service:       service,
		stateManager: stateManager,
	}
}

// BeginAuth 开始OAuth认证流程
func (flow *OAuthFlow) BeginAuth(providerName, redirectURI string, scopes []string) (string, error) {
	// 生成state参数
	state := flow.stateManager.GenerateState()

	// 存储state信息
	stateData := map[string]interface{}{
		"provider_name": providerName,
		"redirect_uri":  redirectURI,
		"scopes":        scopes,
		"created_at":    time.Now(),
	}

	if err := flow.stateManager.StoreState(state, stateData); err != nil {
		return "", fmt.Errorf("存储state失败: %w", err)
	}

	// 获取OAuth提供者
	provider, err := flow.service.GetProvider(providerName)
	if err != nil {
		return "", err
	}

	// 生成认证URL
	authURL := provider.GetAuthURL(state, scopes)

	return authURL, nil
}

// HandleCallback 处理OAuth回调
func (flow *OAuthFlow) HandleCallback(ctx context.Context, state, code string) (*UserInfo, *Token, error) {
	// 验证state参数
	valid, err := flow.stateManager.ValidateState(state)
	if err != nil {
		return nil, nil, fmt.Errorf("验证state失败: %w", err)
	}
	if !valid {
		return nil, nil, fmt.Errorf("无效的state参数")
	}

	// 获取state信息
	stateData, err := flow.stateManager.GetState(state)
	if err != nil {
		return nil, nil, fmt.Errorf("获取state信息失败: %w", err)
	}

	// 解析state信息
	stateMap, ok := stateData.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("state信息格式错误")
	}

	providerName, ok := stateMap["provider_name"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("state中缺少provider_name")
	}

	// 获取OAuth提供者
	provider, err := flow.service.GetProvider(providerName)
	if err != nil {
		return nil, nil, err
	}

	// 用code换取access_token
	token, err := provider.ExchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("换取token失败: %w", err)
	}

	// 获取用户信息
	userInfo, err := provider.GetUserInfo(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 删除state
	flow.stateManager.DeleteState(state)

	return userInfo, token, nil
}

// RefreshToken 刷新访问令牌
func (flow *OAuthFlow) RefreshToken(ctx context.Context, providerName, refreshToken string) (*Token, error) {
	provider, err := flow.service.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.RefreshToken(ctx, refreshToken)
}

// 全局OAuth服务实例
var GlobalOAuthService *OAuthService

// Init 初始化OAuth服务
func Init() {
	GlobalOAuthService = NewOAuthService()

	// TODO: 注册具体的OAuth提供者
	// GlobalOAuthService.RegisterProvider(NewGitHubProvider(config.Get()))
	// GlobalOAuthService.RegisterProvider(NewGoogleProvider(config.Get()))
	// GlobalOAuthService.RegisterProvider(NewWeChatProvider(config.Get()))
}

// GetOAuthService 获取全局OAuth服务实例
func GetOAuthService() *OAuthService {
	if GlobalOAuthService == nil {
		Init()
	}
	return GlobalOAuthService
}

// CacheStateManager 基于缓存的状态管理器
type CacheStateManager struct {
	prefix string
	ttl    time.Duration
}

// NewCacheStateManager 创建缓存状态管理器
func NewCacheStateManager(prefix string) *CacheStateManager {
	return &CacheStateManager{
		prefix: prefix,
		ttl:    10 * time.Minute, // state有效期10分钟
	}
}

// GenerateState 生成state参数
func (m *CacheStateManager) GenerateState() string {
	// 生成随机state
	return generateRandomString(32)
}

// ValidateState 验证state参数
func (m *CacheStateManager) ValidateState(state string) (bool, error) {
	// 检查state是否存在
	data, err := m.GetState(state)
	if err != nil {
		return false, err
	}
	return data != nil, nil
}

// StoreState 存储state
func (m *CacheStateManager) StoreState(state string, data interface{}) error {
	// 这里需要导入cache包并使用
	// 暂时返回nil，实际实现需要调用cache.Set
	return nil
}

// GetState 获取state
func (m *CacheStateManager) GetState(state string) (interface{}, error) {
	// 这里需要导入cache包并使用
	// 暂时返回nil，实际实现需要调用cache.Get
	return nil, nil
}

// DeleteState 删除state
func (m *CacheStateManager) DeleteState(state string) error {
	// 这里需要导入cache包并使用
	// 暂时返回nil，实际实现需要调用cache.Delete
	return nil
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	// 简单实现，实际项目中应该使用crypto/rand
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		// 这里简化处理，实际应该使用随机数生成器
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

// ToJSON 转换为JSON字符串
func (t *Token) ToJSON() string {
	data, _ := json.Marshal(t)
	return string(data)
}

// ToJSON 转换为JSON字符串
func (u *UserInfo) ToJSON() string {
	data, _ := json.Marshal(u)
	return string(data)
}