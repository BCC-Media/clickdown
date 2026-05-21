<script setup lang="ts">
import { ref, watch, computed } from "vue";
import { api, type Comment } from "../api";
import CommentBody from "./CommentBody.vue";

const props = defineProps<{ taskId: number }>();

const comments = ref<Comment[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);
const draft = ref("");
const posting = ref(false);
const replyDrafts = ref<Record<string, string>>({});
const replyOpen = ref<Record<string, boolean>>({});
const postingReply = ref<Record<string, boolean>>({});

type Thread = { parent: Comment; replies: Comment[] };

const threads = computed<Thread[]>(() => {
  const byParent: Record<string, Comment[]> = {};
  const tops: Comment[] = [];
  for (const c of comments.value) {
    if (c.parent_clickup_id) {
      (byParent[c.parent_clickup_id] ??= []).push(c);
    } else {
      tops.push(c);
    }
  }
  return tops.map((parent) => ({
    parent,
    replies: parent.clickup_id ? (byParent[parent.clickup_id] || []) : [],
  }));
});

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

async function postReply(parentClickupID: string) {
  const text = (replyDrafts.value[parentClickupID] || "").trim();
  if (!text || postingReply.value[parentClickupID]) return;
  postingReply.value = { ...postingReply.value, [parentClickupID]: true };
  error.value = null;
  try {
    const c = await api.postTaskComment(props.taskId, text, parentClickupID);
    comments.value = [...comments.value, c];
    replyDrafts.value = { ...replyDrafts.value, [parentClickupID]: "" };
    replyOpen.value = { ...replyOpen.value, [parentClickupID]: false };
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    postingReply.value = { ...postingReply.value, [parentClickupID]: false };
  }
}

function toggleReply(parentClickupID: string) {
  replyOpen.value = { ...replyOpen.value, [parentClickupID]: !replyOpen.value[parentClickupID] };
}

function onKey(e: KeyboardEvent, send: () => void) {
  // Cmd/Ctrl+Enter or Shift+Enter posts; plain Enter inserts a newline.
  if (e.key === "Enter" && (e.metaKey || e.ctrlKey || e.shiftKey)) {
    e.preventDefault();
    send();
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
      <div v-for="t in threads" :key="t.parent.id" class="thread">
        <div :class="['comment', { pending: t.parent.pending }]">
          <div class="comment-meta">
            <span class="comment-author">{{ t.parent.author || "you" }}</span>
            <span class="comment-time">{{ fmtTime(t.parent.created_at) }}</span>
            <span v-if="t.parent.pending" class="comment-pending">syncing…</span>
          </div>
          <CommentBody class="comment-text" :blocks="t.parent.blocks" :text="t.parent.text" />
          <button
            v-if="t.parent.clickup_id"
            class="comment-reply-btn"
            @click.stop="toggleReply(t.parent.clickup_id!)"
          >{{ replyOpen[t.parent.clickup_id] ? "cancel" : "reply" }}</button>
        </div>
        <div v-if="t.replies.length || (t.parent.clickup_id && replyOpen[t.parent.clickup_id])" class="reply-list">
          <div v-for="r in t.replies" :key="r.id" :class="['comment', 'reply', { pending: r.pending }]">
            <div class="comment-meta">
              <span class="comment-author">{{ r.author || "you" }}</span>
              <span class="comment-time">{{ fmtTime(r.created_at) }}</span>
              <span v-if="r.pending" class="comment-pending">syncing…</span>
            </div>
            <CommentBody class="comment-text" :blocks="r.blocks" :text="r.text" />
          </div>
          <div v-if="t.parent.clickup_id && replyOpen[t.parent.clickup_id]" class="comment-compose reply-compose">
            <textarea
              :value="replyDrafts[t.parent.clickup_id] || ''"
              @input="replyDrafts = { ...replyDrafts, [t.parent.clickup_id!]: ($event.target as HTMLTextAreaElement).value }"
              :disabled="postingReply[t.parent.clickup_id]"
              placeholder="Reply…  (⇧⏎ or ⌘/Ctrl+⏎ to post)"
              rows="2"
              @keydown="onKey($event, () => postReply(t.parent.clickup_id!))"
            />
            <button
              class="comment-post"
              :disabled="postingReply[t.parent.clickup_id] || !(replyDrafts[t.parent.clickup_id] || '').trim()"
              @click.stop="postReply(t.parent.clickup_id!)"
            >{{ postingReply[t.parent.clickup_id] ? "posting…" : "post" }}</button>
          </div>
        </div>
      </div>
    </div>
    <div class="comment-compose">
      <textarea
        v-model="draft"
        :disabled="posting"
        placeholder="Add a comment…  (⇧⏎ or ⌘/Ctrl+⏎ to post)"
        rows="2"
        @keydown="onKey($event, post)"
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
.thread {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.reply-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-left: 18px;
  padding-left: 8px;
  border-left: 2px solid var(--border);
}
.comment {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px;
  background: var(--surface-2);
  border-radius: 4px;
}
.comment.reply {
  background: var(--surface);
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
.comment-reply-btn {
  align-self: flex-start;
  font-family: "Geist Mono", monospace;
  font-size: 10px;
  color: var(--text-3);
  padding: 2px 6px;
  margin-top: 2px;
  border-radius: 3px;
  border: 1px solid transparent;
  background: transparent;
  cursor: pointer;
}
.comment-reply-btn:hover {
  color: var(--text);
  border-color: var(--border);
}
.comment-compose {
  display: flex;
  gap: 6px;
  align-items: flex-start;
  margin-top: 4px;
}
.reply-compose {
  margin-top: 2px;
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
