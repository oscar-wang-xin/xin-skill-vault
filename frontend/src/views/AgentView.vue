<script setup lang="ts">
import PhasePlaceholder from "../components/PhasePlaceholder.vue";
</script>

<template>
  <PhasePlaceholder
    phase="M2"
    icon="agent"
    title="Agent"
    description="自动检测本机 Agent 的 skill 目录，管理链接/复制分发与批量导入。"
    :features="['本机 Agent 适配器', '链接 / Junction / 复制降级', '批量导入与删除', '刷新目录']"
  />
</template>
