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
	if err := os.MkdirAll(absoluteInput, 0o755); err != nil {
		t.Fatalf("mkdir absolute input: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(absoluteOutput), 0o755); err != nil {
		t.Fatalf("mkdir absolute output dir: %v", err)
	}
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

func TestLoadConfigEResolvesBackslashConfigRootRelativePaths(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{
		"input_path": "\\input",
		"output_path": "\\output\\demo.cfe",
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

func TestLoadConfigESetsDerivedIdentityPaths(t *testing.T) {
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

	if cfg.ProjectRootPath != tempDir {
		t.Fatalf("unexpected project root: got %q want %q", cfg.ProjectRootPath, tempDir)
	}
	expectedBindings := filepath.Join(tempDir, "configs", "base-bindings.json")
	if cfg.BaseBindingsPath != expectedBindings {
		t.Fatalf("unexpected base bindings path: got %q want %q", cfg.BaseBindingsPath, expectedBindings)
	}
	expectedIdentityMap := filepath.Join(tempDir, "output", "_state", "identity-map.json")
	if cfg.IdentityMapPath != expectedIdentityMap {
		t.Fatalf("unexpected identity map path: got %q want %q", cfg.IdentityMapPath, expectedIdentityMap)
	}
}

func TestLoadConfigEBackfillsExtensionPropertiesFromLegacyFields(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{
		"extension": "ТестовоеРасширение",
		"prefix": "тст_",
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

	if cfg.ExtensionProperties.Name != "ТестовоеРасширение" {
		t.Fatalf("unexpected extension_properties.name: got %q", cfg.ExtensionProperties.Name)
	}
	if cfg.ExtensionProperties.Prefix != "тст_" {
		t.Fatalf("unexpected extension_properties.prefix: got %q", cfg.ExtensionProperties.Prefix)
	}
}

func TestLoadConfigEResolvesTargetXMLDumpPath(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	configJSON := `{
		"input_path": "/input",
		"output_path": "/output/demo.cfe",
		"target": {
			"xml_dump": "/receiver"
		},
		"conversion_type": "srcConvert"
	}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigE(configPath)
	if err != nil {
		t.Fatalf("LoadConfigE: %v", err)
	}

	expected := filepath.Join(tempDir, "receiver")
	if cfg.Target.XMLDump != expected {
		t.Fatalf("unexpected target xml_dump path: got %q want %q", cfg.Target.XMLDump, expected)
	}
}

func TestLoadConfigEPreservesNormalizedConfigPath(t *testing.T) {
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

	cfg, err := LoadConfigE(filepath.Join(configDir, ".", "config.json"))
	if err != nil {
		t.Fatalf("LoadConfigE: %v", err)
	}

	if cfg.ConfigPath != filepath.Clean(configPath) {
		t.Fatalf("unexpected normalized config path: got %q want %q", cfg.ConfigPath, filepath.Clean(configPath))
	}
}

func TestConfigurationDefaultsToExactSearchResultTemplates(t *testing.T) {
	var cfg Configuration
	if !cfg.IsExactSearchResultTemplatesEnabled() {
		t.Fatalf("expected exact SearchResult templates to be enabled by default")
	}

	disabled := false
	cfg.AdditionalProcessing.UseExactTemplates = &disabled
	if cfg.IsExactSearchResultTemplatesEnabled() {
		t.Fatalf("expected explicit soft SearchResult matching to disable exact mode")
	}
}

func TestConfigurationDefaultsExtensionIdentifier(t *testing.T) {
	cfg, err := LoadDefaultConfigE()
	if err != nil {
		t.Fatalf("LoadDefaultConfigE: %v", err)
	}

	if got := cfg.ExtensionIdentifier(); got != "83b63dda-4eec-11f1-b61f-e0d55ee14481" {
		t.Fatalf("unexpected default extension identifier: got %q", got)
	}
}
