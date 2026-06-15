package system

import (
	"github.com/gin-gonic/gin"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/utils/operlog"
)

// OperType constants are re-exported as local aliases pointing to the shared
// operlog package constants. Existing callers (e.g. the 9 AD domain handler
// sites that reference OperTypeCreate / OperTypeUpdate / OperTypeDelete /
// OperTypeOther) continue to compile without modification.
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
)

// recordOperLog records an operation log entry. Function signature is preserved
// (it takes *core.Core so the 9 AD domain handler callers are unchanged) but
// the body now delegates to operlog.Record so the actual recording logic lives
// in the shared package importable by every api/v1/* module.
func recordOperLog(c *gin.Context, core *core.Core, module string, operType int) {
	operlog.Record(c, core.OperLogService, core.GetDB(), module, operType)
}
