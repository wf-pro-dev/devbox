# Devbox Finder UI — Implementation Plan

## Overview

Replace the current tree-based `DirectoriesTab` with a three-zone Finder-style layout:
sidebar → scrollable column stack → preview pane. Three view modes: Column, List, Grid.
Every existing action (upload, send, delete, diff, status, versions, tags, download) is
preserved and mapped to a consistent placement across all three views.

---

## Files to create

| File | Purpose |
|---|---|
| `web/src/components/finder/FinderView.svelte` | Top-level orchestrator; owns all state |
| `web/src/components/finder/FinderSidebar.svelte` | Left panel — Locations, Tags, Machines |
| `web/src/components/finder/FinderToolbar.svelte` | Breadcrumb, view toggle, tag pills, upload |
| `web/src/components/finder/DirColumn.svelte` | One scrollable column for a single prefix |
| `web/src/components/finder/PreviewPane.svelte` | Right panel — icon, actions, tab strip |
| `web/src/components/finder/GridView.svelte` | Icon grid renderer (grid view mode) |
| `web/src/components/finder/ListView.svelte` | Flat table renderer (list view mode) |
| `web/src/components/finder/ContextMenu.svelte` | Right-click floating menu |
| `web/src/components/finder/icons.ts` | SVG icon factory functions (file / folder) |
| `web/src/components/finder/fileColor.ts` | Language → icon tint mapping |

---

## Files to update

| File | Change |
|---|---|
| `web/src/App.svelte` | Replace `<DirectoriesTab>` with `<FinderView>` in the `directories` tab branch |
| `web/src/api.ts` | Add `moveFile(id, newPath)` alias; already has all other needed calls |
| `web/src/types.ts` | No changes needed — `DirListing`, `DirEntry`, `File` types already correct |

---

## Files to delete

| File | Reason |
|---|---|
| `web/src/components/DirectoriesTab.svelte` | Replaced by `FinderView.svelte` |
| `web/src/components/DirNode.svelte` | Replaced by `DirColumn.svelte` + `GridView.svelte` |
| `web/src/components/SubDirNode.svelte` | Replaced by recursive `DirColumn.svelte` |
| `web/src/components/DirFile.svelte` | Replaced by entries in `DirColumn`, `GridView`, `ListView` |

---

## Dependencies to install

```bash
npm install @thisux/sveltednd
```

`@thisux/sveltednd` — Svelte 5 native (built on `$state` runes). Provides
`use:draggable` and `use:droppable` Svelte actions. Zero external deps.
Supports grid layouts and nested containers. Used only in `GridView.svelte`.

---

## Implementation order

Follow this sequence strictly. Each step is independently testable.

1. **`icons.ts` + `fileColor.ts`** — pure functions, no Svelte, testable in isolation
2. **`FinderSidebar.svelte`** — read-only panel, no state owned
3. **`FinderToolbar.svelte`** — emits events only, no data fetching
4. **`ContextMenu.svelte`** — stateless, positioned via props
5. **`PreviewPane.svelte`** — reads a `DirEntry | null` prop, contains the tab strip
6. **`DirColumn.svelte`** — fetches one prefix, renders entries, emits `select`
7. **`ListView.svelte`** — flat table, same data as `DirColumn`, different renderer
8. **`GridView.svelte`** — icon grid with `@thisux/sveltednd`, same data source
9. **`FinderView.svelte`** — assembles all pieces, owns `columns[]` and `selectedEntry`
10. **`App.svelte`** — swap `<DirectoriesTab>` for `<FinderView>`
11. **Delete** old components after smoke-testing step 10

See individual spec files for detailed contracts per component.

---

## Design tokens (add to `app.css` or a new `finder.css`)

```css
:root {
  /* Finder palette — warm parchment */
  --f-bg0: #F7F6F3;
  --f-bg1: #EFEDE8;   /* chrome: titlebar, toolbar, sidebar, statusbar */
  --f-bg2: #E5E3DC;   /* hover state */
  --f-surface: #FAFAF8; /* content area */
  --f-surface2: #F3F1EC; /* preview pane */
  --f-border: rgba(0,0,0,0.09);
  --f-border2: rgba(0,0,0,0.15);
  --f-text: #1A1916;
  --f-text2: #5C5A54;
  --f-text3: #9A9790;
  --f-accent: #2B5CE6;          /* selection, primary actions */
  --f-accent-bg: rgba(43,92,230,0.09);
  --f-accent-border: rgba(43,92,230,0.22);
  --f-selection: rgba(43,92,230,0.10); /* selected row/cell background */
  --f-folder: #E8922A;          /* folder icon fill */
  /* Search input — warm, not dark */
  --f-search-bg: #FFFEFA;
  --f-search-border: rgba(0,0,0,0.14);
}
```
