import { beforeEach, describe, expect, it, vi } from "vitest";

import { APP_ERROR_CODE } from "../../features/converter/types";
import { convertBatch, inspectBatchInputs, pickInputFile, pickInputFiles } from "./converterClient";

describe("converterClient", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    delete window.go;
  });

  it("calls the Wails bindings from window.go.app.App", async () => {
    const pickInputFileBinding = vi.fn().mockResolvedValue("/images/photo.jpg");

    window.go = {
      app: {
        App: {
          PickInputFile: pickInputFileBinding,
          PickInputFiles: vi.fn(),
          GetImageInfo: vi.fn(),
          InspectBatchInputs: vi.fn(),
          PickOutputPath: vi.fn(),
          PreflightBatch: vi.fn(),
          ConvertToWebP: vi.fn(),
          ConvertBatch: vi.fn(),
        },
      },
    };

    await expect(pickInputFile()).resolves.toBe("/images/photo.jpg");
    expect(pickInputFileBinding).toHaveBeenCalledTimes(1);
  });

  it("returns a stable fallback error when bindings are unavailable", async () => {
    await expect(pickInputFile()).rejects.toMatchObject({
      code: APP_ERROR_CODE.READ_FAILED,
      message: "Wails backend bindings are not available.",
    });
  });

  it("calls the batch picker binding when requested", async () => {
    const pickInputFilesBinding = vi.fn().mockResolvedValue(["/images/photo.jpg", "/images/other.jpeg"]);

    window.go = {
      app: {
        App: {
          PickInputFile: vi.fn(),
          PickInputFiles: pickInputFilesBinding,
          GetImageInfo: vi.fn(),
          InspectBatchInputs: vi.fn(),
          PickOutputPath: vi.fn(),
          PreflightBatch: vi.fn(),
          ConvertToWebP: vi.fn(),
          ConvertBatch: vi.fn(),
        },
      },
    };

    await expect(pickInputFiles()).resolves.toEqual(["/images/photo.jpg", "/images/other.jpeg"]);
    expect(pickInputFilesBinding).toHaveBeenCalledTimes(1);
  });

  it("returns parsed app errors from batch inspection bindings", async () => {
    const inspectBinding = vi.fn().mockRejectedValue(
      JSON.stringify({
        code: APP_ERROR_CODE.INVALID_INPUT,
        message: "Select between 1 and 10 JPEG files.",
        details: "unsupported file extension",
      }),
    );

    window.go = {
      app: {
        App: {
          PickInputFile: vi.fn(),
          PickInputFiles: vi.fn(),
          GetImageInfo: vi.fn(),
          InspectBatchInputs: inspectBinding,
          PickOutputPath: vi.fn(),
          PreflightBatch: vi.fn(),
          ConvertToWebP: vi.fn(),
          ConvertBatch: vi.fn(),
        },
      },
    };

    await expect(inspectBatchInputs(["/images/photo.jpg"])).rejects.toMatchObject({
      code: APP_ERROR_CODE.INVALID_INPUT,
      message: "Select between 1 and 10 JPEG files.",
      details: "unsupported file extension",
    });
  });

  it("calls the batch convert binding with overwrite state", async () => {
    const convertBatchBinding = vi.fn().mockResolvedValue({
      items: [],
      summary: {
        totalInputs: 2,
        completedInputs: 2,
        failedInputs: 0,
        totalOutputs: 6,
        writtenOutputs: 6,
        overwrittenOutputs: 3,
      },
    });

    window.go = {
      app: {
        App: {
          PickInputFile: vi.fn(),
          PickInputFiles: vi.fn(),
          GetImageInfo: vi.fn(),
          InspectBatchInputs: vi.fn(),
          PickOutputPath: vi.fn(),
          PreflightBatch: vi.fn(),
          ConvertToWebP: vi.fn(),
          ConvertBatch: convertBatchBinding,
        },
      },
    };

    await expect(convertBatch({ inputs: ["/images/photo.jpg", "/images/other.jpeg"], overwrite: true })).resolves.toMatchObject({
      summary: {
        overwrittenOutputs: 3,
      },
    });
    expect(convertBatchBinding).toHaveBeenCalledWith({
      inputs: ["/images/photo.jpg", "/images/other.jpeg"],
      overwrite: true,
    });
  });
});
