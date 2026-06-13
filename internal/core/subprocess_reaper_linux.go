//go:build linux || darwin

package core

import (
	"context"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// startSubprocessReaper starts a goroutine that periodically reaps zombie
// child processes left behind after subprocess termination.
// This prevents zombie FD accumulation from VM agent account manager
// and other subprocess-based operations.
func (c *Core) startSubprocessReaper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		logrus.Info("subprocess reaper started")
		for {
			select {
			case <-ctx.Done():
				logrus.Info("subprocess reaper stopped")
				return
			case <-ticker.C:
				// Reap all available zombie children in a batch
				var status syscall.WaitStatus
				for {
					pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
					if err != nil || pid <= 0 {
						break
					}
					logrus.Debugf("reaper: collected zombie process pid=%d", pid)
				}
			}
		}
	}()
}
