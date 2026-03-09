<script>
  import { onMount } from 'svelte';
  import { api, langColor, formatBytes, formatDate } from './api.js';
  import FileCard from './components/FileCard.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import PreviewPanel from './components/PreviewPanel.svelte';
  import UploadModal from './components/UploadModal.svelte';
  import DirectoriesTab from './components/DirectoriesTab.svelte';

  let activeMainTab = 'files'; // 'files' | 'directories'

  let files = [];
  let health = null;
  let loading = true;
  let error = '';
  let searchQuery = '';
  let activeTag = '';
  let selectedFile = null;
  let showUpload = false;
  let searchTimer;

  function onSearch() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(loadFiles, 300);
  }

  async function loadFiles() {
    loading = true; error = '';
    try {
      files = await api.listFiles({ q: searchQuery || undefined, tag: activeTag || undefined });
    } catch (e) { error = e.message; }
    finally { loading = false; }
  }

  async function loadHealth() {
    try { health = await api.health(); } catch {}
  }

  function selectTag(tag) {
    activeTag = activeTag === tag ? '' : tag;
    searchQuery = '';
    loadFiles();
  }

  function clearFilters() { activeTag = ''; searchQuery = ''; loadFiles(); }

  function onFileDeleted(id) {
    files = files.filter(f => f.id !== id);
    if (selectedFile?.id === id) selectedFile = null;
  }

  function onFileUploaded(file) {
    files = [file, ...files];
    showUpload = false;
    selectedFile = file;
  }

  onMount(() => { loadFiles(); loadHealth(); });

  $: allTags = [...new Set(files.flatMap(f => f.tags))].sort();
  $: recentFiles = [...files].sort((a,b) => b.created_at.localeCompare(a.created_at)).slice(0,5);
</script>

<div class="app">
  <header>
    <div class="logo">
      <span class="logo-mark">db</span>
      <span class="logo-name">devbox</span>
    </div>
    <div class="search-wrap">
      <svg class="si" viewBox="0 0 16 16" fill="none" width="14" height="14">
        <circle cx="6.5" cy="6.5" r="4.5" stroke="currentColor" stroke-width="1.5"/>
        <path d="M10.5 10.5L14 14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
      </svg>
      <input class="search" type="text" placeholder="Search files, content, scripts…"
        bind:value={searchQuery} on:input={onSearch} />
      {#if searchQuery}
        <button class="sc" on:click={() => { searchQuery=''; loadFiles(); }}>×</button>
      {/if}
    </div>
    <button class="btn-upload" on:click={() => showUpload = true}>
      <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
        <path d="M8 2v9M4 5l4-3 4 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M2 12h12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
      </svg>
      Upload
    </button>
  </header>

  <div class="body">
    <Sidebar
      {health} {recentFiles} {activeTag} {allTags}
      on:selectTag={(e) => selectTag(e.detail)}
      on:selectFile={(e) => selectedFile = e.detail}
    />

    <main>
      <div class="tab-bar">
        <button class="tab-btn" class:active={activeMainTab === 'files'} on:click={() => activeMainTab = 'files'}>
          Files
        </button>
        <button class="tab-btn" class:active={activeMainTab === 'directories'} on:click={() => activeMainTab = 'directories'}>
          Directories
        </button>
      </div>

      {#if activeMainTab === 'directories'}
        <DirectoriesTab onFileSelect={(f) => selectedFile = f} />
      {:else}

      {#if activeTag || searchQuery}
        <div class="filter-bar">
          {#if searchQuery}
            <span class="chip">"{searchQuery}" <button on:click={() => { searchQuery=''; loadFiles(); }}>×</button></span>
          {/if}
          {#if activeTag}
            <span class="chip blue">#{activeTag} <button on:click={() => { activeTag=''; loadFiles(); }}>×</button></span>
          {/if}
          <button class="clear-all" on:click={clearFilters}>Clear all</button>
        </div>
      {/if}

      <div class="count-row">
        <span class="count">{loading ? 'Loading…' : `${files.length} file${files.length !== 1 ? 's' : ''}`}</span>
      </div>

      {#if error}
        <div class="err">{error}</div>
      {/if}

      {#if !loading && files.length === 0 && !error}
        <div class="empty">
          <p class="empty-title">No files yet</p>
          <p class="empty-sub">Upload a script, config, or snippet to get started.</p>
          <button class="btn-upload" on:click={() => showUpload = true}>Upload your first file</button>
        </div>
      {:else}
        <div class="grid">
          {#each files as file (file.id)}
            <FileCard
              {file}
              selected={selectedFile?.id === file.id}
              on:click={() => selectedFile = selectedFile?.id === file.id ? null : file}
              on:tagClick={(e) => selectTag(e.detail)}
            />
          {/each}
        </div>
      {/if}

      {/if} <!-- end files tab -->
    </main>

    {#if selectedFile}
      <PreviewPanel
        file={selectedFile}
        on:close={() => selectedFile = null}
        on:deleted={(e) => onFileDeleted(e.detail)}
        on:tagsUpdated={(e) => {
          files = files.map(f => f.id === e.detail.id ? e.detail : f);
          selectedFile = e.detail;
        }}
      />
    {/if}
  </div>
</div>

{#if showUpload}
  <UploadModal
    on:close={() => showUpload = false}
    on:uploaded={(e) => onFileUploaded(e.detail)}
  />
{/if}

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }

  :global(:root) {
    --bg: #fafaf9; --bg-2: #f5f4f0; --bg-3: #eeede8;
    --border: #e2e0d8; --border-2: #d0cec4;
    --text: #1c1b18; --text-2: #6b6860; --text-3: #9b9890;
    --radius: 6px; --radius-lg: 10px;
    --mono: 'DM Mono', 'Fira Mono', monospace;
    --serif: 'Instrument Serif', Georgia, serif;
    --sans: 'DM Sans', system-ui, sans-serif;
  }

  :global(html, body) {
    height: 100%; background: var(--bg); color: var(--text);
    font-family: var(--sans); font-size: 14px; line-height: 1.5;
    -webkit-font-smoothing: antialiased;
  }

  :global(a) { color: inherit; text-decoration: none; }
  :global(button) { cursor: pointer; font-family: inherit; }
  :global(input, select, textarea) { font-family: inherit; color: var(--text); }

  :global(::-webkit-scrollbar) { width: 6px; height: 6px; }
  :global(::-webkit-scrollbar-track) { background: transparent; }
  :global(::-webkit-scrollbar-thumb) { background: var(--border-2); border-radius: 3px; }

  @import url('https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=DM+Mono:wght@300;400;500&family=DM+Sans:wght@300;400;500;600&display=swap');

  .app { display: flex; flex-direction: column; height: 100vh; overflow: hidden; }

  header {
    height: 56px; border-bottom: 1px solid var(--border);
    display: flex; align-items: center; gap: 16px; padding: 0 20px;
    background: var(--bg); flex-shrink: 0;
  }

  .logo { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
  .logo-mark {
    width: 28px; height: 28px; background: var(--text); color: var(--bg);
    font-family: var(--mono); font-size: 11px; font-weight: 500;
    display: flex; align-items: center; justify-content: center;
    border-radius: 5px; letter-spacing: -0.5px;
  }
  .logo-name { font-family: var(--serif); font-size: 18px; letter-spacing: -0.3px; }

  .search-wrap { flex: 1; position: relative; max-width: 560px; }
  .si { position: absolute; left: 11px; top: 50%; transform: translateY(-50%); color: var(--text-3); pointer-events: none; }
  .search {
    width: 100%; height: 34px; padding: 0 32px 0 34px;
    background: var(--bg-2); border: 1px solid var(--border);
    border-radius: var(--radius); font-size: 13px; outline: none; transition: border-color 0.15s;
  }
  .search::placeholder { color: var(--text-3); }
  .search:focus { border-color: var(--border-2); background: white; }
  .sc { position: absolute; right: 8px; top: 50%; transform: translateY(-50%); background: none; border: none; color: var(--text-3); font-size: 16px; line-height: 1; padding: 2px 4px; }
  .sc:hover { color: var(--text); }

  .btn-upload {
    display: flex; align-items: center; gap: 6px; height: 34px; padding: 0 14px;
    background: var(--text); color: var(--bg); border: none; border-radius: var(--radius);
    font-size: 13px; font-weight: 500; flex-shrink: 0; transition: background 0.15s;
  }
  .btn-upload:hover { background: #3d3c38; }

  .body { display: flex; flex: 1; overflow: hidden; }

  main { flex: 1; overflow-y: auto; display: flex; flex-direction: column; }
  .tab-bar {
    display: flex; gap: 0; border-bottom: 1px solid var(--border);
    padding: 0 24px; background: white; flex-shrink: 0;
  }
  .tab-btn {
    padding: 12px 18px; background: none; border: none; border-bottom: 2px solid transparent;
    font-size: 13px; font-weight: 500; color: var(--text-3); cursor: pointer;
    margin-bottom: -1px; transition: color 0.15s, border-color 0.15s;
  }
  .tab-btn:hover { color: var(--text); }
  .tab-btn.active { color: var(--text); border-bottom-color: var(--text); }

  .filter-bar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
  .chip {
    display: flex; align-items: center; gap: 4px; padding: 3px 8px;
    background: var(--bg-3); border: 1px solid var(--border);
    border-radius: 20px; font-size: 12px; color: var(--text-2);
  }
  .chip button { background: none; border: none; color: var(--text-3); font-size: 14px; line-height: 1; padding: 0 2px; }
  .chip button:hover { color: var(--text); }
  .chip.blue { color: #2563eb; border-color: #bfdbfe; background: #eff6ff; }
  .clear-all { background: none; border: none; font-size: 12px; color: var(--text-3); text-decoration: underline; }
  .clear-all:hover { color: var(--text-2); }

  .count-row { margin-bottom: 16px; }
  .count { font-size: 12px; color: var(--text-3); font-family: var(--mono); }

  .err {
    padding: 12px 16px; background: #fef2f2; border: 1px solid #fecaca;
    border-radius: var(--radius); color: #dc2626; font-size: 13px; margin-bottom: 16px;
  }

  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }

  .empty { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 80px 20px; text-align: center; gap: 8px; }
  .empty-title { font-family: var(--serif); font-size: 20px; }
  .empty-sub { font-size: 13px; color: var(--text-3); margin-bottom: 8px; }
</style>