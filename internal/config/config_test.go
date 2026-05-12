package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigEResolvesConfigRootRelativePaths(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{
		"input_path": "/input",
		"output_path": "/output/demo.cfe",
		"conversion_type": "srcConvert"
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigE(configPath)
	if err != nil {
		t.Fatalf("LoadConfigE: %v", err)
	}

	expectedInput := filepath.Join(tempDir, "input")
	expectedOutput := filepath.Join(tempDir, "output", "demo.cfe")
	if cfg.InputPath != expectedInput {
		t.Fatalf("unexpected input path: got %q want %q", cfg.InputPath, expectedInput)
	}
	if cfg.OutputPath != expectedOutput {
		t.Fatalf("unexpected output path: got %q want %q", cfg.OutputPath, expectedOutput)
	}
}

func TestLoadConfigEPreservesAbsolutePaths(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}

	absoluteInput := filepath.Join(tempDir, "external", "input")
	absoluteOutput := filepath.Join(tempDir, "external", "output", "demo.cfe")
	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{
		"input_path": "` + filepath.ToSlash(absoluteInput) + `",
		"output_path": "` + filepath.ToSlash(absoluteOutput) + `",
		"conversion_type": "srcConvert"
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigE(configPath)
	if err != nil {
		t.Fatalf("LoadConfigE: %v", err)
	}

	if cfg.InputPath != filepath.Clean(absoluteInput) {
		t.Fatalf("unexpected absolute input path: got %q want %q", cfg.InputPath, filepath.Clean(absoluteInput))
	}
	if cfg.OutputPath != filepath.Clean(absoluteOutput) {
		t.Fatalf("unexpected absolute output path: got %q want %q", cfg.OutputPath, filepath.Clean(absoluteOutput))
	}
}
