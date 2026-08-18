package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestErrorCode_DefaultHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		code       ErrorCode
		wantStatus int
	}{
		{"Success", CodeSuccess, http.StatusOK},
		{"ParamError", CodeParamError, http.StatusBadRequest},
		{"Unauthorized", CodeUnauthorized, http.StatusUnauthorized},
		{"TokenExpired", CodeTokenExpired, http.StatusUnauthorized},
		{"UserNotFound", CodeUserNotFound, http.StatusBadRequest},
		{"BuildingNotFound", CodeBuildingNotFound, http.StatusBadRequest},
		{"ServerError", CodeServerError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.DefaultHTTPStatus(); got != tt.wantStatus {
				t.Errorf("ErrorCode.DefaultHTTPStatus() = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func TestErrorCode_DefaultMessage(t *testing.T) {
	tests := []struct {
		name        string
		code        ErrorCode
		wantMessage string
	}{
		{"Success", CodeSuccess, "成功"},
		{"ParamError", CodeParamError, "参数错误"},
		{"UserNotFound", CodeUserNotFound, "用户不存在"},
		{"BuildingNotFound", CodeBuildingNotFound, "楼宇不存在"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.DefaultMessage(); got != tt.wantMessage {
				t.Errorf("ErrorCode.DefaultMessage() = %v, want %v", got, tt.wantMessage)
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	t.Run("Without underlying error", func(t *testing.T) {
		err := New(CodeUserNotFound, "test message")
		if got := err.Error(); got != "[2010] test message" {
			t.Errorf("AppError.Error() = %v, want [2010] test message", got)
		}
	})

	t.Run("With underlying error", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := Wrap(underlying, CodeUserNotFound, "test message")
		if got := err.Error(); got != "[2010] test message: underlying error" {
			t.Errorf("AppError.Error() = %v, want [2010] test message: underlying error", got)
		}
	})
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := Wrap(underlying, CodeUserNotFound, "test message")

	if got := err.Unwrap(); got != underlying {
		t.Errorf("AppError.Unwrap() = %v, want %v", got, underlying)
	}
}

func TestAppError_WithHTTPStatus(t *testing.T) {
	err := New(CodeUserNotFound, "test")
	if got := err.GetHTTPStatus(); got != http.StatusBadRequest {
		t.Errorf("AppError.GetHTTPStatus() = %v, want %v", got, http.StatusBadRequest)
	}

	customStatusErr := NewWithHTTPStatus(CodeUserNotFound, http.StatusCreated, "test")
	if got := customStatusErr.GetHTTPStatus(); got != http.StatusCreated {
		t.Errorf("AppError.GetHTTPStatus() (custom) = %v, want %v", got, http.StatusCreated)
	}
}

func TestAppError_WithContext(t *testing.T) {
	err := New(CodeUserNotFound, "test").
		WithContext("user_id", "123").
		WithContext("username", "testuser")

	if err.Context == nil {
		t.Fatal("Context should not be nil")
	}

	if got, ok := err.Context["user_id"]; !ok || got != "123" {
		t.Errorf("Context[\"user_id\"] = %v, want 123", got)
	}

	if got, ok := err.Context["username"]; !ok || got != "testuser" {
		t.Errorf("Context[\"username\"] = %v, want testuser", got)
	}
}

// TestWrap_NilError 锁定 Wrap 的文档契约(errors.go:86-89):
// 即使 err == nil 也返回 *AppError(不返回 nil),避免调用方拿到
// nil error 后误判为"成功"。如需 nil 透传语义,用 WrapWithHTTPStatus。
func TestWrap_NilError(t *testing.T) {
	var err error = nil
	wrapped := Wrap(err, CodeUserNotFound, "test")
	if wrapped == nil {
		t.Fatal("Wrap(nil, ...) should return non-nil *AppError per documented contract (see errors.go:86)")
	}
	if wrapped.Code != CodeUserNotFound {
		t.Errorf("Wrap(nil).Code = %v, want %v", wrapped.Code, CodeUserNotFound)
	}
	// 对照组:WrapWithHTTPStatus 保持 nil 透传语义
	if got := WrapWithHTTPStatus(err, CodeUserNotFound, http.StatusBadRequest, "test"); got != nil {
		t.Errorf("WrapWithHTTPStatus(nil, ...) should return nil, got %v", got)
	}
}

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name       string
		fn         func() *AppError
		wantCode   ErrorCode
		wantStatus int
	}{
		{"UserNotFound", UserNotFound, CodeUserNotFound, http.StatusBadRequest},
		{"UserExists", UserExists, CodeUserExists, http.StatusBadRequest},
		{"UserNotFoundWithID", func() *AppError { return UserNotFoundWithID("123") }, CodeUserNotFound, http.StatusBadRequest},
		{"BuildingNotFound", BuildingNotFound, CodeBuildingNotFound, http.StatusBadRequest},
		{"BuildingNotFoundWithID", func() *AppError { return BuildingNotFoundWithID("456") }, CodeBuildingNotFound, http.StatusBadRequest},
		{"ParamError", ParamError, CodeParamError, http.StatusBadRequest},
		{"DatabaseError", func() *AppError { return DatabaseError(errors.New("db error")) }, CodeDatabaseError, CodeDatabaseError.DefaultHTTPStatus()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err.Code != tt.wantCode {
				t.Errorf("convenience function returned code %v, want %v", err.Code, tt.wantCode)
			}
			if err.GetHTTPStatus() != tt.wantStatus {
				t.Errorf("convenience function returned HTTP status %v, want %v", err.GetHTTPStatus(), tt.wantStatus)
			}
		})
	}
}

func TestIsAppError(t *testing.T) {
	t.Run("With AppError", func(t *testing.T) {
		err := UserNotFound()
		if !IsAppError(err) {
			t.Error("IsAppError(AppError) should return true")
		}
	})

	t.Run("With standard error", func(t *testing.T) {
		err := errors.New("standard error")
		if IsAppError(err) {
			t.Error("IsAppError(standard error) should return false")
		}
	})

	t.Run("With nil", func(t *testing.T) {
		if IsAppError(nil) {
			t.Error("IsAppError(nil) should return false")
		}
	})
}

func TestGetAppError(t *testing.T) {
	t.Run("With AppError", func(t *testing.T) {
		err := UserNotFound()
		if got := GetAppError(err); got == nil {
			t.Error("GetAppError(AppError) should not return nil")
		}
	})

	t.Run("With wrapped AppError", func(t *testing.T) {
		appErr := UserNotFound()
		wrapped := fmt.Errorf("wrapped: %w", appErr)
		if got := GetAppError(wrapped); got == nil {
			t.Error("GetAppError(wrapped AppError) should not return nil")
		}
	})

	t.Run("With standard error", func(t *testing.T) {
		err := errors.New("standard error")
		if got := GetAppError(err); got != nil {
			t.Error("GetAppError(standard error) should return nil")
		}
	})
}

func TestGetErrorCode(t *testing.T) {
	t.Run("With AppError", func(t *testing.T) {
		err := UserNotFound()
		if got := GetErrorCode(err); got != CodeUserNotFound {
			t.Errorf("GetErrorCode(AppError) = %v, want %v", got, CodeUserNotFound)
		}
	})

	t.Run("With standard error", func(t *testing.T) {
		err := errors.New("standard error")
		if got := GetErrorCode(err); got != CodeServerError {
			t.Errorf("GetErrorCode(standard error) = %v, want %v", got, CodeServerError)
		}
	})
}
