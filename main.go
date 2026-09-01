// Generator CRUD otomatis untuk GoFrame.
// Membaca semua file entity di internal/model/entity dan men-generate:
//   - api/<name>/v1/<name>.go         (Request/Response structs)
//   - internal/logic/<name>/<name>_gen.go (CRUD logic)
//   - resource/template/<name>/list.html, form.html, detail.html, filter.html
//   - internal/controller/<name>/*    (Controller implementations & HTML route handlers)
//   - Rebuilds internal/cmd/cmd.go to register routes automatically.
//
// Usage:
//
//	go run ./hack/gen_crud.go
//	go run ./hack/gen_crud.go --table=users
//	go run ./hack/gen_crud.go --overwrite
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
	"github.com/manifoldco/promptui"
)

// ============================================================
// Data structures
// ============================================================

// FieldInfo represents a parsed struct field.
type FieldInfo struct {
	Name        string   // Go field name, e.g. "NoHp"
	Type        string   // Go type, e.g. "string", "int64", "*gtime.Time"
	JsonTag     string   // json tag value, e.g. "no_hp"
	OrmTag      string   // orm tag value, e.g. "no_hp"
	IsSkip      bool     // true = exclude from create/update forms
	IsAudit     bool     // true = auto-managed (CreatedAt, UpdatedAt, etc.)
	HTMLType    string   // HTML input type: "text", "number", "date", "datetime-local", "time", "checkbox"
	IsTextarea  bool     // true = render as <textarea> (TEXT/LONGTEXT/JSON columns)
	EnumValues  []string // non-empty = render as <select> with these options
	IsJson      bool     // true = JSON column
	IsFullWidth bool     // true = render full width in forms
	IsPK        bool     // true = Primary Key
	DataType    string   // Simplified type: "integer", "string", "boolean", "float", "enum", "datetime", "date", "time", "text"
	Rules       map[string]interface{}
}

type NavItem struct {
	Name      string
	TableName string
	Active    bool
	Abbrev    string
}

// TableInfo contains all metadata for one entity.
type TableInfo struct {
	StructName string // e.g. "Users", "PersonalProfiles"
	TableName  string // e.g. "users", "personal_profiles"
	VarName    string // e.g. "user", "personal_profile"
	ShortName  string // e.g. "User", "PersonalProfile"
	Fields     []FieldInfo
	ListFields []FieldInfo
	FormFields []FieldInfo
	ModuleName string // Module name from go.mod
	NavItems   []NavItem
	HasGtime   bool   // true jika ada field bertipe gtime.Time
	HasUpload  bool   // true jika ada field bertipe file
	IsReadOnly bool   // true = hanya view, tidak ada create/edit/delete
	FilterType string // "input", "form", "both", "none"
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

// Fields that should never appear in create/update forms.
var skipFormFields = map[string]bool{
	"Id":                    true,
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

func loadGenConfig(root string) *GenConfig {
	cfg := &GenConfig{Tables: map[string]TableGenConfig{}}
	data, err := os.ReadFile(filepath.Join(root, "gen.config.json"))

	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	if cfg.Tables == nil {
		cfg.Tables = map[string]TableGenConfig{}
	}
	return cfg
}

func saveGenConfig(root string, cfg *GenConfig) {
	data, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(root, "gen.config.json"), data, 0644)
}

func promptViewAndFilter(tableName string) TableGenConfig {
	tc := TableGenConfig{ViewMode: "write", FilterType: "both"}

	viewPrompt := promptui.Select{
		Label: fmt.Sprintf("[%s] Mode view", tableName),
		Items: []string{
			"write - tampilkan create, edit, delete",
			"view - hanya lihat, tanpa aksi tulis",
		},
	}

	viewIdx, _, _ := viewPrompt.Run()
	if viewIdx == 1 {
		tc.ViewMode = "view"
	}

	filterPrompt := promptui.Select{
		Label: fmt.Sprintf("[%s] Filter di halaman List", tableName),
		Items: []string{
			"input - satu kolom pencarian (inline di list page)",
			"form - halaman filter terpisah",
			"both - inline + halaman filter terpisah",
			"none - tanpa filter sama sekali",
		},
	}

	filterIdx, _, _ := filterPrompt.Run()
	switch filterIdx {
	case 0:
		tc.FilterType = "input"
	case 1:
		tc.FilterType = "form"
	case 2:
		tc.FilterType = "both"
	case 3:
		tc.FilterType = "none"
	}
	return tc
}

// split strings with space, underscore, or hyphen
func abbrev(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return "??"
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '_' || r == '-'
	})
	if len(parts) == 1 {
		var camelParts []string
		start := 0
		for i, r := range s {
			if i > 0 && unicode.IsUpper(r) {
				camelParts = append(camelParts, s[start:i])
				start = i
			}
		}
		camelParts = append(camelParts, s[start:])
		if len(camelParts) >= 2 {
			parts = camelParts
		}
	}
	if len(parts) >= 2 {
		return strings.ToUpper(parts[0][:1] + parts[1][:1])
	}
	if len(s) >= 2 {
		return strings.ToUpper(s[:2])
	}
	return strings.ToUpper(s)
}

func snakeToTitle(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// structNameToSnake converts "PersonalProfiles" → "personal_profiles"
func structNameToSnake(s string) string {
	var result []rune
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// structNameToVar converts "PersonalProfiles" → "personalProfiles"
func structNameToVar(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// shortName removes trailing "s" for singular form: "Users" → "User"
func shortName(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") || strings.HasSuffix(s, "ches") || strings.HasSuffix(s, "shes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

func goTypeToHTMLInput(t string) string {
	switch {
	case strings.Contains(t, "[]byte"):
		return "file"
	case strings.Contains(t, "int"):
		return "number"
	case strings.Contains(t, "float"):
		return "number"
	case strings.Contains(t, "Time"):
		return "datetime-local"
	case strings.Contains(t, "bool"):
		return "checkbox"
	default:
		return "text"
	}
}

func goTypeToDataType(t string) string {
	t = strings.ReplaceAll(t, "*", "")
	switch {
	case strings.Contains(t, "int"):
		return "integer"
	case strings.Contains(t, "float") || t == "float32" || t == "float64":
		return "float"
	case t == "bool":
		return "boolean"
	case strings.Contains(t, "Time"):
		return "datetime"
	default:
		return "string"
	}
}

// DBColMeta holds raw column_type info from INFORMATION_SCHEMA.
type DBColMeta struct {
	HTMLType    string
	IsTextarea  bool
	EnumValues  []string
	IsJson      bool
	IsFullWidth bool
	IsPK        bool
	DataType    string
}

// parseMySQLLink parses a GoFrame MySQL link string:
//
//	mysql:user:pass@tcp(host:port)/dbname
func parseMySQLLink(link string) (dsn string, ok bool) {
	link = strings.TrimSpace(link)
	parts := strings.SplitN(link, ":", 3)
	if len(parts) < 3 || strings.ToLower(parts[0]) != "mysql" {
		return "", false
	}
	user := parts[1]
	rest := parts[2] // pass@tcp(host:port)/dbname
	atIdx := strings.LastIndex(rest, "@")
	if atIdx < 0 {
		return "", false
	}
	pass := rest[:atIdx]
	hostDB := rest[atIdx+1:] // tcp(host:port)/dbname
	slashIdx := strings.Index(hostDB, "/")
	if slashIdx < 0 {
		return "", false
	}
	hostPart := hostDB[:slashIdx]
	dbname := hostDB[slashIdx+1:]
	if q := strings.Index(dbname, "?"); q >= 0 {
		dbname = dbname[:q]
	}
	hostPart = strings.TrimPrefix(hostPart, "tcp(")
	hostPart = strings.TrimSuffix(hostPart, ")")
	dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, hostPart, dbname)
	return dsn, true
}

// fetchDBColumnTypes reads hack/config.yaml to get the DB DSN, then queries
// INFORMATION_SCHEMA.COLUMNS for the given table and returns a map of
// column_name → DBColMeta. Returns empty map on any error (graceful degradation).
func fetchDBColumnTypes(root, tableName string) map[string]DBColMeta {
	cfg, err := os.ReadFile(filepath.Join(root, "hack", "config.yaml"))
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`link:\s*["']?([^"'\n]+)["']?`)
	m := re.FindSubmatch(cfg)
	if len(m) < 2 {
		return nil
	}
	link := strings.TrimSpace(string(m[1]))
	dsn, ok := parseMySQLLink(link)
	if !ok {
		return nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_KEY
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, tableName)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := map[string]DBColMeta{}
	for rows.Next() {
		var colName, colType, colKey string
		if err := rows.Scan(&colName, &colType, &colKey); err != nil {
			continue
		}
		colTypeLow := strings.ToLower(colType)
		colNameLow := strings.ToLower(colName)
		meta := DBColMeta{HTMLType: "text", DataType: "string"}
		if strings.ToUpper(colKey) == "PRI" {
			meta.IsPK = true
		}
		if strings.Contains(colTypeLow, "json") || strings.Contains(colNameLow, "json") || strings.Contains(colNameLow, "metadata") {
			meta.IsJson = true
			meta.IsFullWidth = true
		}
		switch {
		case strings.HasPrefix(colTypeLow, "enum("):
			inner := colType[5 : len(colType)-1]
			for _, v := range strings.Split(inner, ",") {
				v = strings.Trim(strings.TrimSpace(v), "'\"")
				if v != "" {
					meta.EnumValues = append(meta.EnumValues, v)
				}
			}
			meta.HTMLType = "select"
			meta.DataType = "enum"
		case strings.Contains(colTypeLow, "text"),
			strings.Contains(colTypeLow, "json"):
			meta.IsTextarea = true
			meta.HTMLType = "textarea"
			meta.IsFullWidth = true
			meta.DataType = "text"
		case strings.Contains(colTypeLow, "blob"):
			meta.HTMLType = "file"
			meta.IsFullWidth = true
			meta.DataType = "text"
		case strings.Contains(colTypeLow, "datetime"),
			strings.Contains(colTypeLow, "timestamp"):
			meta.HTMLType = "datetime-local"
			meta.DataType = "datetime"
		case strings.HasPrefix(colTypeLow, "date"):
			meta.HTMLType = "date"
			meta.DataType = "date"
		case strings.HasPrefix(colTypeLow, "time"):
			meta.HTMLType = "time"
			meta.DataType = "time"
		case strings.HasPrefix(colTypeLow, "year"):
			meta.HTMLType = "number"
			meta.DataType = "integer"
		case strings.Contains(colTypeLow, "bool"),
			strings.Contains(colTypeLow, "tinyint(1)"):
			meta.HTMLType = "checkbox"
			meta.DataType = "boolean"
		case strings.Contains(colTypeLow, "int"):
			meta.HTMLType = "number"
			meta.DataType = "integer"
		case strings.Contains(colTypeLow, "float"),
			strings.Contains(colTypeLow, "double"),
			strings.Contains(colTypeLow, "decimal"),
			strings.Contains(colTypeLow, "numeric"):
			meta.HTMLType = "number"
			meta.DataType = "float"
		}
		reVarchar := regexp.MustCompile(`varchar\((\d+)\)`)
		if m := reVarchar.FindStringSubmatch(colTypeLow); len(m) > 1 {
			if length, err := strconv.Atoi(m[1]); err == nil && length >= 150 {
				meta.IsFullWidth = true
			}
		}
		result[colName] = meta
	}
	return result
}

// extractTag extracts a tag value from a raw tag string like `json:"no_hp" orm:"no_hp"`.
func extractTag(raw, key string) string {
	re := regexp.MustCompile(key + `:"([^"]*)"`)
	m := re.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return strings.Split(m[1], ",")[0]
}

// readModuleName reads the module name from go.mod.
func readModuleName(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "backends"
	}
	re := regexp.MustCompile(`module\s+(\S+)`)
	m := re.FindSubmatch(data)
	if len(m) < 2 {
		return "backends"
	}
	return string(m[1])
}

// ============================================================
// Parser
// ============================================================

// parseEntityFile parses one entity .go file and returns a TableInfo.
func parseEntityFile(filePath, moduleName string) (*TableInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var info *TableInfo

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			sName := typeSpec.Name.Name
			snakeName := structNameToSnake(sName)
			short := shortName(sName)

			info = &TableInfo{
				StructName: sName,
				TableName:  snakeName,
				VarName:    structNameToVar(sName),
				ShortName:  short,
				ModuleName: moduleName,
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				fieldName := field.Names[0].Name
				typStr := goTypeStr(field.Type)

				var jsonTag, ormTag string
				if field.Tag != nil {
					raw := field.Tag.Value
					jsonTag = extractTag(raw, "json")
					ormTag = extractTag(raw, "orm")
				}
				if jsonTag == "-" || jsonTag == "" {
					jsonTag = strings.ToLower(fieldName[:1]) + fieldName[1:]
				}

				isSkip := skipFormFields[fieldName]
				isAudit := fieldName == "CreatedAt" || fieldName == "UpdatedAt" ||
					fieldName == "CreatedBy" || fieldName == "UpdatedBy"

				fi := FieldInfo{
					Name:     fieldName,
					Type:     typStr,
					JsonTag:  jsonTag,
					OrmTag:   ormTag,
					IsSkip:   isSkip,
					IsAudit:  isAudit,
					HTMLType: goTypeToHTMLInput(typStr),
					DataType: goTypeToDataType(typStr),
				}
				info.Fields = append(info.Fields, fi)
				if !isAudit {
					info.ListFields = append(info.ListFields, fi)
				}
				if !isSkip {
					info.FormFields = append(info.FormFields, fi)
				}
				if strings.Contains(typStr, "gtime.Time") {
					info.HasGtime = true
				}
			}
			break
		}
	}

	return info, nil
}

// goTypeStr converts an ast.Expr to a readable Go type string.
func goTypeStr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + goTypeStr(t.X)
	case *ast.SelectorExpr:
		return goTypeStr(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + goTypeStr(t.Elt)
	case *ast.MapType:
		return "map[" + goTypeStr(t.Key) + "]" + goTypeStr(t.Value)
	default:
		return "any"
	}
}

// ============================================================
// Write helpers
// ============================================================

func writeFile(path, content string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  [SKIP] %s (already exists, use --overwrite to replace)\n", path)
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	fmt.Printf("  [OK]   %s\n", path)
	return nil
}

func renderTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ============================================================
// Main
// ============================================================

func main() {
	tableFlag := flag.String("table", "", "Generate hanya untuk satu tabel (e.g. --table=users)")
	skipView := flag.Bool("skip-view", false, "Jangan generate HTML template")
	skipLogic := flag.Bool("skip-logic", false, "Jangan generate logic file")
	overwrite := flag.Bool("overwrite", false, "Timpa file yang sudah ada")
	viewModeFlag := flag.String("view-mode", "", "Mode view: write atau view")
	filterTypeFlag := flag.String("filter-type", "", "Jenis filter: input, form, both, atau none")
	interactive := flag.Bool("interactive", false, "Jalankan prompt interaktif")
	initFlag := flag.Bool("init", false, "Inisialisasi driver database dan helper gen.bat")
	reconfigure := flag.Bool("reconfigure", false, "Ubah view-mode & filter-type yang sudah pernah di konfigurasi")
	flag.Parse()

	root, _ := filepath.Abs(".")
	entityDir := filepath.Join(root, "internal", "model", "entity")
	moduleName := readModuleName(root)

	// Check if a database driver is already imported in main.go
	hasDriver := false
	mainPath := filepath.Join(root, "main.go")
	if mainData, err := os.ReadFile(mainPath); err == nil {
		mainStr := string(mainData)
		re := regexp.MustCompile(`(?m)^\s*_\s*"github\.com/gogf/gf/contrib/drivers/`)
		if re.MatchString(mainStr) {
			hasDriver = true
		}
	}

	// Mandatory interactive terminal choice for database driver if initializing
	var driverType string
	if !hasDriver && (*initFlag || *interactive) {
		prompt := promptui.Select{
			Label: "Pilih driver database yang ingin Anda gunakan di project ini",
			Items: []string{
				"mysql  - MySQL / MariaDB",
				"pgsql  - PostgreSQL",
				"sqlite - SQLite",
				"mssql  - SQL Server",
				"oracle - Oracle",
				"Lewatkan (Skip)",
			},
		}

		index, _, err := prompt.Run()
		if err != nil {
			fmt.Printf("Pilihan driver batal: %v\n", err)
			os.Exit(1)
		}

		switch index {
		case 0:
			driverType = "mysql"
		case 1:
			driverType = "pgsql"
		case 2:
			driverType = "sqlite"
		case 3:
			driverType = "mssql"
		case 4:
			driverType = "oracle"
		default:
			fmt.Println("Melewati konfigurasi driver database.")
		}
		fmt.Println()
	}

	// Auto-import driver to main.go and run go get
	if driverType != "" {
		mainPath := filepath.Join(root, "main.go")
		if mainData, err := os.ReadFile(mainPath); err == nil {
			mainStr := string(mainData)
			driverImport := `_ "github.com/gogf/gf/contrib/drivers/` + driverType + `/v2"`
			if !strings.Contains(mainStr, driverImport) {
				reImport := regexp.MustCompile(`import\s*\(`)
				if reImport.MatchString(mainStr) {
					mainStr = reImport.ReplaceAllString(mainStr, "import (\n\t"+driverImport)
					_ = os.WriteFile(mainPath, []byte(mainStr), 0644)
					fmt.Printf("=== Automatically added %s driver import to main.go ===\n", driverType)

					fmt.Printf("=== Fetching database driver package: %s ===\n", driverType)
					cmd := exec.Command("go", "get", "github.com/gogf/gf/contrib/drivers/"+driverType+"/v2")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					_ = cmd.Run()
				}
			}
		}
	}

	// Generate helper gen.bat automatically in project root if it does not exist
	genBatPath := filepath.Join(root, "gen.bat")
	if _, err := os.Stat(genBatPath); os.IsNotExist(err) {
		fmt.Println("=== Generating helper gen.bat ===")
		genBatContent := `@echo off
echo === Running gf gen dao ===
gf gen dao %*

echo === Running gen_crud.go ===
gf-gen-crud --overwrite --interactive %*

echo === Running gf gen service ===
gf gen service

echo === Done! ===
`
		_ = writeFile(genBatPath, genBatContent, false)
	}

	if *initFlag {
		fmt.Println("=== Proyek berhasil di-inisialisasi! ===")
		fmt.Println("Silakan gunakan perintah berikut untuk mulai men-generate kode CRUD:")
		fmt.Println("  .\\gen             → Men-generate semua tabel secara interaktif")
		fmt.Println("  .\\gen --table=xyz → Men-generate tabel tertentu secara interaktif")
		fmt.Println("========================================")
		os.Exit(0)
	}

	fmt.Printf("Module  : %s\n", moduleName)
	fmt.Printf("Entities: %s\n\n", entityDir)

	files, err := filepath.Glob(filepath.Join(entityDir, "*.go"))
	if err != nil || len(files) == 0 {
		fmt.Println("Tidak ada file entity ditemukan di", entityDir)
		os.Exit(1)
	}

	// Load rules.json if it exists
	var rulesMap map[string]map[string]interface{}
	rulesPath := filepath.Join(root, "rules.json")
	if rData, err := os.ReadFile(rulesPath); err == nil {
		if err := json.Unmarshal(rData, &rulesMap); err != nil {
			fmt.Printf("Warning: Gagal membaca rules.json: %v\n", err)
		}
	}

	var allEntities []*TableInfo
	for _, f := range files {
		info, err := parseEntityFile(f, moduleName)
		if err != nil || info == nil {
			fmt.Printf("Skip %s: %v\n", f, err)
			continue
		}
		dbCols := fetchDBColumnTypes(root, info.TableName)
		if len(dbCols) > 0 {
			enrichFields := func(fields []FieldInfo) []FieldInfo {
				for i, fi := range fields {
					lowName := strings.ToLower(fi.Name)
					lowOrm := strings.ToLower(fi.OrmTag)
					if meta, ok := dbCols[fi.OrmTag]; ok {
						fields[i].HTMLType = meta.HTMLType
						fields[i].IsTextarea = meta.IsTextarea
						fields[i].EnumValues = meta.EnumValues
						fields[i].IsJson = meta.IsJson
						fields[i].IsFullWidth = meta.IsFullWidth
						fields[i].DataType = meta.DataType
						fields[i].IsPK = meta.IsPK
						if meta.HTMLType == "file" {
							info.HasUpload = true
						}
						if meta.HTMLType == "date" || meta.HTMLType == "datetime-local" || meta.HTMLType == "time" {
							info.HasGtime = true
						}
					} else {
						if fi.IsTextarea || fi.HTMLType == "file" || fi.HTMLType == "textarea" ||
							strings.Contains(lowName, "address") || strings.Contains(lowName, "alamat") ||
							strings.Contains(lowName, "desc") || strings.Contains(lowName, "deskripsi") ||
							strings.Contains(lowName, "image") || strings.Contains(lowName, "file") ||
							strings.Contains(lowName, "path") || strings.Contains(lowName, "url") ||
							strings.Contains(lowName, "avatar") || strings.Contains(lowName, "cover") {
							fields[i].IsFullWidth = true
						}
					}
					// Fallback check if ID column
					if !fields[i].IsPK && (lowName == "id" || lowOrm == "id" || strings.HasSuffix(lowOrm, "_id") && len(lowOrm) == 3) {
						fields[i].IsPK = true
					}
				}
				return fields
			}
			info.Fields = enrichFields(info.Fields)
			info.ListFields = enrichFields(info.ListFields)
			info.FormFields = enrichFields(info.FormFields)
		}

		enrichRules := func(fields []FieldInfo) []FieldInfo {
			for i, fi := range fields {
				if rulesMap != nil {
					rule, exists := rulesMap[fi.JsonTag]
					if !exists {
						rule, exists = rulesMap[strings.ToLower(fi.Name)]
					}
					if exists {
						fields[i].Rules = make(map[string]interface{})
						for k, v := range rule {
							if k == "type" {
								if typeStr, ok := v.(string); ok {
									fields[i].DataType = typeStr
									if typeStr == "range" {
										fields[i].HTMLType = "range"
									}
								}
							} else {
								fields[i].Rules[k] = v
							}
						}
					}
				}
			}
			return fields
		}
		info.Fields = enrichRules(info.Fields)
		info.ListFields = enrichRules(info.ListFields)
		info.FormFields = enrichRules(info.FormFields)

		allEntities = append(allEntities, info)
	}

	genConfig := loadGenConfig(root)

	// Kumpulkan tabel yang akan diproses (sudah difilter --table kalau ada)
	var targetTables []*TableInfo
	for _, info := range allEntities {
		if *tableFlag != "" {
			tableSet := map[string]bool{}
			for _, t := range strings.Split(*tableFlag, ",") {
				tableSet[strings.TrimSpace(t)] = true
			}
			if !tableSet[info.TableName] {
				continue
			}
		}
		targetTables = append(targetTables, info)
	}

	// Tentukan konfigurasi SEKALI untuk semua tabel yang perlu (belum known / --reconfigure)
	if !*skipView {
		var needsPrompt []string
		for _, info := range targetTables {
			_, known := genConfig.Tables[info.TableName]
			if !known || *reconfigure {
				needsPrompt = append(needsPrompt, info.TableName)
			}
		}

		if len(needsPrompt) > 0 {
			var tc TableGenConfig
			switch {
			case *viewModeFlag != "" || *filterTypeFlag != "":
				tc = TableGenConfig{ViewMode: "write", FilterType: "both"}
				if *viewModeFlag != "" {
					tc.ViewMode = *viewModeFlag
				}
				if *filterTypeFlag != "" {
					tc.FilterType = *filterTypeFlag
				}
			case *interactive:
				fmt.Printf("Tabel berikut belum dikonfigurasi: %s\n", strings.Join(needsPrompt, ", "))
				tc = promptViewAndFilter("SEMUA tabel di atas")
			default:
				tc = TableGenConfig{ViewMode: "write", FilterType: "both"}
			}

			// Terapkan hasil yang SAMA ke semua tabel yang butuh
			for _, tn := range needsPrompt {
				genConfig.Tables[tn] = tc
			}
		}
	}

	for _, info := range targetTables {
		var navItems []NavItem
		for _, ent := range allEntities {
			navItems = append(navItems, NavItem{
				Name:      ent.ShortName,
				TableName: ent.TableName,
				Active:    ent.TableName == info.TableName,
				Abbrev:    abbrev(ent.ShortName),
			})
		}
		info.NavItems = navItems

		if !*skipView {
			tc := genConfig.Tables[info.TableName] // sudah pasti ada, sudah diisi di atas
			info.IsReadOnly = tc.ViewMode == "view"
			info.FilterType = tc.FilterType
		}

		fmt.Printf("=== Generating: %s (%s) ===\n", info.StructName, info.TableName)

		apiContent, err := renderTemplate(apiTemplate, info)
		if err != nil {
			fmt.Printf("  [ERROR] api template: %v\n", err)
		} else {
			apiPath := filepath.Join(root, "api", info.TableName, "v1", info.TableName+".go")
			_ = writeFile(apiPath, apiContent, *overwrite)
		}

		apiInterfaceContent, err := renderTemplate(apiInterfaceTemplate, info)
		if err != nil {
			fmt.Printf("  [ERROR] api interface template: %v\n", err)
		} else {
			apiInterfacePath := filepath.Join(root, "api", info.TableName, info.TableName+".go")
			_ = writeFile(apiInterfacePath, apiInterfaceContent, *overwrite)
		}

		if !*skipLogic {
			logicContent, err := renderTemplate(logicTemplate, info)
			if err != nil {
				fmt.Printf("  [ERROR] logic template: %v\n", err)
			} else {
				logicDir := filepath.Join(root, "internal", "logic", info.VarName)
				logicPath := filepath.Join(logicDir, info.VarName+".go")
				_ = os.Remove(filepath.Join(logicDir, info.VarName+"_gen.go"))
				_ = writeFile(logicPath, logicContent, *overwrite)
			}
		}

		if !*skipView {
			tplDir := filepath.Join(root, "resource", "template", info.TableName)

			listContent, err := renderTemplate(listHTMLTemplate, info)
			if err != nil {
				fmt.Printf("  [ERROR] list view: %v\n", err)
			} else {
				_ = writeFile(filepath.Join(tplDir, "list.html"), listContent, *overwrite)
			}

			if !info.IsReadOnly {
				formContent, err := renderTemplate(formHTMLTemplate, info)
				if err != nil {
					fmt.Printf("  [ERROR] form view: %v\n", err)
				} else {
					_ = writeFile(filepath.Join(tplDir, "form.html"), formContent, *overwrite)
				}
			}

			if info.FilterType == "form" || info.FilterType == "both" {
				filterContent, err := renderTemplate(filterHTMLTemplate, info)
				if err != nil {
					fmt.Printf("  [ERROR] filter view: %v\n", err)
				} else {
					_ = writeFile(filepath.Join(tplDir, "filter.html"), filterContent, *overwrite)
				}
			}

			detailContent, err := renderTemplate(detailHTMLTemplate, info)
			if err != nil {
				fmt.Printf("  [ERROR] detail view: %v\n", err)
			} else {
				_ = writeFile(filepath.Join(tplDir, "detail.html"), detailContent, *overwrite)
			}
		}

		ctrlDir := filepath.Join(root, "internal", "controller", info.TableName)
		_ = os.MkdirAll(ctrlDir, 0755)

		ctrlNewContent, _ := renderTemplate(ctrlNewTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, info.TableName+"_new.go"), ctrlNewContent, *overwrite)

		ctrlFormContent, _ := renderTemplate(ctrlFormTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, info.TableName+"_form.go"), ctrlFormContent, *overwrite)

		shortSnake := structNameToSnake(info.ShortName)
		fileSuffix := "_" + shortSnake
		if info.TableName == shortSnake {
			fileSuffix = ""
		}

		listContentCtrl, _ := renderTemplate(ctrlListTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_list%s.go", info.TableName, fileSuffix)), listContentCtrl, *overwrite)

		getContentCtrl, _ := renderTemplate(ctrlGetTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_get%s.go", info.TableName, fileSuffix)), getContentCtrl, *overwrite)

		createContentCtrl, _ := renderTemplate(ctrlCreateTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_create%s.go", info.TableName, fileSuffix)), createContentCtrl, *overwrite)

		updateContentCtrl, _ := renderTemplate(ctrlUpdateTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_update%s.go", info.TableName, fileSuffix)), updateContentCtrl, *overwrite)

		deleteContentCtrl, _ := renderTemplate(ctrlDeleteTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_delete%s.go", info.TableName, fileSuffix)), deleteContentCtrl, *overwrite)

		fmt.Println()
	}

	fmt.Println("=== Rebuilding internal/cmd/cmd.go ===")
	cmdCtrlDir := filepath.Join(root, "internal", "controller")
	dirs, err := os.ReadDir(cmdCtrlDir)
	if err != nil {
		fmt.Printf("Gagal membaca folder controller: %v\n", err)
		os.Exit(1)
	}

	var controllers []CmdControllerInfo
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		pkgName := d.Name()

		short := snakeToTitle(pkgName)
		for _, ent := range allEntities {
			if ent.TableName == pkgName {
				short = ent.ShortName
				break
			}
		}

		hasHTML := false
		hasWrite := false
		hasFilterPage := false
		tplPath := filepath.Join(root, "resource", "template", pkgName)
		if _, err := os.Stat(tplPath); err == nil {
			hasHTML = true
			if _, err2 := os.Stat(filepath.Join(tplPath, "form.html")); err2 == nil {
				hasWrite = true
			}
			if _, err3 := os.Stat(filepath.Join(tplPath, "filter.html")); err3 == nil {
				hasFilterPage = true
			}
		}
		controllers = append(controllers, CmdControllerInfo{
			PackageName:   pkgName,
			TableName:     pkgName,
			HasHTML:       hasHTML,
			HasWrite:      hasWrite,
			HasFilterPage: hasFilterPage,
			ShortName:     short,
		})
	}

	if !*skipView {
		fmt.Println("=== Generating index.html & query.html ===")
		var indexNavItems []NavItem
		for _, ent := range allEntities {
			indexNavItems = append(indexNavItems, NavItem{
				Name:      ent.ShortName,
				TableName: ent.TableName,
				Active:    false,
				Abbrev:    abbrev(ent.ShortName),
			})
		}

		tplDir := filepath.Join(root, "resource", "template")

		// Generate shared public layout inclusions (sidebar.html and mobile_nav.html)
		publicTplDir := filepath.Join(tplDir, "public")
		_ = os.MkdirAll(publicTplDir, 0755)

		sidebarContent, err := renderTemplate(sidebarHTMLTemplate, allEntities)
		if err == nil {
			_ = writeFile(filepath.Join(publicTplDir, "sidebar.html"), sidebarContent, true)
		} else {
			fmt.Printf("Gagal merender sidebar.html: %v\n", err)
		}

		mobileNavContent, err := renderTemplate(mobileNavHTMLTemplate, allEntities)
		if err == nil {
			_ = writeFile(filepath.Join(publicTplDir, "mobile_nav.html"), mobileNavContent, true)
		} else {
			fmt.Printf("Gagal merender mobile_nav.html: %v\n", err)
		}

		indexContent, err := renderTemplate(indexHTMLTemplate, map[string]interface{}{
			"NavItems": indexNavItems,
		})
		if err == nil {
			_ = writeFile(filepath.Join(tplDir, "index.html"), indexContent, true)
		} else {
			fmt.Printf("Gagal merender index.html: %v\n", err)
		}

		queryContent, err := renderTemplate(queryHTMLTemplate, map[string]interface{}{
			"NavItems": indexNavItems,
		})
		if err == nil {
			_ = writeFile(filepath.Join(tplDir, "query.html"), queryContent, true)
		} else {
			fmt.Printf("Gagal merender query.html: %v\n", err)
		}
	}

	cmdContent, err := renderTemplate(cmdTemplate, struct {
		ModuleName  string
		Controllers []CmdControllerInfo
	}{
		ModuleName:  moduleName,
		Controllers: controllers,
	})
	if err != nil {
		fmt.Printf("Gagal merender template cmd: %v\n", err)
	} else {
		cmdPath := filepath.Join(root, "internal", "cmd", "cmd.go")
		_ = writeFile(cmdPath, cmdContent, true)
	}

	fmt.Println("===================================")
	fmt.Println("Selesai! Langkah berikutnya:")
	fmt.Println("  1. gf gen service   → update service interfaces")
	fmt.Println("  2. go build ./...   → pastikan kompilasi sukses")
	fmt.Println("===================================")

	saveGenConfig(root, genConfig)
}
