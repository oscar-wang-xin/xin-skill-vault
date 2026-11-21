<p align="center">
  <img src="brand/logo.svg" width="92" alt="Xin Skill Vault" />
</p>

<h1 align="center">Xin Skill Vault</h1>

<p align="center">
  一个本地优先的 AI Agent Skill 中央仓库。<br/>
  导入、编辑、组合、分发、追踪、回滚，全部集中在一个桌面应用里。
</p>

<p align="center">
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/license-MIT-0e9f74" /></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-555" />
  <img alt="Version" src="https://img.shields.io/badge/version-0.1.0-0e9f74" />
</p>

---

## 为什么需要它

Claude Code、Codex、Cursor、OpenCode、WorkBuddy、Alma、Cherry Studio、CodeBuddy……每个 Agent 都有自己的 skill 目录。

同一个技能经常被复制到多个目录：改了一处，忘了另外几处；谁装了什么、版本是否一致、目标目录有没有被改过，全靠记忆。

Xin Skill Vault 把这些目录收拢成一个本地仓库：

- **仓库是主干**：`~/.skill-repo` 中每个技能只有一份实体。
- **Agent 目录是部署目标**：通过 junction、符号链接或复制安装，状态可追踪。
- **文件系统是事实来源**：SQLite 只是可重建的索引。
- **本地优先**：不需要账号，不上传技能内容，不依赖云端服务。

## 功能

### Skill 仓库

- 从本地目录、ZIP、`.skill`、Bundle 导入
- 扫描本机 Agent 目录，发现未导入/有变更的技能
- SKILL.md 查看与编辑，保存前自动生成快照
- 启用/禁用、删除到回收站、历史快照恢复
- SHA256 内容指纹、分发计数、标签与笔记

### 分发与项目

- 单个或批量分发到多个 Agent
- 分发模式：
  - Windows 优先 **junction**
  - 次选 **symlink**
  - 失败自动降级 **copy**
- 项目级 skill 目录，不污染全局 Agent
- 漂移检测：仓库已更新、安装件被修改、安装件缺失
- 单条/批量修复；复制件修复前自动备份
- Agent 页与项目页均支持查看、移除和打开目录

### 组合、Bundle 与打包

- 技能组合：保存技能集合，一键复用分发
- 持久化 Bundle：保存多个技能，重复导出
- 单技能导出 `.skill`（zip + manifest）
- Bundle 导入：一次导入多个技能

### 提示词

- 分类树、自定义保存目录、同级拖动排序
- 每个版本独立保存为 `.md` 文件
- 80vw Markdown 编辑器，左右分栏可拖动
- 版本历史、复制、删除版本、未保存保护

### 市场

只展示实际可调用的源：

- **SkillHub**：搜索、分类、zip 安装
- **skills.sh**：sitemap 搜索、GitHub 仓库安装

带搜索结果缓存、安装前脚本/危险命令审计；高风险安装需要二次确认。

## 支持的 Agent

| Agent | 默认 skill 目录 |
| --- | --- |
| Claude Code | `~/.claude/skills` |
| Codex CLI | `~/.codex/skills` |
| Cursor | `~/.cursor/skills` |
| OpenCode | `~/.config/opencode/skills` |
| WorkBuddy | `~/.workbuddy/skills` |
| DeepSeek Harness | `~/.dsh/skills` |
| Alma | `~/.config/alma/skills` |
| CodeBuddy | `~/.codebuddy/skills` |
| Reasonix | `~/.reasonix/skills` |
| Universal | `~/.agents/skills` |
| Cherry Studio | `%APPDATA%\CherryStudio\Data\Skills` |

检测不到目录的 Agent 不会显示为可分发目标。

## 快速开始

### 直接使用（Windows）

从 [Releases](https://github.com/oscar-wang-xin/xin-skill-vault/releases) 下载：

```text
XinSkillVault-0.1.0-win-x64.zip
```

解压后运行：

```text
xinskillvault.exe
```

免安装，不写系统 PATH。

### 从源码构建

环境要求：

- Go 1.26+
- Node.js 20+
- Wails v3.1.0

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0

# 安装前端依赖
cd frontend && npm ci && cd ..

# 开发模式
wails3 task dev

# 生产构建
wails3 build
```

Windows 免安装包：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/package-windows.ps1 -Version 0.1.0
```

产物：

```text
bin/release/XinSkillVault-0.1.0-win-x64.zip
```

## 数据与隐私

默认位置：

| 内容 | 路径 |
| --- | --- |
| 配置 | `%APPDATA%\XinSkillVault\config.json` |
| 仓库 | `~/.skill-repo` |
| SQLite 索引 | `<vault>/.skillrepo/vault.db` |
| 日志 | `<vault>/.skillrepo/logs/app.log` |
| 提示词 | `~/.xin-skill-vault/prompts` |
| 回收站 | `<vault>/.skillrepo/trash` |
| 修复备份 | `<vault>/.skillrepo/backups` |

- 不收集遥测。
- 不上传技能或提示词内容。
- 市场搜索会访问 SkillHub / skills.sh 及其 GitHub 仓库。
- 诊断包只包含应用信息、配置、日志路径和最近日志，不包含技能正文。

## 安全边界

- ZIP / Bundle 导入有路径穿越（zip-slip）防护与体积限制
- 技能目录内容指纹忽略 `.git`、`node_modules`、manifest 等元数据
- 分发链接失败自动降级为复制
- junction 删除只移除 reparse point，不递归删除目标内容
- 市场安装前检查脚本文件与危险命令
- 高风险市场安装需要显式确认

## 项目结构

```text
main.go                 Wails 入口与依赖装配
internal/
  app/                  Wails 服务、项目、市场、提示词、Bundle
  agent/                Agent 适配器与检测
  config/               配置读写
  distribute/           分发、漂移检测与修复
  market/               SkillHub / skills.sh Provider
  scan/                 Agent 目录扫描
  skill/                SKILL.md 解析、slug、指纹
  skillpack/            .skill / Bundle 打包
  store/                SQLite 迁移与索引
  vault/                仓库导入、编辑、快照、回收站
frontend/               Vue 3 + TypeScript + Vite
examples/demo-skill/    可导入示例技能
brand/                  标识与图标资源
docs/architecture.md    架构说明
scripts/                发布构建脚本
```

## 开发

```bash
# Go 测试
go test ./internal/...

# 前端类型检查与构建
cd frontend && npm run build

# Wails 构建
wails3 build

# 生成绑定（不要手改 frontend/bindings）
wails3 generate bindings -ts -i -d frontend/bindings ./...
```

`main` 是应用开发主线和默认分支；`v0` 保留早期项目主页与历史记录。向 `main` 提交代码。

## 已知限制

- URL 导入尚未接入
- 应用内自动更新尚未接入
- Windows 目前只提供免安装 zip
- 提示词分类暂无跨父级拖动
- 笔记双链尚未接入

## License

[MIT](LICENSE) © 2026 Xin
