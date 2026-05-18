package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/ilyakaznacheev/cleanenv"
)

//go:embed default.json
var defaultConfig []byte

var globalDumpInfo *DumpInfo

func init() {
	globalDumpInfo = NewDumpInfo()
}

func NewDumpInfo() *DumpInfo {
	return &DumpInfo{ConfigName: "default"}
}

func GetDumpInfo() *DumpInfo {
	return globalDumpInfo
}

func (info *DumpInfo) SetVersion(version string) {
	if version != "" {
		info.Version = "V" + version
	}
}

func (info *DumpInfo) SetConfigName(name string) {
	if name != "" {
		info.ConfigName = name
	}
}

func LoadConfigE(configPath string) (*Configuration, error) {
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	if configPath == "" {
		return nil, errors.New("не указан путь конфигурации")
	}

	resolvedConfigPath, err := NormalizePath(configPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось нормализовать путь к конфигурации: %w", err)
	}

	if _, err := os.Stat(resolvedConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("конфигурационный файл не существует: %w", err)
	}

	var cfg Configuration

	if err := cleanenv.ReadConfig(resolvedConfigPath, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
	}

	normalizeLegacyConfigFields(&cfg)

	if err := normalizeConfigPaths(&cfg, resolvedConfigPath); err != nil {
		return nil, fmt.Errorf("ошибка в обработке путей: %w", err)
	}
	if err := populateDerivedPaths(&cfg, resolvedConfigPath); err != nil {
		return nil, fmt.Errorf("ошибка в вычислении служебных путей: %w", err)
	}

	return &cfg, nil
}

func LoadConfig(configPath string) *Configuration {
	cfg, err := LoadConfigE(configPath)
	if err != nil {
		panic(err.Error())
	}
	return cfg
}

func LoadDefaultConfigE() (*Configuration, error) {
	var cfg Configuration

	if err := json.Unmarshal(defaultConfig, &cfg); err != nil {
		return nil, fmt.Errorf("не удалось прочитать конфигурацию по умолчанию: %w", err)
	}

	normalizeLegacyConfigFields(&cfg)

	if err := normalizeConfigPaths(&cfg, ""); err != nil {
		return nil, fmt.Errorf("ошибка в обработке путей: %w", err)
	}
	if err := populateDerivedPaths(&cfg, ""); err != nil {
		return nil, fmt.Errorf("ошибка в вычислении служебных путей: %w", err)
	}

	return &cfg, nil
}

func LoadDefaultConfig() *Configuration {
	cfg, err := LoadDefaultConfigE()
	if err != nil {
		panic(err.Error())
	}
	return cfg
}

func NormalizePath(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	expandedPath := os.ExpandEnv(strings.TrimSpace(input))

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return "", err
	}

	cleanPath := filepath.Clean(absPath)

	return cleanPath, nil
}

func NewElementOperation(name string, value string, operation OperationType) *ElementOperation {
	return &ElementOperation{
		ElementName: name,
		Value:       value,
		Operation:   operation,
	}
}

func MergeConfigurations(defaultCfg, cfg *Configuration) {
	if defaultCfg == nil || cfg == nil {
		return
	}

	if cfg.PlatformVersion != "" {
		defaultCfg.PlatformVersion = cfg.PlatformVersion
	}

	if cfg.Extension != "" {
		defaultCfg.Extension = cfg.Extension
	}

	if cfg.Prefix != "" {
		defaultCfg.Prefix = cfg.Prefix
	}

	if cfg.ExtensionProperties.Name != "" {
		defaultCfg.ExtensionProperties.Name = cfg.ExtensionProperties.Name
	}
	if cfg.ExtensionProperties.Prefix != "" {
		defaultCfg.ExtensionProperties.Prefix = cfg.ExtensionProperties.Prefix
	}
	if cfg.ExtensionProperties.Identifier != "" {
		defaultCfg.ExtensionProperties.Identifier = cfg.ExtensionProperties.Identifier
	}

	if len(cfg.NativePrefixes) > 0 {
		defaultCfg.NativePrefixes = append([]string{}, cfg.NativePrefixes...)
	}

	if len(cfg.IncludedNativeObjects) > 0 {
		defaultCfg.IncludedNativeObjects = append([]string{}, cfg.IncludedNativeObjects...)
	}

	if len(cfg.IncludedAdoptedStubObjects) > 0 {
		defaultCfg.IncludedAdoptedStubObjects = append([]string{}, cfg.IncludedAdoptedStubObjects...)
	}

	if len(cfg.ForbiddenAdoptedStubObjects) > 0 {
		defaultCfg.ForbiddenAdoptedStubObjects = append([]string{}, cfg.ForbiddenAdoptedStubObjects...)
	}

	if cfg.EnableFormValidation != nil {
		defaultCfg.EnableFormValidation = cfg.EnableFormValidation
	}

	if cfg.InputPath != "" {
		defaultCfg.InputPath = cfg.InputPath
	}

	if cfg.OutputPath != "" {
		defaultCfg.OutputPath = cfg.OutputPath
	}
	if cfg.Target.Base != "" {
		defaultCfg.Target.Base = cfg.Target.Base
	}
	if cfg.Target.XMLDump != "" {
		defaultCfg.Target.XMLDump = cfg.Target.XMLDump
	}

	if cfg.ConversionType != "" {
		defaultCfg.ConversionType = cfg.ConversionType
	}

	if cfg.ConfigPath != "" {
		defaultCfg.ConfigPath = cfg.ConfigPath
	}
	if cfg.ProjectRootPath != "" {
		defaultCfg.ProjectRootPath = cfg.ProjectRootPath
	}
	if cfg.BaseBindingsPath != "" {
		defaultCfg.BaseBindingsPath = cfg.BaseBindingsPath
	}
	if cfg.IdentityMapPath != "" {
		defaultCfg.IdentityMapPath = cfg.IdentityMapPath
	}

	defaultCfg.KeepXMLDump = cfg.KeepXMLDump
	defaultCfg.StopAfterXMLDump = cfg.StopAfterXMLDump
	defaultCfg.AdditionalProcessing = cfg.AdditionalProcessing

	if len(cfg.AdditionalAdoptedObjects) > 0 {
		defaultCfg.IncludedAdoptedStubObjects = append(defaultCfg.IncludedAdoptedStubObjects, cfg.AdditionalAdoptedObjects...)
		defaultCfg.AdditionalAdoptedObjects = append(defaultCfg.AdditionalAdoptedObjects, cfg.AdditionalAdoptedObjects...)
	}

	if len(cfg.IncludedObjects) > 0 {
		defaultCfg.IncludedAdoptedStubObjects = append(defaultCfg.IncludedAdoptedStubObjects, cfg.IncludedObjects...)
	}

	defaultCfg.XMLFiles = append(defaultCfg.XMLFiles, cfg.XMLFiles...)
}

func normalizeLegacyConfigFields(cfg *Configuration) {
	if cfg == nil {
		return
	}

	if cfg.ExtensionProperties.Name == "" {
		cfg.ExtensionProperties.Name = strings.TrimSpace(cfg.Extension)
	}
	if cfg.ExtensionProperties.Prefix == "" {
		cfg.ExtensionProperties.Prefix = strings.TrimSpace(cfg.Prefix)
	}
	if cfg.Extension == "" {
		cfg.Extension = cfg.ExtensionProperties.Name
	}
	if cfg.Prefix == "" {
		cfg.Prefix = cfg.ExtensionProperties.Prefix
	}

	if len(cfg.IncludedObjects) > 0 {
		cfg.IncludedAdoptedStubObjects = append(cfg.IncludedAdoptedStubObjects, cfg.IncludedObjects...)
		cfg.IncludedObjects = nil
	}

	if len(cfg.AdditionalAdoptedObjects) > 0 {
		cfg.IncludedAdoptedStubObjects = append(cfg.IncludedAdoptedStubObjects, cfg.AdditionalAdoptedObjects...)
		cfg.AdditionalAdoptedObjects = nil
	}
}

func normalizeConfigPaths(config *Configuration, configPath string) error {
	baseDir := configBaseDir(configPath)

	var err error
	config.InputPath, err = NormalizeProjectPath(config.InputPath, baseDir)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь к входному файлу: %w", err)
	}

	config.OutputPath, err = NormalizeProjectPath(config.OutputPath, baseDir)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь к выходному файлу: %w", err)
	}

	config.Target.XMLDump, err = NormalizeProjectPath(config.Target.XMLDump, baseDir)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь к XML-дампу конфигурации-приемника: %w", err)
	}

	return nil
}

func populateDerivedPaths(config *Configuration, configPath string) error {
	if config == nil {
		return nil
	}

	projectRoot := configBaseDir(configPath)
	if strings.TrimSpace(projectRoot) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		projectRoot = wd
	}

	normalizedProjectRoot, err := NormalizePath(projectRoot)
	if err != nil {
		return err
	}

	config.ConfigPath = strings.TrimSpace(configPath)
	config.ProjectRootPath = normalizedProjectRoot
	config.BaseBindingsPath = filepath.Join(normalizedProjectRoot, "configs", "base-bindings.json")
	config.IdentityMapPath = filepath.Join(normalizedProjectRoot, "output", "_state", "identity-map.json")
	return nil
}

func NormalizeProjectPath(input, baseDir string) (string, error) {
	if input == "" {
		return "", nil
	}

	expandedPath := os.ExpandEnv(strings.TrimSpace(input))
	if expandedPath == "" {
		return "", nil
	}

	if isConfigRootRelativePath(expandedPath) {
		if filepath.IsAbs(expandedPath) {
			if _, err := os.Stat(expandedPath); err == nil {
				return filepath.Clean(expandedPath), nil
			}
		}
		if baseDir == "" {
			return NormalizePath(expandedPath)
		}
		trimmed := strings.TrimLeft(expandedPath, `/\`)
		return filepath.Clean(filepath.Join(baseDir, trimmed)), nil
	}

	if filepath.IsAbs(expandedPath) {
		return filepath.Clean(expandedPath), nil
	}

	if baseDir == "" {
		return NormalizePath(expandedPath)
	}

	return filepath.Clean(filepath.Join(baseDir, expandedPath)), nil
}

func configBaseDir(configPath string) string {
	if strings.TrimSpace(configPath) == "" {
		return ""
	}

	configDir := filepath.Dir(configPath)
	if strings.EqualFold(filepath.Base(configDir), "configs") {
		return filepath.Dir(configDir)
	}

	return configDir
}

func isConfigRootRelativePath(path string) bool {
	if path == "" {
		return false
	}

	if filepath.VolumeName(path) != "" {
		return false
	}

	first := path[0]
	return first == '/' || first == '\\'
}
