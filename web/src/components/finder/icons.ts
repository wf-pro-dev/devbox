import { fileTint } from "./fileColor";

function escapeAttr(value: string): string {
  return value.replace(/"/g, "&quot;");
}

export function fileIcon(lang: string, size = 28): string {
  const tint = fileTint(lang);
  const lower = (lang || "").toLowerCase();
  let content = `
    <line x1="7" y1="16" x2="21" y2="16" stroke="${tint.stroke}" stroke-width="0.8" opacity="0.6"/>
    <line x1="7" y1="20" x2="18" y2="20" stroke="${tint.stroke}" stroke-width="0.8" opacity="0.4"/>
    <line x1="7" y1="24" x2="15" y2="24" stroke="${tint.stroke}" stroke-width="0.8" opacity="0.25"/>
  `;
  if (["go", "typescript", "javascript", "python", "bash", "sh"].includes(lower)) {
    content = `
      <path d="M9 16l3 4-3 4" stroke="${tint.stroke}" stroke-width="1" stroke-linecap="round" fill="none"/>
      <line x1="14" y1="24" x2="20" y2="24" stroke="${tint.stroke}" stroke-width="0.8" opacity="0.5"/>
    `;
  } else if (["yaml", "yml", "json", "sql", "csv", "toml", "ini"].includes(lower)) {
    content = `
      <rect x="7" y="16" width="5" height="3" rx="0.5" fill="${tint.stroke}" opacity="0.4"/>
      <rect x="14" y="16" width="7" height="3" rx="0.5" fill="${tint.stroke}" opacity="0.3"/>
      <rect x="7" y="21" width="8" height="3" rx="0.5" fill="${tint.stroke}" opacity="0.3"/>
    `;
  }
  return `<svg width="${size}" height="${(size * 34) / 28}" viewBox="0 0 28 34" fill="none" aria-hidden="true">
    <path d="M3 1h14l8 8v24a1 1 0 01-1 1H4a1 1 0 01-1-1V2a1 1 0 011-1z" fill="${escapeAttr(tint.fill)}" stroke="${escapeAttr(tint.stroke)}" stroke-width="0.8"/>
    <path d="M17 1v8h8" stroke="${escapeAttr(tint.stroke)}" stroke-width="0.8" fill="none"/>
    ${content}
  </svg>`;
}

export function folderIcon(
  state: "default" | "hover" | "selected" | "open" | "drop" | "ghost",
  mini = false,
): string {
  const width = mini ? 13 : 36;
  const height = mini ? 11 : 30;
  const viewBox = mini ? "0 0 13 11" : "0 0 36 30";
  const isDashed = state === "drop" || state === "ghost";
  const fillMap = {
    default: "#EFEDE8",
    hover: "#E5E3DC",
    selected: "rgba(43,92,230,0.09)",
    open: "rgba(232,146,42,0.18)",
    drop: "rgba(43,92,230,0.12)",
    ghost: "rgba(239,237,232,0.35)",
  };
  const tabMap = {
    default: "#EFEDE8",
    hover: "#E5E3DC",
    selected: "rgba(43,92,230,0.15)",
    open: "rgba(232,146,42,0.28)",
    drop: "rgba(43,92,230,0.12)",
    ghost: "rgba(239,237,232,0.35)",
  };
  const strokeMap = {
    default: "#9A9790",
    hover: "#888780",
    selected: "#2B5CE6",
    open: "#E8922A",
    drop: "#2B5CE6",
    ghost: "#9A9790",
  };
  const swMap = { default: 0.8, hover: 0.8, selected: 1, open: 1, drop: 1.2, ghost: 0.8 };
  if (mini) {
    return `<svg width="${width}" height="${height}" viewBox="${viewBox}" fill="none" aria-hidden="true">
      <path d="M1 4a1 1 0 011-1h3l1 1H12a1 1 0 011 1v5a1 1 0 01-1 1H2a1 1 0 01-1-1V4z" fill="${fillMap[state]}" stroke="${strokeMap[state]}" stroke-width="0.7" ${isDashed ? 'stroke-dasharray="2 1"' : ""}/>
      <path d="M1 5h12" stroke="${strokeMap[state]}" stroke-width="0.6" opacity="0.45"/>
    </svg>`;
  }
  return `<svg width="${width}" height="${height}" viewBox="${viewBox}" fill="none" aria-hidden="true">
    <rect x="1" y="6" width="34" height="22" rx="3" fill="${fillMap[state]}" stroke="${strokeMap[state]}" stroke-width="${swMap[state]}" ${isDashed ? 'stroke-dasharray="3 2"' : ""}/>
    <rect x="1" y="6" width="11" height="5.5" rx="2" fill="${tabMap[state]}" stroke="${strokeMap[state]}" stroke-width="${swMap[state]}" ${isDashed ? 'stroke-dasharray="3 2"' : ""}/>
    <path d="M1 11h34" stroke="${strokeMap[state]}" stroke-width="0.5" opacity="0.5"/>
  </svg>`;
}
