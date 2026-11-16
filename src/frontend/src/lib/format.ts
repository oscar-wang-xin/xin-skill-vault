export function formatBytes(bytes: number): string {
  if (!bytes || bytes < 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = bytes / 1024
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  return `${value >= 100 ? Math.round(value) : value.toFixed(1)} ${units[i]}`
}

export function shortPath(path: string, max = 48): string {
  if (!path) return ''
  let display = path
    .replace(/^[A-Za-z]:\\Users\\[^\\]+/i, '~')
    .replace(/^\/Users\/[^/]+/i, '~')
    .replace(/^\/home\/[^/]+/i, '~')
  if (display.length > max) display = '…' + display.slice(display.length - max + 1)
  return display
}

export function relativeTime(iso: string): string {
  if (!iso) return '—'
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return '—'
  const diff = Date.now() - t
  if (diff < 45_000) return '刚刚'
  const mins = Math.floor(diff / 60_000)
  if (mins < 60) return `${mins} 分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  const d = new Date(t)
  return `${d.getMonth() + 1} 月 ${d.getDate()} 日`
}

export function timeShort(iso: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function modeLabel(mode: string): string {
  if (mode === 'copy') return '复制'
  if (mode === 'symlink') return 'Symlink'
  if (mode === 'junction') return 'Junction'
  return mode || '未分发'
}

export function modeShort(mode: string): string {
  if (mode === 'copy') return 'COPY'
  if (mode === 'symlink') return 'SYMLINK'
  return 'JUNCTION'
}

export function statusLabel(status: string): string {
  switch (status) {
    case 'ok': return '正常'
    case 'broken': return '断链'
    case 'drifted': return '副本漂移'
    case 'missing': return '缺失'
    default: return status || '未知'
  }
}

export function contentLabel(kind: string): string {
  switch (kind) {
    case 'executable': return '有可执行内容'
    case 'script': return '含脚本'
    case 'none': return '无脚本'
    default: return '未识别'
  }
}

export function contentToneClass(kind: string): string {
  switch (kind) {
    case 'executable': return 'tone-slate'
    case 'script': return 'tone-data'
    default: return ''
  }
}

export function contentSummary(kind: string): string {
  switch (kind) {
    case 'executable':
      return '目录中存在可执行文件（如 .exe、.dll、.jar）。应用不会运行这些文件。'
    case 'script':
      return '目录中存在脚本文件（如 .ps1、.bat、.sh、.py、.js）。应用不会执行这些脚本。'
    case 'none':
      return '未发现脚本或可执行文件，内容以文档、配置和静态资源为主。'
    default:
      return '尚未识别文件构成；重新扫描仓库后可更新。'
  }
}

// Native title tooltips support newlines in Chromium/WebView2, so a single
// title attribute can list the files behind a content label.
export function contentTooltip(kind: string, files: string[] = []): string {
  const lines = [contentSummary(kind)]
  if (files.length > 0) {
    lines.push('', '相关文件：')
    for (const file of files) lines.push('• ' + file)
  }
  return lines.join('\n')
}

export function sourceKindLabel(kind: string): string {
  switch (kind) {
    case 'git': return 'Git 仓库'
    case 'registry': return '注册表'
    case 'url': return '网络地址'
    case 'local': return '本地目录'
    default: return kind
  }
}

export function actionLabel(action: string): string {
  switch (action) {
    case 'import': return '导入'
    case 'distribute': return '分发'
    case 'unlink': return '取消分发'
    case 'remove': return '移除'
    case 'repair': return '健康检查'
    case 'skin': return '换肤'
    case 'check': return '检查更新'
    default: return action
  }
}

const AGENT_MARKS: Record<string, string> = {
  opencode: 'OC',
  'claude-code': 'CC',
  codex: 'CX',
  'deepseek-harness': 'DS',
  'agent-skills': 'AS',
  cursor: 'CU',
  gemini: 'GM',
  pi: 'π',
  workbuddy: 'WB',
  codebuddy: 'CB',
  'qwen-code': 'QC',
  windsurf: 'WS',
  trae: 'TR',
  cline: 'CL',
  'roo-code': 'RC',
  openclaw: 'OL',
}

const AGENT_TONES: Record<string, string> = {
  opencode: 'moss',
  'claude-code': 'clay',
  codex: 'slate',
  'deepseek-harness': 'plum',
  'agent-skills': 'data',
  cursor: 'slate',
  gemini: 'ochre',
  pi: 'ochre',
  workbuddy: 'moss',
  codebuddy: 'clay',
  'qwen-code': 'plum',
  windsurf: 'data',
  trae: 'slate',
  cline: 'moss',
  'roo-code': 'ochre',
  openclaw: 'clay',
}

const TONES = ['moss', 'slate', 'plum', 'ochre', 'clay', 'data']

export function agentMark(id: string, name = ''): string {
  if (AGENT_MARKS[id]) return AGENT_MARKS[id]
  const ascii = name.replace(/[^A-Za-z0-9 ]/g, '').trim()
  if (!ascii) return name.slice(0, 2) || 'AG'
  const words = ascii.split(/\s+/)
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase()
  return (words[0][0] + words[1][0]).toUpperCase()
}

export function agentTone(id: string): string {
  if (AGENT_TONES[id]) return AGENT_TONES[id]
  let sum = 0
  for (let i = 0; i < id.length; i++) sum += id.charCodeAt(i)
  return TONES[sum % TONES.length]
}

export function skillMark(name: string): string {
  const words = name.replace(/[^A-Za-z0-9 -]/g, ' ').trim().split(/[\s-]+/).filter(Boolean)
  if (!words.length) return 'SK'
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase()
  return (words[0][0] + words[1][0]).toUpperCase()
}

export function skillTone(name: string): string {
  let sum = 0
  for (let i = 0; i < name.length; i++) sum += name.charCodeAt(i)
  return TONES[sum % TONES.length]
}

export function truncate(value: string, max: number): string {
  if (!value) return ''
  return value.length > max ? value.slice(0, max - 1) + '…' : value
}
