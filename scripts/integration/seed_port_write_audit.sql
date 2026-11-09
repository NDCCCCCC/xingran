-- ============================================================================
-- v1.51 集成验证窗口 · 审计页测试数据 seed（integration checklist §4）
-- 目标表：sys_port_write_audit（单数表名，Phase 52 D-01 锁定）
-- 方言：SQLite / PostgreSQL 通用（固定 UUID 主键，无方言函数）
-- 幂等：先删固定清单再插，可重复执行
-- 用法：sqlite3 <sqlite-db-path> < seed_port_write_audit.sql
--      （PG: psql -f；PG 下 jsonb 列接受 JSON 文本自动转换）
-- ============================================================================

-- 幂等清理（只动本 seed 的固定行，不碰真实审计数据）
DELETE FROM sys_port_write_audit WHERE id IN (
  'a1e2c300-0000-4000-8000-000000000001',
  'a1e2c300-0000-4000-8000-000000000002',
  'a1e2c300-0000-4000-8000-000000000003',
  'a1e2c300-0000-4000-8000-000000000004',
  'a1e2c300-0000-4000-8000-000000000005',
  'a1e2c300-0000-4000-8000-000000000006',
  'a1e2c300-0000-4000-8000-000000000007',
  'a1e2c300-0000-4000-8000-000000000008',
  'a1e2c300-0000-4000-8000-000000000009',
  'a1e2c300-0000-4000-8000-000000000010'
);

-- 设备/端口占位 UUID（前端联调用，非真实外键——表无外键约束）
-- dev-1111... = 汇聚交换机-01；dev-2222... = 接入交换机-02
INSERT INTO sys_port_write_audit
  (id, device_id, port_id, action, before_value, after_value,
   command_sent, device_response, status, failure_reason, operator, created_at)
VALUES
-- ---- succeeded × 5：覆盖 shutdown / undo_shutdown / description / dot1x_enable / access_vlan ----
('a1e2c300-0000-4000-8000-000000000001',
 'de111111-0000-4000-8000-000000000001', 'po111111-0000-4000-8000-000000000001',
 'shutdown',
 '{"adminStatus":"up"}', '{"adminStatus":"down"}',
 'interface GigabitEthernet0/0/1\nshutdown',
 '',
 'succeeded', NULL, 'admin', datetime('now','-1 day')),
('a1e2c300-0000-4000-8000-000000000002',
 'de111111-0000-4000-8000-000000000001', 'po111111-0000-4000-8000-000000000002',
 'undo_shutdown',
 '{"adminStatus":"down"}', '{"adminStatus":"up"}',
 'interface GigabitEthernet0/0/2\nundo shutdown',
 '',
 'succeeded', NULL, 'admin', datetime('now','-1 day')),
('a1e2c300-0000-4000-8000-000000000003',
 'de111111-0000-4000-8000-000000000001', 'po111111-0000-4000-8000-000000000003',
 'description',
 '{"description":""}', '{"description":"TO-ServerRoom-A"}',
 'interface GigabitEthernet0/0/3\ndescription TO-ServerRoom-A',
 '',
 'succeeded', NULL, 'ops-admin', datetime('now','-2 day')),
('a1e2c300-0000-4000-8000-000000000004',
 'de222222-0000-4000-8000-000000000002', 'po222222-0000-4000-8000-000000000001',
 'dot1x_enable',
 '{"dot1x":"disabled"}', '{"dot1x":"enabled"}',
 'interface GigabitEthernet1/0/1\ndot1x enable',
 '',
 'succeeded', NULL, 'admin', datetime('now','-3 day')),
('a1e2c300-0000-4000-8000-000000000005',
 'de222222-0000-4000-8000-000000000002', 'po222222-0000-4000-8000-000000000002',
 'access_vlan',
 '{"accessVlanId":1}', '{"accessVlanId":120}',
 'interface GigabitEthernet1/0/2\nport access vlan 120',
 '',
 'succeeded', NULL, 'ops-admin', datetime('now','-4 day')),

-- ---- failed × 5：含 failureReason（仅 failed 行），覆盖设备拒绝/超时/参数回滚 ----
('a1e2c300-0000-4000-8000-000000000006',
 'de111111-0000-4000-8000-000000000001', 'po111111-0000-4000-8000-000000000004',
 'shutdown',
 '{"adminStatus":"up"}', NULL,
 'interface GigabitEthernet0/0/4\nshutdown',
 'Error: Port is a member of a link-aggregation group',
 'failed', '设备拒绝：端口属于链路聚合组，禁止单独 shutdown', 'admin', datetime('now','-5 day')),
('a1e2c300-0000-4000-8000-000000000007',
 'de111111-0000-4000-8000-000000000001', 'po111111-0000-4000-8000-000000000005',
 'description',
 NULL, NULL,
 'interface GigabitEthernet0/0/5\ndescription X',
 '',
 'failed', 'SSH 会话超时（30s），命令未送达', 'ops-admin', datetime('now','-6 day')),
('a1e2c300-0000-4000-8000-000000000008',
 'de222222-0000-4000-8000-000000000002', 'po222222-0000-4000-8000-000000000003',
 'dot1x_enable',
 '{"dot1x":"disabled"}', NULL,
 'interface GigabitEthernet1/0/3\ndot1x enable',
 'Error: 802.1X conflicts with MAC authentication on this port',
 'failed', '设备拒绝：与 MAC 认证冲突', 'admin', datetime('now','-7 day')),
('a1e2c300-0000-4000-8000-000000000009',
 'de222222-0000-4000-8000-000000000002', 'po222222-0000-4000-8000-000000000004',
 'access_vlan',
 '{"accessVlanId":1}', NULL,
 'interface GigabitEthernet1/0/4\nport access vlan 999',
 'Error: VLAN 999 does not exist',
 'failed', '设备拒绝：目标 VLAN 不存在（预览阶段未校验 VLAN 预存在）', 'ops-admin', datetime('now','-8 day')),
('a1e2c300-0000-4000-8000-000000000010',
 'de111111-0000-4000-8000-000000000001', 'po111111-0000-4000-8000-000000000006',
 'undo_shutdown',
 NULL, NULL,
 'interface GigabitEthernet0/0/6\nundo shutdown',
 '',
 'failed', '设备连接中断，已回滚（显式告警路径）', 'admin', datetime('now','-9 day'));

-- 核对（应返回 10，succeeded=5 / failed=5）
SELECT status, COUNT(*) FROM sys_port_write_audit
 WHERE id LIKE 'a1e2c300-%' GROUP BY status;
