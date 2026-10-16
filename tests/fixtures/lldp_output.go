package fixtures

const (
	// HuaweiLLDPBrief 华为设备LLDP邻居简要信息输出
	// 格式: 本地接口  邻居ID  邻居端口  保持时间  状态  使能状态
	HuaweiLLDPBrief = `  GigabitEthernet0/0/1  3815.sep12.eth0   60P-28   120    Tx/Rx   Enabled
  GigabitEthernet0/0/2  3815.sep13.eth0   60P-29   120    Tx/Rx   Enabled
  GigabitEthernet0/0/3  --               --       --     --      Disabled
  GigabitEthernet0/0/4  3815.sep14.eth0   60P-30   120    Tx/Rx   Enabled
  GigabitEthernet0/0/5  3815.sep15.eth0   60P-31   120    Tx/Rx   Enabled`

	// HuaweiLLDPEmpty 华为设备无LLDP邻居信息
	HuaweiLLDPEmpty = `No LLDP neighbor information exists`

	// HuaweiLLDPDisabled 华为设备LLDP未启用
	HuaweiLLDPDisabled = `LLDP is not enabled`

	// HuaweiLLDPDetailed 华为设备LLDP详细信息输出
	HuaweiLLDPDetailed = `GigabitEthernet0/0/1 has 1 neighbor(s):
  Neighbor index           : 1
    Chassis type           : MAC address
    Chassis ID             : 3815.sep12.eth0
    Port index             : 1
    Port type              : Interface name
    Port ID                : 60P-28
    Time to live           : 120
    System name            : 60P-28
    System description     : Ruijie Internal Switch
    System capabilities    : Bridge Router
    Enabled capabilities   : Bridge Router
    Management address     : 192.168.1.1`

	// RuijieLLDPNeighbors 锐捷设备LLDP邻居信息输出
	// 格式: 本地接口  Chassis ID  端口ID  系统名称
	RuijieLLDPNeighbors = `Local Interface    Chassis ID          Port ID          System Name
-----------------  ------------------  ---------------  ----------
Gi0/1              3815.sep12.eth0     eth0             60P-28
Gi0/2              3815.sep13.eth0     eth0             60P-29
Gi0/3              --                  --               --
Gi0/4              3815.sep14.eth0     eth0             60P-30
Gi0/5              3815.sep15.eth0     eth0             60P-31`

	// RuijieLLDPEmpty 锐捷设备无LLDP邻居
	RuijieLLDPEmpty = `No LLDP neighbors found`

	// RuijieLLDPDisabled 锐捷设备LLDP未启用
	RuijieLLDPDisabled = `% LLDP is not enabled`

	// H3CLLDPNeighbors H3C设备LLDP邻居信息（格式与华为相同）
	H3CLLDPNeighbors = `  GigabitEthernet1/0/1  3815.sep12.eth0   eth0     120    Tx/Rx   Enabled
  GigabitEthernet1/0/2  3815.sep13.eth0   eth0     120    Tx/Rx   Enabled
  GigabitEthernet1/0/3  --               --       --     --      Disabled`

	// MaipuLLDPNeighbors 迈普设备LLDP邻居信息（格式与锐捷相同）
	MaipuLLDPNeighbors = `Local Interface    Chassis ID          Port ID          System Name
-----------------  ------------------  ---------------  ----------
Eth1/1             3815.sep12.eth0     eth0             60P-28
Eth1/2             3815.sep13.eth0     eth0             60P-29
Eth1/3             --                  --               --`

	// MalformedLLDPOutput 格式错误的LLDP输出
	MalformedLLDPOutput = `Invalid LLDP output format
cannot parse this properly!!!`

	// EmptyLLDPOutput 空LLDP输出
	EmptyLLDPOutput = ``

	// LLDPWithSpecialCharacters 包含特殊字符的LLDP输出
	LLDPWithSpecialCharacters = `  GigabitEthernet0/0/1  3815.sep12.eth0   eth0/0/1   120    Tx/Rx   Enabled
  GigabitEthernet0/0/2  router-1.core   Port-Channel1   120    Tx/Rx   Enabled`
)
