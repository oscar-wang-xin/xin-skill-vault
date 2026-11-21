<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import { errorMessage, formatBytes, getVaultSummary, openVaultFolder } from "../lib/api";
import type { VaultSummary } from "../../bindings/xinskillvault/internal/app/models";

type Tab = "mine" | "scan" | "groups";
const tab = ref<Tab>("mine");
const summary = ref<VaultSummary | null>(null);
const loading = ref(true);
const refreshing = ref(false);
const viewMode = ref<"list" | "grid">("grid");
const sortBy = ref("name");
const query = ref("");
const filter = ref("all");
const toast = ref("");
let toastTimer: number | undefined;

const tabs: { id: Tab; label: string; count?: number }[] = [
  { id: "mine", label: "Skill 仓库" },
  { id: "scan", label: "发现" },
  { id: "groups", label: "组合" },
];

const filters = computed(() => [
  { id: "all", label: "全部", count: summary.value?.skillCount ?? 0 },
  { id: "distributed", label: "已分发", count: 0 },
  { id: "none", label: "未分发", count: summary.value?.skillCount ?? 0 },
  { id: "update", label: "有更新", count: 0 },
  { id: "disabled", label: "已禁用", count: 0 },
]);

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2200);
}

async function load() {
  loading.value = true;
  try {
    summary.value = await getVaultSummary();
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function refresh() {
  if (refreshing.value) return;
  refreshing.value = true;
  await load();
  refreshing.value = false;
  showToast("已刷新本地索引");
}

async function openRepo() {
  try {
    await openVaultFolder();
  } catch (error) {
    showToast(errorMessage(error));
  }
}

onMounted(load);
</script>

<template>
  <div class="page library-page">
    <div class="tabs">
      <button
        v-for="item in tabs"
        :key="item.id"
        class="tab"
        :class="{ active: tab === item.id }"
        @click="tab = item.id"
      >
        {{ item.label }}
        <i v-if="item.id === 'mine'">{{ summary?.skillCount ?? 0 }}</i>
        <i v-else-if="item.id === 'groups'">0</i>
      </button>
    </div>

    <template v-if="tab === 'mine'">
      <div class="lib-rows">
        <div class="lib-row">
          <span class="repo-label">本地仓库</span>
          <button class="repo-path" title="打开本地仓库目录" @click="openRepo">
            <Icon name="folder" :size="12" />
            <span class="mono">{{ summary?.vaultPath ?? "~/.skill-repo" }}</span>
          </button>
          <div class="grow" />
          <button class="btn primary" disabled title="批量分发将在 M1 接入">分发</button>
          <button class="btn" title="导入将在 M1 接入" @click="showToast('导入将在 M1 接入')">
            <Icon name="upload" :size="14" /> 导入
          </button>
          <button class="btn" :disabled="refreshing" title="重新读取本地索引" @click="refresh">
            <Icon name="refresh" :size="14" :class="{ spin: refreshing }" />
            {{ refreshing ? "刷新中…" : "刷新" }}
          </button>
        </div>

        <div class="lib-row filters-row">
          <div class="filters">
            <button
              v-for="item in filters"
              :key="item.id"
              class="chip"
              :class="{ active: filter === item.id }"
              @click="filter = item.id"
            >
              {{ item.label }} <span class="chip-count">{{ item.count }}</span>
            </button>
          </div>
          <div class="grow" />
          <div class="search lib-search">
            <Icon name="search" :size="14" />
            <input v-model="query" class="input" placeholder="搜索技能名称、描述或标签…" />
          </div>
        </div>

        <div class="lib-row">
          <div class="seg">
            <button :class="{ active: viewMode === 'list' }" title="列表视图" @click="viewMode = 'list'"><Icon name="list" :size="13" /></button>
            <button :class="{ active: viewMode === 'grid' }" title="卡片视图" @click="viewMode = 'grid'"><Icon name="grid" :size="13" /></button>
          </div>
          <select v-model="sortBy" class="input sort-select" title="排序字段">
            <option value="name">名称</option>
            <option value="updated">更新时间</option>
            <option value="source">来源</option>
            <option value="agents">分发数</option>
          </select>
          <button class="btn sort-dir" title="切换正序 / 倒序"><Icon name="chevronDown" :size="11" /> 正序</button>
          <div class="grow" />
          <span class="sel-count zero">已选 0</span>
          <button class="chip ghost-chip" disabled>全选</button>
          <button class="chip ghost-chip" disabled>全不选</button>
        </div>
      </div>

      <div v-if="loading" class="empty">
        <div class="empty-ico"><Icon name="database" :size="18" /></div>
        <h3>正在读取本地索引…</h3>
      </div>
      <div v-else class="empty">
        <div class="empty-ico"><Icon name="library" :size="19" /></div>
        <h3>仓库还没有技能</h3>
        <p>
          M0 已完成 Wails、SQLite 与配置骨架。下一步 M1 接入目录扫描、导入与技能详情，
          首个技能入库后会显示在这里。
        </p>
        <div class="empty-actions">
          <button class="btn" @click="openRepo"><Icon name="folder" :size="13" /> 打开仓库目录</button>
          <button class="btn ghost" @click="showToast('导入将在 M1 接入')">查看 M1 计划</button>
        </div>
        <div class="runtime-note">
          <span class="badge accent">SQLite v{{ summary?.schemaVersion ?? 1 }}</span>
          <span>{{ formatBytes(summary?.databaseBytes ?? 0) }}</span>
          <span>·</span>
          <span>{{ summary?.targetCount ?? 0 }} 个目标</span>
          <span>·</span>
          <span>{{ summary?.promptCount ?? 0 }} 条提示词</span>
        </div>
      </div>
    </template>

    <div v-else class="empty">
      <div class="empty-ico"><Icon :name="tab === 'scan' ? 'search' : 'layers'" :size="18" /></div>
      <h3>{{ tab === "scan" ? "发现" : "组合" }}将在 M1 接入</h3>
      <p>
        {{ tab === "scan"
          ? "扫描本机 Agent 目录、识别未导入/有变更技能，并支持忽略与恢复。"
          : "把技能编成组合，支持搜索、排序、编辑与一键分发。" }}
      </p>
      <button class="btn ghost" @click="tab = 'mine'">返回 Skill 仓库</button>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.library-page { padding-top: 8px; }
.lib-rows {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 4px 0 11px;
  margin-bottom: 4px;
}
.lib-rows::after {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 1px;
  background: var(--border-soft);
}
.lib-row { display: flex; align-items: center; gap: 7px; min-width: 0; flex-wrap: wrap; row-gap: 6px; }
.repo-label {
  color: var(--text-4);
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: .08em;
  text-transform: uppercase;
}
.repo-path {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border: 1px solid var(--border);
  border-radius: 99px;
  background: var(--panel);
  color: var(--text-3);
  font-size: 10.5px;
}
.repo-path:hover { color: var(--accent); border-color: var(--accent); }
.filters { display: flex; gap: 5px; flex-wrap: wrap; }
.chip-count { color: var(--text-4); font-size: 9px; }
.chip.active .chip-count { color: var(--accent); }
.lib-search { width: 300px; max-width: 100%; flex: none; }
.seg {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  border-radius: 8px;
  background: var(--panel-3);
  box-shadow: var(--nm-inset);
}
.seg button {
  display: grid;
  place-items: center;
  width: 27px;
  height: 23px;
  border-radius: 6px;
  color: var(--text-4);
}
.seg button.active { color: var(--text); background: var(--panel); box-shadow: var(--shadow-sm); }
.sort-select { width: 108px; height: 27px; font-size: 11px; }
.sort-dir { height: 27px; font-size: 10.5px; }
.ghost-chip { color: var(--text-4); border-color: transparent; background: transparent; }
.sel-count { font-size: 10px; color: var(--accent); background: var(--accent-soft); border-radius: 99px; padding: 2px 7px; }
.sel-count.zero { color: var(--text-4); background: var(--panel-3); }
.empty-actions { display: flex; gap: 7px; margin-top: 14px; }
.runtime-note { display: flex; align-items: center; gap: 7px; margin-top: 18px; color: var(--text-4); font-size: 10px; }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 900px) {
  .lib-search { width: 230px; }
}
</style>
