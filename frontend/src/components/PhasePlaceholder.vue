<script setup lang="ts">
import Icon from "./Icon.vue";

withDefaults(defineProps<{
  phase: string;
  icon?: string;
  title: string;
  description: string;
  features?: string[];
}>(), { icon: "layers", features: () => [] });
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>{{ title }} <span class="badge phase">{{ phase }}</span></h1>
        <p>{{ description }}</p>
      </div>
    </div>
    <div class="empty">
      <div class="empty-ico"><Icon :name="icon" :size="19" /></div>
      <h3>界面骨架已就位</h3>
      <p>{{ description }}</p>
      <ul v-if="features.length" class="feature-list">
        <li v-for="feature in features" :key="feature">{{ feature }}</li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.feature-list {
  margin: 14px 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px;
}
.feature-list li {
  padding: 3px 8px;
  border-radius: 99px;
  color: var(--text-3);
  background: var(--panel-3);
  font-size: 10px;
}
</style>
