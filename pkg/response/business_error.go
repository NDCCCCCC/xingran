package response

// BusinessError carries semantic HTTP status and business error code for
// recoverable business conflicts (e.g., "resource in use", "does not belong").
// HandleServiceError (handler_helpers.go) recognizes *BusinessError and returns
// its HTTPStatus instead of the default 500.
//
// Usage: any service layer needing semantic error codes can return *BusinessError.
// Example:
//
//	return nil, &BusinessError{HTTPStatus: 409, Code: 409001, Message: "设备正在执行其他任务"}
type BusinessError struct {
	// HTTPStatus is the HTTP response status code (e.g., 409 Conflict, 400 Bad Request).
	HTTPStatus int
	// Code is the domain-specific error code (e.g., 409001 for "resource conflict").
	Code int
	// Message is the human-readable error description.
	Message string
}

// Error implements the error interface.
func (e *BusinessError) Error() string {
	return e.Message
}
