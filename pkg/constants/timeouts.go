package constants

import "time"

// timeouts.go — business timeout constants for network commands, LDAP, AD sync, and scheduler.
// All values match current runtime behavior (D-09: zero business behavior change).
// Uses time.Duration strong typing per Go standard library convention (net/http, database/sql, ldap).

const (
	// CommandExecTimeout is the maximum time allowed for a single device command execution.
	// Shared by command_handler.go:61 and execution_handler.go:104 (D-06).
	CommandExecTimeout = 300 * time.Second

	// CommandReadTimeout is the read timeout for quick command responses.
	// command_handler.go:98 (QuickCommand).
	CommandReadTimeout = 60 * time.Second

	// LDAPConnTimeout is the connection timeout for LDAP/AD servers.
	// ad_ldap_client.go:74.
	LDAPConnTimeout = 30 * time.Second

	// ADSyncTimeout is the context timeout for full AD synchronization tasks.
	// scheduler/ad_sync_tasks.go:42 (renamed from adSchedulerSyncTimeout, D-08).
	ADSyncTimeout = 30 * time.Minute

	// ADGroupSyncTimeout is the context timeout for a group-list AD sync cycle.
	// api/v1/system/ad_domain_handler.go (inline 30min/10min/2min extracted, v129-recheck WR-04).
	ADGroupSyncTimeout = 10 * time.Minute

	// ADSingleGroupSyncTimeout is the context timeout for syncing one AD group.
	ADSingleGroupSyncTimeout = 2 * time.Minute

	// SchedulerShutdownTimeout is the grace period for cron engine shutdown.
	// scheduler/cron.go:18 (renamed from defaultShutdownTimeout, D-08).
	SchedulerShutdownTimeout = 5 * time.Second

	// ADSyncTaskTimeout is the timeout for a single AD sync task (account pool recovery).
	// scheduler/ad_sync_tasks.go:164 (inline 1*time.Minute extracted, D-08).
	ADSyncTaskTimeout = 1 * time.Minute

	// RestoreBackupTimeout is the context timeout for the pre-restore backup phase
	// (CreateBackup in runRestore, Phase 97 V130R-01 D-01). Short TTL because
	// CreateBackup just reads the device config and stores it — typically <10s.
	RestoreBackupTimeout = 30 * time.Second

	// RestoreConfigExecTimeout is the context timeout for the RestoreConfig下发 phase
	// in runRestore (Phase 97 V130R-01 D-01). A full config can be hundreds of
	// lines sent line-by-line, far exceeding the single-command CommandExecTimeout.
	// Formerly RestoreConfigTimeout (10min); split into two independent budgets.
	RestoreConfigExecTimeout = 5 * time.Minute
)
