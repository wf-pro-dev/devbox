async function request(path, init) {
  const res = await fetch(path, init);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error ?? `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined;
  return res.json();
}

export const api = {
  listFiles: (params) => {
    const qs = new URLSearchParams();
    if (params?.tag) qs.set('tag', params.tag);
    if (params?.q)   qs.set('q', params.q);
    const query = qs.toString() ? `?${qs}` : '';
    return request(`/files${query}`);
  },
  getFileMeta: (id) => request(`/files/${id}?meta=true`),
  uploadFile:  (form) => request('/files', { method: 'POST', body: form }),
  deleteFile:  (id) => request(`/files/${id}`, { method: 'DELETE' }),
  addTags: (id, tags) => request(`/files/${id}/tags`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tags })
  }),
  removeTag: (id, tag) => request(`/files/${id}/tags/${tag}`, { method: 'DELETE' }),
  health: () => request('/health'),
};

export function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export function formatDate(iso) {
  return new Date(iso).toLocaleDateString('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit'
  });
}

export function langColor(lang) {
  const colors = {
    bash: '#22c55e', yaml: '#3b82f6', toml: '#f59e0b', json: '#f59e0b',
    python: '#6366f1', go: '#06b6d4', typescript: '#3b82f6',
    javascript: '#eab308', sql: '#ec4899', systemd: '#8b5cf6',
    ini: '#94a3b8', markdown: '#64748b', dockerfile: '#0ea5e9', text: '#94a3b8',
  };
  return colors[lang] ?? '#94a3b8';
}

// Delivery
export const deliverFile = (id, targets, broadcast = false, destDir = '') =>
  request(`/files/${id}/deliver`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targets, broadcast, dest_dir: destDir })
  });

export const listPeers = () => request('/peers');

// Directories
export const listDirectories = () => request('/directories');
export const getDirectory = (id) => request(`/directories/${id}`);
export const deleteDirectory = (id) => request(`/directories/${id}`, { method: 'DELETE' });
export const tagDirectory = (id, tags) => request(`/directories/${id}/tags`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ tags })
});
export const deliverDirectory = (id, targets, broadcast = false, destDir = '') =>
  request(`/directories/${id}/deliver`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ targets, broadcast, dest_dir: destDir })
  });