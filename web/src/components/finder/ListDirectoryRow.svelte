<script lang="ts">
  import { formatBytes, formatDateShort } from "../../api";
  import { droppable, type DragDropState } from "@thisux/sveltednd";
  import type { DirEntry } from "../../types";
  import { folderIcon } from "./icons";

  export let entry: DirEntry;
  export let prefix = "/";
  export let selected = false;
  export let dragTarget = false;
  export let totalSize = 0;
  export let latestUpdated = "";
  export let oldestEntry: DirEntry | null = null;
  export let onSelect: (entry: DirEntry) => void = () => {};
  export let onOpen: (entry: DirEntry) => void = () => {};
  export let onContextMenu: (e: MouseEvent, entry: DirEntry) => void = () => {};
  export let onQuickSend: (entry: DirEntry) => void = () => {};
  export let onQuickDownload: (entry: DirEntry) => void = () => {};
  export let onQuickDelete: (entry: DirEntry) => void = () => {};
  export let onDragEnter: (entry: DirEntry) => void = () => {};
  export let onDragLeave: (entry: DirEntry) => void = () => {};
  export let onDrop: (state: DragDropState<DirEntry>, entry: DirEntry) => void = () => {};
</script>

<tr
  class:selected
  class:drag-target={dragTarget}
  class="container"
  use:droppable={{
    container: prefix,
    callbacks: {
      onDragEnter: () => onDragEnter(entry),
      onDragLeave: () => onDragLeave(entry),
      onDrop: (state: DragDropState<DirEntry>) => onDrop(state, entry),
    },
  }}
  on:click={() => onSelect(entry)}
  on:dblclick={() => onOpen(entry)}
  on:contextmenu|preventDefault={(e) => onContextMenu(e, entry)}
>
  <td>
    <span class="ic">
      {@html folderIcon(dragTarget ? "drop" : selected ? "selected" : "default", true)}
    </span>
    <span class="nm">{entry.name}/</span>
  </td>
  <td >folder</td>
  <td>{formatBytes(totalSize)}</td>
  <td >-</td>
  <td >—</td>
  <td >{oldestEntry ? oldestEntry.file?.uploaded_by : "-"}</td>
  <td>{latestUpdated ? formatDateShort(latestUpdated) : "—"}</td>
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
  tr.drag-target {
    background: rgba(43, 92, 230, 0.12);
    outline: 1.5px dashed var(--f-accent);
    outline-offset: -1px;
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
  .actions {
    width: 82px;
    opacity: 0;
    text-align: right;
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
