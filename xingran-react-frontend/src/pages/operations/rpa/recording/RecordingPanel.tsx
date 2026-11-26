/**
 * Phase W6 / REC-01：录制面板
 *
 * 交互流（REC-01 验收路径）：
 *   1. 进入面板 → 调 start 录制 → 后端下发 start 指令给 Worker
 *   2. Worker 开页面 + 注入录制器 → 画面帧经 155 liveview 通道流回（rpa_liveview_frame）
 *   3. 员工看着画面操作，录制器自动采集 click/fill/select…
 *   4. 画布上点元素 → pick → 7 级阶梯浮层 → 逐级「可用/不可用」
 *   5. 点「完成录制」→ stop → Worker 产出 actions → 前端轮询拿到 → 填进向导第 2 步
 *
 * 复用纪律（归属协议第 3 条）：
 *   - LiveCanvas / liveFrameBus 只读复用（155 地盘零改动）。
 *   - 画面帧的 executionId 位放 sessionId —— Worker 侧录制会话在 liveview
 *     通道上就是一个「伪 executionId」的会话，前端按 frame.executionId === sessionId 过滤。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Alert, App, Button, Card, Space, Spin, Tag, Typography } from "antd";
import { CloseCircleOutlined, AimOutlined, VideoCameraOutlined } from "@ant-design/icons";
import { LiveCanvas } from "../live/LiveCanvas";
import { installLiveFrameTap, onLiveFrame } from "../live/liveFrameBus";
import type { LiveFrame } from "../live/types";
import type { Action } from "@/types/rpa";
import {
  abortRecording,
  fetchRecordingSession,
  keepaliveRecording,
  pickElement,
  startRecording,
  stopRecording,
} from "./recordingApi";
import type { RecordingPickResult, RecordingResultData, RecordingTier } from "./types";
import { isTerminalRecordingState } from "./types";

const { Text } = Typography;

/** 心跳周期（与 155 liveview 一致）—— 由调用方的轮询节奏复用，本组件无需自带常量 */
/** 会话快照轮询周期 */
const POLL_INTERVAL_MS = 1_500;
/** 等待画面就绪的超时（超时给可读提示，不停会话） */
const FRAME_READY_TIMEOUT_MS = 30_000;

export interface RecordingPanelProps {
  /** 关联任务（可选：产出直接写回该任务脚本） */
  taskId?: string;
  /** 起始页面地址（可选） */
  targetUrl?: string;
  /** 录制完成：把产出动作交给调用方（向导第 2 步回填） */
  onFinish: (actions: Action[], result: RecordingResultData) => void;
  /** 取消录制（关闭面板） */
  onCancel: () => void;
}

/**
 * 把后端产出的动作 JSON 翻译成前端 Action（types/rpa.ts 形状）。
 *
 * 后端 actions 与 worker types.Action 同构：{id,type,description,selector,params,...}。
 * selector 是字符串（worker 侧），前端 Action.selector 是 {type,value} 对象。
 * 导出以便单测（动作回填是录制链路的最后一公里，字段映射错误会让步骤列表全哑）。
 */
// eslint-disable-next-line react-refresh/only-export-components -- 纯函数与组件同文件是有意收拢：toFrontendActions 是本组件独占的 DTO→Action 映射（recordingPanel.test.ts 直接测它）
export function toFrontendActions(raw: Array<Record<string, unknown>> | undefined): Action[] {
  if (!Array.isArray(raw)) return [];
  return raw.map((item, index) => {
    const selectorValue = typeof item.selector === "string" ? item.selector : "";
    const params =
      item.params && typeof item.params === "object"
        ? (item.params as Record<string, unknown>)
        : {};
    return {
      id: typeof item.id === "string" && item.id ? item.id : `recorded_${index + 1}`,
      type: (typeof item.type === "string" ? item.type : "click") as Action["type"],
      description: typeof item.description === "string" ? item.description : undefined,
      selector: selectorValue ? { type: "css", value: selectorValue } : undefined,
      params,
      timeout: typeof item.timeout === "number" ? item.timeout : undefined,
      retry: typeof item.retry === "number" ? item.retry : undefined,
    };
  });
}

/** 拾取浮层：把 7 级阶梯逐级显示「可用 / 不可用」 */
function PickOverlay({
  pick,
  onChoose,
  onClose,
}: {
  pick: RecordingPickResult;
  onChoose: (selector: string) => void;
  onClose: () => void;
}) {
  const tiers: RecordingTier[] = Array.isArray(pick.tiers) ? pick.tiers : [];
  const usable = tiers.filter((t) => t.ok);
  return (
    <Card
      size="small"
      title="选择定位方式（越靠前越稳定）"
      extra={<Button type="text" size="small" icon={<CloseCircleOutlined />} onClick={onClose} />}
      style={{ marginTop: 12 }}
      data-testid="recording-pick-overlay"
    >
      {usable.length === 0 ? (
        <Text type="secondary">{pick.message || "没有找到可用的定位方式"}</Text>
      ) : (
        <Space orientation="vertical" size={6} style={{ width: "100%" }}>
          {usable.map((t) => (
            <Button
              key={t.tier}
              size="small"
              block
              style={{ justifyContent: "space-between", textAlign: "left" }}
              onClick={() => onChoose(t.selector)}
            >
              <Space size={6}>
                <Tag color="green">第 {t.tier} 级</Tag>
                <span>{t.name}</span>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  稳定度 {t.score}
                </Text>
              </Space>
            </Button>
          ))}
        </Space>
      )}
    </Card>
  );
}

export function RecordingPanel({ taskId, targetUrl, onFinish }: RecordingPanelProps) {
  const { message } = App.useApp();

  const [sessionId, setSessionId] = useState("");
  const [state, setState] = useState<string>("created");
  const [frame, setFrame] = useState<LiveFrame | null>(null);
  const [pick, setPick] = useState<RecordingPickResult | null>(null);
  const [fatal, setFatal] = useState("");
  const [starting, setStarting] = useState(true);
  const [stopping, setStopping] = useState(false);

  const sessionRef = useRef("");
  const finishedRef = useRef(false);
  const [hasFrame, setHasFrame] = useState(false);

  // ==================== 1) 开始录制 ====================
  useEffect(() => {
    let cancelled = false;
    // 不在 effect 内同步 setStarting(true)：starting 的 useState 初值已是 true，
    // 这里同步 setState 会触发 react-hooks/set-state-in-effect（级联渲染）。
    // taskId/targetUrl 变化需要重置时由调用方 remount（key 变化）承担。
    startRecording({ taskId, targetUrl })
      .then((res) => {
        if (cancelled) return;
        const data = res.data;
        if (!data?.sessionId) {
          setFatal("开始录制失败：后端未返回会话标识");
          setStarting(false);
          return;
        }
        sessionRef.current = data.sessionId;
        setSessionId(data.sessionId);
        setState(data.state || "starting");
      })
      .catch((error) => {
        if (cancelled) return;
        const msg = (error as { message?: string })?.message || "开始录制失败，请稍后重试";
        setFatal(msg);
        setStarting(false);
      });
    return () => {
      cancelled = true;
    };
  }, [taskId, targetUrl]);

  // ==================== 2) 画面帧接入（复用 live 帧总线，不新建 WS） ====================
  useEffect(() => {
    const releaseTap = installLiveFrameTap();
    const offFrame = onLiveFrame((envelope) => {
      // 只画本会话的画面（帧的 executionId 位放的是 sessionId）
      if (!sessionRef.current || envelope.frame.executionId !== sessionRef.current) return;
      setFrame(envelope.frame);
      setHasFrame(true);
    });
    return () => {
      offFrame();
      releaseTap();
    };
  }, []);

  // ==================== 3) 心跳 + 轮询会话快照 ====================
  useEffect(() => {
    if (!sessionId || fatal) return;

    const poll = setInterval(() => {
      // 心跳
      void keepaliveRecording(sessionId).catch(() => {
        // 心跳失败只记 warn：一次失败不足以判定被回收
      });
      // 轮询快照
      void fetchRecordingSession(sessionId)
        .then((res) => {
          const snap = res.data;
          if (!snap) return;
          setState(snap.state);
          if (snap.lastPick) setPick(snap.lastPick);
          if (isTerminalRecordingState(snap.state)) {
            if (snap.state === "stopped" && snap.result && !finishedRef.current) {
              finishedRef.current = true;
              const actions = toFrontendActions(snap.result.actions);
              if (actions.length === 0) {
                message.warning("没有录到操作步骤，请再试一次");
              }
              onFinish(actions, snap.result);
            } else if (snap.state === "failed") {
              setFatal(snap.message || "录制出错，请再试一次");
            } else if (snap.state === "expired" || snap.state === "aborted") {
              if (!finishedRef.current) {
                setFatal("录制已结束（长时间无操作）");
              }
            }
          }
        })
        .catch(() => {
          // 轮询失败不打断（可能是网络抖动）；连续失败由心跳兜底
        });
    }, POLL_INTERVAL_MS);

    return () => clearInterval(poll);
  }, [sessionId, fatal, message, onFinish]);

  // ==================== 4) 帧就绪超时提示 ====================
  useEffect(() => {
    if (!sessionId || hasFrame || fatal) return;
    const timer = setTimeout(() => {
      if (!hasFrame) {
        message.warning("画面还在准备中，若长时间不显示请联系管理员检查执行机");
      }
    }, FRAME_READY_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [sessionId, hasFrame, fatal, message]);

  // ==================== 5) 卸载时中止会话（尽力而为） ====================
  useEffect(() => {
    return () => {
      const id = sessionRef.current;
      if (id && !finishedRef.current) {
        void abortRecording(id, "关闭录制面板").catch(() => {
          // 尽力而为：后端闲置超时会兜底回收
        });
      }
    };
  }, []);

  // ==================== 操作 ====================
  const handleStop = useCallback(() => {
    const id = sessionRef.current;
    if (!id || stopping) return;
    setStopping(true);
    stopRecording(id)
      // 「正在整理步骤…」这个过渡态不参与渲染（进度看 Card 头部的 state Tag），
      // 停止成功后的收尾由轮询侧的 terminal state 驱动，这里无需额外的 setState。
      .catch((error) => {
        setStopping(false);
        message.error((error as { message?: string })?.message || "结束录制失败");
      });
  }, [message, stopping]);

  const handleCanvasClick = useCallback(
    (command: { x?: number; y?: number }) => {
      const id = sessionRef.current;
      if (!id || !hasFrame) return;
      const x = typeof command.x === "number" ? command.x : 0;
      const y = typeof command.y === "number" ? command.y : 0;
      void pickElement(id, x, y).catch((error) => {
        message.error((error as { message?: string })?.message || "拾取元素失败");
      });
    },
    [hasFrame, message]
  );

  const handleChooseSelector = useCallback(
    (_selector: string) => {
      // 拾取选择器当前只做展示确认（REC-01 的拾取主要用于「看元素定位」；
      // 把选择器写进动作由录制器自身完成 —— 7 级阶梯是录制产出的一部分）。
      setPick(null);
      message.success("已确认元素定位");
    },
    [message]
  );

  const stateTag = useMemo(() => {
    if (fatal) return <Tag color="error">已结束</Tag>;
    if (state === "recording") return <Tag color="processing">正在录制</Tag>;
    if (state === "stopped") return <Tag color="success">已完成</Tag>;
    if (isTerminalRecordingState(state)) return <Tag color="default">已结束</Tag>;
    return <Tag color="default">正在准备</Tag>;
  }, [fatal, state]);

  return (
    <div data-testid="recording-panel">
      {fatal && (
        <Alert
          type="error"
          showIcon
          title={fatal}
          description="录制已结束。可以关闭本窗口重试。"
          style={{ marginBottom: 12 }}
        />
      )}

      <Card
        size="small"
        title={
          <Space size={8} wrap>
            <VideoCameraOutlined />
            <span>录制画面</span>
            {stateTag}
            {hasFrame && state === "recording" && (
              <Tag color="red" data-testid="recording-indicator">
                ● 录制中
              </Tag>
            )}
          </Space>
        }
        extra={
          <Space size={8}>
            <Button
              size="small"
              icon={<AimOutlined />}
              disabled={!hasFrame || state !== "recording"}
              onClick={() => message.info("请在下方画面上直接点击要确认的元素")}
            >
              拾取元素
            </Button>
            <Button
              size="small"
              type="primary"
              data-testid="recording-stop"
              loading={stopping}
              disabled={state !== "recording" && state !== "starting"}
              onClick={handleStop}
            >
              完成录制
            </Button>
          </Space>
        }
      >
        {starting && !fatal ? (
          <div
            style={{
              height: 520,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flexDirection: "column",
              gap: 12,
            }}
          >
            <Spin />
            <Text type="secondary">正在打开录制画面，请稍候…</Text>
          </div>
        ) : (
          <LiveCanvas
            // sessionId 在 liveview 帧通道上扮演 executionId（W6-PLAN §2.2）
            executionId={sessionId || "pending"}
            frame={frame}
            // 录制画面**只可看**：操作注入走「员工操作真实页面」而不是画布点击
            //（员工在自己的电脑上正常操作，录制器采集真实事件）。
            isController={false}
            onInput={handleCanvasClick}
            height={520}
          />
        )}
      </Card>

      {pick && (
        <PickOverlay pick={pick} onChoose={handleChooseSelector} onClose={() => setPick(null)} />
      )}

      <Alert
        type="info"
        showIcon
        style={{ marginTop: 12 }}
        title="录制完成后点「完成录制」，系统会把你的操作整理成步骤清单"
      />
    </div>
  );
}

export default RecordingPanel;
