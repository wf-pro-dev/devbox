<script lang="ts">
  import { api, getDirectory, getLocationDirectory, getLocationDirs, listDirectories } from "../../api";
  import type { DirEntry, FinderLocation, DirListing } from "../../types";
  import { toast } from "svelte-sonner";
  import { droppable, type DragDropState } from "@thisux/sveltednd";
  import DirColumnEntry from "./DirColumnEntry.svelte";
  import { entryPath, joinPath, pathSegments } from "./entryPaths";

  export let prefix = "/";
  export let location: FinderLocation = { kind: "local" };
  export let activeTag = "";
  export let selectedEntry: DirEntry | null = null;
  export let depth = 0;
  export let invalidate = 0;
  export let onSelect: (entry: DirEntry) => void = () => {};
  export let onOpen: (entry: DirEntry) => void = () => {};
  export let onContextMenu: (e: MouseEvent, entry: DirEntry) => void = () => {};
  export let onEntriesLoaded: (prefix: string, entries: DirEntry[]) => void = () => {};
  export let onMove: (payload: { fileId: string; fromPath: string; toPath: string; sourcePrefix: string; targetPrefix: string }) => void = () => {};

  let entries: DirEntry[] = [];
  let loading = true;
  let error = "";
  let movingFileId = "";
  let columnDropActive = false;
  let rowDropTarget = "";
  let loadSeq = 0;

  function sortEntries(items: DirEntry[]) {
    return [...items].sort((a, b) => {
      if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
      return a.name.localeCompare(b.name);
    });
  }

  function canonicalPrefix(prefix: string) {
    if (!prefix || prefix === "/") return "/";
    let normalized = prefix.startsWith("/") ? prefix : `/${prefix}`;
    if (!normalized.endsWith("/")) normalized += "/";
    return normalized;
  }

  function prefixToColumns(prefix: string) {
    const canonical = canonicalPrefix(prefix);
    if (canonical === "/") return ["/"];
    const segments = canonical.replace(/^\/|\/$/g, "").split("/").filter(Boolean);
    const result = ["/"];
    let current = "/";
    for (const segment of segments) {
      current = current === "/" ? `/${segment}/` : `${current}${segment}/`;
      result.push(current);
    }
    console.log(prefix, result.length);
    return result;
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
        baseLength: prefixToColumns(listing.prefix).length - 2,
      } as DirEntry;
    });
  }

  function filtered(items: DirEntry[]) {
    if (!activeTag) return items;
    return items.filter((entry) => entry.file?.tags?.includes(activeTag) || false);
  }

  function isSelected(entry: DirEntry) {
    if (!selectedEntry) return false;
    const selectedPath = entryPath(selectedEntry);
    const currentPath = entryPath(entry);
    if (!selectedPath || !currentPath) return false;
    const selectedParts = pathSegments(selectedPath);
    const currentParts = pathSegments(currentPath);

    const compareLen = depth + 1;
    if (selectedParts.length < compareLen || currentParts.length < compareLen) return false;
    for (let i = 0; i < compareLen; i++) {
      if (selectedParts[i] !== currentParts[i]) return false;
    }
    return true;
  }

  async function handleDropToPrefix(state: DragDropState<DirEntry>, targetPrefix: string) {
    if (location.kind !== "local") return;
    columnDropActive = false;
    const dragged = state.draggedItem;
    if (!dragged?.file) return;

    if (targetPrefix.length == 0 || targetPrefix == "" || targetPrefix[0] !== '/') targetPrefix = '/' + targetPrefix;
    const newPath = joinPath(targetPrefix, dragged.name);

    if (dragged.file.path === newPath) return;
    movingFileId = dragged.file.id!;
    try {
      await api.moveFile(dragged.file.id!, newPath);
      const sourcePrefix = dragged.file.path.includes("/") ? dragged.file.path.slice(0, dragged.file.path.lastIndexOf("/")) || "/" : "/";
      onMove({
        fileId: dragged.file.id!,
        fromPath: dragged.file.path,
        toPath: newPath,
        sourcePrefix,
        targetPrefix,
      });
      toast.success(`Moved ${dragged.name}`);
    } catch (e: unknown) {
      toast.error((e as Error).message);
    } finally {
      movingFileId = "";
    }
  }

  function onDragEnter(state: DragDropState<DirEntry>) {
    if (location.kind !== "local") return;
    const dragged = state.draggedItem;

    if (!dragged?.file ) return;
   
    const normalizedPrefix = prefix[0] === "/" ? prefix : `/${prefix}`;
    const newPath = joinPath(normalizedPrefix, dragged.name);
    if (dragged.file.path === newPath) return;
    columnDropActive = true;
  }

  async function GetRemoteDirectory(prefix: string,location: FinderLocation): Promise<DirListing> {
    let baseLength = prefixToColumns(prefix).length - 2;
    let path = selectedEntry?.prefix?.split("/").slice(selectedEntry?.baseLength ?? baseLength).join("/") ?? prefix
    let directory = await getLocationDirectory(location.hostname!, canonicalPrefix(path));
    return {
      ...directory,
      baseLength: baseLength,

    }
  }

  async function load() {
    const seq = ++loadSeq;
    loading = true;
    error = "";
    try {
      const listing = location.kind === "remote" && location.hostname
        ? (prefix === "/"
          ? { prefix, entries: remoteRootEntries(await getLocationDirs(location.hostname)) }
          : await GetRemoteDirectory(
            prefix,
            location
          ))
        : (prefix === "/" ? await listDirectories(activeTag) : await getDirectory(prefix, activeTag));
      if (seq !== loadSeq) return;
      entries = sortEntries(filtered(listing.entries));
      onEntriesLoaded(prefix, entries);
    } catch (e: unknown) {
      if (seq !== loadSeq) return;
      error = (e as Error).message;
      entries = [];
      onEntriesLoaded(prefix, []);
    } finally {
      console.log("loading", seq !== loadSeq, seq, loadSeq);
      if (seq !== loadSeq) return;
      loading = false;
    }
  }

  $: prefix, activeTag, invalidate, load();
  $: selectedKey = selectedEntry?.prefix ?? selectedEntry?.file?.id ?? "none";
</script>

<div
  class="column"
  class:column-drop-active={columnDropActive && !rowDropTarget}
  use:droppable={location.kind === "local" ? {
    container: prefix,
    callbacks: {
      onDragEnter: onDragEnter ,
      onDragLeave: () => columnDropActive = false,
      onDrop: (state: DragDropState<DirEntry>) => handleDropToPrefix(state, prefix),
    },
  } : {
    container: "",
    callbacks: undefined,
  }}
>
  {#if loading}
    <div class="state">Loading…</div>
  {:else if error}
    <div class="state err">{error}</div>
  {:else}
    {#key selectedKey}
    {#each entries as entry}
      <DirColumnEntry
        {entry}
        {prefix}
        selected={isSelected(entry)}
        moving={movingFileId === entry.file?.id}
        dragTarget={rowDropTarget === entry.prefix}
        {onSelect}
        {onOpen}
        {onContextMenu}
        onDragEnter={() => rowDropTarget = entry.prefix || ""}
        onDragLeave={() => rowDropTarget = ""}
        onDrop={(state, entry) => {
          rowDropTarget = "";
          handleDropToPrefix(state, entry.prefix || "");
        }}
      />
    {/each}
    {/key}
  {/if}
</div>

<style>
  .column {
    width: 252px;
    min-width: 252px;
    border-right: 0.5px solid var(--f-border);
    background: var(--f-surface);
    overflow-y: auto;
  }
  .column.column-drop-active {
    background: color-mix(in srgb, var(--f-selection) 55%, white);
    box-shadow: inset 0 0 0 1.5px var(--f-accent-border);
  }
  .state {
    padding: 12px;
    font-size: 11px;
    color: var(--f-text3);
  }
  .err {
    color: var(--f-danger);
  }
</style>
