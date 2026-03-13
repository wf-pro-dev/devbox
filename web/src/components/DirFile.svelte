<script lang="ts">
  import { formatBytes, langColor, api } from "../api";
  import type { File } from "../types";
  import { createEventDispatcher } from "svelte";
  import { toast } from "svelte-sonner";

  export let file: File;
  export let paddingLeft: number;
  export let onFileSelect: (f: File) => void;
  export let onDownload: (f: File, e: MouseEvent) => void;

  let deleting = false;

  const dispatch = createEventDispatcher<{
    deleted: string;
  }>();

  function shortText(fileName: string, maxLength: number = 30) {
    return fileName.length < maxLength
      ? fileName
      : fileName.slice(0, maxLength) + "...";
  }

  async function deleteFile() {
    toast(`Delete "${shortText(file.file_name)}"?`, {
      action: {
        label: "Confirm",
        onClick: async () => {
          deleting = true;
          try {
            await api.deleteFile(file.id);
            dispatch("deleted", file.id);
            toast.success(`Deleted "${shortText(file.file_name, 40)}"`);
          } catch (e: unknown) {
            toast.error((e as Error).message);
            console.error(e);
          } finally {
            deleting = false;
          }
        },
      },
    });
  }

</script>

<div class="file-row" style="padding-left: {paddingLeft}px">
  <!-- Clickable area: icon + name + meta -->
  <button class="file-inner" on:click={() => onFileSelect(file)}>
    <!-- File icon -->
    <svg
      viewBox="0 0 14 14"
      fill="none"
      width="12"
      height="12"
      class="file-icon"
    >
      <path
        d="M3 1h5l3 3v9H3V1z"
        stroke="currentColor"
        stroke-width="1.2"
        stroke-linejoin="round"
      />
      <path
        d="M8 1v3h3"
        stroke="currentColor"
        stroke-width="1.2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>

    <span class="file-name">{file.file_name}</span>

    {#if file.language}
      <span class="file-lang" style="--c:{langColor(file.language)}"
        >{file.language}</span
      >
    {/if}

    {#if file.tags && file.tags.length > 0}
      <div class="file-tags">
        {#each file.tags.slice(0, 3) as tag}
          <span class="ftag">#{tag}</span>
        {/each}
      </div>
    {/if}

    <span class="file-size">{formatBytes(file.size)}</span>
  </button>

  <!-- File actions (download + delete) -->
  <div class="file-actions" on:click|stopPropagation>
  <a
    class="fa-btn"
    href="/files/{file.id}"
    download={file.file_name}
    title="Download File"
    on:click|stopPropagation
  >
    <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
      <path
        d="M8 2v8M5 7l3 3 3-3"
        stroke="currentColor"
        stroke-width="1.4"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        d="M2 12h12"
        stroke="currentColor"
        stroke-width="1.4"
        stroke-linecap="round"
      />
    </svg>
  </a>
    <button
      class="fa-btn danger"
      title="Delete file"
      on:click={deleteFile} 
    >
      <svg viewBox="0 0 16 16" fill="none" width="11" height="11">
        <path
          d="M3 4h10M6 4V3h4v1M5 4v8a1 1 0 001 1h4a1 1 0 001-1V4"
          stroke="currentColor"
          stroke-width="1.3"
          stroke-linecap="round"
        />
      </svg>
    </button>
  </div>
</div>

<style>
  .file-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-width: 100%;
    background: none;
    border: none;
    padding-top: 8px;
    padding-bottom: 8px;
    padding-right: 14px;
    border-top: 1px solid var(--border);
    transition: background 0.1s;
    font-size: 12px;
    color: var(--text-2);
    box-sizing: border-box;
  }
  .file-row:hover {
    background: #f8f7f4;
  }

  /* Left-side action buttons — hidden until row hover */
  .file-actions {
    display: flex;
    gap: 2px;
    opacity: 0;
    transition: opacity 0.1s;
    flex-shrink: 0;
  }
  .file-row:hover .file-actions {
    opacity: 1;
  }

  .fa-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: none;
    border: none;
    border-radius: 4px;
    color: var(--text-3);
    cursor: pointer;
    transition:
      background 0.1s,
      color 0.1s;
  }
  .fa-btn:hover {
    background: var(--bg-3);
    color: var(--text);
  }
  .fa-btn.danger:hover {
    background: #fef2f2;
    color: #dc2626;
  }

  /* Inner clickable area stretches to fill the rest of the row */
  .file-inner {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    text-align: left;
    color: inherit;
    font-size: inherit;
  }

  .file-icon {
    flex-shrink: 0;
    color: var(--text-3);
  }

  .file-name {
    font-family: var(--mono);
    font-size: 12px;
    color: var(--text);
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }
  .file-lang {
    font-family: var(--mono);
    font-size: 10px;
    font-weight: 500;
    padding: 1px 6px;
    border-radius: 10px;
    flex-shrink: 0;
    background: color-mix(in srgb, var(--c) 12%, transparent);
    color: color-mix(in srgb, var(--c) 80%, #000);
    border: 1px solid color-mix(in srgb, var(--c) 18%, transparent);
  }
  .file-tags {
    display: flex;
    gap: 3px;
    flex-shrink: 0;
  }
  .ftag {
    font-size: 10px;
    font-family: var(--mono);
    color: var(--text-3);
    background: var(--bg-3);
    border: 1px solid var(--border);
    padding: 1px 5px;
    border-radius: 3px;
  }
  .file-size {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-3);
    flex-shrink: 0;
    min-width: 50px;
    text-align: right;
  }
</style>
