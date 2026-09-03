package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/manifoldco/promptui"
)

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

// generateFile combines renderTemplate and writeFile into a single unified step.
func generateFile(path string, tmpl *template.Template, data interface{}, overwrite bool) error {
	content, err := renderTemplate(tmpl, data)
	if err != nil {
		fmt.Printf("  [ERROR] Gagal render template untuk %s: %v\n", path, err)
		return err
	}
	return writeFile(path, content, overwrite)
}

func setupDatabaseDriverAndGenBat(root string) {
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

	var driverType string
	if !hasDriver {
		prompt := promptui.Select{
			Label: "Pilih driver database yang ingin Anda gunakan di project ini",
			Items: []string{
				"mysql  - MySQL / MariaDB",
				"pgsql  - PostgreSQL",
				"sqlite - SQLite",
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
		default:
			fmt.Println("Melewati konfigurasi driver database.")
		}
		fmt.Println()
	}

	// Auto-import driver to main.go and run go get
	if driverType != "" {
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
gf-gen-crud %*

echo === Running gf gen service ===
gf gen service

echo === Done! ===
`
		_ = writeFile(genBatPath, genBatContent, false)
	}

}

func configureTableViews(targetTables []*TableInfo, genConfig *GenConfig, reconfigure bool) {
	var needsPrompt []string
	for _, info := range targetTables {
		_, known := genConfig.Tables[info.TableName]
		if !known || reconfigure {
			needsPrompt = append(needsPrompt, info.TableName)
		}
	}

	if len(needsPrompt) > 0 {
		var tc TableGenConfig
		fmt.Printf("Tabel berikut belum dikonfigurasi: %s\n", strings.Join(needsPrompt, ", "))
		tc = promptViewAndFilter("SEMUA tabel di atas")

		for _, tn := range needsPrompt {
			genConfig.Tables[tn] = tc
		}
	}
}

func generateModel(root string, info *TableInfo, overwrite bool) {
	labelField := info.PKName
	for i, f := range info.Fields {
		if f.IsPK {
			if i+1 < len(info.Fields) {
				labelField = info.Fields[i+1].Name
				break
			}
		}
	}
	if labelField == "" && len(info.Fields) > 0 {
		labelField = info.Fields[0].Name
	}
	info.LabelFieldName = labelField

	modelDir := filepath.Join(root, "internal", "model")
	_ = os.MkdirAll(modelDir, 0755)

	// Clean up legacy singular/plural duplicate model file if exists
	shortVar := structNameToVar(info.ShortName)
	if shortVar != info.VarName {
		_ = os.Remove(filepath.Join(modelDir, shortVar+".go"))
	}

	_ = generateFile(filepath.Join(modelDir, info.VarName+".go"), modelTemplate, info, overwrite)
}

func generateAPI(root string, info *TableInfo, overwrite bool) {
	_ = generateFile(filepath.Join(root, "api", info.TableName, "v1", info.TableName+".go"), apiTemplate, info, overwrite)
	_ = generateFile(filepath.Join(root, "api", info.TableName, info.TableName+".go"), apiInterfaceTemplate, info, overwrite)
}

func generateLogic(root string, info *TableInfo, overwrite bool) {
	logicDir := filepath.Join(root, "internal", "logic", info.VarName)
	_ = os.Remove(filepath.Join(logicDir, info.VarName+"_gen.go"))
	_ = generateFile(filepath.Join(logicDir, info.VarName+".go"), logicTemplate, info, overwrite)
}

func generateViews(root string, info *TableInfo, overwrite bool) {
	tplDir := filepath.Join(root, "resource", "template", info.TableName)
	_ = generateFile(filepath.Join(tplDir, "list.html"), listHTMLTemplate, info, overwrite)

	if !info.IsReadOnly {
		_ = generateFile(filepath.Join(tplDir, "form.html"), formHTMLTemplate, info, overwrite)
	}
	if info.FilterType == "form" || info.FilterType == "both" {
		_ = generateFile(filepath.Join(tplDir, "filter.html"), filterHTMLTemplate, info, overwrite)
	}
	_ = generateFile(filepath.Join(tplDir, "detail.html"), detailHTMLTemplate, info, overwrite)
}

func generateControllers(root string, info *TableInfo, overwrite bool) {
	ctrlDir := filepath.Join(root, "internal", "controller", info.TableName)
	_ = os.MkdirAll(ctrlDir, 0755)

	shortSnake := structNameToSnake(info.ShortName)
	fileSuffix := "_" + shortSnake
	if info.TableName == shortSnake {
		fileSuffix = ""
	}

	var controllersToGen []struct {
		filename string
		tmpl     *template.Template
	}
	controllersToGen = append(controllersToGen,
		struct {
			filename string
			tmpl     *template.Template
		}{info.TableName + "_new.go", ctrlNewTemplate},
		struct {
			filename string
			tmpl     *template.Template
		}{fmt.Sprintf("%s_v1_list%s.go", info.TableName, fileSuffix), ctrlListTemplate},
		struct {
			filename string
			tmpl     *template.Template
		}{fmt.Sprintf("%s_v1_get%s.go", info.TableName, fileSuffix), ctrlGetTemplate},
	)

	if !info.IsReadOnly {
		controllersToGen = append(controllersToGen,
			struct {
				filename string
				tmpl     *template.Template
			}{info.TableName + "_form.go", ctrlFormTemplate},
			struct {
				filename string
				tmpl     *template.Template
			}{fmt.Sprintf("%s_v1_create%s.go", info.TableName, fileSuffix), ctrlCreateTemplate},
			struct {
				filename string
				tmpl     *template.Template
			}{fmt.Sprintf("%s_v1_update%s.go", info.TableName, fileSuffix), ctrlUpdateTemplate},
			struct {
				filename string
				tmpl     *template.Template
			}{fmt.Sprintf("%s_v1_delete%s.go", info.TableName, fileSuffix), ctrlDeleteTemplate},
			struct {
				filename string
				tmpl     *template.Template
			}{fmt.Sprintf("%s_v1_batch_delete%s.go", info.TableName, fileSuffix), ctrlBatchDeleteTemplate},
		)
	} else {
		_ = os.Remove(filepath.Join(ctrlDir, info.TableName+"_form.go"))
		_ = os.Remove(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_create%s.go", info.TableName, fileSuffix)))
		_ = os.Remove(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_update%s.go", info.TableName, fileSuffix)))
		_ = os.Remove(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_delete%s.go", info.TableName, fileSuffix)))
		_ = os.Remove(filepath.Join(ctrlDir, fmt.Sprintf("%s_v1_batch_delete%s.go", info.TableName, fileSuffix)))
	}
	for _, item := range controllersToGen {
		_ = generateFile(filepath.Join(ctrlDir, item.filename), item.tmpl, info, overwrite)
	}
}

func rebuildCmdFile(root, moduleName string, allEntities []*TableInfo) {
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

	_ = generateFile(filepath.Join(root, "internal", "cmd", "cmd.go"), cmdTemplate, struct {
		ModuleName  string
		Controllers []CmdControllerInfo
	}{
		ModuleName:  moduleName,
		Controllers: controllers,
	}, true)
}

func generatePublicTemplates(root string, allEntities []*TableInfo) {
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
	publicTplDir := filepath.Join(tplDir, "public")
	_ = os.MkdirAll(publicTplDir, 0755)

	_ = generateFile(filepath.Join(publicTplDir, "sidebar.html"), sidebarHTMLTemplate, allEntities, true)
	_ = generateFile(filepath.Join(publicTplDir, "mobile_nav.html"), mobileNavHTMLTemplate, allEntities, true)
	_ = generateFile(filepath.Join(tplDir, "index.html"), indexHTMLTemplate, map[string]interface{}{"NavItems": indexNavItems}, true)
	_ = generateFile(filepath.Join(tplDir, "query.html"), queryHTMLTemplate, map[string]interface{}{"NavItems": indexNavItems}, true)
}
