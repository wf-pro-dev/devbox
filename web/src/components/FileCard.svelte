<script>
  import { formatBytes, formatDate, langColor } from '../api.js';

  export let file;
  export let selected = false;

  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();
</script>

<button class="card" class:selected on:click={() => dispatch('click')}>
  <div class="top">
    <span class="filename">{file.file_name || file.path}</span>
    {#if file.dir_id}
      <span class="dir-badge" title={file.path}>📁 {file.path.split('/')[0]}</span>
    {/if}
    <span class="lang" style="--c:{langColor(file.language)}">{file.language}</span>
  </div>
  {#if file.description}
    <p class="desc">{file.description}</p>
  {/if}
  <div class="tags">
    {#each file.tags as tag}
      <button class="tag" on:click|stopPropagation={() => dispatch('tagClick', tag)}>#{tag}</button>
    {/each}
  </div>
  <div class="meta-row">
    <span>{formatBytes(file.size)}</span>
    <span class="dot">·</span>
    <span>{file.uploaded_by}</span>
    <span class="dot">·</span>
    <span>{formatDate(file.created_at)}</span>
  </div>
</button>

<style>
  .card {
    display: flex; flex-direction: column; gap: 8px;
    padding: 14px 16px; background: white;
    border: 1px solid var(--border); border-radius: var(--radius-lg);
    text-align: left; width: 100%; cursor: pointer;
    transition: border-color 0.1s, box-shadow 0.1s;
  }
  .card:hover { border-color: var(--border-2); box-shadow: 0 1px 4px rgba(0,0,0,0.06); }
  .card.selected { border-color: var(--text); box-shadow: 0 0 0 1px var(--text); }

  .top { display: flex; align-items: center; justify-content: space-between; gap: 8px; }

  .filename {
    font-family: var(--mono); font-size: 13px; font-weight: 500;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }

  .dir-badge {
    font-size: 10px; color: var(--text-3); background: var(--bg-3);
    padding: 1px 6px; border-radius: 3px; flex-shrink: 0;
    white-space: nowrap; font-family: var(--mono);
  }

  .lang {
    flex-shrink: 0; font-family: var(--mono); font-size: 10px; font-weight: 500;
    padding: 2px 7px; border-radius: 20px;
    background: color-mix(in srgb, var(--c) 12%, transparent);
    color: color-mix(in srgb, var(--c) 80%, #000);
    border: 1px solid color-mix(in srgb, var(--c) 20%, transparent);
  }

  .desc {
    font-size: 12px; color: var(--text-2); line-height: 1.4;
    overflow: hidden; display: -webkit-box;
    -webkit-line-clamp: 2; -webkit-box-orient: vertical;
  }

  .tags { display: flex; flex-wrap: wrap; gap: 4px; }

  .tag {
    background: none; border: 1px solid var(--border); border-radius: 4px;
    padding: 1px 6px; font-size: 11px; font-family: var(--mono); color: var(--text-3);
    cursor: pointer; transition: all 0.1s;
  }
  .tag:hover { border-color: #3b82f6; color: #3b82f6; }

  .meta-row { display: flex; gap: 5px; }
  .meta-row span { font-size: 11px; color: var(--text-3); font-family: var(--mono); }
  .dot { color: var(--border-2); }
</style>