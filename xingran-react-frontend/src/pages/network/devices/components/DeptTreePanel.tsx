/**
 * Phase 67-T3: 左侧部门树面板（原 index.tsx :670-679 纯搬迁）
 *
 * UI 统一整改：Sider 改为 transparent 透出全局奶油画布，白卡由 DeptTree 自带
 * <Card> 承担 —— 与系统管理用户页 xr-dept-panel 范本一致。去掉自设 #fff 白底
 * 与自绘 borderRight，消除设备管理页「多一块白色背景」。
 */
import { Layout } from "antd";
import DeptTree from "@/components/DeptTree";

const { Sider } = Layout;

interface DeptTreePanelProps {
  selectedDeptId: string;
  onSelect: (selectedKeys: React.Key[]) => void;
}

const DeptTreePanel: React.FC<DeptTreePanelProps> = ({ selectedDeptId, onSelect }) => (
  <Sider
    width={250}
    className="dept-list-sider xr-dept-panel"
    style={{ background: "transparent", padding: "0 14px 16px 0" }}
  >
    <DeptTree onSelect={onSelect} selectedKeys={selectedDeptId ? [selectedDeptId] : []} />
  </Sider>
);

export default DeptTreePanel;
