---
status: diagnosing
trigger: "Linux服务器从6月24号左右起log目录下就没有生成日志了"
created: 2026-07-09T08:40:00+08:00
updated: 2026-07-09T08:45:00+08:00
---

## Current Focus

hypothesis: "待确认：需要用户提供远程诊断命令输出"
test: "在Linux服务器执行诊断命令"
expecting: "找到日志停止写入的具体原因"
next_action: "请用户在Linux服务器执行诊断命令收集信息"

## Symptoms

expected: "logs/目录下持续有新的日志文件生成（app.log + 定期压缩的 app-YYYY-MM-DDTHH-MM-SS.mmm.log.gz）"
actual: "logs/目录下最新压缩日志是 app-2026-06-13T22-48-01.156.log.gz，app.log 最后修改时间是 2026-06-22 14:02，6月24号之后没有任何新日志"
errors: "无错误提示（用户只是发现日志停止生成）"
reproduction: "无法在本地复现（需要登录Linux服务器诊断）"
started: "2026-06-24 左右"

## Eliminated

- hypothesis: "lumberjack MaxSize=100MB 导致日志轮转后旧日志被删除"
  evidence: "app.log 只有 79MB，远未达到 100MB 阈值；且 max_backups=30 意味着最多保留30个压缩备份，不会停止生成"
  timestamp: 2026-07-09

- hypothesis: "log_dir 配置被修改为其他目录"
  evidence: "代码中 log_dir 来自 cfg.Log.LogDir，默认值是 logs，config.yaml 中显式配置为 logs"
  timestamp: 2026-07-09

- hypothesis: "log level 被改为 error 导致 info/debug 日志消失"
  evidence: "即使 level=error，logrus 仍然会创建/写入 app.log 文件，只是记录更少内容；且启动日志一定会写"
  timestamp: 2026-07-09

## Evidence

- timestamp: 2026-07-09
  checked: "pkg/logger/logger.go"
  found: "使用 logrus + lumberjack 做日志管理，createFileWriter() 创建 lumberjack.Logger 写入 logs/app.log，MaxSize=100MB, MaxBackups=30, MaxAge=90天, Compress=true。Init() 中 ConsoleOutput: true 硬编码，导致 applogger.Info() 等写入 stdout（被 systemd journal 捕获）。fileLogger 单独写文件（JSON格式）"
  implication: "日志分两路：文件(app.log) vs stdout(journald)，HTTP请求日志走 journald 不走文件"

- timestamp: 2026-07-09
  checked: "cmd/main.go"
  found: "initializeLogger() 调用 applogger.Init(logConfig)，ConsoleOutput: true。所有 applogger.Info/Error 调用都走 stdout (journald)。关键启动日志在启动时写入"
  implication: "启动日志和应用层日志走 journald，不是 app.log"

- timestamp: 2026-07-09
  checked: "pkg/middleware/logger.go"
  found: "Request Logger 中间件使用 logger.WithFields() 记录 HTTP 请求。logger.WithFields() 调用 GetLogger() → console logger → stdout/journald"
  implication: "HTTP 请求日志走 journald，不写入 app.log！app.log 只包含少量启动/业务日志"

- timestamp: 2026-07-09
  checked: "本地 logs/ 目录（开发机）"
  found: "app-2026-04-21 ~ app-2026-06-13 共9个压缩文件，app.log 79MB 最后修改 Jun 22"
  implication: "开发机 logs/ 和生产环境无直接关联，本地日志仅作架构参考"

- timestamp: 2026-07-09
  checked: "docs/deployment.md"
  found: "systemd service 配置: StandardOutput=journal, StandardError=journal。服务名: szh-backend。日志查看命令: journalctl -u szh-backend -f"
  implication: "所有 stdout/stderr 被 journald 捕获，logs/app.log 是文件日志，journald 是另一路"

## Resolution

root_cause: "待远程诊断确认"
fix: ""
verification: ""
files_changed: []
---

## 诊断待确认项（需要用户在Linux服务器执行）

### 1. 服务状态检查
sudo systemctl status szh-backend

### 2. 查看 journald 日志（可能日志在这里）
sudo journalctl -u szh-backend --since "2026-06-24" | tail -50
sudo journalctl -u szh-backend -n 100 --no-pager

### 3. 查看 logs/ 目录实际内容
ls -la /app/szh/logs/
stat /app/szh/logs/app.log

### 4. 检查磁盘空间
df -h /app/szh/
df -h /var/log/

### 5. 检查进程是否还在运行
ps aux | grep xingran
pidof xingran-backend

### 6. 查看服务启动时间和最近重启
sudo systemctl show szh-backend -p ActiveEnterTimestamp,ActiveExitTimestamp
sudo journalctl -u szh-backend -b | head -20

---

## 根因假设（按概率排序）

### 假设 1：服务在 6月22日之后重启，新的 app.log 在其他位置 [概率最高]
app.log 最后修改时间是 6月22日 14:02。如果服务在 6月22日之后重启，lumberjack 会创建新的 app.log，旧的 79MB 文件保留。
诊断：执行 ls -la /app/szh/logs/ 看是否有新的 app.log 或其他时间戳的文件。

### 假设 2：服务崩溃退出但 systemd 重启了它，journald 在接收日志但文件日志停止 [概率中等]
如果应用崩溃但 systemd 重启了它（Restart=always），可能新进程没有正确初始化日志。
诊断：执行 sudo systemctl status szh-backend 看服务状态，journalctl 是否有持续日志。

### 假设 3：磁盘空间不足，lumberjack 无法写入 [概率较低]
df -h /app/szh/ 如果磁盘满了，lumberjack 写入会静默失败。
诊断：执行 df -h /app/szh/ 检查磁盘空间。

### 假设 4：systemd 服务配置变更，WorkingDirectory 不再是 /app/szh [概率低]
诊断：执行 sudo systemctl show szh-backend -p WorkingDirectory 确认。

---

## 关键发现：日志架构问题

重要发现：pkg/middleware/logger.go 的 HTTP 请求日志通过 logger.WithFields() → GetLogger() → stdout → journald，不写入 logs/app.log。

logs/app.log 只包含：
- applogger.Info/Error 等显式调用（启动日志、错误日志）
- 任何使用 logger.GetFileLogger() 的代码

这意味着：
1. HTTP 请求日志不会写入 app.log，而是进入 journald
2. 查看完整日志需要用 journalctl -u szh-backend 而不是只检查 logs/ 目录
3. app.log 只包含少量应用层日志（启动、错误、业务警告等）

建议：告知用户在 Linux 服务器执行 sudo journalctl -u szh-backend --since "2026-06-24" | head -100 检查 journald 是否有持续日志。
