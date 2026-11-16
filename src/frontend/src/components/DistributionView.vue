<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Agent, Snapshot } from '../lib/types'
import {
  agentMark,
  agentTone,
  formatBytes,
  modeLabel,
  shortPath,
  statusLabel,
} from '../lib/format'
import { removeProject } from '../lib/appState'
import Icon from './Icon.vue'
import ProjectDialog from './ProjectDialog.vue'

const props = defineProps<{
  snapshot: Snapshot
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'toggle-distribution', skillId: string, agentId: string, enabled: boolean): void
  (e: 'set-agent-enabled', agentId: string, enabled: boolean): void
  (e: 'open-agent-folder', agentId: string): void
  (e: 'select', skillId: string): void
  (e: 'add'): void
}>()

const enabledAgents = computed(() => props.snapshot.agents.filter((a) => a.enabled))
const disabledAgents = computed(() => props.snapshot.agents.filter((a) => !a.enabled && a.kind !== 'project'))
const skills = computed(() => props.snapshot.skills)
const projectOpen = ref(false)
const enabledCount = computed(() => enabledAgents.value.length)
const projectCount = computed(() => enabledAgents.value.filter((a) => a.kind === 'project').length)
const distKey = (skillId: string, agentId: string) => `${skillId}::${agentId}`
const distMap = computed(() => {
  const map: Record<string, (typeof props.snapshot.distributions)[number]> = {}
  for (const dist of props.snapshot.distributions) map[distKey(dist.skillId, dist.agentId)] = dist
  return map
})

const linkCount = computed(() => props.snapshot.stats.links)
const copyCount = computed(() => props.snapshot.stats.copies)
const brokenCount = computed(() => props.snapshot.stats.broken)

const agentCardMeta = (agent: Agent) => {
  const skillsForAgent = props.snapshot.distributions.filter((d) => d.agentId === agent.id)
  const links = skillsForAgent.filter((d) => d.mode !== 'copy').length
  const copies = skillsForAgent.filter((d) => d.mode === 'copy').length
  return { links, copies, total: skillsForAgent.length }
}

function cellTitle(skillName: string, agentName: string, dist?: (typeof props.snapshot.distributions)[number]) {
  if (!dist) return `将 ${skillName} 分发到 ${agentName}`
  return `${skillName} → ${agentName} · ${modeLabel(dist.mode)} · ${statusLabel(dist.status)}（点击移除）`
}

function onCell(skillId: string, agentId: string) {
  const dist = distMap.value[distKey(skillId, agentId)]
  emit('toggle-distribution', skillId, agentId, !dist)
}
</script>

<template>
  <div class="page">
    <header class="page-head">
      <div>
        <div class="eyebrow">FAN-OUT · 分发轨</div>
        <h1>分发</h1>
        <p class="page-sub">
          一个技能，多条分发轨。Junction 不占空间并随仓库更新；复制保留独立副本。
        </p>
      </div>
      <div class="page-actions">
        <button class="btn" @click="projectOpen = true"><Icon name="plus" />项目目录</button>
        <div class="mode-legend">
          <span><Icon name="link" size="sm" />{{ snapshot.vault.platform.startsWith('windows') ? 'Junction' : 'Symlink' }}</span>
          <span><Icon name="copy" size="sm" />复制</span>
        </div>
      </div>
    </header>

    <section class="ledger">
      <div class="ledger-item">
        <div class="ledger-value">{{ enabledCount }}</div>
        <div class="ledger-label">已启用目标</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ linkCount }}<small>links</small></div>
        <div class="ledger-label">链接</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ copyCount }}<small>copies</small></div>
        <div class="ledger-label">复制</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value" :class="{ 'value-alert': brokenCount > 0 }">{{ brokenCount }}</div>
        <div class="ledger-label">需修复</div>
      </div>
      <div class="ledger-note">
        {{ enabledCount - projectCount }} 个 Agent · {{ projectCount }} 个项目目录<br />
        {{ formatBytes(snapshot.stats.vaultSizeBytes) }} 仓库内容
      </div>
    </section>

    <section class="agent-section">
      <div class="section-head">
        <span class="section-title">已启用 Agent / 项目</span>
        <span class="section-count">{{ enabledCount }} 个 · 点击卡片打开目录</span>
      </div>
      <div v-if="enabledCount === 0" class="inline-empty">
        <Icon name="alert" size="sm" />
        <span>还没有启用 Agent 或项目目录。从下方列表启用一个，才能建立分发。</span>
      </div>
      <div class="agent-grid">
        <div v-for="agent in enabledAgents" :key="agent.id" class="agent-card">
          <div class="agent-card-head">
            <span class="agent-mark lg" :class="`tone-${agentTone(agent.id)}`">{{ agentMark(agent.id, agent.name) }}</span>
            <div class="agent-card-title">
              <span class="agent-name">{{ agent.name }} <span v-if="agent.kind === 'project'" class="tag">项目</span></span>
              <span class="agent-path" :title="agent.skillsPath">{{ shortPath(agent.skillsPath, 34) }}</span>
            </div>
            <span class="agent-state" :class="agent.kind === 'project' ? 'declared' : agent.detected ? 'detected' : 'declared'">
              <span class="dot" :class="agent.kind === 'project' ? 'ochre' : agent.detected ? 'moss' : 'ochre'" />
              {{ agent.kind === 'project' ? '项目目录' : agent.detected ? '已检测' : '已声明' }}
            </span>
          </div>
          <div class="agent-card-stats">
            <span><strong>{{ agentCardMeta(agent).total }}</strong> 条分发</span>
            <span>{{ agentCardMeta(agent).links }} 链接 · {{ agentCardMeta(agent).copies }} 复制</span>
          </div>
          <div class="agent-card-actions">
            <button class="btn ghost sm" @click="emit('open-agent-folder', agent.id)">
              <Icon name="folder-open" size="sm" />打开
            </button>
            <button
              v-if="agent.kind === 'project'"
              class="btn ghost sm"
              :disabled="busy"
              @click="removeProject(agent.id)"
            >移除项目</button>
            <button
              v-else
              class="btn ghost sm"
              :disabled="busy"
              @click="emit('set-agent-enabled', agent.id, false)"
            >停用</button>
          </div>
        </div>
      </div>
    </section>

    <section class="matrix-section">
      <div class="section-head">
        <span class="section-title">分发矩阵</span>
        <span class="section-count">{{ skills.length }} 技能 × {{ enabledCount }} 目标</span>
      </div>

      <div v-if="skills.length === 0" class="inline-empty large">
        <Icon name="share" size="lg" />
        <div>
          <h3>还没有可分发的技能</h3>
          <p>先把一个技能导入仓库，再回到这里建立分发轨。</p>
          <button class="btn primary sm" @click="emit('add')"><Icon name="plus" size="sm" />添加技能</button>
        </div>
      </div>

      <div v-else-if="enabledCount === 0" class="inline-empty large">
        <Icon name="alert" size="lg" />
        <div>
          <h3>先启用一个目标</h3>
          <p>启用 Agent 或添加项目目录后，才能建立 Junction 或复制。</p>
        </div>
      </div>

      <div v-else class="matrix-wrap">
        <div class="matrix" :style="{ '--agent-cols': enabledAgents.length }">
          <div class="matrix-head">
            <div class="matrix-corner">技能 / TARGET</div>
            <div v-for="agent in enabledAgents" :key="agent.id" class="matrix-agent" :title="agent.name">
              <span class="agent-mark" :class="`tone-${agentTone(agent.id)}`">{{ agentMark(agent.id, agent.name) }}</span>
              <span class="matrix-agent-name">{{ agent.name }}</span>
              <span v-if="agent.kind === 'project'" class="matrix-project">PRJ</span>
            </div>
          </div>
          <div v-for="skill in skills" :key="skill.id" class="matrix-row">
            <button class="matrix-skill" @click="emit('select', skill.id)">
              <span class="matrix-skill-name">{{ skill.name }}</span>
              <span class="matrix-skill-meta">{{ skill.agents.length }} / {{ enabledCount }}</span>
            </button>
            <button
              v-for="agent in enabledAgents"
              :key="agent.id"
              class="matrix-cell"
              :class="{
                on: !!distMap[distKey(skill.id, agent.id)],
                copy: distMap[distKey(skill.id, agent.id)]?.mode === 'copy',
                broken: distMap[distKey(skill.id, agent.id)] && distMap[distKey(skill.id, agent.id)].status !== 'ok',
              }"
              :title="cellTitle(skill.name, agent.name, distMap[distKey(skill.id, agent.id)])"
              :disabled="busy"
              @click="onCell(skill.id, agent.id)"
            >
              <template v-if="distMap[distKey(skill.id, agent.id)]">
                <Icon :name="distMap[distKey(skill.id, agent.id)].mode === 'copy' ? 'copy' : 'link'" size="sm" />
                <span
                  v-if="distMap[distKey(skill.id, agent.id)].status !== 'ok'"
                  class="cell-status"
                />
              </template>
              <Icon v-else name="plus" size="sm" />
            </button>
          </div>
        </div>
      </div>
      <div class="matrix-legend">
        <span><span class="legend-swatch link" />链接 / Junction</span>
        <span><span class="legend-swatch copy" />复制副本</span>
        <span><span class="legend-swatch broken" />需要修复</span>
        <span class="matrix-hint">点击单元格即可分发或移除</span>
      </div>
    </section>

    <section v-if="disabledAgents.length" class="agent-section">
      <div class="section-head">
        <span class="section-title">可用但未启用</span>
        <span class="section-count">{{ disabledAgents.length }} 个</span>
      </div>
      <div class="disabled-agent-list">
        <div v-for="agent in disabledAgents" :key="agent.id" class="disabled-agent">
          <span class="agent-mark" :class="`tone-${agentTone(agent.id)}`">{{ agentMark(agent.id, agent.name) }}</span>
          <span class="disabled-agent-copy">
            <span>{{ agent.name }}</span>
            <small>{{ shortPath(agent.skillsPath, 42) }}</small>
          </span>
          <span class="disabled-agent-state">{{ agent.detected ? '已检测 · 已停用' : '未检测到' }}</span>
          <button class="btn ghost sm" :disabled="busy" @click="emit('set-agent-enabled', agent.id, true)">
            <Icon name="plus" size="sm" />启用
          </button>
        </div>
      </div>
    </section>

    <ProjectDialog v-if="projectOpen" :busy="busy" @close="projectOpen = false" />
  </div>
</template>
