<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import { checkDrift, errorMessage, openAgentFolder, repairDrift, scanAgents } from "../lib/api";
import type { Result as ScanResult } from "../../bindings/xinskillvault/internal/scan/models";
import type { Report as DriftReport } from "../../bindings/xinskillvault/internal/distribute/models";

const result = ref<ScanResult | null>(null);
const scanning = ref(false);
const drift = ref<DriftReport[]>([]);
const repairing = ref("");
const confirmReport = ref<DriftReport | null>(null);
const toast = ref("");
let toastTimer: number | undefined;

const agents = computed(() => (result.value?.agents ?? []).slice().sort((a, b) => Number(b.detected) - Number(a.detected) || a.name.localeCompare(b.name)));
const detected = computed(() => agents.value.filter((agent) => agent.detected).length);
const issues = computed(() => drift.value.filter((report) => report.status !== "ok"));

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

async function refresh() {
  if (scanning.value) return;
  scanning.value = true;
  try {
    const [scan, reports] = await Promise.all([scanAgents(), checkDrift()]);
    result.value = scan;
    drift.value = reports;
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    scanning.value = false;
  }
}

async function openFolder(id: string) {
  try {
    await openAgentFolder(id);
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function driftLabel(status: string): string {
  if (status === "stale") return "仓库已更新";
  if (status === "modified") return "安装件被修改";
  if (status === "missing") return "安装件缺失";
  return status;
}

function askRepair(report: DriftReport) {
  if (!repairing.value) confirmReport.value = report;
}

async function runRepair() {
  const report = confirmReport.value;
  if (!report || repairing.value) return;
  const key = `${report.skillSlug}:${report.agentId}`;
  repairing.value = key;
  try {
    await repairDrift(report.skillSlug, report.agentId);
    confirmReport.value = null;
    await refresh();
    showToast(`已修复：${report.skillName} → ${report.targetName}`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    repairing.value = "";
  }
}

onMounted(refresh);
</script>

<template>
  <div class="page agent-page">
    <div class="page-head">
      <div>
        <h1>Agent <span class="badge phase">M2</span></h1>
        <p>{{ detected }} 个目标已检测 · 扫描 {{ result?.scannedSkills ?? 0 }} 个技能 · {{ issues.length }} 个漂移</p>
      </div>
      <div class="actions">
        <button class="btn" :disabled="scanning" @click="refresh">
          <Icon name="refresh" :size="14" :class="{ spin: scanning }" />
          {{ scanning ? "扫描中…" : "刷新" }}
        </button>
      </div>
    </div>

    <div v-if="!agents.length && !scanning" class="empty">
      <div class="empty-ico"><Icon name="agent" :size="19" /></div>
      <h3>没有检测到 Agent 目录</h3>
      <p>桌面应用会检测本机 Claude Code、Codex、Cursor、Alma 等常见 skill 目录。</p>
    </div>

    <div v-else class="agent-grid">
      <div v-for="agent in agents" :key="agent.id" class="card agent-card" :class="{ muted: !agent.detected }">
        <div class="agent-ava" :style="{ background: agent.color }">{{ agent.short }}</div>
        <div class="agent-main">
          <div class="agent-name">
            {{ agent.name }}
            <span class="badge" :class="agent.detected ? 'accent' : ''">{{ agent.detected ? "已检测" : "未安装" }}</span>
          </div>
          <div class="agent-path mono">{{ agent.path }}</div>
        </div>
        <div class="agent-count">
          <b>{{ agent.count }}</b>
          <span>技能</span>
        </div>
        <button class="btn sm" :disabled="!agent.detected" title="打开该 Agent 的 skill 目录" @click="openFolder(agent.id)">
          <Icon name="folder" :size="12" /> 打开
        </button>
      </div>
    </div>

    <div v-if="issues.length" class="drift-section">
      <div class="drift-head">
        <div>
          <h2>漂移与异常安装 <span class="badge warn">{{ issues.length }}</span></h2>
          <p>修复会用仓库当前版本覆盖 Agent 目录中的安装件；“安装件被修改”的本地改动会被覆盖。</p>
        </div>
      </div>
      <div class="drift-list">
        <div v-for="report in issues" :key="`${report.skillSlug}:${report.agentId}`" class="drift-row">
          <span class="drift-status" :class="report.status">{{ driftLabel(report.status) }}</span>
          <div class="drift-main">
            <b>{{ report.skillName }}</b>
            <span class="mono">{{ report.targetName }} · {{ report.mode === "link" ? "链接" : "复制" }}</span>
            <span class="mono drift-path">{{ report.installedPath }}</span>
          </div>
          <button class="btn sm" :disabled="!!repairing" @click="askRepair(report)">
            {{ repairing === `${report.skillSlug}:${report.agentId}` ? "修复中…" : "修复" }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="result?.errors?.length" class="agent-errors">
      <b>{{ result.errors.length }} 个目录扫描失败</b>
      <span v-for="error in result.errors.slice(0, 3)" :key="error">{{ error }}</span>
    </div>

    <div v-if="confirmReport" class="agent-confirm" @click.self.stop>
      <div class="agent-confirm-card">
        <div class="agent-confirm-head">
          <div>
            <h3>修复漂移</h3>
            <p>将用仓库当前版本覆盖 <b>{{ confirmReport.skillName }}</b> 在 {{ confirmReport.targetName }} 的安装件。若安装件被本地修改，这些修改会丢失。</p>
          </div>
          <button class="icon-btn" :disabled="!!repairing" @click="confirmReport = null"><Icon name="x" :size="15" /></button>
        </div>
        <div class="agent-confirm-foot">
          <button class="btn" :disabled="!!repairing" @click="confirmReport = null">取消</button>
          <button class="btn primary" :disabled="!!repairing" @click="runRepair">{{ repairing ? "修复中…" : "确认修复" }}</button>
        </div>
      </div>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.agent-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 8px; }
.agent-card { display: flex; align-items: center; gap: 10px; padding: 11px 12px; }
.agent-card.muted { opacity: .55; }
.agent-ava { display: grid; place-items: center; width: 28px; height: 28px; border-radius: 8px; color: #fff; font-size: 9px; font-weight: 700; flex: none; }
.agent-main { min-width: 0; flex: 1; }
.agent-name { display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 550; }
.agent-path { margin-top: 3px; color: var(--text-4); font-size: 9px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.agent-count { display: flex; flex-direction: column; align-items: flex-end; flex: none; }
.agent-count b { font-size: 14px; font-weight: 600; }
.agent-count span { color: var(--text-4); font-size: 8.5px; }
.drift-section { margin-top: 18px; border-top: 1px solid var(--border-soft); padding-top: 14px; }
.drift-head h2 { margin: 0; font-size: 14px; display: flex; align-items: center; gap: 6px; }
.drift-head p { margin: 5px 0 0; color: var(--text-4); font-size: 10px; line-height: 1.6; }
.drift-list { margin-top: 9px; display: flex; flex-direction: column; }
.drift-row { display: flex; align-items: center; gap: 9px; padding: 8px 4px; border-bottom: 1px dashed var(--border-soft); min-width: 0; }
.drift-row:last-child { border-bottom: 0; }
.drift-status { flex: none; padding: 2px 6px; border-radius: 4px; font-size: 9px; color: var(--warning); background: var(--warning-soft); }
.drift-status.missing { color: var(--danger); background: var(--danger-soft); }
.drift-status.modified { color: var(--danger); background: var(--danger-soft); }
.drift-main { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: 2px; font-size: 10px; }
.drift-main b { font-size: 11px; font-weight: 550; }
.drift-main .mono { color: var(--text-4); font-size: 9px; }
.drift-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.agent-confirm { position: fixed; inset: 0; z-index: 200; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.agent-confirm-card { width: min(420px, calc(100vw - 32px)); border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); }
.agent-confirm-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px; }
.agent-confirm-head h3 { margin: 0; font-size: 14px; }
.agent-confirm-head p { margin: 5px 0 0; color: var(--text-3); font-size: 11px; line-height: 1.65; }
.agent-confirm-head .icon-btn { margin-left: auto; flex: none; }
.agent-confirm-foot { display: flex; justify-content: flex-end; gap: 6px; padding: 10px 14px; border-top: 1px solid var(--border-soft); }
.agent-errors { margin-top: 12px; padding: 9px 10px; border: 1px dashed var(--border); border-radius: 8px; color: var(--text-4); font-size: 9.5px; display: flex; flex-direction: column; gap: 3px; }
.agent-errors b { color: var(--warning); font-weight: 550; }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
