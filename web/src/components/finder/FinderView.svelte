<script lang="ts">
  import { onMount } from "svelte";
  import { api, listPeers, deleteDirectory } from "../../api";
  import type { DirEntry, File, HealthResponse, Peer } from "../../types";
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

  export let health: HealthResponse | null = null;
  export let onFileSelect: (f: File) => void = () => {};

  let columns: string[] = ["/"];
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

  function prefixToColumns(prefix: string) {
    if (prefix === "/") return ["/"];
    const segments = prefix.replace(/^\/|\/$/g, "").split("/").filter(Boolean);
    return ["/", ...segments.map((_, i) => segments.slice(0, i + 1).join("/"))];
  }

  function navigateToPrefix(
    nextPrefix: string,
    mode: "push" | "back" | "forward" | "replace" = "push",
    clearSelection = true,
  ) {
    if (nextPrefix === currentPrefix) return;

    if (mode === "push") {
      backHistory = [...backHistory, currentPrefix];
      forwardHistory = [];
    } else if (mode === "back") {
      forwardHistory = [currentPrefix, ...forwardHistory];
    } else if (mode === "forward") {
      backHistory = [...backHistory, currentPrefix];
    }

    columns = prefixToColumns(nextPrefix);
    if (clearSelection) selectedEntry = null;
  }

  function selectEntry(entry: DirEntry, colDepth: number) {
    selectedEntry = entry;
    columns = columns.slice(0, colDepth + 1);
    if (entry.file) onFileSelect(entry.file);
  }

  function openEntry(entry: DirEntry) {
    selectedEntry = entry;
    if (entry.is_dir && entry.prefix) {
      navigateToPrefix(entry.prefix, "push", false);
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
    columnInvalidators = { ...columnInvalidators, [prefix]: Date.now() };
  }

  function normalizePrefix(prefix: string) {
    if (!prefix || prefix === "/") return "/";
    return prefix.startsWith("/") ? prefix : `/${prefix}`;
  }

  function handleColumnMove(payload: {
    fileId: string;
    fromPath: string;
    toPath: string;
    sourcePrefix: string;
    targetPrefix: string;
  }) {
    const sourcePrefix = normalizePrefix(payload.sourcePrefix);
    const targetPrefix = normalizePrefix(payload.targetPrefix);

    invalidateColumn(sourcePrefix);
    invalidateColumn(targetPrefix);

    for (const openPrefix of columns) {
      const normalized = normalizePrefix(openPrefix);
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
    visibleEntries = { ...visibleEntries, [prefix]: entries };
  }

  function showCtxMenu(e: MouseEvent, entry: DirEntry | null) {
    e.preventDefault();
    ctxMenu = { x: e.clientX, y: e.clientY, entry };
  }

  function handleDelete(entry: DirEntry) {
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
        navigateToPrefix("/");
      }}
    />

    <div class="finder-content">
      {#if viewMode === "column"}
        <div class="finder-columns">
          {#each columns as prefix, i (prefix + (columnInvalidators[prefix] ?? 0))}
            <DirColumn
              {prefix}
              {activeTag}
              {selectedEntry}
              depth={i}
              invalidate={columnInvalidators[prefix] ?? 0}
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
          {activeTag}
          {selectedEntry}
          invalidate={columnInvalidators[currentPrefix] ?? 0}
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
          {activeTag}
          {selectedEntry}
          {iconSize}
          invalidate={columnInvalidators[currentPrefix] ?? 0}
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
    <span>{Object.values(visibleEntries[currentPrefix] ?? []).length} items</span>
    <span>{selectedEntry ? selectedEntry.name : "No selection"}</span>
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
      if (ctxMenu?.entry) selectedEntry = ctxMenu.entry;
      showSend = true;
    }}
    onDelete={() => {
      if (ctxMenu?.entry) handleDelete(ctxMenu.entry);
    }}
    onDownload={() => handleDownload(ctxMenu?.entry ?? null)}
    onRename={() => {
      if (ctxMenu?.entry?.file) renameFile = ctxMenu.entry.file;
    }}
    onCopyPath={() => handleCopyPath(ctxMenu?.entry ?? null)}
    onDiff={() => {
      if (ctxMenu?.entry) selectedEntry = ctxMenu.entry;
      focusPreview("diff");
    }}
    onStatus={() => {
      if (ctxMenu?.entry) selectedEntry = ctxMenu.entry;
      focusPreview("fleet");
    }}
    onUploadHere={() => showUpload = true}
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

{#if showUpload}
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
