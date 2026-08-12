//go:build windows

package core

import (
	"context"

	"github.com/sirupsen/logrus"
)

// startSubprocessReaper is a no-op on Windows.
// Windows doesn't have the zombie process problem; the OS reaps
// child processes automatically when the parent process terminates.
func (c *Core) startSubprocessReaper(ctx context.Context) {
	logrus.Debug("subprocess reaper: skipped on Windows (no-op)")
}
