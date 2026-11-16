<script setup lang="ts">
import { dismissToast, state } from '../lib/appState'
import Icon from './Icon.vue'

const iconFor = (kind: string) => (kind === 'error' ? 'alert' : kind === 'warning' ? 'alert' : 'check-circle')
</script>

<template>
  <div class="toasts" aria-live="polite">
    <TransitionGroup name="toast">
      <div
        v-for="toast in state.toasts"
        :key="toast.id"
        class="toast"
        :class="toast.kind"
        role="status"
        @click="dismissToast(toast.id)"
      >
        <span class="toast-icon"><Icon :name="iconFor(toast.kind)" /></span>
        <div class="toast-copy">
          <div class="toast-title">{{ toast.title }}</div>
          <div v-if="toast.detail" class="toast-detail">{{ toast.detail }}</div>
        </div>
        <button class="toast-close" aria-label="关闭提示" @click.stop="dismissToast(toast.id)">
          <Icon name="x" size="sm" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>
