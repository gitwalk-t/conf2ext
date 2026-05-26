package fileutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type CopyStats struct {
	Files int
	Dirs  int
	Bytes int64
}

// CopyFile copies a file from src to dest.
func CopyFile(src, dest string) error {
	sourceFileInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	_, err = copyFileWithInfo(src, dest, sourceFileInfo)
	return err
}

func copyFileWithInfo(src, dest string, info fs.FileInfo) (int64, error) {
	if info == nil {
		var err error
		info, err = os.Stat(src)
		if err != nil {
			return 0, err
		}
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = destFile.Close()
	}()

	written, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return written, err
	}

	return written, nil
}

func CopyDir(src, dest string) error {
	_, err := CopyDirWithStats(src, dest)
	return err
}

func CopyDirWithStats(src, dest string) (CopyStats, error) {
	var stats CopyStats

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("ошибка при получении относительного пути %s: %w", path, err)
		}
		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return err
			}
			stats.Dirs++
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		written, err := copyFileWithInfo(path, destPath, info)
		if err != nil {
			return err
		}
		stats.Files++
		stats.Bytes += written
		return nil
	})

	return stats, err
}
