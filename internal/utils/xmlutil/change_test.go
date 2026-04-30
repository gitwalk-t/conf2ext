package xmlutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
	"github.com/firstBitSportivnaya/files-converter/internal/config"
)

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

func TestApplyAdoptedStubMetaDataRulesDoesNotRestoreExcludedObject(t *testing.T) {
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
	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected excluded metadata-file object to stay excluded, got %#v", decision)
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

	if !normalizeAdoptedStubMetaDataComposition(doc, "Catalog", rule) {
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

func TestNormalizeAdoptedStubMetaDataCompositionKeepsRetainedTabularSectionChildrenNative(t *testing.T) {
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

	if !normalizeAdoptedStubMetaDataComposition(doc, "Document", rule) {
		t.Fatalf("expected adopted metadata composition to change tabular section child attributes")
	}

	sectionProps := doc.FindElement("//*[local-name()='TabularSection']/*[local-name()='Properties']")
	if sectionProps == nil {
		t.Fatalf("expected retained tabular section properties")
	}
	if got := textOf(sectionProps, "ObjectBelonging"); got != "" {
		t.Fatalf("expected retained tabular section to be native, got ObjectBelonging=%q", got)
	}
	if got := textOf(sectionProps, "ExtendedConfigurationObject"); got != "" {
		t.Fatalf("expected retained native tabular section to drop ExtendedConfigurationObject, got %q", got)
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
}

func TestEnsureAdoptedExtendedConfigurationObjectsPreservesRetainedMetaDataFileAttributeNative(t *testing.T) {
	t.Parallel()

	doc := etree.NewDocument()
	if err := doc.ReadFromString(`<?xml version="1.0" encoding="UTF-8"?>
<MetaDataObject xmlns="http://v8.1c.ru/8.3/MDClasses">
  <Catalog uuid="catalog-guid">
    <Properties>
      <Name>НаправленияДеятельности</Name>
      <ObjectBelonging>Adopted</ObjectBelonging>
    </Properties>
    <ChildObjects>
      <Attribute uuid="attr-guid">
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

	if !normalizeAdoptedStubMetaDataComposition(doc, "Catalog", rule) {
		t.Fatalf("expected adopted metadata composition to change child attribute")
	}

	original := collectMetadataOriginalUUIDs(doc)
	if !ensureAdoptedExtendedConfigurationObjects(doc, original) {
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

	if !cleanupConfigDumpInfoNonNativeChildren(configDump, contexts, decisions) {
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

	if !cleanupConfigDumpInfoNonNativeChildren(configDump, contexts, decisions) {
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

func TestDecideObjectExcludedOverridesPrimaryNative(t *testing.T) {
	t.Parallel()

	ctx := &FileProcessingContext{
		OwnerKey:  "DataProcessor.упо_ФормированиеЗаказовЭлементаПлана",
		OwnerKind: "DataProcessor",
		OwnerName: "упо_ФормированиеЗаказовЭлементаПлана",
	}

	decision := decideObject(
		ctx,
		&config.Configuration{},
		map[string]struct{}{ctx.OwnerKey: {}},
		map[string]struct{}{ctx.OwnerKey: {}},
		nil,
		nil,
	)

	if !decision.Excluded || decision.Belonging != "" {
		t.Fatalf("expected excluded decision to override primary native, got %#v", decision)
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
		map[string]struct{}{target.OwnerKey: {}},
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

func TestNormalizeChartOfCharacteristicTypesPredefinedKeepsItemQualifiers(t *testing.T) {
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
		t.Fatalf("expected valid predefined qualifiers to stay untouched")
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

func textOrEmpty(el *etree.Element) string {
	if el == nil {
		return ""
	}
	return strings.TrimSpace(el.Text())
}
