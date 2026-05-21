package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gitwalk-m/conf2ext/internal/config"
)

func TestApplyCodeOverlayIfEnabledDisabledPreservesFiles(t *testing.T) {
	root := t.TempDir()
	modulePath := filepath.Join(root, "CommonModules", "Тест", "Ext", "Module.bsl")
	if err := os.MkdirAll(filepath.Dir(modulePath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(modulePath, []byte("// generated\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg := &config.Configuration{
		ProjectRootPath: root,
	}

	if err := applyCodeOverlayIfEnabled(cfg, root); err != nil {
		t.Fatalf("applyCodeOverlayIfEnabled: %v", err)
	}

	content, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "// generated\n" {
		t.Fatalf("expected disabled overlay to preserve generated content, got %q", string(content))
	}
}
