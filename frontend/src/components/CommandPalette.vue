<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { routes } from "../router";
import Icon from "./Icon.vue";

const props = defineProps<{ open: boolean }>();
const emit = defineEmits<{ close: [] }>();
const router = useRouter();
const query = ref("");
const active = ref(0);
const input = ref<HTMLInputElement | null>(null);

const items = computed(() => {
  const q = query.value.trim().toLowerCase();
  return routes
    .filter((route) => route.name && route.meta)
    .map((route) => ({
      path: route.path,
      title: String(route.meta?.title ?? ""),
      group: String(route.meta?.group ?? "工作区"),
      phase: String(route.meta?.phase ?? ""),
      icon: route.name === "library" ? "library"
        : route.name === "market" ? "market"
          : route.name === "agent" ? "agent"
            : route.name === "projects" ? "projects"
              : route.name === "prompts" ? "prompts" : "settings",
    }))
    .filter((item) => !q || item.title.toLowerCase().includes(q) || item.path.includes(q));
});

watch(() => props.open, async (open) => {
  if (!open) return;
  query.value = "";
  active.value = 0;
  await nextTick();
  input.value?.focus();
});

watch(query, () => { active.value = 0; });

function move(delta: number) {
  if (!items.value.length) return;
  active.value = (active.value + delta + items.value.length) % items.value.length;
}

function select(index = active.value) {
  const item = items.value[index];
  if (!item) return;
  router.push(item.path);
  emit("close");
}
</script>

<template>
  <div v-if="open" class="overlay" @click.self.stop>
    <div class="palette" role="dialog" aria-label="命令面板">
      <div class="palette-input">
        <Icon name="search" :size="15" />
        <input
          ref="input"
          v-model="query"
          placeholder="搜索页面、动作…"
          @keydown.esc.prevent="emit('close')"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.enter.prevent="select()"
        />
        <span class="kbd">ESC</span>
      </div>
      <div class="palette-list">
        <button
          v-for="(item, index) in items"
          :key="item.path"
          class="palette-item"
          :class="{ active: index === active }"
          @mouseenter="active = index"
          @click="select(index)"
        >
          <Icon :name="item.icon" :size="14" />
          <span>{{ item.title }}</span>
          <span class="pi-sub">{{ item.group }} · {{ item.phase }}</span>
        </button>
        <div v-if="!items.length" class="palette-empty">没有匹配项</div>
      </div>
    </div>
  </div>
</template>
