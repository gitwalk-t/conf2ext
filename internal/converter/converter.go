package converter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/firstBitSportivnaya/files-converter/internal/config"
	"github.com/firstBitSportivnaya/files-converter/internal/export_format"
	"github.com/firstBitSportivnaya/files-converter/internal/utils/fileutil"
	xmlutil "github.com/firstBitSportivnaya/files-converter/internal/utils/xmlutil"

	v8 "github.com/v8platform/api"
	"github.com/v8platform/runner"
)

type SourceFileConverter struct{}

func (s *SourceFileConverter) Convert(cfg *config.Configuration) error {
	if err := ConvertToCfe(cfg); err != nil {
		return err
	}
	return nil
}

type CfConverter struct{}

func (c *CfConverter) Convert(cfg *config.Configuration) error {
	if err := ConvertToCfe(cfg); err != nil {
		return err
	}
	return nil
}

type TempInfoBase struct {
	Infobase     *v8.Infobase
	infobasePath string
}

func (ib *TempInfoBase) SetPath() {
	if ib.Infobase != nil {
		ib.infobasePath = strings.TrimSuffix(strings.TrimPrefix(ib.Infobase.Connect.String(), "File='"), "';")
	}
}

func (ib *TempInfoBase) GetPath() string {
	return ib.infobasePath
}

func (ib *TempInfoBase) Remove() {
	if ib.infobasePath != "" {
		removeDir(ib.infobasePath)
	}
}

func ConvertToCfe(cfg *config.Configuration) error {
	dumpInfo := config.GetDumpInfo()
	keepTemp := cfg.KeepXMLDump || os.Getenv("FILES_CONVERTER_KEEP_TMP") == "1"
	log.Printf("conversion started: input=%s output=%s conversion=%s", cfg.InputPath, cfg.OutputPath, cfg.ConversionType)
	log.Printf("conversion flags: keep_xml_dump=%t stop_after_xml_dump=%t enable_form_validation=%t keep_tmp_env=%t", cfg.KeepXMLDump, cfg.StopAfterXMLDump, cfg.IsFormValidationEnabled(), os.Getenv("FILES_CONVERTER_KEEP_TMP") == "1")

	tempRoot, err := prepareTempEnvironment(cfg)
	if err != nil {
		return err
	}

	version := v8.WithVersion(cfg.PlatformVersion)
	tmpIB, err := createTempIB()
	if err != nil {
		return err
	}
	defer func() {
		if !keepTemp {
			tmpIB.Remove()
		}
	}()

	tmpInfoBase := tmpIB.Infobase

	tmpDir := newTempDir(tempRoot, "v8_src")
	defer func() {
		if !keepTemp {
			removeDir(tmpDir)
		}
	}()

	switch cfg.ConversionType {
	case config.SrcConvert:
		log.Printf("step: copy source directory")
		if err = fileutil.CopyDir(cfg.InputPath, tmpDir); err != nil {
			return err
		}

		log.Printf("step: read format version")
		formatVersion, err := xmlutil.GetFormatVersion(tmpDir)
		if err != nil {
			return err
		}

		log.Printf("step: load export format versions")
		exportFomatVersions, err := export_format.LoadFormatVersions("")
		if err != nil {
			return err
		}

		platformVersion := exportFomatVersions[formatVersion]
		if platformVersion == "" {
			return fmt.Errorf("не найдена версия платформы для формата выгрузки %s", formatVersion)
		}

		version = v8.WithVersion(platformVersion)
	case config.CfConvert:
		log.Printf("step: load cf config")
		if err = loadCfConfig(cfg, tmpInfoBase, version); err != nil {
			return err
		}

		log.Printf("step: dump config to files")
		comDumpConfigToFiles := v8.DumpConfigToFiles(tmpDir)
		if err = v8.Run(tmpInfoBase, comDumpConfigToFiles, version); err != nil {
			return fmt.Errorf("ошибка получения исходных файлов: %w", err)
		}
	}

	if os.Getenv("FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE") == "1" {
		snapshotDir := newTempDir(tempRoot, "v8_src_before_change")
		if err = fileutil.CopyDir(tmpDir, snapshotDir); err != nil {
			return fmt.Errorf("ошибка создания снимка исходной выгрузки: %w", err)
		}
		log.Printf("source snapshot saved: %s", snapshotDir)
	}

	log.Printf("step: change files")
	if err = xmlutil.ChangeFiles(cfg, tmpDir); err != nil {
		return err
	}

	if keepTemp {
		if err = saveXMLDumpSnapshot(cfg.OutputPath, tmpDir); err != nil {
			return err
		}
	}

	if cfg.StopAfterXMLDump {
		log.Printf("step: stop after xml dump")
		return nil
	}

	extension := cfg.Extension
	if extension == "" {
		extension = dumpInfo.ConfigName
	}

	log.Printf("step: load extension config from files")
	load := v8.LoadExtensionConfigFromFiles(tmpDir, extension)
	if err = v8.Run(tmpInfoBase, load, version); err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации расширения: %w", err)
	}

	outPath, err := resolveOutputPath(cfg.OutputPath, extension, dumpInfo.Version)
	if err != nil {
		return err
	}

	log.Printf("step: dump extension cfe")
	dump := v8.DumpExtensionCfg(outPath, extension)
	if err = v8.Run(tmpInfoBase, dump, version); err != nil {
		return fmt.Errorf("ошибка при выгрузке в файл .cfe: %w", err)
	}

	fmt.Printf("файл *.cfe успешно сохранен: %s\n", outPath)

	return nil
}

func loadCfConfig(cfg *config.Configuration, tmpInfoBase *v8.Infobase, version runner.Option) error {
	comLoadCfg := v8.LoadCfg(cfg.InputPath)
	if err := v8.Run(tmpInfoBase, comLoadCfg, version); err != nil {
		return fmt.Errorf("ошибка при загрузка конфигурации из файла: %w", err)
	}
	return nil
}

func newTempDir(dir, pattern string) string {
	tempDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		log.Fatal(err)
	}
	return tempDir
}

func removeDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Fatal(err)
	}
}

func createTempIB() (*TempInfoBase, error) {
	tmpInfoBase, err := v8.CreateTempInfobase()
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании базы: %w", err)
	}

	infobase := &TempInfoBase{
		Infobase: tmpInfoBase,
	}

	infobase.SetPath()

	return infobase, nil
}

func prepareTempEnvironment(cfg *config.Configuration) (string, error) {
	base := cfg.OutputPath
	if strings.EqualFold(filepath.Ext(base), ".cfe") {
		base = filepath.Dir(base)
	}
	if base == "" {
		base = "."
	}

	tempRoot := filepath.Join(base, "_tmp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return "", fmt.Errorf("не удалось подготовить временную директорию: %w", err)
	}

	for _, name := range []string{"TMP", "TEMP"} {
		if err := os.Setenv(name, tempRoot); err != nil {
			return "", fmt.Errorf("не удалось настроить переменную %s: %w", name, err)
		}
	}

	return tempRoot, nil
}

func resolveOutputPath(outputPath, extension, version string) (string, error) {
	if outputPath == "" {
		return "", fmt.Errorf("не указан выходной путь")
	}

	if strings.EqualFold(filepath.Ext(outputPath), ".cfe") {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return "", fmt.Errorf("не удалось создать директорию для выходного файла: %w", err)
		}
		return outputPath, nil
	}

	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		return "", fmt.Errorf("не удалось создать выходную директорию: %w", err)
	}

	outputFile := extension
	if version != "" {
		outputFile += "_" + strings.ReplaceAll(version, ".", "_")
	}
	outputFile += ".cfe"

	return filepath.Join(outputPath, outputFile), nil
}

func saveXMLDumpSnapshot(outputPath, tmpDir string) error {
	base := outputPath
	if strings.EqualFold(filepath.Ext(base), ".cfe") {
		base = filepath.Dir(base)
	}
	if base == "" {
		base = "."
	}

	snapshotRoot := filepath.Join(base, "_log", "xml_dumps")
	if err := os.MkdirAll(snapshotRoot, 0o755); err != nil {
		return fmt.Errorf("не удалось создать директорию для сохранения XML-дампа: %w", err)
	}

	snapshotDir := filepath.Join(snapshotRoot, filepath.Base(tmpDir))
	if err := os.RemoveAll(snapshotDir); err != nil {
		return fmt.Errorf("не удалось очистить старый снимок XML-дампа: %w", err)
	}
	if err := fileutil.CopyDir(tmpDir, snapshotDir); err != nil {
		return fmt.Errorf("ошибка сохранения XML-дампа: %w", err)
	}

	log.Printf("source snapshot saved: %s", snapshotDir)
	return nil
}
