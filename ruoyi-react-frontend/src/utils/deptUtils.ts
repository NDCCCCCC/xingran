/**
 * 部门树工具函数
 *
 * 历史:
 * - DepartmentTreeSelect 与 DeptTree 各自独立实现过 filterExternalOrgDepts;
 *   useWorkstationData 也有一个未使用的副本。本次改造统一收敛到本文件。
 *
 * 节点 ID 约定:
 * - 不同模块的"部门树节点"类型不同:
 *   - `Department` (@/types) 用 `id`
 *   - `DeptTreeNode` (workstations/types) 用 `value` 与 `key`
 *   - antd `DataNode` 用 `key`
 *   - `DepartmentNode` (useWorkstationData 本地类型) 用 `id`
 * - 工具函数通过 `getDeptNodeId` 兼容以上所有形状:优先取 `id`,缺省取 `value`,
 *   再缺省取 `key`。调用者无需关心具体类型。
 *
 * 过滤/收集过程中,过滤掉的子树会被完全丢弃,不会保留为"空 children 节点"。
 */

export interface DeptLikeNode {
  id?: string;
  value?: string;
  key?: string;
  isExternalOrg?: number;
  children?: DeptLikeNode[];
}

/** 从节点中提取 id,兼容多种节点形状。 */
export function getDeptNodeId(node: DeptLikeNode): string {
  return node.id ?? node.value ?? node.key ?? "";
}

/**
 * 仅保留 isExternalOrg===1 的节点及其后代。
 *
 * 行为与原 DeptTree.filterExternalOrgDepts 完全一致:
 * - 节点 isExternalOrg===1 → 保留(连同其全部后代)
 * - 节点 isExternalOrg!==1 但有 externalOrg 后代 → 保留为祖先,只保留 externalOrg 子树
 * - 节点既非 externalOrg 又无 externalOrg 后代 → 丢弃
 *
 * @example
 *   filterExternalOrgDepts([
 *     { id: 'A', isExternalOrg: 1, children: [{ id: 'A1' }] },
 *     { id: 'B', isExternalOrg: 0, children: [{ id: 'B1', isExternalOrg: 1 }] },
 *     { id: 'C' }
 *   ])
 *   // → [
 *   //     { id: 'A', isExternalOrg: 1, children: [{ id: 'A1' }] },
 *   //     { id: 'B', isExternalOrg: 0, children: [{ id: 'B1', isExternalOrg: 1 }] }
 *   //   ]
 */
export function filterExternalOrgDepts<T extends DeptLikeNode>(nodes: T[]): T[] {
  const walk = (list: T[]): T[] => {
    return list.reduce<T[]>((acc, node) => {
      const keptChildren = node.children?.length
        ? walk(node.children as T[])
        : [];
      if (node.isExternalOrg === 1 || keptChildren.length > 0) {
        acc.push({
          ...node,
          children: keptChildren.length > 0 ? keptChildren : undefined,
        } as T);
      }
      return acc;
    }, []);
  };
  return walk(nodes ?? []);
}

/**
 * 在树中查找指定 id 节点(深度优先)。
 * 通过 getDeptNodeId 兼容多种节点形状(见模块顶部说明)。
 * 找到则返回节点引用,未找到返回 null。
 *
 * 注意:返回的是原对象引用,不会克隆。修改返回值会影响原树。
 */
export function findDeptNode<T extends DeptLikeNode>(
  nodes: T[],
  id: string
): T | null {
  for (const node of nodes) {
    if (getDeptNodeId(node) === id) return node;
    if (node.children?.length) {
      const hit = findDeptNode(node.children as T[], id);
      if (hit) return hit;
    }
  }
  return null;
}

/**
 * 收集某节点及其所有后代的 id(含自身)。
 * 通过 getDeptNodeId 兼容多种节点形状。未找到 id 时返回空数组。
 */
export function collectDescendantIds<T extends DeptLikeNode>(
  nodes: T[],
  id: string
): string[] {
  const out: string[] = [];
  const walk = (list: T[]): boolean => {
    for (const n of list) {
      if (getDeptNodeId(n) === id) {
        out.push(getDeptNodeId(n));
        const collect = (subs: T[]) => {
          for (const s of subs) {
            out.push(getDeptNodeId(s));
            if (s.children?.length) collect(s.children as T[]);
          }
        };
        if (n.children?.length) collect(n.children as T[]);
        return true;
      }
      if (n.children?.length && walk(n.children as T[])) return true;
    }
    return false;
  };
  walk(nodes);
  return out;
}

/**
 * 将树节点的 title 收窄为最后一段(去掉祖先路径)。
 *
 * 背景:
 * - useWorkstationData.buildTreeData 会把节点的 title 拼接为完整路径,
 *   如 "中国太平洋财产保险股份有限公司 / 分公司本部 / 人力资源部"。
 *   在"所属机构"下拉里这种全路径有助于辨识,但在"所属部门"下拉里
 *   同一个字符串太冗长。
 * - 此函数仅替换 title 字段,保持 id/value/key 与 children 结构不变。
 *   父级 orgTreeData 仍然使用全路径,不影响辨识。
 *
 * 行为:
 * - 节点有 `title = "A / B / C"` → 替换为 `"C"`
 * - 节点无 `title` 或 title 不含 " / " → 保持原样
 * - 节点有 children → 递归处理
 *
 * @example
 *   trimTitleToLastSegment([
 *     { title: "A / B / C", children: [{ title: "A / B / C / D" }] }
 *   ])
 *   // → [
 *   //     { title: "C", children: [{ title: "D" }] }
 *   //   ]
 */
export function trimTitleToLastSegment<
  T extends { title?: string; children?: T[] },
>(nodes: T[]): T[] {
  const walk = (list: T[]): T[] => {
    return list.map((n) => {
      const nextChildren = n.children?.length
        ? walk(n.children)
        : n.children;
      const nextTitle =
        typeof n.title === "string" && n.title.includes(" / ")
          ? n.title.split(" / ").pop() ?? n.title
          : n.title;
      return { ...n, title: nextTitle, children: nextChildren } as T;
    });
  };
  return walk(nodes ?? []);
}
