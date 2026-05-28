# np (npm-proxy)

Bypass internal registry for npm install. A Go binary that wraps npm install to work around inaccessible internal registries.

## Features

- Cross-platform: Linux, macOS, Windows (x86_64/ARM64)
- Single binary, no dependencies
- Preserves original package.json, package-lock.json, .npmrc
- Supports custom registry via environment variable or flag
- Handles cubeManualLenovo local tgz automatically

## Installation

Download the latest binary from [Releases](https://github.com/bugfix2020/np/releases).

```bash
# Linux/macOS
chmod +x np-*
sudo mv np-* /usr/local/bin/np

# Or add to PATH
export PATH=$PATH:$(pwd)
```

## Usage

```bash
# Full install (equivalent to npm install)
np
np i
np install

# Install specific package
np i @agent-ils/logger@latest
np i -D vitest@latest
np i pkg-a@latest pkg-b@^1.0.0

# Custom registry
np --registry https://registry.npmjs.org

# Show help
np --help

# Show version
np --version
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FALLBACK_REGISTRY` | Registry URL to use | `https://registry.npmmirror.com` |
| `NP_CUBE_TGZ` | Path to cubeManualLenovo tgz | `~/wwwroot/localDeps/.../cubeManualLenovo-2.6.8.tgz` |

## How It Works

1. Backs up original files (package.json, package-lock.json, .npmrc) to .bak
2. Temporarily patches cubeManualLenovo to use local tgz
3. Temporarily switches .npmrc to use fallback registry
4. Cleans npm cache and removes node_modules/lock files
5. Runs npm install from fallback registry
6. Merges new dependencies back into original files
7. Restores all original files

## License

MIT
