<script lang="ts">
  import type { Directory, File, TreeNode } from "../types";
  import { formatBytes, deleteDirectory } from "../api";
  import DirFile from "./DirFile.svelte";
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

  // A SubDirNode is an intermediate folder — it may or may not have a real
  // Directory backing it. Use the prefix to track open/close state.
  $: isOpen = expanded.has(node.prefix) || virtualOpen;
  $: hasChildren = node.children.length > 0;
  $: isLeafDir = node.dir != null && !hasChildren;
  function directFiles(
    allFiles: File[] | null | undefined,
    prefix: string,
  ): File[] | null {
    if (allFiles == null) return null;
    return allFiles.filter((f) => {
      const relative = f.path.startsWith(prefix)
        ? f.path.slice(prefix.length)
        : f.file_name;
      return !relative.includes("/");
    });
  }
  $: files = node.dir
    ? directFiles(dirFiles[node.dir.prefix], node.dir.prefix)
    : null;

  let virtualOpen = false;

  const INDENT = 20;

  function handleToggle() {
    if (node.dir) {
      onToggle(node.dir);
    } else {
      virtualOpen = !virtualOpen;
    }
  }
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

<div class="subnode">
  <!-- Row -->
  <button
    class="subnode-row"
    class:open={isOpen}
    style="padding-left: {14 + depth * INDENT}px"
    on:click={handleToggle}
  >
    <!-- Chevron -->
    <span class="chevron" class:rotated={isOpen}>
      <svg viewBox="0 0 10 10" fill="none" width="9" height="9">
        <path
          d="M3 2l4 3-4 3"
          stroke="currentColor"
          stroke-width="1.4"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
      </svg>
    </span>

    <!-- Subdirectory folder icon (slightly muted vs root) -->
    <svg
      class="folder-icon"
      viewBox="0 0 16 16"
      fill="none"
      width="13"
      height="13"
    >
      <path
        d="M2 5a1 1 0 011-1h3l1.5 1.5H13a1 1 0 011 1V12a1 1 0 01-1 1H3a1 1 0 01-1-1V5z"
        fill={isOpen ? "#fde68a" : "none"}
        stroke={isOpen ? "#d97706" : "currentColor"}
        stroke-width="1.3"
      />
    </svg>

    <span class="subnode-name">{node.segment}</span>

    {#if node.dir}
      <span class="file-count"
        >{node.dir.file_count} file{node.dir.file_count !== 1 ? "s" : ""}</span
      >
      {#if node.dir.size}
       
        <span class="dir-size">{formatBytes(node.dir.size)}</span>
      {/if}
    {:else}
      <!-- Virtual intermediate node — no real directory backing -->
      <span class="virtual-badge">folder</span>
    {/if}

    {#if node.dir}
      <div class="node-actions" on:click|stopPropagation>
        <button
          class="na-btn"
          title="Send directory"
          on:click={() => onDeliver(node.dir)}
        >
          <svg viewBox="0 0 16 16" fill="none" width="11" height="11">
            <path
              d="M2 8h10M8 4l4 4-4 4"
              stroke="currentColor"
              stroke-width="1.4"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
        <button
          class="na-btn danger"
          title="Delete directory"
          on:click={(e) => handleDirDelete(node.dir.prefix, e)}
        >
          <svg viewBox="0 0 16 16" fill="none" width="11" height="11">
            <path
              d="M3 4h10M6 4V3h4v1M5 4v8a1 1 0 001 1h4a1 1 0 001-1V4"
              stroke="currentColor"
              stroke-width="1.3"
              stroke-linecap="round"
            />
          </svg>
        </button>
      </div>
    {/if}
  </button>

  <!-- Expanded content -->
  {#if isOpen}
    <!-- Recurse into child nodes -->
    {#each node.children as child}
      <svelte:self
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

    <!-- Files for this dir (only when it's a real directory node) -->
    {#if node.dir}
      {#if files === null}
        <div
          class="file-row loading"
          style="padding-left: {14 + (depth + 1) * INDENT}px"
        >
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
  .subnode {
    border-bottom: 1px solid var(--border);
  }
  .subnode:last-child {
    border-bottom: none;
  }

  .subnode-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-width: 100%;
    background: none;
    border: none;
    padding-top: 8px;
    padding-bottom: 8px;
    padding-right: 14px;
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
    box-sizing: border-box;
    /* Slightly tinted background to distinguish from root dirs */
    background: var(--bg);
  }
  .subnode-row:hover {
    background: var(--bg-2);
  }
  .subnode-row.open {
    background: color-mix(in srgb, var(--bg-2) 60%, transparent);
  }

  .chevron {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    color: var(--text-3);
    flex-shrink: 0;
    transition: transform 0.15s;
  }
  .chevron.rotated {
    transform: rotate(90deg);
  }

  .folder-icon {
    flex-shrink: 0;
    color: var(--text-3);
  }

  .subnode-name {
    font-family: var(--mono);
    font-size: 12px;
    font-weight: 400;
    color: var(--text-2);
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }

  .file-count {
    font-size: 10px;
    color: var(--text-3);
    background: var(--bg-3);
    border: 1px solid var(--border);
    padding: 1px 5px;
    border-radius: 10px;
    flex-shrink: 0;
  }
  .dir-size {
    font-family: var(--mono);
    font-size: 11px;
    color: var(--text-3);
    flex-shrink: 0;
  }
  .virtual-badge {
    font-size: 9.5px;
    font-family: var(--mono);
    color: var(--text-3);
    background: var(--bg-3);
    border: 1px dashed var(--border-2);
    padding: 1px 5px;
    border-radius: 10px;
    flex-shrink: 0;
    opacity: 0.7;
  }

  /* Hover actions */
  .node-actions {
    display: flex;
    gap: 2px;
    opacity: 0;
    transition: opacity 0.1s;
    flex-shrink: 0;
  }
  .subnode-row:hover .node-actions {
    opacity: 1;
  }
  .na-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: none;
    border: none;
    border-radius: 4px;
    color: var(--text-3);
    cursor: pointer;
    transition:
      background 0.1s,
      color 0.1s;
  }
  .na-btn:hover {
    background: var(--bg-3);
    color: var(--text);
  }
  .na-btn.danger:hover {
    background: #fef2f2;
    color: #dc2626;
  }

  /* File rows */
  .file-row {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    min-width: 100%;
    background: none;
    border: none;
    padding-top: 8px;
    padding-bottom: 8px;
    padding-right: 14px;
    cursor: pointer;
    text-align: left;
    border-top: 1px solid var(--border);
    transition: background 0.1s;
    font-size: 12px;
    color: var(--text-2);
    box-sizing: border-box;
  }
  .file-row:hover {
    background: #f8f7f4;
  }
  .file-row.loading {
    color: var(--text-3);
    font-style: italic;
    cursor: default;
    border-top: 1px dashed var(--border);
  }
  .file-row.loading:hover {
    background: none;
  }

</style>
