import { useState } from "react";

import {
  convertBatch,
  inspectBatchInputs,
  pickInputFiles,
  preflightBatch,
} from "../../lib/wails/converterClient";
import { FilePicker } from "./components/FilePicker";
import { ResultBanner } from "./components/ResultBanner";
import {
  APP_ERROR_CODE,
  type AppErrorDTO,
  type AppErrorCode,
  BATCH_SCREEN_STATUS,
  type BatchConvertResultDTO,
  type BatchInspectionDTO,
  type BatchPreflightDTO,
  type BatchScreenStatus,
} from "./types";

export function ConverterPage() {
  const [inspection, setInspection] = useState<BatchInspectionDTO | null>(null);
  const [status, setStatus] = useState<BatchScreenStatus>(BATCH_SCREEN_STATUS.IDLE);
  const [error, setError] = useState<AppErrorDTO | null>(null);
  const [preflight, setPreflight] = useState<BatchPreflightDTO | null>(null);
  const [result, setResult] = useState<BatchConvertResultDTO | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  const isBusy = status === BATCH_SCREEN_STATUS.CONVERTING;
  const canConvert = inspection !== null && !isBusy;
  const canReset = !isBusy && hasActiveBatchState({
    error,
    inspection,
    preflight,
    result,
    status,
    statusMessage,
  });

  async function handleBrowse() {
    if (isBusy) {
      return;
    }

    resetFeedback();

    try {
      const selectedPaths = await pickInputFiles();
      if (selectedPaths.length === 0) {
        return;
      }

      const nextInspection = await inspectBatchInputs(selectedPaths);
      setInspection(nextInspection);
      setStatus(BATCH_SCREEN_STATUS.REVIEW);
    } catch (nextError: unknown) {
      setInspection(null);
      setStatus(BATCH_SCREEN_STATUS.FAILURE);
      setError(toAppError(nextError));
    }
  }

  async function handleConvert() {
    if (!inspection || isBusy) {
      return;
    }

    resetFeedback();

    try {
      const nextPreflight = await preflightBatch(getActiveInputs(inspection));
      if (nextPreflight.needsOverwrite) {
        setPreflight(nextPreflight);
        setStatus(BATCH_SCREEN_STATUS.PREFLIGHT);
        return;
      }

      await runBatchConversion(inspection, false);
    } catch (nextError: unknown) {
      setStatus(BATCH_SCREEN_STATUS.FAILURE);
      setError(toAppError(nextError));
    }
  }

  async function handleConfirmOverwrite() {
    if (!inspection || !preflight) {
      return;
    }

    try {
      await runBatchConversion(inspection, true);
    } catch (nextError: unknown) {
      setStatus(BATCH_SCREEN_STATUS.FAILURE);
      setError(toAppError(nextError));
    }
  }

  function handleCancelOverwrite() {
    setPreflight(null);
    setStatus(BATCH_SCREEN_STATUS.REVIEW);
    setStatusMessage("Overwrite canceled.");
  }

  function handleReset() {
    if (!canReset) {
      return;
    }

    setInspection(null);
    setStatus(BATCH_SCREEN_STATUS.IDLE);
    resetFeedback();
  }

  function resetFeedback() {
    setError(null);
    setPreflight(null);
    setResult(null);
    setStatusMessage(null);
  }

  async function runBatchConversion(
    activeInspection: BatchInspectionDTO,
    overwrite: boolean,
  ) {
    resetFeedback();
    setStatus(BATCH_SCREEN_STATUS.CONVERTING);

    const nextResult = await convertBatch({
      inputs: getActiveInputs(activeInspection),
      overwrite,
    });

    setResult(nextResult);
    setStatus(getCompletionStatus(nextResult));
  }

  return (
    <section className="panel converter-panel">
      <header className="panel-header">
        <h1>JPG to WEBP</h1>
        <button className="secondary-button panel-header-action" disabled={!canReset} type="button" onClick={handleReset}>
          Clear
        </button>
      </header>

      <div className="converter-layout">
        <FilePicker
          inspection={inspection}
          isBusy={isBusy}
          onBrowse={handleBrowse}
        />
      </div>

      <div className="action-row">
        <button className="primary-button" disabled={!canConvert} type="button" onClick={handleConvert}>
          {isBusy ? "Converting…" : "Convert batch"}
        </button>
      </div>

      <ResultBanner
        error={error}
        inspection={inspection}
        preflight={preflight}
        result={result}
        status={status}
        statusMessage={statusMessage}
        onCancelOverwrite={handleCancelOverwrite}
        onConfirmOverwrite={handleConfirmOverwrite}
      />
    </section>
  );
}

interface ActiveBatchState {
  error: AppErrorDTO | null;
  inspection: BatchInspectionDTO | null;
  preflight: BatchPreflightDTO | null;
  result: BatchConvertResultDTO | null;
  status: BatchScreenStatus;
  statusMessage: string | null;
}

function hasActiveBatchState({
  error,
  inspection,
  preflight,
  result,
  status,
  statusMessage,
}: ActiveBatchState): boolean {
  return (
    inspection !== null ||
    preflight !== null ||
    result !== null ||
    error !== null ||
    statusMessage !== null ||
    status !== BATCH_SCREEN_STATUS.IDLE
  );
}

function getActiveInputs(inspection: BatchInspectionDTO): string[] {
  return inspection.items.map((item) => item.input.inputPath);
}

function getCompletionStatus(result: BatchConvertResultDTO): BatchScreenStatus {
  if (result.summary.failedInputs === 0) {
    return BATCH_SCREEN_STATUS.SUCCESS;
  }

  if (result.summary.writtenOutputs === 0) {
    return BATCH_SCREEN_STATUS.FAILURE;
  }

  return BATCH_SCREEN_STATUS.PARTIAL;
}

function toAppError(error: unknown): AppErrorDTO {
  if (
    typeof error === "object" &&
    error !== null &&
    "code" in error &&
    typeof error.code === "string" &&
    "message" in error &&
    typeof error.message === "string"
  ) {
    return {
      code: isAppErrorCode(error.code) ? error.code : APP_ERROR_CODE.READ_FAILED,
      message: error.message,
      details: "details" in error && typeof error.details === "string" ? error.details : undefined,
    };
  }

  return {
    code: APP_ERROR_CODE.READ_FAILED,
    message: "Unexpected application error.",
  };
}

function isAppErrorCode(value: string): value is AppErrorCode {
  return Object.values(APP_ERROR_CODE).includes(value as AppErrorCode);
}
