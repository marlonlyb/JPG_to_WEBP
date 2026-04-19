import type { BatchInspectionDTO } from "../types";

interface FilePickerProps {
  inspection: BatchInspectionDTO | null;
  isBusy: boolean;
  onBrowse: () => void;
}

export function FilePicker({
  inspection,
  isBusy,
  onBrowse,
}: FilePickerProps) {
  return (
    <section className="card file-picker">
      <div className="section-header">
        <h2>JPEG batch</h2>
        <button className="secondary-button" disabled={isBusy} type="button" onClick={onBrowse}>
          {inspection ? "Change" : "Choose JPEGs"}
        </button>
      </div>

      {inspection ? (
        <>
          <p>
            {inspection.totalInputs} file{inspection.totalInputs === 1 ? "" : "s"} selected · {inspection.totalPlannedOutputs} planned WebP exports
          </p>

          <ul className="batch-file-list">
            {inspection.items.map((item) => (
              <li key={item.input.inputPath} className="batch-file-item">
                <dl className="metadata-grid">
                  <div>
                    <dt>Name</dt>
                    <dd>{item.input.fileName}</dd>
                  </div>
                  <div>
                    <dt>Size</dt>
                    <dd>{formatBytes(item.input.inputBytes)}</dd>
                  </div>
                  <div>
                    <dt>Px</dt>
                    <dd>
                      {item.input.width} × {item.input.height}px
                    </dd>
                  </div>
                </dl>

                <ul className="output-list">
                  {item.outputs.map((output) => (
                    <li key={output.outputPath}>
                      <strong>{output.suffix}</strong> · Q{output.quality} · {output.outputPath}
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        </>
      ) : (
        <p className="muted-copy">Choose 1 to 10 local JPEG files. Outputs stay beside each source.</p>
      )}
    </section>
  );
}

function formatBytes(value: number): string {
  if (value < 1024) {
    return `${value} B`;
  }

  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }

  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}
