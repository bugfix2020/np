package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var fallbackRegistry = "https://registry.npmmirror.com"

// SetRegistry overrides the default fallback registry
func SetRegistry(registry string) {
	if registry != "" {
		fallbackRegistry = registry
	}
}

// RunInit initializes the localDeps directory for the current project
func RunInit() {
	fmt.Println("[Step 1/4] 检查项目环境...")
	dir := getProjectRoot()
	fmt.Printf("[OK] 已定位项目根目录: %s\n", dir)

	fmt.Println("\n[Step 2/4] 获取项目名称...")
	projectName := getProjectName(dir)
	fmt.Printf("[OK] 项目名称: %s\n", projectName)

	fmt.Println("\n[Step 3/4] 创建本地依赖目录...")
	parentDir := filepath.Dir(dir)
	localDepsDir := filepath.Join(parentDir, "localDeps", projectName)

	if err := os.MkdirAll(localDepsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] 创建目录失败 %s: %v\n", localDepsDir, err)
		os.Exit(1)
	}
	fmt.Printf("[OK] 目录已创建: %s\n", localDepsDir)

	fmt.Println("\n[Step 4/4] 检查预置本地依赖包...")
	// Check if there are any built-in tgz packages for this project
	builtinPkgs := getBuiltinPackages(projectName)

	if len(builtinPkgs) == 0 {
		fmt.Printf("[INFO] 项目 %s 没有预置的本地依赖包\n", projectName)
		fmt.Println("[INFO] 你可以手动将 .tgz 文件放入目录:")
		fmt.Printf("       %s\n", localDepsDir)
	} else {
		for pkgName, srcPath := range builtinPkgs {
			dstPath := filepath.Join(localDepsDir, filepath.Base(srcPath))

			// Check if already exists
			if _, err := os.Stat(dstPath); err == nil {
				fmt.Printf("[SKIP] %s 已存在，跳过\n", filepath.Base(srcPath))
				continue
			}

			// Check if source exists
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				fmt.Printf("[WARN] 预置包源文件不存在: %s\n", srcPath)
				fmt.Printf("[HINT] 请手动将 %s 放入 %s\n", pkgName, localDepsDir)
				continue
			}

			// Copy
			fmt.Printf("[COPY] %s → %s\n", srcPath, dstPath)
			if err := copyFile(srcPath, dstPath); err != nil {
				fmt.Printf("[ERR] 复制失败: %v\n", err)
				fmt.Printf("[HINT] 请手动将 %s 放入 %s\n", pkgName, localDepsDir)
			} else {
				fmt.Printf("[OK] %s 已复制到本地依赖目录\n", filepath.Base(srcPath))
			}
		}
	}

	// List current contents
	fmt.Println("\n=========================================")
	fmt.Println("[DONE] 初始化完成")
	fmt.Printf("  项目: %s\n", projectName)
	fmt.Printf("  目录: %s\n", localDepsDir)
	entries, _ := os.ReadDir(localDepsDir)
	if len(entries) == 0 {
		fmt.Println("  内容: (空)")
	} else {
		fmt.Println("  内容:")
		for _, e := range entries {
			fmt.Printf("    - %s\n", e.Name())
		}
	}
	fmt.Println("  运行 np i 即可使用本地依赖安装")
	fmt.Println("=========================================")
}

// getBuiltinPackages returns built-in tgz packages for a given project
// Only projects with known dependencies will have packages
func getBuiltinPackages(projectName string) map[string]string {
	result := make(map[string]string)

	// Define built-in packages per project
	builtin := map[string][]struct {
		pkgName string
		tgzName string
	}{
		"baiying-intelligent-web": {
			{pkgName: "cubeManualLenovo", tgzName: "cubeManualLenovo-2.6.8.tgz"},
		},
	}

	pkgs, ok := builtin[projectName]
	if !ok {
		return result
	}

	// Search for tgz files in common locations
	home, _ := os.UserHomeDir()
	searchPaths := []string{
		filepath.Join(home, "wwwroot/lenovo/localDeps/baiying-intelligent-web"),
		filepath.Join(home, "wwwroot/self/localDeps", projectName),
		filepath.Join(home, ".np/localDeps", projectName),
	}

	for _, pkg := range pkgs {
		for _, searchDir := range searchPaths {
			tgzPath := filepath.Join(searchDir, pkg.tgzName)
			if _, err := os.Stat(tgzPath); err == nil {
				result[pkg.pkgName] = tgzPath
				break
			}
		}
	}

	return result
}

// RunInstall is the main entry point for npm install operations
func RunInstall(pkgArgs []string) {
	dir := getProjectRoot()

	// Check for registry env var
	if envRegistry := os.Getenv("FALLBACK_REGISTRY"); envRegistry != "" {
		fallbackRegistry = envRegistry
	}

	// Get project name and find local tgz files
	projectName := getProjectName(dir)
	parentDir := filepath.Dir(dir)
	localDepsDir := filepath.Join(parentDir, "localDeps", projectName)

	// Scan localDeps for tgz files
	localTgzs := findLocalTgzs(localDepsDir)
	if len(localTgzs) > 0 {
		fmt.Printf("[INFO] 找到 %d 个本地依赖包:\n", len(localTgzs))
		for name, path := range localTgzs {
			fmt.Printf("  - %s: %s\n", name, filepath.Base(path))
		}
	}

	// Check which dependencies need local tgz
	neededTgzs := findNeededTgzs(dir, localTgzs)
	if len(neededTgzs) == 0 {
		fmt.Println("[INFO] 无需本地 tgz 替换")
	}

	// Step 1: Backup files
	fmt.Println("\n[Step 1/7] 备份原始文件...")
	if err := BackupFiles(dir); err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] backup failed: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Patch local tgz dependencies
	if len(neededTgzs) > 0 {
		fmt.Println("\n[Step 2/7] 替换本地依赖...")
		for pkgName, tgzPath := range neededTgzs {
			relPath, _ := filepath.Rel(dir, tgzPath)
			fmt.Printf("[RUN] %s → %s\n", pkgName, relPath)
			if err := PatchPackageJson(dir, pkgName, relPath); err != nil {
				fmt.Fprintf(os.Stderr, "[ERR] patch %s failed: %v\n", pkgName, err)
				RestoreOnFailure(dir)
				os.Exit(1)
			}
			if err := PatchLockJson(dir, pkgName, relPath); err != nil {
				fmt.Fprintf(os.Stderr, "[ERR] patch %s in lock failed: %v\n", pkgName, err)
				RestoreOnFailure(dir)
				os.Exit(1)
			}
		}
		fmt.Printf("[OK] 已替换 %d 个本地依赖\n", len(neededTgzs))
	} else {
		fmt.Println("\n[Step 2/7] 跳过 (无需本地依赖替换)")
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

// getProjectRoot checks for .git and returns project root directory
func getProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERR] cannot get working directory: %v\n", err)
		os.Exit(1)
	}

	// Check for .git
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[ERR] 未找到 .git 目录，请在项目根目录下运行\n")
		os.Exit(1)
	}

	// Check for package.json
	if _, err := os.Stat(filepath.Join(dir, "package.json")); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "[ERR] missing package.json in %s\n", dir)
		os.Exit(1)
	}

	return dir
}

// getProjectName gets the project name from git remote or package.json
func getProjectName(dir string) string {
	// Try git remote
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err == nil {
		url := strings.TrimSpace(string(output))
		// Extract project name from URL
		// e.g., http://gitlab.lenovohuishang.com/baiying/baiying-intelligent-web.git
		//     → baiying-intelligent-web
		if idx := strings.LastIndex(url, "/"); idx != -1 {
			name := url[idx+1:]
			name = strings.TrimSuffix(name, ".git")
			if name != "" {
				return name
			}
		}
	}

	// Fallback to package.json name
	cmd = exec.Command("node", "-e", "console.log(require('./package.json').name)")
	cmd.Dir = dir
	output, err = cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	// Fallback to directory name
	return filepath.Base(dir)
}

// findLocalTgzs scans localDeps directory for tgz files
func findLocalTgzs(localDepsDir string) map[string]string {
	result := make(map[string]string)

	entries, err := os.ReadDir(localDepsDir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Scan subdirectory for tgz files
			subDir := filepath.Join(localDepsDir, entry.Name())
			subEntries, err := os.ReadDir(subDir)
			if err != nil {
				continue
			}
			for _, subEntry := range subEntries {
				if !subEntry.IsDir() && strings.HasSuffix(subEntry.Name(), ".tgz") {
					pkgName := strings.TrimSuffix(subEntry.Name(), ".tgz")
					// Remove version suffix if present
					if idx := strings.LastIndex(pkgName, "-"); idx != -1 {
						pkgName = pkgName[:idx]
					}
					result[pkgName] = filepath.Join(subDir, subEntry.Name())
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".tgz") {
			pkgName := strings.TrimSuffix(entry.Name(), ".tgz")
			if idx := strings.LastIndex(pkgName, "-"); idx != -1 {
				pkgName = pkgName[:idx]
			}
			result[pkgName] = filepath.Join(localDepsDir, entry.Name())
		}
	}

	return result
}

// findNeededTgzs checks which dependencies in package.json have local tgz files
func findNeededTgzs(dir string, localTgzs map[string]string) map[string]string {
	result := make(map[string]string)

	for pkgName, tgzPath := range localTgzs {
		if hasDependency(dir, pkgName) {
			result[pkgName] = tgzPath
		}
	}

	return result
}

// hasDependency checks if a package exists in package.json
func hasDependency(dir, pkgName string) bool {
	path := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	// Simple string search for now
	content := string(data)
	return strings.Contains(content, `"`+pkgName+`"`)
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
