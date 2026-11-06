package operations

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// ReferenceResolver 引用解析器接口
// 将Excel中用户填写的名称/编码转换为数据库中的ID
type ReferenceResolver interface {
	ResolveBatch(ctx context.Context, refs []ReferenceRequest) (map[string]string, error)
	ResolveBatchWithDependencies(ctx context.Context, refs []ReferenceRequest, resolvedIDs map[string]string) (map[string]string, error)
	ResolveSingle(ctx context.Context, ref ReferenceRequest) (string, error)
	ResolveSingleWithCondition(ctx context.Context, ref ReferenceRequest, conditions map[string]string) (string, error)
	// ResolveBatchWithCondition 批量解析引用值，使用相同的条件（如按楼宇分组查询楼层）
	ResolveBatchWithCondition(ctx context.Context, refType string, values []string, conditions map[string]string) (map[string]string, error)
}

// ReferenceRequest 引用解析请求
type ReferenceRequest struct {
	Reference string // 引用配置，格式："table.field"，例如："sys_dept.dept_code"
	Value     string // 待解析的值（用户在Excel中填写的名称/编码）
}

// NewReferenceResolver 创建引用解析器
func NewReferenceResolver(db *gorm.DB) ReferenceResolver {
	return &referenceResolverImpl{db: db}
}

// referenceResolverImpl 引用解析器实现
type referenceResolverImpl struct {
	db *gorm.DB
}

// tablesWithoutSoftDelete 不支持软删除的表名列表（没有deleted_at列）
var tablesWithoutSoftDelete = map[string]bool{
	"sys_device_port_status": true, // 设备端口状态表（实时采集数据，无需软删除）
	"sys_dept":               true, // 部门表（无 deleted_at 列，部门管理走 status 字段软启用/停用）
}

// ResolveBatch 批量解析引用值
// 通过按引用类型分组，每种类型只执行一次查询，大幅提升性能
func (r *referenceResolverImpl) ResolveBatch(
	ctx context.Context,
	refs []ReferenceRequest,
) (map[string]string, error) {
	if len(refs) == 0 {
		return make(map[string]string), nil
	}

	// 按引用类型分组
	grouped := r.groupByReference(refs)

	result := make(map[string]string)
	var firstErr error

	// 对每种引用类型执行一次查询
	for refType, requests := range grouped {
		table, field := r.parseReference(refType)

		// 验证引用配置格式。
		//
		// ⚠️ 纵深防御（工位导入填主设备序列号后报"所属楼宇不存在"事故）：
		// 此处曾直接 `return nil, err` 中断**整批**解析，导致该批次里
		// 其他合法引用（楼宇/部门/用户）全部丢失解析结果，
		// 最终表现为"引用的 所属楼宇 不存在"这类与真实原因无关的报错。
		// 现在改为跳过无效分组并继续处理其余引用，同时保留已解析的部分结果。
		if table == "" || field == "" {
			if firstErr == nil {
				firstErr = fmt.Errorf("无效的引用配置: %s", refType)
			}
			logger.Warnf("跳过无效的引用配置 [%s]，继续解析其余引用", refType)
			continue
		}

		// 提取所有待解析的值（去重）
		values := r.extractValues(requests)

		// 批量查询数据库
		idMap, err := r.batchQueryIDs(ctx, table, field, values)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("批量查询失败 [%s.%s]: %w", table, field, err)
			}
			// 继续处理其他引用类型
			continue
		}

		// 组装结果
		for _, req := range requests {
			key := r.makeKey(req.Reference, req.Value)
			if id, exists := idMap[req.Value]; exists {
				result[key] = id
			}
			// 如果找不到对应ID，不设置result中的key，调用者可以判断引用不存在
		}
	}

	return result, firstErr
}

// ResolveSingle 单个引用解析（用于回退或单个查询）
func (r *referenceResolverImpl) ResolveSingle(
	ctx context.Context,
	ref ReferenceRequest,
) (string, error) {
	table, field := r.parseReference(ref.Reference)

	if table == "" || field == "" {
		return "", fmt.Errorf("无效的引用配置: %s", ref.Reference)
	}

	var result struct {
		ID string `gorm:"column:id;type:uuid"`
	}

	query := r.db.WithContext(ctx).
		Table(table).
		Select("id").
		Where(field+" = ?", ref.Value)

	// 只对支持软删除的表添加 deleted_at IS NULL 条件
	if !tablesWithoutSoftDelete[table] {
		query = query.Where("deleted_at IS NULL")
	}

	err := query.First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("引用记录不存在 [%s.%s = %s]", table, field, ref.Value)
		}
		return "", fmt.Errorf("引用解析失败 [%s.%s = %s]: %w", table, field, ref.Value, err)
	}

	return result.ID, nil
}

// ResolveBatchWithDependencies 批量解析引用值，支持依赖条件
// resolvedIDs 包含已解析的字段值（如 buildingName -> building_id）
func (r *referenceResolverImpl) ResolveBatchWithDependencies(
	ctx context.Context,
	refs []ReferenceRequest,
	resolvedIDs map[string]string,
) (map[string]string, error) {
	if len(refs) == 0 {
		return make(map[string]string), nil
	}

	result := make(map[string]string)
	var firstErr error

	for _, req := range refs {
		table, field := r.parseReference(req.Reference)
		if table == "" || field == "" {
			// 纵深防御：跳过无效引用配置，不中断整批（详见 ResolveBatch 同处注释）
			if firstErr == nil {
				firstErr = fmt.Errorf("无效的引用配置: %s", req.Reference)
			}
			logger.Warnf("跳过无效的引用配置 [%s]，继续解析其余引用", req.Reference)
			continue
		}

		// 使用带条件查询的方式
		id, err := r.ResolveSingleWithCondition(ctx, req, resolvedIDs)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		key := r.makeKey(req.Reference, req.Value)
		result[key] = id
	}

	return result, firstErr
}

// ResolveSingleWithCondition 单个引用解析，支持额外的查询条件
// conditions 格式：{"building_id": "xxx"} 表示只查询指定楼宇下的记录
func (r *referenceResolverImpl) ResolveSingleWithCondition(
	ctx context.Context,
	ref ReferenceRequest,
	conditions map[string]string,
) (string, error) {
	table, field := r.parseReference(ref.Reference)
	if table == "" || field == "" {
		return "", fmt.Errorf("无效的引用配置: %s", ref.Reference)
	}

	var result struct {
		ID string `gorm:"column:id;type:uuid"`
	}

	query := r.db.WithContext(ctx).
		Table(table).
		Select("id").
		Where(field+" = ?", ref.Value)

	// 只对支持软删除的表添加 deleted_at IS NULL 条件
	if !tablesWithoutSoftDelete[table] {
		query = query.Where("deleted_at IS NULL")
	}

	// 添加额外的查询条件
	for condField, condValue := range conditions {
		query = query.Where(condField+" = ?", condValue)
	}

	err := query.First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 构建详细的错误信息
			conditionInfo := ""
			if len(conditions) > 0 {
				parts := make([]string, 0, len(conditions))
				for k, v := range conditions {
					parts = append(parts, fmt.Sprintf("%s=%s", k, v))
				}
				conditionInfo = fmt.Sprintf(" (条件: %s)", strings.Join(parts, ", "))
			}
			return "", fmt.Errorf("引用记录不存在 [%s.%s = %s%s]", table, field, ref.Value, conditionInfo)
		}
		return "", fmt.Errorf("引用解析失败 [%s.%s = %s]: %w", table, field, ref.Value, err)
	}

	return result.ID, nil
}

// groupByReference 按引用类型分组
func (r *referenceResolverImpl) groupByReference(
	refs []ReferenceRequest,
) map[string][]ReferenceRequest {
	grouped := make(map[string][]ReferenceRequest)
	for _, ref := range refs {
		grouped[ref.Reference] = append(grouped[ref.Reference], ref)
	}
	return grouped
}

// parseReference 解析引用配置 "table.field" -> (table, field)
func (r *referenceResolverImpl) parseReference(ref string) (table, field string) {
	parts := strings.Split(ref, ".")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// extractValues 提取所有待解析的值（去重）
func (r *referenceResolverImpl) extractValues(requests []ReferenceRequest) []string {
	values := make([]string, 0, len(requests))
	seen := make(map[string]bool)

	for _, req := range requests {
		if !seen[req.Value] {
			values = append(values, req.Value)
			seen[req.Value] = true
		}
	}

	return values
}

// batchQueryIDs 批量查询ID映射 value -> id
func (r *referenceResolverImpl) batchQueryIDs(
	ctx context.Context,
	table, field string,
	values []string,
) (map[string]string, error) {
	if len(values) == 0 {
		return make(map[string]string), nil
	}

	const maxBatchChunk = 400 // 防止 SQLite 999 或 PG 参数超限

	// 针对 sys_dept：同时支持按 dept_name（部门名称）和 dept_code（部门代码）双向匹配。
	//
	// 优先级按 field 语义隔离，避免「名称命中」与「代码命中」互相污染：
	//   - field=dept_code → 代码命中为主、名称命中为兜底（用户误把名称填进「部门代码」列仍可导入）
	//   - field=dept_name → 名称命中为主、代码命中为兜底
	// 两者共享同一个 idMap，因此「主」必须后写（后写覆盖先写）。
	if table == "sys_dept" && (field == "dept_name" || field == "dept_code") {
		type DeptResult struct {
			ID       string `gorm:"column:id;type:uuid"`
			DeptName string `gorm:"column:dept_name"`
			DeptCode string `gorm:"column:dept_code"`
		}
		idMap := make(map[string]string)
		primaryHits := make(map[string]string)  // 主字段命中（按 field 语义）
		fallbackHits := make(map[string]string) // 另一字段命中（兜底）
		for i := 0; i < len(values); i += maxBatchChunk {
			end := i + maxBatchChunk
			if end > len(values) {
				end = len(values)
			}
			chunk := values[i:end]

			var deptResults []DeptResult
			query := r.db.WithContext(ctx).Table("sys_dept").
				Select("id, dept_name, dept_code").
				Where("dept_name IN ? OR dept_code IN ?", chunk, chunk)
			if !tablesWithoutSoftDelete["sys_dept"] {
				query = query.Where("deleted_at IS NULL")
			}
			if err := query.Find(&deptResults).Error; err != nil {
				return nil, err
			}
			for _, d := range deptResults {
				if field == "dept_code" {
					if d.DeptCode != "" {
						primaryHits[d.DeptCode] = d.ID
						primaryHits[strings.TrimSpace(d.DeptCode)] = d.ID
					}
					if d.DeptName != "" {
						fallbackHits[d.DeptName] = d.ID
						fallbackHits[strings.TrimSpace(d.DeptName)] = d.ID
					}
				} else {
					if d.DeptName != "" {
						primaryHits[d.DeptName] = d.ID
						primaryHits[strings.TrimSpace(d.DeptName)] = d.ID
					}
					if d.DeptCode != "" {
						fallbackHits[d.DeptCode] = d.ID
						fallbackHits[strings.TrimSpace(d.DeptCode)] = d.ID
					}
				}
			}
		}
		// 合并：兜底先填，主命中后填 → 主匹配永远优先，且与 DB 返回顺序无关
		for k, v := range fallbackHits {
			idMap[k] = v
		}
		for k, v := range primaryHits {
			idMap[k] = v
		}
		return idMap, nil
	}

	// 针对 sys_user：同时支持按 username（登录账号）和 nickname（人员姓名/昵称）双向匹配。
	//
	// 优先级同样按 field 语义隔离，避免「账号命中」与「姓名命中」互相污染：
	//   - field=username → 账号命中为主、姓名命中为兜底
	//   - field=nickname → 姓名命中为主、账号命中为兜底
	if table == "sys_user" && (field == "username" || field == "nickname") {
		type UserResult struct {
			ID       string `gorm:"column:id;type:uuid"`
			Username string `gorm:"column:username"`
			Nickname string `gorm:"column:nickname"`
		}
		idMap := make(map[string]string)
		primaryHits := make(map[string]string)  // 主字段命中（按 field 语义）
		fallbackHits := make(map[string]string) // 另一字段命中（兜底）
		for i := 0; i < len(values); i += maxBatchChunk {
			end := i + maxBatchChunk
			if end > len(values) {
				end = len(values)
			}
			chunk := values[i:end]

			var userResults []UserResult
			query := r.db.WithContext(ctx).Table("sys_user").
				Select("id, username, nickname").
				Where("username IN ? OR nickname IN ?", chunk, chunk)
			if !tablesWithoutSoftDelete["sys_user"] {
				query = query.Where("deleted_at IS NULL")
			}
			if err := query.Find(&userResults).Error; err != nil {
				return nil, err
			}
			for _, u := range userResults {
				if field == "username" {
					if u.Username != "" {
						primaryHits[u.Username] = u.ID
						primaryHits[strings.TrimSpace(u.Username)] = u.ID
					}
					if u.Nickname != "" {
						fallbackHits[u.Nickname] = u.ID
						fallbackHits[strings.TrimSpace(u.Nickname)] = u.ID
					}
				} else {
					if u.Nickname != "" {
						primaryHits[u.Nickname] = u.ID
						primaryHits[strings.TrimSpace(u.Nickname)] = u.ID
					}
					if u.Username != "" {
						fallbackHits[u.Username] = u.ID
						fallbackHits[strings.TrimSpace(u.Username)] = u.ID
					}
				}
			}
		}
		// 合并：兜底先填，主命中后填 → 主匹配永远优先，且与 DB 返回顺序无关
		for k, v := range fallbackHits {
			idMap[k] = v
		}
		for k, v := range primaryHits {
			idMap[k] = v
		}
		return idMap, nil
	}

	type Result struct {
		ID    string `gorm:"column:id;type:uuid"`
		Value string `gorm:"column:value"`
	}

	idMap := make(map[string]string)
	for i := 0; i < len(values); i += maxBatchChunk {
		end := i + maxBatchChunk
		if end > len(values) {
			end = len(values)
		}
		chunk := values[i:end]

		var results []Result
		// 构建查询：SELECT id, {field} FROM {table} WHERE {field} IN (...)
		query := r.db.WithContext(ctx).
			Table(table).
			Select("id, "+field+" as value").
			Where(field+" IN ?", chunk)

		// 只对支持软删除的表添加 deleted_at IS NULL 条件
		if !tablesWithoutSoftDelete[table] {
			query = query.Where("deleted_at IS NULL")
		}

		if err := query.Find(&results).Error; err != nil {
			return nil, err
		}

		// 转换为 value -> id 映射
		for _, r := range results {
			idMap[r.Value] = r.ID
		}
	}

	return idMap, nil
}

// ResolveDept resolves department name to dept_id
func (r *referenceResolverImpl) ResolveDept(ctx context.Context, deptName string) (string, error) {
	if deptName == "" {
		return "", nil
	}

	var deptID string
	err := r.db.WithContext(ctx).
		Table("sys_dept").
		Where("dept_name = ? OR dept_code = ?", deptName, deptName).
		Pluck("id", &deptID).Error

	if err != nil {
		return "", fmt.Errorf("查询部门失败: %w", err)
	}

	if deptID == "" {
		return "", fmt.Errorf("部门不存在: %s", deptName)
	}

	return deptID, nil
}

// ResolveUser resolves username to user_id
func (r *referenceResolverImpl) ResolveUser(ctx context.Context, userName string) (string, error) {
	if userName == "" {
		return "", nil
	}

	var userID string
	err := r.db.WithContext(ctx).
		Table("sys_user").
		Where("username = ? OR nickname = ?", userName, userName).
		Pluck("id", &userID).Error

	if err != nil {
		return "", fmt.Errorf("查询用户失败: %w", err)
	}

	if userID == "" {
		return "", fmt.Errorf("用户不存在: %s", userName)
	}

	return userID, nil
}

// ResolveAssetDept resolves department name to dept_id for asset import
// Usage: Excel import with "deptName" column -> dept_id
func (r *referenceResolverImpl) ResolveAssetDept(ctx context.Context, deptName string) (string, error) {
	return r.ResolveDept(ctx, deptName)
}

// ResolveAssetUser resolves username to user_id for asset import
// Usage: Excel import with "userName" column -> user_id
func (r *referenceResolverImpl) ResolveAssetUser(ctx context.Context, userName string) (string, error) {
	return r.ResolveUser(ctx, userName)
}

// makeKey 生成缓存键，格式："reference:value"
func (r *referenceResolverImpl) makeKey(ref, value string) string {
	return ref + ":" + value
}

// ResolveBatchWithCondition 批量解析引用值，使用相同的条件
// refType: 引用类型，如 "ops_floors.name"
// values: 待解析的值列表（楼层名称列表）
// conditions: 查询条件（如 {"building_id": "xxx-uuid"}）
// 返回: value -> id 映射
func (r *referenceResolverImpl) ResolveBatchWithCondition(
	ctx context.Context,
	refType string,
	values []string,
	conditions map[string]string,
) (map[string]string, error) {
	if len(values) == 0 {
		return make(map[string]string), nil
	}

	table, field := r.parseReference(refType)
	if table == "" || field == "" {
		return nil, fmt.Errorf("无效的引用配置: %s", refType)
	}

	type Result struct {
		ID    string `gorm:"column:id;type:uuid"`
		Value string `gorm:"column:value"`
	}

	var results []Result

	query := r.db.WithContext(ctx).
		Table(table).
		Select("id, "+field+" as value").
		Where(field+" IN ?", values)

	// 只对支持软删除的表添加 deleted_at IS NULL 条件
	if !tablesWithoutSoftDelete[table] {
		query = query.Where("deleted_at IS NULL")
	}

	// 添加额外的查询条件
	for condField, condValue := range conditions {
		query = query.Where(condField+" = ?", condValue)
	}

	err := query.Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询失败 [%s.%s]: %w", table, field, err)
	}

	// 转换为 value -> id 映射
	idMap := make(map[string]string)
	for _, r := range results {
		idMap[r.Value] = r.ID
	}

	return idMap, nil
}
