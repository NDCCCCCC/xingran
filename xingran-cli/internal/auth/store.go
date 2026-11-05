package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TokenStore token 持久化抽象。
type TokenStore interface {
	Save(ctx context.Context, pair TokenPair) error
	Load(ctx context.Context) (*TokenPair, error)
	Clear(ctx context.Context) error
}

// 环境变量（企业环境下 %USERPROFILE% 可能被 EDR/杀软拦截写入，提供自定义位置）：
//   XINGRAN_TOKEN_FILE 完整 token 文件路径（优先级最高）
//   XINGRAN_HOME       存放 tokens.json 的目录
const (
	EnvTokenFile = "XINGRAN_TOKEN_FILE"
	EnvHome      = "XINGRAN_HOME"
)

// DefaultStoreDir 返回 token 存储目录，优先级：
// XINGRAN_HOME > %USERPROFILE%/.xingran-cli。
func DefaultStoreDir() string {
	if d := os.Getenv(EnvHome); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".xingran-cli"
	}
	return filepath.Join(home, ".xingran-cli")
}

// FileTokenStore 文件存储实现：tokens.json，文件权限 0600，目录 0700。
// （Windows 上 0600 退化为只读位；POSIX 下严格生效。）
type FileTokenStore struct {
	Path string
}

// NewFileTokenStore 使用默认位置构造，优先级：
// XINGRAN_TOKEN_FILE（完整路径） > XINGRAN_HOME/tokens.json > %USERPROFILE%/.xingran-cli/tokens.json。
func NewFileTokenStore() *FileTokenStore {
	if f := os.Getenv(EnvTokenFile); f != "" {
		return &FileTokenStore{Path: f}
	}
	return &FileTokenStore{Path: filepath.Join(DefaultStoreDir(), "tokens.json")}
}

// Save 写入 token；失败时返回带绝对路径与排查建议的错误。
func (s *FileTokenStore) Save(_ context.Context, pair TokenPair) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("创建 token 目录 %s 失败: %w（若为企业环境，可能被安全软件拦截写入；"+
			"可用环境变量 %s 指定可写路径，例如 %s=D:\\tools\\xingran\\tokens.json）",
			dir, err, EnvTokenFile, EnvTokenFile)
	}
	// 先写临时文件再 rename，避免并发读写撕裂
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, []byte(marshalPair(pair)), 0o600); err != nil {
		return fmt.Errorf("写入 token 文件 %s 失败: %w（可用环境变量 %s 指定可写路径）",
			tmp, err, EnvTokenFile)
	}
	if err := os.Chmod(tmp, 0o600); err != nil { // 确保权限（覆盖 umask 影响）
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("落盘 token 文件 %s 失败: %w（可用环境变量 %s 指定可写路径）",
			s.Path, err, EnvTokenFile)
	}
	return nil
}

func (s *FileTokenStore) Load(_ context.Context) (*TokenPair, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 未登录
		}
		return nil, err
	}
	pair, err := unmarshalPair(data)
	if err != nil {
		return nil, err
	}
	return pair, nil
}

func (s *FileTokenStore) Clear(_ context.Context) error {
	err := os.Remove(s.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func marshalPair(pair TokenPair) string {
	if pair.ExpiresAt.IsZero() && pair.ExpiresIn > 0 {
		pair.ExpiresAt = time.Now().Add(time.Duration(pair.ExpiresIn) * time.Second)
	}
	b, _ := json.MarshalIndent(pair, "", "  ")
	return string(b)
}

func unmarshalPair(data []byte) (*TokenPair, error) {
	var pair TokenPair
	if err := json.Unmarshal(data, &pair); err != nil {
		return nil, err
	}
	if pair.ExpiresAt.IsZero() && pair.ExpiresIn > 0 {
		// 旧文件无 ExpiresAt：以文件修改时间近似已不可靠，标记为立即过期触发 refresh
		pair.ExpiresAt = time.Time{}
	}
	return &pair, nil
}
