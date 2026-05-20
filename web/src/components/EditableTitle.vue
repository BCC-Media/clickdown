<script setup lang="ts">
import { ref, watch, nextTick } from "vue";

const props = defineProps<{ value: string; editing: boolean }>();
const emit = defineEmits<{
  (e: "commit", value: string): void;
  (e: "stop-edit"): void;
}>();

const el = ref<HTMLElement | null>(null);

watch(
  () => props.editing,
  async (v) => {
    if (v) {
      await nextTick();
      if (!el.value) return;
      el.value.focus();
      const r = document.createRange();
      r.selectNodeContents(el.value);
      r.collapse(false);
      const sel = window.getSelection();
      sel?.removeAllRanges();
      sel?.addRange(r);
    }
  }
);

function onBlur(e: FocusEvent) {
  const t = e.target as HTMLElement;
  const v = (t.textContent || "").trim();
  if (v && v !== props.value) emit("commit", v);
  else t.textContent = props.value;
  emit("stop-edit");
}

function onKey(e: KeyboardEvent) {
  if (e.key === "Enter") {
    e.preventDefault();
    (e.target as HTMLElement).blur();
  } else if (e.key === "Escape") {
    e.preventDefault();
    (e.target as HTMLElement).textContent = props.value;
    emit("stop-edit");
    (e.target as HTMLElement).blur();
  }
  e.stopPropagation();
}
</script>

<template>
  <span
    ref="el"
    class="row-title-text"
    :contenteditable="editing"
    spellcheck="false"
    @blur="onBlur"
    @keydown="onKey"
  >{{ value }}</span>
</template>
