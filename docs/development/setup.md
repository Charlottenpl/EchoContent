# 开发环境搭建

## 概述
本文档指导开发者搭建博客管理系统的本地开发环境。

## 前置条件

### 系统要求
- **操作系统**: Windows 10+, macOS 10.15+, Ubuntu 18.04+
- **内存**: 最少4GB，推荐8GB
- **磁盘空间**: 至少2GB可用空间

### 必需软件
- **Go**: 1.21或更高版本
- **Git**: 2.0或更高版本
- **SQLite**: 3.40或更高版本

### 可选软件
- **Docker**: 20.10+ (用于容器化开发)
- **Docker Compose**: 2.0+ (用于多容器编排)
- **VS Code**: 推荐的开发IDE

## 安装步骤

### 1. 安装Go语言

#### Windows
1. 访问 [Go官网](https://golang.org/dl/)
2. 下载适用于Windows的安装包
3. 运行安装包，按默认设置安装
4. 验证安装：
```cmd
go version
```

#### macOS
```bash
# 使用Homebrew安装
brew install go

# 验证安装
go version
```

#### Linux (Ubuntu/Debian)
```bash
# 下载Go二进制包
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz

# 解压到/usr/local
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# 添加到PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

### 2. 安装Git

#### Windows
1. 访问 [Git官网](https://git-scm.com/download/win)
2. 下载并安装Git for Windows
3. 使用Git Bash进行以下验证：
```bash
git --version
```

#### macOS
```bash
# 使用Xcode Command Line Tools
xcode-select --install

# 或使用Homebrew
brew install git

# 验证安装
git --version
```

#### Linux
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install git

# 验证安装
git --version
```

### 3. 安装SQLite

#### Windows
1. 访问 [SQLite官网](https://sqlite.org/download.html)
2. 下载SQLite工具包
3. 解压并将sqlite3.exe添加到PATH

#### macOS
```bash
# 使用Homebrew
brew install sqlite

# 验证安装
sqlite3 --version
```

#### Linux
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install sqlite3 libsqlite3-dev

# 验证安装
sqlite3 --version
```

### 4. 克隆项目

```bash
# 克隆项目仓库
git clone https://github.com/your-org/blog-system.git
cd blog-system

# 查看项目结构
ls -la
```

### 5. 安装项目依赖

```bash
# 下载Go模块依赖
go mod download

# 验证依赖
go mod verify

# 整理依赖
go mod tidy
```

### 6. 配置开发环境

#### 6.1 复制配置文件
```bash
# 复制配置模板
cp configs/config.example.yaml configs/config.yaml
```

#### 6.2 编辑配置文件
```yaml
# configs/config.yaml
app:
  name: "Blog System"
  version: "1.0.0"
  port: 8080
  mode: "debug"  # debug, release, test

database:
  type: "sqlite"
  dsn: "./data/blog.db"
  max_idle_conns: 10
  max_open_conns: 100

jwt:
  secret: "your-secret-key-here"
  expire_hours: 24

log:
  level: "debug"
  format: "json"
  output: "stdout"

upload:
  path: "./uploads"
  max_size: 10485760  # 10MB
  allowed_types: ["jpg", "jpeg", "png", "gif", "mp4", "avi"]

github:
  sync_enabled: false
  token: ""
  repo: ""
  branch: "main"
```

### 7. 初始化数据库

```bash
# 创建数据目录
mkdir -p data uploads

# 运行数据库迁移
go run cmd/server/main.go migrate

# 或使用Make命令
make migrate
```

### 8. 启动开发服务器

```bash
# 直接运行
go run cmd/server/main.go

# 或使用Make命令
make dev

# 或使用air实现热重载
air
```

## 开发工具配置

### 1. VS Code配置

安装推荐的VS Code扩展：
- Go (官方Go语言支持)
- SQLite Viewer
- Docker
- GitLens
- Prettier

创建 `.vscode/settings.json`:
```json
{
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v"],
  "go.coverOnSave": true,
  "go.coverageDecorator": {
    "type": "gutter",
    "coveredHighlightColor": "rgba(64,128,64,0.5)",
    "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
  }
}
```

创建 `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/server",
      "env": {
        "GIN_MODE": "debug"
      },
      "args": []
    }
  ]
}
```

### 2. Go开发工具安装

```bash
# 安装golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 安装air (热重载工具)
go install github.com/cosmtrek/air@latest

# 安装swag (API文档生成)
go install github.com/swaggo/swag/cmd/swag@latest

# 安装go mock
go install github.com/golang/mock/mockgen@latest
```

### 3. Makefile配置

项目包含一个便捷的Makefile：

```bash
# 查看所有可用命令
make help

# 常用命令
make dev          # 启动开发服务器
make test         # 运行测试
make lint         # 代码检查
make build        # 构建应用
make clean        # 清理构建文件
make migrate      # 数据库迁移
make docs         # 生成API文档
```

## Docker开发环境

### 1. 使用Docker Compose

```bash
# 启动开发环境
docker-compose -f docker-compose.dev.yml up -d

# 查看日志
docker-compose -f docker-compose.dev.yml logs -f

# 停止环境
docker-compose -f docker-compose.dev.yml down
```

### 2. Docker开发配置

创建 `docker-compose.dev.yml`:
```yaml
version: '3.8'

services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.dev
    ports:
      - "8080:8080"
    volumes:
      - .:/app
      - /app/data
      - /app/uploads
    environment:
      - GIN_MODE=debug
    command: air -c .air.toml

  db:
    image: sqlite:latest
    volumes:
      - ./data:/data
```

## 验证安装

### 1. 检查服务状态
访问以下URL验证服务是否正常运行：
- API健康检查: http://localhost:8080/health
- API文档: http://localhost:8080/swagger/index.html

### 2. 运行测试
```bash
# 运行所有测试
make test

# 运行特定测试
go test ./internal/blog/...

# 运行基准测试
go test -bench=. ./...
```

### 3. 检查代码质量
```bash
# 代码格式检查
make fmt

# 代码静态分析
make lint

# 安全检查
gosec ./...
```

## 常见问题

### Q: Go模块下载失败怎么办？
A: 可以尝试以下解决方案：
```bash
# 设置Go代理
go env -w GOPROXY=https://goproxy.cn,direct

# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod download
```

### Q: 数据库连接失败怎么办？
A: 检查以下几点：
1. SQLite是否正确安装
2. 配置文件中的数据库路径是否正确
3. 数据目录是否有写入权限

### Q: 端口被占用怎么办？
A: 修改配置文件中的端口号，或停止占用端口的进程：
```bash
# 查看端口占用
lsof -i :8080

# 修改配置文件中的端口
vim configs/config.yaml
```

## 下一步

环境搭建完成后，建议继续阅读：
- [API文档](api.md) - 了解接口设计
- [代码规范](coding-standards.md) - 了解编码标准
- [测试指南](testing.md) - 了解测试方法

---

**最后更新**: 2024-11-02
**维护者**: 开发团队