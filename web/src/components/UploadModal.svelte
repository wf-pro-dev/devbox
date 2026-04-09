<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { api, listDirectories } from '../api';
  import type { File, Directory } from '../types';

  const dispatch = createEventDispatcher<{
    close: void;
    uploaded: File;
  }>();

  let files: FileList | null = null;
  let description = '';
  let prefix = '';
  let tags = '';
  let language = '';
  let uploading = false;
  let error = '';
  let dragOver = false;

  // Directory combobox
  let dirs: Directory[] = [];
  let prefixInput = '';
  let showSuggestions = false;

  onMount(async () => {
    try {
      dirs = await listDirectories();
    } catch {
      dirs = [];
    }
  });

  $: prefix = prefixInput.trim();
  $: suggestions = prefixInput
    ? dirs.filter(d => d.prefix.toLowerCase().includes(prefixInput.toLowerCase()))
    : dirs;

  function selectSuggestion(p: string) {
    prefixInput = p;
    showSuggestions = false;
  }

  function onPrefixKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') showSuggestions = false;
  }


  const LANGUAGES = [
    '', 'bash', 'yaml', 'toml', 'json', 'python', 'go',
    'typescript', 'javascript', 'sql', 'systemd', 'ini',
    'markdown', 'dockerfile', 'text',
  ];

  async function upload() {
    if (!files?.[0]) return;
    uploading = true; error = '';
    try {
      const form = new FormData();
      form.append('file', files[0]);
      form.append('description', description);
      form.append('tags', tags);
      form.append('path', prefix + '/' + files[0].name);
      form.append('local_path', files[0].webkitRelativePath);
      if (language) form.append('language', language);
      dispatch('uploaded', await api.uploadFile(form));
    } catch (e: unknown) {
      error = (e as Error).message;
    } finally {
      uploading = false;
    }
  }

  function onDrop(e: DragEvent) {
    e.preventDefault(); dragOver = false;
    if (e.dataTransfer?.files.length) files = e.dataTransfer.files

  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') dispatch('close');
  }
</script>

<svelte:window on:keydown={onKey} />

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  class="backdrop"
  on:click={() => dispatch('close')}
  on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
>
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
    <div class="mhdr">
      <h2>Upload file</h2>
      <button class="close" on:click={() => dispatch('close')}>
        <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
          <path d="M3 3l10 10M13 3L3 13" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
    </div>

    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div
      class="dropzone"
      class:drag-over={dragOver}
      class:has-file={!!files?.[0]}
      on:dragover|preventDefault={() => (dragOver = true)}
      on:dragleave={() => (dragOver = false)}
      on:drop={onDrop}
      role="region"
      aria-label="File drop zone"
    >
      {#if files?.[0]}
        <div class="chosen">
          <svg viewBox="0 0 16 16" fill="none" width="22" height="22" class="file-icon-lg">
            <path d="M4 2h6l4 4v10H4V2z" stroke="#16a34a" stroke-width="1.3" stroke-linejoin="round"/>
            <path d="M10 2v4h4" stroke="#16a34a" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <div class="chosen-info">
            <span class="chosen-name">{files[0].name}</span>
            <span class="chosen-size">{(files[0].size / 1024).toFixed(1)} KB</span>
          </div>
          <button class="clear-file" on:click={() => files = null}>
            <svg viewBox="0 0 12 12" fill="none" width="11" height="11">
              <path d="M2 2l8 8M10 2L2 10" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
            </svg>
          </button>
        </div>
      {:else}
        <div class="hint">
          <svg viewBox="0 0 24 24" fill="none" width="28" height="28">
            <path d="M12 4v12M7 9l5-5 5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M4 18h16" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <span>Drop file here or <label class="browse">browse<input type="file" bind:files  /></label></span>
        </div>
      {/if}
    </div>

    <div class="fields">
      <label class="field">
        <span>Description</span>
        <input type="text" placeholder="What is this file?" bind:value={description} />
      </label>
      <label class="field">
        <span>Tags <em>(comma separated)</em></span>
        <input type="text" placeholder="bash, deploy, prod" bind:value={tags} />
      </label>
      <div class="field">
        <span>Directory <em>(optional — type freely or pick from list)</em></span>
        <div class="prefix-combobox">
          <input
            type="text"
            placeholder="e.g. devbox-web/components/"
            bind:value={prefixInput}
            on:focus={() => (showSuggestions = true)}
            on:blur={() => setTimeout(() => (showSuggestions = false), 120)}
            on:keydown={onPrefixKeydown}
          />
          {#if showSuggestions && suggestions.length > 0}
            <ul class="suggestions">
              {#if prefixInput}
                <li class="suggestion hint-item" on:mousedown={() => selectSuggestion('')}>
                  <span class="sug-prefix muted">— root (no directory) —</span>
                </li>
              {/if}
              {#each suggestions as dir}
                <li class="suggestion" on:mousedown={() => selectSuggestion(dir.prefix)}>
                  <svg viewBox="0 0 14 14" fill="none" width="11" height="11" class="sug-icon">
                    <path d="M1 4a1 1 0 011-1h2.5l1 1H12a1 1 0 011 1v5a1 1 0 01-1 1H2a1 1 0 01-1-1V4z" stroke="currentColor" stroke-width="1.2"/>
                  </svg>
                  <span class="sug-prefix">{dir.prefix}</span>
                  {#if dir.file_count != null}
                    <span class="sug-count">{dir.file_count} file{dir.file_count !== 1 ? 's' : ''}</span>
                  {/if}
                </li>
              {/each}
            </ul>
          {/if}
        </div>
      </div>
      <label class="field">
        <span>Language <em>(optional — auto-detected)</em></span>
        <select bind:value={language}>
          {#each LANGUAGES as l}
            <option value={l}>{l || '— auto detect —'}</option>
          {/each}
        </select>
      </label>
    </div>

    {#if error}
      <div class="error">{error}</div>
    {/if}

    <div class="mftr">
      <button class="btn-cancel" on:click={() => dispatch('close')}>Cancel</button>
      <button
        class="btn-submit"
        on:click={upload}
        disabled={!files?.[0] || uploading}
      >
        {uploading ? 'Uploading…' : 'Upload'}
      </button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0; background: rgba(0,0,0,0.35);
    display: flex; align-items: center; justify-content: center;
    z-index: 200; backdrop-filter: blur(3px);
  }
  .modal {
    background: white; border: 1px solid var(--border); border-radius: var(--radius-lg);
    width: 460px; max-width: 94vw;
    box-shadow: 0 12px 40px rgba(0,0,0,0.14);
    display: flex; flex-direction: column; overflow: hidden;
  }
  .mhdr {
    display: flex; align-items: center; justify-content: space-between;
    padding: 16px 20px; border-bottom: 1px solid var(--border);
  }
  .mhdr h2 { font-family: var(--serif); font-size: 18px; font-weight: normal; }
  .close {
    display: flex; align-items: center; justify-content: center;
    width: 28px; height: 28px; background: none; border: none;
    color: var(--text-3); cursor: pointer; border-radius: var(--radius);
    transition: all 0.1s;
  }
  .close:hover { background: var(--bg-2); color: var(--text); }

  .dropzone {
    margin: 16px 20px; border: 1.5px dashed var(--border-2);
    border-radius: var(--radius-lg); padding: 28px; text-align: center;
    transition: all 0.15s; background: var(--bg);
  }
  .dropzone.drag-over { border-color: #2563eb; background: #eff6ff; }
  .dropzone.has-file { border-style: solid; border-color: #16a34a; background: #f0fdf4; }

  .hint {
    display: flex; flex-direction: column; align-items: center; gap: 10px;
    color: var(--text-3); font-size: 13px;
  }
  .browse { color: #2563eb; text-decoration: underline; cursor: pointer; }
  .browse input { display: none; }

  .chosen {
    display: flex; align-items: center; gap: 12px; justify-content: center;
  }
  .file-icon-lg { flex-shrink: 0; }
  .chosen-info { display: flex; flex-direction: column; gap: 2px; text-align: left; }
  .chosen-name { font-family: var(--mono); font-size: 13px; color: var(--text); }
  .chosen-size { font-size: 11px; color: var(--text-3); }
  .clear-file {
    background: none; border: none; color: var(--text-3); cursor: pointer;
    padding: 4px; border-radius: 4px; display: flex;
  }
  .clear-file:hover { background: #fee2e2; color: #dc2626; }

  .fields { display: flex; flex-direction: column; gap: 12px; padding: 0 20px 16px; }
  .field { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--text-2); }
  .field em { color: var(--text-3); font-style: normal; }
  .prefix-combobox { position: relative; }
  .prefix-combobox input { width: 100%; box-sizing: border-box; }

  .suggestions {
    position: absolute; top: calc(100% + 4px); left: 0; right: 0;
    background: white; border: 1px solid var(--border);
    border-radius: var(--radius); box-shadow: 0 6px 20px rgba(0,0,0,0.1);
    list-style: none; margin: 0; padding: 4px 0;
    max-height: 180px; overflow-y: auto; z-index: 50;
  }
  .suggestion {
    display: flex; align-items: center; gap: 7px;
    padding: 6px 10px; cursor: pointer; font-size: 12px;
    transition: background 0.1s;
  }
  .suggestion:hover { background: var(--bg-2); }
  .sug-icon { flex-shrink: 0; color: var(--text-3); }
  .sug-prefix { font-family: var(--mono); color: var(--text); flex: 1; }
  .sug-prefix.muted { color: var(--text-3); font-style: italic; }
  .sug-count { font-size: 10.5px; color: var(--text-3); flex-shrink: 0; }

  .field input, .field select {
    height: 34px; padding: 0 10px; border: 1px solid var(--border);
    border-radius: var(--radius); font-size: 13px; background: var(--bg); outline: none;
  }
  .field input:focus, .field select:focus { border-color: var(--border-2); background: white; }

  .error {
    margin: 0 20px 12px; padding: 10px 12px;
    background: #fef2f2; border: 1px solid #fecaca;
    border-radius: var(--radius); font-size: 12px; color: #dc2626;
  }

  .mftr {
    display: flex; gap: 8px; justify-content: flex-end;
    padding: 14px 20px; border-top: 1px solid var(--border);
    background: var(--bg);
  }
  .btn-cancel {
    height: 34px; padding: 0 16px; border: 1px solid var(--border);
    border-radius: var(--radius); background: white; font-size: 13px;
    color: var(--text-2); cursor: pointer;
  }
  .btn-cancel:hover { background: var(--bg-2); }
  .btn-submit {
    height: 34px; padding: 0 18px; background: var(--text);
    color: white; border: none; border-radius: var(--radius);
    font-size: 13px; font-weight: 500; cursor: pointer; transition: background 0.15s;
  }
  .btn-submit:hover:not(:disabled) { background: #3d3c38; }
  .btn-submit:disabled { opacity: 0.4; pointer-events: none; }
</style>