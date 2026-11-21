import { defineStore } from "pinia";
import { ref, watch } from "vue";

export type ThemeDefinition = {
  id: string;
  name: string;
  colors: string[];
};

export const themes: ThemeDefinition[] = [
  { id: "mech-gray", name: "机甲冷灰", colors: ["#3f4653", "#f6f7f8", "#e4e6ea"] },
  { id: "business-blue", name: "商务蓝", colors: ["#2563eb", "#f6f8fc", "#dfe7f5"] },
  { id: "mint", name: "薄荷绿", colors: ["#0e9f74", "#f5faf8", "#d9efe7"] },
  { id: "alibaba-orange", name: "阿里橙", colors: ["#f0610a", "#fbf8f5", "#f6e2d3"] },
  { id: "grape", name: "葡萄紫", colors: ["#7c3aed", "#f9f8fe", "#e7dffb"] },
  { id: "mech-amber", name: "机甲橙黄", colors: ["#d97706", "#faf8f3", "#f2e6cd"] },
];

const THEME_KEY = "xsv-theme";
const NEUMORPH_KEY = "xsv-neumorph";
const SIDEBAR_KEY = "xsv-sidebar-collapsed";

function readTheme(): string {
  const saved = localStorage.getItem(THEME_KEY);
  return themes.some((theme) => theme.id === saved) ? saved! : "mech-gray";
}

export const useUiStore = defineStore("ui", () => {
  const theme = ref(readTheme());
  const neumorph = ref(localStorage.getItem(NEUMORPH_KEY) !== "off");
  const sidebarCollapsed = ref(localStorage.getItem(SIDEBAR_KEY) === "true");
  const themeMenuOpen = ref(false);
  const commandOpen = ref(false);

  function apply() {
    document.documentElement.dataset.theme = theme.value;
    document.documentElement.dataset.neumorph = neumorph.value ? "on" : "off";
  }

  function setTheme(next: string) {
    if (!themes.some((item) => item.id === next)) return;
    theme.value = next;
    localStorage.setItem(THEME_KEY, next);
    themeMenuOpen.value = false;
    apply();
  }

  function toggleNeumorph() {
    neumorph.value = !neumorph.value;
    localStorage.setItem(NEUMORPH_KEY, neumorph.value ? "on" : "off");
    apply();
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value;
    localStorage.setItem(SIDEBAR_KEY, String(sidebarCollapsed.value));
  }

  function applyConfig(next: { theme: string; neumorph: boolean }) {
    if (themes.some((item) => item.id === next.theme)) theme.value = next.theme;
    neumorph.value = next.neumorph;
    localStorage.setItem(THEME_KEY, theme.value);
    localStorage.setItem(NEUMORPH_KEY, neumorph.value ? "on" : "off");
    apply();
  }

  watch([theme, neumorph], apply, { immediate: true });

  return {
    theme,
    neumorph,
    sidebarCollapsed,
    themeMenuOpen,
    commandOpen,
    setTheme,
    toggleNeumorph,
    toggleSidebar,
    applyConfig,
  };
});
