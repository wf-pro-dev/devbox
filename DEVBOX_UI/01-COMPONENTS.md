# Devbox Finder UI — Component Contracts

## 1. `icons.ts`

Pure TypeScript. No Svelte. Returns inline SVG strings.

```ts
// File icon — tinted by language/extension
export function fileIcon(lang: string, size = 28): string

// Folder icon — closed or open state
export function folderIcon(open: boolean, selected: boolean, size = 36): string
```

### File icon tint map (`fileColor.ts`)

```ts
export function fileTint(lang: string): { fill: string; stroke: string } {
  // generic  → { fill: '#F3F1EC', stroke: '#B4B2A9' }
  // config / nginx / yaml / toml → { fill: '#EEF4FF', stroke: '#7DA8E8' }
  // bash / sh / python / go / ts / js → { fill: '#F0FAF0', stroke: '#8DC88D' }
  // json / data / sql → { fill: '#FFFBF0', stroke: '#DEB86A' }
  // Selected state: replace fill with rgba(43,92,230,0.08), stroke with #2B5CE6
}
```

The icon SVG is a simple document shape: `path` for the page body + corner fold,
`path` for the folded corner triangle, and 2–3 short `line` elements for text
representation. No emoji. No external image references.

---

## 2. `FinderSidebar.svelte`

### Props
```ts
let { health, allTags, peers, activeTag, onSelectTag, onSelectRoot }
  = $props<{
    health: HealthResponse | null
    allTags: Array<{ name: string; count: number; color: string }>
    peers: Peer[]
    activeTag: string
    onSelectTag: (tag: string) => void
    onSelectRoot: () => void
  }>()
```

### Sections
1. **Locations** — "Devbox" (root), "Recent"
2. **Tags** — coloured dot + name + count badge. Click sets `activeTag`, re-filters columns.
3. **Machines** — green/grey dot, hostname, "you" badge for caller

### Width: 152px fixed, no resize handle in v1.

---

## 3. `FinderToolbar.svelte`

### Props
```ts
let { prefix, viewMode, activeTag, selectedEntry, onNavigate, onViewChange,
      onTagToggle, onUpload, onSend, onDelete, onStatus, onDiff }
  = $props<{ ... }>()
```

### Sections (left → right)
1. Back / Forward buttons (navigate `columns` history)
2. Separator
3. Breadcrumb — one `<button>` per segment; clicking navigates to that depth
4. Separator
5. View toggle — Column / List / Grid (3 icon buttons)
6. Separator
7. Active tag pills (removable `×`)
8. Separator (flex spacer)
9. **Context-sensitive action buttons** — appear only when `selectedEntry !== null`:
   - Send `ti-send` (files and dirs)
   - Fleet status `ti-radar` (files only)
   - Diff `ti-git-compare` (files only)
   - Delete `ti-trash` (always, danger color)
10. Upload `ti-upload` (always visible, right edge)

### Keyboard shortcuts (emit via `onXxx` callbacks)
- `⌫` → `onDelete`
- `⌘D` → `onDiff`
- `⎵` → quick look (open PreviewModal at full size)

---

## 4. `ContextMenu.svelte`

### Props
```ts
let { x, y, entry, onClose, onSend, onDiff, onStatus, onDownload,
      onCopyPath, onTags, onDelete, onUploadHere }
  = $props<{ ... }>()
```

### Behaviour
- Positioned absolutely at `{ x, y }` relative to `FinderView`'s container.
- Closes on: click outside, `Escape`, any action selected.
- `entry === null` → show canvas menu (Upload here, New directory, Sort by)
- `entry.is_dir === true` → show directory menu
- `entry.is_dir === false` → show file menu

### File menu items (in order)
```
Quick look        ⎵
Send to node…
Diff…             ⌘D
Check fleet status
────
Download          ⌘↓
Copy path
Tags…
────
Get Info          ⌘I
────
Delete            ⌫   (danger)
```

### Directory menu items
```
Send directory…
Download .tar.gz
Tags…
Copy path
────
Delete all files  ⌫   (danger)
```

---

## 5. `PreviewPane.svelte`

### Props
```ts
let { entry, onView, onSend, onDownload, onDelete, onTagsUpdated }
  = $props<{
    entry: DirEntry | null
    onView: () => void        // open full PreviewModal
    onSend: () => void
    onDownload: () => void
    onDelete: () => void
    onTagsUpdated: (f: File) => void
  }>()
```

### Layout (top to bottom)
1. **Icon area** — `folderIcon()` or `fileIcon()` at 40px
2. **Name** — 12.5px / 500
3. **Path** — 10px mono, `var(--f-text3)`, wraps
4. **Action row** — View, Download, Send (primary), Delete (danger)
5. **Tab strip** — 4 tabs for files, 1 for dirs:
   - `Info` — metadata grid (size, kind/lang, version, by, created, tags editable)
   - `⧖ History` — version list with rollback (files only)
   - `⬤ Fleet` — FleetStatusTab (files only), "Run check" button
   - `⟷ Diff` — DiffTab (files only)
6. **Tab body** — scrollable area below the strip

### For `entry === null` show an empty state:
```
(folder icon, muted)
Select a file or directory
to see its details
```

### Reuse existing tab content components:
- `FleetStatusTab.svelte` — unchanged, slot into Fleet tab
- `DiffTab.svelte` — unchanged, slot into Diff tab
- `VersionRow.svelte` — unchanged, used inside History tab

---

## 6. `DirColumn.svelte`

This is the core rendering primitive. One column = one prefix.

### Props
```ts
let { prefix, activeTag, selectedEntry, depth, onSelect }
  = $props<{
    prefix: string
    activeTag: string
    selectedEntry: DirEntry | null
    depth: number
    onSelect: (entry: DirEntry) => void
  }>()
```

### Behaviour
- On mount: `GET /dirs/{prefix}` (or `/dirs` for root)
- Renders `DirEntry[]` sorted: directories first (alpha), then files (alpha)
- Each row: icon (from `icons.ts`) + name (mono) + right-side meta (size / file count)
- Directories: show `ti-chevron-right` arrow
- Selected row: `var(--f-selection)` background, accent text color
- Right-click on any row: position and show `ContextMenu`
- Width: `192px` fixed. No horizontal scroll — names truncate with ellipsis.

### Row hover actions
- Appear on hover as absolutely-positioned icon buttons (right edge of row)
- File: Send `ti-send`, Delete `ti-trash`
- Dir: Send `ti-send`, Delete `ti-trash`
- These are the only inline affordances in column view — full action set is in the toolbar and context menu

---

## 7. `FinderView.svelte`

This is the orchestrator. Owns all state. Renders the three zones.

### State
```ts
let columns = $state<string[]>(['/'])      // prefix stack
let selectedEntry = $state<DirEntry | null>(null)
let viewMode = $state<'column' | 'list' | 'grid'>('column')
let activeTag = $state('')
let ctxMenu = $state<{ x: number; y: number; entry: DirEntry | null } | null>(null)
let showSendModal = $state(false)
let showUploadModal = $state(false)
```

### Column navigation algorithm
```ts
function selectEntry(entry: DirEntry, columnDepth: number) {
  selectedEntry = entry
  if (entry.is_dir) {
    // Slice off any columns deeper than this click, append new prefix
    columns = [...columns.slice(0, columnDepth + 1), entry.prefix!]
  }
  // File selection doesn't change columns
}

function navigateToBreadcrumb(segmentIndex: number) {
  // segmentIndex 0 = root '/', 1 = first segment, etc.
  columns = columns.slice(0, segmentIndex + 1)
  selectedEntry = null
}
```

### Layout (flexbox, full viewport height)
```
┌─ titlebar (38px, fixed) ──────────────────────────┐
├─ toolbar (36px, fixed) ───────────────────────────┤
├─ body (flex-1, overflow hidden) ──────────────────┤
│  ┌─sidebar─┐  ┌── columns / list / grid ──┐  ┌─pv─┐ │
│  │ 152px   │  │  flex-1, overflow-x auto  │  │210px│ │
│  └─────────┘  └───────────────────────────┘  └────┘ │
├─ statusbar (21px, fixed) ─────────────────────────┤
└───────────────────────────────────────────────────┘
```

### View mode switching
- Column: render `{#each columns as prefix, i}` → `<DirColumn>`
- List: render `<ListView prefix={columns.at(-1)} activeTag />`
- Grid: render `<GridView prefix={columns.at(-1)} activeTag />`
- Sidebar and PreviewPane are always visible regardless of view mode

---

## 8. `ListView.svelte`

### Props
```ts
let { prefix, activeTag, selectedEntry, onSelect }
  = $props<{ ... }>()
```

### Columns (sortable by clicking header)
```
[icon] Name | Kind | Size | Ver | Tags | By | Modified | [actions]
```

### Row hover actions (same as column view)
File: `ti-send`, `ti-git-compare`, `ti-radar`, `ti-trash`
Dir: `ti-send`, `ti-trash`

Reuses the same `GET /dirs/{prefix}` call as `DirColumn` with `?recursive=false`.

---

## 9. `GridView.svelte`

### Props
```ts
let { prefix, activeTag, selectedEntry, iconSize, onSelect }
  = $props<{
    prefix: string
    activeTag: string
    selectedEntry: DirEntry | null
    iconSize: number      // 60–120, default 84. Controlled by toolbar slider.
    onSelect: (entry: DirEntry) => void
  }>()
```

### Layout
```css
display: grid;
grid-template-columns: repeat(auto-fill, minmax(var(--icon-size), 1fr));
gap: 5px;
```

Two sections with a section label: "Folders" then "Files".

### Drag-and-drop (via `@thisux/sveltednd`)
```svelte
<div use:draggable={{ container: prefix, dragData: entry }}>...</div>
<div use:droppable={{ container: prefix, callbacks: { onDrop: handleDrop } }}>...</div>
```

`handleDrop` calls `api.moveFile(entry.file!.id, targetDir.prefix! + entry.name)`
which maps to `PATCH /files/{id}` with `{ path: newPath }`.

### DnD visual states
- **Dragging source**: `opacity: 0.35`, dashed outline (`--f-border2`)
- **Drop target (folder)**: `background: rgba(43,92,230,0.12)`, `outline: 1.5px dashed var(--f-accent)`
- **Drop target (invalid)**: no visual change (folders only accept files)

### Icon size slider
Lives in `FinderToolbar`. Emits `iconSize` as a number. `GridView` reads it as a prop
and sets `--icon-size: {iconSize}px` on the grid container.
