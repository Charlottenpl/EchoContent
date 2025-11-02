# 开发文档

## 概述
本文档集合为博客管理系统的开发者提供完整的开发指南，包括环境搭建、API文档、代码规范等内容。

## 目标读者
- 后端开发工程师
- 前端开发工程师
- DevOps工程师
- 项目技术负责人

## 文档结构

### 📚 核心文档
- [开发环境搭建](setup.md) - 本地开发环境配置指南
- [API文档](api.md) - RESTful API接口文档
- [数据库设计](database.md) - 数据库结构和设计说明
- [代码规范](coding-standards.md) - Go语言编码规范和最佳实践

### 🔧 开发工具
- [测试指南](testing.md) - 单元测试和集成测试指南
- [贡献指南](contributing.md) - 项目贡献流程和规范

### 📖 相关文档
- [系统架构总览](../architecture/overview.md)
- [部署指南](../deployment/README.md)
- [用户手册](../user/README.md)

## 快速开始

### 1. 环境要求
- Go 1.21+
- SQLite 3.40+
- Git 2.0+
- Docker (可选)

### 2. 快速启动
```bash
# 克隆项目
git clone https://github.com/your-org/blog-system.git
cd blog-system

# 安装依赖
go mod download

# 启动开发服务器
make dev
```

### 3. 访问应用
- API文档: http://localhost:8080/swagger/index.html
- 管理后台: http://localhost:8080/admin
- 前端应用: http://localhost:3000

## 开发流程

### 分支管理
- `main` - 生产环境分支
- `develop` - 开发环境分支
- `feature/*` - 功能开发分支
- `hotfix/*` - 紧急修复分支

### 提交规范
```
type(scope): description

[optional body]

[optional footer]
```

类型说明：
- `feat`: 新功能
- `fix`: 修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

### 代码审查
所有代码变更都需要通过Pull Request进行代码审查，确保代码质量。

## 开发工具配置

### IDE配置
推荐使用VS Code或GoLand，并安装以下插件：
- Go语言支持
- Docker插件
- GitLens
- Prettier/GoFormat

### 调试配置
项目包含预设的调试配置，支持断点调试和性能分析。

## 常见问题

### Q: 如何添加新的API端点？
A: 请参考[API文档](api.md)和[代码规范](coding-standards.md)了解接口设计规范。

### Q: 如何运行测试？
A: 请参考[测试指南](testing.md)了解测试流程和编写规范。

### Q: 如何处理数据库迁移？
A: 项目使用GORM的AutoMigrate功能，具体请参考[数据库设计文档](database.md)。

## 获取帮助

如果在开发过程中遇到问题，可以通过以下方式获取帮助：
- 查阅本文档集合
- 查看项目Issues
- 联系项目维护者

---

**最后更新**: 2024-11-02
**维护者**: 开发团队