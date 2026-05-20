<script setup lang="ts">
import { ref } from "vue";
import type { SyncStatus } from "../api";

const props = defineProps<{
  theme: string;
  accent: string;
  syncStatus: SyncStatus;
  intervalSeconds: number;
}>();
const emit = defineEmits<{
  (e: "set", patch: { theme?: string; accent?: string; sync_interval_seconds?: number }): void;
  (e: "sync-now"): void;
}>();

const open = ref(true);
const intervalMinutes = ref(Math.max(1, Math.round((props.intervalSeconds || 600) / 60)));

const ACCENTS = ["#f5a524", "#ef4444", "#10b981", "#3b82f6", "#a855f7", "#ec4899"];

function setIntervalMinutes() {
  emit("set", { sync_interval_seconds: Math.max(30, intervalMinutes.value * 60) });
}

function lastSyncLabel(): string {
  if (!props.syncStatus.last_sync_at) return "never";
  const diff = Date.now() - props.syncStatus.last_sync_at;
  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  return `${Math.floor(diff / 3_600_000)}h ago`;
}
</script>

<template>
  <div v-if="open" class="tweaks">
    <div class="tweaks-hd">
      <span>Tweaks</span>
      <button @click="open = false">✕</button>
    </div>
    <div class="tweaks-body">
      <div class="tw-row">
        <span class="tw-lbl">Theme</span>
        <div class="tw-seg">
          <button v-for="opt in ['light', 'dark']" :key="opt" :class="{ on: theme === opt }" @click="$emit('set', { theme: opt })">{{ opt }}</button>
        </div>
      </div>
      <div class="tw-row">
        <span class="tw-lbl">Accent</span>
        <div class="tw-swatches">
          <button v-for="c in ACCENTS" :key="c" :class="['tw-sw', { on: accent === c }]" :style="{ background: c }" @click="$emit('set', { accent: c })"></button>
        </div>
      </div>
      <div class="tw-row">
        <span class="tw-lbl">Sync every</span>
        <div class="tw-int">
          <input type="number" min="1" max="120" v-model.number="intervalMinutes" @change="setIntervalMinutes" />
          <span class="dim mono">min</span>
        </div>
      </div>
      <div class="tw-sync">
        <button class="tw-btn" @click="$emit('sync-now')" :disabled="syncStatus.running">
          {{ syncStatus.running ? 'syncing…' : '↻ Sync now' }}
        </button>
        <div class="dim mono tw-sync-meta">
          last: {{ lastSyncLabel() }}
          <span v-if="syncStatus.last_error" class="tw-err" :title="syncStatus.last_error">· error</span>
        </div>
      </div>
    </div>
  </div>
  <button v-else class="tweaks-fab" @click="open = true" title="Tweaks">⚙</button>
</template>

<style scoped>
.tw-int { display: inline-flex; align-items: center; gap: 4px; }
.tw-int input {
  width: 56px;
  height: 22px;
  border-radius: 4px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 0 6px;
  font-family: "Geist Mono", monospace;
  font-size: 11px;
  outline: none;
}
.tw-int input:focus { border-color: var(--accent); }
.tw-sync { display: flex; flex-direction: column; gap: 4px; }
.tw-sync-meta { font-size: 10.5px; }
.tw-err { color: var(--filter-no); margin-left: 4px; }
.tweaks-fab {
  position: fixed; right: 16px; bottom: 40px;
  width: 32px; height: 32px;
  border-radius: 16px;
  background: var(--surface);
  border: 1px solid var(--border-strong);
  color: var(--text-2);
  font-size: 14px;
  z-index: 100;
  box-shadow: 0 4px 12px rgba(0,0,0,0.2);
}
.tweaks-fab:hover { color: var(--text); }
</style>
