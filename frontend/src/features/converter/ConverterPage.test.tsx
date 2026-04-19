import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ConverterPage } from "./ConverterPage";
import { APP_ERROR_CODE, BATCH_ITEM_STATUS, type AppErrorDTO, type BatchConvertResultDTO, type BatchInspectionDTO } from "./types";

vi.mock("../../lib/wails/converterClient", () => ({
  convertBatch: vi.fn(),
  inspectBatchInputs: vi.fn(),
  pickInputFiles: vi.fn(),
  preflightBatch: vi.fn(),
}));

import {
  convertBatch,
  inspectBatchInputs,
  pickInputFiles,
  preflightBatch,
} from "../../lib/wails/converterClient";

const FIXTURE_INSPECTION: BatchInspectionDTO = {
  items: [
    {
      input: {
        inputPath: "/images/photo.jpg",
        fileName: "photo.jpg",
        width: 1200,
        height: 800,
        inputBytes: 2048,
      },
      outputs: [
        { suffix: "_high", quality: 100, outputPath: "/images/photo_high.webp", exists: false },
        { suffix: "_medium", quality: 50, outputPath: "/images/photo_medium.webp", exists: false },
        { suffix: "_low", quality: 25, outputPath: "/images/photo_low.webp", exists: false },
      ],
    },
    {
      input: {
        inputPath: "/images/other.jpeg",
        fileName: "other.jpeg",
        width: 640,
        height: 480,
        inputBytes: 1024,
      },
      outputs: [
        { suffix: "_high", quality: 100, outputPath: "/images/other_high.webp", exists: false },
        { suffix: "_medium", quality: 50, outputPath: "/images/other_medium.webp", exists: false },
        { suffix: "_low", quality: 25, outputPath: "/images/other_low.webp", exists: false },
      ],
    },
  ],
  totalInputs: 2,
  totalPlannedOutputs: 6,
};

describe("ConverterPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(convertBatch).mockReset();
    vi.mocked(inspectBatchInputs).mockReset();
    vi.mocked(pickInputFiles).mockReset();
    vi.mocked(preflightBatch).mockReset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    cleanup();
  });

  it("renders the minimal batch picker on first render", () => {
    render(<ConverterPage />);

    expect(screen.getByText("Choose 1 to 10 local JPEG files. Outputs stay beside each source.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Convert batch" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Clear" })).toBeDisabled();
  });

  it("loads a valid batch review and shows planned outputs", async () => {
    mockSelectedBatch();

    render(<ConverterPage />);

    const convertButton = screen.getByRole("button", { name: "Convert batch" });
    expect(convertButton).toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));

    expect(await screen.findByText("photo.jpg")).toBeInTheDocument();
    expect(screen.getByText("other.jpeg")).toBeInTheDocument();
    expect(screen.getByText("2 files selected · 6 planned WebP exports")).toBeInTheDocument();
    expect(screen.getByText(/photo_high\.webp/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Change" })).toBeInTheDocument();
    expect(convertButton).toBeEnabled();
  });

  it("rejects invalid selections without activating a batch", async () => {
    vi.mocked(pickInputFiles).mockRejectedValue(createAppError({
      code: APP_ERROR_CODE.INVALID_INPUT,
      message: "Select between 1 and 10 JPEG files.",
      details: "unsupported file extension: /images/document.png",
    }));

    render(<ConverterPage />);

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Select between 1 and 10 JPEG files.",
    );
    expect(screen.getByText("unsupported file extension: /images/document.png")).toBeInTheDocument();
    expect(screen.queryByText("photo.jpg")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Convert batch" })).toBeDisabled();
  });

  it("shows overwrite confirmation and returns to review when canceled", async () => {
    mockSelectedBatch();
    vi.mocked(preflightBatch).mockResolvedValue({
      conflicts: ["/images/photo_high.webp", "/images/other_low.webp"],
      totalConflicts: 2,
      needsOverwrite: true,
    });

    render(<ConverterPage />);

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));
    await screen.findByText("photo.jpg");

    fireEvent.click(screen.getByRole("button", { name: "Convert batch" }));

    expect(await screen.findByText("Overwrite confirmation")).toBeInTheDocument();
    expect(screen.getByText("/images/photo_high.webp")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() => {
      expect(screen.getByText("Overwrite canceled.")).toBeInTheDocument();
    });
    expect(convertBatch).not.toHaveBeenCalled();
  });

  it("confirms overwrite and converts the inspected batch", async () => {
    mockSelectedBatch();
    vi.mocked(preflightBatch).mockResolvedValue({
      conflicts: ["/images/photo_high.webp"],
      totalConflicts: 1,
      needsOverwrite: true,
    });
    vi.mocked(convertBatch).mockResolvedValue(createBatchResult());

    render(<ConverterPage />);

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));
    await screen.findByText("photo.jpg");

    fireEvent.click(screen.getByRole("button", { name: "Convert batch" }));
    await screen.findByText("Overwrite confirmation");

    fireEvent.click(screen.getByRole("button", { name: "Replace all" }));

    await waitFor(() => {
      expect(convertBatch).toHaveBeenCalledWith({
        inputs: ["/images/photo.jpg", "/images/other.jpeg"],
        overwrite: true,
      });
    });
    expect(await screen.findByText("Batch completed")).toBeInTheDocument();
  });

  it("disables browse controls while conversion is running", async () => {
    mockSelectedBatch();
    vi.mocked(preflightBatch).mockResolvedValue({
      conflicts: [],
      totalConflicts: 0,
      needsOverwrite: false,
    });

    let resolveConversion: ((value: BatchConvertResultDTO) => void) | undefined;
    const conversionPromise = new Promise<BatchConvertResultDTO>((resolve) => {
      resolveConversion = resolve;
    });
    vi.mocked(convertBatch).mockReturnValue(conversionPromise);

    render(<ConverterPage />);

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));
    await screen.findByText("photo.jpg");

    fireEvent.click(screen.getByRole("button", { name: "Convert batch" }));

    expect(await screen.findByText("Converting")).toBeInTheDocument();
    expect(screen.getByText("0 / 2 files processed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Change" })).toBeDisabled();

    resolveConversion?.(createBatchResult());

    await waitFor(() => {
      expect(screen.getByText("Batch completed")).toBeInTheDocument();
    });
  });

  it("renders partial success without network access", async () => {
    mockSelectedBatch();
    vi.mocked(preflightBatch).mockResolvedValue({
      conflicts: [],
      totalConflicts: 0,
      needsOverwrite: false,
    });
    vi.mocked(convertBatch).mockResolvedValue(createBatchResult({ partial: true }));
    const fetchSpy = vi.fn(async () => {
      throw new Error("network unavailable");
    });
    vi.stubGlobal("fetch", fetchSpy);

    render(<ConverterPage />);

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));
    await screen.findByText("photo.jpg");

    fireEvent.click(screen.getByRole("button", { name: "Convert batch" }));

    expect(await screen.findByText("Batch completed with issues")).toBeInTheDocument();
    expect(screen.getByText("2 / 2 files processed · 3 / 6 outputs written")).toBeInTheDocument();
    expect(screen.getByText("The selected JPEG could not be read or decoded.")).toBeInTheDocument();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("renders failure messaging when conversion fails before results", async () => {
    mockSelectedBatch();
    vi.mocked(preflightBatch).mockResolvedValue({
      conflicts: [],
      totalConflicts: 0,
      needsOverwrite: false,
    });
    vi.mocked(convertBatch).mockRejectedValue(createAppError({
      code: APP_ERROR_CODE.READ_FAILED,
      message: "The batch could not be read or decoded.",
      details: "decode failed",
    }));

    render(<ConverterPage />);

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));
    await screen.findByText("photo.jpg");

    fireEvent.click(screen.getByRole("button", { name: "Convert batch" }));

    expect(await screen.findByText("Error")).toBeInTheDocument();
    expect(screen.getByText("The batch could not be read or decoded.")).toBeInTheDocument();
    expect(screen.getByText("decode failed")).toBeInTheDocument();
  });

  it("clears the completed batch back to the initial state", async () => {
    mockSelectedBatch();
    vi.mocked(preflightBatch).mockResolvedValue({
      conflicts: [],
      totalConflicts: 0,
      needsOverwrite: false,
    });
    vi.mocked(convertBatch).mockResolvedValue(createBatchResult());

    render(<ConverterPage />);

    fireEvent.click(screen.getByRole("button", { name: "Choose JPEGs" }));
    await screen.findByText("photo.jpg");

    fireEvent.click(screen.getByRole("button", { name: "Convert batch" }));

    expect(await screen.findByText("Batch completed")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Clear" }));

    await waitFor(() => {
      expect(screen.getByText("Choose 1 to 10 local JPEG files. Outputs stay beside each source.")).toBeInTheDocument();
    });
    expect(screen.queryByText("Batch completed")).not.toBeInTheDocument();
    expect(screen.queryByText("photo.jpg")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Convert batch" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Clear" })).toBeDisabled();
  });
});

function mockSelectedBatch() {
  vi.mocked(pickInputFiles).mockResolvedValue([
    "/images/photo.jpg",
    "/images/other.jpeg",
  ]);
  vi.mocked(inspectBatchInputs).mockResolvedValue(FIXTURE_INSPECTION);
}

function createAppError(error: AppErrorDTO): AppErrorDTO {
  return error;
}

function createBatchResult(options?: { partial?: boolean }): BatchConvertResultDTO {
  if (options?.partial) {
    return {
      items: [
        {
          input: FIXTURE_INSPECTION.items[0].input,
          outputs: [
            { outputPath: "/images/photo_high.webp", outputBytes: 200, quality: 100, overwritten: false },
            { outputPath: "/images/photo_medium.webp", outputBytes: 150, quality: 50, overwritten: false },
            { outputPath: "/images/photo_low.webp", outputBytes: 100, quality: 25, overwritten: false },
          ],
          status: BATCH_ITEM_STATUS.SUCCESS,
        },
        {
          input: FIXTURE_INSPECTION.items[1].input,
          outputs: [],
          status: BATCH_ITEM_STATUS.FAILED,
          error: {
            code: APP_ERROR_CODE.READ_FAILED,
            message: "The selected JPEG could not be read or decoded.",
            details: "decode failed",
          },
        },
      ],
      summary: {
        totalInputs: 2,
        completedInputs: 2,
        failedInputs: 1,
        totalOutputs: 6,
        writtenOutputs: 3,
        overwrittenOutputs: 0,
      },
    };
  }

  return {
    items: [
      {
        input: FIXTURE_INSPECTION.items[0].input,
        outputs: [
          { outputPath: "/images/photo_high.webp", outputBytes: 200, quality: 100, overwritten: false },
          { outputPath: "/images/photo_medium.webp", outputBytes: 150, quality: 50, overwritten: false },
          { outputPath: "/images/photo_low.webp", outputBytes: 100, quality: 25, overwritten: false },
        ],
        status: BATCH_ITEM_STATUS.SUCCESS,
      },
      {
        input: FIXTURE_INSPECTION.items[1].input,
        outputs: [
          { outputPath: "/images/other_high.webp", outputBytes: 200, quality: 100, overwritten: true },
          { outputPath: "/images/other_medium.webp", outputBytes: 150, quality: 50, overwritten: true },
          { outputPath: "/images/other_low.webp", outputBytes: 100, quality: 25, overwritten: true },
        ],
        status: BATCH_ITEM_STATUS.SUCCESS,
      },
    ],
    summary: {
      totalInputs: 2,
      completedInputs: 2,
      failedInputs: 0,
      totalOutputs: 6,
      writtenOutputs: 6,
      overwrittenOutputs: 3,
    },
  };
}
