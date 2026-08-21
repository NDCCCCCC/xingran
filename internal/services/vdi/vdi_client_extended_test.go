package vdi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// Phase 74-06: vdi_client_extended.go + vdi_auth_manager.go 测试。
// 通过 fakeVDIExtServer（httptest）驱动真实 HTTP 路径。
// =====================================================================

// newExtClientWithServer 建 sqlite + 假 VDI API + 指向它的 extended client。
func newExtClientWithServer(t *testing.T, invalidOnce bool) (*fakeVDIExtServer, *vdiClientExtendedImpl, *gorm.DB) {
	t.Helper()
	fake := newFakeVDIExtServer(t, invalidOnce)
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s1", fake.URL, 0)
	client := NewVDIClientFromDB(db, "srv1")
	impl, ok := client.(*vdiClientExtendedImpl)
	require.True(t, ok)
	return fake, impl, db
}

func TestVDIClientExtended_Constructors(t *testing.T) {
	db := newVDITestDB(t)
	ctx := context.Background()

	// 记录不存在 → 空 client，Authenticate 报 VDI server not found
	missing := NewVDIClientFromDB(db, "missing")
	_, err := missing.Authenticate(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VDI server not found: missing")

	legacy := NewVDIClientExtended(db, "missing", nil)
	_, err = legacy.Authenticate(ctx)
	require.Error(t, err)
}

func TestVDIAuthManager_Authenticate(t *testing.T) {
	fake := newFakeVDIExtServer(t, false)
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s1", fake.URL, 0)
	ctx := context.Background()

	// 1. 缓存 token 未过期 → 直接命中，无 HTTP
	var server models.VDIServer
	require.NoError(t, db.Where("id = ?", "srv1").First(&server).Error)
	server.AuthToken = "cached-token"
	expiry := time.Now().Add(time.Hour)
	server.TokenExpiry = &expiry
	mgr := NewVDIAuthManager(db, "srv1", server)
	tok, err := mgr.Authenticate(ctx)
	require.NoError(t, err)
	assert.Equal(t, "cached-token", tok)
	assert.Equal(t, 0, fake.authCount())

	// 2. 过期 → 重新认证 + 写回 DB
	server.AuthToken = ""
	past := time.Now().Add(-time.Hour)
	server.TokenExpiry = &past
	mgr2 := NewVDIAuthManager(db, "srv1", server)
	tok, err = mgr2.Authenticate(ctx)
	require.NoError(t, err)
	assert.Equal(t, "tok-1", tok)
	var dbToken string
	var dbExpiry *time.Time
	require.NoError(t, db.Raw(`SELECT auth_token FROM sys_vdi_server WHERE id = 'srv1'`).Scan(&dbToken).Error)
	assert.Equal(t, "tok-1", dbToken)
	require.NoError(t, db.Raw(`SELECT token_expiry FROM sys_vdi_server WHERE id = 'srv1'`).Scan(&dbExpiry).Error)
	require.NotNil(t, dbExpiry)

	// IsTokenExpired 从 DB 读（用 GORM Update 写 time.Time，与 cacheToken 格式一致）
	further := time.Now().Add(10 * time.Hour)
	require.NoError(t, db.Model(&models.VDIServer{}).Where("id = ?", "srv1").Update("token_expiry", further).Error)
	assert.False(t, mgr2.IsTokenExpired(ctx))
	require.NoError(t, db.Model(&models.VDIServer{}).Where("id = ?", "srv1").Update("token_expiry", time.Now().Add(-time.Hour)).Error)
	assert.True(t, mgr2.IsTokenExpired(ctx))

	// RefreshToken == Authenticate
	tok, err = mgr2.RefreshToken(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	// ClearTokenCache 后重新认证
	mgr2.ClearTokenCache()
	tok, err = mgr2.Authenticate(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	// 未导出的 callAPI 直连（callAPIWithEndpoint 包装）
	var apiResp struct {
		ErrorCode int `json:"error_code"`
	}
	err = mgr2.callAPI(ctx, "/v1/auth/tokens", map[string]interface{}{
		"auth": map[string]string{"name": "admin", "password": "secret"},
	}, &apiResp)
	require.NoError(t, err)
	assert.Zero(t, apiResp.ErrorCode)

	// 3. 密码解密失败（空）→ 报错
	server2 := models.VDIServer{}
	server2.ID = "srv1"
	server2.Username = "admin"
	server2.PasswordEncrypted = "not-base64!!!" // decrypt 返回 ""
	server2.Endpoint = fake.URL
	mgr3 := NewVDIAuthManager(db, "srv1", server2)
	_, err = mgr3.Authenticate(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrypt VDI server password")

	// 4. 认证 401 → unexpected status code
	fake.mu.Lock()
	fake.authFail = true
	fake.mu.Unlock()
	failSrv := models.VDIServer{}
	failSrv.ID = "srv1"
	failSrv.Username = "admin"
	failSrv.PasswordEncrypted = encryptVDIPassword("secret")
	failSrv.Endpoint = fake.URL
	mgr4 := NewVDIAuthManager(db, "srv1", failSrv)
	_, err = mgr4.Authenticate(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")

	// 5. error_code != 0 → VDIError
	fake.mu.Lock()
	fake.authFail = false
	fake.mu.Unlock()
	// 复用 /v1/auth/tokens 正常响应即可；error_code 非零路径由 callAPI 层覆盖（见 callAPI 测试）
}

func TestVDIClientExtended_Queries(t *testing.T) {
	fake, client, _ := newExtClientWithServer(t, false)
	ctx := context.Background()

	// GetVM
	detail, err := client.GetVM(ctx, "101")
	require.NoError(t, err)
	assert.Equal(t, "vm-alpha", detail.Name)

	// ListVMs（rc_id 过滤）
	sums, err := client.ListVMs(ctx, "1")
	require.NoError(t, err)
	require.Len(t, sums, 1)

	// ListVMs（无过滤）
	sums, err = client.ListVMs(ctx, "")
	require.NoError(t, err)
	require.Len(t, sums, 1)

	// GetUserVMs
	sums, err = client.GetUserVMs(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, sums, 1)

	// GetAvailableUsers → 恒空（VDI 无此端点）
	users, err := client.GetAvailableUsers(ctx, "101")
	require.NoError(t, err)
	assert.Empty(t, users)

	// ListResourceGroups
	groups, err := client.ListResourceGroups(ctx)
	require.NoError(t, err)
	require.Len(t, groups, 3)

	// ListResources
	res, err := client.ListResources(ctx, "g1")
	require.NoError(t, err)
	require.Len(t, res, 2)

	// ListResourceServers：totalCount 字符串解析
	vms, total, err := client.ListResourceServers(ctx, "1", 1, 100)
	require.NoError(t, err)
	require.Len(t, vms, 2)
	assert.Equal(t, 2, total)

	// GetRunPositions / GetStorages / GetNetworks / CreateServer
	pos, err := client.GetRunPositions(ctx, 7)
	require.NoError(t, err)
	require.Len(t, pos, 1)

	st, err := client.GetStorages(ctx, 7)
	require.NoError(t, err)
	require.Len(t, st, 1)

	nw, err := client.GetNetworks(ctx, 7)
	require.NoError(t, err)
	require.Len(t, nw, 1)

	resp, err := client.CreateServer(ctx, CreateServerRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Data.ServerID, 1)
	assert.Equal(t, 9, resp.Data.TaskID)

	assert.GreaterOrEqual(t, fake.authCount(), 1)
}

func TestVDIClientExtended_OperateDeleteBind(t *testing.T) {
	fake, client, _ := newExtClientWithServer(t, false)
	ctx := context.Background()

	// OperateVM: 请求体格式 {"action":{"name":...},"servers":{"ids":"a,b"}}
	require.NoError(t, client.OperateVM(ctx, []string{"101", "102"}, "restart"))
	fake.mu.Lock()
	body := fake.lastBodies["/v1/servers/action"]
	fake.mu.Unlock()
	var sent struct {
		Action  map[string]string `json:"action"`
		Servers map[string]string `json:"servers"`
	}
	require.NoError(t, json.Unmarshal(body, &sent))
	assert.Equal(t, "reboot", sent.Action["name"])
	assert.Equal(t, "101,102", sent.Servers["ids"])

	// DeleteVM
	require.NoError(t, client.DeleteVM(ctx, []string{"101", "102"}))

	// BindUser
	detail, err := client.BindUser(ctx, "101", "u-1")
	require.NoError(t, err)
	assert.Equal(t, "vm-alpha", detail.Name)

	// 不支持的 action
	err = client.OperateVM(ctx, []string{"101"}, "explode")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported VDI VM action")
}

func TestOperateActionToName(t *testing.T) {
	cases := map[string]string{
		"start": "startup", "stop": "shutdown", "restart": "reboot", "suspend": "suspend",
	}
	for in, want := range cases {
		got, err := operateActionToName(in)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := operateActionToName("bogus")
	require.Error(t, err)
}

func TestVDIClientExtended_GetVTPPlatforms(t *testing.T) {
	_, client, _ := newExtClientWithServer(t, false)
	ctx := context.Background()

	platforms, err := client.GetVTPPlatforms(ctx)
	require.NoError(t, err)
	// rc1: vtp 7/7 → 唯一; rc3: vtp 8 → 共 2 个平台
	require.Len(t, platforms, 2)
	names := map[int]string{}
	for _, p := range platforms {
		names[p.ID] = p.Name
	}
	assert.Equal(t, "VMP-A", names[7])
	assert.Equal(t, "VMP-B", names[8])
}

func TestVDIClientExtended_AuthTokenInvalidRetry(t *testing.T) {
	fake, client, _ := newExtClientWithServer(t, true)
	ctx := context.Background()

	// invalidOnce=true: 第一个 /v1/servers 请求（token=tok-1）返回 1101 → 清缓存重认证 → 重试成功
	sums, err := client.ListVMs(ctx, "")
	require.NoError(t, err)
	require.Len(t, sums, 1)
	assert.GreaterOrEqual(t, fake.authCount(), 2, "AUTH_TOKEN_INVALID 后应重新认证")
}

func TestVDIClientExtended_Failures(t *testing.T) {
	_, client, _ := newExtClientWithServer(t, false)
	ctx := context.Background()

	// authManager 缺失（记录不存在时构造的空 client）→ Authenticate 报错
	client.authManager = nil
	_, err := client.GetVM(ctx, "101")
	require.Error(t, err)

	_, client2, _ := newExtClientWithServer(t, false)
	// 坏 endpoint → 网络错误
	client2.server.Endpoint = "http://127.0.0.1:1"
	_, err = client2.ListVMs(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send request failed")
}

func TestVDIError_Error(t *testing.T) {
	e := &VDIError{Code: 42, Message: "boom"}
	assert.Contains(t, e.Error(), "42")
	assert.Contains(t, e.Error(), "boom")
}

// =====================================================================
// 真实 HTTP 全链路：ListVMs 触发 syncVMsFromVDI（资源组/VM 同步 + 孤儿清理）
// =====================================================================

func TestVMService_FullSyncViaListVMs(t *testing.T) {
	fake := newFakeVDIExtServer(t, false)
	db := newVDITestDB(t)
	seedVDIServer(t, db, "srv1", "s1", fake.URL, 0)
	ctx := context.Background()

	// 本地表空 → ListVMs 自动触发同步
	svc := NewVMService(db, nil) // 动态 client → 指向 fake
	page, err := svc.ListVMs(ctx, &ListVMRequest{}, "", 0)
	require.NoError(t, err)
	// g1(2 VMs) + g2(1 VM) = 3
	assert.Equal(t, int64(3), page.Total)

	// VM 字段映射：assign_ip 优先 / power_state / ip_type
	var ip, state, ipType string
	require.NoError(t, db.Raw(`SELECT ip_address, power_state, ip_type FROM sys_vdi_vm WHERE vm_id = '101'`).Row().Scan(&ip, &state, &ipType))
	assert.Equal(t, "192.168.1.101", ip, "assign_ip 优先")
	assert.Equal(t, "in_use", state)
	assert.Equal(t, "STATIC", ipType)

	var ip2 string
	require.NoError(t, db.Raw(`SELECT ip_address FROM sys_vdi_vm WHERE vm_id = '102'`).Scan(&ip2).Error)
	assert.Equal(t, "10.0.0.2", ip2, "assign_ip 为 '-' 回退到 ip")

	// 资源组同步：g1/g2 启用(0)、g3 停用(1)
	var rgCnt int
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_vdi_resource_group WHERE deleted_at IS NULL`).Scan(&rgCnt).Error)
	assert.Equal(t, 3, rgCnt)
	var g3Status int
	require.NoError(t, db.Raw(`SELECT status FROM sys_vdi_resource_group WHERE resource_group_id = 'g3'`).Scan(&g3Status).Error)
	assert.Equal(t, 1, g3Status)

	// 二次同步（SyncVMsFromVDIByServer）→ 走更新分支 + 清理孤儿
	require.NoError(t, db.Exec(`UPDATE sys_vdi_vm SET name = 'stale' WHERE vm_id = '101'`).Error)
	insertVM(t, db, "ghost", "v-ghost", "ghost", "srv1")
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_resource_group (id, resource_group_id, name, vdi_server_id, type, status, created_at)
		VALUES ('rg-old','g-gone','old','srv1','',0,?)`, time.Now().Format("2006-01-02 15:04:05")).Error)

	server := &models.VDIServer{}
	server.ID = "srv1"
	server.Name = "s1"
	require.NoError(t, svc.SyncVMsFromVDIByServer(ctx, server))

	var name string
	require.NoError(t, db.Raw(`SELECT name FROM sys_vdi_vm WHERE vm_id = '101' AND deleted_at IS NULL`).Scan(&name).Error)
	assert.Equal(t, "vm-alpha", name, "同步应刷新名称")

	var cnt int
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_vdi_vm WHERE id = 'ghost' AND deleted_at IS NULL`).Scan(&cnt).Error)
	assert.Zero(t, cnt, "孤儿 VM 被软删")
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM sys_vdi_resource_group WHERE deleted_at IS NULL`).Scan(&cnt).Error)
	assert.Equal(t, 3, cnt, "孤儿资源组被清理")

	// 无启用的 VDI 服务器 → getClient 报错（经 ListResources 触发）
	require.NoError(t, db.Exec(`UPDATE sys_vdi_server SET status = 1`).Error)
	dyn := NewVMService(db, nil)
	_, err = dyn.ListResources(ctx, "", "g1")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no enabled VDI server") || strings.Contains(err.Error(), "failed to query VDI server"))
}
