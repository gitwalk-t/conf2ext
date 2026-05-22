package codeoverlayid

import (
	"fmt"
	"strings"
)

var supportedKinds = map[string]struct{}{
	"ObjectModule":  {},
	"ManagerModule": {},
	"FormModule":    {},
	"CommandModule": {},
	"CommonModule":  {},
	"SessionModule": {},
}

func Make(objectKey, kind string) string {
	return strings.TrimSpace(objectKey) + ":" + strings.TrimSpace(kind)
}

func NormalizeConfigured(value string) (string, error) {
	return normalize(strings.TrimSpace(value))
}

func NormalizeArtifact(id, objectKey, kind string) (string, error) {
	normalizedFields, hasFields, err := normalizeFields(objectKey, kind)
	if err != nil {
		return "", err
	}

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		if hasFields {
			return normalizedFields, nil
		}
		return "", fmt.Errorf("empty block id")
	}

	normalizedID, err := normalize(trimmedID)
	if err != nil {
		return "", err
	}

	if hasFields && normalizedID != normalizedFields {
		return "", fmt.Errorf("block id %q conflicts with object=%q kind=%q", id, strings.TrimSpace(objectKey), strings.TrimSpace(kind))
	}

	if hasFields {
		return normalizedFields, nil
	}
	return normalizedID, nil
}

func normalizeFields(objectKey, kind string) (string, bool, error) {
	trimmedObject := strings.TrimSpace(objectKey)
	trimmedKind := strings.TrimSpace(kind)
	if trimmedObject == "" && trimmedKind == "" {
		return "", false, nil
	}
	if trimmedObject == "" || trimmedKind == "" {
		return "", false, fmt.Errorf("incomplete block identity object=%q kind=%q", objectKey, kind)
	}
	if _, ok := supportedKinds[trimmedKind]; !ok {
		return "", false, fmt.Errorf("unsupported block kind %q", trimmedKind)
	}
	return Make(trimmedObject, trimmedKind), true, nil
}

func normalize(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("empty block id")
	}

	objectKey, kind, hasKind := strings.Cut(value, ":")
	if hasKind {
		trimmedObject := strings.TrimSpace(objectKey)
		trimmedKind := strings.TrimSpace(kind)
		if trimmedObject == "" || trimmedKind == "" {
			return "", fmt.Errorf("invalid canonical block id %q", value)
		}
		if _, ok := supportedKinds[trimmedKind]; !ok {
			return "", fmt.Errorf("unsupported block kind %q", trimmedKind)
		}
		return Make(trimmedObject, trimmedKind), nil
	}

	switch {
	case value == "SessionModule":
		return Make("Session", "SessionModule"), nil
	case strings.HasPrefix(value, "CommonModule.") && len(value) > len("CommonModule."):
		return Make(value, "CommonModule"), nil
	case strings.HasPrefix(value, "CommonCommand.") && len(value) > len("CommonCommand."):
		return Make(value, "CommandModule"), nil
	case strings.Contains(value, ".Command."):
		return Make(value, "CommandModule"), nil
	case strings.HasPrefix(value, "CommonForm.") && len(value) > len("CommonForm."):
		return Make(value, "FormModule"), nil
	case strings.Contains(value, ".Form."):
		return Make(value, "FormModule"), nil
	default:
		return "", fmt.Errorf("ambiguous shorthand block id %q: use canonical <object>:<kind>", value)
	}
}
