# 架构说明

> 对应 Xin Skill Vault v0.1.0

## 1. 运行时结构

```text
Wails v3 application
├── main.go                    依赖装配、窗口、生命周期
├── internal/config            配置读写（%APPDATA%/XinSkillVault/config.json）
├── internal/store             SQLite 打开、PRAGMA、迁移、索引查询
├── internal/app               Wails 服务、目录布局、打开文件夹
├── internal/logx              JSON 日志（vault/.skillrepo/logs/app.log）
├── internal/version           应用名与版本
└── frontend/                  Vue 3 + TS + Vite + Pinia + Router
    ├── src/components         AppShell / Sidebar / CommandPalette / Icon
    ├── src/views              6 个一级页面
    ├── src/stores/ui.ts       主题、拟态、侧栏折叠
    └── src/lib/api.ts         Wails 绑定 + 浏览器 demo fallback
```

## 2. 启动顺序

1. 读取用户配置目录，加载 `config.json`；不存在时写入默认值。
2. 创建仓库目录：`skills/`、`bundles/`、`.skillrepo/{cache,trash,logs}` 与提示词根目录。
3. 打开 `<vault>/.skillrepo/vault.db`，设置 WAL / foreign_keys / busy_timeout。
4. 执行 `PRAGMA user_version` 迁移；升级前 checkpoint 并备份数据库。
5. 注册 `AppService`、创建窗口、启动 Wails。
6. 前端启动时调用 `GetAppInfo` / `GetVaultSummary`；浏览器预览无后端时使用 demo 配置。

## 3. 文件与数据库边界

- **文件系统是事实来源**：技能目录、提示词 `.md`、版本快照都落盘。
- **SQLite 是索引**：迁移可从文件重建；数据库删除不应丢业务数据。
- **配置**：`config.json` 只保存应用偏好与路径，不保存业务实体。
- **数据库位置**：`<vault>/.skillrepo/vault.db`，与仓库一起可迁移/备份。
- **迁移**：`internal/store/migrations/NNN_name.sql`，按文件名排序，`user_version` 记录版本；升级前保留 `.bak.<时间>`。

## 4. Wails 服务

`internal/app.Service` 已绑定：

| 方法 | 用途 |
| --- | --- |
| `GetAppInfo` | 名称、版本、平台、配置/数据/仓库/数据库路径、schema 版本 |
| `GetConfig` | 读取当前配置 |
| `SaveConfig` | 校验主题、创建目录、原子写配置；仓库路径变更返回 `restartRequired` |
| `GetVaultSummary` | 技能/目标/分配/提示词/分类计数、数据库大小、schema |
| `OpenVaultFolder` / `OpenPromptFolder` / `OpenConfigFolder` | 系统文件管理器打开目录 |
| `ListSkills` / `GetSkillDetail` | 仓库技能列表与详情（含文件清单、快照数） |
| `PickSkillDirectory` / `ImportSkillDirectory` | 原生目录选择与目录导入 |
| `PickSkillArchive` / `ImportSkillArchive` | `.zip` / `.skill` 选择与导入 |
| `ReadSkillDocument` / `SaveSkillDocument` | 读取/保存 SKILL.md，保存前写快照 |
| `ListSkillSnapshots` | 技能历史快照列表 |
| `SetSkillEnabled` / `DeleteSkill` | 启用/禁用、移动到回收站、软删除索引 |
| `RestoreSkillSnapshot` / `DeleteSkillSnapshot` | 恢复/删除历史快照（恢复前会再写快照） |
| `OpenSkillFolder` | 打开单个技能目录 |
| `ScanAgents` | 检测 Agent 目录、扫描技能并返回 Finding |
| `ImportScannedSkill` / `UpdateScannedSkill` | 从扫描结果导入/覆盖更新（覆盖前写完整目录 zip 快照） |
| `OpenAgentFolder` | 打开 Agent skill 目录 |
| `ListDistributionTargets` | 确保 Agent target 入库，返回检测/已分发状态 |
| `DistributeSkill` / `UninstallSkill` | 链接（失败降级复制）或复制分发；移除安装 |
| `ListSkillAssignments` | 技能的分发记录 |
| `CheckDrift` / `RepairDrift` / `RepairAllDrift` | 检测安装件缺失/修改/仓库更新，单条或批量修复；复制件先备份 |
| `ListProjects` / `CreateProject` / `UpdateProject` / `DeleteProject` | 项目 CRUD，移除项目不删除项目目录 |
| `SetProjectAgents` | 项目级 Agent 配置，维护 `targets(kind='project')` |
| `ListProjectSkills` / `AddProjectSkills` / `RemoveProjectSkill` | 项目技能聚合、批量安装与移除 |
| `PickProjectDirectory` / `OpenProjectFolder` | 项目目录选择与打开 |
| `ListMarketSources` | 内置市场源与健康/限制说明 |
| `SearchMarket` / `InstallMarketSkill` | SkillHub / skills.sh 搜索、分页、缓存、分类、审计后导入，来源记为 market |
| `ListMarketCategories` / `AuditMarketSkill` | SkillHub 分类；安装包脚本/危险命令审计 |
| `ListCombos` / `CreateCombo` / `UpdateCombo` / `DeleteCombo` | 组合 CRUD 与技能引用 |
| `ListComboSkills` | 组合内技能，用于一键分发 |
| `ListPromptCategories` / `CreatePromptCategory` / `UpdatePromptCategory` / `DeletePromptCategory` | 提示词分类与目录 |
| `ListPrompts` / `CreatePrompt` / `GetPrompt` / `SavePrompt` / `DeletePrompt` | 提示词与 `.md` 版本文件 |
| `GetPromptVersion` / `DeletePromptVersion` | 读取/删除版本，当前版本删除后自动切换 |
| `ReorderPromptCategories` | 同一父级下的分类排序（服务端校验不可跨级） |
| `ListTags` / `SetSkillTags` | 标签词典与技能标签 |
| `RenameTag` / `DeleteTag` / `MergeTags` | 标签词典维护与合并（UI 已接重命名/删除/合并） |
| `GetSkillNote` / `SaveSkillNote` | 技能笔记读写 |
| `ExportSkillPackage` / `ExportSkillsBundle` | 导出单个 `.skill` 或多技能 Bundle zip |
| `ExportDiagnostics` | 导出诊断 zip：诊断 JSON、配置、日志、README |
| `PickBundleArchive` / `ImportBundleArchive` | 选择并导入 Bundle：解析 bundle.json、逐个技能入库 |
| `ListBundles` / `CreateBundle` / `UpdateBundle` / `DeleteBundle` | 持久化 Bundle CRUD |
| `ListBundleSkills` / `ExportBundle` | Bundle 技能引用与重复导出 |
| `Timestamp` | 前端探测真实后端是否可用 |

绑定由以下命令生成，禁止手改：

```bash
wails3 generate bindings -ts -i -d frontend/bindings ./...
```

## 5. 前端边界

- Vue 3 + TypeScript + Vite + Pinia + Vue Router。
- 页面：技能库、市场、Agent、项目、提示词、设置。
- 技能库包含 Skill 仓库 / 发现 / 组合三个页签。
- Wails 绑定由 `wails3 generate bindings` 生成，不手改。
- 浏览器预览会探测 Wails 后端；没有后端时使用 demo 配置，不写真实文件。

## 6. 安全边界

- ZIP / Bundle 导入拒绝绝对路径、`..` 与盘符路径。
- ZIP 大小限制 200 MB，解压大小限制 512 MB。
- 技能指纹忽略 `.git`、`node_modules`、`.skillrepo-manifest.json` 等元数据。
- 链接分发按 junction → symlink → copy 降级；junction 删除只移除 reparse point。
- 市场安装前审计脚本文件与危险命令；高风险安装需要显式 force。
- 配置与诊断包不包含技能正文；诊断日志最多读取 2 MB。

## 7. 数据与恢复

- 文件系统是事实来源，SQLite 索引可从文件重建。
- 迁移按 `NNN_name.sql` 排序，升级前 checkpoint 并备份数据库。
- 技能编辑与覆盖更新先写快照；复制件漂移修复前写 `.skillrepo/backups`。
- 删除技能移动到 `.skillrepo/trash`，只软删除索引。
