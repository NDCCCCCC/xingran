# scripts/, test/, tests/ 目录整理方案

## 当前状态分析

### scripts/ (54 个文件)
- **33 个 .go 文件** - 迁移脚本和工具脚本
- **11 个 .sql 文件** - SQL 修复脚本
- **4 个 .bat 文件** - Windows 构建脚本
- **2 个 .sh 文件** - Linux 构建脚本
- **其他文件** - README.md, Python 脚本等

### test/ (2 个文件)
- `test/fixtures/lldp_output.go` - LLDP 测试数据
- `test/fixtures/mac_collection.go` - MAC 采集测试数据

### tests/ (4 个文件)
- `tests/benchmark/login_encryption_bench_test.go` - 性能基准测试
- `tests/integration/login_encryption_test.go` - 集成测试
- `tests/e2e/login-encryption.spec.ts` - E2E 测试
- `tests/integration/logs/app.log` - 测试日志

## 整理方案

### 方案 A：保守整理（推荐）
**原则：** 归档已完成的迁移，保留可能使用的工具

#### 1. scripts/ 目录重组
```
scripts/
├── build/              # 构建脚本
│   ├── build.bat
│   ├── build_and_deploy.bat
│   └── generate_swagger.bat
├── migrate/            # 已执行的迁移脚本（归档）
│   ├── 042_*.go
│   ├── 053_*.go
│   ├── 054_*.go
│   ├── 056_*.go
│   ├── 057_*.go
│   ├── 058_*.go
│   ├── 074_*.go
│   ├── 075_*.go
│   ├── 076_*.go
│   ├── 077_*.go
│   ├── 096_*.go
│   ├── 098_*.go
│   ├── 099_*.go
│   ├── 104_*.go
│   ├── 106_*.go
│   └── 113_*.go
├── tools/              # 可能重复使用的工具
│   ├── check_and_fix_apikey_menu.go
│   ├── check_all_including_buttons.go
│   ├── cleanup_menus.go
│   ├── execute_cleanup.go
│   ├── find_duplicate_names.go
│   ├── find_real_duplicates.go
│   ├── full_check.go
│   ├── generate_cleanup.go
│   ├── list_all_menus.go
│   └── search_menus.go
├── sql/                # SQL 修复脚本
│   ├── *.sql
│   └── migrate/        # SQL 迁移子目录
└── README.md           # 更新说明文档
```

#### 2. test/ 目录保持不变
- 仅有 2 个文件，已经是合理的结构
- 移动到 `internal/test/fixtures/` 可能更合适

#### 3. tests/ 目录重组
```
tests/
├── benchmark/          # 性能基准测试
│   └── login_encryption_bench_test.go
├── integration/        # 集成测试
│   ├── login_encryption_test.go
│   └── logs/           # 测试日志（添加到 .gitignore）
└── e2e/                # 端到端测试
    └── login-encryption.spec.ts
```

### 方案 B：激进整理
**原则：** 删除所有已执行的迁移，只保留活跃使用的工具

#### 删除内容
- `scripts/migrate/` - 所有已执行的迁移脚本（已在数据库中应用）
- `scripts/sql/` - 历史修复 SQL（已归档到 internal/core/db/migrations/archive/）
- `scripts/tools/` - 菜单修复工具（Phase 17 已完成，不再需要）
- `tests/integration/logs/app.log` - 测试日志（应该被忽略）

#### 保留内容
- `scripts/build/` - 构建脚本（开发时使用）
- `scripts/test_excel_import.go` - 可能还需要使用
- `test/fixtures/` - 活跃使用的测试数据
- `tests/*` - 活跃的测试文件

## 推荐执行方案 A

### 理由
1. **保留历史** - 迁移脚本可能对故障排查有用
2. **工具可能复用** - 菜单工具在遇到类似问题时可能需要
3. **风险较低** - 只是重组目录结构，不删除文件

### 执行步骤
1. 创建新的目录结构
2. 移动文件到对应目录
3. 更新 scripts/README.md
4. 更新 .gitignore（忽略测试日志）
5. 提交更改

是否需要我执行方案 A 的整理？
