<script setup lang="ts">
import { computed } from 'vue'
import type { Snapshot } from '../lib/types'
import { agentMark, agentTone, formatBytes, relativeTime, timeShort } from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'check'): void
  (e: 'select', id: string): void
  (e: 'navigate-library'): void
}>()

const updates = computed(() => props.snapshot.skills.filter((s) => s.updateAvailable))
const affected = computed(() => updates.value.reduce((n, s) => n + s.agents.length, 0))
const lastCheck = computed(() => props.snapshot.settings['updates.lastCheckAt'] || '')
</script>

<template>
  <div class="page">
    <header class="page-head">
      <div>
        <div class="eyebrow">REVISIONS · 修订</div>
        <h1>更新</h1>
        <p class="page-sub">检查来源是否有新提交，并查看一次更新会影响哪些 Agent。v1 只提示，不自动合并。</p>
      </div>
      <div class="page-actions">
        <button class="btn" :disabled="busy" @click="emit('check')">
          <Icon name="refresh" :class="{ spin: busy }" />{{ busy ? '检查中…' : '检查更新' }}
        </button>
        <button v-if="updates.length" class="btn primary" @click="emit('check')">
          <Icon name="arrow-down" />重新核对
        </button>
      </div>
    </header>

    <section class="ledger">
      <div class="ledger-item">
        <div class="ledger-value" :class="{ 'value-alert': updates.length > 0 }">{{ updates.length }}</div>
        <div class="ledger-label">可更新技能</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ affected }}</div>
        <div class="ledger-label">受影响分发</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ snapshot.stats.skills }}</div>
        <div class="ledger-label">纳入核对</div>
      </div>
      <div class="ledger-note">
        最近检查 {{ lastCheck ? relativeTime(lastCheck) : '尚未检查' }}<br />
        来源缓存与差异保存在本地记录中
      </div>
    </section>

    <div v-if="updates.length === 0" class="updates-empty">
      <div class="empty-graphic small">
        <svg viewBox="0 0 120 100">
          <circle class="g-circle" cx="60" cy="48" r="28" />
          <path class="g-accent" d="m47 48 9 9 18-20" />
          <path class="g-rule" d="M26 84h68" />
        </svg>
      </div>
      <div class="eyebrow" style="justify-content: center">ALL REVISIONS ACCOUNTED FOR</div>
      <h2>所有技能都是最新的</h2>
      <p>
        {{ lastCheck ? `上次检查：${relativeTime(lastCheck)}（${timeShort(lastCheck)}）` : '还没有执行过更新检查。' }}
        发现新提交后，会在这里列出受影响的分发，再由你决定是否更新。
      </p>
      <div class="empty-actions">
        <button class="btn primary" :disabled="busy" @click="emit('check')">
          <Icon name="refresh" />检查所有来源
        </button>
        <button class="btn" @click="emit('navigate-library')">
          <Icon name="book" />回到技能库
        </button>
      </div>
    </div>

    <div v-else class="update-list">
      <div class="update-list-head">
        <span>可更新的技能</span>
        <span>{{ updates.length }} 项 · v1 仅提示</span>
      </div>
      <button v-for="skill in updates" :key="skill.id" class="update-row" @click="emit('select', skill.id)">
        <span class="tile tone-ochre">{{ skill.name.slice(0, 2).toUpperCase() }}</span>
        <span class="update-copy">
          <span class="update-name">{{ skill.name }} <span class="mini-badge">UPDATE</span></span>
          <span class="update-meta">
            {{ skill.sourceName }} · 当前 {{ skill.version || 'unversioned' }} · {{ formatBytes(skill.sizeBytes) }}
          </span>
          <span class="update-agents">
            <span class="muted">影响</span>
            <span
              v-for="agentId in skill.agents"
              :key="agentId"
              class="agent-mark"
              :class="`tone-${agentTone(agentId)}`"
              :title="agentId"
            >{{ agentMark(agentId, snapshot.agents.find((a) => a.id === agentId)?.name || agentId) }}</span>
            <span class="muted">{{ skill.agents.length }} 条分发</span>
          </span>
        </span>
        <span class="update-note">来源有新提交<br /><span>不会自动合并</span></span>
        <Icon name="chevron-right" size="sm" />
      </button>
      <div class="update-footer">
        <Icon name="info" size="sm" />
        <span>v1 不在后台执行 git pull 或覆盖技能内容；更新流程将在 M6 接入来源缓存后开放。</span>
      </div>
    </div>
  </div>
</template>
