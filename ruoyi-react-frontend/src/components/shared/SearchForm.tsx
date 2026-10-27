/**
 * Phase 66 契约组件 — SearchForm（schema 驱动搜索区，66-UI-SPEC §2）
 *
 * 用法（通常经 PageTable 的 searchFields/searchChildren 使用，也可独立）：
 * ```tsx
 * <SearchForm
 *   form={manager.searchForm}
 *   fields={[
 *     { type: "input", name: "postCode", label: "岗位编码" },
 *     { type: "select", name: "status", label: "状态", options: STATUS_OPTIONS },
 *     { type: "textarea", name: "remark", label: "备注", placeholder: "模糊匹配备注", rows: 3 },
 *   ]}
 *   onSearch={handleSearch} onReset={handleReset} onRefresh={handleRefresh}
 * />
 * ```
 *
 * 契约：渲染端点只允许 AppInput / AppTextArea / AppSelect / RemoteSelect / custom.render，
 * 内部禁止裸 <Input>/<Select>（65 §3 纪律，封装层内亦自律）。
 */
import { Card, Button, Form } from "antd";
import { SearchOutlined, ReloadOutlined } from "@ant-design/icons";
import type { FormInstance } from "antd";
import type { ReactNode } from "react";
import { AppInput } from "@/components/ui/AppInput";
import { AppSelect } from "@/components/ui/AppSelect";
import { AppTextArea } from "@/components/ui/AppTextArea";
import { RemoteSelect } from "@/components/ui/RemoteSelect";

/** 搜索字段 schema（§2；remoteSelect 端点为 RemoteSelect，T5 接通前的占位见 renderField） */
export type SearchFieldOption = { label: string; value: string | number | boolean | null };

export type SearchField =
  | { type: "input"; name: string; label: string; placeholder?: string }
  // Phase 70（TEXT-03）：多行搜索字段；rows/autoSize 显式透传（T1 审计定稿：不下沉默认）
  | {
      type: "textarea";
      name: string;
      label: string;
      placeholder?: string;
      rows?: number;
      autoSize?: boolean | { minRows: number; maxRows: number };
    }
  | { type: "select"; name: string; label: string; options: SearchFieldOption[]; placeholder?: string }
  | {
      type: "remoteSelect";
      name: string;
      label: string;
      /** RemoteSelect 契约（65 §2.3）：fetcher(keyword)=>Promise<Option[]>；其余透传 */
      fetcher: (keyword: string) => Promise<SearchFieldOption[]>;
      initialOptions?: SearchFieldOption[];
      debounceMs?: number;
      placeholder?: string;
    }
  | {
      type: "custom";
      name: string;
      /** 受控渲染：收 value/onChange，由 SearchForm 接入表单（filters）状态 */
      render: (value: unknown, onChange: (v: unknown) => void) => ReactNode;
    };

export interface SearchFormProps {
  /** useTableManager 暴露的 searchForm 实例（filters 读写都走它） */
  form: FormInstance;
  /** schema 字段（与 children 二选一） */
  fields?: SearchField[];
  /** 逃生口：自行渲染 Form.Item（同 Form 上下文，配合 form 实例） */
  children?: ReactNode;
  /** 搜索（=useTableManager.handleSearch：filterEmpty+回第 1 页） */
  onSearch: () => void;
  /** 重置（=useTableManager.handleReset：清 filters 持久化+清排序+回第 1 页） */
  onReset: () => void;
  /** 刷新（=loadData） */
  onRefresh: () => void;
}

/** 渲染单个字段（input→AppInput / textarea→AppTextArea / select→AppSelect / remoteSelect→T5 占位 / custom→受控 render） */
function renderField(field: SearchField, onSearch: () => void): ReactNode {
  switch (field.type) {
    case "input":
      return (
        <Form.Item key={field.name} name={field.name} label={field.label}>
          <AppInput placeholder={field.placeholder} onPressEnter={onSearch} />
        </Form.Item>
      );
    case "textarea":
      // Phase 70：多行字段不传 onPressEnter——多行回车必须换行（区别于 input 分支）；
      // 清空 = onChange('') 直通 filters（空串由 useTableManager.filterEmpty 过滤）
      return (
        <Form.Item key={field.name} name={field.name} label={field.label}>
          <AppTextArea
            placeholder={field.placeholder}
            rows={field.rows}
            autoSize={field.autoSize}
          />
        </Form.Item>
      );
    case "select":
      return (
        <Form.Item key={field.name} name={field.name} label={field.label}>
          <AppSelect
            options={field.options}
            placeholder={field.placeholder}
            style={{ minWidth: 140 }}
          />
        </Form.Item>
      );
    case "remoteSelect":
      // T5 契约（§4）：fetcher+debounceMs=300+initialOptions；竞态/失败/清空语义在
      // RemoteSelect 内建，这里只接线
      return (
        <Form.Item key={field.name} name={field.name} label={field.label}>
          <RemoteSelect
            fetcher={field.fetcher}
            initialOptions={field.initialOptions}
            debounceMs={field.debounceMs}
            placeholder={field.placeholder}
            style={{ minWidth: 140 }}
          />
        </Form.Item>
      );
    case "custom":
      return (
        <Form.Item key={field.name} noStyle shouldUpdate>
          {(form) =>
            field.render(
              form.getFieldValue(field.name),
              (v: unknown) => form.setFieldsValue({ [field.name]: v })
            )
          }
        </Form.Item>
      );
    default:
      return null;
  }
}

/**
 * SearchForm — 63 惯例内联搜索卡（layout="inline"，Card marginBottom:16）；
 * 按钮 搜索/重置/刷新（刷新 icon 按钮 aria-label+title，64 FIX-05 口径）。
 */
export function SearchForm({ form, fields, children, onSearch, onReset, onRefresh }: SearchFormProps) {
  return (
    <Card style={{ marginBottom: 16 }}>
      <Form form={form} layout="inline">
        {fields?.map((f) => renderField(f, onSearch))}
        {children}
        <Form.Item>
          <Button type="primary" icon={<SearchOutlined />} onClick={onSearch}>
            搜索
          </Button>
        </Form.Item>
        <Form.Item>
          <Button onClick={onReset}>重置</Button>
        </Form.Item>
        <Form.Item>
          <Button type="text" icon={<ReloadOutlined />} aria-label="刷新" title="刷新" onClick={onRefresh} />
        </Form.Item>
      </Form>
    </Card>
  );
}

export default SearchForm;
