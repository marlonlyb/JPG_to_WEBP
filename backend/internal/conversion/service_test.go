package conversion

import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"testing"

	"jpg-to-webp/backend/internal/filesystem"
)

type stubEncoder struct {
	encodedBytes []byte
	err          error
	qualities    []float32
	errByQuality map[float32]error
	errRemaining map[float32]int
}

func (s *stubEncoder) Encode(writer io.Writer, _ image.Image, options EncodeOptions) error {
	s.qualities = append(s.qualities, options.Quality)
	if s.errByQuality != nil {
		if err, ok := s.errByQuality[options.Quality]; ok {
			if s.errRemaining != nil {
				remaining := s.errRemaining[options.Quality]
				if remaining <= 0 {
					goto fallback
				}
				s.errRemaining[options.Quality] = remaining - 1
			}
			return err
		}
	}

fallback:
	if s.err != nil {
		return s.err
	}

	payload := s.encodedBytes
	if len(payload) == 0 {
		payload = []byte("webp payload")
	}

	_, err := writer.Write(payload)
	return err
}

func TestServiceConvertRejectsInvalidQuality(t *testing.T) {
	service := NewService(&stubEncoder{})

	tests := []struct {
		name    string
		quality int
	}{
		{name: "below range", quality: -1},
		{name: "above range", quality: 101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Convert(ConvertRequest{Quality: tt.quality})
			if !errors.Is(err, ErrInvalidQuality) {
				t.Fatalf("Convert() error = %v, want %v", err, ErrInvalidQuality)
			}
		})
	}
}

func TestServiceConvertWritesWebPOutput(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createTestJPEG(t, tempDir, "fixture.jpg")
	outputPath := filepath.Join(tempDir, "fixture.webp")
	encoder := &stubEncoder{encodedBytes: []byte("converted-webp")}
	service := NewService(encoder)

	result, err := service.Convert(ConvertRequest{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Quality:    85,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if result.OutputPath != outputPath {
		t.Fatalf("Convert() output path = %q, want %q", result.OutputPath, outputPath)
	}
	if result.Quality != 85 {
		t.Fatalf("Convert() quality = %d, want 85", result.Quality)
	}
	if result.Overwritten {
		t.Fatal("Convert() overwritten = true, want false")
	}
	if result.OutputBytes != int64(len("converted-webp")) {
		t.Fatalf("Convert() output bytes = %d, want %d", result.OutputBytes, len("converted-webp"))
	}
	if len(encoder.qualities) != 1 || encoder.qualities[0] != 85 {
		t.Fatalf("encoder qualities = %v, want [85]", encoder.qualities)
	}

	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
	}
	if string(outputData) != "converted-webp" {
		t.Fatalf("output contents = %q, want %q", outputData, "converted-webp")
	}
}

func TestServiceConvertOverwritesExistingFileWhenConfirmed(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createTestJPEG(t, tempDir, "fixture.jpg")
	outputPath := filepath.Join(tempDir, "fixture.webp")
	if err := os.WriteFile(outputPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed existing output: %v", err)
	}

	service := NewService(&stubEncoder{encodedBytes: []byte("new")})

	result, err := service.Convert(ConvertRequest{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Quality:    100,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if !result.Overwritten {
		t.Fatal("Convert() overwritten = false, want true")
	}

	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", outputPath, err)
	}
	if string(outputData) != "new" {
		t.Fatalf("output contents = %q, want %q", outputData, "new")
	}
}

func TestServiceConvertRequiresOverwriteConfirmation(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createTestJPEG(t, tempDir, "fixture.jpg")
	outputPath := filepath.Join(tempDir, "fixture.webp")
	if err := os.WriteFile(outputPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed existing output: %v", err)
	}

	service := NewService(&stubEncoder{})

	_, err := service.Convert(ConvertRequest{
		InputPath:  inputPath,
		OutputPath: outputPath,
		Quality:    80,
	})
	if !errors.Is(err, filesystem.ErrOutputExists) {
		t.Fatalf("Convert() error = %v, want %v", err, filesystem.ErrOutputExists)
	}
}

func TestServiceInspectBatchReturnsPlannedOutputs(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createTestJPEG(t, tempDir, "fixture.jpg")
	service := NewService(&stubEncoder{})

	items, err := service.InspectBatch([]string{inputPath})
	if err != nil {
		t.Fatalf("InspectBatch() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("InspectBatch() items = %d, want 1", len(items))
	}
	if items[0].Image.InputPath != inputPath {
		t.Fatalf("InspectBatch() input path = %q, want %q", items[0].Image.InputPath, inputPath)
	}
	if len(items[0].Outputs) != 3 {
		t.Fatalf("InspectBatch() outputs = %d, want 3", len(items[0].Outputs))
	}
	if items[0].Outputs[0].Quality != 100 || items[0].Outputs[1].Quality != 50 || items[0].Outputs[2].Quality != 25 {
		t.Fatalf("InspectBatch() output qualities = %#v, want [100 50 25]", items[0].Outputs)
	}
}

func TestServiceConvertBatchContinuesAfterPerFileFailure(t *testing.T) {
	tempDir := t.TempDir()
	firstInputPath := createTestJPEG(t, tempDir, "first.jpg")
	brokenInputPath := filepath.Join(tempDir, "broken.jpg")
	if err := os.WriteFile(brokenInputPath, []byte("not a jpeg"), 0o644); err != nil {
		t.Fatalf("write broken jpeg: %v", err)
	}
	thirdInputPath := createTestJPEG(t, tempDir, "third.jpeg")

	service := NewService(&stubEncoder{encodedBytes: []byte("batch-webp")})

	result, err := service.ConvertBatch(BatchConvertRequest{Inputs: []string{firstInputPath, brokenInputPath, thirdInputPath}})
	if err != nil {
		t.Fatalf("ConvertBatch() error = %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("ConvertBatch() items = %d, want 3", len(result.Items))
	}
	if result.Items[0].Status != BatchItemStatusSuccess {
		t.Fatalf("first status = %q, want %q", result.Items[0].Status, BatchItemStatusSuccess)
	}
	if result.Items[1].Status != BatchItemStatusFailed {
		t.Fatalf("second status = %q, want %q", result.Items[1].Status, BatchItemStatusFailed)
	}
	if result.Items[2].Status != BatchItemStatusSuccess {
		t.Fatalf("third status = %q, want %q", result.Items[2].Status, BatchItemStatusSuccess)
	}
	if result.Summary.TotalInputs != 3 || result.Summary.CompletedInputs != 3 || result.Summary.FailedInputs != 1 || result.Summary.TotalOutputs != 9 || result.Summary.WrittenOutputs != 6 {
		t.Fatalf("ConvertBatch() summary = %#v", result.Summary)
	}
	for _, outputPath := range []string{
		filepath.Join(tempDir, "first_high.webp"),
		filepath.Join(tempDir, "first_medium.webp"),
		filepath.Join(tempDir, "first_low.webp"),
		filepath.Join(tempDir, "third_high.webp"),
		filepath.Join(tempDir, "third_medium.webp"),
		filepath.Join(tempDir, "third_low.webp"),
	} {
		if _, statErr := os.Stat(outputPath); statErr != nil {
			t.Fatalf("Stat(%q) error = %v", outputPath, statErr)
		}
	}
}

func TestServiceConvertBatchMarksPartialResultsAndContinues(t *testing.T) {
	tempDir := t.TempDir()
	partialInputPath := createTestJPEG(t, tempDir, "partial.jpg")
	successInputPath := createTestJPEG(t, tempDir, "success.jpg")
	service := NewService(&stubEncoder{
		encodedBytes: []byte("batch-webp"),
		errByQuality: map[float32]error{50: errors.New("medium encode boom")},
		errRemaining: map[float32]int{50: 1},
	})

	result, err := service.ConvertBatch(BatchConvertRequest{Inputs: []string{partialInputPath, successInputPath}, Overwrite: true})
	if err != nil {
		t.Fatalf("ConvertBatch() error = %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("ConvertBatch() items = %d, want 2", len(result.Items))
	}
	if result.Items[0].Status != BatchItemStatusPartial {
		t.Fatalf("first status = %q, want %q", result.Items[0].Status, BatchItemStatusPartial)
	}
	if len(result.Items[0].Outputs) != 1 {
		t.Fatalf("first outputs = %d, want 1", len(result.Items[0].Outputs))
	}
	if !errors.Is(result.Items[0].Error, ErrEncodeFailed) {
		t.Fatalf("first error = %v, want %v", result.Items[0].Error, ErrEncodeFailed)
	}
	if result.Items[1].Status != BatchItemStatusSuccess {
		t.Fatalf("second status = %q, want %q", result.Items[1].Status, BatchItemStatusSuccess)
	}
	if len(result.Items[1].Outputs) != 3 {
		t.Fatalf("second outputs = %d, want 3", len(result.Items[1].Outputs))
	}
	if result.Summary.TotalInputs != 2 || result.Summary.CompletedInputs != 2 || result.Summary.FailedInputs != 1 || result.Summary.TotalOutputs != 6 || result.Summary.WrittenOutputs != 4 {
		t.Fatalf("ConvertBatch() summary = %#v", result.Summary)
	}
}

func createTestJPEG(t *testing.T, directory string, name string) string {
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
