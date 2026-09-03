package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// abbrev splits strings with space, underscore, or hyphen to create short badges (e.g. "Users" -> "US")
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

// uncountables/invariable words that should not be trimmed of their trailing 's'
var invariantWords = map[string]bool{
	"status":       true,
	"campus":       true,
	"apparatus":    true,
	"virus":        true,
	"fetus":        true,
	"bonus":        true,
	"nexus":        true,
	"census":       true,
	"focus":        true,
	"radius":       true,
	"minus":        true,
	"plus":         true,
	"stimulus":     true,
	"syllabus":     true,
	"news":         true,
	"series":       true,
	"species":      true,
	"corps":        true,
	"debris":       true,
	"headquarters": true,
	"basis":        true,
	"crisis":       true,
	"analysis":     true,
	"thesis":       true,
	"axis":         true,
	"oasis":        true,
	"diagnosis":    true,
	"prognosis":    true,
	"synopsis":     true,
	"parenthesis":  true,
	"paralysis":    true,
	"genesis":      true,
	"canvas":       true,
	"atlas":        true,
	"alias":        true,
	"bias":         true,
	"gas":          true,
	"lens":         true,
	"bus":          true,
	"business":     true,
	"process":      true,
	"address":      true,
	"access":       true,
	"class":        true,
	"glass":        true,
	"grass":        true,
	"cross":        true,
	"loss":         true,
	"boss":         true,
	"mass":         true,
	"pass":         true,
}

// shortName removes trailing plural indicators for singular form: "Users" → "User", "Statuses" → "Status"
func shortName(s string) string {
	lower := strings.ToLower(s)

	// If the word itself is an invariant/singular word ending with s, keep it as is
	if invariantWords[lower] {
		return s
	}

	// Specific plural forms for -us, -is, -ss
	if strings.HasSuffix(lower, "statuses") {
		return s[:len(s)-2] // Statuses -> Status
	}
	if strings.HasSuffix(lower, "campuses") {
		return s[:len(s)-2] // Campuses -> Campus
	}
	if strings.HasSuffix(lower, "buses") {
		return s[:len(s)-2] // Buses -> Bus
	}
	if strings.HasSuffix(lower, "processes") {
		return s[:len(s)-2] // Processes -> Process
	}
	if strings.HasSuffix(lower, "addresses") {
		return s[:len(s)-2] // Addresses -> Address
	}
	if strings.HasSuffix(lower, "businesses") {
		return s[:len(s)-2] // Businesses -> Business
	}
	if strings.HasSuffix(lower, "classes") {
		return s[:len(s)-2] // Classes -> Class
	}
	if strings.HasSuffix(lower, "matrices") {
		return s[:len(s)-4] + "x" // Matrices -> Matrix
	}
	if strings.HasSuffix(lower, "indices") {
		return s[:len(s)-4] + "x" // Indices -> Index
	}
	if strings.HasSuffix(lower, "analyses") {
		return s[:len(s)-2] + "is" // Analyses -> Analysis
	}
	if strings.HasSuffix(lower, "theses") {
		return s[:len(s)-2] + "is" // Theses -> Thesis
	}
	if strings.HasSuffix(lower, "crises") {
		return s[:len(s)-2] + "is" // Crises -> Crisis
	}
	if strings.HasSuffix(lower, "quizzes") {
		return s[:len(s)-3] // Quizzes -> Quiz
	}

	// General suffixes
	if strings.HasSuffix(lower, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "xes") || strings.HasSuffix(lower, "ches") || strings.HasSuffix(lower, "shes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") {
		// Ensure stripping 's' doesn't leave an empty string or single letter
		if len(s) > 2 {
			candidate := s[:len(s)-1]
			if !invariantWords[strings.ToLower(candidate)] {
				return candidate
			}
		}
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

// parseTableFlags converts command-line --table flag string into a lookup map
func parseTableFlags(tableFlag string) map[string]bool {
	tableSet := map[string]bool{}
	if tableFlag != "" {
		for _, t := range strings.Split(tableFlag, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tableSet[strings.ToLower(t)] = true
			}
		}
	}
	return tableSet
}

// isTargetTable checks if a given entity matches the requested table filter
func isTargetTable(info *TableInfo, tableSet map[string]bool) bool {
	if len(tableSet) == 0 {
		return true
	}
	return tableSet[strings.ToLower(info.TableName)] ||
		tableSet[strings.ToLower(info.StructName)] ||
		tableSet[strings.ToLower(info.ShortName)]
}

// buildNavItems builds navigation items for all entities with active state set for current table
func buildNavItems(allEntities []*TableInfo, activeTableName string) []NavItem {
	var navItems []NavItem
	for _, ent := range allEntities {
		navItems = append(navItems, NavItem{
			Name:      ent.ShortName,
			TableName: ent.TableName,
			Active:    ent.TableName == activeTableName,
			Abbrev:    abbrev(ent.ShortName),
		})
	}
	return navItems
}
