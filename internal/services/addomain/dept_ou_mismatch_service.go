package addomain

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ============================================================================
// Phase 59 / GUARD-02：部门↔OU 对账只读查询服务
//
// 数据源是普通 SQL 视图 v_dept_ou_mismatch（migration_217，D-09~D-13），
// 服务层**只做**「白名单校验 + 参数绑定筛选 + Count 后分页 + 确定性排序」。
//
// ⚠ 分页归一化（默认 1/10、上限 200）是 **handler 的职责**（query.NormalizePagination），
// 本服务不归一化：service 被多个调用方复用时，静默改写调用方传入的请求对象
// 会让上层拿到的 current/pageSize 与响应元数据不一致。所有用例显式传 Current=1,
// PageSize=10 —— 传 0/0 会走 Limit(0) 得到空集，让断言假失败。
//
// ⚠ 只读端点：不写 operlog、不写任何业务表。
// ============================================================================

// MismatchTypes D-10 的四类不一致口径，**单一事实源**。
//
// 视图的 mismatch_type 列只产出这四个字面量；handler 用它做入参白名单校验，
// 服务测试用它遍历断言。新增第五类时必须同时改视图 DDL 与本列表，
// 两处漂移会让「视图能查出来但接口拒绝筛选」这类不对称缺陷静默存在。
var MismatchTypes = []string{
	"user_ou_mismatch",
	"dept_soft_deleted",
	"mapping_disabled",
	"config_unrouted",
}

// validMismatchType 判定 mismatchType 是否属于 D-10 四值白名单。
//
// 空串不在此判定范围内：空 = 「不按类型筛选」，由调用方在调用前以 `!= ""` 短路。
// 非法值不静默忽略——handler 必须返回 400（参数无效），否则用户会拿到
// 一个「看起来正常但永远空」的分页结果，把拼写错误伪装成「没有不一致」。
func validMismatchType(t string) bool {
	for _, v := range MismatchTypes {
		if v == t {
			return true
		}
	}
	return false
}

// escapeLike 转义 LIKE 模式元字符，使 `%` / `_` / `\` 在关键字里被当作字面量。
//
// ⚠ 顺序敏感：**必须先转义反斜杠**，否则后续插入的 `\%` 会被第二步再转义成 `\\%`，
// 让 `%` 重新变回通配符（经典的双重转义缺陷）。
// 使用方必须配 `LIKE ? ESCAPE '\'`，否则转义序列无意义。
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

// DeptOUMismatchQuery 对账查询请求（D-16）。
// json tag 与前端 POST body 逐字对齐。
type DeptOUMismatchQuery struct {
	Current      int    `json:"current"`
	PageSize     int    `json:"pageSize"`
	MismatchType string `json:"mismatchType,omitempty"`
	DeptID       string `json:"deptId,omitempty"`
	Keyword      string `json:"keyword,omitempty"`
}

// DeptOUMismatchRow 视图 v_dept_ou_mismatch 的一行（字段顺序对齐 D-11 十一列）。
//
// ID 列与时间列一律 *string：视图里它们都是 `CAST(... AS TEXT)`，
// 用 *string 可以同时躲开 PG 的 uuid/time 与 sqlite 的 TEXT 之间的驱动层差异
// （RESEARCH Open Question 6 的裁决）。部门级口径（②③）的 user 三列为 NULL，
// 用户级口径（④）的配置三列为 NULL，故除 mismatch_type / mismatch_detail 外全部可空。
// ⚠ 每个字段都显式声明 `gorm:"column:..."`：GORM 的默认命名策略只会对常见首字母
// 缩写做替换，`MappedOUDN` 会被推成 `mapped_oudn`、`UserADOUDN` → `user_adoudn`，
// 两者与 D-11 锁定的视图列名 `mapped_ou_dn` / `user_ad_ou_dn` 都不一致 →
// `Scan` 时那两列静默丢值（无报错，字段恒 nil）。显式列名同时把 D-11 契约写进源码。
type DeptOUMismatchRow struct {
	DeptID         *string `gorm:"column:dept_id" json:"deptId"`
	DeptName       *string `gorm:"column:dept_name" json:"deptName"`
	DeptDeletedAt  *string `gorm:"column:dept_deleted_at" json:"deptDeletedAt"`
	ADConfigID     *string `gorm:"column:ad_config_id" json:"adConfigId"`
	ConfigName     *string `gorm:"column:config_name" json:"configName"`
	MappedOUDN     *string `gorm:"column:mapped_ou_dn" json:"mappedOuDn"`
	UserID         *string `gorm:"column:user_id" json:"userId"`
	Username       *string `gorm:"column:username" json:"username"`
	UserADOUDN     *string `gorm:"column:user_ad_ou_dn" json:"userAdOuDn"`
	MismatchType   string  `gorm:"column:mismatch_type" json:"mismatchType"`
	MismatchDetail string  `gorm:"column:mismatch_detail" json:"mismatchDetail"`
}

// DeptOUMismatchService 部门↔OU 对账只读查询服务接口。
type DeptOUMismatchService interface {
	// ListDeptOUMismatch 分页查询 v_dept_ou_mismatch（确定性排序，先 Count 后分页）
	ListDeptOUMismatch(ctx context.Context, req *DeptOUMismatchQuery) ([]DeptOUMismatchRow, int64, error)
}

// deptOUMismatchServiceImpl 私有实现。
type deptOUMismatchServiceImpl struct {
	db *gorm.DB
}

// NewDeptOUMismatchService 创建对账查询服务。
func NewDeptOUMismatchService(db *gorm.DB) DeptOUMismatchService {
	return &deptOUMismatchServiceImpl{db: db}
}

// ListDeptOUMismatch 查询对账条目。
//
// 安全约束（T-59-06）：mismatchType / deptId / keyword 全部走 GORM 参数绑定，
// **绝不**拼接进 SQL 文本。mismatchType 另有白名单把关（handler 层 400）。
//
// 确定性排序（T-59-08 / Pitfall 10）：视图是 UNION ALL，没有固有行序；
// 不加 ORDER BY 会让翻页重复或漏行。排序键覆盖全部可区分列，
// 且全部已在视图内 CAST 为文本，两种方言下都可比较。
func (s *deptOUMismatchServiceImpl) ListDeptOUMismatch(ctx context.Context, req *DeptOUMismatchQuery) ([]DeptOUMismatchRow, int64, error) {
	// 空结果必须序列化为 `[]` 而非 `null`（must_have 锁定）
	rows := make([]DeptOUMismatchRow, 0)

	q := s.db.WithContext(ctx).Table("v_dept_ou_mismatch")

	if req.MismatchType != "" {
		q = q.Where("mismatch_type = ?", req.MismatchType)
	}
	if req.DeptID != "" {
		q = q.Where("dept_id = ?", req.DeptID)
	}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		// 大小写不敏感包含匹配，覆盖 D-16 的四个可搜索列。
		// LIKE 元字符先转义再包通配符，并用 ESCAPE '\' 声明转义字符；
		// 否则输入 `%` / `_` 会退化成「匹配一切」或「匹配任一字符」。
		like := "%" + escapeLike(strings.ToLower(kw)) + "%"
		q = q.Where(
			`(LOWER(COALESCE(username, '')) LIKE ? ESCAPE '\'`+
				` OR LOWER(COALESCE(dept_name, '')) LIKE ? ESCAPE '\'`+
				` OR LOWER(COALESCE(mapped_ou_dn, '')) LIKE ? ESCAPE '\'`+
				` OR LOWER(COALESCE(user_ad_ou_dn, '')) LIKE ? ESCAPE '\')`,
			like, like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计部门-OU 对账条目总数失败: %w", err)
	}

	offset := (req.Current - 1) * req.PageSize
	if offset < 0 {
		// 防御负 OFFSET（sqlite 会直接报错）；正常路径由 handler 归一化保证 Current >= 1
		offset = 0
	}

	if err := q.Order("mismatch_type, dept_name, username, dept_id, user_id, ad_config_id").
		Offset(offset).Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("查询部门-OU 对账条目失败: %w", err)
	}

	return rows, total, nil
}
