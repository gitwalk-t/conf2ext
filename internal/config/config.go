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

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("конфигурационный файл не существует: %w", err)
	}

	var cfg Configuration

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %w", err)
	}

	normalizeLegacyConfigFields(&cfg)

	if err := normalizeConfigPaths(&cfg); err != nil {
		return nil, fmt.Errorf("ошибка в обработке путей: %w", err)
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

	if err := normalizeConfigPaths(&cfg); err != nil {
		return nil, fmt.Errorf("ошибка в обработке путей: %w", err)
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

	if cfg.ConversionType != "" {
		defaultCfg.ConversionType = cfg.ConversionType
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

	if len(cfg.IncludedObjects) > 0 {
		cfg.IncludedAdoptedStubObjects = append(cfg.IncludedAdoptedStubObjects, cfg.IncludedObjects...)
		cfg.IncludedObjects = nil
	}

	if len(cfg.AdditionalAdoptedObjects) > 0 {
		cfg.IncludedAdoptedStubObjects = append(cfg.IncludedAdoptedStubObjects, cfg.AdditionalAdoptedObjects...)
		cfg.AdditionalAdoptedObjects = nil
	}
}

func normalizeConfigPaths(config *Configuration) error {
	var err error
	config.InputPath, err = NormalizePath(config.InputPath)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь к входному файлу: %w", err)
	}

	config.OutputPath, err = NormalizePath(config.OutputPath)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь к выходному файлу: %w", err)
	}

	return nil
}
