package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateJPEGInputPath(t *testing.T) {
	tempDir := t.TempDir()
	validJPEGPath := filepath.Join(tempDir, "photo.JPG")
	if err := os.WriteFile(validJPEGPath, []byte("jpeg"), 0o644); err != nil {
		t.Fatalf("write valid jpeg placeholder: %v", err)
	}

	directoryPath := filepath.Join(tempDir, "folder.jpeg")
	if err := os.Mkdir(directoryPath, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}

	tests := []struct {
		name      string
		inputPath string
		wantPath  string
		wantErr   error
	}{
		{
			name:      "accepts jpg path case insensitive",
			inputPath: validJPEGPath,
			wantPath:  validJPEGPath,
		},
		{
			name:      "trims outer whitespace and quotes",
			inputPath: "  \" " + validJPEGPath + " \"  ",
			wantPath:  validJPEGPath,
		},
		{
			name:      "rejects empty path",
			inputPath: "   ",
			wantErr:   ErrEmptyPath,
		},
		{
			name:      "rejects raw windows drive syntax",
			inputPath: `H:\\Pictures\\photo.jpg`,
			wantErr:   ErrUnsupportedPathSyntax,
		},
		{
			name:      "rejects unsupported extension",
			inputPath: filepath.Join(tempDir, "photo.png"),
			wantErr:   ErrUnsupportedExtension,
		},
		{
			name:      "rejects missing file",
			inputPath: filepath.Join(tempDir, "missing.jpg"),
			wantErr:   ErrMissingFile,
		},
		{
			name:      "rejects directories",
			inputPath: directoryPath,
			wantErr:   ErrNotRegularFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := ValidateJPEGInputPath(tt.inputPath)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateJPEGInputPath() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if gotPath != tt.wantPath {
				t.Fatalf("ValidateJPEGInputPath() path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestDefaultOutputPath(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "nested", "photo.jpeg")
	if err := os.MkdirAll(filepath.Dir(inputPath), 0o755); err != nil {
		t.Fatalf("create input parent: %v", err)
	}

	gotPath, err := DefaultOutputPath(inputPath)
	if err != nil {
		t.Fatalf("DefaultOutputPath() error = %v", err)
	}

	wantPath := filepath.Join(tempDir, "nested", "photo.webp")
	if gotPath != wantPath {
		t.Fatalf("DefaultOutputPath() = %q, want %q", gotPath, wantPath)
	}
}

func TestSuggestOutputPath(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGPlaceholder(t, tempDir, "photo.jpg")

	t.Run("uses default output when no conflict exists", func(t *testing.T) {
		gotPath, hadConflict, err := SuggestOutputPath(inputPath)
		if err != nil {
			t.Fatalf("SuggestOutputPath() error = %v", err)
		}

		wantPath := filepath.Join(tempDir, "photo.webp")
		if gotPath != wantPath {
			t.Fatalf("SuggestOutputPath() path = %q, want %q", gotPath, wantPath)
		}
		if hadConflict {
			t.Fatal("SuggestOutputPath() hadConflict = true, want false")
		}
	})

	t.Run("suggests incremented suffix when conflicts exist", func(t *testing.T) {
		for _, name := range []string{"photo.webp", "photo (1).webp"} {
			if err := os.WriteFile(filepath.Join(tempDir, name), []byte(name), 0o644); err != nil {
				t.Fatalf("seed existing output %q: %v", name, err)
			}
		}

		gotPath, hadConflict, err := SuggestOutputPath(inputPath)
		if err != nil {
			t.Fatalf("SuggestOutputPath() error = %v", err)
		}

		wantPath := filepath.Join(tempDir, "photo (2).webp")
		if gotPath != wantPath {
			t.Fatalf("SuggestOutputPath() path = %q, want %q", gotPath, wantPath)
		}
		if !hadConflict {
			t.Fatal("SuggestOutputPath() hadConflict = false, want true")
		}
	})
}

func TestValidateOutputPath(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGPlaceholder(t, tempDir, "source.jpg")
	existingOutputPath := filepath.Join(tempDir, "result.webp")
	if err := os.WriteFile(existingOutputPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed existing output: %v", err)
	}

	tests := []struct {
		name       string
		outputPath string
		overwrite  bool
		wantPath   string
		wantErr    error
	}{
		{
			name:       "rejects non webp output extension",
			outputPath: filepath.Join(tempDir, "result.png"),
			wantErr:    ErrInvalidOutputExtension,
		},
		{
			name:       "rejects missing parent directory",
			outputPath: filepath.Join(tempDir, "missing", "result.webp"),
			wantErr:    ErrMissingParentDirectory,
		},
		{
			name:       "rejects overwrite without confirmation",
			outputPath: existingOutputPath,
			wantErr:    ErrOutputExists,
		},
		{
			name:       "allows overwrite with confirmation",
			outputPath: existingOutputPath,
			overwrite:  true,
			wantPath:   existingOutputPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := ValidateOutputPath(inputPath, tt.outputPath, tt.overwrite)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateOutputPath() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if gotPath != tt.wantPath {
				t.Fatalf("ValidateOutputPath() path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestNormalizeBatchInputPaths(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := createJPEGPlaceholder(t, tempDir, "photo.jpg")
	secondInputPath := createJPEGPlaceholder(t, tempDir, "photo-2.jpeg")
	maxBatchInputs := createJPEGPlaceholders(t, tempDir, []string{
		"max-01.jpg",
		"max-02.jpg",
		"max-03.jpg",
		"max-04.jpg",
		"max-05.jpg",
		"max-06.jpg",
		"max-07.jpg",
		"max-08.jpg",
		"max-09.jpg",
		"max-10.jpg",
	})

	tests := []struct {
		name       string
		inputPaths []string
		wantPaths  []string
		wantErr    error
	}{
		{
			name:       "dedupes normalized paths while preserving order",
			inputPaths: []string{inputPath, " \"" + inputPath + "\" ", secondInputPath},
			wantPaths:  []string{inputPath, secondInputPath},
		},
		{
			name:       "accepts exactly max supported inputs",
			inputPaths: maxBatchInputs,
			wantPaths:  maxBatchInputs,
		},
		{
			name:       "rejects empty selections",
			inputPaths: nil,
			wantErr:    ErrBatchInputCount,
		},
		{
			name:       "rejects selections above limit",
			inputPaths: []string{inputPath, secondInputPath, inputPath, secondInputPath, inputPath, secondInputPath, inputPath, secondInputPath, inputPath, secondInputPath, createJPEGPlaceholder(t, tempDir, "photo-3.jpg")},
			wantErr:    ErrBatchInputCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPaths, err := NormalizeBatchInputPaths(tt.inputPaths)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NormalizeBatchInputPaths() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if len(gotPaths) != len(tt.wantPaths) {
				t.Fatalf("NormalizeBatchInputPaths() len = %d, want %d", len(gotPaths), len(tt.wantPaths))
			}

			for index, gotPath := range gotPaths {
				if gotPath != tt.wantPaths[index] {
					t.Fatalf("NormalizeBatchInputPaths()[%d] = %q, want %q", index, gotPath, tt.wantPaths[index])
				}
			}
		})
	}
}

func TestPlanBatchOutputs(t *testing.T) {
	tempDir := t.TempDir()
	tests := []struct {
		name          string
		inputName     string
		existingPaths []string
		wantOutputs   []struct {
			suffix  string
			quality int
			path    string
			exists  bool
		}
	}{
		{
			name:      "derives three fixed adjacent outputs for jpg",
			inputName: "photo.jpg",
			wantOutputs: []struct {
				suffix  string
				quality int
				path    string
				exists  bool
			}{
				{suffix: "high", quality: 100, path: filepath.Join(tempDir, "photo_high.webp"), exists: false},
				{suffix: "medium", quality: 50, path: filepath.Join(tempDir, "photo_medium.webp"), exists: false},
				{suffix: "low", quality: 25, path: filepath.Join(tempDir, "photo_low.webp"), exists: false},
			},
		},
		{
			name:          "marks existing conflicts in derived outputs",
			inputName:     "existing.jpeg",
			existingPaths: []string{filepath.Join(tempDir, "existing_medium.webp")},
			wantOutputs: []struct {
				suffix  string
				quality int
				path    string
				exists  bool
			}{
				{suffix: "high", quality: 100, path: filepath.Join(tempDir, "existing_high.webp"), exists: false},
				{suffix: "medium", quality: 50, path: filepath.Join(tempDir, "existing_medium.webp"), exists: true},
				{suffix: "low", quality: 25, path: filepath.Join(tempDir, "existing_low.webp"), exists: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputPath := createJPEGPlaceholder(t, tempDir, tt.inputName)
			for _, existingPath := range tt.existingPaths {
				if err := os.WriteFile(existingPath, []byte("existing"), 0o644); err != nil {
					t.Fatalf("seed existing output %q: %v", existingPath, err)
				}
			}

			plans, err := PlanBatchOutputs([]string{inputPath})
			if err != nil {
				t.Fatalf("PlanBatchOutputs() error = %v", err)
			}
			if len(plans) != 1 {
				t.Fatalf("PlanBatchOutputs() plans = %d, want 1", len(plans))
			}

			for index, output := range plans[0].Outputs {
				wantOutput := tt.wantOutputs[index]
				if output.Suffix != wantOutput.suffix || output.Quality != wantOutput.quality || output.OutputPath != wantOutput.path || output.Exists != wantOutput.exists {
					t.Fatalf("PlanBatchOutputs() output[%d] = %#v, want %#v", index, output, wantOutput)
				}
			}
		})
	}
}

func TestBatchOverwriteConflicts(t *testing.T) {
	tempDir := t.TempDir()
	firstInputPath := createJPEGPlaceholder(t, tempDir, "photo.jpg")
	secondInputPath := createJPEGPlaceholder(t, tempDir, "other.jpeg")

	conflictPaths := []string{
		filepath.Join(tempDir, "photo_high.webp"),
		filepath.Join(tempDir, "other_low.webp"),
	}
	for _, conflictPath := range conflictPaths {
		if err := os.WriteFile(conflictPath, []byte("existing"), 0o644); err != nil {
			t.Fatalf("seed conflict %q: %v", conflictPath, err)
		}
	}

	conflicts, err := BatchOverwriteConflicts([]string{firstInputPath, secondInputPath})
	if err != nil {
		t.Fatalf("BatchOverwriteConflicts() error = %v", err)
	}
	if len(conflicts) != len(conflictPaths) {
		t.Fatalf("BatchOverwriteConflicts() len = %d, want %d", len(conflicts), len(conflictPaths))
	}
	for index, conflictPath := range conflictPaths {
		if conflicts[index] != conflictPath {
			t.Fatalf("BatchOverwriteConflicts()[%d] = %q, want %q", index, conflicts[index], conflictPath)
		}
	}
}

func createJPEGPlaceholder(t *testing.T, directory string, name string) string {
	t.Helper()

	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("jpeg placeholder"), 0o644); err != nil {
		t.Fatalf("write jpeg placeholder: %v", err)
	}

	return path
}

func createJPEGPlaceholders(t *testing.T, directory string, names []string) []string {
	t.Helper()

	paths := make([]string, 0, len(names))
	for _, name := range names {
		paths = append(paths, createJPEGPlaceholder(t, directory, name))
	}

	return paths
}
