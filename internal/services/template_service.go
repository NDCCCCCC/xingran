package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/utils"
	"gorm.io/gorm"
)

// TemplateService 配置模板服务
type TemplateService struct {
	db *gorm.DB
}

// NewTemplateService 创建配置模板服务
func NewTemplateService(db *gorm.DB) *TemplateService {
	return &TemplateService{db: db}
}

// TemplateStatistics 配置模板统计结果。
type TemplateStatistics struct {
	Total  int64 `json:"total"`
	System int64 `json:"system"` // is_system = true
	Custom int64 `json:"custom"` // is_system = false
	Init   int64 `json:"init"`   // template_type = 'init'
}

// GetStatistics 统计配置模板总数/系统/自定义/初始化模板数。
// 用条件聚合避免「加载全量行进内存再 filter」; is_system 用裸布尔表达式(PG/SQLite 双兼容)。
func (s *TemplateService) GetStatistics(ctx context.Context) (*TemplateStatistics, error) {
	var result TemplateStatistics
	err := s.db.WithContext(ctx).Model(&models.ConfigTemplate{}).
		Select(
			"COUNT(*) AS total",
			"SUM(CASE WHEN is_system THEN 1 ELSE 0 END) AS system",
			"SUM(CASE WHEN NOT is_system THEN 1 ELSE 0 END) AS custom",
			"SUM(CASE WHEN template_type = 'init' THEN 1 ELSE 0 END) AS init",
		).
		Scan(&result).Error
	if err != nil {
		return nil, fmt.Errorf("统计配置模板失败: %w", err)
	}
	return &result, nil
}

// ListRequest 列表请求
type ListTemplateRequest struct {
	base.BaseListRequest
	TemplateName *string
	TemplateType *models.TemplateType
	Vendor       *models.DeviceVendor
	DeviceType   *models.DeviceType
	IsSystem     *bool
}

// templateAllowedSortFields 配置模板可排序字段白名单(对应 sys_config_template 表列名)。
var templateAllowedSortFields = map[string]string{
	"templateName": "template_name",
	"templateType": "template_type",
	"vendor":       "vendor",
	"deviceType":   "device_type",
	"isSystem":     "is_system",
	"createdAt":    "created_at",
}

// List 获取配置模板列表
func (s *TemplateService) List(ctx context.Context, req *ListTemplateRequest) ([]models.ConfigTemplate, int64, error) {
	var templates []models.ConfigTemplate
	var total int64

	query := s.db.Model(&models.ConfigTemplate{})

	if req.TemplateName != nil && *req.TemplateName != "" {
		query = query.Where("template_name LIKE ?", "%"+*req.TemplateName+"%")
	}
	if req.TemplateType != nil {
		query = query.Where("template_type = ?", *req.TemplateType)
	}
	if req.Vendor != nil {
		query = query.Where("vendor = ?", *req.Vendor)
	}
	if req.DeviceType != nil {
		query = query.Where("device_type = ?", *req.DeviceType)
	}
	if req.IsSystem != nil {
		query = query.Where("is_system = ?", *req.IsSystem)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板总数失败: %w", err)
	}

	// 分页查询 - 用户排序(白名单)优先,无 OrderByColumn 时保留 created_at DESC 默认
	offset := (req.Current - 1) * req.PageSize
	query = base.ApplySort(query, req.BaseListRequest, templateAllowedSortFields)
	if req.OrderByColumn == "" {
		query = query.Order("created_at DESC")
	}
	if err := query.Offset(offset).Limit(req.PageSize).Find(&templates).Error; err != nil {
		return nil, 0, fmt.Errorf("查询模板列表失败: %w", err)
	}

	return templates, total, nil
}

// CreateRequest 创建请求
type CreateTemplateRequest struct {
	TemplateName    string
	TemplateCode    string
	TemplateType    models.TemplateType
	Vendor          models.DeviceVendor
	DeviceType      models.DeviceType
	TemplateContent string
	Variables       []models.TemplateVariable
	Description     string
	IsSystem        bool
	CreatedBy       string
}

// Create 创建配置模板
func (s *TemplateService) Create(ctx context.Context, req *CreateTemplateRequest) (*models.ConfigTemplate, error) {
	// 检查模板编码是否已存在
	var count int64
	if err := s.db.Model(&models.ConfigTemplate{}).Where("template_code = ?", req.TemplateCode).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查模板编码失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("模板编码已存在")
	}

	// 验证模板语法
	engine := utils.NewTemplateEngine()
	if err := engine.ValidateTemplate(req.TemplateContent); err != nil {
		return nil, fmt.Errorf("模板语法错误: %w", err)
	}

	// 序列化变量
	var variables models.TemplateVariables
	if req.Variables != nil {
		variables = req.Variables
	}

	template := models.ConfigTemplate{
		TemplateName:    req.TemplateName,
		TemplateCode:    req.TemplateCode,
		TemplateType:    req.TemplateType,
		Vendor:          req.Vendor,
		DeviceType:      req.DeviceType,
		TemplateContent: req.TemplateContent,
		Variables:       variables,
		Description:     req.Description,
		IsSystem:        req.IsSystem,
		BaseModel:       models.BaseModel{CreatedBy: req.CreatedBy},
	}

	if err := s.db.Create(&template).Error; err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	return &template, nil
}

// GetByID 根据ID获取模板
func (s *TemplateService) GetByID(ctx context.Context, id string) (*models.ConfigTemplate, error) {
	var template models.ConfigTemplate
	if err := s.db.Where("id = ?", id).First(&template).Error; err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}
	return &template, nil
}

// GetByCode 根据编码获取模板
func (s *TemplateService) GetByCode(ctx context.Context, code string) (*models.ConfigTemplate, error) {
	var template models.ConfigTemplate
	if err := s.db.Where("template_code = ?", code).First(&template).Error; err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}
	return &template, nil
}

// UpdateRequest 更新请求
type UpdateTemplateRequest struct {
	ID              string
	TemplateName    string
	TemplateType    models.TemplateType
	Vendor          models.DeviceVendor
	DeviceType      models.DeviceType
	TemplateContent string
	Variables       []models.TemplateVariable
	Description     string
	UpdatedBy       string
}

// Update 更新配置模板
func (s *TemplateService) Update(ctx context.Context, req *UpdateTemplateRequest) error {
	var template models.ConfigTemplate
	if err := s.db.Where("id = ?", req.ID).First(&template).Error; err != nil {
		return fmt.Errorf("模板不存在: %w", err)
	}

	// 系统内置模板不允许修改内容
	if template.IsSystem {
		return fmt.Errorf("系统内置模板不允许修改")
	}

	// 验证模板语法
	engine := utils.NewTemplateEngine()
	if err := engine.ValidateTemplate(req.TemplateContent); err != nil {
		return fmt.Errorf("模板语法错误: %w", err)
	}

	// 序列化变量
	var variables models.TemplateVariables
	if req.Variables != nil {
		variables = req.Variables
	}

	updates := map[string]interface{}{
		"template_name":    req.TemplateName,
		"template_type":    req.TemplateType,
		"vendor":           req.Vendor,
		"device_type":      req.DeviceType,
		"template_content": req.TemplateContent,
		"variables":        variables,
		"description":      req.Description,
		"updated_by":       req.UpdatedBy,
	}

	if err := s.db.Model(&template).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新模板失败: %w", err)
	}

	return nil
}

// Delete 删除配置模板
func (s *TemplateService) Delete(ctx context.Context, id string) error {
	var template models.ConfigTemplate
	if err := s.db.Where("id = ?", id).First(&template).Error; err != nil {
		return fmt.Errorf("模板不存在: %w", err)
	}

	// 系统内置模板不允许删除
	if template.IsSystem {
		return fmt.Errorf("系统内置模板不允许删除")
	}

	// 检查是否有执行记录使用此模板
	var count int64
	if err := s.db.Model(&models.ConfigExecution{}).Where("template_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查模板使用情况失败: %w", err)
	}

	if err := s.db.Where("id = ?", id).Delete(&models.ConfigTemplate{}).Error; err != nil {
		return fmt.Errorf("删除模板失败: %w", err)
	}

	return nil
}

// BatchDelete 批量删除模板
func (s *TemplateService) BatchDelete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Preview 预览模板渲染结果
func (s *TemplateService) Preview(ctx context.Context, templateID string, variables map[string]string) (string, error) {
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return "", err
	}

	engine := utils.NewTemplateEngine()

	// 将models.TemplateVariables转换为utils.TemplateVariables
	utilsVars := make([]utils.TemplateVariable, len(template.Variables))
	for i, v := range template.Variables {
		utilsVars[i] = utils.TemplateVariable{
			Name:         v.Name,
			Description:  v.Description,
			DefaultValue: v.DefaultValue,
			Required:     v.Required,
			Type:         v.Type,
			Options:      v.Options,
		}
	}

	// 从模板定义构建变量映射
	varMap, err := engine.BuildVariablesMap(utilsVars, variables)
	if err != nil {
		return "", err
	}

	// 渲染模板
	result, err := engine.Render(template.TemplateContent, varMap)
	if err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return result, nil
}

// Render 渲染模板（用于实际配置下发）
func (s *TemplateService) Render(ctx context.Context, templateCode string, variables map[string]string) (string, error) {
	template, err := s.GetByCode(ctx, templateCode)
	if err != nil {
		return "", err
	}

	engine := utils.NewTemplateEngine()

	// 将models.TemplateVariables转换为utils.TemplateVariables
	utilsVars := make([]utils.TemplateVariable, len(template.Variables))
	for i, v := range template.Variables {
		utilsVars[i] = utils.TemplateVariable{
			Name:         v.Name,
			Description:  v.Description,
			DefaultValue: v.DefaultValue,
			Required:     v.Required,
			Type:         v.Type,
			Options:      v.Options,
		}
	}

	// 从模板定义构建变量映射
	varMap, err := engine.BuildVariablesMap(utilsVars, variables)
	if err != nil {
		return "", err
	}

	// 渲染模板
	result, err := engine.Render(template.TemplateContent, varMap)
	if err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return result, nil
}

// GetVariables 获取模板变量定义
func (s *TemplateService) GetVariables(ctx context.Context, templateID string) ([]models.TemplateVariable, error) {
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	return template.Variables, nil
}

// ValidateVariables 验证模板变量
func (s *TemplateService) ValidateVariables(ctx context.Context, templateID string, variables map[string]string) error {
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return err
	}

	engine := utils.NewTemplateEngine()

	// 提取必需的变量名
	var requiredVars []string
	for _, v := range template.Variables {
		if v.Required {
			requiredVars = append(requiredVars, v.Name)
		}
	}

	// 将map[string]string转换为map[string]interface{}
	varMap := make(map[string]interface{})
	for k, v := range variables {
		varMap[k] = v
	}

	// 验证并渲染
	_, err = engine.RenderWithValidation(template.TemplateContent, varMap, requiredVars)
	return err
}

// GetTemplatesByVendor 根据厂商获取模板列表
func (s *TemplateService) GetTemplatesByVendor(ctx context.Context, vendor models.DeviceVendor, deviceType models.DeviceType) ([]models.ConfigTemplate, error) {
	var templates []models.ConfigTemplate

	query := s.db.Where("vendor = ?", vendor)
	if deviceType != "" {
		query = query.Where("device_type = ?", deviceType)
	}

	if err := query.Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("查询模板失败: %w", err)
	}

	return templates, nil
}

// Clone 克隆模板
func (s *TemplateService) Clone(ctx context.Context, templateID string, newName, newCode string, createdBy string) (*models.ConfigTemplate, error) {
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// 检查新编码是否已存在
	var count int64
	if err := s.db.Model(&models.ConfigTemplate{}).Where("template_code = ?", newCode).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查模板编码失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("模板编码已存在")
	}

	newTemplate := models.ConfigTemplate{
		TemplateName:    newName,
		TemplateCode:    newCode,
		TemplateType:    template.TemplateType,
		Vendor:          template.Vendor,
		DeviceType:      template.DeviceType,
		TemplateContent: template.TemplateContent,
		Variables:       template.Variables,
		Description:     template.Description,
		IsSystem:        false,
		BaseModel:       models.BaseModel{CreatedBy: createdBy},
	}

	if err := s.db.Create(&newTemplate).Error; err != nil {
		return nil, fmt.Errorf("克隆模板失败: %w", err)
	}

	return &newTemplate, nil
}

// Export 导出模板（JSON格式）
func (s *TemplateService) Export(ctx context.Context, templateID string) ([]byte, error) {
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("导出模板失败: %w", err)
	}

	return data, nil
}

// Import 导入模板（从JSON格式）
func (s *TemplateService) Import(ctx context.Context, data []byte, createdBy string) (*models.ConfigTemplate, error) {
	var template models.ConfigTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("解析模板数据失败: %w", err)
	}

	// 检查模板编码是否已存在
	var count int64
	if err := s.db.Model(&models.ConfigTemplate{}).Where("template_code = ?", template.TemplateCode).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("检查模板编码失败: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("模板编码已存在")
	}

	// 重置ID和系统标志
	template.ID = ""
	template.IsSystem = false
	template.CreatedBy = createdBy

	if err := s.db.Create(&template).Error; err != nil {
		return nil, fmt.Errorf("导入模板失败: %w", err)
	}

	return &template, nil
}
