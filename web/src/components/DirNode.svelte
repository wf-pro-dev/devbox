<script lang="ts">
  import { formatBytes, langColor, deleteDirectory } from '../api';
  import type { Directory, File, TreeNode } from '../types';
  import SubDirNode from './SubDirNode.svelte';
  import DirFile from './DirFile.svelte';
  import { toast } from "svelte-sonner";


  export let node: TreeNode;
  export let expanded: Set<string>;
  export let dirFiles: Record<string, File[] | undefined>;
  export let onToggle: (dir: Directory) => void;
  export let onDelete: (prefix: string) => void;
  export let onDeliver: (dir: Directory) => void;
  export let onFileSelect: (f: File) => void;
  export let onFileDownload: (f: File, e: MouseEvent) => void;
  export let onFileDelete: (f: File) => void;
  export let depth: number;

  $: isExpanded = node.dir ? expanded.has(node.dir.prefix) : false;
  // Only show files directly in this dir, not in any subdirectory.
  // e.g. for prefix "dev/", file "dev/go/main.go" has a sub-segment after stripping
  // the prefix ("go/main.go") so it belongs to a child node, not here.
  function directFiles(allFiles: File[] | null | undefined, prefix: string): File[] | null {
    if (allFiles == null) return null;
    return allFiles.filter(f => {
      const relative = f.path.startsWith(prefix) ? f.path.slice(prefix.length) : f.file_name;
      return !relative.includes('/');
    });
  }
  $: files = node.dir ? directFiles(dirFiles[node.dir.prefix], node.dir.prefix) : null;
  $: hasChildren = node.children.length > 0;
  $: isExpandable = hasChildren || node.dir != null;

  const INDENT = 20;

  function handleToggle() {
    if (node.dir) onToggle(node.dir);
    else {
      virtualOpen = !virtualOpen;
    }
  }

  let virtualOpen = false;
  $: isOpen = node.dir ? isExpanded : virtualOpen;

  const handleDirDelete = (prefix: string, e: MouseEvent) => {
    e.stopPropagation();
    onDelete(prefix);
  }

  async function handleSubDelete(prefix: string) {
    toast(`Delete directory "${prefix}"?`, {
      action: {
        label: "Confirm",
        onClick: async () => {
          try {
            await deleteDirectory(prefix);
            if (node.children) {
              node.children = node.children.filter((child) => child.dir?.prefix !== prefix);
            }
            toast.success(`Deleted directory "${prefix}"`);
          } catch (e: unknown) {
            toast.error((e as Error).message);
            console.error(e);
          }
        },
      },
    });
  }

  const handleFileDelete = (file: File) => {
    if (files) {
      files = files.filter((f) => f.id !== file.id);
    }
    onFileDelete(file);
  }
</script>

<div class="node">
<!-- Row -->
<button
  class="node-row"
  class:open={isOpen}
  style="padding-left: {14 + depth * INDENT}px"
  on:click={handleToggle}
>
  <!-- Chevron -->
  <span class="chevron" class:rotated={isOpen} class:leaf={!isExpandable}>
    {#if isExpandable}
      <svg viewBox="0 0 10 10" fill="none" width="10" height="10">
        <path d="M3 2l4 3-4 3" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    {:else}
      <span class="leaf-dot"></span>
    {/if}
  </span>

  <!-- Folder icon -->
  <svg class="folder-icon" viewBox="0 0 16 16" fill="none" width="14" height="14">
    <path
      d="M2 5a1 1 0 011-1h3l1.5 1.5H13a1 1 0 011 1V12a1 1 0 01-1 1H3a1 1 0 01-1-1V5z"
      fill={isOpen ? '#f59e0b' : 'none'}
      stroke={isOpen ? '#d97706' : 'currentColor'}
      stroke-width="1.3"
    />
  </svg>

  <span class="node-name">{node.segment}</span>

  {#if node.dir}
    <span class="file-count">{node.dir.file_count} file{node.dir.file_count !== 1 ? 's' : ''}</span>
    {#if node.dir.size}
      <span class="dir-size">{formatBytes(node.dir.size)}</span>
    {/if}
  {/if}

  <!-- Actions (hover) -->
  {#if node.dir}
    <div class="node-actions" on:click|stopPropagation>
      <button
        class="na-btn"
        title="Send directory"
        on:click={() => onDeliver(node.dir)}
      >
        <svg viewBox="0 0 16 16" fill="none" width="11" height="11">
          <path d="M2 8h10M8 4l4 4-4 4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </button>
      <button
        class="na-btn danger"
        title="Delete directory"
        on:click={(e) => handleDirDelete(node.dir.prefix, e)}
      >
        <svg viewBox="0 0 16 16" fill="none" width="11" height="11">
          <path d="M3 4h10M6 4V3h4v1M5 4v8a1 1 0 001 1h4a1 1 0 001-1V4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
  {/if}
</button>

<!-- Expanded children -->
{#if isOpen}
  <!-- Child sub-directories -->
  {#if hasChildren}
    {#each node.children as child}
      <SubDirNode
        node={child}
        {expanded}
        {dirFiles}
        {onToggle}
        onDelete={handleSubDelete}
        {onDeliver}
        {onFileSelect}
        {onFileDownload}
        {onFileDelete}
        depth={depth + 1}
      />
    {/each}
  {/if}

  <!-- File rows for this dir -->
  {#if node.dir}
    {#if files === null}
      <div class="file-row loading" style="padding-left: {14 + (depth + 1) * INDENT}px">
        Loading files…
      </div>
    {:else}
      {#each files as file}
        <DirFile
          {file}
          paddingLeft={14 + (depth + 1) * INDENT}
          {onFileSelect}
          onDownload={onFileDownload}
          on:deleted={() => handleFileDelete(file)}
        />
      {/each}
    {/if}
  {/if}
{/if}


</div>



<style>
.node {
  border-bottom: 1px solid var(--border);
}
.node:last-child { border-bottom: none; }

/* ---------------------------------------------------------------
   Node row (directory header)
   min-width: 100% ensures the row stretches to fill the full
   scrollable width of .tree-root (set by min-width: max-content
   in DirectoriesTab), so hover backgrounds reach edge-to-edge
   even when the container is wider than the row's natural width.
--------------------------------------------------------------- */
.node-row {
  display: flex; align-items: center; gap: 8px;
  width: 100%;
  min-width: 100%;        /* fills horizontal scroll width */
  background: none; border: none;
  padding-top: 9px; padding-bottom: 9px; padding-right: 14px;
  cursor: pointer; text-align: left;
  transition: background 0.1s;
  box-sizing: border-box;
}
.node-row:hover { background: var(--bg-2); }
.node-row.open { background: var(--bg); }

.chevron {
  display: flex; align-items: center; justify-content: center;
  width: 14px; height: 14px; color: var(--text-3); flex-shrink: 0;
  transition: transform 0.15s;
}
.chevron.rotated { transform: rotate(90deg); }
.leaf-dot {
  width: 4px; height: 4px; border-radius: 50%;
  background: var(--border-2); display: block;
}

.folder-icon { flex-shrink: 0; color: var(--text-3); }

.node-name {
  font-family: var(--mono); font-size: 12.5px; font-weight: 500;
  color: var(--text); flex: 1;
  /* Allow the name to show fully when h-scrolling; 
     truncate only when the viewport is genuinely narrow */
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;           /* required for ellipsis to kick in inside flex */
}

.file-count {
  font-size: 10.5px; color: var(--text-3);
  background: var(--bg-3); border: 1px solid var(--border);
  padding: 1px 6px; border-radius: 10px; flex-shrink: 0;
}
.dir-size {
  font-family: var(--mono); font-size: 11px; color: var(--text-3); flex-shrink: 0;
}

/* Node hover actions */
.node-actions {
  display: flex; gap: 2px; opacity: 0; transition: opacity 0.1s; flex-shrink: 0;
}
.node-row:hover .node-actions { opacity: 1; }
.na-btn {
  display: flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; background: none;
  border: none; border-radius: 4px;
  color: var(--text-3); cursor: pointer; transition: background 0.1s, color 0.1s;
}
.na-btn:hover { background: var(--bg-3); color: var(--text); }
.na-btn.danger:hover { background: #fef2f2; color: #dc2626; }

/* ---------------------------------------------------------------
   File rows — same min-width treatment as node-row
--------------------------------------------------------------- */
.file-row {
  display: flex; align-items: center; gap: 8px;
  width: 100%;
  min-width: 100%;        /* fills horizontal scroll width */
  background: none; border: none;
  padding-top: 8px; padding-bottom: 8px; padding-right: 14px;
  cursor: pointer; text-align: left;
  border-top: 1px solid var(--border);
  transition: background 0.1s;
  font-size: 12px;
  color: var(--text-2);
  box-sizing: border-box;
}
.file-row:hover { background: #f8f7f4; }
.file-row.loading {
  color: var(--text-3); font-style: italic; cursor: default;
  border-top: 1px dashed var(--border);
}
.file-row.loading:hover { background: none; }

.file-icon { flex-shrink: 0; color: var(--text-3); }

.file-name {
  font-family: var(--mono); font-size: 12px; color: var(--text); flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
.file-lang {
  font-family: var(--mono); font-size: 10px; font-weight: 500;
  padding: 1px 6px; border-radius: 10px; flex-shrink: 0;
  background: color-mix(in srgb, var(--c) 12%, transparent);
  color: color-mix(in srgb, var(--c) 80%, #000);
  border: 1px solid color-mix(in srgb, var(--c) 18%, transparent);
}
.file-tags { display: flex; gap: 3px; flex-shrink: 0; }
.ftag {
  font-size: 10px; font-family: var(--mono); color: var(--text-3);
  background: var(--bg-3); border: 1px solid var(--border);
  padding: 1px 5px; border-radius: 3px;
}
.file-size {
  font-family: var(--mono); font-size: 11px; color: var(--text-3); flex-shrink: 0;
  min-width: 50px; text-align: right;
}
</style>