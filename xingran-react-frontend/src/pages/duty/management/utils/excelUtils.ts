/**
 * Holiday Excel Import Utils
 * 节假日 Excel 导入工具函数
 *
 * xlsx 库采用 dynamic import，仅在用户点击"导入"或"下载模板"按钮时按需加载
 * （D-06 Wave 2: xlsx 按需加载）
 */

import { getAppMessage } from "@/utils/antdMessage";
import dayjs from "dayjs";
import type { HolidayExcelRow, ImportOptions, HolidayCreateData } from "../types";

/** 静态 fallback 类型白名单（字典不可用/未命中时的行为基线） */
const STATIC_VALID_HOLIDAY_TYPES = ["legal", "workday", "custom"] as const;

/**
 * 计算导入允许的节假日类型集合 = 字典值集合 ∪ 静态 fallback。
 *
 * 背景：节假日类型是数据字典（duty_holiday_type），管理员可新增类型。
 * 静态白名单会让新增类型的 Excel 导入报「类型值不正确」，
 * 这里改为「字典值 ∪ 静态值」，不传字典值时保持原静态白名单行为。
 */
export function resolveValidHolidayTypes(dictValues?: readonly string[]): string[] {
  const allowed = new Set<string>(STATIC_VALID_HOLIDAY_TYPES);
  for (const value of dictValues ?? []) {
    const trimmed = value?.trim();
    if (trimmed) allowed.add(trimmed);
  }
  return [...allowed];
}

const TEMPLATE_DATA = [
  {
    "日期(YYYY-MM-DD)": "2024-01-01",
    名称: "元旦",
    "类型(legal/workday/custom)": "legal",
    "是否休息(true/false)": "true",
    备注: "法定节假日",
  },
];

function getRowDate(row: HolidayExcelRow, rowNum: number): string {
  const dateStr = row["日期(YYYY-MM-DD)"] || row["日期"];
  if (!dateStr) {
    throw new Error(`第 ${rowNum} 行：日期字段不能为空`);
  }
  if (!dayjs(dateStr).isValid()) {
    throw new Error(`第 ${rowNum} 行：日期格式不正确 (${dateStr})`);
  }
  return dateStr;
}

function getRowName(row: HolidayExcelRow, rowNum: number): string {
  const holidayName = row["名称"] || row["节假日名称"];
  if (!holidayName) {
    throw new Error(`第 ${rowNum} 行：名称字段不能为空`);
  }
  return holidayName;
}

function getRowType(row: HolidayExcelRow, rowNum: number, allowedTypes: readonly string[]): string {
  const holidayTypeRaw = row["类型(legal/workday/custom)"] || row["类型"] || "custom";
  if (!allowedTypes.includes(holidayTypeRaw)) {
    throw new Error(
      `第 ${rowNum} 行：类型值不正确 (${holidayTypeRaw})，应为 ${allowedTypes.join("、")}`
    );
  }
  return holidayTypeRaw;
}

function parseHolidayRow(
  row: HolidayExcelRow,
  rowNum: number,
  allowedTypes: readonly string[]
): HolidayCreateData {
  const dateStr = getRowDate(row, rowNum);
  const holidayName = getRowName(row, rowNum);
  const holidayType = getRowType(row, rowNum, allowedTypes);

  return {
    holidayDate: dateStr,
    holidayName,
    isOffday: row["是否休息(true/false)"] !== false,
    // 字典可扩展节假日类型，值域宽于前端既有联合类型
    holidayType: holidayType as HolidayCreateData["holidayType"],
    year: dayjs(dateStr).year(),
    remark: row["备注"] || "",
  };
}

/**
 * 校验并解析节假日 Excel 行（可传字典值集合扩展类型白名单）。
 * @param rows Excel 行数据
 * @param dictValues duty_holiday_type 字典的 dictValue 集合（不传则仅静态白名单）
 */
export function validateHolidayRows(
  rows: HolidayExcelRow[],
  dictValues?: readonly string[]
): HolidayCreateData[] {
  const allowedTypes = resolveValidHolidayTypes(dictValues);
  return rows.map((row, index) => parseHolidayRow(row, index + 2, allowedTypes));
}

export async function downloadHolidayTemplate() {
  // D-06: 动态加载 xlsx，仅在用户点击下载模板时按需加载
  const XLSX = await import("xlsx");
  const worksheet = XLSX.utils.json_to_sheet(TEMPLATE_DATA);
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, "节假日模板");
  XLSX.writeFile(workbook, "节假日导入模板.xlsx");
  getAppMessage().success("模板下载成功");
}

export async function handleHolidayImport(
  options: ImportOptions,
  batchCreate: (holidays: HolidayCreateData[]) => Promise<boolean | void>,
  holidayTypeValues?: readonly string[]
): Promise<void> {
  const { file, onProgress, onSuccess, onError } = options;

  // D-06: 动态加载 xlsx，仅在用户点击导入按钮时按需加载
  const XLSX = await import("xlsx");

  const reader = new FileReader();

  reader.onprogress = (e) => {
    if (e.lengthComputable && onProgress) {
      const percent = Math.round((e.loaded / e.total) * 100);
      onProgress({ percent });
    }
  };

  reader.onload = async (e) => {
    try {
      const data = e.target?.result;
      if (!data) {
        throw new Error("文件读取失败");
      }

      const workbook = XLSX.read(data, { type: "binary" });
      const sheetName = workbook.SheetNames[0];
      const worksheet = workbook.Sheets[sheetName];
      const jsonData = XLSX.utils.sheet_to_json<HolidayExcelRow>(worksheet);

      if (jsonData.length === 0) {
        throw new Error("Excel 文件为空");
      }

      const holidays = validateHolidayRows(jsonData, holidayTypeValues);

      await batchCreate(holidays);
      onSuccess?.(undefined);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : "导入失败";
      getAppMessage().error(errorMsg);
      onError?.(error instanceof Error ? error : new Error("导入失败"));
    }
  };

  reader.onerror = () => {
    getAppMessage().error("文件读取失败");
    onError?.(new Error("文件读取失败"));
  };

  reader.readAsBinaryString(file);
}
