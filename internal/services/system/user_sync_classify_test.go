package system

import (
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// classifyUser 是 AD 用户批量同步的核心分类逻辑（纯函数，不依赖 DB）。
// 这些用例锁定 4 类用户分类与优先级 ①>③>②>④，回归守护防止顺序误改。

func strPtrSyncTest(s string) *string { return &s }

func TestClassifyUser(t *testing.T) {
	activeAD := func(name string) *models.User {
		return &models.User{Username: name, ADUsername: strPtrSyncTest(name)}
	}
	localUser := func(name string) *models.User {
		return &models.User{Username: name, AuthSource: "local"}
	}
	softDeleted := func(name string) *models.User {
		return &models.User{Username: name, AuthSource: "local"}
	}

	tests := []struct {
		name              string
		adUsername        string
		byADUsername      map[string]*models.User
		activeLocalByName map[string]*models.User
		softDeletedByName map[string]*models.User
		wantCat           userCategory
		wantFound         bool
	}{
		{
			name:              "① 活跃AD用户(ad_username精确命中)",
			adUsername:        "zhang",
			byADUsername:      map[string]*models.User{"zhang": activeAD("zhang")},
			activeLocalByName: map[string]*models.User{},
			softDeletedByName: map[string]*models.User{},
			wantCat:           catActiveAD,
			wantFound:         true,
		},
		{
			name:              "③ 活跃local同名用户(对应 yanglong-013 场景)",
			adUsername:        "yanglong-013",
			byADUsername:      map[string]*models.User{},
			activeLocalByName: map[string]*models.User{"yanglong-013": localUser("yanglong-013")},
			softDeletedByName: map[string]*models.User{},
			wantCat:           catLocalSameName,
			wantFound:         true,
		},
		{
			name:              "② 软删除用户(username同名但deleted_at非空)",
			adUsername:        "deleted-user",
			byADUsername:      map[string]*models.User{},
			activeLocalByName: map[string]*models.User{},
			softDeletedByName: map[string]*models.User{"deleted-user": softDeleted("deleted-user")},
			wantCat:           catSoftDeleted,
			wantFound:         true,
		},
		{
			name:              "④ 全新用户(无任何命中)",
			adUsername:        "brand-new",
			byADUsername:      map[string]*models.User{},
			activeLocalByName: map[string]*models.User{},
			softDeletedByName: map[string]*models.User{},
			wantCat:           catNew,
			wantFound:         false,
		},
		{
			name:              "优先级边界: ad_username命中时,即使存在local同名,也取①活跃AD",
			adUsername:        "winner",
			byADUsername:      map[string]*models.User{"winner": activeAD("winner")},
			activeLocalByName: map[string]*models.User{"winner": localUser("winner")},
			softDeletedByName: map[string]*models.User{},
			wantCat:           catActiveAD,
			wantFound:         true,
		},
		{
			name:              "优先级边界: 同时local同名+软删除,取③local(③优先于②)",
			adUsername:        "both",
			byADUsername:      map[string]*models.User{},
			activeLocalByName: map[string]*models.User{"both": localUser("both")},
			softDeletedByName: map[string]*models.User{"both": softDeleted("both")},
			wantCat:           catLocalSameName,
			wantFound:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cat, u := classifyUser(tt.adUsername, tt.byADUsername, tt.activeLocalByName, tt.softDeletedByName)
			assert.Equal(t, tt.wantCat, cat, "分类结果不符")
			if tt.wantFound {
				assert.NotNil(t, u, "应返回命中的现有用户")
				assert.Equal(t, tt.adUsername, u.Username)
			} else {
				assert.Nil(t, u, "catNew 应返回 nil 用户")
			}
		})
	}
}

// TestClassifyUser_PriorityOrder 锁定优先级顺序 ①>③>②>④ 的数值语义。
// catActiveAD=0 < catLocalSameName=1 < catSoftDeleted=2 < catNew=3
func TestClassifyUser_PriorityOrder(t *testing.T) {
	assert.Equal(t, userCategory(0), catActiveAD)
	assert.Equal(t, userCategory(1), catLocalSameName)
	assert.Equal(t, userCategory(2), catSoftDeleted)
	assert.Equal(t, userCategory(3), catNew)
	assert.True(t, catActiveAD < catLocalSameName)
	assert.True(t, catLocalSameName < catSoftDeleted)
	assert.True(t, catSoftDeleted < catNew)
}
