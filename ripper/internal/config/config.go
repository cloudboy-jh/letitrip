package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Naming struct {
	ArtistFormat   string `json:"artist_format"`
	FolderTemplate string `json:"folder_template"`
	FileTemplate   string `json:"file_template"`
}

type Config struct {
	OutputDir       string `json:"output_dir"`
	Drive           string `json:"drive"`
	EjectOnComplete bool   `json:"eject_on_complete"`
	Naming          Naming `json:"naming"`
}

func DefaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "D:\\Music"
	}
	return filepath.Join(home, "Music")
}

func defaultConfig() Config {
	return Config{
		OutputDir:       "",
		Drive:           "",
		EjectOnComplete: true,
		Naming: Naming{
			ArtistFormat:   "last, first",
			FolderTemplate: "{artist}/{album}",
			FileTemplate:   "{track} - {title}",
		},
	}
}

func Load() (Config, error) {
	cfg := defaultConfig()
	path, err := configPath()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "letitrip", "config.json"), nil
}
