/**
 * Excel导出组件
 * 直接导出当前筛选条件的数据
 */

import { useState, useCallback } from "react";
import type { FC } from "react";
import {
  App,
  Button,
  Space,
  Modal,
  Alert,
  Descriptions,
} from "antd";
import { ExportOutlined } from "@ant-design/icons";
import { getAuthHeaders } from "@/utils/authHelpers";
import { excelApi } from "@/lib/opsApi";

export interface ExcelExportProps {
  entityType: string;
  entityName?: string;
  exportUrl?: string;
  visible?: boolean;
  onClose?: () => void;
  // 接收页面当前筛选参数
  filters?: Record<string, any>;
}

const ExcelExport: FC<ExcelExportProps> = ({
  entityType,
  entityName = entityType,
  exportUrl,
  visible = false,
  onClose,
  filters = {},
}) => {
  const [exporting, setExporting] = useState(false);
  const { message } = App.useApp();

  const handleExport = useCallback(async () => {
    setExporting(true);

    try {
      if (!exportUrl) {
        // 默认路径：调用 opsApi 包装函数，避免硬编码 /api/v1/ops/ 前缀
        await excelApi.export(entityType, filters);
      } else {
        // 自定义路径：保留原内联 fetch（处理 system/dept 等非 ops 模块的导出）
        const authHeaders = await getAuthHeaders();
        const response = await fetch(exportUrl, {
          method: "POST",
          headers: {
            ...authHeaders,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(filters),
        });

        if (!response.ok) {
          throw new Error("导出失败");
        }

        const filename = extractFilename(response.headers.get("content-disposition"), entityName);
        await downloadFile(response, filename);
      }

      message.success("导出成功");
      onClose?.();
    } catch (_error) {
      message.error("导出失败");
    } finally {
      setExporting(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [exportUrl, entityType, entityName, onClose, filters]);

  return (
    <Modal
      title={`导出${entityName}数据`}
      open={visible}
      onCancel={onClose}
      footer={
        <Space>
          <Button onClick={onClose} disabled={exporting}>取消</Button>
          <Button
            type="primary"
            icon={<ExportOutlined />}
            onClick={handleExport}
            loading={exporting}
          >
            立即导出
          </Button>
        </Space>
      }
    >
      {Object.keys(filters).length > 0 ? (
        <Alert
          message="将导出以下筛选条件的数据"
          description={
            <Descriptions size="small" column={1} bordered>
              {Object.entries(filters).map(([key, value]) => {
                if (value === undefined || value === null || value === "") return null;
                return (
                  <Descriptions.Item key={key} label={getFilterLabel(key)}>
                    {formatFilterValue(key, value)}
                  </Descriptions.Item>
                );
              })}
            </Descriptions>
          }
          type="info"
          showIcon
        />
      ) : (
        <Alert
          message="将导出全部数据"
          description="当前页面没有设置任何筛选条件，将导出所有数据。"
          type="warning"
          showIcon
        />
      )}

      <div style={{ marginTop: 16, padding: 12, background: "#f0f7ff", borderRadius: 4 }}>
        <Space orientation="vertical" size={4}>
          <div>• 点击"立即导出"按钮即可下载数据</div>
          <div>• 导出文件为 .xlsx 格式</div>
          <div>• 大量数据导出可能需要较长时间，请耐心等待</div>
        </Space>
      </div>
    </Modal>
  );
};

function getFilterLabel(key: string): string {
  const labels: Record<string, string> = {
    name: "名称",
    code: "编码",
    status: "状态",
    orgId: "所属机构",
    buildingId: "所属楼宇",
    floorId: "所属楼层",
    deptId: "所属部门",
    userId: "所属人员",
    level: "层级",
    workstationType: "工位类型",
    roomCode: "机房编码",
    isp: "运营商",
  };
  return labels[key] || key;
}

function formatFilterValue(key: string, value: any): string {
  if (value === undefined || value === null || value === "") {
    return "-";
  }

  // 状态值格式化
  if (key === "status") {
    const statusMap: Record<number, string> = {
      0: "正常",
      1: "停用",
      2: "故障",
    };
    if (typeof value === "number") {
      return statusMap[value] ?? String(value);
    }
  }

  // 层级格式化
  if (key === "level") {
    const levelMap: Record<number, string> = {
      1: "城市级汇总",
      2: "具体楼宇",
    };
    if (typeof value === "number") {
      return levelMap[value] ?? String(value);
    }
  }

  return String(value);
}

function extractFilename(contentDisposition: string | null, entityName: string): string {
  if (!contentDisposition) {
    return `${entityName}_${Date.now()}.xlsx`;
  }
  const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
  if (match && match[1]) {
    return decodeURIComponent(match[1].replace(/['"]/g, ""));
  }
  return `${entityName}_${Date.now()}.xlsx`;
}

async function downloadFile(response: Response, filename: string): Promise<void> {
  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
}

export default ExcelExport;
