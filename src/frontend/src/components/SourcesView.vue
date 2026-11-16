<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import type { Snapshot, Source } from '../lib/types'
import { addSource, pushToast, removeSource } from '../lib/appState'
import { relativeTime, sourceKindLabel } from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
}>()

const emit = defineEmits<{
  (e: 'select', skillId: string): void
}>()

const builtinCount = computed(() => props.snapshot.sources.filter((s) => s.builtin).length)
const totalSkills = computed(() => props.snapshot.stats.skills)
const addOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const removeTarget = ref<Source | null>(null)
const form = reactive({ name: '', kind: 'git', url: '' })

const iconFor = (source: Source) => {
  if (source.kind === 'git') return 'git'
  if (source.kind === 'registry') return 'database'
  return 'package'
}

async function copy(source: Source) {
  const value = source.url || props.snapshot.vault.path
  try {
    await navigator.clipboard.writeText(value)
    pushToast('success', '地址已复制', value)
  } catch {
    pushToast('error', '无法访问剪贴板')
  }
}

function selectSource(sourceId: string) {
  const first = props.snapshot.skills.find((s) => s.sourceId === sourceId)
  if (first) emit('select', first.id)
}

async function submitSource() {
  formError.value = ''
  if (!form.name.trim()) {
    formError.value = '请填写来源名称'
    return
  }
  if (!form.url.trim()) {
    formError.value = '请填写来源地址或本地路径'
    return
  }
  saving.value = true
  const ok = await addSource({ name: form.name.trim(), kind: form.kind, url: form.url.trim() })
  saving.value = false
  if (ok) {
    addOpen.value = false
    form.name = ''
    form.kind = 'git'
    form.url = ''
  }
}

async function confirmRemove() {
  if (!removeTarget.value) return
  const ok = await removeSource(removeTarget.value.id)
  if (ok) removeTarget.value = null
}
</script>

<template>
  <div class="page">
    <header class="page-head">
      <div>
        <div class="eyebrow">PROVENANCE · 来源</div>
        <h1>来源</h1>
        <p class="page-sub">每个技能都保留来源与提交线索。v1 的记录以本地仓库为准，来源缓存将在 M6 接入。</p>
      </div>
      <div class="page-actions">
        <span class="count-chip">{{ snapshot.sources.length }} 个来源</span>
        <button class="btn primary" @click="addOpen = true"><Icon name="plus" />添加来源</button>
      </div>
    </header>

    <section class="ledger">
      <div class="ledger-item">
        <div class="ledger-value">{{ snapshot.sources.length }}</div>
        <div class="ledger-label">来源总数</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ builtinCount }}</div>
        <div class="ledger-label">内置来源</div>
      </div>
      <div class="ledger-item">
        <div class="ledger-value">{{ totalSkills }}</div>
        <div class="ledger-label">已收录技能</div>
      </div>
      <div class="ledger-note">
        技能内容始终复制进本地仓库<br />
        Git / Registry 来源不会直接分发
      </div>
    </section>

    <div class="source-grid">
      <article v-for="source in snapshot.sources" :key="source.id" class="source-card">
        <div class="source-head">
          <span class="source-icon"><Icon :name="iconFor(source)" /></span>
          <div class="source-title">
            <h3>{{ source.name }}</h3>
            <span class="source-kind">{{ sourceKindLabel(source.kind) }}</span>
          </div>
          <span v-if="source.builtin" class="tag">内置</span>
          <span v-else class="tag">自定义</span>
        </div>
        <div class="source-url">{{ source.url || (source.id === 'network' ? '由网络导入自动记录，地址见各技能来源' : snapshot.vault.path) }}</div>
        <div class="source-stats">
          <div>
            <strong>{{ source.skillCount }}</strong>
            <span>技能</span>
          </div>
          <div>
            <strong>{{ source.syncedAt ? relativeTime(source.syncedAt) : '—' }}</strong>
            <span>最后同步</span>
          </div>
        </div>
        <div class="source-actions">
          <button v-if="source.skillCount" class="btn ghost sm" @click="selectSource(source.id)">
            <Icon name="book" size="sm" />查看技能
          </button>
          <button v-if="source.url || source.kind === 'local'" class="btn ghost sm" @click="copy(source)">
            <Icon name="copy" size="sm" />复制地址
          </button>
          <button
            v-if="!source.builtin"
            class="btn ghost sm"
            :disabled="source.skillCount > 0"
            :title="source.skillCount > 0 ? '该来源下还有技能，不能移除' : '移除自定义来源'"
            @click="removeTarget = source"
          >
            <Icon name="trash" size="sm" />移除
          </button>
        </div>
      </article>

      <article class="source-card source-note-card">
        <div class="source-head">
          <span class="source-icon"><Icon name="info" /></span>
          <div class="source-title">
            <h3>来源如何工作</h3>
            <span class="source-kind">v1 行为说明</span>
          </div>
        </div>
        <ul class="source-rules">
          <li>导入时复制到 <code>~/.skill-vault/skills</code>，来源只作为归属记录。</li>
          <li>本地目录与现有 Agent 安装是 v1 的两个真实入口。</li>
          <li>Git / ClawHub 来源将在 M6 接入缓存、提交 diff 与更新提示。</li>
          <li>分发只指向仓库内容，不会直接链接到远程来源。</li>
        </ul>
      </article>
    </div>

    <section class="source-table-section">
      <div class="section-head">
        <span class="section-title">来源归属</span>
        <span class="section-count">SOURCES · 来源</span>
      </div>
      <div v-if="snapshot.skills.length === 0" class="inline-empty">
        <Icon name="database" size="sm" />
        <span>录入了第一个技能后，这里会显示来源与技能数量的对应关系。</span>
      </div>
      <div v-else class="source-bars">
        <div v-for="source in snapshot.sources" :key="source.id" class="source-bar-row">
          <span class="source-bar-name">{{ source.name }}</span>
          <span class="source-bar-track">
            <span
              class="source-bar-fill"
              :style="{ width: Math.max(3, (source.skillCount / Math.max(1, totalSkills)) * 100) + '%' }"
            />
          </span>
          <span class="source-bar-value">{{ source.skillCount }}</span>
        </div>
      </div>
    </section>

    <div v-if="addOpen" class="modal-overlay" @click.self="addOpen = false">
      <div class="modal source-modal" role="dialog" aria-modal="true" aria-label="添加来源">
        <header class="modal-head compact">
          <div>
            <div class="eyebrow">SOURCE · 来源</div>
            <h2>添加来源</h2>
            <p>手动来源只记录归属与地址，不会自动下载；技能仍通过导入进入仓库。</p>
          </div>
          <button class="modal-close" aria-label="关闭" @click="addOpen = false"><Icon name="x" /></button>
        </header>
        <div class="source-body">
          <div class="form-block">
            <label class="form-label" for="source-name">显示名称</label>
            <input id="source-name" v-model="form.name" class="input" placeholder="例如：团队设计规范" />
          </div>
          <div class="form-block">
            <label class="form-label" for="source-kind">类型</label>
            <select id="source-kind" v-model="form.kind" class="select" style="width: 100%">
              <option value="git">Git 仓库</option>
              <option value="registry">注册表</option>
              <option value="url">网络地址</option>
              <option value="local">本地目录</option>
            </select>
          </div>
          <div class="form-block">
            <label class="form-label" for="source-url">地址 / 路径</label>
            <input id="source-url" v-model="form.url" class="input" placeholder="github.com/owner/repo 或本地路径" />
          </div>
          <div v-if="formError" class="inline-error"><Icon name="alert" size="sm" />{{ formError }}</div>
        </div>
        <footer class="modal-foot">
          <div class="foot-summary"><span class="dot" />来源会出现在筛选和来源图中</div>
          <button class="btn ghost" @click="addOpen = false">取消</button>
          <button class="btn primary" :disabled="saving" @click="submitSource">
            <Icon name="plus" size="sm" />{{ saving ? '添加中…' : '添加来源' }}
          </button>
        </footer>
      </div>
    </div>

    <div v-if="removeTarget" class="modal-overlay" @click.self="removeTarget = null">
      <div class="modal confirm-modal" role="dialog" aria-modal="true" aria-label="移除来源">
        <header class="modal-head compact">
          <div><div class="eyebrow">CONFIRM · 确认</div><h2>移除来源「{{ removeTarget.name }}」？</h2></div>
          <button class="modal-close" aria-label="关闭" @click="removeTarget = null"><Icon name="x" /></button>
        </header>
        <div class="confirm-body">
          <p>只删除这条来源记录，不会删除技能内容。</p>
          <div class="confirm-impact">{{ removeTarget.url || snapshot.vault.path }}</div>
        </div>
        <footer class="modal-foot">
          <button class="btn ghost" @click="removeTarget = null">取消</button>
          <button class="btn danger-solid" :disabled="saving" @click="confirmRemove">移除来源</button>
        </footer>
      </div>
    </div>
  </div>
</template>
