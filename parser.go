package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

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

// parseEntityFile parses one entity .go file and returns a TableInfo.
func parseEntityFile(filePath, moduleName string) (*TableInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	// ast.Print(fset, f)

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

			// Inisialisasi default Primary Key
			info.PKName = "Id"
			info.PKJsonTag = "id"
			info.PKOrmTag = "id"
			info.PKType = "uint64"
			info.IsPKAutoIncrement = true
			for i, fi := range info.Fields {
				lowName := strings.ToLower(fi.Name)
				lowOrm := strings.ToLower(fi.OrmTag)
				if fi.IsPK || lowName == "id" || lowOrm == "id" {
					info.Fields[i].IsPK = true
					info.PKName = fi.Name
					info.PKJsonTag = fi.JsonTag
					info.PKOrmTag = fi.OrmTag
					info.PKType = fi.Type
					info.IsPKAutoIncrement = strings.Contains(fi.Type, "int")
					break
				}
			}

			// Reconstruct FormFields based on PK auto-increment status
			info.FormFields = nil
			for _, fi := range info.Fields {
				if fi.IsAudit {
					continue
				}
				if fi.IsPK && info.IsPKAutoIncrement {
					continue
				}
				if fi.IsSkip {
					continue
				}
				info.FormFields = append(info.FormFields, fi)
			}
			break
		}
	}

	if info != nil && info.PKType == "" {
		info.PKName = "Id"
		info.PKJsonTag = "id"
		info.PKOrmTag = "id"
		info.PKType = "uint64"
		info.IsPKAutoIncrement = true
	}

	return info, nil
}

// enrichTableFields updates table field metadata using database column information.
func enrichTableFields(info *TableInfo, dbCols map[string]DBColMeta) {
	if len(dbCols) == 0 {
		return
	}
	enrich := func(fields []FieldInfo) []FieldInfo {
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
				fields[i].IsRequired = meta.IsRequired
				fields[i].IsFK = meta.IsFK
				if meta.IsFK {
					info.HasFK = true
				}
				fields[i].FKTable = meta.FKTable
				fields[i].FKStructName = snakeToTitle(meta.FKTable)
				fields[i].IsSearchable = (meta.DataType == "string" || meta.DataType == "text")
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
			if !fields[i].IsPK && (lowName == "id" || lowOrm == "id" || (strings.HasSuffix(lowOrm, "_id") && len(lowOrm) == 3)) {
				fields[i].IsPK = true
			}
			if fields[i].IsPK {
				fields[i].IsSearchable = false
			}
		}
		return fields
	}

	info.Fields = enrich(info.Fields)
	info.ListFields = enrich(info.ListFields)

	// Update PK metadata dari hasil enrichment
	for _, fi := range info.Fields {
		if fi.IsPK {
			info.PKName = fi.Name
			info.PKJsonTag = fi.JsonTag
			info.PKOrmTag = fi.OrmTag
			info.PKType = fi.Type
			if meta, ok := dbCols[fi.OrmTag]; ok {
				info.IsPKAutoIncrement = meta.IsAutoIncrement
			} else {
				info.IsPKAutoIncrement = strings.Contains(fi.Type, "int")
			}
			break
		}
	}
	if info.PKType == "" {
		info.PKName = "Id"
		info.PKJsonTag = "id"
		info.PKOrmTag = "id"
		info.PKType = "uint64"
		info.IsPKAutoIncrement = true
	}

	// Reconstruct FormFields based on enriched PK auto-increment status
	var newFormFields []FieldInfo
	for _, fi := range info.Fields {
		if fi.IsAudit {
			continue
		}
		if fi.IsPK && info.IsPKAutoIncrement {
			continue
		}
		if fi.IsSkip {
			continue
		}
		newFormFields = append(newFormFields, fi)
	}
	info.FormFields = newFormFields
}

// enrichTableRules applies rules from rules.json to table fields.
func enrichTableRules(info *TableInfo, rulesMap map[string]map[string]interface{}) {
	if rulesMap == nil {
		return
	}
	enrich := func(fields []FieldInfo) []FieldInfo {
		for i, fi := range fields {
			rule, exists := rulesMap[fi.JsonTag]
			if !exists {
				rule, exists = rulesMap[strings.ToLower(fi.Name)]
			}
			if exists {
				fields[i].Rules = make(map[string]interface{})
				for k, v := range rule {
					switch k {
					case "type":
						if typeStr, ok := v.(string); ok {
							fields[i].DataType = typeStr
							if typeStr == "range" {
								fields[i].HTMLType = "range"
							}
						}
					case "required":
						if reqBool, ok := v.(bool); ok {
							fields[i].IsRequired = reqBool
						}
					default:
						fields[i].Rules[k] = v
					}
				}
			}
		}
		return fields
	}

	info.Fields = enrich(info.Fields)
	info.ListFields = enrich(info.ListFields)
	info.FormFields = enrich(info.FormFields)
}
