.PHONY: build run clean test help dev reload deps fmt lint

# 变量定义
BINARY_NAME=qzone-memory
CMD_DIR=./cmd/server
MAIN_FILE=$(CMD_DIR)/main.go

ifeq ($(OS),Windows_NT)
	BINARY_EXT=.exe
	RM_FILE=if exist "$(BINARY_NAME)$(BINARY_EXT)" del /f /q "$(BINARY_NAME)$(BINARY_EXT)"
	RM_DIR=if exist data rmdir /s /q data & if exist logs rmdir /s /q logs
else
	BINARY_EXT=
	RM_FILE=rm -f "$(BINARY_NAME)$(BINARY_EXT)"
	RM_DIR=rm -rf ./data ./logs
endif

# 默认目标
help:
	@echo "QQ 空间回忆项目 - Makefile 命令"
	@echo ""
	@echo "使用方法:"
	@echo "  make build    - 编译项目"
	@echo "  make run      - 运行项目"
	@echo "  make clean    - 清理编译文件"
	@echo "  make test     - 运行测试"
	@echo "  make dev      - 开发模式"
	@echo "  make reload   - Air 热重启开发模式"
	@echo "  make deps     - 安装依赖"
	@echo "  make fmt      - 格式化代码"
	@echo "  make lint     - 代码检查"
	@echo ""

# 编译项目
build:
	@echo "正在编译项目..."
	@go build -o $(BINARY_NAME)$(BINARY_EXT) $(MAIN_FILE)
	@echo "编译完成: $(BINARY_NAME)$(BINARY_EXT)"

# 运行项目
run: build
	@echo "启动服务..."
	@./$(BINARY_NAME)$(BINARY_EXT)

# 开发模式(Air 热重启)
reload:
	@echo "Air 热重启开发模式启动..."
	@go tool air -c .air.toml

# 开发模式
dev:
	@echo "开发模式启动..."
	@go run $(MAIN_FILE)

# 清理编译文件
clean:
	@echo "清理编译文件..."
	@$(RM_FILE)
	@$(RM_DIR)
	@echo "清理完成"

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod tidy
	@go mod download
	@echo "依赖安装完成"

# 格式化代码
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@echo "格式化完成"

# 代码检查
lint:
	@echo "代码检查..."
	@go vet ./...
	@echo "检查完成"
