package models

// DictType 字典类型模型
type DictType struct {
	BaseModel
	DictName string `gorm:"size:100;not null" json:"dictName"`
	DictType string `gorm:"uniqueIndex;size:100;not null" json:"dictType"`
	Status   int    `gorm:"default:0" json:"status"`
	Remark   string `gorm:"size:500" json:"remark,omitempty"`
}

// DictData 字典数据模型
type DictData struct {
	BaseModel
	DictSort  int     `gorm:"default:0" json:"dictSort"`
	DictLabel string  `gorm:"size:100;not null" json:"dictLabel"`
	DictValue string  `gorm:"size:100;not null" json:"dictValue"`
	DictType  string  `gorm:"size:100;index" json:"dictType"`
	CssClass  *string `gorm:"size:100" json:"cssClass,omitempty"`
	ListClass *string `gorm:"size:100" json:"listClass,omitempty"`
	IsDefault bool    `gorm:"default:false" json:"isDefault"`
	Status    int     `gorm:"default:0" json:"status"`
	Remark    string  `gorm:"size:500" json:"remark,omitempty"`
}

// TableName 设置表名
func (DictType) TableName() string {
	return "sys_dict_type"
}

// TableName 设置表名
func (DictData) TableName() string {
	return "sys_dict_data"
}
