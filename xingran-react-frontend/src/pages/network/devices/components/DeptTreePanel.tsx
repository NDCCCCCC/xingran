/**
 * Phase 67-T3: 左侧部门树面板（原 index.tsx :670-679 纯搬迁）
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
    width={360}
    className="dept-list-sider"
    style={{
      background: "#fff",
      padding: "0 16px 16px 0",
      borderRight: "1px solid var(--theme-border-primary)",
    }}
  >
    <DeptTree onSelect={onSelect} selectedKeys={selectedDeptId ? [selectedDeptId] : []} />
  </Sider>
);

export default DeptTreePanel;
