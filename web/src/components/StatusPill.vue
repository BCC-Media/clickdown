<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import type { Status } from "../api";

const props = defineProps<{
  status: string;
  statuses: Status[];
}>();
const emit = defineEmits<{ (e: "change", value: string): void }>();

const open = ref(false);
const wrap = ref<HTMLElement | null>(null);

function handler(e: MouseEvent) {
  if (!wrap.value?.contains(e.target as Node)) open.value = false;
}
onMounted(() => document.addEventListener("mousedown", handler));
onUnmounted(() => document.removeEventListener("mousedown", handler));

function glyphFor(type: string): string {
  switch (type) {
    case "closed": return "●";
    case "open": return "○";
    default: return "◐";
  }
}

function statusObj(name: string): Status | undefined {
  return props.statuses.find((s) => s.name === name);
}

function pick(name: string) {
  emit("change", name);
  open.value = false;
}
</script>

<template>
  <span class="pill-wrap" ref="wrap">
    <button
      class="pill pill-dyn"
      :style="{ '--st-c': statusObj(status)?.color || '#888' }"
      :class="{ 'st-closed': statusObj(status)?.type === 'closed' }"
      @click.stop="open = !open"
      :title="'Status: ' + status"
    >
      <span class="pill-glyph">{{ glyphFor(statusObj(status)?.type || '') }}</span>
      <span class="pill-name">{{ status }}</span>
    </button>
    <div v-if="open" class="menu" @click.stop>
      <button
        v-for="(s, idx) in statuses"
        :key="s.name"
        class="menu-item pill-dyn"
        :style="{ '--st-c': s.color }"
        :class="{ on: s.name === status }"
        @click="pick(s.name)"
      >
        <span class="menu-glyph">{{ glyphFor(s.type) }}</span>
        <span class="menu-label">{{ s.name }}</span>
        <span v-if="idx < 9" class="menu-kbd">{{ idx + 1 }}</span>
      </button>
    </div>
  </span>
</template>
