# Scripts 说明

本目录包含用于数据库迁移、维护和构建的脚本。

## 目录结构

```
scripts/
├── build/          # 构建脚本
├── .archive-migrate-2026-06-15/  # 已归档的一次性迁移脚本(原 migrate/)
├── tools/          # 可复用的维护工具
├── sql/            # SQL 修复脚本
└── README.md       # 本文档
```

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

## build/ - 构建脚本

用于构建和部署的脚本：

- `build.bat` - 构建后端
- `build_and_deploy.bat` - 构建并部署
- `generate_swagger.bat` - 生成 Swagger 文档

### 使用示例

```bash
cd scripts/build

# 构建项目
./build.bat

# 生成 Swagger 文档
./generate_swagger.bat
```

## .archive-migrate-2026-06-15/ - 已归档的一次性迁移脚本

> **历史归档目录**(2026-06-15 重命名,原名 `migrate/`)。
> 这些一次性运维脚本全部带 `//go:build ignore`,正常编译不会包含。所有脚本对应的迁移已经在生产 DB 执行过,无需重跑。
> 配套的 SQL 文件已移至 `internal/core/db/migrations/archive/legacy-2026-06-15/`。
> 详见归档说明: `internal/core/db/migrations/archive/legacy-2026-06-15/README.md`

历史脚本清单(供参考):

### 用户中心菜单路径修复 (2026-01)
- `056_fix_user_center_paths.go` - 修复用户中心子菜单路径
- `057_fix_user_center_parent_path.go` - 修复用户中心父菜单路径
- `058_fix_user_center_relative_paths.go` - 修复为相对路径

### 菜单路径分析和修复
- `053_fix_menu_paths.go` - 修复菜单路径
- `054_analyze_menu_paths.go` - 分析菜单路径问题

### AD 域设备管理 (2026-01)
- `074_run_migration.go` - 运行电脑设备唯一约束迁移
- `075_cleanup_duplicate_computers.go` - 清理重复的电脑设备
- `076_check_computer_data.go` - 检查电脑设备数据
- `077_run_ou_optimization.go` - 运行OU查询优化

### 其他迁移
- `042_add_hidden_special_page_menus.go` - 添加隐藏的特殊页面菜单
- `096_fix_floor_and_file_tables.go` - 修复楼层和文件表
- `098_add_floor_unique_constraint.go` - 添加楼层唯一约束
- `099_add_workstation_unique_constraint.go` - 添加工位唯一约束
- `104_run_rpa_scaling_migration.go` - RPA 扩展迁移
- `106_run_rpa_migration.go` - RPA 迁移
- `113_fix_screenshots_column.go` - 修复截图列类型

**注意：** 这些脚本已经执行过，通常不需要再次运行。如需参考，请查看脚本中的注释。

## tools/ - 维护工具

可重复使用的维护工具脚本：

### 菜单管理工具
- `check_and_fix_apikey_menu.go` - 检查并修复 API 密钥菜单
- `check_all_including_buttons.go` - 检查所有按钮权限
- `check_menu_permissions.go` - 检查菜单权限
- `cleanup_menus.go` - 清理重复菜单
- `execute_cleanup.go` - 执行清理操作
- `find_duplicate_names.go` - 查找重复名称
- `find_real_duplicates.go` - 查找真实重复项
- `full_check.go` - 完整检查
- `generate_cleanup.go` - 生成清理脚本
- `list_all_menus.go` - 列出所有菜单
- `search_menus.go` - 搜索菜单
- `show_duplicate_details.go` - 显示重复详情

### 其他工具
- `analyze_paths.go` - 深入分析路径关系
- `check_user_menu.go` - 检查用户菜单
- `fix_user_settings_menu.go` - 修复用户设置菜单

### 使用示例

```bash
cd scripts/tools

# 设置环境变量
set DB_HOST=10.62.10.34
set DB_PORT=5432
set DB_USER=postgres
set DB_PASSWORD=<your-db-password>
set DB_NAME=ruoyi

# 列出所有菜单
go run list_all_menus.go

# 检查重复菜单
go run find_real_duplicates.go
```

## sql/ - SQL 脚本

SQL 修复脚本和查询：

### 直接执行 SQL

```bash
cd scripts/sql

# 通过 psql 执行 SQL 文件
psql -h localhost -U postgres -d ruoyi -f check_duplicate_menus.sql
psql -h localhost -U postgres -d ruoyi -f cleanup_duplicate_menus.sql
```

## 测试

### Excel 导入测试
- `.archive-migrate-2026-06-15/test_excel_import.go` - Excel 导入功能测试脚本(已归档)

```bash
cd scripts/.archive-migrate-2026-06-15

# 运行 Excel 导入测试
go run test_excel_import.go
```

## 注意事项

1. **生产环境使用前请备份数据库**
2. 确保环境变量配置正确
3. 建议先在测试环境验证脚本
4. 某些脚本需要按顺序执行（如用户中心路径修复系列）
5. **`.archive-migrate-2026-06-15/` 目录中的脚本已执行过(原 migrate/),通常不需要再次运行**

## 数据库连接工具

所有脚本使用 `internal/db` 包进行数据库连接，提供统一的：

- 配置管理
- 连接池
- 错误处理
- 日志记录
