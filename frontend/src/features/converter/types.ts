export const APP_ERROR_CODE = {
  INVALID_INPUT: "INVALID_INPUT",
  INVALID_QUALITY: "INVALID_QUALITY",
  READ_FAILED: "READ_FAILED",
  ENCODE_FAILED: "ENCODE_FAILED",
  WRITE_FAILED: "WRITE_FAILED",
} as const;

export type AppErrorCode = (typeof APP_ERROR_CODE)[keyof typeof APP_ERROR_CODE];

export interface ImageInfoDTO {
  inputPath: string;
  fileName: string;
  width: number;
  height: number;
  inputBytes: number;
}

export interface ConvertRequestDTO {
  inputPath: string;
  outputPath?: string;
  quality: number;
  overwrite: boolean;
}

export interface ConvertResultDTO {
  outputPath: string;
  outputBytes: number;
  quality: number;
  overwritten: boolean;
}

export const BATCH_ITEM_STATUS = {
  PENDING: "pending",
  SUCCESS: "success",
  FAILED: "failed",
  PARTIAL: "partial",
} as const;

export type BatchItemStatus = (typeof BATCH_ITEM_STATUS)[keyof typeof BATCH_ITEM_STATUS];

export const BATCH_SCREEN_STATUS = {
  IDLE: "idle",
  REVIEW: "review",
  PREFLIGHT: "preflight",
  CONVERTING: "converting",
  SUCCESS: "success",
  PARTIAL: "partial",
  FAILURE: "failure",
} as const;

export type BatchScreenStatus = (typeof BATCH_SCREEN_STATUS)[keyof typeof BATCH_SCREEN_STATUS];

export interface OutputVariantDTO {
  suffix: string;
  quality: number;
  outputPath: string;
  exists: boolean;
}

export interface BatchInspectItemDTO {
  input: ImageInfoDTO;
  outputs: OutputVariantDTO[];
}

export interface BatchInspectionDTO {
  items: BatchInspectItemDTO[];
  totalInputs: number;
  totalPlannedOutputs: number;
}

export interface BatchPreflightDTO {
  conflicts: string[];
  totalConflicts: number;
  needsOverwrite: boolean;
}

export interface BatchConvertRequestDTO {
  inputs: string[];
  overwrite: boolean;
}

export interface BatchItemResultDTO {
  input: ImageInfoDTO;
  outputs: ConvertResultDTO[];
  status: BatchItemStatus;
  error?: AppErrorDTO;
}

export interface BatchSummaryDTO {
  totalInputs: number;
  completedInputs: number;
  failedInputs: number;
  totalOutputs: number;
  writtenOutputs: number;
  overwrittenOutputs: number;
}

export interface BatchConvertResultDTO {
  items: BatchItemResultDTO[];
  summary: BatchSummaryDTO;
}

export interface AppErrorDTO {
  code: AppErrorCode;
  message: string;
  details?: string;
}
