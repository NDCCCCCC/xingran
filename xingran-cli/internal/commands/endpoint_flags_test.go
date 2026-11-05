package commands

// endpointFlags 覆盖率补强（M8：upload/download 命令的 flag 规格）。

import "testing"

func TestEndpointFlagsUpload(t *testing.T) {
	got := endpointFlags(EndpointMeta{Kind: KindUpload})
	if len(got) != 1 || got[0].Name != "file" || got[0].Type != "string" || !got[0].Required {
		t.Fatalf("upload flag 规格不正确: %+v", got)
	}
}

func TestEndpointFlagsDownload(t *testing.T) {
	got := endpointFlags(EndpointMeta{Kind: KindDownload})
	if len(got) != 1 || got[0].Name != "out" || got[0].Required {
		t.Fatalf("download flag 规格不正确: %+v", got)
	}
}

func TestEndpointFlagsJSONFallback(t *testing.T) {
	m := EndpointMeta{
		Flags: []FlagSpec{
			{Name: "username", Type: "string", Required: true},
			{Name: "pageSize", Type: "int"},
		},
	}
	got := endpointFlags(m)
	if len(got) != 2 {
		t.Fatalf("JSON 端点应返回全部业务 flag，实际 %d: %+v", len(got), got)
	}
	if got[0].Name != "username" {
		t.Fatalf("必填应排在前: %+v", got)
	}
}
