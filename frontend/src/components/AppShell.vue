<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { RouterLink, RouterView, useRoute } from "vue-router";
import { getAppInfo, getVaultSummary, openVaultFolder, errorMessage } from "../lib/api";
import type { AppInfo, VaultSummary } from "../../bindings/xinskillvault/internal/app/models";
import { themes, useUiStore } from "../stores/ui";
import Icon from "./Icon.vue";
import CommandPalette from "./CommandPalette.vue";

const ui = useUiStore();
const route = useRoute();
const info = ref<AppInfo | null>(null);
const summary = ref<VaultSummary | null>(null);
const loading = ref(true);
const toast = ref("");
let toastTimer: number | undefined;

const nav = [
  { path: "/library", label: "技能库", icon: "library", countKey: "skillCount" },
  { path: "/market", label: "市场", icon: "market", countKey: "" },
  { path: "/agent", label: "Agent", icon: "agent", countKey: "targetCount" },
  { path: "/projects", label: "项目", icon: "projects", countKey: "" },
  { path: "/prompts", label: "提示词", icon: "prompts", countKey: "promptCount" },
];
const systemNav = [{ path: "/settings", label: "设置", icon: "settings" }];

const pageTitle = computed(() => String(route.meta?.title ?? "技能库"));
const pageGroup = computed(() => String(route.meta?.group ?? "工作区"));
const repoPath = computed(() => summary.value?.vaultPath ?? info.value?.vaultPath ?? "~/.skill-repo");

function countFor(key: string): string {
  if (!key || !summary.value) return "";
  const value = summary.value[key as keyof VaultSummary];
  return typeof value === "number" && value > 0 ? String(value) : "";
}

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

async function load() {
  loading.value = true;
  try {
    const [nextInfo, nextSummary] = await Promise.all([getAppInfo(), getVaultSummary()]);
    info.value = nextInfo;
    summary.value = nextSummary;
    if (nextInfo.name) document.title = nextInfo.name;
  } catch (error) {
    showToast(`初始化失败：${errorMessage(error)}`);
  } finally {
    loading.value = false;
  }
}

async function openRepo() {
  try {
    await openVaultFolder();
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function toggleThemeMenu() {
  ui.themeMenuOpen = !ui.themeMenuOpen;
}

function onWindowClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null;
  if (!target?.closest(".theme-wrap")) ui.themeMenuOpen = false;
}

function onKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
    event.preventDefault();
    ui.commandOpen = !ui.commandOpen;
    return;
  }
  if (event.key === "Escape") {
    ui.commandOpen = false;
    ui.themeMenuOpen = false;
  }
}

watch(() => route.path, () => { ui.themeMenuOpen = false; });

onMounted(() => {
  load();
  window.addEventListener("click", onWindowClick);
  window.addEventListener("keydown", onKeydown);
});
onUnmounted(() => {
  window.removeEventListener("click", onWindowClick);
  window.removeEventListener("keydown", onKeydown);
  window.clearTimeout(toastTimer);
});
</script>

<template>
  <div class="shell" :data-collapsed="ui.sidebarCollapsed">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">XIN</div>
        <div class="brand-name">
          <b>Xin Skill Vault</b>
          <span>AI AGENT SKILL VAULT</span>
        </div>
      </div>

      <nav>
        <div class="nav-section">
          <div class="nav-label">工作区</div>
          <RouterLink v-for="item in nav" :key="item.path" :to="item.path" class="nav-item">
            <Icon :name="item.icon" :size="15" />
            <span>{{ item.label }}</span>
            <span v-if="countFor(item.countKey)" class="nav-count">{{ countFor(item.countKey) }}</span>
          </RouterLink>
        </div>
        <div class="nav-section">
          <div class="nav-label">系统</div>
          <RouterLink v-for="item in systemNav" :key="item.path" :to="item.path" class="nav-item">
            <Icon :name="item.icon" :size="15" />
            <span>{{ item.label }}</span>
          </RouterLink>
        </div>
      </nav>

      <div class="sidebar-bottom">
        <button class="repo-mini" data-tip="打开本地仓库目录" @click="openRepo">
          <div class="repo-mini-head">
            <Icon name="folder" :size="12" />
            <span>本地仓库</span>
          </div>
          <span class="repo-mini-path mono">{{ repoPath }}</span>
          <div class="repo-mini-meta">
            <span>{{ summary?.skillCount ?? 0 }} 个技能</span>
            <span>v{{ info?.version ?? "0.1.0" }}</span>
          </div>
        </button>
      </div>
    </aside>

    <main class="main">
      <header class="topbar">
        <button class="icon-btn plain collapse" title="折叠 / 展开侧边栏" @click="ui.toggleSidebar">
          <Icon name="chevronRight" :size="15" :style="{ transform: ui.sidebarCollapsed ? 'rotate(0)' : 'rotate(180deg)' }" />
        </button>
        <div class="crumbs">
          <span>{{ pageGroup }}</span>
          <Icon name="chevronRight" :size="11" />
          <b>{{ pageTitle }}</b>
        </div>
        <div class="top-actions">
          <button
            class="switch"
            :class="{ on: ui.neumorph }"
            :title="ui.neumorph ? '拟态风格：开' : '拟态风格：关'"
            role="switch"
            :aria-checked="ui.neumorph"
            @click="ui.toggleNeumorph()"
          />
          <div class="theme-wrap">
            <button class="icon-btn" title="选择主题" @click.stop="toggleThemeMenu">
              <Icon name="palette" :size="15" />
            </button>
            <div v-if="ui.themeMenuOpen" class="theme-pop" @click.stop>
              <button
                v-for="theme in themes"
                :key="theme.id"
                :class="{ active: ui.theme === theme.id }"
                @click="ui.setTheme(theme.id)"
              >
                <span class="theme-dots">
                  <i v-for="color in theme.colors" :key="color" :style="{ background: color }" />
                </span>
                <span>{{ theme.name }}</span>
                <Icon v-if="ui.theme === theme.id" class="theme-check" name="check" :size="12" />
              </button>
            </div>
          </div>
          <button class="icon-btn" title="通知" @click="showToast('M0 骨架运行中，功能将按 M1–M5 接入')">
            <Icon name="bell" :size="15" />
          </button>
        </div>
      </header>

      <div class="content">
        <RouterView v-if="!loading" />
        <div v-else class="page">
          <div class="empty">
            <div class="empty-ico"><Icon name="database" :size="18" /></div>
            <h3>正在准备本地索引…</h3>
            <p>检查仓库目录、SQLite 迁移与配置。</p>
          </div>
        </div>
      </div>
    </main>

    <div v-if="toast" class="toast">{{ toast }}</div>
    <CommandPalette :open="ui.commandOpen" @close="ui.commandOpen = false" />
  </div>
</template>

<style scoped>
.switch {
  position: relative;
  width: 38px;
  height: 20px;
  border-radius: 99px;
  background: var(--panel-3);
  box-shadow: var(--nm-inset);
  transition: background var(--dur) var(--ease);
}
.switch::after {
  content: "";
  position: absolute;
  top: 3px;
  left: 3px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--panel);
  box-shadow: 0 1px 3px rgba(16, 18, 22, .22);
  transition: transform var(--dur) var(--ease);
}
.switch.on { background: var(--accent); }
.switch.on::after { transform: translateX(18px); }
.collapse svg { transition: transform var(--dur) var(--ease); }
</style>
