# Go语言代码规范

## 概述
本文档定义了博客管理系统项目的Go语言编码规范，确保代码质量、可读性和可维护性。

## 基础规范

### 1. 格式化
- 使用 `gofmt` 进行代码格式化
- 使用 `goimports` 管理导入语句
- 行长度限制在120字符以内

### 2. 命名规范

#### 包名
- 使用简短、有意义的小写字母
- 避免使用复数形式
- 不要与标准库包名重复

```go
// 好的包名
package blog
package user
package auth

// 不好的包名
package blogs
package users
package authentication
```

#### 变量名
- 使用驼峰命名法 (camelCase)
- 局部变量使用简短名称
- 全局变量使用描述性名称

```go
// 好的变量名
var userCount int
var isConfigured bool
var httpClient *http.Client

// 函数内部可以使用简短变量
func process() {
    u, err := getUser(id)
    if err != nil {
        return err
    }

    n := len(u.Posts)
    // ...
}
```

#### 函数名
- 使用驼峰命名法
- 动词开头，描述性命名
- 公开函数首字母大写，私有函数首字母小写

```go
// 好的函数名
func CreateUser(user *User) error
func validateEmail(email string) bool
func (s *UserService) GetByID(id int) (*User, error)

// 不好的函数名
func user(user *User) error
func check(email string) bool
func GetUser(id int) (*User, error) // 方法名不应该重复类型名
```

#### 常量名
- 使用大写字母和下划线
- 描述性命名

```go
const (
    MAX_FILE_SIZE = 10 * 1024 * 1024
    DEFAULT_PAGE_SIZE = 20
    JWT_EXPIRE_HOURS = 24
)
```

#### 接口名
- 通常以 -er 结尾
- 描述行为能力

```go
type Reader interface {
    Read([]byte) (int, error)
}

type UserRepository interface {
    Create(*User) error
    GetByID(int) (*User, error)
    Update(*User) error
    Delete(int) error
}
```

### 3. 注释规范

#### 包注释
每个包都应该有包注释，说明包的用途：

```go
// Package blog provides blog post management functionality
// including creating, updating, deleting and publishing blog posts.
package blog
```

#### 公开函数注释
所有公开的函数、方法、类型、变量都必须有注释：

```go
// CreateUser creates a new user with the provided user data.
// It returns the created user ID or an error if the operation fails.
func CreateUser(user *User) (int, error) {
    // implementation
}

// User represents a user in the system.
type User struct {
    ID       int    `json:"id" db:"id"`
    Username string `json:"username" db:"username"`
    Email    string `json:"email" db:"email"`
}
```

#### 复杂逻辑注释
对于复杂的业务逻辑，添加解释性注释：

```go
func (s *PostService) PublishPost(id int) error {
    // Reason: We need to validate the post before publishing
    // to ensure it meets all publication requirements.
    if err := s.validatePostForPublishing(id); err != nil {
        return err
    }

    // Reason: Update both status and published_at atomically
    // to maintain data consistency.
    now := time.Now()
    return s.repo.UpdateStatus(id, "published", now)
}
```

## 代码组织

### 1. 文件组织
- 每个文件应该有单一职责
- 文件长度不要超过500行
- 相关功能放在同一个包中

```go
// 文件组织示例
internal/
├── blog/
│   ├── model.go       // 数据模型
│   ├── repository.go  // 数据访问层
│   ├── service.go     // 业务逻辑层
│   └── handler.go     // HTTP处理器
├── user/
│   ├── model.go
│   ├── repository.go
│   ├── service.go
│   └── handler.go
```

### 2. 导入语句
- 按标准库、第三方库、本地包分组
- 每组按字母顺序排列
- 删除未使用的导入

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    "github.com/your-org/blog-system/internal/core/database"
    "github.com/your-org/blog-system/pkg/response"
)
```

### 3. 结构体定义
- 字段按照访问权限分组（公开字段在前）
- 添加结构体标签 (json, db, validate等)

```go
type User struct {
    // 公开字段
    ID       int       `json:"id" db:"id" gorm:"primaryKey"`
    Username string    `json:"username" db:"username" gorm:"unique;not null"`
    Email    string    `json:"email" db:"email" gorm:"unique;not null"`
    Role     string    `json:"role" db:"role" gorm:"default:user"`

    // 私有字段
    passwordHash string `json:"-" db:"password_hash" gorm:"not null"`
    createdAt    time.Time `json:"created_at" db:"created_at"`
    updatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
```

## 错误处理

### 1. 错误定义
- 使用 `errors.New()` 创建简单错误
- 使用 `fmt.Errorf()` 创建带格式的错误
- 定义自定义错误类型

```go
var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidEmail = errors.New("invalid email format")
    ErrPermissionDenied = errors.New("permission denied")
)

// 自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}
```

### 2. 错误处理原则
- 立即处理错误，不要忽略
- 使用适当的日志级别记录错误
- 向用户返回友好的错误信息

```go
func (s *UserService) CreateUser(user *User) error {
    if err := validateUser(user); err != nil {
        s.logger.Warn("user validation failed", "error", err)
        return fmt.Errorf("validation failed: %w", err)
    }

    if err := s.repo.Create(user); err != nil {
        s.logger.Error("failed to create user", "error", err)
        return fmt.Errorf("database error: %w", err)
    }

    s.logger.Info("user created successfully", "user_id", user.ID)
    return nil
}
```

### 3. 错误包装
- 使用 `%w` 动词包装错误，保留原始错误信息
- 使用 `%v` 动词添加上下文信息

```go
func (s *PostService) GetPost(id int) (*Post, error) {
    post, err := s.repo.GetByID(id)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, fmt.Errorf("post with id %d not found: %w", id, ErrPostNotFound)
        }
        return nil, fmt.Errorf("failed to get post %d: %w", id, err)
    }
    return post, nil
}
```

## 并发编程

### 1. Goroutine使用
- 不要创建无限制的goroutine
- 使用 `sync.WaitGroup` 等待goroutine完成
- 使用 channel 进行goroutine间通信

```go
func (s *PostService) ProcessPosts(posts []Post) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(posts))
    semaphore := make(chan struct{}, 10) // 限制并发数

    for _, post := range posts {
        wg.Add(1)
        go func(p Post) {
            defer wg.Done()

            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            if err := s.processPost(p); err != nil {
                errChan <- err
            }
        }(post)
    }

    wg.Wait()
    close(errChan)

    for err := range errChan {
        if err != nil {
            return err
        }
    }
    return nil
}
```

### 2. 互斥锁使用
- 使用 `sync.Mutex` 保护共享资源
- 避免在锁内进行耗时操作
- 使用 `defer` 确保锁被释放

```go
type Counter struct {
    mu    sync.RWMutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

func (c *Counter) Value() int {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.count
}
```

## 测试规范

### 1. 测试文件命名
- 测试文件以 `_test.go` 结尾
- 与被测试文件放在同一个包中

### 2. 测试函数命名
- 测试函数以 `Test` 开头
- 使用描述性名称说明测试场景

```go
func TestUserService_CreateUser_Success(t *testing.T) {
    // test implementation
}

func TestUserService_CreateUser_InvalidEmail(t *testing.T) {
    // test implementation
}
```

### 3. 测试结构
- 使用子测试组织相关测试
- 使用表驱动测试进行多场景测试

```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name    string
        user    *User
        wantErr bool
        errType error
    }{
        {
            name: "valid user",
            user: &User{
                Username: "testuser",
                Email:    "test@example.com",
            },
            wantErr: false,
        },
        {
            name: "invalid email",
            user: &User{
                Username: "testuser",
                Email:    "invalid-email",
            },
            wantErr: true,
            errType: ErrInvalidEmail,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateUser(tt.user)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateUser() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if tt.wantErr && !errors.Is(err, tt.errType) {
                t.Errorf("validateUser() error = %v, want %v", err, tt.errType)
            }
        })
    }
}
```

## 性能优化

### 1. 内存管理
- 避免内存泄漏
- 使用对象池复用对象
- 注意字符串拼接的内存消耗

```go
// 好的做法：使用strings.Builder
var builder strings.Builder
for _, item := range items {
    builder.WriteString(item.Name)
    builder.WriteString(",")
}
result := builder.String()

// 不好的做法：使用+拼接字符串
var result string
for _, item := range items {
    result += item.Name + ","
}
```

### 2. 数据库操作
- 使用预编译语句
- 批量操作代替单条操作
- 适当使用索引

```go
// 好的做法：批量插入
func (r *PostRepository) CreatePosts(posts []Post) error {
    return r.db.CreateInBatches(posts, 100).Error
}

// 不好的做法：循环单条插入
func (r *PostRepository) CreatePosts(posts []Post) error {
    for _, post := range posts {
        if err := r.db.Create(&post).Error; err != nil {
            return err
        }
    }
    return nil
}
```

## 安全规范

### 1. 密码处理
- 使用强哈希算法 (bcrypt)
- 不要在日志中记录密码
- 使用安全的随机数生成器

```go
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### 2. SQL注入防护
- 使用参数化查询
- 验证用户输入
- 使用ORM的安全特性

```go
// 好的做法：使用参数化查询
func (r *UserRepository) GetByEmail(email string) (*User, error) {
    var user User
    err := r.db.Where("email = ?", email).First(&user).Error
    return &user, err
}

// 不好的做法：字符串拼接
func (r *UserRepository) GetByEmail(email string) (*User, error) {
    var user User
    query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email) // 危险！
    err := r.db.Raw(query).Scan(&user).Error
    return &user, err
}
```

## 代码审查检查清单

### 1. 代码质量
- [ ] 代码是否符合格式化规范
- [ ] 变量和函数命名是否清晰
- [ ] 是否有适当的注释
- [ ] 是否有重复代码

### 2. 错误处理
- [ ] 所有错误都被适当处理
- [ ] 错误信息对用户友好
- [ ] 关键错误被记录日志

### 3. 性能考虑
- [ ] 是否有内存泄漏风险
- [ ] 数据库查询是否优化
- [ ] 是否有不必要的计算

### 4. 安全性
- [ ] 输入是否经过验证
- [ ] 敏感信息是否安全处理
- [ ] 是否有SQL注入风险

### 5. 测试
- [ ] 是否有足够的单元测试
- [ ] 测试是否覆盖边界情况
- [ ] 测试是否易于理解和维护

---

**最后更新**: 2024-11-02
**维护者**: 开发团队