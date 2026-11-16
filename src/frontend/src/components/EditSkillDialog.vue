<script setup lang="ts">
import { computed, ref } from 'vue'
import { updateSkillMeta } from '../lib/appState'
import type { Skill, Snapshot } from '../lib/types'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
  skill: Skill
  busy: boolean
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const notes = ref(props.skill.notes || '')
const tagsText = ref((props.skill.tags || []).join(', '))
const groupId = ref(props.skill.groupId || '')

const tags = computed(() =>
  tagsText.value
    .split(/[,，\s]+/)
    .map((tag) => tag.trim().replace(/^#/, ''))
    .filter(Boolean)
    .slice(0, 30),
)

async function submit() {
  const ok = await updateSkillMeta({
    skillId: props.skill.id,
    notes: notes.value,
    groupId: groupId.value,
    tags: tags.value,
  })
  if (ok) emit('close')
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal edit-skill-modal" role="dialog" aria-modal="true" aria-label="编辑技能信息">
      <header class="modal-head compact">
        <div>
          <div class="eyebrow">METADATA · 技能信息</div>
          <h2>编辑「{{ skill.name }}」</h2>
          <p>备注、标签与分组保存在本地数据库中，不会修改 SKILL.md。</p>
        </div>
        <button class="modal-close" aria-label="关闭" @click="emit('close')"><Icon name="x" /></button>
      </header>

      <div class="edit-skill-body">
        <div class="form-block">
          <label class="form-label" for="skill-notes">备注</label>
          <textarea
            id="skill-notes"
            v-model="notes"
            class="textarea"
            rows="4"
            maxlength="8000"
            placeholder="例如：发布前必须跑一次 finish gate；不要用于内部后台。"
          />
          <div class="form-hint">最多 8000 字，仅本机可见。</div>
        </div>

        <div class="form-block">
          <label class="form-label" for="skill-tags">标签</label>
          <input
            id="skill-tags"
            v-model="tagsText"
            class="input"
            placeholder="ui, review, design-system"
          />
          <div class="form-hint">用逗号或空格分隔；最多 30 个标签。</div>
          <div v-if="tags.length" class="tag-preview">
            <span v-for="tag in tags" :key="tag" class="tag">#{{ tag }}</span>
          </div>
        </div>

        <div class="form-block">
          <label class="form-label" for="skill-group">分组</label>
          <select id="skill-group" v-model="groupId" class="select" style="width: 100%">
            <option value="">未分组</option>
            <option v-for="group in snapshot.groups" :key="group.id" :value="group.id">
              {{ group.name }}（{{ group.skillCount }}）
            </option>
          </select>
          <div class="form-hint">在技能库工具栏的「管理分组」中可以新建、重命名或删除分组。</div>
        </div>
      </div>

      <footer class="modal-foot">
        <div class="foot-summary"><span class="dot" :class="tags.length ? 'moss' : ''" />{{ tags.length }} 个标签</div>
        <button class="btn ghost" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="busy" @click="submit">
          <Icon name="check" size="sm" />{{ busy ? '保存中…' : '保存信息' }}
        </button>
      </footer>
    </div>
  </div>
</template>
