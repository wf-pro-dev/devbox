const TAG_COLORS = [
  "#2B5CE6",
  "#1A7A44",
  "#935B10",
  "#993556",
  "#0F6E56",
  "#6B3FA0",
  "#B91C1C",
];

export function fileTint(lang: string): { fill: string; stroke: string } {
  const value = (lang || "text").toLowerCase();
  if (["go", "typescript", "javascript", "python", "bash", "sh"].includes(value)) {
    return { fill: "#F0FAF0", stroke: "#8DC88D" };
  }
  if (["nginx", "yaml", "yml", "toml", "ini", "dockerfile", "systemd"].includes(value)) {
    return { fill: "#EEF4FF", stroke: "#7DA8E8" };
  }
  if (["json", "sql", "csv"].includes(value)) {
    return { fill: "#FFFBF0", stroke: "#DEB86A" };
  }
  return { fill: "#F3F1EC", stroke: "#B4B2A9" };
}

export function tagColor(name: string): string {
  let h = 0;
  for (const c of name) h = (h * 31 + c.charCodeAt(0)) & 0xffffff;
  return TAG_COLORS[Math.abs(h) % TAG_COLORS.length];
}
