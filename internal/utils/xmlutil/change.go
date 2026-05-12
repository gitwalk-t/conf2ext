package xmlutils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/beevik/etree"
	"github.com/firstBitSportivnaya/files-converter/internal/config"
)

const (
	dirCommonModules                  = "CommonModules"
	mainFile                          = "Configuration.xml"
	configDumpInfo                    = "ConfigDumpInfo.xml"
	preserveNativeObjectBelongingAttr = "codexPreserveNativeObjectBelonging"
)

var (
	guidPattern              = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	metadataReferencePattern = regexp.MustCompile(`(?:[A-Za-z_][A-Za-z0-9_.-]*:)?(AccountingRegister|AccumulationRegister|BusinessProcess|CalculationRegister|Catalog|ChartOfAccounts|ChartOfCalculationTypes|ChartOfCharacteristicTypes|CommandGroup|CommonAttribute|CommonCommand|CommonForm|CommonModule|CommonPicture|CommonTemplate|Constant|DataProcessor|DefinedType|Document|DocumentJournal|Enum|EventSubscription|ExchangePlan|ExternalDataSource|FilterCriterion|FunctionalOption|FunctionalOptionsParameter|HTTPService|InformationRegister|IntegrationService|Interface|Language|Report|Role|ScheduledJob|Sequence|Session|SessionParameter|SettingsStorage|Style|StyleItem|Subsystem|Task|WebService|XDTOPackage)(?:Ref|Object|Selection|List|Manager|ValueManager|RecordSet|TabularSectionRow|TabularSection)?\.([^\s<>"':/\\]+)`)
	styleReferencePattern    = regexp.MustCompile(`(?:^|[^[:alnum:]_])style:([^\s<>"':/\\]+)`)

	adoptedSubsystemRoots = map[string]struct{}{
		"СтандартныеПодсистемы":        {},
		"Администрирование":            {},
		"ПодключаемыеОтчетыИОбработки": {},
	}

	guidFieldNames = map[string]struct{}{
		"id":       {},
		"objectid": {},
		"typeid":   {},
		"uuid":     {},
		"valueid":  {},
	}

	metadataKinds = map[string]string{
		"AccountingRegisters":         "AccountingRegister",
		"AccumulationRegisters":       "AccumulationRegister",
		"BusinessProcesses":           "BusinessProcess",
		"CalculationRegisters":        "CalculationRegister",
		"Catalogs":                    "Catalog",
		"ChartsOfAccounts":            "ChartOfAccounts",
		"ChartsOfCalculationTypes":    "ChartOfCalculationTypes",
		"ChartsOfCharacteristicTypes": "ChartOfCharacteristicTypes",
		"CommonCommands":              "CommonCommand",
		"CommonForms":                 "CommonForm",
		"CommonAttributes":            "CommonAttribute",
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

	metadataKindAliases = map[string]string{
		"обработка":                   "DataProcessor",
		"общийреквизит":               "CommonAttribute",
		"chartsofcharacteristictypes": "ChartOfCharacteristicTypes",
		"подписка":                    "EventSubscription",
		"событиеподписки":             "EventSubscription",
	}

	metaDataFileKindAliases = map[string]string{
		"БизнесПроцесс":          "BusinessProcess",
		"Документ":               "Document",
		"Константа":              "Constant",
		"КритерийОтбора":         "FilterCriterion",
		"Обработка":              "DataProcessor",
		"Перечисление":           "Enum",
		"ПланВидовХарактеристик": "ChartOfCharacteristicTypes",
		"ПланОбмена":             "ExchangePlan",
		"РегистрБухгалтерии":     "AccountingRegister",
		"РегистрНакопления":      "AccumulationRegister",
		"РегистрСведений":        "InformationRegister",
		"Справочник":             "Catalog",
		"ПодпискаНаСобытие":      "EventSubscription",
	}

	configurationChildObjectKinds = map[string]string{
		"AccountingRegister":         "AccountingRegister",
		"AccumulationRegister":       "AccumulationRegister",
		"BusinessProcess":            "BusinessProcess",
		"CalculationRegister":        "CalculationRegister",
		"Catalog":                    "Catalog",
		"ChartOfAccounts":            "ChartOfAccounts",
		"ChartOfCalculationTypes":    "ChartOfCalculationTypes",
		"ChartOfCharacteristicTypes": "ChartOfCharacteristicTypes",
		"CommandGroup":               "CommandGroup",
		"CommonAttribute":            "CommonAttribute",
		"CommonCommand":              "CommonCommand",
		"CommonForm":                 "CommonForm",
		"CommonModule":               "CommonModule",
		"CommonPicture":              "CommonPicture",
		"CommonTemplate":             "CommonTemplate",
		"Constant":                   "Constant",
		"DataProcessor":              "DataProcessor",
		"DefinedType":                "DefinedType",
		"Document":                   "Document",
		"DocumentJournal":            "DocumentJournal",
		"Enum":                       "Enum",
		"EventSubscription":          "EventSubscription",
		"ExchangePlan":               "ExchangePlan",
		"ExternalDataSource":         "ExternalDataSource",
		"FilterCriterion":            "FilterCriterion",
		"FunctionalOption":           "FunctionalOption",
		"FunctionalOptionsParameter": "FunctionalOptionsParameter",
		"HTTPService":                "HTTPService",
		"InformationRegister":        "InformationRegister",
		"IntegrationService":         "IntegrationService",
		"Interface":                  "Interface",
		"Language":                   "Language",
		"Report":                     "Report",
		"Role":                       "Role",
		"ScheduledJob":               "ScheduledJob",
		"Sequence":                   "Sequence",
		"Session":                    "Session",
		"SessionParameter":           "SessionParameter",
		"SettingsStorage":            "SettingsStorage",
		"Style":                      "Style",
		"StyleItem":                  "StyleItem",
		"Subsystem":                  "Subsystem",
		"Task":                       "Task",
		"WebService":                 "WebService",
		"XDTOPackage":                "XDTOPackage",
	}

	configurationRootPromotedKinds = map[string]struct{}{
		"CommandGroup":     {},
		"CommonModule":     {},
		"Enum":             {},
		"SessionParameter": {},
		"Style":            {},
		"StyleItem":        {},
	}
)

type FileProcessingContext struct {
	Doc              *etree.Document
	Path             string
	RelPath          string
	FileName         string
	Metadata         bool
	TopLevelMetadata bool
	Properties       *etree.Element
	OwnerKey         string
	OwnerKind        string
	OwnerName        string
}

type objectDecision struct {
	Belonging string
	Excluded  bool
	Truncated bool
}

type subsystemState struct {
	Key   string
	Name  string
	Chain []string
}

type formDynamicListContract struct {
	RequiredFields   map[string]struct{}
	QueryAliases     map[string]struct{}
	RequiredCommands map[string]struct{}
}

type adoptedStubMetaDataRule struct {
	NativeAttributes      map[string]struct{}
	NativeTabularSections map[string]map[string]struct{}
}

type changeFilesState struct {
	contexts                 []*FileProcessingContext
	indexes                  *contextIndexes
	decisions                map[string]objectDecision
	formDynamicListContracts map[string]formDynamicListContract
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule
	excludedPaths            map[string]struct{}
}

type contextIndexes struct {
	byOwnerKey map[string][]*FileProcessingContext
	byRelPath  map[string]*FileProcessingContext
	byPath     map[string]*FileProcessingContext
	byFileName map[string][]*FileProcessingContext
}

type nodeMark struct {
	hasExcludedRef bool
	computed       bool
}

type excludedMetadataTraversalState struct {
	prefixSet    map[string]struct{}
	subtreeCache map[*etree.Element]nodeMark
	valueCache   map[string]bool
}

func ChangeFiles(cfg *config.Configuration, dir string) error {
	log.Printf("xml change started: dir=%s", dir)
	filesOps := make(map[string][]*config.ElementOperation, len(cfg.XMLFiles))
	for _, file := range cfg.XMLFiles {
		filesOps[file.FileName] = file.ElementOperations
	}

	loadContextsStartedAt := time.Now()
	contexts, err := loadXMLContexts(dir)
	if err != nil {
		return err
	}
	indexes := buildContextIndexes(contexts)
	logXMLStepCompleted("loadXMLContexts", loadContextsStartedAt, fmt.Sprintf("contexts=%d", len(contexts)))

	fillDumpInfo(contexts)

	log.Printf("xml step: build object sets")
	buildObjectSetsStartedAt := time.Now()
	includedNativeObjects := collectConfiguredNativeObjects(contexts, cfg.IncludedNativeObjects)
	includedAdoptedStubObjects := collectConfiguredAdoptedStubObjects(contexts, cfg)
	adoptedStubMetaDataRules := collectAdoptedStubMetaDataRules(cfg, dir)
	for key := range adoptedStubMetaDataRules {
		includedAdoptedStubObjects[key] = struct{}{}
	}
	forbiddenAdoptedStubObjects := collectConfiguredForbiddenStubObjects(contexts, cfg.ForbiddenAdoptedStubObjects)
	primaryNativeObjects := collectPrimaryNativeObjects(contexts, cfg.NativePrefixes, includedNativeObjects)
	excludedSubsystemObjects := collectExcludedSubsystemObjects(contexts, cfg.ExcludedSubsystems, cfg.NativePrefixes)
	configuredExcludedObjects := collectConfiguredExcludedObjects(contexts, cfg.ExcludedObjects)
	excludedObjects := mergeObjectSets(excludedSubsystemObjects, configuredExcludedObjects)
	logXMLStepCompleted("build object sets", buildObjectSetsStartedAt)
	log.Printf("xml step: collect subsystem decisions")
	collectSubsystemDecisionsStartedAt := time.Now()
	subsystemDecisions := collectSubsystemDecisions(contexts, cfg)
	decisions := make(map[string]objectDecision)
	for _, ctx := range contexts {
		if ctx.OwnerKey == "" {
			continue
		}
		if _, exists := decisions[ctx.OwnerKey]; exists {
			continue
		}
		if ctx.OwnerKind == "Subsystem" {
			decision, ok := subsystemDecisions[ctx.OwnerKey]
			if ok {
				decisions[ctx.OwnerKey] = decision
				continue
			}
		}
		decisions[ctx.OwnerKey] = decideObject(ctx, cfg, primaryNativeObjects, excludedObjects, includedAdoptedStubObjects, forbiddenAdoptedStubObjects)
	}
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	logXMLStepCompleted("collect subsystem decisions", collectSubsystemDecisionsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	log.Printf("xml step: promote referenced objects")
	promoteReferencedObjectsStartedAt := time.Now()
	referenceGraph := collectReferenceGraph(contexts, cfg, primaryNativeObjects)
	incomingReferenceGraph := collectIncomingReferenceGraph(referenceGraph)
	adoptedStubExtReferenceGraph := collectAdoptedStubExtReferenceGraph(contexts, decisions)
	formDynamicListContracts := collectFormDynamicListContracts(contexts, decisions)
	promoteReferencedObjectsToAdoptedStubIndexed(contexts, indexes, decisions, cfg, referenceGraph, incomingReferenceGraph, adoptedStubExtReferenceGraph, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects)
	promoteRegisterDocumentOwnersToNativeIndexed(contexts, indexes, decisions, cfg, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, collectRegisterDocumentReferences(contexts))
	applyFormDynamicListContracts(decisions, formDynamicListContracts, forbiddenAdoptedStubObjects)
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	retainedOwnerCommands := collectRetainedOwnerCommands(contexts, decisions)
	logXMLStepCompleted("promote referenced objects", promoteReferencedObjectsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	log.Printf("xml step: collect cleanup sets")
	collectCleanupSetsStartedAt := time.Now()
	excludedRefs := collectExcludedReferences(decisions)
	blockedForbiddenObjectKeys := collectBlockedForbiddenObjectKeys(decisions, forbiddenAdoptedStubObjects)
	blockedForbiddenRefs := collectReferenceMapFromObjectKeys(blockedForbiddenObjectKeys)
	excludedRefs = mergeReferenceMaps(excludedRefs, blockedForbiddenRefs)
	excludedMetadataPrefixes := collectExcludedMetadataPrefixes(excludedRefs)
	truncatedKeys := collectTruncatedKeys(decisions)
	truncatedChildPrefixes := collectTruncatedChildPrefixes(truncatedKeys)
	guidReplacements := collectGUIDReplacements(contexts, decisions)
	// Для DefinedType режем только hard forbidden: мягко исключенные типы
	// должны сохраняться в составе и дотягиваться по RefDrivenInclusion.
	blockedDefinedTypeObjects := blockedForbiddenObjectKeys

	excludedPaths := make(map[string]struct{})
	logXMLStepCompleted("collect cleanup sets", collectCleanupSetsStartedAt)

	log.Printf("xml step: apply object changes")
	applyObjectChangesStartedAt := time.Now()
	lastApplyObjectChangesLogAt := time.Now()
	changedFilesCount := 0
	writtenFilesCount := 0
	for idx, ctx := range contexts {
		if idx%50 == 0 || time.Since(lastApplyObjectChangesLogAt) >= 30*time.Second {
			log.Printf("xml progress: apply object changes %d/%d file=%s", idx, len(contexts), ctx.RelPath)
			lastApplyObjectChangesLogAt = time.Now()
		}
		decision := decisions[ctx.OwnerKey]
		if decision.Excluded {
			excludedPaths[ctx.Path] = struct{}{}
			continue
		}

		if isRootServiceFile(ctx) {
			excludedPaths[ctx.Path] = struct{}{}
			continue
		}

		if ctx.FileName != configDumpInfo && ctx.OwnerKey != "Configuration" && !isTopLevelMetadataFile(ctx) && decision.Belonging != "Native" {
			excludedPaths[ctx.Path] = struct{}{}
			continue
		}

		changed := false

		if ctx.Metadata && ctx.Properties != nil && ctx.OwnerKey != "" && ctx.FileName != configDumpInfo {
			if decision.Belonging != "" {
				setObjectBelonging(ctx.Properties, decision.Belonging)
				changed = true
			}
		}

		if ctx.OwnerKind == "Enum" {
			changed = ensureEnumValueDefaultColor(ctx.Doc) || changed
		}

		if ctx.OwnerKind == "DefinedType" && decision.Belonging == "Native" {
			changed = cleanupDefinedTypeExcludedTypes(ctx.Properties, blockedDefinedTypeObjects) || changed
		}

		if operations, found := filesOps[ctx.FileName]; found && ctx.Properties != nil {
			for _, operation := range operations {
				changed = processElement(ctx.Properties, operation) || changed
			}
		}

		if ctx.OwnerKey == "Configuration" {
			changed = normalizeRootConfiguration(ctx.Properties, cfg) || changed
			changed = normalizeRootConfigurationInternalInfo(ctx.Doc) || changed
			changed = normalizeRootConfigurationChildObjects(ctx.Doc, contexts, decisions) || changed
		}

		if ctx.FileName == configDumpInfo {
			changed = normalizeConfigDumpInfoRootNames(ctx.Doc, config.GetDumpInfo().ConfigName, cfg.Extension) || changed
			changed = cleanupConfigDumpInfoRootServiceEntries(ctx.Doc, cfg.Extension) || changed
			changed = cleanupConfigDumpInfoNonNativeChildren(ctx.Doc, contexts, decisions) || changed
		}

		if ctx.FileName == "CommandInterface.xml" {
			changed = normalizeCommandInterfaceFileIndexed(ctx, contexts, indexes) || changed
			changed = cleanupDanglingCommandInterfaceCommands(ctx.Doc, contexts) || changed
		}

		if ctx.FileName == "MainSectionCommandInterface.xml" {
			changed = cleanupDanglingCommandInterfaceCommands(ctx.Doc, contexts) || changed
		}

		if strings.EqualFold(ctx.FileName, "Rights.xml") {
			changed = cleanupRoleConfigurationRights(ctx.Doc, config.GetDumpInfo().ConfigName) || changed
			changed = cleanupRoleExcludedMetadataRights(ctx.Doc, decisions) || changed
			changed = cleanupRoleDanglingMetadataRights(ctx.Doc, contexts) || changed
		}

		if strings.Contains(filepath.ToSlash(ctx.RelPath), "/Forms/") {
			changed = cleanupNonNativeDynamicListMainTables(ctx.Doc, decisions) || changed
			changed = normalizeManualQueryWithoutMainTable(ctx.Doc) || changed
			changed = cleanupMissingFormConstantsSetReferencesIndexed(ctx.Doc, contexts, indexes, decisions) || changed
			changed = cleanupMissingFormCommonAttributeDynamicListFieldsIndexed(ctx.Doc, contexts, indexes, decisions) || changed
			changed = cleanupMissingFormCommandReferences(ctx.Doc, contexts) || changed
		}

		if ctx.OwnerKey == "Language.Русский" {
			changed = normalizeLanguageObject(ctx.Properties) || changed
		}

		if decision.Belonging != "Native" && ctx.Metadata && isTopLevelMetadataFile(ctx) &&
			ctx.OwnerKey != "Configuration" && ctx.OwnerKey != "Language.Русский" &&
			ctx.OwnerKind != "DefinedType" && ctx.OwnerKind != "EventSubscription" {
			changed = cleanupAdoptedObjectFormReferences(ctx.Properties) || changed
			contract, hasContract := formDynamicListContracts[ctx.OwnerKey]
			retainedCommands := retainedOwnerCommands[ctx.OwnerKey]
			rule, hasRule := adoptedStubMetaDataRules[ctx.OwnerKey]
			if hasContract && hasRule {
				changed = normalizeAdoptedStubExtFormComposition(ctx.Doc, mergeAdoptedStubMetaDataIntoFormContract(contract, rule), retainedCommands) || changed
			} else if hasRule {
				changed = normalizeAdoptedStubMetaDataComposition(ctx.Doc, ctx.OwnerKind, rule, retainedCommands) || changed
			} else if hasContract {
				changed = normalizeAdoptedStubExtFormComposition(ctx.Doc, contract, retainedCommands) || changed
			} else {
				changed = normalizeAdoptedObjectComposition(ctx.Doc, ctx.OwnerKind, retainedCommands) || changed
			}
		}

		if ctx.OwnerKind == "Subsystem" && ctx.Metadata {
			changed = normalizeSubsystemChildObjects(ctx.Doc, subsystemChain(ctx.RelPath), contexts, decisions) || changed
			changed = normalizeSubsystemContent(ctx.Doc, contexts, decisions) || changed
		}

		if ctx.OwnerKind == "ChartOfCharacteristicTypes" && strings.EqualFold(ctx.FileName, "Predefined.xml") {
			changed = normalizeChartOfCharacteristicTypesPredefined(ctx, contexts) || changed
		}

		if decision.Truncated && ctx.Metadata && isTopLevelMetadataFile(ctx) {
			changed = normalizeTruncatedMetadataStub(ctx.Doc, ctx.Properties) || changed
		}

		if filepath.Base(filepath.Dir(ctx.Path)) == dirCommonModules && ctx.Properties != nil {
			updated, err := disablePrivilegedMode(ctx.Properties)
			if err != nil {
				return fmt.Errorf("ошибка при изменении привилегированного режима в файле %s: %w", ctx.FileName, err)
			}
			changed = updated || changed
		}

		changed = cleanupForbiddenRegisterMovements(ctx.Doc, blockedForbiddenObjectKeys) || changed
		if ctx.OwnerKind != "DefinedType" && ctx.OwnerKind != "EventSubscription" {
			changed = cleanupExcludedReferences(ctx.Doc, excludedRefs, excludedMetadataPrefixes, truncatedKeys, truncatedChildPrefixes) || changed
		}
		originalExtendedObjects := collectMetadataOriginalUUIDs(ctx.Doc)
		changed = replaceGUIDsInDoc(ctx.Doc, guidReplacements) || changed
		if decision.Belonging != "Native" && ctx.Metadata && isTopLevelMetadataFile(ctx) &&
			ctx.OwnerKey != "Configuration" && ctx.OwnerKey != "Language.Русский" {
			changed = ensureAdoptedExtendedConfigurationObjects(ctx.Doc, originalExtendedObjects) || changed
		}

		if changed {
			changedFilesCount++
			if err := ctx.Doc.WriteToFile(ctx.Path); err != nil {
				return fmt.Errorf("ошибка при записи файла %s: %w", ctx.Path, err)
			}
			writtenFilesCount++
		}
	}
	log.Printf("xml progress: apply object changes %d/%d file=done", len(contexts), len(contexts))
	logXMLStepCompleted("apply object changes", applyObjectChangesStartedAt, fmt.Sprintf("changed_files=%d written_files=%d", changedFilesCount, writtenFilesCount))

	if cfg.IsFormValidationEnabled() {
		log.Printf("xml step: validate dynamic list contracts")
		validateDynamicListsStartedAt := time.Now()
		if err := validateFormDynamicListContractsIndexed(contexts, indexes, decisions, formDynamicListContracts); err != nil {
			return err
		}
		logXMLStepCompleted("validate dynamic list contracts", validateDynamicListsStartedAt)
	}

	log.Printf("xml step: verify old guids")
	verifyOldGUIDsStartedAt := time.Now()
	if err := verifyNoOldGUIDs(contexts, guidReplacements, excludedPaths); err != nil {
		return err
	}
	logXMLStepCompleted("verify old GUIDs", verifyOldGUIDsStartedAt)

	log.Printf("xml step: remove root service artifacts")
	cleanupRootServiceArtifactsStartedAt := time.Now()
	if err := cleanupRootServiceArtifacts(dir); err != nil {
		return err
	}
	logXMLStepCompleted("cleanup root service artifacts", cleanupRootServiceArtifactsStartedAt)

	log.Printf("xml step: remove excluded files")
	removeExcludedFilesStartedAt := time.Now()
	removedExcludedFilesCount, err := removeExcludedFiles(dir, excludedPaths)
	if err != nil {
		return err
	}
	logXMLStepCompleted("remove excluded files", removeExcludedFilesStartedAt, fmt.Sprintf("excluded_paths=%d removed_files=%d", len(excludedPaths), removedExcludedFilesCount))

	log.Printf("xml change completed: dir=%s contexts=%d decisions=%d excluded_paths=%d changed_files=%d written_files=%d", dir, len(contexts), len(decisions), len(excludedPaths), changedFilesCount, writtenFilesCount)
	return nil
}

func ResumeChangeFilesFromValidation(cfg *config.Configuration, dir string) error {
	log.Printf("xml resume started from validation: dir=%s", dir)

	state, err := buildChangeFilesState(cfg, dir)
	if err != nil {
		return err
	}

	if cfg.IsFormValidationEnabled() {
		log.Printf("xml step: validate dynamic list contracts")
		validateDynamicListsStartedAt := time.Now()
		if err := validateFormDynamicListContractsIndexed(state.contexts, state.indexes, state.decisions, state.formDynamicListContracts); err != nil {
			return err
		}
		logXMLStepCompleted("validate dynamic list contracts", validateDynamicListsStartedAt)
	}

	// Resume mode runs on an already rewritten temp tree. At this point we no longer
	// have the original old->new GUID mapping, so we only finish validation/cleanup.
	log.Printf("xml step: remove root service artifacts")
	cleanupRootServiceArtifactsStartedAt := time.Now()
	if err := cleanupRootServiceArtifacts(dir); err != nil {
		return err
	}
	logXMLStepCompleted("cleanup root service artifacts", cleanupRootServiceArtifactsStartedAt)

	log.Printf("xml step: remove excluded files")
	removeExcludedFilesStartedAt := time.Now()
	removedExcludedFilesCount, err := removeExcludedFiles(dir, state.excludedPaths)
	if err != nil {
		return err
	}
	logXMLStepCompleted("remove excluded files", removeExcludedFilesStartedAt, fmt.Sprintf("excluded_paths=%d removed_files=%d", len(state.excludedPaths), removedExcludedFilesCount))

	log.Printf("xml resume completed: dir=%s", dir)
	return nil
}

func GetFormatVersion(path string) (string, error) {
	doc, err := readXMLFile(filepath.Join(path, mainFile))
	if err != nil {
		return "", err
	}

	metaDataObject := doc.SelectElement("MetaDataObject")
	if metaDataObject == nil {
		return "", fmt.Errorf("MetaDataObject элемент не найден в %s", mainFile)
	}

	return metaDataObject.SelectAttrValue("version", ""), nil
}

func loadXMLContexts(root string) ([]*FileProcessingContext, error) {
	contexts := make([]*FileProcessingContext, 0, 128)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("ошибка при обработке файла %s: %w", path, walkErr)
		}

		if d.IsDir() || !isXMLFile(d.Name()) {
			return nil
		}

		doc, err := readXMLFile(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("ошибка при получении относительного пути %s: %w", path, err)
		}
		relPathSlash := filepath.ToSlash(relPath)

		kind, name, key := detectOwner(relPath, doc)
		contexts = append(contexts, &FileProcessingContext{
			Doc:              doc,
			Path:             path,
			RelPath:          relPathSlash,
			FileName:         d.Name(),
			Metadata:         isMetadataObjectDoc(doc),
			TopLevelMetadata: isTopLevelMetadataRelPath(relPathSlash),
			Properties:       findProperties(doc),
			OwnerKind:        kind,
			OwnerName:        name,
			OwnerKey:         key,
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка при обходе директорий: %w", err)
	}

	return contexts, nil
}

func buildChangeFilesState(cfg *config.Configuration, dir string) (*changeFilesState, error) {
	loadContextsStartedAt := time.Now()
	contexts, err := loadXMLContexts(dir)
	if err != nil {
		return nil, err
	}
	indexes := buildContextIndexes(contexts)
	logXMLStepCompleted("loadXMLContexts", loadContextsStartedAt, fmt.Sprintf("contexts=%d", len(contexts)))

	fillDumpInfo(contexts)

	log.Printf("xml step: build object sets")
	buildObjectSetsStartedAt := time.Now()
	includedNativeObjects := collectConfiguredNativeObjects(contexts, cfg.IncludedNativeObjects)
	includedAdoptedStubObjects := collectConfiguredAdoptedStubObjects(contexts, cfg)
	adoptedStubMetaDataRules := collectAdoptedStubMetaDataRules(cfg, dir)
	for key := range adoptedStubMetaDataRules {
		includedAdoptedStubObjects[key] = struct{}{}
	}
	forbiddenAdoptedStubObjects := collectConfiguredForbiddenStubObjects(contexts, cfg.ForbiddenAdoptedStubObjects)
	primaryNativeObjects := collectPrimaryNativeObjects(contexts, cfg.NativePrefixes, includedNativeObjects)
	excludedSubsystemObjects := collectExcludedSubsystemObjects(contexts, cfg.ExcludedSubsystems, cfg.NativePrefixes)
	configuredExcludedObjects := collectConfiguredExcludedObjects(contexts, cfg.ExcludedObjects)
	excludedObjects := mergeObjectSets(excludedSubsystemObjects, configuredExcludedObjects)
	logXMLStepCompleted("build object sets", buildObjectSetsStartedAt)

	log.Printf("xml step: collect subsystem decisions")
	collectSubsystemDecisionsStartedAt := time.Now()
	subsystemDecisions := collectSubsystemDecisions(contexts, cfg)
	decisions := make(map[string]objectDecision)
	for _, ctx := range contexts {
		if ctx.OwnerKey == "" {
			continue
		}
		if _, exists := decisions[ctx.OwnerKey]; exists {
			continue
		}
		if ctx.OwnerKind == "Subsystem" {
			decision, ok := subsystemDecisions[ctx.OwnerKey]
			if ok {
				decisions[ctx.OwnerKey] = decision
				continue
			}
		}
		decisions[ctx.OwnerKey] = decideObject(ctx, cfg, primaryNativeObjects, excludedObjects, includedAdoptedStubObjects, forbiddenAdoptedStubObjects)
	}
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	logXMLStepCompleted("collect subsystem decisions", collectSubsystemDecisionsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	log.Printf("xml step: promote referenced objects")
	promoteReferencedObjectsStartedAt := time.Now()
	referenceGraph := collectReferenceGraph(contexts, cfg, primaryNativeObjects)
	incomingReferenceGraph := collectIncomingReferenceGraph(referenceGraph)
	adoptedStubExtReferenceGraph := collectAdoptedStubExtReferenceGraph(contexts, decisions)
	formDynamicListContracts := collectFormDynamicListContracts(contexts, decisions)
	promoteReferencedObjectsToAdoptedStubIndexed(contexts, indexes, decisions, cfg, referenceGraph, incomingReferenceGraph, adoptedStubExtReferenceGraph, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects)
	promoteRegisterDocumentOwnersToNativeIndexed(contexts, indexes, decisions, cfg, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, collectRegisterDocumentReferences(contexts))
	applyFormDynamicListContracts(decisions, formDynamicListContracts, forbiddenAdoptedStubObjects)
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	logXMLStepCompleted("promote referenced objects", promoteReferencedObjectsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	collectCleanupSetsStartedAt := time.Now()
	excludedPaths := make(map[string]struct{})
	for _, ctx := range contexts {
		decision := decisions[ctx.OwnerKey]
		if decision.Excluded {
			excludedPaths[ctx.Path] = struct{}{}
			continue
		}
		if isRootServiceFile(ctx) {
			excludedPaths[ctx.Path] = struct{}{}
			continue
		}
		if ctx.FileName != configDumpInfo && ctx.OwnerKey != "Configuration" && !isTopLevelMetadataFile(ctx) && decision.Belonging != "Native" {
			excludedPaths[ctx.Path] = struct{}{}
		}
	}
	collectAdoptedCommonModuleModulePaths(dir, decisions, excludedPaths)
	collectAdoptedCommandModulePaths(contexts, decisions, excludedPaths)
	logXMLStepCompleted("collect cleanup sets", collectCleanupSetsStartedAt, fmt.Sprintf("excluded_paths=%d", len(excludedPaths)))

	return &changeFilesState{
		contexts:                 contexts,
		indexes:                  indexes,
		decisions:                decisions,
		formDynamicListContracts: formDynamicListContracts,
		adoptedStubMetaDataRules: adoptedStubMetaDataRules,
		excludedPaths:            excludedPaths,
	}, nil
}

func buildContextIndexes(contexts []*FileProcessingContext) *contextIndexes {
	indexes := &contextIndexes{
		byOwnerKey: make(map[string][]*FileProcessingContext),
		byRelPath:  make(map[string]*FileProcessingContext, len(contexts)),
		byPath:     make(map[string]*FileProcessingContext, len(contexts)),
		byFileName: make(map[string][]*FileProcessingContext),
	}

	for _, ctx := range contexts {
		if ctx == nil {
			continue
		}
		if ctx.OwnerKey != "" {
			indexes.byOwnerKey[ctx.OwnerKey] = append(indexes.byOwnerKey[ctx.OwnerKey], ctx)
		}
		if ctx.RelPath != "" {
			indexes.byRelPath[ctx.RelPath] = ctx
		}
		if ctx.Path != "" {
			indexes.byPath[ctx.Path] = ctx
		}
		if ctx.FileName != "" {
			indexes.byFileName[ctx.FileName] = append(indexes.byFileName[ctx.FileName], ctx)
		}
	}

	return indexes
}

func logXMLStepCompleted(step string, startedAt time.Time, details ...string) {
	duration := time.Since(startedAt)
	if len(details) > 0 && strings.TrimSpace(details[0]) != "" {
		log.Printf("xml step completed: %s duration=%s %s", step, duration, details[0])
		return
	}
	log.Printf("xml step completed: %s duration=%s", step, duration)
}

func removeExcludedFiles(root string, excludedPaths map[string]struct{}) (int, error) {
	removedCount := 0
	for path := range excludedPaths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return removedCount, fmt.Errorf("ошибка удаления исключенного файла %s: %w", path, err)
		}
		cleanupEmptyParents(filepath.Dir(path), root)
		removedCount++
	}

	return removedCount, nil
}

func fillDumpInfo(contexts []*FileProcessingContext) {
	for _, ctx := range contexts {
		if ctx.FileName != mainFile || ctx.Properties == nil {
			continue
		}
		getInfo(ctx.Properties)
		return
	}
}

func collectSubsystemDecisions(contexts []*FileProcessingContext, cfg *config.Configuration) map[string]objectDecision {
	states := make([]subsystemState, 0, 32)
	adopted := make(map[string]struct{})
	decisions := make(map[string]objectDecision)

	for _, ctx := range contexts {
		if ctx.OwnerKind != "Subsystem" {
			continue
		}

		chain := subsystemChain(ctx.RelPath)
		if len(chain) == 0 {
			continue
		}
		if isExcludedSubsystemChain(chain, cfg.ExcludedSubsystems) {
			decisions[ctx.OwnerKey] = objectDecision{Excluded: true}
			continue
		}

		root := chain[0]
		_, adoptedRoot := adoptedSubsystemRoots[root]
		if !adoptedRoot && !hasNativePrefix(root, cfg.NativePrefixes) && !subsystemChainHasNativeAncestor(chain, cfg.NativePrefixes) {
			continue
		}

		states = append(states, subsystemState{
			Key:   ctx.OwnerKey,
			Name:  ctx.OwnerName,
			Chain: chain,
		})
	}

	for _, state := range states {
		if !hasNativePrefix(state.Name, cfg.NativePrefixes) {
			continue
		}
		for _, ancestor := range state.Chain[:len(state.Chain)-1] {
			adopted[ancestor] = struct{}{}
		}
	}

	grouped := make(map[string][]subsystemState)
	for _, state := range states {
		grouped[state.Key] = append(grouped[state.Key], state)
	}

	for key, group := range grouped {
		decision := objectDecision{Excluded: true}

		for _, state := range group {
			switch {
			case hasNativePrefix(state.Name, cfg.NativePrefixes):
				decision = objectDecision{Belonging: "Native"}
			case subsystemChainHasNativeAncestor(state.Chain, cfg.NativePrefixes):
				decision = objectDecision{Belonging: "Native"}
			case len(state.Chain) == 1:
				if decision.Belonging != "Native" {
					decision = objectDecision{Belonging: "AdoptedStub"}
				}
			case containsName(adopted, state.Name):
				if decision.Belonging != "Native" {
					decision = objectDecision{Belonging: "AdoptedStub"}
				}
			}
			if decision.Belonging == "Native" {
				break
			}
		}

		switch {
		case decision.Belonging == "Native":
			debugDecision(key, "subsystem decision: native")
		case decision.Belonging == "AdoptedStub":
			debugDecision(key, "subsystem decision: adopted stub")
		default:
			debugDecision(key, "subsystem decision: excluded")
		}

		decisions[key] = decision
	}

	return decisions
}

func subsystemChainHasNativeAncestor(chain []string, nativePrefixes []string) bool {
	if len(chain) < 2 {
		return false
	}

	for _, name := range chain[:len(chain)-1] {
		if hasNativePrefix(name, nativePrefixes) {
			return true
		}
	}

	return false
}

func decideObject(
	ctx *FileProcessingContext,
	cfg *config.Configuration,
	primaryNativeObjects map[string]struct{},
	excludedObjects map[string]struct{},
	includedAdoptedStubObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
) objectDecision {
	if ctx.OwnerKey == "" {
		return objectDecision{}
	}

	if ctx.OwnerKey == "Configuration" {
		debugDecision(ctx.OwnerKey, "adopted stub as special root configuration")
		return objectDecision{Belonging: "AdoptedStub"}
	}

	if ctx.OwnerKey == "Language.Русский" {
		debugDecision(ctx.OwnerKey, "adopted stub as special russian language")
		return objectDecision{Belonging: "AdoptedStub"}
	}

	if _, forbidden := forbiddenAdoptedStubObjects[ctx.OwnerKey]; forbidden {
		debugDecision(ctx.OwnerKey, "hard-excluded by forbidden adopted stub list")
		return objectDecision{Excluded: true}
	}

	if _, excluded := excludedObjects[ctx.OwnerKey]; excluded {
		debugDecision(ctx.OwnerKey, "soft-excluded by excluded object set")
		return objectDecision{Excluded: true}
	}

	if _, primary := primaryNativeObjects[ctx.OwnerKey]; primary {
		debugDecision(ctx.OwnerKey, "native by primary set")
		return objectDecision{Belonging: "Native"}
	}

	if _, included := includedAdoptedStubObjects[ctx.OwnerKey]; included {
		debugDecision(ctx.OwnerKey, "adopted stub by config include")
		return objectDecision{Belonging: "AdoptedStub", Truncated: shouldTruncateAdoptedStub(ctx)}
	}

	debugDecision(ctx.OwnerKey, "soft-excluded by default")
	return objectDecision{Excluded: true}
}

func isHardExcludedObject(
	ctx *FileProcessingContext,
	key string,
	cfg *config.Configuration,
	excludedObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
) bool {
	// Жесткое исключение здесь означает только forbidden_*.
	// excluded_* и excluded_subsystems остаются мягкими и могут быть восстановлены по ссылкам.
	if key != "" {
		if _, forbidden := forbiddenAdoptedStubObjects[key]; forbidden {
			return true
		}
	}
	_ = ctx
	_ = cfg
	_ = excludedObjects
	return false
}

func collectGUIDReplacements(contexts []*FileProcessingContext, decisions map[string]objectDecision) map[string]string {
	replacements := make(map[string]string)

	collectGUIDReplacementsFromConfigDump(contexts, decisions, replacements)

	for _, ctx := range contexts {
		decision, ok := decisions[ctx.OwnerKey]
		if !ok || !ctx.Metadata || decision.Excluded || decision.Belonging != "AdoptedStub" {
			continue
		}
		for guid := range collectOwnedGUIDs(ctx.Doc) {
			normalized := strings.ToLower(guid)
			if _, exists := replacements[normalized]; !exists {
				replacements[normalized] = newGUID()
			}
		}
	}

	return replacements
}

func collectGUIDReplacementsFromConfigDump(contexts []*FileProcessingContext, decisions map[string]objectDecision, replacements map[string]string) {
	ctx := findConfigDumpContext(contexts)
	if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
		return
	}

	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		if localName(el.Tag) == "Metadata" {
			name := strings.TrimSpace(el.SelectAttrValue("name", ""))
			id := strings.TrimSpace(el.SelectAttrValue("id", ""))
			if isAdoptedMetadata(name, decisions) {
				for _, guid := range extractGUIDs(id) {
					ensureGUIDReplacement(replacements, guid)
				}
			}
		}

		for _, child := range el.ChildElements() {
			walk(child)
		}
	}

	walk(ctx.Doc.Root())
}

func findConfigDumpContext(contexts []*FileProcessingContext) *FileProcessingContext {
	for _, ctx := range contexts {
		if ctx.FileName == configDumpInfo {
			return ctx
		}
	}
	return nil
}

func isAdoptedMetadata(metadataName string, decisions map[string]objectDecision) bool {
	key := metadataDecisionKey(metadataName)
	if key == "" {
		return false
	}

	decision, ok := decisions[key]
	return ok && !decision.Excluded && decision.Belonging == "AdoptedStub"
}

func metadataDecisionKey(metadataName string) string {
	parts := strings.Split(strings.TrimSpace(metadataName), ".")
	if len(parts) < 2 {
		return ""
	}

	if parts[0] == "Configuration" {
		return "Configuration"
	}

	return parts[0] + "." + parts[1]
}

func ensureGUIDReplacement(replacements map[string]string, guid string) {
	normalized := strings.ToLower(strings.TrimSpace(guid))
	if normalized == "" {
		return
	}
	if _, exists := replacements[normalized]; exists {
		return
	}
	replacements[normalized] = newGUID()
}

func collectOwnedGUIDs(doc *etree.Document) map[string]struct{} {
	result := make(map[string]struct{})
	root := doc.Root()
	if root == nil {
		return result
	}

	var walk func(el *etree.Element, inInternalInfo bool)
	walk = func(el *etree.Element, inInternalInfo bool) {
		tag := strings.ToLower(localName(el.Tag))
		nextInternal := inInternalInfo || strings.Contains(tag, "internalinfo")

		if tag != "classid" {
			if _, ok := guidFieldNames[tag]; ok {
				for _, guid := range extractGUIDs(el.Text()) {
					result[guid] = struct{}{}
				}
			}
			if nextInternal {
				for _, guid := range extractGUIDs(el.Text()) {
					result[guid] = struct{}{}
				}
			}
		}

		for _, attr := range el.Attr {
			name := strings.ToLower(localName(attr.Key))
			if name == "classid" {
				continue
			}
			if _, ok := guidFieldNames[name]; ok || nextInternal {
				for _, guid := range extractGUIDs(attr.Value) {
					result[guid] = struct{}{}
				}
			}
		}

		for _, child := range el.ChildElements() {
			walk(child, nextInternal)
		}
	}

	walk(root, false)

	return result
}

func collectExcludedReferences(decisions map[string]objectDecision) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	for key, decision := range decisions {
		if !decision.Excluded {
			continue
		}

		kind, name := splitObjectKey(key)
		if kind == "" || name == "" {
			continue
		}

		if _, ok := result[kind]; !ok {
			result[kind] = make(map[string]struct{})
		}
		result[kind][name] = struct{}{}
	}
	return result
}

func collectBlockedForbiddenObjectKeys(decisions map[string]objectDecision, forbidden map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for key := range forbidden {
		decision, ok := decisions[key]
		if ok && decision.Belonging == "Native" {
			continue
		}
		result[key] = struct{}{}
	}
	return result
}

func collectExcludedDecisionKeys(decisions map[string]objectDecision) map[string]struct{} {
	result := make(map[string]struct{})
	for key, decision := range decisions {
		if decision.Excluded {
			result[key] = struct{}{}
		}
	}
	return result
}

func collectReferenceMapFromObjectKeys(keys map[string]struct{}) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	for key := range keys {
		kind, name := splitObjectKey(key)
		if kind == "" || name == "" {
			continue
		}
		if _, ok := result[kind]; !ok {
			result[kind] = make(map[string]struct{})
		}
		result[kind][name] = struct{}{}
	}
	return result
}

func cleanupForbiddenRegisterMovements(doc *etree.Document, forbiddenObjects map[string]struct{}) bool {
	root := doc.Root()
	if root == nil || len(forbiddenObjects) == 0 {
		return false
	}

	forbiddenRegisters := make(map[string]map[string]struct{})
	for key := range forbiddenObjects {
		kind, name := splitObjectKey(key)
		if !isRegisterKind(kind) || kind == "" || name == "" {
			continue
		}
		if _, ok := forbiddenRegisters[kind]; !ok {
			forbiddenRegisters[kind] = make(map[string]struct{})
		}
		forbiddenRegisters[kind][name] = struct{}{}
	}
	if len(forbiddenRegisters) == 0 {
		return false
	}

	changed := false

	var walk func(parent *etree.Element, inMovementContext bool)
	walk = func(parent *etree.Element, inMovementContext bool) {
		currentMovementContext := inMovementContext || isMovementContextRoot(localName(parent.Tag))
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if shouldRemoveForbiddenRegisterMovement(parent, child, forbiddenRegisters) {
				parent.RemoveChild(child)
				changed = true
				continue
			}
			walk(child, currentMovementContext)
		}

		if currentMovementContext && isMovementContainer(localName(parent.Tag)) && parent.Parent() != nil && len(parent.ChildElements()) == 0 {
			parent.Parent().RemoveChild(parent)
			changed = true
		}
	}

	walk(root, false)
	return changed
}

func isMovementContextRoot(tag string) bool {
	switch strings.ToLower(localName(tag)) {
	case "registerrecords", "source":
		return true
	default:
		return false
	}
}

func shouldRemoveForbiddenRegisterMovement(parent, child *etree.Element, forbiddenRegisters map[string]map[string]struct{}) bool {
	if len(forbiddenRegisters) == 0 {
		return false
	}

	if !isMovementContainer(localName(parent.Tag)) {
		return false
	}

	for kind, names := range forbiddenRegisters {
		for ref := range collectMetadataReferences(child) {
			refKind, refName := splitObjectKey(ref)
			if !strings.EqualFold(refKind, kind) {
				continue
			}
			if _, ok := names[refName]; ok {
				return true
			}
		}
	}

	return false
}

func isRegisterKind(kind string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(kind)), "register")
}

func isMovementContainer(tag string) bool {
	switch strings.ToLower(localName(tag)) {
	case "registerrecords", "source", "content":
		return true
	default:
		return false
	}
}

func mergeReferenceMaps(maps ...map[string]map[string]struct{}) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	for _, refs := range maps {
		for kind, names := range refs {
			if _, ok := result[kind]; !ok {
				result[kind] = make(map[string]struct{})
			}
			for name := range names {
				result[kind][name] = struct{}{}
			}
		}
	}
	return result
}

func findContextByOwnerKey(contexts []*FileProcessingContext, ownerKey string) *FileProcessingContext {
	return findContextByOwnerKeyIndexed(nil, contexts, ownerKey)
}

func findContextByOwnerKeyIndexed(indexes *contextIndexes, contexts []*FileProcessingContext, ownerKey string) *FileProcessingContext {
	var fallback *FileProcessingContext
	candidates := contexts
	if indexes != nil && ownerKey != "" {
		candidates = indexes.byOwnerKey[ownerKey]
	}
	for _, ctx := range candidates {
		if ctx == nil || ctx.OwnerKey != ownerKey {
			continue
		}
		if isTopLevelMetadataFile(ctx) {
			return ctx
		}
		if fallback == nil {
			fallback = ctx
		}
	}
	return fallback
}

func findContextByFileName(contexts []*FileProcessingContext, fileName string) *FileProcessingContext {
	return findContextByFileNameIndexed(nil, contexts, fileName)
}

func findContextByFileNameIndexed(indexes *contextIndexes, contexts []*FileProcessingContext, fileName string) *FileProcessingContext {
	candidates := contexts
	if indexes != nil && fileName != "" {
		candidates = indexes.byFileName[fileName]
	}
	for _, ctx := range candidates {
		if ctx == nil {
			continue
		}
		if strings.EqualFold(ctx.FileName, fileName) {
			return ctx
		}
	}
	return nil
}

func shouldTruncateAdoptedStub(ctx *FileProcessingContext) bool {
	if ctx == nil {
		return true
	}

	// AdoptedStubExt — это частный случай AdoptedStub:
	// сохраняем реквизитный состав, но не переносим формы и код.
	switch ctx.OwnerKind {
	case "Configuration", "Language", "DefinedType", "EventSubscription":
		return false
	default:
		return true
	}
}

func cleanupExcludedReferences(doc *etree.Document, excluded map[string]map[string]struct{}, excludedMetadataPrefixes []string, truncatedKeys map[string]struct{}, truncatedChildPrefixes []string) bool {
	root := doc.Root()
	if root == nil {
		return false
	}
	if len(excluded) == 0 && len(excludedMetadataPrefixes) == 0 && len(truncatedKeys) == 0 && len(truncatedChildPrefixes) == 0 {
		return false
	}

	changed := false
	excludedTraversal := newExcludedMetadataTraversalState(excludedMetadataPrefixes)

	var walk func(parent *etree.Element)
	walk = func(parent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if shouldRemoveElement(parent, child, excluded, truncatedKeys, truncatedChildPrefixes, excludedTraversal) {
				parent.RemoveChild(child)
				changed = true
				continue
			}
			walk(child)
		}
	}

	walk(root)
	return changed
}

func normalizeCommandInterfaceFile(ctx *FileProcessingContext, contexts []*FileProcessingContext) bool {
	return normalizeCommandInterfaceFileIndexed(ctx, contexts, nil)
}

func normalizeCommandInterfaceFileIndexed(ctx *FileProcessingContext, contexts []*FileProcessingContext, indexes *contextIndexes) bool {
	if ctx == nil || ctx.Doc == nil {
		return false
	}

	source := findContextByFileNameIndexed(indexes, contexts, "MainSectionCommandInterface.xml")
	if source == nil || source.Doc == nil {
		return false
	}

	if ctx.Doc != nil && source.Doc != nil && ctx.Doc.Root() != nil && source.Doc.Root() != nil {
		ctx.Doc = source.Doc.Copy()
		return true
	}

	return false
}

func shouldRemoveElement(parent, child *etree.Element, excluded map[string]map[string]struct{}, truncatedKeys map[string]struct{}, truncatedChildPrefixes []string, excludedTraversal *excludedMetadataTraversalState) bool {
	tag := localName(child.Tag)
	parentTag := localName(parent.Tag)

	if parentTag == "Commands" && tag == "Command" {
		return false
	}

	if names, ok := excluded[tag]; ok {
		if _, exists := names[strings.TrimSpace(child.Text())]; exists {
			return true
		}
	}

	if names, ok := excluded[parentTag]; ok && tag == "Item" {
		nameElem := child.FindElement(".//Name")
		if nameElem != nil {
			if _, exists := names[strings.TrimSpace(nameElem.Text())]; exists {
				return true
			}
		}
	}

	if tag == "Metadata" {
		metadataName := strings.TrimSpace(child.SelectAttrValue("name", ""))
		if metadataName != "" && isExcludedMetadata(metadataName, excluded) {
			return true
		}
		if metadataName != "" && isTruncatedMetadataChild(metadataName, truncatedKeys) {
			return true
		}
	}

	if isDynamicListFieldChildKind(tag) && subtreeContainsExcludedMetadataRef(child.FindElement("./Properties/Type"), excludedTraversal) {
		return true
	}

	if isCharacteristicElement(tag) && subtreeContainsExcludedMetadataRef(child, excludedTraversal) {
		return true
	}

	if subtreeContainsExcludedMetadataRef(child, excludedTraversal) {
		if isMetadataReferenceValueElement(tag) || isMetadataReferenceContainer(parentTag) {
			return true
		}
	}

	if subtreeContainsMetadataRefPrefix(child, truncatedChildPrefixes) {
		if isCharacteristicElement(tag) || strings.EqualFold(localName(parent.Tag), "Characteristics") {
			return true
		}
		if isMetadataReferenceValueElement(tag) {
			return true
		}
		if isMetadataReferenceContainer(parentTag) {
			return true
		}
	}

	if isCommandReferenceElement(tag) {
		if subtreeContainsExcludedMetadataRef(child, excludedTraversal) || subtreeContainsMetadataRefPrefix(child, truncatedChildPrefixes) {
			return true
		}
	}

	return false
}

func promoteReferencedObjectsToAdoptedStub(
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	cfg *config.Configuration,
	referenceGraph map[string]map[string]struct{},
	incomingReferenceGraph map[string]map[string]struct{},
	adoptedStubExtReferenceGraph map[string]map[string]struct{},
	primaryNativeObjects map[string]struct{},
	excludedObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
) {
	promoteReferencedObjectsToAdoptedStubIndexed(contexts, nil, decisions, cfg, referenceGraph, incomingReferenceGraph, adoptedStubExtReferenceGraph, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects)
}

func promoteReferencedObjectsToAdoptedStubIndexed(
	contexts []*FileProcessingContext,
	indexes *contextIndexes,
	decisions map[string]objectDecision,
	cfg *config.Configuration,
	referenceGraph map[string]map[string]struct{},
	incomingReferenceGraph map[string]map[string]struct{},
	adoptedStubExtReferenceGraph map[string]map[string]struct{},
	primaryNativeObjects map[string]struct{},
	excludedObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
) {
	for {
		changed := false

		for _, ctx := range contexts {
			if ctx == nil || ctx.OwnerKey == "" || ctx.OwnerKey == "Language.Русский" {
				continue
			}

			decision, ok := decisions[ctx.OwnerKey]
			if !ok || decision.Excluded {
				continue
			}
			if !isRefDrivenInclusionSource(ctx, decision) {
				continue
			}

			refs := referenceGraph[ctx.OwnerKey]
			adoptedStubExtRefs := adoptedStubExtReferenceGraph[ctx.OwnerKey]
			if ctx.OwnerKey == "Configuration" {
				refs = collectConfigurationChildObjectReferences(ctx.Doc.Root(), primaryNativeObjects)
			}

			for ref := range refs {
				if ref == "" || ref == ctx.OwnerKey {
					continue
				}

				refDecision, refExists := decisions[ref]
				if refExists && refDecision.Belonging == "Native" {
					continue
				}
				if ctx.OwnerKind == "Role" && (!refExists || refDecision.Excluded) {
					debugDecision(ref, "kept excluded: role rights are not a ref-driven restore source")
					continue
				}
				if ctx.OwnerKind == "Subsystem" && (!refExists || refDecision.Excluded) {
					debugDecision(ref, "kept excluded: native subsystem is not a ref-driven restore source")
					continue
				}
				if refExists &&
					refDecision.Excluded &&
					isReferencedOnlyByNativeSubsystems(ref, incomingReferenceGraph, contexts, indexes, decisions) {
					debugDecision(ref, "kept excluded: only native subsystems reference it")
					continue
				}

				if _, primary := primaryNativeObjects[ref]; primary {
					refCtx := findContextByOwnerKeyIndexed(indexes, contexts, ref)
					if isHardExcludedObject(refCtx, ref, cfg, excludedObjects, forbiddenAdoptedStubObjects) {
						continue
					}
					if !refExists || refDecision.Excluded {
						decisions[ref] = objectDecision{Belonging: "Native"}
						debugDecision(ref, "promoted to native from reference "+ctx.OwnerKey)
						changed = true
					}
					continue
				}

				if _, forbidden := forbiddenAdoptedStubObjects[ref]; forbidden {
					continue
				}

				refCtx := findContextByOwnerKeyIndexed(indexes, contexts, ref)
				needsAdoptedStubExt := shouldUseAdoptedStubExtForReference(ctx, decision, ref, refCtx, adoptedStubExtRefs)

				if !refExists || refDecision.Excluded {
					if isHardExcludedObject(refCtx, ref, cfg, excludedObjects, forbiddenAdoptedStubObjects) {
						continue
					}
					if refCtx != nil && hasNativePrefix(refCtx.OwnerName, cfg.NativePrefixes) {
						continue
					}
					decisions[ref] = objectDecision{Belonging: "AdoptedStub", Truncated: !needsAdoptedStubExt && shouldTruncateAdoptedStub(refCtx)}
					debugDecision(ref, "promoted to adopted stub from reference "+ctx.OwnerKey)
					changed = true
					continue
				}

				if needsAdoptedStubExt && refDecision.Belonging == "AdoptedStub" && refDecision.Truncated {
					refDecision.Truncated = false
					decisions[ref] = refDecision
					debugDecision(ref, "kept as AdoptedStubExt from hard reference "+ctx.OwnerKey)
					changed = true
				}
			}
		}

		if !changed {
			return
		}
	}
}

func collectIncomingReferenceGraph(referenceGraph map[string]map[string]struct{}) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	for sourceKey, refs := range referenceGraph {
		if sourceKey == "" {
			continue
		}
		for ref := range refs {
			if ref == "" || ref == sourceKey {
				continue
			}
			if _, ok := result[ref]; !ok {
				result[ref] = make(map[string]struct{})
			}
			result[ref][sourceKey] = struct{}{}
		}
	}
	return result
}

func isReferencedOnlyByNativeSubsystems(
	target string,
	incomingReferenceGraph map[string]map[string]struct{},
	contexts []*FileProcessingContext,
	indexes *contextIndexes,
	decisions map[string]objectDecision,
) bool {
	sources := incomingReferenceGraph[target]
	if len(sources) == 0 {
		return false
	}

	hasEligibleSource := false
	for sourceKey := range sources {
		sourceCtx := findContextByOwnerKeyIndexed(indexes, contexts, sourceKey)
		if sourceCtx == nil {
			continue
		}
		sourceDecision, ok := decisions[sourceKey]
		if !ok || sourceDecision.Excluded || !isRefDrivenInclusionSource(sourceCtx, sourceDecision) {
			continue
		}
		hasEligibleSource = true
		if sourceCtx.OwnerKind != "Subsystem" || sourceDecision.Belonging != "Native" {
			return false
		}
	}

	return hasEligibleSource
}

func isRefDrivenInclusionSource(ctx *FileProcessingContext, decision objectDecision) bool {
	if decision.Belonging == "Native" {
		return true
	}

	// Ненативные определяемые типы и подписки на события остаются источником RefDrivenInclusion,
	// если они уже перенесены как AdoptedStubExt и сохранили состав.
	return ctx != nil &&
		(ctx.OwnerKind == "DefinedType" || ctx.OwnerKind == "EventSubscription") &&
		decision.Belonging == "AdoptedStub" &&
		!decision.Truncated
}

func shouldUseAdoptedStubExtForReference(
	sourceCtx *FileProcessingContext,
	sourceDecision objectDecision,
	ref string,
	refCtx *FileProcessingContext,
	sourceAdoptedStubExtRefs map[string]struct{},
) bool {
	if sourceCtx != nil &&
		sourceCtx.OwnerKind == "DefinedType" &&
		sourceDecision.Belonging == "AdoptedStub" &&
		!sourceDecision.Truncated {
		return false
	}

	if sourceAdoptedStubExtRefs != nil {
		if _, exists := sourceAdoptedStubExtRefs[ref]; exists {
			return true
		}
	}

	return refCtx != nil && refCtx.OwnerKind == "DefinedType"
}

func collectMetadataReferences(el *etree.Element) map[string]struct{} {
	result := make(map[string]struct{})
	if el == nil {
		return result
	}

	var walk func(*etree.Element)
	walk = func(node *etree.Element) {
		for _, ref := range metadataReferencesFromValue(strings.TrimSpace(node.Text())) {
			result[ref] = struct{}{}
		}

		for _, attr := range node.Attr {
			for _, ref := range metadataReferencesFromValue(strings.TrimSpace(attr.Value)) {
				result[ref] = struct{}{}
			}
		}

		for _, child := range node.ChildElements() {
			walk(child)
		}
	}

	walk(el)
	return result
}

func collectStyleItemReferences(el *etree.Element, styleItemKeys map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	if el == nil || len(styleItemKeys) == 0 {
		return result
	}

	var addFromValue func(string)
	addFromValue = func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}

		matches := styleReferencePattern.FindAllStringSubmatch(value, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			key := "StyleItem." + strings.TrimSpace(match[1])
			if _, ok := styleItemKeys[key]; ok {
				result[key] = struct{}{}
			}
		}
	}

	var walk func(*etree.Element)
	walk = func(node *etree.Element) {
		addFromValue(node.Text())

		for _, attr := range node.Attr {
			addFromValue(attr.Value)
		}

		for _, child := range node.ChildElements() {
			walk(child)
		}
	}

	walk(el)
	return result
}

func collectExistingOwnerKeys(contexts []*FileProcessingContext, ownerKind string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" {
			continue
		}
		if ownerKind != "" && !strings.EqualFold(ctx.OwnerKind, ownerKind) {
			continue
		}
		result[ctx.OwnerKey] = struct{}{}
	}
	return result
}

func collectReferenceGraph(contexts []*FileProcessingContext, cfg *config.Configuration, primaryNativeObjects map[string]struct{}) map[string]map[string]struct{} {
	graph := make(map[string]map[string]struct{})
	styleItemKeys := collectExistingOwnerKeys(contexts, "StyleItem")

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}

		refs := collectMetadataReferences(ctx.Doc.Root())
		for key := range collectStyleItemReferences(ctx.Doc.Root(), styleItemKeys) {
			refs[key] = struct{}{}
		}
		if ctx.OwnerKey == "Configuration" {
			for key := range collectConfigurationChildObjectReferences(ctx.Doc.Root(), primaryNativeObjects) {
				refs[key] = struct{}{}
			}
			for key := range refs {
				kind, _ := splitObjectKey(key)
				if strings.EqualFold(kind, "Catalog") {
					delete(refs, key)
				}
			}
		}
		if existing, ok := graph[ctx.OwnerKey]; ok {
			for ref := range refs {
				existing[ref] = struct{}{}
			}
			continue
		}
		graph[ctx.OwnerKey] = refs
	}

	return graph
}

func collectAdoptedStubExtReferenceGraph(contexts []*FileProcessingContext, decisions map[string]objectDecision) map[string]map[string]struct{} {
	graph := make(map[string]map[string]struct{})

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}
		if strings.Contains(ctx.RelPath, "/Forms/") {
			decision, ok := decisions[ctx.OwnerKey]
			if !ok || decision.Excluded || decision.Belonging != "Native" {
				continue
			}
		}

		refs := collectAdoptedStubExtReferences(ctx)
		if len(refs) == 0 {
			continue
		}

		if existing, ok := graph[ctx.OwnerKey]; ok {
			for ref := range refs {
				existing[ref] = struct{}{}
			}
			continue
		}
		graph[ctx.OwnerKey] = refs
	}

	return graph
}

func collectAdoptedStubExtReferences(ctx *FileProcessingContext) map[string]struct{} {
	result := make(map[string]struct{})
	root := ctx.Doc.Root()
	if root == nil {
		return result
	}

	if strings.Contains(ctx.RelPath, "/Forms/") {
		for ref := range collectMetadataReferences(root) {
			result[ref] = struct{}{}
		}
	}

	return result
}

func collectRegisterDocumentReferences(contexts []*FileProcessingContext) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}
		if ctx.OwnerKind != "Document" {
			continue
		}

		for _, registerRecords := range ctx.Doc.Root().FindElements(".//RegisterRecords") {
			for ref := range collectMetadataReferences(registerRecords) {
				kind, _ := splitObjectKey(ref)
				if !isRegisterKind(kind) {
					continue
				}
				if _, ok := result[ref]; !ok {
					result[ref] = make(map[string]struct{})
				}
				result[ref][ctx.OwnerKey] = struct{}{}
			}
		}
	}

	return result
}

func promoteRegisterDocumentOwnersToNative(
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	cfg *config.Configuration,
	primaryNativeObjects map[string]struct{},
	excludedObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
	registerDocumentReferences map[string]map[string]struct{},
) {
	promoteRegisterDocumentOwnersToNativeIndexed(contexts, nil, decisions, cfg, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, registerDocumentReferences)
}

func promoteRegisterDocumentOwnersToNativeIndexed(
	contexts []*FileProcessingContext,
	indexes *contextIndexes,
	decisions map[string]objectDecision,
	cfg *config.Configuration,
	primaryNativeObjects map[string]struct{},
	excludedObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
	registerDocumentReferences map[string]map[string]struct{},
) {
	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.OwnerKind == "" {
			continue
		}

		decision, ok := decisions[ctx.OwnerKey]
		if !ok || decision.Excluded || decision.Belonging != "Native" || !isRegisterKind(ctx.OwnerKind) {
			continue
		}

		for docKey := range registerDocumentReferences[ctx.OwnerKey] {
			if docKey == "" {
				continue
			}

			docCtx := findContextByOwnerKeyIndexed(indexes, contexts, docKey)
			if isHardExcludedObject(docCtx, docKey, cfg, excludedObjects, forbiddenAdoptedStubObjects) {
				continue
			}
			if _, primary := primaryNativeObjects[docKey]; primary {
				continue
			}

			docDecision, exists := decisions[docKey]
			if exists && docDecision.Belonging == "Native" && !docDecision.Excluded {
				continue
			}

			decisions[docKey] = objectDecision{Belonging: "Native"}
			debugDecision(docKey, "promoted to native from register owner "+ctx.OwnerKey)
		}
	}
}

func collectFormDynamicListContracts(contexts []*FileProcessingContext, decisions map[string]objectDecision) map[string]formDynamicListContract {
	result := make(map[string]formDynamicListContract)

	for _, ctx := range contexts {
		if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}

		relPath := filepath.ToSlash(ctx.RelPath)
		if !strings.Contains(relPath, "/Forms/") {
			continue
		}
		decision, ok := decisions[ctx.OwnerKey]
		if !ok || decision.Excluded || decision.Belonging != "Native" {
			continue
		}

		root := ctx.Doc.Root()
		for _, attr := range root.FindElements(".//Attribute") {
			attrName := strings.TrimSpace(attr.SelectAttrValue("name", ""))
			if attrName == "" || !isDynamicListAttribute(attr) {
				continue
			}

			mainTable := strings.TrimSpace(textOfFirst(attr, ".//MainTable"))
			if mainTable == "" {
				continue
			}

			refs := metadataReferencesFromValue(mainTable)
			if len(refs) == 0 {
				continue
			}

			targetKey := refs[0]
			contract := result[targetKey]
			if contract.RequiredFields == nil {
				contract.RequiredFields = make(map[string]struct{})
			}
			for field := range collectDynamicListAttributeFields(root, attrName) {
				contract.RequiredFields[field] = struct{}{}
			}
			for field := range collectDynamicListDeclaredFields(attr) {
				contract.RequiredFields[field] = struct{}{}
			}
			for alias := range collectDynamicListVirtualFields(attr) {
				if contract.QueryAliases == nil {
					contract.QueryAliases = make(map[string]struct{})
				}
				contract.QueryAliases[alias] = struct{}{}
			}
			result[targetKey] = contract
		}
	}

	return result
}

func isDynamicListAttribute(attr *etree.Element) bool {
	if attr == nil {
		return false
	}

	for _, typeEl := range attr.FindElements(".//Type/v8:Type") {
		if strings.EqualFold(strings.TrimSpace(typeEl.Text()), "cfg:DynamicList") ||
			strings.EqualFold(strings.TrimSpace(typeEl.Text()), "v8:DynamicList") ||
			strings.EqualFold(strings.TrimSpace(typeEl.Text()), "DynamicList") {
			return true
		}
	}

	return false
}

func textOfFirst(parent *etree.Element, path string) string {
	if parent == nil {
		return ""
	}
	el := parent.FindElement(path)
	if el == nil {
		return ""
	}
	return strings.TrimSpace(el.Text())
}

func collectDynamicListAttributeFields(root *etree.Element, attrName string) map[string]struct{} {
	result := make(map[string]struct{})
	if root == nil || attrName == "" {
		return result
	}

	var walk func(*etree.Element)
	walk = func(node *etree.Element) {
		tag := strings.ToLower(localName(node.Tag))
		switch tag {
		case "field", "datapath", "rowpicturedatapath", "keyfield", "typesfilterfield", "objectfield", "typefield", "valuefield", "datapathfield":
			if field, ok := extractDynamicListFieldName(strings.TrimSpace(node.Text()), attrName); ok {
				result[field] = struct{}{}
			}
		}
		for _, child := range node.ChildElements() {
			walk(child)
		}
	}

	walk(root)
	return result
}

func collectDynamicListVirtualFields(attr *etree.Element) map[string]struct{} {
	result := make(map[string]struct{})
	if attr == nil {
		return result
	}

	for field := range collectDynamicListCalculatedFields(attr) {
		result[field] = struct{}{}
	}

	if !isDynamicListManualQuery(attr) {
		return result
	}

	queryText := textOfFirst(attr, ".//Settings/QueryText")
	if queryText == "" {
		queryText = textOfFirst(attr, ".//QueryText")
	}
	if queryText == "" {
		return result
	}

	selectPart := queryText
	if idx := indexOfQueryFromClause(queryText); idx >= 0 {
		selectPart = queryText[:idx]
	}

	for _, match := range dynamicListQueryAliasRegexp.FindAllStringSubmatch(selectPart, -1) {
		if len(match) < 2 {
			continue
		}
		alias := strings.TrimSpace(match[1])
		if alias == "" {
			continue
		}
		result[alias] = struct{}{}
	}

	return result
}

func collectDynamicListCalculatedFields(attr *etree.Element) map[string]struct{} {
	result := make(map[string]struct{})
	if attr == nil {
		return result
	}

	for _, calc := range attr.FindElements(".//CalculatedField") {
		for _, child := range calc.ChildElements() {
			if !strings.EqualFold(localName(child.Tag), "dataPath") {
				continue
			}
			name := strings.TrimSpace(child.Text())
			if name == "" {
				continue
			}
			result[name] = struct{}{}
		}
	}

	return result
}

func collectDynamicListDeclaredFields(attr *etree.Element) map[string]struct{} {
	result := make(map[string]struct{})
	if attr == nil {
		return result
	}

	for _, field := range attr.FindElements(".//Field") {
		for _, child := range field.ChildElements() {
			tag := localName(child.Tag)
			if !strings.EqualFold(tag, "dataPath") && !strings.EqualFold(tag, "field") {
				continue
			}
			name := strings.TrimSpace(child.Text())
			if name == "" {
				continue
			}
			result[name] = struct{}{}
		}
	}

	return result
}

func isDynamicListManualQuery(attr *etree.Element) bool {
	value := strings.TrimSpace(textOfFirst(attr, ".//Settings/ManualQuery"))
	if value == "" {
		value = strings.TrimSpace(textOfFirst(attr, ".//ManualQuery"))
	}
	return strings.EqualFold(value, "true")
}

func indexOfQueryFromClause(queryText string) int {
	upper := strings.ToUpper(queryText)
	if idx := strings.Index(upper, "\nИЗ"); idx >= 0 {
		return idx + 1
	}
	if idx := strings.Index(upper, "\r\nИЗ"); idx >= 0 {
		return idx + 2
	}
	if idx := strings.Index(upper, " ИЗ "); idx >= 0 {
		return idx + 1
	}
	return -1
}

func extractDynamicListFieldName(value, attrName string) (string, bool) {
	value = strings.TrimSpace(value)
	attrName = strings.TrimSpace(attrName)
	if value == "" || attrName == "" {
		return "", false
	}

	prefix := attrName + "."
	if !strings.HasPrefix(value, prefix) {
		return "", false
	}

	rest := strings.TrimPrefix(value, prefix)
	if rest == "" {
		return "", false
	}

	field := strings.TrimSpace(rest)
	if field == "" {
		return "", false
	}

	return field, true
}

func applyFormDynamicListContracts(decisions map[string]objectDecision, contracts map[string]formDynamicListContract, forbidden map[string]struct{}) {
	for key := range contracts {
		if _, blocked := forbidden[key]; blocked {
			continue
		}

		decision, ok := decisions[key]
		if !ok || decision.Excluded || decision.Belonging == "Native" {
			continue
		}

		if decision.Belonging == "AdoptedStub" && decision.Truncated {
			decision.Truncated = false
			decisions[key] = decision
			debugDecision(key, "kept as AdoptedStubExt(Form) from dynamic list contract")
		}
	}
}

func collectAdoptedStubMetaDataRules(cfg *config.Configuration, dir string) map[string]adoptedStubMetaDataRule {
	result := make(map[string]adoptedStubMetaDataRule)
	if cfg == nil || !cfg.IsMetaDataFileEnabled() {
		return result
	}

	templatePath := filepath.Join(dir, "CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		log.Printf("additional processing: skip AdoptedStubMetaData, cannot read %s: %v", templatePath, err)
		return result
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("additional processing: skip AdoptedStubMetaData, cannot parse %s: %v", templatePath, err)
		return result
	}

	collectAdoptedStubMetaDataRulesFromValue(raw, result)
	return result
}

func collectAdoptedStubMetaDataRulesFromValue(value any, rules map[string]adoptedStubMetaDataRule) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch key {
			case "Реквизиты":
				collectAdoptedStubMetaDataAttributePaths(child, rules)
			case "ТабличныеЧасти":
				collectAdoptedStubMetaDataTablePaths(child, rules)
			default:
				collectAdoptedStubMetaDataRulesFromValue(child, rules)
			}
		}
	case []any:
		for _, child := range typed {
			if path, ok := child.(string); ok {
				registerAdoptedStubMetaDataPath(rules, path)
				continue
			}
			collectAdoptedStubMetaDataRulesFromValue(child, rules)
		}
	}
}

func collectAdoptedStubMetaDataAttributePaths(value any, rules map[string]adoptedStubMetaDataRule) {
	items, ok := value.([]any)
	if !ok {
		return
	}

	for _, item := range items {
		path, ok := item.(string)
		if !ok {
			continue
		}
		registerAdoptedStubMetaDataPath(rules, path)
	}
}

func collectAdoptedStubMetaDataTablePaths(value any, rules map[string]adoptedStubMetaDataRule) {
	switch typed := value.(type) {
	case map[string]any:
		for _, child := range typed {
			collectAdoptedStubMetaDataTablePaths(child, rules)
		}
	case []any:
		for _, item := range typed {
			path, ok := item.(string)
			if !ok {
				continue
			}
			registerAdoptedStubMetaDataPath(rules, path)
		}
	}
}

func registerAdoptedStubMetaDataPath(rules map[string]adoptedStubMetaDataRule, path string) {
	segments := strings.Split(strings.TrimSpace(path), ".")
	if len(segments) < 4 {
		return
	}

	kind := normalizeMetaDataFileKind(segments[0])
	name := strings.TrimSpace(segments[1])
	if kind == "" || name == "" {
		return
	}

	key := kind + "." + name
	rule := rules[key]

	switch {
	case len(segments) >= 4 && strings.EqualFold(segments[2], "Реквизит"):
		attrName := strings.TrimSpace(segments[3])
		if hasAdoptedStubMetaDataNativePrefix(attrName) {
			if rule.NativeAttributes == nil {
				rule.NativeAttributes = make(map[string]struct{})
			}
			rule.NativeAttributes[attrName] = struct{}{}
		}
	case len(segments) >= 6 && strings.EqualFold(segments[2], "ТабличнаяЧасть") && strings.EqualFold(segments[4], "Реквизит"):
		sectionName := strings.TrimSpace(segments[3])
		attrName := strings.TrimSpace(segments[5])
		if sectionName != "" && hasAdoptedStubMetaDataNativePrefix(attrName) {
			if rule.NativeTabularSections == nil {
				rule.NativeTabularSections = make(map[string]map[string]struct{})
			}
			if rule.NativeTabularSections[sectionName] == nil {
				rule.NativeTabularSections[sectionName] = make(map[string]struct{})
			}
			rule.NativeTabularSections[sectionName][attrName] = struct{}{}
		}
	default:
		return
	}

	rules[key] = rule
}

func normalizeMetaDataFileKind(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if canonical, ok := metaDataFileKindAliases[trimmed]; ok {
		return canonical
	}

	return normalizeConfiguredKind(trimmed)
}

func hasAdoptedStubMetaDataNativePrefix(name string) bool {
	return strings.HasPrefix(strings.TrimSpace(name), "упо_")
}

func applyAdoptedStubMetaDataRules(
	decisions map[string]objectDecision,
	rules map[string]adoptedStubMetaDataRule,
	excludedObjects map[string]struct{},
	forbidden map[string]struct{},
) {
	for key := range rules {
		if _, blocked := forbidden[key]; blocked {
			continue
		}
		if _, excluded := excludedObjects[key]; excluded {
			continue
		}

		decision, ok := decisions[key]
		if !ok || decision.Belonging == "Native" {
			continue
		}

		if decision.Excluded || decision.Belonging == "" {
			decisions[key] = objectDecision{Belonging: "AdoptedStub"}
			debugDecision(key, "kept as AdoptedStubMetaData from CommonTemplate упо_MetaDataFile")
			continue
		}

		if decision.Belonging == "AdoptedStub" && decision.Truncated {
			decision.Truncated = false
			decisions[key] = decision
			debugDecision(key, "expanded to AdoptedStubMetaData from CommonTemplate упо_MetaDataFile")
		}
	}
}

func collectConfigurationChildObjectReferences(root *etree.Element, primaryNativeObjects map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	if root == nil {
		return result
	}

	var walk func(*etree.Element)
	walk = func(node *etree.Element) {
		if strings.EqualFold(node.Tag, "ChildObjects") {
			for _, child := range node.ChildElements() {
				kind, ok := configurationChildObjectKinds[child.Tag]
				if !ok {
					continue
				}
				name := strings.TrimSpace(child.Text())
				if name == "" {
					continue
				}
				key := kind + "." + name
				if _, ok := primaryNativeObjects[key]; !ok {
					continue
				}

				result[key] = struct{}{}
			}
		}

		for _, child := range node.ChildElements() {
			walk(child)
		}
	}

	walk(root)
	return result
}

func normalizeRootConfigurationChildObjects(doc *etree.Document, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	allowed := collectRootConfigurationChildObjects(contexts, decisions)
	if len(allowed) == 0 {
		return false
	}

	changed := false
	for _, childObjects := range root.FindElements(".//ChildObjects") {
		existing := make(map[string]struct{})
		for _, child := range append([]etree.Token(nil), childObjects.Child...) {
			el, ok := child.(*etree.Element)
			if !ok {
				continue
			}
			kind, ok := configurationChildObjectKinds[el.Tag]
			if !ok {
				continue
			}
			name := strings.TrimSpace(el.Text())
			if name == "" {
				continue
			}
			key := kind + "." + name
			if _, keep := allowed[key]; keep {
				existing[key] = struct{}{}
				continue
			}
			childObjects.RemoveChild(el)
			changed = true
		}

		for key := range allowed {
			if _, exists := existing[key]; exists {
				continue
			}
			parts := strings.SplitN(key, ".", 2)
			if len(parts) != 2 || parts[1] == "" {
				continue
			}
			tag, ok := configurationChildObjectTag(parts[0])
			if !ok {
				continue
			}
			addSimpleElement(childObjects, tag, parts[1])
			changed = true
		}
	}

	return changed
}

func collectRootConfigurationChildObjects(contexts []*FileProcessingContext, decisions map[string]objectDecision) map[string]struct{} {
	result := make(map[string]struct{})
	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.OwnerKey == "Configuration" {
			continue
		}
		if !ctx.Metadata || !isTopLevelMetadataFile(ctx) {
			continue
		}
		decision, ok := decisions[ctx.OwnerKey]
		if !ok || decision.Excluded {
			continue
		}
		result[ctx.OwnerKey] = struct{}{}
	}
	return result
}

func configurationChildObjectTag(kind string) (string, bool) {
	for tag, candidateKind := range configurationChildObjectKinds {
		if candidateKind == kind {
			return tag, true
		}
	}
	return "", false
}

func collectExcludedSubsystemObjects(contexts []*FileProcessingContext, excludedSubsystems []string, nativePrefixes []string) map[string]struct{} {
	result := make(map[string]struct{})
	if len(excludedSubsystems) == 0 {
		return result
	}

	for _, ctx := range contexts {
		if ctx.OwnerKind != "Subsystem" || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}

		if !isExcludedSubsystemChain(subsystemChain(ctx.RelPath), excludedSubsystems) {
			continue
		}

		for key := range collectMetadataReferences(ctx.Doc.Root()) {
			if key == "" || key == ctx.OwnerKey {
				continue
			}
			debugDecision(key, "found in excluded subsystem content: "+ctx.OwnerKey)
			result[key] = struct{}{}
		}
	}

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.OwnerKind == "Subsystem" {
			continue
		}
		if !belongsToExcludedSubsystem(ctx, excludedSubsystems) {
			continue
		}
		debugDecision(ctx.OwnerKey, "found by excluded subsystem reference in owner properties")
		result[ctx.OwnerKey] = struct{}{}
	}

	return result
}

func collectConfiguredNativeObjects(contexts []*FileProcessingContext, configured []string) map[string]struct{} {
	return collectConfiguredObjectKeys(contexts, configured)
}

func collectConfiguredExcludedObjects(contexts []*FileProcessingContext, configured []string) map[string]struct{} {
	return collectConfiguredObjectKeys(contexts, configured)
}

func collectConfiguredAdoptedStubObjects(contexts []*FileProcessingContext, cfg *config.Configuration) map[string]struct{} {
	result := collectConfiguredObjectKeys(contexts, cfg.IncludedAdoptedStubObjects)
	for key := range collectConfiguredObjectKeys(contexts, cfg.AdditionalAdoptedObjects) {
		result[key] = struct{}{}
	}
	for key := range collectConfiguredObjectKeys(contexts, cfg.IncludedObjects) {
		result[key] = struct{}{}
	}
	return result
}

func collectConfiguredForbiddenStubObjects(contexts []*FileProcessingContext, configured []string) map[string]struct{} {
	return collectConfiguredObjectKeys(contexts, configured)
}

func collectConfiguredObjectKeys(contexts []*FileProcessingContext, configured []string) map[string]struct{} {
	result := make(map[string]struct{})
	if len(configured) == 0 {
		return result
	}

	byName := make(map[string][]string)
	for _, ctx := range contexts {
		if ctx.OwnerKey == "" || ctx.OwnerName == "" {
			continue
		}
		normalizedName := strings.ToLower(strings.TrimSpace(ctx.OwnerName))
		byName[normalizedName] = append(byName[normalizedName], ctx.OwnerKey)
	}

	for _, raw := range configured {
		for key := range resolveConfiguredObjectKey(raw, byName) {
			result[key] = struct{}{}
		}
	}

	return result
}

func collectPrimaryNativeObjects(contexts []*FileProcessingContext, nativePrefixes []string, includedNativeObjects map[string]struct{}) map[string]struct{} {
	result := collectNativePrefixObjects(contexts, nativePrefixes)
	for key := range includedNativeObjects {
		result[key] = struct{}{}
	}
	return result
}

func collectNativePrefixObjects(contexts []*FileProcessingContext, nativePrefixes []string) map[string]struct{} {
	result := make(map[string]struct{})
	if len(nativePrefixes) == 0 {
		return result
	}

	for _, ctx := range contexts {
		if ctx.OwnerKey == "" || ctx.OwnerName == "" {
			continue
		}
		if hasNativePrefix(ctx.OwnerName, nativePrefixes) {
			result[ctx.OwnerKey] = struct{}{}
		}
	}

	return result
}

func resolveConfiguredObjectKey(raw string, byName map[string][]string) map[string]struct{} {
	result := make(map[string]struct{})

	value := strings.TrimSpace(raw)
	if value == "" {
		return result
	}

	if strings.Contains(value, ".") {
		parts := strings.SplitN(value, ".", 2)
		kind := normalizeConfiguredKind(parts[0])
		name := strings.TrimSpace(parts[1])
		if kind != "" && name != "" {
			result[kind+"."+name] = struct{}{}
			return result
		}
	}

	for _, key := range byName[strings.ToLower(value)] {
		result[key] = struct{}{}
	}

	return result
}

func normalizeConfiguredKind(kind string) string {
	trimmed := strings.TrimSpace(kind)
	if trimmed == "" {
		return ""
	}

	if canonical, ok := metadataKindAliases[strings.ToLower(trimmed)]; ok {
		return canonical
	}

	if strings.EqualFold(trimmed, "Справочник") {
		return "Catalog"
	}

	for _, known := range metadataKinds {
		if strings.EqualFold(trimmed, known) {
			return known
		}
	}

	return ""
}

func mergeObjectSets(sets ...map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for _, set := range sets {
		for key := range set {
			result[key] = struct{}{}
		}
	}
	return result
}

func debugDecision(key, message string) {
	if os.Getenv("FILES_CONVERTER_DEBUG_DECISIONS") == "" {
		return
	}
	if key != "InformationRegister.упо_СхемаОтраженияНачисленийPM_ERP" &&
		key != "AccumulationRegister.упо_ДенежныеСредстваКВыплате" &&
		key != "Document.упо_ФактическиеРесурсы" &&
		key != "AccumulationRegister.упо_ФактическиеДанныеРесурсы" &&
		key != "Document.упо_ОтражениеФактаВПроектномБюджетировании" &&
		key != "Document.Удалитьупо_ОтражениеФактаВПроектномБюджетировании" &&
		key != "Catalog.упо_ВерсииПланов" &&
		key != "DataProcessor.упо_ЗагрузкаДанныхMSProject" &&
		key != "DataProcessor.упо_ОграничениеДоступаПоОбъектамPM" &&
		key != "DataProcessor.упо_ФормированиеЗаказовЭлементаПлана" &&
		key != "Catalog.ГруппыПользователей" &&
		key != "Catalog.ПрофилиГруппДоступа" &&
		key != "Subsystem.Администрирование" &&
		key != "CommonModule.ВерсионированиеОбъектовСобытия" &&
		key != "Document.ЗаказКлиента" &&
		key != "Document.ЗаказПоставщику" &&
		key != "Document.ЗаказНаПроизводство2_2" &&
		key != "CommonCommand.ИнтеграцияС1СДокументооборотСоздатьПисьмо" &&
		key != "CommonCommand.ПрисоединенныеФайлы" &&
		key != "CommonCommand.ИнтеграцияС1СДокументооборотНачатьОбработку" &&
		key != "CommonAttribute.ОбластьДанныхВспомогательныеДанные" &&
		key != "CommonAttribute.ОбластьДанныхОсновныеДанные" &&
		key != "DefinedType.упо_ОбъектИнтеграцииДляОбработкиЗаписи" &&
		key != "DefinedType.упо_ОбъектИнтеграцииДругойКонфигурации" &&
		key != "DefinedType.упо_ОбъектИнтеграцииРесурс" &&
		key != "DefinedType.упо_ОбъектИнтеграцииЭлементПланаОбъект" &&
		key != "DefinedType.упо_ОбъектыЗаданий" {
		return
	}
	log.Printf("decision debug: %s -> %s", key, message)
}

func metadataReferencesFromValue(value string) []string {
	if value == "" {
		return nil
	}

	result := make([]string, 0, 1)
	visitMetadataReferencesFast(value, func(ref string) bool {
		if idx := strings.Index(ref, "."); idx >= 0 {
			name := ref[idx+1:]
			if nextDot := strings.Index(name, "."); nextDot >= 0 {
				ref = ref[:idx+1+nextDot]
			}
		}
		result = append(result, ref)
		return false
	})
	if len(result) > 0 {
		return result
	}

	matches := metadataReferencePattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}

	result = make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		name := strings.TrimSpace(match[2])
		if idx := strings.Index(name, "."); idx >= 0 {
			name = name[:idx]
		}
		if name == "" {
			continue
		}
		result = append(result, match[1]+"."+name)
	}
	return result
}

func mayContainMetadataRef(value string) bool {
	if value == "" || !strings.Contains(value, ".") {
		return false
	}

	return strings.Contains(value, ":") ||
		strings.Contains(value, "Ref") ||
		strings.Contains(value, "Object") ||
		strings.Contains(value, "Catalog") ||
		strings.Contains(value, "Document") ||
		strings.Contains(value, "Register") ||
		strings.Contains(value, "Chart") ||
		strings.Contains(value, "Common") ||
		strings.Contains(value, "Subsystem") ||
		strings.Contains(value, "Command")
}

func visitMetadataReferencesFast(value string, visitor func(ref string) bool) bool {
	if value == "" || visitor == nil || !mayContainMetadataRef(value) {
		return false
	}

	tokenStart := -1
	for idx, r := range value {
		if isMetadataTokenSeparator(r) {
			if tokenStart >= 0 && visitMetadataReferencesInToken(value[tokenStart:idx], visitor) {
				return true
			}
			tokenStart = -1
			continue
		}
		if tokenStart < 0 {
			tokenStart = idx
		}
	}

	if tokenStart >= 0 {
		return visitMetadataReferencesInToken(value[tokenStart:], visitor)
	}

	return false
}

func visitMetadataReferencesInToken(token string, visitor func(ref string) bool) bool {
	if token == "" || !strings.Contains(token, ".") {
		return false
	}

	dotIdx := strings.Index(token, ".")
	if dotIdx <= 0 || dotIdx >= len(token)-1 {
		return false
	}

	kind, ok := normalizeMetadataReferenceKind(token[:dotIdx])
	if !ok {
		return false
	}

	path := normalizeMetadataReferencePath(token[dotIdx+1:])
	if path == "" {
		return false
	}

	return visitor(kind + "." + path)
}

func normalizeMetadataReferenceKind(token string) (string, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}

	if idx := strings.LastIndex(token, ":"); idx >= 0 {
		token = token[idx+1:]
	}

	for _, suffix := range []string{
		"TabularSectionRow",
		"TabularSection",
		"ValueManager",
		"RecordSet",
		"Selection",
		"Manager",
		"Object",
		"List",
		"Ref",
	} {
		if strings.HasSuffix(token, suffix) {
			token = strings.TrimSuffix(token, suffix)
			break
		}
	}

	_, ok := configurationChildObjectKinds[token]
	return token, ok
}

func normalizeMetadataReferencePath(token string) string {
	if token == "" {
		return ""
	}

	var builder strings.Builder
	prevDot := false
	for _, r := range token {
		switch {
		case isMetadataReferencePathRune(r):
			builder.WriteRune(r)
			prevDot = false
		case r == '.':
			if prevDot || builder.Len() == 0 {
				return strings.TrimSuffix(builder.String(), ".")
			}
			builder.WriteRune(r)
			prevDot = true
		default:
			return strings.TrimSuffix(builder.String(), ".")
		}
	}

	return strings.TrimSuffix(builder.String(), ".")
}

func isMetadataReferencePathRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func isMetadataTokenSeparator(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}

	switch r {
	case '<', '>', '"', '\'', '/', '\\', '=', ',', ';', '(', ')', '[', ']', '{', '}', '|':
		return true
	default:
		return false
	}
}

func cleanupDefinedTypeExcludedTypes(properties *etree.Element, excludedObjects map[string]struct{}) bool {
	if properties == nil || len(excludedObjects) == 0 {
		return false
	}

	typeElement := properties.FindElement("Type")
	if typeElement == nil {
		return false
	}

	changed := false
	for i := len(typeElement.Child) - 1; i >= 0; i-- {
		child, ok := typeElement.Child[i].(*etree.Element)
		if !ok || !strings.EqualFold(localName(child.Tag), "Type") {
			continue
		}

		for _, ref := range metadataReferencesFromValue(strings.TrimSpace(child.Text())) {
			if _, excluded := excludedObjects[ref]; excluded {
				typeElement.RemoveChild(child)
				changed = true
				break
			}
		}
	}

	return changed
}

func collectExcludedMetadataPrefixes(excluded map[string]map[string]struct{}) []string {
	result := make([]string, 0, len(excluded)*2)
	for kind, names := range excluded {
		for name := range names {
			result = append(result, kind+"."+name)
		}
	}
	return result
}

func collectTruncatedKeys(decisions map[string]objectDecision) map[string]struct{} {
	result := make(map[string]struct{})
	for key, decision := range decisions {
		if decision.Truncated {
			result[key] = struct{}{}
		}
	}
	return result
}

func collectTruncatedChildPrefixes(truncatedKeys map[string]struct{}) []string {
	result := make([]string, 0, len(truncatedKeys))
	for key := range truncatedKeys {
		result = append(result, key+".")
	}
	return result
}

func collectNonNativeKeys(decisions map[string]objectDecision) map[string]struct{} {
	result := make(map[string]struct{})
	for key, decision := range decisions {
		if key == "" || decision.Belonging == "Native" {
			continue
		}
		if key == "Configuration" || key == "Language.Русский" {
			continue
		}
		result[key] = struct{}{}
	}
	return result
}

func isCharacteristicElement(tag string) bool {
	switch strings.ToLower(localName(tag)) {
	case "characteristics", "characteristic", "characteristictypes", "characteristicvalues":
		return true
	default:
		return false
	}
}

func newExcludedMetadataTraversalState(prefixes []string) *excludedMetadataTraversalState {
	if len(prefixes) == 0 {
		return nil
	}

	prefixSet := make(map[string]struct{}, len(prefixes))
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		prefixSet[prefix] = struct{}{}
	}
	if len(prefixSet) == 0 {
		return nil
	}

	return &excludedMetadataTraversalState{
		prefixSet:    prefixSet,
		subtreeCache: make(map[*etree.Element]nodeMark),
		valueCache:   make(map[string]bool),
	}
}

func subtreeContainsExcludedMetadataRef(el *etree.Element, state *excludedMetadataTraversalState) bool {
	if el == nil || state == nil || len(state.prefixSet) == 0 {
		return false
	}

	if mark, ok := state.subtreeCache[el]; ok && mark.computed {
		return mark.hasExcludedRef
	}

	result := nodeContainsExcludedMetadataRef(el, state)
	if !result {
		for _, child := range el.ChildElements() {
			if subtreeContainsExcludedMetadataRef(child, state) {
				result = true
				break
			}
		}
	}

	state.subtreeCache[el] = nodeMark{
		hasExcludedRef: result,
		computed:       true,
	}

	return result
}

func nodeContainsExcludedMetadataRef(el *etree.Element, state *excludedMetadataTraversalState) bool {
	if el == nil || state == nil {
		return false
	}

	if hasExcludedMetadataRefFast(strings.TrimSpace(el.Text()), state) {
		return true
	}

	for _, attr := range el.Attr {
		if hasExcludedMetadataRefFast(strings.TrimSpace(attr.Value), state) {
			return true
		}
	}

	return false
}

func hasExcludedMetadataRefFast(value string, state *excludedMetadataTraversalState) bool {
	if value == "" || state == nil || len(state.prefixSet) == 0 {
		return false
	}

	if cached, ok := state.valueCache[value]; ok {
		return cached
	}

	if !mayContainMetadataRef(value) {
		state.valueCache[value] = false
		return false
	}

	result := visitMetadataReferencesFast(value, func(ref string) bool {
		return isExcludedMetadataReference(ref, state.prefixSet)
	})
	state.valueCache[value] = result
	return result
}

func isExcludedMetadataReference(ref string, prefixSet map[string]struct{}) bool {
	current := strings.TrimSpace(ref)
	for current != "" {
		if _, ok := prefixSet[current]; ok {
			return true
		}
		idx := strings.LastIndex(current, ".")
		if idx < 0 {
			break
		}
		current = current[:idx]
	}
	return false
}

func subtreeContainsMetadataRefPrefix(el *etree.Element, prefixes []string) bool {
	if el == nil || len(prefixes) == 0 {
		return false
	}

	for _, prefix := range prefixes {
		if strings.Contains(strings.TrimSpace(el.Text()), prefix) {
			return true
		}
	}

	for _, attr := range el.Attr {
		for _, prefix := range prefixes {
			if strings.Contains(strings.TrimSpace(attr.Value), prefix) {
				return true
			}
		}
	}

	for _, child := range el.ChildElements() {
		if subtreeContainsMetadataRefPrefix(child, prefixes) {
			return true
		}
	}

	return false
}

func containsMetadataReference(value, prefix string) bool {
	if value == "" || prefix == "" {
		return false
	}

	if !mayContainMetadataRef(value) {
		return value == prefix ||
			strings.Contains(value, prefix+".") ||
			strings.Contains(value, "cfg:"+prefix) ||
			strings.Contains(value, "xr:"+prefix)
	}

	matched := visitMetadataReferencesFast(value, func(ref string) bool {
		return ref == prefix || strings.HasPrefix(ref, prefix+".")
	})
	if matched {
		return true
	}

	for _, ref := range metadataReferencesFromValue(value) {
		if ref == prefix || strings.HasPrefix(ref, prefix+".") {
			return true
		}
	}

	return value == prefix ||
		strings.Contains(value, prefix+".") ||
		strings.Contains(value, "cfg:"+prefix) ||
		strings.Contains(value, "xr:"+prefix)
}

func isMetadataReferenceValueElement(tag string) bool {
	switch strings.ToLower(localName(tag)) {
	case "item", "name", "field", "maintable", "keyfield", "typesfilterfield", "objectfield", "typefield", "valuefield", "datapathfield", "datapath", "defaultobjectform", "defaultlistform", "defaultchoiceform", "extendedconfigurationobject":
		return true
	default:
		return false
	}
}

func isMetadataReferenceContainer(tag string) bool {
	switch strings.ToLower(localName(tag)) {
	case "content", "registerrecords", "source", "type", "types", "list", "choices", "inputbystring", "basedon":
		return true
	default:
		return false
	}
}

func isCommandReferenceElement(tag string) bool {
	switch strings.ToLower(localName(tag)) {
	case "command", "commandname":
		return true
	default:
		return false
	}
}

func isTruncatedMetadataChild(metadataName string, truncatedKeys map[string]struct{}) bool {
	for key := range truncatedKeys {
		if metadataName != key && strings.HasPrefix(metadataName, key+".") {
			return true
		}
	}
	return false
}

func removeForbiddenStandardCommands(doc *etree.Document) bool {
	root := doc.Root()
	if root == nil {
		return false
	}

	targets := []string{
		"Form.StandardCommand.Change",
		"Form.StandardCommand.ChangeHistory",
		"Change",
		"ChangeHistory",
		"GetURL",
	}

	changed := false

	var walk func(parent *etree.Element)
	walk = func(parent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if isForbiddenStandardCommand(parent, child, targets) {
				parent.RemoveChild(child)
				changed = true
				continue
			}
			walk(child)
		}
	}

	walk(root)
	return changed
}

func cleanupMissingFormCommandReferences(doc *etree.Document, contexts []*FileProcessingContext) bool {
	root := doc.Root()
	if root == nil {
		return false
	}

	defined := collectDefinedFormCommands(root)

	changed := false

	var walk func(parent, grandparent *etree.Element)
	walk = func(parent, grandparent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if shouldRemoveFormCommandReference(child, defined, contexts) {
				if isCommandReferenceElement(child.Tag) && grandparent != nil {
					grandparent.RemoveChild(parent)
					changed = true
					return
				} else {
					parent.RemoveChild(child)
				}
				changed = true
				continue
			}
			walk(child, parent)
		}
	}

	walk(root, nil)
	return changed
}

func cleanupMissingFormConstantsSetReferences(doc *etree.Document, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	return cleanupMissingFormConstantsSetReferencesIndexed(doc, contexts, nil, decisions)
}

func cleanupMissingFormConstantsSetReferencesIndexed(doc *etree.Document, contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) bool {
	root := doc.Root()
	if root == nil {
		return false
	}

	changed := false

	var walk func(parent, grandparent *etree.Element)
	walk = func(parent, grandparent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if shouldRemoveMissingFormConstantsSetReference(child, contexts, indexes, decisions) {
				if grandparent != nil {
					grandparent.RemoveChild(parent)
				} else {
					parent.RemoveChild(child)
				}
				changed = true
				continue
			}
			walk(child, parent)
		}
	}

	walk(root, nil)
	return changed
}

func shouldRemoveMissingFormConstantsSetReference(el *etree.Element, contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) bool {
	if el == nil || !strings.EqualFold(localName(el.Tag), "DataPath") {
		return false
	}

	text := strings.TrimSpace(el.Text())
	if !strings.HasPrefix(text, "НаборКонстант.") {
		return false
	}

	constantName := strings.TrimPrefix(text, "НаборКонстант.")
	if constantName == "" {
		return false
	}

	return !topLevelMetadataIncludedIndexed("Constant."+constantName, contexts, indexes, decisions)
}

func cleanupMissingFormCommonAttributeDynamicListFields(doc *etree.Document, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	return cleanupMissingFormCommonAttributeDynamicListFieldsIndexed(doc, contexts, nil, decisions)
}

func cleanupMissingFormCommonAttributeDynamicListFieldsIndexed(doc *etree.Document, contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) bool {
	root := doc.Root()
	if root == nil {
		return false
	}

	blockedByAttribute := collectMissingFormCommonAttributeDynamicListFields(root, contexts, indexes, decisions)
	if len(blockedByAttribute) == 0 {
		return false
	}

	changed := false
	for _, attr := range root.FindElements(".//Attribute") {
		attrName := strings.TrimSpace(attr.SelectAttrValue("name", ""))
		blockedFields := blockedByAttribute[attrName]
		if len(blockedFields) == 0 {
			continue
		}

		for _, field := range append([]*etree.Element(nil), attr.FindElements(".//Field")...) {
			if shouldRemoveDynamicListDeclaredField(field, blockedFields) {
				if parent := field.Parent(); parent != nil {
					parent.RemoveChild(field)
					changed = true
				}
			}
		}
	}

	var walk func(parent, grandparent *etree.Element)
	walk = func(parent, grandparent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if shouldRemoveMissingFormDynamicListFieldReference(child, blockedByAttribute) {
				if grandparent != nil {
					grandparent.RemoveChild(parent)
				} else {
					parent.RemoveChild(child)
				}
				changed = true
				continue
			}
			walk(child, parent)
		}
	}

	walk(root, nil)
	return changed
}

func collectMissingFormCommonAttributeDynamicListFields(root *etree.Element, contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	if root == nil {
		return result
	}

	for _, attr := range root.FindElements(".//Attribute") {
		if !isDynamicListAttribute(attr) {
			continue
		}

		attrName := strings.TrimSpace(attr.SelectAttrValue("name", ""))
		if attrName == "" {
			continue
		}

		mainTable := strings.TrimSpace(textOfFirst(attr, ".//MainTable"))
		if mainTable == "" {
			continue
		}
		refs := metadataReferencesFromValue(mainTable)
		if len(refs) == 0 {
			continue
		}

		targetCtx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, refs[0])
		available := collectAvailableDynamicListFields(targetCtx)
		requiredFields := collectDynamicListDeclaredFields(attr)
		for field := range collectDynamicListAttributeFields(root, attrName) {
			requiredFields[field] = struct{}{}
		}
		for field := range requiredFields {
			if _, ok := available[field]; ok {
				continue
			}
			if topLevelMetadataIncludedIndexed("CommonAttribute."+field, contexts, indexes, decisions) {
				continue
			}
			if result[attrName] == nil {
				result[attrName] = make(map[string]struct{})
			}
			result[attrName][field] = struct{}{}
		}
	}

	return result
}

func shouldRemoveDynamicListDeclaredField(field *etree.Element, blockedFields map[string]struct{}) bool {
	if field == nil || len(blockedFields) == 0 {
		return false
	}

	for _, child := range field.ChildElements() {
		tag := localName(child.Tag)
		if !strings.EqualFold(tag, "dataPath") && !strings.EqualFold(tag, "field") {
			continue
		}
		if _, blocked := blockedFields[strings.TrimSpace(child.Text())]; blocked {
			return true
		}
	}

	return false
}

func shouldRemoveMissingFormDynamicListFieldReference(el *etree.Element, blockedByAttribute map[string]map[string]struct{}) bool {
	if el == nil || !strings.EqualFold(localName(el.Tag), "DataPath") {
		return false
	}

	text := strings.TrimSpace(el.Text())
	if text == "" {
		return false
	}

	for attrName, blockedFields := range blockedByAttribute {
		field, ok := extractDynamicListFieldName(text, attrName)
		if !ok {
			continue
		}
		_, blocked := blockedFields[field]
		return blocked
	}

	return false
}

func cleanupNativeFormNonNativeReferences(doc *etree.Document, nonNativeKeys map[string]struct{}) bool {
	root := doc.Root()
	if root == nil || len(nonNativeKeys) == 0 {
		return false
	}

	// Для форм нативных объектов допускаем расширенный stub:
	// ссылки на не-нативные объекты сохраняются только как реквизитный состав.
	changed := false
	for {
		blockedAttributes := collectBlockedFormAttributes(root, nonNativeKeys)
		passChanged := false

		var walk func(parent *etree.Element)
		walk = func(parent *etree.Element) {
			children := parent.ChildElements()
			for i := len(children) - 1; i >= 0; i-- {
				child := children[i]
				if shouldRemoveNativeFormElement(child, blockedAttributes, nonNativeKeys) {
					parent.RemoveChild(child)
					passChanged = true
					continue
				}
				walk(child)
			}
		}

		walk(root)
		if !passChanged {
			return changed
		}
		changed = true
	}
}

func collectBlockedFormAttributes(root *etree.Element, nonNativeKeys map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	if root == nil {
		return result
	}

	for _, attr := range root.FindElements(".//Attribute") {
		name := strings.TrimSpace(attr.SelectAttrValue("name", ""))
		if name == "" {
			continue
		}
		for _, mainTable := range attr.FindElements(".//MainTable") {
			for _, ref := range metadataReferencesFromValue(strings.TrimSpace(mainTable.Text())) {
				if _, blocked := nonNativeKeys[ref]; blocked {
					result[name] = struct{}{}
					break
				}
			}
		}
	}

	return result
}

func cleanupNonNativeDynamicListMainTables(doc *etree.Document, decisions map[string]objectDecision) bool {
	if doc == nil || len(decisions) == 0 {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	changed := false
	for _, attr := range root.FindElements(".//Attribute") {
		if !isDynamicListAttribute(attr) || !isDynamicListManualQuery(attr) {
			continue
		}

		for _, mainTable := range append([]*etree.Element(nil), attr.FindElements(".//MainTable")...) {
			remove := false
			for _, ref := range metadataReferencesFromValue(strings.TrimSpace(mainTable.Text())) {
				decision, ok := decisions[ref]
				if ok && !decision.Excluded && decision.Belonging != "Native" {
					remove = true
					break
				}
			}
			if !remove {
				continue
			}

			if parent := mainTable.Parent(); parent != nil {
				parent.RemoveChild(mainTable)
				changed = true
			}
		}
	}

	return changed
}

func normalizeManualQueryWithoutMainTable(doc *etree.Document) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	attrsWithoutMainTable := make(map[string]map[string]struct{})
	changed := false

	for _, attr := range root.FindElements(".//Attribute") {
		if !isDynamicListAttribute(attr) || !isDynamicListManualQuery(attr) {
			continue
		}

		attrName := strings.TrimSpace(attr.SelectAttrValue("name", ""))
		if attrName == "" {
			continue
		}
		if strings.TrimSpace(textOfFirst(attr, ".//Settings/MainTable")) != "" {
			continue
		}

		declaredFields := collectDynamicListDeclaredFields(attr)
		if queryEl := attr.FindElement(".//Settings/QueryText"); queryEl != nil {
			if normalized, ok := normalizeManualQuerySelectAliases(queryEl.Text(), declaredFields); ok {
				queryEl.SetText(normalized)
				changed = true
			}
		}

		attrsWithoutMainTable[attrName] = declaredFields
	}

	if len(attrsWithoutMainTable) == 0 {
		return changed
	}

	for _, table := range root.FindElements(".//Table") {
		dataPath := strings.TrimSpace(textOfFirst(table, "./DataPath"))
		if dataPath == "" {
			continue
		}
		if _, ok := attrsWithoutMainTable[dataPath]; !ok {
			continue
		}

		for _, excluded := range append([]*etree.Element(nil), table.FindElements("./CommandSet/ExcludedCommand")...) {
			switch strings.TrimSpace(excluded.Text()) {
			case "Change", "ChangeHistory", "GetURL", "LevelDown", "LevelUp":
				if parent := excluded.Parent(); parent != nil {
					parent.RemoveChild(excluded)
					changed = true
				}
			}
		}

	}

	return changed
}

func normalizeDynamicListChildItemDataPath(value, attrName string, declaredFields map[string]struct{}) (string, bool) {
	value = strings.TrimSpace(value)
	attrName = strings.TrimSpace(attrName)
	if value == "" || attrName == "" || len(declaredFields) == 0 {
		return value, false
	}

	prefix := attrName + "."
	if strings.HasPrefix(value, "~"+prefix) {
		field := strings.TrimPrefix(value, "~"+prefix)
		if _, ok := declaredFields[field]; ok {
			return "~" + field, true
		}
		return value, false
	}

	if !strings.HasPrefix(value, prefix) {
		return value, false
	}

	field := strings.TrimPrefix(value, prefix)
	if _, ok := declaredFields[field]; !ok {
		return value, false
	}

	return field, true
}

var simpleManualQuerySelectRegexp = regexp.MustCompile(`^(\s*)([\p{L}_][\p{L}\p{N}_]*(?:\.[\p{L}_][\p{L}\p{N}_]*)+)(\s*,\s*)$`)

func normalizeManualQuerySelectAliases(queryText string, declaredFields map[string]struct{}) (string, bool) {
	if queryText == "" || len(declaredFields) == 0 {
		return queryText, false
	}

	idx := indexOfQueryFromClause(queryText)
	if idx < 0 {
		return queryText, false
	}

	selectPart := queryText[:idx]
	rest := queryText[idx:]
	lines := strings.Split(selectPart, "\n")
	changed := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.EqualFold(trimmed, "ВЫБРАТЬ") {
			continue
		}
		if strings.Contains(strings.ToUpper(trimmed), " КАК ") {
			continue
		}

		match := simpleManualQuerySelectRegexp.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}

		expr := strings.TrimSpace(match[2])
		lastDot := strings.LastIndex(expr, ".")
		if lastDot < 0 || lastDot+1 >= len(expr) {
			continue
		}

		alias := expr[lastDot+1:]
		if _, ok := declaredFields[alias]; !ok {
			continue
		}

		if nestedFields := collectDirectNestedDeclaredFields(alias, declaredFields); len(nestedFields) > 0 {
			lines[i] = buildNestedManualQuerySelect(match[1], expr, alias, nestedFields, match[3])
			changed = true
			continue
		}

		lines[i] = match[1] + expr + " КАК " + alias + match[3]
		changed = true
	}

	if !changed {
		return queryText, false
	}

	return strings.Join(lines, "\n") + rest, true
}

func collectDirectNestedDeclaredFields(parent string, declaredFields map[string]struct{}) []string {
	if parent == "" || len(declaredFields) == 0 {
		return nil
	}

	prefix := parent + "."
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for field := range declaredFields {
		if !strings.HasPrefix(field, prefix) {
			continue
		}

		tail := strings.TrimPrefix(field, prefix)
		if tail == "" || strings.Contains(tail, ".") {
			continue
		}
		if _, ok := seen[tail]; ok {
			continue
		}
		seen[tail] = struct{}{}
		result = append(result, tail)
	}

	sort.Strings(result)
	preferred := []string{"Ссылка", "НомерСтроки"}
	ordered := make([]string, 0, len(result))
	used := make(map[string]struct{})
	for _, name := range preferred {
		if _, ok := seen[name]; ok {
			ordered = append(ordered, name)
			used[name] = struct{}{}
		}
	}
	for _, name := range result {
		if _, ok := used[name]; ok {
			continue
		}
		ordered = append(ordered, name)
	}

	return ordered
}

func buildNestedManualQuerySelect(indent, expr, alias string, nestedFields []string, suffix string) string {
	var b strings.Builder
	b.WriteString(indent)
	b.WriteString(expr)
	b.WriteString(".(\n")

	childIndent := indent + "\t"
	for i, field := range nestedFields {
		b.WriteString(childIndent)
		b.WriteString(field)
		b.WriteString(" КАК ")
		b.WriteString(field)
		if i < len(nestedFields)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}

	b.WriteString(indent)
	b.WriteString(") КАК ")
	b.WriteString(alias)
	b.WriteString(suffix)
	return b.String()
}

func shouldRemoveNativeFormElement(el *etree.Element, blockedAttributes, nonNativeKeys map[string]struct{}) bool {
	if el == nil {
		return false
	}

	tag := strings.ToLower(localName(el.Tag))
	if tag == "maintable" || tag == "commandname" || tag == "command" || tag == "datapath" || tag == "field" || tag == "item" || tag == "rowpicturedatapath" {
		if subtreeContainsAnyMetadataReference(el, nonNativeKeys) {
			return true
		}
	}
	if !isRemovableFormContainer(tag) {
		return false
	}

	if tag == "attribute" {
		name := strings.TrimSpace(el.SelectAttrValue("name", ""))
		_, blocked := blockedAttributes[name]
		return blocked
	}

	if subtreeContainsFormAttributeReference(el, blockedAttributes) {
		return true
	}

	if subtreeContainsCommonCommandReference(el, nonNativeKeys) {
		return true
	}

	return false
}

func subtreeContainsAnyMetadataReference(el *etree.Element, nonNativeKeys map[string]struct{}) bool {
	if el == nil || len(nonNativeKeys) == 0 {
		return false
	}

	for _, value := range collectElementValues(el) {
		for _, ref := range metadataReferencesFromValue(value) {
			if _, blocked := nonNativeKeys[ref]; blocked {
				return true
			}
		}
	}

	return false
}

func isRemovableFormContainer(tag string) bool {
	switch tag {
	case "", "form", "managedform", "items", "attributes", "commands", "events", "columns", "rows", "pages", "content", "childitems":
		return false
	case "datapath", "field", "item", "command", "commandname", "maintable", "rowpicturedatapath":
		return false
	default:
		return true
	}
}

func subtreeContainsFormAttributeReference(el *etree.Element, blockedAttributes map[string]struct{}) bool {
	if el == nil || len(blockedAttributes) == 0 {
		return false
	}

	for _, value := range collectElementValues(el) {
		for name := range blockedAttributes {
			if containsFormAttributeReference(value, name) {
				return true
			}
		}
	}

	return false
}

func containsFormAttributeReference(value, name string) bool {
	value = strings.TrimSpace(value)
	name = strings.TrimSpace(name)
	if value == "" || name == "" {
		return false
	}

	return value == name ||
		value == "~"+name ||
		strings.HasPrefix(value, name+".") ||
		strings.HasPrefix(value, "~"+name+".")
}

func subtreeContainsCommonCommandReference(el *etree.Element, nonNativeKeys map[string]struct{}) bool {
	if el == nil || len(nonNativeKeys) == 0 {
		return false
	}

	for _, value := range collectElementValues(el) {
		for _, ref := range metadataReferencesFromValue(value) {
			if _, blocked := nonNativeKeys[ref]; !blocked {
				continue
			}
			kind, _ := splitObjectKey(ref)
			if kind == "CommonCommand" {
				return true
			}
		}
	}

	return false
}

func collectElementValues(el *etree.Element) []string {
	if el == nil {
		return nil
	}

	values := make([]string, 0, 8)
	var walk func(*etree.Element)
	walk = func(node *etree.Element) {
		values = append(values, strings.TrimSpace(node.Text()))
		for _, attr := range node.Attr {
			values = append(values, strings.TrimSpace(attr.Value))
		}
		for _, child := range node.ChildElements() {
			walk(child)
		}
	}

	walk(el)
	return values
}

func cleanupUniversalFormNoise(doc *etree.Document) bool {
	root := doc.Root()
	if root == nil {
		return false
	}

	changed := false
	var walk func(parent, grandparent *etree.Element)
	walk = func(parent, grandparent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if shouldRemoveUniversalFormNoise(child) {
				if removesWholeFormNoiseElement(child) && grandparent != nil {
					grandparent.RemoveChild(parent)
					changed = true
					return
				}
				parent.RemoveChild(child)
				changed = true
				continue
			}
			walk(child, parent)
		}
	}

	walk(root, nil)
	return changed
}

func shouldRemoveUniversalFormNoise(el *etree.Element) bool {
	tag := strings.ToLower(localName(el.Tag))
	text := strings.TrimSpace(el.Text())

	if text == "LevelDown" || text == "LevelUp" {
		return tag == "commandname" || tag == "command" || tag == "excludedcommand"
	}

	if tag == "datapath" && text == "НаборКонстант.упо_ИспользоватьРаспределениеЗаработнойПлаты" {
		return true
	}

	return false
}

func removesWholeFormNoiseElement(el *etree.Element) bool {
	tag := strings.ToLower(localName(el.Tag))
	return tag == "commandname" || tag == "command" || tag == "datapath"
}

func collectDefinedFormCommands(root *etree.Element) map[string]struct{} {
	defined := make(map[string]struct{})
	for _, child := range root.ChildElements() {
		if strings.EqualFold(localName(child.Tag), "commands") {
			for _, command := range child.ChildElements() {
				if !strings.EqualFold(localName(command.Tag), "command") {
					continue
				}
				name := strings.TrimSpace(command.SelectAttrValue("name", ""))
				if name != "" {
					defined[name] = struct{}{}
				}
			}
		}
	}
	return defined
}

func shouldRemoveFormCommandReference(el *etree.Element, defined map[string]struct{}, contexts []*FileProcessingContext) bool {
	tag := strings.ToLower(localName(el.Tag))
	text := strings.TrimSpace(el.Text())

	switch tag {
	case "commandname", "command":
		if text == "" {
			return false
		}
		if text == "LevelDown" || text == "LevelUp" {
			return true
		}
		if strings.HasPrefix(text, "Form.Item.") && (strings.Contains(text, ".StandardCommand.LevelDown") || strings.Contains(text, ".StandardCommand.LevelUp")) {
			return true
		}
		if strings.HasPrefix(text, "Form.Command.") {
			commandName := strings.TrimPrefix(text, "Form.Command.")
			_, ok := defined[commandName]
			return !ok
		}
		if isMetadataCommandReference(text) {
			return !roleMetadataTargetExists(text, contexts)
		}
		return false
	case "excludedcommand":
		return text == "LevelDown" || text == "LevelUp"
	default:
		return false
	}
}

func isMetadataCommandReference(value string) bool {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) == 2 && parts[0] == "CommonCommand" {
		return true
	}
	return len(parts) >= 4 && parts[2] == "Command"
}

func isForbiddenStandardCommand(parent, child *etree.Element, targets []string) bool {
	if !subtreeContainsAnyText(child, targets) && !subtreeContainsStandardCommandSuffix(child, []string{"LevelDown", "LevelUp"}) {
		return false
	}

	childTag := strings.ToLower(localName(child.Tag))
	parentTag := strings.ToLower(localName(parent.Tag))

	return strings.Contains(childTag, "standardcommand") ||
		strings.Contains(parentTag, "standardcommand") ||
		childTag == "item"
}

func subtreeContainsStandardCommandSuffix(el *etree.Element, suffixes []string) bool {
	text := strings.TrimSpace(el.Text())
	for _, suffix := range suffixes {
		if strings.HasSuffix(text, suffix) {
			return true
		}
	}

	for _, attr := range el.Attr {
		value := strings.TrimSpace(attr.Value)
		for _, suffix := range suffixes {
			if strings.HasSuffix(value, suffix) {
				return true
			}
		}
	}

	for _, child := range el.ChildElements() {
		if subtreeContainsStandardCommandSuffix(child, suffixes) {
			return true
		}
	}

	return false
}

func subtreeContainsAnyText(el *etree.Element, targets []string) bool {
	text := strings.TrimSpace(el.Text())
	if slices.Contains(targets, text) {
		return true
	}

	for _, attr := range el.Attr {
		if slices.Contains(targets, strings.TrimSpace(attr.Value)) {
			return true
		}
	}

	for _, child := range el.ChildElements() {
		if subtreeContainsAnyText(child, targets) {
			return true
		}
	}

	return false
}

func replaceGUIDsInDoc(doc *etree.Document, replacements map[string]string) bool {
	root := doc.Root()
	if root == nil || len(replacements) == 0 {
		return false
	}

	changed := false

	var walk func(el *etree.Element)
	walk = func(el *etree.Element) {
		tag := strings.ToLower(localName(el.Tag))
		if tag != "classid" {
			replacedText, textChanged := replaceGUIDs(el.Text(), replacements)
			if textChanged {
				el.SetText(replacedText)
				changed = true
			}
		}

		for i, attr := range el.Attr {
			if strings.EqualFold(localName(attr.Key), "ClassId") {
				continue
			}
			replacedValue, valueChanged := replaceGUIDs(attr.Value, replacements)
			if valueChanged {
				el.Attr[i].Value = replacedValue
				changed = true
			}
		}

		for _, child := range el.ChildElements() {
			walk(child)
		}
	}

	walk(root)
	return changed
}

func replaceGUIDs(value string, replacements map[string]string) (string, bool) {
	changed := false
	replaced := guidPattern.ReplaceAllStringFunc(value, func(match string) string {
		newValue, ok := replacements[strings.ToLower(match)]
		if !ok {
			return match
		}
		changed = true
		return newValue
	})

	return replaced, changed
}

func collectMetadataOriginalUUIDs(doc *etree.Document) map[*etree.Element]string {
	result := make(map[*etree.Element]string)
	if doc == nil || doc.Root() == nil {
		return result
	}

	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		if el == nil {
			return
		}

		uuid := strings.TrimSpace(el.SelectAttrValue("uuid", ""))
		if uuid != "" && el.FindElement("./Properties") != nil {
			result[el] = uuid
		}

		for _, child := range el.ChildElements() {
			walk(child)
		}
	}

	walk(doc.Root())
	return result
}

func ensureAdoptedExtendedConfigurationObjects(doc *etree.Document, originalUUIDs map[*etree.Element]string) bool {
	if doc == nil || doc.Root() == nil || len(originalUUIDs) == 0 {
		return false
	}

	changed := false
	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		if el == nil {
			return
		}

		if originalUUID, ok := originalUUIDs[el]; ok {
			properties := el.FindElement("./Properties")
			if properties != nil {
				preserveNative := strings.EqualFold(strings.TrimSpace(el.SelectAttrValue(preserveNativeObjectBelongingAttr, "")), "true")
				if !preserveNative {
					setObjectBelonging(properties, "Adopted")
					if !modifyElement(properties, "ExtendedConfigurationObject", originalUUID) {
						addElement(properties, "ExtendedConfigurationObject", originalUUID)
					}
				} else if deleteElement(properties, "ExtendedConfigurationObject") {
					changed = true
				}
				changed = true
			}
		}
		for i := len(el.Attr) - 1; i >= 0; i-- {
			if el.Attr[i].Key == preserveNativeObjectBelongingAttr {
				el.Attr = append(el.Attr[:i], el.Attr[i+1:]...)
				changed = true
			}
		}

		for _, child := range el.ChildElements() {
			walk(child)
		}
	}

	walk(doc.Root())
	return changed
}

func verifyNoOldGUIDs(contexts []*FileProcessingContext, replacements map[string]string, excludedPaths map[string]struct{}) error {
	if len(replacements) == 0 {
		return nil
	}

	for _, ctx := range contexts {
		if _, excluded := excludedPaths[ctx.Path]; excluded || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}

		if leftovers := findOldGUIDs(ctx.Doc.Root(), replacements, false); len(leftovers) > 0 {
			return fmt.Errorf("в файле %s остались старые идентификаторы Adopted-объектов: %s", ctx.RelPath, strings.Join(leftovers, ", "))
		}
	}

	return nil
}

func findOldGUIDs(el *etree.Element, replacements map[string]string, inClassID bool) []string {
	found := make(map[string]struct{})

	var walk func(*etree.Element, bool)
	walk = func(node *etree.Element, inClass bool) {
		tagIsClassID := strings.EqualFold(localName(node.Tag), "ClassId")
		currentClass := inClass || tagIsClassID

		if !currentClass && !strings.EqualFold(localName(node.Tag), "ExtendedConfigurationObject") {
			for _, guid := range extractGUIDs(node.Text()) {
				if _, ok := replacements[guid]; ok {
					found[guid] = struct{}{}
				}
			}
		}

		for _, attr := range node.Attr {
			if strings.EqualFold(localName(attr.Key), "ClassId") {
				continue
			}
			for _, guid := range extractGUIDs(attr.Value) {
				if _, ok := replacements[guid]; ok {
					found[guid] = struct{}{}
				}
			}
		}

		for _, child := range node.ChildElements() {
			walk(child, currentClass)
		}
	}

	walk(el, inClassID)

	if len(found) == 0 {
		return nil
	}

	result := make([]string, 0, len(found))
	for guid := range found {
		result = append(result, guid)
	}
	slices.Sort(result)
	return result
}

func detectOwner(relPath string, doc *etree.Document) (kind, name, key string) {
	normalized := filepath.ToSlash(relPath)
	if normalized == mainFile {
		return "Configuration", "Configuration", "Configuration"
	}

	parts := strings.Split(normalized, "/")
	if len(parts) == 0 {
		return "", "", ""
	}

	kind = metadataKinds[parts[0]]
	if kind == "" {
		return "", "", ""
	}

	if kind == "Subsystem" {
		chain := subsystemChain(normalized)
		if len(chain) > 0 {
			name = chain[len(chain)-1]
			key = subsystemObjectKey(chain)
		}
	} else if len(parts) > 1 {
		name = ownerNameFromPath(parts)
	}

	if name == "" {
		name = propertyName(findProperties(doc))
	}
	if name == "" {
		return kind, "", ""
	}

	if key != "" {
		return kind, name, key
	}

	return kind, name, kind + "." + name
}

func ownerNameFromPath(parts []string) string {
	if len(parts) < 2 {
		return ""
	}

	second := parts[1]
	if strings.HasSuffix(strings.ToLower(second), ".xml") {
		return strings.TrimSuffix(second, filepath.Ext(second))
	}

	return second
}

func subsystemChain(relPath string) []string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) < 2 {
		return nil
	}

	chain := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if strings.EqualFold(part, "Subsystems") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(part), ".xml") {
			part = strings.TrimSuffix(part, filepath.Ext(part))
		}
		if part != "" {
			chain = append(chain, part)
		}
	}

	return chain
}

func subsystemObjectKey(chain []string) string {
	if len(chain) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("Subsystem.")
	for i, part := range chain {
		if i > 0 {
			builder.WriteString(".Subsystem.")
		}
		builder.WriteString(part)
	}

	return builder.String()
}

func hasNativePrefix(name string, prefixes []string) bool {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	if normalizedName == "" {
		return false
	}

	for _, prefix := range prefixes {
		normalizedPrefix := strings.ToLower(strings.TrimSpace(prefix))
		if normalizedPrefix == "" {
			continue
		}

		candidates := []string{
			normalizedPrefix,
			"удалить_" + normalizedPrefix,
			"удалить" + normalizedPrefix,
		}

		for _, candidate := range candidates {
			if strings.HasPrefix(normalizedName, candidate) {
				return true
			}
		}
	}

	return false
}

func referencedSubsystems(properties *etree.Element) map[string]struct{} {
	result := make(map[string]struct{})
	if properties == nil {
		return result
	}

	for _, node := range properties.FindElements(".//Subsystems//*") {
		text := normalizeReferenceName(node.Text())
		if text != "" {
			result[text] = struct{}{}
		}
	}

	for _, node := range properties.FindElements(".//Subsystem") {
		text := normalizeReferenceName(node.Text())
		if text != "" {
			result[text] = struct{}{}
		}
	}

	return result
}

func normalizeReferenceName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || guidPattern.MatchString(trimmed) {
		return ""
	}

	trimmed = strings.TrimPrefix(trimmed, "Subsystem.")
	trimmed = strings.TrimPrefix(trimmed, "Подсистема.")
	trimmed = strings.Trim(trimmed, ".")
	if trimmed == "" {
		return ""
	}

	return trimmed
}

func belongsToExcludedSubsystem(ctx *FileProcessingContext, excludedSubsystems []string) bool {
	if len(excludedSubsystems) == 0 {
		return false
	}

	if ctx.OwnerKind == "Subsystem" {
		return isExcludedSubsystemChain(subsystemChain(ctx.RelPath), excludedSubsystems)
	}

	return containsExcludedSubsystemReference(referencedSubsystems(ctx.Properties), excludedSubsystems)
}

func isExcludedSubsystemChain(chain []string, excludedSubsystems []string) bool {
	if len(chain) == 0 || len(excludedSubsystems) == 0 {
		return false
	}

	fullPath := strings.Join(chain, ".")
	for _, excluded := range excludedSubsystems {
		excluded = strings.Trim(strings.TrimSpace(excluded), ".")
		if excluded == "" {
			continue
		}
		if fullPath == excluded || strings.HasPrefix(fullPath, excluded+".") {
			return true
		}
	}

	return false
}

func containsExcludedSubsystemReference(references map[string]struct{}, excludedSubsystems []string) bool {
	if len(references) == 0 || len(excludedSubsystems) == 0 {
		return false
	}

	for reference := range references {
		for _, excluded := range excludedSubsystems {
			excluded = strings.Trim(strings.TrimSpace(excluded), ".")
			if excluded == "" {
				continue
			}
			if reference == excluded || strings.HasPrefix(reference, excluded+".") {
				return true
			}
		}
	}

	return false
}

func containsAny(values map[string]struct{}, targets map[string]struct{}) bool {
	for value := range values {
		if _, ok := targets[value]; ok {
			return true
		}
	}
	return false
}

func containsName(values map[string]struct{}, name string) bool {
	_, ok := values[name]
	return ok
}

func containsObject(objects []string, target string) bool {
	for _, object := range objects {
		if strings.EqualFold(strings.TrimSpace(object), target) {
			return true
		}
	}
	return false
}

func getInfo(properties *etree.Element) {
	dumpInfo := config.GetDumpInfo()

	currentElem := properties.FindElement("Name")
	if currentElem != nil {
		dumpInfo.SetConfigName(currentElem.Text())
	}
	currentElem = properties.FindElement("Version")
	if currentElem != nil {
		dumpInfo.SetVersion(currentElem.Text())
	}
}

func normalizeRootConfiguration(properties *etree.Element, cfg *config.Configuration) bool {
	if properties == nil {
		return false
	}

	name := strings.TrimSpace(cfg.Extension)
	if name == "" {
		name = strings.TrimSpace(textOf(properties, "Name"))
	}
	if name == "" {
		name = "Extension"
	}

	compatibility := strings.TrimSpace(textOf(properties, "ConfigurationExtensionCompatibilityMode"))
	if compatibility == "" {
		compatibility = platformCompatibilityMode(cfg.PlatformVersion)
	}
	if compatibility == "" {
		compatibility = "Version8_3_27"
	}

	removeAllChildren(properties)

	addSimpleElement(properties, "ObjectBelonging", "Adopted")
	addSimpleElement(properties, "Name", name)
	addSimpleElement(properties, "Synonym", "")
	setRussianSynonym(properties.FindElement("Synonym"), name)
	addSimpleElement(properties, "Comment", "")
	addSimpleElement(properties, "ConfigurationExtensionPurpose", "Customization")
	addSimpleElement(properties, "KeepMappingToExtendedConfigurationObjectsByIDs", "true")
	addSimpleElement(properties, "NamePrefix", cfg.Prefix)
	addSimpleElement(properties, "ConfigurationExtensionCompatibilityMode", compatibility)
	addSimpleElement(properties, "DefaultRunMode", "ManagedApplication")
	addUsePurposes(properties)
	addSimpleElement(properties, "ScriptVariant", "Russian")
	addSimpleElement(properties, "Vendor", "")
	addSimpleElement(properties, "Version", "")
	addSimpleElement(properties, "Caption", "")
	addSimpleElement(properties, "ShortCaption", "")
	addSimpleElement(properties, "BriefInformation", "")
	addSimpleElement(properties, "DetailedInformation", "")
	addSimpleElement(properties, "Copyright", "")
	addSimpleElement(properties, "VendorInformationAddress", "")
	addSimpleElement(properties, "ConfigurationInformationAddress", "")

	return true
}

func normalizeRootConfigurationInternalInfo(doc *etree.Document) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	configuration := root.SelectElement("Configuration")
	if configuration == nil {
		return false
	}

	internalInfo := configuration.SelectElement("InternalInfo")
	if internalInfo == nil {
		return false
	}

	requiredStates := []string{
		"ManagedApplicationModule",
		"SessionModule",
		"ExternalConnectionModule",
		"OrdinaryApplicationModule",
		"CommandInterface",
		"HomePageWorkArea",
		"MainSectionCommandInterface",
		"MainSectionPicture",
		"Logo",
		"Splash",
	}

	changed := false
	for _, property := range requiredStates {
		changed = ensureRootPropertyState(internalInfo, property) || changed
	}

	return changed
}

func ensureRootPropertyState(internalInfo *etree.Element, property string) bool {
	if internalInfo == nil || property == "" {
		return false
	}

	for _, child := range internalInfo.ChildElements() {
		if !strings.EqualFold(localName(child.Tag), "PropertyState") {
			continue
		}
		currentProperty := child.SelectElement("xr:Property")
		if currentProperty != nil && strings.EqualFold(strings.TrimSpace(currentProperty.Text()), property) {
			return false
		}
	}

	propertyState := internalInfo.CreateElement("xr:PropertyState")
	prop := propertyState.CreateElement("xr:Property")
	prop.SetText(property)
	state := propertyState.CreateElement("xr:State")
	state.SetText("Extended")
	return true
}

func normalizeConfigDumpInfoRootNames(doc *etree.Document, oldName, newName string) bool {
	if doc == nil {
		return false
	}

	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" || oldName == newName {
		return false
	}

	oldPrefix := "Configuration." + oldName
	newPrefix := "Configuration." + newName

	changed := false
	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		if strings.EqualFold(localName(el.Tag), "Metadata") {
			current := strings.TrimSpace(el.SelectAttrValue("name", ""))
			if current == "" {
				goto children
			}
			if strings.HasPrefix(current, oldPrefix) {
				updated := newPrefix + strings.TrimPrefix(current, oldPrefix)
				setAttrValue(el, "name", updated)
				changed = true
			}
		}

	children:
		for _, child := range el.ChildElements() {
			walk(child)
		}
	}

	root := doc.Root()
	if root != nil {
		walk(root)
	}

	return changed
}

func cleanupConfigDumpInfoRootServiceEntries(doc *etree.Document, extension string) bool {
	if doc == nil {
		return false
	}

	extension = strings.TrimSpace(extension)
	if extension == "" {
		return false
	}

	targets := map[string]struct{}{
		"Configuration." + extension + ".ClientApplicationInterface": {},
		"Configuration." + extension + ".ParentConfigurations":       {},
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	changed := false
	var walk func(*etree.Element)
	walk = func(parent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if strings.EqualFold(localName(child.Tag), "Metadata") {
				name := strings.TrimSpace(child.SelectAttrValue("name", ""))
				if _, ok := targets[name]; ok {
					parent.RemoveChild(child)
					changed = true
					continue
				}
			}
			walk(child)
		}
	}

	walk(root)
	return changed
}

func cleanupRoleConfigurationRights(doc *etree.Document, configurationName string) bool {
	if doc == nil {
		return false
	}

	configurationName = strings.TrimSpace(configurationName)
	if configurationName == "" {
		return false
	}

	root := doc.Root()
	if root == nil || !strings.EqualFold(localName(root.Tag), "Rights") {
		return false
	}

	target := "Configuration." + configurationName
	changed := false
	for _, object := range append([]*etree.Element(nil), root.ChildElements()...) {
		if !strings.EqualFold(localName(object.Tag), "object") {
			continue
		}
		name := strings.TrimSpace(textOf(object, "name"))
		if name != target {
			continue
		}
		root.RemoveChild(object)
		changed = true
	}

	return changed
}

func cleanupRoleDanglingMetadataRights(doc *etree.Document, contexts []*FileProcessingContext) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil || !strings.EqualFold(localName(root.Tag), "Rights") {
		return false
	}

	changed := false
	for _, object := range append([]*etree.Element(nil), root.ChildElements()...) {
		if !strings.EqualFold(localName(object.Tag), "object") {
			continue
		}
		name := strings.TrimSpace(textOf(object, "name"))
		if name == "" {
			continue
		}
		if !roleMetadataTargetExists(name, contexts) {
			root.RemoveChild(object)
			changed = true
		}
	}

	return changed
}

func cleanupRoleExcludedMetadataRights(doc *etree.Document, decisions map[string]objectDecision) bool {
	if doc == nil || len(decisions) == 0 {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	changed := false
	for _, object := range append([]*etree.Element(nil), root.ChildElements()...) {
		if !strings.EqualFold(localName(object.Tag), "object") {
			continue
		}
		name := strings.TrimSpace(textOf(object, "name"))
		if name == "" {
			continue
		}
		if roleMetadataTargetExcluded(name, decisions) {
			root.RemoveChild(object)
			changed = true
		}
	}

	return changed
}

func roleMetadataTargetExists(name string, contexts []*FileProcessingContext) bool {
	parts := strings.Split(strings.TrimSpace(name), ".")
	if len(parts) < 2 {
		return true
	}

	topKey := parts[0] + "." + parts[1]
	ctx := findContextByOwnerKey(contexts, topKey)
	if ctx == nil || ctx.Doc == nil || !ctx.Metadata {
		return true
	}

	if len(parts) == 2 {
		return true
	}

	root := ctx.Doc.Root()
	if root == nil {
		return false
	}

	target := root
	if strings.EqualFold(localName(root.Tag), "MetaDataObject") {
		children := root.ChildElements()
		if len(children) > 0 {
			target = children[0]
		}
	}

	if metadataPathExistsInElement(target, parts[2:]) {
		return true
	}

	return metadataPathExistsInFilesystem(ctx.Path, parts[2:])
}

func metadataPathExistsInFilesystem(topLevelPath string, parts []string) bool {
	if topLevelPath == "" || len(parts) == 0 {
		return false
	}

	objectDir := strings.TrimSuffix(topLevelPath, filepath.Ext(topLevelPath))
	if objectDir == "" {
		return false
	}

	switch {
	case len(parts) == 2 && strings.EqualFold(parts[0], "Command"):
		commandName := strings.TrimSpace(parts[1])
		if commandName == "" {
			return false
		}
		commandDir := filepath.Join(objectDir, "Commands", commandName)
		if _, err := os.Stat(commandDir); err == nil {
			return true
		}
		commandModule := filepath.Join(commandDir, "Ext", "CommandModule.bsl")
		_, err := os.Stat(commandModule)
		return err == nil
	case len(parts) == 3 && strings.EqualFold(parts[0], "Command") && strings.EqualFold(parts[2], "CommandModule"):
		commandName := strings.TrimSpace(parts[1])
		if commandName == "" {
			return false
		}
		commandModule := filepath.Join(objectDir, "Commands", commandName, "Ext", "CommandModule.bsl")
		_, err := os.Stat(commandModule)
		return err == nil
	case len(parts) == 1 && strings.EqualFold(parts[0], "CommandModule"):
		commandModule := filepath.Join(objectDir, "Ext", "CommandModule.bsl")
		_, err := os.Stat(commandModule)
		return err == nil
	}

	return false
}

func roleMetadataTargetExcluded(name string, decisions map[string]objectDecision) bool {
	parts := strings.Split(strings.TrimSpace(name), ".")
	if len(parts) < 2 {
		return false
	}

	topKey := parts[0] + "." + parts[1]
	decision, ok := decisions[topKey]
	return ok && decision.Excluded
}

func metadataPathExistsInElement(parent *etree.Element, parts []string) bool {
	if parent == nil {
		return false
	}
	if len(parts) == 0 {
		return true
	}
	if len(parts) < 2 {
		return false
	}

	kind := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if kind == "" || name == "" {
		return false
	}

	childObjects := parent.FindElement("./ChildObjects")
	if childObjects == nil {
		return false
	}

	for _, child := range childObjects.ChildElements() {
		if !strings.EqualFold(localName(child.Tag), kind) {
			continue
		}
		if strings.TrimSpace(textOf(child.FindElement("./Properties"), "Name")) != name {
			continue
		}
		return metadataPathExistsInElement(child, parts[2:])
	}

	return false
}

func cleanupDanglingCommandInterfaceCommands(doc *etree.Document, contexts []*FileProcessingContext) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil || !strings.EqualFold(localName(root.Tag), "CommandInterface") {
		return false
	}

	changed := false
	var walk func(*etree.Element)
	walk = func(parent *etree.Element) {
		for _, child := range append([]*etree.Element(nil), parent.ChildElements()...) {
			if strings.EqualFold(localName(child.Tag), "Command") {
				name := strings.TrimSpace(child.SelectAttrValue("name", ""))
				if name != "" && !strings.Contains(name, ".StandardCommand.") && !roleMetadataTargetExists(name, contexts) {
					parent.RemoveChild(child)
					changed = true
					continue
				}
			}
			walk(child)
		}
	}

	walk(root)
	return changed
}

func cleanupConfigDumpInfoNonNativeChildren(doc *etree.Document, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	if doc == nil || len(decisions) == 0 {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	nonNativePrefixes := make(map[string]struct{})
	for key, decision := range decisions {
		if decision.Excluded || decision.Belonging == "Native" {
			continue
		}
		nonNativePrefixes[key+"."] = struct{}{}
	}

	changed := false
	var walk func(*etree.Element)
	walk = func(parent *etree.Element) {
		for _, child := range append([]*etree.Element(nil), parent.ChildElements()...) {
			if !strings.EqualFold(localName(child.Tag), "Metadata") {
				walk(child)
				continue
			}

			name := strings.TrimSpace(child.SelectAttrValue("name", ""))
			removed := false
			for prefix := range nonNativePrefixes {
				if strings.HasPrefix(name, prefix) {
					if _, ok := configDumpInfoTopLevelKey(name); !ok {
						if isDisallowedAdoptedModuleMetadata(name) || !roleMetadataTargetExists(name, contexts) {
							parent.RemoveChild(child)
							changed = true
							removed = true
							break
						}
					}
				}
			}
			if removed {
				continue
			}

			key, ok := configDumpInfoTopLevelKey(name)
			if ok {
				if decision, exists := decisions[key]; exists && !decision.Excluded && decision.Belonging != "Native" {
					for _, nested := range append([]*etree.Element(nil), child.ChildElements()...) {
						nestedName := strings.TrimSpace(nested.SelectAttrValue("name", ""))
						if strings.EqualFold(localName(nested.Tag), "Metadata") &&
							(isDisallowedAdoptedModuleMetadata(nestedName) || !roleMetadataTargetExists(nestedName, contexts)) {
							child.RemoveChild(nested)
							changed = true
						}
					}
					continue
				}
			}

			walk(child)
		}
	}

	walk(root)
	return changed
}

func isDisallowedAdoptedModuleMetadata(name string) bool {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, "CommonModule.") && strings.HasSuffix(name, ".Module") {
		return true
	}
	if strings.HasSuffix(name, ".CommandModule") {
		return true
	}
	return strings.HasSuffix(name, ".ManagerModule") ||
		strings.HasSuffix(name, ".ObjectModule") ||
		strings.HasSuffix(name, ".ValueManagerModule")
}

func collectAdoptedCommonModuleModulePaths(root string, decisions map[string]objectDecision, excludedPaths map[string]struct{}) {
	if root == "" || len(decisions) == 0 || excludedPaths == nil {
		return
	}

	for key, decision := range decisions {
		if decision.Excluded || decision.Belonging == "Native" {
			continue
		}

		kind, name := splitObjectKey(key)
		if kind != "CommonModule" || name == "" {
			continue
		}

		modulePath := filepath.Join(root, dirCommonModules, name, "Ext", "Module.bsl")
		if _, err := os.Stat(modulePath); err == nil {
			excludedPaths[modulePath] = struct{}{}
		}
	}
}

func collectAdoptedCommandModulePaths(contexts []*FileProcessingContext, decisions map[string]objectDecision, excludedPaths map[string]struct{}) {
	if len(contexts) == 0 || len(decisions) == 0 || excludedPaths == nil {
		return
	}

	for _, ctx := range contexts {
		if ctx == nil || !ctx.Metadata || !isTopLevelMetadataFile(ctx) {
			continue
		}

		decision, ok := decisions[ctx.OwnerKey]
		if !ok || decision.Excluded || decision.Belonging == "Native" {
			continue
		}

		objectDir := strings.TrimSuffix(ctx.Path, filepath.Ext(ctx.Path))
		if objectDir == "" {
			continue
		}

		topLevelCommandModulePath := filepath.Join(objectDir, "Ext", "CommandModule.bsl")
		if _, err := os.Stat(topLevelCommandModulePath); err == nil {
			excludedPaths[topLevelCommandModulePath] = struct{}{}
		}

		childCommandModulePaths, err := filepath.Glob(filepath.Join(objectDir, "Commands", "*", "Ext", "CommandModule.bsl"))
		if err != nil {
			continue
		}
		for _, path := range childCommandModulePaths {
			excludedPaths[path] = struct{}{}
		}
	}
}

func configDumpInfoTopLevelKey(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return "", false
	}

	kind := strings.TrimSpace(parts[0])
	objectName := strings.TrimSpace(parts[1])
	if kind == "" || objectName == "" {
		return "", false
	}

	if _, ok := configurationChildObjectKinds[kind]; !ok {
		return "", false
	}

	return kind + "." + objectName, true
}

func isRootServiceFile(ctx *FileProcessingContext) bool {
	if ctx == nil {
		return false
	}

	return strings.EqualFold(ctx.FileName, "ClientApplicationInterface.xml")
}

func cleanupRootServiceArtifacts(dir string) error {
	if dir == "" {
		return nil
	}

	extDir := filepath.Join(dir, "Ext")
	if err := os.Remove(filepath.Join(extDir, "ClientApplicationInterface.xml")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления ClientApplicationInterface.xml: %w", err)
	}
	if err := os.Remove(filepath.Join(extDir, "ParentConfigurations.bin")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка удаления ParentConfigurations.bin: %w", err)
	}
	if err := os.RemoveAll(filepath.Join(extDir, "ParentConfigurations")); err != nil {
		return fmt.Errorf("ошибка удаления ParentConfigurations: %w", err)
	}

	return nil
}

func normalizeLanguageObject(properties *etree.Element) bool {
	if properties == nil {
		return false
	}

	return deleteElement(properties, "ExtendedConfigurationObject")
}

func cleanupAdoptedObjectFormReferences(properties *etree.Element) bool {
	if properties == nil {
		return false
	}

	changed := false
	for _, name := range []string{
		"DefaultObjectForm",
		"DefaultListForm",
		"DefaultChoiceForm",
		"AuxiliaryObjectForm",
		"AuxiliaryListForm",
		"AuxiliaryChoiceForm",
	} {
		if deleteElement(properties, name) {
			changed = true
		}
	}

	return changed
}

func normalizeTruncatedMetadataStub(doc *etree.Document, properties *etree.Element) bool {
	if doc == nil || properties == nil {
		return false
	}

	name := strings.TrimSpace(textOf(properties, "Name"))
	comment := textOf(properties, "Comment")
	synonymValue := extractRussianSynonym(properties.FindElement("Synonym"))

	removeAllChildren(properties)
	addSimpleElement(properties, "Name", name)
	synonym := addSimpleElement(properties, "Synonym", "")
	if synonymValue != "" {
		setRussianSynonym(synonym, synonymValue)
	}
	addSimpleElement(properties, "Comment", comment)
	addSimpleElement(properties, "ObjectBelonging", "Adopted")

	objectElement := properties.Parent()
	if objectElement != nil {
		hadChildObjects := hasDirectChildByLocalName(objectElement, "ChildObjects")
		for _, child := range append([]etree.Token(nil), objectElement.Child...) {
			el, ok := child.(*etree.Element)
			if !ok {
				continue
			}
			tag := localName(el.Tag)
			if tag == "InternalInfo" || tag == "Properties" {
				continue
			}
			objectElement.RemoveChild(el)
		}
		if hadChildObjects && !hasDirectChildByLocalName(objectElement, "ChildObjects") {
			objectElement.CreateElement("ChildObjects")
		}
	}

	return true
}

func normalizeAdoptedStubExtFormComposition(doc *etree.Document, contract formDynamicListContract, retainedCommands map[string]struct{}) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	target := root
	if strings.EqualFold(localName(root.Tag), "MetaDataObject") {
		children := root.ChildElements()
		if len(children) > 0 {
			target = children[0]
		}
	}

	allowedPaths := collectRequiredFormStubPaths(target, contract.RequiredFields)
	changed := removeStandardAttributesFromProperties(target) || false
	if localName(target.Tag) == "Catalog" {
		changed = normalizeFormStubProperties(target, true) || changed
	}

	for _, child := range append([]etree.Token(nil), target.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		tag := localName(el.Tag)
		switch tag {
		case "InternalInfo", "Properties":
			continue
		case "ChildObjects":
			if normalizeFormStubChildObjects(el, allowedPaths, "", retainedCommands) {
				changed = true
			}
			continue
		default:
			target.RemoveChild(el)
			changed = true
		}
	}

	return changed
}

func mergeAdoptedStubMetaDataIntoFormContract(contract formDynamicListContract, rule adoptedStubMetaDataRule) formDynamicListContract {
	merged := formDynamicListContract{
		RequiredFields:   make(map[string]struct{}, len(contract.RequiredFields)+len(rule.NativeAttributes)),
		QueryAliases:     make(map[string]struct{}, len(contract.QueryAliases)),
		RequiredCommands: make(map[string]struct{}, len(contract.RequiredCommands)),
	}

	for field := range contract.RequiredFields {
		merged.RequiredFields[field] = struct{}{}
	}
	for alias := range contract.QueryAliases {
		merged.QueryAliases[alias] = struct{}{}
	}
	for command := range contract.RequiredCommands {
		merged.RequiredCommands[command] = struct{}{}
	}

	for name := range rule.NativeAttributes {
		if strings.TrimSpace(name) == "" {
			continue
		}
		merged.RequiredFields[name] = struct{}{}
	}

	for sectionName, attrs := range rule.NativeTabularSections {
		sectionName = strings.TrimSpace(sectionName)
		if sectionName == "" {
			continue
		}
		merged.RequiredFields[sectionName] = struct{}{}
		for attrName := range attrs {
			attrName = strings.TrimSpace(attrName)
			if attrName == "" {
				continue
			}
			merged.RequiredFields[sectionName+"."+attrName] = struct{}{}
		}
	}

	return merged
}

func normalizeFormStubProperties(target *etree.Element, topLevel bool) bool {
	if target == nil {
		return false
	}

	tag := localName(target.Tag)
	if topLevel && tag != "Catalog" {
		return false
	}

	changed := normalizeSingleFormStubProperties(target, topLevel)

	for _, childObjects := range target.FindElements("./ChildObjects") {
		for _, child := range childObjects.ChildElements() {
			if normalizeFormStubProperties(child, false) {
				changed = true
			}
		}
	}

	return changed
}

func normalizeSingleFormStubProperties(target *etree.Element, topLevel bool) bool {
	if target == nil {
		return false
	}

	properties := target.FindElement("./Properties")
	if properties == nil {
		return false
	}

	tag := localName(target.Tag)
	name := textOf(properties, "Name")
	comment := textOf(properties, "Comment")
	objectBelonging := textOf(properties, "ObjectBelonging")
	if strings.TrimSpace(objectBelonging) == "" {
		objectBelonging = "Adopted"
	}
	extendedConfigurationObject := textOf(properties, "ExtendedConfigurationObject")

	var typeElement *etree.Element
	if originalType := properties.FindElement("./Type"); originalType != nil {
		typeElement = originalType.Copy()
	}
	hierarchical := textOf(properties, "Hierarchical")
	descriptionLength := textOf(properties, "DescriptionLength")

	removeAllChildren(properties)

	addSimpleElement(properties, "ObjectBelonging", objectBelonging)
	addSimpleElement(properties, "Name", name)
	addSimpleElement(properties, "Comment", comment)
	if strings.TrimSpace(extendedConfigurationObject) != "" {
		addSimpleElement(properties, "ExtendedConfigurationObject", extendedConfigurationObject)
	}

	if topLevel && tag == "Catalog" {
		addSimpleElement(properties, "Hierarchical", hierarchical)
		addSimpleElement(properties, "DescriptionLength", descriptionLength)
	}

	if tag == "Attribute" && typeElement != nil {
		properties.AddChild(typeElement)
	}

	return true
}

func removeStandardAttributesFromProperties(target *etree.Element) bool {
	if target == nil {
		return false
	}

	properties := target.FindElement("./Properties")
	if properties == nil {
		return false
	}

	changed := false
	for _, child := range append([]etree.Token(nil), properties.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		if localName(el.Tag) != "StandardAttributes" {
			continue
		}
		properties.RemoveChild(el)
		changed = true
	}

	return changed
}

func collectRequiredFormStubPaths(target *etree.Element, requiredFields map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	if target == nil || len(requiredFields) == 0 {
		return result
	}

	standard := collectStandardAttributeNames(target)
	for field := range requiredFields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		parts := strings.Split(field, ".")
		if len(parts) == 0 {
			continue
		}

		if _, isStandard := standard[parts[0]]; isStandard {
			continue
		}

		for idx := range parts {
			prefix := strings.Join(parts[:idx+1], ".")
			if prefix == "" {
				continue
			}
			result[prefix] = struct{}{}
		}
	}

	return result
}

func collectStandardAttributeNames(target *etree.Element) map[string]struct{} {
	result := make(map[string]struct{})
	if target == nil {
		return result
	}

	properties := target.FindElement("./Properties")
	if properties == nil {
		return result
	}

	for _, attr := range properties.FindElements(".//StandardAttributes/*") {
		name := strings.TrimSpace(attr.SelectAttrValue("name", ""))
		if name != "" {
			result[name] = struct{}{}
		}
	}

	return result
}

func normalizeFormStubChildObjects(childObjects *etree.Element, allowedPaths map[string]struct{}, parentPath string, retainedCommands map[string]struct{}) bool {
	if childObjects == nil {
		return false
	}

	changed := false
	for _, child := range append([]etree.Token(nil), childObjects.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		tag := localName(el.Tag)
		if tag == "Command" {
			if _, keep := retainedCommands[metadataChildName(el)]; keep {
				continue
			}
			childObjects.RemoveChild(el)
			changed = true
			continue
		}
		if tag != "Attribute" && tag != "TabularSection" {
			childObjects.RemoveChild(el)
			changed = true
			continue
		}

		name := strings.TrimSpace(textOf(el.FindElement("./Properties"), "Name"))
		currentPath := name
		if parentPath != "" {
			currentPath = parentPath + "." + name
		}

		if _, keep := allowedPaths[currentPath]; !keep {
			childObjects.RemoveChild(el)
			changed = true
			continue
		}

		if tag == "TabularSection" {
			nestedChildObjects := el.FindElement("./ChildObjects")
			if nestedChildObjects != nil {
				if normalizeFormStubChildObjects(nestedChildObjects, allowedPaths, currentPath, retainedCommands) {
					changed = true
				}
			}
		}
	}

	return changed
}

func metadataTargetElement(doc *etree.Document) *etree.Element {
	if doc == nil {
		return nil
	}

	root := doc.Root()
	if root == nil {
		return nil
	}

	if strings.EqualFold(localName(root.Tag), "MetaDataObject") {
		children := root.ChildElements()
		if len(children) > 0 {
			return children[0]
		}
	}

	return root
}

func normalizeAdoptedObjectComposition(doc *etree.Document, ownerKind string, retainedCommands map[string]struct{}) bool {
	if doc == nil {
		return false
	}

	target := metadataTargetElement(doc)
	if target == nil {
		return false
	}

	hadChildObjects := hasDirectChildByLocalName(target, "ChildObjects")
	changed := removeStandardAttributesFromProperties(target)
	for _, child := range append([]etree.Token(nil), target.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		tag := localName(el.Tag)
		if tag == "InternalInfo" || tag == "Properties" {
			continue
		}
		if tag == "ChildObjects" && len(retainedCommands) > 0 {
			if normalizeRetainedAdoptedCommandChildObjects(el, retainedCommands) {
				changed = true
			}
			continue
		}
		if ownerKind == "Subsystem" && tag == "ChildObjects" {
			continue
		}
		target.RemoveChild(el)
		changed = true
	}

	if hadChildObjects && !hasDirectChildByLocalName(target, "ChildObjects") {
		target.CreateElement("ChildObjects")
		changed = true
	}

	return changed
}

func normalizeAdoptedStubMetaDataComposition(doc *etree.Document, ownerKind string, rule adoptedStubMetaDataRule, retainedCommands map[string]struct{}) bool {
	if doc == nil {
		return false
	}

	target := metadataTargetElement(doc)
	if target == nil {
		return false
	}

	hadChildObjects := hasDirectChildByLocalName(target, "ChildObjects")
	changed := removeStandardAttributesFromProperties(target)
	var childObjects *etree.Element

	for _, child := range append([]etree.Token(nil), target.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		tag := localName(el.Tag)
		switch tag {
		case "InternalInfo", "Properties":
			continue
		case "ChildObjects":
			childObjects = el
		default:
			if ownerKind == "Subsystem" && tag == "ChildObjects" {
				childObjects = el
				continue
			}
			target.RemoveChild(el)
			changed = true
		}
	}

	if childObjects != nil {
		if normalizeAdoptedStubMetaDataChildObjects(childObjects, rule, retainedCommands) {
			changed = true
		}
	} else if hadChildObjects {
		target.CreateElement("ChildObjects")
		changed = true
	}

	if hadChildObjects && !hasDirectChildByLocalName(target, "ChildObjects") {
		target.CreateElement("ChildObjects")
		changed = true
	}

	return changed
}

func normalizeAdoptedStubMetaDataChildObjects(childObjects *etree.Element, rule adoptedStubMetaDataRule, retainedCommands map[string]struct{}) bool {
	if childObjects == nil {
		return false
	}

	changed := false
	for _, child := range append([]etree.Token(nil), childObjects.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}

		tag := localName(el.Tag)
		name := metadataChildName(el)
		switch tag {
		case "Attribute":
			if _, keep := rule.NativeAttributes[name]; keep {
				if normalizeAdoptedStubMetaDataRetainedChild(el) {
					changed = true
				}
				continue
			}
		case "TabularSection":
			allowedAttrs, keep := rule.NativeTabularSections[name]
			if keep {
				if normalizeAdoptedStubMetaDataRetainedChild(el) {
					changed = true
				}
				if normalizeAdoptedStubMetaDataTabularSection(el, allowedAttrs) {
					changed = true
				}
				continue
			}
		case "Command":
			if _, keep := retainedCommands[name]; keep {
				continue
			}
		}

		childObjects.RemoveChild(el)
		changed = true
	}

	return changed
}

func normalizeAdoptedStubMetaDataTabularSection(section *etree.Element, allowedAttrs map[string]struct{}) bool {
	if section == nil {
		return false
	}

	changed := false
	for _, child := range append([]etree.Token(nil), section.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}

		tag := localName(el.Tag)
		switch tag {
		case "InternalInfo", "Properties":
			continue
		case "ChildObjects":
			if normalizeAdoptedStubMetaDataSectionChildObjects(el, allowedAttrs) {
				changed = true
			}
			continue
		default:
			section.RemoveChild(el)
			changed = true
		}
	}

	if childObjects := section.FindElement("./ChildObjects"); childObjects == nil {
		section.CreateElement("ChildObjects")
		changed = true
	}

	return changed
}

func normalizeAdoptedStubMetaDataSectionChildObjects(childObjects *etree.Element, allowedAttrs map[string]struct{}) bool {
	if childObjects == nil {
		return false
	}

	changed := false
	for _, child := range append([]etree.Token(nil), childObjects.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}

		if localName(el.Tag) != "Attribute" {
			childObjects.RemoveChild(el)
			changed = true
			continue
		}

		if _, keep := allowedAttrs[metadataChildName(el)]; keep {
			if normalizeAdoptedStubMetaDataRetainedChild(el) {
				changed = true
			}
			continue
		}

		childObjects.RemoveChild(el)
		changed = true
	}

	return changed
}

func normalizeAdoptedStubMetaDataRetainedChild(el *etree.Element) bool {
	if el == nil {
		return false
	}

	properties := el.FindElement("./Properties")
	if properties == nil {
		return false
	}

	before := strings.TrimSpace(textOf(properties, "ObjectBelonging"))
	removedExtended := deleteElement(properties, "ExtendedConfigurationObject")
	if before == "" {
		if el.SelectAttrValue(preserveNativeObjectBelongingAttr, "") == "true" {
			return removedExtended
		}
		el.CreateAttr(preserveNativeObjectBelongingAttr, "true")
		return true
	}

	setObjectBelonging(properties, "Native")
	if el.SelectAttrValue(preserveNativeObjectBelongingAttr, "") != "true" {
		el.CreateAttr(preserveNativeObjectBelongingAttr, "true")
	}
	return true
}

func metadataChildName(el *etree.Element) string {
	if el == nil {
		return ""
	}

	if name := strings.TrimSpace(el.SelectAttrValue("name", "")); name != "" {
		return name
	}

	return strings.TrimSpace(textOfFirst(el, "./Properties/Name"))
}

func normalizeSubsystemChildObjects(doc *etree.Document, parentChain []string, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	if doc == nil || len(parentChain) == 0 {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	target := root
	if strings.EqualFold(localName(root.Tag), "MetaDataObject") {
		children := root.ChildElements()
		if len(children) > 0 {
			target = children[0]
		}
	}

	childObjects := target.FindElement("./ChildObjects")
	if childObjects == nil {
		return false
	}

	allowed := collectAllowedSubsystemChildren(parentChain, contexts, decisions)
	changed := false
	for _, child := range append([]etree.Token(nil), childObjects.Child...) {
		el, ok := child.(*etree.Element)
		if !ok || !strings.EqualFold(localName(el.Tag), "Subsystem") {
			continue
		}
		name := strings.TrimSpace(el.Text())
		if name == "" {
			childObjects.RemoveChild(el)
			changed = true
			continue
		}
		if _, ok := allowed[name]; ok {
			continue
		}
		childObjects.RemoveChild(el)
		changed = true
	}

	return changed
}

func normalizeSubsystemContent(doc *etree.Document, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	target := root
	if strings.EqualFold(localName(root.Tag), "MetaDataObject") {
		children := root.ChildElements()
		if len(children) > 0 {
			target = children[0]
		}
	}

	content := target.FindElement("./Properties/Content")
	if content == nil {
		return false
	}

	allowed := collectRootConfigurationChildObjects(contexts, decisions)
	changed := false
	for _, child := range append([]etree.Token(nil), content.Child...) {
		el, ok := child.(*etree.Element)
		if !ok || !strings.EqualFold(localName(el.Tag), "Item") {
			continue
		}

		key := strings.TrimSpace(el.Text())
		if key == "" {
			content.RemoveChild(el)
			changed = true
			continue
		}

		if _, keep := allowed[key]; keep {
			continue
		}

		content.RemoveChild(el)
		changed = true
	}

	return changed
}

func normalizeChartOfCharacteristicTypesPredefined(ctx *FileProcessingContext, contexts []*FileProcessingContext) bool {
	if ctx == nil || ctx.Doc == nil || ctx.OwnerKey == "" {
		return false
	}

	ownerCtx := findTopLevelMetadataContextByOwnerKey(contexts, ctx.OwnerKey)
	if ownerCtx == nil || ownerCtx.Doc == nil {
		return false
	}

	ownerType := ownerCtx.Doc.FindElement("//*[local-name()='Properties']/*[local-name()='Type']")
	if ownerType == nil {
		return false
	}

	allowed := make(map[string]string)
	ownerTypeChildren := append([]*etree.Element(nil), ownerType.ChildElements()...)
	for _, child := range ownerTypeChildren {
		tag := localName(child.Tag)
		if tag != "Type" && tag != "TypeSet" {
			continue
		}
		value := strings.TrimSpace(child.Text())
		if value == "" {
			continue
		}
		allowed[normalizeComparableTypeValue(value)] = value
	}
	if len(allowed) == 0 {
		return false
	}

	root := ctx.Doc.Root()
	if root == nil {
		return false
	}

	changed := false

	for _, item := range root.ChildElements() {
		if !strings.EqualFold(localName(item.Tag), "Item") {
			continue
		}
		typeEl := item.FindElement("./Type")
		if typeEl == nil {
			continue
		}

		typeNodes := make([]*etree.Element, 0, len(typeEl.ChildElements()))
		hasUnknown := false
		for _, child := range typeEl.ChildElements() {
			tag := localName(child.Tag)
			if tag != "Type" && tag != "TypeSet" {
				continue
			}
			typeNodes = append(typeNodes, child)
			current := strings.TrimSpace(child.Text())
			_, ok := allowed[normalizeComparableTypeValue(current)]
			if !ok {
				hasUnknown = true
				break
			}
		}

		if len(typeNodes) == 0 {
			continue
		}

		if hasUnknown {
			root.RemoveChild(item)
			changed = true
			continue
		}

		if syncCharacteristicPredefinedTypeQualifiers(typeEl, ownerTypeChildren) {
			changed = true
		}
	}

	return changed
}

func normalizeRetainedAdoptedCommandChildObjects(childObjects *etree.Element, retainedCommands map[string]struct{}) bool {
	if childObjects == nil {
		return false
	}

	changed := false
	for _, child := range append([]etree.Token(nil), childObjects.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		if localName(el.Tag) == "Command" {
			if _, keep := retainedCommands[metadataChildName(el)]; keep {
				continue
			}
		}
		childObjects.RemoveChild(el)
		changed = true
	}

	return changed
}

func collectRetainedOwnerCommands(contexts []*FileProcessingContext, decisions map[string]objectDecision) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})

	for _, ctx := range contexts {
		if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}
		decision, ok := decisions[ctx.OwnerKey]
		if ok && decision.Excluded {
			continue
		}
		fromForm := strings.Contains(filepath.ToSlash(ctx.RelPath), "/Forms/")

		for _, value := range collectElementValues(ctx.Doc.Root()) {
			if !isMetadataCommandReference(value) {
				continue
			}
			parts := strings.Split(strings.TrimSpace(value), ".")
			if len(parts) < 4 || !strings.EqualFold(parts[2], "Command") {
				continue
			}
			ownerKey := strings.TrimSpace(parts[0] + "." + parts[1])
			commandName := strings.TrimSpace(parts[3])
			if ownerKey == "" || commandName == "" {
				continue
			}
			if fromForm && ownerKey != ctx.OwnerKey {
				continue
			}
			ownerDecision, exists := decisions[ownerKey]
			if !exists || ownerDecision.Excluded || ownerDecision.Belonging == "Native" {
				continue
			}
			if result[ownerKey] == nil {
				result[ownerKey] = make(map[string]struct{})
			}
			result[ownerKey][commandName] = struct{}{}
		}
	}

	return result
}

func syncCharacteristicPredefinedTypeQualifiers(typeEl *etree.Element, ownerTypeNodes []*etree.Element) bool {
	if typeEl == nil || len(ownerTypeNodes) == 0 {
		return false
	}

	itemNodes := make([]*etree.Element, 0, len(typeEl.ChildElements()))
	itemComparable := make(map[string]struct{})
	for _, child := range typeEl.ChildElements() {
		tag := localName(child.Tag)
		if tag != "Type" && tag != "TypeSet" {
			continue
		}
		value := strings.TrimSpace(child.Text())
		if value == "" {
			continue
		}
		itemNodes = append(itemNodes, child)
		itemComparable[normalizeComparableTypeValue(value)] = struct{}{}
	}
	if len(itemNodes) == 0 {
		return false
	}

	ownerQualifiers := collectOwnerCharacteristicTypeQualifiers(ownerTypeNodes, itemComparable)
	if len(ownerQualifiers) == 0 {
		return false
	}

	existingQualifierIndexes := make([]int, 0, 4)
	for idx, child := range typeEl.ChildElements() {
		if isCharacteristicTypeQualifierNode(child) {
			existingQualifierIndexes = append(existingQualifierIndexes, idx)
		}
	}

	changed := false
	for i := len(existingQualifierIndexes) - 1; i >= 0; i-- {
		idx := existingQualifierIndexes[i]
		children := typeEl.ChildElements()
		if idx >= 0 && idx < len(children) {
			typeEl.RemoveChild(children[idx])
			changed = true
		}
	}

	for _, qualifier := range ownerQualifiers {
		typeEl.AddChild(qualifier.Copy())
		changed = true
	}

	return changed
}

func collectOwnerCharacteristicTypeQualifiers(ownerTypeNodes []*etree.Element, itemComparable map[string]struct{}) []*etree.Element {
	result := make([]*etree.Element, 0, 4)
	if len(ownerTypeNodes) == 0 || len(itemComparable) == 0 {
		return result
	}

	for _, node := range ownerTypeNodes {
		if node == nil {
			continue
		}
		if isCharacteristicTypeQualifierNode(node) {
			result = append(result, node)
		}
	}

	return result
}

func isCharacteristicTypeQualifierNode(el *etree.Element) bool {
	if el == nil {
		return false
	}
	switch localName(el.Tag) {
	case "StringQualifiers", "NumberQualifiers", "DateQualifiers", "BinaryDataQualifiers":
		return true
	default:
		return false
	}
}

func normalizeComparableTypeValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "xs:") {
		return value
	}

	if idx := strings.IndexByte(value, ':'); idx >= 0 {
		return value[idx+1:]
	}

	return value
}

func collectAllowedSubsystemChildren(parentChain []string, contexts []*FileProcessingContext, decisions map[string]objectDecision) map[string]struct{} {
	result := make(map[string]struct{})
	parentLen := len(parentChain)
	if parentLen == 0 {
		return result
	}

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKind != "Subsystem" || ctx.OwnerKey == "" {
			continue
		}
		decision, ok := decisions[ctx.OwnerKey]
		if !ok || decision.Excluded {
			continue
		}

		chain := subsystemChain(ctx.RelPath)
		if len(chain) != parentLen+1 {
			continue
		}
		matches := true
		for idx, part := range parentChain {
			if chain[idx] != part {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		result[chain[len(chain)-1]] = struct{}{}
	}

	return result
}

func findTopLevelMetadataContextByOwnerKey(contexts []*FileProcessingContext, key string) *FileProcessingContext {
	return findTopLevelMetadataContextByOwnerKeyIndexed(nil, contexts, key)
}

func findTopLevelMetadataContextByOwnerKeyIndexed(indexes *contextIndexes, contexts []*FileProcessingContext, key string) *FileProcessingContext {
	candidates := contexts
	if indexes != nil && key != "" {
		candidates = indexes.byOwnerKey[key]
	}
	for _, ctx := range candidates {
		if ctx == nil || ctx.OwnerKey != key || !ctx.Metadata || !isTopLevelMetadataFile(ctx) {
			continue
		}
		return ctx
	}

	return nil
}

func topLevelMetadataIncluded(key string, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	return topLevelMetadataIncludedIndexed(key, contexts, nil, decisions)
}

func topLevelMetadataIncludedIndexed(key string, contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) bool {
	if key == "" {
		return false
	}

	ctx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, key)
	if ctx == nil || ctx.Doc == nil {
		return false
	}

	decision, ok := decisions[key]
	if !ok {
		return false
	}

	return !decision.Excluded && decision.Belonging != ""
}

func hasDirectChildByLocalName(parent *etree.Element, name string) bool {
	if parent == nil || name == "" {
		return false
	}

	for _, child := range parent.ChildElements() {
		if strings.EqualFold(localName(child.Tag), name) {
			return true
		}
	}

	return false
}

func validateFormDynamicListContracts(
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	contracts map[string]formDynamicListContract,
) error {
	return validateFormDynamicListContractsIndexed(contexts, nil, decisions, contracts)
}

func validateFormDynamicListContractsIndexed(
	contexts []*FileProcessingContext,
	indexes *contextIndexes,
	decisions map[string]objectDecision,
	contracts map[string]formDynamicListContract,
) error {
	for key, contract := range contracts {
		decision, ok := decisions[key]
		if !ok || decision.Excluded {
			return fmt.Errorf("динамический список требует объект %s, но он исключен из итогового состава", key)
		}
		if decision.Belonging != "Native" && decision.Truncated {
			return fmt.Errorf("динамический список требует объект %s в режиме AdoptedStubExt(Form), но объект остался урезанным", key)
		}

		ctx := findContextByOwnerKeyIndexed(indexes, contexts, key)
		if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
			return fmt.Errorf("динамический список требует объект %s, но его XML не найден", key)
		}
		if decision.Belonging == "Native" {
			continue
		}

		if err := validateDynamicListFields(ctx, contract.RequiredFields, contract.QueryAliases); err != nil {
			return err
		}
	}

	return nil
}

func validateDynamicListFields(ctx *FileProcessingContext, requiredFields map[string]struct{}, queryAliases map[string]struct{}) error {
	if ctx == nil || len(requiredFields) == 0 {
		return nil
	}

	available := collectAvailableDynamicListFields(ctx)
	for field := range requiredFields {
		if _, ok := available[field]; ok {
			continue
		}
		if _, ok := queryAliases[field]; ok {
			continue
		}
		if isKnownDynamicListVirtualField(ctx.OwnerKind, field) {
			continue
		}
		return fmt.Errorf("динамический список требует поле %s.%s, но оно не сохранено в %s", ctx.OwnerKey, field, ctx.Path)
	}

	return nil
}

var dynamicListQueryAliasRegexp = regexp.MustCompile(`(?i)\bКАК\s+([\p{L}_][\p{L}\p{N}_]*)`)

func collectAvailableDynamicListFields(ctx *FileProcessingContext) map[string]struct{} {
	result := make(map[string]struct{})
	if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
		return result
	}

	root := ctx.Doc.Root()
	target := root
	if strings.EqualFold(localName(root.Tag), "MetaDataObject") {
		children := root.ChildElements()
		if len(children) > 0 {
			target = children[0]
		}
	}

	for name := range collectStandardAttributeNames(target) {
		result[name] = struct{}{}
		for _, alias := range dynamicListStandardAttributeAliases(name) {
			result[alias] = struct{}{}
		}
	}

	for _, childObjects := range target.FindElements("./ChildObjects") {
		collectAvailableDynamicListChildFields(childObjects, "", result)
	}

	return result
}

func collectAvailableDynamicListChildFields(childObjects *etree.Element, parentPath string, result map[string]struct{}) {
	if childObjects == nil {
		return
	}

	for _, child := range childObjects.ChildElements() {
		tag := localName(child.Tag)
		if !isDynamicListFieldChildKind(tag) && tag != "TabularSection" {
			continue
		}

		name := strings.TrimSpace(textOf(child.FindElement("./Properties"), "Name"))
		if name == "" {
			continue
		}

		currentPath := name
		if parentPath != "" {
			currentPath = parentPath + "." + name
		}
		result[currentPath] = struct{}{}

		if tag != "TabularSection" {
			continue
		}

		for _, field := range []string{"Ref", "LineNumber"} {
			result[currentPath+"."+field] = struct{}{}
			for _, alias := range dynamicListStandardAttributeAliases(field) {
				result[currentPath+"."+alias] = struct{}{}
			}
		}

		collectAvailableDynamicListChildFields(child.FindElement("./ChildObjects"), currentPath, result)
	}
}

func dynamicListStandardAttributeAliases(name string) []string {
	switch strings.TrimSpace(name) {
	case "Ref":
		return []string{"Ссылка"}
	case "DeletionMark":
		return []string{"ПометкаУдаления"}
	case "IsFolder":
		return []string{"ЭтоГруппа"}
	case "Owner":
		return []string{"Владелец"}
	case "Parent":
		return []string{"Родитель"}
	case "Description":
		return []string{"Наименование"}
	case "Code":
		return []string{"Код"}
	case "Number":
		return []string{"Номер"}
	case "Date":
		return []string{"Дата"}
	case "Posted":
		return []string{"Проведен"}
	case "Recorder":
		return []string{"Регистратор"}
	case "Period":
		return []string{"Период"}
	case "LineNumber":
		return []string{"НомерСтроки"}
	case "Active":
		return []string{"Активность"}
	default:
		return nil
	}
}

func isDynamicListFieldChildKind(tag string) bool {
	switch strings.TrimSpace(tag) {
	case "Attribute", "Dimension", "Resource", "Measure":
		return true
	default:
		return false
	}
}

func isKnownDynamicListVirtualField(kind, field string) bool {
	switch field {
	case "Ref", "Description", "Code", "Number", "Date", "IsFolder", "Posted", "DeletionMark", "Presentation", "DefaultPicture",
		"Ссылка", "Наименование", "Код", "Номер", "Дата", "ЭтоГруппа", "Проведен", "ПометкаУдаления", "Владелец":
		return true
	}

	if strings.HasPrefix(field, "НомерКартинки") {
		return true
	}

	if strings.EqualFold(kind, "Catalog") && (field == "Parent" || field == "Родитель") {
		return true
	}

	return false
}

func disablePrivilegedMode(properties *etree.Element) (bool, error) {
	current := properties.FindElement("Privileged")
	if current == nil {
		return false, nil
	}

	value, err := strconv.ParseBool(strings.TrimSpace(current.Text()))
	if err != nil {
		return false, err
	}
	if value {
		current.SetText("false")
		return true, nil
	}

	return false, nil
}

func findProperties(doc *etree.Document) *etree.Element {
	return doc.FindElement("//Properties")
}

func processElement(properties *etree.Element, element *config.ElementOperation) bool {
	switch element.Operation {
	case config.Add:
		return addElement(properties, element.ElementName, element.Value)
	case config.Modify:
		return modifyElement(properties, element.ElementName, element.Value)
	case config.Delete:
		return deleteElement(properties, element.ElementName)
	default:
		log.Printf("неизвестная операция: %v для элемента: %s", element.Operation, element.ElementName)
		return false
	}
}

func setObjectBelonging(properties *etree.Element, value string) {
	if properties == nil || value == "" {
		return
	}

	if value == "Native" {
		deleteElement(properties, "ObjectBelonging")
		return
	}

	xmlValue := value
	if xmlValue == "AdoptedStub" {
		xmlValue = "Adopted"
	}

	if !modifyElement(properties, "ObjectBelonging", xmlValue) {
		addElement(properties, "ObjectBelonging", xmlValue)
	}
}

func addElement(properties *etree.Element, tag, value string) bool {
	if properties == nil {
		return false
	}

	if currentElem := properties.FindElement(tag); currentElem == nil {
		currentElem = properties.CreateElement(tag)
		currentElem.SetText(value)
		return true
	}

	return modifyElement(properties, tag, value)
}

func addSimpleElement(parent *etree.Element, tag, value string) *etree.Element {
	element := parent.CreateElement(tag)
	element.SetText(value)
	return element
}

func setAttrValue(element *etree.Element, key, value string) bool {
	if element == nil || key == "" {
		return false
	}

	for i := range element.Attr {
		if strings.EqualFold(localName(element.Attr[i].Key), key) {
			if element.Attr[i].Value == value {
				return false
			}
			element.Attr[i].Value = value
			return true
		}
	}

	element.CreateAttr(key, value)
	return true
}

func modifyElement(properties *etree.Element, path, value string) bool {
	if properties == nil {
		return false
	}

	currentElem := properties.FindElement(path)
	if currentElem == nil {
		return false
	}

	if currentElem.Text() == value {
		return false
	}

	currentElem.SetText(value)
	return true
}

func deleteElement(properties *etree.Element, path string) bool {
	if properties == nil {
		return false
	}

	currentElem := properties.FindElement(path)
	if currentElem == nil {
		return false
	}

	properties.RemoveChild(currentElem)
	return true
}

func readXMLFile(path string) (*etree.Document, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла %s: %w", path, err)
	}
	return doc, nil
}

func textOf(properties *etree.Element, tag string) string {
	if properties == nil {
		return ""
	}
	element := properties.FindElement(tag)
	if element == nil {
		return ""
	}
	return element.Text()
}

func removeAllChildren(element *etree.Element) {
	if element == nil {
		return
	}
	for _, child := range append([]etree.Token(nil), element.Child...) {
		element.RemoveChild(child)
	}
}

func setRussianSynonym(synonym *etree.Element, value string) {
	if synonym == nil {
		return
	}

	item := synonym.CreateElement("v8:item")
	lang := item.CreateElement("v8:lang")
	lang.SetText("ru")
	content := item.CreateElement("v8:content")
	content.SetText(value)
}

func extractRussianSynonym(synonym *etree.Element) string {
	if synonym == nil {
		return ""
	}

	for _, item := range synonym.ChildElements() {
		lang := item.FindElement("v8:lang")
		content := item.FindElement("v8:content")
		if lang != nil && content != nil && strings.EqualFold(strings.TrimSpace(lang.Text()), "ru") {
			return strings.TrimSpace(content.Text())
		}
	}

	return ""
}

func ensureEnumValueDefaultColor(doc *etree.Document) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	changed := false
	for _, enumValue := range root.FindElements(".//EnumValue") {
		properties := enumValue.FindElement("./Properties")
		if properties == nil {
			continue
		}
		if properties.FindElement("Color") != nil {
			continue
		}
		addSimpleElement(properties, "Color", "auto")
		changed = true
	}

	return changed
}

func addUsePurposes(properties *etree.Element) {
	if properties == nil {
		return
	}

	usePurposes := properties.CreateElement("UsePurposes")
	value := usePurposes.CreateElement("v8:Value")
	value.CreateAttr("xsi:type", "app:ApplicationUsePurpose")
	value.SetText("PlatformApplication")
}

func isXMLFile(fileName string) bool {
	return strings.EqualFold(filepath.Ext(fileName), ".xml")
}

func isTopLevelMetadataRelPath(relPath string) bool {
	parts := strings.Split(relPath, "/")
	return len(parts) == 2 && strings.HasSuffix(strings.ToLower(parts[1]), ".xml")
}

func isMetadataObjectDoc(doc *etree.Document) bool {
	root := doc.Root()
	return root != nil && localName(root.Tag) == "MetaDataObject"
}

func isTopLevelMetadataFile(ctx *FileProcessingContext) bool {
	if ctx == nil {
		return false
	}
	if ctx.TopLevelMetadata {
		return true
	}
	return isTopLevelMetadataRelPath(ctx.RelPath)
}

func propertyName(properties *etree.Element) string {
	if properties == nil {
		return ""
	}
	nameElem := properties.FindElement("Name")
	if nameElem == nil {
		return ""
	}
	return strings.TrimSpace(nameElem.Text())
}

func extractGUIDs(value string) []string {
	if value == "" {
		return nil
	}

	matches := guidPattern.FindAllString(value, -1)
	if len(matches) == 0 {
		return nil
	}

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, strings.ToLower(match))
	}
	return result
}

func platformCompatibilityMode(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return ""
	}

	replacer := strings.NewReplacer(".", "_")
	return "Version" + replacer.Replace(trimmed)
}

func newGUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("не удалось сгенерировать guid: %w", err))
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	parts := []string{
		hex.EncodeToString(buf[0:4]),
		hex.EncodeToString(buf[4:6]),
		hex.EncodeToString(buf[6:8]),
		hex.EncodeToString(buf[8:10]),
		hex.EncodeToString(buf[10:16]),
	}

	return strings.Join(parts, "-")
}

func localName(tag string) string {
	if idx := strings.Index(tag, ":"); idx >= 0 {
		return tag[idx+1:]
	}
	return tag
}

func cleanupEmptyParents(path, root string) {
	root = filepath.Clean(root)
	current := filepath.Clean(path)

	for strings.HasPrefix(current, root) && current != root {
		entries, err := os.ReadDir(current)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(current); err != nil {
			return
		}
		current = filepath.Dir(current)
	}
}

func splitObjectKey(key string) (string, string) {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func isExcludedMetadata(metadataName string, excluded map[string]map[string]struct{}) bool {
	for kind, names := range excluded {
		prefix := kind + "."
		if !strings.HasPrefix(metadataName, prefix) {
			continue
		}

		rest := strings.TrimPrefix(metadataName, prefix)
		for name := range names {
			if rest == name || strings.HasPrefix(rest, name+".") {
				return true
			}
		}
	}

	return false
}
