package codeoverlay

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/gitwalk-m/conf2ext/internal/codeoverlayid"
)

var metadataKinds = map[string]string{
	"AccountingRegisters":         "AccountingRegister",
	"AccumulationRegisters":       "AccumulationRegister",
	"BusinessProcesses":           "BusinessProcess",
	"CalculationRegisters":        "CalculationRegister",
	"Catalogs":                    "Catalog",
	"ChartsOfAccounts":            "ChartOfAccounts",
	"ChartsOfCalculationTypes":    "ChartOfCalculationTypes",
	"ChartsOfCharacteristicTypes": "ChartOfCharacteristicTypes",
	"CommonAttributes":            "CommonAttribute",
	"CommonCommands":              "CommonCommand",
	"CommonForms":                 "CommonForm",
	"CommonModules":               "CommonModule",
	"CommonPictures":              "CommonPicture",
	"CommonTemplates":             "CommonTemplate",
	"CommandGroups":               "CommandGroup",
	"Constants":                   "Constant",
	"DataProcessors":              "DataProcessor",
	"DefinedTypes":                "DefinedType",
	"Documents":                   "Document",
	"DocumentJournals":            "DocumentJournal",
	"Enums":                       "Enum",
	"EventSubscriptions":          "EventSubscription",
	"ExchangePlans":               "ExchangePlan",
	"ExternalDataSources":         "ExternalDataSource",
	"FilterCriteria":              "FilterCriterion",
	"FunctionalOptions":           "FunctionalOption",
	"FunctionalOptionsParameters": "FunctionalOptionsParameter",
	"HTTPServices":                "HTTPService",
	"InformationRegisters":        "InformationRegister",
	"IntegrationServices":         "IntegrationService",
	"Interfaces":                  "Interface",
	"Languages":                   "Language",
	"Reports":                     "Report",
	"Roles":                       "Role",
	"ScheduledJobs":               "ScheduledJob",
	"Sequences":                   "Sequence",
	"Sessions":                    "Session",
	"SessionParameters":           "SessionParameter",
	"SettingsStorages":            "SettingsStorage",
	"StyleItems":                  "StyleItem",
	"Styles":                      "Style",
	"Subsystems":                  "Subsystem",
	"Tasks":                       "Task",
	"WebServices":                 "WebService",
	"XDTOPackages":                "XDTOPackage",
}

func classifyBlock(relPath string) (objectKey, kind string, ok bool) {
	parts := strings.Split(strings.Trim(filepath.ToSlash(relPath), "/"), "/")
	if len(parts) == 2 && parts[0] == "Ext" && parts[1] == "SessionModule.bsl" {
		return "Session", "SessionModule", true
	}

	if len(parts) < 4 {
		return "", "", false
	}

	ownerKind, ok := metadataKinds[parts[0]]
	if !ok {
		return "", "", false
	}

	switch {
	case len(parts) == 4 && parts[2] == "Ext":
		switch parts[3] {
		case "ObjectModule.bsl":
			return ownerKind + "." + parts[1], "ObjectModule", true
		case "ManagerModule.bsl":
			return ownerKind + "." + parts[1], "ManagerModule", true
		case "CommandModule.bsl":
			return ownerKind + "." + parts[1], "CommandModule", true
		case "Module.bsl":
			if ownerKind == "CommonModule" {
				return ownerKind + "." + parts[1], "CommonModule", true
			}
		}
	case len(parts) == 5 && parts[2] == "Ext" && parts[3] == "Form" && parts[4] == "Module.bsl":
		return ownerKind + "." + parts[1], "FormModule", true
	case len(parts) == 6 && parts[2] == "Commands" && parts[4] == "Ext" && parts[5] == "CommandModule.bsl":
		return ownerKind + "." + parts[1] + ".Command." + parts[3], "CommandModule", true
	case len(parts) == 7 && parts[2] == "Forms" && parts[4] == "Ext" && parts[5] == "Form" && parts[6] == "Module.bsl":
		return ownerKind + "." + parts[1] + ".Form." + parts[3], "FormModule", true
	}

	return "", "", false
}

func makeBlockID(objectKey, kind string) string {
	return codeoverlayid.Make(objectKey, kind)
}

func normalizeContent(content string) string {
	content = strings.TrimPrefix(content, "\uFEFF")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func isSupportedModuleFilename(name string) bool {
	switch name {
	case "ObjectModule.bsl", "ManagerModule.bsl", "CommandModule.bsl", "Module.bsl", "SessionModule.bsl":
		return true
	default:
		return false
	}
}
