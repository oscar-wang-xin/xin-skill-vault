<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { distribute } from '../lib/appState'
import type { Agent, Distribution, Skill } from '../lib/types'
import { agentMark, agentTone } from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  skill: Skill
  agents: Agent[]
  distributions: Distribution[]
  defaultMode: string
  platform: string
  busy: boolean
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const selected = reactive(new Set<string>())
const isWindows = computed(() => (props.platform || 'windows').startsWith('windows'))
const linkMode = computed(() => (isWindows.value ? 'junction' : 'symlink'))
const linkLabel = computed(() => (isWindows.value ? 'Junction' : 'Symlink'))
const mode = ref(props.defaultMode === 'copy' ? 'copy' : linkMode.value)

const distributed = computed(() => new Set(props.distributions.map((d) => d.agentId)))
const candidates = computed(() => props.agents.filter((a) => a.enabled || distributed.value.has(a.id) || a.detected))
const pending = computed(() => candidates.value.filter((a) => !distributed.value.has(a.id)))
const canSubmit = computed(() => selected.size > 0 && !props.busy)

onMounted(() => document.addEventListener('keydown', onKey))
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))

function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}

function toggle(id: string) {
  if (distributed.value.has(id)) return
  if (selected.has(id)) selected.delete(id)
  else selected.add(id)
}

function selectAll() {
  if (selected.size === pending.value.length) selected.clear()
  else pending.value.forEach((a) => selected.add(a.id))
}

async function submit() {
  if (!canSubmit.value) return
  await distribute(props.skill.id, [...selected], mode.value)
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal distribute-modal" role="dialog" aria-modal="true" aria-label="分发技能">
      <header class="modal-head compact">
        <div>
          <div class="eyebrow">FAN-OUT · 分发</div>
          <h2>分发「{{ skill.name }}」</h2>
          <p>链接指向仓库中的同一份内容；复制会创建独立副本。</p>
        </div>
        <button class="modal-close" aria-label="关闭" @click="emit('close')"><Icon name="x" /></button>
      </header>

      <div class="distribute-body">
        <div class="option-head">
          <span class="form-label">选择 Agent / 项目</span>
          <button class="mini-link" :disabled="pending.length === 0" @click="selectAll">
            {{ selected.size === pending.length && pending.length > 0 ? '全不选' : '全选未分发' }}
          </button>
        </div>
        <div class="agent-pick-list tall">
          <label
            v-for="agent in candidates"
            :key="agent.id"
            class="agent-pick"
            :class="{ checked: selected.has(agent.id), disabled: distributed.has(agent.id) }"
          >
            <input
              type="checkbox"
              :disabled="distributed.has(agent.id)"
              :checked="selected.has(agent.id)"
              @change="toggle(agent.id)"
            />
            <span class="agent-mark" :class="`tone-${agentTone(agent.id)}`">{{ agentMark(agent.id, agent.name) }}</span>
            <span class="agent-pick-copy">
              <span>{{ agent.name }}</span>
              <small>{{ distributed.has(agent.id) ? '已分发' : agent.kind === 'project' ? '项目目录' : agent.installed + ' 个技能 · 已启用' }}</small>
            </span>
            <span v-if="agent.kind === 'project'" class="tag">项目</span>
            <span class="agent-check">
              <Icon v-if="distributed.has(agent.id)" name="check" size="sm" />
              <Icon v-else-if="selected.has(agent.id)" name="check" size="sm" />
            </span>
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
        <div class="foot-summary"><span class="dot" :class="selected.size ? 'moss' : ''" />已选择 {{ selected.size }} 个目标</div>
        <button class="btn ghost" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="!canSubmit" @click="submit">
          <Icon name="share" size="sm" />{{ busy ? '分发中…' : '建立分发' }}
        </button>
      </footer>
    </div>
  </div>
</template>
