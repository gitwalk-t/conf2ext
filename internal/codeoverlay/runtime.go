package codeoverlay

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gitwalk-m/conf2ext/internal/config"
)

type RuntimeDiagnostics struct {
	Loaded     int      `json:"loaded"`
	Applied    int      `json:"applied"`
	Skipped    int      `json:"skipped"`
	Missing    int      `json:"missing"`
	Fallback   int      `json:"fallback"`
	Conflicted int      `json:"conflicted"`
	Warnings   []string `json:"warnings,omitempty"`
}

type generatedBlock struct {
	ID     string
	Object string
	Kind   string
	Path   string
}

func ResolveRuntimeOverlayPath(cfg *config.Configuration) (string, error) {
	basePath := ""
	overlayFile := DefaultOutputPath
	if cfg != nil {
		basePath = strings.TrimSpace(cfg.ProjectRootPath)
		if candidate := cfg.CodeOverlayFile(); candidate != "" {
			overlayFile = candidate
		}
	}

	return resolveRuntimePath(overlayFile, basePath)
}

func LoadArtifact(path string) (Artifact, error) {
	resolvedPath, err := config.NormalizePath(path)
	if err != nil {
		return Artifact{}, fmt.Errorf("не удалось нормализовать путь к overlay artifact: %w", err)
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("не удалось прочитать overlay artifact %s: %w", resolvedPath, err)
	}

	var artifact Artifact
	if err := json.Unmarshal(content, &artifact); err != nil {
		return Artifact{}, fmt.Errorf("не удалось разобрать overlay artifact %s: %w", resolvedPath, err)
	}
	if artifact.Version != SchemaVersion {
		return Artifact{}, fmt.Errorf("неподдерживаемая версия overlay artifact %s: got %d want %d", resolvedPath, artifact.Version, SchemaVersion)
	}

	return artifact, nil
}

func IndexGeneratedBlocks(rootPath string) (map[string][]generatedBlock, error) {
	resolvedRoot, err := config.NormalizePath(rootPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось нормализовать путь к generated extension: %w", err)
	}

	info, err := os.Stat(resolvedRoot)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть generated extension %s: %w", resolvedRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("generated extension должен быть каталогом: %s", resolvedRoot)
	}

	index := make(map[string][]generatedBlock)
	err = filepath.WalkDir(resolvedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !isSupportedModuleFilename(d.Name()) {
			return nil
		}

		relPath, err := filepath.Rel(resolvedRoot, path)
		if err != nil {
			return fmt.Errorf("не удалось вычислить относительный путь для %s: %w", path, err)
		}

		normalizedRelPath := filepath.ToSlash(filepath.Clean(relPath))
		objectKey, kind, ok := classifyBlock(normalizedRelPath)
		if !ok {
			log.Printf("warning: code overlay: пропущен неподдерживаемый generated layout модуля %s", normalizedRelPath)
			return nil
		}

		id := makeBlockID(objectKey, kind)
		index[id] = append(index[id], generatedBlock{
			ID:     id,
			Object: objectKey,
			Kind:   kind,
			Path:   normalizedRelPath,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось обойти generated extension %s: %w", resolvedRoot, err)
	}

	for id := range index {
		sort.Slice(index[id], func(i, j int) bool {
			return index[id][i].Path < index[id][j].Path
		})
	}

	return index, nil
}

func Apply(rootPath, overlayPath string) (RuntimeDiagnostics, error) {
	artifact, err := LoadArtifact(overlayPath)
	if err != nil {
		return RuntimeDiagnostics{}, err
	}
	return ApplyArtifact(rootPath, artifact)
}

func ApplyArtifact(rootPath string, artifact Artifact) (RuntimeDiagnostics, error) {
	resolvedRoot, err := config.NormalizePath(rootPath)
	if err != nil {
		return RuntimeDiagnostics{}, fmt.Errorf("не удалось нормализовать путь к generated extension: %w", err)
	}

	index, err := IndexGeneratedBlocks(resolvedRoot)
	if err != nil {
		return RuntimeDiagnostics{}, err
	}

	diagnostics := RuntimeDiagnostics{
		Loaded: len(artifact.Blocks),
	}

	blocksByID := make(map[string][]Block, len(artifact.Blocks))
	for _, block := range artifact.Blocks {
		blocksByID[block.ID] = append(blocksByID[block.ID], block)
	}

	ids := make([]string, 0, len(blocksByID))
	for id := range blocksByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		overlayBlocks := blocksByID[id]
		if len(overlayBlocks) > 1 {
			diagnostics.Conflicted += len(overlayBlocks)
			diagnostics.Skipped += len(overlayBlocks)
			diagnostics.Fallback++
			diagnostics.addWarning("code overlay: duplicate overlay block id " + id + ", generated content kept")
			continue
		}

		block := overlayBlocks[0]
		targets := index[id]
		switch {
		case len(targets) == 0:
			diagnostics.Missing++
			diagnostics.Skipped++
			diagnostics.Fallback++
			diagnostics.addWarning("code overlay: generated target not found for " + id + ", generated content kept")
			continue
		case len(targets) > 1:
			diagnostics.Conflicted += len(targets)
			diagnostics.Skipped++
			diagnostics.Fallback++
			diagnostics.addWarning("code overlay: multiple generated targets for " + id + ", generated content kept")
			continue
		}

		target := targets[0]
		targetPath := filepath.Join(resolvedRoot, filepath.FromSlash(target.Path))
		if err := os.WriteFile(targetPath, []byte(normalizeContent(block.Content)), 0o644); err != nil {
			return diagnostics, fmt.Errorf("не удалось применить overlay block %s к %s: %w", id, target.Path, err)
		}
		diagnostics.Applied++
	}

	return diagnostics, nil
}

func (diagnostics *RuntimeDiagnostics) addWarning(message string) {
	if diagnostics == nil {
		return
	}
	log.Printf("warning: %s", message)
	diagnostics.Warnings = append(diagnostics.Warnings, message)
}

func resolveRuntimePath(path, basePath string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return "", nil
	}

	expandedPath := os.ExpandEnv(trimmedPath)
	if filepath.IsAbs(expandedPath) {
		return filepath.Clean(expandedPath), nil
	}

	normalizedBase := strings.TrimSpace(basePath)
	if normalizedBase == "" {
		return config.NormalizePath(expandedPath)
	}

	if isRuntimeProjectRelativePath(expandedPath) {
		expandedPath = strings.TrimLeft(expandedPath, `/\`)
	}

	return filepath.Clean(filepath.Join(normalizedBase, filepath.FromSlash(expandedPath))), nil
}

func isRuntimeProjectRelativePath(path string) bool {
	if path == "" {
		return false
	}
	if filepath.VolumeName(path) != "" {
		return false
	}

	first := path[0]
	return first == '/' || first == '\\'
}
