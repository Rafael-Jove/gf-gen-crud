package main

import (
	"strings"
	"text/template"
)

var modelTemplate = template.Must(template.New("model").Parse(`package model

import(
    "fmt"
    "{{.ModuleName}}/internal/model/entity"
)

// {{.ShortName}}ListInput adalah DTO input untuk pencarian, sorting, dan filter data {{.ShortName}}
type {{.ShortName}}ListInput struct {
	Page      int                    ` + "`" + `json:"page"` + "`" + `
	PageSize  int                    ` + "`" + `json:"page_size"` + "`" + `
	Keyword   string                 ` + "`" + `json:"keyword"` + "`" + `
	SortBy    string                 ` + "`" + `json:"sort_by"` + "`" + `
	SortOrder string                 ` + "`" + `json:"sort_order"` + "`" + `
	Filters   map[string]interface{} ` + "`" + `json:"filters"` + "`" + `
}

type {{.ShortName}}Output struct{
    {{- range .Fields}}
        {{.Name}} {{.Type}} ` + "`" + `json:"{{.JsonTag}}"` + "`" + `
    {{- end}}
        Label string ` + "`" + `json:"label"` + "`" + `
}

func To{{.ShortName}}Output(item *entity.{{.StructName}}) {{.ShortName}}Output {
    if item == nil {
        return {{.ShortName}}Output{}
    }
    return {{.ShortName}}Output{
        {{- range .Fields}}
            {{.Name}}: item.{{.Name}},
        {{- end}}
            Label: fmt.Sprintf("%v", item.{{.LabelFieldName}}),
    }
}
`))

var apiTemplate = template.Must(template.New("api").Funcs(template.FuncMap{
	"contains": strings.Contains,
}).Parse(`package v1

import (
	"encoding/json"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/frame/g"
{{- $hasFile := false}}
{{- range .FormFields}}
{{- if eq .HTMLType "file"}}{{$hasFile = true}}{{end}}
{{- end}}
{{- if $hasFile}}
	"github.com/gogf/gf/v2/net/ghttp"
{{- end}}
{{- $hasGtime := false}}
{{- range .FormFields}}
{{- if contains .Type "gtime"}}{{$hasGtime = true}}{{end}}
{{- end}}
{{- if $hasGtime}}
	"github.com/gogf/gf/v2/os/gtime"
{{- end}}
)

// ---------- Data Item Model (with correct JSON types) ----------

type {{.ShortName}}Field struct {
	Type       string                 ` + "`" + `json:"type"` + "`" + `
	IsPK       bool                   ` + "`" + `json:"is_pk,omitempty"` + "`" + `
	IsRequired bool                   ` + "`" + `json:"is_required"` + "`" + `
	Values     []string               ` + "`" + `json:"values,omitempty"` + "`" + `
	Extra      map[string]interface{} ` + "`" + `json:"meta,omitempty"` + "`" + `
}

func (f {{.ShortName}}Field) MarshalJSON() ([]byte, error) {
	result := map[string]interface{}{
		"type":        f.Type,
		"is_required": f.IsRequired,
	}
	if f.IsPK {
		result["is_pk"] = true
	}
	if len(f.Values) > 0 {
		result["values"] = f.Values
	}
	for k, v := range f.Extra {
		result[k] = v
	}
	return json.Marshal(result)
}

type {{.ShortName}}Header struct {
{{- range .Fields}}
	{{.Name}} {{$.ShortName}}Field ` + "`" + `json:"{{.JsonTag}}"` + "`" + `
{{- end}}
}

// ---------- List ----------

type List{{.ShortName}}Req struct {
	g.Meta    ` + "`" + `path:"/{{.TableName}}" method:"get" tags:"{{.StructName}}" summary:"Daftar {{.ShortName}}"` + "`" + `
	Page      int    ` + "`" + `json:"page" p:"page"` + "`" + `
	PageSize  int    ` + "`" + `json:"page_size" p:"page_size"` + "`" + `
	Keyword   string ` + "`" + `json:"keyword" p:"keyword"` + "`" + `
	SortBy    string ` + "`" + `json:"sort_by" p:"sort_by"` + "`" + `
	SortOrder string ` + "`" + `json:"sort_order" p:"sort_order"` + "`" + `
}

type List{{.ShortName}}Res struct {
	Header        {{.ShortName}}Header ` + "`" + `json:"header"` + "`" + `
	Value         []*gmap.ListMap      ` + "`" + `json:"value"` + "`" + `
	Values        []*gmap.ListMap      ` + "`" + `json:"values,omitempty"` + "`" + `
	Total         int                  ` + "`" + `json:"total"` + "`" + `
	Page          int                  ` + "`" + `json:"page"` + "`" + `
	PageSize      int                  ` + "`" + `json:"page_size"` + "`" + `
	JumlahHalaman int                  ` + "`" + `json:"jumlah_halaman"` + "`" + `
	TotalPage     int                  ` + "`" + `json:"total_page"` + "`" + `
}

// ---------- Filter ----------

type Filter{{.ShortName}}Req struct {
	g.Meta    ` + "`" + `path:"/{{.TableName}}/filter" method:"get" tags:"{{.StructName}}" summary:"Halaman Filter {{.ShortName}}"` + "`" + `
	Page      int    ` + "`" + `json:"page" p:"page"` + "`" + `
	PageSize  int    ` + "`" + `json:"page_size" p:"page_size"` + "`" + `
	Keyword   string ` + "`" + `json:"keyword" p:"keyword"` + "`" + `
	SortBy    string ` + "`" + `json:"sort_by" p:"sort_by"` + "`" + `
	SortOrder string ` + "`" + `json:"sort_order" p:"sort_order"` + "`" + `
}

type Filter{{.ShortName}}Res struct {
	Header        {{.ShortName}}Header ` + "`" + `json:"header"` + "`" + `
	Value         []*gmap.ListMap      ` + "`" + `json:"value"` + "`" + `
	Values        []*gmap.ListMap      ` + "`" + `json:"values,omitempty"` + "`" + `
	Total         int                  ` + "`" + `json:"total"` + "`" + `
	Page          int                  ` + "`" + `json:"page"` + "`" + `
	PageSize      int                  ` + "`" + `json:"page_size"` + "`" + `
	JumlahHalaman int                  ` + "`" + `json:"jumlah_halaman"` + "`" + `
	TotalPage     int                  ` + "`" + `json:"total_page"` + "`" + `
}

// ---------- Get ----------

type Get{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}" method:"get" tags:"{{.StructName}}" summary:"Detail {{.ShortName}}"` + "`" + `
	Id     {{if .PKType}}{{.PKType}}{{else}}uint64{{end}} ` + "`" + `json:"id" in:"path" v:"required#ID wajib diisi"` + "`" + `
}

type Get{{.ShortName}}Res struct {
	Header {{.ShortName}}Header ` + "`" + `json:"header"` + "`" + `
	Value  *gmap.ListMap        ` + "`" + `json:"value"` + "`" + `
	Values *gmap.ListMap        ` + "`" + `json:"values,omitempty"` + "`" + `
}

{{- if not .IsReadOnly}}
type Create{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}" method:"post" tags:"{{.StructName}}" summary:"Buat {{.ShortName}} baru"` + "`" + `
{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	{{.Name}} *ghttp.UploadFile ` + "`" + `json:"{{.JsonTag}}" type:"file"{{if .IsRequired}} v:"required#{{.Name}} wajib diisi"{{end}}` + "`" + `
	{{- else}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JsonTag}}"{{if .IsRequired}} v:"required#{{.Name}} wajib diisi"{{end}}` + "`" + `
	{{- end}}
{{- end}}
}

type Create{{.ShortName}}Res struct {
	Header {{.ShortName}}Header ` + "`" + `json:"header"` + "`" + `
	Value  *gmap.ListMap        ` + "`" + `json:"value"` + "`" + `
	Values *gmap.ListMap        ` + "`" + `json:"values,omitempty"` + "`" + `
}

// ---------- Update ----------

type Update{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}" method:"put" tags:"{{.StructName}}" summary:"Update {{.ShortName}}"` + "`" + `
	Id     {{if .PKType}}{{.PKType}}{{else}}uint64{{end}} ` + "`" + `json:"id" in:"path" v:"required#ID wajib diisi"` + "`" + `
{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	{{.Name}} *ghttp.UploadFile ` + "`" + `json:"{{.JsonTag}}" type:"file"` + "`" + `
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
	Id     {{if .PKType}}{{.PKType}}{{else}}uint64{{end}} ` + "`" + `json:"id" in:"path" v:"required#ID wajib diisi"` + "`" + `
}

type Delete{{.ShortName}}Res struct {
	Message string ` + "`" + `json:"message"` + "`" + `
}

// ---------- Batch Delete ----------

type BatchDelete{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/batch-delete" method:"post" tags:"{{.StructName}}" summary:"Hapus Massal {{.ShortName}}"` + "`" + `
	Ids    []{{if .PKType}}{{.PKType}}{{else}}uint64{{end}} ` + "`" + `json:"ids" p:"ids" v:"required#Pilih minimal satu data untuk dihapus"` + "`" + `
}

type BatchDelete{{.ShortName}}Res struct {
	Count int64 ` + "`" + `json:"count"` + "`" + `
}

// ---------- HTML Forms & Actions ----------

type ShowCreateFormReq struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/create" method:"get" tags:"{{.StructName}}" summary:"Form Tambah {{.ShortName}}"` + "`" + `
}
type ShowCreateFormRes struct{}

type ShowEditFormReq struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}/edit" method:"get" tags:"{{.StructName}}" summary:"Form Edit {{.ShortName}}"` + "`" + `
	Id     {{if .PKType}}{{.PKType}}{{else}}uint64{{end}} ` + "`" + `json:"id" in:"path" v:"required#ID wajib diisi"` + "`" + `
}
type ShowEditFormRes struct{}

type DeleteActionReq struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}/delete" method:"post" tags:"{{.StructName}}" summary:"Hapus {{.ShortName}} Action"` + "`" + `
	Id     {{if .PKType}}{{.PKType}}{{else}}uint64{{end}} ` + "`" + `json:"id" in:"path" v:"required#ID wajib diisi"` + "`" + `
}
type DeleteActionRes struct{}
{{- end}}
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
	{{- if not .IsReadOnly}}
	ShowCreateForm(ctx context.Context, req *v1.ShowCreateFormReq) (res *v1.ShowCreateFormRes, err error)
	ShowEditForm(ctx context.Context, req *v1.ShowEditFormReq) (res *v1.ShowEditFormRes, err error)
	DeleteAction(ctx context.Context, req *v1.DeleteActionReq) (res *v1.DeleteActionRes, err error)
	{{- end}}
	List{{.ShortName}}(ctx context.Context, req *v1.List{{.ShortName}}Req) (res *v1.List{{.ShortName}}Res, err error)
	Filter{{.ShortName}}(ctx context.Context, req *v1.Filter{{.ShortName}}Req) (res *v1.Filter{{.ShortName}}Res, err error)
	Get{{.ShortName}}(ctx context.Context, req *v1.Get{{.ShortName}}Req) (res *v1.Get{{.ShortName}}Res, err error)
	{{- if not .IsReadOnly}}
	Create{{.ShortName}}(ctx context.Context, req *v1.Create{{.ShortName}}Req) (res *v1.Create{{.ShortName}}Res, err error)
	Update{{.ShortName}}(ctx context.Context, req *v1.Update{{.ShortName}}Req) (res *v1.Update{{.ShortName}}Res, err error)
	Delete{{.ShortName}}(ctx context.Context, req *v1.Delete{{.ShortName}}Req) (res *v1.Delete{{.ShortName}}Res, err error)
	BatchDelete{{.ShortName}}(ctx context.Context, req *v1.BatchDelete{{.ShortName}}Req) (res *v1.BatchDelete{{.ShortName}}Res, err error)
	{{- end}}
}
`))

var logicTemplate = template.Must(template.New("logic").Parse(`package {{.VarName}}

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/util/gconv"
	"{{.ModuleName}}/internal/dao"
	"{{.ModuleName}}/internal/model"
	{{- if not .IsReadOnly}}
	"{{.ModuleName}}/internal/model/do"
	{{- end}}
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

// List mengambil daftar {{.ShortName}} dengan pagination, keyword filter, form filters, dan sorting.
func (s *s{{.StructName}}) List(ctx context.Context, in model.{{.ShortName}}ListInput) (list []*entity.{{.StructName}}, total int, err error) {
	m := dao.{{.StructName}}.Ctx(ctx)

	// Apply dynamic form filters from in.Filters
	if len(in.Filters) > 0 {
		{{- range .FormFields}}
		{{- if .EnumValues}}
		if val := gconv.String(in.Filters["f_{{.JsonTag}}"]); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + `", val)
		}
		{{- else if eq .HTMLType "checkbox"}}
		if val := gconv.String(in.Filters["f_{{.JsonTag}}"]); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + `", val)
		}
		{{- else if eq .HTMLType "number"}}
		if valMin := gconv.String(in.Filters["f_{{.JsonTag}}_min"]); valMin != "" {
			m = m.WhereGTE("` + "`" + `{{.OrmTag}}` + "`" + `", valMin)
		}
		if valMax := gconv.String(in.Filters["f_{{.JsonTag}}_max"]); valMax != "" {
			m = m.WhereLTE("` + "`" + `{{.OrmTag}}` + "`" + `", valMax)
		}
		{{- else if or (eq .HTMLType "date") (eq .HTMLType "datetime-local") (eq .HTMLType "time")}}
		if valFrom := gconv.String(in.Filters["f_{{.JsonTag}}_from"]); valFrom != "" {
			m = m.WhereGTE("` + "`" + `{{.OrmTag}}` + "`" + `", valFrom)
		}
		if valTo := gconv.String(in.Filters["f_{{.JsonTag}}_to"]); valTo != "" {
			m = m.WhereLTE("` + "`" + `{{.OrmTag}}` + "`" + `", valTo)
		}
		{{- else if eq .Type "string"}}
		if val := gconv.String(in.Filters["f_{{.JsonTag}}"]); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + ` LIKE ?", "%"+val+"%")
		}
		{{- else}}
		if val := gconv.String(in.Filters["f_{{.JsonTag}}"]); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + `", val)
		}
		{{- end}}
		{{- end}}
	}

	// Keyword search (hanya mencari pada kolom string/text agar kompatibel dengan PostgreSQL & MySQL)
	if in.Keyword != "" {
		var conds []string
		var args []interface{}
		{{- range .ListFields}}
		{{- if .IsSearchable}}
		conds = append(conds, "` + "`" + `{{.OrmTag}}` + "`" + ` LIKE ?")
		args = append(args, "%"+in.Keyword+"%")
		{{- end}}
		{{- end}}
		if len(conds) > 0 {
			m = m.Where(strings.Join(conds, " OR "), args...)
		}
	}

	// Sorting dengan whitelist kolom untuk mencegah SQL injection
	allowedSortCols := map[string]bool{
		{{- range .Fields}}
		"{{.OrmTag}}": true,
		{{- end}}
	}
	if in.SortBy != "" && allowedSortCols[in.SortBy] {
		order := "ASC"
		if strings.ToUpper(in.SortOrder) == "DESC" {
			order = "DESC"
		}
		m = m.Order(fmt.Sprintf("` + "`" + `%s` + "`" + ` %s", in.SortBy, order))
	} else {
		m = m.Order("` + "`" + `{{.PKOrmTag}}` + "`" + ` DESC")
	}

	total, err = m.Count()
	if err != nil {
		return
	}
	if in.Page > 0 && in.PageSize > 0 {
		err = m.Page(in.Page, in.PageSize).Scan(&list)
	} else if in.PageSize > 0 {
		err = m.Limit(in.PageSize).Scan(&list)
	} else {
		err = m.Scan(&list)
	}
	return
}

// Get mengambil satu {{.ShortName}} berdasarkan ID.
func (s *s{{.StructName}}) Get(ctx context.Context, id {{if .PKType}}{{.PKType}}{{else}}uint64{{end}}) (data *entity.{{.StructName}}, err error) {
	err = dao.{{.StructName}}.Ctx(ctx).WherePri(id).Scan(&data)
	return
}

{{- if not .IsReadOnly}}
// Create membuat {{.ShortName}} baru.
func (s *s{{.StructName}}) Create(ctx context.Context, in do.{{.StructName}}) (data *entity.{{.StructName}}, err error) {
	{{- if .IsPKAutoIncrement}}
	lastId, err := dao.{{.StructName}}.Ctx(ctx).Data(in).InsertAndGetId()
	if err != nil {
		return
	}
	return s.Get(ctx, {{if eq .PKType "uint64"}}uint64(lastId){{else if eq .PKType "int"}}int(lastId){{else}}lastId{{end}})
	{{- else}}
	_, err = dao.{{.StructName}}.Ctx(ctx).Data(in).Insert()
	if err != nil {
		return
	}
	// Ambil data yang baru dibuat berdasarkan primary key
	if pkVal := in.{{.PKName}}; pkVal != nil {
		return s.Get(ctx, gconv.{{if eq .PKType "string"}}String{{else if eq .PKType "int"}}Int{{else}}Uint64{{end}}(pkVal))
	}
	return nil, nil
	{{- end}}
}

// Update mengupdate {{.ShortName}} berdasarkan ID.
func (s *s{{.StructName}}) Update(ctx context.Context, id {{if .PKType}}{{.PKType}}{{else}}uint64{{end}}, in do.{{.StructName}}) (err error) {
	_, err = dao.{{.StructName}}.Ctx(ctx).WherePri(id).Data(in).Update()
	return
}

// Delete menghapus {{.ShortName}} berdasarkan ID.
func (s *s{{.StructName}}) Delete(ctx context.Context, id {{if .PKType}}{{.PKType}}{{else}}uint64{{end}}) (err error) {
	_, err = dao.{{.StructName}}.Ctx(ctx).WherePri(id).Delete()
	return
}

// BatchDelete menghapus banyak {{.ShortName}} sekaligus berdasarkan daftar ID.
func (s *s{{.StructName}}) BatchDelete(ctx context.Context, ids []{{if .PKType}}{{.PKType}}{{else}}uint64{{end}}) (count int64, err error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := dao.{{.StructName}}.Ctx(ctx).WhereIn(dao.{{.StructName}}.Columns().{{.PKName}}, ids).Delete()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
{{- end}}
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
        {{"{{"}}include "public/sidebar.html" .{{"}}"}}

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
                {{"{{"}}include "public/mobile_nav.html" .{{"}}"}}

                <!-- Header Section -->
                <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
                    <div>
                        <h1 class="text-2xl font-bold text-slate-900 font-bold">Daftar {{.ShortName}}</h1>
                        <p class="text-sm text-slate-500 mt-0.5">{{- if .IsReadOnly}}Lihat data {{.TableName}}.{{- else}}Kelola data {{.TableName}} dengan mudah.{{- end}}</p>
                    </div>
                    <div class="flex items-center gap-2.5 w-full sm:w-auto">
                        {{- if or (eq .FilterType "form") (eq .FilterType "both")}}
                        <a href="/{{.TableName}}/filter" class="inline-flex items-center justify-center px-4 py-2.5 bg-white border border-slate-200 hover:border-indigo-400 hover:bg-indigo-50 text-slate-700 hover:text-indigo-700 font-semibold text-sm rounded-xl transition-all shadow-sm gap-2 w-full sm:w-auto">
                            <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2a1 1 0 01-.293.707L13 13.414V19a1 1 0 01-.553.894l-4 2A1 1 0 017 21v-7.586L3.293 6.707A1 1 0 013 6V4z" />
                            </svg>
                            Filter
                        </a>
                        {{- end}}
                        {{- if not .IsReadOnly}}
                        <a href="/{{.TableName}}/create" class="inline-flex items-center justify-center px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl shadow-md shadow-indigo-600/10 transition-all gap-2 w-full sm:w-auto">
                            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke-width="2">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                            </svg>
                            Tambah {{.ShortName}}
                        </a>
                        {{- end}}
                    </div>
                </div>

                {{- if or (eq .FilterType "input") (eq .FilterType "both")}}
                <!-- Filter Keyword -->
                <div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-4 mb-4">
                    <form onsubmit="event.preventDefault(); performSearch(this.keyword.value);" class="flex flex-col sm:flex-row items-center gap-3">
                        <div class="relative flex-1 w-full">
                            <div class="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                                </svg>
                            </div>
                            <input 
                                type="text" 
                                name="keyword" 
                                placeholder="Cari di semua kolom..." 
                                class="w-full pl-10 pr-4 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm placeholder-slate-400"
                            />
                        </div>
                        <div class="flex items-center gap-2 w-full sm:w-auto">
                            <select name="page_size" onchange="performSearch(this.form.keyword.value)" class="px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white text-slate-700">
                                <option value="10" {{"{{"}} if eq .PageSize 10 {{"}}"}}selected{{"{{"}} end {{"}}"}}>10 per hal</option>
                                <option value="20" {{"{{"}} if eq .PageSize 20 {{"}}"}}selected{{"{{"}} end {{"}}"}}>20 per hal</option>
                                <option value="50" {{"{{"}} if eq .PageSize 50 {{"}}"}}selected{{"{{"}} end {{"}}"}}>50 per hal</option>
                                <option value="100" {{"{{"}} if eq .PageSize 100 {{"}}"}}selected{{"{{"}} end {{"}}"}}>100 per hal</option>
                            </select>
                            <button type="submit" class="flex-1 sm:flex-initial px-5 py-2.5 bg-slate-900 hover:bg-slate-800 text-white font-semibold text-sm rounded-xl transition-all">Cari</button>
                            <button type="button" onclick="const inp = this.form.keyword; inp.value = ''; performSearch('');" class="flex-1 sm:flex-initial px-5 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-sm rounded-xl transition-all text-center">Reset</button>
                        </div>
                    </form>
                </div>
                {{- end}}

                {{- if eq .FilterType "none"}}
                <!-- Page Size (no filter) -->
                <div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm p-4 mb-4">
                    <form method="GET" action="/{{.TableName}}" class="flex items-center gap-3">
                        <select name="page_size" onchange="this.form.submit()" class="px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white text-slate-700">
                            <option value="10" {{"{{"}} if eq .PageSize 10 {{"}}"}}selected{{"{{"}} end {{"}}"}}>10 per hal</option>
                            <option value="20" {{"{{"}} if eq .PageSize 20 {{"}}"}}selected{{"{{"}} end {{"}}"}}>20 per hal</option>
                            <option value="50" {{"{{"}} if eq .PageSize 50 {{"}}"}}selected{{"{{"}} end {{"}}"}}>50 per hal</option>
                            <option value="100" {{"{{"}} if eq .PageSize 100 {{"}}"}}selected{{"{{"}} end {{"}}"}}>100 per hal</option>
                        </select>
                    </form>
                </div>
                {{- end}}

                <!-- Table Card -->
                <div class="bg-white rounded-2xl border border-slate-200/80 shadow-sm overflow-hidden">
                    <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
                        <div class="flex items-center gap-3">
                            <h2 class="font-bold text-slate-900">Total: {{"{{"}} .Total {{"}}"}} data</h2>
                            {{- if not .IsReadOnly}}
                            <button id="btn-batch-delete" type="button" onclick="submitBatchDelete()" disabled class="opacity-0 pointer-events-none transition-all duration-200 inline-flex items-center gap-1.5 px-3 py-1.5 bg-rose-600 hover:bg-rose-700 disabled:opacity-40 text-white text-xs font-semibold rounded-lg shadow-sm">
                                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
                                <span>Hapus Terpilih (<span id="selected-count">0</span>)</span>
                            </button>
                            {{- end}}
                        </div>
                    </div>

                    <div class="overflow-x-auto relative">
                        <table class="w-full text-left border-collapse min-w-[800px]">
                            <thead>
                                <tr class="bg-slate-50 border-b border-slate-100">
                                    {{- if not .IsReadOnly}}
                                    <th class="w-10 px-4 py-3.5 text-center">
                                        <input type="checkbox" id="check-all" onclick="toggleSelectAll(this)" class="w-4 h-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer" />
                                    </th>
                                    {{- end}}
                                    {{- range .ListFields}}
                                    <th class="px-6 py-3.5 text-xs font-bold text-slate-500 uppercase tracking-wider cursor-pointer hover:text-indigo-600 transition-colors select-none" onclick="sortBy('{{.OrmTag}}')">
                                        <div class="inline-flex items-center gap-1.5">
                                            <span>{{.Name}}</span>
                                            <span class="text-xs">
                                                {{"{{"}} if eq $.SortBy "{{.OrmTag}}" {{"}}"}}
                                                    {{"{{"}} if eq $.SortOrder "desc" {{"}}"}}↓{{"{{"}} else {{"}}"}}↑{{"{{"}} end {{"}}"}}
                                                {{"{{"}} else {{"}}"}}
                                                    <span class="text-slate-300">↕</span>
                                                {{"{{"}} end {{"}}"}}
                                            </span>
                                        </div>
                                    </th>
                                    {{- end}}
                                    <th class="sticky right-0 bg-slate-50 px-6 py-3.5 text-xs font-bold text-slate-500 uppercase tracking-wider text-right shadow-[-4px_0_8px_rgba(0,0,0,0.05)] z-20">Aksi</th>
                                </tr>
                            </thead>
                            <tbody class="divide-y divide-slate-100">
                                {{"{{"}}range .List{{"}}"}}
                                <tr class="group hover:bg-slate-50/50 transition-colors">
                                    {{- if not $.IsReadOnly}}
                                    <td class="w-10 px-4 py-4 text-center">
                                        <input type="checkbox" name="batch_ids" value="{{"{{"}} .{{$.PKName}} {{"}}"}}" onchange="updateBatchDeleteBtn()" class="row-checkbox w-4 h-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500 cursor-pointer" />
                                    </td>
                                    {{- end}}
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
                                    <td class="sticky right-0 bg-white group-hover:bg-slate-50/50 px-6 py-4 text-sm text-slate-700 text-right whitespace-nowrap shadow-[-4px_0_8px_rgba(0,0,0,0.05)] z-10">
                                        <div class="flex items-center justify-end space-x-1">
                                            <a href="/{{$.TableName}}/{{"{{"}} .{{$.PKName}} {{"}}"}}" class="inline-flex items-center justify-center p-1.5 rounded-lg text-emerald-600 hover:bg-emerald-50 hover:text-emerald-950 transition-colors" title="Show">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                                </svg>
                                            </a>
                                            {{- if not .IsReadOnly}}
                                            <a href="/{{$.TableName}}/{{"{{"}} .{{$.PKName}} {{"}}"}}/edit" class="inline-flex items-center justify-center p-1.5 rounded-lg text-indigo-600 hover:bg-indigo-50 hover:text-indigo-900 transition-colors" title="Update">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                </svg>
                                            </a>
                                            <form method="POST" action="/{{$.TableName}}/{{"{{"}} .{{$.PKName}} {{"}}"}}/delete" onsubmit="return confirm('Hapus data ini?')" class="inline">
                                                <button type="submit" class="inline-flex items-center justify-center p-1.5 rounded-lg text-rose-600 hover:bg-rose-50 hover:text-rose-900 transition-colors" title="Delete">
                                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                        <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                    </svg>
                                                </button>
                                            </form>
                                            {{- end}}
                                        </div>
                                    </td>
                                </tr>
                                {{"{{"}}else{{"}}"}}
                                <tr>
                                    <td colspan="{{if not .IsReadOnly}}{{add (len .ListFields) 2}}{{else}}{{add (len .ListFields) 1}}{{end}}" class="px-6 py-10 text-center text-sm text-slate-400">
                                        Tidak ada data ditemukan.
                                    </td>
                                </tr>
                                {{"{{"}}end{{"}}"}}
                            </tbody>
                        </table>
                    </div>

                    <!-- Pagination Controls -->
                    <div class="px-6 py-4 border-t border-slate-100 flex items-center justify-between">
                        <div class="text-sm text-slate-500 flex items-center space-x-1.5">
                            <span>Halaman</span>
                            <input 
                                type="number" 
                                value="{{"{{"}}.Page{{"}}"}}" 
                                min="1" 
                                max="{{"{{"}}.TotalPages{{"}}"}}" 
                                onchange="goToPage(this.value)"
                                onkeydown="if(event.key === 'Enter') goToPage(this.value)"
                                class="w-14 px-1.5 py-1 text-center font-semibold text-slate-800 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 bg-white [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                            />
                            <span>dari <span class="font-semibold text-slate-800">{{"{{"}}.TotalPages{{"}}"}}</span></span>
                        </div>
                        <div class="flex items-center space-x-2">
                            {{"{{"}} if .HasPrev {{"}}"}}
                            <a href="javascript:void(0)" onclick="goToPage('{{"{{"}}.PrevPage{{"}}"}}')" class="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-xs rounded-xl transition-all">
                                Sebelumnya
                            </a>
                            {{"{{"}} else {{"}}"}}
                            <span class="px-4 py-2 bg-slate-50 text-slate-300 font-semibold text-xs rounded-xl cursor-not-allowed">
                                Sebelumnya
                            </span>
                            {{"{{"}} end {{"}}"}}

                            {{"{{"}} if .HasNext {{"}}"}}
                            <a href="javascript:void(0)" onclick="goToPage('{{"{{"}}.NextPage{{"}}"}}')" class="px-4 py-2 bg-slate-900 hover:bg-slate-800 text-white font-semibold text-xs rounded-xl transition-all">
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
        function sortBy(col) {
            const params = new URLSearchParams(window.location.search);
            const currentSort = params.get('sort_by');
            const currentOrder = params.get('sort_order') || 'asc';
            if (currentSort === col) {
                params.set('sort_order', currentOrder === 'asc' ? 'desc' : 'asc');
            } else {
                params.set('sort_by', col);
                params.set('sort_order', 'asc');
            }
            params.set('page', 1);
            window.location.search = params.toString();
        }

        function toggleSelectAll(master) {
            const checkboxes = document.querySelectorAll('.row-checkbox');
            checkboxes.forEach(cb => cb.checked = master.checked);
            updateBatchDeleteBtn();
        }

        function updateBatchDeleteBtn() {
            const checked = document.querySelectorAll('.row-checkbox:checked');
            const btn = document.getElementById('btn-batch-delete');
            const countSpan = document.getElementById('selected-count');
            if (countSpan) countSpan.textContent = checked.length;
            if (btn) {
                if (checked.length > 0) {
                    btn.classList.remove('opacity-0', 'pointer-events-none');
                    btn.removeAttribute('disabled');
                } else {
                    btn.classList.add('opacity-0', 'pointer-events-none');
                    btn.setAttribute('disabled', 'true');
                }
            }
        }

        function submitBatchDelete() {
            const checked = document.querySelectorAll('.row-checkbox:checked');
            if (checked.length === 0) return;
            if (!confirm('Yakin ingin menghapus ' + checked.length + ' data terpilih?')) return;

            const form = document.createElement('form');
            form.method = 'POST';
            form.action = '/{{.TableName}}/batch-delete';
            checked.forEach(cb => {
                const input = document.createElement('input');
                input.type = 'hidden';
                input.name = 'ids';
                input.value = cb.value;
                form.appendChild(input);
            });
            document.body.appendChild(form);
            form.submit();
        }

        async function performSearch(keyword) {
            const params = new URLSearchParams(window.location.search);
            if (keyword) {
                params.set('keyword', keyword);
            } else {
                params.delete('keyword');
            }
            
            const pageSizeSelect = document.getElementsByName('page_size')[0];
            if (pageSizeSelect) {
                params.set('page_size', pageSizeSelect.value);
            }
            
            params.set('page', 1);
            
            await executeSearch(params);
        }

        async function goToPage(page) {
            const minPage = 1;
            const maxPage = parseInt('{{"{{"}}.TotalPages{{"}}"}}') || 1;
            let pageNum = parseInt(page) || 1;
            if (pageNum < minPage) pageNum = minPage;
            if (pageNum > maxPage) pageNum = maxPage;
            
            const params = new URLSearchParams(window.location.search);
            const keywordInput = document.getElementsByName('keyword')[0];
            if (keywordInput && keywordInput.value) {
                params.set('keyword', keywordInput.value);
            } else {
                params.delete('keyword');
            }
            params.set('page', pageNum);
            
            await executeSearch(params);
        }

        async function executeSearch(params) {
            try {
                const response = await fetch(window.location.pathname + '?' + params.toString());
                const html = await response.text();
                const parser = new DOMParser();
                const doc = parser.parseFromString(html, 'text/html');
                
                const newTbody = doc.querySelector('tbody');
                const currentTbody = document.querySelector('tbody');
                if (newTbody && currentTbody) {
                    currentTbody.innerHTML = newTbody.innerHTML;
                }
                
                const newPagination = doc.querySelector('.px-6.py-4.border-t');
                const currentPagination = document.querySelector('.px-6.py-4.border-t');
                if (newPagination && currentPagination) {
                    currentPagination.innerHTML = newPagination.innerHTML;
                }
                
                const newTotal = doc.querySelector('h2.font-bold');
                const currentTotal = document.querySelector('h2.font-bold');
                if (newTotal && currentTotal) {
                    currentTotal.innerHTML = newTotal.innerHTML;
                }
                updateBatchDeleteBtn();
            } catch (err) {
                console.error('AJAX search failed:', err);
            }
        }

        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const isCollapsed = sidebar.classList.toggle('w-18');
            if (isCollapsed) {
                sidebar.classList.remove('w-64');
                sidebar.classList.add('w-18');
                localStorage.setItem('sidebar-collapsed', 'true');
            } else {
                sidebar.classList.add('w-64');
                localStorage.setItem('sidebar-collapsed', 'false');
            }
        }
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
        {{"{{"}}include "public/sidebar.html" .{{"}}"}}

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
                {{"{{"}}include "public/mobile_nav.html" .{{"}}"}}

                <!-- Form Card -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6">
                    <h1 class="text-xl font-bold text-slate-900 mb-2">{{"{{"}} if .Id {{"}}"}}Edit{{"{{"}} else {{"}}"}}Tambah{{"{{"}} end {{"}}"}} {{.ShortName}}</h1>
                    <p class="text-sm text-slate-500 mb-6">Isi formulir berikut untuk menyimpan data.</p>
                    
                    <form method="POST" action="/{{.TableName}}{{"{{"}} if .Id {{"}}"}}/{{"{{"}} .Id {{"}}"}}{{"{{"}} end {{"}}"}}" class="space-y-6"{{if .HasUpload}} enctype="multipart/form-data"{{end}}>
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
                            {{- range .FormFields}}
                            <div class="{{if .IsFullWidth}}md:col-span-2{{else}}md:col-span-1{{end}}">
                                <label class="block text-sm font-semibold text-slate-700 mb-1.5">{{.Name}}{{if .IsRequired}} <span class="text-rose-500">*</span>{{end}}</label>
                                {{- if .EnumValues}}
                                {{- $fieldName := .Name}}
                                <select name="{{.JsonTag}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white"{{if .IsRequired}} required{{end}}>
                                    <option value="">-- Pilih {{.Name}} --</option>
                                    {{- range .EnumValues}}
                                    <option value="{{.}}" {{"{{"}} if eq (printf "%v" $.{{$fieldName}}) "{{.}}" {{"}}"}}selected{{"{{"}} end {{"}}"}}>{{.}}</option>
                                    {{- end}}
                                </select>
                               {{- else if .IsFK}}
                                {{- $fieldName := .Name}}
                                <select name="{{.JsonTag}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white"{{if .IsRequired}} required{{end}} data-fk-table="{{.FKTable}}">
                                    <option value="">-- Pilih dari tabel {{.FKTable}} --</option>
                                    {{"{{"}} range .FkOpts{{.Name}} {{"}}"}}
                                    <option value="{{"{{"}} .Id {{"}}"}}" {{"{{"}} if eq (printf "%v" $.{{$fieldName}}) (printf "%v" .Id) {{"}}"}}selected{{"{{"}} end {{"}}"}}>
                                        {{"{{"}} .Label {{"}}"}}
                                    </option>
                                    {{"{{"}} end {{"}}"}}
                                </select>
                               {{- else if .IsTextarea}}
                                <textarea name="{{.JsonTag}}" rows="4" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm resize-y" placeholder="{{if .IsJson}}Masukkan {{.Name}} (Format JSON, contoh: {}){{else}}Masukkan {{.Name}}...{{end}}"{{if .IsRequired}} required{{end}}>{{"{{"}} .{{.Name}} {{"}}"}}</textarea>
                                {{- else if eq .HTMLType "file"}}
                                <input type="file" name="{{.JsonTag}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm file:mr-4 file:py-1.5 file:px-3 file:rounded-xl file:border-0 file:text-xs file:font-semibold file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100" />
                                {{- else if eq .HTMLType "date"}}
                                <input type="date" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}}.Layout "2006-01-02" {{"}}"}}{{"{{"}} end {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm"{{if .IsRequired}} required{{end}} />
                                {{- else if eq .HTMLType "time"}}
                                <input type="time" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}}.Layout "15:04:05" {{"}}"}}{{"{{"}} end {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm"{{if .IsRequired}} required{{end}} />
                                {{- else if eq .HTMLType "datetime-local"}}
                                <input type="datetime-local" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}}.Layout "2006-01-02T15:04" {{"}}"}}{{"{{"}} end {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm"{{if .IsRequired}} required{{end}} />
                                {{- else if eq .HTMLType "checkbox"}}
                                <label class="flex items-center gap-2 cursor-pointer pt-2">
                                    <input type="checkbox" name="{{.JsonTag}}" value="1" {{"{{"}} if .{{.Name}} {{"}}"}}checked{{"{{"}} end {{"}}"}} class="w-4 h-4 accent-indigo-600" />
                                    <span class="text-sm text-slate-600">Aktif</span>
                                </label>
                                {{- else if eq .HTMLType "range"}}
                                <div class="flex items-center gap-3 pt-2">
                                    <input type="range" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}} {{"}}"}}{{"{{"}} else {{"}}"}}{{if .Rules.min}}{{.Rules.min}}{{else}}0{{end}}{{"{{"}} end {{"}}"}}" min="{{if .Rules.min}}{{.Rules.min}}{{else}}0{{end}}" max="{{if .Rules.max}}{{.Rules.max}}{{else}}100{{end}}" step="{{if .Rules.step}}{{.Rules.step}}{{else}}1{{end}}" class="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-indigo-600 focus:outline-none" oninput="this.nextElementSibling.innerText = this.value" />
                                    <span class="text-sm font-semibold text-slate-600 min-w-[2.5rem] text-right">{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}} {{"}}"}}{{"{{"}} else {{"}}"}}{{if .Rules.min}}{{.Rules.min}}{{else}}0{{end}}{{"{{"}} end {{"}}"}}</span>
                                </div>
                                {{- else}}
                                <input type="{{.HTMLType}}" name="{{.JsonTag}}" value="{{"{{"}} .{{.Name}} {{"}}"}}" class="w-full px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm"{{if .IsRequired}} required{{end}} />
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

        // Intercept form submit:
        // 1. If edit mode (.Id exists), submit via HTTP PUT.
        // 2. Format raw time inputs to YYYY-MM-DD HH:MM:SS for GoFrame parser.
        document.querySelector('form').addEventListener('submit', function(e) {
            this.querySelectorAll('input[type="time"]').forEach(input => {
                if (input.value) {
                    let formattedVal = "";
                    if (input.value.length === 5) {
                        formattedVal = '2000-01-01 ' + input.value + ':00';
                    } else if (input.value.length === 8) {
                        formattedVal = '2000-01-01 ' + input.value;
                    }
                    
                    if (formattedVal) {
                        const hiddenInput = document.createElement('input');
                        hiddenInput.type = 'hidden';
                        hiddenInput.name = input.name;
                        hiddenInput.value = formattedVal;
                        input.removeAttribute('name');
                        this.appendChild(hiddenInput);
                    }
                }
            });

            const isEdit = {{"{{"}} if .Id {{"}}"}}true{{"{{"}} else {{"}}"}}false{{"{{"}} end {{"}}"}};
            if (isEdit) {
                e.preventDefault();
                const formData = new FormData(this);
                fetch(this.action, {
                    method: 'PUT',
                    body: formData
                }).then(res => {
                    if (res.ok || res.redirected) {
                        window.location.href = '/{{.TableName}}';
                    } else {
                        return res.text().then(text => alert(text || 'Gagal update data'));
                    }
                }).catch(err => {
                    console.error(err);
                    alert('Terjadi kesalahan koneksi');
                });
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
        {{"{{"}}include "public/sidebar.html" .{{"}}"}}

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
                {{"{{"}}include "public/mobile_nav.html" .{{"}}"}}

                <!-- Detail Breadcrumb -->
                <div class="flex items-center gap-2 text-sm text-slate-500 mb-6">
                    <a href="/{{.TableName}}" class="hover:text-indigo-600 transition-colors font-medium">Daftar {{.ShortName}}</a>
                    <svg class="h-4 w-4 text-slate-300" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
                    </svg>
                    <span class="text-slate-700 font-semibold">Detail Data</span>
                </div>

                <!-- Detail Card -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden p-6">
                    <div class="flex items-center justify-between pb-4 border-b border-slate-100 mb-6">
                        <div>
                            <h1 class="text-xl font-bold text-slate-900">Detail {{.ShortName}}</h1>
                            <p class="text-xs text-slate-400 mt-1">Menampilkan seluruh informasi data secara lengkap.</p>
                        </div>
                        <div class="flex items-center gap-2">
                            {{- if not .IsReadOnly}}
                            <a href="/{{.TableName}}/{{"{{"}} .Id {{"}}"}}/edit" class="px-4 py-2 bg-indigo-50 hover:bg-indigo-100 text-indigo-700 font-semibold text-xs rounded-xl transition-all flex items-center gap-1.5">
                                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                </svg>
                                Edit
                            </a>
                            {{- end}}
                            <a href="/{{.TableName}}" class="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-xs rounded-xl transition-all">Kembali</a>
                        </div>
                    </div>

                    <div class="grid grid-cols-1 md:grid-cols-2 gap-y-5 gap-x-8">
                        {{- range .Fields}}
                        <div class="border-b border-slate-50 pb-3 {{if .IsFullWidth}}md:col-span-2{{else}}md:col-span-1{{end}}">
                            <span class="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1">{{.Name}}</span>
                            {{- if eq .HTMLType "file"}}
                            {{"{{"}} if .{{.Name}} {{"}}"}}
                            <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-indigo-50 text-indigo-700 border border-indigo-100">
                                Ada File (BLOB/Gambar)
                            </span>
                            {{"{{"}} else {{"}}"}}
                            <span class="text-sm text-slate-500 font-normal">-</span>
                            {{"{{"}} end {{"}}"}}
                            {{- else}}
                            <span class="text-sm font-medium text-slate-700 whitespace-pre-wrap">{{"{{"}} .{{.Name}} {{"}}"}}</span>
                            {{- end}}
                        </div>
                        {{- end}}
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
    </script>
</body>
</html>
`))

var filterHTMLTemplate = template.Must(template.New("filter_html").Funcs(template.FuncMap{
	"abbrev": abbrev,
	"add": func(a, b int) int {
		return a + b
	},
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
}).Parse(`{{"{{"}}$dot := .{{"}}"}}<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Filter {{.ShortName}}</title>
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
        {{"{{"}}include "public/sidebar.html" .{{"}}"}}

        <!-- Content -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 max-w-5xl">

                <!-- Breadcrumb -->
                <div class="flex items-center gap-2 text-sm text-slate-500 mb-6">
                    <a href="/{{.TableName}}" class="hover:text-indigo-600 transition-colors font-medium">{{.ShortName}}</a>
                    <svg class="h-4 w-4 text-slate-300" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
                    </svg>
                    <span class="text-slate-700 font-semibold">Filter Lanjutan</span>
                </div>

                <!-- Header -->
                <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
                    <div>
                        <h1 class="text-2xl font-bold text-slate-900">Filter {{.ShortName}}</h1>
                        <p class="text-sm text-slate-500 mt-0.5">Isi kolom filter yang ingin kamu terapkan, lalu klik Terapkan Filter.</p>
                    </div>
                    {{- if not .IsReadOnly}}
                    <div>
                        <a href="/{{.TableName}}/create" class="inline-flex items-center justify-center px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl shadow-md shadow-indigo-600/10 transition-all gap-2 w-full sm:w-auto">
                            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
                            </svg>
                            Tambah {{.ShortName}}
                        </a>
                    </div>
                    {{- end}}
                </div>

                <!-- Filter Form Card -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 mb-6">
                    <form id="filter-form" onsubmit="event.preventDefault(); performFilter(1);" class="space-y-5">
                        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            {{- range .FormFields}}
                            <div class="{{if or .IsFullWidth (eq .HTMLType "number") (eq .HTMLType "date") (eq .HTMLType "datetime-local") (eq .HTMLType "time")}}sm:col-span-2{{else}}sm:col-span-1{{end}}">
                                <label class="block text-sm font-semibold text-slate-700 mb-1.5">{{.Name}}</label>
                                {{- if .EnumValues}}
                                <select name="f_{{.JsonTag}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white text-slate-700">
                                    <option value="">-- Semua --</option>
                                    {{- range .EnumValues}}
                                    <option value="{{.}}" {{"{{"}} if eq (printf "%v" (index $dot (printf "f_%s" $.JsonTag))) "{{.}}" {{"}}"}}selected{{"{{"}} end {{"}}"}}>{{.}}</option>
                                    {{- end}}
                                </select>
                                {{- else if eq .HTMLType "checkbox"}}
                                <select name="f_{{.JsonTag}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white text-slate-700">
                                    <option value="">-- Semua --</option>
                                    <option value="1" {{"{{"}} if eq (printf "%v" (index $dot (printf "f_%s" .JsonTag))) "1" {{"}}"}}selected{{"{{"}} end {{"}}"}}>Ya</option>
                                    <option value="0" {{"{{"}} if eq (printf "%v" (index $dot (printf "f_%s" .JsonTag))) "0" {{"}}"}}selected{{"{{"}} end {{"}}"}}>Tidak</option>
                                </select>
                                {{- else if eq .HTMLType "number"}}
                                <div class="flex flex-col sm:flex-row items-center gap-2 w-full">
                                    <input type="number" name="f_{{.JsonTag}}_min" value="{{"{{"}} index $dot (printf "f_%s_min" .JsonTag) {{"}}"}}" placeholder="Nilai minimum" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                    <span class="text-slate-400 text-xs font-semibold whitespace-nowrap">s/d</span>
                                    <input type="number" name="f_{{.JsonTag}}_max" value="{{"{{"}} index $dot (printf "f_%s_max" .JsonTag) {{"}}"}}" placeholder="Nilai maksimum" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                </div>
                                {{- else if eq .HTMLType "date"}}
                                <div class="flex flex-col sm:flex-row items-center gap-2 w-full">
                                    <input type="date" name="f_{{.JsonTag}}_from" value="{{"{{"}} index $dot (printf "f_%s_from" .JsonTag) {{"}}"}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                    <span class="text-slate-400 text-xs font-semibold whitespace-nowrap">s/d</span>
                                    <input type="date" name="f_{{.JsonTag}}_to" value="{{"{{"}} index $dot (printf "f_%s_to" .JsonTag) {{"}}"}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                </div>
                                {{- else if eq .HTMLType "datetime-local"}}
                                <div class="flex flex-col sm:flex-row items-center gap-2 w-full">
                                    <input type="datetime-local" name="f_{{.JsonTag}}_from" value="{{"{{"}} index $dot (printf "f_%s_from" .JsonTag) {{"}}"}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                    <span class="text-slate-400 text-xs font-semibold whitespace-nowrap">s/d</span>
                                    <input type="datetime-local" name="f_{{.JsonTag}}_to" value="{{"{{"}} index $dot (printf "f_%s_to" .JsonTag) {{"}}"}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                </div>
                                {{- else if eq .HTMLType "time"}}
                                <div class="flex flex-col sm:flex-row items-center gap-2 w-full">
                                    <input type="time" name="f_{{.JsonTag}}_from" value="{{"{{"}} index $dot (printf "f_%s_from" .JsonTag) {{"}}"}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                    <span class="text-slate-400 text-xs font-semibold whitespace-nowrap">s/d</span>
                                    <input type="time" name="f_{{.JsonTag}}_to" value="{{"{{"}} index $dot (printf "f_%s_to" .JsonTag) {{"}}"}}" class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                </div>
                                {{- else if .IsTextarea}}
                                <input type="text" name="f_{{.JsonTag}}" value="{{"{{"}} index $dot (printf "f_%s" .JsonTag) {{"}}"}}" placeholder="Cari {{.Name}}..." class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                {{- else}}
                                <input type="{{.HTMLType}}" name="f_{{.JsonTag}}" value="{{"{{"}} index $dot (printf "f_%s" .JsonTag) {{"}}"}}" placeholder="Cari {{.Name}}..." class="w-full px-3.5 py-2.5 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm" />
                                {{- end}}
                            </div>
                            {{- end}}
                        </div>

                        <!-- Ukuran Halaman -->
                        <div class="flex items-center gap-2 pt-2 border-t border-slate-100">
                            <label class="text-sm font-semibold text-slate-600 whitespace-nowrap">Per halaman:</label>
                            <select name="page_size" onchange="performFilter(1)" class="px-3.5 py-2 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all text-sm bg-white text-slate-700">
                                <option value="10" {{"{{"}} if eq .PageSize 10 {{"}}"}}selected{{"{{"}} end {{"}}"}} >10</option>
                                <option value="20" {{"{{"}} if eq .PageSize 20 {{"}}"}}selected{{"{{"}} end {{"}}"}} >20</option>
                                <option value="50" {{"{{"}} if eq .PageSize 50 {{"}}"}}selected{{"{{"}} end {{"}}"}} >50</option>
                                <option value="100" {{"{{"}} if eq .PageSize 100 {{"}}"}}selected{{"{{"}} end {{"}}"}} >100</option>
                            </select>
                        </div>

                        <!-- Action Buttons -->
                        <div class="flex items-center gap-3 pt-2">
                            <button type="submit" class="px-6 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl transition-all shadow-md shadow-indigo-600/10">
                                Terapkan Filter
                            </button>
                            <a href="/{{.TableName}}" class="px-6 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-sm rounded-xl transition-all">
                                Reset &amp; Kembali
                            </a>
                        </div>
                    </form>
                </div>

                <!-- Table Card (Results View) -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
                    <div class="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
                        <h2 class="font-bold text-slate-900">Hasil Pencarian: {{"{{"}} .Total {{"}}"}} data</h2>
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
                                    <td class="sticky right-0 bg-white group-hover:bg-slate-50/50 px-6 py-4 text-sm text-slate-700 text-right whitespace-nowrap shadow-[-4px_0_8px_rgba(0,0,0,0.05)] z-10">
                                        <div class="flex items-center justify-end space-x-1">
                                            <a href="/{{$.TableName}}/{{"{{"}} .{{$.PKName}} {{"}}"}}" class="inline-flex items-center justify-center p-1.5 rounded-lg text-emerald-600 hover:bg-emerald-50 hover:text-emerald-950 transition-colors" title="Show">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                                </svg>
                                            </a>
                                            {{- if not .IsReadOnly}}
                                            <a href="/{{$.TableName}}/{{"{{"}} .{{$.PKName}} {{"}}"}}/edit" class="inline-flex items-center justify-center p-1.5 rounded-lg text-indigo-600 hover:bg-indigo-50 hover:text-indigo-900 transition-colors" title="Update">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                </svg>
                                            </a>
                                            <form method="POST" action="/{{$.TableName}}/{{"{{"}} .{{$.PKName}} {{"}}"}}/delete" onsubmit="return confirm('Hapus data ini?')" class="inline">
                                                <button type="submit" class="inline-flex items-center justify-center p-1.5 rounded-lg text-rose-600 hover:bg-rose-50 hover:text-rose-900 transition-colors" title="Delete">
                                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                        <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                    </svg>
                                                </button>
                                            </form>
                                            {{- end}}
                                        </div>
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
                        <div class="text-sm text-slate-500 flex items-center space-x-1.5">
                            <span>Halaman</span>
                            <input 
                                type="number" 
                                value="{{"{{"}}.Page{{"}}"}}" 
                                min="1" 
                                max="{{"{{"}}.TotalPages{{"}}"}}" 
                                onchange="goToPage(this.value)"
                                onkeydown="if(event.key === 'Enter') goToPage(this.value)"
                                class="w-14 px-1.5 py-1 text-center font-semibold text-slate-800 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 bg-white [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                            />
                            <span>dari <span class="font-semibold text-slate-800">{{"{{"}}.TotalPages{{"}}"}}</span></span>
                        </div>
                        <div class="flex items-center space-x-2">
                            {{"{{"}} if .HasPrev {{"}}"}}
                            <a href="javascript:void(0)" onclick="goToPage('{{"{{"}}.PrevPage{{"}}"}}')" class="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-xs rounded-xl transition-all">
                                Sebelumnya
                            </a>
                            {{"{{"}} else {{"}}"}}
                            <span class="px-4 py-2 bg-slate-50 text-slate-300 font-semibold text-xs rounded-xl cursor-not-allowed">
                                Sebelumnya
                            </span>
                            {{"{{"}} end {{"}}"}}

                            {{"{{"}} if .HasNext {{"}}"}}
                            <a href="javascript:void(0)" onclick="goToPage('{{"{{"}}.NextPage{{"}}"}}')" class="px-4 py-2 bg-slate-900 hover:bg-slate-800 text-white font-semibold text-xs rounded-xl transition-all">
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
        async function performFilter(pageNum) {
            const form = document.getElementById('filter-form');
            if (!form) return;
            
            const formData = new FormData(form);
            const params = new URLSearchParams();
            
            for (const [key, value] of formData.entries()) {
                if (value !== "") {
                    params.append(key, value);
                }
            }
            
            params.set('page', pageNum || 1);
            
            try {
                const response = await fetch(window.location.pathname + '?' + params.toString());
                const html = await response.text();
                const parser = new DOMParser();
                const doc = parser.parseFromString(html, 'text/html');
                
                const newTbody = doc.querySelector('tbody');
                const currentTbody = document.querySelector('tbody');
                if (newTbody && currentTbody) {
                    currentTbody.innerHTML = newTbody.innerHTML;
                }
                
                const newPagination = doc.querySelector('.px-6.py-4.border-t');
                const currentPagination = document.querySelector('.px-6.py-4.border-t');
                if (newPagination && currentPagination) {
                    currentPagination.innerHTML = newPagination.innerHTML;
                }
                
                const newTotal = doc.querySelector('h2.font-bold');
                const currentTotal = document.querySelector('h2.font-bold');
                if (newTotal && currentTotal) {
                    currentTotal.innerHTML = newTotal.innerHTML;
                }
            } catch (err) {
                console.error('Filter failed:', err);
            }
        }

        function goToPage(page) {
            const minPage = 1;
            const maxPage = parseInt('{{"{{"}}.TotalPages{{"}}"}}') || 1;
            let pageNum = parseInt(page) || 1;
            if (pageNum < minPage) pageNum = minPage;
            if (pageNum > maxPage) pageNum = maxPage;
            
            performFilter(pageNum);
        }

        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            if (sidebar.classList.contains('w-64')) {
                sidebar.classList.remove('w-64');
                sidebar.classList.add('w-18');
                localStorage.setItem('sidebar-collapsed', 'true');
            } else {
                sidebar.classList.add('w-64');
                localStorage.setItem('sidebar-collapsed', 'false');
            }
        }
    </script>
</body>
</html>
`))

var ctrlNewTemplate = template.Must(template.New("ctrl_new").Parse(`package {{.TableName}}

import (
	"context"
	"github.com/gogf/gf/v2/container/gmap"
{{- if .HasFK}}
	"github.com/gogf/gf/v2/frame/g"
	"{{.ModuleName}}/internal/dao"
	"{{.ModuleName}}/internal/model"
{{- end}}
{{- if .HasGtime}}
	"github.com/gogf/gf/v2/os/gtime"
{{- end}}
	"{{.ModuleName}}/api/{{.TableName}}"
	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/model/entity"
)


type ControllerV1 struct{}

func NewV1() {{.TableName}}.I{{.StructName}}V1 {
	return &ControllerV1{}
}

func get{{.ShortName}}Header(ctx context.Context) v1.{{.ShortName}}Header {
	return v1.{{.ShortName}}Header{
		{{- range .Fields}}
		{{.Name}}: v1.{{$.ShortName}}Field{
			Type: "{{- if .IsFK}}select{{else}}{{.DataType}}{{- end}}",
			{{- if .IsPK}}
			IsPK: true,
			{{- end}}
			IsRequired: {{.IsRequired}},
			{{- if .EnumValues}}
			Values: []string{
				{{- range .EnumValues}}
				"{{.}}",
				{{- end}}
			},
			{{- end}}
            {{- if .IsFK}}
            Extra: map[string]interface{}{
                "options": get{{.FKStructName}}Options(ctx),
            },
			{{- else if .Rules}}
			Extra: map[string]interface{}{
				{{- range $k, $v := .Rules}}
				"{{$k}}": {{$v}},
				{{- end}}
			},
			{{- end}}
		},
		{{- end}}
	}
}

func format{{.ShortName}}Data(data *entity.{{.StructName}}) *gmap.ListMap {
	if data == nil {
		return nil
	}
	m := gmap.NewListMap()
	{{- range .Fields}}
	{{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}
	m.Set("{{.JsonTag}}", data.{{.Name}})
	{{- else if eq .HTMLType "date"}}
	if g.IsEmpty(data.{{.Name}}) {
		m.Set("{{.JsonTag}}", nil)
	} else {
		m.Set("{{.JsonTag}}", gtime.New(data.{{.Name}}).Layout("2006-01-02"))
	}
	{{- else if eq .HTMLType "datetime-local"}}
	if g.IsEmpty(data.{{.Name}}) {
		m.Set("{{.JsonTag}}", nil)
	} else {
		m.Set("{{.JsonTag}}", gtime.New(data.{{.Name}}).Layout("2006-01-02 15:04:05"))
	}
	{{- else if eq .HTMLType "time"}}
	if g.IsEmpty(data.{{.Name}}) {
		m.Set("{{.JsonTag}}", nil)
	} else {
		m.Set("{{.JsonTag}}", gtime.New(data.{{.Name}}).Layout("15:04:05"))
	}
	{{- else}}
	m.Set("{{.JsonTag}}", data.{{.Name}})
	{{- end}}
	{{- end}}
	return m
}

func format{{.ShortName}}DataList(list []*entity.{{.StructName}}) []*gmap.ListMap {
	rows := make([]*gmap.ListMap, 0, len(list))
	for _, item := range list {
		rows = append(rows, format{{.ShortName}}Data(item))
	}
	return rows
}

{{- range .Fields}}
{{- if .IsFK}}
func get{{.FKStructName}}Options(ctx context.Context) []g.Map {
    rows, err := dao.{{.FKStructName}}.Ctx(ctx).All()
    if err != nil {
        return nil
    }
	var options []g.Map
    for _, row := range rows {
        var item entity.{{.FKStructName}}
        _ = row.Struct(&item)

        out := model.To{{.FKStructName}}Output(&item)
        options = append(options, g.Map{
            "id": out.Id,
            "label": out.Label,
        })
    }

    return options
}
{{- end}}
{{- end}}
`))

var ctrlFormTemplate = template.Must(template.New("ctrl_form").Parse(`package {{.TableName}}

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/service"
	{{- if .HasFK}}
	"{{.ModuleName}}/internal/dao"
	"{{.ModuleName}}/internal/model/entity"
	{{- end}}
)

// ShowCreateForm menampilkan form tambah data baru.
func (c *ControllerV1) ShowCreateForm(ctx context.Context, req *v1.ShowCreateFormReq) (res *v1.ShowCreateFormRes, err error) {
	r := g.RequestFromCtx(ctx)

	formValues := map[string]interface{}{}
	{{- range .FormFields}}
	{{- if .IsFK}}
	var fkOpts{{.Name}} []*entity.{{.FKStructName}}
	_ = dao.{{.FKStructName}}.Ctx(ctx).Scan(&fkOpts{{.Name}})
	formValues["FkOpts{{.Name}}"] = fkOpts{{.Name}}
	{{- end}}
	{{- end}}

	r.Response.WriteTpl("{{.TableName}}/form.html", formValues)
	r.Exit()
	return nil, nil
}

// ShowEditForm menampilkan form edit data berdasarkan ID.
func (c *ControllerV1) ShowEditForm(ctx context.Context, req *v1.ShowEditFormReq) (res *v1.ShowEditFormRes, err error) {
	r := g.RequestFromCtx(ctx)
	{{- if or (eq .PKType "uint64") (eq .PKType "int") (eq .PKType "int64")}}
	if req.Id == 0 {
		r.Response.RedirectTo("/{{.TableName}}")
		return nil, nil
	}
	{{- else}}
	if req.Id == "" {
		r.Response.RedirectTo("/{{.TableName}}")
		return nil, nil
	}
	{{- end}}
	data, err := service.{{.StructName}}().Get(ctx, req.Id)
	if err != nil || data == nil {
		r.Response.RedirectTo("/{{.TableName}}")
		return nil, nil
	}

	// Buat map explicit agar GoFrame view engine lancar me-render PascalCase struct fields
	formValues := map[string]interface{}{
		"Id": data.{{.PKName}},
		{{- range .FormFields}}
		{{- if ne .Name "Id"}}
		"{{.Name}}": data.{{.Name}},
		{{- end}}
		{{- if .IsFK}}
		"FkOpts{{.Name}}": func() interface{} {
			var res []*entity.{{.FKStructName}}
			_ = dao.{{.FKStructName}}.Ctx(ctx).Scan(&res)
			return res
		}(),
		{{- end}}
		{{- end}}
	}

	r.Response.WriteTpl("{{.TableName}}/form.html", formValues)
	r.Exit()
	return nil, nil
}

// DeleteAction menghapus data berdasarkan ID dari form POST.
func (c *ControllerV1) DeleteAction(ctx context.Context, req *v1.DeleteActionReq) (res *v1.DeleteActionRes, err error) {
	r := g.RequestFromCtx(ctx)
	{{- if or (eq .PKType "uint64") (eq .PKType "int") (eq .PKType "int64")}}
	if req.Id != 0 {
		_ = service.{{.StructName}}().Delete(ctx, req.Id)
	}
	{{- else}}
	if req.Id != "" {
		_ = service.{{.StructName}}().Delete(ctx, req.Id)
	}
	{{- end}}
	r.Response.RedirectTo("/{{.TableName}}")
	r.Exit()
	return nil, nil
}
`))

var utilityResponseTemplate = template.Must(template.New("utility_response").Parse(`package response

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// JsonSuccess mengirim response JSON sukses dan menghentikan request (r.Exit).
func JsonSuccess(r *ghttp.Request, data ...interface{}) {
	if r == nil {
		return
	}
	resMap := g.Map{
		"code":    0,
		"message": "success",
	}
	if len(data) > 0 && data[0] != nil {
		resMap["data"] = data[0]
	}
	r.Response.WriteJson(resMap)
	r.Exit()
}

// JsonError mengirim response JSON error dengan status code (default 400 Bad Request) dan menghentikan request (r.Exit).
func JsonError(r *ghttp.Request, err error, statusCode ...int) {
	if r == nil {
		return
	}
	status := 400
	if len(statusCode) > 0 {
		status = statusCode[0]
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	r.Response.WriteStatus(status)
	r.Response.WriteJson(g.Map{
		"code":    1,
		"message": msg,
	})
	r.Exit()
}
`))

var ctrlListTemplate = template.Must(template.New("ctrl_list").Parse(`package {{.TableName}}
 
import (
 	"context"
	"strings"
 
 	"github.com/gogf/gf/v2/frame/g"
 	"github.com/gogf/gf/v2/net/ghttp"

 	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/model"
 	"{{.ModuleName}}/internal/service"
 	"{{.ModuleName}}/utility/response"
)

func (c *ControllerV1) List{{.ShortName}}(ctx context.Context, req *v1.List{{.ShortName}}Req) (res *v1.List{{.ShortName}}Res, err error) {
	r := ghttp.RequestFromCtx(ctx)
	isJson := r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json")

	page := req.Page
	pageSize := req.PageSize
	if !isJson {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
	}

	filters := make(map[string]interface{})
	if r != nil {
		for k, v := range r.GetMap() {
			if strings.HasPrefix(k, "f_") {
				filters[k] = v
			}
		}
	}

	in := model.{{.ShortName}}ListInput{
		Page:      page,
		PageSize:  pageSize,
		Keyword:   req.Keyword,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Filters:   filters,
	}

	list, total, err := service.{{.StructName}}().List(ctx, in)
	if err != nil {
		if isJson {
			response.JsonError(r, err)
			return nil, nil
		}
		return nil, err
	}
	// Jika client meminta JSON (misal dari Flutter)
	if isJson {
		dataRows := format{{.ShortName}}DataList(list)
		totalPages := 1
		if pageSize > 0 {
			totalPages = (total + pageSize - 1) / pageSize
			if totalPages == 0 {
				totalPages = 1
			}
		}

		resData := &v1.List{{.ShortName}}Res{
			Header:        get{{.ShortName}}Header(ctx),
			Value:         dataRows,
			Values:        dataRows,
			Total:         total,
			Page:          page,
			PageSize:      pageSize,
			JumlahHalaman: totalPages,
			TotalPage:     totalPages,
		}

		response.JsonSuccess(r, resData)
		return nil, nil
	}

	startIndex := (page-1)*pageSize + 1
	if total == 0 {
		startIndex = 0
	}
	endIndex := page * pageSize
	if endIndex > total {
		endIndex = total
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	r.Response.WriteTpl("{{.TableName}}/list.html", g.Map{
		"List":       list,
		"Total":      total,
		"Page":       page,
		"PageSize":   pageSize,
		"Keyword":    req.Keyword,
		"SortBy":     req.SortBy,
		"SortOrder":  req.SortOrder,
		"PKName":     "{{.PKName}}",
		"StartIndex": startIndex,
		"EndIndex":   endIndex,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
		"HasPrev":    page > 1,
		"HasNext":    page*pageSize < total,
		"TotalPages": totalPages,
	})
	r.Exit()
	return nil, nil
}

func (c *ControllerV1) Filter{{.ShortName}}(ctx context.Context, req *v1.Filter{{.ShortName}}Req) (res *v1.Filter{{.ShortName}}Res, err error) {
	r := ghttp.RequestFromCtx(ctx)
	isJson := r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json")

	page := req.Page
	pageSize := req.PageSize
	if !isJson {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 10
		}
	}

	filters := make(map[string]interface{})
	if r != nil {
		for k, v := range r.GetMap() {
			if strings.HasPrefix(k, "f_") {
				filters[k] = v
			}
		}
	}

	in := model.{{.ShortName}}ListInput{
		Page:      page,
		PageSize:  pageSize,
		Keyword:   req.Keyword,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Filters:   filters,
	}

	list, total, err := service.{{.StructName}}().List(ctx, in)
	if err != nil {
		if isJson {
			response.JsonError(r, err)
			return nil, nil
		}
		return nil, err
	}
	// Jika client meminta JSON (misal dari Flutter)
	if isJson {
		dataRows := format{{.ShortName}}DataList(list)
		totalPages := 1
		if pageSize > 0 {
			totalPages = (total + pageSize - 1) / pageSize
			if totalPages == 0 {
				totalPages = 1
			}
		}

		resData := &v1.Filter{{.ShortName}}Res{
			Header:        get{{.ShortName}}Header(ctx),
			Value:         dataRows,
			Values:        dataRows,
			Total:         total,
			Page:          page,
			PageSize:      pageSize,
			JumlahHalaman: totalPages,
			TotalPage:     totalPages,
		}

		response.JsonSuccess(r, resData)
		return nil, nil
	}

	startIndex := (page-1)*pageSize + 1
	if total == 0 {
		startIndex = 0
	}
	endIndex := page * pageSize
	if endIndex > total {
		endIndex = total
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	tplData := g.Map{
		"List":       list,
		"Total":      total,
		"Page":       page,
		"PageSize":   pageSize,
		"Keyword":    req.Keyword,
		"SortBy":     req.SortBy,
		"SortOrder":  req.SortOrder,
		"PKName":     "{{.PKName}}",
		"StartIndex": startIndex,
		"EndIndex":   endIndex,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
		"HasPrev":    page > 1,
		"HasNext":    page*pageSize < total,
		"TotalPages": totalPages,
	}

	// Populate template data with all request params to keep filter fields filled in
	if r != nil {
		for k, v := range r.GetMap() {
			tplData[k] = v
		}
	}

	r.Response.WriteTpl("{{.TableName}}/filter.html", tplData)
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
 	"{{.ModuleName}}/utility/response"
)

func (c *ControllerV1) Get{{.ShortName}}(ctx context.Context, req *v1.Get{{.ShortName}}Req) (res *v1.Get{{.ShortName}}Res, err error) {
	r := ghttp.RequestFromCtx(ctx)
	isJson := r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json")

	data, err := service.{{.StructName}}().Get(ctx, req.Id)
	if err != nil {
		if isJson {
			response.JsonError(r, err)
			return nil, nil
		}
		return nil, err
	}

	// Jika client meminta JSON (misal dari Flutter)
	if isJson {
		rowData := format{{.ShortName}}Data(data)
		resData := &v1.Get{{.ShortName}}Res{
			Header: get{{.ShortName}}Header(ctx),
			Value:  rowData,
			Values: rowData,
		}
		response.JsonSuccess(r, resData)
		return nil, nil
	}

	r.Response.WriteTpl("{{.TableName}}/detail.html", g.Map{
		"PKName": "{{.PKName}}",
		"Id": data.{{.PKName}},
		{{- range .Fields}}
		{{- if ne .Name "Id"}}
		"{{.Name}}": data.{{.Name}},
		{{- end}}
		{{- end}}
	})
	r.Exit()
	return nil, nil
}
`))

var ctrlCreateTemplate = template.Must(template.New("ctrl_create").Funcs(template.FuncMap{
	"contains": strings.Contains,
}).Parse(`package {{.TableName}}
 
import (
 	"context"
{{- $hasFile := false}}
{{- range .FormFields}}
{{- if eq .HTMLType "file"}}{{$hasFile = true}}{{end}}
{{- end}}
{{- if or .HasUpload $hasFile}}
 	"io"
{{- end}}
 
 	"github.com/gogf/gf/v2/net/ghttp"
 
 	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
 	"{{.ModuleName}}/internal/model/do"
 	"{{.ModuleName}}/internal/service"
 	"{{.ModuleName}}/utility/response"
)

func (c *ControllerV1) Create{{.ShortName}}(ctx context.Context, req *v1.Create{{.ShortName}}Req) (res *v1.Create{{.ShortName}}Res, err error) {
	r := ghttp.RequestFromCtx(ctx)
	isJson := r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json")

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

	data, err := service.{{.StructName}}().Create(ctx, createData)
	if err != nil {
		if isJson {
			response.JsonError(r, err)
			return nil, nil
		}
		return nil, err
	}

	// Jika client meminta JSON (misal dari Flutter)
	if isJson {
		rowData := format{{.ShortName}}Data(data)
		resData := &v1.Create{{.ShortName}}Res{
			Header: get{{.ShortName}}Header(ctx),
			Value:  rowData,
			Values: rowData,
		}
		response.JsonSuccess(r, resData)
		return nil, nil
	}

	r.Response.RedirectTo("/{{.TableName}}")
	r.Exit()
	return nil, nil
}
`))

var ctrlUpdateTemplate = template.Must(template.New("ctrl_update").Funcs(template.FuncMap{
	"contains": strings.Contains,
}).Parse(`package {{.TableName}}

import (
	"context"
{{- $hasFile := false}}
{{- range .FormFields}}
{{- if eq .HTMLType "file"}}{{$hasFile = true}}{{end}}
{{- end}}
{{- if or .HasUpload $hasFile}}
	"io"
{{- end}}

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/model/do"
	"{{.ModuleName}}/internal/service"
	"{{.ModuleName}}/utility/response"
)

func (c *ControllerV1) Update{{.ShortName}}(ctx context.Context, req *v1.Update{{.ShortName}}Req) (res *v1.Update{{.ShortName}}Res, err error) {
	r := ghttp.RequestFromCtx(ctx)
	isJson := r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json")

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
		if isJson {
			response.JsonError(r, err)
			return nil, nil
		}
		return nil, err
	}

	// Jika client meminta JSON (misal dari Flutter)
	if isJson {
		response.JsonSuccess(r)
		return nil, nil
	}

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
	"{{.ModuleName}}/utility/response"
)

func (c *ControllerV1) Delete{{.ShortName}}(ctx context.Context, req *v1.Delete{{.ShortName}}Req) (res *v1.Delete{{.ShortName}}Res, err error) {
	r := ghttp.RequestFromCtx(ctx)
	isJson := r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json")

	err = service.{{.StructName}}().Delete(ctx, req.Id)
	if err != nil {
		if isJson {
			response.JsonError(r, err)
			return nil, nil
		}
		return nil, err
	}

	// Jika client meminta JSON (misal dari Flutter)
	if isJson {
		response.JsonSuccess(r)
		return nil, nil
	}

	r.Response.RedirectTo("/{{.TableName}}")
	r.Exit()
	return nil, nil
}
`))

var ctrlBatchDeleteTemplate = template.Must(template.New("ctrl_batch_delete").Parse(`package {{.TableName}}

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"

	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
	"{{.ModuleName}}/internal/service"
	"{{.ModuleName}}/utility/response"
)

func (c *ControllerV1) BatchDelete{{.ShortName}}(ctx context.Context, req *v1.BatchDelete{{.ShortName}}Req) (res *v1.BatchDelete{{.ShortName}}Res, err error) {
	r := ghttp.RequestFromCtx(ctx)
	isJson := r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json")

	count, err := service.{{.StructName}}().BatchDelete(ctx, req.Ids)
	if err != nil {
		if isJson {
			response.JsonError(r, err)
			return nil, nil
		}
		return nil, err
	}

	if isJson {
		response.JsonSuccess(r, &v1.BatchDelete{{.ShortName}}Res{Count: count})
		return nil, nil
	}

	r.Response.RedirectTo("/{{.TableName}}")
	r.Exit()
	return nil, nil
}
`))

var cmdTemplate = template.Must(template.New("cmd").Funcs(template.FuncMap{
	"abbrev": abbrev,
}).Parse(`package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/util/gconv"

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
				group.Middleware(func(r *ghttp.Request) {
					r.Middleware.Next()
					if err := r.GetError(); err != nil {
						if r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json" {
							r.Response.ClearBuffer()
							r.Response.WriteStatus(400)
							r.Response.WriteJson(g.Map{
								"code":    1,
								"message": err.Error(),
							})
						}
					}
				})
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
						},
						"FooterNavItem": g.Map{"Name": "Query Console", "TableName": "query", "Active": false, "Abbrev": "QC"},
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
						},
						"FooterNavItem": g.Map{"Name": "Query Console", "TableName": "query", "Active": true, "Abbrev": "QC"},
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
						masterDb, err := g.DB().GetCore().Master()
						if err != nil {
							errStr = err.Error()
						} else {
							dbRows, err := masterDb.QueryContext(r.Context(), sqlStr)
							if err != nil {
								errStr = err.Error()
							} else {
								defer dbRows.Close()
								columns, err = dbRows.Columns()
								if err != nil {
									errStr = err.Error()
								} else {
									for dbRows.Next() {
										values := make([]interface{}, len(columns))
										valuePtrs := make([]interface{}, len(columns))
										for i := range values {
											valuePtrs[i] = &values[i]
										}
										if err := dbRows.Scan(valuePtrs...); err != nil {
											errStr = err.Error()
											break
										}
										rowMap := make(g.Map)
										for i, colName := range columns {
											val := values[i]
											if val == nil {
												rowMap[colName] = "NULL"
											} else {
												switch v := val.(type) {
												case []byte:
													rowMap[colName] = string(v)
												default:
													rowMap[colName] = gconv.String(v)
												}
											}
										}
										rows = append(rows, rowMap)
									}
								}
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
						},
					})
					r.Exit()
				})
			})

			s.Run()
			return nil
		},
	}
)
`))

var indexHTMLTemplate = template.Must(template.New("index_html").Funcs(template.FuncMap{
	"abbrev": abbrev,
}).Parse(`<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Admin Dashboard</title>
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
        
        <!-- Sidebar Navigation -->
        {{"{{"}}include "public/sidebar.html" .{{"}}"}}

        <!-- Content Area -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-12 max-w-5xl flex flex-col justify-center min-h-[80vh]">
                <div class="text-center max-w-2xl mx-auto space-y-6">
                    <div class="inline-flex p-4 rounded-3xl bg-indigo-50 text-indigo-600 shadow-inner">
                        <svg class="h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                        </svg>
                    </div>
                    <div class="space-y-2">
                        <h1 class="text-4xl font-black text-slate-900 tracking-tight sm:text-5xl">Selamat Datang</h1>
                        <p class="text-base text-slate-500 leading-relaxed">Pilih salah satu menu modul data di sidebar sebelah kiri untuk mulai mengelola database Anda dengan cepat dan mudah.</p>
                    </div>
                    <div class="pt-4 flex flex-wrap justify-center gap-3">
                        {{range .NavItems}}
                        <a href="/{{.TableName}}" class="px-5 py-3 bg-white border border-slate-200 hover:border-indigo-500 hover:shadow-lg hover:shadow-indigo-500/5 text-slate-700 font-semibold text-sm rounded-2xl transition-all flex items-center gap-2">
                            <span class="w-6 h-6 rounded-lg bg-indigo-50 text-indigo-600 flex items-center justify-center text-xs font-bold">{{.Abbrev}}</span>
                            {{.Name}}
                        </a>
                        {{end}}
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
    </script>
</body>
</html>
`))

var queryHTMLTemplate = template.Must(template.New("query_html").Funcs(template.FuncMap{
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
        
        <!-- Sidebar Navigation -->
        {{"{{"}}include "public/sidebar.html" .{{"}}"}}

        <!-- Content Area -->
        <div class="flex-1 flex flex-col overflow-y-auto">
            <main class="w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 max-w-7xl">
                <div class="mb-6">
                    <h1 class="text-2xl font-bold text-slate-900">Query Console</h1>
                    <p class="text-sm text-slate-500 mt-0.5">Jalankan SQL query langsung ke database aktif.</p>
                </div>

                <!-- Input Console -->
                <div class="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 mb-6">
                    <form method="POST" action="/query" class="space-y-4">
                        <div class="space-y-1.5">
                            <label class="block text-sm font-semibold text-slate-700">SQL Query</label>
                            <textarea 
                                name="sql" 
                                rows="6" 
                                class="w-full px-4 py-3 border border-slate-200 rounded-xl focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all font-mono text-sm resize-y" 
                                placeholder="SELECT * FROM users LIMIT 10;"
                            >{{"{{"}} .Sql {{"}}"}}</textarea>
                        </div>
                        <div class="flex items-center gap-3">
                            <button type="submit" class="px-6 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl shadow-md shadow-indigo-600/10 transition-all">
                                Jalankan Query
                            </button>
                            <a href="/query" class="px-6 py-2.5 bg-slate-100 hover:bg-slate-200 text-slate-600 font-semibold text-sm rounded-xl transition-all">
                                Reset
                            </a>
                        </div>
                    </form>
                </div>

                <!-- Hasil Section -->
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
                        <p class="text-xs mt-1">Perintah SQL telah berhasil dieksekusi.</p>
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
    </script>
</body>
</html>
`))

var sidebarHTMLTemplate = template.Must(template.New("sidebar_html").Funcs(template.FuncMap{
	"abbrev": abbrev,
}).Parse(`<!-- Sidebar Navigation (Desktop) -->
        <aside id="sidebar" class="hidden md:flex md:flex-col md:flex-shrink-0 w-18 bg-slate-900 text-slate-300 border-r border-slate-800 transition-all duration-300 relative">
            <script>
                if (localStorage.getItem('sidebar-collapsed') === 'false') {
                    const sb = document.getElementById('sidebar');
                    sb.classList.remove('w-18');
                    sb.classList.add('w-64');
                }
            </script>
            <div class="logo-container flex items-center justify-between h-16 px-4 bg-slate-950 border-b border-slate-800/50">
                <div class="sidebar-text flex items-center space-x-3 overflow-hidden">
                    <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-500 to-violet-500 flex items-center justify-center text-white font-bold text-base shadow-sm">
                        A
                    </div>
                    <span class="text-white font-bold text-lg tracking-tight whitespace-nowrap">Admin Console</span>
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
            <div class="flex-1 flex flex-col overflow-y-auto px-3 py-6 space-y-1.5" id="sidebar-nav-items">
                <span class="sidebar-text px-3 text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 whitespace-nowrap">Modul Data</span>
                {{- range .}}
                <a href="/{{.TableName}}" data-table="{{.TableName}}" class="nav-item flex items-center px-3 py-2.5 text-sm font-medium rounded-xl transition-all group text-slate-400 hover:bg-slate-800 hover:text-slate-200">
                    <span class="nav-item-icon flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold mr-3 transition-all bg-slate-800 text-slate-400 group-hover:bg-slate-700 group-hover:text-slate-200">
                        {{abbrev .ShortName}}
                    </span>
                    <span class="sidebar-text whitespace-nowrap">{{.ShortName}}</span>
                </a>
                {{- end}}
            </div>
            <div class="footer-container border-t border-slate-800 p-3">
                <a href="/query" class="nav-item flex items-center px-3 py-2.5 text-sm font-medium rounded-xl transition-all group text-slate-400 hover:bg-slate-800 hover:text-slate-200" title="Query Console">
                    <span class="nav-item-icon flex-shrink-0 w-7 h-7 rounded-lg flex items-center justify-center text-xs font-bold mr-3 transition-all bg-slate-800 text-slate-400 group-hover:bg-slate-700 group-hover:text-slate-200">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                        </svg>
                    </span>
                    <span class="sidebar-text whitespace-nowrap">Query Console</span>
                </a>
            </div>
            <script>
                (function() {
                    const currentPath = window.location.pathname;
                    document.querySelectorAll('#sidebar-nav-items .nav-item').forEach(item => {
                        const table = item.getAttribute('data-table');
                        if (currentPath === '/' + table || currentPath.startsWith('/' + table + '/')) {
                            item.classList.remove('text-slate-400', 'hover:bg-slate-800', 'hover:text-slate-200');
                            item.classList.add('bg-indigo-600', 'text-white', 'shadow-md', 'shadow-indigo-600/10');
                            const icon = item.querySelector('.nav-item-icon');
                            if (icon) {
                                icon.classList.remove('bg-slate-800', 'text-slate-400');
                                icon.classList.add('bg-indigo-500', 'text-white');
                            }
                        }
                    });
                })();
            </script>
        </aside>`))

var mobileNavHTMLTemplate = template.Must(template.New("mobile_nav_html").Parse(`<!-- Mobile Navigation Tabs -->
                <div class="flex md:hidden overflow-x-auto py-2 mb-6 border-b border-slate-200 space-x-2 scrollbar-none" id="mobile-nav-tabs">
                    {{- range .}}
                    <a href="/{{.TableName}}" data-table="{{.TableName}}" class="mobile-nav-item whitespace-nowrap px-4 py-1.5 text-xs font-semibold rounded-full transition-all bg-slate-100 text-slate-600 hover:bg-slate-200">
                        {{.ShortName}}
                    </a>
                    {{- end}}
                    <script>
                        (function() {
                            const currentPath = window.location.pathname;
                            document.querySelectorAll('#mobile-nav-tabs .mobile-nav-item').forEach(item => {
                                const table = item.getAttribute('data-table');
                                if (currentPath === '/' + table || currentPath.startsWith('/' + table + '/')) {
                                    item.classList.remove('bg-slate-100', 'text-slate-600', 'hover:bg-slate-200');
                                    item.classList.add('bg-indigo-600', 'text-white');
                                }
                            });
                        })();
                    </script>
                </div>`))
