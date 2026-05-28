# np (npm-proxy)

> **np 不是 npm！** `np` 是一个独立的命令行工具，用于在无法访问内网 npm registry 的情况下，通过外部源（默认 npmmirror）安装项目依赖。

## 这是什么？

当你在公司内网开发时，`npm install` 会从内网 registry 拉取依赖包。但如果你在外网环境（比如家里、咖啡厅），`npm install` 会因为无法访问内网而失败。

`np` 就是为了解决这个问题而生的。它会：
1. 临时切换 npm 的 registry 到外部源
2. 执行 `npm install`
3. 安装完成后自动恢复原始配置
4. **不会修改你的 package.json 和 package-lock.json**（除非你添加了新依赖）

## 下载安装

### 1. 下载二进制文件

前往 [Releases](https://github.com/bugfix2020/np/releases/tag/v1.0.0) 页面，下载适合你系统的版本：

| 系统 | 文件名 |
|------|--------|
| macOS (Intel) | `np-darwin-amd64` |
| macOS (M1/M2/ARM) | `np-darwin-arm64` |
| Linux (x86_64) | `np-linux-amd64` |
| Linux (ARM64) | `np-linux-arm64` |
| Windows (x86_64) | `np-windows-amd64.exe` |
| Windows (ARM64) | `np-windows-arm64.exe` |

### 2. 安装到系统

#### macOS / Linux

```bash
# 1. 下载后，给文件添加执行权限
chmod +x np-darwin-arm64  # 或你下载的文件名

# 2. 移动到系统目录（推荐）
sudo mv np-darwin-arm64 /usr/local/bin/np

# 验证安装
np --version
```

#### Windows

**方法一：放到固定目录并添加到 PATH（推荐）**

1. 在 `C:\` 下创建一个目录，比如 `C:\tools`
2. 把下载的 `np-windows-amd64.exe` 重命名为 `np.exe`
3. 把 `np.exe` 移动到 `C:\tools\`
4. 添加 `C:\tools` 到系统 PATH：
   - 右键「此电脑」→「属性」→「高级系统设置」→「环境变量」
   - 在「系统变量」里找到 `Path`，点击「编辑」
   - 点击「新建」，输入 `C:\tools`
   - 点击「确定」保存
5. **重新打开一个终端**，运行 `np --version` 验证

**方法二：直接使用完整路径**

如果不想配置 PATH，可以直接使用完整路径：

```powershell
C:\tools\np.exe --version
C:\tools\np.exe i wms
```

**方法三：在 WSL 中使用**

如果你在 WSL (Windows Subsystem for Linux) 中开发：

```bash
# 把 np 复制到 WSL 的 /usr/local/bin/
sudo cp /mnt/c/tools/np-windows-amd64.exe /usr/local/bin/np.exe
chmod +x /usr/local/bin/np.exe

# 验证
np.exe --version
```

## 使用方法

### 基本用法

```bash
# 全量安装（等同于 npm install）
np

# 或者显式调用
np i
np install
```

### 安装新包

```bash
# 安装单个包到 dependencies
np i wms

# 安装单个包到 devDependencies
np i -D @aws-sdk/types

# 安装多个包
np i wms mws @aws-sdk/types -D

# 指定版本
np i wms@4.2.1
```

### 初始化本地依赖

```bash
# 在项目根目录运行，创建 localDeps 目录
np init
```

### 使用自定义 registry

```bash
# 使用其他 registry
np --registry https://registry.npmjs.org

# 或者通过环境变量
export FALLBACK_REGISTRY=https://registry.npmjs.org
np i wms
```

### 查看帮助

```bash
np --help
np --version
```

## 命令对照表

| npm 命令 | np 命令 | 说明 |
|----------|---------|------|
| `npm install` | `np` 或 `np i` | 全量安装 |
| `npm install wms` | `np i wms` | 安装包到 dependencies |
| `npm install -D wms` | `np i -D wms` | 安装包到 devDependencies |
| `npm install wms mws` | `np i wms mws` | 安装多个包 |

## 常见问题

### Q: np 和 npm 有什么区别？

`np` 是一个**包装器**，它在底层调用 `npm install`，但会自动处理 registry 切换。安装完成后，你的项目文件不会被修改。

### Q: 运行 np 后，我的 package.json 会变吗？

- **不会**：如果你只是运行 `np` 或 `np i` 进行全量安装，package.json 不会被修改
- **会**：如果你运行 `np i wms` 安装新包，package.json 会新增对应的依赖条目（这是正常行为）

### Q: 运行 np 后，我的 node_modules 会被清空吗？

会。`np` 会先删除 `node_modules` 和 lock 文件，然后重新安装。这是为了确保从正确的 registry 拉取依赖。

### Q: 我需要在项目根目录运行吗？

是的。`np` 会检查当前目录是否有 `.git` 目录。如果没有，会提示你切换到项目根目录。

### Q: Windows 上运行报错怎么办？

1. 确保你下载的是 `.exe` 文件
2. 如果被 Windows Defender 拦截，点击「更多信息」→「仍要运行」
3. 如果提示找不到命令，检查 PATH 配置是否正确

### Q: 如何卸载？

直接删除 `np` 二进制文件即可：
- macOS/Linux: `sudo rm /usr/local/bin/np`
- Windows: 删除 `C:\tools\np.exe` 并从 PATH 中移除

## 工作原理

```
┌─────────────────────────────────────────────────────────────┐
│  1. 备份原始文件 (.bak)                                      │
│     - package.json                                          │
│     - package-lock.json                                     │
│     - .npmrc                                                │
│     - yarn.lock                                             │
├─────────────────────────────────────────────────────────────┤
│  2. 替换本地依赖（如果存在）                                  │
│     - cubeManualLenovo → local tgz                          │
├─────────────────────────────────────────────────────────────┤
│  3. 临时切换 .npmrc 到外部 registry                          │
├─────────────────────────────────────────────────────────────┤
│  4. 清理缓存和 node_modules                                  │
├─────────────────────────────────────────────────────────────┤
│  5. 执行 npm install                                        │
├─────────────────────────────────────────────────────────────┤
│  6. 合并新增依赖到原始文件                                    │
│     - 新包的条目会添加到 package.json                        │
│     - 新包的 lock 条目会添加到 package-lock.json             │
├─────────────────────────────────────────────────────────────┤
│  7. 恢复所有原始文件                                          │
│     - package.json（原始内容 + 新增依赖）                    │
│     - package-lock.json（原始内容 + 新增 lock 条目）         │
│     - .npmrc（恢复原始 registry）                            │
│     - yarn.lock（恢复原始内容）                              │
└─────────────────────────────────────────────────────────────┘
```

## 许可证

MIT
