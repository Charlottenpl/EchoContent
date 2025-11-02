# 博客管理系统

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://github.com/your-org/blog-system/workflows/CI/badge.svg)](https://github.com/your-org/blog-system/actions)
[![Coverage](https://codecov.io/gh/your-org/blog-system/branch/main/graph/badge.svg)](https://codecov.io/gh/your-org/blog-system)

一个使用Go语言开发的轻量级博客管理系统，支持博客文章、随念（类似微博）、图床等功能，参考HALO系统设计理念，专为个人博客和小团队设计。

## ✨ 特性

### 🚀 核心功能
- **博客管理**: 文章的创建、编辑、发布、分类和标签管理
- **随念系统**: 类似微博的短内容发布，支持300字内容和多媒体
- **用户系统**: 支持多种登录方式扩展（当前实现邮箱验证码登录）
- **评论系统**: 支持嵌套评论的管理和审核
- **图床服务**: 图片和视频文件的上传与管理
- **搜索功能**: 全文搜索博客内容和随念

### 🔧 管理功能
- **后台管理**: 完整的管理后台界面
- **用户管理**: 用户注册、权限管理
- **内容管理**: 文章、随念、评论的统一管理
- **统计分析**: 访问量、用户活跃度等数据统计
- **系统日志**: 完整的操作日志记录和分析

### 🌐 扩展功能
- **GitHub同步**: 文章内容自动同步到GitHub仓库
- **API接口**: 完整的RESTful API支持
- **主题系统**: 支持主题切换和自定义
- **插件架构**: 预留插件化扩展能力

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   前端界面      │    │   管理后台      │    │   API客户端     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────┐
         │              Nginx 反向代理                     │
         └─────────────────────────────────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────┐
         │            Go 应用服务器                         │
         │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌───────┐ │
         │  │ 用户模块│ │博客模块 │ │随念模块 │ │图床   │ │
         │  └─────────┘ └─────────┘ └─────────┘ └───────┘ │
         └─────────────────────────────────────────────────┘
                                 │
         ┌─────────────────────────────────────────────────┐
         │     SQLite 数据库    │     本地文件存储          │
         └─────────────────────────────────────────────────┘
```

## 🚀 快速开始

### 环境要求
- Go 1.21+
- SQLite 3.40+
- Git 2.0+
- Docker (可选)

### 本地开发

```bash
# 1. 克隆项目
git clone https://github.com/your-org/blog-system.git
cd blog-system

# 2. 安装依赖
go mod download

# 3. 复制配置文件
cp configs/config.example.yaml configs/config.yaml

# 4. 初始化数据库
make migrate

# 5. 启动开发服务器
make dev
```

### Docker 部署

```bash
# 使用 Docker Compose 启动
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 访问应用
- **网站首页**: http://localhost:8080
- **管理后台**: http://localhost:8080/admin
- **API文档**: http://localhost:8080/swagger/index.html
- **健康检查**: http://localhost:8080/health

## 📚 文档

### 开发文档
- [开发指南](docs/development/README.md) - 开发环境搭建和规范
- [API文档](docs/development/api.md) - RESTful API接口说明
- [数据库设计](docs/development/database.md) - 数据库结构和设计
- [代码规范](docs/development/coding-standards.md) - Go语言编码规范

### 部署文档
- [部署指南](docs/deployment/README.md) - 生产环境部署指南
- [Docker部署](docs/deployment/docker.md) - 容器化部署方案
- [CI/CD配置](docs/deployment/github-actions.md) - 持续集成配置

### 用户文档
- [用户手册](docs/user/README.md) - 系统使用说明
- [管理员指南](docs/user/admin-guide.md) - 管理后台操作指南
- [常见问题](docs/user/faq.md) - 常见问题解答

## 🛠️ 技术栈

- **后端语言**: Go 1.21+
- **Web框架**: Gin
- **数据库**: SQLite
- **ORM**: GORM
- **认证**: JWT + OAuth 2.0
- **日志**: Logrus
- **配置**: Viper
- **部署**: Docker + GitHub Actions

## 📊 系统要求

### 最低配置
- **CPU**: 1核心
- **内存**: 1GB
- **存储**: 10GB
- **网络**: 1Mbps

### 推荐配置
- **CPU**: 2核心
- **内存**: 2GB
- **存储**: 40GB SSD
- **网络**: 5Mbps

### 支持规模
- **并发用户**: 100人
- **文章数量**: 1000篇+
- **用户数量**: 10000人+
- **日均访问**: 1000次+

## 🔧 配置说明

### 主要配置项

```yaml
app:
  name: "Blog System"
  port: 8080
  mode: "debug"

database:
  type: "sqlite"
  dsn: "./data/blog.db"

jwt:
  secret: "your-secret-key"
  expire_hours: 24

upload:
  path: "./uploads"
  max_size: 10485760  # 10MB
```

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `GIN_MODE` | 运行模式 | `debug` |
| `PORT` | 服务端口 | `8080` |
| `DB_PATH` | 数据库路径 | `./data/blog.db` |
| `JWT_SECRET` | JWT密钥 | - |

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 贡献方式
1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 开发规范
- 遵循 [Go代码规范](docs/development/coding-standards.md)
- 编写单元测试
- 更新相关文档
- 通过所有CI检查

## 📝 更新日志

### v1.0.0 (2024-11-02)
- ✨ 初始版本发布
- 🚀 基础博客功能
- 👤 用户系统
- 📝 随念功能
- 🖼️ 图床服务
- 🔧 管理后台

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [Gin](https://github.com/gin-gonic/gin) - 优秀的Go Web框架
- [GORM](https://github.com/go-gorm/gorm) - 友好的Go ORM库
- [HALO](https://halo.run/) - 设计灵感来源

## 📞 联系我们

- **项目主页**: https://github.com/your-org/blog-system
- **问题反馈**: https://github.com/your-org/blog-system/issues
- **邮箱**: dev@example.com

---

⭐ 如果这个项目对你有帮助，请给我们一个星标！