import React, { useState, useEffect } from 'react';
import { Tree, Card, Spin, Input } from 'antd';
import type { DataNode, EventDataNode } from 'antd/es/tree';
import { ApartmentOutlined, SearchOutlined } from '@ant-design/icons';
import { post } from '@/lib/api';

const { Search } = Input;

interface DeptTreeProps {
  onSelect?: (selectedKeys: React.Key[], info: {
    event: 'select';
    selected: boolean;
    node: EventDataNode<DataNode>;
    selectedNodes: DataNode[];
  }) => void;
  selectedKeys?: React.Key[];
  style?: React.CSSProperties;
}

const DeptTree: React.FC<DeptTreeProps> = ({
  onSelect: onDeptSelect,
  selectedKeys = [],
  style
}) => {
  const [treeData, setTreeData] = useState<DataNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);
  const [autoExpandParent, setAutoExpandParent] = useState(true);
  const [prevExpandedKeys, setPrevExpandedKeys] = useState<React.Key[]>([]);

  // 获取所有父节点（有子节点的节点）的 key
  const getParentKeys = (data: DataNode[]): React.Key[] => {
    const parentKeys: React.Key[] = [];
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

  // 加载部门树数据
  const loadDeptTree = async () => {
    setLoading(true);
    try {
      const result = await post('/system/departments/list');
      const deptData = result.data || [];
      const treeData = transformToTreeData(deptData);
      setTreeData(treeData);

      // 默认只展开第一个父节点（手风琴模式）
      const parentKeys = getParentKeys(treeData);
      const firstParentKey = parentKeys.length > 0 ? [parentKeys[0]] : [];
      setExpandedKeys(firstParentKey);
      setPrevExpandedKeys(firstParentKey);
    } catch (error) {
      console.error('加载部门树失败:', error);
      setTreeData([]);
    } finally {
      setLoading(false);
    }
  };

  // 转换部门数据为树形结构
  const transformToTreeData = (data: any[]): DataNode[] => {
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

  // 获取所有节点的key
  const getAllKeys = (data: DataNode[]): React.Key[] => {
    const keys: React.Key[] = [];
    const traverse = (nodes: DataNode[]) => {
      nodes.forEach(node => {
        keys.push(node.key);
        if (node.children) {
          traverse(node.children);
        }
      });
    };
    traverse(data);
    return keys;
  };

  // 搜索处理
  const onSearch = (value: string) => {
    setSearchValue(value);
    if (!value) {
      // 如果搜索为空，只展开第一个父节点（手风琴模式）
      const parentKeys = getParentKeys(treeData);
      const firstParentKey = parentKeys.length > 0 ? [parentKeys[0]] : [];
      setExpandedKeys(firstParentKey);
      setPrevExpandedKeys(firstParentKey);
      setAutoExpandParent(true);
      return;
    }

    // 搜索匹配的节点
    const expandedKeys = getExpandedKeys(treeData, value);
    setExpandedKeys(expandedKeys);
    setPrevExpandedKeys(expandedKeys);
    setAutoExpandParent(true);
  };

  // 获取需要展开的节点key
  const getExpandedKeys = (data: DataNode[], searchValue: string): React.Key[] => {
    const expandedKeys: React.Key[] = [];
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
  const onExpand = (expandedKeys: React.Key[]) => {
    // 获取所有父节点的 key（排除叶子节点）
    const parentKeys = getParentKeys(treeData);

    // 找出当前展开的父节点
    const currentlyExpanded = expandedKeys.filter(key => parentKeys.includes(key));
    const previouslyExpanded = prevExpandedKeys.filter(key => parentKeys.includes(key));

    // 判断是展开还是折叠操作
    if (currentlyExpanded.length > previouslyExpanded.length) {
      // 展开操作：找出新展开的节点，只保留它（手风琴效果）
      const newExpandedKey = currentlyExpanded.find(key => !previouslyExpanded.includes(key));
      if (newExpandedKey) {
        setExpandedKeys([newExpandedKey]);
        setPrevExpandedKeys([newExpandedKey]);
      } else {
        setExpandedKeys(expandedKeys);
        setPrevExpandedKeys(expandedKeys);
      }
    } else {
      // 折叠操作：使用新的展开键列表
      setExpandedKeys(expandedKeys);
      setPrevExpandedKeys(expandedKeys);
    }
    setAutoExpandParent(false);
  };

  // 树节点选择处理
  const handleTreeSelect = (
    selectedKeys: React.Key[],
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
  }, []);

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
    >
      <Search
        placeholder="搜索部门"
        value={searchValue}
        onChange={(e) => setSearchValue(e.target.value)}
        onSearch={onSearch}
        style={{ marginBottom: 8 }}
        allowClear
        prefix={<SearchOutlined />}
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
          style={{ minHeight: 300, maxHeight: 500, overflowY: 'auto' }}
        />
      </Spin>
    </Card>
  );
};

export default DeptTree;