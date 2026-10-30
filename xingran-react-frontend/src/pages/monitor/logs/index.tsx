/**
 * 日志管理页面
 * Log Monitor Page
 */

import React, { useState, useEffect, useCallback, useMemo } from "react";
import { useLocation, useSearchParams } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Card,
  Table,
  Button,
  Space,
  DatePicker,
  Row,
  Col,
  Tabs,
  Modal,
  Typography,
  Form,
} from "antd";
// Phase 71 (W1)：裸 Input/Select/TextArea → 契约组件
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  FileTextOutlined,
  UserOutlined,
} from "@ant-design/icons";

import type { OperLog, LoginLog, SearchFormState } from "./types";
import { BUSINESS_TYPE_OPTIONS, LOG_STATUS_OPTIONS, LOGIN_STATUS_OPTIONS } from "./constants";
import { formatLocalTime, renderLogStatusTag } from "./utils";
import { getOperLogColumns, getLoginLogColumns } from "./columns";
import { useTableManager } from "@/hooks/useTableManager";
import type { BaseResponse, PageResponse } from "@/types";
import { post } from "@/lib/api";
import { createSorterMeta } from "@/utils/tableHelpers";
import { confirmDelete } from "@/components/shared";

const { Title } = Typography;
const { RangePicker } = DatePicker;

// ==================== 数据加载函数 ====================

// 加载操作日志
const loadOperLogs = async (
  params: SearchFormState & { current?: number; pageSize?: number }
): Promise<{ list: OperLog[]; total: number }> => {
  const requestParams: Record<string, unknown> = {
    ...params,
  };

  // 处理时间范围
  if (params.timeRange && params.timeRange.length === 2) {
    requestParams.startTime = params.timeRange[0].toISOString();
    requestParams.endTime = params.timeRange[1].toISOString();
    delete requestParams.timeRange;
  }

  const result = await post<PageResponse<OperLog>>("/monitor/oper-logs/list", requestParams);

  return {
    list: result.data?.list || [],
    total: result.data?.total || 0,
  };
};

// 加载登录日志
const loadLoginLogs = async (
  params: SearchFormState & { current?: number; pageSize?: number }
): Promise<{ list: LoginLog[]; total: number }> => {
  const requestParams: Record<string, unknown> = {
    ...params,
  };

  // 处理时间范围
  if (params.timeRange && params.timeRange.length === 2) {
    requestParams.startTime = params.timeRange[0].toISOString();
    requestParams.endTime = params.timeRange[1].toISOString();
    delete requestParams.timeRange;
  }

  const result = await post<PageResponse<LoginLog>>("/monitor/login-logs/list", requestParams);

  return {
    list: result.data?.list || [],
    total: result.data?.total || 0,
  };
};

// ==================== 主组件 ====================

const LogMonitor: React.FC = () => {
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const [activeTab, setActiveTab] = usePersistedStateController<"oper" | "login">({
    keyPrefix: location.pathname,
    keySuffix: "activeTab",
    defaultValue: "oper",
  });

  // 服务端排序:field 对应后端 operLogAllowedSortFields 白名单 key
  const operSorterMetas = useMemo(
    () => [
      createSorterMeta<OperLog>("title"),
      createSorterMeta<OperLog>("businessType"),
      createSorterMeta<OperLog>("operName"),
      createSorterMeta<OperLog>("status"),
      createSorterMeta<OperLog>("operTime", "date"),
    ],
    []
  );

  // 服务端排序:field 对应后端 loginLogAllowedSortFields 白名单 key
  const loginSorterMetas = useMemo(
    () => [
      createSorterMeta<LoginLog>("userName"),
      createSorterMeta<LoginLog>("ipAddr"),
      createSorterMeta<LoginLog>("status"),
      createSorterMeta<LoginLog>("loginTime", "date"),
    ],
    []
  );

  // 操作日志管理器 - 不传入 pageSize，使用用户设置中的默认值
  const operLogManager = useTableManager<OperLog>(loadOperLogs, {
    sorterMetas: operSorterMetas,
    onSuccess: () => {
      // 可选：操作成功后的回调
    },
  });

  // 登录日志管理器 - 不传入 pageSize，使用用户设置中的默认值
  const loginLogManager = useTableManager<LoginLog>(loadLoginLogs, {
    sorterMetas: loginSorterMetas,
    onSuccess: () => {
      // 可选：操作成功后的回调
    },
  });

  // 详情模态框状态
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedLog, setSelectedLog] = useState<OperLog | LoginLog | null>(null);

  // 根据当前标签页选择对应的管理器
  const currentManager = activeTab === "oper" ? operLogManager : loginLogManager;

  // 初始化加载
  useEffect(() => {
    if (activeTab === "oper") {
      operLogManager.loadData();
    } else {
      loginLogManager.loadData();
    }
    // 仅在 activeTab 切换时重新加载;manager 引用变化不应触发
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab]);

  // Phase 53 W4 D-10/LANDMINE #2: mount-only 读 URL ?module=xxx 预填 title 字段并触发查询。
  // 来源: 端口写操作 Toast 的"查看审计日志"链接 navigate('/monitor/logs?module=端口管理')。
  // 后端 sys_oper_log.title 列对应 PortWrite handler 的 ModulePortWrite 常量, LIKE 过滤无需后端改动。
  // CLAUDE.md useEffect 纪律: mount-only [] deps, eslint-disable exhaustive-deps (参考 reconciliation/exceptions:172)
  useEffect(() => {
    const moduleFromUrl = searchParams.get("module");
    if (moduleFromUrl && activeTab === "oper") {
      operLogManager.searchForm.setFieldsValue({ title: moduleFromUrl });
      operLogManager.handleSearch();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 查看详情
  const handleViewDetail = useCallback((log: OperLog | LoginLog) => {
    setSelectedLog(log);
    setDetailModalVisible(true);
  }, []);

  // 清空日志（68.1-T4 §1.9 非删除破坏性变体）
  const handleClearLogs = useCallback(() => {
    confirmDelete({
      title: "确认清空",
      target: activeTab === "oper" ? "全部操作日志" : "全部登录日志",
      actionLabel: "清空",
      actionContext: "清空日志",
      contentText: "将{target}，操作不可恢复，是否继续？",
      successText: "已清空日志",
      onDelete: async () => {
        const endpoint =
          activeTab === "oper" ? "/monitor/oper-logs/clear" : "/monitor/login-logs/clear";
        await post<BaseResponse<unknown>>(endpoint, {});
        currentManager.handleRefresh();
      },
    });
  }, [activeTab, currentManager]);

  // 刷新
  const handleRefresh = useCallback(() => {
    currentManager.handleRefresh();
  }, [currentManager]);

  // 表格列
  const operColumns = useMemo(
    () =>
      getOperLogColumns({
        handleViewDetail,
        getColumnSortOrder: operLogManager.getColumnSortOrder,
      }),
    [handleViewDetail, operLogManager.getColumnSortOrder]
  );
  const loginColumns = useMemo(
    () =>
      getLoginLogColumns({
        handleViewDetail,
        getColumnSortOrder: loginLogManager.getColumnSortOrder,
      }),
    [handleViewDetail, loginLogManager.getColumnSortOrder]
  );

  // 搜索
  const handleSearch = useCallback(() => {
    currentManager.handleSearch();
  }, [currentManager]);

  // 重置
  const handleReset = useCallback(() => {
    currentManager.handleReset();
  }, [currentManager]);

  // 标签页变化
  const handleTabChange = (key: string) => {
    setActiveTab(key as "oper" | "login");
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">日志管理</h1>

        {/* 操作日志搜索表单 */}
        {activeTab === "oper" && (
          <Card>
            <Form form={operLogManager.searchForm} layout="inline">
              <Row gutter={16}>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="title">
                    <AppInput placeholder="操作模块" allowClear className="user-form-input" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="businessType">
                    <AppSelect
                      placeholder="业务类型"
                      allowClear
                      className="user-form-input"
                      style={{ width: "100%" }}
                      options={BUSINESS_TYPE_OPTIONS}
                      onSearch={() => {}}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="status">
                    <AppSelect
                      placeholder="操作状态"
                      allowClear
                      className="user-form-input"
                      style={{ width: "100%" }}
                      options={LOG_STATUS_OPTIONS}
                      onSearch={() => {}}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="operName">
                    <AppInput placeholder="操作人员" allowClear className="user-form-input" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={16} md={6}>
                  <Form.Item name="timeRange">
                    <RangePicker
                      showTime
                      placeholder={["开始时间", "结束时间"]}
                      style={{ width: "100%" }}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Space>
                    <Button
                      type="primary"
                      icon={<SearchOutlined />}
                      onClick={handleSearch}
                      aria-label="搜索"
                      title="搜索"
                    >
                      搜索
                    </Button>
                    <Button onClick={handleReset}>重置</Button>
                    <Button
                      icon={<ReloadOutlined />}
                      onClick={handleRefresh}
                      aria-label="刷新"
                      title="刷新"
                    >
                      刷新
                    </Button>
                    <Button
                      danger
                      icon={<DeleteOutlined />}
                      onClick={handleClearLogs}
                      aria-label="清空日志"
                      title="清空日志"
                    >
                      清空
                    </Button>
                  </Space>
                </Col>
              </Row>
            </Form>
          </Card>
        )}

        {/* 登录日志搜索表单 */}
        {activeTab === "login" && (
          <Card>
            <Form form={loginLogManager.searchForm} layout="inline">
              <Row gutter={16}>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="userName">
                    <AppInput placeholder="用户名称" allowClear className="user-form-input" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="ipAddr">
                    <AppInput placeholder="IP地址" allowClear className="user-form-input" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="status">
                    <AppSelect
                      placeholder="登录状态"
                      allowClear
                      className="user-form-input"
                      style={{ width: "100%" }}
                      options={LOGIN_STATUS_OPTIONS}
                      onSearch={() => {}}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={16} md={6}>
                  <Form.Item name="timeRange">
                    <RangePicker
                      showTime
                      placeholder={["开始时间", "结束时间"]}
                      style={{ width: "100%" }}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Space>
                    <Button
                      type="primary"
                      icon={<SearchOutlined />}
                      onClick={handleSearch}
                      aria-label="搜索"
                      title="搜索"
                    >
                      搜索
                    </Button>
                    <Button onClick={handleReset}>重置</Button>
                    <Button
                      icon={<ReloadOutlined />}
                      onClick={handleRefresh}
                      aria-label="刷新"
                      title="刷新"
                    >
                      刷新
                    </Button>
                    <Button
                      danger
                      icon={<DeleteOutlined />}
                      onClick={handleClearLogs}
                      aria-label="清空日志"
                      title="清空日志"
                    >
                      清空
                    </Button>
                  </Space>
                </Col>
              </Row>
            </Form>
          </Card>
        )}
      </div>

      {/* 日志列表 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          items={[
            {
              key: "oper",
              label: (
                <span>
                  <FileTextOutlined />
                  操作日志
                </span>
              ),
              children: (
                <Table
                  columns={operColumns}
                  dataSource={operLogManager.data}
                  rowKey="id"
                  loading={operLogManager.loading}
                  virtual
                  pagination={{
                    current: operLogManager.current,
                    pageSize: operLogManager.pageSize,
                    total: operLogManager.total,
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total) => `共 ${total} 条记录`,
                  }}
                  onChange={operLogManager.handleTableChange}
                  scroll={{ x: 1500, y: 600 }}
                />
              ),
            },
            {
              key: "login",
              label: (
                <span>
                  <UserOutlined />
                  登录日志
                </span>
              ),
              children: (
                <Table
                  columns={loginColumns}
                  dataSource={loginLogManager.data}
                  rowKey="id"
                  loading={loginLogManager.loading}
                  virtual
                  pagination={{
                    current: loginLogManager.current,
                    pageSize: loginLogManager.pageSize,
                    total: loginLogManager.total,
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total) => `共 ${total} 条记录`,
                  }}
                  onChange={loginLogManager.handleTableChange}
                  scroll={{ x: 1400, y: 600 }}
                />
              ),
            },
          ]}
        />
      </Card>

      {/* 详情模态框 */}
      <Modal
        title="日志详情"
        mask={{ closable: false }}
        keyboard={false}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
        width={800}
      >
        {selectedLog && (
          <div>
            <Title level={5}>基本信息</Title>
            <div className="mb-4 space-y-2">
              {activeTab === "oper" ? (
                <>
                  <div>
                    <strong>日志编号：</strong>
                    {(selectedLog as OperLog).id}
                  </div>
                  <div>
                    <strong>操作模块：</strong>
                    {(selectedLog as OperLog).title}
                  </div>
                  <div>
                    <strong>请求方式：</strong>
                    {(selectedLog as OperLog).requestMethod}
                  </div>
                  <div>
                    <strong>操作人员：</strong>
                    {(selectedLog as OperLog).operName}
                  </div>
                  <div>
                    <strong>部门名称：</strong>
                    {(selectedLog as OperLog).deptName || "-"}
                  </div>
                  <div>
                    <strong>操作地址：</strong>
                    {(selectedLog as OperLog).operUrl}
                  </div>
                  <div>
                    <strong>操作IP：</strong>
                    {(selectedLog as OperLog).operIp}
                  </div>
                  <div>
                    <strong>操作地点：</strong>
                    {(selectedLog as OperLog).operLocation}
                  </div>
                  <div>
                    <strong>操作状态：</strong>
                    {renderLogStatusTag((selectedLog as OperLog).status, "oper")}
                  </div>
                  <div>
                    <strong>操作时间：</strong>
                    {formatLocalTime((selectedLog as OperLog).operTime)}
                  </div>
                </>
              ) : (
                <>
                  <div>
                    <strong>访问编号：</strong>
                    {(selectedLog as LoginLog).id}
                  </div>
                  <div>
                    <strong>用户名称：</strong>
                    {(selectedLog as LoginLog).userName}
                  </div>
                  <div>
                    <strong>登录IP：</strong>
                    {(selectedLog as LoginLog).ipAddr}
                  </div>
                  <div>
                    <strong>登录地点：</strong>
                    {(selectedLog as LoginLog).loginLocation}
                  </div>
                  <div>
                    <strong>浏览器：</strong>
                    {(selectedLog as LoginLog).browser || "-"}
                  </div>
                  <div>
                    <strong>操作系统：</strong>
                    {(selectedLog as LoginLog).os || "-"}
                  </div>
                  <div>
                    <strong>登录状态：</strong>
                    {renderLogStatusTag((selectedLog as LoginLog).status, "login")}
                  </div>
                  <div>
                    <strong>操作信息：</strong>
                    {(selectedLog as LoginLog).message}
                  </div>
                  <div>
                    <strong>登录时间：</strong>
                    {formatLocalTime((selectedLog as LoginLog).loginTime)}
                  </div>
                </>
              )}
            </div>

            {(selectedLog as OperLog).operParam && (
              <>
                <Title level={5}>请求参数</Title>
                <AppTextArea
                  value={(selectedLog as OperLog).operParam}
                  rows={10}
                  readOnly
                  // PITFALLS 1.2：只读展示区（受控 value、无 onChange）——契约默认 allowClear
                  // 会渲染一个点了既清不掉内容也无副作用的清空按钮，属纯 UI 倒退，显式关闭。
                  allowClear={false}
                  className="mb-4"
                />
              </>
            )}

            {(selectedLog as OperLog).errorMessage && (
              <>
                <Title level={5}>错误信息</Title>
                <AppTextArea
                  value={(selectedLog as OperLog).errorMessage}
                  rows={5}
                  readOnly
                  // 同上：只读展示区，显式关闭契约默认 allowClear
                  allowClear={false}
                  className="text-red-500"
                />
              </>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default LogMonitor;
