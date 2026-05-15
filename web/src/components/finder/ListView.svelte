<script lang="ts">
  import { api, getDirectory, getLocationDirectory, getLocationDirs, listDirectories } from "../../api";
  import type { DirEntry, FinderLocation } from "../../types";
  import { toast } from "svelte-sonner";
  import type { DragDropState } from "@thisux/sveltednd";
  import ListDirectoryRow from "./ListDirectoryRow.svelte";
  import ListFileRow from "./ListFileRow.svelte";
  import { entryPath, joinPath } from "./entryPaths";

  export let prefix = "/";
  export let location: FinderLocation = { kind: "local" };
  export let activeTag = "";
  export let selectedEntry: DirEntry | null = null;
  export let invalidate = 0;
  export let onSelect: (entry: DirEntry) => void = () => {};
  export let onOpen: (entry: DirEntry) => void = () => {};
  export let onContextMenu: (e: MouseEvent, entry: DirEntry) => void = () => {};
  export let onQuickSend: (entry: DirEntry) => void = () => {};
  export let onQuickDownload: (entry: DirEntry) => void = () => {};
  export let onQuickDelete: (entry: DirEntry) => void = () => {};
  export let onEntriesLoaded: (prefix: string, entries: DirEntry[]) => void = () => {};

  let entries: DirEntry[] = [];
  let sortField: "name" | "size" | "modified" | "version" = "name";
  let sortDir: 1 | -1 = 1;
  let dragTarget = "";
  let movingFileId = "";
  let loadSeq = 0;

  function baseSort(items: DirEntry[]) {
    return [...items].sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
      if (sortField === "size") return ((a.file?.size ?? 0) - (b.file?.size ?? 0)) * sortDir;
      if (sortField === "modified") return String(a.file?.updated_at ?? "").localeCompare(String(b.file?.updated_at ?? "")) * sortDir;
      if (sortField === "version") return ((a.file?.version ?? 0) - (b.file?.version ?? 0)) * sortDir;
      return a.name.localeCompare(b.name) * sortDir;
    });
  }

  function remoteRootEntries(listings: { prefix: string }[]): DirEntry[] {
    return listings.map((listing) => {
      const trimmed = listing.prefix.replace(/\/$/, "");
      const name = trimmed.split("/").filter(Boolean).at(-1) ?? listing.prefix;
      return {
        name,
        is_dir: true,
        prefix: listing.prefix,
        file_count: 0,
        stats: { total_size: 0 },
      } as DirEntry;
    });
  }

  async function load() {
    const seq = ++loadSeq;
    const listing = location.kind === "remote" && location.hostname
      ? (prefix === "/"
        ? { prefix, entries: remoteRootEntries(await getLocationDirs(location.hostname)) }
        : await getLocationDirectory(location.hostname, prefix))
      : (prefix === "/" ? await listDirectories(activeTag) : await getDirectory(prefix, activeTag));
    if (seq !== loadSeq) return;
    entries = baseSort(listing.entries);
    onEntriesLoaded(prefix, entries);
  }

  function setSort(field: typeof sortField) {
    if (sortField === field) sortDir = sortDir === 1 ? -1 : 1;
    else {
      sortField = field;
      sortDir = 1;
    }
    entries = baseSort(entries);
    onEntriesLoaded(prefix, entries);
  }

  function isSelected(entry: DirEntry) {
    if (!selectedEntry) return false;
    if (entry.is_dir) return selectedEntry.prefix === entry.prefix;
    return entryPath(selectedEntry) === entryPath(entry) &&
      (selectedEntry.file?.hostname ?? location.hostname ?? "") === (entry.file?.hostname ?? location.hostname ?? "");
  }

  async function handleDrop(state: DragDropState<DirEntry>, targetDir: DirEntry) {
    if (location.kind !== "local") return;
    dragTarget = "";
    const dragged = state.draggedItem;
    if (!dragged?.file || !targetDir.prefix) return;
    const newPath = joinPath(targetDir.prefix, dragged.name);
    if (dragged.file.path === newPath) return;
    movingFileId = dragged.file.id;
    try {
      await api.moveFile(dragged.file.id, newPath);
      entries = baseSort(entries.filter((entry) => entry.file?.id !== dragged.file?.id));
      onEntriesLoaded(prefix, entries);
      toast.success(`Moved ${dragged.name}`);
    } catch (e: unknown) {
      toast.error((e as Error).message);
    } finally {
      movingFileId = "";
    }
  }

  $: prefix, activeTag, invalidate, load();
  $: selectedKey = selectedEntry?.prefix ?? selectedEntry?.file?.id ?? "none";
</script>

<div class="list-wrap">
  <table class="list">
    <thead>
      <tr>
        <th>
          <button class:active-sort={sortField === "name"} class="sort-btn" on:click={() => setSort("name")}>
            <span>Name</span>
            <span class="sort-ind" aria-hidden="true">{sortField === "name" ? (sortDir === 1 ? "↑" : "↓") : ""}</span>
          </button>
        </th>
        <th>Kind</th>
        <th>
          <button class:active-sort={sortField === "size"} class="sort-btn" on:click={() => setSort("size")}>
            <span>Size</span>
            <span class="sort-ind" aria-hidden="true">{sortField === "size" ? (sortDir === 1 ? "↑" : "↓") : ""}</span>
          </button>
        </th>
        <th>
          <button class:active-sort={sortField === "version"} class="sort-btn" on:click={() => setSort("version")}>
            <span>Ver</span>
            <span class="sort-ind" aria-hidden="true">{sortField === "version" ? (sortDir === 1 ? "↑" : "↓") : ""}</span>
          </button>
        </th>
        <th>Tags</th>
        <th>By</th>
        <th>
          <button class:active-sort={sortField === "modified"} class="sort-btn" on:click={() => setSort("modified")}>
            <span>Modified</span>
            <span class="sort-ind" aria-hidden="true">{sortField === "modified" ? (sortDir === 1 ? "↑" : "↓") : ""}</span>
          </button>
        </th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#key selectedKey}
      {#each entries as entry}
        {#if entry.is_dir}
          <ListDirectoryRow
            {entry}
            {prefix}
            selected={isSelected(entry)}
            {onSelect}
            {onOpen}
            {onContextMenu}
            {onQuickSend}
            {onQuickDownload}
            {onQuickDelete}
            droppableEnabled={location.kind === "local"}
            onDragEnter={(entry) => dragTarget = entry.prefix || ""}
            onDragLeave={() => dragTarget = ""}
            onDrop={handleDrop}
            dragTarget={dragTarget === entry.prefix}
          />
        {:else}
          <ListFileRow
            {entry}
            {prefix}
            selected={isSelected(entry)}
            {onSelect}
            {onOpen}
            {onContextMenu}
            {onQuickSend}
            {onQuickDownload}
            {onQuickDelete}
            moving={movingFileId === entry.file?.id}
            dragEnabled={location.kind === "local"}
          />
        {/if}
      {/each}
      {/key}
    </tbody>
  </table>
</div>

<style>
  .list-wrap {
    flex: 1;
    overflow: auto;
    background: var(--f-surface);
  }
  .list {
    width: 100%;
    border-collapse: collapse;
  }

  th {
    padding: 8px 10px;
    font-size: 10.5px;
    border-bottom: 0.5px solid var(--f-border);
    white-space: nowrap;
    color: var(--f-text2);
    position: sticky;
    top: 0;
    background: var(--f-bg1);
    text-align: left;
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .sort-btn {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    border: none;
    padding: 0;
    background: transparent;
    color: inherit;
    font: inherit;
    text-transform: inherit;
    letter-spacing: inherit;
    cursor: pointer;
  }
  .sort-btn.active-sort {
    color: var(--f-text);
  }
  .sort-ind {
    display: inline-block;
    min-width: 10px;
    text-align: right;
    color: var(--f-text3);
  }
  .sort-btn.active-sort .sort-ind {
    color: var(--f-accent);
  }
</style>
