package converter

import (
	publicconfig "github.com/firstBitSportivnaya/files-converter/pkg/config"

	internalconverter "github.com/firstBitSportivnaya/files-converter/internal/converter"
)

func RunConversion(cfg *publicconfig.Configuration) error {
	return internalconverter.RunConversion(cfg)
}
