# 架构说明（M0）

> 更新：2026-11-21 · 对应 v0.1.0 M0 骨架

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
| `Timestamp` | 前端探测真实后端是否可用 |

绑定由以下命令生成，禁止手改：

```bash
wails3 generate bindings -ts -i -d frontend/bindings ./...
```

## 5. 前端边界

- **M0 AppShell**：侧栏、顶栏、主题下拉、拟态开关、命令面板、6 个路由。
- **真实数据**：技能库从 `GetVaultSummary` 读取计数；当前为空状态。
- **未接入页面**：市场（M3）、Agent（M2）、项目（M2）、提示词（M4）显示阶段占位，不伪造数据。
- **浏览器预览**：`api.ts` 先调用 `Timestamp` 探测后端；失败时使用 localStorage demo 配置，不写真实文件。
- **主题与拟态**：localStorage 立即生效；设置页保存时写入后端 config。

## 6. M0 验收

- [x] `wails3 build` 产出 Windows 可执行文件
- [x] 配置/仓库/日志/数据库首次启动自动创建
- [x] SQLite 迁移可重复执行，schema v2（含 snapshots）
- [x] Go 单元测试覆盖配置、迁移、服务摘要
- [x] 前端 `vue-tsc` + `vite build`
- [x] 6 个路由、主题/拟态、命令面板、配置保存
- [x] CI 工作流（Go + 前端）

## 7. M1 进度

已接通：

- `internal/skill`：SKILL.md frontmatter/正文解析与校验
- `internal/vault`：目录/ZIP 导入、SHA256 指纹、原子复制、文件清单、SKILL.md 快照保存
- `internal/store/skills.go`：技能创建/按 slug 查询/列表/更新/快照
- Wails 服务：`ListSkills` / `GetSkillDetail` / `PickSkillDirectory` / `ImportSkillDirectory` / `PickSkillArchive` / `ImportSkillArchive` / `ReadSkillDocument` / `SaveSkillDocument` / `ListSkillSnapshots` / `OpenSkillFolder`
- 前端技能库：真实数据、搜索/筛选/排序、列表/卡片、详情抽屉、目录/ZIP 导入、SKILL.md 编辑与快照列表、启停/删除/快照恢复
- 示例：`examples/demo-skill/`

尚未完成：

- Agent 目录扫描与 Finding（M2）
- URL 导入
- 分发、项目、提示词实际数据
- 回收站恢复界面、快照恢复后的可视化 diff

## 8. 下一步（M1 剩余 + M2）

1. SKILL.md 解析与校验。
2. 扫描本机 Agent 目录，生成 Finding。
3. 导入目录/ZIP 到 `skills/<slug>/`，写入指纹与快照。
4. 技能库列表/详情抽屉接真实数据。
5. 搜索、筛选、排序与批量选择。

详细规格见 `docs/dev-plan.md` §4、§16。
