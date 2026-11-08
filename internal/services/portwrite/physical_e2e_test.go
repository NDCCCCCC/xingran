package portwrite

// FIX-04（v1.53 Phase 138）：物理参数 e2e —— 复用 Phase 54 FileTransport 基建，
// 证明 RenderCommand → writeSinglePort → scrapligo SendConfigs → parseConfigError
// 全链路对物理参数命令可用（零 mock 短路）。

import (
	"context"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// TestE2E_Physical138_SetSpeed_Huawei_Success 华为 speed 1000 全链路 happy path。
//
// fixture 复用 huawei_set_speed_success.fixture（interface GE0/0/1 → speed 1000 → quit）。
// 验证：Status=succeeded、CommandSent 含 "speed 1000"、Extra 回显 speed=1000。
func TestE2E_Physical138_SetSpeed_Huawei_Success(t *testing.T) {
	resetCollectNotif()
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-phy-e2e", "device-1", "GE0/0/1", "up", false, "")

	exec := &fileTransportExecutor{
		fixtureProvider: singleFixture(e2eFixturePath(t, "huawei_set_speed_success.fixture")),
	}
	mockColl := new(mockCollectionSvc)
	svc := newTestService(exec, mockColl, db)

	result, err := svc.PhysicalWrite(context.Background(), "port-phy-e2e",
		portcollection.ActionSetSpeed, PhysicalParams{Speed: 1000}, "e2e-op-138")
	if err != nil {
		t.Fatalf("PhysicalWrite returned err: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Status != "succeeded" {
		t.Fatalf("status = %q, want succeeded; result=%+v", result.Status, result)
	}
	if !contains(result.CommandSent, "speed 1000") {
		t.Fatalf("CommandSent %q missing 'speed 1000'", result.CommandSent)
	}
	if result.Action != portcollection.ActionSetSpeed {
		t.Fatalf("Action = %q, want %q", result.Action, portcollection.ActionSetSpeed)
	}
	if got := result.Extra["speed"]; got != 1000 {
		t.Fatalf("Extra[speed] = %v, want 1000（审计 after_value 载体）", got)
	}
}

// TestE2E_Physical138_SetSpeed_InvalidSkipsSSH 非法速率在 ExecuteCustom 之前被拒。
func TestE2E_Physical138_SetSpeed_InvalidSkipsSSH(t *testing.T) {
	db := newTestDB(t)
	seedPortAndDevice(t, db, "port-phy-e2e2", "device-1", "GE0/0/1", "up", false, "")

	// fixtureProvider 若被调用即 panic，守护「零 SSH 流量」
	exec := &fileTransportExecutor{
		fixtureProvider: func(_ int) string {
			t.Fatal("非法参数不应触发 ExecuteCustom / FileTransport")
			return ""
		},
	}
	svc := newTestService(exec, new(mockCollectionSvc), db)

	_, err := svc.PhysicalWrite(context.Background(), "port-phy-e2e2",
		portcollection.ActionSetSpeed, PhysicalParams{Speed: 999}, "e2e-op")
	if err == nil {
		t.Fatal("非法速率应返回 sentinel error")
	}
}
