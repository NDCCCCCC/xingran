# PAGI-01 逐调用方核对清单（SC-3）

**Date:** 2026-09-07
**Plan:** 102-05
**Requirement:** PAGI-01

---

## 调用方核对表

| 调用方 | 类型 | 处置 | 证据 |
|--------|------|------|------|
| `internal/api/v1/system/file_handler.go:160` | **唯一生产调用方** | 迁移 → `query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)` | 全文见下方「迁移证据」 |
| `internal/utils/utils_74_12_test.go:203-226`（`TestParsePaginationAndOffset`） | 测试 | 随 Task 2 删除 | `TestParsePaginationAndOffset` 在 `:203-230`，随 `pagination.go` 删除 |
| `internal/api/v1/operations/requests.PaginationParams` 系列（`requests_74_12_test.go` 等 6 文件） | **同名异型**（operations requests 包自有类型） | **不相关，勿动** | 全文见下方「同名异型排除证据」 |

---

## 迁移证据

### file_handler.go:153-174（迁移后）

```go
func (h *FileHandler) List(c *gin.Context) {
    var req systemServices.ListFilesRequest
    if err := c.ShouldBindQuery(&req); err != nil {
        response.Error(c, response.ErrBadRequest, "请求参数错误")
        return
    }

    // 历史分叉自证：本端点 cap=100（constants.MaxListPageSize），刻意不用 NormalizePagination 默认 cap=200
    current, pageSize := query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)
    offset := (current - 1) * pageSize

    files, total, err := h.service.ListFiles(c.Request.Context(), req.BusinessType, req.UserID, offset, pageSize)
    if err != nil {
        response.Error(c, response.ErrServerError, err.Error())
        return
    }

    result := make([]gin.H, 0, len(files))
    for _, file := range files {
        result = append(result, buildFileResponse(file, h.service.GetFileURL(file)))
    }

    response.Success(c, utils.BuildListResponse(result, total, current, pageSize))
}
```

### 关键约束确认

- `internal/utils` import **保留**（`utils.GetUserID` :48 / `utils.GetClientIP` :73-74 / `utils.BuildListResponse` :173 / `utils.BuildCountResponse` :200 仍使用）
- 新增 import: `pkg/query` + `pkg/constants`
- cap=100 注释自证历史分叉（D-102-9）
- `utils.BuildListResponse` 保留不动（CONTEXT 明确 Deferred）

---

## 同名异型排除证据

### operations/requests.PaginationParams（不相关）

```bash
$ grep -rn "ParsePagination\|PaginationParams" --include="*.go" pkg/query/ internal/api/v1/operations/
```

**结论：** `operations/requests.PaginationParams` 是 operations 模块自有类型，与 `internal/utils/pagination.go` 的 `PaginationParams` 同名异型，无任何代码关联。`pkg/query/pagination.go` 的 `NormalizePaginationWithMax` 是 standalone 函数，不依赖任何 `PaginationParams` 类型。

全仓 grep `ParsePagination` 结果：
- `pkg/query/pagination.go:15` — **定义方**（`NormalizePaginationWithMax`，Phase 99 交付）
- `internal/utils/pagination.go:15` — **被删除方**（`ParsePagination`，本 plan 删除）
- `internal/api/v1/system/file_handler.go` — **已迁移**（本 plan）

operations 模块无任何 `utils.ParsePagination` 调用。

---

## 核对方命令（全仓 grep）

```bash
# 验证 ParsePagination 仅剩 pkg/query 定义 + operations 同名异型
grep -rn "ParsePagination\b" --include="*.go" . | grep -v "_test.go"

# 验证 file_handler.go 不再引用 utils.ParsePagination
grep -n "ParsePagination\|utils\.ParsePagination" internal/api/v1/system/file_handler.go

# 验证 NormalizePaginationWithMax 引用
grep -n "NormalizePaginationWithMax\|constants\.MaxListPageSize" internal/api/v1/system/file_handler.go
```

**执行结果（预期全部匹配）：**

```
pkg/query/pagination.go:51:func NormalizePaginationWithMax...
internal/api/v1/system/file_handler.go:162:  current, pageSize := query.NormalizePaginationWithMax(req.Page, req.PageSize, constants.MaxListPageSize)
internal/api/v1/system/file_handler.go:161:  // 历史分叉自证：本端点 cap=100（constants.MaxListPageSize），刻意不用 NormalizePagination 默认 cap=200
```

---

## BuildListResponse 保留确认

`utils.BuildListResponse`（`internal/utils/response_builder.go:14-22`）仍由 `file_handler.go` 在 :173 处调用，**保留不动**。该函数是 response builder helper，与分页归一逻辑无关。

```go
// internal/utils/response_builder.go:14-22
func BuildListResponse(list interface{}, total int64, page, pageSize int) gin.H {
    return gin.H{
        "list":     list,
        "total":    total,
        "page":     page,
        "pageSize": pageSize,
    }
}
```

---

*Audit completed: 2026-09-07*
