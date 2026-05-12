package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	publicconfig "github.com/firstBitSportivnaya/files-converter/pkg/config"
)

func TestRunCheckConfigSkipsConversion(t *testing.T) {
	originalLoadConfig := loadConfig
	originalRunConversion := runConversion
	t.Cleanup(func() {
		loadConfig = originalLoadConfig
		runConversion = originalRunConversion
	})

	loadCalled := false
	runCalled := false

	loadConfig = func(path string) (*publicconfig.Configuration, error) {
		loadCalled = true
		if path != "./configs/config.json" {
			t.Fatalf("unexpected config path: %s", path)
		}
		return &publicconfig.Configuration{}, nil
	}
	runConversion = func(cfg *publicconfig.Configuration) error {
		runCalled = true
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"--check-config"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d, stderr=%q", exitCode, stderr.String())
	}
	if !loadCalled {
		t.Fatal("expected config to be loaded")
	}
	if runCalled {
		t.Fatal("expected conversion to be skipped in check-config mode")
	}
	if !strings.Contains(stdout.String(), "конфиг валиден") {
		t.Fatalf("expected validation message, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "config load duration:") {
		t.Fatalf("expected config duration, got %q", stdout.String())
	}
}

func TestRunUsesConfigFlagAndRunsConversion(t *testing.T) {
	originalLoadConfig := loadConfig
	originalRunConversion := runConversion
	t.Cleanup(func() {
		loadConfig = originalLoadConfig
		runConversion = originalRunConversion
	})

	expectedCfg := &publicconfig.Configuration{}
	runCalled := false

	loadConfig = func(path string) (*publicconfig.Configuration, error) {
		if path != "./configs/custom.json" {
			t.Fatalf("unexpected config path: %s", path)
		}
		return expectedCfg, nil
	}
	runConversion = func(cfg *publicconfig.Configuration) error {
		runCalled = true
		if cfg != expectedCfg {
			t.Fatal("unexpected config pointer passed to conversion")
		}
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"-c", "./configs/custom.json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d, stderr=%q", exitCode, stderr.String())
	}
	if !runCalled {
		t.Fatal("expected conversion to run")
	}
	if !strings.Contains(stdout.String(), "config load duration:") {
		t.Fatalf("expected config duration, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "conversion duration:") {
		t.Fatalf("expected conversion duration, got %q", stdout.String())
	}
}

func TestRunPreservesConversionErrorText(t *testing.T) {
	originalLoadConfig := loadConfig
	originalRunConversion := runConversion
	t.Cleanup(func() {
		loadConfig = originalLoadConfig
		runConversion = originalRunConversion
	})

	loadConfig = func(path string) (*publicconfig.Configuration, error) {
		return &publicconfig.Configuration{}, nil
	}
	runConversion = func(cfg *publicconfig.Configuration) error {
		return errors.New("boom")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected non-zero exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "конвертация завершилась ошибкой: boom") {
		t.Fatalf("expected original error text, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "config load duration:") {
		t.Fatalf("expected config duration, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "conversion duration:") {
		t.Fatalf("expected conversion duration, got %q", stderr.String())
	}
}
