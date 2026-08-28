package errors

// =====================================================================
// Phase 80-04 Task 4: pkg/errors codes.go —— ErrorCode 枚举 DefaultHTTPStatus/DefaultMessage 全量表驱动。
//
// 纪律:codes.go 全部 ErrorCode 枚举逐一遍历,
// DefaultHTTPStatus / DefaultMessage 全表驱动一遍即收;
// 断言引用 codes.go 常量(禁裸数字 status/code)。
// =====================================================================

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// ErrorCode 常量完整列表(从 codes.go 枚举盘点)。
// 格式:{Name, Code, ExpectedHTTPStatus}
// ExpectedHTTPStatus 从 codes.go DefaultHTTPStatus() switch 分支推算。
// ============================================================================

type errorCodeCase struct {
	name               string
	code               ErrorCode
	expectedHTTPStatus int
	skip               bool
}

var allErrorCodeCases = []errorCodeCase{
	// 0: CodeSuccess
	{name: "CodeSuccess", code: CodeSuccess, expectedHTTPStatus: http.StatusOK},

	// 1000-1999 系统级
	{name: "CodeParamError", code: CodeParamError, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeParamMissing", code: CodeParamMissing, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeParamInvalid", code: CodeParamInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRecordNotFound", code: CodeRecordNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRecordExists", code: CodeRecordExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDatabaseError", code: CodeDatabaseError, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeUnauthorized", code: CodeUnauthorized, expectedHTTPStatus: http.StatusUnauthorized},
	{name: "CodeTokenExpired", code: CodeTokenExpired, expectedHTTPStatus: http.StatusUnauthorized},
	{name: "CodeTokenInvalid", code: CodeTokenInvalid, expectedHTTPStatus: http.StatusUnauthorized},
	{name: "CodePermissionDenied", code: CodePermissionDenied, expectedHTTPStatus: http.StatusUnauthorized},
	{name: "CodeServerError", code: CodeServerError, expectedHTTPStatus: http.StatusInternalServerError},
	{name: "CodeNotImplemented", code: CodeNotImplemented, expectedHTTPStatus: http.StatusInternalServerError},

	// 2000-2999 用户权限
	{name: "CodeUserNotFound", code: CodeUserNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeUserExists", code: CodeUserExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeUserDisabled", code: CodeUserDisabled, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeUserPassword", code: CodeUserPassword, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeUserHasRoles", code: CodeUserHasRoles, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeUserDeleteSelf", code: CodeUserDeleteSelf, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoleNotFound", code: CodeRoleNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoleExists", code: CodeRoleExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoleHasUsers", code: CodeRoleHasUsers, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoleHasMenus", code: CodeRoleHasMenus, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoleHasDepts", code: CodeRoleHasDepts, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoleIsAdmin", code: CodeRoleIsAdmin, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoleIsSuper", code: CodeRoleIsSuper, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDeptNotFound", code: CodeDeptNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDeptExists", code: CodeDeptExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDeptHasUsers", code: CodeDeptHasUsers, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDeptHasChildren", code: CodeDeptHasChildren, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDeptHasRoles", code: CodeDeptHasRoles, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDeptInvalid", code: CodeDeptInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDeptIsParent", code: CodeDeptIsParent, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodePostNotFound", code: CodePostNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodePostExists", code: CodePostExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodePostHasUsers", code: CodePostHasUsers, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuNotFound", code: CodeMenuNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuExists", code: CodeMenuExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuHasChildren", code: CodeMenuHasChildren, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuHasRoles", code: CodeMenuHasRoles, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuInvalid", code: CodeMenuInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuIsParent", code: CodeMenuIsParent, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuTypeInvalid", code: CodeMenuTypeInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeMenuHasButtons", code: CodeMenuHasButtons, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDictTypeNotFound", code: CodeDictTypeNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDictTypeExists", code: CodeDictTypeExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDictDataNotFound", code: CodeDictDataNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDictDataExists", code: CodeDictDataExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDictTypeInUse", code: CodeDictTypeInUse, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeConfigNotFound", code: CodeConfigNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeConfigExists", code: CodeConfigExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeConfigReserved", code: CodeConfigReserved, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeNoticeNotFound", code: CodeNoticeNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeADConfigNotFound", code: CodeADConfigNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeADConfigExists", code: CodeADConfigExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeADConnectionFailed", code: CodeADConnectionFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeADSyncFailed", code: CodeADSyncFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeSettingsNotFound", code: CodeSettingsNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDashboardNotFound", code: CodeDashboardNotFound, expectedHTTPStatus: http.StatusBadRequest},

	// 3000-3999 运维
	{name: "CodeBuildingNotFound", code: CodeBuildingNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeBuildingExists", code: CodeBuildingExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeBuildingHasFloors", code: CodeBuildingHasFloors, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeBuildingInvalid", code: CodeBuildingInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeBuildingOrgInvalid", code: CodeBuildingOrgInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeFloorNotFound", code: CodeFloorNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeFloorExists", code: CodeFloorExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeFloorHasWorkstations", code: CodeFloorHasWorkstations, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeFloorInvalid", code: CodeFloorInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWorkstationNotFound", code: CodeWorkstationNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWorkstationExists", code: CodeWorkstationExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWorkstationInvalid", code: CodeWorkstationInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeServerRoomNotFound", code: CodeServerRoomNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeServerRoomExists", code: CodeServerRoomExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeServerRoomHasServers", code: CodeServerRoomHasServers, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoomDeviceNotFound", code: CodeRoomDeviceNotFound, expectedHTTPStatus: http.StatusNotFound},
	{name: "CodeRoomDeviceCodeExists", code: CodeRoomDeviceCodeExists, expectedHTTPStatus: http.StatusConflict},
	{name: "CodeInfoPointNotFound", code: CodeInfoPointNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeInfoPointExists", code: CodeInfoPointExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDoorNotFound", code: CodeDoorNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDoorExists", code: CodeDoorExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWallNotFound", code: CodeWallNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWallExists", code: CodeWallExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDedicatedLineNotFound", code: CodeDedicatedLineNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDedicatedLineExists", code: CodeDedicatedLineExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeFloorPlanTextNotFound", code: CodeFloorPlanTextNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoomPhotoNotFound", code: CodeRoomPhotoNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeRoomPhotoExists", code: CodeRoomPhotoExists, expectedHTTPStatus: http.StatusBadRequest},

	// 4000-4999 调度
	{name: "CodeJobNotFound", code: CodeJobNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeJobExists", code: CodeJobExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeJobIsRunning", code: CodeJobIsRunning, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeJobHasCron", code: CodeJobHasCron, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeCronInvalid", code: CodeCronInvalid, expectedHTTPStatus: http.StatusBadRequest},

	// 5000-5999 工单
	{name: "CodeWorkorderNotFound", code: CodeWorkorderNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWorkorderExists", code: CodeWorkorderExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWorkorderInvalid", code: CodeWorkorderInvalid, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeWorkorderStatus", code: CodeWorkorderStatus, expectedHTTPStatus: http.StatusBadRequest},

	// 6000-6999 监控
	{name: "CodeMonitorDataNotFound", code: CodeMonitorDataNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeCacheKeyNotFound", code: CodeCacheKeyNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeCacheOperationFailed", code: CodeCacheOperationFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeServerInfoNotFound", code: CodeServerInfoNotFound, expectedHTTPStatus: http.StatusBadRequest},

	// 7000-7999 网络设备
	{name: "CodeNetworkDeviceNotFound", code: CodeNetworkDeviceNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeNetworkDeviceExists", code: CodeNetworkDeviceExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeNetworkDeviceConnect", code: CodeNetworkDeviceConnect, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeTemplateNotFound", code: CodeTemplateNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeTemplateExists", code: CodeTemplateExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeCredentialNotFound", code: CodeCredentialNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeCredentialExists", code: CodeCredentialExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeCommandFailed", code: CodeCommandFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeBackupNotFound", code: CodeBackupNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodePortCollectionFailed", code: CodePortCollectionFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDiscoveryFailed", code: CodeDiscoveryFailed, expectedHTTPStatus: http.StatusBadRequest},

	// 8000-8999 知识库
	{name: "CodeKnowledgeNotFound", code: CodeKnowledgeNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeKnowledgeExists", code: CodeKnowledgeExists, expectedHTTPStatus: http.StatusBadRequest},

	// 9000-9999 值班
	{name: "CodeDutyNotFound", code: CodeDutyNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeDutyExists", code: CodeDutyExists, expectedHTTPStatus: http.StatusBadRequest},

	// 54000-54999 VDI
	{name: "CodeVDIServerNotFound", code: CodeVDIServerNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVDIServerExists", code: CodeVDIServerExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVDIApiFailed", code: CodeVDIApiFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVDIAuthFailed", code: CodeVDIAuthFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVDITokenExpired", code: CodeVDITokenExpired, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVMNotFound", code: CodeVMNotFound, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVMExists", code: CodeVMExists, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVMOperationFailed", code: CodeVMOperationFailed, expectedHTTPStatus: http.StatusBadRequest},
	{name: "CodeVMInconsistentState", code: CodeVMInconsistentState, expectedHTTPStatus: http.StatusBadRequest},
}

// TestEdc8004_DefaultHTTPStatus_AllCodes 全量 ErrorCode 的 DefaultHTTPStatus 映射。
func TestEdc8004_DefaultHTTPStatus_AllCodes(t *testing.T) {
	t.Parallel()
	for _, tc := range allErrorCodeCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.skip {
				t.Skip("skipped")
			}
			status := tc.code.DefaultHTTPStatus()
			assert.Equal(t, tc.expectedHTTPStatus, status,
				"DefaultHTTPStatus() mismatch for %s(code=%d)", tc.name, tc.code)
		})
	}
}

// TestEdc8004_DefaultMessage_AllCodes 全量 ErrorCode 的 DefaultMessage 非空验证。
func TestEdc8004_DefaultMessage_AllCodes(t *testing.T) {
	t.Parallel()
	// 占位符残留检查表。
	placeholderFragments := []string{
		"<nil>", "TODO", "FIXME", "unknown", "Unknown",
	}
	for _, tc := range allErrorCodeCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.skip {
				t.Skip("skipped")
			}
			msg := tc.code.DefaultMessage()
			assert.NotEmpty(t, msg, "DefaultMessage() must be non-empty for %s", tc.name)
			// 不能是未知错误占位符。
			assert.NotEqual(t, "未知错误", msg, "DefaultMessage() must not be generic placeholder for %s", tc.name)
			// 检查占位符残留。
			for _, ph := range placeholderFragments {
				assert.NotContains(t, strings.ToLower(msg), strings.ToLower(ph),
					"DefaultMessage() contains placeholder '%s' for %s", ph, tc.name)
			}
		})
	}
}

// TestEdc8004_DefaultMessage_SelectedCodes 抽样 10 个关键码精确断言。
func TestEdc8004_DefaultMessage_SelectedCodes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		code         ErrorCode
		expectedMsg  string
	}{
		{CodeSuccess, "成功"},
		{CodeParamError, "参数错误"},
		{CodeParamMissing, "参数缺失"},
		{CodeRecordNotFound, "记录不存在"},
		{CodeUnauthorized, "未授权"},
		{CodeTokenExpired, "令牌已过期"},
		{CodePermissionDenied, "权限不足"},
		{CodeServerError, "服务器内部错误"},
		{CodeUserNotFound, "用户不存在"},
		{CodeRoleNotFound, "角色不存在"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("code_%d", tc.code), func(t *testing.T) {
			t.Parallel()
			msg := tc.code.DefaultMessage()
			assert.Equal(t, tc.expectedMsg, msg, "DefaultMessage() mismatch for code=%d", tc.code)
		})
	}
}

// TestEdc8004_UnknownCode_Fallback 越界 ErrorCode 的 fallback 行为。
func TestEdc8004_UnknownCode_Fallback(t *testing.T) {
	t.Parallel()
	// 越界值 → DefaultHTTPStatus 返回 http.StatusInternalServerError。
	unknownCode := ErrorCode(99999)
	assert.Equal(t, http.StatusInternalServerError, unknownCode.DefaultHTTPStatus())

	// 越界值 → DefaultMessage 返回 "未知错误"。
	assert.Equal(t, "未知错误", unknownCode.DefaultMessage())

	// 负数值。
	negCode := ErrorCode(-1)
	status := negCode.DefaultHTTPStatus()
	assert.GreaterOrEqual(t, status, http.StatusBadRequest)
	assert.Less(t, status, http.StatusNetworkAuthenticationRequired) // 合理范围。
}

// TestEdc8004_ErrorCode_ValueConsistency 验证 ErrorCode 值在 int 范围内部不会静默溢出。
func TestEdc8004_ErrorCode_ValueConsistency(t *testing.T) {
	t.Parallel()
	// 所有枚举值必须 > 0(CodeSuccess=0 除外)。
	for _, tc := range allErrorCodeCases {
		if tc.code == CodeSuccess {
			continue
		}
		assert.Greater(t, int(tc.code), 0, "%s must have positive int value", tc.name)
	}
	// VDI 段。
	vdicodes := []ErrorCode{
		CodeVDIServerNotFound, CodeVDIServerExists, CodeVDIApiFailed,
		CodeVDIAuthFailed, CodeVDITokenExpired, CodeVMNotFound, CodeVMExists,
		CodeVMOperationFailed, CodeVMInconsistentState,
	}
	for _, c := range vdicodes {
		assert.GreaterOrEqual(t, int(c), 54000, "VDI code=%d must be >= 54000", c)
		assert.Less(t, int(c), 55000, "VDI code=%d must be < 55000", c)
	}
}
