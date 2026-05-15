<script lang="ts">
  import { onMount } from "svelte";
  import { api, listPeers, deleteDirectory } from "../../api";
  import type { DirEntry, File, FinderLocation, HealthResponse, Peer } from "../../types";
  import { toast } from "svelte-sonner";
  import SendModal from "../SendModal.svelte";
  import UploadModal from "../UploadModal.svelte";
  import PreviewModal from "../PreviewModal.svelte";
  import ContextMenu from "./ContextMenu.svelte";
  import DirColumn from "./DirColumn.svelte";
  import FinderSidebar from "./FinderSidebar.svelte";
  import FinderToolbar from "./FinderToolbar.svelte";
  import GridView from "./GridView.svelte";
  import ListView from "./ListView.svelte";
  import PreviewPane from "./PreviewPane.svelte";
  import RenameFileModal from "./RenameFileModal.svelte";
  import { tagColor } from "./fileColor";
  import { entryPath } from "./entryPaths";

  export let health: HealthResponse | null = null;
  export let onFileSelect: (f: File) => void = () => {};

  let columns: string[] = ["/"];
  let location: FinderLocation = { kind: "local" };
  let selectedEntry: DirEntry | null = null;
  let viewMode: "column" | "list" | "grid" = "column";
  let activeTag = "";
  let iconSize = 84;
  let ctxMenu: { x: number; y: number; entry: DirEntry | null } | null = null;
  let showSend = false;
  let showUpload = false;
  let peers: Peer[] = [];
  let columnInvalidators: Record<string, number> = {};
  let previewFile: File | null = null;
  let visibleEntries: Record<string, DirEntry[]> = {};
  let backHistory: string[] = [];
  let forwardHistory: string[] = [];
  let renameFile: File | null = null;
  let renaming = false;


  $: currentPrefix = columns.at(-1) ?? "/";
  $: locationLabel = location.kind === "remote" ? location.hostname ?? "remote" : "devbox";
  $: allTags = Object.entries(
    Object.values(visibleEntries)
      .flat()
      .flatMap((entry) => entry.file?.tags ?? [])
      .reduce<Record<string, number>>((acc, tag) => {
        acc[tag] = (acc[tag] ?? 0) + 1;
        return acc;
      }, {}),
  )
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, count]) => ({ name, count, color: tagColor(name) }));

  onMount(async () => {
    peers = await listPeers().catch(() => []);
  });

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
    return result;
  }

  function locationKey(loc: FinderLocation) {
    return loc.kind === "remote" ? `remote:${loc.hostname ?? ""}` : "local";
  }

  function scopedPrefixKey(prefix: string, loc: FinderLocation = location) {
    return `${locationKey(loc)}:${canonicalPrefix(prefix)}`;
  }

  function entryIdentity(entry: DirEntry | null) {
    if (!entry) return "";
    if (entry.is_dir) return `dir:${canonicalPrefix(entry.prefix ?? "/")}`;
    const path = entry.file?.path ?? entryPath(entry);
    const host = entry.file?.hostname ?? location.hostname ?? "";
    const source = entry.file?.source ?? location.kind;
    return `file:${source}:${host}:${path}`;
  }

  function navigateToPrefix(
    nextPrefix: string,
    mode: "push" | "back" | "forward" | "replace" = "push",
    clearSelection = true,
  ) {
    const canonicalNextPrefix = canonicalPrefix(nextPrefix);
    if (canonicalNextPrefix === currentPrefix) return;

    if (mode === "push") {
      backHistory = [...backHistory, currentPrefix];
      forwardHistory = [];
    } else if (mode === "back") {
      forwardHistory = [currentPrefix, ...forwardHistory];
    } else if (mode === "forward") {
      backHistory = [...backHistory, currentPrefix];
    }

    columns = prefixToColumns(canonicalNextPrefix);
    if (clearSelection) selectedEntry = null;
  }

  function switchLocation(next: FinderLocation) {
    location = next;
    columns = ["/"];
    selectedEntry = null;
    previewFile = null;
    visibleEntries = {};
    backHistory = [];
    forwardHistory = [];
  }

  function selectEntry(entry: DirEntry, colDepth: number) {
    selectedEntry = entry;
    columns = columns.slice(0, colDepth + 1);
    if (entry.file) onFileSelect(entry.file);
  }

  function openEntry(entry: DirEntry) {
    selectedEntry = entry;
    if (entry.is_dir && entry.prefix) {
      let prefix = entry.prefix;
      if (location.kind === "remote") {
        // Remove the base length of the prefix to get the relative prefix
        columns = entry.prefix.split("/").filter((segment) => segment !== "");
        prefix = canonicalPrefix(columns.slice(entry.baseLength ?? 0).join("/"));
      }
      navigateToPrefix(prefix, "push", false);
      return;
    }
    if (entry.file) {
      previewFile = entry.file;
      onFileSelect(entry.file);
    }
  }

  function navigateTo(segIdx: number) {
    navigateToPrefix(columns[segIdx] ?? "/");
  }

  function onNavigateBack() {
    const previousPrefix = backHistory.at(-1);
    if (!previousPrefix) return;
    backHistory = backHistory.slice(0, -1);
    navigateToPrefix(previousPrefix, "back");
  }

  function onNavigateForward() {
    const nextPrefix = forwardHistory[0];
    if (!nextPrefix) return;
    forwardHistory = forwardHistory.slice(1);
    navigateToPrefix(nextPrefix, "forward");
  }

  function invalidateColumn(prefix: string) {
    const key = scopedPrefixKey(prefix);
    columnInvalidators = { ...columnInvalidators, [key]: Date.now() };
  }

  function handleColumnMove(payload: {
    fileId: string;
    fromPath: string;
    toPath: string;
    sourcePrefix: string;
    targetPrefix: string;
  }) {
    const sourcePrefix = canonicalPrefix(payload.sourcePrefix);
    const targetPrefix = canonicalPrefix(payload.targetPrefix);

    invalidateColumn(sourcePrefix);
    invalidateColumn(targetPrefix);

    for (const openPrefix of columns) {
      const normalized = canonicalPrefix(openPrefix);
      if (
        normalized === sourcePrefix ||
        normalized === targetPrefix ||
        normalized.startsWith(`${sourcePrefix}/`) ||
        normalized.startsWith(`${targetPrefix}/`)
      ) {
        invalidateColumn(openPrefix);
      }
    }

    if (selectedEntry?.file?.id === payload.fileId && selectedEntry.file) {
      const file = { ...selectedEntry.file, path: payload.toPath, file_name: payload.toPath.split("/").pop() || selectedEntry.file.file_name };
      selectedEntry = { ...selectedEntry, name: file.file_name, file };
      if (previewFile?.id === payload.fileId) previewFile = file;
    }
  }

  function updateEntries(prefix: string, entries: DirEntry[]) {
    const key = scopedPrefixKey(prefix);
    visibleEntries = { ...visibleEntries, [key]: entries };
  }

  function showCtxMenu(e: MouseEvent, entry: DirEntry | null) {
    e.preventDefault();
    ctxMenu = { x: e.clientX, y: e.clientY, entry };
  }

  function handleDelete(entry: DirEntry) {
    if (location.kind !== "local") {
      toast.error("Delete is only available for devbox files");
      return;
    }
    const label = entry.is_dir ? `directory "${entry.name}"` : `"${entry.name}"`;
    toast(`Delete ${label}?`, {
      action: {
        label: "Confirm",
        onClick: async () => {
          try {
            if (entry.is_dir) await deleteDirectory(entry.prefix!);
            else await api.deleteFile(entry.file!.id);
            invalidateColumn(currentPrefix);
            if (selectedEntry === entry) selectedEntry = null;
            toast.success(`Deleted ${label}`);
          } catch (e: unknown) {
            toast.error((e as Error).message);
          }
        },
      },
    });
  }

  function handleDownload(entry: DirEntry | null = selectedEntry) {
    if (!entry) return;
    const a = document.createElement("a");
    if (location.kind === "remote" && location.hostname && entry.file) {
      const qs = new URLSearchParams({ path: entry.file.path });
      a.href = `/locations/${location.hostname}/files?${qs}`;
      a.download = entry.file.file_name;
      a.click();
      return;
    }
    a.href = entry.is_dir ? `/dirs/${encodeURIComponent(entry.prefix ?? "")}?content=true` : `/files/${entry.file?.id}`;
    if (!entry.is_dir && entry.file) a.download = entry.file.file_name;
    a.click();
  }

  function handleCopyPath(entry: DirEntry | null = selectedEntry) {
    const path = entry?.prefix ?? entry?.file?.path ?? "";
    navigator.clipboard.writeText(path);
    toast.success("Path copied");
  }

  async function submitRename(path: string) {
    if (!renameFile?.id) return;
    if (!renameFile) return;
    renaming = true;
    try {
      await api.moveFile(renameFile.id, path);
      renameFile = null;
      invalidateColumn(currentPrefix);
      toast.success("File renamed");
    } catch (e: unknown) {
      toast.error((e as Error).message);
    } finally {
      renaming = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    const tag = (e.target as HTMLElement).tagName;
    if (tag === "INPUT" || tag === "TEXTAREA") return;
    if (e.key === "Backspace" && selectedEntry) handleDelete(selectedEntry);
    if (e.key === " " && selectedEntry?.file) {
      e.preventDefault();
      previewFile = selectedEntry.file;
    }
    if (e.key === "ArrowUp" && e.metaKey) onNavigateBack();
    if (e.key === "ArrowDown" && e.metaKey) onNavigateForward();
    if (e.key === "Escape") {
      ctxMenu = null;
      selectedEntry = null;
    }
  }

  $: canNavigateBack = backHistory.length > 0;
  $: canNavigateForward = forwardHistory.length > 0;
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="finder-root" on:contextmenu|self={(e) => showCtxMenu(e, null)}>
  

  <FinderToolbar
    prefix={currentPrefix}
    {viewMode}
    {activeTag}
    {selectedEntry}
    {iconSize}
    onNavigate={navigateTo}
    {canNavigateBack}
    {canNavigateForward}
    onViewChange={(v) => viewMode = v}
    onTagToggle={(t) => activeTag = activeTag === t ? "" : t}
    onUpload={() => showUpload = true}
    onSend={() => showSend = true}
    onDelete={() => selectedEntry && handleDelete(selectedEntry)}
    onStatus={() => {}}
    onDiff={() => {}}
    onNavigateBack={onNavigateBack}
    onNavigateForward={onNavigateForward}
    onIconSize={(s) => iconSize = s}
  />

  <div class="finder-body">
      <FinderSidebar
      {health}
      {peers}
      {activeTag}
      {allTags}
      onSelectTag={(t) => activeTag = activeTag === t ? "" : t}
      onSelectRoot={() => {
        switchLocation({ kind: "local" });
      }}
      onSelectLocation={(hostname) => {
        switchLocation({ kind: "remote", hostname });
      }}
    />

    <div class="finder-content">
      {#if viewMode === "column"}
        <div class="finder-columns">
          {#each columns as prefix, i (scopedPrefixKey(prefix) + (columnInvalidators[scopedPrefixKey(prefix)] ?? 0))}
            <DirColumn
              {prefix}
              {location}
              {activeTag}
              {selectedEntry}
              depth={i}
              invalidate={columnInvalidators[scopedPrefixKey(prefix)] ?? 0}
              onSelect={(entry) => selectEntry(entry, i)}
              onOpen={openEntry}
              onContextMenu={showCtxMenu}
              onMove={handleColumnMove}
              onEntriesLoaded={updateEntries}
            />
          {/each}
        </div>
      {:else if viewMode === "list"}
        <ListView
          prefix={currentPrefix}
          {location}
          {activeTag}
          {selectedEntry}
          invalidate={columnInvalidators[scopedPrefixKey(currentPrefix)] ?? 0}
          onSelect={(entry) => selectEntry(entry, columns.length - 1)}
          onOpen={openEntry}
          onContextMenu={showCtxMenu}
          onQuickSend={(entry) => { selectedEntry = entry; showSend = true; }}
          onQuickDownload={handleDownload}
          onQuickDelete={handleDelete}
          onEntriesLoaded={updateEntries}
        />
      {:else}
        <GridView
          prefix={currentPrefix}
          {location}
          {activeTag}
          {selectedEntry}
          {iconSize}
          invalidate={columnInvalidators[scopedPrefixKey(currentPrefix)] ?? 0}
          onSelect={(entry) => selectEntry(entry, columns.length - 1)}
          onOpen={openEntry}
          onContextMenu={showCtxMenu}
          onQuickSend={(entry) => { selectedEntry = entry; showSend = true; }}
          onQuickDownload={handleDownload}
          onQuickDelete={handleDelete}
          onMove={() => invalidateColumn(currentPrefix)}
          onEntriesLoaded={updateEntries}
        />
      {/if}
    </div>

    <PreviewPane
      entry={selectedEntry}
      onView={() => {
        if (selectedEntry?.file) previewFile = selectedEntry.file;
      }}
      onSend={() => showSend = true}
      onDownload={() => handleDownload()}
      onDelete={() => selectedEntry && handleDelete(selectedEntry)}
      onTagsUpdated={(f) => {
        if (selectedEntry) selectedEntry = { ...selectedEntry, file: f };
      }}
    />
  </div>

  <div class="finder-statusbar">
    <span>{(visibleEntries[scopedPrefixKey(currentPrefix)] ?? []).length} items</span>
    <span>{locationLabel}: {selectedEntry ? selectedEntry.name : "No selection"}</span>
  </div>
</div>

{#if ctxMenu}
  <ContextMenu
    x={ctxMenu.x}
    y={ctxMenu.y}
    entry={ctxMenu.entry}
    onClose={() => ctxMenu = null}
    onView={() => {
      if (ctxMenu?.entry?.file) previewFile = ctxMenu.entry.file;
    }}
    onSend={() => {
      if (location.kind !== "local") return;
      if (ctxMenu?.entry) selectedEntry = ctxMenu.entry;
      showSend = true;
    }}
    onDelete={() => {
      if (ctxMenu?.entry) handleDelete(ctxMenu.entry);
    }}
    onDownload={() => handleDownload(ctxMenu?.entry ?? null)}
    onRename={() => {
      if (location.kind !== "local") return;
      if (ctxMenu?.entry?.file) renameFile = ctxMenu.entry.file;
    }}
    onCopyPath={() => handleCopyPath(ctxMenu?.entry ?? null)}
    onDiff={() => {
      if (location.kind !== "local") return;
      if (ctxMenu?.entry) selectedEntry = ctxMenu.entry;
      if (ctxMenu?.entry?.file) previewFile = ctxMenu.entry.file;
    }}
    onStatus={() => {
      if (location.kind !== "local") return;
      if (ctxMenu?.entry) selectedEntry = ctxMenu.entry;
      if (ctxMenu?.entry?.file) previewFile = ctxMenu.entry.file;
    }}
    onUploadHere={() => {
      if (location.kind !== "local") return;
      showUpload = true;
    }}
  />
{/if}

{#if renameFile}
  <RenameFileModal
    file={renameFile}
    busy={renaming}
    on:close={() => renameFile = null}
    on:submit={(e) => submitRename(e.detail.path)}
  />
{/if}

{#if showSend}
  <SendModal
    file={selectedEntry?.file ?? null}
    dir={selectedEntry?.is_dir ? selectedEntry : null}
    on:close={() => showSend = false}
  />
{/if}

{#if showUpload && location.kind === "local"}
  <UploadModal
    on:close={() => showUpload = false}
    on:uploaded={() => {
      invalidateColumn(currentPrefix);
      showUpload = false;
    }}
  />
{/if}

{#if previewFile}
  <PreviewModal
    file={previewFile}
    {location}
    on:close={() => previewFile = null}
    on:deleted={() => {
      previewFile = null;
      if (selectedEntry) handleDelete(selectedEntry);
    }}
    on:tagsUpdated={(e) => {
      if (selectedEntry) selectedEntry = { ...selectedEntry, file: e.detail };
      previewFile = e.detail;
    }}
  />
{/if}

<style>
  .finder-root {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: var(--f-bg0);
    border: 0.5px solid var(--f-border);
    border-radius: 10px;
    overflow: hidden;
  }
  
  .tl-dots {
    display: flex;
    gap: 6px;
  }
  .tl-dots span {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    background: var(--f-border2);
  }
  
 
  .finder-body {
    flex: 1;
    min-height: 0;
    display: flex;
    overflow: hidden;
  }
  .finder-content {
    flex: 1;
    min-width: 0;
    min-height: 0;
    overflow: hidden;
    display: flex;
    background: var(--f-surface);
  }
  .finder-columns {
    flex: 1;
    display: flex;
    overflow-x: auto;
    overflow-y: hidden;
  }
  .finder-statusbar {
    height: 21px;
    border-top: 0.5px solid var(--f-border);
    background: var(--f-bg1);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 10px;
    font-family: var(--mono);
    font-size: 10px;
    color: var(--f-text3);
  }
</style>
