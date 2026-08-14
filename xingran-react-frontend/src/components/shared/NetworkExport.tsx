/**
 * NetworkExport 网络设备导出组件
 * 支持三种导出模式：筛选导出、当前页导出、全部导出
 */

import { useState, useCallback, type FC } from "react";
import { Button, Dropdown, App, Modal, Radio, Space } from "antd";
import type { MenuProps } from "antd";
import { ExportOutlined, DownloadOutlined } from "@ant-design/icons";
import { getAccessToken } from "@/utils/authHelpers";

export interface NetworkExportProps {
  entityType: string;         // 实体类型（用于拼接导出路径 /api/v1/network/${entityType}/export）
  entityName: string;         // 实体名称（用于文件名和提示）
  filters?: Record<string, any>; // 当前筛选条件
  current?: number;           // 当前页码
  pageSize?: number;          // 每页条数
}

const NetworkExport: FC<NetworkExportProps> = ({
  entityType,
  entityName,
  filters = {},
  current = 1,
  pageSize = 10,
}) => {
  const { message } = App.useApp();
  const exportUrl = `/api/v1/network/${entityType}/export`;
  const [exporting, setExporting] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [exportMode, setExportMode] = useState<string>("filtered");

  const doExport = useCallback(async (mode: string) => {
    setExporting(true);
    try {
      const token = await getAccessToken();
      const body: Record<string, any> = { exportMode: mode };

      if (mode === "filtered") {
        // 筛选导出：带筛选条件，不分页
        body.filters = filters;
      } else if (mode === "currentPage") {
        // 当前页：带筛选条件 + 分页
        body.filters = filters;
        body.current = current;
        body.pageSize = pageSize;
      }
      // mode === 'all': 不带任何参数

      const response = await fetch(exportUrl, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) throw new Error("导出失败");

      const disposition = response.headers.get("content-disposition");
      let filename = `${entityName}_${Date.now()}.xlsx`;
      if (disposition) {
        const match = disposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
        if (match?.[1]) filename = decodeURIComponent(match[1].replace(/['"]/g, ""));
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      message.success(`${entityName}导出成功`);
      setModalVisible(false);
    } catch {
      message.error(`${entityName}导出失败`);
    } finally {
      setExporting(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [exportUrl, entityName, filters, current, pageSize]);

  const menuItems: MenuProps["items"] = [
    {
      key: "filtered",
      label: "筛选导出",
      onClick: () => { setExportMode("filtered"); setModalVisible(true); },
    },
    {
      key: "currentPage",
      label: "导出当前页",
      onClick: () => { setExportMode("currentPage"); setModalVisible(true); },
    },
    {
      key: "all",
      label: "导出全部",
      onClick: () => { setExportMode("all"); setModalVisible(true); },
    },
  ];

  return (
    <>
      <Dropdown menu={{ items: menuItems }} placement="bottomRight">
        <Button icon={<ExportOutlined />}>导出</Button>
      </Dropdown>
      <Modal
        title={`导出${entityName}数据`}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={
          <Space>
            <Button onClick={() => setModalVisible(false)} disabled={exporting}>取消</Button>
            <Button type="primary" icon={<DownloadOutlined />} loading={exporting} onClick={() => doExport(exportMode)}>
              确认导出
            </Button>
          </Space>
        }
      >
        <Radio.Group value={exportMode} onChange={e => setExportMode(e.target.value)} style={{ width: "100%" }}>
          <Space direction="vertical" style={{ width: "100%" }}>
            <Radio value="filtered">
              筛选导出 — 按当前筛选条件导出所有匹配数据
              {Object.keys(filters).length > 0 && <span style={{ color: "var(--theme-info, #1890ff)", marginLeft: 8 }}>(已设置筛选条件)</span>}
            </Radio>
            <Radio value="currentPage">
              导出当前页 — 仅导出当前页面显示的数据（第{current}页，每页{pageSize}条）
            </Radio>
            <Radio value="all">
              导出全部 — 导出所有数据，无视筛选条件和分页
            </Radio>
          </Space>
        </Radio.Group>
      </Modal>
    </>
  );
};

export default NetworkExport;
