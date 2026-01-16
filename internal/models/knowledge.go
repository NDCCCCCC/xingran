package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==================== 枚举定义 ====================

// KnowledgeArticleStatus 知识库文章状态枚举
type KnowledgeArticleStatus int

const (
	KnowledgeArticleStatusDraft     KnowledgeArticleStatus = 0 // 草稿
	KnowledgeArticleStatusPublished KnowledgeArticleStatus = 1 // 已发布
)

// ==================== 模型定义 ====================

// KnowledgeCategory 知识库分类
type KnowledgeCategory struct {
	BaseModel
	CategoryName string                 `gorm:"size:100;not null;uniqueIndex:idx_kb_category_name" json:"categoryName"`
	Description  string                 `gorm:"size:500" json:"description,omitempty"`
	ParentID     *string                `gorm:"type:uuid" json:"parentId,omitempty"`
	SortOrder    int                    `gorm:"default:0" json:"sortOrder"`
	Status       KnowledgeArticleStatus `gorm:"default:1" json:"status"`

	// 关联
	Parent   *KnowledgeCategory  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []KnowledgeCategory `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Articles []KnowledgeArticle  `gorm:"foreignKey:CategoryID" json:"articles,omitempty"`
}

// TableName 指定表名
func (KnowledgeCategory) TableName() string {
	return "sys_knowledge_category"
}

// KnowledgeArticle 知识库文章
type KnowledgeArticle struct {
	BaseModel
	Title             string                 `gorm:"size:200;not null" json:"title"`
	Content           string                 `gorm:"type:text;not null" json:"content"`
	Summary           string                 `gorm:"type:text" json:"summary,omitempty"`
	CategoryID        string                 `gorm:"type:uuid;not null;index:idx_kb_article_category,priority:1" json:"categoryId"`
	Status            KnowledgeArticleStatus `gorm:"default:0" json:"status"`
	ViewCount         int                    `gorm:"default:0" json:"viewCount"`
	LikeCount         int                    `gorm:"default:0" json:"likeCount"`
	SourceWorkOrderID *string                `gorm:"type:uuid" json:"sourceWorkOrderId,omitempty"` // 来源工单ID

	// 关联
	Category        *KnowledgeCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	SourceWorkOrder *WorkOrder         `gorm:"foreignKey:SourceWorkOrderID" json:"sourceWorkOrder,omitempty"`
	Tags            []KnowledgeTag     `gorm:"many2many:sys_kb_article_tags;joinForeignKey:ArticleID;joinReferences:TagID" json:"tags,omitempty"`
}

// TableName 指定表名
func (KnowledgeArticle) TableName() string {
	return "sys_knowledge_article"
}

// KnowledgeTag 知识库标签
type KnowledgeTag struct {
	ID        string    `gorm:"primaryKey;type:uuid" json:"id"`
	TagName   string    `gorm:"size:50;not null;uniqueIndex:idx_kb_tag_name" json:"tagName"`
	UseCount  int       `gorm:"default:0" json:"useCount"`
	CreatedAt time.Time `json:"createdAt"`

	// 关联
	Articles []KnowledgeArticle `gorm:"many2many:sys_kb_article_tags;joinForeignKey:TagID;joinReferences:ArticleID" json:"articles,omitempty"`
}

// BeforeCreate GORM钩子 - KnowledgeTag
func (kt *KnowledgeTag) BeforeCreate(tx *gorm.DB) error {
	if kt.ID == "" {
		kt.ID = uuid.New().String()
	}
	if kt.CreatedAt.IsZero() {
		kt.CreatedAt = time.Now()
	}
	return nil
}

// TableName 指定表名
func (KnowledgeTag) TableName() string {
	return "sys_knowledge_tag"
}

// KnowledgeArticleTag 文章标签关联表（用于many2many）
type KnowledgeArticleTag struct {
	ArticleID string    `gorm:"type:uuid;not null;index:idx_kb_article_tag_article,priority:1" json:"articleId"`
	TagID     string    `gorm:"type:uuid;not null;index:idx_kb_article_tag_tag,priority:1" json:"tagId"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName 指定表名
func (KnowledgeArticleTag) TableName() string {
	return "sys_kb_article_tags"
}
