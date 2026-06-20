import type { WorkstationOps } from "@/types";

export interface Building {
  id: string;
  name: string;
  code: string;
  address?: string;
  totalFloors: number;
  workstationCount: number;
  status: number;
  floors?: Floor[];
}

export interface Floor {
  id: string;
  buildingId: string;
  floorNo: string;
  name: string;
  workstationCount: number;
  workstations?: WorkstationOps[];
}

export interface WorkstationWithPosition extends WorkstationOps {
  position?: {
    x: number;
    y: number;
  };
}

export type AnimationState = "stacked" | "expanding" | "expanded" | "flattening" | "flat";
export type ModalView = "floors" | "workstation" | "transition";
