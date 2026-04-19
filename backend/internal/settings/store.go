package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type AppSettings struct {
	LastInputDirectory string `json:"lastInputDirectory,omitempty"`
}

type Store interface {
	Load() (AppSettings, error)
	Save(AppSettings) error
}

type JSONStore struct {
	path string
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

func NewDefaultStore() *JSONStore {
	return NewJSONStore(defaultSettingsPath())
}

func (s *JSONStore) Load() (AppSettings, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppSettings{}, nil
		}
		return AppSettings{}, err
	}

	var appSettings AppSettings
	if err := json.Unmarshal(data, &appSettings); err != nil {
		return AppSettings{}, nil
	}

	return appSettings, nil
}

func (s *JSONStore) Save(appSettings AppSettings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(appSettings)
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o644)
}

func defaultSettingsPath() string {
	configDirectory, err := os.UserConfigDir()
	if err != nil || configDirectory == "" {
		return filepath.Join(".", ".jpg-to-webp", "settings.json")
	}

	return filepath.Join(configDirectory, "jpg-to-webp", "settings.json")
}
