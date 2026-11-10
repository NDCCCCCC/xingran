package operations

// B-3 / v1.55 B-7 补测（GOV-03 diff coverage）：覆盖 room_photo_handler.go
// 8 个 service 错误分支对 writeScopeAwareServiceError 的调用。
//
// 手法：newTestCore(t) 提供空内存 sqlite（不 AutoMigrate 照片表），
// 每个 handler 的 service 查询必然返回 error → 走 writeScopeAwareServiceError。
// 闭包式 handler 用 gin.CreateTestContext 直接调用；uploadPhotos 需 multipart +
// user_id，单独构造。全部零网络、零生产数据。

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func invokeRoomPhotoHandler(t *testing.T, h gin.HandlerFunc, method, path, body string, params gin.Params, userID string, multipartBody *strings.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var req *http.Request
	if multipartBody != nil {
		req = httptest.NewRequest(method, path, multipartBody)
		req.Header.Set("Content-Type", contentType)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	if len(params) > 0 {
		c.Params = params
	}
	if userID != "" {
		c.Set("user_id", userID)
	}
	h(c)
	return w
}

func TestB7_RoomPhotoServiceErrorBranches(t *testing.T) {
	c := newTestCore(t)
	// 空 DB 无照片表 → 每个 service 调用都返回 error
	pid := gin.Params{{Key: "id", Value: "photo-1"}}
	rid := gin.Params{{Key: "roomId", Value: "room-1"}}

	cases := []struct {
		name   string
		h      gin.HandlerFunc
		method string
		path   string
		body   string
		params gin.Params
		userID string
	}{
		{"listPhotos", listPhotos(c), http.MethodGet, "/list/room-1", "", rid, ""},
		{"getPhoto", getPhoto(c), http.MethodGet, "/photo-1", "", pid, ""},
		{"setPrimary", setPrimary(c), http.MethodPut, "/photo-1/primary", "", pid, ""},
		{"updateDescription", updateDescription(c), http.MethodPut, "/photo-1/description", `{"description":"x"}`, pid, ""},
		{"updateSort", updateSort(c), http.MethodPut, "/sort", `{"photoIds":["photo-1"]}`, nil, ""},
		{"deletePhoto", deletePhoto(c), http.MethodDelete, "/photo-1", "", pid, ""},
		{"batchDeletePhotos", batchDeletePhotos(c), http.MethodPost, "/batch-delete", `{"ids":["photo-1"]}`, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := invokeRoomPhotoHandler(t, tc.h, tc.method, tc.path, tc.body, tc.params, tc.userID, nil, "")
			// service error 走 writeScopeAwareServiceError：普通 error → 500
			assert.Equal(t, http.StatusInternalServerError, w.Code,
				"%s 的 service 错误分支必须走 writeScopeAwareServiceError（500）", tc.name)
		})
	}
}

func TestB7_UploadPhotosServiceErrorBranch(t *testing.T) {
	c := newTestCore(t)
	// uploadPhotos 需 multipart 表单 + user_id + roomId；空 DB 让 UploadPhotos 失败。
	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("roomId", "room-1"))
	fw, err := mw.CreateFormFile("files", "a.png")
	require.NoError(t, err)
	_, err = fw.Write([]byte("fake-png"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	w := invokeRoomPhotoHandler(t, uploadPhotos(c), http.MethodPost, "/upload", "", nil,
		"user-1", strings.NewReader(buf.String()), mw.FormDataContentType())
	assert.Equal(t, http.StatusInternalServerError, w.Code,
		"uploadPhotos 的 service 错误分支必须走 writeScopeAwareServiceError（500）")
}
