<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { distributeMany } from '../lib/appState'
import type { Snapshot } from '../lib/types'
import { agentMark, agentTone } from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
  skillIds: string[]
  label: string
  busy: boolean
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const enabledTargets = computed(() => props.snapshot.agents.filter((a) => a.enabled))
const isWindows = computed(() => (props.snapshot.vault.platform || 'windows').startsWith('windows'))
const linkMode = computed(() => (isWindows.value ? 'junction' : 'symlink'))
const linkLabel = computed(() => (isWindows.value ? 'Junction' : 'Symlink'))
const mode = ref(props.snapshot.settings['install.defaultMode'] === 'copy' ? 'copy' : linkMode.value)
const selected = reactive(new Set<string>())

function toggle(id: string) {
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
}

function selectAll() {
  if (selected.size === enabledTargets.value.length) selected.clear()
  else enabledTargets.value.forEach((target) => selected.add(target.id))
}

function existingCount(targetId: string) {
  return props.snapshot.distributions.filter(
    (dist) => dist.agentId === targetId && props.skillIds.includes(dist.skillId),
  ).length
}

const canSubmit = computed(() => selected.size > 0 && props.skillIds.length > 0 && !props.busy)

async function submit() {
  if (!canSubmit.value) return
  const ok = await distributeMany(props.skillIds, [...selected], mode.value)
  if (ok) emit('close')
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal distribute-modal" role="dialog" aria-modal="true" aria-label="批量分发">
      <header class="modal-head compact">
        <div>
          <div class="eyebrow">BULK FAN-OUT · 批量分发</div>
          <h2>分发「{{ label }}」</h2>
          <p>{{ skillIds.length }} 个技能将分发到所选 Agent 或项目目录；已存在的分发会跳过并提示。</p>
        </div>
        <button class="modal-close" aria-label="关闭" @click="emit('close')"><Icon name="x" /></button>
      </header>

      <div class="distribute-body">
        <div class="option-head">
          <span class="form-label">分发到 Agent / 项目</span>
          <button class="mini-link" @click="selectAll">
            {{ selected.size === enabledTargets.length ? '全不选' : '全选' }}
          </button>
        </div>
        <div class="agent-pick-list tall">
          <label
            v-for="target in enabledTargets"
            :key="target.id"
            class="agent-pick"
            :class="{ checked: selected.has(target.id) }"
          >
            <input type="checkbox" :checked="selected.has(target.id)" @change="toggle(target.id)" />
            <span class="agent-mark" :class="`tone-${agentTone(target.id)}`">{{ agentMark(target.id, target.name) }}</span>
            <span class="agent-pick-copy">
              <span>{{ target.name }}</span>
              <small>
                {{ target.kind === 'project' ? '项目目录' : '已启用 Agent' }} ·
                已分发 {{ existingCount(target.id) }} / {{ skillIds.length }}
              </small>
            </span>
            <span v-if="target.kind === 'project'" class="tag">项目</span>
            <span class="agent-check"><Icon v-if="selected.has(target.id)" name="check" size="sm" /></span>
          </label>
        </div>

        <span class="form-label" style="margin-top: 16px">安装方式</span>
        <div class="mode-cards horizontal">
          <button class="mode-card" :class="{ active: mode === linkMode }" @click="mode = linkMode">
            <Icon name="link" />
            <span><strong>{{ linkLabel }}</strong><small>仓库更新即生效</small></span>
            <span class="mode-radio" />
          </button>
          <button class="mode-card" :class="{ active: mode === 'copy' }" @click="mode = 'copy'">
            <Icon name="copy" />
            <span><strong>复制</strong><small>独立副本</small></span>
            <span class="mode-radio" />
          </button>
        </div>
      </div>

      <footer class="modal-foot">
        <div class="foot-summary">
          <span class="dot" :class="selected.size ? 'moss' : ''" />
          {{ skillIds.length }} 个技能 · {{ selected.size }} 个目标
        </div>
        <button class="btn ghost" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="!canSubmit" @click="submit">
          <Icon name="share" size="sm" />{{ busy ? '分发中…' : '建立批量分发' }}
        </button>
      </footer>
    </div>
  </div>
</template>
