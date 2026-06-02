package server

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// AccountManager 账号管理器
type AccountManager struct {
	platform string
}

// NewAccountManager 创建账号管理器
func NewAccountManager() *AccountManager {
	return &AccountManager{
		platform: runtime.GOOS,
	}
}

// Account 账号信息
type Account struct {
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	IsAdmin   bool   `json:"is_admin"`
	IsEnabled bool   `json:"is_enabled"`
	OSType    string `json:"os_type"`
}

// CreateAccount 创建本地账号
func (m *AccountManager) CreateAccount(ctx context.Context, account *Account) error {
	switch m.platform {
	case "windows":
		return m.createWindowsAccount(ctx, account)
	case "linux":
		return m.createLinuxAccount(ctx, account)
	default:
		return fmt.Errorf("unsupported platform: %s", m.platform)
	}
}

// createWindowsAccount 创建 Windows 账号
func (m *AccountManager) createWindowsAccount(ctx context.Context, account *Account) error {
	// 使用 PowerShell 创建账号
	psCommand := fmt.Sprintf(
		"New-LocalUser -Name %s -Password (ConvertTo-SecureString %s -AsPlainText -Force) -Description 'RuoYi VDI Account'",
		account.Username,
		account.Password,
	)

	cmd := exec.CommandContext(ctx, "powershell", "-Command", psCommand)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Windows user: %w", err)
	}

	if account.IsAdmin {
		// 添加到管理员组
		addCmd := exec.CommandContext(ctx, "powershell", "-Command",
			fmt.Sprintf("Add-LocalGroupMember -Group 'Administrators' -Member %s", account.Username))
		if err := addCmd.Run(); err != nil {
			return fmt.Errorf("failed to add to admin group: %w", err)
		}
	}

	return nil
}

// createLinuxAccount 创建 Linux 账号
func (m *AccountManager) createLinuxAccount(ctx context.Context, account *Account) error {
	// 创建用户
	cmd := exec.CommandContext(ctx, "useradd", "-m", account.Username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// 设置密码
	if err := m.setLinuxPassword(ctx, account.Username, account.Password); err != nil {
		return err
	}

	if account.IsAdmin {
		// 配置 sudo 权限
		return m.configureSudo(ctx, account.Username)
	}

	return nil
}

// setLinuxPassword 设置 Linux 用户密码
func (m *AccountManager) setLinuxPassword(ctx context.Context, username, password string) error {
	cmd := exec.CommandContext(ctx, "chpasswd")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		fmt.Fprintf(stdin, "%s:%s\n", username, password)
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	return nil
}

// configureSudo 配置 sudo 权限
func (m *AccountManager) configureSudo(ctx context.Context, username string) error {
	// 写入 /etc/sudoers.d/ruoyi-agent
	sudoersContent := fmt.Sprintf(
		"%s ALL=(root) NOPASSWD: /usr/sbin/useradd, /usr/sbin/userdel, /usr/sbin/usermod, /usr/bin/passwd\n",
		username,
	)

	sudoersPath := fmt.Sprintf("/etc/sudoers.d/%s", username)

	// 使用 tee 写入文件
	cmd := exec.CommandContext(ctx, "tee", sudoersPath)
	cmd.Stdin = strings.NewReader(sudoersContent)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write sudoers file: %w", err)
	}

	// 设置权限
	chmodCmd := exec.CommandContext(ctx, "chmod", "440", sudoersPath)
	if err := chmodCmd.Run(); err != nil {
		return fmt.Errorf("failed to set sudoers permissions: %w", err)
	}

	return nil
}

// DeleteAccount 删除账号
func (m *AccountManager) DeleteAccount(ctx context.Context, username string) error {
	switch m.platform {
	case "windows":
		return exec.CommandContext(ctx, "powershell", "-Command",
			fmt.Sprintf("Remove-LocalUser -Name %s", username)).Run()
	case "linux":
		return exec.CommandContext(ctx, "userdel", "-r", username).Run()
	default:
		return fmt.Errorf("unsupported platform: %s", m.platform)
	}
}

// ResetPassword 重置密码
func (m *AccountManager) ResetPassword(ctx context.Context, username, newPassword string) error {
	switch m.platform {
	case "windows":
		return exec.CommandContext(ctx, "powershell", "-Command",
			fmt.Sprintf("Set-LocalUser -Name %s -Password (ConvertTo-SecureString %s -AsPlainText -Force)",
				username, newPassword)).Run()
	case "linux":
		return m.setLinuxPassword(ctx, username, newPassword)
	default:
		return fmt.Errorf("unsupported platform: %s", m.platform)
	}
}

// EnableAccount 启用账号
func (m *AccountManager) EnableAccount(ctx context.Context, username string) error {
	switch m.platform {
	case "windows":
		return exec.CommandContext(ctx, "powershell", "-Command",
			fmt.Sprintf("Enable-LocalUser -Name %s", username)).Run()
	case "linux":
		return exec.CommandContext(ctx, "usermod", "-U", username).Run()
	default:
		return fmt.Errorf("unsupported platform: %s", m.platform)
	}
}

// DisableAccount 禁用账号
func (m *AccountManager) DisableAccount(ctx context.Context, username string) error {
	switch m.platform {
	case "windows":
		return exec.CommandContext(ctx, "powershell", "-Command",
			fmt.Sprintf("Disable-LocalUser -Name %s", username)).Run()
	case "linux":
		return exec.CommandContext(ctx, "usermod", "-L", username).Run()
	default:
		return fmt.Errorf("unsupported platform: %s", m.platform)
	}
}

// ListAccounts 列出所有账号
func (m *AccountManager) ListAccounts(ctx context.Context) ([]string, error) {
	switch m.platform {
	case "windows":
		cmd := exec.CommandContext(ctx, "powershell", "-Command", "Get-LocalUser | Select-Object -ExpandProperty Name")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to list Windows users: %w", err)
		}
		return parseWindowsUsers(string(output)), nil
	case "linux":
		cmd := exec.CommandContext(ctx, "getent", "passwd")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to list Linux users: %w", err)
		}
		return parseLinuxUsers(string(output)), nil
	default:
		return nil, fmt.Errorf("unsupported platform: %s", m.platform)
	}
}

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
	users := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 1 && parts[0] != "" {
			// 只返回普通用户（UID >= 1000）
			if len(parts) > 2 {
				users = append(users, parts[0])
			}
		}
	}
	return users
}
