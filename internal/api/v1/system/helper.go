package system

import (
	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
)

// OperType constants are re-exported as local aliases pointing to the shared
// operlog package constants. Existing callers (e.g. the 9 AD domain handler
// sites that reference OperTypeCreate / OperTypeUpdate / OperTypeDelete /
// OperTypeOther) continue to compile without modification. Phase 34 Wave 7
// additionally aliases the new state/sync/reset verbs so handlers in this
// package can reference them without qualifying with the operlog package name.
const (
	OperTypeOther   = operlog.OperTypeOther
	OperTypeCreate  = operlog.OperTypeCreate
	OperTypeUpdate  = operlog.OperTypeUpdate
	OperTypeDelete  = operlog.OperTypeDelete
	OperTypeGrant   = operlog.OperTypeGrant
	OperTypeExport  = operlog.OperTypeExport
	OperTypeImport  = operlog.OperTypeImport
	OperTypeForce   = operlog.OperTypeForce
	OperTypeGenCode = operlog.OperTypeGenCode
	OperTypeClean   = operlog.OperTypeClean

	// Phase 34 Wave 7 additions — state / sync / reset verbs used by the
	// dashboard / column-config / notification-config / notice / OU-mapping /
	// AD-sync handlers.
	OperTypeStatus = operlog.OperTypeStatus
	OperTypeReset  = operlog.OperTypeReset
	OperTypeSync   = operlog.OperTypeSync
)

// recordOperLog records an operation log entry. Function signature is preserved
// (it takes *core.Core so the 9 AD domain handler callers are unchanged) but
// the body now delegates to operlog.Record so the actual recording logic lives
// in the shared package importable by every api/v1/* module.
func recordOperLog(c *gin.Context, core *core.Core, module string, operType int) {
	// Defensive nil-check: a test that injects a partial Core (e.g. one
	// without OperLogService) must not panic before the downstream Record's
	// own nil-guard can short-circuit.
	if core == nil {
		return
	}
	operlog.Record(c, core.OperLogService, core.GetDB(), module, operType)
}
