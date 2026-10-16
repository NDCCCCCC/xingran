package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GUARD-04: TestHandleGetByID_Returns404_NotBadRequest
//
// RED baseline:  HandleGetByID calls Error(c, http.StatusNotFound, ...) at
// handler_helpers.go:62.  The Error() function accepts an int, which
// toAppError() at response.go:157 treats as an ErrorCode rather than an
// HTTPStatus — resulting in HTTP 400 instead of 404.
//
// GREEN after Phase 112 HANDLER-05: Handler uses apperrors.New(404, ...)
// so toAppError returns HTTP 404.
//
// This test is a regression guard ensuring the fix produces HTTP 404.
func TestHandleGetByID_Returns404_NotBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Simulate URL param "id" so HandleIDParam returns ok=true
	c, w := newRouterCtx(t, "")
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}

	// Getter returns an error (simulates "entity not found")
	getter := func(id string) (interface{}, error) {
		return nil, errors.New("entity not found")
	}

	// Call HandleGetByID — currently maps http.StatusNotFound(int) to HTTP 400
	HandleGetByID(c, getter, "实体不存在")

	// Assert HTTP 404, not 400
	require.False(t, HandleGetByID(c, getter, "实体不存在"),
		"HandleGetByID should return false when getter errors")
	assert.Equal(t, http.StatusNotFound, w.Code,
		"HandleGetByID should return HTTP 404 when getter returns error, got %d", w.Code)
	assert.Contains(t, w.Body.String(), "实体不存在",
		"Response should contain the notFoundMessage")
}
