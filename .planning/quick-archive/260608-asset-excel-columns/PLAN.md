# 调整资产管理 Excel 导入配置以匹配用户提供的 43 列模板格式

## 背景
用户提供的 Excel 文件 `设备信息查询_20260604135253067.xls` 包含 **43 列**，需要调整 `internal/services/operations/excel_config.go` 中 Asset 配置的表头名称以匹配该格式。

## 用户 Excel 文件的 43 列

| 序号 | Excel 表头 | 数据库字段 | 现有配置 Header |
|------|-----------|-----------|----------------|
| 0 | 设备序列号 | devicesn | 设备序列号(DEVICESN) |
| 1 | 设备中类 | device_category_second_name | 中类(DEVICE_CATEGORY_SECOND_NAME) |
| 2 | 设备类型 | device_type_name | 类型(DEVICE_TYPE_NAME) |
| 3 | 设备型号 | device_model_name | 型号(DEVICE_MODEL_NAME) |
| 4 | 设备渠道 | qudao_name | 设备渠道(QUDAO_NAME) |
| 5 | 设备属性 | attribute_value | 设备属性(ATTRIBUTE_VALUE) |
| 6 | 是否固定资产 | device_basic_type_name | 是否固定资产(DEVICE_BASIC_TYPE_NAME) |
| 7 | 固定资产编号 | fixassetno | 固定资产编号(FIXASSETNO) |
| 8 | 入库日期 | storage_datetime | 入库日期(STORAGE_DATETIME) |
| 9 | 发放日期 | use_date | 发放日期(USE_DATE) |
| 10 | 接收日期 | drawing_date | 接收日期(DRAWING_DATE) |
| 11 | 状态 | usestatus_label | 状态(USESTATUS_LABEL) |
| 12 | 是否拟报废 | nbf_status | 是否拟报废(NBF_STATUS) |
| 13 | 归属机构 | sign_orgno_name | 归属机构(SIGN_ORGNO_NAME) |
| 14 | 使用机构 | orgno_name | 使用机构(ORGNO_NAME) |
| 15 | 部门 | useful_dept_name | 部门名称(USEFUL_DEPT_NAME) |
| 16 | 受益部门 | deptname | 受益部门(DEPTNAME) |
| 17 | 领取人 | deviceuser_name | 领取人(DEVICEUSER_NAME) |
| 18 | 责任人 | nowuser_name | 责任人(NOWUSER_NAME) |
| 19 | 使用人 | outer_user | 使用人(OUTER_USER) |
| 20 | 责任人岗位 | nowuser_job_name | 责任人岗位(NOWUSER_JOB_NAME) |
| 21 | 用途 | using_type_name | 用途(USING_TYPE_NAME) |
| 22 | 子用途 | **需要新增** | - |
| 23 | 库房 | storeroom_name | 存放地址(STOREROOM_NAME) |
| 24 | 存放地址 | - (需要新增) | - |
| 25 | 合同号 | contractno | 合同号(CONTRACTNO) |
| 26 | 资产标签打印状态 | print_flag_name | 打印状态(PRINT_FLAG_NAME) |
| 27 | 异常标识 | error_flag_name | 异常标识(ERROR_FLAG_NAME) |
| 28 | 新设备标识 | new_flag_label | 新设备标识(NEW_FLAG_LABEL) |
| 29 | 备注 | remark | 备注(REMARK) |
| 30 | 申请标准 | is_no_standard_name | 申请标准名称(IS_NO_STANDARD_NAME) |
| 31 | IP地址 | machine_ip | 加域IP(MACHINE_IP) |
| 32 | 加域标识 | machine_bs | 加域标识(MACHINE_BS) |
| 33 | 加域ip地址 | - (可能同machine_ip) | - |
| 34 | 有线MAC地址 | mac1 | 有线MAC(MAC1) |
| 35 | 无线MAC地址 | mac2 | 无线MAC(MAC2) |
| 36 | 最后上线时间 | machine_uptime | 最后上线时间(MACHINE_UPTIME) |
| 37 | 最后上线账号 | machine_user_id | 最后上线账号(MACHINE_USER_ID) |
| 38 | APP扫码账号 | user_name | APP扫码账号(USER_NAME) |
| 39 | APP扫码时间 | last_update_date | APP扫码时间(LAST_UPDATE_DATE) |
| 40 | AAP扫码地理位置 | scan_site | AAP扫码地理位置(SCAN_SITE) |
| 41 | 最近一次盘点时间 | **需要新增** | - |
| 42 | 盘点结果 | **需要新增** | - |

## 任务

### 1. 检查并添加缺失的数据库字段
以下字段在用户 Excel 中存在，但 Asset 模型中可能缺失：
- **子用途** (sub_using_type_name)
- **存放地址** (storage_address) - 与库房(storeroom)区分
- **最近一次盘点时间** (last_inventory_date)
- **盘点结果** (inventory_result)

### 2. 更新 Excel 配置
修改 `internal/services/operations/excel_config.go` 中 Asset 配置：
- 更新所有表头名称，移除英文后缀，使用纯中文
- 添加新字段的列定义
- 保持字段映射关系正确

### 3. 验证
- 运行 `go build ./...` 确保编译通过
- 如果有测试，运行相关测试

## 文件变更
- `internal/models/asset.go` - 添加新字段（如需要）
- `internal/services/operations/excel_config.go` - 更新 Asset 配置
