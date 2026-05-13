package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPreservesDerivedConfigPaths(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{
		"input_path": "/input",
		"output_path": "/output/demo.cfe",
		"conversion_type": "srcConvert",
		"AdditionalProcessing": {
			"Use_упо_SearchResult": true
		}
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(filepath.Join(configDir, ".", "config.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.ConfigPath != filepath.Clean(configPath) {
		t.Fatalf("unexpected config path: got %q want %q", cfg.ConfigPath, filepath.Clean(configPath))
	}
	if cfg.ProjectRootPath != tempDir {
		t.Fatalf("unexpected project root path: got %q want %q", cfg.ProjectRootPath, tempDir)
	}
	if cfg.BaseBindingsPath != filepath.Join(tempDir, "configs", "base-bindings.json") {
		t.Fatalf("unexpected base bindings path: got %q", cfg.BaseBindingsPath)
	}
	if cfg.IdentityMapPath != filepath.Join(tempDir, "output", "_state", "identity-map.json") {
		t.Fatalf("unexpected identity map path: got %q", cfg.IdentityMapPath)
	}
}
