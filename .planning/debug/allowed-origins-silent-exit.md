---
slug: allowed-origins-silent-exit
name: allowed-origins-silent-exit
status: awaiting_human_verify
trigger: 启动链第三环: 迁移/服务全部初始化成功后,HTTP 启动前静默 exit status 1,终端无任何错误输出
created: 2026-10-15
updated: 2026-10-15 14:15
---

## 症状 (Symptoms)

**期望行为:**
- 应用启动完成，HTTP 服务器监听端口（"服务器启动在端口: %d"）

**实际行为:**
- 所有迁移/缓存/调度器/RPA 初始化日志正常输出后，进程直接 `exit status 1`，终端无错误信息
- 终端最后一条日志: `[SkipSetup=true] 跳过 cmd-level seed (MAC history retention / OUI)`

**错误信息:**
```
(终端无错误 — 静默 exit 1)
logs/app.log 最后一行:
{"level":"fatal","msg":"server.allowed_origins must be configured in production: at least one origin is required","time":"2026-10-15 13:51:36"}
```

**时间线:**
- 未提交改动（git diff cmd/main.go）：原 `allowedOrigins := []string{"*"}`（dev 兜底）被替换为 `cfg.Server.AllowedOrigins` + D-02 无条件 fail-fast

**重现步骤:**
1. `go run ./cmd/main.go`（前两环修复后）
2. 100% 复现

---

## 当前关注点 (Current Focus)

```yaml
reasoning_checkpoint:
  hypothesis: "[已确认] 双重根因: (1) config.yaml 无 server.allowed_origins 且无 SetDefault → 空列表 → main.go D-02 无 mode 判断的无条件 Fatalf; (2) pkg/logger Fatalf 先调 fl.Fatal(内部 os.Exit(1)) → 后续 stdout 输出是死代码 → 终端静默"
  confirming_evidence:
    - "logs/app.log 尾行 fatal: 'server.allowed_origins must be configured in production...' 而终端无任何输出"
    - "git diff: 原 dev 兜底 allowedOrigins := []string{\"*\"} 被删,替换为无条件 fail-fast"
    - "代码审查: logger.go:302 fl.Fatal(msg) → logrus 内部 os.Exit(1) → :305 GetLogger().Error(msg) 不可达"
    - "100% 复现: go run ./cmd/main.go 必现静默 exit 1"
  falsification_test: "若 Fatalf 后 stdout 可见错误消息且非 release 模式不退出 → 根因被推翻(未发生)"
  fix_rationale: "FIX1 按 D-02 注释本意('in production')给 fail-fast 加 Mode==release 判据,非 release 恢复被删的 dev 兜底 {'*'}+Warn;FIX2 重排 Fatalf: stdout 先行(Error 级,避开 fileMirrorHook Fatal 级镜像的 hook 内 exit 风险)→ fl.Log(FatalLevel) 落文件(不触发 logrus 自带 exit)→ Close flush → exitFunc(1) 最后退出。针对根因(顺序错误)而非症状(加 print)"
  blind_spots: "ConsoleOutput=true 时 console Error 经 fileMirrorHook 会在 app.log 多落一条 error 级镜像(与 fatal 条目并存,无害);lumberjack Close flush 语义依赖;exitFunc 注入点为新增包级变量"
```

**下一步行动 (Next Action):**
修复已应用并通过 build/vet/test 自验证(status=awaiting_human_verify)。等待用户执行 `go run ./cmd/main.go` 端到端确认:终端应出现 allowed_origins fallback Warnf + "*" 通配 Warnf,进程不再静默退出。用户确认后归档到 resolved/ 并更新 knowledge base;未提交任何 git commit。

---

## 证据 (Evidence)

- timestamp: 2026-10-15 13:55:00
  source: main_flow_analysis
  detail: |
    cmd/main.go main(): initializeCoreModule(171 最后日志) → setupGinEngine → :106-108 allowed_origins 空检查 Fatalf → setupRoutes → startServer(:266 "服务器启动在端口" 日志) 。
    "服务器启动在端口" 未出现在终端 → exit 点在 :107 Fatalf。logs/app.log 尾行 fatal 消息证实。

- timestamp: 2026-10-15 13:56:00
  source: git_diff
  detail: |
    git diff cmd/main.go（未提交）:
    - 移除: `allowedOrigins := []string{"*"}`
    - 新增: `allowedOrigins := cfg.Server.AllowedOrigins` + D-02 fail-fast `if len(allowedOrigins)==0 { applogger.Fatalf(...) }` — 无 Mode 判断，dev 同样命中。

- timestamp: 2026-10-15 13:57:00
  source: config_audit
  detail: |
    - configs/config.yaml: 无 allowed_origins 键
    - internal/config/config.go:56: `AllowedOrigins []string` 无 v.SetDefault
    - configs/config.prod.example.yaml:38-39: 有生产示例（显式域名列表）
    → dev 环境空列表 → Fatal 必现

- timestamp: 2026-10-15 13:58:00
  source: code_review
  detail: |
    pkg/logger/logger.go:294-306 Fatalf:
    ```go
    func Fatalf(format string, args ...interface{}) {
        msg := fmt.Sprintf(format, args...)
        if fl != nil {
            fl.Fatal(msg)        // ← logrus Fatal 内部 os.Exit(1)，进程到此结束
        }
        GetLogger().Error(msg)   // ← 死代码，永不可达
        ...
    ```
    注释声称"手动写镜像 + sync 再 exit"，但 logrus `logger.Fatal()` 自带 `os.Exit(1)`，顺序错误导致 console 输出永远丢失。所有 Fatalf 调用点（如 "创建核心模块失败"、"服务器启动失败"）在终端都不可见，只落文件。

- timestamp: 2026-10-15 14:12:00
  source: fix_applied
  detail: |
    FIX1 cmd/main.go: D-02 加 `cfg.Server.Mode == "release"` 判据;非 release 空列表 → fallback `{"*"}` + Warnf(恢复被删的 dev 兜底);"*" 通配 Warnf 循环原样保留。
    FIX2 pkg/logger/logger.go: Fatalf 重排——① GetLogger().Error(msg) stdout 先行(Error 级避开 fileMirrorHook 对 Fatal 级镜像的 hook 内 exit 风险) → ② fl.Log(logrus.FatalLevel, msg) 文件镜像(Log 不触发 logrus 自带 exit) → ③ fw.Close() flush → ④ exitFunc(1) 最后退出。新增包级 `var exitFunc = os.Exit` 注入点。
    FIX3 pkg/logger/logger_test.go: 追加 TestFatalf_StdoutBeforeExit_FatalMirrorFlushed(exitFunc 注入桩,解除 QUIRK-80-05-E Fatal 系豁免)。

- timestamp: 2026-10-15 14:14:00
  source: verification
  detail: |
    go build ./... → PASS(无输出)
    go vet ./cmd/... ./pkg/logger/... → PASS(无输出)
    go test ./pkg/logger/... → ok 0.729s(全量)
    go test -v -run TestFatalf_StdoutBeforeExit_FatalMirrorFlushed → PASS 0.03s(确认执行非跳过;验证 stdout 先收到消息 + 退出码 1 + app.log 含 "level":"fatal" 镜像)

---

## 已排除 (Eliminated)

- hypothesis: 端口占用 / ListenAndServe 失败
  reason: ListenAndServe 失败走 :273 Fatalf 且 goroutine 在 srv 创建后立即启动；app.log 的 fatal 消息是 allowed_origins，非端口错误。

---

## 解决方案 (Resolution)

root_cause: 双重根因——(1) config.yaml 缺 server.allowed_origins 且无 SetDefault → 空列表命中 cmd/main.go D-02 无 Mode 判断的无条件 Fatalf(dev 兜底 `{"*"}` 在重构中被删);(2) pkg/logger Fatalf 先调 fl.Fatal(内部 os.Exit(1))致后续 stdout 输出成死代码 → 终端静默仅 app.log 有记录。
fix: FIX1 main.go D-02 仅在 `Mode=="release"` 时 Fatalf,非 release 空 → fallback `{"*"}` + Warnf;FIX2 Fatalf 重排为 stdout(Error)先行 → fl.Log(FatalLevel) 文件镜像 → Close flush → exitFunc(1);新增可注入 `var exitFunc = os.Exit`;追加顺序回归测试 TestFatalf_StdoutBeforeExit_FatalMirrorFlushed。
verification: go build ./... PASS;go vet ./cmd/... ./pkg/logger/... PASS;go test ./pkg/logger/... PASS(含新回归测试 -v 确认执行)。端到端 go run 由用户确认。
files_changed:
  - cmd/main.go
  - pkg/logger/logger.go
  - pkg/logger/logger_test.go
