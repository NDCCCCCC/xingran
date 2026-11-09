#!/usr/bin/env python3
"""Fix VDI VM sync IP address update logic"""

file_path = "D:/CODE/ClaudeCode/xingran-go-backend/internal/services/vdi/vm_service_impl.go"

with open(file_path, 'r', encoding='utf-8') as f:
    lines = f.readlines()

# 找到 "power_state": 那一行（大约在331行），在 "mac_address": 之前插入智能IP逻辑
for i, line in enumerate(lines):
    # 在 updates map 中找到 "mac_address" 这一行
    if '"mac_address":   resource.MAC,' in line and i > 300 and i < 360:
        # 检查前面是否有 "ip_address": ipAddress,
        if i > 0 and '"ip_address"' in lines[i-1]:
            # 删除 ip_address 那一行
            del lines[i-1]
            # 在 "mac_address" 之前插入注释
            lines.insert(i-1, '\t\t\t\t// ip_address will be set by smart logic below\n')
            print(f"Removed ip_address from map at line {i}")
            break

# 现在在 updates["bound_user_name"] 之后插入智能IP更新逻辑
for i, line in enumerate(lines):
    if 'updates["bound_user_name"] = &resource.ApplyUser' in line and i > 300 and i < 370:
        # 找到下一个右大括号
        indent = '\t\t\t'
        new_code = f'''\n
{indent}// Smart IP update strategy
{indent}// For DHCP VMs, if new IP is empty, keep existing IP (last known DHCP address)
{indent}// For static IP VMs, or when new IP is not empty, use new IP
{indent}if ipAddress == "" || ipAddress == "-" {{
{indent}\t// New IP is empty, check if we should keep old IP
{indent}\tif vm.IPType == "dhcp" && vm.IPAddress != "" && vm.IPAddress != "-" {{
{indent}\t\t// DHCP VM with history IP, keep old IP
{indent}\t\tupdates["ip_address"] = vm.IPAddress
{indent}\t\tfmt.Printf("[VDI SYNC] VM %s: DHCP mode new IP empty, keeping history IP %s\\n", resource.VMName, vm.IPAddress)
{indent}\t}} else {{
{indent}\t\t// Static IP or no history IP, use new value (may be empty)
{indent}\t\tupdates["ip_address"] = ipAddress
{indent}\t}}
{indent}}} else {{
{indent}\t// New IP not empty, use new IP
{indent}\tupdates["ip_address"] = ipAddress
{indent}}}
'''
        lines.insert(i+1, new_code)
        print(f"Inserted smart IP logic after line {i+1}")
        break

with open(file_path, 'w', encoding='utf-8') as f:
    f.writelines(lines)

print("File modification completed")
