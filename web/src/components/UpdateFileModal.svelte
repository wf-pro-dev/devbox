<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { api } from '../api';
    import type { File, UpdateResponse } from '../types';
  
    export let file: File;
  
    const dispatch = createEventDispatcher<{
      close: void;
      updated: UpdateResponse;
    }>();
  
    type Mode = 'drop' | 'editor';
    let mode: Mode = 'drop';
  
    // Drop zone
    let droppedFiles: FileList | null = null;
    let dragOver = false;
  
    // Editor
    let editorContent = '';
    let editorLoading = true;
  
    // Shared
    let message = '';
    let uploading = false;
    let error = '';
  
    async function switchToEditor() {
      mode = 'editor';
      if (!editorContent) {
        editorLoading = true;
        try {
          const res = await fetch(`/files/${file.id}`);
          editorContent = await res.text();
        } catch {
          editorContent = '';
        } finally {
          editorLoading = false;
        }
      }
    }
  
    function onDrop(e: DragEvent) {
      e.preventDefault();
      dragOver = false;
      if (e.dataTransfer?.files.length) droppedFiles = e.dataTransfer.files;
    }
  
    async function submit() {
      uploading = true; error = '';
      try {
        const form = new FormData();
        if (mode === 'drop') {
          if (!droppedFiles?.[0]) { error = 'No file selected.'; uploading = false; return; }
          form.append('file', droppedFiles[0]);
        } else {
          const blob = new Blob([editorContent], { type: 'text/plain' });
          form.append('file', blob, file.file_name);
        }
        if (message.trim()) form.append('message', message.trim());
        const result = await api.updateFile(file.id, form);
        dispatch('updated', result);
      } catch (e: unknown) {
        error = (e as Error).message;
      } finally {
        uploading = false;
      }
    }
  
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') dispatch('close');
    }
  
    $: canSubmit = mode === 'drop'
      ? !!droppedFiles?.[0]
      : editorContent.trim().length > 0 && !editorLoading;
  </script>
  
  <svelte:window on:keydown={onKey} />
  
  <div
    class="backdrop"
    on:click={() => dispatch('close')}
    on:keydown={(e) => e.key === 'Escape' && dispatch('close')}
    role="presentation"
  >
    <div class="modal" on:click|stopPropagation role="dialog" aria-modal="true">
  
      <div class="mhdr">
        <div>
          <h2>Update file</h2>
          <p class="subtitle">Replacing <span class="fname">{file.file_name}</span> · currently v{file.version}</p>
        </div>
        <button class="close" on:click={() => dispatch('close')}>
          <svg viewBox="0 0 16 16" fill="none" width="14" height="14">
            <path d="M3 3l10 10M13 3L3 13" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </button>
      </div>
  
      <div class="mode-bar">
        <button class="mode-btn" class:active={mode === 'drop'} on:click={() => mode = 'drop'}>
          <svg viewBox="0 0 14 14" fill="none" width="12" height="12">
            <path d="M7 1v8M4 4l3-3 3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M1 11h12" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
          </svg>
          Upload file
        </button>
        <button class="mode-btn" class:active={mode === 'editor'} on:click={switchToEditor}>
          <svg viewBox="0 0 14 14" fill="none" width="12" height="12">
            <path d="M2 4h10M2 7h7M2 10h5" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
          </svg>
          Edit content
        </button>
      </div>
  
      <div class="mbody">
        {#if mode === 'drop'}
          <div
            class="dropzone"
            class:drag-over={dragOver}
            class:has-file={!!droppedFiles?.[0]}
            on:dragover|preventDefault={() => (dragOver = true)}
            on:dragleave={() => (dragOver = false)}
            on:drop={onDrop}
            role="region"
            aria-label="Drop zone"
          >
            {#if droppedFiles?.[0]}
              <div class="chosen">
                <svg viewBox="0 0 16 16" fill="none" width="24" height="24">
                  <path d="M4 2h6l4 4v10H4V2z" stroke="#16a34a" stroke-width="1.3" stroke-linejoin="round"/>
                  <path d="M10 2v4h4" stroke="#16a34a" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
                <div class="chosen-info">
                  <span class="chosen-name">{droppedFiles[0].name}</span>
                  <span class="chosen-size">{(droppedFiles[0].size / 1024).toFixed(1)} KB</span>
                </div>
                <button class="clear-btn" on:click={() => droppedFiles = null}>
                  <svg viewBox="0 0 12 12" fill="none" width="10" height="10">
                    <path d="M2 2l8 8M10 2L2 10" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/>
                  </svg>
                </button>
              </div>
            {:else}
              <div class="drop-hint">
                <svg viewBox="0 0 24 24" fill="none" width="30" height="30">
                  <path d="M12 4v12M7 9l5-5 5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                  <path d="M4 18h16" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                </svg>
                <span>Drop the new file here, or
                  <label class="browse-lbl">browse
                    <input type="file" on:change={(e) => {
                      const t = e.target;
                      if (t && t.files && t.files?.length) droppedFiles = t.files;
                    }} />
                  </label>
                </span>
              </div>
            {/if}
          </div>
  
        {:else}
          <div class="editor-wrap">
            {#if editorLoading}
              <div class="editor-loading"><span class="spinner"></span> Loading current content…</div>
            {:else}
              <textarea
                class="editor"
                bind:value={editorContent}
                spellcheck="false"
                autocomplete="off"
              ></textarea>
            {/if}
          </div>
        {/if}
  
        <div class="msg-row">
          <label class="field">
            <span class="field-label">Commit message <em>(optional)</em></span>
            <input type="text" placeholder="What changed?" bind:value={message} />
          </label>
        </div>
  
        {#if error}<div class="error-msg">{error}</div>{/if}
      </div>
  
      <div class="mftr">
        <button class="btn-cancel" on:click={() => dispatch('close')}>Cancel</button>
        <button class="btn-submit" on:click={submit} disabled={!canSubmit || uploading}>
          {#if uploading}
            <span class="spinner-sm"></span> Updating…
          {:else}
            <svg viewBox="0 0 14 14" fill="none" width="12" height="12">
              <path d="M7 1v8M4 4l3-3 3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M1 11h12" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
            </svg>
            Save as v{file.version + 1}
          {/if}
        </button>
      </div>
    </div>
  </div>
  
  <style>
    .backdrop {
      position: fixed; inset: 0; background: rgba(0,0,0,0.4);
      display: flex; align-items: center; justify-content: center;
      z-index: 200; backdrop-filter: blur(3px);
    }
    .modal {
      background: white; border: 1px solid var(--border); border-radius: var(--radius-lg);
      width: min(680px, 94vw); height: 72vh;
      display: flex; flex-direction: column; overflow: hidden;
      box-shadow: 0 20px 60px rgba(0,0,0,0.18), 0 4px 16px rgba(0,0,0,0.08);
    }
    .mhdr {
      display: flex; align-items: flex-start; justify-content: space-between;
      padding: 16px 20px; border-bottom: 1px solid var(--border); flex-shrink: 0;
    }
    .mhdr h2 { font-family: var(--serif); font-size: 18px; font-weight: normal; }
    .subtitle { font-size: 12px; color: var(--text-3); margin-top: 3px; }
    .fname { font-family: var(--mono); color: var(--text-2); }
    .close {
      display: flex; align-items: center; justify-content: center;
      width: 28px; height: 28px; background: none; border: none;
      color: var(--text-3); cursor: pointer; border-radius: var(--radius); transition: all 0.1s;
    }
    .close:hover { background: var(--bg-2); color: var(--text); }
    .mode-bar {
      display: flex; border-bottom: 1px solid var(--border);
      padding: 0 20px; flex-shrink: 0; background: var(--bg);
    }
    .mode-btn {
      display: flex; align-items: center; gap: 6px;
      padding: 9px 14px; background: none; border: none;
      border-bottom: 2px solid transparent;
      font-size: 12.5px; font-weight: 500; color: var(--text-3);
      cursor: pointer; margin-bottom: -1px;
      transition: color 0.12s, border-color 0.12s;
    }
    .mode-btn:hover { color: var(--text-2); }
    .mode-btn.active { color: var(--text); border-bottom-color: var(--text); }
    .mbody {
      flex: 1; display: flex; flex-direction: column;
      overflow: hidden; min-height: 0; padding: 16px 20px; gap: 14px;
    }
    .dropzone {
      flex: 1; border: 1.5px dashed var(--border-2); border-radius: var(--radius-lg);
      transition: all 0.15s; background: var(--bg);
      display: flex; align-items: center; justify-content: center; min-height: 0;
    }
    .dropzone.drag-over { border-color: #2563eb; background: #eff6ff; }
    .dropzone.has-file { border-style: solid; border-color: #16a34a; background: #f0fdf4; }
    .drop-hint {
      display: flex; flex-direction: column; align-items: center; gap: 10px;
      color: var(--text-3); font-size: 13px; text-align: center; padding: 20px;
    }
    .browse-lbl { color: #2563eb; text-decoration: underline; cursor: pointer; }
    .browse-lbl input { display: none; }
    .chosen { display: flex; align-items: center; gap: 12px; justify-content: center; padding: 20px; }
    .chosen-info { display: flex; flex-direction: column; gap: 2px; }
    .chosen-name { font-family: var(--mono); font-size: 13px; color: var(--text); }
    .chosen-size { font-size: 11px; color: var(--text-3); }
    .clear-btn {
      background: none; border: none; color: var(--text-3);
      cursor: pointer; padding: 4px; border-radius: 4px; display: flex;
    }
    .clear-btn:hover { background: #fee2e2; color: #dc2626; }
    .editor-wrap {
      flex: 1; display: flex; flex-direction: column;
      border: 1px solid var(--border); border-radius: var(--radius);
      overflow: hidden; background: var(--bg); min-height: 0;
    }
    .editor-loading {
      display: flex; align-items: center; gap: 8px;
      padding: 20px; font-size: 12px; color: var(--text-3);
    }
    .editor {
      flex: 1; border: none; outline: none; resize: none;
      font-family: var(--mono); font-size: 12px; line-height: 1.65;
      color: var(--text); background: var(--bg); padding: 14px 16px; tab-size: 2;
    }
    .editor:focus { background: white; }
    .msg-row { flex-shrink: 0; }
    .field { display: flex; flex-direction: column; gap: 5px; }
    .field-label { font-size: 11.5px; color: var(--text-2); }
    .field-label em { color: var(--text-3); font-style: normal; }
    .field input {
      height: 32px; padding: 0 10px;
      border: 1px solid var(--border); border-radius: var(--radius);
      font-size: 13px; background: var(--bg); outline: none; transition: border-color 0.1s;
    }
    .field input:focus { border-color: var(--border-2); background: white; }
    .error-msg {
      padding: 8px 12px; background: #fef2f2; border: 1px solid #fecaca;
      border-radius: var(--radius); font-size: 12px; color: #dc2626; flex-shrink: 0;
    }
    .mftr {
      display: flex; gap: 8px; justify-content: flex-end;
      padding: 14px 20px; border-top: 1px solid var(--border);
      background: var(--bg); flex-shrink: 0;
    }
    .btn-cancel {
      height: 34px; padding: 0 16px; border: 1px solid var(--border);
      border-radius: var(--radius); background: white; font-size: 13px;
      color: var(--text-2); cursor: pointer;
    }
    .btn-cancel:hover { background: var(--bg-2); }
    .btn-submit {
      height: 34px; padding: 0 18px;
      display: flex; align-items: center; gap: 6px;
      background: var(--text); color: white; border: none;
      border-radius: var(--radius); font-size: 13px; font-weight: 500;
      cursor: pointer; transition: background 0.15s;
    }
    .btn-submit:hover:not(:disabled) { background: #3d3c38; }
    .btn-submit:disabled { opacity: 0.4; pointer-events: none; }
    .spinner {
      width: 12px; height: 12px; flex-shrink: 0;
      border: 1.5px solid var(--border-2); border-top-color: var(--text-3);
      border-radius: 50%; animation: spin 0.7s linear infinite;
    }
    .spinner-sm {
      width: 11px; height: 11px; flex-shrink: 0;
      border: 1.5px solid rgba(255,255,255,0.35); border-top-color: white;
      border-radius: 50%; animation: spin 0.7s linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }
  </style>