<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { formatBytes, formatDate, langColor, api } from '../api.js';
  import DeliverModal from './DeliverModal.svelte';
  import { File } from '../types.ts';

  let showDeliver = false;

  export let file: File;
  const dispatch = createEventDispatcher();

  let content = '';
  let contentLoading = true;
  let newTag = '';
  let deleting = false;

  async function loadContent() {
    contentLoading = true;
    try {
      const res = await fetch(`/files/${file.id}`);
      content = await res.text();
    } catch {
      content = '(could not load content)';
    } finally {
      contentLoading = false;
    }
  }

  async function deleteFile() {
    if (!confirm(`Delete "${file.file_name}"?`)) return;
    deleting = true;
    try {
      await api.deleteFile(file.id);
      dispatch('deleted', file.id);
    } catch (e) { alert(e.message); }
    finally { deleting = false; }
  }

  async function addTag() {
    const tag = newTag.trim().toLowerCase();
    if (!tag) return;
    try {
      await api.addTags(file.id, [tag]);
      const updated = await api.getFileMeta(file.id);
      dispatch('tagsUpdated', updated);
      newTag = '';
    } catch (e) { alert(e.message); }
  }

  async function removeTag(tag) {
    try {
      await api.removeTag(file.id, tag);
      const updated = await api.getFileMeta(file.id);
      dispatch('tagsUpdated', updated);
    } catch (e) { alert(e.message); }
  }

  function download() {
    const a = document.createElement('a');
    a.href = `/files/${file.id}`;
    a.download = file.file_name;
    a.click();
  }

  function copyContent() { navigator.clipboard.writeText(content); }

  $: file.id, loadContent();
</script>

<div class="panel">
  <div class="header">
    <div class="title">
      <span class="fname">{file.file_name}</span>
      <span class="lang" style="--c:{langColor(file.language)}">{file.language}</span>
    </div>
    <button class="close" on:click={() => dispatch('close')}>
      <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
        <path d="M3 3l10 10M13 3L3 13" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
      </svg>
    </button>
  </div>

  <div class="meta-grid">
    <span class="ml">Size</span>      <span class="mv">{formatBytes(file.size)}</span>
    <span class="ml">Uploaded by</span> <span class="mv mono">{file.uploaded_by}</span>
    <span class="ml">Created</span>   <span class="mv">{formatDate(file.created_at)}</span>
    {#if file.description}
      <span class="ml">Description</span><span class="mv">{file.description}</span>
    {/if}
    <span class="ml">SHA256</span>
    <span class="mv mono trunc" title={file.sha256}>{file.sha256.slice(0,16)}…</span>
  </div>

  <div class="tags-section">
    <span class="slabel">Tags</span>
    <div class="tags-row">
      {#each file.tags as tag}
        <span class="tag-pill">
          #{tag}
          <button class="trm" on:click={() => removeTag(tag)}>×</button>
        </span>
      {/each}
      <div class="tag-add">
        <input placeholder="add tag…" bind:value={newTag}
          on:keydown={(e) => e.key === 'Enter' && addTag()} />
        <button on:click={addTag} disabled={!newTag.trim()}>+</button>
      </div>
    </div>
  </div>

  <div class="preview">
    <div class="preview-hdr">
      <span class="slabel">Preview</span>
      <button class="icon-btn" on:click={copyContent} title="Copy">
        <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
          <rect x="5" y="5" width="8" height="9" rx="1.5" stroke="currentColor" stroke-width="1.3"/>
          <path d="M3 11V3a1 1 0 011-1h6" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
    <div class="preview-body">
      {#if contentLoading}
        <span class="loading">Loading…</span>
      {:else}
        <pre><code>{content.slice(0,4000)}{content.length > 4000 ? '\n…' : ''}</code></pre>
      {/if}
    </div>
  </div>

  <div class="actions">
    <button class="btn" on:click={download}>
      <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
        <path d="M8 2v8M5 7l3 3 3-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        <path d="M2 12h12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
      </svg>
      Download
    </button>
    <button class="btn" on:click={() => showDeliver = true}>
      <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
        <path d="M2 8h10M8 4l4 4-4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      Deliver
    </button>
    <button class="btn danger" on:click={deleteFile} disabled={deleting}>
      <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
        <path d="M3 4h10M6 4V3h4v1M5 4v8a1 1 0 001 1h4a1 1 0 001-1V4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
      </svg>
      {deleting ? 'Deleting…' : 'Delete'}
    </button>
  </div>
</div>

{#if showDeliver}
  <DeliverModal {file} on:close={() => showDeliver = false} />
{/if}

<style>
  .panel {
    width: 320px; min-width: 320px; border-left: 1px solid var(--border);
    display: flex; flex-direction: column; overflow: hidden; background: white;
  }
  .header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 14px 16px; border-bottom: 1px solid var(--border); flex-shrink: 0;
  }
  .title { display: flex; align-items: center; gap: 8px; overflow: hidden; }
  .fname { font-family: var(--mono); font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .lang {
    flex-shrink: 0; font-family: var(--mono); font-size: 10px; padding: 2px 7px; border-radius: 20px;
    background: color-mix(in srgb, var(--c) 12%, transparent);
    color: color-mix(in srgb, var(--c) 80%, #000);
    border: 1px solid color-mix(in srgb, var(--c) 20%, transparent);
  }
  .close {
    background: none; border: none; color: var(--text-3); padding: 4px;
    border-radius: var(--radius); display: flex; cursor: pointer;
  }
  .close:hover { background: var(--bg-2); color: var(--text); }

  .meta-grid {
    display: grid; grid-template-columns: auto 1fr; gap: 5px 12px;
    padding: 14px 16px; border-bottom: 1px solid var(--border);
  }
  .ml { font-size: 11px; color: var(--text-3); text-transform: uppercase; letter-spacing: 0.05em; white-space: nowrap; padding-top: 1px; }
  .mv { font-size: 12px; color: var(--text); }
  .mono { font-family: var(--mono); }
  .trunc { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .tags-section { padding: 12px 16px; border-bottom: 1px solid var(--border); display: flex; flex-direction: column; gap: 8px; }
  .slabel { font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-3); }
  .tags-row { display: flex; flex-wrap: wrap; gap: 5px; align-items: center; }
  .tag-pill {
    display: flex; align-items: center; gap: 3px; padding: 2px 8px;
    background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 4px;
    font-size: 11px; font-family: var(--mono); color: #2563eb;
  }
  .trm { background: none; border: none; color: #93c5fd; font-size: 14px; line-height: 1; padding: 0 1px; cursor: pointer; }
  .trm:hover { color: #2563eb; }
  .tag-add { display: flex; border: 1px solid var(--border); border-radius: 4px; overflow: hidden; }
  .tag-add input {
    border: none; outline: none; padding: 2px 7px; font-size: 11px;
    font-family: var(--mono); width: 80px; background: var(--bg-2);
  }
  .tag-add button {
    border: none; background: var(--bg-3); border-left: 1px solid var(--border);
    padding: 2px 7px; font-size: 14px; color: var(--text-2); cursor: pointer; line-height: 1.3;
  }
  .tag-add button:hover:not(:disabled) { background: var(--text); color: white; }
  .tag-add button:disabled { opacity: 0.4; }

  .preview { flex: 1; display: flex; flex-direction: column; overflow: hidden; border-bottom: 1px solid var(--border); }
  .preview-hdr { display: flex; align-items: center; justify-content: space-between; padding: 10px 16px 6px; flex-shrink: 0; }
  .preview-body { flex: 1; overflow: auto; padding: 8px 16px 12px; background: var(--bg); }
  pre { margin: 0; font-family: var(--mono); font-size: 11.5px; line-height: 1.6; white-space: pre-wrap; word-break: break-all; }
  .loading { font-size: 12px; color: var(--text-3); font-style: italic; }
  .icon-btn { background: none; border: none; color: var(--text-3); padding: 3px; border-radius: var(--radius); display: flex; cursor: pointer; }
  .icon-btn:hover { background: var(--bg-2); color: var(--text); }

  .actions { display: flex; gap: 8px; padding: 12px 16px; flex-shrink: 0; }
  .btn {
    flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px;
    height: 32px; border: 1px solid var(--border); border-radius: var(--radius);
    background: white; font-size: 12px; font-weight: 500; cursor: pointer; transition: all 0.1s;
  }
  .btn:hover { background: var(--bg-2); border-color: var(--border-2); }
  .btn.danger:hover { background: #fef2f2; border-color: #fecaca; color: #dc2626; }
  .btn:disabled { opacity: 0.5; pointer-events: none; }
</style>