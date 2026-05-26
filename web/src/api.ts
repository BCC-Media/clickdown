export interface TaskTag {
  name: string;
  origin: string;
}

export interface Task {
  id: number;
  clickup_id: string | null;
  title: string;
  desc: string;
  status: string;
  priority: number | null;
  tags: TaskTag[];
  list_id: string | null;
  due_date: number | null;
  updated_at: number;
}

export interface ListEntity {
  id: string;
  name: string;
  team_id: string | null;
  updated_at: number;
}

export interface Status {
  list_id: string;
  name: string;
  color: string;
  type: string;
  orderindex: number;
}

export interface SyncStatus {
  running: boolean;
  last_sync_at: number;
  last_error: string;
  last_duration_ms: number;
  interval_seconds: number;
}

export interface Settings {
  [key: string]: string;
}

export interface CommentImage {
  id?: string;
  name?: string;
  title?: string;
  url?: string;
  thumbnail_small?: string;
  thumbnail_medium?: string;
  thumbnail_large?: string;
  width?: number;
  height?: number;
}

export interface CommentBlock {
  type?: "tag" | "image" | string;
  text?: string;
  image?: CommentImage;
  user?: { id?: number | string; username?: string };
}

export interface Comment {
  id: number;
  clickup_id: string | null;
  task_id: number;
  parent_clickup_id: string | null;
  author: string;
  text: string;
  blocks?: CommentBlock[];
  created_at: number;
  pending: boolean;
}

async function req<T>(input: RequestInfo, init?: RequestInit): Promise<T> {
  const r = await fetch(input, init);
  if (!r.ok) {
    const body = await r.text();
    throw new Error(`HTTP ${r.status}: ${body}`);
  }
  if (r.status === 204) return undefined as T;
  const ct = r.headers.get("content-type") || "";
  if (!ct.includes("json")) return undefined as T;
  return r.json() as Promise<T>;
}

export const api = {
  listTasks: () => req<Task[]>("/api/tasks"),
  createTask: (body: { list_id: string; title: string; description?: string; status?: string }) =>
    req<Task>("/api/tasks", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  listLists: () => req<ListEntity[]>("/api/lists"),
  patchTask: (id: number, body: Partial<Pick<Task, "title" | "desc" | "status">> & { description?: string }) =>
    req<Task>(`/api/tasks/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        title: body.title,
        description: body.description ?? body.desc,
        status: body.status,
      }),
    }),
  putTaskTags: (id: number, tags: string[]) =>
    req<Task>(`/api/tasks/${id}/tags`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(tags),
    }),
  listStatuses: () => req<Status[]>("/api/statuses"),
  listTags: () => req<{ name: string; origin: string }[]>("/api/tags"),
  syncStatus: () => req<SyncStatus>("/api/sync/status"),
  triggerSync: () => req<SyncStatus>("/api/sync", { method: "POST" }),
  getSettings: () => req<Settings>("/api/settings"),
  patchSettings: (body: Partial<{ sync_interval_seconds: number; theme: string; accent: string }>) =>
    req<Settings>("/api/settings", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  listTaskComments: (id: number, opts?: { refresh?: boolean }) =>
    req<Comment[]>(`/api/tasks/${id}/comments${opts?.refresh ? "?refresh=1" : ""}`),
  postTaskComment: (id: number, text: string, parentClickupID?: string | null) =>
    req<Comment>(`/api/tasks/${id}/comments`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ text, parent_clickup_id: parentClickupID ?? null }),
    }),
};
