package system

import (
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// TestCacheKeyCollision verifies that buildListCacheKey does not produce
// identical keys for different parameter combinations that happen to contain
// the same colon-separated segments.
func TestCacheKeyCollision(t *testing.T) {
	svc := &userCacheService{}

	// Case 1: Username="bob:status:1" must NOT collide with Username="bob"+Status=1
	params1 := requests.UserListParams{
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}
	usernameBobStatus := "bob:status:1"
	params1.Username = &usernameBobStatus

	params2 := requests.UserListParams{
		Username: strPtr("bob"),
		Status:   intPtr(1),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}

	key1 := svc.buildListCacheKey(params1)
	key2 := svc.buildListCacheKey(params2)

	if key1 == key2 {
		t.Errorf("Cache key collision detected:\n  key1 (Username=bob:status:1): %s\n  key2 (Username=bob, Status=1): %s\n  expected different keys", key1, key2)
	}

	// Case 2: Multiple colons in username
	params3 := requests.UserListParams{
		Username: strPtr("a:b:c"),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}
	params4 := requests.UserListParams{
		Username: strPtr("a"),
		Status:   intPtr(1),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}
	params5 := requests.UserListParams{
		Username: strPtr("b"),
		Status:   intPtr(1),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}

	key3 := svc.buildListCacheKey(params3)
	key4 := svc.buildListCacheKey(params4)
	key5 := svc.buildListCacheKey(params5)

	if key3 == key4 {
		t.Errorf("Cache key collision: Username=a:b:c must not collide with Username=a, Status=1\n  got: %s", key3)
	}
	if key3 == key5 {
		t.Errorf("Cache key collision: Username=a:b:c must not collide with Username=b, Status=1\n  got: %s", key3)
	}
	if key4 == key5 {
		t.Errorf("Cache key collision: Username=a, Status=1 must not collide with Username=b, Status=1\n  got: %s", key4)
	}

	// Case 3: Empty values should still produce valid keys
	params6 := requests.UserListParams{
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}
	params7 := requests.UserListParams{
		Username: strPtr(""),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}

	key6 := svc.buildListCacheKey(params6)
	key7 := svc.buildListCacheKey(params7)

	if key6 != key7 {
		t.Logf("Note: empty username produces different key (acceptable):\n  key6: %s\n  key7: %s", key6, key7)
	}

	// Case 4: DeptID with colons
	params8 := requests.UserListParams{
		DeptID: strPtr("dept:a:b"),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}
	params9 := requests.UserListParams{
		DeptID: strPtr("dept:a"),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}
	params10 := requests.UserListParams{
		DeptID: strPtr("b"),
		BaseListRequest: base.BaseListRequest{
			OrderByColumn: "created_at",
			Current:       1,
			PageSize:      10,
		},
	}

	key8 := svc.buildListCacheKey(params8)
	key9 := svc.buildListCacheKey(params9)
	key10 := svc.buildListCacheKey(params10)

	if key8 == key9 {
		t.Errorf("Cache key collision: DeptID=dept:a:b must not collide with DeptID=dept:a\n  got: %s", key8)
	}
	if key8 == key10 {
		t.Errorf("Cache key collision: DeptID=dept:a:b must not collide with DeptID=b\n  got: %s", key8)
	}
}

func intPtr(i int) *int {
	return &i
}
