<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import {
  errorMessage,
  formatBytes,
  getSkillDetail,
  importSkillDirectory,
  listSkills,
  openSkillFolder,
  openVaultFolder,
  pickSkillDirectory,
} from "../lib/api";
import type { SkillDetail, SkillItem } from "../../bindings/xinskillvault/internal/app/models";
import { useVaultStore } from "../stores/vault";

type Tab = "mine" | "scan" | "groups";
const tab = ref<Tab>("mine");
const vault = useVaultStore();
const skills = ref<SkillItem[]>([]);
const loading = ref(true);
const refreshing = ref(false);
const importing = ref(false);
const viewMode = ref<"list" | "grid">("grid");
const sortBy = ref<"name" | "updated" | "source">("name");
const sortAsc = ref(true);
const query = ref("");
const filter = ref("all");
const checked = ref(new Set<string>());
const activeSlug = ref<string | null>(null);
const detail = ref<SkillDetail | null>(null);
const detailLoading = ref(false);
const toast = ref("");
const closeButton = ref<HTMLButtonElement | null>(null);
let toastTimer: number | undefined;
let returnFocus: HTMLElement | null = null;

const tabs: { id: Tab; label: string }[] = [
  { id: "mine", label: "Skill 仓库" },
  { id: "scan", label: "发现" },
  { id: "groups", label: "组合" },
];

const sourceLabel: Record<string, string> = {
  dir: "本地目录",
  zip: "压缩包",
  url: "链接",
  market: "市场",
  scan: "本地扫描",
  ai: "自建",
};

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  let list = skills.value.filter((item) => {
    if (filter.value === "disabled" && item.enabled) return false;
    if (filter.value === "update") return false;
    if (filter.value === "distributed" || filter.value === "none") return filter.value === "none";
    if (!q) return true;
    return item.name.toLowerCase().includes(q)
      || item.slug.toLowerCase().includes(q)
      || item.description.toLowerCase().includes(q);
  });
  const direction = sortAsc.value ? 1 : -1;
  list = list.slice().sort((a, b) => {
    let result = 0;
    if (sortBy.value === "updated") result = a.updatedAt - b.updatedAt;
    else if (sortBy.value === "source") result = a.sourceType.localeCompare(b.sourceType);
    else result = a.name.localeCompare(b.name, "zh-Hans-CN");
    if (!result) result = a.name.localeCompare(b.name, "zh-Hans-CN");
    return result * direction;
  });
  return list;
});

const filters = computed(() => [
  { id: "all", label: "全部", count: skills.value.length },
  { id: "distributed", label: "已分发", count: 0 },
  { id: "none", label: "未分发", count: skills.value.length },
  { id: "update", label: "有更新", count: 0 },
  { id: "disabled", label: "已禁用", count: skills.value.filter((item) => !item.enabled).length },
]);

const allChecked = computed(() => filtered.value.length > 0 && filtered.value.every((item) => checked.value.has(item.slug)));

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

function formatDate(seconds: number): string {
  if (!seconds) return "—";
  return new Date(seconds * 1000).toLocaleString("zh-CN", { hour12: false });
}

function initials(name: string): string {
  const trimmed = name.trim();
  return trimmed ? Array.from(trimmed).slice(0, 2).join("").toUpperCase() : "SK";
}

async function loadSkills() {
  loading.value = true;
  try {
    skills.value = await listSkills();
    const valid = new Set(skills.value.map((item) => item.slug));
    checked.value = new Set([...checked.value].filter((slug) => valid.has(slug)));
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function refresh() {
  if (refreshing.value) return;
  refreshing.value = true;
  await Promise.all([vault.refresh(), loadSkills()]);
  refreshing.value = false;
  showToast("已刷新本地仓库");
}

async function openRepo() {
  try {
    await openVaultFolder();
  } catch (error) {
    showToast(errorMessage(error));
  }
}

async function importSkill() {
  if (importing.value) return;
  importing.value = true;
  try {
    const source = await pickSkillDirectory();
    if (!source) return;
    const item = await importSkillDirectory(source);
    await Promise.all([vault.refresh(), loadSkills()]);
    showToast(`已导入：${item.name}`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    importing.value = false;
  }
}

function toggleCheck(slug: string) {
  if (checked.value.has(slug)) checked.value.delete(slug);
  else checked.value.add(slug);
  checked.value = new Set(checked.value);
}

function clearChecked() {
  checked.value = new Set();
}

function toggleAll() {
  if (allChecked.value) {
    filtered.value.forEach((item) => checked.value.delete(item.slug));
  } else {
    filtered.value.forEach((item) => checked.value.add(item.slug));
  }
  checked.value = new Set(checked.value);
}

async function openDetail(item: SkillItem) {
  returnFocus = document.activeElement as HTMLElement | null;
  activeSlug.value = item.slug;
  detail.value = null;
  detailLoading.value = true;
  await nextTick();
  closeButton.value?.focus();
  try {
    detail.value = await getSkillDetail(item.slug);
  } catch (error) {
    showToast(errorMessage(error));
    activeSlug.value = null;
  } finally {
    detailLoading.value = false;
  }
}

function closeDetail() {
  activeSlug.value = null;
  detail.value = null;
  nextTick(() => returnFocus?.focus());
}

async function openSkillDir() {
  if (!activeSlug.value) return;
  try {
    await openSkillFolder(activeSlug.value);
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Escape" && activeSlug.value) closeDetail();
}

onMounted(async () => {
  window.addEventListener("keydown", onKeydown);
  await Promise.all([vault.refresh(), loadSkills()]);
});
onUnmounted(() => {
  window.removeEventListener("keydown", onKeydown);
  window.clearTimeout(toastTimer);
});
</script>

<template>
  <div class="page library-page">
    <div class="tabs">
      <button v-for="item in tabs" :key="item.id" class="tab" :class="{ active: tab === item.id }" @click="tab = item.id">
        {{ item.label }}
        <i v-if="item.id === 'mine'">{{ skills.length }}</i>
        <i v-else-if="item.id === 'groups'">0</i>
      </button>
    </div>

    <template v-if="tab === 'mine'">
      <div class="lib-rows">
        <div class="lib-row">
          <span class="repo-label">本地仓库</span>
          <button class="repo-path" title="打开本地仓库目录" @click="openRepo">
            <Icon name="folder" :size="12" />
            <span class="mono">{{ vault.summary?.vaultPath ?? "~/.skill-repo" }}</span>
          </button>
          <div class="grow" />
          <button class="btn primary" disabled title="批量分发将在 M2 接入">
            <Icon name="link" :size="13" /> 分发
          </button>
          <button class="btn" :disabled="importing" title="从本地目录导入包含 SKILL.md 的技能" @click="importSkill">
            <Icon name="upload" :size="14" />
            {{ importing ? "导入中…" : "导入" }}
          </button>
          <button class="btn" :disabled="refreshing" title="重新读取本地仓库" @click="refresh">
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
            <input v-model="query" class="input" placeholder="搜索技能名称、描述或 slug…" />
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
          </select>
          <button class="btn sort-dir" title="切换正序 / 倒序" @click="sortAsc = !sortAsc">
            <Icon name="chevronDown" :size="11" :style="{ transform: sortAsc ? 'rotate(180deg)' : 'none' }" />
            {{ sortAsc ? "正序" : "倒序" }}
          </button>
          <div class="grow" />
          <span class="sel-count" :class="{ zero: !checked.size }">已选 {{ checked.size }}</span>
          <button class="chip ghost-chip" :disabled="!filtered.length || allChecked" @click="toggleAll">全选</button>
          <button class="chip ghost-chip" :disabled="!checked.size" @click="clearChecked">全不选</button>
        </div>
      </div>

      <div v-if="loading" class="empty">
        <div class="empty-ico"><Icon name="database" :size="18" /></div>
        <h3>正在读取本地仓库…</h3>
      </div>

      <div v-else-if="!filtered.length" class="empty">
        <div class="empty-ico"><Icon name="library" :size="19" /></div>
        <h3>{{ skills.length ? "没有匹配的技能" : "仓库还没有技能" }}</h3>
        <p v-if="skills.length">换个关键词或清空筛选条件。</p>
        <p v-else>从一个包含 <code>SKILL.md</code> 的目录导入，仓库会成为本机唯一可编辑副本。</p>
        <div class="empty-actions">
          <button v-if="!skills.length" class="btn primary" @click="importSkill"><Icon name="upload" :size="13" /> 导入技能目录</button>
          <button class="btn" @click="openRepo"><Icon name="folder" :size="13" /> 打开仓库目录</button>
          <button class="btn ghost" @click="query = ''; filter = 'all'">清空筛选</button>
        </div>
        <div class="runtime-note">
          <span class="badge accent">SQLite v{{ vault.summary?.schemaVersion ?? 1 }}</span>
          <span>{{ formatBytes(vault.summary?.databaseBytes ?? 0) }}</span>
          <span>·</span>
          <span>{{ vault.summary?.targetCount ?? 0 }} 个目标</span>
          <span>·</span>
          <span>{{ vault.summary?.promptCount ?? 0 }} 条提示词</span>
        </div>
      </div>

      <div v-else-if="viewMode === 'grid'" class="skill-grid">
        <div
          v-for="item in filtered"
          :key="item.id"
          class="skill-card"
          role="button"
          tabindex="0"
          @click="openDetail(item)"
          @keydown.enter.self="openDetail(item)"
        >
          <div class="sc-head">
            <span class="skill-glyph">{{ item.icon || initials(item.name) }}</span>
            <div class="sc-title">
              <b>{{ item.name }}</b>
              <span class="mono">{{ item.slug }}</span>
            </div>
            <button
              class="mini-check"
              :class="{ on: checked.has(item.slug) }"
              title="选择技能"
              @click.stop="toggleCheck(item.slug)"
            ><Icon v-if="checked.has(item.slug)" name="check" :size="10" /></button>
          </div>
          <p class="sc-desc">{{ item.description }}</p>
          <div class="sc-meta">
            <span class="badge" :class="{ accent: item.enabled }">{{ item.enabled ? "启用" : "禁用" }}</span>
            <span class="badge">{{ sourceLabel[item.sourceType] ?? item.sourceType }}</span>
            <span v-if="item.version" class="mono">v{{ item.version }}</span>
            <span class="grow" />
            <span>{{ item.fileCount }} 文件</span>
          </div>
          <div class="sc-foot">
            <span class="muted">{{ formatDate(item.updatedAt) }}</span>
            <span class="grow" />
            <span class="muted">{{ formatBytes(item.sizeBytes) }}</span>
          </div>
        </div>
      </div>

      <div v-else class="skill-table">
        <div class="st-row st-head">
          <span>#</span><span></span><span>技能</span><span>状态</span><span>来源</span><span>版本</span><span>文件</span><span>更新时间</span>
        </div>
        <div
          v-for="(item, index) in filtered"
          :key="item.id"
          class="st-row"
          role="button"
          tabindex="0"
          @click="openDetail(item)"
          @keydown.enter.self="openDetail(item)"
        >
          <span class="st-idx">{{ index + 1 }}</span>
          <button class="mini-check" :class="{ on: checked.has(item.slug) }" @click.stop="toggleCheck(item.slug)">
            <Icon v-if="checked.has(item.slug)" name="check" :size="10" />
          </button>
          <span class="st-main">
            <span class="skill-glyph sm">{{ item.icon || initials(item.name) }}</span>
            <span><b>{{ item.name }}</b><em>{{ item.description }}</em></span>
          </span>
          <span class="badge" :class="{ accent: item.enabled }">{{ item.enabled ? "启用" : "禁用" }}</span>
          <span class="badge">{{ sourceLabel[item.sourceType] ?? item.sourceType }}</span>
          <span class="mono">{{ item.version || "—" }}</span>
          <span>{{ item.fileCount }}</span>
          <span class="st-time">{{ formatDate(item.updatedAt) }}</span>
        </div>
      </div>
    </template>

    <div v-else class="empty">
      <div class="empty-ico"><Icon :name="tab === 'scan' ? 'search' : 'layers'" :size="18" /></div>
      <h3>{{ tab === "scan" ? "发现" : "组合" }}将在 M1–M2 接入</h3>
      <p>
        {{ tab === "scan"
          ? "扫描本机 Agent 目录、识别未导入/有变更技能，并支持忽略与恢复。"
          : "把技能编成组合，支持搜索、排序、编辑与一键分发。" }}
      </p>
      <button class="btn ghost" @click="tab = 'mine'">返回 Skill 仓库</button>
    </div>

    <div v-if="activeSlug" class="drawer-scrim" aria-hidden="true" />
    <aside v-if="activeSlug" class="skill-drawer" role="dialog" aria-label="技能详情">
      <header class="drawer-head">
        <div>
          <h2>{{ detail?.name ?? "技能详情" }}</h2>
          <span class="mono">{{ detail?.slug ?? activeSlug }}</span>
        </div>
        <button ref="closeButton" class="icon-btn" title="关闭详情" @click="closeDetail"><Icon name="x" :size="16" /></button>
      </header>
      <div v-if="detailLoading" class="drawer-loading">正在读取技能目录…</div>
      <div v-else-if="detail" class="drawer-body">
        <p class="detail-desc">{{ detail.description }}</p>
        <div class="detail-tags">
          <span class="badge accent">{{ detail.enabled ? "启用" : "禁用" }}</span>
          <span class="badge">{{ sourceLabel[detail.sourceType] ?? detail.sourceType }}</span>
          <span v-if="detail.version" class="badge">v{{ detail.version }}</span>
          <span v-if="detail.author" class="badge">{{ detail.author }}</span>
        </div>
        <dl class="detail-kv">
          <div><dt>路径</dt><dd class="mono">{{ detail.path }}</dd></div>
          <div><dt>指纹</dt><dd class="mono">{{ detail.contentHash || "—" }}</dd></div>
          <div><dt>大小</dt><dd>{{ formatBytes(detail.sizeBytes) }}</dd></div>
          <div><dt>文件</dt><dd>{{ detail.fileCount }}</dd></div>
          <div><dt>更新</dt><dd>{{ formatDate(detail.updatedAt) }}</dd></div>
        </dl>
        <h3>技能文件</h3>
        <div class="file-list">
          <div v-for="file in detail.files" :key="file.path" class="file-row">
            <Icon :name="file.isDir ? 'folder' : 'file'" :size="12" />
            <span class="mono">{{ file.path }}</span>
            <span class="muted">{{ file.isDir ? "" : formatBytes(file.size) }}</span>
          </div>
        </div>
      </div>
      <footer class="drawer-foot">
        <button class="btn" @click="openSkillDir"><Icon name="folder" :size="13" /> 打开目录</button>
        <div class="grow" />
        <button class="btn" @click="closeDetail">关闭</button>
      </footer>
    </aside>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.library-page { padding-top: 8px; }
.lib-rows { position: relative; display: flex; flex-direction: column; gap: 8px; padding: 4px 0 11px; margin-bottom: 4px; }
.lib-rows::after { content: ""; position: absolute; left: 0; right: 0; bottom: 0; height: 1px; background: var(--border-soft); }
.lib-row { display: flex; align-items: center; gap: 7px; min-width: 0; flex-wrap: wrap; row-gap: 6px; }
.repo-label { color: var(--text-4); font-size: 9.5px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
.repo-path { display: inline-flex; align-items: center; gap: 5px; padding: 3px 8px; border: 1px solid var(--border); border-radius: 99px; background: var(--panel); color: var(--text-3); font-size: 10.5px; }
.repo-path:hover { color: var(--accent); border-color: var(--accent); }
.filters { display: flex; gap: 5px; flex-wrap: wrap; }
.chip-count { color: var(--text-4); font-size: 9px; }
.chip.active .chip-count { color: var(--accent); }
.lib-search { width: 300px; max-width: 100%; flex: none; }
.seg { display: inline-flex; gap: 2px; padding: 2px; border-radius: 8px; background: var(--panel-3); box-shadow: var(--nm-inset); }
.seg button { display: grid; place-items: center; width: 27px; height: 23px; border-radius: 6px; color: var(--text-4); }
.seg button.active { color: var(--text); background: var(--panel); box-shadow: var(--shadow-sm); }
.sort-select { width: 108px; height: 27px; font-size: 11px; }
.sort-dir { height: 27px; font-size: 10.5px; }
.ghost-chip { color: var(--text-4); border-color: transparent; background: transparent; }
.sel-count { font-size: 10px; color: var(--accent); background: var(--accent-soft); border-radius: 99px; padding: 2px 7px; }
.sel-count.zero { color: var(--text-4); background: var(--panel-3); }
.empty-actions { display: flex; gap: 7px; margin-top: 14px; flex-wrap: wrap; justify-content: center; }
.runtime-note { display: flex; align-items: center; gap: 7px; margin-top: 18px; color: var(--text-4); font-size: 10px; }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.skill-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(238px, 1fr)); gap: 10px; }
.skill-card { display: flex; flex-direction: column; gap: 8px; min-height: 146px; padding: 13px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--panel); text-align: left; box-shadow: var(--shadow-sm); transition: border-color var(--dur) var(--ease), transform var(--dur) var(--ease); }
.skill-card:hover { border-color: var(--border-strong); transform: translateY(-1px); }
.sc-head { display: flex; align-items: center; gap: 8px; min-width: 0; }
.skill-glyph { display: grid; place-items: center; width: 30px; height: 30px; border-radius: 8px; color: var(--accent); background: var(--accent-soft); font-size: 10px; font-weight: 700; flex: none; }
.skill-glyph.sm { width: 22px; height: 22px; border-radius: 6px; font-size: 8px; }
.sc-title { min-width: 0; flex: 1; }
.sc-title b { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; font-weight: 600; }
.sc-title span { display: block; margin-top: 2px; color: var(--text-4); font-size: 9px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mini-check { display: grid; place-items: center; width: 16px; height: 16px; border: 1px solid var(--border-strong); border-radius: 5px; background: var(--panel); color: var(--accent-fg); flex: none; }
.mini-check.on { background: var(--accent); border-color: var(--accent); }
.sc-desc { display: -webkit-box; overflow: hidden; -webkit-line-clamp: 2; -webkit-box-orient: vertical; margin: 0; color: var(--text-3); font-size: 10.5px; line-height: 1.55; }
.sc-meta { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; color: var(--text-4); font-size: 9px; }
.sc-foot { display: flex; align-items: center; margin-top: auto; padding-top: 8px; border-top: 1px solid var(--border-soft); color: var(--text-4); font-size: 9px; }
.sc-foot .muted { font-size: 9px; }

.skill-table { border: 1px solid var(--border); border-radius: var(--radius); background: var(--panel); overflow: hidden; }
.st-row { display: grid; grid-template-columns: 22px 18px minmax(0, 1fr) 54px 66px 56px 42px 112px; gap: 8px; align-items: center; width: 100%; padding: 8px 12px; border-bottom: 1px solid var(--border-soft); text-align: left; font-size: 10.5px; color: var(--text-2); }
.st-row:last-child { border-bottom: 0; }
.st-row:not(.st-head):hover { background: var(--panel-2); }
.st-head { color: var(--text-4); background: var(--panel-2); font-size: 9px; padding-top: 6px; padding-bottom: 6px; }
.st-idx { color: var(--text-4); text-align: center; }
.st-main { display: flex; align-items: center; gap: 7px; min-width: 0; }
.st-main > span:last-child { min-width: 0; }
.st-main b { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text); font-size: 11.5px; font-weight: 550; }
.st-main em { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-top: 1px; color: var(--text-4); font-size: 9px; font-style: normal; }
.st-time { color: var(--text-4); font-size: 9px; }

.drawer-scrim { position: fixed; inset: 0; z-index: 60; background: rgba(18, 20, 26, 0.28); backdrop-filter: blur(2px); }
.skill-drawer { position: fixed; top: var(--topbar-h); right: 0; bottom: 0; z-index: 61; width: min(420px, 86vw); display: flex; flex-direction: column; border-left: 1px solid var(--border); background: var(--panel); box-shadow: -24px 0 56px -22px rgba(16, 18, 22, .42); }
.drawer-head { display: flex; align-items: center; gap: 10px; padding: 12px 14px; border-bottom: 1px solid var(--border-soft); flex: none; }
.drawer-head h2 { margin: 0; font-size: 14px; font-weight: 650; }
.drawer-head span { display: block; margin-top: 2px; color: var(--text-4); font-size: 9.5px; }
.drawer-head .icon-btn { margin-left: auto; }
.drawer-loading { padding: 24px; color: var(--text-4); font-size: 11px; }
.drawer-body { flex: 1; overflow: auto; padding: 13px 14px; }
.detail-desc { margin: 0; color: var(--text-3); font-size: 11.5px; line-height: 1.65; }
.detail-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 9px; }
.detail-kv { margin: 14px 0 0; }
.detail-kv div { display: grid; grid-template-columns: 52px minmax(0, 1fr); gap: 8px; padding: 6px 0; border-bottom: 1px dashed var(--border-soft); font-size: 10.5px; }
.detail-kv dt { color: var(--text-4); }
.detail-kv dd { margin: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-2); }
.drawer-body h3 { margin: 16px 0 7px; font-size: 10px; font-weight: 650; letter-spacing: .06em; text-transform: uppercase; color: var(--text-4); }
.file-list { border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
.file-row { display: flex; align-items: center; gap: 7px; padding: 6px 8px; border-bottom: 1px dashed var(--border-soft); font-size: 9.5px; color: var(--text-2); }
.file-row:last-child { border-bottom: 0; }
.file-row span:nth-child(2) { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.drawer-foot { display: flex; align-items: center; gap: 6px; padding: 10px 14px; border-top: 1px solid var(--border-soft); flex: none; }
@media (max-width: 900px) { .lib-search { width: 230px; } .st-row { grid-template-columns: 18px 16px minmax(0,1fr) 46px 56px 44px 34px 84px; } }
</style>
