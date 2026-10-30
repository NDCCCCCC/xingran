package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate210NormalizeMenuParentID 把 sys_menu 中 parent_id='0' 的脏行归一为 NULL,
// 修复 Q-11(normalizeParentID 双实现分歧导致 Update 路径落库字面 "0").
//
// 背景:
//   - menu_service.go 原 normalizeParentID 只塌缩 "",不处理 "0";
//   - requests/menu_requests.go 的 ToModel 已把 nil/""/"0" 全塌缩为 nil;
//   - 两者语义不一致,Update 携带 ParentID="0" 时落库字面 "0",buildMenuTree
//     找不到对应父节点,该菜单从树中丢失(孤儿节点)。
//
// 范围: 只 UPDATE parent_id 一个字段,仅当当前值为 '0' 时触发,幂等可重放。
// 双方言: 纯 GORM Update,无 PG 专有语法,SQLite 与 PostgreSQL 行为等价。
// 返回 changed = RowsAffected > 0,供调用方按需失效菜单缓存(与 Migrate209 风格一致)。
func Migrate210NormalizeMenuParentID(db *gorm.DB) (bool, error) {
	log.Println("Running migration 210: normalize sys_menu.parent_id='0' -> NULL")

	res := db.Model(&models.Menu{}).
		Where("parent_id = ?", "0").
		Update("parent_id", nil)
	if res.Error != nil {
		return false, fmt.Errorf("migration 210: parent_id='0' 归一失败: %w", res.Error)
	}

	changed := res.RowsAffected > 0
	if changed {
		applogger.Infof("[迁移] 210 已将 %d 条 sys_menu.parent_id='0' 归一为 NULL", res.RowsAffected)
	}
	return changed, nil
}
