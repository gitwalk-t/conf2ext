package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	publicconfig "github.com/firstBitSportivnaya/files-converter/pkg/config"
	publicconverter "github.com/firstBitSportivnaya/files-converter/pkg/converter"
)

var loadConfig = publicconfig.Load
var runConversion = publicconverter.RunConversion

type options struct {
	configPath  string
	checkConfig bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		return 1
	}

	loadStartedAt := time.Now()
	cfg, err := loadConfig(opts.configPath)
	loadDuration := time.Since(loadStartedAt)
	if err != nil {
		fmt.Fprintf(stderr, "не удалось загрузить конфиг: %v\n", err)
		fmt.Fprintf(stderr, "config load duration: %s\n", loadDuration)
		return 1
	}

	if opts.checkConfig {
		fmt.Fprintf(stdout, "конфиг валиден: %s\n", opts.configPath)
		fmt.Fprintf(stdout, "config load duration: %s\n", loadDuration)
		return 0
	}

	conversionStartedAt := time.Now()
	err = runConversion(cfg)
	conversionDuration := time.Since(conversionStartedAt)
	if err != nil {
		fmt.Fprintf(stderr, "конвертация завершилась ошибкой: %v\n", err)
		fmt.Fprintf(stderr, "config load duration: %s\n", loadDuration)
		fmt.Fprintf(stderr, "conversion duration: %s\n", conversionDuration)
		return 1
	}

	fmt.Fprintf(stdout, "config load duration: %s\n", loadDuration)
	fmt.Fprintf(stdout, "conversion duration: %s\n", conversionDuration)
	return 0
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var opts options

	flags := flag.NewFlagSet("app", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.configPath, "config", "./configs/config.json", "path to config")
	flags.StringVar(&opts.configPath, "c", "./configs/config.json", "path to config")
	flags.BoolVar(&opts.checkConfig, "check-config", false, "load config and exit without conversion")

	err := flags.Parse(args)
	return opts, err
}
