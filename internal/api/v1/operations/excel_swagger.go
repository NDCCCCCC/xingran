package operations

import "github.com/xingran-next/xingran-go-backend/pkg/response"

// excel_swagger.go —— Excel 导入导出端点的 Swagger 文档占位
//
// 背景：本包的 Excel import/export/template 端点由 SetupExcelRouter /
// SetupExcelImportRouter / SetupExcelExportRouter / SetupExcelReadRouter 通过
// 闭包（importData / exportData / downloadTemplate）动态生成，swag 无法给闭包
// 直接挂 @Router 注解；因此本文件用一组「文档占位函数」承载 @Router 等注解，
// 供 swag init 扫描生成 OpenAPI 文档。
//
// 约束（task-12：excel 导入导出 25 端点 swag 注解占位）：
//   - 这些函数只承载注释，函数体为空，不注册任何路由、不含任何业务逻辑；
//   - 不动其他目录、不修改 excel_handler.go 里的真实处理逻辑；
//   - 路径与 internal/api/router.go 中 8 类运维实体实际注册的路径一一对应；
//   - 8 类实体 × 3 动作（import / export / template）= 24，
//     加上工位部门映射表模板 dept-mapping-template = 25 个端点。
//
// @BasePath 为 /api/v1（cmd/main.go），故下方 @Router 均省略该前缀。

// swag 类型解析引用：swag 解析注解中的 response.Response 需要本文件 import response 包。
var _ = response.Response{}

// ============================================================================
// 楼宇管理 /ops/building
// ============================================================================

// swagExcelBuildingImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入楼宇
// @Description 上传 .xlsx 文件批量导入楼宇数据
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building/import [post]
func swagExcelBuildingImport() {}

// swagExcelBuildingExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出楼宇
// @Description 按筛选条件导出楼宇数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/building/export [post]
func swagExcelBuildingExport() {}

// swagExcelBuildingTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载楼宇导入模板
// @Description 下载楼宇 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/building/template [get]
func swagExcelBuildingTemplate() {}

// ============================================================================
// 楼层管理 /ops/floor
// ============================================================================

// swagExcelFloorImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入楼层
// @Description 上传 .xlsx 文件批量导入楼层数据
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor/import [post]
func swagExcelFloorImport() {}

// swagExcelFloorExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出楼层
// @Description 按筛选条件导出楼层数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/floor/export [post]
func swagExcelFloorExport() {}

// swagExcelFloorTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载楼层导入模板
// @Description 下载楼层 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/floor/template [get]
func swagExcelFloorTemplate() {}

// ============================================================================
// 工位管理 /ops/workstation
// ============================================================================

// swagExcelWorkstationImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入工位
// @Description 上传 .xlsx 文件批量导入工位数据，支持按部门名称/用户名建立关联
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation/import [post]
func swagExcelWorkstationImport() {}

// swagExcelWorkstationExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出工位
// @Description 按筛选条件导出工位数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/workstation/export [post]
func swagExcelWorkstationExport() {}

// swagExcelWorkstationTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载工位导入模板
// @Description 下载工位 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/workstation/template [get]
func swagExcelWorkstationTemplate() {}

// ============================================================================
// 资产管理 /ops/asset
// ============================================================================

// swagExcelAssetImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入资产
// @Description 上传 .xlsx 文件批量导入资产数据
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/asset/import [post]
func swagExcelAssetImport() {}

// swagExcelAssetExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出资产
// @Description 按筛选条件导出资产数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/asset/export [post]
func swagExcelAssetExport() {}

// swagExcelAssetTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载资产导入模板
// @Description 下载资产 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/asset/template [get]
func swagExcelAssetTemplate() {}

// ============================================================================
// 信息点管理 /ops/infoPoint
// ============================================================================

// swagExcelInfoPointImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入信息点
// @Description 上传 .xlsx 文件批量导入信息点数据
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/import [post]
func swagExcelInfoPointImport() {}

// swagExcelInfoPointExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出信息点
// @Description 按筛选条件导出信息点数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/export [post]
func swagExcelInfoPointExport() {}

// swagExcelInfoPointTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载信息点导入模板
// @Description 下载信息点 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/infoPoint/template [get]
func swagExcelInfoPointTemplate() {}

// ============================================================================
// 机房管理 /ops/serverRoom
// ============================================================================

// swagExcelServerRoomImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入机房
// @Description 上传 .xlsx 文件批量导入机房数据
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom/import [post]
func swagExcelServerRoomImport() {}

// swagExcelServerRoomExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出机房
// @Description 按筛选条件导出机房数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom/export [post]
func swagExcelServerRoomExport() {}

// swagExcelServerRoomTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载机房导入模板
// @Description 下载机房 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/serverRoom/template [get]
func swagExcelServerRoomTemplate() {}

// ============================================================================
// 专线管理 /ops/dedicatedLine
// ============================================================================

// swagExcelDedicatedLineImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入专线
// @Description 上传 .xlsx 文件批量导入专线数据
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine/import [post]
func swagExcelDedicatedLineImport() {}

// swagExcelDedicatedLineExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出专线
// @Description 按筛选条件导出专线数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine/export [post]
func swagExcelDedicatedLineExport() {}

// swagExcelDedicatedLineTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载专线导入模板
// @Description 下载专线 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/dedicatedLine/template [get]
func swagExcelDedicatedLineTemplate() {}

// ============================================================================
// 机房设备管理 /ops/roomDevice
// ============================================================================

// swagExcelRoomDeviceImport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导入机房设备
// @Description 上传 .xlsx 文件批量导入机房设备数据
// @Tags 运维管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel 文件(.xlsx)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice/import [post]
func swagExcelRoomDeviceImport() {}

// swagExcelRoomDeviceExport 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary Excel 导出机房设备
// @Description 按筛选条件导出机房设备数据为 .xlsx 文件
// @Tags 运维管理
// @Accept json
// @Produce application/octet-stream
// @Param request body object false "导出筛选条件(动态参数)"
// @Success 200 {file} file
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice/export [post]
func swagExcelRoomDeviceExport() {}

// swagExcelRoomDeviceTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载机房设备导入模板
// @Description 下载机房设备 Excel 导入模板文件(.xlsx)
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/roomDevice/template [get]
func swagExcelRoomDeviceTemplate() {}

// ============================================================================
// 工位导入辅助端点：部门名称↔代码映射表模板
// ============================================================================

// swagExcelDeptMappingTemplate 仅用于承载 Swagger 注解，不参与路由与业务。
// @Summary 下载部门名称↔代码映射表模板
// @Description 下载 sys_dept 启用部门的全量映射表(dept_name/parent_path/dept_code)，供工位导入时按代码校对部门名称
// @Tags 运维管理
// @Produce application/octet-stream
// @Success 200 {file} file
// @Failure 500 {object} response.Response
// @Router /ops/workstation/dept-mapping-template [get]
func swagExcelDeptMappingTemplate() {}
