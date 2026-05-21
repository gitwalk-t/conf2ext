package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gitwalk-m/conf2ext/internal/codeoverlay"
)

var runExtraction = codeoverlay.Run

type options struct {
	extensionPath       string
	outputPath          string
	configPath          string
	extractorConfigPath string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args, stderr)
	if err != nil {
		return 1
	}

	result, err := runExtraction(codeoverlay.Options{
		ExtensionPath:       opts.extensionPath,
		OutputPath:          opts.outputPath,
		ConfigPath:          opts.configPath,
		ExtractorConfigPath: opts.extractorConfigPath,
	})
	if err != nil {
		fmt.Fprintf(stderr, "extract code overlay failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "overlay blocks extracted: %d\n", len(result.Artifact.Blocks))
	fmt.Fprintf(stdout, "config path: %s\n", result.ConfigPath)
	fmt.Fprintf(stdout, "extractor config: %s\n", result.ExtractorConfigPath)
	fmt.Fprintf(stdout, "extension path: %s\n", result.ExtensionPath)
	fmt.Fprintf(stdout, "output: %s\n", result.OutputPath)
	return 0
}

func parseOptions(args []string, stderr io.Writer) (options, error) {
	var opts options

	flags := flag.NewFlagSet("extract_code_overlay", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.extensionPath, "extension-path", codeoverlay.DefaultExtensionPath, "path to reference extension XML dump")
	flags.StringVar(&opts.outputPath, "output", codeoverlay.DefaultOutputPath, "path to output overlay artifact")
	flags.StringVar(&opts.configPath, "config", codeoverlay.DefaultConfigPath, "path to project config")
	flags.StringVar(&opts.extractorConfigPath, "extractor-config", codeoverlay.DefaultExtractorConfigPath, "path to extractor config")

	err := flags.Parse(args)
	return opts, err
}
