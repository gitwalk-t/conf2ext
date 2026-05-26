package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirWithStatsPreservesFileMode(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "src")
	dest := filepath.Join(t.TempDir(), "dest")

	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir source tree: %v", err)
	}

	srcFile := filepath.Join(src, "nested", "file.txt")
	if err := os.WriteFile(srcFile, []byte("payload"), 0o640); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	stats, err := CopyDirWithStats(src, dest)
	if err != nil {
		t.Fatalf("CopyDirWithStats: %v", err)
	}

	if stats.Files != 1 || stats.Dirs != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}

	destFile := filepath.Join(dest, "nested", "file.txt")
	sourceInfo, err := os.Stat(srcFile)
	if err != nil {
		t.Fatalf("stat source file: %v", err)
	}
	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("stat copied file: %v", err)
	}
	if info.Mode().Perm() != sourceInfo.Mode().Perm() {
		t.Fatalf("expected copied file mode %o, got %o", sourceInfo.Mode().Perm(), info.Mode().Perm())
	}
}
