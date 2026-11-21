<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import { checkDrift, errorMessage, importScannedSkill, listAgentSkills, openAgentFolder, repairAllDrift, repairDrift, scanAgents, uninstallSkill } from "../lib/api";
import type { AgentSkillItem } from "../../bindings/xinskillvault/internal/app/models";
import type { Result as ScanResult } from "../../bindings/xinskillvault/internal/scan/models";
import type { Report as DriftReport } from "../../bindings/xinskillvault/internal/distribute/models";

const result = ref<ScanResult | null>(null);
const scanning = ref(false);
const drift = ref<DriftReport[]>([]);
const repairing = ref("");
const confirmReport = ref<DriftReport | null>(null);
const confirmAll = ref(false);
const confirmRemove = ref<AgentSkillItem | null>(null);
const selectedID = ref("");
const agentSkills = ref<AgentSkillItem[]>([]);
const skillsLoading = ref(false);
const toast = ref("");
const agentSideWidth = ref(Number(localStorage.getItem("xsv-agent-side-w")) || 250);
const layoutEl = ref<HTMLElement | null>(null);
let resizeStartX = 0;
let resizeStartWidth = 0;
let resizing = false;
let toastTimer: number | undefined;

const agents = computed(() => (result.value?.agents ?? []).slice().sort((a, b) => Number(b.detected) - Number(a.detected) || a.name.localeCompare(b.name)));
const detected = computed(() => agents.value.filter((agent) => agent.detected).length);
const issues = computed(() => drift.value.filter((report) => report.status !== "ok"));
const selectedAgent = computed(() => agents.value.find((agent) => agent.id === selectedID.value) ?? null);

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

function modeLabel(mode: string): string {
  if (mode === "copy") return "复制";
  if (mode === "junction" || mode === "link") return "链接";
  return "—";
}

function statusLabel(status: string): string {
  if (status === "ok") return "正常";
  if (status === "stale") return "仓库已更新";
  if (status === "modified") return "安装件被修改";
  if (status === "missing") return "安装件缺失";
  if (status === "unmanaged") return "未纳管";
  return status;
}

function driftLabel(status: string): string {
  return statusLabel(status);
}

async function loadAgentSkills() {
  if (!selectedAgent.value) {
    agentSkills.value = [];
    return;
  }
  skillsLoading.value = true;
  try {
    agentSkills.value = await listAgentSkills(selectedAgent.value.id);
  } catch (error) {
    agentSkills.value = [];
    showToast(errorMessage(error));
  } finally {
    skillsLoading.value = false;
  }
}

async function selectAgent(id: string) {
  selectedID.value = id;
  await loadAgentSkills();
}

async function refresh() {
  if (scanning.value) return;
  scanning.value = true;
  try {
    const [scan, reports] = await Promise.all([scanAgents(), checkDrift()]);
    result.value = scan;
    drift.value = reports;
    if (!scan.agents?.some((agent) => agent.id === selectedID.value && agent.detected)) {
      selectedID.value = scan.agents?.find((agent) => agent.detected)?.id ?? "";
    }
    await loadAgentSkills();
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

async function importOne(item: AgentSkillItem) {
  if (!selectedAgent.value) return;
  try {
    await importScannedSkill(selectedAgent.value.id, item.installedPath);
    await refresh();
    showToast(`已导入：${item.name}`);
  } catch (error) {
    showToast(errorMessage(error));
  }
}

async function removeOne() {
  if (!confirmRemove.value || !selectedAgent.value) return;
  const item = confirmRemove.value;
  try {
    await uninstallSkill(item.slug, selectedAgent.value.id);
    confirmRemove.value = null;
    await refresh();
    showToast(`已移除：${item.name}`);
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function askRepairAll() {
  if (!repairing.value && issues.value.length) confirmAll.value = true;
}

async function runRepairAll() {
  if (repairing.value) return;
  repairing.value = "all";
  try {
    const summary = await repairAllDrift();
    confirmAll.value = false;
    await refresh();
    showToast(summary.errors.length ? `已修复 ${summary.repaired} 个，${summary.errors.length} 个失败` : `已修复 ${summary.repaired} 个漂移`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    repairing.value = "";
  }
}

function askRepair(report: DriftReport) {
  if (!repairing.value) confirmReport.value = report;
}

async function runRepair() {
  const report = confirmReport.value;
  if (!report || repairing.value) return;
  const key = `${report.skillSlug}:${report.targetId}`;
  repairing.value = key;
  try {
    await repairDrift(report.skillSlug, report.targetId);
    confirmReport.value = null;
    await refresh();
    showToast(`已修复：${report.skillName} → ${report.targetName}`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    repairing.value = "";
  }
}

function startResize(event: PointerEvent) {
  resizing = true;
  resizeStartX = event.clientX;
  resizeStartWidth = agentSideWidth.value;
  window.addEventListener("pointermove", onResize);
  window.addEventListener("pointerup", stopResize);
  event.preventDefault();
}

function onResize(event: PointerEvent) {
  if (!resizing) return;
  agentSideWidth.value = Math.max(160, Math.min(340, Math.round(resizeStartWidth + event.clientX - resizeStartX)));
}

function stopResize() {
  if (!resizing) return;
  resizing = false;
  localStorage.setItem("xsv-agent-side-w", String(agentSideWidth.value));
  window.removeEventListener("pointermove", onResize);
  window.removeEventListener("pointerup", stopResize);
}

function resetSideWidth() {
  agentSideWidth.value = 250;
  localStorage.setItem("xsv-agent-side-w", "250");
}

onMounted(refresh);
onUnmounted(() => {
  window.removeEventListener("pointermove", onResize);
  window.removeEventListener("pointerup", stopResize);
});
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

    <div v-else ref="layoutEl" class="agent-layout" :style="{ gridTemplateColumns: `${agentSideWidth}px minmax(0, 1fr)` }">
      <aside class="agent-list">
        <div class="agent-resizer" title="拖动调整宽度，双击复位" @pointerdown="startResize" @dblclick="resetSideWidth" />
        <button
          v-for="agent in agents"
          :key="agent.id"
          class="agent-list-row"
          :class="{ active: agent.id === selectedID, muted: !agent.detected }"
          @click="selectAgent(agent.id)"
        >
          <span class="agent-ava" :style="{ background: agent.color }">{{ agent.short }}</span>
          <span class="agent-list-main">
            <b>{{ agent.name }}</b>
            <em class="mono">{{ agent.path }}</em>
            <small>{{ agent.count }} 个技能 · {{ agent.detected ? "已检测" : "未安装" }}</small>
          </span>
        </button>
      </aside>

      <section v-if="selectedAgent" class="agent-detail">
        <header class="agent-detail-head">
          <div>
            <h2>{{ selectedAgent.name }}</h2>
            <span class="mono">{{ selectedAgent.path }}</span>
          </div>
          <div class="agent-detail-actions">
            <button class="btn sm" @click="openFolder(selectedAgent.id)"><Icon name="folder" :size="12" /> 打开目录</button>
          </div>
        </header>

        <div v-if="skillsLoading" class="empty-mini">正在读取 Agent 技能…</div>
        <div v-else-if="!agentSkills.length" class="empty-mini">这个 Agent 目录里还没有发现技能。</div>
        <div v-else class="agent-skill-table">
          <div class="as-row as-head"><span>#</span><span>技能</span><span>方式</span><span>状态</span><span>操作</span></div>
          <div v-for="(item, index) in agentSkills" :key="`${item.slug}:${item.installedPath}`" class="as-row">
            <span class="as-idx">{{ index + 1 }}</span>
            <span class="as-main">
              <b>{{ item.name }}</b>
              <em>{{ item.description }}</em>
              <small class="mono">{{ item.slug }}<template v-if="item.version"> · v{{ item.version }}</template></small>
            </span>
            <span class="badge" :class="{ accent: item.managed }">{{ modeLabel(item.mode) }}</span>
            <span class="badge" :class="{ warn: item.status !== 'ok', accent: item.status === 'ok' }">{{ statusLabel(item.status) }}</span>
            <span class="as-actions">
              <button v-if="!item.managed" class="btn sm" @click="importOne(item)">导入</button>
              <button v-else class="mini-link danger" @click="confirmRemove = item">移除</button>
            </span>
          </div>
        </div>
      </section>

      <section v-else class="agent-detail agent-placeholder">
        <div class="empty">
          <div class="empty-ico"><Icon name="agent" :size="19" /></div>
          <h3>选择一个 Agent</h3>
          <p>左侧选择 Agent，查看它目录中的技能和安装状态。</p>
        </div>
      </section>
    </div>

    <div v-if="issues.length" class="drift-section">
      <div class="drift-head">
        <div>
          <h2>漂移与异常安装 <span class="badge warn">{{ issues.length }}</span></h2>
          <p>修复会用仓库当前版本覆盖 Agent 目录中的安装件；“安装件被修改”的本地改动会被覆盖。</p>
        </div>
        <button class="btn sm" :disabled="!!repairing" @click="askRepairAll">
          {{ repairing === "all" ? "修复中…" : "全部修复" }}
        </button>
      </div>
      <div class="drift-list">
        <div v-for="report in issues" :key="`${report.skillSlug}:${report.targetId}`" class="drift-row">
          <span class="drift-status" :class="report.status">{{ driftLabel(report.status) }}</span>
          <div class="drift-main">
            <b>{{ report.skillName }}</b>
            <span class="mono">{{ report.targetName }} · {{ modeLabel(report.mode) }}</span>
            <span class="mono drift-path">{{ report.installedPath }}</span>
          </div>
          <button class="btn sm" :disabled="!!repairing" @click="askRepair(report)">
            {{ repairing === `${report.skillSlug}:${report.targetId}` ? "修复中…" : "修复" }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="result?.errors?.length" class="agent-errors">
      <b>{{ result.errors.length }} 个目录扫描失败</b>
      <span v-for="error in result.errors.slice(0, 3)" :key="error">{{ error }}</span>
    </div>

    <div v-if="confirmAll" class="agent-confirm" @click.self.stop>
      <div class="agent-confirm-card">
        <div class="agent-confirm-head">
          <div>
            <h3>修复全部漂移</h3>
            <p>将用仓库当前版本覆盖 {{ issues.length }} 个异常安装件。若安装件被本地修改，这些修改会丢失。</p>
          </div>
          <button class="icon-btn" :disabled="!!repairing" @click="confirmAll = false"><Icon name="x" :size="15" /></button>
        </div>
        <div class="agent-confirm-foot">
          <button class="btn" :disabled="!!repairing" @click="confirmAll = false">取消</button>
          <button class="btn primary" :disabled="!!repairing" @click="runRepairAll">{{ repairing === "all" ? "修复中…" : "确认修复" }}</button>
        </div>
      </div>
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

    <div v-if="confirmRemove" class="agent-confirm" @click.self.stop>
      <div class="agent-confirm-card">
        <h3>移除 Agent 技能</h3>
        <p>将删除 <b>{{ confirmRemove.name }}</b> 在 {{ selectedAgent?.name }} 目录中的链接/复制件，不会删除仓库技能。</p>
        <div class="agent-confirm-foot">
          <button class="btn" @click="confirmRemove = null">取消</button>
          <button class="btn danger" @click="removeOne">移除</button>
        </div>
      </div>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.agent-layout { display: grid; grid-template-columns: 250px minmax(0, 1fr); gap: 14px; align-items: start; min-height: 360px; }
.agent-list { display: flex; flex-direction: column; gap: 3px; min-width: 0; position: relative; }
.agent-resizer { position: absolute; top: 0; right: -7px; bottom: 0; width: 7px; cursor: col-resize; z-index: 2; }
.agent-resizer:hover { background: var(--accent-soft); }
.agent-list-row { display: flex; align-items: flex-start; gap: 7px; width: 100%; padding: 8px; border: 1px solid transparent; border-radius: 8px; text-align: left; }
.agent-list-row:hover { background: var(--panel-2); }
.agent-list-row.active { border-color: var(--border); background: var(--panel); box-shadow: var(--shadow-sm); }
.agent-list-row.muted { opacity: .55; }
.agent-list-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
.agent-list-main b { font-size: 11px; font-weight: 600; }
.agent-list-main em { color: var(--text-4); font-size: 8.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.agent-list-main small { color: var(--text-4); font-size: 8.5px; }
.agent-detail { min-width: 0; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); overflow: hidden; }
.agent-detail-head { display: flex; align-items: flex-start; gap: 10px; padding: 13px 15px 10px; border-bottom: 1px solid var(--border-soft); }
.agent-detail-head h2 { margin: 0; font-size: 15px; }
.agent-detail-head span { color: var(--text-4); font-size: 9px; }
.agent-detail-actions { margin-left: auto; }
.agent-skill-table { padding: 0 14px 10px; overflow: auto; min-width: 620px; }
.as-row { display: grid; grid-template-columns: 20px minmax(200px, 1fr) 62px 86px 50px; gap: 7px; align-items: center; padding: 7px 0; border-bottom: 1px dashed var(--border-soft); min-width: 0; }
.as-row:last-child { border-bottom: 0; }
.as-head { color: var(--text-4); font-size: 9px; border-bottom-style: solid; }
.as-idx { color: var(--text-4); font-size: 9px; }
.as-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.as-main b { font-size: 11px; font-weight: 550; }
.as-main em { color: var(--text-4); font-style: normal; font-size: 9px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.as-main small { color: var(--text-4); font-size: 8px; }
.as-actions { display: flex; justify-content: flex-end; }
.agent-placeholder { min-height: 300px; }
.empty-mini { padding: 24px; text-align: center; color: var(--text-4); font-size: 10px; }
.drift-section { margin-top: 18px; border-top: 1px solid var(--border-soft); padding-top: 14px; }
.drift-head { display: flex; align-items: flex-start; gap: 10px; }
.drift-head > .btn { margin-left: auto; flex: none; }
.drift-head h2 { margin: 0; font-size: 14px; display: flex; align-items: center; gap: 6px; }
.drift-head p { margin: 5px 0 0; color: var(--text-4); font-size: 10px; line-height: 1.6; }
.drift-list { margin-top: 9px; display: flex; flex-direction: column; }
.drift-row { display: flex; align-items: center; gap: 9px; padding: 8px 4px; border-bottom: 1px dashed var(--border-soft); min-width: 0; }
.drift-row:last-child { border-bottom: 0; }
.drift-status { flex: none; padding: 2px 6px; border-radius: 4px; font-size: 9px; color: var(--warning); background: var(--warning-soft); }
.drift-status.missing, .drift-status.modified { color: var(--danger); background: var(--danger-soft); }
.drift-main { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: 2px; font-size: 10px; }
.drift-main b { font-size: 11px; font-weight: 550; }
.drift-main .mono { color: var(--text-4); font-size: 9px; }
.drift-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.agent-errors { margin-top: 12px; padding: 9px 10px; border: 1px dashed var(--border); border-radius: 8px; color: var(--text-4); font-size: 9.5px; display: flex; flex-direction: column; gap: 3px; }
.agent-errors b { color: var(--warning); font-weight: 550; }
.agent-confirm { position: fixed; inset: 0; z-index: 200; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.agent-confirm-card { width: min(420px, calc(100vw - 32px)); border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); padding: 16px; }
.agent-confirm-head { display: flex; align-items: flex-start; gap: 10px; }
.agent-confirm-head h3, .agent-confirm-card h3 { margin: 0; font-size: 14px; }
.agent-confirm-head p, .agent-confirm-card p { margin: 5px 0 0; color: var(--text-3); font-size: 11px; line-height: 1.65; }
.agent-confirm-head .icon-btn { margin-left: auto; flex: none; }
.agent-confirm-foot { display: flex; justify-content: flex-end; gap: 6px; margin-top: 14px; }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
@media (max-width: 900px) { .agent-layout { grid-template-columns: 200px minmax(0, 1fr); } .as-row { grid-template-columns: 16px minmax(130px, 1fr) 55px 75px 45px; } }
</style>
