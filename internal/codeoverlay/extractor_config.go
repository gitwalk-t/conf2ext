package codeoverlay

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gitwalk-m/conf2ext/internal/codeoverlayid"
)

type extractorConfig struct {
	Forbidden []string `json:"forbidden"`
	Included  []string `json:"included"`
}

type extractorBlockSets struct {
	Forbidden map[string]struct{}
	Included  map[string]struct{}
}

func loadExtractorBlockSets(path string) (extractorBlockSets, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return extractorBlockSets{}, fmt.Errorf("read extractor config: %w", err)
	}

	var cfg extractorConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return extractorBlockSets{}, fmt.Errorf("parse extractor config: %w", err)
	}

	forbidden, err := normalizeConfiguredBlockSet("forbidden", cfg.Forbidden)
	if err != nil {
		return extractorBlockSets{}, err
	}
	included, err := normalizeConfiguredBlockSet("included", cfg.Included)
	if err != nil {
		return extractorBlockSets{}, err
	}

	return extractorBlockSets{
		Forbidden: forbidden,
		Included:  included,
	}, nil
}

func normalizeConfiguredBlockSet(field string, values []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(values))
	for index, value := range values {
		normalizedID, err := normalizeConfiguredBlockID(value)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", field, index, err)
		}
		result[normalizedID] = struct{}{}
	}
	return result, nil
}

func normalizeConfiguredBlockID(value string) (string, error) {
	return codeoverlayid.NormalizeConfigured(value)
}
