package conversion

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"jpg-to-webp/backend/internal/filesystem"
)

var (
	ErrInvalidQuality = errors.New("quality must be between 0 and 100")
	ErrReadFailed     = errors.New("failed to read source image")
	ErrDecodeFailed   = errors.New("failed to decode jpeg image")
	ErrEncodeFailed   = errors.New("failed to encode webp image")
	ErrWriteFailed    = errors.New("failed to write webp image")
)

type ImageInfo struct {
	InputPath  string
	FileName   string
	Width      int
	Height     int
	InputBytes int64
}

type ConvertRequest struct {
	InputPath  string
	OutputPath string
	Quality    int
	Overwrite  bool
}

type ConvertResult struct {
	OutputPath  string
	OutputBytes int64
	Quality     int
	Overwritten bool
}

type BatchInspectItem struct {
	Image   ImageInfo
	Outputs []filesystem.OutputVariantPlan
}

type BatchConvertRequest struct {
	Inputs    []string
	Overwrite bool
}

type BatchItemStatus string

const (
	BatchItemStatusPending BatchItemStatus = "pending"
	BatchItemStatusSuccess BatchItemStatus = "success"
	BatchItemStatusFailed  BatchItemStatus = "failed"
	BatchItemStatusPartial BatchItemStatus = "partial"
)

type BatchItemResult struct {
	Input   ImageInfo
	Outputs []ConvertResult
	Status  BatchItemStatus
	Error   error
}

type BatchSummary struct {
	TotalInputs        int
	CompletedInputs    int
	FailedInputs       int
	TotalOutputs       int
	WrittenOutputs     int
	OverwrittenOutputs int
}

type BatchConvertResult struct {
	Items   []BatchItemResult
	Summary BatchSummary
}

type Service struct {
	encoder Encoder
}

func NewService(encoder Encoder) *Service {
	return &Service{encoder: encoder}
}

func (s *Service) InspectJPEG(inputPath string) (ImageInfo, error) {
	normalizedInputPath, err := filesystem.ValidateJPEGInputPath(inputPath)
	if err != nil {
		return ImageInfo{}, err
	}

	metadata, err := readJPEGMetadata(normalizedInputPath)
	if err != nil {
		return ImageInfo{}, err
	}

	normalizedWidth, normalizedHeight := metadata.normalizedDimensions()

	return ImageInfo{
		InputPath:  normalizedInputPath,
		FileName:   filepath.Base(normalizedInputPath),
		Width:      normalizedWidth,
		Height:     normalizedHeight,
		InputBytes: metadata.InputBytes,
	}, nil
}

func (s *Service) Convert(request ConvertRequest) (ConvertResult, error) {
	if err := validateQuality(request.Quality); err != nil {
		return ConvertResult{}, err
	}

	normalizedInputPath, err := filesystem.ValidateJPEGInputPath(request.InputPath)
	if err != nil {
		return ConvertResult{}, err
	}

	outputPath := strings.TrimSpace(request.OutputPath)
	if outputPath == "" {
		outputPath, _, err = filesystem.SuggestOutputPath(normalizedInputPath)
	} else {
		outputPath, err = filesystem.ValidateOutputPath(normalizedInputPath, outputPath, request.Overwrite)
	}
	if err != nil {
		return ConvertResult{}, err
	}

	sourceImage, err := decodeNormalizedJPEG(normalizedInputPath)
	if err != nil {
		return ConvertResult{}, err
	}

	return s.writeWebPOutput(normalizedInputPath, sourceImage, outputPath, request.Quality, request.Overwrite)
}

func (s *Service) InspectBatch(inputPaths []string) ([]BatchInspectItem, error) {
	plans, err := filesystem.PlanBatchOutputs(inputPaths)
	if err != nil {
		return nil, err
	}

	items := make([]BatchInspectItem, 0, len(plans))
	for _, plan := range plans {
		imageInfo, err := s.InspectJPEG(plan.InputPath)
		if err != nil {
			return nil, err
		}

		items = append(items, BatchInspectItem{
			Image:   imageInfo,
			Outputs: plan.Outputs,
		})
	}

	return items, nil
}

func (s *Service) ConvertBatch(request BatchConvertRequest) (BatchConvertResult, error) {
	normalizedInputs, err := filesystem.NormalizeBatchInputPaths(request.Inputs)
	if err != nil {
		return BatchConvertResult{}, err
	}

	if !request.Overwrite {
		conflicts, err := filesystem.BatchOverwriteConflicts(normalizedInputs)
		if err != nil {
			return BatchConvertResult{}, err
		}
		if len(conflicts) > 0 {
			return BatchConvertResult{}, fmt.Errorf("%w: %s", filesystem.ErrOutputExists, conflicts[0])
		}
	}

	result := BatchConvertResult{
		Items: make([]BatchItemResult, 0, len(normalizedInputs)),
		Summary: BatchSummary{
			TotalInputs:  len(normalizedInputs),
			TotalOutputs: len(normalizedInputs) * 3,
		},
	}

	for _, inputPath := range normalizedInputs {
		itemResult := BatchItemResult{Status: BatchItemStatusPending}

		imageInfo, inspectErr := s.InspectJPEG(inputPath)
		if inspectErr != nil {
			itemResult.Status = BatchItemStatusFailed
			itemResult.Error = inspectErr
			result.Summary.CompletedInputs++
			result.Summary.FailedInputs++
			result.Items = append(result.Items, itemResult)
			continue
		}
		itemResult.Input = imageInfo

		plannedOutputs, planErr := filesystem.PlanOutputVariants(inputPath)
		if planErr != nil {
			itemResult.Status = BatchItemStatusFailed
			itemResult.Error = planErr
			result.Summary.CompletedInputs++
			result.Summary.FailedInputs++
			result.Items = append(result.Items, itemResult)
			continue
		}

		sourceImage, decodeErr := decodeNormalizedJPEG(inputPath)
		if decodeErr != nil {
			itemResult.Status = BatchItemStatusFailed
			itemResult.Error = decodeErr
			result.Summary.CompletedInputs++
			result.Summary.FailedInputs++
			result.Items = append(result.Items, itemResult)
			continue
		}

		for _, plannedOutput := range plannedOutputs {
			convertResult, convertErr := s.writeWebPOutput(inputPath, sourceImage, plannedOutput.OutputPath, plannedOutput.Quality, request.Overwrite)
			if convertErr != nil {
				itemResult.Error = convertErr
				if len(itemResult.Outputs) == 0 {
					itemResult.Status = BatchItemStatusFailed
				} else {
					itemResult.Status = BatchItemStatusPartial
				}
				break
			}

			itemResult.Outputs = append(itemResult.Outputs, convertResult)
			result.Summary.WrittenOutputs++
			if convertResult.Overwritten {
				result.Summary.OverwrittenOutputs++
			}
		}

		if itemResult.Status == BatchItemStatusPending {
			itemResult.Status = BatchItemStatusSuccess
		}
		if itemResult.Status == BatchItemStatusFailed || itemResult.Status == BatchItemStatusPartial {
			result.Summary.FailedInputs++
		}

		result.Summary.CompletedInputs++
		result.Items = append(result.Items, itemResult)
	}

	return result, nil
}
func (s *Service) writeWebPOutput(inputPath string, sourceImage image.Image, outputPath string, quality int, overwrite bool) (ConvertResult, error) {
	if err := validateQuality(quality); err != nil {
		return ConvertResult{}, err
	}

	normalizedOutputPath, err := filesystem.ValidateOutputPath(inputPath, outputPath, overwrite)
	if err != nil {
		return ConvertResult{}, err
	}

	overwritten := pathExists(normalizedOutputPath)

	temporaryFile, err := os.CreateTemp(filepath.Dir(normalizedOutputPath), "jpg-to-webp-*.tmp")
	if err != nil {
		return ConvertResult{}, fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}
	temporaryPath := temporaryFile.Name()
	cleanupTemporaryFile := true
	defer func() {
		if cleanupTemporaryFile {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := s.encoder.Encode(temporaryFile, sourceImage, EncodeOptions{Quality: float32(quality)}); err != nil {
		_ = temporaryFile.Close()
		return ConvertResult{}, fmt.Errorf("%w: %v", ErrEncodeFailed, err)
	}

	if err := temporaryFile.Close(); err != nil {
		return ConvertResult{}, fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}

	if overwritten {
		if err := os.Remove(normalizedOutputPath); err != nil {
			return ConvertResult{}, fmt.Errorf("%w: %v", ErrWriteFailed, err)
		}
	}

	if err := os.Rename(temporaryPath, normalizedOutputPath); err != nil {
		return ConvertResult{}, fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}
	cleanupTemporaryFile = false

	outputInfo, err := os.Stat(normalizedOutputPath)
	if err != nil {
		return ConvertResult{}, fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}

	return ConvertResult{
		OutputPath:  normalizedOutputPath,
		OutputBytes: outputInfo.Size(),
		Quality:     quality,
		Overwritten: overwritten,
	}, nil
}

func validateQuality(quality int) error {
	if quality < 0 || quality > 100 {
		return fmt.Errorf("%w: %d", ErrInvalidQuality, quality)
	}

	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
