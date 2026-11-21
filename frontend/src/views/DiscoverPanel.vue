<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import { errorMessage, formatBytes, importScannedSkill, scanAgents, updateScannedSkill } from "../lib/api";
import type { Finding, Result as ScanResult } from "../../bindings/xinskillvault/internal/scan/models";
import { useVaultStore } from "../stores/vault";

const emit = defineEmits<{ imported: [] }>();
const vault = useVaultStore();
const result = ref<ScanResult | null>(null);
const scanning = ref(false);
const importing = ref("");
const updating = ref("");
const filter = ref<"all" | "new" | "changed" | "ignored">("all");
const toast = ref("");
let toastTimer: number | undefined;

const findings = computed(() => result.value?.findings ?? []);
const filtered = computed(() => findings.value.filter((item) => filter.value === "all" || item.status === filter.value));
const counts = computed(() => ({
  all: findings.value.length,
  new: findings.value.filter((item) => item.status === "new").length,
  changed: findings.value.filter((item) => item.status === "changed").length,
  ignored: 0,
}));

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

async function scan() {
  if (scanning.value) return;
  scanning.value = true;
  try {
    result.value = await scanAgents();
    filter.value = "all";
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    scanning.value = false;
  }
}

async function importFinding(finding: Finding) {
  if (importing.value || updating.value || finding.status !== "new") return;
  importing.value = finding.key;
  try {
    const item = await importScannedSkill(finding.agentId, finding.path);
    await vault.refresh();
    await scan();
    emit("imported");
    showToast(`已导入：${item.name}`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    importing.value = "";
  }
}

async function updateFinding(finding: Finding) {
  if (importing.value || updating.value || finding.status !== "changed") return;
  updating.value = finding.key;
  try {
    const item = await updateScannedSkill(finding.agentId, finding.path);
    await vault.refresh();
    await scan();
    emit("imported");
    showToast(`已更新：${item.name}（旧版本已写入快照）`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    updating.value = "";
  }
}

function formatTime(seconds: number): string {
  if (!seconds) return "尚未扫描";
  return new Date(seconds * 1000).toLocaleString("zh-CN", { hour12: false });
}

onMounted(scan);
</script>

<template>
  <div class="discover">
    <div class="discover-bar">
      <span class="muted" style="font-size: 10.5px">
        上次扫描 {{ formatTime(result?.scannedAt ?? 0) }} ·
        {{ result?.agents?.filter((a) => a.detected).length ?? 0 }} 个目标 ·
        {{ result?.scannedSkills ?? 0 }} 个技能 · {{ findings.length }} 项待处理
      </span>
      <div class="grow" />
      <button class="btn" :disabled="scanning" @click="scan">
        <Icon name="refresh" :size="14" :class="{ spin: scanning }" />
        {{ scanning ? "扫描中…" : "扫描本机" }}
      </button>
    </div>

    <div class="discover-bar filters-bar">
      <div class="filters">
        <button
          v-for="item in ([['all','全部'],['new','未导入'],['changed','有变更'],['ignored','忽略']] as const)"
          :key="item[0]"
          class="chip"
          :class="{ active: filter === item[0] }"
          @click="filter = item[0]"
        >
          {{ item[1] }} <span class="chip-count">{{ counts[item[0]] }}</span>
        </button>
      </div>
      <div class="grow" />
      <span class="muted" style="font-size: 10px">{{ filtered.length }} 个结果</span>
    </div>

    <div v-if="scanning && !result" class="empty">
      <div class="empty-ico"><Icon name="scan" :size="18" /></div>
      <h3>正在扫描本机 Agent 目录…</h3>
      <p>只读取已检测到的目标目录，每个技能最多扫描一层子目录。</p>
    </div>

    <div v-else-if="!filtered.length" class="empty">
      <div class="empty-ico"><Icon name="search" :size="18" /></div>
      <h3>{{ findings.length ? "这个分类没有结果" : "没有待处理的技能" }}</h3>
      <p>{{ findings.length ? "换个分类查看。" : "检测到的 Agent 技能要么已导入且内容一致，要么还没有技能目录。" }}</p>
      <button class="btn" :disabled="scanning" @click="scan">重新扫描</button>
    </div>

    <div v-else class="find-list">
      <div v-for="finding in filtered" :key="finding.key" class="find-row">
        <span class="find-icon" :class="finding.status">
          <Icon :name="finding.status === 'new' ? 'plus' : 'refresh'" :size="13" />
        </span>
        <div class="find-main">
          <div class="find-name">
            {{ finding.name }}
            <span class="badge" :class="finding.status === 'new' ? 'accent' : 'warn'">
              {{ finding.status === "new" ? "未导入" : "有变更" }}
            </span>
            <span v-if="finding.version" class="badge mono">v{{ finding.version }}</span>
          </div>
          <div class="find-sub">
            {{ finding.agentName }} ·
            <span class="mono">{{ finding.path }}</span>
          </div>
        </div>
        <div class="find-meta">
          <span class="mono">{{ finding.hash.slice(0, 10) }}</span>
          <span>{{ finding.fileCount }} 文件</span>
          <span>{{ formatBytes(finding.sizeBytes) }}</span>
        </div>
        <div class="find-actions">
          <button
            v-if="finding.status === 'new'"
            class="btn sm primary"
            :disabled="!!importing"
            @click="importFinding(finding)"
          >
            {{ importing === finding.key ? "导入中…" : "导入" }}
          </button>
          <button
            v-else
            class="btn sm"
            :disabled="!!importing || !!updating"
            @click="updateFinding(finding)"
          >
            {{ updating === finding.key ? "更新中…" : "更新" }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="result?.errors?.length" class="scan-errors">
      <b>扫描中有 {{ result.errors.length }} 个目录被跳过</b>
      <span v-for="error in result.errors.slice(0, 3)" :key="error">{{ error }}</span>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.discover { min-height: 300px; }
.discover-bar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; padding: 2px 0 8px; }
.filters-bar { border-bottom: 1px solid var(--border-soft); padding-bottom: 8px; margin-bottom: 8px; }
.filters { display: flex; gap: 5px; flex-wrap: wrap; }
.chip-count { color: var(--text-4); font-size: 9px; }
.chip.active .chip-count { color: var(--accent); }
.find-list { display: flex; flex-direction: column; }
.find-row { display: flex; align-items: center; gap: 10px; padding: 9px 4px; border-bottom: 1px dashed var(--border-soft); min-width: 0; }
.find-row:last-child { border-bottom: 0; }
.find-icon { display: grid; place-items: center; width: 26px; height: 26px; border-radius: 7px; flex: none; color: var(--accent); background: var(--accent-soft); }
.find-icon.changed { color: var(--warning); background: var(--warning-soft); }
.find-main { flex: 1; min-width: 0; }
.find-name { display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 550; min-width: 0; }
.find-name > .badge { flex: none; }
.find-sub { margin-top: 3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-4); font-size: 9.5px; }
.find-sub .mono { color: var(--text-3); }
.find-meta { display: flex; flex-direction: column; align-items: flex-end; gap: 2px; color: var(--text-4); font-size: 9px; flex: none; }
.find-actions { width: 62px; display: flex; justify-content: flex-end; flex: none; }
.scan-errors { margin-top: 12px; padding: 9px 10px; border: 1px dashed var(--border); border-radius: 8px; color: var(--text-4); font-size: 9.5px; display: flex; flex-direction: column; gap: 3px; }
.scan-errors b { color: var(--warning); font-weight: 550; }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 800px) {
  .find-meta { display: none; }
}
</style>
