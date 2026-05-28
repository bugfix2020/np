package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MergeNewDeps merges new dependencies from generated files back into .bak files
func MergeNewDeps(dir string) error {
	addedPkgs, err := mergePackageJson(dir)
	if err != nil {
		return fmt.Errorf("merge package.json: %w", err)
	}

	addedLock, err := mergePackageLockJson(dir)
	if err != nil {
		return fmt.Errorf("merge package-lock.json: %w", err)
	}

	fmt.Printf("[OK] 合并完成: %d 个 package.json 依赖, %d 个 lock 条目\n", addedPkgs, addedLock)
	return nil
}

// mergePackageJson patches package.json.bak by inserting new dependencies
func mergePackageJson(dir string) (int, error) {
	origPath := filepath.Join(dir, "package.json.bak")
	newPath := filepath.Join(dir, "package.json")

	origData, err := os.ReadFile(origPath)
	if err != nil {
		return 0, err
	}

	var origPkg, newPkg map[string]interface{}
	json.Unmarshal(origData, &origPkg)
	json.Unmarshal(readFileSafe(newPath), &newPkg)

	added := 0
	content := string(origData)

	for _, section := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
		origS := getMap(origPkg, section)
		newS := getMap(newPkg, section)

		// Find new keys
		var newKeys []string
		for name := range newS {
			if _, exists := origS[name]; !exists {
				newKeys = append(newKeys, name)
			}
		}

		if len(newKeys) == 0 {
			continue
		}

		sort.Strings(newKeys)

		// Find the closing brace of this section
		sectionEnd := findSectionEnd(content, section)
		if sectionEnd < 0 {
			continue
		}

		// Build insertion lines
		var lines []string
		for _, key := range newKeys {
			val, _ := json.Marshal(newS[key])
			lines = append(lines, fmt.Sprintf("    %s: %s,", `"`+key+`"`, string(val)))
		}
		insertStr := strings.Join(lines, "\n") + "\n"

		// Insert before closing brace
		content = content[:sectionEnd] + insertStr + content[sectionEnd:]
		added += len(newKeys)
	}

	// Clean empty optionalDependencies
	if strings.Contains(content, `"optionalDependencies": {}`) {
		content = removeEmptySection(content, "optionalDependencies")
	}

	if err := os.WriteFile(origPath, []byte(content), 0644); err != nil {
		return 0, err
	}

	return added, nil
}

// findSectionEnd finds the position of the closing brace for a section
func findSectionEnd(content, section string) int {
	// Find "section": {
	search := `"` + section + `": {`
	idx := strings.Index(content, search)
	if idx < 0 {
		// Try with extra spaces
		search = `"` + section + `":{`
		idx = strings.Index(content, search)
		if idx < 0 {
			return -1
		}
	}

	// Find the opening brace
	braceStart := idx + len(search) - 1
	braceCount := 0
	for i := braceStart; i < len(content); i++ {
		if content[i] == '{' {
			braceCount++
		} else if content[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return i
			}
		}
	}
	return -1
}

// removeEmptySection removes a section with empty object value
func removeEmptySection(content, section string) string {
	search := `"` + section + `": {}`
	idx := strings.Index(content, search)
	if idx < 0 {
		return content
	}

	// Find the start of this line (including leading whitespace)
	start := idx
	for start > 0 && content[start-1] == ' ' {
		start--
	}
	if start > 0 && content[start-1] == '\n' {
		start++
	}

	// Find the end of this line (including trailing comma and newline)
	end := idx + len(search)
	if end < len(content) && content[end] == ',' {
		end++
	}
	if end < len(content) && content[end] == '\n' {
		end++
	}

	return content[:start] + content[end:]
}

// mergePackageLockJson merges new lock entries into the backup
func mergePackageLockJson(dir string) (int, error) {
	origPath := filepath.Join(dir, "package-lock.json.bak")
	newPath := filepath.Join(dir, "package-lock.json")

	origData, err := os.ReadFile(origPath)
	if err != nil {
		return 0, err
	}

	newData, err := os.ReadFile(newPath)
	if err != nil {
		return 0, err
	}

	var origLock, newLock map[string]interface{}
	json.Unmarshal(origData, &origLock)
	json.Unmarshal(newData, &newLock)

	added := 0

	// Merge packages (lockfileVersion 3)
	origPkgKeys := make(map[string]bool)
	if packages, ok := origLock["packages"].(map[string]interface{}); ok {
		for k := range packages {
			if k != "" {
				origPkgKeys[k] = true
			}
		}
	}
	if newPackages, ok := newLock["packages"].(map[string]interface{}); ok {
		for k, v := range newPackages {
			if k == "" {
				continue
			}
			if !origPkgKeys[k] {
				origLock["packages"].(map[string]interface{})[k] = v
				added++
			}
		}
	}

	// Merge dependencies (lockfileVersion 1)
	origDepKeys := make(map[string]bool)
	if deps, ok := origLock["dependencies"].(map[string]interface{}); ok {
		for k := range deps {
			origDepKeys[k] = true
		}
	}
	if newDeps, ok := newLock["dependencies"].(map[string]interface{}); ok {
		for k, v := range newDeps {
			if !origDepKeys[k] {
				origLock["dependencies"].(map[string]interface{})[k] = v
				added++
			}
		}
	}

	// Write back
	output, err := json.MarshalIndent(origLock, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(origPath, append(output, '\n'), 0644); err != nil {
		return 0, err
	}

	return added, nil
}

func readFileSafe(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return make(map[string]interface{})
}
