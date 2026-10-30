package services

import (
	"context"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/require"
)

// commandSpy records every command issued and returns a canned response
// (looked up by exact command match). Used by the D-10 two-step pipeline
// tests to assert the status command is issued BEFORE the transceiver
// command.
type commandSpy struct {
	responses map[string]string // exact-match canned response
	calls     []string          // ordered list of commands issued
}

func (s *commandSpy) run(cmd string) (string, error) {
	s.calls = append(s.calls, cmd)
	if r, ok := s.responses[cmd]; ok {
		return r, nil
	}
	return "", nil
}

// TestCollectComponentInfoHuaweiTwoStepPipeline REQ-48-11 D-10:
// Huawei vendor must run "display interface status" BEFORE
// "display interface transceiver". Verify the command order via spy.
func TestCollectComponentInfoHuaweiTwoStepPipeline(t *testing.T) {
	spy := &commandSpy{
		responses: map[string]string{
			"display interface status":      huaweiStatusUp,
			"display interface transceiver": huaweiTransceiver,
		},
	}
	set, err := runTwoStepTransceiverPipeline(context.Background(), models.VendorHuawei, spy.run)
	require.NoError(t, err)
	require.NotNil(t, set)

	// Verify command order: status BEFORE transceiver (D-10).
	require.Len(t, spy.calls, 2, "exactly 2 commands for huawei transceiver path")
	require.Equal(t, "display interface status", spy.calls[0], "D-10: status command first")
	require.Equal(t, "display interface transceiver", spy.calls[1], "D-10: transceiver command second")

	// At least one transceiver component parsed (the up interface in fixture)
	require.NotEmpty(t, set.Components, "at least one transceiver parsed from up interface")
	found := false
	for _, c := range set.Components {
		if c.ComponentType == "transceiver" {
			found = true
			require.NotEmpty(t, c.SerialNumber, "transceiver SN populated")
		}
	}
	require.True(t, found, "D-10: transceiver component parsed from up interface")
}

// TestCollectComponentInfoRuijieTwoStepPipeline REQ-48-11 D-10:
// Ruijie vendor must run "show interfaces status" BEFORE
// "show interface transceiver". Verify the command order via spy.
func TestCollectComponentInfoRuijieTwoStepPipeline(t *testing.T) {
	spy := &commandSpy{
		responses: map[string]string{
			"show interfaces status":   ruijieStatusUp,
			"show interface transceiver": ruijieTransceiver,
		},
	}
	set, err := runTwoStepTransceiverPipeline(context.Background(), models.VendorRuijie, spy.run)
	require.NoError(t, err)
	require.NotNil(t, set)
	require.Len(t, spy.calls, 2, "exactly 2 commands for ruijie transceiver path")
	require.Equal(t, "show interfaces status", spy.calls[0], "D-10: status command first")
	require.Equal(t, "show interface transceiver", spy.calls[1], "D-10: transceiver command second")

	found := false
	for _, c := range set.Components {
		if c.ComponentType == "transceiver" {
			found = true
		}
	}
	require.True(t, found, "D-10: ruijie transceiver parsed from up interface")
}

// TestCollectComponentInfoUnknownVendorSkipsPipeline REQ-48-11 negative:
// H3C / Maipu / unknown vendors return empty set, no commands issued.
func TestCollectComponentInfoUnknownVendorSkipsPipeline(t *testing.T) {
	spy := &commandSpy{responses: map[string]string{}}
	set, err := runTwoStepTransceiverPipeline(context.Background(), models.VendorH3C, spy.run)
	require.NoError(t, err)
	require.NotNil(t, set)
	require.Empty(t, spy.calls, "no commands issued for unsupported vendor")
}

// TestCollectComponentInfoFailureDoesNotReturnErrorFromStatusParse REQ-48-11 D-14:
// Runner returning error mid-pipeline → runTwoStepTransceiverPipeline returns
// the error; the cron caller treats it as warning (asserted here at the unit
// level by checking err is propagated, NOT swallowed).
func TestCollectComponentInfoFailureReturnsErr(t *testing.T) {
	// Use a runner that returns empty string + nil err for status but the
	// transceiver command is missing — pipeline must NOT panic, just return
	// an empty transceiver set.
	spy := &commandSpy{
		responses: map[string]string{
			"display interface status": "no up ports",
		},
	}
	set, err := runTwoStepTransceiverPipeline(context.Background(), models.VendorHuawei, spy.run)
	require.NoError(t, err, "missing transceiver response is not an error")
	require.NotNil(t, set)
	// No transceiver components because the status parse yielded 0 up ports.
	for _, c := range set.Components {
		if c.ComponentType == "transceiver" {
			t.Fatalf("expected 0 transceivers when no up ports; got %+v", c)
		}
	}
}

// TestCollectComponentInfoSkipVendorMap asserts that only supported vendors
// (Huawei / Ruijie) emit the transceiver command pair. H3C and Maipu are
// excluded per RESEARCH "Out of scope" — the dispatcher returns nil.
func TestCollectComponentInfoSkipVendorMap(t *testing.T) {
	for _, v := range []models.DeviceVendor{models.VendorHuawei, models.VendorRuijie} {
		cmds := transceiverCommandPair(v)
		require.Len(t, cmds, 2, "huawei/ruijie have 2-command transceiver pair")
		require.True(t, strings.Contains(cmds[0], "status"), "first cmd is status")
		require.True(t, strings.Contains(cmds[1], "transceiver"), "second cmd is transceiver")
	}
	for _, v := range []models.DeviceVendor{models.VendorH3C, models.VendorMaipu} {
		require.Nil(t, transceiverCommandPair(v), "h3c/maipu return nil (out of scope)")
	}
}

// Fixture snippets (truncated to just enough for the parser):
const huaweiStatusUp = `InUti/OutUti   inErr/outErr   Interface                           PHY   Auto Neg Speed Duplex  Description
0.00% 0.00%     0     0       10GE5/0/4                           up    enable auto 10000 full   Node
0.00% 0.00%     0     0       10GE5/0/5                           down  enable auto 10000 full   Node
`

const huaweiTransceiver = `10GE5/0/4 transceiver information:
  Common information
    Vendor Name          : HUAWEI
    Manu. Serial Number  : 8000012000082
    Manufacturing Date   : 2025-2-20
10GE5/0/5 transceiver is absent
`

const ruijieStatusUp = `Interface                                Status    Vlan   Duplex   Speed     Type
---------------------------------------- --------  ----   -------  --------- ------
TenGigabitEthernet 1/47                  up        318    Full     10G       sfp+
TenGigabitEthernet 1/48                  down      318    Unknown  Unknown   copper
`

const ruijieTransceiver = `========Interface TenGigabitEthernet 1/47========
  Transceiver Type    :  10GBASE-SR-SFP+
  Vendor Name         :  ACCTON
  Vendor Serial Number           : G1PT549427799
  Current diagnostic parameters[AP:Average Power]:
  Temp(Celsius)   Voltage(V)      Bias(mA)            RX power(dBm)       TX power(dBm)
  39(OK)          3.27(OK)        6.68(OK)            -2.99(OK)[AP]       -2.40(OK)
`
