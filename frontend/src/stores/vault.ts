import { defineStore } from "pinia";
import { ref } from "vue";
import { getAppInfo, getVaultSummary, errorMessage } from "../lib/api";
import type { AppInfo, VaultSummary } from "../../bindings/xinskillvault/internal/app/models";

export const useVaultStore = defineStore("vault", () => {
  const info = ref<AppInfo | null>(null);
  const summary = ref<VaultSummary | null>(null);
  const loading = ref(false);
  const error = ref("");
  let inflight: Promise<void> | null = null;

  async function load() {
    try {
      const [nextInfo, nextSummary] = await Promise.all([getAppInfo(), getVaultSummary()]);
      info.value = nextInfo;
      summary.value = nextSummary;
      error.value = "";
      if (nextInfo.name) document.title = nextInfo.name;
    } catch (cause) {
      error.value = errorMessage(cause);
    }
  }

  async function refresh() {
    if (inflight) return inflight;
    loading.value = true;
    inflight = load().finally(() => {
      loading.value = false;
      inflight = null;
    });
    return inflight;
  }

  return { info, summary, loading, error, refresh };
});
