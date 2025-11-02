package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/charlottepl/blog-system/internal/blog/service"
	"github.com/charlottepl/blog-system/internal/core/logger"
	"github.com/charlottepl/blog-system/internal/core/validator"
)

// BlogHandler 博客处理器
type BlogHandler struct {
	*BaseHandler
	postService    *service.PostService
	categoryService *service.CategoryService
	tagService     *service.TagService
}

// NewBlogHandler 创建博客处理器实例
func NewBlogHandler() *BlogHandler {
	return &BlogHandler{
		BaseHandler:    NewBaseHandler(),
		postService:    service.NewPostService(),
		categoryService: service.NewCategoryService(),
		tagService:     service.NewTagService(),
	}
}

// CreatePostRequest 创建文章请求
type CreatePostRequest struct {
	Title       string   `json:"title" validate:"required,min=1,max=200"`
	Content     string   `json:"content" validate:"required,min=1"`
	Excerpt     string   `json:"excerpt" validate:"max=500"`
	CategoryID  *int     `json:"category_id"`
	TagNames    []string `json:"tag_names"`
	Status      string   `json:"status" validate:"oneof=draft published private"`
	Featured    bool     `json:"featured"`
	SEOKeywords string   `json:"seo_keywords" validate:"max=200"`
	SEODesc     string   `json:"seo_desc" validate:"max:300"`
}

// UpdatePostRequest 更新文章请求
type UpdatePostRequest struct {
	Title       string   `json:"title" validate:"min=1,max=200"`
	Content     string   `json:"content" validate:"min=1"`
	Excerpt     string   `json:"excerpt" validate:"max=500"`
	CategoryID  *int     `json:"category_id"`
	TagNames    []string `json:"tag_names"`
	Status      string   `json:"status" validate:"oneof=draft published private"`
	Featured    bool     `json:"featured"`
	SEOKeywords string   `json:"seo_keywords" validate:"max=200"`
	SEODesc     string   `json:"seo_desc" validate:"max:300"`
}

// PostListRequest 文章列表请求
type PostListRequest struct {
	Page       int    `form:"page,default=1" validate:"min=1"`
	Size       int    `form:"size,default=20" validate:"min=1,max=100"`
	Status     string `form:"status" validate:"omitempty,oneof=draft published private"`
	CategoryID *int   `form:"category_id"`
	AuthorID   *int   `form:"author_id"`
	TagID      *int   `form:"tag_id"`
	Keyword    string `form:"keyword"`
	SortBy     string `form:"sort_by" validate:"omitempty,oneof=created_at updated_at view_count like_count"`
	SortOrder  string `form:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=50"`
	Slug        string `json:"slug" validate:"max=50"`
	Description string `json:"description" validate:"max=200"`
	ParentID    *int   `json:"parent_id"`
	Color       string `json:"color" validate:"max=7"`
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name        string `json:"name" validate:"min=1,max=50"`
	Slug        string `json:"slug" validate:"max=50"`
	Description string `json:"description" validate:"max=200"`
	ParentID    *int   `json:"parent_id"`
	Color       string `json:"color" validate:"max=7"`
}

// CreateTagRequest 创建标签请求
type CreateTagRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=30"`
	Slug        string `json:"slug" validate:"max=30"`
	Description string `json:"description" validate:"max=200"`
	Color       string `json:"color" validate:"max=7"`
}

// UpdateTagRequest 更新标签请求
type UpdateTagRequest struct {
	Name        string `json:"name" validate:"min=1,max=30"`
	Slug        string `json:"slug" validate:"max=30"`
	Description string `json:"description" validate:"max=200"`
	Color       string `json:"color" validate:"max=7"`
}

// GetPosts 获取文章列表
func (h *BlogHandler) GetPosts(c *gin.Context) {
	var req PostListRequest
	if err := h.BindQuery(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 获取用户信息（可选认证）
	userID := h.GetUserID(c)

	// 构建过滤器
	filters := &service.PostFilters{
		Status:     req.Status,
		CategoryID: req.CategoryID,
		AuthorID:   req.AuthorID,
		Keyword:    req.Keyword,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}

	if req.TagID != nil {
		filters.TagIDs = []int{*req.TagID}
	}

	// 调用服务层获取文章列表
	posts, total, err := h.postService.GetPosts(filters, req.Page, req.Size, userID)
	if err != nil {
		h.InternalError(c, "获取文章列表失败: "+err.Error())
		return
	}

	// 构建分页信息
	pagination := map[string]interface{}{
		"page":    req.Page,
		"size":    req.Size,
		"total":   total,
		"pages":   (total + int64(req.Size) - 1) / int64(req.Size),
		"has_next": req.Page*req.Size < int(total),
		"has_prev": req.Page > 1,
	}

	h.SuccessWithData(c, gin.H{
		"posts":      posts,
		"pagination": pagination,
	})
}

// GetPost 获取单篇文章
func (h *BlogHandler) GetPost(c *gin.Context) {
	// 获取文章ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.ValidationError(c, "无效的文章ID")
		return
	}

	// 获取用户信息（可选认证）
	userID := h.GetUserID(c)

	// 调用服务层获取文章
	post, err := h.postService.GetPostByID(id, userID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			h.NotFoundError(c, "文章不存在")
		} else if strings.Contains(err.Error(), "权限") {
			h.ForbiddenError(c, "无权限访问该文章")
		} else {
			h.InternalError(c, "获取文章失败: "+err.Error())
		}
		return
	}

	// 如果是已发布的文章，增加浏览量
	if post.Status == "published" {
		if err := h.postService.IncrementViewCount(id); err != nil {
			logger.Errorf("增加文章浏览量失败: %v", err)
		}
	}

	h.SuccessWithMessage(c, "获取文章成功", post)
}

// CreatePost 创建文章
func (h *BlogHandler) CreatePost(c *gin.Context) {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	var req CreatePostRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层创建文章
	post, err := h.postService.CreatePost(&service.CreatePostRequest{
		Title:       req.Title,
		Content:     req.Content,
		Excerpt:     req.Excerpt,
		CategoryID:  req.CategoryID,
		TagNames:    req.TagNames,
		Status:      req.Status,
		Featured:    req.Featured,
		SEOKeywords: req.SEOKeywords,
		SEODesc:     req.SEODesc,
		AuthorID:    *userID,
	})
	if err != nil {
		h.InternalError(c, "创建文章失败: "+err.Error())
		return
	}

	logger.Infof("文章创建成功: %s (ID: %d, 作者ID: %d)", post.Title, post.ID, *userID)
	h.SuccessWithMessage(c, "文章创建成功", post)
}

// UpdatePost 更新文章
func (h *BlogHandler) UpdatePost(c *gin.Context) {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	// 获取文章ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.ValidationError(c, "无效的文章ID")
		return
	}

	var req UpdatePostRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层更新文章
	post, err := h.postService.UpdatePost(id, *userID, &service.UpdatePostRequest{
		Title:       req.Title,
		Content:     req.Content,
		Excerpt:     req.Excerpt,
		CategoryID:  req.CategoryID,
		TagNames:    req.TagNames,
		Status:      req.Status,
		Featured:    req.Featured,
		SEOKeywords: req.SEOKeywords,
		SEODesc:     req.SEODesc,
	})
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			h.NotFoundError(c, "文章不存在")
		} else if strings.Contains(err.Error(), "权限") {
			h.ForbiddenError(c, "无权限编辑该文章")
		} else {
			h.InternalError(c, "更新文章失败: "+err.Error())
		}
		return
	}

	logger.Infof("文章更新成功: %s (ID: %d, 作者ID: %d)", post.Title, post.ID, *userID)
	h.SuccessWithMessage(c, "文章更新成功", post)
}

// DeletePost 删除文章
func (h *BlogHandler) DeletePost(c *gin.Context) {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	// 获取文章ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.ValidationError(c, "无效的文章ID")
		return
	}

	// 调用服务层删除文章
	err = h.postService.DeletePost(id, *userID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			h.NotFoundError(c, "文章不存在")
		} else if strings.Contains(err.Error(), "权限") {
			h.ForbiddenError(c, "无权限删除该文章")
		} else {
			h.InternalError(c, "删除文章失败: "+err.Error())
		}
		return
	}

	logger.Infof("文章删除成功 (ID: %d, 作者ID: %d)", id, *userID)
	h.SuccessWithMessage(c, "文章删除成功", nil)
}

// PublishPost 发布文章
func (h *BlogHandler) PublishPost(c *gin.Context) {
	userID := h.GetUserID(c)
	if userID == nil {
		h.UnauthorizedError(c, "请先登录")
		return
	}

	// 获取文章ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.ValidationError(c, "无效的文章ID")
		return
	}

	// 调用服务层发布文章
	err = h.postService.PublishPost(id, *userID)
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			h.NotFoundError(c, "文章不存在")
		} else if strings.Contains(err.Error(), "权限") {
			h.ForbiddenError(c, "无权限发布该文章")
		} else {
			h.InternalError(c, "发布文章失败: "+err.Error())
		}
		return
	}

	logger.Infof("文章发布成功 (ID: %d, 作者ID: %d)", id, *userID)
	h.SuccessWithMessage(c, "文章发布成功", nil)
}

// SearchPosts 搜索文章
func (h *BlogHandler) SearchPosts(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		h.ValidationError(c, "搜索关键词不能为空")
		return
	}

	// 获取分页参数
	page, size := h.GetPagination(c)

	// 获取用户信息（可选认证）
	userID := h.GetUserID(c)

	// 调用服务层搜索文章
	posts, total, err := h.postService.SearchPosts(keyword, page, size, userID)
	if err != nil {
		h.InternalError(c, "搜索文章失败: "+err.Error())
		return
	}

	// 构建分页信息
	pagination := map[string]interface{}{
		"page":    page,
		"size":    size,
		"total":   total,
		"pages":   (total + int64(size) - 1) / int64(size),
		"has_next": page*size < int(total),
		"has_prev": page > 1,
	}

	h.SuccessWithData(c, gin.H{
		"keyword":    keyword,
		"posts":      posts,
		"pagination": pagination,
	})
}

// GetCategories 获取分类列表
func (h *BlogHandler) GetCategories(c *gin.Context) {
	// 调用服务层获取分类列表
	categories, err := h.categoryService.GetCategories()
	if err != nil {
		h.InternalError(c, "获取分类列表失败: "+err.Error())
		return
	}

	h.SuccessWithMessage(c, "获取分类列表成功", categories)
}

// CreateCategory 创建分类
func (h *BlogHandler) CreateCategory(c *gin.Context) {
	if !h.RequireAdmin(c) {
		return
	}

	var req CreateCategoryRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层创建分类
	category, err := h.categoryService.CreateCategory(&service.CreateCategoryRequest{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		ParentID:    req.ParentID,
		Color:       req.Color,
	})
	if err != nil {
		if strings.Contains(err.Error(), "已存在") {
			h.Error(c, http.StatusConflict, err.Error())
		} else {
			h.InternalError(c, "创建分类失败: "+err.Error())
		}
		return
	}

	logger.Infof("分类创建成功: %s (ID: %d)", category.Name, category.ID)
	h.SuccessWithMessage(c, "分类创建成功", category)
}

// GetTags 获取标签列表
func (h *BlogHandler) GetTags(c *gin.Context) {
	// 调用服务层获取标签列表
	tags, err := h.tagService.GetTags()
	if err != nil {
		h.InternalError(c, "获取标签列表失败: "+err.Error())
		return
	}

	h.SuccessWithMessage(c, "获取标签列表成功", tags)
}

// CreateTag 创建标签
func (h *BlogHandler) CreateTag(c *gin.Context) {
	if !h.RequireAdmin(c) {
		return
	}

	var req CreateTagRequest
	if err := h.BindJSON(c, &req); err != nil {
		return
	}

	// 验证请求数据
	if err := validator.Validate(&req); err != nil {
		h.ValidationError(c, "请求参数验证失败: "+err.Error())
		return
	}

	// 调用服务层创建标签
	tag, err := h.tagService.CreateTag(&service.CreateTagRequest{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		if strings.Contains(err.Error(), "已存在") {
			h.Error(c, http.StatusConflict, err.Error())
		} else {
			h.InternalError(c, "创建标签失败: "+err.Error())
		}
		return
	}

	logger.Infof("标签创建成功: %s (ID: %d)", tag.Name, tag.ID)
	h.SuccessWithMessage(c, "标签创建成功", tag)
}

// GetPopularTags 获取热门标签
func (h *BlogHandler) GetPopularTags(c *gin.Context) {
	// 获取限制数量
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// 调用服务层获取热门标签
	tags, err := h.tagService.GetPopularTags(limit)
	if err != nil {
		h.InternalError(c, "获取热门标签失败: "+err.Error())
		return
	}

	h.SuccessWithMessage(c, "获取热门标签成功", tags)
}

// GetPostStats 获取文章统计
func (h *BlogHandler) GetPostStats(c *gin.Context) {
	userID := h.GetUserID(c)

	var stats *service.PostStats
	var err error

	if userID != nil {
		stats, err = h.postService.GetUserPostStats(*userID)
	} else {
		stats, err = h.postService.GetPostStats()
	}

	if err != nil {
		h.InternalError(c, "获取文章统计失败: "+err.Error())
		return
	}

	h.SuccessWithMessage(c, "获取文章统计成功", stats)
}