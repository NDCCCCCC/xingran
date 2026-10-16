/**
 * Phase 88 Batch77 — duty/management/excelUtils 测试(56 stmts, 3.6% → 高)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { downloadHolidayTemplate, handleHolidayImport } from "../excelUtils";
import { getAppMessage } from "@/utils/antdMessage";

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

const xlsxMocks = {
  utils: {
    json_to_sheet: vi.fn(() => ({})),
    book_new: vi.fn(() => ({})),
    book_append_sheet: vi.fn(),
    sheet_to_json: vi.fn(() => []),
  },
  read: vi.fn(() => ({ SheetNames: ["Sheet1"], Sheets: { Sheet1: {} } })),
  writeFile: vi.fn(),
};

vi.mock("xlsx", () => xlsxMocks);

class MockFileReader {
  onload: ((e: any) => void) | null = null;
  onerror: ((e: any) => void) | null = null;
  onprogress: ((e: any) => void) | null = null;
  readAsBinaryString(_file: Blob) {
    /* set per-test */
  }
}

type ReaderBehavior = "load" | "error" | "progress";

function makeReader(
  behavior: ReaderBehavior,
  payload?: { loaded?: number; total?: number }
): MockFileReader {
  const r = new MockFileReader();
  r.readAsBinaryString = function () {
    setTimeout(() => {
      if (behavior === "load") {
        r.onload?.({ target: { result: "binary" } });
      } else if (behavior === "error") {
        r.onerror?.({});
      } else {
        r.onprogress?.({
          lengthComputable: true,
          loaded: payload?.loaded ?? 50,
          total: payload?.total ?? 100,
        });
        r.onload?.({ target: { result: "binary" } });
      }
    }, 10);
  };
  return r;
}

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
  (globalThis as any).FileReader = function () {
    return new MockFileReader();
  };
});

function makeFile(): File {
  return new File([""], "test.xlsx", { type: "application/octet-stream" });
}

describe("duty excelUtils", () => {
  describe("downloadHolidayTemplate", () => {
    it("正常下载: xlsx utils 调用 + message.success", async () => {
      await downloadHolidayTemplate();
      expect(msgApi.success).toHaveBeenCalledWith("模板下载成功");
      expect(xlsxMocks.utils.json_to_sheet).toHaveBeenCalled();
    });
  });

  describe("handleHolidayImport", () => {
    function installReader(r: MockFileReader) {
      // 用普通函数,确保可被 `new` 调用
      function Reader() {
        return r;
      }
      (globalThis as any).FileReader = Reader as any;
    }

    it("1 行合法 → batchCreate + onSuccess", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        {
          "日期(YYYY-MM-DD)": "2026-01-01",
          名称: "元旦",
          "类型(legal/workday/custom)": "legal",
          "是否休息(true/false)": "true",
          备注: "",
        },
      ] as any);
      installReader(makeReader("load"));

      const batchCreate = vi.fn().mockResolvedValueOnce(true);
      const onSuccess = vi.fn();
      await handleHolidayImport({ file: makeFile(), onSuccess }, batchCreate);
      await new Promise((r) => setTimeout(r, 50));

      expect(batchCreate).toHaveBeenCalled();
      expect(onSuccess).toHaveBeenCalled();
    });

    it("数据为空 → onError + 'Excel 文件为空'", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([] as any);
      installReader(makeReader("load"));

      const onError = vi.fn();
      await handleHolidayImport({ file: makeFile(), onError }, vi.fn());
      await new Promise((r) => setTimeout(r, 50));

      expect(onError).toHaveBeenCalled();
      expect(msgApi.error).toHaveBeenCalledWith("Excel 文件为空");
    });

    it("日期字段为空 → onError 含行号", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        { 名称: "x", "类型(legal/workday/custom)": "custom" },
      ] as any);
      installReader(makeReader("load"));

      const onError = vi.fn();
      await handleHolidayImport({ file: makeFile(), onError }, vi.fn());
      await new Promise((r) => setTimeout(r, 50));

      const errArg = onError.mock.calls[0][0] as Error;
      expect(errArg.message).toContain("日期字段不能为空");
    });

    it("类型非法 → onError 含期望值", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        {
          "日期(YYYY-MM-DD)": "2026-01-01",
          名称: "x",
          "类型(legal/workday/custom)": "bogus",
        },
      ] as any);
      installReader(makeReader("load"));

      const onError = vi.fn();
      await handleHolidayImport({ file: makeFile(), onError }, vi.fn());
      await new Promise((r) => setTimeout(r, 50));

      const errArg = onError.mock.calls[0][0] as Error;
      expect(errArg.message).toContain("类型值不正确");
    });

    it("reader.onerror → message.error + onError", async () => {
      installReader(makeReader("error"));

      const onError = vi.fn();
      await handleHolidayImport({ file: makeFile(), onError }, vi.fn());
      await new Promise((r) => setTimeout(r, 50));

      expect(onError).toHaveBeenCalled();
      expect(msgApi.error).toHaveBeenCalledWith("文件读取失败");
    });

    it("onProgress 触发: percent=50", async () => {
      installReader(makeReader("progress", { loaded: 50, total: 100 }));
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        {
          "日期(YYYY-MM-DD)": "2026-01-01",
          名称: "x",
          "类型(legal/workday/custom)": "legal",
        },
      ] as any);

      const onProgress = vi.fn();
      const batchCreate = vi.fn().mockResolvedValueOnce(true);
      await handleHolidayImport({ file: makeFile(), onProgress }, batchCreate);
      await new Promise((r) => setTimeout(r, 50));

      expect(onProgress).toHaveBeenCalledWith({ percent: 50 });
    });

    it("类型字段缺失 → 回退 custom", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        {
          "日期(YYYY-MM-DD)": "2026-05-01",
          名称: "劳动节",
          "是否休息(true/false)": "true",
        },
      ] as any);
      installReader(makeReader("load"));

      const batchCreate = vi.fn().mockResolvedValueOnce(true);
      await handleHolidayImport({ file: makeFile() }, batchCreate);
      await new Promise((r) => setTimeout(r, 50));

      expect(batchCreate).toHaveBeenCalled();
      const args = batchCreate.mock.calls[0][0];
      expect(args[0].holidayType).toBe("custom");
    });

    it("日期格式无效 → onError", async () => {
      xlsxMocks.utils.sheet_to_json.mockReturnValueOnce([
        {
          "日期(YYYY-MM-DD)": "not-a-date",
          名称: "x",
          "类型(legal/workday/custom)": "legal",
        },
      ] as any);
      installReader(makeReader("load"));

      const onError = vi.fn();
      await handleHolidayImport({ file: makeFile(), onError }, vi.fn());
      await new Promise((r) => setTimeout(r, 50));

      const errArg = onError.mock.calls[0][0] as Error;
      expect(errArg.message).toContain("日期格式不正确");
    });
  });
});
