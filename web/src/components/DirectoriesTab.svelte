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
  import type { Directory, File, TreeNode } from "../types";
  import { toast } from "svelte-sonner";

  export let onFileSelect: (f: File) => void = () => {};
  export let onFileDownload: (f: File, e: MouseEvent) => void = () => {};
  export let onFileDelete: (f: File) => void = () => {};

  let dirs: Directory[] = [];
  let loading = true;
  let error = "";
  let expanded = new Set<string>();
  let dirFiles: Record<string, File[] | undefined> = {};

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

  async function handleToggle(dir: Directory) {
    const next = new Set(expanded);
    if (next.has(dir.prefix)) {
      next.delete(dir.prefix);
    } else {
      next.add(dir.prefix);
      if (dirFiles[dir.prefix] === undefined) {
        try {
          const d = await getDirectory(dir.prefix);
          dirFiles[dir.prefix] = d.files ?? [];
          dirFiles = { ...dirFiles };
        } catch {
          dirFiles[dir.prefix] = [];
          dirFiles = { ...dirFiles };
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
            dirs = dirs.filter((d) => d.prefix !== prefix);
            toast.success(`Deleted directory "${prefix}"`);
          } catch (e: unknown) {
            toast.error((e as Error).message);
            console.error(e);
          }
        },
      },
    });
  }

  function handleDeliver(prefix: string) {
    alert(`Deliver ${prefix} — wire up DeliverModal`);
  }

  function buildTree(flatDirs: Directory[]): TreeNode[] {
    // nodeMap: prefix -> TreeNode, so we can look up a parent node by prefix.
    const nodeMap = new Map<string, TreeNode>();
    const root: TreeNode[] = [];

    // Stack holds directories still to be processed.
    // Seed it with every top-level directory returned by the API.
    const stack: Directory[] = [...flatDirs];

    while (stack.length > 0) {
      const dir = stack.pop()!;

      // Split the prefix into path segments, e.g.
      //   "devbox-web/components/" -> ["devbox-web", "components"]
      const parts = dir.prefix.replace(/\/$/, "").split("/").filter(Boolean);
      const segment = parts[parts.length - 1]; // last segment = this dir's name
      const parentPrefix =
        parts.slice(0, -1).join("/") + (parts.length > 1 ? "/" : "");

      // Create (or promote) the node for this directory.
      let node = nodeMap.get(dir.prefix);
      if (!node) {
        // Compute size from files if the API didn't provide it.
        const size =
          dir.size ||
          (dir.files ? dir.files.reduce((acc, f) => acc + f.size, 0) : 0);
        const idx = dirs.findIndex((d) => d.prefix === dir.prefix);
        dirs[idx] = { ...dirs[idx], size };
        node = {
          segment,
          prefix: dir.prefix,
          dir: { ...dir, size },
          children: [],
        };
        nodeMap.set(dir.prefix, node);
      } else {
        // Was created as a virtual placeholder by a deeper child — attach real dir now.
        node.dir = {
          ...dir,
          size:
            dir.size ||
            (dir.files ? dir.files.reduce((acc, f) => acc + f.size, 0) : 0),
        };
        node.segment = segment;
      }

      // Attach to parent node, or to root if top-level.
      if (parts.length === 1) {
        if (!root.find((n) => n.prefix === dir.prefix)) root.push(node);
      } else {
        let parentNode = nodeMap.get(parentPrefix);
        if (!parentNode) {
          // Parent not seen yet — create a virtual placeholder and push to stack
          // so it gets processed and promoted once we encounter its real Directory.
          const parentSeg = parts[parts.length - 2];
          parentNode = {
            segment: parentSeg,
            prefix: parentPrefix,
            dir: null as unknown as Directory,
            children: [],
          };
          nodeMap.set(parentPrefix, parentNode);

          // Find the grandparent to attach the virtual placeholder correctly.
          const grandParts = parts.slice(0, -2);
          if (grandParts.length === 0) {
            if (!root.find((n) => n.prefix === parentPrefix))
              root.push(parentNode);
          } else {
            const grandPrefix = grandParts.join("/") + "/";
            let grandNode = nodeMap.get(grandPrefix);
            if (!grandNode) {
              grandNode = {
                segment: grandParts[grandParts.length - 1],
                prefix: grandPrefix,
                dir: null as unknown as Directory,
                children: [],
              };
              nodeMap.set(grandPrefix, grandNode);
              root.push(grandNode); // will be re-parented if needed
            }
            if (!grandNode.children.find((c) => c.prefix === parentPrefix)) {
              grandNode.children.push(parentNode);
            }
          }
        }
        if (!parentNode.children.find((c) => c.prefix === dir.prefix)) {
          parentNode.children.push(node);
        }
      }

      // Discover sub-directories from this dir's files (if already loaded).
      // A file at "devbox-web/components/Foo.svelte" implies the subdir
      // "devbox-web/components/" — push it onto the stack so it gets a node.
      if (dir.files) {
        const seenSubs = new Set<string>();
        for (const file of dir.files) {
          // Strip the current dir prefix, then check if there are more segments.
          const relative = file.path.slice(dir.prefix.length);
          const subParts = relative.split("/").filter(Boolean);
          if (subParts.length > 1) {
            // There is at least one sub-directory level.
            const subPrefix = dir.prefix + subParts[0] + "/";
            if (!seenSubs.has(subPrefix) && !nodeMap.has(subPrefix)) {
              seenSubs.add(subPrefix);
              // Build a synthetic Directory for this sub-dir and push to stack.
              const subFiles = dir.files!.filter((f) =>
                f.path.startsWith(subPrefix),
              );
              const syntheticDir: Directory = {
                id: subPrefix,
                name: subParts[0],
                prefix: subPrefix,
                file_count: subFiles.length,
                size: subFiles.reduce((acc, f) => acc + f.size, 0),
                files: subFiles,
              };
              stack.push(syntheticDir);
            }
          }
        }
      }
    }

    return root;
  }

  $: tree = buildTree(dirs.sort((a, b) => b.prefix.localeCompare(a.prefix)));
</script>

<div class="dirs-tab">
  {#if loading}
    <div class="empty-state">Loading directories…</div>
  {:else if error}
    <div class="empty-state err">{error}</div>
  {:else if dirs.length === 0}
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
        >{dirs.length} director{dirs.length !== 1 ? "ies" : "y"}</span
      >
      <span class="summary-total">
        {formatBytes(dirs.reduce((acc, d) => acc + (d.size ?? 0), 0))} total
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
        {#each tree as node}
          {#if node.children.length > 0 || node.dir}
            <!-- Root-level nodes: use DirNode for the top-level dir row,
                 which internally renders SubDirNode for all nested children. -->
            <DirNode
              {node}
              {expanded}
              {dirFiles}
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
