package config

import internalconfig "github.com/firstBitSportivnaya/files-converter/internal/config"

type Configuration = internalconfig.Configuration

func Load(path string) (*Configuration, error) {
	defaultCfg, err := internalconfig.LoadDefaultConfigE()
	if err != nil {
		return nil, err
	}

	cfg, err := internalconfig.LoadConfigE(path)
	if err != nil {
		return nil, err
	}

	internalconfig.MergeConfigurations(defaultCfg, cfg)

	return defaultCfg, nil
}
