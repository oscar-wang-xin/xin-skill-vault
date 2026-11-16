export type SkinID = 'cold-white' | 'cold-gray' | 'mech-yellow' | 'paper' | 'paper-dark' | 'system'

export interface Stats {
  skills: number
  links: number
  copies: number
  enabledAgents: number
  detectedAgents: number
  broken: number
  updates: number
  vaultSizeBytes: number
}

export interface VaultInfo {
  path: string
  appVersion: string
  platform: string
  lastScanAt: string
}

export interface Skill {
  id: string
  name: string
  description: string
  notes: string
  tags: string[]
  groupId: string
  groupName: string
  sourceId: string
  sourceName: string
  vaultPath: string
  contentHash: string
  content: 'executable' | 'script' | 'none' | string
  contentFiles: string[]
  version: string
  sizeBytes: number
  fileCount: number
  installedAt: string
  updatedAt: string
  updateAvailable: boolean
  agents: string[]
  linkCount: number
  copyCount: number
  brokenCount: number
}

export interface Agent {
  id: string
  name: string
  kind: 'agent' | 'project' | string
  skillsPath: string
  configPath: string
  detected: boolean
  enabled: boolean
  installed: number
}

export interface SkillGroup {
  id: string
  name: string
  skillCount: number
  sortOrder: number
}

export interface Source {
  id: string
  name: string
  kind: string
  url: string
  builtin: boolean
  createdAt: string
  syncedAt: string
  skillCount: number
}

export interface Distribution {
  skillId: string
  agentId: string
  mode: 'junction' | 'symlink' | 'copy' | string
  linkPath: string
  status: 'ok' | 'broken' | 'drifted' | 'missing' | string
  contentHash: string
  updatedAt: string
}

export interface Activity {
  id: number
  action: string
  subject: string
  detail: string
  createdAt: string
}

export interface Snapshot {
  skills: Skill[]
  agents: Agent[]
  sources: Source[]
  groups: SkillGroup[]
  distributions: Distribution[]
  activity: Activity[]
  settings: Record<string, string>
  stats: Stats
  vault: VaultInfo
  generatedAt: string
}

export interface MutationResult {
  snapshot: Snapshot
  warnings: string[]
  message: string
}

export interface Inspection {
  name: string
  description: string
  slug: string
  version: string
  license: string
  path: string
  content: string
  contentFiles: string[]
  sizeBytes: number
  fileCount: number
  hasSkillMd: boolean
  alreadyInstalled?: boolean
}

export interface Importable {
  path: string
  name: string
  description: string
  slug: string
  content: string
  contentFiles: string[]
  sizeBytes: number
  fileCount: number
  agents: string[]
  alreadyInstalled: boolean
  isLink: boolean
  sourceId: string
  sourceRef: string
}

export interface ImportItem {
  path: string
  sourceId: string
  sourceRef: string
}

export interface ImportManyRequest {
  items: ImportItem[]
  agentIds: string[]
  mode: string
}

export interface DirectoryEntry {
  name: string
  displayName: string
  description: string
  path: string
  sourcePath: string
  linkTarget: string
  kind: string
  status: string
  mode: string
  content: string
  contentFiles: string[]
  sizeBytes: number
  fileCount: number
  isLink: boolean
  managed: boolean
  vaultSkillId: string
  alreadyInVault: boolean
  distributionPath: string
}

export interface DirectoryView {
  target: Agent
  path: string
  entries: DirectoryEntry[]
  managed: number
  external: number
  broken: number
}

export interface SourceInput {
  name: string
  kind: string
  url: string
}

export interface ProjectInput {
  name: string
  skillsPath: string
}

export interface SkillMetaUpdate {
  skillId: string
  notes: string
  groupId: string
  tags: string[]
}

export interface FileEntry {
  path: string
  size: number
  modTime: string
}

export interface DistributeRequest {
  skillId: string
  agentIds: string[]
  mode: string
}

export interface ImportRequest {
  path: string
  agentIds: string[]
  mode: string
}

export const EMPTY_STATS: Stats = {
  skills: 0,
  links: 0,
  copies: 0,
  enabledAgents: 0,
  detectedAgents: 0,
  broken: 0,
  updates: 0,
  vaultSizeBytes: 0,
}

export const EMPTY_SNAPSHOT: Snapshot = {
  skills: [],
  agents: [],
  sources: [],
  groups: [],
  distributions: [],
  activity: [],
  settings: {},
  stats: EMPTY_STATS,
  vault: {
    path: '',
    appVersion: '',
    platform: '',
    lastScanAt: '',
  },
  generatedAt: '',
}

export interface SkinDef {
  id: SkinID
  name: string
  en: string
  colors: [string, string, string]
  hint: string
}

export const SKINS: SkinDef[] = [
  { id: 'cold-white', name: '冷白机甲', en: 'COLD WHITE', colors: ['#E8EEF3', '#F7FAFC', '#1467D8'], hint: '冷白装甲 · 蓝色信号 · 默认' },
  { id: 'cold-gray', name: '冷灰白', en: 'COLD GRAY', colors: ['#E6E8E8', '#F5F6F6', '#2F3A40'], hint: '中性冷灰 · 石墨强调' },
  { id: 'mech-yellow', name: '机械冷黄', en: 'MECH YELLOW', colors: ['#181A17', '#272B23', '#D9B53C'], hint: '枪灰底 · 冷工业黄' },
  { id: 'paper', name: '纸张', en: 'PAPER', colors: ['#F3EFE6', '#FBF8F1', '#C24E2E'], hint: '暖纸 · 墨色 · 朱砂' },
  { id: 'paper-dark', name: '夜读', en: 'NIGHT', colors: ['#161713', '#24261F', '#E0714C'], hint: '暖炭色夜间模式' },
  { id: 'system', name: '跟随系统', en: 'SYSTEM', colors: ['#E9E7E1', '#2A2A26', '#C24E2E'], hint: '在纸张 / 夜读间切换' },
]

export const SKIN_MAP: Record<string, SkinDef> = Object.fromEntries(SKINS.map((s) => [s.id, s]))

export function resolvedSkin(skin: string): string {
  if (skin === 'system') {
    return typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches
      ? 'paper-dark'
      : 'paper'
  }
  return SKIN_MAP[skin] ? skin : 'cold-white'
}
