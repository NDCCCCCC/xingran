package gormutil

import (
	"gorm.io/gorm"
)

// BatchLoader 批量加载器，用于解决N+1查询问题
type BatchLoader struct {
	db *gorm.DB
}

// NewBatchLoader 创建批量加载器
func NewBatchLoader(db *gorm.DB) *BatchLoader {
	return &BatchLoader{db: db}
}

// LoadBelongsTo 批量加载所属关联（解决N+1问题）
// 适用于用户加载部门、角色加载权限等场景
func (l *BatchLoader) LoadBelongsTo(
	dests []interface{}, // 目标对象列表
	getForeignKey func(interface{}) string, // 获取外键的函数
	model interface{}, // 要加载的模型
	setAssociation func(interface{}, interface{}), // 设置关联的函数
) error {
	if len(dests) == 0 {
		return nil
	}

	// 收集所有外键ID
	foreignKeys := make([]string, 0, len(dests))
	keyMap := make(map[string]bool)
	for _, dest := range dests {
		key := getForeignKey(dest)
		if key != "" && !keyMap[key] {
			foreignKeys = append(foreignKeys, key)
			keyMap[key] = true
		}
	}

	if len(foreignKeys) == 0 {
		return nil
	}

	// 批量查询关联数据
	var records []interface{}
	if err := l.db.Where("id IN ?", foreignKeys).Find(model, &records).Error; err != nil {
		return err
	}

	// 构建ID到记录的映射
	recordMap := make(map[string]interface{})
	// 这里需要根据实际模型类型来实现反射获取ID
	// 简化版本：假设records有GetID()方法或ID字段

	// 设置关联
	for _, dest := range dests {
		key := getForeignKey(dest)
		if record, ok := recordMap[key]; ok {
			setAssociation(dest, record)
		}
	}

	return nil
}

// LoadManyToMany 批量加载多对多关联
// 适用于用户加载角色、角色加载菜单等场景
func (l *BatchLoader) LoadManyToMany(
	ctx *gorm.DB,
	entityIDs []string,
	junctionTable string,
	foreignKey string,
	associationForeignKey string,
	associationModel interface{},
) (map[string][]string, error) {
	if len(entityIDs) == 0 {
		return make(map[string][]string), nil
	}

	// 批量查询关联表
	type Junction struct {
		EntityID      string `gorm:"column:${entity_foreign_key}"`
		AssociationID string `gorm:"column:${association_foreign_key}"`
	}

	junctions := make([]Junction, 0)
	query := ctx.Table(junctionTable).
		Where(foreignKey+" IN ?", entityIDs).
		Find(&junctions)

	if query.Error != nil {
		return nil, query.Error
	}

	// 构建实体ID到关联ID列表的映射
	result := make(map[string][]string)
	for _, j := range junctions {
		result[j.EntityID] = append(result[j.EntityID], j.AssociationID)
	}

	return result, nil
}

// LoadAssociationsBatch 批量加载关联数据并映射到实体
func (l *BatchLoader) LoadAssociationsBatch(
	ctx *gorm.DB,
	entityIDs []string,
	junctionTable, junctionEntityField, junctionAssocField string,
	associationTableName string,
	associationIDField string,
) (map[string][]map[string]interface{}, error) {
	if len(entityIDs) == 0 {
		return make(map[string][]map[string]interface{}), nil
	}

	// 1. 查询关联表数据
	junctionData := make([]map[string]interface{}, 0)
	if err := ctx.Table(junctionTable).
		Where(junctionEntityField+" IN ?", entityIDs).
		Find(&junctionData).Error; err != nil {
		return nil, err
	}

	// 2. 收集所有关联ID
	assocIDs := make([]string, 0)
	assocIDSet := make(map[string]bool)
	entityToAssocIDs := make(map[string][]string)

	for _, j := range junctionData {
		entityID := j[junctionEntityField].(string)
		assocID := j[junctionAssocField].(string)

		entityToAssocIDs[entityID] = append(entityToAssocIDs[entityID], assocID)

		if !assocIDSet[assocID] {
			assocIDs = append(assocIDs, assocID)
			assocIDSet[assocID] = true
		}
	}

	if len(assocIDs) == 0 {
		return make(map[string][]map[string]interface{}), nil
	}

	// 3. 批量查询关联数据
	associations := make([]map[string]interface{}, 0)
	if err := ctx.Table(associationTableName).
		Where(associationIDField+" IN ?", assocIDs).
		Find(&associations).Error; err != nil {
		return nil, err
	}

	// 4. 构建关联ID到数据的映射
	assocMap := make(map[string]map[string]interface{})
	for _, assoc := range associations {
		id := assoc[associationIDField].(string)
		assocMap[id] = assoc
	}

	// 5. 组装最终结果
	result := make(map[string][]map[string]interface{})
	for _, entityID := range entityIDs {
		assocIDs := entityToAssocIDs[entityID]
		assocList := make([]map[string]interface{}, 0, len(assocIDs))
		for _, aid := range assocIDs {
			if assocData, ok := assocMap[aid]; ok {
				assocList = append(assocList, assocData)
			}
		}
		result[entityID] = assocList
	}

	return result, nil
}
