<script setup lang="ts">
import PhasePlaceholder from "../components/PhasePlaceholder.vue";
</script>

<template>
  <PhasePlaceholder
    phase="M2"
    icon="projects"
    title="项目"
    description="为一个项目绑定 Agent 与技能集合，查看分配方式与链接健康状态。"
    :features="['项目级 Agent 配置', '技能多选与批量移除', '链接 / 复制模式', '项目目录树']"
  />
</template>
