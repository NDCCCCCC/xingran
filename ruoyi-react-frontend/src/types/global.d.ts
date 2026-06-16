/**
 * 全局类型声明
 */

declare module "@breejs/later" {
  export interface Schedule {
    schedules: unknown[];
  }
  export function schedule(cronExpression: string): Schedule;
  export function timeout(expression: string, date?: Date): Date;
}

declare module "sm-crypto" {
  export function sm2Encrypt(data: string, publicKey: string): string;
  export function sm2Decrypt(encryptedData: string, privateKey: string): string;
  export function sm3Encrypt(data: string): string;
  export function sm4Encrypt(data: string, key: string): string;
  export function sm4Decrypt(encryptedData: string, key: string): string;
  export function doEncrypt(message: string, publicKey: string): string;
  export function doDecrypt(encryptedText: string, privateKey: string): string;
}

declare module "@uiw/react-md-editor" {
  import type React from "react";

  export interface MDEditorProps {
    value?: string;
    onChange?: (value: string | undefined) => void;
    preview?: "live" | "edit" | "full";
    height?: number;
    visibleDragBar?: boolean;
  }

  const MDEditor: React.FC<MDEditorProps>;
  export default MDEditor;
}

declare module "@uiw/react-md-editor/markdown-editor.css" {
  const content: string;
  export default content;
}

// 扩展 CSSProperties 类型
declare namespace CSS {
  interface Properties {
    paddingY?: string | number;
    paddingX?: string | number;
  }
}

// 百度地图全局类型
declare global {
  interface Window {
    BMap?: any;
    BMapGL?: any;
    initBMapGL?: () => void;
    init?: () => void;
    viewBuildingDetails?: (buildingId: string) => void;
    viewBuildingDetailsGL?: (buildingId: string) => void;
    BMAPGL_ANCHOR_TOP_LEFT?: number;
    BMAPGL_ANCHOR_TOP_RIGHT?: number;
    BMAPGL_ANCHOR_BOTTOM_LEFT?: number;
    BMAPGL_ANCHOR_BOTTOM_RIGHT?: number;
  }
}
