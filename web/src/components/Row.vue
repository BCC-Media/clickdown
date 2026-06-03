<script setup lang="ts">
import { computed, ref, watch, nextTick } from "vue";
import { renderMarkdown } from "../markdown";
import StatusPill from "./StatusPill.vue";
import TagList from "./TagList.vue";
import EditableTitle from "./EditableTitle.vue";
import PriorityBadge from "./PriorityBadge.vue";
import DueDate from "./DueDate.vue";
import CommentsSection from "./CommentsSection.vue";
import type { Task, Status } from "../api";

const props = defineProps<{
  task: Task;
  idx: number;
  focused: boolean;
  expanded: boolean;
  selected: boolean;
  tagFilter: Record<string, "include" | "exclude" | undefined>;
  statuses: Status[];
}>();
const emit = defineEmits<{
  (e: "focus"): void;
  (e: "expand", value: boolean): void;
  (e: "patch", patch: Partial<{ title: string; desc: string; status: string; tags: string[] }>): void;
  (e: "select"): void;
  (e: "tag-click", tag: string): void;
  (e: "tag-exclude", tag: string): void;
  (e: "open"): void;
}>();

const editingTitle = ref(false);
const editingDesc = ref(false);
const descDraft = ref("");
const rowEl = ref<HTMLElement | null>(null);
const descEl = ref<HTMLElement | null>(null);

// The description is stored as Markdown. We render it (images inline) in the
// read view and switch to editing the raw Markdown source on demand.
const renderedDesc = computed(() => renderMarkdown(props.task.desc || ""));

// One-line collapsed preview: strip Markdown syntax down to readable text so
// raw ![](url)/[link](url) markup doesn't leak into the row.
const descPreview = computed(() =>
  (props.task.desc || "")
    .replace(/!\[[^\]]*\]\([^)]*\)/g, "")
    .replace(/\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/[#>*_`~]/g, " ")
    .replace(/\s+/g, " ")
    .trim()
);

watch(
  () => props.focused,
  (v) => {
    if (v && rowEl.value) rowEl.value.scrollIntoView({ block: "nearest" });
  }
);

watch(
  () => props.expanded,
  (v) => {
    // Opening an empty description drops straight into edit mode; otherwise it
    // shows the rendered Markdown until the user clicks to edit.
    if (v && !props.task.desc) startEditDesc();
    else if (!v) editingDesc.value = false;
  }
);

function isClosed(): boolean {
  return props.statuses.find((s) => s.name === props.task.status)?.type === "closed";
}

function onRowClick() {
  emit("focus");
  emit("expand", !props.expanded);
}

function onTitleDbl(e: MouseEvent) {
  e.stopPropagation();
  editingTitle.value = true;
}

function onTitleCommit(v: string) {
  emit("patch", { title: v });
}

function startEditDesc() {
  descDraft.value = props.task.desc || "";
  editingDesc.value = true;
  nextTick(() => descEl.value?.focus());
}

// Clicking the rendered description enters edit mode, but let clicks on links
// (and images) behave normally so they remain openable.
function onReadClick(e: MouseEvent) {
  if ((e.target as HTMLElement).closest("a, img")) return;
  startEditDesc();
}

function onDescBlur(e: FocusEvent) {
  const t = e.target as HTMLElement;
  const v = (t.innerText || "").replace(/\n{3,}/g, "\n\n").trim();
  editingDesc.value = false;
  if (v !== (props.task.desc || "")) emit("patch", { desc: v });
}

function onDescKey(e: KeyboardEvent) {
  if (e.key === "Escape") {
    e.preventDefault();
    (e.target as HTMLElement).blur();
    emit("expand", false);
  }
  e.stopPropagation();
}
</script>

<template>
  <div
    ref="rowEl"
    :class="['row', { focused, expanded, selected, 'row-closed': isClosed() }]"
    :data-idx="idx"
    @click="onRowClick"
  >
    <div class="row-main">
      <button
        :class="['sel', { on: selected }]"
        @click.stop="$emit('select')"
        title="Select (x)"
      >{{ selected ? '■' : '□' }}</button>
      <span class="row-id mono">{{ String(task.id).padStart(3, '0') }}</span>
      <PriorityBadge :priority="task.priority" />
      <StatusPill :status="task.status" :statuses="statuses" @change="$emit('patch', { status: $event })" />
      <span class="row-title">
        <EditableTitle
          :value="task.title"
          :editing="editingTitle"
          @commit="onTitleCommit"
          @stop-edit="editingTitle = false"
          @dblclick="onTitleDbl"
        />
      </span>
      <TagList
        :tags="task.tags"
        :tag-filter="tagFilter"
        @change="$emit('patch', { tags: $event })"
        @tag-click="$emit('tag-click', $event)"
        @tag-exclude="$emit('tag-exclude', $event)"
      />
      <DueDate :due="task.due_date" :closed="isClosed()" />
      <span class="row-desc-preview">{{ !expanded && task.desc ? descPreview : '' }}</span>
      <button class="row-open" @click.stop="$emit('open')" title="Open in ClickUp (o)">open <span class="row-open-arr">↗</span></button>
    </div>
    <div v-if="expanded" class="row-expand" @click.stop>
      <div class="expand-label">description</div>
      <div
        v-if="editingDesc"
        ref="descEl"
        class="expand-body"
        contenteditable
        spellcheck="false"
        :data-placeholder="'Add a description…  (⌫ to close, ⏎ for newline)'"
        @blur="onDescBlur"
        @keydown="onDescKey"
      >{{ descDraft }}</div>
      <div
        v-else-if="task.desc"
        class="expand-body markdown-body"
        v-html="renderedDesc"
        @click="onReadClick"
      ></div>
      <div
        v-else
        class="expand-body expand-body-empty"
        @click="startEditDesc"
      >Add a description…</div>
      <div class="expand-meta">
        <span><kbd>e</kbd> edit title</span>
        <span><kbd>1</kbd>–<kbd>9</kbd> status</span>
        <span><kbd>o</kbd> open in ClickUp</span>
        <span><kbd>esc</kbd> close</span>
      </div>
      <CommentsSection :task-id="task.id" />
    </div>
  </div>
</template>
