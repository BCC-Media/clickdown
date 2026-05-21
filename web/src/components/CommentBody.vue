<script setup lang="ts">
import { computed, ref } from "vue";
import type { CommentBlock } from "../api";

const props = defineProps<{
  blocks?: CommentBlock[] | null;
  text: string;
}>();

// ClickUp returns a heterogeneous array of blocks. For unknown types, fall
// back to the block's text so nothing is silently dropped.
const items = computed(() => props.blocks ?? []);

// ClickUp's CDN requires a logged-in clickup.com session to serve attachments
// (the API token alone isn't accepted there). If the user isn't signed in to
// ClickUp in this browser, the <img> will 404 — track failures so we render
// a click-through link instead of a broken-image icon.
const failed = ref<Record<number, boolean>>({});
function markFailed(i: number) {
  failed.value = { ...failed.value, [i]: true };
}
</script>

<template>
  <div v-if="items.length" class="comment-body">
    <template v-for="(b, i) in items" :key="i">
      <template v-if="b.type === 'image' && (b.image?.url || b.image?.thumbnail_medium)">
        <a
          v-if="failed[i]"
          :href="b.image?.url || b.image?.thumbnail_large || b.image?.thumbnail_medium"
          target="_blank"
          rel="noopener noreferrer"
          class="comment-image-fallback"
        >📎 {{ b.image?.name || b.image?.title || b.text || "image" }}</a>
        <a
          v-else
          :href="b.image?.url || b.image?.thumbnail_large || b.image?.thumbnail_medium"
          target="_blank"
          rel="noopener noreferrer"
          class="comment-image-link"
        >
          <img
            :src="b.image?.thumbnail_medium || b.image?.thumbnail_large || b.image?.url"
            :alt="b.image?.title || b.image?.name || b.text || 'image'"
            loading="lazy"
            @error="markFailed(i)"
          />
        </a>
      </template>
      <span v-else-if="b.type === 'tag'" class="comment-mention">{{ b.text }}</span>
      <span v-else class="comment-span">{{ b.text }}</span>
    </template>
  </div>
  <div v-else class="comment-body comment-body-plain">{{ text }}</div>
</template>

<style scoped>
.comment-body {
  font-size: 12.5px;
  color: var(--text);
  line-height: 1.5;
  white-space: pre-wrap;
}
.comment-body-plain {
  white-space: pre-wrap;
}
.comment-mention {
  color: var(--accent);
  font-weight: 500;
}
.comment-image-link {
  display: inline-block;
  margin: 2px 0;
}
.comment-image-link img {
  max-width: 100%;
  max-height: 240px;
  border-radius: 4px;
  border: 1px solid var(--border);
  display: block;
}
.comment-image-fallback {
  display: inline-block;
  padding: 4px 8px;
  margin: 2px 0;
  font-family: "Geist Mono", monospace;
  font-size: 11px;
  color: var(--text-2);
  background: var(--surface);
  border: 1px dashed var(--border);
  border-radius: 4px;
  text-decoration: none;
}
.comment-image-fallback:hover {
  color: var(--text);
  border-color: var(--border-strong);
}
</style>
