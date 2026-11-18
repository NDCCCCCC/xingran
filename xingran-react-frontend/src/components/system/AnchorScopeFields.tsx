/**
 * AnchorScopeFields —— 角色/岗位「机构锚点」受控字段组
 *
 * 契约来源：`.planning/notes/261118-anchor-scope-contract.md` §二 2.2 / §五 / §六
 *
 * 职责：只渲染**机构锚点**这一个维度（模式 + 参数），不感知 dataScope。
 * ⚠️ 数据范围（dataScope）与机构锚点是**两个独立控件、两个独立维度**：
 * 锚点绝不参与 `MAX(data_scope)` 排序，也不得与数据范围合并成同一个下拉（契约 C1）。
 *
 * 受控契约：
 * - 入参 `mode` / `param` 是唯一事实来源，组件内部不持有状态；
 * - 任何用户交互都通过 `onChange({ mode, param })` 向外抛出**完整的下一组值**；
 * - `mode` 切到 0 / 3 时 `param` 会被清空（形状不匹配的旧参数同样清空），
 *   避免"选了层级却残留部门 UUID"之类的静默脏值。
 *
 * 数据获取不在本组件内（部门树由调用方通过 `departments` 喂入）：
 * 与 `DepartmentTreeSelect` 的 D-LOCKED 约定一致 —— 组件不调 hook。
 */
import { Form, Row, Col } from "antd";
import { AppSelect } from "@/components/ui";
import { DepartmentTreeSelect } from "@/components/shared";
import type { SimpleDept } from "@/lib/dutyApi";
import {
  ANCHOR_MODE_OPTIONS,
  ANCHOR_LEVEL_OPTIONS,
  normalizeAnchorMode,
  retargetAnchorParam,
} from "@/constants/anchor";
import type { AnchorMode } from "@/types/system";

/** 受控值（一次性成对抛出，避免父级出现 mode 已变、param 未清的中间态） */
export interface AnchorScopeChange {
  mode: AnchorMode;
  param: string;
}

export interface AnchorScopeFieldsProps {
  /** 当前锚点模式（0/2/3/5）；缺省/非法值按 0 处理 */
  mode?: number | null;
  /** 当前锚点参数：mode=2 为 dept_id，mode=5 为 "1"~"4"，其余为空 */
  param?: string | null;
  onChange?: (next: AnchorScopeChange) => void;
  /** 部门树数据源（仅 mode=2 使用；由调用方喂入，组件不取数） */
  departments?: SimpleDept[];
  /** 部门树加载中 */
  departmentsLoading?: boolean;
  disabled?: boolean;
}

export function AnchorScopeFields({
  mode,
  param,
  onChange,
  departments,
  departmentsLoading = false,
  disabled = false,
}: AnchorScopeFieldsProps) {
  const currentMode = normalizeAnchorMode(mode);
  const currentParam = param ?? "";

  // 模式切换：参数按新模式的形状重定位（0/3 → 必定清空；2↔5 形状不符则清空）
  const handleModeChange = (value: unknown) => {
    const nextMode = normalizeAnchorMode(value);
    onChange?.({ mode: nextMode, param: retargetAnchorParam(nextMode, currentParam) });
  };

  // 参数变更：只在需要参数的两种模式下允许（避免脏写）
  const handleParamChange = (value: unknown) => {
    onChange?.({ mode: currentMode, param: typeof value === "string" ? value : "" });
  };

  return (
    <>
      <Row gutter={16}>
        <Col span={currentMode === 2 || currentMode === 5 ? 12 : 24}>
          <Form.Item
            label="机构授权模式"
            tooltip="与「数据范围」是两个独立维度：机构授权在角色/岗位原有范围之外额外叠加（宽者胜），未命中不会回退到本部门。0=不参与。"
          >
            {/* 形态分类器②④：锚点模式固定枚举（业务禁搜） */}
            <AppSelect
              value={currentMode}
              onChange={handleModeChange}
              options={[...ANCHOR_MODE_OPTIONS]}
              showSearch={false}
              disabled={disabled}
              placeholder="请选择机构授权模式"
            />
          </Form.Item>
        </Col>

        {currentMode === 2 && (
          <Col span={12}>
            <Form.Item label="指定部门（含下级）" required>
              <DepartmentTreeSelect
                value={currentParam}
                // allowClear 清空 → 传空串，保持参数类型恒为 string
                onChange={(value) => handleParamChange(value ?? "")}
                departments={departments}
                loading={departmentsLoading}
                disabled={disabled}
                placeholder="请选择锚点部门"
              />
            </Form.Item>
          </Col>
        )}

        {currentMode === 5 && (
          <Col span={12}>
            <Form.Item label="指定机构层级（最近）" required>
              {/* 形态分类器②④：机构层级固定枚举（业务禁搜）；值必须是 "1"~"4" 字符串 */}
              <AppSelect
                value={currentParam === "" ? undefined : currentParam}
                onChange={handleParamChange}
                options={[...ANCHOR_LEVEL_OPTIONS]}
                showSearch={false}
                disabled={disabled}
                placeholder="请选择机构层级"
              />
            </Form.Item>
          </Col>
        )}
      </Row>

      {currentMode === 3 && (
        <div style={{ color: "var(--theme-text-tertiary)", fontSize: 12 }}>
          无需参数：沿用户所在部门的祖先链由近及远，取第一个「外部机构」作为锚点。
        </div>
      )}
      {currentMode === 0 && (
        <div style={{ color: "var(--theme-text-tertiary)", fontSize: 12 }}>
          不参与机构授权：该角色/岗位不产生任何机构维度范围。
        </div>
      )}
      {currentMode === 5 && (
        <div style={{ color: "var(--theme-text-tertiary)", fontSize: 12 }}>
          无需锚点部门：沿用户所在部门的祖先链由近及远，取第一个指定层级的机构。
        </div>
      )}
    </>
  );
}

export default AnchorScopeFields;
