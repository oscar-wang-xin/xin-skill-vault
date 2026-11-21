<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import { createBundle, deleteBundle, errorMessage, exportBundle, listBundles, listSkills, updateBundle } from "../lib/api";
import type { BundleItem, SkillItem } from "../../bindings/xinskillvault/internal/app/models";

const bundles = ref<BundleItem[]>([]);
const skills = ref<SkillItem[]>([]);
const editingId = ref<string | null>(null);
const modal = ref(false);
const nameDraft = ref("");
const descDraft = ref("");
const slugDraft = ref<string[]>([]);
const skillQuery = ref("");
const busy = ref(false);
const confirm = ref<BundleItem | null>(null);
const toast = ref("");
let toastTimer: number | undefined;

const filteredSkills = computed(() => {
  const q = skillQuery.value.trim().toLowerCase();
  return skills.value.filter((item) => !q || item.name.toLowerCase().includes(q) || item.slug.toLowerCase().includes(q));
});

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

function formatDate(seconds: number): string {
  return seconds ? new Date(seconds * 1000).toLocaleDateString("zh-CN") : "—";
}

function openCreate() {
  editingId.value = null;
  nameDraft.value = "";
  descDraft.value = "";
  slugDraft.value = [];
  skillQuery.value = "";
  modal.value = true;
}

function openEdit(bundle: BundleItem) {
  editingId.value = bundle.id;
  nameDraft.value = bundle.name;
  descDraft.value = bundle.description;
  slugDraft.value = [...(bundle.slugs ?? [])];
  skillQuery.value = "";
  modal.value = true;
}

function toggleSlug(slug: string) {
  slugDraft.value = slugDraft.value.includes(slug)
    ? slugDraft.value.filter((item) => item !== slug)
    : [...slugDraft.value, slug];
}

async function save() {
  if (busy.value) return;
  if (!nameDraft.value.trim()) {
    showToast("Bundle 名称不能为空");
    return;
  }
  busy.value = true;
  try {
    if (editingId.value) await updateBundle(editingId.value, nameDraft.value, descDraft.value, slugDraft.value);
    else await createBundle(nameDraft.value, descDraft.value, slugDraft.value);
    modal.value = false;
    await load();
    showToast(editingId.value ? "Bundle 已更新" : "Bundle 已创建");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

async function doExport(bundle: BundleItem) {
  try {
    const path = await exportBundle(bundle.id);
    if (path) showToast(`已导出：${path}`);
  } catch (error) {
    showToast(errorMessage(error));
  }
}

async function remove() {
  if (!confirm.value || busy.value) return;
  busy.value = true;
  try {
    await deleteBundle(confirm.value.id);
    confirm.value = null;
    await load();
    showToast("Bundle 已删除");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

async function load() {
  try {
    bundles.value = await listBundles();
  } catch (error) {
    showToast(errorMessage(error));
  }
}

onMounted(async () => {
  await Promise.all([load(), listSkills().then((items) => { skills.value = items; })]);
});
</script>

<template>
  <section class="bundles">
    <div class="bundle-head">
      <div>
        <h2>Bundle 包 <span class="badge">{{ bundles.length }}</span></h2>
        <p>把多个技能保存为一组，随时重复导出为可分发包。</p>
      </div>
      <button class="btn" @click="openCreate"><Icon name="plus" :size="12" /> 新建 Bundle</button>
    </div>

    <div v-if="!bundles.length" class="bundle-empty">还没有持久化 Bundle。也可以在上方勾选技能后直接临时打包。</div>
    <div v-else class="bundle-grid">
      <div v-for="bundle in bundles" :key="bundle.id" class="bundle-card">
        <div class="bundle-card-head">
          <span class="bundle-icon"><Icon name="layers" :size="13" /></span>
          <div class="bundle-name"><b>{{ bundle.name }}</b><span>{{ bundle.skillCount }} 个技能 · {{ formatDate(bundle.updatedAt) }}</span></div>
          <button class="combo-mini" title="编辑" @click="openEdit(bundle)"><Icon name="edit" :size="11" /></button>
          <button class="combo-mini danger" title="删除" @click="confirm = bundle"><Icon name="trash" :size="11" /></button>
        </div>
        <p>{{ bundle.description || "暂无简介" }}</p>
        <button class="btn sm" @click="doExport(bundle)"><Icon name="download" :size="11" /> 导出 Bundle</button>
      </div>
    </div>

    <div v-if="modal" class="combo-overlay" @click.self.stop>
      <div class="combo-modal">
        <header class="combo-modal-head">
          <div><h2>{{ editingId ? "编辑 Bundle" : "新建 Bundle" }}</h2><p>Bundle 只保存技能引用，导出时再读取最新目录。</p></div>
          <button class="icon-btn" @click="modal = false"><Icon name="x" :size="16" /></button>
        </header>
        <div class="combo-modal-body">
          <label class="field"><span>名称</span><input v-model="nameDraft" class="input" placeholder="例如：前端交付包" /></label>
          <label class="field"><span>简介</span><input v-model="descDraft" class="input" placeholder="这个 Bundle 用来做什么" /></label>
          <div class="field">
            <span>技能 <em>已选 {{ slugDraft.length }}</em></span>
            <div class="search" style="margin-bottom: 5px"><Icon name="search" :size="12" /><input v-model="skillQuery" class="input" placeholder="搜索技能…" /></div>
            <div class="combo-skill-list">
              <button v-for="skill in filteredSkills" :key="skill.slug" class="combo-skill" :class="{ on: slugDraft.includes(skill.slug) }" @click="toggleSlug(skill.slug)">
                <span class="mini-check" :class="{ on: slugDraft.includes(skill.slug) }"><Icon v-if="slugDraft.includes(skill.slug)" name="check" :size="10" /></span>
                <span class="combo-skill-main"><b>{{ skill.name }}</b><em>{{ skill.description }}</em></span>
                <span class="mono">{{ skill.version ? `v${skill.version}` : "" }}</span>
              </button>
            </div>
          </div>
        </div>
        <footer class="combo-modal-foot">
          <button class="btn" :disabled="busy" @click="modal = false">取消</button>
          <button class="btn primary" :disabled="busy" @click="save">{{ busy ? "保存中…" : "保存" }}</button>
        </footer>
      </div>
    </div>

    <div v-if="confirm" class="combo-overlay" @click.self.stop>
      <div class="combo-confirm">
        <h3>删除 Bundle</h3>
        <p>只会删除「{{ confirm.name }}」这个 Bundle，不会删除技能。</p>
        <div class="combo-confirm-foot">
          <button class="btn" :disabled="busy" @click="confirm = null">取消</button>
          <button class="btn danger" :disabled="busy" @click="remove">{{ busy ? "删除中…" : "删除" }}</button>
        </div>
      </div>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </section>
</template>

<style scoped>
.bundles { margin-top: 22px; padding-top: 16px; border-top: 1px solid var(--border-soft); }
.bundle-head { display: flex; align-items: flex-start; gap: 10px; margin-bottom: 9px; }
.bundle-head h2 { margin: 0; font-size: 14px; display: flex; gap: 6px; align-items: center; }
.bundle-head p { margin: 4px 0 0; color: var(--text-4); font-size: 9.5px; }
.bundle-head > .btn { margin-left: auto; }
.bundle-empty { padding: 18px; color: var(--text-4); font-size: 10px; border: 1px dashed var(--border); border-radius: 8px; text-align: center; }
.bundle-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 8px; }
.bundle-card { padding: 10px 11px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--panel); }
.bundle-card-head { display: flex; align-items: flex-start; gap: 6px; }
.bundle-icon { display: grid; place-items: center; width: 25px; height: 25px; border-radius: 6px; color: var(--accent); background: var(--accent-soft); flex: none; }
.bundle-name { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
.bundle-name b { font-size: 11px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bundle-name span { color: var(--text-4); font-size: 8.5px; }
.bundle-card > p { margin: 7px 0; color: var(--text-4); font-size: 9px; min-height: 25px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.combo-mini { color: var(--text-4); padding: 2px; border-radius: 4px; }
.combo-mini:hover { color: var(--accent); background: var(--accent-soft); }
.combo-mini.danger:hover { color: var(--danger); background: var(--danger-soft); }
.combo-overlay { position: fixed; inset: 0; z-index: 150; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.combo-modal { width: min(580px, calc(100vw - 32px)); max-height: min(700px, calc(100vh - 40px)); display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); overflow: hidden; }
.combo-modal-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px 16px 10px; border-bottom: 1px solid var(--border-soft); }
.combo-modal-head h2 { margin: 0; font-size: 15px; }
.combo-modal-head p { margin: 4px 0 0; color: var(--text-4); font-size: 10px; }
.combo-modal-head .icon-btn { margin-left: auto; }
.combo-modal-body { padding: 12px 16px; overflow: auto; display: flex; flex-direction: column; gap: 9px; }
.field { display: flex; flex-direction: column; gap: 4px; font-size: 10px; color: var(--text-3); }
.field em { color: var(--text-4); font-style: normal; font-size: 9px; margin-left: 4px; }
.combo-skill-list { max-height: 280px; overflow: auto; border: 1px solid var(--border-soft); border-radius: 7px; padding: 3px; }
.combo-skill { display: flex; align-items: center; gap: 7px; width: 100%; padding: 6px 5px; border-bottom: 1px dashed var(--border-soft); text-align: left; }
.combo-skill.on { background: var(--accent-soft); }
.combo-skill-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
.combo-skill-main b { font-size: 11px; font-weight: 550; }
.combo-skill-main em { color: var(--text-4); font-style: normal; font-size: 9px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mini-check { display: grid; place-items: center; width: 14px; height: 14px; border: 1px solid var(--border); border-radius: 4px; color: #fff; flex: none; }
.mini-check.on { background: var(--accent); border-color: var(--accent); }
.combo-modal-foot { display: flex; justify-content: flex-end; gap: 6px; padding: 10px 16px; border-top: 1px solid var(--border-soft); }
.combo-confirm { width: min(400px, calc(100vw - 32px)); padding: 16px; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); }
.combo-confirm h3 { margin: 0; font-size: 14px; }
.combo-confirm p { margin: 6px 0 14px; color: var(--text-3); font-size: 11px; line-height: 1.6; }
.combo-confirm-foot { display: flex; justify-content: flex-end; gap: 6px; }
</style>
