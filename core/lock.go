package core

import (
	"encoding/json"
	"fmt"
	"os"
)

// PatchCubeManualLenovoInLock replaces cubeManualLenovo URL with local tgz in package-lock.json
func PatchCubeManualLenovoInLock(dir string) error {
	path := dir + "/package-lock.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read package-lock.json: %w", err)
	}

	var lock map[string]interface{}
	if err := json.Unmarshal(data, &lock); err != nil {
		return fmt.Errorf("parse package-lock.json: %w", err)
	}

	// Update lockfileVersion 3: packages[""].dependencies
	if packages, ok := lock["packages"].(map[string]interface{}); ok {
		if root, ok := packages[""].(map[string]interface{}); ok {
			if deps, ok := root["dependencies"].(map[string]interface{}); ok {
				if _, exists := deps["cubeManualLenovo"]; exists {
					deps["cubeManualLenovo"] = cubeLocalRef
				}
			}
		}
	}

	// Update lockfileVersion 1: dependencies.cubeManualLenovo
	if deps, ok := lock["dependencies"].(map[string]interface{}); ok {
		if cube, ok := deps["cubeManualLenovo"].(map[string]interface{}); ok {
			cube["version"] = cubeLocalRef
			if _, exists := cube["resolved"]; exists {
				cube["resolved"] = cubeLocalRef
			}
		}
	}

	// Update packages["node_modules/cubeManualLenovo"]
	if packages, ok := lock["packages"].(map[string]interface{}); ok {
		if cube, ok := packages["node_modules/cubeManualLenovo"].(map[string]interface{}); ok {
			cube["resolved"] = cubeLocalRef
		}
	}

	output, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal package-lock.json: %w", err)
	}

	if err := os.WriteFile(path, append(output, '\n'), 0644); err != nil {
		return fmt.Errorf("write package-lock.json: %w", err)
	}

	fmt.Println("[OK] cubeManualLenovo → local tgz (lock)")
	return nil
}
