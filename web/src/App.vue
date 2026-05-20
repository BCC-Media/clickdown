<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick, watch } from "vue";
import Row from "./components/Row.vue";
import TweaksPanel from "./components/TweaksPanel.vue";
import { api, type Task, type Status, type SyncStatus } from "./api";

const tasks = ref<Task[]>([]);
const statuses = ref<Status[]>([]);
const syncStatus = ref<SyncStatus>({ running: false, last_sync_at: 0, last_error: "", last_duration_ms: 0, interval_seconds: 600 });

const tweaks = reactive({ theme: "dark", accent: "#f5a524" });

// Statuses whose names suggest "completed work" should be hidden by default,
// even if ClickUp marks them as type='open' (e.g. "published", "archived").
const DEFAULT_HIDDEN_RE = /^(closed|done|complete[d]?|cancel(l?ed)?|archiv(ed|e)|publish(ed)?)$/i;

const STORAGE_KEY = "clickdown:filters:v1";

function loadPersistedFilters() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as { status: typeof statusFilter.value; tag: typeof tagFilter.value };
  } catch {
    return null;
  }
}

const persisted = loadPersistedFilters();

const q = ref("");
const statusFilter = ref<Record<string, "include" | "exclude" | undefined>>(persisted?.status || {});
const tagFilter = ref<Record<string, "include" | "exclude" | undefined>>(persisted?.tag || {});
const sort = ref<"id" | "status" | "title" | "priority">("status");

let filtersInitialized = !!persisted;

watch([statusFilter, tagFilter], ([s, t]) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ status: s, tag: t }));
  } catch {}
}, { deep: true });

const focusIdx = ref(0);
const expandedId = ref<number | null>(null);
const selected = ref<Set<number>>(new Set());

const searchEl = ref<HTMLInputElement | null>(null);

const statusOrder = computed(() => {
  const m: Record<string, number> = {};
  statuses.value.forEach((s, i) => { m[s.name] = s.orderindex || i; });
  return m;
});

const counts = computed(() => {
  const c: Record<string, number> = { all: tasks.value.length };
  statuses.value.forEach((s) => { c[s.name] = 0; });
  tasks.value.forEach((t) => {
    c[t.status] = (c[t.status] || 0) + 1;
  });
  return c;
});

function tagHue(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  return h % 360;
}

const visible = computed(() => {
  const ql = q.value.trim().toLowerCase();
  const sInc = Object.keys(statusFilter.value).filter((k) => statusFilter.value[k] === "include");
  const sExc = Object.keys(statusFilter.value).filter((k) => statusFilter.value[k] === "exclude");
  const tInc = Object.keys(tagFilter.value).filter((k) => tagFilter.value[k] === "include");
  const tExc = Object.keys(tagFilter.value).filter((k) => tagFilter.value[k] === "exclude");

  let v = tasks.value.filter((x) => {
    if (sInc.length && !sInc.includes(x.status)) return false;
    if (sExc.includes(x.status)) return false;
    if (tInc.length && !tInc.every((t) => x.tags.includes(t))) return false;
    if (tExc.some((t) => x.tags.includes(t))) return false;
    if (ql && !(x.title.toLowerCase().includes(ql) || (x.desc || "").toLowerCase().includes(ql) || x.tags.some((t) => t.toLowerCase().includes(ql)))) return false;
    return true;
  });

  if (sort.value === "id") v.sort((a, b) => a.id - b.id);
  else if (sort.value === "status") v.sort((a, b) => (statusOrder.value[a.status] ?? 99) - (statusOrder.value[b.status] ?? 99) || a.id - b.id);
  else if (sort.value === "title") v.sort((a, b) => a.title.localeCompare(b.title));
  else if (sort.value === "priority") v.sort((a, b) => {
    const pa = a.priority || 0, pb = b.priority || 0;
    const ka = pa === 0 ? 99 : pa;
    const kb = pb === 0 ? 99 : pb;
    return ka - kb || a.id - b.id;
  });
  return v;
});

const titleWidth = computed(() => {
  if (typeof document === "undefined" || !visible.value.length) return 0;
  const ctx = document.createElement("canvas").getContext("2d");
  if (!ctx) return 0;
  ctx.font = '500 13px Geist, ui-sans-serif, -apple-system, sans-serif';
  let max = 0;
  for (const t of visible.value) {
    const w = ctx.measureText(t.title || "").width;
    if (w > max) max = w;
  }
  return Math.max(120, Math.min(Math.ceil(max) + 6, 640));
});

function cycle(map: Record<string, "include" | "exclude" | undefined>, key: string) {
  const cur = map[key];
  const next = cur === undefined ? "include" : cur === "include" ? "exclude" : undefined;
  const m = { ...map };
  if (next) m[key] = next;
  else delete m[key];
  return m;
}

function cycleStatus(name: string) { statusFilter.value = cycle(statusFilter.value, name); }
function cycleTag(name: string) { tagFilter.value = cycle(tagFilter.value, name); }
function setStatusExclude(name: string) {
  const m = { ...statusFilter.value };
  if (m[name] === "exclude") delete m[name]; else m[name] = "exclude";
  statusFilter.value = m;
}
function setTagExclude(name: string) {
  const m = { ...tagFilter.value };
  if (m[name] === "exclude") delete m[name]; else m[name] = "exclude";
  tagFilter.value = m;
}
function clearFilters() { statusFilter.value = {}; tagFilter.value = {}; }
function clearTag(name: string) { const m = { ...tagFilter.value }; delete m[name]; tagFilter.value = m; }

async function load() {
  const [t, s, ss, settings] = await Promise.all([
    api.listTasks(),
    api.listStatuses(),
    api.syncStatus(),
    api.getSettings(),
  ]);
  tasks.value = t;
  statuses.value = s;
  syncStatus.value = ss;
  if (settings.theme) tweaks.theme = settings.theme;
  if (settings.accent) tweaks.accent = settings.accent;
  // Default filter on first launch: hide statuses that mean "completed work".
  // Covers both ClickUp's type='closed' and common name patterns
  // (published/archived/done/etc.) since ClickUp lets users mark closed-style
  // states as type='open'.
  if (!filtersInitialized) {
    const hide = s
      .filter((x) => x.type === "closed" || DEFAULT_HIDDEN_RE.test(x.name))
      .map((x) => x.name);
    if (hide.length) {
      const m: Record<string, "exclude"> = {};
      hide.forEach((c) => { m[c] = "exclude"; });
      statusFilter.value = m;
    }
    filtersInitialized = true;
  }
}

async function refreshSync() {
  try { syncStatus.value = await api.syncStatus(); } catch {}
}

let syncPoll: number | undefined;
onMounted(() => {
  load();
  syncPoll = window.setInterval(refreshSync, 3000);
});
onUnmounted(() => {
  if (syncPoll) window.clearInterval(syncPoll);
});

async function patchTask(id: number, p: Partial<{ title: string; desc: string; status: string; tags: string[] }>) {
  const task = tasks.value.find((x) => x.id === id);
  if (!task) return;
  // Optimistic
  const prev = { ...task };
  if (p.title !== undefined) task.title = p.title;
  if (p.desc !== undefined) task.desc = p.desc;
  if (p.status !== undefined) task.status = p.status;
  if (p.tags !== undefined) task.tags = p.tags;
  try {
    let updated: Task;
    if (p.tags !== undefined) {
      updated = await api.putTaskTags(id, p.tags);
    } else {
      updated = await api.patchTask(id, { title: p.title, description: p.desc, status: p.status });
    }
    Object.assign(task, updated);
  } catch (e) {
    Object.assign(task, prev);
    console.error(e);
  }
}

async function setTweaks(patch: { theme?: string; accent?: string; sync_interval_seconds?: number }) {
  if (patch.theme) tweaks.theme = patch.theme;
  if (patch.accent) tweaks.accent = patch.accent;
  try {
    const out = await api.patchSettings(patch);
    if (out.theme) tweaks.theme = out.theme;
    if (out.accent) tweaks.accent = out.accent;
    if (out.sync_interval_seconds) syncStatus.value.interval_seconds = parseInt(out.sync_interval_seconds, 10);
  } catch (e) {
    console.error(e);
  }
}

async function syncNow() {
  syncStatus.value = { ...syncStatus.value, running: true };
  try {
    await api.triggerSync();
  } catch (e) {
    console.error(e);
  }
  setTimeout(async () => {
    await refreshSync();
    if (!syncStatus.value.running) await load();
  }, 800);
}

watch(() => syncStatus.value.running, async (v, prev) => {
  if (prev && !v) {
    await load();
  }
});

function toggleSelect(id: number) {
  const s = new Set(selected.value);
  if (s.has(id)) s.delete(id); else s.add(id);
  selected.value = s;
}

function bulkDone() {
  // Mark with the first 'closed' type status if any exists.
  const closed = statuses.value.find((s) => s.type === "closed");
  if (!closed) return;
  [...selected.value].forEach((id) => patchTask(id, { status: closed.name }));
  selected.value = new Set();
}

function openInClickup(t: Task) {
  if (!t.clickup_id) return;
  window.open(`https://app.clickup.com/t/${t.clickup_id}`, "_blank");
}

function onKey(e: KeyboardEvent) {
  const t = e.target as HTMLElement;
  const tag = (t.tagName || "").toLowerCase();
  const editing = t.isContentEditable || tag === "input" || tag === "textarea";
  if (e.key === "/" && !editing) {
    e.preventDefault();
    searchEl.value?.focus();
    return;
  }
  if (editing) return;

  const list = visible.value;
  const cur = list[focusIdx.value];

  if (e.key === "j" || e.key === "ArrowDown") {
    e.preventDefault();
    focusIdx.value = Math.min(list.length - 1, focusIdx.value + 1);
  } else if (e.key === "k" || e.key === "ArrowUp") {
    e.preventDefault();
    focusIdx.value = Math.max(0, focusIdx.value - 1);
  } else if (e.key === "Enter") {
    e.preventDefault();
    if (cur) expandedId.value = expandedId.value === cur.id ? null : cur.id;
  } else if (e.key === "Escape") {
    expandedId.value = null;
  } else if (e.key === "x" && cur) {
    e.preventDefault();
    toggleSelect(cur.id);
  } else if (e.key === "o" && cur) {
    e.preventDefault();
    openInClickup(cur);
  } else if (e.key === "e" && cur) {
    e.preventDefault();
    expandedId.value = cur.id;
    nextTick(() => {
      const el = document.querySelector(`.row[data-idx="${focusIdx.value}"] .row-title-text`);
      el?.dispatchEvent(new MouseEvent("dblclick", { bubbles: true }));
    });
  } else if (/^[1-9]$/.test(e.key) && cur) {
    e.preventDefault();
    const idx = parseInt(e.key, 10) - 1;
    const target = statuses.value[idx];
    if (target) patchTask(cur.id, { status: target.name });
  }
}

onMounted(() => window.addEventListener("keydown", onKey));
onUnmounted(() => window.removeEventListener("keydown", onKey));

watch(() => visible.value.length, (n) => {
  if (focusIdx.value >= n) focusIdx.value = Math.max(0, n - 1);
});

function syncDotClass(): string {
  if (syncStatus.value.last_error) return "dot-err";
  if (syncStatus.value.running) return "dot-run";
  return "dot-ok";
}
</script>

<template>
  <div
    :class="['app', tweaks.theme]"
    :style="{ '--accent': tweaks.accent, '--title-w': titleWidth + 'px' }"
  >
    <header class="top">
      <div class="brand">
        <span class="brand-mark">⌗</span>
        <span class="brand-name">clickdown</span>
        <span class="brand-count">{{ visible.length }}<span class="dim">/{{ tasks.length }}</span></span>
      </div>

      <div class="search">
        <span class="search-icon">/</span>
        <input ref="searchEl" v-model="q" placeholder="Search title, description, tags…" @keydown.escape="q = ''; ($event.target as HTMLInputElement).blur()" @keydown.stop />
        <button v-if="q" class="search-clear" @click="q = ''">✕</button>
      </div>

      <div class="chips">
        <button
          :class="['chip', { on: !Object.keys(statusFilter).length && !Object.keys(tagFilter).length }]"
          @click="clearFilters"
          title="Clear all filters"
        >all<span class="chip-n">{{ counts.all }}</span></button>
        <button
          v-for="s in statuses"
          :key="s.name"
          :class="['chip', 'chip-dyn', { on: statusFilter[s.name] === 'include', exc: statusFilter[s.name] === 'exclude' }]"
          :style="{ '--st-c': s.color }"
          @click="cycleStatus(s.name)"
          @contextmenu.prevent="setStatusExclude(s.name)"
          :title="'click: include/exclude/clear · right-click: toggle exclude'"
        >
          <span class="chip-glyph">{{ statusFilter[s.name] === 'exclude' ? '⊘' : '●' }}</span>{{ s.name }}<span class="chip-n">{{ counts[s.name] || 0 }}</span>
        </button>
        <template v-for="(state, tg) in tagFilter" :key="tg">
          <button
            :class="['chip', 'chip-tag', state === 'include' ? 'on' : 'exc']"
            :style="{ '--tag-h': tagHue(tg) }"
            @click="cycleTag(tg)"
          >
            <span class="chip-pre">{{ state === 'exclude' ? '!' : '#' }}</span>{{ tg }} <span class="chip-clear" @click.stop="clearTag(tg)">✕</span>
          </button>
        </template>
      </div>

      <div class="sort">
        <label>sort</label>
        <select v-model="sort">
          <option value="id">id</option>
          <option value="status">status</option>
          <option value="title">title</option>
          <option value="priority">priority</option>
        </select>
      </div>
    </header>

    <div class="list">
      <div v-if="visible.length === 0" class="empty">
        <div v-if="!tasks.length && !syncStatus.last_sync_at">syncing from ClickUp…</div>
        <div v-else-if="!tasks.length">no tasks assigned to you</div>
        <div v-else>no tasks match</div>
        <div class="dim">try clearing filters, or press <kbd>/</kbd> to search</div>
      </div>
      <Row
        v-for="(task, i) in visible"
        :key="task.id"
        :task="task"
        :idx="i"
        :focused="i === focusIdx"
        :expanded="expandedId === task.id"
        :selected="selected.has(task.id)"
        :tag-filter="tagFilter"
        :statuses="statuses"
        @focus="focusIdx = i"
        @expand="expandedId = $event ? task.id : null"
        @patch="patchTask(task.id, $event)"
        @select="toggleSelect(task.id)"
        @tag-click="cycleTag($event)"
        @tag-exclude="setTagExclude($event)"
        @open="openInClickup(task)"
      />
    </div>

    <footer class="foot">
      <div class="foot-shortcuts">
        <kbd>j</kbd><kbd>k</kbd> nav ·
        <kbd>↵</kbd> expand ·
        <kbd>e</kbd> edit ·
        <kbd>1</kbd>–<kbd>9</kbd> status ·
        <kbd>o</kbd> open ·
        <kbd>/</kbd> search ·
        <kbd>x</kbd> select
      </div>
      <div class="foot-sel">
        <template v-if="selected.size > 0">
          <span>{{ selected.size }} selected</span>
          <button @click="bulkDone">mark done</button>
          <button @click="selected = new Set()">clear</button>
        </template>
        <span v-else class="dim foot-sync">
          <span :class="['sync-dot', syncDotClass()]"></span>
          <span :title="syncStatus.last_error || ''">{{ syncStatus.running ? 'syncing…' : (syncStatus.last_sync_at ? 'synced ' + Math.max(0, Math.floor((Date.now() - syncStatus.last_sync_at)/1000)) + 's ago' : 'never synced') }}</span>
        </span>
      </div>
    </footer>

    <TweaksPanel
      :theme="tweaks.theme"
      :accent="tweaks.accent"
      :sync-status="syncStatus"
      :interval-seconds="syncStatus.interval_seconds"
      @set="setTweaks"
      @sync-now="syncNow"
    />
  </div>
</template>
