import * as AppService from "../../bindings/xinskillvault/internal/app/service";
import type { AppInfo, BundleImportResult, ComboItem, ComboSkillItem, DistributionTarget, ProjectBulkResult, ProjectItem, ProjectSkill, PromptCategoryItem, PromptDetail, PromptItem, SaveConfigResult, SkillAssignment, SkillDetail, SkillItem, SnapshotItem, TagItem, VaultSummary } from "../../bindings/xinskillvault/internal/app/models";
import type { Result as ScanResult } from "../../bindings/xinskillvault/internal/scan/models";
import type { Report as DriftReport } from "../../bindings/xinskillvault/internal/distribute/models";
import type { Audit as MarketAudit, Category as MarketCategory, Result as MarketResult, Source as MarketSource } from "../../bindings/xinskillvault/internal/market/models";
import type { Config } from "../../bindings/xinskillvault/internal/config/models";

const CONFIG_KEY = "xsv-demo-config";

const demoConfig: Config = {
  vaultPath: "~/.skill-repo",
  promptRoot: "~/.xin-skill-vault/prompts",
  theme: "mech-gray",
  neumorph: true,
  language: "zh-CN",
};

const demoInfo: AppInfo = {
  name: "Xin Skill Vault",
  version: "0.1.0",
  platform: "preview",
  configPath: "~/.config/XinSkillVault/config.json",
  dataDir: "~/.config/XinSkillVault",
  vaultPath: demoConfig.vaultPath,
  promptRoot: demoConfig.promptRoot,
  databasePath: "~/.skill-repo/.skillrepo/vault.db",
  schemaVersion: 3,
};

const demoSummary: VaultSummary = {
  vaultPath: demoConfig.vaultPath,
  promptRoot: demoConfig.promptRoot,
  databasePath: demoInfo.databasePath,
  databaseBytes: 0,
  skillCount: 0,
  targetCount: 0,
  projectCount: 0,
  assignmentCount: 0,
  promptCount: 0,
  categoryCount: 0,
  schemaVersion: 1,
};

let backendProbe: Promise<boolean> | null = null;

async function hasBackend(): Promise<boolean> {
  if (typeof window === "undefined" || !(window as unknown as { _wails?: unknown })._wails) return false;
  if (!backendProbe) {
    const probe = AppService.Timestamp().then(() => true).catch(() => false);
    const timeout = new Promise<boolean>((resolve) => window.setTimeout(() => resolve(false), 1500));
    backendProbe = Promise.race([probe, timeout]);
  }
  return backendProbe;
}

function demoConfigFromStorage(): Config {
  try {
    const saved = localStorage.getItem(CONFIG_KEY);
    if (!saved) return { ...demoConfig };
    return { ...demoConfig, ...JSON.parse(saved) } as Config;
  } catch {
    return { ...demoConfig };
  }
}

export async function getAppInfo(): Promise<AppInfo> {
  if (!(await hasBackend())) return { ...demoInfo, ...demoConfigFromStorage(), platform: "preview" };
  try {
    return await AppService.GetAppInfo();
  } catch (error) {
    console.error("GetAppInfo failed", error);
    throw error;
  }
}

export async function getConfig(): Promise<Config> {
  if (!(await hasBackend())) return demoConfigFromStorage();
  return AppService.GetConfig();
}

export async function saveConfig(next: Config): Promise<SaveConfigResult> {
  if (!(await hasBackend())) {
    localStorage.setItem(CONFIG_KEY, JSON.stringify(next));
    return { restartRequired: false };
  }
  return AppService.SaveConfig(next);
}

export async function getVaultSummary(): Promise<VaultSummary> {
  if (!(await hasBackend())) {
    const cfg = demoConfigFromStorage();
    return { ...demoSummary, vaultPath: cfg.vaultPath, promptRoot: cfg.promptRoot };
  }
  return AppService.GetVaultSummary();
}

export async function openVaultFolder(): Promise<void> {
  if (!(await hasBackend())) return;
  await AppService.OpenVaultFolder();
}

export async function openPromptFolder(): Promise<void> {
  if (!(await hasBackend())) return;
  await AppService.OpenPromptFolder();
}

export async function openConfigFolder(): Promise<void> {
  if (!(await hasBackend())) return;
  await AppService.OpenConfigFolder();
}


export async function listSkills(): Promise<SkillItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListSkills()) ?? [];
}

export async function getSkillDetail(slug: string): Promise<SkillDetail> {
  if (!(await hasBackend())) throw new Error("技能详情需要桌面应用后端");
  const detail = await AppService.GetSkillDetail(slug);
  if (!detail) throw new Error("技能不存在");
  return detail;
}

export async function pickSkillDirectory(): Promise<string> {
  if (!(await hasBackend())) throw new Error("导入技能需要桌面应用");
  return AppService.PickSkillDirectory();
}

export async function importSkillDirectory(source: string): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("导入技能需要桌面应用后端");
  return AppService.ImportSkillDirectory(source);
}

export async function openSkillFolder(slug: string): Promise<void> {
  if (!(await hasBackend())) return;
  await AppService.OpenSkillFolder(slug);
}


export async function pickSkillArchive(): Promise<string> {
  if (!(await hasBackend())) throw new Error("导入压缩包需要桌面应用");
  return AppService.PickSkillArchive();
}

export async function importSkillArchive(path: string): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("导入压缩包需要桌面应用");
  return AppService.ImportSkillArchive(path);
}

export async function readSkillDocument(slug: string): Promise<string> {
  if (!(await hasBackend())) throw new Error("编辑技能需要桌面应用");
  return AppService.ReadSkillDocument(slug);
}

export async function saveSkillDocument(slug: string, content: string, note = "编辑前"): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("保存技能需要桌面应用");
  return AppService.SaveSkillDocument(slug, content, note);
}

export async function listSkillSnapshots(slug: string): Promise<SnapshotItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListSkillSnapshots(slug)) ?? [];
}


export async function setSkillEnabled(slug: string, enabled: boolean): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("修改技能状态需要桌面应用");
  return AppService.SetSkillEnabled(slug, enabled);
}

export async function deleteSkill(slug: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("删除技能需要桌面应用");
  await AppService.DeleteSkill(slug);
}

export async function restoreSkillSnapshot(slug: string, snapshotID: string): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("恢复快照需要桌面应用");
  return AppService.RestoreSkillSnapshot(slug, snapshotID);
}

export async function deleteSkillSnapshot(slug: string, snapshotID: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("删除快照需要桌面应用");
  await AppService.DeleteSkillSnapshot(slug, snapshotID);
}


export async function scanAgents(): Promise<ScanResult> {
  if (!(await hasBackend())) {
    return { scannedAt: 0, agents: [], findings: [], errors: [], scannedSkills: 0 };
  }
  return AppService.ScanAgents();
}

export async function importScannedSkill(agentID: string, source: string): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("导入扫描技能需要桌面应用");
  return AppService.ImportScannedSkill(agentID, source);
}

export async function openAgentFolder(agentID: string): Promise<void> {
  if (!(await hasBackend())) return;
  await AppService.OpenAgentFolder(agentID);
}


export async function listDistributionTargets(slug: string): Promise<DistributionTarget[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListDistributionTargets(slug)) ?? [];
}

export async function distributeSkill(slug: string, agentID: string, mode: string): Promise<SkillAssignment> {
  if (!(await hasBackend())) throw new Error("分发技能需要桌面应用");
  return AppService.DistributeSkill(slug, agentID, mode);
}

export async function uninstallSkill(slug: string, agentID: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("移除分发需要桌面应用");
  await AppService.UninstallSkill(slug, agentID);
}

export async function listSkillAssignments(slug: string): Promise<SkillAssignment[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListSkillAssignments(slug)) ?? [];
}


export async function updateScannedSkill(agentID: string, source: string): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("更新扫描技能需要桌面应用");
  return AppService.UpdateScannedSkill(agentID, source);
}


export async function checkDrift(): Promise<DriftReport[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.CheckDrift()) ?? [];
}

export async function repairDrift(slug: string, agentID: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("修复漂移需要桌面应用");
  await AppService.RepairDrift(slug, agentID);
}


export async function listProjects(): Promise<ProjectItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListProjects()) ?? [];
}

export async function createProject(name: string, path: string, description: string, agentIDs: string[]): Promise<ProjectItem> {
  if (!(await hasBackend())) throw new Error("新建项目需要桌面应用");
  return AppService.CreateProject(name, path, description, agentIDs);
}

export async function updateProject(id: string, name: string, description: string): Promise<ProjectItem> {
  if (!(await hasBackend())) throw new Error("修改项目需要桌面应用");
  return AppService.UpdateProject(id, name, description);
}

export async function deleteProject(id: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("删除项目需要桌面应用");
  await AppService.DeleteProject(id);
}

export async function setProjectAgents(id: string, agentIDs: string[]): Promise<ProjectItem> {
  if (!(await hasBackend())) throw new Error("配置项目 Agent 需要桌面应用");
  return AppService.SetProjectAgents(id, agentIDs);
}

export async function listProjectSkills(id: string): Promise<ProjectSkill[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListProjectSkills(id)) ?? [];
}

export async function addProjectSkills(id: string, slugs: string[], mode: string): Promise<ProjectBulkResult> {
  if (!(await hasBackend())) throw new Error("添加项目技能需要桌面应用");
  return AppService.AddProjectSkills(id, slugs, mode);
}

export async function removeProjectSkill(id: string, slug: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("移除项目技能需要桌面应用");
  await AppService.RemoveProjectSkill(id, slug);
}

export async function pickProjectDirectory(): Promise<string> {
  if (!(await hasBackend())) throw new Error("选择项目目录需要桌面应用");
  return AppService.PickProjectDirectory();
}

export async function openProjectFolder(id: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("打开项目目录需要桌面应用");
  await AppService.OpenProjectFolder(id);
}


export async function repairAllDrift(): Promise<{ repaired: number; errors: string[] }> {
  if (!(await hasBackend())) throw new Error("批量修复需要桌面应用");
  const result = await AppService.RepairAllDrift();
  return { repaired: result.repaired, errors: result.errors ?? [] };
}


export async function listMarketSources(): Promise<MarketSource[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListMarketSources()) ?? [];
}

export async function searchMarket(source: string, query: string, category: string, page = 1): Promise<MarketResult> {
  if (!(await hasBackend())) throw new Error("市场搜索需要桌面应用");
  return AppService.SearchMarket(source, query, category, page);
}

export async function listMarketCategories(): Promise<MarketCategory[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListMarketCategories()) ?? [];
}

export async function auditMarketSkill(source: string, slug: string, namespace: string): Promise<MarketAudit> {
  if (!(await hasBackend())) throw new Error("市场审计需要桌面应用");
  return AppService.AuditMarketSkill(source, slug, namespace);
}

export async function installMarketSkill(source: string, slug: string, namespace: string, force = false): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("安装市场技能需要桌面应用");
  return AppService.InstallMarketSkill(source, slug, namespace, force);
}


export async function listCombos(): Promise<ComboItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListCombos()) ?? [];
}

export async function createCombo(name: string, description: string, icon: string, slugs: string[]): Promise<ComboItem> {
  if (!(await hasBackend())) throw new Error("新建组合需要桌面应用");
  return AppService.CreateCombo(name, description, icon, slugs);
}

export async function updateCombo(id: string, name: string, description: string, icon: string, slugs: string[]): Promise<ComboItem> {
  if (!(await hasBackend())) throw new Error("编辑组合需要桌面应用");
  return AppService.UpdateCombo(id, name, description, icon, slugs);
}

export async function deleteCombo(id: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("删除组合需要桌面应用");
  await AppService.DeleteCombo(id);
}

export async function listComboSkills(id: string): Promise<ComboSkillItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListComboSkills(id)) ?? [];
}


export async function listPromptCategories(): Promise<PromptCategoryItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListPromptCategories()) ?? [];
}

export async function createPromptCategory(name: string, parentID: string, dirPath: string): Promise<PromptCategoryItem> {
  if (!(await hasBackend())) throw new Error("新建分类需要桌面应用");
  return AppService.CreatePromptCategory(name, parentID, dirPath);
}

export async function updatePromptCategory(id: string, name: string): Promise<PromptCategoryItem> {
  if (!(await hasBackend())) throw new Error("编辑分类需要桌面应用");
  return AppService.UpdatePromptCategory(id, name);
}

export async function deletePromptCategory(id: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("删除分类需要桌面应用");
  await AppService.DeletePromptCategory(id);
}

export async function listPrompts(category = "all"): Promise<PromptItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListPrompts(category)) ?? [];
}

export async function createPrompt(title: string, categoryID: string, tags: string[], content: string): Promise<PromptDetail> {
  if (!(await hasBackend())) throw new Error("新建提示词需要桌面应用");
  return AppService.CreatePrompt(title, categoryID, tags, content);
}

export async function getPrompt(id: string): Promise<PromptDetail> {
  if (!(await hasBackend())) throw new Error("读取提示词需要桌面应用");
  return AppService.GetPrompt(id);
}

export async function savePrompt(id: string, title: string, categoryID: string, tags: string[], content: string): Promise<PromptDetail> {
  if (!(await hasBackend())) throw new Error("保存提示词需要桌面应用");
  return AppService.SavePrompt(id, title, categoryID, tags, content);
}

export async function getPromptVersion(id: string, version: number): Promise<string> {
  if (!(await hasBackend())) throw new Error("读取版本需要桌面应用");
  return AppService.GetPromptVersion(id, version);
}

export async function deletePromptVersion(id: string, version: number): Promise<PromptDetail> {
  if (!(await hasBackend())) throw new Error("删除版本需要桌面应用");
  return AppService.DeletePromptVersion(id, version);
}

export async function deletePrompt(id: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("删除提示词需要桌面应用");
  await AppService.DeletePrompt(id);
}


export async function listTags(): Promise<TagItem[]> {
  if (!(await hasBackend())) return [];
  return (await AppService.ListTags()) ?? [];
}

export async function setSkillTags(slug: string, tags: string[]): Promise<SkillItem> {
  if (!(await hasBackend())) throw new Error("编辑标签需要桌面应用");
  return AppService.SetSkillTags(slug, tags);
}

export async function getSkillNote(slug: string): Promise<string> {
  if (!(await hasBackend())) return "";
  return AppService.GetSkillNote(slug);
}

export async function saveSkillNote(slug: string, body: string): Promise<string> {
  if (!(await hasBackend())) throw new Error("保存笔记需要桌面应用");
  return AppService.SaveSkillNote(slug, body);
}


export async function exportSkillPackage(slug: string): Promise<string> {
  if (!(await hasBackend())) throw new Error("导出 .skill 需要桌面应用");
  return AppService.ExportSkillPackage(slug);
}


export async function renameTag(id: string, name: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("重命名标签需要桌面应用");
  await AppService.RenameTag(id, name);
}

export async function deleteTag(id: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("删除标签需要桌面应用");
  await AppService.DeleteTag(id);
}

export async function mergeTags(sourceID: string, targetID: string): Promise<void> {
  if (!(await hasBackend())) throw new Error("合并标签需要桌面应用");
  await AppService.MergeTags(sourceID, targetID);
}


export async function exportSkillsBundle(name: string, slugs: string[]): Promise<string> {
  if (!(await hasBackend())) throw new Error("导出 Bundle 需要桌面应用");
  return AppService.ExportSkillsBundle(name, slugs);
}


export async function pickBundleArchive(): Promise<string> {
  if (!(await hasBackend())) throw new Error("导入 Bundle 需要桌面应用");
  return AppService.PickBundleArchive();
}

export async function importBundleArchive(path: string): Promise<BundleImportResult> {
  if (!(await hasBackend())) throw new Error("导入 Bundle 需要桌面应用");
  return AppService.ImportBundleArchive(path);
}

export function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
