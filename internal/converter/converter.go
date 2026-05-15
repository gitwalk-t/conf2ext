package converter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gitwalk-m/conf2ext/internal/config"
	"github.com/gitwalk-m/conf2ext/internal/export_format"
	"github.com/gitwalk-m/conf2ext/internal/utils/fileutil"
	xmlutil "github.com/gitwalk-m/conf2ext/internal/utils/xmlutil"

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

	prepareStartedAt := time.Now()
	tempRoot, err := prepareTempEnvironment(cfg)
	if err != nil {
		return err
	}
	logStepCompleted("prepare temp environment", prepareStartedAt)

	version := v8.WithVersion(cfg.PlatformVersion)
	createIBStartedAt := time.Now()
	tmpIB, err := createTempIB()
	if err != nil {
		return err
	}
	logStepCompleted("create temp infobase", createIBStartedAt)
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
		copySourceStartedAt := time.Now()
		copyStats, err := fileutil.CopyDirWithStats(cfg.InputPath, tmpDir)
		if err != nil {
			return err
		}
		logStepCompleted("copy source directory", copySourceStartedAt, fmt.Sprintf("files=%d dirs=%d bytes=%d", copyStats.Files, copyStats.Dirs, copyStats.Bytes))

		log.Printf("step: read format version")
		readFormatStartedAt := time.Now()
		formatVersion, err := xmlutil.GetFormatVersion(tmpDir)
		if err != nil {
			return err
		}
		logStepCompleted("read format version", readFormatStartedAt, fmt.Sprintf("format=%s", formatVersion))

		log.Printf("step: load export format versions")
		loadVersionsStartedAt := time.Now()
		exportFomatVersions, err := export_format.LoadFormatVersions("")
		if err != nil {
			return err
		}
		logStepCompleted("load export format versions", loadVersionsStartedAt, fmt.Sprintf("count=%d", len(exportFomatVersions)))

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
		dumpConfigStartedAt := time.Now()
		comDumpConfigToFiles := v8.DumpConfigToFiles(tmpDir)
		if err = v8.Run(tmpInfoBase, comDumpConfigToFiles, version); err != nil {
			return fmt.Errorf("ошибка получения исходных файлов: %w", err)
		}
		logStepCompleted("dump config to files", dumpConfigStartedAt)
	}

	if os.Getenv("FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE") == "1" {
		snapshotBeforeChangeStartedAt := time.Now()
		snapshotDir := newTempDir(tempRoot, "v8_src_before_change")
		copyStats, err := fileutil.CopyDirWithStats(tmpDir, snapshotDir)
		if err != nil {
			return fmt.Errorf("ошибка создания снимка исходной выгрузки: %w", err)
		}
		log.Printf("source snapshot saved: %s", snapshotDir)
		logStepCompleted("snapshot before change", snapshotBeforeChangeStartedAt, fmt.Sprintf("files=%d dirs=%d bytes=%d", copyStats.Files, copyStats.Dirs, copyStats.Bytes))
	}

	log.Printf("step: change files")
	changeFilesStartedAt := time.Now()
	if err = xmlutil.ChangeFiles(cfg, tmpDir); err != nil {
		return err
	}
	logStepCompleted("change files", changeFilesStartedAt)

	if keepTemp {
		saveDumpSnapshotStartedAt := time.Now()
		if err = saveXMLDumpSnapshot(cfg.OutputPath, tmpDir); err != nil {
			return err
		}
		logStepCompleted("save XML dump snapshot", saveDumpSnapshotStartedAt)
	}

	if cfg.StopAfterXMLDump {
		log.Printf("step: stop after xml dump")
		return nil
	}

	extension := cfg.ExtensionName()
	if extension == "" {
		extension = dumpInfo.ConfigName
	}

	log.Printf("step: load extension config from files")
	loadExtensionStartedAt := time.Now()
	load := v8.LoadExtensionConfigFromFiles(tmpDir, extension)
	if err = v8.Run(tmpInfoBase, load, version); err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации расширения: %w", err)
	}
	logStepCompleted("load extension config from files", loadExtensionStartedAt)

	outPath, err := resolveOutputPath(cfg.OutputPath, extension, dumpInfo.Version)
	if err != nil {
		return err
	}

	log.Printf("step: dump extension cfe")
	dumpExtensionStartedAt := time.Now()
	dump := v8.DumpExtensionCfg(outPath, extension)
	if err = v8.Run(tmpInfoBase, dump, version); err != nil {
		return fmt.Errorf("ошибка при выгрузке в файл .cfe: %w", err)
	}
	logStepCompleted("dump extension cfe", dumpExtensionStartedAt)

	fmt.Printf("файл *.cfe успешно сохранен: %s\n", outPath)

	return nil
}

func logStepCompleted(step string, startedAt time.Time, details ...string) {
	duration := time.Since(startedAt)
	if len(details) > 0 && strings.TrimSpace(details[0]) != "" {
		log.Printf("step completed: %s duration=%s %s", step, duration, details[0])
		return
	}
	log.Printf("step completed: %s duration=%s", step, duration)
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
	copyStats, err := fileutil.CopyDirWithStats(tmpDir, snapshotDir)
	if err != nil {
		return fmt.Errorf("ошибка сохранения XML-дампа: %w", err)
	}

	log.Printf("source snapshot saved: %s", snapshotDir)
	log.Printf("xml dump snapshot stats: files=%d dirs=%d bytes=%d", copyStats.Files, copyStats.Dirs, copyStats.Bytes)
	return nil
}
