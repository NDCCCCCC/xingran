import { useState, useEffect, useCallback } from "react";
import type { FC, Key, CSSProperties } from "react";
import { Tree, Card, Spin, Input } from "antd";
import type { DataNode, EventDataNode } from "antd/es/tree";
import { ApartmentOutlined, SearchOutlined } from "@ant-design/icons";
import { post } from "@/lib/api";
import type { Department } from "@/types";
import { filterExternalOrgDepts } from "@/utils/deptUtils";

interface DeptTreeProps {
  onSelect?: (selectedKeys: Key[], info: {
    event: "select";
    selected: boolean;
    node: EventDataNode<DataNode>;
    selectedNodes: DataNode[];
  }) => void;
  selectedKeys?: Key[];
  style?: CSSProperties;
  externalOnly?: boolean;
}

const DeptTree: FC<DeptTreeProps> = ({
  onSelect: onDeptSelect,
  selectedKeys = [],
  style,
  externalOnly = false,
}) => {
  const [treeData, setTreeData] = useState<DataNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchValue, setSearchValue] = useState("");
  const [expandedKeys, setExpandedKeys] = useState<Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  const getParentKeys = (data: DataNode[]): Key[] => {
    const parentKeys: Key[] = [];
    const traverse = (nodes: DataNode[]) => {
      for (const node of nodes) {
        if (node.children && node.children.length > 0) {
          parentKeys.push(node.key);
          traverse(node.children);
        }
      }
    };
    traverse(data);
    return parentKeys;
  };

  const loadDeptTree = useCallback(async () => {
    setLoading(true);
    try {
      const result = await post<Department[]>("/system/departments/tree");
      let deptData = result.data || [];

      if (externalOnly) {
        // 收敛到 src/utils/deptUtils 的共享实现(原 DepartmentTreeSelect 与
        // useWorkstationData 也有重复版本,现统一到 deptUtils.filterExternalOrgDepts)。
        deptData = filterExternalOrgDepts(deptData as Department[]);
      }

      const treeData = transformToTreeData(deptData);
      setTreeData(treeData);

      const parentKeys = getParentKeys(treeData);
      const firstParentKey = parentKeys.length > 0 ? [parentKeys[0]] : [];
      setExpandedKeys(firstParentKey);
    } catch (error) {
      console.error("加载部门树失败:", error);
      setTreeData([]);
    } finally {
      setLoading(false);
    }
  }, [externalOnly]);

  const transformToTreeData = (data: Department[]): DataNode[] => {
    if (!Array.isArray(data)) {
      return [];
    }

    return data.map(item => ({
      title: item.deptName,
      key: item.id,
      children: item.children && item.children.length > 0 ? transformToTreeData(item.children) : undefined,
      isLeaf: !item.children || item.children.length === 0,
    }));
  };

  const onSearch = (value: string) => {
    setSearchValue(value);
    if (!value) {
      const parentKeys = getParentKeys(treeData);
      const firstParentKey = parentKeys.length > 0 ? [parentKeys[0]] : [];
      setExpandedKeys(firstParentKey);
      setAutoExpandParent(true);
      return;
    }

    const expandedKeys = getExpandedKeys(treeData, value);
    setExpandedKeys(expandedKeys);
    setAutoExpandParent(true);
  };

  const getExpandedKeys = (data: DataNode[], searchValue: string): Key[] => {
    const expandedKeys: Key[] = [];
    const lowerSearchValue = searchValue.toLowerCase();
    const traverse = (nodes: DataNode[]) => {
      for (const node of nodes) {
        if (node.title?.toString().toLowerCase().includes(lowerSearchValue)) {
          expandedKeys.push(node.key);
        }
        if (node.children) {
          traverse(node.children);
        }
      }
    };
    traverse(data);
    return expandedKeys;
  };

  const onExpand = (expandedKeys: Key[]) => {
    setExpandedKeys(expandedKeys);
    setAutoExpandParent(false);
  };

  const handleTreeSelect = (
    selectedKeys: Key[],
    info: {
      event: "select";
      selected: boolean;
      node: EventDataNode<DataNode>;
      selectedNodes: DataNode[];
    }
  ) => {
    if (onDeptSelect) {
      onDeptSelect(selectedKeys, info);
    }
  };

  const filterTreeData = (data: DataNode[], searchValue: string): DataNode[] => {
    if (!searchValue) {
      return data;
    }

    const lowerSearchValue = searchValue.toLowerCase();
    return data.filter(node => {
      const titleMatch = node.title?.toString().toLowerCase().includes(lowerSearchValue);
      const childrenMatch = node.children ? filterTreeData(node.children, searchValue).length > 0 : false;
      return titleMatch || childrenMatch;
    }).map(node => ({
      ...node,
      children: node.children ? filterTreeData(node.children, searchValue) : undefined,
    }));
  };

  useEffect(() => {
    loadDeptTree();
  }, [loadDeptTree]);

  const filteredTreeData = filterTreeData(treeData, searchValue);

  return (
    <Card
      title={
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <ApartmentOutlined />
          <span>部门列表</span>
        </div>
      }
      size="small"
      style={style}
      styles={{ body: { padding: "16px" } }}
    >
      <Input
        placeholder="搜索部门"
        value={searchValue}
        onChange={(e) => {
          setSearchValue(e.target.value);
          onSearch(e.target.value);
        }}
        allowClear
        prefix={<SearchOutlined />}
        className="dept-tree-search user-form-input"
        style={{ marginBottom: 8 }}
      />

      <Spin spinning={loading}>
        <div className="dept-tree-container" style={{ maxHeight: "calc(100vh - 220px)", overflowX: "hidden" }}>
          <Tree
            showLine
            treeData={filteredTreeData}
            onSelect={handleTreeSelect}
            selectedKeys={selectedKeys}
            expandedKeys={expandedKeys}
            onExpand={onExpand}
            autoExpandParent={autoExpandParent}
          />
        </div>
      </Spin>
    </Card>
  );
};

export default DeptTree;