<script lang="ts">
  import { onMount } from "svelte";
  import { api } from "./api";
  import FileRow from "./components/FileRow.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import PreviewModal from "./components/PreviewModal.svelte";
  import UploadModal from "./components/UploadModal.svelte";
  import DirectoriesTab from "./components/DirectoriesTab.svelte";
  import type { File, HealthResponse, MainTab } from "./types";
  import { Toaster } from "svelte-sonner";
  

  let activeMainTab: MainTab = "files";

  let files: File[] = [];
  let health: HealthResponse | null = null;
  let loading = true;
  let error = "";
  let searchQuery = "";
  let activeTag = "";
  let previewFile: File | null = null;
  let showUpload = false;
  let searchTimer: ReturnType<typeof setTimeout>;

  // Column sort
  type SortField =
    | "file_name"
    | "size"
    | "created_at"
    | "language"
    | "uploaded_by";
  let sortField: SortField = "created_at";
  let sortDir: 1 | -1 = -1;

  function setSort(field: SortField) {
    if (sortField === field) sortDir = sortDir === 1 ? -1 : 1;
    else {
      sortField = field;
      sortDir = -1;
    }
  }

  function onSearch() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(loadFiles, 300);
  }

  async function loadFiles() {
    loading = true;
    error = "";
    try {
      files = await api.listFiles({
        q: searchQuery || undefined,
        tag: activeTag || undefined,
      });
    } catch (e: unknown) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function loadHealth() {
    try {
      health = await api.health();
    } catch {}
  }

  function selectTag(tag: string) {
    activeTag = activeTag === tag ? "" : tag;
    searchQuery = "";
    loadFiles();
  }

  function clearFilters() {
    activeTag = "";
    searchQuery = "";
    loadFiles();
  }

  function onFileDeleted(id: string) {
    files = files.filter((f) => f.id !== id);
    if (previewFile?.id === id) previewFile = null;
  }

  function onFileUploaded(file: File) {
    files = [file, ...files];
    showUpload = false;
    previewFile = file;
  }

  onMount(() => {
    loadFiles();
    loadHealth();
  });

  $: allTags = [...new Set(files.flatMap((f) => f.tags ?? []))].sort();
  $: recentFiles = [...files]
    .sort((a, b) => b.created_at.localeCompare(a.created_at))
    .slice(0, 5);

  $: sortedFiles = [...files].sort((a, b) => {
    const av = a[sortField];
    const bv = b[sortField];
    if (typeof av === "number" && typeof bv === "number")
      return (av - bv) * sortDir;
    return String(av).localeCompare(String(bv)) * sortDir;
  });
</script>

<div class="app">
  <Toaster position="top-center"  />
  <!-- ── Top bar ──────────────────────────────────────────────────── -->
  <header>
    <div class="logo">
      <span class="logo-mark">db</span>
      <span class="logo-name">Devbox</span>
    </div>

    <div class="search-wrap">
      <svg class="si" viewBox="0 0 16 16" fill="none" width="14" height="14">
        <circle
          cx="6.5"
          cy="6.5"
          r="4.5"
          stroke="currentColor"
          stroke-width="1.5"
        />
        <path
          d="M10.5 10.5L14 14"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
        />
      </svg>
      <input
        class="search"
        type="text"
        placeholder="Search files, content, scripts…"
        bind:value={searchQuery}
        on:input={onSearch}
      />
      {#if searchQuery}
        <button
          class="sc"
          on:click={() => {
            searchQuery = "";
            loadFiles();
          }}
        >
          <svg viewBox="0 0 12 12" fill="none" width="10" height="10">
            <path
              d="M2 2l8 8M10 2L2 10"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
            />
          </svg>
        </button>
      {/if}
    </div>

    <button class="btn-upload" on:click={() => (showUpload = true)}>
      <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
        <path
          d="M8 2v9M4 5l4-3 4 3"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path
          d="M2 12h12"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
        />
      </svg>
      Upload
    </button>
  </header>

  <!-- ── Body ───────────────────────────────────────────────────────── -->
  <div class="body">
    <Sidebar
      {health}
      {recentFiles}
      {activeTag}
      {allTags}
      on:selectTag={(e) => selectTag(e.detail)}
      on:selectFile={(e) => {
        previewFile = e.detail;
        activeMainTab = "files";
      }}
    />

    <main>
      <!-- Tab bar -->
      <div class="tab-bar">
        <button
          class="tab-btn"
          class:active={activeMainTab === "files"}
          on:click={() => (activeMainTab = "files")}>Files</button
        >
        <button
          class="tab-btn"
          class:active={activeMainTab === "directories"}
          on:click={() => (activeMainTab = "directories")}>Directories</button
        >
      </div>

      {#if activeMainTab === "directories"}
        <DirectoriesTab
          onFileSelect={(f) => {
            previewFile = f;
            activeMainTab = "files";
          }}

          onFileDelete={(f) => {
            onFileDeleted(f.id);
          }}
        />
      {:else}
        <!-- Active filters -->
        {#if activeTag || searchQuery}
          <div class="filter-bar">
            {#if searchQuery}
              <span class="chip">
                "{searchQuery}"
                <button
                  on:click={() => {
                    searchQuery = "";
                    loadFiles();
                  }}>×</button
                >
              </span>
            {/if}
            {#if activeTag}
              <span class="chip blue">
                #{activeTag}
                <button
                  on:click={() => {
                    activeTag = "";
                    loadFiles();
                  }}>×</button
                >
              </span>
            {/if}
            <button class="clear-all" on:click={clearFilters}>Clear all</button>
          </div>
        {/if}

        <!-- Files table -->
        {#if error}
          <div class="err">{error}</div>
        {:else if !loading && files.length === 0}
          <div class="empty">
            <div class="empty-icon">
              <svg viewBox="0 0 24 24" fill="none" width="40" height="40">
                <path
                  d="M4 4h8l2 2h6v14H4V4z"
                  stroke="currentColor"
                  stroke-width="1.3"
                  stroke-linejoin="round"
                />
                <path
                  d="M9 13h6M12 10v6"
                  stroke="currentColor"
                  stroke-width="1.3"
                  stroke-linecap="round"
                />
              </svg>
            </div>
            <p class="empty-title">No files yet</p>
            <p class="empty-sub">
              Upload a script, config, or snippet to get started.
            </p>
            <button class="btn-upload" on:click={() => (showUpload = true)}
              >Upload your first file</button
            >
          </div>
        {:else}
          <div class="table-wrap">
            <table class="file-table">
              <thead>
                <tr>
                  <th class="th-sort" on:click={() => setSort("file_name")}>
                    Name
                    {#if sortField === "file_name"}<span class="sort-arrow"
                        >{sortDir === 1 ? "↑" : "↓"}</span
                      >{/if}
                  </th>
                  <th>Description</th>
                  <th class="th-sort" on:click={() => setSort("language")}>
                    Language
                    {#if sortField === "language"}<span class="sort-arrow"
                        >{sortDir === 1 ? "↑" : "↓"}</span
                      >{/if}
                  </th>
                  <th>Tags</th>
                  <th class="th-sort th-right" on:click={() => setSort("size")}>
                    Size
                    {#if sortField === "size"}<span class="sort-arrow"
                        >{sortDir === 1 ? "↑" : "↓"}</span
                      >{/if}
                  </th>
                  <th class="th-sort" on:click={() => setSort("uploaded_by")}>
                    By
                    {#if sortField === "uploaded_by"}<span class="sort-arrow"
                        >{sortDir === 1 ? "↑" : "↓"}</span
                      >{/if}
                  </th>
                  <th class="th-sort" on:click={() => setSort("created_at")}>
                    Date
                    {#if sortField === "created_at"}<span class="sort-arrow"
                        >{sortDir === 1 ? "↑" : "↓"}</span
                      >{/if}
                  </th>
                  <th class="th-actions">
                    <span class="count-badge">
                      {loading ? "…" : `${files.length}`}
                    </span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {#if loading}
                  {#each { length: 5 } as _}
                    <tr class="skeleton-row">
                      <td><div class="skel skel-name"></div></td>
                      <td><div class="skel skel-desc"></div></td>
                      <td><div class="skel skel-lang"></div></td>
                      <td></td>
                      <td></td>
                      <td></td>
                      <td></td>
                      <td></td>
                    </tr>
                  {/each}
                {:else}
                  {#each sortedFiles as file (file.id)}
                    <FileRow
                      {file}
                      selected={previewFile?.id === file.id}
                      on:click={() => {
                        previewFile = previewFile?.id === file.id ? null : file;
                      }}
                      on:tagClick={(e) => selectTag(e.detail)}
                      on:preview={(e) => {
                        previewFile = e.detail;
                      }}
                      on:deleted={(e) => {
                        onFileDeleted(e.detail);
                      }}
                    />
                  {/each}
                {/if}
              </tbody>
            </table>
          </div>
        {/if}
      {/if}
    </main>
  </div>
</div>

<!-- ── Modals ───────────────────────────────────────────────────────────── -->

{#if previewFile}
  <PreviewModal
    file={previewFile}
    on:close={() => (previewFile = null)}
    on:deleted={(e) => onFileDeleted(e.detail)}
    on:tagsUpdated={(e) => {
      files = files.map((f) => (f.id === e.detail.id ? e.detail : f));
      previewFile = e.detail;
    }}
  />
{/if}

{#if showUpload}
  <UploadModal
    on:close={() => (showUpload = false)}
    on:uploaded={(e) => onFileUploaded(e.detail)}
  />
{/if}

<style>
  /* ── Global resets & tokens ──────────────────────────────────────── */
  :global(*, *::before, *::after) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(:root) {
    --bg: #fafaf9;
    --bg-2: #f5f4f0;
    --bg-3: #eeede8;
    --border: #e2e0d8;
    --border-2: #d0cec4;
    --text: #1c1b18;
    --text-2: #6b6860;
    --text-3: #9b9890;
    --radius: 6px;
    --radius-lg: 10px;
    --mono: "DM Mono", "Fira Mono", monospace;
    --serif: "Instrument Serif", Georgia, serif;
    --sans: "DM Sans", system-ui, sans-serif;
  }

  :global(html, body) {
    height: 100%;
    background: var(--bg);
    color: var(--text);
    font-family: var(--sans);
    font-size: 14px;
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
  }

  :global(a) {
    color: inherit;
    text-decoration: none;
  }
  :global(button) {
    cursor: pointer;
    font-family: inherit;
  }
  :global(input, select, textarea) {
    font-family: inherit;
    color: var(--text);
  }
  :global(::-webkit-scrollbar) {
    width: 5px;
    height: 5px;
  }
  :global(::-webkit-scrollbar-track) {
    background: transparent;
  }
  :global(::-webkit-scrollbar-thumb) {
    background: var(--border-2);
    border-radius: 3px;
  }

  @import url("https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=DM+Mono:wght@300;400;500&family=DM+Sans:wght@300;400;500;600&display=swap");

  /* ── Layout ──────────────────────────────────────────────────────── */
  .app {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }

  header {
    height: 52px;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 0 18px;
    background: var(--bg);
    flex-shrink: 0;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }
  .logo-mark {
    width: 26px;
    height: 26px;
    background: var(--text);
    color: var(--bg);
    font-family: var(--mono);
    font-size: 11px;
    font-weight: 500;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 5px;
    letter-spacing: -0.5px;
  }
  .logo-name {
    font-family: var(--serif);
    font-size: 17px;
    letter-spacing: -0.3px;
  }

  .search-wrap {
    flex: 1;
    position: relative;
    max-width: 520px;
  }
  .si {
    position: absolute;
    left: 10px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-3);
    pointer-events: none;
  }
  .search {
    width: 100%;
    height: 32px;
    padding: 0 30px 0 32px;
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    font-size: 13px;
    outline: none;
    transition: border-color 0.15s;
  }
  .search::placeholder {
    color: var(--text-3);
  }
  .search:focus {
    border-color: var(--border-2);
    background: white;
  }
  .sc {
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: var(--text-3);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 3px;
  }
  .sc:hover {
    color: var(--text);
  }

  .btn-upload {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 32px;
    padding: 0 13px;
    background: var(--text);
    color: var(--bg);
    border: none;
    border-radius: var(--radius);
    font-size: 13px;
    font-weight: 500;
    flex-shrink: 0;
    transition: background 0.15s;
  }
  .btn-upload:hover {
    background: #3d3c38;
  }

  .body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  main {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  /* ── Tabs ────────────────────────────────────────────────────────── */
  .tab-bar {
    display: flex;
    border-bottom: 1px solid var(--border);
    padding: 0 20px;
    background: white;
    flex-shrink: 0;
  }
  .tab-btn {
    padding: 11px 16px;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    font-size: 13px;
    font-weight: 500;
    color: var(--text-3);
    cursor: pointer;
    margin-bottom: -1px;
    transition:
      color 0.12s,
      border-color 0.12s;
  }
  .tab-btn:hover {
    color: var(--text-2);
  }
  .tab-btn.active {
    color: var(--text);
    border-bottom-color: var(--text);
  }

  /* ── Filter bar ──────────────────────────────────────────────────── */
  .filter-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 20px 0;
    flex-wrap: wrap;
  }
  .chip {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 3px 8px;
    background: var(--bg-3);
    border: 1px solid var(--border);
    border-radius: 20px;
    font-size: 12px;
    color: var(--text-2);
  }
  .chip button {
    background: none;
    border: none;
    color: var(--text-3);
    font-size: 14px;
    line-height: 1;
    padding: 0 2px;
  }
  .chip button:hover {
    color: var(--text);
  }
  .chip.blue {
    color: #2563eb;
    border-color: #bfdbfe;
    background: #eff6ff;
  }
  .clear-all {
    background: none;
    border: none;
    font-size: 12px;
    color: var(--text-3);
    text-decoration: underline;
  }
  .clear-all:hover {
    color: var(--text-2);
  }

  /* ── Table ───────────────────────────────────────────────────────── */
  .table-wrap {
    flex: 1;
    overflow-y: auto;
    overflow-x: auto;
  }

  .file-table {
    width: 100%;
    border-collapse: collapse;
    min-width: 680px;
  }

  .file-table thead {
    position: sticky;
    top: 0;
    z-index: 2;
    background: white;
  }

  .file-table th {
    text-align: left;
    font-size: 10.5px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-3);
    padding: 9px 12px;
    border-bottom: 1px solid var(--border);
    white-space: nowrap;
    user-select: none;
  }
  .th-sort {
    cursor: pointer;
  }
  .th-sort:hover {
    color: var(--text-2);
  }
  .th-right {
    text-align: right;
  }
  .th-actions {
    text-align: right;
    padding-right: 14px;
  }

  .sort-arrow {
    font-size: 10px;
    margin-left: 3px;
  }

  .count-badge {
    font-size: 10px;
    font-family: var(--mono);
    font-weight: normal;
    background: var(--bg-3);
    border: 1px solid var(--border);
    padding: 1px 6px;
    border-radius: 10px;
    color: var(--text-3);
  }

  /* ── Skeleton rows ───────────────────────────────────────────────── */
  .skeleton-row td {
    padding: 12px;
  }
  .skel {
    height: 12px;
    border-radius: 4px;
    background: var(--bg-3);
    animation: shimmer 1.4s ease-in-out infinite;
  }
  .skel-name {
    width: 140px;
  }
  .skel-desc {
    width: 200px;
  }
  .skel-lang {
    width: 60px;
  }

  @keyframes shimmer {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }

  /* ── Error ───────────────────────────────────────────────────────── */
  .err {
    margin: 16px 20px;
    padding: 12px 16px;
    background: #fef2f2;
    border: 1px solid #fecaca;
    border-radius: var(--radius);
    color: #dc2626;
    font-size: 13px;
  }

  /* ── Empty state ─────────────────────────────────────────────────── */
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    padding: 80px 20px;
    text-align: center;
    gap: 10px;
  }
  .empty-icon {
    color: var(--border-2);
    margin-bottom: 4px;
  }
  .empty-title {
    font-family: var(--serif);
    font-size: 20px;
    color: var(--text);
  }
  .empty-sub {
    font-size: 13px;
    color: var(--text-3);
    margin-bottom: 4px;
  }
</style>
