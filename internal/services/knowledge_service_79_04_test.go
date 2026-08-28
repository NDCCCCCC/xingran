package services

// =====================================================================
// Phase 79-04 Task 1: knowledge 文章族 CRUD/搜索/统计/计数器
// Phase 79-04 Task 2(同文件追加): 分类树递归 + 标签族(GetOrCreateTag 双路径)
//
// 覆盖目标: knowledge_service.go 1.4% → ≥70%(基线 289 stmts / 285 unc,
// 79-RESEARCH §2)。
//
// 纪律(79-01 SUMMARY 手注沿用):
//   - helper 名带 7904 后缀(R5/D-79-06);fixture 用 t.TempDir() 文件库。
//   - 禁 t.Parallel()(同包串行)。
//   - 状态断言一律引用 models.KnowledgeArticleStatus* / models.WorkOrderStatus*
//     具名常量(E 簇反转:KnowledgeArticleStatusPublished=1,Phase 69-03/04 判定)。
//   - 生产代码零改动;quirk 按"只锁不修"纪律注释记录(Phase 73-03 决策)。
//
// quirk 锁定(Phase 73-03 记录,本 plan 复述,SUMMARY 复记):
//   Q1 knowledge UUID inline-Delete —— DeleteKnowledgeCategory(:683)/DeleteTag(:775)
//      用 `db.Delete(&Model{}, id)` 内联条件;GORM 对非数字字符串内联实参按原生 SQL
//      片段处理,UUID 字符串会生成非法 WHERE → 报错。测试用数字串主键走 happy path,
//      另用真实 UUID 锁定报错行为(见 TestKsv7904_CategoryCRUD / TestKsv7904_TagFamily)。
//   Q2 knowledge GetOrCreateTag 跨 tx 连接 —— CreateKnowledgeArticle 在事务内通过
//      外层 s.db(非 tx)做标签读建,连接池层面跨连接;:memory: sqlite 每连接独立空库
//      会读到空库,文件库(t.TempDir)无此问题。按现行为断言,不修。
// =====================================================================

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

const ksv7904Creator = "creator-7904"

// baseListReq7904 构造分页请求(缺省排序走 service 内 created_at DESC 默认分支)。
func baseListReq7904(current, pageSize int) base.BaseListRequest {
	return base.BaseListRequest{Current: current, PageSize: pageSize}
}

// newKsv7904 装配 KnowledgeService + sqlite(t.TempDir 文件库)+ AutoMigrate
// 知识库三族 model(文章/分类/标签 + 关联表)与 ConvertWorkOrderToArticle 引用的工单表。
// 文件库(而非 :memory:)是 Q2(GetOrCreateTag 跨 tx 连接)的规避前提:
// 事务连接与外层连接必须看到同一份数据。
func newKsv7904(t *testing.T) (*KnowledgeService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ksv7904.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "open sqlite temp db")
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.AutoMigrate(
		&models.KnowledgeCategory{},
		&models.KnowledgeArticle{},
		&models.KnowledgeTag{},
		&models.KnowledgeArticleTag{},
		&models.WorkOrder{},
	), "auto migrate knowledge chain models")
	return NewKnowledgeService(db), db
}

// ksv7904SeedTag 预置标签(绕开 Q2:事务内经外层连接 INSERT 必然 SQLITE_BUSY,
// Phase 73-03 同款处置 —— "Tag pre-seeded ... to avoid tx-vs-second-conn write deadlock")。
// CreateKnowledgeArticle/UpdateKnowledgeArticle/AutoCreateTagsFromContent 传入的
// tag 名一律先经本 helper 落库,让 GetOrCreateTag 走纯读分支。
func ksv7904SeedTag(t *testing.T, db *gorm.DB, name string) *models.KnowledgeTag {
	t.Helper()
	tag := &models.KnowledgeTag{TagName: name}
	require.NoError(t, db.Create(tag).Error, "seed tag %q", name)
	return tag
}

// ksv7904Category 预置一个分类(文章创建的 CategoryID 必填)。
func ksv7904Category(t *testing.T, db *gorm.DB, name string) *models.KnowledgeCategory {
	t.Helper()
	cat := &models.KnowledgeCategory{
		CategoryName: name,
		Status:       models.KnowledgeArticleStatusPublished,
		BaseModel:    models.BaseModel{CreatedBy: ksv7904Creator, UpdatedBy: ksv7904Creator},
	}
	require.NoError(t, db.Create(cat).Error, "seed category")
	return cat
}

// ksv7904WorkOrder 预置一条工单(ConvertWorkOrderToArticle 的来源行)。
func ksv7904WorkOrder(t *testing.T, db *gorm.DB, status models.WorkOrderStatus) *models.WorkOrder {
	t.Helper()
	wo := &models.WorkOrder{
		Title:       "打印机卡纸处理",
		Type:        models.WorkOrderTypeFault,
		CategoryID:  uuid.New().String(),
		SubmitterID: uuid.New().String(),
		Status:      status,
		Description: "3F 打印机卡纸,已更换搓纸轮",
	}
	require.NoError(t, db.Create(wo).Error, "seed work order")
	return wo
}

// ksv7904TagCounts 返回 tag_id → use_count 映射,便于断言使用次数差量。
func ksv7904TagCounts(t *testing.T, db *gorm.DB) map[string]int {
	t.Helper()
	var tags []models.KnowledgeTag
	require.NoError(t, db.Find(&tags).Error, "load tags")
	out := make(map[string]int, len(tags))
	for _, tg := range tags {
		out[tg.ID] = tg.UseCount
	}
	return out
}

// ksv7904AssocTagIDs 返回指定文章已关联的 tag_id 集合(升序,便于比较)。
func ksv7904AssocTagIDs(t *testing.T, db *gorm.DB, articleID string) []string {
	t.Helper()
	var assoc []models.KnowledgeArticleTag
	require.NoError(t, db.Where("article_id = ?", articleID).Find(&assoc).Error, "load associations")
	ids := make([]string, 0, len(assoc))
	for _, a := range assoc {
		ids = append(ids, a.TagID)
	}
	sort.Strings(ids)
	return ids
}

// -------------------------------------------------------------------------
// Task 1: 文章族
// -------------------------------------------------------------------------

// TestKsv7904_CreateArticle_WithTags 请求含 tag 名列表 + UUID tag → 文章行/标签行/关联行齐备,
// creatorID 落库(GetKnowledgeArticle 返回值带 Preload Tags/Category)。
func TestKsv7904_CreateArticle_WithTags(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	cat := ksv7904Category(t, db, "运维手册")

	// 预置标签:名称 2 个(Q2 锁定:事务内经外层连接 INSERT 会 SQLITE_BUSY,
	// 必须 pre-seed 让 GetOrCreateTag 走读分支)+ UUID 形态既有标签
	// (走 "是有效 UUID 直接使用" 分支)
	network := ksv7904SeedTag(t, db, "网络")
	fault := ksv7904SeedTag(t, db, "故障")
	existing := &models.KnowledgeTag{ID: uuid.New().String(), TagName: "既有标签"}
	require.NoError(t, db.Create(existing).Error)

	art, err := svc.CreateKnowledgeArticle(ctx, &KnowledgeArticleCreateRequest{
		Title:      "交换机端口故障排查",
		Content:    "先看光模块收发光,再看端口错包。",
		Summary:    "端口排查三步法",
		CategoryID: cat.ID,
		Status:     int(models.KnowledgeArticleStatusPublished),
		TagIDs:     []string{"网络", "故障", existing.ID},
	}, ksv7904Creator)
	require.NoError(t, err)
	require.NotNil(t, art)
	assert.NotEmpty(t, art.ID)
	assert.Equal(t, "交换机端口故障排查", art.Title)
	assert.Equal(t, models.KnowledgeArticleStatusPublished, art.Status)
	assert.Equal(t, ksv7904Creator, art.CreatedBy, "creatorID 必须落库")
	require.NotNil(t, art.Category)
	assert.Equal(t, cat.ID, art.Category.ID, "Preload(Category) 命中")

	// 标签行:名称 2 个自动创建 + UUID 1 个既有 = 3 行
	var tagNames []string
	for _, tg := range art.Tags {
		tagNames = append(tagNames, tg.TagName)
	}
	assert.ElementsMatch(t, []string{"网络", "故障", "既有标签"}, tagNames)

	// 关联行 3 条
	assert.Len(t, ksv7904AssocTagIDs(t, db, art.ID), 3, "article-tag 关联行必须齐备")
	// 新建标签 use_count=1;UUID 既有标签同样 +1(首次关联)
	counts := ksv7904TagCounts(t, db)
	assert.Equal(t, 1, counts[existing.ID], "UUID 标签首次关联 use_count +1")

	// 重复 tag 名去重:同一 tag 输入两次只落一条关联(名称已 pre-seed → 纯读分支)
	assert.NotNil(t, network)
	assert.NotNil(t, fault)
	art2, err := svc.CreateKnowledgeArticle(ctx, &KnowledgeArticleCreateRequest{
		Title:      "重复标签文章",
		Content:    "内容",
		CategoryID: cat.ID,
		TagIDs:     []string{"网络", "网络"},
	}, ksv7904Creator)
	require.NoError(t, err)
	assert.Len(t, ksv7904AssocTagIDs(t, db, art2.ID), 1, "重复 tag 名必须去重(检查已关联分支)")
}

// TestKsv7904_ArticleCRUD_RoundTrip List(分页 + 状态/分类过滤 + 默认排序)→ Get →
// Update → Delete;Get 不存在 → 错误。
func TestKsv7904_ArticleCRUD_RoundTrip(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	catA := ksv7904Category(t, db, "分类A")
	catB := ksv7904Category(t, db, "分类B")

	var created []*models.KnowledgeArticle
	for i := 0; i < 3; i++ {
		art, err := svc.CreateKnowledgeArticle(ctx, &KnowledgeArticleCreateRequest{
			Title:      fmt.Sprintf("文章%d", i),
			Content:    "内容",
			CategoryID: catA.ID,
			Status:     int(models.KnowledgeArticleStatusPublished),
		}, ksv7904Creator)
		require.NoError(t, err)
		created = append(created, art)
	}
	draft, err := svc.CreateKnowledgeArticle(ctx, &KnowledgeArticleCreateRequest{
		Title:      "草稿文章",
		Content:    "内容",
		CategoryID: catB.ID,
		Status:     int(models.KnowledgeArticleStatusDraft),
	}, ksv7904Creator)
	require.NoError(t, err)

	// 分页:pageSize=2 current=1 → 2 行 + total=4
	list, total, err := svc.GetKnowledgeArticleList(ctx, &KnowledgeArticleListRequest{
		BaseListRequest: baseListReq7904(1, 2),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, list, 2)
	assert.NotNil(t, list[0].Category, "列表必须 Preload(Category)")

	// 状态过滤:只看已发布 → 3 篇
	pub := int(models.KnowledgeArticleStatusPublished)
	_, total, err = svc.GetKnowledgeArticleList(ctx, &KnowledgeArticleListRequest{
		BaseListRequest: baseListReq7904(1, 10),
		Status:          &pub,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total, "状态过滤必须命中 models.KnowledgeArticleStatusPublished")

	// 分类过滤
	list, total, err = svc.GetKnowledgeArticleList(ctx, &KnowledgeArticleListRequest{
		BaseListRequest: baseListReq7904(1, 10),
		CategoryID:      catB.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, draft.ID, list[0].ID)

	// 标题模糊过滤
	_, total, err = svc.GetKnowledgeArticleList(ctx, &KnowledgeArticleListRequest{
		BaseListRequest: baseListReq7904(1, 10),
		Title:           "草稿",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// CreatedBy 过滤(无命中 → 空集)
	list, total, err = svc.GetKnowledgeArticleList(ctx, &KnowledgeArticleListRequest{
		BaseListRequest: baseListReq7904(1, 10),
		CreatedBy:       "no-such-creator",
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)

	// Get 读回
	got, err := svc.GetKnowledgeArticle(ctx, created[0].ID)
	require.NoError(t, err)
	assert.Equal(t, created[0].Title, got.Title)

	// Get 不存在 → 错误
	_, err = svc.GetKnowledgeArticle(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")

	// Update(指针字段)读回
	newTitle := "文章0-改"
	newStatus := int(models.KnowledgeArticleStatusDraft)
	require.NoError(t, svc.UpdateKnowledgeArticle(ctx, created[0].ID, &KnowledgeArticleUpdateRequest{
		Title:  &newTitle,
		Status: &newStatus,
	}, "operator-7904"))
	got, err = svc.GetKnowledgeArticle(ctx, created[0].ID)
	require.NoError(t, err)
	assert.Equal(t, newTitle, got.Title)
	assert.Equal(t, models.KnowledgeArticleStatusDraft, got.Status)
	assert.Equal(t, "operator-7904", got.UpdatedBy)

	// Update 不存在 → 错误
	err = svc.UpdateKnowledgeArticle(ctx, uuid.New().String(), &KnowledgeArticleUpdateRequest{}, "op")
	require.Error(t, err)

	// Delete 后 Get 报错
	require.NoError(t, svc.DeleteKnowledgeArticle(ctx, created[2].ID))
	_, err = svc.GetKnowledgeArticle(ctx, created[2].ID)
	require.Error(t, err)
	// Delete 不存在 → 错误
	err = svc.DeleteKnowledgeArticle(ctx, uuid.New().String())
	require.Error(t, err)
}

// TestKsv7904_UpdateArticle_TagSync 更新时替换 tag 集合 → 关联行差量正确(新增/移除)
// 且 use_count 同步增减。
func TestKsv7904_UpdateArticle_TagSync(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	cat := ksv7904Category(t, db, "同步分类")

	// Q2 锁定:所有 tag 名必须先 pre-seed(事务内经外层连接 INSERT 会 SQLITE_BUSY)。
	jia := ksv7904SeedTag(t, db, "甲")
	yi := ksv7904SeedTag(t, db, "乙")
	bing := ksv7904SeedTag(t, db, "丙")

	art, err := svc.CreateKnowledgeArticle(ctx, &KnowledgeArticleCreateRequest{
		Title:      "标签同步",
		Content:    "内容",
		CategoryID: cat.ID,
		TagIDs:     []string{"甲", "乙"},
	}, ksv7904Creator)
	require.NoError(t, err)
	require.NoError(t, err)
	before := ksv7904TagCounts(t, db)

	// 替换为 {乙, 丙}:甲 移除、丙 新增
	require.NoError(t, svc.UpdateKnowledgeArticle(ctx, art.ID, &KnowledgeArticleUpdateRequest{
		TagIDs: []string{"乙", "丙"},
	}, "op-7904"))

	assoc := ksv7904AssocTagIDs(t, db, art.ID)
	after := ksv7904TagCounts(t, db)

	// QUIRK-79-04-A(新发现,锁定不修,零生产改动):
	// UpdateKnowledgeArticle 的旧关联清理用 `tx.Delete(&oldTag)`(knowledge_service.go:333),
	// 而 models.KnowledgeArticleTag 三列均非主键 → GORM 报 "WHERE conditions required"
	// 且返回值被忽略 → 旧关联行从不删除。净效果:标签同步只增不减。
	// 生产修复属 escape hatch 范畴(需给关联表补主键或改 Where 删除),本 plan 只锁现行为。
	assert.ElementsMatch(t, []string{jia.ID, yi.ID, bing.ID}, assoc,
		"锁定现行为:旧关联(甲)未被删除,关联集累积为 {甲,乙,丙}")
	assert.Equal(t, before[jia.ID]-1, after[jia.ID],
		"锁定现行为:use_count -1 与关联删除不同步(failed delete 后无条件减)")
	assert.Equal(t, before[yi.ID]-1, after[yi.ID],
		"锁定现行为:乙 是旧标签 → use_count 同样被 -1(尽管关联行仍在,不再 +1)")
	assert.Equal(t, before[bing.ID]+1, after[bing.ID], "新增的 丙 use_count +1")

	// TagIDs = 空切片(非 nil)→ 按 QUIRK-79-04-A 锁定:旧关联删不掉,集合保持不变
	require.NoError(t, svc.UpdateKnowledgeArticle(ctx, art.ID, &KnowledgeArticleUpdateRequest{
		TagIDs: []string{},
	}, "op-7904"))
	assert.ElementsMatch(t, []string{jia.ID, yi.ID, bing.ID}, ksv7904AssocTagIDs(t, db, art.ID),
		"锁定现行为:空 TagIDs 也清不掉旧关联(QUIRK-79-04-A)")
	// TagIDs = nil → 不进关联分支,集合同样不变
	require.NoError(t, svc.UpdateKnowledgeArticle(ctx, art.ID, &KnowledgeArticleUpdateRequest{}, "op-7904"))
	assert.ElementsMatch(t, []string{jia.ID, yi.ID, bing.ID}, ksv7904AssocTagIDs(t, db, art.ID),
		"nil TagIDs 不改动关联")
}

// TestKsv7904_SearchArticles 关键字命中标题/内容/摘要三分支 + 已发布过滤 + 分页 +
// 分类/标签过滤;空关键字返回全部已发布。
func TestKsv7904_SearchArticles(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	catA := ksv7904Category(t, db, "搜索A")
	catB := ksv7904Category(t, db, "搜索B")

	mk := func(title, content, summary string, status models.KnowledgeArticleStatus, catID string) *models.KnowledgeArticle {
		art, err := svc.CreateKnowledgeArticle(ctx, &KnowledgeArticleCreateRequest{
			Title: title, Content: content, Summary: summary, CategoryID: catID,
			Status: int(status),
		}, ksv7904Creator)
		require.NoError(t, err)
		return art
	}
	hitTitle := mk("核心交换机巡检", "正文", "", models.KnowledgeArticleStatusPublished, catA.ID)
	_ = mk("机房规范", "空调漏水应急处理流程", "", models.KnowledgeArticleStatusPublished, catA.ID)
	_ = mk("无关标题", "正文", "摘要里提到光纤熔接", models.KnowledgeArticleStatusPublished, catB.ID)
	_ = mk("光纤熔接(草稿)", "正文", "", models.KnowledgeArticleStatusDraft, catA.ID) // 草稿不参与搜索

	// 空关键字 → 全部已发布(3 篇)
	list, total, err := svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total, "空关键字必须返回全部已发布文章")
	assert.Len(t, list, 3)

	// 标题命中
	list, total, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{Keyword: "巡检"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, hitTitle.ID, list[0].ID)

	// 内容命中
	_, total, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{Keyword: "空调漏水"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// 摘要命中
	_, total, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{Keyword: "熔接"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total, "summary LIKE 分支必须命中")

	// 分类过滤
	_, total, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{CategoryID: catB.ID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	// QUIRK-79-04-B(新发现,锁定不修,零生产改动):TagID 过滤走
	// INNER JOIN sys_kb_article_tags(knowledge_service.go:478)后仍追加
	// Order("created_at DESC")(:504),两表都有 created_at → sqlite/PG 均报
	// "ambiguous column name: created_at"。生产修复属 escape hatch 范畴,本 plan 只锁现行为。
	reserved := ksv7904SeedTag(t, db, "专属标签")
	tagged := mk("带标签文章", "正文", "", models.KnowledgeArticleStatusPublished, catA.ID)
	require.NoError(t, svc.UpdateKnowledgeArticle(ctx, tagged.ID, &KnowledgeArticleUpdateRequest{
		TagIDs: []string{"专属标签"},
	}, "op"))
	_, _, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{TagID: reserved.ID})
	require.Error(t, err, "锁定现行为:TagID 过滤必报 ambiguous column name")
	assert.Contains(t, err.Error(), "ambiguous column name")

	// 分页边界:pageSize=0 → 默认 100;pageSize>500 → 钳制到 500(钳制分支断言不炸即可)
	list, total, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{PageSize: 0, PageNum: 0})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, list, 4)
	_, _, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{PageSize: 1000, PageNum: 0})
	require.NoError(t, err)
	assert.Len(t, list, 4, "pageSize 钳制到 500 后仍返回全部 4 条")
	_, _, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{PageSize: 2, PageNum: -1})
	require.NoError(t, err)
	assert.Len(t, list, 2, "负 pageNum 归零后等同第 1 页")

	// 分页:pageSize=2 pageNum=1 → 第二页 1 条
	list, total, err = svc.SearchKnowledgeArticles(ctx, &SearchKnowledgeRequest{PageSize: 2, PageNum: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Len(t, list, 2, "第 2 页必须只有剩余 2 条")
}

// TestKsv7904_IncrementCounters IncrementViewCount/IncrementLikeCount 各 +1;
// 不存在 ID 不报错(Update 语义,RowsAffected 不校验)。
func TestKsv7904_IncrementCounters(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	cat := ksv7904Category(t, db, "计数分类")

	art, err := svc.CreateKnowledgeArticle(ctx, &KnowledgeArticleCreateRequest{
		Title: "计数文章", Content: "正文", CategoryID: cat.ID,
		Status: int(models.KnowledgeArticleStatusPublished),
	}, ksv7904Creator)
	require.NoError(t, err)

	require.NoError(t, svc.IncrementViewCount(ctx, art.ID))
	require.NoError(t, svc.IncrementViewCount(ctx, art.ID))
	require.NoError(t, svc.IncrementLikeCount(ctx, art.ID))

	got, err := svc.GetKnowledgeArticle(ctx, art.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.ViewCount)
	assert.Equal(t, 1, got.LikeCount)

	// 不存在 ID:UPDATE 影响 0 行但 GORM 不视为错误 → 按现行为断言不报错
	require.NoError(t, svc.IncrementViewCount(ctx, uuid.New().String()))
	require.NoError(t, svc.IncrementLikeCount(ctx, uuid.New().String()))
}

// TestKsv7904_ConvertWorkOrderToArticle 预置工单行 → 转换后文章字段映射一致;
// 工单不存在/状态不允许/已转换过 → 错误。
func TestKsv7904_ConvertWorkOrderToArticle(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	cat := ksv7904Category(t, db, "工单转化分类")

	// 状态不允许:待处理工单不可转
	pending := ksv7904WorkOrder(t, db, models.WorkOrderStatusPending)
	_, err := svc.ConvertWorkOrderToArticle(ctx, pending.ID, &ConvertWorkOrderToArticleRequest{
		Title: "x", Content: "y", CategoryID: cat.ID,
	}, ksv7904Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已完成或已关闭")

	// 工单不存在
	_, err = svc.ConvertWorkOrderToArticle(ctx, uuid.New().String(), &ConvertWorkOrderToArticleRequest{
		Title: "x", Content: "y", CategoryID: cat.ID,
	}, ksv7904Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "工单不存在")

	// 已完成工单 → 转换成功,字段映射一致(Q2 锁定:tag 先 pre-seed)
	ksv7904SeedTag(t, db, "打印")
	done := ksv7904WorkOrder(t, db, models.WorkOrderStatusCompleted)
	art, err := svc.ConvertWorkOrderToArticle(ctx, done.ID, &ConvertWorkOrderToArticleRequest{
		Title:      "打印机卡纸知识库",
		Content:    "更换搓纸轮步骤",
		Summary:    "卡纸处理",
		CategoryID: cat.ID,
		Status:     int(models.KnowledgeArticleStatusPublished),
		TagIDs:     []string{"打印"},
	}, ksv7904Creator)
	require.NoError(t, err)
	require.NotNil(t, art)
	assert.Equal(t, "打印机卡纸知识库", art.Title)
	assert.Equal(t, cat.ID, art.CategoryID)
	require.NotNil(t, art.SourceWorkOrderID)
	assert.Equal(t, done.ID, *art.SourceWorkOrderID, "来源工单必须回填")
	require.NotNil(t, art.SourceWorkOrder, "Preload(SourceWorkOrder) 命中")
	assert.Equal(t, done.ID, art.SourceWorkOrder.ID)

	// 同一工单二次转换 → 拒绝
	_, err = svc.ConvertWorkOrderToArticle(ctx, done.ID, &ConvertWorkOrderToArticleRequest{
		Title: "再转一次", Content: "y", CategoryID: cat.ID,
	}, ksv7904Creator)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已经转换")

	// 已关闭工单同样允许转换
	closed := ksv7904WorkOrder(t, db, models.WorkOrderStatusClosed)
	_, err = svc.ConvertWorkOrderToArticle(ctx, closed.ID, &ConvertWorkOrderToArticleRequest{
		Title: "已关闭工单转化", Content: "y", CategoryID: cat.ID,
	}, ksv7904Creator)
	require.NoError(t, err)
}

// TestKsv7904_ArticleStatistics 预置多状态文章 → KnowledgeArticleStatistics 各计数与手算
// 一致(对照既有 knowledge_statistics_test.go 的条件聚合口径,本测试补足 service 链路)。
func TestKsv7904_ArticleStatistics(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	cat := ksv7904Category(t, db, "统计分类")

	seed := func(title string, status models.KnowledgeArticleStatus, views, likes int) {
		t.Helper()
		require.NoError(t, db.Create(&models.KnowledgeArticle{
			Title: title, Content: "正文", CategoryID: cat.ID, Status: status,
			ViewCount: views, LikeCount: likes,
		}).Error)
	}
	seed("pub-1", models.KnowledgeArticleStatusPublished, 10, 2)
	seed("pub-2", models.KnowledgeArticleStatusPublished, 20, 3)
	seed("draft-1", models.KnowledgeArticleStatusDraft, 5, 0)

	stats, err := svc.GetArticleStatistics(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(3), stats.Total)
	assert.Equal(t, int64(1), stats.Draft, "draft = models.KnowledgeArticleStatusDraft 计数")
	assert.Equal(t, int64(2), stats.Published, "published = models.KnowledgeArticleStatusPublished 计数")
	assert.Equal(t, int64(35), stats.TotalViews)
	assert.Equal(t, int64(5), stats.TotalLikes)

	// 空表 → 全 0(COALESCE 分支)
	svc2, db2 := newKsv7904(t)
	_ = db2
	empty, err := svc2.GetArticleStatistics(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), empty.Total)
	assert.Equal(t, int64(0), empty.TotalViews, "空集 SUM 必须 COALESCE 为 0")
}

// TestKsv7904_ParseTagsFromContent 表驱动:#标记 解析(无标记/单标记/多标记/重复/空串/
// 超长标记),纯函数直调。
func TestKsv7904_ParseTagsFromContent(t *testing.T) {
	svc, _ := newKsv7904(t)

	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "无标记", content: "普通正文,没有标签", want: []string{}},
		{name: "空串", content: "", want: []string{}},
		{name: "单标记", content: "看这个 #网络 问题", want: []string{"网络"}},
		{name: "多标记", content: "#网络 和 #故障 一起出现", want: []string{"网络", "故障"}},
		{name: "重复标记去重", content: "#网络 与 #网络 重复", want: []string{"网络"}},
		{name: "井号包围去分隔符", content: "#故障#", want: []string{"故障"}},
		{name: "超长标记丢弃", content: "#abcdefghijklmnopqrstuv", want: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.ParseTagsFromContent(tc.content)
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}

// -------------------------------------------------------------------------
// Task 2: 分类树 + 标签族
// -------------------------------------------------------------------------

// TestKsv7904_CategoryTree_Recursive 建父→子→孙三层 → GetKnowledgeCategoryList 返回树
// 且 loadKnowledgeChildrenCategories 递归填充 Children;ParentID/Status 过滤与
// GetKnowledgeCategory(含 Parent Preload)分支。
func TestKsv7904_CategoryTree_Recursive(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)

	root, err := svc.CreateKnowledgeCategory(ctx, &KnowledgeCategoryCreateRequest{
		CategoryName: "根分类", SortOrder: 1,
		Status: int(models.KnowledgeArticleStatusPublished),
	}, ksv7904Creator)
	require.NoError(t, err)
	mid, err := svc.CreateKnowledgeCategory(ctx, &KnowledgeCategoryCreateRequest{
		CategoryName: "中间分类", ParentID: &root.ID, SortOrder: 2,
		Status: int(models.KnowledgeArticleStatusPublished),
	}, ksv7904Creator)
	require.NoError(t, err)
	leaf, err := svc.CreateKnowledgeCategory(ctx, &KnowledgeCategoryCreateRequest{
		CategoryName: "叶子分类", ParentID: &mid.ID, SortOrder: 3,
		Status: int(models.KnowledgeArticleStatusPublished),
	}, ksv7904Creator)
	require.NoError(t, err)
	// 停用分类(Status 过滤分支用)。KnowledgeArticleStatusDraft=0 是零值,GORM 建行
	// 时会被列 default:1 覆盖(与 73-03 network seed 同款零值跳过 quirk)→ 建后强制回写。
	disabled, err := svc.CreateKnowledgeCategory(ctx, &KnowledgeCategoryCreateRequest{
		CategoryName: "停用分类", SortOrder: 4,
		Status: int(models.KnowledgeArticleStatusDraft),
	}, ksv7904Creator)
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.KnowledgeCategory{}).Where("id = ?", disabled.ID).
		Update("status", int(models.KnowledgeArticleStatusDraft)).Error)

	// 全树:根层只有 根分类 + 停用分类,且孙层被递归填充
	roots, err := svc.GetKnowledgeCategoryList(ctx, &KnowledgeCategoryListRequest{})
	require.NoError(t, err)
	require.Len(t, roots, 2, "根层只应返回 parent_id IS NULL 的分类")
	byName := map[string]*models.KnowledgeCategory{}
	for i := range roots {
		byName[roots[i].CategoryName] = &roots[i]
	}
	gotRoot := byName["根分类"]
	require.NotNil(t, gotRoot)
	require.Len(t, gotRoot.Children, 1, "loadKnowledgeChildrenCategories 第一层")
	require.Len(t, gotRoot.Children[0].Children, 1, "递归第二层必须填充")
	assert.Equal(t, mid.ID, gotRoot.Children[0].ID)
	assert.Equal(t, leaf.ID, gotRoot.Children[0].Children[0].ID)

	// ParentID 过滤:parent_id = mid → 返回 mid 的直接子层(leaf),leaf 无下级
	parentID := mid.ID
	subtree, err := svc.GetKnowledgeCategoryList(ctx, &KnowledgeCategoryListRequest{ParentID: &parentID})
	require.NoError(t, err)
	require.Len(t, subtree, 1)
	assert.Equal(t, leaf.ID, subtree[0].ID, "ParentID 过滤返回该层子分类而非该分类自身")
	assert.Empty(t, subtree[0].Children)

	// Status 过滤:只看停用 → 只返回 停用分类
	stopped := int(models.KnowledgeArticleStatusDraft)
	stoppedList, err := svc.GetKnowledgeCategoryList(ctx, &KnowledgeCategoryListRequest{Status: &stopped})
	require.NoError(t, err)
	require.Len(t, stoppedList, 1)
	assert.Equal(t, disabled.ID, stoppedList[0].ID)

	// GetKnowledgeCategory:Parent Preload 命中
	got, err := svc.GetKnowledgeCategory(ctx, mid.ID)
	require.NoError(t, err)
	require.NotNil(t, got.Parent)
	assert.Equal(t, root.ID, got.Parent.ID)

	// GetKnowledgeCategory 不存在 → 错误
	_, err = svc.GetKnowledgeCategory(ctx, uuid.New().String())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "知识库分类不存在")
}

// TestKsv7904_CategoryCRUD Create/Update round-trip;删除含子类/含文章的分类被拒;
// 删除叶子分类 happy path 用数字串主键(Q1 UUID inline-Delete 锁定,见文件头)。
func TestKsv7904_CategoryCRUD(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	root := ksv7904Category(t, db, "CRUD根分类")

	// Create round-trip
	cat, err := svc.CreateKnowledgeCategory(ctx, &KnowledgeCategoryCreateRequest{
		CategoryName: "CRUD子分类",
		Description:  "描述",
		ParentID:     &root.ID,
		SortOrder:    7,
		Status:       int(models.KnowledgeArticleStatusPublished),
	}, ksv7904Creator)
	require.NoError(t, err)
	assert.Equal(t, "CRUD子分类", cat.CategoryName)
	assert.Equal(t, 7, cat.SortOrder)
	assert.Equal(t, ksv7904Creator, cat.CreatedBy)

	// Update(指针字段)读回
	newName := "CRUD子分类-改"
	newDesc := "新描述"
	newSort := 9
	newStatus := int(models.KnowledgeArticleStatusDraft)
	require.NoError(t, svc.UpdateKnowledgeCategory(ctx, cat.ID, &KnowledgeCategoryUpdateRequest{
		CategoryName: &newName, Description: &newDesc, SortOrder: &newSort, Status: &newStatus,
	}, "op-7904"))
	got, err := svc.GetKnowledgeCategory(ctx, cat.ID)
	require.NoError(t, err)
	assert.Equal(t, newName, got.CategoryName)
	assert.Equal(t, newDesc, got.Description)
	assert.Equal(t, newSort, got.SortOrder)
	assert.Equal(t, models.KnowledgeArticleStatusDraft, got.Status)
	assert.Equal(t, "op-7904", got.UpdatedBy)

	// Update 不存在 → 错误
	err = svc.UpdateKnowledgeCategory(ctx, uuid.New().String(), &KnowledgeCategoryUpdateRequest{}, "op")
	require.Error(t, err)

	// 删除被拒:含子分类
	err = svc.DeleteKnowledgeCategory(ctx, root.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "子分类")

	// 删除被拒:含关联文章
	_ = ksv7904Article(t, svc, cat.ID)
	err = svc.DeleteKnowledgeCategory(ctx, cat.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "关联文章")

	// 删除 happy path:叶子且无文章。Q1 锁定:inline Delete 只认数字串主键,
	// 因此本分支用数字串 ID 直插一行(GORM 直插不受 inline 条件限制)。
	digit := &models.KnowledgeCategory{
		BaseModel:    models.BaseModel{ID: "9901"},
		CategoryName: "数字串叶子分类",
		Status:       models.KnowledgeArticleStatusPublished,
	}
	require.NoError(t, db.Create(digit).Error)
	require.NoError(t, svc.DeleteKnowledgeCategory(ctx, "9901"), "数字串主键的 inline Delete 必须成功")
	_, err = svc.GetKnowledgeCategory(ctx, "9901")
	require.Error(t, err, "删除后必须不可再查")

	// Q1 锁定:真实 UUID 主键 → inline Delete 被当作原生 SQL 片段 → 报错(不修)
	uuidLeaf := ksv7904Category(t, db, "UUID叶子分类")
	err = svc.DeleteKnowledgeCategory(ctx, uuidLeaf.ID)
	require.Error(t, err, "锁定现行为:UUID 主键的 inline Delete 报错(Q1)")
}

// ksv7904Article 快捷建一篇文章(Q2:不带标签,避免 pre-seed 噪音)。
func ksv7904Article(t *testing.T, svc *KnowledgeService, categoryID string) *models.KnowledgeArticle {
	t.Helper()
	art, err := svc.CreateKnowledgeArticle(context.Background(), &KnowledgeArticleCreateRequest{
		Title: "关联文章", Content: "正文", CategoryID: categoryID,
		Status: int(models.KnowledgeArticleStatusPublished),
	}, ksv7904Creator)
	require.NoError(t, err)
	return art
}

// TestKsv7904_TagFamily GetAllTags 排序 / GetTagByName 命中与未命中 / CreateTag 重复名 /
// UpdateTag / DeleteTag(数字串 happy + UUID Q1 锁定 + 关联清理)全链。
func TestKsv7904_TagFamily(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)

	// GetTagByName 未命中 → (nil, nil) 不报错(实现的分支形态)
	miss, err := svc.GetTagByName(ctx, "不存在标签")
	require.NoError(t, err)
	assert.Nil(t, miss, "未命中必须返回 nil tag 而非错误")

	// CreateTag
	tag, err := svc.CreateTag(ctx, "标签甲")
	require.NoError(t, err)
	assert.NotEmpty(t, tag.ID)
	assert.Equal(t, "标签甲", tag.TagName)
	assert.Equal(t, 0, tag.UseCount)

	// CreateTag 重复名 → 唯一索引拒绝
	_, err = svc.CreateTag(ctx, "标签甲")
	require.Error(t, err, "重复名必须被 idx_kb_tag_name 唯一索引拒绝")

	// GetTagByName 命中
	hit, err := svc.GetTagByName(ctx, "标签甲")
	require.NoError(t, err)
	require.NotNil(t, hit)
	assert.Equal(t, tag.ID, hit.ID)

	// GetAllTags 排序:use_count DESC, created_at ASC
	_ = ksv7904SeedTag(t, db, "标签乙")
	_ = ksv7904SeedTag(t, db, "标签丙")
	require.NoError(t, db.Model(&models.KnowledgeTag{}).Where("tag_name = ?", "标签乙").
		UpdateColumn("use_count", 5).Error)
	require.NoError(t, db.Model(&models.KnowledgeTag{}).Where("tag_name = ?", "标签丙").
		UpdateColumn("use_count", 9).Error)
	all, err := svc.GetAllTags(ctx)
	require.NoError(t, err)
	require.Len(t, all, 3)
	assert.Equal(t, []string{"标签丙", "标签乙", "标签甲"},
		[]string{all[0].TagName, all[1].TagName, all[2].TagName}, "use_count 降序排列")

	// UpdateTag 读回
	require.NoError(t, svc.UpdateTag(ctx, tag.ID, "标签甲-改"))
	got, err := svc.GetTagByName(ctx, "标签甲-改")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, tag.ID, got.ID)

	// UpdateTag 不存在 ID → GORM 0 行更新不报错(锁定 Update 语义)
	require.NoError(t, svc.UpdateTag(ctx, uuid.New().String(), "无人生还"))

	// DeleteTag happy path:数字串主键(Q1 锁定)+ 关联清理
	digit := &models.KnowledgeTag{ID: "8801", TagName: "数字串标签"}
	require.NoError(t, db.Create(digit).Error)
	digitArt := ksv7904Category(t, db, "标签删除分类")
	art := ksv7904Article(t, svc, digitArt.ID)
	require.NoError(t, db.Create(&models.KnowledgeArticleTag{
		ArticleID: art.ID, TagID: "8801", CreatedAt: db.NowFunc(),
	}).Error)
	require.NoError(t, svc.DeleteTag(ctx, "8801"))
	var assocCount int64
	require.NoError(t, db.Model(&models.KnowledgeArticleTag{}).
		Where("tag_id = ?", "8801").Count(&assocCount).Error)
	assert.Equal(t, int64(0), assocCount, "DeleteTag 必须先清文章关联")

	// Q1 锁定:UUID 主键的 inline Delete 报错(不修)
	err = svc.DeleteTag(ctx, tag.ID)
	require.Error(t, err, "锁定现行为:UUID 主键的 DeleteTag inline Delete 报错(Q1)")
	var still int64
	require.NoError(t, db.Model(&models.KnowledgeTag{}).Where("id = ?", tag.ID).Count(&still).Error)
	assert.Equal(t, int64(1), still, "锁定现行为:报错时标签行不被删除")
}

// TestKsv7904_GetOrCreateTag_BothPaths 不存在 → 创建;已存在 → 返回既有行(不新建)。
// 本测试直调(无外层事务)所以两条路径都可走;Q2(跨 tx 连接)只在
// CreateKnowledgeArticle/UpdateKnowledgeArticle/AutoCreateTagsFromContent 的事务内
// 触发,彼时必须 pre-seed(见 ksv7904SeedTag),锁定不修。
func TestKsv7904_GetOrCreateTag_BothPaths(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)

	// 路径 1:不存在 → 创建
	created, err := svc.GetOrCreateTag(ctx, "或建标签")
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "或建标签", created.TagName)
	var count int64
	require.NoError(t, db.Model(&models.KnowledgeTag{}).
		Where("tag_name = ?", "或建标签").Count(&count).Error)
	assert.Equal(t, int64(1), count, "只应创建一行")

	// 路径 2:已存在 → 返回既有行
	again, err := svc.GetOrCreateTag(ctx, "或建标签")
	require.NoError(t, err)
	require.NotNil(t, again)
	assert.Equal(t, created.ID, again.ID, "必须复用既有行")
	require.NoError(t, db.Model(&models.KnowledgeTag{}).
		Where("tag_name = ?", "或建标签").Count(&count).Error)
	assert.Equal(t, int64(1), count, "不得重复创建")
}

// TestKsv7904_AutoCreateTagsFromContent 内容含 N 个标记 → 关联 N 标签并 +use_count;
// 重复调用幂等;无标记内容直接返回;Q2 锁定:标记先 pre-seed(事务内建标签必 BUSY)。
func TestKsv7904_AutoCreateTagsFromContent(t *testing.T) {
	ctx := context.Background()
	svc, db := newKsv7904(t)
	cat := ksv7904Category(t, db, "自动标签分类")
	art := ksv7904Article(t, svc, cat.ID)

	tagA := ksv7904SeedTag(t, db, "巡检")
	tagB := ksv7904SeedTag(t, db, "应急")
	_ = tagA
	_ = tagB

	content := "#巡检 与 #应急 的处理流程"
	require.NoError(t, svc.AutoCreateTagsFromContent(ctx, art.ID, content))

	assert.ElementsMatch(t, []string{tagA.ID, tagB.ID}, ksv7904AssocTagIDs(t, db, art.ID),
		"两个标记都必须关联到文章")
	counts := ksv7904TagCounts(t, db)
	assert.Equal(t, 1, counts[tagA.ID], "巡检 use_count +1")
	assert.Equal(t, 1, counts[tagB.ID], "应急 use_count +1")

	// 重复调用幂等:关联不重复,use_count 不再增长
	require.NoError(t, svc.AutoCreateTagsFromContent(ctx, art.ID, content))
	assert.ElementsMatch(t, []string{tagA.ID, tagB.ID}, ksv7904AssocTagIDs(t, db, art.ID))
	countsAfter := ksv7904TagCounts(t, db)
	assert.Equal(t, counts[tagA.ID], countsAfter[tagA.ID], "幂等:use_count 不再增长")

	// 无标记内容 → 解析为空 → 不开事务直接返回
	require.NoError(t, svc.AutoCreateTagsFromContent(ctx, art.ID, "没有标记的正文"))
	assert.ElementsMatch(t, []string{tagA.ID, tagB.ID}, ksv7904AssocTagIDs(t, db, art.ID))
}
