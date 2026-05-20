<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{ priority: number | null }>();

const meta = computed(() => {
  const v = props.priority || 0;
  switch (v) {
    case 1: return { label: "P0", name: "urgent" };
    case 2: return { label: "P1", name: "high" };
    case 3: return { label: "P2", name: "med" };
    case 4: return { label: "P3", name: "low" };
    default: return { label: "—", name: "none" };
  }
});

const cls = computed(() => `pri pri-${props.priority || 0}`);
</script>

<template>
  <span :class="cls" :title="'Priority: ' + meta.name + ' (read-only)'">{{ meta.label }}</span>
</template>
