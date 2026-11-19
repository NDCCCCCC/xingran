package portwrite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Phase 127 前置（PWX-04）：端口写审计查询 API 的 service 层。
//
// 纯读查询：分页 + 过滤（deviceId/portId/operator/action/status/时间范围），
// 返回 sys_port_write_audit 行集（含 before/after JSON 与设备原始响应）。
//
// v1.57 可读性优化：审计页不再展示 deviceId/portId UUID，改为设备名称 + 接口名。
// 名称解析采用「快照优先、实时兜底」两级：
//  1. 行内快照（device_name_snapshot 等，migration 254 起写入）—— 审计不可变事实，
//     设备改名/删除、端口描述变更后仍显示"操作当时"的名称；
//  2. 快照为空（migration 254 之前的历史行，或写审计时端口采集行缺失）——
//     实时查 sys_network_device / sys_device_port_status 补名（设备用 Unscoped）。

// ErrAuditPortOptionDeviceRequired 端口选择器缺少 deviceId（接口名跨设备重名，
// 禁止全局裸选）。
var ErrAuditPortOptionDeviceRequired = fmt.Errorf("portwrite: 端口选项查询必须指定 deviceId")

// AuditListRequest 审计查询请求（Phase 127 对齐 webfe 预期契约）。
type AuditListRequest struct {
	DeviceID string `json:"deviceId,omitempty"`
	PortID   string `json:"portId,omitempty"`
	Operator string `json:"operator,omitempty"`
	Action   string `json:"action,omitempty"`
	// Status 过滤：succeeded | failed（可空 = 全部）
	Status string `json:"status,omitempty"`
	// BatchID Phase 122 收尾：批次精确对账过滤（migration 232；历史行为 NULL 不命中）
	BatchID string `json:"batchId,omitempty"`
	// DeviceName / InterfaceName v1.57：按名称模糊过滤（审计页选择器的"按名称搜索"）。
	// 用子查询而非 JOIN，避免一对多放大行数；设备子查询不过滤 deleted_at ——
	// 已删设备的审计行仍需可按名称检索。与 DeviceID/PortID 同时给出时取交集。
	DeviceName    string `json:"deviceName,omitempty"`
	InterfaceName string `json:"interfaceName,omitempty"`
	// StartTime/EndTime RFC3339 或 "2006-01-02 15:04:05"（可空）
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	Current   int        `json:"current,omitempty"`  // 默认 1
	PageSize  int        `json:"pageSize,omitempty"` // 默认 20，上限 100
}

// AuditRow 审计行视图：内嵌原始审计行 + 可读名称。
//
// 内嵌 models.PortWriteAudit 保证 id/deviceId/portId/action/beforeValue/... 契约不变
// （既有前端与 BulkWriteDrawer 断线对账零改动）；新增字段仅做展示增强。
type AuditRow struct {
	models.PortWriteAudit
	// DeviceName 设备名称；空串 = 快照与实时表都查不到。
	DeviceName string `json:"deviceName"`
	// DeviceIP 设备管理 IP（区分重名设备）。
	DeviceIP string `json:"deviceIp,omitempty"`
	// InterfaceName 接口名（GE0/0/1）；空串 = 端口采集行缺失且无快照。
	InterfaceName string `json:"interfaceName"`
	// PortDescription 端口描述。
	PortDescription string `json:"portDescription,omitempty"`
	// NameFromSnapshot 名称是否完整来自审计行快照（true = 操作当时的名称；
	// false = 至少一侧为实时补名，主数据改名后展示的是当前名称）。前端仅作提示。
	NameFromSnapshot bool `json:"nameFromSnapshot"`
}

// AuditListResult 审计查询分页响应（项目惯例 list/total/current/pageSize）。
type AuditListResult struct {
	List     []AuditRow `json:"list"`
	Total    int64      `json:"total"`
	Current  int        `json:"current"`
	PageSize int        `json:"pageSize"`
}

// AuditDeviceOption 设备选择器选项（audit/device-options）。
type AuditDeviceOption struct {
	Value     string `json:"value"` // deviceId
	Label     string `json:"label"` // 设备名称
	IPAddress string `json:"ipAddress,omitempty"`
}

// AuditPortOption 端口选择器选项（audit/port-options）。
type AuditPortOption struct {
	Value         string `json:"value"` // portId
	Label         string `json:"label"` // 接口名
	Description   string `json:"description,omitempty"`
	InterfaceName string `json:"interfaceName"` // 与 Label 同值，便于前端复用字段名
}

// AuditOptionRequest 选择器查询请求（设备/端口共用）。
type AuditOptionRequest struct {
	// DeviceID 端口选择器必填（接口名跨设备重复，禁止全局裸选）。
	DeviceID string `json:"deviceId,omitempty"`
	// Keyword 名称模糊搜索（设备名/IP、接口名/描述）。
	Keyword  string `json:"keyword,omitempty"`
	Current  int    `json:"current,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// AuditDeviceOptionResult 设备选择器分页响应。
type AuditDeviceOptionResult struct {
	List     []AuditDeviceOption `json:"list"`
	Total    int64               `json:"total"`
	Current  int                 `json:"current"`
	PageSize int                 `json:"pageSize"`
}

// AuditPortOptionResult 端口选择器分页响应。
type AuditPortOptionResult struct {
	List     []AuditPortOption `json:"list"`
	Total    int64             `json:"total"`
	Current  int               `json:"current"`
	PageSize int               `json:"pageSize"`
}

// normalizeAuditQuery 入参兜底（分页默认值 + 上限）。
func normalizeAuditQuery(req AuditListRequest) AuditListRequest {
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	return req
}

// normalizeOptionQuery 选择器分页兜底（默认 20，上限 100）。
func normalizeOptionQuery(req AuditOptionRequest) AuditOptionRequest {
	if req.Current < 1 {
		req.Current = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	return req
}

// escapeLike 转义 LIKE 通配符，避免用户输入的 % / _ 被当成通配。
//
// 调用方统一拼 `LIKE ? ESCAPE '\'`：PG 默认反斜杠转义、SQLite 无默认转义符，
// 显式 ESCAPE 让双方言行为一致。
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// ListAudit 分页查询端口写审计（Phase 127 前置 + v1.57 名称可读性）。
//
// 纯 GORM chainable 查询；等值过滤 + 名称模糊 + 时间范围，按 created_at 倒序。
// 名称补全走「快照优先 + 实时兜底」，每页固定 3 条 SQL（count + 主查询 +
// 设备/端口各一次批量补名），无 N+1。
func (s *portWriteServiceImpl) ListAudit(ctx context.Context, req AuditListRequest) (*AuditListResult, error) {
	req = normalizeAuditQuery(req)

	q := s.db.WithContext(ctx).Model(&models.PortWriteAudit{})
	if req.DeviceID != "" {
		q = q.Where("device_id = ?", req.DeviceID)
	}
	if req.PortID != "" {
		q = q.Where("port_id = ?", req.PortID)
	}
	if req.DeviceName != "" {
		q = q.Where("device_id IN (?)",
			s.db.WithContext(ctx).Unscoped().
				Model(&models.NetworkDevice{}).
				Select("id").
				Where(`device_name LIKE ? ESCAPE '\'`, "%"+escapeLike(req.DeviceName)+"%"))
	}
	if req.InterfaceName != "" {
		q = q.Where("port_id IN (?)",
			s.db.WithContext(ctx).
				Model(&models.DevicePortStatus{}).
				Select("id").
				Where(`interface_name LIKE ? ESCAPE '\'`, "%"+escapeLike(req.InterfaceName)+"%"))
	}
	if req.Operator != "" {
		q = q.Where("operator = ?", req.Operator)
	}
	if req.Action != "" {
		q = q.Where("action = ?", req.Action)
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if req.BatchID != "" {
		q = q.Where("batch_id = ?", req.BatchID)
	}
	if req.StartTime != nil {
		q = q.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		q = q.Where("created_at <= ?", *req.EndTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count audit: %w", err)
	}

	rows := make([]models.PortWriteAudit, 0, req.PageSize)
	if err := q.Order("created_at DESC").
		Offset((req.Current - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}

	list := s.enrichAuditRows(ctx, rows)
	return &AuditListResult{List: list, Total: total, Current: req.Current, PageSize: req.PageSize}, nil
}

// enrichAuditRows 为审计行补设备/端口可读名称（快照优先、实时兜底）。
//
// 只对缺失快照的行发补名查询（两表各一次 IN 查询），快照完整的行零额外开销。
func (s *portWriteServiceImpl) enrichAuditRows(ctx context.Context, rows []models.PortWriteAudit) []AuditRow {
	out := make([]AuditRow, 0, len(rows))
	if len(rows) == 0 {
		return out
	}

	needDevice := make(map[string]struct{}, len(rows))
	needPort := make(map[string]struct{}, len(rows))
	for i := range rows {
		if rows[i].DeviceNameSnapshot == "" || rows[i].DeviceIPSnapshot == "" {
			needDevice[rows[i].DeviceID] = struct{}{}
		}
		if rows[i].InterfaceNameSnapshot == "" || rows[i].PortDescriptionSnapshot == "" {
			needPort[rows[i].PortID] = struct{}{}
		}
	}

	type devInfo struct {
		ID         string
		DeviceName string
		IPAddress  string
	}
	devMap := make(map[string]devInfo, len(needDevice))
	if len(needDevice) > 0 {
		ids := make([]string, 0, len(needDevice))
		for id := range needDevice {
			ids = append(ids, id)
		}
		var devs []devInfo
		// Unscoped：设备软删除后审计仍应显示名称（审计真相源不随主数据消失）
		if err := s.db.WithContext(ctx).Unscoped().
			Model(&models.NetworkDevice{}).
			Select("id, device_name, ip_address").
			Where("id IN ?", ids).
			Scan(&devs).Error; err == nil {
			for _, d := range devs {
				devMap[d.ID] = d
			}
		}
	}

	type portInfo struct {
		ID            string
		InterfaceName string
		Description   string
	}
	portMap := make(map[string]portInfo, len(needPort))
	if len(needPort) > 0 {
		ids := make([]string, 0, len(needPort))
		for id := range needPort {
			ids = append(ids, id)
		}
		var ports []portInfo
		if err := s.db.WithContext(ctx).
			Model(&models.DevicePortStatus{}).
			Select("id, interface_name, description").
			Where("id IN ?", ids).
			Scan(&ports).Error; err == nil {
			for _, p := range ports {
				portMap[p.ID] = p
			}
		}
	}

	for i := range rows {
		row := rows[i]
		view := AuditRow{PortWriteAudit: row}
		// 快照优先（写入时刻的名称 = 审计事实）
		view.DeviceName = row.DeviceNameSnapshot
		view.DeviceIP = row.DeviceIPSnapshot
		view.InterfaceName = row.InterfaceNameSnapshot
		view.PortDescription = row.PortDescriptionSnapshot

		if d, ok := devMap[row.DeviceID]; ok {
			if view.DeviceName == "" {
				view.DeviceName = d.DeviceName
			}
			if view.DeviceIP == "" {
				view.DeviceIP = d.IPAddress
			}
		}
		if p, ok := portMap[row.PortID]; ok {
			if view.InterfaceName == "" {
				view.InterfaceName = p.InterfaceName
			}
			if view.PortDescription == "" {
				view.PortDescription = p.Description
			}
		}

		view.NameFromSnapshot = row.DeviceNameSnapshot != "" && row.InterfaceNameSnapshot != ""
		out = append(out, view)
	}
	return out
}

// ListDeviceOptions 审计页设备选择器（audit/device-options）。
//
// 只返回 id/名称/IP 三个展示必要字段：审计页仅需选择设备，不应顺带获得
// 设备详情读取面。不过滤软删除设备 —— 历史审计行可能指向已删设备。
func (s *portWriteServiceImpl) ListDeviceOptions(ctx context.Context, req AuditOptionRequest) (*AuditDeviceOptionResult, error) {
	req = normalizeOptionQuery(req)

	q := s.db.WithContext(ctx).Unscoped().Model(&models.NetworkDevice{})
	if req.Keyword != "" {
		kw := "%" + escapeLike(req.Keyword) + "%"
		q = q.Where(`device_name LIKE ? ESCAPE '\' OR ip_address LIKE ? ESCAPE '\'`, kw, kw)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count device options: %w", err)
	}

	var rows []struct {
		ID         string
		DeviceName string
		IPAddress  string
	}
	if err := q.Select("id, device_name, ip_address").
		Order("device_name ASC").
		Offset((req.Current - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query device options: %w", err)
	}

	list := make([]AuditDeviceOption, 0, len(rows))
	for _, r := range rows {
		list = append(list, AuditDeviceOption{Value: r.ID, Label: r.DeviceName, IPAddress: r.IPAddress})
	}
	return &AuditDeviceOptionResult{List: list, Total: total, Current: req.Current, PageSize: req.PageSize}, nil
}

// ListPortOptions 审计页端口选择器（audit/port-options）。
//
// DeviceID 必填：接口名（GE0/0/1）跨设备大量重名，全局裸选必然选错端口。
// 返回 id/接口名/描述，供前端展示"GE0/0/1（上行到核心）"这类可辨识标签。
func (s *portWriteServiceImpl) ListPortOptions(ctx context.Context, req AuditOptionRequest) (*AuditPortOptionResult, error) {
	req = normalizeOptionQuery(req)
	if req.DeviceID == "" {
		return nil, ErrAuditPortOptionDeviceRequired
	}

	q := s.db.WithContext(ctx).
		Model(&models.DevicePortStatus{}).
		Where("device_id = ?", req.DeviceID)
	if req.Keyword != "" {
		kw := "%" + escapeLike(req.Keyword) + "%"
		q = q.Where(`interface_name LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\'`, kw, kw)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count port options: %w", err)
	}

	var rows []struct {
		ID            string
		InterfaceName string
		Description   string
	}
	if err := q.Select("id, interface_name, description").
		Order("interface_name ASC").
		Offset((req.Current - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query port options: %w", err)
	}

	list := make([]AuditPortOption, 0, len(rows))
	for _, r := range rows {
		list = append(list, AuditPortOption{
			Value:         r.ID,
			Label:         r.InterfaceName,
			InterfaceName: r.InterfaceName,
			Description:   r.Description,
		})
	}
	return &AuditPortOptionResult{List: list, Total: total, Current: req.Current, PageSize: req.PageSize}, nil
}
