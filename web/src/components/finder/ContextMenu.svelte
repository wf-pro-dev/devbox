<script lang="ts">
  import { onDestroy } from "svelte";
  import type { DirEntry } from "../../types";

  export let x = 0;
  export let y = 0;
  export let entry: DirEntry | null = null;
  export let onClose: () => void = () => {};
  export let onView: () => void = () => {};
  export let onSend: () => void = () => {};
  export let onDiff: () => void = () => {};
  export let onStatus: () => void = () => {};
  export let onDownload: () => void = () => {};
  export let onCopyPath: () => void = () => {};
  export let onRename: () => void = () => {};
  export let onTags: () => void = () => {};
  export let onDelete: () => void = () => {};
  export let onUploadHere: () => void = () => {};

  function clickAway() {
    onClose();
  }

  function handleKey(e: KeyboardEvent) {
    if (e.key === "Escape") onClose();
  }

  window.addEventListener("click", clickAway);
  window.addEventListener("keydown", handleKey);

  onDestroy(() => {
    window.removeEventListener("click", clickAway);
    window.removeEventListener("keydown", handleKey);
  });

  function run(action: () => void) {
    action();
    onClose();
  }

  $: isRemoteFile = entry?.file?.source === "remote";
</script>

<div class="ctx" style="left:{x}px; top:{y}px" on:click|stopPropagation>
  {#if entry === null}
    <button class="mi" on:click={() => run(onUploadHere)}>Upload files here…</button>
    <button class="mi" disabled>New directory</button>
    <button class="mi" disabled>Sort by</button>
  {:else if entry.is_dir}
    <button class="mi" disabled={isRemoteFile} on:click={() => run(onSend)}>Send directory…</button>
    <button class="mi" disabled={isRemoteFile} on:click={() => run(onDownload)}>Download .tar.gz</button>
    <button class="mi" disabled on:click={() => run(onTags)}>Tags…</button>
    <button class="mi" on:click={() => run(onCopyPath)}>Copy path</button>
    <div class="sep"></div>
    <button class="mi danger" disabled={isRemoteFile} on:click={() => run(onDelete)}>Delete all files</button>
  {:else}
    <button class="mi" on:click={() => run(onView)}>Quick look</button>
    <button class="mi" disabled={isRemoteFile} on:click={() => run(onSend)}>Send to node…</button>
    <button class="mi" disabled={isRemoteFile} on:click={() => run(onDiff)}>Diff…</button>
    <button class="mi" disabled={isRemoteFile} on:click={() => run(onStatus)}>Check fleet status</button>
    <div class="sep"></div>
    <button class="mi" on:click={() => run(onDownload)}>Download</button>
    <button class="mi" disabled={isRemoteFile} on:click={() => run(onRename)}>Rename…</button>
    <button class="mi" on:click={() => run(onCopyPath)}>Copy path</button>
    <button class="mi" disabled={isRemoteFile} on:click={() => run(onTags)}>Tags…</button>
    <div class="sep"></div>
    <button class="mi" on:click={() => run(onView)}>Get Info</button>
    <div class="sep"></div>
    <button class="mi danger" disabled={isRemoteFile} on:click={() => run(onDelete)}>Delete</button>
  {/if}
</div>

<style>
  .ctx {
    position: fixed;
    z-index: 300;
    width: 160px;
    padding: 5px;
    border: 0.5px solid var(--f-border2);
    border-radius: 8px;
    background: var(--f-surface);
    box-shadow: 0 14px 35px rgba(0, 0, 0, 0.14);
  }
  .mi {
    width: 100%;
    border: none;
    background: transparent;
    text-align: left;
    padding: 7px 9px;
    border-radius: 5px;
    font-size: 11px;
    color: var(--f-text);
  }
  .mi:hover:not(:disabled) {
    background: var(--f-selection);
  }
  .mi:disabled {
    opacity: 0.45;
    cursor: default;
  }
  .mi.danger {
    color: var(--f-danger);
  }
  .sep {
    height: 1px;
    background: var(--f-border);
    margin: 4px 2px;
  }
</style>
