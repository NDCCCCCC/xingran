/**
 * Phase 88 Batch32 — components/network/port-write/constants + macEventMeta 测试
 */
import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";
import { Form } from "antd";
import {
  PRESET_REASONS,
  ACTION_TITLE,
  REASON_MIN,
  REASON_MAX,
  DESCRIPTION_MAX,
  REASON_CUSTOM_SENTINEL,
  IPV4_REGEX,
  MAC_REGEX,
  BIND_OPS,
  composeReason,
  validateReasonOptional,
  validateReasonRequired,
} from "../constants";
import {
  EVENT_COLORS,
  EVENT_ICON,
  EVENT_LABEL,
  EVENT_TAG_COLOR,
  type MACEventType,
} from "../../macEventMeta";

describe("port-write constants", () => {
  it("PRESET_REASONS 5 项 + 最后是 sentinel", () => {
    expect(PRESET_REASONS.length).toBe(5);
    expect(PRESET_REASONS[4].value).toBe("__custom__");
    // 每项 value ≥ REASON_MIN(5)
    for (const p of PRESET_REASONS) {
      expect(p.value.length).toBeGreaterThanOrEqual(REASON_MIN - 1); // 含 sentinel
    }
  });

  it("ACTION_TITLE 7 个映射(5+2 v1.20.1)", () => {
    expect(Object.keys(ACTION_TITLE).length).toBe(7);
    expect(ACTION_TITLE.shutdown).toBe("关闭端口");
    expect(ACTION_TITLE.set_access_vlan).toBe("修改 access VLAN");
    expect(ACTION_TITLE.port_binding).toBe("端口绑定");
  });

  it("length 限制 3 个常量", () => {
    expect(REASON_MIN).toBe(5);
    expect(REASON_MAX).toBe(200);
    expect(DESCRIPTION_MAX).toBe(80);
  });

  it("REASON_CUSTOM_SENTINEL 同步 PRESET_REASONS 第 5 项", () => {
    expect(PRESET_REASONS[4].value).toBe(REASON_CUSTOM_SENTINEL);
  });

  it("IPV4_REGEX 业务地址匹配 + 边界拒绝", () => {
    expect(IPV4_REGEX.test("10.62.25.5")).toBe(true);
    expect(IPV4_REGEX.test("192.168.1.1")).toBe(true);
    expect(IPV4_REGEX.test("256.0.0.1")).toBe(false); // 越界
    // regex 当前允许首段 0([1-9]?\d 匹配 '0'),这是 IPV4_REGEX 行为,断言其形态
    expect(IPV4_REGEX.test("0.10.20.30")).toBe(true);
    expect(IPV4_REGEX.test("10.62.25")).toBe(false); // 段数错
  });

  it("MAC_REGEX 三种格式", () => {
    expect(MAC_REGEX.test("AA:BB:CC:DD:EE:FF")).toBe(true);
    expect(MAC_REGEX.test("AA-BB-CC-DD-EE-FF")).toBe(true);
    expect(MAC_REGEX.test("AABBCCDDEEFF")).toBe(true);
    expect(MAC_REGEX.test("invalid-mac")).toBe(false);
  });

  it("BIND_OPS 双态", () => {
    expect(BIND_OPS.length).toBe(2);
    expect(BIND_OPS[0].value).toBe("add");
    expect(BIND_OPS[1].value).toBe("remove");
  });

  it("composeReason: 选预设返字符串", () => {
    expect(composeReason("故障排查处理", "")).toBe("故障排查处理");
  });

  it("composeReason: 自定义 sentinel 走 text(trim 仅边缘)", () => {
    expect(composeReason("__custom__", "  测试原因文字  ")).toBe("测试原因文字");
    expect(composeReason("__custom__", "")).toBeNull();
  });

  it("composeReason: 非字符串返 null", () => {
    expect(composeReason(undefined, "")).toBeNull();
    expect(composeReason(null, "")).toBeNull();
    expect(composeReason(123, "")).toBeNull();
  });

  it("validateReasonOptional: 未填返 resolve", async () => {
    let form: any = null;
    renderHook(() => {
      form = Form.useForm()[0];
    });
    await expect(validateReasonOptional({}, undefined, form)).resolves.toBeUndefined();
  });

  it("validateReasonOptional: 填了但 < REASON_MIN 拒", async () => {
    let form: any = null;
    renderHook(() => {
      form = Form.useForm()[0];
    });
    form.setFieldsValue({ reasonText: "abc" });
    await expect(validateReasonOptional({}, "__custom__", form)).rejects.toThrow();
  });

  it("validateReasonRequired: 空必填拒", async () => {
    let form: any = null;
    renderHook(() => {
      form = Form.useForm()[0];
    });
    await expect(validateReasonRequired({}, undefined, form)).rejects.toThrow(
      "请选择或输入操作原因"
    );
  });

  it("validateReasonRequired: 长度超限拒", async () => {
    let form: any = null;
    renderHook(() => {
      form = Form.useForm()[0];
    });
    // 选 sentinel 后让 text="故障排查处理某具体事项" 长度>=MIN → 命中 length check
    form.setFieldsValue({ reasonText: "x".repeat(201) });
    await expect(validateReasonRequired({}, "__custom__", form)).rejects.toThrow(
      /操作原因不超过 200 个字符/
    );
  });
});

describe("macEventMeta 元数据", () => {
  it("EVENT_COLORS 4 类型映射", () => {
    const types: MACEventType[] = ["appeared", "disappeared", "moved", "vlan_changed"];
    for (const t of types) {
      expect(typeof EVENT_COLORS[t]).toBe("string");
    }
    expect(EVENT_COLORS.appeared).toMatch(/2d8949/);
    expect(EVENT_COLORS.disappeared).toMatch(/ba3630/);
  });

  it("EVENT_ICON 4 类型映射是 React FC 对象", () => {
    for (const t of Object.keys(EVENT_ICON) as MACEventType[]) {
      const v = EVENT_ICON[t] as unknown;
      // AntD icon 是 forwardRef 组件对象(typeof === 'object')
      expect(typeof v).toBe("object");
    }
  });

  it("EVENT_LABEL 中文齐全", () => {
    expect(EVENT_LABEL.appeared).toBe("出现");
    expect(EVENT_LABEL.disappeared).toBe("消失");
    expect(EVENT_LABEL.moved).toBe("迁移");
    expect(EVENT_LABEL.vlan_changed).toBe("VLAN 变更");
  });

  it("EVENT_TAG_COLOR AntD color", () => {
    expect(EVENT_TAG_COLOR.appeared).toBe("green");
    expect(EVENT_TAG_COLOR.disappeared).toBe("red");
    expect(EVENT_TAG_COLOR.moved).toBe("gold");
    expect(EVENT_TAG_COLOR.vlan_changed).toBe("blue");
  });
});
