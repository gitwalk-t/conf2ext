package main

import (
	"log"
	"os"

	pkgconfig "github.com/firstBitSportivnaya/files-converter/pkg/config"
	xmlutils "github.com/firstBitSportivnaya/files-converter/internal/utils/xmlutil"
)

func main() {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		log.Fatalf("usage: changefiles <config-path> <xml-dir> [--resume-from-validation]")
	}

	cfg, err := pkgconfig.Load(os.Args[1])
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if len(os.Args) == 4 {
		if os.Args[3] != "--resume-from-validation" {
			log.Fatalf("unknown flag: %s", os.Args[3])
		}
		if err := xmlutils.ResumeChangeFilesFromValidation(cfg, os.Args[2]); err != nil {
			log.Fatalf("resume change files: %v", err)
		}
		return
	}

	if err := xmlutils.ChangeFiles(cfg, os.Args[2]); err != nil {
		log.Fatalf("change files: %v", err)
	}
}
