import { useState, useMemo } from "react";
import type { FC, Key, CSSProperties } from "react";
import { Tree, Card, Spin, Input } from "antd";
import type { DataNode, EventDataNode } from "antd/es/tree";
import { ApartmentOutlined, SearchOutlined } from "@ant-design/icons";
import { useDeptTree } from "@/hooks/useDeptTree";
import type { SimpleDept } from "@/lib/dutyApi";
import { filterExternalOrgDepts, toShortNameDataNode, type DeptLikeNode } from "@/utils/deptUtils";

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
  onSelect?: (
    selectedKeys: Key[],
    info: {
      event: "select";
      selected: boolean;
      node: EventDataNode<DataNode>;
      selectedNodes: DataNode[];
    }
  ) => void;
  selectedKeys?: Key[];
  style?: CSSProperties;
  externalOnly?: boolean;
}

// 模块级稳定空数组：useDeptTree 加载期(data===undefined)的兜底，保证 rawDept/treeData 引用稳定，
// 避免 rc-tree onMotionEnd 死循环 (React #185)。详见组件内 useDeptTree 调用处的注释。
const EMPTY_DEPTS: SimpleDept[] = [];

const DeptTree: FC<DeptTreeProps> = ({
  onSelect: onDeptSelect,
  selectedKeys = [],
  style,
  externalOnly = false,
}) => {
  // 数据层: 消费 canonical hook (Phase 37 收敛)。不再内部 post fetch。
  // ⚠️ 必须用模块级稳定常量 EMPTY_DEPTS 兜底，绝不能用内联 []：加载期 data===undefined 时内联 []
  // 每次渲染都是新引用 → treeData 每次新引用 → rc-tree(@rc-component/tree) 的 onVisibleChange 对空
  // 列表 every()=true → onMotionEnd → setPrevData(新[]) 永不 Object.is bail → React #185 死循环
  // （冷刷新整页空白，工位页等"繁忙"页面先触发）。见 memory rc-tree-virtual-motion-infinite-loop。
  const { data, isLoading: loading } = useDeptTree();
  const rawDept = data ?? EMPTY_DEPTS;

  // 把 SimpleDept[] 窄化为本地 RawDept[] (运行时同对象, 仅 TS 类型层面加 isExternalOrg 字段)。
  const rawDeptTyped = rawDept as unknown as RawDept[];

  // 转换为 antd DataNode 短名格式。
  // externalOnly 模式: 只保留 isExternalOrg===1 的子树 (filterExternalOrgDepts)。
  const treeData = useMemo<DataNode[]>(() => {
    const filtered = externalOnly
      ? filterExternalOrgDepts<RawDept & DeptLikeNode>(
          rawDeptTyped as unknown as Array<RawDept & DeptLikeNode>
        )
      : rawDeptTyped;
    // toShortNameDataNode 输出 { title, key, value, children?, isLeaf? }, 兼容 DataNode。
    return toShortNameDataNode(filtered) as unknown as DataNode[];
  }, [rawDeptTyped, externalOnly]);

  const [searchValue, setSearchValue] = useState("");
  // userExpandedKeys=null 表示用户尚未手动调整 — 用派生值 firstParentKey 兜底首次自动展开。
  // 一旦用户展开/搜索展开，setExpandedKeys 会写入具体数组 → 派生 fallback 不再生效。
  // 这替代了原 useEffect + didInitExpandRef 的"首次设 state"模式：
  // 既满足 Vercel rerender-derived-state-no-effect，也避开 React 19 lint react-hooks/set-state-in-effect。
  const [userExpandedKeys, setExpandedKeys] = useState<Key[] | null>(null);
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // getParentKeys 提前到 useMemo 之前定义（函数声明会提升，避免 TDZ）。
  function getParentKeys(data: DataNode[]): Key[] {
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
  }

  // getParentKeys 是纯派生 (仅依赖 treeData), useMemo 避免重复全树遍历。
  // 同时给 firstParentKey 派生和 onSearch 复用 — 两者此前各跑一次全树遍历。
  const parentKeys = useMemo(() => getParentKeys(treeData), [treeData]);

  // 首次默认展开第一个父节点 — render 阶段派生，不需 effect/ref，
  // treeData 后到达或更新时自动反映在 expandedKeys（前提：用户未手动调整）。
  const firstParentKey = useMemo<Key[]>(
    () => (parentKeys.length > 0 ? [parentKeys[0]] : []),
    [parentKeys]
  );
  const expandedKeys = userExpandedKeys ?? firstParentKey;

  const onSearch = (value: string) => {
    setSearchValue(value);
    if (!value) {
      // 搜空：恢复派生 firstParentKey（与初始首屏行为一致）
      setExpandedKeys(firstParentKey);
      setAutoExpandParent(true);
      return;
    }

    const nextExpanded = getExpandedKeys(treeData, value);
    setExpandedKeys(nextExpanded);
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

  // 过滤结果按 (treeData, searchValue) 缓存 — treeData 大时每次输入变化避免全树重算。
  // 此前用 useCallback([]) 只稳定函数引用，但函数被同步调用于 render body，
  // 每次 render 仍重算（不符合 Vercel rerender-memo 建议）。
  const filteredTreeData = useMemo(() => {
    if (!searchValue) {
      return treeData;
    }

    const filterFn = (data: DataNode[]): DataNode[] => {
      const lowerSearchValue = searchValue.toLowerCase();
      return data
        .filter((node) => {
          const titleMatch = node.title?.toString().toLowerCase().includes(lowerSearchValue);
          const childrenMatch = node.children ? filterFn(node.children).length > 0 : false;
          return titleMatch || childrenMatch;
        })
        .map((node) => ({
          ...node,
          children: node.children ? filterFn(node.children) : undefined,
        }));
    };
    return filterFn(treeData);
  }, [treeData, searchValue]);

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
        <div
          className="dept-tree-container"
          style={{ maxHeight: "calc(100vh - 220px)", overflowX: "hidden" }}
        >
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
