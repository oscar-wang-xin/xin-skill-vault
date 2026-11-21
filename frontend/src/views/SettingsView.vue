<script setup lang="ts">
import { onMounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import {
  errorMessage,
  exportDiagnostics,
  getAppInfo,
  getConfig,
  openConfigFolder,
  openPromptFolder,
  openVaultFolder,
  saveConfig,
} from "../lib/api";
import type { AppInfo } from "../../bindings/xinskillvault/internal/app/models";
import type { Config } from "../../bindings/xinskillvault/internal/config/models";
import { themes, useUiStore } from "../stores/ui";

const ui = useUiStore();
const info = ref<AppInfo | null>(null);
const config = ref<Config>({
  vaultPath: "",
  promptRoot: "",
  theme: "mech-gray",
  neumorph: true,
  language: "zh-CN",
});
const loading = ref(true);
const saving = ref(false);
const diagnosticsBusy = ref(false);
const notice = ref("");

function showNotice(message: string) {
  notice.value = message;
  window.setTimeout(() => {
    if (notice.value === message) notice.value = "";
  }, 2600);
}

async function load() {
  loading.value = true;
  try {
    const [nextInfo, nextConfig] = await Promise.all([getAppInfo(), getConfig()]);
    info.value = nextInfo;
    config.value = nextConfig;
  } catch (error) {
    showNotice(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function diagnosticsExport() {
  if (diagnosticsBusy.value) return;
  diagnosticsBusy.value = true;
  try {
    const path = await exportDiagnostics();
    if (path) notice.value = `诊断包已导出：${path}`;
  } catch (error) {
    notice.value = errorMessage(error);
  } finally {
    diagnosticsBusy.value = false;
  }
}

async function save() {
  saving.value = true;
  config.value.theme = ui.theme;
  config.value.neumorph = ui.neumorph;
  try {
    const result = await saveConfig(config.value);
    ui.applyConfig({ theme: config.value.theme, neumorph: config.value.neumorph });
    showNotice(result.restartRequired ? "配置已保存；仓库路径变更将在重启后生效" : "配置已保存");
  } catch (error) {
    showNotice(errorMessage(error));
  } finally {
    saving.value = false;
  }
}

async function openFolder(kind: "vault" | "prompt" | "config") {
  try {
    if (kind === "vault") await openVaultFolder();
    else if (kind === "prompt") await openPromptFolder();
    else await openConfigFolder();
  } catch (error) {
    showNotice(errorMessage(error));
  }
}

onMounted(load);
</script>

<template>
  <div class="page settings-page">
    <div class="page-head">
      <div>
        <h1>设置 <span class="badge phase">M0</span></h1>
        <p>所有配置保存在本地；修改后写入配置文件，仓库路径变更需重启。</p>
      </div>
      <div class="actions">
        <button class="btn primary" :disabled="saving || loading" @click="save">
          <Icon name="check" :size="13" />
          {{ saving ? "保存中…" : "保存配置" }}
        </button>
      </div>
    </div>

    <div v-if="notice" class="notice" :class="{ warn: notice.includes('失败') || notice.includes('error') }">{{ notice }}</div>

    <section class="card set-card">
      <div class="set-head">
        <div><h2>外观</h2><p>主题与拟态风格会同步到顶栏开关。</p></div>
        <span class="badge accent">6 套主题</span>
      </div>
      <div class="theme-grid">
        <button
          v-for="theme in themes"
          :key="theme.id"
          class="theme-card"
          :class="{ active: ui.theme === theme.id }"
          @click="ui.setTheme(theme.id)"
        >
          <span class="theme-swatch">
            <i v-for="color in theme.colors" :key="color" :style="{ background: color }" />
          </span>
          <span>{{ theme.name }}</span>
          <Icon v-if="ui.theme === theme.id" name="check" :size="12" />
        </button>
      </div>
      <div class="set-row">
        <div class="set-text"><b>拟态风格</b><span>关闭后使用线性边框与轻投影</span></div>
        <button class="ui-switch" :class="{ on: ui.neumorph }" role="switch" :aria-checked="ui.neumorph" @click="ui.toggleNeumorph()"><span /></button>
      </div>
    </section>

    <section class="card set-card">
      <div class="set-head">
        <div><h2>存储</h2><p>仓库与提示词目录；仓库是技能的唯一定位来源。</p></div>
      </div>
      <label class="field">
        <span>本地仓库</span>
        <div class="field-row">
          <input v-model="config.vaultPath" class="input mono" />
          <button class="btn" title="打开仓库目录" @click="openFolder('vault')"><Icon name="folder" :size="13" /></button>
        </div>
      </label>
      <label class="field">
        <span>提示词根目录</span>
        <div class="field-row">
          <input v-model="config.promptRoot" class="input mono" />
          <button class="btn" title="打开提示词目录" @click="openFolder('prompt')"><Icon name="folder" :size="13" /></button>
        </div>
      </label>
      <div class="set-row">
        <div class="set-text"><b>配置文件</b><span class="mono">{{ info?.configPath ?? "—" }}</span></div>
        <button class="btn" @click="openFolder('config')"><Icon name="external" :size="12" /> 打开</button>
      </div>
    </section>

    <section class="card set-card">
      <div class="set-head">
        <div><h2>运行信息</h2><p>前端与本地索引的启动状态。</p></div>
        <button class="btn" :disabled="diagnosticsBusy" @click="diagnosticsExport"><Icon name="download" :size="12" /> {{ diagnosticsBusy ? "导出中…" : "导出诊断包" }}</button>
      </div>
      <div class="kv-grid">
        <div class="kv"><span>版本</span><b>{{ info?.version ?? "0.1.0" }}</b></div>
        <div class="kv"><span>平台</span><b>{{ info?.platform ?? "—" }}</b></div>
        <div class="kv"><span>Schema</span><b>v{{ info?.schemaVersion ?? 1 }}</b></div>
        <div class="kv"><span>数据库</span><b class="mono small">{{ info?.databasePath ?? "—" }}</b></div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-page { max-width: 920px; }
.notice {
  margin-bottom: 10px;
  padding: 8px 10px;
  border: 1px solid color-mix(in oklab, var(--success) 24%, var(--border));
  border-radius: 8px;
  background: var(--success-soft);
  color: var(--success);
  font-size: 11px;
}
.notice.warn { color: var(--warning); background: var(--warning-soft); border-color: color-mix(in oklab, var(--warning) 24%, var(--border)); }
.set-card { padding: 16px; margin-bottom: 12px; }
.set-head { display: flex; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
.set-head h2 { margin: 0; font-size: 13px; font-weight: 650; }
.set-head p { margin: 4px 0 0; color: var(--text-3); font-size: 10.5px; }
.set-head .badge { margin-left: auto; }
.theme-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(145px, 1fr)); gap: 7px; margin-bottom: 8px; }
.theme-card { display: flex; align-items: center; gap: 7px; padding: 8px; border: 1px solid var(--border); border-radius: 8px; background: var(--panel); text-align: left; font-size: 11px; color: var(--text-2); }
.theme-card:hover { border-color: var(--border-strong); }
.theme-card.active { border-color: var(--accent); background: var(--accent-soft); color: var(--text); box-shadow: var(--nm-inset); }
.theme-swatch { display: flex; gap: 2px; }
.theme-swatch i { width: 12px; height: 12px; border-radius: 4px; }
.theme-card svg { margin-left: auto; color: var(--accent); }
.set-row { display: flex; align-items: center; gap: 16px; padding: 10px 0; border-top: 1px solid var(--border-soft); }
.set-text { min-width: 0; flex: 1; }
.set-text b { display: block; font-size: 11.5px; font-weight: 550; }
.set-text span { display: block; margin-top: 3px; color: var(--text-4); font-size: 10px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ui-switch { position: relative; width: 38px; height: 20px; border-radius: 99px; background: var(--panel-3); box-shadow: var(--nm-inset); flex: none; }
.ui-switch span { position: absolute; top: 3px; left: 3px; width: 14px; height: 14px; border-radius: 50%; background: var(--panel); box-shadow: 0 1px 3px rgba(16, 18, 22, .22); transition: transform var(--dur) var(--ease); }
.ui-switch.on { background: var(--accent); }
.ui-switch.on span { transform: translateX(18px); }
.field { display: block; margin-bottom: 10px; }
.field > span { display: block; margin-bottom: 4px; color: var(--text-3); font-size: 10px; }
.field-row { display: flex; gap: 6px; }
.field-row .input { flex: 1; }
.field-row .btn { flex: none; }
.kv-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; }
.kv { display: flex; align-items: baseline; gap: 10px; padding: 8px 0; border-bottom: 1px dashed var(--border-soft); }
.kv span { width: 70px; color: var(--text-4); font-size: 10px; flex: none; }
.kv b { font-size: 11px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kv b.small { font-size: 10px; font-family: var(--mono); }
@media (max-width: 700px) { .kv-grid { grid-template-columns: 1fr; } }
</style>
