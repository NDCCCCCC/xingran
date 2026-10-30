package security

// Phase 60 D-04：错误 → SyncErrorReason 的纯函数映射。

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhase60_SyncErrorReasonFor(t *testing.T) {
	assert.Equal(t, "user_sync", syncErrorReasonFor(errors.New("db down")),
		"普通错误沿用既有默认原因码")

	assert.Equal(t, SyncReasonSoftDeleteMismatch,
		syncErrorReasonFor(fmt.Errorf("wrap: %w", ErrSoftDeleteIdentityMismatch)),
		"安全策略拒绝使用专用原因码")

	assert.Equal(t, SyncReasonSoftDeleteMismatch,
		syncErrorReasonFor(fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrSoftDeleteIdentityMismatch))),
		"多层包装后 errors.Is 仍应命中")
}
