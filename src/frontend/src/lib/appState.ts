import { computed, reactive } from 'vue'
import { api, isDesktop } from './api'
import { EMPTY_SNAPSHOT, SKIN_MAP, resolvedSkin } from './types'
import type {
  ImportItem,
  MutationResult,
  ProjectInput,
  SkillMetaUpdate,
  SkinID,
  Snapshot,
  SourceInput,
} from './types'

export type View = 'library' | 'updates' | 'distribution' | 'directories' | 'sources' | 'settings'
export type Layout = 'table' | 'cards'
export type AddTab = 'local' | 'existing' | 'network'

export interface Toast {
  id: number
  title: string
  detail?: string
  kind: 'success' | 'error' | 'info' | 'warning'
}

export const state = reactive({
  snapshot: { ...EMPTY_SNAPSHOT } as Snapshot,
  loading: true,
  ready: false,
  error: '',
  view: 'library' as View,
  selectedId: null as string | null,
  layout: 'table' as Layout,
  filter: 'all',
  query: '',
  skinMenuOpen: false,
  addOpen: false,
  addTab: 'local' as AddTab,
  paletteOpen: false,
  distributeOpen: false,
  closePromptOpen: false,
  busy: false,
  toasts: [] as Toast[],
})

export const FONT_FAMILIES = [
  {
    id: 'system',
    name: '系统界面字体',
    stack: '"Segoe UI Variable Text","Segoe UI","Microsoft YaHei UI","Microsoft YaHei",system-ui,sans-serif',
  },
  {
    id: 'segoe',
    name: 'Segoe UI',
    stack: '"Segoe UI Variable Text","Segoe UI","Microsoft YaHei UI",system-ui,sans-serif',
  },
  {
    id: 'yahei',
    name: '微软雅黑',
    stack: '"Microsoft YaHei UI","Microsoft YaHei","Segoe UI",system-ui,sans-serif',
  },
  {
    id: 'noto',
    name: 'Noto Sans SC / 思源黑体',
    stack: '"Noto Sans SC","Source Han Sans SC","Microsoft YaHei UI","Segoe UI",sans-serif',
  },
] as const

export const FONT_SCALES = [
  { label: '小', value: 0.95 },
  { label: '标准', value: 1 },
  { label: '大', value: 1.12 },
  { label: '特大', value: 1.28 },
] as const

const FONT_MAP = Object.fromEntries(FONT_FAMILIES.map((font) => [font.id, font.stack]))

export function applyTypography(settings: Record<string, string>) {
  const rawScale = Number.parseFloat(settings['ui.fontScale'] || '1.12')
  const scale = Number.isFinite(rawScale) ? Math.min(1.5, Math.max(0.85, rawScale)) : 1.12
  const familyId = settings['ui.fontFamily'] || 'system'
  const stack = FONT_MAP[familyId] || FONT_MAP.system
  document.documentElement.style.setProperty('--font-scale', String(scale))
  document.documentElement.style.setProperty('--ui-font-family', stack)
  try {
    localStorage.setItem('xsv-font-scale', String(scale))
    localStorage.setItem('xsv-font-family', familyId)
  } catch {
    /* storage may be unavailable */
  }
}

let toastSeq = 0
let initialized = false

export const selectedSkill = computed(() =>
  state.snapshot.skills.find((s) => s.id === state.selectedId) || null,
)

export const currentSkin = computed<SkinID>(() => {
  const saved = state.snapshot.settings['ui.skin']
  if (saved && SKIN_MAP[saved]) return saved as SkinID
  return 'cold-white'
})

export const enabledAgents = computed(() => state.snapshot.agents.filter((a) => a.enabled))
export const detectedAgents = computed(() => state.snapshot.agents.filter((a) => a.detected))

export function applySkin(id: string, persistLocal = true) {
  const resolved = resolvedSkin(id)
  document.documentElement.dataset.skin = resolved
  if (persistLocal) {
    try {
      localStorage.setItem('xsv-skin', id)
    } catch {
      /* storage can be unavailable in hardened webviews */
    }
  }
}

export async function initApp() {
  if (initialized) return
  initialized = true
  const stored = readStoredSkin()
  applySkin(stored || 'cold-white')
  applyTypography({
    'ui.fontFamily': readStoredFont('xsv-font-family', 'system'),
    'ui.fontScale': readStoredFont('xsv-font-scale', '1.12'),
  })
  if (isDesktop) {
    import('@wailsio/runtime').then(({ Events }) => {
      Events.On('close-requested', () => {
        state.closePromptOpen = true
      })
    })
  }
  try {
    const snap = await api.snapshot()
    setSnapshot(snap)
    const skin = snap.settings['ui.skin'] || stored || 'cold-white'
    applySkin(skin)
    state.layout = snap.settings['ui.layout'] === 'cards' ? 'cards' : 'table'
    state.ready = true
    if (typeof window !== 'undefined' && window.matchMedia) {
      window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (currentSkin.value === 'system') applySkin('system', false)
      })
    }
  } catch (err) {
    state.error = errorMessage(err)
  } finally {
    state.loading = false
  }
}

export function setSnapshot(snap: Snapshot) {
  state.snapshot = snap
  applyTypography(snap.settings)
  if (state.selectedId && !snap.skills.some((s) => s.id === state.selectedId)) {
    state.selectedId = null
  }
}

function applyMutation(result: MutationResult, successToast = true) {
  if (result?.snapshot) setSnapshot(result.snapshot)
  if (result?.warnings?.length) {
    for (const warning of result.warnings) pushToast('warning', warning)
  }
  if (successToast && result?.message) pushToast('success', result.message)
}

export async function refresh() {
  if (state.busy) return
  state.busy = true
  try {
    applyMutation(await api.refresh())
  } catch (err) {
    pushToast('error', errorMessage(err))
  } finally {
    state.busy = false
  }
}

export async function checkUpdates() {
  if (state.busy) return
  state.busy = true
  try {
    applyMutation(await api.checkUpdates())
  } catch (err) {
    pushToast('error', errorMessage(err))
  } finally {
    state.busy = false
  }
}

export async function repair() {
  if (state.busy) return
  state.busy = true
  try {
    applyMutation(await api.repair())
  } catch (err) {
    pushToast('error', errorMessage(err))
  } finally {
    state.busy = false
  }
}

export async function setSkin(id: SkinID) {
  applySkin(id)
  try {
    const snap = await api.setSkin(id)
    setSnapshot(snap)
    pushToast('success', `皮肤已切换 · ${SKIN_MAP[id]?.name || id}`)
  } catch (err) {
    pushToast('error', errorMessage(err))
  }
}

export async function saveSetting(key: string, value: string, announce = false) {
  try {
    const snap = await api.saveSetting(key, value)
    setSnapshot(snap)
    if (announce) pushToast('success', '设置已保存')
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  }
}

export async function setLayout(layout: Layout) {
  state.layout = layout
  await saveSetting('ui.layout', layout)
}

export async function setAgentEnabled(id: string, enabled: boolean) {
  if (state.busy) return
  state.busy = true
  try {
    applyMutation(await api.setAgentEnabled(id, enabled))
  } catch (err) {
    pushToast('error', errorMessage(err))
  } finally {
    state.busy = false
  }
}

export async function importSkills(items: ImportItem[], agentIds: string[], mode: string) {
  if (state.busy) return false
  if (items.length === 0) {
    pushToast('warning', '请先选择要导入的技能')
    return false
  }
  state.busy = true
  try {
    applyMutation(await api.importSkills({ items, agentIds, mode }))
    state.addOpen = false
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function distribute(skillId: string, agentIds: string[], mode: string) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.distribute({ skillId, agentIds, mode }))
    state.distributeOpen = false
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function distributeMany(skillIds: string[], agentIds: string[], mode: string) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.distributeMany({ skillIds, agentIds, mode }))
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function setDistribution(skillId: string, agentId: string, enabled: boolean) {
  if (state.busy) return
  state.busy = true
  try {
    applyMutation(await api.setDistribution(skillId, agentId, enabled), false)
    if (!enabled) pushToast('info', '分发已移除')
  } catch (err) {
    pushToast('error', errorMessage(err))
  } finally {
    state.busy = false
  }
}

export async function removeSkill(skillId: string) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.removeSkill(skillId))
    if (state.selectedId === skillId) state.selectedId = null
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function updateSkillMeta(req: SkillMetaUpdate) {
  if (state.busy) return false
  state.busy = true
  try {
    setSnapshot(await api.updateSkillMeta(req))
    pushToast('success', '技能信息已保存')
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function createGroup(name: string) {
  try {
    setSnapshot(await api.createGroup(name))
    pushToast('success', `已创建分组「${name}」`)
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  }
}

export async function renameGroup(id: string, name: string) {
  try {
    setSnapshot(await api.renameGroup(id, name))
    pushToast('success', '分组已重命名')
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  }
}

export async function deleteGroup(id: string) {
  try {
    setSnapshot(await api.deleteGroup(id))
    pushToast('info', '分组已删除，技能回到未分组')
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  }
}

export async function addSource(req: SourceInput) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.addSource(req))
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function removeSource(id: string) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.removeSource(id))
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function addProject(req: ProjectInput) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.addProject(req))
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function removeProject(id: string) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.removeProject(id))
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function removeDirectoryEntry(targetId: string, name: string) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.removeDirectoryEntry(targetId, name))
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function rebuildDistribution(targetId: string, skillId: string, mode: string) {
  if (state.busy) return false
  state.busy = true
  try {
    applyMutation(await api.rebuildDistribution(targetId, skillId, mode))
    return true
  } catch (err) {
    pushToast('error', errorMessage(err))
    return false
  } finally {
    state.busy = false
  }
}

export async function openDirectoryEntry(targetId: string, name: string) {
  try {
    await api.openDirectoryEntry(targetId, name)
  } catch (err) {
    pushToast('error', errorMessage(err))
  }
}

export async function resolveClose(action: 'quit' | 'tray', remember: boolean) {
  try {
    await api.resolveClose(action, remember)
    state.closePromptOpen = false
  } catch (err) {
    pushToast('error', errorMessage(err))
  }
}

export async function cancelClose() {
  try {
    await api.cancelClose()
  } catch (err) {
    pushToast('error', errorMessage(err))
  } finally {
    state.closePromptOpen = false
  }
}

export async function openVault() {
  try {
    await api.openVault()
  } catch (err) {
    pushToast('error', errorMessage(err))
  }
}

export async function openSkillFolder(skillId: string) {
  try {
    await api.openSkillFolder(skillId)
  } catch (err) {
    pushToast('error', errorMessage(err))
  }
}

export async function openAgentFolder(agentId: string) {
  try {
    await api.openAgentFolder(agentId)
  } catch (err) {
    pushToast('error', errorMessage(err))
  }
}

export function navigate(view: View) {
  state.view = view
  state.skinMenuOpen = false
  state.paletteOpen = false
}

export function selectSkill(id: string | null) {
  state.selectedId = id
}

export function openAdd(tab: AddTab = 'local') {
  state.addTab = tab
  state.addOpen = true
}

export function pushToast(kind: Toast['kind'], title: string, detail?: string) {
  const toast: Toast = { id: ++toastSeq, kind, title, detail }
  state.toasts.push(toast)
  window.setTimeout(() => dismissToast(toast.id), kind === 'error' ? 6500 : 3800)
}

export function dismissToast(id: number) {
  const idx = state.toasts.findIndex((t) => t.id === id)
  if (idx >= 0) state.toasts.splice(idx, 1)
}

export function errorMessage(err: unknown): string {
  if (err instanceof Error) return err.message
  return String(err)
}

function readStoredSkin(): string | null {
  try {
    const value = localStorage.getItem('xsv-skin')
    if (value && SKIN_MAP[value]) return value
    const legacy = localStorage.getItem('xsv-theme')
    if (legacy === 'dark') return 'paper-dark'
    if (legacy === 'light') return 'paper'
  } catch {
    /* ignore */
  }
  return null
}

function readStoredFont(key: string, fallback: string): string {
  try {
    return localStorage.getItem(key) || fallback
  } catch {
    return fallback
  }
}
