package codeoverlay

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExtractBuildsExpectedBlocks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Catalogs/ТестКаталог/Ext/ObjectModule.bsl", "Процедура Объект()\r\nКонецПроцедуры\r\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Ext/ManagerModule.bsl", "Функция Менеджер()\r\n\tВозврат 1;\r\nКонецФункции\r\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Commands/Открыть/Ext/CommandModule.bsl", "Процедура Команда()\nКонецПроцедуры\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Forms/ФормаСписка/Ext/Form/Module.bsl", "\uFEFFПроцедура Форма()\rКонецПроцедуры\r")
	writeFile(t, root, "CommonCommands/ОбщаяКоманда/Ext/CommandModule.bsl", "Процедура ОбщаяКоманда()\nКонецПроцедуры\n")
	writeFile(t, root, "CommonForms/ОбщаяФорма/Ext/Form/Module.bsl", "Процедура ОбщаяФорма()\nКонецПроцедуры\n")
	writeFile(t, root, "CommonModules/ОбщийМодуль/Ext/Module.bsl", "Процедура ОбщийМодуль()\nКонецПроцедуры\n")
	writeFile(t, root, "Ext/SessionModule.bsl", "Процедура ПриНачалеРаботыСистемы()\nКонецПроцедуры\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Ext/ValueManagerModule.bsl", "ignored")

	artifact, err := Extract(root, map[string]struct{}{
		"Catalog.ТестКаталог:ManagerModule":                 {},
		"Catalog.ТестКаталог:ObjectModule":                  {},
		"Catalog.ТестКаталог.Command.Открыть:CommandModule": {},
		"Catalog.ТестКаталог.Form.ФормаСписка:FormModule":   {},
		"CommonCommand.ОбщаяКоманда:CommandModule":          {},
		"CommonForm.ОбщаяФорма:FormModule":                  {},
		"CommonModule.ОбщийМодуль:CommonModule":             {},
		"Session:SessionModule":                             {},
	}, nil)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	gotIDs := make([]string, 0, len(artifact.Blocks))
	for _, block := range artifact.Blocks {
		gotIDs = append(gotIDs, block.ID)
	}

	wantIDs := []string{
		"Catalog.ТестКаталог.Command.Открыть:CommandModule",
		"Catalog.ТестКаталог.Form.ФормаСписка:FormModule",
		"Catalog.ТестКаталог:ManagerModule",
		"Catalog.ТестКаталог:ObjectModule",
		"CommonCommand.ОбщаяКоманда:CommandModule",
		"CommonForm.ОбщаяФорма:FormModule",
		"CommonModule.ОбщийМодуль:CommonModule",
		"Session:SessionModule",
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected block ids:\n got %#v\nwant %#v", gotIDs, wantIDs)
	}

	formBlock := artifact.Blocks[1]
	if formBlock.Object != "Catalog.ТестКаталог.Form.ФормаСписка" {
		t.Fatalf("unexpected form object key: %q", formBlock.Object)
	}
	if formBlock.Path != "Catalogs/ТестКаталог/Forms/ФормаСписка/Ext/Form/Module.bsl" {
		t.Fatalf("unexpected form path: %q", formBlock.Path)
	}
	if formBlock.Content != "Процедура Форма()\nКонецПроцедуры\n" {
		t.Fatalf("unexpected normalized form content: %q", formBlock.Content)
	}
	if formBlock.Hash != hashContent(formBlock.Content) {
		t.Fatalf("unexpected form hash: got %q want %q", formBlock.Hash, hashContent(formBlock.Content))
	}

	sessionBlock := artifact.Blocks[len(artifact.Blocks)-1]
	if sessionBlock.Path != "Ext/SessionModule.bsl" {
		t.Fatalf("unexpected session module path: %q", sessionBlock.Path)
	}
	if sessionBlock.Object != "Session" {
		t.Fatalf("unexpected session module object: %q", sessionBlock.Object)
	}
}

func TestExtractIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "CommonModules/Бета/Ext/Module.bsl", "Процедура Бета()\nКонецПроцедуры\n")
	writeFile(t, root, "Catalogs/Альфа/Forms/ФормаСписка/Ext/Form/Module.bsl", "Процедура Форма()\nКонецПроцедуры\n")
	writeFile(t, root, "Catalogs/Альфа/Ext/ObjectModule.bsl", "Процедура Альфа()\nКонецПроцедуры\n")

	allowedObjects := map[string]struct{}{
		"Catalog.Альфа:ObjectModule":                {},
		"Catalog.Альфа.Form.ФормаСписка:FormModule": {},
		"CommonModule.Бета:CommonModule":            {},
	}
	first, err := Extract(root, allowedObjects, nil)
	if err != nil {
		t.Fatalf("first Extract: %v", err)
	}
	second, err := Extract(root, allowedObjects, nil)
	if err != nil {
		t.Fatalf("second Extract: %v", err)
	}

	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first artifact: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second artifact: %v", err)
	}

	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("expected deterministic output:\nfirst=%s\nsecond=%s", firstJSON, secondJSON)
	}
}

func TestRunUsesDefaultPathsAndWritesOutput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "configs/config.json", "{\n  \"input_path\": \"/input/source\",\n  \"output_path\": \"/output/demo.cfe\",\n  \"conversion_type\": \"srcConvert\"\n}")
	writeFile(t, root, "configs/searchingTemplateText.json", "{\n  \"PM\": [\"//{PM}\"]\n}")
	writeFile(t, root, DefaultExtractorConfigPath, "{\n  \"included\": [\"SessionModule\", \"CommonModule.ВключенныйПринудительно\"],\n  \"forbidden\": [\"CommonModule.Запрещенный:CommonModule\"]\n}")
	writeFile(t, root, "input/source/CommonTemplates/упо_SearchResult/Ext/Template.txt", "{\n  \"ОбщиеМодули\": {\n    \"ОбщийМодуль\": {\n      \"ОбщийМодуль\": {\n        \"PM\": 1\n      }\n    },\n    \"НулевойМодуль\": {\n      \"ОбщийМодуль\": {\n        \"PM\": 0\n      }\n    }\n  }\n}")
	writeFile(t, root, filepath.ToSlash(filepath.Join(DefaultExtensionPath, "CommonModules", "ОбщийМодуль", "Ext", "Module.bsl")), "Процедура Тест()\nКонецПроцедуры\n")
	writeFile(t, root, filepath.ToSlash(filepath.Join(DefaultExtensionPath, "CommonModules", "ЛишнийМодуль", "Ext", "Module.bsl")), "Процедура Лишний()\nКонецПроцедуры\n")
	writeFile(t, root, filepath.ToSlash(filepath.Join(DefaultExtensionPath, "CommonModules", "НулевойМодуль", "Ext", "Module.bsl")), "Процедура Ноль()\nКонецПроцедуры\n")
	writeFile(t, root, filepath.ToSlash(filepath.Join(DefaultExtensionPath, "CommonModules", "ВключенныйПринудительно", "Ext", "Module.bsl")), "Процедура Included()\nКонецПроцедуры\n")
	writeFile(t, root, filepath.ToSlash(filepath.Join(DefaultExtensionPath, "Ext", "SessionModule.bsl")), "Процедура Сеанс()\nКонецПроцедуры\n")

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(oldWD); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()

	result, err := Run(Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.HasSuffix(filepath.ToSlash(result.ExtensionPath), filepath.ToSlash(DefaultExtensionPath)) {
		t.Fatalf("unexpected default extension path: %q", result.ExtensionPath)
	}
	if !strings.HasSuffix(filepath.ToSlash(result.OutputPath), filepath.ToSlash(DefaultOutputPath)) {
		t.Fatalf("unexpected default output path: %q", result.OutputPath)
	}
	if !strings.HasSuffix(filepath.ToSlash(result.ConfigPath), filepath.ToSlash(DefaultConfigPath)) {
		t.Fatalf("unexpected default config path: %q", result.ConfigPath)
	}
	if !strings.HasSuffix(filepath.ToSlash(result.ExtractorConfigPath), filepath.ToSlash(DefaultExtractorConfigPath)) {
		t.Fatalf("unexpected default extractor config path: %q", result.ExtractorConfigPath)
	}

	outputBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(DefaultOutputPath)))
	if err != nil {
		t.Fatalf("ReadFile output: %v", err)
	}
	if !strings.Contains(string(outputBytes), "\"CommonModule.ОбщийМодуль:CommonModule\"") {
		t.Fatalf("expected generated overlay artifact to contain common module block, got %s", outputBytes)
	}
	if strings.Contains(string(outputBytes), "\"CommonModule.ЛишнийМодуль:CommonModule\"") {
		t.Fatalf("did not expect non-template object to be exported, got %s", outputBytes)
	}
	if strings.Contains(string(outputBytes), "\"CommonModule.НулевойМодуль:CommonModule\"") {
		t.Fatalf("did not expect zero-counter object to be exported, got %s", outputBytes)
	}
	if !strings.Contains(string(outputBytes), "\"CommonModule.ВключенныйПринудительно:CommonModule\"") {
		t.Fatalf("expected forcibly included common module to be exported, got %s", outputBytes)
	}
	if !strings.Contains(string(outputBytes), "\"Session:SessionModule\"") {
		t.Fatalf("expected forcibly included session module to be exported, got %s", outputBytes)
	}
}

func TestExtractFiltersByTopLevelObjects(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Catalogs/ТестКаталог/Ext/ObjectModule.bsl", "Процедура Объект()\nКонецПроцедуры\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Forms/ФормаСписка/Ext/Form/Module.bsl", "Процедура Форма()\nКонецПроцедуры\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Forms/ЛишняяФорма/Ext/Form/Module.bsl", "Процедура Лишняя()\nКонецПроцедуры\n")
	writeFile(t, root, "CommonModules/ЛишнийМодуль/Ext/Module.bsl", "Процедура Лишний()\nКонецПроцедуры\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Ext/ManagerModule.bsl", " \r\n\t\r\n")

	artifact, err := Extract(root, map[string]struct{}{
		"Catalog.ТестКаталог:ObjectModule":                {},
		"Catalog.ТестКаталог.Form.ФормаСписка:FormModule": {},
	}, nil)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	gotIDs := make([]string, 0, len(artifact.Blocks))
	for _, block := range artifact.Blocks {
		gotIDs = append(gotIDs, block.ID)
	}

	wantIDs := []string{
		"Catalog.ТестКаталог.Form.ФормаСписка:FormModule",
		"Catalog.ТестКаталог:ObjectModule",
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected filtered block ids:\n got %#v\nwant %#v", gotIDs, wantIDs)
	}
}

func TestExtractSkipsEmptyModuleContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Catalogs/ПустойКаталог/Ext/ObjectModule.bsl", "\r\n \t \r\n")

	artifact, err := Extract(root, map[string]struct{}{
		"Catalog.ПустойКаталог:ObjectModule": {},
	}, nil)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if len(artifact.Blocks) != 0 {
		t.Fatalf("expected empty module to be skipped, got %#v", artifact.Blocks)
	}
}

func TestExtractSkipsForbiddenBlocks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "CommonModules/РазрешенныйМодуль/Ext/Module.bsl", "Процедура Разрешенный()\nКонецПроцедуры\n")
	writeFile(t, root, "CommonModules/ЗапрещенныйМодуль/Ext/Module.bsl", "Процедура Запрещенный()\nКонецПроцедуры\n")

	artifact, err := Extract(root, map[string]struct{}{
		"CommonModule.РазрешенныйМодуль:CommonModule": {},
		"CommonModule.ЗапрещенныйМодуль:CommonModule": {},
	}, map[string]struct{}{
		"CommonModule.ЗапрещенныйМодуль:CommonModule": {},
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if len(artifact.Blocks) != 1 {
		t.Fatalf("expected one allowed block, got %#v", artifact.Blocks)
	}
	if artifact.Blocks[0].ID != "CommonModule.РазрешенныйМодуль:CommonModule" {
		t.Fatalf("unexpected remaining block id: %q", artifact.Blocks[0].ID)
	}
}

func TestNormalizeConfiguredBlockIDSupportsIncludedShorthands(t *testing.T) {
	tests := map[string]string{
		"SessionModule": "Session:SessionModule",
		"CommonModule.ОбновлениеИнформационнойБазыУНФ":     "CommonModule.ОбновлениеИнформационнойБазыУНФ:CommonModule",
		"CommonModule.Тест:CommonModule":                   "CommonModule.Тест:CommonModule",
		"  CommonModule.ОбновлениеИнформационнойБазыУНФ  ": "CommonModule.ОбновлениеИнформационнойБазыУНФ:CommonModule",
	}

	for input, want := range tests {
		got, ok := normalizeConfiguredBlockID(input)
		if !ok {
			t.Fatalf("normalizeConfiguredBlockID(%q) unexpectedly returned ok=false", input)
		}
		if got != want {
			t.Fatalf("normalizeConfiguredBlockID(%q) = %q, want %q", input, got, want)
		}
	}
}

func writeFile(t *testing.T, root, relativePath, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", fullPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", fullPath, err)
	}
}
