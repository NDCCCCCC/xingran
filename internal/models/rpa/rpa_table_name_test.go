package rpa

// 本文件承接 internal/models/models_hooks_80_03_test.go 中原属根包遗留模型
// （RPATask / RPAWorker / RPAExecution / RPASchedule / RPAVariable / RPATemplate）
// 的 TableName 断言。根包 internal/models/rpa.go 于 2026-11-16 整体删除
// （见 .planning/notes/261113-hardcode-audit-handoff.md R5），其模型是
// 「无引用遗留定义」——database.go 明确不注册，生产代码一律走本子包。
//
// 断言迁到子包后覆盖更完整：子包 9 个模型全部纳入表名锁定，
// 而根包当时只锁了 6 个（缺 Credentials / Sessions / AuditLog 等）。

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPAModelsTableName(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		// ── 原根包遗留模型（6 个）──
		{"Task", Task{}.TableName(), "sys_rpa_tasks"},
		{"Worker", Worker{}.TableName(), "sys_rpa_workers"},
		{"Execution", Execution{}.TableName(), "sys_rpa_executions"},
		{"Schedule", Schedule{}.TableName(), "sys_rpa_schedules"},
		{"Variable", Variable{}.TableName(), "sys_rpa_variables"},
		{"Template", Template{}.TableName(), "sys_rpa_templates"},

		// ── 子包其余模型（补齐覆盖）──
		{"RPACredential", RPACredential{}.TableName(), "sys_rpa_credentials"},
		{"RPASession", RPASession{}.TableName(), "sys_rpa_sessions"},
		{"AuditLog", AuditLog{}.TableName(), "sys_rpa_audit_logs"},
		{"Notification", Notification{}.TableName(), "sys_rpa_notifications"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
