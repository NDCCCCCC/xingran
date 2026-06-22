import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import type { FC, Key, CSSProperties } from "react";
import { Tree, Card, Spin, Input } from "antd";
import type { DataNode, EventDataNode } from "antd/es/tree";
import { ApartmentOutlined, SearchOutlined } from "@ant-design/icons";
import { useDeptTree } from "@/hooks/useDeptTree";
import type { SimpleDept } from "@/lib/dutyApi";
import {
  filterExternalOrgDepts,
  toShortNameDataNode,
  type DeptLikeNode,
} from "@/utils/deptUtils";

/**
 * 后端部门树接口实际返回的节点形状。
 *
 * `SimpleDept` (canonical) 只声明 id/deptName/parentId/children, 但运行时数据
 * 还含 isExternalOrg 等字段(见 `@/types/system.ts` 的 Department 接口)。
 * 本组件 `externalOnly` 模式依赖 isExternalOrg, 故本地扩展一个带该字段的类型,
 * 与 useDeptTree 的 SimpleDept 在运行时完全兼容(仅静态类型层面窄化)。
 */
interface RawDept extends SimpleDept {
  isExternalOrg?: number;
  children?: RawDept[];
}

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
  // 数据层: 消费 canonical hook (Phase 37 收敛)。不再内部 post fetch。
  const { data: rawDept = [], isLoading: loading } = useDeptTree();

  // 把 SimpleDept[] 窄化为本地 RawDept[] (运行时同对象, 仅 TS 类型层面加 isExternalOrg 字段)。
  const rawDeptTyped = rawDept as unknown as RawDept[];

  // 转换为 antd DataNode 短名格式。
  // externalOnly 模式: 只保留 isExternalOrg===1 的子树 (filterExternalOrgDepts)。
  const treeData = useMemo<DataNode[]>(() => {
    const filtered = externalOnly
      ? filterExternalOrgDepts<RawDept & DeptLikeNode>(
          rawDeptTyped as unknown as Array<RawDept & DeptLikeNode>,
        )
      : rawDeptTyped;
    // toShortNameDataNode 输出 { title, key, value, children?, isLeaf? }, 兼容 DataNode。
    return toShortNameDataNode(filtered) as unknown as DataNode[];
  }, [rawDeptTyped, externalOnly]);

  const [searchValue, setSearchValue] = useState("");
  const [expandedKeys, setExpandedKeys] = useState<Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // 首次自动展开守卫: 原 loadDeptTree 在数据到达后把 expandedKeys 初始化为
  // parentKeys[0] (第一个父节点)。改为 useDeptTree 后, 用 ref 守卫确保**只在首次
  // 数据到达时跑一次**, 避免每次 rawDept 引用变化都重置用户手动展开状态。
  const didInitExpandRef = useRef(false);
  useEffect(() => {
    if (didInitExpandRef.current) return;
    if (!treeData || treeData.length === 0) return;
    const parentKeys = getParentKeys(treeData);
    const firstParentKey = parentKeys.length > 0 ? [parentKeys[0]] : [];
    if (firstParentKey.length > 0) {
      setExpandedKeys(firstParentKey);
      setAutoExpandParent(true);
      didInitExpandRef.current = true;
    }
  }, [treeData]);

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

  const filterTreeData = useCallback(
    (data: DataNode[], searchValue: string): DataNode[] => {
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
    },
    [],
  );

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
