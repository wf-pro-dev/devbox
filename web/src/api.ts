import type {
  File, HealthResponse, Peer, Directory, Version, UpdateResponse,
  SendResponse
} from './types';

// ---------------------------------------------------------------------------
// Core fetch wrapper
// ---------------------------------------------------------------------------

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error((err as { error?: string }).error ?? `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as unknown as T;
  return res.json() as Promise<T>;
}

// ---------------------------------------------------------------------------
// List / filter params
// ---------------------------------------------------------------------------

export interface ListFilesParams {
  dir?: string;
  tag?: string;
  lang?: string;
  q?: string;
}

// ---------------------------------------------------------------------------
// Files
// ---------------------------------------------------------------------------

export const api = {
  listFiles: (params?: ListFilesParams): Promise<File[]> => {
    const qs = new URLSearchParams();
    if (params?.dir)  qs.set('dir',  params.dir);
    if (params?.tag)  qs.set('tag',  params.tag);
    if (params?.lang) qs.set('lang', params.lang);
    if (params?.q)    qs.set('q',    params.q);
    const query = qs.toString() ? `?${qs}` : '';
    return request<File[]>(`/files${query}`);
  },

  /** GET /files/{id}?meta=true  → JSON metadata */
  getFileMeta: (id: string): Promise<File> =>
    request<File>(`/files/${id}?meta=true`),

  /** POST /files  multipart */
  uploadFile: (form: FormData): Promise<File> =>
    request<File>('/files', { method: 'POST', body: form }),

  /** PUT /files/{id}  replace content */
  updateFile: (id: string, form: FormData): Promise<UpdateResponse> =>
    request<UpdateResponse>(`/files/${id}`, { method: 'PUT', body: form }),

  /** PATCH /files/{id}  edit description / language / path */
  editMeta: (id: string, body: Partial<Pick<File, 'description' | 'language' | 'path'>>): Promise<File> =>
    request<File>(`/files/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),

  /** DELETE /files/{id} */
  deleteFile: (id: string): Promise<void> =>
    request<void>(`/files/${id}`, { method: 'DELETE' }),

  /** POST /files/{id}/tags */
  addTags: (id: string, tags: string[]): Promise<void> =>
    request<void>(`/files/${id}/tags`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tags }),
    }),

  /** DELETE /files/{id}/tags/{tag} */
  removeTag: (id: string, tag: string): Promise<void> =>
    request<void>(`/files/${id}/tags/${tag}`, { method: 'DELETE' }),

  /** POST /files/{id}/copy */
  copyFile: (id: string, destPath: string): Promise<File> =>
    request<File>(`/files/${id}/copy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: destPath }),
    }),

  /** POST /files/{id}/move */
  moveFile: (id: string, destPath: string): Promise<File> =>
    request<File>(`/files/${id}/move`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: destPath }),
    }),

  /** GET /files/{id}/versions */
  listVersions: (id: string): Promise<Version[]> =>
    request<Version[]>(`/files/${id}/versions`),

  getVersion: (id: string, n: number): Promise<File> =>
    request<File>(`/files/${id}/versions/${n}`),

  /** POST /files/{id}/versions/{n}/rollback */
  rollback: (id: string, n: number): Promise<File> =>
    request<File>(`/files/${id}/versions/${n}/rollback`, { method: 'POST' }),

  /** POST /files/{id}/deliver */
  sendFile: (id: string, targets: string[], broadcast = false, destDir = ''): Promise<SendResponse> =>
    request<SendResponse>(`/files/${id}/send`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targets, broadcast, dest_dir: destDir }),
    }),

  /** GET /health */
  health: (): Promise<HealthResponse> =>
    request<HealthResponse>('/health'),
};

// ---------------------------------------------------------------------------
// Directories
// ---------------------------------------------------------------------------

export const listDirectories = (): Promise<Directory[]> =>
  request<Directory[]>('/dirs');

export const getDirectory = (dir: string): Promise<Directory> =>
  request<Directory>(`/dirs/${encodeURIComponent(dir)}`);

export const deleteDirectory = (dir: string): Promise<void> =>
  request<void>(`/dirs/${encodeURIComponent(dir)}`, { method: 'DELETE' });

export const tagDirectory = (dir: string, tags: string[]): Promise<void> =>
  request<void>(`/dirs/${encodeURIComponent(dir)}/tags`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tags }),
  });

export const sendDirectory = (dir: string, targets: string[], broadcast = false, destDir = ''): Promise<SendResponse> =>
  request<SendResponse>(`/dirs/${encodeURIComponent(dir)}/send`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targets, broadcast, dest_dir: destDir }),
  });

// ---------------------------------------------------------------------------
// Peers
// ---------------------------------------------------------------------------

export const listPeers = (): Promise<Peer[]> =>
  request<Peer[]>('/peers');

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  });
}

export function formatDateShort(iso: string): string {
  return new Date(iso).toLocaleDateString('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric',
  });
}

export function langColor(lang: string): string {
  const colors: Record<string, string> = {
    bash: '#22c55e', yaml: '#3b82f6', toml: '#f59e0b', json: '#f59e0b',
    python: '#6366f1', go: '#06b6d4', typescript: '#3b82f6',
    javascript: '#eab308', sql: '#ec4899', systemd: '#8b5cf6',
    ini: '#94a3b8', markdown: '#64748b', dockerfile: '#0ea5e9', text: '#94a3b8',
  };
  return colors[lang] ?? '#94a3b8';
}