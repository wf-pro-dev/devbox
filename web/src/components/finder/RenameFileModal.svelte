<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { File } from "../../types";
  import { splitPath } from "./entryPaths";

  export let file: File;
  export let busy = false;

  const dispatch = createEventDispatcher<{
    close: void;
    submit: { path: string };
  }>();

  const parts = splitPath(file.path);
  let name = parts.name;

  function submit() {
    const trimmed = name.trim();
    if (!trimmed) return;
    const path = parts.dir ? `${parts.dir}/${trimmed}` : trimmed;
    dispatch("submit", { path });
  }
</script>

<div class="backdrop" on:click={() => dispatch("close")}>
  <div class="modal" role="dialog" aria-modal="true" tabindex="-1" on:click|stopPropagation>
    <div class="hdr">
      <h2>Rename “{file.file_name}”</h2>
    </div>
    <p class="copy">Choose a new file name. This uses the same move path update as Finder drag and drop.</p>
    <label class="field">
      <span>Name</span>
      <input bind:value={name} autofocus on:keydown={(e) => e.key === "Enter" && submit()} />
    </label>
    {#if parts.dir}
      <div class="path-hint">Location: /{parts.dir}</div>
    {/if}
    <div class="actions">
      <button class="secondary" on:click={() => dispatch("close")} disabled={busy}>Cancel</button>
      <button class="primary" on:click={submit} disabled={busy || !name.trim()}>Rename</button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 320;
    background: rgba(25, 26, 31, 0.22);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .modal {
    width: 420px;
    border: 0.5px solid var(--f-border2);
    border-radius: 12px;
    background: var(--f-surface);
    box-shadow: 0 20px 48px rgba(0, 0, 0, 0.18);
    padding: 18px;
  }
  .hdr h2 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--f-text);
  }
  .copy {
    margin: 10px 0 14px;
    font-size: 11px;
    color: var(--f-text2);
  }
  .field {
    display: block;
  }
  .field span {
    display: block;
    margin-bottom: 6px;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--f-text3);
  }
  .field input {
    width: 100%;
    height: 34px;
    border: 0.5px solid var(--f-border2);
    border-radius: 8px;
    background: #fff;
    padding: 0 10px;
    font-size: 12px;
  }
  .path-hint {
    margin-top: 8px;
    font-family: var(--mono);
    font-size: 10.5px;
    color: var(--f-text3);
  }
  .actions {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
  .actions button {
    height: 30px;
    border-radius: 7px;
    padding: 0 12px;
    font-size: 11px;
    border: 0.5px solid var(--f-border2);
  }
  .secondary {
    background: transparent;
    color: var(--f-text2);
  }
  .primary {
    background: var(--f-accent);
    border-color: var(--f-accent);
    color: white;
  }
</style>
