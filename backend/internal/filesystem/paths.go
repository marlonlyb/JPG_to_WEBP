package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrEmptyPath              = errors.New("path is required")
	ErrBatchInputCount        = errors.New("batch input selection must contain between 1 and 10 files")
	ErrUnsupportedExtension   = errors.New("unsupported file extension")
	ErrMissingFile            = errors.New("file does not exist")
	ErrNotRegularFile         = errors.New("path is not a regular file")
	ErrInvalidOutputExtension = errors.New("output must end with .webp")
	ErrOutputExists           = errors.New("output file already exists")
	ErrSamePath               = errors.New("input and output paths must differ")
	ErrMissingParentDirectory = errors.New("output parent directory does not exist")
	ErrUnsupportedPathSyntax  = errors.New("unsupported path syntax")
)

const MaxBatchInputs = 10

type OutputVariantPlan struct {
	Suffix     string
	Quality    int
	OutputPath string
	Exists     bool
}

type BatchOutputPlan struct {
	InputPath string
	Outputs   []OutputVariantPlan
}

var batchOutputVariants = []struct {
	suffix  string
	quality int
}{
	{suffix: "high", quality: 100},
	{suffix: "medium", quality: 50},
	{suffix: "low", quality: 25},
}

func ValidateJPEGInputPath(inputPath string) (string, error) {
	normalizedPath, err := normalizePath(inputPath)
	if err != nil {
		return "", err
	}

	if !hasJPEGExtension(normalizedPath) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedExtension, normalizedPath)
	}

	info, err := os.Stat(normalizedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrMissingFile, normalizedPath)
		}
		return "", err
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s", ErrNotRegularFile, normalizedPath)
	}

	return normalizedPath, nil
}

func DefaultOutputPath(inputPath string) (string, error) {
	normalizedInputPath, err := normalizePath(inputPath)
	if err != nil {
		return "", err
	}

	if !hasJPEGExtension(normalizedInputPath) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedExtension, normalizedInputPath)
	}

	baseName := strings.TrimSuffix(filepath.Base(normalizedInputPath), filepath.Ext(normalizedInputPath))
	return filepath.Join(filepath.Dir(normalizedInputPath), baseName+".webp"), nil
}

func SuggestOutputPath(inputPath string) (string, bool, error) {
	normalizedInputPath, err := ValidateJPEGInputPath(inputPath)
	if err != nil {
		return "", false, err
	}

	defaultPath, err := DefaultOutputPath(normalizedInputPath)
	if err != nil {
		return "", false, err
	}

	if !pathExists(defaultPath) {
		return defaultPath, false, nil
	}

	directory := filepath.Dir(defaultPath)
	baseName := strings.TrimSuffix(filepath.Base(defaultPath), filepath.Ext(defaultPath))
	for index := 1; ; index++ {
		candidatePath := filepath.Join(directory, fmt.Sprintf("%s (%d).webp", baseName, index))
		if !pathExists(candidatePath) {
			return candidatePath, true, nil
		}
	}
}

func ValidateOutputPath(inputPath string, outputPath string, overwrite bool) (string, error) {
	normalizedInputPath, err := normalizePath(inputPath)
	if err != nil {
		return "", err
	}

	normalizedOutputPath, err := normalizePath(outputPath)
	if err != nil {
		return "", err
	}

	if !hasWebPExtension(normalizedOutputPath) {
		return "", fmt.Errorf("%w: %s", ErrInvalidOutputExtension, normalizedOutputPath)
	}

	if normalizedInputPath == normalizedOutputPath {
		return "", fmt.Errorf("%w: %s", ErrSamePath, normalizedOutputPath)
	}

	parentDirectory := filepath.Dir(normalizedOutputPath)
	parentInfo, err := os.Stat(parentDirectory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %s", ErrMissingParentDirectory, parentDirectory)
		}
		return "", err
	}
	if !parentInfo.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrMissingParentDirectory, parentDirectory)
	}

	outputInfo, err := os.Stat(normalizedOutputPath)
	if err == nil {
		if !outputInfo.Mode().IsRegular() {
			return "", fmt.Errorf("%w: %s", ErrNotRegularFile, normalizedOutputPath)
		}
		if !overwrite {
			return "", fmt.Errorf("%w: %s", ErrOutputExists, normalizedOutputPath)
		}
		return normalizedOutputPath, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	return normalizedOutputPath, nil
}

func NormalizeBatchInputPaths(inputPaths []string) ([]string, error) {
	if len(inputPaths) == 0 || len(inputPaths) > MaxBatchInputs {
		return nil, ErrBatchInputCount
	}

	seenPaths := make(map[string]struct{}, len(inputPaths))
	normalizedPaths := make([]string, 0, len(inputPaths))
	for _, inputPath := range inputPaths {
		normalizedPath, err := ValidateJPEGInputPath(inputPath)
		if err != nil {
			return nil, err
		}

		if _, seen := seenPaths[normalizedPath]; seen {
			continue
		}

		seenPaths[normalizedPath] = struct{}{}
		normalizedPaths = append(normalizedPaths, normalizedPath)
	}

	if len(normalizedPaths) == 0 {
		return nil, ErrBatchInputCount
	}

	return normalizedPaths, nil
}

func PlanOutputVariants(inputPath string) ([]OutputVariantPlan, error) {
	normalizedInputPath, err := ValidateJPEGInputPath(inputPath)
	if err != nil {
		return nil, err
	}

	baseName := strings.TrimSuffix(filepath.Base(normalizedInputPath), filepath.Ext(normalizedInputPath))
	outputDirectory := filepath.Dir(normalizedInputPath)
	outputs := make([]OutputVariantPlan, 0, len(batchOutputVariants))
	for _, variant := range batchOutputVariants {
		outputPath := filepath.Join(outputDirectory, fmt.Sprintf("%s_%s.webp", baseName, variant.suffix))
		outputs = append(outputs, OutputVariantPlan{
			Suffix:     variant.suffix,
			Quality:    variant.quality,
			OutputPath: outputPath,
			Exists:     pathExists(outputPath),
		})
	}

	return outputs, nil
}

func PlanBatchOutputs(inputPaths []string) ([]BatchOutputPlan, error) {
	normalizedPaths, err := NormalizeBatchInputPaths(inputPaths)
	if err != nil {
		return nil, err
	}

	plans := make([]BatchOutputPlan, 0, len(normalizedPaths))
	for _, inputPath := range normalizedPaths {
		outputs, err := PlanOutputVariants(inputPath)
		if err != nil {
			return nil, err
		}

		plans = append(plans, BatchOutputPlan{
			InputPath: inputPath,
			Outputs:   outputs,
		})
	}

	return plans, nil
}

func BatchOverwriteConflicts(inputPaths []string) ([]string, error) {
	plans, err := PlanBatchOutputs(inputPaths)
	if err != nil {
		return nil, err
	}

	conflicts := make([]string, 0)
	for _, plan := range plans {
		for _, output := range plan.Outputs {
			if output.Exists {
				conflicts = append(conflicts, output.OutputPath)
			}
		}
	}

	return conflicts, nil
}

func normalizePath(value string) (string, error) {
	trimmedValue := strings.TrimSpace(value)
	trimmedValue = trimMatchingOuterQuotes(trimmedValue)
	trimmedValue = strings.TrimSpace(trimmedValue)
	if trimmedValue == "" {
		return "", ErrEmptyPath
	}

	if runtime.GOOS != "windows" && hasWindowsDrivePrefix(trimmedValue) {
		return "", fmt.Errorf("%w: %s", ErrUnsupportedPathSyntax, trimmedValue)
	}

	absolutePath, err := filepath.Abs(trimmedValue)
	if err != nil {
		return "", err
	}

	return filepath.Clean(absolutePath), nil
}

func trimMatchingOuterQuotes(value string) string {
	if len(value) < 2 {
		return value
	}

	firstCharacter := value[0]
	lastCharacter := value[len(value)-1]
	if (firstCharacter == '"' || firstCharacter == '\'') && firstCharacter == lastCharacter {
		return value[1 : len(value)-1]
	}

	return value
}

func hasWindowsDrivePrefix(value string) bool {
	if len(value) < 3 {
		return false
	}

	firstCharacter := value[0]
	if !((firstCharacter >= 'a' && firstCharacter <= 'z') || (firstCharacter >= 'A' && firstCharacter <= 'Z')) {
		return false
	}

	return value[1] == ':' && (value[2] == '\\' || value[2] == '/')
}

func hasJPEGExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return true
	default:
		return false
	}
}

func hasWebPExtension(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".webp")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
