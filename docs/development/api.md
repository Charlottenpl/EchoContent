# API文档

## 概述
本文档描述了博客管理系统的RESTful API接口规范。

## 基础信息

### 服务器地址
- **开发环境**: `http://localhost:8080`
- **生产环境**: `https://your-domain.com`

### API版本
- **当前版本**: v1
- **基础路径**: `/api/v1`

### 认证方式
- **JWT Token**: 在请求头中包含 `Authorization: Bearer <token>`

### 请求格式
- **Content-Type**: `application/json`
- **字符编码**: UTF-8

### 响应格式
所有API响应都遵循统一的格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": "2024-11-02T10:00:00Z"
}
```

## 状态码说明

| 状态码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 认证失败 |
| 1003 | 权限不足 |
| 1004 | 资源不存在 |
| 1005 | 服务器内部错误 |
| 1006 | 数据库操作失败 |
| 1007 | 文件操作失败 |

## 认证接口

### 用户注册
```http
POST /api/v1/auth/register
```

**请求体**:
```json
{
  "email": "user@example.com",
  "password": "password123",
  "verification_code": "123456"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "注册成功",
  "data": {
    "user": {
      "id": 1,
      "email": "user@example.com",
      "username": "user123",
      "role": "user",
      "created_at": "2024-11-02T10:00:00Z"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

### 用户登录
```http
POST /api/v1/auth/login
```

**请求体**:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应**:
```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "user": {
      "id": 1,
      "email": "user@example.com",
      "username": "user123",
      "role": "user"
    },
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-11-03T10:00:00Z"
  }
}
```

### 发送验证码
```http
POST /api/v1/auth/send-verification-code
```

**请求体**:
```json
{
  "email": "user@example.com"
}
```

### 刷新Token
```http
POST /api/v1/auth/refresh
```

**请求头**:
```http
Authorization: Bearer <refresh_token>
```

## 用户接口

### 获取用户信息
```http
GET /api/v1/users/profile
```

**请求头**:
```http
Authorization: Bearer <token>
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "username": "user123",
    "email": "user@example.com",
    "nickname": "用户昵称",
    "avatar": "https://example.com/avatar.jpg",
    "role": "user",
    "created_at": "2024-11-02T10:00:00Z"
  }
}
```

### 更新用户信息
```http
PUT /api/v1/users/profile
```

**请求体**:
```json
{
  "nickname": "新昵称",
  "avatar": "https://example.com/new-avatar.jpg"
}
```

## 博客文章接口

### 获取文章列表
```http
GET /api/v1/posts?page=1&size=10&category=tech&tag=go&keyword=搜索关键词
```

**查询参数**:
- `page`: 页码 (默认: 1)
- `size`: 每页数量 (默认: 10)
- `category`: 分类筛选
- `tag`: 标签筛选
- `keyword`: 关键词搜索

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "posts": [
      {
        "id": 1,
        "title": "Go语言开发指南",
        "slug": "go-development-guide",
        "excerpt": "这是一篇关于Go语言开发的文章...",
        "content": "完整的文章内容...",
        "type": "blog",
        "status": "published",
        "author": {
          "id": 1,
          "username": "admin",
          "nickname": "管理员"
        },
        "category": {
          "id": 1,
          "name": "技术",
          "slug": "tech"
        },
        "tags": [
          {
            "id": 1,
            "name": "Go",
            "slug": "go"
          }
        ],
        "featured_image": "https://example.com/image.jpg",
        "view_count": 100,
        "like_count": 10,
        "comment_count": 5,
        "published_at": "2024-11-01T10:00:00Z",
        "created_at": "2024-11-01T09:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "size": 10,
      "total": 50,
      "pages": 5
    }
  }
}
```

### 获取文章详情
```http
GET /api/v1/posts/{id}
```

**路径参数**:
- `id`: 文章ID

### 创建文章
```http
POST /api/v1/posts
```

**请求体**:
```json
{
  "title": "文章标题",
  "content": "文章内容",
  "excerpt": "文章摘要",
  "type": "blog",
  "status": "draft",
  "category_id": 1,
  "tag_ids": [1, 2, 3],
  "featured_image": "https://example.com/image.jpg"
}
```

### 更新文章
```http
PUT /api/v1/posts/{id}
```

### 删除文章
```http
DELETE /api/v1/posts/{id}
```

### 发布文章
```http
POST /api/v1/posts/{id}/publish
```

## 随念接口

### 获取随念列表
```http
GET /api/v1/moments?page=1&size=20
```

### 创建随念
```http
POST /api/v1/moments
```

**请求体**:
```json
{
  "content": "今天天气不错！",
  "status": "published",
  "media_ids": [1, 2]
}
```

## 评论接口

### 获取文章评论
```http
GET /api/v1/posts/{post_id}/comments?page=1&size=10
```

### 创建评论
```http
POST /api/v1/posts/{post_id}/comments
```

**请求体**:
```json
{
  "content": "评论内容",
  "parent_id": null
}
```

### 删除评论
```http
DELETE /api/v1/comments/{id}
```

## 分类和标签接口

### 获取分类列表
```http
GET /api/v1/categories
```

### 创建分类
```http
POST /api/v1/categories
```

**请求体**:
```json
{
  "name": "技术",
  "slug": "tech",
  "description": "技术相关文章"
}
```

### 获取标签列表
```http
GET /api/v1/tags
```

### 创建标签
```http
POST /api/v1/tags
```

## 媒体文件接口

### 上传文件
```http
POST /api/v1/media/upload
```

**请求**: multipart/form-data
- `file`: 文件内容

**响应**:
```json
{
  "code": 0,
  "message": "上传成功",
  "data": {
    "id": 1,
    "filename": "image.jpg",
    "original_name": "my-image.jpg",
    "mime_type": "image/jpeg",
    "size": 1024000,
    "url": "https://example.com/uploads/image.jpg",
    "created_at": "2024-11-02T10:00:00Z"
  }
}
```

### 获取媒体文件列表
```http
GET /api/v1/media?page=1&size=20
```

### 删除媒体文件
```http
DELETE /api/v1/media/{id}
```

## 互动接口

### 点赞文章/随念
```http
POST /api/v1/likes
```

**请求体**:
```json
{
  "target_type": "post",
  "target_id": 1
}
```

### 取消点赞
```http
DELETE /api/v1/likes
```

**查询参数**:
- `target_type`: 目标类型 (post/comment)
- `target_id`: 目标ID

### 收藏文章
```http
POST /api/v1/favorites
```

**请求体**:
```json
{
  "post_id": 1
}
```

### 取消收藏
```http
DELETE /api/v1/favorites/{post_id}
```

### 获取收藏列表
```http
GET /api/v1/favorites?page=1&size=10
```

## 管理后台接口

### 获取统计数据
```http
GET /api/v1/admin/stats
```

**响应**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_users": 1000,
    "total_posts": 500,
    "total_moments": 200,
    "total_comments": 1000,
    "today_views": 100,
    "week_views": 500
  }
}
```

### 用户管理
```http
GET /api/v1/admin/users?page=1&size=20&status=active
PUT /api/v1/admin/users/{id}/status
DELETE /api/v1/admin/users/{id}
```

### 内容管理
```http
GET /api/v1/admin/posts?page=1&size=20&status=all
PUT /api/v1/admin/posts/{id}/status
DELETE /api/v1/admin/posts/{id}
```

### 评论管理
```http
GET /api/v1/admin/comments?page=1&size=20&status=pending
PUT /api/v1/admin/comments/{id}/approve
PUT /api/v1/admin/comments/{id}/reject
DELETE /api/v1/admin/comments/{id}
```

## GitHub同步接口

### 手动同步
```http
POST /api/v1/admin/sync/github
```

### 获取同步状态
```http
GET /api/v1/admin/sync/github/status
```

### 配置同步设置
```http
PUT /api/v1/admin/sync/github/config
```

**请求体**:
```json
{
  "enabled": true,
  "token": "github_token",
  "repo": "username/repo",
  "branch": "main",
  "auto_sync": true,
  "sync_interval": "1h"
}
```

## 系统接口

### 健康检查
```http
GET /health
```

**响应**:
```json
{
  "status": "ok",
  "timestamp": "2024-11-02T10:00:00Z",
  "version": "1.0.0",
  "uptime": "24h30m15s"
}
```

### 系统信息
```http
GET /api/v1/system/info
```

### 系统配置
```http
GET /api/v1/system/config
PUT /api/v1/system/config
```

## 错误处理

### 标准错误响应
```json
{
  "code": 1001,
  "message": "参数错误",
  "error": "email格式不正确",
  "timestamp": "2024-11-02T10:00:00Z"
}
```

### 常见错误码
- 1001: 参数验证失败
- 1002: 认证失败或token过期
- 1003: 权限不足
- 1004: 资源不存在
- 1005: 服务器内部错误
- 1006: 数据库操作失败
- 1007: 文件上传失败

## 接口限流

### 限流规则
- 普通接口: 100次/分钟
- 认证接口: 10次/分钟
- 上传接口: 20次/分钟

### 限流响应
```json
{
  "code": 429,
  "message": "请求过于频繁",
  "error": "Rate limit exceeded",
  "retry_after": 60
}
```

## 接口版本管理

### 版本策略
- 使用语义化版本号
- 向后兼容的更新不增加主版本号
- 破坏性变更需要增加主版本号

### 版本指定
```http
# 在URL中指定版本
GET /api/v1/posts

# 在请求头中指定版本
Accept: application/vnd.api+json;version=1
```

---

**最后更新**: 2024-11-02
**维护者**: 开发团队