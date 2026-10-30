package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// BuildDeptRecursiveFilter returns a scope for "department + all sub-departments" filtering.
// Use this instead of duplicating the logic across building/workstation/infopoint/
// asset/server_room services.
//
// The ancestors column uses comma-separated format: ",dept1,dept2,dept3,"
// Match conditions (all four inherited from the 99-03 middle-of-hierarchy fix):
//   - idColumn = deptID (exact match)
//   - sys_dept.ancestors LIKE '%,<deptID>,%' (deptID in the middle of hierarchy)
//   - sys_dept.ancestors LIKE '%,<deptID' (deptID at the end of hierarchy)
//   - sys_dept.ancestors = <deptID> (deptID as root with no ancestors)
//
// Both the outer idColumn and the subquery output are wrapped in CAST(... AS TEXT):
// sys_dept.id is uuid while ops_* org columns are varchar, and PG has no implicit
// cast between the two (42883 operator does not exist). CAST keeps PG/SQLite
// dual-dialect parity (no-op on sqlite), matching the repo-wide uuid/varchar
// CAST convention (see workstation_service.go workstationJoinClause).
//
// Usage:
//
//	db.Scopes(BuildDeptRecursiveFilter(deptID, "org_id")) // Scopes style
//	sub := db.Session(&gorm.Session{NewDB: true}).Table(...)...
//	db.Where("EXISTS (?)", BuildDeptRecursiveFilter(deptID, "b.org_id")(sub)) // EXISTS subquery
func BuildDeptRecursiveFilter(deptID, idColumn string) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if deptID == "" {
			return db
		}
		return db.Where(
			"CAST("+idColumn+" AS TEXT) = ? OR CAST("+idColumn+" AS TEXT) IN (SELECT CAST(id AS TEXT) FROM sys_dept WHERE id = ? OR ancestors LIKE ? OR ancestors LIKE ? OR ancestors = ?)",
			deptID, deptID, "%,"+deptID+",%", "%,"+deptID, deptID,
		)
	}
}
