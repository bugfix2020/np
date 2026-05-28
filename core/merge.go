package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// mergePackageJson merges new dependencies into package.json.bak
func mergePackageJson(dir string) (int, error) {
	origPath := filepath.Join(dir, "package.json.bak")
	newPath := filepath.Join(dir, "package.json")

	origData, err := os.ReadFile(origPath)
	if err != nil {
		return 0, err
	}

	var origPkg, newPkg map[string]interface{}
	if err := json.Unmarshal(origData, &origPkg); err != nil {
		return 0, err
	}

	newData, err := os.ReadFile(newPath)
	if err != nil {
		return 0, err
	}
	if err := json.Unmarshal(newData, &newPkg); err != nil {
		return 0, err
	}

	added := 0
	content := string(origData)

	for _, section := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
		origS := getMap(origPkg, section)
		newS := getMap(newPkg, section)

		var newKeys []string
		for name := range newS {
			if _, exists := origS[name]; !exists {
				newKeys = append(newKeys, name)
			}
		}

		if len(newKeys) == 0 {
			continue
		}

		// Find the section and its closing brace
		insertPos, err := findSectionInsertPosition(content, section)
		if err != nil {
			continue
		}

		// Build insertion lines
		var lines []string
		for _, key := range newKeys {
			val, _ := json.Marshal(newS[key])
			lines = append(lines, fmt.Sprintf("    %s: %s", `"`+key+`"`, string(val)))
		}
		insertStr := ",\n" + strings.Join(lines, "\n")

		content = content[:insertPos] + insertStr + content[insertPos:]
		added += len(newKeys)
	}

	// Clean empty optionalDependencies
	if strings.Contains(content, `"optionalDependencies": {}`) {
		idx := strings.Index(content, `"optionalDependencies": {}`)
		if idx > 0 {
			start := idx
			for start > 0 && content[start-1] == ' ' {
				start--
			}
			if start > 0 && content[start-1] == '\n' {
				start++
			}
			end := idx + len(`"optionalDependencies": {}`)
			if end < len(content) && content[end] == ',' {
				end++
			}
			if end < len(content) && content[end] == '\n' {
				end++
			}
			content = content[:start] + content[end:]
		}
	}

	// Validate JSON before writing
	var validate interface{}
	if err := json.Unmarshal([]byte(content), &validate); err != nil {
		return 0, fmt.Errorf("merged package.json is invalid JSON: %w", err)
	}
	fmt.Println("[OK] package.json JSON 校验通过")

	if err := os.WriteFile(origPath, []byte(content), 0644); err != nil {
		return 0, err
	}

	return added, nil
}

// findSectionInsertPosition finds where to insert new keys in a section
// Returns two positions: where to add comma, and where to insert new lines
func findSectionInsertPosition(content, section string) (int, error) {
	// Find the section opening
	search := `"` + section + `"`
	idx := strings.Index(content, search)
	if idx < 0 {
		return 0, fmt.Errorf("section %s not found", section)
	}

	// Find the opening brace
	braceIdx := -1
	for i := idx + len(search); i < len(content); i++ {
		if content[i] == '{' {
			braceIdx = i
			break
		}
	}
	if braceIdx < 0 {
		return 0, fmt.Errorf("opening brace not found for %s", section)
	}

	// Find matching closing brace
	depth := 1
	closeBraceIdx := -1
	for i := braceIdx + 1; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				closeBraceIdx = i
				break
			}
		}
	}
	if closeBraceIdx < 0 {
		return 0, fmt.Errorf("closing brace not found for %s", section)
	}

	// Find the last key-value pair before closing brace
	// Look for the last newline before closing brace
	lastNewline := -1
	for i := closeBraceIdx - 1; i > braceIdx; i-- {
		if content[i] == '\n' {
			lastNewline = i
			break
		}
	}

	if lastNewline < 0 {
		return 0, fmt.Errorf("no newline found in %s", section)
	}

	// Find the end of the last value (before trailing whitespace)
	valueEnd := lastNewline
	for valueEnd > braceIdx && (content[valueEnd-1] == ' ' || content[valueEnd-1] == '\t') {
		valueEnd--
	}

	return valueEnd, nil
}

// mergePackageLockJson merges new entries into package-lock.json.bak
func mergePackageLockJson(dir string) (int, error) {
	origLock, err := readJson(filepath.Join(dir, "package-lock.json.bak"))
	if err != nil {
		return 0, err
	}

	newLock, err := readJson(filepath.Join(dir, "package-lock.json"))
	if err != nil {
		return 0, err
	}

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

	if err := writeJson(filepath.Join(dir, "package-lock.json.bak"), origLock); err != nil {
		return 0, err
	}
	fmt.Println("[OK] package-lock.json JSON 校验通过")
	return added, nil
}

func readJson(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return result, nil
}

func writeJson(path string, data map[string]interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(output, '\n'), 0644)
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return make(map[string]interface{})
}
