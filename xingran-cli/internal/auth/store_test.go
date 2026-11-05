package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// TestStoreSaveErrorPaths 覆盖 Save 的三个失败分支：
// 目录创建失败、临时文件写入失败、最终 rename 失败。
// 企业环境被安全软件拦截时正是走这些分支，错误信息必须带可操作建议。
func TestStoreSaveErrorPaths(t *testing.T) {
	pair := TokenPair{AccessToken: "x", RefreshToken: "y", ExpiresIn: 60}

	t.Run("目录创建失败", func(t *testing.T) {
		dir := t.TempDir()
		// 用一个已存在的普通文件占住本该是目录的位置 → MkdirAll 失败
		blocker := filepath.Join(dir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		st := &FileTokenStore{Path: filepath.Join(blocker, "sub", "tokens.json")}
		err := st.Save(nil, pair)
		if err == nil {
			t.Fatal("期望目录创建失败，实际成功")
		}
		if !strings.Contains(err.Error(), "创建 token 目录") ||
			!strings.Contains(err.Error(), EnvTokenFile) {
			t.Fatalf("错误信息应含路径与 %s 建议，实际: %v", EnvTokenFile, err)
		}
	})

	t.Run("临时文件写入失败", func(t *testing.T) {
		dir := t.TempDir()
		// 预先用目录占住 <path>.tmp，使 WriteFile 失败
		if err := os.MkdirAll(filepath.Join(dir, "tokens.json.tmp"), 0o700); err != nil {
			t.Fatal(err)
		}
		st := &FileTokenStore{Path: filepath.Join(dir, "tokens.json")}
		err := st.Save(nil, pair)
		if err == nil {
			t.Fatal("期望写入失败，实际成功")
		}
		if !strings.Contains(err.Error(), "写入 token 文件") ||
			!strings.Contains(err.Error(), EnvTokenFile) {
			t.Fatalf("错误信息应含 %s 建议，实际: %v", EnvTokenFile, err)
		}
	})

	t.Run("rename 失败", func(t *testing.T) {
		dir := t.TempDir()
		// 目标路径被一个非空目录占住 → rename 失败
		target := filepath.Join(dir, "tokens.json")
		if err := os.MkdirAll(filepath.Join(target, "inner"), 0o700); err != nil {
			t.Fatal(err)
		}
		st := &FileTokenStore{Path: target}
		err := st.Save(nil, pair)
		if err == nil {
			t.Fatal("期望 rename 失败，实际成功")
		}
		if !strings.Contains(err.Error(), "落盘 token 文件") ||
			!strings.Contains(err.Error(), EnvTokenFile) {
			t.Fatalf("错误信息应含 %s 建议，实际: %v", EnvTokenFile, err)
		}
	})
}

// TestStoreClearAndLoadErrorPaths 覆盖 Clear 的删除失败分支与 Load 的非 NotExist 错误分支。
func TestStoreClearAndLoadErrorPaths(t *testing.T) {
	dir := t.TempDir()

	t.Run("Clear 删除非空目录失败", func(t *testing.T) {
		target := filepath.Join(dir, "clear-dir")
		if err := os.MkdirAll(filepath.Join(target, "inner"), 0o700); err != nil {
			t.Fatal(err)
		}
		st := &FileTokenStore{Path: target}
		if err := st.Clear(nil); err == nil {
			t.Fatal("期望删除非空目录失败，实际成功")
		}
	})

	t.Run("Load 读取目录报错", func(t *testing.T) {
		target := filepath.Join(dir, "load-dir")
		if err := os.MkdirAll(target, 0o700); err != nil {
			t.Fatal(err)
		}
		st := &FileTokenStore{Path: target}
		if _, err := st.Load(nil); err == nil {
			t.Fatal("期望读取目录报错，实际成功")
		}
	})
}
