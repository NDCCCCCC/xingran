import { useState, useEffect, useCallback } from 'react';
import type { FC, Key, CSSProperties } from 'react';
import { Tree, Card, Spin, Input } from 'antd';
import type { DataNode, EventDataNode } from 'antd/es/tree';
import { ApartmentOutlined, SearchOutlined } from '@ant-design/icons';
import { post } from '@/lib/api';
import type { BaseResponse, Department } from '@/types';

interface DeptTreeProps {
  onSelect?: (selectedKeys: Key[], info: {
    event: 'select';
    selected: boolean;
    node: EventDataNode<DataNode>;
    selectedNodes: DataNode[];
  }) => void;
  selectedKeys?: Key[];
  style?: CSSProperties;
  externalOnly?: boolean;  // 仅显示外部机构部门
}

const DeptTree: FC<DeptTreeProps> = ({
  onSelect: onDeptSelect,
  selectedKeys = [],
  style,
  externalOnly = false,
}) => {
  const [treeData, setTreeData] = useState<DataNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // 获取所有父节点（有子节点的节点）的 key
  const getParentKeys = (data: DataNode[]): Key[] => {
    const parentKeys: Key[] = [];
    const traverse = (nodes: DataNode[]) => {
      nodes.forEach(node => {
        if (node.children && node.children.length > 0) {
          parentKeys.push(node.key);
          traverse(node.children);
        }
      });
    };
    traverse(data);
    return parentKeys;
  };

  // 过滤出外部机构部门（isExternalOrg === 1）
  const filterExternalOrgDepts = (data: Department[]): Department[] => {
    if (!data || !Array.isArray(data)) {
      return [];
    }

    const filterNodes = (nodes: Department[]): Department[] => {
      return nodes.reduce<Department[]>((acc, node) => {
        // 先递归过滤子节点
        const filteredChildren = node.children && node.children.length > 0
          ? filterNodes(node.children)
          : [];

        // 判断是否保留该节点：
        // 1. 当前节点本身是外部机构（isExternalOrg === 1），或者
        // 2. 该节点的子树中有外部机构部门（需要保留作为路径）
        const isExternal = node.isExternalOrg === 1;
        const hasExternalDescendants = filteredChildren.length > 0;

        if (isExternal || hasExternalDescendants) {
          acc.push({
            ...node,
            // 统一使用过滤后的子节点，不保留原始子节点
            children: filteredChildren.length > 0 ? filteredChildren : undefined,
          });
        }

        return acc;
      }, []);
    };

    return filterNodes(data);
  };

  // 加载部门树数据
  const loadDeptTree = useCallback(async () => {
    setLoading(true);
    try {
      const result = await post<BaseResponse<Department[]>>('/system/departments/tree');
      let deptData = result.data || [];

      // 如果只需要外部机构部门，进行过滤
      if (externalOnly) {
        deptData = filterExternalOrgDepts(deptData);
      }

      const treeData = transformToTreeData(deptData);
      setTreeData(treeData);

      // 默认只展开第一个父节点（手风琴模式）
      const parentKeys = getParentKeys(treeData);
      const firstParentKey = parentKeys.length > 0 ? [parentKeys[0]] : [];
      setExpandedKeys(firstParentKey);
    } catch (error) {
      console.error('加载部门树失败:', error);
      setTreeData([]);
    } finally {
      setLoading(false);
    }
  }, [externalOnly]);

  // 转换部门数据为树形结构
  const transformToTreeData = (data: Department[]): DataNode[] => {
    if (!data || !Array.isArray(data)) {
      return [];
    }

    return data.map(item => ({
      title: item.deptName,
      key: item.id,
      children: item.children && item.children.length > 0 ? transformToTreeData(item.children) : undefined,
      isLeaf: !item.children || item.children.length === 0,
    }));
  };

  // 搜索处理
  const onSearch = (value: string) => {
    setSearchValue(value);
    if (!value) {
      // 如果搜索为空，只展开第一个父节点（手风琴模式）
      const parentKeys = getParentKeys(treeData);
      const firstParentKey = parentKeys.length > 0 ? [parentKeys[0]] : [];
      setExpandedKeys(firstParentKey);
      setAutoExpandParent(true);
      return;
    }

    // 搜索匹配的节点
    const expandedKeys = getExpandedKeys(treeData, value);
    setExpandedKeys(expandedKeys);
    setAutoExpandParent(true);
  };

  // 获取需要展开的节点key
  const getExpandedKeys = (data: DataNode[], searchValue: string): Key[] => {
    const expandedKeys: Key[] = [];
    const traverse = (nodes: DataNode[]) => {
      nodes.forEach(node => {
        if (node.title && node.title.toString().toLowerCase().includes(searchValue.toLowerCase())) {
          // 如果当前节点匹配，将其所有父节点加入展开列表
          expandedKeys.push(node.key);
        }
        if (node.children) {
          traverse(node.children);
        }
      });
    };
    traverse(data);
    return expandedKeys;
  };

  // 树节点展开处理 - 实现手风琴效果（同一时间只展开一个节点）
  const onExpand = (expandedKeys: Key[]) => {
    setExpandedKeys(expandedKeys);
    setAutoExpandParent(false);
  };

  // 树节点选择处理
  const handleTreeSelect = (
    selectedKeys: Key[],
    info: {
      event: 'select';
      selected: boolean;
      node: EventDataNode<DataNode>;
      selectedNodes: DataNode[];
    }
  ) => {
    if (onDeptSelect) {
      onDeptSelect(selectedKeys, info);
    }
  };

  // 过滤树数据
  const filterTreeData = (data: DataNode[], searchValue: string): DataNode[] => {
    if (!searchValue) {
      return data;
    }

    return data.filter(node => {
      const titleMatch = node.title && node.title.toString().toLowerCase().includes(searchValue.toLowerCase());
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
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <ApartmentOutlined />
          <span>部门列表</span>
        </div>
      }
      size="small"
      style={style}
      styles={{ body: { padding: '16px' } }}
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
        <Tree
          showLine
          treeData={filteredTreeData}
          onSelect={handleTreeSelect}
          selectedKeys={selectedKeys}
          expandedKeys={expandedKeys}
          onExpand={onExpand}
          autoExpandParent={autoExpandParent}
          style={{ minHeight: 300, maxHeight: 'calc(100vh - 250px)', overflowY: 'auto' }}
        />
      </Spin>
    </Card>
  );
};

export default DeptTree;