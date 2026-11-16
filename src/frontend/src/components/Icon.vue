<script setup lang="ts">
defineProps<{
  name: string
  size?: 'sm' | 'lg'
}>()
</script>

<template>
  <svg class="ic" :class="size" aria-hidden="true">
    <use :href="`#i-${name}`" />
  </svg>
</template>
