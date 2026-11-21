<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import Icon from "../components/Icon.vue";
import {
  addProjectSkills,
  createProject,
  deleteProject,
  errorMessage,
  listProjectSkills,
  listProjects,
  listSkills,
  openProjectFolder,
  pickProjectDirectory,
  removeProjectSkill,
  scanAgents,
  setProjectAgents,
  updateProject,
} from "../lib/api";
import type { ProjectItem, ProjectSkill, SkillItem } from "../../bindings/xinskillvault/internal/app/models";
import type { AgentResult } from "../../bindings/xinskillvault/internal/scan/models";

const projects = ref<ProjectItem[]>([]);
const projectSkills = ref<ProjectSkill[]>([]);
const allSkills = ref<SkillItem[]>([]);
const allAgents = ref<AgentResult[]>([]);
const selectedId = ref<string | null>(null);
const loading = ref(true);
const toast = ref("");
let toastTimer: number | undefined;

const selected = computed(() => projects.value.find((item) => item.id === selectedId.value) ?? null);
const availableAgents = computed(() => allAgents.value.filter((item) => item.detected));
const selectedAgentIDs = computed(() => (selected.value?.agents ?? []).map((item) => item.id));

const projectModal = ref<{ open: boolean; mode: "create" | "edit"; name: string; path: string; description: string }>({
  open: false, mode: "create", name: "", path: "", description: "",
});
const projectModalAgents = ref<string[]>([]);
const agentModal = ref(false);
const agentDraft = ref<string[]>([]);
const skillModal = ref(false);
const skillQuery = ref("");
const skillDraft = ref<string[]>([]);
const skillMode = ref<"link" | "copy">("link");
const confirm = ref<null | { kind: "project" | "skill"; title: string; message: string; slug?: string }>(null);
const busy = ref(false);

const filteredSkills = computed(() => {
  const query = skillQuery.value.trim().toLowerCase();
  if (!query) return allSkills.value;
  return allSkills.value.filter((item) =>
    item.name.toLowerCase().includes(query)
    || item.slug.toLowerCase().includes(query)
    || item.description.toLowerCase().includes(query));
});

function formatDate(seconds: number): string {
  if (!seconds) return "—";
  return new Date(seconds * 1000).toLocaleString("zh-CN", { hour12: false });
}

function showToast(message: string) {
  toast.value = message;
  window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => { toast.value = ""; }, 2600);
}

async function loadProjects(preferred?: string) {
  try {
    projects.value = await listProjects();
    const next = preferred ?? selectedId.value;
    if (next && projects.value.some((item) => item.id === next)) selectedId.value = next;
    else selectedId.value = projects.value[0]?.id ?? null;
    await loadProjectSkills();
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    loading.value = false;
  }
}

async function loadProjectSkills() {
  if (!selectedId.value) {
    projectSkills.value = [];
    return;
  }
  try {
    projectSkills.value = await listProjectSkills(selectedId.value);
  } catch (error) {
    projectSkills.value = [];
    showToast(errorMessage(error));
  }
}

async function selectProject(id: string) {
  selectedId.value = id;
  await loadProjectSkills();
}

async function loadAgents() {
  try {
    const result = await scanAgents();
    allAgents.value = result.agents ?? [];
  } catch {
    allAgents.value = [];
  }
}

async function openCreate() {
  projectModal.value = { open: true, mode: "create", name: "", path: "", description: "" };
  projectModalAgents.value = availableAgents.value.slice(0, 2).map((item) => item.id);
}

function openEdit() {
  if (!selected.value) return;
  projectModal.value = {
    open: true, mode: "edit", name: selected.value.name,
    path: selected.value.path, description: selected.value.description,
  };
  projectModalAgents.value = [];
}

async function browseProject() {
  try {
    const path = await pickProjectDirectory();
    if (path) projectModal.value.path = path;
  } catch (error) {
    showToast(errorMessage(error));
  }
}

async function saveProject() {
  if (busy.value) return;
  busy.value = true;
  try {
    if (!projectModal.value.name.trim() || (!projectModal.value.path.trim() && projectModal.value.mode === "create")) {
      showToast("项目名称和目录不能为空");
      return;
    }
    let item: ProjectItem;
    if (projectModal.value.mode === "create") {
      item = await createProject(projectModal.value.name, projectModal.value.path, projectModal.value.description, projectModalAgents.value);
    } else if (selected.value) {
      item = await updateProject(selected.value.id, projectModal.value.name, projectModal.value.description);
    } else {
      return;
    }
    projectModal.value.open = false;
    await loadProjects(item.id);
    showToast(projectModal.value.mode === "create" ? "项目已创建" : "项目已更新");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

function openAgents() {
  if (!selected.value) return;
  agentDraft.value = [...selectedAgentIDs.value];
  agentModal.value = true;
}

function toggleAgent(id: string) {
  agentDraft.value = agentDraft.value.includes(id)
    ? agentDraft.value.filter((item) => item !== id)
    : [...agentDraft.value, id];
}

async function saveAgents() {
  if (!selected.value || busy.value) return;
  busy.value = true;
  try {
    await setProjectAgents(selected.value.id, agentDraft.value);
    agentModal.value = false;
    await loadProjects(selected.value.id);
    showToast("项目 Agent 已更新");
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

function openSkillModal() {
  if (!selected.value) return;
  if (!selectedAgentIDs.value.length) {
    showToast("先配置项目 Agent，再添加技能");
    return;
  }
  skillDraft.value = [];
  skillQuery.value = "";
  skillMode.value = "link";
  skillModal.value = true;
}

function toggleSkill(slug: string) {
  skillDraft.value = skillDraft.value.includes(slug)
    ? skillDraft.value.filter((item) => item !== slug)
    : [...skillDraft.value, slug];
}

async function saveSkills() {
  if (!selected.value || !skillDraft.value.length || busy.value) return;
  busy.value = true;
  try {
    const result = await addProjectSkills(selected.value.id, skillDraft.value, skillMode.value);
    await loadProjects(selected.value.id);
    skillModal.value = false;
    if (result.errors?.length) showToast(`已添加 ${result.added} 个，${result.errors.length} 个失败`);
    else showToast(`已添加 ${result.added} 个技能`);
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

function askRemoveSkill(skill: ProjectSkill) {
  confirm.value = {
    kind: "skill", slug: skill.slug, title: "从项目移除技能",
    message: `将从项目支持的所有 Agent 目录移除「${skill.name}」的链接/复制件，不会删除仓库技能。`,
  };
}

function askRemoveProject() {
  if (!selected.value) return;
  confirm.value = {
    kind: "project", title: "移除项目",
    message: `移除「${selected.value.name}」及其项目级分发记录，不删除项目目录和文件。`,
  };
}

async function runConfirm() {
  if (!confirm.value || !selected.value) return;
  busy.value = true;
  try {
    if (confirm.value.kind === "project") {
      await deleteProject(selected.value.id);
      confirm.value = null;
      await loadProjects();
      showToast("项目已移除");
    } else if (confirm.value.slug) {
      await removeProjectSkill(selected.value.id, confirm.value.slug);
      confirm.value = null;
      await loadProjects(selected.value.id);
      showToast("技能已从项目移除");
    }
  } catch (error) {
    showToast(errorMessage(error));
  } finally {
    busy.value = false;
  }
}

async function openFolder() {
  if (!selected.value) return;
  try {
    await openProjectFolder(selected.value.id);
  } catch (error) {
    showToast(errorMessage(error));
  }
}

function modes(skill: ProjectSkill): string {
  const set = new Set((skill.agents ?? []).map((item) => item.mode === "link" ? "链接" : "复制"));
  return Array.from(set).join(" + ") || "—";
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== "Escape" || busy.value) return;
  if (confirm.value) confirm.value = null;
  else if (skillModal.value) skillModal.value = false;
  else if (agentModal.value) agentModal.value = false;
  else if (projectModal.value.open) projectModal.value.open = false;
}

onMounted(async () => {
  await Promise.all([loadAgents(), listSkills().then((items) => { allSkills.value = items; })]);
  await loadProjects();
  window.addEventListener("keydown", onKeydown);
});
onUnmounted(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <div class="page projects-page">
    <div class="page-head">
      <div>
        <h1>项目 <span class="badge phase">M2</span></h1>
        <p>{{ projects.length }} 个项目 · 技能只进项目目录，不污染全局 Agent</p>
      </div>
      <div class="actions">
        <button class="btn primary" @click="openCreate"><Icon name="plus" :size="13" /> 新建项目</button>
      </div>
    </div>

    <div class="projects-layout">
      <aside class="project-side">
        <div v-if="!projects.length && !loading" class="project-empty">
          <Icon name="projects" :size="18" />
          <b>还没有项目</b>
          <span>新建一个项目，把技能装进项目目录。</span>
        </div>
        <button
          v-for="project in projects"
          :key="project.id"
          class="project-card"
          :class="{ active: project.id === selectedId }"
          @click="selectProject(project.id)"
        >
          <span class="project-ava">{{ project.name.slice(0, 1).toUpperCase() }}</span>
          <span class="project-info">
            <b>{{ project.name }}</b>
            <em class="mono">{{ project.path }}</em>
            <small>{{ project.skillCount }} 个技能 · {{ project.agents?.length ?? 0 }} 个 Agent</small>
          </span>
        </button>
      </aside>

      <section v-if="selected" class="project-detail">
        <div class="project-head">
          <div class="project-title">
            <h2>{{ selected.name }}</h2>
            <span class="mono">{{ selected.path }}</span>
            <span class="muted">创建于 {{ formatDate(selected.createdAt) }}</span>
          </div>
          <div class="project-actions">
            <button class="btn sm" @click="openFolder"><Icon name="folder" :size="12" /> 打开</button>
            <button class="btn sm" @click="openEdit"><Icon name="edit" :size="12" /> 编辑</button>
            <button class="btn sm" @click="openAgents"><Icon name="agent" :size="12" /> 配置 Agent</button>
            <button class="btn sm danger" @click="askRemoveProject">移除</button>
          </div>
        </div>

        <div class="project-agents">
          <span class="section-label">项目内 Agent</span>
          <span v-if="!selected.agents?.length" class="muted">未配置，先「配置 Agent」</span>
          <span v-for="agent in selected.agents ?? []" :key="agent.id" class="badge">
            <i class="agent-dot" :style="{ background: agent.color }">{{ agent.short }}</i>{{ agent.name }}
          </span>
        </div>

        <div class="project-skill-head">
          <div>
            <h3>项目技能 <span class="badge">{{ projectSkills.length }}</span></h3>
            <p>链接/复制到项目目录，项目之间互不影响。</p>
          </div>
          <button class="btn primary" :disabled="!selected.agents?.length" @click="openSkillModal">
            <Icon name="plus" :size="13" /> 添加技能
          </button>
        </div>

        <div v-if="!projectSkills.length" class="project-empty detail-empty">
          <Icon name="layers" :size="18" />
          <b>项目还没有技能</b>
          <span>添加后只会写入该项目选择 Agent 的项目级目录。</span>
          <button class="btn sm" :disabled="!selected.agents?.length" @click="openSkillModal">添加技能</button>
        </div>
        <div v-else class="project-skill-table">
          <div class="ps-row ps-head"><span>#</span><span>技能</span><span>方式</span><span>Agent</span><span>状态</span><span></span></div>
          <div v-for="(skill, index) in projectSkills" :key="skill.slug" class="ps-row">
            <span class="ps-idx">{{ index + 1 }}</span>
            <span class="ps-main">
              <b>{{ skill.name }}</b>
              <em>{{ skill.description }}</em>
              <small class="mono">v{{ skill.version || "—" }}</small>
            </span>
            <span class="badge">{{ modes(skill) }}</span>
            <span class="ps-agents">
              <i v-for="agent in skill.agents ?? []" :key="agent.agentId" class="agent-avatar" :style="{ background: agent.color }" :title="`${agent.agentName} · ${agent.mode}`">{{ agent.short }}</i>
            </span>
            <span class="badge" :class="{ warn: (skill.agents ?? []).some((agent) => agent.status !== 'ok') }">
              {{ (skill.agents ?? []).some((agent) => agent.status !== 'ok') ? "有漂移" : "正常" }}
            </span>
            <button class="ps-remove" title="从项目移除" @click="askRemoveSkill(skill)"><Icon name="x" :size="12" /></button>
          </div>
        </div>
      </section>

      <section v-else class="project-detail project-placeholder">
        <div class="empty">
          <div class="empty-ico"><Icon name="projects" :size="19" /></div>
          <h3>选择一个项目</h3>
          <p>左侧选择项目，或新建一个项目开始。</p>
          <button class="btn" @click="openCreate">新建项目</button>
        </div>
      </section>
    </div>

    <!-- project create/edit -->
    <div v-if="projectModal.open" class="proj-overlay" @click.self.stop>
      <div class="proj-modal">
        <header class="proj-modal-head">
          <div><h2>{{ projectModal.mode === "create" ? "新建项目" : "编辑项目" }}</h2><p>项目目录只用于项目级技能，不修改全局 Agent 配置。</p></div>
          <button class="icon-btn" @click="projectModal.open = false"><Icon name="x" :size="16" /></button>
        </header>
        <div class="proj-modal-body">
          <label class="field"><span>项目名称</span><input v-model="projectModal.name" class="input" placeholder="例如：客户官网" /></label>
          <label class="field">
            <span>项目目录</span>
            <div class="input-with-btn">
              <input v-model="projectModal.path" class="input mono" :disabled="projectModal.mode === 'edit'" placeholder="D:\\work\\my-project" />
              <button v-if="projectModal.mode === 'create'" class="btn sm" @click="browseProject">浏览…</button>
            </div>
          </label>
          <label class="field"><span>简介（可选）</span><input v-model="projectModal.description" class="input" placeholder="一句话说明这个项目" /></label>
          <div v-if="projectModal.mode === 'create'" class="field">
            <span>初始 Agent <em>可稍后配置</em></span>
            <div class="agent-pick">
              <button v-for="agent in availableAgents" :key="agent.id" class="agent-option" :class="{ on: projectModalAgents.includes(agent.id) }" @click="projectModalAgents.includes(agent.id) ? projectModalAgents = projectModalAgents.filter((id) => id !== agent.id) : projectModalAgents.push(agent.id)">
                <i class="agent-avatar" :style="{ background: agent.color }">{{ agent.short }}</i>{{ agent.name }}
              </button>
              <span v-if="!availableAgents.length" class="muted" style="font-size: 10px">未检测到 Agent 目录，可先创建项目。</span>
            </div>
          </div>
        </div>
        <footer class="proj-modal-foot">
          <button class="btn" :disabled="busy" @click="projectModal.open = false">取消</button>
          <button class="btn primary" :disabled="busy" @click="saveProject">{{ busy ? "保存中…" : "保存" }}</button>
        </footer>
      </div>
    </div>

    <!-- project agents -->
    <div v-if="agentModal" class="proj-overlay" @click.self.stop>
      <div class="proj-modal small">
        <header class="proj-modal-head">
          <div><h2>配置项目 Agent</h2><p>只把技能写入这些 Agent 的项目级 skill 目录。</p></div>
          <button class="icon-btn" @click="agentModal = false"><Icon name="x" :size="16" /></button>
        </header>
        <div class="proj-modal-body">
          <div class="agent-pick">
            <button v-for="agent in availableAgents" :key="agent.id" class="agent-option" :class="{ on: agentDraft.includes(agent.id) }" @click="toggleAgent(agent.id)">
              <i class="agent-avatar" :style="{ background: agent.color }">{{ agent.short }}</i>{{ agent.name }}
            </button>
            <span v-if="!availableAgents.length" class="muted">没有检测到 Agent 目录。</span>
          </div>
        </div>
        <footer class="proj-modal-foot">
          <button class="btn" :disabled="busy" @click="agentModal = false">取消</button>
          <button class="btn primary" :disabled="busy" @click="saveAgents">{{ busy ? "保存中…" : "保存" }}</button>
        </footer>
      </div>
    </div>

    <!-- add skills -->
    <div v-if="skillModal" class="proj-overlay" @click.self.stop>
      <div class="proj-modal">
        <header class="proj-modal-head">
          <div><h2>添加技能</h2><p>已加入的技能会自动跳过；写入项目级目录。</p></div>
          <button class="icon-btn" @click="skillModal = false"><Icon name="x" :size="16" /></button>
        </header>
        <div class="proj-modal-toolbar">
          <div class="search"><Icon name="search" :size="13" /><input v-model="skillQuery" class="input" placeholder="搜索技能…" /></div>
          <div class="mode-switch">
            <button :class="{ active: skillMode === 'link' }" @click="skillMode = 'link'"><Icon name="link" :size="11" /> 链接</button>
            <button :class="{ active: skillMode === 'copy' }" @click="skillMode = 'copy'"><Icon name="file" :size="11" /> 复制</button>
          </div>
        </div>
        <div class="skill-pick-list">
          <button v-for="skill in filteredSkills" :key="skill.slug" class="skill-pick" :class="{ on: skillDraft.includes(skill.slug) }" @click="toggleSkill(skill.slug)">
            <span class="pick-check"><Icon v-if="skillDraft.includes(skill.slug)" name="check" :size="10" /></span>
            <span class="pick-main"><b>{{ skill.name }}</b><em>{{ skill.description }}</em></span>
            <span class="mono">{{ skill.version ? `v${skill.version}` : "" }}</span>
          </button>
          <div v-if="!filteredSkills.length" class="empty-mini">仓库里没有匹配的技能。</div>
        </div>
        <footer class="proj-modal-foot">
          <span class="muted">已选 {{ skillDraft.length }} 个 · {{ selectedAgentIDs.length }} 个 Agent</span>
          <span class="grow" />
          <button class="btn" :disabled="busy" @click="skillModal = false">取消</button>
          <button class="btn primary" :disabled="busy || !skillDraft.length" @click="saveSkills">{{ busy ? "添加中…" : "添加" }}</button>
        </footer>
      </div>
    </div>

    <!-- confirm -->
    <div v-if="confirm" class="confirm-overlay" @click.self.stop>
      <div class="confirm-card">
        <div class="confirm-head">
          <div><h3>{{ confirm.title }}</h3><p>{{ confirm.message }}</p></div>
          <button class="icon-btn" :disabled="busy" @click="confirm = null"><Icon name="x" :size="15" /></button>
        </div>
        <div class="confirm-foot">
          <button class="btn" :disabled="busy" @click="confirm = null">取消</button>
          <button class="btn danger" :disabled="busy" @click="runConfirm">{{ busy ? "处理中…" : "确认" }}</button>
        </div>
      </div>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>
  </div>
</template>

<style scoped>
.projects-layout { display: grid; grid-template-columns: 270px minmax(0, 1fr); gap: 14px; align-items: start; }
.project-side { display: flex; flex-direction: column; gap: 5px; min-width: 0; }
.project-card { display: flex; align-items: flex-start; gap: 8px; width: 100%; padding: 9px; border: 1px solid transparent; border-radius: 8px; text-align: left; }
.project-card:hover { background: var(--panel-2); }
.project-card.active { border-color: var(--border); background: var(--panel); box-shadow: var(--shadow-sm); }
.project-ava { display: grid; place-items: center; width: 26px; height: 26px; border-radius: 7px; background: var(--accent-soft); color: var(--accent); font-size: 10px; font-weight: 700; flex: none; }
.project-info { display: flex; flex-direction: column; min-width: 0; gap: 2px; }
.project-info b { font-size: 11.5px; font-weight: 600; }
.project-info em { color: var(--text-4); font-size: 8.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.project-info small { color: var(--text-4); font-size: 9px; }
.project-empty { display: flex; flex-direction: column; align-items: center; gap: 5px; padding: 18px 8px; color: var(--text-4); font-size: 10px; text-align: center; }
.project-empty b { color: var(--text-2); font-size: 11px; }
.project-detail { min-width: 0; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); overflow: hidden; }
.project-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px 16px 10px; }
.project-title { display: flex; flex-direction: column; gap: 3px; min-width: 0; flex: 1; }
.project-title h2 { margin: 0; font-size: 17px; }
.project-title .mono { color: var(--text-3); font-size: 9.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.project-title .muted { color: var(--text-4); font-size: 9px; }
.project-actions { display: flex; gap: 4px; flex-wrap: wrap; justify-content: flex-end; flex: none; }
.project-agents { display: flex; align-items: center; gap: 5px; flex-wrap: wrap; padding: 0 16px 12px; }
.section-label { color: var(--text-4); font-size: 9px; margin-right: 3px; }
.agent-dot { display: inline-grid; place-items: center; width: 14px; height: 14px; border-radius: 4px; color: #fff; font-style: normal; font-size: 6.5px; margin-right: 3px; }
.project-skill-head { display: flex; align-items: center; gap: 10px; padding: 11px 16px 8px; border-top: 1px solid var(--border-soft); }
.project-skill-head h3 { margin: 0; font-size: 13px; display: flex; gap: 6px; align-items: center; }
.project-skill-head p { margin: 3px 0 0; color: var(--text-4); font-size: 9.5px; }
.project-skill-head .btn { margin-left: auto; }
.project-skill-table { padding: 0 16px 12px; }
.ps-row { display: grid; grid-template-columns: 20px minmax(160px, 1fr) 75px 70px 60px 22px; gap: 7px; align-items: center; padding: 7px 0; border-bottom: 1px dashed var(--border-soft); min-width: 0; }
.ps-row:last-child { border-bottom: 0; }
.ps-head { color: var(--text-4); font-size: 9px; border-bottom-style: solid; }
.ps-idx { color: var(--text-4); font-size: 9px; }
.ps-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.ps-main b { font-size: 11px; font-weight: 550; }
.ps-main em { color: var(--text-4); font-style: normal; font-size: 9px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ps-main small { color: var(--text-3); font-size: 8px; }
.ps-agents { display: flex; gap: 2px; }
.agent-avatar { display: inline-grid; place-items: center; width: 20px; height: 20px; border-radius: 5px; color: #fff; font-style: normal; font-size: 7px; font-weight: 700; }
.ps-remove { color: var(--text-4); padding: 2px; border-radius: 4px; }
.ps-remove:hover { color: var(--danger); background: var(--danger-soft); }
.detail-empty { padding: 36px 8px; }
.proj-overlay { position: fixed; inset: 0; z-index: 150; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.proj-modal { width: min(580px, calc(100vw - 32px)); max-height: min(680px, calc(100vh - 40px)); display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); overflow: hidden; }
.proj-modal.small { width: min(420px, calc(100vw - 32px)); }
.proj-modal-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px 16px 11px; border-bottom: 1px solid var(--border-soft); }
.proj-modal-head h2 { margin: 0; font-size: 15px; }
.proj-modal-head p { margin: 4px 0 0; color: var(--text-4); font-size: 10px; }
.proj-modal-head .icon-btn { margin-left: auto; }
.proj-modal-body { padding: 12px 16px; overflow: auto; display: flex; flex-direction: column; gap: 10px; }
.field { display: flex; flex-direction: column; gap: 4px; font-size: 10px; color: var(--text-3); }
.field em { color: var(--text-4); font-style: normal; font-size: 9px; margin-left: 4px; }
.input-with-btn { display: flex; gap: 5px; }
.input-with-btn .input { flex: 1; min-width: 0; }
.agent-pick { display: flex; flex-wrap: wrap; gap: 4px; }
.agent-option { display: inline-flex; align-items: center; gap: 4px; padding: 4px 7px; border: 1px solid var(--border); border-radius: 6px; font-size: 10px; color: var(--text-2); }
.agent-option.on { border-color: var(--accent); background: var(--accent-soft); color: var(--accent); }
.proj-modal-foot { display: flex; align-items: center; gap: 6px; padding: 10px 16px; border-top: 1px solid var(--border-soft); flex: none; }
.proj-modal-toolbar { display: flex; gap: 8px; align-items: center; padding: 9px 16px; border-bottom: 1px solid var(--border-soft); }
.proj-modal-toolbar .search { flex: 1; }
.mode-switch { display: flex; gap: 2px; padding: 2px; border: 1px solid var(--border); border-radius: 7px; background: var(--panel-2); }
.mode-switch button { display: inline-flex; align-items: center; gap: 3px; padding: 4px 7px; border-radius: 5px; color: var(--text-4); font-size: 9.5px; }
.mode-switch button.active { color: var(--accent); background: var(--accent-soft); }
.skill-pick-list { overflow: auto; padding: 5px 10px; max-height: 360px; }
.skill-pick { display: flex; align-items: center; gap: 7px; width: 100%; padding: 7px 6px; text-align: left; border-bottom: 1px dashed var(--border-soft); }
.skill-pick.on { background: var(--accent-soft); }
.pick-check { display: grid; place-items: center; width: 15px; height: 15px; border: 1px solid var(--border); border-radius: 4px; color: #fff; flex: none; }
.skill-pick.on .pick-check { background: var(--accent); border-color: var(--accent); }
.pick-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
.pick-main b { font-size: 11px; font-weight: 550; }
.pick-main em { color: var(--text-4); font-style: normal; font-size: 9px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.empty-mini { padding: 18px; color: var(--text-4); font-size: 10px; text-align: center; }
.project-placeholder { min-height: 320px; display: grid; place-items: center; }
.confirm-overlay { position: fixed; inset: 0; z-index: 200; display: flex; align-items: center; justify-content: center; background: rgba(20,22,27,.32); backdrop-filter: blur(3px); }
.confirm-card { width: min(420px, calc(100vw - 32px)); border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--panel); box-shadow: var(--shadow-lg); }
.confirm-head { display: flex; align-items: flex-start; gap: 10px; padding: 14px; }
.confirm-head h3 { margin: 0; font-size: 14px; }
.confirm-head p { margin: 5px 0 0; color: var(--text-3); font-size: 11px; line-height: 1.6; }
.confirm-head .icon-btn { margin-left: auto; flex: none; }
.confirm-foot { display: flex; justify-content: flex-end; gap: 6px; padding: 10px 14px; border-top: 1px solid var(--border-soft); }
@media (max-width: 860px) { .projects-layout { grid-template-columns: 220px minmax(0, 1fr); } .ps-row { grid-template-columns: 16px minmax(120px, 1fr) 60px 50px 50px 20px; } }
</style>
