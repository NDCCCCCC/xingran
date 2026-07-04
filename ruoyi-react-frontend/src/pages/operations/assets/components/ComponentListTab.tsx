/**
 * Phase 48 Wave 3 — 从属组件清单 Tab
 *
 * Props: parentAssetId (string, UUID)。父交换机 ops_asset.id。
 * 渲染 Antd Table,展示父设备下所有从属组件(板卡/引擎/电源/风扇/光模块)。
 *
 * useEffect deps 必须是 primitive(parentAssetId 字符串),不可传 object,
 * 否则会触发无限请求循环(per CLAUDE.md useEffect 警告)。
 */

import { useEffect, useState } from "react";
import type { FC } from "react";
import { Table, Tag, App } from "antd";
import type { ColumnsType } from "antd/es/table";
import { componentApi } from "@/lib/opsApi";
import type { Asset } from "@/types/operations";

// D-05 组件类型字典标签(component_type → 中文展示)
const COMPONENT_TYPE_LABELS: Record<string, string> = {
  chassis: "整机",
  card: "业务板",
  engine: "主控/引擎",
  power: "电源",
  fan: "风扇",
  transceiver: "光模块",
};

// 截断 UUID 显示(只保留前 8 位)
const truncateUuid = (s?: string | null): string => {
  if (!s) return "-";
  return s.length > 12 ? `${s.slice(0, 8)}…` : s;
};

interface ComponentListTabProps {
  parentAssetId: string;
}

const ComponentListTab: FC<ComponentListTabProps> = ({ parentAssetId }) => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<Asset[]>([]);

  useEffect(() => {
    if (!parentAssetId) {
      setList([]);
      return;
    }
    let cancelled = false;
    const load = async () => {
      setLoading(true);
      try {
        const res = await componentApi.list(parentAssetId);
        if (cancelled) return;
        const data = (res?.data?.list ?? []) as Asset[];
        setList(data);
      } catch (e) {
        if (!cancelled) {
          message.error(`从属组件加载失败: ${(e as Error).message}`);
          setList([]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [parentAssetId, message]);

  const columns: ColumnsType<Asset> = [
    {
      title: "类型",
      dataIndex: "componentType",
      key: "componentType",
      width: 110,
      render: (t?: string) =>
        t ? (
          <Tag color="blue">{COMPONENT_TYPE_LABELS[t] ?? t}</Tag>
        ) : (
          <span>-</span>
        ),
    },
    {
      title: "槽位/接口",
      dataIndex: "componentSlot",
      key: "componentSlot",
      width: 150,
      render: (s?: string) => s ?? "-",
    },
    {
      title: "序列号",
      dataIndex: "devicesn",
      key: "devicesn",
      width: 180,
      render: (s?: string) => s ?? "-",
    },
    {
      title: "型号",
      dataIndex: "deviceModelName",
      key: "deviceModelName",
      width: 200,
      render: (s?: string) => s ?? "-",
    },
    {
      title: "采集来源设备",
      dataIndex: "sourceDeviceId",
      key: "sourceDeviceId",
      width: 120,
      render: (s?: string) => truncateUuid(s),
    },
  ];

  return (
    <Table<Asset>
      rowKey="id"
      columns={columns}
      dataSource={list}
      loading={loading}
      pagination={false}
      size="small"
      locale={{ emptyText: "无可查询的从属组件" }}
      scroll={{ x: 760 }}
    />
  );
};

export default ComponentListTab;
