import type {
  Activity,
  Agent,
  DirectoryEntry,
  DirectoryView,
  Distribution,
  FileEntry,
  Importable,
  Inspection,
  MutationResult,
  Skill,
  SkillGroup,
  Snapshot,
  Source,
  SourceInput,
  ProjectInput,
  SkillMetaUpdate,
  Stats,
} from './types'

const now = () => new Date().toISOString()
const daysAgo = (n: number) => new Date(Date.now() - n * 86400000).toISOString()
const HOME = 'C:\\Users\\you'

const AGENTS: Agent[] = [
  { id: 'opencode', name: 'OpenCode', kind: 'agent', skillsPath: `${HOME}\\.config\\opencode\\skills`, configPath: `${HOME}\\.config\\opencode`, detected: true, enabled: true, installed: 0 },
  { id: 'claude-code', name: 'Claude Code', kind: 'agent', skillsPath: `${HOME}\\.claude\\skills`, configPath: `${HOME}\\.claude`, detected: true, enabled: true, installed: 0 },
  { id: 'agent-skills', name: 'Agent Skills（通用）', kind: 'agent', skillsPath: `${HOME}\\.agents\\skills`, configPath: `${HOME}\\.agents`, detected: true, enabled: true, installed: 0 },
  { id: 'codex', name: 'Codex', kind: 'agent', skillsPath: `${HOME}\\.codex\\skills`, configPath: `${HOME}\\.codex`, detected: true, enabled: true, installed: 0 },
  { id: 'deepseek-harness', name: 'DeepSeek Harness', kind: 'agent', skillsPath: `${HOME}\\.dsh\\skills`, configPath: `${HOME}\\.dsh`, detected: true, enabled: true, installed: 0 },
  { id: 'pi', name: 'Pi', kind: 'agent', skillsPath: `${HOME}\\.pi\\agent\\skills`, configPath: `${HOME}\\.pi\\agent`, detected: true, enabled: true, installed: 0 },
  { id: 'cursor', name: 'Cursor', kind: 'agent', skillsPath: `${HOME}\\.cursor\\skills`, configPath: `${HOME}\\.cursor`, detected: false, enabled: false, installed: 0 },
  { id: 'gemini', name: 'Gemini CLI', kind: 'agent', skillsPath: `${HOME}\\.gemini\\skills`, configPath: `${HOME}\\.gemini`, detected: false, enabled: false, installed: 0 },
  { id: 'project-atlas', name: 'Atlas 项目', kind: 'project', skillsPath: 'D:\\Work\\atlas\\.agents\\skills', configPath: 'D:\\Work\\atlas', detected: true, enabled: true, installed: 0 },
]

const GROUPS: SkillGroup[] = [
  { id: 'design', name: '设计系统', skillCount: 0, sortOrder: 1 },
  { id: 'docs', name: '文档处理', skillCount: 0, sortOrder: 2 },
  { id: 'engineering', name: '工程效率', skillCount: 0, sortOrder: 3 },
  { id: 'reading', name: '阅读与内容', skillCount: 0, sortOrder: 4 },
]

interface SeedSkill {
  id: string
  name: string
  desc: string
  notes?: string
  tags: string[]
  groupId: string
  content: string
  contentFiles?: string[]
  size: number
  files: number
  version: string
  agents: string[]
  copies?: string[]
}

const SEED_SKILLS: SeedSkill[] = [
  { id: 'anti-ui-slop', name: 'anti-ui-slop', desc: 'Stop coding agents from shipping generic UI. Ground interfaces in real screens and run the finish gate.', notes: '设计评审前必须跑一次 finish gate。', tags: ['ui', 'review', 'design'], groupId: 'design', content: 'none', size: 192 * 1024, files: 37, version: '2.3.0', agents: ['opencode', 'claude-code', 'agent-skills', 'codex'], copies: ['agent-skills'] },
  { id: 'archify', name: 'archify', desc: 'Create polished, validated architecture, workflow, sequence and data-flow diagrams as standalone HTML.', tags: ['diagram', 'architecture'], groupId: 'design', content: 'script', size: 3.1 * 1024 * 1024, files: 84, version: '1.8.2', agents: ['opencode', 'claude-code', 'deepseek-harness'] },
  { id: 'codebase-to-course', name: 'codebase-to-course', desc: 'Turn any codebase into an interactive single-page HTML course that teaches how the code works.', tags: ['course', 'teaching'], groupId: 'engineering', content: 'script', size: 486 * 1024, files: 22, version: '1.2.0', agents: ['opencode', 'agent-skills', 'pi'] },
  { id: 'diagram-design', name: 'diagram-design', desc: 'Branded architecture, flowchart, sequence and data-model diagrams as standalone HTML/SVG.', tags: ['diagram', 'svg'], groupId: 'design', content: 'none', size: 1.4 * 1024 * 1024, files: 61, version: '3.0.1', agents: ['claude-code', 'codex'] },
  { id: 'docx', name: 'docx', desc: 'Create, read, edit and manipulate Word documents with professional formatting.', tags: ['office', 'word'], groupId: 'docs', content: 'script', size: 628 * 1024, files: 41, version: '1.4.3', agents: ['opencode', 'claude-code', 'agent-skills'] },
  { id: 'pdf', name: 'pdf', desc: 'Read, extract, merge, split, watermark, fill and OCR PDF files.', tags: ['office', 'pdf'], groupId: 'docs', content: 'script', size: 912 * 1024, files: 33, version: '2.1.0', agents: ['opencode', 'claude-code', 'codex', 'deepseek-harness'], copies: ['codex'] },
  { id: 'ui-ux-pro-max', name: 'ui-ux-pro-max', desc: 'UI/UX design intelligence with searchable styles, palettes, font pairings and stack guidance.', tags: ['ui', 'design-system'], groupId: 'design', content: 'none', size: 2.7 * 1024 * 1024, files: 118, version: '4.6.0', agents: ['opencode', 'claude-code', 'agent-skills', 'pi'] },
  { id: 'weread-skills', name: 'weread-skills', desc: '微信读书助手：搜索书籍、管理书架、查看笔记划线、浏览书评与阅读统计。', tags: ['reading', 'weread'], groupId: 'reading', content: 'script', size: 218 * 1024, files: 19, version: '0.9.4', agents: ['opencode', 'deepseek-harness'] },
  { id: 'xyq-skill', name: 'xyq-skill', desc: '小云雀媒体生成：图片、视频、超分、Canvas 画布与多轨草稿。', tags: ['media', 'video'], groupId: '', content: 'executable', contentFiles: ['scripts/upload.ps1（脚本）', 'bin/ffmpeg-helper.exe（可执行文件）'], size: 864 * 1024, files: 47, version: '1.7.1', agents: ['claude-code', 'project-atlas'] },
  { id: 'skill-creator', name: 'skill-creator', desc: 'Guide for creating effective skills that extend an agent with specialized knowledge and workflows.', tags: ['meta', 'authoring'], groupId: 'engineering', content: 'none', size: 156 * 1024, files: 12, version: '1.0.6', agents: ['opencode', 'claude-code', 'agent-skills', 'codex', 'pi'] },
  { id: 'find-skills', name: 'find-skills', desc: 'Help users discover and install agent skills when they ask how to extend capabilities.', tags: ['discovery'], groupId: 'engineering', content: 'none', size: 88 * 1024, files: 8, version: '1.1.2', agents: ['opencode', 'agent-skills', 'project-atlas'] },
  { id: 'web-design-guidelines', name: 'web-design-guidelines', desc: 'Review UI code against Web Interface Guidelines for accessibility and UX best practices.', tags: ['ui', 'accessibility'], groupId: 'design', content: 'none', size: 132 * 1024, files: 9, version: '2.0.0', agents: ['opencode', 'claude-code'] },
]

const SOURCES: Source[] = [
  { id: 'local', name: '本地仓库', kind: 'local', url: '', builtin: true, createdAt: daysAgo(30), syncedAt: now(), skillCount: 0 },
  { id: 'anthropics', name: 'Anthropic Skills', kind: 'git', url: 'github.com/anthropics/skills', builtin: true, createdAt: daysAgo(30), syncedAt: daysAgo(1), skillCount: 6 },
  { id: 'clawhub', name: 'ClawHub', kind: 'registry', url: 'clawhub.ai', builtin: true, createdAt: daysAgo(30), syncedAt: daysAgo(2), skillCount: 6 },
  { id: 'network', name: '网络导入', kind: 'url', url: '', builtin: true, createdAt: daysAgo(30), syncedAt: '', skillCount: 0 },
]

const IMPORTABLES: Importable[] = [
  { path: `${HOME}\\.xin\\skills\\anti-ui-slop`, name: 'anti-ui-slop', description: 'Ground interfaces in real product screens and run the finish gate.', slug: 'anti-ui-slop', content: 'none', contentFiles: [], sizeBytes: 192 * 1024, fileCount: 37, agents: ['Agent Skills（通用）'], alreadyInstalled: true, isLink: true, sourceId: 'local', sourceRef: `${HOME}\\.xin\\skills\\anti-ui-slop` },
  { path: `${HOME}\\.agents\\skills\\data-viz-kit`, name: 'data-viz-kit', description: 'Charts and dashboards with a strict visual grammar for dense product data.', slug: 'data-viz-kit', content: 'script', contentFiles: [], sizeBytes: 486 * 1024, fileCount: 28, agents: ['Agent Skills（通用）'], alreadyInstalled: false, isLink: true, sourceId: 'local', sourceRef: `${HOME}\\.agents\\skills\\data-viz-kit` },
  { path: `${HOME}\\.claude\\skills\\release-notes`, name: 'release-notes', description: 'Draft release notes from commits and PR titles, grouped by impact.', slug: 'release-notes', content: 'none', contentFiles: [], sizeBytes: 74 * 1024, fileCount: 6, agents: ['Claude Code'], alreadyInstalled: false, isLink: false, sourceId: 'local', sourceRef: `${HOME}\\.claude\\skills\\release-notes` },
  { path: `${HOME}\\.codex\\skills\\sql-review`, name: 'sql-review', description: 'Review SQL migrations for locking, indexes and backward compatibility.', slug: 'sql-review', content: 'script', contentFiles: [], sizeBytes: 128 * 1024, fileCount: 11, agents: ['Codex'], alreadyInstalled: false, isLink: false, sourceId: 'local', sourceRef: `${HOME}\\.codex\\skills\\sql-review` },
]

const FOLDER_ITEMS: Importable[] = [
  { path: `${HOME}\\Downloads\\skill-pack\\data-viz-kit`, name: 'data-viz-kit', description: 'Charts and dashboards with a strict visual grammar for dense product data.', slug: 'data-viz-kit', content: 'script', contentFiles: [], sizeBytes: 486 * 1024, fileCount: 28, agents: [], alreadyInstalled: false, isLink: false, sourceId: 'local', sourceRef: `${HOME}\\Downloads\\skill-pack\\data-viz-kit` },
  { path: `${HOME}\\Downloads\\skill-pack\\release-notes`, name: 'release-notes', description: 'Draft release notes from commits and PR titles, grouped by impact.', slug: 'release-notes', content: 'none', contentFiles: [], sizeBytes: 74 * 1024, fileCount: 6, agents: [], alreadyInstalled: false, isLink: false, sourceId: 'local', sourceRef: `${HOME}\\Downloads\\skill-pack\\release-notes` },
  { path: `${HOME}\\Downloads\\skill-pack\\scripts\\sql-review`, name: 'sql-review', description: 'Review SQL migrations for locking, indexes and backward compatibility.', slug: 'sql-review', content: 'script', contentFiles: [], sizeBytes: 128 * 1024, fileCount: 11, agents: [], alreadyInstalled: false, isLink: false, sourceId: 'local', sourceRef: `${HOME}\\Downloads\\skill-pack\\scripts\\sql-review` },
]

const EXTERNAL: Record<string, DirectoryEntry[]> = {
  'agent-skills': [
    {
      name: 'release-notes', displayName: 'release-notes', description: 'External real directory in the Agent skills folder.',
      path: `${HOME}\\.agents\\skills\\release-notes`, sourcePath: `${HOME}\\.agents\\skills\\release-notes`,
      linkTarget: '', kind: 'external-dir', status: 'external', mode: '', content: 'none', contentFiles: [], sizeBytes: 74 * 1024,
      fileCount: 6, isLink: false, managed: false, vaultSkillId: '', alreadyInVault: false, distributionPath: '',
    },
  ],
  'project-atlas': [
    {
      name: 'release-notes', displayName: 'release-notes', description: 'External link in the project skill directory.',
      path: 'D:\\Work\\atlas\\.agents\\skills\\release-notes', sourcePath: `${HOME}\\.claude\\skills\\release-notes`,
      linkTarget: `${HOME}\\.claude\\skills\\release-notes`, kind: 'external-link', status: 'external', mode: '', content: 'none',
      contentFiles: [],
      sizeBytes: 74 * 1024, fileCount: 6, isLink: true, managed: false, vaultSkillId: '',
      alreadyInVault: false, distributionPath: '',
    },
  ],
}

function build(): Snapshot {
  const distributions: Distribution[] = []
  const skills: Skill[] = SEED_SKILLS.map((seed, i) => {
    const agentIds: string[] = []
    for (const agentId of seed.agents) {
      const isCopy = seed.copies?.includes(agentId)
      const agent = AGENTS.find((a) => a.id === agentId)
      distributions.push({
        skillId: seed.id, agentId, mode: isCopy ? 'copy' : 'junction',
        linkPath: `${agent?.skillsPath}\\${seed.id}`,
        status: i === 4 && agentId === 'agent-skills' ? 'broken' : 'ok',
        contentHash: '9f3c1a2d', updatedAt: daysAgo(i),
      })
      agentIds.push(agentId)
    }
    const group = GROUPS.find((g) => g.id === seed.groupId)
    return {
      id: seed.id, name: seed.name, description: seed.desc, notes: seed.notes || '',
      tags: seed.tags, groupId: seed.groupId, groupName: group?.name || '',
      sourceId: i > 5 ? 'clawhub' : 'anthropics',
      sourceName: i > 5 ? 'ClawHub' : 'Anthropic Skills',
      vaultPath: `${HOME}\\.skill-vault\\skills\\${seed.id}`,
      contentHash: `9f3c1a2d${i}a7b4c81e`, content: seed.content, contentFiles: seed.contentFiles || [], version: seed.version,
      sizeBytes: seed.size, fileCount: seed.files, installedAt: daysAgo(12 - i),
      updatedAt: daysAgo(i % 4), updateAvailable: i === 2, agents: agentIds,
      linkCount: agentIds.length - (seed.copies?.length || 0),
      copyCount: seed.copies?.length || 0, brokenCount: i === 4 ? 1 : 0,
    }
  })
  for (const agent of AGENTS) {
    agent.installed = distributions.filter((d) => d.agentId === agent.id).length
  }
  const stats: Stats = {
    skills: skills.length,
    links: distributions.filter((d) => d.mode !== 'copy').length,
    copies: distributions.filter((d) => d.mode === 'copy').length,
    enabledAgents: AGENTS.filter((a) => a.enabled).length,
    detectedAgents: AGENTS.filter((a) => a.detected).length,
    broken: distributions.filter((d) => d.status !== 'ok').length,
    updates: skills.filter((s) => s.updateAvailable).length,
    vaultSizeBytes: skills.reduce((n, s) => n + s.sizeBytes, 0),
  }
  for (const group of GROUPS) group.skillCount = skills.filter((s) => s.groupId === group.id).length
  const activity: Activity[] = [
    { id: 5, action: 'group', subject: '设计系统', detail: '创建分组', createdAt: daysAgo(0) },
    { id: 4, action: 'distribute', subject: 'pdf', detail: '分发到 Codex · Junction', createdAt: daysAgo(0) },
    { id: 3, action: 'import', subject: 'ui-ux-pro-max', detail: '从 Agent Skills 导入', createdAt: daysAgo(1) },
    { id: 2, action: 'repair', subject: '健康检查', detail: '1 条断链已标记', createdAt: daysAgo(2) },
    { id: 1, action: 'skin', subject: '冷白机甲', detail: '默认皮肤', createdAt: daysAgo(9) },
  ]
  return {
    skills, agents: AGENTS,
    sources: SOURCES.map((s) => ({ ...s, skillCount: skills.filter((k) => k.sourceId === s.id).length })),
    groups: GROUPS.map((g) => ({ ...g })),
    distributions, activity,
    settings: {
      'ui.skin': localStorage.getItem('xsv-skin') || 'cold-white',
      'ui.layout': 'table', 'ui.language': 'zh-CN', 'install.defaultMode': 'junction',
      'ui.fontFamily': 'system', 'ui.fontScale': '1.12',
      'updates.checkOnStart': 'true', 'updates.autoUpdateLinks': 'false',
      'window.closeToTray': 'true', 'window.closeAction': 'ask', 'updates.lastCheckAt': daysAgo(0),
      'vault.path': `${HOME}\\.skill-vault`,
    },
    stats,
    vault: {
      path: `${HOME}\\.skill-vault`, appVersion: '0.1.0', platform: 'windows/amd64',
      lastScanAt: now(),
    },
    generatedAt: now(),
  }
}

let STATE: Snapshot = build()

function clone<T>(v: T): T { return JSON.parse(JSON.stringify(v)) as T }

function recalc() {
  for (const agent of STATE.agents) {
    agent.installed = STATE.distributions.filter((d) => d.agentId === agent.id).length
  }
  for (const skill of STATE.skills) {
    const group = STATE.groups.find((g) => g.id === skill.groupId)
    skill.groupName = group?.name || ''
    const ds = STATE.distributions.filter((d) => d.skillId === skill.id)
    skill.agents = [...new Set(ds.map((d) => d.agentId))]
    skill.linkCount = ds.filter((d) => d.mode !== 'copy').length
    skill.copyCount = ds.filter((d) => d.mode === 'copy').length
    skill.brokenCount = ds.filter((d) => d.status !== 'ok').length
  }
  for (const group of STATE.groups) group.skillCount = STATE.skills.filter((s) => s.groupId === group.id).length
  STATE.stats = {
    skills: STATE.skills.length,
    links: STATE.distributions.filter((d) => d.mode !== 'copy').length,
    copies: STATE.distributions.filter((d) => d.mode === 'copy').length,
    enabledAgents: STATE.agents.filter((a) => a.enabled).length,
    detectedAgents: STATE.agents.filter((a) => a.detected).length,
    broken: STATE.distributions.filter((d) => d.status !== 'ok').length,
    updates: STATE.skills.filter((s) => s.updateAvailable).length,
    vaultSizeBytes: STATE.skills.reduce((n, s) => n + s.sizeBytes, 0),
  }
  for (const src of STATE.sources) src.skillCount = STATE.skills.filter((k) => k.sourceId === src.id).length
  STATE.generatedAt = now()
  STATE.vault.lastScanAt = now()
}

function slugify(name: string): string {
  return name.toLowerCase().replace(/[^a-z0-9\u4e00-\u9fff-]+/g, '-').replace(/^-+|-+$/g, '') || 'skill'
}

function addDist(skillId: string, agentId: string, mode: string) {
  if (STATE.distributions.some((d) => d.skillId === skillId && d.agentId === agentId)) return
  const agent = STATE.agents.find((a) => a.id === agentId)
  const skill = STATE.skills.find((s) => s.id === skillId)
  if (!agent || !skill) return
  STATE.distributions.push({
    skillId, agentId, mode: mode === 'copy' ? 'copy' : 'junction',
    linkPath: `${agent.skillsPath}\\${skillId}`, status: 'ok', contentHash: skill.contentHash, updatedAt: now(),
  })
}

function importOne(path: string, sourceId = 'local', sourceRef = path): Skill {
  const parsed = path.split(/[\\/]/).filter(Boolean)
  const name = parsed[parsed.length - 1] || 'new-skill'
  const id = slugify(name)
  const existing = STATE.skills.find((s) => s.id === id)
  if (existing) throw new Error(`仓库中已存在同名技能「${id}」`)
  const source = STATE.sources.find((s) => s.id === sourceId)
  const skill: Skill = {
    id, name, description: 'A newly imported skill with a SKILL.md manifest and reference files.',
    notes: '', tags: [], groupId: '', groupName: '',
    sourceId, sourceName: source?.name || '本地导入', vaultPath: `${HOME}\\.skill-vault\\skills\\${id}`,
    contentHash: 'c81d4e0a91b2', content: 'none', contentFiles: [], version: '0.1.0', sizeBytes: 64 * 1024, fileCount: 5,
    installedAt: now(), updatedAt: now(), updateAvailable: false, agents: [], linkCount: 0, copyCount: 0, brokenCount: 0,
  }
  STATE.skills.push(skill)
  STATE.activity.unshift({ id: Date.now(), action: 'import', subject: name, detail: sourceRef, createdAt: now() })
  return skill
}

function directoryView(targetId: string): DirectoryView {
  const target = STATE.agents.find((a) => a.id === targetId)
  if (!target) throw new Error('目标不存在')
  const entries: DirectoryEntry[] = []
  for (const dist of STATE.distributions.filter((d) => d.agentId === targetId)) {
    const skill = STATE.skills.find((s) => s.id === dist.skillId)
    if (!skill) continue
    entries.push({
      name: skill.id, displayName: skill.name, description: skill.description,
      path: `${target.skillsPath}\\${skill.id}`, sourcePath: skill.vaultPath, linkTarget: skill.vaultPath,
      kind: dist.mode, status: dist.status === 'ok' ? 'managed' : dist.status, mode: dist.mode,
      content: skill.content, contentFiles: skill.contentFiles, sizeBytes: skill.sizeBytes, fileCount: skill.fileCount,
      isLink: dist.mode !== 'copy', managed: true, vaultSkillId: skill.id,
      alreadyInVault: true, distributionPath: dist.linkPath,
    })
  }
  entries.push(...(EXTERNAL[targetId] || []).map((e) => ({ ...e })))
  entries.sort((a, b) => Number(b.managed) - Number(a.managed) || a.displayName.localeCompare(b.displayName))
  return {
    target, path: target.skillsPath, entries,
    managed: entries.filter((e) => e.status === 'managed').length,
    external: entries.filter((e) => e.status === 'external').length,
    broken: entries.filter((e) => e.status === 'broken').length,
  }
}

function networkItems(url: string): Importable[] {
  const base = `${HOME}\\AppData\\Local\\Temp\\xsv-net\\session`
  return [
    { path: `${base}\\skills\\pdf`, name: 'pdf', description: 'Read, extract and edit PDF files from a GitHub repository.', slug: 'pdf', content: 'script', contentFiles: [], sizeBytes: 912 * 1024, fileCount: 33, agents: [], alreadyInstalled: true, isLink: false, sourceId: 'network', sourceRef: url },
    { path: `${base}\\skills\\release-notes`, name: 'release-notes', description: 'Draft release notes from commits and PR titles.', slug: 'release-notes', content: 'none', contentFiles: [], sizeBytes: 74 * 1024, fileCount: 6, agents: [], alreadyInstalled: false, isLink: false, sourceId: 'network', sourceRef: url },
    { path: `${base}\\skills\\sql-review`, name: 'sql-review', description: 'Review SQL migrations for locking, indexes and compatibility.', slug: 'sql-review', content: 'script', contentFiles: [], sizeBytes: 128 * 1024, fileCount: 11, agents: [], alreadyInstalled: false, isLink: false, sourceId: 'network', sourceRef: url },
  ]
}

export async function mockCall(method: string, args: any[]): Promise<any> {
  await new Promise((r) => setTimeout(r, 90))
  switch (method) {
    case 'Snapshot': return clone(STATE)
    case 'Refresh':
    case 'Repair':
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: '索引已刷新' })
    case 'CheckUpdates':
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: '已检查全部技能，当前没有可用更新' })
    case 'PickFolder': return `${HOME}\\Downloads\\skill-pack`
    case 'PickVaultFolder': return `${HOME}\\SkillVault-New`
    case 'ChangeVaultPath': {
      const path = String(args[0] || `${HOME}\\SkillVault-New`)
      const mode = String(args[1] || 'move')
      const oldPath = STATE.vault.path
      STATE.vault.path = path
      STATE.settings['vault.path'] = path
      for (const skill of STATE.skills) {
        if (skill.vaultPath.startsWith(oldPath)) {
          skill.vaultPath = path + skill.vaultPath.slice(oldPath.length)
        }
      }
      return clone({ snapshot: STATE, warnings: [], message: mode === 'move' ? '仓库已移动' : '仓库已复制' })
    }
    case 'InspectPath': {
      const p = String(args[0] || 'new-skill')
      const name = p.split(/[\\/]/).filter(Boolean).pop() || 'new-skill'
      return clone<Inspection>({
        name, description: 'A newly imported skill with a SKILL.md manifest and reference files.',
        slug: slugify(name), version: '0.1.0', license: 'MIT', path: p, content: 'none',
        contentFiles: [],
        sizeBytes: 64 * 1024, fileCount: 5, hasSkillMd: true,
      })
    }
    case 'Importables': return clone(IMPORTABLES)
    case 'DiscoverFolder': return clone(FOLDER_ITEMS)
    case 'NetworkImport': return clone(networkItems(String(args[0] || 'https://github.com/example/skills')))
    case 'Import': {
      const req = args[0] || {}
      const skill = importOne(String(req.path || 'new-skill'))
      for (const agentId of req.agentIds || []) addDist(skill.id, agentId, req.mode)
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: `已导入「${skill.name}」` })
    }
    case 'ImportMany': {
      const req = args[0] || {}
      let success = 0
      const warnings: string[] = []
      for (const item of req.items || []) {
        try {
          const skill = importOne(item.path, item.sourceId, item.sourceRef)
          for (const agentId of req.agentIds || []) addDist(skill.id, agentId, req.mode)
          success++
        } catch (err) {
          warnings.push(err instanceof Error ? err.message : String(err))
        }
      }
      if (!success && warnings.length) throw new Error(`没有技能被导入：${warnings[0]}`)
      recalc()
      return clone({ snapshot: STATE, warnings, message: `已导入 ${success} 个技能` })
    }
    case 'Distribute': {
      const req = args[0] || {}
      for (const agentId of req.agentIds || []) addDist(req.skillId, agentId, req.mode)
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: '分发已建立' })
    }
    case 'DistributeMany': {
      const req = args[0] || {}
      for (const skillId of req.skillIds || []) {
        for (const agentId of req.agentIds || []) addDist(skillId, agentId, req.mode)
      }
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: `已处理 ${(req.skillIds || []).length} 个技能` })
    }
    case 'SetDistribution': {
      const [skillId, agentId, enabled] = args
      if (enabled) addDist(skillId, agentId, STATE.settings['install.defaultMode'] || 'junction')
      else STATE.distributions = STATE.distributions.filter((d) => !(d.skillId === skillId && d.agentId === agentId))
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: enabled ? '分发已建立' : '分发已移除' })
    }
    case 'RemoveSkill': {
      const id = String(args[0])
      STATE.skills = STATE.skills.filter((s) => s.id !== id)
      STATE.distributions = STATE.distributions.filter((d) => d.skillId !== id)
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: '技能已移除' })
    }
    case 'SetAgentEnabled': {
      const [id, enabled] = args
      const agent = STATE.agents.find((a) => a.id === id)
      if (agent) { agent.enabled = !!enabled; if (enabled) agent.detected = true }
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: enabled ? '已启用目标' : '已停用目标' })
    }
    case 'SaveSetting': {
      const [key, value] = args
      STATE.settings[key] = String(value)
      if (key === 'ui.skin') localStorage.setItem('xsv-skin', String(value))
      return clone(STATE)
    }
    case 'SetSkin':
      STATE.settings['ui.skin'] = String(args[0])
      localStorage.setItem('xsv-skin', String(args[0]))
      return clone(STATE)
    case 'SkillFiles': {
      const id = String(args[0])
      return clone<FileEntry[]>([
        { path: 'SKILL.md', size: 4210, modTime: '2026-11-14 18:20' },
        { path: `references/${id}-guide.md`, size: 18244, modTime: '2026-11-12 09:41' },
        { path: 'scripts/validate.js', size: 9104, modTime: '2026-11-10 14:02' },
      ])
    }
    case 'UpdateSkillMeta': {
      const req = args[0] as SkillMetaUpdate
      const skill = STATE.skills.find((s) => s.id === req.skillId)
      if (!skill) throw new Error('技能不存在')
      skill.notes = req.notes
      skill.tags = req.tags
      skill.groupId = req.groupId
      recalc()
      return clone(STATE)
    }
    case 'CreateGroup': {
      const name = String(args[0])
      STATE.groups.push({ id: slugify(name), name, skillCount: 0, sortOrder: STATE.groups.length + 1 })
      recalc()
      return clone(STATE)
    }
    case 'RenameGroup': {
      const [id, name] = args
      const group = STATE.groups.find((g) => g.id === id)
      if (group) group.name = String(name)
      recalc()
      return clone(STATE)
    }
    case 'DeleteGroup': {
      const id = String(args[0])
      STATE.groups = STATE.groups.filter((g) => g.id !== id)
      for (const skill of STATE.skills) if (skill.groupId === id) skill.groupId = ''
      recalc()
      return clone(STATE)
    }
    case 'AddSource': {
      const req = args[0] as SourceInput
      STATE.sources.push({ id: slugify(req.name), name: req.name, kind: req.kind, url: req.url, builtin: false, createdAt: now(), syncedAt: '', skillCount: 0 })
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: `已添加来源「${req.name}」` })
    }
    case 'RemoveSource': {
      const id = String(args[0])
      const source = STATE.sources.find((s) => s.id === id)
      if (!source || source.builtin) throw new Error('内置来源不能移除')
      if (STATE.skills.some((s) => s.sourceId === id)) throw new Error('该来源下还有技能')
      STATE.sources = STATE.sources.filter((s) => s.id !== id)
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: '来源已移除' })
    }
    case 'AddProject': {
      const req = args[0] as ProjectInput
      STATE.agents.push({ id: `project-${slugify(req.name)}`, name: req.name, kind: 'project', skillsPath: req.skillsPath, configPath: '', detected: true, enabled: true, installed: 0 })
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: `已添加项目目录「${req.name}」` })
    }
    case 'RemoveProject': {
      const id = String(args[0])
      const target = STATE.agents.find((a) => a.id === id)
      if (!target || target.kind !== 'project') throw new Error('只能移除项目目录')
      if (STATE.distributions.some((d) => d.agentId === id)) throw new Error('项目目录下还有分发')
      STATE.agents = STATE.agents.filter((a) => a.id !== id)
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: '项目目录已移除' })
    }
    case 'DirectorySkills': return clone(directoryView(String(args[0])))
    case 'RemoveDirectoryEntry': {
      const [targetId, name] = args
      STATE.distributions = STATE.distributions.filter((d) => !(d.agentId === targetId && d.skillId === name))
      if (EXTERNAL[targetId]) EXTERNAL[targetId] = EXTERNAL[targetId].filter((e) => e.name !== name)
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: '目录项已移除' })
    }
    case 'RebuildDirectoryDistribution': {
      const [targetId, skillId, mode] = args
      const dist = STATE.distributions.find((d) => d.agentId === targetId && d.skillId === skillId)
      if (!dist) throw new Error('分发不存在')
      dist.mode = mode
      dist.status = 'ok'
      recalc()
      return clone({ snapshot: STATE, warnings: [], message: `已切换为 ${mode}` })
    }
    case 'OpenDirectoryEntry':
    case 'OpenVault':
    case 'OpenSkillFolder':
    case 'OpenAgentFolder':
    case 'OpenDataFolder':
      return null
    case 'ResolveClose': {
      const [action, remember] = args
      if (remember) STATE.settings['window.closeAction'] = String(action)
      return null
    }
    case 'CancelClose':
      return null
    default:
      throw new Error(`Mock 未实现: ${method}`)
  }
}
