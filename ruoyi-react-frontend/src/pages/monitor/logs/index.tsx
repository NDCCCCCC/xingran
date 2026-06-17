/**
 * 日志管理页面
 * Log Monitor Page
 */

import React, { useState, useEffect, useCallback } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Card,
  Table,
  Button,
  Space,
  Input,
  Select,
  DatePicker,
  Row,
  Col,
  Tabs,
  Modal,
  Typography,
  Form,
} from "antd";
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  FileTextOutlined,
  UserOutlined,
} from "@ant-design/icons";

import type { OperLog, LoginLog } from "./types";
import { BUSINESS_TYPE_OPTIONS, LOG_STATUS_OPTIONS, LOGIN_STATUS_OPTIONS } from "./constants";
import { formatLocalTime, renderLogStatusTag } from "./utils";
import { getOperLogColumns, getLoginLogColumns } from "./columns";
import { useTableManager } from "@/hooks/useTableManager";
import type { BaseResponse, PageResponse } from "@/types";
import { post } from "@/lib/api";
import { message } from "antd";

const { Title } = Typography;
const { RangePicker } = DatePicker;

// ==================== 数据加载函数 ====================

// 加载操作日志
const loadOperLogs = async (params: any): Promise<{ list: OperLog[]; total: number }> => {
  const requestParams: any = {
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
const loadLoginLogs = async (params: any): Promise<{ list: LoginLog[]; total: number }> => {
  const requestParams: any = {
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
  const [activeTab, setActiveTab] = usePersistedStateController<"oper" | "login">({
    keyPrefix: location.pathname,
    keySuffix: "activeTab",
    defaultValue: "oper",
  });

  // 操作日志管理器 - 不传入 pageSize，使用用户设置中的默认值
  const operLogManager = useTableManager<OperLog>(loadOperLogs, {
    onSuccess: () => {
      // 可选：操作成功后的回调
    },
  });

  // 登录日志管理器 - 不传入 pageSize，使用用户设置中的默认值
  const loginLogManager = useTableManager<LoginLog>(loadLoginLogs, {
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

  // 查看详情
  const handleViewDetail = useCallback((log: OperLog | LoginLog) => {
    setSelectedLog(log);
    setDetailModalVisible(true);
  }, []);

  // 清空日志
  const handleClearLogs = useCallback(async () => {
    Modal.confirm({
      title: "确认清空",
      content: `确定要清空${activeTab === "oper" ? "操作" : "登录"}日志吗？此操作不可恢复！`,
      onOk: async () => {
        try {
          const endpoint = activeTab === "oper" ? "/monitor/oper-logs/clear" : "/monitor/login-logs/clear";
          await post<BaseResponse<any>>(endpoint, {});
          message.success("清空成功");
          currentManager.handleRefresh();
        } catch (error) {
          console.error("清空日志失败:", error);
          message.error("清空失败");
        }
      },
    });
  }, [activeTab, currentManager]);

  // 刷新
  const handleRefresh = useCallback(() => {
    currentManager.handleRefresh();
  }, [currentManager]);

  // 表格列
  const operColumns = getOperLogColumns({ handleViewDetail });
  const loginColumns = getLoginLogColumns({ handleViewDetail });

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
                    <Input placeholder="操作模块" allowClear className="user-form-input" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="businessType">
                    <Select
                      placeholder="业务类型"
                      allowClear
                      className="user-form-input"
                      style={{ width: "100%" }}
                      options={BUSINESS_TYPE_OPTIONS}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="status">
                    <Select
                      placeholder="操作状态"
                      allowClear
                      className="user-form-input"
                      style={{ width: "100%" }}
                      options={LOG_STATUS_OPTIONS}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="operName">
                    <Input placeholder="操作人员" allowClear className="user-form-input" />
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
                    <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                      搜索
                    </Button>
                    <Button onClick={handleReset}>重置</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                      刷新
                    </Button>
                    <Button danger icon={<DeleteOutlined />} onClick={handleClearLogs}>
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
                    <Input placeholder="用户名称" allowClear className="user-form-input" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="ipAddr">
                    <Input placeholder="IP地址" allowClear className="user-form-input" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={8} md={6}>
                  <Form.Item name="status">
                    <Select
                      placeholder="登录状态"
                      allowClear
                      className="user-form-input"
                      style={{ width: "100%" }}
                      options={LOGIN_STATUS_OPTIONS}
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
                    <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                      搜索
                    </Button>
                    <Button onClick={handleReset}>重置</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                      刷新
                    </Button>
                    <Button danger icon={<DeleteOutlined />} onClick={handleClearLogs}>
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
                  pagination={{
                    current: operLogManager.current,
                    pageSize: operLogManager.pageSize,
                    total: operLogManager.total,
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total) => `共 ${total} 条记录`,
                    onChange: (page, size) => {
                      operLogManager.setCurrent(page);
                      if (size !== operLogManager.pageSize) {
                        operLogManager.setPageSize(size);
                      }
                      operLogManager.loadData();
                    },
                  }}
                  scroll={{ x: 1500 }}
                />
              )
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
                  pagination={{
                    current: loginLogManager.current,
                    pageSize: loginLogManager.pageSize,
                    total: loginLogManager.total,
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total) => `共 ${total} 条记录`,
                    onChange: (page, size) => {
                      loginLogManager.setCurrent(page);
                      if (size !== loginLogManager.pageSize) {
                        loginLogManager.setPageSize(size);
                      }
                      loginLogManager.loadData();
                    },
                  }}
                  scroll={{ x: 1400 }}
                />
              )
            }
          ]}
        />
      </Card>

      {/* 详情模态框 */}
      <Modal
        title="日志详情"
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
                  <div><strong>日志编号：</strong>{(selectedLog as OperLog).id}</div>
                  <div><strong>操作模块：</strong>{(selectedLog as OperLog).title}</div>
                  <div><strong>请求方式：</strong>{(selectedLog as OperLog).requestMethod}</div>
                  <div><strong>操作人员：</strong>{(selectedLog as OperLog).operName}</div>
                  <div><strong>部门名称：</strong>{(selectedLog as OperLog).deptName || "-"}</div>
                  <div><strong>操作地址：</strong>{(selectedLog as OperLog).operUrl}</div>
                  <div><strong>操作IP：</strong>{(selectedLog as OperLog).operIp}</div>
                  <div><strong>操作地点：</strong>{(selectedLog as OperLog).operLocation}</div>
                  <div><strong>操作状态：</strong>
                    {renderLogStatusTag((selectedLog as OperLog).status, "oper")}
                  </div>
                  <div><strong>操作时间：</strong>{formatLocalTime((selectedLog as OperLog).operTime)}</div>
                </>
              ) : (
                <>
                  <div><strong>访问编号：</strong>{(selectedLog as LoginLog).id}</div>
                  <div><strong>用户名称：</strong>{(selectedLog as LoginLog).userName}</div>
                  <div><strong>登录IP：</strong>{(selectedLog as LoginLog).ipAddr}</div>
                  <div><strong>登录地点：</strong>{(selectedLog as LoginLog).loginLocation}</div>
                  <div><strong>浏览器：</strong>{(selectedLog as LoginLog).browser || "-"}</div>
                  <div><strong>操作系统：</strong>{(selectedLog as LoginLog).os || "-"}</div>
                  <div><strong>登录状态：</strong>
                    {renderLogStatusTag((selectedLog as LoginLog).status, "login")}
                  </div>
                  <div><strong>操作信息：</strong>{(selectedLog as LoginLog).message}</div>
                  <div><strong>登录时间：</strong>{formatLocalTime((selectedLog as LoginLog).loginTime)}</div>
                </>
              )}
            </div>

            {(selectedLog as OperLog).operParam && (
              <>
                <Title level={5}>请求参数</Title>
                <Input.TextArea
                  value={(selectedLog as OperLog).operParam}
                  rows={10}
                  readOnly
                  className="mb-4"
                />
              </>
            )}

            {(selectedLog as OperLog).errorMessage && (
              <>
                <Title level={5}>错误信息</Title>
                <Input.TextArea
                  value={(selectedLog as OperLog).errorMessage}
                  rows={5}
                  readOnly
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
