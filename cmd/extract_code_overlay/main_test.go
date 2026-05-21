package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gitwalk-m/conf2ext/internal/codeoverlay"
)

func TestParseOptionsUsesDefaults(t *testing.T) {
	opts, err := parseOptions(nil, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}

	if opts.extensionPath != codeoverlay.DefaultExtensionPath {
		t.Fatalf("unexpected default extension path: got %q want %q", opts.extensionPath, codeoverlay.DefaultExtensionPath)
	}
	if opts.outputPath != codeoverlay.DefaultOutputPath {
		t.Fatalf("unexpected default output path: got %q want %q", opts.outputPath, codeoverlay.DefaultOutputPath)
	}
	if opts.configPath != codeoverlay.DefaultConfigPath {
		t.Fatalf("unexpected default config path: got %q want %q", opts.configPath, codeoverlay.DefaultConfigPath)
	}
	if opts.extractorConfigPath != codeoverlay.DefaultExtractorConfigPath {
		t.Fatalf("unexpected default extractor config path: got %q want %q", opts.extractorConfigPath, codeoverlay.DefaultExtractorConfigPath)
	}
}

func TestRunPassesStandaloneOptionsToExtractor(t *testing.T) {
	oldRunExtraction := runExtraction
	defer func() {
		runExtraction = oldRunExtraction
	}()

	var gotOpts codeoverlay.Options
	runExtraction = func(opts codeoverlay.Options) (codeoverlay.Result, error) {
		gotOpts = opts
		return codeoverlay.Result{
			Artifact: codeoverlay.Artifact{
				Version: 1,
				Blocks: []codeoverlay.Block{
					{ID: "CommonModule.Тест:CommonModule"},
				},
			},
			ConfigPath:          "resolved/configs/config.json",
			ExtractorConfigPath: "resolved/cmd/extract_code_overlay/config.json",
			ExtensionPath:       "resolved/input/etalonCode",
			OutputPath:          "resolved/configs/code_overlay.json",
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run([]string{"--extension-path", "custom/input", "--output", "custom/output.json", "--config", "custom/config.json", "--extractor-config", "custom/extractor.json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run exit code = %d, stderr=%s", exitCode, stderr.String())
	}

	if gotOpts.ExtensionPath != "custom/input" {
		t.Fatalf("unexpected extension path passed to extractor: %q", gotOpts.ExtensionPath)
	}
	if gotOpts.OutputPath != "custom/output.json" {
		t.Fatalf("unexpected output path passed to extractor: %q", gotOpts.OutputPath)
	}
	if gotOpts.ConfigPath != "custom/config.json" {
		t.Fatalf("unexpected config path passed to extractor: %q", gotOpts.ConfigPath)
	}
	if gotOpts.ExtractorConfigPath != "custom/extractor.json" {
		t.Fatalf("unexpected extractor config path passed to extractor: %q", gotOpts.ExtractorConfigPath)
	}

	expected := "" +
		"overlay blocks extracted: 1\n" +
		"config path: resolved/configs/config.json\n" +
		"extractor config: resolved/cmd/extract_code_overlay/config.json\n" +
		"extension path: resolved/input/etalonCode\n" +
		"output: resolved/configs/code_overlay.json\n"
	if stdout.String() != expected {
		t.Fatalf("unexpected stdout:\n got %q\nwant %q", stdout.String(), expected)
	}
}

func TestRunReportsExtractorError(t *testing.T) {
	oldRunExtraction := runExtraction
	defer func() {
		runExtraction = oldRunExtraction
	}()

	runExtraction = func(opts codeoverlay.Options) (codeoverlay.Result, error) {
		return codeoverlay.Result{}, fmt.Errorf("boom")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run(nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.String() != "extract code overlay failed: boom\n" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
