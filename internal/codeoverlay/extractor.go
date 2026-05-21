package codeoverlay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gitwalk-m/conf2ext/internal/config"
	xmlutils "github.com/gitwalk-m/conf2ext/internal/utils/xmlutil"
)

const (
	SchemaVersion              = 1
	DefaultExtensionPath       = "input/etalonCode"
	DefaultOutputPath          = "configs/code_overlay.json"
	DefaultConfigPath          = "configs/config.json"
	DefaultExtractorConfigPath = "cmd/extract_code_overlay/config.json"
)

type Artifact struct {
	Version int     `json:"version"`
	Blocks  []Block `json:"blocks"`
}

type Block struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Hash    string `json:"hash"`
	Content string `json:"content"`
}

type Options struct {
	ExtensionPath       string
	OutputPath          string
	ConfigPath          string
	ExtractorConfigPath string
}

type Result struct {
	Artifact            Artifact
	ExtensionPath       string
	OutputPath          string
	ConfigPath          string
	ExtractorConfigPath string
}

func Run(opts Options) (Result, error) {
	extensionPath, err := resolvePath(opts.ExtensionPath, DefaultExtensionPath)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось нормализовать путь к эталонному расширению: %w", err)
	}
	outputPath, err := resolvePath(opts.OutputPath, DefaultOutputPath)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось нормализовать путь к overlay artifact: %w", err)
	}
	configPath, err := resolvePath(opts.ConfigPath, DefaultConfigPath)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось нормализовать путь к конфигу проекта: %w", err)
	}
	extractorConfigPath, err := resolvePath(opts.ExtractorConfigPath, DefaultExtractorConfigPath)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось нормализовать путь к конфигу extractor: %w", err)
	}

	cfg, err := config.LoadConfigE(configPath)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось загрузить конфиг проекта %s: %w", configPath, err)
	}
	allowedBlockIDs, err := xmlutils.CollectSearchResultTemplateBlockIDs(cfg)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось определить объекты из упо_SearchResult: %w", err)
	}
	extractorBlockSets, err := loadExtractorBlockSets(extractorConfigPath)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось загрузить конфиг extractor %s: %w", extractorConfigPath, err)
	}
	mergeAllowedBlockIDs(allowedBlockIDs, extractorBlockSets.Included)

	log.Printf(
		"code overlay: start extraction extension_path=%s output=%s config=%s extractor_config=%s allowed_blocks=%d included_blocks=%d forbidden_blocks=%d",
		extensionPath,
		outputPath,
		configPath,
		extractorConfigPath,
		len(allowedBlockIDs),
		len(extractorBlockSets.Included),
		len(extractorBlockSets.Forbidden),
	)

	artifact, err := Extract(extensionPath, allowedBlockIDs, extractorBlockSets.Forbidden)
	if err != nil {
		return Result{}, err
	}
	if err := Write(outputPath, artifact); err != nil {
		return Result{}, err
	}

	log.Printf("code overlay: extraction completed blocks=%d output=%s", len(artifact.Blocks), outputPath)
	return Result{
		Artifact:            artifact,
		ExtensionPath:       extensionPath,
		OutputPath:          outputPath,
		ConfigPath:          configPath,
		ExtractorConfigPath: extractorConfigPath,
	}, nil
}

func Extract(extensionPath string, allowedBlockIDs, forbiddenBlockIDs map[string]struct{}) (Artifact, error) {
	resolvedExtensionPath, err := resolvePath(extensionPath, DefaultExtensionPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("не удалось нормализовать путь к эталонному расширению: %w", err)
	}

	info, err := os.Stat(resolvedExtensionPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("не удалось открыть эталонное расширение %s: %w", resolvedExtensionPath, err)
	}
	if !info.IsDir() {
		return Artifact{}, fmt.Errorf("эталонное расширение должно быть каталогом: %s", resolvedExtensionPath)
	}

	blocks := make([]Block, 0, 64)
	seenIDs := make(map[string]string)

	err = filepath.WalkDir(resolvedExtensionPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		block, ok, err := buildBlock(resolvedExtensionPath, path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if !shouldKeepBlock(block, allowedBlockIDs, forbiddenBlockIDs) {
			return nil
		}

		if existingPath, exists := seenIDs[block.ID]; exists {
			return fmt.Errorf("дублирующийся code overlay block id %q: %s и %s", block.ID, existingPath, block.Path)
		}

		seenIDs[block.ID] = block.Path
		blocks = append(blocks, block)
		return nil
	})
	if err != nil {
		return Artifact{}, fmt.Errorf("не удалось обойти XML dump %s: %w", resolvedExtensionPath, err)
	}

	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].ID != blocks[j].ID {
			return blocks[i].ID < blocks[j].ID
		}
		return blocks[i].Path < blocks[j].Path
	})

	if len(blocks) == 0 {
		log.Printf("warning: code overlay: в %s не найдено ни одного поддерживаемого кодового блока", resolvedExtensionPath)
	}

	return Artifact{
		Version: SchemaVersion,
		Blocks:  blocks,
	}, nil
}

func mergeAllowedBlockIDs(dst, src map[string]struct{}) {
	for id := range src {
		dst[id] = struct{}{}
	}
}

func Write(outputPath string, artifact Artifact) error {
	resolvedOutputPath, err := resolvePath(outputPath, DefaultOutputPath)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь к overlay artifact: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(resolvedOutputPath), 0o755); err != nil {
		return fmt.Errorf("не удалось создать каталог для overlay artifact %s: %w", resolvedOutputPath, err)
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(artifact); err != nil {
		return fmt.Errorf("не удалось сериализовать overlay artifact: %w", err)
	}

	if err := os.WriteFile(resolvedOutputPath, buffer.Bytes(), 0o644); err != nil {
		return fmt.Errorf("не удалось записать overlay artifact %s: %w", resolvedOutputPath, err)
	}
	return nil
}

func buildBlock(root, path string) (Block, bool, error) {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return Block{}, false, fmt.Errorf("не удалось вычислить относительный путь для %s: %w", path, err)
	}

	normalizedRelPath := filepath.ToSlash(filepath.Clean(relPath))
	objectKey, kind, ok := classifyBlock(normalizedRelPath)
	if !ok {
		if isSupportedModuleFilename(filepath.Base(path)) {
			log.Printf("warning: code overlay: пропущен неподдерживаемый layout модуля %s", normalizedRelPath)
		}
		return Block{}, false, nil
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return Block{}, false, fmt.Errorf("не удалось прочитать кодовый блок %s: %w", path, err)
	}

	content := normalizeContent(string(contentBytes))
	if strings.TrimSpace(content) == "" {
		log.Printf("warning: code overlay: пропущен пустой модуль %s", normalizedRelPath)
		return Block{}, false, nil
	}

	return Block{
		ID:      makeBlockID(objectKey, kind),
		Object:  objectKey,
		Kind:    kind,
		Path:    normalizedRelPath,
		Hash:    hashContent(content),
		Content: content,
	}, true, nil
}

func resolvePath(path, defaultPath string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		trimmedPath = defaultPath
	}
	return config.NormalizePath(trimmedPath)
}

func shouldKeepBlock(block Block, allowedBlockIDs, forbiddenBlockIDs map[string]struct{}) bool {
	if len(allowedBlockIDs) == 0 {
		return false
	}
	if _, ok := allowedBlockIDs[block.ID]; !ok {
		return false
	}
	if _, forbidden := forbiddenBlockIDs[block.ID]; forbidden {
		log.Printf("warning: code overlay: блок %s исключен extractor config forbidden", block.ID)
		return false
	}
	return true
}
