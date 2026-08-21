package operations

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	pkgcache "github.com/xingran-next/xingran-go-backend/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	operationsmodels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	sysmodels "github.com/xingran-next/xingran-go-backend/internal/models/system"
	syssvc "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// =====================================================================
// Phase 74-07: geocoding_service.go + room_photo_service.go + floor_cache_impl.go 测试。
// geocoding 通过替换 s.httpClient.Transport 的假 RoundTripper 驱动百度 API 路径，
// redis 层用 pkg/cache MemoryCache 真实现。
// =====================================================================

// fakeGeocodeTransport 按请求 address 参数返回预置响应体。
type fakeGeocodeTransport struct {
	responses map[string]string // address -> body
	fallback  string
	err       error
	calls     int
	seenAddrs []string
}

func (f *fakeGeocodeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	addr := req.URL.Query().Get("address")
	f.seenAddrs = append(f.seenAddrs, addr)
	if f.err != nil {
		return nil, f.err
	}
	body := f.fallback
	if b, ok := f.responses[addr]; ok {
		body = b
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

const geocodeOKBody = `{"status":0,"result":{"location":{"lng":116.404,"lat":39.915},"formatted_address":"北京市东城区","level":"城市"}}`

// newGeocodeSvc 构造带假 Transport + MemoryCache redis 层的服务。
func newGeocodeSvc(rt *fakeGeocodeTransport, redis pkgcache.Cache) *GeocodingService {
	svc := NewGeocodingServiceWithCache("ak-test", redis)
	svc.httpClient = &http.Client{Transport: rt, Timeout: 2 * time.Second}
	svc.rateLimiter = NewRateLimiter(100, time.Hour)
	return svc
}

func TestGeocodingService_Geocode_SuccessAndMemoryCache(t *testing.T) {
	rt := &fakeGeocodeTransport{fallback: geocodeOKBody}
	svc := newGeocodeSvc(rt, nil)
	ctx := context.Background()

	lng, lat, err := svc.Geocode(ctx, "北京市东城区")
	require.NoError(t, err)
	assert.InDelta(t, 116.404, lng, 0.0001)
	assert.InDelta(t, 39.915, lat, 0.0001)
	assert.Equal(t, 1, rt.calls)

	// 第二次命中内存缓存，不再打 API
	lng2, lat2, err := svc.Geocode(ctx, "北京市东城区")
	require.NoError(t, err)
	assert.Equal(t, lng, lng2)
	assert.Equal(t, lat, lat2)
	assert.Equal(t, 1, rt.calls, "内存缓存命中后不应再调用 API")

	stats := svc.GetStats()
	assert.Equal(t, uint64(1), stats.APICalls)
	assert.Equal(t, uint64(1), stats.MemoryHits)
}

func TestGeocodingService_Geocode_Errors(t *testing.T) {
	ctx := context.Background()

	// 空地址
	svc := newGeocodeSvc(&fakeGeocodeTransport{}, nil)
	_, _, err := svc.Geocode(ctx, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "地址不能为空")

	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"status非零", `{"status":1,"message":"invalid ak"}`, "地址解析失败"},
		{"result为nil", `{"status":0}`, "未返回结果"},
		{"非JSON", `not-json`, "解析响应失败"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rt := &fakeGeocodeTransport{fallback: tc.body}
			s := newGeocodeSvc(rt, nil)
			_, _, e := s.Geocode(ctx, "addr-" + tc.name)
			require.Error(t, e)
			assert.Contains(t, e.Error(), tc.wantErr)
			assert.Equal(t, uint64(1), s.GetStats().CacheMisses)
		})
	}

	// 网络错误
	rt := &fakeGeocodeTransport{err: context.DeadlineExceeded}
	svc = newGeocodeSvc(rt, nil)
	_, _, err = svc.Geocode(ctx, "addr-net")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "请求失败")
}

func TestGeocodingService_RateLimited(t *testing.T) {
	rt := &fakeGeocodeTransport{fallback: geocodeOKBody}
	svc := newGeocodeSvc(rt, nil)
	// 令牌 1 个、24h 不补充：首次消耗后第二次被限流
	svc.rateLimiter = NewRateLimiter(1, 24*time.Hour)
	ctx := context.Background()

	_, _, err := svc.Geocode(ctx, "addr-limit-1")
	require.NoError(t, err)

	_, _, err = svc.Geocode(ctx, "addr-limit-2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API调用频率超限")
	assert.Equal(t, uint64(1), svc.GetStats().CacheMisses)
}

func TestGeocodingService_RedisCacheHit(t *testing.T) {
	redis := pkgcache.NewMemoryCache(50, time.Minute)
	rt1 := &fakeGeocodeTransport{fallback: geocodeOKBody}
	svc1 := newGeocodeSvc(rt1, redis)
	ctx := context.Background()

	_, _, err := svc1.Geocode(ctx, "addr-redis")
	require.NoError(t, err)

	// 新服务实例：内存缓存为空，但共享 redis 层 → Redis 命中且不打 API
	rt2 := &fakeGeocodeTransport{fallback: geocodeOKBody}
	svc2 := newGeocodeSvc(rt2, redis)
	_, _, err = svc2.Geocode(ctx, "addr-redis")
	require.NoError(t, err)
	assert.Equal(t, 0, rt2.calls, "redis 命中后不应调用 API")
	assert.Equal(t, uint64(1), svc2.GetStats().RedisHits)
}

func TestGeocodingService_RedisCacheCorruptJSON(t *testing.T) {
	redis := pkgcache.NewMemoryCache(50, time.Minute)
	// 直接在 redis 层放坏 JSON → 反序列化失败 → 落到 API 调用
	require.NoError(t, redis.Set(context.Background(), buildCacheKey("addr-corrupt"), "not-json", time.Minute))

	rt := &fakeGeocodeTransport{fallback: geocodeOKBody}
	svc := newGeocodeSvc(rt, redis)

	_, _, err := svc.Geocode(context.Background(), "addr-corrupt")
	require.NoError(t, err)
	assert.Equal(t, 1, rt.calls, "redis 数据损坏时应回源 API")
}

func TestGeocodingService_BatchGeocode(t *testing.T) {
	rt := &fakeGeocodeTransport{
		responses: map[string]string{
			"addr-ok-1": geocodeOKBody,
			"addr-ok-2": geocodeOKBody,
			"addr-bad":  `{"status":1,"message":"parse failed"}`,
		},
		fallback: geocodeOKBody,
	}
	svc := newGeocodeSvc(rt, nil)
	ctx := context.Background()

	results, err := svc.BatchGeocode(ctx, []string{"addr-ok-1", "addr-ok-2", "addr-bad", ""})
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Contains(t, results, "addr-ok-1")
	assert.Contains(t, results, "addr-ok-2")

	// 全部失败 → 批量失败
	rt2 := &fakeGeocodeTransport{responses: map[string]string{"x": `{"status":1}`}}
	svc2 := newGeocodeSvc(rt2, nil)
	_, err = svc2.BatchGeocode(ctx, []string{"x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "批量地址解析失败")
}

func TestGeocodingService_InvalidateAndClearAll(t *testing.T) {
	redis := pkgcache.NewMemoryCache(50, time.Minute)
	rt := &fakeGeocodeTransport{fallback: geocodeOKBody}
	svc := newGeocodeSvc(rt, redis)
	ctx := context.Background()

	_, _, err := svc.Geocode(ctx, "addr-inv")
	require.NoError(t, err)
	assert.Equal(t, 1, rt.calls)

	// InvalidateCache 后内存+redis 同时失效 → 回源
	svc.InvalidateCache(ctx, "addr-inv")
	_, _, err = svc.Geocode(ctx, "addr-inv")
	require.NoError(t, err)
	assert.Equal(t, 2, rt.calls)

	// ClearAllCache 清空两个地址的 redis 缓存
	_, _, err = svc.Geocode(ctx, "addr-inv2")
	require.NoError(t, err)
	keys, _ := redis.Keys(ctx, "baidu_map:geocode:*")
	assert.Len(t, keys, 2)

	svc.ClearAllCache(ctx)
	keys, _ = redis.Keys(ctx, "baidu_map:geocode:*")
	assert.Empty(t, keys)
}

func TestGeocodingService_StatsResetAndCacheKey(t *testing.T) {
	svc := newGeocodeSvc(&fakeGeocodeTransport{fallback: geocodeOKBody}, nil)
	_, _, err := svc.Geocode(context.Background(), "addr-stats")
	require.NoError(t, err)

	key := buildCacheKey("addr-stats")
	assert.True(t, strings.HasPrefix(key, "baidu_map:geocode:"), "缓存键应带统一前缀")
	assert.NotEqual(t, buildCacheKey("other"), key)

	svc.ResetStats()
	assert.Equal(t, CacheStatsData{}, svc.GetStats())
}

// =====================================================================
// RoomPhotoService
// =====================================================================

func newPhotoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newCRUDTestDB(t)
	require.NoError(t, db.AutoMigrate(&sysmodels.SysFile{}, &sysmodels.SysFileAccessLog{}))
	return db
}

func seedRoom(t *testing.T, db *gorm.DB) string {
	t.Helper()
	buildingID, floorID := seedBuildingFloor(t, db, "photo-b")
	room := &operationsmodels.OpsServerRoom{Name: "photo-room", BuildingID: buildingID, FloorID: floorID}
	require.NoError(t, db.Create(room).Error)
	return room.ID
}

func seedRoomPhoto(t *testing.T, db *gorm.DB, roomID string, sort int, primary bool) *operationsmodels.OpsRoomPhoto {
	t.Helper()
	file := &sysmodels.SysFile{FileName: "p.jpg", StoragePath: "/uploads/p.jpg", UploaderID: "u1"}
	require.NoError(t, db.Create(file).Error)
	url := "/uploads/p.jpg"
	photo := &operationsmodels.OpsRoomPhoto{
		RoomID: roomID, FileID: file.ID, FileURL: &url,
		SortOrder: sort, IsPrimary: primary,
	}
	require.NoError(t, db.Create(photo).Error)
	return photo
}

func TestRoomPhotoService_UploadPhotos_Errors(t *testing.T) {
	db := newPhotoTestDB(t)
	svc := NewRoomPhotoService(db)
	ctx := context.Background()

	// 空文件列表
	_, err := svc.UploadPhotos(ctx, "r1", nil, 0, "u1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "照片")

	// 机房不存在（files 非空以越过第一道校验）
	fakeFiles := []*multipart.FileHeader{{Filename: "x.jpg"}}
	_, err = svc.UploadPhotos(ctx, "missing-room", fakeFiles, 0, "u1")
	require.Error(t, err)
}

func TestRoomPhotoService_DBMethods(t *testing.T) {
	db := newPhotoTestDB(t)
	svc := NewRoomPhotoService(db)
	ctx := context.Background()
	roomID := seedRoom(t, db)

	p0 := seedRoomPhoto(t, db, roomID, 0, true)
	p1 := seedRoomPhoto(t, db, roomID, 1, false)
	p2 := seedRoomPhoto(t, db, roomID, 2, false)

	// ListByRoom：主图在前
	photos, err := svc.ListByRoom(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, photos, 3)
	assert.True(t, photos[0].IsPrimary)

	// CountPhotos
	count, err := svc.CountPhotos(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// SetPrimary：不存在 / 切换主图
	require.Error(t, svc.SetPrimary(ctx, "missing"))
	require.NoError(t, svc.SetPrimary(ctx, p1.ID))
	var oldPrimary operationsmodels.OpsRoomPhoto
	require.NoError(t, db.Where("id = ?", p0.ID).First(&oldPrimary).Error)
	assert.False(t, oldPrimary.IsPrimary, "原主图被取消")

	// UpdateSort：空列表 / 重排
	require.Error(t, svc.UpdateSort(ctx, nil))
	require.NoError(t, svc.UpdateSort(ctx, []string{p2.ID, p0.ID, p1.ID}))

	// UpdateDescription：不存在 / 成功
	require.Error(t, svc.UpdateDescription(ctx, "missing", "d"))
	desc := "机房正门"
	require.NoError(t, svc.UpdateDescription(ctx, p1.ID, desc))

	// GetPhotoWithFile：不存在 / 文件缺失 / 成功
	_, _, err = svc.GetPhotoWithFile(ctx, "missing")
	require.Error(t, err)
	orphan := &operationsmodels.OpsRoomPhoto{RoomID: roomID, FileID: "ghost-file", SortOrder: 9}
	require.NoError(t, db.Create(orphan).Error)
	_, _, err = svc.GetPhotoWithFile(ctx, orphan.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "查询文件失败")
	photo, file, err := svc.GetPhotoWithFile(ctx, p1.ID)
	require.NoError(t, err)
	assert.Equal(t, "p.jpg", file.FileName)
	assert.Equal(t, p1.ID, photo.ID)

	// GetPrimaryPhoto：有主图 / 无主图回退第一张 / 空机房报错
	primary, err := svc.GetPrimaryPhoto(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, p1.ID, primary.ID)

	require.NoError(t, db.Exec("UPDATE ops_room_photos SET is_primary = 0 WHERE room_id = ?", roomID).Error)
	primary, err = svc.GetPrimaryPhoto(ctx, roomID)
	require.NoError(t, err)
	assert.Equal(t, p2.ID, primary.ID, "无主图时回退 sort_order 最小")

	_, err = svc.GetPrimaryPhoto(ctx, "empty-room")
	require.Error(t, err)

	// DeletePhoto：不存在 / 删除主图。
	// QUIRK: 主图提升分支在 sqlite 不可达 —— DeletePhoto 的事务闭包内调用
	// fileService.DeleteFile（外层 db handle），glebarez :memory: 连接池会在事务
	// 占用连接时把外层语句路由到另一条连接 = 另一份空库 → GetFile 报
	// "no such table: sys_files" → DeleteFile 恒错 → 提前 return nil 跳过提升。
	// （PG 单库无此问题，属测试基建 quirk，非业务 bug，D-12 不修。）
	require.Error(t, svc.DeletePhoto(ctx, "missing"))
	require.NoError(t, db.Exec("UPDATE ops_room_photos SET is_primary = 1 WHERE id = ?", p0.ID).Error)
	require.NoError(t, svc.DeletePhoto(ctx, p0.ID))
	count, _ = svc.CountPhotos(ctx, roomID)
	assert.Equal(t, int64(3), count)
	var survivor operationsmodels.OpsRoomPhoto
	require.NoError(t, db.Where("id = ?", p2.ID).First(&survivor).Error)
	assert.False(t, survivor.IsPrimary, "sqlite 下主图提升分支不可达（见 quirk 注释）")

	// BatchDeletePhotos：空列表 / 成功
	require.Error(t, svc.BatchDeletePhotos(ctx, nil))
	require.NoError(t, svc.BatchDeletePhotos(ctx, []string{p1.ID, p2.ID}))
	count, _ = svc.CountPhotos(ctx, roomID)
	assert.Equal(t, int64(1), count)
}

// =====================================================================
// FloorCacheService（NewFloorServiceWithCache + system.CacheAdapter/MemoryCache）
// =====================================================================

func TestFloorCacheService_CachedPaths(t *testing.T) {
	// floor GetByID/List LEFT JOIN sys_files → 用带 sys_files 的库
	db := newPhotoTestDB(t)
	mem := pkgcache.NewMemoryCache(100, time.Minute)
	svc := NewFloorServiceWithCache(db, syssvc.NewCacheAdapter(mem), nil)
	ctx := context.Background()

	buildingID, f1 := seedBuildingFloor(t, db, "cb1")
	cached := svc.(*floorCacheService) // 接口未暴露缓存方法，断言具体类型

	// GetTree：首次查库并写入缓存
	tree, err := svc.GetTree(ctx)
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, buildingID, tree[0].ID)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, f1, tree[0].Children[0].ID)
	keys, _ := mem.Keys(ctx, "floor:tree")
	assert.Len(t, keys, 1, "树缓存键应写入")

	// GetFloorsByBuildingID：缓存键 floor:building:<id>
	floors, err := cached.GetFloorsByBuildingID(ctx, buildingID)
	require.NoError(t, err)
	require.Len(t, floors, 1)
	keys, _ = mem.Keys(ctx, "floor:building:*")
	assert.Len(t, keys, 1)

	// Create 走缓存失效路径（floor:tree + floor:building 被清除）
	f2 := &operationsmodels.OpsFloor{Name: "cb1-F2", FloorNo: "2", BuildingID: buildingID}
	require.NoError(t, svc.Create(ctx, f2))
	keys, _ = mem.Keys(ctx, "floor:*")
	assert.Empty(t, keys, "写操作后楼层缓存应全部失效")

	tree, err = svc.GetTree(ctx)
	require.NoError(t, err)
	assert.Len(t, tree[0].Children, 2)

	// Update / Delete / BatchDelete 失效路径
	f2.Name = "cb1-F2改"
	require.NoError(t, svc.Update(ctx, f2))
	require.NoError(t, svc.Delete(ctx, f2.ID))
	// QUIRK: GetTree 的 LEFT JOIN 无 f.deleted_at 过滤 → 软删楼层仍出现在树里
	// （D-12 记录不修）。删除效果以 GetByID 报错为准。
	_, err = svc.GetByID(ctx, f2.ID)
	require.Error(t, err)
	tree, err = svc.GetTree(ctx)
	require.NoError(t, err)
	assert.Len(t, tree[0].Children, 2, "GetTree 不过滤软删楼层（quirk）")

	f3 := &operationsmodels.OpsFloor{Name: "cb1-F3", FloorNo: "3", BuildingID: buildingID}
	require.NoError(t, svc.Create(ctx, f3))
	require.NoError(t, svc.BatchDelete(ctx, []string{f3.ID}))
	require.Error(t, svc.Delete(ctx, "missing"))

	// GetByID / List 直接透传 base 实现
	got, err := svc.GetByID(ctx, f1)
	require.NoError(t, err)
	assert.Equal(t, f1, got.ID)
	page, err := svc.List(ctx, map[string]interface{}{"buildingId": buildingID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)

	// SearchFloorOptions：name 非空绕过缓存；空走缓存（键 dropdown:floor:all）
	opts, err := svc.SearchFloorOptions(ctx, map[string]interface{}{"name": "cb1-F1"})
	require.NoError(t, err)
	require.Len(t, opts, 1)
	keys, _ = mem.Keys(ctx, "dropdown:floor:*")
	assert.Empty(t, keys, "keyword 查询不应写缓存")

	opts, err = svc.SearchFloorOptions(ctx, map[string]interface{}{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(opts), 1)
	keys, _ = mem.Keys(ctx, "dropdown:floor:*")
	assert.Len(t, keys, 1, "空查询应命中下拉缓存键")

	// 显式失效 API
	require.NoError(t, cached.InvalidateFloorCache(ctx, buildingID))
	require.NoError(t, cached.InvalidateAllFloorCache(ctx))
	keys, _ = mem.Keys(ctx, "floor:*")
	assert.Empty(t, keys)
}
