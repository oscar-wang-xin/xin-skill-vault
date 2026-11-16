<script setup lang="ts">
import { computed, ref } from 'vue'
import { createGroup, deleteGroup, renameGroup } from '../lib/appState'
import type { Snapshot } from '../lib/types'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'bulk', skillIds: string[], label: string): void
}>()

const newName = ref('')
const editingId = ref('')
const editingName = ref('')
const deletingId = ref('')

const ungroupedCount = computed(() => props.snapshot.skills.filter((s) => !s.groupId).length)

async function add() {
  const name = newName.value.trim()
  if (!name) return
  if (await createGroup(name)) newName.value = ''
}

function startRename(id: string, name: string) {
  editingId.value = id
  editingName.value = name
  deletingId.value = ''
}

async function saveRename() {
  if (!editingId.value || !editingName.value.trim()) return
  if (await renameGroup(editingId.value, editingName.value.trim())) editingId.value = ''
}

function requestDelete(id: string) {
  if (deletingId.value === id) {
    deleteGroup(id)
    deletingId.value = ''
  } else {
    deletingId.value = id
    editingId.value = ''
  }
}

function skillsInGroup(groupId: string) {
  return props.snapshot.skills.filter((skill) => skill.groupId === groupId).map((skill) => skill.id)
}

function bulkGroup(groupId: string, name: string) {
  const ids = skillsInGroup(groupId)
  if (ids.length === 0) return
  emit('bulk', ids, name)
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal group-modal" role="dialog" aria-modal="true" aria-label="管理分组">
      <header class="modal-head compact">
        <div>
          <div class="eyebrow">GROUPS · 分组</div>
          <h2>管理分组</h2>
          <p>分组用于筛选和批量分发；技能可以随时换组或回到未分组。</p>
        </div>
        <button class="modal-close" aria-label="关闭" @click="emit('close')"><Icon name="x" /></button>
      </header>

      <div class="group-body">
        <div class="group-create">
          <input v-model="newName" class="input" placeholder="新分组名称，例如：设计系统" @keydown.enter="add" />
          <button class="btn primary sm" :disabled="busy || !newName.trim()" @click="add">
            <Icon name="plus" size="sm" />新建
          </button>
        </div>

        <div class="group-list">
          <div v-for="group in snapshot.groups" :key="group.id" class="group-row">
            <span class="group-badge">{{ group.name.slice(0, 1).toUpperCase() }}</span>
            <div class="group-copy">
              <template v-if="editingId === group.id">
                <input v-model="editingName" class="input group-rename" @keydown.enter="saveRename" />
              </template>
              <template v-else>
                <div class="group-name">{{ group.name }}</div>
                <div class="group-meta">{{ group.skillCount }} 个技能</div>
              </template>
            </div>
            <div class="group-actions">
              <template v-if="editingId === group.id">
                <button class="btn ghost sm" @click="saveRename">保存</button>
                <button class="btn ghost sm" @click="editingId = ''">取消</button>
              </template>
              <template v-else>
                <button class="btn ghost sm" :disabled="group.skillCount === 0" @click="bulkGroup(group.id, group.name)">
                  <Icon name="share" size="sm" />分发整组
                </button>
                <button class="btn ghost sm" @click="startRename(group.id, group.name)">
                  <Icon name="edit" size="sm" />重命名
                </button>
                <button
                  class="btn ghost sm"
                  :class="{ danger: deletingId === group.id }"
                  @click="requestDelete(group.id)"
                >
                  <Icon name="trash" size="sm" />{{ deletingId === group.id ? '确认删除' : '删除' }}
                </button>
              </template>
            </div>
          </div>
          <div v-if="snapshot.groups.length === 0" class="inspect-placeholder small">
            <Icon name="book" size="lg" />
            <h3>还没有分组</h3>
            <p>先创建一个分组，再把技能分配到分组中。</p>
          </div>
        </div>

        <div class="group-foot-note">
          <Icon name="info" size="sm" />
          <span>未分组技能：{{ ungroupedCount }} 个。删除分组不会删除技能，只会把技能放回未分组。</span>
        </div>
      </div>

      <footer class="modal-foot">
        <div class="foot-summary"><span class="dot moss" />{{ snapshot.groups.length }} 个分组</div>
        <button class="btn ghost" @click="emit('close')">关闭</button>
      </footer>
    </div>
  </div>
</template>
