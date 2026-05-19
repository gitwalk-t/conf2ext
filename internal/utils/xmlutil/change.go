package xmlutils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"github.com/gitwalk-m/conf2ext/internal/config"
)

const (
	dirCommonModules                  = "CommonModules"
	mainFile                          = "Configuration.xml"
	configDumpInfo                    = "ConfigDumpInfo.xml"
	preserveNativeObjectBelongingAttr = "codexPreserveNativeObjectBelonging"
)

var (
	guidPattern               = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	metadataReferencePattern  = regexp.MustCompile(`(?:[A-Za-z_][A-Za-z0-9_.-]*:)?(AccountingRegister|AccumulationRegister|BusinessProcess|CalculationRegister|Catalog|ChartOfAccounts|ChartOfCalculationTypes|ChartOfCharacteristicTypes|CommandGroup|CommonAttribute|CommonCommand|CommonForm|CommonModule|CommonPicture|CommonTemplate|Constant|DataProcessor|DefinedType|Document|DocumentJournal|Enum|EventSubscription|ExchangePlan|ExternalDataSource|FilterCriterion|FunctionalOption|FunctionalOptionsParameter|HTTPService|InformationRegister|IntegrationService|Interface|Language|Report|Role|ScheduledJob|Sequence|Session|SessionParameter|SettingsStorage|Style|StyleItem|Subsystem|Task|WebService|XDTOPackage)(?:Ref|Object|Selection|List|Manager|ValueManager|RecordSet|TabularSectionRow|TabularSection)?\.([^\s<>"':/\\]+)`)
	styleReferencePattern     = regexp.MustCompile(`(?:^|[^[:alnum:]_])style:([^\s<>"':/\\]+)`)
	moduleMethodHeaderPattern = regexp.MustCompile(`^(\s*)(Процедура|Функция)(\s+)([A-Za-zА-Яа-яЁё_][A-Za-zА-Яа-яЁё0-9_]*)`)

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
		"ОпределяемыйТип":        "DefinedType",
		"Перечисление":           "Enum",
		"ПланВидовХарактеристик": "ChartOfCharacteristicTypes",
		"ПланОбмена":             "ExchangePlan",
		"РегистрБухгалтерии":     "AccountingRegister",
		"РегистрНакопления":      "AccumulationRegister",
		"РегистрСведений":        "InformationRegister",
		"Справочник":             "Catalog",
		"ПодпискаНаСобытие":      "EventSubscription",
	}

	searchResultKindAliases = map[string]string{
		"БизнесПроцессы":               "BusinessProcess",
		"Документы":                    "Document",
		"ЖурналыДокументов":            "DocumentJournal",
		"Задачи":                       "Task",
		"Константы":                    "Constant",
		"КритерииОтбора":               "FilterCriterion",
		"Обработки":                    "DataProcessor",
		"ОбщиеКоманды":                 "CommonCommand",
		"ОбщиеМакеты":                  "CommonTemplate",
		"ОбщиеМодули":                  "CommonModule",
		"ОбщиеФормы":                   "CommonForm",
		"ОбщиеКартинки":                "CommonPicture",
		"ОбщиеРеквизиты":               "CommonAttribute",
		"Отчеты":                       "Report",
		"ПараметрыСеанса":              "SessionParameter",
		"Перечисления":                 "Enum",
		"ПланыВидовРасчета":            "ChartOfCalculationTypes",
		"ПланыВидовХарактеристик":      "ChartOfCharacteristicTypes",
		"ПланыОбмена":                  "ExchangePlan",
		"ПодпискиНаСобытия":            "EventSubscription",
		"Последовательности":           "Sequence",
		"РегистрыБухгалтерии":          "AccountingRegister",
		"РегистрыНакопления":           "AccumulationRegister",
		"РегистрыРасчета":              "CalculationRegister",
		"РегистрыСведений":             "InformationRegister",
		"Роли":                         "Role",
		"Справочники":                  "Catalog",
		"Стили":                        "Style",
		"ЭлементыСтиля":                "StyleItem",
		"ФункциональныеОпции":          "FunctionalOption",
		"ПараметрыФункциональныхОпций": "FunctionalOptionsParameter",
		"ХранилищаНастроек":            "SettingsStorage",
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
	Belonging              string
	Excluded               bool
	Truncated              bool
	SearchResultCode       bool
	AdoptedStubExtMetaData bool
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

type targetMergeRuleSet struct {
	ObjectKeys map[string]struct{}
}

type targetCompatibilitySet struct {
	Enabled bool
	Keys    map[string]struct{}
}

type preparedTargetMergeObject struct {
	Key                 string
	Kind                string
	CurrentCtx          *FileProcessingContext
	TargetCtx           *FileProcessingContext
	CurrentContentCtx   *FileProcessingContext
	TargetContentCtx    *FileProcessingContext
	ExchangePlanRelPath string
}

type targetMergePerfStats struct {
	MergeObjects       int
	TargetRefs         int
	ImportedTargetRefs int
	SkippedTargetRefs  int
	CacheHits          int
	CacheMisses        int
}

type searchResultObjectOverlay struct {
	PreserveForms    map[string]struct{}
	PreserveCommands map[string]struct{}
}

type searchResultModuleWrite struct {
	OwnerKey string
	Path     string
	Content  string
}

type searchResultState struct {
	ObjectOverlays          map[string]searchResultObjectOverlay
	ModuleWrites            map[string]searchResultModuleWrite
	ExpectedAdoptedObjects  map[string]struct{}
	PreservedPaths          map[string]struct{}
	PreservedConfigDumpInfo map[string]struct{}
}

type changeFilesState struct {
	contexts                 []*FileProcessingContext
	indexes                  *contextIndexes
	decisions                map[string]objectDecision
	formDynamicListContracts map[string]formDynamicListContract
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule
	searchResultState        *searchResultState
	excludedPaths            map[string]struct{}
}

type identityMapState struct {
	Version int                                 `json:"version"`
	Objects map[string]identityMapObjectBinding `json:"objects"`
}

type identityMapObjectBinding struct {
	ExtensionID string `json:"extension_id"`
}

type metadataPathSegment struct {
	Kind string
	Name string
}

type metadataBindingTarget struct {
	MetadataPath string
	CurrentID    string
	BaseObjectID string
	HasBinding   bool
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

type metadataTargetExistsFunc func(string) bool

type liveCommandReferenceIndex struct {
	byOwner     map[string]map[string]struct{}
	byDoc       map[*FileProcessingContext]map[string]struct{}
	scannedDocs int
}

type retainedOwnerCommandFinalizationStats struct {
	ChangedFiles       int
	WrittenFiles       int
	FinalizedOwnerDocs int
	AffectedDocs       int
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
	explicitNativeObjects := collectConfiguredNativeObjects(contexts, cfg.IncludedNativeObjects)
	includedAdoptedStubObjects := collectConfiguredAdoptedStubObjects(contexts, cfg)
	adoptedStubMetaDataRules := collectAdoptedStubMetaDataRules(cfg, dir)
	for key := range adoptedStubMetaDataRules {
		includedAdoptedStubObjects[key] = struct{}{}
	}
	forbiddenAdoptedStubObjects := collectConfiguredForbiddenStubObjects(contexts, cfg.ForbiddenAdoptedStubObjects)
	prefixNativeObjects := collectNativePrefixObjects(contexts, cfg.NativePrefixes)
	primaryNativeObjects := mergeObjectSets(explicitNativeObjects, prefixNativeObjects)
	excludedSubsystemObjects := collectExcludedSubsystemObjects(contexts, cfg.ExcludedSubsystems, cfg.NativePrefixes)
	configuredExcludedObjects := collectConfiguredExcludedObjects(contexts, cfg.ExcludedObjects)
	excludedObjects := mergeObjectSets(excludedSubsystemObjects, configuredExcludedObjects)
	targetCompatibilitySet, err := collectTargetCompatibilitySet(cfg)
	if err != nil {
		return err
	}
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
		decisions[ctx.OwnerKey] = decideObject(ctx, cfg, explicitNativeObjects, prefixNativeObjects, excludedObjects, includedAdoptedStubObjects, forbiddenAdoptedStubObjects)
	}
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	applyTargetCompatibilitySet(decisions, contexts, targetCompatibilitySet, forbiddenAdoptedStubObjects)
	logXMLStepCompleted("collect subsystem decisions", collectSubsystemDecisionsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	searchResultState, err := collectSearchResultState(cfg, dir, contexts, decisions, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects)
	if err != nil {
		return err
	}
	targetMergeRules := collectTargetMergeRules(cfg, dir)

	log.Printf("xml step: promote referenced objects")
	promoteReferencedObjectsStartedAt := time.Now()
	referenceGraph := collectReferenceGraph(contexts, cfg, primaryNativeObjects, decisions, searchResultState)
	incomingReferenceGraph := collectIncomingReferenceGraph(referenceGraph)
	adoptedStubExtReferenceGraph := collectAdoptedStubExtReferenceGraph(contexts, decisions, searchResultState)
	formDynamicListContracts := collectFormDynamicListContracts(contexts, decisions, searchResultState)
	promoteReferencedObjectsToAdoptedStubIndexed(contexts, indexes, decisions, cfg, referenceGraph, incomingReferenceGraph, adoptedStubExtReferenceGraph, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, targetMergeRules.ObjectKeys, targetCompatibilitySet)
	promoteRegisterDocumentOwnersToNativeIndexed(contexts, indexes, decisions, cfg, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, collectRegisterDocumentReferences(contexts))
	applyFormDynamicListContracts(decisions, formDynamicListContracts, forbiddenAdoptedStubObjects)
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	applyTargetCompatibilitySet(decisions, contexts, targetCompatibilitySet, forbiddenAdoptedStubObjects)
	contexts, indexes, err = mergeTargetMetadataComposition(cfg, dir, contexts, indexes, decisions, targetMergeRules.ObjectKeys, excludedObjects, forbiddenAdoptedStubObjects, targetCompatibilitySet)
	if err != nil {
		return err
	}
	applyTargetCompatibilitySet(decisions, contexts, targetCompatibilitySet, forbiddenAdoptedStubObjects)
	retainedOwnerCommandCandidates := collectOwnerCommandCandidates(contexts, decisions)
	logXMLStepCompleted("promote referenced objects", promoteReferencedObjectsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	log.Printf("xml step: collect cleanup sets")
	collectCleanupSetsStartedAt := time.Now()
	baseBindings, err := loadBaseBindings(cfg)
	if err != nil {
		return err
	}
	identityMap, err := loadIdentityMapState(cfg)
	if err != nil {
		return err
	}
	excludedRefs := collectExcludedReferences(decisions)
	blockedForbiddenObjectKeys := collectBlockedForbiddenObjectKeys(decisions, forbiddenAdoptedStubObjects)
	forbiddenChildMetadataPaths := collectForbiddenChildMetadataPaths(blockedForbiddenObjectKeys)
	blockedForbiddenRefs := collectReferenceMapFromObjectKeys(blockedForbiddenObjectKeys)
	excludedRefs = mergeReferenceMaps(excludedRefs, blockedForbiddenRefs)
	excludedMetadataPrefixes := collectExcludedMetadataPrefixes(excludedRefs)
	truncatedKeys := collectTruncatedKeys(decisions)
	truncatedChildPrefixes := collectTruncatedChildPrefixes(truncatedKeys)
	guidReplacements := collectGUIDReplacements(contexts, decisions, identityMap, adoptedStubMetaDataRules)
	bindingTargetsByDoc := collectMetadataBindingTargetsByDoc(contexts, baseBindings, decisions, adoptedStubMetaDataRules)
	baseBindingReplacements := collectBaseBindingReferenceReplacements(bindingTargetsByDoc, guidReplacements)
	if err := saveIdentityMapState(cfg, identityMap); err != nil {
		return err
	}
	// Для DefinedType режем только hard forbidden: мягко исключенные типы
	// должны сохраняться в составе и дотягиваться по RefDrivenInclusion.
	blockedDefinedTypeObjects := blockedForbiddenObjectKeys
	excludedPaths := collectExcludedPaths(contexts, decisions, dir, searchResultState, forbiddenChildMetadataPaths, targetMergeRules.ObjectKeys)
	logXMLStepCompleted("collect cleanup sets", collectCleanupSetsStartedAt)

	log.Printf("xml step: apply object changes")
	applyObjectChangesStartedAt := time.Now()
	lastApplyObjectChangesLogAt := time.Now()
	changedFilesCount := 0
	writtenFilesCount := 0
	liveCommandRefs := newLiveCommandReferenceIndex()
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

		if ctx.FileName != configDumpInfo && ctx.OwnerKey != "Configuration" && !isTopLevelMetadataFile(ctx) &&
			decision.Belonging != "Native" && !searchResultPreservesPath(searchResultState, ctx.Path) &&
			!preserveTargetMergePath(ctx, targetMergeRules.ObjectKeys) {
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
			changed = cleanupRootConfigurationModuleTexts(ctx.Doc) || changed
		}

		if ctx.FileName == configDumpInfo {
			changed = normalizeConfigDumpInfoRootNames(ctx.Doc, config.GetDumpInfo().ConfigName, cfg.ExtensionName()) || changed
			changed = cleanupConfigDumpInfoRootServiceEntries(ctx.Doc, cfg.ExtensionName()) || changed
			changed = cleanupConfigDumpInfoForbiddenMetadata(ctx.Doc, blockedForbiddenObjectKeys) || changed
			changed = cleanupConfigDumpInfoNonNativeChildren(ctx.Doc, contexts, decisions, searchResultState.PreservedConfigDumpInfo) || changed
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
			if decision.Belonging != "Native" {
				changed = cleanupNonNativeManualQueryOrphanReferences(ctx.Doc) || changed
				ownerCtx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, ctx.OwnerKey)
				changed = cleanupMissingFormOwnerObjectReferences(ctx.Doc, ownerCtx) || changed
				changed = cleanupNonNativeFormLifecycleEvents(ctx.Doc) || changed
				changed = cleanupNonNativeFormStandardCommands(ctx.Doc) || changed
			}
		}

		if ctx.OwnerKind == "FunctionalOptionsParameter" && ctx.Properties != nil {
			changed = cleanupFunctionalOptionsParameterUseNativeChildRefs(ctx.Properties, decisions) || changed
		}

		if ctx.OwnerKey == "Language.Русский" {
			changed = normalizeLanguageObject(ctx.Properties) || changed
		}

		if decision.Belonging != "Native" && ctx.Metadata && isTopLevelMetadataFile(ctx) &&
			ctx.OwnerKey != "Configuration" && ctx.OwnerKey != "Language.Русский" {
			changed = cleanupAdoptedObjectFormReferences(ctx.Properties) || changed
			contract, hasContract := formDynamicListContracts[ctx.OwnerKey]
			overlay := searchResultObjectOverlayForKey(searchResultState, ctx.OwnerKey)
			overlay = mergeSearchResultOverlayCommands(overlay, retainedOwnerCommandCandidates[ctx.OwnerKey])
			rule, hasRule := adoptedStubMetaDataRules[ctx.OwnerKey]
			if isAdoptedStubExtMetaData(ctx, decision) {
				changed = normalizeAdoptedStubExtMetaData(ctx.Doc, ctx.OwnerKind, overlay) || changed
			} else if ctx.OwnerKind == "DefinedType" || ctx.OwnerKind == "EventSubscription" {
				// Для специальных adopted metadata-object сохраняем composition/source
				// и не минимизируем их до обычного AdoptedStub.
			} else if hasContract && hasRule {
				changed = normalizeAdoptedStubExtFormComposition(ctx.Doc, mergeAdoptedStubMetaDataIntoFormContract(contract, rule), overlay) || changed
			} else if hasRule {
				changed = normalizeAdoptedStubMetaDataComposition(ctx.Doc, ctx.OwnerKind, rule, overlay) || changed
			} else if hasContract {
				changed = normalizeAdoptedStubExtFormComposition(ctx.Doc, contract, overlay) || changed
			} else {
				changed = normalizeAdoptedObjectComposition(ctx.Doc, ctx.OwnerKind, overlay) || changed
			}
		}

		if ctx.OwnerKind == "Subsystem" && ctx.Metadata {
			changed = normalizeSubsystemChildObjects(ctx.Doc, subsystemChain(ctx.RelPath), contexts, decisions) || changed
			changed = normalizeSubsystemContent(ctx.Doc, contexts, decisions) || changed
		}

		if ctx.OwnerKind == "ChartOfCharacteristicTypes" && strings.EqualFold(ctx.FileName, "Predefined.xml") {
			changed = normalizeChartOfCharacteristicTypesPredefined(ctx, contexts) || changed
		}

		if decision.Truncated && ctx.Metadata && isTopLevelMetadataFile(ctx) && !isAdoptedStubExtMetaData(ctx, decision) {
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
		if ctx.OwnerKind != "DefinedType" && ctx.OwnerKind != "EventSubscription" && !isAdoptedStubExtMetaData(ctx, decision) {
			changed = cleanupExcludedReferences(ctx.Doc, excludedRefs, excludedMetadataPrefixes, truncatedKeys, truncatedChildPrefixes) || changed
		}
		bindingTargets := bindingTargetsByDoc[ctx.Doc]
		changed = replaceGUIDsInDoc(ctx.Doc, guidReplacements) || changed
		changed = replaceBaseBindingGUIDsInDoc(ctx.Doc, baseBindingReplacements) || changed
		if decision.Belonging != "Native" && ctx.Metadata && isTopLevelMetadataFile(ctx) &&
			ctx.OwnerKey != "Configuration" && ctx.OwnerKey != "Language.Русский" {
			changed = ensureAdoptedExtendedConfigurationObjects(ctx.Doc, bindingTargets) || changed
		}
		if ctx.Metadata && isTopLevelMetadataFile(ctx) {
			changed = cleanupForbiddenChildMetadataPaths(ctx.Doc, ctx.OwnerKey, forbiddenChildMetadataPaths) || changed
		}

		indexLiveCommandReferences(liveCommandRefs, ctx, decision, excludedPaths)

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

	postFormCleanupChanged, postFormCleanupWritten, err := cleanupFinalNonNativeFormNoise(contexts, indexes, decisions)
	if err != nil {
		return err
	}
	changedFilesCount += postFormCleanupChanged
	writtenFilesCount += postFormCleanupWritten

	log.Printf("xml step: filter retained owner commands")
	filterRetainedOwnerCommandsStartedAt := time.Now()
	retainedOwnerCommands := filterRetainedOwnerCommandsByLiveReferences(retainedOwnerCommandCandidates, liveCommandRefs)
	logXMLStepCompleted("filter retained owner commands", filterRetainedOwnerCommandsStartedAt, fmt.Sprintf("candidate_commands=%d retained_commands=%d scanned_command_bearing_docs=%d", countRetainedOwnerCommands(retainedOwnerCommandCandidates), countRetainedOwnerCommands(retainedOwnerCommands), liveCommandRefs.scannedDocs))

	log.Printf("xml step: finalize retained owner commands")
	finalizeRetainedOwnerCommandsStartedAt := time.Now()
	finalizationStats, err := finalizeRetainedOwnerCommands(contexts, indexes, decisions, excludedPaths, adoptedStubMetaDataRules, formDynamicListContracts, retainedOwnerCommandCandidates, retainedOwnerCommands, liveCommandRefs, searchResultState, blockedForbiddenObjectKeys)
	if err != nil {
		return err
	}
	changedFilesCount += finalizationStats.ChangedFiles
	writtenFilesCount += finalizationStats.WrittenFiles
	logXMLStepCompleted("finalize retained owner commands", finalizeRetainedOwnerCommandsStartedAt, fmt.Sprintf("finalized_owner_docs=%d affected_docs=%d changed_files=%d written_files=%d", finalizationStats.FinalizedOwnerDocs, finalizationStats.AffectedDocs, finalizationStats.ChangedFiles, finalizationStats.WrittenFiles))

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

	if err := writeSearchResultModuleFiles(searchResultState, decisions); err != nil {
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

	if err := validateSearchResultAdoptedObjects(indexes, contexts, decisions, excludedPaths, searchResultState); err != nil {
		return err
	}

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

	if err := writeSearchResultModuleFiles(state.searchResultState, state.decisions); err != nil {
		return err
	}

	log.Printf("xml step: remove excluded files")
	removeExcludedFilesStartedAt := time.Now()
	removedExcludedFilesCount, err := removeExcludedFiles(dir, state.excludedPaths)
	if err != nil {
		return err
	}
	logXMLStepCompleted("remove excluded files", removeExcludedFilesStartedAt, fmt.Sprintf("excluded_paths=%d removed_files=%d", len(state.excludedPaths), removedExcludedFilesCount))

	if err := validateSearchResultAdoptedObjects(state.indexes, state.contexts, state.decisions, state.excludedPaths, state.searchResultState); err != nil {
		return err
	}

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

func loadXMLContexts(root string, relativeDirs ...string) ([]*FileProcessingContext, error) {
	contexts := make([]*FileProcessingContext, 0, 128)
	seenPaths := make(map[string]struct{})

	walkRoots := []string{root}
	if len(relativeDirs) > 0 {
		walkRoots = walkRoots[:0]
		for _, relativeDir := range relativeDirs {
			relativeDir = strings.TrimSpace(relativeDir)
			if relativeDir == "" {
				continue
			}
			relativeDir = strings.TrimLeft(relativeDir, `\/`)
			cleanRelativeDir := filepath.Clean(relativeDir)
			if cleanRelativeDir == "." {
				walkRoots = append(walkRoots, root)
				continue
			}
			if filepath.IsAbs(cleanRelativeDir) || strings.HasPrefix(cleanRelativeDir, "..") {
				return nil, fmt.Errorf("путь загрузки XML должен быть относительным к корню выгрузки: %s", relativeDir)
			}
			walkRoots = append(walkRoots, filepath.Join(root, cleanRelativeDir))
		}
	}

	walkRoot := func(walkRoot string) error {
		info, err := os.Stat(walkRoot)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("ошибка при доступе к каталогу %s: %w", walkRoot, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("путь загрузки XML должен указывать на каталог: %s", walkRoot)
		}

		return filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return fmt.Errorf("ошибка при обработке файла %s: %w", path, walkErr)
			}

			if d.IsDir() || !isXMLFile(d.Name()) {
				return nil
			}

			path = filepath.Clean(path)
			if _, exists := seenPaths[path]; exists {
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
			seenPaths[path] = struct{}{}

			return nil
		})
	}

	for _, currentRoot := range walkRoots {
		if err := walkRoot(currentRoot); err != nil {
			return nil, fmt.Errorf("ошибка при обходе директорий: %w", err)
		}
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
	explicitNativeObjects := collectConfiguredNativeObjects(contexts, cfg.IncludedNativeObjects)
	includedAdoptedStubObjects := collectConfiguredAdoptedStubObjects(contexts, cfg)
	adoptedStubMetaDataRules := collectAdoptedStubMetaDataRules(cfg, dir)
	for key := range adoptedStubMetaDataRules {
		includedAdoptedStubObjects[key] = struct{}{}
	}
	forbiddenAdoptedStubObjects := collectConfiguredForbiddenStubObjects(contexts, cfg.ForbiddenAdoptedStubObjects)
	prefixNativeObjects := collectNativePrefixObjects(contexts, cfg.NativePrefixes)
	primaryNativeObjects := mergeObjectSets(explicitNativeObjects, prefixNativeObjects)
	excludedSubsystemObjects := collectExcludedSubsystemObjects(contexts, cfg.ExcludedSubsystems, cfg.NativePrefixes)
	configuredExcludedObjects := collectConfiguredExcludedObjects(contexts, cfg.ExcludedObjects)
	excludedObjects := mergeObjectSets(excludedSubsystemObjects, configuredExcludedObjects)
	targetCompatibilitySet, err := collectTargetCompatibilitySet(cfg)
	if err != nil {
		return nil, err
	}
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
		decisions[ctx.OwnerKey] = decideObject(ctx, cfg, explicitNativeObjects, prefixNativeObjects, excludedObjects, includedAdoptedStubObjects, forbiddenAdoptedStubObjects)
	}
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	applyTargetCompatibilitySet(decisions, contexts, targetCompatibilitySet, forbiddenAdoptedStubObjects)
	logXMLStepCompleted("collect subsystem decisions", collectSubsystemDecisionsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	searchResultState, err := collectSearchResultState(cfg, dir, contexts, decisions, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects)
	if err != nil {
		return nil, err
	}
	targetMergeRules := collectTargetMergeRules(cfg, dir)

	log.Printf("xml step: promote referenced objects")
	promoteReferencedObjectsStartedAt := time.Now()
	referenceGraph := collectReferenceGraph(contexts, cfg, primaryNativeObjects, decisions, searchResultState)
	incomingReferenceGraph := collectIncomingReferenceGraph(referenceGraph)
	adoptedStubExtReferenceGraph := collectAdoptedStubExtReferenceGraph(contexts, decisions, searchResultState)
	formDynamicListContracts := collectFormDynamicListContracts(contexts, decisions, searchResultState)
	promoteReferencedObjectsToAdoptedStubIndexed(contexts, indexes, decisions, cfg, referenceGraph, incomingReferenceGraph, adoptedStubExtReferenceGraph, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, targetMergeRules.ObjectKeys, targetCompatibilitySet)
	promoteRegisterDocumentOwnersToNativeIndexed(contexts, indexes, decisions, cfg, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, collectRegisterDocumentReferences(contexts))
	applyFormDynamicListContracts(decisions, formDynamicListContracts, forbiddenAdoptedStubObjects)
	applyAdoptedStubMetaDataRules(decisions, adoptedStubMetaDataRules, excludedObjects, forbiddenAdoptedStubObjects)
	applyTargetCompatibilitySet(decisions, contexts, targetCompatibilitySet, forbiddenAdoptedStubObjects)
	contexts, indexes, err = mergeTargetMetadataComposition(cfg, dir, contexts, indexes, decisions, targetMergeRules.ObjectKeys, excludedObjects, forbiddenAdoptedStubObjects, targetCompatibilitySet)
	if err != nil {
		return nil, err
	}
	applyTargetCompatibilitySet(decisions, contexts, targetCompatibilitySet, forbiddenAdoptedStubObjects)
	logXMLStepCompleted("promote referenced objects", promoteReferencedObjectsStartedAt, fmt.Sprintf("decisions=%d", len(decisions)))

	collectCleanupSetsStartedAt := time.Now()
	forbiddenChildMetadataPaths := collectForbiddenChildMetadataPaths(collectBlockedForbiddenObjectKeys(decisions, forbiddenAdoptedStubObjects))
	excludedPaths := collectExcludedPaths(contexts, decisions, dir, searchResultState, forbiddenChildMetadataPaths, targetMergeRules.ObjectKeys)
	logXMLStepCompleted("collect cleanup sets", collectCleanupSetsStartedAt, fmt.Sprintf("excluded_paths=%d", len(excludedPaths)))

	return &changeFilesState{
		contexts:                 contexts,
		indexes:                  indexes,
		decisions:                decisions,
		formDynamicListContracts: formDynamicListContracts,
		adoptedStubMetaDataRules: adoptedStubMetaDataRules,
		searchResultState:        searchResultState,
		excludedPaths:            excludedPaths,
	}, nil
}

func collectExcludedPaths(
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	root string,
	searchResultState *searchResultState,
	forbiddenChildMetadataPaths map[string]map[string]struct{},
	targetMergeObjectKeys map[string]struct{},
) map[string]struct{} {
	excludedPaths := make(map[string]struct{})
	if len(contexts) == 0 || len(decisions) == 0 {
		return excludedPaths
	}

	for _, ctx := range contexts {
		if ctx == nil {
			continue
		}
		if preserveTargetMergePath(ctx, targetMergeObjectKeys) {
			continue
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
		if ctx.FileName != configDumpInfo && ctx.OwnerKey != "Configuration" && !isTopLevelMetadataFile(ctx) &&
			decision.Belonging != "Native" && !searchResultPreservesPath(searchResultState, ctx.Path) {
			excludedPaths[ctx.Path] = struct{}{}
		}
	}

	collectAdoptedCodeModulePaths(contexts, decisions, excludedPaths)
	collectAdoptedCommonModuleModulePaths(root, decisions, excludedPaths)
	collectAdoptedCommandModulePaths(contexts, decisions, excludedPaths)
	collectForbiddenMetadataFilePaths(contexts, forbiddenChildMetadataPaths, excludedPaths)
	collectRootConfigurationModulePaths(root, excludedPaths)
	applySearchResultStateToExcludedPaths(searchResultState, excludedPaths)

	return excludedPaths
}

func preserveTargetMergePath(ctx *FileProcessingContext, targetMergeObjectKeys map[string]struct{}) bool {
	if ctx == nil || len(targetMergeObjectKeys) == 0 {
		return false
	}
	if ctx.OwnerKind != "ExchangePlan" || !strings.EqualFold(ctx.FileName, "Content.xml") {
		return false
	}
	_, ok := targetMergeObjectKeys[ctx.OwnerKey]
	return ok
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

		states = append(states, subsystemState{
			Key:   ctx.OwnerKey,
			Name:  ctx.OwnerName,
			Chain: chain,
		})
	}

	for _, state := range states {
		if !hasNativePrefix(state.Name, cfg.NativePrefixes) && !subsystemChainHasNativeAncestor(state.Chain, cfg.NativePrefixes) {
			continue
		}
		for idx := 1; idx < len(state.Chain); idx++ {
			adopted[subsystemChainKey(state.Chain[:idx])] = struct{}{}
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
			case containsName(adopted, subsystemChainKey(state.Chain)):
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

func subsystemChainKey(chain []string) string {
	return strings.Join(chain, ".")
}

func decideObject(
	ctx *FileProcessingContext,
	cfg *config.Configuration,
	explicitNativeObjects map[string]struct{},
	prefixNativeObjects map[string]struct{},
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

	if _, explicit := explicitNativeObjects[ctx.OwnerKey]; explicit {
		debugDecision(ctx.OwnerKey, "native by explicit include")
		return objectDecision{Belonging: "Native"}
	}

	if _, excluded := excludedObjects[ctx.OwnerKey]; excluded {
		debugDecision(ctx.OwnerKey, "soft-excluded by excluded object set")
		return objectDecision{Excluded: true}
	}

	if _, primary := prefixNativeObjects[ctx.OwnerKey]; primary {
		debugDecision(ctx.OwnerKey, "native by prefix-native set")
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

func collectGUIDReplacements(
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	identityMap *identityMapState,
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule,
) map[string]string {
	replacements := make(map[string]string)

	collectGUIDReplacementsFromConfigDump(contexts, decisions, replacements, identityMap, adoptedStubMetaDataRules)

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

func collectGUIDReplacementsFromConfigDump(
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	replacements map[string]string,
	identityMap *identityMapState,
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule,
) {
	ctx := findConfigDumpContext(contexts)
	if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
		return
	}
	if identityMap == nil {
		identityMap = newIdentityMapState()
	}

	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		if localName(el.Tag) == "Metadata" {
			name := strings.TrimSpace(el.SelectAttrValue("name", ""))
			id := strings.TrimSpace(el.SelectAttrValue("id", ""))
			if shouldTrackIdentityMetadataPath(name, decisions, adoptedStubMetaDataRules) {
				ensureIdentityReplacement(replacements, identityMap, name, id)
			} else if identityMap.Objects != nil {
				delete(identityMap.Objects, name)
			}
		}

		for _, child := range el.ChildElements() {
			walk(child)
		}
	}

	walk(ctx.Doc.Root())
}

func ensureIdentityReplacement(replacements map[string]string, identityMap *identityMapState, metadataPath, currentID string) {
	if replacements == nil || identityMap == nil {
		return
	}

	metadataPath = strings.TrimSpace(metadataPath)
	if metadataPath == "" {
		return
	}

	identityMap.ensureDefaults()
	state := identityMap.Objects[metadataPath]
	extensionID := normalizeGUIDValue(state.ExtensionID)
	if extensionID == "" {
		log.Printf("missing extension_id: metadata=%s", metadataPath)
		extensionID = newGUID()
		log.Printf("generated extension_id: metadata=%s extension_id=%s", metadataPath, extensionID)
		state.ExtensionID = extensionID
		identityMap.Objects[metadataPath] = state
	}

	for _, guid := range extractGUIDs(currentID) {
		replacements[strings.ToLower(guid)] = extensionID
	}
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

func newIdentityMapState() *identityMapState {
	state := &identityMapState{}
	state.ensureDefaults()
	return state
}

func (state *identityMapState) ensureDefaults() {
	if state == nil {
		return
	}
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Objects == nil {
		state.Objects = make(map[string]identityMapObjectBinding)
	}
}

func loadIdentityMapState(cfg *config.Configuration) (*identityMapState, error) {
	state := newIdentityMapState()
	if cfg == nil || strings.TrimSpace(cfg.IdentityMapPath) == "" {
		return state, nil
	}

	content, err := os.ReadFile(cfg.IdentityMapPath)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, fmt.Errorf("не удалось прочитать identity map %s: %w", cfg.IdentityMapPath, err)
	}

	if err := json.Unmarshal(content, state); err != nil {
		return nil, fmt.Errorf("не удалось разобрать identity map %s: %w", cfg.IdentityMapPath, err)
	}
	state.ensureDefaults()
	return state, nil
}

func saveIdentityMapState(cfg *config.Configuration, state *identityMapState) error {
	if cfg == nil || strings.TrimSpace(cfg.IdentityMapPath) == "" || state == nil {
		return nil
	}

	state.ensureDefaults()
	if err := os.MkdirAll(filepath.Dir(cfg.IdentityMapPath), 0o755); err != nil {
		return fmt.Errorf("не удалось создать каталог identity map %s: %w", filepath.Dir(cfg.IdentityMapPath), err)
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("не удалось сериализовать identity map %s: %w", cfg.IdentityMapPath, err)
	}

	if err := os.WriteFile(cfg.IdentityMapPath, append(content, '\n'), 0o644); err != nil {
		return fmt.Errorf("не удалось сохранить identity map %s: %w", cfg.IdentityMapPath, err)
	}

	return nil
}

func loadBaseBindings(cfg *config.Configuration) (map[string]string, error) {
	result := make(map[string]string)
	if cfg == nil || strings.TrimSpace(cfg.BaseBindingsPath) == "" {
		return result, nil
	}

	content, err := os.ReadFile(cfg.BaseBindingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("не удалось прочитать base bindings %s: %w", cfg.BaseBindingsPath, err)
	}

	var raw config.BaseBindingsFile
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("не удалось разобрать base bindings %s: %w", cfg.BaseBindingsPath, err)
	}

	for metadataPath, binding := range raw.Bindings {
		metadataPath = strings.TrimSpace(metadataPath)
		baseObjectID := normalizeGUIDValue(binding.BaseObjectID)
		if metadataPath == "" || baseObjectID == "" {
			continue
		}
		result[metadataPath] = baseObjectID
	}

	return result, nil
}

func collectTargetMergeRules(cfg *config.Configuration, dir string) targetMergeRuleSet {
	result := targetMergeRuleSet{ObjectKeys: make(map[string]struct{})}
	if cfg == nil || strings.TrimSpace(cfg.Target.XMLDump) == "" {
		return result
	}

	templatePath := filepath.Join(dir, "CommonTemplates", "упо_MetaDataFile", "Ext", "Template.txt")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		log.Printf("xml step: skip target merge rules, cannot read %s: %v", templatePath, err)
		return result
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("xml step: skip target merge rules, cannot parse %s: %v", templatePath, err)
		return result
	}

	root, ok := raw.(map[string]any)
	if !ok {
		return result
	}

	for rawKind, child := range root {
		kind := normalizeMetaDataFileKind(rawKind)
		if kind != "DefinedType" && kind != "ExchangePlan" && kind != "EventSubscription" {
			continue
		}
		objects, ok := child.(map[string]any)
		if !ok {
			continue
		}
		for objectName := range objects {
			objectName = strings.TrimSpace(objectName)
			if objectName == "" {
				continue
			}
			result.ObjectKeys[kind+"."+objectName] = struct{}{}
		}
	}

	return result
}

func collectTargetCompatibilitySet(cfg *config.Configuration) (targetCompatibilitySet, error) {
	result := targetCompatibilitySet{Keys: make(map[string]struct{})}
	if cfg == nil || strings.TrimSpace(cfg.Target.XMLDump) == "" {
		return result, nil
	}

	info, err := os.Stat(cfg.Target.XMLDump)
	if err != nil {
		return result, fmt.Errorf("не удалось получить доступ к XML-дампу конфигурации-приемника %s: %w", cfg.Target.XMLDump, err)
	}
	if !info.IsDir() {
		return result, fmt.Errorf("путь xml_dump должен указывать на каталог XML-выгрузки конфигурации-приемника: %s", cfg.Target.XMLDump)
	}

	result.Enabled = true
	topLevelDirs := map[string]string{
		"DefinedTypes":       "DefinedType",
		"EventSubscriptions": "EventSubscription",
		"ExchangePlans":      "ExchangePlan",
	}

	for relDir, expectedKind := range topLevelDirs {
		dirPath := filepath.Join(cfg.Target.XMLDump, relDir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return result, fmt.Errorf("не удалось прочитать каталог %s: %w", dirPath, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
				continue
			}
			relPath := filepath.ToSlash(filepath.Join(relDir, entry.Name()))
			doc, err := readXMLFile(filepath.Join(dirPath, entry.Name()))
			if err != nil {
				return result, err
			}
			kind, _, key := detectOwner(relPath, doc)
			if kind != expectedKind || key == "" {
				continue
			}
			result.Keys[key] = struct{}{}
		}
	}

	return result, nil
}

func applyTargetCompatibilitySet(
	decisions map[string]objectDecision,
	contexts []*FileProcessingContext,
	targetCompatibility targetCompatibilitySet,
	forbidden map[string]struct{},
) {
	if !targetCompatibility.Enabled || len(decisions) == 0 {
		return
	}

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || !isTargetSensitiveKind(ctx.OwnerKind) {
			continue
		}
		if _, blocked := forbidden[ctx.OwnerKey]; blocked {
			continue
		}

		decision, exists := decisions[ctx.OwnerKey]
		if !exists || decision.Excluded || decision.Belonging == "Native" {
			continue
		}
		if _, ok := targetCompatibility.Keys[ctx.OwnerKey]; ok {
			continue
		}

		decision = objectDecision{
			Excluded:         true,
			SearchResultCode: decision.SearchResultCode,
		}
		decisions[ctx.OwnerKey] = decision
		debugDecision(ctx.OwnerKey, "soft-excluded by targetCompatibilitySet")
	}
}

func isTargetSensitiveKind(kind string) bool {
	switch kind {
	case "DefinedType", "EventSubscription", "ExchangePlan":
		return true
	default:
		return false
	}
}

func isTargetSensitiveObjectKey(key string) bool {
	kind, _ := splitObjectKey(key)
	return isTargetSensitiveKind(kind)
}

func targetCompatibilityAllowsDecision(key string, decision objectDecision, targetCompatibility targetCompatibilitySet) bool {
	if !targetCompatibility.Enabled || !isTargetSensitiveObjectKey(key) {
		return true
	}
	if !decision.Excluded && decision.Belonging == "Native" {
		return true
	}
	_, ok := targetCompatibility.Keys[key]
	return ok
}

func canPromoteTargetSensitiveObject(
	key string,
	decision objectDecision,
	primaryNativeObjects map[string]struct{},
	targetCompatibility targetCompatibilitySet,
) bool {
	if !targetCompatibility.Enabled || !isTargetSensitiveObjectKey(key) {
		return true
	}
	if _, primary := primaryNativeObjects[key]; primary {
		return true
	}
	return targetCompatibilityAllowsDecision(key, decision, targetCompatibility)
}

func mergeTargetMetadataComposition(
	cfg *config.Configuration,
	root string,
	contexts []*FileProcessingContext,
	indexes *contextIndexes,
	decisions map[string]objectDecision,
	targetMergeObjectKeys map[string]struct{},
	excludedObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
	targetCompatibility targetCompatibilitySet,
) ([]*FileProcessingContext, *contextIndexes, error) {
	startedAt := time.Now()
	if cfg == nil || len(targetMergeObjectKeys) == 0 {
		return contexts, indexes, nil
	}
	if strings.TrimSpace(cfg.Target.XMLDump) == "" {
		log.Printf("xml step: skip target metadata merge xml_dump is empty")
		return contexts, indexes, nil
	}

	info, err := os.Stat(cfg.Target.XMLDump)
	if err != nil {
		return nil, nil, fmt.Errorf("не удалось получить доступ к XML-дампу конфигурации-приемника %s: %w", cfg.Target.XMLDump, err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("путь xml_dump должен указывать на каталог XML-выгрузки конфигурации-приемника: %s", cfg.Target.XMLDump)
	}
	log.Printf("xml step: target merge started merge_objects=%d", len(targetMergeObjectKeys))

	configurationCtx := findContextByOwnerKeyIndexed(indexes, contexts, "Configuration")
	configDumpCtx := findContextByRelPath(indexes, contexts, configDumpInfo)
	targetContexts := make([]*FileProcessingContext, 0, len(targetMergeObjectKeys))
	targetIndexes := buildContextIndexes(targetContexts)
	targetContextsByKey := make(map[string]*FileProcessingContext, len(targetMergeObjectKeys))
	targetMissingKeys := make(map[string]struct{})
	targetContextsByRelPath := make(map[string]*FileProcessingContext, len(targetMergeObjectKeys))
	targetMissingRelPaths := make(map[string]struct{})
	collectedTargetRefs := make(map[string]struct{})
	mergeObjects := make([]preparedTargetMergeObject, 0, len(targetMergeObjectKeys))
	dirtyDocs := make(map[string]*FileProcessingContext)
	stats := targetMergePerfStats{MergeObjects: len(targetMergeObjectKeys)}
	configurationDirty := false
	configDumpDirty := false

	loadTargetContextByRelPath := func(relPath string) (*FileProcessingContext, error) {
		if relPath == "" {
			return nil, nil
		}
		relPath = filepath.ToSlash(relPath)
		if existing, ok := targetContextsByRelPath[relPath]; ok {
			stats.CacheHits++
			return existing, nil
		}
		if _, missing := targetMissingRelPaths[relPath]; missing {
			stats.CacheHits++
			return nil, nil
		}
		stats.CacheMisses++

		path := filepath.Join(cfg.Target.XMLDump, filepath.FromSlash(relPath))
		info, err := os.Stat(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				targetMissingRelPaths[relPath] = struct{}{}
				return nil, nil
			}
			return nil, fmt.Errorf("ошибка при доступе к файлу %s: %w", path, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("ожидался XML-файл, а не каталог: %s", path)
		}

		doc, err := readXMLFile(path)
		if err != nil {
			return nil, err
		}

		kind, name, key := detectOwner(relPath, doc)
		ctx := &FileProcessingContext{
			Doc:              doc,
			Path:             path,
			RelPath:          filepath.ToSlash(relPath),
			FileName:         filepath.Base(path),
			Metadata:         isMetadataObjectDoc(doc),
			TopLevelMetadata: isTopLevelMetadataRelPath(relPath),
			Properties:       findProperties(doc),
			OwnerKind:        kind,
			OwnerName:        name,
			OwnerKey:         key,
		}
		targetContexts = append(targetContexts, ctx)
		appendContextToIndexes(targetIndexes, ctx)
		targetContextsByRelPath[ctx.RelPath] = ctx
		if ctx.OwnerKey != "" {
			targetContextsByKey[ctx.OwnerKey] = ctx
		}
		return ctx, nil
	}

	loadTargetTopLevelContextByKey := func(key string) (*FileProcessingContext, error) {
		if existing, ok := targetContextsByKey[key]; ok {
			stats.CacheHits++
			return existing, nil
		}
		if _, missing := targetMissingKeys[key]; missing {
			stats.CacheHits++
			return nil, nil
		}
		relPath, ok := topLevelMetadataRelPathForKey(key)
		if !ok {
			targetMissingKeys[key] = struct{}{}
			return nil, nil
		}
		ctx, err := loadTargetContextByRelPath(relPath)
		if err != nil {
			return nil, err
		}
		if ctx == nil {
			targetMissingKeys[key] = struct{}{}
			return nil, nil
		}
		if ctx.OwnerKey != key {
			return nil, fmt.Errorf("ожидался объект %s в XML-дампе конфигурации-приемника, получен %s", key, ctx.OwnerKey)
		}
		targetContextsByKey[key] = ctx
		return ctx, nil
	}

	markDirty := func(ctx *FileProcessingContext) {
		if ctx == nil || ctx.Path == "" {
			return
		}
		dirtyDocs[ctx.Path] = ctx
	}

	ensureTargetTopLevelObjectImported := func(key string, currentCtx *FileProcessingContext, targetCtx *FileProcessingContext) (*FileProcessingContext, bool, error) {
		if targetCtx == nil || targetCtx.Doc == nil || !isTopLevelMetadataFile(targetCtx) {
			return nil, false, fmt.Errorf("не удалось найти объект %s в XML-дампе конфигурации-приемника", key)
		}

		imported := false
		if currentCtx == nil || currentCtx.Doc == nil {
			var err error
			currentCtx, err = cloneTopLevelContextIntoRoot(root, targetCtx)
			if err != nil {
				return nil, false, err
			}
			contexts = append(contexts, currentCtx)
			appendContextToIndexes(indexes, currentCtx)
			imported = true
		}

		decisions[key] = objectDecision{
			Belonging: "AdoptedStub",
			Truncated: shouldTruncateAdoptedStub(currentCtx),
		}

		if configurationCtx != nil && configurationCtx.Doc != nil {
			changed := ensureConfigurationChildObject(configurationCtx.Doc, key)
			if changed {
				configurationDirty = true
			}
		}

		if configDumpCtx != nil && configDumpCtx.Doc != nil {
			changed, err := ensureConfigDumpInfoMetadataEntry(configDumpCtx.Doc, key, currentCtx, targetCtx)
			if err != nil {
				return nil, false, err
			}
			if changed {
				configDumpDirty = true
			}
		}

		return currentCtx, imported, nil
	}

	collectTargetValueRefs := func(value string) (bool, error) {
		refs := metadataReferencesFromValue(value)
		if len(refs) == 0 {
			return true, nil
		}

		for _, ref := range refs {
			if _, forbidden := forbiddenAdoptedStubObjects[ref]; forbidden {
				return false, nil
			}
			if _, excluded := excludedObjects[ref]; excluded {
				return false, nil
			}
			if decision, exists := decisions[ref]; exists {
				if decision.Excluded {
					return false, nil
				}
				continue
			}

			targetCtx, err := loadTargetTopLevelContextByKey(ref)
			if err != nil {
				return false, err
			}
			if targetCtx == nil {
				return false, nil
			}
			collectedTargetRefs[ref] = struct{}{}
		}

		return true, nil
	}

	allowCollectedTargetValue := func(value string) (bool, error) {
		refs := metadataReferencesFromValue(value)
		if len(refs) == 0 {
			return true, nil
		}
		for _, ref := range refs {
			if _, forbidden := forbiddenAdoptedStubObjects[ref]; forbidden {
				return false, nil
			}
			if _, excluded := excludedObjects[ref]; excluded {
				return false, nil
			}
			decision, exists := decisions[ref]
			if !exists || decision.Excluded {
				return false, nil
			}
		}
		return true, nil
	}

	keys := make([]string, 0, len(targetMergeObjectKeys))
	for key := range targetMergeObjectKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if _, forbidden := forbiddenAdoptedStubObjects[key]; forbidden {
			continue
		}

		targetCtx, err := loadTargetTopLevelContextByKey(key)
		if err != nil {
			return nil, nil, err
		}
		if targetCtx == nil || targetCtx.Doc == nil {
			continue
		}
		currentCtx := findContextByOwnerKeyIndexed(indexes, contexts, key)
		decision, exists := decisions[key]
		if !exists || decision.Excluded || currentCtx == nil || currentCtx.Doc == nil {
			if _, compatible := targetCompatibility.Keys[key]; !compatible {
				continue
			}
			currentCtx, _, err = ensureTargetTopLevelObjectImported(key, currentCtx, targetCtx)
			if err != nil {
				return nil, nil, err
			}
			decision = decisions[key]
			exists = true
		}
		if !exists || decision.Excluded || currentCtx == nil || currentCtx.Doc == nil {
			continue
		}
		if !targetCompatibilityAllowsDecision(key, decision, targetCompatibility) {
			continue
		}
		decision = preserveAdoptedStubExtMetaData(decision)
		decisions[key] = decision

		kind, _ := splitObjectKey(key)
		prepared := preparedTargetMergeObject{
			Key:        key,
			Kind:       kind,
			CurrentCtx: currentCtx,
			TargetCtx:  targetCtx,
		}
		switch kind {
		case "DefinedType":
			targetType := targetCtx.Properties.FindElement("./Type")
			if err := collectMetadataValueContainerRefs(targetType, collectTargetValueRefs); err != nil {
				return nil, nil, err
			}
		case "EventSubscription":
			targetSource := targetCtx.Properties.FindElement("./Source")
			if err := collectMetadataValueContainerRefs(targetSource, collectTargetValueRefs); err != nil {
				return nil, nil, err
			}
		case "ExchangePlan":
			contentRelPath := strings.TrimSuffix(currentCtx.RelPath, ".xml") + "/Ext/Content.xml"
			targetContentCtx, err := loadTargetContextByRelPath(contentRelPath)
			if err != nil {
				return nil, nil, err
			}
			if targetContentCtx == nil || targetContentCtx.Doc == nil {
				break
			}
			currentContentCtx := findContextByRelPath(indexes, contexts, contentRelPath)
			if currentContentCtx == nil {
				currentContentCtx, err = cloneContextIntoRoot(root, targetContentCtx)
				if err != nil {
					return nil, nil, err
				}
				contexts = append(contexts, currentContentCtx)
				appendContextToIndexes(indexes, currentContentCtx)
			}
			prepared.ExchangePlanRelPath = contentRelPath
			prepared.CurrentContentCtx = currentContentCtx
			prepared.TargetContentCtx = targetContentCtx
			if err := collectExchangePlanContentRefs(targetContentCtx.Doc, collectTargetValueRefs); err != nil {
				return nil, nil, err
			}
		}
		mergeObjects = append(mergeObjects, prepared)
	}

	stats.TargetRefs = len(collectedTargetRefs)
	importKeys := make([]string, 0, len(collectedTargetRefs))
	for key := range collectedTargetRefs {
		importKeys = append(importKeys, key)
	}
	sort.Strings(importKeys)

	for _, key := range importKeys {
		if _, forbidden := forbiddenAdoptedStubObjects[key]; forbidden {
			stats.SkippedTargetRefs++
			continue
		}
		if _, excluded := excludedObjects[key]; excluded {
			stats.SkippedTargetRefs++
			continue
		}

		currentCtx := findContextByOwnerKeyIndexed(indexes, contexts, key)
		decision, exists := decisions[key]
		if exists && decision.Excluded {
			stats.SkippedTargetRefs++
			continue
		} else if exists && currentCtx != nil && currentCtx.Doc != nil {
			stats.SkippedTargetRefs++
			continue
		}
		if isTargetSensitiveObjectKey(key) {
			stats.SkippedTargetRefs++
			continue
		}

		targetCtx, err := loadTargetTopLevelContextByKey(key)
		if err != nil {
			return nil, nil, err
		}
		if targetCtx == nil {
			stats.SkippedTargetRefs++
			continue
		}

		_, imported, err := ensureTargetTopLevelObjectImported(key, currentCtx, targetCtx)
		if err != nil {
			return nil, nil, err
		}
		if imported {
			stats.ImportedTargetRefs++
			continue
		}
		stats.SkippedTargetRefs++
	}

	for _, prepared := range mergeObjects {
		switch prepared.Kind {
		case "DefinedType":
			currentType := prepared.CurrentCtx.Properties.FindElement("./Type")
			targetType := prepared.TargetCtx.Properties.FindElement("./Type")
			changed, err := mergeMetadataValueContainer(prepared.CurrentCtx.Properties, "Type", currentType, targetType, decisions, allowCollectedTargetValue)
			if err != nil {
				return nil, nil, err
			}
			if changed {
				markDirty(prepared.CurrentCtx)
			}
		case "EventSubscription":
			currentSource := prepared.CurrentCtx.Properties.FindElement("./Source")
			targetSource := prepared.TargetCtx.Properties.FindElement("./Source")
			changed, err := mergeMetadataValueContainer(prepared.CurrentCtx.Properties, "Source", currentSource, targetSource, decisions, allowCollectedTargetValue)
			if err != nil {
				return nil, nil, err
			}
			if changed {
				markDirty(prepared.CurrentCtx)
			}
		case "ExchangePlan":
			if prepared.CurrentContentCtx == nil || prepared.TargetContentCtx == nil {
				continue
			}
			changed, err := mergeExchangePlanContent(prepared.CurrentContentCtx.Doc, prepared.TargetContentCtx.Doc, decisions, allowCollectedTargetValue)
			if err != nil {
				return nil, nil, err
			}
			if changed {
				markDirty(prepared.CurrentContentCtx)
			}
		}
	}

	dirtyPaths := make([]string, 0, len(dirtyDocs))
	for path := range dirtyDocs {
		dirtyPaths = append(dirtyPaths, path)
	}
	sort.Strings(dirtyPaths)
	for _, path := range dirtyPaths {
		ctx := dirtyDocs[path]
		if err := ctx.Doc.WriteToFile(ctx.Path); err != nil {
			return nil, nil, fmt.Errorf("ошибка при записи файла %s: %w", ctx.Path, err)
		}
	}

	if configurationDirty && configurationCtx != nil && configurationCtx.Doc != nil {
		if err := configurationCtx.Doc.WriteToFile(configurationCtx.Path); err != nil {
			return nil, nil, fmt.Errorf("ошибка при записи файла %s: %w", configurationCtx.Path, err)
		}
	}
	if configDumpDirty && configDumpCtx != nil && configDumpCtx.Doc != nil {
		if err := configDumpCtx.Doc.WriteToFile(configDumpCtx.Path); err != nil {
			return nil, nil, fmt.Errorf("ошибка при записи файла %s: %w", configDumpCtx.Path, err)
		}
	}

	log.Printf(
		"xml step: target merge completed in %s (merge_objects=%d target_refs=%d imported=%d skipped=%d cache_hits=%d cache_misses=%d)",
		time.Since(startedAt).Round(100*time.Millisecond),
		stats.MergeObjects,
		stats.TargetRefs,
		stats.ImportedTargetRefs,
		stats.SkippedTargetRefs,
		stats.CacheHits,
		stats.CacheMisses,
	)

	return contexts, indexes, nil
}

func collectMetadataValueContainerRefs(
	targetContainer *etree.Element,
	allowTargetValue func(string) (bool, error),
) error {
	if targetContainer == nil {
		return nil
	}
	for _, child := range targetContainer.ChildElements() {
		value := strings.TrimSpace(child.Text())
		if value == "" {
			continue
		}
		if _, err := allowTargetValue(value); err != nil {
			return err
		}
	}
	return nil
}

func collectExchangePlanContentRefs(
	targetDoc *etree.Document,
	allowTargetValue func(string) (bool, error),
) error {
	if targetDoc == nil || targetDoc.Root() == nil {
		return nil
	}
	for _, item := range targetDoc.Root().FindElements("./Item") {
		metadata := strings.TrimSpace(textOfFirst(item, "./Metadata"))
		if metadata == "" {
			continue
		}
		if _, err := allowTargetValue(metadata); err != nil {
			return err
		}
	}
	return nil
}

func mergeMetadataValueContainer(
	properties *etree.Element,
	containerName string,
	currentContainer *etree.Element,
	targetContainer *etree.Element,
	decisions map[string]objectDecision,
	allowTargetValue func(string) (bool, error),
) (bool, error) {
	if properties == nil || targetContainer == nil {
		return false, nil
	}
	if currentContainer == nil {
		currentContainer = properties.CreateElement(targetContainer.Tag)
	}

	changed := false
	existingValues := make(map[string]struct{})
	for _, child := range append([]*etree.Element(nil), currentContainer.ChildElements()...) {
		value := strings.TrimSpace(child.Text())
		if !shouldKeepCurrentMergedMetadataValue(value, decisions) {
			currentContainer.RemoveChild(child)
			changed = true
			continue
		}
		existingValues[value] = struct{}{}
	}

	for _, child := range targetContainer.ChildElements() {
		value := strings.TrimSpace(child.Text())
		if value == "" {
			continue
		}
		allowed, err := allowTargetValue(value)
		if err != nil {
			return false, err
		}
		if !allowed {
			continue
		}
		if _, exists := existingValues[value]; exists {
			continue
		}
		currentContainer.AddChild(child.Copy())
		existingValues[value] = struct{}{}
		changed = true
	}

	if len(currentContainer.ChildElements()) == 0 && currentContainer.Parent() != nil && localName(currentContainer.Tag) == containerName {
		currentContainer.Parent().RemoveChild(currentContainer)
		changed = true
	}

	return changed, nil
}

func shouldKeepCurrentMergedMetadataValue(value string, decisions map[string]objectDecision) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	refs := metadataReferencesFromValue(value)
	if len(refs) == 0 {
		return true
	}

	for _, ref := range refs {
		decision, exists := decisions[ref]
		if !exists || decision.Excluded {
			return false
		}
	}

	return true
}

func mergeExchangePlanContent(
	currentDoc *etree.Document,
	targetDoc *etree.Document,
	decisions map[string]objectDecision,
	allowTargetValue func(string) (bool, error),
) (bool, error) {
	currentRoot := currentDoc.Root()
	targetRoot := targetDoc.Root()
	if currentRoot == nil || targetRoot == nil {
		return false, nil
	}

	changed := false
	existingValues := make(map[string]struct{})
	for _, item := range append([]*etree.Element(nil), currentRoot.FindElements("./Item")...) {
		metadata := strings.TrimSpace(textOfFirst(item, "./Metadata"))
		if !shouldKeepCurrentMergedMetadataValue(metadata, decisions) {
			currentRoot.RemoveChild(item)
			changed = true
			continue
		}
		existingValues[metadata] = struct{}{}
	}

	for _, item := range targetRoot.FindElements("./Item") {
		metadata := strings.TrimSpace(textOfFirst(item, "./Metadata"))
		if metadata == "" {
			continue
		}
		allowed, err := allowTargetValue(metadata)
		if err != nil {
			return false, err
		}
		if !allowed {
			continue
		}
		if _, exists := existingValues[metadata]; exists {
			continue
		}
		currentRoot.AddChild(item.Copy())
		existingValues[metadata] = struct{}{}
		changed = true
	}

	return changed, nil
}

func cloneTopLevelContextIntoRoot(root string, source *FileProcessingContext) (*FileProcessingContext, error) {
	ctx, err := cloneContextIntoRoot(root, source)
	if err != nil {
		return nil, err
	}
	if !ctx.Metadata || !isTopLevelMetadataFile(ctx) {
		return nil, fmt.Errorf("объект %s в target.xml_dump не является top-level metadata", source.OwnerKey)
	}
	return ctx, nil
}

func cloneContextIntoRoot(root string, source *FileProcessingContext) (*FileProcessingContext, error) {
	if source == nil || source.Doc == nil {
		return nil, fmt.Errorf("не удалось скопировать пустой XML-контекст из target.xml_dump")
	}

	path := filepath.Join(root, filepath.FromSlash(source.RelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("не удалось создать каталог для target-ref-driven объекта %s: %w", filepath.Dir(path), err)
	}

	doc := source.Doc.Copy()
	if err := doc.WriteToFile(path); err != nil {
		return nil, fmt.Errorf("ошибка при записи target-ref-driven объекта %s: %w", path, err)
	}

	target := metadataTargetElement(doc)
	var properties *etree.Element
	if target != nil {
		properties = target.FindElement("./Properties")
	}

	return &FileProcessingContext{
		Doc:              doc,
		Path:             path,
		RelPath:          source.RelPath,
		FileName:         source.FileName,
		Metadata:         source.Metadata,
		TopLevelMetadata: source.TopLevelMetadata,
		Properties:       properties,
		OwnerKey:         source.OwnerKey,
		OwnerKind:        source.OwnerKind,
		OwnerName:        source.OwnerName,
	}, nil
}

func appendContextToIndexes(indexes *contextIndexes, ctx *FileProcessingContext) {
	if indexes == nil || ctx == nil {
		return
	}
	if indexes.byOwnerKey == nil {
		indexes.byOwnerKey = make(map[string][]*FileProcessingContext)
	}
	if indexes.byRelPath == nil {
		indexes.byRelPath = make(map[string]*FileProcessingContext)
	}
	if indexes.byPath == nil {
		indexes.byPath = make(map[string]*FileProcessingContext)
	}
	if indexes.byFileName == nil {
		indexes.byFileName = make(map[string][]*FileProcessingContext)
	}

	indexes.byOwnerKey[ctx.OwnerKey] = append(indexes.byOwnerKey[ctx.OwnerKey], ctx)
	indexes.byRelPath[ctx.RelPath] = ctx
	indexes.byPath[ctx.Path] = ctx
	indexes.byFileName[ctx.FileName] = append(indexes.byFileName[ctx.FileName], ctx)
}

func ensureConfigDumpInfoMetadataEntry(doc *etree.Document, metadataName string, candidateContexts ...*FileProcessingContext) (bool, error) {
	if doc == nil {
		return false, nil
	}

	configVersions := findConfigVersionsElement(doc)
	if configVersions == nil {
		return false, fmt.Errorf("не удалось обновить ConfigDumpInfo.xml для %s", metadataName)
	}

	if findConfigDumpMetadataEntry(configVersions, metadataName) != nil {
		return false, nil
	}

	metadataID := ""
	for _, ctx := range candidateContexts {
		metadataID = metadataUUIDForConfigDumpEntry(ctx)
		if metadataID != "" {
			break
		}
	}
	if metadataID == "" {
		return false, fmt.Errorf("не удалось определить uuid для записи %s в ConfigDumpInfo.xml", metadataName)
	}

	entry := etree.NewElement("Metadata")
	entry.CreateAttr("name", metadataName)
	entry.CreateAttr("id", metadataID)
	configVersions.AddChild(entry)
	return true, nil
}

func metadataUUIDForConfigDumpEntry(ctx *FileProcessingContext) string {
	if ctx == nil || ctx.Doc == nil {
		return ""
	}
	target := metadataTargetElement(ctx.Doc)
	if target == nil {
		return ""
	}
	return normalizeGUIDValue(target.SelectAttrValue("uuid", ""))
}

func ensureConfigurationChildObject(doc *etree.Document, metadataName string) bool {
	target := metadataTargetElement(doc)
	if target == nil {
		return false
	}

	kind, name := splitObjectKey(metadataName)
	if kind == "" || name == "" {
		return false
	}

	tag, ok := configurationChildObjectTag(kind)
	if !ok {
		return false
	}

	childObjects := target.FindElement("./ChildObjects")
	if childObjects == nil {
		childObjects = target.CreateElement("ChildObjects")
	}

	for _, child := range childObjects.ChildElements() {
		if strings.EqualFold(localName(child.Tag), tag) && strings.TrimSpace(child.Text()) == name {
			return false
		}
	}

	childObjects.CreateElement(tag).SetText(name)
	return true
}

func findConfigVersionsElement(doc *etree.Document) *etree.Element {
	if doc == nil || doc.Root() == nil {
		return nil
	}
	return doc.Root().FindElement("./ConfigVersions")
}

func findConfigDumpMetadataEntry(configVersions *etree.Element, metadataName string) *etree.Element {
	if configVersions == nil {
		return nil
	}
	for _, child := range configVersions.ChildElements() {
		if !strings.EqualFold(localName(child.Tag), "Metadata") {
			continue
		}
		if strings.TrimSpace(child.SelectAttrValue("name", "")) == metadataName {
			return child
		}
	}
	return nil
}

func topLevelMetadataRelPathForKey(key string) (string, bool) {
	kind, name := splitObjectKey(key)
	if kind == "" || name == "" {
		return "", false
	}

	for dir, candidateKind := range metadataKinds {
		if candidateKind != kind {
			continue
		}
		return filepath.ToSlash(filepath.Join(dir, name+".xml")), true
	}

	return "", false
}

func normalizeGUIDValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	guid := extractGUIDs(value)
	if len(guid) == 0 {
		return ""
	}
	return strings.ToLower(guid[0])
}

func shouldTrackIdentityMetadataPath(
	metadataPath string,
	decisions map[string]objectDecision,
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule,
) bool {
	topKey, segments, ok := parseMetadataPath(metadataPath)
	if !ok || topKey == "Configuration" || topKey == "Language.Русский" {
		return false
	}

	decision, exists := decisions[topKey]
	if !exists || decision.Excluded || decision.Belonging == "Native" {
		return false
	}

	if len(segments) == 0 {
		return true
	}

	switch len(segments) {
	case 1:
		if segments[0].Kind != "Attribute" && segments[0].Kind != "TabularSection" && segments[0].Kind != "Command" {
			return false
		}
	case 2:
		if segments[0].Kind != "TabularSection" || segments[1].Kind != "Attribute" {
			return false
		}
	default:
		return false
	}

	if retainedAsNativeMetadataPath(topKey, segments, adoptedStubMetaDataRules) {
		return false
	}

	return true
}

func parseMetadataPath(metadataPath string) (string, []metadataPathSegment, bool) {
	parts := strings.Split(strings.TrimSpace(metadataPath), ".")
	if len(parts) < 2 || len(parts)%2 != 0 {
		return "", nil, false
	}

	topKey := strings.TrimSpace(parts[0]) + "." + strings.TrimSpace(parts[1])
	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", nil, false
	}

	segments := make([]metadataPathSegment, 0, (len(parts)-2)/2)
	for idx := 2; idx < len(parts); idx += 2 {
		kind := strings.TrimSpace(parts[idx])
		name := strings.TrimSpace(parts[idx+1])
		if kind == "" || name == "" {
			return "", nil, false
		}
		segments = append(segments, metadataPathSegment{Kind: kind, Name: name})
	}

	return topKey, segments, true
}

func retainedAsNativeMetadataPath(
	topKey string,
	segments []metadataPathSegment,
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule,
) bool {
	rule, ok := adoptedStubMetaDataRules[topKey]
	if !ok || len(segments) == 0 {
		return false
	}

	if len(segments) == 1 && segments[0].Kind == "Attribute" {
		_, keep := rule.NativeAttributes[segments[0].Name]
		return keep
	}

	if len(segments) == 1 && segments[0].Kind == "TabularSection" {
		_, keep := rule.NativeTabularSections[segments[0].Name]
		return keep
	}

	if len(segments) == 2 && segments[0].Kind == "TabularSection" && segments[1].Kind == "Attribute" {
		attrs := rule.NativeTabularSections[segments[0].Name]
		_, keep := attrs[segments[1].Name]
		return keep
	}

	return false
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

func collectForbiddenChildMetadataPaths(forbidden map[string]struct{}) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	for path := range forbidden {
		ownerKey := metadataDecisionKey(path)
		if ownerKey == "" || ownerKey == path {
			continue
		}
		if result[ownerKey] == nil {
			result[ownerKey] = make(map[string]struct{})
		}
		result[ownerKey][path] = struct{}{}
	}
	return result
}

func isForbiddenMetadataPath(forbidden map[string]struct{}, path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || len(forbidden) == 0 {
		return false
	}
	for key := range forbidden {
		if key == path || strings.HasPrefix(key, path+".") || strings.HasPrefix(path, key+".") {
			return true
		}
	}
	return false
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

func findContextByRelPath(indexes *contextIndexes, contexts []*FileProcessingContext, relPath string) *FileProcessingContext {
	if indexes != nil && relPath != "" {
		if ctx := indexes.byRelPath[relPath]; ctx != nil {
			return ctx
		}
	}
	for _, ctx := range contexts {
		if ctx == nil || ctx.RelPath != relPath {
			continue
		}
		return ctx
	}
	return nil
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

func preserveAdoptedStubExtMetaData(decision objectDecision) objectDecision {
	decision.Belonging = "AdoptedStub"
	decision.Excluded = false
	decision.Truncated = false
	decision.AdoptedStubExtMetaData = true
	return decision
}

func isAdoptedStubExtMetaData(ctx *FileProcessingContext, decision objectDecision) bool {
	if !decision.AdoptedStubExtMetaData || decision.Belonging != "AdoptedStub" {
		return false
	}
	if ctx == nil {
		return false
	}
	switch ctx.OwnerKind {
	case "DefinedType", "ExchangePlan", "EventSubscription":
		return true
	default:
		return false
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
	promoteReferencedObjectsToAdoptedStubIndexed(contexts, nil, decisions, cfg, referenceGraph, incomingReferenceGraph, adoptedStubExtReferenceGraph, primaryNativeObjects, excludedObjects, forbiddenAdoptedStubObjects, nil, targetCompatibilitySet{})
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
	targetMergeObjectKeys map[string]struct{},
	targetCompatibility targetCompatibilitySet,
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
			regularSource := isRefDrivenInclusionSource(ctx, decision)
			if regularSource && !targetCompatibilityAllowsDecision(ctx.OwnerKey, decision, targetCompatibility) {
				regularSource = false
			}
			if regularSource && shouldDeferTargetMergeSource(ctx, targetMergeObjectKeys) {
				regularSource = false
			}
			functionalOptionStorageSource := isFunctionalOptionStorageReferenceSource(ctx, decision)
			if !regularSource && !functionalOptionStorageSource {
				continue
			}

			refs := referenceGraph[ctx.OwnerKey]
			if !regularSource && functionalOptionStorageSource {
				refs = collectFunctionalOptionStorageReferences(ctx.Doc)
			}
			adoptedStubExtRefs := adoptedStubExtReferenceGraph[ctx.OwnerKey]
			if ctx.OwnerKey == "Configuration" {
				refs = collectConfigurationChildObjectReferences(ctx.Doc.Root(), primaryNativeObjects)
			}

			for ref := range refs {
				if ref == "" || ref == ctx.OwnerKey {
					continue
				}

				refDecision, refExists := decisions[ref]
				if !canPromoteTargetSensitiveObject(ref, refDecision, primaryNativeObjects, targetCompatibility) {
					debugDecision(ref, "kept excluded: targetCompatibilitySet")
					continue
				}
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
	if ctx != nil && ctx.OwnerKind == "Subsystem" {
		return decision.Belonging == "Native"
	}
	if decision.Belonging == "Native" {
		return true
	}
	if decision.SearchResultCode && decision.Belonging == "AdoptedStub" && !decision.Truncated {
		return true
	}

	// Ненативные определяемые типы и подписки на события остаются источником RefDrivenInclusion,
	// если они уже перенесены как AdoptedStubExt и сохранили состав.
	return ctx != nil &&
		(ctx.OwnerKind == "DefinedType" || ctx.OwnerKind == "EventSubscription") &&
		decision.Belonging == "AdoptedStub" &&
		!decision.Truncated
}

func shouldDeferTargetMergeSource(ctx *FileProcessingContext, targetMergeObjectKeys map[string]struct{}) bool {
	if ctx == nil || len(targetMergeObjectKeys) == 0 {
		return false
	}
	if ctx.OwnerKind != "DefinedType" && ctx.OwnerKind != "ExchangePlan" && ctx.OwnerKind != "EventSubscription" {
		return false
	}
	_, ok := targetMergeObjectKeys[ctx.OwnerKey]
	return ok
}

func isFunctionalOptionStorageReferenceSource(ctx *FileProcessingContext, decision objectDecision) bool {
	return ctx != nil &&
		ctx.OwnerKind == "FunctionalOption" &&
		decision.Belonging == "AdoptedStub" &&
		!decision.Excluded
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

func collectFunctionalOptionStorageReferences(doc *etree.Document) map[string]struct{} {
	result := make(map[string]struct{})
	target := metadataTargetElement(doc)
	if target == nil || !strings.EqualFold(localName(target.Tag), "FunctionalOption") {
		return result
	}

	properties := target.FindElement("./Properties")
	if properties == nil {
		return result
	}

	for _, tag := range []string{"Location", "Storage"} {
		el := properties.FindElement("./" + tag)
		if el == nil {
			continue
		}
		for ref := range collectMetadataReferences(el) {
			result[ref] = struct{}{}
		}
	}

	return result
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

func collectReferenceGraph(contexts []*FileProcessingContext, cfg *config.Configuration, primaryNativeObjects map[string]struct{}, decisions map[string]objectDecision, searchResultState *searchResultState) map[string]map[string]struct{} {
	graph := make(map[string]map[string]struct{})
	styleItemKeys := collectExistingOwnerKeys(contexts, "StyleItem")

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}
		decision, hasDecision := decisions[ctx.OwnerKey]
		if hasDecision && decision.SearchResultCode && decision.Belonging != "Native" &&
			!isTopLevelMetadataFile(ctx) && !searchResultPreservesPath(searchResultState, ctx.Path) {
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

func collectAdoptedStubExtReferenceGraph(contexts []*FileProcessingContext, decisions map[string]objectDecision, searchResultState *searchResultState) map[string]map[string]struct{} {
	graph := make(map[string]map[string]struct{})

	for _, ctx := range contexts {
		if ctx == nil || ctx.OwnerKey == "" || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}
		if strings.Contains(ctx.RelPath, "/Forms/") {
			decision, ok := decisions[ctx.OwnerKey]
			if !ok || decision.Excluded {
				continue
			}
			if decision.Belonging != "Native" && !(decision.SearchResultCode && !decision.Truncated && searchResultPreservesPath(searchResultState, ctx.Path)) {
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

func collectFormDynamicListContracts(contexts []*FileProcessingContext, decisions map[string]objectDecision, searchResultState *searchResultState) map[string]formDynamicListContract {
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
		if !ok || decision.Excluded {
			continue
		}
		if decision.Belonging != "Native" && !(decision.SearchResultCode && !decision.Truncated && searchResultPreservesPath(searchResultState, ctx.Path)) {
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
	attrName := strings.TrimSpace(attr.SelectAttrValue("name", ""))

	for _, field := range attr.FindElements(".//Field") {
		if len(field.ChildElements()) == 0 {
			name := strings.TrimSpace(field.Text())
			if normalized, ok := extractDynamicListFieldName(name, attrName); ok {
				name = normalized
			}
			if name != "" {
				result[name] = struct{}{}
			}
			continue
		}
		for _, child := range field.ChildElements() {
			tag := localName(child.Tag)
			if !strings.EqualFold(tag, "dataPath") && !strings.EqualFold(tag, "field") {
				continue
			}
			name := strings.TrimSpace(child.Text())
			if normalized, ok := extractDynamicListFieldName(name, attrName); ok {
				name = normalized
			}
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
			decisions[key] = objectDecision{Belonging: "AdoptedStub", SearchResultCode: decision.SearchResultCode}
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

type searchResultPlaceRequest struct {
	Place  string
	Groups []string
}

type searchResultParsedMethod struct {
	Name           string
	IsFunction     bool
	DirectiveLines []string
	HeaderLine     string
	BodyLines      []string
	GeneratedName  string
	DirectTransfer bool
}

func newSearchResultState() *searchResultState {
	return &searchResultState{
		ObjectOverlays:          make(map[string]searchResultObjectOverlay),
		ModuleWrites:            make(map[string]searchResultModuleWrite),
		ExpectedAdoptedObjects:  make(map[string]struct{}),
		PreservedPaths:          make(map[string]struct{}),
		PreservedConfigDumpInfo: make(map[string]struct{}),
	}
}

func searchResultPreservesPath(state *searchResultState, path string) bool {
	if state == nil || len(state.PreservedPaths) == 0 {
		return false
	}
	_, ok := state.PreservedPaths[path]
	return ok
}

func searchResultObjectOverlayForKey(state *searchResultState, key string) searchResultObjectOverlay {
	if state == nil || key == "" {
		return searchResultObjectOverlay{}
	}
	return state.ObjectOverlays[key]
}

func mergeSearchResultOverlayCommands(overlay searchResultObjectOverlay, retained map[string]struct{}) searchResultObjectOverlay {
	if len(retained) == 0 {
		return overlay
	}
	if overlay.PreserveCommands == nil {
		overlay.PreserveCommands = make(map[string]struct{}, len(retained))
	}
	for name := range retained {
		overlay.PreserveCommands[name] = struct{}{}
	}
	return overlay
}

func applySearchResultStateToExcludedPaths(state *searchResultState, excludedPaths map[string]struct{}) {
	if state == nil || excludedPaths == nil {
		return
	}
	for path := range state.PreservedPaths {
		delete(excludedPaths, path)
	}
}

func collectSearchResultState(
	cfg *config.Configuration,
	dir string,
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	primaryNativeObjects map[string]struct{},
	excludedObjects map[string]struct{},
	forbiddenAdoptedStubObjects map[string]struct{},
) (*searchResultState, error) {
	state := newSearchResultState()
	if cfg == nil || !cfg.IsSearchResultEnabled() {
		return state, nil
	}

	markerGroups, err := loadSearchResultMarkerGroups(cfg)
	if err != nil {
		return nil, err
	}
	if len(markerGroups) == 0 {
		return state, nil
	}

	placeRequests, err := collectSearchResultPlaceRequests(dir, markerGroups)
	if err != nil {
		return nil, err
	}
	if len(placeRequests) == 0 {
		return state, nil
	}

	for key, places := range placeRequests {
		if _, forbidden := forbiddenAdoptedStubObjects[key]; forbidden {
			continue
		}
		if _, excluded := excludedObjects[key]; excluded {
			continue
		}
		if _, primary := primaryNativeObjects[key]; primary {
			continue
		}

		decision := decisions[key]
		if decision.Belonging == "Native" {
			continue
		}

		topCtx := findTopLevelMetadataContextByOwnerKeyIndexed(nil, contexts, key)
		if topCtx == nil {
			return nil, fmt.Errorf("упо_SearchResult ссылается на отсутствующий объект %s", key)
		}

		if decision.Excluded || decision.Belonging == "" {
			decisions[key] = objectDecision{Belonging: "AdoptedStub", SearchResultCode: true}
			debugDecision(key, "kept as AdoptedStubCode from CommonTemplate упо_SearchResult")
		} else if decision.Truncated {
			decision.Truncated = false
			decision.SearchResultCode = true
			decisions[key] = decision
			debugDecision(key, "expanded to AdoptedStubCode from CommonTemplate упо_SearchResult")
		} else {
			decision.SearchResultCode = true
			decisions[key] = decision
		}
		state.ExpectedAdoptedObjects[key] = struct{}{}

		for _, place := range places {
			if err := registerSearchResultPlace(state, dir, topCtx, place, markerGroups, cfg.ExtensionPrefix(), cfg.IsExactSearchResultTemplatesEnabled(), searchResultDiagnosticsPath(cfg)); err != nil {
				return nil, err
			}
		}
	}

	return state, nil
}

func validateSearchResultAdoptedObjects(
	indexes *contextIndexes,
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	excludedPaths map[string]struct{},
	state *searchResultState,
) error {
	if state == nil || len(state.ExpectedAdoptedObjects) == 0 {
		return nil
	}

	log.Printf("xml step: validate search result adopted objects")
	startedAt := time.Now()

	for key := range state.ExpectedAdoptedObjects {
		decision, ok := decisions[key]
		if !ok {
			return fmt.Errorf("упо_SearchResult: отсутствует итоговое решение для %s", key)
		}
		if decision.Excluded || decision.Belonging == "" || decision.Belonging == "Native" {
			return fmt.Errorf("упо_SearchResult: объект %s не попал в расширение в режиме adopted", key)
		}

		ctx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, key)
		if ctx == nil {
			return fmt.Errorf("упо_SearchResult: не найден top-level XML объекта %s", key)
		}
		if _, excluded := excludedPaths[ctx.Path]; excluded {
			return fmt.Errorf("упо_SearchResult: объект %s исключен из итогового состава", key)
		}
		if ctx.Properties == nil || textOf(ctx.Properties, "ObjectBelonging") != "Adopted" {
			return fmt.Errorf("упо_SearchResult: объект %s сохранен не в режиме adopted", key)
		}
	}

	logXMLStepCompleted("validate search result adopted objects", startedAt, fmt.Sprintf("objects=%d", len(state.ExpectedAdoptedObjects)))
	return nil
}

func loadSearchResultMarkerGroups(cfg *config.Configuration) (map[string][]string, error) {
	result := make(map[string][]string)
	if cfg == nil {
		return result, nil
	}
	configPath := strings.TrimSpace(cfg.ConfigPath)
	if configPath == "" {
		return nil, fmt.Errorf("Use_упо_SearchResult включен, но путь к файлу конфига не определен")
	}

	markersPath := filepath.Join(filepath.Dir(configPath), "searchingTemplateText.json")
	data, err := os.ReadFile(markersPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать searchingTemplateText.json: %w", err)
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var raw map[string][]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("не удалось разобрать searchingTemplateText.json: %w", err)
	}

	for group, markers := range raw {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		for _, marker := range markers {
			marker = strings.TrimSpace(marker)
			if marker == "" {
				continue
			}
			result[group] = append(result[group], marker)
		}
	}

	return result, nil
}

func collectSearchResultPlaceRequests(dir string, markerGroups map[string][]string) (map[string][]searchResultPlaceRequest, error) {
	result := make(map[string][]searchResultPlaceRequest)
	if dir == "" || len(markerGroups) == 0 {
		return result, nil
	}

	templatePath := filepath.Join(dir, "CommonTemplates", "упо_SearchResult", "Ext", "Template.txt")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать макет упо_SearchResult: %w", err)
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("не удалось разобрать макет упо_SearchResult: %w", err)
	}

	root, ok := raw.(map[string]any)
	if !ok {
		return result, nil
	}

	for kindName, objectsValue := range root {
		kind := normalizeSearchResultKind(kindName)
		if kind == "" {
			continue
		}

		objects, ok := objectsValue.(map[string]any)
		if !ok {
			continue
		}

		for objectName, objectValue := range objects {
			places, ok := objectValue.(map[string]any)
			if !ok {
				continue
			}

			key := kind + "." + strings.TrimSpace(objectName)
			for placeName, countersValue := range places {
				counters, ok := countersValue.(map[string]any)
				if !ok {
					continue
				}

				activeGroups := collectActiveSearchResultGroups(counters, markerGroups)
				if len(activeGroups) == 0 {
					continue
				}

				result[key] = append(result[key], searchResultPlaceRequest{
					Place:  strings.TrimSpace(placeName),
					Groups: activeGroups,
				})
			}
		}
	}

	return result, nil
}

func normalizeSearchResultKind(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if kind, ok := searchResultKindAliases[trimmed]; ok {
		return kind
	}
	return normalizeConfiguredKind(trimmed)
}

func collectActiveSearchResultGroups(counters map[string]any, markerGroups map[string][]string) []string {
	result := make([]string, 0, len(counters))
	for group := range markerGroups {
		count, ok := counters[group]
		if !ok {
			continue
		}
		if searchResultCounterPositive(count) {
			result = append(result, group)
		}
	}
	slices.Sort(result)
	return result
}

func searchResultCounterPositive(value any) bool {
	switch typed := value.(type) {
	case float64:
		return typed > 0
	case int:
		return typed > 0
	case int32:
		return typed > 0
	case int64:
		return typed > 0
	case json.Number:
		num, err := typed.Int64()
		return err == nil && num > 0
	case string:
		num, err := strconv.Atoi(strings.TrimSpace(typed))
		return err == nil && num > 0
	default:
		return false
	}
}

func registerSearchResultPlace(
	state *searchResultState,
	root string,
	topCtx *FileProcessingContext,
	place searchResultPlaceRequest,
	markerGroups map[string][]string,
	prefix string,
	exactTemplates bool,
	diagnosticsPath string,
) error {
	if state == nil || topCtx == nil {
		return nil
	}

	objectDir := strings.TrimSuffix(topCtx.Path, filepath.Ext(topCtx.Path))
	if objectDir == "" {
		return fmt.Errorf("не удалось определить каталог объекта для %s", topCtx.OwnerKey)
	}

	trimmedPlace := strings.TrimSpace(place.Place)
	if trimmedPlace == "" {
		return nil
	}

	switch {
	case trimmedPlace == "ОбщийМодуль":
		if topCtx.OwnerKind != "CommonModule" {
			return fmt.Errorf("место %q допустимо только для CommonModule, получен %s", trimmedPlace, topCtx.OwnerKey)
		}
		return addSearchResultModuleWrite(state, filepath.Join(objectDir, "Ext", "Module.bsl"), topCtx.OwnerKey+".Module", topCtx.OwnerKey, place.Groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	case trimmedPlace == "МодульМенеджера":
		return addSearchResultModuleWrite(state, filepath.Join(objectDir, "Ext", "ManagerModule.bsl"), topCtx.OwnerKey+".ManagerModule", topCtx.OwnerKey, place.Groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	case trimmedPlace == "МодульОбъекта":
		return addSearchResultModuleWrite(state, filepath.Join(objectDir, "Ext", "ObjectModule.bsl"), topCtx.OwnerKey+".ObjectModule", topCtx.OwnerKey, place.Groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	case trimmedPlace == "МодульНабораЗаписей":
		return addSearchResultModuleWrite(state, filepath.Join(objectDir, "Ext", "RecordSetModule.bsl"), topCtx.OwnerKey+".RecordSetModule", topCtx.OwnerKey, place.Groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	case trimmedPlace == "МодульКоманды":
		if topCtx.OwnerKind != "CommonCommand" {
			return fmt.Errorf("место %q без имени команды допустимо только для CommonCommand, получен %s", trimmedPlace, topCtx.OwnerKey)
		}
		return addSearchResultModuleWrite(state, filepath.Join(objectDir, "Ext", "CommandModule.bsl"), topCtx.OwnerKey+".CommandModule", topCtx.OwnerKey, place.Groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	case strings.HasPrefix(trimmedPlace, "МодульФормы"):
		formName := strings.TrimSpace(strings.TrimPrefix(trimmedPlace, "МодульФормы"))
		if formName == "" {
			return fmt.Errorf("не удалось определить имя формы из %q для %s", trimmedPlace, topCtx.OwnerKey)
		}
		return addSearchResultFormOverlay(state, objectDir, topCtx, formName, place.Groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	case strings.HasPrefix(trimmedPlace, "МодульКоманды"):
		commandName := strings.TrimSpace(strings.TrimPrefix(trimmedPlace, "МодульКоманды"))
		if commandName == "" {
			return fmt.Errorf("не удалось определить имя команды из %q для %s", trimmedPlace, topCtx.OwnerKey)
		}
		return addSearchResultCommandOverlay(state, objectDir, topCtx, commandName, place.Groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	}

	_ = root
	return nil
}

func addSearchResultFormOverlay(
	state *searchResultState,
	objectDir string,
	topCtx *FileProcessingContext,
	formName string,
	groups []string,
	markerGroups map[string][]string,
	prefix string,
	exactTemplates bool,
	diagnosticsPath string,
) error {
	var formXMLPath string
	var formModuleXMLPath string
	var modulePath string
	var metadataNames []string

	if topCtx.OwnerKind == "CommonForm" {
		formModuleXMLPath = filepath.Join(objectDir, "Ext", "Form.xml")
		modulePath = filepath.Join(objectDir, "Ext", "Form", "Module.bsl")
		metadataNames = []string{topCtx.OwnerKey + ".Form"}
	} else {
		formXMLPath = filepath.Join(objectDir, "Forms", formName+".xml")
		formModuleXMLPath = filepath.Join(objectDir, "Forms", formName, "Ext", "Form.xml")
		modulePath = filepath.Join(objectDir, "Forms", formName, "Ext", "Form", "Module.bsl")
		metadataNames = []string{
			topCtx.OwnerKey + ".Form." + formName,
			topCtx.OwnerKey + ".Form." + formName + ".Form",
		}
		overlay := state.ObjectOverlays[topCtx.OwnerKey]
		if overlay.PreserveForms == nil {
			overlay.PreserveForms = make(map[string]struct{})
		}
		overlay.PreserveForms[formName] = struct{}{}
		state.ObjectOverlays[topCtx.OwnerKey] = overlay
	}

	if formXMLPath != "" {
		if _, err := os.Stat(formXMLPath); err != nil {
			return fmt.Errorf("не найден XML формы %s для %s: %w", formName, topCtx.OwnerKey, err)
		}
		state.PreservedPaths[formXMLPath] = struct{}{}
	}
	if _, err := os.Stat(formModuleXMLPath); err != nil {
		return fmt.Errorf("не найден Form.xml формы %s для %s: %w", formName, topCtx.OwnerKey, err)
	}
	state.PreservedPaths[formModuleXMLPath] = struct{}{}

	for _, metadataName := range metadataNames {
		state.PreservedConfigDumpInfo[metadataName] = struct{}{}
	}

	return addSearchResultModuleWrite(state, modulePath, "", topCtx.OwnerKey, groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
}

func addSearchResultCommandOverlay(
	state *searchResultState,
	objectDir string,
	topCtx *FileProcessingContext,
	commandName string,
	groups []string,
	markerGroups map[string][]string,
	prefix string,
	exactTemplates bool,
	diagnosticsPath string,
) error {
	if topCtx.OwnerKind == "CommonCommand" {
		return addSearchResultModuleWrite(state, filepath.Join(objectDir, "Ext", "CommandModule.bsl"), topCtx.OwnerKey+".CommandModule", topCtx.OwnerKey, groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	}

	overlay := state.ObjectOverlays[topCtx.OwnerKey]
	if overlay.PreserveCommands == nil {
		overlay.PreserveCommands = make(map[string]struct{})
	}
	overlay.PreserveCommands[commandName] = struct{}{}
	state.ObjectOverlays[topCtx.OwnerKey] = overlay

	state.PreservedConfigDumpInfo[topCtx.OwnerKey+".Command."+commandName] = struct{}{}
	state.PreservedConfigDumpInfo[topCtx.OwnerKey+".Command."+commandName+".CommandModule"] = struct{}{}

	return addSearchResultModuleWrite(state, filepath.Join(objectDir, "Commands", commandName, "Ext", "CommandModule.bsl"), "", topCtx.OwnerKey, groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
}

func addSearchResultModuleWrite(
	state *searchResultState,
	modulePath string,
	metadataName string,
	ownerKey string,
	groups []string,
	markerGroups map[string][]string,
	prefix string,
	exactTemplates bool,
	diagnosticsPath string,
) error {
	if state == nil {
		return nil
	}
	if modulePath == "" {
		return fmt.Errorf("не задан путь к модулю для SearchResult")
	}
	if _, exists := state.ModuleWrites[modulePath]; exists {
		return fmt.Errorf("повторное наложение текста модуля SearchResult для %s", modulePath)
	}
	if _, err := os.Stat(modulePath); err != nil {
		return fmt.Errorf("не найден текст модуля %s: %w", modulePath, err)
	}

	content, err := buildSearchResultModuleContent(modulePath, groups, markerGroups, prefix, exactTemplates, diagnosticsPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(content) == "" {
		return nil
	}

	state.ModuleWrites[modulePath] = searchResultModuleWrite{
		OwnerKey: ownerKey,
		Path:     modulePath,
		Content:  content,
	}
	state.PreservedPaths[modulePath] = struct{}{}
	if metadataName != "" {
		state.PreservedConfigDumpInfo[metadataName] = struct{}{}
	}
	return nil
}

func buildSearchResultModuleContent(modulePath string, groups []string, markerGroups map[string][]string, prefix string, exactTemplates bool, diagnosticsPath string) (string, error) {
	data, err := os.ReadFile(modulePath)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать текст модуля %s: %w", modulePath, err)
	}
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	newline := detectModuleNewline(data)
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	exactMarkers := collectSearchResultMarkers(groups, markerGroups)
	if len(exactMarkers) == 0 {
		return "", nil
	}

	allMethodNames := make(map[string]struct{})
	allMethods := make([]searchResultParsedMethod, 0, 8)
	allModuleBlocks := make([][]string, 0, 2)
	outsideBlock := make([]string, 0, 8)

	for i := 0; i < len(lines); {
		method, next, ok, err := parseSearchResultModuleMethod(lines, i)
		if err != nil {
			return "", fmt.Errorf("не удалось разобрать метод в %s: %w", modulePath, err)
		}
		if !ok {
			outsideBlock = append(outsideBlock, lines[i])
			i++
			continue
		}

		if len(outsideBlock) > 0 {
			allModuleBlocks = append(allModuleBlocks, append([]string(nil), outsideBlock...))
		}
		outsideBlock = outsideBlock[:0]

		allMethodNames[method.Name] = struct{}{}
		method.DirectTransfer = shouldTransferSearchResultMethodDirectly(method.Name)
		if method.DirectTransfer {
			method.GeneratedName = method.Name
		} else {
			method.GeneratedName = prefix + method.Name
		}
		allMethods = append(allMethods, method)
		i = next
	}

	if len(outsideBlock) > 0 {
		allModuleBlocks = append(allModuleBlocks, append([]string(nil), outsideBlock...))
	}

	selectedBlocks := selectSearchResultModuleBlocks(allModuleBlocks, exactMarkers)
	selectedMethods := selectSearchResultMethods(allMethods, exactMarkers)

	exactMismatchMessage := ""
	if len(selectedBlocks) == 0 && len(selectedMethods) == 0 {
		exactMismatchMessage = formatSearchResultMarkerMismatch(modulePath, groups, markerGroups, lines, true)
	}

	if len(selectedBlocks) == 0 && len(selectedMethods) == 0 {
		if exactMismatchMessage != "" {
			writeSearchResultDiagnostics(diagnosticsPath, exactMismatchMessage)
		}
		softMarkers := collectAllSearchResultMarkers(markerGroups)
		selectedBlocks = selectSearchResultModuleBlocks(allModuleBlocks, softMarkers)
		selectedMethods = selectSearchResultMethods(allMethods, softMarkers)
	}

	for _, method := range selectedMethods {
		if method.DirectTransfer {
			continue
		}
		if _, exists := allMethodNames[method.GeneratedName]; exists {
			return "", fmt.Errorf("в модуле %s уже существует метод %s; SearchResult не может безопасно наложить код", modulePath, method.GeneratedName)
		}
	}

	if len(selectedBlocks) == 0 && len(selectedMethods) == 0 {
		mismatchMessage := exactMismatchMessage
		if mismatchMessage == "" {
			mismatchMessage = formatSearchResultMarkerMismatch(modulePath, groups, markerGroups, lines, exactTemplates)
		}
		writeSearchResultDiagnostics(diagnosticsPath, mismatchMessage)
		return "", nil
	}

	rendered := make([]string, 0, len(selectedBlocks)+len(selectedMethods)*6)
	for _, block := range selectedBlocks {
		rendered = append(rendered, block...)
		if len(rendered) > 0 && rendered[len(rendered)-1] != "" {
			rendered = append(rendered, "")
		}
	}

	for idx, method := range selectedMethods {
		rendered = append(rendered, method.DirectiveLines...)
		if method.DirectTransfer {
			rendered = append(rendered, method.HeaderLine)
		} else if method.IsFunction {
			rendered = append(rendered, `&ИзменениеИКонтроль("`+method.Name+`")`)
			rendered = append(rendered, renameSearchResultMethodHeader(method.HeaderLine, method.Name, method.GeneratedName))
		} else {
			rendered = append(rendered, `&После("`+method.Name+`")`)
			rendered = append(rendered, renameSearchResultMethodHeader(method.HeaderLine, method.Name, method.GeneratedName))
		}
		rendered = append(rendered, method.BodyLines...)
		if idx != len(selectedMethods)-1 {
			rendered = append(rendered, "")
		}
	}

	for len(rendered) > 0 && rendered[len(rendered)-1] == "" {
		rendered = rendered[:len(rendered)-1]
	}

	return strings.Join(rendered, newline) + newline, nil
}

func detectModuleNewline(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

func collectSearchResultMarkers(groups []string, markerGroups map[string][]string) []string {
	result := make([]string, 0, len(groups)*4)
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, marker := range markerGroups[group] {
			if marker == "" {
				continue
			}
			if _, ok := seen[marker]; ok {
				continue
			}
			seen[marker] = struct{}{}
			result = append(result, marker)
		}
	}
	return result
}

func collectAllSearchResultMarkers(markerGroups map[string][]string) []string {
	groups := make([]string, 0, len(markerGroups))
	for group := range markerGroups {
		groups = append(groups, group)
	}
	slices.Sort(groups)
	return collectSearchResultMarkers(groups, markerGroups)
}

func selectSearchResultModuleBlocks(blocks [][]string, markers []string) [][]string {
	selected := make([][]string, 0, len(blocks))
	for _, block := range blocks {
		if blockContainsSearchResultMarker(block, markers) {
			selected = append(selected, block)
		}
	}
	return selected
}

func selectSearchResultMethods(methods []searchResultParsedMethod, markers []string) []searchResultParsedMethod {
	selected := make([]searchResultParsedMethod, 0, len(methods))
	for _, method := range methods {
		if blockContainsSearchResultMarker(searchResultMethodLines(method), markers) {
			selected = append(selected, method)
		}
	}
	return selected
}

func searchResultMethodLines(method searchResultParsedMethod) []string {
	lines := make([]string, 0, len(method.DirectiveLines)+1+len(method.BodyLines))
	lines = append(lines, method.DirectiveLines...)
	lines = append(lines, method.HeaderLine)
	lines = append(lines, method.BodyLines...)
	return lines
}

func collectSearchResultFoundGroups(lines []string, markerGroups map[string][]string) []string {
	found := make([]string, 0, len(markerGroups))
	for group, markers := range markerGroups {
		if blockContainsSearchResultMarker(lines, markers) {
			found = append(found, group)
		}
	}
	slices.Sort(found)
	return found
}

func formatSearchResultMarkerMismatch(modulePath string, expectedGroups []string, markerGroups map[string][]string, lines []string, exactTemplates bool) string {
	expected := append([]string(nil), expectedGroups...)
	slices.Sort(expected)
	found := collectSearchResultFoundGroups(lines, markerGroups)

	if exactTemplates {
		if len(found) == 0 {
			return fmt.Sprintf("макет упо_SearchResult ожидает точные метки групп [%s] в %s, но в модуле не найдена ни одна метка из searchingTemplateText.json", strings.Join(expected, ", "), modulePath)
		}
		return fmt.Sprintf("макет упо_SearchResult ожидает точные метки групп [%s] в %s, но найдены только группы [%s]", strings.Join(expected, ", "), modulePath, strings.Join(found, ", "))
	}

	if len(found) == 0 {
		return fmt.Sprintf("макет упо_SearchResult ожидает метки групп [%s] в %s, но в модуле не найдена ни одна метка из searchingTemplateText.json", strings.Join(expected, ", "), modulePath)
	}
	return fmt.Sprintf("макет упо_SearchResult ожидает метки групп [%s] в %s, но не удалось собрать код даже после мягкого сопоставления; найдены группы [%s]", strings.Join(expected, ", "), modulePath, strings.Join(found, ", "))
}

func searchResultDiagnosticsPath(cfg *config.Configuration) string {
	if cfg == nil || strings.TrimSpace(cfg.OutputPath) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(cfg.OutputPath), "_log", "searchresult-template-errors.log")
}

func writeSearchResultDiagnostics(path string, message string) {
	path = strings.TrimSpace(path)
	message = strings.TrimSpace(message)
	if path == "" || message == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Printf("searchresult diagnostics: cannot create log dir for %s: %v", path, err)
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("searchresult diagnostics: cannot open %s: %v", path, err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(message + "\n"); err != nil {
		log.Printf("searchresult diagnostics: cannot write %s: %v", path, err)
	}
}

func blockContainsSearchResultMarker(lines []string, markers []string) bool {
	for _, line := range lines {
		for _, marker := range markers {
			if strings.Contains(line, marker) {
				return true
			}
		}
	}
	return false
}

func parseSearchResultModuleMethod(lines []string, start int) (method searchResultParsedMethod, next int, ok bool, err error) {
	if start >= len(lines) {
		return method, start, false, nil
	}

	idx := start
	directives := make([]string, 0, 2)
	for idx < len(lines) && isModuleDirectiveLine(lines[idx]) {
		directives = append(directives, lines[idx])
		idx++
	}
	if idx >= len(lines) {
		return method, start, false, nil
	}

	name, isFunction, headerOK := parseModuleMethodHeader(lines[idx])
	if !headerOK {
		return method, start, false, nil
	}

	endKeyword := "КонецПроцедуры"
	if isFunction {
		endKeyword = "КонецФункции"
	}

	end := idx + 1
	for end < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[end]), endKeyword) {
		end++
	}
	if end >= len(lines) {
		return method, start, false, fmt.Errorf("не найдено %s для метода %s", endKeyword, name)
	}

	method.Name = name
	method.IsFunction = isFunction
	method.DirectiveLines = directives
	method.HeaderLine = lines[idx]
	method.BodyLines = append([]string(nil), lines[idx+1:end+1]...)
	return method, end + 1, true, nil
}

func isModuleDirectiveLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "&")
}

func parseModuleMethodHeader(line string) (name string, isFunction bool, ok bool) {
	matches := moduleMethodHeaderPattern.FindStringSubmatch(line)
	if len(matches) != 5 {
		return "", false, false
	}
	return matches[4], strings.EqualFold(matches[2], "Функция"), true
}

func renameSearchResultMethodHeader(headerLine, originalName, generatedName string) string {
	indexes := moduleMethodHeaderPattern.FindStringSubmatchIndex(headerLine)
	if len(indexes) < 10 {
		return headerLine
	}
	nameStart := indexes[8]
	nameEnd := indexes[9]
	if nameStart < 0 || nameEnd < 0 || originalName == "" || generatedName == "" {
		return headerLine
	}
	return headerLine[:nameStart] + generatedName + headerLine[nameEnd:]
}

func shouldTransferSearchResultMethodDirectly(name string) bool {
	name = strings.TrimSpace(name)
	return strings.HasPrefix(name, "упо_") || strings.HasPrefix(name, "Подключаемый_упо")
}

func writeSearchResultModuleFiles(state *searchResultState, decisions map[string]objectDecision) error {
	if state == nil {
		return nil
	}
	for _, write := range state.ModuleWrites {
		if write.OwnerKey != "" {
			if decision, ok := decisions[write.OwnerKey]; ok && decision.Belonging == "Native" {
				continue
			}
		}
		if err := os.WriteFile(write.Path, []byte(write.Content), 0o644); err != nil {
			return fmt.Errorf("не удалось записать текст модуля %s: %w", write.Path, err)
		}
	}
	return nil
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
	return cleanupMissingFormCommandReferencesWithResolver(doc, func(name string) bool {
		return roleMetadataTargetExists(name, contexts)
	})
}

func cleanupMissingFormCommandReferencesWithResolver(doc *etree.Document, exists metadataTargetExistsFunc) bool {
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
			if shouldRemoveFormCommandReference(child, defined, exists) {
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
	for _, field := range append([]*etree.Element(nil), root.FindElements(".//Field")...) {
		if !shouldRemoveMissingFormConstantsSetField(field, contexts, indexes, decisions) {
			continue
		}
		if parent := field.Parent(); parent != nil {
			parent.RemoveChild(field)
			changed = true
		}
	}

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

func shouldRemoveMissingFormConstantsSetField(el *etree.Element, contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) bool {
	if el == nil || !strings.EqualFold(localName(el.Tag), "Field") {
		return false
	}
	constantName := missingFormConstantSetReferenceName(strings.TrimSpace(el.Text()))
	if constantName == "" {
		return false
	}
	return !topLevelMetadataIncludedIndexed("Constant."+constantName, contexts, indexes, decisions)
}

func shouldRemoveMissingFormConstantsSetReference(el *etree.Element, contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) bool {
	if el == nil || !strings.EqualFold(localName(el.Tag), "DataPath") {
		return false
	}

	constantName := missingFormConstantSetReferenceName(strings.TrimSpace(el.Text()))
	if constantName == "" {
		return false
	}

	return !topLevelMetadataIncludedIndexed("Constant."+constantName, contexts, indexes, decisions)
}

func missingFormConstantSetReferenceName(text string) string {
	if !strings.HasPrefix(text, "НаборКонстант.") {
		return ""
	}
	return strings.TrimPrefix(text, "НаборКонстант.")
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

func cleanupMissingFormOwnerObjectReferences(doc *etree.Document, ownerCtx *FileProcessingContext) bool {
	if doc == nil || ownerCtx == nil {
		return false
	}

	available := collectAvailableDynamicListFields(ownerCtx)

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
			if shouldRemoveMissingFormOwnerObjectReference(child, ownerCtx, available) {
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

	for _, columns := range append([]*etree.Element(nil), root.FindElements(".//*[local-name()='AdditionalColumns']")...) {
		if !shouldRemoveMissingFormOwnerObjectAdditionalColumns(columns, ownerCtx, available) {
			continue
		}
		parent := columns.Parent()
		if parent == nil {
			continue
		}
		parent.RemoveChild(columns)
		changed = true
	}

	return changed
}

func shouldRemoveMissingFormOwnerObjectReference(el *etree.Element, ownerCtx *FileProcessingContext, available map[string]struct{}) bool {
	if el == nil || !strings.EqualFold(localName(el.Tag), "DataPath") {
		return false
	}

	text := strings.TrimSpace(el.Text())
	if !strings.HasPrefix(text, "Объект.") {
		return false
	}

	field := strings.TrimPrefix(text, "Объект.")
	if field == "" {
		return false
	}
	return isMissingFormOwnerObjectField(field, ownerCtx.OwnerKind, available)
}

func shouldRemoveMissingFormOwnerObjectAdditionalColumns(el *etree.Element, ownerCtx *FileProcessingContext, available map[string]struct{}) bool {
	if el == nil || !strings.EqualFold(localName(el.Tag), "AdditionalColumns") {
		return false
	}

	table := strings.TrimSpace(el.SelectAttrValue("table", ""))
	if !strings.HasPrefix(table, "Объект.") {
		return false
	}

	field := strings.TrimPrefix(table, "Объект.")
	if field == "" {
		return false
	}

	return isMissingFormOwnerObjectField(field, ownerCtx.OwnerKind, available)
}

func isMissingFormOwnerObjectField(field, ownerKind string, available map[string]struct{}) bool {
	if _, ok := available[field]; ok {
		return false
	}
	return !isKnownDynamicListVirtualField(ownerKind, field)
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

func cleanupNonNativeManualQueryOrphanReferences(doc *etree.Document) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	attrsWithoutMainTable := make(map[string]map[string]struct{})
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

		attrsWithoutMainTable[attrName] = collectDynamicListSchemaDeclaredFields(attr)
	}

	if len(attrsWithoutMainTable) == 0 {
		return false
	}

	changed := false
	for _, field := range append([]*etree.Element(nil), root.FindElements(".//Field")...) {
		if !shouldRemoveNonNativeManualQueryOrphanField(field, attrsWithoutMainTable) {
			continue
		}
		if parent := field.Parent(); parent != nil {
			parent.RemoveChild(field)
			changed = true
		}
	}

	for _, rowPicture := range append([]*etree.Element(nil), root.FindElements(".//RowPictureDataPath")...) {
		if !shouldRemoveNonNativeManualQueryOrphanPath(rowPicture, attrsWithoutMainTable) {
			continue
		}
		if parent := rowPicture.Parent(); parent != nil {
			parent.RemoveChild(rowPicture)
			changed = true
		}
	}

	for _, titleDataPath := range append([]*etree.Element(nil), root.FindElements(".//TitleDataPath")...) {
		if !shouldRemoveNonNativeManualQueryOrphanPath(titleDataPath, attrsWithoutMainTable) {
			continue
		}
		if parent := titleDataPath.Parent(); parent != nil {
			parent.RemoveChild(titleDataPath)
			changed = true
		}
	}

	var walk func(parent, grandparent *etree.Element)
	walk = func(parent, grandparent *etree.Element) {
		children := parent.ChildElements()
		for i := len(children) - 1; i >= 0; i-- {
			child := children[i]
			if shouldRemoveNonNativeManualQueryOrphanPath(child, attrsWithoutMainTable) {
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

func collectDynamicListSchemaDeclaredFields(attr *etree.Element) map[string]struct{} {
	result := make(map[string]struct{})
	if attr == nil {
		return result
	}

	attrName := strings.TrimSpace(attr.SelectAttrValue("name", ""))
	for _, field := range attr.FindElements(".//Settings//Field") {
		if len(field.ChildElements()) == 0 {
			name := strings.TrimSpace(field.Text())
			if normalized, ok := extractDynamicListFieldName(name, attrName); ok {
				name = normalized
			}
			if name != "" {
				result[name] = struct{}{}
			}
			continue
		}
		for _, child := range field.ChildElements() {
			tag := localName(child.Tag)
			if !strings.EqualFold(tag, "dataPath") && !strings.EqualFold(tag, "field") {
				continue
			}
			name := strings.TrimSpace(child.Text())
			if normalized, ok := extractDynamicListFieldName(name, attrName); ok {
				name = normalized
			}
			if name != "" {
				result[name] = struct{}{}
			}
		}
	}

	for field := range collectDynamicListCalculatedFields(attr) {
		result[field] = struct{}{}
	}

	return result
}

func shouldRemoveNonNativeManualQueryOrphanField(el *etree.Element, attrsWithoutMainTable map[string]map[string]struct{}) bool {
	if el == nil || !strings.EqualFold(localName(el.Tag), "Field") {
		return false
	}

	attrName, field, ok := extractNonNativeManualQueryAttrFieldReference(strings.TrimSpace(el.Text()), attrsWithoutMainTable)
	if !ok {
		return false
	}
	_, allowed := attrsWithoutMainTable[attrName][field]
	return !allowed
}

func shouldRemoveNonNativeManualQueryOrphanPath(el *etree.Element, attrsWithoutMainTable map[string]map[string]struct{}) bool {
	if el == nil {
		return false
	}

	tag := localName(el.Tag)
	if !strings.EqualFold(tag, "DataPath") && !strings.EqualFold(tag, "TitleDataPath") && !strings.EqualFold(tag, "RowPictureDataPath") {
		return false
	}

	attrName, field, ok := extractNonNativeManualQueryAttrFieldReference(strings.TrimSpace(el.Text()), attrsWithoutMainTable)
	if !ok {
		return false
	}
	_, allowed := attrsWithoutMainTable[attrName][field]
	return !allowed
}

func extractNonNativeManualQueryAttrFieldReference(value string, attrsWithoutMainTable map[string]map[string]struct{}) (string, string, bool) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "~"))
	if value == "" {
		return "", "", false
	}

	for attrName := range attrsWithoutMainTable {
		if field, ok := extractDynamicListFieldName(value, attrName); ok {
			return attrName, field, true
		}

		currentDataPrefix := "Items." + attrName + ".CurrentData."
		if strings.HasPrefix(value, currentDataPrefix) {
			field := strings.TrimSpace(strings.TrimPrefix(value, currentDataPrefix))
			if field != "" {
				return attrName, field, true
			}
		}
	}

	return "", "", false
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
		for field := range collectDynamicListAttributeFields(root, attrName) {
			declaredFields[field] = struct{}{}
		}
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

		for _, rowPicture := range append([]*etree.Element(nil), table.FindElements("./RowPictureDataPath")...) {
			if strings.TrimSpace(rowPicture.Text()) != dataPath+".DefaultPicture" {
				continue
			}
			if parent := rowPicture.Parent(); parent != nil {
				parent.RemoveChild(rowPicture)
				changed = true
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
		targetAlias := alias
		if _, ok := declaredFields[targetAlias]; !ok {
			aliases := dynamicListStandardMetadataNameAliases(alias)
			matched := false
			for _, candidate := range aliases {
				if _, ok := declaredFields[candidate]; ok {
					targetAlias = candidate
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if nestedFields := collectDirectNestedDeclaredFields(targetAlias, declaredFields); len(nestedFields) > 0 {
			lines[i] = buildNestedManualQuerySelect(match[1], expr, targetAlias, nestedFields, match[3])
			changed = true
			continue
		}

		lines[i] = match[1] + expr + " КАК " + targetAlias + match[3]
		changed = true
	}

	if !changed {
		return queryText, false
	}

	return strings.Join(lines, "\n") + rest, true
}

func dynamicListStandardMetadataNameAliases(field string) []string {
	switch field {
	case "Ссылка":
		return []string{"Ref"}
	case "Наименование":
		return []string{"Description"}
	case "Код":
		return []string{"Code"}
	case "Номер":
		return []string{"Number"}
	case "Дата":
		return []string{"Date"}
	case "ПометкаУдаления":
		return []string{"DeletionMark"}
	case "ЭтоГруппа":
		return []string{"IsFolder"}
	case "Владелец":
		return []string{"Owner"}
	case "Родитель":
		return []string{"Parent"}
	case "Проведен":
		return []string{"Posted"}
	case "Регистратор":
		return []string{"Recorder"}
	case "Период":
		return []string{"Period"}
	case "НомерСтроки":
		return []string{"LineNumber"}
	case "Активность":
		return []string{"Active"}
	default:
		return nil
	}
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

func cleanupNonNativeFormLifecycleEvents(doc *etree.Document) bool {
	root := doc.Root()
	if root == nil {
		return false
	}

	events := root.FindElements("./Events/Event")
	if len(events) == 0 {
		return false
	}

	blocked := map[string]struct{}{
		"AfterWrite":          {},
		"AfterWriteAtServer":  {},
		"BeforeWrite":         {},
		"BeforeWriteAtServer": {},
		"OnReadAtServer":      {},
		"OnWriteAtServer":     {},
	}

	changed := false
	for _, event := range append([]*etree.Element(nil), events...) {
		name := strings.TrimSpace(event.SelectAttrValue("name", ""))
		if _, ok := blocked[name]; !ok {
			continue
		}
		if parent := event.Parent(); parent != nil {
			parent.RemoveChild(event)
			changed = true
		}
	}

	return changed
}

func cleanupNonNativeFormStandardCommands(doc *etree.Document) bool {
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
			if shouldRemoveNonNativeFormStandardCommand(child) {
				if removesWholeFormNoiseElement(child) && grandparent != nil {
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

func cleanupFinalNonNativeFormNoise(contexts []*FileProcessingContext, indexes *contextIndexes, decisions map[string]objectDecision) (changedFiles int, writtenFiles int, err error) {
	for _, ctx := range contexts {
		if ctx == nil || ctx.Doc == nil {
			continue
		}
		if !strings.Contains(filepath.ToSlash(ctx.RelPath), "/Forms/") {
			continue
		}

		decision, ok := decisions[ctx.OwnerKey]
		if !ok || decision.Excluded || decision.Belonging == "Native" {
			continue
		}

		ownerCtx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, ctx.OwnerKey)
		changed := false
		changed = normalizeManualQueryWithoutMainTable(ctx.Doc) || changed
		changed = cleanupNonNativeManualQueryOrphanReferences(ctx.Doc) || changed
		changed = cleanupMissingFormOwnerObjectReferences(ctx.Doc, ownerCtx) || changed
		changed = cleanupNonNativeFormLifecycleEvents(ctx.Doc) || changed
		changed = cleanupNonNativeFormStandardCommands(ctx.Doc) || changed
		if !changed {
			continue
		}

		changedFiles++
		if writeErr := ctx.Doc.WriteToFile(ctx.Path); writeErr != nil {
			return changedFiles, writtenFiles, fmt.Errorf("ошибка при записи файла %s: %w", ctx.Path, writeErr)
		}
		writtenFiles++
	}

	return changedFiles, writtenFiles, nil
}

func shouldRemoveNonNativeFormStandardCommand(el *etree.Element) bool {
	if el == nil {
		return false
	}

	tag := strings.ToLower(localName(el.Tag))
	text := strings.TrimSpace(el.Text())
	if text == "" {
		return false
	}

	if tag != "commandname" && tag != "command" && tag != "excludedcommand" {
		return false
	}

	switch text {
	case "WriteAndClose", "Create", "Delete", "Copy", "Change", "MoveItem", "List", "HierarchicalList", "Tree":
		return true
	}

	return strings.HasSuffix(text, ".StandardCommand.Create") ||
		strings.HasSuffix(text, ".StandardCommand.Copy") ||
		strings.HasSuffix(text, ".StandardCommand.Change") ||
		strings.HasSuffix(text, ".StandardCommand.Delete") ||
		strings.HasSuffix(text, ".StandardCommand.MoveItem") ||
		strings.HasSuffix(text, ".StandardCommand.List") ||
		strings.HasSuffix(text, ".StandardCommand.HierarchicalList") ||
		strings.HasSuffix(text, ".StandardCommand.Tree") ||
		strings.HasSuffix(text, ".StandardCommand.WriteAndClose")
}

func shouldRemoveUniversalFormNoise(el *etree.Element) bool {
	tag := strings.ToLower(localName(el.Tag))
	text := strings.TrimSpace(el.Text())

	if text == "LevelDown" || text == "LevelUp" {
		return tag == "commandname" || tag == "command" || tag == "excludedcommand"
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

func shouldRemoveFormCommandReference(el *etree.Element, defined map[string]struct{}, exists metadataTargetExistsFunc) bool {
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
			return exists != nil && !exists(text)
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

func replaceBaseBindingGUIDsInDoc(doc *etree.Document, replacements map[string]string) bool {
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

		hasProperties := el.FindElement("./Properties") != nil
		isMetadataDumpEntry := strings.EqualFold(localName(el.Tag), "Metadata")
		for i, attr := range el.Attr {
			key := localName(attr.Key)
			if strings.EqualFold(key, "ClassId") {
				continue
			}
			if strings.EqualFold(key, "uuid") && hasProperties {
				continue
			}
			if strings.EqualFold(key, "id") && isMetadataDumpEntry {
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

func collectMetadataBindingTargets(
	doc *etree.Document,
	ownerKey string,
	baseBindings map[string]string,
	decisions map[string]objectDecision,
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule,
) map[*etree.Element]metadataBindingTarget {
	result := make(map[*etree.Element]metadataBindingTarget)
	target := metadataTargetElement(doc)
	if target == nil {
		return result
	}

	var walk func(*etree.Element, string)
	walk = func(el *etree.Element, metadataPath string) {
		if el == nil {
			return
		}

		uuid := normalizeGUIDValue(el.SelectAttrValue("uuid", ""))
		if uuid != "" && el.FindElement("./Properties") != nil {
			baseObjectID := uuid
			hasBinding := false
			if shouldTrackIdentityMetadataPath(metadataPath, decisions, adoptedStubMetaDataRules) {
				if bindingID := normalizeGUIDValue(baseBindings[metadataPath]); bindingID != "" {
					baseObjectID = bindingID
					hasBinding = true
					log.Printf("binding applied: metadata=%s base_object_id=%s", metadataPath, bindingID)
				} else {
					log.Printf("missing base binding: metadata=%s", metadataPath)
				}
			}
			result[el] = metadataBindingTarget{
				MetadataPath: metadataPath,
				CurrentID:    uuid,
				BaseObjectID: baseObjectID,
				HasBinding:   hasBinding,
			}
		}

		childObjects := el.FindElement("./ChildObjects")
		if childObjects == nil {
			return
		}

		for _, child := range childObjects.ChildElements() {
			childName := metadataChildName(child)
			if childName == "" {
				continue
			}
			childPath := strings.TrimSpace(localName(child.Tag)) + "." + childName
			if metadataPath != "" {
				childPath = metadataPath + "." + childPath
			}
			walk(child, childPath)
		}
	}

	walk(target, ownerKey)
	return result
}

func collectMetadataBindingTargetsByDoc(
	contexts []*FileProcessingContext,
	baseBindings map[string]string,
	decisions map[string]objectDecision,
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule,
) map[*etree.Document]map[*etree.Element]metadataBindingTarget {
	result := make(map[*etree.Document]map[*etree.Element]metadataBindingTarget)
	for _, ctx := range contexts {
		if ctx == nil || ctx.Doc == nil || ctx.OwnerKey == "" {
			continue
		}
		targets := collectMetadataBindingTargets(ctx.Doc, ctx.OwnerKey, baseBindings, decisions, adoptedStubMetaDataRules)
		if len(targets) == 0 {
			continue
		}
		result[ctx.Doc] = targets
	}
	return result
}

func collectBaseBindingReferenceReplacements(
	bindingTargetsByDoc map[*etree.Document]map[*etree.Element]metadataBindingTarget,
	guidReplacements map[string]string,
) map[string]string {
	result := make(map[string]string)
	for _, targets := range bindingTargetsByDoc {
		for _, target := range targets {
			if !target.HasBinding {
				continue
			}
			currentID := normalizeGUIDValue(target.CurrentID)
			baseObjectID := normalizeGUIDValue(target.BaseObjectID)
			if currentID == "" || baseObjectID == "" {
				continue
			}
			result[currentID] = baseObjectID
			if extensionID := normalizeGUIDValue(guidReplacements[currentID]); extensionID != "" {
				result[extensionID] = baseObjectID
			}
		}
	}
	return result
}

func ensureAdoptedExtendedConfigurationObjects(doc *etree.Document, bindingTargets map[*etree.Element]metadataBindingTarget) bool {
	if doc == nil || doc.Root() == nil || len(bindingTargets) == 0 {
		return false
	}

	changed := false
	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		if el == nil {
			return
		}

		if target, ok := bindingTargets[el]; ok {
			properties := el.FindElement("./Properties")
			if properties != nil {
				preserveNative := strings.EqualFold(strings.TrimSpace(el.SelectAttrValue(preserveNativeObjectBelongingAttr, "")), "true")
				if !preserveNative {
					setObjectBelonging(properties, "Adopted")
					if !modifyElement(properties, "ExtendedConfigurationObject", target.BaseObjectID) {
						addElement(properties, "ExtendedConfigurationObject", target.BaseObjectID)
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

func stripPreserveNativeObjectBelongingMarkers(doc *etree.Document) bool {
	if doc == nil || doc.Root() == nil {
		return false
	}

	changed := false
	var walk func(*etree.Element)
	walk = func(el *etree.Element) {
		if el == nil {
			return
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

	configuration := properties.Parent()
	identifier := ""
	if configuration != nil {
		identifier = normalizeGUIDValue(configuration.SelectAttrValue("uuid", ""))
	}
	if cfg != nil {
		if configured := normalizeGUIDValue(cfg.ExtensionIdentifier()); configured != "" {
			identifier = configured
		}
	}

	name := strings.TrimSpace(cfg.ExtensionName())
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

	vendor := strings.TrimSpace(textOf(properties, "Vendor"))
	version := strings.TrimSpace(textOf(properties, "Version"))

	removeAllChildren(properties)
	if configuration != nil && identifier != "" {
		configuration.RemoveAttr("uuid")
		configuration.CreateAttr("uuid", identifier)
	}

	addSimpleElement(properties, "ObjectBelonging", "Adopted")
	addSimpleElement(properties, "Name", name)
	addSimpleElement(properties, "Synonym", "")
	setRussianSynonym(properties.FindElement("Synonym"), name)
	addSimpleElement(properties, "Comment", "")
	addSimpleElement(properties, "ConfigurationExtensionPurpose", "Customization")
	addSimpleElement(properties, "KeepMappingToExtendedConfigurationObjectsByIDs", "true")
	addSimpleElement(properties, "NamePrefix", cfg.ExtensionPrefix())
	addSimpleElement(properties, "ConfigurationExtensionCompatibilityMode", compatibility)
	addSimpleElement(properties, "DefaultRunMode", "ManagedApplication")
	addUsePurposes(properties)
	addSimpleElement(properties, "ScriptVariant", "Russian")
	addSimpleElement(properties, "Vendor", vendor)
	addSimpleElement(properties, "Version", version)
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

func cleanupConfigDumpInfoForbiddenMetadata(doc *etree.Document, forbidden map[string]struct{}) bool {
	if doc == nil || len(forbidden) == 0 {
		return false
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
				if isForbiddenMetadataPath(forbidden, name) {
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

func cleanupRootConfigurationModuleTexts(doc *etree.Document) bool {
	if doc == nil {
		return false
	}

	root := doc.Root()
	if root == nil {
		return false
	}

	configuration := metadataTargetElement(doc)
	if configuration == nil || !strings.EqualFold(localName(configuration.Tag), "Configuration") {
		return false
	}

	changed := false
	for _, tag := range []string{"ManagedApplicationModule", "SessionModule", "ExternalConnectionModule", "OrdinaryApplicationModule"} {
		for _, el := range configuration.FindElements("./" + tag) {
			configuration.RemoveChild(el)
			changed = true
		}
	}
	return changed
}

func cleanupRoleDanglingMetadataRights(doc *etree.Document, contexts []*FileProcessingContext) bool {
	return cleanupRoleDanglingMetadataRightsWithResolver(doc, func(name string) bool {
		return roleMetadataTargetExists(name, contexts)
	})
}

func cleanupRoleDanglingMetadataRightsWithResolver(doc *etree.Document, exists metadataTargetExistsFunc) bool {
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
		if exists != nil && !exists(name) {
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
	return roleMetadataTargetExistsWithOptions(name, contexts, nil, false)
}

func roleMetadataTargetExistsFinal(name string, contexts []*FileProcessingContext, decisions map[string]objectDecision) bool {
	return roleMetadataTargetExistsWithOptions(name, contexts, decisions, true)
}

func roleMetadataTargetExistsWithOptions(name string, contexts []*FileProcessingContext, decisions map[string]objectDecision, disableFilesystemForNonNative bool) bool {
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

	if disableFilesystemForNonNative {
		if decision, ok := decisions[topKey]; ok && !decision.Excluded && decision.Belonging != "Native" {
			return false
		}
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
	return cleanupDanglingCommandInterfaceCommandsWithResolver(doc, func(name string) bool {
		return roleMetadataTargetExists(name, contexts)
	})
}

func cleanupDanglingCommandInterfaceCommandsWithResolver(doc *etree.Document, exists metadataTargetExistsFunc) bool {
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
				if name != "" && !strings.Contains(name, ".StandardCommand.") && exists != nil && !exists(name) {
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

func cleanupConfigDumpInfoNonNativeChildren(doc *etree.Document, contexts []*FileProcessingContext, decisions map[string]objectDecision, preserved map[string]struct{}) bool {
	return cleanupConfigDumpInfoNonNativeChildrenWithResolver(doc, decisions, func(name string) bool {
		return roleMetadataTargetExists(name, contexts)
	}, preserved)
}

func cleanupConfigDumpInfoNonNativeChildrenWithResolver(doc *etree.Document, decisions map[string]objectDecision, exists metadataTargetExistsFunc, preserved map[string]struct{}) bool {
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
			if _, keep := preserved[name]; keep {
				walk(child)
				continue
			}
			removed := false
			for prefix := range nonNativePrefixes {
				if strings.HasPrefix(name, prefix) {
					if _, ok := configDumpInfoTopLevelKey(name); !ok {
						if isDisallowedAdoptedModuleMetadata(name) || (exists != nil && !exists(name)) {
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
				if decision, decisionExists := decisions[key]; decisionExists && !decision.Excluded && decision.Belonging != "Native" {
					for _, nested := range append([]*etree.Element(nil), child.ChildElements()...) {
						nestedName := strings.TrimSpace(nested.SelectAttrValue("name", ""))
						if _, keep := preserved[nestedName]; keep {
							continue
						}
						if strings.EqualFold(localName(nested.Tag), "Metadata") &&
							(isDisallowedAdoptedModuleMetadata(nestedName) || (exists != nil && !exists(nestedName))) {
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

func collectAdoptedCodeModulePaths(contexts []*FileProcessingContext, decisions map[string]objectDecision, excludedPaths map[string]struct{}) {
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

		for _, name := range []string{"ManagerModule.bsl", "ObjectModule.bsl", "ValueManagerModule.bsl"} {
			modulePath := filepath.Join(objectDir, "Ext", name)
			if _, err := os.Stat(modulePath); err == nil {
				excludedPaths[modulePath] = struct{}{}
			}
		}
	}
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

func collectForbiddenMetadataFilePaths(contexts []*FileProcessingContext, forbiddenByOwner map[string]map[string]struct{}, excludedPaths map[string]struct{}) {
	if len(contexts) == 0 || len(forbiddenByOwner) == 0 || excludedPaths == nil {
		return
	}

	for _, ctx := range contexts {
		if ctx == nil || !ctx.Metadata || !isTopLevelMetadataFile(ctx) {
			continue
		}

		forbidden := forbiddenByOwner[ctx.OwnerKey]
		if len(forbidden) == 0 {
			continue
		}

		objectDir := strings.TrimSuffix(ctx.Path, filepath.Ext(ctx.Path))
		if objectDir == "" {
			continue
		}

		for path := range forbidden {
			parts := strings.Split(strings.TrimSpace(path), ".")
			if len(parts) < 4 {
				continue
			}
			switch parts[2] {
			case "Form":
				formName := strings.TrimSpace(parts[3])
				if formName == "" {
					continue
				}
				excludedPaths[filepath.Join(objectDir, "Forms", formName+".xml")] = struct{}{}
				excludedPaths[filepath.Join(objectDir, "Forms", formName, "Ext", "Form.xml")] = struct{}{}
				excludedPaths[filepath.Join(objectDir, "Forms", formName, "Ext", "Form", "Module.bsl")] = struct{}{}
			case "Command":
				commandName := strings.TrimSpace(parts[3])
				if commandName == "" {
					continue
				}
				excludedPaths[filepath.Join(objectDir, "Commands", commandName, "Ext", "CommandModule.bsl")] = struct{}{}
			case "ManagerModule":
				excludedPaths[filepath.Join(objectDir, "Ext", "ManagerModule.bsl")] = struct{}{}
			case "ObjectModule":
				excludedPaths[filepath.Join(objectDir, "Ext", "ObjectModule.bsl")] = struct{}{}
			case "ValueManagerModule":
				excludedPaths[filepath.Join(objectDir, "Ext", "ValueManagerModule.bsl")] = struct{}{}
			}
		}
	}
}

func collectRootConfigurationModulePaths(root string, excludedPaths map[string]struct{}) {
	if strings.TrimSpace(root) == "" || excludedPaths == nil {
		return
	}
	for _, name := range []string{"ManagedApplicationModule.bsl", "SessionModule.bsl", "ExternalConnectionModule.bsl", "OrdinaryApplicationModule.bsl"} {
		excludedPaths[filepath.Join(root, "Ext", name)] = struct{}{}
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

func cleanupFunctionalOptionsParameterUseNativeChildRefs(
	properties *etree.Element,
	decisions map[string]objectDecision,
) bool {
	if properties == nil || len(decisions) == 0 {
		return false
	}

	use := properties.FindElement("./Use")
	if use == nil {
		return false
	}

	topLevelRefs := make(map[string]struct{})
	for _, child := range use.ChildElements() {
		ref := strings.TrimSpace(child.Text())
		if ref == "" {
			continue
		}
		topKey := metadataDecisionKey(ref)
		if topKey == "" || topKey != ref {
			continue
		}
		topLevelRefs[topKey] = struct{}{}
	}

	changed := false
	for _, child := range append([]etree.Token(nil), use.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}

		if !shouldRemoveFunctionalOptionsParameterUseRef(strings.TrimSpace(el.Text()), decisions, topLevelRefs) {
			continue
		}
		use.RemoveChild(el)
		changed = true
	}

	return changed
}

func shouldRemoveFunctionalOptionsParameterUseRef(
	ref string,
	decisions map[string]objectDecision,
	topLevelRefs map[string]struct{},
) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}

	parts := strings.Split(ref, ".")
	if len(parts) < 4 {
		return false
	}

	topKey := metadataDecisionKey(ref)
	if topKey == "" {
		return false
	}
	if _, ok := topLevelRefs[topKey]; !ok {
		return false
	}

	decision, exists := decisions[topKey]
	if !exists || decision.Excluded || decision.Belonging != "Native" {
		return false
	}

	return true
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

func hasSearchResultPreservedChildren(overlay searchResultObjectOverlay) bool {
	return len(overlay.PreserveForms) > 0 || len(overlay.PreserveCommands) > 0
}

func shouldKeepSearchResultChild(el *etree.Element, overlay searchResultObjectOverlay) bool {
	if el == nil {
		return false
	}

	switch localName(el.Tag) {
	case "Form":
		_, keep := overlay.PreserveForms[strings.TrimSpace(el.Text())]
		return keep
	case "Command":
		name := metadataChildName(el)
		_, keep := overlay.PreserveCommands[name]
		return keep
	default:
		return false
	}
}

func normalizeSearchResultChildObjects(childObjects *etree.Element, overlay searchResultObjectOverlay) bool {
	if childObjects == nil {
		return false
	}

	changed := false
	for _, child := range append([]etree.Token(nil), childObjects.Child...) {
		el, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		if shouldKeepSearchResultChild(el, overlay) {
			continue
		}
		childObjects.RemoveChild(el)
		changed = true
	}
	return changed
}

func cleanupForbiddenChildMetadataPaths(doc *etree.Document, ownerKey string, forbiddenByOwner map[string]map[string]struct{}) bool {
	if doc == nil || ownerKey == "" || len(forbiddenByOwner) == 0 {
		return false
	}

	forbidden := forbiddenByOwner[ownerKey]
	if len(forbidden) == 0 {
		return false
	}

	target := metadataTargetElement(doc)
	if target == nil {
		return false
	}

	var walk func(*etree.Element, string) bool
	walk = func(parent *etree.Element, currentPath string) bool {
		childObjects := parent.FindElement("./ChildObjects")
		if childObjects == nil {
			return false
		}

		changed := false
		for _, token := range append([]etree.Token(nil), childObjects.Child...) {
			child, ok := token.(*etree.Element)
			if !ok {
				continue
			}

			kind := strings.TrimSpace(localName(child.Tag))
			name := strings.TrimSpace(metadataChildName(child))
			if kind == "" || name == "" {
				continue
			}

			childPath := currentPath + "." + kind + "." + name
			if isForbiddenMetadataPath(forbidden, childPath) {
				childObjects.RemoveChild(child)
				changed = true
				continue
			}

			if walk(child, childPath) {
				changed = true
			}
		}

		return changed
	}

	return walk(target, ownerKey)
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

func normalizeAdoptedStubExtFormComposition(doc *etree.Document, contract formDynamicListContract, overlay searchResultObjectOverlay) bool {
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
			if normalizeFormStubChildObjects(el, allowedPaths, "", overlay) {
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

func normalizeFormStubChildObjects(childObjects *etree.Element, allowedPaths map[string]struct{}, parentPath string, overlay searchResultObjectOverlay) bool {
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
		if parentPath == "" && shouldKeepSearchResultChild(el, overlay) {
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
				if normalizeFormStubChildObjects(nestedChildObjects, allowedPaths, currentPath, overlay) {
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

func normalizeAdoptedObjectComposition(doc *etree.Document, ownerKind string, overlay searchResultObjectOverlay) bool {
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
		if tag == "ChildObjects" && hasSearchResultPreservedChildren(overlay) {
			if normalizeSearchResultChildObjects(el, overlay) {
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

func normalizeAdoptedStubExtMetaData(doc *etree.Document, ownerKind string, overlay searchResultObjectOverlay) bool {
	switch ownerKind {
	case "DefinedType", "EventSubscription":
		return false
	case "ExchangePlan":
		return normalizeAdoptedObjectComposition(doc, ownerKind, overlay)
	default:
		return false
	}
}

func normalizeAdoptedStubMetaDataComposition(doc *etree.Document, ownerKind string, rule adoptedStubMetaDataRule, overlay searchResultObjectOverlay) bool {
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
		if normalizeAdoptedStubMetaDataChildObjects(childObjects, rule, overlay) {
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

func normalizeAdoptedStubMetaDataChildObjects(childObjects *etree.Element, rule adoptedStubMetaDataRule, overlay searchResultObjectOverlay) bool {
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
		if shouldKeepSearchResultChild(el, overlay) {
			continue
		}
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

	if name := strings.TrimSpace(textOfFirst(el, "./Properties/Name")); name != "" {
		return name
	}

	return strings.TrimSpace(el.Text())
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

func countRetainedOwnerCommands(commands map[string]map[string]struct{}) int {
	total := 0
	for _, ownerCommands := range commands {
		total += len(ownerCommands)
	}
	return total
}

func newLiveCommandReferenceIndex() *liveCommandReferenceIndex {
	return &liveCommandReferenceIndex{
		byOwner: make(map[string]map[string]struct{}),
		byDoc:   make(map[*FileProcessingContext]map[string]struct{}),
	}
}

func shouldIndexLiveCommandReferences(ctx *FileProcessingContext, decision objectDecision, excludedPaths map[string]struct{}) bool {
	if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
		return false
	}
	if _, excluded := excludedPaths[ctx.Path]; excluded {
		return false
	}
	if decision.Excluded || ctx.FileName == configDumpInfo || isRootServiceFile(ctx) {
		return false
	}
	if strings.EqualFold(ctx.FileName, "Form.xml") && strings.Contains(filepath.ToSlash(ctx.RelPath), "/Forms/") {
		return true
	}
	if ctx.FileName == "CommandInterface.xml" || ctx.FileName == "MainSectionCommandInterface.xml" {
		return true
	}
	return ctx.OwnerKind == "FunctionalOption" && ctx.Metadata && isTopLevelMetadataFile(ctx)
}

func indexLiveCommandReferences(index *liveCommandReferenceIndex, ctx *FileProcessingContext, decision objectDecision, excludedPaths map[string]struct{}) {
	if index == nil || !shouldIndexLiveCommandReferences(ctx, decision, excludedPaths) {
		return
	}

	index.scannedDocs++
	refs := collectLiveOwnerCommandReferences(ctx)
	if len(refs) == 0 {
		return
	}

	flattened := make(map[string]struct{})
	for ownerKey, commands := range refs {
		for commandName := range commands {
			if index.byOwner[ownerKey] == nil {
				index.byOwner[ownerKey] = make(map[string]struct{})
			}
			index.byOwner[ownerKey][commandName] = struct{}{}
			flattened[ownerKey+".Command."+commandName] = struct{}{}
		}
	}
	if len(flattened) > 0 {
		index.byDoc[ctx] = flattened
	}
}

func buildLiveCommandReferenceIndex(contexts []*FileProcessingContext, decisions map[string]objectDecision, excludedPaths map[string]struct{}) *liveCommandReferenceIndex {
	index := newLiveCommandReferenceIndex()
	for _, ctx := range contexts {
		if ctx == nil {
			continue
		}
		indexLiveCommandReferences(index, ctx, decisions[ctx.OwnerKey], excludedPaths)
	}
	return index
}

func diffRetainedOwnerCommands(candidates, retained map[string]map[string]struct{}) (map[string]struct{}, map[string]map[string]struct{}) {
	removedPaths := make(map[string]struct{})
	dirtyOwners := make(map[string]map[string]struct{})

	for ownerKey, candidateCommands := range candidates {
		for commandName := range candidateCommands {
			if _, ok := retained[ownerKey][commandName]; ok {
				continue
			}
			if dirtyOwners[ownerKey] == nil {
				dirtyOwners[ownerKey] = make(map[string]struct{})
			}
			dirtyOwners[ownerKey][commandName] = struct{}{}
			removedPaths[ownerKey+".Command."+commandName] = struct{}{}
		}
	}

	return removedPaths, dirtyOwners
}

func buildFinalMetadataPathIndex(
	contexts []*FileProcessingContext,
	decisions map[string]objectDecision,
	excludedPaths map[string]struct{},
	retainedOwnerCommands map[string]map[string]struct{},
	searchResultState *searchResultState,
) map[string]struct{} {
	index := make(map[string]struct{})

	for _, ctx := range contexts {
		if ctx == nil || ctx.Doc == nil || !ctx.Metadata || !isTopLevelMetadataFile(ctx) {
			continue
		}
		if _, excluded := excludedPaths[ctx.Path]; excluded {
			continue
		}
		decision, ok := decisions[ctx.OwnerKey]
		if !ok || decision.Excluded {
			continue
		}
		overlay := searchResultObjectOverlayForKey(searchResultState, ctx.OwnerKey)
		overlay = mergeSearchResultOverlayCommands(overlay, retainedOwnerCommands[ctx.OwnerKey])
		addFinalMetadataPathsFromContext(index, ctx, decision, overlay)
	}

	return index
}

func addFinalMetadataPathsFromContext(index map[string]struct{}, ctx *FileProcessingContext, decision objectDecision, overlay searchResultObjectOverlay) {
	if index == nil || ctx == nil || ctx.OwnerKey == "" {
		return
	}

	index[ctx.OwnerKey] = struct{}{}
	target := metadataTargetElement(ctx.Doc)
	addFinalMetadataChildPaths(index, target, ctx.OwnerKey, decision, overlay)
	addFinalMetadataFilesystemPaths(index, ctx, decision, overlay)
}

func addFinalMetadataChildPaths(index map[string]struct{}, parent *etree.Element, ownerPath string, decision objectDecision, overlay searchResultObjectOverlay) {
	if index == nil || parent == nil || ownerPath == "" {
		return
	}

	childObjects := parent.FindElement("./ChildObjects")
	if childObjects == nil {
		return
	}

	for _, child := range childObjects.ChildElements() {
		kind := strings.TrimSpace(localName(child.Tag))
		name := strings.TrimSpace(metadataChildName(child))
		if kind == "" || name == "" {
			continue
		}
		if decision.Belonging != "Native" {
			if kind == "Command" {
				if _, keep := overlay.PreserveCommands[name]; !keep {
					continue
				}
			}
			if kind == "Form" {
				if _, keep := overlay.PreserveForms[name]; !keep {
					continue
				}
			}
		}
		path := ownerPath + "." + kind + "." + name
		index[path] = struct{}{}
		addFinalMetadataChildPaths(index, child, path, decision, overlay)
	}
}

func addFinalMetadataFilesystemPaths(index map[string]struct{}, ctx *FileProcessingContext, decision objectDecision, overlay searchResultObjectOverlay) {
	if index == nil || ctx == nil {
		return
	}

	objectDir := strings.TrimSuffix(ctx.Path, filepath.Ext(ctx.Path))
	if objectDir == "" {
		return
	}

	topLevelCommandModulePath := filepath.Join(objectDir, "Ext", "CommandModule.bsl")
	if _, err := os.Stat(topLevelCommandModulePath); err == nil {
		index[ctx.OwnerKey+".CommandModule"] = struct{}{}
	}

	if decision.Belonging == "Native" {
		commandDirs, err := filepath.Glob(filepath.Join(objectDir, "Commands", "*"))
		if err != nil {
			return
		}
		for _, commandDir := range commandDirs {
			info, err := os.Stat(commandDir)
			if err != nil || !info.IsDir() {
				continue
			}
			commandName := filepath.Base(commandDir)
			if strings.TrimSpace(commandName) == "" {
				continue
			}
			index[ctx.OwnerKey+".Command."+commandName] = struct{}{}
			commandModulePath := filepath.Join(commandDir, "Ext", "CommandModule.bsl")
			if _, err := os.Stat(commandModulePath); err == nil {
				index[ctx.OwnerKey+".Command."+commandName+".CommandModule"] = struct{}{}
			}
		}
		return
	}

	for commandName := range overlay.PreserveCommands {
		commandName = strings.TrimSpace(commandName)
		if commandName == "" {
			continue
		}
		index[ctx.OwnerKey+".Command."+commandName] = struct{}{}
		commandModulePath := filepath.Join(objectDir, "Commands", commandName, "Ext", "CommandModule.bsl")
		if _, err := os.Stat(commandModulePath); err == nil {
			index[ctx.OwnerKey+".Command."+commandName+".CommandModule"] = struct{}{}
		}
	}
}

func metadataPathExistsInIndex(index map[string]struct{}, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return true
	}
	_, ok := index[name]
	return ok
}

func finalizeRetainedOwnerCommands(
	contexts []*FileProcessingContext,
	indexes *contextIndexes,
	decisions map[string]objectDecision,
	excludedPaths map[string]struct{},
	adoptedStubMetaDataRules map[string]adoptedStubMetaDataRule,
	formDynamicListContracts map[string]formDynamicListContract,
	retainedOwnerCommandCandidates map[string]map[string]struct{},
	retainedOwnerCommands map[string]map[string]struct{},
	liveCommandRefs *liveCommandReferenceIndex,
	searchResultState *searchResultState,
	forbiddenObjectKeys map[string]struct{},
) (*retainedOwnerCommandFinalizationStats, error) {
	stats := &retainedOwnerCommandFinalizationStats{}
	forbiddenChildMetadataPaths := collectForbiddenChildMetadataPaths(forbiddenObjectKeys)
	removedPaths, dirtyOwners := diffRetainedOwnerCommands(retainedOwnerCommandCandidates, retainedOwnerCommands)
	if len(removedPaths) == 0 {
		for ownerKey := range adoptedStubMetaDataRules {
			ctx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, ownerKey)
			if ctx == nil || ctx.Doc == nil {
				continue
			}
			decision, ok := decisions[ownerKey]
			if !ok || decision.Excluded || decision.Belonging == "Native" {
				continue
			}

			stats.FinalizedOwnerDocs++
			if !stripPreserveNativeObjectBelongingMarkers(ctx.Doc) {
				continue
			}
			stats.ChangedFiles++
			if err := ctx.Doc.WriteToFile(ctx.Path); err != nil {
				return stats, fmt.Errorf("ошибка при записи файла %s: %w", ctx.Path, err)
			}
			stats.WrittenFiles++
		}
		return stats, nil
	}

	finalMetadataPathIndex := buildFinalMetadataPathIndex(contexts, decisions, excludedPaths, retainedOwnerCommands, searchResultState)
	finalExists := func(name string) bool {
		return metadataPathExistsInIndex(finalMetadataPathIndex, name)
	}
	var preservedConfigDumpInfo map[string]struct{}
	if searchResultState != nil {
		preservedConfigDumpInfo = searchResultState.PreservedConfigDumpInfo
	}

	for ownerKey := range dirtyOwners {
		ctx := findTopLevelMetadataContextByOwnerKeyIndexed(indexes, contexts, ownerKey)
		if ctx == nil || ctx.Doc == nil {
			continue
		}
		decision, ok := decisions[ownerKey]
		if !ok || decision.Excluded || decision.Belonging == "Native" {
			continue
		}

		changed := false
		overlay := searchResultObjectOverlayForKey(searchResultState, ownerKey)
		overlay = mergeSearchResultOverlayCommands(overlay, retainedOwnerCommands[ownerKey])
		contract, hasContract := formDynamicListContracts[ownerKey]
		rule, hasRule := adoptedStubMetaDataRules[ownerKey]
		if isAdoptedStubExtMetaData(ctx, decision) {
			changed = normalizeAdoptedStubExtMetaData(ctx.Doc, ctx.OwnerKind, overlay) || changed
		} else if ctx.OwnerKind == "DefinedType" || ctx.OwnerKind == "EventSubscription" {
			// Для специальных adopted metadata-object сохраняем composition/source
			// и не минимизируем их до обычного AdoptedStub.
		} else if hasContract && hasRule {
			changed = normalizeAdoptedStubExtFormComposition(ctx.Doc, mergeAdoptedStubMetaDataIntoFormContract(contract, rule), overlay) || changed
		} else if hasRule {
			changed = normalizeAdoptedStubMetaDataComposition(ctx.Doc, ctx.OwnerKind, rule, overlay) || changed
		} else if hasContract {
			changed = normalizeAdoptedStubExtFormComposition(ctx.Doc, contract, overlay) || changed
		} else {
			changed = normalizeAdoptedObjectComposition(ctx.Doc, ctx.OwnerKind, overlay) || changed
		}
		changed = cleanupForbiddenChildMetadataPaths(ctx.Doc, ownerKey, forbiddenChildMetadataPaths) || changed
		changed = stripPreserveNativeObjectBelongingMarkers(ctx.Doc) || changed
		stats.FinalizedOwnerDocs++
		if !changed {
			continue
		}
		stats.ChangedFiles++
		if err := ctx.Doc.WriteToFile(ctx.Path); err != nil {
			return stats, fmt.Errorf("ошибка при записи файла %s: %w", ctx.Path, err)
		}
		stats.WrittenFiles++
	}

	if configDumpCtx := findContextByRelPath(indexes, contexts, configDumpInfo); configDumpCtx != nil && configDumpCtx.Doc != nil {
		changed := cleanupConfigDumpInfoForbiddenMetadata(configDumpCtx.Doc, forbiddenObjectKeys)
		changed = cleanupConfigDumpInfoNonNativeChildrenWithResolver(configDumpCtx.Doc, decisions, finalExists, preservedConfigDumpInfo) || changed
		if changed {
			stats.ChangedFiles++
			if err := configDumpCtx.Doc.WriteToFile(configDumpCtx.Path); err != nil {
				return stats, fmt.Errorf("ошибка при записи файла %s: %w", configDumpCtx.Path, err)
			}
			stats.WrittenFiles++
		}
	}

	for ctx, refs := range collectAffectedCommandReferenceDocs(liveCommandRefs, removedPaths) {
		if ctx == nil || ctx.Doc == nil || len(refs) == 0 {
			continue
		}

		changed := false
		switch {
		case ctx.FileName == "CommandInterface.xml" || ctx.FileName == "MainSectionCommandInterface.xml":
			changed = cleanupDanglingCommandInterfaceCommandsWithResolver(ctx.Doc, finalExists)
		case strings.EqualFold(ctx.FileName, "Rights.xml"):
			changed = cleanupRoleDanglingMetadataRightsWithResolver(ctx.Doc, finalExists)
		case strings.EqualFold(ctx.FileName, "Form.xml") && strings.Contains(filepath.ToSlash(ctx.RelPath), "/Forms/"):
			changed = cleanupMissingFormCommandReferencesWithResolver(ctx.Doc, finalExists)
		}

		stats.AffectedDocs++
		if !changed {
			continue
		}
		stats.ChangedFiles++
		if err := ctx.Doc.WriteToFile(ctx.Path); err != nil {
			return stats, fmt.Errorf("ошибка при записи файла %s: %w", ctx.Path, err)
		}
		stats.WrittenFiles++
	}

	for _, ctx := range contexts {
		if ctx == nil || ctx.Doc == nil || !strings.EqualFold(ctx.FileName, "Rights.xml") {
			continue
		}
		if _, excluded := excludedPaths[ctx.Path]; excluded {
			continue
		}

		changed := cleanupRoleDanglingMetadataRightsWithResolver(ctx.Doc, finalExists)
		stats.AffectedDocs++
		if !changed {
			continue
		}
		stats.ChangedFiles++
		if err := ctx.Doc.WriteToFile(ctx.Path); err != nil {
			return stats, fmt.Errorf("ошибка при записи файла %s: %w", ctx.Path, err)
		}
		stats.WrittenFiles++
	}

	return stats, nil
}

func collectOwnerCommandCandidates(contexts []*FileProcessingContext, decisions map[string]objectDecision) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})

	for _, ctx := range contexts {
		if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
			continue
		}
		decision, ok := decisions[ctx.OwnerKey]
		if ok && decision.Excluded {
			continue
		}

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

func collectAffectedCommandReferenceDocs(index *liveCommandReferenceIndex, removedPaths map[string]struct{}) map[*FileProcessingContext]map[string]struct{} {
	result := make(map[*FileProcessingContext]map[string]struct{})
	if index == nil || len(removedPaths) == 0 {
		return result
	}

	for ctx, refs := range index.byDoc {
		for ref := range refs {
			if _, removed := removedPaths[ref]; !removed {
				continue
			}
			if result[ctx] == nil {
				result[ctx] = make(map[string]struct{})
			}
			result[ctx][ref] = struct{}{}
		}
	}

	return result
}

func filterRetainedOwnerCommandsByLiveReferences(
	candidates map[string]map[string]struct{},
	liveRefs *liveCommandReferenceIndex,
) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	if len(candidates) == 0 || liveRefs == nil {
		return result
	}

	for ownerKey, commands := range liveRefs.byOwner {
		ownerCandidates := candidates[ownerKey]
		if ownerCandidates == nil {
			continue
		}
		for commandName := range commands {
			if _, ok := ownerCandidates[commandName]; !ok {
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

func collectLiveOwnerCommandReferences(ctx *FileProcessingContext) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	if ctx == nil || ctx.Doc == nil || ctx.Doc.Root() == nil {
		return result
	}

	root := ctx.Doc.Root()
	if ctx.Metadata && isTopLevelMetadataFile(ctx) {
		root = metadataTargetElement(ctx.Doc)
	}
	collectLiveOwnerCommandReferencesFromElement(root, ctx.Metadata && isTopLevelMetadataFile(ctx), result)
	return result
}

func collectLiveOwnerCommandReferencesFromElement(el *etree.Element, skipChildObjects bool, result map[string]map[string]struct{}) {
	if el == nil {
		return
	}

	addLiveOwnerCommandReference(result, strings.TrimSpace(el.Text()))
	for _, attr := range el.Attr {
		addLiveOwnerCommandReference(result, strings.TrimSpace(attr.Value))
	}

	for _, child := range el.ChildElements() {
		if skipChildObjects && strings.EqualFold(localName(child.Tag), "ChildObjects") {
			continue
		}
		collectLiveOwnerCommandReferencesFromElement(child, false, result)
	}
}

func addLiveOwnerCommandReference(result map[string]map[string]struct{}, value string) {
	if !isMetadataCommandReference(value) {
		return
	}

	parts := strings.Split(value, ".")
	if len(parts) < 4 || !strings.EqualFold(parts[2], "Command") {
		return
	}

	ownerKey := strings.TrimSpace(parts[0] + "." + parts[1])
	commandName := strings.TrimSpace(parts[3])
	if ownerKey == "" || commandName == "" {
		return
	}

	if result[ownerKey] == nil {
		result[ownerKey] = make(map[string]struct{})
	}
	result[ownerKey][commandName] = struct{}{}
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
