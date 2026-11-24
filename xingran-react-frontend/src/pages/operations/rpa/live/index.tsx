/**
 * Phase 155 / LIVEVIEW-01~05：RPA 实时画面页
 *
 * 归属：协议 §9「路由：只通过 sys_menu seed 注册，页面路径
 * pages/operations/rpa/live/index」。所以本文件必须叫 index.tsx，靠
 * componentLoader 的通配 glob（index 或 detail 命名的 tsx）自动加载
 * （**不改** componentLoader.tsx / DynamicRoutes.tsx）。
 *
 * 覆盖的 REQ：
 *   - LIVEVIEW-01：可观看执行列表 + Canvas 逐帧渲染 + 帧率/带宽小字
 *   - LIVEVIEW-02：鼠标/滚轮/键盘输入注入（画布坐标 → viewport 坐标），非接管者禁输
 *   - LIVEVIEW-03：三模式切换（带审计确认）、接管/释放（409 二次确认抢占）、继续放行
 *   - LIVEVIEW-04：情境叠加层（步骤/动作/日志/脱敏变量/接管角标）
 *   - LIVEVIEW-05：心跳 20s、卸载停流（尽量 keepalive fetch + beacon 兜底）、
 *                  403/401 可读错误并停发所有请求、429 显式展示后端 message
 *
 * WS：**不新建连接**。复用 NotificationBell 的全局唯一 /system/ws/notices
 * 连接 —— 通过 liveFrameBus 的运行时挂接（详见 liveFrameBus.ts 头注释，
 * 以及 notice_hub.go:118 `clients[userID]=client` 的一用户一条连接事实）。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Alert, App, Button, Card, Empty, Space, Spin, Tag, Typography } from "antd";
import { useAuthStore } from "@/store/authStore";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import type {
  LiveExecutionSummary,
  LiveFrame,
  LiveInputCommand,
  LiveContext,
  LiveMode,
} from "./types";
import { computeFrameStats, type FrameStats } from "./liveFrameParser";
import { installLiveFrameTap, onLiveFrame, onLiveState } from "./liveFrameBus";
import {
  continueExecution,
  fetchLiveState,
  keepaliveExecution,
  listWatchableExecutions,
  releaseExecution,
  sendLiveInput,
  setLiveMode,
  stopExecution,
  stopExecutionOnUnload,
  takeoverExecution,
  watchExecution,
} from "./liveApi";
import { LiveCanvas } from "./LiveCanvas";
import { LiveHud } from "./LiveHud";

const { Text } = Typography;

/** 心跳周期（需求硬要求 20s） */
export const KEEPALIVE_INTERVAL_MS = 20_000;
/** 帧率统计窗口内保留的最大样本数（3s 窗口 × 10fps ≈ 30，留一倍余量） */
const MAX_FRAME_SAMPLES = 80;

/** 从错误对象提取 HTTP 状态码（api.ts 的 handleHttpResponseError 会把 status 挂上） */
function errorStatus(error: unknown): number | undefined {
  const status = (error as { status?: unknown })?.status;
  return typeof status === "number" ? status : undefined;
}

/** 从错误对象提取可读 message（后端 429/409 的 message 一律透出，不静默） */
function errorMessage(error: unknown, fallback: string): string {
  const msg = (error as { message?: unknown })?.message;
  return typeof msg === "string" && msg !== "" ? msg : fallback;
}

/** 读取 URL 上的 ?executionId=（页面是 history 路由，query 在 location.search） */
function readExecutionIdFromUrl(): string {
  if (typeof window === "undefined") return "";
  const params = new URLSearchParams(window.location.search);
  return (params.get("executionId") ?? "").trim();
}

export default function RpaLiveViewPage() {
  const { message, modal } = App.useApp();

  // Phase 86 三态门控：观看/接管/输入/放行全部只对持有
  // ops:rpa:execution:list（协议 §3「复用既有读权限，不新增权限码」）的人开放。
  // 未就绪 → 全部入口渲染但 disabled + 提示；就绪且无权限 → 不渲染入口。
  const gate = useActionPermission();
  const readonlyGate = gate.decide(PERMISSION_CODES.rpaExecution.list);

  // 自己的展示名：后端 takeover 记录的是 nickname/username（liveview_handler.go
  // operatorName），HUD 的 takeoverBy 与之后 input 的 403 校验都以它为准。
  const myName = useAuthStore((s) => s.user?.nickname || s.user?.username || "");

  // ---------- 可观看执行列表 ----------
  const [executions, setExecutions] = useState<LiveExecutionSummary[]>([]);
  // ⚠ 初值即 true：effect 内不再同步 setState（react-hooks/set-state-in-effect），
  // 只在请求 settle 的 finally 里落回 false。
  const [listLoading, setListLoading] = useState(true);
  const [curId, setCurId] = useState("");
  const curIdRef = useRef("");

  // ---------- 观看状态 ----------
  const [watching, setWatching] = useState(false);
  const [context, setContext] = useState<LiveContext | null>(null);
  const [mode, setMode] = useState<LiveMode>("copilot");
  const [frame, setFrame] = useState<LiveFrame | null>(null);
  const [stats, setStats] = useState<FrameStats>({ fps: 0, mbps: 0, samples: 0 });

  // ---------- 权限门控（403/401 后置位，之后不再发任何请求） ----------
  const [fatalError, setFatalError] = useState("");
  const blockedRef = useRef(false);

  const samplesRef = useRef<Array<{ timestamp: number; byteSize: number }>>([]);

  useEffect(() => {
    curIdRef.current = curId;
  }, [curId]);

  const blockAll = useCallback((reason: string) => {
    blockedRef.current = true;
    setFatalError(reason);
    setWatching(false);
  }, []);

  // ==================== 1) 加载可观看执行列表 ====================
  useEffect(() => {
    // 门控未就绪 / 无权限 → 一个请求都不发（避免必然 403 的噪声）
    if (gate.state !== "ready") return;
    if (!readonlyGate.visible) return;

    let cancelled = false;
    listWatchableExecutions()
      .then((list) => {
        if (cancelled) return;
        setExecutions(list);
        // URL 带 ?executionId= → 直接定位；否则默认看第一个可观看执行
        const fromUrl = readExecutionIdFromUrl();
        if (fromUrl) {
          setCurId(fromUrl);
        } else if (list.length > 0) {
          setCurId(list[0].id);
        }
      })
      .catch((error) => {
        if (cancelled) return;
        const status = errorStatus(error);
        if (status === 401 || status === 403) {
          blockAll(errorMessage(error, "无权观看 RPA 实时画面"));
          return;
        }
        // 列表失败不致命（仍可手填 executionId 观看），只提示
        message.error(errorMessage(error, "加载可观看执行失败"));
      })
      .finally(() => {
        if (!cancelled) setListLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [blockAll, message, gate.state, readonlyGate.visible]);

  // ==================== 2) 观看生命周期（state 校验 → watch → 停流） ====================
  useEffect(() => {
    if (!curId || blockedRef.current) return;

    let cancelled = false;
    let joined = false;

    // 2a) 进入即调 GET .../liveview/state：既是权限探测也是 HUD 初始值来源
    fetchLiveState(curId)
      .then((res) => {
        if (cancelled) return;
        if (res.data?.context) setContext(res.data.context);
      })
      .catch((error) => {
        if (cancelled) return;
        const status = errorStatus(error);
        if (status === 401 || status === 403) {
          blockAll(errorMessage(error, "无权观看该执行：请确认你拥有 ops:rpa:execution:list"));
          return;
        }
        // 状态端点失败不阻断 watch（后端 hub 未启用等场景仍可能给可读原因）
        message.warning(errorMessage(error, "读取实时画面状态失败"));
      });

    // 2b) watch：429 = 并发上限，message 由后端给出（不静默）
    watchExecution(curId)
      .then((res) => {
        if (cancelled) return;
        joined = true;
        setWatching(true);
        const watchedMode = res.data?.mode;
        if (watchedMode) setMode(watchedMode);
        if (res.data?.takeoverBy) {
          setContext((prev) => ({
            ...(prev ?? { executionId: curId, step: 0, totalSteps: 0 }),
            takeoverBy: res.data?.takeoverBy,
          }));
        }
      })
      .catch((error) => {
        if (cancelled) return;
        const status = errorStatus(error);
        if (status === 401 || status === 403) {
          blockAll(errorMessage(error, "无权观看该执行"));
          return;
        }
        // 429 / 503 / 其它：把后端可读 message 原样展示（禁止静默）
        message.error(errorMessage(error, "开始观看失败"));
      });

    return () => {
      cancelled = true;
      // 切换观看对象 / 卸载：尽力停流（后端 IdleTimeout 是最终保障）
      if (joined) {
        void stopExecution(curId, "前端切换观看对象").catch(() => {
          // 停流失败不影响任何 UI：后端闲置计时会回收
        });
      }
      setWatching(false);
    };
  }, [curId, blockAll, message]);

  // ==================== 3) 心跳（20s） ====================
  useEffect(() => {
    if (!curId || !watching || blockedRef.current) return;
    const timer = setInterval(() => {
      if (blockedRef.current) return;
      void keepaliveExecution(curId).catch((error) => {
        // 心跳失败只记 warn：一次失败不足以判定被踢（后端按 IdleTimeout 判定）
        console.warn("[LiveView] 心跳失败:", errorMessage(error, "keepalive failed"));
      });
    }, KEEPALIVE_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [curId, watching]);

  // ==================== 4) 卸载 / 关页 停流 ====================
  useEffect(() => {
    if (!curId) return;
    const onUnload = () => stopExecutionOnUnload(curId, "页面卸载");
    window.addEventListener("beforeunload", onUnload);
    return () => {
      window.removeEventListener("beforeunload", onUnload);
    };
  }, [curId]);

  // ==================== 5) WS 帧接入（复用唯一连接，不新建 WS） ====================
  useEffect(() => {
    const releaseTap = installLiveFrameTap();

    const offFrame = onLiveFrame((envelope) => {
      // 只画当前观看对象；其它执行的帧（同账号可能多页签观看）直接丢
      const frameExec = envelope.frame.executionId;
      if (frameExec && frameExec !== curIdRef.current) return;

      setFrame(envelope.frame);
      if (envelope.context) setContext(envelope.context);

      const now = Date.now();
      const samples = samplesRef.current;
      samples.push({ timestamp: now, byteSize: envelope.frame.byteSize });
      if (samples.length > MAX_FRAME_SAMPLES) samples.splice(0, samples.length - MAX_FRAME_SAMPLES);
      setStats(computeFrameStats(samples, now));
    });

    const offState = onLiveState((ctx) => {
      if (ctx.executionId && ctx.executionId !== curIdRef.current) return;
      setContext(ctx);
    });

    return () => {
      offFrame();
      offState();
      releaseTap();
    };
  }, []);

  // ==================== 6) 输入注入 ====================
  const handleInput = useCallback((command: LiveInputCommand) => {
    const id = curIdRef.current;
    if (!id || blockedRef.current) return;
    sendLiveInput(id, command).catch((error) => {
      const status = errorStatus(error);
      if (status === 403) {
        // 已不是接管者（被抢占 / 释放）—— 提示一次，遮罩会随下一次 state 帧收敛
        console.warn("[LiveView] 输入被拒（当前非接管者）");
        return;
      }
      // 其它错误不让高频鼠标事件刷屏：只记 warn
      console.warn("[LiveView] 输入下发失败:", errorMessage(error, "input failed"));
    });
  }, []);

  // ==================== 7) 接管 / 释放 / 模式 / 继续 ====================
  const refreshState = useCallback((id: string) => {
    fetchLiveState(id)
      .then((res) => {
        if (res.data?.context) setContext(res.data.context);
      })
      .catch(() => {
        // 刷新失败不致命：下一次 rpa_liveview_state 帧会把 HUD 收敛
      });
  }, []);

  const handleTakeover = useCallback(() => {
    const id = curIdRef.current;
    if (!id) return;

    modal.confirm({
      title: "确认接管",
      content: "接管后你的鼠标/键盘操作会注入到远端页面，并被写入审计日志。确认接管？",
      okText: "接管",
      cancelText: "取消",
      onOk: () => {
        const doTakeover = (force: boolean) =>
          takeoverExecution(id, force)
            .then(() => {
              message.success(force ? "已抢占接管" : "已接管");
              refreshState(id);
            })
            .catch((error) => {
              if (errorStatus(error) === 409) {
                // 409 = 已被他人接管 → 二次确认抢占（协议 §3 force 语义）
                modal.confirm({
                  title: "对方正在接管",
                  content: `${errorMessage(error, "该执行已被他人接管")} —— 是否抢占？抢占会中继对方的接管并写审计。`,
                  okText: "抢占接管",
                  okButtonProps: { danger: true },
                  cancelText: "取消",
                  onOk: () =>
                    takeoverExecution(id, true)
                      .then(() => {
                        message.success("已抢占接管");
                        refreshState(id);
                      })
                      .catch((err) => {
                        message.error(errorMessage(err, "抢占失败"));
                      }),
                });
                return;
              }
              message.error(errorMessage(error, "接管失败"));
            });
        return doTakeover(false);
      },
    });
  }, [message, modal, refreshState]);

  const handleRelease = useCallback(() => {
    const id = curIdRef.current;
    if (!id) return;

    modal.confirm({
      title: "确认释放接管",
      content: "释放后控制权交回自动流程，本次接管时长会写审计。",
      okText: "释放",
      cancelText: "取消",
      onOk: () =>
        releaseExecution(id, "前端手动释放")
          .then(() => {
            message.success("已释放接管");
            refreshState(id);
          })
          .catch((error) => {
            message.error(errorMessage(error, "释放失败"));
          }),
    });
  }, [message, modal, refreshState]);

  const handleModeChange = useCallback(
    (next: LiveMode) => {
      const id = curIdRef.current;
      if (!id || next === mode) return;

      modal.confirm({
        title: "切换协作模式",
        content: `切换到「${next}」会写审计日志（mode_switch）。确认切换？`,
        okText: "切换",
        cancelText: "取消",
        onOk: () =>
          setLiveMode(id, next)
            .then(() => {
              setMode(next);
              message.success("模式已切换");
            })
            .catch((error) => {
              // 非法值后端 400，可读 message 原样展示
              message.error(errorMessage(error, "切换模式失败"));
            }),
      });
    },
    [message, modal, mode]
  );

  const handleContinue = useCallback(() => {
    const id = curIdRef.current;
    if (!id) return;

    modal.confirm({
      title: "确认放行",
      content: "放行后执行将从 paused 恢复为 running，本操作写审计。",
      okText: "放行",
      cancelText: "取消",
      onOk: () =>
        continueExecution(id, "人工确认继续")
          .then(() => {
            message.success("已放行，执行将继续");
            refreshState(id);
          })
          .catch((error) => {
            message.error(errorMessage(error, "放行失败"));
          }),
    });
  }, [message, modal, refreshState]);

  // ==================== 派生状态 ====================
  const takeoverBy = context?.takeoverBy ?? "";

  // 是否「我是接管者」：后端 input 校验 taker == operatorName（即 nickname/username）
  const isController = useMemo(
    () => Boolean(myName) && takeoverBy !== "" && takeoverBy === myName,
    [myName, takeoverBy]
  );

  // 「继续（放行）」仅当执行 paused 且自己是接管者
  const canContinue = context?.status === "paused" && isController;

  return (
    <div style={{ padding: 0 }}>
      {!readonlyGate.visible && (
        <Alert
          type="warning"
          showIcon
          title="无实时画面权限"
          description="实时画面与控制注入需要 ops:rpa:execution:list（执行记录读权限）。请联系管理员授权。"
          style={{ marginBottom: 12 }}
        />
      )}

      {readonlyGate.visible && (
        <Card size="small" title="可观看执行" style={{ marginBottom: 12 }}>
          {listLoading ? (
            <Spin size="small" />
          ) : executions.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="当前没有运行中或已暂停的执行"
            />
          ) : (
            <Space wrap size={8}>
              {executions.map((item) => (
                <Button
                  key={item.id}
                  size="small"
                  type={item.id === curId ? "primary" : "default"}
                  onClick={() => setCurId(item.id)}
                >
                  {item.taskName || item.id}
                  <Text type="secondary" style={{ marginLeft: 6, fontSize: 11 }}>
                    {item.status}
                  </Text>
                </Button>
              ))}
            </Space>
          )}
        </Card>
      )}

      {fatalError && (
        <Alert
          type="error"
          showIcon
          title={fatalError}
          description="已停止所有实时画面请求。请确认权限或联系管理员。"
          style={{ marginBottom: 12 }}
        />
      )}

      {!curId && !fatalError && (
        <Card>
          <Empty description="请选择一个执行开始观看" />
        </Card>
      )}

      {curId && (
        <Card
          size="small"
          title={
            <Space size={8} wrap>
              <span>实时画面</span>
              {!watching && !fatalError && <Tag color="default">未推流</Tag>}
              {watching && <Tag color="processing">推流中</Tag>}
              {takeoverBy ? (
                <Tag color="error">已接管 · {takeoverBy}</Tag>
              ) : (
                <Tag color="success">自动执行</Tag>
              )}
            </Space>
          }
          extra={
            <Space size={8}>
              {isController ? (
                <Button size="small" danger onClick={handleRelease} disabled={fatalError !== ""}>
                  释放接管
                </Button>
              ) : (
                <Button
                  size="small"
                  type="primary"
                  onClick={handleTakeover}
                  disabled={fatalError !== ""}
                >
                  接管
                </Button>
              )}
              {canContinue && (
                <Button size="small" onClick={handleContinue} disabled={fatalError !== ""}>
                  继续（放行）
                </Button>
              )}
            </Space>
          }
        >
          <LiveCanvas
            executionId={curId}
            frame={frame}
            isController={isController}
            onInput={handleInput}
            overlay={
              <LiveHud
                context={context}
                mode={mode}
                onModeChange={handleModeChange}
                stats={stats}
                hasFrame={frame !== null}
              />
            }
          />
        </Card>
      )}
    </div>
  );
}
