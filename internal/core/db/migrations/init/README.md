# 初始数据目录

此目录包含新环境部署所需的初始数据。

## 重要说明

- **表结构**: 由 GORM AutoMigrate 自动创建（项目启动时执行）
- **初始数据**: 需要手动执行 SQL 文件导入

## 使用方法

### 1. 创建数据库
```bash
createdb -U postgres xingran_next
```

### 2. 启动应用（自动创建表结构）
```bash
./xingran-backend
```
应用启动时会自动调用 `AutoMigrate()` 创建所有表。

### 3. 导入初始数据（按顺序执行）

```bash
# 网络设备类型字典
psql -U postgres -d xingran_next -f ../002_init_network_device_type_dict.sql

# 运维管理菜单
psql -U postgres -d xingran_next -f ../006_add_operations_management_menu.sql

# 工单和知识库菜单
psql -U postgres -d xingran_next -f ../013_add_workorder_knowledge_menus.sql

# AD 域管理菜单
psql -U postgres -d xingran_next -f ../017_add_ad_domain_menus.sql

# 缓存配置
psql -U postgres -d xingran_next -f ../025_insert_cache_config.sql

# 专线类型字典
psql -U postgres -d xingran_next -f ../047_add_dedicated_line_type_dict.sql

# ISP 字典
psql -U postgres -d xingran_next -f ../048_add_isp_dict.sql

# 信息点类型字典
psql -U postgres -d xingran_next -f ../033_add_info_point_type_dict.sql

# RPA 菜单（如果使用 RPA 功能）
psql -U postgres -d xingran_next -f ../106_add_rpa_menus.sql

# API 密钥管理菜单（最终版本）
psql -U postgres -d xingran_next -f ../123_create_apikey_menu_simple.sql
```

## 文件说明

| 文件 | 用途 | 必需 |
|------|------|------|
| 002_init_network_device_type_dict.sql | 网络设备类型字典 | ✅ |
| 006_add_operations_management_menu.sql | 运维管理菜单 | ✅ |
| 013_add_workorder_knowledge_menus.sql | 工单知识库菜单 | ✅ |
| 017_add_ad_domain_menus.sql | AD 域管理菜单 | ⚠️ |
| 025_insert_cache_config.sql | 缓存配置 | ✅ |
| 033_add_info_point_type_dict.sql | 信息点类型字典 | ✅ |
| 047_add_dedicated_line_type_dict.sql | 专线类型字典 | ⚠️ |
| 048_add_isp_dict.sql | ISP 字典 | ⚠️ |
| 106_add_rpa_menus.sql | RPA 菜单 | ⚠️ |
| 123_create_apikey_menu_simple.sql | API 密钥管理菜单 | ✅ |

## 归档说明

`../archive/` 目录包含历史修复类迁移文件，仅用于问题追溯和已有数据库升级参考。新环境部署不需要执行这些文件。

## 生成时间

2026-05-21
