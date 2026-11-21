import * as AppService from "../../bindings/xinskillvault/internal/app/service";
import type { AppInfo, DistributionTarget, SaveConfigResult, SkillAssignment, SkillDetail, SkillItem, SnapshotItem, VaultSummary } from "../../bindings/xinskillvault/internal/app/models";
import type { Result as ScanResult } from "../../bindings/xinskillvault/internal/scan/models";
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
  schemaVersion: 1,
};

const demoSummary: VaultSummary = {
  vaultPath: demoConfig.vaultPath,
  promptRoot: demoConfig.promptRoot,
  databasePath: demoInfo.databasePath,
  databaseBytes: 0,
  skillCount: 0,
  targetCount: 0,
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

export function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
