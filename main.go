// Generator CRUD otomatis untuk GoFrame.
// Membaca semua file entity di internal/model/entity dan men-generate CRUD lengkap.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func main() {
	tablesFlag := flag.String("tables", "", "Generate hanya untuk tabel tertentu, pisahkan dengan koma (e.g. --tables=users,roles)")
	skipView := flag.Bool("skip-view", false, "Jangan generate HTML template")
	skipLogic := flag.Bool("skip-logic", false, "Jangan generate logic file")
	overwrite := flag.Bool("overwrite", false, "Timpa file yang sudah ada")
	reconfigure := flag.Bool("reconfigure", false, "Ubah view-mode & filter-type yang sudah pernah di konfigurasi")
	flag.Parse()

	root, _ := filepath.Abs(".")
	entityDir := filepath.Join(root, "internal", "model", "entity")
	moduleName := readModuleName(root)

	// Step 1: Database driver setup & helper gen.bat initialization
	setupDatabaseDriverAndGenBat(root)

	fmt.Printf("Module  : %s\n", moduleName)
	fmt.Printf("Entities: %s\n\n", entityDir)

	files, err := filepath.Glob(filepath.Join(entityDir, "*.go"))
	if err != nil || len(files) == 0 {
		fmt.Println("Tidak ada file entity ditemukan di", entityDir)
		os.Exit(1)
	}

	// Step 2: Load rules.json jika ada
	var rulesMap map[string]map[string]interface{}
	if rData, err := os.ReadFile(filepath.Join(root, "rules.json")); err == nil {
		_ = json.Unmarshal(rData, &rulesMap)
	}

	// Step 3: Persiapkan filter tabel target
	tableSet := parseTableFlags(*tablesFlag)

	// Step 4: Parse file entity & enrichment metadata DB (dengan early filtering)
	var allEntities []*TableInfo
	var targetTables []*TableInfo

	for _, f := range files {
		info, err := parseEntityFile(f, moduleName)
		if err != nil || info == nil {
			fmt.Printf("Skip %s: %v\n", f, err)
			continue
		}
		allEntities = append(allEntities, info)

		if isTargetTable(info, tableSet) {
			dbCols := fetchDBColumnTypes(root, info.TableName)
			enrichTableFields(info, dbCols)
			enrichTableRules(info, rulesMap)
			targetTables = append(targetTables, info)
		}
	}

	genConfig := loadGenConfig(root)

	// Step 5: Konfigurasi interactive mode (jika belum ada atau --reconfigure)
	if !*skipView {
		if *reconfigure {
			*overwrite = true // Reconfigure pasti membutuhkan overwrite file agar settingan baru diterapkan
		}
		configureTableViews(targetTables, genConfig, *reconfigure)
	}

	// Step 6: Generate response utility
	_ = generateFile(filepath.Join(root, "utility", "response", "response.go"), utilityResponseTemplate, nil, *overwrite)

	// Step 7: Generate kode & template per tabel target
	for _, info := range targetTables {
		info.NavItems = buildNavItems(allEntities, info.TableName)

		if !*skipView {
			tc := genConfig.Tables[info.TableName]
			info.IsReadOnly = tc.ViewMode == "view"
			info.FilterType = tc.FilterType
		}

		fmt.Printf("=== Generating: %s (%s) ===\n", info.StructName, info.TableName)

		generateModel(root, info, *overwrite)

		if !*skipLogic {
			generateAPI(root, info, *overwrite)
			generateLogic(root, info, *overwrite)
			generateControllers(root, info, *overwrite)
		}

		if !*skipView {
			generateViews(root, info, *overwrite)
		}

		fmt.Println()
	}

	// Step 8: Rebuild cmd.go & Public Layout Templates
	if !*skipView {
		generatePublicTemplates(root, allEntities)
	}

	if !*skipLogic {
		rebuildCmdFile(root, moduleName, allEntities)
	}

	saveGenConfig(root, genConfig)

	fmt.Println("=== Updating service interfaces (gf gen service) ===")
	cmdService := exec.Command("gf", "gen", "service")
	cmdService.Dir = root
	cmdService.Stdout = os.Stdout
	cmdService.Stderr = os.Stderr
	_ = cmdService.Run()

	fmt.Println("=== Formatting generated code (go fmt) ===")
	cmdFmt := exec.Command("go", "fmt", "./...")
	cmdFmt.Dir = root
	_ = cmdFmt.Run()

	fmt.Println("===================================")
	fmt.Println("Selesai! Semua file & service interface berhasil di-update.")
	fmt.Println("Sekarang jalankan: gf run main.go")
	fmt.Println("===================================")
}
