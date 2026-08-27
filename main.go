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
	Name       string   // Go field name, e.g. "NoHp"
	Type       string   // Go type, e.g. "string", "int64", "*gtime.Time"
	JsonTag    string   // json tag value, e.g. "no_hp"
	OrmTag     string   // orm tag value, e.g. "no_hp"
	IsSkip     bool     // true = exclude from create/update forms
	IsAudit    bool     // true = auto-managed (CreatedAt, UpdatedAt, etc.)
	HTMLType   string   // HTML input type: "text", "number", "date", "datetime-local", "time", "checkbox"
	IsTextarea bool     // true = render as <textarea> (TEXT/LONGTEXT/JSON columns)
	EnumValues []string // non-empty = render as <select> with these options
	IsJson     bool     // true = JSON column
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

// ============================================================
// Helpers
// ============================================================

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
	HTMLType   string
	IsTextarea bool
	EnumValues []string
	IsJson     bool
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
		case strings.Contains(colTypeLow, "blob"):
			meta.HTMLType = "file"
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

var apiTemplate = template.Must(template.New("api").Parse(`package v1

import (
	"github.com/gogf/gf/v2/frame/g"
{{- if .HasGtime}}
	"github.com/gogf/gf/v2/os/gtime"
{{- end}}
)

// ---------- List ----------

type List{{.ShortName}}Req struct {
	g.Meta   ` + "`" + `path:"/{{.TableName}}" method:"get" tags:"{{.StructName}}" summary:"Daftar {{.ShortName}}"` + "`" + `
	Page     int ` + "`" + `json:"page" d:"1"` + "`" + `
	PageSize int ` + "`" + `json:"page_size" d:"10"` + "`" + `
	EditId   uint64 ` + "`" + `json:"edit_id"` + "`" + `
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

// List mengambil daftar {{.ShortName}} dengan pagination.
func (s *s{{.StructName}}) List(ctx context.Context, page, pageSize int) (list []*entity.{{.StructName}}, total int, err error) {
	m := dao.{{.StructName}}.Ctx(ctx)
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
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen">
    <div class="flex h-screen overflow-hidden">
        
        <!-- Sidebar Navigation (Desktop) -->
        <aside class="hidden md:flex md:flex-shrink-0">
            <div class="flex flex-col w-64 bg-slate-900 text-slate-300 border-r border-slate-800">
                <div class="flex items-center h-16 px-6 bg-slate-950 space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        {{slice .ShortName 0 1}}
                    </div>
                    <span class="text-white font-bold text-lg tracking-tight">Admin Console</span>
                </div>
                <div class="flex-1 flex flex-col overflow-y-auto px-4 py-6 space-y-1.5">
                    <span class="px-2 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Modul Data</span>
                    {{- range .NavItems}}
                    <a href="/{{.TableName}}" class="flex items-center px-4 py-2.5 text-sm font-medium rounded-xl transition-all {{if .Active}}bg-indigo-600 text-white shadow-md shadow-indigo-600/10{{else}}text-slate-400 hover:bg-slate-800 hover:text-slate-200{{end}}">
                        <span class="w-2 h-2 rounded-full mr-3 {{if .Active}}bg-white{{else}}bg-slate-500{{end}}"></span>
                        {{.Name}}
                    </a>
                    {{- end}}
                </div>
                <div class="p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center space-x-2">
                    <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                    <span>Sistem Aktif (Port: 8000)</span>
                </div>
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

                <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    <!-- Left Side: Form -->
                    <div class="lg:col-span-1">
                        <div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-6 sticky top-8">
                            <h2 class="text-xl font-bold text-slate-900 mb-2">
                                {{"{{"}} if .Edit {{"}}"}}Edit {{.ShortName}}{{"{{"}} else {{"}}"}}Tambah {{.ShortName}}{{"{{"}} end {{"}}"}}
                            </h2>
                            <p class="text-sm text-slate-500 mb-6">
                                {{"{{"}} if .Edit {{"}}"}}Perbarui informasi data yang sudah ada.{{"{{"}} else {{"}}"}}Masukkan informasi untuk membuat data baru.{{"{{"}} end {{"}}"}}
                            </p>

                            <form method="POST" action="/{{.TableName}}{{"{{"}} if .Edit {{"}}"}}/{{"{{"}} .Edit.Id {{"}}"}}{{"{{"}} end {{"}}"}}" class="space-y-4"{{if .HasUpload}} enctype="multipart/form-data"{{end}}>
                                {{- range .FormFields}}
                                <div>
                                    <label class="block text-sm font-semibold text-slate-700 mb-1.5">{{.Name}}</label>
                                    {{- if .EnumValues}}
                                    {{- $fieldName := .Name}}
                                    <select name="{{.JsonTag}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white">
                                        <option value="">-- Pilih {{.Name}} --</option>
                                        {{- range .EnumValues}}
                                        <option value="{{.}}" {{"{{"}} if $.Edit {{"}}"}}{{"{{"}} if eq (printf "%v" $.Edit.{{$fieldName}}) "{{.}}" {{"}}"}}selected{{"{{"}} end {{"}}"}}{{"{{"}} end {{"}}"}}>{{.}}</option>
                                        {{- end}}
                                    </select>
                                    {{- else if .IsTextarea}}
                                    <textarea name="{{.JsonTag}}" rows="4" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm placeholder-slate-400 resize-y" placeholder="{{if .IsJson}}Masukkan {{.Name}} (Format JSON, contoh: {}){{else}}Masukkan {{.Name}}...{{end}}">{{"{{"}} if $.Edit {{"}}"}}{{"{{"}} $.Edit.{{.Name}} {{"}}"}}{{"{{"}} else {{"}}"}}{{if .IsJson}}{}{{end}}{{"{{"}} end {{"}}"}}</textarea>
                                    {{- else if eq .HTMLType "file"}}
                                    <input type="file" name="{{.JsonTag}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm file:mr-4 file:py-1.5 file:px-3 file:rounded-xl file:border-0 file:text-xs file:font-semibold file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100" />
                                    {{- else if eq .HTMLType "checkbox"}}
                                    <label class="flex items-center gap-2 cursor-pointer">
                                        <input type="checkbox" name="{{.JsonTag}}" value="1" {{"{{"}} if $.Edit {{"}}"}}{{"{{"}} if $.Edit.{{.Name}} {{"}}"}}checked{{"{{"}} end {{"}}"}}{{"{{"}} end {{"}}"}} class="w-4 h-4 accent-indigo-600" />
                                        <span class="text-sm text-slate-600">Aktif</span>
                                    </label>
                                    {{- else}}
                                    <input 
                                        type="{{.HTMLType}}" 
                                        name="{{.JsonTag}}" 
                                        value="{{"{{"}} if $.Edit {{"}}"}}{{"{{"}} $.Edit.{{.Name}} {{"}}"}}{{"{{"}} end {{"}}"}}" 
                                        class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm placeholder-slate-400" 
                                        placeholder="Masukkan {{.Name}}..."
                                    />
                                    {{- end}}
                                </div>
                                {{- end}}
                                
                                <div class="pt-2 flex items-center space-x-3">
                                    <button type="submit" class="flex-1 px-4 py-2.5 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-700 hover:to-violet-700 text-white font-medium text-sm rounded-xl shadow-md shadow-indigo-100 transition-all">
                                        Simpan
                                    </button>
                                    {{"{{"}} if .Edit {{"}}"}}
                                    <a href="/{{.TableName}}" class="px-4 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-medium text-sm rounded-xl transition-all text-center">
                                        Batal
                                    </a>
                                    {{"{{"}} end {{"}}"}}
                                </div>
                            </form>
                        </div>
                    </div>

                    <!-- Right Side: Data List -->
                    <div class="lg:col-span-2 space-y-6">
                        <div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm overflow-hidden">
                            <div class="p-6 border-b border-slate-100 flex items-center justify-between">
                                <div>
                                    <h2 class="text-xl font-bold text-slate-900">Daftar {{.ShortName}}</h2>
                                    <p class="text-sm text-slate-500 mt-0.5">Total: {{"{{"}} .Total {{"}}"}} data ditemukan</p>
                                </div>
                            </div>

                            <div class="overflow-x-auto">
                                <table class="w-full text-left border-collapse">
                                    <thead>
                                        <tr class="bg-slate-50 border-b border-slate-100">
                                            {{- range .ListFields}}
                                            <th class="px-6 py-3.5 text-xs font-bold text-slate-500 uppercase tracking-wider">{{.Name}}</th>
                                            {{- end}}
                                            <th class="px-6 py-3.5 text-xs font-bold text-slate-500 uppercase tracking-wider text-right">Aksi</th>
                                        </tr>
                                    </thead>
                                    <tbody class="divide-y divide-slate-100">
                                        {{"{{"}}range .List{{"}}"}}
                                        <tr class="hover:bg-slate-50/50 transition-colors">
                                            {{- range .ListFields}}
                                            <td class="px-6 py-4 text-sm text-slate-700 font-medium">{{"{{"}} .{{.Name}} {{"}}"}}</td>
                                            {{- end}}
                                            <td class="px-6 py-4 text-sm text-slate-700 text-right space-x-3">
                                                <a href="/{{$.TableName}}?edit_id={{"{{"}} .Id {{"}}"}}" class="inline-flex items-center text-xs font-semibold text-indigo-600 hover:text-indigo-900 transition-colors">
                                                    Edit
                                                </a>
                                                <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}/delete" onclick="return confirm('Hapus data ini?')" class="inline-flex items-center text-xs font-semibold text-rose-600 hover:text-rose-900 transition-colors">
                                                    Hapus
                                                </a>
                                            </td>
                                        </tr>
                                        {{"{{"}}end{{"}}"}}
                                    </tbody>
                                </table>
                            </div>
                        </div>
                    </div>
                </div>
            </main>
        </div>
    </div>
`))

var formHTMLTemplate = template.Must(template.New("form_html").Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Form {{.ShortName}}</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 w-full max-w-md">
        <h1 class="text-xl font-bold text-slate-900 mb-2">{{"{{"}} if .Id {{"}}"}}Edit{{"{{"}} else {{"}}"}}Tambah{{"{{"}} end {{"}}"}} {{.ShortName}}</h1>
        <p class="text-sm text-slate-500 mb-6">Isi formulir berikut untuk menyimpan data.</p>
        <form method="POST" action="/{{.TableName}}{{"{{"}} if .Id {{"}}"}}/{{"{{"}} .Id {{"}}"}}{{"{{"}} end {{"}}"}}" class="space-y-4"{{if .HasUpload}} enctype="multipart/form-data"{{end}}>
            {{- range .FormFields}}
            <div>
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
                {{- else if eq .HTMLType "checkbox"}}
                <label class="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" name="{{.JsonTag}}" value="1" {{"{{"}} if .{{.Name}} {{"}}"}}checked{{"{{"}} end {{"}}"}} class="w-4 h-4 accent-indigo-600" />
                    <span class="text-sm text-slate-600">Aktif</span>
                </label>
                {{- else}}
                <input type="{{.HTMLType}}" name="{{.JsonTag}}" value="{{"{{"}} .{{.Name}} {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                {{- end}}
            </div>
            {{- end}}
            <div class="pt-2 flex items-center space-x-3">
                <button type="submit" class="flex-1 px-4 py-2.5 bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-700 hover:to-violet-700 text-white font-medium text-sm rounded-xl shadow-md transition-all">Simpan</button>
                <a href="/{{.TableName}}" class="px-4 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-medium text-sm rounded-xl text-center">Batal</a>
            </div>
        </form>
    </div>
</body>
</html>
`))

var detailHTMLTemplate = template.Must(template.New("detail_html").Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Detail {{.ShortName}}</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body { font-family: 'Inter', sans-serif; }
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 w-full max-w-md">
        <h1 class="text-xl font-bold text-slate-900 mb-4">Detail {{.ShortName}} #{{"{{"}} .Id {{"}}"}}</h1>
        <table class="w-full text-left border-collapse mb-6 text-sm">
            <tbody class="divide-y divide-slate-100">
                {{- range .Fields}}
                <tr>
                    <th class="py-2.5 font-semibold text-slate-500 w-1/3">{{.Name}}</th>
                    <td class="py-2.5 text-slate-700">{{"{{"}} .{{.Name}} {{"}}"}}</td>
                </tr>
                {{- end}}
            </tbody>
        </table>
        <div class="flex items-center space-x-3">
            <a href="/{{.TableName}}/{{"{{"}} .Id {{"}}"}}/edit" class="flex-1 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-medium text-sm rounded-xl text-center shadow-md transition-all">Edit</a>
            <a href="/{{.TableName}}" class="px-4 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-medium text-sm rounded-xl text-center transition-all">Kembali</a>
        </div>
    </div>
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
    </style>
</head>
<body class="bg-slate-50 text-slate-800 min-h-screen">
    <div class="flex h-screen overflow-hidden">
        
        <!-- Sidebar Navigation (Desktop) -->
        <aside class="hidden md:flex md:flex-shrink-0">
            <div class="flex flex-col w-64 bg-slate-900 text-slate-300 border-r border-slate-800">
                <div class="flex items-center h-16 px-6 bg-slate-950 space-x-3">
                    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        A
                    </div>
                    <span class="text-white font-bold text-lg tracking-tight">Admin Console</span>
                </div>
                <div class="flex-1 flex flex-col overflow-y-auto px-4 py-6 space-y-1.5">
                    <span class="px-2 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">Modul Data</span>
                    {{"{{"}}range .NavItems{{"}}"}}
                    <a href="/{{"{{"}}.TableName{{"}}"}}" class="flex items-center px-4 py-2.5 text-sm font-medium rounded-xl transition-all text-slate-400 hover:bg-slate-800 hover:text-slate-200">
                        <span class="w-2 h-2 rounded-full mr-3 bg-slate-500"></span>
                        {{"{{"}}.Name{{"}}"}}
                    </a>
                    {{"{{"}}end{{"}}"}}
                </div>
                <div class="p-4 border-t border-slate-800 text-xs text-slate-500 flex items-center space-x-2">
                    <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                    <span>Sistem Aktif (Port: 8000)</span>
                </div>
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
                                {{"{{"}} slice .Name 0 1 {{"}}"}}
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
	"fmt"

	"github.com/gogf/gf/v2/net/ghttp"

	"{{.ModuleName}}/internal/service"
)

// ShowCreateForm menampilkan form tambah data baru.
func ShowCreateForm(r *ghttp.Request) {
	r.Response.RedirectTo("/{{.TableName}}")
}

// ShowEditForm menampilkan form edit data berdasarkan ID.
func ShowEditForm(r *ghttp.Request) {
	id := r.GetRouter("id").Uint64()
	if id == 0 {
		r.Response.RedirectTo("/{{.TableName}}")
		return
	}
	r.Response.RedirectTo(fmt.Sprintf("/{{.TableName}}?edit_id=%d", id))
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
	"{{.ModuleName}}/internal/model/entity"
	"{{.ModuleName}}/internal/service"
)

func (c *ControllerV1) List{{.ShortName}}(ctx context.Context, req *v1.List{{.ShortName}}Req) (res *v1.List{{.ShortName}}Res, err error) {
	list, total, err := service.{{.StructName}}().List(ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}

	var editData *entity.{{.StructName}}
	if req.EditId != 0 {
		editData, _ = service.{{.StructName}}().Get(ctx, req.EditId)
	}

	r := ghttp.RequestFromCtx(ctx)
	r.Response.WriteTpl("{{.TableName}}/list.html", g.Map{
		"List":  list,
		"Total": total,
		"Page":  req.Page,
		"Edit":  editData,
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
	{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	var {{.JsonTag}}Bytes []byte
	if req.{{.Name}} != nil {
		if f, errOpen := req.{{.Name}}.Open(); errOpen == nil {
			{{.JsonTag}}Bytes, _ = io.ReadAll(f)
			_ = f.Close()
		}
	}
	{{- end}}
	{{- end}}

	_, err = service.{{.StructName}}().Create(ctx, do.{{.StructName}}{
		{{- range .FormFields}}
		{{- if eq .HTMLType "file"}}
		{{.Name}}: {{.JsonTag}}Bytes,
		{{- else}}
		{{.Name}}: req.{{.Name}},
		{{- end}}
		{{- end}}
	})
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

	"github.com/gogf/gf/v2/frame/g"
{{- end}}

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/model/do"
	"{{.ModuleName}}/internal/service"
)

func (c *ControllerV1) Update{{.ShortName}}(ctx context.Context, req *v1.Update{{.ShortName}}Req) (res *v1.Update{{.ShortName}}Res, err error) {
	{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	var {{.JsonTag}}Bytes []byte
	var has{{.Name}}Upload bool
	if req.{{.Name}} != nil {
		if f, errOpen := req.{{.Name}}.Open(); errOpen == nil {
			{{.JsonTag}}Bytes, _ = io.ReadAll(f)
			_ = f.Close()
			has{{.Name}}Upload = true
		}
	}
	{{- end}}
	{{- end}}

	err = service.{{.StructName}}().Update(ctx, req.Id, do.{{.StructName}}{
		{{- range .FormFields}}
		{{- if eq .HTMLType "file"}}
		{{.Name}}: g.Conditional(has{{.Name}}Upload, {{.JsonTag}}Bytes, nil), // Only update file if new one is uploaded
		{{- else}}
		{{.Name}}: req.{{.Name}},
		{{- end}}
		{{- end}}
	})
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

var cmdTemplate = template.Must(template.New("cmd").Parse(`package cmd

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
							g.Map{"Name": "{{.ShortName}}", "TableName": "{{.TableName}}", "Active": false},
{{- end}}
{{- end}}
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
						if meta.HTMLType == "file" {
							info.HasUpload = true
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

	// 4.9. Generate Root Index HTML
	if !*skipView {
		fmt.Println("=== Generating index.html ===")
		var indexNavItems []NavItem
		for _, ent := range allEntities {
			indexNavItems = append(indexNavItems, NavItem{
				Name:      ent.ShortName,
				TableName: ent.TableName,
				Active:    false,
			})
		}
		indexContent, err := renderTemplate(indexHTMLTemplate, map[string]interface{}{
			"NavItems": indexNavItems,
		})
		if err == nil {
			tplDir := filepath.Join(root, "resource", "template")
			_ = writeFile(filepath.Join(tplDir, "index.html"), indexContent, true)
		} else {
			fmt.Printf("Gagal merender index.html: %v\n", err)
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
