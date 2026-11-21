<script setup lang="ts">
import PhasePlaceholder from "../components/PhasePlaceholder.vue";
</script>

<template>
  <PhasePlaceholder
    phase="M4"
    icon="prompts"
    title="提示词"
    description="分类树、Markdown 编辑与版本历史；每个版本保存为独立 .md 文档。"
    :features="['分类树 / 同级拖动', 'Markdown 工具栏', '版本历史与删除', '.md 文件存储']"
  />
</template>
