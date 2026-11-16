<script setup lang="ts">
import { computed, onBeforeUnmount, onErrorCaptured, onMounted, reactive, ref } from 'vue'
import IconSprite from './components/IconSprite.vue'
import Icon from './components/Icon.vue'
import TitleBar from './components/TitleBar.vue'
import SideBar from './components/SideBar.vue'
import LibraryView from './components/LibraryView.vue'
import SkillInspector from './components/SkillInspector.vue'
import UpdatesView from './components/UpdatesView.vue'
import DistributionView from './components/DistributionView.vue'
import DirectoryView from './components/DirectoryView.vue'
import SourcesView from './components/SourcesView.vue'
import SettingsView from './components/SettingsView.vue'
import AddSkillModal from './components/AddSkillModal.vue'
import DistributeDialog from './components/DistributeDialog.vue'
import CommandPalette from './components/CommandPalette.vue'
import Toasts from './components/Toasts.vue'
import {
  applySkin,
  cancelClose,
  checkUpdates,
  currentSkin,
  initApp,
  navigate,
  openAdd,
  openAgentFolder,
  openSkillFolder,
  openVault,
  refresh,
  removeSkill,
  repair,
  resolveClose,
  selectSkill,
  selectedSkill,
  setAgentEnabled,
  setDistribution,
  setLayout,
  setSkin,
  state,
  type View,
} from './lib/appState'
import { api } from './lib/api'
import type { SkinID } from './lib/types'
import { relativeTime, timeShort } from './lib/format'

const confirmState = reactive({
  open: false,
  title: '',
  message: '',
  impact: '',
  label: '确认',
  danger: true,
  action: async () => {},
})

const renderError = ref('')
const rememberClose = ref(true)

onErrorCaptured((err) => {
  const e = err as Error
  renderError.value = `${e?.message || String(err)}\n\n${e?.stack || ''}`
  return false
})

const syncLabel = computed(() => {
  const at = state.snapshot.vault.lastScanAt
  if (!at) return '尚未扫描'
  return `同步于 ${timeShort(at)} · ${relativeTime(at)}`
})

const inspectorDistributions = computed(() => {
  if (!selectedSkill.value) return []
  return state.snapshot.distributions.filter((d) => d.skillId === selectedSkill.value!.id)
})

async function removeSelectedSkill() {
  if (!selectedSkill.value) return
  const skill = selectedSkill.value
  confirmState.open = true
  confirmState.title = `移除「${skill.name}」？`
  confirmState.message = '仓库中的技能目录会移入 .trash 回收区，所有 Junction 与复制副本会一并移除；原始导入来源不受影响。'
  confirmState.impact = `${skill.agents.length} 条分发 · ${skill.fileCount} 个文件 · ${skill.vaultPath}`
  confirmState.label = '移除技能'
  confirmState.danger = true
  confirmState.action = async () => {
    confirmState.open = false
    await removeSkill(skill.id)
  }
}

function removeDistributionFor(agentId: string) {
  if (!selectedSkill.value) return
  const skill = selectedSkill.value
  const agent = state.snapshot.agents.find((a) => a.id === agentId)
  const dist = state.snapshot.distributions.find((d) => d.skillId === skill.id && d.agentId === agentId)
  confirmState.open = true
  confirmState.title = `移除「${skill.name}」在 ${agent?.name || agentId} 的分发？`
  confirmState.message = '只删除该 Agent 目录下的链接或副本，仓库中的技能内容不会改变。'
  confirmState.impact = dist?.linkPath || ''
  confirmState.label = '移除分发'
  confirmState.danger = true
  confirmState.action = async () => {
    confirmState.open = false
    await setDistribution(skill.id, agentId, false)
  }
}

function onPaletteAction(action: string) {
  state.paletteOpen = false
  switch (action) {
    case 'add': openAdd('local'); break
    case 'refresh': refresh(); break
    case 'check': checkUpdates(); break
    case 'repair': repair(); break
    case 'open-repo': openVault(); break
  }
}

function onSetSkin(id: SkinID) {
  setSkin(id)
}

function reload() {
  window.location.reload()
}

function selectFromOther(id: string) {
  navigate('library')
  selectSkill(id)
}

function onKeydown(event: KeyboardEvent) {
  const meta = event.ctrlKey || event.metaKey
  if (meta && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    state.paletteOpen = !state.paletteOpen
    return
  }
  if (meta && event.key.toLowerCase() === 'n') {
    event.preventDefault()
    openAdd('local')
    return
  }
  if (event.key === 'F5') {
    event.preventDefault()
    refresh()
    return
  }
  if (event.key === 'Escape') {
    if (state.paletteOpen) state.paletteOpen = false
    else if (state.addOpen) state.addOpen = false
    else if (state.distributeOpen) state.distributeOpen = false
    else if (confirmState.open) confirmState.open = false
    else if (state.selectedId) selectSkill(null)
  }
}

async function boot() {
  await initApp()
  if (!state.ready) return
  const qp = new URLSearchParams(window.location.search)
  const view = qp.get('view')
  if (view && ['library', 'updates', 'distribution', 'directories', 'sources', 'settings'].includes(view)) {
    state.view = view as View
  }
  const skin = qp.get('skin')
  if (skin) {
    state.snapshot.settings['ui.skin'] = skin
    applySkin(skin, false)
  }
  const skill = qp.get('skill')
  if (skill && state.snapshot.skills.some((s) => s.id === skill)) {
    state.view = 'library'
    state.selectedId = skill
  }
  if (qp.get('add')) {
    const addTab = qp.get('tab')
    openAdd(addTab === 'existing' || addTab === 'network' ? addTab : 'local')
  }
  if (qp.get('palette')) state.paletteOpen = true
  if (qp.get('distribute') && state.snapshot.skills.length) {
    state.selectedId = state.snapshot.skills[0].id
    state.distributeOpen = true
  }
}

onMounted(() => {
  boot()
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <IconSprite />
  <div v-if="state.loading" class="boot">
    <div class="boot-mark">
      <svg viewBox="0 0 24 24"><use href="#i-logo" /></svg>
    </div>
    <div class="boot-word">XIN SKILL VAULT</div>
    <div class="boot-line"><span class="boot-pulse" />正在打开技能台账…</div>
    <div class="boot-note">本地索引 · 本地优先 · 无遥测</div>
  </div>

  <div v-else-if="state.error" class="boot boot-error">
    <div class="boot-mark"><Icon name="alert" /></div>
    <div class="boot-word">无法打开技能仓库</div>
    <div class="boot-line">{{ state.error }}</div>
    <button class="btn primary" style="margin-top: 16px" @click="reload">重试</button>
  </div>

  <div v-else class="app" :class="{ 'with-inspector': !!selectedSkill && state.view === 'library' }">
    <TitleBar
      :vault-path="state.snapshot.vault.path"
      :skin="currentSkin"
      :sync-label="syncLabel"
      @open-palette="state.paletteOpen = true"
      @open-repo="openVault"
      @set-skin="onSetSkin"
      @open-settings="navigate('settings')"
    />
    <SideBar :snapshot="state.snapshot" :view="state.view" @navigate="navigate" @open-vault="openVault" />

    <main class="main">
      <div class="view-scroll">
        <div v-if="renderError" class="render-error">
          <h2>界面渲染失败</h2>
          <p>这是一个可复现的前端错误。请把下面的信息发回给开发会话。</p>
          <pre>{{ renderError }}</pre>
          <button class="btn" @click="renderError = ''">重试</button>
        </div>
        <template v-else>
          <LibraryView
            v-if="state.view === 'library'"
            :snapshot="state.snapshot"
            :selected-id="state.selectedId"
            :layout="state.layout"
            :query="state.query"
            :filter="state.filter"
            :busy="state.busy"
            @select="selectSkill"
            @add="openAdd"
            @refresh="refresh"
            @set-layout="setLayout"
            @update:query="state.query = $event"
            @update:filter="state.filter = $event"
            @open-folder="openSkillFolder"
          />
          <UpdatesView
            v-else-if="state.view === 'updates'"
            :snapshot="state.snapshot"
            :busy="state.busy"
            @check="checkUpdates"
            @select="selectFromOther"
            @navigate-library="navigate('library')"
          />
          <DistributionView
            v-else-if="state.view === 'distribution'"
            :snapshot="state.snapshot"
            :busy="state.busy"
            @toggle-distribution="setDistribution"
            @set-agent-enabled="setAgentEnabled"
            @open-agent-folder="openAgentFolder"
            @select="selectFromOther"
            @add="openAdd('local')"
          />
          <DirectoryView
            v-else-if="state.view === 'directories'"
            :snapshot="state.snapshot"
            :busy="state.busy"
            @select-skill="selectFromOther"
          />
          <SourcesView
            v-else-if="state.view === 'sources'"
            :snapshot="state.snapshot"
            @select="selectFromOther"
          />
          <SettingsView
            v-else-if="state.view === 'settings'"
            :snapshot="state.snapshot"
            :busy="state.busy"
            @open-vault="openVault"
            @refresh="refresh"
            @repair="repair"
            @check-updates="checkUpdates"
          />
        </template>
      </div>

      <SkillInspector
        v-if="selectedSkill && state.view === 'library'"
        :snapshot="state.snapshot"
        :skill="selectedSkill"
        :agents="state.snapshot.agents"
        :distributions="inspectorDistributions"
        :busy="state.busy"
        @close="selectSkill(null)"
        @distribute="state.distributeOpen = true"
        @remove="removeSelectedSkill"
        @open-folder="openSkillFolder(selectedSkill.id)"
        @remove-distribution="removeDistributionFor"
      />
    </main>

    <AddSkillModal
      v-if="state.addOpen"
      :snapshot="state.snapshot"
      :initial-tab="state.addTab"
      :busy="state.busy"
      @close="state.addOpen = false"
    />

    <DistributeDialog
      v-if="state.distributeOpen && selectedSkill"
      :skill="selectedSkill"
      :agents="state.snapshot.agents"
      :distributions="inspectorDistributions"
      :default-mode="state.snapshot.settings['install.defaultMode'] || 'junction'"
      :platform="state.snapshot.vault.platform"
      :busy="state.busy"
      @close="state.distributeOpen = false"
    />

    <CommandPalette
      v-if="state.paletteOpen"
      :snapshot="state.snapshot"
      @close="state.paletteOpen = false"
      @select-skill="selectFromOther"
      @action="onPaletteAction"
      @navigate="navigate"
      @set-skin="onSetSkin"
    />

    <div v-if="confirmState.open" class="modal-overlay" @click.self="confirmState.open = false">
      <div class="modal confirm-modal" role="dialog" aria-modal="true" :aria-label="confirmState.title">
        <header class="modal-head compact">
          <div>
            <div class="eyebrow">CONFIRM · 确认</div>
            <h2>{{ confirmState.title }}</h2>
          </div>
          <button class="modal-close" aria-label="关闭" @click="confirmState.open = false"><Icon name="x" /></button>
        </header>
        <div class="confirm-body">
          <p>{{ confirmState.message }}</p>
          <div v-if="confirmState.impact" class="confirm-impact">{{ confirmState.impact }}</div>
        </div>
        <footer class="modal-foot">
          <button class="btn ghost" @click="confirmState.open = false">取消</button>
          <button class="btn" :class="confirmState.danger ? 'danger-solid' : 'primary'" @click="confirmState.action()">
            {{ confirmState.label }}
          </button>
        </footer>
      </div>
    </div>

    <div v-if="state.closePromptOpen" class="modal-overlay" @click.self="cancelClose">
      <div class="modal close-modal" role="dialog" aria-modal="true" aria-label="关闭 Xin Skill Vault">
        <header class="modal-head compact">
          <div>
            <div class="eyebrow">CLOSE · 关闭</div>
            <h2>关闭 Xin Skill Vault？</h2>
            <p>第一次关闭需要选择行为；之后可以在设置 → 窗口中修改。</p>
          </div>
          <button class="modal-close" aria-label="取消关闭" @click="cancelClose"><Icon name="x" /></button>
        </header>
        <div class="confirm-body">
          <p>选择「最小化到托盘」后，应用会继续在后台运行，可从任务栏通知区的图标重新打开。</p>
          <label class="remember-row">
            <input v-model="rememberClose" type="checkbox" />
            <span>记住我的选择，不再询问</span>
          </label>
        </div>
        <footer class="modal-foot">
          <button class="btn ghost" @click="cancelClose">取消</button>
          <button class="btn danger" @click="resolveClose('quit', rememberClose)">
            <Icon name="x" size="sm" />直接退出
          </button>
          <button class="btn primary" @click="resolveClose('tray', rememberClose)">
            <Icon name="minus" size="sm" />最小化到托盘
          </button>
        </footer>
      </div>
    </div>

    <Toasts />
  </div>
</template>
