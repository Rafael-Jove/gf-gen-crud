package main

import (
	"strings"
	"text/template"
)

var apiTemplate = template.Must(template.New("api").Funcs(template.FuncMap{
	"contains": strings.Contains,
}).Parse(`package v1

import (
	"encoding/json"

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

// ---------- Data Item Model (with correct JSON types) ----------

type {{.ShortName}}Field struct {
	Value  interface{} ` + "`" + `json:"value"` + "`" + `
	Type   string      ` + "`" + `json:"type"` + "`" + `
	Values []string    ` + "`" + `json:"values,omitempty"` + "`" + `
	Extra  map[string]interface{}` + "`" + `json:"meta,omitempty"` + "`" + `
}

func (f {{.ShortName}}Field) MarshalJSON() ([]byte, error) {
	result := map[string]interface{}{
		"value": f.Value,
		"type":  f.Type,
	}
	if len(f.Values) > 0 {
		result["values"] = f.Values
	}
	for k, v := range f.Extra {
		result[k] = v
	}
	return json.Marshal(result)
}

type {{.ShortName}}Item struct {
{{- range .Fields}}
	{{.Name}} {{$.ShortName}}Field ` + "`" + `json:"{{.JsonTag}}"` + "`" + `
{{- end}}
}

// ---------- List ----------

type List{{.ShortName}}Req struct {
	g.Meta   ` + "`" + `path:"/{{.TableName}}" method:"get" tags:"{{.StructName}}" summary:"Daftar {{.ShortName}}"` + "`" + `
	Page     int ` + "`" + `json:"page" d:"1"` + "`" + `
	PageSize int ` + "`" + `json:"page_size" d:"10"` + "`" + `
	Keyword  string ` + "`" + `json:"keyword"` + "`" + `
}

type List{{.ShortName}}Res struct {
	List  []{{.ShortName}}Item ` + "`" + `json:"list"` + "`" + `
	Total int                  ` + "`" + `json:"total"` + "`" + `
	Page  int                  ` + "`" + `json:"page"` + "`" + `
}

// ---------- Filter ----------

type Filter{{.ShortName}}Req struct {
	g.Meta   ` + "`" + `path:"/{{.TableName}}/filter" method:"get" tags:"{{.StructName}}" summary:"Halaman Filter {{.ShortName}}"` + "`" + `
	Page     int ` + "`" + `json:"page" d:"1"` + "`" + `
	PageSize int ` + "`" + `json:"page_size" d:"10"` + "`" + `
	Keyword  string ` + "`" + `json:"keyword"` + "`" + `
}

type Filter{{.ShortName}}Res struct {
	List  []{{.ShortName}}Item ` + "`" + `json:"list"` + "`" + `
	Total int                  ` + "`" + `json:"total"` + "`" + `
	Page  int                  ` + "`" + `json:"page"` + "`" + `
}

// ---------- Get ----------

type Get{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}" method:"get" tags:"{{.StructName}}" summary:"Detail {{.ShortName}}"` + "`" + `
	Id     uint64 ` + "`" + `json:"id" v:"required#ID wajib diisi"` + "`" + `
}

type Get{{.ShortName}}Res struct {
	Data *{{.ShortName}}Item ` + "`" + `json:"data"` + "`" + `
}

type Create{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}" method:"post" tags:"{{.StructName}}" summary:"Buat {{.ShortName}} baru"` + "`" + `
{{- range .FormFields}}
	{{- if eq .HTMLType "file"}}
	{{.Name}} *ghttp.UploadFile ` + "`" + `json:"{{.JsonTag}}" type:"file" v:"required#{{.Name}} wajib diisi"` + "`" + `
	{{- else}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JsonTag}}" v:"required#{{.Name}} wajib diisi"` + "`" + `
	{{- end}}
{{- end}}
}

type Create{{.ShortName}}Res struct {
	Data *{{.ShortName}}Item ` + "`" + `json:"data"` + "`" + `
}

// ---------- Update ----------

type Update{{.ShortName}}Req struct {
	g.Meta ` + "`" + `path:"/{{.TableName}}/{id}" method:"post" tags:"{{.StructName}}" summary:"Update {{.ShortName}}"` + "`" + `
	Id     uint64 ` + "`" + `json:"id" v:"required#ID wajib diisi"` + "`" + `
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
	Filter{{.ShortName}}(ctx context.Context, req *v1.Filter{{.ShortName}}Req) (res *v1.Filter{{.ShortName}}Res, err error)
	Get{{.ShortName}}(ctx context.Context, req *v1.Get{{.ShortName}}Req) (res *v1.Get{{.ShortName}}Res, err error)
	Create{{.ShortName}}(ctx context.Context, req *v1.Create{{.ShortName}}Req) (res *v1.Create{{.ShortName}}Res, err error)
	Update{{.ShortName}}(ctx context.Context, req *v1.Update{{.ShortName}}Req) (res *v1.Update{{.ShortName}}Res, err error)
	Delete{{.ShortName}}(ctx context.Context, req *v1.Delete{{.ShortName}}Req) (res *v1.Delete{{.ShortName}}Res, err error)
}
`))

var logicTemplate = template.Must(template.New("logic").Parse(`package {{.VarName}}

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
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

// List mengambil daftar {{.ShortName}} dengan pagination, keyword filter, dan form filters.
func (s *s{{.StructName}}) List(ctx context.Context, page, pageSize int, keyword string) (list []*entity.{{.StructName}}, total int, err error) {
	m := dao.{{.StructName}}.Ctx(ctx)

	// Apply form filters starting with f_ from HTTP request
	r := ghttp.RequestFromCtx(ctx)
	if r != nil {
		{{- range .FormFields}}
		{{- if .EnumValues}}
		if val := r.Get("f_{{.JsonTag}}").String(); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + `", val)
		}
		{{- else if eq .HTMLType "checkbox"}}
		if val := r.Get("f_{{.JsonTag}}").String(); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + `", val)
		}
		{{- else if eq .HTMLType "number"}}
		if valMin := r.Get("f_{{.JsonTag}}_min").String(); valMin != "" {
			m = m.WhereGTE("` + "`" + `{{.OrmTag}}` + "`" + `", valMin)
		}
		if valMax := r.Get("f_{{.JsonTag}}_max").String(); valMax != "" {
			m = m.WhereLTE("` + "`" + `{{.OrmTag}}` + "`" + `", valMax)
		}
		{{- else if or (eq .HTMLType "date") (eq .HTMLType "datetime-local") (eq .HTMLType "time")}}
		if valFrom := r.Get("f_{{.JsonTag}}_from").String(); valFrom != "" {
			m = m.WhereGTE("` + "`" + `{{.OrmTag}}` + "`" + `", valFrom)
		}
		if valTo := r.Get("f_{{.JsonTag}}_to").String(); valTo != "" {
			m = m.WhereLTE("` + "`" + `{{.OrmTag}}` + "`" + `", valTo)
		}
		{{- else if eq .Type "string"}}
		if val := r.Get("f_{{.JsonTag}}").String(); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + ` LIKE ?", "%"+val+"%")
		}
		{{- else}}
		if val := r.Get("f_{{.JsonTag}}").String(); val != "" {
			m = m.Where("` + "`" + `{{.OrmTag}}` + "`" + `", val)
		}
		{{- end}}
		{{- end}}
	}

	if keyword != "" {
		var conds []string
		var args []interface{}
		{{- range .ListFields}}
		conds = append(conds, "` + "`" + `{{.OrmTag}}` + "`" + ` LIKE ?")
		args = append(args, "%"+keyword+"%")
		{{- end}}
		if len(conds) > 0 {
			m = m.Where(strings.Join(conds, " OR "), args...)
		}
	}
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
                                    <td class="sticky right-0 bg-white group-hover:bg-slate-50/50 px-6 py-4 text-sm text-slate-700 text-right whitespace-nowrap shadow-[-4px_0_8px_rgba(0,0,0,0.05)] z-10">
                                        <div class="flex items-center justify-end space-x-1">
                                            <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}" class="inline-flex items-center justify-center p-1.5 rounded-lg text-emerald-600 hover:bg-emerald-50 hover:text-emerald-950 transition-colors" title="Show">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                                </svg>
                                            </a>
                                            {{- if not .IsReadOnly}}
                                            <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}/edit" class="inline-flex items-center justify-center p-1.5 rounded-lg text-indigo-600 hover:bg-indigo-50 hover:text-indigo-900 transition-colors" title="Update">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                </svg>
                                            </a>
                                            <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}/delete" onclick="return confirm('Hapus data ini?')" class="inline-flex items-center justify-center p-1.5 rounded-lg text-rose-600 hover:bg-rose-50 hover:text-rose-900 transition-colors" title="Delete">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                </svg>
                                            </a>
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
            } catch (err) {
                console.error('AJAX search failed:', err);
            }
        }

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
                                {{- else if eq .HTMLType "range"}}
                                <div class="flex items-center gap-3 pt-2">
                                    <input type="range" name="{{.JsonTag}}" value="{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}} {{"}}"}}{{"{{"}} else {{"}}"}}{{if .Rules.min}}{{.Rules.min}}{{else}}0{{end}}{{"{{"}} end {{"}}"}}" min="{{if .Rules.min}}{{.Rules.min}}{{else}}0{{end}}" max="{{if .Rules.max}}{{.Rules.max}}{{else}}100{{end}}" step="{{if .Rules.step}}{{.Rules.step}}{{else}}1{{end}}" class="w-full h-2 bg-slate-200 rounded-lg appearance-none cursor-pointer accent-indigo-600 focus:outline-none" oninput="this.nextElementSibling.innerText = this.value" />
                                    <span class="text-sm font-semibold text-slate-600 min-w-[2.5rem] text-right">{{"{{"}} if .{{.Name}} {{"}}"}}{{"{{"}} .{{.Name}} {{"}}"}}{{"{{"}} else {{"}}"}}{{if .Rules.min}}{{.Rules.min}}{{else}}0{{end}}{{"{{"}} end {{"}}"}}</span>
                                </div>
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

        // Intercept form submit to convert raw "HH:MM" time values to full "YYYY-MM-DD HH:MM:SS"
        // so that GoFrame's gtime.Time parser can bind them successfully instead of falling back to nil/null.
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
                                            <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}" class="inline-flex items-center justify-center p-1.5 rounded-lg text-emerald-600 hover:bg-emerald-50 hover:text-emerald-950 transition-colors" title="Show">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                                                </svg>
                                            </a>
                                            {{- if not .IsReadOnly}}
                                            <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}/edit" class="inline-flex items-center justify-center p-1.5 rounded-lg text-indigo-600 hover:bg-indigo-50 hover:text-indigo-900 transition-colors" title="Update">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M11 5H6a2 2 0 00-2 2v12a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                                </svg>
                                            </a>
                                            <a href="/{{$.TableName}}/{{"{{"}} .Id {{"}}"}}/delete" onclick="return confirm('Hapus data ini?')" class="inline-flex items-center justify-center p-1.5 rounded-lg text-rose-600 hover:bg-rose-50 hover:text-rose-900 transition-colors" title="Delete">
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-4v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                                </svg>
                                            </a>
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

	"{{.ModuleName}}/internal/service"
)

// ShowCreateForm menampilkan form tambah data baru.
func ShowCreateForm(r *ghttp.Request) {
	r.Response.WriteTpl("{{.TableName}}/form.html", map[string]interface{}{})
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

	// Buat map explicit agar GoFrame view engine lancar me-render PascalCase struct fields
	formValues := map[string]interface{}{
		"Id": data.Id,
		{{- range .FormFields}}
		"{{.Name}}": data.{{.Name}},
		{{- end}}
	}

	r.Response.WriteTpl("{{.TableName}}/form.html", formValues)
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
{{- $hasGconv := false}}
{{- range .Fields}}
{{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}{{$hasGconv = true}}{{end}}
{{- end}}
{{- if $hasGconv}}
	"github.com/gogf/gf/v2/util/gconv"
{{- end}}
{{- if .HasGtime}}
 	"github.com/gogf/gf/v2/os/gtime"
{{- end}}
 
 	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
 	"{{.ModuleName}}/internal/service"
 )

func (c *ControllerV1) List{{.ShortName}}(ctx context.Context, req *v1.List{{.ShortName}}Req) (res *v1.List{{.ShortName}}Res, err error) {
	list, total, err := service.{{.StructName}}().List(ctx, req.Page, req.PageSize, req.Keyword)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	// Jika client meminta JSON (misal dari Flutter)
	if r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json") {
		var listItems []v1.{{.ShortName}}Item
		for _, item := range list {
			listItems = append(listItems, v1.{{.ShortName}}Item{
				{{- range .Fields}}
				{{.Name}}: v1.{{$.ShortName}}Field{
                    Type: "{{.DataType}}",
                    {{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}
                    Value: gconv.String(item.{{.Name}}),
                    {{- else if eq .HTMLType "date"}}
                    Value: func() interface{} {
                        if g.IsEmpty(item.{{.Name}}) {
                            return nil
                        }
                        return gtime.New(item.{{.Name}}).Layout("2006-01-02")
                    }(),
                    {{- else if eq .HTMLType "datetime-local"}}
                    Value: func() interface{} {
                        if g.IsEmpty(item.{{.Name}}) {
                            return nil
                        }
                        return gtime.New(item.{{.Name}}).Layout("2006-01-02 15:04:05")
                    }(),
                    {{- else if eq .HTMLType "time"}}
                    Value: func() interface{} {
                        if g.IsEmpty(item.{{.Name}}) {
                            return nil
                        }
                        return gtime.New(item.{{.Name}}).Layout("15:04:05")
                    }(),
                    {{- else}}
                    Value: item.{{.Name}},
                    {{- end}}
                    {{- if .EnumValues}}
                    Values: []string{
                        {{- range .EnumValues}}
                        "{{.}}",
                        {{- end}}
                    },
                    {{- end}}
                    {{- if .Rules}}
                    Extra: map[string]interface{}{
                        {{- range $k, $v := .Rules}}
                        "{{$k}}": {{$v}},
                        {{- end}}
                },
                {{- end}}
            },
				{{- end}}
			})
		}
		r.Response.WriteJson(g.Map{
			"code":    0,
			"message": "success",
			"data": g.Map{
				"list":  listItems,
				"total": total,
				"page":  req.Page,
			},
		})
		r.Exit()
		return nil, nil
	}

	startIndex := (req.Page-1)*req.PageSize + 1
	if total == 0 {
		startIndex = 0
	}
	endIndex := req.Page * req.PageSize
	if endIndex > total {
		endIndex = total
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	r.Response.WriteTpl("{{.TableName}}/list.html", g.Map{
		"List":       list,
		"Total":      total,
		"Page":       req.Page,
		"PageSize":   req.PageSize,
		"Keyword":    req.Keyword,
		"StartIndex": startIndex,
		"EndIndex":   endIndex,
		"PrevPage":   req.Page - 1,
		"NextPage":   req.Page + 1,
		"HasPrev":    req.Page > 1,
		"HasNext":    req.Page*req.PageSize < total,
		"TotalPages": totalPages,
	})
	r.Exit()
	return nil, nil
}

func (c *ControllerV1) Filter{{.ShortName}}(ctx context.Context, req *v1.Filter{{.ShortName}}Req) (res *v1.Filter{{.ShortName}}Res, err error) {
	list, total, err := service.{{.StructName}}().List(ctx, req.Page, req.PageSize, req.Keyword)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	// Jika client meminta JSON (misal dari Flutter)
	if r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json") {
		var listItems []v1.{{.ShortName}}Item
		for _, item := range list {
			listItems = append(listItems, v1.{{.ShortName}}Item{
				{{- range .Fields}}
				{{.Name}}: v1.{{$.ShortName}}Field{
					Type: "{{.DataType}}",
					{{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}
					Value: gconv.String(item.{{.Name}}),
					{{- else if eq .HTMLType "date"}}
					Value: func() interface{} {
						if g.IsEmpty(item.{{.Name}}) {
							return nil
						}
						return gtime.New(item.{{.Name}}).Layout("2006-01-02")
					}(),
					{{- else if eq .HTMLType "datetime-local"}}
					Value: func() interface{} {
						if g.IsEmpty(item.{{.Name}}) {
							return nil
						}
						return gtime.New(item.{{.Name}}).Layout("2006-01-02 15:04:05")
					}(),
					{{- else if eq .HTMLType "time"}}
					Value: func() interface{} {
						if g.IsEmpty(item.{{.Name}}) {
							return nil
						}
						return gtime.New(item.{{.Name}}).Layout("15:04:05")
					}(),
					{{- else}}
					Value: item.{{.Name}},
					{{- end}}
					{{- if .EnumValues}}
					Values: []string{
						{{- range .EnumValues}}
						"{{.}}",
						{{- end}}
					},
					{{- end}}
                    {{- if .Rules}}
					Extra: map[string]interface{}{
						{{- range $k, $v := .Rules}}
						"{{$k}}": {{$v}},
						{{- end}}
					},
					{{- end}}
				},
				{{- end}}
			})
		}
		r.Response.WriteJson(g.Map{
			"code":    0,
			"message": "success",
			"data": g.Map{
				"list":  listItems,
				"total": total,
				"page":  req.Page,
			},
		})
		r.Exit()
		return nil, nil
	}

	startIndex := (req.Page-1)*req.PageSize + 1
	if total == 0 {
		startIndex = 0
	}
	endIndex := req.Page * req.PageSize
	if endIndex > total {
		endIndex = total
	}

	totalPages := (total + req.PageSize - 1) / req.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	tplData := g.Map{
		"List":       list,
		"Total":      total,
		"Page":       req.Page,
		"PageSize":   req.PageSize,
		"Keyword":    req.Keyword,
		"StartIndex": startIndex,
		"EndIndex":   endIndex,
		"PrevPage":   req.Page - 1,
		"NextPage":   req.Page + 1,
		"HasPrev":    req.Page > 1,
		"HasNext":    req.Page*req.PageSize < total,
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
{{- $hasGconv := false}}
{{- range .Fields}}
{{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}{{$hasGconv = true}}{{end}}
{{- end}}
{{- if $hasGconv}}
	"github.com/gogf/gf/v2/util/gconv"
{{- end}}
{{- if .HasGtime}}
 	"github.com/gogf/gf/v2/os/gtime"
{{- end}}
 
 	v1 "{{.ModuleName}}/api/{{.TableName}}/v1"
 	"{{.ModuleName}}/internal/service"
 )

func (c *ControllerV1) Get{{.ShortName}}(ctx context.Context, req *v1.Get{{.ShortName}}Req) (res *v1.Get{{.ShortName}}Res, err error) {
	data, err := service.{{.StructName}}().Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	// Jika client meminta JSON (misal dari Flutter)
	if r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json") {
		item := v1.{{.ShortName}}Item{
			{{- range .Fields}}
			{{.Name}}: v1.{{$.ShortName}}Field{
				Type: "{{.DataType}}",
				{{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}
				Value: gconv.String(data.{{.Name}}),
				{{- else if eq .HTMLType "date"}}
				Value: func() interface{} {
					if g.IsEmpty(data.{{.Name}}) {
						return nil
					}
					return gtime.New(data.{{.Name}}).Layout("2006-01-02")
				}(),
				{{- else if eq .HTMLType "datetime-local"}}
				Value: func() interface{} {
					if g.IsEmpty(data.{{.Name}}) {
						return nil
					}
					return gtime.New(data.{{.Name}}).Layout("2006-01-02 15:04:05")
				}(),
				{{- else if eq .HTMLType "time"}}
				Value: func() interface{} {
					if g.IsEmpty(data.{{.Name}}) {
						return nil
					}
					return gtime.New(data.{{.Name}}).Layout("15:04:05")
				}(),
				{{- else}}
				Value: data.{{.Name}},
				{{- end}}
				{{- if .EnumValues}}
				Values: []string{
					{{- range .EnumValues}}
					"{{.}}",
					{{- end}}
				},
				{{- end}}
                {{- if .Rules}}
					Extra: map[string]interface{}{
						{{- range $k, $v := .Rules}}
						"{{$k}}": {{$v}},
						{{- end}}
					},
				{{- end}}
			},
			{{- end}}
		}
		r.Response.WriteJson(g.Map{
			"code":    0,
			"message": "success",
			"data":    item,
		})
		r.Exit()
		return nil, nil
	}

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
{{- $hasGconv := false}}
{{- range .Fields}}
{{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}{{$hasGconv = true}}{{end}}
{{- end}}
{{- if $hasGconv}}
	"github.com/gogf/gf/v2/util/gconv"
{{- end}}
{{- if .HasUpload}}
 	"io"
{{- end}}
 
 	"github.com/gogf/gf/v2/frame/g"
 	"github.com/gogf/gf/v2/net/ghttp"
{{- if .HasGtime}}
 	"github.com/gogf/gf/v2/os/gtime"
{{- end}}
 
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

	data, err := service.{{.StructName}}().Create(ctx, createData)
	if err != nil {
		return nil, err
	}

	r := ghttp.RequestFromCtx(ctx)
	// Jika client meminta JSON (misal dari Flutter)
	if r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json") {
		item := v1.{{.ShortName}}Item{
			{{- range .Fields}}
			{{.Name}}: v1.{{$.ShortName}}Field{
				Type: "{{.DataType}}",
				{{- if or (eq .Type "int64") (eq .Type "uint64") (eq .Type "*int64") (eq .Type "*uint64")}}
				Value: gconv.String(data.{{.Name}}),
				{{- else if eq .HTMLType "date"}}
				Value: func() interface{} {
					if g.IsEmpty(data.{{.Name}}) {
						return nil
					}
					return gtime.New(data.{{.Name}}).Layout("2006-01-02")
				}(),
				{{- else if eq .HTMLType "datetime-local"}}
				Value: func() interface{} {
					if g.IsEmpty(data.{{.Name}}) {
						return nil
					}
					return gtime.New(data.{{.Name}}).Layout("2006-01-02 15:04:05")
				}(),
				{{- else if eq .HTMLType "time"}}
				Value: func() interface{} {
					if g.IsEmpty(data.{{.Name}}) {
						return nil
					}
					return gtime.New(data.{{.Name}}).Layout("15:04:05")
				}(),
				{{- else}}
				Value: data.{{.Name}},
				{{- end}}
				{{- if .EnumValues}}
				Values: []string{
					{{- range .EnumValues}}
					"{{.}}",
					{{- end}}
				},
				{{- end}}
                {{- if .Rules}}
					Extra: map[string]interface{}{
						{{- range $k, $v := .Rules}}
						"{{$k}}": {{$v}},
						{{- end}}
					},
				{{- end}}
			},
			{{- end}}
		}
		r.Response.WriteJson(g.Map{
			"code":    0,
			"message": "success",
			"data":    item,
		})
		r.Exit()
		return nil, nil
	}

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

	"github.com/gogf/gf/v2/frame/g"
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
	// Jika client meminta JSON (misal dari Flutter)
	if r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json") {
		r.Response.WriteJson(g.Map{
			"code":    0,
			"message": "success",
		})
		r.Exit()
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

	"github.com/gogf/gf/v2/frame/g"
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
	// Jika client meminta JSON (misal dari Flutter)
	if r != nil && (r.Header.Get("Accept") == "application/json" || r.Get("format").String() == "json") {
		r.Response.WriteJson(g.Map{
			"code":    0,
			"message": "success",
		})
		r.Exit()
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

{{range .Controllers}}
{{- if .HasHTML}}
{{- if .HasWrite}}
			s.Group("/{{.TableName}}", func(group *ghttp.RouterGroup) {
				group.GET("/create", {{.PackageName}}.ShowCreateForm)
				group.GET("/{id}/edit", {{.PackageName}}.ShowEditForm)
				group.GET("/{id}/delete", {{.PackageName}}.DeleteAction)
			})
{{- end}}
{{- end}}
{{- end}}

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
