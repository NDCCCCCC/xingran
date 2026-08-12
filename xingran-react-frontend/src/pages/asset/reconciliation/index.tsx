/**
 * 资产对账父路由 — 302 跳转到 Dashboard (Phase 42 R1 / D-04)
 *
 * 不渲染任何内容,仅作为菜单 click target,访问时直接 Navigate
 * 跳转到 /asset/reconciliation/dashboard。
 */
import { Navigate } from "react-router-dom";

const ReconciliationIndex = () => {
  return <Navigate to="/asset/reconciliation/dashboard" replace />;
};

export default ReconciliationIndex;
