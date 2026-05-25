package xmlutils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/gitwalk-m/conf2ext/internal/config"
)

func TestCollectGUIDReplacementsFromConfigDumpReusesPersistedAdoptedIDs(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Пользователи" id="11111111-1111-1111-1111-111111111111">
    <Metadata name="Catalog.Пользователи.Command.ПользователиИнформационнойБазы" id="22222222-2222-2222-2222-222222222222"/>
  </Metadata>
  <Metadata name="Catalog.Номенклатура" id="33333333-3333-3333-3333-333333333333"/>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	replacements := make(map[string]string)
	identityMap := &identityMapState{
		Version: 1,
		Objects: map[string]identityMapObjectBinding{
			"Catalog.Пользователи": {ExtensionID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
		},
	}

	collectGUIDReplacementsFromConfigDump(
		[]*FileProcessingContext{{Doc: doc, FileName: configDumpInfo}},
		map[string]objectDecision{
			"Catalog.Пользователи": {Belonging: "AdoptedStub"},
			"Catalog.Номенклатура": {Belonging: "Native"},
		},
		replacements,
		identityMap,
		nil,
	)

	if got := replacements["11111111-1111-1111-1111-111111111111"]; got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected reused top-level id: got %q", got)
	}

	commandState, ok := identityMap.Objects["Catalog.Пользователи.Command.ПользователиИнформационнойБазы"]
	if !ok || commandState.ExtensionID == "" {
		t.Fatalf("expected generated command identity, got %#v", identityMap.Objects)
	}
	if got := replacements["22222222-2222-2222-2222-222222222222"]; got != commandState.ExtensionID {
		t.Fatalf("unexpected generated command replacement: got %q want %q", got, commandState.ExtensionID)
	}
	if _, ok := identityMap.Objects["Catalog.Номенклатура"]; ok {
		t.Fatalf("native metadata must not be stored in identity map: %#v", identityMap.Objects)
	}
}

func TestCollectGUIDReplacementsFromConfigDumpSkipsNoOpReplacementForCurrentID(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Валюты" id="11111111-1111-1111-1111-111111111111"/>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	replacements := make(map[string]string)
	identityMap := &identityMapState{
		Version: 1,
		Objects: map[string]identityMapObjectBinding{
			"Catalog.Валюты": {ExtensionID: "11111111-1111-1111-1111-111111111111"},
		},
	}

	collectGUIDReplacementsFromConfigDump(
		[]*FileProcessingContext{{Doc: doc, FileName: configDumpInfo}},
		map[string]objectDecision{
			"Catalog.Валюты": {Belonging: "AdoptedStub"},
		},
		replacements,
		identityMap,
		nil,
	)

	state := identityMap.Objects["Catalog.Валюты"]
	if state.ExtensionID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected identity map to keep current id, got %#v", state)
	}
	if got := replacements["11111111-1111-1111-1111-111111111111"]; got != "" {
		t.Fatalf("expected no-op replacement to be skipped, got %q", got)
	}
}

func TestCollectGUIDReplacementsFromConfigDumpUpdatesPersistedSourceUUIDToCurrentConfigDumpID(t *testing.T) {
	t.Parallel()

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="BusinessProcess.Задание" id="22222222-2222-2222-2222-222222222222"/>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	objectDoc := etree.NewDocument()
	if err := objectDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <BusinessProcess uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Задание</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
      <ExtendedConfigurationObject>22222222-2222-2222-2222-222222222222</ExtendedConfigurationObject>
    </Properties>
  </BusinessProcess>
</MetaDataObject>`); err != nil {
		t.Fatalf("read metadata object xml: %v", err)
	}

	replacements := make(map[string]string)
	identityMap := &identityMapState{
		Version: 1,
		Objects: map[string]identityMapObjectBinding{
			"BusinessProcess.Задание": {ExtensionID: "11111111-1111-1111-1111-111111111111"},
		},
	}

	collectGUIDReplacementsFromConfigDump(
		[]*FileProcessingContext{
			{Doc: configDumpDoc, FileName: configDumpInfo},
			{Doc: objectDoc, OwnerKey: "BusinessProcess.Задание", Metadata: true, TopLevelMetadata: true},
		},
		map[string]objectDecision{
			"BusinessProcess.Задание": {Belonging: "AdoptedStub"},
		},
		replacements,
		identityMap,
		nil,
	)

	state := identityMap.Objects["BusinessProcess.Задание"]
	if state.ExtensionID != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("expected identity map to move from source uuid to current config dump id, got %#v", state)
	}
	if got := replacements["22222222-2222-2222-2222-222222222222"]; got != "" {
		t.Fatalf("expected no replacement for already-correct current config dump id, got %q", got)
	}
}

func TestCollectGUIDReplacementsFromConfigDumpTracksAdoptedTabularSectionButSkipsRetainedNativeChildren(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Пользователи" id="11111111-1111-1111-1111-111111111111">
    <Metadata name="Catalog.Пользователи.Attribute.упо_Код" id="22222222-2222-2222-2222-222222222222"/>
    <Metadata name="Catalog.Пользователи.TabularSection.Товары" id="33333333-3333-3333-3333-333333333333"/>
    <Metadata name="Catalog.Пользователи.TabularSection.Товары.Attribute.упо_Цена" id="44444444-4444-4444-4444-444444444444"/>
    <Metadata name="Catalog.Пользователи.Command.Открыть" id="55555555-5555-5555-5555-555555555555"/>
  </Metadata>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	identityMap := newIdentityMapState()
	replacements := make(map[string]string)

	collectGUIDReplacementsFromConfigDump(
		[]*FileProcessingContext{{Doc: doc, FileName: configDumpInfo}},
		map[string]objectDecision{
			"Catalog.Пользователи": {Belonging: "AdoptedStub"},
		},
		replacements,
		identityMap,
		map[string]adoptedStubMetaDataRule{
			"Catalog.Пользователи": {
				NativeAttributes: map[string]struct{}{
					"упо_Код": {},
				},
				NativeTabularSections: map[string]map[string]struct{}{
					"Товары": {
						"упо_Цена": {},
					},
				},
			},
		},
	)

	for _, key := range []string{
		"Catalog.Пользователи.Attribute.упо_Код",
		"Catalog.Пользователи.TabularSection.Товары.Attribute.упо_Цена",
	} {
		if _, ok := identityMap.Objects[key]; ok {
			t.Fatalf("retained native child must not be stored in identity map: %s", key)
		}
	}
	if _, ok := identityMap.Objects["Catalog.Пользователи.TabularSection.Товары"]; !ok {
		t.Fatalf("expected adopted tabular section to stay tracked, got %#v", identityMap.Objects)
	}
	if _, ok := identityMap.Objects["Catalog.Пользователи.Command.Открыть"]; !ok {
		t.Fatalf("expected adopted command to stay tracked, got %#v", identityMap.Objects)
	}
}

func TestCollectMetadataBindingTargetsAppliesOverridesOnlyToTrackedAdoptedPaths(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Пользователи</Name>
    </Properties>
    <ChildObjects>
      <Command uuid="22222222-2222-2222-2222-222222222222">
        <Properties>
          <Name>Открыть</Name>
        </Properties>
      </Command>
      <Attribute uuid="33333333-3333-3333-3333-333333333333">
        <Properties>
          <Name>упо_Код</Name>
        </Properties>
      </Attribute>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read metadata xml: %v", err)
	}

	targets := collectMetadataBindingTargets(
		doc,
		"Catalog.Пользователи",
		map[string]string{
			"Catalog.Пользователи":                 "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"Catalog.Пользователи.Command.Открыть": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		},
		map[string]objectDecision{
			"Catalog.Пользователи": {Belonging: "AdoptedStub"},
		},
		map[string]adoptedStubMetaDataRule{
			"Catalog.Пользователи": {
				NativeAttributes: map[string]struct{}{
					"упо_Код": {},
				},
			},
		},
	)

	target := metadataTargetElement(doc)
	if got := targets[target].BaseObjectID; got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected top-level binding: got %q", got)
	}
	if got := targets[target].CurrentID; got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected top-level current id: got %q", got)
	}
	if !targets[target].HasBinding {
		t.Fatalf("expected top-level binding to be marked as explicit")
	}

	command := target.FindElement("./ChildObjects/*[local-name()='Command']")
	if command == nil {
		t.Fatalf("expected command child")
	}
	if got := targets[command].BaseObjectID; got != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected command binding: got %q", got)
	}

	attribute := target.FindElement("./ChildObjects/*[local-name()='Attribute']")
	if attribute == nil {
		t.Fatalf("expected attribute child")
	}
	if got := targets[attribute].BaseObjectID; got != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("retained native attribute must keep original base object id, got %q", got)
	}
	if targets[attribute].HasBinding {
		t.Fatalf("retained native attribute must not be marked as explicit binding")
	}
}

func TestCollectMetadataBindingTargetsTracksAdoptedTabularSectionAndSkipsRetainedNativeTabularAttributes(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Пользователи</Name>
    </Properties>
    <ChildObjects>
      <TabularSection uuid="22222222-2222-2222-2222-222222222222">
        <Properties>
          <Name>Товары</Name>
        </Properties>
        <ChildObjects>
          <Attribute uuid="33333333-3333-3333-3333-333333333333">
            <Properties>
              <Name>упо_Цена</Name>
            </Properties>
          </Attribute>
        </ChildObjects>
      </TabularSection>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read metadata xml: %v", err)
	}

	targets := collectMetadataBindingTargets(
		doc,
		"Catalog.Пользователи",
		map[string]string{
			"Catalog.Пользователи":                                          "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			"Catalog.Пользователи.TabularSection.Товары":                    "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
			"Catalog.Пользователи.TabularSection.Товары.Attribute.упо_Цена": "cccccccc-cccc-cccc-cccc-cccccccccccc",
		},
		map[string]objectDecision{
			"Catalog.Пользователи": {Belonging: "AdoptedStub"},
		},
		map[string]adoptedStubMetaDataRule{
			"Catalog.Пользователи": {
				NativeTabularSections: map[string]map[string]struct{}{
					"Товары": {
						"упо_Цена": {},
					},
				},
			},
		},
	)

	target := metadataTargetElement(doc)
	section := target.FindElement("./ChildObjects/*[local-name()='TabularSection']")
	if section == nil {
		t.Fatalf("expected tabular section child")
	}
	if got := targets[section].BaseObjectID; got != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("unexpected tabular section binding: got %q", got)
	}
	if !targets[section].HasBinding {
		t.Fatalf("expected adopted tabular section to keep explicit binding")
	}

	attribute := target.FindElement("./ChildObjects/*[local-name()='TabularSection']/*[local-name()='ChildObjects']/*[local-name()='Attribute']")
	if attribute == nil {
		t.Fatalf("expected tabular attribute child")
	}
	if got := targets[attribute].BaseObjectID; got != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("retained native tabular attribute must keep original base object id, got %q", got)
	}
	if targets[attribute].HasBinding {
		t.Fatalf("retained native tabular attribute must not be marked as explicit binding")
	}
}

func TestEnsureAdoptedExtendedConfigurationObjectsAppliesBaseBindingToObjectItself(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Организации</Name>
    </Properties>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read metadata xml: %v", err)
	}

	targets := collectMetadataBindingTargets(
		doc,
		"Catalog.Организации",
		map[string]string{
			"Catalog.Организации": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
		map[string]objectDecision{
			"Catalog.Организации": {Belonging: "AdoptedStub"},
		},
		nil,
	)

	if !ensureAdoptedExtendedConfigurationObjects(doc, targets) {
		t.Fatalf("expected binding to be written to adopted object")
	}

	properties := doc.FindElement("//*[local-name()='Catalog']/*[local-name()='Properties']")
	if properties == nil {
		t.Fatalf("expected catalog properties")
	}
	if got := textOf(properties, "ExtendedConfigurationObject"); got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected base binding on object itself, got %q", got)
	}
}

func TestBaseBindingReferenceReplacementPropagatesToOtherObjectReferences(t *testing.T) {
	t.Parallel()

	boundDoc := etree.NewDocument()
	if err := boundDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Организации</Name>
    </Properties>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read bound metadata xml: %v", err)
	}

	refDoc := etree.NewDocument()
	if err := refDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>ПодразделенияОрганизаций</Name>
      <Owners>
        <xr:Item xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">11111111-1111-1111-1111-111111111111</xr:Item>
      </Owners>
    </Properties>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read referencing metadata xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:      boundDoc,
			OwnerKey: "Catalog.Организации",
		},
	}
	targetsByDoc := collectMetadataBindingTargetsByDoc(
		contexts,
		map[string]string{
			"Catalog.Организации": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
		map[string]objectDecision{
			"Catalog.Организации": {Belonging: "AdoptedStub"},
		},
		nil,
	)

	replacements := collectBaseBindingReferenceReplacements(targetsByDoc, map[string]string{
		"11111111-1111-1111-1111-111111111111": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	})

	if !replaceBaseBindingGUIDsInDoc(refDoc, replacements) {
		t.Fatalf("expected other object references to be updated from base binding")
	}

	if got := textOfFirst(refDoc.Root(), ".//*[local-name()='Owners']/*[1]"); got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected owner reference to be rebound to base object id, got %q", got)
	}
}

func TestReplaceBaseBindingGUIDsInDocDoesNotReplaceClassID(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <ClassId>11111111-1111-1111-1111-111111111111</ClassId>
    <ExtendedConfigurationObject>11111111-1111-1111-1111-111111111111</ExtendedConfigurationObject>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if !replaceBaseBindingGUIDsInDoc(doc, map[string]string{
		"11111111-1111-1111-1111-111111111111": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	}) {
		t.Fatalf("expected base binding replacement")
	}

	if got := textOfFirst(doc.Root(), ".//*[local-name()='ClassId']"); got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected ClassId to stay unchanged, got %q", got)
	}
	if got := textOfFirst(doc.Root(), ".//*[local-name()='ExtendedConfigurationObject']"); got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected non-ClassId GUID to be replaced, got %q", got)
	}
}

func TestReplaceBaseBindingGUIDsInDocDoesNotReplaceNativeObjectOwnUUID(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Организации</Name>
    </Properties>
    <Owner>11111111-1111-1111-1111-111111111111</Owner>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if !replaceBaseBindingGUIDsInDoc(doc, map[string]string{
		"11111111-1111-1111-1111-111111111111": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	}) {
		t.Fatalf("expected replacement in reference fields")
	}

	target := metadataTargetElement(doc)
	if target == nil {
		t.Fatalf("expected metadata target")
	}
	if got := target.SelectAttrValue("uuid", ""); got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected native object own uuid to stay unchanged, got %q", got)
	}
	if got := textOfFirst(doc.Root(), ".//*[local-name()='Owner']"); got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected reference field to be rebound, got %q", got)
	}
}

func TestCollectAdoptedStubMetaDataRulesReadsBOMAndNestedPaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templateDir := filepath.Join(dir, "CommonTemplates", "упо_MetaDataFile", "Ext")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}

	const attrPath = "Документ.ЗаказКлиента.Реквизит.упо_ОбъектИнтеграцииЭлементПлана"
	content := "\xEF\xBB\xBF{\n" +
		`  "КритерийОтбора": {` + "\n" +
		`    "СвязанныеДокументы": {` + "\n" +
		`      "Состав": ["` + attrPath + `"]` + "\n" +
		`    }` + "\n" +
		`  }` + "\n" +
		`}`
	if err := os.WriteFile(filepath.Join(templateDir, "Template.txt"), []byte(content), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := &config.Configuration{
		AdditionalProcessing: config.AdditionalProcessing{UseMetaDataFile: true},
	}

	rules := collectAdoptedStubMetaDataRules(cfg, dir)
	rule, ok := rules["Document.ЗаказКлиента"]
	if !ok {
		t.Fatalf("expected rule for Document.ЗаказКлиента, got %#v", rules)
	}
	if _, ok := rule.NativeAttributes["упо_ОбъектИнтеграцииЭлементПлана"]; !ok {
		t.Fatalf("expected native attribute from nested path, got %#v", rule.NativeAttributes)
	}
}

func TestValidateMetaDataFileTemplateRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templateDir := filepath.Join(dir, "CommonTemplates", "упо_MetaDataFile", "Ext")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}

	content := `{
  "ПланОбмена": {
    "ИнтеграцияС1СДокументооборотомПереопределяемый": {
      "Состав": [
        "Справочник.упо_Планы"
      ]
    }
  }
},
{
  "Документ": {
    "Бюджет": {
      "Реквизиты": [
        "Документ.Бюджет.Реквизит.упо_План"
      ]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(templateDir, "Template.txt"), []byte(content), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	err := validateMetaDataFileTemplate(&config.Configuration{
		AdditionalProcessing: config.AdditionalProcessing{UseMetaDataFile: true},
	}, dir)
	if err == nil {
		t.Fatal("expected invalid MetaDataFile template to fail validation")
	}
	if !strings.Contains(err.Error(), "CommonTemplates/упо_MetaDataFile/Ext/Template.txt") {
		t.Fatalf("expected template path in error, got %v", err)
	}
}

func TestMergeAdoptedStubMetaDataIntoFormContractUnionsFields(t *testing.T) {
	t.Parallel()

	contract := formDynamicListContract{
		RequiredFields: map[string]struct{}{
			"Номер":             {},
			"Товары.Количество": {},
		},
		QueryAliases: map[string]struct{}{
			"ПсевдонимПоля": {},
		},
	}
	rule := adoptedStubMetaDataRule{
		NativeAttributes: map[string]struct{}{
			"упо_ОбъектИнтеграцииЭлементПлана": {},
		},
		NativeTabularSections: map[string]map[string]struct{}{
			"Товары": {
				"упо_ДопРеквизит": {},
			},
		},
	}

	merged := mergeAdoptedStubMetaDataIntoFormContract(contract, rule)

	for _, field := range []string{
		"Номер",
		"Товары.Количество",
		"упо_ОбъектИнтеграцииЭлементПлана",
		"Товары",
		"Товары.упо_ДопРеквизит",
	} {
		if _, ok := merged.RequiredFields[field]; !ok {
			t.Fatalf("expected merged field %q, got %#v", field, merged.RequiredFields)
		}
	}
	if _, ok := merged.QueryAliases["ПсевдонимПоля"]; !ok {
		t.Fatalf("expected query alias to survive merge, got %#v", merged.QueryAliases)
	}
}

func TestApplyAdoptedStubMetaDataRulesRestoresSoftExcludedObject(t *testing.T) {
	t.Parallel()

	key := "Document.ОтражениеЗарплатыВФинансовомУчете"
	decisions := map[string]objectDecision{
		key: {Excluded: true},
	}
	rules := map[string]adoptedStubMetaDataRule{
		key: {
			NativeAttributes: map[string]struct{}{
				"упо_ВыполненоРаспределениеПоТрудозатратам": {},
			},
		},
	}

	applyAdoptedStubMetaDataRules(
		decisions,
		rules,
		map[string]struct{}{key: {}},
		nil,
	)

	decision := decisions[key]
	if decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected metadata-file object to override soft exclude as AdoptedStub, got %#v", decision)
	}
}

func TestCleanupMissingFormCommonAttributeDynamicListFields(t *testing.T) {
	t.Parallel()

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:dcssch="http://v8.1c.ru/8.1/data-composition-system/schema" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="СписокТаблица">
      <DataPath>Список</DataPath>
      <ChildItems>
        <LabelField name="упо_ДатаСоздания">
          <DataPath>Список.упо_ДатаСоздания</DataPath>
        </LabelField>
        <LabelField name="Ссылка">
          <DataPath>Список.Ref</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Список">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <Settings xsi:type="DynamicList">
        <MainTable>Catalog.упо_ДокументыОбязательств</MainTable>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>упо_ДатаСоздания</dcssch:dataPath>
          <dcssch:field>упо_ДатаСоздания</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Ref</dcssch:dataPath>
          <dcssch:field>Ref</dcssch:field>
        </Field>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>упо_ДокументыОбязательств</Name>
      <StandardAttributes>
        <StandardAttribute name="Ref"/>
      </StandardAttributes>
    </Properties>
    <ChildObjects/>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:       ownerDoc,
			Metadata:  true,
			OwnerKey:  "Catalog.упо_ДокументыОбязательств",
			OwnerKind: "Catalog",
			RelPath:   "Catalogs/упо_ДокументыОбязательств.xml",
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.упо_ДокументыОбязательств": {Belonging: "Native"},
		"CommonAttribute.упо_ДатаСоздания":  {Excluded: true},
	}

	if !cleanupMissingFormCommonAttributeDynamicListFields(formDoc, contexts, decisions) {
		t.Fatalf("expected missing common attribute dynamic list cleanup to change form")
	}

	for _, field := range formDoc.FindElements("//*[local-name()='LabelField']") {
		if strings.TrimSpace(field.SelectAttrValue("name", "")) == "упо_ДатаСоздания" {
			t.Fatalf("expected label bound to missing common attribute to be removed")
		}
	}
	for _, dataPath := range formDoc.FindElements("//*[local-name()='dataPath']") {
		if strings.TrimSpace(dataPath.Text()) == "упо_ДатаСоздания" {
			t.Fatalf("expected dynamic list field declaration for missing common attribute to be removed")
		}
	}

	keptReferenceField := false
	for _, field := range formDoc.FindElements("//*[local-name()='LabelField']") {
		if strings.TrimSpace(field.SelectAttrValue("name", "")) == "Ссылка" {
			keptReferenceField = true
			break
		}
	}
	if !keptReferenceField {
		t.Fatalf("expected unrelated field to remain")
	}
}

func TestCleanupMissingFormCommonAttributeDynamicListFieldsFromControlOnly(t *testing.T) {
	t.Parallel()

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="СписокТаблица">
      <DataPath>Список</DataPath>
      <ChildItems>
        <LabelField name="упо_ДатаСоздания">
          <DataPath>Список.упо_ДатаСоздания</DataPath>
        </LabelField>
        <LabelField name="Ссылка">
          <DataPath>Список.Ref</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Список">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <Settings xsi:type="DynamicList">
        <MainTable>Catalog.упо_ДокументыОбязательств</MainTable>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>упо_ДокументыОбязательств</Name>
      <StandardAttributes>
        <StandardAttribute name="Ref"/>
      </StandardAttributes>
    </Properties>
    <ChildObjects/>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:       ownerDoc,
			Metadata:  true,
			OwnerKey:  "Catalog.упо_ДокументыОбязательств",
			OwnerKind: "Catalog",
			RelPath:   "Catalogs/упо_ДокументыОбязательств.xml",
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.упо_ДокументыОбязательств": {Belonging: "Native"},
		"CommonAttribute.упо_ДатаСоздания":  {Excluded: true},
	}

	if !cleanupMissingFormCommonAttributeDynamicListFields(formDoc, contexts, decisions) {
		t.Fatalf("expected cleanup to remove control-only missing common attribute field")
	}

	for _, field := range formDoc.FindElements("//*[local-name()='LabelField']") {
		if strings.TrimSpace(field.SelectAttrValue("name", "")) == "упо_ДатаСоздания" {
			t.Fatalf("expected label bound to missing common attribute to be removed")
		}
	}
}

func TestCleanupFormDocumentIndexedKeepsIssue15VersionPlanNativeFields(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:dcssch="http://v8.1c.ru/8.1/data-composition-system/schema" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="ВерсииТаблица">
      <DataPath>Версии</DataPath>
      <ChildItems>
        <LabelField name="ВерсииНаименование">
          <DataPath>Версии.Наименование</DataPath>
        </LabelField>
        <LabelField name="Статус">
          <DataPath>Версии.Статус</DataPath>
        </LabelField>
        <LabelField name="КоличествоЭлементов">
          <DataPath>Версии.КоличествоЭлементов</DataPath>
        </LabelField>
        <LabelField name="ВерсииСписокСценариев">
          <DataPath>Версии.СписокСценариев</DataPath>
        </LabelField>
        <LabelField name="Ссылка">
          <DataPath>Версии.Ссылка</DataPath>
        </LabelField>
        <LabelField name="Код">
          <DataPath>Версии.Код</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Версии">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <Settings xsi:type="DynamicList">
        <MainTable>Catalog.упо_ВерсииПланов</MainTable>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Наименование</dcssch:dataPath>
          <dcssch:field>Наименование</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Статус</dcssch:dataPath>
          <dcssch:field>Статус</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>КоличествоЭлементов</dcssch:dataPath>
          <dcssch:field>КоличествоЭлементов</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>СписокСценариев</dcssch:dataPath>
          <dcssch:field>СписокСценариев</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Ссылка</dcssch:dataPath>
          <dcssch:field>Ссылка</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Код</dcssch:dataPath>
          <dcssch:field>Код</dcssch:field>
        </Field>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`)
	ownerDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>упо_ВерсииПланов</Name>
      <StandardAttributes>
        <StandardAttribute name="Ref"/>
        <StandardAttribute name="Description"/>
        <StandardAttribute name="Code"/>
      </StandardAttributes>
    </Properties>
    <ChildObjects/>
  </Catalog>
</MetaDataObject>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_ВерсииПланов",
		OwnerKind: "Catalog",
		RelPath:   "Catalogs/упо_ВерсииПланов/Forms/ФормаСписка/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{
		{
			Doc:       ownerDoc,
			Metadata:  true,
			OwnerKey:  "Catalog.упо_ВерсииПланов",
			OwnerKind: "Catalog",
			RelPath:   "Catalogs/упо_ВерсииПланов.xml",
		},
		formCtx,
	}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.упо_ВерсииПланов": {Belonging: "Native"},
	}

	if cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected native form cleanup to preserve issue #15 bindings")
	}

	for _, dataPath := range []string{
		"Версии.Наименование",
		"Версии.Статус",
		"Версии.КоличествоЭлементов",
		"Версии.СписокСценариев",
		"Версии.Ссылка",
		"Версии.Код",
	} {
		if !formDocHasDataPath(formDoc, dataPath) {
			t.Fatalf("expected DataPath %q to remain in native form", dataPath)
		}
	}

	for _, field := range []string{
		"Наименование",
		"Статус",
		"КоличествоЭлементов",
		"СписокСценариев",
		"Ссылка",
		"Код",
	} {
		if !dynamicListFieldDeclared(formDoc, "Версии", field) {
			t.Fatalf("expected dynamic list field %q to remain declared", field)
		}
	}
}

func TestCleanupFormDocumentIndexedKeepsIssue15PlansNativeFields(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:dcssch="http://v8.1c.ru/8.1/data-composition-system/schema" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="СписокПланов">
      <DataPath>Список</DataPath>
      <ChildItems>
        <LabelField name="ОбъектПлана">
          <DataPath>Список.ОбъектПлана</DataPath>
        </LabelField>
        <LabelField name="Наименование">
          <DataPath>Список.Наименование</DataPath>
        </LabelField>
        <LabelField name="Код">
          <DataPath>Список.Код</DataPath>
        </LabelField>
        <LabelField name="Ссылка">
          <DataPath>Список.Ссылка</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Список">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <Settings xsi:type="DynamicList">
        <MainTable>Catalog.упо_Планы</MainTable>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>ОбъектПлана</dcssch:dataPath>
          <dcssch:field>ОбъектПлана</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Наименование</dcssch:dataPath>
          <dcssch:field>Наименование</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Код</dcssch:dataPath>
          <dcssch:field>Код</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Ссылка</dcssch:dataPath>
          <dcssch:field>Ссылка</dcssch:field>
        </Field>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`)
	ownerDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>упо_Планы</Name>
      <StandardAttributes>
        <StandardAttribute name="Ref"/>
        <StandardAttribute name="Description"/>
        <StandardAttribute name="Code"/>
      </StandardAttributes>
    </Properties>
    <ChildObjects/>
  </Catalog>
</MetaDataObject>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_Планы",
		OwnerKind: "Catalog",
		RelPath:   "Catalogs/упо_Планы/Forms/ФормаСписка/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{
		{
			Doc:       ownerDoc,
			Metadata:  true,
			OwnerKey:  "Catalog.упо_Планы",
			OwnerKind: "Catalog",
			RelPath:   "Catalogs/упо_Планы.xml",
		},
		formCtx,
	}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.упо_Планы": {Belonging: "Native"},
	}

	if cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected native plans form cleanup to preserve issue #15 bindings")
	}

	for _, dataPath := range []string{
		"Список.ОбъектПлана",
		"Список.Наименование",
		"Список.Код",
		"Список.Ссылка",
	} {
		if !formDocHasDataPath(formDoc, dataPath) {
			t.Fatalf("expected DataPath %q to remain in native form", dataPath)
		}
	}

	for _, field := range []string{"ОбъектПлана", "Наименование", "Код", "Ссылка"} {
		if !dynamicListFieldDeclared(formDoc, "Список", field) {
			t.Fatalf("expected dynamic list field %q to remain declared", field)
		}
	}
}

func TestCleanupFormDocumentIndexedNonNativeStillRemovesMissingDynamicListFields(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:dcssch="http://v8.1c.ru/8.1/data-composition-system/schema" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="СписокПланов">
      <DataPath>Список</DataPath>
      <ChildItems>
        <LabelField name="ОбъектПлана">
          <DataPath>Список.ОбъектПлана</DataPath>
        </LabelField>
        <LabelField name="Ссылка">
          <DataPath>Список.Ссылка</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Список">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <Settings xsi:type="DynamicList">
        <MainTable>Catalog.упо_Планы</MainTable>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>ОбъектПлана</dcssch:dataPath>
          <dcssch:field>ОбъектПлана</dcssch:field>
        </Field>
        <Field xsi:type="dcssch:DataSetFieldField">
          <dcssch:dataPath>Ссылка</dcssch:dataPath>
          <dcssch:field>Ссылка</dcssch:field>
        </Field>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`)
	ownerDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>упо_Планы</Name>
      <StandardAttributes>
        <StandardAttribute name="Ref"/>
        <StandardAttribute name="Description"/>
        <StandardAttribute name="Code"/>
      </StandardAttributes>
    </Properties>
    <ChildObjects/>
  </Catalog>
</MetaDataObject>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_Планы",
		OwnerKind: "Catalog",
		RelPath:   "Catalogs/упо_Планы/Forms/ФормаСписка/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{
		{
			Doc:       ownerDoc,
			Metadata:  true,
			OwnerKey:  "Catalog.упо_Планы",
			OwnerKind: "Catalog",
			RelPath:   "Catalogs/упо_Планы.xml",
		},
		formCtx,
	}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.упо_Планы": {Belonging: "AdoptedStub"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected non-native form cleanup to remove missing dynamic list field")
	}
	if formDocHasDataPath(formDoc, "Список.ОбъектПлана") {
		t.Fatalf("expected non-native cleanup to remove missing issue #15 field")
	}
	if !formDocHasDataPath(formDoc, "Список.Ссылка") {
		t.Fatalf("expected standard field to remain after non-native cleanup")
	}
	if dynamicListFieldDeclared(formDoc, "Список", "ОбъектПлана") {
		t.Fatalf("expected missing field declaration to be removed for non-native form")
	}
}

func TestCleanupFormDocumentIndexedNativeStillRemovesBlockedNonNativeReferences(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="ВнешнийСписок">
      <DataPath>ВнешнийСписок</DataPath>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="ВнешнийСписок">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <Settings xsi:type="DynamicList">
        <MainTable>Catalog.НенативныйСписок</MainTable>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`)
	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_Планы",
		OwnerKind: "Catalog",
		RelPath:   "Catalogs/упо_Планы/Forms/ФормаСписка/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{formCtx}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.упо_Планы":        {Belonging: "Native"},
		"Catalog.НенативныйСписок": {Belonging: "AdoptedStub"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected native cleanup to remove blocked non-native dynamic list reference")
	}
	if formDocHasDataPath(formDoc, "ВнешнийСписок") {
		t.Fatalf("expected blocked non-native table to be removed from native form")
	}
}

func TestCleanupFormDocumentIndexedNativeKeepsPM5CommandsWithCommonPictureAndFunctionalOption(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <Button name="КомандаЗаполнитьАдресБазыПоТекущемуСоединению">
      <Type>UsualButton</Type>
      <CommandName>Form.Command.КомандаЗаполнитьАдресБазыПоТекущемуСоединению</CommandName>
    </Button>
    <Button name="КомандаПоказатьПодсказуАдресаБазы">
      <Type>UsualButton</Type>
      <CommandName>Form.Command.КомандаПоказатьПодсказуАдресаБазы</CommandName>
    </Button>
    <Button name="НенативнаяКоманда">
      <Type>Hyperlink</Type>
      <CommandName>Catalog.НенативныйСправочник.Command.ОткрытьНенативнуюФорму</CommandName>
    </Button>
  </ChildItems>
  <Commands>
    <Command name="КомандаЗаполнитьАдресБазыПоТекущемуСоединению">
      <Picture>
        <xr:Ref xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">CommonPicture.ЗаполнитьПоОснованию</xr:Ref>
      </Picture>
      <Action>КомандаЗаполнитьАдресБазыПоТекущемуСоединению</Action>
      <FunctionalOptions>
        <Item>FunctionalOption.упо_ИспользоватьОповещения</Item>
      </FunctionalOptions>
    </Command>
    <Command name="КомандаПоказатьПодсказуАдресаБазы">
      <Picture>
        <xr:Ref xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">CommonPicture.Информация16АнимированнаяБЗК</xr:Ref>
      </Picture>
      <Action>КомандаПоказатьПодсказуАдресаБазы</Action>
      <FunctionalOptions>
        <Item>FunctionalOption.упо_ИспользоватьОповещения</Item>
      </FunctionalOptions>
    </Command>
  </Commands>
</Form>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "CommonForm.упо_НастройкиPM5",
		OwnerKind: "CommonForm",
		FileName:  "Form.xml",
		RelPath:   "CommonForms/упо_НастройкиPM5/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{formCtx}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"CommonForm.упо_НастройкиPM5":                 {Belonging: "Native"},
		"CommonPicture.ЗаполнитьПоОснованию":          {Belonging: "AdoptedStub"},
		"CommonPicture.Информация16АнимированнаяБЗК":  {Belonging: "AdoptedStub"},
		"FunctionalOption.упо_ИспользоватьОповещения": {Belonging: "AdoptedStub"},
		"Catalog.НенативныйСправочник":                {Belonging: "AdoptedStub"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected native form cleanup to keep PM5 commands while removing unrelated non-native refs")
	}

	for _, value := range []string{
		"Form.Command.КомандаЗаполнитьАдресБазыПоТекущемуСоединению",
		"Form.Command.КомандаПоказатьПодсказуАдресаБазы",
		"CommonPicture.ЗаполнитьПоОснованию",
		"CommonPicture.Информация16АнимированнаяБЗК",
		"FunctionalOption.упо_ИспользоватьОповещения",
		"КомандаЗаполнитьАдресБазыПоТекущемуСоединению",
		"КомандаПоказатьПодсказуАдресаБазы",
	} {
		if !formDocContainsText(formDoc, value) {
			t.Fatalf("expected PM5 command-related value %q to remain", value)
		}
	}

	if formDocContainsText(formDoc, "Catalog.НенативныйСправочник.Command.ОткрытьНенативнуюФорму") {
		t.Fatalf("expected unrelated blocked metadata command reference to be removed")
	}
}

func TestCleanupFormDocumentIndexedNativeKeepsTelegramUserMappingCommandWithCommonPicture(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Button name="НенативнаяКоманда">
      <Type>CommandBarButton</Type>
      <CommandName>Catalog.НенативныйСправочник.Command.ОткрытьНенативнуюФорму</CommandName>
    </Button>
    <Table name="СообщенияБоту">
      <ContextMenu name="СообщенияБотуКонтекстноеМеню">
        <ChildItems>
          <Button name="СообщенияБотуКонтекстноеМенюКомандаУстановитьСоответствиеПользователя">
            <Type>CommandBarButton</Type>
            <CommandName>Form.Command.КомандаУстановитьСоответствиеПользователя</CommandName>
          </Button>
        </ChildItems>
      </ContextMenu>
      <AutoCommandBar name="СообщенияБотуКоманднаяПанель">
        <ChildItems>
          <Button name="СообщенияБотуКомандаУстановитьСоответствиеПользователя">
            <Type>CommandBarButton</Type>
            <CommandName>Form.Command.КомандаУстановитьСоответствиеПользователя</CommandName>
          </Button>
        </ChildItems>
      </AutoCommandBar>
    </Table>
    <Table name="СоответствиеПользователейTelegram">
      <DataPath>СоответствиеПользователейTelegram</DataPath>
      <CommandSet>
        <ExcludedCommand>Copy</ExcludedCommand>
      </CommandSet>
    </Table>
  </ChildItems>
  <Commands>
    <Command name="КомандаУстановитьСоответствиеПользователя">
      <Picture>
        <xr:Ref xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">CommonPicture.ОбычныйДоступСФлажком</xr:Ref>
      </Picture>
      <Action>КомандаУстановитьСоответствиеПользователя</Action>
      <AssociatedTableElementId xsi:type="xs:string">СообщенияБоту</AssociatedTableElementId>
    </Command>
  </Commands>
</Form>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "CommonForm.упо_НастройкиТранспортаОповещенийTelegram",
		OwnerKind: "CommonForm",
		FileName:  "Form.xml",
		RelPath:   "CommonForms/упо_НастройкиТранспортаОповещенийTelegram/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{formCtx}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"CommonForm.упо_НастройкиТранспортаОповещенийTelegram": {Belonging: "Native"},
		"CommonPicture.ОбычныйДоступСФлажком":                  {Belonging: "AdoptedStub"},
		"Catalog.НенативныйСправочник":                         {Belonging: "AdoptedStub"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected native form cleanup to keep telegram user mapping command")
	}

	for _, value := range []string{
		"Form.Command.КомандаУстановитьСоответствиеПользователя",
		"КомандаУстановитьСоответствиеПользователя",
		"CommonPicture.ОбычныйДоступСФлажком",
	} {
		if !formDocContainsText(formDoc, value) {
			t.Fatalf("expected telegram command-related value %q to remain", value)
		}
	}
	for _, name := range []string{
		"СообщенияБотуКонтекстноеМенюКомандаУстановитьСоответствиеПользователя",
		"СообщенияБотуКомандаУстановитьСоответствиеПользователя",
	} {
		if !hasFormElementWithName(formDoc, "Button", name) {
			t.Fatalf("expected telegram button %q to remain", name)
		}
	}
	var telegramTableCommandSet *etree.Element
	for _, table := range formDoc.Root().FindElements(".//*[local-name()='Table']") {
		if table.SelectAttrValue("name", "") != "СоответствиеПользователейTelegram" {
			continue
		}
		telegramTableCommandSet = table.FindElement("./*[local-name()='CommandSet']")
		break
	}
	if telegramTableCommandSet == nil {
		t.Fatalf("expected telegram user-mapping table command set to remain")
	}
	if _, ok := collectExcludedCommands(telegramTableCommandSet)["Copy"]; !ok {
		t.Fatalf("expected telegram user-mapping table copy command to remain")
	}
	if formDocContainsText(formDoc, "Catalog.НенативныйСправочник.Command.ОткрытьНенативнуюФорму") {
		t.Fatalf("expected unrelated blocked metadata command reference to be removed")
	}
}

func TestCleanupFormDocumentIndexedRemovesForbiddenStandardCommandsForNativeForms(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <Table name="СопоставлениеРеквизитов">
      <DataPath>СопоставлениеРеквизитов</DataPath>
      <CommandSet>
        <ExcludedCommand>Change</ExcludedCommand>
        <ExcludedCommand>Copy</ExcludedCommand>
        <ExcludedCommand>Delete</ExcludedCommand>
      </CommandSet>
      <AutoCommandBar name="Панель">
        <ChildItems>
          <Button name="Изменить">
            <Type>CommandBarButton</Type>
            <CommandName>Form.Item.СопоставлениеРеквизитов.StandardCommand.Change</CommandName>
          </Button>
          <Button name="История">
            <Type>CommandBarButton</Type>
            <CommandName>Form.Item.СопоставлениеРеквизитов.StandardCommand.ChangeHistory</CommandName>
          </Button>
          <Button name="Обновить">
            <Type>CommandBarButton</Type>
            <CommandName>Form.Item.СопоставлениеРеквизитов.StandardCommand.Refresh</CommandName>
          </Button>
        </ChildItems>
      </AutoCommandBar>
    </Table>
  </ChildItems>
</Form>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_ДокументыОбязательств",
		OwnerKind: "Catalog",
		RelPath:   "Catalogs/упо_ДокументыОбязательств/Forms/ФормаСписка/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{formCtx}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.упо_ДокументыОбязательств": {Belonging: "Native"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected native form cleanup to remove forbidden standard command references")
	}

	for _, forbidden := range []string{
		"Form.Item.СопоставлениеРеквизитов.StandardCommand.Change",
		"Form.Item.СопоставлениеРеквизитов.StandardCommand.ChangeHistory",
	} {
		if formDocContainsText(formDoc, forbidden) {
			t.Fatalf("expected forbidden command %q to be removed", forbidden)
		}
	}

	if !formDocContainsText(formDoc, "Change") {
		t.Fatalf("expected table command-set excluded command to remain")
	}
	if !formDocContainsText(formDoc, "Copy") {
		t.Fatalf("expected copy table command-set excluded command to remain")
	}
	if !formDocContainsText(formDoc, "Delete") {
		t.Fatalf("expected unrelated excluded command to remain")
	}
	if !formDocContainsText(formDoc, "Form.Item.СопоставлениеРеквизитов.StandardCommand.Refresh") {
		t.Fatalf("expected unrelated standard command to remain")
	}
}

func TestCleanupFormDocumentIndexedRemovesOnlyRootFormExcludedStandardCommands(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <CommandSet>
    <ExcludedCommand>Change</ExcludedCommand>
    <ExcludedCommand>Delete</ExcludedCommand>
    <ExcludedCommand>WriteAndClose</ExcludedCommand>
  </CommandSet>
  <ChildItems>
    <Table name="Список">
      <DataPath>Список</DataPath>
      <CommandSet>
        <ExcludedCommand>Change</ExcludedCommand>
        <ExcludedCommand>Delete</ExcludedCommand>
      </CommandSet>
    </Table>
  </ChildItems>
</Form>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_Тест",
		OwnerKind: "Catalog",
		FileName:  "Form.xml",
		RelPath:   "Catalogs/упо_Тест/Forms/ФормаСписка/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{formCtx}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.упо_Тест": {Belonging: "AdoptedStub"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected cleanup to remove invalid root form excluded commands")
	}

	if rootCommandSet := formDoc.Root().FindElement("./*[local-name()='CommandSet']"); rootCommandSet != nil {
		t.Fatalf("expected empty root form command set to be removed")
	}

	tableCommandSet := formDoc.Root().FindElement(".//*[local-name()='Table']/*[local-name()='CommandSet']")
	if tableCommandSet == nil {
		t.Fatalf("expected table command set to remain")
	}
	remaining := make(map[string]struct{})
	for _, child := range tableCommandSet.ChildElements() {
		if strings.EqualFold(localName(child.Tag), "ExcludedCommand") {
			remaining[strings.TrimSpace(child.Text())] = struct{}{}
		}
	}
	for _, command := range []string{"Change", "Delete"} {
		if _, ok := remaining[command]; !ok {
			t.Fatalf("expected table excluded command %q to remain", command)
		}
	}
}

func TestCleanupFormDocumentIndexedRemovesInvalidChoiceTableExcludedCommandsInOrdinaryForms(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/data">
  <CommandSet>
    <ExcludedCommand>Close</ExcludedCommand>
  </CommandSet>
  <ChildItems>
    <Table name="ГруппыПользователей">
      <Representation>Tree</Representation>
      <CommandBarLocation>None</CommandBarLocation>
      <ChoiceMode>true</ChoiceMode>
      <RowSelectionMode>Row</RowSelectionMode>
      <DataPath>ГруппыПользователей</DataPath>
      <Title>
        <v8:item>
          <v8:lang>ru</v8:lang>
          <v8:content>Группы пользователей</v8:content>
        </v8:item>
      </Title>
      <CommandSet>
        <ExcludedCommand>Delete</ExcludedCommand>
      </CommandSet>
      <CurrentRowUse>SelectionPresentationAndChoice</CurrentRowUse>
      <ContextMenu name="ГруппыПользователейКонтекстноеМеню"/>
      <AutoCommandBar name="ГруппыПользователейКоманднаяПанель"/>
    </Table>
    <Table name="Список">
      <CommandBarLocation>None</CommandBarLocation>
      <Height>8</Height>
      <ChoiceMode>true</ChoiceMode>
      <UseAlternationRowColor>true</UseAlternationRowColor>
      <EnableStartDrag>true</EnableStartDrag>
      <EnableDrag>true</EnableDrag>
      <FileDragMode>AsFile</FileDragMode>
      <DataPath>Список</DataPath>
      <Title>
        <v8:item>
          <v8:lang>ru</v8:lang>
          <v8:content>Список</v8:content>
        </v8:item>
      </Title>
      <CommandSet>
        <ExcludedCommand>CancelSearch</ExcludedCommand>
        <ExcludedCommand>Change</ExcludedCommand>
        <ExcludedCommand>Choose</ExcludedCommand>
        <ExcludedCommand>Copy</ExcludedCommand>
        <ExcludedCommand>CopyToClipboard</ExcludedCommand>
        <ExcludedCommand>Create</ExcludedCommand>
        <ExcludedCommand>Delete</ExcludedCommand>
        <ExcludedCommand>DynamicListStandardSettings</ExcludedCommand>
        <ExcludedCommand>Find</ExcludedCommand>
        <ExcludedCommand>MoveItem</ExcludedCommand>
        <ExcludedCommand>Refresh</ExcludedCommand>
      </CommandSet>
      <SearchStringLocation>None</SearchStringLocation>
      <ViewStatusLocation>None</ViewStatusLocation>
      <SearchControlLocation>None</SearchControlLocation>
      <ContextMenu name="СписокКонтекстноеМеню"/>
      <AutoCommandBar name="СписокКоманднаяПанель"/>
    </Table>
  </ChildItems>
</Form>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.Пользователи",
		OwnerKind: "Catalog",
		FileName:  "Form.xml",
		RelPath:   "Catalogs/Пользователи/Forms/ФормаСписка/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{formCtx}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.Пользователи": {Belonging: "Native"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected invalid choice-table excluded commands in ordinary forms to be removed")
	}

	findTableCommandSetByName := func(name string) *etree.Element {
		for _, table := range formDoc.Root().FindElements(".//*[local-name()='Table']") {
			if table.SelectAttrValue("name", "") != name {
				continue
			}
			return table.FindElement("./*[local-name()='CommandSet']")
		}
		return nil
	}

	groupTableCommandSet := findTableCommandSetByName("ГруппыПользователей")
	if groupTableCommandSet != nil {
		t.Fatalf("expected empty users choice-table command set to be removed")
	}

	fileTableCommandSet := findTableCommandSetByName("Список")
	if fileTableCommandSet == nil {
		t.Fatalf("expected file list command set to remain with non-problematic commands")
	}
	remaining := collectExcludedCommands(fileTableCommandSet)
	for _, command := range []string{"Change", "Copy", "Create", "Delete", "MoveItem"} {
		if _, ok := remaining[command]; ok {
			t.Fatalf("expected orphaned choice-table command %q to be removed", command)
		}
	}
	for _, command := range []string{"CancelSearch", "Choose", "CopyToClipboard", "DynamicListStandardSettings", "Find", "Refresh"} {
		if _, ok := remaining[command]; !ok {
			t.Fatalf("expected non-problematic choice-table command %q to remain", command)
		}
	}
}

func TestCleanupFormDocumentIndexedKeepsChoiceTableExcludedCommandsInSelectionForms(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/data">
  <CommandSet>
    <ExcludedCommand>Create</ExcludedCommand>
  </CommandSet>
  <ChildItems>
    <Table name="Список" id="1">
      <Representation>Tree</Representation>
      <CommandBarLocation>None</CommandBarLocation>
      <DefaultItem>true</DefaultItem>
      <ChoiceMode>true</ChoiceMode>
      <EnableStartDrag>true</EnableStartDrag>
      <EnableDrag>true</EnableDrag>
      <FileDragMode>AsFile</FileDragMode>
      <DataPath>Список</DataPath>
      <RowPictureDataPath>Список.DefaultPicture</RowPictureDataPath>
      <Title>
        <v8:item>
          <v8:lang>ru</v8:lang>
          <v8:content>Список</v8:content>
        </v8:item>
      </Title>
      <CommandSet>
        <ExcludedCommand>Create</ExcludedCommand>
      </CommandSet>
      <SearchStringLocation>None</SearchStringLocation>
      <ViewStatusLocation>None</ViewStatusLocation>
      <SearchControlLocation>None</SearchControlLocation>
    </Table>
  </ChildItems>
</Form>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_Показатели",
		OwnerKind: "Catalog",
		FileName:  "Form.xml",
		RelPath:   "Catalogs/упо_Показатели/Forms/ФормаВыбораГруппы/Ext/Form.xml",
	}
	contexts := []*FileProcessingContext{formCtx}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.упо_Показатели": {Belonging: "Native"},
	}

	if !cleanupFormDocumentIndexed(formCtx, contexts, indexes, decisions, collectNonNativeKeys(decisions)) {
		t.Fatalf("expected selection form cleanup to update only the root command set")
	}

	if formDoc.Root().FindElement("./*[local-name()='CommandSet']") != nil {
		t.Fatalf("expected root form command set to be removed after cleanup")
	}

	tableCommandSet := formDoc.Root().FindElement(".//*[local-name()='Table']/*[local-name()='CommandSet']")
	if tableCommandSet == nil {
		t.Fatalf("expected selection-form table command set to remain")
	}
	if _, ok := collectExcludedCommands(tableCommandSet)["Create"]; !ok {
		t.Fatalf("expected selection-form table excluded command to remain")
	}
}

func TestCleanupFormDocumentIndexedRemovesRootExcludedCommandsWithoutDecision(t *testing.T) {
	t.Parallel()

	formDoc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <CommandSet>
    <ExcludedCommand>Copy</ExcludedCommand>
    <ExcludedCommand>Create</ExcludedCommand>
  </CommandSet>
</Form>`)

	formCtx := &FileProcessingContext{
		Doc:       formDoc,
		OwnerKey:  "Catalog.упо_БезРешения",
		OwnerKind: "Catalog",
		FileName:  "Form.xml",
		RelPath:   "Catalogs/упо_БезРешения/Forms/ФормаСписка/Ext/Form.xml",
	}

	if !cleanupFormDocumentIndexed(formCtx, []*FileProcessingContext{formCtx}, buildContextIndexes([]*FileProcessingContext{formCtx}), map[string]objectDecision{}, nil) {
		t.Fatalf("expected root form excluded commands to be cleaned even without decision")
	}

	if formDoc.Root().FindElement("./*[local-name()='CommandSet']") != nil {
		t.Fatalf("expected invalid root form command set to be removed")
	}
}

func TestIsFormCleanupContextIncludesCommonFormExtForm(t *testing.T) {
	t.Parallel()

	if !isFormCleanupContext(&FileProcessingContext{
		FileName:  "Form.xml",
		OwnerKind: "CommonForm",
		RelPath:   "CommonForms/упо_РабочийСтолУправлениеДоговорами/Ext/Form.xml",
	}) {
		t.Fatalf("expected common form Ext/Form.xml to go through form cleanup")
	}

	if isFormCleanupContext(&FileProcessingContext{
		FileName:  "упо_РабочийСтолУправлениеДоговорами.xml",
		OwnerKind: "CommonForm",
		RelPath:   "CommonForms/упо_РабочийСтолУправлениеДоговорами.xml",
	}) {
		t.Fatalf("did not expect common form metadata object xml to go through form cleanup")
	}
}

func TestShouldIndexLiveCommandReferencesIncludesCommonFormExtForm(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		Doc:       mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?><Form xmlns="http://v8.1c.ru/8.3/xcf/logform"/>`),
		FileName:  "Form.xml",
		OwnerKind: "CommonForm",
		RelPath:   "CommonForms/упо_ПерсональныеНастройки/Ext/Form.xml",
	}

	if !shouldIndexLiveCommandReferences(ctx, objectDecision{Belonging: "AdoptedStub"}, nil) {
		t.Fatalf("expected common form Ext/Form.xml to participate in live command reference indexing")
	}
}

func TestNormalizeAdoptedStubMetaDataCompositionKeepsRetainedAttributeNative(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>НаправленияДеятельности</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <Attribute>
        <Properties>
          <Name>упо_ЭлементПлана</Name>
          <ObjectBelonging>Adopted</ObjectBelonging>
          <ExtendedConfigurationObject>attr-guid</ExtendedConfigurationObject>
        </Properties>
      </Attribute>
      <Attribute>
        <Properties>
          <Name>ЛишнийРеквизит</Name>
          <ObjectBelonging>Adopted</ObjectBelonging>
        </Properties>
      </Attribute>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	rule := adoptedStubMetaDataRule{
		NativeAttributes: map[string]struct{}{
			"упо_ЭлементПлана": {},
		},
	}

	if !normalizeAdoptedStubMetaDataComposition(doc, "Catalog", rule, searchResultObjectOverlay{}) {
		t.Fatalf("expected adopted metadata composition to change child attributes")
	}

	attrProps := doc.FindElement("//*[local-name()='Attribute']/*[local-name()='Properties']")
	if attrProps == nil {
		t.Fatalf("expected retained attribute properties")
	}
	if got := textOf(attrProps, "ObjectBelonging"); got != "" {
		t.Fatalf("expected retained attribute to be native, got ObjectBelonging=%q", got)
	}
	if got := textOf(attrProps, "ExtendedConfigurationObject"); got != "" {
		t.Fatalf("expected retained native attribute to drop ExtendedConfigurationObject, got %q", got)
	}

	attrs := doc.FindElements("//*[local-name()='ChildObjects']/*[local-name()='Attribute']")
	if len(attrs) != 1 {
		t.Fatalf("expected only retained attribute to survive, got %d", len(attrs))
	}
}

func TestNormalizeAdoptedStubMetaDataCompositionKeepsTabularSectionAdoptedAndRetainedChildrenNative(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Document>
    <Properties>
      <Name>Тест</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <TabularSection>
        <Properties>
          <Name>Состав</Name>
          <ObjectBelonging>Adopted</ObjectBelonging>
          <ExtendedConfigurationObject>section-guid</ExtendedConfigurationObject>
        </Properties>
        <ChildObjects>
          <Attribute>
            <Properties>
              <Name>упо_Реквизит</Name>
              <ObjectBelonging>Adopted</ObjectBelonging>
              <ExtendedConfigurationObject>attr-guid</ExtendedConfigurationObject>
            </Properties>
          </Attribute>
          <Attribute>
            <Properties>
              <Name>Лишний</Name>
              <ObjectBelonging>Adopted</ObjectBelonging>
            </Properties>
          </Attribute>
        </ChildObjects>
      </TabularSection>
    </ChildObjects>
  </Document>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	rule := adoptedStubMetaDataRule{
		NativeTabularSections: map[string]map[string]struct{}{
			"Состав": {
				"упо_Реквизит": {},
			},
		},
	}

	if !normalizeAdoptedStubMetaDataComposition(doc, "Document", rule, searchResultObjectOverlay{}) {
		t.Fatalf("expected adopted metadata composition to change tabular section child attributes")
	}

	sectionProps := doc.FindElement("//*[local-name()='TabularSection']/*[local-name()='Properties']")
	if sectionProps == nil {
		t.Fatalf("expected retained tabular section properties")
	}
	if got := textOf(sectionProps, "ObjectBelonging"); got != "Adopted" {
		t.Fatalf("expected retained tabular section to stay adopted, got ObjectBelonging=%q", got)
	}
	if got := textOf(sectionProps, "ExtendedConfigurationObject"); got != "section-guid" {
		t.Fatalf("expected retained adopted tabular section to keep ExtendedConfigurationObject, got %q", got)
	}

	attrProps := doc.FindElement("//*[local-name()='TabularSection']/*[local-name()='ChildObjects']/*[local-name()='Attribute']/*[local-name()='Properties']")
	if attrProps == nil {
		t.Fatalf("expected retained tabular attribute properties")
	}
	if got := textOf(attrProps, "ObjectBelonging"); got != "" {
		t.Fatalf("expected retained tabular attribute to be native, got ObjectBelonging=%q", got)
	}
	if got := textOf(attrProps, "ExtendedConfigurationObject"); got != "" {
		t.Fatalf("expected retained native tabular attribute to drop ExtendedConfigurationObject, got %q", got)
	}

	attrs := doc.FindElements("//*[local-name()='TabularSection']/*[local-name()='ChildObjects']/*[local-name()='Attribute']")
	if len(attrs) != 1 {
		t.Fatalf("expected only retained tabular attribute to survive, got %d", len(attrs))
	}
}

func TestNormalizeAdoptedStubMetaDataCompositionKeepsMultipleRetainedTabularSectionChildrenNative(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Document>
    <Properties>
      <Name>Тест</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <TabularSection>
        <Properties>
          <Name>Состав</Name>
          <ObjectBelonging>Adopted</ObjectBelonging>
          <ExtendedConfigurationObject>section-guid</ExtendedConfigurationObject>
        </Properties>
        <ChildObjects>
          <Attribute>
            <Properties>
              <Name>упо_Реквизит1</Name>
              <ObjectBelonging>Adopted</ObjectBelonging>
              <ExtendedConfigurationObject>attr-guid-1</ExtendedConfigurationObject>
            </Properties>
          </Attribute>
          <Attribute>
            <Properties>
              <Name>упо_Реквизит2</Name>
              <ObjectBelonging>Adopted</ObjectBelonging>
              <ExtendedConfigurationObject>attr-guid-2</ExtendedConfigurationObject>
            </Properties>
          </Attribute>
          <Attribute>
            <Properties>
              <Name>Лишний</Name>
              <ObjectBelonging>Adopted</ObjectBelonging>
            </Properties>
          </Attribute>
        </ChildObjects>
      </TabularSection>
    </ChildObjects>
  </Document>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	rule := adoptedStubMetaDataRule{
		NativeTabularSections: map[string]map[string]struct{}{
			"Состав": {
				"упо_Реквизит1": {},
				"упо_Реквизит2": {},
			},
		},
	}

	if !normalizeAdoptedStubMetaDataComposition(doc, "Document", rule, searchResultObjectOverlay{}) {
		t.Fatalf("expected adopted metadata composition to keep multiple native tabular attributes")
	}

	sectionProps := doc.FindElement("//*[local-name()='TabularSection']/*[local-name()='Properties']")
	if sectionProps == nil {
		t.Fatalf("expected retained tabular section properties")
	}
	if got := textOf(sectionProps, "ObjectBelonging"); got != "Adopted" {
		t.Fatalf("expected retained tabular section to stay adopted, got ObjectBelonging=%q", got)
	}

	attrs := doc.FindElements("//*[local-name()='TabularSection']/*[local-name()='ChildObjects']/*[local-name()='Attribute']")
	if len(attrs) != 2 {
		t.Fatalf("expected only retained tabular attributes to survive, got %d", len(attrs))
	}
	for _, attr := range attrs {
		props := attr.FindElement("./Properties")
		if props == nil {
			t.Fatalf("expected retained tabular attribute properties")
		}
		if got := textOf(props, "ObjectBelonging"); got != "" {
			t.Fatalf("expected retained tabular attribute to be native, got ObjectBelonging=%q", got)
		}
		if got := textOf(props, "ExtendedConfigurationObject"); got != "" {
			t.Fatalf("expected retained native tabular attribute to drop ExtendedConfigurationObject, got %q", got)
		}
	}
}

func TestEnsureAdoptedExtendedConfigurationObjectsPreservesRetainedMetaDataFileAttributeNative(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>НаправленияДеятельности</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <Attribute uuid="22222222-2222-2222-2222-222222222222">
        <Properties>
          <Name>упо_ЭлементПлана</Name>
          <ObjectBelonging>Adopted</ObjectBelonging>
        </Properties>
      </Attribute>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	rule := adoptedStubMetaDataRule{
		NativeAttributes: map[string]struct{}{
			"упо_ЭлементПлана": {},
		},
	}

	if !normalizeAdoptedStubMetaDataComposition(doc, "Catalog", rule, searchResultObjectOverlay{}) {
		t.Fatalf("expected adopted metadata composition to change child attribute")
	}

	bindingTargets := collectMetadataBindingTargets(
		doc,
		"Catalog.НаправленияДеятельности",
		nil,
		map[string]objectDecision{
			"Catalog.НаправленияДеятельности": {Belonging: "AdoptedStub"},
		},
		map[string]adoptedStubMetaDataRule{
			"Catalog.НаправленияДеятельности": rule,
		},
	)
	if !ensureAdoptedExtendedConfigurationObjects(doc, bindingTargets) {
		t.Fatalf("expected ensure adopted objects to write mapping")
	}

	attr := doc.FindElement("//*[local-name()='Attribute']")
	if attr == nil {
		t.Fatalf("expected retained attribute")
	}
	if got := attr.SelectAttrValue(preserveNativeObjectBelongingAttr, ""); got != "" {
		t.Fatalf("expected native-preserve marker to be removed, got %q", got)
	}

	attrProps := attr.FindElement("./Properties")
	if attrProps == nil {
		t.Fatalf("expected retained attribute properties")
	}
	if got := textOf(attrProps, "ObjectBelonging"); got != "" {
		t.Fatalf("expected retained attribute to stay native after ensure, got %q", got)
	}
	if got := textOf(attrProps, "ExtendedConfigurationObject"); got != "" {
		t.Fatalf("expected retained native attribute to stay without ExtendedConfigurationObject, got %q", got)
	}
}

func TestCollectConfigurationChildObjectReferencesIncludesConfiguredNativeObjects(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <CommonPicture>Информация33</CommonPicture>
      <Catalog>ПравилаПолученияФактаПоСтатьямБюджетов</Catalog>
      <Catalog>ОбычныйКаталогБезИнклуда</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	refs := collectConfigurationChildObjectReferences(doc.Root(), map[string]struct{}{
		"CommonPicture.Информация33":                     {},
		"Catalog.ПравилаПолученияФактаПоСтатьямБюджетов": {},
	})

	if _, ok := refs["CommonPicture.Информация33"]; !ok {
		t.Fatalf("expected configuration refs to include configured native common picture")
	}
	if _, ok := refs["Catalog.ПравилаПолученияФактаПоСтатьямБюджетов"]; !ok {
		t.Fatalf("expected configuration refs to include configured native catalog")
	}
	if _, ok := refs["Catalog.ОбычныйКаталогБезИнклуда"]; ok {
		t.Fatalf("did not expect configuration refs to include object outside primary native set")
	}
}

func TestCleanupConfigDumpInfoNonNativeChildrenRemovesAdoptedModules(t *testing.T) {
	t.Parallel()

	configDump := etree.NewDocument()
	if err := configDump.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/readable" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <Metadata name="Catalog.Тест">
    <Metadata name="Catalog.Тест.ManagerModule"/>
    <Metadata name="Catalog.Тест.ObjectModule"/>
    <Metadata name="Catalog.Тест.Command.ТестКоманда.CommandModule"/>
  </Metadata>
  <Metadata name="CommonModule.ТестОбщийМодуль">
    <Metadata name="CommonModule.ТестОбщийМодуль.Module"/>
  </Metadata>
  <Metadata name="CommonCommand.ТестОбщаяКоманда">
    <Metadata name="CommonCommand.ТестОбщаяКоманда.CommandModule"/>
  </Metadata>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	adoptedDoc := etree.NewDocument()
	if err := adoptedDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Тест</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <ManagerModule>
        <Properties>
          <Name>ManagerModule</Name>
        </Properties>
      </ManagerModule>
      <ObjectModule>
        <Properties>
          <Name>ObjectModule</Name>
        </Properties>
      </ObjectModule>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read adopted xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:       adoptedDoc,
			Metadata:  true,
			OwnerKey:  "Catalog.Тест",
			OwnerKind: "Catalog",
			OwnerName: "Тест",
		},
		{
			Metadata:  true,
			OwnerKey:  "CommonModule.ТестОбщийМодуль",
			OwnerKind: "CommonModule",
			OwnerName: "ТестОбщийМодуль",
		},
		{
			Metadata:  true,
			OwnerKey:  "CommonCommand.ТестОбщаяКоманда",
			OwnerKind: "CommonCommand",
			OwnerName: "ТестОбщаяКоманда",
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Belonging: "AdoptedStub"},
		"CommonModule.ТестОбщийМодуль":   {Belonging: "AdoptedStub"},
		"CommonCommand.ТестОбщаяКоманда": {Belonging: "AdoptedStub"},
	}

	if !cleanupConfigDumpInfoNonNativeChildren(configDump, contexts, decisions, nil) {
		t.Fatalf("expected config dump cleanup to remove adopted module entries")
	}

	root := configDump.Root()
	if root == nil {
		t.Fatalf("expected config dump root")
	}
	hasMetadataName := func(target string) bool {
		var walk func(*etree.Element) bool
		walk = func(el *etree.Element) bool {
			if el == nil {
				return false
			}
			if strings.EqualFold(localName(el.Tag), "Metadata") && strings.TrimSpace(el.SelectAttrValue("name", "")) == target {
				return true
			}
			for _, child := range el.ChildElements() {
				if walk(child) {
					return true
				}
			}
			return false
		}
		return walk(root)
	}

	if hasMetadataName("Catalog.Тест.ManagerModule") {
		t.Fatalf("expected ManagerModule metadata entry to be removed")
	}
	if hasMetadataName("Catalog.Тест.ObjectModule") {
		t.Fatalf("expected ObjectModule metadata entry to be removed")
	}
	if hasMetadataName("Catalog.Тест.Command.ТестКоманда.CommandModule") {
		t.Fatalf("expected CommandModule metadata entry to be removed")
	}
	if hasMetadataName("CommonModule.ТестОбщийМодуль.Module") {
		t.Fatalf("expected CommonModule Module metadata entry to be removed")
	}
	if hasMetadataName("CommonCommand.ТестОбщаяКоманда.CommandModule") {
		t.Fatalf("expected CommonCommand CommandModule metadata entry to be removed")
	}
}

func TestCleanupConfigDumpInfoNonNativeChildrenRemovesAdoptedConstantValueManagerModule(t *testing.T) {
	t.Parallel()

	configDump := etree.NewDocument()
	if err := configDump.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/readable" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <Metadata name="Constant.ТестКонстанта">
    <Metadata name="Constant.ТестКонстанта.ValueManagerModule"/>
  </Metadata>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	adoptedDoc := etree.NewDocument()
	if err := adoptedDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Constant>
    <Properties>
      <Name>ТестКонстанта</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <ValueManagerModule>
        <Properties>
          <Name>ValueManagerModule</Name>
        </Properties>
      </ValueManagerModule>
    </ChildObjects>
  </Constant>
</MetaDataObject>`); err != nil {
		t.Fatalf("read adopted xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:       adoptedDoc,
			Metadata:  true,
			OwnerKey:  "Constant.ТестКонстанта",
			OwnerKind: "Constant",
			OwnerName: "ТестКонстанта",
		},
	}
	decisions := map[string]objectDecision{
		"Constant.ТестКонстанта": {Belonging: "AdoptedStub"},
	}

	if !cleanupConfigDumpInfoNonNativeChildren(configDump, contexts, decisions, nil) {
		t.Fatalf("expected config dump cleanup to remove adopted constant value manager module entry")
	}

	root := configDump.Root()
	if root == nil {
		t.Fatalf("expected config dump root")
	}

	var hasMetadataName func(*etree.Element, string) bool
	hasMetadataName = func(el *etree.Element, target string) bool {
		if el == nil {
			return false
		}
		if strings.EqualFold(localName(el.Tag), "Metadata") && strings.TrimSpace(el.SelectAttrValue("name", "")) == target {
			return true
		}
		for _, child := range el.ChildElements() {
			if hasMetadataName(child, target) {
				return true
			}
		}
		return false
	}

	if hasMetadataName(root, "Constant.ТестКонстанта.ValueManagerModule") {
		t.Fatalf("expected ValueManagerModule metadata entry to be removed")
	}
}

func TestCollectAdoptedCommonModuleModulePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	moduleDir := filepath.Join(root, "CommonModules", "ТестОбщийМодуль", "Ext")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}
	modulePath := filepath.Join(moduleDir, "Module.bsl")
	if err := os.WriteFile(modulePath, []byte("// adopted common module"), 0o644); err != nil {
		t.Fatalf("write module file: %v", err)
	}

	excludedPaths := make(map[string]struct{})
	collectAdoptedCommonModuleModulePaths(root, map[string]objectDecision{
		"CommonModule.ТестОбщийМодуль": {Belonging: "AdoptedStub"},
		"CommonModule.НативныйМодуль":  {Belonging: "Native"},
	}, excludedPaths)

	if _, ok := excludedPaths[modulePath]; !ok {
		t.Fatalf("expected adopted common module file to be added to excluded paths")
	}
	if _, ok := excludedPaths[filepath.Join(root, "CommonModules", "НативныйМодуль", "Ext", "Module.bsl")]; ok {
		t.Fatalf("did not expect native common module file to be added to excluded paths")
	}
}

func TestCollectAdoptedCommandModulePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	objectCommandDir := filepath.Join(root, "Catalogs", "ТестКаталог", "Commands", "ТестКоманда", "Ext")
	if err := os.MkdirAll(objectCommandDir, 0o755); err != nil {
		t.Fatalf("mkdir object command dir: %v", err)
	}
	objectCommandModulePath := filepath.Join(objectCommandDir, "CommandModule.bsl")
	if err := os.WriteFile(objectCommandModulePath, []byte("// adopted object command"), 0o644); err != nil {
		t.Fatalf("write object command module: %v", err)
	}

	commonCommandDir := filepath.Join(root, "CommonCommands", "ТестОбщаяКоманда", "Ext")
	if err := os.MkdirAll(commonCommandDir, 0o755); err != nil {
		t.Fatalf("mkdir common command dir: %v", err)
	}
	commonCommandModulePath := filepath.Join(commonCommandDir, "CommandModule.bsl")
	if err := os.WriteFile(commonCommandModulePath, []byte("// adopted common command"), 0o644); err != nil {
		t.Fatalf("write common command module: %v", err)
	}

	nativeObjectCommandDir := filepath.Join(root, "Catalogs", "НативныйКаталог", "Commands", "НативнаяКоманда", "Ext")
	if err := os.MkdirAll(nativeObjectCommandDir, 0o755); err != nil {
		t.Fatalf("mkdir native object command dir: %v", err)
	}
	nativeObjectCommandModulePath := filepath.Join(nativeObjectCommandDir, "CommandModule.bsl")
	if err := os.WriteFile(nativeObjectCommandModulePath, []byte("// native object command"), 0o644); err != nil {
		t.Fatalf("write native object command module: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Path:      filepath.Join(root, "Catalogs", "ТестКаталог.xml"),
			RelPath:   "Catalogs/ТестКаталог.xml",
			FileName:  "ТестКаталог.xml",
			Metadata:  true,
			OwnerKey:  "Catalog.ТестКаталог",
			OwnerKind: "Catalog",
			OwnerName: "ТестКаталог",
		},
		{
			Path:      filepath.Join(root, "CommonCommands", "ТестОбщаяКоманда.xml"),
			RelPath:   "CommonCommands/ТестОбщаяКоманда.xml",
			FileName:  "ТестОбщаяКоманда.xml",
			Metadata:  true,
			OwnerKey:  "CommonCommand.ТестОбщаяКоманда",
			OwnerKind: "CommonCommand",
			OwnerName: "ТестОбщаяКоманда",
		},
		{
			Path:      filepath.Join(root, "Catalogs", "НативныйКаталог.xml"),
			RelPath:   "Catalogs/НативныйКаталог.xml",
			FileName:  "НативныйКаталог.xml",
			Metadata:  true,
			OwnerKey:  "Catalog.НативныйКаталог",
			OwnerKind: "Catalog",
			OwnerName: "НативныйКаталог",
		},
	}

	excludedPaths := make(map[string]struct{})
	collectAdoptedCommandModulePaths(contexts, map[string]objectDecision{
		"Catalog.ТестКаталог":            {Belonging: "AdoptedStub"},
		"CommonCommand.ТестОбщаяКоманда": {Belonging: "AdoptedStub"},
		"Catalog.НативныйКаталог":        {Belonging: "Native"},
	}, excludedPaths)

	if _, ok := excludedPaths[objectCommandModulePath]; !ok {
		t.Fatalf("expected adopted object command module to be added to excluded paths")
	}
	if _, ok := excludedPaths[commonCommandModulePath]; !ok {
		t.Fatalf("expected adopted common command module to be added to excluded paths")
	}
	if _, ok := excludedPaths[nativeObjectCommandModulePath]; ok {
		t.Fatalf("did not expect native object command module to be added to excluded paths")
	}
}

func TestCollectAdoptedCodeModulePathsMarksAdoptedRegisterRecordSetModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	adoptedDir := filepath.Join(root, "InformationRegisters", "ИсполнителиЗадач", "Ext")
	if err := os.MkdirAll(adoptedDir, 0o755); err != nil {
		t.Fatalf("mkdir adopted register dir: %v", err)
	}
	adoptedModulePath := filepath.Join(adoptedDir, "RecordSetModule.bsl")
	if err := os.WriteFile(adoptedModulePath, []byte("// adopted record set"), 0o644); err != nil {
		t.Fatalf("write adopted record set module: %v", err)
	}

	nativeDir := filepath.Join(root, "InformationRegisters", "НативныйРегистр", "Ext")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("mkdir native register dir: %v", err)
	}
	nativeModulePath := filepath.Join(nativeDir, "RecordSetModule.bsl")
	if err := os.WriteFile(nativeModulePath, []byte("// native record set"), 0o644); err != nil {
		t.Fatalf("write native record set module: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Path:      filepath.Join(root, "InformationRegisters", "ИсполнителиЗадач.xml"),
			RelPath:   "InformationRegisters/ИсполнителиЗадач.xml",
			FileName:  "ИсполнителиЗадач.xml",
			Metadata:  true,
			OwnerKey:  "InformationRegister.ИсполнителиЗадач",
			OwnerKind: "InformationRegister",
			OwnerName: "ИсполнителиЗадач",
		},
		{
			Path:      filepath.Join(root, "InformationRegisters", "НативныйРегистр.xml"),
			RelPath:   "InformationRegisters/НативныйРегистр.xml",
			FileName:  "НативныйРегистр.xml",
			Metadata:  true,
			OwnerKey:  "InformationRegister.НативныйРегистр",
			OwnerKind: "InformationRegister",
			OwnerName: "НативныйРегистр",
		},
	}

	excludedPaths := make(map[string]struct{})
	collectAdoptedCodeModulePaths(contexts, map[string]objectDecision{
		"InformationRegister.ИсполнителиЗадач": {Belonging: "AdoptedStub"},
		"InformationRegister.НативныйРегистр":  {Belonging: "Native"},
	}, excludedPaths)

	if _, ok := excludedPaths[adoptedModulePath]; !ok {
		t.Fatalf("expected adopted register RecordSetModule to be added to excluded paths")
	}
	if _, ok := excludedPaths[nativeModulePath]; ok {
		t.Fatalf("did not expect native register RecordSetModule to be added to excluded paths")
	}
}

func TestBuildChangeFilesStateTargetAdoptedRegisterExcludesRecordSetModule(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	registerXML := `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <InformationRegister>
    <Properties>
      <Name>КурсыВалют</Name>
    </Properties>
  </InformationRegister>
</MetaDataObject>`

	writeFile(root, filepath.Join("InformationRegisters", "КурсыВалют.xml"), registerXML)
	writeFile(root, filepath.Join("InformationRegisters", "КурсыВалют", "Ext", "RecordSetModule.bsl"), "// record set")
	writeFile(targetRoot, filepath.Join("InformationRegisters", "КурсыВалют.xml"), registerXML)

	state, err := buildChangeFilesState(&config.Configuration{
		IncludedAdoptedStubObjects: []string{"InformationRegister.КурсыВалют"},
		Target:                     config.Target{XMLDump: targetRoot},
	}, root)
	if err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	decision := state.decisions["InformationRegister.КурсыВалют"]
	if decision.Belonging != "AdoptedStub" || decision.Excluded {
		t.Fatalf("expected target-adopted register to stay AdoptedStub, got %#v", decision)
	}

	recordSetModulePath := filepath.Join(root, "InformationRegisters", "КурсыВалют", "Ext", "RecordSetModule.bsl")
	if _, ok := state.excludedPaths[recordSetModulePath]; !ok {
		t.Fatalf("expected RecordSetModule to be excluded for target-adopted register")
	}

	if _, err := removeExcludedFiles(root, state.excludedPaths); err != nil {
		t.Fatalf("remove excluded files: %v", err)
	}
	if _, err := os.Stat(recordSetModulePath); !os.IsNotExist(err) {
		t.Fatalf("expected RecordSetModule to be removed, stat err=%v", err)
	}
}

func TestCollectExcludedPathsMatchesChangeFilesCleanupInputs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	catalogPath := filepath.Join(root, "Catalogs", "ТестКаталог.xml")
	if err := os.MkdirAll(filepath.Join(root, "Catalogs", "ТестКаталог", "Ext"), 0o755); err != nil {
		t.Fatalf("mkdir catalog ext dir: %v", err)
	}
	for _, name := range []string{"ManagerModule.bsl", "ObjectModule.bsl"} {
		if err := os.WriteFile(filepath.Join(root, "Catalogs", "ТестКаталог", "Ext", name), []byte("// adopted"), 0o644); err != nil {
			t.Fatalf("write catalog module %s: %v", name, err)
		}
	}
	if err := os.WriteFile(catalogPath, []byte("<Catalog/>"), 0o644); err != nil {
		t.Fatalf("write catalog xml: %v", err)
	}

	constantPath := filepath.Join(root, "Constants", "ТестКонстанта.xml")
	if err := os.MkdirAll(filepath.Join(root, "Constants", "ТестКонстанта", "Ext"), 0o755); err != nil {
		t.Fatalf("mkdir constant ext dir: %v", err)
	}
	valueManagerModulePath := filepath.Join(root, "Constants", "ТестКонстанта", "Ext", "ValueManagerModule.bsl")
	if err := os.WriteFile(valueManagerModulePath, []byte("// adopted"), 0o644); err != nil {
		t.Fatalf("write value manager module: %v", err)
	}
	if err := os.WriteFile(constantPath, []byte("<Constant/>"), 0o644); err != nil {
		t.Fatalf("write constant xml: %v", err)
	}

	commonModulePath := filepath.Join(root, "CommonModules", "ТестОбщийМодуль", "Ext", "Module.bsl")
	if err := os.MkdirAll(filepath.Dir(commonModulePath), 0o755); err != nil {
		t.Fatalf("mkdir common module dir: %v", err)
	}
	if err := os.WriteFile(commonModulePath, []byte("// adopted"), 0o644); err != nil {
		t.Fatalf("write common module: %v", err)
	}

	commandPath := filepath.Join(root, "CommonCommands", "ТестКоманда.xml")
	if err := os.MkdirAll(filepath.Join(root, "CommonCommands", "ТестКоманда", "Ext"), 0o755); err != nil {
		t.Fatalf("mkdir common command dir: %v", err)
	}
	commandModulePath := filepath.Join(root, "CommonCommands", "ТестКоманда", "Ext", "CommandModule.bsl")
	if err := os.WriteFile(commandModulePath, []byte("// adopted"), 0o644); err != nil {
		t.Fatalf("write command module: %v", err)
	}
	if err := os.WriteFile(commandPath, []byte("<CommonCommand/>"), 0o644); err != nil {
		t.Fatalf("write common command xml: %v", err)
	}

	ownerPath := filepath.Join(root, "Catalogs", "Пользователи.xml")
	if err := os.MkdirAll(filepath.Join(root, "Catalogs", "Пользователи", "Forms", "ФормаВыбора", "Ext", "Form"), 0o755); err != nil {
		t.Fatalf("mkdir owner form dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "Catalogs", "Пользователи", "Commands", "ПользователиИнформационнойБазы", "Ext"), 0o755); err != nil {
		t.Fatalf("mkdir owner command dir: %v", err)
	}
	formMetadataPath := filepath.Join(root, "Catalogs", "Пользователи", "Forms", "ФормаВыбора.xml")
	formXMLPath := filepath.Join(root, "Catalogs", "Пользователи", "Forms", "ФормаВыбора", "Ext", "Form.xml")
	formModulePath := filepath.Join(root, "Catalogs", "Пользователи", "Forms", "ФормаВыбора", "Ext", "Form", "Module.bsl")
	childCommandModulePath := filepath.Join(root, "Catalogs", "Пользователи", "Commands", "ПользователиИнформационнойБазы", "Ext", "CommandModule.bsl")
	for _, item := range []string{ownerPath, formMetadataPath, formXMLPath, formModulePath, childCommandModulePath} {
		if err := os.WriteFile(item, []byte("<xml/>"), 0o644); err != nil {
			t.Fatalf("write %s: %v", item, err)
		}
	}

	rootModulePath := filepath.Join(root, "Ext", "ManagedApplicationModule.bsl")
	if err := os.MkdirAll(filepath.Dir(rootModulePath), 0o755); err != nil {
		t.Fatalf("mkdir root ext dir: %v", err)
	}
	if err := os.WriteFile(rootModulePath, []byte("// root"), 0o644); err != nil {
		t.Fatalf("write root module: %v", err)
	}

	contexts := []*FileProcessingContext{
		{Path: catalogPath, RelPath: "Catalogs/ТестКаталог.xml", FileName: "ТестКаталог.xml", Metadata: true, OwnerKey: "Catalog.ТестКаталог", OwnerKind: "Catalog", OwnerName: "ТестКаталог"},
		{Path: constantPath, RelPath: "Constants/ТестКонстанта.xml", FileName: "ТестКонстанта.xml", Metadata: true, OwnerKey: "Constant.ТестКонстанта", OwnerKind: "Constant", OwnerName: "ТестКонстанта"},
		{Path: filepath.Join(root, "CommonModules", "ТестОбщийМодуль.xml"), RelPath: "CommonModules/ТестОбщийМодуль.xml", FileName: "ТестОбщийМодуль.xml", Metadata: true, OwnerKey: "CommonModule.ТестОбщийМодуль", OwnerKind: "CommonModule", OwnerName: "ТестОбщийМодуль"},
		{Path: commandPath, RelPath: "CommonCommands/ТестКоманда.xml", FileName: "ТестКоманда.xml", Metadata: true, OwnerKey: "CommonCommand.ТестКоманда", OwnerKind: "CommonCommand", OwnerName: "ТестКоманда"},
		{Path: ownerPath, RelPath: "Catalogs/Пользователи.xml", FileName: "Пользователи.xml", Metadata: true, OwnerKey: "Catalog.Пользователи", OwnerKind: "Catalog", OwnerName: "Пользователи"},
	}

	decisions := map[string]objectDecision{
		"Catalog.ТестКаталог":          {Belonging: "AdoptedStub"},
		"Constant.ТестКонстанта":       {Belonging: "AdoptedStub"},
		"CommonModule.ТестОбщийМодуль": {Belonging: "AdoptedStub"},
		"CommonCommand.ТестКоманда":    {Belonging: "AdoptedStub"},
		"Catalog.Пользователи":         {Belonging: "AdoptedStub"},
	}

	searchResultState := newSearchResultState()
	searchResultState.PreservedPaths[formXMLPath] = struct{}{}

	excludedPaths := collectExcludedPaths(contexts, decisions, root, searchResultState, map[string]map[string]struct{}{
		"Catalog.Пользователи": {
			"Catalog.Пользователи.Form.ФормаВыбора":                       {},
			"Catalog.Пользователи.Command.ПользователиИнформационнойБазы": {},
		},
	}, nil)

	for _, path := range []string{
		filepath.Join(root, "Catalogs", "ТестКаталог", "Ext", "ManagerModule.bsl"),
		filepath.Join(root, "Catalogs", "ТестКаталог", "Ext", "ObjectModule.bsl"),
		valueManagerModulePath,
		commonModulePath,
		commandModulePath,
		formMetadataPath,
		formModulePath,
		childCommandModulePath,
		rootModulePath,
	} {
		if _, ok := excludedPaths[path]; !ok {
			t.Fatalf("expected %s to be collected into excluded paths", path)
		}
	}
	if _, ok := excludedPaths[formXMLPath]; ok {
		t.Fatalf("expected SearchResult-preserved path to stay out of excluded paths")
	}
}

func TestDecideObjectSoftExcludeOverridesPrimaryNative(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		OwnerKey:  "DataProcessor.упо_ФормированиеЗаказовЭлементаПлана",
		OwnerKind: "DataProcessor",
		OwnerName: "упо_ФормированиеЗаказовЭлементаПлана",
	}

	decision := decideObject(
		ctx,
		&config.Configuration{},
		nil,
		map[string]struct{}{ctx.OwnerKey: {}},
		map[string]struct{}{ctx.OwnerKey: {}},
		nil,
		nil,
	)

	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected soft exclude to override primary native decision, got %#v", decision)
	}
}

func TestDecideObjectExplicitNativeOverridesSoftExclude(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		OwnerKey:  "Constant.упо_ИспользоватьРаспределениеЗаработнойПлаты",
		OwnerKind: "Constant",
		OwnerName: "упо_ИспользоватьРаспределениеЗаработнойПлаты",
	}

	decision := decideObject(
		ctx,
		&config.Configuration{},
		map[string]struct{}{ctx.OwnerKey: {}},
		map[string]struct{}{ctx.OwnerKey: {}},
		map[string]struct{}{ctx.OwnerKey: {}},
		nil,
		nil,
	)

	if decision.Excluded || decision.Belonging != "Native" {
		t.Fatalf("expected explicit native include to override soft exclude, got %#v", decision)
	}
}

func TestDecideObjectForbiddenOverridesExplicitNative(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		OwnerKey:  "Constant.упо_ИспользоватьРаспределениеЗаработнойПлаты",
		OwnerKind: "Constant",
		OwnerName: "упо_ИспользоватьРаспределениеЗаработнойПлаты",
	}

	decision := decideObject(
		ctx,
		&config.Configuration{},
		map[string]struct{}{ctx.OwnerKey: {}},
		map[string]struct{}{ctx.OwnerKey: {}},
		map[string]struct{}{ctx.OwnerKey: {}},
		nil,
		map[string]struct{}{ctx.OwnerKey: {}},
	)

	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected forbidden object to override explicit native include, got %#v", decision)
	}
}

func TestCleanupMissingFormConstantsSetReferencesRemovesSoftExcludedConstantFromExcludedSubsystem(t *testing.T) {
	t.Parallel()

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <CheckBoxField name="РаспределениеЗП" id="1">
      <DataPath>НаборКонстант.упо_ИспользоватьРаспределениеЗаработнойПлаты</DataPath>
    </CheckBoxField>
  </ChildItems>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	constantDoc := etree.NewDocument()
	if err := constantDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Constant><Properties><Name>упо_ИспользоватьРаспределениеЗаработнойПлаты</Name></Properties></Constant>
</MetaDataObject>`); err != nil {
		t.Fatalf("read constant xml: %v", err)
	}

	constantCtx := &FileProcessingContext{
		Doc:        constantDoc,
		Metadata:   true,
		OwnerKey:   "Constant.упо_ИспользоватьРаспределениеЗаработнойПлаты",
		OwnerKind:  "Constant",
		OwnerName:  "упо_ИспользоватьРаспределениеЗаработнойПлаты",
		RelPath:    "Constants/упо_ИспользоватьРаспределениеЗаработнойПлаты.xml",
		FileName:   "упо_ИспользоватьРаспределениеЗаработнойПлаты.xml",
		Properties: constantDoc.FindElement("//Properties"),
	}

	decision := decideObject(
		constantCtx,
		&config.Configuration{},
		nil,
		map[string]struct{}{constantCtx.OwnerKey: {}},
		map[string]struct{}{constantCtx.OwnerKey: {}},
		nil,
		nil,
	)

	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected constant to stay excluded, got %#v", decision)
	}

	if !cleanupMissingFormConstantsSetReferences(formDoc, []*FileProcessingContext{constantCtx}, map[string]objectDecision{
		constantCtx.OwnerKey: decision,
	}) {
		t.Fatalf("expected constants-set binding to be removed for excluded constant")
	}

	if len(formDoc.FindElements("//CheckBoxField")) != 0 {
		t.Fatalf("expected constants-set control to be removed from form")
	}
}

func TestCleanupMissingFormConstantsSetReferencesKeepsExplicitNativeConstantFromExcludedSubsystem(t *testing.T) {
	t.Parallel()

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <CheckBoxField name="РаспределениеЗП" id="1">
      <DataPath>НаборКонстант.упо_ИспользоватьРаспределениеЗаработнойПлаты</DataPath>
    </CheckBoxField>
  </ChildItems>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	constantDoc := etree.NewDocument()
	if err := constantDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Constant><Properties><Name>упо_ИспользоватьРаспределениеЗаработнойПлаты</Name></Properties></Constant>
</MetaDataObject>`); err != nil {
		t.Fatalf("read constant xml: %v", err)
	}

	constantCtx := &FileProcessingContext{
		Doc:        constantDoc,
		Metadata:   true,
		OwnerKey:   "Constant.упо_ИспользоватьРаспределениеЗаработнойПлаты",
		OwnerKind:  "Constant",
		OwnerName:  "упо_ИспользоватьРаспределениеЗаработнойПлаты",
		RelPath:    "Constants/упо_ИспользоватьРаспределениеЗаработнойПлаты.xml",
		FileName:   "упо_ИспользоватьРаспределениеЗаработнойПлаты.xml",
		Properties: constantDoc.FindElement("//Properties"),
	}

	decision := decideObject(
		constantCtx,
		&config.Configuration{},
		map[string]struct{}{constantCtx.OwnerKey: {}},
		map[string]struct{}{constantCtx.OwnerKey: {}},
		map[string]struct{}{constantCtx.OwnerKey: {}},
		nil,
		nil,
	)

	if decision.Excluded || decision.Belonging != "Native" {
		t.Fatalf("expected explicit native constant to stay native, got %#v", decision)
	}

	if cleanupMissingFormConstantsSetReferences(formDoc, []*FileProcessingContext{constantCtx}, map[string]objectDecision{
		constantCtx.OwnerKey: decision,
	}) {
		t.Fatalf("expected constants-set binding to remain for explicit native constant")
	}

	if len(formDoc.FindElements("//CheckBoxField")) != 1 {
		t.Fatalf("expected constants-set control to remain in form")
	}
}

func TestPromoteReferencedObjectsToAdoptedStubDoesNotRestoreFromSubsystem(t *testing.T) {
	t.Parallel()

	source := &FileProcessingContext{
		OwnerKey:  "Subsystem.ИнтеграцияERP",
		OwnerKind: "Subsystem",
		OwnerName: "ИнтеграцияERP",
	}
	target := &FileProcessingContext{
		OwnerKey:  "DataProcessor.упо_ФормированиеЗаказовЭлементаПлана",
		OwnerKind: "DataProcessor",
		OwnerName: "упо_ФормированиеЗаказовЭлементаПлана",
	}

	decisions := map[string]objectDecision{
		source.OwnerKey: {Belonging: "Native"},
		target.OwnerKey: {Excluded: true},
	}

	promoteReferencedObjectsToAdoptedStub(
		[]*FileProcessingContext{source, target},
		decisions,
		&config.Configuration{},
		map[string]map[string]struct{}{
			source.OwnerKey: {target.OwnerKey: {}},
		},
		map[string]map[string]struct{}{
			target.OwnerKey: {source.OwnerKey: {}},
		},
		nil,
		nil,
		nil,
		nil,
	)

	decision := decisions[target.OwnerKey]
	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected subsystem reference to keep target excluded, got %#v", decision)
	}
}

func TestExcludedObjectReferencedOnlyBySubsystemStaysExcluded(t *testing.T) {
	t.Parallel()

	target := &FileProcessingContext{
		OwnerKey:   "DataProcessor.упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов",
		OwnerKind:  "DataProcessor",
		OwnerName:  "упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов",
		Properties: nil,
	}
	source := &FileProcessingContext{
		OwnerKey:  "Subsystem.ИнтеграцияERP",
		OwnerKind: "Subsystem",
		OwnerName: "ИнтеграцияERP",
	}

	decision := decideObject(
		target,
		&config.Configuration{},
		nil,
		nil,
		map[string]struct{}{target.OwnerKey: {}},
		nil,
		nil,
	)
	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected target to start excluded, got %#v", decision)
	}

	decisions := map[string]objectDecision{
		target.OwnerKey: decision,
		source.OwnerKey: {Belonging: "Native"},
	}

	promoteReferencedObjectsToAdoptedStub(
		[]*FileProcessingContext{source, target},
		decisions,
		&config.Configuration{},
		map[string]map[string]struct{}{
			source.OwnerKey: {target.OwnerKey: {}},
		},
		map[string]map[string]struct{}{
			target.OwnerKey: {source.OwnerKey: {}},
		},
		nil,
		nil,
		map[string]struct{}{target.OwnerKey: {}},
		nil,
	)

	decision = decisions[target.OwnerKey]
	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected excluded object with subsystem-only refs to stay excluded, got %#v", decision)
	}
}

func TestExcludedPrimaryNativeObjectReferencedBySubsystemStaysExcluded(t *testing.T) {
	t.Parallel()

	target := &FileProcessingContext{
		OwnerKey:  "DataProcessor.упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов",
		OwnerKind: "DataProcessor",
		OwnerName: "упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов",
	}
	source := &FileProcessingContext{
		OwnerKey:  "Subsystem.ОтражениеПервичныхДокументовВПроектномБюджетировании",
		OwnerKind: "Subsystem",
		OwnerName: "ОтражениеПервичныхДокументовВПроектномБюджетировании",
	}

	decisions := map[string]objectDecision{
		target.OwnerKey: {Excluded: true},
		source.OwnerKey: {Belonging: "Native"},
	}

	promoteReferencedObjectsToAdoptedStub(
		[]*FileProcessingContext{source, target},
		decisions,
		&config.Configuration{
			NativePrefixes: []string{"упо_"},
		},
		map[string]map[string]struct{}{
			source.OwnerKey: {target.OwnerKey: {}},
		},
		map[string]map[string]struct{}{
			target.OwnerKey: {source.OwnerKey: {}},
		},
		nil,
		map[string]struct{}{target.OwnerKey: {}},
		map[string]struct{}{target.OwnerKey: {}},
		nil,
	)

	decision := decisions[target.OwnerKey]
	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected excluded primary native object to stay excluded, got %#v", decision)
	}
}

func TestDecideObjectSoftExcludedBeatsPrimaryNative(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		OwnerKey:  "DataProcessor.упо_Тест",
		OwnerKind: "DataProcessor",
		OwnerName: "упо_Тест",
	}

	decision := decideObject(
		ctx,
		&config.Configuration{},
		nil,
		map[string]struct{}{ctx.OwnerKey: {}},
		map[string]struct{}{ctx.OwnerKey: {}},
		nil,
		nil,
	)
	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected excluded object to win over primary native classification, got %#v", decision)
	}
}

func TestCollectSubsystemDecisionsDoesNotAdoptFormerSpecialRootWithoutNativeDescendants(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		RelPath:   "Subsystems/СтандартныеПодсистемы.xml",
		OwnerKey:  "Subsystem.СтандартныеПодсистемы",
		OwnerKind: "Subsystem",
		OwnerName: "СтандартныеПодсистемы",
	}

	decisions := collectSubsystemDecisions([]*FileProcessingContext{ctx}, &config.Configuration{
		NativePrefixes: []string{"упо_", "слк", "упф", "pm"},
	})
	if decision, ok := decisions[ctx.OwnerKey]; ok && !decision.Excluded {
		t.Fatalf("expected non-native root subsystem without native descendants to stay excluded, got %#v", decision)
	}
}

func TestCollectSubsystemDecisionsAdoptsNonNativeRootWithNestedNativeSubsystem(t *testing.T) {
	t.Parallel()

	contexts := []*FileProcessingContext{
		{
			RelPath:   "Subsystems/Корень.xml",
			OwnerKey:  "Subsystem.Корень",
			OwnerKind: "Subsystem",
			OwnerName: "Корень",
		},
		{
			RelPath:   "Subsystems/Корень/Subsystems/упо_Нативная.xml",
			OwnerKey:  "Subsystem.Корень.Subsystem.упо_Нативная",
			OwnerKind: "Subsystem",
			OwnerName: "упо_Нативная",
		},
	}

	decisions := collectSubsystemDecisions(contexts, &config.Configuration{
		NativePrefixes: []string{"упо_"},
	})
	if decision := decisions["Subsystem.Корень"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected non-native root with nested native subsystem to become AdoptedStub, got %#v", decision)
	}
	if decision := decisions["Subsystem.Корень.Subsystem.упо_Нативная"]; decision.Excluded || decision.Belonging != "Native" {
		t.Fatalf("expected nested prefixed subsystem to stay Native, got %#v", decision)
	}
}

func TestIsRefDrivenInclusionSourceRejectsAdoptedSubsystem(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		OwnerKey:  "Subsystem.Тест",
		OwnerKind: "Subsystem",
		OwnerName: "Тест",
	}
	if isRefDrivenInclusionSource(ctx, objectDecision{Belonging: "AdoptedStub"}) {
		t.Fatalf("expected adopted subsystem not to be ref-driven source")
	}
	if !isRefDrivenInclusionSource(ctx, objectDecision{Belonging: "Native"}) {
		t.Fatalf("expected native subsystem to remain ref-driven source")
	}
}

func TestNonNativeVersioningSubsystemDoesNotRestoreEventSubscriptionsOrHandlerModule(t *testing.T) {
	t.Parallel()

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <EventSubscription>ЗаписатьВерсиюОбъекта</EventSubscription>
      <EventSubscription>ОчиститьИнформациюОбАвтореВерсии</EventSubscription>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}

	subsystemDoc := etree.NewDocument()
	if err := subsystemDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <Subsystem>
    <Properties>
      <Name>ВерсионированиеОбъектов</Name>
      <Content>
        <xr:Item xsi:type="xr:MDObjectRef">EventSubscription.ЗаписатьВерсиюОбъекта</xr:Item>
        <xr:Item xsi:type="xr:MDObjectRef">EventSubscription.ОчиститьИнформациюОбАвтореВерсии</xr:Item>
        <xr:Item xsi:type="xr:MDObjectRef">CommonModule.ВерсионированиеОбъектовСобытия</xr:Item>
      </Content>
    </Properties>
  </Subsystem>
</MetaDataObject>`); err != nil {
		t.Fatalf("read subsystem xml: %v", err)
	}

	eventSubscriptionDoc := etree.NewDocument()
	if err := eventSubscriptionDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <EventSubscription>
    <Properties>
      <Name>ЗаписатьВерсиюОбъекта</Name>
      <Handler>CommonModule.ВерсионированиеОбъектовСобытия.ЗаписатьВерсиюОбъекта</Handler>
    </Properties>
  </EventSubscription>
</MetaDataObject>`); err != nil {
		t.Fatalf("read event subscription xml: %v", err)
	}

	eventSubscriptionCleanupDoc := etree.NewDocument()
	if err := eventSubscriptionCleanupDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <EventSubscription>
    <Properties>
      <Name>ОчиститьИнформациюОбАвтореВерсии</Name>
      <Handler>CommonModule.ВерсионированиеОбъектовСобытия.УдалитьИнформациюОбАвтореВерсии</Handler>
    </Properties>
  </EventSubscription>
</MetaDataObject>`); err != nil {
		t.Fatalf("read cleanup event subscription xml: %v", err)
	}

	commonModuleDoc := etree.NewDocument()
	if err := commonModuleDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <CommonModule>
    <Properties>
      <Name>ВерсионированиеОбъектовСобытия</Name>
    </Properties>
  </CommonModule>
</MetaDataObject>`); err != nil {
		t.Fatalf("read common module xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			OwnerKey:  "Configuration",
			OwnerKind: "Configuration",
			Doc:       configurationDoc,
		},
		{
			OwnerKey:  "Subsystem.ВерсионированиеОбъектов",
			OwnerKind: "Subsystem",
			OwnerName: "ВерсионированиеОбъектов",
			Doc:       subsystemDoc,
		},
		{
			OwnerKey:  "EventSubscription.ЗаписатьВерсиюОбъекта",
			OwnerKind: "EventSubscription",
			OwnerName: "ЗаписатьВерсиюОбъекта",
			Doc:       eventSubscriptionDoc,
		},
		{
			OwnerKey:  "EventSubscription.ОчиститьИнформациюОбАвтореВерсии",
			OwnerKind: "EventSubscription",
			OwnerName: "ОчиститьИнформациюОбАвтореВерсии",
			Doc:       eventSubscriptionCleanupDoc,
		},
		{
			OwnerKey:  "CommonModule.ВерсионированиеОбъектовСобытия",
			OwnerKind: "CommonModule",
			OwnerName: "ВерсионированиеОбъектовСобытия",
			Doc:       commonModuleDoc,
		},
	}

	decisions := map[string]objectDecision{
		"Configuration": {Belonging: "AdoptedStub"},
		"Subsystem.ВерсионированиеОбъектов":                  {Belonging: "AdoptedStub"},
		"EventSubscription.ЗаписатьВерсиюОбъекта":            {Excluded: true},
		"EventSubscription.ОчиститьИнформациюОбАвтореВерсии": {Excluded: true},
		"CommonModule.ВерсионированиеОбъектовСобытия":        {Excluded: true},
	}

	referenceGraph := collectReferenceGraph(contexts, &config.Configuration{}, nil, decisions, nil)
	incomingReferenceGraph := collectIncomingReferenceGraph(referenceGraph)

	promoteReferencedObjectsToAdoptedStub(
		contexts,
		decisions,
		&config.Configuration{},
		referenceGraph,
		incomingReferenceGraph,
		nil,
		nil,
		nil,
		nil,
	)

	if decision := decisions["EventSubscription.ЗаписатьВерсиюОбъекта"]; !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected non-native subsystem and configuration to keep event subscription excluded, got %#v", decision)
	}
	if decision := decisions["EventSubscription.ОчиститьИнформациюОбАвтореВерсии"]; !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected cleanup event subscription to stay excluded, got %#v", decision)
	}
	if decision := decisions["CommonModule.ВерсионированиеОбъектовСобытия"]; !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected handler module to stay excluded when only non-native sources reference it, got %#v", decision)
	}
}

func TestNormalizeSubsystemContentRemovesMissingMetadataRefs(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <Subsystem>
    <Properties>
      <Name>СлужебнаяПодсистема</Name>
      <Content>
        <xr:Item xsi:type="xr:MDObjectRef">Catalog.Партнеры</xr:Item>
        <xr:Item xsi:type="xr:MDObjectRef">DataProcessor.ОперацииЗакрытияМесяца</xr:Item>
        <xr:Item xsi:type="xr:MDObjectRef">Constant.ИспользоватьКоннект</xr:Item>
      </Content>
    </Properties>
    <ChildObjects/>
  </Subsystem>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Metadata:  true,
			RelPath:   "Subsystems/СлужебнаяПодсистема.xml",
			OwnerKey:  "Subsystem.СлужебнаяПодсистема",
			OwnerKind: "Subsystem",
		},
		{
			Metadata:  true,
			RelPath:   "Catalogs/Партнеры.xml",
			OwnerKey:  "Catalog.Партнеры",
			OwnerKind: "Catalog",
		},
		{
			Metadata:  true,
			RelPath:   "DataProcessors/ОперацииЗакрытияМесяца.xml",
			OwnerKey:  "DataProcessor.ОперацииЗакрытияМесяца",
			OwnerKind: "DataProcessor",
		},
		{
			Metadata:  true,
			RelPath:   "Constants/ИспользоватьКоннект.xml",
			OwnerKey:  "Constant.ИспользоватьКоннект",
			OwnerKind: "Constant",
		},
	}

	decisions := map[string]objectDecision{
		"Subsystem.СлужебнаяПодсистема":        {Belonging: "Native"},
		"Catalog.Партнеры":                     {Belonging: "Native"},
		"DataProcessor.ОперацииЗакрытияМесяца": {Excluded: true},
		"Constant.ИспользоватьКоннект":         {Excluded: true},
	}

	changed := normalizeSubsystemContent(doc, contexts, decisions)
	if !changed {
		t.Fatalf("expected subsystem content cleanup to remove missing refs")
	}

	items := doc.Root().FindElements(".//Properties/Content/*")
	if len(items) != 1 {
		t.Fatalf("expected one subsystem content item to remain, got %d", len(items))
	}

	if strings.TrimSpace(items[0].Text()) != "Catalog.Партнеры" {
		t.Fatalf("unexpected remaining subsystem content ref: %q", strings.TrimSpace(items[0].Text()))
	}
}

func TestPromoteReferencedObjectsToAdoptedStubDoesNotRestoreFromRole(t *testing.T) {
	t.Parallel()

	source := &FileProcessingContext{
		OwnerKey:  "Role.упо_ПросмотрСопоставлениеФинансовыхСтатейСтатьямБюджетов",
		OwnerKind: "Role",
		OwnerName: "упо_ПросмотрСопоставлениеФинансовыхСтатейСтатьямБюджетов",
	}
	target := &FileProcessingContext{
		OwnerKey:  "DataProcessor.упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов",
		OwnerKind: "DataProcessor",
		OwnerName: "упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов",
	}

	decisions := map[string]objectDecision{
		source.OwnerKey: {Belonging: "Native"},
		target.OwnerKey: {Excluded: true},
	}

	promoteReferencedObjectsToAdoptedStub(
		[]*FileProcessingContext{source, target},
		decisions,
		&config.Configuration{},
		map[string]map[string]struct{}{
			source.OwnerKey: {target.OwnerKey: {}},
		},
		map[string]map[string]struct{}{
			target.OwnerKey: {source.OwnerKey: {}},
		},
		nil,
		map[string]struct{}{target.OwnerKey: {}},
		nil,
		nil,
	)

	decision := decisions[target.OwnerKey]
	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected role reference to keep target excluded, got %#v", decision)
	}
}

func TestCollectExcludedSubsystemObjectsIncludesOwnerSubsystemReference(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject>
  <DataProcessor>
    <Properties>
      <Name>упо_ТестОбъект</Name>
      <Subsystems>
        <Subsystem>упо_УправлениеПроектамиPM.Интеграция.ИнтеграцияERP</Subsystem>
      </Subsystems>
    </Properties>
  </DataProcessor>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:        doc,
			Metadata:   true,
			Properties: doc.FindElement("//Properties"),
			OwnerKey:   "DataProcessor.упо_ТестОбъект",
			OwnerKind:  "DataProcessor",
			OwnerName:  "упо_ТестОбъект",
		},
	}

	excluded := collectExcludedSubsystemObjects(contexts, []string{"упо_УправлениеПроектамиPM.Интеграция.ИнтеграцияERP"}, nil)
	if _, ok := excluded["DataProcessor.упо_ТестОбъект"]; !ok {
		t.Fatalf("expected object to be collected into excluded set, got %#v", excluded)
	}
}

func TestNormalizeManualQueryWithoutMainTableKeepsChildItemDataPaths(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <Table name="Список" id="1">
      <DataPath>Список</DataPath>
      <ChildItems>
        <LabelField name="Колонка1" id="2">
          <DataPath>Список.Наименование</DataPath>
        </LabelField>
        <LabelField name="Колонка2" id="3">
          <DataPath>~Список.ФинансоваяСтатья</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Список" id="10">
      <Type><v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:DynamicList</v8:Type></Type>
      <Settings xsi:type="DynamicList" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
        <ManualQuery>true</ManualQuery>
        <Field>
          <dataPath>Наименование</dataPath>
          <field>Наименование</field>
        </Field>
        <Field>
          <dataPath>ФинансоваяСтатья</dataPath>
          <field>ФинансоваяСтатья</field>
        </Field>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if normalizeManualQueryWithoutMainTable(doc) {
		t.Fatalf("expected form normalization to keep child item data paths unchanged")
	}

	values := []string{}
	for _, el := range doc.FindElements("//Table/ChildItems//DataPath") {
		values = append(values, el.Text())
	}

	if len(values) != 2 || values[0] != "Список.Наименование" || values[1] != "~Список.ФинансоваяСтатья" {
		t.Fatalf("unexpected child item data paths: %#v", values)
	}
}

func TestCleanupNonNativeManualQueryOrphanReferencesRemovesUndeclaredListBindings(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="Список">
      <DataPath>Список</DataPath>
      <RowPictureDataPath>Список.НомерКартинки</RowPictureDataPath>
      <ChildItems>
        <LabelField name="Наименование">
          <DataPath>Список.Наименование</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
    <UsualGroup name="Комментарий">
      <TitleDataPath>Items.Список.CurrentData.Наименование</TitleDataPath>
      <ChildItems>
        <InputField name="КомментарийПоле">
          <DataPath>Items.Список.CurrentData.Комментарий</DataPath>
        </InputField>
      </ChildItems>
    </UsualGroup>
  </ChildItems>
  <Attributes>
    <Attribute name="Список">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <UseAlways>
        <Field>Список.Ссылка</Field>
      </UseAlways>
      <Settings xsi:type="DynamicList">
        <ManualQuery>true</ManualQuery>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if !cleanupNonNativeManualQueryOrphanReferences(doc) {
		t.Fatalf("expected orphan manual-query bindings to be removed")
	}

	if doc.FindElement("//RowPictureDataPath") != nil {
		t.Fatalf("expected row picture data path to be removed")
	}
	if doc.FindElement("//TitleDataPath") != nil {
		t.Fatalf("expected title data path to be removed")
	}
	if len(doc.FindElements("//UseAlways/Field")) != 0 {
		t.Fatalf("expected UseAlways field to be removed")
	}
	if len(doc.FindElements("//InputField")) != 0 {
		t.Fatalf("expected input field bound to orphan current data to be removed")
	}
	if len(doc.FindElements("//LabelField")) != 0 {
		t.Fatalf("expected label field bound to orphan list field to be removed")
	}
}

func TestCleanupNonNativeManualQueryOrphanReferencesKeepsSchemaDeclaredField(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="Список">
      <DataPath>Список</DataPath>
      <ChildItems>
        <LabelField name="RefField">
          <DataPath>Список.Ref</DataPath>
        </LabelField>
      </ChildItems>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Список">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <Settings xsi:type="DynamicList">
        <ManualQuery>true</ManualQuery>
        <Field>
          <dataPath>Ref</dataPath>
          <field>Ref</field>
        </Field>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if cleanupNonNativeManualQueryOrphanReferences(doc) {
		t.Fatalf("expected schema-declared field binding to stay")
	}
	if got := textOfFirst(doc.Root(), ".//LabelField/DataPath"); got != "Список.Ref" {
		t.Fatalf("unexpected remaining data path: %q", got)
	}
}

func TestCleanupMissingFormConstantsSetReferencesRemovesMissingConstantControl(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <CheckBoxField name="Удаляемый" id="1">
      <DataPath>НаборКонстант.упо_ИспользоватьРаспределениеЗаработнойПлаты</DataPath>
    </CheckBoxField>
    <CheckBoxField name="Оставляемый" id="2">
      <DataPath>НаборКонстант.ИспользоватьКоннект</DataPath>
    </CheckBoxField>
  </ChildItems>
</Form>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	constantDoc := etree.NewDocument()
	if err := constantDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Constant><Properties><Name>ИспользоватьКоннект</Name></Properties></Constant>
</MetaDataObject>`); err != nil {
		t.Fatalf("read constant xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:        constantDoc,
			Metadata:   true,
			OwnerKey:   "Constant.ИспользоватьКоннект",
			OwnerKind:  "Constant",
			OwnerName:  "ИспользоватьКоннект",
			RelPath:    "Constants/ИспользоватьКоннект.xml",
			FileName:   "ИспользоватьКоннект.xml",
			Properties: constantDoc.FindElement("//Properties"),
		},
	}

	decisions := map[string]objectDecision{
		"Constant.ИспользоватьКоннект": {Belonging: "Native"},
	}

	if !cleanupMissingFormConstantsSetReferences(doc, contexts, decisions) {
		t.Fatalf("expected missing constant control to be removed")
	}

	names := []string{}
	for _, el := range doc.FindElements("//ChildItems/*") {
		names = append(names, el.SelectAttrValue("name", ""))
	}

	if len(names) != 1 || names[0] != "Оставляемый" {
		t.Fatalf("unexpected remaining form items: %#v", names)
	}
}

func TestCleanupMissingFormConstantsSetReferencesRemovesMissingConstantField(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <Attributes>
    <Attribute name="НаборКонстант" id="1">
      <UseAlways>
        <Field>НаборКонстант.УдаляемаяКонстанта</Field>
        <Field>НаборКонстант.ОставляемаяКонстанта</Field>
      </UseAlways>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	constantDoc := etree.NewDocument()
	if err := constantDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Constant><Properties><Name>ОставляемаяКонстанта</Name></Properties></Constant>
</MetaDataObject>`); err != nil {
		t.Fatalf("read constant xml: %v", err)
	}

	contexts := []*FileProcessingContext{{
		Doc:        constantDoc,
		Metadata:   true,
		OwnerKey:   "Constant.ОставляемаяКонстанта",
		OwnerKind:  "Constant",
		OwnerName:  "ОставляемаяКонстанта",
		RelPath:    "Constants/ОставляемаяКонстанта.xml",
		FileName:   "ОставляемаяКонстанта.xml",
		Properties: constantDoc.FindElement("//Properties"),
	}}

	decisions := map[string]objectDecision{
		"Constant.ОставляемаяКонстанта": {Belonging: "Native"},
	}

	if !cleanupMissingFormConstantsSetReferences(doc, contexts, decisions) {
		t.Fatalf("expected missing constant field to be removed")
	}

	fields := []string{}
	for _, el := range doc.FindElements("//Attribute[@name='НаборКонстант']//Field") {
		fields = append(fields, strings.TrimSpace(el.Text()))
	}
	if len(fields) != 1 || fields[0] != "НаборКонстант.ОставляемаяКонстанта" {
		t.Fatalf("unexpected remaining fields: %#v", fields)
	}
}

func TestCleanupMissingFormOwnerObjectReferencesRemovesUnavailableObjectField(t *testing.T) {
	t.Parallel()

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <InputField name="Недействителен"><DataPath>Объект.Недействителен</DataPath></InputField>
    <InputField name="Комментарий"><DataPath>Объект.Комментарий</DataPath></InputField>
  </ChildItems>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties><Name>Пользователи</Name></Properties>
    <ChildObjects>
      <Attribute><Properties><Name>Недействителен</Name></Properties></Attribute>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	ownerCtx := &FileProcessingContext{
		Doc:        ownerDoc,
		Metadata:   true,
		OwnerKey:   "Catalog.Пользователи",
		OwnerKind:  "Catalog",
		OwnerName:  "Пользователи",
		RelPath:    "Catalogs/Пользователи.xml",
		FileName:   "Пользователи.xml",
		Properties: ownerDoc.FindElement("//Properties"),
	}

	if !cleanupMissingFormOwnerObjectReferences(formDoc, ownerCtx) {
		t.Fatalf("expected unavailable object field to be removed")
	}

	names := []string{}
	for _, el := range formDoc.FindElements("//ChildItems/*") {
		names = append(names, el.SelectAttrValue("name", ""))
	}
	if len(names) != 1 || names[0] != "Недействителен" {
		t.Fatalf("unexpected remaining object fields: %#v", names)
	}
}

func TestCleanupMissingFormOwnerObjectReferencesWithoutRetainedFieldsRemovesOwnerTablesAndAdditionalColumns(t *testing.T) {
	t.Parallel()

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <InputField name="Описание"><DataPath>Объект.Description</DataPath></InputField>
    <InputField name="Ответственный"><DataPath>Объект.Ответственный</DataPath></InputField>
    <Table name="ВидыДоступа"><DataPath>Объект.ВидыДоступа</DataPath></Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Объект">
      <Columns>
        <AdditionalColumns table="Объект.ВидыДоступа">
          <Column name="Тест"/>
        </AdditionalColumns>
      </Columns>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties><Name>ГруппыДоступа</Name></Properties>
    <ChildObjects>
      <Form>ФормаЭлемента</Form>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	ownerCtx := &FileProcessingContext{
		Doc:        ownerDoc,
		Metadata:   true,
		OwnerKey:   "Catalog.ГруппыДоступа",
		OwnerKind:  "Catalog",
		OwnerName:  "ГруппыДоступа",
		RelPath:    "Catalogs/ГруппыДоступа.xml",
		FileName:   "ГруппыДоступа.xml",
		Properties: ownerDoc.FindElement("//Properties"),
	}

	if !cleanupMissingFormOwnerObjectReferences(formDoc, ownerCtx) {
		t.Fatalf("expected owner object cleanup to remove missing owner fields and columns")
	}

	if hasFormElementWithName(formDoc, "InputField", "Ответственный") {
		t.Fatalf("expected missing owner input field to be removed")
	}
	if hasFormElementWithName(formDoc, "Table", "ВидыДоступа") {
		t.Fatalf("expected missing owner table to be removed")
	}
	if formDoc.FindElement("//*[local-name()='AdditionalColumns']") != nil {
		t.Fatalf("expected additional columns for missing owner table to be removed")
	}
	if !hasFormElementWithName(formDoc, "InputField", "Описание") {
		t.Fatalf("expected standard owner Description field to stay")
	}
}

func hasFormElementWithName(doc *etree.Document, tag string, name string) bool {
	if doc == nil {
		return false
	}
	for _, el := range doc.FindElements("//*") {
		if !strings.EqualFold(localName(el.Tag), tag) {
			continue
		}
		if el.SelectAttrValue("name", "") == name {
			return true
		}
	}
	return false
}

func TestCleanupNonNativeFormStandardCommandsAndEvents(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <Events>
    <Event name="AfterWrite">ПослеЗаписи</Event>
    <Event name="OnOpen">ПриОткрытии</Event>
  </Events>
  <ChildItems>
    <Button name="Удаляемая"><CommandName>Form.StandardCommand.Create</CommandName></Button>
    <Button name="Оставляемая"><CommandName>Form.StandardCommand.Help</CommandName></Button>
  </ChildItems>
  <ExcludedCommand>WriteAndClose</ExcludedCommand>
</Form>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if !cleanupNonNativeFormLifecycleEvents(doc) {
		t.Fatalf("expected non-native lifecycle event cleanup")
	}
	if !cleanupNonNativeFormStandardCommands(doc) {
		t.Fatalf("expected non-native standard command cleanup")
	}

	if events := doc.FindElements("//Events/Event"); len(events) != 1 || events[0].SelectAttrValue("name", "") != "OnOpen" {
		t.Fatalf("unexpected remaining events")
	}

	names := []string{}
	for _, el := range doc.FindElements("//ChildItems/*") {
		names = append(names, el.SelectAttrValue("name", ""))
	}
	if len(names) != 1 || names[0] != "Оставляемая" {
		t.Fatalf("unexpected remaining buttons: %#v", names)
	}
	if !formDocContainsText(doc, "WriteAndClose") {
		t.Fatalf("expected excluded command composition to remain untouched")
	}
}

func TestCleanupMissingFormCommandReferencesKeepsTableExcludedCommands(t *testing.T) {
	t.Parallel()

	doc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <Table name="Список">
      <DataPath>Список</DataPath>
      <CommandSet>
        <ExcludedCommand>LevelDown</ExcludedCommand>
        <ExcludedCommand>LevelUp</ExcludedCommand>
      </CommandSet>
      <AutoCommandBar name="Панель">
        <ChildItems>
          <Button name="Вниз">
            <Type>CommandBarButton</Type>
            <CommandName>Form.Item.Список.StandardCommand.LevelDown</CommandName>
          </Button>
        </ChildItems>
      </AutoCommandBar>
    </Table>
  </ChildItems>
</Form>`)

	if !cleanupMissingFormCommandReferencesWithResolver(doc, func(string) bool { return true }) {
		t.Fatalf("expected invalid form command reference cleanup to remove explicit UI command reference")
	}

	if formDocContainsText(doc, "Form.Item.Список.StandardCommand.LevelDown") {
		t.Fatalf("expected explicit invalid LevelDown UI command reference to be removed")
	}
	for _, command := range []string{"LevelDown", "LevelUp"} {
		if !formDocContainsText(doc, command) {
			t.Fatalf("expected excluded command %q to remain in table command set", command)
		}
	}
}

func TestCleanupMissingFormCommandReferencesRemovesDanglingShortNames(t *testing.T) {
	t.Parallel()

	doc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <Button name="Delete"><CommandName>Delete</CommandName></Button>
    <Button name="Change"><CommandName>Change</CommandName></Button>
    <Button name="Copy"><CommandName>Copy</CommandName></Button>
    <Button name="Create"><CommandName>Create</CommandName></Button>
    <Button name="MoveItem"><CommandName>MoveItem</CommandName></Button>
    <Button name="Локальная"><CommandName>ЛокальнаяКоманда</CommandName></Button>
    <Button name="КомандаОбъекта"><CommandName>КомандаОбъекта</CommandName></Button>
  </ChildItems>
  <Commands>
    <Command name="ЛокальнаяКоманда">
      <Action>ЛокальнаяКоманда</Action>
    </Command>
  </Commands>
</Form>`)

	resolver := formCommandReferenceResolver("Catalog.Тест", func(name string) bool {
		return name == "Catalog.Тест.Command.КомандаОбъекта"
	})

	if !cleanupMissingFormCommandReferencesWithResolver(doc, resolver) {
		t.Fatalf("expected dangling short command references to be removed")
	}

	for _, command := range []string{"Delete", "Change", "Copy", "Create", "MoveItem"} {
		if formDocContainsText(doc, command) {
			t.Fatalf("expected dangling short command %q to be removed", command)
		}
	}
	for _, command := range []string{"ЛокальнаяКоманда", "КомандаОбъекта"} {
		if !formDocContainsText(doc, command) {
			t.Fatalf("expected valid short command %q to remain", command)
		}
	}
}

func TestRemoveInvalidFormLevelExcludedStandardCommandsKeepsNestedTableCommands(t *testing.T) {
	t.Parallel()

	doc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/data" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <CommandSet>
    <ExcludedCommand>Change</ExcludedCommand>
    <ExcludedCommand>WriteAndClose</ExcludedCommand>
  </CommandSet>
  <ChildItems>
    <Table name="Список">
      <Title>
        <v8:item>
          <v8:lang>ru</v8:lang>
          <v8:content>Список</v8:content>
        </v8:item>
      </Title>
      <CommandSet>
        <ExcludedCommand>CancelSearch</ExcludedCommand>
        <ExcludedCommand>Change</ExcludedCommand>
        <ExcludedCommand>Choose</ExcludedCommand>
        <ExcludedCommand>Copy</ExcludedCommand>
        <ExcludedCommand>Create</ExcludedCommand>
        <ExcludedCommand>Delete</ExcludedCommand>
        <ExcludedCommand>MoveItem</ExcludedCommand>
        <ExcludedCommand>Refresh</ExcludedCommand>
      </CommandSet>
    </Table>
  </ChildItems>
</Form>`)

	if !removeInvalidFormLevelExcludedStandardCommands(doc) {
		t.Fatalf("expected root invalid excluded standard commands to be removed")
	}

	if doc.Root().FindElement("./*[local-name()='CommandSet']") != nil {
		t.Fatalf("expected root command set with only invalid commands to be removed")
	}
	for _, command := range []string{"CancelSearch", "Change", "Choose", "Copy", "Create", "Delete", "MoveItem", "Refresh"} {
		if !formDocContainsText(doc, command) {
			t.Fatalf("expected nested excluded command %q to remain", command)
		}
	}
}

func TestRemoveInvalidFormLevelExcludedStandardCommandsDoesNothingForNestedOnlyCommandSets(t *testing.T) {
	t.Parallel()

	doc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/data">
  <ChildItems>
    <Table name="ГруппыПользователей">
      <Title>
        <v8:item>
          <v8:lang>ru</v8:lang>
          <v8:content>Группы пользователей</v8:content>
        </v8:item>
      </Title>
      <CommandSet>
        <ExcludedCommand>Delete</ExcludedCommand>
      </CommandSet>
      <CurrentRowUse>SelectionPresentationAndChoice</CurrentRowUse>
    </Table>
  </ChildItems>
</Form>`)

	if removeInvalidFormLevelExcludedStandardCommands(doc) {
		t.Fatalf("expected nested-only command set to stay untouched")
	}

	if !formDocContainsText(doc, "Delete") {
		t.Fatalf("expected nested invalid excluded command to remain")
	}
	if doc.FindElement("//*[local-name()='Table']/*[local-name()='CommandSet']") == nil {
		t.Fatalf("expected nested command set container to remain")
	}
}

func TestRemoveInvalidChoiceTableExcludedStandardCommandsKeepsSelectionFormCommands(t *testing.T) {
	t.Parallel()

	doc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <CommandSet>
    <ExcludedCommand>Change</ExcludedCommand>
  </CommandSet>
  <ChildItems>
    <Table name="Список">
      <ChoiceMode>true</ChoiceMode>
      <DataPath>Список</DataPath>
      <CommandSet>
        <ExcludedCommand>Change</ExcludedCommand>
      </CommandSet>
    </Table>
  </ChildItems>
</Form>`)

	if removeInvalidChoiceTableExcludedStandardCommands(doc, true) {
		t.Fatalf("expected selection-form choice-table command to stay untouched")
	}

	tableCommandSet := doc.Root().FindElement(".//*[local-name()='Table']/*[local-name()='CommandSet']")
	if tableCommandSet == nil {
		t.Fatalf("expected selection-form choice-table command set to remain")
	}
	if _, ok := collectExcludedCommands(tableCommandSet)["Change"]; !ok {
		t.Fatalf("expected selection-form choice-table command to remain")
	}
}

func TestRemoveInvalidChoiceTableExcludedStandardCommandsMarksChangedWhenRemovingEmptyCommandSet(t *testing.T) {
	t.Parallel()

	doc := mustReadTestXML(t, `<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <Table name="ГруппыПользователей">
      <ChoiceMode>true</ChoiceMode>
      <DataPath>ГруппыПользователей</DataPath>
      <CommandSet>
        <ExcludedCommand>Delete</ExcludedCommand>
      </CommandSet>
    </Table>
  </ChildItems>
</Form>`)

	if !removeInvalidChoiceTableExcludedStandardCommands(doc, false) {
		t.Fatalf("expected cleanup to report document change when removing last invalid choice-table command")
	}

	if doc.Root().FindElement(".//*[local-name()='Table']/*[local-name()='CommandSet']") != nil {
		t.Fatalf("expected empty choice-table command set container to be removed")
	}
}

func TestNormalizeManualQueryWithoutMainTableAddsStandardAliasAndRemovesDefaultPicture(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <ChildItems>
    <Table name="Список" id="1">
      <DataPath>Список</DataPath>
      <CommandSet>
        <ExcludedCommand>Change</ExcludedCommand>
        <ExcludedCommand>ChangeHistory</ExcludedCommand>
        <ExcludedCommand>GetURL</ExcludedCommand>
        <ExcludedCommand>LevelDown</ExcludedCommand>
        <ExcludedCommand>LevelUp</ExcludedCommand>
      </CommandSet>
      <RowPictureDataPath>Список.DefaultPicture</RowPictureDataPath>
    </Table>
  </ChildItems>
  <Attributes>
    <Attribute name="Список" id="1">
      <Type><v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:DynamicList</v8:Type></Type>
      <UseAlways>
        <Field>Список.Ref</Field>
      </UseAlways>
      <Settings xsi:type="DynamicList">
        <ManualQuery>true</ManualQuery>
        <QueryText>ВЫБРАТЬ
  Справочник.Ссылка,
  Справочник.ПометкаУдаления
ИЗ
  Справочник.Пользователи КАК Справочник</QueryText>
      </Settings>
    </Attribute>
  </Attributes>
</Form>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if !normalizeManualQueryWithoutMainTable(doc) {
		t.Fatalf("expected manual query normalization to change form")
	}

	queryText := textOfFirst(doc.Root(), ".//Attribute[@name='Список']//QueryText")
	if !strings.Contains(queryText, "Справочник.Ссылка КАК Ref") {
		t.Fatalf("expected query to get standard alias, got:\n%s", queryText)
	}
	if doc.FindElement("//Table/RowPictureDataPath") != nil {
		t.Fatalf("expected default picture row data path to be removed")
	}
	for _, command := range []string{"Change", "ChangeHistory", "GetURL", "LevelDown", "LevelUp"} {
		if !formDocContainsText(doc, command) {
			t.Fatalf("expected table command-set excluded command %q to remain", command)
		}
	}
}

func TestCleanupRoleExcludedMetadataRightsRemovesExcludedObjects(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Rights xmlns="http://v8.1c.ru/8.2/roles">
  <object>
    <name>Configuration.Демо</name>
  </object>
  <object>
    <name>DataProcessor.упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов</name>
  </object>
  <object>
    <name>DataProcessor.упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов.Command.СопоставлениеФинансовыхСтатей</name>
  </object>
</Rights>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	changed := cleanupRoleExcludedMetadataRights(doc, map[string]objectDecision{
		"DataProcessor.упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов": {Excluded: true},
	})
	if !changed {
		t.Fatalf("expected excluded rights to be removed")
	}

	names := []string{}
	for _, el := range doc.FindElements("//object/name") {
		names = append(names, el.Text())
	}

	if len(names) != 1 || names[0] != "Configuration.Демо" {
		t.Fatalf("unexpected remaining role rights: %#v", names)
	}
}

func TestCleanupExcludedReferencesRemovesAttributeWithExcludedType(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <Document>
    <ChildObjects>
      <Attribute>
        <Properties>
          <Name>упо_НачислениеPM</Name>
          <Type>
            <v8:Type>cfg:CatalogRef.упо_НачисленияPM</v8:Type>
          </Type>
        </Properties>
      </Attribute>
      <Attribute>
        <Properties>
          <Name>ОбычныйРеквизит</Name>
          <Type>
            <v8:Type>xs:string</v8:Type>
          </Type>
        </Properties>
      </Attribute>
    </ChildObjects>
  </Document>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	excluded := map[string]map[string]struct{}{
		"Catalog": {
			"упо_НачисленияPM": {},
		},
	}

	changed := cleanupExcludedReferences(doc, excluded, collectExcludedMetadataPrefixes(excluded), nil, nil)
	if !changed {
		t.Fatalf("expected excluded type cleanup to remove attribute")
	}

	names := []string{}
	for _, el := range doc.FindElements("//ChildObjects/Attribute/Properties/Name") {
		names = append(names, el.Text())
	}

	if len(names) != 1 || names[0] != "ОбычныйРеквизит" {
		t.Fatalf("unexpected remaining attribute names: %#v", names)
	}
}

func TestMetadataReferencesFromValueSupportsCurrentConfigNamespaceAlias(t *testing.T) {
	t.Parallel()

	refs := metadataReferencesFromValue(`d4p1:CatalogRef.упо_СтатьиБДР`)
	if len(refs) != 1 || refs[0] != "Catalog.упо_СтатьиБДР" {
		t.Fatalf("expected d4p1 current-config ref to resolve, got %#v", refs)
	}
}

func TestContainsMetadataReferenceSupportsCurrentConfigAlias(t *testing.T) {
	t.Parallel()

	if !containsMetadataReference(`d4p1:CatalogRef.упо_СтатьиБДР`, "Catalog.упо_СтатьиБДР") {
		t.Fatalf("expected current-config alias to match metadata prefix")
	}
}

func TestContainsMetadataReferenceMatchesChildPath(t *testing.T) {
	t.Parallel()

	value := `Catalog.Заказы.Command.КомандаПечати`
	if !containsMetadataReference(value, "Catalog.Заказы.Command.КомандаПечати") {
		t.Fatalf("expected child metadata path to match exactly")
	}
}

func TestCleanupExcludedReferencesRemovesPredefinedItemWithExcludedType(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<PredefinedData xmlns="http://v8.1c.ru/8.3/xcf/predef" xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <Item>
    <Name>ПлохойЭлемент</Name>
    <Type>
      <v8:Type xmlns:d4p1="http://v8.1c.ru/8.1/data/enterprise/current-config">d4p1:CatalogRef.упо_НачисленияPM</v8:Type>
    </Type>
  </Item>
  <Item>
    <Name>ХорошийЭлемент</Name>
    <Type>
      <v8:Type xmlns:d4p1="http://v8.1c.ru/8.1/data/enterprise/current-config">d4p1:CatalogRef.упо_СтатьиБДР</v8:Type>
    </Type>
  </Item>
</PredefinedData>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	excluded := map[string]map[string]struct{}{
		"Catalog": {
			"упо_НачисленияPM": {},
		},
	}

	changed := cleanupExcludedReferences(doc, excluded, collectExcludedMetadataPrefixes(excluded), nil, nil)
	if !changed {
		t.Fatalf("expected predefined cleanup to remove excluded item")
	}

	items := doc.Root().FindElements("./Item")
	if len(items) != 1 {
		t.Fatalf("expected one predefined item to remain, got %d", len(items))
	}

	name := items[0].FindElement("./Name")
	if name == nil || name.Text() != "ХорошийЭлемент" {
		t.Fatalf("unexpected remaining predefined item")
	}
}

func TestCleanupExcludedReferencesRemovesExcludedTypeValueFromPropertiesType(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <Properties>
    <Type>
      <v8:Type>cfg:CatalogRef.упо_НачисленияPM</v8:Type>
      <v8:Type>cfg:CatalogRef.упо_СтатьиБДР</v8:Type>
    </Type>
  </Properties>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	excluded := map[string]map[string]struct{}{
		"Catalog": {
			"упо_НачисленияPM": {},
		},
	}

	changed := cleanupExcludedReferences(doc, excluded, collectExcludedMetadataPrefixes(excluded), nil, nil)
	if !changed {
		t.Fatalf("expected type cleanup to remove excluded v8:Type")
	}

	types := doc.Root().FindElements("./Properties/Type/*")
	if len(types) != 1 {
		t.Fatalf("expected one type to remain, got %d", len(types))
	}
	if strings.TrimSpace(types[0].Text()) != "cfg:CatalogRef.упо_СтатьиБДР" {
		t.Fatalf("unexpected remaining type: %q", strings.TrimSpace(types[0].Text()))
	}
}

func TestNormalizeChartOfCharacteristicTypesPredefinedKeepsCurrentConfigAlias(t *testing.T) {
	t.Parallel()

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <ChartOfCharacteristicTypes>
    <Properties>
      <Type>
        <v8:Type>cfg:CatalogRef.упо_Планы</v8:Type>
      </Type>
    </Properties>
  </ChartOfCharacteristicTypes>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	predefinedDoc := etree.NewDocument()
	if err := predefinedDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<PredefinedData xmlns="http://v8.1c.ru/8.3/xcf/predef" xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <Item>
    <Name>План</Name>
    <Type>
      <v8:Type xmlns:d4p1="http://v8.1c.ru/8.1/data/enterprise/current-config">d4p1:CatalogRef.упо_Планы</v8:Type>
    </Type>
  </Item>
</PredefinedData>`); err != nil {
		t.Fatalf("read predefined xml: %v", err)
	}

	ctx := &FileProcessingContext{
		Doc:      predefinedDoc,
		FileName: "Predefined.xml",
		OwnerKey: "ChartOfCharacteristicTypes.упо_Владельцы",
	}
	contexts := []*FileProcessingContext{
		ctx,
		{
			Doc:      ownerDoc,
			RelPath:  "ChartsOfCharacteristicTypes/упо_Владельцы.xml",
			Metadata: true,
			OwnerKey: "ChartOfCharacteristicTypes.упо_Владельцы",
		},
	}

	changed := normalizeChartOfCharacteristicTypesPredefined(ctx, contexts)
	if changed {
		t.Fatalf("expected semantic match to keep predefined item untouched")
	}

	got := predefinedDoc.Root().FindElement("./Item/Type/*[local-name()='Type']")
	if got == nil || strings.TrimSpace(got.Text()) != "d4p1:CatalogRef.упо_Планы" {
		t.Fatalf("expected d4p1 alias to stay unchanged, got %q", textOrEmpty(got))
	}
}

func TestNormalizeChartOfCharacteristicTypesPredefinedKeepsItemSpecificScalarQualifiers(t *testing.T) {
	t.Parallel()

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <ChartOfCharacteristicTypes>
    <Properties>
      <Type>
        <v8:Type>xs:string</v8:Type>
        <v8:StringQualifiers>
          <v8:Length>36</v8:Length>
          <v8:AllowedLength>Variable</v8:AllowedLength>
        </v8:StringQualifiers>
      </Type>
    </Properties>
  </ChartOfCharacteristicTypes>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	predefinedDoc := etree.NewDocument()
	if err := predefinedDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<PredefinedData xmlns="http://v8.1c.ru/8.3/xcf/predef" xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <Item>
    <Name>ЭлементПлана</Name>
    <Type>
      <v8:Type>xs:string</v8:Type>
      <v8:StringQualifiers>
        <v8:Length>25</v8:Length>
        <v8:AllowedLength>Fixed</v8:AllowedLength>
      </v8:StringQualifiers>
    </Type>
  </Item>
</PredefinedData>`); err != nil {
		t.Fatalf("read predefined xml: %v", err)
	}

	ctx := &FileProcessingContext{
		Doc:      predefinedDoc,
		FileName: "Predefined.xml",
		OwnerKey: "ChartOfCharacteristicTypes.упо_Владельцы",
	}
	contexts := []*FileProcessingContext{
		ctx,
		{
			Doc:      ownerDoc,
			RelPath:  "ChartsOfCharacteristicTypes/упо_Владельцы.xml",
			Metadata: true,
			OwnerKey: "ChartOfCharacteristicTypes.упо_Владельцы",
		},
	}

	changed := normalizeChartOfCharacteristicTypesPredefined(ctx, contexts)
	if changed {
		t.Fatalf("expected predefined item-specific scalar qualifiers to stay untouched")
	}

	qualifier := predefinedDoc.Root().FindElement("./Item/Type/*[local-name()='StringQualifiers']")
	if qualifier == nil {
		t.Fatalf("expected string qualifier to remain")
	}
	length := qualifier.FindElement("./*[local-name()='Length']")
	mode := qualifier.FindElement("./*[local-name()='AllowedLength']")
	if length == nil || strings.TrimSpace(length.Text()) != "25" {
		t.Fatalf("expected original string length to remain, got %q", textOrEmpty(length))
	}
	if mode == nil || strings.TrimSpace(mode.Text()) != "Fixed" {
		t.Fatalf("expected original allowed length to remain, got %q", textOrEmpty(mode))
	}
}

func TestNormalizeChartOfCharacteristicTypesPredefinedDoesNotCopyUnrelatedOwnerQualifiersToReferenceItem(t *testing.T) {
	t.Parallel()

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <ChartOfCharacteristicTypes>
    <Properties>
      <Type>
        <v8:Type>cfg:CatalogRef.упо_Подразделения</v8:Type>
        <v8:Type>xs:string</v8:Type>
        <v8:StringQualifiers>
          <v8:Length>150</v8:Length>
          <v8:AllowedLength>Variable</v8:AllowedLength>
        </v8:StringQualifiers>
      </Type>
    </Properties>
  </ChartOfCharacteristicTypes>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	predefinedDoc := etree.NewDocument()
	if err := predefinedDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<PredefinedData xmlns="http://v8.1c.ru/8.3/xcf/predef" xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <Item>
    <Name>СписокПодразделение</Name>
    <Type>
      <v8:Type xmlns:d4p1="http://v8.1c.ru/8.1/data/enterprise/current-config">d4p1:CatalogRef.упо_Подразделения</v8:Type>
    </Type>
  </Item>
</PredefinedData>`); err != nil {
		t.Fatalf("read predefined xml: %v", err)
	}

	ctx := &FileProcessingContext{
		Doc:      predefinedDoc,
		FileName: "Predefined.xml",
		OwnerKey: "ChartOfCharacteristicTypes.упо_Владельцы",
	}
	contexts := []*FileProcessingContext{
		ctx,
		{
			Doc:      ownerDoc,
			RelPath:  "ChartsOfCharacteristicTypes/упо_Владельцы.xml",
			Metadata: true,
			OwnerKey: "ChartOfCharacteristicTypes.упо_Владельцы",
		},
	}

	changed := normalizeChartOfCharacteristicTypesPredefined(ctx, contexts)
	if changed {
		t.Fatalf("expected unrelated owner qualifiers to stay out of predefined reference item")
	}

	qualifier := predefinedDoc.Root().FindElement("./Item/Type/*[local-name()='StringQualifiers']")
	if qualifier != nil {
		t.Fatalf("did not expect owner string qualifier to be copied to predefined reference item")
	}
}

func TestNormalizeRootConfigurationKeepsAdoptedBelonging(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <ObjectBelonging>Adopted</ObjectBelonging>
      <Name>СтароеИмя</Name>
      <NamePrefix>old_</NamePrefix>
      <Vendor>ООО Ромашка</Vendor>
      <Version>1.2.3</Version>
    </Properties>
  </Configuration>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='Configuration']/*[local-name()='Properties']")
	if properties == nil {
		t.Fatalf("expected root configuration properties")
	}

	cfg := &config.Configuration{
		ExtensionProperties: config.ExtensionProperties{
			Name:       "УправлениеПроектами",
			Prefix:     "упо_",
			Identifier: "83b63dda-4eec-11f1-b61f-e0d55ee14481",
		},
		PlatformVersion: "8.3.27.1540",
	}

	if !normalizeRootConfiguration(properties, cfg) {
		t.Fatalf("expected root configuration normalization to change properties")
	}

	if got := textOf(properties, "ObjectBelonging"); got != "Adopted" {
		t.Fatalf("expected root configuration belonging to stay Adopted, got %q", got)
	}
	if got := textOf(properties, "Vendor"); got != `ООО Ромашка` {
		t.Fatalf("expected root configuration vendor to be preserved, got %q", got)
	}
	if got := textOf(properties, "Version"); got != "1.2.3" {
		t.Fatalf("expected root configuration version to be preserved, got %q", got)
	}
	configuration := doc.FindElement("//*[local-name()='Configuration']")
	if configuration == nil {
		t.Fatalf("expected root configuration element")
	}
	if got := configuration.SelectAttrValue("uuid", ""); got != "83b63dda-4eec-11f1-b61f-e0d55ee14481" {
		t.Fatalf("expected root configuration identifier to be preserved from config, got %q", got)
	}
}

func TestCollectTargetMergeRulesRecognizesDefinedTypeAlias(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	templatePath := filepath.Join(root, "CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt")
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	if err := os.WriteFile(templatePath, []byte(`{
  "ОпределяемыйТип": {
    "ТипИзШаблона": {
      "Тип": []
    }
  },
  "ПланОбмена": {
    "ПланИзШаблона": {
      "Состав": []
    }
  },
  "ПодпискаНаСобытие": {
    "ПодпискаИзШаблона": {
      "Источник": []
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	rules := collectTargetMergeRules(&config.Configuration{
		Target: config.Target{XMLDump: t.TempDir()},
	}, root)

	for _, key := range []string{
		"DefinedType.ТипИзШаблона",
		"ExchangePlan.ПланИзШаблона",
		"EventSubscription.ПодпискаИзШаблона",
	} {
		if _, ok := rules.ObjectKeys[key]; !ok {
			t.Fatalf("expected %s in target merge rules, got %#v", key, rules.ObjectKeys)
		}
	}
}

func TestLoadXMLContextsScopesRelativeDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	files := map[string]string{
		filepath.Join("DefinedTypes", "НужныйТип.xml"): `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>НужныйТип</Name></Properties></DefinedType></MetaDataObject>`,
		filepath.Join("EventSubscriptions", "НужнаяПодписка.xml"): `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><EventSubscription><Properties><Name>НужнаяПодписка</Name></Properties></EventSubscription></MetaDataObject>`,
		filepath.Join("Catalogs", "Лишний.xml"): `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog><Properties><Name>Лишний</Name></Properties></Catalog></MetaDataObject>`,
	}

	for relPath, content := range files {
		path := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir test dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test xml: %v", err)
		}
	}

	contexts, err := loadXMLContexts(root, `\DefinedTypes`, `EventSubscriptions`)
	if err != nil {
		t.Fatalf("load scoped xml contexts: %v", err)
	}

	gotKeys := make(map[string]struct{}, len(contexts))
	for _, ctx := range contexts {
		gotKeys[ctx.OwnerKey] = struct{}{}
	}

	for _, key := range []string{
		"DefinedType.НужныйТип",
		"EventSubscription.НужнаяПодписка",
	} {
		if _, ok := gotKeys[key]; !ok {
			t.Fatalf("expected scoped load to include %s, got %#v", key, gotKeys)
		}
	}
	if _, ok := gotKeys["Catalog.Лишний"]; ok {
		t.Fatalf("did not expect scoped load to include unrelated catalog, got %#v", gotKeys)
	}
	if len(gotKeys) != 2 {
		t.Fatalf("expected exactly two scoped metadata contexts, got %#v", gotKeys)
	}
}

func TestCollectTargetCompatibilitySetReadsOnlyTargetSensitiveTopLevelXML(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile := func(relPath, content string) {
		t.Helper()
		path := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(filepath.Join("DefinedTypes", "НужныйТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>НужныйТип</Name></Properties></DefinedType></MetaDataObject>`)
	writeFile(filepath.Join("EventSubscriptions", "НужнаяПодписка.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><EventSubscription><Properties><Name>НужнаяПодписка</Name></Properties></EventSubscription></MetaDataObject>`)
	writeFile(filepath.Join("ExchangePlans", "НужныйПлан.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><ExchangePlan><Properties><Name>НужныйПлан</Name></Properties><ChildObjects/></ExchangePlan></MetaDataObject>`)
	writeFile(filepath.Join("Catalogs", "ЛишнийКаталог.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog><Properties><Name>ЛишнийКаталог</Name></Properties></Catalog></MetaDataObject>`)
	writeFile(filepath.Join("DefinedTypes", "Nested", "ВложенныйТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>ВложенныйТип</Name></Properties></DefinedType></MetaDataObject>`)
	writeFile(filepath.Join("EventSubscriptions", "Nested", "ВложеннаяПодписка.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><EventSubscription><Properties><Name>ВложеннаяПодписка</Name></Properties></EventSubscription></MetaDataObject>`)
	writeFile(filepath.Join("ExchangePlans", "Nested", "ВложенныйПлан.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><ExchangePlan><Properties><Name>ВложенныйПлан</Name></Properties><ChildObjects/></ExchangePlan></MetaDataObject>`)

	set, err := collectTargetCompatibilitySet(&config.Configuration{
		Target: config.Target{XMLDump: root},
	})
	if err != nil {
		t.Fatalf("collect target compatibility set: %v", err)
	}
	if !set.Enabled {
		t.Fatalf("expected targetCompatibilitySet to be enabled")
	}

	expectedKeys := map[string]struct{}{
		"DefinedType.НужныйТип":            {},
		"EventSubscription.НужнаяПодписка": {},
		"ExchangePlan.НужныйПлан":          {},
	}
	if len(set.Keys) != len(expectedKeys) {
		t.Fatalf("expected only top-level target-sensitive keys, got %#v", set.Keys)
	}
	for key := range expectedKeys {
		if _, ok := set.Keys[key]; !ok {
			t.Fatalf("expected %s in targetCompatibilitySet, got %#v", key, set.Keys)
		}
	}
	for _, unexpected := range []string{
		"Catalog.ЛишнийКаталог",
		"DefinedType.ВложенныйТип",
		"EventSubscription.ВложеннаяПодписка",
		"ExchangePlan.ВложенныйПлан",
	} {
		if _, ok := set.Keys[unexpected]; ok {
			t.Fatalf("did not expect %s in targetCompatibilitySet, got %#v", unexpected, set.Keys)
		}
	}
}

func TestChangeFilesFailsOnInvalidMetaDataFileTemplate(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile := func(relPath, content string) {
		t.Helper()
		path := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile("Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration><ChildObjects/></Configuration>
</MetaDataObject>`)
	writeFile("ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions/></ConfigDumpInfo>`)
	writeFile(filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ПланОбмена": {}
},
{
  "Документ": {
    "Бюджет": {
      "Реквизиты": []
    }
  }
}`)

	err := ChangeFiles(&config.Configuration{
		AdditionalProcessing: config.AdditionalProcessing{
			UseMetaDataFile: true,
		},
	}, root)
	if err == nil {
		t.Fatal("expected ChangeFiles to fail on invalid MetaDataFile template")
	}
	if !strings.Contains(err.Error(), "CommonTemplates/упо_MetaDataFile/Ext/Template.txt") {
		t.Fatalf("expected template path in ChangeFiles error, got %v", err)
	}
}

func TestCollectTargetTopLevelMetadataKeysByUUIDReadsOnlyTopLevelConfigDumpEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "ConfigDumpInfo.xml")
	if err := os.WriteFile(path, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.Источник" id="11111111-1111-1111-1111-111111111111">
      <Metadata name="Catalog.Источник.Attribute.Код" id="22222222-2222-2222-2222-222222222222"/>
    </Metadata>
    <Metadata name="Catalog.ИсточникУстаревший" id="33333333-3333-3333-3333-333333333333"/>
    <Metadata name="DefinedType.ЦелевойТип" id="44444444-4444-4444-4444-444444444444"/>
  </ConfigVersions>
</ConfigDumpInfo>`), 0o644); err != nil {
		t.Fatalf("write config dump info: %v", err)
	}

	got, err := collectTargetTopLevelMetadataKeysByUUID(root)
	if err != nil {
		t.Fatalf("collect target keys by uuid: %v", err)
	}

	expected := map[string]string{
		"11111111-1111-1111-1111-111111111111": "Catalog.Источник",
		"33333333-3333-3333-3333-333333333333": "Catalog.ИсточникУстаревший",
		"44444444-4444-4444-4444-444444444444": "DefinedType.ЦелевойТип",
	}
	if len(got) != len(expected) {
		t.Fatalf("unexpected top-level target key map: %#v", got)
	}
	for id, key := range expected {
		if got[id] != key {
			t.Fatalf("unexpected target key for %s: got %q want %q", id, got[id], key)
		}
	}
	if _, ok := got["22222222-2222-2222-2222-222222222222"]; ok {
		t.Fatalf("did not expect nested metadata id in result: %#v", got)
	}
}

func TestMergeDefinedTypeTargetCompositionUsesExtendedProperty(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ТестовыйТип</Name>
      <Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:CatalogRef.B</v8:Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core" xmlns:d4p1="http://v8.1c.ru/8.1/data/enterprise/current-config">d4p1:CatalogRef.C</v8:Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:CatalogRef.D</v8:Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`); err != nil {
		t.Fatalf("read current defined type xml: %v", err)
	}
	targetDoc := etree.NewDocument()
	if err := targetDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ТестовыйТип</Name>
      <Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:CatalogRef.A</v8:Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:CatalogRef.B</v8:Type>
        <StringQualifiers>
          <Length>20</Length>
          <AllowedLength>Variable</AllowedLength>
        </StringQualifiers>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`); err != nil {
		t.Fatalf("read target defined type xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']")
	currentType := properties.FindElement("./Type")
	targetType := targetDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")

	changed, err := mergeDefinedTypeTargetComposition(properties, currentType, targetType, map[string]objectDecision{
		"Catalog.A": {Belonging: "Native"},
		"Catalog.B": {Belonging: "Native"},
		"Catalog.C": {Belonging: "Native"},
		"Catalog.D": {Belonging: "Native"},
	}, func(string) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Fatalf("merge defined type target composition: %v", err)
	}
	if !changed {
		t.Fatalf("expected defined type merge to rewrite type container")
	}

	typeEl := doc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")
	if typeEl == nil {
		t.Fatalf("expected merged type element")
	}
	if got := typeEl.SelectAttrValue("xsi:type", ""); got != "xr:ExtendedProperty" {
		t.Fatalf("expected xr:ExtendedProperty, got %q", got)
	}

	checkValue := typeEl.FindElement("./*[local-name()='CheckValue']")
	extendValue := typeEl.FindElement("./*[local-name()='ExtendValue']")
	if checkValue == nil || extendValue == nil {
		t.Fatalf("expected CheckValue and ExtendValue children")
	}
	if got := checkValue.SelectAttrValue("xsi:type", ""); got != "v8:TypeDescription" {
		t.Fatalf("unexpected CheckValue xsi:type: %q", got)
	}
	if got := extendValue.SelectAttrValue("xsi:type", ""); got != "v8:TypeDescription" {
		t.Fatalf("unexpected ExtendValue xsi:type: %q", got)
	}

	definedType := doc.FindElement("//*[local-name()='DefinedType']")
	internalInfo := definedType.FindElement("./*[local-name()='InternalInfo']")
	propertiesEl := definedType.FindElement("./*[local-name()='Properties']")
	if internalInfo == nil {
		t.Fatalf("expected InternalInfo for merged DefinedType")
	}
	if propertiesEl == nil {
		t.Fatalf("expected Properties element")
	}
	children := definedType.ChildElements()
	if len(children) < 2 || children[0] != internalInfo || children[1] != propertiesEl {
		order := make([]string, 0, len(children))
		for _, child := range children {
			order = append(order, localName(child.Tag))
		}
		t.Fatalf("expected InternalInfo before Properties, got %#v", order)
	}
	generatedType := internalInfo.FindElement("./*[local-name()='GeneratedType']")
	if generatedType == nil {
		t.Fatalf("expected GeneratedType in InternalInfo")
	}
	if got := generatedType.SelectAttrValue("name", ""); got != "DefinedType.ТестовыйТип" {
		t.Fatalf("unexpected GeneratedType name: %q", got)
	}
	if got := generatedType.SelectAttrValue("category", ""); got != "DefinedType" {
		t.Fatalf("unexpected GeneratedType category: %q", got)
	}
	if got := strings.TrimSpace(textOfFirst(generatedType, "./*[local-name()='TypeId']")); got == "" {
		t.Fatalf("expected non-empty GeneratedType TypeId")
	}
	if got := strings.TrimSpace(textOfFirst(generatedType, "./*[local-name()='ValueId']")); got == "" {
		t.Fatalf("expected non-empty GeneratedType ValueId")
	}
	typeStateCount := 0
	for _, propertyState := range internalInfo.FindElements("./*[local-name()='PropertyState']") {
		if strings.TrimSpace(textOfFirst(propertyState, "./*[local-name()='Property']")) != "Type" {
			continue
		}
		typeStateCount++
		if got := strings.TrimSpace(textOfFirst(propertyState, "./*[local-name()='State']")); got != "MultiState" {
			t.Fatalf("expected PropertyState(Type)=MultiState, got %q", got)
		}
	}
	if typeStateCount != 1 {
		t.Fatalf("expected exactly one PropertyState for Type, got %d", typeStateCount)
	}

	checkValues := []string{}
	for _, el := range checkValue.ChildElements() {
		if localName(el.Tag) == "Type" {
			checkValues = append(checkValues, strings.TrimSpace(el.Text()))
		}
	}
	if fmt.Sprint(checkValues) != fmt.Sprint([]string{"cfg:CatalogRef.A", "cfg:CatalogRef.B"}) {
		t.Fatalf("unexpected CheckValue composition: %#v", checkValues)
	}
	stringQualifiers := checkValue.FindElement("./*[local-name()='StringQualifiers']")
	if stringQualifiers == nil {
		t.Fatalf("expected target qualifier block in CheckValue")
	}
	if got := strings.TrimSpace(textOf(stringQualifiers, "Length")); got != "20" {
		t.Fatalf("unexpected StringQualifiers/Length in CheckValue: %q", got)
	}

	extendValues := []string{}
	for _, el := range extendValue.ChildElements() {
		if localName(el.Tag) == "Type" {
			extendValues = append(extendValues, strings.TrimSpace(el.Text()))
		}
	}
	if fmt.Sprint(extendValues) != fmt.Sprint([]string{"d4p1:CatalogRef.C", "cfg:CatalogRef.D"}) {
		t.Fatalf("unexpected ExtendValue composition: %#v", extendValues)
	}
	if extendValue.FindElement("./*[local-name()='StringQualifiers']") != nil {
		t.Fatalf("did not expect qualifier block in ExtendValue")
	}
	for _, el := range extendValue.ChildElements() {
		if strings.TrimSpace(el.Text()) == "d4p1:CatalogRef.C" && el.SelectAttrValue("xmlns:d4p1", "") == "" {
			t.Fatalf("expected current-config alias namespace to be preserved on source addition")
		}
	}
}

func TestMergeDefinedTypeTargetCompositionPreservesExistingGeneratedTypeIDs(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <InternalInfo>
      <xr:GeneratedType xmlns:xr="http://v8.1c.ru/8.3/xcf/readable" name="DefinedType.ТестовыйТип" category="DefinedType">
        <xr:TypeId>aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa</xr:TypeId>
        <xr:ValueId>bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb</xr:ValueId>
      </xr:GeneratedType>
      <xr:PropertyState xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
        <xr:Property>Type</xr:Property>
        <xr:State>Extended</xr:State>
      </xr:PropertyState>
    </InternalInfo>
    <Properties>
      <Name>ТестовыйТип</Name>
      <Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:CatalogRef.C</v8:Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`); err != nil {
		t.Fatalf("read current defined type xml: %v", err)
	}
	targetDoc := etree.NewDocument()
	if err := targetDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <InternalInfo>
      <xr:GeneratedType xmlns:xr="http://v8.1c.ru/8.3/xcf/readable" name="DefinedType.ТестовыйТип" category="DefinedType">
        <xr:TypeId>cccccccc-cccc-cccc-cccc-cccccccccccc</xr:TypeId>
        <xr:ValueId>dddddddd-dddd-dddd-dddd-dddddddddddd</xr:ValueId>
      </xr:GeneratedType>
    </InternalInfo>
    <Properties>
      <Name>ТестовыйТип</Name>
      <Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:CatalogRef.A</v8:Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`); err != nil {
		t.Fatalf("read target defined type xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']")
	currentType := properties.FindElement("./Type")
	targetType := targetDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")

	if _, err := mergeDefinedTypeTargetComposition(properties, currentType, targetType, map[string]objectDecision{
		"Catalog.A": {Belonging: "Native"},
		"Catalog.C": {Belonging: "Native"},
	}, func(string) (bool, error) {
		return true, nil
	}); err != nil {
		t.Fatalf("merge defined type target composition: %v", err)
	}

	generatedType := doc.FindElement("//*[local-name()='DefinedType']/*[local-name()='InternalInfo']/*[local-name()='GeneratedType']")
	if generatedType == nil {
		t.Fatalf("expected GeneratedType after merge")
	}
	if got := strings.TrimSpace(textOfFirst(generatedType, "./*[local-name()='TypeId']")); got != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected existing current TypeId to be preserved, got %q", got)
	}
	if got := strings.TrimSpace(textOfFirst(generatedType, "./*[local-name()='ValueId']")); got != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("expected existing current ValueId to be preserved, got %q", got)
	}

	propertyStates := doc.FindElements("//*[local-name()='DefinedType']/*[local-name()='InternalInfo']/*[local-name()='PropertyState']")
	typeStateCount := 0
	for _, propertyState := range propertyStates {
		if strings.TrimSpace(textOfFirst(propertyState, "./*[local-name()='Property']")) != "Type" {
			continue
		}
		typeStateCount++
		if got := strings.TrimSpace(textOfFirst(propertyState, "./*[local-name()='State']")); got != "MultiState" {
			t.Fatalf("expected PropertyState(Type)=MultiState, got %q", got)
		}
	}
	if typeStateCount != 1 {
		t.Fatalf("expected exactly one Type PropertyState after merge, got %d", typeStateCount)
	}
}

func TestMergeDefinedTypeTargetCompositionUsesTargetGeneratedTypeIDsWhenCurrentMissing(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ТестовыйТип</Name>
      <Type/>
    </Properties>
  </DefinedType>
</MetaDataObject>`); err != nil {
		t.Fatalf("read current defined type xml: %v", err)
	}
	targetDoc := etree.NewDocument()
	if err := targetDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <InternalInfo>
      <xr:GeneratedType xmlns:xr="http://v8.1c.ru/8.3/xcf/readable" name="DefinedType.ТестовыйТип" category="DefinedType">
        <xr:TypeId>cccccccc-cccc-cccc-cccc-cccccccccccc</xr:TypeId>
        <xr:ValueId>dddddddd-dddd-dddd-dddd-dddddddddddd</xr:ValueId>
      </xr:GeneratedType>
    </InternalInfo>
    <Properties>
      <Name>ТестовыйТип</Name>
      <Type>
        <v8:Type xmlns:v8="http://v8.1c.ru/8.1/data/core">cfg:CatalogRef.A</v8:Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`); err != nil {
		t.Fatalf("read target defined type xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']")
	currentType := properties.FindElement("./Type")
	targetType := targetDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")

	if _, err := mergeDefinedTypeTargetComposition(properties, currentType, targetType, map[string]objectDecision{
		"Catalog.A": {Belonging: "Native"},
	}, func(string) (bool, error) {
		return true, nil
	}); err != nil {
		t.Fatalf("merge defined type target composition: %v", err)
	}

	generatedType := doc.FindElement("//*[local-name()='DefinedType']/*[local-name()='InternalInfo']/*[local-name()='GeneratedType']")
	if generatedType == nil {
		t.Fatalf("expected GeneratedType after merge")
	}
	if got := strings.TrimSpace(textOfFirst(generatedType, "./*[local-name()='TypeId']")); got != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Fatalf("expected target TypeId to be preserved, got %q", got)
	}
	if got := strings.TrimSpace(textOfFirst(generatedType, "./*[local-name()='ValueId']")); got != "dddddddd-dddd-dddd-dddd-dddddddddddd" {
		t.Fatalf("expected target ValueId to be preserved, got %q", got)
	}
}

func TestMergeDefinedTypeTargetCompositionLeavesOtherContainersUntouched(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <EventSubscription>
    <Properties>
      <Name>Подписка</Name>
      <Source>
        <Item>Catalog.A</Item>
      </Source>
    </Properties>
  </EventSubscription>
</MetaDataObject>`); err != nil {
		t.Fatalf("read event subscription xml: %v", err)
	}
	properties := doc.FindElement("//*[local-name()='EventSubscription']/*[local-name()='Properties']")
	currentSource := properties.FindElement("./Source")

	targetDoc := etree.NewDocument()
	if err := targetDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <EventSubscription>
    <Properties>
      <Name>Подписка</Name>
      <Source>
        <Item>Catalog.B</Item>
      </Source>
    </Properties>
  </EventSubscription>
</MetaDataObject>`); err != nil {
		t.Fatalf("read target event subscription xml: %v", err)
	}
	targetSource := targetDoc.FindElement("//*[local-name()='EventSubscription']/*[local-name()='Properties']/*[local-name()='Source']")

	changed, err := mergeMetadataValueContainer(properties, "Source", currentSource, targetSource, map[string]objectDecision{
		"Catalog.A": {Belonging: "Native"},
		"Catalog.B": {Belonging: "Native"},
	}, func(string) (bool, error) {
		return true, nil
	})
	if err != nil {
		t.Fatalf("merge metadata value container: %v", err)
	}
	if !changed {
		t.Fatalf("expected EventSubscription source merge to change container")
	}
	if got := currentSource.SelectAttrValue("xsi:type", ""); got != "" {
		t.Fatalf("did not expect EventSubscription Source to change xsi:type, got %q", got)
	}
}

func TestBuildChangeFilesStateOrdinaryDefinedTypeDoesNotBecomeExtendedProperty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Configuration><ChildObjects><DefinedType>ОбычныйТип</DefinedType></ChildObjects></Configuration></MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions><Metadata name="DefinedType.ОбычныйТип" id="11111111-1111-1111-1111-111111111111"/></ConfigVersions></ConfigDumpInfo>`)
	writeFile(root, filepath.Join("DefinedTypes", "ОбычныйТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ОбычныйТип</Name>
      <Type><Type>cfg:CatalogRef.Номенклатура</Type></Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)

	if _, err := buildChangeFilesState(&config.Configuration{
		IncludedAdoptedStubObjects: []string{"DefinedType.ОбычныйТип"},
	}, root); err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromFile(filepath.Join(root, "DefinedTypes", "ОбычныйТип.xml")); err != nil {
		t.Fatalf("read defined type xml: %v", err)
	}
	typeEl := doc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")
	if typeEl == nil {
		t.Fatalf("expected ordinary defined type Type element")
	}
	if got := typeEl.SelectAttrValue("xsi:type", ""); got != "" {
		t.Fatalf("did not expect ordinary defined type to become typed merge container, got %q", got)
	}
}

func TestCanonicalizeTargetRenamedAdoptedObjectsPrefersTargetName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <Catalog>Источник</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.Источник" id="11111111-1111-1111-1111-111111111111">
      <Metadata name="Catalog.Источник.Attribute.Код" id="22222222-2222-2222-2222-222222222222"/>
    </Metadata>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("Catalogs", "Источник.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Источник</Name>
    </Properties>
  </Catalog>
</MetaDataObject>`)
	writeFile(root, filepath.Join("DefinedTypes", "ИспользуетИсточник.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <DefinedType>
    <Properties>
      <Name>ИспользуетИсточник</Name>
      <Type>
        <v8:Type>cfg:CatalogRef.Источник</v8:Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)

	writeFile(targetRoot, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.ИсточникУстаревший" id="11111111-1111-1111-1111-111111111111"/>
    <Metadata name="Catalog.ИсточникНовый" id="33333333-3333-3333-3333-333333333333"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "ИсточникУстаревший.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ИсточникУстаревший</Name>
    </Properties>
  </Catalog>
</MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "ИсточникНовый.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="33333333-3333-3333-3333-333333333333">
    <Properties>
      <Name>ИсточникНовый</Name>
    </Properties>
  </Catalog>
</MetaDataObject>`)

	contexts, err := loadXMLContexts(root)
	if err != nil {
		t.Fatalf("load contexts: %v", err)
	}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Configuration":    {Belonging: "AdoptedStub"},
		"Catalog.Источник": {Belonging: "AdoptedStub"},
	}

	contexts, indexes, err = canonicalizeTargetRenamedAdoptedObjects(&config.Configuration{
		Target: config.Target{XMLDump: targetRoot},
	}, root, contexts, indexes, decisions)
	if err != nil {
		t.Fatalf("canonicalize target renamed adopted objects: %v", err)
	}

	if decision := decisions["Catalog.Источник"]; !decision.Excluded {
		t.Fatalf("expected source name to be excluded after target canonicalization, got %#v", decision)
	}
	if decision := decisions["Catalog.ИсточникУстаревший"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected target name to stay adopted, got %#v", decision)
	}

	if _, err := os.Stat(filepath.Join(root, "Catalogs", "ИсточникУстаревший.xml")); err != nil {
		t.Fatalf("expected target canonical xml to be imported: %v", err)
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	if hasConfigurationChildObject(configurationDoc, "Catalog", "Источник") {
		t.Fatalf("did not expect stale source child object after canonicalization")
	}
	if !hasConfigurationChildObject(configurationDoc, "Catalog", "ИсточникУстаревший") {
		t.Fatalf("expected target child object after canonicalization")
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}
	if hasMetadataName(configDumpDoc, "Catalog.Источник") || hasMetadataName(configDumpDoc, "Catalog.Источник.Attribute.Код") {
		t.Fatalf("did not expect stale source ConfigDumpInfo entries after canonicalization")
	}
	if !hasMetadataName(configDumpDoc, "Catalog.ИсточникУстаревший") {
		t.Fatalf("expected target canonical ConfigDumpInfo entry")
	}
	if got := metadataEntryID(configDumpDoc, "Catalog.ИсточникУстаревший"); got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected target canonical ConfigDumpInfo id: %q", got)
	}

	if ctx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, "Catalog.ИсточникУстаревший"); ctx == nil {
		t.Fatalf("expected canonical target context to be present")
	}

	definedTypeDoc := etree.NewDocument()
	if err := definedTypeDoc.ReadFromFile(filepath.Join(root, "DefinedTypes", "ИспользуетИсточник.xml")); err != nil {
		t.Fatalf("read defined type xml: %v", err)
	}
	typeEl := definedTypeDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']/*[local-name()='Type']")
	if typeEl == nil || strings.TrimSpace(typeEl.Text()) != "cfg:CatalogRef.ИсточникУстаревший" {
		t.Fatalf("expected canonical target name in type-bearing ref, got %q", textOrEmpty(typeEl))
	}
}

func TestCanonicalizeTargetRenamedAdoptedObjectsSkipsSameTargetKey(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <Catalog>Источник</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.Источник" id="11111111-1111-1111-1111-111111111111"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("Catalogs", "Источник.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties><Name>Источник</Name></Properties>
  </Catalog>
</MetaDataObject>`)

	writeFile(targetRoot, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.Источник" id="11111111-1111-1111-1111-111111111111"/>
  </ConfigVersions>
</ConfigDumpInfo>`)

	contexts, err := loadXMLContexts(root)
	if err != nil {
		t.Fatalf("load contexts: %v", err)
	}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Configuration":    {Belonging: "AdoptedStub"},
		"Catalog.Источник": {Belonging: "AdoptedStub"},
	}

	contexts, indexes, err = canonicalizeTargetRenamedAdoptedObjects(&config.Configuration{
		Target: config.Target{XMLDump: targetRoot},
	}, root, contexts, indexes, decisions)
	if err != nil {
		t.Fatalf("canonicalize target renamed adopted objects: %v", err)
	}

	if decision := decisions["Catalog.Источник"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected source object to stay unchanged, got %#v", decision)
	}
	if ctx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, "Catalog.Источник"); ctx == nil {
		t.Fatalf("expected original context to stay present")
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	if !hasConfigurationChildObject(configurationDoc, "Catalog", "Источник") {
		t.Fatalf("expected configuration child object to stay unchanged")
	}
	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}
	if !hasMetadataName(configDumpDoc, "Catalog.Источник") {
		t.Fatalf("expected config dump entry to stay unchanged")
	}
}

func TestCanonicalizeTargetRenamedAdoptedObjectsUsesExistingCanonicalContext(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <Catalog>Источник</Catalog>
      <Catalog>ИсточникУстаревший</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.Источник" id="11111111-1111-1111-1111-111111111111">
      <Metadata name="Catalog.Источник.Attribute.Код" id="22222222-2222-2222-2222-222222222222"/>
    </Metadata>
    <Metadata name="Catalog.ИсточникУстаревший" id="11111111-1111-1111-1111-111111111111"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("Catalogs", "Источник.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties><Name>Источник</Name></Properties>
  </Catalog>
</MetaDataObject>`)
	writeFile(root, filepath.Join("Catalogs", "ИсточникУстаревший.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties><Name>ИсточникУстаревший</Name></Properties>
  </Catalog>
</MetaDataObject>`)

	writeFile(targetRoot, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.ИсточникУстаревший" id="11111111-1111-1111-1111-111111111111"/>
  </ConfigVersions>
</ConfigDumpInfo>`)

	contexts, err := loadXMLContexts(root)
	if err != nil {
		t.Fatalf("load contexts: %v", err)
	}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Configuration":    {Belonging: "AdoptedStub"},
		"Catalog.Источник": {Belonging: "AdoptedStub", SearchResultCode: true},
	}

	_, _, err = canonicalizeTargetRenamedAdoptedObjects(&config.Configuration{
		Target: config.Target{XMLDump: targetRoot},
	}, root, contexts, indexes, decisions)
	if err != nil {
		t.Fatalf("canonicalize target renamed adopted objects: %v", err)
	}

	if decision := decisions["Catalog.Источник"]; !decision.Excluded {
		t.Fatalf("expected stale source key to be excluded, got %#v", decision)
	}
	if decision := decisions["Catalog.ИсточникУстаревший"]; decision.Excluded || decision.Belonging != "AdoptedStub" || !decision.SearchResultCode {
		t.Fatalf("expected canonical decision to reuse existing context and carry flags, got %#v", decision)
	}
}

func TestCanonicalizeTargetRenamedAdoptedObjectsBatchRemovesMultipleStaleEntries(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <Catalog>Источник1</Catalog>
      <Catalog>Источник2</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.Источник1" id="11111111-1111-1111-1111-111111111111">
      <Metadata name="Catalog.Источник1.Attribute.Код" id="21111111-1111-1111-1111-111111111111"/>
    </Metadata>
    <Metadata name="Catalog.Источник2" id="33333333-3333-3333-3333-333333333333">
      <Metadata name="Catalog.Источник2.Attribute.Код" id="43333333-3333-3333-3333-333333333333"/>
    </Metadata>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("Catalogs", "Источник1.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties><Name>Источник1</Name></Properties>
  </Catalog>
</MetaDataObject>`)
	writeFile(root, filepath.Join("Catalogs", "Источник2.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="33333333-3333-3333-3333-333333333333">
    <Properties><Name>Источник2</Name></Properties>
  </Catalog>
</MetaDataObject>`)

	writeFile(targetRoot, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Catalog.Канон1" id="11111111-1111-1111-1111-111111111111"/>
    <Metadata name="Catalog.Канон2" id="33333333-3333-3333-3333-333333333333"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "Канон1.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="11111111-1111-1111-1111-111111111111">
    <Properties><Name>Канон1</Name></Properties>
  </Catalog>
</MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "Канон2.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="33333333-3333-3333-3333-333333333333">
    <Properties><Name>Канон2</Name></Properties>
  </Catalog>
</MetaDataObject>`)

	contexts, err := loadXMLContexts(root)
	if err != nil {
		t.Fatalf("load contexts: %v", err)
	}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Configuration":     {Belonging: "AdoptedStub"},
		"Catalog.Источник1": {Belonging: "AdoptedStub"},
		"Catalog.Источник2": {Belonging: "AdoptedStub"},
	}

	_, _, err = canonicalizeTargetRenamedAdoptedObjects(&config.Configuration{
		Target: config.Target{XMLDump: targetRoot},
	}, root, contexts, indexes, decisions)
	if err != nil {
		t.Fatalf("canonicalize target renamed adopted objects: %v", err)
	}

	for _, staleKey := range []string{"Catalog.Источник1", "Catalog.Источник2"} {
		if decision := decisions[staleKey]; !decision.Excluded {
			t.Fatalf("expected stale key %s to be excluded, got %#v", staleKey, decision)
		}
	}
	for _, canonicalKey := range []string{"Catalog.Канон1", "Catalog.Канон2"} {
		if decision := decisions[canonicalKey]; decision.Excluded || decision.Belonging != "AdoptedStub" {
			t.Fatalf("expected canonical key %s to stay adopted, got %#v", canonicalKey, decision)
		}
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	if hasConfigurationChildObject(configurationDoc, "Catalog", "Источник1") || hasConfigurationChildObject(configurationDoc, "Catalog", "Источник2") {
		t.Fatalf("did not expect stale child objects after canonicalization")
	}
	if !hasConfigurationChildObject(configurationDoc, "Catalog", "Канон1") || !hasConfigurationChildObject(configurationDoc, "Catalog", "Канон2") {
		t.Fatalf("expected canonical child objects after canonicalization")
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}
	for _, staleName := range []string{
		"Catalog.Источник1",
		"Catalog.Источник1.Attribute.Код",
		"Catalog.Источник2",
		"Catalog.Источник2.Attribute.Код",
	} {
		if hasMetadataName(configDumpDoc, staleName) {
			t.Fatalf("did not expect stale config dump entry %s", staleName)
		}
	}
	for _, canonicalName := range []string{"Catalog.Канон1", "Catalog.Канон2"} {
		if !hasMetadataName(configDumpDoc, canonicalName) {
			t.Fatalf("expected canonical config dump entry %s", canonicalName)
		}
	}
}

func TestCanonicalizeTargetRenamedAdoptedObjectsSkipsNativeAndExcluded(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		decision objectDecision
	}

	for _, tc := range []testCase{
		{name: "Native", decision: objectDecision{Belonging: "Native"}},
		{name: "Excluded", decision: objectDecision{Excluded: true}},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			targetRoot := t.TempDir()
			writeFile := func(base, relPath, content string) {
				t.Helper()
				path := filepath.Join(base, relPath)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}
			}

			writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration><ChildObjects><Catalog>Источник</Catalog></ChildObjects></Configuration>
</MetaDataObject>`)
			writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions><Metadata name="Catalog.Источник" id="11111111-1111-1111-1111-111111111111"/></ConfigVersions></ConfigDumpInfo>`)
			writeFile(root, filepath.Join("Catalogs", "Источник.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="11111111-1111-1111-1111-111111111111"><Properties><Name>Источник</Name></Properties></Catalog></MetaDataObject>`)

			writeFile(targetRoot, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions><Metadata name="Catalog.Канон" id="11111111-1111-1111-1111-111111111111"/></ConfigVersions></ConfigDumpInfo>`)
			writeFile(targetRoot, filepath.Join("Catalogs", "Канон.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="11111111-1111-1111-1111-111111111111"><Properties><Name>Канон</Name></Properties></Catalog></MetaDataObject>`)

			contexts, err := loadXMLContexts(root)
			if err != nil {
				t.Fatalf("load contexts: %v", err)
			}
			indexes := buildContextIndexes(contexts)
			decisions := map[string]objectDecision{
				"Configuration":    {Belonging: "AdoptedStub"},
				"Catalog.Источник": tc.decision,
			}

			_, _, err = canonicalizeTargetRenamedAdoptedObjects(&config.Configuration{
				Target: config.Target{XMLDump: targetRoot},
			}, root, contexts, indexes, decisions)
			if err != nil {
				t.Fatalf("canonicalize target renamed adopted objects: %v", err)
			}

			if _, exists := decisions["Catalog.Канон"]; exists {
				t.Fatalf("did not expect canonical target decision for skipped %s object", tc.name)
			}
			if _, err := os.Stat(filepath.Join(root, "Catalogs", "Канон.xml")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("did not expect canonical xml import for skipped %s object, err=%v", tc.name, err)
			}
		})
	}
}

func TestBuildChangeFilesStateTargetCompatibilitySetBlocksRefDrivenAdoptedSensitiveObjectsAbsentInTarget(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name            string
		key             string
		relPath         string
		configurationEl string
		configDumpName  string
		sourceRef       string
		objectXML       string
	}

	cases := []testCase{
		{
			name:            "DefinedType",
			key:             "DefinedType.НецелевойТип",
			relPath:         filepath.Join("DefinedTypes", "НецелевойТип.xml"),
			configurationEl: "<DefinedType>НецелевойТип</DefinedType>",
			configDumpName:  "DefinedType.НецелевойТип",
			sourceRef:       "cfg:DefinedType.НецелевойТип",
			objectXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>НецелевойТип</Name><Type/></Properties></DefinedType></MetaDataObject>`,
		},
		{
			name:            "EventSubscription",
			key:             "EventSubscription.НецелеваяПодписка",
			relPath:         filepath.Join("EventSubscriptions", "НецелеваяПодписка.xml"),
			configurationEl: "<EventSubscription>НецелеваяПодписка</EventSubscription>",
			configDumpName:  "EventSubscription.НецелеваяПодписка",
			sourceRef:       "EventSubscription.НецелеваяПодписка",
			objectXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><EventSubscription><Properties><Name>НецелеваяПодписка</Name><Source/></Properties></EventSubscription></MetaDataObject>`,
		},
		{
			name:            "ExchangePlan",
			key:             "ExchangePlan.НецелевойПлан",
			relPath:         filepath.Join("ExchangePlans", "НецелевойПлан.xml"),
			configurationEl: "<ExchangePlan>НецелевойПлан</ExchangePlan>",
			configDumpName:  "ExchangePlan.НецелевойПлан",
			sourceRef:       "ExchangePlan.НецелевойПлан",
			objectXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><ExchangePlan><Properties><Name>НецелевойПлан</Name></Properties><ChildObjects/></ExchangePlan></MetaDataObject>`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			targetRoot := t.TempDir()
			writeFile := func(base, relPath, content string) {
				t.Helper()
				path := filepath.Join(base, relPath)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}
			}

			writeFile(root, "Configuration.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Configuration><ChildObjects><Catalog>Источник</Catalog>%s</ChildObjects></Configuration></MetaDataObject>`, tc.configurationEl))
			writeFile(root, "ConfigDumpInfo.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions><Metadata name="Catalog.Источник" id="11111111-1111-1111-1111-111111111111"/><Metadata name="%s" id="22222222-2222-2222-2222-222222222222"/></ConfigVersions></ConfigDumpInfo>`, tc.configDumpName))
			writeFile(root, filepath.Join("Catalogs", "Источник.xml"), fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog><Properties><Name>Источник</Name><Explanation>%s</Explanation></Properties></Catalog></MetaDataObject>`, tc.sourceRef))
			writeFile(root, tc.relPath, tc.objectXML)

			state, err := buildChangeFilesState(&config.Configuration{
				IncludedNativeObjects: []string{"Catalog.Источник"},
				Target:                config.Target{XMLDump: targetRoot},
			}, root)
			if err != nil {
				t.Fatalf("build change files state: %v", err)
			}

			if decision := state.decisions["Catalog.Источник"]; decision.Excluded || decision.Belonging != "Native" {
				t.Fatalf("expected native source object to stay native, got %#v", decision)
			}
			if decision := state.decisions[tc.key]; !decision.Excluded {
				t.Fatalf("expected incompatible target-sensitive object to stay excluded, got %#v", decision)
			}
		})
	}
}

func TestBuildChangeFilesStateTargetCompatibilitySetKeepsNativeSensitiveObjectsAbsentInTarget(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name            string
		key             string
		relPath         string
		configurationEl string
		configDumpName  string
		objectXML       string
	}

	cases := []testCase{
		{
			name:            "DefinedType",
			key:             "DefinedType.упо_НативныйТип",
			relPath:         filepath.Join("DefinedTypes", "упо_НативныйТип.xml"),
			configurationEl: "<DefinedType>упо_НативныйТип</DefinedType>",
			configDumpName:  "DefinedType.упо_НативныйТип",
			objectXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>упо_НативныйТип</Name><Type/></Properties></DefinedType></MetaDataObject>`,
		},
		{
			name:            "EventSubscription",
			key:             "EventSubscription.упо_НативнаяПодписка",
			relPath:         filepath.Join("EventSubscriptions", "упо_НативнаяПодписка.xml"),
			configurationEl: "<EventSubscription>упо_НативнаяПодписка</EventSubscription>",
			configDumpName:  "EventSubscription.упо_НативнаяПодписка",
			objectXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><EventSubscription><Properties><Name>упо_НативнаяПодписка</Name><Source/></Properties></EventSubscription></MetaDataObject>`,
		},
		{
			name:            "ExchangePlan",
			key:             "ExchangePlan.упо_НативныйПлан",
			relPath:         filepath.Join("ExchangePlans", "упо_НативныйПлан.xml"),
			configurationEl: "<ExchangePlan>упо_НативныйПлан</ExchangePlan>",
			configDumpName:  "ExchangePlan.упо_НативныйПлан",
			objectXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><ExchangePlan><Properties><Name>упо_НативныйПлан</Name></Properties><ChildObjects/></ExchangePlan></MetaDataObject>`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			targetRoot := t.TempDir()
			writeFile := func(base, relPath, content string) {
				t.Helper()
				path := filepath.Join(base, relPath)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}
			}

			writeFile(root, "Configuration.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Configuration><ChildObjects>%s</ChildObjects></Configuration></MetaDataObject>`, tc.configurationEl))
			writeFile(root, "ConfigDumpInfo.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions><Metadata name="%s" id="11111111-1111-1111-1111-111111111111"/></ConfigVersions></ConfigDumpInfo>`, tc.configDumpName))
			writeFile(root, tc.relPath, tc.objectXML)

			state, err := buildChangeFilesState(&config.Configuration{
				NativePrefixes: []string{"упо_"},
				Target:         config.Target{XMLDump: targetRoot},
			}, root)
			if err != nil {
				t.Fatalf("build change files state: %v", err)
			}

			if decision := state.decisions[tc.key]; decision.Excluded || decision.Belonging != "Native" {
				t.Fatalf("expected native target-sensitive object to stay native without target match, got %#v", decision)
			}
		})
	}
}

func TestBuildChangeFilesStateForbiddenBeatsTargetCompatibilitySet(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Configuration><ChildObjects><DefinedType>ЦелевойТип</DefinedType></ChildObjects></Configuration></MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions><Metadata name="DefinedType.ЦелевойТип" id="11111111-1111-1111-1111-111111111111"/></ConfigVersions></ConfigDumpInfo>`)
	writeFile(root, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>ЦелевойТип</Name><Type/></Properties></DefinedType></MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>ЦелевойТип</Name><Type/></Properties></DefinedType></MetaDataObject>`)

	state, err := buildChangeFilesState(&config.Configuration{
		IncludedAdoptedStubObjects:  []string{"DefinedType.ЦелевойТип"},
		ForbiddenAdoptedStubObjects: []string{"DefinedType.ЦелевойТип"},
		Target:                      config.Target{XMLDump: targetRoot},
	}, root)
	if err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	if decision := state.decisions["DefinedType.ЦелевойТип"]; !decision.Excluded {
		t.Fatalf("expected forbidden target-sensitive object to stay excluded, got %#v", decision)
	}
}

func TestBuildChangeFilesStateMergesTargetMetadataOnlyForMetaDataFileObjects(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <DefinedType>ЦелевойТип</DefinedType>
      <DefinedType>ОбычныйТип</DefinedType>
    </ChildObjects>
  </Configuration>
 </MetaDataObject>`)

	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="DefinedType.ЦелевойТип" id="11111111-1111-1111-1111-111111111111"/>
    <Metadata name="DefinedType.ОбычныйТип" id="22222222-2222-2222-2222-222222222222"/>
  </ConfigVersions>
</ConfigDumpInfo>`)

	writeFile(root, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ЦелевойТип</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзSource</Type>
      </Type>
    </Properties>
  </DefinedType>
 </MetaDataObject>`)
	writeFile(root, filepath.Join("DefinedTypes", "ОбычныйТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ОбычныйТип</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзOrdinary</Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(root, filepath.Join("Catalogs", "ИзSource.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog><Properties><Name>ИзSource</Name></Properties></Catalog></MetaDataObject>`)
	writeFile(root, filepath.Join("Catalogs", "ИзOrdinary.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog><Properties><Name>ИзOrdinary</Name></Properties></Catalog></MetaDataObject>`)

	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ОпределяемыйТип": {
    "ЦелевойТип": {
      "Тип": []
    }
  }
}`)

	writeFile(targetRoot, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ЦелевойТип</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзTarget</Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "ИзTarget.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="33333333-3333-3333-3333-333333333333"><Properties><Name>ИзTarget</Name></Properties></Catalog></MetaDataObject>`)

	state, err := buildChangeFilesState(&config.Configuration{
		IncludedAdoptedStubObjects: []string{
			"DefinedType.ЦелевойТип",
			"DefinedType.ОбычныйТип",
		},
		Target: config.Target{
			XMLDump: targetRoot,
		},
	}, root)
	if err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	if decision := state.decisions["Catalog.ИзSource"]; !decision.Excluded {
		t.Fatalf("expected source ref from target-merge defined type not to be promoted before merge")
	}
	if decision := state.decisions["DefinedType.ОбычныйТип"]; !decision.Excluded {
		t.Fatalf("expected adopted defined type absent in target.xml_dump to stay excluded, got %#v", decision)
	}
	if decision := state.decisions["Catalog.ИзOrdinary"]; !decision.Excluded {
		t.Fatalf("expected incompatible defined type source ref not to promote catalog, got %#v", decision)
	}
	if decision := state.decisions["Catalog.ИзTarget"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected target-ref-driven catalog to be imported as AdoptedStub, got %#v", decision)
	}

	if _, err := os.Stat(filepath.Join(root, "Catalogs", "ИзTarget.xml")); err != nil {
		t.Fatalf("expected target-ref-driven catalog file to be created: %v", err)
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	if !hasConfigurationChildObject(configurationDoc, "Catalog", "ИзTarget") {
		t.Fatalf("expected target-ref-driven catalog in Configuration.xml ChildObjects")
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}
	if !hasMetadataName(configDumpDoc, "Catalog.ИзTarget") {
		t.Fatalf("expected target-ref-driven catalog entry in ConfigDumpInfo.xml")
	}
	if got := metadataEntryID(configDumpDoc, "Catalog.ИзTarget"); got != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("expected minimal target-ref-driven metadata id from target xml, got %q", got)
	}
}

func TestBuildChangeFilesStateRevivesExcludedTargetMergeObject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <DefinedType>ЦелевойТип</DefinedType>
      <Catalog>ИзSource</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="DefinedType.ЦелевойТип" id="11111111-1111-1111-1111-111111111111"/>
    <Metadata name="Catalog.ИзSource" id="22222222-2222-2222-2222-222222222222"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ЦелевойТип</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзSource</Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(root, filepath.Join("Catalogs", "ИзSource.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="22222222-2222-2222-2222-222222222222"><Properties><Name>ИзSource</Name></Properties></Catalog></MetaDataObject>`)
	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ОпределяемыйТип": {
    "ЦелевойТип": {
      "Тип": []
    }
  }
}`)

	writeFile(targetRoot, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ЦелевойТип</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзTarget</Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "ИзTarget.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="33333333-3333-3333-3333-333333333333"><Properties><Name>ИзTarget</Name></Properties></Catalog></MetaDataObject>`)

	state, err := buildChangeFilesState(&config.Configuration{
		IncludedNativeObjects: []string{
			"Catalog.ИзSource",
		},
		Target: config.Target{
			XMLDump: targetRoot,
		},
	}, root)
	if err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	if decision := state.decisions["DefinedType.ЦелевойТип"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected excluded target merge object to be revived as AdoptedStub, got %#v", decision)
	}
	if decision := state.decisions["Catalog.ИзTarget"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected target-ref-driven catalog to be imported as AdoptedStub, got %#v", decision)
	}

	definedTypeDoc := etree.NewDocument()
	if err := definedTypeDoc.ReadFromFile(filepath.Join(root, "DefinedTypes", "ЦелевойТип.xml")); err != nil {
		t.Fatalf("read defined type xml: %v", err)
	}
	typeEl := definedTypeDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")
	if typeEl == nil {
		t.Fatalf("expected merged DefinedType type element")
	}
	if got := typeEl.SelectAttrValue("xsi:type", ""); got != "xr:ExtendedProperty" {
		t.Fatalf("expected merged DefinedType type to be xr:ExtendedProperty, got %q", got)
	}
	checkValues := map[string]struct{}{}
	for _, el := range typeEl.FindElements("./*[local-name()='CheckValue']/*") {
		checkValues[strings.TrimSpace(el.Text())] = struct{}{}
	}
	if _, ok := checkValues["cfg:CatalogRef.ИзTarget"]; !ok {
		t.Fatalf("expected CheckValue to retain target ref, got %#v", checkValues)
	}
	extendValues := map[string]struct{}{}
	for _, el := range typeEl.FindElements("./*[local-name()='ExtendValue']/*") {
		extendValues[strings.TrimSpace(el.Text())] = struct{}{}
	}
	if _, ok := extendValues["cfg:CatalogRef.ИзSource"]; !ok {
		t.Fatalf("expected ExtendValue to contain source addition, got %#v", extendValues)
	}
	if _, ok := extendValues["cfg:CatalogRef.ИзTarget"]; ok {
		t.Fatalf("did not expect target ref duplication in ExtendValue, got %#v", extendValues)
	}
	internalInfo := definedTypeDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='InternalInfo']")
	if internalInfo == nil {
		t.Fatalf("expected InternalInfo for merged DefinedType")
	}
	if internalInfo.FindElement("./*[local-name()='GeneratedType']") == nil {
		t.Fatalf("expected GeneratedType in merged DefinedType InternalInfo")
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	if !hasConfigurationChildObject(configurationDoc, "DefinedType", "ЦелевойТип") {
		t.Fatalf("expected revived DefinedType in Configuration.xml ChildObjects")
	}
	if !hasConfigurationChildObject(configurationDoc, "Catalog", "ИзTarget") {
		t.Fatalf("expected target-ref-driven catalog in Configuration.xml ChildObjects")
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}
	for _, name := range []string{
		"DefinedType.ЦелевойТип",
		"Catalog.ИзTarget",
	} {
		if !hasMetadataName(configDumpDoc, name) {
			t.Fatalf("expected %s entry in ConfigDumpInfo.xml", name)
		}
	}
	if got := metadataEntryID(configDumpDoc, "Catalog.ИзTarget"); got != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("expected minimal target-ref-driven metadata id from target xml, got %q", got)
	}
}

func TestBuildChangeFilesStateRevivesSoftExcludedTargetRefFromDefinedTypeMerge(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <DefinedType>ЦелевойТип</DefinedType>
      <Catalog>ИзTarget</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="DefinedType.ЦелевойТип" id="11111111-1111-1111-1111-111111111111"/>
    <Metadata name="Catalog.ИзTarget" id="33333333-3333-3333-3333-333333333333"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ЦелевойТип</Name>
      <Type/>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(root, filepath.Join("Catalogs", "ИзTarget.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="33333333-3333-3333-3333-333333333333"><Properties><Name>ИзTarget</Name></Properties></Catalog></MetaDataObject>`)
	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ОпределяемыйТип": {
    "ЦелевойТип": {
      "Тип": []
    }
  }
}`)

	writeFile(targetRoot, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ЦелевойТип</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзTarget</Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "ИзTarget.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="33333333-3333-3333-3333-333333333333"><Properties><Name>ИзTarget</Name></Properties></Catalog></MetaDataObject>`)

	state, err := buildChangeFilesState(&config.Configuration{
		ExcludedObjects: []string{
			"Catalog.ИзTarget",
		},
		Target: config.Target{
			XMLDump: targetRoot,
		},
	}, root)
	if err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	if decision := state.decisions["Catalog.ИзTarget"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected target merge to revive soft-excluded target ref as AdoptedStub, got %#v", decision)
	}

	definedTypeDoc := etree.NewDocument()
	if err := definedTypeDoc.ReadFromFile(filepath.Join(root, "DefinedTypes", "ЦелевойТип.xml")); err != nil {
		t.Fatalf("read defined type xml: %v", err)
	}
	typeEl := definedTypeDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")
	if typeEl == nil {
		t.Fatalf("expected merged DefinedType type element")
	}
	if got := typeEl.SelectAttrValue("xsi:type", ""); got != "xr:ExtendedProperty" {
		t.Fatalf("expected merged DefinedType type to be xr:ExtendedProperty, got %q", got)
	}
	checkValues := map[string]struct{}{}
	for _, el := range typeEl.FindElements("./*[local-name()='CheckValue']/*") {
		checkValues[strings.TrimSpace(el.Text())] = struct{}{}
	}
	if _, ok := checkValues["cfg:CatalogRef.ИзTarget"]; !ok {
		t.Fatalf("expected merged DefinedType CheckValue to retain target ref, got %#v", checkValues)
	}
	extendValues := typeEl.FindElements("./*[local-name()='ExtendValue']/*")
	if len(extendValues) != 0 {
		t.Fatalf("did not expect source additions in ExtendValue, got %d", len(extendValues))
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	if !hasConfigurationChildObject(configurationDoc, "Catalog", "ИзTarget") {
		t.Fatalf("expected revived target ref in Configuration.xml ChildObjects")
	}
}

func TestBuildChangeFilesStateTargetMergedDefinedTypeUsesExtendedProperty(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <DefinedType>ВерсионируемыеДанные</DefinedType>
      <Catalog>ИзSource</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="DefinedType.ВерсионируемыеДанные" id="11111111-1111-1111-1111-111111111111"/>
    <Metadata name="Catalog.ИзSource" id="22222222-2222-2222-2222-222222222222"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("DefinedTypes", "ВерсионируемыеДанные.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType>
    <Properties>
      <Name>ВерсионируемыеДанные</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзSource</Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(root, filepath.Join("Catalogs", "ИзSource.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="22222222-2222-2222-2222-222222222222"><Properties><Name>ИзSource</Name></Properties></Catalog></MetaDataObject>`)
	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ОпределяемыйТип": {
    "ВерсионируемыеДанные": {
      "Тип": []
    }
  }
}`)

	writeFile(targetRoot, filepath.Join("DefinedTypes", "ВерсионируемыеДанные.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ВерсионируемыеДанные</Name>
      <Type>
        <Type>cfg:CatalogRef.ИзTarget</Type>
      </Type>
    </Properties>
  </DefinedType>
</MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "ИзTarget.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="33333333-3333-3333-3333-333333333333"><Properties><Name>ИзTarget</Name></Properties></Catalog></MetaDataObject>`)

	if _, err := buildChangeFilesState(&config.Configuration{
		IncludedNativeObjects: []string{
			"Catalog.ИзSource",
		},
		Target: config.Target{
			XMLDump: targetRoot,
		},
	}, root); err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	definedTypeDoc := etree.NewDocument()
	definedTypePath := filepath.Join(root, "DefinedTypes", "ВерсионируемыеДанные.xml")
	if err := definedTypeDoc.ReadFromFile(definedTypePath); err != nil {
		t.Fatalf("read defined type xml: %v", err)
	}
	typeEl := definedTypeDoc.FindElement("//*[local-name()='DefinedType']/*[local-name()='Properties']/*[local-name()='Type']")
	if typeEl == nil {
		t.Fatalf("expected merged DefinedType type element in %s", definedTypePath)
	}
	if got := typeEl.SelectAttrValue("xsi:type", ""); got != "xr:ExtendedProperty" {
		t.Fatalf("expected Properties/Type to be xr:ExtendedProperty, got %q", got)
	}
	if typeEl.FindElement("./*[local-name()='CheckValue']") == nil || typeEl.FindElement("./*[local-name()='ExtendValue']") == nil {
		t.Fatalf("expected split CheckValue/ExtendValue serialization in %s", definedTypePath)
	}
}

func TestBuildChangeFilesStateMarksExchangePlanAsAdoptedStubExtMetaData(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <ExchangePlan>ЦелевойПлан</ExchangePlan>
      <Catalog>ИзSource</Catalog>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="ExchangePlan.ЦелевойПлан" id="11111111-1111-1111-1111-111111111111"/>
    <Metadata name="Catalog.ИзSource" id="22222222-2222-2222-2222-222222222222"/>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("ExchangePlans", "ЦелевойПлан.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ExchangePlan uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ЦелевойПлан</Name>
    </Properties>
    <ChildObjects/>
  </ExchangePlan>
</MetaDataObject>`)
	writeFile(root, filepath.Join("ExchangePlans", "ЦелевойПлан", "Ext", "Content.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<Content xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Item><Metadata>Catalog.ИзSource</Metadata></Item>
</Content>`)
	writeFile(root, filepath.Join("Catalogs", "ИзSource.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="22222222-2222-2222-2222-222222222222"><Properties><Name>ИзSource</Name></Properties></Catalog></MetaDataObject>`)
	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ПланОбмена": {
    "ЦелевойПлан": {
      "Состав": []
    }
  }
}`)

	writeFile(targetRoot, filepath.Join("ExchangePlans", "ЦелевойПлан.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ExchangePlan uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>ЦелевойПлан</Name>
    </Properties>
    <ChildObjects/>
  </ExchangePlan>
</MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("ExchangePlans", "ЦелевойПлан", "Ext", "Content.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<Content xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Item><Metadata>Catalog.ИзTarget</Metadata></Item>
</Content>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "ИзTarget.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog uuid="33333333-3333-3333-3333-333333333333"><Properties><Name>ИзTarget</Name></Properties></Catalog></MetaDataObject>`)

	state, err := buildChangeFilesState(&config.Configuration{
		IncludedAdoptedStubObjects: []string{
			"Catalog.ИзSource",
		},
		Target: config.Target{
			XMLDump: targetRoot,
		},
	}, root)
	if err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	decision := state.decisions["ExchangePlan.ЦелевойПлан"]
	if decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected ExchangePlan target merge object to stay adopted, got %#v", decision)
	}
	if decision.Truncated {
		t.Fatalf("expected ExchangePlan target merge object not to be truncated")
	}
	if !decision.AdoptedStubExtMetaData {
		t.Fatalf("expected ExchangePlan target merge object to be marked as AdoptedStubExtMetaData")
	}

	contentCtx := findContextByRelPath(state.indexes, state.contexts, "ExchangePlans/ЦелевойПлан/Ext/Content.xml")
	if contentCtx == nil || contentCtx.Doc == nil {
		t.Fatalf("expected merged ExchangePlan content xml")
	}
	metadataValues := make(map[string]struct{})
	for _, el := range contentCtx.Doc.FindElements("//*[local-name()='Item']/*[local-name()='Metadata']") {
		metadataValues[strings.TrimSpace(el.Text())] = struct{}{}
	}
	for _, expected := range []string{
		"Catalog.ИзSource",
		"Catalog.ИзTarget",
	} {
		if _, ok := metadataValues[expected]; !ok {
			t.Fatalf("expected merged ExchangePlan content to contain %s, got %#v", expected, metadataValues)
		}
	}

	if decision := state.decisions["Catalog.ИзTarget"]; decision.Excluded || decision.Belonging != "AdoptedStub" {
		t.Fatalf("expected target-ref-driven catalog from ExchangePlan content, got %#v", decision)
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}
	if got := metadataEntryID(configDumpDoc, "Catalog.ИзTarget"); got != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("expected ExchangePlan target-ref-driven metadata id from target xml, got %q", got)
	}
}

func TestBuildChangeFilesStateDoesNotImportForbiddenTargetRef(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Configuration><ChildObjects><DefinedType>ЦелевойТип</DefinedType></ChildObjects></Configuration></MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions><Metadata name="DefinedType.ЦелевойТип" id="11111111-1111-1111-1111-111111111111"/></ConfigVersions></ConfigDumpInfo>`)
	writeFile(root, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType><Properties><Name>ЦелевойТип</Name><Type/></Properties></DefinedType></MetaDataObject>`)
	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ОпределяемыйТип": {
    "ЦелевойТип": {
      "Тип": []
    }
  }
}`)

	writeFile(targetRoot, filepath.Join("DefinedTypes", "ЦелевойТип.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><DefinedType uuid="11111111-1111-1111-1111-111111111111"><Properties><Name>ЦелевойТип</Name><Type><Type>cfg:CatalogRef.Запрещенный</Type></Type></Properties></DefinedType></MetaDataObject>`)
	writeFile(targetRoot, filepath.Join("Catalogs", "Запрещенный.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses"><Catalog><Properties><Name>Запрещенный</Name></Properties></Catalog></MetaDataObject>`)

	state, err := buildChangeFilesState(&config.Configuration{
		IncludedAdoptedStubObjects: []string{
			"DefinedType.ЦелевойТип",
		},
		ForbiddenAdoptedStubObjects: []string{
			"Catalog.Запрещенный",
		},
		Target: config.Target{XMLDump: targetRoot},
	}, root)
	if err != nil {
		t.Fatalf("build change files state: %v", err)
	}

	if decision, exists := state.decisions["Catalog.Запрещенный"]; exists && !decision.Excluded {
		t.Fatalf("expected forbidden target-ref-driven object to stay excluded")
	}
	if _, err := os.Stat(filepath.Join(root, "Catalogs", "Запрещенный.xml")); !os.IsNotExist(err) {
		t.Fatalf("did not expect forbidden target-ref-driven file to be created")
	}
}

func TestChangeFilesKeepsCanonicalRenamedTargetSensitiveMetaDataFileObjects(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	targetRoot := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	type targetSensitiveObject struct {
		kind          string
		sourceName    string
		canonicalName string
		uuid          string
		sourceXML     string
		targetXML     string
		sourceExtra   map[string]string
		targetExtra   map[string]string
	}

	topLevelDirByKind := map[string]string{
		"DefinedType":       "DefinedTypes",
		"EventSubscription": "EventSubscriptions",
		"ExchangePlan":      "ExchangePlans",
	}

	objects := []targetSensitiveObject{
		{
			kind:          "DefinedType",
			sourceName:    "ЗначениеДоступаОбъект",
			canonicalName: "ЗначениеДоступаОбъектКанон",
			uuid:          "11111111-1111-1111-1111-111111111111",
			sourceXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="11111111-1111-1111-1111-111111111111">
    <Properties><Name>ЗначениеДоступаОбъект</Name><Type/></Properties>
  </DefinedType>
</MetaDataObject>`,
			targetXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="11111111-1111-1111-1111-111111111111">
    <Properties><Name>ЗначениеДоступаОбъектКанон</Name><Type/></Properties>
  </DefinedType>
</MetaDataObject>`,
		},
		{
			kind:          "DefinedType",
			sourceName:    "ПредметЗаметок",
			canonicalName: "ПредметЗаметокКанон",
			uuid:          "22222222-2222-2222-2222-222222222222",
			sourceXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="22222222-2222-2222-2222-222222222222">
    <Properties><Name>ПредметЗаметок</Name><Type/></Properties>
  </DefinedType>
</MetaDataObject>`,
			targetXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <DefinedType uuid="22222222-2222-2222-2222-222222222222">
    <Properties><Name>ПредметЗаметокКанон</Name><Type/></Properties>
  </DefinedType>
</MetaDataObject>`,
		},
		{
			kind:          "ExchangePlan",
			sourceName:    "ИнтеграцияС1СДокументооборотомПереопределяемый",
			canonicalName: "ИнтеграцияС1СДокументооборотомПереопределяемыйКанон",
			uuid:          "33333333-3333-3333-3333-333333333333",
			sourceXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ExchangePlan uuid="33333333-3333-3333-3333-333333333333">
    <Properties><Name>ИнтеграцияС1СДокументооборотомПереопределяемый</Name></Properties>
    <ChildObjects/>
  </ExchangePlan>
</MetaDataObject>`,
			targetXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ExchangePlan uuid="33333333-3333-3333-3333-333333333333">
    <Properties><Name>ИнтеграцияС1СДокументооборотомПереопределяемыйКанон</Name></Properties>
    <ChildObjects/>
  </ExchangePlan>
</MetaDataObject>`,
			sourceExtra: map[string]string{
				filepath.Join("ExchangePlans", "ИнтеграцияС1СДокументооборотомПереопределяемый", "Ext", "Content.xml"): `<?xml version="1.0" encoding="UTF-8"?>
<Content xmlns="http://v8.1c.ru/8.3/MDClasses"/>`,
			},
			targetExtra: map[string]string{
				filepath.Join("ExchangePlans", "ИнтеграцияС1СДокументооборотомПереопределяемыйКанон", "Ext", "Content.xml"): `<?xml version="1.0" encoding="UTF-8"?>
<Content xmlns="http://v8.1c.ru/8.3/MDClasses"/>`,
			},
		},
		{
			kind:          "ExchangePlan",
			sourceName:    "ОбновлениеИнформационнойБазы",
			canonicalName: "ОбновлениеИнформационнойБазыКанон",
			uuid:          "44444444-4444-4444-4444-444444444444",
			sourceXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ExchangePlan uuid="44444444-4444-4444-4444-444444444444">
    <Properties><Name>ОбновлениеИнформационнойБазы</Name></Properties>
    <ChildObjects/>
  </ExchangePlan>
</MetaDataObject>`,
			targetXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ExchangePlan uuid="44444444-4444-4444-4444-444444444444">
    <Properties><Name>ОбновлениеИнформационнойБазыКанон</Name></Properties>
    <ChildObjects/>
  </ExchangePlan>
</MetaDataObject>`,
			sourceExtra: map[string]string{
				filepath.Join("ExchangePlans", "ОбновлениеИнформационнойБазы", "Ext", "Content.xml"): `<?xml version="1.0" encoding="UTF-8"?>
<Content xmlns="http://v8.1c.ru/8.3/MDClasses"/>`,
			},
			targetExtra: map[string]string{
				filepath.Join("ExchangePlans", "ОбновлениеИнформационнойБазыКанон", "Ext", "Content.xml"): `<?xml version="1.0" encoding="UTF-8"?>
<Content xmlns="http://v8.1c.ru/8.3/MDClasses"/>`,
			},
		},
		{
			kind:          "EventSubscription",
			sourceName:    "ИнтеграцияС1СДокументооборотомПодписка",
			canonicalName: "ИнтеграцияС1СДокументооборотомПодпискаКанон",
			uuid:          "55555555-5555-5555-5555-555555555555",
			sourceXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <EventSubscription uuid="55555555-5555-5555-5555-555555555555">
    <Properties><Name>ИнтеграцияС1СДокументооборотомПодписка</Name><Source/></Properties>
  </EventSubscription>
</MetaDataObject>`,
			targetXML: `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <EventSubscription uuid="55555555-5555-5555-5555-555555555555">
    <Properties><Name>ИнтеграцияС1СДокументооборотомПодпискаКанон</Name><Source/></Properties>
  </EventSubscription>
</MetaDataObject>`,
		},
	}

	configurationChildren := make([]string, 0, len(objects))
	sourceConfigDumpEntries := make([]string, 0, len(objects))
	targetConfigDumpEntries := make([]string, 0, len(objects))
	for _, object := range objects {
		tag, ok := configurationChildObjectTag(object.kind)
		if !ok {
			t.Fatalf("missing configuration child object tag for %s", object.kind)
		}
		relDir, ok := topLevelDirByKind[object.kind]
		if !ok {
			t.Fatalf("missing top-level dir for %s", object.kind)
		}

		configurationChildren = append(configurationChildren, fmt.Sprintf("<%s>%s</%s>", tag, object.sourceName, tag))
		sourceConfigDumpEntries = append(sourceConfigDumpEntries, fmt.Sprintf(`<Metadata name="%s.%s" id="%s"/>`, object.kind, object.sourceName, object.uuid))
		targetConfigDumpEntries = append(targetConfigDumpEntries, fmt.Sprintf(`<Metadata name="%s.%s" id="%s"/>`, object.kind, object.canonicalName, object.uuid))

		writeFile(root, filepath.Join(relDir, object.sourceName+".xml"), object.sourceXML)
		writeFile(targetRoot, filepath.Join(relDir, object.canonicalName+".xml"), object.targetXML)
		for relPath, content := range object.sourceExtra {
			writeFile(root, relPath, content)
		}
		for relPath, content := range object.targetExtra {
			writeFile(targetRoot, relPath, content)
		}
	}

	writeFile(root, "Configuration.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration><ChildObjects>%s</ChildObjects></Configuration>
</MetaDataObject>`, strings.Join(configurationChildren, "")))
	writeFile(root, "ConfigDumpInfo.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions>%s</ConfigVersions></ConfigDumpInfo>`, strings.Join(sourceConfigDumpEntries, "")))
	writeFile(targetRoot, "ConfigDumpInfo.xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo"><ConfigVersions>%s</ConfigVersions></ConfigDumpInfo>`, strings.Join(targetConfigDumpEntries, "")))
	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ПланОбмена": {
    "ИнтеграцияС1СДокументооборотомПереопределяемый": {
      "Состав": []
    },
    "ОбновлениеИнформационнойБазы": {
      "Состав": []
    }
  },
  "ОпределяемыйТип": {
    "ЗначениеДоступаОбъект": {
      "Тип": []
    },
    "ПредметЗаметок": {
      "Тип": []
    }
  },
  "ПодпискаНаСобытие": {
    "ИнтеграцияС1СДокументооборотомПодписка": {
      "Источник": []
    }
  }
}`)

	if err := ChangeFiles(&config.Configuration{
		Target: config.Target{XMLDump: targetRoot},
		AdditionalProcessing: config.AdditionalProcessing{
			UseMetaDataFile: true,
		},
	}, root); err != nil {
		t.Fatalf("change files: %v", err)
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}

	for _, object := range objects {
		tag, ok := configurationChildObjectTag(object.kind)
		if !ok {
			t.Fatalf("missing configuration child object tag for %s", object.kind)
		}
		relDir, ok := topLevelDirByKind[object.kind]
		if !ok {
			t.Fatalf("missing top-level dir for %s", object.kind)
		}

		if hasConfigurationChildObject(configurationDoc, tag, object.sourceName) {
			t.Fatalf("did not expect stale configuration child object %s.%s", object.kind, object.sourceName)
		}
		if !hasConfigurationChildObject(configurationDoc, tag, object.canonicalName) {
			t.Fatalf("expected canonical configuration child object %s.%s", object.kind, object.canonicalName)
		}

		sourceKey := object.kind + "." + object.sourceName
		canonicalKey := object.kind + "." + object.canonicalName
		if hasMetadataName(configDumpDoc, sourceKey) {
			t.Fatalf("did not expect stale ConfigDumpInfo entry %s", sourceKey)
		}
		if !hasMetadataName(configDumpDoc, canonicalKey) {
			t.Fatalf("expected canonical ConfigDumpInfo entry %s", canonicalKey)
		}

		if _, err := os.Stat(filepath.Join(root, relDir, object.canonicalName+".xml")); err != nil {
			t.Fatalf("expected canonical top-level xml for %s: %v", canonicalKey, err)
		}
		if _, err := os.Stat(filepath.Join(root, relDir, object.sourceName+".xml")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("did not expect stale top-level xml for %s, err=%v", sourceKey, err)
		}
	}

}

func TestChangeFilesKeepsSoftExcludedMetaDataFileOwnerObjectWithNativeLeafFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile := func(base, relPath, content string) {
		t.Helper()
		path := filepath.Join(base, relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(root, "Configuration.xml", `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Configuration>
    <ChildObjects>
      <Document>Бюджет</Document>
    </ChildObjects>
  </Configuration>
</MetaDataObject>`)
	writeFile(root, "ConfigDumpInfo.xml", `<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo xmlns="http://v8.1c.ru/8.3/xcf/dumpinfo">
  <ConfigVersions>
    <Metadata name="Document.Бюджет" id="11111111-1111-1111-1111-111111111111">
      <Metadata name="Document.Бюджет.Attribute.упо_План" id="22222222-2222-2222-2222-222222222222"/>
      <Metadata name="Document.Бюджет.TabularSection.Поступления" id="33333333-3333-3333-3333-333333333333">
        <Metadata name="Document.Бюджет.TabularSection.Поступления.Attribute.упо_План" id="44444444-4444-4444-4444-444444444444"/>
      </Metadata>
    </Metadata>
  </ConfigVersions>
</ConfigDumpInfo>`)
	writeFile(root, filepath.Join("Documents", "Бюджет.xml"), `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Document uuid="11111111-1111-1111-1111-111111111111">
    <Properties>
      <Name>Бюджет</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <Attribute>
        <Properties>
          <Name>упо_План</Name>
          <ObjectBelonging>Adopted</ObjectBelonging>
        </Properties>
      </Attribute>
      <TabularSection>
        <Properties>
          <Name>Поступления</Name>
          <ObjectBelonging>Adopted</ObjectBelonging>
        </Properties>
        <ChildObjects>
          <Attribute>
            <Properties>
              <Name>упо_План</Name>
              <ObjectBelonging>Adopted</ObjectBelonging>
            </Properties>
          </Attribute>
        </ChildObjects>
      </TabularSection>
    </ChildObjects>
  </Document>
</MetaDataObject>`)
	writeFile(root, filepath.Join("CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt"), `{
  "ПланОбмена": {},
  "Документ": {
    "Бюджет": {
      "Реквизиты": [
        "Документ.Бюджет.Реквизит.упо_План"
      ],
      "ТабличныеЧасти": {
        "Поступления": [
          "Документ.Бюджет.ТабличнаяЧасть.Поступления.Реквизит.упо_План"
        ]
      }
    }
  }
}`)

	if err := ChangeFiles(&config.Configuration{
		ExcludedObjects: []string{
			"Document.Бюджет",
		},
		AdditionalProcessing: config.AdditionalProcessing{
			UseMetaDataFile: true,
		},
	}, root); err != nil {
		t.Fatalf("change files: %v", err)
	}

	configurationDoc := etree.NewDocument()
	if err := configurationDoc.ReadFromFile(filepath.Join(root, "Configuration.xml")); err != nil {
		t.Fatalf("read configuration xml: %v", err)
	}
	if !hasConfigurationChildObject(configurationDoc, "Document", "Бюджет") {
		t.Fatalf("expected Document.Бюджет to stay in Configuration.xml")
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromFile(filepath.Join(root, "ConfigDumpInfo.xml")); err != nil {
		t.Fatalf("read config dump info: %v", err)
	}
	for _, name := range []string{
		"Document.Бюджет",
		"Document.Бюджет.Attribute.упо_План",
		"Document.Бюджет.TabularSection.Поступления",
		"Document.Бюджет.TabularSection.Поступления.Attribute.упо_План",
	} {
		if !hasMetadataName(configDumpDoc, name) {
			t.Fatalf("expected ConfigDumpInfo to keep %s", name)
		}
	}

	documentDoc := etree.NewDocument()
	documentPath := filepath.Join(root, "Documents", "Бюджет.xml")
	if err := documentDoc.ReadFromFile(documentPath); err != nil {
		t.Fatalf("read document xml: %v", err)
	}

	ownerProps := documentDoc.FindElement("//*[local-name()='Document']/*[local-name()='Properties']")
	if ownerProps == nil {
		t.Fatalf("expected document properties in %s", documentPath)
	}
	if got := textOf(ownerProps, "ObjectBelonging"); got != "Adopted" {
		t.Fatalf("expected owner object to stay adopted, got ObjectBelonging=%q", got)
	}

	attrProps := documentDoc.FindElement("//*[local-name()='Document']/*[local-name()='ChildObjects']/*[local-name()='Attribute']/*[local-name()='Properties']")
	if attrProps == nil {
		t.Fatalf("expected retained owner attribute properties in %s", documentPath)
	}
	if got := textOf(attrProps, "ObjectBelonging"); got != "" {
		t.Fatalf("expected retained owner attribute to stay native leaf, got ObjectBelonging=%q", got)
	}

	sectionProps := documentDoc.FindElement("//*[local-name()='TabularSection']/*[local-name()='Properties']")
	if sectionProps == nil {
		t.Fatalf("expected retained tabular section properties in %s", documentPath)
	}
	if got := textOf(sectionProps, "ObjectBelonging"); got != "Adopted" {
		t.Fatalf("expected retained tabular section to stay adopted, got ObjectBelonging=%q", got)
	}

	sectionAttrProps := documentDoc.FindElement("//*[local-name()='TabularSection']/*[local-name()='ChildObjects']/*[local-name()='Attribute']/*[local-name()='Properties']")
	if sectionAttrProps == nil {
		t.Fatalf("expected retained tabular section attribute properties in %s", documentPath)
	}
	if got := textOf(sectionAttrProps, "ObjectBelonging"); got != "" {
		t.Fatalf("expected retained tabular section attribute to stay native leaf, got ObjectBelonging=%q", got)
	}
}

func TestCleanupForbiddenChildMetadataPathsRemovesForbiddenChildren(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Пользователи</Name>
    </Properties>
    <ChildObjects>
      <Form>ФормаВыбора</Form>
      <Command>
        <Properties>
          <Name>ПользователиИнформационнойБазы</Name>
        </Properties>
      </Command>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	changed := cleanupForbiddenChildMetadataPaths(doc, "Catalog.Пользователи", map[string]map[string]struct{}{
		"Catalog.Пользователи": {
			"Catalog.Пользователи.Form.ФормаВыбора":                       {},
			"Catalog.Пользователи.Command.ПользователиИнформационнойБазы": {},
		},
	})
	if !changed {
		t.Fatalf("expected forbidden child cleanup to report change")
	}

	childObjects := doc.FindElement("//*[local-name()='ChildObjects']")
	if childObjects == nil {
		t.Fatalf("expected ChildObjects")
	}
	if len(childObjects.ChildElements()) != 0 {
		t.Fatalf("expected forbidden children to be removed, got %d", len(childObjects.ChildElements()))
	}
}

func TestCleanupConfigDumpInfoForbiddenMetadataRemovesExactAndNestedEntries(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataInfo xmlns="http://v8.1c.ru/8.3/xcf/readable">
  <Metadata name="Catalog.Пользователи">
    <Metadata name="Catalog.Пользователи.Command.ПользователиИнформационнойБазы"/>
    <Metadata name="Catalog.Пользователи.Command.ПользователиИнформационнойБазы.CommandModule"/>
  </Metadata>
  <Metadata name="Catalog.Пользователи.Form.ФормаСписка"/>
</MetaDataInfo>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	if !cleanupConfigDumpInfoForbiddenMetadata(doc, map[string]struct{}{
		"Catalog.Пользователи.Command.ПользователиИнформационнойБазы": {},
	}) {
		t.Fatalf("expected forbidden ConfigDumpInfo cleanup to change document")
	}

	if hasMetadataName(doc, "Catalog.Пользователи.Command.ПользователиИнформационнойБазы") {
		t.Fatalf("expected forbidden command metadata to be removed")
	}
	if hasMetadataName(doc, "Catalog.Пользователи.Command.ПользователиИнформационнойБазы.CommandModule") {
		t.Fatalf("expected nested forbidden command module metadata to be removed")
	}
	if !hasMetadataName(doc, "Catalog.Пользователи.Form.ФормаСписка") {
		t.Fatalf("expected unrelated metadata to remain")
	}
}

func TestCollectAdoptedCodeModulePathsMarksNonNativeModules(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	objectPath := filepath.Join(root, "Catalogs", "Тест.xml")
	objectDir := filepath.Join(root, "Catalogs", "Тест", "Ext")
	if err := os.MkdirAll(objectDir, 0o755); err != nil {
		t.Fatalf("mkdir object dir: %v", err)
	}
	for _, name := range []string{"ManagerModule.bsl", "ObjectModule.bsl", "ValueManagerModule.bsl"} {
		if err := os.WriteFile(filepath.Join(objectDir, name), []byte("// module"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	contexts := []*FileProcessingContext{{
		Path:             objectPath,
		Metadata:         true,
		TopLevelMetadata: true,
		OwnerKey:         "Catalog.Тест",
	}}
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Belonging: "AdoptedStub"},
	}
	excluded := map[string]struct{}{}

	collectAdoptedCodeModulePaths(contexts, decisions, excluded)

	for _, name := range []string{"ManagerModule.bsl", "ObjectModule.bsl", "ValueManagerModule.bsl"} {
		path := filepath.Join(objectDir, name)
		if _, ok := excluded[path]; !ok {
			t.Fatalf("expected %s to be marked for exclusion", path)
		}
	}
}

func TestRoleMetadataTargetExistsForCommandModuleOnlyCommand(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	objectDir := filepath.Join(root, "Catalogs", "Пользователи", "Commands", "ПользователиИнформационнойБазы", "Ext")
	if err := os.MkdirAll(objectDir, 0o755); err != nil {
		t.Fatalf("mkdir command dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(objectDir, "CommandModule.bsl"), []byte("// command module"), 0o644); err != nil {
		t.Fatalf("write command module: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Пользователи</Name>
    </Properties>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:      doc,
			Path:     filepath.Join(root, "Catalogs", "Пользователи.xml"),
			Metadata: true,
			OwnerKey: "Catalog.Пользователи",
		},
	}

	if !roleMetadataTargetExists("Catalog.Пользователи.Command.ПользователиИнформационнойБазы", contexts) {
		t.Fatalf("expected command metadata target to exist from command module directory")
	}
	if !roleMetadataTargetExists("Catalog.Пользователи.Command.ПользователиИнформационнойБазы.CommandModule", contexts) {
		t.Fatalf("expected command module metadata target to exist from command module file")
	}
}

func TestNormalizeAdoptedObjectCompositionKeepsReferencedCommands(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Пользователи</Name>
    </Properties>
    <ChildObjects>
      <Command>
        <Properties>
          <Name>ПользователиИнформационнойБазы</Name>
        </Properties>
      </Command>
      <Command>
        <Properties>
          <Name>ЛишняяКоманда</Name>
        </Properties>
      </Command>
      <Attribute>
        <Properties>
          <Name>Недействителен</Name>
        </Properties>
      </Attribute>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	retained := map[string]struct{}{
		"ПользователиИнформационнойБазы": {},
	}
	if !normalizeAdoptedObjectComposition(doc, "Catalog", searchResultObjectOverlay{PreserveCommands: retained}) {
		t.Fatalf("expected adopted composition to change")
	}

	childObjects := doc.Root().FindElement("./Catalog/ChildObjects")
	if childObjects == nil {
		t.Fatalf("expected ChildObjects to remain")
	}
	commands := childObjects.FindElements("./Command")
	if len(commands) != 1 {
		t.Fatalf("expected exactly one retained command, got %d", len(commands))
	}
	if got := strings.TrimSpace(textOf(commands[0].FindElement("./Properties"), "Name")); got != "ПользователиИнформационнойБазы" {
		t.Fatalf("expected retained command to stay, got %q", got)
	}
}

func TestCleanupAdoptedObjectFormReferencesRemovesReportMainProperties(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Properties>
  <Name>ОтчетТест</Name>
  <DefaultObjectForm>Form.Default</DefaultObjectForm>
  <MainForm>Form.Main</MainForm>
  <MainSettingsForm>Form.Settings</MainSettingsForm>
  <MainVariantForm>Form.Variant</MainVariantForm>
  <MainDataCompositionSchema>DataCompositionSchema.Main</MainDataCompositionSchema>
</Properties>`); err != nil {
		t.Fatalf("read properties xml: %v", err)
	}

	properties := doc.Root()
	if !cleanupAdoptedObjectFormReferences(properties, "Report") {
		t.Fatalf("expected report property cleanup to change document")
	}

	for _, tag := range []string{
		"DefaultObjectForm",
		"MainForm",
		"MainSettingsForm",
		"MainVariantForm",
		"MainDataCompositionSchema",
	} {
		if properties.FindElement("./"+tag) != nil {
			t.Fatalf("expected %s to be removed for adopted report", tag)
		}
	}
	if got := strings.TrimSpace(textOf(properties, "Name")); got != "ОтчетТест" {
		t.Fatalf("expected unrelated property to remain, got %q", got)
	}
}

func TestCollectOwnerCommandCandidatesFromFunctionalOption(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <FunctionalOption>
    <Properties>
      <Name>СтандартныеПодсистемыВЛокальномРежиме</Name>
      <Content>
        <xr:Object>Catalog.Пользователи.Command.ПользователиИнформационнойБазы</xr:Object>
      </Content>
    </Properties>
  </FunctionalOption>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:      doc,
			RelPath:  "FunctionalOptions/СтандартныеПодсистемыВЛокальномРежиме.xml",
			Metadata: true,
			OwnerKey: "FunctionalOption.СтандартныеПодсистемыВЛокальномРежиме",
		},
	}
	decisions := map[string]objectDecision{
		"FunctionalOption.СтандартныеПодсистемыВЛокальномРежиме": {Belonging: "AdoptedStub"},
		"Catalog.Пользователи": {Belonging: "AdoptedStub"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	if _, ok := candidates["Catalog.Пользователи"]["ПользователиИнформационнойБазы"]; !ok {
		t.Fatalf("expected command candidate to be collected from functional option content")
	}
}

func TestPromoteReferencedObjectsFromFunctionalOptionStorageOnly(t *testing.T) {
	t.Parallel()

	functionalOptionDoc := etree.NewDocument()
	if err := functionalOptionDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <FunctionalOption>
    <Properties>
      <Name>ИспользоватьДатыЗапретаИзменения</Name>
      <Location>Constant.ИспользоватьДатыЗапретаИзменения</Location>
      <Content>
        <xr:Object>Report.ДатыЗапретаИзменения</xr:Object>
      </Content>
    </Properties>
  </FunctionalOption>
</MetaDataObject>`); err != nil {
		t.Fatalf("read functional option xml: %v", err)
	}

	constantDoc := etree.NewDocument()
	if err := constantDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Constant>
    <Properties>
      <Name>ИспользоватьДатыЗапретаИзменения</Name>
    </Properties>
  </Constant>
</MetaDataObject>`); err != nil {
		t.Fatalf("read constant xml: %v", err)
	}

	reportDoc := etree.NewDocument()
	if err := reportDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Report>
    <Properties>
      <Name>ДатыЗапретаИзменения</Name>
    </Properties>
  </Report>
</MetaDataObject>`); err != nil {
		t.Fatalf("read report xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:       functionalOptionDoc,
			RelPath:   "FunctionalOptions/ИспользоватьДатыЗапретаИзменения.xml",
			Metadata:  true,
			OwnerKey:  "FunctionalOption.ИспользоватьДатыЗапретаИзменения",
			OwnerKind: "FunctionalOption",
		},
		{
			Doc:       constantDoc,
			RelPath:   "Constants/ИспользоватьДатыЗапретаИзменения.xml",
			Metadata:  true,
			OwnerKey:  "Constant.ИспользоватьДатыЗапретаИзменения",
			OwnerKind: "Constant",
		},
		{
			Doc:       reportDoc,
			RelPath:   "Reports/ДатыЗапретаИзменения.xml",
			Metadata:  true,
			OwnerKey:  "Report.ДатыЗапретаИзменения",
			OwnerKind: "Report",
		},
	}
	decisions := map[string]objectDecision{
		"FunctionalOption.ИспользоватьДатыЗапретаИзменения": {Belonging: "AdoptedStub"},
		"Constant.ИспользоватьДатыЗапретаИзменения":         {Excluded: true},
		"Report.ДатыЗапретаИзменения":                       {Excluded: true},
	}

	referenceGraph := collectReferenceGraph(contexts, &config.Configuration{}, nil, nil, nil)
	incomingReferenceGraph := collectIncomingReferenceGraph(referenceGraph)

	promoteReferencedObjectsToAdoptedStubIndexed(
		contexts,
		buildContextIndexes(contexts),
		decisions,
		&config.Configuration{},
		referenceGraph,
		incomingReferenceGraph,
		nil,
		nil,
		nil,
		nil,
		nil,
		targetCompatibilitySet{},
	)

	constantDecision := decisions["Constant.ИспользоватьДатыЗапретаИзменения"]
	if constantDecision.Excluded || constantDecision.Belonging != "AdoptedStub" {
		t.Fatalf("expected storage constant to be promoted to AdoptedStub, got %#v", constantDecision)
	}

	reportDecision := decisions["Report.ДатыЗапретаИзменения"]
	if !reportDecision.Excluded {
		t.Fatalf("expected functional option content report to stay excluded, got %#v", reportDecision)
	}
}

func TestCollectOwnerCommandCandidatesFromRetainedOwnerForm(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Form>
    <ChildItems>
      <Item>
        <CommandName>Catalog.Пользователи.Command.ПользователиИнформационнойБазы</CommandName>
      </Item>
    </ChildItems>
  </Form>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:      doc,
			RelPath:  "Catalogs/Пользователи/Forms/ФормаСписка/Ext/Form.xml",
			Metadata: false,
			OwnerKey: "Catalog.Пользователи",
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Пользователи": {Belonging: "Adopted"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	if _, ok := candidates["Catalog.Пользователи"]["ПользователиИнформационнойБазы"]; !ok {
		t.Fatalf("expected command candidate to be collected from owner form command reference")
	}
}

func TestCollectOwnerCommandCandidatesSkipsReportAndDataProcessorCommands(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Form>
    <ChildItems>
      <Item><CommandName>Report.ТестОтчет.Command.Сформировать</CommandName></Item>
      <Item><CommandName>DataProcessor.ТестОбработка.Command.Выполнить</CommandName></Item>
      <Item><CommandName>Catalog.ТестКаталог.Command.Оставить</CommandName></Item>
    </ChildItems>
  </Form>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	contexts := []*FileProcessingContext{{
		Doc:      doc,
		RelPath:  "CommonForms/ТестоваяФорма/Ext/Form.xml",
		Metadata: false,
		OwnerKey: "CommonForm.ТестоваяФорма",
	}}
	decisions := map[string]objectDecision{
		"Report.ТестОтчет":            {Belonging: "AdoptedStub"},
		"DataProcessor.ТестОбработка": {Belonging: "AdoptedStub"},
		"Catalog.ТестКаталог":         {Belonging: "AdoptedStub"},
		"CommonForm.ТестоваяФорма":    {Belonging: "AdoptedStub"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	if _, ok := candidates["Catalog.ТестКаталог"]["Оставить"]; !ok {
		t.Fatalf("expected non-report command candidate to remain")
	}
	if _, ok := candidates["Report.ТестОтчет"]; ok {
		t.Fatalf("did not expect report command candidates without SearchResult")
	}
	if _, ok := candidates["DataProcessor.ТестОбработка"]; ok {
		t.Fatalf("did not expect data processor command candidates without SearchResult")
	}
}

func TestFinalizeRetainedOwnerCommandsCleansMissingReportAndDataProcessorFormCommands(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		dir       string
		ownerKind string
	}{
		{name: "report", dir: "Reports", ownerKind: "Report"},
		{name: "data processor", dir: "DataProcessors", ownerKind: "DataProcessor"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			ownerName := "ТестОбъект"
			ownerKey := tc.ownerKind + "." + ownerName

			ownerDoc := etree.NewDocument()
			if err := ownerDoc.ReadFromString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <%[1]s>
    <Properties>
      <Name>%[2]s</Name>
    </Properties>
  </%[1]s>
</MetaDataObject>`, tc.ownerKind, ownerName)); err != nil {
				t.Fatalf("read owner xml: %v", err)
			}

			formDoc := etree.NewDocument()
			if err := formDoc.ReadFromString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform">
  <ChildItems>
    <Button name="Удаляемая">
      <CommandName>%[1]s.Command.Удалить</CommandName>
    </Button>
    <Button name="Оставляемая">
      <CommandName>Form.StandardCommand.Help</CommandName>
    </Button>
  </ChildItems>
</Form>`, ownerKey)); err != nil {
				t.Fatalf("read form xml: %v", err)
			}

			ownerRelPath := filepath.ToSlash(filepath.Join(tc.dir, ownerName+".xml"))
			ownerPath := filepath.Join(root, tc.dir, ownerName+".xml")
			if err := os.MkdirAll(filepath.Dir(ownerPath), 0o755); err != nil {
				t.Fatalf("mkdir owner dir: %v", err)
			}
			if err := ownerDoc.WriteToFile(ownerPath); err != nil {
				t.Fatalf("write owner xml: %v", err)
			}

			formRelPath := filepath.ToSlash(filepath.Join(tc.dir, ownerName, "Forms", "ОсновнаяФорма", "Ext", "Form.xml"))
			formPath := filepath.Join(root, tc.dir, ownerName, "Forms", "ОсновнаяФорма", "Ext", "Form.xml")
			if err := os.MkdirAll(filepath.Dir(formPath), 0o755); err != nil {
				t.Fatalf("mkdir form dir: %v", err)
			}
			if err := formDoc.WriteToFile(formPath); err != nil {
				t.Fatalf("write form xml: %v", err)
			}

			staleCommandModulePath := filepath.Join(root, tc.dir, ownerName, "Commands", "Удалить", "Ext", "CommandModule.bsl")
			if err := os.MkdirAll(filepath.Dir(staleCommandModulePath), 0o755); err != nil {
				t.Fatalf("mkdir stale command dir: %v", err)
			}
			if err := os.WriteFile(staleCommandModulePath, []byte("// stale command"), 0o644); err != nil {
				t.Fatalf("write stale command module: %v", err)
			}

			contexts := []*FileProcessingContext{
				{
					Doc:              ownerDoc,
					Path:             ownerPath,
					RelPath:          ownerRelPath,
					FileName:         ownerName + ".xml",
					Metadata:         true,
					TopLevelMetadata: true,
					OwnerKey:         ownerKey,
					OwnerKind:        tc.ownerKind,
				},
				{
					Doc:       formDoc,
					Path:      formPath,
					RelPath:   formRelPath,
					FileName:  "Form.xml",
					OwnerKey:  ownerKey,
					OwnerKind: tc.ownerKind,
				},
			}
			decisions := map[string]objectDecision{
				ownerKey: {Belonging: "AdoptedStub"},
			}

			if !roleMetadataTargetExists(ownerKey+".Command.Удалить", contexts) {
				t.Fatalf("expected raw resolver to see stale command on filesystem")
			}

			stats, err := finalizeRetainedOwnerCommands(contexts, buildContextIndexes(contexts), decisions, map[string]struct{}{}, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatalf("finalize retained commands: %v", err)
			}
			if stats.ChangedFiles == 0 || stats.WrittenFiles == 0 {
				t.Fatalf("expected finalization to rewrite stale form references, got changed=%d written=%d", stats.ChangedFiles, stats.WrittenFiles)
			}

			if hasFormElementWithName(formDoc, "Button", "Удаляемая") {
				t.Fatalf("expected stale command button to be removed")
			}
			if !hasFormElementWithName(formDoc, "Button", "Оставляемая") {
				t.Fatalf("expected unrelated button to stay")
			}
		})
	}
}

func TestFilterRetainedOwnerCommandsDropsCleanedFunctionalOptionReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Тест</Name>
    </Properties>
    <ChildObjects>
      <Command>
        <Properties>
          <Name>ТестКоманда</Name>
        </Properties>
      </Command>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	functionalOptionDoc := etree.NewDocument()
	if err := functionalOptionDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <FunctionalOption>
    <Properties>
      <Name>ТестОпция</Name>
      <Content>
        <xr:Object>Catalog.Тест.Command.ТестКоманда</xr:Object>
      </Content>
    </Properties>
  </FunctionalOption>
</MetaDataObject>`); err != nil {
		t.Fatalf("read functional option xml: %v", err)
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Тест" id="11111111-1111-1111-1111-111111111111">
    <Metadata name="Catalog.Тест.Command.ТестКоманда" id="22222222-2222-2222-2222-222222222222"/>
  </Metadata>
  <Metadata name="Catalog.Тест.Command.ТестКоманда" id="33333333-3333-3333-3333-333333333333"/>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	ownerPath := filepath.Join(root, "Catalogs", "Тест.xml")
	if err := os.MkdirAll(filepath.Dir(ownerPath), 0o755); err != nil {
		t.Fatalf("mkdir owner dir: %v", err)
	}
	if err := ownerDoc.WriteToFile(ownerPath); err != nil {
		t.Fatalf("write owner xml: %v", err)
	}

	configDumpPath := filepath.Join(root, configDumpInfo)
	if err := configDumpDoc.WriteToFile(configDumpPath); err != nil {
		t.Fatalf("write config dump xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:              ownerDoc,
			Path:             ownerPath,
			RelPath:          "Catalogs/Тест.xml",
			FileName:         "Тест.xml",
			Metadata:         true,
			TopLevelMetadata: true,
			OwnerKey:         "Catalog.Тест",
			OwnerKind:        "Catalog",
		},
		{
			Doc:       functionalOptionDoc,
			Path:      filepath.Join(root, "FunctionalOptions", "ТестОпция.xml"),
			RelPath:   "FunctionalOptions/ТестОпция.xml",
			FileName:  "ТестОпция.xml",
			Metadata:  true,
			OwnerKey:  "FunctionalOption.ТестОпция",
			OwnerKind: "FunctionalOption",
		},
		{
			Doc:      configDumpDoc,
			Path:     configDumpPath,
			RelPath:  configDumpInfo,
			FileName: configDumpInfo,
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест":               {Belonging: "AdoptedStub"},
		"FunctionalOption.ТестОпция": {Belonging: "AdoptedStub"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	if _, ok := candidates["Catalog.Тест"]["ТестКоманда"]; !ok {
		t.Fatalf("expected command candidate from raw functional option content")
	}

	if objectRef := functionalOptionDoc.FindElement("//*[local-name()='Content']/*[local-name()='Object']"); objectRef == nil {
		t.Fatalf("expected functional option content reference")
	} else {
		objectRef.SetText("")
	}

	liveRefs := buildLiveCommandReferenceIndex(contexts, decisions, map[string]struct{}{})
	retained := filterRetainedOwnerCommandsByLiveReferences(candidates, liveRefs)
	if len(retained["Catalog.Тест"]) != 0 {
		t.Fatalf("expected no live retained commands after cleanup, got %d", len(retained["Catalog.Тест"]))
	}

	stats, err := finalizeRetainedOwnerCommands(contexts, buildContextIndexes(contexts), decisions, map[string]struct{}{}, nil, nil, candidates, retained, liveRefs, nil, nil)
	if err != nil {
		t.Fatalf("finalize retained commands: %v", err)
	}
	if stats.ChangedFiles == 0 || stats.WrittenFiles == 0 {
		t.Fatalf("expected finalization to rewrite owner/config dump, got changed=%d written=%d", stats.ChangedFiles, stats.WrittenFiles)
	}

	if commands := ownerDoc.FindElements("//*[local-name()='Catalog']/*[local-name()='ChildObjects']/*[local-name()='Command']"); len(commands) != 0 {
		t.Fatalf("expected stale command to be removed from ChildObjects, got %d", len(commands))
	}
	if hasMetadataName(configDumpDoc, "Catalog.Тест.Command.ТестКоманда") {
		t.Fatalf("expected stale command metadata to be removed from ConfigDumpInfo")
	}
}

func TestFilterRetainedOwnerCommandsKeepsLiveNativeFormReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Тест</Name>
    </Properties>
    <ChildObjects>
      <Command>
        <Properties>
          <Name>ТестКоманда</Name>
        </Properties>
      </Command>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ChildItems>
    <Item>
      <Command>Catalog.Тест.Command.ТестКоманда</Command>
    </Item>
  </ChildItems>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Тест" id="11111111-1111-1111-1111-111111111111">
    <Metadata name="Catalog.Тест.Command.ТестКоманда" id="22222222-2222-2222-2222-222222222222"/>
  </Metadata>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	ownerPath := filepath.Join(root, "Catalogs", "Тест.xml")
	if err := os.MkdirAll(filepath.Dir(ownerPath), 0o755); err != nil {
		t.Fatalf("mkdir owner dir: %v", err)
	}
	if err := ownerDoc.WriteToFile(ownerPath); err != nil {
		t.Fatalf("write owner xml: %v", err)
	}

	configDumpPath := filepath.Join(root, configDumpInfo)
	if err := configDumpDoc.WriteToFile(configDumpPath); err != nil {
		t.Fatalf("write config dump xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:              ownerDoc,
			Path:             ownerPath,
			RelPath:          "Catalogs/Тест.xml",
			FileName:         "Тест.xml",
			Metadata:         true,
			TopLevelMetadata: true,
			OwnerKey:         "Catalog.Тест",
			OwnerKind:        "Catalog",
		},
		{
			Doc:       formDoc,
			Path:      filepath.Join(root, "Catalogs", "Носитель", "Forms", "ФормаСписка", "Ext", "Form.xml"),
			RelPath:   "Catalogs/Носитель/Forms/ФормаСписка/Ext/Form.xml",
			FileName:  "Form.xml",
			OwnerKey:  "Catalog.Носитель",
			OwnerKind: "Catalog",
		},
		{
			Doc:      configDumpDoc,
			Path:     configDumpPath,
			RelPath:  configDumpInfo,
			FileName: configDumpInfo,
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест":     {Belonging: "AdoptedStub"},
		"Catalog.Носитель": {Belonging: "Native"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	liveRefs := buildLiveCommandReferenceIndex(contexts, decisions, map[string]struct{}{})
	retained := filterRetainedOwnerCommandsByLiveReferences(candidates, liveRefs)
	if _, ok := retained["Catalog.Тест"]["ТестКоманда"]; !ok {
		t.Fatalf("expected live native form reference to retain adopted owner command")
	}

	if _, err := finalizeRetainedOwnerCommands(contexts, buildContextIndexes(contexts), decisions, map[string]struct{}{}, nil, nil, candidates, retained, liveRefs, nil, nil); err != nil {
		t.Fatalf("finalize retained commands: %v", err)
	}

	if commands := ownerDoc.FindElements("//*[local-name()='Catalog']/*[local-name()='ChildObjects']/*[local-name()='Command']"); len(commands) != 1 {
		t.Fatalf("expected retained command to stay in ChildObjects, got %d", len(commands))
	}
	if !hasMetadataName(configDumpDoc, "Catalog.Тест.Command.ТестКоманда") {
		t.Fatalf("expected retained command metadata to stay in ConfigDumpInfo")
	}
}

func TestFilterRetainedOwnerCommandsKeepsLiveCommandInterfaceAttributeReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Тест</Name>
    </Properties>
    <ChildObjects>
      <Command>
        <Properties>
          <Name>ТестКоманда</Name>
        </Properties>
      </Command>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	commandInterfaceDoc := etree.NewDocument()
	if err := commandInterfaceDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<CommandInterface xmlns="http://v8.1c.ru/8.3/MDClasses">
  <CommandsVisibility>
    <Command name="Catalog.Тест.Command.ТестКоманда"/>
  </CommandsVisibility>
</CommandInterface>`); err != nil {
		t.Fatalf("read command interface xml: %v", err)
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Тест" id="11111111-1111-1111-1111-111111111111">
    <Metadata name="Catalog.Тест.Command.ТестКоманда" id="22222222-2222-2222-2222-222222222222"/>
  </Metadata>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	ownerPath := filepath.Join(root, "Catalogs", "Тест.xml")
	if err := os.MkdirAll(filepath.Dir(ownerPath), 0o755); err != nil {
		t.Fatalf("mkdir owner dir: %v", err)
	}
	if err := ownerDoc.WriteToFile(ownerPath); err != nil {
		t.Fatalf("write owner xml: %v", err)
	}

	commandInterfacePath := filepath.Join(root, "Ext", "CommandInterface.xml")
	if err := os.MkdirAll(filepath.Dir(commandInterfacePath), 0o755); err != nil {
		t.Fatalf("mkdir command interface dir: %v", err)
	}
	if err := commandInterfaceDoc.WriteToFile(commandInterfacePath); err != nil {
		t.Fatalf("write command interface xml: %v", err)
	}

	configDumpPath := filepath.Join(root, configDumpInfo)
	if err := configDumpDoc.WriteToFile(configDumpPath); err != nil {
		t.Fatalf("write config dump xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:              ownerDoc,
			Path:             ownerPath,
			RelPath:          "Catalogs/Тест.xml",
			FileName:         "Тест.xml",
			Metadata:         true,
			TopLevelMetadata: true,
			OwnerKey:         "Catalog.Тест",
			OwnerKind:        "Catalog",
		},
		{
			Doc:      commandInterfaceDoc,
			Path:     commandInterfacePath,
			RelPath:  "Ext/CommandInterface.xml",
			FileName: "CommandInterface.xml",
		},
		{
			Doc:      configDumpDoc,
			Path:     configDumpPath,
			RelPath:  configDumpInfo,
			FileName: configDumpInfo,
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Belonging: "AdoptedStub"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	liveRefs := buildLiveCommandReferenceIndex(contexts, decisions, map[string]struct{}{})
	retained := filterRetainedOwnerCommandsByLiveReferences(candidates, liveRefs)
	if _, ok := retained["Catalog.Тест"]["ТестКоманда"]; !ok {
		t.Fatalf("expected live command interface attribute reference to retain adopted owner command")
	}

	if _, err := finalizeRetainedOwnerCommands(contexts, buildContextIndexes(contexts), decisions, map[string]struct{}{}, nil, nil, candidates, retained, liveRefs, nil, nil); err != nil {
		t.Fatalf("finalize retained commands: %v", err)
	}

	if commands := ownerDoc.FindElements("//*[local-name()='Catalog']/*[local-name()='ChildObjects']/*[local-name()='Command']"); len(commands) != 1 {
		t.Fatalf("expected retained command to stay in ChildObjects, got %d", len(commands))
	}
	if !hasMetadataName(configDumpDoc, "Catalog.Тест.Command.ТестКоманда") {
		t.Fatalf("expected retained command metadata to stay in ConfigDumpInfo")
	}
}

func TestFilterRetainedOwnerCommandsSkipsExcludedAdoptedOwnerForm(t *testing.T) {
	t.Parallel()

	formDoc := etree.NewDocument()
	if err := formDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/MDClasses">
  <ChildItems>
    <Item>
      <Command>Catalog.Тест.Command.ТестКоманда</Command>
    </Item>
  </ChildItems>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:       formDoc,
			Path:      filepath.Join(t.TempDir(), "Catalogs", "Тест", "Forms", "ФормаСписка", "Ext", "Form.xml"),
			RelPath:   "Catalogs/Тест/Forms/ФормаСписка/Ext/Form.xml",
			FileName:  "Form.xml",
			OwnerKey:  "Catalog.Тест",
			OwnerKind: "Catalog",
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Belonging: "AdoptedStub"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	if _, ok := candidates["Catalog.Тест"]["ТестКоманда"]; !ok {
		t.Fatalf("expected command candidate from raw adopted owner form")
	}

	liveRefs := buildLiveCommandReferenceIndex(contexts, decisions, map[string]struct{}{
		contexts[0].Path: {},
	})
	retained := filterRetainedOwnerCommandsByLiveReferences(candidates, liveRefs)
	if len(retained["Catalog.Тест"]) != 0 {
		t.Fatalf("expected excluded adopted owner form not to keep command alive")
	}
}

func TestFilterRetainedOwnerCommandsIgnoresRightsReference(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	ownerDoc := etree.NewDocument()
	if err := ownerDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Тест</Name>
    </Properties>
    <ChildObjects>
      <Command>
        <Properties>
          <Name>ТестКоманда</Name>
        </Properties>
      </Command>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read owner xml: %v", err)
	}

	rightsDoc := etree.NewDocument()
	if err := rightsDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Rights xmlns="http://v8.1c.ru/8.2/roles">
  <object>
    <name>Catalog.Тест.Command.ТестКоманда</name>
    <right>Read</right>
  </object>
</Rights>`); err != nil {
		t.Fatalf("read rights xml: %v", err)
	}

	configDumpDoc := etree.NewDocument()
	if err := configDumpDoc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Тест" id="11111111-1111-1111-1111-111111111111">
    <Metadata name="Catalog.Тест.Command.ТестКоманда" id="22222222-2222-2222-2222-222222222222"/>
  </Metadata>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump xml: %v", err)
	}

	ownerPath := filepath.Join(root, "Catalogs", "Тест.xml")
	if err := os.MkdirAll(filepath.Dir(ownerPath), 0o755); err != nil {
		t.Fatalf("mkdir owner dir: %v", err)
	}
	if err := ownerDoc.WriteToFile(ownerPath); err != nil {
		t.Fatalf("write owner xml: %v", err)
	}

	rightsPath := filepath.Join(root, "Roles", "ТестРоль", "Ext", "Rights.xml")
	if err := os.MkdirAll(filepath.Dir(rightsPath), 0o755); err != nil {
		t.Fatalf("mkdir rights dir: %v", err)
	}
	if err := rightsDoc.WriteToFile(rightsPath); err != nil {
		t.Fatalf("write rights xml: %v", err)
	}

	configDumpPath := filepath.Join(root, configDumpInfo)
	if err := configDumpDoc.WriteToFile(configDumpPath); err != nil {
		t.Fatalf("write config dump xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:              ownerDoc,
			Path:             ownerPath,
			RelPath:          "Catalogs/Тест.xml",
			FileName:         "Тест.xml",
			Metadata:         true,
			TopLevelMetadata: true,
			OwnerKey:         "Catalog.Тест",
			OwnerKind:        "Catalog",
		},
		{
			Doc:       rightsDoc,
			Path:      rightsPath,
			RelPath:   "Roles/ТестРоль/Ext/Rights.xml",
			FileName:  "Rights.xml",
			OwnerKey:  "Role.ТестРоль",
			OwnerKind: "Role",
		},
		{
			Doc:      configDumpDoc,
			Path:     configDumpPath,
			RelPath:  configDumpInfo,
			FileName: configDumpInfo,
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест":  {Belonging: "AdoptedStub"},
		"Role.ТестРоль": {Belonging: "Native"},
	}

	candidates := collectOwnerCommandCandidates(contexts, decisions)
	if _, ok := candidates["Catalog.Тест"]["ТестКоманда"]; !ok {
		t.Fatalf("expected command candidate from rights xml")
	}

	liveRefs := buildLiveCommandReferenceIndex(contexts, decisions, map[string]struct{}{})
	retained := filterRetainedOwnerCommandsByLiveReferences(candidates, liveRefs)
	if len(retained["Catalog.Тест"]) != 0 {
		t.Fatalf("expected rights xml not to retain adopted owner command")
	}

	if _, err := finalizeRetainedOwnerCommands(contexts, buildContextIndexes(contexts), decisions, map[string]struct{}{}, nil, nil, candidates, retained, liveRefs, nil, nil); err != nil {
		t.Fatalf("finalize retained commands: %v", err)
	}

	if commands := ownerDoc.FindElements("//*[local-name()='Catalog']/*[local-name()='ChildObjects']/*[local-name()='Command']"); len(commands) != 0 {
		t.Fatalf("expected rights-only command to be removed from ChildObjects, got %d", len(commands))
	}
	if hasMetadataName(configDumpDoc, "Catalog.Тест.Command.ТестКоманда") {
		t.Fatalf("expected rights-only command metadata to be removed from ConfigDumpInfo")
	}
	if objects := rightsDoc.FindElements("//*[local-name()='object']"); len(objects) != 0 {
		t.Fatalf("expected dangling rights reference to be removed, got %d objects", len(objects))
	}
}

func TestFinalizeRetainedOwnerCommandsStripsNativePreserveMarker(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>НаправленияДеятельности</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
      <ExtendedConfigurationObject>base-id</ExtendedConfigurationObject>
    </Properties>
    <ChildObjects>
      <Command>
        <Properties>
          <Name>ТестКоманда</Name>
        </Properties>
      </Command>
      <Attribute codexPreserveNativeObjectBelonging="true">
        <Properties>
          <Name>упо_ЭлементПлана</Name>
        </Properties>
      </Attribute>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	path := filepath.Join(root, "Catalogs", "НаправленияДеятельности.xml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}
	if err := doc.WriteToFile(path); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:              doc,
			Path:             path,
			RelPath:          "Catalogs/НаправленияДеятельности.xml",
			FileName:         "НаправленияДеятельности.xml",
			Metadata:         true,
			TopLevelMetadata: true,
			OwnerKey:         "Catalog.НаправленияДеятельности",
			OwnerKind:        "Catalog",
		},
	}

	rules := map[string]adoptedStubMetaDataRule{
		"Catalog.НаправленияДеятельности": {
			NativeAttributes: map[string]struct{}{
				"упо_ЭлементПлана": {},
			},
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.НаправленияДеятельности": {Belonging: "AdoptedStub"},
	}

	candidates := map[string]map[string]struct{}{
		"Catalog.НаправленияДеятельности": {
			"ТестКоманда": {},
		},
	}

	if _, err := finalizeRetainedOwnerCommands(contexts, buildContextIndexes(contexts), decisions, map[string]struct{}{}, rules, nil, candidates, nil, buildLiveCommandReferenceIndex(contexts, decisions, map[string]struct{}{}), nil, nil); err != nil {
		t.Fatalf("finalize retained commands: %v", err)
	}

	attr := doc.FindElement("//*[local-name()='Attribute']")
	if attr == nil {
		t.Fatalf("expected retained attribute")
	}
	if got := attr.SelectAttrValue(preserveNativeObjectBelongingAttr, ""); got != "" {
		t.Fatalf("expected preserve marker to be stripped, got %q", got)
	}
}

func TestFinalizeRetainedOwnerCommandsStripsNativePreserveMarkerWithoutRetainedDiff(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>НаправленияДеятельности</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
      <ExtendedConfigurationObject>base-id</ExtendedConfigurationObject>
    </Properties>
    <ChildObjects>
      <Attribute codexPreserveNativeObjectBelonging="true">
        <Properties>
          <Name>упо_ЭлементПлана</Name>
        </Properties>
      </Attribute>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	path := filepath.Join(root, "Catalogs", "НаправленияДеятельности.xml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}
	if err := doc.WriteToFile(path); err != nil {
		t.Fatalf("write xml: %v", err)
	}

	contexts := []*FileProcessingContext{
		{
			Doc:              doc,
			Path:             path,
			RelPath:          "Catalogs/НаправленияДеятельности.xml",
			FileName:         "НаправленияДеятельности.xml",
			Metadata:         true,
			TopLevelMetadata: true,
			OwnerKey:         "Catalog.НаправленияДеятельности",
			OwnerKind:        "Catalog",
		},
	}
	rules := map[string]adoptedStubMetaDataRule{
		"Catalog.НаправленияДеятельности": {
			NativeAttributes: map[string]struct{}{
				"упо_ЭлементПлана": {},
			},
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.НаправленияДеятельности": {Belonging: "AdoptedStub"},
	}

	retained := map[string]map[string]struct{}{
		"Catalog.НаправленияДеятельности": {},
	}

	stats, err := finalizeRetainedOwnerCommands(contexts, buildContextIndexes(contexts), decisions, map[string]struct{}{}, rules, nil, retained, retained, buildLiveCommandReferenceIndex(contexts, decisions, map[string]struct{}{}), nil, nil)
	if err != nil {
		t.Fatalf("finalize retained commands: %v", err)
	}
	if stats.WrittenFiles == 0 {
		t.Fatalf("expected preserve marker cleanup to write owner xml even without retained diff")
	}

	attr := doc.FindElement("//*[local-name()='Attribute']")
	if attr == nil {
		t.Fatalf("expected retained attribute")
	}
	if got := attr.SelectAttrValue(preserveNativeObjectBelongingAttr, ""); got != "" {
		t.Fatalf("expected preserve marker to be stripped, got %q", got)
	}
}

func TestCleanupFunctionalOptionsParameterUseNativeChildRefsRemovesPrefixNativeChildRefWhenTopLevelExists(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<FunctionalOptionsParameter xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <Properties>
    <Use>
      <xr:Item xsi:type="xr:MDObjectRef" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">Catalog.упо_Тест</xr:Item>
      <xr:Item xsi:type="xr:MDObjectRef" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">Catalog.упо_Тест.Attribute.Реквизит</xr:Item>
    </Use>
  </Properties>
</FunctionalOptionsParameter>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='Properties']")
	if properties == nil {
		t.Fatalf("expected properties")
	}

	changed := cleanupFunctionalOptionsParameterUseNativeChildRefs(
		properties,
		map[string]objectDecision{
			"Catalog.упо_Тест": {Belonging: "Native"},
		},
	)
	if !changed {
		t.Fatalf("expected cleanup to report change")
	}
	if got := functionalOptionsParameterUseRefs(properties); len(got) != 1 || got[0] != "Catalog.упо_Тест" {
		t.Fatalf("unexpected Use refs after cleanup: %#v", got)
	}
}

func TestCleanupFunctionalOptionsParameterUseNativeChildRefsRemovesIncludedNativeChildRefWithoutPrefix(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<FunctionalOptionsParameter xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <Properties>
    <Use>
      <xr:Item xsi:type="xr:MDObjectRef" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">Catalog.Тест</xr:Item>
      <xr:Item xsi:type="xr:MDObjectRef" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">Catalog.Тест.Attribute.Реквизит</xr:Item>
    </Use>
  </Properties>
</FunctionalOptionsParameter>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='Properties']")
	if properties == nil {
		t.Fatalf("expected properties")
	}

	changed := cleanupFunctionalOptionsParameterUseNativeChildRefs(
		properties,
		map[string]objectDecision{
			"Catalog.Тест": {Belonging: "Native"},
		},
	)
	if !changed {
		t.Fatalf("expected cleanup to report change")
	}
	if got := functionalOptionsParameterUseRefs(properties); len(got) != 1 || got[0] != "Catalog.Тест" {
		t.Fatalf("unexpected Use refs after cleanup: %#v", got)
	}
}

func TestCleanupFunctionalOptionsParameterUseNativeChildRefsKeepsChildRefWithoutTopLevelOwner(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<FunctionalOptionsParameter xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <Properties>
    <Use>
      <xr:Item xsi:type="xr:MDObjectRef" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">Catalog.Тест.Attribute.Реквизит</xr:Item>
    </Use>
  </Properties>
</FunctionalOptionsParameter>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='Properties']")
	if properties == nil {
		t.Fatalf("expected properties")
	}

	changed := cleanupFunctionalOptionsParameterUseNativeChildRefs(
		properties,
		map[string]objectDecision{
			"Catalog.Тест": {Belonging: "Native"},
		},
	)
	if changed {
		t.Fatalf("expected child ref without top-level owner to stay intact")
	}
	if got := functionalOptionsParameterUseRefs(properties); len(got) != 1 || got[0] != "Catalog.Тест.Attribute.Реквизит" {
		t.Fatalf("unexpected Use refs after cleanup: %#v", got)
	}
}

func TestCleanupFunctionalOptionsParameterUseNativeChildRefsKeepsNonNativeOwnerChildRef(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<FunctionalOptionsParameter xmlns="http://v8.1c.ru/8.3/MDClasses" xmlns:xr="http://v8.1c.ru/8.3/xcf/readable">
  <Properties>
    <Use>
      <xr:Item xsi:type="xr:MDObjectRef" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">Catalog.Тест</xr:Item>
      <xr:Item xsi:type="xr:MDObjectRef" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">Catalog.Тест.Attribute.Реквизит</xr:Item>
    </Use>
  </Properties>
</FunctionalOptionsParameter>`); err != nil {
		t.Fatalf("read xml: %v", err)
	}

	properties := doc.FindElement("//*[local-name()='Properties']")
	if properties == nil {
		t.Fatalf("expected properties")
	}

	changed := cleanupFunctionalOptionsParameterUseNativeChildRefs(
		properties,
		map[string]objectDecision{
			"Catalog.Тест": {Belonging: "AdoptedStub"},
		},
	)
	if changed {
		t.Fatalf("expected non-native child ref to stay intact")
	}
	if got := functionalOptionsParameterUseRefs(properties); len(got) != 2 || got[0] != "Catalog.Тест" || got[1] != "Catalog.Тест.Attribute.Реквизит" {
		t.Fatalf("unexpected Use refs after cleanup: %#v", got)
	}
}

func TestCollectSearchResultPlaceRequestsReadsBOMAndFiltersGroups(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	templateDir := filepath.Join(dir, "CommonTemplates", "упо_SearchResult", "Ext")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}

	content := "\xEF\xBB\xBF{\n" +
		`  "Справочники": {` + "\n" +
		`    "ДоговорыКонтрагентов": {` + "\n" +
		`      "МодульФормыФормаВыбора": {` + "\n" +
		`        "EPM": 1,` + "\n" +
		`        "ЧистыйPM": 2` + "\n" +
		`      }` + "\n" +
		`    }` + "\n" +
		`  }` + "\n" +
		`}`
	if err := os.WriteFile(filepath.Join(templateDir, "Template.txt"), []byte(content), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	requests, err := collectSearchResultPlaceRequests(dir, map[string][]string{
		"EPM": {"//{EPM}"},
	})
	if err != nil {
		t.Fatalf("collect search result requests: %v", err)
	}

	places := requests["Catalog.ДоговорыКонтрагентов"]
	if len(places) != 1 {
		t.Fatalf("expected one place request, got %#v", requests)
	}
	if places[0].Place != "МодульФормыФормаВыбора" {
		t.Fatalf("unexpected place %q", places[0].Place)
	}
	if len(places[0].Groups) != 1 || places[0].Groups[0] != "EPM" {
		t.Fatalf("expected only configured EPM group, got %#v", places[0].Groups)
	}
}

func TestBuildSearchResultModuleContentPreservesDirectiveOrderAndComments(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modulePath := filepath.Join(dir, "Module.bsl")
	source := strings.Join([]string{
		"//{EPM}",
		"// комментарий в начале",
		"",
		"&НаКлиенте",
		"Процедура Команда()",
		"\t//{EPM}",
		"КонецПроцедуры",
		"",
		"Функция Проверка(Знач Парам)",
		"\t// {EPM}",
		"\tВозврат Истина;",
		"КонецФункции",
		"",
	}, "\n")
	if err := os.WriteFile(modulePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	content, err := buildSearchResultModuleContent(modulePath, []string{"EPM"}, map[string][]string{
		"EPM": {"//{EPM}", "// {EPM}"},
	}, "упо_", true, "")
	if err != nil {
		t.Fatalf("build search result module content: %v", err)
	}

	if !strings.Contains(content, "//{EPM}\n// комментарий в начале") {
		t.Fatalf("expected top-level comment block to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "&НаКлиенте\n&После(\"Команда\")\nПроцедура упо_Команда()") {
		t.Fatalf("expected procedure directive order to be preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "&ИзменениеИКонтроль(\"Проверка\")\nФункция упо_Проверка(Знач Парам)") {
		t.Fatalf("expected function interceptor to be added, got:\n%s", content)
	}
	if !strings.Contains(content, "\t//{EPM}") || !strings.Contains(content, "\t// {EPM}") {
		t.Fatalf("expected marker comments inside methods to remain, got:\n%s", content)
	}
}

func TestBuildSearchResultModuleContentStrictMismatchLogsAndFallsBackToSoftTransfer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modulePath := filepath.Join(dir, "Module.bsl")
	diagnosticsPath := filepath.Join(dir, "searchresult-template-errors.log")
	source := strings.Join([]string{
		"Процедура Тест()",
		"\t// {PM.НеМодуль}",
		"\tСообщить(\"ok\");",
		"КонецПроцедуры",
		"",
	}, "\n")
	if err := os.WriteFile(modulePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	content, err := buildSearchResultModuleContent(modulePath, []string{"PM"}, map[string][]string{
		"PM":       {"//{PM}", "// {PM}"},
		"НеМодуль": {"// {PM.НеМодуль}"},
	}, "упо_", true, diagnosticsPath)
	if err != nil {
		t.Fatalf("build search result module content: %v", err)
	}
	if !strings.Contains(content, "&После(\"Тест\")\nПроцедура упо_Тест()") {
		t.Fatalf("expected strict mismatch to fall back to general transfer, got:\n%s", content)
	}

	data, readErr := os.ReadFile(diagnosticsPath)
	if readErr != nil {
		t.Fatalf("read diagnostics: %v", readErr)
	}
	if !strings.Contains(string(data), "найдены только группы [НеМодуль]") {
		t.Fatalf("expected diagnostics file to contain exact mismatch, got: %s", string(data))
	}
}

func TestBuildSearchResultModuleContentTransfersPrefixedMethodsWithoutAfterDirective(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modulePath := filepath.Join(dir, "Module.bsl")
	source := strings.Join([]string{
		"Процедура упо_ПослеЗаписи()",
		"\t//{EPM}",
		"КонецПроцедуры",
		"",
		"Процедура Подключаемый_упо_ПослеЗаписи()",
		"\t//{EPM}",
		"КонецПроцедуры",
		"",
	}, "\n")
	if err := os.WriteFile(modulePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	content, err := buildSearchResultModuleContent(modulePath, []string{"EPM"}, map[string][]string{
		"EPM": {"//{EPM}"},
	}, "упо_", true, "")
	if err != nil {
		t.Fatalf("build search result module content: %v", err)
	}

	if strings.Contains(content, `&После("упо_ПослеЗаписи")`) {
		t.Fatalf("expected direct transfer for prefixed method without &После, got:\n%s", content)
	}
	if strings.Contains(content, `&После("Подключаемый_упо_ПослеЗаписи")`) {
		t.Fatalf("expected direct transfer for prefixed plugin method without &После, got:\n%s", content)
	}
	if !strings.Contains(content, "Процедура упо_ПослеЗаписи()") || !strings.Contains(content, "Процедура Подключаемый_упо_ПослеЗаписи()") {
		t.Fatalf("expected original prefixed method headers to be preserved, got:\n%s", content)
	}
}

func TestAddSearchResultCommandOverlaySkipsCommandWithoutTransferredCode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	commandModuleDir := filepath.Join(root, "DataProcessors", "ТестОбработка", "Commands", "КомандаБезДоработок", "Ext")
	if err := os.MkdirAll(commandModuleDir, 0o755); err != nil {
		t.Fatalf("mkdir command module dir: %v", err)
	}
	modulePath := filepath.Join(commandModuleDir, "CommandModule.bsl")
	if err := os.WriteFile(modulePath, []byte("Процедура БезМаркеров()\nКонецПроцедуры\n"), 0o644); err != nil {
		t.Fatalf("write command module: %v", err)
	}

	state := newSearchResultState()
	topCtx := &FileProcessingContext{
		OwnerKey:  "DataProcessor.ТестОбработка",
		OwnerKind: "DataProcessor",
	}

	if err := addSearchResultCommandOverlay(
		state,
		filepath.Join(root, "DataProcessors", "ТестОбработка"),
		topCtx,
		"КомандаБезДоработок",
		[]string{"EPM"},
		map[string][]string{"EPM": {"//{EPM}"}},
		"упо_",
		true,
		"",
	); err != nil {
		t.Fatalf("add search result command overlay: %v", err)
	}

	if _, ok := state.ModuleWrites[modulePath]; ok {
		t.Fatalf("did not expect command module write without transferred code")
	}
	if _, ok := state.ObjectOverlays[topCtx.OwnerKey].PreserveCommands["КомандаБезДоработок"]; ok {
		t.Fatalf("did not expect command without transferred code to be preserved")
	}
	if _, ok := state.PreservedConfigDumpInfo[topCtx.OwnerKey+".Command.КомандаБезДоработок"]; ok {
		t.Fatalf("did not expect command metadata to be preserved without transferred code")
	}
}

func TestCollectSearchResultStatePromotesExcludedObjectAndPreservesFormPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "searchingTemplateText.json"), []byte(`{"EPM":["//{EPM}"]}`), 0o644); err != nil {
		t.Fatalf("write markers: %v", err)
	}

	templateDir := filepath.Join(root, "CommonTemplates", "упо_SearchResult", "Ext")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	template := `{
  "Справочники": {
    "Тест": {
      "МодульФормыФормаВыбора": {
        "EPM": 1,
        "ЧистыйPM": 3
      }
    }
  }
}`
	if err := os.WriteFile(filepath.Join(templateDir, "Template.txt"), []byte(template), 0o644); err != nil {
		t.Fatalf("write search result template: %v", err)
	}

	objectDir := filepath.Join(root, "Catalogs", "Тест")
	formDir := filepath.Join(objectDir, "Forms", "ФормаВыбора", "Ext", "Form")
	if err := os.MkdirAll(formDir, 0o755); err != nil {
		t.Fatalf("mkdir form dir: %v", err)
	}

	topLevelXML := `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog>
    <Properties>
      <Name>Тест</Name>
    </Properties>
    <ChildObjects>
      <Form>ФормаВыбора</Form>
    </ChildObjects>
  </Catalog>
</MetaDataObject>`
	if err := os.WriteFile(filepath.Join(root, "Catalogs", "Тест.xml"), []byte(topLevelXML), 0o644); err != nil {
		t.Fatalf("write top-level xml: %v", err)
	}

	formXML := `<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Form>
    <Properties>
      <Name>ФормаВыбора</Name>
    </Properties>
  </Form>
</MetaDataObject>`
	if err := os.WriteFile(filepath.Join(objectDir, "Forms", "ФормаВыбора.xml"), []byte(formXML), 0o644); err != nil {
		t.Fatalf("write form xml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(objectDir, "Forms", "ФормаВыбора", "Ext", "Form.xml"), []byte(formXML), 0o644); err != nil {
		t.Fatalf("write ext form xml: %v", err)
	}

	formModule := strings.Join([]string{
		"//{EPM}",
		"Процедура ОткрытьФорму()",
		"\t//{EPM}",
		"КонецПроцедуры",
		"",
	}, "\n")
	modulePath := filepath.Join(formDir, "Module.bsl")
	if err := os.WriteFile(modulePath, []byte(formModule), 0o644); err != nil {
		t.Fatalf("write form module: %v", err)
	}

	contexts, err := loadXMLContexts(root)
	if err != nil {
		t.Fatalf("load xml contexts: %v", err)
	}

	cfg := &config.Configuration{
		Prefix:               "упо_",
		ConfigPath:           filepath.Join(configDir, "config.json"),
		AdditionalProcessing: config.AdditionalProcessing{UseSearchResult: true},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Excluded: true},
	}

	state, err := collectSearchResultState(cfg, root, contexts, decisions, nil, nil, nil)
	if err != nil {
		t.Fatalf("collect search result state: %v", err)
	}

	decision := decisions["Catalog.Тест"]
	if decision.Excluded || decision.Belonging != "AdoptedStub" || decision.Truncated {
		t.Fatalf("expected SearchResult to promote object to AdoptedStub without truncation, got %#v", decision)
	}

	overlay := state.ObjectOverlays["Catalog.Тест"]
	if _, ok := overlay.PreserveForms["ФормаВыбора"]; !ok {
		t.Fatalf("expected overlay to preserve form, got %#v", overlay)
	}

	formObjectPath := filepath.Join(objectDir, "Forms", "ФормаВыбора.xml")
	formMetadataPath := filepath.Join(objectDir, "Forms", "ФормаВыбора", "Ext", "Form.xml")
	for _, path := range []string{formObjectPath, formMetadataPath, modulePath} {
		if _, ok := state.PreservedPaths[path]; !ok {
			t.Fatalf("expected preserved path %s, got %#v", path, state.PreservedPaths)
		}
	}

	for _, name := range []string{
		"Catalog.Тест.Form.ФормаВыбора",
		"Catalog.Тест.Form.ФормаВыбора.Form",
	} {
		if _, ok := state.PreservedConfigDumpInfo[name]; !ok {
			t.Fatalf("expected preserved ConfigDumpInfo name %s, got %#v", name, state.PreservedConfigDumpInfo)
		}
	}

	write, ok := state.ModuleWrites[modulePath]
	if !ok {
		t.Fatalf("expected generated module write for %s", modulePath)
	}
	if !strings.Contains(write.Content, `&После("ОткрытьФорму")`) || !strings.Contains(write.Content, "Процедура упо_ОткрытьФорму()") {
		t.Fatalf("unexpected generated module content:\n%s", write.Content)
	}
}

func TestCollectFormDynamicListContractsIncludesSearchResultPreservedForms(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<Form xmlns="http://v8.1c.ru/8.3/xcf/logform" xmlns:v8="http://v8.1c.ru/8.1/data/core">
  <Attributes>
    <Attribute name="Список">
      <Type>
        <v8:Type>cfg:DynamicList</v8:Type>
      </Type>
      <MainTable>Catalog.Валюты</MainTable>
    </Attribute>
  </Attributes>
  <ChildItems>
    <Table name="Список">
      <ChildItems>
        <InputField name="Наименование">
          <Settings>
            <DataPath>Список.Наименование</DataPath>
          </Settings>
        </InputField>
      </ChildItems>
    </Table>
  </ChildItems>
</Form>`); err != nil {
		t.Fatalf("read form xml: %v", err)
	}

	path := filepath.Join(t.TempDir(), "Catalogs", "Тест", "Forms", "ФормаСписка", "Ext", "Form.xml")
	contexts := []*FileProcessingContext{{
		Doc:      doc,
		Path:     path,
		RelPath:  filepath.ToSlash(path),
		OwnerKey: "Catalog.Тест",
	}}
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Belonging: "AdoptedStub", SearchResultCode: true},
	}
	state := &searchResultState{
		PreservedPaths: map[string]struct{}{path: {}},
	}

	contracts := collectFormDynamicListContracts(contexts, decisions, state)
	contract, ok := contracts["Catalog.Валюты"]
	if !ok {
		t.Fatalf("expected SearchResult preserved form to participate in dynamic list contracts")
	}
	if _, ok := contract.RequiredFields["Наименование"]; !ok {
		t.Fatalf("expected dynamic list contract to retain required field, got %#v", contract.RequiredFields)
	}
}

func TestCleanupConfigDumpInfoNonNativeChildrenKeepsSearchResultPreservedMetadata(t *testing.T) {
	t.Parallel()

	configDump := etree.NewDocument()
	if err := configDump.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<ConfigDumpInfo>
  <Metadata name="Catalog.Тест" id="1">
    <Metadata name="Catalog.Тест.Form.ФормаВыбора" id="2"/>
    <Metadata name="Catalog.Тест.Form.ФормаВыбора.Form" id="3"/>
    <Metadata name="Catalog.Тест.Command.Открыть" id="4"/>
    <Metadata name="Catalog.Тест.Command.Открыть.CommandModule" id="5"/>
  </Metadata>
</ConfigDumpInfo>`); err != nil {
		t.Fatalf("read config dump: %v", err)
	}

	changed := cleanupConfigDumpInfoNonNativeChildren(
		configDump,
		nil,
		map[string]objectDecision{
			"Catalog.Тест": {Belonging: "AdoptedStub", SearchResultCode: true},
		},
		map[string]struct{}{
			"Catalog.Тест.Form.ФормаВыбора":              {},
			"Catalog.Тест.Form.ФормаВыбора.Form":         {},
			"Catalog.Тест.Command.Открыть":               {},
			"Catalog.Тест.Command.Открыть.CommandModule": {},
		},
	)
	if changed {
		t.Fatalf("expected preserved SearchResult metadata to survive ConfigDumpInfo cleanup")
	}

	for _, name := range []string{
		"Catalog.Тест.Form.ФормаВыбора",
		"Catalog.Тест.Form.ФормаВыбора.Form",
		"Catalog.Тест.Command.Открыть",
		"Catalog.Тест.Command.Открыть.CommandModule",
	} {
		if !hasMetadataName(configDump, name) {
			t.Fatalf("expected preserved metadata name %s to remain", name)
		}
	}
}

func TestWriteSearchResultModuleFilesSkipsNativeOwners(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	modulePath := filepath.Join(dir, "Module.bsl")
	if err := os.WriteFile(modulePath, []byte("старый"), 0o644); err != nil {
		t.Fatalf("write initial module: %v", err)
	}

	state := &searchResultState{
		ModuleWrites: map[string]searchResultModuleWrite{
			modulePath: {
				OwnerKey: "Catalog.Тест",
				Path:     modulePath,
				Content:  "новый",
			},
		},
	}
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Belonging: "Native"},
	}

	if err := writeSearchResultModuleFiles(state, decisions); err != nil {
		t.Fatalf("write search result module files: %v", err)
	}

	data, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatalf("read module after write: %v", err)
	}
	if string(data) != "старый" {
		t.Fatalf("expected native owner module to stay untouched, got %q", string(data))
	}
}

func functionalOptionsParameterUseRefs(properties *etree.Element) []string {
	if properties == nil {
		return nil
	}

	use := properties.FindElement("./Use")
	if use == nil {
		return nil
	}

	refs := make([]string, 0, len(use.ChildElements()))
	for _, child := range use.ChildElements() {
		refs = append(refs, strings.TrimSpace(child.Text()))
	}
	return refs
}

func TestValidateSearchResultAdoptedObjectsRequiresAdoptedTopLevelXML(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject>
  <Catalog>
    <Properties>
      <Name>Тест</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
  </Catalog>
</MetaDataObject>`); err != nil {
		t.Fatalf("read doc: %v", err)
	}

	contexts := []*FileProcessingContext{{
		Doc:              doc,
		Path:             filepath.Join(t.TempDir(), "Catalogs", "Тест.xml"),
		RelPath:          "Catalogs/Тест.xml",
		FileName:         "Тест.xml",
		Metadata:         true,
		TopLevelMetadata: true,
		Properties:       findProperties(doc),
		OwnerKind:        "Catalog",
		OwnerName:        "Тест",
		OwnerKey:         "Catalog.Тест",
	}}
	indexes := buildContextIndexes(contexts)
	decisions := map[string]objectDecision{
		"Catalog.Тест": {Belonging: "AdoptedStub", SearchResultCode: true},
	}
	state := &searchResultState{
		ExpectedAdoptedObjects: map[string]struct{}{
			"Catalog.Тест": {},
		},
	}

	if err := validateSearchResultAdoptedObjects(indexes, contexts, decisions, map[string]struct{}{}, state); err != nil {
		t.Fatalf("validate search result adopted objects: %v", err)
	}

	if err := validateSearchResultAdoptedObjects(indexes, contexts, map[string]objectDecision{
		"Catalog.Тест": {Belonging: "Native", SearchResultCode: true},
	}, map[string]struct{}{}, state); err == nil {
		t.Fatalf("expected native decision to fail validation")
	}
}

func mustReadTestXML(t *testing.T, content string) *etree.Document {
	t.Helper()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(content); err != nil {
		t.Fatalf("read test xml: %v", err)
	}

	return doc
}

func formDocHasDataPath(doc *etree.Document, target string) bool {
	if doc == nil {
		return false
	}

	for _, dataPath := range doc.FindElements("//*[local-name()='DataPath']") {
		if strings.TrimSpace(dataPath.Text()) == target {
			return true
		}
	}

	return false
}

func dynamicListFieldDeclared(doc *etree.Document, attrName, target string) bool {
	if doc == nil || doc.Root() == nil {
		return false
	}

	for _, attr := range doc.FindElements("//*[local-name()='Attribute']") {
		if strings.TrimSpace(attr.SelectAttrValue("name", "")) != attrName {
			continue
		}
		for _, field := range attr.FindElements(".//*[local-name()='Field']") {
			if len(field.ChildElements()) == 0 {
				if strings.TrimSpace(field.Text()) == target {
					return true
				}
				continue
			}
			for _, child := range field.ChildElements() {
				tag := localName(child.Tag)
				if !strings.EqualFold(tag, "dataPath") && !strings.EqualFold(tag, "field") {
					continue
				}
				if strings.TrimSpace(child.Text()) == target {
					return true
				}
			}
		}
	}

	return false
}

func formDocContainsText(doc *etree.Document, target string) bool {
	if doc == nil || doc.Root() == nil {
		return false
	}

	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}

	for _, el := range doc.FindElements("//*") {
		if strings.TrimSpace(el.Text()) == target {
			return true
		}
	}

	return false
}

func hasMetadataName(doc *etree.Document, name string) bool {
	if doc == nil {
		return false
	}
	for _, metadata := range doc.FindElements("//*[local-name()='Metadata']") {
		if strings.TrimSpace(metadata.SelectAttrValue("name", "")) == name {
			return true
		}
	}
	return false
}

func metadataEntryID(doc *etree.Document, name string) string {
	if doc == nil {
		return ""
	}
	for _, metadata := range doc.FindElements("//*[local-name()='Metadata']") {
		if strings.TrimSpace(metadata.SelectAttrValue("name", "")) != name {
			continue
		}
		return strings.TrimSpace(metadata.SelectAttrValue("id", ""))
	}
	return ""
}

func hasConfigurationChildObject(doc *etree.Document, kind, name string) bool {
	if doc == nil || doc.Root() == nil {
		return false
	}
	childObjects := doc.Root().FindElement(".//ChildObjects")
	if childObjects == nil {
		return false
	}
	for _, child := range childObjects.ChildElements() {
		if strings.EqualFold(localName(child.Tag), kind) && strings.TrimSpace(child.Text()) == name {
			return true
		}
	}
	return false
}

func textOrEmpty(el *etree.Element) string {
	if el == nil {
		return ""
	}
	return strings.TrimSpace(el.Text())
}
