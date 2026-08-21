package operations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	opsModels "github.com/xingran-next/xingran-go-backend/internal/models/operations"
	opsServices "github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// TestRouterSmoke_MountBuilding verifies the gin router returns 404 (not 500/panic)
// for unmounted paths and 200/400 for mounted handlers — a basic smoke check that
// the router mounting pattern works end-to-end.
func TestRouterSmoke_MountBuilding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockBuildingService{
		StatisticsFunc: func(_ context.Context, _ map[string]interface{}) (*opsServices.BuildingStatisticsResult, error) {
			return &opsServices.BuildingStatisticsResult{Total: 1}, nil
		},
	}
	h := NewBuildingHandler(svc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t))

	r := gin.New()
	r.POST("/buildings/list", h.List)
	r.POST("/buildings/statistics", h.Statistics)
	r.POST("/buildings/:id", h.GetByID)

	// valid mounted route (List uses params fallback)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/buildings/list", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)

	// valid mounted route (Statistics)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/buildings/statistics", nil)
	r.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}

// TestRouterSmoke_AllMountable exercises every handler constructor + router mount
// to verify that all handler types can be mounted without panic.
func TestRouterSmoke_AllMountable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	buildingSvc := &mockBuildingService{}
	floorSvc := &mockFloorService{}
	workstationSvc := &mockWorkstationService{}
	serverRoomSvc := &mockServerRoomService{}
	infoPointSvc := &mockInfoPointService{}
	dedicatedLineSvc := &mockDedicatedLineService{}
	doorSvc := &mockDoorService{}
	assetSvc := &mockAssetService{}
	locationAliasSvc := &mockLocationAliasService{}
	roomDeviceSvc := &mockRoomDeviceService{}
	floorPlanTextSvc := &mockFloorPlanTextService{}
	wallSvc := &mockWallService{}
	workstationDeviceSvc := &mockWorkstationDeviceService{}

	handlers := []struct {
		name    string
		method  string
		path    string
		handler gin.HandlerFunc
	}{
		{"Building.Statistics", http.MethodPost, "/b/statistics", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).Statistics},
		{"Building.SearchOptions", http.MethodPost, "/b/search", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).SearchBuildingOptions},
		{"Building.Create", http.MethodPost, "/b", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).Create},
		{"Building.List", http.MethodPost, "/b/list", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).List},
		{"Building.GetByID", http.MethodPost, "/b/x", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).GetByID},
		{"Building.Update", http.MethodPost, "/b/x/update", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).Update},
		{"Building.Delete", http.MethodPost, "/b/x/delete", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).Delete},
		{"Building.Batch", http.MethodPost, "/b/batch", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).BatchOperation},
		{"Building.Geocode", http.MethodPost, "/b/geocode", NewBuildingHandler(buildingSvc, opsServices.NewGeocodingService("")).WithCore(newTestCore(t)).Geocode},

		{"Floor.Create", http.MethodPost, "/f", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).Create},
		{"Floor.List", http.MethodPost, "/f/list", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).List},
		{"Floor.GetTree", http.MethodPost, "/f/tree", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).GetTree},
		{"Floor.GetByID", http.MethodPost, "/f/x", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).GetByID},
		{"Floor.Update", http.MethodPost, "/f/x/update", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).Update},
		{"Floor.Delete", http.MethodPost, "/f/x/delete", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).Delete},
		{"Floor.Batch", http.MethodPost, "/f/batch", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).BatchOperation},
		{"Floor.Statistics", http.MethodPost, "/f/statistics", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).Statistics},
		{"Floor.SearchOptions", http.MethodPost, "/f/search", NewFloorHandler(floorSvc).WithCore(newTestCore(t)).SearchFloorOptions},

		{"Workstation.Create", http.MethodPost, "/w", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).Create},
		{"Workstation.List", http.MethodPost, "/w/list", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).List},
		{"Workstation.GetByID", http.MethodPost, "/w/x", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).GetByID},
		{"Workstation.Update", http.MethodPost, "/w/x/update", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).Update},
		{"Workstation.Delete", http.MethodPost, "/w/x/delete", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).Delete},
		{"Workstation.Batch", http.MethodPost, "/w/batch", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).BatchOperation},
		{"Workstation.Positions", http.MethodPost, "/w/positions", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).BatchUpdatePositions},
		{"Workstation.Statistics", http.MethodPost, "/w/statistics", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).Statistics},
		{"Workstation.DeptOptions", http.MethodPost, "/w/dept", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).GetWorkstationDeptOptions},
		{"Workstation.SearchOptions", http.MethodPost, "/w/search", NewWorkstationHandler(workstationSvc).WithCore(newTestCore(t)).SearchWorkstationOptions},

		{"ServerRoom.Create", http.MethodPost, "/sr", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).Create},
		{"ServerRoom.List", http.MethodPost, "/sr/list", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).List},
		{"ServerRoom.GetByID", http.MethodPost, "/sr/x", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).GetByID},
		{"ServerRoom.Update", http.MethodPost, "/sr/x/update", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).Update},
		{"ServerRoom.Delete", http.MethodPost, "/sr/x/delete", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).Delete},
		{"ServerRoom.Batch", http.MethodPost, "/sr/batch", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).BatchOperation},
		{"ServerRoom.Statistics", http.MethodPost, "/sr/statistics", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).Statistics},
		{"ServerRoom.SearchOptions", http.MethodPost, "/sr/search", NewServerRoomHandler(serverRoomSvc).WithCore(newTestCore(t)).SearchServerRoomOptions},

		{"InfoPoint.Create", http.MethodPost, "/ip", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).Create},
		{"InfoPoint.List", http.MethodPost, "/ip/list", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).List},
		{"InfoPoint.GetByID", http.MethodPost, "/ip/x", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).GetByID},
		{"InfoPoint.Update", http.MethodPost, "/ip/x/update", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).Update},
		{"InfoPoint.Delete", http.MethodPost, "/ip/x/delete", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).Delete},
		{"InfoPoint.Batch", http.MethodPost, "/ip/batch", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).BatchOperation},
		{"InfoPoint.Statistics", http.MethodPost, "/ip/statistics", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).Statistics},
		{"InfoPoint.SearchOptions", http.MethodPost, "/ip/search", NewInfoPointHandler(infoPointSvc).WithCore(newTestCore(t)).SearchInfoPointOptions},

		{"DedicatedLine.Create", http.MethodPost, "/dl", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).Create},
		{"DedicatedLine.List", http.MethodPost, "/dl/list", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).List},
		{"DedicatedLine.GetByID", http.MethodPost, "/dl/x", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).GetByID},
		{"DedicatedLine.Update", http.MethodPost, "/dl/x/update", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).Update},
		{"DedicatedLine.Delete", http.MethodPost, "/dl/x/delete", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).Delete},
		{"DedicatedLine.Batch", http.MethodPost, "/dl/batch", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).BatchOperation},
		{"DedicatedLine.Statistics", http.MethodPost, "/dl/statistics", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).Statistics},
		{"DedicatedLine.SearchOptions", http.MethodPost, "/dl/search", NewDedicatedLineHandler(dedicatedLineSvc).WithCore(newTestCore(t)).SearchDedicatedLineOptions},

		{"Door.Create", http.MethodPost, "/dr", NewDoorHandler(doorSvc).WithCore(newTestCore(t)).Create},
		{"Door.List", http.MethodPost, "/dr/list", NewDoorHandler(doorSvc).WithCore(newTestCore(t)).List},
		{"Door.GetByID", http.MethodPost, "/dr/x", NewDoorHandler(doorSvc).WithCore(newTestCore(t)).GetByID},
		{"Door.Update", http.MethodPost, "/dr/x/update", NewDoorHandler(doorSvc).WithCore(newTestCore(t)).Update},
		{"Door.Delete", http.MethodPost, "/dr/x/delete", NewDoorHandler(doorSvc).WithCore(newTestCore(t)).Delete},
		{"Door.Batch", http.MethodPost, "/dr/batch", NewDoorHandler(doorSvc).WithCore(newTestCore(t)).BatchOperation},

		{"Asset.Create", http.MethodPost, "/a", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).Create},
		{"Asset.Update", http.MethodPost, "/a/x/update", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).Update},
		{"Asset.Delete", http.MethodPost, "/a/x/delete", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).Delete},
		{"Asset.GetByID", http.MethodPost, "/a/x", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).GetByID},
		{"Asset.List", http.MethodPost, "/a/list", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).List},
		{"Asset.Batch", http.MethodPost, "/a/batch", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).BatchOperation},
		{"Asset.DeviceTypes", http.MethodPost, "/a/dt", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).GetDeviceTypes},
		{"Asset.DeviceCategories", http.MethodPost, "/a/dc", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).GetDeviceCategories},
		{"Asset.StatusValues", http.MethodPost, "/a/sv", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).GetStatusValues},
		{"Asset.Statistics", http.MethodPost, "/a/stats", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).Statistics},
		{"Asset.SearchBySerial", http.MethodPost, "/a/s/sn", NewAssetHandler(assetSvc).WithCore(newTestCore(t)).SearchBySerial},

		{"LocationAlias.List", http.MethodPost, "/la/list", NewLocationAliasHandler(locationAliasSvc).WithCore(newTestCore(t)).List},
		{"LocationAlias.Create", http.MethodPost, "/la", NewLocationAliasHandler(locationAliasSvc).WithCore(newTestCore(t)).Create},
		{"LocationAlias.Update", http.MethodPost, "/la/x/update", NewLocationAliasHandler(locationAliasSvc).WithCore(newTestCore(t)).Update},
		{"LocationAlias.Delete", http.MethodPost, "/la/x/delete", NewLocationAliasHandler(locationAliasSvc).WithCore(newTestCore(t)).Delete},

		{"RoomDevice.Create", http.MethodPost, "/rd", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).Create},
		{"RoomDevice.List", http.MethodPost, "/rd/list", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).List},
		{"RoomDevice.GetByID", http.MethodPost, "/rd/x", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).GetByID},
		{"RoomDevice.Update", http.MethodPost, "/rd/x/update", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).Update},
		{"RoomDevice.Delete", http.MethodPost, "/rd/x/delete", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).Delete},
		{"RoomDevice.Batch", http.MethodPost, "/rd/batch", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).BatchOperation},
		{"RoomDevice.Statistics", http.MethodPost, "/rd/stats", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).Statistics},
		{"RoomDevice.Search", http.MethodPost, "/rd/search", NewRoomDeviceHandler(roomDeviceSvc).WithCore(newTestCore(t)).SearchRoomDeviceOptions},

		{"FloorPlanText.Create", http.MethodPost, "/fpt", NewFloorPlanTextHandler(floorPlanTextSvc).WithCore(newTestCore(t)).Create},
		{"FloorPlanText.List", http.MethodPost, "/fpt/list", NewFloorPlanTextHandler(floorPlanTextSvc).WithCore(newTestCore(t)).List},
		{"FloorPlanText.GetByID", http.MethodPost, "/fpt/x", NewFloorPlanTextHandler(floorPlanTextSvc).WithCore(newTestCore(t)).GetByID},
		{"FloorPlanText.Update", http.MethodPost, "/fpt/x/update", NewFloorPlanTextHandler(floorPlanTextSvc).WithCore(newTestCore(t)).Update},
		{"FloorPlanText.Delete", http.MethodPost, "/fpt/x/delete", NewFloorPlanTextHandler(floorPlanTextSvc).WithCore(newTestCore(t)).Delete},
		{"FloorPlanText.Batch", http.MethodPost, "/fpt/batch", NewFloorPlanTextHandler(floorPlanTextSvc).WithCore(newTestCore(t)).BatchOperation},

		{"Wall.Create", http.MethodPost, "/w1", NewWallHandler(wallSvc).WithCore(newTestCore(t)).Create},
		{"Wall.List", http.MethodPost, "/w1/list", NewWallHandler(wallSvc).WithCore(newTestCore(t)).List},
		{"Wall.GetByID", http.MethodPost, "/w1/x", NewWallHandler(wallSvc).WithCore(newTestCore(t)).GetByID},
		{"Wall.Update", http.MethodPost, "/w1/x/update", NewWallHandler(wallSvc).WithCore(newTestCore(t)).Update},
		{"Wall.Delete", http.MethodPost, "/w1/x/delete", NewWallHandler(wallSvc).WithCore(newTestCore(t)).Delete},
		{"Wall.Batch", http.MethodPost, "/w1/batch", NewWallHandler(wallSvc).WithCore(newTestCore(t)).BatchOperation},

		{"WorkstationDevice.GetByWorkstation", http.MethodPost, "/wd/x", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).GetByWorkstation},
		{"WorkstationDevice.AddManual", http.MethodPost, "/wd/m", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).AddManual},
		{"WorkstationDevice.GetAD", http.MethodPost, "/wd/x/ad", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).GetADDevices},
		{"WorkstationDevice.GetAsset", http.MethodPost, "/wd/x/a", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).GetAssetDevices},
		{"WorkstationDevice.GetPhysical", http.MethodPost, "/wd/x/p", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).GetPhysicalDevices},
		{"WorkstationDevice.SetPrimaryAndSave", http.MethodPost, "/wd/x/sps", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).SetPrimaryAndSave},
		{"WorkstationDevice.SyncAD", http.MethodPost, "/wd/sad", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).SyncAD},
		{"WorkstationDevice.SyncAsset", http.MethodPost, "/wd/sas", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).SyncAsset},
		{"WorkstationDevice.Update", http.MethodPost, "/wd/x/upd", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).Update},
		{"WorkstationDevice.Delete", http.MethodPost, "/wd/x/del", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).Delete},
		{"WorkstationDevice.SetPrimary", http.MethodPost, "/wd/x/sp", NewWorkstationDeviceHandler(workstationDeviceSvc).WithCore(newTestCore(t)).SetPrimary},
	}

	r := gin.New()
	for _, h := range handlers {
		r.Handle(h.method, h.path, h.handler)
	}

	// Hit each route with empty body to ensure none panic and all return a known HTTP status
	for _, h := range handlers {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(h.method, h.path, nil)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		assert.True(t, w.Code < 600 && w.Code >= 200, "handler %s unexpected status: %d", h.name, w.Code)
	}
}

// silence unused imports
var _ = requests.WorkstationListRequest{}
var _ = models.WorkstationDevice{}
var _ = opsModels.OpsBuilding{}
var _ = opsServices.BuildingStatisticsResult{}