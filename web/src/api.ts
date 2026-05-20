export interface Task {
  id: number;
  clickup_id: string;
  title: string;
  desc: string;
  status: string;
  priority: number | null;
  tags: string[];
  updated_at: number;
}

export interface Status {
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
};
