//go:build integration

package tests

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/xingran-next/xingran-go-backend/xingran-cli/internal/auth"
)

// tempTokenPath 每个测试独立的 token 存储路径（互不污染真实 ~/.xingran-cli）。
func tempTokenPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "tokens.json")
}

// asAPIError 用 errors.As 提取 *auth.APIError。
func asAPIError(err error) (*auth.APIError, bool) {
	var e *auth.APIError
	ok := errors.As(err, &e)
	return e, ok
}
