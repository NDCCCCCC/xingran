/**
 * 资产对账 Dashboard 页面 (Phase 42 R1)
 *
 * 内容:
 *   - 5 KPI 卡片(全量资产数 / 未解决异常数 / critical 数 / 7d 新增 / Top1 冲突类型)
 *   - 3 个 ECharts 图表:
 *       1. 饼图 按冲突类型(Type A-F) — 点击扇区 → 跳 /exceptions?type=X
 *       2. 柱状图 按严重级别(low/medium/high/critical) — 点击柱条 → 跳 /exceptions?severity=Y
 *       3. 趋势图 7d 健康度(3 条线: openCount / criticalCount / newCount)
 *
 * 设计要点:
 *   - 所有 KPI 走 useDashboard() 独立 useQuery,**严禁**用 list.length 路径
 *     (memory: stat-cards-from-list-length-capped-at-100)
 *   - 图表 click 事件通过 onEvents 实现 → navigate D-05 双向打通
 *   - 图表用 EChartsWrapper(懒加载,见 memory: vendor-react-bundle-composition)
 */

import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Card, Row, Col, Spin, Empty, Alert } from "antd";
import {
  AppstoreOutlined,
  WarningOutlined,
  AlertOutlined,
  RiseOutlined,
  TrophyOutlined,
} from "@ant-design/icons";
import ReactECharts from "@/components/charts/EChartsWrapper";
import { useDashboard } from "@/hooks/useDashboard";

/** 7d 默认窗口,后续可加 30d/90d 切换 */
const DEFAULT_WINDOW_DAYS = 7;

const SEVERITY_COLORS: Record<string, string> = {
  low: "var(--theme-success, #52c41a)",
  medium: "var(--theme-info, #1890ff)",
  high: "var(--theme-warning, #faad14)",
  critical: "var(--theme-error, #f5222d)",
};

/** 饼图按冲突类型 fallback 顺序(保证 seed merge 6 keys 都在) */
const CONFLICT_TYPE_FALLBACK = ["A", "B", "C", "D", "E", "F"];
const SEVERITY_FALLBACK = ["low", "medium", "high", "critical"];

const Dashboard = () => {
  const navigate = useNavigate();
  const {
    summary,
    byConflictType,
    bySeverity,
    healthTrend,
    isLoading,
    isError,
  } = useDashboard(DEFAULT_WINDOW_DAYS);

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
          itemStyle: { color: "#1890ff" },
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
          itemStyle: { color: "#52c41a" },
        },
      ],
    };
  }, [healthTrend.data]);

  // 饼图 click → /exceptions?type=A|B|C|...
  const onPieClick = (params: { name?: string }) => {
    if (!params?.name) return;
    navigate(`/asset/reconciliation/exceptions?type=${encodeURIComponent(params.name)}`);
  };

  // 柱状图 click → /exceptions?severity=low|medium|high|critical
  const onBarClick = (params: { name?: string }) => {
    if (!params?.name) return;
    navigate(`/asset/reconciliation/exceptions?severity=${encodeURIComponent(params.name)}`);
  };

  // 错误兜底
  if (isError && !isLoading) {
    return (
      <div className="p-6">
        <Alert
          type="error"
          showIcon
          message="对账看板数据加载失败"
          description="请稍后重试或联系管理员检查后端服务"
        />
      </div>
    );
  }

  return (
    <div className="p-6">
      <h2 style={{ marginBottom: 16 }}>资产对账看板</h2>

      {/* 5 KPI 卡片 */}
      <Spin spinning={isLoading}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} md={8} lg={Math.ceil(24 / 5)}>
            <Card>
              <div className="flex items-center gap-3">
                <AppstoreOutlined style={{ fontSize: 32, color: "#1890ff" }} />
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
                <WarningOutlined style={{ fontSize: 32, color: "#faad14" }} />
                <div>
                  <div className="text-2xl font-bold" style={{ color: "#faad14" }}>
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
                <RiseOutlined style={{ fontSize: 32, color: "#52c41a" }} />
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
                  <div className="text-2xl font-bold">
                    {summary.data?.topConflictType || "—"}
                  </div>
                  <div className="text-gray-500">
                    Top1 冲突类型 ({summary.data?.topConflictCount ?? 0})
                  </div>
                </div>
              </div>
            </Card>
          </Col>
        </Row>
      </Spin>

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
