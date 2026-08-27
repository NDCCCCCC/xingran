// Phase 79-06 (TAIL-01) — device_discovery_service.go Tier-1 coverage tests.
//
// Tier-1 scope per 79-06-PLAN.md Task 2: the pure IP math block
// (calculateIPCount / generateIPList / ipLessEqual / incrementIP ≈ 80 stmts),
// the isAlive TCP probe, the CRUD/scan-task sqlite chain and the pure masking
// helper. SNMP segments (snmpProbe / discoverBySNMP / ProbeSingleDevice) are
// Task 7 (D-79-05 fake).
//
// Environment discipline (78-04 / research §8 precedent):
//   - No ICMP anywhere — isAlive is a TCP-connect probe, so tests bind a real
//     TCP listener on 127.0.0.1 for the alive branch. isAlive's port list is
//     fixed (80/443/22/23/161/8080), so the helper binds the first free port
//     from that exact list and skips when all are taken (never flakes).
//   - The dead-IP case uses 127.0.0.2: the whole 127/8 loopback rejects
//     instantly (RST), so the six dials cost milliseconds instead of the 1s
//     per-port dial timeout a non-routable address would burn.
//   - Naming: TestDdv7906_* + newDdv7906 helper (D-79-06).
package services

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

// newDdv7906 assembles a DeviceDiscoveryService over a fresh sqlite DB with the
// DeviceDiscovery model migrated (newDB7906 supplies the UUID fill callback —
// models.DeviceDiscovery carries no BeforeCreate hook).
func newDdv7906(t *testing.T) (*DeviceDiscoveryService, *gorm.DB) {
	t.Helper()
	db := newDB7906(t, &models.DeviceDiscovery{})
	return &DeviceDiscoveryService{db: db}, db
}

// ddv7906AliveListener binds the first free port from isAlive's fixed probe
// list (80/443/22/23/161/8080) on 127.0.0.1 so the alive branch is
// deterministic. Skips when all six are taken — that is an environment
// property, not a test failure.
func ddv7906AliveListener(t *testing.T) net.Listener {
	t.Helper()
	for _, port := range []int{80, 443, 22, 23, 161, 8080} {
		l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err == nil {
			t.Cleanup(func() { _ = l.Close() })
			return l
		}
	}
	t.Skip("no isAlive probe port (80/443/22/23/161/8080) bindable on 127.0.0.1 — alive branch skipped")
	return nil
}

// -----------------------------------------------------------------------------
// pure IP math
// -----------------------------------------------------------------------------

// TestDdv7906_CalculateIPCount table-drives the range-size math: inclusive
// bounds, single address, /16 scale, invalid input and the negative-count
// quirk for end < start.
func TestDdv7906_CalculateIPCount(t *testing.T) {
	cases := []struct {
		name     string
		startIP  string
		endIP    string
		expected int
		note     string
	}{
		{"slash24", "192.168.1.0", "192.168.1.255", 256, "inclusive /24"},
		{"single_address", "10.0.0.7", "10.0.0.7", 1, "start == end → exactly 1"},
		{"slash16", "172.16.0.0", "172.16.255.255", 65536, "inclusive /16"},
		{"cross_octet", "192.168.0.250", "192.168.1.10", 273,
			"QUIRK-79-06-B (locked, not fixed): the byte difference is computed in " +
				"uint8 arithmetic, so a last octet where end<start WRAPS (10-250 → 16) " +
				"and the range counts 256+16+1 instead of the true 17"},
		{"invalid_start", "not-an-ip", "192.168.1.5", 0, "unparseable start"},
		{"invalid_end", "192.168.1.5", "not-an-ip", 0, "unparseable end"},
		{"ipv6_returns_zero", "::1", "::2", 0, "To4() nil → 0 (IPv4-only math)"},
		{"empty_strings", "", "", 0, "ParseIP(nil-ish) → 0"},
		{"end_before_start", "192.168.0.10", "192.168.0.2", 249,
			"same uint8 wrap: 2-10 → 248, so end<start counts 249 instead of being " +
				"sanitized to 0 — TotalIPs in a mis-keyed range silently overcounts"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := calculateIPCount(tc.startIP, tc.endIP)
			assert.Equal(t, tc.expected, got, tc.note)
		})
	}
}

// TestDdv7906_GenerateIPList table-drives list generation: ordering, the
// third-octet carry, invalid-range skipping, single-address ranges and the
// 65536-entry safety cap.
func TestDdv7906_GenerateIPList(t *testing.T) {
	t.Run("carry_across_segment", func(t *testing.T) {
		got := generateIPList([]models.IPRange{{StartIP: "192.168.0.254", EndIP: "192.168.1.1"}})
		require.Len(t, got, 4)
		assert.Equal(t, []string{"192.168.0.254", "192.168.0.255", "192.168.1.0", "192.168.1.1"}, got,
			"list must walk the third-octet carry in order")
	})

	t.Run("multiple_ranges_concatenate", func(t *testing.T) {
		got := generateIPList([]models.IPRange{
			{StartIP: "10.0.0.1", EndIP: "10.0.0.2"},
			{StartIP: "10.0.1.1", EndIP: "10.0.1.1"},
		})
		require.Len(t, got, 3)
		assert.Equal(t, []string{"10.0.0.1", "10.0.0.2", "10.0.1.1"}, got)
	})

	t.Run("invalid_ranges_skipped", func(t *testing.T) {
		got := generateIPList([]models.IPRange{
			{StartIP: "bogus", EndIP: "10.0.0.2"},
			{StartIP: "10.0.2.1", EndIP: "10.0.2.2"},
			{StartIP: "10.0.3.1", EndIP: "alsobogus"},
		})
		assert.Equal(t, []string{"10.0.2.1", "10.0.2.2"}, got,
			"ranges with unparseable bounds are skipped, valid ranges still emitted")
	})

	t.Run("empty_input", func(t *testing.T) {
		assert.Empty(t, generateIPList(nil))
		assert.Empty(t, generateIPList([]models.IPRange{}))
	})

	t.Run("safety_cap_65536", func(t *testing.T) {
		// /15-scale range (131072 addresses) must be capped at maxCount=65536.
		got := generateIPList([]models.IPRange{{StartIP: "1.0.0.0", EndIP: "1.1.255.255"}})
		require.Len(t, got, 65536, "generation stops at the built-in 65536 cap")
		assert.Equal(t, "1.0.0.0", got[0])
		assert.Equal(t, "1.0.0.255", got[255], "second-octet walk stays in order")
		assert.Equal(t, "1.0.255.255", got[65535], "cap lands exactly at the /16 boundary")
	})
}

// TestDdv7906_IPLessEqual_Increment drives both remaining pure IP functions
// with equality / ordering / multi-octet carry / full-overflow wrap cases.
//
// QUIRK-79-06-C (locked, not fixed): ipLessEqual has no To4() nil guard — an
// IPv6 argument panics on the byte-wise compare, so only IPv4 inputs are fed
// here (the production caller chain only produces To4()-normalized IPs).
func TestDdv7906_IPLessEqual_Increment(t *testing.T) {
	mustIP := func(s string) net.IP {
		ip := net.ParseIP(s)
		require.NotNil(t, ip, "parse %s", s)
		return ip
	}

	t.Run("ipLessEqual", func(t *testing.T) {
		cases := []struct {
			name string
			a, b string
			want bool
		}{
			{"equal", "192.168.1.1", "192.168.1.1", true},
			{"less_last_octet", "192.168.1.1", "192.168.1.2", true},
			{"greater_last_octet", "192.168.1.9", "192.168.1.2", false},
			{"less_second_octet", "10.0.0.255", "10.1.0.0", true},
			{"greater_first_octet", "192.168.0.1", "10.0.0.1", false},
			{"zero_vs_broadcast", "0.0.0.0", "255.255.255.255", true},
			{"broadcast_vs_zero", "255.255.255.255", "0.0.0.0", false},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.want, ipLessEqual(mustIP(tc.a), mustIP(tc.b)))
			})
		}
	})

	t.Run("incrementIP", func(t *testing.T) {
		cases := []struct {
			name string
			in   string
			want string
		}{
			{"plain_increment", "192.168.0.10", "192.168.0.11"},
			{"last_octet_carry", "192.168.0.254", "192.168.0.255"},
			{"cross_octet_carry", "192.168.0.255", "192.168.1.0"},
			{"multi_octet_carry", "192.168.255.255", "192.169.0.0"},
			{"full_overflow_wrap", "255.255.255.255", "0.0.0.0"},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				got := incrementIP(mustIP(tc.in))
				require.NotNil(t, got)
				assert.Equal(t, tc.want, got.String())
			})
		}

		t.Run("nil_returns_nil", func(t *testing.T) {
			assert.Nil(t, incrementIP(nil), "nil input must short-circuit")
		})
		t.Run("ipv6_returns_nil", func(t *testing.T) {
			assert.Nil(t, incrementIP(net.ParseIP("::1")), "To4() nil → nil (nil-guarded)")
		})
	})
}

// -----------------------------------------------------------------------------
// isAlive (TCP probe)
// -----------------------------------------------------------------------------

// TestDdv7906_IsAlive_TCP proves both branches of the TCP-only liveness probe:
// a bound probe port answers alive; a closed loopback address on the same /8
// rejects instantly and reports dead. No ICMP is involved anywhere.
func TestDdv7906_IsAlive_TCP(t *testing.T) {
	t.Run("alive_via_bound_probe_port", func(t *testing.T) {
		l := ddv7906AliveListener(t)
		require.NotNil(t, l)
		require.True(t, isAlive("127.0.0.1"),
			"a listener on one of isAlive's probe ports must report the host alive")
	})

	t.Run("dead_on_refused_loopback", func(t *testing.T) {
		// Some hosts run services on 0.0.0.0 (so EVERY 127.x.y.z answers on a
		// probe port); hunt for a loopback address whose six probe ports all
		// refuse, and skip when none exists — that is an environment property,
		// not a test failure. Candidates refuse instantly (RST), so the sweep
		// costs milliseconds; the dial timeout only bounds a pathological host.
		dead := ""
		for i := 2; i <= 12 && dead == ""; i++ {
			candidate := fmt.Sprintf("127.0.0.%d", i)
			answers := false
			for _, port := range []int{80, 443, 22, 23, 161, 8080} {
				conn, err := net.DialTimeout("tcp", net.JoinHostPort(candidate, strconv.Itoa(port)), 250*time.Millisecond)
				if err == nil {
					_ = conn.Close()
					answers = true
					break
				}
			}
			if !answers {
				dead = candidate
			}
		}
		if dead == "" {
			t.Skip("every 127.0.0.x candidate answers on an isAlive probe port — dead branch skipped")
		}
		require.False(t, isAlive(dead),
			"a loopback address with all six probe ports refusing must report dead")
	})
}

// -----------------------------------------------------------------------------
// pure masking helper
// -----------------------------------------------------------------------------

// TestDdv7906_MaskSensitiveStringDiscovery table-drives the SNMP community
// masker used by snmpProbe / ProbeSingleDevice logging.
func TestDdv7906_MaskSensitiveStringDiscovery(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", "(empty)"},
		{"four_chars_uses_full_mask", "abcd", "*** (4 chars)"},
		{"five_chars_keeps_two_and_two", "abcde", "ab***de (5 chars)"},
		{"public_is_six_chars", "public", "pu***ic (6 chars)"},
		{"long_community", "verylongcommunitystring", "ve***ng (23 chars)"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, maskSensitiveStringDiscovery(tc.input))
		})
	}
}

// -----------------------------------------------------------------------------
// CRUD / 扫描任务主链
// -----------------------------------------------------------------------------

// TestDdv7906_DiscoveryCRUD drives the sqlite scan-task chain:
// CreateDiscoveryTask (with totalIP math) → GetDiscoveryList (pagination +
// sort whitelist) → GetDiscoveryByID → CancelDiscovery → DeleteDiscovery →
// BatchDeleteDiscoveries, plus ImportDiscoveredDevices and GetDiscoveryResults
// (stub-backed, behavior locked).
func TestDdv7906_DiscoveryCRUD(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDdv7906(t)

	var taskIDs []string
	t.Run("create_computes_total_ips", func(t *testing.T) {
		id, err := svc.CreateDiscoveryTask(ctx, &DiscoveryRequest{
			TaskName:      "ddv-crud-1",
			DiscoveryType: models.DiscoveryTypeScan,
			IPRanges: []models.IPRange{
				{StartIP: "192.168.10.0", EndIP: "192.168.10.3"},
				{StartIP: "192.168.20.0", EndIP: "192.168.20.1"},
			},
			SNMPPort:  161,
			CreatedBy: "ddv7906",
		})
		require.NoError(t, err)
		require.NotEmpty(t, id, "created task must get a PK (UUID fill callback)")
		taskIDs = append(taskIDs, id)

		got, err := svc.GetDiscoveryByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "ddv-crud-1", got.TaskName)
		assert.Equal(t, models.DiscoveryStatusPending, got.Status, "new task starts pending")
		assert.Equal(t, 6, got.TotalIPs, "4 + 2 across both ranges")
		assert.Equal(t, models.DiscoveryTypeScan, got.DiscoveryType)
	})

	t.Run("create_more_and_list", func(t *testing.T) {
		for _, name := range []string{"ddv-crud-2", "ddv-crud-3", "ddv-crud-0"} {
			id, err := svc.CreateDiscoveryTask(ctx, &DiscoveryRequest{
				TaskName:      name,
				DiscoveryType: models.DiscoveryTypeScan,
				IPRanges:      []models.IPRange{{StartIP: "10.79.0.1", EndIP: "10.79.0.1"}},
			})
			require.NoError(t, err)
			taskIDs = append(taskIDs, id)
		}

		tasks, total, err := svc.GetDiscoveryList(ctx, 1, 3, "", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(4), total, "no filter → all four tasks")
		require.Len(t, tasks, 3, "pageSize caps the page")

		// Sort whitelist: taskName ascending is fully deterministic.
		tasks, total, err = svc.GetDiscoveryList(ctx, 1, 10, "taskName", boolPtr7906(true))
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		require.Len(t, tasks, 4)
		var names []string
		for _, tk := range tasks {
			names = append(names, tk.TaskName)
		}
		assert.Equal(t, []string{"ddv-crud-0", "ddv-crud-1", "ddv-crud-2", "ddv-crud-3"}, names,
			"whitelisted sort column must order ascending")

		// Non-whitelisted column falls back without error (ApplySort warn path).
		_, _, err = svc.GetDiscoveryList(ctx, 1, 10, "1; DROP TABLE sys_device_discovery", nil)
		require.NoError(t, err, "unknown sort column must not error (whitelist fallback)")

		// Second page.
		page2, _, err := svc.GetDiscoveryList(ctx, 2, 3, "", nil)
		require.NoError(t, err)
		require.Len(t, page2, 1, "4 rows with pageSize 3 → second page has 1")
	})

	t.Run("get_by_id_miss", func(t *testing.T) {
		got, err := svc.GetDiscoveryByID(ctx, "no-such-task")
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "发现任务不存在")
	})

	t.Run("cancel_pending_then_reject_cancelled", func(t *testing.T) {
		require.NoError(t, svc.CancelDiscovery(ctx, taskIDs[0]), "pending task is cancellable")
		got, err := svc.GetDiscoveryByID(ctx, taskIDs[0])
		require.NoError(t, err)
		assert.Equal(t, models.DiscoveryStatusCancelled, got.Status)

		err = svc.CancelDiscovery(ctx, taskIDs[0])
		require.Error(t, err, "cancelled task is neither pending nor running")
		assert.Contains(t, err.Error(), "只能取消待执行或执行中的任务")
	})

	t.Run("cancel_missing", func(t *testing.T) {
		err := svc.CancelDiscovery(ctx, "no-such-task")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "发现任务不存在")
	})

	t.Run("delete_rejects_running_and_deletes_cancelled", func(t *testing.T) {
		// A running task cannot be deleted.
		running, err := svc.CreateDiscoveryTask(ctx, &DiscoveryRequest{
			TaskName:      "ddv-running",
			DiscoveryType: models.DiscoveryTypeScan,
			IPRanges:      []models.IPRange{{StartIP: "10.79.1.1", EndIP: "10.79.1.1"}},
		})
		require.NoError(t, err)
		row, err := svc.GetDiscoveryByID(ctx, running)
		require.NoError(t, err)
		row.Status = models.DiscoveryStatusRunning
		require.NoError(t, svc.db.Save(row).Error, "flip the row to running directly")
		err = svc.DeleteDiscovery(ctx, running)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "无法删除执行中的任务")

		// A cancelled task deletes cleanly and disappears.
		require.NoError(t, svc.DeleteDiscovery(ctx, taskIDs[0]))
		_, err = svc.GetDiscoveryByID(ctx, taskIDs[0])
		require.Error(t, err, "deleted task must be gone (soft delete filters it out)")

		// Missing task → explicit error.
		err = svc.DeleteDiscovery(ctx, "no-such-task")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "发现任务不存在")
	})

	t.Run("batch_delete_continues_on_error", func(t *testing.T) {
		// One missing ID must not abort the rest of the batch (continue semantics).
		require.NoError(t, svc.BatchDeleteDiscoveries(ctx, []string{"no-such-a", taskIDs[1], "no-such-b"}))
		_, err := svc.GetDiscoveryByID(ctx, taskIDs[1])
		require.Error(t, err, "the present task must still be deleted")
	})

	t.Run("import_and_results_stub", func(t *testing.T) {
		n, err := svc.ImportDiscoveredDevices(ctx, taskIDs[2], []string{"d1", "d2", "d3"}, "ddv7906")
		require.NoError(t, err)
		assert.Equal(t, 3, n, "stub implementation echoes the requested device count")

		_, err = svc.ImportDiscoveredDevices(ctx, "no-such-task", []string{"d1"}, "ddv7906")
		require.Error(t, err)

		devices, err := svc.GetDiscoveryResults(ctx, taskIDs[2])
		require.NoError(t, err)
		assert.Empty(t, devices, "result store is a stub — always empty (locked)")
	})
}

// -----------------------------------------------------------------------------
// ExecuteDiscovery(scan) — 端到端扫描路径(无 SNMP)
// -----------------------------------------------------------------------------

// TestDdv7906_ExecuteDiscovery_ScanType runs the scan discovery end-to-end
// against 127.0.0.1 with a real bound probe port (Task 2 Tier-1 scope; the
// SNMP type is Task 7 with the D-79-05 fake). Status transitions and the
// discovered-device list are asserted off the returned result and re-read row.
func TestDdv7906_ExecuteDiscovery_ScanType(t *testing.T) {
	ctx := context.Background()
	l := ddv7906AliveListener(t)
	require.NotNil(t, l, "alive branch needs a bound probe port")

	svc, _ := newDdv7906(t)
	id, err := svc.CreateDiscoveryTask(ctx, &DiscoveryRequest{
		TaskName:      "ddv-scan-e2e",
		DiscoveryType: models.DiscoveryTypeScan,
		IPRanges:      []models.IPRange{{StartIP: "127.0.0.1", EndIP: "127.0.0.1"}},
	})
	require.NoError(t, err)

	result, err := svc.ExecuteDiscovery(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, id, result.DiscoveryID)
	assert.Equal(t, 1, result.TotalIPs)
	assert.Equal(t, len(result.DiscoveredDevices), result.DiscoveredCount)
	assert.Equal(t, models.DiscoveryStatusSuccess, result.Status, "scan path never errors → success")
	for _, dev := range result.DiscoveredDevices {
		assert.Equal(t, "127.0.0.1", dev.IPAddress)
		assert.True(t, dev.IsAlive, "only alive hosts are appended by discoverByScan")
	}

	// The task row must carry the terminal state and timestamps.
	row, err := svc.GetDiscoveryByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, models.DiscoveryStatusSuccess, row.Status)
	require.NotNil(t, row.StartedAt)
	require.NotNil(t, row.CompletedAt)
	assert.Equal(t, len(result.DiscoveredDevices), row.DiscoveredCount)
}

// TestDdv7906_ExecuteDiscovery_MissingTask covers the unknown-ID error branch
// (no listener needed — the DB lookup fails before any probing).
func TestDdv7906_ExecuteDiscovery_MissingTask(t *testing.T) {
	svc, _ := newDdv7906(t)
	result, err := svc.ExecuteDiscovery(context.Background(), "no-such-task")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "发现任务不存在")
}

// boolPtr7906 is a tiny helper for sort-direction arguments.
func boolPtr7906(v bool) *bool { return &v }
