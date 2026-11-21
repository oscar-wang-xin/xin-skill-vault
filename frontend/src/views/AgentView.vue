<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import { errorMessage, openAgentFolder, scanAgents } from "../lib/api";
import type { Result as ScanResult } from "../../bindings/xinskillvault/internal/scan/models";

const result = ref<ScanResult | null>(null);
const scanning = ref(false);
const toast = ref("");
let toastTimer: number | undefined;

const agents = computed(() => (result.value?.agents ?? []).slice().sort((a, b) => Number(b.detected) - Number(a.detected) || a.name.localeCompare(b.name)));
const detected = computed(() => agents.value.filter((agent) => agent.detected).length);

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2400);
}

async function refresh() {
  if (scanning.value) return;
  scanning.value = true;
  try {
    result.value = await scanAgents();
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

onMounted(refresh);
</script>

<template>
  <div class="page agent-page">
    <div class="page-head">
      <div>
        <h1>Agent <span class="badge phase">M2</span></h1>
        <p>{{ detected }} 个目标已检测 · 扫描 {{ result?.scannedSkills ?? 0 }} 个技能 · 分发入口在技能详情抽屉</p>
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

    <div v-if="result?.errors?.length" class="agent-errors">
      <b>{{ result.errors.length }} 个目录扫描失败</b>
      <span v-for="error in result.errors.slice(0, 3)" :key="error">{{ error }}</span>
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
.agent-errors { margin-top: 12px; padding: 9px 10px; border: 1px dashed var(--border); border-radius: 8px; color: var(--text-4); font-size: 9.5px; display: flex; flex-direction: column; gap: 3px; }
.agent-errors b { color: var(--warning); font-weight: 550; }
.spin { animation: spin 900ms linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
