/**
 * Phase 88 Batch88 — duty/holidays/utils 测试(62 stmts, 24.2% → 高)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { downloadTemplate, handleExcelImport } from "../../utils";
import { batchCreateHolidays } from "@/lib/dutyApi";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const msgApi = {
  success: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  loading: vi.fn(),
};

vi.mock("@/utils/antdMessage", () => ({
  getAppMessage: () => msgApi,
}));

vi.mock("@/lib/dutyApi", () => ({
  batchCreateHolidays: vi.fn(),
}));

const xlsxMocks = {
  utils: {
    json_to_sheet: vi.fn(() => ({})),
    aoa_to_sheet: vi.fn(() => ({})),
    book_new: vi.fn(() => ({})),
    book_append_sheet: vi.fn(),
    sheet_to_json: vi.fn(() => []),
  },
  read: vi.fn(() => ({ SheetNames: ["Sheet1"], Sheets: { Sheet1: {} } })),
  writeFile: vi.fn(),
  SSF: { parse_date_code: vi.fn(() => ({ y: 2026, m: 1, d: 1 })) },
};

vi.mock("xlsx", () => xlsxMocks);

beforeEach(() => {
  msgApi.success.mockClear();
  msgApi.error.mockClear();
  msgApi.info.mockClear();
  msgApi.warning.mockClear();
  msgApi.loading.mockClear();
  xlsxMocks.utils.sheet_to_json.mockReturnValue([]);
  xlsxMocks.utils.json_to_sheet.mockClear();
  xlsxMocks.utils.book_new.mockClear();
  xlsxMocks.utils.book_append_sheet.mockClear();
  xlsxMocks.read.mockClear();
  xlsxMocks.writeFile.mockClear();
  vi.mocked(batchCreateHolidays).mockClear();
});

function makeFile(): File {
  const file = new File([""], "test.xlsx", { type: "application/octet-stream" });
  file.arrayBuffer = vi.fn(() => Promise.resolve(new ArrayBuffer(0)));
  return file;
}

describe("duty holidays utils", () => {
  describe("downloadTemplate", () => {
    it("正常下载 + utils 调用 + writeFile", async () => {
      await downloadTemplate();
      expect(xlsxMocks.utils.book_new).toHaveBeenCalled();
      expect(xlsxMocks.utils.book_append_sheet).toHaveBeenCalled();
      expect(xlsxMocks.writeFile).toHaveBeenCalled();
    });
  });

  describe("handleExcelImport", () => {
    it("1 行合法 → batchCreate + onSuccess + message.success", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        ["日期", "名称", "类型", "是否休息", "year", "remark"],
        ["2026-01-01", "元旦", "法定", "是", 2026, ""],
      ] as any);
      vi.mocked(batchCreateHolidays).mockResolvedValueOnce({ code: 0 } as any);

      const onSuccess = vi.fn();
      await handleExcelImport({ file: makeFile(), onSuccess }, onSuccess);
      expect(batchCreateHolidays).toHaveBeenCalled();
      expect(msgApi.success).toHaveBeenCalled();
    });

    it("类型'法定' → legal", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        [],
        ["2026-01-01", "元旦", "legal", "是", 2026, ""],
      ] as any);
      vi.mocked(batchCreateHolidays).mockResolvedValueOnce({ code: 0 } as any);

      await handleExcelImport({ file: makeFile() });
      const arg = vi.mocked(batchCreateHolidays).mock.calls[0][0];
      expect(arg[0].holidayType).toBe("legal");
    });

    it("类型'workday' → workday", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        [],
        ["2026-04-26", "调休", "workday", "否", 2026, ""],
      ] as any);
      vi.mocked(batchCreateHolidays).mockResolvedValueOnce({ code: 0 } as any);

      await handleExcelImport({ file: makeFile() });
      const arg = vi.mocked(batchCreateHolidays).mock.calls[0][0];
      expect(arg[0].holidayType).toBe("workday");
    });

    it("类型'custom' → custom", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        [],
        ["2026-03-15", "纪念日", "custom", "是", 2026, ""],
      ] as any);
      vi.mocked(batchCreateHolidays).mockResolvedValueOnce({ code: 0 } as any);

      await handleExcelImport({ file: makeFile() });
      const arg = vi.mocked(batchCreateHolidays).mock.calls[0][0];
      expect(arg[0].holidayType).toBe("custom");
    });

    it("类型为 undefined → 回退 legal", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        [],
        ["2026-04-26", "调休", undefined, "否", 2026, ""],
      ] as any);
      vi.mocked(batchCreateHolidays).mockResolvedValueOnce({ code: 0 } as any);

      await handleExcelImport({ file: makeFile() });
      const arg = vi.mocked(batchCreateHolidays).mock.calls[0][0];
      expect(arg[0].holidayType).toBe("legal");
    });

    it("日期格式无效 → message.error 路径", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        [],
        ["bad-date", "x", "法定", "是", 2026, ""],
      ] as any);

      await handleExcelImport({ file: makeFile() });
      expect(msgApi.error).toHaveBeenCalledWith(expect.stringContaining("日期格式不正确"));
    });

    it("数据 <2 行 → Excel 空错误", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([["header"]] as any);

      await handleExcelImport({ file: makeFile() });
      expect(msgApi.error).toHaveBeenCalledWith("Excel文件为空或格式不正确");
    });

    it("全空行 → 没有有效数据错误", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        ["header"],
        [null, null, null, null, null, null],
      ] as any);

      await handleExcelImport({ file: makeFile() });
      expect(msgApi.error).toHaveBeenCalledWith("没有有效的节假日数据");
    });

    it("数组异常 → catch 路径 + onError", async () => {
      // 强制 sheet_to_json 抛错
      xlsxMocks.utils.sheet_to_json.mockImplementationOnce(() => {
        throw new Error("sheet parse failed");
      });

      const onError = vi.fn();
      await handleExcelImport({ file: makeFile(), onError });
      expect(onError).toHaveBeenCalled();
      expect(msgApi.error).toHaveBeenCalledWith(expect.stringContaining("导入失败"));
    });

    it("日期为数字 → SSF.parse_date_code 路径", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        [],
        [45292, "数字日期", "法定", "是", 2026, ""],
      ] as any);
      vi.mocked(batchCreateHolidays).mockResolvedValueOnce({ code: 0 } as any);

      await handleExcelImport({ file: makeFile() });
      expect(batchCreateHolidays).toHaveBeenCalled();
    });
  });
});
