<script setup lang="ts">
import { ref, watch, nextTick } from "vue";
import type { TaskTag } from "../api";

const props = defineProps<{
  tags: TaskTag[];
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
    const merged = Array.from(new Set([...props.tags.map((t) => t.name), ...next]));
    emit("change", merged);
  }
  draft.value = "";
  editing.value = false;
}

function startEdit(e: MouseEvent) {
  e.stopPropagation();
  editing.value = true;
}

function remove(name: string) {
  emit("change", props.tags.filter((x) => x.name !== name).map((x) => x.name));
}

function stateOf(name: string) {
  return props.tagFilter?.[name];
}
</script>

<template>
  <span class="tags">
    <span v-for="tg in tags" :key="tg.name" class="tag-group" :style="{ '--tag-h': tagHue(tg.name) }">
      <button
        :class="['tag', { 'tag-include': stateOf(tg.name) === 'include', 'tag-exclude': stateOf(tg.name) === 'exclude', 'tag-local': tg.origin === 'local' }]"
        @click.stop="$emit('tag-click', tg.name)"
        @contextmenu.prevent.stop="($event as MouseEvent).shiftKey ? remove(tg.name) : $emit('tag-exclude', tg.name)"
        :title="'#' + tg.name + '  ·  click: filter cycle  ·  right-click: toggle exclude' + (tg.origin === 'local' ? '  ·  × to remove' : '')"
      >{{ tg.name }}</button>
      <button
        v-if="tg.origin === 'local'"
        class="tag-remove"
        @click.stop="remove(tg.name)"
        title="Remove tag"
      >×</button>
    </span>
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
