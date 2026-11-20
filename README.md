# Xin Skill Vault

**One vault for every AI agent skill on your machine.**
一个仓库，统一管理本机所有 AI Agent 的 skill：导入、编辑、分组、分发、追踪、回滚。

> 当前阶段：**规划 + 高保真原型**，尚未开始编码。

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
| 多来源导入 | 扫描本机 Agent、市场下载、URL、ZIP、本地目录、新建 |
| 组织 | 分组、智能分组、自由标签、笔记（双链）、Bundle 合并、打包 `.skill` |
| 提示词 | 卡片管理提示词与模板，一键生成自建 skill（默认 skill-creator） |
| 分发 | 单个 / 批量 / 分组；软链 / Junction / 复制；事务与回滚 |
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
│   └── dev-plan.md          # 开发计划：功能规格、架构、数据模型、路线图
└── prototype/
    ├── index.html           # 高保真原型入口（双击即可打开）
    └── assets/
        ├── styles.css       # 设计系统 + 6 套主题
        ├── data.js          # 原型 mock 数据
        └── app.js           # 交互逻辑
```

## 看原型

直接双击 `prototype/index.html`，或在 Alma 侧边栏打开 **Preview**。

原型是纯静态、零依赖的，覆盖：

- 概览（统计、分发覆盖率、活动、健康检查）
- 技能库（列表 / 网格、筛选、搜索、右侧 Inspector 五个页签）
- 分发矩阵（技能 × Agent，单元格三态切换、批量操作条）
- 发现（扫描结果分组、导入 / 更新 / 忽略）
- 市场（多来源、安装态、添加市场）
- 分组与预设、活动日志、设置（6 套主题可实时切换）

可玩的交互：`⌘K` 命令面板、`G` 后接 `L/D/F/M` 跳转、主题切换、行选中、分发开关、导入弹层。

## 设计

- Linear 风格：13px 正文、1px 边框、极轻阴影、紧凑 8pt 栅格。
- 默认浅色，内置 6 套主题：机甲冷灰（默认）、商务蓝、薄荷绿、阿里橙、葡萄紫、机甲橙黄。
- 信息密度可调（紧凑 / 舒适 / 宽松）。

## 路线图

| 里程碑 | 内容 | 估时 |
| --- | --- | --- |
| M0 | 脚手架、CI、设计系统 | 3 天 |
| M1 | 仓库、技能管理、编辑器 | 7 天 |
| M2 | 扫描、分发、漂移检测 | 8 天 |
| M3 | 市场与更新 | 6 天 |
| M4 | 分组、笔记、Bundle、打包 | 5 天 |
| M5 | 打磨、i18n、安装包、开源发布 | 5 天 |

详见 [docs/dev-plan.md](docs/dev-plan.md)。

## 参考

- [SkillsGate](https://github.com/skillsgate/skillsgate) — 多 Agent 安装与 skills.sh 集成
- [Skills Manager](https://github.com/xingkongliang/skills-manager) — Preset、工作区与 Git 备份

## License

MIT（待仓库初始化时添加）
