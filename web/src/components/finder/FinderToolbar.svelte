<script lang="ts">
  import type { DirEntry } from "../../types";

  export let prefix = "/";
  export let viewMode: "column" | "list" | "grid" = "column";
  export let activeTag = "";
  export let selectedEntry: DirEntry | null = null;
  export let iconSize = 84;
  export let onNavigate: (index: number) => void = () => {};
  export let onViewChange: (
    mode: "column" | "list" | "grid",
  ) => void = () => {};
  export let onTagToggle: (tag: string) => void = () => {};
  export let onUpload: () => void = () => {};
  export let onSend: () => void = () => {};
  export let onDelete: () => void = () => {};
  export let onStatus: () => void = () => {};
  export let onDiff: () => void = () => {};
  export let onNavigateBack: () => void = () => {};
  export let onNavigateForward: () => void = () => {};
  export let canNavigateBack = false;
  export let canNavigateForward = false;
  export let onIconSize: (size: number) => void = () => {};

  $: cleanPrefix =
    prefix === "/"
      ? []
      : prefix
          .replace(/^\/|\/$/g, "")
          .split("/")
          .filter(Boolean);
</script>

<div class="toolbar">
  <div class="left">
    <button
      class="tb-btn chevron"
      on:click={onNavigateBack}
      title="Navigate back"
      disabled={!canNavigateBack}><i class="ti ti-chevron-left"></i></button
    >
    <button
      class="tb-btn chevron"
      on:click={onNavigateForward}
      title="Navigate forward"
      disabled={!canNavigateForward}
    >
      <i class="ti ti-chevron-right"></i></button
    >
    <div class="sep"></div>
    <button class="crumb active" on:click={() => onNavigate(0)}>/</button>
    {#each cleanPrefix as segment, i}
      <i class="ti ti-chevron-right bc"></i>
      <button class="crumb" on:click={() => onNavigate(i + 1)}>{segment}</button
      >
    {/each}
    <div class="sep"></div>
  </div>
  <div class="right">
    {#if viewMode === "grid"}

      <div class="slider-wrap">
        <i class="ti ti-photo-minus"></i>
        <input
          type="range"
          min="60"
          max="120"
          step="4"
          value={iconSize}
          on:input={(e) =>
            onIconSize(Number((e.currentTarget as HTMLInputElement).value))}
        />
        <i class="ti ti-photo-plus"></i>
      </div>

      <div class="sep"></div>
    {/if}



    <button
      title="Column view"
      class="tb-btn"
      class:active={viewMode === "column"}
      on:click={() => onViewChange("column")}
      ><i class="ti ti-layout-columns"></i></button
    >
    <button
      title="List view"
      class="tb-btn"
      class:active={viewMode === "list"}
      on:click={() => onViewChange("list")}><i class="ti ti-list"></i></button
    >
    <button
      title="Grid view"
      class="tb-btn"
      class:active={viewMode === "grid"}
      on:click={() => onViewChange("grid")}
      ><i class="ti ti-grid-dots"></i></button
    >
    <div class="sep"></div>

    <button
      title="Upload"
      class="tb-btn"
      on:click={() => onUpload()}
      ><i class="ti ti-upload"></i></button
    >

    {#if activeTag}
      <div class="sep"></div>
      <button class="tag-pill" on:click={() => onTagToggle(activeTag)}>
        #{activeTag} ×
      </button>
    {/if}

    <div class="sep"></div>

    <div class="finder-search">
      <i class="ti ti-search"></i>
      <input placeholder="Search files, paths, tags…" />
    </div>
  </div>
</div>

<style>
  .toolbar {
    height: 44px;
    border-bottom: 0.5px solid var(--f-border);
    background: var(--f-bg1);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 8px;
    gap: 10px;
  }
  .left,
  .right {
    display: flex;
    align-items: center;
    gap: 4px;
    min-width: 0;
  }
  .tb-btn,
  .crumb,
  .tag-pill {
    height: 32px;
    border: 0.5px solid transparent;
    background: transparent;
    color: var(--f-text2);
    border-radius: 6px;
    padding: 0 8px;
    font-size: 20px;
  }
  .crumb {
    font-size: 16px;
  }
  .tb-btn {
    width: 32px;
    padding: 0;
  }
  .tb-btn:hover,
  .crumb:hover,
  .tag-pill:hover,
  .tb-btn.active {
    background: var(--f-surface);
    border-color: var(--f-border);
    color: var(--f-text);
  }
  .tb-btn.danger {
    color: var(--f-danger);
  }
  .chevron {
    font-size: 16px;
  }
  .chevron:disabled {
    opacity: 0.8;
    cursor: not-allowed;
  }
  .chevron:disabled i {
    color: var(--f-text3);
  }
  .crumb {
    white-space: nowrap;
  }
  .crumb.active {
    font-weight: 500;
  }
  .tag-pill {
    font-family: var(--mono);
  }
  .sep {
    width: 1px;
    height: 16px;
    background: var(--f-border);
    margin: 0 2px;
  }
  .bc {
    font-size: 9px;
    color: var(--f-text3);
  }
  .slider-wrap {
    display: flex;
    align-items: center;
    gap: 5px;
    color: var(--f-text3);
  }
  .slider-wrap input {
    height: 4px;
    width: 78px;
  }
  .finder-search {
    flex: 1;
    max-width: 360px;
    position: relative;
  }
  .finder-search i {
    position: absolute;
    left: 9px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--f-text3);
  }
  .finder-titlebar {
    height: 38px;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 10px;
    background: var(--f-bg1);
    border-bottom: 0.5px solid var(--f-border);
  }
  .finder-title-actions {
    margin-left: auto;
    display: flex;
    gap: 4px;
  }
  .finder-title-actions button {
    width: 24px;
    height: 24px;
    border: none;
    background: transparent;
    border-radius: 6px;
  }
  .finder-search input {
    width: 360px;
    height: 28px;
    border: 0.5px solid var(--f-search-border);
    border-radius: 12px;
    background: var(--f-search-bg);
    padding: 0 10px 0 28px;
    font-size: 12px;
  }
</style>
