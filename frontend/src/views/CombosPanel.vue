<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import DistributeModal from "./DistributeModal.vue";
import { createCombo, deleteCombo, errorMessage, listComboSkills, listCombos, listSkills, updateCombo } from "../lib/api";
import type { ComboItem, SkillItem } from "../../bindings/xinskillvault/internal/app/models";

const emit = defineEmits<{ count: [value: number] }>();
const combos = ref<ComboItem[]>([]);
const skills = ref<SkillItem[]>([]);
const loading = ref(true);
const query = ref("");
const sortBy = ref<"created" | "name">("created");
const sortAsc = ref(false);
const toast = ref("");
let toastTimer: number | undefined;

const modal = ref(false);
const editingId = ref<string | null>(null);
const draftName = ref("");
const draftDescription = ref("");
const draftIcon = ref("");
const draftSlugs = ref<string[]>([]);
const skillQuery = ref("");
const busy = ref(false);
const confirm = ref<ComboItem | null>(null);
const distributeItems = ref<{ slug: string; name: string }[]>([]);

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  let list = combos.value.filter((item) => !q || item.name.toLowerCase().includes(q) || item.description.toLowerCase().includes(q));
  const direction = sortAsc.value ? 1 : -1;
  return list.slice().sort((a, b) => {
    const result = sortBy.value === "name" ? a.name.localeCompare(b.name, "zh-Hans-CN") : a.createdAt - b.createdAt;
    return result * direction;
  });
});

const filteredSkills = computed(() => {
  const q = skillQuery.value.trim().toLowerCase();
  return skills.value.filter((item) => !q || item.name.toLowerCase().includes(q) || item.slug.toLowerCase().includes(q));
});

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

function shortDesc(text: string): string {
  const value = (text || "暂无简介").trim();
  return value.length > 20 ? `${value.slice(0, 20)}…` : value;
}

function formatDate(seconds: number): string {
  if (!seconds) return "—";
  return new Date(seconds * 1000).toLocaleDateString("zh-CN");
}

function openCreate() {
  editingId.value = null;
  draftName.value = "";
  draftDescription.value = "";
  draftIcon.value = "";
  draftSlugs.value = [];
  skillQuery.value = "";
  modal.value = true;
}

function openEdit(combo: ComboItem) {
  editingId.value = combo.id;
  draftName.value = combo.name;
  draftDescription.value = combo.description;
  draftIcon.value = combo.icon;
  draftSlugs.value = [...(combo.slugs ?? [])];
  skillQuery.value = "";
  modal.value = true;
}

function toggleSlug(slug: string) {
  draftSlugs.value = draftSlugs.value.includes(slug)
    ? draftSlugs.value.filter((item) => item !== slug)
    : [...draftSlugs.value, slug];
}

async function save() {
  if (busy.value) return;
  if (!draftName.value.trim()) {
    showToast("组合名称不能为空");
    return;
  }
  busy.value = true;
  try {
    if (editingId.value) await updateCombo(editingId.value, draftName.value, draftDescription.value, draftIcon.value, draftSlugs.value);
    else await createCombo(draftName.value, draftDescription.value, draftIcon.value, draftSlugs.value);
    modal.value = false;
    await load();
    showToast(editingId.value ? "组合已更新" : "组合已创建");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

async function remove() {
  if (!confirm.value || busy.value) return;
  busy.value = true;
  try {
    await deleteCombo(confirm.value.id);
    confirm.value = null;
    await load();
    showToast("组合已删除");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

async function distribute(combo: ComboItem) {
  try {
    const list = await listComboSkills(combo.id);
    distributeItems.value = list.map((item) => ({ slug: item.slug, name: item.name }));
    if (!distributeItems.value.length) {
      showToast("组合里还没有技能");
      return;
    }
  } catch (error) {
    showToast(errorMessage(error));
  }
}

async function load() {
  try {
    combos.value = await listCombos();
    emit("count", combos.value.length);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await Promise.all([load(), listSkills().then((items) => { skills.value = items; })]);
});
</script>

<template>
  <div class="combos">
    <div class="combo-bar">
      <div class="search">
        <Icon name="search" :size="13" />
        <input v-model="query" class="input" placeholder="搜索组合…" />
      </div>
      <div class="grow" />
      <div class="seg">
        <button :class="{ active: sortBy === 'created' }" @click="sortBy = 'created'">创建时间</button>
        <button :class="{ active: sortBy === 'name' }" @click="sortBy = 'name'">名称</button>
      </div>
      <button class="btn sm" @click="sortAsc = !sortAsc">{{ sortAsc ? "升序" : "降序" }}</button>
      <button class="btn primary" @click="openCreate"><Icon name="plus" :size="13" /> 新建组合</button>
    </div>

    <div v-if="loading" class="empty-mini">正在读取组合…</div>
    <div v-else-if="!filtered.length" class="empty">
      <div class="empty-ico"><Icon name="layers" :size="18" /></div>
      <h3>{{ combos.length ? "没有匹配的组合" : "还没有组合" }}</h3>
      <p>{{ combos.length ? "换个关键词试试。" : "把常用技能编成组合，一次分发。" }}</p>
      <button class="btn" @click="openCreate">新建组合</button>
    </div>
    <div v-else class="combo-grid">
      <div v-for="combo in filtered" :key="combo.id" class="combo-card" @dblclick="openEdit(combo)">
        <div class="combo-head">
          <span class="combo-icon">{{ combo.icon || "✦" }}</span>
          <div class="combo-title">
            <b>{{ combo.name }}</b>
            <span>{{ combo.skillCount }} 个技能 · 创建于 {{ formatDate(combo.createdAt) }}</span>
          </div>
          <button class="combo-mini" title="编辑" @click.stop="openEdit(combo)"><Icon name="edit" :size="12" /></button>
          <button class="combo-mini danger" title="删除" @click.stop="confirm = combo"><Icon name="trash" :size="12" /></button>
        </div>
        <p class="combo-desc">{{ shortDesc(combo.description) }}</p>
        <button class="btn primary combo-dist" @click.stop="distribute(combo)">
          <Icon name="link" :size="12" /> 分发
        </button>
      </div>
    </div>

    <div v-if="modal" class="combo-overlay" @click.self.stop>
      <div class="combo-modal">
        <header class="combo-modal-head">
          <div><h2>{{ editingId ? "编辑组合" : "新建组合" }}</h2><p>组合只保存技能引用，不复制技能内容。</p></div>
          <button class="icon-btn" @click="modal = false"><Icon name="x" :size="16" /></button>
        </header>
        <div class="combo-modal-body">
          <div class="combo-form">
            <label class="field"><span>图标</span><input v-model="draftIcon" class="input" maxlength="4" placeholder="✦" /></label>
            <label class="field grow"><span>名称</span><input v-model="draftName" class="input" placeholder="例如：前端三件套" /></label>
          </div>
          <label class="field"><span>简介</span><input v-model="draftDescription" class="input" placeholder="一句话说明这个组合" /></label>
          <div class="field">
            <span>技能 <em>已选 {{ draftSlugs.length }}</em></span>
            <div class="search" style="margin-bottom: 6px"><Icon name="search" :size="12" /><input v-model="skillQuery" class="input" placeholder="搜索技能…" /></div>
            <div class="combo-skill-list">
              <button v-for="skill in filteredSkills" :key="skill.slug" class="combo-skill" :class="{ on: draftSlugs.includes(skill.slug) }" @click="toggleSlug(skill.slug)">
                <span class="mini-check" :class="{ on: draftSlugs.includes(skill.slug) }"><Icon v-if="draftSlugs.includes(skill.slug)" name="check" :size="10" /></span>
                <span class="combo-skill-main"><b>{{ skill.name }}</b><em>{{ skill.description }}</em></span>
                <span class="mono">{{ skill.version ? `v${skill.version}` : "" }}</span>
              </button>
              <div v-if="!filteredSkills.length" class="empty-mini">没有匹配的技能。</div>
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
        <h3>删除组合</h3>
        <p>只会删除「{{ confirm.name }}」这个组合，不会删除其中的技能。</p>
        <div class="combo-confirm-foot">
          <button class="btn" :disabled="busy" @click="confirm = null">取消</button>
          <button class="btn danger" :disabled="busy" @click="remove">{{ busy ? "删除中…" : "删除" }}</button>
        </div>
      </div>
    </div>

    <DistributeModal
      v-if="distributeItems.length"
      :items="distributeItems"
      @close="distributeItems = []"
      @changed="distributeItems = []"
    />

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.combo-bar { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-bottom: 10px; }
.combo-bar .search { width: 220px; }
.combo-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 9px; }
.combo-card { display: flex; flex-direction: column; padding: 12px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--panel); box-shadow: var(--shadow-sm); cursor: default; }
.combo-card:hover { border-color: var(--border-strong, var(--border)); }
.combo-head { display: flex; align-items: flex-start; gap: 7px; min-width: 0; }
.combo-icon { display: grid; place-items: center; width: 28px; height: 28px; border-radius: 8px; background: var(--accent-soft); color: var(--accent); font-size: 13px; flex: none; }
.combo-title { display: flex; flex-direction: column; gap: 3px; min-width: 0; flex: 1; }
.combo-title b { font-size: 12px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.combo-title span { color: var(--text-4); font-size: 8.5px; }
.combo-mini { color: var(--text-4); padding: 2px; border-radius: 4px; opacity: 0; }
.combo-card:hover .combo-mini { opacity: 1; }
.combo-mini:hover { color: var(--accent); background: var(--accent-soft); }
.combo-mini.danger:hover { color: var(--danger); background: var(--danger-soft); }
.combo-desc { margin: 8px 0 10px; color: var(--text-4); font-size: 9.5px; line-height: 1.55; min-height: 29px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.combo-dist { width: 100%; justify-content: center; }
.combo-overlay { position: fixed; inset: 0; z-index: 150; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.combo-modal { width: min(580px, calc(100vw - 32px)); max-height: min(700px, calc(100vh - 40px)); display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); overflow: hidden; }
.combo-modal-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px 16px 10px; border-bottom: 1px solid var(--border-soft); }
.combo-modal-head h2 { margin: 0; font-size: 15px; }
.combo-modal-head p { margin: 4px 0 0; color: var(--text-4); font-size: 10px; }
.combo-modal-head .icon-btn { margin-left: auto; }
.combo-modal-body { padding: 12px 16px; overflow: auto; display: flex; flex-direction: column; gap: 10px; }
.combo-form { display: flex; gap: 8px; }
.combo-form .field:first-child { width: 70px; flex: none; }
.combo-form .grow { flex: 1; }
.field { display: flex; flex-direction: column; gap: 4px; font-size: 10px; color: var(--text-3); }
.field em { color: var(--text-4); font-style: normal; font-size: 9px; margin-left: 4px; }
.combo-skill-list { max-height: 300px; overflow: auto; border: 1px solid var(--border-soft); border-radius: 7px; padding: 3px; }
.combo-skill { display: flex; align-items: center; gap: 7px; width: 100%; padding: 6px 5px; border-bottom: 1px dashed var(--border-soft); text-align: left; }
.combo-skill:last-child { border-bottom: 0; }
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
.empty-mini { padding: 18px; color: var(--text-4); font-size: 10px; text-align: center; }
</style>
