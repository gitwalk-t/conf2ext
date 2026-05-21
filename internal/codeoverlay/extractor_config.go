package codeoverlay

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type extractorConfig struct {
	Forbidden []string `json:"forbidden"`
}

func loadForbiddenBlockIDs(path string) (map[string]struct{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read extractor config: %w", err)
	}

	var cfg extractorConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse extractor config: %w", err)
	}

	forbidden := make(map[string]struct{}, len(cfg.Forbidden))
	for _, id := range cfg.Forbidden {
		normalizedID := strings.TrimSpace(id)
		if normalizedID == "" {
			continue
		}
		forbidden[normalizedID] = struct{}{}
	}
	return forbidden, nil
}
