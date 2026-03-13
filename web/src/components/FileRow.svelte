<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import { formatBytes, formatDateShort, langColor, api } from "../api";
  import type { File } from "../types";
  import { toast } from "svelte-sonner";

  export let file: File;
  export let selected = false;
  export let deleting = false;

  const dispatch = createEventDispatcher<{
    click: void;
    tagClick: string;
    preview: File;
    deleted: string;
  }>();

  function shortText(fileName: string, maxLength: number = 30) {
    return fileName.length < maxLength
      ? fileName
      : fileName.slice(0, maxLength) + "...";
  }

  async function deleteFile(e: MouseEvent) {
    e.stopPropagation();
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
          } finally {
            deleting = false;
          }
        },
      },
    });
  }
</script>

<tr
  class="file-row"
  class:selected
  on:click={() => dispatch("click")}
  role="button"
  tabindex="0"
  on:keydown={(e) => e.key === "Enter" && dispatch("click")}
>
  <!-- Name + path -->
  <td class="cell-name">
    <div class="name-stack">
      <span class="filename">{file.file_name}</span>
      <span class="filepath">{file.path}</span>
    </div>
  </td>

  <!-- Description -->
  <td class="cell-desc">
    {#if file.description}
      <span class="desc">{file.description}</span>
    {:else}
      <span class="empty-desc">—</span>
    {/if}
  </td>

  <!-- Language badge -->
  <td class="cell-lang">
    <span class="lang" style="--c:{langColor(file.language)}"
      >{file.language || "—"}</span
    >
  </td>

  <!-- Tags -->
  <td class="cell-tags">
    <div class="tags">
      {#if file.tags}
        {#each file.tags ?? [] as tag}
          <button
            class="tag"
            on:click|stopPropagation={() => dispatch("tagClick", tag)}
            >#{tag}</button
          >
        {/each}
      {:else}
        <span class="empty-desc">—</span>
      {/if}
    </div>
  </td>

  <!-- Size -->
  <td class="cell-size">
    <span class="meta">{formatBytes(file.size)}</span>
  </td>

  <!-- Uploaded by -->
  <td class="cell-who">
    <span class="meta who">{file.uploaded_by}</span>
  </td>

  <!-- Date -->
  <td class="cell-date">
    <span class="meta">{formatDateShort(file.created_at)}</span>
  </td>

  <!-- Actions -->
  <td class="cell-actions" on:click|stopPropagation>
    <a
      class="action-btn"
      href="/files/{file.id}"
      download={file.file_name}
      title="Download"
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
      class="action-btn danger"
      title="Delete"
      disabled={deleting}
      on:click={deleteFile}
    >
      <svg viewBox="0 0 16 16" fill="none" width="13" height="13">
        <path
          d="M3 4h10M6 4V3h4v1M5 4v8a1 1 0 001 1h4a1 1 0 001-1V4"
          stroke="currentColor"
          stroke-width="1.3"
          stroke-linecap="round"
        />
      </svg>
    </button>
  </td>
</tr>

<style>
  .file-row {
    cursor: pointer;
    transition: background 0.1s;
    border-bottom: 1px solid var(--border);
  }
  .file-row:hover {
    background: var(--bg-2);
  }
  .file-row.selected {
    background: #faf9f6;
  }
  .file-row.selected td:first-child {
    box-shadow: inset 2px 0 0 var(--text);
  }

  td {
    padding: 10px 12px;
    vertical-align: middle;
    white-space: nowrap;
  }

  /* Name */
  .cell-name {
    min-width: 180px;
    max-width: 240px;
  }
  .name-stack {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .filename {
    font-family: var(--mono);
    font-size: 12.5px;
    font-weight: 500;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 220px;
  }
  .filepath {
    font-family: var(--mono);
    font-size: 10.5px;
    color: var(--text-3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 220px;
  }

  /* Description */
  .cell-desc {
    max-width: 260px;
    white-space: normal;
  }
  .desc {
    font-size: 12px;
    color: var(--text-2);
    display: -webkit-box;
    -webkit-line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .empty-desc {
    color: var(--text-3);
    font-size: 12px;
  }

  /* Language */
  .cell-lang {
    width: 90px;
  }
  .lang {
    font-family: var(--mono);
    font-size: 10.5px;
    font-weight: 500;
    padding: 2px 7px;
    border-radius: 20px;
    background: color-mix(in srgb, var(--c) 12%, transparent);
    color: color-mix(in srgb, var(--c) 80%, #000);
    border: 1px solid color-mix(in srgb, var(--c) 22%, transparent);
  }

  /* Tags */
  .cell-tags {
    max-width: 180px;
    white-space: normal;
  }
  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 3px;
  }
  .tag {
    background: none;
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 5px;
    font-size: 10.5px;
    font-family: var(--mono);
    color: var(--text-3);
    cursor: pointer;
    transition: all 0.1s;
    white-space: nowrap;
  }
  .tag:hover {
    border-color: #3b82f6;
    color: #3b82f6;
    background: #eff6ff;
  }

  /* Meta */
  .meta {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-3);
  }
  .who {
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
    display: block;
  }
  .cell-size {
    width: 72px;
    text-align: right;
  }
  .cell-who {
    width: 100px;
  }
  .cell-date {
    width: 110px;
  }

  /* Actions */
  .cell-actions {
    width: 68px;
    text-align: right;
    padding-right: 14px;
  }
  .action-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    background: none;
    border: none;
    border-radius: 4px;
    color: var(--text-3);
    cursor: pointer;
    text-decoration: none;
    transition:
      background 0.1s,
      color 0.1s;
    opacity: 0;
  }
  .file-row:hover .action-btn {
    opacity: 1;
  }
  .action-btn:hover {
    background: var(--bg-3);
    color: var(--text);
  }

  .action-btn.danger:hover {
    background: #fef2f2;
    color: #dc2626;
  }
</style>
