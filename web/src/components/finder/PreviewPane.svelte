<script lang="ts">
  import { formatBytes, formatDate } from "../../api";
  import type { DirEntry, File } from "../../types";
  import { fileIcon, folderIcon } from "./icons";
  import { tagColor } from "./fileColor";

  export let entry: DirEntry | null = null;
  export let onView: () => void = () => {};
  export let onSend: () => void = () => {};
  export let onDownload: () => void = () => {};
  export let onDelete: () => void = () => {};
  export let onTagsUpdated: (f: File) => void = () => {};
</script>

<aside class="preview">
  {#if entry}
    <div class="hero">
      <div class="icon-wrap">
        {@html entry.is_dir ? folderIcon("open") : fileIcon(entry.file?.language || "text", 40)}
      </div>
      <div class="name">{entry.name}</div>
      <div class="path">{entry.prefix ?? entry.file?.path}</div>
      <div class="actions">
        {#if entry.file}
          <button on:click={onView}>View</button>
        {/if}
        <button on:click={onDownload}>Download</button>
        <button class="primary" on:click={onSend}>Send</button>
        <button class="danger" on:click={onDelete}>Delete</button>
      </div>
    </div>

    <div class="tabs">
      <button class="active">Info</button>
    </div>

    <div class="body">
      <div class="meta-grid">
        {#if entry.file}
          <div class="k">Size</div><div class="v">{formatBytes(entry.file.size)}</div>
          <div class="k">Kind</div><div class="v">{entry.file.language || "text"}</div>
          <div class="k">Version</div><div class="v">v{entry.file.version}</div>
          <div class="k">By</div><div class="v">{entry.file.uploaded_by}</div>
          <div class="k">Created</div><div class="v">{formatDate(entry.file.created_at)}</div>
          <div class="k">Path</div><div class="v">{entry.file.path}</div>
          <div class="k">Description</div><div class="v">{entry.file.description || "—"}</div>
        {:else}
          <div class="k">Kind</div><div class="v">folder</div>
          <div class="k">Items</div><div class="v">{entry.file_count ?? 0}</div>
          <div class="k">Path</div><div class="v">{entry.prefix}</div>
        {/if}
        <div class="k">Tags</div>
        <div class="v">
          <div class="tags">
            {#if entry.file?.tags && entry.file.tags.length > 0}
              {#each entry.file.tags as tag}
                <span class="tp" style="--tc:{tagColor(tag)}">#{tag}</span>
              {/each}
            {:else}
              <span class="empty-dash">—</span>
            {/if}
          </div>
        </div>
      </div>
    </div>
  {:else}
    <div class="empty">
      <div class="icon-wrap muted">
        {@html folderIcon("default")}
      </div>
      <p>Select a file or directory</p>
      <p>to see its details</p>
    </div>
  {/if}
</aside>

<style>
  .preview {
    width: 210px;
    min-width: 210px;
    border-left: 0.5px solid var(--f-border);
    background: var(--f-surface2);
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .hero {
    padding: 14px 12px 10px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    border-bottom: 0.5px solid var(--f-border);
  }
  .icon-wrap {
    display: flex;
    justify-content: center;
    min-height: 42px;
  }
  .muted {
    opacity: 0.55;
  }
  .name {
    font-size: 12.5px;
    font-weight: 500;
    text-align: center;
  }
  .path {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--f-text3);
    text-align: center;
    word-break: break-word;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 6px;
  }
  .actions button, .tabs button {
    border: 0.5px solid var(--f-border);
    background: var(--f-surface);
    border-radius: 6px;
    padding: 4px 8px;
    font-size: 10.5px;
  }
  .actions .primary {
    color: var(--f-accent);
    border-color: var(--f-accent-border);
    background: var(--f-accent-bg);
  }
  .actions .danger {
    color: var(--f-danger);
  }
  .tabs {
    display: flex;
    gap: 3px;
    padding: 8px 8px 0;
    border-bottom: 0.5px solid var(--f-border);
  }
  .tabs button {
    flex: 1;
    border-bottom-left-radius: 0;
    border-bottom-right-radius: 0;
  }
  .tabs button.active {
    background: var(--f-bg1);
    color: var(--f-text);
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .meta-grid {
    display: grid;
    grid-template-columns: 52px 1fr;
    gap: 8px;
    padding: 10px;
  }
  .k {
    color: var(--f-text3);
    font-size: 10px;
  }
  .v {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--f-text);
  }
  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .tp {
    border: 0.5px solid color-mix(in srgb, var(--tc) 25%, transparent);
    background: color-mix(in srgb, var(--tc) 12%, white);
    color: color-mix(in srgb, var(--tc) 80%, #000);
    font-family: var(--mono);
    font-size: 10px;
    border-radius: 10px;
    padding: 1px 6px;
  }
  .empty, .empty-copy {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    color: var(--f-text3);
    gap: 4px;
    text-align: center;
    padding: 16px;
  }
  .empty-dash {
    color: var(--f-text3);
  }
</style>
