package converter

import (
	publicconfig "github.com/gitwalk-m/conf2ext/pkg/config"

	internalconverter "github.com/gitwalk-m/conf2ext/internal/converter"
)

func RunConversion(cfg *publicconfig.Configuration) error {
	return internalconverter.RunConversion(cfg)
}
