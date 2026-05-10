# Devbox Finder UI — Step-by-Step Implementation Guide

This file is for an AI coding agent executing the migration.
Follow each step in order. Do not skip ahead. Each step ends with a
verifiable checkpoint before proceeding to the next.

---

## Step 0 — Install dependency

```bash
cd web
npm install @thisux/sveltednd
```

**Checkpoint:** `package.json` contains `"@thisux/sveltednd"` in dependencies.

---

## Step 1 — Add design tokens

**File:** `web/src/app.css`

Append the `--f-*` custom properties block from `03-TOKENS-AND-ICONS.md`
inside the existing `:root { }` rule. Do not remove any existing variables.

**Checkpoint:** No visual change to the app. CSS variables resolve in browser
DevTools under `:root`.

---

## Step 2 — Create `icons.ts` and `fileColor.ts`

**Files to create:**
- `web/src/components/finder/icons.ts`
- `web/src/components/finder/fileColor.ts`

Implement the functions as specified in `03-TOKENS-AND-ICONS.md`:

`fileColor.ts` exports:
```ts
export function fileTint(lang: string): { fill: string; stroke: string }
export function tagColor(name: string): string
```

`icons.ts` exports:
```ts
export function fileIcon(lang: string, size?: number): string
export function folderIcon(state: 'default'|'hover'|'selected'|'open'|'drop'|'ghost', mini?: boolean): string
```

Both files are pure TypeScript — no Svelte, no imports from the project except
each other.

**Checkpoint:** Import both in a browser console snippet and call
`fileIcon('bash')` — should return a non-empty SVG string.

---

## Step 3 — Create `ContextMenu.svelte`

**File:** `web/src/components/finder/ContextMenu.svelte`

Spec: `01-COMPONENTS.md` § 4.

Key implementation notes:
- Use `position: fixed` on the outer `<div>` — `FinderView` passes `{ x, y }`
  from `MouseEvent.clientX/Y`.
- Register a `window` click listener on mount that calls `onClose()`.
- Register `Escape` keydown listener that calls `onClose()`.
- Both listeners must be cleaned up in `onDestroy`.
- Render conditionally based on `entry` type (null / dir / file) as specified.

```svelte
<script lang="ts">
  import { onDestroy } from 'svelte'
  let { x, y, entry, onClose, ...actions } = $props()

  function handleWindowClick() { onClose() }
  function handleKey(e: KeyboardEvent) { if (e.key === 'Escape') onClose() }
  window.addEventListener('click', handleWindowClick)
  window.addEventListener('keydown', handleKey)
  onDestroy(() => {
    window.removeEventListener('click', handleWindowClick)
    window.removeEventListener('keydown', handleKey)
  })
</script>

<div style="position:fixed;left:{x}px;top:{y}px;z-index:200"
     on:click|stopPropagation>
  <!-- menu items -->
</div>
```

**Checkpoint:** Import into `App.svelte` temporarily, render with dummy props,
confirm it appears at the correct position and closes on click-outside.

---

## Step 4 — Create `FinderSidebar.svelte`

**File:** `web/src/components/finder/FinderSidebar.svelte`

Spec: `01-COMPONENTS.md` § 2.

Uses no API calls — all data passed as props from `FinderView`.

Tag dots use `tagColor(name)` from `fileColor.ts`.

Machine status: green dot if `peer.status.Online`, grey otherwise.
The "you" entry uses `health?.caller_host` with a blue "you" badge.

**Checkpoint:** Render standalone with dummy `allTags` and `peers` props.
Confirm tag dots appear in correct colors, machine status renders.

---

## Step 5 — Create `FinderToolbar.svelte`

**File:** `web/src/components/finder/FinderToolbar.svelte`

Spec: `01-COMPONENTS.md` § 3.

Key points:
- Breadcrumb segments are derived from `pathutil.Segments(prefix)` equivalent
  in JS: `prefix.replace(/^\/|\/$/g, '').split('/')`. Add "root" as segment 0.
- Context-sensitive action buttons (`ti-send`, `ti-radar`, `ti-git-compare`,
  `ti-trash`) use `{#if selectedEntry}` — hidden when nothing is selected.
- `ti-radar` and `ti-git-compare` use `{#if selectedEntry && !selectedEntry.is_dir}` — hidden for directories.
- The icon size slider renders only in grid view: `{#if viewMode === 'grid'}`.
- The sort selector renders in list and grid views: `{#if viewMode !== 'column'}`.

**Checkpoint:** Render with `selectedEntry = null` and confirm action buttons
are hidden. Set `selectedEntry` to a file mock and confirm they appear.

---

## Step 6 — Create `PreviewPane.svelte`

**File:** `web/src/components/finder/PreviewPane.svelte`

Spec: `01-COMPONENTS.md` § 5.

Reuses (import, do not copy):
- `FleetStatusTab.svelte` (existing)
- `DiffTab.svelte` (existing)
- `VersionRow.svelte` (existing)

Tab state is local: `let activeTab = $state<'info'|'history'|'fleet'|'diff'>('info')`

When `entry` changes, reset `activeTab` to `'info'`.

The History tab body calls `api.listVersions(entry.file!.id)` lazily on first
activation (same pattern as `PreviewModal`).

The Fleet tab body renders `<FleetStatusTab file={entry.file} />` — the
existing component already handles its own loading state.

The Diff tab renders `<DiffTab file={entry.file} />`.

Info tab for **files** renders an editable metadata grid:
- Description: click-to-edit inline input (same pattern as `PreviewModal`)
- Tags: tag pills with `×` remove button + add input
- Language: static display (editable via PATCH in future)
- All other fields: read-only

Info tab for **directories** renders:
- File count, total size (sum of `entry.fileCount`)
- Tags with add/remove

Action buttons map to props: `onView`, `onSend`, `onDownload`, `onDelete`.

**Checkpoint:** Render with a mock file entry. Tab strip appears with 4 tabs.
Switching tabs updates displayed content. Empty state renders when `entry = null`.

---

## Step 7 — Create `DirColumn.svelte`

**File:** `web/src/components/finder/DirColumn.svelte`

Spec: `01-COMPONENTS.md` § 6.

```svelte
<script lang="ts">
  import { getDirectory, listDirectories } from '../../api'
  import { fileIcon, folderIcon } from './icons'
  import type { DirEntry } from '../../types'

  let { prefix, activeTag, selectedEntry, depth, onSelect, onContextMenu }
    = $props<{
      prefix: string
      activeTag: string
      selectedEntry: DirEntry | null
      depth: number
      onSelect: (entry: DirEntry) => void
      onContextMenu: (e: MouseEvent, entry: DirEntry) => void
    }>()

  let entries = $state<DirEntry[]>([])
  let loading = $state(true)
  let error = $state('')

  $effect(() => {
    loading = true
    const call = prefix === '/'
      ? listDirectories()
      : getDirectory(prefix)
    call
      .then(d => { entries = d.entries; loading = false })
      .catch(e => { error = e.message; loading = false })
  })
</script>
```

Sort entries: directories first (alpha), then files (alpha).
This sort is client-side on the fetched array.

Right-click on any row calls `onContextMenu(event, entry)`.
`event.preventDefault()` to suppress browser default menu.

Hover shows inline action icons (Send, Delete) on the right side of the row
using `position: absolute` within a `position: relative` row container.

**Checkpoint:** Mount with `prefix="/"`. Should show root-level directories.
Click a directory row: `onSelect` fires with the directory entry. Right-click:
`onContextMenu` fires.

---

## Step 8 — Create `ListView.svelte`

**File:** `web/src/components/finder/ListView.svelte`

Spec: `01-COMPONENTS.md` § 8.

Fetches the same data as `DirColumn` (`GET /dirs/{prefix}`).

Sortable columns: clicking a `<th>` toggles `sortField` / `sortDir` (local state).
Default sort: directories first, then files by name.

Row hover: inline action icons appear on the right using opacity transition
(same pattern as existing `FileRow.svelte`).

File rows show: folder/file icon, name (mono), kind/lang, size, version,
tags (colored pills), uploaded_by, modified date, action icons.

Directory rows show: folder icon, name + `/` suffix, "folder" kind, file count,
`—` for other fields, action icons (Send, Delete only).

**Checkpoint:** Mount with a real prefix. Rows render. Clicking a column header
sorts the list. Row hover reveals action icons.

---

## Step 9 — Create `GridView.svelte`

**File:** `web/src/components/finder/GridView.svelte`

Spec: `01-COMPONENTS.md` § 9.

Uses `@thisux/sveltednd`:

```svelte
<script lang="ts">
  import { draggable, droppable } from '@thisux/sveltednd'
  // ...
</script>

{#each fileEntries as entry}
  <div class="gc"
       use:draggable={{ container: prefix, dragData: entry }}
       on:contextmenu|preventDefault={e => onContextMenu(e, entry)}
       class:sel={selectedEntry?.file?.id === entry.file?.id}
       on:click={() => onSelect(entry)}>
    {@html fileIcon(entry.file!.language, iconSize)}
    <span class="gc-name">{entry.name}</span>
  </div>
{/each}

{#each dirEntries as entry}
  <div class="gc"
       use:droppable={{
         container: prefix,
         callbacks: { onDrop: (data) => handleDrop(data, entry) }
       }}
       on:contextmenu|preventDefault={e => onContextMenu(e, entry)}
       class:drop-target={isDragOver(entry)}
       class:sel={selectedEntry?.prefix === entry.prefix}
       on:click={() => onSelect(entry)}>
    {@html folderIcon('default')}
    <span class="gc-name">{entry.name}</span>
  </div>
{/each}
```

`handleDrop(dragData: DirEntry, targetDir: DirEntry)`:
```ts
async function handleDrop(dragData: DirEntry, targetDir: DirEntry) {
  if (!dragData.file || !targetDir.prefix) return
  const newPath = targetDir.prefix + dragData.name
  await api.editMeta(dragData.file.id, { path: newPath })
  // Invalidate and re-fetch current column
  invalidate()
}
```

Icon size is controlled by CSS variable:
```svelte
<div class="grid-root" style="--icon-size: {iconSize}px">
```

```css
.grid-root {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(var(--icon-size), 1fr));
  gap: 5px;
}
```

**Checkpoint:** Mount with a real prefix. Icons render. Click selects. Right-click
shows context menu. Drag a file onto a folder: `editMeta` API call fires.

---

## Step 10 — Create `FinderView.svelte`

**File:** `web/src/components/finder/FinderView.svelte`

This is the orchestrator. Assembles all previous components.

```svelte
<script lang="ts">
  import { onMount } from 'svelte'
  import FinderSidebar from './FinderSidebar.svelte'
  import FinderToolbar from './FinderToolbar.svelte'
  import DirColumn from './DirColumn.svelte'
  import ListView from './ListView.svelte'
  import GridView from './GridView.svelte'
  import PreviewPane from './PreviewPane.svelte'
  import ContextMenu from './ContextMenu.svelte'
  import SendModal from '../SendModal.svelte'
  import UploadModal from '../UploadModal.svelte'
  import { toast } from 'svelte-sonner'
  import { api, listPeers } from '../../api'
  import type { DirEntry, File, Peer, HealthResponse } from '../../types'

  let { health, onFileSelect }: {
    health: HealthResponse | null
    onFileSelect: (f: File) => void
  } = $props()

  // ── Core state ───────────────────────────────────────────────
  let columns = $state<string[]>(['/'])
  let selectedEntry = $state<DirEntry | null>(null)
  let viewMode = $state<'column' | 'list' | 'grid'>('column')
  let activeTag = $state('')
  let iconSize = $state(84)
  let ctxMenu = $state<{ x: number; y: number; entry: DirEntry | null } | null>(null)
  let showSend = $state(false)
  let showUpload = $state(false)
  let peers = $state<Peer[]>([])
  let columnInvalidators = $state<Record<string, number>>({}) // bump to re-fetch

  // ── Derived ──────────────────────────────────────────────────
  const currentPrefix = $derived(columns.at(-1) ?? '/')

  // ── Navigation ───────────────────────────────────────────────
  function selectEntry(entry: DirEntry, colDepth: number) {
    selectedEntry = entry
    if (entry.is_dir) {
      columns = [...columns.slice(0, colDepth + 1), entry.prefix!]
    } else {
      // File selected — don't extend columns
      columns = columns.slice(0, colDepth + 1)
    }
    if (entry.file) onFileSelect(entry.file)
  }

  function navigateTo(segIdx: number) {
    columns = columns.slice(0, segIdx + 1)
    selectedEntry = null
  }

  function navigateUp() {
    if (columns.length > 1) {
      columns = columns.slice(0, -1)
      selectedEntry = null
    }
  }

  function invalidateColumn(prefix: string) {
    columnInvalidators = { ...columnInvalidators, [prefix]: Date.now() }
  }

  // ── Context menu ─────────────────────────────────────────────
  function showCtxMenu(e: MouseEvent, entry: DirEntry | null) {
    e.preventDefault()
    ctxMenu = { x: e.clientX, y: e.clientY, entry }
  }

  // ── Actions ──────────────────────────────────────────────────
  async function handleDelete(entry: DirEntry) {
    const label = entry.is_dir ? `directory "${entry.name}"` : `"${entry.name}"`
    toast(`Delete ${label}?`, {
      action: {
        label: 'Confirm',
        onClick: async () => {
          try {
            if (entry.is_dir) {
              await api.deleteDirectory(entry.prefix!)   // uses request() wrapper
            } else {
              await api.deleteFile(entry.file!.id)
            }
            invalidateColumn(currentPrefix)
            if (selectedEntry === entry) selectedEntry = null
            toast.success(`Deleted ${label}`)
          } catch (e: unknown) {
            toast.error((e as Error).message)
          }
        }
      }
    })
  }

  // ── Keyboard ─────────────────────────────────────────────────
  function handleKeydown(e: KeyboardEvent) {
    const tag = (e.target as HTMLElement).tagName
    if (tag === 'INPUT' || tag === 'TEXTAREA') return
    if (e.key === 'Backspace' && selectedEntry) handleDelete(selectedEntry)
    if (e.key === ' ' && selectedEntry?.file) { e.preventDefault(); onFileSelect(selectedEntry.file) }
    if (e.key === 'ArrowUp') navigateUp()
    if (e.key === 'Escape') { ctxMenu = null; selectedEntry = null }
  }

  onMount(async () => {
    peers = await listPeers().catch(() => [])
  })
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="finder-root" on:contextmenu|self={e => showCtxMenu(e, null)}>
  <!-- Titlebar -->
  <div class="finder-titlebar">
    <div class="tl-dots">...</div>
    <div class="finder-search">
      <i class="ti ti-search"></i>
      <input placeholder="Search files, paths, tags…" />
    </div>
    <div class="finder-title-actions">
      <button on:click={() => showUpload = true}><i class="ti ti-upload"></i></button>
      <button><i class="ti ti-layout-sidebar"></i></button>
    </div>
  </div>

  <!-- Toolbar -->
  <FinderToolbar
    prefix={currentPrefix}
    {viewMode}
    {activeTag}
    {selectedEntry}
    {iconSize}
    onNavigate={navigateTo}
    onViewChange={v => viewMode = v}
    onTagToggle={t => activeTag = activeTag === t ? '' : t}
    onUpload={() => showUpload = true}
    onSend={() => showSend = true}
    onDelete={() => selectedEntry && handleDelete(selectedEntry)}
    onStatus={() => { /* focus preview fleet tab */ }}
    onDiff={() => { /* focus preview diff tab */ }}
    onIconSize={s => iconSize = s}
  />

  <!-- Body -->
  <div class="finder-body">
    <FinderSidebar
      {health}
      allTags={[]}
      {peers}
      {activeTag}
      onSelectTag={t => activeTag = activeTag === t ? '' : t}
      onSelectRoot={() => { columns = ['/'] ; selectedEntry = null }}
    />

    <!-- Column / List / Grid -->
    <div class="finder-content">
      {#if viewMode === 'column'}
        <div class="finder-columns">
          {#each columns as prefix, i (prefix + columnInvalidators[prefix])}
            <DirColumn
              {prefix}
              {activeTag}
              {selectedEntry}
              depth={i}
              onSelect={entry => selectEntry(entry, i)}
              onContextMenu={showCtxMenu}
            />
          {/each}
        </div>
      {:else if viewMode === 'list'}
        <ListView
          prefix={currentPrefix}
          {activeTag}
          {selectedEntry}
          onSelect={entry => selectEntry(entry, columns.length - 1)}
          onContextMenu={showCtxMenu}
          invalidate={columnInvalidators[currentPrefix]}
        />
      {:else}
        <GridView
          prefix={currentPrefix}
          {activeTag}
          {selectedEntry}
          {iconSize}
          onSelect={entry => selectEntry(entry, columns.length - 1)}
          onContextMenu={showCtxMenu}
          onMove={() => invalidateColumn(currentPrefix)}
          invalidate={columnInvalidators[currentPrefix]}
        />
      {/if}
    </div>

    <PreviewPane
      entry={selectedEntry}
      onView={() => selectedEntry?.file && onFileSelect(selectedEntry.file)}
      onSend={() => showSend = true}
      onDownload={() => {}}
      onDelete={() => selectedEntry && handleDelete(selectedEntry)}
      onTagsUpdated={f => { if (selectedEntry) selectedEntry = { ...selectedEntry, file: f } }}
    />
  </div>

  <!-- Status bar -->
  <div class="finder-statusbar">
    <!-- populated from current column's entry count + selection info -->
  </div>
</div>

{#if ctxMenu}
  <ContextMenu
    x={ctxMenu.x} y={ctxMenu.y} entry={ctxMenu.entry}
    onClose={() => ctxMenu = null}
    onSend={() => { showSend = true; ctxMenu = null }}
    onDelete={() => { ctxMenu?.entry && handleDelete(ctxMenu.entry); ctxMenu = null }}
    onDownload={() => ctxMenu = null}
    onCopyPath={() => {
      const p = ctxMenu?.entry?.prefix ?? ctxMenu?.entry?.file?.path ?? ''
      navigator.clipboard.writeText(p)
      toast.success('Path copied')
      ctxMenu = null
    }}
    onDiff={() => ctxMenu = null}
    onStatus={() => ctxMenu = null}
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
    on:uploaded={e => { invalidateColumn(currentPrefix); showUpload = false }}
  />
{/if}
```

**Checkpoint:** The full Finder UI renders. Navigating into directories appends
columns. Selecting a file shows the preview pane. Context menu appears on
right-click. All modals open.

---

## Step 11 — Wire into `App.svelte`

In `web/src/App.svelte`, replace the `directories` tab branch:

```svelte
<!-- Before -->
{#if activeMainTab === "directories"}
  <DirectoriesTab
    onFileSelect={(f) => { previewFile = f; }}
    onFileDelete={(f) => { onFileDeleted(f.id); }}
  />

<!-- After -->
{#if activeMainTab === "directories"}
  <FinderView
    {health}
    onFileSelect={(f) => { previewFile = f; }}
  />
```

Remove the import of `DirectoriesTab`. Add import of `FinderView`.

**Checkpoint:** Switch to the Directories tab. FinderView renders. Files tab
is unaffected. Existing `PreviewModal` still opens from file rows in Files tab.

---

## Step 12 — Delete old components

Only after step 11 passes smoke testing:

```bash
rm web/src/components/DirectoriesTab.svelte
rm web/src/components/DirNode.svelte
rm web/src/components/SubDirNode.svelte
rm web/src/components/DirFile.svelte
```

Run `npm run build` and confirm no TypeScript errors.

**Checkpoint:** Build succeeds. No orphaned imports.

---

## Notes for the agent

- `SendModal.svelte` accepts either `file` or `dir` prop — it already supports
  both. Pass `dir={selectedEntry}` when a directory is selected.
- `UploadModal.svelte` calls `listDirectories()` to populate the prefix combobox.
  This still works because `listDirectories()` calls `GET /dirs` which returns
  a `DirListing` — compatible with the new backend.
- `api.ts` already exports `deleteDirectory` via the `request()` wrapper.
  Verify the function name matches before step 10.
- The `health` prop passed to `FinderView` is the same `HealthResponse` already
  loaded in `App.svelte` — pass it through as a prop, don't re-fetch.
- Tag colors in the sidebar use `tagColor()` from `fileColor.ts`. The sidebar
  receives `allTags` as a derived value computed in `FinderView` from the union
  of all tags across visible entries.
