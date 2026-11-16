import { mockCall } from './mock'
import type {
  DirectoryView,
  FileEntry,
  Importable,
  ImportManyRequest,
  ImportRequest,
  Inspection,
  MutationResult,
  ProjectInput,
  SkillMetaUpdate,
  Snapshot,
  SourceInput,
} from './types'

// The Wails host injects `window._wails.environment` before application
// scripts run in most builds. WebView2 exposes `chrome.webview` even earlier,
// which is the reliable Windows signal; macOS/Linux use the WKWebView bridge.
export const isDesktop =
  typeof window !== 'undefined' &&
  (!!(window as any)._wails?.environment ||
    !!(window as any).chrome?.webview ||
    !!(window as any).webkit?.messageHandlers?.external)

function cleanError(err: unknown): Error {
  const raw = err instanceof Error ? err.message : String(err)
  const message = raw
    .replace(/^RuntimeError:\s*/i, '')
    .replace(/^Error:\s*/i, '')
    .trim()
  return new Error(message || '操作失败')
}

const SERVICE = 'github.com/xin-skill-vault/xin-skill-vault/internal/service.AppService'
// Keep SERVICE in sync with go.mod's module path and the internal/service
// package. Wails resolves dynamic calls by fully-qualified method name.

async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  if (!isDesktop) return mockCall(method, args) as Promise<T>
  try {
    const { Call } = await import('@wailsio/runtime')
    return (await Call.ByName(`${SERVICE}.${method}`, ...args)) as T
  } catch (err) {
    throw cleanError(err)
  }
}

export const api = {
  snapshot: () => call<Snapshot>('Snapshot'),
  refresh: () => call<MutationResult>('Refresh'),
  checkUpdates: () => call<MutationResult>('CheckUpdates'),
  saveSetting: (key: string, value: string) => call<Snapshot>('SaveSetting', key, value),
  setSkin: (id: string) => call<Snapshot>('SetSkin', id),
  setAgentEnabled: (id: string, enabled: boolean) => call<MutationResult>('SetAgentEnabled', id, enabled),
  pickFolder: () => call<string>('PickFolder'),
  pickVaultFolder: () => call<string>('PickVaultFolder'),
  changeVaultPath: (path: string, mode: 'move' | 'copy') =>
    call<MutationResult>('ChangeVaultPath', path, mode),
  inspectPath: (path: string) => call<Inspection>('InspectPath', path),
  importables: () => call<Importable[]>('Importables'),
  discoverFolder: (path: string) => call<Importable[]>('DiscoverFolder', path),
  networkImport: (url: string) => call<Importable[]>('NetworkImport', url),
  importSkill: (req: ImportRequest) => call<MutationResult>('Import', req),
  importSkills: (req: ImportManyRequest) => call<MutationResult>('ImportMany', req),
  distribute: (req: { skillId: string; agentIds: string[]; mode: string }) => call<MutationResult>('Distribute', req),
  distributeMany: (req: { skillIds: string[]; agentIds: string[]; mode: string }) =>
    call<MutationResult>('DistributeMany', req),
  setDistribution: (skillId: string, agentId: string, enabled: boolean) =>
    call<MutationResult>('SetDistribution', skillId, agentId, enabled),
  removeSkill: (skillId: string) => call<MutationResult>('RemoveSkill', skillId),
  repair: () => call<MutationResult>('Repair'),
  skillFiles: (skillId: string) => call<FileEntry[]>('SkillFiles', skillId),
  updateSkillMeta: (req: SkillMetaUpdate) => call<Snapshot>('UpdateSkillMeta', req),
  createGroup: (name: string) => call<Snapshot>('CreateGroup', name),
  renameGroup: (id: string, name: string) => call<Snapshot>('RenameGroup', id, name),
  deleteGroup: (id: string) => call<Snapshot>('DeleteGroup', id),
  addSource: (req: SourceInput) => call<MutationResult>('AddSource', req),
  removeSource: (id: string) => call<MutationResult>('RemoveSource', id),
  addProject: (req: ProjectInput) => call<MutationResult>('AddProject', req),
  removeProject: (id: string) => call<MutationResult>('RemoveProject', id),
  directorySkills: (targetId: string) => call<DirectoryView>('DirectorySkills', targetId),
  removeDirectoryEntry: (targetId: string, name: string) =>
    call<MutationResult>('RemoveDirectoryEntry', targetId, name),
  rebuildDistribution: (targetId: string, skillId: string, mode: string) =>
    call<MutationResult>('RebuildDirectoryDistribution', targetId, skillId, mode),
  openDirectoryEntry: (targetId: string, name: string) =>
    call<void>('OpenDirectoryEntry', targetId, name),
  resolveClose: (action: 'quit' | 'tray', remember: boolean) =>
    call<void>('ResolveClose', action, remember),
  cancelClose: () => call<void>('CancelClose'),
  openVault: () => call<void>('OpenVault'),
  openDataFolder: () => call<void>('OpenDataFolder'),
  openSkillFolder: (skillId: string) => call<void>('OpenSkillFolder', skillId),
  openAgentFolder: (agentId: string) => call<void>('OpenAgentFolder', agentId),
}
