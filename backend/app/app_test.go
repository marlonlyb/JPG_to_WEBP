package app

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"testing"

	"jpg-to-webp/backend/internal/conversion"
	"jpg-to-webp/backend/internal/settings"
)

type stubDialogs struct {
	openResult           string
	openErr              error
	openResults          []string
	openCallCount        int
	openManyResults      []string
	openManyErr          error
	openManyDefaultDir   string
	saveResult           string
	saveErr              error
	openDefaultDirectory string
}

func (s *stubDialogs) OpenJPEGFile(_ context.Context, defaultDirectory string) (string, error) {
	s.openDefaultDirectory = defaultDirectory
	if s.openCallCount < len(s.openResults) {
		result := s.openResults[s.openCallCount]
		s.openCallCount++
		return result, s.openErr
	}
	return s.openResult, s.openErr
}

func (s *stubDialogs) OpenJPEGFiles(_ context.Context, defaultDirectory string) ([]string, error) {
	s.openManyDefaultDir = defaultDirectory
	return s.openManyResults, s.openManyErr
}

func (s *stubDialogs) SaveWebPFile(context.Context, string) (string, error) {
	return s.saveResult, s.saveErr
}

type stubEncoder struct {
	err error
}

func (s stubEncoder) Encode(writer io.Writer, _ image.Image, _ conversion.EncodeOptions) error {
	if s.err != nil {
		return s.err
	}
	_, err := writer.Write([]byte("webp"))
	return err
}

func TestPickOutputPathReturnsEmptyWhenSaveIsCanceled(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{saveResult: ""},
		converter: conversion.NewService(stubEncoder{}),
	}

	selectedPath, err := application.PickOutputPath(inputPath)
	if err != nil {
		t.Fatalf("PickOutputPath() error = %v", err)
	}
	if selectedPath != "" {
		t.Fatalf("PickOutputPath() = %q, want empty string", selectedPath)
	}
}

func TestPickOutputPathMapsExistingOutputToWriteFailure(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	existingOutputPath := filepath.Join(tempDir, "photo.webp")
	if err := os.WriteFile(existingOutputPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed existing output: %v", err)
	}

	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{saveResult: existingOutputPath},
		converter: conversion.NewService(stubEncoder{}),
	}

	_, err := application.PickOutputPath(inputPath)
	assertAppError(t, err, AppErrorCodeWriteFailed)
}

func TestPickInputFileRejectsUnsupportedFile(t *testing.T) {
	tempDir := t.TempDir()
	unsupportedPath := filepath.Join(tempDir, "notes.txt")
	if err := os.WriteFile(unsupportedPath, []byte("not a jpeg"), 0o644); err != nil {
		t.Fatalf("write unsupported file: %v", err)
	}

	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{openResult: unsupportedPath},
		converter: conversion.NewService(stubEncoder{}),
	}

	_, err := application.PickInputFile()
	assertAppError(t, err, AppErrorCodeInvalidInput)
}

func TestGetImageInfoMapsDecodeFailuresToReadFailed(t *testing.T) {
	tempDir := t.TempDir()
	invalidJPEGPath := filepath.Join(tempDir, "broken.jpg")
	if err := os.WriteFile(invalidJPEGPath, []byte("not a real jpeg"), 0o644); err != nil {
		t.Fatalf("write invalid jpeg: %v", err)
	}

	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{},
		converter: conversion.NewService(stubEncoder{}),
	}

	_, err := application.GetImageInfo(invalidJPEGPath)
	assertAppError(t, err, AppErrorCodeReadFailed)
}

func TestConvertToWebPMapsInvalidQuality(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	outputPath := filepath.Join(tempDir, "photo.webp")
	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{},
		converter: conversion.NewService(stubEncoder{}),
	}

	_, err := application.ConvertToWebP(ConvertRequestDTO{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Quality:    101,
	})
	assertAppError(t, err, AppErrorCodeInvalidQuality)
}

func TestConvertToWebPMapsEncodeFailures(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	outputPath := filepath.Join(tempDir, "photo.webp")
	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{},
		converter: conversion.NewService(stubEncoder{err: errors.New("encode boom")}),
	}

	_, err := application.ConvertToWebP(ConvertRequestDTO{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Quality:    80,
	})
	assertAppError(t, err, AppErrorCodeEncodeFailed)
}

func TestPickInputFileUsesRememberedDirectoryBeforeFallbacks(t *testing.T) {
	tempDir := t.TempDir()
	rememberedDirectory := filepath.Join(tempDir, "remembered")
	mountHDirectory := filepath.Join(tempDir, "mnt-h")
	homeDirectory := filepath.Join(tempDir, "home")

	for _, directory := range []string{rememberedDirectory, mountHDirectory, homeDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", directory, err)
		}
	}

	settingsStore := settings.NewJSONStore(filepath.Join(tempDir, "settings.json"))
	if err := settingsStore.Save(settings.AppSettings{LastInputDirectory: rememberedDirectory}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dialogs := &stubDialogs{}
	application := &App{
		ctx:                 context.Background(),
		dialogs:             dialogs,
		converter:           conversion.NewService(stubEncoder{}),
		settingsStore:       settingsStore,
		preferredInputRoots: []string{mountHDirectory},
		homeDirectory:       homeDirectory,
	}

	if _, err := application.PickInputFile(); err != nil {
		t.Fatalf("PickInputFile() error = %v", err)
	}

	if dialogs.openDefaultDirectory != rememberedDirectory {
		t.Fatalf("OpenJPEGFile() defaultDirectory = %q, want %q", dialogs.openDefaultDirectory, rememberedDirectory)
	}
}

func TestPickInputFileFallsBackToPreferredMountWhenRememberedDirectoryIsStale(t *testing.T) {
	tempDir := t.TempDir()
	staleDirectory := filepath.Join(tempDir, "missing")
	mountHDirectory := filepath.Join(tempDir, "mnt-h")
	homeDirectory := filepath.Join(tempDir, "home")

	for _, directory := range []string{mountHDirectory, homeDirectory} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", directory, err)
		}
	}

	settingsStore := settings.NewJSONStore(filepath.Join(tempDir, "settings.json"))
	if err := settingsStore.Save(settings.AppSettings{LastInputDirectory: staleDirectory}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dialogs := &stubDialogs{}
	application := &App{
		ctx:                 context.Background(),
		dialogs:             dialogs,
		converter:           conversion.NewService(stubEncoder{}),
		settingsStore:       settingsStore,
		preferredInputRoots: []string{mountHDirectory, filepath.Join(tempDir, "mnt")},
		homeDirectory:       homeDirectory,
	}

	if _, err := application.PickInputFile(); err != nil {
		t.Fatalf("PickInputFile() error = %v", err)
	}

	if dialogs.openDefaultDirectory != mountHDirectory {
		t.Fatalf("OpenJPEGFile() defaultDirectory = %q, want %q", dialogs.openDefaultDirectory, mountHDirectory)
	}
}

func TestPickInputFilePersistsSuccessfulBrowseDirectory(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	settingsStore := settings.NewJSONStore(filepath.Join(tempDir, "settings.json"))
	application := &App{
		ctx:           context.Background(),
		dialogs:       &stubDialogs{openResult: inputPath},
		converter:     conversion.NewService(stubEncoder{}),
		settingsStore: settingsStore,
	}

	selectedPath, err := application.PickInputFile()
	if err != nil {
		t.Fatalf("PickInputFile() error = %v", err)
	}
	if selectedPath != inputPath {
		t.Fatalf("PickInputFile() = %q, want %q", selectedPath, inputPath)
	}

	storedSettings, err := settingsStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if storedSettings.LastInputDirectory != tempDir {
		t.Fatalf("LastInputDirectory = %q, want %q", storedSettings.LastInputDirectory, tempDir)
	}
}

func TestGetImageInfoPersistsOnlySuccessfulInspections(t *testing.T) {
	tempDir := t.TempDir()
	validInputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	invalidInputPath := filepath.Join(tempDir, "broken.jpg")
	if err := os.WriteFile(invalidInputPath, []byte("not a jpeg"), 0o644); err != nil {
		t.Fatalf("write invalid jpeg: %v", err)
	}

	settingsStore := settings.NewJSONStore(filepath.Join(tempDir, "settings.json"))
	application := &App{
		ctx:           context.Background(),
		dialogs:       &stubDialogs{},
		converter:     conversion.NewService(stubEncoder{}),
		settingsStore: settingsStore,
	}

	if _, err := application.GetImageInfo(validInputPath); err != nil {
		t.Fatalf("GetImageInfo(valid) error = %v", err)
	}

	storedSettings, err := settingsStore.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if storedSettings.LastInputDirectory != tempDir {
		t.Fatalf("LastInputDirectory after success = %q, want %q", storedSettings.LastInputDirectory, tempDir)
	}

	_, err = application.GetImageInfo(invalidInputPath)
	assertAppError(t, err, AppErrorCodeReadFailed)

	storedSettings, err = settingsStore.Load()
	if err != nil {
		t.Fatalf("Load() after failed inspect error = %v", err)
	}
	if storedSettings.LastInputDirectory != tempDir {
		t.Fatalf("LastInputDirectory after failed inspect = %q, want %q", storedSettings.LastInputDirectory, tempDir)
	}
}

func TestGetImageInfoReturnsNormalizedExifDimensions(t *testing.T) {
	inputPath := copyConversionFixtureForAppTest(t, "orientation-6.jpg")
	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{},
		converter: conversion.NewService(stubEncoder{}),
	}

	info, err := application.GetImageInfo(inputPath)
	if err != nil {
		t.Fatalf("GetImageInfo() error = %v", err)
	}

	if info.Width != 3 || info.Height != 2 {
		t.Fatalf("GetImageInfo() dimensions = %dx%d, want 3x2", info.Width, info.Height)
	}
}

func TestPickInputFilesFallsBackToSinglePickerAndDedupes(t *testing.T) {
	tempDir := t.TempDir()
	firstInputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	secondInputPath := createJPEGFixture(t, tempDir, "photo-2.jpeg")
	dialogs := &stubDialogs{
		openManyErr: errors.New("multi picker unavailable"),
		openResults: []string{firstInputPath, "  \"" + firstInputPath + "\"  ", secondInputPath, ""},
	}
	application := &App{
		ctx:       context.Background(),
		dialogs:   dialogs,
		converter: conversion.NewService(stubEncoder{}),
	}

	selectedPaths, err := application.PickInputFiles()
	if err != nil {
		t.Fatalf("PickInputFiles() error = %v", err)
	}
	if len(selectedPaths) != 2 {
		t.Fatalf("PickInputFiles() len = %d, want 2", len(selectedPaths))
	}
	if selectedPaths[0] != firstInputPath || selectedPaths[1] != secondInputPath {
		t.Fatalf("PickInputFiles() = %#v, want [%q %q]", selectedPaths, firstInputPath, secondInputPath)
	}
}

func TestPickInputFilesRejectsInvalidSelections(t *testing.T) {
	tempDir := t.TempDir()
	validInputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	unsupportedPath := filepath.Join(tempDir, "notes.txt")
	if err := os.WriteFile(unsupportedPath, []byte("not jpeg"), 0o644); err != nil {
		t.Fatalf("write unsupported file: %v", err)
	}
	overLimitPaths := []string{
		createJPEGFixture(t, tempDir, "over-01.jpg"),
		createJPEGFixture(t, tempDir, "over-02.jpg"),
		createJPEGFixture(t, tempDir, "over-03.jpg"),
		createJPEGFixture(t, tempDir, "over-04.jpg"),
		createJPEGFixture(t, tempDir, "over-05.jpg"),
		createJPEGFixture(t, tempDir, "over-06.jpg"),
		createJPEGFixture(t, tempDir, "over-07.jpg"),
		createJPEGFixture(t, tempDir, "over-08.jpg"),
		createJPEGFixture(t, tempDir, "over-09.jpg"),
		createJPEGFixture(t, tempDir, "over-10.jpg"),
		createJPEGFixture(t, tempDir, "over-11.jpg"),
	}

	tests := []struct {
		name            string
		openManyResults []string
	}{
		{
			name:            "rejects unsupported paths returned by picker",
			openManyResults: []string{validInputPath, unsupportedPath},
		},
		{
			name:            "rejects selections above max batch limit",
			openManyResults: overLimitPaths,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := &App{
				ctx:       context.Background(),
				dialogs:   &stubDialogs{openManyResults: tt.openManyResults},
				converter: conversion.NewService(stubEncoder{}),
			}

			_, err := application.PickInputFiles()
			assertAppError(t, err, AppErrorCodeInvalidInput)
		})
	}
}

func TestInspectBatchInputsRejectsUnsupportedSelections(t *testing.T) {
	tempDir := t.TempDir()
	validInputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	invalidInputPath := filepath.Join(tempDir, "notes.txt")
	if err := os.WriteFile(invalidInputPath, []byte("not jpeg"), 0o644); err != nil {
		t.Fatalf("write invalid input: %v", err)
	}
	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{},
		converter: conversion.NewService(stubEncoder{}),
	}

	_, err := application.InspectBatchInputs([]string{validInputPath, invalidInputPath})
	assertAppError(t, err, AppErrorCodeInvalidInput)
}

func TestInspectBatchInputsReturnsNormalizedDimensions(t *testing.T) {
	inputPath := copyConversionFixtureForAppTest(t, "orientation-8.jpg")
	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{},
		converter: conversion.NewService(stubEncoder{}),
	}

	inspection, err := application.InspectBatchInputs([]string{inputPath})
	if err != nil {
		t.Fatalf("InspectBatchInputs() error = %v", err)
	}

	if len(inspection.Items) != 1 {
		t.Fatalf("InspectBatchInputs() items = %d, want 1", len(inspection.Items))
	}
	if inspection.Items[0].Input.Width != 3 || inspection.Items[0].Input.Height != 2 {
		t.Fatalf("InspectBatchInputs() dimensions = %dx%d, want 3x2", inspection.Items[0].Input.Width, inspection.Items[0].Input.Height)
	}
}

func TestPreflightBatchReturnsConflictPaths(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGFixture(t, tempDir, "photo.jpg")
	conflictPath := filepath.Join(tempDir, "photo_medium.webp")
	if err := os.WriteFile(conflictPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed conflict: %v", err)
	}
	application := &App{
		ctx:       context.Background(),
		dialogs:   &stubDialogs{},
		converter: conversion.NewService(stubEncoder{}),
	}

	result, err := application.PreflightBatch([]string{inputPath})
	if err != nil {
		t.Fatalf("PreflightBatch() error = %v", err)
	}
	if result.TotalConflicts != 1 || len(result.Conflicts) != 1 || result.Conflicts[0] != conflictPath {
		t.Fatalf("PreflightBatch() = %#v, want one conflict %q", result, conflictPath)
	}
}

func assertAppError(t *testing.T, err error, wantCode AppErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("expected app error, got nil")
	}

	var appError AppErrorDTO
	if decodeErr := json.Unmarshal([]byte(err.Error()), &appError); decodeErr != nil {
		t.Fatalf("decode app error: %v", decodeErr)
	}
	if appError.Code != wantCode {
		t.Fatalf("app error code = %q, want %q", appError.Code, wantCode)
	}
}

func createJPEGFixture(t *testing.T, directory string, name string) string {
	t.Helper()

	path := filepath.Join(directory, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	defer file.Close()

	imageData := image.NewRGBA(image.Rect(0, 0, 2, 2))
	imageData.Set(0, 0, color.RGBA{R: 255, A: 255})
	imageData.Set(1, 0, color.RGBA{G: 255, A: 255})
	imageData.Set(0, 1, color.RGBA{B: 255, A: 255})
	imageData.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	if err := jpeg.Encode(file, imageData, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("jpeg.Encode() error = %v", err)
	}

	return path
}

func copyConversionFixtureForAppTest(t *testing.T, name string) string {
	t.Helper()

	sourcePath := filepath.Join("..", "internal", "conversion", "testdata", name)
	payload, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", sourcePath, err)
	}

	destinationPath := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(destinationPath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", destinationPath, err)
	}

	return destinationPath
}
