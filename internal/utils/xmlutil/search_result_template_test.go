package xmlutils

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/gitwalk-m/conf2ext/internal/config"
)

func TestCollectSearchResultTemplateBlockIDsKeepsOnlyPositiveCounters(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "searchingTemplateText.json"), []byte(`{"PM":["//{PM}"],"EPM":["//{EPM}"],"НеМодуль":["//{НеМодуль}"]}`), 0o644); err != nil {
		t.Fatalf("write markers: %v", err)
	}

	inputDir := filepath.Join(root, "input", "source")
	templateDir := filepath.Join(inputDir, "CommonTemplates", "упо_SearchResult", "Ext")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	template := `{
  "Справочники": {
    "Тест": {
      "МодульФормыФормаВыбора": {
        "PM": 1,
        "EPM": 0
      },
      "МодульМенеджера": {
        "PM": 0,
        "EPM": 0
      }
    }
  },
  "ОбщиеМодули": {
    "ВызовОнлайнПоддержки": {
      "ОбщийМодуль": {
        "PM": 0,
        "EPM": 0,
        "НеМодуль": 0
      }
    }
  }
}`
	if err := os.WriteFile(filepath.Join(templateDir, "Template.txt"), []byte(template), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	blockIDs, err := CollectSearchResultTemplateBlockIDs(&config.Configuration{
		ConfigPath: configPath,
		InputPath:  inputDir,
	})
	if err != nil {
		t.Fatalf("CollectSearchResultTemplateBlockIDs: %v", err)
	}

	got := make([]string, 0, len(blockIDs))
	for id := range blockIDs {
		got = append(got, id)
	}
	slices.Sort(got)
	want := []string{"Catalog.Тест.Form.ФормаВыбора:FormModule"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected block ids:\n got %#v\nwant %#v", got, want)
	}
}

func TestCollectSearchResultTemplateBlockIDsBuildsCommandIDs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "searchingTemplateText.json"), []byte(`{"PM":["//{PM}"]}`), 0o644); err != nil {
		t.Fatalf("write markers: %v", err)
	}

	inputDir := filepath.Join(root, "input", "source")
	templateDir := filepath.Join(inputDir, "CommonTemplates", "упо_SearchResult", "Ext")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	template := `{
  "Обработки": {
    "ТестОбработка": {
      "МодульКомандыВыполнить": {
        "PM": 2
      }
    }
  },
  "ОбщиеКоманды": {
    "ОбщаяКоманда": {
      "МодульКоманды": {
        "PM": 1
      }
    }
  }
}`
	if err := os.WriteFile(filepath.Join(templateDir, "Template.txt"), []byte(template), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	blockIDs, err := CollectSearchResultTemplateBlockIDs(&config.Configuration{
		ConfigPath: configPath,
		InputPath:  inputDir,
	})
	if err != nil {
		t.Fatalf("CollectSearchResultTemplateBlockIDs: %v", err)
	}

	got := make([]string, 0, len(blockIDs))
	for id := range blockIDs {
		got = append(got, id)
	}
	slices.Sort(got)
	want := []string{
		"CommonCommand.ОбщаяКоманда:CommandModule",
		"DataProcessor.ТестОбработка.Command.Выполнить:CommandModule",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected block ids:\n got %#v\nwant %#v", got, want)
	}
}
