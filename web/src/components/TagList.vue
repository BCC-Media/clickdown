<script setup lang="ts">
import { ref, watch, nextTick } from "vue";

const props = defineProps<{
  tags: string[];
  tagFilter: Record<string, "include" | "exclude" | undefined>;
}>();
const emit = defineEmits<{
  (e: "change", value: string[]): void;
  (e: "tag-click", tag: string): void;
  (e: "tag-exclude", tag: string): void;
}>();

const editing = ref(false);
const draft = ref("");
const input = ref<HTMLInputElement | null>(null);

watch(editing, async (v) => {
  if (v) {
    await nextTick();
    input.value?.focus();
  }
});

function tagHue(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h % 360;
}

function commit() {
  const next = draft.value.split(/[,\s]+/).map((s) => s.trim()).filter(Boolean);
  if (next.length) {
    const merged = Array.from(new Set([...props.tags, ...next]));
    emit("change", merged);
  }
  draft.value = "";
  editing.value = false;
}

function startEdit(e: MouseEvent) {
  e.stopPropagation();
  editing.value = true;
}

function remove(tg: string) {
  emit("change", props.tags.filter((x) => x !== tg));
}

function stateOf(tg: string) {
  return props.tagFilter?.[tg];
}
</script>

<template>
  <span class="tags">
    <button
      v-for="tg in tags"
      :key="tg"
      :class="['tag', { 'tag-include': stateOf(tg) === 'include', 'tag-exclude': stateOf(tg) === 'exclude' }]"
      :style="{ '--tag-h': tagHue(tg) }"
      @click.stop="$emit('tag-click', tg)"
      @contextmenu.prevent.stop="($event as MouseEvent).shiftKey ? remove(tg) : $emit('tag-exclude', tg)"
      :title="'#' + tg + '  ·  click: filter cycle  ·  right-click: toggle exclude  ·  shift+right-click: remove'"
    >{{ tg }}</button>
    <input
      v-if="editing"
      ref="input"
      class="tag-input mono"
      v-model="draft"
      placeholder="tag…"
      @keydown.enter.prevent="commit"
      @keydown.escape="draft = ''; editing = false"
      @keydown.stop
      @blur="commit"
      :size="Math.max(6, draft.length + 1)"
    />
    <button v-else class="tag-add" @click.stop="startEdit($event)" title="Add tag">+</button>
  </span>
</template>
