<script lang="ts">
  import { onMount } from "svelte";
  import {
    listDirectories,
    getDirectory,
    deleteDirectory,
    formatBytes,
  } from "../api";
  import DirNode from "./DirNode.svelte";
  import SubDirNode from "./SubDirNode.svelte";
  import type { Directory, File, TreeNode, DirListing, DirEntry } from "../types";
  import { toast } from "svelte-sonner";
  import SendModal from "./SendModal.svelte";

  export let onFileSelect: (f: File) => void = () => {};
  export let onFileDownload: (f: File, e: MouseEvent) => void = () => {};
  export let onFileDelete: (f: File) => void = () => {};

  let dirs: DirListing = { prefix: "", entries: [] };
  let loading = true;
  let error = "";
  let expanded = new Set<string>();
  let dirEntries: Record<string, DirEntry[] | undefined> = {};
  let showDeliver = false;
  let dirToSend: DirEntry | null = null;

  onMount(load);

  async function load() {
    loading = true;
    error = "";
    try {
      dirs = await listDirectories();
    } catch (e: unknown) {
      error = (e as Error).message;
    } finally {
      loading = false;
    }
  }

  async function handleToggle(dir: DirEntry) {
    let prefix = dir.prefix ?? "";
    const next = new Set(expanded);
    if (next.has(prefix)) {
      next.delete(prefix);
    } else {
      next.add(prefix);
      if (dirEntries[prefix] === undefined) {
        try {
          const d = await getDirectory(prefix);
          dirEntries[prefix] = d.entries;
        } catch {
          dirEntries[prefix] = undefined;
        }
      }
    }
    expanded = next;
  }



  async function handleDelete(prefix: string) {
    toast(`Delete directory "${prefix}"?`, {
      action: {
        label: "Confirm",
        onClick: async () => {
          try {
            await deleteDirectory(prefix);
            dirs.entries = dirs.entries.filter((d) => d.prefix !== prefix);
            toast.success(`Deleted directory "${prefix}"`);
          } catch (e: unknown) {
            toast.error((e as Error).message);
            console.error(e);
          }
        },
      },
    });
  }

  function handleDeliver(dir: DirEntry) {
    dirToSend = dir;
    showDeliver = true;
  }

</script>

<div class="dirs-tab">
  {#if loading}
    <div class="empty-state">Loading directories…</div>
  {:else if error}
    <div class="empty-state err">{error}</div>
  {:else if dirs.entries.length === 0}
    <div class="empty-state">
      <div class="empty-icon">
        <svg viewBox="0 0 24 24" fill="none" width="36" height="36">
          <path
            d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z"
            stroke="currentColor"
            stroke-width="1.4"
          />
        </svg>
      </div>
      <p class="empty-title">No directories yet</p>
      <p class="empty-sub">
        Push a directory with <code>devbox push -r ./mydir/</code>
      </p>
    </div>
  {:else}
    <div class="summary-bar">
      <span class="summary-count"
        >{dirs.entries.length} director{dirs.entries.length !== 1 ? "ies" : "y"}</span
      >
      <span class="summary-total">
        {formatBytes(dirs.entries.reduce((acc, d) => acc + (d.file?.size ?? 0), 0))} total
      </span>
    </div>

    <!--
      Scroll container: scrolls in BOTH axes.
      - overflow-y: auto  → vertical scroll when tree is taller than the panel
      - overflow-x: auto  → horizontal scroll when deeply-nested rows are wider than the panel
      The inner .tree-root uses min-width: max-content so it never wraps or clips.
    -->
    <div class="tree-scroll">
      <div class="tree-root">
        {#each dirs.entries as node}
          {#if node.is_dir}
            <!-- Root-level nodes: use DirNode for the top-level dir row,
                 which internally renders SubDirNode for all nested children. -->
            <DirNode
              {node}
              {expanded}
              {dirEntries}
              onToggle={handleToggle}
              onDelete={handleDelete}
              onDeliver={handleDeliver}
              {onFileSelect}
              {onFileDownload}
              {onFileDelete}
              depth={0}
            />
          {/if}
        {/each}
      </div>
      
    </div>
  {/if}
</div>

{#if showDeliver && dirToSend}
  <SendModal dir={dirToSend} on:close={() => (showDeliver = false)} />
{/if}

<style>
  .dirs-tab {
    /* Fill whatever height the parent gives; column layout lets summary-bar
       stay fixed at the top while tree-scroll takes the remaining space. */
    flex: 1;
    min-height: 0; /* critical: lets flex children shrink below content size */
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px 20px;
    overflow: hidden; /* clip at this boundary; scrolling lives in .tree-scroll */
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    text-align: center;
    gap: 10px;
    color: var(--text-3);
  }
  .empty-state.err {
    color: #dc2626;
  }
  .empty-icon {
    color: var(--border-2);
    margin-bottom: 4px;
  }
  .empty-title {
    font-family: var(--serif);
    font-size: 18px;
    color: var(--text-2);
  }
  .empty-sub {
    font-size: 12px;
  }
  .empty-sub code {
    font-family: var(--mono);
    background: var(--bg-2);
    padding: 1px 6px;
    border-radius: 3px;
  }

  .summary-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 2px;
    flex-shrink: 0; /* never let the summary bar get squished */
  }
  .summary-count {
    font-size: 12px;
    color: var(--text-3);
    font-family: var(--mono);
  }
  .summary-total {
    font-size: 11px;
    color: var(--text-3);
    font-family: var(--mono);
  }

  /* The scrollable viewport — grows to fill remaining space in .dirs-tab */
  .tree-scroll {
    flex: 1;
    min-height: 0; /* must be set on flex children that should scroll */
    overflow-x: auto; /* horizontal scroll for deep nesting */
    overflow-y: auto; /* vertical scroll for long lists */
    border-radius: var(--radius-lg);
    /* Thin, unobtrusive scrollbars (WebKit) */
    scrollbar-width: thin;
    scrollbar-color: var(--border-2) transparent;
  }
  .tree-scroll::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }
  .tree-scroll::-webkit-scrollbar-track {
    background: transparent;
  }
  .tree-scroll::-webkit-scrollbar-thumb {
    background: var(--border-2);
    border-radius: 3px;
  }
  .tree-scroll::-webkit-scrollbar-corner {
    background: transparent;
  }

  /* The actual tree — sized to its content so the scroll container can measure it */
  .tree-root {
    min-width: max-content; /* never shrink below intrinsic width → enables h-scroll */
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    overflow: hidden; /* keep rounded corners on the border */
    background: white;
  }
</style>
