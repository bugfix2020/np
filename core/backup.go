package core

import (
	"fmt"
	"io"
	"os"
)

var (
	// Files that must always be backed up
	BakFiles = []string{"package.json", "package-lock.json", ".npmrc"}

	// Lock files that exist optionally
	OptionalLocks = []string{"yarn.lock", "pnpm-lock.yaml", "pnpm-lock.yml", "shrinkwrap.json", "npm-shrinkwrap.json"}
)

// BackupFiles copies original files to .bak
func BackupFiles(dir string) error {
	// Clean leftover .bak files
	for _, f := range append(BakFiles, OptionalLocks...) {
		os.Remove(dir + "/" + f + ".bak")
	}
	os.Remove(dir + "/.npmrc.bak")

	// Backup required files
	for _, f := range BakFiles {
		src := dir + "/" + f
		dst := src + ".bak"
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("backup %s: %w", f, err)
			}
		}
	}

	// Backup optional lock files
	for _, f := range OptionalLocks {
		src := dir + "/" + f
		dst := src + ".bak"
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, dst); err != nil {
				return fmt.Errorf("backup %s: %w", f, err)
			}
			fmt.Printf("[OK] 备份 %s → %s.bak\n", f, f)
		}
	}

	fmt.Println("[OK] 原始文件已备份为 .bak")
	return nil
}

// RestoreFiles restores .bak files to original names
func RestoreFiles(dir string) error {
	// Restore main files
	for _, f := range []string{"package.json", "package-lock.json"} {
		bak := dir + "/" + f + ".bak"
		dst := dir + "/" + f
		if _, err := os.Stat(bak); err == nil {
			if err := copyFile(bak, dst); err != nil {
				return fmt.Errorf("restore %s: %w", f, err)
			}
		}
	}

	// Restore optional lock files
	for _, f := range OptionalLocks {
		bak := dir + "/" + f + ".bak"
		dst := dir + "/" + f
		if _, err := os.Stat(bak); err == nil {
			if err := copyFile(bak, dst); err != nil {
				return fmt.Errorf("restore %s: %w", f, err)
			}
			fmt.Printf("[OK] 恢复 %s\n", f)
		}
	}

	// Restore .npmrc BEFORE deleting .bak files
	npmrcBak := dir + "/.npmrc.bak"
	npmrcDst := dir + "/.npmrc"
	if _, err := os.Stat(npmrcBak); err == nil {
		if err := copyFile(npmrcBak, npmrcDst); err != nil {
			return fmt.Errorf("restore .npmrc: %w", err)
		}
	}

	// Clean up .bak files
	for _, f := range append(BakFiles, OptionalLocks...) {
		os.Remove(dir + "/" + f + ".bak")
	}
	os.Remove(dir + "/.npmrc.bak")

	return nil
}

// RestoreOnFailure restores all .bak files and cleans up
func RestoreOnFailure(dir string) {
	for _, f := range append(BakFiles, OptionalLocks...) {
		bak := dir + "/" + f + ".bak"
		dst := dir + "/" + f
		if _, err := os.Stat(bak); err == nil {
			copyFile(bak, dst)
		}
	}
	// Restore .npmrc
	npmrcBak := dir + "/.npmrc.bak"
	npmrcDst := dir + "/.npmrc"
	if _, err := os.Stat(npmrcBak); err == nil {
		copyFile(npmrcBak, npmrcDst)
	}
	// Clean up
	for _, f := range append(BakFiles, OptionalLocks...) {
		os.Remove(dir + "/" + f + ".bak")
	}
	os.Remove(dir + "/.npmrc.bak")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
