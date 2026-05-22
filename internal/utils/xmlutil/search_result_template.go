package xmlutils

import (
	"fmt"
	"strings"

	"github.com/gitwalk-m/conf2ext/internal/codeoverlayid"
	"github.com/gitwalk-m/conf2ext/internal/config"
)

// CollectSearchResultTemplateBlockIDs returns exact code block identifiers
// requested by the SearchResult template for the current project config.
// Only places with at least one configured marker group whose counter is > 0
// are included.
func CollectSearchResultTemplateBlockIDs(cfg *config.Configuration) (map[string]struct{}, error) {
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

	for objectKey, places := range placeRequests {
		for _, place := range places {
			blockID, ok := searchResultPlaceToBlockID(objectKey, place.Place)
			if !ok {
				continue
			}
			result[blockID] = struct{}{}
		}
	}

	return result, nil
}

func searchResultPlaceToBlockID(objectKey, place string) (string, bool) {
	trimmedObjectKey := strings.TrimSpace(objectKey)
	trimmedPlace := strings.TrimSpace(place)
	if trimmedObjectKey == "" || trimmedPlace == "" {
		return "", false
	}

	switch {
	case trimmedPlace == "ОбщийМодуль":
		return codeoverlayid.Make(trimmedObjectKey, "CommonModule"), true
	case trimmedPlace == "МодульМенеджера":
		return codeoverlayid.Make(trimmedObjectKey, "ManagerModule"), true
	case trimmedPlace == "МодульОбъекта":
		return codeoverlayid.Make(trimmedObjectKey, "ObjectModule"), true
	case trimmedPlace == "МодульКоманды":
		if strings.HasPrefix(trimmedObjectKey, "CommonCommand.") {
			return codeoverlayid.Make(trimmedObjectKey, "CommandModule"), true
		}
		return "", false
	case strings.HasPrefix(trimmedPlace, "МодульФормы"):
		formName := strings.TrimSpace(strings.TrimPrefix(trimmedPlace, "МодульФормы"))
		if formName == "" {
			return "", false
		}
		return codeoverlayid.Make(trimmedObjectKey+".Form."+formName, "FormModule"), true
	case strings.HasPrefix(trimmedPlace, "МодульКоманды"):
		commandName := strings.TrimSpace(strings.TrimPrefix(trimmedPlace, "МодульКоманды"))
		if commandName == "" {
			return "", false
		}
		if strings.HasPrefix(trimmedObjectKey, "CommonCommand.") {
			return codeoverlayid.Make(trimmedObjectKey, "CommandModule"), true
		}
		return codeoverlayid.Make(trimmedObjectKey+".Command."+commandName, "CommandModule"), true
	default:
		return "", false
	}
}
