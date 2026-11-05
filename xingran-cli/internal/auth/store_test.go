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

// TestStoreEnvOverride 企业环境 %USERPROFILE% 写入被拦截时的逃生通道：
// XINGRAN_TOKEN_FILE / XINGRAN_HOME 必须生效。
func TestStoreEnvOverride(t *testing.T) {
	tmp := t.TempDir()

	t.Run("XINGRAN_HOME", func(t *testing.T) {
		t.Setenv(EnvHome, tmp)
		t.Setenv(EnvTokenFile, "")
		if got := DefaultStoreDir(); got != tmp {
			t.Fatalf("XINGRAN_HOME 未生效: got %s want %s", got, tmp)
		}
		st := NewFileTokenStore()
		if want := filepath.Join(tmp, "tokens.json"); st.Path != want {
			t.Fatalf("tokens 路径不正确: got %s want %s", st.Path, want)
		}
		if err := st.Save(nil, TokenPair{AccessToken: "a", RefreshToken: "r", ExpiresIn: 60}); err != nil {
			t.Fatalf("保存失败: %v", err)
		}
		got, err := st.Load(nil)
		if err != nil || got == nil || got.AccessToken != "a" {
			t.Fatalf("回读失败: %+v err=%v", got, err)
		}
	})

	t.Run("XINGRAN_TOKEN_FILE", func(t *testing.T) {
		custom := filepath.Join(tmp, "custom", "my-token.json")
		t.Setenv(EnvTokenFile, custom)
		st := NewFileTokenStore()
		if st.Path != custom {
			t.Fatalf("XINGRAN_TOKEN_FILE 未生效: got %s want %s", st.Path, custom)
		}
		if err := st.Save(nil, TokenPair{AccessToken: "b", RefreshToken: "r2", ExpiresIn: 60}); err != nil {
			t.Fatalf("保存失败（应自动创建父目录）: %v", err)
		}
		got, err := st.Load(nil)
		if err != nil || got == nil || got.AccessToken != "b" {
			t.Fatalf("回读失败: %+v err=%v", got, err)
		}
	})
}
