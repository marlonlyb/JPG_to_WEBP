package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONStoreLoad(t *testing.T) {
	tempDir := t.TempDir()
	settingsPath := filepath.Join(tempDir, "settings.json")
	store := NewJSONStore(settingsPath)

	tests := []struct {
		name    string
		seed    func(t *testing.T)
		want    AppSettings
		wantErr bool
	}{
		{
			name: "returns empty settings when file is missing",
			want: AppSettings{},
		},
		{
			name: "returns empty settings when file is corrupt",
			seed: func(t *testing.T) {
				t.Helper()
				if err := os.WriteFile(settingsPath, []byte("{not-json"), 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			},
			want: AppSettings{},
		},
		{
			name: "loads persisted input directory",
			seed: func(t *testing.T) {
				t.Helper()
				if err := store.Save(AppSettings{LastInputDirectory: "/mnt/h/Pictures"}); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			},
			want: AppSettings{LastInputDirectory: "/mnt/h/Pictures"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Remove(settingsPath); err != nil && !os.IsNotExist(err) {
				t.Fatalf("Remove() error = %v", err)
			}

			if tt.seed != nil {
				tt := tt
				tt.seed(t)
			}

			got, err := store.Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
