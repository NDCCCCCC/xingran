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
      const keptChildren = node.children?.length ? walk(node.children as T[]) : [];
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
export function findDeptNode<T extends DeptLikeNode>(nodes: T[], id: string): T | null {
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
export function collectDescendantIds<T extends DeptLikeNode>(nodes: T[], id: string): string[] {
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
export function trimTitleToLastSegment<T extends { title?: string; children?: T[] }>(
  nodes: T[]
): T[] {
  const walk = (list: T[]): T[] => {
    return list.map((n) => {
      const nextChildren = n.children?.length ? walk(n.children) : n.children;
      const nextTitle =
        typeof n.title === "string" && n.title.includes(" / ")
          ? (n.title.split(" / ").pop() ?? n.title)
          : n.title;
      return { ...n, title: nextTitle, children: nextChildren } as T;
    });
  };
  return walk(nodes ?? []);
}

/**
 * 全路径树转换(语义 1):把部门树转换为 antd TreeSelect/Tree 期望的
 * `{ title, value, key, children?, isExternalOrg? }` 形状,且 `title` 拼接为
 * 祖先 + 自身的完整路径(以 " / " 连接)。
 *
 * 语义维度(D-LOCKED, Phase 37 CONTEXT):
 * - 全路径显示(从二级部门拼 "A / B / C"),与 `toShortNameDataNode` 的"短名"语义区分。
 * - 两个函数**不可合成一个**(D-LOCKED,关键陷阱):调用方按需选用。
 *
 * `opts.startFromLevel` 控制祖先路径裁剪起点(默认 1):
 * - `1`(默认):不裁剪祖先,所有祖先名都进 title。
 *   顶级节点 `title = node.deptName`;深级 `title = [...allAncestors, node.deptName].join(" / ")`。
 * - `2`:行为与原 `DepartmentTreeSelect.convertDeptTreeData` 的 `currentPath.slice(1)` 等价——
 *   **总是丢弃 ancestors[0]**(即顶级祖先名),无论当前节点深度。
 *   结果:顶级 `title = node.deptName`;孙级及以上 `title = ancestors[1..].join(" / ") + " / " + node.deptName`。
 *   **为保持 UI 文案不变**,`DepartmentTreeSelect` 调用本函数时传 `{ startFromLevel: 2 }`。
 *
 * 替代原函数:
 * - `components/shared/DepartmentTreeSelect.tsx:49` `convertDeptTreeData`(全路径,从二级开始)
 * - `pages/operations/workstations/hooks/useWorkstationData.ts:78` `buildTreeData`(全路径)
 *
 * 字段透传:
 * - `value`/`key` 取自 `getDeptNodeId(node)`(兼容 `id`/`value`/`key` 三种形状)
 * - **`isExternalOrg` 透传**(workstations 高风险依赖:若不透传,`filterExternalOrgDepts`
 *   会把整棵树过滤为空——见 `toFullPathTree` 在 `useWorkstationData` 的消费链路)
 *
 * 泛型约束 `T extends DeptLikeNode & { deptName?: string }`:允许消费方传入
 * `SimpleDept` / `DeptTreeNode` 等任何带 `deptName` 的形状,返回类型叠加 `title/value/key`。
 *
 * @example
 *   toFullPathTree([
 *     { id: "1", deptName: "集团", children: [{ id: "2", deptName: "分公司" }] },
 *   ])
 *   // → [{ title: "集团", value: "1", key: "1", children: [{ title: "集团 / 分公司", value: "2", key: "2" }] }]
 *
 *   toFullPathTree([...], { startFromLevel: 2 })  // 复现旧 convertDeptTreeData.slice(1) 语义
 */
/**
 * 全路径树节点:`toFullPathTree` 的递归输出类型。
 *
 * 每个节点(含深层 children)都叠加 `title/value/key/isExternalOrg?`。
 * 用 `Omit<T, "children">` 避开 T 自身 children 类型(如 `SimpleDept.children: SimpleDept[]`)
 * 与叠加 children 的冲突 —— 否则交集类型的 children 仍是非递归的 `SimpleDept[]`,
 * 无法满足 `trimTitleToLastSegment`/`dedupTreeByKey` 等自引用泛型约束
 * `<T extends { children?: T[] }>`(深层子节点也必须有 title/value/key)。
 */
export type FullPathTreeNode<T> = Omit<T, "children"> & {
  title: string;
  value: string;
  key: string;
  isExternalOrg?: number;
  children?: FullPathTreeNode<T>[];
};

export function toFullPathTree<T extends DeptLikeNode & { deptName?: string }>(
  nodes: T[],
  opts?: { startFromLevel?: 1 | 2 }
): FullPathTreeNode<T>[] {
  const startFromLevel = opts?.startFromLevel ?? 1;
  // startFromLevel=k 意味着 ancestors[0..k-2] 全部丢弃,保留 ancestors[k-1..]
  // (k=1 时不裁剪;k=2 时丢 ancestors[0],即等价于旧 convertDeptTreeData 的 slice(1))
  const ancestorKeepFrom = Math.max(0, startFromLevel - 1);
  type Out = FullPathTreeNode<T>;

  const build = (list: T[], ancestors: string[]): Out[] => {
    return (list ?? []).map((node) => {
      const name = node.deptName ?? "";
      const trimmedAncestors = ancestors.slice(ancestorKeepFrom);
      const title = trimmedAncestors.length === 0 ? name : [...trimmedAncestors, name].join(" / ");
      const id = getDeptNodeId(node);
      const nextChildren = node.children?.length
        ? build(node.children as T[], [...ancestors, name])
        : undefined;
      return {
        ...node,
        title,
        value: id,
        key: id,
        ...(node.isExternalOrg !== undefined ? { isExternalOrg: node.isExternalOrg } : {}),
        children: nextChildren,
      } as Out;
    });
  };

  return build(nodes ?? [], []);
}

/**
 * 短名 DataNode 转换(语义 2):把部门树转换为 antd `DataNode` 期望的
 * `{ title, key, value?, children?, isLeaf? }` 形状,且 `title` **只显示 deptName 短名**,
 * 不拼接祖先路径。
 *
 * 语义维度(D-LOCKED, Phase 37 CONTEXT):
 * - 短名显示(只显示当前节点名),与 `toFullPathTree` 的"全路径"语义区分。
 *
 * 替代原函数:
 * - `components/DeptTree/index.tsx:74` `transformToTreeData`(短名)
 * - `pages/system/notice/hooks/useTargetSelector.ts:52` `convertTree`(短名 + 透传 `...node`)
 * - `pages/system/user/utils.tsx:24` `convertDeptTreeData`(实测为短名)
 *
 * 字段:
 * - `title` = `node.deptName`
 * - `key`/`value` 取自 `getDeptNodeId(node)`
 * - `isLeaf` = 无 children 或 children 长度为 0
 * - `children` 递归;无子节点时为 `undefined`(antd DataNode 约定)
 *
 * @example
 *   toShortNameDataNode([
 *     { id: "1", deptName: "集团", children: [{ id: "2", deptName: "分公司" }] },
 *   ])
 *   // → [{ title: "集团", key: "1", value: "1", children: [{ title: "分公司", key: "2", value: "2", isLeaf: true }] }]
 */
export interface ShortNameDataNode {
  title: string;
  key: string;
  value: string;
  children?: ShortNameDataNode[];
  isLeaf?: boolean;
}

export function toShortNameDataNode<T extends DeptLikeNode & { deptName?: string }>(
  nodes: T[]
): ShortNameDataNode[] {
  const build = (list: T[]): ShortNameDataNode[] => {
    return (list ?? []).map((node) => {
      const id = getDeptNodeId(node);
      const hasChildren = !!node.children?.length;
      return {
        title: node.deptName ?? "",
        key: id,
        value: id,
        children: hasChildren ? build(node.children as T[]) : undefined,
        isLeaf: !hasChildren,
      };
    });
  };
  return build(nodes ?? []);
}

/**
 * 按节点 value/key 去重部门树(深度优先,保留首次出现的节点)。
 *
 * 背景:后端 buildDeptTree 把 ParentID 非空但 Ancestors 为空的历史/导入数据
 * 提升到根,同时又嵌套在其真实父节点下 → 同一部门在树中出现两次 → antd
 * Tree/TreeSelect "Same key exist in the Tree" 告警 + 选中歧义。
 * 本函数在前端兜底去重,保证喂给 TreeSelect 的 treeData 每个 value 唯一。
 * 对 TreeSelect 是安全的(同 value 的重复节点本就无法区分)。
 */
export function dedupTreeByKey<T extends { value?: string; key?: string; children?: T[] }>(
  nodes: T[]
): T[] {
  const seen = new Set<string>();
  const walk = (list: T[]): T[] => {
    const result: T[] = [];
    for (const node of list) {
      const id = node.value ?? node.key ?? "";
      if (!id || seen.has(id)) continue;
      seen.add(id);
      const next = { ...node };
      if (node.children?.length) {
        next.children = walk(node.children);
      }
      result.push(next);
    }
    return result;
  };
  return walk(nodes ?? []);
}
