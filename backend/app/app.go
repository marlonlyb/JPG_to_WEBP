package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"jpg-to-webp/backend/internal/conversion"
	"jpg-to-webp/backend/internal/filesystem"
	"jpg-to-webp/backend/internal/settings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AppErrorCode string

const (
	AppErrorCodeInvalidInput   AppErrorCode = "INVALID_INPUT"
	AppErrorCodeInvalidQuality AppErrorCode = "INVALID_QUALITY"
	AppErrorCodeReadFailed     AppErrorCode = "READ_FAILED"
	AppErrorCodeEncodeFailed   AppErrorCode = "ENCODE_FAILED"
	AppErrorCodeWriteFailed    AppErrorCode = "WRITE_FAILED"
)

type ImageInfoDTO struct {
	InputPath  string `json:"inputPath"`
	FileName   string `json:"fileName"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	InputBytes int64  `json:"inputBytes"`
}

type ConvertRequestDTO struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath,omitempty"`
	Quality    int    `json:"quality"`
	Overwrite  bool   `json:"overwrite"`
}

type ConvertResultDTO struct {
	OutputPath  string `json:"outputPath"`
	OutputBytes int64  `json:"outputBytes"`
	Quality     int    `json:"quality"`
	Overwritten bool   `json:"overwritten"`
}

type OutputVariantDTO struct {
	Suffix     string `json:"suffix"`
	Quality    int    `json:"quality"`
	OutputPath string `json:"outputPath"`
	Exists     bool   `json:"exists"`
}

type BatchInspectItemDTO struct {
	Input   ImageInfoDTO       `json:"input"`
	Outputs []OutputVariantDTO `json:"outputs"`
}

type BatchInspectionDTO struct {
	Items               []BatchInspectItemDTO `json:"items"`
	TotalInputs         int                   `json:"totalInputs"`
	TotalPlannedOutputs int                   `json:"totalPlannedOutputs"`
}

type BatchPreflightDTO struct {
	Conflicts      []string `json:"conflicts"`
	TotalConflicts int      `json:"totalConflicts"`
	NeedsOverwrite bool     `json:"needsOverwrite"`
}

type BatchConvertRequestDTO struct {
	Inputs    []string `json:"inputs"`
	Overwrite bool     `json:"overwrite"`
}

type BatchItemResultDTO struct {
	Input   ImageInfoDTO       `json:"input"`
	Outputs []ConvertResultDTO `json:"outputs"`
	Status  string             `json:"status"`
	Error   *AppErrorDTO       `json:"error,omitempty"`
}

type BatchSummaryDTO struct {
	TotalInputs        int `json:"totalInputs"`
	CompletedInputs    int `json:"completedInputs"`
	FailedInputs       int `json:"failedInputs"`
	TotalOutputs       int `json:"totalOutputs"`
	WrittenOutputs     int `json:"writtenOutputs"`
	OverwrittenOutputs int `json:"overwrittenOutputs"`
}

type BatchConvertResultDTO struct {
	Items   []BatchItemResultDTO `json:"items"`
	Summary BatchSummaryDTO      `json:"summary"`
}

type AppErrorDTO struct {
	Code    AppErrorCode `json:"code"`
	Message string       `json:"message"`
	Details string       `json:"details,omitempty"`
}

type FileDialogs interface {
	OpenJPEGFile(ctx context.Context, defaultDirectory string) (string, error)
	OpenJPEGFiles(ctx context.Context, defaultDirectory string) ([]string, error)
	SaveWebPFile(ctx context.Context, defaultPath string) (string, error)
}

type App struct {
	ctx                 context.Context
	dialogs             FileDialogs
	converter           *conversion.Service
	settingsStore       settings.Store
	preferredInputRoots []string
	homeDirectory       string
}

func New() *App {
	return &App{
		dialogs:             RuntimeDialogs{},
		converter:           conversion.NewService(conversion.NewWebPEncoder()),
		settingsStore:       settings.NewDefaultStore(),
		preferredInputRoots: []string{"/mnt/h", "/mnt"},
		homeDirectory:       os.Getenv("HOME"),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) PickInputFile() (string, error) {
	path, err := a.dialogs.OpenJPEGFile(a.ctx, a.defaultInputDirectory())
	if err != nil {
		return "", encodeAppError(AppErrorDTO{
			Code:    AppErrorCodeReadFailed,
			Message: "Could not open the JPEG picker.",
			Details: err.Error(),
		})
	}
	if path == "" {
		return "", nil
	}

	normalizedPath, err := filesystem.ValidateJPEGInputPath(path)
	if err != nil {
		return "", mapAppError(err)
	}

	a.persistLastInputDirectory(normalizedPath)

	return normalizedPath, nil
}

func (a *App) PickInputFiles() ([]string, error) {
	defaultDirectory := a.defaultInputDirectory()
	paths, err := a.dialogs.OpenJPEGFiles(a.ctx, defaultDirectory)
	if err != nil {
		paths, err = a.pickInputFilesFallback(defaultDirectory)
		if err != nil {
			return nil, encodeAppError(AppErrorDTO{
				Code:    AppErrorCodeReadFailed,
				Message: "Could not open the JPEG picker.",
				Details: err.Error(),
			})
		}
	}
	if len(paths) == 0 {
		return []string{}, nil
	}

	normalizedPaths, err := filesystem.NormalizeBatchInputPaths(paths)
	if err != nil {
		return nil, mapAppError(err)
	}

	a.persistLastInputDirectory(normalizedPaths[len(normalizedPaths)-1])

	return normalizedPaths, nil
}

func (a *App) GetImageInfo(inputPath string) (ImageInfoDTO, error) {
	imageInfo, err := a.converter.InspectJPEG(inputPath)
	if err != nil {
		return ImageInfoDTO{}, mapAppError(err)
	}

	a.persistLastInputDirectory(imageInfo.InputPath)

	return ImageInfoDTO{
		InputPath:  imageInfo.InputPath,
		FileName:   imageInfo.FileName,
		Width:      imageInfo.Width,
		Height:     imageInfo.Height,
		InputBytes: imageInfo.InputBytes,
	}, nil
}

func (a *App) InspectBatchInputs(inputPaths []string) (BatchInspectionDTO, error) {
	items, err := a.converter.InspectBatch(inputPaths)
	if err != nil {
		return BatchInspectionDTO{}, mapAppError(err)
	}

	dtoItems := make([]BatchInspectItemDTO, 0, len(items))
	for _, item := range items {
		dtoItems = append(dtoItems, BatchInspectItemDTO{
			Input:   mapImageInfoDTO(item.Image),
			Outputs: mapOutputVariantDTOs(item.Outputs),
		})
	}

	if len(items) > 0 {
		a.persistLastInputDirectory(items[len(items)-1].Image.InputPath)
	}

	return BatchInspectionDTO{
		Items:               dtoItems,
		TotalInputs:         len(dtoItems),
		TotalPlannedOutputs: len(dtoItems) * 3,
	}, nil
}

func (a *App) PreflightBatch(inputPaths []string) (BatchPreflightDTO, error) {
	conflicts, err := filesystem.BatchOverwriteConflicts(inputPaths)
	if err != nil {
		return BatchPreflightDTO{}, mapAppError(err)
	}

	return BatchPreflightDTO{
		Conflicts:      conflicts,
		TotalConflicts: len(conflicts),
		NeedsOverwrite: len(conflicts) > 0,
	}, nil
}

func (a *App) PickOutputPath(inputPath string) (string, error) {
	normalizedInputPath, err := filesystem.ValidateJPEGInputPath(inputPath)
	if err != nil {
		return "", mapAppError(err)
	}

	defaultPath, _, err := filesystem.SuggestOutputPath(normalizedInputPath)
	if err != nil {
		return "", mapAppError(err)
	}

	selectedPath, err := a.dialogs.SaveWebPFile(a.ctx, defaultPath)
	if err != nil {
		return "", encodeAppError(AppErrorDTO{
			Code:    AppErrorCodeWriteFailed,
			Message: "Could not open the save dialog.",
			Details: err.Error(),
		})
	}
	if selectedPath == "" {
		return "", nil
	}

	normalizedOutputPath, err := filesystem.ValidateOutputPath(normalizedInputPath, selectedPath, false)
	if err != nil {
		return "", mapAppError(err)
	}

	return normalizedOutputPath, nil
}

func (a *App) ConvertToWebP(request ConvertRequestDTO) (ConvertResultDTO, error) {
	result, err := a.converter.Convert(conversion.ConvertRequest{
		InputPath:  request.InputPath,
		OutputPath: request.OutputPath,
		Quality:    request.Quality,
		Overwrite:  request.Overwrite,
	})
	if err != nil {
		return ConvertResultDTO{}, mapAppError(err)
	}

	return ConvertResultDTO{
		OutputPath:  result.OutputPath,
		OutputBytes: result.OutputBytes,
		Quality:     result.Quality,
		Overwritten: result.Overwritten,
	}, nil
}

func (a *App) ConvertBatch(request BatchConvertRequestDTO) (BatchConvertResultDTO, error) {
	result, err := a.converter.ConvertBatch(conversion.BatchConvertRequest{
		Inputs:    request.Inputs,
		Overwrite: request.Overwrite,
	})
	if err != nil {
		return BatchConvertResultDTO{}, mapAppError(err)
	}

	itemResults := make([]BatchItemResultDTO, 0, len(result.Items))
	for _, item := range result.Items {
		itemResult := BatchItemResultDTO{
			Input:   mapImageInfoDTO(item.Input),
			Outputs: mapConvertResultDTOs(item.Outputs),
			Status:  string(item.Status),
		}
		if item.Error != nil {
			appError := toAppErrorDTO(item.Error)
			itemResult.Error = &appError
		}

		itemResults = append(itemResults, itemResult)
	}

	return BatchConvertResultDTO{
		Items: itemResults,
		Summary: BatchSummaryDTO{
			TotalInputs:        result.Summary.TotalInputs,
			CompletedInputs:    result.Summary.CompletedInputs,
			FailedInputs:       result.Summary.FailedInputs,
			TotalOutputs:       result.Summary.TotalOutputs,
			WrittenOutputs:     result.Summary.WrittenOutputs,
			OverwrittenOutputs: result.Summary.OverwrittenOutputs,
		},
	}, nil
}

type RuntimeDialogs struct{}

func (RuntimeDialogs) OpenJPEGFile(ctx context.Context, defaultDirectory string) (string, error) {
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
		Title:            "Select a JPEG file",
		DefaultDirectory: defaultDirectory,
		Filters: []runtime.FileFilter{{
			DisplayName: "JPEG Images",
			Pattern:     "*.jpg;*.jpeg;*.JPG;*.JPEG",
		}},
	})
}

func (RuntimeDialogs) OpenJPEGFiles(ctx context.Context, defaultDirectory string) ([]string, error) {
	return runtime.OpenMultipleFilesDialog(ctx, runtime.OpenDialogOptions{
		Title:            "Select JPEG files",
		DefaultDirectory: defaultDirectory,
		Filters: []runtime.FileFilter{{
			DisplayName: "JPEG Images",
			Pattern:     "*.jpg;*.jpeg;*.JPG;*.JPEG",
		}},
	})
}

func (a *App) defaultInputDirectory() string {
	storedSettings, err := a.loadSettings()
	if err != nil {
		storedSettings = settings.AppSettings{}
	}

	candidates := make([]string, 0, 1+len(a.preferredInputRoots)+2)
	if storedSettings.LastInputDirectory != "" {
		candidates = append(candidates, storedSettings.LastInputDirectory)
	}
	candidates = append(candidates, a.preferredInputRoots...)
	candidates = append(candidates, a.homeDirectory, ".")

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}

	return "."
}

func (a *App) loadSettings() (settings.AppSettings, error) {
	if a.settingsStore == nil {
		return settings.AppSettings{}, nil
	}

	return a.settingsStore.Load()
}

func (a *App) persistLastInputDirectory(inputPath string) {
	if a.settingsStore == nil {
		return
	}

	_ = a.settingsStore.Save(settings.AppSettings{LastInputDirectory: filepath.Dir(inputPath)})
}

func (RuntimeDialogs) SaveWebPFile(ctx context.Context, defaultPath string) (string, error) {
	return runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:                "Save converted WebP",
		DefaultFilename:      filepath.Base(defaultPath),
		DefaultDirectory:     filepath.Dir(defaultPath),
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{{
			DisplayName: "WebP Images",
			Pattern:     "*.webp;*.WEBP",
		}},
	})
}

func (a *App) pickInputFilesFallback(defaultDirectory string) ([]string, error) {
	paths := make([]string, 0, filesystem.MaxBatchInputs)
	currentDirectory := defaultDirectory
	for len(paths) < filesystem.MaxBatchInputs {
		path, err := a.dialogs.OpenJPEGFile(a.ctx, currentDirectory)
		if err != nil {
			return nil, err
		}
		if path == "" {
			break
		}

		paths = append(paths, path)
		if normalizedPath, err := filesystem.ValidateJPEGInputPath(path); err == nil {
			currentDirectory = filepath.Dir(normalizedPath)
		}
	}

	return paths, nil
}

func mapImageInfoDTO(imageInfo conversion.ImageInfo) ImageInfoDTO {
	return ImageInfoDTO{
		InputPath:  imageInfo.InputPath,
		FileName:   imageInfo.FileName,
		Width:      imageInfo.Width,
		Height:     imageInfo.Height,
		InputBytes: imageInfo.InputBytes,
	}
}

func mapOutputVariantDTOs(outputs []filesystem.OutputVariantPlan) []OutputVariantDTO {
	result := make([]OutputVariantDTO, 0, len(outputs))
	for _, output := range outputs {
		result = append(result, OutputVariantDTO{
			Suffix:     output.Suffix,
			Quality:    output.Quality,
			OutputPath: output.OutputPath,
			Exists:     output.Exists,
		})
	}

	return result
}

func mapConvertResultDTOs(results []conversion.ConvertResult) []ConvertResultDTO {
	dtoResults := make([]ConvertResultDTO, 0, len(results))
	for _, result := range results {
		dtoResults = append(dtoResults, ConvertResultDTO{
			OutputPath:  result.OutputPath,
			OutputBytes: result.OutputBytes,
			Quality:     result.Quality,
			Overwritten: result.Overwritten,
		})
	}

	return dtoResults
}

func mapAppError(err error) error {
	return encodeAppError(toAppErrorDTO(err))
}

func toAppErrorDTO(err error) AppErrorDTO {
	switch {
	case errors.Is(err, filesystem.ErrEmptyPath),
		errors.Is(err, filesystem.ErrBatchInputCount),
		errors.Is(err, filesystem.ErrUnsupportedPathSyntax),
		errors.Is(err, filesystem.ErrUnsupportedExtension),
		errors.Is(err, filesystem.ErrMissingFile),
		errors.Is(err, filesystem.ErrNotRegularFile),
		errors.Is(err, filesystem.ErrInvalidOutputExtension),
		errors.Is(err, filesystem.ErrSamePath):
		return AppErrorDTO{
			Code:    AppErrorCodeInvalidInput,
			Message: "Select valid local JPEG input files and valid .webp targets.",
			Details: err.Error(),
		}
	case errors.Is(err, conversion.ErrInvalidQuality):
		return AppErrorDTO{
			Code:    AppErrorCodeInvalidQuality,
			Message: "Quality must be an integer between 0 and 100.",
			Details: err.Error(),
		}
	case errors.Is(err, conversion.ErrReadFailed),
		errors.Is(err, conversion.ErrDecodeFailed):
		return AppErrorDTO{
			Code:    AppErrorCodeReadFailed,
			Message: "The selected JPEG could not be read or decoded.",
			Details: err.Error(),
		}
	case errors.Is(err, conversion.ErrEncodeFailed):
		return AppErrorDTO{
			Code:    AppErrorCodeEncodeFailed,
			Message: "The WebP encoder could not process the selected image.",
			Details: err.Error(),
		}
	case errors.Is(err, filesystem.ErrOutputExists),
		errors.Is(err, filesystem.ErrMissingParentDirectory),
		errors.Is(err, conversion.ErrWriteFailed):
		return AppErrorDTO{
			Code:    AppErrorCodeWriteFailed,
			Message: "The output file could not be written without confirmation or a valid target path.",
			Details: err.Error(),
		}
	default:
		return AppErrorDTO{
			Code:    AppErrorCodeReadFailed,
			Message: "The application encountered an unexpected error.",
			Details: err.Error(),
		}
	}
}

func encodeAppError(appError AppErrorDTO) error {
	payload, err := json.Marshal(appError)
	if err != nil {
		return fmt.Errorf("marshal app error: %w", err)
	}

	return errors.New(string(payload))
}
