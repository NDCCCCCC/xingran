/**
 * Phase 133 / DS-06：按后端可见集过滤部门树（保持树形结构）。
 *
 * 独立成文件（不放 DeptTree/index.tsx）以避开 react-refresh/only-export-components
 * 门禁：组件文件只导出组件，纯函数单独导出。
 *
 * 语义：可见集（visibleDeptIds）来自后端 tree/dept-options 已收口的结果，前端
 * 不硬编码任何过滤规则；这里只做树形呈现裁剪：
 *   - 节点自身命中 → 连同其子树全部保留；
 *   - 节点未命中但子树有命中 → 保留该节点作为祖先路径，只保留命中的子树；
 *   - 节点与子树都未命中 → 丢弃；
 *   - visibleDeptIds 为空/undefined → 原样返回（后端 tree 已是收口结果）。
 */
export interface VisibleDeptFilterNode {
  key?: string | number;
  children?: VisibleDeptFilterNode[];
}

export function filterTreeByVisibleDeptIds<T extends VisibleDeptFilterNode>(
  nodes: T[],
  visibleDeptIds?: string[]
): T[] {
  if (!visibleDeptIds || visibleDeptIds.length === 0) return nodes;
  const visible = new Set(visibleDeptIds.map(String));
  const walk = (list: T[]): T[] =>
    list.reduce<T[]>((acc, node) => {
      const selfHit = node.key !== undefined && visible.has(String(node.key));
      const keptChildren = node.children?.length ? walk(node.children as T[]) : [];
      if (selfHit) {
        acc.push(node);
      } else if (keptChildren.length > 0) {
        acc.push({ ...node, children: keptChildren } as T);
      }
      return acc;
    }, []);
  return walk(nodes);
}
