<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '../lib/api'
import type { Agent, Distribution, FileEntry, Skill, Snapshot } from '../lib/types'
import {
  agentMark,
  agentTone,
  contentLabel,
  contentSummary,
  contentToneClass,
  contentTooltip,
  formatBytes,
  modeLabel,
  relativeTime,
  skillMark,
  skillTone,
  truncate,
} from '../lib/format'
import Icon from './Icon.vue'
import EditSkillDialog from './EditSkillDialog.vue'

const props = defineProps<{
  snapshot: Snapshot
  skill: Skill
  agents: Agent[]
  distributions: Distribution[]
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'distribute'): void
  (e: 'remove'): void
  (e: 'open-folder'): void
  (e: 'remove-distribution', agentId: string): void
}>()

const tab = ref<'overview' | 'files' | 'content'>('overview')
const files = ref<FileEntry[]>([])
const filesLoading = ref(false)
const filesError = ref('')
const editOpen = ref(false)

const distByAgent = computed(() => {
  const map: Record<string, Distribution> = {}
  for (const d of props.distributions) map[d.agentId] = d
  return map
})

const rows = computed(() =>
  props.agents
    .filter((agent) => agent.enabled || distByAgent.value[agent.id])
    .sort((a, b) => {
      const ad = distByAgent.value[a.id] ? 0 : 1
      const bd = distByAgent.value[b.id] ? 0 : 1
      return ad - bd || a.name.localeCompare(b.name)
    })
    .map((agent) => ({
      agent,
      distribution: distByAgent.value[agent.id],
      mark: agentMark(agent.id, agent.name),
      tone: agentTone(agent.id),
    })),
)

const sourceCount = computed(() => props.distributions.length)
const hashShort = computed(() => (props.skill.contentHash || '').slice(0, 14))
const contentTone = computed(() => contentToneClass(props.skill.content))
const contentCopy = computed(() => contentSummary(props.skill.content))
const contentFiles = computed(() => props.skill.contentFiles || [])

async function loadFiles() {
  if (filesLoading.value) return
  filesLoading.value = true
  filesError.value = ''
  try {
    files.value = await api.skillFiles(props.skill.id)
  } catch (err) {
    filesError.value = err instanceof Error ? err.message : String(err)
  } finally {
    filesLoading.value = false
  }
}

watch(
  () => props.skill.id,
  () => {
    tab.value = 'overview'
    files.value = []
    filesError.value = ''
    editOpen.value = false
  },
)

watch(tab, (value) => {
  if (value === 'files' && files.value.length === 0) loadFiles()
})
</script>

<template>
  <aside class="inspector" aria-label="技能详情">
    <div class="insp-head">
      <span class="insp-code">SKILL · {{ skill.id.slice(0, 18).toUpperCase() }}</span>
      <span class="spacer" />
      <button class="insp-close" aria-label="关闭详情" title="关闭" @click="emit('close')">
        <Icon name="x" size="sm" />
      </button>
    </div>

    <div class="insp-body">
      <div class="insp-title-row">
        <span class="tile lg" :class="`tone-${skillTone(skill.name)}`">{{ skillMark(skill.name) }}</span>
        <div class="insp-title-copy">
          <h2 class="insp-title">{{ skill.name }}</h2>
          <div class="insp-source">
            <Icon name="database" size="sm" />
            {{ skill.sourceName || '本地导入' }}
            <span class="sep">·</span>
            <span>{{ skill.version ? 'v' + skill.version : 'unversioned' }}</span>
            <span v-if="skill.groupName" class="sep">·</span>
            <span v-if="skill.groupName" class="group-inline">{{ skill.groupName }}</span>
          </div>
        </div>
      </div>

      <div v-if="skill.tags.length" class="insp-tags">
        <span v-for="tag in skill.tags" :key="tag" class="tag">#{{ tag }}</span>
      </div>

      <p class="insp-desc">{{ skill.description || '该技能没有在 SKILL.md 中提供描述。' }}</p>

      <div class="insp-actions">
        <button class="btn primary sm" :disabled="busy" @click="emit('distribute')">
          <Icon name="share" size="sm" />分发…
        </button>
        <button class="btn sm" @click="editOpen = true">
          <Icon name="edit" size="sm" />编辑
        </button>
        <button class="btn sm" @click="emit('open-folder')">
          <Icon name="folder-open" size="sm" />目录
        </button>
        <button class="btn sm danger" :disabled="busy" @click="emit('remove')">
          <Icon name="trash" size="sm" />移除
        </button>
      </div>

      <div class="insp-status">
        <span class="dot" :class="skill.brokenCount ? 'clay' : 'moss'" />
        <span>{{ skill.brokenCount ? `${skill.brokenCount} 条分发异常` : '分发健康' }}</span>
        <span class="sep">·</span>
        <span class="mono">{{ hashShort || 'no-hash' }}</span>
        <span class="sep">·</span>
        <span>更新于 {{ relativeTime(skill.updatedAt) }}</span>
      </div>

      <div v-if="skill.brokenCount" class="drift-note">
        <Icon name="alert" size="sm" />
        <span>有 {{ skill.brokenCount }} 条分发指向了不存在或错误的位置。打开「分发」视图可查看并重建。</span>
      </div>

      <div class="section-head">
        <span class="section-title">分发轨 / FAN-OUT</span>
        <span class="section-count">{{ sourceCount }} 条</span>
      </div>

      <div class="fanout-source" :title="skill.vaultPath">
        <Icon name="package" size="sm" />
        <span class="path">{{ truncate(skill.vaultPath, 58) }}</span>
      </div>
      <div class="fanout-list">
        <div
          v-for="row in rows"
          :key="row.agent.id"
          class="fanout-row"
          :class="row.distribution ? (row.distribution.mode === 'copy' ? 'copied' : 'linked') : ''"
        >
          <span class="agent-mark" :class="`tone-${row.tone}`">{{ row.mark }}</span>
          <div class="fanout-info">
            <div class="fanout-name">{{ row.agent.name }}</div>
            <div class="fanout-path">
              {{ row.distribution ? modeLabel(row.distribution.mode) : '未分发' }} ·
              {{ truncate(row.distribution?.linkPath || row.agent.skillsPath, 46) }}
            </div>
          </div>
          <span
            v-if="row.distribution"
            class="fanout-mode"
            :class="row.distribution.mode === 'copy' ? 'copy' : 'link'"
          >{{ modeLabel(row.distribution.mode) }}</span>
          <button
            v-if="row.distribution"
            class="fanout-remove"
            title="移除该分发"
            aria-label="移除该分发"
            @click="emit('remove-distribution', row.agent.id)"
          >
            <Icon name="x" size="sm" />
          </button>
          <button v-else class="fanout-add" title="分发到此 Agent" @click="emit('distribute')">分发</button>
        </div>
        <div v-if="rows.length === 0" class="fanout-empty">
          还没有启用任何 Agent。前往「分发」视图启用一个。
        </div>
      </div>

      <div class="tabs" role="tablist">
        <button class="tab" :class="{ active: tab === 'overview' }" role="tab" @click="tab = 'overview'">概览</button>
        <button class="tab" :class="{ active: tab === 'files' }" role="tab" @click="tab = 'files'">
          文件 <span class="tab-count">{{ skill.fileCount }}</span>
        </button>
        <button class="tab" :class="{ active: tab === 'content' }" role="tab" @click="tab = 'content'">内容</button>
      </div>

      <div v-if="tab === 'overview'" class="insp-tab-body">
        <dl class="meta">
          <div><dt>来源</dt><dd>{{ skill.sourceName || '本地导入' }}</dd></div>
          <div><dt>版本</dt><dd>{{ skill.version || '—' }}</dd></div>
          <div><dt>分组</dt><dd>{{ skill.groupName || '未分组' }}</dd></div>
          <div><dt>内容</dt><dd class="content-inline" :class="contentTone" :title="contentTooltip(skill.content, skill.contentFiles)">{{ contentLabel(skill.content) }}</dd></div>
          <div><dt>大小</dt><dd>{{ formatBytes(skill.sizeBytes) }}</dd></div>
          <div><dt>文件数</dt><dd>{{ skill.fileCount }}</dd></div>
          <div><dt>安装时间</dt><dd>{{ relativeTime(skill.installedAt) }}</dd></div>
          <div><dt>标签</dt><dd>{{ skill.tags.length }} 个</dd></div>
        </dl>
        <div v-if="skill.notes" class="insp-notes">
          <div class="hash-label">NOTES · 备注</div>
          <p>{{ skill.notes }}</p>
        </div>
        <div class="hash-box">
          <div class="hash-label">CONTENT HASH · SHA-256</div>
          <div class="hash-value">{{ skill.contentHash || '尚未计算' }}</div>
        </div>
        <div class="insp-note">
          <Icon name="info" size="sm" />
          <span>SKILL.md 是技能的元数据来源；本地索引保存来源与分发状态，可随时重建。</span>
        </div>
      </div>

      <div v-else-if="tab === 'files'" class="insp-tab-body">
        <div class="file-head">
          <span>{{ files.length }} 个文件</span>
          <button class="btn ghost sm" :disabled="filesLoading" @click="loadFiles">
            <Icon name="refresh" size="sm" :class="{ spin: filesLoading }" />刷新
          </button>
        </div>
        <div v-if="filesLoading" class="file-row muted">读取目录…</div>
        <div v-else-if="filesError" class="file-row file-error">{{ filesError }}</div>
        <div v-else class="file-list">
          <div v-for="file in files" :key="file.path" class="file-row" :title="file.path">
            <Icon name="file" size="sm" />
            <span class="file-path">{{ file.path }}</span>
            <span class="file-size">{{ formatBytes(file.size) }}</span>
          </div>
        </div>
      </div>

      <div v-else class="insp-tab-body">
        <div class="content-panel" :class="contentTone" :title="contentTooltip(skill.content, skill.contentFiles)">
          <span class="content-level">{{ contentLabel(skill.content) }}</span>
          <p>{{ contentCopy }}</p>
        </div>

        <div v-if="contentFiles.length" class="content-files">
          <div class="hash-label">CONTENT FILES · 相关文件</div>
          <ul>
            <li v-for="file in contentFiles" :key="file">
              <Icon name="file" size="sm" />
              <span>{{ file }}</span>
            </li>
          </ul>
        </div>
        <div v-else class="content-files content-files-empty">
          目录中没有脚本或可执行文件。
        </div>

        <div class="section-head" style="margin-top: 14px">
          <span class="section-title">说明</span>
        </div>
        <div class="content-list">
          <div class="content-row">
            <Icon name="shield" size="sm" />
            <span>安装与分发不会执行技能中的任何内容</span>
            <span class="content-note">说明</span>
          </div>
        </div>
        <button class="btn sm content-open" @click="tab = 'files'">查看全部文件</button>
      </div>
    </div>
  </aside>

  <Teleport to="body">
    <EditSkillDialog
      v-if="editOpen"
      :snapshot="snapshot"
      :skill="skill"
      :busy="busy"
      @close="editOpen = false"
    />
  </Teleport>
</template>
