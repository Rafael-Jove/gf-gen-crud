package main

// FieldInfo represents a parsed struct field.
type FieldInfo struct {
	Name         string
	Type         string
	JsonTag      string
	OrmTag       string
	IsSkip       bool
	IsAudit      bool
	HTMLType     string
	IsTextarea   bool
	EnumValues   []string
	IsJson       bool
	IsFullWidth  bool
	IsPK         bool
	IsRequired   bool
	IsSearchable bool
	IsFK         bool
	FKTable      string
	FKStructName string
	DataType     string
	Rules        map[string]interface{}
}

type NavItem struct {
	Name      string
	TableName string
	Active    bool
	Abbrev    string
}

// TableInfo contains all metadata for one entity.
type TableInfo struct {
	StructName        string // e.g. "Users", "PersonalProfiles"
	TableName         string // e.g. "users", "personal_profiles"
	VarName           string // e.g. "user", "personal_profile"
	ShortName         string // e.g. "User", "PersonalProfile"
	Fields            []FieldInfo
	ListFields        []FieldInfo
	FormFields        []FieldInfo
	ModuleName        string // Module name from go.mod
	NavItems          []NavItem
	HasGtime          bool   // true jika ada field bertipe gtime.Time
	HasUpload         bool   // true jika ada field bertipe file
	HasFK             bool   // true jika ada setidaknya satu field Foreign Key
	IsReadOnly        bool   // true = hanya view, tidak ada create/edit/delete
	FilterType        string // "input", "form", "both", "none"
	PKName            string // e.g. "Id"
	PKJsonTag         string // e.g. "id"
	PKOrmTag          string // e.g. "id"
	PKType            string // e.g. "uint64", "string", "int"
	LabelFieldName    string // e.g. "Name"
	IsPKAutoIncrement bool   // true if auto increment
}

// CmdControllerInfo holds info to build cmd.go
type CmdControllerInfo struct {
	PackageName   string
	TableName     string
	HasHTML       bool
	HasWrite      bool // false = read-only, skip create/edit/delete HTML routes
	HasFilterPage bool // true = has separate /filter page
	ShortName     string
}

type TableGenConfig struct {
	ViewMode   string `json:"viewMode"`
	FilterType string `json:"filterType"`
}

type GenConfig struct {
	Tables map[string]TableGenConfig `json:"table"`
}

// DBColMeta holds raw column_type info from INFORMATION_SCHEMA.
type DBColMeta struct {
	HTMLType        string
	IsTextarea      bool
	EnumValues      []string
	IsJson          bool
	IsFullWidth     bool
	IsPK            bool
	IsAutoIncrement bool
	IsRequired      bool
	IsFK            bool
	FKTable         string
	DataType        string
}

// Fields that should never appear in create/update forms.
var skipFormFields = map[string]bool{
	"CreatedAt":             true,
	"UpdatedAt":             true,
	"CreatedBy":             true,
	"UpdatedBy":             true,
	"OtpCode":               true,
	"OtpExpiresAt":          true,
	"RefreshToken":          true,
	"RefreshTokenExpiresAt": true,
	"IsBlocker":             true,
	"PinUser":               true,
}
