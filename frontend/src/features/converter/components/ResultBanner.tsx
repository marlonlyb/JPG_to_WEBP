import { BATCH_ITEM_STATUS, BATCH_SCREEN_STATUS, type AppErrorDTO, type BatchConvertResultDTO, type BatchInspectionDTO, type BatchPreflightDTO, type BatchScreenStatus } from "../types";

interface ResultBannerProps {
  error: AppErrorDTO | null;
  inspection: BatchInspectionDTO | null;
  preflight: BatchPreflightDTO | null;
  result: BatchConvertResultDTO | null;
  status: BatchScreenStatus;
  statusMessage: string | null;
  onCancelOverwrite: () => void;
  onConfirmOverwrite: () => void;
}

export function ResultBanner({
  error,
  inspection,
  preflight,
  result,
  status,
  statusMessage,
  onCancelOverwrite,
  onConfirmOverwrite,
}: ResultBannerProps) {
  if (
    (status === BATCH_SCREEN_STATUS.SUCCESS ||
      status === BATCH_SCREEN_STATUS.PARTIAL ||
      status === BATCH_SCREEN_STATUS.FAILURE) &&
    result
  ) {
    return (
      <section
        className={`status-banner ${status === BATCH_SCREEN_STATUS.SUCCESS ? "status-success" : status === BATCH_SCREEN_STATUS.PARTIAL ? "status-warning" : "status-error"}`}
        role={status === BATCH_SCREEN_STATUS.FAILURE ? "alert" : "status"}
      >
        <h2>{getCompletionHeading(status)}</h2>
        <p>
          {result.summary.completedInputs} / {result.summary.totalInputs} files processed · {result.summary.writtenOutputs} / {result.summary.totalOutputs} outputs written
        </p>
        {result.summary.overwrittenOutputs > 0 ? (
          <p>{result.summary.overwrittenOutputs} output{result.summary.overwrittenOutputs === 1 ? " was" : "s were"} overwritten.</p>
        ) : null}

        <ul className="result-list">
          {result.items.map((item) => (
            <li key={item.input.inputPath} className="result-item">
              <strong>{item.input.fileName}</strong> · {item.status}
              {item.status !== BATCH_ITEM_STATUS.FAILED ? (
                <ul className="output-list compact-list">
                  {item.outputs.map((output) => (
                    <li key={output.outputPath}>
                      {output.outputPath}
                      {output.overwritten ? " · overwritten" : ""}
                    </li>
                  ))}
                </ul>
              ) : null}
              {item.error ? <p className="status-details">{item.error.message}</p> : null}
            </li>
          ))}
        </ul>
        {statusMessage ? <p className="status-details">{statusMessage}</p> : null}
      </section>
    );
  }

  if (status === BATCH_SCREEN_STATUS.PREFLIGHT && preflight) {
    return (
      <section className="status-banner status-warning" role="alert">
        <h2>Overwrite confirmation</h2>
        <p>
          {preflight.totalConflicts} output{preflight.totalConflicts === 1 ? "" : "s"} already exist.
        </p>
        <ul className="output-list compact-list">
          {preflight.conflicts.map((conflict) => (
            <li key={conflict}>{conflict}</li>
          ))}
        </ul>
        <div className="banner-actions">
          <button className="primary-button" type="button" onClick={onConfirmOverwrite}>
            Replace all
          </button>
          <button className="secondary-button" type="button" onClick={onCancelOverwrite}>
            Cancel
          </button>
        </div>
      </section>
    );
  }

  if (status === BATCH_SCREEN_STATUS.FAILURE && error) {
    return (
      <section className="status-banner status-error" role="alert">
        <h2>Error</h2>
        <p>{error.message}</p>
        {error.details ? <p className="status-details">{error.details}</p> : null}
      </section>
    );
  }

  if (status === BATCH_SCREEN_STATUS.CONVERTING) {
    return (
      <section className="status-banner status-info" role="status">
        <h2>Converting</h2>
        <p>
          0 / {inspection?.totalInputs ?? 0} files processed
        </p>
        <p className="status-details">The conversion stays local on this device.</p>
        {statusMessage ? <p className="status-details">{statusMessage}</p> : null}
      </section>
    );
  }

  if (status === BATCH_SCREEN_STATUS.REVIEW && statusMessage) {
    return (
      <section className="status-banner status-info" role="status">
        <p>{statusMessage}</p>
      </section>
    );
  }

  return null;
}

function getCompletionHeading(status: BatchScreenStatus): string {
  if (status === BATCH_SCREEN_STATUS.SUCCESS) {
    return "Batch completed";
  }

  if (status === BATCH_SCREEN_STATUS.PARTIAL) {
    return "Batch completed with issues";
  }

  return "Batch failed";
}
