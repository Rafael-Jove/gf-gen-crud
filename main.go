// Generator CRUD otomatis untuk GoFrame.
// Membaca semua file entity di internal/model/entity dan men-generate:
//   - api/<name>/v1/<name>.go         (Request/Response structs)
//   - internal/logic/<name>/<name>.go (CRUD logic)
//   - resource/template/<name>/list.html, form.html, detail.html
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
	HasGtime   bool // true jika ada field bertipe gtime.Time
	HasUpload  bool // true jika ada field bertipe file
}

// CmdControllerInfo holds info to build cmd.go
type CmdControllerInfo struct {
	PackageName string
	TableName   string
	HasHTML     bool
	ShortName   string
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
	// Format: mysql:user:pass@tcp(host:port)/dbname
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
	// Extract dbname
	slashIdx := strings.Index(hostDB, "/")
	if slashIdx < 0 {
		return "", false
	}
	hostPart := hostDB[:slashIdx]
	dbname := hostDB[slashIdx+1:]
	// Strip query params from dbname
	if q := strings.Index(dbname, "?"); q >= 0 {
		dbname = dbname[:q]
	}
	// hostPart = tcp(127.0.0.1:3306) → 127.0.0.1:3306
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
	// Extract link value from YAML (simple regex, no full YAML parser needed)
	// Matches: link: "mysql:root:@tcp(127.0.0.1:3306)/dbname" or link: 'mysql:...'
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
			// enum('val1','val2') → ["val1","val2"]
			inner := colType[5 : len(colType)-1] // strip enum( and )
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
		// Check varchar length to decide layout width
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
					continue // embedded field like g.Meta
				}
				fieldName := field.Names[0].Name

				// Get type string
				var typeBuf bytes.Buffer
				_ = ast.Fprint(&typeBuf, fset, field.Type, ast.NotNilFilter)
				typStr := goTypeStr(field.Type)

				// Get tags
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

				_ = typeBuf

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
// Templates
// ============================================================

var apiTemplate = template.Must(template.New("api").Funcs(template.FuncMap{
	"contains": strings.Contains,
}).Parse(`package v1

import (
	"github.com/gogf/gf/v2/frame/g"
{{- $hasGtime := false}}
{{- range .FormFields}}
{{- if contains .Type "gtime"}}{{$hasGtime = true}}{{end}}
{{- end}}
{{- if $hasGtime}}
	"github.com/gogf/gf/v2/os/gtime"
{{- end}}
{{- if .HasUpload}}
	"github.com/gogf/gf/v2/net/ghttp"
{{- end}}
)

// ---------- List ----------

type List{{.ShortName}}Req struct {
	g.Meta   ` + "`" + `path:"/{{.TableName}}" method:"get" tags:"{{.StructName}}" summary:"Daftar {{.ShortName}}"` + "`" + `
	Page     int ` + "`" + `json:"page" d:"1"` + "`" + `
	PageSize int ` + "`" + `json:"page_size" d:"10"` + "`" + `
	Keyword  string ` + "`" + `json:"keyword"` + "`" + `
}

type List{{.ShortName}}Res struct {
	List  interface{} ` + "`" + `json:"list"` + "`" + `
	Total int         ` + "`" + `json:"total"` + "`" + `
	Page  int         ` + "`" + `json:"page"` + "`" + `
}

// ---------- Get ----------

type Get{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}" method:"get" tags:"{{.StructName}}" summary:"Detail {{.ShortName}}"` + "`" + `
	Id     uint64 ` + "`" + `json:"id" v:"required#ID wajib diisi"` + "`" + `
}

type Get{{.ShortName}}Res struct {
	Data interface{} ` + "`" + `json:"data"` + "`" + `
}

type Create{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}" method:"post" tags:"{{.StructName}}" summary:"Buat {{.ShortName}} baru"` + "`" + `
{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	{{.Name}} *ghttp.UploadFile ` + "`" + `json:"{{.JsonTag}}" v:"required#{{.Name}} wajib diisi"` + "`" + `
	{{- else}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JsonTag}}" v:"required#{{.Name}} wajib diisi"` + "`" + `
	{{- end}}
{{- end}}
}

type Create{{.ShortName}}Res struct {
	Data interface{} ` + "`" + `json:"data"` + "`" + `
}

// ---------- Update ----------

type Update{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}" method:"post" tags:"{{.StructName}}" summary:"Update {{.ShortName}}"` + "`" + `
	Id     uint64 ` + "`" + `json:"id" v:"required#ID wajib diisi"` + "`" + `
{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	{{.Name}} *ghttp.UploadFile ` + "`" + `json:"{{.JsonTag}}"` + "`" + `
	{{- else}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JsonTag}}"` + "`" + `
	{{- end}}
{{- end}}
}

type Update{{.ShortName}}Res struct {
	Message string ` + "`" + `json:"message"` + "`" + `
}

// ---------- Delete ----------

type Delete{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}" method:"delete" tags:"{{.StructName}}" summary:"Hapus {{.ShortName}}"` + "`" + `
	Id     uint64 ` + "`" + `json:"id" v:"required#ID wajib diisi"` + "`" + `
}

type Delete{{.ShortName}}Res struct {
	Message string ` + "`" + `json:"message"` + "`" + `
}
`))

var apiInterfaceTemplate = template.Must(template.New("api_interface").Parse(`// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package {{.TableName}}

import (
	"context"

	"{{.ModuleName}}/api/{{.TableName}}/v1"
)

type I{{.StructName}}V1 interface {
	List{{.ShortName}}(ctx context.Context, req *v1.List{{.ShortName}}Req) (res *v1.List{{.ShortName}}Res, err error)
	Get{{.ShortName}}(ctx context.Context, req *v1.Get{{.ShortName}}Req) (res *v1.Get{{.ShortName}}Res, err error)
	Create{{.ShortName}}(ctx context.Context, req *v1.Create{{.ShortName}}Req) (res *v1.Create{{.ShortName}}Res, err error)
	Update{{.ShortName}}(ctx context.Context, req *v1.Update{{.ShortName}}Req) (res *v1.Update{{.ShortName}}Res, err error)
	Delete{{.ShortName}}(ctx context.Context, req *v1.Delete{{.ShortName}}Req) (res *v1.Delete{{.ShortName}}Res, err error)
}
`))

var logicTemplate = template.Must(template.New("logic").Parse(`package {{.VarName}}

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"{{.ModuleName}}/internal/dao"
	"{{.ModuleName}}/internal/model/do"
	"{{.ModuleName}}/internal/model/entity"
	"{{.ModuleName}}/internal/service"
)

type s{{.StructName}} struct{}

func init() {
	service.Register{{.StructName}}(New())
}

func New() *s{{.StructName}} {
	return &s{{.StructName}}{}
}

// List mengambil daftar {{.ShortName}} dengan pagination dan keyword filter.
func (s *s{{.StructName}}) List(ctx context.Context, page, pageSize int, keyword string) (list []*entity.{{.StructName}}, total int, err error) {
	m := dao.{{.StructName}}.Ctx(ctx)
	{{- $hasStringField := false}}
	{{- range .ListFields}}
	{{- if eq .Type "string"}}{{$hasStringField = true}}{{end}}
	{{- end}}
	{{- if $hasStringField}}
	if keyword != "" {
		m = m.Where(func(m *gdb.Model) {
			{{- $first := true}}
			{{- range .ListFields}}
			{{- if eq .Type "string"}}
			{{- if $first}}
			m.Where("` + "`" + `{{.OrmTag}}` + "`" + ` LIKE ?", "%"+keyword+"%")
			{{- $first = false}}
			{{- else}}
			m.WhereOr("` + "`" + `{{.OrmTag}}` + "`" + ` LIKE ?", "%"+keyword+"%")
			{{- end}}
			{{- end}}
			{{- end}}
		})
	}
	{{- end}}
	total, err = m.Count()
	if err != nil {
		return
	}
	err = m.Page(page, pageSize).Scan(&list)
	return
}

// Get mengambil satu {{.ShortName}} berdasarkan ID.
func (s *s{{.StructName}}) Get(ctx context.Context, id uint64) (data *entity.{{.StructName}}, err error) {
	err = dao.{{.StructName}}.Ctx(ctx).WherePri(id).Scan(&data)
	return
}

// Create membuat {{.ShortName}} baru.
func (s *s{{.StructName}}) Create(ctx context.Context, in do.{{.StructName}}) (data *entity.{{.StructName}}, err error) {
	lastId, err := dao.{{.StructName}}.Ctx(ctx).Data(in).InsertAndGetId()
	if err != nil {
		return
	}
	return s.Get(ctx, uint64(lastId))
}

// Update mengupdate {{.ShortName}} berdasarkan ID.
func (s *s{{.StructName}}) Update(ctx context.Context, id uint64, in do.{{.StructName}}) (err error) {
	_, err = dao.{{.StructName}}.Ctx(ctx).WherePri(id).Data(in).Update()
	return
}

// Delete menghapus {{.ShortName}} berdasarkan ID.
func (s *s{{.StructName}}) Delete(ctx context.Context, id uint64) (err error) {
	_, err = dao.{{.StructName}}.Ctx(ctx).WherePri(id).Delete()
	return
}
`))

var listHTMLTemplate = template.Must(template.New("list_html").Funcs(template.FuncMap{
	"slice": func(s string, start, end int) string {
		if start < 0 {
			start = 0
		}
		if end > len(s) {
			end = len(s)
		}
		if start > end {
			return ""
		}
		return s[start:end]
	},
	"add": func(a, b int) int {
		return a + b
	},
	"subtract": func(a, b int) int {
		return a - b
	},
	"multiply": func(a, b int) int {
		return a * b
	},
	"min": func(a, b int) int {
		if a < b {
			return a
		}
		return b
	},
	"abbrev": abbrev,
}).Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard {{.ShortName}}</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
        #sidebar.w-18 { width: 4.5rem !important; }
        #sidebar.w-18 .sidebar-text { display: none !important; }
        #sidebar.w-18 .nav-item { justify-content: center !important; }
        #sidebar.w-18 .nav-item-icon { margin-right: 0 !important; }
        #sidebar.w-18 .logo-container { justify-content: center !important; }
        #sidebar.w-18 #toggle-button-collapsed { display: flex !important; }
        #sidebar.w-18 .footer-container { justify-content: center !important; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen">
    <div class="flex h-screen overflow-hidden">
        
        <!-- Sidebar Navigation (Desktop) -->
        <aside id="sidebar" class="hidden md:flex md:flex-col md:flex-shrink-0 w-64 bg-slate-900 text-slate-300 border-r border-slate-800 transition-all duration-300 relative">
            <div class="logo-container flex items-center justify-between h-16 px-4 bg-slate-950 border-b border-slate-800/50">
                <div class="flex items-center space-x-3 overflow-hidden">
                    <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        {{slice .ShortName 0 1}}
                    </div>
                    <span class="sidebar-text text-white font-bold text-lg tracking-tight whitespace-nowrap">Admin Console</span>
                </div>
                <button onclick="toggleSidebar()" class="sidebar-text p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
                    </svg>
                </button>
                <button id="toggle-button-collapsed" onclick="toggleSidebar()" class="hidden p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center mx-auto">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>
            <div class="flex-1 flex flex-col overflow-y-auto px-3 py-6 space-y-1.5">
                <span class="sidebar-text px-3 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 whitespace-nowrap">Modul Data</span>
                {{- range .NavItems}}
                <a href="/{{.TableName}}" class="nav-item flex items-center px-3 py-2.5 text-sm font-medium rounded-xl transition-all group {{if .Active}}bg-indigo-600 text-white shadow-md shadow-indigo-600/10{{else}}text-slate-400 hover:bg-slate-800 hover:text-slate-200{{end}}">
                    <span class="nav-item-icon flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold mr-3 transition-all {{if .Active}}bg-indigo-500 text-white{{else}}bg-slate-800 text-slate-400 group-hover:bg-slate-700 group-hover:text-slate-200{{end}}">
                        {{abbrev .Name}}
                    </span>
                    <span class="sidebar-text whitespace-nowrap">{{.Name}}</span>
                </a>
                {{- end}}
            </div>
            <div class="footer-container p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center space-x-2 overflow-hidden">
                <span class="flex-shrink-0 w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                <span class="sidebar-text whitespace-nowrap">Sistem Aktif (Port: 8000)</span>
            </div>
        </aside>

        <!-- Content Area -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            
            <!-- Mobile Header -->
            <header class="md:hidden bg-white border-b border-slate-200 h-16 px-4 flex items-center justify-between sticky top-0 z-30 shadow-sm">
                <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center text-white font-bold text-base">
                        {{slice .ShortName 0 1}}
                    </div>
                    <span class="font-bold text-slate-900">{{.ShortName}} Manager</span>
                </div>
                <div class="text-xs text-slate-500 flex items-center space-x-2">
                    <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
                    <span>Aktif</span>
                </div>
            </header>

            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 max-w-7xl">
                
                <!-- Mobile Navigation Tabs -->
                <div class="flex md:hidden overflow-x-auto py-2 mb-6 border-b border-slate-200 space-x-2 scrollbar-none">
                    {{- range .NavItems}}
                    <a href="/{{.TableName}}" class="whitespace-nowrap px-4 py-1.5 text-xs font-semibold rounded-full transition-all {{if .Active}}bg-indigo-600 text-white{{else}}bg-slate-100 text-slate-600 hover:bg-slate-200{{end}}">
                        {{.Name}}
                    </a>
                    {{- end}}
                </div>

                <!-- Header Section -->
                <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
                    <div>
                        <h1 class="text-2xl font-bold text-slate-900 font-bold">Daftar {{.ShortName}}</h1>
                        <p class="text-sm text-slate-500 mt-0.5">Kelola data {{.TableName}} dengan mudah.</p>
                    </div>
                    <div>
                        <a href="/{{.TableName}}/create" class="inline-flex items-center justify-center px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl shadow-md shadow-indigo-600/10 transition-all gap-2 w-full sm:w-auto">
                            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                            </svg>
                            Tambah {{.ShortName}}
                        </a>
                    </div>
                </div>

                <!-- Filter Card -->
                <div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-4 mb-6">
                    <form method="GET" action="/{{.TableName}}" class="flex flex-col sm:flex-row items-center gap-3">
                        <div class="relative flex-1 w-full">
                            <div class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                                </svg>
                            </div>
                            <input 
                                type="text" 
                                name="keyword" 
                                value="{{"{{"}} .Keyword {{"}}"}}" 
                                placeholder="Cari berdasarkan kata kunci..." 
                                class="w-full pl-10 pr-4 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm placeholder-slate-400"
                            />
                        </div>
                        <div class="flex items-center gap-2 w-full sm:w-auto">
                            <select name="page_size" onchange="this.form.submit()" class="px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white text-slate-700">
                                <option value="10" {{"{{"}} if eq .PageSize 10 {{"}}"}}selected{{"{{"}} end {{"}}"}}>10 per hal</option>
                                <option value="20" {{"{{"}} if eq .PageSize 20 {{"}}"}}selected{{"{{"}} end {{"}}"}}>20 per hal</option>
                                <option value="50" {{"{{"}} if eq .PageSize 50 {{"}}"}}selected{{"{{"}} end {{"}}"}}>50 per hal</option>
                                <option value="100" {{"{{"}} if eq .PageSize 100 {{"}}"}}selected{{"{{"}} end {{"}}"}}>100 per hal</option>
                            </select>
                            <button type="submit" class="flex-1 sm:flex-initial px-5 py-2.5 bg-slate-900 hover:bg-slate-800 text-white font-semibold text-sm rounded-xl transition-all">
                                Filter
                            </button>
                            {{"{{"}} if .Keyword {{"}}"}}
                            <a href="/{{.TableName}}" class="flex-1 sm:flex-initial px-5 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-sm rounded-xl transition-all text-center">
                                Reset
                            </a>
                            {{"{{"}} end {{"}}"}}
                        </div>
                    </form>
                </div>

                <!-- Table Card -->
                <div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm overflow-hidden">
                    <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
                        <h2 class="font-bold text-slate-900">Total: {{"{{"}} .Total {{"}}"}} data</h2>
                    </div>

                    <div class="overflow-x-auto relative">
                        <table class="w-full text-left border-collapse min-w-[800px]">
                            <thead>
                                <tr class="bg-slate-50 border-b border-slate-100">
                                    {{- range .ListFields}}
                                    <th class="px-6 py-3.5 text-xs font-bold text-slate-500 uppercase tracking-wider">{{.Name}}</th>
                                    {{- end}}
                                    <th class="sticky right-0 bg-slate-50 px-6 py-3.5 text-xs font-bold text-slate-500 uppercase tracking-wider text-right shadow-[-4px_0_8px_rgba(0,0,0,0.05)] z-20">Aksi</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-slate-100">
                                {{"{{"}}range .List{{"}}"}}
                                <tr class="group hover:bg-slate-50/50 transition-colors">
                                    {{- range .ListFields}}
                                    <td class="px-6 py-4 text-sm text-slate-700 font-medium">
                                        {{- if eq .HTMLType "file"}}
                                        {{"{{"}} if .{{.Name}} {{"}}"}}
                                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-50 text-indigo-700 border border-indigo-100">
                                            Ada File
                                        </span>
                                        {{"{{"}} else {{"}}"}}
                                        <span class="text-slate-400 font-normal">-</span>
                                        {{"{{"}} end {{"}}"}}
                                        {{- else}}
                                        {{"{{"}} .{{.Name}} {{"}}"}}
                                        {{- end}}
                                    </td>
                                    {{- end}}
                                    <td class="sticky right-0 bg-white group-hover:bg-slate-50/50 px-6 py-4 text-sm text-slate-700 text-right space-x-3 shadow-[-4px_0_8px_rgba(0,0,0,0.05)] z-10">
                                        <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}" class="inline-flex items-center text-xs font-semibold text-emerald-600 hover:text-emerald-950 transition-colors">
                                            Show
                                        </a>
                                        <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}/edit" class="inline-flex items-center text-xs font-semibold text-indigo-600 hover:text-indigo-900 transition-colors">
                                            Update
                                        </a>
                                        <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}/delete" onclick="return confirm('Hapus data ini?')" class="inline-flex items-center text-xs font-semibold text-rose-600 hover:text-rose-900 transition-colors">
                                            Delete
                                        </a>
                                    </td>
                                </tr>
                                {{"{{"}}else{{"}}"}}
                                <tr>
                                    <td colspan="{{add (len .ListFields) 1}}" class="px-6 py-10 text-center text-sm text-slate-400">
                                        Tidak ada data ditemukan.
                                    </td>
                                </tr>
                                {{"{{"}}end{{"}}"}}
                            </tbody>
                        </table>
                    </div>

                    <!-- Pagination Controls -->
                    <div class="px-6 py-4 border-t border-slate-100 flex items-center justify-between">
                        <div class="text-sm text-slate-500">
                            Menampilkan <span class="font-semibold text-slate-800">{{"{{"}}add (multiply (subtract .Page 1) .PageSize) 1{{"}}"}}</span> - <span class="font-semibold text-slate-800">{{"{{"}}min (multiply .Page .PageSize) .Total{{"}}"}}</span> dari <span class="font-semibold text-slate-800">{{"{{"}}.Total{{"}}"}}</span> data
                        </div>
                        <div class="flex items-center space-x-2">
                            {{"{{"}} if gt .Page 1 {{"}}"}}
                            <a href="?page={{"{{"}}subtract .Page 1{{"}}"}}&page_size={{"{{"}}.PageSize{{"}}"}}&keyword={{"{{"}}.Keyword{{"}}"}}" class="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-xs rounded-xl transition-all">
                                Sebelumnya
                            </a>
                            {{"{{"}} else {{"}}"}}
                            <span class="px-4 py-2 bg-slate-50 text-slate-300 font-semibold text-xs rounded-xl cursor-not-allowed">
                                Sebelumnya
                            </span>
                            {{"{{"}} end {{"}}"}}

                            {{"{{"}} if lt (multiply .Page .PageSize) .Total {{"}}"}}
                            <a href="?page={{"{{"}}add .Page 1{{"}}"}}&page_size={{"{{"}}.PageSize{{"}}"}}&keyword={{"{{"}}.Keyword{{"}}"}}" class="px-4 py-2 bg-slate-900 hover:bg-slate-800 text-white font-semibold text-xs rounded-xl transition-all">
                                Berikutnya
                            </a>
                            {{"{{"}} else {{"}}"}}
                            <span class="px-4 py-2 bg-slate-50 text-slate-300 font-semibold text-xs rounded-xl cursor-not-allowed">
                                Berikutnya
                            </span>
                            {{"{{"}} end {{"}}"}}
                        </div>
                    </div>
                </div>
            </main>
        </div>
    </div>
    <script>
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const isCollapsed = sidebar.classList.toggle('w-18');
            if (isCollapsed) {
                sidebar.classList.remove('w-64');
                localStorage.setItem('sidebar-collapsed', 'true');
            } else {
                sidebar.classList.add('w-64');
                localStorage.setItem('sidebar-collapsed', 'false');
            }
        }
        document.addEventListener('DOMContentLoaded', () => {
            if (localStorage.getItem('sidebar-collapsed') === 'true') {
                const sidebar = document.getElementById('sidebar');
                if (sidebar) {
                    sidebar.classList.remove('w-64');
                    sidebar.classList.add('w-18');
                }
            }
        });
    </script>
</body>
</html>
`))

var formHTMLTemplate = template.Must(template.New("form_html").Funcs(template.FuncMap{
	"slice": func(s string, start, end int) string {
		if start < 0 {
			start = 0
		}
		if end > len(s) {
			end = len(s)
		}
		if start > end {
			return ""
		}
		return s[start:end]
	},
	"abbrev": abbrev,
}).Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Form {{.ShortName}}</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
        #sidebar.w-18 { width: 4.5rem !important; }
        #sidebar.w-18 .sidebar-text { display: none !important; }
        #sidebar.w-18 .nav-item { justify-content: center !important; }
        #sidebar.w-18 .nav-item-icon { margin-right: 0 !important; }
        #sidebar.w-18 .logo-container { justify-content: center !important; }
        #sidebar.w-18 #toggle-button-collapsed { display: flex !important; }
        #sidebar.w-18 .footer-container { justify-content: center !important; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen">
    <div class="flex h-screen overflow-hidden">
        
        <!-- Sidebar Navigation (Desktop) -->
        <aside id="sidebar" class="hidden md:flex md:flex-col md:flex-shrink-0 w-64 bg-slate-900 text-slate-300 border-r border-slate-800 transition-all duration-300 relative">
            <div class="logo-container flex items-center justify-between h-16 px-4 bg-slate-950 border-b border-slate-800/50">
                <div class="flex items-center space-x-3 overflow-hidden">
                    <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        {{slice .ShortName 0 1}}
                    </div>
                    <span class="sidebar-text text-white font-bold text-lg tracking-tight whitespace-nowrap">Admin Console</span>
                </div>
                <button onclick="toggleSidebar()" class="sidebar-text p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
                    </svg>
                </button>
                <button id="toggle-button-collapsed" onclick="toggleSidebar()" class="hidden p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center mx-auto">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>
            <div class="flex-1 flex flex-col overflow-y-auto px-3 py-6 space-y-1.5">
                <span class="sidebar-text px-3 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 whitespace-nowrap">Modul Data</span>
                {{- range .NavItems}}
                <a href="/{{.TableName}}" class="nav-item flex items-center px-3 py-2.5 text-sm font-medium rounded-xl transition-all group {{if .Active}}bg-indigo-600 text-white shadow-md shadow-indigo-600/10{{else}}text-slate-400 hover:bg-slate-800 hover:text-slate-200{{end}}">
                    <span class="nav-item-icon flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold mr-3 transition-all {{if .Active}}bg-indigo-500 text-white{{else}}bg-slate-800 text-slate-400 group-hover:bg-slate-700 group-hover:text-slate-200{{end}}">
                        {{abbrev .Name}}
                    </span>
                    <span class="sidebar-text whitespace-nowrap">{{.Name}}</span>
                </a>
                {{- end}}
            </div>
            <div class="footer-container p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center space-x-2 overflow-hidden">
                <span class="flex-shrink-0 w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                <span class="sidebar-text whitespace-nowrap">Sistem Aktif (Port: 8000)</span>
            </div>
        </aside>

        <!-- Content Area -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            
            <!-- Mobile Header -->
            <header class="md:hidden bg-white border-b border-slate-200 h-16 px-4 flex items-center justify-between sticky top-0 z-30 shadow-sm">
                <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center text-white font-bold text-base">
                        {{slice .ShortName 0 1}}
                    </div>
                    <span class="font-bold text-slate-900">{{.ShortName}} Manager</span>
                </div>
                <div class="text-xs text-slate-500 flex items-center space-x-2">
                    <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
                    <span>Aktif</span>
                </div>
            </header>

            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 max-w-4xl">
                <!-- Mobile Navigation Tabs -->
                <div class="flex md:hidden overflow-x-auto py-2 mb-6 border-b border-slate-200 space-x-2 scrollbar-none">
                    {{- range .NavItems}}
                    <a href="/{{.TableName}}" class="whitespace-nowrap px-4 py-1.5 text-xs font-semibold rounded-full transition-all {{if .Active}}bg-indigo-600 text-white{{else}}bg-slate-100 text-slate-600 hover:bg-slate-200{{end}}">
                        {{.Name}}
                    </a>
                    {{- end}}
                </div>

                <!-- Form Card -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6">
                    <h1 class="text-xl font-bold text-slate-900 mb-2">{{"{{"}} if .Id {{"}}"}}Edit{{"{{"}} else {{"}}"}}Tambah{{"{{"}} end {{"}}"}} {{.ShortName}}</h1>
                    <p class="text-sm text-slate-500 mb-6">Isi formulir berikut untuk menyimpan data.</p>
                    
                    <form method="POST" action="/{{.TableName}}{{"{{"}} if .Id {{"}}"}}/{{"{{"}} .Id {{"}}"}}{{"{{"}} end {{"}}"}}" class="space-y-6"{{if .HasUpload}} enctype="multipart/form-data"{{end}}>
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
                            {{- range .FormFields}}
                            <div class="{{if .IsFullWidth}}md:col-span-2{{else}}md:col-span-1{{end}}">
                                <label class="block text-sm font-semibold text-slate-700 mb-1.5">{{.Name}}</label>
                                {{- if .EnumValues}}
                                {{- $fieldName := .Name}}
                                <select name="{{.JsonTag}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white">
                                    <option value="">-- Pilih {{.Name}} --</option>
                                    {{- range .EnumValues}}
                                    <option value="{{.}}" {{"{{"}} if eq (printf "%v" $.{{$fieldName}}) "{{.}}" {{"}}"}}selected{{"{{"}} end {{"}}"}}>{{.}}</option>
                                    {{- end}}
                                </select>
                                {{- else if .IsTextarea}}
                                <textarea name="{{.JsonTag}}" rows="4" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm resize-y" placeholder="{{if .IsJson}}Masukkan {{.Name}} (Format JSON, contoh: {}){{else}}Masukkan {{.Name}}...{{end}}">{{"{{"}} .{{.Name}} {{"}}"}}</textarea>
                                {{- else if eq .HTMLType "file"}}
                                <input type="file" name="{{.JsonTag}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm file:mr-4 file:py-1.5 file:px-3 file:rounded-xl file:border-0 file:text-xs file:font-semibold file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100" />
                                {{- else if eq .HTMLType "date"}}
                                <input type="date" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}}.Layout "2006-01-02" {{"}}"}}{{"{{"}} end {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                {{- else if eq .HTMLType "time"}}
                                <input type="time" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}}.Layout "15:04:05" {{"}}"}}{{"{{"}} end {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                {{- else if eq .HTMLType "datetime-local"}}
                                <input type="datetime-local" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}}.Layout "2006-01-02T15:04" {{"}}"}}{{"{{"}} end {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                {{- else if eq .HTMLType "checkbox"}}
                                <label class="flex items-center gap-2 cursor-pointer pt-2">
                                    <input type="checkbox" name="{{.JsonTag}}" value="1" {{"{{"}} if .{{.Name}} {{"}}"}}checked{{"{{"}} end {{"}}"}} class="w-4 h-4 accent-indigo-600" />
                                    <span class="text-sm text-slate-600">Aktif</span>
                                </label>
                                {{- else}}
                                <input type="{{.HTMLType}}" name="{{.JsonTag}}" value="{{"{{"}} .{{.Name}} {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                {{- end}}
                            </div>
                            {{- end}}
                        </div>
                        <div class="pt-4 border-t border-slate-100 flex items-center space-x-3">
                            <button type="submit" class="flex-1 px-4 py-2.5 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-700 hover:to-violet-700 text-white font-semibold text-sm rounded-xl shadow-md transition-all">Simpan</button>
                            <a href="/{{.TableName}}" class="px-4 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-sm rounded-xl text-center">Batal</a>
                        </div>
                    </form>
                </div>
            </main>
        </div>
    </div>
    <script>
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const isCollapsed = sidebar.classList.toggle('w-18');
            if (isCollapsed) {
                sidebar.classList.remove('w-64');
                localStorage.setItem('sidebar-collapsed', 'true');
            } else {
                sidebar.classList.add('w-64');
                localStorage.setItem('sidebar-collapsed', 'false');
            }
        }
        document.addEventListener('DOMContentLoaded', () => {
            if (localStorage.getItem('sidebar-collapsed') === 'true') {
                const sidebar = document.getElementById('sidebar');
                if (sidebar) {
                    sidebar.classList.remove('w-64');
                    sidebar.classList.add('w-18');
                }
            }
        });
    </script>
</body>
</html>
`))

var detailHTMLTemplate = template.Must(template.New("detail_html").Funcs(template.FuncMap{
	"slice": func(s string, start, end int) string {
		if start < 0 {
			start = 0
		}
		if end > len(s) {
			end = len(s)
		}
		if start > end {
			return ""
		}
		return s[start:end]
	},
	"abbrev": abbrev,
}).Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Detail {{.ShortName}}</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
        #sidebar.w-18 { width: 4.5rem !important; }
        #sidebar.w-18 .sidebar-text { display: none !important; }
        #sidebar.w-18 .nav-item { justify-content: center !important; }
        #sidebar.w-18 .nav-item-icon { margin-right: 0 !important; }
        #sidebar.w-18 .logo-container { justify-content: center !important; }
        #sidebar.w-18 #toggle-button-collapsed { display: flex !important; }
        #sidebar.w-18 .footer-container { justify-content: center !important; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen">
    <div class="flex h-screen overflow-hidden">
        
        <!-- Sidebar Navigation (Desktop) -->
        <aside id="sidebar" class="hidden md:flex md:flex-col md:flex-shrink-0 w-64 bg-slate-900 text-slate-300 border-r border-slate-800 transition-all duration-300 relative">
            <div class="logo-container flex items-center justify-between h-16 px-4 bg-slate-950 border-b border-slate-800/50">
                <div class="flex items-center space-x-3 overflow-hidden">
                    <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        {{slice .ShortName 0 1}}
                    </div>
                    <span class="sidebar-text text-white font-bold text-lg tracking-tight whitespace-nowrap">Admin Console</span>
                </div>
                <button onclick="toggleSidebar()" class="sidebar-text p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
                    </svg>
                </button>
                <button id="toggle-button-collapsed" onclick="toggleSidebar()" class="hidden p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center mx-auto">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>
            <div class="flex-1 flex flex-col overflow-y-auto px-3 py-6 space-y-1.5">
                <span class="sidebar-text px-3 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 whitespace-nowrap">Modul Data</span>
                {{- range .NavItems}}
                <a href="/{{.TableName}}" class="nav-item flex items-center px-3 py-2.5 text-sm font-medium rounded-xl transition-all group {{if .Active}}bg-indigo-600 text-white shadow-md shadow-indigo-600/10{{else}}text-slate-400 hover:bg-slate-800 hover:text-slate-200{{end}}">
                    <span class="nav-item-icon flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold mr-3 transition-all {{if .Active}}bg-indigo-500 text-white{{else}}bg-slate-800 text-slate-400 group-hover:bg-slate-700 group-hover:text-slate-200{{end}}">
                        {{abbrev .Name}}
                    </span>
                    <span class="sidebar-text whitespace-nowrap">{{.Name}}</span>
                </a>
                {{- end}}
            </div>
            <div class="footer-container p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center space-x-2 overflow-hidden">
                <span class="flex-shrink-0 w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                <span class="sidebar-text whitespace-nowrap">Sistem Aktif (Port: 8000)</span>
            </div>
        </aside>

        <!-- Content Area -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            
            <!-- Mobile Header -->
            <header class="md:hidden bg-white border-b border-slate-200 h-16 px-4 flex items-center justify-between sticky top-0 z-30 shadow-sm">
                <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center text-white font-bold text-base">
                        {{slice .ShortName 0 1}}
                    </div>
                    <span class="font-bold text-slate-900">{{.ShortName}} Manager</span>
                </div>
                <div class="text-xs text-slate-500 flex items-center space-x-2">
                    <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
                    <span>Aktif</span>
                </div>
            </header>

            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 max-w-4xl">
                <!-- Mobile Navigation Tabs -->
                <div class="flex md:hidden overflow-x-auto py-2 mb-6 border-b border-slate-200 space-x-2 scrollbar-none">
                    {{- range .NavItems}}
                    <a href="/{{.TableName}}" class="whitespace-nowrap px-4 py-1.5 text-xs font-semibold rounded-full transition-all {{if .Active}}bg-indigo-600 text-white{{else}}bg-slate-100 text-slate-600 hover:bg-slate-200{{end}}">
                        {{.Name}}
                    </a>
                    {{- end}}
                </div>

                <!-- Detail Card -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6">
                    <h1 class="text-xl font-bold text-slate-900 mb-4">Detail {{.ShortName}} #{{"{{"}} .Id {{"}}"}}</h1>
                    
                    <div class="overflow-hidden border border-slate-100 rounded-xl mb-6">
                        <table class="w-full text-left border-collapse text-sm">
                            <tbody class="divide-y divide-slate-100">
                                {{- range .Fields}}
                                <tr class="hover:bg-slate-50/50 transition-colors">
                                    <th class="px-4 py-3 font-semibold text-slate-500 w-1/3 bg-slate-50/50">{{.Name}}</th>
                                    <td class="px-4 py-3 text-slate-700 break-words whitespace-pre-wrap">
                                        {{- if eq .HTMLType "file"}}
                                        {{"{{"}} if .{{.Name}} {{"}}"}}
                                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-50 text-indigo-700 border border-indigo-100">
                                            Ada File
                                        </span>
                                        {{"{{"}} else {{"}}"}}
                                        <span class="text-slate-400">-</span>
                                        {{"{{"}} end {{"}}"}}
                                        {{- else}}
                                        {{"{{"}} .{{.Name}} {{"}}"}}
                                        {{- end}}
                                    </td>
                                </tr>
                                {{- end}}
                            </tbody>
                        </table>
                    </div>

                    <div class="flex items-center space-x-3">
                        <a href="/{{.TableName}}/{{"{{"}} .Id {{"}}"}}/edit" class="flex-1 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl text-center shadow-md transition-all">Edit</a>
                        <a href="/{{.TableName}}" class="px-4 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-sm rounded-xl text-center transition-all">Kembali</a>
                    </div>
                </div>
            </main>
        </div>
    </div>
    <script>
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const isCollapsed = sidebar.classList.toggle('w-18');
            if (isCollapsed) {
                sidebar.classList.remove('w-64');
                localStorage.setItem('sidebar-collapsed', 'true');
            } else {
                sidebar.classList.add('w-64');
                localStorage.setItem('sidebar-collapsed', 'false');
            }
        }
        document.addEventListener('DOMContentLoaded', () => {
            if (localStorage.getItem('sidebar-collapsed') === 'true') {
                const sidebar = document.getElementById('sidebar');
                if (sidebar) {
                    sidebar.classList.remove('w-64');
                    sidebar.classList.add('w-18');
                }
            }
        });
    </script>
</body>
</html>
`))

var indexHTMLTemplate = template.Must(template.New("index_html").Funcs(template.FuncMap{
	"slice": func(s string, start, end int) string {
		if start < 0 {
			start = 0
		}
		if end > len(s) {
			end = len(s)
		}
		if start > end {
			return ""
		}
		return s[start:end]
	},
	"abbrev": abbrev,
}).Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Console</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
        #sidebar.w-18 { width: 4.5rem !important; }
        #sidebar.w-18 .sidebar-text { display: none !important; }
        #sidebar.w-18 .nav-item { justify-content: center !important; }
        #sidebar.w-18 .nav-item-icon { margin-right: 0 !important; }
        #sidebar.w-18 .logo-container { justify-content: center !important; }
        #sidebar.w-18 #toggle-button-collapsed { display: flex !important; }
        #sidebar.w-18 .footer-container { justify-content: center !important; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen">
    <div class="flex h-screen overflow-hidden">
        
        <!-- Sidebar Navigation (Desktop) -->
        <aside id="sidebar" class="hidden md:flex md:flex-col md:flex-shrink-0 w-64 bg-slate-900 text-slate-300 border-r border-slate-800 transition-all duration-300 relative">
            <div class="logo-container flex items-center justify-between h-16 px-4 bg-slate-950 border-b border-slate-800/50">
                <div class="flex items-center space-x-3 overflow-hidden">
                    <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        A
                    </div>
                    <span class="sidebar-text text-white font-bold text-lg tracking-tight whitespace-nowrap">Admin Console</span>
                </div>
                <button onclick="toggleSidebar()" class="sidebar-text p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
                    </svg>
                </button>
                <button id="toggle-button-collapsed" onclick="toggleSidebar()" class="hidden p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center mx-auto">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>
            <div class="flex-1 flex flex-col overflow-y-auto px-3 py-6 space-y-1.5">
                <span class="sidebar-text px-3 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 whitespace-nowrap">Modul Data</span>
                {{"{{"}}range .NavItems{{"}}"}}
                <a href="/{{"{{"}}.TableName{{"}}"}}" class="nav-item flex items-center px-3 py-2.5 text-sm font-medium rounded-xl transition-all group text-slate-400 hover:bg-slate-800 hover:text-slate-200">
                    <span class="nav-item-icon flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold mr-3 transition-all bg-slate-800 text-slate-400 group-hover:bg-slate-700 group-hover:text-slate-200">
                        {{"{{"}}.Abbrev{{"}}"}}
                    </span>
                    <span class="sidebar-text whitespace-nowrap">{{"{{"}}.Name{{"}}"}}</span>
                </a>
                {{"{{"}}end{{"}}"}}
            </div>
            <div class="footer-container p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center space-x-2 overflow-hidden">
                <span class="flex-shrink-0 w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                <span class="sidebar-text whitespace-nowrap">Sistem Aktif (Port: 8000)</span>
            </div>
        </aside>

        <!-- Content Area -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            
            <!-- Mobile Header -->
            <header class="md:hidden bg-white border-b border-slate-200 h-16 px-4 flex items-center justify-between sticky top-0 z-30 shadow-sm">
                <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center text-white font-bold text-base">
                        A
                    </div>
                    <span class="font-bold text-slate-900">Admin Console</span>
                </div>
                <div class="text-xs text-slate-500 flex items-center space-x-2">
                    <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
                    <span>Aktif</span>
                </div>
            </header>

            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-12 max-w-7xl">
                <!-- Mobile Navigation Tabs -->
                <div class="flex md:hidden overflow-x-auto py-2 mb-8 border-b border-slate-200 space-x-2 scrollbar-none">
                    {{"{{"}}range .NavItems{{"}}"}}
                    <a href="/{{"{{"}}.TableName{{"}}"}}" class="whitespace-nowrap px-4 py-1.5 text-xs font-semibold rounded-full bg-slate-100 text-slate-600 hover:bg-slate-200 transition-all">
                        {{"{{"}}.Name{{"}}"}}
                    </a>
                    {{"{{"}}end{{"}}"}}
                </div>

                <!-- Welcome Banner -->
                <div class="bg-gradient-to-r from-slate-900 to-indigo-950 rounded-3xl p-8 md:p-12 text-white shadow-xl mb-12 relative overflow-hidden">
                    <div class="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(99,102,241,0.15),transparent_50%)]"></div>
                    <div class="relative z-10 max-w-2xl">
                        <span class="px-3 py-1 text-xs font-semibold bg-indigo-500/20 text-indigo-300 rounded-full border border-indigo-500/30">Console Dashboard</span>
                        <h2 class="text-3xl md:text-4xl font-extrabold tracking-tight mt-4 mb-2">Selamat Datang di Admin Console</h2>
                        <p class="text-slate-400 text-sm md:text-base leading-relaxed">
                            Kelola seluruh data sistem Anda secara terintegrasi melalui satu halaman kontrol. Pilih modul di bawah atau melalui sidebar untuk memulai manajemen data.
                        </p>
                    </div>
                </div>

                <!-- Grid Modul -->
                <div class="space-y-6">
                    <h3 class="text-xl font-bold text-slate-900">Pilih Modul Data</h3>
                    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                        {{"{{"}}range .NavItems{{"}}"}}
                        <a href="/{{"{{"}}.TableName{{"}}"}}" class="group bg-white p-6 rounded-2xl border border-slate-200/80 shadow-sm hover:shadow-md hover:border-indigo-500/50 transition-all">
                            <div class="w-12 h-12 rounded-xl bg-indigo-50 flex items-center justify-center text-indigo-600 font-bold text-lg group-hover:bg-indigo-600 group-hover:text-white transition-all">
                                {{"{{"}} .Abbrev {{"}}"}}
                            </div>
                            <h4 class="text-lg font-bold text-slate-900 mt-4 group-hover:text-indigo-600 transition-all">{{"{{"}}.Name{{"}}"}}</h4>
                            <p class="text-xs text-slate-500 mt-1">Kelola data {{"{{"}}.Name{{"}}"}} (tambah, edit, detail, hapus).</p>
                        </a>
                        {{"{{"}}end{{"}}"}}
                    </div>
                </div>
            </main>
        </div>
    </div>
    <script>
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const isCollapsed = sidebar.classList.toggle('w-18');
            if (isCollapsed) {
                sidebar.classList.remove('w-64');
                localStorage.setItem('sidebar-collapsed', 'true');
            } else {
                sidebar.classList.add('w-64');
                localStorage.setItem('sidebar-collapsed', 'false');
            }
        }
        document.addEventListener('DOMContentLoaded', () => {
            if (localStorage.getItem('sidebar-collapsed') === 'true') {
                const sidebar = document.getElementById('sidebar');
                if (sidebar) {
                    sidebar.classList.remove('w-64');
                    sidebar.classList.add('w-18');
                }
            }
        });
    </script>
`))

var queryHTMLTemplate = template.Must(template.New("query_html").Funcs(template.FuncMap{
	"slice": func(s string, start, end int) string {
		if start < 0 {
			start = 0
		}
		if end > len(s) {
			end = len(s)
		}
		if start > end {
			return ""
		}
		return s[start:end]
	},
	"abbrev": abbrev,
}).Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Query Console</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
        #sidebar.w-18 { width: 4.5rem !important; }
        #sidebar.w-18 .sidebar-text { display: none !important; }
        #sidebar.w-18 .nav-item { justify-content: center !important; }
        #sidebar.w-18 .nav-item-icon { margin-right: 0 !important; }
        #sidebar.w-18 .logo-container { justify-content: center !important; }
        #sidebar.w-18 #toggle-button-collapsed { display: flex !important; }
        #sidebar.w-18 .footer-container { justify-content: center !important; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen">
    <div class="flex h-screen overflow-hidden">
        
        <!-- Sidebar Navigation (Desktop) -->
        <aside id="sidebar" class="hidden md:flex md:flex-col md:flex-shrink-0 w-64 bg-slate-900 text-slate-300 border-r border-slate-800 transition-all duration-300 relative">
            <div class="logo-container flex items-center justify-between h-16 px-4 bg-slate-950 border-b border-slate-800/50">
                <div class="flex items-center space-x-3 overflow-hidden">
                    <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        Q
                    </div>
                    <span class="sidebar-text text-white font-bold text-lg tracking-tight whitespace-nowrap">Admin Console</span>
                </div>
                <button onclick="toggleSidebar()" class="sidebar-text p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
                    </svg>
                </button>
                <button id="toggle-button-collapsed" onclick="toggleSidebar()" class="hidden p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 hover:text-white text-slate-400 transition-colors flex items-center justify-center mx-auto">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                    </svg>
                </button>
            </div>
            <div class="flex-1 flex flex-col overflow-y-auto px-3 py-6 space-y-1.5">
                <span class="sidebar-text px-3 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 whitespace-nowrap">Modul Data</span>
                {{"{{"}}range .NavItems{{"}}"}}
                <a href="/{{"{{"}}.TableName{{"}}"}}" class="nav-item flex items-center px-3 py-2.5 text-sm font-medium rounded-xl transition-all group {{"{{"}}if .Active{{"}}"}}bg-indigo-600 text-white shadow-md shadow-indigo-600/10{{"{{"}}else{{"}}"}}text-slate-400 hover:bg-slate-800 hover:text-slate-200{{"{{"}}end{{"}}"}}">
                    <span class="nav-item-icon flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold mr-3 transition-all {{"{{"}}if .Active{{"}}"}}bg-indigo-500 text-white{{"{{"}}else{{"}}"}}bg-slate-800 text-slate-400 group-hover:bg-slate-700 group-hover:text-slate-200{{"{{"}}end{{"}}"}}">
                        {{"{{"}}.Abbrev{{"}}"}}
                    </span>
                    <span class="sidebar-text whitespace-nowrap">{{"{{"}}.Name{{"}}"}}</span>
                </a>
                {{"{{"}}end{{"}}"}}
            </div>
            <div class="footer-container p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center space-x-2 overflow-hidden">
                <span class="flex-shrink-0 w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                <span class="sidebar-text whitespace-nowrap">Sistem Aktif (Port: 8000)</span>
            </div>
        </aside>

        <!-- Content Area -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            
            <!-- Mobile Header -->
            <header class="md:hidden bg-white border-b border-slate-200 h-16 px-4 flex items-center justify-between sticky top-0 z-30 shadow-sm">
                <div class="flex items-center space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-600 to-violet-500 flex items-center justify-center text-white font-bold text-base">
                        Q
                    </div>
                    <span class="font-bold text-slate-900">Query Console</span>
                </div>
                <div class="text-xs text-slate-500 flex items-center space-x-2">
                    <span class="w-2 h-2 bg-emerald-500 rounded-full animate-pulse"></span>
                    <span>Aktif</span>
                </div>
            </header>

            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-12 max-w-7xl">
                <!-- Mobile Navigation Tabs -->
                <div class="flex md:hidden overflow-x-auto py-2 mb-8 border-b border-slate-200 space-x-2 scrollbar-none">
                    {{"{{"}}range .NavItems{{"}}"}}
                    <a href="/{{"{{"}}.TableName{{"}}"}}" class="whitespace-nowrap px-4 py-1.5 text-xs font-semibold rounded-full transition-all {{"{{"}}if .Active{{"}}"}}bg-indigo-600 text-white{{"{{"}}else{{"}}"}}bg-slate-100 text-slate-600 hover:bg-slate-200{{"{{"}}end{{"}}"}}">
                        {{"{{"}}.Name{{"}}"}}
                    </a>
                    {{"{{"}}end{{"}}"}}
                </div>

                <!-- Header Section -->
                <div class="mb-6">
                    <h1 class="text-2xl font-bold text-slate-900 font-bold">Query Console</h1>
                    <p class="text-sm text-slate-500 mt-0.5">Jalankan perintah SQL kustom langsung ke database sistem.</p>
                </div>

                <!-- Query Form -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 mb-6">
                    <form method="POST" action="/query" class="space-y-4">
                        <div>
                            <label class="block text-sm font-semibold text-slate-700 mb-1.5">SQL Query</label>
                            <textarea 
                                name="sql" 
                                rows="6" 
                                placeholder="SELECT * FROM user LIMIT 10;" 
                                class="w-full px-4 py-3 font-mono text-sm border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all bg-slate-50"
                            >{{"{{"}} .Sql {{"}}"}}</textarea>
                        </div>
                        <div class="flex items-center justify-end">
                            <button type="submit" class="px-5 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl shadow-md shadow-indigo-600/10 transition-all flex items-center gap-2">
                                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                Jalankan Query
                            </button>
                        </div>
                    </form>
                </div>

                <!-- Results Section -->
                {{"{{"}} if .Error {{"}}"}}
                <div class="bg-rose-50 border border-rose-200 text-rose-800 rounded-2xl p-5 mb-6 flex items-start gap-3">
                    <svg class="h-5 w-5 text-rose-500 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <div>
                        <h4 class="font-bold text-rose-950 text-sm">Query Error</h4>
                        <p class="text-xs font-mono mt-1 break-all">{{"{{"}} .Error {{"}}"}}</p>
                    </div>
                </div>
                {{"{{"}} else if .Columns {{"}}"}}
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
                    <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
                        <h2 class="font-bold text-slate-900 font-bold">Hasil Query ({{"{{"}} len .Rows {{"}}"}} baris)</h2>
                    </div>
                    <div class="overflow-x-auto">
                        <table class="w-full text-left border-collapse text-sm min-w-[800px]">
                            <thead>
                                <tr class="bg-slate-50 border-b border-slate-100">
                                    {{"{{"}} range .Columns {{"}}"}}
                                    <th class="px-6 py-3.5 text-xs font-bold text-slate-500 uppercase tracking-wider">{{"{{"}} . {{"}}"}}</th>
                                    {{"{{"}} end {{"}}"}}
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-slate-100">
                                {{"{{"}} range $row := .Rows {{"}}"}}
                                <tr class="hover:bg-slate-50/50 transition-colors">
                                    {{"{{"}} range $col := $.Columns {{"}}"}}
                                    <td class="px-6 py-4 text-slate-700 font-medium whitespace-nowrap overflow-hidden max-w-[300px] text-ellipsis">
                                        {{"{{"}} index $row $col {{"}}"}}
                                    </td>
                                    {{"{{"}} end {{"}}"}}
                                </tr>
                                {{"{{"}} end {{"}}"}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{"{{"}} else if .Sql {{"}}"}}
                <div class="bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-2xl p-5 mb-6 flex items-start gap-3">
                    <svg class="h-5 w-5 text-emerald-500 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <div>
                        <h4 class="font-bold text-emerald-950 text-sm">Query Sukses</h4>
                        <p class="text-xs mt-1">Perintah SQL telah berhasil dieksekusi tanpa hasil data baris (misal INSERT/UPDATE/DELETE).</p>
                    </div>
                </div>
                {{"{{"}} end {{"}}"}}
            </main>
        </div>
    </div>
    <script>
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const isCollapsed = sidebar.classList.toggle('w-18');
            if (isCollapsed) {
                sidebar.classList.remove('w-64');
                localStorage.setItem('sidebar-collapsed', 'true');
            } else {
                sidebar.classList.add('w-64');
                localStorage.setItem('sidebar-collapsed', 'false');
            }
        }
        document.addEventListener('DOMContentLoaded', () => {
            if (localStorage.getItem('sidebar-collapsed') === 'true') {
                const sidebar = document.getElementById('sidebar');
                if (sidebar) {
                    sidebar.classList.remove('w-64');
                    sidebar.classList.add('w-18');
                }
            }
        });
    </script>
</body>
</html>
`))

// ============================================================
// Controller Templates
// ============================================================

var ctrlNewTemplate = template.Must(template.New("ctrl_new").Parse(`package {{.TableName}}

import (
	"{{.ModuleName}}/api/{{.TableName}}"
)

type ControllerV1 struct{}

func NewV1() {{.TableName}}.I{{.StructName}}V1 {
	return &ControllerV1{}
}
`))

var ctrlFormTemplate = template.Must(template.New("ctrl_form").Parse(`package {{.TableName}}

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"

	"{{.ModuleName}}/internal/service"
)

// ShowCreateForm menampilkan form tambah data baru.
func ShowCreateForm(r *ghttp.Request) {
	r.Response.WriteTpl("{{.TableName}}/form.html", nil)
	r.Exit()
}

// ShowEditForm menampilkan form edit data berdasarkan ID.
func ShowEditForm(r *ghttp.Request) {
	id := r.GetRouter("id").Uint64()
	if id == 0 {
		r.Response.RedirectTo("/{{.TableName}}")
		return
	}
	data, err := service.{{.StructName}}().Get(r.Context(), id)
	if err != nil || data == nil {
		r.Response.RedirectTo("/{{.TableName}}")
		return
	}
	r.Response.WriteTpl("{{.TableName}}/form.html", gconv.Map(data))
	r.Exit()
}

// DeleteAction menghapus data berdasarkan ID dari link GET.
func DeleteAction(r *ghttp.Request) {
	id := r.GetRouter("id").Uint64()
	if id != 0 {
		_ = service.{{.StructName}}().Delete(r.Context(), id)
	}
	r.Response.RedirectTo("/{{.TableName}}")
    r.Exit()
}
`))

var ctrlListTemplate = template.Must(template.New("ctrl_list").Parse(`package {{.TableName}}

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/service"
)

func (c *ControllerV1) List{{.ShortName}}(ctx context.Context, req *v1.List{{.ShortName}}Req) (res *v1.List{{.ShortName}}Res, err error) {
	list, total, err := service.{{.StructName}}().List(ctx, req.Page, req.PageSize, req.Keyword)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	r.Response.WriteTpl("{{.TableName}}/list.html", g.Map{
		"List":     list,
		"Total":    total,
		"Page":     req.Page,
		"PageSize": req.PageSize,
		"Keyword":  req.Keyword,
	})
	r.Exit()
	return nil, nil
}
`))

var ctrlGetTemplate = template.Must(template.New("ctrl_get").Parse(`package {{.TableName}}

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/service"
)

func (c *ControllerV1) Get{{.ShortName}}(ctx context.Context, req *v1.Get{{.ShortName}}Req) (res *v1.Get{{.ShortName}}Res, err error) {
	data, err := service.{{.StructName}}().Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	r.Response.WriteTpl("{{.TableName}}/detail.html", g.Map{
		{{- range .Fields}}
		"{{.Name}}": data.{{.Name}},
		{{- end}}
	})
	r.Exit()
	return nil, nil
}
`))

var ctrlCreateTemplate = template.Must(template.New("ctrl_create").Parse(`package {{.TableName}}

import (
	"context"
{{- if .HasUpload}}
	"io"
{{- end}}

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/model/do"
	"{{.ModuleName}}/internal/service"
)

func (c *ControllerV1) Create{{.ShortName}}(ctx context.Context, req *v1.Create{{.ShortName}}Req) (res *v1.Create{{.ShortName}}Res, err error) {
	createData := do.{{.StructName}}{
		{{- range .FormFields}}
		{{- if ne .HTMLType "file"}}
		{{.Name}}: req.{{.Name}},
		{{- end}}
		{{- end}}
	}

	{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	if req.{{.Name}} != nil {
		if f, errOpen := req.{{.Name}}.Open(); errOpen == nil {
			b, _ := io.ReadAll(f)
			_ = f.Close()
			createData.{{.Name}} = b
		}
	}
	{{- end}}
	{{- end}}

	_, err = service.{{.StructName}}().Create(ctx, createData)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	r.Response.RedirectTo("/{{.TableName}}")
	r.Exit()
	return nil, nil
}
`))

var ctrlUpdateTemplate = template.Must(template.New("ctrl_update").Parse(`package {{.TableName}}

import (
	"context"
{{- if .HasUpload}}
	"io"
{{- end}}

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/model/do"
	"{{.ModuleName}}/internal/service"
)

func (c *ControllerV1) Update{{.ShortName}}(ctx context.Context, req *v1.Update{{.ShortName}}Req) (res *v1.Update{{.ShortName}}Res, err error) {
	updateData := do.{{.StructName}}{
		{{- range .FormFields}}
		{{- if ne .HTMLType "file"}}
		{{.Name}}: req.{{.Name}},
		{{- end}}
		{{- end}}
	}

	{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	if req.{{.Name}} != nil {
		if f, errOpen := req.{{.Name}}.Open(); errOpen == nil {
			b, _ := io.ReadAll(f)
			_ = f.Close()
			updateData.{{.Name}} = b
		}
	}
	{{- end}}
	{{- end}}

	err = service.{{.StructName}}().Update(ctx, req.Id, updateData)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	r.Response.RedirectTo("/{{.TableName}}")
	r.Exit()
	return nil, nil
}
`))

var ctrlDeleteTemplate = template.Must(template.New("ctrl_delete").Parse(`package {{.TableName}}

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/service"
)

func (c *ControllerV1) Delete{{.ShortName}}(ctx context.Context, req *v1.Delete{{.ShortName}}Req) (res *v1.Delete{{.ShortName}}Res, err error) {
	err = service.{{.StructName}}().Delete(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	r.Response.RedirectTo("/{{.TableName}}")
	r.Exit()
	return nil, nil
}
`))

// ============================================================
// Cmd/Routing Template
// ============================================================

var cmdTemplate = template.Must(template.New("cmd").Funcs(template.FuncMap{
	"abbrev": abbrev,
}).Parse(`package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

{{range .Controllers}}
	"{{$.ModuleName}}/internal/controller/{{.PackageName}}"
{{- end}}
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			s := g.Server()
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
{{- range .Controllers}}
{{- if not .HasHTML}}
					{{.PackageName}}.NewV1(),
{{- end}}
{{- end}}
				)
			})

			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Bind(
{{- range .Controllers}}
{{- if .HasHTML}}
					{{.PackageName}}.NewV1(),
{{- end}}
{{- end}}
				)
			})

			s.Group("/", func(group *ghttp.RouterGroup) {
				group.GET("/", func(r *ghttp.Request) {
					r.Response.WriteTpl("index.html", g.Map{
						"NavItems": g.Slice{
{{- range .Controllers}}
{{- if .HasHTML}}
							g.Map{"Name": "{{.ShortName}}", "TableName": "{{.TableName}}", "Active": false, "Abbrev": "{{abbrev .ShortName}}"},
{{- end}}
{{- end}}
							g.Map{"Name": "Query Console", "TableName": "query", "Active": false, "Abbrev": "QC"},
						},
					})
					r.Exit()
				})
			})

			s.Group("/", func(group *ghttp.RouterGroup) {
				group.GET("/query", func(r *ghttp.Request) {
					r.Response.WriteTpl("query.html", g.Map{
						"NavItems": g.Slice{
{{- range .Controllers}}
{{- if .HasHTML}}
							g.Map{"Name": "{{.ShortName}}", "TableName": "{{.TableName}}", "Active": false, "Abbrev": "{{abbrev .ShortName}}"},
{{- end}}
{{- end}}
							g.Map{"Name": "Query Console", "TableName": "query", "Active": true, "Abbrev": "QC"},
						},
					})
					r.Exit()
				})
				group.POST("/query", func(r *ghttp.Request) {
					sqlStr := r.Get("sql").String()
					var (
						columns []string
						rows    []g.Map
						errStr  string
					)
					if sqlStr != "" {
						res, err := g.DB().GetAll(r.Context(), sqlStr)
						if err != nil {
							errStr = err.Error()
						} else if len(res) > 0 {
							for k := range res[0] {
								columns = append(columns, k)
							}
							for _, rMap := range res {
								rows = append(rows, rMap.Map())
							}
						}
					}
					r.Response.WriteTpl("query.html", g.Map{
						"Sql":     sqlStr,
						"Columns": columns,
						"Rows":    rows,
						"Error":   errStr,
						"NavItems": g.Slice{
{{- range .Controllers}}
{{- if .HasHTML}}
							g.Map{"Name": "{{.ShortName}}", "TableName": "{{.TableName}}", "Active": false, "Abbrev": "{{abbrev .ShortName}}"},
{{- end}}
{{- end}}
							g.Map{"Name": "Query Console", "TableName": "query", "Active": true, "Abbrev": "QC"},
						},
					})
					r.Exit()
				})
			})

{{range .Controllers}}
{{- if .HasHTML}}
			s.Group("/{{.TableName}}", func(group *ghttp.RouterGroup) {
				group.GET("/create", {{.PackageName}}.ShowCreateForm)
				group.GET("/{id}/edit", {{.PackageName}}.ShowEditForm)
				group.GET("/{id}/delete", {{.PackageName}}.DeleteAction)
			})
{{- end}}
{{- end}}

			s.Run()
			return nil
		},
	}
)
`))

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
	flag.Parse()

	// Determine project root (satu level di atas folder hack/)
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
				// Inject driver import right after "import ("
				reImport := regexp.MustCompile(`import\s*\(`)
				if reImport.MatchString(mainStr) {
					mainStr = reImport.ReplaceAllString(mainStr, "import (\n\t"+driverImport)
					_ = os.WriteFile(mainPath, []byte(mainStr), 0644)
					fmt.Printf("=== Automatically added %s driver import to main.go ===\n", driverType)

					// Run go get to download the package
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
gf gen dao

echo === Running gen_crud.go ===
gf-gen-crud --overwrite

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
		// Enrich field metadata from DB schema (ENUM, TEXT, date subtypes)
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
						// Fallback if DB offline
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

	for _, info := range allEntities {
		// Filter by --table flag
		if *tableFlag != "" && info.TableName != *tableFlag {
			continue
		}

		// Build NavItems
		var navItems []NavItem
		for _, ent := range allEntities {
			navItems = append(navItems, NavItem{
				Name:      ent.ShortName,
				TableName: ent.TableName,
				Active:    ent.TableName == info.TableName,
			})
		}
		navItems = append(navItems, NavItem{
			Name:      "Query Console",
			TableName: "query",
			Active:    info.TableName == "query",
		})
		info.NavItems = navItems

		fmt.Printf("=== Generating: %s (%s) ===\n", info.StructName, info.TableName)

		// 1. API Request/Response
		apiContent, err := renderTemplate(apiTemplate, info)
		if err != nil {
			fmt.Printf("  [ERROR] api template: %v\n", err)
		} else {
			apiPath := filepath.Join(root, "api", info.TableName, "v1", info.TableName+".go")
			_ = writeFile(apiPath, apiContent, *overwrite)
		}

		// 1.2. API Interface
		apiInterfaceContent, err := renderTemplate(apiInterfaceTemplate, info)
		if err != nil {
			fmt.Printf("  [ERROR] api interface template: %v\n", err)
		} else {
			apiInterfacePath := filepath.Join(root, "api", info.TableName, info.TableName+".go")
			_ = writeFile(apiInterfacePath, apiInterfaceContent, *overwrite)
		}

		// 2. Logic
		if !*skipLogic {
			logicContent, err := renderTemplate(logicTemplate, info)
			if err != nil {
				fmt.Printf("  [ERROR] logic template: %v\n", err)
			} else {
				logicPath := filepath.Join(root, "internal", "logic", info.VarName, info.VarName+".go")
				_ = writeFile(logicPath, logicContent, *overwrite)
			}
		}

		// 3. HTML Views
		if !*skipView {
			tplDir := filepath.Join(root, "resource", "template", info.TableName)

			listContent, err := renderTemplate(listHTMLTemplate, info)
			if err != nil {
				fmt.Printf("  [ERROR] list view: %v\n", err)
			} else {
				_ = writeFile(filepath.Join(tplDir, "list.html"), listContent, *overwrite)
			}

			formContent, err := renderTemplate(formHTMLTemplate, info)
			if err != nil {
				fmt.Printf("  [ERROR] form view: %v\n", err)
			} else {
				_ = writeFile(filepath.Join(tplDir, "form.html"), formContent, *overwrite)
			}

			detailContent, err := renderTemplate(detailHTMLTemplate, info)
			if err != nil {
				fmt.Printf("  [ERROR] detail view: %v\n", err)
			} else {
				_ = writeFile(filepath.Join(tplDir, "detail.html"), detailContent, *overwrite)
			}
		}

		// 4. Controller
		ctrlDir := filepath.Join(root, "internal", "controller", info.TableName)
		_ = os.MkdirAll(ctrlDir, 0755)

		// 4.1. Controller New
		ctrlNewContent, _ := renderTemplate(ctrlNewTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, info.TableName+"_new.go"), ctrlNewContent, *overwrite)

		// 4.2. Controller Form
		ctrlFormContent, _ := renderTemplate(ctrlFormTemplate, info)
		_ = writeFile(filepath.Join(ctrlDir, info.TableName+"_form.go"), ctrlFormContent, *overwrite)

		// 4.3. Controller Actions (List, Get, Create, Update, Delete)
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

	// 5. Rebuild internal/cmd/cmd.go with all controllers
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

		// Find shortname from allEntities if possible
		short := snakeToTitle(pkgName)
		for _, ent := range allEntities {
			if ent.TableName == pkgName {
				short = ent.ShortName
				break
			}
		}

		// Check if it has views to declare HTML routes
		hasHTML := false
		tplPath := filepath.Join(root, "resource", "template", pkgName)
		if _, err := os.Stat(tplPath); err == nil {
			hasHTML = true
		}
		controllers = append(controllers, CmdControllerInfo{
			PackageName: pkgName,
			TableName:   pkgName,
			HasHTML:     hasHTML,
			ShortName:   short,
		})
	}

	// 4.9. Generate Root Index & Query Console HTML
	if !*skipView {
		fmt.Println("=== Generating index.html & query.html ===")
		var indexNavItems []NavItem
		for _, ent := range allEntities {
			indexNavItems = append(indexNavItems, NavItem{
				Name:      ent.ShortName,
				TableName: ent.TableName,
				Active:    false,
			})
		}
		indexNavItems = append(indexNavItems, NavItem{
			Name:      "Query Console",
			TableName: "query",
			Active:    false,
		})

		tplDir := filepath.Join(root, "resource", "template")

		indexContent, err := renderTemplate(indexHTMLTemplate, map[string]interface{}{
			"NavItems": indexNavItems,
		})
		if err == nil {
			_ = writeFile(filepath.Join(tplDir, "index.html"), indexContent, true)
		} else {
			fmt.Printf("Gagal merender index.html: %v\n", err)
		}

		// Renders query.html
		queryNavItems := make([]NavItem, len(indexNavItems))
		copy(queryNavItems, indexNavItems)
		queryNavItems[len(queryNavItems)-1].Active = true // Make Query Console active

		queryContent, err := renderTemplate(queryHTMLTemplate, map[string]interface{}{
			"NavItems": queryNavItems,
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
		_ = writeFile(cmdPath, cmdContent, true) // Selalu overwrite cmd.go agar terupdate otomatis
	}

	fmt.Println("===================================")
	fmt.Println("Selesai! Langkah berikutnya:")
	fmt.Println("  1. gf gen service   → update service interfaces")
	fmt.Println("  2. go build ./...   → pastikan kompilasi sukses")
	fmt.Println("===================================")
}
