package datascope

// user_permission_preview.go —— 用户权限预览（编辑用户时"所见即所得"）。
//
// 需求（用户 2026-11-17）：在编辑用户页、角色字段下方加"权限预览"，
// 除了菜单权限，也要展示数据范围（含机构锚点命中明细）。
//
// 设计要点：
//   - 接受**表单当前值**（尚未保存的 deptId / roleIds / postIds），因此不能复用
//     queryAnchorGrants（它只查库里"已保存"的关联）；
//   - 锚点解析复用 ResolveAnchorByDept + resolveAnchorByChain（与运行期同一内核），
//     保证"预览结果 = 保存后结果"，绝无两套算法漂移；
//   - 数据范围档位仍按"角色 MAX(data_scope)"口径（与 getUserMaxDataScope 对齐）；
//   - 菜单权限由调用方（system service）查 sys_role_menu 汇总，本包只负责数据范围侧。

import (
	"context"
	"fmt"
	"sort"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// UserScopePreview 用户数据范围预览结果（保存前模拟）。
type UserScopePreview struct {
	// DataScope 角色维度生效档位（MAX(data_scope)，与运行期同口径）。
	DataScope models.DataScope `json:"dataScope"`
	// DataScopeLabel 中文档位说明。
	DataScopeLabel string `json:"dataScopeLabel"`
	// All 是否全库（角色含 All(1) → 锚点不参与）。
	All bool `json:"all"`
	// AnchorSources 机构锚点来源解析明细（角色锚点 + 岗位锚点）。
	AnchorSources []UserAnchorSource `json:"anchorSources"`
	// VisibleDeptIDs 最终可见部门 ID 集（去重排序）；All 档为空（表示全库）。
	VisibleDeptIDs []string `json:"visibleDeptIds"`
	// VisibleDeptNames 可见部门名称（按 ID 顺序对应，便于人读）。
	VisibleDeptNames []string `json:"visibleDeptNames"`
	// Notes 解释性说明（如"All 档优先，机构锚点不生效"）。
	Notes []string `json:"notes"`
	// Permissions 表单选中角色汇总出的菜单/按钮权限码（sys_menu.perms 去重排序）。
	Permissions []string `json:"permissions"`
	// MenuIds 选中角色汇总出的**菜单树节点 ID 集**（含父菜单），供前端
	// <Tree checkable checkedKeys> 勾选渲染（与编辑角色模态框同款菜单树）。
	MenuIds []string `json:"menuIds"`
	// DeptIds 最终可见部门 ID 集，供前端部门树勾选渲染。
	// 与 VisibleDeptIDs 同源：非 All 档 = 可见集；All 档 = 空（前端渲染为"全选"提示）。
	DeptIds []string `json:"deptIds"`
	// AllDepts 是否全库可见（All 档）——前端据此把部门树渲染为全选并加提示。
	AllDepts bool `json:"allDepts"`
	// RoleNames 选中角色名（供展示）。
	RoleNames []string `json:"roleNames"`
	// PostNames 选中岗位名（供展示）。
	PostNames []string `json:"postNames"`
}

// UserAnchorSource 单个锚点来源的解析明细。
type UserAnchorSource struct {
	// Source 来源类型：role / post。
	Source string `json:"source"`
	// SourceName 角色名/岗位名。
	SourceName string `json:"sourceName"`
	Mode       int    `json:"mode"`
	Param      string `json:"param"`
	// ModeLabel 中文模式说明。
	ModeLabel string `json:"modeLabel"`
	// Resolved 是否命中锚点。
	Resolved bool `json:"resolved"`
	// AnchorDeptID / AnchorDeptName 命中锚点部门。
	AnchorDeptID   string `json:"anchorDeptId,omitempty"`
	AnchorDeptName string `json:"anchorDeptName,omitempty"`
	// Reason 未命中原因（中文，可直接展示）。
	Reason string `json:"reason,omitempty"`
	// DeptCount 该锚点展开的子树部门数（命中时）。
	DeptCount int `json:"deptCount,omitempty"`
}

// DataScopeLabel 档位中文说明（与前端 DATA_SCOPE_OPTIONS 一致）。
func DataScopeLabel(scope models.DataScope) string {
	switch scope {
	case models.DataScopeAll:
		return "全部数据"
	case models.DataScopeCustom:
		return "自定义数据"
	case models.DataScopeDept:
		return "本部门数据"
	case models.DataScopeDeptChild:
		return "本部门及子部门数据"
	case models.DataScopeSelf:
		return "仅本人数据"
	default:
		return fmt.Sprintf("未知档位(%d)", scope)
	}
}

// AnchorModeLabel 模式中文说明。
func AnchorModeLabel(mode models.AnchorMode) string {
	switch mode {
	case models.AnchorModeNone:
		return "不参与"
	case models.AnchorModeDept:
		return "指定部门（含下级）"
	case models.AnchorModeExternalOrg:
		return "外部机构（最近）"
	case models.AnchorModeLevel:
		return "指定机构层级（最近）"
	default:
		return fmt.Sprintf("未知模式(%d)", mode)
	}
}

// PreviewUserScope 按表单当前值模拟用户数据范围。
//
// 参数：
//   - deptID 表单选中的部门（可能尚未保存）
//   - roleIDs / postIDs 表单选中的角色/岗位 ID（可能尚未保存）
//
// 返回：数据范围画像（档位 + 锚点明细 + 可见部门），菜单权限由调用方另查。
func PreviewUserScope(
	ctx context.Context,
	db *gorm.DB,
	deptID string,
	roleIDs []string,
	postIDs []string,
) (*UserScopePreview, error) {

	res := &UserScopePreview{
		AnchorSources: []UserAnchorSource{},
		Notes:         []string{},
	}

	// 1) 角色维度档位：MAX(data_scope)，与 getUserMaxDataScope 同口径（无角色→Self）。
	var maxScope *int
	if len(roleIDs) > 0 {
		if err := db.WithContext(ctx).Raw(`
			SELECT MAX(r.data_scope) FROM sys_role r
			 WHERE CAST(r.id AS TEXT) IN ? AND r.status = ? AND r.deleted_at IS NULL
		`, roleIDs, int(models.RoleStatusEnabled)).Scan(&maxScope).Error; err != nil {
			return nil, fmt.Errorf("查询角色数据范围失败: %w", err)
		}
	}
	if maxScope == nil {
		res.DataScope = models.DataScopeSelf
	} else {
		res.DataScope = models.DataScope(*maxScope)
	}
	res.DataScopeLabel = DataScopeLabel(res.DataScope)
	res.All = res.DataScope == models.DataScopeAll || res.DataScope == 0

	// 2) 部门链（预览用：按表单 deptID 查 ancestors，不经过 userID）。
	var ancestors string
	if deptID != "" {
		if err := db.WithContext(ctx).Raw(`
			SELECT COALESCE(ancestors, '') FROM sys_dept
			 WHERE CAST(id AS TEXT) = ? AND deleted_at IS NULL LIMIT 1
		`, deptID).Scan(&ancestors).Error; err != nil {
			return nil, fmt.Errorf("查询部门祖先链失败: %w", err)
		}
	}

	// 3) 汇总锚点来源（表单选中的角色/岗位中 anchor_mode<>0 的）。
	type sourceRow struct {
		ID          string
		Name        string
		AnchorMode  int
		AnchorParam string
	}
	var roleRows, postRows []sourceRow
	if len(roleIDs) > 0 {
		if err := db.WithContext(ctx).Raw(`
			SELECT CAST(id AS TEXT) AS id, role_name AS name,
			       COALESCE(anchor_mode,0) AS anchor_mode, COALESCE(anchor_param,'') AS anchor_param
			  FROM sys_role WHERE CAST(id AS TEXT) IN ? AND status = ? AND deleted_at IS NULL
			   AND COALESCE(anchor_mode,0) <> 0
		`, roleIDs, int(models.RoleStatusEnabled)).Scan(&roleRows).Error; err != nil {
			return nil, fmt.Errorf("查询角色锚点失败: %w", err)
		}
	}
	if len(postIDs) > 0 {
		if err := db.WithContext(ctx).Raw(`
			SELECT CAST(id AS TEXT) AS id, post_name AS name,
			       COALESCE(anchor_mode,0) AS anchor_mode, COALESCE(anchor_param,'') AS anchor_param
			  FROM sys_post WHERE CAST(id AS TEXT) IN ? AND status = ? AND deleted_at IS NULL
			   AND COALESCE(anchor_mode,0) <> 0
		`, postIDs, int(models.PostStatusEnabled)).Scan(&postRows).Error; err != nil {
			return nil, fmt.Errorf("查询岗位锚点失败: %w", err)
		}
	}

	collected := make([]string, 0, 64)
	appendSource := func(kind string, s sourceRow) {
		mode := models.AnchorMode(s.AnchorMode)
		item := UserAnchorSource{
			Source: kind, SourceName: s.Name, Mode: int(mode), Param: s.AnchorParam,
			ModeLabel: AnchorModeLabel(mode),
		}
		anchorID, resolved, reason, err := ResolveAnchorByDept(
			ctx, db, deptID, ancestors, mode, s.AnchorParam,
		)
		if err != nil {
			// 单个来源解析异常不中断整体预览——记入 reason 供用户看到。
			item.Reason = fmt.Sprintf("解析失败：%v", err)
			res.AnchorSources = append(res.AnchorSources, item)
			return
		}
		item.Resolved = resolved
		if !resolved {
			item.Reason = string(reason)
			res.AnchorSources = append(res.AnchorSources, item)
			return
		}
		item.AnchorDeptID = anchorID
		// 锚点部门名（供展示）
		var name string
		_ = db.WithContext(ctx).Raw(`SELECT dept_name FROM sys_dept WHERE CAST(id AS TEXT)=? AND deleted_at IS NULL`, anchorID).Scan(&name).Error
		item.AnchorDeptName = name
		ids, qErr := queryDeptChildIDs(ctx, db, anchorID)
		if qErr != nil {
			item.Reason = fmt.Sprintf("展开子树失败：%v", qErr)
			res.AnchorSources = append(res.AnchorSources, item)
			return
		}
		item.DeptCount = len(ids)
		collected = append(collected, ids...)
		res.AnchorSources = append(res.AnchorSources, item)
	}
	for _, r := range roleRows {
		appendSource("role", r)
	}
	for _, p := range postRows {
		appendSource("post", p)
	}

	// 3.5) 菜单权限 + 角色/岗位名（表单选中项，汇总去重）——All 档也需要展示菜单权限。
	if len(roleIDs) > 0 {
		// 权限码（叶子/按钮等实际授权点，去重排序）。
		if err := db.WithContext(ctx).Raw(`
			SELECT DISTINCT m.perms
			  FROM sys_menu m
			  INNER JOIN sys_role_menu rm ON CAST(rm.menu_id AS TEXT) = CAST(m.id AS TEXT)
			 WHERE CAST(rm.role_id AS TEXT) IN ? AND m.perms IS NOT NULL AND m.perms != ''
			   AND m.status = ?
		`, roleIDs, int(models.MenuStatusNormal)).Scan(&res.Permissions).Error; err != nil {
			return nil, fmt.Errorf("查询角色菜单权限失败: %w", err)
		}
		sort.Strings(res.Permissions)
		if res.Permissions == nil {
			res.Permissions = []string{}
		}
		// 菜单树勾选集：角色直接关联的 menu id。
		// 角色编辑页用 <Tree checkable> 勾选，父菜单勾选时前端已把父节点 id 一并
		// 写入 sys_role_menu（与勾选态一致），故这里直接取关联表即可。
		var menuIDs []string
		if err := db.WithContext(ctx).Raw(`
			SELECT DISTINCT CAST(m.id AS TEXT)
			  FROM sys_menu m
			  INNER JOIN sys_role_menu rm ON CAST(rm.menu_id AS TEXT) = CAST(m.id AS TEXT)
			 WHERE CAST(rm.role_id AS TEXT) IN ? AND m.deleted_at IS NULL
		`, roleIDs).Scan(&menuIDs).Error; err != nil {
			return nil, fmt.Errorf("查询角色菜单 ID 失败: %w", err)
		}
		sort.Strings(menuIDs)
		res.MenuIds = menuIDs
		var names []string
		if err := db.WithContext(ctx).Raw(`
			SELECT role_name FROM sys_role
			 WHERE CAST(id AS TEXT) IN ? AND deleted_at IS NULL ORDER BY role_name
		`, roleIDs).Scan(&names).Error; err == nil {
			res.RoleNames = names
		}
	}
	if len(postIDs) > 0 {
		var names []string
		if err := db.WithContext(ctx).Raw(`
			SELECT post_name FROM sys_post
			 WHERE CAST(id AS TEXT) IN ? AND deleted_at IS NULL ORDER BY post_name
		`, postIDs).Scan(&names).Error; err == nil {
			res.PostNames = names
		}
	}

	if res.All {
		res.Notes = append(res.Notes, "角色数据范围为「全部数据」，机构锚点不参与（All 优先，与运行期一致）。")
		res.AllDepts = true
		res.DeptIds = []string{}
		if res.MenuIds == nil {
			res.MenuIds = []string{}
		}
		return res, nil
	}

	// 4) 角色维度本身产出的部门集，与锚点取并集。
	//    按 MAX 档位模拟（多角色取最窄），与 getUserMaxDataScope 一致；
	//    Dept/DeptChild 以**表单 deptID**为锚点（表单值可能尚未保存，不能查 sys_user）。
	switch res.DataScope {
	case models.DataScopeDept:
		if deptID != "" {
			collected = append(collected, deptID)
		}
	case models.DataScopeDeptChild:
		if deptID != "" {
			ids, qErr := queryDeptChildIDs(ctx, db, deptID)
			if qErr != nil {
				res.Notes = append(res.Notes, "角色维度子树展开失败（已忽略，按锚点结果为准）")
			} else {
				collected = append(collected, ids...)
			}
		}
	case models.DataScopeCustom:
		// Custom 的部门集来自"已保存"的 sys_role_dept（角色勾选树），
		// 表单里只改 roleIds 时沿用库中勾选清单（角色本身的配置不会在用户表单里改）。
		if len(roleIDs) > 0 {
			var ids []string
			if err := db.WithContext(ctx).Raw(`
				SELECT DISTINCT CAST(rd.dept_id AS TEXT)
				  FROM sys_role_dept rd
				 WHERE CAST(rd.role_id AS TEXT) IN ?
			`, roleIDs).Scan(&ids).Error; err == nil {
				collected = append(collected, ids...)
			}
		}
	}

	res.VisibleDeptIDs = normalizeIDs(collected)
	// 部门树勾选集：非 All 档 = 可见部门；All 档留空由前端渲染为"全选"。
	res.DeptIds = res.VisibleDeptIDs
	res.AllDepts = res.All
	if len(res.VisibleDeptIDs) > 0 {
		// 批量查部门名（一次 SQL），按 ID 顺序输出。
		type nameRow struct {
			ID       string `gorm:"column:id"`
			DeptName string `gorm:"column:dept_name"`
		}
		var names []nameRow
		if err := db.WithContext(ctx).Raw(`
			SELECT CAST(id AS TEXT) AS id, dept_name FROM sys_dept
			 WHERE CAST(id AS TEXT) IN ? AND deleted_at IS NULL
		`, res.VisibleDeptIDs).Scan(&names).Error; err == nil {
			nameMap := make(map[string]string, len(names))
			for _, n := range names {
				nameMap[n.ID] = n.DeptName
			}
			res.VisibleDeptNames = make([]string, 0, len(res.VisibleDeptIDs))
			for _, id := range res.VisibleDeptIDs {
				res.VisibleDeptNames = append(res.VisibleDeptNames, nameMap[id])
			}
		}
	} else {
		res.Notes = append(res.Notes, "未解析到任何机构锚点，且角色维度为「仅本人」——保存后该用户将只能看到自己。")
	}

	sort.Strings(res.Notes)
	return res, nil
}
