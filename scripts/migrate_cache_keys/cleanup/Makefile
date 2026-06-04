.PHONY: build clean run-dry run-force help

# 默认目标
help:
	@echo "缓存键迁移清理工具"
	@echo ""
	@echo "可用命令:"
	@echo "  make build         - 编译清理工具"
	@echo "  make clean         - 清理编译产物"
	@echo "  make run-dry       - 运行 DRY RUN 模式（只查看，不删除）"
	@echo "  make run-force     - 运行强制删除模式（实际删除旧键）"
	@echo "  make test          - 测试模式（使用测试数据库）"
	@echo ""
	@echo "示例:"
	@echo "  make build"
	@echo "  make run-dry"
	@echo "  make run-force"

# 编译工具
build:
	@echo "编译清理工具..."
	@go build -o cleanup_tool cleanup_tool.go
	@echo "编译完成: cleanup_tool"

# 清理编译产物
clean:
	@echo "清理编译产物..."
	@rm -f cleanup_tool
	@echo "清理完成"

# DRY RUN 模式（只查看，不删除）
run-dry: build
	@echo "=== DRY RUN 模式 ==="
	@echo "只显示将要删除的键，不实际删除"
	@echo ""
	./cleanup_tool -dry-run=true -verbose=true

# 强制删除模式
run-force: build
	@echo "=== 警告：即将删除旧规范缓存键 ==="
	@echo "按 Ctrl+C 取消，按 Enter 继续..."
	@read _ignore
	./cleanup_tool -dry-run=false -verbose=true

# 测试模式（使用测试数据库）
test: build
	@echo "=== 测试模式（使用 DB 15） ==="
	./cleanup_tool -db=15 -dry-run=true -verbose=true

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod tidy
	@go get github.com/redis/go-redis/v9
	@go get golang.org/x/net
