import { getDirectory, listDirectories } from "../../api";
import type { DirEntry } from "../../types";

export interface DirectoryStats {
  totalSize: number;
  latestUpdated: string;
  oldestEntry: DirEntry | null;
}

function older(a: DirEntry | null, b: DirEntry | null) {
  if (!a) return b;
  if (!b) return a;
  return new Date(a.file?.created_at ?? "").getTime() < new Date(b.file?.created_at ?? "").getTime() ? a : b;
}

function newer(a: string, b: string) {
  if (!a) return b;
  if (!b) return a;
  return new Date(a).getTime() >= new Date(b).getTime() ? a : b;
}

async function loadDirectoryEntries(prefix: string): Promise<DirEntry[]> {
  const listing = prefix === "/" ? await listDirectories() : await getDirectory(prefix);
  return listing.entries;
}

export async function computeDirectoryStats(prefix: string): Promise<DirectoryStats> {
  const entries = await loadDirectoryEntries(prefix);
  let totalSize = 0;
  let latestUpdated = "";
  let oldestEntry: DirEntry | null = null;

  for (const entry of entries) {
    if (entry.is_dir && entry.prefix) {
      const nested = await computeDirectoryStats(entry.prefix);
      totalSize += nested.totalSize;
      latestUpdated = newer(latestUpdated, nested.latestUpdated);
      oldestEntry = older(oldestEntry, nested.oldestEntry);
    } else if (entry.file) {
      totalSize += entry.file.size ?? 0;
      latestUpdated = newer(latestUpdated, entry.file.updated_at || entry.file.created_at || "");
      oldestEntry = older(oldestEntry, entry);
    }
  }

  return { totalSize, latestUpdated, oldestEntry };
}
