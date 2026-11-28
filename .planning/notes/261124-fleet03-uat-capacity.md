---
created: 2026-11-24
created_by: Phase 157 / FLEET-03（v1.56 W2 波次）
status: 已排期·待执行（0 勾选 / 严禁伪造实测）
related:
  - .planning/notes/261123-rpa-worker-capacity-model.md
  - .planning/notes/261108-human-verification-tracker.md
  - rpa-worker/cmd/capacitybench/main.go
---

# UAT 待办：FLEET-03 容量实测（VDI / 容器内矩阵）

> 本环境（2026-11-24 实证）：**无 Docker**、**无真机 VDI**、**无生产 PG 写授权**。
> 凡以下条目，本波次一律不执行、不推算冒充实测；宿主机可跑部分已在
> `261123-rpa-worker-capacity-model.md` §3/§5 回填（口径见该节，非容器/VM 内数字）。

## UAT-FLEET-01 8GB/4vCPU VM 内矩阵

- **内容**：两档 VM（8GB/4vCPU、16GB/8vCPU）× N=1/2/4/8，采集 P95 执行时延 / Chrome 崩溃率 /
  OOM 率 / 单路 screencast 带宽 / 容器 RSS。
- **前置**：真实 VDI 环境 + `sys_vdi_server` 配置 + 预装 worker 的 VM 镜像（FLEET-02 镜像构建）
  + `rpa.scaling.provider: vdivm`。
- **口径**：容器内 `docker stats` / `cadvisor` 采集；P95 用执行上报时间戳统计；screencast 带宽
  由 LIVEVIEW（Phase 155）帧流口径测量。
- **owner（角色）**：现场运维 + QA；**计划窗口**：v1.56 W4~W5（LIVEVIEW 合入后）
- **证据落盘**：`evidence/UAT-FLEET-01-<YYYYMMDD>.md`（原始数字表 + 环境规格 + 时间窗）。

## UAT-FLEET-02 docker compose config 解析校验

- **内容**：`docker compose -f docker-compose.yml config --quiet` 与 local 版同检
  （`rpa-worker/deployments/drift_test.go` 的 `TestCapacityDrift_ComposeParse`）。
- **前置**：任一有 Docker 的环境（本机无 docker 已在测试中 SKIP 登记）。
- **owner（角色）**：QA / CI；**计划窗口**：v1.56 W3。
- **证据落盘**：CI 日志或 `evidence/UAT-FLEET-02-<YYYYMMDD>.md`。

## UAT-FLEET-03 VDI provider 真机扩缩容全流程

- **内容**：`provider: vdivm` 下 ScaleUp → VM 创建/启动 → worker 注册上线判定 → ScaleDown 回收
  全链路真机验证（含超时回收路径）。
- **前置**：真机 VDI（深信服 API）+ 管理员授权。
- **owner（角色）**：现场运维；**计划窗口**：v1.56 W5。
- **证据落盘**：`evidence/UAT-FLEET-03-<YYYYMMDD>.md`。

## 决策纪律

- 「一 VM 一容器」结构性结论已按承载模型笔记定为默认；**数字级结论**（默认 N、8GB vs 16GB 选型）
  待 UAT-FLEET-01 实测回填，在此之前维持 N=8 作为**待验证的部署基线**，不声称已被实测确认。
