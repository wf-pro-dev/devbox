# Devbox Finder UI — Design Tokens & Icon Spec

---

## CSS custom properties

Add to `web/src/app.css` under the existing `:root` block.
The existing `--bg`, `--text`, `--radius` etc. remain untouched for the
Files tab. The Finder view uses its own `--f-*` namespace so both UIs
can coexist during the migration.

```css
:root {
  /* ── Finder palette (warm parchment) ─────────────────────── */
  --f-bg0:            #F7F6F3;   /* page background              */
  --f-bg1:            #EFEDE8;   /* chrome: toolbar, sidebar, statusbar */
  --f-bg2:            #E5E3DC;   /* hover background             */
  --f-surface:        #FAFAF8;   /* column / list / grid content */
  --f-surface2:       #F3F1EC;   /* preview pane background      */

  --f-border:         rgba(0,0,0,0.09);   /* default border      */
  --f-border2:        rgba(0,0,0,0.15);   /* emphasis border     */

  --f-text:           #1A1916;
  --f-text2:          #5C5A54;
  --f-text3:          #9A9790;   /* muted / labels               */

  --f-accent:         #2B5CE6;   /* selection, primary actions   */
  --f-accent-bg:      rgba(43,92,230,0.09);
  --f-accent-border:  rgba(43,92,230,0.22);
  --f-selection:      rgba(43,92,230,0.10); /* selected row bg   */

  --f-folder:         #E8922A;   /* folder icon orange           */

  /* Search input — warm near-white, NOT the dark pill */
  --f-search-bg:      #FFFEFA;
  --f-search-border:  rgba(0,0,0,0.14);

  /* Status semantic */
  --f-ok:             #1A7A44;
  --f-ok-bg:          #E8F5EE;
  --f-ok-border:      #9FD3B3;
  --f-warn:           #935B10;
  --f-warn-bg:        #FFF5E0;
  --f-warn-border:    #F4C97A;
  --f-danger:         #B91C1C;
  --f-danger-bg:      #FEF2F2;
  --f-danger-border:  #FECACA;
}
```

---

## Typography

All Finder components use `DM Mono` for paths, filenames, sizes, versions,
and tag labels. `DM Sans` for everything else. Both are already loaded via
the Google Fonts import in `App.svelte`.

| Role | Font | Size | Weight |
|---|---|---|---|
| Window title | DM Sans | 12.5px | 500 |
| Breadcrumb active segment | DM Sans | 11px | 500 |
| Breadcrumb inactive segments | DM Sans | 11px | 400 |
| Column item name | DM Mono | 11px | 400 |
| Column item size / count | DM Mono | 10px | 400 |
| Preview pane name | DM Sans | 12.5px | 500 |
| Preview pane path | DM Mono | 10px | 400 |
| Preview meta key | DM Sans | 10px | 400 |
| Preview meta value | DM Mono | 10px | 400 |
| Section heading (sidebar) | DM Sans | 9.5px | 600 UC |
| Sidebar item | DM Sans | 11.5px | 400 |
| Status bar | DM Mono | 10px | 400 |
| Tab strip label | DM Sans | 9.5px | 400 |
| Action button | DM Sans | 10.5px | 400 |
| Tag pill | DM Mono | 10px | 400 |
| Grid cell name | DM Mono | 10px | 400 |
| List table header | DM Sans | 9.5px | 600 UC |
| List table cell name | DM Mono | 11px | 400 |
| List table cell meta | DM Mono | 10.5px | 400 |

---

## Dimensions

| Element | Value |
|---|---|
| Titlebar height | 38px |
| Toolbar height | 36px |
| Statusbar height | 21px |
| Sidebar width | 152px |
| Preview pane width | 210px |
| Column width | 192px |
| Column row height | ~26px (4px top + 4px bottom padding) |
| Grid cell min width | 84px (adjustable 60–120px via slider) |
| Grid cell gap | 5px |
| Context menu width | 160px |
| Border radius (window) | 10px |
| Border radius (cells, items) | 5–6px |
| Border radius (search input) | 12px (pill) |
| Border thickness | 0.5px everywhere |

---

## Icon system

### Tabler Icons (outline only)

Used for all UI controls. Already loaded in the project.
Never use `-filled` variants — they are not loaded.

| Action | Icon class |
|---|---|
| Upload | `ti-upload` |
| Send / AirDrop | `ti-send` |
| Download | `ti-download` |
| Delete | `ti-trash` |
| View / Quick look | `ti-eye` |
| Fleet status | `ti-radar` |
| Diff | `ti-git-compare` |
| Version history | `ti-history` |
| Info / Get Info | `ti-info-circle` |
| Tags | `ti-tag` |
| Copy path | `ti-copy` |
| Column view | `ti-layout-columns` |
| List view | `ti-list` |
| Grid view | `ti-grid-dots` |
| Toggle sidebar | `ti-layout-sidebar` |
| Back | `ti-chevron-left` |
| Forward | `ti-chevron-right` |
| Breadcrumb separator | `ti-chevron-right` (9px) |
| Sort selector | `ti-selector` |
| Icon size decrease | `ti-photo-minus` |
| Icon size increase | `ti-photo-plus` |
| New directory | `ti-folder-plus` |
| Sort ascending | `ti-sort-ascending` |
| Search | `ti-search` |

---

## File icon SVG specification (`icons.ts`)

All icons are inline SVG strings. No external image files.
The document shape is consistent across all file types; only the
fill color and interior detail lines vary.

### Base document shape (28×34 viewBox)

```svg
<svg width="{size}" height="{size*34/28}" viewBox="0 0 28 34" fill="none">
  <!-- Page body + corner fold -->
  <path d="M3 1h14l8 8v24a1 1 0 01-1 1H4a1 1 0 01-1-1V2a1 1 0 011-1z"
        fill="{tint.fill}" stroke="{tint.stroke}" stroke-width="0.8"/>
  <!-- Folded corner -->
  <path d="M17 1v8h8" stroke="{tint.stroke}" stroke-width="0.8" fill="none"/>
  <!-- Content lines (2–3 depending on type) -->
  <line x1="7" y1="16" x2="21" y2="16" stroke="{tint.stroke}" stroke-width="0.8" opacity="0.6"/>
  <line x1="7" y1="20" x2="18" y2="20" stroke="{tint.stroke}" stroke-width="0.8" opacity="0.4"/>
  <line x1="7" y1="24" x2="15" y2="24" stroke="{tint.stroke}" stroke-width="0.8" opacity="0.25"/>
</svg>
```

### Script file variant — replace content lines with a code indicator

```svg
<path d="M9 16l3 4-3 4" stroke="{tint.stroke}" stroke-width="1"
      stroke-linecap="round" fill="none"/>
<line x1="14" y1="24" x2="20" y2="24" stroke="{tint.stroke}"
      stroke-width="0.8" opacity="0.5"/>
```

### Data/YAML/JSON variant — replace content lines with small rect blocks

```svg
<rect x="7" y="16" width="5" height="3" rx="0.5" fill="{tint.stroke}" opacity="0.4"/>
<rect x="14" y="16" width="7" height="3" rx="0.5" fill="{tint.stroke}" opacity="0.3"/>
<rect x="7" y="21" width="8" height="3" rx="0.5" fill="{tint.stroke}" opacity="0.3"/>
```

### Tint map

| Language(s) | Fill | Stroke |
|---|---|---|
| `go`, `typescript`, `javascript`, `python`, `bash`, `sh` | `#F0FAF0` | `#8DC88D` |
| `nginx`, `yaml`, `toml`, `ini`, `dockerfile`, `systemd` | `#EEF4FF` | `#7DA8E8` |
| `json`, `sql`, `csv`, `toml` | `#FFFBF0` | `#DEB86A` |
| `markdown`, `text`, everything else | `#F3F1EC` | `#B4B2A9` |
| **Selected state** (any lang) | `rgba(43,92,230,0.08)` | `#2B5CE6` |

---

## Folder icon SVG specification

### Base folder shape (36×30 viewBox)

```svg
<svg width="{size}" height="{size*30/36}" viewBox="0 0 36 30" fill="none">
  <!-- Folder body -->
  <rect x="1" y="6" width="34" height="22" rx="3"
        fill="{fill}" stroke="{stroke}" stroke-width="{sw}"/>
  <!-- Tab (top-left) -->
  <rect x="1" y="6" width="11" height="5.5" rx="2"
        fill="{tabFill}" stroke="{stroke}" stroke-width="{sw}"/>
  <!-- Shelf line -->
  <path d="M1 11h34" stroke="{stroke}" stroke-width="0.5" opacity="0.5"/>
</svg>
```

### Folder states

| State | Fill | Tab fill | Stroke | SW |
|---|---|---|---|---|
| Default closed | `#EFEDE8` | `#EFEDE8` | `#9A9790` | 0.8 |
| Hovered | `#E5E3DC` | `#E5E3DC` | `#888780` | 0.8 |
| Selected (column) | `rgba(43,92,230,0.09)` | `rgba(43,92,230,0.15)` | `#2B5CE6` | 1 |
| Open (has children in next column) | `#E8922A` at 0.18 opacity | `#E8922A` at 0.28 opacity | `#E8922A` | 1 |
| Drop target (DnD) | `rgba(43,92,230,0.12)` | — | `#2B5CE6` dashed | 1.2 |
| Ghost (being dragged) | `#EFEDE8` at 0.35 opacity | — | `#9A9790` dashed | 0.8 |

For the column view (13×11 mini icon), scale the viewBox to `0 0 13 11` and
reduce stroke-width to `0.7`. The shelf line is omitted at small sizes.

---

## Tag color palette

Tags are displayed as colored dot + name in the sidebar, and as colored pills
inline. The color is derived from a simple hash of the tag name so the same
tag always gets the same color across sessions.

```ts
const TAG_COLORS = [
  '#2B5CE6', // blue
  '#1A7A44', // green
  '#935B10', // amber
  '#993556', // pink
  '#0F6E56', // teal
  '#6B3FA0', // purple
  '#B91C1C', // red
]

export function tagColor(name: string): string {
  let h = 0
  for (const c of name) h = (h * 31 + c.charCodeAt(0)) & 0xFFFFFF
  return TAG_COLORS[Math.abs(h) % TAG_COLORS.length]
}
```
