package dbbackup

// router_test.go —— 恢复收尾缓存清理 flushRestoreCaches 的全分支单测。
// 直接驱动包级函数（注册闭包仅做参数捕获），不启动 gin、不连 Redis：
// 用可注入的 fake cache 覆盖 nil / 正常 / Keys 失败 / MDelete 失败 / 空键跳过。

import (
	"context"
	"errors"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/services/dbbackup"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// fakeCache 仅覆写 Keys/MDelete，其余方法不会被 flushRestoreCaches 触达。
type fakeCache struct {
	cache.Cache // 嵌入接口；未覆写方法在本测试路径不会被调用
	keys        []string
	keysErr     error
	delErr      error
	gotPatterns []string
	deletedKeys []string
}

func (f *fakeCache) Keys(_ context.Context, pattern string) ([]string, error) {
	f.gotPatterns = append(f.gotPatterns, pattern)
	if f.keysErr != nil {
		return nil, f.keysErr
	}
	return f.keys, nil
}

func (f *fakeCache) MDelete(_ context.Context, keys ...string) error {
	if f.delErr != nil {
		return f.delErr
	}
	f.deletedKeys = append(f.deletedKeys, keys...)
	return nil
}

func TestFlushRestoreCaches_NilCacheReturnsError(t *testing.T) {
	err := flushRestoreCaches(context.Background(), nil)
	if err == nil {
		t.Fatal("cache 未初始化时应返回错误")
	}
}

func TestFlushRestoreCaches_NormalDeletesMatchedKeys(t *testing.T) {
	fc := &fakeCache{keys: []string{"menu:tree", "role:1"}}
	err := flushRestoreCaches(context.Background(), fc)
	if err != nil {
		t.Fatalf("正常清理不应报错: %v", err)
	}
	// fake 对全部 10 个模式都返回同一组键 → 删除总数 = 10 × 2。
	if len(fc.deletedKeys) != 20 {
		t.Fatalf("应删除 20 个键（10 模式 × 2），实得 %v", fc.deletedKeys)
	}
	// strict| 前缀模式必须包含（对应 strictCacheKeyPrefix，竖线非笔误）
	found := false
	for _, p := range fc.gotPatterns {
		if p == "strict|*" {
			found = true
		}
	}
	if !found {
		t.Fatalf("清理模式应包含 strict|*，实得 %v", fc.gotPatterns)
	}
}

func TestFlushRestoreCaches_KeysErrorKeepsFirstErr(t *testing.T) {
	fc := &fakeCache{keysErr: errors.New("redis down")}
	err := flushRestoreCaches(context.Background(), fc)
	if err == nil {
		t.Fatal("Keys 失败应返回首个错误")
	}
	if !errors.Is(err, fc.keysErr) {
		t.Fatalf("错误应保留底层原因: %v", err)
	}
}

func TestFlushRestoreCaches_MDeleteErrorKeepsFirstErr(t *testing.T) {
	fc := &fakeCache{keys: []string{"k1"}, delErr: errors.New("delete boom")}
	err := flushRestoreCaches(context.Background(), fc)
	if err == nil {
		t.Fatal("MDelete 失败应返回首个错误")
	}
	if !errors.Is(err, fc.delErr) {
		t.Fatalf("错误应保留底层原因: %v", err)
	}
}

func TestFlushRestoreCaches_EmptyKeysSkipsDelete(t *testing.T) {
	fc := &fakeCache{keys: nil}
	if err := flushRestoreCaches(context.Background(), fc); err != nil {
		t.Fatalf("空键应静默跳过: %v", err)
	}
	if len(fc.deletedKeys) != 0 {
		t.Fatalf("空键不应触发删除: %v", fc.deletedKeys)
	}
}

// 编译期防退化：flushRestoreCaches 签名必须匹配 dbbackup.RestoreCacheFlusher。
var _ dbbackup.RestoreCacheFlusher = func(ctx context.Context) error {
	return flushRestoreCaches(ctx, nil)
}
