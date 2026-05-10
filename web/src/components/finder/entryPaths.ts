import type { DirEntry } from "../../types";

export function joinPath(prefix: string, fileName: string) {
  const clean = prefix.replace(/\/+$/g, "");
  return clean ? `${clean}/${fileName}` : '/'+fileName;
}

export function splitPath(path: string) {
  const clean = path.replace(/\/+$/g, "");
  const idx = clean.lastIndexOf("/");
  if (idx === -1) return { dir: "", name: clean };
  return { dir: clean.slice(0, idx), name: clean.slice(idx + 1) };
}

export function entryPath(entry: DirEntry) {
  return entry.is_dir ? entry.prefix ?? "" : entry.file?.path ?? "";
}

export function pathSegments(path: string) {
  return path.replace(/^\/|\/$/g, "").split("/").filter(Boolean);
}
