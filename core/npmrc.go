package core

import (
	"fmt"
	"os"
	"strings"
)

// PatchNpmrc temporarily switches .npmrc to use the fallback registry
func PatchNpmrc(dir string, registry string) error {
	path := dir + "/.npmrc"
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read .npmrc: %w", err)
	}

	content := string(data)

	// Comment out all registry= lines
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "registry=") {
			lines[i] = "# " + line + "  (disabled for install)"
		}
	}

	// Append new registry at the end
	result := strings.Join(lines, "\n")
	result = strings.TrimRight(result, "\n") + "\nregistry=" + registry + "\n"

	if err := os.WriteFile(path, []byte(result), 0644); err != nil {
		return fmt.Errorf("write .npmrc: %w", err)
	}

	fmt.Printf("[OK] .npmrc → %s\n", registry)
	return nil
}
