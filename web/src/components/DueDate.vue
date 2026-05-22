<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{ due: number | null; closed?: boolean }>();

const DAY_MS = 24 * 60 * 60 * 1000;

function startOfDay(ts: number): number {
  const d = new Date(ts);
  d.setHours(0, 0, 0, 0);
  return d.getTime();
}

const info = computed(() => {
  if (props.due == null) return null;
  const today = startOfDay(Date.now());
  const dueDay = startOfDay(props.due);
  const days = Math.round((dueDay - today) / DAY_MS);

  // bucket: drives color + tone. Closed tasks stay muted regardless.
  let bucket: "overdue" | "today" | "soon" | "week" | "later" = "later";
  if (days < 0) bucket = "overdue";
  else if (days === 0) bucket = "today";
  else if (days <= 2) bucket = "soon";
  else if (days <= 7) bucket = "week";

  let label: string;
  if (days === 0) label = "today";
  else if (days === 1) label = "tomorrow";
  else if (days === -1) label = "yesterday";
  else if (days < 0 && days >= -7) label = `${-days}d ago`;
  else if (days > 0 && days <= 7) label = `${days}d`;
  else {
    const d = new Date(props.due);
    const sameYear = d.getFullYear() === new Date().getFullYear();
    label = d.toLocaleDateString(undefined, sameYear
      ? { month: "short", day: "numeric" }
      : { month: "short", day: "numeric", year: "2-digit" });
  }

  const title = new Date(props.due).toLocaleString(undefined, {
    weekday: "short", month: "short", day: "numeric",
    hour: "numeric", minute: "2-digit",
  });

  return { bucket, label, title };
});
</script>

<template>
  <span
    v-if="info"
    :class="['due', `due-${info.bucket}`, { 'due-closed': closed }]"
    :title="`Due ${info.title}`"
  >
    <span class="due-glyph">⏱</span>{{ info.label }}
  </span>
  <span v-else class="due-empty"></span>
</template>
