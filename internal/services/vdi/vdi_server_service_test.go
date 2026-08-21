package vdi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestMain 设置 SM4_KEY：newVDIHTTPClient → loadTLSSkipVerify → config.Load 的
// 校验要求该环境变量存在（默认配置下无 config.yaml）。
func TestMain(m *testing.M) {
	if os.Getenv("SM4_KEY") == "" {
		_ = os.Setenv("SM4_KEY", "MTIzNDU2Nzg5MDEyMzQ1Ng==")
	}
	os.Exit(m.Run())
}

func writeJSONBody(w http.ResponseWriter, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

func readAll(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}

// =====================================================================
// Phase 74-06: vdi_server_service_impl.go + client_manager.go 测试。
// 同时提供整个 vdi 包测试共享的 sqlite 建表 helper。
// =====================================================================

// newVDITestDB 创建包含 VDI 模块全部依赖表的内存 sqlite。
func newVDITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE sys_vdi_server (
			id TEXT PRIMARY KEY,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0,
			name TEXT, endpoint TEXT, username TEXT, password_encrypted TEXT,
			tenant_id INTEGER DEFAULT 0, auth_token TEXT, token_expiry DATETIME,
			last_sync_time DATETIME, status INTEGER DEFAULT 0
		);
		CREATE TABLE sys_vdi_vm (
			id TEXT PRIMARY KEY,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0,
			vm_id TEXT NOT NULL UNIQUE,
			name TEXT, resource_id TEXT, power_state TEXT,
			ip_address TEXT, mac_address TEXT, os_type TEXT,
			cpu_number INTEGER DEFAULT 0, cpu_core INTEGER DEFAULT 0, cpu_per INTEGER DEFAULT 0,
			memory INTEGER DEFAULT 0, memory_per INTEGER DEFAULT 0,
			disk INTEGER DEFAULT 0, disk_per INTEGER DEFAULT 0,
			bound_user_id TEXT, bound_user_name TEXT, policy_group_id TEXT,
			ip_type TEXT, subnet_mask TEXT, default_gateway TEXT,
			name_server TEXT, assign_ip TEXT, last_sync_at DATETIME,
			vdi_server_id TEXT NOT NULL
		);
		CREATE TABLE sys_vdi_resource_group (
			id TEXT PRIMARY KEY,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0,
			resource_group_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL, vdi_server_id TEXT NOT NULL,
			type TEXT, status INTEGER DEFAULT 0
		);
		CREATE TABLE sys_rpa_audit_logs (
			id TEXT PRIMARY KEY,
			resource_type TEXT, resource_id TEXT, action TEXT,
			old_value TEXT, new_value TEXT,
			operator_id TEXT, operator_name TEXT, ip_address TEXT, user_agent TEXT,
			result TEXT DEFAULT 'success', error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE sys_user (
			id TEXT PRIMARY KEY,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME,
			created_by TEXT, updated_by TEXT, version INTEGER DEFAULT 0,
			username TEXT, nickname TEXT, dept_id TEXT
		);
		CREATE TABLE sys_dept (
			id TEXT PRIMARY KEY, parent_id TEXT, status INTEGER DEFAULT 0, deleted_at DATETIME
		);
		CREATE TABLE sys_user_role (user_id TEXT, role_id TEXT);
		CREATE TABLE sys_role_dept (role_id TEXT, dept_id TEXT);
	`).Error)
	return db
}

// seedVDIServer 插入一条 VDI 服务器记录。
func seedVDIServer(t *testing.T, db *gorm.DB, id, name, endpoint string, status int) {
	t.Helper()
	now := time.Now().Format("2006-01-02 15:04:05")
	require.NoError(t, db.Exec(
		`INSERT INTO sys_vdi_server (id, name, endpoint, username, password_encrypted, tenant_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'admin', ?, 1, ?, ?, ?)`,
		id, name, endpoint, encryptVDIPassword("secret"), status, now, now,
	).Error)
}

// =====================================================================
// fakeVDIExtServer: 深信服风格 VDI API 假服务器（vdiClientExtendedImpl 端点集）。
// =====================================================================

type fakeVDIExtServer struct {
	*httptest.Server
	mu             sync.Mutex
	authCalls      int
	apiCalls       map[string]int
	lastBodies     map[string][]byte
	authFail       bool // auth 端点返回 401
	emptyCreateIDs bool // POST /v1/servers 返回空 server_id
}

func (f *fakeVDIExtServer) token() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return "tok-" + itoa(f.authCalls)
}

func (f *fakeVDIExtServer) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.apiCalls[path]
}

func (f *fakeVDIExtServer) authCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.authCalls
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// newFakeVDIExtServer 启动 VDI extended-client 假服务器。
// invalidOnce=true 时首个 /v1/servers 请求返回 AUTH_TOKEN_INVALID(1101) 触发重试路径。
func newFakeVDIExtServer(t *testing.T, invalidOnce bool) *fakeVDIExtServer {
	t.Helper()
	f := &fakeVDIExtServer{apiCalls: map[string]int{}, lastBodies: map[string][]byte{}}
	serversInvalid := invalidOnce

	writeJSON := func(w http.ResponseWriter, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		//nolint:errcheck
		_ = writeJSONBody(w, v)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.authCalls++
		fail := f.authFail
		n := f.authCalls
		f.mu.Unlock()
		if fail {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(w, map[string]interface{}{"error_code": 1001, "error_message": "Invalid credentials"})
			return
		}
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"token": map[string]interface{}{"tenant_id": 1, "auth_token": "tok-" + itoa(n)},
		})
	})
	mux.HandleFunc("/v1/resources_group", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/resources_group"]++
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data": []map[string]string{
				{"id": "g1", "name": "grp-1", "note": "独享桌面", "enable": "1"},
				{"id": "g2", "name": "grp-2", "note": "池桌面", "enable": "1"},
				{"id": "g3", "name": "grp-disabled", "note": "", "enable": "0"},
			},
		})
	})
	mux.HandleFunc("/v1/resources/list/", func(w http.ResponseWriter, r *http.Request) {
		groupID := r.URL.Path[len("/v1/resources/list/"):]
		f.mu.Lock()
		f.apiCalls["/v1/resources/list"]++
		f.mu.Unlock()
		resources := []map[string]interface{}{}
		if groupID == "g1" {
			resources = append(resources,
				map[string]interface{}{"id": 1, "name": "res-1", "note": "", "grp_id": 1},
				map[string]interface{}{"id": 2, "name": "res-2", "note": "", "grp_id": 1},
			)
		} else if groupID == "g2" {
			resources = append(resources, map[string]interface{}{"id": 3, "name": "res-3", "note": "", "grp_id": 2})
		}
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data":       map[string]interface{}{"resources": resources},
		})
	})
	mux.HandleFunc("/v1/resource/servers", func(w http.ResponseWriter, r *http.Request) {
		rcid := r.URL.Query().Get("rcid")
		f.mu.Lock()
		f.apiCalls["/v1/resource/servers"]++
		f.lastBodies["/v1/resource/servers"] = nil
		f.mu.Unlock()
		vms := []map[string]interface{}{}
		if rcid == "1" {
			vms = append(vms,
				map[string]interface{}{
					"_id": "101", "status": "15", "vm_name": "vm-alpha", "vtp_id": "7", "vtp_name": "VMP-A",
					"rc_id": "1", "rc_name": "res-1", "ip": "10.0.0.1", "mac": "AA:BB:CC:00:00:01",
					"cpu_number": "2", "cpu_core": "4", "cpu_per": "12", "mem_all": "8192", "mem_per": "33",
					"disc_all": "100", "disc_per": "44", "assign_ip": "192.168.1.101",
					"subnetmask": "255.255.255.0", "defaultgateway": "192.168.1.1", "nameserver": "8.8.8.8",
					"ip_state": "1", "osType": "win10",
				},
				map[string]interface{}{
					"_id": "102", "status": "11", "vm_name": "vm-beta", "vtp_id": "7", "vtp_name": "VMP-A",
					"rc_id": "1", "rc_name": "res-1", "ip": "10.0.0.2", "mac": "AA:BB:CC:00:00:02",
					"cpu_number": "4", "cpu_core": "8", "cpu_per": "5", "mem_all": "4096", "mem_per": "66",
					"disc_all": "200", "disc_per": "10", "assign_ip": "-",
					"subnetmask": "", "defaultgateway": "", "nameserver": "",
					"ip_state": "0", "osType": "win11",
				},
			)
		} else if rcid == "3" {
			vms = append(vms, map[string]interface{}{
				"_id": "301", "status": "12", "vm_name": "vm-gamma", "vtp_id": "8", "vtp_name": "VMP-B",
				"rc_id": "3", "rc_name": "res-3", "ip": "10.0.0.3", "mac": "AA:BB:CC:00:00:03",
				"cpu_number": "2", "cpu_core": "2", "cpu_per": "1", "mem_all": "2048", "mem_per": "20",
				"disc_all": "50", "disc_per": "5", "assign_ip": "", "ip_state": "0", "osType": "l2664",
			})
		}
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data": map[string]interface{}{"totalCount": itoa(len(vms)), "data": vms},
		})
	})
	mux.HandleFunc("/v1/servers/detail/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/servers/detail"]++
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data": map[string]interface{}{"vm_id": "101", "name": "vm-alpha", "power_state": "in_use"},
		})
	})
	mux.HandleFunc("/v1/servers", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/servers:"+r.Method]++
		f.mu.Unlock()
		// AUTH_TOKEN_INVALID 一次 → 客户端清缓存重认证后重试
		f.mu.Lock()
		invalid := serversInvalid
		serversInvalid = false
		f.mu.Unlock()
		if r.Header.Get("Auth-Token") == "tok-1" && invalid {
			writeJSON(w, map[string]interface{}{"error_code": 1101, "error_message": "AUTH_TOKEN_INVALID"})
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, map[string]interface{}{
				"error_code": 0, "error_message": "",
				"data": []map[string]string{{"vm_id": "101", "name": "vm-alpha", "power_state": "in_use"}},
			})
		case http.MethodPost:
			f.mu.Lock()
			empty := f.emptyCreateIDs
			f.mu.Unlock()
			ids := []string{"9001"}
			if empty {
				ids = []string{}
			}
			writeJSON(w, map[string]interface{}{
				"error_code": 0, "error_message": "",
				"data":       map[string]interface{}{"task_id": 9, "server_id": ids},
			})
		case http.MethodDelete:
			writeJSON(w, map[string]interface{}{"error_code": 0, "error_message": ""})
		}
	})
	mux.HandleFunc("/v1/servers/action", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/servers/action"]++
		f.lastBodies["/v1/servers/action"], _ = readAll(r)
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{"error_code": 0, "error_message": ""})
	})
	mux.HandleFunc("/v1/servers/bind_users", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/servers/bind_users"]++
		f.lastBodies["/v1/servers/bind_users"], _ = readAll(r)
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data":       map[string]interface{}{"vm_id": "101", "name": "vm-alpha"},
		})
	})
	mux.HandleFunc("/v1/run_position", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/run_position"]++
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data": map[string]interface{}{"run": []map[string]string{
				{"id": "h1", "name": "host-1", "father_id": "h1"},
			}},
		})
	})
	mux.HandleFunc("/v1/storages", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/storages"]++
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data": map[string]interface{}{"storages": []map[string]interface{}{
				{"id": "s1", "name": "storage-1", "type": "local", "total": "100", "avail": "50", "shared": 0, "status": 0},
			}},
		})
	})
	mux.HandleFunc("/v1/networks", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/v1/networks"]++
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data": map[string]interface{}{"networks": []map[string]string{
				{"id": "n1", "name": "net-1", "mode": "bridge"},
			}},
		})
	})
	mux.HandleFunc("/api/v1/user/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.apiCalls["/api/v1/user"]++
		f.mu.Unlock()
		writeJSON(w, map[string]interface{}{
			"error_code": 0, "error_message": "",
			"data": []map[string]string{{"vm_id": "101", "name": "vm-alpha", "power_state": "in_use"}},
		})
	})

	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

// =====================================================================
// VDIServerService CRUD
// =====================================================================

func TestVDIServerService_CRUD(t *testing.T) {
	db := newVDITestDB(t)
	svc := NewVDIServerService(db)
	ctx := context.Background()

	// Create: 密码 AES 加密落库（不是明文）
	created, err := svc.CreateServer(ctx, &CreateVDIServerRequest{
		Name: "s1", Endpoint: "http://vdi.local", Username: "admin", Password: "plain-pw", TenantID: 5,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	var stored string
	require.NoError(t, db.Raw(`SELECT password_encrypted FROM sys_vdi_server WHERE id = ?`, created.ID).Scan(&stored).Error)
	assert.NotEqual(t, "plain-pw", stored)
	assert.NotEmpty(t, stored)

	// Get: 命中 + 未命中
	got, err := svc.GetServer(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "s1", got.Name)
	assert.Equal(t, 5, got.TenantID)
	_, err = svc.GetServer(ctx, "missing")
	require.Error(t, err)

	// List: 默认分页 + 排序白名单
	seedVDIServer(t, db, "srv2", "aaa", "http://x", 0)
	seedVDIServer(t, db, "srv3", "zzz", "http://y", 1)

	page, err := svc.ListServers(ctx, 0, 0, "", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 10, page.PageSize)

	asc := true
	page, err = svc.ListServers(ctx, 1, 2, "name", &asc)
	require.NoError(t, err)
	require.Len(t, page.List, 2)
	assert.Equal(t, "aaa", page.List[0].Name)

	desc := false
	page, err = svc.ListServers(ctx, 1, 2, "name", &desc)
	require.NoError(t, err)
	assert.Equal(t, "zzz", page.List[0].Name)

	// pageSize > 100 → 回落 10
	page, err = svc.ListServers(ctx, 1, 500, "", nil)
	require.NoError(t, err)
	assert.Equal(t, 10, page.PageSize)

	// Update: 全字段 + 密码更新清 token
	future := time.Now().Add(time.Hour)
	require.NoError(t, db.Exec(`UPDATE sys_vdi_server SET auth_token = 'old', token_expiry = ? WHERE id = ?`,
		future.Format("2006-01-02 15:04:05"), created.ID).Error)
	newTenant := 9
	newStatus := 1
	require.NoError(t, svc.UpdateServer(ctx, created.ID, &UpdateVDIServerRequest{
		Name: strPtr("s1-new"), Endpoint: strPtr("http://vdi2.local"),
		Username: strPtr("root"), Password: strPtr("new-pw"),
		TenantID: &newTenant, Status: &newStatus,
	}))
	var name, token string
	require.NoError(t, db.Raw(`SELECT name, auth_token FROM sys_vdi_server WHERE id = ?`, created.ID).Row().Scan(&name, &token))
	assert.Equal(t, "s1-new", name)
	assert.Equal(t, "", token, "改密码应清空缓存 token")

	// Update: 不存在的服务器
	require.Error(t, svc.UpdateServer(ctx, "missing", &UpdateVDIServerRequest{Name: strPtr("x")}))

	// Delete: 有 VM → 拒绝
	require.NoError(t, db.Exec(`INSERT INTO sys_vdi_vm (id, vm_id, name, vdi_server_id, created_at) VALUES ('vm1','v1','n', ?, ?)`,
		created.ID, time.Now().Format("2006-01-02 15:04:05")).Error)
	err = svc.DeleteServer(ctx, created.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "associated VMs")

	// Delete: 无 VM → 软删
	require.NoError(t, db.Exec(`DELETE FROM sys_vdi_vm`).Error)
	require.NoError(t, svc.DeleteServer(ctx, created.ID))
	_, err = svc.GetServer(ctx, created.ID)
	require.Error(t, err)
}

func strPtr(s string) *string { return &s }

func TestVDIServerService_TestConnection(t *testing.T) {
	fake := newFakeVDIExtServer(t, false)
	db := newVDITestDB(t)
	svc := NewVDIServerService(db)
	ctx := context.Background()

	seedVDIServer(t, db, "srv-ok", "good", fake.URL, 0)
	require.NoError(t, svc.TestConnection(ctx, "srv-ok"))

	// 认证失败（401）
	fake.mu.Lock()
	fake.authFail = true
	fake.mu.Unlock()
	seedVDIServer(t, db, "srv-bad", "bad", fake.URL, 0)
	err := svc.TestConnection(ctx, "srv-bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed for server")

	// 服务器不存在 → 空 client → "VDI server not found"
	err = svc.TestConnection(ctx, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VDI server not found")
}

// =====================================================================
// ClientManager
// =====================================================================

func TestClientManager(t *testing.T) {
	db := newVDITestDB(t)
	fake := newFakeVDIExtServer(t, false)
	ctx := context.Background()

	// 无 db 句柄 → 错误
	noDB := &ClientManager{clients: map[string]*VDIClient{}}
	_, err := noDB.GetClient(ctx, "any")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVDIServerNotConfigured)

	m := NewClientManager(db)

	// 空 serverID
	_, err = m.GetClient(ctx, "")
	require.Error(t, err)

	// 记录不存在
	_, err = m.GetClient(ctx, "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVDIServerNotConfigured)

	// endpoint 为空
	seedVDIServer(t, db, "srv-empty", "e", "", 0)
	_, err = m.GetClient(ctx, "srv-empty")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVDIServerNotConfigured)

	// 正常创建 + 缓存复用
	seedVDIServer(t, db, "srv-ok", "ok", fake.URL, 0)
	c1, err := m.GetClient(ctx, "srv-ok")
	require.NoError(t, err)
	require.NotNil(t, c1)
	assert.Equal(t, fake.URL, c1.ServerURL)
	c2, err := m.GetClient(ctx, "srv-ok")
	require.NoError(t, err)
	assert.Same(t, c1, c2, "同 serverID 应复用缓存实例")

	// RemoveClient 后重建
	m.RemoveClient("srv-ok")
	c3, err := m.GetClient(ctx, "srv-ok")
	require.NoError(t, err)
	assert.NotSame(t, c1, c3)

	m.ClearAll()
}

func TestClientManager_Singleton(t *testing.T) {
	// 进程级单例：InitClientManager 首次注入 db，GetClientManager 返回同一实例
	db := newVDITestDB(t)
	inst := InitClientManager(db)
	require.NotNil(t, inst)
	assert.Same(t, inst, GetClientManager())
}

func TestVDIUtils_EncryptDecryptRoundtrip(t *testing.T) {
	enc := encryptVDIPassword("mypassword")
	assert.NotEqual(t, "mypassword", enc)
	assert.Equal(t, "mypassword", decryptVDIPassword(enc))

	// 非 base64 输入 → 空串（vdi 包版本返回 ""）
	assert.Equal(t, "", decryptVDIPassword("not-base64!!!"))
	// 过短密文 → 空串
	assert.Equal(t, "", decryptVDIPassword("YWJj"))
	// 随机字节（base64 合法但 GCM open 失败）→ 空串
	assert.Equal(t, "", decryptVDIPassword("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="))
}
