package codeoverlay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	SchemaVersion        = 1
	DefaultExtensionPath = "input/etalonCode"
	DefaultOutputPath    = "configs/code_overlay.json"
	DefaultConfigPath    = "configs/config.json"
)

var metadataKinds = map[string]string{
	"AccountingRegisters":         "AccountingRegister",
	"AccumulationRegisters":       "AccumulationRegister",
	"BusinessProcesses":           "BusinessProcess",
	"CalculationRegisters":        "CalculationRegister",
	"Catalogs":                    "Catalog",
	"ChartsOfAccounts":            "ChartOfAccounts",
	"ChartsOfCalculationTypes":    "ChartOfCalculationTypes",
	"ChartsOfCharacteristicTypes": "ChartOfCharacteristicTypes",
	"CommonAttributes":            "CommonAttribute",
	"CommonCommands":              "CommonCommand",
	"CommonForms":                 "CommonForm",
	"CommonModules":               "CommonModule",
	"CommonPictures":              "CommonPicture",
	"CommonTemplates":             "CommonTemplate",
	"CommandGroups":               "CommandGroup",
	"Constants":                   "Constant",
	"DataProcessors":              "DataProcessor",
	"DefinedTypes":                "DefinedType",
	"Documents":                   "Document",
	"DocumentJournals":            "DocumentJournal",
	"Enums":                       "Enum",
	"EventSubscriptions":          "EventSubscription",
	"ExchangePlans":               "ExchangePlan",
	"ExternalDataSources":         "ExternalDataSource",
	"FilterCriteria":              "FilterCriterion",
	"FunctionalOptions":           "FunctionalOption",
	"FunctionalOptionsParameters": "FunctionalOptionsParameter",
	"HTTPServices":                "HTTPService",
	"InformationRegisters":        "InformationRegister",
	"IntegrationServices":         "IntegrationService",
	"Interfaces":                  "Interface",
	"Languages":                   "Language",
	"Reports":                     "Report",
	"Roles":                       "Role",
	"ScheduledJobs":               "ScheduledJob",
	"Sequences":                   "Sequence",
	"Sessions":                    "Session",
	"SessionParameters":           "SessionParameter",
	"SettingsStorages":            "SettingsStorage",
	"StyleItems":                  "StyleItem",
	"Styles":                      "Style",
	"Subsystems":                  "Subsystem",
	"Tasks":                       "Task",
	"WebServices":                 "WebService",
	"XDTOPackages":                "XDTOPackage",
}

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
	ExtensionPath string
	OutputPath    string
	ConfigPath    string
}

type Result struct {
	Artifact      Artifact
	ExtensionPath string
	OutputPath    string
	ConfigPath    string
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

	cfg, err := config.LoadConfigE(configPath)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось загрузить конфиг проекта %s: %w", configPath, err)
	}
	allowedObjects, err := xmlutils.CollectSearchResultTemplateObjectKeys(cfg)
	if err != nil {
		return Result{}, fmt.Errorf("не удалось определить объекты из упо_SearchResult: %w", err)
	}

	log.Printf("code overlay: start extraction extension_path=%s output=%s config=%s allowed_objects=%d", extensionPath, outputPath, configPath, len(allowedObjects))

	artifact, err := Extract(extensionPath, allowedObjects)
	if err != nil {
		return Result{}, err
	}
	if err := Write(outputPath, artifact); err != nil {
		return Result{}, err
	}

	log.Printf("code overlay: extraction completed blocks=%d output=%s", len(artifact.Blocks), outputPath)
	return Result{
		Artifact:      artifact,
		ExtensionPath: extensionPath,
		OutputPath:    outputPath,
		ConfigPath:    configPath,
	}, nil
}

func Extract(extensionPath string, allowedObjects map[string]struct{}) (Artifact, error) {
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
		if !shouldKeepBlock(block, allowedObjects) {
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
	return Block{
		ID:      makeBlockID(objectKey, kind),
		Object:  objectKey,
		Kind:    kind,
		Path:    normalizedRelPath,
		Hash:    hashContent(content),
		Content: content,
	}, true, nil
}

func classifyBlock(relPath string) (objectKey, kind string, ok bool) {
	parts := strings.Split(strings.Trim(filepath.ToSlash(relPath), "/"), "/")
	if len(parts) == 2 && parts[0] == "Ext" && parts[1] == "SessionModule.bsl" {
		return "Session", "SessionModule", true
	}

	if len(parts) < 4 {
		return "", "", false
	}

	ownerKind, ok := metadataKinds[parts[0]]
	if !ok {
		return "", "", false
	}

	switch {
	case len(parts) == 4 && parts[2] == "Ext":
		switch parts[3] {
		case "ObjectModule.bsl":
			return ownerKind + "." + parts[1], "ObjectModule", true
		case "ManagerModule.bsl":
			return ownerKind + "." + parts[1], "ManagerModule", true
		case "CommandModule.bsl":
			return ownerKind + "." + parts[1], "CommandModule", true
		case "Module.bsl":
			if ownerKind == "CommonModule" {
				return ownerKind + "." + parts[1], "CommonModule", true
			}
		}
	case len(parts) == 5 && parts[2] == "Ext" && parts[3] == "Form" && parts[4] == "Module.bsl":
		return ownerKind + "." + parts[1], "FormModule", true
	case len(parts) == 6 && parts[2] == "Commands" && parts[4] == "Ext" && parts[5] == "CommandModule.bsl":
		return ownerKind + "." + parts[1] + ".Command." + parts[3], "CommandModule", true
	case len(parts) == 7 && parts[2] == "Forms" && parts[4] == "Ext" && parts[5] == "Form" && parts[6] == "Module.bsl":
		return ownerKind + "." + parts[1] + ".Form." + parts[3], "FormModule", true
	}

	return "", "", false
}

func resolvePath(path, defaultPath string) (string, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		trimmedPath = defaultPath
	}
	return config.NormalizePath(trimmedPath)
}

func makeBlockID(objectKey, kind string) string {
	return objectKey + ":" + kind
}

func normalizeContent(content string) string {
	content = strings.TrimPrefix(content, "\uFEFF")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func isSupportedModuleFilename(name string) bool {
	switch name {
	case "ObjectModule.bsl", "ManagerModule.bsl", "CommandModule.bsl", "Module.bsl", "SessionModule.bsl":
		return true
	default:
		return false
	}
}

func shouldKeepBlock(block Block, allowedObjects map[string]struct{}) bool {
	if len(allowedObjects) == 0 {
		return false
	}
	_, ok := allowedObjects[topLevelObjectKey(block.Object)]
	return ok
}

func topLevelObjectKey(object string) string {
	parts := strings.Split(strings.TrimSpace(object), ".")
	if len(parts) < 2 {
		return strings.TrimSpace(object)
	}
	return parts[0] + "." + parts[1]
}
