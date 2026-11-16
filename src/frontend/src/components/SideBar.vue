<script setup lang="ts">
import { computed } from 'vue'
import type { Snapshot } from '../lib/types'
import { formatBytes, shortPath } from '../lib/format'
import type { View } from '../lib/appState'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
  view: View
}>()

const emit = defineEmits<{
  (e: 'navigate', view: View): void
  (e: 'open-vault'): void
}>()

const stats = computed(() => props.snapshot.stats)
const totalDists = computed(() => stats.value.links + stats.value.copies)
const linkPct = computed(() =>
  totalDists.value ? Math.max(2, Math.round((stats.value.links / totalDists.value) * 100)) : 0,
)
const copyPct = computed(() =>
  totalDists.value ? Math.max(0, 100 - linkPct.value) : 0,
)
const repoLabel = computed(() => shortPath(props.snapshot.vault.path, 30))
const targetsCount = computed(
  () => props.snapshot.agents.filter((a) => a.kind === 'project' || a.enabled || a.detected).length,
)
</script>

<template>
  <aside class="sidebar">
    <div class="nav-group">
      <div class="nav-label">工作台</div>
      <button
        class="nav-item"
        :class="{ active: view === 'library' }"
        title="技能库"
        @click="emit('navigate', 'library')"
      >
        <Icon name="book" /><span>技能库</span>
        <span class="nav-count">{{ stats.skills }}</span>
      </button>
      <button
        class="nav-item"
        :class="{ active: view === 'updates' }"
        title="更新"
        @click="emit('navigate', 'updates')"
      >
        <Icon name="arrow-down" /><span>更新</span>
        <span class="nav-badge" :class="{ hidden: stats.updates === 0 }">{{ stats.updates }}</span>
      </button>
      <button
        class="nav-item"
        :class="{ active: view === 'distribution' }"
        title="分发"
        @click="emit('navigate', 'distribution')"
      >
        <Icon name="share" /><span>分发</span>
        <span class="nav-count">{{ totalDists }}</span>
      </button>
      <button
        class="nav-item"
        :class="{ active: view === 'directories' }"
        title="目录"
        @click="emit('navigate', 'directories')"
      >
        <Icon name="folder" /><span>目录</span>
        <span class="nav-count">{{ targetsCount }}</span>
      </button>
    </div>

    <div class="nav-group">
      <div class="nav-label">仓库</div>
      <button
        class="nav-item"
        :class="{ active: view === 'sources' }"
        title="来源"
        @click="emit('navigate', 'sources')"
      >
        <Icon name="database" /><span>来源</span>
        <span class="nav-count">{{ snapshot.sources.length }}</span>
      </button>
    </div>

    <div class="sidebar-bottom">
      <div class="repo-card">
        <div class="repo-card-head">
          <span class="nav-label">当前仓库</span>
          <Icon name="git" size="sm" />
        </div>
        <div class="repo-path" :title="snapshot.vault.path">{{ repoLabel }}</div>
        <div class="repo-meta">
          <span>{{ formatBytes(stats.vaultSizeBytes) }}</span>
          <span>{{ totalDists }} 条分发</span>
        </div>
        <div class="repo-bar" aria-hidden="true">
          <span class="linked" :style="{ width: linkPct + '%' }" />
          <span class="copied" :style="{ width: copyPct + '%' }" />
        </div>
        <button class="repo-open" @click="emit('open-vault')">
          <Icon name="folder-open" size="sm" />打开文件夹
        </button>
      </div>
      <button
        class="nav-item"
        :class="{ active: view === 'settings' }"
        title="设置"
        @click="emit('navigate', 'settings')"
      >
        <Icon name="settings" /><span>设置</span>
      </button>
      <div class="version">
        Xin Skill Vault {{ snapshot.vault.appVersion || '0.1.0' }}<br />
        <span>Xin技能管理 · {{ snapshot.vault.platform || 'windows' }}</span>
      </div>
    </div>
  </aside>
</template>
