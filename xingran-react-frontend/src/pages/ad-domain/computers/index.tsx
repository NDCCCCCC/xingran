import { useState, useEffect, useMemo } from "react";
import {
  App,
  Card,
  Table,
  Select,
  TreeSelect,
  Input,
  Space,
  Button,
  Tag,
  Form,
  Descriptions,
  Modal,
  Tooltip,
} from "antd";
import {
  ReloadOutlined,
  DesktopOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  SearchOutlined,
  FolderOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import {
  getADComputerList,
  getADComputerDetail,
  getADOUTree,
  type ADComputerDetail,
  type ADOUNode,
} from "@/lib/adDomainApi";
import { useADConfigs } from "@/hooks/useADConfigs";
import { createSorter, createSorterMeta } from "@/utils/tableHelpers";
import { useServerSort } from "@/hooks/useServerSort";
import type { FC } from "react";
import { usePagination } from "@/hooks/usePagination";

const ADComputerPage: FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();

  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  // 获取当前选择的配置对象
  const currentConfig = useMemo(() => {
    return configs.find((c) => c.id === selectedConfig);
  }, [configs, selectedConfig]);

  const [computers, setComputers] = useState<ADComputerDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const { paginationProps, setCurrent, setTotal } = usePagination();

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedComputer, setSelectedComputer] = useState<ADComputerDetail | null>(null);
  const [ouSelectVisible, setOUSelectVisible] = useState(false);

  // 常用OU快捷列表（可配置）
  const commonOUs = useMemo(() => {
    if (!ouTree.length) return [];
    const findOUByName = (name: string, nodes: ADOUNode[] = ouTree): ADOUNode | null => {
      for (const node of nodes) {
        if (node.name === name) return node;
        if (node.children) {
          const found = findOUByName(name, node.children);
          if (found) return found;
        }
      }
      return null;
    };

    const commonNames = ["湖北分公司", "失效终端", "本部部门分组"];
    return commonNames.map((name) => findOUByName(name)).filter(Boolean) as ADOUNode[];
  }, [ouTree]);

  const fetchOUTree = async () => {
    if (!selectedConfig) return;
    try {
      const res = await getADOUTree(selectedConfig);
      if (res.code === 0 && res.data) {
        // 过滤出 ou=computer 下的子OU
        const computerOU = res.data.find((ou) => ou.name === "Computer");
        setOUTree(computerOU?.children ?? []);
      }
    } catch (error) {
      console.error("获取OU树失败", error);
    }
  };

  useEffect(() => {
    if (selectedConfig) {
      fetchOUTree();
      fetchComputers();
    }
  }, [selectedConfig, paginationProps.current, paginationProps.pageSize]);

  // 服务端排序:field 与 columns.dataIndex 对齐(useServerSort 按 sorter.field 匹配)
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<ADComputerDetail>("computerName"),
      createSorterMeta<ADComputerDetail>("lastLogonUser"),
      createSorterMeta<ADComputerDetail>("ipAddress"),
      createSorterMeta<ADComputerDetail>("operatingSystem"),
      createSorterMeta<ADComputerDetail>("status"),
    ],
    []
  );
  const {
    orderByColumn,
    isAsc,
    handleTableChange: handleCompSortChange,
    sortOrder: compSortOrder,
  } = useServerSort<ADComputerDetail>({
    sorterMetas,
  });

  const fetchComputers = async () => {
    if (!selectedConfig) return;

    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const res = await getADComputerList({
        configId: selectedConfig,
        ouDn: values.ouDn,
        computerName: values.computerName,
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        ...(orderByColumn ? { orderByColumn, isAsc } : {}),
      });
      if (res.code === 0) {
        setComputers(res.data?.list ?? []);
        setTotal(res.data?.total ?? 0);
      }
    } catch {
      message.error("获取电脑设备列表失败");
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setCurrent(1);
    fetchComputers();
  };

  const handleViewDetail = async (computer: ADComputerDetail) => {
    if (!selectedConfig) return;
    try {
      const res = await getADComputerDetail(selectedConfig, computer.distinguishedName);
      if (res.code === 0) {
        setSelectedComputer(res.data ?? null);
        setDetailModalVisible(true);
      }
    } catch (error) {
      message.error("获取电脑设备详情失败");
    }
  };

  // 构建TreeSelect所需的数据结构
  const buildTreeSelectData = (
    nodes: ADOUNode[]
  ): Array<{
    value: string;
    title: string;
    children?: Array<{ value: string; title: string }>;
  }> => {
    return nodes.map((node) => ({
      value: node.dn,
      title: node.name,
      children:
        node.children && node.children.length > 0 ? buildTreeSelectData(node.children) : undefined,
    }));
  };

  const treeSelectData = useMemo(() => buildTreeSelectData(ouTree), [ouTree]);

  const handleQuickOUSelect = (ouDn: string) => {
    form.setFieldValue("ouDn", ouDn);
    handleSearch();
  };

  const columns: ColumnsType<ADComputerDetail> = [
    {
      title: "计算机名",
      dataIndex: "computerName",
      key: "computerName",
      fixed: "left",
      width: 180,
      sorter: createSorter<ADComputerDetail>("computerName", "string"),
    },
    {
      title: "最后登录用户",
      dataIndex: "lastLogonUser",
      key: "lastLogonUser",
      width: 150,
      sorter: createSorter<ADComputerDetail>("lastLogonUser", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "IP地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 140,
      sorter: createSorter<ADComputerDetail>("ipAddress", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "MAC地址",
      dataIndex: "macAddress",
      key: "macAddress",
      width: 160,
      sorter: createSorter<ADComputerDetail>("macAddress", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "操作系统",
      dataIndex: "operatingSystem",
      key: "operatingSystem",
      width: 200,
      ellipsis: true,
      sorter: createSorter<ADComputerDetail>("operatingSystem", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "CPU",
      dataIndex: "cpuModel",
      key: "cpuModel",
      width: 180,
      ellipsis: true,
      sorter: createSorter<ADComputerDetail>("cpuModel", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "架构",
      dataIndex: "architecture",
      key: "architecture",
      width: 100,
      sorter: createSorter<ADComputerDetail>("architecture", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "内存",
      dataIndex: "memoryCapacity",
      key: "memoryCapacity",
      width: 100,
      sorter: createSorter<ADComputerDetail>("memoryCapacity", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "硬盘",
      dataIndex: "hardDiskCapacity",
      key: "hardDiskCapacity",
      width: 100,
      sorter: createSorter<ADComputerDetail>("hardDiskCapacity", "string"),
      render: (text: string) => text || "-",
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      sorter: createSorter<ADComputerDetail>("status", "number"),
      render: (status: number) => (
        <Tag
          icon={status === 0 ? <CheckCircleOutlined /> : <WarningOutlined />}
          color={status === 0 ? "success" : "default"}
        >
          {status === 0 ? "在线" : "离线"}
        </Tag>
      ),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 100,
      render: (_: unknown, record: ADComputerDetail) => (
        <Button type="link" size="small" onClick={() => handleViewDetail(record)}>
          详情
        </Button>
      ),
    },
  ];

  return (
    <Card>
      <Space orientation="vertical" style={{ width: "100%" }} size="large">
        {/* 搜索栏 */}
        <Form form={form} layout="inline">
          <Form.Item label="AD配置">
            <Select
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={setSelectedConfig}
              options={configs.map((c) => ({ label: c.configName, value: c.id }))}
              onSearch={() => {}}
            />
          </Form.Item>
          <Form.Item name="ouDn" label="所属OU">
            <Space.Compact style={{ width: "100%" }}>
              <TreeSelect
                placeholder="请选择OU"
                allowClear
                showSearch
                treeDefaultExpandAll={false}
                treeIcon={<FolderOutlined />}
                treeNodeFilterProp="title"
                style={{ width: 300 }}
                styles={{ popup: { root: { maxHeight: 400, overflow: "auto" } } }}
                treeData={treeSelectData}
                onChange={(value) => {
                  form.setFieldValue("ouDn", value);
                }}
              />
              {commonOUs.length > 0 && (
                <Tooltip title="快捷选择常用OU">
                  <Button icon={<SearchOutlined />} onClick={() => setOUSelectVisible(true)}>
                    快捷
                  </Button>
                </Tooltip>
              )}
            </Space.Compact>
          </Form.Item>
          <Form.Item name="computerName">
            <Input placeholder="计算机名称" allowClear style={{ width: 180 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={fetchComputers}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>

        {/* 电脑设备列表 */}
        <Table
          columns={columns.map((col) =>
            "dataIndex" in col &&
            col.dataIndex &&
            sorterMetas.some((m) => m?.field === String(col.dataIndex))
              ? { ...col, sortOrder: orderByColumn === col.dataIndex ? compSortOrder : null }
              : col
          )}
          dataSource={computers}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1800 }}
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            handleCompSortChange(pagination, _filters, sorter);
            setCurrent(pagination.current ?? 1);
            fetchComputers();
          }}
        />
      </Space>

      {/* 详情弹窗 */}
      <Modal
        title={
          <Space>
            <DesktopOutlined />
            <span>电脑设备详情 - {selectedComputer?.computerName}</span>
          </Space>
        }
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>,
        ]}
        width={800}
      >
        {selectedComputer && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="计算机名" span={2}>
              {selectedComputer.computerName}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录用户" span={1}>
              {selectedComputer.lastLogonUser || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="状态" span={1}>
              <Tag
                icon={selectedComputer.status === 0 ? <CheckCircleOutlined /> : <WarningOutlined />}
                color={selectedComputer.status === 0 ? "success" : "default"}
              >
                {selectedComputer.status === 0 ? "在线" : "离线"}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="IP地址" span={1}>
              {selectedComputer.ipAddress || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="MAC地址" span={1}>
              {selectedComputer.macAddress || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="操作系统" span={2}>
              {selectedComputer.operatingSystem || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="系统版本" span={2}>
              {selectedComputer.osVersion || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="CPU型号" span={2}>
              {selectedComputer.cpuModel || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="架构" span={1}>
              {selectedComputer.architecture || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="内存容量" span={1}>
              {selectedComputer.memoryCapacity || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="硬盘容量" span={1}>
              {selectedComputer.hardDiskCapacity || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="序列号" span={1}>
              {selectedComputer.serialNumber || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="管理者" span={1}>
              {selectedComputer.managedBy || "-"}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录时间" span={1}>
              {selectedComputer.lastLogon
                ? new Date(selectedComputer.lastLogon).toLocaleString("zh-CN")
                : "-"}
            </Descriptions.Item>
            <Descriptions.Item label="最后上线时间" span={1}>
              {selectedComputer.lastOnlineTime
                ? new Date(selectedComputer.lastOnlineTime).toLocaleString("zh-CN")
                : "-"}
            </Descriptions.Item>
            <Descriptions.Item label="登录次数" span={1}>
              {selectedComputer.logonCount}
            </Descriptions.Item>
            <Descriptions.Item label="DN" span={2}>
              <span style={{ fontSize: "12px", fontFamily: "monospace" }}>
                {selectedComputer.distinguishedName}
              </span>
            </Descriptions.Item>
            {selectedComputer.originalDescription && (
              <Descriptions.Item label="原始描述" span={2}>
                <span style={{ fontSize: "12px", fontFamily: "monospace", whiteSpace: "pre-wrap" }}>
                  {selectedComputer.originalDescription}
                </span>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Modal>

      {/* OU快捷选择弹窗 */}
      <Modal
        title="快捷选择常用OU"
        open={ouSelectVisible}
        onCancel={() => setOUSelectVisible(false)}
        footer={[
          <Button key="close" onClick={() => setOUSelectVisible(false)}>
            关闭
          </Button>,
        ]}
        width={600}
      >
        <Space orientation="vertical" style={{ width: "100%" }} size="middle">
          <div style={{ color: "#666", fontSize: "14px" }}>
            点击下方OU标签快速筛选该OU下的电脑设备
          </div>
          <div style={{ maxHeight: 400, overflowY: "auto" }}>
            <Space wrap>
              {commonOUs.map((ou) => (
                <Tag
                  key={ou.dn}
                  style={{
                    cursor: "pointer",
                    padding: "8px 16px",
                    fontSize: "14px",
                    marginBottom: "8px",
                  }}
                  icon={<FolderOutlined />}
                  onClick={() => {
                    handleQuickOUSelect(ou.dn);
                    setOUSelectVisible(false);
                  }}
                >
                  {ou.name}
                </Tag>
              ))}
            </Space>
          </div>
          {commonOUs.length === 0 && (
            <div style={{ textAlign: "center", color: "#999", padding: "40px" }}>
              暂无常用OU配置
            </div>
          )}
        </Space>
      </Modal>
    </Card>
  );
};

export default ADComputerPage;
