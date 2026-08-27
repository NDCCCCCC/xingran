package server

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// 命令注入缝: 让 77-05 的真策略体测试可经 re-exec 子进程桩驱动, 而非真跑
// powershell/useradd/chpasswd/tee/chmod (P-77-9: Windows 上 powershell 真实调用
// 危险)。var 初值即原直调, 生产路径 byte 行为不变; 76-02b 覆盖纪律 — 测试
// 改前先 t.Cleanup 恢复, 禁 t.Parallel。
var (
	runAccountCmd       = runCommand
	runAccountCmdOutput = runCommandOutput
	newAccountCmd       = newCommand
)

// platformStrategy 平台策略接口
type platformStrategy interface {
	createAccount(ctx context.Context, username, password string, isAdmin bool) error
	deleteAccount(ctx context.Context, username string) error
	resetPassword(ctx context.Context, username, newPassword string) error
	enableAccount(ctx context.Context, username string) error
	disableAccount(ctx context.Context, username string) error
	listAccounts(ctx context.Context) ([]string, error)
}

// windowsPlatformStrategy Windows 平台实现
type windowsPlatformStrategy struct{}

// linuxPlatformStrategy Linux 平台实现
type linuxPlatformStrategy struct{}

// AccountManager 账号管理器
type AccountManager struct {
	strategy platformStrategy
}

// NewAccountManager 创建账号管理器
func NewAccountManager() *AccountManager {
	var strategy platformStrategy

	switch runtime.GOOS {
	case "windows":
		strategy = &windowsPlatformStrategy{}
	case "linux":
		strategy = &linuxPlatformStrategy{}
	default:
		strategy = &linuxPlatformStrategy{} // 默认使用 Linux 策略
	}

	return &AccountManager{
		strategy: strategy,
	}
}

// Account 账号信息
type Account struct {
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	IsAdmin   bool   `json:"is_admin"`
	IsEnabled bool   `json:"is_enabled"`
}

// CreateAccount 创建本地账号
func (m *AccountManager) CreateAccount(ctx context.Context, account *Account) error {
	return m.strategy.createAccount(ctx, account.Username, account.Password, account.IsAdmin)
}

// DeleteAccount 删除账号
func (m *AccountManager) DeleteAccount(ctx context.Context, username string) error {
	return m.strategy.deleteAccount(ctx, username)
}

// ResetPassword 重置密码
func (m *AccountManager) ResetPassword(ctx context.Context, username, newPassword string) error {
	return m.strategy.resetPassword(ctx, username, newPassword)
}

// EnableAccount 启用账号
func (m *AccountManager) EnableAccount(ctx context.Context, username string) error {
	return m.strategy.enableAccount(ctx, username)
}

// DisableAccount 禁用账号
func (m *AccountManager) DisableAccount(ctx context.Context, username string) error {
	return m.strategy.disableAccount(ctx, username)
}

// ListAccounts 列出所有账号
func (m *AccountManager) ListAccounts(ctx context.Context) ([]string, error) {
	return m.strategy.listAccounts(ctx)
}

// ============ Windows 平台实现 ============

func (w *windowsPlatformStrategy) createAccount(ctx context.Context, username, password string, isAdmin bool) error {
	// 使用 PowerShell 创建账号，通过内联脚本传递密码避免命令行泄露
	psScript := fmt.Sprintf(
		"$password = ConvertTo-SecureString '%s' -AsPlainText -Force; "+
			"New-LocalUser -Name %s -Password $password -Description 'XingRan VDI Account'",
		password, username,
	)

	if err := runAccountCmd(ctx, "powershell", "-Command", psScript); err != nil {
		return fmt.Errorf("failed to create Windows user: %w", err)
	}

	if isAdmin {
		addScript := fmt.Sprintf("Add-LocalGroupMember -Group 'Administrators' -Member %s", username)
		if err := runAccountCmd(ctx, "powershell", "-Command", addScript); err != nil {
			return fmt.Errorf("failed to add to admin group: %w", err)
		}
	}

	return nil
}

func (w *windowsPlatformStrategy) deleteAccount(ctx context.Context, username string) error {
	return runAccountCmd(ctx, "powershell", "-Command",
		fmt.Sprintf("Remove-LocalUser -Name %s", username))
}

func (w *windowsPlatformStrategy) resetPassword(ctx context.Context, username, newPassword string) error {
	psScript := fmt.Sprintf(
		"$password = ConvertTo-SecureString '%s' -AsPlainText -Force; "+
			"Set-LocalUser -Name %s -Password $password",
		newPassword, username,
	)
	return runAccountCmd(ctx, "powershell", "-Command", psScript)
}

func (w *windowsPlatformStrategy) enableAccount(ctx context.Context, username string) error {
	return runAccountCmd(ctx, "powershell", "-Command",
		fmt.Sprintf("Enable-LocalUser -Name %s", username))
}

func (w *windowsPlatformStrategy) disableAccount(ctx context.Context, username string) error {
	return runAccountCmd(ctx, "powershell", "-Command",
		fmt.Sprintf("Disable-LocalUser -Name %s", username))
}

func (w *windowsPlatformStrategy) listAccounts(ctx context.Context) ([]string, error) {
	output, err := runAccountCmdOutput(ctx, "powershell", "-Command", "Get-LocalUser | Select-Object -ExpandProperty Name")
	if err != nil {
		return nil, fmt.Errorf("failed to list Windows users: %w", err)
	}
	return parseWindowsUsers(string(output)), nil
}

// ============ Linux 平台实现 ============

func (l *linuxPlatformStrategy) createAccount(ctx context.Context, username, password string, isAdmin bool) error {
	if err := runAccountCmd(ctx, "useradd", "-m", username); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if err := l.setLinuxPassword(ctx, username, password); err != nil {
		return err
	}

	if isAdmin {
		return l.configureSudo(ctx, username)
	}

	return nil
}

func (l *linuxPlatformStrategy) deleteAccount(ctx context.Context, username string) error {
	return runAccountCmd(ctx, "userdel", "-r", username)
}

func (l *linuxPlatformStrategy) resetPassword(ctx context.Context, username, newPassword string) error {
	return l.setLinuxPassword(ctx, username, newPassword)
}

func (l *linuxPlatformStrategy) enableAccount(ctx context.Context, username string) error {
	return runAccountCmd(ctx, "usermod", "-U", username)
}

func (l *linuxPlatformStrategy) disableAccount(ctx context.Context, username string) error {
	return runAccountCmd(ctx, "usermod", "-L", username)
}

func (l *linuxPlatformStrategy) listAccounts(ctx context.Context) ([]string, error) {
	output, err := runAccountCmdOutput(ctx, "getent", "passwd")
	if err != nil {
		return nil, fmt.Errorf("failed to list Linux users: %w", err)
	}
	return parseLinuxUsers(string(output)), nil
}

// setLinuxPassword 设置 Linux 用户密码
func (l *linuxPlatformStrategy) setLinuxPassword(ctx context.Context, username, password string) error {
	cmd := newAccountCmd(ctx, "chpasswd")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	// 确保在 goroutine 出错时也关闭 stdin
	defer stdin.Close()

	// 使用单独的 goroutine 写入，避免阻塞
	go func() {
		fmt.Fprintf(stdin, "%s:%s\n", username, password)
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	return nil
}

// configureSudo 配置 sudo 权限
func (l *linuxPlatformStrategy) configureSudo(ctx context.Context, username string) error {
	sudoersContent := fmt.Sprintf(
		"%s ALL=(root) NOPASSWD: /usr/sbin/useradd, /usr/sbin/userdel, /usr/sbin/usermod, /usr/bin/passwd\n",
		username,
	)
	sudoersPath := fmt.Sprintf("/etc/sudoers.d/%s", username)

	teeCmd := newAccountCmd(ctx, "tee", sudoersPath)
	teeCmd.Stdin = strings.NewReader(sudoersContent)
	if err := teeCmd.Run(); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}

	if err := runAccountCmd(ctx, "chmod", "440", sudoersPath); err != nil {
		return fmt.Errorf("failed to set sudoers permissions: %w", err)
	}

	return nil
}

// ============ 解析辅助函数 ============

// parseWindowsUsers 解析 Windows 用户列表
func parseWindowsUsers(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	users := make([]string, 0, len(lines))

	for _, line := range lines {
		if user := strings.TrimSpace(line); user != "" {
			users = append(users, user)
		}
	}

	return users
}

// parseLinuxUsers 解析 Linux 用户列表
func parseLinuxUsers(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	users := make([]string, 0)

	for _, line := range lines {
		// 使用 Index 查找冒号位置，避免分割整个字符串
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}

		username := line[:idx]

		// 跳过系统用户（UID < 1000）
		// 格式: username:x:uid:gid:...
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			uidStr := parts[2]
			if uid, err := strconv.Atoi(uidStr); err == nil {
				if uid >= 1000 {
					users = append(users, username)
				}
			}
		}
	}

	return users
}
