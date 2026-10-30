package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
)

// =====================================================================
// Phase 74-07 收尾补缺:profile ChangePassword/UploadAvatar 成功路径 +
// department Update 分支矩阵 / fillLeaderInfo / checkDeptCodeExists。
// =====================================================================

func TestProfileService_ChangePassword_Success(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db).(*profileService)
	ctx := context.Background()

	// 用真实 PasswordManager 造存量 hash
	oldHash, err := svc.passwordMgr.HashPassword("OldPass123")
	require.NoError(t, err)
	userID := seedProfileSvcUser(t, db, "carol", oldHash)

	require.NoError(t, svc.ChangePassword(ctx, userID, "OldPass123", "NewPass456"))

	var row struct {
		Password     string
		PwdStartTime *string
		InitFlag     *bool
	}
	require.NoError(t, db.Raw(
		`SELECT password, pwd_update_time AS pwd_start_time, init_flag FROM sys_user WHERE id = ?`, userID,
	).Scan(&row).Error)
	ok, err := svc.passwordMgr.VerifyPassword("NewPass456", row.Password)
	require.NoError(t, err)
	assert.True(t, ok, "密码应更新为新哈希")
	assert.NotNil(t, row.PwdStartTime, "pwd_update_time 应写入")
	require.NotNil(t, row.InitFlag)
	assert.False(t, *row.InitFlag, "init_flag 应置 false")
}

func TestProfileService_ChangePassword_WrongOld(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db).(*profileService)

	hash, err := svc.passwordMgr.HashPassword("right")
	require.NoError(t, err)
	userID := seedProfileSvcUser(t, db, "dave", hash)

	err = svc.ChangePassword(context.Background(), userID, "wrong", "new")
	require.ErrorIs(t, err, ErrOldPasswordIncorrect, "sentinel error 应可被 errors.Is 识别")
}

func TestProfileService_UploadAvatar(t *testing.T) {
	db := setupProfileServiceDB(t)
	svc := NewProfileService(db).(*profileService)
	svc.uploadBaseDir = filepath.Join(t.TempDir(), "avatar")
	ctx := context.Background()
	userID := seedProfileSvcUser(t, db, "erin", "x")

	// 成功:落盘 + 更新 avatar 列
	url, err := svc.UploadAvatar(ctx, userID, fileHeaderFor(t, tinyPNG(t, 2, 2), "head.png", "image/png"))
	require.NoError(t, err)
	assert.Contains(t, url, "/uploads/avatar/")
	assert.Contains(t, url, userID)
	files, err := filepath.Glob(filepath.Join(svc.uploadBaseDir, userID+"_*"))
	require.NoError(t, err)
	assert.Len(t, files, 1, "文件应落盘")

	var avatar string
	require.NoError(t, db.Raw(`SELECT avatar FROM sys_user WHERE id = ?`, userID).Scan(&avatar).Error)
	assert.Equal(t, url, avatar)

	// 非法扩展名
	_, err = svc.UploadAvatar(ctx, userID, fileHeaderFor(t, []byte("x"), "virus.exe", ""))
	require.ErrorContains(t, err, "不支持的文件类型")
	// 未上传用户也无影响(错误在扩展名检查,不触库)
	_ = os.Remove(files[0])
}

func TestDeptService_Update_BranchMatrix(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	ctx := context.Background()

	rootID := seedDeptDirect(t, db, &models.Department{DeptName: "Root", DeptCode: "R", Status: models.DeptStatusNormal})
	otherID := seedDeptDirect(t, db, &models.Department{DeptName: "Other", DeptCode: "O", Status: models.DeptStatusNormal})
	childID := seedDeptDirect(t, db, &models.Department{
		DeptName: "Child", DeptCode: "C", Ancestors: rootID, Status: models.DeptStatusNormal,
	})
	// 同级重名目标(Other 下已有 "Dup")—— ParentID 必须显式给,否则 GORM 写 NULL,
	// checkDeptNameExists 用 parent_id=Other 匹配时查不到。
	dupPID := otherID
	seedDeptDirect(t, db, &models.Department{
		DeptName: "Dup", DeptCode: "DUP", ParentID: &dupPID, Ancestors: otherID, Status: models.DeptStatusNormal,
	})

	// 改编码撞他人 → 拦截
	err := svc.Update(ctx, &requests.DepartmentUpdateRequest{
		ID: childID, DeptName: "Child", DeptCode: "R", Status: models.DeptStatusNormal,
	})
	require.ErrorContains(t, err, "部门编码已存在")

	// 编码不变(自身相同)→ 放行不查重
	require.NoError(t, svc.Update(ctx, &requests.DepartmentUpdateRequest{
		ID: childID, DeptName: "Child", DeptCode: "C", Status: models.DeptStatusNormal,
	}))

	// 同级重名 → 拦截(Child 挂到 Other 下并改名 Dup)
	other := otherID
	err = svc.Update(ctx, &requests.DepartmentUpdateRequest{
		ID: childID, DeptName: "Dup", DeptCode: "C2", ParentID: &other, Status: models.DeptStatusNormal,
	})
	require.ErrorContains(t, err, "同级部门名称已存在")

	// 换父成功 → ancestors 重建为 Other
	require.NoError(t, svc.Update(ctx, &requests.DepartmentUpdateRequest{
		ID: childID, DeptName: "Child2", DeptCode: "C2", ParentID: &other, Status: models.DeptStatusNormal,
	}))
	var got models.Department
	require.NoError(t, db.First(&got, "id = ?", childID).Error)
	assert.Equal(t, otherID, got.Ancestors, "换父后祖先链应重建")

	// 父不存在 → buildAncestors 报错
	ghost := "no-such-parent"
	err = svc.Update(ctx, &requests.DepartmentUpdateRequest{
		ID: childID, DeptName: "Child3", DeptCode: "C3", ParentID: &ghost, Status: models.DeptStatusNormal,
	})
	require.ErrorContains(t, err, "构建祖先链失败")
}

func TestDeptService_FillLeaderInfo(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	ctx := context.Background()

	// fillLeaderInfo Select nickname → setup 表缺该列,先补
	require.NoError(t, db.Exec(`ALTER TABLE sys_user ADD COLUMN nickname TEXT`).Error)

	leaderID := seedUserWithNickname(t, db, "boss", "王领导")
	plainID := seedUserWithNickname(t, db, "plain", "张员工")

	depts := []models.Department{
		{DeptName: "A", Leader: &leaderID},
		{DeptName: "B", Leader: strPtr("not-a-uuid")}, // 非 UUID → 过滤
		{DeptName: "C", Leader: &plainID},
		{DeptName: "D"}, // 无 leader
	}
	svc.fillLeaderInfo(ctx, depts)

	require.NotNil(t, depts[0].LeaderName)
	assert.Equal(t, "王领导", *depts[0].LeaderName)
	require.NotNil(t, depts[0].LeaderUsername)
	assert.Equal(t, "boss", *depts[0].LeaderUsername)
	assert.Nil(t, depts[1].LeaderName, "非 UUID leader 应被过滤")
	require.NotNil(t, depts[2].LeaderName)
	assert.Equal(t, "张员工", *depts[2].LeaderName)
	assert.Nil(t, depts[3].LeaderName)

	// 全空 leader → 直接返回
	svc.fillLeaderInfo(ctx, []models.Department{{DeptName: "X"}})
}

func seedUserWithNickname(t *testing.T, db *gorm.DB, username, nickname string) string {
	t.Helper()
	id := uuid.NewString()
	require.NoError(t, db.Exec(
		`INSERT INTO sys_user (id, username, nickname) VALUES (?, ?, ?)`, id, username, nickname,
	).Error)
	return id
}

func TestDeptService_CheckDeptCodeExists(t *testing.T) {
	db := setupDeptServiceDB(t)
	svc := NewDepartmentService(db).(*departmentService)
	ctx := context.Background()

	id := seedDeptDirect(t, db, &models.Department{DeptName: "D", DeptCode: "CODE1", Status: models.DeptStatusNormal})

	exists, err := svc.checkDeptCodeExists(ctx, "CODE1", "")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = svc.checkDeptCodeExists(ctx, "CODE1", id)
	require.NoError(t, err)
	assert.False(t, exists, "排除自身")

	exists, err = svc.checkDeptCodeExists(ctx, "NOPE", "")
	require.NoError(t, err)
	assert.False(t, exists)
}
