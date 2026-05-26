<script setup lang="ts">
import { ref, computed, watch, nextTick } from "vue";
import { api, type Task, type Status, type ListEntity } from "../api";

const props = defineProps<{
  modelValue: boolean;
  statuses: Status[];
}>();

const emit = defineEmits<{
  (e: "update:modelValue", v: boolean): void;
  (e: "created", task: Task): void;
}>();

const lists = ref<ListEntity[]>([]);
const listId = ref<string>("");
const title = ref("");
const description = ref("");
const status = ref<string>("");
const error = ref<string>("");
const submitting = ref(false);
const titleEl = ref<HTMLInputElement | null>(null);

const statusesForList = computed(() => {
  if (!listId.value) return [] as Status[];
  return props.statuses.filter((s) => s.list_id === listId.value);
});

// Default status is the first open-type status by orderindex; fall back to the
// first status row. Mirrors the server-side default so the user sees what will
// actually be sent.
function pickDefaultStatus(): string {
  const opts = [...statusesForList.value].sort((a, b) => a.orderindex - b.orderindex);
  return opts.find((s) => s.type === "open")?.name || opts[0]?.name || "";
}

watch(listId, () => {
  status.value = pickDefaultStatus();
});

async function loadLists() {
  try {
    lists.value = await api.listLists();
    if (!listId.value && lists.value.length) {
      listId.value = lists.value[0].id;
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  }
}

function reset() {
  title.value = "";
  description.value = "";
  error.value = "";
  submitting.value = false;
  if (lists.value.length && !listId.value) listId.value = lists.value[0].id;
  status.value = pickDefaultStatus();
}

watch(() => props.modelValue, (open) => {
  if (open) {
    reset();
    loadLists();
    nextTick(() => titleEl.value?.focus());
  }
});

function close() {
  if (submitting.value) return;
  emit("update:modelValue", false);
}

function onBackdropClick(e: MouseEvent) {
  if (e.target === e.currentTarget) close();
}

async function submit() {
  error.value = "";
  if (!listId.value) { error.value = "Pick a list."; return; }
  if (!title.value.trim()) { error.value = "Title is required."; return; }
  submitting.value = true;
  try {
    const task = await api.createTask({
      list_id: listId.value,
      title: title.value.trim(),
      description: description.value,
      status: status.value || undefined,
    });
    emit("created", task);
    emit("update:modelValue", false);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    submitting.value = false;
  }
}

function onKey(e: KeyboardEvent) {
  if (e.key === "Escape") {
    e.preventDefault();
    close();
  } else if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
    e.preventDefault();
    submit();
  }
}
</script>

<template>
  <div
    v-if="modelValue"
    class="modal-backdrop"
    @click="onBackdropClick"
    @keydown="onKey"
  >
    <div class="modal create-modal" role="dialog" aria-modal="true">
      <div class="modal-head">
        <span class="modal-id">New task</span>
        <span class="modal-spacer"></span>
        <button class="modal-close" @click="close" title="Close (Esc)">✕</button>
      </div>
      <div class="modal-body">
        <div class="modal-meta">
          <div class="modal-meta-row">
            <span class="modal-meta-lbl">List</span>
            <select v-model="listId" :disabled="!lists.length" class="create-input">
              <option v-if="!lists.length" value="">no known lists yet — sync first</option>
              <option v-for="l in lists" :key="l.id" :value="l.id">{{ l.name }}</option>
            </select>
          </div>
          <div class="modal-meta-row">
            <span class="modal-meta-lbl">Status</span>
            <select v-model="status" :disabled="!statusesForList.length" class="create-input">
              <option v-if="!statusesForList.length" value="">(none)</option>
              <option v-for="s in statusesForList" :key="s.list_id + '/' + s.name" :value="s.name">{{ s.name }}</option>
            </select>
          </div>
          <div class="modal-meta-row">
            <span class="modal-meta-lbl">Assignee</span>
            <span class="modal-meta-aside dim">me</span>
          </div>
        </div>

        <label class="modal-desc-lbl" for="ct-title">Title</label>
        <input
          id="ct-title"
          ref="titleEl"
          v-model="title"
          class="create-input create-title"
          placeholder="What needs doing?"
          @keydown.enter.prevent="submit"
        />

        <label class="modal-desc-lbl" for="ct-desc">Description</label>
        <textarea
          id="ct-desc"
          v-model="description"
          class="create-input create-desc"
          placeholder="Add detail (optional)"
          rows="5"
        ></textarea>

        <div v-if="error" class="create-err">{{ error }}</div>

        <div class="create-actions">
          <button class="create-cancel" @click="close" :disabled="submitting">Cancel</button>
          <button class="create-submit" @click="submit" :disabled="submitting || !title.trim() || !listId">
            {{ submitting ? "Creating…" : "Create" }}
          </button>
        </div>
      </div>
      <div class="modal-foot">
        <kbd>⌘↵</kbd> create · <kbd>Esc</kbd> close
      </div>
    </div>
  </div>
</template>

<style scoped>
.create-modal { max-height: 90vh; }
.create-input {
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text);
  border-radius: 4px;
  padding: 6px 8px;
  font-size: 13px;
  font-family: inherit;
  outline: none;
  width: 100%;
}
.create-input:focus { border-color: var(--accent); }
.create-title { font-size: 16px; font-weight: 500; }
.create-desc { resize: vertical; min-height: 96px; line-height: 1.5; }
.create-err {
  color: var(--filter-no, #e44);
  background: var(--filter-no-bg, rgba(228,68,68,0.08));
  padding: 6px 10px;
  border-radius: 4px;
  font-size: 12px;
  white-space: pre-wrap;
}
.create-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
.create-cancel, .create-submit {
  height: 30px;
  padding: 0 14px;
  border-radius: 5px;
  font-size: 12px;
  font-weight: 500;
}
.create-cancel {
  background: var(--surface-2);
  border: 1px solid var(--border);
  color: var(--text-2);
}
.create-cancel:hover:not(:disabled) { color: var(--text); }
.create-submit {
  background: var(--accent);
  color: #0e0e0c;
  border: 1px solid transparent;
}
.create-submit:hover:not(:disabled) { filter: brightness(1.05); }
.create-submit:disabled, .create-cancel:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
select.create-input { padding-right: 24px; }
</style>
