/**
 * PortWriteAuditsPage — v1.51 Phase 127 PWX-04 端口写审计明细页
 *
 * 功能：
 * - 分页表 + 过滤表单（设备 / 端口 / 操作人 / 动作 / 状态 / 时间范围）
 * - 展开行：beforeValue / afterValue JSON 对比 + 设备原始响应折叠 + 失败原因
 *
 * v1.57 可读性优化：
 * - 列表不再展示 deviceId/portId UUID：设备名称（+IP）、接口名（+端口描述）为主信息，
 *   UUID 降级为 Tooltip / 展开行里的追溯标识
 * - 过滤器从"手输 UUID"改为设备下拉（远程搜索名称/IP）+ 端口下拉（按设备联动）
 * - 数据源走 audit 专用 options 端点：审计员通常只有 network:port:write，
 *   直接调 /network/devices/list（需 network:device:list）会 403
 * - 名称缺失（设备已删/端口采集行被清理）显示明确兜底文案，不出现空白单元格
 *
 * 约束：
 * - API 走 @/lib/api 封装（listPortWriteAudits / listPortWriteAudit*Options），无裸 axios
 * - 样式全部 antd token/预设色，零手写暗色覆盖（C13 预算纪律）
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
  Tooltip,
  Descriptions,
} from "antd";
import { SearchOutlined, ReloadOutlined } from "@ant-design/icons";
import type { Dayjs } from "dayjs";
import type { ColumnsType } from "antd/es/table";
import { AppInput, AppSelect, RemoteSelect } from "@/components/ui";
import type { RemoteSelectOption } from "@/components/ui";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import {
  listPortWriteAudits,
  listPortWriteAuditDeviceOptions,
  listPortWriteAuditPortOptions,
} from "@/lib/api/networkApi";
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

/** 选择器默认条数（设备/端口均按此预取首屏，搜索时再远程刷新） */
const OPTION_PAGE_SIZE = 50;

/** JSON 快照格式化（beforeValue/afterValue 为任意 JSON） */
function formatJson(value: unknown): string {
  if (value === undefined || value === null) return "(无记录)";
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

/** 设备选项 label：名称 + IP（重名设备靠 IP 区分） */
function deviceOptionLabel(label: string, ip?: string): string {
  return ip ? `${label}（${ip}）` : label;
}

/** 端口选项 label：接口名 + 描述（同名接口跨设备靠描述辅助辨识） */
function portOptionLabel(label: string, description?: string): string {
  return description ? `${label}（${description}）` : label;
}

/** 单元格副信息行（IP / 端口描述）；无值不占位 */
function SubText({ children }: { children?: React.ReactNode }) {
  if (!children) return null;
  return (
    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
      {children}
    </Typography.Text>
  );
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

  // v1.57 选择器状态：设备选项（首次进入预取首屏）+ 端口选项（随设备联动）
  const [deviceOptions, setDeviceOptions] = useState<RemoteSelectOption[]>([]);
  const [portOptions, setPortOptions] = useState<RemoteSelectOption[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | undefined>();

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
      // 网络错误: post() 拦截器已弹 Toast, 页面空态兜底
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

  // 设备选项首屏预取（RemoteSelect 无关键字时展示 initialOptions）
  useEffect(() => {
    let alive = true;
    listPortWriteAuditDeviceOptions({ current: 1, pageSize: OPTION_PAGE_SIZE })
      .then((res) => {
        if (!alive) return;
        setDeviceOptions(
          (res.list ?? []).map((d) => ({
            label: deviceOptionLabel(d.label, d.ipAddress),
            value: d.value,
          }))
        );
      })
      .catch(() => {
        // 选项失败不阻塞列表：post() 拦截器已提示，选择器留空可手输关键字重试
      });
    return () => {
      alive = false;
    };
  }, []);

  /** 设备远程搜索：名称或 IP 模糊匹配 */
  const fetchDeviceOptions = useCallback(async (keyword: string): Promise<RemoteSelectOption[]> => {
    const res = await listPortWriteAuditDeviceOptions({
      keyword,
      current: 1,
      pageSize: OPTION_PAGE_SIZE,
    });
    return (res.list ?? []).map((d) => ({
      label: deviceOptionLabel(d.label, d.ipAddress),
      value: d.value,
    }));
  }, []);

  /** 端口远程搜索：必须带 deviceId（接口名跨设备重名） */
  const fetchPortOptions = useCallback(
    async (deviceId: string, keyword: string): Promise<RemoteSelectOption[]> => {
      const res = await listPortWriteAuditPortOptions({
        deviceId,
        keyword: keyword || undefined,
        current: 1,
        pageSize: OPTION_PAGE_SIZE,
      });
      return (res.list ?? []).map((p) => ({
        label: portOptionLabel(p.label, p.description),
        value: p.value,
      }));
    },
    []
  );

  /** 设备切换：清空已选端口 + 重新预取该设备端口首屏 */
  const handleDeviceChange = useCallback(
    (deviceId?: string) => {
      setSelectedDeviceId(deviceId);
      form.setFieldValue("portId", undefined);
      setPortOptions([]);
      if (!deviceId) return;
      // 首屏预取该设备端口（RemoteSelect 无关键字时展示 initialOptions；
      // 输入关键字后由 fetcher 防抖远程刷新）
      fetchPortOptions(deviceId, "")
        .then(setPortOptions)
        .catch(() => setPortOptions([]));
    },
    [fetchPortOptions, form]
  );

  /** 端口远程搜索（未选设备时禁用，此处仅做防御） */
  const handlePortSearch = useCallback(
    (keyword: string): Promise<RemoteSelectOption[]> => {
      if (!selectedDeviceId) return Promise.resolve([]);
      return fetchPortOptions(selectedDeviceId, keyword);
    },
    [fetchPortOptions, selectedDeviceId]
  );

  const handleSearch = () => {
    const values = form.getFieldsValue();
    const range = values.timeRange as [Dayjs, Dayjs] | undefined;
    // 契约 30a80c94: 时间为 RFC3339; 分页 current 从 1 起, pageSize 上限 100。
    // v1.57: 仍以 deviceId/portId 精确提交（走既有索引, 不会因同名端口误命中），
    // 名称只用于选择器内检索。
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
    setSelectedDeviceId(undefined);
    setPortOptions([]);
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
      {
        // v1.57：设备名称（主）+ IP（副）；UUID 仅在 Tooltip 内可追溯
        title: "设备",
        dataIndex: "deviceName",
        key: "deviceName",
        width: 200,
        render: (_v: string, row: PortWriteAuditRow) => (
          <Tooltip
            title={
              <span>
                设备 ID：{row.deviceId}
                {row.nameFromSnapshot ? "（名称取自操作当时快照）" : ""}
              </span>
            }
          >
            <div>
              <Typography.Text
                ellipsis
                type={row.deviceName ? undefined : "secondary"}
                style={{ maxWidth: 180 }}
              >
                {row.deviceName || "已删除设备"}
              </Typography.Text>
              <div>
                <SubText>{row.deviceIp}</SubText>
              </div>
            </div>
          </Tooltip>
        ),
      },
      {
        // v1.57：端口接口名（主）+ 描述（副）；UUID 仅在 Tooltip 内可追溯
        title: "端口",
        dataIndex: "interfaceName",
        key: "interfaceName",
        width: 200,
        render: (_v: string, row: PortWriteAuditRow) => (
          <Tooltip title={`端口 ID：${row.portId}`}>
            <div>
              <Typography.Text
                ellipsis
                type={row.interfaceName ? undefined : "secondary"}
                style={{ maxWidth: 180 }}
              >
                {row.interfaceName || "未采集/已删除"}
              </Typography.Text>
              <div>
                <SubText>{row.portDescription}</SubText>
              </div>
            </div>
          </Tooltip>
        ),
      },
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
              <Form.Item name="deviceId" label="设备">
                <RemoteSelect
                  placeholder="搜索设备名称 / IP"
                  allowClear
                  style={{ width: 260 }}
                  initialOptions={deviceOptions}
                  fetcher={fetchDeviceOptions}
                  onChange={(v) => handleDeviceChange(v as string | undefined)}
                />
              </Form.Item>
              <Form.Item
                name="portId"
                label="端口"
                extra={selectedDeviceId ? undefined : "请先选择设备"}
              >
                <RemoteSelect
                  // 设备切换后重建组件，清掉内部缓存的 options 与在途请求
                  key={selectedDeviceId ?? "no-device"}
                  placeholder={selectedDeviceId ? "搜索接口名 / 描述" : "请先选择设备"}
                  allowClear
                  disabled={!selectedDeviceId}
                  style={{ width: 240 }}
                  initialOptions={portOptions}
                  fetcher={handlePortSearch}
                />
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
                    {/* v1.57：展开行顶部保留完整标识（名称 + UUID），便于追溯与复制 */}
                    <Descriptions
                      size="small"
                      column={2}
                      style={{ marginBottom: 12 }}
                      items={[
                        {
                          key: "device",
                          label: "设备",
                          children: row.deviceName
                            ? `${row.deviceName}${row.deviceIp ? `（${row.deviceIp}）` : ""}`
                            : row.deviceId,
                        },
                        {
                          key: "port",
                          label: "端口",
                          children: row.interfaceName
                            ? `${row.interfaceName}${
                                row.portDescription ? `（${row.portDescription}）` : ""
                              }`
                            : row.portId,
                        },
                        { key: "deviceId", label: "设备 ID", children: row.deviceId },
                        { key: "portId", label: "端口 ID", children: row.portId },
                      ]}
                    />
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
