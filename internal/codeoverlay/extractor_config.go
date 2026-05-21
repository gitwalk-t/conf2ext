package codeoverlay

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

	return extractorBlockSets{
		Forbidden: normalizeConfiguredBlockSet(cfg.Forbidden),
		Included:  normalizeConfiguredBlockSet(cfg.Included),
	}, nil
}

func normalizeConfiguredBlockSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalizedID, ok := normalizeConfiguredBlockID(value)
		if !ok {
			continue
		}
		result[normalizedID] = struct{}{}
	}
	return result
}

func normalizeConfiguredBlockID(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	if strings.Contains(trimmed, ":") {
		return trimmed, true
	}
	if trimmed == "SessionModule" {
		return "Session:SessionModule", true
	}
	if strings.HasPrefix(trimmed, "CommonModule.") {
		return trimmed + ":CommonModule", true
	}
	return trimmed, true
}
