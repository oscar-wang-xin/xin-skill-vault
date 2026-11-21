<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import {
  createPrompt,
  createPromptCategory,
  deletePromptVersion,
  errorMessage,
  getPrompt,
  getPromptVersion,
  listPromptCategories,
  listPrompts,
  openPromptFolder,
  reorderPromptCategories,
  savePrompt,
  updatePromptCategory,
} from "../lib/api";
import type { PromptCategoryItem, PromptDetail, PromptItem, PromptVersionItem } from "../../bindings/xinskillvault/internal/app/models";

type SortKey = "updated" | "created" | "name" | "category";

const categories = ref<PromptCategoryItem[]>([]);
const prompts = ref<PromptItem[]>([]);
const selectedCategory = ref("all");
const query = ref("");
const sortBy = ref<SortKey>("updated");
const sortAsc = ref(false);
const loading = ref(true);
const toast = ref("");
let toastTimer: number | undefined;

const drawerOpen = ref(false);
const isNew = ref(false);
const promptId = ref<string | null>(null);
const draftTitle = ref("");
const draftCategory = ref("");
const draftTags = ref("");
const draftContent = ref("");
const versions = ref<PromptVersionItem[]>([]);
const selectedVersion = ref(1);
const saving = ref(false);
const dirty = ref(false);
const textarea = ref<HTMLTextAreaElement | null>(null);
const closeConfirm = ref(false);
const versionConfirm = ref<{ version: number } | null>(null);

const categoryModal = ref(false);
const categoryEditingID = ref<string | null>(null);
const categoryName = ref("");
const categoryParent = ref("");
const categoryDir = ref("");
const categoryBusy = ref(false);
const editorPercent = ref(Number(localStorage.getItem("xsv-prompt-editor-split")) || 66);
const pdBodyEl = ref<HTMLElement | null>(null);
let splitStartX = 0;
let splitStartPercent = 0;
let splitting = false;
const draggingCategory = ref("");
const dragOverCategory = ref("");

const categoryMap = computed(() => {
  const map = new Map<string, PromptCategoryItem>();
  for (const item of categories.value) map.set(item.id, item);
  return map;
});

function categoryDepth(id: string): number {
  let depth = 0;
  let current = categoryMap.value.get(id);
  while (current?.parentId) {
    depth += 1;
    current = categoryMap.value.get(current.parentId);
  }
  return depth;
}

const categoryRows = computed(() => {
  const byParent = new Map<string, PromptCategoryItem[]>();
  for (const item of categories.value) {
    const key = item.parentId || "";
    const list = byParent.get(key) ?? [];
    list.push(item);
    byParent.set(key, list);
  }
  for (const list of byParent.values()) {
    list.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name));
  }
  const out: PromptCategoryItem[] = [];
  const walk = (parentID: string) => {
    for (const item of byParent.get(parentID) ?? []) {
      out.push(item);
      walk(item.id);
    }
  };
  walk("");
  return out;
});

function startCategoryDrag(category: PromptCategoryItem) {
  draggingCategory.value = category.id;
}

function overCategory(category: PromptCategoryItem, event: DragEvent) {
  const source = categories.value.find((item) => item.id === draggingCategory.value);
  if (!source || source.id === category.id || (source.parentId || "") !== (category.parentId || "")) return;
  event.preventDefault();
  dragOverCategory.value = category.id;
}

async function dropCategory(target: PromptCategoryItem) {
  const source = categories.value.find((item) => item.id === draggingCategory.value);
  dragOverCategory.value = "";
  draggingCategory.value = "";
  if (!source || source.id === target.id || (source.parentId || "") !== (target.parentId || "")) return;
  const parentID = source.parentId || "";
  const siblings = categoryRows.value.filter((item) => (item.parentId || "") === parentID).map((item) => item.id);
  const ids = siblings.filter((id) => id !== source.id);
  const targetIndex = ids.indexOf(target.id);
  ids.splice(targetIndex < 0 ? ids.length : targetIndex, 0, source.id);
  try {
    await reorderPromptCategories(parentID, ids);
    await loadCategories();
    showToast("分类顺序已更新");
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function endCategoryDrag() {
  draggingCategory.value = "";
  dragOverCategory.value = "";
}

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  let list = prompts.value.filter((item) => {
    if (selectedCategory.value === "none" && item.categoryId) return false;
    if (selectedCategory.value !== "all" && selectedCategory.value !== "none" && item.categoryId !== selectedCategory.value) return false;
    if (!q) return true;
    return item.title.toLowerCase().includes(q) || (item.tags ?? []).some((tag) => tag.toLowerCase().includes(q));
  });
  const direction = sortAsc.value ? 1 : -1;
  return list.slice().sort((a, b) => {
    let result = 0;
    if (sortBy.value === "name") result = a.title.localeCompare(b.title, "zh-Hans-CN");
    else if (sortBy.value === "created") result = a.createdAt - b.createdAt;
    else if (sortBy.value === "category") result = a.categoryName.localeCompare(b.categoryName, "zh-Hans-CN");
    else result = a.updatedAt - b.updatedAt;
    if (!result) result = a.title.localeCompare(b.title, "zh-Hans-CN");
    return result * direction;
  });
});

const selectedCategoryName = computed(() => {
  if (selectedCategory.value === "all") return "全部";
  if (selectedCategory.value === "none") return "未分类";
  return categoryMap.value.get(selectedCategory.value)?.name ?? "分类";
});

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2600);
}

function formatDate(seconds: number): string {
  if (!seconds) return "—";
  return new Date(seconds * 1000).toLocaleString("zh-CN", { hour12: false });
}

function tagList(value: string): string[] {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

function formatTags(tags: string[]): string {
  return (tags ?? []).join(", ");
}

async function loadCategories() {
  categories.value = await listPromptCategories();
}

async function loadPrompts() {
  prompts.value = await listPrompts(selectedCategory.value);
}

async function refresh() {
  await Promise.all([loadCategories(), loadPrompts()]);
}

async function selectCategory(id: string) {
  selectedCategory.value = id;
  await loadPrompts();
}

async function openNew() {
  isNew.value = true;
  promptId.value = null;
  draftTitle.value = "";
  draftCategory.value = selectedCategory.value !== "all" && selectedCategory.value !== "none" ? selectedCategory.value : "";
  draftTags.value = "";
  draftContent.value = "";
  versions.value = [];
  selectedVersion.value = 1;
  dirty.value = false;
  drawerOpen.value = true;
  await nextTick();
  textarea.value?.focus();
}

async function openPrompt(item: PromptItem) {
  try {
    const detail = await getPrompt(item.id);
    isNew.value = false;
    promptId.value = detail.id;
    draftTitle.value = detail.title;
    draftCategory.value = detail.categoryId;
    draftTags.value = formatTags(detail.tags ?? []);
    draftContent.value = detail.content;
    versions.value = detail.versions ?? [];
    selectedVersion.value = detail.currentVersion;
    dirty.value = false;
    drawerOpen.value = true;
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function markDirty() {
  dirty.value = true;
}

async function save() {
  if (!draftTitle.value.trim()) {
    showToast("提示词标题不能为空");
    return;
  }
  if (saving.value) return;
  saving.value = true;
  try {
    let detail: PromptDetail;
    if (isNew.value) {
      detail = await createPrompt(draftTitle.value, draftCategory.value, tagList(draftTags.value), draftContent.value);
      isNew.value = false;
      promptId.value = detail.id;
    } else if (promptId.value) {
      detail = await savePrompt(promptId.value, draftTitle.value, draftCategory.value, tagList(draftTags.value), draftContent.value);
    } else {
      return;
    }
    versions.value = detail.versions ?? [];
    selectedVersion.value = detail.currentVersion;
    dirty.value = false;
    await refresh();
    showToast("已保存新版本");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    saving.value = false;
  }
}

function close() {
  if (dirty.value) {
    closeConfirm.value = true;
    return;
  }
  drawerOpen.value = false;
}

function forceClose() {
  closeConfirm.value = false;
  drawerOpen.value = false;
}

async function loadVersion(version: number) {
  if (!promptId.value || version === selectedVersion.value) return;
  if (dirty.value) {
    versionConfirm.value = { version };
    return;
  }
  await applyVersion(version);
}

async function applyVersion(version: number) {
  if (!promptId.value) return;
  try {
    const content = await getPromptVersion(promptId.value, version);
    const meta = versions.value.find((item) => item.version === version);
    draftContent.value = String(content);
    if (meta) {
      draftTitle.value = meta.title;
      draftCategory.value = meta.categoryId;
      draftTags.value = formatTags(meta.tags ?? []);
    }
    selectedVersion.value = version;
    dirty.value = false;
    versionConfirm.value = null;
  } catch (error) {
    showToast(errorMessage(error));
  }
}

async function removeVersion(version: number) {
  if (!promptId.value) return;
  if (!window.confirm(`删除版本 v${version}？该操作不可撤销。`)) return;
  try {
    const detail = await deletePromptVersion(promptId.value, version);
    versions.value = detail.versions ?? [];
    selectedVersion.value = detail.currentVersion;
    draftContent.value = detail.content;
    draftTitle.value = detail.title;
    draftCategory.value = detail.categoryId;
    draftTags.value = formatTags(detail.tags ?? []);
    dirty.value = false;
    await refresh();
    showToast("版本已删除");
  } catch (error) {
    showToast(errorMessage(error));
  }
}

async function copyContent() {
  try {
    await navigator.clipboard.writeText(draftContent.value);
    showToast("正文已复制");
  } catch {
    showToast("复制失败，请手动选择");
  }
}

function insert(before: string, after = "") {
  const element = textarea.value;
  if (!element) return;
  const start = element.selectionStart;
  const end = element.selectionEnd;
  draftContent.value = draftContent.value.slice(0, start) + before + draftContent.value.slice(start, end) + after + draftContent.value.slice(end);
  markDirty();
  nextTick(() => {
    element.focus();
    element.setSelectionRange(start + before.length, end + before.length);
  });
}

function openCategory(item?: PromptCategoryItem) {
  categoryEditingID.value = item?.id ?? null;
  categoryName.value = item?.name ?? "";
  categoryParent.value = item?.parentId ?? "";
  categoryDir.value = item?.dirPath ?? "";
  categoryModal.value = true;
}

async function saveCategory() {
  if (!categoryName.value.trim() || categoryBusy.value) return;
  categoryBusy.value = true;
  try {
    if (categoryEditingID.value) {
      await updatePromptCategory(categoryEditingID.value, categoryName.value);
    } else {
      await createPromptCategory(categoryName.value, categoryParent.value, categoryDir.value);
    }
    categoryModal.value = false;
    await refresh();
    showToast(categoryEditingID.value ? "分类已重命名" : "分类已创建");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    categoryBusy.value = false;
  }
}

async function openFolder() {
  try {
    await openPromptFolder();
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function startSplit(event: PointerEvent) {
  if (!pdBodyEl.value) return;
  splitting = true;
  splitStartX = event.clientX;
  splitStartPercent = editorPercent.value;
  window.addEventListener("pointermove", moveSplit);
  window.addEventListener("pointerup", endSplit);
  event.preventDefault();
}

function moveSplit(event: PointerEvent) {
  if (!splitting || !pdBodyEl.value) return;
  const width = pdBodyEl.value.getBoundingClientRect().width;
  if (!width) return;
  const delta = ((event.clientX - splitStartX) / width) * 100;
  editorPercent.value = Math.max(45, Math.min(80, Math.round(splitStartPercent + delta)));
}

function endSplit() {
  if (!splitting) return;
  splitting = false;
  localStorage.setItem("xsv-prompt-editor-split", String(editorPercent.value));
  window.removeEventListener("pointermove", moveSplit);
  window.removeEventListener("pointerup", endSplit);
}

onUnmounted(() => {
  window.removeEventListener("pointermove", moveSplit);
  window.removeEventListener("pointerup", endSplit);
});

onMounted(async () => {
  try {
    await refresh();
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="page prompts-page">
    <div class="page-head">
      <div>
        <h1>提示词 <span class="badge phase">M4</span></h1>
        <p>{{ prompts.length }} 条提示词 · 每个版本独立 `.md` 文件</p>
      </div>
      <div class="actions">
        <button class="btn" @click="openFolder"><Icon name="folder" :size="13" /> 打开目录</button>
        <button class="btn primary" @click="openNew"><Icon name="plus" :size="13" /> 新建提示词</button>
      </div>
    </div>

    <div class="prompt-layout">
      <aside class="prompt-side">
        <div class="cat-head">
          <span>分类</span>
          <button class="cat-add" title="新建分类" @click="openCategory()"><Icon name="plus" :size="12" /></button>
        </div>
        <button class="cat-row all" :class="{ active: selectedCategory === 'all' }" @click="selectCategory('all')">全部</button>
        <button class="cat-row none" :class="{ active: selectedCategory === 'none' }" @click="selectCategory('none')">未分类</button>
        <div class="cat-sep" />
        <button
          v-for="category in categoryRows"
          :key="category.id"
          class="cat-row"
          :class="{ active: selectedCategory === category.id, 'drag-over': dragOverCategory === category.id }"
          :style="{ paddingLeft: `${10 + categoryDepth(category.id) * 14}px` }"
          draggable="true"
          title="拖动同级排序；双击重命名"
          @dragstart="startCategoryDrag(category)"
          @dragover="overCategory(category, $event)"
          @drop="dropCategory(category)"
          @dragend="endCategoryDrag"
          @dblclick="openCategory(category)"
          @click="selectCategory(category.id)"
        >
          <span class="cat-grip">⋮</span>
          <span v-if="category.parentId" class="cat-branch">└</span>
          <span class="cat-name">{{ category.name }}</span>
          <span class="cat-path mono">{{ category.dirPath }}</span>
        </button>
        <div v-if="!categories.length" class="cat-empty">还没有分类，点击 + 新建。</div>
      </aside>

      <section class="prompt-main">
        <div class="prompt-toolbar">
          <div class="search">
            <Icon name="search" :size="13" />
            <input v-model="query" class="input" placeholder="搜索标题或标签…" />
          </div>
          <div class="grow" />
          <select v-model="sortBy" class="input sort-select">
            <option value="updated">更新时间</option>
            <option value="created">创建时间</option>
            <option value="name">名称</option>
            <option value="category">分类</option>
          </select>
          <button class="btn sm" @click="sortAsc = !sortAsc">{{ sortAsc ? "升序" : "降序" }}</button>
        </div>

        <div class="prompt-section-label">{{ selectedCategoryName }} <span>{{ filtered.length }}</span></div>

        <div v-if="loading" class="empty-mini">正在读取提示词…</div>
        <div v-else-if="!filtered.length" class="empty">
          <div class="empty-ico"><Icon name="prompts" :size="18" /></div>
          <h3>没有提示词</h3>
          <p>新建一条提示词，每次保存都会生成一个独立版本。</p>
          <button class="btn" @click="openNew">新建提示词</button>
        </div>
        <div v-else class="prompt-grid">
          <button v-for="item in filtered" :key="item.id" class="prompt-card" @click="openPrompt(item)">
            <div class="pc-head">
              <b>{{ item.title }}</b>
              <span class="badge">{{ item.categoryName || "未分类" }}</span>
            </div>
            <div class="pc-meta">
              <span class="mono">v{{ item.currentVersion }}</span>
              <span>{{ formatDate(item.updatedAt) }}</span>
            </div>
            <div class="pc-tags">
              <span v-for="tag in (item.tags ?? []).slice(0, 4)" :key="tag" class="badge">{{ tag }}</span>
            </div>
          </button>
        </div>
      </section>
    </div>

    <!-- editor drawer -->
    <div v-if="drawerOpen" class="prompt-overlay">
      <div class="prompt-drawer" role="dialog" aria-label="提示词编辑器">
        <header class="pd-head">
          <div>
            <h2>{{ isNew ? "新建提示词" : "编辑提示词" }}</h2>
            <span v-if="!isNew" class="mono">v{{ selectedVersion }} · {{ versions.length }} 个版本</span>
          </div>
          <div class="pd-actions">
            <button class="btn sm" @click="copyContent"><Icon name="copy" :size="12" /> 复制</button>
            <button class="icon-btn" title="关闭" @click="close"><Icon name="x" :size="16" /></button>
          </div>
        </header>

        <div ref="pdBodyEl" class="pd-body" :style="{ gridTemplateColumns: `minmax(0, ${editorPercent}%) 7px minmax(220px, 1fr)` }">
          <div class="pd-editor">
            <div class="pd-form">
              <label class="field"><span>标题</span><input v-model="draftTitle" class="input" placeholder="提示词标题" @input="markDirty" /></label>
              <div class="field-row">
                <label class="field"><span>分类</span>
                  <select v-model="draftCategory" class="input" @change="markDirty">
                    <option value="">未分类</option>
                    <option v-for="category in categoryRows" :key="category.id" :value="category.id">{{ category.name }}</option>
                  </select>
                </label>
                <label class="field"><span>标签 <em>逗号分隔</em></span><input v-model="draftTags" class="input" placeholder="写作, 结构" @input="markDirty" /></label>
              </div>
              <div class="md-toolbar">
                <button title="加粗" @click="insert('**', '**')"><b>B</b></button>
                <button title="斜体" @click="insert('*', '*')"><i>I</i></button>
                <button title="标题" @click="insert('## ')">H</button>
                <button title="列表" @click="insert('- ')">≡</button>
                <button title="行内代码" @click="insert('`', '`')"><span class="mono">`</span></button>
                <button title="代码块" @click="insert('```\n', '\n```')">▤</button>
                <button title="链接" @click="insert('[', '](https://)')">↗</button>
              </div>
              <textarea ref="textarea" v-model="draftContent" class="prompt-editor" spellcheck="false" placeholder="写点什么…" @input="markDirty" />
              <div class="pd-save">
                <span v-if="dirty" class="dirty-dot">● 未保存</span>
                <button class="btn primary" :disabled="saving" @click="save">{{ saving ? "保存中…" : "保存新版本" }}</button>
              </div>
            </div>
          </div>
          <div class="pd-resizer" title="拖动调整编辑区宽度" @pointerdown="startSplit" />
          <aside class="pd-versions">
            <div class="pv-head"><b>版本历史</b><span>{{ versions.length }}</span></div>
            <div v-if="!versions.length" class="pv-empty">保存后生成第一个版本。</div>
            <div
              v-for="version in versions"
              :key="version.version"
              class="pv-row"
              role="button"
              tabindex="0"
              :class="{ active: version.version === selectedVersion }"
              @click="loadVersion(version.version)"
              @keydown.enter="loadVersion(version.version)"
            >
              <span class="pv-version">v{{ version.version }}</span>
              <span class="pv-main"><b>{{ version.title || "未命名" }}</b><em>{{ version.note || "保存版本" }}</em></span>
              <span class="pv-time">{{ formatDate(version.createdAt) }}</span>
              <button class="pv-del" title="删除版本" :disabled="versions.length <= 1" @click.stop="removeVersion(version.version)"><Icon name="trash" :size="11" /></button>
            </div>
            <div class="pv-hint">点击版本可载入正文；删除当前版本会自动切到最新剩余版本。</div>
          </aside>
        </div>
      </div>
    </div>

    <!-- close confirm -->
    <div v-if="closeConfirm" class="prompt-confirm" @click.self.stop>
      <div class="prompt-confirm-card">
        <h3>未保存的修改</h3>
        <p>关闭后当前草稿会丢失，确定关闭？</p>
        <div class="prompt-confirm-foot">
          <button class="btn" @click="closeConfirm = false">继续编辑</button>
          <button class="btn danger" @click="forceClose">丢弃并关闭</button>
        </div>
      </div>
    </div>

    <!-- version switch confirm -->
    <div v-if="versionConfirm" class="prompt-confirm" @click.self.stop>
      <div class="prompt-confirm-card">
        <h3>切换版本</h3>
        <p>当前草稿尚未保存，切换版本会丢失修改。继续？</p>
        <div class="prompt-confirm-foot">
          <button class="btn" @click="versionConfirm = null">继续编辑</button>
          <button class="btn primary" @click="applyVersion(versionConfirm.version)">切换版本</button>
        </div>
      </div>
    </div>

    <!-- category modal -->
    <div v-if="categoryModal" class="prompt-confirm" @click.self.stop>
      <div class="prompt-confirm-card category-card">
        <h3>{{ categoryEditingID ? "重命名分类" : "新建分类" }}</h3>
        <label class="field"><span>名称</span><input v-model="categoryName" class="input" placeholder="例如：营销" autofocus /></label>
        <label v-if="!categoryEditingID" class="field"><span>父分类</span>
          <select v-model="categoryParent" class="input">
            <option value="">无（根分类）</option>
            <option v-for="category in categoryRows" :key="category.id" :value="category.id">{{ category.name }}</option>
          </select>
        </label>
        <label v-if="!categoryEditingID" class="field"><span>保存目录 <em>相对提示词根目录，可留空</em></span><input v-model="categoryDir" class="input mono" placeholder="留空自动按名称创建" /></label>
        <div class="prompt-confirm-foot">
          <button class="btn" :disabled="categoryBusy" @click="categoryModal = false">取消</button>
          <button class="btn primary" :disabled="categoryBusy" @click="saveCategory">{{ categoryBusy ? "保存中…" : "保存" }}</button>
        </div>
      </div>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.prompt-layout { display: grid; grid-template-columns: 220px minmax(0, 1fr); gap: 14px; align-items: start; min-height: 360px; }
.prompt-side { border-right: 1px solid var(--border-soft); padding-right: 10px; min-width: 0; }
.cat-head { display: flex; align-items: center; justify-content: space-between; padding: 2px 6px 6px; color: var(--text-4); font-size: 9px; }
.cat-add { color: var(--text-4); padding: 2px; border-radius: 4px; }
.cat-add:hover { color: var(--accent); background: var(--accent-soft); }
.cat-row { display: flex; flex-wrap: wrap; align-items: baseline; gap: 5px; width: 100%; padding: 6px 8px; border-radius: 6px; text-align: left; color: var(--text-2); font-size: 10.5px; }
.cat-row:hover { background: var(--panel-2); }
.cat-row.active { color: var(--accent); background: var(--accent-soft); }
.cat-row.all { font-weight: 600; }
.cat-row.none { color: var(--text-4); }
.cat-grip { color: var(--text-4); font-size: 8px; opacity: .5; margin-right: -2px; }
.cat-row.drag-over { box-shadow: inset 0 2px 0 var(--accent); }
.cat-name { flex: none; }
.cat-path { flex-basis: 100%; color: var(--text-4); font-size: 8.5px; font-weight: 400; padding-left: 0; opacity: .75; }
.cat-branch { color: var(--text-4); font-size: 9px; }
.cat-sep { border-top: 1px dashed var(--border-soft); margin: 5px 4px; }
.cat-empty { padding: 12px 6px; color: var(--text-4); font-size: 9.5px; }
.prompt-main { min-width: 0; }
.prompt-toolbar { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-bottom: 8px; }
.prompt-toolbar .search { width: 230px; }
.prompt-section-label { display: flex; gap: 6px; align-items: baseline; padding: 5px 2px 8px; color: var(--text-3); font-size: 11px; font-weight: 550; border-bottom: 1px solid var(--border-soft); margin-bottom: 8px; }
.prompt-section-label span { color: var(--text-4); font-size: 9px; font-weight: 400; }
.prompt-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(235px, 1fr)); gap: 8px; }
.prompt-card { display: flex; flex-direction: column; padding: 11px 12px; border: 1px solid var(--border); border-radius: var(--radius); background: var(--panel); text-align: left; }
.prompt-card:hover { border-color: var(--accent); }
.pc-head { display: flex; align-items: center; gap: 6px; }
.pc-head b { flex: 1; min-width: 0; font-size: 12px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pc-meta { display: flex; justify-content: space-between; gap: 6px; margin-top: 8px; color: var(--text-4); font-size: 8.5px; }
.pc-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 6px; min-height: 15px; }
.pc-tags .badge { font-size: 8px; padding: 1px 5px; }
.prompt-overlay { position: fixed; inset: 0; z-index: 100; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); display: flex; justify-content: flex-end; }
.prompt-drawer { width: 80vw; min-width: 620px; max-width: 1280px; height: 100%; background: var(--panel); border-left: 1px solid var(--border); box-shadow: var(--shadow-lg); display: flex; flex-direction: column; }
.pd-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px 18px 11px; border-bottom: 1px solid var(--border-soft); flex: none; }
.pd-head h2 { margin: 0; font-size: 15px; }
.pd-head span { color: var(--text-4); font-size: 9px; }
.pd-actions { margin-left: auto; display: flex; align-items: center; gap: 5px; }
.pd-body { display: grid; grid-template-columns: minmax(0, 2fr) 7px minmax(220px, 1fr); min-height: 0; flex: 1; }
.pd-resizer { cursor: col-resize; background: transparent; }
.pd-resizer:hover { background: var(--accent-soft); }
.pd-editor { padding: 12px 16px 14px; overflow: auto; min-width: 0; }
.pd-form { display: flex; flex-direction: column; gap: 8px; height: 100%; }
.field { display: flex; flex-direction: column; gap: 4px; font-size: 10px; color: var(--text-3); }
.field em { color: var(--text-4); font-style: normal; font-size: 9px; margin-left: 4px; }
.field-row { display: flex; gap: 8px; }
.field-row .field { flex: 1; min-width: 0; }
.md-toolbar { display: flex; gap: 2px; padding: 3px; border: 1px solid var(--border); border-radius: 7px; background: var(--panel-2); width: fit-content; }
.md-toolbar button { min-width: 25px; height: 23px; padding: 0 5px; border-radius: 4px; color: var(--text-3); font-size: 10px; }
.md-toolbar button:hover { background: var(--accent-soft); color: var(--accent); }
.prompt-editor { flex: 1; min-height: 300px; resize: vertical; border: 1px solid var(--border); border-radius: 8px; background: var(--panel-2); color: var(--text); padding: 10px 12px; font-family: "Cascadia Code", Consolas, monospace; font-size: 11px; line-height: 1.7; outline: none; }
.prompt-editor:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-soft); }
.pd-save { display: flex; align-items: center; justify-content: flex-end; gap: 8px; }
.dirty-dot { color: var(--warning); font-size: 9px; margin-right: auto; }
.pd-versions { border-left: 1px solid var(--border-soft); background: var(--panel-2); overflow: auto; padding: 11px 9px; min-width: 0; }
.pv-head { display: flex; justify-content: space-between; color: var(--text-3); font-size: 10px; padding: 2px 4px 8px; }
.pv-head span { color: var(--text-4); }
.pv-row { display: grid; grid-template-columns: 32px minmax(0, 1fr) auto 16px; gap: 5px; align-items: center; width: 100%; padding: 7px 4px; border-bottom: 1px dashed var(--border-soft); text-align: left; }
.pv-row.active { background: var(--accent-soft); border-radius: 5px; }
.pv-version { color: var(--accent); font-size: 9px; font-weight: 600; }
.pv-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.pv-main b { font-size: 10px; font-weight: 550; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.pv-main em { color: var(--text-4); font-style: normal; font-size: 8.5px; }
.pv-time { color: var(--text-4); font-size: 8px; white-space: nowrap; }
.pv-del { color: var(--text-4); padding: 1px; border-radius: 3px; }
.pv-del:hover { color: var(--danger); background: var(--danger-soft); }
.pv-hint { padding: 9px 4px; color: var(--text-4); font-size: 8.5px; line-height: 1.6; }
.pv-empty { padding: 14px 4px; color: var(--text-4); font-size: 9.5px; }
.prompt-confirm { position: fixed; inset: 0; z-index: 200; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.prompt-confirm-card { width: min(400px, calc(100vw - 32px)); padding: 16px; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); display: flex; flex-direction: column; gap: 9px; }
.prompt-confirm-card h3 { margin: 0; font-size: 14px; }
.prompt-confirm-card p { margin: 0; color: var(--text-3); font-size: 11px; line-height: 1.6; }
.prompt-confirm-foot { display: flex; justify-content: flex-end; gap: 6px; margin-top: 4px; }
.category-card { gap: 10px; }
.empty-mini { padding: 24px; text-align: center; color: var(--text-4); font-size: 10px; }
@media (max-width: 900px) { .prompt-layout { grid-template-columns: 180px minmax(0, 1fr); } .prompt-drawer { width: 94vw; min-width: 0; } .pd-body { grid-template-columns: 1fr !important; } .pd-resizer { display: none; } .pd-versions { border-left: 0; border-top: 1px solid var(--border-soft); max-height: 250px; } }
</style>
