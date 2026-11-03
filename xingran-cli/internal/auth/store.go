package auth

import (
	"context"
	"encoding/json"
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

// DefaultStoreDir 返回默认 token 存储目录：%USERPROFILE%/.xingran-cli/。
func DefaultStoreDir() string {
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

// NewFileTokenStore 使用默认目录构造。
func NewFileTokenStore() *FileTokenStore {
	return &FileTokenStore{Path: filepath.Join(DefaultStoreDir(), "tokens.json")}
}

func (s *FileTokenStore) Save(_ context.Context, pair TokenPair) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	// 先写临时文件再 rename，避免并发读写撕裂
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, []byte(marshalPair(pair)), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil { // 确保权限（覆盖 umask 影响）
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		_ = os.Remove(tmp)
		return err
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
