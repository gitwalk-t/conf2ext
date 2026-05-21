package xmlutils

import (
	"fmt"
	"strings"

	"github.com/gitwalk-m/conf2ext/internal/config"
)

// CollectSearchResultTemplateObjectKeys returns top-level metadata object keys
// that are requested by the SearchResult template for the current project config.
func CollectSearchResultTemplateObjectKeys(cfg *config.Configuration) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	if cfg == nil {
		return result, nil
	}

	if strings.TrimSpace(cfg.InputPath) == "" {
		return nil, fmt.Errorf("не задан input_path для поиска CommonTemplates/упо_SearchResult/Ext/Template.txt")
	}

	markerGroups, err := loadSearchResultMarkerGroups(cfg)
	if err != nil {
		return nil, err
	}

	placeRequests, err := collectSearchResultPlaceRequests(cfg.InputPath, markerGroups)
	if err != nil {
		return nil, err
	}

	for key := range placeRequests {
		result[key] = struct{}{}
	}

	return result, nil
}
