package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// BuildDeptRecursiveFilter returns a scope for "department + all sub-departments" filtering.
// Use this instead of duplicating the logic across building/workstation/infopoint services.
//
// The ancestors column uses comma-separated format: ",dept1,dept2,dept3,"
// Match conditions:
//   - id = deptID (exact match)
//   - ancestors LIKE '%,<deptID>,%' (deptID in the middle of hierarchy)
//   - ancestors LIKE '%,<deptID' (deptID at the end of hierarchy)
//   - ancestors = <deptID> (deptID as root with no ancestors)
//
// Usage:
//   scope := BuildDeptRecursiveFilter(deptID, "dept_id")  // simple column
//   scope := BuildDeptRecursiveFilter(deptID, "ws.dept_id") // qualified column
func BuildDeptRecursiveFilter(deptID, idColumn string) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if deptID == "" {
			return db
		}
		return db.Where(
			idColumn+" = ? OR "+idColumn+" IN (SELECT id FROM sys_dept WHERE id = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors = ?)",
			deptID, deptID, "%,"+deptID+",%", "%,"+deptID, deptID,
		)
	}
}
