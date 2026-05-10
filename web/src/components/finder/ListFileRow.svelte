<script lang="ts">
  import { draggable } from "@thisux/sveltednd";
  import { formatBytes, formatDateShort } from "../../api";
  import type { DirEntry } from "../../types";
  import { fileIcon } from "./icons";
  import { tagColor } from "./fileColor";

  export let entry: DirEntry;
  export let prefix = "/";
  export let selected = false;
  export let moving = false;
  export let onSelect: (entry: DirEntry) => void = () => {};
  export let onOpen: (entry: DirEntry) => void = () => {};
  export let onContextMenu: (e: MouseEvent, entry: DirEntry) => void = () => {};
  export let onQuickSend: (entry: DirEntry) => void = () => {};
  export let onQuickDownload: (entry: DirEntry) => void = () => {};
  export let onQuickDelete: (entry: DirEntry) => void = () => {};
</script>

<tr
  class:selected
  class:moving
  class="container"
  use:draggable={{ container: prefix, dragData: entry }}
  on:click={() => onSelect(entry)}
  on:dblclick={() => onOpen(entry)}
  on:contextmenu|preventDefault={(e) => onContextMenu(e, entry)}
>
  <td>
    <span class="ic">
      {@html fileIcon(
        selected ? "selected" : entry.file?.language || "text",
        16,
      )}
    </span>
    <span class="nm">{entry.name}</span>
  </td>
  <td>{entry.file?.language || "text"}</td>
  <td>{formatBytes(entry.file?.size ?? 0)}</td>
  <td>{`v${entry.file?.version ?? 1}`}</td>
  <td class="tag-cell">
    {#if entry.file?.tags}
      {#each entry.file.tags as tag}
        <span class="tag" style="--tc:{tagColor(tag)}">#{tag}</span>
      {/each}
    {:else}
      —
    {/if}
  </td>
  <td>{entry.file?.uploaded_by}</td>
  <td
    >{formatDateShort(
      entry.file?.updated_at || entry.file?.created_at || "",
    )}</td
  >
  <td class="actions">
    <button aria-label="Send" title="Send" on:click|stopPropagation={() => onQuickSend(entry)}><i class="ti ti-send"></i></button>
    <button aria-label="Download" title="Download" on:click|stopPropagation={() => onQuickDownload(entry)}><i class="ti ti-download"></i></button>
    <button aria-label="Delete" title="Delete" class="danger" on:click|stopPropagation={() => onQuickDelete(entry)}><i class="ti ti-trash"></i></button>
  </td>
</tr>

<style>
  .container {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--f-text);
    height: 26px;
    padding: 0px 8px;
  }
  tr:hover {
    background: var(--f-bg2);
  }
  tr.selected {
    background: var(--f-selection);
  }
  tr.selected:hover {
    background: var(--f-selection);
  }
  tr.selected td {
    color: var(--f-text);
  }
  tr.moving {
    opacity: 0.45;
  }

  td {
    padding: 8px 10px;
    font-size: 11px;
    border-bottom: 0.5px solid var(--f-border);
    white-space: nowrap;
    color: var(--f-text2);
  }

  .nm {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--f-text);
  }
  tr.selected .nm {
    font-weight: 500;
  }
  .tag-cell {
    max-width: 180px;
  }
  .tag {
    display: inline-block;
    margin-right: 4px;
    border-radius: 10px;
    padding: 1px 6px;
    font-family: var(--mono);
    border: 0.5px solid color-mix(in srgb, var(--tc) 24%, transparent);
    background: color-mix(in srgb, var(--tc) 12%, white);
  }
  tr.selected .tag {
    border-color: var(--f-accent-border);
    background: rgba(255, 255, 255, 0.75);
    color: var(--f-text2);
  }
  .actions {
    opacity: 0;
    text-align: right;
    width: 82px;
  }
  tr:hover .actions,
  tr.selected .actions {
    opacity: 1;
  }
  .actions button {
    border: none;
    background: transparent;
    color: var(--f-text2);
    width: 22px;
    height: 22px;
    border-radius: 4px;
  }
  .actions button:hover {
    background: rgba(255, 255, 255, 0.8);
  }
  tr.selected .actions button:hover {
    background: rgba(43, 92, 230, 0.1);
  }
  .actions .danger {
    color: var(--f-danger);
  }
</style>
