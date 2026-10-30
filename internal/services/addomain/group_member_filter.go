package addomain

import (
	"context"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// filterNestedGroupMemberDNs 将 memberDNs 按本轮已知 group DN 集合分流：
// kept = 用户/其他 DN（应写入 sys_ad_group_member），filtered = 嵌套组 DN（不写入）。
//
// HYGIENE-04：LDAP 的 member 属性混有组套组的组 DN，原实现对 member 无任何区分，
// 导致组成员关系表混入非用户行、GetMembers total 虚高（生产实锤 8 行）。
// 比较用 strings.EqualFold：LDAP DN 值大小写不敏感，生产存量嵌套 DN 与
// sys_ad_group.group_dn 大小写可能不一致。
//
// knownGroupDNs 为 nil 时全部保留（防御语义，调用方必须显式传入集合）。
func filterNestedGroupMemberDNs(memberDNs []string, knownGroupDNs map[string]struct{}) (kept, filtered []string) {
	kept = make([]string, 0, len(memberDNs))
	if knownGroupDNs == nil {
		kept = append(kept, memberDNs...)
		return kept, nil
	}
	for _, dn := range memberDNs {
		key := strings.ToLower(strings.TrimSpace(dn))
		if _, ok := knownGroupDNs[key]; ok {
			filtered = append(filtered, dn)
			continue
		}
		kept = append(kept, dn)
	}
	return kept, filtered
}

// loadKnownGroupDNSet 每轮一次查询本轮配置下全部组 DN（小写归一）。
// 用 DB 全量而非仅本轮 entries：覆盖嵌套组在本轮 entry 集之外（如 BaseDN 外组）的情况。
func (s *SyncService) loadKnownGroupDNSet(ctx context.Context, configID string) (map[string]struct{}, error) {
	var dns []string
	if err := s.db.WithContext(ctx).Model(&models.ADGroup{}).
		Where("ad_config_id = ? AND deleted_at IS NULL", configID).
		Pluck("group_dn", &dns).Error; err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(dns))
	for _, dn := range dns {
		set[strings.ToLower(strings.TrimSpace(dn))] = struct{}{}
	}
	return set, nil
}
