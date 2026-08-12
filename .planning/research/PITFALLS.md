# Pitfalls Research: v1.19 网络设备写命令 (Network Device Write Operations)

**Domain:** Network Device Management — SSH Write Operations (port config push)
**Researched:** 2026-07-06
**Confidence:** HIGH (existing codebase patterns verified) / MEDIUM (vendor-specific write quirks from prior knowledge)
**Milestone:** v1.19 — adding SSH write commands to extend the v1.18 "read" pipeline to "read+write" closure
**Scope:** NEW pitfall surface (write path) on TOP of the already-shipped v1.18 read path. v1.18 pitfalls (chassis SN, OPS chassis write, M196 trigger, M198 cron, etc.) are NOT re-researched.

---

## Executive Summary

Adding SSH write operations to a network management system is **not** a "use the same pool" exercise. Writes carry destructive semantics: a wrong command (`shutdown` on an uplink, `undo port-security` on a server farm port) can immediately cause user-visible outages, and the audit trail must be **recoverable** in a way that read pipelines never need. The v1.19 design is a fresh surface, but the underlying `DeviceConnectionPool` and `ScrapliWrapper` were validated for **read** concurrency — they were not validated for the much higher contention, session-lifetime, and error-rate shape of a write path.

The most critical pitfalls are:

1. **No transactional boundary between "command sent" and "device accepted"** — the SSH write returns success at the network level (`% Sent successfully`) but the device may reject the command at the next prompt, leaving the operator with a "succeeded but didn't" record.
2. **No pre/post state capture** — without reading the port state BEFORE issuing the write, the operlog only shows "the user wanted to disable GE0/0/1", not "the port WAS up, then user disabled it". Forensic investigations need both.
3. **Concurrent write collision** — two operators opening the same port description at the same time get `last-write-wins` semantics, with no warning. The connection pool's per-device lock serialises commands, but the port-level intent is not locked.
4. **Operlog masking leaks** — `description` body may legitimately contain words like "key", "password", or "secret" (e.g. "DMZ-port for key-service") that trip the sensitive-keyword filter and over-mask non-sensitive data. Worse: the masked command body hides what was actually sent, defeating audit purpose.
5. **Vendor command subtlety breaks silently** — Huawei vs H3C vs Ruijie have non-obvious differences (e.g. Ruijie doesn't have `system-view`, it has `configure terminal`; Huawei uses `interface GigabitEthernet0/0/1`, H3C accepts both short and long form; dot1x command keywords differ). A hardcoded template map per vendor is the locked decision, but each entry must be testable per-vendor.

The mitigations are concrete and specific — they involve new tables for `sys_port_write_audit` (with before/after snapshots), a new `network:port:write` permission namespace, structured result streaming for batches, and command-body masking rules that preserve audit readability.

---

## Critical Pitfalls

### Pitfall 1: SSH Write Succeeds at Transport Layer, Rejected at Device Layer

**What goes wrong:** Scrapli's `SendConfig` returns successfully if the SSH channel accepts the bytes. The device itself may immediately echo back `% Error: Unrecognized command found at '^' position.` and the next prompt is still the user-view prompt (not system-view). The operator sees `result.status = success` and the operlog records "操作成功", but the port is unchanged.

**Why it happens:**
- `ScrapliWrapper.SendConfig` (line 567-591 of `scrapli_wrapper.go`) returns `r.Result` text but does not parse for error markers (`%`, `Error:`, `^`, `Illegal`, `Wrong parameter`)
- Huawei VRP returns success on the channel write but rejects commands inline in the response
- The `acquireOp`/`releaseOp` lock guarantees connection safety but not command semantics
- Vendor config-mode lockout: if a previous batch left the device in system-view (mid-config), a new `system-view` command gets "Error: System view is locked by user X" instead of a fresh prompt

**Consequences:**
- Operlog shows "shutdown GE0/0/1 成功" but the port is still UP
- Cascading "retry until device returns the right error" loops that compound the audit record noise
- After 24h of being in a bad state, no one notices because the system says "succeeded"

**Prevention:**
```go
// ✅ CORRECT: Parse error markers from device response
func parseConfigError(result string) error {
    errMarkers := []string{
        "% Error:",
        "% Unrecognized command",
        "Error:",
        "Info: This operation may take a long time",
        "Warning:",
    }
    for _, m := range errMarkers {
        if strings.Contains(result, m) {
            // Strip the prompt tail and return the error line(s)
            return fmt.Errorf("设备拒绝命令: %s",
                strings.TrimSpace(result))
        }
    }
    return nil
}

// Use AFTER SendConfig returns nil
resp, err := wrapper.SendConfig("interface GigabitEthernet0/0/1\nshutdown")
if err != nil {
    return WriteResult{Status: "transport_error", Err: err}
}
if devErr := parseConfigError(resp.Result); devErr != nil {
    return WriteResult{Status: "device_rejected", Err: devErr, Raw: resp.Result}
}
```

**Warning signs:**
- Operlog shows N consecutive successes but the v1.18 collector (next enqueue) reports the port state didn't change
- Syslog/sys_oper_log row contains `%` in the response field — flag in metrics

**Phase to address:** Phase 50 (Write Command Service) — `internal/services/network_write/parser.go`

**Code references:**
- `internal/device/scrapli_wrapper.go:567` (SendConfig returns response, no error parsing)
- `internal/services/portcollection/collection.go:1-80` (read-only use of same wrapper, never sees this problem)

---

### Pitfall 2: No Before/After State Snapshot for Audit

**What goes wrong:** Operlog shows "用户A 在 2026-07-06 14:23:01 修改端口 GE0/0/1 description 为 'Core-Switch-Uplink'". If 6 months later an operator needs to know what the description WAS before, the only evidence is the live device state — which has been overwritten by 1000 subsequent writes. The audit trail is write-only.

**Why it happens:**
- v1.18's collector does NOT store historical port state (no `sys_port_state_history` table)
- Operlog's `oper_param` is the request body (the new value), not the current state
- `sys_oper_log` does not have a column for "before_state" — adding it requires schema migration
- The existing `sys_device_port_status` is a snapshot table that gets overwritten on each collection

**Consequences:**
- Failed forensic investigation: "why was this port renamed in March?"
- Compliance audit gaps: many regulations require "what was the state at the time of the change"
- Inability to "revert" — the previous value is lost the moment the new value is written

**Prevention:**
```go
// ✅ CORRECT: Read port state BEFORE writing, persist both in audit table
type PortWriteAudit struct {
    ID            string    `gorm:"type:uuid;primary_key"`
    DeviceID      string    `gorm:"type:uuid;not null;index"`
    InterfaceName string    `gorm:"size:64;not null;index"`
    Operation     string    `gorm:"size:32;not null"`         // shutdown/description/dot1x
    BeforeValue   string    `gorm:"type:text"`                // captured state BEFORE write
    AfterValue    string    `gorm:"type:text"`                // captured state AFTER write (from next collect)
    CommandSent   string    `gorm:"type:text"`                // exact SSH command bytes
    DeviceResponse string   `gorm:"type:text"`                // raw device response
    Status        string    `gorm:"size:32;not null;index"`   // transport_error/device_rejected/success
    ErrorMessage  string    `gorm:"type:text"`
    OperatorID    string    `gorm:"type:uuid;index"`
    OperatorName  string    `gorm:"size:64"`
    ClientIP      string    `gorm:"size:64"`
    StartedAt     time.Time `gorm:"not null;index"`
    CompletedAt   *time.Time
    DurationMs    int64
}

// In the write service:
func (s *WriteService) WriteOnePort(ctx context.Context, req *WriteRequest) (*PortWriteAudit, error) {
    audit := &PortWriteAudit{
        DeviceID:      req.DeviceID,
        InterfaceName: req.InterfaceName,
        Operation:     req.Operation,
        OperatorID:    req.OperatorID,
        StartedAt:     time.Now(),
    }
    // 1. Capture BEFORE state via v1.18 collector's query path
    if before, err := s.queryPortState(ctx, req.DeviceID, req.InterfaceName); err == nil {
        audit.BeforeValue = before
    }
    // 2. Issue the write
    resp, devErr := s.issueWrite(ctx, req)
    if devErr != nil {
        audit.Status = "device_rejected"
        audit.ErrorMessage = devErr.Error()
    } else {
        audit.Status = "success"
    }
    audit.CommandSent = req.RenderCommand()
    audit.DeviceResponse = resp.Result
    // 3. Persist audit (even on failure — that's the whole point)
    s.db.Create(audit)
    return audit, devErr
}
```

**Warning signs:**
- Incident review reveals the audit table only contains "what was requested", not "what was there before"
- Cannot answer "did this change cause the outage?" without re-reading the device

**Phase to address:** Phase 50 — `sys_port_write_audit` migration + write service integration

**Code references:**
- `internal/services/portcollection/query.go` (existing read query, can be reused for before-state)
- `internal/utils/operlog/operlog.go:215-263` (operlog.Record is write-only; no before-state field)

---

### Pitfall 3: Vendor Command Syntax Subtlety Breaks Silently

**What goes wrong:** Huawei and H3C share a VRP-derived CLI but H3C accepts the shorthand `interface GE0/0/1` while Huawei only accepts `interface GigabitEthernet0/0/1` (some firmware). Ruijie uses Cisco-style `configure terminal` and `interface GigabitEthernet 0/0/1` (with a SPACE, not slash). A hardcoded `vendor: command` map gets one entry wrong and the wrong device silently does nothing.

**Why it happens:**
- Different vendors parse the same words differently
- Even within a vendor, firmware version matters: VRP V200 vs V600 differ in some keywords
- The `GetCommandForVendor` function in `scrapli_wrapper.go:682-728` is read-only and doesn't have an equivalent write map yet
- The `huawei_vrp.yaml` / `hp_comware.yaml` / `ruijie_rjos.yaml` platform definitions in scrapligo handle the READ path's prompt handling but do NOT inject write-mode commands

**Consequences:**
- Wrong device reads `interface GigabitEthernet 0/0/1` as "create VLAN 0/0/1" on Ruijie (if the string is mishandled)
- `shutdown` on Huawei VRP uses no `undo`, while Cisco/IOS uses `no shutdown` — same intent, opposite keyword
- H3C's `dot1x port-control` vs Huawei's `dot1x port-control auto` — same effect, but the helper function name differs by vendor

**Prevention:**
```go
// ✅ CORRECT: Vendor-specific command template registry
type WriteOp struct {
    Operation string  // "shutdown", "description", "dot1x"
    HuaweiCmd func(iface, value string) []string
    H3CCmd    func(iface, value string) []string
    RuijieCmd func(iface, value string) []string
}

var writeTemplates = map[string]WriteOp{
    "shutdown": {
        HuaweiCmd: func(i, _ string) []string {
            return []string{"system-view", "interface " + i, "shutdown"}
        },
        H3CCmd: func(i, _ string) []string {
            return []string{"system-view", "interface " + i, "shutdown"}
        },
        // Ruijie: Cisco-style, no system-view
        RuijieCmd: func(i, _ string) []string {
            return []string{"configure terminal", "interface GigabitEthernet " + i, "shutdown", "end", "write"}
        },
    },
    "description": {
        HuaweiCmd: func(i, v string) []string {
            return []string{"system-view", "interface " + i, "description " + v, "commit", "quit", "quit"}
        },
        // H3C: accepts both forms; long form is safer
        H3CCmd: func(i, v string) []string {
            return []string{"system-view", "interface GigabitEthernet" + strings.TrimPrefix(i, "GE"), "description " + v, "quit", "quit"}
        },
        // Ruijie: NO commit (auto-save), but requires "end" to apply
        RuijieCmd: func(i, v string) []string {
            return []string{"configure terminal", "interface GigabitEthernet " + i, "description " + v, "end"}
        },
    },
    "dot1x_enable": {
        HuaweiCmd: func(i, _ string) []string {
            return []string{"system-view", "interface " + i, "dot1x enable", "dot1x port-control auto", "quit", "quit"}
        },
        H3CCmd: func(i, _ string) []string {
            return []string{"system-view", "interface GigabitEthernet" + strings.TrimPrefix(i, "GE"), "dot1x", "dot1x port-control auto", "quit", "quit"}
        },
        RuijieCmd: func(i, _ string) []string {
            return []string{"configure terminal", "interface GigabitEthernet " + i, "dot1x port-control auto", "end"}
        },
    },
}
```

**Warning signs:**
- Same operation on identical port names produces different results across devices
- Operlog `CommandSent` field shows "configure terminal" on Huawei (H3C-ism leak)
- Vendor mapping errors only surface during UAT on real devices, not in unit tests

**Phase to address:** Phase 50 — `internal/services/network_write/templates.go` with full per-vendor coverage

**Code references:**
- `internal/device/scrapli_wrapper.go:682-728` (read-only `GetCommandForVendor` template pattern to mirror)
- `internal/services/device_info_collection_service.go:352-367` (per-vendor command list pattern)

---

### Pitfall 4: Concurrent Write Collision — Two Operators, Same Port, Last-Write-Wins

**What goes wrong:** Operator A is mid-batch on `GE0/0/1-24` (changing descriptions). Operator B opens the same `GE0/0/1` description in a single-port modal. Both writes succeed at the transport layer, but the audit trail shows two successful writes with the second's `CommandSent` value winning. No one sees that they overwrote each other.

**Why it happens:**
- `DeviceConnectionPool` (`connection_pool.go`) has per-device locks — but only at the SSH session level, not the port-intent level
- The `acquireOp` lock is held for the duration of ONE command sequence (system-view → interface X → description), not the whole port write
- Two operators' transactions can interleave between commands (Operator A is at `interface GE0/0/1`, Operator B is at `interface GE0/0/2`) — they don't actually collide on the device, but their audit trail is now ambiguous about which was first
- No `sys_port_lock` table exists for "operator X is currently editing this port"

**Consequences:**
- Audit ambiguity: "who changed GE0/0/1 description at 14:23?"
- Last-write-wins: if A's "uplink" is overwritten by B's "server-farm" mid-batch, A's audit row shows "success" but the actual state matches B's intent
- No conflict resolution: there is no "your changes conflict with another operator's session" UI

**Prevention:**
```go
// ✅ CORRECT: Port-level intent lock via DB table (advisory, not mutex)
type PortWriteLock struct {
    DeviceID      string    `gorm:"type:uuid;primaryKey"`
    InterfaceName string    `gorm:"size:64;primaryKey"`
    OperatorID    string    `gorm:"type:uuid;not null"`
    OperatorName  string    `gorm:"size:64"`
    Operation     string    `gorm:"size:32;not null"`  // shutdown/description/dot1x/batch
    AcquiredAt    time.Time `gorm:"not null"`
    ExpiresAt     time.Time `gorm:"not null;index"`    // TTL: 5min default
}

func (s *WriteService) AcquirePortLock(ctx context.Context, deviceID, iface, op, operatorID, opName string) (*PortWriteLock, error) {
    now := time.Now()
    lock := &PortWriteLock{
        DeviceID:      deviceID,
        InterfaceName: iface,
        Operation:     op,
        OperatorID:    operatorID,
        OperatorName:  opName,
        AcquiredAt:    now,
        ExpiresAt:     now.Add(5 * time.Minute),
    }
    // Upsert with WHERE expires_at < now (steal expired locks)
    result := s.db.Clauses(clause.OnConflict{
        Columns: []clause.Column{{Name: "device_id"}, {Name: "interface_name"}},
        DoUpdates: clause.Assignments(map[string]interface{}{
            "operator_id":   operatorID,
            "operator_name": opName,
            "operation":     op,
            "acquired_at":   now,
            "expires_at":    now.Add(5 * time.Minute),
        }),
    }).Create(lock)
    if result.Error != nil {
        return nil, result.Error
    }
    // Return current holder (may not be us if not expired)
    var current PortWriteLock
    s.db.Where("device_id = ? AND interface_name = ?", deviceID, iface).First(&current)
    if current.OperatorID != operatorID && current.ExpiresAt.After(now) {
        return &current, fmt.Errorf("端口 %s 正在被 %s 编辑中(操作: %s, 剩余 %v)",
            iface, current.OperatorName, current.Operation,
            current.ExpiresAt.Sub(now).Round(time.Second))
    }
    return &current, nil
}
```

**Warning signs:**
- Operlog shows two "success" rows for the same `device_id+interface_name+operation` within 5 minutes
- No "conflict" or "denied" rows in the audit table (all writes succeed)
- User reports "my change didn't apply" but audit shows success

**Phase to address:** Phase 50 — `sys_port_write_lock` table + `AcquiredBy` in audit row

**Code references:**
- `internal/device/connection_pool.go:159-318` (existing per-device SSH lock, not port-aware)
- `internal/services/addomain/account_pool.go` (existing `SELECT FOR UPDATE` row-lock pattern to mirror)

---

### Pitfall 5: Operlog Sensitive-Keyword Masking Hides the Audit Value

**What goes wrong:** The 11 mandatory sensitive keywords (`password`, `pwd`, `secret`, `token`, `key`, `salt`, `privateKey`, `oldPassword`, `macKey`, `sm4Key`, `sm2Key`, etc. — see `internal/utils/operlog/operlog.go:135-170`) match by contiguous-substring on `"<key>":"<value>"` JSON. A legitimate port description like `DMZ-port-for-key-service` or `HR-system-api-token-endpoint` will have its VALUE masked to `******`, even though the field is a non-sensitive description. Worse: if the user enters a description that contains the literal text `password X` (e.g. "Reset password hint"), the entire description is masked, leaving the audit log useless.

**Why it happens:**
- The `FilterSensitiveParams` function masks by JSON key name, not by semantic field
- Operlog's `RecordWithBody` reads the request body once, filters, and stores — there's no per-field policy
- The 11 keywords were designed for API key / credential bodies, not for free-form network config commands
- The locked `regression_test.go:188-207` "mandatorySensitiveKeywords" test prevents adding exceptions to the keyword list

**Consequences:**
- Audit log shows `description: ******` when the operator typed `DMZ-port-for-key-service` — no record of what was actually written
- Compliance review finds the operlog "redacts too much" and either disables masking (regression) or rejects the audit trail entirely
- Forensic investigation cannot determine "what was the description before?" because the AfterValue is also masked

**Prevention:**
```go
// ✅ CORRECT: Use dedicated, unmasked audit table for write operations
// Don't rely on operlog.Record for port write details — use sys_port_write_audit instead.
// Operlog still records "操作类型 + 设备 + 端口" but the actual command body
// lives in sys_port_write_audit (Pitfall 2) where it can be selectively redacted
// (only the password/secret VALUES if a write operation actually contains them,
// which shutdown/description/dot1x never do for port config commands).

// For commands that DO contain credentials (e.g. setting SNMP community string):
func maskCommandBody(cmd string) string {
    // Pattern: only mask known sensitive PATTERNS, not free text
    sensitivePatterns := []struct{ re *regexp.Regexp }{
        {regexp.MustCompile(`(?i)snmp-agent\s+community\s+read\s+(\S+)`)},
        {regexp.MustCompile(`(?i)snmp-server\s+community\s+(\S+)`)},
        {regexp.MustCompile(`(?i)(local-user|user)\s+\S+\s+password\s+(?:cipher\s+)?(\S+)`)},
    }
    masked := cmd
    for _, p := range sensitivePatterns {
        masked = p.re.ReplaceAllString(masked, "$1******")
    }
    return masked
}

// In the write service:
audit.CommandSent = maskCommandBody(req.RenderCommand())  // masked only when needed
audit.RawCommandSent = req.RenderCommand()                // full text, for admin redaction
```

**Warning signs:**
- Operlog shows `description: ******` for a port whose actual config has a non-sensitive description
- Audit log volume drops by 90% after v1.19 ships (over-masking broke operator trust in the log)
- User complains "I can see the operlog row but not what was actually written"

**Phase to address:** Phase 50 — `sys_port_write_audit.CommandSent` is a separate column with targeted regex masking; operlog is used for the high-level "user X changed port Y" entry only

**Code references:**
- `internal/utils/operlog/operlog.go:135-170` (the 11 mandatory sensitive keywords)
- `internal/utils/operlog/regression_test.go:188-207` (locked keyword set, no exceptions possible)
- `internal/api/v1/network/command_handler.go:74` (existing read-only QuickCommand uses operlog for audit)

---

### Pitfall 6: Batch Execution Exceeds 30s Core Shutdown Deadline

**What goes wrong:** A batch of 50 ports × 3 SSH commands each = 150 commands × ~2s each = 5 minutes total. The Gin handler's `c.Request.Context()` is bound to the HTTP request lifetime, but the connection pool's `cleanupTicker` (1-minute interval in `connection_pool.go:459`) can race the batch. More importantly, the `Core.Close()` 30s shutdown deadline (`core.go:488`) can fire mid-batch — the connection pool is forcibly closed while a write is still issuing, leaving the device in a half-configured state (e.g. description was set but `quit` was never sent, so the operator is "still in system-view").

**Why it happens:**
- `c.Request.Context()` is the standard Gin pattern but has no extension mechanism
- SSH `SendConfig` calls don't carry a sub-context with longer timeout
- The 30s `coreShutdownTimeout` in `core.go:488` is non-negotiable (process must exit)
- Per-port timeouts in `portCollectionDeviceTimeout = 10*time.Minute` are read-only — they don't apply to writes
- The "write failed because the connection was closed mid-config" is silently swallowed by `acquireOp`'s `state != StateReady` check

**Consequences:**
- Half-configured devices: GE0/0/1 description is set, but `quit` was never sent → next operator session on the device starts in system-view (not user-view) → next command "succeeds" but is operating in a weird context
- Operlog shows "成功" but the device state is partial
- On service restart, all in-flight batches are abandoned with no operator notification

**Prevention:**
```go
// ✅ CORRECT: Detach batch execution from HTTP request lifetime
// Use a separate background context with a longer deadline (e.g. 30min),
// but still keep operlog.Record in the HTTP handler so the operator's
// request is tied to the audit row.

func (h *PortWriteHandler) BatchWrite(c *gin.Context) {
    var req BatchWriteRequest
    if !responseHelpers.HandleJSONBinding(c, &req) { return }

    // Cap batch size: 50 ports max (Pitfall 11)
    if len(req.Ports) > 50 {
        response.Error(c, http.StatusBadRequest, "单次批量操作最多 50 个端口")
        return
    }

    // Detached context with longer deadline, NOT c.Request.Context()
    batchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()

    // Submit to the same DeviceInfoCollectionService.enqueue-style background queue
    batchID := uuid.New().String()
    go s.writeQueue.Submit(batchCtx, batchID, &req)

    // Return batch ID immediately, operator polls for progress
    response.Success(c, gin.H{
        "batchId": batchID,
        "status":  "queued",
        "totalPorts": len(req.Ports),
    })
}

// Poll endpoint for progress:
func (h *PortWriteHandler) GetBatchStatus(c *gin.Context) {
    batchID := c.Param("batchId")
    progress, _ := s.writeQueue.GetProgress(batchID)
    response.Success(c, progress)
}
```

**Warning signs:**
- `Core.Close` logs `[Core.Close] 已超过 30s 强制结束` during or after a batch
- Operlog shows 50 rows of "成功" but a device audit trail shows only 30 commands landed
- Next v1.18 collection shows port state inconsistent with operlog "success" rows

**Phase to address:** Phase 50 — Detach batch context, introduce `sys_port_write_batch` (progress tracking table)

**Code references:**
- `internal/core/core.go:488-520` (30s shutdown timeout, hard cap)
- `internal/services/device_info_collection_service.go:75-130` (existing background worker pattern with detached context)
- `internal/services/device_info_collection_service.go:101-129` (Stop() with timeout — pattern to mirror)

---

### Pitfall 7: Connection Pool Exhaustion Holding the Pool for the Whole Batch

**What goes wrong:** A 50-port batch holds the SSH connection to the device for 5 minutes. The pool's `maxConnections=50` (in `connection_pool.go:184`) means a single batch can hold 1/50 of the pool for 5 minutes. If 3 operators each run a 50-port batch on different devices simultaneously, 3/50 of the pool is held. If a v1.18 collection cron triggers at the same time, the collector's `Enqueue` (in `device_info_collection_service.go:133-163`) returns "任务队列已满" (queue full at 1000 entries) but the underlying SSH pool is also blocked.

**Why it happens:**
- The connection pool locks are per-device, but the pool itself has a global count
- PooledConnection.Acquire (line 45-75) holds the device lock for the duration of `Execute`'s callback
- Long-running batches lock the connection even when no command is in flight (between iterations)
- The pool's `cleanupTicker` skips connections with `refCount > 0`, so a held batch prevents the connection from being cleaned up
- `maxConnections=50` was set when only the v1.18 collector used it (5 workers × N devices) — not sized for concurrent write batches

**Consequences:**
- "连接池已满" errors when concurrent batches + cron collision
- v1.18 collector cron fails to collect from devices whose connections are held by an in-flight batch
- Frontend shows "批量写入中" forever if the pool's acquire fails — no recovery path

**Prevention:**
```go
// ✅ CORRECT: Release the connection between commands, not across the batch
func (s *WriteService) WriteBatch(ctx context.Context, req *BatchWriteRequest) error {
    for i, port := range req.Ports {
        // Check context cancellation BETWEEN ports (not just at the top)
        if err := ctx.Err(); err != nil {
            return fmt.Errorf("batch cancelled at port %d: %w", i, err)
        }
        // Per-port write uses its OWN Acquire/Release cycle
        if err := s.writeOnePort(ctx, req.DeviceID, port); err != nil {
            // Failed: continue to next port, log this one's failure
            s.recordPortFailure(i, port, err)
            continue
        }
    }
    return nil
}

func (s *WriteService) writeOnePort(ctx context.Context, deviceID string, port PortRef) error {
    conn, err := s.pool.GetConnection(ctx, deviceID)
    if err != nil {
        return err
    }
    defer conn.ReleaseRef()  // Release after THIS port's commands, not after the batch

    wrapper := conn.GetWrapper()
    // Per-port context with 30s timeout
    portCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    return s.issueWrite(portCtx, wrapper, port)
}
```

**Warning signs:**
- `GetStats()` reports `active_connections` close to `max_connections` (50)
- v1.18 cron `EnqueueAllOnlineDevices` log shows "加入队列失败"
- Operlog for write operations shows latency > 30s

**Phase to address:** Phase 50 — `writeOnePort` boundary; Phase 51 — pool size config (move to `sys_config`)

**Code references:**
- `internal/device/connection_pool.go:184-208` (DefaultPoolConfig, hardcoded 50)
- `internal/device/connection_pool.go:159-318` (GetConnection returns pc with refCount=1; cleanup skipped)
- `internal/services/device_info_collection_service.go:133-163` (existing enqueue with full-queue rejection)

---

### Pitfall 8: Frontend Optimistic Update Race vs Slow Batch Result

**What goes wrong:** Operator clicks "批量关闭 GE0/0/1-24" with optimistic UI update (port shows red/disabled immediately). The batch takes 90 seconds. The actual device result is 23 success + 1 failure (GE0/0/17 is "stuck" in err-disabled state and won't accept shutdown). The optimistic UI shows all 24 as "已禁用" for 90s, then the partial result arrives and 1 of them flips back to "实际状态: enabled". The operator may have already left the page or made decisions based on the wrong state.

**Why it happens:**
- React Query's optimistic update pattern (`onMutate` + `onError` rollback) is per-mutation, not per-batch
- TanStack React Query doesn't have native batched mutation with partial rollback
- The existing `port_handler.go:96-111` Collect endpoint is per-device (synchronous), not batched
- Long-running batch results arrive AFTER the operator's click, not before the operator's next action

**Consequences:**
- Operator makes a follow-up decision ("I disabled it, let me also disable GE0/0/25") based on stale optimistic state
- The rollback (1 port going from "disabled" back to "enabled") is hidden under the batch progress overlay
- If the operator navigates away, the partial result is lost (state in React Query cache is the only source of truth)

**Prevention:**
```typescript
// ✅ CORRECT: Use a batch progress dialog state, not optimistic row updates
// In the ports list page:
const [batchProgress, setBatchProgress] = useState<BatchProgress | null>(null);

const batchWriteMutation = useMutation({
  mutationFn: (req: BatchWriteRequest) => post('/network/port/write-batch', req),
  onSuccess: (res) => {
    setBatchProgress({ batchId: res.data.batchId, totalPorts: res.data.totalPorts, completed: 0 });
    // Do NOT optimistically update rows — show a dialog instead
  },
});

useEffect(() => {
  if (!batchProgress) return;
  const poll = setInterval(async () => {
    const res = await post('/network/port/write-batch/status', { batchId: batchProgress.batchId });
    setBatchProgress(res.data);
    if (res.data.status === 'completed' || res.data.status === 'failed') {
      clearInterval(poll);
      // NOW invalidate the ports query to refetch real state
      queryClient.invalidateQueries(['ports', res.data.deviceId]);
    }
  }, 2000);
  return () => clearInterval(poll);
}, [batchProgress]);
```

**Warning signs:**
- Operlog and visible port state diverge (UI says "disabled", operlog says "failed")
- React Query DevTools shows `isStale: true` for ports while batch is in-flight
- `port_handler.go` List endpoint returns "enabled" for a port that the batch said was "successfully disabled"

**Phase to address:** Phase 52 (Frontend) — BatchProgressDialog component with poll-based progress, no row-level optimistic updates

**Code references:**
- `xingran-react-frontend/src/lib/opsApi.ts` (existing opsApi pattern for batch operations)
- `xingran-react-frontend/src/pages/network/port/index.tsx` (to be created in v1.19)

---

### Pitfall 9: Operlog.RecordBackground Required for Post-Write Enqueue Audit

**What goes wrong:** After a write succeeds, v1.19 enqueues a v1.18 collection (via `DeviceInfoCollectionService.Enqueue`) to refresh the port state. The collection runs in a background worker, NOT in the HTTP request context. The collection's pipeline writes to `sys_oper_log` via `OpsAssetWriter` (per Phase 48 D-13). If the writer uses the regular `operlog.Record` (which reads `*gin.Context` for operator name), the row is dropped silently (defensive `if c == nil` in `operlog.go:229`). The audit trail of "port state was refreshed after write" is missing.

**Why it happens:**
- Phase 48 D-13 explicitly introduced `operlog.RecordBackground` for this exact reason — `operlog.go:265-321` — but the documentation in `operlog.go:280-281` notes "D-13 scope is the UPDATE path only. Failure-path events from cron pipelines use applogger.Warnf"
- The post-write enqueue is a NEW caller that needs `RecordBackground` but might be wired with `Record` (using the now-stale request context that returns nil)
- The `c.Request.Context()` of a completed write handler is already Done — passing it to a background worker triggers "context canceled" immediately

**Consequences:**
- `sys_oper_log` is missing rows for the post-write collection step
- Compliance audit shows the write, but not the verification step
- Debugging "why is the port state stale?" requires correlating to `applogger` (debug) output, not `sys_oper_log`

**Prevention:**
```go
// ✅ CORRECT: Use RecordBackground for the post-write enqueue's audit
func (s *WriteService) EnqueuePostWriteCollection(ctx context.Context, deviceID string, writeAuditID string) error {
    // Enqueue the collection (v1.18 path)
    if err := s.deviceInfoService.Enqueue(deviceID); err != nil {
        applogger.Warnf("[端口写入] 触发改后采集失败 [deviceID=%s, writeAuditID=%s]: %v",
            deviceID, writeAuditID, err)
        return err
    }
    // Audit the enqueue event using RecordBackground (no gin.Context)
    operlog.RecordBackground(
        s.core.OperLogService,
        s.core.GetDB(),
        "端口写入",
        operlog.OperTypeSync,  // Sync = 改后同步采集
        "system-port-write",   // operator name sentinel
        map[string]interface{}{
            "deviceId":   deviceID,
            "writeAuditId": writeAuditID,
            "trigger":    "post_write_collection",
        },
    )
    return nil
}
```

**Warning signs:**
- `sys_oper_log` shows the write row but no following "改后采集" row
- Search by `device_id + operUrl contains /network/port/write` returns 1 row, not 2
- Phase 48 D-13 documentation in `.planning/notes/` is followed but the new caller forgot to use `RecordBackground`

**Phase to address:** Phase 50 — `EnqueuePostWriteCollection` service method with `RecordBackground`

**Code references:**
- `internal/utils/operlog/operlog.go:265-321` (RecordBackground signature)
- `internal/services/device_info_collection_service.go:265-300` (processTask path, no gin.Context)
- `internal/services/component_collector/ops_asset_writer.go` (Phase 48 D-13 RecordBackground example)

---

### Pitfall 10: Device Unreachable Mid-Batch — Half-Executed, No Reconnect

**What goes wrong:** Batch starts, ports 1-15 succeed, port 16's SSH command times out (device rebooted, network blip, or device CPU spiked). The connection pool marks the wrapper as `StateClosed` (per `scrapli_wrapper.go:526-530` `containsEOF`/`containsConnectionError` check). The batch continues to port 17, calls `pool.GetConnection()`, gets a NEW connection (the old one is gone), and the new connection works — but the device might still be rebooting, and the new command silently enters user-view (not system-view) on the rebooting device, producing a `% Unknown command` for `system-view` (because the device is in pre-boot state).

**Why it happens:**
- `ScrapliWrapper.SendConfig` does not retry on connection error
- `DeviceConnectionPool.removeConnection` (line 427-455) is called only when `IsReady()` returns false in `GetConnection` — but if the connection WAS ready and then broke mid-command, the pool doesn't proactively remove it
- The 30s `core.go:469` (refresh view) and similar 30s timeouts are for background tasks, not the write path's per-command timeout
- No "post-error reconnect" logic — the next port's `GetConnection` may or may not get a fresh connection depending on whether the cleanup ticker has fired

**Consequences:**
- Mid-batch state corruption: ports 16-50 are written against a device in an unknown state
- Operlog shows mixed success/failure with no operator-friendly "device unreachable at port 16" message
- Operator has to manually investigate each port's state (no easy "re-run the failed portion only")

**Prevention:**
```go
// ✅ CORRECT: Proactive health check + per-port reconnect + batch resume
func (s *WriteService) WriteBatchWithResume(ctx context.Context, batchID string, req *BatchWriteRequest) {
    var failedPorts []PortRef
    for i, port := range req.Ports {
        if ctx.Err() != nil { return }

        // Capture pre-port health check
        if err := s.healthCheckDevice(ctx, req.DeviceID); err != nil {
            applogger.Warnf("[批量写入] 设备健康检查失败 [deviceID=%s, port=%s, 已完成=%d/%d]: %v",
                req.DeviceID, port.InterfaceName, i, len(req.Ports), err)
            failedPorts = append(failedPorts, port)
            s.updateBatchProgress(batchID, i+1, "device_unreachable")
            continue
        }

        if err := s.writeOnePort(ctx, req.DeviceID, port); err != nil {
            // Distinguish: device_unreachable (retry next batch) vs device_rejected (don't retry)
            if isDeviceUnreachable(err) {
                failedPorts = append(failedPorts, port)
                s.updateBatchProgress(batchID, i+1, "device_unreachable")
                // Wait 5s before next attempt to let device recover
                time.Sleep(5 * time.Second)
            } else {
                s.recordPortFailure(i, port, err)
                s.updateBatchProgress(batchID, i+1, "device_rejected")
            }
            continue
        }
        s.updateBatchProgress(batchID, i+1, "success")
    }
    // Resume endpoint can re-run failedPorts
    s.markBatchCompleted(batchID, failedPorts)
}
```

**Warning signs:**
- Operlog shows a sequence like `[success × 15, device_rejected, success × 34]` with no obvious boundary
- Port state on device doesn't match operlog "success" rows
- v1.18 collection after the batch shows half the ports have new state, half have old state

**Phase to address:** Phase 50 — health check + resume support; Phase 51 — `failed_ports` JSON column in `sys_port_write_batch`

**Code references:**
- `internal/device/scrapli_wrapper.go:526-530` (EOF/connection error → state=Closed, no proactive removal)
- `internal/device/connection_pool.go:427-455` (removeConnection requires explicit IsReady=false check)

---

### Pitfall 11: No Batch Size Cap — Operator Initiates 1000-Port Batch, Pool Starves

**What goes wrong:** No MVP design constraint is placed on batch size. An operator selects 1000 ports (e.g. "disable all unused ports on this device"). The batch holds the connection for 1000 × 2s = 33 minutes, exceeding the 30-minute detached context deadline. The pool's maxConnections=50 is overwhelmed by similar concurrent batches, and v1.18 collection cron starts failing.

**Why it happens:**
- v1.19 scope says "批量执行：串行失败即停" but doesn't specify a cap
- Frontend's Ant Design Table row selection supports unlimited selection by default
- No per-operator or per-device rate limit exists at the handler layer
- The pool's `maxConnections=50` is a soft cap, not a per-batch limit

**Consequences:**
- Batch exceeds 30min context, the whole batch is abandoned with partial results
- Concurrent batches from different operators all share the pool's 50-connection budget
- v1.18 collection cron fails to schedule new tasks (queue full from operator batches)

**Prevention:**
```go
// ✅ CORRECT: Hard cap + soft warning + automatic chunking
const (
    maxBatchSizeHardCap     = 50    // absolute limit per batch
    maxBatchSizeSoftWarning = 20    // warn operator if > 20 ports
    maxConcurrentBatches    = 5     // per-operator concurrent limit
)

// In the handler:
func (h *PortWriteHandler) BatchWrite(c *gin.Context) {
    var req BatchWriteRequest
    if !responseHelpers.HandleJSONBinding(c, &req) { return }

    if len(req.Ports) == 0 {
        response.Error(c, http.StatusBadRequest, "请选择至少一个端口")
        return
    }
    if len(req.Ports) > maxBatchSizeHardCap {
        response.Error(c, http.StatusBadRequest,
            fmt.Sprintf("单次批量操作最多 %d 个端口(已选 %d 个),请分批执行",
                maxBatchSizeHardCap, len(req.Ports)))
        return
    }
    if len(req.Ports) > maxBatchSizeSoftWarning {
        // Return a warning header but proceed
        c.Header("X-Batch-Warning",
            fmt.Sprintf("批量操作 %d 个端口,建议分批执行", len(req.Ports)))
    }
    // ... continue
}
```

**Warning signs:**
- `sys_port_write_batch` has rows with `total_ports > 50`
- Operlog shows writes with `DurationMs > 600000` (10 minutes)
- Pool stats show `active_connections == max_connections` for sustained periods

**Phase to address:** Phase 50 — hard cap constant + handler validation; Phase 52 (Frontend) — Ant Design Table selection warning

**Code references:**
- `internal/services/portcollection/collection.go:65-80` (existing per-device loop pattern, no per-port cap)
- `internal/services/config_execution_service.go:50-100` (existing batch template execution, may have similar issue)

---

## Moderate Pitfalls

### Pitfall 12: Privilege Escalation — Operator User Role Can Disable Critical Uplinks

**What goes wrong:** The new `network:port:write` permission grants ALL operators the ability to issue `shutdown` on any port, including uplink ports (GE0/0/47-48 on an access switch) that would knock out the entire building's network access. There is no per-port or per-port-role policy ("operator can manage access ports but not uplinks").

**Prevention:**
- Add a `sys_port_write_policy` table: `(device_id, interface_pattern, allowed_roles)` — e.g. `GE0/0/4[0-7]$` is restricted to `role:network-admin`
- Default MVP policy: any port ending in `47|48` requires `network:port:write:critical` perm
- The locked decision in PROJECT.md only mentions `network:port:write` (single perm); consider a follow-up v1.19+ for `network:port:write:critical`

**Warning signs:** Incident report shows operator disabled uplink port; no policy blocked it

**Phase to address:** Phase 50 (MVP) / Phase 53 (v1.19+ follow-up)

---

### Pitfall 13: Auto-Collection Race with Operator's Manual Refresh

**What goes wrong:** Operator clicks "Refresh ports" while the post-write enqueue is mid-collection. Both reads hit the same `sys_device_port_status` table, the v1.18 collector may overwrite a row the operator's manual refresh is about to display.

**Prevention:** Use `find()` with `Order("collected_at DESC").First()` and a 1-second staleness window; the UI should not refresh individual rows during an in-flight write batch.

**Warning signs:** UI shows port state changing every 1-2 seconds during a write batch

**Phase to address:** Phase 52 (Frontend) — disable manual refresh while batch in-flight

---

### Pitfall 14: Test Coverage Gap — No Mock SSH Server for Real Vendor Prompts

**What goes wrong:** Unit tests pass with mock `ScrapliWrapper` returning canned responses, but production failures (Huawei V200R005 vs V600R024 different `% Error: Unrecognized command` wording) are not caught until UAT on real devices.

**Prevention:**
- Add a `internal/device/test_server.go` that runs a real `ssh.NewServer` (golang.org/x/crypto/ssh) with canned prompt handling for each vendor
- Run integration tests against this server: `TestHuaweiShutdownCommand`, `TestH3CDescriptionWithSpace`, `TestRuijieConfigureTerminalError`
- Wire to `make test-integration` so they're not silently skipped

**Warning signs:** Unit tests pass, UAT on real devices finds syntax errors; no pre-deploy integration test failure

**Phase to address:** Phase 50 (test infrastructure) / Phase 54 (full integration test suite)

**Code references:**
- `internal/device/connection_pool_test.go` (existing test pattern using mocks)
- `internal/services/portcollection/collection_test.go` (existing read-only integration test)

---

### Pitfall 15: Rollback Strategy Missing — Write Succeeds, Operator Wants Undo

**What goes wrong:** Operator disables GE0/0/1 by accident. There's no "undo" button. They have to remember what the previous description was and manually re-issue a write. The v1.19 PROJECT.md explicitly defers auto-rollback ("MVP 仅'失败即停 + 回传失败点'"), but the OPERATOR EXPECTATION is different — they assume a "single-port undo" exists.

**Prevention:**
- The `sys_port_write_audit.BeforeValue` (from Pitfall 2) makes undo trivial: re-issue a write with `value=BeforeValue`
- Frontend's "操作历史" tab can show "revert" button for any audit row
- No automatic rollback (explicitly out of MVP scope per PROJECT.md lock), but make the data available

**Warning signs:** Operator reports "I want to undo this" and the only path is manual SSH

**Phase to address:** Phase 52 (Frontend) — "Revert" action reading from `sys_port_write_audit`

---

## Minor Pitfalls

### Pitfall 16: Cache Prefix Pollution When Adding Port Write Cache Invalidation

**What goes wrong:** The Redis prefix is `xingran:` (`core.go:342`); all cache keys must be added with the prefix implicitly via the cache interface, not by hardcoding `xingran:port:write:status`. If the write service hardcodes the prefix in a cache key like `cache.Set(ctx, "xingran:port:status:device-123", ...)`, the actual stored key becomes `xingran:xingran:port:status:device-123` — broken.

**Prevention:** Use a helper function `GetPortStatusKey(deviceID) string` that returns `"port:status:" + deviceID` (no prefix), and let the cache layer add `xingran:`. Strip prefix on read (`strings.TrimPrefix(key, "xingran:")`) when accepting user input from the cache monitor UI.

**Warning signs:** Cache monitor shows keys with double prefix; cache invalidation never hits the right key

**Phase to address:** Phase 50 — `cache_keys.go` for port write audit cache

**Code references:**
- `internal/core/core.go:342` (prefix hardcoded)
- `internal/services/cache_keys.go` (existing helper pattern)

---

### Pitfall 17: Operlog Module Name Inconsistency — "端口管理" vs "网络设备" vs "端口写入"

**What goes wrong:** Three different modules exist for related operations: existing `端口管理` (port collect/list), `网络设备` (device CRUD), and a new `端口写入` (port write). A query "all operations on GE0/0/1" must search 2-3 different module names.

**Prevention:** Use a single module name `端口管理` for all port-related operations (consistent with the existing `port_handler.go` convention), and use `OperType` (Write=Update, BatchWrite=Batch) to distinguish. The new "端口写入" module name is redundant.

**Warning signs:** Audit search returns 3 separate module rows for one user action

**Phase to address:** Phase 50 — use existing module name `端口管理` with `OperTypeUpdate` / `OperTypeBatch` / `OperTypeStatus`

**Code references:**
- `internal/api/v1/network/port_handler.go:109` (existing `端口管理` convention)
- `internal/utils/operlog/operlog.go:35-67` (25 OperType constants cover this)

---

### Pitfall 18: UUID Validation for Device/Port Parameters

**What goes wrong:** v1.19 introduces `device_id` (UUID) and `interface_name` (string, not UUID) in write requests. Hand-written validation may accept malformed UUIDs, leading to GORM SQLSTATE 42703 or 22P02 on PostgreSQL.

**Prevention:** Use the same UUID regex as `building_service.go validateOrg()`: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`. For `interface_name`, validate against `^[A-Za-z]+[A-Za-z0-9/\-\.]+$` to reject shell-injection patterns.

**Warning signs:** SQLSTATE 22P02 in handler logs; PostgreSQL "invalid input syntax for type uuid"

**Phase to address:** Phase 50 — request struct binding tags + manual validation in handler

**Code references:**
- `internal/services/operations/building_service.go validateOrg()` (existing UUID pattern)
- `internal/services/portcollection/collection.go:30-40` (existing pattern, no UUID validation because the IDs come from query)

---

## Phase-Specific Warnings

| Phase | Likely Pitfall | Mitigation |
|-------|---------------|------------|
| **Phase 50: Write Command Service** | Pitfall 1, 3, 4, 5, 6, 7, 9, 10, 11 | Hard cap, vendor templates, port lock, separate audit table, RecordBackground, health check |
| **Phase 51: Audit & Operlog Integration** | Pitfall 2, 5, 9 | `sys_port_write_audit` table; targeted regex masking; RecordBackground for cron-path enqueue |
| **Phase 52: Frontend Batch Dialog** | Pitfall 8, 15, 17 | BatchProgressDialog (poll-based, no row optimistic updates); Revert button using audit BeforeValue; single module name `端口管理` |
| **Phase 53: Permission Policy (v1.19+)** | Pitfall 12 | `sys_port_write_policy` table + `network:port:write:critical` permission |
| **Phase 54: Test Infrastructure** | Pitfall 14 | Mock SSH server for vendor-specific prompt tests |

---

## "Looks Done But Isn't" Checklist

- [ ] **Audit completeness:** `sys_port_write_audit` captures BOTH `BeforeValue` AND `AfterValue` (not just the command sent)
- [ ] **Error parsing:** `SendConfig` results are checked for `% Error:` / `Unrecognized command` markers, not just transport errors
- [ ] **Vendor coverage:** All 3 vendors (Huawei / H3C / Ruijie) have tested templates for `shutdown` / `description` / `dot1x` — verify by running `TestVendorWriteTemplate` for each
- [ ] **Port lock:** Concurrent writes to the same port from different operators return a "locked by X" error, not last-write-wins
- [ ] **Batch detach:** `c.Request.Context()` is NOT used for the batch execution; the context outlives the HTTP request
- [ ] **Post-write collection:** `Enqueue(deviceID)` is called after write success AND the enqueue itself is audited via `RecordBackground` (not `Record`)
- [ ] **Sensitive masking:** `description` body is NOT auto-masked (the field is not sensitive), but commands containing `snmp community / password / secret` are regex-masked
- [ ] **Module name consistency:** All port write operlog rows use `端口管理` (not a new `端口写入` module)
- [ ] **Cache prefix:** Port write cache keys use the helper function (no hardcoded `xingran:` prefix in the key string)
- [ ] **Batch cap:** 50-port hard cap is enforced at the handler; >20-port soft warning is visible in the UI
- [ ] **Health check:** Device unreachable during batch results in `failed_ports` array, not silent skip
- [ ] **Revert support:** Audit `BeforeValue` is stored long-term (no auto-purge) so "revert" is always possible

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| 1. SSH write silent rejection | MEDIUM | Run `display current-configuration interface GE0/0/X` on the device to verify state; if mismatched, re-run the write |
| 2. Missing before-state in audit | HIGH | Cannot recover historical state; future writes now have it, but past writes are write-only |
| 3. Vendor syntax wrong | LOW | Update template map; re-run failed ports from the failed_ports list |
| 4. Concurrent write collision | MEDIUM | Re-apply the LATER operator's intent (last-write-wins is the current semantics; manually re-issue if it was the earlier one's intent that mattered) |
| 5. Over-masked audit | MEDIUM | Cannot recover; sys_oper_log row is permanently masked. Re-issue writes with `sys_port_write_audit.CommandSent` (which is unmasked) as the source of truth |
| 6. Batch exceeds 30s shutdown | LOW | Core.Close already handles this (Pitfall 6 prevention); recovery is automatic |
| 7. Pool exhaustion | LOW | Pool self-recovers when batches complete; temporary "连接池已满" errors are user-actionable (retry) |
| 8. Optimistic UI race | LOW | React Query cache invalidation after batch completion (Pitfall 8 prevention) |
| 9. Missing post-write enqueue audit | LOW | Re-query sys_oper_log with broader filter; future batches will have the row |
| 10. Device unreachable mid-batch | LOW | Re-run `failed_ports` from the batch status response |
| 11. Batch size overflow | LOW | Operator chunks the batch manually; UI shows soft warning |
| 12. Privilege escalation | HIGH | Manually re-enable the port; review policy; future writes need the critical permission |
| 13. Auto-collection race | LOW | UI clears after batch completes; no persistent damage |
| 14. Test coverage gap | HIGH | Add mock SSH server; rebuild + redeploy; no in-place fix |
| 15. Rollback missing | LOW | `sys_port_write_audit.BeforeValue` enables manual revert (just re-issue a write with the old value) |
| 16. Cache prefix pollution | MEDIUM | `cache.DeleteByPattern("xingran:port:status:*")` to clean polluted keys; fix helper function |
| 17. Module name inconsistency | LOW | Update module name in new handlers; no data migration needed |
| 18. UUID validation missing | LOW | Add binding tag; failed requests get 400 instead of 500 |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1. SSH silent rejection | Phase 50 | Unit test: `SendConfig` with mock returning `% Error: Unrecognized command` → service returns `device_rejected` |
| 2. No before/after snapshot | Phase 50 + 51 | Integration test: `BeforeValue` field is populated before write, `AfterValue` is populated by next v1.18 collect |
| 3. Vendor syntax subtlety | Phase 50 | Integration test: `TestHuaweiShutdownCommand`, `TestH3CDescriptionWithSpace`, `TestRuijieConfigureTerminalError` against mock SSH server |
| 4. Concurrent write collision | Phase 50 | Concurrency test: 2 goroutines writing same port → second returns `ErrPortLocked` |
| 5. Operlog masking | Phase 51 | Unit test: `description: DMZ-port-for-key-service` → `sys_port_write_audit.CommandSent` shows full text, `sys_oper_log.oper_param` shows only high-level summary |
| 6. Batch 30s shutdown | Phase 50 | Load test: 50-port batch with mock 100s-per-command SSH → context timeout cancels batch cleanly |
| 7. Pool exhaustion | Phase 50 + 51 | Load test: 5 concurrent 50-port batches + 1 cron enqueue → cron succeeds (no "连接池已满") |
| 8. Frontend optimistic UI | Phase 52 | Manual test: click batch, navigate away, navigate back → batch progress dialog persists in URL state |
| 9. RecordBackground for post-write | Phase 50 | Unit test: post-write enqueue → `sys_oper_log` row with `method=BACKGROUND` exists |
| 10. Device unreachable mid-batch | Phase 50 | Integration test: kill SSH server after 15 ports → `failed_ports` array contains 16-50, status=failed |
| 11. Batch size cap | Phase 50 | Manual test: select 100 ports → 400 error "单次批量操作最多 50 个端口" |
| 12. Privilege escalation | Phase 53 (v1.19+) | Manual UAT: operator role tries to disable uplink port → 403 forbidden |
| 13. Auto-collection race | Phase 52 | Manual test: click batch + manual refresh simultaneously → no flicker |
| 14. Test coverage gap | Phase 54 | CI: `make test-integration` runs vendor-specific tests; green check required for merge |
| 15. Rollback missing | Phase 52 | Manual test: "Revert" button reads from `sys_port_write_audit.BeforeValue` and re-issues a write |
| 16. Cache prefix | Phase 50 | Unit test: `GetPortStatusKey("uuid")` returns `"port:status:uuid"` (no `xingran:` prefix) |
| 17. Module name | Phase 50 | Query test: `SELECT title FROM sys_oper_log WHERE title IN ('端口管理', '网络设备', '端口写入') AND url LIKE '/network/port/write%'` → returns ONLY `端口管理` rows |
| 18. UUID validation | Phase 50 | Unit test: invalid UUID in request body → 400 error, not 500 |

---

## Sources

- **HIGH Confidence (codebase-verified):**
  - `internal/device/scrapli_wrapper.go:567-617` (SendConfig/SendConfigs do not parse device error markers)
  - `internal/device/connection_pool.go:1-583` (DefaultPoolConfig maxConnections=50, per-device lock, no per-port lock)
  - `internal/utils/operlog/operlog.go:135-170` (11 mandatory sensitive keywords, locked by regression_test.go)
  - `internal/utils/operlog/operlog.go:265-321` (RecordBackground signature for cron-path audit)
  - `internal/services/device_info_collection_service.go:75-130` (existing background queue + 8s shutdown timeout pattern)
  - `internal/services/device_info_collection_service.go:133-163` (existing enqueue with queue-full rejection)
  - `internal/core/core.go:469, 488` (30s shutdown deadlines, non-negotiable)
  - `internal/api/v1/network/port_handler.go:1-203` (existing port management handler pattern, no write endpoints yet)
  - `internal/api/v1/network/command_handler.go:74, 106` (existing read-only QuickCommand uses operlog with OperTypeOther)
  - `internal/services/portcollection/collection.go:17, 72` (portCollectionDeviceTimeout=10min, per-device context)
  - `internal/services/portcollection/parser.go` (existing parser abstraction to mirror for write)
  - `internal/services/component_collector/ops_asset_writer.go` (Phase 48 D-13 RecordBackground example)
  - `internal/services/addomain/account_pool.go` (existing SELECT FOR UPDATE row-lock pattern to mirror)
  - `internal/services/operations/building_service.go validateOrg()` (UUID validation pattern)
  - `internal/services/cache_keys.go` (cache key helper pattern, no `xingran:` prefix in key string)
  - `.planning/PROJECT.md:7-30` (v1.19 locked decisions: vendor template hardcoded, MVP single device + batch, no auto-rollback, post-write enqueue via DeviceInfoCollectionService)

- **MEDIUM Confidence (vendor knowledge):**
  - Huawei VRP V200/V600 command syntax differences (shutdown vs undo, system-view semantics)
  - H3C Comware 7 command compatibility with Huawei (shared VRP heritage, subtle differences)
  - Ruijie RGOS Cisco-IOS-XE-style syntax (configure terminal, end, write, interface NAME NUMBER)
  - dot1x command keyword differences across vendors
  - commit/quit semantics (Huawei VRP auto-commit, Ruijie requires end, H3C similar to Huawei)

- **MEDIUM Confidence (community patterns):**
  - SSH keepalive / ServerAliveInterval (Go x/crypto/ssh default 0 = disabled)
  - Scrapli community patterns for error marker parsing (`%`, `Error:`, `^`, `Illegal`)
  - Industry patterns for "before/after state capture" (Ansible, Salt, NAPALM)

- **LOW Confidence (assumed):**
  - Exact 30s Core.Close deadline impact on write batches (depends on actual SSH round-trip times in production)
  - Actual user expectation of "revert" button (depends on v1.19+ UAT)
  - Specific firmware version differences (Huawei V600R024C00 chassis SN retirement, but write command syntax version sensitivity not directly verified)

---

## Open Questions Requiring Phase-Specific Research

1. **Vendor command coverage for firmware variants:** Do we need to support multiple command variants per vendor for different firmware (e.g. Huawei V200R005 vs V600R024C00)? The MVP scope says "硬编码 vendor→template map", but a single map per vendor may not cover all deployments. **Answer via Phase 50 UAT on real devices.**
2. **Batch resume semantics:** If a batch fails at port 30/50 due to "device unreachable", should the resume endpoint re-run from port 30, or re-run only the failed ports? This is a UX question requiring user feedback. **Answer via Phase 52 UAT.**
3. **Auto-rollback policy:** If write N+1 of a batch fails, should the service automatically issue `undo` commands for writes 1..N? PROJECT.md says "MVP 仅失败即停", but the auto-rollback question is a v1.19+ follow-up. **Answer via Phase 51 hardening (deferred).**
4. **Critical port policy granularity:** Should the `sys_port_write_policy` be by `interface_name` regex, by `port_role` (uplink/access/server), or by `device_tag`? **Answer via Phase 53 (v1.19+ follow-up).**
5. **Operlog `oper_param` size for command bodies:** The 8192-byte cap (`operlog.go:71`) is fine for single-command writes, but a batch of 50 commands × ~80 chars = 4000 chars — within cap, but a 100-port description batch (deferred but possible) would be 8000+ chars. **Answer via Phase 50 (use `sys_port_write_audit` for the full body, operlog only has the high-level summary).**
6. **Sensitive keyword "key" in port descriptions:** The "key" keyword matches `description: "Key-Service-VLAN-10"` and over-masks. Should the masking be opt-out for `description` specifically? Or should the audit row use `sys_port_write_audit` (which has targeted regex) and not rely on operlog masking for descriptions? **Answer: use `sys_port_write_audit` (decision locked).**

---

## Conclusion

Adding SSH write operations to XingRan-Next's network management system is a **fundamentally different** engineering challenge than the v1.18 read path. Reads are idempotent and lossless; writes are destructive, require state capture, and have vendor-specific semantics that the existing v1.18 code did not need to handle.

The critical risks are:

1. **Silent device rejection** (Pitfall 1) — SSH transport success ≠ device acceptance; the parser must check for `% Error:` markers
2. **Missing audit before-state** (Pitfall 2) — without `BeforeValue`, compliance investigations fail
3. **Vendor syntax subtlety** (Pitfall 3) — Huawei/H3C/Ruijie have non-obvious differences; a hardcoded template map must be tested against each vendor's actual output
4. **Concurrent write collision** (Pitfall 4) — the connection pool's per-device lock is not port-aware; a separate `sys_port_write_lock` table is needed
5. **Operlog masking leaks** (Pitfall 5) — the 11 sensitive keywords over-mask port descriptions; `sys_port_write_audit` is the right home for the full body
6. **30s shutdown deadline** (Pitfall 6) — batches must detach from the HTTP request context
7. **Pool exhaustion** (Pitfall 7) — per-port release, not per-batch release
8. **Frontend optimistic UI race** (Pitfall 8) — use a batch progress dialog, not row-level optimistic updates
9. **RecordBackground for cron-path audit** (Pitfall 9) — `operlog.Record` from background fails silently
10. **Device unreachable mid-batch** (Pitfall 10) — health check + per-port retry with `failed_ports` array
11. **No batch size cap** (Pitfall 11) — 50-port hard cap + 20-port soft warning

**Success criteria for v1.19:**
- All 11 critical pitfall mitigations are implemented and verified
- `sys_port_write_audit` table is populated for every write attempt (success or failure)
- Operlog rows for port writes use module name `端口管理` (not `端口写入`)
- Pool stats show no `active_connections == max_connections` sustained condition
- Frontend batch dialog polls for progress; no row-level optimistic updates
- UAT on real devices: at least 1 successful batch per vendor, with `failed_ports` correctly handling device_unreachable vs device_rejected
- All Phase 50-52 plans have a "pitfall verification" step that references one of the pitfalls above

**Recommended approach:**
1. Phase 50: SSH write service + `sys_port_write_audit` + `sys_port_write_lock` + vendor templates + RecordBackground integration
2. Phase 51: Operlog integration (high-level only) + cache invalidation + batch progress tracking
3. Phase 52: Frontend batch dialog + Revert button + manual refresh suppression during batch
4. Phase 53 (v1.19+): Privilege policy table for critical ports
5. Phase 54 (v1.19+): Mock SSH server for vendor-specific integration tests

By addressing these pitfalls proactively, v1.19 will add destructive write capability to the network management system **without** compromising the read-path reliability established in v1.18, and **without** compromising the audit trail integrity that Phase 34 (operlog) established as a non-negotiable invariant.

---

*Pitfalls research for: v1.19 网络设备写命令 (Network Device Write Operations)*
*Researched: 2026-07-06*
