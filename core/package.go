package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PatchPackageJson replaces a dependency URL with a local tgz path
func PatchPackageJson(dir, pkgName, localRef string) error {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("parse package.json: %w", err)
	}

	found := false
	for _, section := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
		if deps, ok := pkg[section].(map[string]interface{}); ok {
			if _, exists := deps[pkgName]; exists {
				deps[pkgName] = localRef
				found = true
			}
		}
	}

	if !found {
		return fmt.Errorf("package %s not found in package.json", pkgName)
	}

	output, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package.json: %w", err)
	}

	if err := os.WriteFile(path, append(output, '\n'), 0644); err != nil {
		return fmt.Errorf("write package.json: %w", err)
	}

	return nil
}

// PatchLockJson replaces a dependency URL with a local tgz path in package-lock.json
func PatchLockJson(dir, pkgName, localRef string) error {
	path := filepath.Join(dir, "package-lock.json")
	data, err := os.ReadFile(path)
	if err != nil {
		// Lock file might not exist yet
		return nil
	}

	var lock map[string]interface{}
	if err := json.Unmarshal(data, &lock); err != nil {
		return fmt.Errorf("parse package-lock.json: %w", err)
	}

	// Update lockfileVersion 3: packages[""].dependencies
	if packages, ok := lock["packages"].(map[string]interface{}); ok {
		if root, ok := packages[""].(map[string]interface{}); ok {
			if deps, ok := root["dependencies"].(map[string]interface{}); ok {
				if _, exists := deps[pkgName]; exists {
					deps[pkgName] = localRef
				}
			}
		}
	}

	// Update lockfileVersion 1: dependencies[pkgName]
	if deps, ok := lock["dependencies"].(map[string]interface{}); ok {
		if pkg, ok := deps[pkgName].(map[string]interface{}); ok {
			pkg["version"] = localRef
			if _, exists := pkg["resolved"]; exists {
				pkg["resolved"] = localRef
			}
		}
	}

	// Update packages["node_modules/"+pkgName]
	if packages, ok := lock["packages"].(map[string]interface{}); ok {
		key := "node_modules/" + pkgName
		if pkg, ok := packages[key].(map[string]interface{}); ok {
			pkg["resolved"] = localRef
		}
	}

	output, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package-lock.json: %w", err)
	}

	if err := os.WriteFile(path, append(output, '\n'), 0644); err != nil {
		return fmt.Errorf("write package-lock.json: %w", err)
	}

	return nil
}

// HasCubeManualLenovo checks if package.json has cubeManualLenovo dependency
func HasCubeManualLenovo(dir string) bool {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	for _, section := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
		if deps, ok := pkg[section].(map[string]interface{}); ok {
			if _, exists := deps["cubeManualLenovo"]; exists {
				return true
			}
		}
	}
	return false
}
