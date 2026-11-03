package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// storeSeq 每次调用生成唯一目录名，避免测试间存储状态污染。
var storeSeq atomic.Int64

func tempStorePath() string {
	return filepath.Join(os.TempDir(),
		fmt.Sprintf("xingran-cli-test-%d-%d", os.Getpid(), storeSeq.Add(1)), "tokens.json")
}

func TestFileTokenStoreRoundTrip(t *testing.T) {
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "tokens.json")}

	pair := TokenPair{
		AccessToken:  "at-1",
		RefreshToken: "rt-1",
		ExpiresIn:    300,
		TokenType:    "Bearer",
	}
	if err := store.Save(nil, pair); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	loaded, err := store.Load(nil)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}
	if loaded == nil || loaded.AccessToken != "at-1" || loaded.RefreshToken != "rt-1" {
		t.Fatalf("往返数据不一致: %+v", loaded)
	}
	if loaded.ExpiresAt.IsZero() || time.Until(loaded.ExpiresAt) <= 0 {
		t.Fatalf("ExpiresAt 未计算: %+v", loaded)
	}

	// 0600 权限（POSIX 严格校验；Windows 仅验证可读）
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(store.Path)
		if err != nil {
			t.Fatalf("Stat 失败: %v", err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("期望 0600 权限，实际 %v", fi.Mode().Perm())
		}
	}

	if err := store.Clear(nil); err != nil {
		t.Fatalf("Clear 失败: %v", err)
	}
	loaded, err = store.Load(nil)
	if err != nil || loaded != nil {
		t.Fatalf("Clear 后 Load 应返回 nil, got %v, %v", loaded, err)
	}
}

func TestFileTokenStoreLoadMissing(t *testing.T) {
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "nonexistent.json")}
	loaded, err := store.Load(nil)
	if err != nil {
		t.Fatalf("未登录时 Load 不应报错: %v", err)
	}
	if loaded != nil {
		t.Fatalf("期望 nil, got %+v", loaded)
	}
}

func TestDefaultStoreDir(t *testing.T) {
	dir := DefaultStoreDir()
	if dir == "" {
		t.Fatal("默认目录为空")
	}
	if filepath.Base(dir) != ".xingran-cli" {
		t.Fatalf("期望目录名 .xingran-cli, got %s", dir)
	}
}
