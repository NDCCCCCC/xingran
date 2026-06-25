---
slug: scrapligo-snmp-panic
status: resolved
trigger: "scrapligo transport panic during SNMP Ping operation - transport.(*Standard).Write crash during Channel.GetPrompt when trying to get system name OID 1.3.6.1.2.1.1.5.0"
created: "2026-04-20"
updated: "2026-04-27"
resolved: "2026-04-27"
---

## Symptoms

- **Expected**: SNMP Ping checks device connectivity AND collects system information (hostname via sysName OID)
- **Actual**: Intermittent panic crash in scrapligo transport layer during Channel.GetPrompt
- **Error**: goroutine panic at transport.(*Standard).Write, exit status 2
- **Timeline**: Intermittent - not consistently reproducible
- **Reproduction**: Triggered by scheduled job (cron/task scheduler)

## Stack Trace

```
INFO[2026-04-18 23:00:15] [SNMP Ping] 尝试获取系统名称 OID: 1.3.6.1.2.1.1.5.0
goroutine 72399 [running]:
github.com/scrapli/scrapligo/transport.(*Standard).Write(0x2ac3510?, {0xc00210c8e0?, 0x1?, 0xc00223bf08?})
        C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.3.3/transport/standard.go:249 +0x1b
github.com/scrapli/scrapligo/transport.(*Transport).Write(...)
        C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.3.3/transport/transport.go:212
github.com/scrapli/scrapligo/channel.(*Channel).Write(0xc001eb2820, {0xc00210c8e0, 0x1, 0x1}, 0x0)
        C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.3.3/channel/write.go:12 +0xb9
github.com/scrapli/scrapligo/channel.(*Channel).WriteReturn(...)
        C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.3.3/channel/write.go:17
github.com/scrapli/scrapligo/channel.(*Channel).GetPrompt.func1()
        C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.3.3/channel/getprompt.go:25 +0x8d
created by github.com/scrapli/scrapligo/channel.(*Channel).GetPrompt in goroutine 72423
        C:/Users/CPIC/go/pkg/mod/github.com/scrapli/scrapligo@v1.3.3/channel/getprompt.go:22 +0x112
exit status 2
```

## Current Focus

- hypothesis: null
- next_action: gather initial evidence
- test: null
- expecting: null
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

## Eliminated

## Resolution

- root_cause: Scrapligo transport layer panic during concurrent SNMP operations
- fix: Phase 8 implemented panic recovery wrapper, RWMutex concurrency control, and connection readiness validation
- verification: All 3 plans in Phase 8 completed, SNMP operations stabilized
- files_changed:
  - internal/device/scrapli_device.go
  - internal/scheduler/cron.go
  - internal/api/v1/network/snmp_handler.go
