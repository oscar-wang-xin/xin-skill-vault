import { createRouter, createWebHashHistory } from "vue-router";
import LibraryView from "../views/LibraryView.vue";
import MarketView from "../views/MarketView.vue";
import AgentView from "../views/AgentView.vue";
import ProjectsView from "../views/ProjectsView.vue";
import PromptsView from "../views/PromptsView.vue";
import SettingsView from "../views/SettingsView.vue";

export const routes = [
  { path: "/", redirect: "/library" },
  { path: "/library", name: "library", component: LibraryView, meta: { title: "技能库", group: "工作区", phase: "M1" } },
  { path: "/market", name: "market", component: MarketView, meta: { title: "市场", group: "工作区", phase: "M3" } },
  { path: "/agent", name: "agent", component: AgentView, meta: { title: "Agent", group: "工作区", phase: "M2" } },
  { path: "/projects", name: "projects", component: ProjectsView, meta: { title: "项目", group: "工作区", phase: "M2" } },
  { path: "/prompts", name: "prompts", component: PromptsView, meta: { title: "提示词", group: "工作区", phase: "M4" } },
  { path: "/settings", name: "settings", component: SettingsView, meta: { title: "设置", group: "系统", phase: "M0" } },
];

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
});
