package core

import (
	"encoding/json"
	"fmt"
	"os"
)

// MergeNewDeps merges new dependencies from generated files back into .bak files
func MergeNewDeps(dir string) error {
	origPkg, err := readJSON(dir + "/package.json.bak")
	if err != nil {
		return fmt.Errorf("read original package.json: %w", err)
	}

	origLock, err := readJSON(dir + "/package-lock.json.bak")
	if err != nil {
		return fmt.Errorf("read original package-lock.json: %w", err)
	}

	newPkg, err := readJSON(dir + "/package.json")
	if err != nil {
		return fmt.Errorf("read new package.json: %w", err)
	}

	newLock, err := readJSON(dir + "/package-lock.json")
	if err != nil {
		return fmt.Errorf("read new package-lock.json: %w", err)
	}

	addedPkgs := 0
	addedLock := 0

	// Merge package.json
	for _, section := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
		origS := getMap(origPkg, section)
		newS := getMap(newPkg, section)
		for name, ver := range newS {
			if _, exists := origS[name]; !exists {
				origS[name] = ver
				addedPkgs++
			}
		}
		origPkg[section] = origS
	}

	// Clean empty optionalDependencies
	if optDeps, ok := origPkg["optionalDependencies"].(map[string]interface{}); ok {
		if len(optDeps) == 0 {
			delete(origPkg, "optionalDependencies")
		}
	}

	// Merge package-lock.json (lockfileVersion 3: packages)
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
				if origLock["packages"] == nil {
					origLock["packages"] = make(map[string]interface{})
				}
				origLock["packages"].(map[string]interface{})[k] = v
				addedLock++
			}
		}
	}

	// Merge package-lock.json (lockfileVersion 1: dependencies)
	origDepKeys := make(map[string]bool)
	if deps, ok := origLock["dependencies"].(map[string]interface{}); ok {
		for k := range deps {
			origDepKeys[k] = true
		}
	}
	if newDeps, ok := newLock["dependencies"].(map[string]interface{}); ok {
		for k, v := range newDeps {
			if !origDepKeys[k] {
				if origLock["dependencies"] == nil {
					origLock["dependencies"] = make(map[string]interface{})
				}
				origLock["dependencies"].(map[string]interface{})[k] = v
				addedLock++
			}
		}
	}

	// Write back
	if err := writeJSON(dir+"/package.json.bak", origPkg); err != nil {
		return fmt.Errorf("write package.json.bak: %w", err)
	}
	if err := writeJSON(dir+"/package-lock.json.bak", origLock); err != nil {
		return fmt.Errorf("write package-lock.json.bak: %w", err)
	}

	fmt.Printf("[OK] 合并完成: %d 个 package.json 依赖, %d 个 lock 条目\n", addedPkgs, addedLock)
	return nil
}

func readJSON(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func writeJSON(path string, data map[string]interface{}) error {
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
