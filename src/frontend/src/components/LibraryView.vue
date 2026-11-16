<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Snapshot, Skill } from '../lib/types'
import type { Layout } from '../lib/appState'
import {
  agentMark,
  agentTone,
  contentLabel,
  contentToneClass,
  contentTooltip,
  formatBytes,
  relativeTime,
  shortPath,
  skillMark,
  skillTone,
  timeShort,
} from '../lib/format'
import Icon from './Icon.vue'
import GroupManagerDialog from './GroupManagerDialog.vue'
import BulkDistributeDialog from './BulkDistributeDialog.vue'

const props = defineProps<{
  snapshot: Snapshot
  selectedId: string | null
  layout: Layout
  query: string
  filter: string
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'select', id: string): void
  (e: 'add', tab: 'local' | 'existing'): void
  (e: 'refresh'): void
  (e: 'set-layout', layout: Layout): void
  (e: 'update:query', value: string): void
  (e: 'update:filter', value: string): void
  (e: 'open-folder', id: string): void
}>()

const sourceFilter = ref('all')
const groupFilter = ref('all')
const groupManagerOpen = ref(false)
const bulkOpen = ref(false)
const bulkSkillIds = ref<string[]>([])
const bulkLabel = ref('')

const agentName = computed(() => {
  const map: Record<string, string> = {}
  for (const agent of props.snapshot.agents) map[agent.id] = agent.name
  return map
})

const skillAgentMarks = (skill: Skill) =>
  skill.agents.slice(0, 4).map((id) => ({
    id,
    mark: agentMark(id, agentName.value[id]),
    tone: agentTone(id),
    name: agentName.value[id] || id,
  }))

const filtered = computed(() => {
  const q = props.query.trim().toLowerCase()
  return props.snapshot.skills.filter((skill) => {
    if (q) {
      const haystack = `${skill.name} ${skill.description} ${skill.notes} ${skill.tags.join(' ')} ${skill.groupName} ${skill.id} ${skill.sourceName}`.toLowerCase()
      if (!haystack.includes(q)) return false
    }
    if (sourceFilter.value !== 'all' && skill.sourceId !== sourceFilter.value) return false
    if (groupFilter.value === 'ungrouped' && skill.groupId) return false
    if (groupFilter.value !== 'all' && groupFilter.value !== 'ungrouped' && skill.groupId !== groupFilter.value) return false
    switch (props.filter) {
      case 'link':
        return skill.linkCount > 0
      case 'copy':
        return skill.copyCount > 0
      case 'attention':
        return skill.brokenCount > 0 || skill.updateAvailable
      default:
        return true
    }
  })
})

const hasAny = computed(() => props.snapshot.skills.length > 0)
const filteredCount = computed(() => filtered.value.length)
const attentionCount = computed(() => props.snapshot.stats.broken + props.snapshot.stats.updates)

const sourceKind = (id: string) => {
  const source = props.snapshot.sources.find((s) => s.id === id)
  if (!source) return 'LOCAL'
  if (source.kind === 'git') return 'GIT'
  if (source.kind === 'registry') return 'REGISTRY'
  return 'LOCAL'
}

const activityIcon = (action: string) => {
  switch (action) {
    case 'import': return 'download'
    case 'distribute': return 'share'
    case 'remove': return 'trash'
    case 'repair': return 'shield'
    case 'skin': return 'palette'
    case 'unlink': return 'x'
    case 'edit': return 'edit'
    case 'group': return 'book'
    case 'source': return 'database'
    case 'vault': return 'folder-open'
    case 'project': return 'folder'
    case 'remove-directory': return 'trash'
    default: return 'clock'
  }
}

function openBulk(skillIds: string[], label: string) {
  bulkSkillIds.value = skillIds
  bulkLabel.value = label
  groupManagerOpen.value = false
  bulkOpen.value = true
}

function clearFilters() {
  emit('update:query', '')
  emit('update:filter', 'all')
  sourceFilter.value = 'all'
  groupFilter.value = 'all'
}
</script>

<template>
  <div class="page">
    <header class="page-head">
      <div>
        <div class="eyebrow">SKILL LEDGER · 技能台账</div>
        <h1>技能库</h1>
        <p class="page-sub">
          单一事实源位于 <code>{{ snapshot.vault.path }}</code>，每一行都是一条可核对的技能记录。
        </p>
      </div>
      <div class="page-actions">
        <button class="btn ghost" :disabled="busy" @click="emit('refresh')">
          <Icon name="refresh" :class="{ spin: busy }" />{{ busy ? '扫描中…' : '刷新索引' }}
        </button>
        <button class="btn primary" @click="emit('add', 'local')">
          <Icon name="plus" />添加技能
        </button>
      </div>
    </header>

    <section class="ledger" aria-label="台账统计">
      <div class="ledger-item">
        <div class="ledger-value">{{ snapshot.stats.skills }}</div>
        <div class="ledger-label">技能总数</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ snapshot.stats.links }}<small>links</small></div>
        <div class="ledger-label">链接分发</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ snapshot.stats.copies }}<small>copies</small></div>
        <div class="ledger-label">复制副本</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value" :class="{ 'value-alert': attentionCount > 0 }">{{ attentionCount }}</div>
        <div class="ledger-label">需处理</div>
      </div>
      <div class="ledger-note">
        {{ snapshot.stats.detectedAgents }} 个目标已检测 · {{ snapshot.stats.enabledAgents }} 个已启用<br />
        扫描于 {{ timeShort(snapshot.vault.lastScanAt) }}
      </div>
    </section>

    <template v-if="hasAny">
      <div class="toolbar">
        <label class="search">
          <Icon name="search" size="sm" />
          <input
            :value="query"
            type="search"
            placeholder="搜索名称、描述或来源…"
            aria-label="搜索技能"
            @input="emit('update:query', ($event.target as HTMLInputElement).value)"
          />
          <kbd>/</kbd>
        </label>
        <div class="seg" role="group" aria-label="分发筛选">
          <button :class="{ active: filter === 'all' }" @click="emit('update:filter', 'all')">全部</button>
          <button :class="{ active: filter === 'link' }" @click="emit('update:filter', 'link')">链接</button>
          <button :class="{ active: filter === 'copy' }" @click="emit('update:filter', 'copy')">复制</button>
          <button :class="{ active: filter === 'attention' }" @click="emit('update:filter', 'attention')">需注意</button>
        </div>
        <select v-model="sourceFilter" class="select source-select" aria-label="来源筛选">
          <option value="all">全部来源</option>
          <option v-for="source in snapshot.sources" :key="source.id" :value="source.id">
            {{ source.name }}
          </option>
        </select>
        <select v-model="groupFilter" class="select source-select" aria-label="分组筛选">
          <option value="all">全部分组</option>
          <option value="ungrouped">未分组</option>
          <option v-for="group in snapshot.groups" :key="group.id" :value="group.id">
            {{ group.name }}（{{ group.skillCount }}）
          </option>
        </select>
        <button class="btn ghost sm group-manage-btn" @click="groupManagerOpen = true">
          <Icon name="book" size="sm" />管理分组
        </button>
        <div class="toolbar-spacer" />
        <span class="result-count">{{ filteredCount }} / {{ snapshot.stats.skills }}</span>
        <div class="view-toggle" role="group" aria-label="布局">
          <button :class="{ active: layout === 'table' }" title="表格" @click="emit('set-layout', 'table')">
            <Icon name="list" />
          </button>
          <button :class="{ active: layout === 'cards' }" title="卡片" @click="emit('set-layout', 'cards')">
            <Icon name="grid" />
          </button>
        </div>
      </div>

      <div v-if="filtered.length === 0" class="no-results">
        <Icon name="search" size="lg" />
        <h3>没有匹配的技能</h3>
        <p>试试更短的关键词，或清除来源与分发筛选。</p>
        <button class="btn sm" @click="clearFilters">
          清除筛选
        </button>
      </div>

      <template v-else-if="layout === 'table'">
        <div class="table">
          <div class="thead">
            <span>#</span>
            <span>技能 / SKILL</span>
            <span>来源</span>
            <span>分发到</span>
            <span>大小</span>
            <span>状态</span>
            <span></span>
          </div>
          <div
            v-for="(skill, index) in filtered"
            :key="skill.id"
            class="trow"
            :class="{ selected: skill.id === selectedId }"
            tabindex="0"
            role="button"
            :aria-label="`查看 ${skill.name}`"
            @click="emit('select', skill.id)"
            @keydown.enter="emit('select', skill.id)"
          >
            <span class="row-num">{{ String(index + 1).padStart(2, '0') }}</span>
            <div class="skill-cell">
              <span class="tile" :class="`tone-${skillTone(skill.name)}`">{{ skillMark(skill.name) }}</span>
              <span class="skill-text">
                <span class="skill-name">
                  {{ skill.name }}
                  <span v-if="skill.updateAvailable" class="mini-badge">UPDATE</span>
                  <span v-if="skill.groupName" class="group-inline">{{ skill.groupName }}</span>
                  <span
                    class="tag content-tag table-content-tag"
                    :class="contentToneClass(skill.content)"
                    :title="contentTooltip(skill.content, skill.contentFiles)"
                  >{{ contentLabel(skill.content) }}</span>
                </span>
                <span class="skill-desc">{{ skill.description || '暂无描述' }}</span>
              </span>
            </div>
            <div class="source-cell">
              {{ skill.sourceName }}
              <span>{{ sourceKind(skill.sourceId) }} · {{ skill.version ? 'v' + skill.version : '—' }}</span>
            </div>
            <div class="dist-cell">
              <span v-if="skill.agents.length === 0" class="muted">未分发</span>
              <template v-else>
                <div class="chip-stack">
                  <span
                    v-for="agent in skillAgentMarks(skill)"
                    :key="agent.id"
                    class="agent-mark"
                    :class="`tone-${agent.tone}`"
                    :title="agent.name"
                  >{{ agent.mark }}</span>
                  <span v-if="skill.agents.length > 4" class="agent-more">+{{ skill.agents.length - 4 }}</span>
                </div>
                <span class="dist-mode" :title="skill.copyCount ? '含复制副本' : '全部为链接'">
                  <Icon :name="skill.copyCount ? 'copy' : 'link'" size="sm" />
                </span>
              </template>
            </div>
            <div class="size-cell">
              {{ formatBytes(skill.sizeBytes) }}
              <span>{{ skill.fileCount }} files</span>
            </div>
            <div>
              <span v-if="skill.brokenCount" class="status status-bad"><span class="dot clay" />{{ skill.brokenCount }} 条异常</span>
              <span v-else-if="skill.updateAvailable" class="status status-warn"><span class="dot ochre" />可更新</span>
              <span v-else class="status"><span class="dot" :class="{ moss: skill.linkCount > 0 }" />{{ skill.linkCount > 0 ? '已同步' : '仅在仓库' }}</span>
            </div>
            <button
              class="row-menu"
              title="打开技能目录"
              aria-label="打开技能目录"
              @click.stop="emit('open-folder', skill.id)"
            >
              <Icon name="folder-open" size="sm" />
            </button>
          </div>
        </div>
      </template>

      <div v-else class="cards">
        <button
          v-for="skill in filtered"
          :key="skill.id"
          class="card"
          :class="{ selected: skill.id === selectedId }"
          @click="emit('select', skill.id)"
        >
          <span class="card-head">
            <span class="tile lg" :class="`tone-${skillTone(skill.name)}`">{{ skillMark(skill.name) }}</span>
            <span class="card-title">
              <span class="card-name">{{ skill.name }}</span>
              <span class="card-source">{{ skill.sourceName }} · {{ formatBytes(skill.sizeBytes) }}</span>
            </span>
            <Icon name="chevron-right" size="sm" class="card-arrow" />
          </span>
          <span class="card-desc">{{ skill.description || '暂无描述' }}</span>
          <span class="card-tags">
            <span v-if="skill.groupName" class="tag group-tag">{{ skill.groupName }}</span>
            <span
              class="tag content-tag"
              :class="contentToneClass(skill.content)"
              :title="contentTooltip(skill.content, skill.contentFiles)"
            >{{ contentLabel(skill.content) }}</span>
            <span v-for="tag in skill.tags.slice(0, 3)" :key="tag" class="tag">#{{ tag }}</span>
            <span v-if="skill.copyCount" class="tag">COPY {{ skill.copyCount }}</span>
            <span v-if="skill.updateAvailable" class="tag tag-stamp">可更新</span>
          </span>
          <span class="card-foot">
            <span v-if="skill.agents.length === 0" class="muted">未分发</span>
            <span v-else class="chip-stack">
              <span
                v-for="agent in skillAgentMarks(skill)"
                :key="agent.id"
                class="agent-mark"
                :class="`tone-${agent.tone}`"
                :title="agent.name"
              >{{ agent.mark }}</span>
              <span v-if="skill.agents.length > 4" class="agent-more">+{{ skill.agents.length - 4 }}</span>
            </span>
            <span class="card-mode">
              <Icon :name="skill.copyCount ? 'copy' : 'link'" size="sm" />
              {{ skill.copyCount ? '含复制' : '链接' }}
            </span>
          </span>
        </button>
      </div>

      <section v-if="snapshot.activity && snapshot.activity.length" class="activity-strip">
        <div class="section-head">
          <span class="section-title">最近活动</span>
          <span class="section-count">ACTIVITY · 活动</span>
        </div>
        <div class="activity-list">
          <div v-for="item in snapshot.activity.slice(0, 4)" :key="item.id" class="activity-row">
            <span class="activity-ic"><Icon :name="activityIcon(item.action)" size="sm" /></span>
            <span class="activity-subject">{{ item.subject }}</span>
            <span class="activity-detail">{{ item.detail }}</span>
            <span class="activity-time">{{ relativeTime(item.createdAt) }}</span>
          </div>
        </div>
      </section>
    </template>

    <section v-else class="empty-ledger">
      <div class="empty-graphic" aria-hidden="true">
        <svg viewBox="0 0 160 120">
          <path class="g-rule" d="M16 18h128M16 102h128" />
          <rect class="g-box" x="30" y="30" width="100" height="60" rx="5" />
          <path class="g-line" d="M30 48h100M52 30v60M108 30v60" />
          <path class="g-accent" d="M65 62h30M80 47v30" />
          <circle class="g-dot" cx="44" cy="39" r="2.5" />
          <circle class="g-dot" cx="120" cy="39" r="2.5" />
        </svg>
      </div>
      <div class="eyebrow" style="justify-content: center">THE LEDGER IS EMPTY</div>
      <h2>仓库尚未收录技能</h2>
      <p>
        把散落在各个 Agent 目录里的技能收进 <code>~/.skill-vault</code>，
        然后用 Junction 或复制，一次分发到多个 Agent。原始内容不会被修改。
      </p>
      <div class="empty-actions">
        <button class="btn primary" @click="emit('add', 'local')">
          <Icon name="plus" />从本地目录导入
        </button>
        <button class="btn" @click="emit('add', 'existing')">
          <Icon name="inbox" />扫描现有安装
        </button>
      </div>
      <div class="empty-steps">
        <div class="empty-step">
          <span class="step-num">01</span>
          <span class="step-title">收录</span>
          <span class="step-copy">复制一份到仓库，记录哈希、大小与内容构成。</span>
        </div>
        <div class="empty-step">
          <span class="step-num">02</span>
          <span class="step-title">分发</span>
          <span class="step-copy">为每个 Agent 建 Junction，或复制独立副本。</span>
        </div>
        <div class="empty-step">
          <span class="step-num">03</span>
          <span class="step-title">核对</span>
          <span class="step-copy">断链与副本漂移会进入台账，随时可修复。</span>
        </div>
      </div>
    </section>

    <GroupManagerDialog
      v-if="groupManagerOpen"
      :snapshot="snapshot"
      :busy="busy"
      @close="groupManagerOpen = false"
      @bulk="openBulk"
    />
    <BulkDistributeDialog
      v-if="bulkOpen"
      :snapshot="snapshot"
      :skill-ids="bulkSkillIds"
      :label="bulkLabel"
      :busy="busy"
      @close="bulkOpen = false"
    />
  </div>
</template>
