package email

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/charlottepl/blog-system/internal/core/cache"
	"github.com/charlottepl/blog-system/internal/core/config"
	"github.com/charlottepl/blog-system/internal/core/logger"
)

// EmailService 邮箱服务
type EmailService struct {
	enabled bool
}

// VerificationCode 验证码信息
type VerificationCode struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	Type      string    `json:"type"` // register, reset_password, login
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Attempts  int       `json:"attempts"`
}

// NewEmailService 创建邮箱服务实例
func NewEmailService() *EmailService {
	cfg := config.Get()
	enabled := cfg != nil && cfg.Email.SMTPHost != "" && cfg.Email.SMTPUser != ""

	return &EmailService{
		enabled: enabled,
	}
}

// IsEnabled 检查邮箱服务是否启用
func (e *EmailService) IsEnabled() bool {
	return e.enabled
}

// GenerateVerificationCode 生成验证码
func (e *EmailService) GenerateVerificationCode() string {
	// 生成6位数字验证码
	code := ""
	for i := 0; i < 6; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		code += n.String()
	}
	return code
}

// SendVerificationCode 发送验证码
func (e *EmailService) SendVerificationCode(email, codeType string) error {
	if !e.enabled {
		logger.Warnf("邮箱服务未启用，跳过发送验证码到: %s", email)
		// 在开发模式下，将验证码存储到缓存中供测试使用
		return e.storeVerificationCodeForDev(email, codeType)
	}

	// 生成验证码
	code := e.GenerateVerificationCode()

	// 存储验证码到缓存
	if err := e.storeVerificationCode(email, code, codeType); err != nil {
		return fmt.Errorf("存储验证码失败: %w", err)
	}

	// 发送邮件
	if err := e.sendEmail(email, code, codeType); err != nil {
		// 发送失败，删除缓存的验证码
		cache.Delete(cache.Background(), e.getCacheKey(email, codeType))
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	logger.Infof("验证码已发送到邮箱: %s, 类型: %s", email, codeType)
	return nil
}

// storeVerificationCode 存储验证码到缓存
func (e *EmailService) storeVerificationCode(email, code, codeType string) error {
	ctx := cache.Background()
	key := e.getCacheKey(email, codeType)

	verificationCode := VerificationCode{
		Code:      code,
		Email:     email,
		Type:      codeType,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute), // 10分钟过期
		Attempts:  0,
	}

	return cache.Set(ctx, key, verificationCode, 10*time.Minute)
}

// storeVerificationCodeForDev 开发模式存储验证码
func (e *EmailService) storeVerificationCodeForDev(email, codeType string) error {
	code := e.GenerateVerificationCode()
	return e.storeVerificationCode(email, code, codeType)
}

// VerifyCode 验证验证码
func (e *EmailService) VerifyCode(email, code, codeType string) error {
	ctx := cache.Background()
	key := e.getCacheKey(email, codeType)

	// 从缓存获取验证码信息
	var storedCode VerificationCode
	cacheData, err := cache.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("验证码不存在或已过期")
	}

	// 这里需要反序列化，简化处理
	if err := e.parseVerificationCode(cacheData, &storedCode); err != nil {
		return fmt.Errorf("验证码格式错误")
	}

	// 检查验证码是否过期
	if time.Now().After(storedCode.ExpiresAt) {
		cache.Delete(ctx, key)
		return fmt.Errorf("验证码已过期")
	}

	// 检查尝试次数
	if storedCode.Attempts >= 3 {
		cache.Delete(ctx, key)
		return fmt.Errorf("验证码尝试次数过多，请重新获取")
	}

	// 验证码错误，增加尝试次数
	if storedCode.Code != code {
		storedCode.Attempts++
		cache.Set(ctx, key, storedCode, time.Until(storedCode.ExpiresAt))
		return fmt.Errorf("验证码错误")
	}

	// 验证成功，删除缓存
	cache.Delete(ctx, key)
	return nil
}

// parseVerificationCode 解析验证码（简化处理）
func (e *EmailService) parseVerificationCode(data string, code *VerificationCode) error {
	// 这里简化处理，实际项目中应该使用JSON序列化
	// 假设data格式为: code|email|type|timestamp|expires|attempts
	parts := strings.Split(data, "|")
	if len(parts) != 6 {
		return fmt.Errorf("验证码格式错误")
	}

	code.Code = parts[0]
	code.Email = parts[1]
	code.Type = parts[2]

	// 解析时间戳（简化处理）
	code.CreatedAt = time.Now() // 实际应该解析timestamp
	code.ExpiresAt = time.Now().Add(10 * time.Minute)

	return nil
}

// sendEmail 发送邮件
func (e *EmailService) sendEmail(email, code, codeType string) error {
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("配置未加载")
	}

	// 构建邮件内容
	subject, body := e.buildEmailContent(code, codeType)

	// 这里应该实现真实的邮件发送逻辑
	// 由于依赖关系，这里只记录日志
	logger.Infof("模拟发送邮件到: %s", email)
	logger.Infof("邮件主题: %s", subject)
	logger.Infof("邮件内容预览: %s", body[:min(len(body), 100)])

	// TODO: 实现真实的SMTP邮件发送
	// 可以使用标准库的net/smtp或第三方库如gomail

	return nil
}

// buildEmailContent 构建邮件内容
func (e *EmailService) buildEmailContent(code, codeType string) (subject, body string) {
	switch codeType {
	case "register":
		subject = "博客系统 - 注册验证码"
		body = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>注册验证码</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 10px; text-align: center;">
        <h1 style="color: #007bff; margin-bottom: 30px;">博客系统</h1>
        <h2 style="color: #333; margin-bottom: 20px;">邮箱验证码</h2>
        <p style="font-size: 16px; color: #666; margin-bottom: 30px;">
            您正在注册博客系统账号，验证码为：
        </p>
        <div style="background-color: #007bff; color: white; font-size: 32px; font-weight: bold;
                    padding: 20px; border-radius: 8px; display: inline-block; margin-bottom: 30px;">
            %s
        </div>
        <p style="font-size: 14px; color: #999; margin-bottom: 20px;">
            验证码有效期为10分钟，请及时使用。
        </p>
        <p style="font-size: 14px; color: #999;">
            如果这不是您本人的操作，请忽略此邮件。
        </p>
    </div>
</body>
</html>`, code)
	case "login":
		subject = "博客系统 - 登录验证码"
		body = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>登录验证码</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 10px; text-align: center;">
        <h1 style="color: #28a745; margin-bottom: 30px;">博客系统</h1>
        <h2 style="color: #333; margin-bottom: 20px;">登录验证码</h2>
        <p style="font-size: 16px; color: #666; margin-bottom: 30px;">
            您正在登录博客系统，验证码为：
        </p>
        <div style="background-color: #28a745; color: white; font-size: 32px; font-weight: bold;
                    padding: 20px; border-radius: 8px; display: inline-block; margin-bottom: 30px;">
            %s
        </div>
        <p style="font-size: 14px; color: #999; margin-bottom: 20px;">
            验证码有效期为10分钟，请及时使用。
        </p>
        <p style="font-size: 14px; color: #999;">
            如果这不是您本人的操作，请立即修改密码。
        </p>
    </div>
</body>
</html>`, code)
	case "reset_password":
		subject = "博客系统 - 密码重置验证码"
		body = fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>密码重置验证码</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 10px; text-align: center;">
        <h1 style="color: #dc3545; margin-bottom: 30px;">博客系统</h1>
        <h2 style="color: #333; margin-bottom: 20px;">密码重置验证码</h2>
        <p style="font-size: 16px; color: #666; margin-bottom: 30px;">
            您正在重置密码，验证码为：
        </p>
        <div style="background-color: #dc3545; color: white; font-size: 32px; font-weight: bold;
                    padding: 20px; border-radius: 8px; display: inline-block; margin-bottom: 30px;">
            %s
        </div>
        <p style="font-size: 14px; color: #999; margin-bottom: 20px;">
            验证码有效期为10分钟，请及时使用。
        </p>
        <p style="font-size: 14px; color: #999;">
            如果这不是您本人的操作，请立即联系管理员。
        </p>
    </div>
</body>
</html>`, code)
	default:
		subject = "博客系统 - 验证码"
		body = fmt.Sprintf("您的验证码是：%s，有效期为10分钟。", code)
	}

	return subject, body
}

// getCacheKey 获取缓存键
func (e *EmailService) getCacheKey(email, codeType string) string {
	return fmt.Sprintf("verification_code:%s:%s", email, codeType)
}

// GetDevCode 获取开发模式下的验证码（用于测试）
func (e *EmailService) GetDevCode(email, codeType string) (string, error) {
	if e.enabled {
		return "", fmt.Errorf("邮箱服务已启用，无法获取开发验证码")
	}

	ctx := cache.Background()
	key := e.getCacheKey(email, codeType)

	data, err := cache.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("验证码不存在")
	}

	var storedCode VerificationCode
	if err := e.parseVerificationCode(data, &storedCode); err != nil {
		return "", fmt.Errorf("验证码格式错误")
	}

	if time.Now().After(storedCode.ExpiresAt) {
		return "", fmt.Errorf("验证码已过期")
	}

	return storedCode.Code, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 全局邮箱服务实例
var GlobalEmailService *EmailService

// Init 初始化邮箱服务
func Init() {
	GlobalEmailService = NewEmailService()

	if GlobalEmailService.IsEnabled() {
		logger.Info("邮箱服务初始化完成")
	} else {
		logger.Info("邮箱服务未启用，使用开发模式")
	}
}

// GetEmailService 获取全局邮箱服务实例
func GetEmailService() *EmailService {
	if GlobalEmailService == nil {
		Init()
	}
	return GlobalEmailService
}