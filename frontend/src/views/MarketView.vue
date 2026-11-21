<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import { auditMarketSkill, errorMessage, installMarketSkill, listMarketCategories, listMarketSources, searchMarket } from "../lib/api";
import type { Audit as MarketAudit, Category as MarketCategory, Skill as MarketSkill, Source as MarketSource } from "../../bindings/xinskillvault/internal/market/models";
import { useVaultStore } from "../stores/vault";

const vault = useVaultStore();
const sources = ref<MarketSource[]>([]);
const categories = ref<MarketCategory[]>([]);
const activeSource = ref("all");
const activeCategory = ref("");
const query = ref("");
const items = ref<MarketSkill[]>([]);
const total = ref(0);
const page = ref(1);
const loading = ref(false);
const installing = ref("");
const auditing = ref("");
const auditTarget = ref<MarketSkill | null>(null);
const auditResult = ref<MarketAudit | null>(null);
const sortMode = ref<"installs" | "name">("installs");
const toast = ref("");
let toastTimer: number | undefined;

const active = computed(() => sources.value.find((item) => item.id === activeSource.value));
const unavailable = computed(() => !!active.value && active.value.status !== "available");
const sorted = computed(() => items.value.slice().sort((a, b) => {
  if (sortMode.value === "name") return a.name.localeCompare(b.name, "zh-Hans-CN");
  return b.installs - a.installs || a.name.localeCompare(b.name);
}));

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2600);
}

function formatInstalls(value: number): string {
  if (value >= 1000000) return `${(value / 1000000).toFixed(1)}m`;
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`;
  return String(value);
}

function initials(name: string): string {
  return Array.from(name.trim()).slice(0, 2).join("").toUpperCase() || "SK";
}

async function search(reset = true) {
  if (loading.value) return;
  if (reset) page.value = 1;
  loading.value = true;
  try {
    const result = await searchMarket(activeSource.value, query.value, activeCategory.value, page.value);
    items.value = reset ? (result.items ?? []) : [...items.value, ...(result.items ?? [])];
    total.value = result.total ?? 0;
  } catch (error) {
    if (reset) items.value = [];
    showToast(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function loadMore() {
  if (items.value.length >= total.value || loading.value) return;
  page.value += 1;
  await search(false);
}

async function doInstall(skill: MarketSkill, force: boolean) {
  if (skill.installed || installing.value) return;
  installing.value = skill.id;
  try {
    await installMarketSkill(skill.source, skill.slug, skill.namespace, force);
    skill.installed = true;
    await vault.refresh();
    showToast(`已入库：${skill.name}`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    installing.value = "";
  }
}

async function install(skill: MarketSkill) {
  if (skill.installed || installing.value || auditing.value) return;
  auditing.value = skill.id;
  try {
    const audit = await auditMarketSkill(skill.source, skill.slug, skill.namespace);
    if (audit.level === "high") {
      auditTarget.value = skill;
      auditResult.value = audit;
      return;
    }
    if (audit.level === "medium") showToast("检查到脚本文件，安装后请自行确认内容");
    await doInstall(skill, false);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    auditing.value = "";
  }
}

function cancelAudit() {
  if (installing.value) return;
  auditTarget.value = null;
  auditResult.value = null;
}

async function confirmInstall() {
  if (!auditTarget.value) return;
  const skill = auditTarget.value;
  await doInstall(skill, true);
  auditTarget.value = null;
  auditResult.value = null;
}

async function selectSource(id: string) {
  if (activeSource.value === id) return;
  activeSource.value = id;
  if (id === "skills-sh") activeCategory.value = "";
  await search(true);
}

async function selectCategory(key: string) {
  activeCategory.value = key;
  await search(true);
}

async function loadCategories() {
  try {
    categories.value = await listMarketCategories();
  } catch {
    categories.value = [];
  }
}

async function loadSources() {
  try {
    sources.value = await listMarketSources();
  } catch (error) {
    showToast(errorMessage(error));
  }
}

onMounted(async () => {
  await Promise.all([loadSources(), loadCategories()]);
  await search(true);
});
</script>

<template>
  <div class="page market-page">
    <div class="page-head">
      <div>
        <h1>市场 <span class="badge phase">M3</span></h1>
        <p>{{ total ? `${total} 个技能` : "从社区发现并入库" }} · SkillHub 已接入</p>
      </div>
      <div class="actions">
        <div class="search market-search">
          <Icon name="search" :size="14" />
          <input v-model="query" class="input" placeholder="搜索技能名称或描述…" @keydown.enter="search(true)" />
        </div>
        <button class="btn primary" :disabled="loading || unavailable" @click="search(true)">
          {{ loading ? "搜索中…" : "搜索" }}
        </button>
      </div>
    </div>

    <div class="market-tabs">
      <button class="chip" :class="{ active: activeSource === 'all' }" @click="selectSource('all')">全部</button>
      <button v-for="source in sources" :key="source.id" class="chip" :class="{ active: activeSource === source.id }" :title="source.note" @click="selectSource(source.id)">
        {{ source.name }}
        <span class="source-status" :class="source.status">{{ source.status === "available" ? "可用" : source.status === "limited" ? "受限" : "实验性" }}</span>
      </button>
      <div class="grow" />
      <select v-model="sortMode" class="input sort-select" title="排序">
        <option value="installs">按热度</option>
        <option value="name">按名称</option>
      </select>
    </div>

    <div v-if="activeSource !== 'skills-sh' && categories.length" class="market-categories">
      <button class="chip" :class="{ active: !activeCategory }" @click="selectCategory('')">全部分类</button>
      <button v-for="category in categories" :key="category.key" class="chip" :class="{ active: activeCategory === category.key }" @click="selectCategory(category.key)">
        {{ category.name }}
      </button>
    </div>

    <div v-if="active && active.note && unavailable" class="market-note">
      <Icon name="info" :size="13" />
      {{ active.name }}：{{ active.note }}
    </div>

    <div v-if="loading && !items.length" class="empty">
      <div class="empty-ico"><Icon name="market" :size="18" /></div>
      <h3>正在从市场拉取技能…</h3>
      <p>首次搜索需要访问 SkillHub API。</p>
    </div>

    <div v-else-if="!sorted.length" class="empty">
      <div class="empty-ico"><Icon name="search" :size="18" /></div>
      <h3>{{ unavailable ? "这个源暂不可用" : "没有找到技能" }}</h3>
      <p>{{ unavailable ? active?.note : "换个关键词，或稍后重试。" }}</p>
      <button v-if="!unavailable" class="btn" :disabled="loading" @click="search(true)">重试</button>
    </div>

    <div v-else class="market-grid">
      <article v-for="skill in sorted" :key="skill.id" class="market-card">
        <div class="mk-head">
          <span class="mk-icon">
            <img v-if="skill.iconUrl" :src="skill.iconUrl" alt="" loading="lazy" />
            <span v-else>{{ initials(skill.name) }}</span>
          </span>
          <div class="mk-title">
            <b>{{ skill.name }}</b>
            <span>{{ skill.author || "Unknown" }}<template v-if="skill.version"> · v{{ skill.version }}</template></span>
          </div>
          <span class="badge">{{ skill.sourceName }}</span>
        </div>
        <p class="mk-desc">{{ skill.description }}</p>
        <div class="mk-tags">
          <span v-for="tag in (skill.tags ?? []).slice(0, 3)" :key="tag" class="badge">{{ tag }}</span>
          <span class="grow" />
          <span class="mk-installs"><Icon name="download" :size="11" /> {{ formatInstalls(skill.installs) }}</span>
        </div>
        <div class="mk-foot">
          <span class="mono">{{ skill.slug }}</span>
          <button class="btn sm primary" :disabled="skill.installed || !!installing" @click="install(skill)">
            {{ skill.installed ? "已入库" : installing === skill.id ? "安装中…" : auditing === skill.id ? "检查中…" : "安装" }}
          </button>
        </div>
      </article>
    </div>

    <div v-if="items.length < total" class="market-more">
      <button class="btn" :disabled="loading" @click="loadMore">{{ loading ? "加载中…" : "加载更多" }}</button>
    </div>

    <div v-if="auditTarget && auditResult" class="audit-overlay" @click.self.stop>
      <div class="audit-card">
        <header class="audit-head">
          <div>
            <h3>安装前检查 · 高风险</h3>
            <p>{{ auditTarget.name }} 包含需要确认的脚本或命令。</p>
          </div>
          <button class="icon-btn" :disabled="!!installing" @click="cancelAudit"><Icon name="x" :size="15" /></button>
        </header>
        <div class="audit-body">
          <div class="badge warn">高风险</div>
          <ul>
            <li v-for="finding in (auditResult.findings ?? [])" :key="finding">{{ finding }}</li>
          </ul>
          <p v-if="!(auditResult.findings ?? []).length" class="muted">检查到脚本文件，未命中已知危险命令。</p>
        </div>
        <footer class="audit-foot">
          <button class="btn" :disabled="!!installing" @click="cancelAudit">取消</button>
          <button class="btn danger" :disabled="!!installing" @click="confirmInstall">{{ installing ? "安装中…" : "仍然安装" }}</button>
        </footer>
      </div>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.market-search { width: 260px; }
.market-tabs { display: flex; align-items: center; gap: 5px; flex-wrap: wrap; margin-bottom: 8px; }
.market-categories { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; margin-bottom: 10px; }
.source-status { font-size: 8px; color: var(--text-4); margin-left: 3px; }
.source-status.available { color: var(--success, #15803d); }
.source-status.limited, .source-status.experimental { color: var(--warning); }
.market-note { display: flex; align-items: center; gap: 6px; padding: 8px 10px; margin-bottom: 10px; border: 1px dashed var(--border); border-radius: 8px; color: var(--text-4); font-size: 10px; }
.market-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 9px; }
.market-card { display: flex; flex-direction: column; min-width: 0; padding: 11px 12px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--panel); box-shadow: var(--shadow-sm); }
.mk-head { display: flex; align-items: flex-start; gap: 8px; min-width: 0; }
.mk-icon { display: grid; place-items: center; width: 30px; height: 30px; border-radius: 8px; overflow: hidden; background: var(--accent-soft); color: var(--accent); font-size: 9px; font-weight: 700; flex: none; }
.mk-icon img { width: 100%; height: 100%; object-fit: cover; }
.mk-title { display: flex; flex-direction: column; gap: 3px; min-width: 0; flex: 1; }
.mk-title b { font-size: 12px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mk-title span { color: var(--text-4); font-size: 9px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mk-head > .badge { flex: none; }
.mk-desc { margin: 8px 0 7px; color: var(--text-3); font-size: 10px; line-height: 1.55; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; min-height: 31px; }
.mk-tags { display: flex; align-items: center; gap: 4px; min-height: 17px; }
.mk-tags .badge { font-size: 8.5px; padding: 1px 5px; }
.mk-installs { display: inline-flex; align-items: center; gap: 3px; color: var(--text-4); font-size: 9px; }
.mk-foot { display: flex; align-items: center; gap: 6px; margin-top: 9px; padding-top: 8px; border-top: 1px dashed var(--border-soft); }
.mk-foot .mono { color: var(--text-4); font-size: 8.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1; }
.market-more { display: flex; justify-content: center; padding: 14px; }
.audit-overlay { position: fixed; inset: 0; z-index: 200; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.audit-card { width: min(500px, calc(100vw - 32px)); max-height: min(600px, calc(100vh - 40px)); display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); overflow: hidden; }
.audit-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px 16px 10px; border-bottom: 1px solid var(--border-soft); }
.audit-head h3 { margin: 0; font-size: 14px; }
.audit-head p { margin: 4px 0 0; color: var(--text-4); font-size: 10px; }
.audit-head .icon-btn { margin-left: auto; }
.audit-body { padding: 12px 16px; overflow: auto; }
.audit-body ul { margin: 8px 0 0; padding-left: 18px; color: var(--text-3); font-size: 10px; line-height: 1.7; }
.audit-foot { display: flex; justify-content: flex-end; gap: 6px; padding: 10px 16px; border-top: 1px solid var(--border-soft); }
@media (max-width: 720px) { .market-search { width: 100%; } }
</style>
