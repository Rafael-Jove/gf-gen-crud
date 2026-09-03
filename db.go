package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// parseDBLink parses a GoFrame link string for mysql, pgsql/postgres, or sqlite/sqlite3
func parseDBLink(link string) (driverName string, dsn string, ok bool) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", "", false
	}
	colonIdx := strings.Index(link, ":")
	if colonIdx < 0 {
		return "", "", false
	}
	driver := strings.ToLower(link[:colonIdx])

	switch driver {
	case "mysql":
		parts := strings.SplitN(link, ":", 3)
		if len(parts) < 3 {
			return "", "", false
		}
		user := parts[1]
		rest := parts[2] // pass@tcp(host:port)/dbname
		atIdx := strings.LastIndex(rest, "@")
		if atIdx < 0 {
			return "", "", false
		}
		pass := rest[:atIdx]
		hostDB := rest[atIdx+1:] // tcp(host:port)/dbname
		slashIdx := strings.Index(hostDB, "/")
		if slashIdx < 0 {
			return "", "", false
		}
		hostPart := hostDB[:slashIdx]
		dbname := hostDB[slashIdx+1:]
		if q := strings.Index(dbname, "?"); q >= 0 {
			dbname = dbname[:q]
		}
		hostPart = strings.TrimPrefix(hostPart, "tcp(")
		hostPart = strings.TrimSuffix(hostPart, ")")
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, hostPart, dbname)
		return "mysql", dsn, true

	case "pgsql", "postgres":
		if strings.HasPrefix(link, "postgres://") || strings.HasPrefix(link, "postgresql://") {
			return "postgres", link, true
		}
		parts := strings.SplitN(link, ":", 3)
		if len(parts) >= 3 {
			user := parts[1]
			rest := parts[2]
			atIdx := strings.LastIndex(rest, "@")
			if atIdx >= 0 {
				pass := rest[:atIdx]
				hostDB := rest[atIdx+1:]
				slashIdx := strings.Index(hostDB, "/")
				if slashIdx >= 0 {
					hostPart := hostDB[:slashIdx]
					dbname := hostDB[slashIdx+1:]
					if q := strings.Index(dbname, "?"); q >= 0 {
						dbname = dbname[:q]
					}
					hostPart = strings.TrimPrefix(hostPart, "tcp(")
					hostPart = strings.TrimSuffix(hostPart, ")")
					dsn = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, pass, hostPart, dbname)
					return "postgres", dsn, true
				}
			}
		}
		raw := strings.TrimPrefix(link, driver+":")
		return "postgres", raw, true

	case "sqlite", "sqlite3":
		raw := strings.TrimPrefix(link, driver+":")
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return "", "", false
		}
		return "sqlite", raw, true

	default:
		return "", "", false
	}
}

// fetchDBColumnTypes reads hack/config.yaml to get the DB DSN, then queries
// schema columns for the given table and returns a map of column_name → DBColMeta.
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
	driverName, dsn, ok := parseDBLink(link)
	if !ok {
		return nil
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil
	}
	defer db.Close()

	var query string
	var args []interface{}

	switch driverName {
	case "mysql":
		query = `
			SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_KEY, IS_NULLABLE, COLUMN_DEFAULT, EXTRA
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION`
		args = append(args, tableName)

	case "postgres":
		query = `
			SELECT column_name, data_type,
			       CASE WHEN is_nullable = 'NO' THEN 'PRI' ELSE '' END as col_key,
			       is_nullable, column_default
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1
			ORDER BY ordinal_position`
		args = append(args, tableName)

	case "sqlite":
		query = `PRAGMA table_info(` + tableName + `)`

	default:
		return nil
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := map[string]DBColMeta{}
	for rows.Next() {
		var colName, colType, colKey string
		var isRequired bool
		var isAutoIncrement bool

		if driverName == "sqlite" {
			var cid int
			var notNull int
			var dfltValue interface{}
			var pk int
			if err := rows.Scan(&cid, &colName, &colType, &notNull, &dfltValue, &pk); err != nil {
				continue
			}
			if pk > 0 {
				colKey = "PRI"
				if strings.Contains(strings.ToLower(colType), "int") {
					isAutoIncrement = true
				}
			}
			if notNull == 1 && dfltValue == nil {
				isRequired = true
			}
		} else if driverName == "postgres" {
			var isNullable, colDefault sql.NullString
			if err := rows.Scan(&colName, &colType, &colKey, &isNullable, &colDefault); err != nil {
				continue
			}
			if strings.ToUpper(isNullable.String) == "NO" && !colDefault.Valid {
				isRequired = true
			}
			if colDefault.Valid && strings.Contains(strings.ToLower(colDefault.String), "nextval") {
				isAutoIncrement = true
			}
		} else {
			var isNullable, colDefault, extra sql.NullString
			if err := rows.Scan(&colName, &colType, &colKey, &isNullable, &colDefault, &extra); err != nil {
				continue
			}
			if strings.ToUpper(isNullable.String) == "NO" && !colDefault.Valid {
				isRequired = true
			}
			if extra.Valid && strings.Contains(strings.ToLower(extra.String), "auto_increment") {
				isAutoIncrement = true
			}
		}
		colTypeLow := strings.ToLower(colType)
		colNameLow := strings.ToLower(colName)
		meta := DBColMeta{HTMLType: "text", DataType: "string", IsRequired: isRequired, IsAutoIncrement: isAutoIncrement}
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
	_ = rows.Err()

	var fkQuery string
	var fkArgs []interface{}

	switch driverName {
	case "mysql":
		fkQuery = `SELECT COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
			FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL`
		fkArgs = append(fkArgs, tableName)

	case "postgres":
		fkQuery = `SELECT kcu.column_name, ccu.table_name, ccu.column_name
			FROM information_schema.table_constraints AS tc
			JOIN information_schema.key_column_usage AS kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage AS ccu ON ccu.constraint_name = tc.constraint_name AND ccu.table_schema = tc.table_schema
			WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = $1`
		fkArgs = append(fkArgs, tableName)

	case "sqlite":
		fkQuery = fmt.Sprintf("PRAGMA foreign_key_list('%s')", tableName)
	}

	if fkQuery != "" {
		fkRows, err := db.Query(fkQuery, fkArgs...)
		if err == nil {
			defer fkRows.Close()
			for fkRows.Next() {
				var colName, refTable, refCol string

				if driverName == "sqlite" {
					var id, seq int
					var onUpd, onDel, match string
					_ = fkRows.Scan(&id, &seq, &refTable, &refCol, &onUpd, &onDel, &match)
				} else {
					_ = fkRows.Scan(&colName, &refTable, &refCol)
				}

				if meta, ok := result[colName]; ok {
					meta.IsFK = true
					meta.FKTable = refTable
					result[colName] = meta
				}
			}
			_ = fkRows.Err()
		}
	}

	fmt.Printf("Hasil: %v", result)

	return result
}
