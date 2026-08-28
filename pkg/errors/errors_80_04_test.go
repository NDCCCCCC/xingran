package errors

// =====================================================================
// Phase 80-04 Task 3: pkg/errors errors.go —— 构造器群 + Wrap/Context/Getter 全 table-driven。
//
// 纪律:errors.go ~100 个构造器全 0%,纯表驱动一次收满;
// 断言引用 codes.go 常量(禁裸数字 status/code)。
// =====================================================================

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Constructor table entries
// ============================================================================

type constructorCase struct {
	name           string
	constructor    func() *AppError
	expectedCode   ErrorCode
	expectedMsgNonEmpty bool
	skip           bool
}

// constructorTestCases 全部 ~100 个 errors.go 构造器(按代码顺序盘点)。
// 每行验证:返回非 nil、GetCode() 对应、GetHTTPStatus() 合规、Error() 含消息。
var constructorTestCases = []constructorCase{
	// --- 系统级通用错误 ---
	{name: "ParamError", constructor: ParamError, expectedCode: CodeParamError},
	{name: "BadRequest", constructor: func() *AppError { return BadRequest("bad request test") }, expectedCode: CodeParamError},
	{name: "ParamMissing", constructor: func() *AppError { return ParamMissing("field") }, expectedCode: CodeParamMissing},
	{name: "ParamInvalid", constructor: func() *AppError { return ParamInvalid("field") }, expectedCode: CodeParamInvalid},
	{name: "RecordNotFound", constructor: RecordNotFound, expectedCode: CodeRecordNotFound},
	{name: "RecordNotFoundWithMsg", constructor: func() *AppError { return RecordNotFoundWithMsg("custom not found") }, expectedCode: CodeRecordNotFound},
	{name: "RecordExists", constructor: RecordExists, expectedCode: CodeRecordExists},
	{name: "DatabaseError", constructor: func() *AppError { return DatabaseError(errors.New("db error")) }, expectedCode: CodeDatabaseError},
	{name: "Unauthorized", constructor: Unauthorized, expectedCode: CodeUnauthorized},
	{name: "UnauthorizedWithMsg", constructor: func() *AppError { return UnauthorizedWithMsg("custom unauthorized") }, expectedCode: CodeUnauthorized},
	{name: "TokenExpired", constructor: TokenExpired, expectedCode: CodeTokenExpired},
	{name: "TokenInvalid", constructor: TokenInvalid, expectedCode: CodeTokenInvalid},
	{name: "PermissionDenied", constructor: PermissionDenied, expectedCode: CodePermissionDenied},
	{name: "PermissionDeniedWithMsg", constructor: func() *AppError { return PermissionDeniedWithMsg("custom denied") }, expectedCode: CodePermissionDenied},
	{name: "Forbidden", constructor: Forbidden, expectedCode: CodePermissionDenied},
	{name: "ServerError", constructor: func() *AppError { return ServerError(errors.New("server err")) }, expectedCode: CodeServerError},
	{name: "InternalServerError", constructor: func() *AppError { return InternalServerError(errors.New("internal err")) }, expectedCode: CodeServerError},
	{name: "InternalServerErrorWithMsg", constructor: func() *AppError { return InternalServerErrorWithMsg("custom internal") }, expectedCode: CodeServerError},
	{name: "NotFound", constructor: func() *AppError { return NotFound("custom not found") }, expectedCode: CodeRecordNotFound},
	{name: "InvalidOperation", constructor: func() *AppError { return InvalidOperation("op") }, expectedCode: CodeParamInvalid},
	{name: "InvalidOperationWithMsg", constructor: func() *AppError { return InvalidOperationWithMsg("custom invalid op") }, expectedCode: CodeParamInvalid},
	{name: "NotImplemented", constructor: NotImplemented, expectedCode: CodeNotImplemented},

	// --- 用户权限模块错误 ---
	{name: "UserNotFound", constructor: UserNotFound, expectedCode: CodeUserNotFound},
	{name: "UserNotFoundWithID", constructor: func() *AppError { return UserNotFoundWithID("user-123") }, expectedCode: CodeUserNotFound},
	{name: "UserExists", constructor: UserExists, expectedCode: CodeUserExists},
	{name: "UserExistsWithUsername", constructor: func() *AppError { return UserExistsWithUsername("alice") }, expectedCode: CodeUserExists},
	{name: "UserDisabled", constructor: UserDisabled, expectedCode: CodeUserDisabled},
	{name: "PasswordError", constructor: PasswordError, expectedCode: CodeUserPassword},
	{name: "UserHasRoles", constructor: UserHasRoles, expectedCode: CodeUserHasRoles},
	{name: "UserDeleteSelf", constructor: UserDeleteSelf, expectedCode: CodeUserDeleteSelf},

	// --- 角色模块错误 ---
	{name: "RoleNotFound", constructor: RoleNotFound, expectedCode: CodeRoleNotFound},
	{name: "RoleNotFoundWithID", constructor: func() *AppError { return RoleNotFoundWithID("role-123") }, expectedCode: CodeRoleNotFound},
	{name: "RoleExists", constructor: RoleExists, expectedCode: CodeRoleExists},
	{name: "RoleExistsWithName", constructor: func() *AppError { return RoleExistsWithName("Admin") }, expectedCode: CodeRoleExists},
	{name: "RoleKeyExists", constructor: func() *AppError { return RoleKeyExists("admin") }, expectedCode: CodeRoleExists},
	{name: "RoleHasUsers", constructor: RoleHasUsers, expectedCode: CodeRoleHasUsers},
	{name: "RoleHasMenus", constructor: RoleHasMenus, expectedCode: CodeRoleHasMenus},
	{name: "RoleHasDepts", constructor: RoleHasDepts, expectedCode: CodeRoleHasDepts},
	{name: "RoleIsAdmin", constructor: RoleIsAdmin, expectedCode: CodeRoleIsAdmin},
	{name: "RoleIsSuper", constructor: RoleIsSuper, expectedCode: CodeRoleIsSuper},

	// --- 部门模块错误 ---
	{name: "DeptNotFound", constructor: DeptNotFound, expectedCode: CodeDeptNotFound},
	{name: "DeptNotFoundWithID", constructor: func() *AppError { return DeptNotFoundWithID("dept-123") }, expectedCode: CodeDeptNotFound},
	{name: "DeptExists", constructor: DeptExists, expectedCode: CodeDeptExists},
	{name: "DeptHasUsers", constructor: DeptHasUsers, expectedCode: CodeDeptHasUsers},
	{name: "DeptHasChildren", constructor: DeptHasChildren, expectedCode: CodeDeptHasChildren},
	{name: "DeptHasRoles", constructor: DeptHasRoles, expectedCode: CodeDeptHasRoles},
	{name: "DeptInvalid", constructor: DeptInvalid, expectedCode: CodeDeptInvalid},
	{name: "DeptIsParent", constructor: DeptIsParent, expectedCode: CodeDeptIsParent},

	// --- 岗位模块错误 ---
	{name: "PostNotFound", constructor: PostNotFound, expectedCode: CodePostNotFound},
	{name: "PostExists", constructor: PostExists, expectedCode: CodePostExists},
	{name: "PostHasUsers", constructor: PostHasUsers, expectedCode: CodePostHasUsers},

	// --- 菜单模块错误 ---
	{name: "MenuNotFound", constructor: MenuNotFound, expectedCode: CodeMenuNotFound},
	{name: "MenuNotFoundWithID", constructor: func() *AppError { return MenuNotFoundWithID("menu-123") }, expectedCode: CodeMenuNotFound},
	{name: "MenuExists", constructor: MenuExists, expectedCode: CodeMenuExists},
	{name: "MenuHasChildren", constructor: MenuHasChildren, expectedCode: CodeMenuHasChildren},
	{name: "MenuHasRoles", constructor: MenuHasRoles, expectedCode: CodeMenuHasRoles},
	{name: "MenuInvalid", constructor: MenuInvalid, expectedCode: CodeMenuInvalid},
	{name: "MenuIsParent", constructor: MenuIsParent, expectedCode: CodeMenuIsParent},
	{name: "MenuTypeInvalid", constructor: MenuTypeInvalid, expectedCode: CodeMenuTypeInvalid},
	{name: "MenuHasButtons", constructor: MenuHasButtons, expectedCode: CodeMenuHasButtons},

	// --- 字典模块错误 ---
	{name: "DictTypeNotFound", constructor: DictTypeNotFound, expectedCode: CodeDictTypeNotFound},
	{name: "DictTypeExists", constructor: DictTypeExists, expectedCode: CodeDictTypeExists},
	{name: "DictDataNotFound", constructor: DictDataNotFound, expectedCode: CodeDictDataNotFound},
	{name: "DictDataExists", constructor: DictDataExists, expectedCode: CodeDictDataExists},
	{name: "DictTypeInUse", constructor: DictTypeInUse, expectedCode: CodeDictTypeInUse},

	// --- 配置模块错误 ---
	{name: "ConfigNotFound", constructor: ConfigNotFound, expectedCode: CodeConfigNotFound},
	{name: "ConfigExists", constructor: ConfigExists, expectedCode: CodeConfigExists},
	{name: "ConfigReserved", constructor: ConfigReserved, expectedCode: CodeConfigReserved},

	// --- 通知公告错误 ---
	{name: "NoticeNotFound", constructor: NoticeNotFound, expectedCode: CodeNoticeNotFound},

	// --- 运维管理模块错误 ---
	{name: "BuildingNotFound", constructor: BuildingNotFound, expectedCode: CodeBuildingNotFound},
	{name: "BuildingNotFoundWithID", constructor: func() *AppError { return BuildingNotFoundWithID("bld-123") }, expectedCode: CodeBuildingNotFound},
	{name: "BuildingExists", constructor: BuildingExists, expectedCode: CodeBuildingExists},
	{name: "BuildingExistsWithMsg", constructor: func() *AppError { return BuildingExistsWithMsg("custom building exists") }, expectedCode: CodeBuildingExists},
	{name: "BuildingHasFloors", constructor: BuildingHasFloors, expectedCode: CodeBuildingHasFloors},
	{name: "BuildingInvalid", constructor: BuildingInvalid, expectedCode: CodeBuildingInvalid},
	{name: "BuildingOrgInvalid", constructor: BuildingOrgInvalid, expectedCode: CodeBuildingOrgInvalid},
	{name: "BuildingOrgInvalidWithMsg", constructor: func() *AppError { return BuildingOrgInvalidWithMsg("custom org invalid") }, expectedCode: CodeBuildingOrgInvalid},

	{name: "FloorNotFound", constructor: FloorNotFound, expectedCode: CodeFloorNotFound},
	{name: "FloorNotFoundWithID", constructor: func() *AppError { return FloorNotFoundWithID("floor-123") }, expectedCode: CodeFloorNotFound},
	{name: "FloorExists", constructor: FloorExists, expectedCode: CodeFloorExists},
	{name: "FloorHasWorkstations", constructor: FloorHasWorkstations, expectedCode: CodeFloorHasWorkstations},
	{name: "FloorInvalid", constructor: FloorInvalid, expectedCode: CodeFloorInvalid},

	{name: "WorkstationNotFound", constructor: WorkstationNotFound, expectedCode: CodeWorkstationNotFound},
	{name: "WorkstationNotFoundWithID", constructor: func() *AppError { return WorkstationNotFoundWithID("ws-123") }, expectedCode: CodeWorkstationNotFound},
	{name: "WorkstationExists", constructor: WorkstationExists, expectedCode: CodeWorkstationExists},
	{name: "WorkstationInvalid", constructor: WorkstationInvalid, expectedCode: CodeWorkstationInvalid},

	{name: "ServerRoomNotFound", constructor: ServerRoomNotFound, expectedCode: CodeServerRoomNotFound},
	{name: "ServerRoomExists", constructor: ServerRoomExists, expectedCode: CodeServerRoomExists},
	{name: "ServerRoomHasServers", constructor: ServerRoomHasServers, expectedCode: CodeServerRoomHasServers},

	{name: "RoomDeviceNotFound", constructor: RoomDeviceNotFound, expectedCode: CodeRoomDeviceNotFound},
	{name: "DeviceCodeAlreadyExists", constructor: DeviceCodeAlreadyExists, expectedCode: CodeRoomDeviceCodeExists},

	{name: "InfoPointNotFound", constructor: InfoPointNotFound, expectedCode: CodeInfoPointNotFound},
	{name: "InfoPointExists", constructor: InfoPointExists, expectedCode: CodeInfoPointExists},

	// --- 调度任务模块错误 ---
	{name: "JobNotFound", constructor: JobNotFound, expectedCode: CodeJobNotFound},
	{name: "JobExists", constructor: JobExists, expectedCode: CodeJobExists},
	{name: "JobIsRunning", constructor: JobIsRunning, expectedCode: CodeJobIsRunning},
	{name: "JobHasCron", constructor: JobHasCron, expectedCode: CodeJobHasCron},
	{name: "CronInvalid", constructor: CronInvalid, expectedCode: CodeCronInvalid},
	{name: "CronInvalidWithMsg", constructor: func() *AppError { return CronInvalidWithMsg("bad cron") }, expectedCode: CodeCronInvalid},

	// --- 工单模块错误 ---
	{name: "WorkorderNotFound", constructor: WorkorderNotFound, expectedCode: CodeWorkorderNotFound},
	{name: "WorkorderExists", constructor: WorkorderExists, expectedCode: CodeWorkorderExists},
	{name: "WorkorderInvalid", constructor: WorkorderInvalid, expectedCode: CodeWorkorderInvalid},
	{name: "WorkorderStatus", constructor: WorkorderStatus, expectedCode: CodeWorkorderStatus},
	{name: "WorkorderStatusWithMsg", constructor: func() *AppError { return WorkorderStatusWithMsg("custom status") }, expectedCode: CodeWorkorderStatus},

	// --- 监控模块错误 ---
	{name: "MonitorDataNotFound", constructor: MonitorDataNotFound, expectedCode: CodeMonitorDataNotFound},

	// --- 网络设备模块错误 ---
	{name: "NetworkDeviceNotFound", constructor: NetworkDeviceNotFound, expectedCode: CodeNetworkDeviceNotFound},
	{name: "NetworkDeviceExists", constructor: NetworkDeviceExists, expectedCode: CodeNetworkDeviceExists},
	{name: "NetworkDeviceConnect", constructor: NetworkDeviceConnect, expectedCode: CodeNetworkDeviceConnect},

	// --- 知识库模块错误 ---
	{name: "KnowledgeNotFound", constructor: KnowledgeNotFound, expectedCode: CodeKnowledgeNotFound},
	{name: "KnowledgeExists", constructor: KnowledgeExists, expectedCode: CodeKnowledgeExists},

	// --- 值班管理模块错误 ---
	{name: "DutyNotFound", constructor: DutyNotFound, expectedCode: CodeDutyNotFound},
	{name: "DutyExists", constructor: DutyExists, expectedCode: CodeDutyExists},

	// --- AD域管理模块错误 ---
	{name: "ADConfigNotFound", constructor: ADConfigNotFound, expectedCode: CodeADConfigNotFound},
	{name: "ADConfigExists", constructor: ADConfigExists, expectedCode: CodeADConfigExists},
	{name: "ADConnectionFailed", constructor: func() *AppError { return ADConnectionFailed(errors.New("ldap err")) }, expectedCode: CodeADConnectionFailed},
	{name: "ADSyncFailed", constructor: func() *AppError { return ADSyncFailed(errors.New("sync err")) }, expectedCode: CodeADSyncFailed},

	// --- 系统设置模块错误 ---
	{name: "SettingsNotFound", constructor: SettingsNotFound, expectedCode: CodeSettingsNotFound},
	{name: "DashboardNotFound", constructor: DashboardNotFound, expectedCode: CodeDashboardNotFound},

	// --- 运维补充 ---
	{name: "DoorNotFound", constructor: DoorNotFound, expectedCode: CodeDoorNotFound},
	{name: "DoorExists", constructor: DoorExists, expectedCode: CodeDoorExists},
	{name: "WallNotFound", constructor: WallNotFound, expectedCode: CodeWallNotFound},
	{name: "WallExists", constructor: WallExists, expectedCode: CodeWallExists},
	{name: "DedicatedLineNotFound", constructor: DedicatedLineNotFound, expectedCode: CodeDedicatedLineNotFound},
	{name: "DedicatedLineExists", constructor: DedicatedLineExists, expectedCode: CodeDedicatedLineExists},
	{name: "FloorPlanTextNotFound", constructor: FloorPlanTextNotFound, expectedCode: CodeFloorPlanTextNotFound},
	{name: "RoomPhotoNotFound", constructor: RoomPhotoNotFound, expectedCode: CodeRoomPhotoNotFound},
	{name: "RoomPhotoExists", constructor: RoomPhotoExists, expectedCode: CodeRoomPhotoExists},

	// --- 监控补充 ---
	{name: "CacheKeyNotFound", constructor: CacheKeyNotFound, expectedCode: CodeCacheKeyNotFound},
	{name: "CacheKeyNotFoundWithMsg", constructor: func() *AppError { return CacheKeyNotFoundWithMsg("custom cache miss") }, expectedCode: CodeCacheKeyNotFound},
	{name: "CacheOperationFailed", constructor: func() *AppError { return CacheOperationFailed(errors.New("cache err")) }, expectedCode: CodeCacheOperationFailed},
	{name: "ServerInfoNotFound", constructor: ServerInfoNotFound, expectedCode: CodeServerInfoNotFound},

	// --- 网络设备补充 ---
	{name: "TemplateNotFound", constructor: TemplateNotFound, expectedCode: CodeTemplateNotFound},
	{name: "TemplateExists", constructor: TemplateExists, expectedCode: CodeTemplateExists},
	{name: "CredentialNotFound", constructor: CredentialNotFound, expectedCode: CodeCredentialNotFound},
	{name: "CredentialExists", constructor: CredentialExists, expectedCode: CodeCredentialExists},
	{name: "CommandFailed", constructor: func() *AppError { return CommandFailed(errors.New("cmd err")) }, expectedCode: CodeCommandFailed},
	{name: "CommandFailedWithMsg", constructor: func() *AppError { return CommandFailedWithMsg("custom cmd", errors.New("cmd err")) }, expectedCode: CodeCommandFailed},
	{name: "BackupNotFound", constructor: BackupNotFound, expectedCode: CodeBackupNotFound},
	{name: "PortCollectionFailed", constructor: func() *AppError { return PortCollectionFailed(errors.New("port err")) }, expectedCode: CodePortCollectionFailed},
	{name: "DiscoveryFailed", constructor: func() *AppError { return DiscoveryFailed(errors.New("discovery err")) }, expectedCode: CodeDiscoveryFailed},

	// --- VDI 模块错误 ---
	{name: "VDIServerNotFound", constructor: func() *AppError { return VDIServerNotFound("vdi-123") }, expectedCode: CodeVDIServerNotFound},
	{name: "VDIServerExists", constructor: VDIServerExists, expectedCode: CodeVDIServerExists},
	{name: "VDIApiFailed", constructor: func() *AppError { return VDIApiFailed(errors.New("vdi api err")) }, expectedCode: CodeVDIApiFailed},
	{name: "VDIAuthFailed", constructor: VDIAuthFailed, expectedCode: CodeVDIAuthFailed},
	{name: "VDITokenExpired", constructor: VDITokenExpired, expectedCode: CodeVDITokenExpired},
	{name: "VMNotFound", constructor: func() *AppError { return VMNotFound("vm-123") }, expectedCode: CodeVMNotFound},
	{name: "VMExists", constructor: VMExists, expectedCode: CodeVMExists},
	{name: "VMOperationFailed", constructor: func() *AppError { return VMOperationFailed("start", errors.New("vm err")) }, expectedCode: CodeVMOperationFailed},
	{name: "VMInconsistentState", constructor: func() *AppError { return VMInconsistentState("vm-123") }, expectedCode: CodeVMInconsistentState},
}

// TestApe8004_Constructors_Table 全构造器表驱动测试。
func TestApe8004_Constructors_Table(t *testing.T) {
	t.Parallel()
	for _, tc := range constructorTestCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.skip {
				t.Skip("skipped")
			}
			err := tc.constructor()
			require.NotNil(t, err, "constructor %s must return non-nil", tc.name)
			assert.Equal(t, tc.expectedCode, err.GetCode(), "GetCode() mismatch for %s", tc.name)
			// HTTPStatus 必须 >0。
			httpStatus := err.GetHTTPStatus()
			assert.Greater(t, httpStatus, 0, "GetHTTPStatus() must be >0 for %s", tc.name)
			// Error() 字符串非空。
			errStr := err.Error()
			assert.NotEmpty(t, errStr, "Error() must be non-empty for %s", tc.name)
			// 消息体不能是默认未知错误占位符。
			assert.NotEqual(t, "未知错误", errStr, "Error() must not be generic unknown-error placeholder for %s", tc.name)
		})
	}
}

// TestApe8004_NewAndNewWithHTTPStatus New 与 NewWithHTTPStatus 语义。
func TestApe8004_NewAndNewWithHTTPStatus(t *testing.T) {
	t.Parallel()

	// New: HTTPStatus 取 DefaultHTTPStatus。
	err := New(CodeParamError, "test param")
	assert.Equal(t, CodeParamError, err.GetCode())
	assert.Equal(t, CodeParamError.DefaultHTTPStatus(), err.GetHTTPStatus())
	assert.Equal(t, "test param", err.Message)

	// NewWithHTTPStatus: 显式覆盖 HTTPStatus。
	err2 := NewWithHTTPStatus(CodeParamError, http.StatusConflict, "conflict param")
	assert.Equal(t, CodeParamError, err2.GetCode())
	assert.Equal(t, http.StatusConflict, err2.GetHTTPStatus())
	assert.Equal(t, "conflict param", err2.Message)

	// New: nil err 字段。
	assert.Nil(t, err.Err)
}

// TestApe8004_Wrap_Variants Wrap 与 WrapWithHTTPStatus 语义。
func TestApe8004_Wrap_Variants(t *testing.T) {
	t.Parallel()
	underlying := errors.New("underlying error")

	// Wrap: err!=nil 时返回 *AppError,Err=underlying,HTTPStatus 取 DefaultHTTPStatus。
	wrapped := Wrap(underlying, CodeDatabaseError, "wrapped db error")
	require.NotNil(t, wrapped)
	assert.Equal(t, CodeDatabaseError, wrapped.GetCode())
	assert.Equal(t, CodeDatabaseError.DefaultHTTPStatus(), wrapped.GetHTTPStatus())
	assert.Equal(t, "wrapped db error", wrapped.Message)
	assert.Equal(t, underlying, wrapped.Err)
	assert.True(t, errors.Is(wrapped, underlying))

	// Wrap(nil,...): 仍返回 non-nil *AppError(注释说明)。
	wrappedNil := Wrap(nil, CodeParamError, "no underlying")
	require.NotNil(t, wrappedNil)
	assert.Equal(t, CodeParamError, wrappedNil.GetCode())

	// WrapWithHTTPStatus(nil,...): 返回 nil。
	wrappedHTTPNil := WrapWithHTTPStatus(nil, CodeParamError, http.StatusBadRequest, "should be nil")
	assert.Nil(t, wrappedHTTPNil)

	// WrapWithHTTPStatus: 显式覆盖 HTTPStatus。
	wrappedHTTP := WrapWithHTTPStatus(underlying, CodeUnauthorized, http.StatusForbidden, "forbidden access")
	require.NotNil(t, wrappedHTTP)
	assert.Equal(t, CodeUnauthorized, wrappedHTTP.GetCode())
	assert.Equal(t, http.StatusForbidden, wrappedHTTP.GetHTTPStatus())
	assert.Equal(t, "forbidden access", wrappedHTTP.Message)
	assert.Equal(t, underlying, wrappedHTTP.Err)
}

// TestApe8004_WithContext WithContext / WithContexts 链式追加。
func TestApe8004_WithContext(t *testing.T) {
	t.Parallel()

	// WithContext 单键追加。
	err := New(CodeParamError, "test").WithContext("key1", "value1")
	require.NotNil(t, err)
	ctx := err.GetContext()
	require.NotNil(t, ctx)
	assert.Equal(t, "value1", ctx["key1"])

	// WithContext 链式追加第二个键。
	err2 := err.WithContext("key2", 42)
	ctx2 := err2.GetContext()
	assert.Equal(t, "value1", ctx2["key1"]) // 原键保留。
	assert.Equal(t, 42, ctx2["key2"])

	// WithContexts 批量追加。
	err3 := New(CodeParamError, "batch").WithContexts(map[string]interface{}{
		"batch1": "v1",
		"batch2": "v2",
	})
	ctx3 := err3.GetContext()
	assert.Equal(t, "v1", ctx3["batch1"])
	assert.Equal(t, "v2", ctx3["batch2"])

	// WithContexts(nil) 兼容:不 panic,但 WithContexts 内部初始化 map。
	err5 := New(CodeParamError, "nil batch").WithContexts(nil)
	assert.NotNil(t, err5.GetContext()) // 内部 make 了 map。
}

// TestApe8004_TypeAssert_Helpers IsAppError / GetAppError / GetErrorCode。
func TestApe8004_TypeAssert_Helpers(t *testing.T) {
	t.Parallel()

	appErr := New(CodeUserNotFound, "user missing")
	plainErr := errors.New("plain error")

	// IsAppError。
	assert.True(t, IsAppError(appErr))
	assert.False(t, IsAppError(plainErr))
	assert.False(t, IsAppError(nil))

	// GetAppError。
	assert.Equal(t, appErr, GetAppError(appErr))
	assert.Nil(t, GetAppError(plainErr))
	assert.Nil(t, GetAppError(nil))

	// GetErrorCode。
	assert.Equal(t, CodeUserNotFound, GetErrorCode(appErr))
	assert.Equal(t, CodeServerError, GetErrorCode(plainErr)) // 回退 CodeServerError。
	assert.Equal(t, CodeSuccess, GetErrorCode(nil))            // nil → CodeSuccess。
}

// TestApe8004_ErrorAndUnwrap Error() 格式化 + Unwrap 链。
func TestApe8004_ErrorAndUnwrap(t *testing.T) {
	t.Parallel()

	// 无底层错误。
	err1 := New(CodeParamError, "param error")
	errStr1 := err1.Error()
	assert.Contains(t, errStr1, "param error")
	assert.Contains(t, errStr1, "1001") // CodeParamError = 1001。

	// 有底层错误。
	causeErr := errors.New("cause")
	underlying := fmt.Errorf("original: %w", causeErr)
	err2 := Wrap(underlying, CodeDatabaseError, "db failure")
	errStr2 := err2.Error()
	assert.Contains(t, errStr2, "db failure")
	assert.Contains(t, errStr2, "1014") // CodeDatabaseError = 1014。

	// Unwrap 链:AppError.Err → underlying(fmt包装) → cause( errors.Is 穿透)。
	assert.Equal(t, underlying, err2.Unwrap())
	assert.True(t, errors.Is(err2, causeErr)) // errors.Is 穿透两层。
}

// TestApe8004_ConstructorCodes_Exhaustive 验证每构造器的 Code 与 codes.go 常量对应。
func TestApe8004_ConstructorCodes_Exhaustive(t *testing.T) {
	t.Parallel()
	// 本测试做全量盘点:errors.go 每构造器调一次 GetCode(),
	// 确认返回值与预期 Code 一致,防止误用了错误的 ErrorCode。
	cases := []struct {
		name         string
		constructor  func() *AppError
		expectedCode ErrorCode
	}{
		{"CodeParamError", func() *AppError { return ParamError() }, CodeParamError},
		{"CodeRecordNotFound", func() *AppError { return RecordNotFound() }, CodeRecordNotFound},
		{"CodeRecordExists", func() *AppError { return RecordExists() }, CodeRecordExists},
		{"CodeDatabaseError", func() *AppError { return DatabaseError(errors.New("e")) }, CodeDatabaseError},
		{"CodeUnauthorized", func() *AppError { return Unauthorized() }, CodeUnauthorized},
		{"CodeTokenExpired", func() *AppError { return TokenExpired() }, CodeTokenExpired},
		{"CodeTokenInvalid", func() *AppError { return TokenInvalid() }, CodeTokenInvalid},
		{"CodePermissionDenied", func() *AppError { return PermissionDenied() }, CodePermissionDenied},
		{"CodeServerError", func() *AppError { return ServerError(errors.New("e")) }, CodeServerError},
		{"CodeNotImplemented", func() *AppError { return NotImplemented() }, CodeNotImplemented},
		{"CodeUserNotFound", func() *AppError { return UserNotFound() }, CodeUserNotFound},
		{"CodeUserExists", func() *AppError { return UserExists() }, CodeUserExists},
		{"CodeUserDisabled", func() *AppError { return UserDisabled() }, CodeUserDisabled},
		{"CodeUserPassword", func() *AppError { return PasswordError() }, CodeUserPassword},
		{"CodeRoleNotFound", func() *AppError { return RoleNotFound() }, CodeRoleNotFound},
		{"CodeRoleExists", func() *AppError { return RoleExists() }, CodeRoleExists},
		{"CodeDeptNotFound", func() *AppError { return DeptNotFound() }, CodeDeptNotFound},
		{"CodeDeptExists", func() *AppError { return DeptExists() }, CodeDeptExists},
		{"CodePostNotFound", func() *AppError { return PostNotFound() }, CodePostNotFound},
		{"CodeMenuNotFound", func() *AppError { return MenuNotFound() }, CodeMenuNotFound},
		{"CodeDictTypeNotFound", func() *AppError { return DictTypeNotFound() }, CodeDictTypeNotFound},
		{"CodeConfigNotFound", func() *AppError { return ConfigNotFound() }, CodeConfigNotFound},
		{"CodeNoticeNotFound", func() *AppError { return NoticeNotFound() }, CodeNoticeNotFound},
		{"CodeBuildingNotFound", func() *AppError { return BuildingNotFound() }, CodeBuildingNotFound},
		{"CodeFloorNotFound", func() *AppError { return FloorNotFound() }, CodeFloorNotFound},
		{"CodeWorkstationNotFound", func() *AppError { return WorkstationNotFound() }, CodeWorkstationNotFound},
		{"CodeServerRoomNotFound", func() *AppError { return ServerRoomNotFound() }, CodeServerRoomNotFound},
		{"CodeJobNotFound", func() *AppError { return JobNotFound() }, CodeJobNotFound},
		{"CodeWorkorderNotFound", func() *AppError { return WorkorderNotFound() }, CodeWorkorderNotFound},
		{"CodeMonitorDataNotFound", func() *AppError { return MonitorDataNotFound() }, CodeMonitorDataNotFound},
		{"CodeNetworkDeviceNotFound", func() *AppError { return NetworkDeviceNotFound() }, CodeNetworkDeviceNotFound},
		{"CodeKnowledgeNotFound", func() *AppError { return KnowledgeNotFound() }, CodeKnowledgeNotFound},
		{"CodeDutyNotFound", func() *AppError { return DutyNotFound() }, CodeDutyNotFound},
		{"CodeADConfigNotFound", func() *AppError { return ADConfigNotFound() }, CodeADConfigNotFound},
		{"CodeSettingsNotFound", func() *AppError { return SettingsNotFound() }, CodeSettingsNotFound},
		{"CodeDashboardNotFound", func() *AppError { return DashboardNotFound() }, CodeDashboardNotFound},
		{"CodeVDIServerNotFound", func() *AppError { return VDIServerNotFound("id") }, CodeVDIServerNotFound},
		{"CodeVMNotFound", func() *AppError { return VMNotFound("id") }, CodeVMNotFound},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.constructor()
			assert.Equal(t, tc.expectedCode, err.GetCode())
		})
	}
}
