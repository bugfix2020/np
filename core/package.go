package core

import (
	"encoding/json"
	"fmt"
	"os"
)

const cubeLocalRef = "file:../localDeps/baiying-intelligent-web/cubeManualLenovo-2.6.8.tgz"

// PatchCubeManualLenovo replaces cubeManualLenovo URL with local tgz in package.json
func PatchCubeManualLenovo(dir string) (bool, error) {
	path := dir + "/package.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false, fmt.Errorf("parse package.json: %w", err)
	}

	found := false
	for _, section := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
		if deps, ok := pkg[section].(map[string]interface{}); ok {
			if _, exists := deps["cubeManualLenovo"]; exists {
				deps["cubeManualLenovo"] = cubeLocalRef
				found = true
			}
		}
	}

	if !found {
		return false, nil
	}

	output, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal package.json: %w", err)
	}

	if err := os.WriteFile(path, append(output, '\n'), 0644); err != nil {
		return false, fmt.Errorf("write package.json: %w", err)
	}

	fmt.Println("[OK] cubeManualLenovo → local tgz")
	return true, nil
}

// HasCubeManualLenovo checks if package.json has cubeManualLenovo dependency
func HasCubeManualLenovo(dir string) bool {
	path := dir + "/package.json"
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
