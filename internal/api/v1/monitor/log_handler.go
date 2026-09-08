package monitor

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/utils/operlog"
	apperrors "github.com/xingran-next/xingran-go-backend/pkg/errors"
	"github.com/xingran-next/xingran-go-backend/pkg/response"
)

// logMutator defines the two mutation methods shared between OperLogService and LoginLogService.
type logMutator interface {
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
}

// MonitorLogHandler[T] provides shared delete methods for log-type handlers.
// Delete and BatchDelete are identical across OperLog and LoginLog.
type MonitorLogHandler[T any] struct {
	svc            logMutator
	moduleName     string
	operTypeDelete operlog.OperType
	operTypeBatch  operlog.OperType
	needOperlog    bool // true for LoginLog (calls operlog after mutations), false for OperLog
	core           *core.Core
}

// NewMonitorLogHandler creates a MonitorLogHandler for the shared mutation methods.
func NewMonitorLogHandler[T any](
	svc logMutator,
	moduleName string,
	operTypeDelete, operTypeBatch operlog.OperType,
	needOperlog bool,
) *MonitorLogHandler[T] {
	return &MonitorLogHandler[T]{
		svc:            svc,
		moduleName:     moduleName,
		operTypeDelete: operTypeDelete,
		operTypeBatch:  operTypeBatch,
		needOperlog:   needOperlog,
	}
}

// WithCore sets the core dependency (for operlog after mutations).
func (h *MonitorLogHandler[T]) WithCore(core *core.Core) *MonitorLogHandler[T] {
	h.core = core
	return h
}

// Delete removes a single log entry.
func (h *MonitorLogHandler[T]) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.Error(c, apperrors.ParamMissing("ID"))
		return
	}
	err := h.svc.Delete(c.Request.Context(), id)
	if !response.HandleServiceError(c, err, "删除"+h.moduleName) {
		return
	}
	if h.core != nil && h.needOperlog {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), h.moduleName, h.operTypeDelete)
	}
	response.Success(c, nil)
}

// BatchDelete removes multiple log entries.
func (h *MonitorLogHandler[T]) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required,min=1"`
	}
	if !response.HandleJSONBinding(c, &req) {
		return
	}
	err := h.svc.BatchDelete(c.Request.Context(), req.IDs)
	if !response.HandleServiceError(c, err, "批量删除"+h.moduleName) {
		return
	}
	if h.core != nil && h.needOperlog {
		operlog.Record(c, h.core.OperLogService, h.core.GetDB(), h.moduleName, h.operTypeBatch)
	}
	response.Success(c, nil)
}
