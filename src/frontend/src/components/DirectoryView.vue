<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../lib/api'
import {
  distributeMany,
  importSkills,
  openAgentFolder,
  openDirectoryEntry,
  pushToast,
  rebuildDistribution,
  removeDirectoryEntry,
  removeProject,
} from '../lib/appState'
import type { Agent, DirectoryEntry, DirectoryView as DirectoryViewData, ImportItem, Snapshot } from '../lib/types'
import { agentMark, agentTone, formatBytes, shortPath, truncate } from '../lib/format'
import Icon from './Icon.vue'
import ProjectDialog from './ProjectDialog.vue'

const props = defineProps<{
  snapshot: Snapshot
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'select-skill', id: string): void
}>()

const targets = computed(() =>
  props.snapshot.agents.filter((a) => a.kind === 'project' || a.enabled || a.detected),
)
const targetId = ref('')
const view = ref<DirectoryViewData | null>(null)
const loading = ref(false)
const error = ref('')
const addOpen = ref(false)
const projectOpen = ref(false)
const selectedAdd = reactive(new Set<string>())
const addMode = ref(props.snapshot.settings['install.defaultMode'] || 'junction')
const confirmState = reactive({
  open: false,
  title: '',
  message: '',
  impact: '',
  label: '确认',
  action: async () => {},
})

const target = computed(() => targets.value.find((a) => a.id === targetId.value) || null)
const isWindows = computed(() => (props.snapshot.vault.platform || 'windows').startsWith('windows'))
const linkLabel = computed(() => (isWindows.value ? 'Junction' : 'Symlink'))
const addCandidates = computed(() => {
  const managed = new Set((view.value?.entries || []).filter((e) => e.managed).map((e) => e.vaultSkillId))
  return props.snapshot.skills.filter((skill) => !managed.has(skill.id))
})
const canAdd = computed(() => selectedAdd.size > 0 && !props.busy)

onMounted(() => {
  if (targets.value.length) targetId.value = targets.value[0].id
  load()
})

watch(targets, (list) => {
  if (!list.some((a) => a.id === targetId.value)) {
    targetId.value = list[0]?.id || ''
  }
})

watch(targetId, () => load())
watch(() => props.snapshot.generatedAt, () => {
  if (targetId.value) load()
})

async function load() {
  if (!targetId.value) {
    view.value = null
    return
  }
  loading.value = true
  error.value = ''
  try {
    view.value = await api.directorySkills(targetId.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
    view.value = null
  } finally {
    loading.value = false
  }
}

function kindLabel(kind: string) {
  switch (kind) {
    case 'junction': return 'Junction'
    case 'symlink': return 'Symlink'
    case 'copy': return '复制'
    case 'external-link': return '外部链接'
    case 'external-dir': return '真实目录'
    case 'broken': return '断链'
    case 'unregistered-link': return '未登记链接'
    default: return kind || '未知'
  }
}

function statusLabel(status: string) {
  switch (status) {
    case 'managed': return '已托管'
    case 'external': return '外部'
    case 'broken': return '断链'
    case 'drifted': return '副本漂移'
    default: return status
  }
}

function kindClass(kind: string) {
  if (kind === 'external-link' || kind === 'external-dir' || kind === 'broken') return 'external'
  if (kind === 'copy') return 'copy'
  return 'link'
}

async function openEntry(entry: DirectoryEntry) {
  await openDirectoryEntry(targetId.value, entry.name)
}

async function importEntry(entry: DirectoryEntry) {
  const sourcePath = entry.sourcePath || entry.path
  const items: ImportItem[] = [{ path: sourcePath, sourceId: 'local', sourceRef: sourcePath }]
  const ok = await importSkills(items, [], addMode.value)
  if (ok) await load()
}

function askRemove(entry: DirectoryEntry) {
  confirmState.open = true
  confirmState.title = `从 ${target.value?.name || '目录'} 移除「${entry.displayName}」？`
  confirmState.message = entry.managed
    ? '会删除该目录中的链接/副本，并移除对应分发记录；仓库里的技能内容不受影响。'
    : '链接会直接删除；真实目录会重命名为 .removed-时间戳，不会静默删除。'
  confirmState.impact = entry.path
  confirmState.label = '移除'
  confirmState.action = async () => {
    confirmState.open = false
    const ok = await removeDirectoryEntry(targetId.value, entry.name)
    if (ok) await load()
  }
}

function askRemoveProject() {
  if (!target.value || target.value.kind !== 'project') return
  confirmState.open = true
  confirmState.title = `移除项目目录「${target.value.name}」？`
  confirmState.message = '只从列表中移除该目标；项目目录和其中未被托管的文件不会被删除。若还有分发，需要先移除。'
  confirmState.impact = target.value.skillsPath
  confirmState.label = '移除项目'
  confirmState.action = async () => {
    confirmState.open = false
    if (await removeProject(targetId.value)) await load()
  }
}

async function toggleMode(entry: DirectoryEntry) {
  const next = entry.mode === 'copy' ? (isWindows.value ? 'junction' : 'symlink') : 'copy'
  const ok = await rebuildDistribution(targetId.value, entry.vaultSkillId, next)
  if (ok) await load()
}

function startAdd() {
  selectedAdd.clear()
  addMode.value = props.snapshot.settings['install.defaultMode'] || 'junction'
  addOpen.value = true
}

function toggleAdd(id: string) {
  if (selectedAdd.has(id)) selectedAdd.delete(id)
  else selectedAdd.add(id)
}

function selectAllAdd() {
  if (selectedAdd.size === addCandidates.value.length) selectedAdd.clear()
  else addCandidates.value.forEach((s) => selectedAdd.add(s.id))
}

async function submitAdd() {
  if (!canAdd.value) return
  const ok = await distributeMany([...selectedAdd], [targetId.value], addMode.value)
  if (ok) {
    addOpen.value = false
    await load()
  }
}

function editEntry(entry: DirectoryEntry) {
  emit('select-skill', entry.vaultSkillId)
}

function openTarget() {
  openAgentFolder(targetId.value)
}
</script>

<template>
  <div class="page directory-page">
    <header class="page-head">
      <div>
        <div class="eyebrow">DIRECTORIES · 目录管理</div>
        <h1>目录</h1>
        <p class="page-sub">查看每个 Agent / 项目目录里实际存在的 Skill，追溯链接源头，并执行导入、移除或切换链接方式。</p>
      </div>
      <div class="page-actions">
        <button class="btn" @click="projectOpen = true"><Icon name="plus" />项目目录</button>
        <button class="btn ghost" :disabled="loading" @click="load"><Icon name="refresh" :class="{ spin: loading }" />刷新</button>
      </div>
    </header>

    <div class="directory-layout">
      <aside class="directory-targets">
        <div class="section-head" style="margin-top: 0">
          <span class="section-title">目标目录</span>
          <span class="section-count">{{ targets.length }}</span>
        </div>
        <button
          v-for="item in targets"
          :key="item.id"
          class="directory-target"
          :class="{ active: item.id === targetId }"
          @click="targetId = item.id"
        >
          <span class="agent-mark lg" :class="`tone-${agentTone(item.id)}`">{{ agentMark(item.id, item.name) }}</span>
          <span class="directory-target-copy">
            <span class="directory-target-name">{{ item.name }} <span v-if="item.kind === 'project'" class="tag">项目</span></span>
            <span class="directory-target-path">{{ shortPath(item.skillsPath, 38) }}</span>
          </span>
          <span class="directory-target-count">{{ item.installed }}</span>
        </button>
        <div v-if="targets.length === 0" class="inspect-placeholder small">
          <Icon name="folder" size="lg" />
          <h3>没有目标目录</h3>
          <p>先启用 Agent 或添加项目目录。</p>
        </div>
      </aside>

      <section class="directory-main">
        <div v-if="error" class="inline-error"><Icon name="alert" size="sm" />{{ error }}</div>
        <div v-else-if="!target" class="inspect-placeholder">
          <Icon name="folder" size="lg" />
          <h3>选择一个目录</h3>
          <p>左侧选择 Agent 或项目目录，查看其中的技能。</p>
        </div>
        <template v-else>
          <div class="directory-head">
            <div class="directory-title">
              <span class="agent-mark lg" :class="`tone-${agentTone(target.id)}`">{{ agentMark(target.id, target.name) }}</span>
              <div>
                <h2>{{ target.name }}</h2>
                <div class="directory-path" :title="target.skillsPath">{{ target.skillsPath }}</div>
              </div>
            </div>
            <div class="directory-actions">
              <button class="btn sm" :disabled="loading" @click="startAdd">
                <Icon name="plus" size="sm" />从仓库添加
              </button>
              <button class="btn ghost sm" @click="openTarget"><Icon name="folder-open" size="sm" />打开目录</button>
              <button
                v-if="target.kind === 'project'"
                class="btn ghost sm danger"
                @click="askRemoveProject"
              >移除项目</button>
            </div>
          </div>

          <div class="directory-stats">
            <span><strong>{{ view?.entries?.length || 0 }}</strong> 个目录项</span>
            <span><span class="dot moss" /><strong>{{ view?.managed || 0 }}</strong> 已托管</span>
            <span><span class="dot" /><strong>{{ view?.external || 0 }}</strong> 外部</span>
            <span :class="{ 'stat-bad': (view?.broken || 0) > 0 }"><span class="dot clay" /><strong>{{ view?.broken || 0 }}</strong> 断链</span>
            <span class="directory-hint">真实目录移除时会重命名保留，不会静默删除</span>
          </div>

          <div v-if="loading" class="inspect-loading"><Icon name="refresh" class="spin" size="sm" />正在读取目录…</div>
          <div v-else-if="!view || !view.entries || view.entries.length === 0" class="inspect-placeholder">
            <Icon name="folder" size="lg" />
            <h3>目录是空的</h3>
            <p>可以从仓库添加技能，或把外部技能导入仓库后再分发。</p>
            <button class="btn primary sm" style="margin-top: 12px" @click="startAdd"><Icon name="plus" size="sm" />从仓库添加</button>
          </div>
          <div v-else class="dir-table">
            <div class="dir-head">
              <span>技能</span><span>类型</span><span>来源 / 链接目标</span><span>状态</span><span>操作</span>
            </div>
            <div v-for="entry in view.entries" :key="entry.name" class="dir-row">
              <div class="dir-skill">
                <span class="tile" :class="`tone-${agentTone(entry.name)}`">{{ entry.displayName.slice(0, 2).toUpperCase() }}</span>
                <span class="dir-skill-copy">
                  <span class="dir-skill-name">{{ entry.displayName }}</span>
                  <span class="dir-skill-desc">{{ entry.description || entry.name }}</span>
                </span>
              </div>
              <div>
                <span class="tag" :class="`kind-${kindClass(entry.kind)}`">{{ kindLabel(entry.kind) }}</span>
                <div class="dir-sub">{{ entry.managed && entry.mode ? '仓库内容' : (entry.isLink ? '链接' : '独立目录') }}</div>
              </div>
              <div class="dir-source" :title="entry.sourcePath">
                <span>{{ shortPath(entry.sourcePath, 54) }}</span>
                <span v-if="entry.alreadyInVault" class="dir-in-vault">已收录</span>
              </div>
              <div>
                <span class="status" :class="{ 'status-bad': entry.status === 'broken', 'status-warn': entry.status === 'drifted' }">
                  <span class="dot" :class="entry.status === 'broken' ? 'clay' : entry.status === 'managed' ? 'moss' : ''" />
                  {{ statusLabel(entry.status) }}
                </span>
                <div class="dir-sub">{{ entry.fileCount }} files · {{ formatBytes(entry.sizeBytes) }}</div>
              </div>
              <div class="dir-actions">
                <button class="row-icon" title="打开目录" @click="openEntry(entry)"><Icon name="folder-open" size="sm" /></button>
                <button
                  v-if="entry.managed && entry.vaultSkillId"
                  class="row-icon"
                  title="编辑仓库中的备注 / 标签 / 分组"
                  @click="editEntry(entry)"
                ><Icon name="edit" size="sm" /></button>
                <button
                  v-if="entry.managed && entry.vaultSkillId && entry.mode"
                  class="btn ghost sm"
                  :disabled="busy"
                  @click="toggleMode(entry)"
                >
                  <Icon :name="entry.mode === 'copy' ? 'link' : 'copy'" size="sm" />
                  {{ entry.mode === 'copy' ? linkLabel : '复制' }}
                </button>
                <button
                  v-if="!entry.managed && entry.status !== 'broken'"
                  class="btn ghost sm"
                  :disabled="busy || entry.alreadyInVault"
                  :title="entry.alreadyInVault ? '仓库中已有同名技能' : '复制源头内容进仓库'"
                  @click="importEntry(entry)"
                >
                  <Icon name="download" size="sm" />{{ entry.alreadyInVault ? '已收录' : '收进仓库' }}
                </button>
                <button class="row-icon danger" title="移除目录项" @click="askRemove(entry)">
                  <Icon name="trash" size="sm" />
                </button>
              </div>
            </div>
          </div>
        </template>
      </section>
    </div>

    <div v-if="addOpen" class="modal-overlay" @click.self="addOpen = false">
      <div class="modal distribute-modal" role="dialog" aria-modal="true" aria-label="从仓库添加技能">
        <header class="modal-head compact">
          <div>
            <div class="eyebrow">ADD TO TARGET · 添加</div>
            <h2>添加到「{{ target?.name }}」</h2>
            <p>把仓库中的技能分发到该项目/Agent 目录，可多选。</p>
          </div>
          <button class="modal-close" aria-label="关闭" @click="addOpen = false"><Icon name="x" /></button>
        </header>
        <div class="distribute-body">
          <div class="option-head">
            <span class="form-label">仓库技能</span>
            <button class="mini-link" @click="selectAllAdd">
              {{ selectedAdd.size === addCandidates.length ? '全不选' : '全选' }}
            </button>
          </div>
          <div class="agent-pick-list tall">
            <label v-for="skill in addCandidates" :key="skill.id" class="agent-pick" :class="{ checked: selectedAdd.has(skill.id) }">
              <input type="checkbox" :checked="selectedAdd.has(skill.id)" @change="toggleAdd(skill.id)" />
              <span class="tile" :class="`tone-${agentTone(skill.id)}`">{{ skill.name.slice(0, 2).toUpperCase() }}</span>
              <span class="agent-pick-copy">
                <span>{{ skill.name }}</span>
                <small>{{ skill.groupName || '未分组' }} · {{ formatBytes(skill.sizeBytes) }}</small>
              </span>
              <span class="agent-check"><Icon v-if="selectedAdd.has(skill.id)" name="check" size="sm" /></span>
            </label>
            <div v-if="addCandidates.length === 0" class="muted small">所有仓库技能都已在此目录中。</div>
          </div>
          <div class="mode-cards horizontal" style="margin-top: 14px">
            <button class="mode-card" :class="{ active: addMode !== 'copy' }" @click="addMode = isWindows ? 'junction' : 'symlink'">
              <Icon name="link" /><span><strong>{{ linkLabel }}</strong><small>仓库更新即生效</small></span><span class="mode-radio" />
            </button>
            <button class="mode-card" :class="{ active: addMode === 'copy' }" @click="addMode = 'copy'">
              <Icon name="copy" /><span><strong>复制</strong><small>独立副本</small></span><span class="mode-radio" />
            </button>
          </div>
        </div>
        <footer class="modal-foot">
          <div class="foot-summary"><span class="dot" :class="selectedAdd.size ? 'moss' : ''" />已选 {{ selectedAdd.size }} 个技能</div>
          <button class="btn ghost" @click="addOpen = false">取消</button>
          <button class="btn primary" :disabled="!canAdd" @click="submitAdd">
            <Icon name="share" size="sm" />{{ busy ? '添加中…' : '添加到目录' }}
          </button>
        </footer>
      </div>
    </div>

    <ProjectDialog v-if="projectOpen" :busy="busy" @close="projectOpen = false; load()" />

    <div v-if="confirmState.open" class="modal-overlay" @click.self="confirmState.open = false">
      <div class="modal confirm-modal" role="dialog" aria-modal="true" :aria-label="confirmState.title">
        <header class="modal-head compact">
          <div><div class="eyebrow">CONFIRM · 确认</div><h2>{{ confirmState.title }}</h2></div>
          <button class="modal-close" aria-label="关闭" @click="confirmState.open = false"><Icon name="x" /></button>
        </header>
        <div class="confirm-body">
          <p>{{ confirmState.message }}</p>
          <div v-if="confirmState.impact" class="confirm-impact">{{ confirmState.impact }}</div>
        </div>
        <footer class="modal-foot">
          <button class="btn ghost" @click="confirmState.open = false">取消</button>
          <button class="btn danger-solid" @click="confirmState.action()">{{ confirmState.label }}</button>
        </footer>
      </div>
    </div>
  </div>
</template>
