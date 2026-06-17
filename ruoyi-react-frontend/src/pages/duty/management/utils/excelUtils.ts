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

const VALID_HOLIDAY_TYPES = ["legal", "workday", "custom"] as const;
type HolidayType = "legal" | "workday" | "custom";

function isValidHolidayType(value: string): value is HolidayType {
  return VALID_HOLIDAY_TYPES.includes(value as HolidayType);
}

const TEMPLATE_DATA = [
  {
    "日期(YYYY-MM-DD)": "2024-01-01",
    "名称": "元旦",
    "类型(legal/workday/custom)": "legal",
    "是否休息(true/false)": "true",
    "备注": "法定节假日",
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

function getRowType(row: HolidayExcelRow, rowNum: number): HolidayType {
  const holidayTypeRaw = row["类型(legal/workday/custom)"] || row["类型"] || "custom";
  if (!isValidHolidayType(holidayTypeRaw)) {
    throw new Error(`第 ${rowNum} 行：类型值不正确 (${holidayTypeRaw})，应为 legal、workday 或 custom`);
  }
  return holidayTypeRaw;
}

function parseHolidayRow(row: HolidayExcelRow, rowNum: number): HolidayCreateData {
  const dateStr = getRowDate(row, rowNum);
  const holidayName = getRowName(row, rowNum);
  const holidayType = getRowType(row, rowNum);

  return {
    holidayDate: dateStr,
    holidayName,
    isOffday: row["是否休息(true/false)"] !== false,
    holidayType: holidayType,
    year: dayjs(dateStr).year(),
    remark: row["备注"] || "",
  };
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
  batchCreate: (holidays: HolidayCreateData[]) => Promise<boolean | void>
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

      const holidays = jsonData.map((row, index) => parseHolidayRow(row, index + 2));

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