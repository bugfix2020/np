package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var fallbackRegistry = "https://registry.npmmirror.com"

// SetRegistry overrides the default fallback registry
func SetRegistry(registry string) {
	if registry != "" {
		fallbackRegistry = registry
	}
}

// RunInstall is the main entry point for npm install operations
func RunInstall(pkgArgs []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] cannot get working directory: %v\n", err)
		os.Exit(1)
	}

	// Check for package.json
	if _, err := os.Stat(filepath.Join(dir, "package.json")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[ERR] missing package.json in %s\n", dir)
		os.Exit(1)
	}

	// Check for registry env var
	if envRegistry := os.Getenv("FALLBACK_REGISTRY"); envRegistry != "" {
		fallbackRegistry = envRegistry
	}

	// Check if cubeManualLenovo exists and needs local tgz
	hasCube := HasCubeManualLenovo(dir)

	// Check for local tgz file
	if hasCube {
		localTgz := os.Getenv("NP_CUBE_TGZ")
		if localTgz == "" {
			// Try default path
			home, _ := os.UserHomeDir()
			localTgz = filepath.Join(home, "wwwroot/localDeps/baiying-intelligent-web/cubeManualLenovo-2.6.8.tgz")
		}
		if _, err := os.Stat(localTgz); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "[ERR] local tgz not found: %s\n", localTgz)
			os.Exit(1)
		}
	}

	// Step 1: Backup files
	fmt.Println("\n[Step 1/7] 备份原始文件...")
	if err := BackupFiles(dir); err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] backup failed: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Patch cubeManualLenovo (if exists)
	if hasCube {
		fmt.Println("\n[Step 2/7] 替换 cubeManualLenovo...")
		if _, err := PatchCubeManualLenovo(dir); err != nil {
			fmt.Fprintf(os.Stderr, "[ERR] patch cube failed: %v\n", err)
			RestoreOnFailure(dir)
			os.Exit(1)
		}
		if err := PatchCubeManualLenovoInLock(dir); err != nil {
			fmt.Fprintf(os.Stderr, "[ERR] patch cube in lock failed: %v\n", err)
			RestoreOnFailure(dir)
			os.Exit(1)
		}
	} else {
		fmt.Println("\n[Step 2/7] 跳过 (无 cubeManualLenovo)")
	}

	// Step 3: Patch .npmrc
	fmt.Println("\n[Step 3/7] 修改 .npmrc...")
	if _, err := os.Stat(filepath.Join(dir, ".npmrc")); err == nil {
		if err := PatchNpmrc(dir, fallbackRegistry); err != nil {
			fmt.Fprintf(os.Stderr, "[ERR] patch .npmrc failed: %v\n", err)
			RestoreOnFailure(dir)
			os.Exit(1)
		}
	} else {
		fmt.Println("[OK] .npmrc 不存在，跳过")
	}

	// Step 4: Clean cache + node_modules + lock files
	fmt.Println("\n[Step 4/7] 清理缓存和 lock 文件...")
	cleanCache()
	cleanNodeModules(dir)
	cleanLockFiles(dir)

	// Step 5: Run npm install
	fmt.Println("\n[Step 5/7] 执行 npm install...")
	if !runNpmInstall(dir, pkgArgs) {
		fmt.Fprintln(os.Stderr, "[ERR] npm install failed, 正在恢复原始文件...")
		RestoreOnFailure(dir)
		os.Exit(1)
	}
	fmt.Println("[OK] npm install 成功")

	// Step 6: Merge new deps
	fmt.Println("\n[Step 6/7] 合并新增依赖...")
	if _, err := os.Stat(filepath.Join(dir, "package.json.bak")); err == nil {
		if err := MergeNewDeps(dir); err != nil {
			fmt.Fprintf(os.Stderr, "[ERR] merge failed: %v\n", err)
			RestoreOnFailure(dir)
			os.Exit(1)
		}
	}

	// Step 7: Restore original files
	fmt.Println("\n[Step 7/7] 恢复原始文件...")
	if err := RestoreFiles(dir); err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] restore failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=========================================")
	fmt.Println("[DONE]")
	fmt.Println("  node_modules/     ✓ 已从 fallback registry 安装")
	fmt.Println("  package.json      ✓ 原始文件 + 新增依赖已合并")
	fmt.Println("  package-lock.json ✓ 原始文件 + 新增 lock 条目已合并")
	fmt.Println("  yarn.lock         ✓ 已恢复原始文件")
	fmt.Println("  .npmrc            ✓ 已恢复原始 registry")
	fmt.Println("=========================================")
}

func cleanCache() {
	cmd := exec.Command("npm", "cache", "clean", "--force")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // ignore error
	fmt.Println("[OK] npm cache cleaned")
}

func cleanNodeModules(dir string) {
	nodeModules := filepath.Join(dir, "node_modules")
	if _, err := os.Stat(nodeModules); err == nil {
		os.RemoveAll(nodeModules)
		fmt.Println("[OK] node_modules removed")
	}
}

func cleanLockFiles(dir string) {
	lockFiles := []string{
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"pnpm-lock.yml",
		"shrinkwrap.json",
		"npm-shrinkwrap.json",
	}
	for _, f := range lockFiles {
		os.Remove(filepath.Join(dir, f))
	}
	fmt.Println("[OK] lock files removed")
}

func runNpmInstall(dir string, pkgArgs []string) bool {
	args := []string{"install"}

	// Add package arguments (if any)
	if len(pkgArgs) > 0 {
		args = append(args, pkgArgs...)
	}

	// Add registry flag
	args = append(args, "--registry", fallbackRegistry)

	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NODE_OPTIONS=--dns-result-order=ipv4first")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return err == nil
}

// HasLockFile checks if a lock file exists
func HasLockFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// ParseNpmOutput parses npm output to check for csnexus references
func ParseNpmOutput(output string) bool {
	matched, _ := regexp.MatchString(`csnexus\.lenovo\.com\.cn`, output)
	return matched
}

// CleanNpmCacheOutput cleans npm output to remove sensitive info
func CleanNpmCacheOutput(output string) string {
	lines := strings.Split(output, "\n")
	var cleaned []string
	for _, line := range lines {
		if !strings.Contains(line, "csnexus") {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, "\n")
}
