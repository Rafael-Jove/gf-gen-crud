// Generator CRUD otomatis untuk GoFrame.
// Membaca semua file entity di internal/model/entity dan men-generate:
//   - api/<name>/v1/<name>.go         (Request/Response structs)
//   - internal/logic/<name>/<name>.go (CRUD logic)
//   - resource/template/<name>/list.html, form.html, detail.html, filter.html
//   - internal/controller/<name>/*    (Controller implementations & HTML route handlers)
//   - Rebuilds internal/cmd/cmd.go to register routes automatically.
//
// Usage:
//	go run ./hack/gen_crud.go
//	go run ./hack/gen_crud.go --table=users
//	go run ./hack/gen_crud.go --overwrite
package main

import (
	"bytes"
	"database/sql"
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

// DBColMeta holds raw column_type info from INFORMATION_SCHEMA.
type DBColMeta struct {
	HTMLType    string
	IsTextarea  bool
	EnumValues  []string
	IsJson      bool
	IsFullWidth bool
}

// parseMySQLLink parses a GoFrame MySQL link string:
//   mysql:user:pass@tcp(host:port)/dbname
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
		SELECT COLUMN_NAME, COLUMN_TYPE
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`, tableName)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := map[string]DBColMeta{}
	for rows.Next() {
		var colName, colType string
		if err := rows.Scan(&colName, &colType); err != nil {
			continue
		}
		colTypeLow := strings.ToLower(colType)
		colNameLow := strings.ToLower(colName)
		meta := DBColMeta{HTMLType: "text"}
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
		case strings.Contains(colTypeLow, "text"),
			strings.Contains(colTypeLow, "json"):
			meta.IsTextarea = true
			meta.HTMLType = "textarea"
			meta.IsFullWidth = true
		case strings.Contains(colTypeLow, "blob"):
			meta.HTMLType = "file"
			meta.IsFullWidth = true
		case strings.Contains(colTypeLow, "datetime"),
			strings.Contains(colTypeLow, "timestamp"):
			meta.HTMLType = "datetime-local"
		case strings.HasPrefix(colTypeLow, "date"):
			meta.HTMLType = "date"
		case strings.HasPrefix(colTypeLow, "time"):
			meta.HTMLType = "time"
		case strings.HasPrefix(colTypeLow, "year"):
			meta.HTMLType = "number"
		case strings.Contains(colTypeLow, "bool"),
			strings.Contains(colTypeLow, "tinyint(1)"):
			meta.HTMLType = "checkbox"
		case strings.Contains(colTypeLow, "int"):
			meta.HTMLType = "number"
		case strings.Contains(colTypeLow, "float"),
			strings.Contains(colTypeLow, "double"),
			strings.Contains(colTypeLow, "decimal"),
			strings.Contains(colTypeLow, "numeric"):
			meta.HTMLType = "number"
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

	// Mandatory interactive terminal choice for database driver
	var driverType string
	if !hasDriver {
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
gf-gen-crud --overwrite %*

echo === Running gf gen service ===
gf gen service

echo === Done! ===
`
		_ = writeFile(genBatPath, genBatContent, false)
	}

	fmt.Printf("Module  : %s\n", moduleName)
	fmt.Printf("Entities: %s\n\n", entityDir)

	files, err := filepath.Glob(filepath.Join(entityDir, "*.go"))
	if err != nil || len(files) == 0 {
		fmt.Println("Tidak ada file entity ditemukan di", entityDir)
		os.Exit(1)
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
					if meta, ok := dbCols[fi.OrmTag]; ok {
						fields[i].HTMLType = meta.HTMLType
						fields[i].IsTextarea = meta.IsTextarea
						fields[i].EnumValues = meta.EnumValues
						fields[i].IsJson = meta.IsJson
						fields[i].IsFullWidth = meta.IsFullWidth
						if meta.HTMLType == "file" {
							info.HasUpload = true
						}
					} else {
						lowName := strings.ToLower(fi.Name)
						if fi.IsTextarea || fi.HTMLType == "file" || fi.HTMLType == "textarea" ||
							strings.Contains(lowName, "address") || strings.Contains(lowName, "alamat") ||
							strings.Contains(lowName, "desc") || strings.Contains(lowName, "deskripsi") ||
							strings.Contains(lowName, "image") || strings.Contains(lowName, "file") ||
							strings.Contains(lowName, "path") || strings.Contains(lowName, "url") ||
							strings.Contains(lowName, "avatar") || strings.Contains(lowName, "cover") {
							fields[i].IsFullWidth = true
						}
					}
				}
				return fields
			}
			info.Fields = enrichFields(info.Fields)
			info.ListFields = enrichFields(info.ListFields)
			info.FormFields = enrichFields(info.FormFields)
		}
		allEntities = append(allEntities, info)
	}

	if *tableFlag != "" {
		tableSet := map[string]bool{}
		for _, t := range strings.Split(*tableFlag, ",") {
			tableSet[strings.TrimSpace(t)] = true
		}
		for _, info := range allEntities {
			if !tableSet[info.TableName] {
				continue
			}
			_ = info
		}
	}

	globalIsReadOnly := false
	globalFilterType := "input"
	if !*skipView {
		// Use --view-mode flag if provided, otherwise prompt
		if *viewModeFlag != "" {
			globalIsReadOnly = (*viewModeFlag == "view")
		} else {
			viewPrompt := promptui.Select{
				Label: "Mode view untuk tabel yang akan digenerate",
				Items: []string{
					"write — tampilkan create, edit, delete",
					"view  — hanya lihat, tanpa aksi tulis",
				},
			}
			viewIdx, _, _ := viewPrompt.Run()
			globalIsReadOnly = viewIdx == 1
		}

		// Use --filter-type flag if provided, otherwise prompt
		if *filterTypeFlag != "" {
			globalFilterType = *filterTypeFlag
		} else {
			filterPrompt := promptui.Select{
				Label: "Filter di halaman list",
				Items: []string{
					"input — satu kolom pencarian (inline di list page)",
					"form  — form per-kolom (halaman /filter terpisah)",
					"both  — keyword inline + tombol buka halaman /filter",
					"none  — tidak ada filter",
				},
			}
			filterIdx, _, _ := filterPrompt.Run()
			switch filterIdx {
			case 0:
				globalFilterType = "input"
			case 1:
				globalFilterType = "form"
			case 2:
				globalFilterType = "both"
			default:
				globalFilterType = "none"
			}
		}
	}

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

		info.IsReadOnly = globalIsReadOnly
		info.FilterType = globalFilterType

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
				logicPath := filepath.Join(root, "internal", "logic", info.VarName, info.VarName+".go")
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
}
