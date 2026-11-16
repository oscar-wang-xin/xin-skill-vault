<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Snapshot, SkinID } from '../lib/types'
import { SKINS } from '../lib/types'
import type { View } from '../lib/appState'
import {
  formatBytes,
  skillMark,
  skillTone,
} from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'select-skill', id: string): void
  (e: 'action', action: string): void
  (e: 'navigate', view: View): void
  (e: 'set-skin', id: SkinID): void
}>()

interface Item {
  id: string
  group: string
  label: string
  hint?: string
  icon: string
  tone?: string
  mark?: string
  run: () => void
}

const query = ref('')
const index = ref(0)
const input = ref<HTMLInputElement | null>(null)

const actions: Item[] = [
  { id: 'add', group: '操作', label: '添加技能…', hint: 'Ctrl N', icon: 'plus', run: () => emit('action', 'add') },
  { id: 'refresh', group: '操作', label: '重新扫描仓库索引', hint: 'F5', icon: 'refresh', run: () => emit('action', 'refresh') },
  { id: 'check', group: '操作', label: '检查所有来源的更新', icon: 'arrow-down', run: () => emit('action', 'check') },
  { id: 'repair', group: '操作', label: '运行分发健康检查', icon: 'shield', run: () => emit('action', 'repair') },
  { id: 'repo', group: '操作', label: '在资源管理器中打开仓库', icon: 'folder-open', run: () => emit('action', 'open-repo') },
  { id: 'view-library', group: '前往', label: '技能库', icon: 'book', run: () => emit('navigate', 'library') },
  { id: 'view-updates', group: '前往', label: '更新', icon: 'arrow-down', run: () => emit('navigate', 'updates') },
  { id: 'view-distribution', group: '前往', label: '分发', icon: 'share', run: () => emit('navigate', 'distribution') },
  { id: 'view-directories', group: '前往', label: '目录管理', icon: 'folder', run: () => emit('navigate', 'directories') },
  { id: 'view-sources', group: '前往', label: '来源', icon: 'database', run: () => emit('navigate', 'sources') },
  { id: 'view-settings', group: '前往', label: '设置', icon: 'settings', run: () => emit('navigate', 'settings') },
  ...SKINS.map((skin) => ({
    id: `skin-${skin.id}`,
    group: '换肤',
    label: `切换到 ${skin.name}`,
    hint: skin.en,
    icon: 'palette',
    run: () => emit('set-skin', skin.id),
  })),
]

const items = computed<Item[]>(() => {
  const q = query.value.trim().toLowerCase()
  const skills: Item[] = props.snapshot.skills
    .filter((skill) => {
      if (!q) return true
      return `${skill.name} ${skill.description} ${skill.tags.join(' ')} ${skill.sourceName}`.toLowerCase().includes(q)
    })
    .slice(0, 7)
    .map((skill) => ({
      id: `skill-${skill.id}`,
      group: '技能',
      label: skill.name,
      hint: `${formatBytes(skill.sizeBytes)} · ${skill.agents.length} 条分发`,
      icon: 'file',
      tone: skillTone(skill.name),
      mark: skillMark(skill.name),
      run: () => emit('select-skill', skill.id),
    }))
  const acts = actions.filter((item) => {
    if (!q) return true
    return `${item.label} ${item.group}`.toLowerCase().includes(q)
  })
  return [...acts, ...skills]
})

const grouped = computed(() => {
  const groups: { name: string; items: Item[] }[] = []
  for (const item of items.value) {
    let group = groups.find((g) => g.name === item.group)
    if (!group) {
      group = { name: item.group, items: [] }
      groups.push(group)
    }
    group.items.push(item)
  }
  return groups
})

watch(query, () => {
  index.value = 0
})

function run(item: Item) {
  item.run()
  emit('close')
}

function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    emit('close')
    return
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    index.value = Math.min(index.value + 1, items.value.length - 1)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    index.value = Math.max(index.value - 1, 0)
  } else if (event.key === 'Enter') {
    event.preventDefault()
    const item = items.value[index.value]
    if (item) run(item)
  }
}

const globalIndex = (item: Item) => items.value.findIndex((i) => i.id === item.id)

onMounted(async () => {
  document.addEventListener('keydown', onKey)
  await nextTick()
  input.value?.focus()
})

onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="palette-overlay" @click.self="emit('close')">
    <div class="palette" role="dialog" aria-modal="true" aria-label="命令面板">
      <div class="palette-input-row">
        <Icon name="search" />
        <input
          ref="input"
          v-model="query"
          placeholder="搜索技能，或输入命令…"
          aria-label="搜索技能或命令"
        />
        <kbd>ESC</kbd>
      </div>
      <div class="palette-results">
        <div v-for="group in grouped" :key="group.name" class="palette-group">
          <div class="palette-group-label">{{ group.name }}</div>
          <button
            v-for="item in group.items"
            :key="item.id"
            class="palette-item"
            :class="{ active: globalIndex(item) === index }"
            @mouseenter="index = globalIndex(item)"
            @click="run(item)"
          >
            <span v-if="item.mark" class="tile" :class="`tone-${item.tone}`">{{ item.mark }}</span>
            <span v-else class="palette-ic"><Icon :name="item.icon" size="sm" /></span>
            <span class="palette-label">{{ item.label }}</span>
            <span v-if="item.hint" class="palette-hint">{{ item.hint }}</span>
            <Icon name="chevron-right" size="sm" class="palette-chevron" />
          </button>
        </div>
        <div v-if="items.length === 0" class="palette-empty">
          <Icon name="search" size="lg" />
          <span>没有匹配的技能或命令</span>
        </div>
      </div>
      <div class="palette-foot">
        <span><kbd>↑</kbd><kbd>↓</kbd> 选择</span>
        <span><kbd>↵</kbd> 执行</span>
        <span class="palette-foot-note">技能、Agent、操作统一入口</span>
      </div>
    </div>
  </div>
</template>
