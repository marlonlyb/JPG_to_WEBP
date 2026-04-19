import type {
  AppErrorCode,
  AppErrorDTO,
  BatchConvertRequestDTO,
  BatchConvertResultDTO,
  BatchInspectionDTO,
  BatchPreflightDTO,
  ConvertRequestDTO,
  ConvertResultDTO,
  ImageInfoDTO,
} from "../../features/converter/types";
import { APP_ERROR_CODE } from "../../features/converter/types";

interface ConverterBindings {
  PickInputFile(): Promise<string>;
  PickInputFiles(): Promise<string[]>;
  GetImageInfo(inputPath: string): Promise<ImageInfoDTO>;
  InspectBatchInputs(inputPaths: string[]): Promise<BatchInspectionDTO>;
  PickOutputPath(inputPath: string): Promise<string>;
  PreflightBatch(inputPaths: string[]): Promise<BatchPreflightDTO>;
  ConvertToWebP(request: ConvertRequestDTO): Promise<ConvertResultDTO>;
  ConvertBatch(request: BatchConvertRequestDTO): Promise<BatchConvertResultDTO>;
}

interface WailsNamespace {
  App?: ConverterBindings;
}

declare global {
  interface Window {
    go?: {
      app?: WailsNamespace;
    };
  }
}

export async function pickInputFile(): Promise<string> {
  return invoke((bindings) => bindings.PickInputFile());
}

export async function pickInputFiles(): Promise<string[]> {
  return invoke((bindings) => bindings.PickInputFiles());
}

export async function getImageInfo(inputPath: string): Promise<ImageInfoDTO> {
  return invoke((bindings) => bindings.GetImageInfo(inputPath));
}

export async function inspectBatchInputs(
  inputPaths: string[],
): Promise<BatchInspectionDTO> {
  return invoke((bindings) => bindings.InspectBatchInputs(inputPaths));
}

export async function pickOutputPath(inputPath: string): Promise<string> {
  return invoke((bindings) => bindings.PickOutputPath(inputPath));
}

export async function preflightBatch(
  inputPaths: string[],
): Promise<BatchPreflightDTO> {
  return invoke((bindings) => bindings.PreflightBatch(inputPaths));
}

export async function convertToWebP(
  request: ConvertRequestDTO,
): Promise<ConvertResultDTO> {
  return invoke((bindings) => bindings.ConvertToWebP(request));
}

export async function convertBatch(
  request: BatchConvertRequestDTO,
): Promise<BatchConvertResultDTO> {
  return invoke((bindings) => bindings.ConvertBatch(request));
}

async function invoke<T>(
  action: (bindings: ConverterBindings) => Promise<T>,
): Promise<T> {
  const bindings = window.go?.app?.App;
  if (!bindings) {
    throw createFallbackAppError(
      APP_ERROR_CODE.READ_FAILED,
      "Wails backend bindings are not available.",
    );
  }

  try {
    return await action(bindings);
  } catch (error: unknown) {
    throw toAppError(error);
  }
}

function toAppError(error: unknown): AppErrorDTO {
  if (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof error.message === "string"
  ) {
    const parsedError = parseAppError(error.message);
    if (parsedError) {
      return parsedError;
    }

    return createFallbackAppError(APP_ERROR_CODE.READ_FAILED, error.message);
  }

  if (typeof error === "string") {
    const parsedError = parseAppError(error);
    if (parsedError) {
      return parsedError;
    }

    return createFallbackAppError(APP_ERROR_CODE.READ_FAILED, error);
  }

  return createFallbackAppError(
    APP_ERROR_CODE.READ_FAILED,
    "Unexpected application error.",
  );
}

function parseAppError(value: string): AppErrorDTO | null {
  try {
    const parsedValue: unknown = JSON.parse(value);
    if (isAppErrorDTO(parsedValue)) {
      return parsedValue;
    }
  } catch {
    return null;
  }

  return null;
}

function isAppErrorDTO(value: unknown): value is AppErrorDTO {
  return (
    typeof value === "object" &&
    value !== null &&
    "code" in value &&
    typeof value.code === "string" &&
    isAppErrorCode(value.code) &&
    "message" in value &&
    typeof value.message === "string" &&
    (!("details" in value) || typeof value.details === "string")
  );
}

function isAppErrorCode(value: string): value is AppErrorCode {
  return Object.values(APP_ERROR_CODE).includes(value as AppErrorCode);
}

function createFallbackAppError(
  code: AppErrorCode,
  message: string,
  details?: string,
): AppErrorDTO {
  return {
    code,
    message,
    details,
  };
}
