<script lang="ts">
  import { draggable, droppable, type DragDropState } from "@thisux/sveltednd";
  import { formatBytes } from "../../api";
  import type { DirEntry } from "../../types";
  import { fileIcon, folderIcon } from "./icons";

  export let entry: DirEntry;
  export let prefix = "/";
  export let selected = false;
  export let moving = false;
  export let dragTarget = false;
  export let onSelect: (entry: DirEntry) => void = () => {};
  export let onOpen: (entry: DirEntry) => void = () => {};
  export let onContextMenu: (e: MouseEvent, entry: DirEntry) => void = () => {};
  export let onDragEnter: (entry: DirEntry) => void = () => {};
  export let onDragLeave: (entry: DirEntry) => void = () => {};
  export let onDrop: (state: DragDropState<DirEntry>, entry: DirEntry) => void = () => {};

  function handleClick() {
    if (entry.is_dir) onOpen(entry);
    else onSelect(entry);
  }
</script>

<div
  class="row"
  class:selected
  class:drag-target={dragTarget}
  class:moving
  role="button"
  tabindex="0"
  use:draggable={!entry.is_dir ? { container: prefix, dragData: entry } : {container: "", dragData: entry}}
  use:droppable={entry.is_dir
    ? {
        container: prefix,
        callbacks: {
          onDragEnter: () => onDragEnter(entry),
          onDragLeave: () => onDragLeave(entry),
          onDrop: (state: DragDropState<DirEntry>) => onDrop(state, entry),
        },
      }
    : { container: "", dragData: entry }}
  on:click={handleClick}
  on:dblclick={() => onOpen(entry)}
  on:keydown={(e) => e.key === "Enter" && onOpen(entry)}
  on:contextmenu|preventDefault={(e) => onContextMenu(e, entry)}
>
  <div class="icon">
    {@html entry.is_dir
      ? folderIcon(dragTarget ? "drop" : selected ? "selected" : "default", true)
      : fileIcon(selected ? "selected" : entry.file?.language || "text", 16)}
  </div>
  <div class="name">{entry.name}</div>
  <div class="meta">{entry.is_dir ? `${entry.file_count ?? 0}` : formatBytes(entry.file?.size ?? 0)}</div>
  {#if entry.is_dir}
    <i class="ti ti-chevron-right chev"></i>
  {/if}
</div>

<style>
  .row {
    position: relative;
    display: flex;
    align-items: center;
    gap: 7px;
    min-height: 26px;
    padding: 4px 10px 4px 8px;
    color: var(--f-text);
    border-radius: 6px;
    margin: 1px 4px;
    outline: none;
  }
  .row:hover {
    background: var(--f-bg2);
  }
  .row.selected {
    background: var(--f-selection);
    color: var(--f-text);
  }
  .row.selected:hover {
    background: var(--f-selection);
  }
  .row.drag-target {
    background: rgba(43, 92, 230, 0.12);
    outline: 1.5px dashed var(--f-accent);
  }
  .row.moving {
    opacity: 0.45;
  }
  .row:focus-visible {
    box-shadow: inset 0 0 0 1px var(--f-accent-border);
    background: color-mix(in srgb, var(--f-selection) 70%, white);
  }
  .icon {
    width: 16px;
    display: flex;
    justify-content: center;
    flex-shrink: 0;
  }
  .name {
    font-family: var(--mono);
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex: 1;
  }
  .meta, .chev {
    font-family: var(--mono);
    font-size: 10px;
    color: var(--f-text3);
  }
  .row.selected .meta,
  .row.selected .chev {
    color: var(--f-text2);
  }
  .row.selected .name {
    font-weight: 500;
  }
</style>
