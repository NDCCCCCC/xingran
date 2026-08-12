/**
 * Holiday Utils
 * 节假日页面工具函数
 *
 * xlsx 库采用 dynamic import，仅在用户点击"导入"或"下载模板"按钮时按需加载
 * （D-06 Wave 2: xlsx 按需加载）
 */

import { getAppMessage } from "@/utils/antdMessage";
import dayjs from "dayjs";
import type { ExcelHolidayRow, ExcelImportOptions, HolidayType } from "./types";
import { batchCreateHolidays, type Holiday } from "@/lib/dutyApi";

/** 解析节假日类型 */
function parseHolidayType(typeStr: string): HolidayType {
  const typeLower = typeStr.toLowerCase();
  if (typeLower.includes("法定") || typeLower === "legal") {
    return "legal";
  } else if (typeLower.includes("调休") || typeLower.includes("工作") || typeLower === "workday") {
    return "workday";
  } else if (typeLower.includes("自定义") || typeLower === "custom") {
    return "custom";
  }
  return "legal";
}

/** 解析是否休息 */
function parseIsOffday(isOffdayStr: string | boolean | number): boolean {
  if (typeof isOffdayStr === "boolean") {
    return isOffdayStr;
  }
  const str = String(isOffdayStr).toLowerCase();
  return str === "是" || str === "true" || str === "1" || str === "休息";
}

/** 处理 Excel 导入 */
export async function handleExcelImport(
  options: ExcelImportOptions,
  onSuccess?: () => void
): Promise<void> {
  const { file, onError } = options;
  try {
    // D-06: 动态加载 xlsx，避免进入 holidays 页面时即加载 ~150KB gzip
    const XLSX = await import("xlsx");
    const data = await file.arrayBuffer();
    const workbook = XLSX.read(data);
    const firstSheet = workbook.Sheets[workbook.SheetNames[0]];
    const jsonData = XLSX.utils.sheet_to_json(firstSheet, { header: 1 }) as unknown[][];

    // 验证Excel格式（跳过表头）
    if (jsonData.length < 2) {
      getAppMessage().error("Excel文件为空或格式不正确");
      return;
    }

    const holidays: ExcelHolidayRow[] = [];

    // 从第二行开始解析（跳过表头）
    for (let i = 1; i < jsonData.length; i++) {
      const row = jsonData[i] as unknown[];
      if (!row[0] && !row[1]) continue; // 跳过空行

      const dateStr = row[0];
      const name = row[1];
      const typeStr = row[2];
      const isOffdayStr = row[3];
      const year = row[4] || new Date().getFullYear();
      const remark = row[5];

      // 解析日期
      let formattedDate: string;
      if (typeof dateStr === "number") {
        // Excel 日期序列号转换
        const date = XLSX.SSF.parse_date_code(dateStr);
        formattedDate = `${date.y}-${String(date.m).padStart(2, "0")}-${String(date.d).padStart(2, "0")}`;
      } else {
        // 字符串日期，尝试多种格式
        const parsed = dayjs(dateStr as string);
        if (!parsed.isValid()) {
          getAppMessage().error(`第 ${i + 1} 行：日期格式不正确 - ${dateStr}`);
          return;
        }
        formattedDate = parsed.format("YYYY-MM-DD");
      }

      // 解析类型
      const holidayType = typeof typeStr === "string" ? parseHolidayType(typeStr) : "legal";

      // 解析是否休息
      const isOffday = parseIsOffday(isOffdayStr as string | number | boolean);

      holidays.push({
        holidayDate: formattedDate,
        holidayName: (name as string) || "节假日",
        isOffday,
        holidayType,
        year: Number(year),
        remark: remark as string | undefined,
      });
    }

    if (holidays.length === 0) {
      getAppMessage().error("没有有效的节假日数据");
      return;
    }

    // 批量创建
    await batchCreateHolidays(holidays as Omit<Holiday, "id" | "createdAt" | "createdBy">[]);
    getAppMessage().success(`成功导入 ${holidays.length} 条节假日记录`);

    onSuccess?.();
    options.onSuccess?.(holidays);
  } catch (error: unknown) {
    console.error("Excel导入失败:", error);
    const errorMsg = error instanceof Error ? error.message : "未知错误";
    getAppMessage().error(`导入失败: ${errorMsg}`);
    onError?.(error instanceof Error ? error : new Error("导入失败"));
  }
}

/** 下载Excel模板 */
export async function downloadTemplate(): Promise<void> {
  // D-06: 动态加载 xlsx，仅在用户点击下载模板时按需加载
  const XLSX = await import("xlsx");

  // 创建模板数据（仅表头，不包含示例数据）
  const template = [
    ["日期", "名称", "类型", "是否休息", "年份", "备注"],
  ];

  // 创建工作簿
  const ws = XLSX.utils.aoa_to_sheet(template);
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, "节假日模板");

  // 下载
  XLSX.writeFile(wb, "节假日导入模板.xlsx");
}