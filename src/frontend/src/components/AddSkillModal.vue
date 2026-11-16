<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../lib/api'
import { importSkills, type AddTab } from '../lib/appState'
import type { ImportItem, Importable, Snapshot } from '../lib/types'
import { agentMark, agentTone, contentLabel, contentToneClass, contentTooltip, formatBytes, shortPath, truncate } from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
  initialTab: AddTab
  busy: boolean
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const tab = ref<AddTab>(props.initialTab)
const folderPath = ref('')
const folderItems = ref<Importable[]>([])
const existingItems = ref<Importable[]>([])
const networkItems = ref<Importable[]>([])
const networkURL = ref('https://github.com/anthropics/skills')
const existingQuery = ref('')
const selected = reactive(new Set<string>())
const selectedAgents = reactive(new Set<string>())
const mode = ref(props.snapshot.settings['install.defaultMode'] || 'junction')
const folderScanning = ref(false)
const existingLoading = ref(false)
const networkLoading = ref(false)
const folderError = ref('')
const existingError = ref('')
const networkError = ref('')
const scannedFolder = ref(false)
const networkDone = ref(false)

const enabledTargets = computed(() => props.snapshot.agents.filter((a) => a.enabled))
const isWindows = computed(() => (props.snapshot.vault.platform || 'windows').startsWith('windows'))
const linkMode = computed(() => (isWindows.value ? 'junction' : 'symlink'))
const linkLabel = computed(() => (isWindows.value ? 'Junction' : 'Symlink'))

const tabItems = computed<Importable[]>(() => {
  if (tab.value === 'local') return folderItems.value
  if (tab.value === 'existing') return existingItems.value
  return networkItems.value
})

const activeItems = computed(() => {
  if (tab.value !== 'existing' || !existingQuery.value.trim()) return tabItems.value
  const q = existingQuery.value.trim().toLowerCase()
  return tabItems.value.filter((item) =>
    `${item.name} ${item.description} ${item.agents.join(' ')} ${item.path}`.toLowerCase().includes(q),
  )
})

const selectableItems = computed(() => tabItems.value.filter((item) => !item.alreadyInstalled))
const allSelected = computed(
  () => selectableItems.value.length > 0 && selectableItems.value.every((item) => selected.has(item.path)),
)
const selectedItems = computed(() => tabItems.value.filter((item) => selected.has(item.path)))
const canImport = computed(() => selectedItems.value.length > 0 && !props.busy)

const summary = computed(() => {
  const count = selectedItems.value.length
  if (count === 0) return '等待选择技能'
  return `${count} 个技能 · 分发到 ${selectedAgents.size} 个目标 · ${mode.value === 'copy' ? '复制' : linkLabel.value}`
})

onMounted(() => {
  const defaults = enabledTargets.value.slice(0, 3)
  if (enabledTargets.value.some((a) => a.id === 'opencode')) selectedAgents.add('opencode')
  if (enabledTargets.value.some((a) => a.id === 'claude-code')) selectedAgents.add('claude-code')
  if (selectedAgents.size === 0) defaults.forEach((a) => selectedAgents.add(a.id))
  if (tab.value === 'existing') loadExisting()
  document.addEventListener('keydown', onKey)
})

onBeforeUnmount(() => document.removeEventListener('keydown', onKey))

function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}

watch(tab, (value) => {
  selected.clear()
  folderError.value = ''
  existingError.value = ''
  networkError.value = ''
  if (value === 'existing' && existingItems.value.length === 0 && !existingLoading.value) loadExisting()
})

async function loadExisting() {
  existingLoading.value = true
  existingError.value = ''
  try {
    existingItems.value = await api.importables()
    selected.clear()
  } catch (err) {
    existingError.value = err instanceof Error ? err.message : String(err)
  } finally {
    existingLoading.value = false
  }
}

async function chooseFolder() {
  folderError.value = ''
  try {
    const picked = await api.pickFolder()
    if (!picked) return
    folderPath.value = picked
    await scanFolder()
  } catch (err) {
    folderError.value = err instanceof Error ? err.message : String(err)
  }
}

async function scanFolder() {
  if (!folderPath.value.trim()) return
  folderScanning.value = true
  folderError.value = ''
  scannedFolder.value = false
  selected.clear()
  try {
    const items = await api.discoverFolder(folderPath.value.trim())
    folderItems.value = items
    scannedFolder.value = true
    if (items.length === 1 && !items[0].alreadyInstalled) selected.add(items[0].path)
  } catch (err) {
    folderItems.value = []
    folderError.value = err instanceof Error ? err.message : String(err)
  } finally {
    folderScanning.value = false
  }
}

async function runNetwork() {
  if (!networkURL.value.trim()) return
  networkLoading.value = true
  networkError.value = ''
  networkDone.value = false
  selected.clear()
  try {
    const items = await api.networkImport(networkURL.value.trim())
    networkItems.value = items
    networkDone.value = true
    if (items.length === 1 && !items[0].alreadyInstalled) selected.add(items[0].path)
  } catch (err) {
    networkItems.value = []
    networkError.value = err instanceof Error ? err.message : String(err)
  } finally {
    networkLoading.value = false
  }
}

function toggle(item: Importable) {
  if (item.alreadyInstalled) return
  if (selected.has(item.path)) selected.delete(item.path)
  else selected.add(item.path)
}

function selectAll() {
  if (allSelected.value) {
    selected.clear()
    return
  }
  selectableItems.value.forEach((item) => selected.add(item.path))
}

function toggleAgent(id: string) {
  if (selectedAgents.has(id)) selectedAgents.delete(id)
  else selectedAgents.add(id)
}

function selectAllAgents() {
  if (selectedAgents.size === enabledTargets.value.length) selectedAgents.clear()
  else enabledTargets.value.forEach((a) => selectedAgents.add(a.id))
}

async function submit() {
  if (!canImport.value) return
  const items: ImportItem[] = selectedItems.value.map((item) => ({
    path: item.path,
    sourceId: item.sourceId || 'local',
    sourceRef: item.sourceRef || item.path,
  }))
  await importSkills(items, [...selectedAgents], mode.value)
}

function contentTagClass(kind: string) {
  return contentToneClass(kind)
}

function sourceLabel(item: Importable) {
  if (item.sourceId === 'network') return truncate(item.sourceRef || item.path, 88)
  return shortPath(item.sourceRef || item.path, 72)
}

function agentSubline(kind: string, installed: number) {
  return kind === 'project' ? '项目目录' : `${installed} 个技能`
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal add-modal" role="dialog" aria-modal="true" aria-label="添加技能">
      <header class="modal-head">
        <div>
          <div class="eyebrow">ACQUISITION · 收录</div>
          <h2>添加技能</h2>
          <p>从文件夹、已有安装或网络来源发现技能；支持多选，复制进仓库后再分发。</p>
        </div>
        <button class="modal-close" aria-label="关闭" @click="emit('close')"><Icon name="x" /></button>
      </header>

      <div class="add-tabs">
        <button class="add-tab" :class="{ active: tab === 'local' }" @click="tab = 'local'">
          <Icon name="folder" size="sm" />
          <span><strong>本地文件夹</strong><small>递归扫描包含 SKILL.md 的目录</small></span>
        </button>
        <button class="add-tab" :class="{ active: tab === 'existing' }" @click="tab = 'existing'">
          <Icon name="inbox" size="sm" />
          <span><strong>现有安装</strong><small>追溯 Junction / Symlink 的源头</small></span>
        </button>
        <button class="add-tab" :class="{ active: tab === 'network' }" @click="tab = 'network'">
          <Icon name="external" size="sm" />
          <span><strong>网络导入</strong><small>GitHub / zip / tar.gz</small></span>
        </button>
      </div>

      <div class="add-body">
        <section class="add-left">
          <template v-if="tab === 'local'">
            <div class="form-block">
              <label class="form-label" for="skill-folder">技能根目录</label>
              <div class="path-picker">
                <input
                  id="skill-folder"
                  v-model="folderPath"
                  class="input"
                  placeholder="C:\path\to\skills（会递归查找 SKILL.md）"
                  @change="scanFolder"
                />
                <button class="btn" :disabled="folderScanning" @click="chooseFolder">
                  <Icon name="folder-open" size="sm" />选择…
                </button>
              </div>
              <div class="form-hint">递归扫描子目录；命中 SKILL.md 的目录会被识别为一个技能，并自动追溯链接源头。</div>
            </div>
            <div v-if="folderError" class="inline-error"><Icon name="alert" size="sm" /><span>{{ folderError }}</span></div>
            <div v-if="folderScanning" class="inspect-loading">
              <Icon name="refresh" class="spin" size="sm" />正在递归扫描文件夹…
            </div>
            <div v-else-if="scannedFolder && folderItems.length === 0" class="inspect-placeholder">
              <Icon name="folder" size="lg" />
              <h3>没有发现技能</h3>
              <p>选中目录及其子目录中没有包含 SKILL.md 的文件夹。</p>
            </div>
            <div v-else-if="!scannedFolder && !folderPath" class="inspect-placeholder">
              <Icon name="folder" size="lg" />
              <h3>选择一个技能根目录</h3>
              <p>可以是一个技能目录，也可以是包含多个技能的仓库或下载目录。</p>
            </div>
          </template>

          <template v-else-if="tab === 'existing'">
            <div class="existing-toolbar">
              <label class="search" style="width: 100%">
                <Icon name="search" size="sm" />
                <input v-model="existingQuery" placeholder="搜索技能、Agent 或源头路径…" aria-label="搜索现有技能" />
              </label>
              <button class="btn ghost sm" :disabled="existingLoading" @click="loadExisting">
                <Icon name="refresh" size="sm" :class="{ spin: existingLoading }" />重新扫描
              </button>
            </div>
            <div class="form-hint">Junction / Symlink 会先追溯真实源头，再把源头内容复制进仓库，不会导入链接本身。</div>
            <div v-if="existingError" class="inline-error"><Icon name="alert" size="sm" />{{ existingError }}</div>
            <div v-if="existingLoading" class="inspect-loading">
              <Icon name="refresh" class="spin" size="sm" />正在扫描 Agent 目录并追溯链接…
            </div>
            <div v-else-if="!existingLoading && existingItems.length === 0" class="inspect-placeholder">
              <Icon name="inbox" size="lg" />
              <h3>没有发现可导入技能</h3>
              <p>已检测的 Agent / 项目目录中没有包含 SKILL.md 的文件夹。</p>
            </div>
          </template>

          <template v-else>
            <div class="form-block">
              <label class="form-label" for="network-url">网络地址</label>
              <div class="path-picker">
                <input
                  id="network-url"
                  v-model="networkURL"
                  class="input"
                  placeholder="https://github.com/owner/repo 或 .zip / .tar.gz"
                  @keydown.enter="runNetwork"
                />
                <button class="btn" :disabled="networkLoading" @click="runNetwork">
                  <Icon name="download" size="sm" />{{ networkLoading ? '下载中…' : '下载并扫描' }}
                </button>
              </div>
              <div class="form-hint">
                支持 GitHub 仓库、<code>/tree/分支/子目录</code>、raw.githubusercontent.com 的 SKILL.md，以及 zip / tar.gz 压缩包。
                下载后解压到缓存，导入时才复制进仓库。
              </div>
            </div>
            <div v-if="networkError" class="inline-error"><Icon name="alert" size="sm" />{{ networkError }}</div>
            <div v-if="networkLoading" class="inspect-loading">
              <Icon name="refresh" class="spin" size="sm" />正在下载、解压并扫描技能…
            </div>
            <div v-else-if="networkDone && networkItems.length === 0" class="inspect-placeholder">
              <Icon name="external" size="lg" />
              <h3>没有发现技能</h3>
              <p>下载成功，但压缩包或仓库中没有包含 SKILL.md 的目录。</p>
            </div>
            <div v-else-if="!networkDone && !networkLoading" class="inspect-placeholder">
              <Icon name="external" size="lg" />
              <h3>从网络发现技能</h3>
              <p>输入 GitHub 仓库/目录，或 zip / tar.gz 地址；下载后会列出仓库内所有可导入技能。</p>
            </div>
          </template>

          <div v-if="tabItems.length > 0" class="candidate-head">
            <span class="candidate-count">
              显示 {{ activeItems.length }} / {{ tabItems.length }} 个候选 · 已选 {{ selected.size }}
            </span>
            <button class="mini-link" :disabled="selectableItems.length === 0" @click="selectAll">
              {{ allSelected ? '取消全选' : '全选未收录' }}
            </button>
          </div>

          <div v-if="activeItems.length > 0" class="existing-list candidate-list">
            <button
              v-for="item in activeItems"
              :key="item.path"
              class="existing-item"
              :class="{ active: selected.has(item.path), disabled: item.alreadyInstalled }"
              :disabled="item.alreadyInstalled"
              @click="toggle(item)"
            >
              <span class="existing-check">
                <Icon v-if="selected.has(item.path)" name="check" size="sm" />
              </span>
              <span class="existing-copy">
                <span class="existing-name">
                  {{ item.name }}
                  <span v-if="item.alreadyInstalled" class="tag">已收录</span>
                  <span v-if="item.isLink" class="tag">追溯链接</span>
                </span>
                <span class="existing-desc">{{ item.description || '暂无描述' }}</span>
                <span class="existing-meta">
                  <template v-if="item.agents.length">{{ item.agents.join(' · ') }} · </template>
                  {{ formatBytes(item.sizeBytes) }} · {{ item.fileCount }} files
                </span>
                <span class="existing-source" :title="item.sourceRef || item.path">{{ sourceLabel(item) }}</span>
              </span>
              <span
                class="tag content-tag"
                :class="contentTagClass(item.content)"
                :title="contentTooltip(item.content, item.contentFiles)"
              >{{ contentLabel(item.content) }}</span>
            </button>
          </div>

          <div v-if="tab === 'existing' && activeItems.length === 0 && existingQuery && tabItems.length > 0" class="inspect-placeholder small">
            <Icon name="search" size="lg" />
            <h3>没有匹配搜索的技能</h3>
            <p>试试更短的名称或路径关键词。</p>
          </div>
        </section>

        <aside class="add-right">
          <div class="option-section">
            <div class="option-head">
              <span class="form-label">分发到 Agent / 项目</span>
              <button class="mini-link" @click="selectAllAgents">
                {{ selectedAgents.size === enabledTargets.length ? '全不选' : '全选' }}
              </button>
            </div>
            <div class="agent-pick-list">
              <label
                v-for="agent in enabledTargets"
                :key="agent.id"
                class="agent-pick"
                :class="{ checked: selectedAgents.has(agent.id) }"
              >
                <input type="checkbox" :checked="selectedAgents.has(agent.id)" @change="toggleAgent(agent.id)" />
                <span class="agent-mark" :class="`tone-${agentTone(agent.id)}`">{{ agentMark(agent.id, agent.name) }}</span>
                <span class="agent-pick-copy">
                  <span>{{ agent.name }}</span>
                  <small>{{ agentSubline(agent.kind, agent.installed) }}</small>
                </span>
                <span v-if="agent.kind === 'project'" class="tag">项目</span>
                <span class="agent-check"><Icon v-if="selectedAgents.has(agent.id)" name="check" size="sm" /></span>
              </label>
              <div v-if="enabledTargets.length === 0" class="muted small">
                还没有启用 Agent 或项目目录。也可以先只导入仓库。
              </div>
            </div>
            <div class="form-hint">选择项目目录时，技能会分发到该项目的技能目录。</div>
          </div>

          <div class="option-section">
            <span class="form-label">安装方式</span>
            <div class="mode-cards">
              <button class="mode-card" :class="{ active: mode === linkMode }" @click="mode = linkMode">
                <Icon name="link" />
                <span><strong>{{ linkLabel }}</strong><small>不占空间，仓库更新即生效</small></span>
                <span class="mode-radio" />
              </button>
              <button class="mode-card" :class="{ active: mode === 'copy' }" @click="mode = 'copy'">
                <Icon name="copy" />
                <span><strong>复制</strong><small>独立副本，可单独修改</small></span>
                <span class="mode-radio" />
              </button>
            </div>
          </div>

          <div class="option-note">
            <Icon name="shield" size="sm" />
            <span>下载与导入都不会执行技能内容；链接会被追溯为真实源头，复制进仓库后再分发。</span>
          </div>
        </aside>
      </div>

      <footer class="modal-foot">
        <div class="foot-summary">
          <span class="dot" :class="selectedItems.length ? 'moss' : ''" />
          <span>{{ summary }}</span>
        </div>
        <button class="btn ghost" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="!canImport" @click="submit">
          <Icon name="download" size="sm" />{{ busy ? '导入中…' : `导入 ${selectedItems.length} 个技能` }}
        </button>
      </footer>
    </div>
  </div>
</template>
