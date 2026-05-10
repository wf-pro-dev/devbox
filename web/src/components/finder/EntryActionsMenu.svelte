<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { DirEntry } from "../../types";
  import { onDestroy } from "svelte";

  export let entry: DirEntry;

  const dispatch = createEventDispatcher<{
    send: void;
    download: void;
    rename: void;
    delete: void;
  }>();

  let open = false;

  function close() {
    open = false;
  }

  function toggle(event: MouseEvent) {
    event.stopPropagation();
    open = !open;
  }

  function run(name: "send" | "download" | "rename" | "delete", event: MouseEvent) {
    event.stopPropagation();
    dispatch(name);
    close();
  }

  function handleWindowClick() {
    close();
  }

  window.addEventListener("click", handleWindowClick);
  onDestroy(() => window.removeEventListener("click", handleWindowClick));
</script>

<div class="menu-wrap" on:click|stopPropagation>
  <button class="menu-btn" aria-label="File actions" title="File actions" on:click={toggle}>
    <i class="ti ti-dots"></i>
  </button>
  {#if open}
    <div class="menu">
      <button class="item" on:click={(e) => run("send", e)}>Send</button>
      <button class="item" on:click={(e) => run("download", e)}>Download</button>
      <button class="item" on:click={(e) => run("rename", e)}>Rename…</button>
      <button class="item danger" on:click={(e) => run("delete", e)}>Delete</button>
    </div>
  {/if}
</div>

<style>
  .menu-wrap {
    position: relative;
  }
  .menu-btn {
    width: 22px;
    height: 22px;
    border: none;
    background: transparent;
    color: var(--f-text2);
    border-radius: 4px;
  }
  .menu-btn:hover {
    background: rgba(255, 255, 255, 0.8);
  }
  .menu {
    position: absolute;
    right: 0;
    top: calc(100% + 4px);
    z-index: 30;
    min-width: 126px;
    padding: 4px;
    border: 0.5px solid var(--f-border2);
    border-radius: 8px;
    background: var(--f-surface);
    box-shadow: 0 14px 35px rgba(0, 0, 0, 0.14);
  }
  .item {
    width: 100%;
    border: none;
    background: transparent;
    text-align: left;
    padding: 7px 9px;
    border-radius: 5px;
    font-size: 11px;
    color: var(--f-text);
  }
  .item:hover {
    background: var(--f-selection);
  }
  .item.danger {
    color: var(--f-danger);
  }
</style>
