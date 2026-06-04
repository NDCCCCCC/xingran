.PHONY: all help verify cleanup-dry cleanup-force

# 默认目标
help:
	@echo "缓存键命名规范迁移工具集"
	@echo ""
	@echo "可用命令:"
	@echo "  make verify       - 验证迁移结果（检查旧键是否清理完成）"
	@echo "  make cleanup-dry  - 预览将要删除的旧键（DRY RUN 模式）"
	@echo "  make cleanup-force - 强制删除旧键（实际删除操作）"
	@echo ""
	@echo "迁移步骤:"
	@echo "  1. make verify       - 检查当前状态"
	@echo "  2. make cleanup-dry  - 预览将要删除的键"
	@echo "  3. make cleanup-force - 确认后执行删除"
	@echo "  4. make verify       - 验证清理结果"

# 验证迁移结果
verify:
	@echo "=== 验证缓存键迁移结果 ==="
	@cd verify && go run verify_migration.go

# 预览清理（DRY RUN 模式）
cleanup-dry:
	@echo "=== DRY RUN 模式：预览将要删除的旧键 ==="
	@cd cleanup && make run-dry

# 强制清理旧键
cleanup-force:
	@echo "=== 警告：即将删除旧规范缓存键 ==="
	@cd cleanup && make run-force

# 安装依赖
deps:
	@echo "安装依赖..."
	@go mod tidy
	@cd verify && go mod tidy
	@cd cleanup && go mod tidy

# 清理编译产物
clean:
	@cd cleanup && make clean
	@cd verify && go clean
