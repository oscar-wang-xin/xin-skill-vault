<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import {
  distributeSkill,
  errorMessage,
  listDistributionTargets,
  uninstallSkill,
} from "../lib/api";
import type { DistributionTarget } from "../../bindings/xinskillvault/internal/app/models";

const props = defineProps<{ slug: string; name: string }>();
const emit = defineEmits<{ close: []; changed: [] }>();

const targets = ref<DistributionTarget[]>([]);
const selected = ref<string[]>([]);
const mode = ref<"link" | "copy">("link");
const loading = ref(true);
const busy = ref(false);
const error = ref("");
const closeButton = ref<HTMLButtonElement | null>(null);

const detected = computed(() => targets.value.filter((target) => target.detected));
const installed = computed(() => detected.value.filter((target) => target.installed));
const available = computed(() => detected.value.filter((target) => !target.installed));
const selectedCount = computed(() => selected.value.length);
const allSelected = computed(() => available.value.length > 0 && available.value.every((target) => selected.value.includes(target.id)));

function toggle(id: string) {
  if (selected.value.includes(id)) selected.value = selected.value.filter((item) => item !== id);
  else selected.value = [...selected.value, id];
}

function toggleAll() {
  selected.value = allSelected.value ? [] : available.value.map((target) => target.id);
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    targets.value = await listDistributionTargets(props.slug);
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    loading.value = false;
  }
}

async function distribute() {
  if (!selectedCount.value || busy.value) return;
  busy.value = true;
  error.value = "";
  const failures: string[] = [];
  let succeeded = 0;
  for (const id of selected.value) {
    try {
      await distributeSkill(props.slug, id, mode.value);
      succeeded++;
    } catch (e) {
      const target = targets.value.find((item) => item.id === id);
      failures.push(`${target?.name ?? id}: ${errorMessage(e)}`);
    }
  }
  await load();
  selected.value = [];
  busy.value = false;
  if (succeeded > 0) emit("changed");
  if (failures.length) error.value = failures.join("；");
}

async function remove(target: DistributionTarget) {
  if (busy.value) return;
  busy.value = true;
  error.value = "";
  try {
    await uninstallSkill(props.slug, target.id);
    await load();
    emit("changed");
  } catch (e) {
    error.value = errorMessage(e);
  } finally {
    busy.value = false;
  }
}

function close() {
  if (!busy.value) emit("close");
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Escape" && !busy.value) close();
}

onMounted(() => {
  closeButton.value?.focus();
  window.addEventListener("keydown", onKeydown);
  load();
});
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
nextTick(() => closeButton.value?.focus());
</script>

<template>
  <div class="dist-overlay" @click.self.stop>
    <div class="dist-modal" role="dialog" aria-label="分发给 Agent">
      <header class="dist-head">
        <div>
          <h2>分发给 Agent</h2>
          <p>{{ name }} · 仓库版本会作为唯一来源</p>
        </div>
        <button ref="closeButton" class="icon-btn" title="关闭" @click="close"><Icon name="x" :size="16" /></button>
      </header>

      <div v-if="loading" class="dist-empty">正在读取 Agent 目录…</div>
      <div v-else class="dist-body">
        <div class="dist-toolbar">
          <div class="mode-switch">
            <button :class="{ active: mode === 'link' }" :disabled="busy" @click="mode = 'link'">
              <Icon name="link" :size="12" /> 链接
            </button>
            <button :class="{ active: mode === 'copy' }" :disabled="busy" @click="mode = 'copy'">
              <Icon name="file" :size="12" /> 复制
            </button>
          </div>
          <span class="muted dist-hint">{{ mode === "link" ? "优先创建链接，失败自动复制" : "复制完整目录，可独立修改" }}</span>
          <div class="grow" />
          <button class="btn sm" :disabled="!available.length || busy" @click="toggleAll">
            {{ allSelected ? "全不选" : "全选可分发" }}
          </button>
        </div>

        <div v-if="!detected.length" class="dist-empty">没有检测到可写的 Agent 目录。请先在 Agent 页查看检测状态。</div>
        <div v-else class="target-list">
          <div v-for="target in detected" :key="target.id" class="target-row" :class="{ installed: target.installed }">
            <button
              v-if="!target.installed"
              class="target-check"
              :class="{ on: selected.includes(target.id) }"
              :aria-label="`选择 ${target.name}`"
              @click="toggle(target.id)"
            >
              <Icon v-if="selected.includes(target.id)" name="check" :size="11" />
            </button>
            <span v-else class="target-check installed"><Icon name="check" :size="11" /></span>
            <span class="target-ava" :style="{ background: target.color }">{{ target.short }}</span>
            <div class="target-main">
              <div class="target-name">{{ target.name }}</div>
              <div class="target-path mono">{{ target.path }}</div>
            </div>
            <span v-if="target.installed" class="badge accent">已分发 · {{ target.installedMode === "link" ? "链接" : "复制" }}</span>
            <button v-if="target.installed" class="mini-danger" :disabled="busy" @click="remove(target)">移除</button>
          </div>
        </div>

        <div v-if="installed.length" class="dist-foot-note">
          已分发 {{ installed.length }} 个目标。移除只会删除 Agent 目录中的链接/复制件，不会删除仓库技能。
        </div>
      </div>

      <div v-if="error" class="dist-error">{{ error }}</div>

      <footer class="dist-foot">
        <span class="muted">已选 {{ selectedCount }} 个</span>
        <div class="grow" />
        <button class="btn" :disabled="busy" @click="close">取消</button>
        <button class="btn primary" :disabled="busy || !selectedCount" @click="distribute">
          {{ busy ? "分发中…" : "分发" }}
        </button>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.dist-overlay { position: fixed; inset: 0; z-index: 150; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.dist-modal { width: min(560px, calc(100vw - 32px)); max-height: min(640px, calc(100vh - 60px)); display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); overflow: hidden; }
.dist-head { display: flex; align-items: flex-start; gap: 12px; padding: 14px 16px 12px; border-bottom: 1px solid var(--border-soft); }
.dist-head h2 { margin: 0; font-size: 15px; }
.dist-head p { margin: 4px 0 0; color: var(--text-4); font-size: 10.5px; }
.dist-head .icon-btn { margin-left: auto; }
.dist-body { padding: 10px 16px 12px; overflow: auto; }
.dist-toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; padding-bottom: 8px; }
.mode-switch { display: flex; gap: 2px; padding: 2px; border: 1px solid var(--border); border-radius: 7px; background: var(--panel-2); }
.mode-switch button { display: inline-flex; align-items: center; gap: 4px; padding: 4px 8px; border-radius: 5px; color: var(--text-4); font-size: 10px; }
.mode-switch button.active { color: var(--accent); background: var(--accent-soft); }
.dist-hint { font-size: 9.5px; }
.target-list { display: flex; flex-direction: column; gap: 3px; margin-top: 3px; }
.target-row { display: flex; align-items: center; gap: 8px; padding: 7px 6px; border-radius: 7px; }
.target-row:hover { background: var(--panel-2); }
.target-row.installed { opacity: .82; }
.target-check { display: grid; place-items: center; width: 16px; height: 16px; border: 1px solid var(--border); border-radius: 4px; color: #fff; flex: none; }
.target-check.on { background: var(--accent); border-color: var(--accent); }
.target-check.installed { background: var(--accent-soft); border-color: var(--accent-soft); color: var(--accent); }
.target-ava { display: grid; place-items: center; width: 24px; height: 24px; border-radius: 6px; color: #fff; font-size: 8px; font-weight: 700; flex: none; }
.target-main { min-width: 0; flex: 1; }
.target-name { font-size: 11px; font-weight: 550; }
.target-path { margin-top: 2px; color: var(--text-4); font-size: 8.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mini-danger { color: var(--danger); font-size: 9.5px; padding: 2px 5px; border-radius: 4px; }
.mini-danger:hover { background: var(--danger-soft); }
.dist-empty { padding: 20px 10px; color: var(--text-4); font-size: 11px; text-align: center; }
.dist-foot-note { margin-top: 10px; color: var(--text-4); font-size: 9px; line-height: 1.6; }
.dist-error { margin: 0 16px 8px; padding: 7px 9px; border: 1px solid color-mix(in oklab, var(--danger) 25%, var(--border)); border-radius: 7px; color: var(--danger); font-size: 10px; }
.dist-foot { display: flex; align-items: center; gap: 6px; padding: 10px 16px; border-top: 1px solid var(--border-soft); }
</style>
