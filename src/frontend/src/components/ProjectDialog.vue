<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../lib/api'
import { addProject } from '../lib/appState'
import Icon from './Icon.vue'

defineProps<{ busy: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const name = ref('')
const skillsPath = ref('')
const error = ref('')

async function chooseFolder() {
  try {
    const picked = await api.pickFolder()
    if (picked) skillsPath.value = picked
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  }
}

async function submit() {
  error.value = ''
  if (!name.value.trim() || !skillsPath.value.trim()) {
    error.value = '请填写项目名称并选择技能目录'
    return
  }
  const ok = await addProject({ name: name.value.trim(), skillsPath: skillsPath.value.trim() })
  if (ok) emit('close')
}
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal project-modal" role="dialog" aria-modal="true" aria-label="添加项目目录">
      <header class="modal-head compact">
        <div>
          <div class="eyebrow">PROJECT TARGET · 项目目录</div>
          <h2>添加项目目录</h2>
          <p>把某个项目的技能目录作为分发目标，例如 <code>D:\myapp\.agents\skills</code>。</p>
        </div>
        <button class="modal-close" aria-label="关闭" @click="emit('close')"><Icon name="x" /></button>
      </header>

      <div class="project-body">
        <div class="form-block">
          <label class="form-label" for="project-name">项目名称</label>
          <input id="project-name" v-model="name" class="input" placeholder="例如：Atlas Web" />
        </div>
        <div class="form-block">
          <label class="form-label" for="project-path">技能目录</label>
          <div class="path-picker">
            <input id="project-path" v-model="skillsPath" class="input" placeholder="D:\project\.agents\skills" />
            <button class="btn" @click="chooseFolder"><Icon name="folder-open" size="sm" />选择…</button>
          </div>
          <div class="form-hint">
            选择存放 Skill 的目录，而不是项目根目录。目录不存在时会自动创建。
          </div>
        </div>
        <div v-if="error" class="inline-error"><Icon name="alert" size="sm" />{{ error }}</div>
      </div>

      <footer class="modal-foot">
        <div class="foot-summary"><span class="dot" />项目目录会出现在分发矩阵和目录管理中</div>
        <button class="btn ghost" @click="emit('close')">取消</button>
        <button class="btn primary" :disabled="busy" @click="submit">
          <Icon name="plus" size="sm" />{{ busy ? '添加中…' : '添加目录' }}
        </button>
      </footer>
    </div>
  </div>
</template>
