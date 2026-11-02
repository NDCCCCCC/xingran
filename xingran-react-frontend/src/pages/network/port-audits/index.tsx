/**
 * PortWriteAuditsPage — v1.51 Phase 127 PWX-04 端口写审计明细页
 *
 * 功能：
 * - 分页表 + 过滤表单（设备 ID / 操作人 / 动作 / 状态 / 时间范围）
 * - 展开行：beforeValue / afterValue JSON 对比 + 设备原始响应折叠 + 失败原因
 *
 * 约束：
 * - API 走 @/lib/api 封装（listPortWriteAudits），无裸 axios
 * - 样式全部 antd token/预设色，零手写暗色覆盖（C13 预算纪律）
 * - 后端查询契约未就绪（portwriter 排期），接口报错时以空态兜底；
 *   真机数据联调与 Phase 123 集成验证合并进行
 *
 * 菜单注册：migration_229（幂等，复用 TIDY-01 migration 228 模式），
 * 挂网络设备 M 根下，perms=network:port:write（与写同权限，契约选项之一）。
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Card,
  Form,
  Button,
  Space,
  Table,
  Tag,
  Typography,
  Collapse,
  Alert,
  Row,
  Col,
  DatePicker,
} from "antd";
import { SearchOutlined, ReloadOutlined } from "@ant-design/icons";
import type { Dayjs } from "dayjs";
import type { ColumnsType } from "antd/es/table";
import { AppInput, AppSelect } from "@/components/ui";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { listPortWriteAudits } from "@/lib/api/networkApi";
import type { PortWriteAuditRow } from "@/types/network";
import { ACTION_TITLE } from "@/components/network/port-write/constants";

/** 状态枚举选项（契约 30a80c94: audit.status 值域仅 succeeded|failed；
 * 回滚结果在 sys_port_write_snapshot 表, 不在 audit 行, 二期再做关联展示） */
const STATUS_OPTIONS = [
  { label: "成功", value: "succeeded" },
  { label: "失败", value: "failed" },
];

/** 状态 → Tag 色板（antd 预设色，无手写样式） */
const STATUS_TAG: Record<string, { color: string; label: string }> = {
  succeeded: { color: "green", label: "成功" },
  failed: { color: "red", label: "失败" },
};

/** 动作枚举（与 port-write 域 ACTION_TITLE 同源，零漂移） */
const ACTION_OPTIONS = Object.entries(ACTION_TITLE).map(([value, label]) => ({
  label,
  value,
}));

/** JSON 快照格式化（beforeValue/afterValue 为任意 JSON） */
function formatJson(value: unknown): string {
  if (value === undefined || value === null) return "(无记录)";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

const PortWriteAuditsPage = () => {
  const [form] = Form.useForm();
  // Phase 86 三态门控（页面门禁要求）：审计页为网络设备域只读视图, 用已存在的
  // networkDevice.view 作为页面内查看门控码（permissionCodes.ts 零改动红线；
  // 路由可见性由菜单 perms=network:port:write 在 sys_menu 层控制, 双层互不替代）。
  const permGate = useActionPermission();
  const permView = permGate.decide(PERMISSION_CODES.networkDevice.view);
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<PortWriteAuditRow[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  // 查询参数快照：提交时固化，翻页沿用同一组过滤条件
  const [filters, setFilters] = useState<Record<string, unknown>>({});

  const loadData = useCallback(async (page: number, size: number, f: Record<string, unknown>) => {
    setLoading(true);
    try {
      const result = await listPortWriteAudits({
        current: page,
        pageSize: size,
        ...f,
      });
      setRows(result.list ?? []);
      setTotal(result.total ?? 0);
    } catch {
      // 后端契约未就绪/网络错误: post() 拦截器已弹 Toast, 页面空态兜底
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始 + 翻页/改页大小触发（deps 全为原始值/稳定引用, 无无限循环风险）
  useEffect(() => {
    void loadData(current, pageSize, filters);
  }, [loadData, current, pageSize, filters]);

  const handleSearch = () => {
    const values = form.getFieldsValue();
    const range = values.timeRange as [Dayjs, Dayjs] | undefined;
    // 契约 30a80c94: 时间为 RFC3339; 分页 current 从 1 起, pageSize 上限 100
    const next: Record<string, unknown> = {
      deviceId: values.deviceId || undefined,
      portId: values.portId || undefined,
      operator: values.operator || undefined,
      action: values.action || undefined,
      status: values.status || undefined,
      startTime: range?.[0]?.startOf("day")?.toISOString(),
      endTime: range?.[1]?.endOf("day")?.toISOString(),
    };
    setCurrent(1);
    setFilters(next);
  };

  const handleReset = () => {
    form.resetFields();
    setCurrent(1);
    setFilters({});
  };

  const columns: ColumnsType<PortWriteAuditRow> = useMemo(
    () => [
      {
        title: "时间",
        dataIndex: "createdAt",
        key: "createdAt",
        width: 170,
        render: (v: string) => (v ? v.replace("T", " ").slice(0, 19) : "-"),
      },
      { title: "设备 ID", dataIndex: "deviceId", key: "deviceId", width: 280, ellipsis: true },
      { title: "端口 ID", dataIndex: "portId", key: "portId", width: 280, ellipsis: true },
      {
        title: "动作",
        dataIndex: "action",
        key: "action",
        width: 130,
        render: (v: string) => ACTION_TITLE[v as keyof typeof ACTION_TITLE] ?? v,
      },
      {
        title: "状态",
        dataIndex: "status",
        key: "status",
        width: 100,
        render: (v: string) => {
          const meta = STATUS_TAG[v] ?? { color: "default", label: v };
          return <Tag color={meta.color}>{meta.label}</Tag>;
        },
      },
      { title: "操作人", dataIndex: "operator", key: "operator", width: 120, ellipsis: true },
    ],
    []
  );

  return (
    <div>
      {/* Phase 86 三态门控: ready 无权限 → 不渲染数据区; 未就绪 → loading 提示 */}
      {permView.visible === false && (
        <Card>
          {permView.disabled ? (
            <Alert type="info" showIcon title={permView.hint || "权限信息加载中..."} />
          ) : (
            <Alert
              type="warning"
              showIcon
              title="无查看权限"
              description="您没有查看端口写审计的权限，请联系管理员分配（network:port:write）。"
            />
          )}
        </Card>
      )}

      {permView.visible && (
        <>
          <Card style={{ marginBottom: 16 }}>
            <Form form={form} layout="inline" style={{ flexWrap: "wrap", gap: "8px" }}>
              <Form.Item name="deviceId" label="设备 ID">
                <AppInput placeholder="设备 UUID" allowClear style={{ width: 220 }} />
              </Form.Item>
              <Form.Item name="portId" label="端口 ID">
                <AppInput placeholder="端口 UUID" allowClear style={{ width: 220 }} />
              </Form.Item>
              <Form.Item name="operator" label="操作人">
                <AppInput placeholder="操作人" allowClear style={{ width: 140 }} />
              </Form.Item>
              <Form.Item name="action" label="动作">
                <AppSelect
                  placeholder="全部动作"
                  allowClear
                  options={ACTION_OPTIONS}
                  style={{ width: 160 }}
                  showSearch={false}
                />
              </Form.Item>
              <Form.Item name="status" label="状态">
                <AppSelect
                  placeholder="全部状态"
                  allowClear
                  options={STATUS_OPTIONS}
                  style={{ width: 130 }}
                  showSearch={false}
                />
              </Form.Item>
              <Form.Item name="timeRange" label="时间范围">
                <DatePicker.RangePicker style={{ width: 260 }} />
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                    查询
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={handleReset}>
                    重置
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>

          <Card>
            <Table<PortWriteAuditRow>
              dataSource={rows}
              columns={columns}
              rowKey="id"
              loading={loading}
              size="small"
              pagination={{
                current,
                pageSize,
                total,
                pageSizeOptions: [10, 20, 50, 100], // 契约: pageSize 上限 100
                showSizeChanger: true,
                showTotal: (t) => `共 ${t} 条`,
                onChange: (page, size) => {
                  setCurrent(page);
                  setPageSize(Math.min(size, 100));
                },
              }}
              expandable={{
                expandedRowRender: (row) => (
                  <div>
                    {row.failureReason && (
                      <Alert
                        type="error"
                        showIcon
                        title="失败原因"
                        description={row.failureReason}
                        style={{ marginBottom: 12 }}
                      />
                    )}
                    <Row gutter={16}>
                      <Col span={12}>
                        <Typography.Title level={5}>变更前 (beforeValue)</Typography.Title>
                        <Typography.Text code>
                          <pre style={{ margin: 0, maxHeight: 240, overflow: "auto" }}>
                            {formatJson(row.beforeValue)}
                          </pre>
                        </Typography.Text>
                      </Col>
                      <Col span={12}>
                        <Typography.Title level={5}>变更后 (afterValue)</Typography.Title>
                        <Typography.Text code>
                          <pre style={{ margin: 0, maxHeight: 240, overflow: "auto" }}>
                            {formatJson(row.afterValue)}
                          </pre>
                        </Typography.Text>
                      </Col>
                    </Row>
                    <Collapse
                      style={{ marginTop: 12 }}
                      items={[
                        {
                          key: "device",
                          label: "设备原始响应",
                          children: (
                            <Typography.Text code>
                              <pre style={{ margin: 0, maxHeight: 200, overflow: "auto" }}>
                                {row.deviceResponse || "(无记录)"}
                              </pre>
                            </Typography.Text>
                          ),
                        },
                        {
                          key: "command",
                          label: "下发命令",
                          children: (
                            <Typography.Text code>{row.commandSent || "(无记录)"}</Typography.Text>
                          ),
                        },
                      ]}
                    />
                  </div>
                ),
              }}
            />
          </Card>
        </>
      )}
    </div>
  );
};

export default PortWriteAuditsPage;
