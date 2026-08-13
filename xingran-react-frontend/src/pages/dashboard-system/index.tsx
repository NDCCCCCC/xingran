/**
 * 仪表盘统一入口组件
 * 根据 URL 参数和路由参数显示不同视图
 *
 * 路由规则：
 * - /dashboard                    # 默认仪表盘（如果无默认则显示欢迎页）
 * - /dashboard?mode=list          # 仪表盘列表
 * - /dashboard/:id                # 查看指定仪表盘
 * - /dashboard/:id?mode=edit      # 编辑指定仪表盘
 */
import { useEffect } from "react";
import { useSearchParams, useNavigate, useLocation } from "react-router-dom";
import { useDashboardStore } from "@/store/dashboardStore";
import { DashboardHome } from "./components/DashboardHome";
import DashboardList from "./components/DashboardList";
import { DashboardView } from "./components/DashboardView";
import { DashboardEdit } from "./components/DashboardEdit";

const DashboardPage: React.FC = () => {
	// 手动解析路径参数，因为 dashboard/* 通配符路由无法使用 useParams
	const location = useLocation();
	const [searchParams] = useSearchParams();
	const navigate = useNavigate();

	// 从路径中提取 ID：/dashboard/:id 或 /dashboard/:id?mode=edit
	const pathParts = location.pathname.split("/").filter(Boolean);
	const id = pathParts.length > 1 ? pathParts[1] : undefined;
	const mode = searchParams.get("mode") as "list" | "edit" | null;

	const { setPageMode } = useDashboardStore();

	// 使用 location.key 作为渲染键，确保 location 变化时组件重新渲染
	const renderKey = `${id}-${mode}-${location.key}`;

	useEffect(() => {
		// 根据URL状态设置页面模式
		if (mode === "list") {
			setPageMode("list");
		} else if (mode === "edit") {
			setPageMode("edit");
		} else if (id) {
			setPageMode("view");
		} else {
			setPageMode("home");
		}
	}, [id, mode, location.key, setPageMode]);

	// 路由逻辑
	// 1. 无id, mode=list → 显示列表
	if (!id && mode === "list") {
		return <DashboardList key={renderKey} />;
	}

	// 2. 有id, mode=edit → 显示编辑
	if (id && mode === "edit") {
		return <DashboardEdit dashboardId={id} key={renderKey} />;
	}

	// 3. 有id, 无mode或mode=view → 显示查看
	if (id) {
		return <DashboardView dashboardId={id} key={renderKey} />;
	}

	// 4. 默认: 显示首页（用户默认 > 系统仪表盘 > 欢迎页）
	return <DashboardHome key={renderKey} />;
};

export default DashboardPage;
