# Scripts 说明

本目录包含用于数据库迁移和维护的 Go 脚本。

## 环境变量配置

所有脚本都通过环境变量获取数据库连接信息，请在运行前设置以下环境变量：

```bash
# Windows
set DB_HOST=localhost
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=your_password
set DB_NAME=ruoyi

# Linux/macOS
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=ruoyi
```

或者创建 `.env` 文件并在运行时加载：

```bash
# 使用 direnv 等工具自动加载
# 或手动导入
export $(cat .env | xargs)
```

## 迁移脚本 (migrate/)

### 用户中心菜单路径修复

- `056_fix_user_center_paths.go` - 修复用户中心子菜单路径
- `057_fix_user_center_parent_path.go` - 修复用户中心父菜单路径
- `058_fix_user_center_relative_paths.go` - 修复为相对路径

### AD 域设备管理

- `074_run_migration.go` - 运行电脑设备唯一约束迁移
- `075_cleanup_duplicate_computers.go` - 清理重复的电脑设备
- `076_check_computer_data.go` - 检查电脑设备数据
- `077_run_ou_optimization.go` - 运行OU查询优化

### 菜单路径分析和修复

- `053_fix_menu_paths.go` - 修复菜单路径
- `054_analyze_menu_paths.go` - 分析菜单路径问题
- `analyze_paths.go` - 深入分析路径关系

## 使用示例

```bash
# 进入脚本目录
cd scripts

# 设置环境变量（根据实际情况修改）
set DB_HOST=10.62.10.34
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=Cpic1234
set DB_NAME=ruoyi

# 运行迁移脚本
go run migrate/056_fix_user_center_paths.go

# 检查电脑设备数据
go run migrate/076_check_computer_data.go
```

## 测试脚本 (tests/)

### 功能测试
- `test_excel_import.go` - Excel 导入功能测试脚本

运行测试脚本：
```bash
# 进入脚本目录
cd scripts/tests

# 运行 Excel 导入测试
go run test_excel_import.go
```

## SQL 脚本

根目录下的 SQL 文件可以直接通过 psql 执行：

```bash
# 执行用户中心路径修复
psql -h localhost -U postgres -d ruoyi -f fix_user_center_paths_v2.sql
```

## 注意事项

1. **生产环境使用前请备份数据库**
2. 确保环境变量配置正确
3. 建议先在测试环境验证脚本
4. 某些脚本需要按顺序执行（如用户中心路径修复系列）

## 数据库连接工具

所有脚本使用 `internal/db` 包进行数据库连接，提供统一的：

- 配置管理
- 连接池
- 错误处理
- 日志记录
