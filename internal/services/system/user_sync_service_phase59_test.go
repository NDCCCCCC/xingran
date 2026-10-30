package system

// Phase 59 GUARD-01：AD 登录 dept_id 回写的活跃队列守卫。
//
// 守卫只在 SyncADUser 的「OU 解析出部门」分支内生效（user_sync_service.go 的 else-if 链）。
// 因此 fixture 必须让执行真正走进该分支，否则「dept 未变」类断言会空跑通过：
//   - 用户必须被 `ad_username` 精确命中（classifyUser → catActiveAD），否则落 catLocalSameName
//     在上游就提前返回，永远碰不到守卫；
//   - adUserInfo.OUDN 必须与映射行的 ou_dn 逐字符相等（FindDeptByOUDN 是精确匹配，不做归一化）；
//   - 映射行 sync_enabled=1 且 JOIN 到的 sys_dept 未软删；
//   - 构造 service 必须传真实 *addomain.DeptOUmapper，传 nil 会让整个分支不执行。
//
// 「正向」用例（无行 / 终态行 / disable 行 → dept 被改成映射部门 ID）是 fixture 存活的锚点：
// 只有它们观察到 dept 发生变化，才能证明「dept 未变」的拦截类用例不是空跑。

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/core/db/migrations"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// phase59OUdn 与 fixture 写入 sys_dept_ou_mapping.ou_dn 的值逐字符相等。
const phase59OUdn = "OU=IT,DC=corp,DC=com"

// phase59GuardFixture 是六个守卫用例共用的测试基座。
type phase59GuardFixture struct {
	db         *gorm.DB
	svc        *UserSyncService
	userID     string
	username   string
	localDept  string // 预置的用户本地部门（管理员编辑留在队列里的那个值）
	adDept     string // AD OU 解析出的部门（本应被写入的值）
	adConfigID string
}

// setupPhase59GuardFixture 建表 + 有效部门 + sync_enabled 映射 + ad_username 非空用户 + 真实 mapper。
func setupPhase59GuardFixture(t *testing.T) *phase59GuardFixture {
	t.Helper()

	db := setupTestDBForSync(t)
	require.NotNil(t, db, "setupTestDBForSync 必须返回可用 DB")

	// 映射表：FindDeptByOUDN 需要 sys_dept_ou_mapping JOIN sys_dept。
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS sys_dept_ou_mapping (
		id TEXT PRIMARY KEY,
		dept_id TEXT NOT NULL,
		ad_config_id TEXT NOT NULL,
		ou_dn TEXT NOT NULL,
		ou_name TEXT NOT NULL,
		parent_ou_dn TEXT,
		sync_enabled INTEGER DEFAULT 1,
		sync_status TEXT DEFAULT 'pending',
		last_sync_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		UNIQUE(dept_id, ad_config_id)
	)`).Error)

	// 队列表：直调 migration_215，不手写第二份 DDL。
	require.NoError(t, migrations.Migrate215CreateADPushQueue(db))

	f := &phase59GuardFixture{
		db:         db,
		userID:     uuid.NewString(),
		username:   "phase59_" + uuid.NewString()[:8],
		localDept:  uuid.NewString(),
		adDept:     uuid.NewString(),
		adConfigID: uuid.NewString(),
	}

	// 两个未软删的有效部门：localDept 是用户当前归属，adDept 是 OU 映射目标。
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, status, deleted_at) VALUES (?, ?, 0, NULL)`,
		f.localDept, "本地部门").Error)
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept (id, dept_name, status, deleted_at) VALUES (?, ?, 0, NULL)`,
		f.adDept, "AD 映射部门").Error)

	// sync_enabled=1 的 OU → 部门映射；ou_dn 必须与 phase59OUdn 逐字符相等。
	require.NoError(t, db.Exec(
		`INSERT INTO sys_dept_ou_mapping (id, dept_id, ad_config_id, ou_dn, ou_name, sync_enabled, sync_status)
		 VALUES (?, ?, ?, ?, ?, 1, 'pending')`,
		uuid.NewString(), f.adDept, f.adConfigID, phase59OUdn, "IT").Error)

	// 用户：ad_username 非空且等于 AD 用户名 → classifyUser 命中 catActiveAD，
	// 执行路径才会走到 else-if 里的 resolveDeptFromOU + 守卫。
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, password, salt, nickname, auth_source, ad_username, dept_id, status)
		 VALUES (?, ?, '', '', ?, 'ad', ?, ?, 0)`,
		f.userID, f.username, "本地用户", f.username, f.localDept).Error)

	// 第三参必须是真实 mapper：传 nil 会让整个守卫分支不执行，测试变空跑。
	f.svc = NewUserSyncService(db, security.NewPasswordManager(nil), addomain.NewDeptOUmapper(db))
	return f
}

// adUserInfo 构造一个必然走进守卫分支的 AD 用户信息（OUDN 与映射 ou_dn 逐字符相等）。
func (f *phase59GuardFixture) adUserInfo() *security.ADUserInfo {
	return &security.ADUserInfo{
		UserDN:      "CN=" + f.username + "," + phase59OUdn,
		OUDN:        phase59OUdn,
		Username:    f.username,
		DisplayName: "AD User",
		Email:       "aduser@example.com",
		Phone:       "1234567890",
	}
}

// currentDeptID 从 DB 重新读取用户的 dept_id（不经内存缓存）。
func (f *phase59GuardFixture) currentDeptID(t *testing.T) string {
	t.Helper()
	var got string
	require.NoError(t, f.db.Raw(`SELECT dept_id FROM sys_user WHERE id = ?`, f.userID).Scan(&got).Error)
	return got
}

// seedQueueRow 插入一条队列行（走 GORM，BeforeCreate 生成 UUID 与默认值）。
func (f *phase59GuardFixture) seedQueueRow(t *testing.T, action models.ADPushActionType, status models.ADPushStatus, attempts int) string {
	t.Helper()
	row := &models.ADPushQueue{
		EntityType: models.ADPushEntityUser,
		EntityID:   f.userID,
		ActionType: action,
		Status:     status,
		Attempts:   attempts,
	}
	require.NoError(t, f.db.Create(row).Error)
	return row.ID
}

// captureLogs 把日志输出重定向到 buffer，返回读取函数与恢复函数。
func captureLogs(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	restore := applogger.SetTestBuffer(buf)
	return buf, restore
}

// TestPhase59_SyncADUser_ActivePendingQueueSkipsDept
// 存在 pending 的 update_user 队列行 → 本地 dept_id 不被 AD 解析值覆盖，且 WARN 含全量诊断字段。
func TestPhase59_SyncADUser_ActivePendingQueueSkipsDept(t *testing.T) {
	f := setupPhase59GuardFixture(t)

	queueID := f.seedQueueRow(t, models.ADPushActionUpdateUser, models.ADPushStatusPending, 3)

	buf, restore := captureLogs(t)
	defer restore()

	_, err := f.svc.SyncADUser(context.Background(), f.adUserInfo(), "role-1")
	require.NoError(t, err)

	// 守卫拦截：本地部门保持不变
	assert.Equal(t, f.localDept, f.currentDeptID(t), "pending 队列行存在时不得覆盖本地 dept_id")

	logs := buf.String()
	assert.Contains(t, logs, f.userID, "WARN 应含 userID")
	assert.Contains(t, logs, f.username, "WARN 应含 username")
	assert.Contains(t, logs, f.localDept, "WARN 应含本地 dept UUID")
	// AD 解析出的 dept UUID 出现在日志里，证明防线是在 resolveDeptFromOU 成功返回之后被守卫拦下
	assert.Contains(t, logs, f.adDept, "WARN 应含 AD 解析出的 dept UUID")
	assert.Contains(t, logs, queueID, "WARN 应含队列行 ID")
	assert.Contains(t, logs, string(models.ADPushActionUpdateUser), "WARN 应含 action_type")
	assert.Contains(t, logs, "attempts=3", "WARN 应含 attempts 数值")
}

// TestPhase59_SyncADUser_ActiveFailedQueueSkipsDept
// failed 行仍在活跃谓词内（status IN ('pending','failed')）→ 同样拦截 dept 回写。
func TestPhase59_SyncADUser_ActiveFailedQueueSkipsDept(t *testing.T) {
	f := setupPhase59GuardFixture(t)

	queueID := f.seedQueueRow(t, models.ADPushActionUpdateUser, models.ADPushStatusFailed, 2)

	buf, restore := captureLogs(t)
	defer restore()

	_, err := f.svc.SyncADUser(context.Background(), f.adUserInfo(), "role-1")
	require.NoError(t, err)

	assert.Equal(t, f.localDept, f.currentDeptID(t), "failed 队列行存在时不得覆盖本地 dept_id")

	logs := buf.String()
	assert.Contains(t, logs, f.userID)
	assert.Contains(t, logs, f.localDept)
	assert.Contains(t, logs, f.adDept)
	assert.Contains(t, logs, queueID)
	assert.Contains(t, logs, "attempts=2")
}

// TestPhase59_SyncADUser_TerminalRowsDoNotGuard
// pushed / dead 是终态行，退出活跃谓词（Phase 57 D-06）→ 不拦截，dept 照常更新。
// 该用例同时是 fixture 存活锚点：观察到 dept 变为映射部门 ID 才证明执行确实走进了守卫分支。
func TestPhase59_SyncADUser_TerminalRowsDoNotGuard(t *testing.T) {
	for _, status := range []models.ADPushStatus{models.ADPushStatusPushed, models.ADPushStatusDead} {
		t.Run(string(status), func(t *testing.T) {
			f := setupPhase59GuardFixture(t)
			f.seedQueueRow(t, models.ADPushActionUpdateUser, status, 0)

			_, err := f.svc.SyncADUser(context.Background(), f.adUserInfo(), "role-1")
			require.NoError(t, err)

			assert.Equal(t, f.adDept, f.currentDeptID(t),
				"终态行 %s 不在活跃谓词内，dept 应更新为 AD 解析值", status)
		})
	}
}

// TestPhase59_SyncADUser_DisableActionDoesNotGuard
// 守卫只认 action_type='update_user'；pending 的 disable 行不影响 dept 回写。
func TestPhase59_SyncADUser_DisableActionDoesNotGuard(t *testing.T) {
	f := setupPhase59GuardFixture(t)
	f.seedQueueRow(t, models.ADPushActionDisable, models.ADPushStatusPending, 0)

	_, err := f.svc.SyncADUser(context.Background(), f.adUserInfo(), "role-1")
	require.NoError(t, err)

	assert.Equal(t, f.adDept, f.currentDeptID(t),
		"disable 动作的 pending 行不属于守卫命中集合，dept 应更新为 AD 解析值")
}

// TestPhase59_SyncADUser_NoQueueUpdatesDept
// 无任何队列行 → 与 Phase 58 完全相同的 Update 调用，写入值等于映射部门 ID（D-08 不误伤）。
func TestPhase59_SyncADUser_NoQueueUpdatesDept(t *testing.T) {
	f := setupPhase59GuardFixture(t)

	_, err := f.svc.SyncADUser(context.Background(), f.adUserInfo(), "role-1")
	require.NoError(t, err)

	// 正向断言：证明 fixture 真正走进了守卫分支（否则「dept 未变」类用例会空跑通过）
	assert.Equal(t, f.adDept, f.currentDeptID(t), "无活跃队列行时 dept 应更新为映射部门 ID")
}

// TestPhase59_SyncADUser_QueueQueryErrorSkipsDeptAndWarns
// 守卫查询失败（模拟 DB 故障/表缺失）→ 保守跳过覆盖并 WARN，错误不上抛（D-07）。
func TestPhase59_SyncADUser_QueueQueryErrorSkipsDeptAndWarns(t *testing.T) {
	f := setupPhase59GuardFixture(t)
	require.NoError(t, f.db.Exec("DROP TABLE ad_push_queue").Error)

	buf, restore := captureLogs(t)
	defer restore()

	_, err := f.svc.SyncADUser(context.Background(), f.adUserInfo(), "role-1")
	require.NoError(t, err, "守卫查询失败不得阻断登录同步")

	assert.Equal(t, f.localDept, f.currentDeptID(t), "守卫查询失败时应保守跳过 dept 覆盖")

	logs := buf.String()
	assert.Contains(t, logs, "守卫查询失败")
	assert.Contains(t, logs, f.userID)
	assert.Contains(t, logs, f.localDept)
	assert.Contains(t, logs, f.adDept)
}
