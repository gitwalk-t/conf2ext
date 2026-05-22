package codeoverlay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gitwalk-m/conf2ext/internal/config"
)

func TestResolveRuntimeOverlayPathUsesProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	cfg := &config.Configuration{
		ProjectRootPath: projectRoot,
		CodeOverlay: config.CodeOverlayConfig{
			Enabled:     true,
			OverlayFile: "configs/code_overlay.json",
		},
	}

	got, err := ResolveRuntimeOverlayPath(cfg)
	if err != nil {
		t.Fatalf("ResolveRuntimeOverlayPath: %v", err)
	}

	want := filepath.Join(projectRoot, "configs", "code_overlay.json")
	if got != want {
		t.Fatalf("unexpected overlay path: got %q want %q", got, want)
	}
}

func TestLoadArtifact(t *testing.T) {
	root := t.TempDir()
	overlayPath := filepath.Join(root, "code_overlay.json")
	writeFile(t, root, "code_overlay.json", "{\n  \"version\": 1,\n  \"blocks\": [\n    {\n      \"id\": \"CommonModule.Тест:CommonModule\",\n      \"object\": \"CommonModule.Тест\",\n      \"kind\": \"CommonModule\",\n      \"path\": \"CommonModules/Тест/Ext/Module.bsl\",\n      \"hash\": \"abc\",\n      \"content\": \"Процедура Тест()\\nКонецПроцедуры\\n\"\n    }\n  ]\n}")

	artifact, err := LoadArtifact(overlayPath)
	if err != nil {
		t.Fatalf("LoadArtifact: %v", err)
	}

	if artifact.Version != SchemaVersion {
		t.Fatalf("unexpected artifact version: got %d want %d", artifact.Version, SchemaVersion)
	}
	if len(artifact.Blocks) != 1 {
		t.Fatalf("unexpected blocks count: got %d", len(artifact.Blocks))
	}
	if artifact.Blocks[0].ID != "CommonModule.Тест:CommonModule" {
		t.Fatalf("unexpected block id: %q", artifact.Blocks[0].ID)
	}
}

func TestIndexGeneratedBlocks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Catalogs/ТестКаталог/Ext/ObjectModule.bsl", "// object\n")
	writeFile(t, root, "Catalogs/ТестКаталог/Forms/ФормаСписка/Ext/Form/Module.bsl", "// form\n")
	writeFile(t, root, "CommonModules/Тест/Ext/Module.bsl", "// common\n")
	writeFile(t, root, "Ext/SessionModule.bsl", "// session\n")

	index, err := IndexGeneratedBlocks(root)
	if err != nil {
		t.Fatalf("IndexGeneratedBlocks: %v", err)
	}

	for _, id := range []string{
		"Catalog.ТестКаталог:ObjectModule",
		"Catalog.ТестКаталог.Form.ФормаСписка:FormModule",
		"CommonModule.Тест:CommonModule",
		"Session:SessionModule",
	} {
		blocks := index[id]
		if len(blocks) != 1 {
			t.Fatalf("expected one generated block for %s, got %#v", id, blocks)
		}
	}
}

func TestApplyArtifactReplacesSupportedKinds(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"Catalogs/ТестКаталог/Ext/ObjectModule.bsl":                   "// old object\n",
		"Catalogs/ТестКаталог/Ext/ManagerModule.bsl":                  "// old manager\n",
		"Catalogs/ТестКаталог/Forms/ФормаСписка/Ext/Form/Module.bsl":  "// old form\n",
		"Catalogs/ТестКаталог/Commands/Открыть/Ext/CommandModule.bsl": "// old object command\n",
		"CommonModules/ТестОбщийМодуль/Ext/Module.bsl":                "// old common\n",
		"Ext/SessionModule.bsl": "// old session\n",
	}
	for path, content := range files {
		writeFile(t, root, path, content)
	}

	artifact := Artifact{
		Version: SchemaVersion,
		Blocks: []Block{
			{ID: "Catalog.ТестКаталог:ObjectModule", Object: "Catalog.ТестКаталог", Kind: "ObjectModule", Content: "Процедура Объект()\nКонецПроцедуры\n"},
			{ID: "Catalog.ТестКаталог:ManagerModule", Object: "Catalog.ТестКаталог", Kind: "ManagerModule", Content: "Процедура Менеджер()\nКонецПроцедуры\n"},
			{ID: "Catalog.ТестКаталог.Form.ФормаСписка:FormModule", Object: "Catalog.ТестКаталог.Form.ФормаСписка", Kind: "FormModule", Content: "Процедура Форма()\nКонецПроцедуры\n"},
			{ID: "Catalog.ТестКаталог.Command.Открыть:CommandModule", Object: "Catalog.ТестКаталог.Command.Открыть", Kind: "CommandModule", Content: "Процедура Команда()\nКонецПроцедуры\n"},
			{ID: "CommonModule.ТестОбщийМодуль:CommonModule", Object: "CommonModule.ТестОбщийМодуль", Kind: "CommonModule", Content: "Процедура Общий()\nКонецПроцедуры\n"},
			{ID: "Session:SessionModule", Object: "Session", Kind: "SessionModule", Content: "Процедура Сеанс()\nКонецПроцедуры\n"},
		},
	}

	diagnostics, err := ApplyArtifact(root, artifact)
	if err != nil {
		t.Fatalf("ApplyArtifact: %v", err)
	}

	if diagnostics.Loaded != 6 || diagnostics.Applied != 6 || diagnostics.Skipped != 0 || diagnostics.Missing != 0 || diagnostics.Fallback != 0 || diagnostics.Conflicted != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}

	assertFileContent(t, root, "Catalogs/ТестКаталог/Ext/ObjectModule.bsl", "Процедура Объект()\nКонецПроцедуры\n")
	assertFileContent(t, root, "Catalogs/ТестКаталог/Ext/ManagerModule.bsl", "Процедура Менеджер()\nКонецПроцедуры\n")
	assertFileContent(t, root, "Catalogs/ТестКаталог/Forms/ФормаСписка/Ext/Form/Module.bsl", "Процедура Форма()\nКонецПроцедуры\n")
	assertFileContent(t, root, "Catalogs/ТестКаталог/Commands/Открыть/Ext/CommandModule.bsl", "Процедура Команда()\nКонецПроцедуры\n")
	assertFileContent(t, root, "CommonModules/ТестОбщийМодуль/Ext/Module.bsl", "Процедура Общий()\nКонецПроцедуры\n")
	assertFileContent(t, root, "Ext/SessionModule.bsl", "Процедура Сеанс()\nКонецПроцедуры\n")
}

func TestApplyArtifactMissingTargetDoesNotFail(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "CommonModules/Тест/Ext/Module.bsl", "// generated\n")

	diagnostics, err := ApplyArtifact(root, Artifact{
		Version: SchemaVersion,
		Blocks: []Block{
			{ID: "CommonModule.Отсутствует:CommonModule", Object: "CommonModule.Отсутствует", Kind: "CommonModule", Content: "Процедура Отсутствует()\nКонецПроцедуры\n"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyArtifact: %v", err)
	}

	if diagnostics.Missing != 1 || diagnostics.Skipped != 1 || diagnostics.Fallback != 1 || diagnostics.Applied != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(diagnostics.Warnings) != 1 {
		t.Fatalf("expected one warning, got %#v", diagnostics.Warnings)
	}
	assertFileContent(t, root, "CommonModules/Тест/Ext/Module.bsl", "// generated\n")
}

func TestApplyArtifactSessionModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Ext/SessionModule.bsl", "// generated session\n")

	diagnostics, err := ApplyArtifact(root, Artifact{
		Version: SchemaVersion,
		Blocks: []Block{
			{ID: "Session:SessionModule", Object: "Session", Kind: "SessionModule", Content: "Процедура ПриНачалеРаботыСистемы()\nКонецПроцедуры\n"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyArtifact: %v", err)
	}

	if diagnostics.Applied != 1 {
		t.Fatalf("expected session module overlay to be applied, got %#v", diagnostics)
	}
	assertFileContent(t, root, "Ext/SessionModule.bsl", "Процедура ПриНачалеРаботыСистемы()\nКонецПроцедуры\n")
}

func TestApplyNormalizesLegacyShorthandOverlayIDs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Ext/SessionModule.bsl", "// generated session\n")
	writeFile(t, root, "CommonModules/ТестОбщийМодуль/Ext/Module.bsl", "// generated common\n")
	writeFile(t, root, "DataProcessors/ТестОбработка/Forms/ФормаСписка/Ext/Form/Module.bsl", "// generated form\n")
	writeFile(t, root, "DataProcessors/ТестОбработка/Commands/Выполнить/Ext/CommandModule.bsl", "// generated command\n")

	overlayRoot := t.TempDir()
	overlayPath := filepath.Join(overlayRoot, "legacy-overlay.json")
	writeFile(t, overlayRoot, "legacy-overlay.json", "{\n  \"version\": 1,\n  \"blocks\": [\n    {\n      \"id\": \"SessionModule\",\n      \"object\": \"Session\",\n      \"kind\": \"SessionModule\",\n      \"path\": \"Ext/SessionModule.bsl\",\n      \"hash\": \"\",\n      \"content\": \"Процедура Сеанс()\\nКонецПроцедуры\\n\"\n    },\n    {\n      \"id\": \"CommonModule.ТестОбщийМодуль\",\n      \"object\": \"CommonModule.ТестОбщийМодуль\",\n      \"kind\": \"CommonModule\",\n      \"path\": \"CommonModules/ТестОбщийМодуль/Ext/Module.bsl\",\n      \"hash\": \"\",\n      \"content\": \"Процедура Общий()\\nКонецПроцедуры\\n\"\n    },\n    {\n      \"id\": \"DataProcessor.ТестОбработка.Form.ФормаСписка\",\n      \"object\": \"DataProcessor.ТестОбработка.Form.ФормаСписка\",\n      \"kind\": \"FormModule\",\n      \"path\": \"DataProcessors/ТестОбработка/Forms/ФормаСписка/Ext/Form/Module.bsl\",\n      \"hash\": \"\",\n      \"content\": \"Процедура Форма()\\nКонецПроцедуры\\n\"\n    },\n    {\n      \"id\": \"DataProcessor.ТестОбработка.Command.Выполнить\",\n      \"object\": \"DataProcessor.ТестОбработка.Command.Выполнить\",\n      \"kind\": \"CommandModule\",\n      \"path\": \"DataProcessors/ТестОбработка/Commands/Выполнить/Ext/CommandModule.bsl\",\n      \"hash\": \"\",\n      \"content\": \"Процедура Команда()\\nКонецПроцедуры\\n\"\n    }\n  ]\n}")

	diagnostics, err := Apply(root, overlayPath)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if diagnostics.Applied != 4 || diagnostics.Skipped != 0 || diagnostics.Missing != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	assertFileContent(t, root, "Ext/SessionModule.bsl", "Процедура Сеанс()\nКонецПроцедуры\n")
	assertFileContent(t, root, "CommonModules/ТестОбщийМодуль/Ext/Module.bsl", "Процедура Общий()\nКонецПроцедуры\n")
	assertFileContent(t, root, "DataProcessors/ТестОбработка/Forms/ФормаСписка/Ext/Form/Module.bsl", "Процедура Форма()\nКонецПроцедуры\n")
	assertFileContent(t, root, "DataProcessors/ТестОбработка/Commands/Выполнить/Ext/CommandModule.bsl", "Процедура Команда()\nКонецПроцедуры\n")
}

func TestApplyArtifactHandlesDuplicateOverlayIDs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "CommonModules/Тест/Ext/Module.bsl", "// generated\n")

	diagnostics, err := ApplyArtifact(root, Artifact{
		Version: SchemaVersion,
		Blocks: []Block{
			{ID: "CommonModule.Тест:CommonModule", Object: "CommonModule.Тест", Kind: "CommonModule", Content: "Процедура Один()\nКонецПроцедуры\n"},
			{ID: "CommonModule.Тест:CommonModule", Object: "CommonModule.Тест", Kind: "CommonModule", Content: "Процедура Два()\nКонецПроцедуры\n"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyArtifact: %v", err)
	}

	if diagnostics.Loaded != 2 || diagnostics.Conflicted != 2 || diagnostics.Skipped != 2 || diagnostics.Fallback != 1 || diagnostics.Applied != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	assertFileContent(t, root, "CommonModules/Тест/Ext/Module.bsl", "// generated\n")
}

func TestApplyDoesNotReadInputEtalonCode(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "CommonModules/Тест/Ext/Module.bsl", "// generated\n")

	overlayRoot := t.TempDir()
	overlayPath := filepath.Join(overlayRoot, "runtime-overlay.json")
	writeFile(t, overlayRoot, "runtime-overlay.json", "{\n  \"version\": 1,\n  \"blocks\": [\n    {\n      \"id\": \"CommonModule.Тест:CommonModule\",\n      \"object\": \"CommonModule.Тест\",\n      \"kind\": \"CommonModule\",\n      \"path\": \"CommonModules/Тест/Ext/Module.bsl\",\n      \"hash\": \"\",\n      \"content\": \"Процедура Overlay()\\nКонецПроцедуры\\n\"\n    }\n  ]\n}")

	diagnostics, err := Apply(root, overlayPath)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if diagnostics.Applied != 1 {
		t.Fatalf("expected overlay to be applied without reference extension path, got %#v", diagnostics)
	}
	assertFileContent(t, root, "CommonModules/Тест/Ext/Module.bsl", "Процедура Overlay()\nКонецПроцедуры\n")
}

func assertFileContent(t *testing.T, root, relativePath, want string) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", relativePath, err)
	}
	if string(content) != want {
		t.Fatalf("unexpected content for %s:\n got %q\nwant %q", relativePath, string(content), want)
	}
}
