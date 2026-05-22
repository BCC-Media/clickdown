<script setup lang="ts">
import { ref, watch, nextTick } from "vue";
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
const rowEl = ref<HTMLElement | null>(null);
const descEl = ref<HTMLElement | null>(null);

watch(
  () => props.focused,
  (v) => {
    if (v && rowEl.value) rowEl.value.scrollIntoView({ block: "nearest" });
  }
);

watch(
  () => props.expanded,
  async (v) => {
    if (v && !props.task.desc) {
      await nextTick();
      descEl.value?.focus();
    }
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

function onDescBlur(e: FocusEvent) {
  const t = e.target as HTMLElement;
  const v = (t.innerText || "").replace(/\n{3,}/g, "\n\n").trim();
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
      <span class="row-desc-preview">{{ !expanded && task.desc ? task.desc : '' }}</span>
      <button class="row-open" @click.stop="$emit('open')" title="Open in ClickUp (o)">open <span class="row-open-arr">↗</span></button>
    </div>
    <div v-if="expanded" class="row-expand" @click.stop>
      <div class="expand-label">description</div>
      <div
        ref="descEl"
        class="expand-body"
        contenteditable
        spellcheck="false"
        :data-placeholder="'Add a description…  (⌫ to close, ⏎ for newline)'"
        @blur="onDescBlur"
        @keydown="onDescKey"
      >{{ task.desc }}</div>
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
