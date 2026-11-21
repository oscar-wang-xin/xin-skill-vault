# Xin Skill Vault

**One vault for every AI agent skill on your machine.**
一个仓库，统一管理本机所有 AI Agent 的 skill：导入、编辑、分组、分发、追踪、回滚。

> 当前阶段：**M1 进行中**：SKILL.md 解析、目录/ZIP 导入、技能编辑与快照、技能列表/详情已接通；发现扫描与分发待接入。

---

## 为什么

Claude Code、Codex、OpenCode、WorkBuddy、DSH、Alma、Cherry Studio、CodeBuddy……每个工具都有自己的 skill 目录。同一个技能被复制五份，改一处忘四处；谁装了什么、版本是否一致、目标目录是否被改过，全靠记忆。

Xin Skill Vault 把这些目录收拢成一个本地仓库：

- **仓库是主干**：`~/.skill-repo` 里每个技能只有一份实体。
- **Agent 目录是部署目标**：链接或复制，状态可追踪、可回滚。
- **文件系统是事实来源**：SQLite 只是索引，删库可无损重建。

## 核心能力

| | |
| --- | --- |
| 多来源导入 | 扫描本机 Agent、市场下载、URL、ZIP、本地目录（v0.1 不提供新建） |
| 组合与标签 | 组合（新建/编辑/搜索/排序/一键分发）、自由标签、笔记（双链）、Bundle 合并、打包 `.skill` |
| 提示词 | 每个版本保存为独立 Markdown 文档（`.md`），可分类、复制、删除、回溯；SQLite 只做索引 |
| 分发 | 技能库 / 详情统一弹层；Agent / 项目 / 目录三类目标；链接（推荐）/ 复制 |
| Agent 管理 | 左右两列：Agent 列表 + 技能表；批量导入/删除、链接/复制切换确认 |
| 漂移检测 | 全量 + 增量扫描，识别新增 / 变更 / 缺失 / 未纳管 / **源地址变更** |
| 市场 | skills.sh、SkillsMP、SkillHub 三个预置源 + 自定义源 |
| 审计 | 全操作日志、版本快照、回收站 |
| 更新 | GitHub Releases 驱动，覆盖安装并保留仓库与配置 |
| 跨平台 | Windows / macOS / Linux，Wails v3 + Go + Vue 3 |

## 关键约束

- **产品名**：Xin Skill Vault（v0.1.0），CLI 二进制 `xsv`（别名 `xinskillvault`）。
- **品牌**：黑底白字「XIN」字标（X 放大、渐变底、立体字）+ 机甲橙浮动小火箭；托盘同款，更新/扫描时叠加扫描高光。
- **不写 PATH**：CLI 随安装包落在安装目录，应用内以绝对路径调用；设置页可复制路径，是否加入 PATH 由用户决定。
- **仓库位置可配置**：默认 `%USERPROFILE%\.skill-repo`，修改时提示「移动 / 复制 / 仅切换」，移动模式会校验并重建全部分发链接。
- **Cherry Studio**：只适配最新版（实测 v1.9.12），读取 `version.log` 校验并标注已验证版本；版本不符时降级为自定义目录。
- **分配地址可追踪**：每次分发记录源路径、源指纹与来源 URL，源地址变化时提示更新，不静默失联。
- **发布物**：Windows `exe` 安装版 + 免安装 `zip`；macOS / Linux 通过 GitHub Release 发布。

## 目录结构

```
xin-skill-vault/
├── README.md
├── change.md               # 变更记录
├── brand/
│   ├── logo.svg             # 主标识（含金属小火箭）
│   ├── logo-small.svg       # ≤24px 版本（无火箭）
│   ├── xin-skill-vault.ico  # Windows 图标（多尺寸）
│   ├── xin-skill-vault.icns # macOS 图标
│   └── icons/               # 16–1024 PNG + preview.png
├── docs/
│   ├── dev-plan.md          # 开发计划：功能规格、架构、数据模型、路线图
│   └── review.md            # 原型评审、数据契约、开发就绪清单
└── prototype/
    ├── index.html           # 高保真原型入口（双击即可打开）
    └── assets/
        ├── styles.css       # 设计系统 + 6 套主题
        ├── data.js          # 原型 mock 数据
        └── app.js           # 交互逻辑
```

## 看原型

直接双击 `prototype/index.html`，或在 Alma 侧边栏打开 **Preview**。

原型是纯静态、零依赖的，当前覆盖：

- **技能库**：Skill 仓库（列表/卡片、排序、多选分发、来源与状态列）、发现（平铺扫描结果 + 分类）、组合
- **市场**：三源筛选、搜索、安装、管理仓库
- **Agent**：左右两列，左列可拖宽；右侧技能表支持批量导入/删除、链接/复制切换、详情抽屉
- **项目**：左右两列（可拖宽），项目技能表 + 按 Agent 分配、链接/复制
- **提示词**：左侧可编辑分类（父级 / 目录 / 拖动排序）+ 右侧卡片；单击打开 80vw 编辑器（左 Markdown 表单 / 右版本历史），切换版本前提示保存
- **组合（技能库页签）、设置**：6 套主题 + 拟态/线性开关、密度、路径

可玩的交互：`Ctrl/⌘K` 命令面板、`G L/M/D/J/P/G/S` 跳转、主题下拉、拟态开关、行选中、抽屉开合、导入与分发弹层。

## 设计

- **默认拟态**：白色家电按键质感（外凸/内凹双阴影、近白表面、52px 列表行）；顶栏开关可切回 Linear 扁平风格。
- 默认浅色，内置 6 套主题：机甲冷灰（默认）、商务蓝、薄荷绿、阿里橙、葡萄紫、机甲橙黄。
- 信息密度可调（紧凑 / 舒适 / 宽松）；侧栏与 Agent 左列宽度可拖动。

## 路线图

| 里程碑 | 内容 | 估时 |
| --- | --- | --- |
| M0 | 脚手架、SQLite、配置、AppShell、设计系统 | 3 天 |
| M1 | 仓库、技能管理、编辑器 | 7 天 |
| M2 | 扫描、分发、漂移检测 | 8 天 |
| M3 | 市场与更新 | 6 天 |
| M4 | 组合、提示词、标签、笔记、Bundle、打包 | 8 天 |
| M5 | 打磨、i18n、安装包、开源发布 | 5 天 |

详见 [docs/dev-plan.md](docs/dev-plan.md)、[docs/architecture.md](docs/architecture.md) 与 [docs/review.md](docs/review.md)。


## 开发

### 环境

- Go 1.26+
- Node.js 20+
- Wails v3.1.0（`wails3` 已在 PATH）

### 常用命令

```bash
# 启动桌面开发模式（Go 热重载 + Vite）
wails3 task dev

# 只启动前端（浏览器预览，自动使用 demo 配置）
cd frontend && npm run dev

# 生成 Wails TypeScript 绑定
wails3 generate bindings -ts -i -d frontend/bindings ./...

# Go 测试 / 构建
go test ./...
wails3 build
```

### M1 已接入

- **SKILL.md 解析**：YAML frontmatter + Markdown 正文，校验 name / description / slug。
- **目录 / ZIP 导入**：原生选择器 → 校验 → 复制到 `<vault>/skills/<slug>/` → 计算 SHA256 指纹 → 写入 SQLite；ZIP 带 zip-slip 与体积限制。
- **技能列表**：真实 `ListSkills` 数据，支持搜索、状态筛选、名称/时间/来源排序、列表/卡片切换、多选。
- **技能详情**：右侧抽屉显示描述、路径、指纹、大小、文件列表，可打开技能目录。
- **SKILL.md 编辑**：详情内直接编辑，保存前自动写旧版本快照；显示历史快照列表，可恢复/删除。
- **技能生命周期**：启用/禁用；删除移动到 `.skillrepo/trash` 并软删除索引（30 天保留约定）。
- **示例技能**：`examples/demo-skill/`，可直接在应用里导入试用。
- **尚未完成**：Agent 目录扫描 / Finding、URL 导入、回收站恢复界面、分发与项目/提示词实际数据。

### M0 已落地

- Wails v3 + Vue 3 + TypeScript + Vite + Pinia + Vue Router。
- `internal/config`：配置目录 `%APPDATA%\XinSkillVault`，默认仓库 `~/.skill-repo`。
- `internal/store`：SQLite 迁移框架，`PRAGMA user_version`，迁移前备份；索引位于 `<vault>/.skillrepo/vault.db`。
- `internal/app`：配置、仓库概览、打开目录等 Wails 服务。
- 日志：`<vault>/.skillrepo/logs/app.log`（JSON lines）。
- AppShell：侧栏、顶栏、主题下拉、拟态开关、命令面板、6 个页面路由。
- 浏览器预览无 Wails 后端时自动使用 demo 配置，不写真实文件。

### 试用

启动 `wails3 task dev` 后，在技能库点击「导入」，选择仓库下的 `examples/demo-skill/`。导入完成后应看到 `Demo Skill` 卡片，并在详情抽屉看到 `SKILL.md` 与 `references/checklist.md`。

### 目录

```text
main.go                 Wails 入口与依赖装配
internal/config         配置文件读写与默认值
internal/store          SQLite 打开、迁移、索引查询
internal/app            Wails 服务、目录布局、打开文件夹
internal/logx           结构化日志
frontend/src            Vue AppShell、路由、页面、样式
examples/               可导入的示例 SKILL.md
prototype/              高保真原型（设计基线）
docs/                   开发计划、架构、评审报告
```

## 参考

- [SkillsGate](https://github.com/skillsgate/skillsgate) — 多 Agent 安装与 skills.sh 集成
- [Skills Manager](https://github.com/xingkongliang/skills-manager) — Preset、工作区与 Git 备份

## License

MIT（待仓库初始化时添加）
