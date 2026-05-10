# Devbox Finder UI — Action Surface

Every action from the current UI is preserved. This file defines exactly which
API call each action fires, from which surface, and what happens to local state
after the call resolves.

---

## Action inventory

### Upload (push files)

| | |
|---|---|
| **Trigger** | Toolbar `ti-upload` button; right-click canvas → "Upload files here…" |
| **Opens** | Existing `UploadModal.svelte` (unchanged) |
| **API** | `POST /files` multipart |
| **On success** | Re-fetch current column (`invalidateColumn(currentPrefix)`); select new entry |
| **Finder analogy** | Drag file into Finder window |

---

### Send to node

| | |
|---|---|
| **Trigger (file)** | Toolbar Send button; context menu "Send to node…"; Preview pane Send button; List row hover icon |
| **Trigger (dir)** | Toolbar Send button; context menu "Send directory…"; Preview pane Send button |
| **Opens** | Existing `SendModal.svelte` (unchanged) |
| **API file** | `POST /files/{id}/send` |
| **API dir** | `POST /dirs/{dir}/send` |
| **On success** | Toast; modal closes |
| **Finder analogy** | AirDrop |

---

### Delete

| | |
|---|---|
| **Trigger (file)** | Toolbar Delete button; context menu "Delete"; Preview pane Delete button; List/Grid row hover; `⌫` key |
| **Trigger (dir)** | Same surfaces |
| **Confirmation** | `svelte-sonner` toast with Confirm action (same pattern as current code) |
| **API file** | `DELETE /files/{id}` |
| **API dir** | `DELETE /dirs/{dir}` |
| **On success** | Remove entry from `dirEntries[prefix]` in `FinderView` state; clear `selectedEntry` if deleted; async blob cleanup happens server-side |

---

### Download

| | |
|---|---|
| **Trigger (file)** | Context menu "Download"; Preview pane Download button; `⌘↓` shortcut |
| **Trigger (dir)** | Context menu "Download .tar.gz"; Preview pane Download button |
| **Mechanism** | Anchor `href=/files/{id}` with `download` attr (file); `href=/dirs/{dir}?content=true` (dir) |
| **API** | `GET /files/{id}` streaming; `GET /dirs/{dir}?content=true` → tar.gz |
| **No modal** | Browser native download; no UI feedback needed |

---

### Quick look / View content

| | |
|---|---|
| **Trigger** | Click file in any view; `⎵` key on selected file; Preview pane "View" button |
| **Opens** | Existing `PreviewModal.svelte` (unchanged) |
| **Inline preview** | Preview pane shows file icon + metadata; full content only in the modal |

---

### Fleet status

| | |
|---|---|
| **Trigger** | Toolbar `ti-radar` button (file selected); context menu "Check fleet status"; Preview pane Fleet tab |
| **Surface** | Preview pane → Fleet tab (reuses `FleetStatusTab.svelte`) |
| **API** | `GET /files/{id}/status` |
| **"View diff →" link** | Switches Preview pane to Diff tab with that node pre-selected (existing cross-tab nav) |

---

### Diff

| | |
|---|---|
| **Trigger** | Toolbar `ti-git-compare` button; context menu "Diff…"; Preview pane Diff tab; `⌘D` shortcut |
| **Surface** | Preview pane → Diff tab (reuses `DiffTab.svelte`) |
| **API node** | `GET /files/{id}/diff/node?node=hostname` |
| **API local** | `POST /files/{id}/diff/local` multipart |

---

### Version history / rollback

| | |
|---|---|
| **Trigger** | Preview pane → History tab |
| **Surface** | Preview pane → History tab (reuses `VersionRow.svelte`) |
| **API list** | `GET /files/{id}/versions` |
| **API rollback** | `POST /files/{id}/versions/{n}/rollback` |
| **On rollback** | Re-fetch file meta; update `selectedEntry`; invalidate current column |

---

### Edit metadata (description, language, path)

| | |
|---|---|
| **Trigger** | Preview pane → Info tab |
| **Surface** | Preview pane → Info tab inline editing (no separate modal) |
| **API** | `PATCH /files/{id}` with `{ description?, language?, path? }` |
| **On success** | Update `selectedEntry.file` in place |

---

### Tags (add / remove)

| | |
|---|---|
| **Trigger (file)** | Preview pane Info tab tag row; context menu "Tags…" |
| **Trigger (dir)** | Context menu "Tags…"; Preview pane Info area |
| **API add file** | `POST /files/{id}/tags` |
| **API remove file** | `DELETE /files/{id}/tags/{tag}` |
| **API add dir** | `POST /dirs/{dir}/tags` |
| **API remove dir** | `DELETE /dirs/{dir}/tags/{tag}` |
| **On success** | Update tags in `selectedEntry` and invalidate sidebar tag counts |

---

### Copy path

| | |
|---|---|
| **Trigger** | Context menu "Copy path" |
| **Mechanism** | `navigator.clipboard.writeText(entry.prefix ?? entry.file!.path)` |
| **On success** | Toast "Path copied" |

---

### Move file (drag into folder)

| | |
|---|---|
| **Trigger** | Grid view: drag file cell onto folder cell |
| **API** | `PATCH /files/{id}` with `{ path: newPath }` where `newPath = targetPrefix + fileName` |
| **On success** | Remove entry from current column; add to target dir's entry cache if loaded; toast |
| **Note** | This is the only new API call not in the current UI. `api.ts` already has `editMeta` which covers it. No backend change needed. |

---

### Filter by tag

| | |
|---|---|
| **Trigger** | Sidebar tag dot click; tag pill in toolbar |
| **Effect** | Sets `activeTag` in `FinderView`; each `DirColumn` / `ListView` / `GridView` passes `?tag=` to its `GET /dirs/{prefix}` call |
| **Clear** | Click active tag again, or click `×` on toolbar pill |

---

### Sort

| | |
|---|---|
| **Trigger** | List view: column header click; Grid view: "Name ▾" sort button in toolbar |
| **State** | `sortField` + `sortDir` local to `ListView` / `GridView` |
| **Mechanism** | Client-side sort of the fetched `DirEntry[]`; no API change |
| **Fields** | name, size, modified, version (files only) |

---

## Keyboard shortcuts

| Shortcut | Action |
|---|---|
| `⎵` | Quick look (open PreviewModal for selected file) |
| `⌫` | Delete selected entry (with confirmation toast) |
| `⌘D` | Open Diff tab in Preview pane |
| `⌘I` | Focus Preview pane Info tab |
| `⌘↓` | Download selected file |
| `⌘↑` | Navigate to parent directory (pop last column) |
| `←` `→` | Move selection between columns (column view) |
| `↑` `↓` | Move selection within a column |
| `Esc` | Close context menu; deselect |

All shortcuts are registered on `FinderView`'s container `keydown` handler.
Only fire when no input/textarea is focused.
