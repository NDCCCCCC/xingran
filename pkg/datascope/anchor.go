// Package datascope：机构锚点解析（anchor scope）—— v3 设计的核心实现。
//
// 为什么不是新枚举而是独立维度：
//   - DataScope 枚举（1..5）由 getUserMaxDataScope 用 MAX(data_scope) 取数值最大
//     （最窄）档；新增 6 会把 All(1) 用户静默收窄 —— 静默降权事故。
//   - 故锚点作为**与角色维度正交的独立维度**解析，结果与角色维度取并集（宽者胜），
//     保证纯增量、绝不收窄既有用户。
//
// 为什么锚点是「部门子树根」而不是「机构实体」：
//   - sys_dept 没有层级类型字段；is_external_org 只回答"有没有职场"；
//     sys_manager_org 属经理同步域且无种子数据（空表时全员未命中）。
//   - 机构层级只是部门树的一个投影：锚点一律落到某个 dept_id，再用既有
//     queryDeptChildIDs 展开子树 —— 层级与子树天然同源，无需一致性校验。
//
// 关键约束（LOCKED，见 .planning/notes/261118-anchor-scope-contract.md）：
//   - C1：绝不参与 MAX(data_scope)；
//   - C2：与角色维度取并集，All 优先，纯增量；
//   - C3：解析失败 fail-closed，绝不回退到用户 dept_id；
//   - C4：ScopeResult 结构不变（5 字段），各域过滤代码零改动；
//   - C5：子树展开只复用 queryDeptChildIDs，禁止第二套 ancestors LIKE；
//   - C7：uuid/varchar 比较统一 CAST(... AS TEXT)。
package datascope

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// AnchorResolveReason 锚点未命中的原因（供推导预览展示给人看）。
type AnchorResolveReason string

const (
	// AnchorOK 命中。
	AnchorOK AnchorResolveReason = ""
	// AnchorReasonNoDept 用户没有所属部门。
	AnchorReasonNoDept AnchorResolveReason = "该用户未归属任何部门"
	// AnchorReasonNoDeptRow 用户 dept_id 指向的部门不存在（脏数据/已物理删除）。
	AnchorReasonNoDeptRow AnchorResolveReason = "该用户所属部门不存在或已删除"
	// AnchorReasonBadParam 锚点参数非法（空/非数字/超出 1..4/非 UUID）。
	AnchorReasonBadParam AnchorResolveReason = "锚点参数非法"
	// AnchorReasonDeptMissing 显式锚点部门不存在（模式 2）。
	AnchorReasonDeptMissing AnchorResolveReason = "指定的锚点部门不存在或已删除"
	// AnchorReasonLevelMiss 部门链上没有指定层级的机构（模式 5）。
	AnchorReasonLevelMiss AnchorResolveReason = "该用户在部门链中没有指定层级的机构"
	// AnchorReasonExternalMiss 部门链上没有标记为外部机构的节点（模式 3）。
	AnchorReasonExternalMiss AnchorResolveReason = "该用户在部门链中没有标记为外部机构的节点"
	// AnchorReasonBadMode 锚点模式非法（1/4/未知）。
	AnchorReasonBadMode AnchorResolveReason = "锚点模式非法"
)

// AnchorPreviewItem 推导预览中的单个任职人结果。
type AnchorPreviewItem struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Nickname string `json:"nickname,omitempty"`
	Resolved bool   `json:"resolved"`
	// AnchorDeptID 命中时的锚点部门 ID（未命中为空）。
	AnchorDeptID string `json:"anchorDeptId,omitempty"`
	// Reason 未命中原因（中文，可直接展示）。
	Reason string `json:"reason,omitempty"`
}

// AnchorPreviewResult 推导预览结果（角色/岗位详情页展示用）。
type AnchorPreviewResult struct {
	Mode            int                 `json:"mode"`
	Param           string              `json:"param"`
	ResolvedCount   int                 `json:"resolvedCount"`
	UnresolvedCount int                 `json:"unresolvedCount"`
	TotalCount      int                 `json:"totalCount"`
	Items           []AnchorPreviewItem `json:"items"`
}

// ---------------------------------------------------------------------------
// 基础查询
// ---------------------------------------------------------------------------

// queryUserDeptFull 取用户所属部门的 (dept_id, ancestors)。
//
// ancestors 为「根→父」序、逗号分隔、**无首尾逗号**（生产数据惯例）。
// 全部 CAST AS TEXT 保证 PG(uuid/varchar) 与 SQLite 行为一致。
func queryUserDeptFull(ctx context.Context, db *gorm.DB, userID string) (deptID string, ancestors string, ok bool, err error) {
	if userID == "" {
		return "", "", false, nil
	}

	var userDeptID string
	if err := db.WithContext(ctx).Raw(
		"SELECT COALESCE(CAST(dept_id AS TEXT), '') FROM sys_user WHERE CAST(id AS TEXT) = ?",
		userID,
	).Scan(&userDeptID).Error; err != nil {
		return "", "", false, fmt.Errorf("查询用户所属部门失败: %w", err)
	}
	if userDeptID == "" {
		return "", "", false, nil
	}

	var row struct {
		ID        string
		Ancestors string
	}
	if err := db.WithContext(ctx).Raw(
		`SELECT CAST(id AS TEXT) AS id, COALESCE(ancestors, '') AS ancestors
		   FROM sys_dept WHERE CAST(id AS TEXT) = ? AND deleted_at IS NULL LIMIT 1`,
		userDeptID,
	).Scan(&row).Error; err != nil {
		return "", "", false, fmt.Errorf("查询用户部门 ancestors 失败: %w", err)
	}
	if row.ID == "" {
		return "", "", false, nil
	}
	return row.ID, row.Ancestors, true, nil
}

// queryDeptLevel 取部门机构层级；不存在返回 0（= 非机构）。
func queryDeptLevel(ctx context.Context, db *gorm.DB, deptID string) (models.OrgLevel, error) {
	if deptID == "" {
		return models.OrgLevelNone, nil
	}
	var lv int
	if err := db.WithContext(ctx).Raw(
		"SELECT COALESCE(org_level, 0) FROM sys_dept WHERE CAST(id AS TEXT) = ? AND deleted_at IS NULL LIMIT 1",
		deptID,
	).Scan(&lv).Error; err != nil {
		return models.OrgLevelNone, fmt.Errorf("查询部门机构层级失败: %w", err)
	}
	return models.OrgLevel(lv), nil
}

// queryDeptIsExternal 判断部门是否标记为外部机构（is_external_org = 1）。
func queryDeptIsExternal(ctx context.Context, db *gorm.DB, deptID string) (bool, error) {
	if deptID == "" {
		return false, nil
	}
	var v int
	if err := db.WithContext(ctx).Raw(
		"SELECT COALESCE(is_external_org, 0) FROM sys_dept WHERE CAST(id AS TEXT) = ? AND deleted_at IS NULL LIMIT 1",
		deptID,
	).Scan(&v).Error; err != nil {
		return false, fmt.Errorf("查询部门外部机构标记失败: %w", err)
	}
	return v == 1, nil
}

// deptExists 部门是否存在且未软删。
func deptExists(ctx context.Context, db *gorm.DB, deptID string) (bool, error) {
	if deptID == "" {
		return false, nil
	}
	var cnt int64
	if err := db.WithContext(ctx).Model(&models.Department{}).
		Where("CAST(id AS TEXT) = ? AND deleted_at IS NULL", deptID).
		Count(&cnt).Error; err != nil {
		return false, fmt.Errorf("查询部门是否存在失败: %w", err)
	}
	return cnt > 0, nil
}

// splitAncestors 按逗号切分 ancestors，丢弃空串与占位 "0"。
//
// ancestors 形如 "0,id1,id2"（根→父序）；"0" 是历史占位，不是真实部门 ID。
func splitAncestors(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "0" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// walkAncestorsNearest 从「父」向「根」逐级回调，返回第一个满足 match 的部门 ID。
//
// ancestors 是根→父序，**必须反转**才能实现"由近到远"。禁止用 ancestors LIKE
// 求"最近的"节点 —— LIKE 无法表达序关系（契约 C3 的配套约束）。
func walkAncestorsNearest(ancestors []string, match func(deptID string) (bool, error)) (string, bool, error) {
	for i := len(ancestors) - 1; i >= 0; i-- {
		hit, err := match(ancestors[i])
		if err != nil {
			return "", false, err
		}
		if hit {
			return ancestors[i], true, nil
		}
	}
	return "", false, nil
}

// ---------------------------------------------------------------------------
// 锚点解析
// ---------------------------------------------------------------------------

// parseAnchorLevel 解析 anchor_mode=5 的层级参数（只接受 1..4）。
//
// 0（非机构）不可作为 target：否则「支公司管理员」岗位会命中一堆科室节点，
// 语义崩坏。5 号模式只允许指向真实机构层级。
func parseAnchorLevel(param string) (models.OrgLevel, bool) {
	lv, err := strconv.Atoi(strings.TrimSpace(param))
	if err != nil {
		return models.OrgLevelNone, false
	}
	if lv < int(models.OrgLevelBranch) || lv > int(models.OrgLevelSite) {
		return models.OrgLevelNone, false
	}
	return models.OrgLevel(lv), true
}

// ResolveAnchor 解析登录人按指定锚点模式得到的锚点部门 ID。
//
// 返回 (anchorDeptID, resolved, reason, err)：
//   - resolved=false 且 err=nil 是**正常业务结果**（未命中），reason 说明原因；
//   - err != nil 才是数据库/系统异常。
//
// 模式语义（契约 §三）：
//
//	AnchorModeNone(0)        → 不解析，resolved=false
//	AnchorModeDept(2)        → 锚点 = anchor_param（允许跨机构授权，不要求是用户祖先）
//	AnchorModeExternalOrg(3) → 由近到远找第一个 is_external_org=1 的节点
//	AnchorModeLevel(5)       → 由近到远找第一个 org_level == anchor_param 的节点
//
// **绝不回退到用户 dept_id**：回退会造成"以为管了全中支、实际只看到自己部门"的
// 静默错范围事故（契约 C3）。
func ResolveAnchor(
	ctx context.Context,
	db *gorm.DB,
	userID string,
	mode models.AnchorMode,
	param string,
) (anchorDeptID string, resolved bool, reason AnchorResolveReason, err error) {

	if mode == models.AnchorModeNone {
		return "", false, AnchorOK, nil
	}
	if !models.IsValidAnchorMode(mode) {
		return "", false, AnchorReasonBadMode, nil
	}

	switch mode {
	case models.AnchorModeDept:
		// 显式锚点：与用户所属部门无关，支持省公司派驻、跨机构兼任。
		deptID := strings.TrimSpace(param)
		if deptID == "" {
			return "", false, AnchorReasonBadParam, nil
		}
		exists, qErr := deptExists(ctx, db, deptID)
		if qErr != nil {
			return "", false, AnchorOK, qErr
		}
		if !exists {
			return "", false, AnchorReasonDeptMissing, nil
		}
		return deptID, true, AnchorOK, nil
	}

	// 其余模式都依赖「用户自身的部门链」。
	deptID, ancestors, ok, qErr := queryUserDeptFull(ctx, db, userID)
	if qErr != nil {
		return "", false, AnchorOK, qErr
	}
	if !ok {
		return "", false, AnchorReasonNoDept, nil
	}
	return resolveAnchorByChain(ctx, db, deptID, ancestors, mode, param)
}

// ResolveAnchorByDept 按**指定部门**解析锚点（不做 userID→dept 的查询）。
//
// 用途：编辑用户时的「权限预览」——用户在表单里**尚未保存**的新 deptId 也要能
// 预览其机构范围。与 ResolveAnchor 共用同一条解析链（resolveAnchorByChain），
// 保证"预览结果"与"保存后运行期结果"逐字一致，绝不出现两套算法漂移。
//
// deptID 必须是 sys_dept 中未软删的节点；ancestors 由调用方从该部门读取后传入
// （预览接口会在 service 层查出完整 (deptID, ancestors)）。
func ResolveAnchorByDept(
	ctx context.Context,
	db *gorm.DB,
	deptID string,
	ancestors string,
	mode models.AnchorMode,
	param string,
) (anchorDeptID string, resolved bool, reason AnchorResolveReason, err error) {
	if mode == models.AnchorModeNone {
		return "", false, AnchorOK, nil
	}
	if !models.IsValidAnchorMode(mode) {
		return "", false, AnchorReasonBadMode, nil
	}
	if mode == models.AnchorModeDept {
		// 显式锚点与部门链无关，param 即锚点。
		pid := strings.TrimSpace(param)
		if pid == "" {
			return "", false, AnchorReasonBadParam, nil
		}
		exists, e := deptExists(ctx, db, pid)
		if e != nil {
			return "", false, AnchorOK, e
		}
		if !exists {
			return "", false, AnchorReasonDeptMissing, nil
		}
		return pid, true, AnchorOK, nil
	}
	if deptID == "" {
		return "", false, AnchorReasonNoDept, nil
	}
	return resolveAnchorByChain(ctx, db, deptID, ancestors, mode, param)
}

// resolveAnchorByChain ResolveAnchor 与 ResolveAnchorByDept 共用的解析内核。
//
// 入参 deptID/ancestors 已经确定（无论来自 userID 查询还是表单预览），
// 之后按 mode 沿链由近到远定位锚点。两种调用方共用此函数 = 单一真实源。
func resolveAnchorByChain(
	ctx context.Context,
	db *gorm.DB,
	deptID string,
	ancestors string,
	mode models.AnchorMode,
	param string,
) (anchorDeptID string, resolved bool, reason AnchorResolveReason, err error) {
	anc := splitAncestors(ancestors)

	switch mode {
	case models.AnchorModeExternalOrg:
		// 先看自身，再由近到远看祖先。
		isExt, e := queryDeptIsExternal(ctx, db, deptID)
		if e != nil {
			return "", false, AnchorOK, e
		}
		if isExt {
			return deptID, true, AnchorOK, nil
		}
		hit, found, wErr := walkAncestorsNearest(anc, func(id string) (bool, error) {
			return queryDeptIsExternal(ctx, db, id)
		})
		if wErr != nil {
			return "", false, AnchorOK, wErr
		}
		if !found {
			return "", false, AnchorReasonExternalMiss, nil
		}
		return hit, true, AnchorOK, nil

	case models.AnchorModeLevel:
		level, okLv := parseAnchorLevel(param)
		if !okLv {
			return "", false, AnchorReasonBadParam, nil
		}
		selfLevel, e := queryDeptLevel(ctx, db, deptID)
		if e != nil {
			return "", false, AnchorOK, e
		}
		if selfLevel == level {
			return deptID, true, AnchorOK, nil
		}
		hit, found, wErr := walkAncestorsNearest(anc, func(id string) (bool, error) {
			got, gErr := queryDeptLevel(ctx, db, id)
			return got == level, gErr
		})
		if wErr != nil {
			return "", false, AnchorOK, wErr
		}
		if !found {
			return "", false, AnchorReasonLevelMiss, nil
		}
		return hit, true, AnchorOK, nil
	}

	return "", false, AnchorReasonBadMode, nil
}

// ---------------------------------------------------------------------------
// 配置期校验（防静默第 1 道防线）
// ---------------------------------------------------------------------------

// ValidateAnchor 校验 anchor_mode/anchor_param 的合法性（保存角色/岗位时调用）。
//
// 把错误挡在**配置时**而不是运行期：锚点算不出来时运行期只会 fail-closed，
// 用户看到空数据却不知道原因，比配置期直接拒绝危险得多（契约 §五）。
func ValidateAnchor(ctx context.Context, db *gorm.DB, mode int, param string) error {
	m := models.AnchorMode(mode)
	if !models.IsValidAnchorMode(m) {
		return fmt.Errorf("数据权限锚点模式非法：%d（仅支持 0=不参与 / 2=指定部门 / 3=外部机构 / 5=机构层级）", mode)
	}

	switch m {
	case models.AnchorModeNone:
		// 不参与，忽略 param。
		return nil

	case models.AnchorModeDept:
		deptID := strings.TrimSpace(param)
		if deptID == "" {
			return fmt.Errorf("锚点模式为「指定部门」时必须选择锚点部门")
		}
		// 只校验"存在"，不额外做 UUID 形状校验。理由（2026-11-17 lead 裁定）：
		//   - 生产 PG 的 sys_dept.id 是原生 uuid 类型，非 UUID 串在 SQL 层即被拒绝；
		//   - 真实数据 ID 均由 uuid.New() 生成，天然满足形状；
		//   - 前端 isAnchorDeptParam 已有 UUID 正则，配置面不会有非 UUID 提交；
		//   - SQLite 仅用于嵌入/测试，为它增加形状校验会误伤全部短 ID 夹具而收益极低。
		// "部门必须存在"才是真正的防线（防止指向不存在的锚点导致运行期静默空集）。
		exists, err := deptExists(ctx, db, deptID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("锚点部门不存在或已删除：%s", deptID)
		}
		return nil

	case models.AnchorModeExternalOrg:
		if strings.TrimSpace(param) != "" {
			return fmt.Errorf("锚点模式为「外部机构」时不需要参数，请清空锚点参数")
		}
		var cnt int64
		if err := db.WithContext(ctx).Model(&models.Department{}).
			Where("is_external_org = 1 AND deleted_at IS NULL").
			Count(&cnt).Error; err != nil {
			return fmt.Errorf("统计外部机构数量失败: %w", err)
		}
		if cnt == 0 {
			return fmt.Errorf("组织架构中未标记任何「外部机构」，该锚点模式无法解析；请先在部门上设置 isExternalOrg=是")
		}
		return nil

	case models.AnchorModeLevel:
		level, ok := parseAnchorLevel(param)
		if !ok {
			return fmt.Errorf("锚点模式为「机构层级」时参数必须是 1(分公司) / 2(中支) / 3(支公司) / 4(网点)：%q", param)
		}
		var cnt int64
		if err := db.WithContext(ctx).Model(&models.Department{}).
			Where("org_level = ? AND deleted_at IS NULL", int(level)).
			Count(&cnt).Error; err != nil {
			return fmt.Errorf("统计机构层级数量失败: %w", err)
		}
		if cnt == 0 {
			return fmt.Errorf("组织架构中不存在层级为 %d 的机构节点，该锚点模式无法解析；请先为相关部门标注机构层级", int(level))
		}
		return nil
	}

	return fmt.Errorf("数据权限锚点模式非法：%d", mode)
}

// ---------------------------------------------------------------------------
// 汇总当前用户的全部机构维度授权
// ---------------------------------------------------------------------------

// AnchorGrant 一条锚点授权来源（角色或岗位）。
type AnchorGrant struct {
	// Source 来源标识：role / post。
	Source string
	// SourceID 角色或岗位 ID。
	SourceID string
	// SourceName 角色名或岗位名（诊断与日志用）。
	SourceName string
	Mode       models.AnchorMode
	Param      string
}

// queryAnchorGrants 取用户所有**启用**角色/岗位中声明了锚点的来源。
//
// 与 getUserMaxDataScope 的过滤口径保持一致：只看 status=0（正常/启用）。
// 停用的角色/岗位不产生任何权限，包括机构维度。
func queryAnchorGrants(ctx context.Context, db *gorm.DB, userID string) ([]AnchorGrant, error) {
	if userID == "" {
		return nil, nil
	}

	var grants []AnchorGrant

	// 角色来源
	var roleRows []struct {
		ID          string
		RoleName    string
		AnchorMode  int
		AnchorParam string
	}
	if err := db.WithContext(ctx).Raw(`
		SELECT CAST(r.id AS TEXT) AS id, r.role_name, COALESCE(r.anchor_mode, 0) AS anchor_mode,
		       COALESCE(r.anchor_param, '') AS anchor_param
		  FROM sys_user_role ur
		  INNER JOIN sys_role r ON CAST(r.id AS TEXT) = CAST(ur.role_id AS TEXT)
		 WHERE CAST(ur.user_id AS TEXT) = ? AND r.status = ? AND COALESCE(r.anchor_mode, 0) <> 0
		   AND r.deleted_at IS NULL
	`, userID, int(models.RoleStatusEnabled)).Scan(&roleRows).Error; err != nil {
		return nil, fmt.Errorf("查询角色锚点来源失败: %w", err)
	}
	for _, r := range roleRows {
		grants = append(grants, AnchorGrant{
			Source: "role", SourceID: r.ID, SourceName: r.RoleName,
			Mode: models.AnchorMode(r.AnchorMode), Param: r.AnchorParam,
		})
	}

	// 岗位来源
	var postRows []struct {
		ID          string
		PostName    string
		AnchorMode  int
		AnchorParam string
	}
	if err := db.WithContext(ctx).Raw(`
		SELECT CAST(p.id AS TEXT) AS id, p.post_name, COALESCE(p.anchor_mode, 0) AS anchor_mode,
		       COALESCE(p.anchor_param, '') AS anchor_param
		  FROM sys_user_post up
		  INNER JOIN sys_post p ON CAST(p.id AS TEXT) = CAST(up.post_id AS TEXT)
		 WHERE CAST(up.user_id AS TEXT) = ? AND p.status = ? AND COALESCE(p.anchor_mode, 0) <> 0
		   AND p.deleted_at IS NULL
	`, userID, int(models.PostStatusEnabled)).Scan(&postRows).Error; err != nil {
		// sys_user_post 在极端老库可能缺失 —— 非致命：退化为"无岗位来源"。
		//
		// ⚠ 必须显式告警：这条分支不返回 error，中间件的 `anchorErr != nil` 告警
		// 永远不会触发。若静默吞掉，生产上"岗位维度授权全部失效"将没有任何线索，
		// 用户只会觉得"配了岗位但看不到数据"——这是本功能最难排查的失败模式
		// （verifier 2026-11-17 的 C-3 发现）。
		if !isMissingTableError(err) {
			return nil, fmt.Errorf("查询岗位锚点来源失败: %w", err)
		}
		applogger.Warnf(
			"[机构锚点] sys_user_post/sys_post 不可查询（user_id=%s），岗位维度授权本次全部跳过：%v",
			userID, err,
		)
		postRows = nil
	}
	for _, p := range postRows {
		grants = append(grants, AnchorGrant{
			Source: "post", SourceID: p.ID, SourceName: p.PostName,
			Mode: models.AnchorMode(p.AnchorMode), Param: p.AnchorParam,
		})
	}

	return grants, nil
}

// isMissingTableError 判断是否"表不存在"类错误（用于 sys_user_post 缺表的降级）。
func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "undefined table")
}

// ResolveAnchorGrants 汇总当前用户全部机构维度授权并展开为可见部门 ID 集。
//
// 返回的 ID 集已 normalizeIDs（去重+去空+排序），可直接参与 Merge。
// 无任何锚点来源 / 全部未命中 → 返回空切片（调用方按"无锚点"处理，不影响角色维度）。
//
// 单个来源解析失败（例如脏参数）**不中断整体**：跳过该来源并继续，
// 因为契约 C2 要求"新增维度绝不收窄"——若因一个脏来源整体失败，反而会削弱其它来源。
func ResolveAnchorGrants(ctx context.Context, db *gorm.DB, userID string) ([]string, error) {
	grants, err := queryAnchorGrants(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if len(grants) == 0 {
		return nil, nil
	}

	collected := make([]string, 0, 32)
	for _, g := range grants {
		anchorID, resolved, _, rErr := ResolveAnchor(ctx, db, userID, g.Mode, g.Param)
		if rErr != nil {
			// 与函数注释一致：单个来源失败不中断整体（否则一个脏来源会让
			// 其它正常授权全部失效）。跳过并告警，保留其它来源的可见集。
			applogger.Warnf(
				"[机构锚点] 来源解析失败，已跳过（user_id=%s, source=%s/%s, mode=%d, param=%q）: %v",
				userID, g.Source, g.SourceID, g.Mode, g.Param, rErr,
			)
			continue
		}
		if !resolved || anchorID == "" {
			continue
		}
		ids, qErr := queryDeptChildIDs(ctx, db, anchorID)
		if qErr != nil {
			applogger.Warnf(
				"[机构锚点] 子树展开失败，已跳过（user_id=%s, source=%s/%s, anchor=%s）: %v",
				userID, g.Source, g.SourceID, anchorID, qErr,
			)
			continue
		}
		collected = append(collected, ids...)
	}

	return normalizeIDs(collected), nil
}

// ---------------------------------------------------------------------------
// 合并（唯一的不变量，见契约 §三 3.4）
// ---------------------------------------------------------------------------

// Merge 把机构维度并入角色维度。纯函数、无 IO、无 DB。
//
// 规则（严格按序，不可改动顺序）：
//  1. roleScope.All → 原样返回（All 优先；超管/全局角色零回归）。
//  2. roleScope.FailClosed 且无锚点 → 原样返回（异常枚举仍 fail-closed）。
//  3. 并集非空 → DeptIDs=并集、Hook=None、FailClosed=false、All=false
//     （落此分支时 Self 也被机构维度覆盖：显式授予的更大范围胜出）。
//  4. 并集空 且 Hook==HookSelfUser → 保留 Self（仅本人）。
//  5. 其余（并集空且无 Hook）→ FailClosed。
//
// 注意：**绝不复用 MAX 取窄**。并集保证新增维度只会放宽，不会收窄。
func Merge(roleScope ScopeResult, anchorDeptIDs []string) ScopeResult {
	// 1) All 优先
	if roleScope.All {
		return roleScope
	}

	// 2) 异常枚举且无锚点 → 保持 fail-closed
	if roleScope.FailClosed && len(anchorDeptIDs) == 0 {
		return roleScope
	}

	// 3) 并集
	eff := normalizeIDs(append(append([]string(nil), roleScope.DeptIDs...), anchorDeptIDs...))
	if len(eff) > 0 {
		return ScopeResult{
			All:          false,
			DeptIDs:      eff,
			ResourceHook: HookNone,
			FailClosed:   false,
			UserID:       roleScope.UserID,
		}
	}

	// 4) 仅本人
	if roleScope.ResourceHook == HookSelfUser {
		return roleScope
	}

	// 5) fail-closed 兜底
	return FailClosedScope(roleScope.UserID)
}

// ---------------------------------------------------------------------------
// 推导预览（防静默第 2 道防线）
// ---------------------------------------------------------------------------

// PreviewAnchor 计算「某个锚点配置下，这一批用户能否解析到锚点」。
//
// 只解析锚点、**不展开子树**，避免 N 次子树查询（预览页通常一次看几十人）。
// userIDs 由调用方（角色/岗位 service）查任职关系后传入。
func PreviewAnchor(
	ctx context.Context,
	db *gorm.DB,
	mode models.AnchorMode,
	param string,
	userIDs []string,
	usernames map[string]string,
	nicknames map[string]string,
) (*AnchorPreviewResult, error) {

	res := &AnchorPreviewResult{
		Mode:       int(mode),
		Param:      param,
		TotalCount: len(userIDs),
		Items:      make([]AnchorPreviewItem, 0, len(userIDs)),
	}

	for _, uid := range userIDs {
		anchorID, resolved, reason, err := ResolveAnchor(ctx, db, uid, mode, param)
		if err != nil {
			return nil, err
		}

		item := AnchorPreviewItem{
			UserID:   uid,
			Username: usernames[uid],
			Nickname: nicknames[uid],
			Resolved: resolved,
			Reason:   string(reason),
		}
		if resolved {
			item.AnchorDeptID = anchorID
			res.ResolvedCount++
		} else {
			if item.Reason == "" {
				item.Reason = "未能解析到锚点"
			}
			res.UnresolvedCount++
		}
		res.Items = append(res.Items, item)
	}

	return res, nil
}
