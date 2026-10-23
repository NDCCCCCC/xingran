package addomain

import (
	"errors"
	"testing"

	"github.com/go-ldap/ldap/v3"
)

// mockLDAPClient 是 LDAPClientIface 的手写 mock 实现（项目不使用 gomock）。
// 通过预设各方法的返回值（err / 结果），在测试中隔离真实 AD 服务器依赖。
type mockLDAPClient struct {
	// 连接
	connectErr error
	closeErr   error

	// 搜索（按方法分别返回）
	searchOUsErr      error
	searchOUsResult   []*ldap.Entry
	searchGroupsErr   error
	searchGroupsRes   []*ldap.Entry
	searchUsersErr    error
	searchUsersRes    []*ldap.Entry
	// searchUsersFn 非 nil 时优先于 searchUsersRes（多批次/函数式返回驱动，
	// 76-03 walk/分页窄边界：驱动 service 层遍历，非 wire 级分页）
	searchUsersFn     func() ([]*ldap.Entry, error)
	searchComputersErr error
	searchComputersRes []*ldap.Entry

	// 写操作
	addMemberErr        error
	removeMemberErr     error
	addMembersErr       error
	removeMembersErr    error
	createGroupErr      error
	deleteGroupErr      error
	// 域控审查 P0-1/P0-2 新增方法
	getGroupMembersRes map[string][]string // groupDN → 成员 DN 列表
	getGroupMembersErr error
	deleteOUErr        error
	deleteOUCalls      int
	deleteOULastDN     string
	renameOUErr        error
	renameOUCalls      int
	renameOULastDN     string
	renameOULastName   string
	updateUserAttrErr   error
	moveUserErr         error
	enableUserErr       error
	disableUserErr      error
	updateGroupAttrErr  error
	createOUErr         error
	dnExistsRes         bool
	dnExistsErr         error
	searchWithReqRes    *ldap.SearchResult
	searchWithReqErr    error

	// 调用计数
	connectCalls       int
	closeCalls         int
	searchGrpCalls     int
	searchUsersCalls   int
	updateGroupCalls   int
	createOUCalls      int
	dnExistsCalls      int
	searchWithReqCalls int

	// Phase 58（共 4 个，compare-and-move 幂等断言的前提）：
	//   - moveUserCalls / moveUserTargets：断言"目标 OU 相同则不调 MoveUser"以及
	//     "移到了哪个 OU"（moveUserTargets 按调用顺序记录每次的 newOUDN）
	//   - disableUserCalls / updateUserAttrCalls：断言终态路径是否触达对应 LDAP 写操作
	moveUserCalls       int
	moveUserTargets     []string
	disableUserCalls    int
	updateUserAttrCalls int
	// updateUserAttrLastAttrs 记录**最后一次** UpdateUserAttribute 的 attrs 实参
	// （Phase 58 Plan 04 追加）：排水器"闭包内重读最新值"的可观测证据 ——
	// 只有落到 AD 属性上的值才是真的推了最新值，仅计数无法断言。
	// 追加字段不改任何既有方法签名（既有用例零影响）。
	updateUserAttrLastAttrs map[string]string
	// deleteUserAttr* 记录 DeleteUserAttribute 调用次数与最近一次实参（经理同步测试用）
	deleteUserAttrCalls    int
	deleteUserAttrLastDN   string
	deleteUserAttrLastAttr string
}

func (m *mockLDAPClient) Connect() error {
	m.connectCalls++
	return m.connectErr
}

func (m *mockLDAPClient) Close() {
	m.closeCalls++
}

func (m *mockLDAPClient) SearchOUs(baseDN string) ([]*ldap.Entry, error) {
	return m.searchOUsResult, m.searchOUsErr
}

func (m *mockLDAPClient) SearchGroups(baseDN string) ([]*ldap.Entry, error) {
	m.searchGrpCalls++
	return m.searchGroupsRes, m.searchGroupsErr
}

func (m *mockLDAPClient) SearchUsers(baseDN string) ([]*ldap.Entry, error) {
	m.searchUsersCalls++
	if m.searchUsersFn != nil {
		return m.searchUsersFn()
	}
	return m.searchUsersRes, m.searchUsersErr
}

func (m *mockLDAPClient) SearchComputers(baseDN string) ([]*ldap.Entry, error) {
	return m.searchComputersRes, m.searchComputersErr
}

func (m *mockLDAPClient) AddGroupMember(groupDN, userDN string) error {
	return m.addMemberErr
}

func (m *mockLDAPClient) RemoveGroupMember(groupDN, userDN string) error {
	return m.removeMemberErr
}

func (m *mockLDAPClient) AddGroupMembers(groupDN string, userDNs []string) error {
	return m.addMembersErr
}

func (m *mockLDAPClient) RemoveGroupMembers(groupDN string, userDNs []string) error {
	return m.removeMembersErr
}

// GetGroupMembers 实现 P0-1 新增接口；getGroupMembersRes[groupDN] 非空时返回对应列表。
func (m *mockLDAPClient) GetGroupMembers(groupDN string) ([]string, error) {
	if m.getGroupMembersErr != nil {
		return nil, m.getGroupMembersErr
	}
	if m.getGroupMembersRes == nil {
		return nil, nil
	}
	return m.getGroupMembersRes[groupDN], nil
}

// DeleteOU 实现 P0-2 新增接口（部门删除联动）。
func (m *mockLDAPClient) DeleteOU(ouDN string) error {
	m.deleteOUCalls++
	m.deleteOULastDN = ouDN
	return m.deleteOUErr
}

// RenameOU 实现 P0-2 新增接口（部门改名联动）。
func (m *mockLDAPClient) RenameOU(ouDN, newOUName string) error {
	m.renameOUCalls++
	m.renameOULastDN = ouDN
	m.renameOULastName = newOUName
	return m.renameOUErr
}

func (m *mockLDAPClient) CreateGroup(groupDN, groupName, description string, groupType int) error {
	return m.createGroupErr
}

func (m *mockLDAPClient) DeleteGroup(groupDN string) error {
	return m.deleteGroupErr
}

// DeleteUserAttribute 实现 LDAPClientIface 新增的删属性接口（经理同步清空 manager 用）。
func (m *mockLDAPClient) DeleteUserAttribute(userDN, attr string) error {
	m.deleteUserAttrCalls++
	m.deleteUserAttrLastDN = userDN
	m.deleteUserAttrLastAttr = attr
	return nil
}

func (m *mockLDAPClient) UpdateUserAttribute(userDN string, attrs map[string]string) error {
	m.updateUserAttrCalls++
	// 拷贝一份快照：调用方的 attributes map 在同一轮内不会复用，但拷贝能杜绝
	// "测试断言时被后续调用覆写"的隐式耦合（Phase 58 Plan 04）。
	snapshot := make(map[string]string, len(attrs))
	for k, v := range attrs {
		snapshot[k] = v
	}
	m.updateUserAttrLastAttrs = snapshot
	return m.updateUserAttrErr
}

func (m *mockLDAPClient) MoveUser(userDN, newOUDN string) error {
	m.moveUserCalls++
	m.moveUserTargets = append(m.moveUserTargets, newOUDN)
	return m.moveUserErr
}

func (m *mockLDAPClient) EnableUser(userDN string) error {
	return m.enableUserErr
}

func (m *mockLDAPClient) DisableUser(userDN string) error {
	m.disableUserCalls++
	return m.disableUserErr
}

func (m *mockLDAPClient) UpdateGroupAttribute(groupDN string, attrs map[string]string) error {
	m.updateGroupCalls++
	return m.updateGroupAttrErr
}

func (m *mockLDAPClient) CreateOU(ouDN, ouName string) error {
	m.createOUCalls++
	return m.createOUErr
}

func (m *mockLDAPClient) DNExists(dn string) (bool, error) {
	m.dnExistsCalls++
	return m.dnExistsRes, m.dnExistsErr
}

func (m *mockLDAPClient) SearchWithRequest(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error) {
	m.searchWithReqCalls++
	return m.searchWithReqRes, m.searchWithReqErr
}

// 编译期断言：mockLDAPClient 满足 LDAPClientIface
var _ LDAPClientIface = (*mockLDAPClient)(nil)

// ============ Connect 测试 ============

func TestMockLDAPClient_Connect_Success(t *testing.T) {
	mock := &mockLDAPClient{connectErr: nil}

	err := mock.Connect()

	if err != nil {
		t.Errorf("Connect() expected nil error, got %v", err)
	}
	if mock.connectCalls != 1 {
		t.Errorf("expected 1 connect call, got %d", mock.connectCalls)
	}
}

func TestMockLDAPClient_Connect_Failure(t *testing.T) {
	expectedErr := errors.New("connection refused")
	mock := &mockLDAPClient{connectErr: expectedErr}

	err := mock.Connect()

	if err != expectedErr {
		t.Errorf("Connect() expected %v, got %v", expectedErr, err)
	}
	if mock.connectCalls != 1 {
		t.Errorf("expected 1 connect call, got %d", mock.connectCalls)
	}
}

// ============ Close 测试 ============

func TestMockLDAPClient_Close(t *testing.T) {
	mock := &mockLDAPClient{}

	mock.Close()

	if mock.closeCalls != 1 {
		t.Errorf("expected 1 close call, got %d", mock.closeCalls)
	}
}

// ============ SearchGroups 测试 ============

func TestMockLDAPClient_SearchGroups_ReturnsEntries(t *testing.T) {
	entries := []*ldap.Entry{
		ldap.NewEntry("CN=Group1,DC=example,DC=com", map[string][]string{"cn": {"Group1"}}),
		ldap.NewEntry("CN=Group2,DC=example,DC=com", map[string][]string{"cn": {"Group2"}}),
	}
	mock := &mockLDAPClient{searchGroupsRes: entries}

	result, err := mock.SearchGroups("DC=example,DC=com")

	if err != nil {
		t.Errorf("SearchGroups() unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if mock.searchGrpCalls != 1 {
		t.Errorf("expected 1 search call, got %d", mock.searchGrpCalls)
	}
}

func TestMockLDAPClient_SearchGroups_Empty(t *testing.T) {
	mock := &mockLDAPClient{searchGroupsRes: nil, searchGroupsErr: nil}

	result, err := mock.SearchGroups("DC=example,DC=com")

	if err != nil {
		t.Errorf("SearchGroups() unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

func TestMockLDAPClient_SearchGroups_Error(t *testing.T) {
	expectedErr := errors.New("LDAP search failed")
	mock := &mockLDAPClient{searchGroupsErr: expectedErr}

	result, err := mock.SearchGroups("DC=example,DC=com")

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %v", result)
	}
}

// ============ SearchUsers 测试 ============

func TestMockLDAPClient_SearchUsers_ReturnsEntries(t *testing.T) {
	entries := []*ldap.Entry{
		ldap.NewEntry("CN=user1,DC=example,DC=com", map[string][]string{"sAMAccountName": {"user1"}}),
	}
	mock := &mockLDAPClient{searchUsersRes: entries}

	result, err := mock.SearchUsers("DC=example,DC=com")

	if err != nil {
		t.Errorf("SearchUsers() unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result))
	}
}

// ============ 写操作测试（覆盖成员管理与组管理错误路径） ============

func TestMockLDAPClient_AddGroupMember_Success(t *testing.T) {
	mock := &mockLDAPClient{}

	err := mock.AddGroupMember("CN=Group,DC=example,DC=com", "CN=user,DC=example,DC=com")

	if err != nil {
		t.Errorf("AddGroupMember() unexpected error: %v", err)
	}
}

func TestMockLDAPClient_CreateGroup_Error(t *testing.T) {
	expectedErr := errors.New("insufficient rights")
	mock := &mockLDAPClient{createGroupErr: expectedErr}

	err := mock.CreateGroup("CN=NewGroup,DC=example,DC=com", "NewGroup", "", 0)

	if err != expectedErr {
		t.Errorf("CreateGroup() expected %v, got %v", expectedErr, err)
	}
}

func TestMockLDAPClient_DeleteGroup_Error(t *testing.T) {
	expectedErr := errors.New("group not found")
	mock := &mockLDAPClient{deleteGroupErr: expectedErr}

	err := mock.DeleteGroup("CN=Missing,DC=example,DC=com")

	if err != expectedErr {
		t.Errorf("DeleteGroup() expected %v, got %v", expectedErr, err)
	}
}
