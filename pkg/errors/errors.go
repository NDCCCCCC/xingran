package errors

import (
	"errors"
	"fmt"
)

// AppError 应用错误结构
type AppError struct {
	Code       ErrorCode              // 业务错误码
	Message    string                 // 错误消息
	HTTPStatus int                    // HTTP状态码
	Err        error                  // 原始错误
	Context    map[string]interface{} // 错误上下文
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现错误包装接口，支持errors.Is和errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithContexts 添加多个上下文信息
func (e *AppError) WithContexts(ctx map[string]interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	for k, v := range ctx {
		e.Context[k] = v
	}
	return e
}

// GetCode 获取错误码
func (e *AppError) GetCode() ErrorCode {
	return e.Code
}

// GetHTTPStatus 获取HTTP状态码
func (e *AppError) GetHTTPStatus() int {
	if e.HTTPStatus != 0 {
		return e.HTTPStatus
	}
	return e.Code.DefaultHTTPStatus()
}

// GetContext 获取上下文信息
func (e *AppError) GetContext() map[string]interface{} {
	return e.Context
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: code.DefaultHTTPStatus(),
	}
}

// NewWithHTTPStatus 创建带自定义HTTP状态码的应用错误
func NewWithHTTPStatus(code ErrorCode, httpStatus int, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap 包装错误
// 即使 err == nil 也返回 *AppError（不返回 nil），
// 避免调用方拿到 nil error 后误判为"成功"（导致业务逻辑错乱或 nil 指针 panic）。
// 如需创建"无底层错误"的纯业务错误，请使用 New(code, message)。
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: code.DefaultHTTPStatus(),
		Err:        err,
	}
}

// WrapWithHTTPStatus 包装错误并指定HTTP状态码
func WrapWithHTTPStatus(err error, code ErrorCode, httpStatus int, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// IsAppError 判断是否是AppError类型
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError 获取AppError
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}

// GetErrorCode 获取错误码，如果不是AppError则返回CodeServerError
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		// nil error = 无错误 = 成功码。
		// 若此处返回 CodeServerError,调用方在成功路径上会拿到 1500,
		// 导致"按错误码判断结果"的逻辑全部误判(测试 TestValidator_* 即为此暴露)。
		return CodeSuccess
	}
	if appErr := GetAppError(err); appErr != nil {
		return appErr.Code
	}
	return CodeServerError
}

// ============================================================================
// 系统级通用错误 - 便捷构造函数
// ============================================================================

func ParamError() *AppError {
	return New(CodeParamError, CodeParamError.DefaultMessage())
}

func BadRequest(msg string) *AppError {
	return New(CodeParamError, msg)
}

func ParamMissing(field string) *AppError {
	msg := CodeParamMissing.DefaultMessage()
	if field != "" {
		msg = fmt.Sprintf("%s: %s", msg, field)
	}
	return New(CodeParamMissing, msg)
}

func ParamInvalid(field string) *AppError {
	msg := CodeParamInvalid.DefaultMessage()
	if field != "" {
		msg = fmt.Sprintf("%s: %s", msg, field)
	}
	return New(CodeParamInvalid, msg)
}

func RecordNotFound() *AppError {
	return New(CodeRecordNotFound, CodeRecordNotFound.DefaultMessage())
}

func RecordNotFoundWithMsg(msg string) *AppError {
	return New(CodeRecordNotFound, msg)
}

func RecordExists() *AppError {
	return New(CodeRecordExists, CodeRecordExists.DefaultMessage())
}

func DatabaseError(err error) *AppError {
	return Wrap(err, CodeDatabaseError, CodeDatabaseError.DefaultMessage())
}

func Unauthorized() *AppError {
	return New(CodeUnauthorized, CodeUnauthorized.DefaultMessage())
}

func UnauthorizedWithMsg(msg string) *AppError {
	return New(CodeUnauthorized, msg)
}

func TokenExpired() *AppError {
	return New(CodeTokenExpired, CodeTokenExpired.DefaultMessage())
}

func TokenInvalid() *AppError {
	return New(CodeTokenInvalid, CodeTokenInvalid.DefaultMessage())
}

func PermissionDenied() *AppError {
	return New(CodePermissionDenied, CodePermissionDenied.DefaultMessage())
}

func PermissionDeniedWithMsg(msg string) *AppError {
	return New(CodePermissionDenied, msg)
}

func Forbidden() *AppError {
	return New(CodePermissionDenied, CodePermissionDenied.DefaultMessage())
}

func ServerError(err error) *AppError {
	return Wrap(err, CodeServerError, CodeServerError.DefaultMessage())
}

func InternalServerError(err error) *AppError {
	return Wrap(err, CodeServerError, CodeServerError.DefaultMessage())
}

func InternalServerErrorWithMsg(msg string) *AppError {
	return New(CodeServerError, msg)
}

func NotFound(msg string) *AppError {
	return New(CodeRecordNotFound, msg)
}

func InvalidOperation(msg string) *AppError {
	return New(CodeParamInvalid, "不支持的操作: "+msg)
}

func InvalidOperationWithMsg(msg string) *AppError {
	return New(CodeParamInvalid, msg)
}

func NotImplemented() *AppError {
	return New(CodeNotImplemented, CodeNotImplemented.DefaultMessage())
}

// ============================================================================
// 用户权限模块错误 - 便捷构造函数
// ============================================================================

func UserNotFound() *AppError {
	return New(CodeUserNotFound, CodeUserNotFound.DefaultMessage())
}

func UserNotFoundWithID(id string) *AppError {
	return New(CodeUserNotFound, fmt.Sprintf("用户不存在: %s", id))
}

func UserExists() *AppError {
	return New(CodeUserExists, CodeUserExists.DefaultMessage())
}

func UserExistsWithUsername(username string) *AppError {
	return New(CodeUserExists, fmt.Sprintf("用户已存在: %s", username))
}

func UserDisabled() *AppError {
	return New(CodeUserDisabled, CodeUserDisabled.DefaultMessage())
}

func PasswordError() *AppError {
	return New(CodeUserPassword, CodeUserPassword.DefaultMessage())
}

func UserHasRoles() *AppError {
	return New(CodeUserHasRoles, CodeUserHasRoles.DefaultMessage())
}

func UserDeleteSelf() *AppError {
	return New(CodeUserDeleteSelf, CodeUserDeleteSelf.DefaultMessage())
}

func RoleNotFound() *AppError {
	return New(CodeRoleNotFound, CodeRoleNotFound.DefaultMessage())
}

func RoleNotFoundWithID(id string) *AppError {
	return New(CodeRoleNotFound, fmt.Sprintf("角色不存在: %s", id))
}

func RoleExists() *AppError {
	return New(CodeRoleExists, CodeRoleExists.DefaultMessage())
}

func RoleExistsWithName(name string) *AppError {
	return New(CodeRoleExists, fmt.Sprintf("角色名称已存在: %s", name))
}

func RoleKeyExists(key string) *AppError {
	return New(CodeRoleExists, fmt.Sprintf("权限字符已存在: %s", key))
}

func RoleHasUsers() *AppError {
	return New(CodeRoleHasUsers, CodeRoleHasUsers.DefaultMessage())
}

func RoleHasMenus() *AppError {
	return New(CodeRoleHasMenus, CodeRoleHasMenus.DefaultMessage())
}

func RoleHasDepts() *AppError {
	return New(CodeRoleHasDepts, CodeRoleHasDepts.DefaultMessage())
}

func RoleIsAdmin() *AppError {
	return New(CodeRoleIsAdmin, CodeRoleIsAdmin.DefaultMessage())
}

func RoleIsSuper() *AppError {
	return New(CodeRoleIsSuper, CodeRoleIsSuper.DefaultMessage())
}

func DeptNotFound() *AppError {
	return New(CodeDeptNotFound, CodeDeptNotFound.DefaultMessage())
}

func DeptNotFoundWithID(id string) *AppError {
	return New(CodeDeptNotFound, fmt.Sprintf("部门不存在: %s", id))
}

func DeptExists() *AppError {
	return New(CodeDeptExists, CodeDeptExists.DefaultMessage())
}

func DeptHasUsers() *AppError {
	return New(CodeDeptHasUsers, CodeDeptHasUsers.DefaultMessage())
}

func DeptHasChildren() *AppError {
	return New(CodeDeptHasChildren, CodeDeptHasChildren.DefaultMessage())
}

func DeptHasRoles() *AppError {
	return New(CodeDeptHasRoles, CodeDeptHasRoles.DefaultMessage())
}

func DeptInvalid() *AppError {
	return New(CodeDeptInvalid, CodeDeptInvalid.DefaultMessage())
}

func DeptIsParent() *AppError {
	return New(CodeDeptIsParent, CodeDeptIsParent.DefaultMessage())
}

func PostNotFound() *AppError {
	return New(CodePostNotFound, CodePostNotFound.DefaultMessage())
}

func PostExists() *AppError {
	return New(CodePostExists, CodePostExists.DefaultMessage())
}

func PostHasUsers() *AppError {
	return New(CodePostHasUsers, CodePostHasUsers.DefaultMessage())
}

func MenuNotFound() *AppError {
	return New(CodeMenuNotFound, CodeMenuNotFound.DefaultMessage())
}

func MenuNotFoundWithID(id string) *AppError {
	return New(CodeMenuNotFound, fmt.Sprintf("菜单不存在: %s", id))
}

func MenuExists() *AppError {
	return New(CodeMenuExists, CodeMenuExists.DefaultMessage())
}

func MenuHasChildren() *AppError {
	return New(CodeMenuHasChildren, CodeMenuHasChildren.DefaultMessage())
}

func MenuHasRoles() *AppError {
	return New(CodeMenuHasRoles, CodeMenuHasRoles.DefaultMessage())
}

func MenuInvalid() *AppError {
	return New(CodeMenuInvalid, CodeMenuInvalid.DefaultMessage())
}

func MenuIsParent() *AppError {
	return New(CodeMenuIsParent, CodeMenuIsParent.DefaultMessage())
}

func MenuTypeInvalid() *AppError {
	return New(CodeMenuTypeInvalid, CodeMenuTypeInvalid.DefaultMessage())
}

func MenuHasButtons() *AppError {
	return New(CodeMenuHasButtons, CodeMenuHasButtons.DefaultMessage())
}

func DictTypeNotFound() *AppError {
	return New(CodeDictTypeNotFound, CodeDictTypeNotFound.DefaultMessage())
}

func DictTypeExists() *AppError {
	return New(CodeDictTypeExists, CodeDictTypeExists.DefaultMessage())
}

func DictDataNotFound() *AppError {
	return New(CodeDictDataNotFound, CodeDictDataNotFound.DefaultMessage())
}

func DictDataExists() *AppError {
	return New(CodeDictDataExists, CodeDictDataExists.DefaultMessage())
}

func DictTypeInUse() *AppError {
	return New(CodeDictTypeInUse, CodeDictTypeInUse.DefaultMessage())
}

func ConfigNotFound() *AppError {
	return New(CodeConfigNotFound, CodeConfigNotFound.DefaultMessage())
}

func ConfigExists() *AppError {
	return New(CodeConfigExists, CodeConfigExists.DefaultMessage())
}

func ConfigReserved() *AppError {
	return New(CodeConfigReserved, CodeConfigReserved.DefaultMessage())
}

func NoticeNotFound() *AppError {
	return New(CodeNoticeNotFound, CodeNoticeNotFound.DefaultMessage())
}

// ============================================================================
// 运维管理模块错误 - 便捷构造函数
// ============================================================================

func BuildingNotFound() *AppError {
	return New(CodeBuildingNotFound, CodeBuildingNotFound.DefaultMessage())
}

func BuildingNotFoundWithID(id string) *AppError {
	return New(CodeBuildingNotFound, fmt.Sprintf("楼宇不存在: %s", id))
}

func BuildingExists() *AppError {
	return New(CodeBuildingExists, CodeBuildingExists.DefaultMessage())
}

func BuildingExistsWithMsg(msg string) *AppError {
	return New(CodeBuildingExists, msg)
}

func BuildingHasFloors() *AppError {
	return New(CodeBuildingHasFloors, CodeBuildingHasFloors.DefaultMessage())
}

func BuildingInvalid() *AppError {
	return New(CodeBuildingInvalid, CodeBuildingInvalid.DefaultMessage())
}

func BuildingOrgInvalid() *AppError {
	return New(CodeBuildingOrgInvalid, CodeBuildingOrgInvalid.DefaultMessage())
}

func BuildingOrgInvalidWithMsg(msg string) *AppError {
	return New(CodeBuildingOrgInvalid, msg)
}

func FloorNotFound() *AppError {
	return New(CodeFloorNotFound, CodeFloorNotFound.DefaultMessage())
}

func FloorNotFoundWithID(id string) *AppError {
	return New(CodeFloorNotFound, fmt.Sprintf("楼层不存在: %s", id))
}

func FloorExists() *AppError {
	return New(CodeFloorExists, CodeFloorExists.DefaultMessage())
}

func FloorHasWorkstations() *AppError {
	return New(CodeFloorHasWorkstations, CodeFloorHasWorkstations.DefaultMessage())
}

func FloorInvalid() *AppError {
	return New(CodeFloorInvalid, CodeFloorInvalid.DefaultMessage())
}

func WorkstationNotFound() *AppError {
	return New(CodeWorkstationNotFound, CodeWorkstationNotFound.DefaultMessage())
}

func WorkstationNotFoundWithID(id string) *AppError {
	return New(CodeWorkstationNotFound, fmt.Sprintf("工位不存在: %s", id))
}

func WorkstationExists() *AppError {
	return New(CodeWorkstationExists, CodeWorkstationExists.DefaultMessage())
}

func WorkstationInvalid() *AppError {
	return New(CodeWorkstationInvalid, CodeWorkstationInvalid.DefaultMessage())
}

func ServerRoomNotFound() *AppError {
	return New(CodeServerRoomNotFound, CodeServerRoomNotFound.DefaultMessage())
}

func ServerRoomExists() *AppError {
	return New(CodeServerRoomExists, CodeServerRoomExists.DefaultMessage())
}

func ServerRoomHasServers() *AppError {
	return New(CodeServerRoomHasServers, CodeServerRoomHasServers.DefaultMessage())
}

func RoomDeviceNotFound() *AppError {
	return New(CodeRoomDeviceNotFound, CodeRoomDeviceNotFound.DefaultMessage())
}

func DeviceCodeAlreadyExists() *AppError {
	return New(CodeRoomDeviceCodeExists, CodeRoomDeviceCodeExists.DefaultMessage())
}

func InfoPointNotFound() *AppError {
	return New(CodeInfoPointNotFound, CodeInfoPointNotFound.DefaultMessage())
}

func InfoPointExists() *AppError {
	return New(CodeInfoPointExists, CodeInfoPointExists.DefaultMessage())
}

// ============================================================================
// 调度任务模块错误 - 便捷构造函数
// ============================================================================

func JobNotFound() *AppError {
	return New(CodeJobNotFound, CodeJobNotFound.DefaultMessage())
}

func JobExists() *AppError {
	return New(CodeJobExists, CodeJobExists.DefaultMessage())
}

func JobIsRunning() *AppError {
	return New(CodeJobIsRunning, CodeJobIsRunning.DefaultMessage())
}

func JobHasCron() *AppError {
	return New(CodeJobHasCron, CodeJobHasCron.DefaultMessage())
}

func CronInvalid() *AppError {
	return New(CodeCronInvalid, CodeCronInvalid.DefaultMessage())
}

func CronInvalidWithMsg(msg string) *AppError {
	return New(CodeCronInvalid, fmt.Sprintf("Cron表达式无效: %s", msg))
}

// ============================================================================
// 工单模块错误 - 便捷构造函数
// ============================================================================

func WorkorderNotFound() *AppError {
	return New(CodeWorkorderNotFound, CodeWorkorderNotFound.DefaultMessage())
}

func WorkorderExists() *AppError {
	return New(CodeWorkorderExists, CodeWorkorderExists.DefaultMessage())
}

func WorkorderInvalid() *AppError {
	return New(CodeWorkorderInvalid, CodeWorkorderInvalid.DefaultMessage())
}

func WorkorderStatus() *AppError {
	return New(CodeWorkorderStatus, CodeWorkorderStatus.DefaultMessage())
}

func WorkorderStatusWithMsg(msg string) *AppError {
	return New(CodeWorkorderStatus, msg)
}

// ============================================================================
// 监控模块错误 - 便捷构造函数
// ============================================================================

func MonitorDataNotFound() *AppError {
	return New(CodeMonitorDataNotFound, CodeMonitorDataNotFound.DefaultMessage())
}

// ============================================================================
// 网络设备模块错误 - 便捷构造函数
// ============================================================================

func NetworkDeviceNotFound() *AppError {
	return New(CodeNetworkDeviceNotFound, CodeNetworkDeviceNotFound.DefaultMessage())
}

func NetworkDeviceExists() *AppError {
	return New(CodeNetworkDeviceExists, CodeNetworkDeviceExists.DefaultMessage())
}

func NetworkDeviceConnect() *AppError {
	return New(CodeNetworkDeviceConnect, CodeNetworkDeviceConnect.DefaultMessage())
}

// ============================================================================
// 知识库模块错误 - 便捷构造函数
// ============================================================================

func KnowledgeNotFound() *AppError {
	return New(CodeKnowledgeNotFound, CodeKnowledgeNotFound.DefaultMessage())
}

func KnowledgeExists() *AppError {
	return New(CodeKnowledgeExists, CodeKnowledgeExists.DefaultMessage())
}

// ============================================================================
// 值班管理模块错误 - 便捷构造函数
// ============================================================================

func DutyNotFound() *AppError {
	return New(CodeDutyNotFound, CodeDutyNotFound.DefaultMessage())
}

func DutyExists() *AppError {
	return New(CodeDutyExists, CodeDutyExists.DefaultMessage())
}

// ============================================================================
// AD域管理模块错误 - 便捷构造函数
// ============================================================================

func ADConfigNotFound() *AppError {
	return New(CodeADConfigNotFound, CodeADConfigNotFound.DefaultMessage())
}

func ADConfigExists() *AppError {
	return New(CodeADConfigExists, CodeADConfigExists.DefaultMessage())
}

func ADConnectionFailed(err error) *AppError {
	return Wrap(err, CodeADConnectionFailed, CodeADConnectionFailed.DefaultMessage())
}

func ADSyncFailed(err error) *AppError {
	return Wrap(err, CodeADSyncFailed, CodeADSyncFailed.DefaultMessage())
}

// ============================================================================
// 系统设置模块错误 - 便捷构造函数
// ============================================================================

func SettingsNotFound() *AppError {
	return New(CodeSettingsNotFound, CodeSettingsNotFound.DefaultMessage())
}

func DashboardNotFound() *AppError {
	return New(CodeDashboardNotFound, CodeDashboardNotFound.DefaultMessage())
}

// ============================================================================
// 运维管理模块补充 - 便捷构造函数
// ============================================================================

func DoorNotFound() *AppError {
	return New(CodeDoorNotFound, CodeDoorNotFound.DefaultMessage())
}

func DoorExists() *AppError {
	return New(CodeDoorExists, CodeDoorExists.DefaultMessage())
}

func WallNotFound() *AppError {
	return New(CodeWallNotFound, CodeWallNotFound.DefaultMessage())
}

func WallExists() *AppError {
	return New(CodeWallExists, CodeWallExists.DefaultMessage())
}

func DedicatedLineNotFound() *AppError {
	return New(CodeDedicatedLineNotFound, CodeDedicatedLineNotFound.DefaultMessage())
}

func DedicatedLineExists() *AppError {
	return New(CodeDedicatedLineExists, CodeDedicatedLineExists.DefaultMessage())
}

func FloorPlanTextNotFound() *AppError {
	return New(CodeFloorPlanTextNotFound, CodeFloorPlanTextNotFound.DefaultMessage())
}

func RoomPhotoNotFound() *AppError {
	return New(CodeRoomPhotoNotFound, CodeRoomPhotoNotFound.DefaultMessage())
}

func RoomPhotoExists() *AppError {
	return New(CodeRoomPhotoExists, CodeRoomPhotoExists.DefaultMessage())
}

// ============================================================================
// 监控模块补充 - 便捷构造函数
// ============================================================================

func CacheKeyNotFound() *AppError {
	return New(CodeCacheKeyNotFound, CodeCacheKeyNotFound.DefaultMessage())
}

func CacheKeyNotFoundWithMsg(msg string) *AppError {
	return New(CodeCacheKeyNotFound, msg)
}

func CacheOperationFailed(err error) *AppError {
	return Wrap(err, CodeCacheOperationFailed, CodeCacheOperationFailed.DefaultMessage())
}

func ServerInfoNotFound() *AppError {
	return New(CodeServerInfoNotFound, CodeServerInfoNotFound.DefaultMessage())
}

// ============================================================================
// 网络设备模块补充 - 便捷构造函数
// ============================================================================

func TemplateNotFound() *AppError {
	return New(CodeTemplateNotFound, CodeTemplateNotFound.DefaultMessage())
}

func TemplateExists() *AppError {
	return New(CodeTemplateExists, CodeTemplateExists.DefaultMessage())
}

func CredentialNotFound() *AppError {
	return New(CodeCredentialNotFound, CodeCredentialNotFound.DefaultMessage())
}

func CredentialExists() *AppError {
	return New(CodeCredentialExists, CodeCredentialExists.DefaultMessage())
}

func CommandFailed(err error) *AppError {
	return Wrap(err, CodeCommandFailed, CodeCommandFailed.DefaultMessage())
}

func CommandFailedWithMsg(msg string, err error) *AppError {
	return Wrap(err, CodeCommandFailed, msg)
}

func BackupNotFound() *AppError {
	return New(CodeBackupNotFound, CodeBackupNotFound.DefaultMessage())
}

func PortCollectionFailed(err error) *AppError {
	return Wrap(err, CodePortCollectionFailed, CodePortCollectionFailed.DefaultMessage())
}

func DiscoveryFailed(err error) *AppError {
	return Wrap(err, CodeDiscoveryFailed, CodeDiscoveryFailed.DefaultMessage())
}

// ============================================================================
// VDI 模块错误 - 便捷构造函数
// ============================================================================

func VDIServerNotFound(id string) *AppError {
	return New(CodeVDIServerNotFound, fmt.Sprintf("VDI 服务器不存在: %s", id))
}

func VDIServerExists() *AppError {
	return New(CodeVDIServerExists, CodeVDIServerExists.DefaultMessage())
}

func VDIApiFailed(err error) *AppError {
	return Wrap(err, CodeVDIApiFailed, CodeVDIApiFailed.DefaultMessage())
}

func VDIAuthFailed() *AppError {
	return New(CodeVDIAuthFailed, CodeVDIAuthFailed.DefaultMessage())
}

func VDITokenExpired() *AppError {
	return New(CodeVDITokenExpired, CodeVDITokenExpired.DefaultMessage())
}

func VMNotFound(vmID string) *AppError {
	return New(CodeVMNotFound, fmt.Sprintf("虚拟机不存在: %s", vmID))
}

func VMExists() *AppError {
	return New(CodeVMExists, CodeVMExists.DefaultMessage())
}

func VMOperationFailed(operation string, err error) *AppError {
	return Wrap(err, CodeVMOperationFailed, fmt.Sprintf("虚拟机%s失败", operation))
}

func VMInconsistentState(vmID string) *AppError {
	return New(CodeVMInconsistentState, fmt.Sprintf("虚拟机状态不一致: %s", vmID))
}
