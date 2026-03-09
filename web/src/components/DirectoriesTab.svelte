<script>
    import { onMount } from 'svelte';
    import { listDirectories, getDirectory, deleteDirectory } from '../api.js';
  
    export let onFileSelect = (file) => {};
  
    let dirs = [];
    let loading = true;
    let error = '';
    let expanded = new Set();
    let dirFiles = {}; // dirID → files[]
  
    onMount(load);
  
    async function load() {
      loading = true; error = '';
      try {
        dirs = await listDirectories();
      } catch (e) {
        error = e.message;
      } finally {
        loading = false;
      }
    }
  
    async function toggle(dir) {
      const s = new Set(expanded);
      if (s.has(dir.id)) {
        s.delete(dir.id);
      } else {
        s.add(dir.id);
        if (!dirFiles[dir.id]) {
          try {
            const d = await getDirectory(dir.id);
            dirFiles[dir.id] = d.files || [];
            dirFiles = dirFiles; // trigger reactivity
          } catch (e) {
            dirFiles[dir.id] = [];
          }
        }
      }
      expanded = s;
    }
  
    async function del(dir, e) {
      e.stopPropagation();
      if (!confirm(`Delete directory "${dir.name}" and all its files?`)) return;
      try {
        await deleteDirectory(dir.id);
        dirs = dirs.filter(d => d.id !== dir.id);
      } catch (e) {
        alert(e.message);
      }
    }
  
    function formatBytes(n) {
      if (!n) return '0 B';
      const u = ['B','KB','MB','GB'];
      let i = 0;
      while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
      return `${n.toFixed(i ? 1 : 0)} ${u[i]}`;
    }
  
    function formatDate(s) {
      if (!s) return '';
      return new Date(s).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    }
  </script>
  
  <div class="dirs-tab">
    {#if loading}
      <div class="empty">Loading directories…</div>
    {:else if error}
      <div class="empty err">{error}</div>
    {:else if dirs.length === 0}
      <div class="empty">
        <div class="empty-icon">📁</div>
        <p>No directories yet</p>
        <p class="empty-sub">Push a directory with <code>devbox-cli push -r ./mydir/</code></p>
      </div>
    {:else}
      <div class="dir-list">
        {#each dirs as dir (dir.id)}
          <div class="dir-card" class:open={expanded.has(dir.id)}>
            <!-- Directory header row -->
            <button class="dir-hdr" on:click={() => toggle(dir)}>
              <span class="dir-chevron" class:rotated={expanded.has(dir.id)}>›</span>
              <span class="dir-icon">📁</span>
              <span class="dir-name">{dir.name}</span>
              <span class="dir-count">{dir.file_count} file{dir.file_count !== 1 ? 's' : ''}</span>
              <span class="dir-date">{formatDate(dir.created_at)}</span>
              <span class="dir-by">{dir.uploaded_by}</span>
              <button class="dir-del" on:click={(e) => del(dir, e)} title="Delete directory">✕</button>
            </button>
  
            <!-- Expanded files -->
            {#if expanded.has(dir.id)}
              <div class="dir-files">
                {#if !dirFiles[dir.id]}
                  <div class="file-row loading-row">Loading…</div>
                {:else if dirFiles[dir.id].length === 0}
                  <div class="file-row empty-row">No files</div>
                {:else}
                  {#each dirFiles[dir.id] as file (file.id)}
                    <button class="file-row" on:click={() => onFileSelect(file)}>
                      <span class="file-path">{file.path.replace(dir.prefix, '')}</span>
                      <span class="file-lang">{file.language}</span>
                      {#if file.tags && file.tags.length > 0}
                        <span class="file-tags">
                          {#each file.tags as tag}
                            <span class="tag">{tag}</span>
                          {/each}
                        </span>
                      {/if}
                      <span class="file-size">{formatBytes(file.size)}</span>
                    </button>
                  {/each}
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
  
  <style>
    .dirs-tab { flex: 1; overflow-y: auto; padding: 16px; }
  
    .empty {
      display: flex; flex-direction: column; align-items: center;
      justify-content: center; height: 200px; color: var(--text-3);
      font-size: 13px; text-align: center; gap: 6px;
    }
    .empty.err { color: #dc2626; }
    .empty-icon { font-size: 32px; }
    .empty-sub { font-size: 12px; }
    .empty-sub code { font-family: var(--mono); background: var(--bg-2); padding: 1px 5px; border-radius: 3px; }
  
    .dir-list { display: flex; flex-direction: column; gap: 6px; }
  
    .dir-card {
      border: 1px solid var(--border); border-radius: var(--radius-lg);
      overflow: hidden; background: white;
    }
    .dir-card.open { border-color: var(--border-2); }
  
    .dir-hdr {
      display: flex; align-items: center; gap: 8px;
      padding: 10px 14px; width: 100%; background: none; border: none;
      cursor: pointer; text-align: left; transition: background 0.1s;
    }
    .dir-hdr:hover { background: var(--bg-2); }
    .dir-card.open .dir-hdr { background: var(--bg-2); }
  
    .dir-chevron {
      font-size: 16px; color: var(--text-3); transition: transform 0.15s;
      display: inline-block; width: 14px; flex-shrink: 0;
    }
    .dir-chevron.rotated { transform: rotate(90deg); }
    .dir-icon { font-size: 15px; flex-shrink: 0; }
    .dir-name { font-family: var(--mono); font-size: 13px; font-weight: 500; color: var(--text); flex: 1; }
    .dir-count { font-size: 11px; color: var(--text-3); background: var(--bg-3); padding: 2px 7px; border-radius: 10px; flex-shrink: 0; }
    .dir-date { font-size: 11px; color: var(--text-3); flex-shrink: 0; min-width: 80px; text-align: right; }
    .dir-by { font-size: 11px; color: var(--text-3); flex-shrink: 0; min-width: 70px; text-align: right; font-family: var(--mono); }
    .dir-del {
      background: none; border: none; color: var(--text-3); cursor: pointer;
      font-size: 12px; padding: 2px 6px; border-radius: 3px; flex-shrink: 0;
      opacity: 0;
    }
    .dir-hdr:hover .dir-del { opacity: 1; }
    .dir-del:hover { background: #fee2e2; color: #dc2626; }
  
    .dir-files { border-top: 1px solid var(--border); }
  
    .file-row {
      display: flex; align-items: center; gap: 10px;
      padding: 7px 14px 7px 42px; width: 100%; background: none; border: none;
      cursor: pointer; text-align: left; border-bottom: 1px solid var(--border);
      transition: background 0.1s;
    }
    .file-row:last-child { border-bottom: none; }
    .file-row:hover { background: #f8f7f4; }
    .loading-row, .empty-row { color: var(--text-3); font-size: 12px; cursor: default; }
    .loading-row:hover, .empty-row:hover { background: none; }
  
    .file-path { font-family: var(--mono); font-size: 12px; color: var(--text); flex: 1; }
    .file-lang { font-size: 11px; color: var(--text-3); background: var(--bg-3); padding: 1px 6px; border-radius: 3px; flex-shrink: 0; }
    .file-tags { display: flex; gap: 4px; flex-shrink: 0; }
    .tag { font-size: 10px; padding: 1px 6px; background: #e0f2fe; color: #0369a1; border-radius: 3px; }
    .file-size { font-family: var(--mono); font-size: 11px; color: var(--text-3); flex-shrink: 0; min-width: 50px; text-align: right; }
  </style>