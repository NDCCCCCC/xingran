package component_collector

import "github.com/google/uuid"

// newUUID returns a fresh RFC 4122 v4 UUID string. Used by
// ReconciliationEmitter.Emit to populate sys_data_reconciliation.id on
// SQLite (the production PG schema defaults via gen_random_uuid()).
func newUUID() string {
	return uuid.New().String()
}
