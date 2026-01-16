/**
 * Holiday Excel Import Utils
 * 节假日 Excel 导入工具函数
 */

import * as XLSX from 'xlsx';
import { message } from 'antd';
import dayjs from 'dayjs';
import type { HolidayExcelRow, ImportOptions, HolidayCreateData } from '../types';

/** 验证节假日类型 */
export function isValidHolidayType(value: string): value is 'legal' | 'workday' | 'custom' {
  return ['legal', 'workday', 'custom'].includes(value);
}

/** 下载节假日导入模板 */
export function downloadHolidayTemplate() {
  const template = [
    {
      '日期(YYYY-MM-DD)': '2024-01-01',
      '名称': '元旦',
      '类型(legal/workday/custom)': 'legal',
      '是否休息(true/false)': 'true',
      '备注': '法定节假日',
    },
  ];
  const worksheet = XLSX.utils.json_to_sheet(template);
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, '节假日模板');
  XLSX.writeFile(workbook, '节假日导入模板.xlsx');
  message.success('模板下载成功');
}

/** 处理节假日 Excel 导入 */
export function handleHolidayImport(
  options: ImportOptions,
  batchCreate: (holidays: HolidayCreateData[]) => Promise<boolean | void>
) {
  const { file, onProgress, onSuccess, onError } = options;
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
        throw new Error('文件读取失败');
      }
      const workbook = XLSX.read(data, { type: 'binary' });
      const sheetName = workbook.SheetNames[0];
      const worksheet = workbook.Sheets[sheetName];
      const jsonData = XLSX.utils.sheet_to_json<HolidayExcelRow>(worksheet);

      if (jsonData.length === 0) {
        throw new Error('Excel 文件为空');
      }

      const holidays: HolidayCreateData[] = [];

      for (let i = 0; i < jsonData.length; i++) {
        const row = jsonData[i];
        const rowNum = i + 2; // Excel 行号（从1开始，第1行是表头）

        // 验证日期
        const dateStr = row['日期(YYYY-MM-DD)'] || row['日期'];
        if (!dateStr) {
          throw new Error(`第 ${rowNum} 行：日期字段不能为空`);
        }
        if (!dayjs(dateStr).isValid()) {
          throw new Error(`第 ${rowNum} 行：日期格式不正确 (${dateStr})`);
        }

        // 验证名称
        const holidayName = row['名称'] || row['节假日名称'];
        if (!holidayName) {
          throw new Error(`第 ${rowNum} 行：名称字段不能为空`);
        }

        // 验证类型
        const holidayTypeRaw = row['类型(legal/workday/custom)'] || row['类型'] || 'custom';
        if (!isValidHolidayType(holidayTypeRaw)) {
          throw new Error(`第 ${rowNum} 行：类型值不正确 (${holidayTypeRaw})，应为 legal、workday 或 custom`);
        }

        holidays.push({
          holidayDate: dateStr,
          holidayName,
          isOffday: row['是否休息(true/false)'] !== false,
          holidayType: holidayTypeRaw,
          year: dayjs(dateStr).year(),
          remark: row['备注'] || '',
        });
      }

      await batchCreate(holidays);
      onSuccess?.(undefined);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '导入失败';
      message.error(errorMsg);
      onError?.(error instanceof Error ? error : new Error('导入失败'));
    }
  };

  reader.onerror = () => {
    message.error('文件读取失败');
    onError?.(new Error('文件读取失败'));
  };

  reader.readAsBinaryString(file);
}
