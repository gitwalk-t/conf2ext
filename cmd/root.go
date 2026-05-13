package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/gitwalk-m/conf2ext/internal/config"
	publicconfig "github.com/gitwalk-m/conf2ext/pkg/config"
	publicconverter "github.com/gitwalk-m/conf2ext/pkg/converter"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "files-converter",
	Short: "A tool for converting files to the *.cfe format",
	Long: `Files Converter is a command-line application that allows you to convert files to *.cfe format.
	
There are two conversion modes available:
1. Convert from source files to *.cfe.
2. Convert from .cf file to *.cfe.

This tool simplifies the conversion process, making it easy and efficient to manage your files.`,
	Run: runMain,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Путь к конфигурационному файлу")

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

func runMain(cmd *cobra.Command, args []string) {
	effectiveConfigPath := configPath
	if effectiveConfigPath == "" {
		effectiveConfigPath = "./configs/config.json"
	}

	defaultCfg, err := publicconfig.Load(effectiveConfigPath)
	if err != nil {
		log.Fatalf("не удалось загрузить конфигурацию: %v", err)
	}

	changeXmlFiles(defaultCfg)

	runConvert(defaultCfg)
}

func changeXmlFiles(cfg *config.Configuration) {
	for _, xmlFile := range cfg.XMLFiles {
		if xmlFile.FileName == "Configuration.xml" {
			setNamePrefix(xmlFile, cfg.Prefix)
		}
	}
}

func setNamePrefix(file *config.FileOperation, prefix string) {
	for _, operation := range file.ElementOperations {
		if operation.ElementName == config.NamePrefixElement {
			operation.Value = prefix
			return
		}
	}

	element := config.NewElementOperation(config.NamePrefixElement, prefix, config.Add)
	file.ElementOperations = append(file.ElementOperations, element)
}

func runConvert(cfg *config.Configuration) {
	defer pressAnyKeyToExit()

	if err := publicconverter.RunConversion(cfg); err != nil {
		log.Fatalf("не удалось конвертировать файлы: %v", err)
	}
}

func pressAnyKeyToExit() {
	info, err := os.Stdin.Stat()
	if err != nil || (info.Mode()&os.ModeCharDevice) == 0 {
		return
	}

	fmt.Println("Press any key to exit...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
