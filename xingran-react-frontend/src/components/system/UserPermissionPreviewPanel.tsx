/**
 * UserPermissionPreviewPanel —— 编辑用户页「权限预览」面板
 *
 * 需求（用户 2026-11-18）：编辑用户时，角色字段下方的权限预览应和编辑角色
 * 模态框一样，展示**菜单权限树 + 部门树**（带勾选态），而非 Tag 列表/纯文本。
 *
 * 交互：点「权限预览」→ 并发拉取
 *   ① POST /system/users/permission-preview（表单当前值 → 勾选集 menuIds/deptIds + 锚点明细）
 *   ② POST /system/menus/tree-select（菜单树）
 *   ③ POST /system/departments/tree-select（部门树，配置场景全量——与角色模态框同源）
 * 渲染成 <Tree checkable disabled checkedKeys> 两个并排 Card（与角色模态框同款），
 * 再加数据范围档位说明 + 机构锚点命中明细表。
 *
 * 说明：权限预览是**只读展示**，树节点 disabled（不可勾选），仅反映该用户保存后
 * 会获得的权限；锚点动态解析，故部门树勾选集来自服务端模拟结果。
 *
 * 降级：任一请求失败不抛到页面，内部落 error 态友好提示。
 */
import { useState } from "react";
import { Alert, Button, Card, Col, Row, Space, Table, Tag, Tree, Typography } from "antd";
import type { TreeDataNode } from "antd";
import { EyeOutlined } from "@ant-design/icons";
import { post } from "@/lib/api";
import { previewUserPermission } from "@/lib/systemAnchorApi";
import type { UserPermissionPreview, UserAnchorSource } from "@/types/system";

const { Text } = Typography;

export interface UserPermissionPreviewPanelProps {
  /** 表单当前值（由 Form.Item shouldUpdate 提供） */
  deptId?: string;
  roleIds?: string[];
  postIds?: string[];
  disabled?: boolean;
}

/** 后端树节点（形态与角色页 useRoleData 一致：id/key + xxxName/title + children） */
interface RawTreeNode {
  id?: string;
  key?: string;
  menuName?: string;
  deptName?: string;
  title?: string;
  value?: string;
  children?: RawTreeNode[];
}

/** 把后端树节点转成 antd TreeDataNode（与编辑角色模态框同款映射） */
function toTreeData(nodes: RawTreeNode[]): TreeDataNode[] {
  return nodes.map((n) => ({
    key: String(n.key || n.id || n.value || ""),
    title: n.title || n.menuName || n.deptName || "",
    children: n.children?.length ? toTreeData(n.children) : undefined,
  }));
}

export function UserPermissionPreviewPanel({
  deptId,
  roleIds,
  postIds,
  disabled,
}: UserPermissionPreviewPanelProps) {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<UserPermissionPreview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [menuTree, setMenuTree] = useState<TreeDataNode[]>([]);
  const [deptTree, setDeptTree] = useState<TreeDataNode[]>([]);

  const handlePreview = async () => {
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      // 并发请求：权限模拟 + 两棵树（与编辑角色模态框同源端点）
      const [perm, menuRes, deptRes] = await Promise.all([
        previewUserPermission({ deptId, roleIds, postIds }),
        post<RawTreeNode[]>("/system/menus/tree-select", {}),
        post<RawTreeNode[]>("/system/departments/tree-select", {}),
      ]);
      setResult(perm);
      setMenuTree(toTreeData(menuRes.data ?? []));
      setDeptTree(toTreeData(deptRes.data ?? []));
    } catch (e) {
      setError(`权限预览暂不可用：${e instanceof Error ? e.message : "请稍后重试"}`);
    } finally {
      setLoading(false);
    }
  };

  const anchorColumns = [
    {
      title: "来源",
      dataIndex: "source",
      key: "source",
      width: 70,
      render: (v: string) => (
        <Tag color={v === "role" ? "blue" : "green"}>{v === "role" ? "角色" : "岗位"}</Tag>
      ),
    },
    { title: "名称", dataIndex: "sourceName", key: "sourceName" },
    { title: "锚点模式", dataIndex: "modeLabel", key: "modeLabel" },
    {
      title: "结果",
      key: "result",
      render: (_: unknown, r: UserAnchorSource) =>
        r.resolved ? (
          <Text type="success">
            命中「{r.anchorDeptName || r.anchorDeptId}」· 子树 {r.deptCount} 部门
          </Text>
        ) : (
          <Text type="warning">未命中：{r.reason || "无匹配机构"}</Text>
        ),
    },
  ];

  const treeStyle = {
    background: "var(--theme-bg-secondary)",
    padding: "8px",
    borderRadius: "4px",
    maxHeight: "300px",
    overflowY: "auto" as const,
  };

  return (
    <div style={{ marginTop: 8, padding: "8px 0" }}>
      <Space direction="vertical" style={{ width: "100%" }} size="small">
        <Button
          icon={<EyeOutlined />}
          onClick={handlePreview}
          loading={loading}
          disabled={disabled}
          size="small"
        >
          权限预览（菜单权限树 / 部门树 / 数据范围）
        </Button>
        <Text type="secondary" style={{ fontSize: 12 }}>
          按当前选择的部门 / 角色 /
          岗位模拟解析，保存前即可确认该用户能看到的菜单与机构数据（树中勾选即为将获得的权限）。
        </Text>

        {error && <Alert type="error" showIcon message={error} />}

        {result && (
          <Alert
            type={result.all ? "warning" : "success"}
            showIcon
            message={
              result.all
                ? "角色含「全部数据」，机构授权不参与（全库可见）"
                : `数据范围：${result.dataScopeLabel}；最终可见 ${result.visibleDeptIds.length} 个部门`
            }
          />
        )}

        {result && (
          <Row gutter={12}>
            {/* 左：菜单权限树（与编辑角色模态框一致） */}
            <Col span={12}>
              <Card title="菜单权限（模拟勾选）" size="small">
                {result.menuIds.length ? (
                  <Tree
                    checkable
                    disabled
                    checkedKeys={result.menuIds}
                    treeData={menuTree}
                    style={treeStyle}
                  />
                ) : (
                  <Text type="secondary">（无菜单权限）</Text>
                )}
              </Card>
            </Col>
            {/* 右：部门树（与编辑角色模态框一致，勾选=可见部门） */}
            <Col span={12}>
              <Card
                title="部门权限（模拟可见范围）"
                size="small"
                extra={result.allDepts ? <Tag color="orange">全库可见</Tag> : undefined}
              >
                {result.allDepts ? (
                  <Alert
                    type="info"
                    showIcon
                    message="该用户为全库可见（数据范围=全部数据），机构授权不参与。"
                  />
                ) : result.deptIds.length ? (
                  <Tree
                    checkable
                    disabled
                    checkedKeys={result.deptIds}
                    treeData={deptTree}
                    style={treeStyle}
                  />
                ) : (
                  <Text type="secondary">（无可见部门——仅本人，将只能看到自己）</Text>
                )}
              </Card>
            </Col>
          </Row>
        )}

        {result && (
          <>
            {(result.roleNames.length > 0 || result.postNames.length > 0) && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {result.roleNames.length > 0 && <>角色：{result.roleNames.join("、")}</>}
                {result.roleNames.length > 0 && result.postNames.length > 0 && " · "}
                {result.postNames.length > 0 && <>岗位：{result.postNames.join("、")}</>}
              </Text>
            )}
            {result.anchorSources.length > 0 && (
              <Table
                size="small"
                rowKey={(r) => `${r.source}-${r.sourceName}`}
                dataSource={result.anchorSources}
                columns={anchorColumns}
                pagination={false}
              />
            )}
          </>
        )}
      </Space>
    </div>
  );
}

export default UserPermissionPreviewPanel;
