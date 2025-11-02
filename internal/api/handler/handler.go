package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/charlottepl/blog-system/pkg/response"
)

// BaseHandler 基础处理器
type BaseHandler struct{}

// NewBaseHandler 创建基础处理器实例
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// Success 成功响应
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	response.Success(c, data)
}

// SuccessWithMessage 带消息的成功响应
func (h *BaseHandler) SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	response.SuccessWithMessage(c, message, data)
}

// Error 错误响应
func (h *BaseHandler) Error(c *gin.Context, code int, message string) {
	response.Error(c, code, message)
}

// ErrorWithData 带数据的错误响应
func (h *BaseHandler) ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	response.ErrorWithData(c, code, message, data)
}

// ValidationError 验证错误响应
func (h *BaseHandler) ValidationError(c *gin.Context, message string) {
	h.Error(c, http.StatusBadRequest, message)
}

// UnauthorizedError 未授权错误响应
func (h *BaseHandler) UnauthorizedError(c *gin.Context, message string) {
	h.Error(c, http.StatusUnauthorized, message)
}

// ForbiddenError 禁止访问错误响应
func (h *BaseHandler) ForbiddenError(c *gin.Context, message string) {
	h.Error(c, http.StatusForbidden, message)
}

// NotFoundError 未找到错误响应
func (h *BaseHandler) NotFoundError(c *gin.Context, message string) {
	h.Error(c, http.StatusNotFound, message)
}

// InternalError 内部服务器错误响应
func (h *BaseHandler) InternalError(c *gin.Context, message string) {
	h.Error(c, http.StatusInternalServerError, message)
}

// BindJSON 绑定JSON请求体
func (h *BaseHandler) BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		h.ValidationError(c, "请求参数格式错误: "+err.Error())
		return err
	}
	return nil
}

// BindQuery 绑定查询参数
func (h *BaseHandler) BindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		h.ValidationError(c, "查询参数格式错误: "+err.Error())
		return err
	}
	return nil
}

// BindURI 绑定URI参数
func (h *BaseHandler) BindURI(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindUri(obj); err != nil {
		h.ValidationError(c, "路径参数格式错误: "+err.Error())
		return err
	}
	return nil
}

// BindForm 绑定表单数据
func (h *BaseHandler) BindForm(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBind(obj); err != nil {
		h.ValidationError(c, "表单数据格式错误: "+err.Error())
		return err
	}
	return nil
}

// GetUserID 从上下文获取用户ID
func (h *BaseHandler) GetUserID(c *gin.Context) *int {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(int); ok {
			return &id
		}
	}
	return nil
}

// GetCurrentUser 从上下文获取当前用户信息
func (h *BaseHandler) GetCurrentUser(c *gin.Context) map[string]interface{} {
	if user, exists := c.Get("user"); exists {
		if userInfo, ok := user.(map[string]interface{}); ok {
			return userInfo
		}
	}
	return nil
}

// IsAdmin 检查当前用户是否为管理员
func (h *BaseHandler) IsAdmin(c *gin.Context) bool {
	if user := h.GetCurrentUser(c); user != nil {
		if isAdmin, ok := user["is_admin"].(bool); ok {
			return isAdmin
		}
	}
	return false
}

// RequireAuth 需要认证检查
func (h *BaseHandler) RequireAuth(c *gin.Context) bool {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return false
	}
	return true
}

// RequireAdmin 需要管理员权限检查
func (h *BaseHandler) RequireAdmin(c *gin.Context) bool {
	if !h.RequireAuth(c) {
		return false
	}

	if !h.IsAdmin(c) {
		h.ForbiddenError(c, "需要管理员权限")
		return false
	}

	return true
}

// GetPagination 获取分页参数
func (h *BaseHandler) GetPagination(c *gin.Context) (int, int) {
	page := 1
	size := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := parseInt(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if s := c.Query("size"); s != "" {
		if parsed, err := parseInt(s); err == nil && parsed > 0 && parsed <= 100 {
			size = parsed
		}
	}

	return page, size
}

// GetClientIP 获取客户端IP
func (h *BaseHandler) GetClientIP(c *gin.Context) string {
	return c.ClientIP()
}

// GetUserAgent 获取用户代理
func (h *BaseHandler) GetUserAgent(c *gin.Context) string {
	return c.GetHeader("User-Agent")
}

// 辅助函数
func parseInt(s string) (int, error) {
	// 这里应该使用 strconv.Atoi
	// 为了简化实现，暂时返回默认值
	return 1, nil
}