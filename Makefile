# Makefile for Blog System

# 变量定义
APP_NAME = blog-system
VERSION = 1.0.0
BUILD_DIR = build
BIN_DIR = bin
CONFIG_DIR = config
MAIN_FILE = cmd/server/main.go

# Go相关变量
GO = go
GOFMT = gofmt
GOLINT = golint
GOVET = go vet
GOTEST = go test
GOMOD = go mod

# 构建标志
LDFLAGS = -ldflags "-X main.AppVersion=$(VERSION) -X main.BuildTime=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ') -X main.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# 默认目标
.PHONY: all
all: clean deps lint test build

# 帮助信息
.PHONY: help
help:
	@echo "可用命令:"
	@echo "  make deps        - 安装依赖"
	@echo "  make build       - 构建应用"
	@echo "  make run         - 运行应用"
	@echo "  make test        - 运行测试"
	@echo "  make test-cover  - 运行测试并生成覆盖率报告"
	@echo "  make lint        - 代码检查"
	@echo "  make fmt         - 格式化代码"
	@echo "  make vet         - 静态分析"
	@echo "  make clean       - 清理构建文件"
	@echo "  make migrate     - 运行数据库迁移"
	@echo "  make dev         - 开发模式运行"
	@echo "  make docker-build - 构建Docker镜像"
	@echo "  make docker-run   - 运行Docker容器"
	@echo "  make release     - 构建发布版本"

# 安装依赖
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# 代码格式化
.PHONY: fmt
fmt:
	$(GOFMT) -s -w .

# 静态分析
.PHONY: vet
vet:
	$(GOVET) ./...

# 代码检查
.PHONY: lint
lint:
	@which $(GOLINT) > /dev/null || (echo "安装 golint: go get golang.org/x/lint/golint" && exit 1)
	$(GOLINT) ./...

# 运行测试
.PHONY: test
test:
	$(GOTEST) -v ./...

# 运行测试并生成覆盖率报告
.PHONY: test-cover
test-cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 构建应用
.PHONY: build
build: clean
	@echo "构建应用..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

# 构建所有平台的二进制文件
.PHONY: build-all
build-all: clean
	@echo "构建所有平台的二进制文件..."
	@mkdir -p $(BUILD_DIR)

	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_FILE)

	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(MAIN_FILE)

	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_FILE)

	# MacOS AMD64
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN_FILE)

	# MacOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_FILE)

# 构建发布版本
.PHONY: release
release: clean
	@echo "构建发布版本..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -tags=release -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

# 运行应用
.PHONY: run
run: deps
	$(GO) run $(MAIN_FILE) -config $(CONFIG_DIR)/config.yaml

# 开发模式运行
.PHONY: dev
dev: deps
	$(GO) run $(MAIN_FILE) -config $(CONFIG_DIR)/config.yaml

# 生产模式运行
.PHONY: prod
prod: deps
	$(GO) run $(MAIN_FILE) -config $(CONFIG_DIR)/config.yaml

# 创建必要的目录
.PHONY: setup
setup:
	@echo "创建必要的目录..."
	@mkdir -p data logs uploads backups
	@mkdir -p logs/email
	@mkdir -p uploads/images uploads/documents uploads/videos
	@echo "目录创建完成"

# 运行数据库迁移
.PHONY: migrate
migrate: deps
	$(GO) run $(MAIN_FILE) -migrate

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# 清理所有生成的文件
.PHONY: clean-all
clean-all: clean
	@echo "清理所有生成的文件..."
	@rm -rf data/*.db
	@rm -rf logs/*
	@rm -rf uploads/*
	@rm -rf backups/*

# Docker相关命令
.PHONY: docker-build
docker-build:
	docker build -t $(APP_NAME):$(VERSION) .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 --name $(APP_NAME) $(APP_NAME):$(VERSION)

.PHONY: docker-stop
docker-stop:
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

# 生成API文档
.PHONY: docs
docs:
	@echo "生成API文档..."
	@which swag > /dev/null || (echo "安装 swag: go get -u github.com/swaggo/swag/cmd/swag" && exit 1)
	swag init -g cmd/server/main.go -o docs

# 安装开发工具
.PHONY: install-tools
install-tools:
	@echo "安装开发工具..."
	$(GO) install golang.org/x/lint/golint@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 检查代码质量
.PHONY: check
check: fmt vet lint test
	@echo "代码检查完成"

# 快速提交（开发用）
.PHONY: quick-commit
quick-commit:
	@git add .
	@git commit -m "quick commit: $(shell date '+%Y-%m-%d %H:%M:%S')"

# 创建git标签
.PHONY: tag
tag:
	@if [ -n "$(VERSION)" ]; then \
		git tag -a v$(VERSION) -m "Release version $(VERSION)"; \
		git push origin v$(VERSION); \
	else \
		echo "请指定版本号: make tag VERSION=1.0.0"; \
	fi

# 显示项目信息
.PHONY: info
info:
	@echo "项目信息:"
	@echo "  名称: $(APP_NAME)"
	@echo "  版本: $(VERSION)"
	@echo "  Go版本: $(shell go version)"
	@echo "  Git提交: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "  构建时间: $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')"

# 启动依赖服务（如数据库）
.PHONY: start-services
start-services:
	@echo "启动依赖服务..."
	@# 这里可以添加启动数据库等服务的命令

# 停止依赖服务
.PHONY: stop-services
stop-services:
	@echo "停止依赖服务..."
	@# 这里可以添加停止数据库等服务的命令

# 监控文件变化并自动重启（开发用）
.PHONY: watch
watch:
	@which reflex > /dev/null || (echo "安装 reflex: go get github.com/cespare/reflex" && exit 1)
	reflex -s -r '\.go$$' -- $(GO) run $(MAIN_FILE) -config $(CONFIG_DIR)/config.yaml

# 性能测试
.PHONY: bench
bench:
	$(GOTEST) -bench=. -benchmem ./...

# 竞争检测
.PHONY: race
race:
	$(GOTEST) -race ./...

# 内存泄漏检测
.PHONY: memory
memory:
	$(GOTEST) -memprofile=mem.prof -bench=. ./...
	$(GO) tool pprof mem.prof

# CPU性能分析
.PHONY: cpu
cpu:
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	$(GO) tool pprof cpu.prof

# 安装预提交钩子
.PHONY: install-hooks
install-hooks:
	@echo "安装预提交钩子..."
	@cp scripts/pre-commit .git/hooks/ || echo "请先初始化git仓库"
	@chmod +x .git/hooks/pre-commit

# 检查安全漏洞
.PHONY: security
security:
	@which gosec > /dev/null || (echo "安装 gosec: go get github.com/securecodewarrior/gosec/v2/cmd/gosec" && exit 1)
	gosec ./...

# 依赖检查
.PHONY: deps-check
deps-check:
	@which go-mod-tidy-check > /dev/null || (echo "安装 go-mod-tidy-check: go get github.com/ldez/go-mod-tidy-check" && exit 1)
	go-mod-tidy-check

# 更新依赖
.PHONY: deps-update
deps-update:
	$(GO) get -u ./...
	$(GO) mod tidy

# 查看依赖树
.PHONY: deps-tree
deps-tree:
	@which go-mod-graph > /dev/null || (echo "安装 go-mod-graph: go get github.com/PotatoDev/go-mod-graph" && exit 1)
	go-mod-graph | dot -Tpng -o deps-graph.png
	@echo "依赖图已生成: deps-graph.png"