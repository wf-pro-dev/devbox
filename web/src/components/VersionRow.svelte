<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { api, formatBytes, formatDate } from '../api';
    import type { Version } from '../types';
  
    export let version: Version;
    export let currentVersion: number;
    export let fileId: string;
  
    const dispatch = createEventDispatcher<{ rollback: number }>();
  
    let expanded = false;
    let content = '';
    let contentLoading = false;
    let contentError = '';
    let copied = false;
  
    $: isCurrent = version.version === currentVersion;
  
    async function toggle() {
      expanded = !expanded;
      if (expanded && !content && !contentError) {
        contentLoading = true;
        try {
          const res = await fetch(`/files/${fileId}/versions/${version.version}`);
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          content = await res.text();
        } catch (e: unknown) {
          contentError = (e as Error).message;
        } finally {
          contentLoading = false;
        }
      }
    }
  
    function copyContent() {
      navigator.clipboard.writeText(content);
      copied = true;
      setTimeout(() => (copied = false), 1600);
    }
  </script>
  
  <tr
    class="version-row"
    class:is-current={isCurrent}
    class:is-expanded={expanded}
    on:click={toggle}
    role="button"
    tabindex="0"
    on:keydown={(e) => e.key === 'Enter' && toggle()}
  >
    <td class="td-chevron">
      <span class="chevron" class:open={expanded}>
        <svg viewBox="0 0 10 10" fill="none" width="9" height="9">
          <path d="M3 2l4 3-4 3" stroke="currentColor" stroke-width="1.5"
            stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </span>
    </td>
  
    <td class="td-ver">
      <span class="ver-num">v{version.version}</span>
      {#if isCurrent}<span class="current-pill">current</span>{/if}
    </td>
  
    <td class="td-size">{formatBytes(version.size)}</td>
    <td class="td-by">{version.uploaded_by}</td>
  
    <td class="td-msg">
      {#if version.message}
        <span class="msg-text">{version.message}</span>
      {:else}
        <span class="msg-empty">—</span>
      {/if}
    </td>
  
    <td class="td-date">{formatDate(version.created_at)}</td>
  
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <td class="td-action" on:click|stopPropagation>
      {#if !isCurrent}
        <button class="rollback-btn" on:click={() => dispatch('rollback', version.version)}>
          <svg viewBox="0 0 14 14" fill="none" width="11" height="11">
            <path d="M2 7a5 5 0 100 0" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
            <path d="M2 3.5V7h3.5" stroke="currentColor" stroke-width="1.3"
              stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          Rollback
        </button>
      {/if}
    </td>
  </tr>
  
  {#if expanded}
    <tr class="content-row">
      <td colspan="7" class="content-cell">
        <div class="content-panel">
          <div class="content-toolbar">
            <span class="toolbar-meta">
              <svg viewBox="0 0 14 14" fill="none" width="11" height="11">
                <path d="M3 2h5l3 3v7H3V2z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/>
                <path d="M8 2v3h3" stroke="currentColor" stroke-width="1.2"
                  stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              v{version.version} · {formatBytes(version.size)} · {version.uploaded_by}
            </span>
            {#if content}
              <button class="copy-btn" on:click={copyContent}>
                {#if copied}
                  <svg viewBox="0 0 12 12" fill="none" width="11" height="11">
                    <path d="M2 6l2.5 2.5L10 3" stroke="#16a34a" stroke-width="1.5"
                      stroke-linecap="round" stroke-linejoin="round"/>
                  </svg>
                  Copied
                {:else}
                  <svg viewBox="0 0 14 14" fill="none" width="11" height="11">
                    <rect x="4" y="4" width="7" height="8" rx="1.2" stroke="currentColor" stroke-width="1.2"/>
                    <path d="M2.5 9.5V2.5a1 1 0 011-1h5.5" stroke="currentColor"
                      stroke-width="1.2" stroke-linecap="round"/>
                  </svg>
                  Copy
                {/if}
              </button>
            {/if}
          </div>
  
          <div class="content-body">
            {#if contentLoading}
              <div class="loading-row">
                <span class="spinner"></span> Loading…
              </div>
            {:else if contentError}
              <div class="error-row">{contentError}</div>
            {:else}
              <pre class="code"><code>{content}</code></pre>
            {/if}
          </div>
        </div>
      </td>
    </tr>
  {/if}
  
  <style>
    .version-row {
      cursor: pointer;
      transition: background 0.1s;
      border-bottom: 1px solid var(--border);
      outline: none;
    }
    .version-row:hover { background: var(--bg-2); }
    .version-row.is-current { background: #fafaf7; }
    .version-row.is-expanded,
    .version-row.is-expanded:hover { background: var(--bg-3); }
  
    .version-row td { padding: 10px 10px; vertical-align: middle; }
  
    .td-chevron { width: 28px; padding-left: 14px !important; }
    .chevron {
      display: flex; align-items: center; justify-content: center;
      color: var(--text-3); transition: transform 0.15s;
    }
    .chevron.open { transform: rotate(90deg); }
  
    .td-ver { white-space: nowrap; }
    .ver-num {
      font-family: var(--mono); font-size: 12.5px; font-weight: 500; color: var(--text);
    }
    .current-pill {
      margin-left: 6px;
      font-size: 9.5px; font-family: var(--mono);
      background: #f0fdf4; border: 1px solid #bbf7d0;
      color: #16a34a; padding: 1px 6px; border-radius: 3px;
      vertical-align: middle;
    }
  
    .td-size, .td-by, .td-date {
      font-family: var(--mono); font-size: 11.5px; color: var(--text-3);
      white-space: nowrap;
    }
    .td-msg { max-width: 200px; }
    .msg-text { font-size: 12.5px; color: var(--text-2); }
    .msg-empty { color: var(--text-3); font-size: 12px; }
  
    .td-action { text-align: right; padding-right: 14px !important; width: 110px; }
    .rollback-btn {
      display: inline-flex; align-items: center; gap: 5px;
      padding: 3px 10px; background: none;
      border: 1px solid var(--border); border-radius: 4px;
      font-size: 11px; cursor: pointer; color: var(--text-2);
      transition: all 0.1s;
    }
    .rollback-btn:hover { background: var(--bg-3); border-color: var(--border-2); color: var(--text); }
  
    /* Content expansion */
    .content-row { border-bottom: 2px solid var(--border-2); }
    .content-cell { padding: 0 !important; }
    .content-panel { display: flex; flex-direction: column; background: var(--bg); }
  
    .content-toolbar {
      display: flex; align-items: center; justify-content: space-between;
      padding: 7px 16px;
      border-bottom: 1px solid var(--border);
      background: var(--bg-2);
    }
    .toolbar-meta {
      display: flex; align-items: center; gap: 5px;
      font-family: var(--mono); font-size: 11px; color: var(--text-3);
    }
    .copy-btn {
      display: flex; align-items: center; gap: 4px;
      background: none; border: 1px solid var(--border); border-radius: var(--radius);
      padding: 2px 9px; font-size: 11px; color: var(--text-2); cursor: pointer;
      transition: all 0.1s;
    }
    .copy-btn:hover { background: white; border-color: var(--border-2); }
  
    .content-body {
      max-height: 400px; overflow: auto;
      padding: 14px 18px;
    }
  
    .loading-row {
      display: flex; align-items: center; gap: 8px;
      font-size: 12px; color: var(--text-3); padding: 16px 0;
    }
    .spinner {
      width: 12px; height: 12px; flex-shrink: 0;
      border: 1.5px solid var(--border-2); border-top-color: var(--text-3);
      border-radius: 50%; animation: spin 0.7s linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }
  
    .error-row {
      font-size: 12px; color: #dc2626;
      background: #fef2f2; border: 1px solid #fecaca;
      border-radius: var(--radius); padding: 8px 12px;
    }
  
    .code {
      margin: 0; font-family: var(--mono);
      font-size: 11.5px; line-height: 1.65;
      white-space: pre; color: var(--text); tab-size: 2;
    }
    code { font-family: inherit; }
  </style>