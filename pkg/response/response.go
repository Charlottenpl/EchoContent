package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// PageResponse 分页响应结构
type PageResponse struct {
	List       interface{} `json:"list"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page     int   `json:"page"`
	Size     int   `json:"size"`
	Total    int64 `json:"total"`
	Pages    int   `json:"pages"`
	HasNext  bool  `json:"has_next"`
	HasPrev  bool  `json:"has_prev"`
}

// 状态码常量
const (
	CodeSuccess              = 0
	CodeBadRequest          = 1001
	CodeUnauthorized        = 1002
	CodeForbidden           = 1003
	CodeNotFound            = 1004
	CodeInternalServerError = 1005
	CodeDatabaseError       = 1006
	CodeFileError           = 1007
	CodeValidationError     = 1008
	CodeRateLimitExceeded   = 1009
	CodeServiceUnavailable  = 1010
)

// 状态码消息映射
var codeMessages = map[int]string{
	CodeSuccess:              "success",
	CodeBadRequest:          "参数错误",
	CodeUnauthorized:        "认证失败",
	CodeForbidden:           "权限不足",
	CodeNotFound:            "资源不存在",
	CodeInternalServerError: "服务器内部错误",
	CodeDatabaseError:       "数据库操作失败",
	CodeFileError:           "文件操作失败",
	CodeValidationError:     "数据验证失败",
	CodeRateLimitExceeded:   "请求过于频繁",
	CodeServiceUnavailable:  "服务暂不可用",
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      CodeSuccess,
		Message:   codeMessages[CodeSuccess],
		Data:      data,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      CodeSuccess,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})
}

// PageSuccess 分页成功响应
func PageSuccess(c *gin.Context, list interface{}, pagination Pagination) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Message: codeMessages[CodeSuccess],
		Data: PageResponse{
			List:       list,
			Pagination: pagination,
		},
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})
}

// Error 错误响应
func Error(c *gin.Context, code int) {
	message, exists := codeMessages[code]
	if !exists {
		message = "未知错误"
	}

	httpStatus := getHTTPStatus(code)
	c.JSON(httpStatus, Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ErrorWithMessage 带消息的错误响应
func ErrorWithMessage(c *gin.Context, code int, message string) {
	httpStatus := getHTTPStatus(code)
	c.JSON(httpStatus, Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ErrorWithData 带数据的错误响应
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	httpStatus := getHTTPStatus(code)
	c.JSON(httpStatus, Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	})
}

// BadRequest 400错误响应
func BadRequest(c *gin.Context, message ...string) {
	msg := "参数错误"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeBadRequest, msg)
}

// Unauthorized 401错误响应
func Unauthorized(c *gin.Context, message ...string) {
	msg := "认证失败"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeUnauthorized, msg)
}

// Forbidden 403错误响应
func Forbidden(c *gin.Context, message ...string) {
	msg := "权限不足"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeForbidden, msg)
}

// NotFound 404错误响应
func NotFound(c *gin.Context, message ...string) {
	msg := "资源不存在"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeNotFound, msg)
}

// InternalServerError 500错误响应
func InternalServerError(c *gin.Context, message ...string) {
	msg := "服务器内部错误"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeInternalServerError, msg)
}

// DatabaseError 数据库错误响应
func DatabaseError(c *gin.Context, message ...string) {
	msg := "数据库操作失败"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeDatabaseError, msg)
}

// ValidationError 数据验证错误响应
func ValidationError(c *gin.Context, message ...string) {
	msg := "数据验证失败"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeValidationError, msg)
}

// RateLimitExceeded 请求限流错误响应
func RateLimitExceeded(c *gin.Context, message ...string) {
	msg := "请求过于频繁"
	if len(message) > 0 {
		msg = message[0]
	}
	ErrorWithMessage(c, CodeRateLimitExceeded, msg)
}

// getHTTPStatus 根据业务错误码获取HTTP状态码
func getHTTPStatus(code int) int {
	switch code {
	case CodeSuccess:
		return http.StatusOK
	case CodeBadRequest, CodeValidationError:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// NewPagination 创建分页信息
func NewPagination(page, size int, total int64) Pagination {
	pages := int((total + int64(size) - 1) / int64(size))
	if pages < 1 {
		pages = 1
	}

	return Pagination{
		Page:    page,
		Size:    size,
		Total:   total,
		Pages:   pages,
		HasNext: page < pages,
		HasPrev: page > 1,
	}
}