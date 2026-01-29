# Excel导入功能单元测试

## 测试文件

### 1. cache_invalidator_test.go ✅
测试缓存清理器的功能：
- 按实体类型清理缓存
- 按模式列表清理缓存
- 空缓存提供者处理

**状态**: 无需数据库，可直接运行

### 2. reference_resolver_test.go
测试引用解析器的功能：
- 批量解析引用（部门、楼宇等）
- 单个引用解析
- 无效引用处理

**状态**: 需要数据库（SQLite + CGO）

### 3. batch_upserter_test.go
测试批量更新器的功能：
- 批量插入新记录
- 批量更新已存在的记录
- 混合插入和更新操作
- 更新列构建逻辑

**状态**: 需要数据库（SQLite + CGO）

---

## 运行测试

### 方式1: 跳过数据库测试（推荐）
```bash
# 只运行不需要数据库的测试
go test ./internal/services/operations/... -v -tags skip_db_tests
```

### 方式2: 运行所有测试（需要CGO）
```bash
# Windows PowerShell
$env:CGO_ENABLED=1; go test ./internal/services/operations/... -v

# Windows CMD
set CGO_ENABLED=1 && go test ./internal/services/operations/... -v

# Linux/Mac
CGO_ENABLED=1 go test ./internal/services/operations/... -v
```

### 安装依赖
```bash
go mod tidy
```

### 运行特定测试
```bash
# 只运行缓存测试
go test ./internal/services/operations/... -v -run TestCacheInvalidator

# 运行引用解析测试（需要数据库）
go test ./internal/services/operations/... -v -run TestReferenceResolver -tags !skip_db_tests
```

### 测试覆盖率
```bash
# 跳过数据库测试的覆盖率
go test -cover ./internal/services/operations/... -tags skip_db_tests
```

---

## 测试说明

### 构建标签
- `skip_db_tests`: 跳过需要数据库的测试
- 使用 `+build !skip_db_tests` 标记的文件在添加 `-tags skip_db_tests` 时会被跳过

### 数据库要求
需要数据库的测试使用纯 Go SQLite 驱动 (`modernc.org/sqlite`)：
- 优点：无需安装 GCC/MinGW
- 缺点：在某些环境下仍可能有 CGO 问题

### Mock对象
`cache_invalidator_test.go` 使用 `testify/mock` 框架模拟缓存提供者。

---

## CI/CD 集成

### GitHub Actions 示例
```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - name: Run tests
        run: |
          go test -v -tags skip_db_tests ./internal/services/operations/...
```

---

## 当前测试状态

| 测试文件 | 状态 | 覆盖功能 |
|---------|------|---------|
| cache_invalidator_test.go | ✅ 通过 | 缓存清理逻辑 |
| reference_resolver_test.go | ⚠️ 需数据库 | 引用解析 |
| batch_upserter_test.go | ⚠️ 需数据库 | 批量更新 |

**建议**: 在 CI/CD 中使用 `-tags skip_db_tests` 跳过数据库测试，确保核心逻辑测试通过。
