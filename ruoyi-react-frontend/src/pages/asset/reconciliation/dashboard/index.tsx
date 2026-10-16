/**
 * 资产对账 Dashboard 页面 (Phase 42 R1)
 *
 * 内容:
 *   - 5 KPI 卡片(全量资产数 / 未解决异常数 / critical 数 / 7d 新增 / Top1 冲突类型)
 *   - 3 个 ECharts 图表:
 *       1. 饼图 按冲突类型(Type A-F) — 点击扇区 → 跳 /exceptions?type=X
 *       2. 柱状图 按严重级别(low/medium/high/critical) — 点击柱条 → 跳 /exceptions?severity=Y
 *       3. 趋势图 7d 健康度(3 条线: openCount / criticalCount / newCount)
 *   - Phase 44 R3 降噪效果卡片(3 个下降% + 无 baseline 引导 Alert)
 *
 * 设计要点:
 *   - 所有 KPI 走 useDashboard() 独立 useQuery,**严禁**用 list.length 路径
 *     (memory: stat-cards-from-list-length-capped-at-100)
 *   - 图表 click 事件通过 onEvents 实现 → navigate D-05 双向打通
 *   - 图表用 EChartsWrapper(懒加载,见 memory: vendor-react-bundle-composition)
 *   - Phase 44 R3 降噪卡片用 reconciliationApi.baseline.compare()(独立 COUNT 端点,
 *     规避 MaxPageSize=100;queryKey 用 queryKeys.reconciliation.baselineCompare())
 */

import { useMemo } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Card, Row, Col, Spin, Empty, Alert, Badge, Statistic, Tag, App } from "antd";
import {
  AppstoreOutlined,
  WarningOutlined,
  AlertOutlined,
  RiseOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import ReactECharts from "@/components/charts/EChartsWrapper";
import { useDashboard } from "@/hooks/useDashboard";
import { useReconciliationWebSocket } from "@/hooks/useReconciliationWebSocket";
import { reconciliationApi } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";

/** 7d 默认窗口,后续可加 30d/90d 切换 */
const DEFAULT_WINDOW_DAYS = 7;

const SEVERITY_COLORS: Record<string, string> = {
  low: "var(--theme-success, #2d8949)",
  medium: "var(--theme-info, #337ab0)",
  high: "var(--theme-warning, #b07a20)",
  critical: "var(--theme-error, #f5222d)",
};

/** 饼图按冲突类型 fallback 顺序(保证 seed merge 6 keys 都在) */
const CONFLICT_TYPE_FALLBACK = ["A", "B", "C", "D", "E", "F"];
const SEVERITY_FALLBACK = ["low", "medium", "high", "critical"];

const Dashboard = () => {
  const navigate = useNavigate();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { summary, byConflictType, bySeverity, healthTrend, isLoading, isError } =
    useDashboard(DEFAULT_WINDOW_DAYS);

  // Phase 43 R2: 订阅 critical 异常/工单 WS 事件
  // 收到事件后 hook 自动 queryClient.invalidateQueries,这里 onCriticalEvent
  // 只负责给运维一个 toast 提示(可见的反馈,运维不必盯着数据看也能感知到推送)
  const { status: wsStatus } = useReconciliationWebSocket({
    queryClient,
    onCriticalEvent: (event, data) => {
      const labelMap: Record<string, string> = {
        critical_exception_detected: "新 critical 异常",
        critical_workorder_created: "已生成 critical 工单",
      };
      const label = labelMap[event] ?? event;
      const assetHint = data.asset_code || data.conflict_type || "";
      message.info({
        content: `${label}${assetHint ? `: ${assetHint}` : ""}`,
        duration: 3,
      });
    },
  });

  // Phase 44 R3 / Plan 44-02 Task 3 — 降噪基线对比(SC 8 ≥60% 降噪量化验证)
  //
  // 用 queryKeys.reconciliation.baselineCompare() 稳定 queryKey(CLAUDE.md useEffect 稳定性)。
  // CompareBaseline 在无 baseline 时返回 400 → 用 isError 渲染引导 Alert(BLOCKER-3 可观察条件)。
  // 有 baseline 时显示 3 个下降% Statistic(异常 / 工单 / Critical)。
  const baselineCompareQuery = useQuery({
    queryKey: queryKeys.reconciliation.baselineCompare(),
    queryFn: () => reconciliationApi.baseline.compare(),
    retry: false, // 400 不重试(无 baseline 是合法状态,引导用户去 exception-rules 记录)
  });

  // 计算 3 个下降% (baselineCompareQuery.data 字段是后端 BaselineCompareResult snake_case)
  const reductions = useMemo(() => {
    const d = baselineCompareQuery.data;
    if (!d) return null;
    // 后端 BaselineCompareResult 字段: exceptions_reduction_pct / workorders_reduction_pct / critical_reduction_pct
    // (assetApi baseline.compare 返回类型当前用 reductions map, 后端实际返回结构体字段)
    const excPct =
      (d as unknown as { exceptions_reduction_pct?: number }).exceptions_reduction_pct ??
      (d.reductions?.exceptions_reduction_pct as number | undefined) ??
      0;
    const woPct =
      (d as unknown as { workorders_reduction_pct?: number }).workorders_reduction_pct ??
      (d.reductions?.workorders_reduction_pct as number | undefined) ??
      0;
    const critPct =
      (d as unknown as { critical_reduction_pct?: number }).critical_reduction_pct ??
      (d.reductions?.critical_reduction_pct as number | undefined) ??
      0;
    return { excPct, woPct, critPct };
  }, [baselineCompareQuery.data]);

  // 饼图 option — 6 keys(空数据 0)
  const pieOption = useMemo(() => {
    const dataMap = byConflictType.data ?? {};
    const data = CONFLICT_TYPE_FALLBACK.map((key) => ({
      name: key,
      value: Number(dataMap[key] ?? 0),
    }));
    return {
      tooltip: {
        trigger: "item",
        formatter: "{b}: {c} ({d}%)",
      },
      legend: { bottom: 0 },
      series: [
        {
          name: "冲突类型",
          type: "pie",
          radius: ["40%", "70%"],
          avoidLabelOverlap: false,
          itemStyle: { borderRadius: 4, borderColor: "#fff", borderWidth: 2 },
          label: { show: true, formatter: "{b}: {c}" },
          data,
        },
      ],
    };
  }, [byConflictType.data]);

  // 柱状图 option — 4 severity keys
  const barOption = useMemo(() => {
    const dataMap = bySeverity.data ?? {};
    const categories = SEVERITY_FALLBACK;
    const values = categories.map((k) => Number(dataMap[k] ?? 0));
    return {
      tooltip: { trigger: "axis", axisPointer: { type: "shadow" } },
      grid: { left: 40, right: 20, top: 30, bottom: 30 },
      xAxis: { type: "category", data: categories },
      yAxis: { type: "value", minInterval: 1 },
      series: [
        {
          name: "严重级别",
          type: "bar",
          data: values.map((v, i) => ({
            value: v,
            itemStyle: { color: SEVERITY_COLORS[categories[i]] },
          })),
          barWidth: "50%",
          label: { show: true, position: "top" },
        },
      ],
    };
  }, [bySeverity.data]);

  // 趋势图 option — 3 条线(openCount / criticalCount / newCount)
  const lineOption = useMemo(() => {
    const data = healthTrend.data ?? [];
    return {
      tooltip: { trigger: "axis" },
      legend: { top: 0 },
      grid: { left: 40, right: 20, top: 40, bottom: 30 },
      xAxis: { type: "category", data: data.map((p) => p.date) },
      yAxis: { type: "value", minInterval: 1 },
      series: [
        {
          name: "未解决",
          type: "line",
          smooth: true,
          data: data.map((p) => p.openCount),
          itemStyle: { color: "#337ab0" },
        },
        {
          name: "严重未解决",
          type: "line",
          smooth: true,
          data: data.map((p) => p.criticalCount),
          itemStyle: { color: "#f5222d" },
        },
        {
          name: "新增",
          type: "line",
          smooth: true,
          data: data.map((p) => p.newCount),
          itemStyle: { color: "#2d8949" },
        },
      ],
    };
  }, [healthTrend.data]);

  // 饼图 click → /assets/exceptions?type=A|B|C|...
  // 路径必须与 sys_menu 父菜单 path='assets' + 子菜单 path='exceptions' 拼接一致;
  // routeGenerator.resolvePath 会拼成 'assets/exceptions'。原代码写成
  // '/asset/reconciliation/exceptions' 是单数 + 长路径,React Router 找不到 → fallback /dashboard。
  // D-05 双向打通:饼图 click 应跳到异常列表并用 type 预填筛选。
  const onPieClick = (params: { name?: string }) => {
    if (!params?.name) return;
    navigate(`/assets/exceptions?type=${encodeURIComponent(params.name)}`);
  };

  // 柱状图 click → /assets/exceptions?severity=low|medium|high|critical
  // 同上:路径必须是前端动态路由注册的实际路径,后端 API 路径 /api/v1/... 不可用。
  const onBarClick = (params: { name?: string }) => {
    if (!params?.name) return;
    navigate(`/assets/exceptions?severity=${encodeURIComponent(params.name)}`);
  };

  // 错误兜底
  if (isError && !isLoading) {
    return (
      <div className="p-6">
        <Alert
          type="error"
          showIcon
          title="对账看板数据加载失败"
          description="请稍后重试或联系管理员检查后端服务"
        />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 style={{ margin: 0 }}>资产对账看板</h2>
        {/* Phase 43 R2: WS 连接状态 Badge,运维一眼看到实时推送是否就绪 */}
        <Badge
          status={
            wsStatus === "connected"
              ? "success"
              : wsStatus === "connecting"
                ? "processing"
                : wsStatus === "error"
                  ? "error"
                  : "default"
          }
          text={
            wsStatus === "connected"
              ? "实时推送已连接"
              : wsStatus === "connecting"
                ? "正在连接..."
                : wsStatus === "error"
                  ? "推送连接异常"
                  : "推送已断开"
          }
        />
      </div>

      {/* 5 KPI 卡片 */}
      <Spin spinning={isLoading}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={8} lg={Math.ceil(24 / 5)}>
            <Card>
              <div className="flex items-center gap-3">
                <AppstoreOutlined style={{ fontSize: 32, color: "#337ab0" }} />
                <div>
                  <div className="text-2xl font-bold">{summary.data?.totalAssets ?? 0}</div>
                  <div className="text-gray-500">全量资产数</div>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={Math.ceil(24 / 5)}>
            <Card>
              <div className="flex items-center gap-3">
                <WarningOutlined style={{ fontSize: 32, color: "#b07a20" }} />
                <div>
                  <div className="text-2xl font-bold" style={{ color: "#b07a20" }}>
                    {summary.data?.openExceptions ?? 0}
                  </div>
                  <div className="text-gray-500">未解决异常数</div>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={Math.ceil(24 / 5)}>
            <Card>
              <div className="flex items-center gap-3">
                <AlertOutlined style={{ fontSize: 32, color: "#f5222d" }} />
                <div>
                  <div className="text-2xl font-bold" style={{ color: "#f5222d" }}>
                    {summary.data?.criticalOpen ?? 0}
                  </div>
                  <div className="text-gray-500">critical 级未解决</div>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={Math.ceil(24 / 5)}>
            <Card>
              <div className="flex items-center gap-3">
                <RiseOutlined style={{ fontSize: 32, color: "#2d8949" }} />
                <div>
                  <div className="text-2xl font-bold">{summary.data?.last7dNew ?? 0}</div>
                  <div className="text-gray-500">7d 新增异常数</div>
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12} md={8} lg={Math.ceil(24 / 5)}>
            <Card>
              <div className="flex items-center gap-3">
                <TrophyOutlined style={{ fontSize: 32, color: "#722ed1" }} />
                <div>
                  <div className="text-2xl font-bold">{summary.data?.topConflictType || "—"}</div>
                  <div className="text-gray-500">
                    Top1 冲突类型 ({summary.data?.topConflictCount ?? 0})
                  </div>
                </div>
              </div>
            </Card>
          </Col>
        </Row>
      </Spin>

      {/* Phase 44 R3 / Plan 44-02 Task 3 — 降噪效果卡片(SC 8 ≥60% 降噪量化验证)
       * 前置:运维必须先在 /assets/exception-rules 点"记录当前为基线"按钮记录 R2 末期基线
       * 无 baseline 时 compare 返回 400 → 显示 Alert 引导(BLOCKER-3 可观察条件)
       * 有 baseline 时显示 3 个下降% Statistic, ≥60% 绿色达标 + Tag, <60% 橙色未达标 */}
      <Card
        title="降噪效果(SC 8 ≥60% 降噪验证)"
        style={{ marginTop: 16 }}
        extra={
          reductions && (
            <Tag
              color={
                reductions.excPct >= 60 && reductions.woPct >= 60 && reductions.critPct >= 60
                  ? "success"
                  : "warning"
              }
            >
              {reductions.excPct >= 60 && reductions.woPct >= 60 && reductions.critPct >= 60
                ? "达标"
                : "未达标"}
            </Tag>
          )
        }
      >
        {baselineCompareQuery.isLoading ? (
          <Spin />
        ) : baselineCompareQuery.isError ? (
          // 无 baseline 时引导运维去 exception-rules 页记录(BLOCKER-3 可观察条件)
          <Alert
            type="info"
            showIcon
            title="请先到例外规则管理页记录基线"
            description={
              <>
                R3 部署前 + R2 数据保留期内必须调用 Snapshot 记录 R2 末期基线,否则 SC 8 ≥60%
                降噪不可量化验证。前往 <Link to="/assets/exception-rules">例外规则管理</Link>{" "}
                点击"记录当前为基线"按钮。
              </>
            }
          />
        ) : reductions ? (
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={8}>
              <Statistic
                title="异常下降%"
                value={reductions.excPct}
                precision={1}
                suffix="%"
                styles={{ content: { color: reductions.excPct >= 60 ? "#2d8949" : "#b07a20" } }}
              />
              <small style={{ color: "var(--theme-text-secondary, #888)" }}>
                baseline {baselineCompareQuery.data?.baseline?.total_exceptions ?? 0} → current{" "}
                {baselineCompareQuery.data?.current?.total_exceptions ?? 0}
              </small>
            </Col>
            <Col xs={24} sm={8}>
              <Statistic
                title="工单下降%"
                value={reductions.woPct}
                precision={1}
                suffix="%"
                styles={{ content: { color: reductions.woPct >= 60 ? "#2d8949" : "#b07a20" } }}
              />
              <small style={{ color: "var(--theme-text-secondary, #888)" }}>
                baseline {baselineCompareQuery.data?.baseline?.total_workorders ?? 0} → current{" "}
                {baselineCompareQuery.data?.current?.total_workorders ?? 0}
              </small>
            </Col>
            <Col xs={24} sm={8}>
              <Statistic
                title="Critical 下降%"
                value={reductions.critPct}
                precision={1}
                suffix="%"
                styles={{ content: { color: reductions.critPct >= 60 ? "#2d8949" : "#b07a20" } }}
              />
              <small style={{ color: "var(--theme-text-secondary, #888)" }}>
                baseline {baselineCompareQuery.data?.baseline?.critical_exceptions ?? 0} → current{" "}
                {baselineCompareQuery.data?.current?.critical_exceptions ?? 0}
              </small>
            </Col>
          </Row>
        ) : null}
      </Card>

      {/* 3 个图表 */}
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={8}>
          <Card title="冲突类型分布" loading={byConflictType.isLoading}>
            {byConflictType.data && Object.keys(byConflictType.data).length > 0 ? (
              <ReactECharts
                option={pieOption}
                style={{ height: 320 }}
                onEvents={{ click: onPieClick }}
                notMerge
                lazyUpdate
              />
            ) : (
              <Empty description="暂无数据" />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="严重级别分布" loading={bySeverity.isLoading}>
            {bySeverity.data && Object.keys(bySeverity.data).length > 0 ? (
              <ReactECharts
                option={barOption}
                style={{ height: 320 }}
                onEvents={{ click: onBarClick }}
                notMerge
                lazyUpdate
              />
            ) : (
              <Empty description="暂无数据" />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title={`${DEFAULT_WINDOW_DAYS}d 健康度趋势`} loading={healthTrend.isLoading}>
            {healthTrend.data && healthTrend.data.length > 0 ? (
              <ReactECharts option={lineOption} style={{ height: 320 }} notMerge lazyUpdate />
            ) : (
              <Empty description="暂无数据" />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
