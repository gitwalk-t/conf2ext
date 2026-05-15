package config

const NamePrefixElement = "NamePrefix"

type ConvertType string

const (
	SrcConvert ConvertType = "srcConvert"
	CfConvert  ConvertType = "cfConvert"
)

type OperationType string

const (
	Add    OperationType = "add"
	Delete OperationType = "delete"
	Modify OperationType = "modify"
)

type ElementOperation struct {
	ElementName string        `json:"element_name"`
	Value       string        `json:"value,omitempty"`
	Operation   OperationType `json:"operation"`
}

type FileOperation struct {
	FileName          string              `json:"file_name"`
	ElementOperations []*ElementOperation `json:"element_operations"`
}

type AdditionalProcessing struct {
	UseMetaDataFile   bool  `json:"Use_MetaDataFile,omitempty"`
	UseSearchResult   bool  `json:"Use_упо_SearchResult,omitempty"`
	UseExactTemplates *bool `json:"UseExactTemplates,omitempty"`
}

type ExtensionProperties struct {
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	Identifier string `json:"identifier"`
}

type Target struct {
	Base    string `json:"base"`
	XMLDump string `json:"xml_dump"`
}

type BaseBinding struct {
	BaseObjectID string `json:"base_object_id"`
}

type BaseBindingsFile struct {
	Bindings map[string]BaseBinding `json:"bindings"`
}

type Configuration struct {
	PlatformVersion             string               `json:"platform_version"`
	Extension                   string               `json:"extension"`
	Prefix                      string               `json:"prefix"`
	ExtensionProperties         ExtensionProperties  `json:"extension_properties"`
	KeepXMLDump                 bool                 `json:"keep_xml_dump,omitempty"`
	StopAfterXMLDump            bool                 `json:"stop_after_xml_dump,omitempty"`
	EnableFormValidation        *bool                `json:"enable_form_validation,omitempty"`
	NativePrefixes              []string             `json:"native_prefixes"`
	ExcludedSubsystems          []string             `json:"excluded_subsystems"`
	ExcludedObjects             []string             `json:"excluded_objects"`
	IncludedNativeObjects       []string             `json:"included_Native_objects"`
	IncludedAdoptedStubObjects  []string             `json:"included_AdoptedStub_objects"`
	ForbiddenAdoptedStubObjects []string             `json:"forbidden_AdoptedStub_objects"`
	InputPath                   string               `json:"input_path" env-required:"true"`
	OutputPath                  string               `json:"output_path" env-required:"true"`
	Target                      Target               `json:"target,omitempty"`
	ConversionType              ConvertType          `json:"conversion_type" env-required:"true"`
	XMLFiles                    []*FileOperation     `json:"xml_file_changes"`
	AdditionalProcessing        AdditionalProcessing `json:"AdditionalProcessing,omitempty"`
	AdditionalAdoptedObjects    []string             `json:"additional_adopted_objects,omitempty"`

	// Deprecated aliases kept for backwards compatibility with old configs.
	IncludedObjects []string `json:"included_objects,omitempty"`

	ConfigPath       string `json:"-"`
	ProjectRootPath  string `json:"-"`
	BaseBindingsPath string `json:"-"`
	IdentityMapPath  string `json:"-"`
}

type DumpInfo struct {
	ConfigName string
	Version    string
}

func (cfg *Configuration) IsFormValidationEnabled() bool {
	return cfg == nil || cfg.EnableFormValidation == nil || *cfg.EnableFormValidation
}

func (cfg *Configuration) IsMetaDataFileEnabled() bool {
	return cfg != nil && cfg.AdditionalProcessing.UseMetaDataFile
}

func (cfg *Configuration) IsSearchResultEnabled() bool {
	return cfg != nil && cfg.AdditionalProcessing.UseSearchResult
}

func (cfg *Configuration) IsExactSearchResultTemplatesEnabled() bool {
	if cfg == nil || cfg.AdditionalProcessing.UseExactTemplates == nil {
		return true
	}
	return *cfg.AdditionalProcessing.UseExactTemplates
}

func (cfg *Configuration) ExtensionName() string {
	if cfg == nil {
		return ""
	}
	if value := cfg.ExtensionProperties.Name; value != "" {
		return value
	}
	return cfg.Extension
}

func (cfg *Configuration) ExtensionPrefix() string {
	if cfg == nil {
		return ""
	}
	if value := cfg.ExtensionProperties.Prefix; value != "" {
		return value
	}
	return cfg.Prefix
}

func (cfg *Configuration) ExtensionIdentifier() string {
	if cfg == nil {
		return ""
	}
	return cfg.ExtensionProperties.Identifier
}
