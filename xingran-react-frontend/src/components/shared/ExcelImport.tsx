import { useState } from "react";
import type { FC } from "react";
import { App, Button, Upload, Modal, Table, Tag, Space, Alert, Progress, Typography } from "antd";
import {
  UploadOutlined,
  DownloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import type { UploadFile } from "antd";
import type { UploadRequestOption } from "./FileUpload";
import { getAuthHeaders } from "@/utils/authHelpers";
import { excelApi } from "@/lib/opsApi";
import { MAX_GENERAL_FILE_SIZE } from "@/constants/upload";

const { Text } = Typography;

export interface ImportError {
  row: number;
  field: string;
  value: string;
  error: string;
}

export interface ImportResult {
  inserted: number;
  updated: number;
  failed: number;
  errors?: ImportError[];
}

export interface ExcelImportProps {
  entityType: string;
  entityName?: string;
  templateUrl?: string;
  importUrl?: string;
  onImportSuccess?: () => void;
  visible?: boolean;
  onClose?: () => void;
}

const ExcelImport: FC<ExcelImportProps> = ({
  entityType,
  entityName = entityType,
  templateUrl,
  importUrl,
  onImportSuccess,
  visible = false,
  onClose,
}) => {
  // 解析默认导入 URL：
  // - 调用方显式传入 importUrl 时（如 system/dept 模块使用 /api/v1/system/departments/import），
  //   直接使用该 URL；
  // - 未传入时回退到 /api/v1/ops/${entityType}/import，与 opsApi.excelApi.import 的路径保持一致。
  const resolvedImportUrl = importUrl ?? `/api/v1/ops/${entityType}/import`;

  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const { message } = App.useApp();
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);

  const handleDownloadTemplate = async () => {
    try {
      if (!templateUrl) {
        // 默认路径：调用 opsApi 包装函数，避免硬编码 /api/v1/ops/ 前缀
        await excelApi.downloadTemplate(entityType);
      } else {
        // 自定义路径：保留原内联 fetch（处理 system/dept 等非 ops 模块的下载）
        const headers = await getAuthHeaders();
        const response = await fetch(templateUrl, { headers });

        if (!response.ok) {
          throw new Error("下载模板失败");
        }

        const filename = `${entityName}_模板.xlsx`;
        await downloadFile(response, filename);
      }

      message.success("模板下载成功");
    } catch {
      message.error("下载模板失败");
    }
  };

  const beforeUpload = (file: File) => {
    const validTypes = ["application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"];
    const isExcelType = validTypes.includes(file.type) || file.name.endsWith(".xlsx");
    if (!isExcelType) {
      message.error("只支持 .xlsx 格式。如需导入 .xls 文件，请先用 Excel 或 WPS 另存为 .xlsx 格式");
      return Upload.LIST_IGNORE;
    }

    const maxSize = MAX_GENERAL_FILE_SIZE;
    if (file.size > maxSize) {
      message.error("文件大小不能超过 10MB");
      return Upload.LIST_IGNORE;
    }

    return true;
  };

  const customRequest = async (options: UploadRequestOption) => {
    const { file, onProgress, onSuccess, onError } = options;

    // 获取原始文件对象 - 处理 Ant Design 6.x 的 UploadRequestFile 类型
    let rawFile: File | Blob | null = null;

    if (file instanceof File || file instanceof Blob) {
      rawFile = file;
    } else if (typeof file === "string") {
      // file 是字符串（URL）的情况，不应该发生在这里
      onError?.(new Error("不支持URL类型的文件"));
      return;
    } else if (typeof file === "object" && file !== null && "originFileObj" in file) {
      // file 是 UploadFile 对象
      rawFile = (file as UploadFile).originFileObj || null;
    }

    if (!rawFile) {
      onError?.(new Error("无法获取文件"));
      return;
    }

    const formData = new FormData();
    formData.append("file", rawFile);

    setImporting(true);
    setUploadProgress(0);
    setImportResult(null);

    try {
      const xhr = new XMLHttpRequest();

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable && onProgress) {
          const percent = Math.round((e.loaded / e.total) * 100);
          setUploadProgress(percent);
          onProgress({ percent });
        }
      };

      xhr.onload = () => {
        if (xhr.status === 200) {
          const response = JSON.parse(xhr.responseText);
          setImporting(false);
          setUploadProgress(100);
          setImportResult(response.data);
          onSuccess?.(response);

          if (response.data.failed === 0) {
            message.success(
              `导入成功！新增 ${response.data.inserted} 条，更新 ${response.data.updated} 条`
            );
            onImportSuccess?.();
          } else {
            message.warning(
              `导入完成！新增 ${response.data.inserted} 条，更新 ${response.data.updated} 条，失败 ${response.data.failed} 条`
            );
          }
        } else {
          throw new Error(xhr.responseText || "导入失败");
        }
      };

      xhr.onerror = () => {
        setImporting(false);
        const error = new Error("网络错误");
        onError?.(error);
        message.error("导入失败");
      };

      xhr.open("POST", resolvedImportUrl, true);
      const headers = await getAuthHeaders();
      if (headers["Authorization"]) {
        xhr.setRequestHeader("Authorization", headers["Authorization"]);
      }
      xhr.send(formData);
    } catch {
      setImporting(false);
      onError?.(new Error("导入失败"));
      message.error("导入失败");
    }
  };

  const handleReset = () => {
    setFileList([]);
    setImportResult(null);
    setUploadProgress(0);
  };

  const errorColumns = [
    {
      title: "行号",
      dataIndex: "row",
      key: "row",
      width: 80,
    },
    {
      title: "字段",
      dataIndex: "field",
      key: "field",
      width: 120,
    },
    {
      title: "值",
      dataIndex: "value",
      key: "value",
      width: 150,
      ellipsis: true,
    },
    {
      title: "错误信息",
      dataIndex: "error",
      key: "error",
    },
  ];

  return (
    <Modal
      title={`导入${entityName}数据`}
      open={visible}
      onCancel={onClose}
      footer={null}
      width={800}
      destroyOnHidden
    >
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        <Alert
          title="导入说明"
          description={
            <div>
              <div>1. 先下载Excel模板文件</div>
              <div>2. 按照模板格式填写数据</div>
              <div>3. 上传填写好的Excel文件进行导入</div>
              <div>4. 导入完成后会显示结果，如有错误会详细列出</div>
            </div>
          }
          type="info"
          showIcon
        />

        <div>
          <Text strong>步骤1：下载模板</Text>
          <div style={{ marginTop: 8 }}>
            <Button icon={<DownloadOutlined />} onClick={handleDownloadTemplate}>
              下载{entityName}Excel模板
            </Button>
          </div>
        </div>

        <div>
          <Text strong>步骤2：上传文件</Text>
          <div style={{ marginTop: 8 }}>
            <Upload
              fileList={fileList}
              onChange={({ fileList: newFileList }) => setFileList(newFileList)}
              beforeUpload={beforeUpload}
              customRequest={customRequest as any}
              accept=".xlsx"
              maxCount={1}
              disabled={importing}
              onRemove={handleReset}
            >
              <Button icon={<UploadOutlined />} disabled={fileList.length >= 1 || importing}>
                选择Excel文件
              </Button>
            </Upload>
            <div style={{ marginTop: 8 }}>
              <Text type="secondary">支持 .xlsx 格式，文件大小不超过 10MB</Text>
            </div>
          </div>
        </div>

        {importing && (
          <div>
            <Text strong>导入中...</Text>
            <Progress percent={uploadProgress} />
          </div>
        )}

        {importResult && (
          <div>
            <Space orientation="vertical" style={{ width: "100%" }}>
              <div
                style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}
              >
                <Text strong>导入结果</Text>
                <Button size="small" icon={<DeleteOutlined />} onClick={handleReset}>
                  重置
                </Button>
              </div>

              <Space size="large">
                <div>
                  <Tag color="success" icon={<CheckCircleOutlined />}>
                    新增: {importResult.inserted}
                  </Tag>
                </div>
                <div>
                  <Tag color="processing" icon={<CheckCircleOutlined />}>
                    更新: {importResult.updated}
                  </Tag>
                </div>
                {importResult.failed > 0 && (
                  <div>
                    <Tag color="error" icon={<CloseCircleOutlined />}>
                      失败: {importResult.failed}
                    </Tag>
                  </div>
                )}
              </Space>

              {importResult.errors && importResult.errors.length > 0 && (
                <div>
                  <Text type="danger" strong>
                    以下数据导入失败，请检查后重新导入：
                  </Text>
                  <Table
                    columns={errorColumns}
                    dataSource={importResult.errors}
                    pagination={{ pageSize: 10 }}
                    size="small"
                    rowKey="row"
                    style={{ marginTop: 8 }}
                  />
                </div>
              )}
            </Space>
          </div>
        )}
      </Space>
    </Modal>
  );
};

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

export default ExcelImport;
