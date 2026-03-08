<script>
  import { createEventDispatcher } from 'svelte';
  import { api } from '../api.js';

  const dispatch = createEventDispatcher();

  let files = null;
  let description = '';
  let tags = '';
  let language = '';
  let uploading = false;
  let error = '';
  let dragOver = false;

  const LANGUAGES = ['','bash','yaml','toml','json','python','go','typescript','javascript','sql','systemd','ini','markdown','dockerfile','text'];

  async function upload() {
    if (!files?.[0]) return;
    uploading = true; error = '';
    try {
      const form = new FormData();
      form.append('file', files[0]);
      form.append('description', description);
      form.append('tags', tags);
      if (language) form.append('language', language);
      dispatch('uploaded', await api.uploadFile(form));
    } catch (e) { error = e.message; }
    finally { uploading = false; }
  }

  function onDrop(e) {
    e.preventDefault(); dragOver = false;
    if (e.dataTransfer?.files.length) files = e.dataTransfer.files;
  }

  function onKey(e) { if (e.key === 'Escape') dispatch('close'); }
</script>

<svelte:window on:keydown={onKey} />

<div class="backdrop" on:click={() => dispatch('close')} role="button" tabindex="-1" on:keydown={onKey}>
  <div class="modal" on:click|stopPropagation role="dialog">
    <div class="mhdr">
      <h2>Upload file</h2>
      <button class="close" on:click={() => dispatch('close')}>×</button>
    </div>

    <div class="dropzone" class:drag-over={dragOver} class:has-file={!!files?.[0]}
      on:dragover|preventDefault={() => dragOver = true}
      on:dragleave={() => dragOver = false}
      on:drop={onDrop}
      role="region"
    >
      {#if files?.[0]}
        <div class="chosen">
          <span>📄</span>
          <span class="chosen-name">{files[0].name}</span>
          <span class="chosen-size">{(files[0].size/1024).toFixed(1)} KB</span>
        </div>
      {:else}
        <div class="hint">
          <svg viewBox="0 0 24 24" fill="none" width="28" height="28">
            <path d="M12 4v12M7 9l5-5 5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M4 18h16" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          <span>Drop file here or <label class="browse">browse<input type="file" bind:files /></label></span>
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
      <label class="field">
        <span>Language <em>(optional — auto-detected)</em></span>
        <select bind:value={language}>
          {#each LANGUAGES as l}
            <option value={l}>{l || '— auto detect —'}</option>
          {/each}
        </select>
      </label>
    </div>

    {#if error}<div class="error">{error}</div>{/if}

    <div class="mftr">
      <button class="btn-cancel" on:click={() => dispatch('close')}>Cancel</button>
      <button class="btn-submit" on:click={upload} disabled={!files?.[0] || uploading}>
        {uploading ? 'Uploading…' : 'Upload'}
      </button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed; inset: 0; background: rgba(0,0,0,0.3);
    display: flex; align-items: center; justify-content: center;
    z-index: 100; backdrop-filter: blur(2px);
  }
  .modal {
    background: white; border: 1px solid var(--border); border-radius: var(--radius-lg);
    width: 460px; max-width: 94vw; box-shadow: 0 8px 32px rgba(0,0,0,0.12);
    display: flex; flex-direction: column; overflow: hidden;
  }
  .mhdr { display: flex; align-items: center; justify-content: space-between; padding: 16px 20px; border-bottom: 1px solid var(--border); }
  .mhdr h2 { font-family: var(--serif); font-size: 18px; font-weight: normal; }
  .close { background: none; border: none; font-size: 20px; color: var(--text-3); cursor: pointer; padding: 2px 6px; }
  .close:hover { color: var(--text); }

  .dropzone {
    margin: 16px 20px; border: 1.5px dashed var(--border-2);
    border-radius: var(--radius-lg); padding: 28px; text-align: center; transition: all 0.15s;
    background: var(--bg);
  }
  .dropzone.drag-over { border-color: #2563eb; background: #eff6ff; }
  .dropzone.has-file { border-style: solid; border-color: #16a34a; background: #f0fdf4; }
  .hint { display: flex; flex-direction: column; align-items: center; gap: 10px; color: var(--text-3); font-size: 13px; }
  .browse { color: #2563eb; text-decoration: underline; cursor: pointer; }
  .browse input { display: none; }
  .chosen { display: flex; align-items: center; gap: 10px; justify-content: center; font-size: 13px; }
  .chosen-name { font-family: var(--mono); font-size: 13px; }
  .chosen-size { font-size: 11px; color: var(--text-3); }

  .fields { display: flex; flex-direction: column; gap: 12px; padding: 0 20px 16px; }
  .field { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--text-2); }
  .field em { color: var(--text-3); font-style: normal; }
  .field input, .field select {
    height: 34px; padding: 0 10px; border: 1px solid var(--border);
    border-radius: var(--radius); font-size: 13px; background: var(--bg); outline: none;
  }
  .field input:focus, .field select:focus { border-color: var(--border-2); background: white; }

  .error { margin: 0 20px 12px; padding: 10px 12px; background: #fef2f2; border: 1px solid #fecaca; border-radius: var(--radius); font-size: 12px; color: #dc2626; }

  .mftr { display: flex; gap: 8px; justify-content: flex-end; padding: 14px 20px; border-top: 1px solid var(--border); background: var(--bg); }
  .btn-cancel { height: 34px; padding: 0 16px; border: 1px solid var(--border); border-radius: var(--radius); background: white; font-size: 13px; color: var(--text-2); cursor: pointer; }
  .btn-cancel:hover { background: var(--bg-2); }
  .btn-submit { height: 34px; padding: 0 18px; background: var(--text); color: white; border: none; border-radius: var(--radius); font-size: 13px; font-weight: 500; cursor: pointer; }
  .btn-submit:hover:not(:disabled) { background: #3d3c38; }
  .btn-submit:disabled { opacity: 0.4; pointer-events: none; }
</style>