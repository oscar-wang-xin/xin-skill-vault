<script setup lang="ts">
import PhasePlaceholder from "../components/PhasePlaceholder.vue";
</script>

<template>
  <PhasePlaceholder
    phase="M3"
    icon="market"
    title="市场"
    description="从 skills.sh / SkillsMP / SkillHub 搜索并安装技能，保留来源、版本与缓存。"
    :features="['Provider 适配器', 'ETag 缓存与限速', '安装前安全检查', '来源追踪']"
  />
</template>
