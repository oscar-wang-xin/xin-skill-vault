import * as AppService from "../../bindings/xinskillvault/internal/app/service";
import type { AppInfo, SaveConfigResult, VaultSummary } from "../../bindings/xinskillvault/internal/app/models";
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
    backendProbe = AppService.Timestamp().then(() => true).catch(() => false);
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

export function formatBytes(bytes: number): string {
  if (!bytes) return "0 B";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
