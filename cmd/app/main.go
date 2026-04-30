package main

import (
	"log"

	publicconfig "github.com/firstBitSportivnaya/files-converter/pkg/config"
	publicconverter "github.com/firstBitSportivnaya/files-converter/pkg/converter"
)

func main() {
	cfg, err := publicconfig.Load("./configs/config.json")
	if err != nil {
		log.Fatalf("не удалось загрузить конфиг: %v", err)
	}

	if err := publicconverter.RunConversion(cfg); err != nil {
		log.Fatalf("конвертация завершилась ошибкой: %v", err)
	}
}
