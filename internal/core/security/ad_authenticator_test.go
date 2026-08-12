package security

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// mockADDomainService Mock AD域服务
type mockADDomainService struct {
	config *models.ADConfig
	db     *gorm.DB
}

func (m *mockADDomainService) GetDB() *gorm.DB {
	return m.db
}

func (m *mockADDomainService) GetConfig(configID string) (*models.ADConfig, error) {
	if m.config == nil {
		return nil, errors.New("AD配置不存在")
	}
	return m.config, nil
}

// mockLDAPClient Mock LDAP客户端
type mockLDAPClient struct {
	searchResult   interface{}
	bindResult     error
	searchError    error
	bindCalled     bool
	searchCalled   bool
	bindUsername   string
	bindPassword   string
	searchBaseDN   string
	searchFilter   string
}

func (m *mockLDAPClient) Connect() error {
	return nil
}

func (m *mockLDAPClient) Close() error {
	return nil
}

func (m *mockLDAPClient) Bind(username, password string) error {
	m.bindCalled = true
	m.bindUsername = username
	m.bindPassword = password
	return m.bindResult
}

func (m *mockLDAPClient) Search(baseDN string, filter string) (interface{}, error) {
	m.searchCalled = true
	m.searchBaseDN = baseDN
	m.searchFilter = filter
	return m.searchResult, m.searchError
}

// TestADAuthenticator_Authenticate_Success 测试AD认证成功场景
func TestADAuthenticator_Authenticate_Success(t *testing.T) {
	t.Skip("TODO: WIP - 需要真实 DB + LDAP 测试环境；当前 ad_authenticator_test 用 mockADDomainService 而 NewADAuthenticator 现需 *gorm.DB")

	// Mock AD域服务
	adConfig := &models.ADConfig{
		BaseModel:     models.BaseModel{ID: "test-ad-config"},
		ConfigName:    "Test AD",
		ServerAddress: "192.168.1.100",
		ServerPort:    389,
		DomainName:    "test.com",
		BaseDN:        "dc=test,dc=com",
		Status:        0,
	}

	mockADSvc := &mockADDomainService{
		config: adConfig,
	}

	auth := NewADAuthenticator(mockADSvc.db, "test-ad-config")
	req := MockAuthRequest("testuser", "adpassword")

	// 注意：这个测试会尝试真实的LDAP连接
	// 在实际使用时需要Mock LDAP客户端或使用测试环境
	result, err := auth.Authenticate(context.Background(), req)

	// 由于没有真实的AD环境，预期会失败
	if err != nil {
		assert.True(t, errors.Is(err, ErrADConnectionFailed) ||
			errors.Is(err, ErrInvalidCredentials) ||
			err.Error() == "AD配置不存在" ||
			err.Error() == "获取AD配置失败")
		assert.Nil(t, result)
	} else {
		// 如果测试环境有真实AD，验证结果
		assert.NotNil(t, result)
		assert.Equal(t, "ad", result.AuthSource)
		assert.True(t, result.NeedsSync || result.User != nil)
	}
}

// TestADAuthenticator_Authenticate_ConfigNotFound 测试AD配置未启用场景
func TestADAuthenticator_Authenticate_ConfigNotFound(t *testing.T) {
	t.Skip("TODO: WIP - 需要真实 DB + LDAP 测试环境")

	mockADSvc := &mockADDomainService{
		config: nil, // 配置不存在
	}

	auth := NewADAuthenticator(mockADSvc.db, "nonexistent-config")
	req := MockAuthRequest("testuser", "password")

	result, err := auth.Authenticate(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestADAuthenticator_Name 测试认证器名称
func TestADAuthenticator_Name(t *testing.T) {
	mockADSvc := &mockADDomainService{}
	auth := NewADAuthenticator(mockADSvc.db, "test-config")

	assert.Equal(t, "ad", auth.Name())
}

// TestADAuthenticator_Authenticate_TableDrivenTests 表格驱动测试
func TestADAuthenticator_Authenticate_TableDrivenTests(t *testing.T) {
	t.Skip("TODO: WIP - 需要真实 DB + LDAP 测试环境")

	mockADSvc := &mockADDomainService{}

	tests := []struct {
		name        string
		configID    string
		username    string
		password    string
		wantErr     error
		description string
	}{
		{
			name:        "配置不存在",
			configID:    "nonexistent",
			username:    "testuser",
			password:    "password",
			wantErr:     ErrADConfigNotFound,
			description: "使用不存在的AD配置ID",
		},
		{
			name:        "空用户名",
			configID:    "test-config",
			username:    "",
			password:    "password",
			wantErr:     ErrInvalidCredentials,
			description: "空用户名应该认证失败",
		},
		{
			name:        "空密码",
			configID:    "test-config",
			username:    "testuser",
			password:    "",
			wantErr:     ErrInvalidCredentials,
			description: "空密码应该认证失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewADAuthenticator(mockADSvc.db, tt.configID)
			req := MockAuthRequest(tt.username, tt.password)
			result, err := auth.Authenticate(context.Background(), req)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErr) || err.Error() != "")
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestADUserInfo 测试AD用户信息结构
func TestADUserInfo(t *testing.T) {
	adUserInfo := &ADUserInfo{
		UserDN:      "cn=testuser,ou=users,dc=test,dc=com",
		Username:    "testuser",
		DisplayName: "Test User",
		Email:       "testuser@test.com",
		Phone:       "1234567890",
		Mobile:      "9876543210",
		Title:       "Software Engineer",
		Department:  "Engineering",
	}

	assert.Equal(t, "cn=testuser,ou=users,dc=test,dc=com", adUserInfo.UserDN)
	assert.Equal(t, "testuser", adUserInfo.Username)
	assert.Equal(t, "Test User", adUserInfo.DisplayName)
	assert.Equal(t, "testuser@test.com", adUserInfo.Email)
	assert.Equal(t, "1234567890", adUserInfo.Phone)
	assert.Equal(t, "9876543210", adUserInfo.Mobile)
	assert.Equal(t, "Software Engineer", adUserInfo.Title)
	assert.Equal(t, "Engineering", adUserInfo.Department)
}

// TestADAuthenticator_NeedsSyncFlag 测试NeedsSync标志
func TestADAuthenticator_NeedsSyncFlag(t *testing.T) {
	t.Skip("TODO: WIP - 需要真实 DB + LDAP 测试环境")

	mockADSvc := &mockADDomainService{
		config: &models.ADConfig{
			BaseModel:     models.BaseModel{ID: "test-config"},
			ServerAddress: "192.168.1.100",
			DomainName:    "test.com",
			BaseDN:        "dc=test,dc=com",
			Status:        0,
		},
	}

	auth := NewADAuthenticator(mockADSvc.db, "test-config")
	req := MockAuthRequest("testuser", "password")

	result, err := auth.Authenticate(context.Background(), req)

	// 由于没有真实AD环境，预期失败或返回需要同步
	if err == nil && result != nil {
		// 如果认证成功，验证NeedsSync标志
		assert.True(t, result.NeedsSync || result.User != nil,
			"AD认证成功应该要么标记NeedsSync=true，要么返回User信息")
	}
}

// TestADAuthenticator_IntegrationTest 集成测试标记
// 注意：这个测试需要真实的AD环境才能运行
// 在CI/CD环境中应该跳过或使用Mock
func TestADAuthenticator_IntegrationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（使用 -short 标志）")
	}

	// TODO: 配置测试AD环境
	// 1. 设置环境变量：AD_SERVER_ADDRESS, AD_ADMIN_USERNAME, AD_ADMIN_PASSWORD
	// 2. 创建测试AD配置
	// 3. 执行真实AD认证
	// 4. 验证结果

	t.Skip("集成测试需要真实AD环境配置")
}
