<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { formatDate, api } from '../api.js';


  export let health = null;
  export let recentFiles = [];
  export let activeTag = '';
  export let allTags = [];
  export let peers = [];
  let loading = true;
  let error = '';

  const dispatch = createEventDispatcher();

  onMount(async () => {
    try {
      peers = await api.listPeers();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });

</script>



<aside>
  <section>
    <h3>Machines</h3>
    {#if health?.caller_host}
      <div class="machine">
        <span class="dot"></span>
        <span class="machine-name">{health.caller_host}</span>
        <span class="badge">you</span>
      </div>
    {#each peers as peer}
      <div class="machine">
        <span class="dot"></span>
        <span class="machine-name">{peer.hostname}</span>
        <span class="badge">{peer.online ? 'online' : 'offline'}</span>
      </div>
    {/each}
    {:else}
      <p class="empty">Connect via Tailscale to see machines.</p>
    {/if}
  </section>

  <section>
    <h3>Recent</h3>
    {#if recentFiles.length === 0}
      <p class="empty">No files yet.</p>
    {:else}
      {#each recentFiles as file}
        <button class="recent" on:click={() => dispatch('selectFile', file)}>
          <span class="recent-name">{file.name}</span>
          <span class="recent-date">{formatDate(file.created_at).split(',')[0]}</span>
        </button>
      {/each}
    {/if}
  </section>

  {#if allTags.length > 0}
    <section>
      <h3>Tags</h3>
      <div class="tag-list">
        {#each allTags as tag}
          <button
            class="tag" class:active={activeTag === tag}
            on:click={() => dispatch('selectTag', tag)}
          >#{tag}</button>
        {/each}
      </div>
    </section>
  {/if}
</aside>

<style>
  aside {
    width: 220px; min-width: 220px; border-right: 1px solid var(--border);
    padding: 16px 12px; overflow-y: auto;
    display: flex; flex-direction: column; gap: 24px; background: var(--bg);
  }
  section { display: flex; flex-direction: column; gap: 6px; }
  h3 {
    font-size: 10px; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.08em; color: var(--text-3); padding: 0 4px; margin-bottom: 2px;
  }
  .machine {
    display: flex; align-items: center; gap: 7px;
    padding: 6px 8px; border-radius: var(--radius); background: var(--bg-2);
  }
  .dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: #16a34a; flex-shrink: 0;
  }
  .machine-name {
    font-family: var(--mono); font-size: 12px; flex: 1;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .badge {
    font-size: 10px; color: var(--text-3); background: var(--bg-3);
    border: 1px solid var(--border); padding: 1px 5px; border-radius: 3px;
  }
  .recent {
    display: flex; align-items: center; justify-content: space-between; gap: 8px;
    padding: 5px 8px; border-radius: var(--radius);
    background: none; border: none; width: 100%; text-align: left; cursor: pointer;
    transition: background 0.1s;
  }
  .recent:hover { background: var(--bg-2); }
  .recent-name {
    font-family: var(--mono); font-size: 12px;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .recent-date { font-size: 10px; color: var(--text-3); flex-shrink: 0; }
  .tag-list { display: flex; flex-wrap: wrap; gap: 4px; padding: 0 4px; }
  .tag {
    background: none; border: 1px solid var(--border); border-radius: 4px;
    padding: 2px 8px; font-size: 11px; font-family: var(--mono);
    color: var(--text-3); cursor: pointer; transition: all 0.1s;
  }
  .tag:hover, .tag.active { border-color: #3b82f6; color: #3b82f6; background: #eff6ff; }
  .empty { font-size: 11px; color: var(--text-3); padding: 4px 8px; font-style: italic; }
</style>