<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Snapshot } from '../lib/types'
import {
  FONT_FAMILIES,
  FONT_SCALES,
  pushToast,
  saveSetting,
  setLayout,
  setSkin,
  setSnapshot,
  type Layout,
} from '../lib/appState'
import { api } from '../lib/api'
import { SKINS, type SkinID } from '../lib/types'
import { formatBytes, relativeTime, timeShort } from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  snapshot: Snapshot
  busy: boolean
}>()

const emit = defineEmits<{
  (e: 'open-vault'): void
  (e: 'refresh'): void
  (e: 'repair'): void
  (e: 'check-updates'): void
}>()

const saved = ref(false)
let savedTimer = 0

const settings = computed(() => props.snapshot.settings)
const skin = computed<SkinID>(() => (settings.value['ui.skin'] as SkinID) || 'cold-white')
const layout = computed<Layout>(() => (settings.value['ui.layout'] === 'cards' ? 'cards' : 'table'))
const isWindows = computed(() => (props.snapshot.vault.platform || 'windows').startsWith('windows'))
const linkMode = computed(() => (isWindows.value ? 'junction' : 'symlink'))
const linkLabel = computed(() => (isWindows.value ? 'Junction' : 'Symlink'))
const currentMode = computed(() =>
  settings.value['install.defaultMode'] === 'copy' ? 'copy' : linkMode.value,
)
const healthy = computed(() => props.snapshot.stats.broken === 0)
const lastCheck = computed(() => settings.value['updates.lastCheckAt'] || '')
const fontFamily = computed(() => settings.value['ui.fontFamily'] || 'system')
const vaultDialogOpen = ref(false)
const vaultNewPath = ref('')
const vaultChanging = ref(false)
const vaultError = ref('')
const fontScale = computed(() => {
  const value = Number.parseFloat(settings.value['ui.fontScale'] || '1.12')
  return Number.isFinite(value) ? value : 1.12
})
const fontScalePercent = computed(() => Math.round(fontScale.value * 100))
const closeAction = computed(() => settings.value['window.closeAction'] || 'ask')

function pulse() {
  saved.value = true
  window.clearTimeout(savedTimer)
  savedTimer = window.setTimeout(() => (saved.value = false), 1600)
}

async function chooseSkin(id: SkinID) {
  await setSkin(id)
  pulse()
}

async function chooseMode(mode: string) {
  if (await saveSetting('install.defaultMode', mode)) pulse()
}

async function chooseLayout(value: Layout) {
  await setLayout(value)
  pulse()
}

async function chooseFontFamily(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  if (await saveSetting('ui.fontFamily', value)) pulse()
}

async function chooseFontScale(value: number) {
  if (await saveSetting('ui.fontScale', String(value))) pulse()
}

async function chooseCloseAction(value: string) {
  if (await saveSetting('window.closeAction', value)) pulse()
}

function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

async function chooseVaultFolder() {
  vaultError.value = ''
  try {
    const path = await api.pickVaultFolder()
    if (!path) return
    vaultNewPath.value = path
    vaultDialogOpen.value = true
  } catch (err) {
    pushToast('error', '无法选择目录', errorText(err))
  }
}

async function changeVault(mode: 'move' | 'copy') {
  if (vaultChanging.value) return
  vaultChanging.value = true
  vaultError.value = ''
  try {
    const result = await api.changeVaultPath(vaultNewPath.value, mode)
    setSnapshot(result.snapshot)
    vaultDialogOpen.value = false
    pulse()
    pushToast('success', '仓库位置已切换', result.message)
    if (result.warnings.length > 0) {
      pushToast('warning', `${result.warnings.length} 条提示`, result.warnings[0])
    }
  } catch (err) {
    vaultError.value = errorText(err)
  } finally {
    vaultChanging.value = false
  }
}

function closeVaultDialog() {
  if (vaultChanging.value) return
  vaultDialogOpen.value = false
  vaultError.value = ''
}
</script>

<template>
  <div class="page settings-page">
    <header class="page-head">
      <div>
        <div class="eyebrow">PREFERENCES · 偏好</div>
        <h1>设置</h1>
        <p class="page-sub">仓库、安装方式与外观。所有偏好写入本地数据库，不离开这台机器。</p>
      </div>
      <div class="page-actions">
        <span class="saved-hint" :class="{ show: saved }"><Icon name="check" size="sm" />已保存</span>
      </div>
    </header>

    <section class="set-group">
      <h2>仓库</h2>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">中央仓库</div>
          <div class="set-desc">所有技能的唯一事实源。导入会复制到这里，分发只从这里读取。</div>
          <div class="set-path" :title="snapshot.vault.path">{{ snapshot.vault.path }}</div>
        </div>
        <div class="set-control">
          <button class="btn sm" @click="emit('open-vault')"><Icon name="folder-open" size="sm" />打开目录</button>
          <button class="btn sm" @click="chooseVaultFolder"><Icon name="arrow-up-right" size="sm" />更改位置</button>
        </div>
      </div>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">仓库健康</div>
          <div class="set-desc">
            {{ healthy ? '没有发现断链或副本漂移。' : `${snapshot.stats.broken} 条分发需要修复。` }}
            最近扫描 {{ snapshot.vault.lastScanAt ? relativeTime(snapshot.vault.lastScanAt) : '—' }}。
          </div>
        </div>
        <div class="set-control">
          <span class="health-pill" :class="healthy ? 'ok' : 'bad'">
            <span class="dot" :class="healthy ? 'moss' : 'clay'" />{{ healthy ? '健康' : '需修复' }}
          </span>
          <button class="btn sm" :disabled="busy" @click="emit('refresh')"><Icon name="refresh" size="sm" :class="{ spin: busy }" />重新扫描</button>
          <button class="btn sm" :disabled="busy" @click="emit('repair')"><Icon name="shield" size="sm" />健康检查</button>
        </div>
      </div>
    </section>

    <section class="set-group">
      <h2>安装与分发</h2>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">默认安装方式</div>
          <div class="set-desc">
            {{ isWindows ? 'Junction 不需要管理员权限，不占用额外空间；复制创建独立副本。' : 'Symlink 不占空间；复制创建独立副本。' }}
          </div>
        </div>
        <div class="set-control">
          <div class="seg mode-seg">
            <button :class="{ active: currentMode === linkMode }" @click="chooseMode(linkMode)">
              <Icon name="link" size="sm" />{{ linkLabel }}
            </button>
            <button :class="{ active: currentMode === 'copy' }" @click="chooseMode('copy')">
              <Icon name="copy" size="sm" />复制
            </button>
          </div>
        </div>
      </div>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">更新检查</div>
          <div class="set-desc">最近检查 {{ lastCheck ? relativeTime(lastCheck) + '（' + timeShort(lastCheck) + '）' : '尚未检查' }}。v1 只提示更新，不自动覆盖。</div>
        </div>
        <div class="set-control">
          <button class="btn sm" :disabled="busy" @click="emit('check-updates')">
            <Icon name="refresh" size="sm" :class="{ spin: busy }" />立即检查
          </button>
        </div>
      </div>
    </section>

    <section class="set-group">
      <h2>外观</h2>
      <div class="set-row skin-row">
        <div class="set-copy">
          <div class="set-label">皮肤</div>
          <div class="set-desc">
            皮肤只改变颜色 token，不改变布局、字号或交互。默认「冷白机甲」；「跟随系统」在纸张与夜读之间切换。
          </div>
        </div>
        <div class="set-control">
          <div class="skin-grid">
            <button
              v-for="item in SKINS"
              :key="item.id"
              class="skin-card"
              :class="{ active: skin === item.id }"
              :aria-pressed="skin === item.id"
              @click="chooseSkin(item.id)"
            >
              <span class="skin-strip">
                <i v-for="color in item.colors" :key="color" :style="{ background: color }" />
              </span>
              <span class="skin-card-name">
                <span>{{ item.name }}</span>
                <span v-if="skin === item.id" class="skin-dot" />
              </span>
              <span class="skin-card-en">{{ item.en }}</span>
              <span class="skin-card-hint">{{ item.hint }}</span>
            </button>
          </div>
        </div>
      </div>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">界面字体</div>
          <div class="set-desc">只改变界面文本与标题；路径、哈希与代码仍使用等宽字体。</div>
        </div>
        <div class="set-control">
          <select class="select font-select" :value="fontFamily" aria-label="界面字体" @change="chooseFontFamily">
            <option v-for="font in FONT_FAMILIES" :key="font.id" :value="font.id">{{ font.name }}</option>
          </select>
        </div>
      </div>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">界面字号 · {{ fontScalePercent }}%</div>
          <div class="set-desc">高分辨率屏幕可调到「大」或「特大」；只缩放字体，不改变面板与表格布局。</div>
          <div class="font-preview">
            <span class="font-preview-title">技能库 · Agent Skill 台账</span>
            <span class="font-preview-sub">Segoe UI / Microsoft YaHei · 路径与哈希保持等宽</span>
            <span class="font-preview-mono">C:\Users\you\.skill-vault\skills\anti-ui-slop</span>
          </div>
        </div>
        <div class="set-control">
          <div class="seg font-scale-seg">
            <button
              v-for="option in FONT_SCALES"
              :key="option.value"
              :class="{ active: Math.abs(fontScale - option.value) < 0.001 }"
              @click="chooseFontScale(option.value)"
            >{{ option.label }}</button>
          </div>
        </div>
      </div>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">默认布局</div>
          <div class="set-desc">技能库默认使用台账表格，或更宽松的卡片网格。</div>
        </div>
        <div class="set-control">
          <div class="seg">
            <button :class="{ active: layout === 'table' }" @click="chooseLayout('table')"><Icon name="list" size="sm" />表格</button>
            <button :class="{ active: layout === 'cards' }" @click="chooseLayout('cards')"><Icon name="grid" size="sm" />卡片</button>
          </div>
        </div>
      </div>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">界面语言</div>
          <div class="set-desc">技术名词保留英文：Skill、Agent、Junction。English 将在后续版本提供。</div>
        </div>
        <div class="set-control"><span class="set-value">简体中文 · zh-CN</span></div>
      </div>
    </section>

    <section class="set-group">
      <h2>窗口</h2>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">关闭主窗口时</div>
          <div class="set-desc">
            首次关闭会询问。选择「最小化到托盘」后，可从任务栏通知区域的小图标重新打开；选择「直接退出」则结束进程。
          </div>
        </div>
        <div class="set-control">
          <div class="seg close-seg">
            <button :class="{ active: closeAction === 'ask' }" @click="chooseCloseAction('ask')">询问</button>
            <button :class="{ active: closeAction === 'quit' }" @click="chooseCloseAction('quit')">直接退出</button>
            <button :class="{ active: closeAction === 'tray' }" @click="chooseCloseAction('tray')">
              <Icon name="minus" size="sm" />最小化到托盘
            </button>
          </div>
        </div>
      </div>
      <div class="set-row">
        <div class="set-copy">
          <div class="set-label">系统托盘</div>
          <div class="set-desc">托盘菜单提供「显示主窗口」「隐藏到托盘」和「退出 Xin Skill Vault」。</div>
        </div>
        <div class="set-control">
          <span class="health-pill ok"><span class="dot moss" />始终可用</span>
        </div>
      </div>
    </section>

    <section class="set-group about-group">
      <h2>关于</h2>
      <div class="about-brand">
        <svg class="brand-mark lg" viewBox="0 0 24 24"><use href="#i-logo" /></svg>
        <div>
          <div class="about-name">Xin Skill Vault <span>v{{ snapshot.vault.appVersion || '0.1.0' }}</span></div>
          <div class="about-sub">Xin技能管理 · Agent Skill 仓库管理器</div>
        </div>
        <span class="tag">Apache-2.0</span>
      </div>
      <div class="about-meta">
        <div><span>平台</span><strong>{{ snapshot.vault.platform }}</strong></div>
        <div><span>技能</span><strong>{{ snapshot.stats.skills }}</strong></div>
        <div><span>分发</span><strong>{{ snapshot.stats.links + snapshot.stats.copies }}</strong></div>
        <div><span>分组</span><strong>{{ snapshot.groups.length }}</strong></div>
        <div><span>来源</span><strong>{{ snapshot.sources.length }}</strong></div>
        <div><span>仓库</span><strong>{{ formatBytes(snapshot.stats.vaultSizeBytes) }}</strong></div>
      </div>
      <div class="about-foot">
        <span>本地优先 · 无账号 · 无遥测 · 不执行技能内容</span>
      </div>
    </section>
  </div>

  <Teleport to="body">
    <div v-if="vaultDialogOpen" class="modal-overlay" @click.self="closeVaultDialog">
      <div class="modal vault-modal" role="dialog" aria-modal="true" aria-label="更改技能仓库位置">
        <header class="modal-head compact">
          <div>
            <div class="eyebrow">VAULT · 仓库位置</div>
            <h2>迁移技能仓库？</h2>
            <p>新目录必须为空。选择移动或复制后，应用会切换到新目录并重建链接分发。</p>
          </div>
          <button class="modal-close" aria-label="取消" @click="closeVaultDialog"><Icon name="x" /></button>
        </header>
        <div class="confirm-body">
          <div class="vault-path-stack">
            <div class="vault-path-row">
              <span class="hash-label">当前目录</span>
              <code>{{ snapshot.vault.path }}</code>
            </div>
            <div class="vault-path-row">
              <span class="hash-label">新目录</span>
              <code>{{ vaultNewPath }}</code>
            </div>
          </div>
          <p class="vault-note">
            「移动」会把现有 skills 移到新目录并清理旧位置；「复制」会保留旧目录。Junction / Symlink 分发会自动指向新仓库，复制副本保持不变。
          </p>
          <div v-if="vaultError" class="inline-error">
            <Icon name="alert" size="sm" />
            <span>{{ vaultError }}</span>
          </div>
        </div>
        <footer class="modal-foot">
          <button class="btn ghost" :disabled="vaultChanging" @click="closeVaultDialog">取消</button>
          <button class="btn" :disabled="vaultChanging" @click="changeVault('copy')">
            <Icon name="copy" size="sm" />复制到新目录
          </button>
          <button class="btn primary" :disabled="vaultChanging" @click="changeVault('move')">
            <Icon name="arrow-up-right" size="sm" />移动到新目录
          </button>
        </footer>
      </div>
    </div>
  </Teleport>
</template>
