<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { isDesktop } from '../lib/api'
import { SKINS, SKIN_MAP, type SkinID } from '../lib/types'
import { shortPath } from '../lib/format'
import Icon from './Icon.vue'

const props = defineProps<{
  vaultPath: string
  skin: SkinID
  syncLabel: string
}>()

const emit = defineEmits<{
  (e: 'open-palette'): void
  (e: 'open-repo'): void
  (e: 'set-skin', id: SkinID): void
  (e: 'open-settings'): void
}>()

const menuOpen = ref(false)
const root = ref<HTMLElement | null>(null)

const current = computed(() => SKIN_MAP[props.skin] || SKIN_MAP['cold-white'])
const repoLabel = computed(() => shortPath(props.vaultPath, 34))

function pick(id: SkinID) {
  emit('set-skin', id)
  menuOpen.value = false
}

function minimise() {
  if (!isDesktop) return
  import('@wailsio/runtime').then((runtime) => runtime.Window.Minimise())
}

function toggleMaximise() {
  if (!isDesktop) return
  import('@wailsio/runtime').then((runtime) => runtime.Window.ToggleMaximise())
}

function close() {
  if (!isDesktop) return
  import('@wailsio/runtime').then((runtime) => runtime.Window.Close())
}

function onDocClick(event: MouseEvent) {
  if (!menuOpen.value) return
  const target = event.target as Node
  if (root.value && !root.value.contains(target)) menuOpen.value = false
}

function onKey(event: KeyboardEvent) {
  if (event.key === 'Escape') menuOpen.value = false
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onKey)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onKey)
})
</script>

<template>
  <header class="titlebar" @dblclick.self="toggleMaximise">
    <div class="brand" title="Xin Skill Vault · Xin技能管理">
      <svg class="brand-mark" viewBox="0 0 24 24"><use href="#i-logo" /></svg>
      <span class="wordmark">XIN SKILL VAULT</span>
    </div>
    <button class="repo-pill" title="打开仓库目录" @click="emit('open-repo')">
      <span class="dot moss" />
      <span class="repo-pill-path">{{ repoLabel }}</span>
      <Icon name="chevron-down" size="sm" />
    </button>
    <div class="title-spacer" />
    <button class="title-command" @click="emit('open-palette')">
      <Icon name="search" size="sm" />
      <span>搜索或执行</span>
      <kbd>Ctrl</kbd><kbd>K</kbd>
    </button>
    <span class="sync-time">{{ syncLabel }}</span>
    <div ref="root" class="skin-anchor">
      <button
        class="title-icon"
        :class="{ active: menuOpen }"
        :title="`当前皮肤：${current.name}`"
        aria-label="换肤"
        @click.stop="menuOpen = !menuOpen"
      >
        <Icon name="palette" />
      </button>
      <Transition name="pop">
        <div v-if="menuOpen" class="skin-pop" @click.stop>
          <div class="skin-pop-head">
            <span>换肤 / SKINS</span>
            <span class="skin-pop-current">{{ current.name }}</span>
          </div>
          <button
            v-for="s in SKINS"
            :key="s.id"
            class="skin-option"
            :class="{ active: s.id === skin }"
            @click="pick(s.id as SkinID)"
          >
            <span class="skin-swatch">
              <i v-for="color in s.colors" :key="color" :style="{ background: color }" />
            </span>
            <span class="skin-option-copy">
              <span class="skin-option-name">{{ s.name }}</span>
              <span class="skin-option-en">{{ s.en }}</span>
            </span>
            <span class="skin-check">{{ s.id === skin ? '✓' : '' }}</span>
          </button>
          <button class="skin-pop-foot" @click="menuOpen = false; emit('open-settings')">
            在设置中管理皮肤与偏好
            <Icon name="chevron-right" size="sm" />
          </button>
        </div>
      </Transition>
    </div>
    <div class="win-controls">
      <button class="win-btn" aria-label="最小化" title="最小化" @click="minimise">
        <Icon name="minus" />
      </button>
      <button class="win-btn" aria-label="最大化" title="最大化 / 还原" @click="toggleMaximise">
        <Icon name="maximize" />
      </button>
      <button class="win-btn close" aria-label="关闭" title="关闭" @click="close">
        <Icon name="x" />
      </button>
    </div>
  </header>
</template>
