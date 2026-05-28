package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bugfix2020/np/core"
)

var version = "1.0.0"

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		// np → full install
		core.RunInstall(nil)
		return
	}

	// np --help / np -h
	if args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	}

	// np --version / np -v
	if args[0] == "--version" || args[0] == "-v" {
		fmt.Println("np v" + version)
		return
	}

	// np i / np install / np add
	if args[0] == "i" || args[0] == "install" || args[0] =="" {
		// np i → full install
		if len(args) == 1 {
			core.RunInstall(nil)
			return
		}
		// np i pkg@latest → install specific package
		pkgArgs, registry := parseArgs(args[1:])
		core.SetRegistry(registry)
		core.RunInstall(pkgArgs)
		return
	}

	// np i pkg@latest (without explicit install subcommand)
	pkgArgs, registry := parseArgs(args)
	if len(pkgArgs) > 0 {
		core.SetRegistry(registry)
		core.RunInstall(pkgArgs)
		return
	}

	fmt.Fprintf(os.Stderr, "np: unknown command %q\n", args[0])
	printHelp()
	os.Exit(1)
}

func parseArgs(args []string) ([]string, string) {
	var pkgArgs []string
	registry := ""

	for i := 0; i < len(args); i++ {
		if args[i] == "--registry" && i+1 < len(args) {
			registry = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--registry=") {
			registry = strings.TrimPrefix(args[i], "--registry=")
		} else {
			pkgArgs = append(pkgArgs, args[i])
		}
	}

	return pkgArgs, registry
}

func printHelp() {
	fmt.Println(`np (npm-proxy) v` + version + ` — bypass internal registry for npm install

Usage:
  np                          Full install (equivalent to npm install)
  np i                        Same as above
  np install                  Same as above
  np i <pkg>[@version]        Install specific package
  np i -D <pkg>[@version]     Install as devDependency
  np i <pkg1> <pkg2>          Install multiple packages
  np --registry <url>         Use custom registry
  np --help                   Show this help
  np --version                Show version

Environment Variables:
  FALLBACK_REGISTRY           Registry URL (default: https://registry.npmmirror.com)
  NP_CUBE_TGZ                 Path to cubeManualLenovo tgz file

Examples:
  np                                    # full install from npmmirror
  np i @agent-ils/logger@latest         # add single package
  np i -D vitest@latest                 # add devDependency
  np --registry https://registry.npmjs.org  # use custom registry`)
}
