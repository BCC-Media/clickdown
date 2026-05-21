<script setup lang="ts">
import { ref, watch } from "vue";
import { api, type Comment } from "../api";

const props = defineProps<{ taskId: number }>();

const comments = ref<Comment[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const draft = ref("");
const posting = ref(false);

async function load() {
  const id = props.taskId;
  loading.value = true;
  error.value = null;
  // Stale-while-revalidate: render local DB rows first, then pull from
  // ClickUp and re-render with whatever the refresh brings back.
  try {
    const cached = await api.listTaskComments(id);
    if (id === props.taskId) comments.value = cached;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
  try {
    const fresh = await api.listTaskComments(id, { refresh: true });
    if (id === props.taskId) comments.value = fresh;
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

watch(() => props.taskId, load, { immediate: true });

async function post() {
  const text = draft.value.trim();
  if (!text || posting.value) return;
  posting.value = true;
  error.value = null;
  try {
    const c = await api.postTaskComment(props.taskId, text);
    comments.value = [...comments.value, c];
    draft.value = "";
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    posting.value = false;
  }
}

function onKey(e: KeyboardEvent) {
  // Cmd/Ctrl+Enter or Shift+Enter posts; plain Enter inserts a newline.
  if (e.key === "Enter" && (e.metaKey || e.ctrlKey || e.shiftKey)) {
    e.preventDefault();
    post();
  }
  e.stopPropagation();
}

function fmtTime(ms: number): string {
  if (!ms) return "";
  const d = new Date(ms);
  const date =
    `${String(d.getDate()).padStart(2, "0")}.` +
    `${String(d.getMonth() + 1).padStart(2, "0")}.` +
    `${d.getFullYear()}`;

  const diff = Date.now() - ms;
  const min = 60_000, hr = 60 * min, day = 24 * hr, week = 7 * day,
        month = 30 * day, year = 365 * day;
  const plural = (n: number, unit: string) => `${n} ${unit}${n === 1 ? "" : "s"} ago`;

  let rel: string;
  if (diff < min) rel = "just now";
  else if (diff < hr) rel = plural(Math.floor(diff / min), "minute");
  else if (diff < day) rel = plural(Math.floor(diff / hr), "hour");
  else if (diff < week) rel = plural(Math.floor(diff / day), "day");
  else if (diff < month) rel = plural(Math.floor(diff / week), "week");
  else if (diff < year) rel = plural(Math.floor(diff / month), "month");
  else rel = plural(Math.floor(diff / year), "year");

  return `${rel} (${date})`;
}
</script>

<template>
  <div class="comments">
    <div class="expand-label">comments</div>
    <div v-if="loading && comments.length === 0" class="comments-empty">loading…</div>
    <div v-else-if="!loading && comments.length === 0" class="comments-empty">no comments yet</div>
    <div v-else class="comment-list">
      <div v-for="c in comments" :key="c.id" :class="['comment', { pending: c.pending }]">
        <div class="comment-meta">
          <span class="comment-author">{{ c.author || "you" }}</span>
          <span class="comment-time">{{ fmtTime(c.created_at) }}</span>
          <span v-if="c.pending" class="comment-pending">syncing…</span>
        </div>
        <div class="comment-text">{{ c.text }}</div>
      </div>
    </div>
    <div class="comment-compose">
      <textarea
        v-model="draft"
        :disabled="posting"
        placeholder="Add a comment…  (⇧⏎ or ⌘/Ctrl+⏎ to post)"
        rows="2"
        @keydown="onKey"
      />
      <button
        class="comment-post"
        :disabled="posting || !draft.trim()"
        @click.stop="post"
      >{{ posting ? "posting…" : "post" }}</button>
    </div>
    <div v-if="error" class="comment-error">{{ error }}</div>
  </div>
</template>

<style scoped>
.comments {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 6px;
  border-top: 1px dashed var(--border);
  margin-top: 4px;
}
.comments-empty {
  font-size: 12px;
  color: var(--text-3);
  font-style: italic;
}
.comment-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.comment {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px;
  background: var(--surface-2);
  border-radius: 4px;
}
.comment.pending {
  opacity: 0.7;
}
.comment-meta {
  display: flex;
  gap: 10px;
  align-items: baseline;
  font-family: "Geist Mono", monospace;
  font-size: 10px;
  color: var(--text-3);
}
.comment-author {
  color: var(--text-2);
  font-weight: 500;
}
.comment-time {
  color: var(--text-3);
}
.comment-pending {
  color: var(--accent);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.comment-text {
  font-size: 12.5px;
  color: var(--text);
  line-height: 1.5;
  white-space: pre-wrap;
}
.comment-compose {
  display: flex;
  gap: 6px;
  align-items: flex-start;
  margin-top: 4px;
}
.comment-compose textarea {
  flex: 1;
  resize: vertical;
  min-height: 32px;
  padding: 6px 8px;
  font: inherit;
  font-size: 12.5px;
  color: var(--text);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 4px;
  outline: none;
}
.comment-compose textarea:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--accent);
}
.comment-post {
  font-family: "Geist Mono", monospace;
  font-size: 11px;
  padding: 6px 12px;
  background: var(--surface);
  border: 1px solid var(--border-strong);
  color: var(--text-2);
  border-radius: 4px;
  cursor: pointer;
}
.comment-post:hover:not(:disabled) {
  background: var(--surface-2);
  color: var(--text);
}
.comment-post:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.comment-error {
  font-size: 11px;
  color: var(--filter-no);
  font-family: "Geist Mono", monospace;
}
</style>
