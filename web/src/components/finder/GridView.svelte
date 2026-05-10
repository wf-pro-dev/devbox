<script lang="ts">
  import { api, getDirectory, listDirectories } from "../../api";
  import type { DirEntry } from "../../types";
  import { fileIcon, folderIcon } from "./icons";
  import { draggable, droppable, type DragDropState } from "@thisux/sveltednd";
  import { toast } from "svelte-sonner";
  import { joinPath } from "./entryPaths";

  export let prefix = "/";
  export let activeTag = "";
  export let selectedEntry: DirEntry | null = null;
  export let iconSize = 84;
  export let invalidate = 0;
  export let onSelect: (entry: DirEntry) => void = () => {};
  export let onOpen: (entry: DirEntry) => void = () => {};
  export let onContextMenu: (e: MouseEvent, entry: DirEntry) => void = () => {};
  export let onQuickSend: (entry: DirEntry) => void = () => {};
  export let onQuickDownload: (entry: DirEntry) => void = () => {};
  export let onQuickDelete: (entry: DirEntry) => void = () => {};
  export let onMove: () => void = () => {};
  export let onEntriesLoaded: (prefix: string, entries: DirEntry[]) => void = () => {};

  let entries: DirEntry[] = [];
  let dragTarget = "";
  let movingFileId = "";

  async function load() {
    const listing = prefix === "/" ? await listDirectories(activeTag) : await getDirectory(prefix, activeTag);
    entries = [...listing.entries].sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
      return a.name.localeCompare(b.name);
    });
    onEntriesLoaded(prefix, entries);
  }

  async function handleDrop(state: DragDropState<DirEntry>, targetDir: DirEntry) {
    dragTarget = "";
    const dragged = state.draggedItem;
    if (!dragged?.file || !targetDir.prefix) return;
    const newPath = joinPath(targetDir.prefix, dragged.name);
    if (dragged.file.path === newPath) return;
    movingFileId = dragged.file.id;
    try {
      await api.moveFile(dragged.file.id, newPath);
      entries = entries.filter((entry) => entry.file?.id !== dragged.file?.id);
      toast.success(`Moved ${dragged.name}`);
      onMove();
    } catch (e: unknown) {
      toast.error((e as Error).message);
    } finally {
      movingFileId = "";
    }
  }

  $: dirEntries = entries.filter((entry) => entry.is_dir);
  $: fileEntries = entries.filter((entry) => !entry.is_dir);
  $: prefix, activeTag, invalidate, load();
</script>

<div class="grid-host">
  {#if dirEntries.length > 0}
    <div class="sec-label">Folders</div>
    <div class="grid-root" style="--icon-size:{iconSize}px">
      {#each dirEntries as entry}
        <div
          class="gc"
          class:sel={selectedEntry?.prefix === entry.prefix}
          class:drop-target={dragTarget === entry.prefix}
          role="button"
          tabindex="0"
          use:droppable={{ container: prefix, callbacks: { onDragEnter: () => dragTarget = entry.prefix || "", onDragLeave: () => dragTarget = "", onDrop: (data: DirEntry) => handleDrop(data, entry) } }}
          on:click={() => onSelect(entry)}
          on:dblclick={() => onOpen(entry)}
          on:keydown={(e) => e.key === "Enter" && onOpen(entry)}
          on:contextmenu|preventDefault={(e) => onContextMenu(e, entry)}
        >
          {@html folderIcon(dragTarget === entry.prefix ? "drop" : "default")}
          <span class="gc-name">{entry.name}</span>
        </div>
      {/each}
    </div>
  {/if}
  {#if fileEntries.length > 0}
    <div class="sec-label">Files</div>
    <div class="grid-root" style="--icon-size:{iconSize}px">
      {#each fileEntries as entry}
        <div
          class="gc"
          class:sel={selectedEntry?.file?.id === entry.file?.id}
          class:moving={movingFileId === entry.file?.id}
          role="button"
          tabindex="0"
          use:draggable={{ container: prefix, dragData: entry }}
          on:click={() => onSelect(entry)}
          on:dblclick={() => onOpen(entry)}
          on:keydown={(e) => e.key === "Enter" && onOpen(entry)}
          on:contextmenu|preventDefault={(e) => onContextMenu(e, entry)}
        >
          {@html fileIcon(entry.file?.language || "text", iconSize / 2.4)}
          <span class="gc-name">{entry.name}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .grid-host {
    flex: 1;
    overflow: auto;
    background: var(--f-surface);
    padding: 10px;
  }
  .sec-label {
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--f-text3);
    margin: 2px 0 8px;
  }
  .grid-root {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(var(--icon-size), 1fr));
    gap: 5px;
    margin-bottom: 14px;
  }
  .gc {
    position: relative;
    min-height: calc(var(--icon-size) + 18px);
    border: 0.5px solid transparent;
    border-radius: 6px;
    padding: 8px 6px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: flex-start;
    gap: 7px;
    text-align: center;
  }
  .gc:hover {
    background: var(--f-bg2);
  }
  .gc.sel {
    background: var(--f-selection);
    border-color: var(--f-accent-border);
  }
  .gc.moving {
    opacity: 0.45;
  }
  .gc.drop-target {
    background: rgba(43, 92, 230, 0.12);
    outline: 1.5px dashed var(--f-accent);
  }
  .gc-name {
    font-family: var(--mono);
    font-size: 10px;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
